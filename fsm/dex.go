package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"sort"
	"strings"
)

/* Dex.go implements logic to handle AMM style atomic exchanges between root & nested chains
   not to be confused with 1 way order book swaps implemented in swap.go */

// HandleDexBatch() initiates the 'dex' lifecycle
func (s *StateMachine) HandleDexBatch(rootBuildHeight, chainId uint64, buyBatch *lib.DexBatch) (err lib.ErrorI) {
	// handle 'self certificate' as a trigger to get dex data from the root chain
	if chainId == s.Config.ChainId {
		// set 'chain id' as the root chain
		chainId, err = s.GetRootChainId()
		if err != nil {
			return
		}
		// exit without handling as the 'rootBuildHeight' explicitly not set
		if rootBuildHeight == 0 {
			return nil
		}
		// get root chain dex batch
		buyBatch, err = s.RCManager.GetDexBatch(chainId, rootBuildHeight, s.Config.ChainId)
		// return err or nil if dex data is empty
		if err != nil {
			return
		}
	}
	// handle the remote dex batch
	return s.HandleRemoteDexBatch(buyBatch)
}

// HandleRemoteDexBatch() handles the 'dex lifecycle'
//
//	**Trigger:**
//	- **Nested Chain**: On `begin_block` after its own `CertificateResult`
//	- **Root Chain**: On `deliver_tx` with `certificateResultTx` from Nested Chain
//
//	**Steps:**
//
//	1. **Process Inbound Receipts for local LockedBatch**
//	- **If** `LockedBatch` exists but its receipts aren’t fully matched → exit (retry next trigger).
//	- **Else If** Success → move from `HoldingPool` → `LiquidityPool`
//	- **Else If** Fail → refund from `HoldingPool` → seller
//
//	2. **Process local LockedBatch Liquidity**
//	- **Withdraw**: burn points, distribute tokens pro-rata
//	- **Deposit**: pro-rata distribute liquidity points using the included virtualPoolSize
//
//	3. **Process Inbound Sell Orders**
//	- For each order:
//	1. Compute uniform clearing price (AMM curve)
//	2. If within limit → pay from `LiquidityPool` and update virtualPoolSize (dX)
//	3. Record outcome as `BuyReceipt` in `NextBatch`
//
//	4. **Process Inbound Liquidity**
//	- **Withdraw**: burn points, distribute tokens, update virtualPoolSize from 3.
//	- **Deposit**: assign points using virtualPoolSize from 3.
//
//	5. **Rotate Batches**
//	- set `poolSize` of NextBatch = pool.Amount
//	- set `LockedBatch = NextBatch`
//	- Reset `NextBatch`
func (s *StateMachine) HandleRemoteDexBatch(remoteBatch *lib.DexBatch) (err lib.ErrorI) {
	// exit if dex data is empty
	if remoteBatch == nil {
		return
	}
	// handle the batch receipt - this function manages atomicity of the dex operations
	locked, err := s.HandleDexBatchReceipt(remoteBatch)
	if err != nil || locked {
		return
	}
	// load the last block from the indexer
	lastBlock, err := s.LoadBlock(s.Height() - 1)
	if err != nil {
		return
	}
	// process batch - save receipts to state machine
	receipts, err := s.HandleDexBatchOrders(remoteBatch, lastBlock.BlockHeader.Hash)
	if err != nil {
		return
	}
	// (6) set the 'next batch' as 'locked batch' in state with receipts and pool size
	return s.RotateDexSellBatch(remoteBatch, receipts)
}

