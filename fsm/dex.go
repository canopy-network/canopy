package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"math/big"
	"sort"
	"strings"
)

/* Dex.go implements logic to handle AMM style atomic exchanges between root & nested chains
   not to be confused with 1 way order book swaps implemented in swap.go */

// Cross-Chain DEX Pipeline (High-Level Overview)
// ---------------------------------------------
// Two chains (X and Y) continuously alternate between:
//   • Collecting local DEX ops into `nextBatch`
//   • Locking and sending that batch to the other chain
//   • Processing the other chain’s locked batch
//   • Applying receipts from previously locked batches
//
// Pipeline cycle:
//   1) Both chains build `nextBatch` from local orders, deposits, withdrawals
//        - Orders/Deposits: move funds to escrow and add the command to nextBatch
//        - Withdrawals: add the command to nextBatch
//
//   2) ChainX locks nextBatch → sends to ChainY
//
//   3) ChainY receives locked batch:
//        a) Apply receipts for prior locked batch (if any)
//        b) Process ChainX’s locked batch:
//             - Execute Orders: update pool, produce receipts
//             - Execute Withdrawals: distribute tokens, burn points
//             - Execute Deposits: create points, update pool
//        c) Add produced receipts → Y.nextBatch, then lock + send to X
//
//   4) ChainX receives locked batch:
//        a) Apply receipts from earlier X→Y batch
//        b) Process Y’s locked batch (same rules as step 3b)
//        c) Add receipts → X.nextBatch, then lock + send to Y
//
//   5) Repeat steps 3–4 indefinitely.
//
// Effectively: each chain is always processing the other chain’s locked batch
// while preparing its own next locked batch, forming a continuous pipeline.
//
// Ledger mirroring & “shadow tracking”
// ------------------------------------
// - Each chain keeps a local mirror of the counter chain’s liquidity pool (`remoteBatch.PoolSize`)
//   so that AMM math on either side uses the same reserves.
// - When we apply receipts for *our* locked batch (Step 1), we “advance” that mirrored
//   counter-pool snapshot by subtracting what the counter chain already paid out, so the mirror
//   reflects their post-receipt state.
// - We also capture a mid-point snapshot of *our* pool after receipts (`midPointPoolSize`).
//   That snapshot is sent to the counter chain so it can mirror our pool for its next cycle.
// - `CounterPoolSize` stored on a rotated batch is informational (RPC/event pricing) and is not
//   used in AMM execution; the executable pool for the next batch is `nextBatch.PoolSize`, set
//   from the mid-point snapshot.

// HandleDexBatch() initiates the 'dex' lifecycle
func (s *StateMachine) HandleDexBatch(chainId uint64, results *lib.CertificateResult, isNested bool) (err lib.ErrorI) {
	remoteBatch := results.DexBatch
	// if nested, replace chainId with root chainId
	if isNested {
		// set 'chain id' as the root chain
		if chainId, err = s.GetRootChainId(); err != nil {
			return
		}
		// use the root chainId as the remote batch
		remoteBatch = results.RootDexBatch
	}
	// exit without handling as the 'rootHeight' explicitly not set
	if remoteBatch == nil {
		return
	}
	// get the local locked dex batch for the counter chain
	localBatch, err := s.GetDexBatch(chainId, true)
	if err != nil {
		return
	}
	// if executing the liveness fallback (nested chain only)
	if remoteBatch.LivenessFallback {
		// handle the liveness fallback
		if err = s.HandleLivenessFallback(chainId, localBatch, remoteBatch); err != nil {
			return
		}
	}
	// handle the remote dex batch
	return s.HandleRemoteDexBatch(remoteBatch, chainId)
}

