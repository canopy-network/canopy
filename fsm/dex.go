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

// HandleDexBatch() initiates the 'dex' lifecycle
func (s *StateMachine) HandleDexBatch(rootBuildHeight, chainId uint64, remoteBatch *lib.DexBatch) (err lib.ErrorI) {
	// exit without handling as the 'rootBuildHeight' explicitly not set
	if rootBuildHeight == 0 {
		return
	}
	// handle 'self certificate' as a trigger to get dex data from the root chain
	if chainId == s.Config.ChainId {
		// set 'chain id' as the root chain
		if chainId, err = s.GetRootChainId(); err != nil {
			return
		}
		// get root chain dex batch
		if remoteBatch, err = s.RCManager.GetDexBatch(chainId, rootBuildHeight, s.Config.ChainId, false); err != nil {
			return
		}
	}
	// handle the remote dex batch
	return s.HandleRemoteDexBatch(remoteBatch, chainId)
}

// HandleRemoteDexBatch() is the main function of the 'dex lifecycle'
//
//	Trigger:
//	- Nested Chain: On `begin_block` after its own `CertificateResult`
//	- Root Chain: On `deliver_tx` with `certificateResultTx` from Nested Chain
//
//	Steps:
//
//	1. Process Inbound Receipts for local LockedBatch
//	- If `LockedBatch` exists but its receipts aren’t fully matched → exit (retry next trigger).
//	- Else If Success → move from `HoldingPool` → `LiquidityPool`
//	- Else If Fail → refund from `HoldingPool` → seller
//
//	2. Process local LockedBatch Liquidity
//	- Withdraw: burn points, distribute tokens pro-rata
//	- Deposit: pro-rata distribute liquidity points using the included virtualPoolSize
//
//	3. Process Inbound Sell Orders
//	- For each orders sorted pseudorandomly:
//	1. Calculate the price it 'would move' if succeed
//	2. If within limit → pay from `LiquidityPool` and update virtualPoolSize (dX)
//	3. Record outcome as `BuyReceipt` in `NextBatch`
//
//	4. Process Inbound Liquidity
//	- Withdraw: burn points, distribute tokens, update virtualPoolSize from 3.
//	- Deposit: assign points using virtualPoolSize from 3.
//
//	5. Rotate Batches
//	- set `poolSize` of NextBatch = pool.Amount
//	- set `LockedBatch = NextBatch`
//	- Reset `NextBatch`
func (s *StateMachine) HandleRemoteDexBatch(remoteBatch *lib.DexBatch, chainId uint64) (err lib.ErrorI) {
	// exit if dex data is empty
	if remoteBatch == nil {
		return
	}
	// handle the batch receipt - this function manages atomicity of the dex operations
	locked, err := s.HandleDexBatchReceipt(remoteBatch, chainId)
	if err != nil || locked {
		return
	}
	// process batch - save receipts to state machine
	receipts, err := s.HandleDexBatchOrders(remoteBatch, chainId)
	if err != nil {
		return
	}
	// set the 'next batch' as 'locked batch' in state with receipts and pool size
	return s.RotateDexSellBatch(remoteBatch, chainId, receipts)
}

// HandleDexBatchReceipt() handles a 'receipt' for a 'sell batch' and 'locked' liquidity commands
// Sends funds from holding pool to either the liquidity pool or back to sender
func (s *StateMachine) HandleDexBatchReceipt(remoteBatch *lib.DexBatch, chainId uint64) (locked bool, err lib.ErrorI) {
	// get the local locked dex batch
	localBatch, err := s.GetDexBatch(chainId, true)
	if err != nil {
		return true, err
	}
	// check if locked batch is empty
	if localBatch.IsEmpty() {
		return false, nil
	}
	// ensure receipt not mismatch
	if !bytes.Equal(remoteBatch.ReceiptHash, localBatch.Hash()) || len(localBatch.Orders) != len(remoteBatch.Receipts) {
		return true, ErrMismatchDexBatchReceipt()
	}
	// for each order, move the funds in the holding pool depending on the success or failure
	for i, order := range localBatch.Orders {
		// remove funds from the holding pool
		if err = s.PoolSub(chainId+HoldingPoolAddend, order.AmountForSale); err != nil {
			return true, err
		}
		// if order succeeded, add funds to the liquidity pool, else revert back to sender
		if remoteBatch.Receipts[i] {
			err = s.PoolAdd(chainId+LiquidityPoolAddend, order.AmountForSale)
		} else {
			err = s.AccountAdd(crypto.NewAddress(order.Address), order.AmountForSale)
		}
		if err != nil {
			return true, err
		}
	}
	// ensure the proper pool size
	if localBatch.PoolSize, err = s.GetPoolBalance(chainId + LiquidityPoolAddend); err != nil {
		return false, err
	}
	// for each liquidity withdraws, move the funds from the liquidity pool to the account
	if err = s.HandleBatchWithdraw(localBatch, chainId, &remoteBatch.PoolSize, true); err != nil {
		return false, err
	}
	// for each liquidity deposit, move the funds from the holding pool to the liquidity pool
	if err = s.HandleBatchDeposit(localBatch, chainId, remoteBatch.PoolSize, true); err != nil {
		return false, err
	}
	// remove lockedBatch to lift the 'atomic lock' - enabling orders to be sent in the next transaction
	return false, s.Delete(KeyForLockedBatch(chainId))
}