// HandleDexBatchReceipt() handles a 'receipt' for a 'sell batch' and 'locked' liquidity commands
// Sends funds from holding pool to either the liquidity pool or back to sender
func (s *StateMachine) HandleDexBatchReceipt(remoteBatch *lib.DexBatch) (locked bool, err lib.ErrorI) {
	// get locked sell batch
	localLockedBatch, err := s.GetDexBatch(KeyForLockedBatch(remoteBatch.Committee))
	if err != nil {
		return true, err
	}
	// check if locked batch is empty
	if localLockedBatch.IsEmpty() {
		return false, nil
	}
	// ensure receipt not mismatch
	if !bytes.Equal(remoteBatch.ReceiptHash, localLockedBatch.Hash()) || len(localLockedBatch.Orders) != len(remoteBatch.Receipts) {
		return true, ErrMismatchDexBatchReceipt()
	}
	// for each order, move the funds in the holding pool depending on the success or failure
	for i, order := range localLockedBatch.Orders {
		// remove funds from the holding pool
		if err = s.PoolSub(s.Config.ChainId+HoldingPoolAddend, order.AmountForSale); err != nil {
			return true, err
		}
		// if order succeeded, add funds to the liquidity pool, else revert back to sender
		if remoteBatch.Receipts[i] {
			err = s.PoolAdd(s.Config.ChainId+LiquidityPoolAddend, order.AmountForSale)
		} else {
			err = s.AccountAdd(crypto.NewAddress(order.Address), order.AmountForSale)
		}
		if err != nil {
			return true, err
		}
	}
	// ensure the proper pool size
	localLockedBatch.PoolSize, err = s.GetPoolBalance(s.Config.ChainId + LiquidityPoolAddend)
	if err != nil {
		return false, err
	}
	// for each liquidity withdraws, move the funds from the liquidity pool to the account
	remoteVirtualPoolSize, err := s.HandleBatchWithdraw(localLockedBatch, remoteBatch.PoolSize, true)
	if err != nil {
		return false, err
	}
	// for each liquidity deposit, move the funds from the holding pool to the liquidity pool
	if err = s.HandleBatchDeposit(localLockedBatch, remoteVirtualPoolSize, true); err != nil {
		return false, err
	}
	// remove lockedBatch to lift the 'atomic lock' - enabling orders to be sent in the next transaction
	return false, s.Delete(KeyForLockedBatch(remoteBatch.Committee))
}

// HandleDexBatchOrders() executes AMM logic over a 'batch' of limit orders
// (1) calculates the uniform clearing price for a batch
// (2) determines successful orders & distributes from the liquidity pool
// (3) handle 'inbound' liquidity deposits/withdraws
// (4) returns the receipts
func (s *StateMachine) HandleDexBatchOrders(remoteBatch *lib.DexBatch, blockHash []byte) (success []bool, err lib.ErrorI) {
	// initialize the receipt
	success = make([]bool, len(remoteBatch.Orders))
	// initialize a map to track accepted
	accepted := make(map[string]uint64) // map_key -> output
	// get the buy pool size (pool where distributions are made from)
	distributePool, err := s.GetPool(s.Config.ChainId + LiquidityPoolAddend)
	if err != nil {
		return nil, err
	}
	// set pools
	x, y := remoteBatch.PoolSize, distributePool.Amount
	// calculate k
	k := x * y
	// handle bad k
	if k == 0 {
		return nil, ErrInvalidLiquidityPool()
	}
	// make a copy of the orders
	orders := remoteBatch.CopyOrders()
	// sort pseudorandomly
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].MapKey(blockHash) < orders[j].MapKey(blockHash) // TODO cache map key inside order structure
	})
	// working reserves
	newX, newY := x, y
	for _, order := range orders {
		// set dX
		dX := order.AmountForSale
		// uniswap V2 formula
		amountInWithFee := dX * 997
		// calculate the new dY (out)
		dY := (amountInWithFee * newY) / (newX*1000 + amountInWithFee)
		// if below 'limit'
		if dY < order.RequestedAmount {
			continue
		}
		// accept order
		accepted[order.MapKey(blockHash)] = dY
		// update pool reserves like uniswap would
		newX, newY = newX+dX, newY-dY
	}
	// set success in the receipt
	for i, order := range remoteBatch.Orders {
		var out uint64
		if out, success[i] = accepted[order.MapKey(blockHash)]; success[i] {
			// distribute from pool
			if err = s.PoolSub(s.Config.ChainId+LiquidityPoolAddend, out); err != nil {
				return nil, err
			}
			// add to account
			if err = s.AccountAdd(crypto.NewAddress(order.Address), out); err != nil {
				return nil, err
			}
		}
	}
	// get the 'counter' pool balance
	lPool, err := s.GetPool(s.Config.ChainId + LiquidityPoolAddend)
	if err != nil {
		return nil, err
	}
	// save the original pool size to restore later
	originalPoolSize := remoteBatch.PoolSize
	// update the virtual counter pool amount
	remoteBatch.PoolSize = newX
	// for each liquidity withdraws, move the funds from the liquidity pool to the account
	lPool.Amount, err = s.HandleBatchWithdraw(remoteBatch, lPool.Amount, false)
	if err != nil {
		return nil, err
	}
	// for each liquidity deposit, move the funds from the holding pool to the liquidity pool
	if err = s.HandleBatchDeposit(remoteBatch, lPool.Amount, false); err != nil {
		return nil, err
	}
	// restore original pool size
	remoteBatch.PoolSize = originalPoolSize
	// set pool
	return success, nil
}