// HandleRemoteDexBatch is the core function driving the DEX batch lifecycle.
// ─────────────────────────────────────────────────────────────────────────────
// TERMINOLOGY
//   - Locked Batch — A finalized (frozen) batch currently awaiting processing
//     by the remote chain.
//   - Next Batch   — The batch currently collecting orders, deposits,
//     and withdrawals. This becomes the next Locked Batch.
//
// TRIGGER POINTS
//   - Nested Chain: Triggered on `end_block`.
//   - Root Chain:   Triggered on `deliver_tx` when receiving a
//     `certificateResultTx` from a Nested Chain.
//
// ─────────────────────────────────────────────────────────────────────────────
// LIFECYCLE STEPS
//
// 0) SAFETY CHECK
//   - Ensure our locked batch is completely satisfied
//     abort early and retry on the next trigger.
//
// ---------------------------------------------------------------------------
// 1) PROCESS RECEIPTS FOR *OUR* CURRENT LOCKED BATCH
//
//	1.a) Order Receipts
//	     • Success → Move order assets from HoldingPool → LiquidityPool.
//	     • Failure → Refund assets from HoldingPool → Seller.
//	     • Update both local and remote virtual pool ledgers.
//
//	1.b) Implied Withdrawal Receipts
//	     • Burn the proportional liquidity points.
//	     • Distribute the user’s allocated tokens for this chain.
//	     • Update all relevant pool ledgers.
//
//	1.c) Implied Deposit Receipts
//	     • Mint/distribute liquidity points based on local/remote pool states.
//	     • Move deposit from HoldingPool → LiquidityPool.
//
// ---------------------------------------------------------------------------
// 2) CREATE RECEIPTS FOR THE REMOTE CHAIN'S LOCKED BATCH
//
//	2.a) Order Receipts (processed in pseudorandom order)
//	     • Compute how many tokens the order would receive if successful.
//	     • If within order limit (slippage) → Pay from LiquidityPool and update dX.
//	     • Else → Mark as a failed order.
//	     • Record a Receipt into NextBatch and update all pool ledgers.
//
//	2.b) Withdrawal Receipts
//	     • Burn proportional liquidity points.
//	     • Distribute tokens for this chain.
//	     • Update pool ledgers.
//
//	2.c) Deposit Receipts
//	     • Distribute liquidity points based on pooled liquidity states.
//
// ---------------------------------------------------------------------------
// 3) ROTATE BATCHES
//   - Set NextBatch.poolSize = pool.Amount.
//   - Promote NextBatch → LockedBatch.
//   - Reset NextBatch for the next cycle.
//
// ─────────────────────────────────────────────────────────────────────────────
func (s *StateMachine) HandleRemoteDexBatch(remoteBatch *lib.DexBatch, chainId uint64) (err lib.ErrorI) {
	var receipts []uint64
	processRemote := !remoteBatch.IsEmpty()
	// copy because ledgers are mutated during processing
	remoteBatchCopy := remoteBatch.Copy()
	// local shadow of the counter chain pool (advanced as we apply receipts)
	counterPoolSizeMirror := remoteBatch.PoolSize
	// midpoint snapshot of our pool (after receipts, before counter-batch execution)
	midPointPoolSize, err := s.GetPoolBalance(chainId + LiquidityPoolAddend)
	if err != nil {
		return
	}
	if processRemote {
		var locked bool
		// 1) PROCESS RECEIPTS FOR OUR LOCKED BATCH (the locked batch previously sent to the counter chain)
		locked, err = s.HandleReceiptsForOurLockedBatch(remoteBatchCopy, &counterPoolSizeMirror, chainId)
		if err != nil || locked {
			return
		}
		// Step 2 (HandleRemoteChainLockedBatch) may further move the pool, but that movement is intentionally not reflected
		// in midPointPoolSize; the mid-point snapshot is what the counter chain needs to mirror our pool when it processes
		// receipts for our batch (so withdrawals/LP math on the other side uses the same pool size we had after applying
		// their receipts).
		midPointPoolSize, err = s.GetPoolBalance(chainId + LiquidityPoolAddend)
		if err != nil {
			return
		}
		// 2) CREATE RECEIPTS FOR THE REMOTE CHAIN'S LOCKED BATCH
		receipts, err = s.HandleRemoteChainLockedBatch(remoteBatchCopy, &counterPoolSizeMirror, chainId)
		if err != nil {
			return
		}
	}
	// 3) ROTATE BATCHES
	return s.RotateDexBatches(remoteBatch.Hash(), midPointPoolSize, counterPoolSizeMirror, chainId, receipts)
}