// HandleDexBatchOrders() executes AMM logic over a 'batch' of limit orders
// (1) sorts orders pseudorandomly by last block hash
// (2) determines successful orders & distributes from the liquidity pool
// (3) handle 'inbound' liquidity deposits/withdraws
// (4) returns the receipts
func (s *StateMachine) HandleDexBatchOrders(remoteBatch *lib.DexBatch, chainId uint64) (success []bool, err lib.ErrorI) {
	success, accepted, restore := make([]bool, len(remoteBatch.Orders)), map[string]uint64{}, remoteBatch.PoolSize
	// restore original pool size at the end of the function
	defer func() { remoteBatch.PoolSize = restore }()
	// get the buy pool size (pool where distributions are made from)
	distributePoolAmount, err := s.GetPoolBalance(chainId + LiquidityPoolAddend)
	if err != nil {
		return
	}
	// load the last block from the indexer
	prevBlk, err := s.LoadBlock(s.Height() - 1)
	if err != nil || prevBlk == nil || prevBlk.BlockHeader == nil {
		return
	}
	// make 2 copies of the orders with hash keys
	sorted, orders := remoteBatch.CopyOrders(prevBlk.BlockHeader.Hash)
	// sort pseudorandomly by hash key
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	// setup working reserves
	x, y := &remoteBatch.PoolSize, distributePoolAmount
	if *x == 0 || y == 0 {
		return nil, ErrInvalidLiquidityPool()
	}
	// for each order
	for _, order := range sorted {
		// set dX
		dX := order.AmountForSale
		// dY = (dX * y) / (x + dX)
		dY := SafeComputeDY(*x, y, dX)
		// fail (below limit)
		if dY < order.RequestedAmount {
			continue
		}
		// success [map-key] -> output amount
		accepted[order.Key] = dY
		// update pool reserves like uniswap would
		*x, y = *x+dX, y-dY
	}
	var out uint64
	// set success in the receipt
	for i, order := range orders {
		if out, success[i] = accepted[order.Key]; success[i] {
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
	// for each liquidity withdraws, move the funds from the liquidity pool to the account
	if err = s.HandleBatchWithdraw(remoteBatch, chainId, &y, false); err != nil {
		return
	}
	// for each liquidity deposit, move the funds from the holding pool to the liquidity pool
	if err = s.HandleBatchDeposit(remoteBatch, chainId, y, false); err != nil {
		return
	}
	// exit
	return
}

// Two-chain LP accounting:
// - Mirror liquidity ledger on both chains for symmetry
// - Outbound deposits/withdraws: update ledger + move tokens (use dX from order clearing)
// - Inbound deposits/withdraws: update ledger but only token movement for withdraws

// HandleBatchDeposit() handles local/remote liquidity deposits
func (s *StateMachine) HandleBatchDeposit(batch *lib.DexBatch, chainId, y uint64, local bool) lib.ErrorI {
	// get the liquidity pool
	p, err := s.GetPool(chainId + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// sum deposits
	var totalDeposit, distributed uint64
	for _, order := range batch.Deposits {
		totalDeposit += order.Amount
	}
	// x = the initial 'deposit' pool balance
	// y = the 'counter' pool balance
	// L = initial liquidity points
	x, L := batch.PoolSize, p.TotalPoolPoints
	// nothing to add or failed invariant check
	if totalDeposit == 0 || x == 0 || y == 0 {
		return nil
	}
	// if no liq points yet assigned - initialize to 'dead' address
	if L == 0 {
		// calculate the initial liquidity points using L = √( x * y )
		L = lib.SqrtProductUint64(x, y)
		// add points to the dead address
		if err = p.AddPoints(deadAddr.Bytes(), L); err != nil {
			return err
		}
	}
	// calculate dL as if it's one big deposit
	// using integer math and geometric mean of reserves:
	// ΔL = L * ( √((x + totalDeposit) * y) - √(x * y) ) / √(x * y)
	oldK := lib.SqrtProductUint64(x, y)
	newK := lib.SqrtProductUint64(x+totalDeposit, y)
	// L * (newK - oldK) / oldK
	totalDL := lib.SafeMulDiv(L, newK-oldK, oldK)
	// pro-rata distribute the points
	for _, order := range batch.Deposits {
		// calculate pro-rate share
		share := lib.SafeMulDiv(totalDL, order.Amount, totalDeposit)
		// update distributed
		distributed += share
		// add points to pool
		if err = p.AddPoints(order.Address, share); err != nil {
			return err
		}
		// if 'local' request - move from holding pool to liquidity pool
		if local {
			if err = s.PoolSub(chainId+HoldingPoolAddend, order.Amount); err != nil {
				return err
			}
			p.Amount += order.Amount
		}
	}
	// sink dust to the dead account
	if err = p.AddPoints(deadAddr.Bytes(), totalDL-distributed); err != nil {
		return err
	}
	// update the pool
	return s.SetPool(p)
}

// HandleBatchWithdraw() handles local/remote liquidity withdraw requests
func (s *StateMachine) HandleBatchWithdraw(batch *lib.DexBatch, chainId uint64, y *uint64, local bool) lib.ErrorI {
	// initialize vars
	totalPointsToRemove, withdraws := uint64(0), make(map[string]uint64)
	// get liquidity pool
	p, err := s.GetPool(chainId + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// collect withdrawals
	for _, w := range batch.Withdraws {
		initialPoints, e := p.GetPointsFor(w.Address)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		// calculate the points to withdraw
		pointsToRemove := lib.SafeMulDiv(initialPoints, w.Percent, 100)
		if e != nil {
			return e
		}
		// increment the points to remove
		withdraws[lib.BytesToString(w.Address)] += pointsToRemove
		// update the total points to remove
		totalPointsToRemove += pointsToRemove
	}
	// if nothing to withdraw
	if totalPointsToRemove == 0 {
		return nil
	}
	// total local reserve to withdraw
	totalYWithdrawal := lib.SafeMulDiv(*y, totalPointsToRemove, p.TotalPoolPoints)
	// total remote reserve to withdraw
	totalXWithdraw := lib.SafeMulDiv(batch.PoolSize, totalPointsToRemove, p.TotalPoolPoints)
	// pro-rata distribute
	for address, points := range withdraws {
		// get address
		addr, _ := lib.StringToBytes(address)
		// calculate share
		yShare := lib.SafeMulDiv(totalYWithdrawal, points, totalPointsToRemove)
		// calculate virtual share
		xShare := lib.SafeMulDiv(totalXWithdraw, points, totalPointsToRemove)
		// remove points from pool
		if err = p.RemovePoints(addr, points); err != nil {
			return err
		}
		// credit user and update pool balance
		if local {
			if err = s.AccountAdd(crypto.NewAddress(addr), xShare); err != nil {
				return err
			}
		} else {
			if err = s.AccountAdd(crypto.NewAddress(addr), yShare); err != nil {
				return err
			}
		}
	}
	// burn undistributed
	*y -= totalYWithdrawal
	batch.PoolSize -= totalXWithdraw
	if local {
		p.Amount = batch.PoolSize
	} else {
		p.Amount = *y
	}
	// set the pool in state
	return s.SetPool(p)
}

// RotateDexSellBatch() sets 'next batch' as 'locked batch' and deletes reference for 'next batch'
// (1) checks if locked batch is processed yet - if not exit
// (2) sets the upcoming 'sell' batch as 'last' sell batch
// (3) returns the upcoming 'sell' batch to be sent to the root
func (s *StateMachine) RotateDexSellBatch(remoteBatch *lib.DexBatch, chainId uint64, receipts []bool) (err lib.ErrorI) {
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
	// get the liquidity pool size
	lPool, err := s.GetPool(chainId + LiquidityPoolAddend)
	if err != nil {
		return
	}
	// set the pool size
	nextSellBatch.PoolSize = lPool.Amount
	// set the hash
	nextSellBatch.ReceiptHash = remoteBatch.Hash()
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
	var key []byte
	if locked {
		key = KeyForLockedBatch(chainId)
	} else {
		key = KeyForNextBatch(chainId)
	}
	// get bytes from state
	bz, err := s.Get(key)
	if err != nil {
		return
	}
	// create a new batch object reference to ensure no 'nil' batches are used
	b = &lib.DexBatch{Committee: chainId}
	defer b.EnsureNonNil()
	// check for nil bytes
	if len(bz) == 0 {
		return
	}
	// populate the batch object with the bytes
	err = lib.Unmarshal(bz, b)
	// check if points should be attached
	if len(withPoints) == 1 && withPoints[0] {
		var lPool *Pool
		// retrieve the liquidity pool from state
		lPool, err = s.GetPool(chainId + LiquidityPoolAddend)
		if err != nil {
			return
		}
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
