package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"sort"
)

/* Dex.go implements logic to handle AMM style atomic exchanges between root & nested chains
   not to be confused with 1 way order book swaps implemented in swap.go */

// HandleDexBatch() handles dex batch from the counterparty (buying logic)
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
	// handle the 'buying side'
	return s.HandleDexBuyBatch(chainId, buyBatch)
}

// HandleDexBuyBatch() handles a 'buy' batch: (1) dex receipts (2) 'buy' batch
func (s *StateMachine) HandleDexBuyBatch(chainId uint64, buyBatch *lib.DexBatch) (err lib.ErrorI) {
	// exit if dex data is empty
	if buyBatch == nil {
		return
	}
	// handle the batch receipt - this function manages atomicity of the dex operations
	locked, err := s.HandleDexBatchReceipt(chainId, buyBatch.Receipts)
	if err != nil {
		return
	}
	// do no action further until previous batch is handled
	if locked {
		return
	}
	// process batch - save receipts to state machine
	receipts, err := s.FindUCP(buyBatch, buyBatch.PoolSize)
	if err != nil {
		return
	}
	// set the 'next batch' as 'locked batch' in state with receipts and pool size
	return s.RotateDexSellBatch(receipts, chainId)
}

// HandleDexBatchReceipt() handles a 'receipt' for a 'sell batch'
// Sends funds from holding pool to either the liquidity pool or back to sender
func (s *StateMachine) HandleDexBatchReceipt(chainId uint64, receipts []bool) (locked bool, err lib.ErrorI) {
	// get locked sell batch
	lockedBatch, err := s.GetDexBatch(KeyForLockedBatch(chainId))
	if err != nil {
		return true, err
	}
	// check if nil or empty
	if receipts == nil || lockedBatch.IsEmpty() {
		return false, nil
	}
	// ensure receipt not mismatch
	if len(lockedBatch.Orders) != len(receipts) {
		return true, ErrMismatchDexBatchReceipt()
	}
	// for each order, move the funds in the holding pool depending on the success or failure
	for i, order := range lockedBatch.Orders {
		// remove funds from the holding pool
		if err = s.PoolSub(s.Config.ChainId+HoldingPoolAddend, order.AmountForSale); err != nil {
			return true, err
		}
		// if order succeeded, add funds to the liquidity pool, else revert back to sender
		if receipts[i] {
			err = s.PoolAdd(s.Config.ChainId+LiquidityPoolAddend, order.AmountForSale)
		} else {
			err = s.AccountAdd(crypto.NewAddress(order.Address), order.AmountForSale)
		}
		if err != nil {
			return true, err
		}
	}
	// remove lockedBatch to lift the 'atomic lock' - enabling orders to be sent in the next transaction
	return false, s.Delete(KeyForLockedBatch(chainId))
}

// FindUCP() executes AMM logic over a 'batch' of limit orders
// (1) calculates the uniform clearing price for a batch
// (2) determines successful orders & distributes from the liquidity pool
// (3) returns the receipts
func (s *StateMachine) FindUCP(b *lib.DexBatch, addPoolAmount uint64) (success []bool, err lib.ErrorI) {
	// initialize the receipt
	success = make([]bool, len(b.Orders))
	// initialize a map to track accepted
	accepted := make(map[uint64]bool)
	// get the buy pool size (pool where distributions are made from)
	distributePool, err := s.GetPool(s.Config.ChainId + LiquidityPoolAddend)
	if err != nil {
		return nil, err
	}
	// set pools
	x, y := float64(addPoolAmount), float64(distributePool.Amount)
	// calculate k
	k := x * y
	// handle bad k
	if k == 0 {
		return nil, ErrInvalidLiquidityPool()
	}
	// make a copy of the orders
	orders := b.Copy().Orders
	// sort descending by P, then Addr
	sort.SliceStable(orders, func(i, j int) bool {
		if orders[i].MinAsk() != orders[j].MinAsk() {
			return orders[i].MinAsk() > orders[j].MinAsk()
		}
		return bytes.Compare(orders[i].Address, orders[j].Address) == -1
	})
	// calculate if orders are accepted
	var deltaY, deltaX float64
	for i, order := range orders {
		dx := deltaX + float64(orders[i].AmountForSale)
		if dx <= 0 {
			continue
		}
		// recalculate to enforce a proportional, weighted influence on the price
		// ΔY = y - k/(x + ΔX)
		dy := y - (k / (x + dx))
		price := dy / dx

		if price < order.MinAsk() {
			break // can't include this or any later order
		}
		// set order as accepted
		accepted[order.MapKey()] = true
		// set variables
		deltaX, deltaY = dx, dy
	}
	// calculate distributable amount after .3% fee
	distributableDeltaY := deltaY * 0.997
	// set success in the receipt
	for i, order := range b.Orders {
		if success[i] = accepted[order.MapKey()]; success[i] {
			// allocate pro-rata
			share := float64(order.AmountForSale) / deltaX
			// distribute from liquidity pool to address
			distributeAmount := uint64(distributableDeltaY * share)
			// distribute from pool
			if err = s.PoolSub(s.Config.ChainId+LiquidityPoolAddend, distributeAmount); err != nil {
				return nil, err
			}
			// add to account
			if err = s.AccountAdd(crypto.NewAddress(order.Address), distributeAmount); err != nil {
				return nil, err
			}
		}
	}
	return
}

// RotateDexSellBatch() sets 'next batch' as 'locked batch' and deletes reference for 'next batch'
// (1) checks if locked batch is processed yet - if not exit
// (2) sets the upcoming 'sell' batch as 'last' sell batch
// (3) returns the upcoming 'sell' batch to be sent to the root
func (s *StateMachine) RotateDexSellBatch(receipts []bool, chainId uint64) (err lib.ErrorI) {
	// get locked sell batch
	lockedBatch, err := s.GetDexBatch(KeyForLockedBatch(chainId))
	// exit with error or nil if last sell batch not yet processed by root (atomic protection)
	if err != nil || !lockedBatch.IsEmpty() {
		return
	}
	// get upcoming sell batch
	nextSellBatch, err := s.GetDexBatch(KeyForNextBatch(chainId))
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
	// set receipts
	nextSellBatch.Receipts = receipts
	// delete 'next sell batch'
	if err = s.Delete(KeyForNextBatch(chainId)); err != nil {
		return
	}
	// set the upcoming sell batch as 'last'
	err = s.SetDexBatch(KeyForLockedBatch(chainId), nextSellBatch)
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
	b = &lib.DexBatch{
		Committee: s.Config.ChainId,
		Orders:    make([]*lib.DexLimitOrder, 0),
	}
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