// HandleReceiptsForOurLockedBatch() 1. processes receipts for our locked batch
func (s *StateMachine) HandleReceiptsForOurLockedBatch(remoteBatch *lib.DexBatch, counterPoolSizeMirror *uint64, counterChainId uint64) (locked bool, err lib.ErrorI) {
	// get the local locked dex batch
	localBatch, err := s.GetDexBatch(counterChainId, true)
	if err != nil || localBatch.IsEmpty() {
		return false, err
	}
	// ensure receipt not mismatch (expected while waiting on counter chain)
	receiptMismatch := !bytes.Equal(remoteBatch.ReceiptHash, localBatch.Hash()) || len(localBatch.Orders) != len(remoteBatch.Receipts)
	if receiptMismatch {
		// purposefully only log here because while the origin chain waits for the counter to process their batch
		//   this error is expected
		// 1: Root chain & Nested Chain are perfectly in sync AND Nested Chain sends certificate result at END_BLOCK
		// 2: Nested Chain reads N-1 stale rootHeight because it's the latest Root Height as of LAST_QUORUM_CERTIFICATE
		//  Root Chain             | Nested Chain
		//  H=99                   | H=99 -> Sends Certificate Result Tx at END_BLOCK
		//  H=100 (Rec Cert Result)| H=100 Checking w/ RootHeight 99
		//  H=101 Shows Up In State| H=101 Checking w/ RootHeight 100
		//  H=102                  | H=102 Checking w/ RootHeight 101
		s.log.Debug(ErrMismatchDexBatchReceipt().Error())
		return true, nil
	}
	// get the local pool size
	localPoolSize, err := s.GetPoolBalance(counterChainId + LiquidityPoolAddend)
	if err != nil {
		return false, err
	}
	// handle receipts for orders within our locked batch:
	//  moving funds from holding pool to liquidity pool on success or refunding on fail
	if err = s.HandleOrderReceipts(localBatch, remoteBatch, counterChainId, &localPoolSize, counterPoolSizeMirror); err != nil {
		return false, err
	}
	// handle 'implied' receipts for liquidity withdrawals within our locked batch:
	//  burning points and distributing tokens from the liquid pool
	if err = s.HandleBatchWithdraw(localBatch, counterChainId, &localPoolSize, counterPoolSizeMirror, true); err != nil {
		return false, err
	}
	// handle 'implied' receipts for liquidity deposits within our locked batch:
	//  issuing points and moving tokens from the hold pool to the liquid pool
	if err = s.HandleBatchDeposit(localBatch, counterChainId, &localPoolSize, counterPoolSizeMirror, true); err != nil {
		return false, err
	}
	// remove lockedBatch to lift the 'atomic lock' - enabling orders to be sent in the next transaction
	return false, s.Delete(KeyForLockedBatch(counterChainId))
}

// HandleRemoteChainLockedBatch() 2. handles the locked batch for the remote chain, producing receipts
func (s *StateMachine) HandleRemoteChainLockedBatch(remoteBatch *lib.DexBatch, counterPoolSizeMirror *uint64, chainId uint64) (receipts []uint64, err lib.ErrorI) {
	// get the balance for the proper liquidity pool
	localPoolSize, err := s.GetPoolBalance(chainId + LiquidityPoolAddend)
	if err != nil {
		return
	}
	// handle the orders for the remote chain's locked batch (y represents the 'distribute pool balance')
	receipts, err = s.HandleDexBatchOrders(remoteBatch, counterPoolSizeMirror, &localPoolSize, chainId)
	if err != nil {
		return
	}
	// for each liquidity withdraws, move the funds from the liquidity pool to the account
	if err = s.HandleBatchWithdraw(remoteBatch, chainId, counterPoolSizeMirror, &localPoolSize, false); err != nil {
		return
	}
	// for each liquidity deposit, move the funds from the holding pool to the liquidity pool
	if err = s.HandleBatchDeposit(remoteBatch, chainId, counterPoolSizeMirror, &localPoolSize, false); err != nil {
		return
	}
	// exit
	return
}

// HandleOrderReceipts() handles receipts for orders within our locked batch
func (s *StateMachine) HandleOrderReceipts(localBatch, remoteBatch *lib.DexBatch, chainId uint64, x, y *uint64) (err lib.ErrorI) {
	// for each order, move the funds in the holding pool depending on the success or failure
	for i, o := range localBatch.Orders {
		// remove funds from the holding pool
		if err = s.PoolSub(chainId+HoldingPoolAddend, o.AmountForSale); err != nil {
			return
		}
		// convenience variable for amount counter asset amount received
		dY := remoteBatch.Receipts[i]
		success := dY != 0
		if success {
			// Mirror the remote chain's ledger:
			// - remoteBatch.PoolSize is our local shadow of the counter chain's pool at the mid-point snapshot they sent us.
			// - The counter chain already paid this receipt when it produced it; we subtract here to advance our shadow
			//   forward to their post-receipt state so subsequent AMM math (for their locked batch) uses the same reserves.
			if *y <= dY {
				return ErrRemotePoolSizeDebit()
			}
			// update ledgers
			*x += o.AmountForSale
			*y -= dY
			// add to the pool
			err = s.PoolAdd(chainId+LiquidityPoolAddend, o.AmountForSale)
		} else {
			// failed order
			err = s.AccountAdd(crypto.NewAddress(o.Address), o.AmountForSale)
		}
		if err != nil {
			return
		}
		// emit event
		if err = s.EventDexSwap(o.Address, o.OrderId, o.AmountForSale, dY,
			localBatch.Committee, true, success); err != nil {
			return
		}
	}
	// exit
	return
}