// Two-chain LP accounting:
// - Mirror liquidity ledger on both chains for symmetry
// - Outbound deposits/withdraws: update ledger + move tokens (use dX from UCP)
// - Inbound deposits/withdraws: update ledger but only token movement for withdraws

// HandleBatchDeposit() handles inbound/outbound liquidity deposits
func (s *StateMachine) HandleBatchDeposit(batch *lib.DexBatch, counterPoolAmount uint64, outbound bool) lib.ErrorI {
	// get the liquidity pool
	p, err := s.GetPool(batch.Committee + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// sum deposits
	var totalDeposit uint64
	for _, order := range batch.Deposits {
		totalDeposit += order.Amount
	}
	// x = the initial 'deposit' pool balance
	// y = the 'counter' pool balance
	// L = initial liquidity points
	x, y, L := batch.PoolSize, counterPoolAmount, p.TotalPoolPoints
	// nothing to add or failed invariant check
	if totalDeposit == 0 || x == 0 || y == 0 {
		return nil
	}
	// if no liq points yet assigned - initialize to 'dead' address
	if L == 0 {
		// setup dead address
		deadAddr, _ := crypto.NewAddressFromString(strings.Repeat("dead", 10))
		// calculate the initial liquidity points using L = √( x * y )
		L = lib.IntSqrt(x * y)
		// add points to the dead address
		if err = p.AddPoints(deadAddr.Bytes(), L); err != nil {
			return err
		}
	}
	// calculate dL as if it's one big deposit
	// using integer math and geometric mean of reserves:
	// ΔL = L * ( √((x + totalDeposit) * y) - √(x * y) ) / √(x * y)
	oldK := x * y
	newK := (x + totalDeposit) * y
	totalDL := L * (lib.IntSqrt(newK) - lib.IntSqrt(oldK)) / lib.IntSqrt(oldK)
	// pro-rata distribute the points
	for _, order := range batch.Deposits {
		// calculate pro-rate share
		share := totalDL * order.Amount / totalDeposit
		// add points to pool
		if err = p.AddPoints(order.Address, share); err != nil {
			return err
		}
		// update pool
		if outbound {
			// remove from holding pool
			if err = s.PoolSub(s.Config.ChainId+HoldingPoolAddend, order.Amount); err != nil {
				return err
			}
			// add to the liquidity pool amount
			p.Amount += order.Amount
		}
	}
	// update the pool
	return s.SetPool(p)
}

// HandleBatchWithdraw() handles inbound/outbound liquidity withdraw requests
// NOTE: withdraws should come before deposits because it affects pool size
func (s *StateMachine) HandleBatchWithdraw(batch *lib.DexBatch, yPoolSize uint64, outbound bool) (uint64, lib.ErrorI) {
	// initialize vars
	totalPointsToRemove, withdraws := uint64(0), make(map[string]uint64)
	// get pool
	p, err := s.GetPool(batch.Committee + LiquidityPoolAddend)
	if err != nil {
		return 0, err
	}
	// collect withdrawals
	for _, w := range batch.Withdraws {
		initialPoints, e := p.GetPointsFor(w.Address)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		// calculate the points to withdraw
		pointsToRemove := initialPoints * w.Percent / 100
		// increment the points to remove
		withdraws[lib.BytesToString(w.Address)] += pointsToRemove
		// update the total points to remove
		totalPointsToRemove += pointsToRemove
	}
	// if nothing to withdraw
	if totalPointsToRemove == 0 {
		return yPoolSize, nil
	}
	// total reserve to withdraw
	totalReserve := yPoolSize * totalPointsToRemove / p.TotalPoolPoints
	// total virtual reserve to withdraw
	totalVReserve := batch.PoolSize * totalPointsToRemove / p.TotalPoolPoints
	// pro-rata distribute
	for address, points := range withdraws {
		// get address
		addr, _ := lib.StringToBytes(address)
		// calculate share
		share := totalReserve * points / totalPointsToRemove
		// calculate virtual share
		vShare := totalVReserve * points / totalPointsToRemove
		// defensive 1
		if share > yPoolSize {
			share = yPoolSize
		}
		// defensive 2
		if vShare > batch.PoolSize {
			vShare = batch.PoolSize
		}
		// update pools
		yPoolSize -= share
		batch.PoolSize -= vShare
		// remove points from pool
		if err = p.RemovePoints(addr, points); err != nil {
			return 0, err
		}
		if outbound {
			share = vShare
		}
		// credit user
		if err = s.AccountAdd(crypto.NewAddress(addr), share); err != nil {
			return 0, err
		}
	}
	// if outbound
	if outbound {
		p.Amount = batch.PoolSize
	} else {
		p.Amount = yPoolSize
	}
	// return the pool size
	return yPoolSize, s.SetPool(p)
}

// RotateDexSellBatch() sets 'next batch' as 'locked batch' and deletes reference for 'next batch'
// (1) checks if locked batch is processed yet - if not exit
// (2) sets the upcoming 'sell' batch as 'last' sell batch
// (3) returns the upcoming 'sell' batch to be sent to the root
func (s *StateMachine) RotateDexSellBatch(buyBatch *lib.DexBatch, receipts []bool) (err lib.ErrorI) {
	// defensive check for nil buyBatch
	if buyBatch == nil {
		return ErrInvalidInput()
	}
	// get locked sell batch
	lockedBatch, err := s.GetDexBatch(KeyForLockedBatch(buyBatch.Committee))
	// exit with error or nil if last sell batch not yet processed by root (atomic protection)
	if err != nil || !lockedBatch.IsEmpty() {
		return
	}
	// get upcoming sell batch
	nextSellBatch, err := s.GetDexBatch(KeyForNextBatch(buyBatch.Committee))
	if err != nil {
		return
	}
	// get the liquidity pool size
	lPool, err := s.GetPool(buyBatch.Committee + LiquidityPoolAddend)
	if err != nil {
		return
	}
	// set the pool size
	nextSellBatch.PoolSize = lPool.Amount
	// set the hash
	nextSellBatch.ReceiptHash = buyBatch.Hash()
	// set receipts
	if len(receipts) != 0 {
		nextSellBatch.Receipts = receipts
	}
	// delete 'next sell batch'
	if err = s.Delete(KeyForNextBatch(buyBatch.Committee)); err != nil {
		return
	}
	// set the upcoming sell batch as 'last'
	err = s.SetDexBatch(KeyForLockedBatch(buyBatch.Committee), nextSellBatch)
	// exit
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
func (s *StateMachine) GetDexBatch(key []byte) (b *lib.DexBatch, err lib.ErrorI) {
	// get bytes from state
	bz, err := s.Get(key)
	if err != nil {
		return
	}
	// create a new batch object reference to ensure no 'nil' batches are used
	b = &lib.DexBatch{Committee: s.Config.ChainId}
	defer b.EnsureNonNil()
	// check for nil bytes
	if len(bz) == 0 {
		return
	}
	// populate the batch object with the bytes
	err = lib.Unmarshal(bz, b)
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