// HandleRemoteChainLockedBatch() executes AMM logic over a 'batch' of limit orders
// (1) sorts orders pseudorandomly by last block hash
// (2) determines successful orders & distributes from the liquidity pool
// (3) returns the receipts
// x = counter chain pool shadow, y = local pool (both advanced as orders execute)
func (s *StateMachine) HandleDexBatchOrders(remoteBatch *lib.DexBatch, x, y *uint64, chainId uint64) (receipts []uint64, err lib.ErrorI) {
	receipts, result := make([]uint64, len(remoteBatch.Orders)), map[string]uint64{}
	// load the last block from the indexer; caller triggers after genesis so height >= 1
	prevBlk, err := s.LoadBlock(s.Height() - 1)
	if err != nil || prevBlk == nil || prevBlk.BlockHeader == nil {
		return nil, lib.ErrNilBlock()
	}
	// make 2 copies of the orders with hash keys
	sorted, orders := remoteBatch.CopyOrders(prevBlk.BlockHeader.Hash)
	// sort pseudorandomly by hash key
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	if *x == 0 || *y == 0 {
		return nil, ErrInvalidLiquidityPool()
	}
	// for each order
	for _, order := range sorted {
		// set up 'deltaX'
		dX := order.AmountForSale
		// 'deltaY' = (dX * y) / (x + dX)
		dY := SafeComputeDY(*x, *y, dX)
		// if the distribute amount would be below the minimum requested: the order failed
		if dY < order.RequestedAmount {
			dY = 0
		}
		// capture result in map to save receipts later
		result[order.Key] = dY
		// emit swap event
		if err = s.EventDexSwap(order.Address, order.OrderId, dX, dY, chainId, false, dY != 0); err != nil {
			return
		}
		// if succeeded: update pool ledgers like uniswap would
		if dY != 0 {
			*x, *y = *x+dX, *y-dY
		}
	}
	// set success in the receipt
	for i, order := range orders {
		// setup convenience variable
		out := result[order.Key]
		// save receipt
		receipts[i] = out
		// if order succeeded
		if out != 0 {
			// distribute from pool
			if err = s.PoolSub(chainId+LiquidityPoolAddend, out); err != nil {
				return
			}
			// add to account
			if err = s.AccountAdd(crypto.NewAddress(order.Address), out); err != nil {
				return
			}
		}
	}
	return
}

// Two-chain LP accounting:
// - Mirror liquidity ledger on both chains for symmetry
// - Outbound deposits/withdraws: update ledger + move tokens
// - Inbound deposits/withdraws: update ledger but only token movement for withdraws

// HandleBatchWithdraw() handles local/remote liquidity withdraw requests.
// local=true: x=local pool, y=counter mirror; local=false: x=counter mirror, y=local pool.
func (s *StateMachine) HandleBatchWithdraw(batch *lib.DexBatch, counterChainId uint64, x, y *uint64, local bool) lib.ErrorI {
	if len(batch.Withdrawals) == 0 {
		return nil
	}
	// initialize vars
	var totalPointsToRemove uint64
	// get liquidity pool
	p, err := s.GetPool(counterChainId + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// collect withdrawals
	for _, w := range batch.Withdrawals {
		initialPoints, e := p.GetPointsFor(w.Address)
		if e != nil {
			s.log.Errorf("an error occurred retrieving the pool points for: %x, %s", w.Address, e.Error())
			continue // defensive
		}
		// update the total points to remove
		totalPointsToRemove += lib.SafeMulDiv(initialPoints, w.Percent, 100)
	}
	if totalPointsToRemove == 0 || p.TotalPoolPoints == 0 {
		return nil
	}
	// compute totals; actual paid amounts are tracked below to avoid burning rounding dust
	// x = 1000 RONI; totalPoolPoints = 100 ; Pablo has 50 points
	// y = 100 CNPY; totalPoolPoints = 100 ; Pablo has 50 points
	// Both the X and Y pool should have the same a) pool holders b) pool points per holder c) totalPoolPoints
	// Pablo has 50 pool points and Pablo withdrawals 50% = 25 points
	// AmountCNPYToReceiveInWithdrawal = 100 x 25 / 100
	// AmountRONIToReceiveInWithdrawal = 1000 x 25 / 100
	totalYWithdrawal := lib.SafeMulDiv(*y, totalPointsToRemove, p.TotalPoolPoints)
	totalXWithdraw := lib.SafeMulDiv(*x, totalPointsToRemove, p.TotalPoolPoints)
	var paidY, paidX uint64
	// distribute tokens
	for _, w := range batch.Withdrawals {
		initialPoints, e := p.GetPointsFor(w.Address)
		if e != nil {
			s.log.Warnf("an error occurred retrieving the pool points for: %x, %s", w.Address, e.Error())
			continue // defensive
		}
		// calculate points from percent
		points := lib.SafeMulDiv(initialPoints, w.Percent, 100)
		// calculate share
		yShare := lib.SafeMulDiv(totalYWithdrawal, points, totalPointsToRemove)
		// calculate virtual share
		xShare := lib.SafeMulDiv(totalXWithdraw, points, totalPointsToRemove)
		// remove points from pool
		if err = p.RemovePoints(w.Address, points); err != nil {
			return err
		}
		payout, counter := yShare, xShare
		if local {
			payout, counter = xShare, yShare
		}
		paidY += yShare
		paidX += xShare
		// credit user and update pool balance
		if err = s.AccountAdd(crypto.NewAddress(w.Address), payout); err != nil {
			return err
		}
		// emit withdraw event
		if err = s.EventDexLiquidityWithdraw(w.Address, w.OrderId, payout, counter, points, counterChainId); err != nil {
			return err
		}
	}
	// update the remote and local pool size ledgers using actual paid amounts (avoid burning rounding dust)
	*y -= paidY
	*x -= paidX
	// update the actual pool object in state
	if local {
		p.Amount = *x
	} else {
		p.Amount = *y
	}
	// set the pool in state
	return s.SetPool(p)
}

// HandleBatchDeposit() handles local/remote liquidity deposits.
// local=true: x=local pool (actual token movement), y=counter mirror. local=false: x=counter mirror, y=local pool.
func (s *StateMachine) HandleBatchDeposit(batch *lib.DexBatch, chainId uint64, x, y *uint64, local bool) lib.ErrorI {
	if len(batch.Deposits) == 0 {
		return nil
	}
	// get the liquidity pool
	p, err := s.GetPool(chainId + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// define variables
	var totalDeposit, distributed uint64
	// x = the initial 'deposit' pool balance
	// y = the 'counter' pool balance
	// L = initial pool points
	L := p.TotalPoolPoints
	// sum all deposits
	for _, deposit := range batch.Deposits {
		totalDeposit += deposit.Amount
	}
	// nothing to add or failed invariant check
	if totalDeposit == 0 || *x == 0 || *y == 0 {
		return nil
	}
	// if no liq points yet assigned - initialize to 'dead' address
	if L == 0 {
		// calculate the initial pool points using L = √( x * y )
		L = lib.SqrtProductUint64(*x, *y)
		// add points to the dead address
		p.AddPoints(deadAddr.Bytes(), L)
	}
	// using integer math and geometric mean of reserves:
	// deltaPoolPoints = L * ( √((x + totalDeposit) * y) - √(x * y) ) / √(x * y) or simplified as:
	// deltaPoolPoints = L * (newK - oldK) / oldK (in a 1 sided deposit scenario dY=0 thus this formula)
	oldK := lib.SqrtProductUint64(*x, *y)
	if oldK == 0 {
		return ErrInvalidLiquidityPool()
	}
	newK := lib.SqrtProductUint64(*x+totalDeposit, *y)
	// totalDL is calculated as if all deposits is just 1 big deposit
	totalDL := lib.SafeMulDiv(L, newK-oldK, oldK)
	// distribute the points
	for _, deposit := range batch.Deposits {
		// calculate pro-rate share for this particular deposit
		share := lib.SafeMulDiv(totalDL, deposit.Amount, totalDeposit)
		// update the distributed counter
		distributed += share
		// add points to pool
		p.AddPoints(deposit.Address, share)
		// if 'local' request - (actually move from holding pool to liquidity pool, don't *just* update the ledger)
		if local {
			if err = s.PoolSub(chainId+HoldingPoolAddend, deposit.Amount); err != nil {
				return err
			}
			p.Amount += deposit.Amount
		}
		// update the reserve
		*x += deposit.Amount
		// emit a deposit event
		if err = s.EventDexLiquidityDeposit(deposit.Address, deposit.OrderId, deposit.Amount, share, chainId, local); err != nil {
			return err
		}
	}
	// sink dust to the dead account
	p.AddPoints(deadAddr.Bytes(), totalDL-distributed)
	// update the pool
	return s.SetPool(p)
}

// RotateDexBatches() sets 'next batch' as 'locked batch' and deletes reference for 'next batch'
// (1) checks if locked batch is processed yet - if not exit
// (2) sets the upcoming 'sell' batch as 'last' sell batch
// (3) returns the upcoming 'sell' batch to be sent to the root
func (s *StateMachine) RotateDexBatches(receiptsHash []byte, lPoolSize, counterPoolSize, chainId uint64, receipts []uint64) (err lib.ErrorI) {
	// get locked sell batch
	lockedBatch, err := s.GetDexBatch(chainId, true)
	// exit with error or nil if last sell batch not yet processed by root (atomic protection)
	if err != nil || !lockedBatch.IsEmpty() {
		return
	}
	// get upcoming sell batch
	nextSellBatch, err := s.GetDexBatch(chainId, false)
	if err != nil {
		return
	}
	// set the pool size
	nextSellBatch.PoolSize = lPoolSize
	// set the hash
	nextSellBatch.ReceiptHash = receiptsHash
	// set the *computed* counter pool amount (informational; used for pricing display/RPC, not execution)
	nextSellBatch.CounterPoolSize = counterPoolSize
	// set the locked height
	nextSellBatch.LockedHeight = s.Height()
	// set receipts
	if len(receipts) != 0 {
		nextSellBatch.Receipts = receipts
	}
	// delete 'next sell batch'
	if err = s.Delete(KeyForNextBatch(chainId)); err != nil {
		return
	}
	// set the upcoming sell batch as 'last'
	return s.SetDexBatch(KeyForLockedBatch(chainId), nextSellBatch)
}

// HandleLivenessFallback() refunds orders, liquidity deposits, and mirrors the root chain's liquidity points
func (s *StateMachine) HandleLivenessFallback(rcId uint64, localBatch, remoteBatch *lib.DexBatch) (err lib.ErrorI) {
	s.log.Warnf("Dex liveness fallback: refunding orders and dropping locked batch")
	// define a convenience function
	refund := func(address []byte, amount uint64) (e lib.ErrorI) {
		if e = s.PoolSub(rcId+HoldingPoolAddend, amount); e != nil {
			return e
		}
		if e = s.AccountAdd(crypto.NewAddress(address), amount); e != nil {
			return e
		}
		return
	}
	// refund all orders
	for _, order := range localBatch.Orders {
		if err = refund(order.Address, order.AmountForSale); err != nil {
			return
		}
	}
	// refund the liquidity deposits
	for _, deposit := range localBatch.Deposits {
		if err = refund(deposit.Address, deposit.Amount); err != nil {
			return
		}
	}
	// update the liquidity points to mirror the counter
	if err = s.SetPoolPoints(rcId+LiquidityPoolAddend, remoteBatch.GetPoolPoints(), remoteBatch.GetTotalPoolPoints()); err != nil {
		return
	}
	// drop the locked batch
	if err = s.SetDexBatch(KeyForLockedBatch(rcId), &lib.DexBatch{}); err != nil {
		return
	}
	return
}

// HELPERS BELOW

// SetDexBatch() sets a sell batch in the state store
func (s *StateMachine) SetDexBatch(key []byte, b *lib.DexBatch) (err lib.ErrorI) {
	value, err := lib.Marshal(b)
	if err != nil {
		return
	}
	return s.Set(key, value)
}

// GetDexBatch() retrieves a sell batch from the state store
func (s *StateMachine) GetDexBatch(chainId uint64, locked bool, withPoints ...bool) (b *lib.DexBatch, err lib.ErrorI) {
	var lPool *Pool
	key := KeyForNextBatch(chainId)
	if locked {
		key = KeyForLockedBatch(chainId)
	}
	// get bytes from state
	bz, err := s.Get(key)
	if err != nil {
		return
	}
	// retrieve the liquidity pool from state
	lPool, err = s.GetPool(chainId + LiquidityPoolAddend)
	if err != nil {
		return
	}
	// create a new batch object reference to ensure no 'nil' batches are used
	b = &lib.DexBatch{Committee: chainId, PoolSize: lPool.Amount}
	defer b.EnsureNonNil()
	// check for nil bytes
	if len(bz) == 0 {
		return
	}
	// populate the batch object with the bytes
	err = lib.Unmarshal(bz, b)
	// check if points should be attached
	if len(withPoints) == 1 && withPoints[0] {
		// set the pool points
		b.PoolPoints = lPool.Points
		// set total pool points
		b.TotalPoolPoints = lPool.TotalPoolPoints
	}
	// exit
	return
}

// GetDexBatches() retrieves the lists for all dex batches
func (s *StateMachine) GetDexBatches(lockedBatch bool) (b []*lib.DexBatch, err lib.ErrorI) {
	b = make([]*lib.DexBatch, 0)
	// create a prefix to iterate over
	var prefix []byte
	// create an iterator over the dex batches
	if lockedBatch {
		prefix = lib.JoinLenPrefix(dexPrefix, lockedBatchSegment)
	} else {
		prefix = lib.JoinLenPrefix(dexPrefix, nextBatchSement)
	}
	// iterate over the dex prefix
	it, err := s.Iterator(prefix)
	if err != nil {
		return
	}
	// memory cleanup the iterator
	defer it.Close()
	// for each item under the dex prefix
	for ; it.Valid(); it.Next() {
		batch := new(lib.DexBatch)
		// unmarshal to dex batch
		if err = lib.Unmarshal(it.Value(), batch); err != nil {
			s.log.Error(err.Error())
		}
		// add the batch to the list
		b = append(b, batch)
	}
	// exit
	return
}

// GetDexPrice() returns the chain price
func (s *StateMachine) GetDexPrice(chainId uint64) (p *lib.DexPrice, err lib.ErrorI) {
	// get the dex batch
	dexBatch, err := s.GetDexBatch(chainId, true)
	if err != nil {
		return
	}
	return s.getPrice(dexBatch)
}

// GetDexPrices() returns the prices for all 'locked' batches
func (s *StateMachine) GetDexPrices() (p []*lib.DexPrice, err lib.ErrorI) {
	// get the dex batch
	dexBatches, err := s.GetDexBatches(true)
	if err != nil {
		return
	}
	// for each dex batch
	for _, batch := range dexBatches {
		price, e := s.getPrice(batch)
		if e == nil {
			p = append(p, price)
		}
	}
	return
}

// getPrice() returns the dex price for a dex batch
func (s *StateMachine) getPrice(batch *lib.DexBatch) (p *lib.DexPrice, err lib.ErrorI) {
	// if the dex batch is uninitalized
	if batch.PoolSize == 0 || batch.CounterPoolSize == 0 {
		return nil, ErrInvalidLiquidityPool()
	}
	// exit with the dex price
	return &lib.DexPrice{
		LocalChainId:  s.Config.ChainId,
		RemoteChainId: batch.Committee,
		LocalPool:     batch.PoolSize,
		RemotePool:    batch.CounterPoolSize,
		E6ScaledPrice: batch.PoolSize * 1_000_000 / batch.CounterPoolSize,
	}, nil
}

// SafeComputeDY() executes overflow protected uniswap V2 formula
func SafeComputeDY(x, y, dX uint64) uint64 {
	bx := new(big.Int).SetUint64(x)
	by := new(big.Int).SetUint64(y)
	bdX := new(big.Int).SetUint64(dX)

	// amountInWithFee = dX * 997
	amountInWithFee := new(big.Int).Mul(bdX, big.NewInt(997))

	// numerator = amountInWithFee * y
	numerator := new(big.Int).Mul(amountInWithFee, by)

	// denominator = x*1000 + amountInWithFee
	denominator := new(big.Int).Mul(bx, big.NewInt(1000))
	denominator.Add(denominator, amountInWithFee)

	// dY = numerator / denominator
	dY := new(big.Int).Div(numerator, denominator)

	// integer flooring
	return dY.Uint64()
}

var deadAddr, _ = crypto.NewAddressFromString(strings.Repeat("dead", 10))
