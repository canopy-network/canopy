package oracle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

// Terminology
// 1. *Observer Chain* - The Canopy nested chain recording the witnessed transactions
// 2. *Source Chain* - The source chain, such as Ethereum, where this Oracle witnesses transactions
// 2. *Witness Node* - An individual validator monitoring source chain transactions, running in the observer chain
// 4. *Transaction Oracle* - The overall system connecting Ethereum to Canopy

// Oracle is a chain-agnostic type implementing validation and storage logic for a cross-chain Oracle
// It coordinates between three components:
// - The source chain where transactions containing Canopy lock & close orders are witnessed
// - The witness nodes order store where Canopy lock & close orders are persisted
// - The witness nodes participation in the observer chain BFT process.
// - The root chain by submitting certificate results containing majority witnessed orders, and receiving order book updates which represent root chain order book activity

// The oracle integrates with the BFT consensus process through two key methods:
// - WitnessedOrders
// - ValidateProposedOrders
// It also receives root chain updates through:
// - UpdateRootChainInfo
type Oracle struct {
	// blockProvider is where the oracle will receive new blocks from
	blockProvider types.BlockProvider
	// the store with which the oracle can persist witnessed orders
	orderStore types.OrderStore
	// copy of the latest root chain order book
	orderBook *lib.OrderBook
	// mutex to protect order book
	orderBookMu sync.RWMutex
	// state handles block processing state, gap detection, and reorg detection
	state *OracleState
	// oracle configuration
	config lib.OracleConfig
	// committee to use when constructing close orders. this must match the order bookc committee
	committee uint64
	// safeHeight is the highest block height that has received sufficient confirmations to be considered safe
	safeHeight uint64
	// context to allow graceful shutdown
	ctx       context.Context
	ctxCancel context.CancelFunc
	// logger
	log lib.LoggerI
}

// NewOracle creates a new Oracle instance
func NewOracle(ctx context.Context, config lib.OracleConfig, blockProvider types.BlockProvider, transactionStore types.OrderStore, logger lib.LoggerI) (*Oracle, error) {
	// create context cancel function for the passed context
	ctx, cancel := context.WithCancel(ctx)
	// create new oracle instance
	o := &Oracle{
		blockProvider: blockProvider,
		orderStore:    transactionStore,
		log:           logger,
		state:         NewOracleState(config.StateFile, logger),
		config:        config,
		committee:     config.Committee,
		ctx:           ctx,
		ctxCancel:     cancel,
	}
	// return new oracle instance
	return o, nil
}

// reorgRollback gets the last known good height and removes orders from the store until
// the reorgRollbackDelta
func (o *Oracle) reorgRollback() {
	// get the last height processed by the oracle
	height := o.state.GetLastHeight()
	if height == 0 {
		o.log.Warnf("Reorg detected but no last known good height")
		return
	}
	// calculate the rollback height - orders witnessed above this height will be removed
	rollbackHeight := height - o.config.ReorgRollbackDelta
	o.log.Infof("Rolling back orders witnessed above height %d (last height %d - delta %d)", rollbackHeight, height, o.config.ReorgRollbackDelta)
	// process lock orders first
	o.rollbackOrderType(types.LockOrderType, rollbackHeight)
	// process close orders second
	o.rollbackOrderType(types.CloseOrderType, rollbackHeight)
	o.log.Infof("Reorg rollback completed")
}

// rollbackOrderType removes orders of the specified type that were witnessed above the rollback height
func (o *Oracle) rollbackOrderType(orderType types.OrderType, rollbackHeight uint64) {
	// get all order ids of the specified type from the order store
	orderIds, err := o.orderStore.GetAllOrderIds(orderType)
	if err != nil {
		o.log.Errorf("Error getting all %s order ids during rollback: %s", orderType, err.Error())
		return
	}
	// track rollback statistics
	totalOrders := len(orderIds)
	removedCount := 0
	// examine each stored order and remove if witnessed above rollback height
	for _, orderId := range orderIds {
		// read the witnessed order to check its witnessed height
		witnessedOrder, err := o.orderStore.ReadOrder(orderId, orderType)
		if err != nil {
			o.log.Errorf("Error reading %s order %x during rollback: %s", orderType, orderId, err.Error())
			continue
		}
		// check if this order was witnessed above the rollback height
		if witnessedOrder.WitnessedHeight > rollbackHeight {
			// remove the order from the store
			err = o.orderStore.RemoveOrder(orderId, orderType)
			if err != nil {
				o.log.Errorf("Error removing %s order %x during rollback: %s", orderType, orderId, err.Error())
				continue
			}
			removedCount++
			o.log.Debugf("Removed %s order %x witnessed at height %d", orderType, orderId, witnessedOrder.WitnessedHeight)
		}
	}
	o.log.Infof("Rollback processed %d %s orders, removed %d orders witnessed above height %d", totalOrders, orderType, removedCount, rollbackHeight)
}

// Start begins listening for blocks from the configured block provider
func (o *Oracle) Start(ctx context.Context) {
	// log that we're starting the oracle
	o.log.Info("Starting oracle")
	go func() {
		for {
			// an order book must be present to validate incoming orders
			// wait for the controller to set it
			for o.orderBook == nil {
				o.log.Warnf("Oracle waiting for order book")
				time.Sleep(1 * time.Second)
			}
			// listen for blocks
			err := o.run(ctx)
			if err == nil {
				// oracle context cancelled
				return
			}
			o.log.Errorf("Oracle stopped running: %s", err.Error())
			// handle specific error types
			switch err.Code() {
			case CodeBlockSequence:
				// log an error and allow provider to be restarted at last known good height
				o.log.Errorf("Block sequence gap detected, restarting block provider")
			case CodeChainReorg:
				o.reorgRollback()
				o.log.Errorf("Chain reorganization detected - oracle may need to rollback and reprocess from fork point")
			}
		}
	}()
}

// run runs the main Oracle loop
// - get last height from state manager
// - start block provider
func (o *Oracle) run(ctx context.Context) lib.ErrorI {
	// get the last height processed by the oracle
	if height := o.state.GetLastHeight(); height == 0 {
		// zero signals the block provider to determine its own starting height
		o.blockProvider.SetHeight(new(big.Int).SetUint64(0))
	} else { // height found
		// set the starting height for the block provider
		o.blockProvider.SetHeight(new(big.Int).SetUint64(height + 1))
	}
	// create a new context from the existing one
	blockProviderCtx, cancelBlockProvider := context.WithCancel(ctx)
	defer cancelBlockProvider()
	// start block provider with the new context
	o.blockProvider.Start(blockProviderCtx)
	// get the block channel from provider
	blockCh := o.blockProvider.BlockCh()
	// start the main oracle loop
	for {
		select {
		case block, ok := <-blockCh:
			if !ok {
				o.log.Warn("Block channel closed, stopping oracle")
				return ErrChannelClosed()
			}
			if block == nil {
				o.log.Warn("received nil block, skipping")
				continue
			}
			// check block for gaps and reorganizations
			if err := o.state.ValidateSequence(block); err != nil {
				o.log.Errorf("block validation failed for height %d: %v", block.Number(), err)
				return err
			}
			// process the received block
			err := o.processBlock(block)
			// check for processing error
			if err != nil {
				o.log.Errorf("Failed to process block at height %d: %v", block.Number(), err)
				continue
			}
			// update safe height after successful block processing
			o.updateSafeHeight(block.Number())
			// save state after successful block processing
			if err := o.state.SaveProcessedBlock(block); err != nil {
				o.log.Errorf("Failed to save block state for height %d: %v", block.Number(), err)
				// continue processing despite state save failure
			}
		case <-ctx.Done():
			// context cancelled, stop the goroutine
			o.log.Info("Oracle context cancelled, stopping block processing")
			return nil
		}
	}
}

// updateSafeHeight calculates and updates the safe block height with monotonic guarantee
// The safe height can only increase, never decrease, providing stability during reorgs
func (o *Oracle) updateSafeHeight(currentBlockHeight uint64) {
	// calculate new safe height by subtracting required confirmations
	var newSafeHeight uint64
	if currentBlockHeight > o.config.SafeBlockConfirmations {
		newSafeHeight = currentBlockHeight - o.config.SafeBlockConfirmations
	} else {
		// handle startup case where current height is less than required confirmations
		newSafeHeight = 0
	}
	// only update if the new safe height is higher (monotonic property)
	if newSafeHeight > o.safeHeight {
		o.log.Debugf("Updating safe height from %d to %d (current height %d, confirmations %d)",
			o.safeHeight, newSafeHeight, currentBlockHeight, o.config.SafeBlockConfirmations)
		o.safeHeight = newSafeHeight
	}
}

// Stop gracefully shuts down the oracle and all oracle components
func (o *Oracle) Stop() {
	if o == nil {
		return
	}
	o.log.Info("Stopping Oracle")
	// Cancel the context, stopping oracle and oracle components
	o.ctxCancel()
}

// validateOrder ensures the witnessed order passes basic sanity checks, then validates any lock or close orders with more specific functions
func (o *Oracle) validateOrder(tx types.TransactionI, sellOrder *lib.SellOrder) lib.ErrorI {
	// get witnessed order from transaction
	order := tx.Order()
	if order == nil {
		return ErrOrderValidation("witnessed order cannot be nil")
	}
	// convenience variables
	hasLock := order.LockOrder != nil
	hasClose := order.CloseOrder != nil
	// witnessed order must contain either a lock or close order
	if !hasLock && !hasClose {
		return ErrOrderValidation("witnessed order must contain either lock or close order")
	}
	// witnessed order cannot contain both a lock or close order
	if hasLock && hasClose {
		return ErrOrderValidation("witnessed order cannot contain both lock and close orders")
	}
	// validate the lock order
	if hasLock {
		return o.validateLockOrder(order.LockOrder, sellOrder)
	}
	// validate the close order
	return o.validateCloseOrder(order.CloseOrder, sellOrder, tx)
}

// validateLockOrder ensures a lock order matches a sell order
func (o *Oracle) validateLockOrder(lockOrder *lib.LockOrder, sellOrder *lib.SellOrder) lib.ErrorI {
	if !bytes.Equal(lockOrder.OrderId, sellOrder.Id) {
		return ErrOrderValidation("lock order ID does not match sell order ID")
	}
	if lockOrder.ChainId != sellOrder.Committee {
		return ErrOrderValidation("lock order chain ID does not match sell order committee")
	}
	return nil
}

// validateCloseOrder ensures a close order matches a sell order
// as each field is user-supplied arbitrary data coming from off chain, strict validation
// is required to protect against costly erroneous behavior or malicious activity
func (o *Oracle) validateCloseOrder(closeOrder *lib.CloseOrder, sellOrder *lib.SellOrder, tx types.TransactionI) lib.ErrorI {
	// Order data being equal to transaction To address is Ethereum-specific validation
	// TODO move this logic into the block provider
	sellOrderDataHex := fmt.Sprintf("%x", sellOrder.Data)
	txToAddress := strings.ToLower(strings.TrimPrefix(tx.To(), "0x"))
	if sellOrderDataHex != txToAddress {
		return ErrOrderValidation("sell order data field does not match transaction recipient")
	}
	// ensure the order ids are a match
	if !bytes.Equal(closeOrder.OrderId, sellOrder.Id) {
		return ErrOrderValidation("close order ID does not match sell order ID")
	}
	// ensure the chain and committee are a match
	if closeOrder.ChainId != sellOrder.Committee {
		return ErrOrderValidation("close order chain ID does not match sell order committee")
	}
	// convenience variable
	tokenTransfer := tx.TokenTransfer()
	recipient, err := lib.StringToBytes(tokenTransfer.RecipientAddress)
	if err != nil {
		return ErrOrderValidation("error converting recipient address to bytes")
	}
	// verify the recipient of the transfer was the seller receive address
	if !bytes.Equal(sellOrder.SellerReceiveAddress, recipient) {
		return ErrOrderValidation("tokens not transferred to sell receive address")
	}
	// ensure transfer amount is not nil
	// TODO validate further fields here?
	if tokenTransfer.TokenBaseAmount == nil {
		return ErrOrderValidation("token transfer amount cannot be nil")
	}
	// ensure the correct amount was transferred
	if tokenTransfer.TokenBaseAmount.Uint64() != sellOrder.RequestedAmount {
		return ErrOrderValidation(fmt.Sprintf("transfer amount %d does not match requested amount %d",
			tokenTransfer.TokenBaseAmount.Uint64(), sellOrder.RequestedAmount))
	}
	return nil
}

// processBlock processes a block received from the source chain
// examines any witnessed orders in the block, validates them, and writes them to the order store
// any orders that are not present in the order book, or fail validation, are dropped and not saved to the order store
func (o *Oracle) processBlock(block types.BlockI) lib.ErrorI {
	// lock order book for reading
	o.orderBookMu.RLock()
	defer o.orderBookMu.RUnlock()
	// log that we received a new block
	if len(block.Transactions()) > 0 {
		o.log.Infof("Received block %s at height %d (%d transactions)", block.Hash(), block.Number(), len(block.Transactions()))
	}
	// iterate through each transaction
	for _, tx := range block.Transactions() {
		// get order in this transaction
		order := tx.Order()
		if order == nil {
			// no order in this transaction
			continue
		}
		// find the order in the order book
		canopyOrder, orderErr := o.orderBook.GetOrder(order.OrderId)
		// check for order error - only error possible is nil order book
		if orderErr != nil {
			o.log.Errorf("Error getting order from order book: %s", orderErr.Error())
			return orderErr
		}
		// the order book returns a nil order if no order was found
		// this should not happen under normal circumstances but is not an error
		if canopyOrder == nil {
			// log a warning and continue processing transactions
			o.log.Warnf("Order %s not found in order book", lib.BytesToString(order.OrderId))
			continue
		}
		// validate the order that was witnessed against the order found in the order book
		if err := o.validateOrder(tx, canopyOrder); err != nil {
			// log a warning and continue processing transactions
			o.log.Warnf(err.Error())
			continue
		}
		// determine order type
		orderType := types.LockOrderType
		if order.CloseOrder != nil {
			orderType = types.CloseOrderType
		}
		// check if the witnessed order already exists in store
		_, err := o.orderStore.ReadOrder(order.OrderId, orderType)
		if err == nil {
			o.log.Warnf("Order %s already exists in store, skipping new order", lib.BytesToString(order.OrderId))
			// order exists, skip writing
			// this prevents newer orders from overwriting older orders
			// TODO should there be any more logic here?
			continue
		}
		// write order to disk.
		err = o.orderStore.WriteOrder(order, orderType)
		if err != nil {
			o.log.Errorf("Failed to write order %s: %v", lib.BytesToString(order.OrderId), err)
			return err
		}
		// write order to archive
		err = o.orderStore.ArchiveOrder(order, orderType)
		if err != nil {
			o.log.Errorf("Failed to archive order %s: %v", lib.BytesToString(order.OrderId), err)
			return err
		}
		o.log.Debugf("Wrote order %s %s to store", order, orderType)
	}
	return nil
}

// ValidateProposedOrders verifies that the passed orders are all present in the local order store.
// This is called when the BFT module validates a block proposal to ensure that each order
// in the proposed block is an exact match for an order in the witnessed order store.
func (o *Oracle) ValidateProposedOrders(orders *lib.Orders) lib.ErrorI {
	// oracle is disabled
	if o == nil {
		return nil
	}
	// handle nil orders case
	if orders == nil {
		o.log.Debug("orders == nil, no orders to validate")
		return nil
	}
	// skip validation when no orders are present
	if len(orders.LockOrders) == 0 && len(orders.CloseOrders) == 0 {
		o.log.Debug("No orders to validate")
		return nil
	}
	// validate each lock order against the witnessed order store
	for _, lock := range orders.LockOrders {
		o.log.Infof("Validating proposed lock order %s", lib.BytesToString(lock.OrderId))
		// get order from order store
		witnessedOrder, err := o.orderStore.ReadOrder(lock.OrderId, types.LockOrderType)
		if err != nil {
			o.log.Warnf("Proposed lock order %s not validated", lib.BytesToString(lock.OrderId))
			return ErrOrderNotVerified(lib.BytesToString(lock.OrderId), err)
		}
		// check if the witnessed order is from a safe block (has sufficient confirmations)
		if witnessedOrder.WitnessedHeight > o.safeHeight {
			o.log.Warnf("Proposed lock order %s not validated: witnessed at height %d, safe height is %d", lib.BytesToString(lock.OrderId), witnessedOrder.WitnessedHeight, o.safeHeight)
			return ErrOrderNotVerified(lib.BytesToString(lock.OrderId), errors.New("order witnessed above safe height"))
		}
		// compare orderbook order and witnessed order
		if !lock.Equals(witnessedOrder.LockOrder) {
			o.log.Warnf("Proposed lock order %s not validated", lib.BytesToString(lock.OrderId))
			return ErrOrderNotVerified(lib.BytesToString(lock.OrderId), errors.New("lock order unequal"))
		}
		o.log.Infof("Validated proposed lock order %s successfully", lib.BytesToString(lock.OrderId))
	}
	// validate each close order against the witnessed order store
	for _, orderId := range orders.CloseOrders {
		o.log.Infof("Validating proposed close order %s", lib.BytesToString(orderId))
		// get the witnessd order
		witnessedOrder, err := o.orderStore.ReadOrder(orderId, types.CloseOrderType)
		if err != nil {
			o.log.Warnf("Proposed close order %s not validated", lib.BytesToString(orderId))
			return ErrOrderNotVerified(lib.BytesToString(orderId), err)
		}
		// check if the witnessed order is from a safe block (has sufficient confirmations)
		if witnessedOrder.WitnessedHeight > o.safeHeight {
			o.log.Warnf("Proposed close order %s not validated: witnessed at height %d, safe height is %d", lib.BytesToString(orderId), witnessedOrder.WitnessedHeight, o.safeHeight)
			return ErrOrderNotVerified(lib.BytesToString(orderId), errors.New("order witnessed above safe height"))
		}
		// construct close order for comparison
		order := lib.CloseOrder{
			OrderId:    orderId,
			ChainId:    o.committee,
			CloseOrder: true,
		}
		// compare orderbook order and witnessed order
		if !order.Equals(witnessedOrder.CloseOrder) {
			o.log.Warnf("Proposed close order %s not validated", lib.BytesToString(orderId))
			return ErrOrderNotVerified(lib.BytesToString(orderId), errors.New("close order unequal"))
		}
		o.log.Infof("Validated proposed close order %s successfully", lib.BytesToString(order.OrderId))
	}
	if len(orders.LockOrders) == 0 && len(orders.CloseOrders) == 0 {
		o.log.Debug("Validated off chain orders successfully")
	}
	return nil
}

// UpdateOrderBook receives a new order book and stores it for order validation
func (o *Oracle) UpdateOrderBook(orderBook *lib.OrderBook) {
	o.orderBookMu.Lock()
	defer o.orderBookMu.Unlock()
	o.orderBook = orderBook
	o.log.Debugf("Orderbook updated, %d orders", len(orderBook.Orders))
}

// CommitCertificate is executed after the quorum agrees on a block
func (o *Oracle) CommitCertificate(qc *lib.QuorumCertificate, block *lib.Block, blockResult *lib.BlockResult, ts uint64) (err lib.ErrorI) {

	// Update the last submit height for all lock orders in this certificate
	for _, order := range qc.Results.Orders.LockOrders {
		// get order from order store
		wOrder, err := o.orderStore.ReadOrder(order.OrderId, types.LockOrderType)
		if err != nil {
			o.log.Warnf("CommitCertificate unable to find order %s in order store", lib.BytesToString(order.OrderId))
			return ErrOrderNotVerified(lib.BytesToString(order.OrderId), err)
		}
		// update the last height this order was submitted
		// TODO is this the proper way to get the root height?
		wOrder.LastSubmitHeight = qc.Header.RootHeight
		// save this update to disk
		err = o.orderStore.WriteOrder(wOrder, types.LockOrderType)
		if err != nil {
			o.log.Errorf("CommitCertificate failed to write order %s: %v", lib.BytesToString(order.OrderId), err)
			continue
		}
		o.log.Infof("CommitCertificate updated last submit height for lock order %s: %d", lib.BytesToString(order.OrderId), qc.Header.RootHeight)
	}
	// Update the last submit height for all close orders in this certificate
	for _, orderId := range qc.Results.Orders.CloseOrders {
		// get order from order store
		wOrder, err := o.orderStore.ReadOrder(orderId, types.CloseOrderType)
		if err != nil {
			o.log.Warnf("CommitCertificate unable to find order %s in order store", lib.BytesToString(orderId))
			return ErrOrderNotVerified(lib.BytesToString(orderId), err)
		}
		// update the last height this order was submitted
		// TODO is this the proper way to get the root height?
		wOrder.LastSubmitHeight = qc.Header.RootHeight
		// save this update to disk
		err = o.orderStore.WriteOrder(wOrder, types.CloseOrderType)
		if err != nil {
			o.log.Errorf("CommitCertificate failed to write close order %s: %v", lib.BytesToString(orderId), err)
			continue
		}
		o.log.Infof("CommitCertificate updated last submit height for close order %s: %d", lib.BytesToString(orderId), qc.Header.RootHeight)
	}

	return
}

// UpdateRootChainInfo examines the new root chain order book and updates the local order store.
// The method performs the following operations:
//   - saves the order book for later use
//   - removes lock orders from the store when corresponding sell orders are locked on the root chain
//   - removes lock/close orders when their corresponding sell orders are no longer present
func (o *Oracle) UpdateRootChainInfo(info *lib.RootChainInfo) {
	// oracle is disabled
	if o == nil {
		return
	}
	// lock order book while updating it and updating order store
	o.orderBookMu.Lock()
	defer o.orderBookMu.Unlock()

	if o.orderBook == nil {
		o.log.Warn("Order book is nil, skipping order store cleanup")
		return
	}
	// log a warning for a nil order book
	if info.Orders == nil {
		o.log.Warn("OrderBook from root chain was nil")
	}
	// get all lock orders from the order store
	storedOrders, err := o.orderStore.GetAllOrderIds(types.LockOrderType)
	if err != nil {
		o.log.Errorf("Error getting all order ids: %s", err.Error())
		return
	}
	// examine stored lock orders and remove any not present in the order book
	for _, id := range storedOrders {
		// o.log.Debugf("UpdateRootChainInfo checking stored lock order %x for removal", id)
		// attempt to get stored lock order from order book
		order, err := o.orderBook.GetOrder(id)
		if err != nil {
			o.log.Errorf("Error getting order from order book: %s", err.Error())
			continue
		}
		// remove lock order from store if one of the following conditions is met:
		//   - corresponding sell order was not found in the root chain order book
		//   - root chain sell order is locked
		switch {
		case order == nil:
			o.log.Infof("Order %x no longer in order book, removing lock order from store", id)
		case order.BuyerSendAddress != nil:
			o.log.Infof("Order %x is locked in order book, removing lock order from store", order.Id)
		default:
			// neither condition was met, do not remove this order
			// continue processing remaining stored orders
			continue
		}
		// remove lock order from the store
		err = o.orderStore.RemoveOrder(id, types.LockOrderType)
		if err != nil {
			o.log.Errorf("Error removing order from order store: %s", err.Error())
		}
	}
	// get all close orders from the order store
	storedOrders, err = o.orderStore.GetAllOrderIds(types.CloseOrderType)
	if err != nil {
		o.log.Errorf("Error getting all order ids: %s", err.Error())
		return
	}
	// examine every stored close order and remove it if is no long present in the order book
	for _, id := range storedOrders {
		// o.log.Debugf("UpdateRootChainInfo checking stored close order %x for removal", id)
		// attempt to get stored close order from order book
		order, err := o.orderBook.GetOrder(id)
		if err != nil {
			o.log.Errorf("Error getting order from order book: %s", err.Error())
			continue
		}
		// remove close order from store if it was not found in the order book
		if order == nil {
			o.log.Infof("Removing close order %x from store", id)
			err := o.orderStore.RemoveOrder(id, types.CloseOrderType)
			if err != nil {
				o.log.Errorf("Error removing order from order store: %s", err.Error())
			}
		}
	}
}

// WitnessedOrders returns witnessed orders that match orders in the order book
// When the block proposer produces a block proposal it uses the orders returned here to build the proposed block
// TODO watch for conflicts while syncing ethereum block, prooducer might resubmit order
func (o *Oracle) WitnessedOrders(orderBook *lib.OrderBook, rootHeight uint64) ([]*lib.LockOrder, [][]byte) {
	lockOrders := []*lib.LockOrder{}
	closeOrders := [][]byte{}
	// oracle is disabled
	if o == nil {
		return lockOrders, closeOrders
	}
	// loop through the order book searching the order store for lock/close orders witnessed by this node
	for _, order := range orderBook.Orders {
		// buyer receive address being nil indicates this is an unlocked sell order
		if order.BuyerReceiveAddress == nil {
			// process unlocked sell orders - look for witnessed lock orders
			wOrder, err := o.orderStore.ReadOrder(order.Id, types.LockOrderType)
			if err != nil {
				if err.Code() != CodeReadOrder {
					o.log.Errorf("Failed to read order %s: %v", order, err)
				}
				continue
			}
			// check if the witnessed order is from a safe block (has sufficient confirmations)
			if wOrder.WitnessedHeight > o.safeHeight {
				o.log.Debugf("Not submitting lock order %s: witnessed at height %d, safe height is %d", lib.BytesToString(order.Id), wOrder.WitnessedHeight, o.safeHeight)
				continue
			}
			// check whether this witnessed lock order should be submitted in the next proposed block
			if !o.state.shouldSubmit(wOrder, rootHeight, o.config) {
				o.log.Debugf("Not submitting lock order %s: LastSubmightHeight %d rootHeight %d", lib.BytesToString(order.Id), wOrder.LastSubmitHeight, rootHeight)
				continue
			}
			o.log.Debugf("Informing controller of witnessed lock order %s", wOrder)
			// submit this witnessed lock order by returning it in the lockOrders slice
			lockOrders = append(lockOrders, wOrder.LockOrder)
		} else {
			// process locked orders - look for witnessed close orders
			wOrder, err := o.orderStore.ReadOrder(order.Id, types.CloseOrderType)
			if err != nil {
				if err.Code() != CodeReadOrder {
					o.log.Errorf("Failed to read order %s: %v", lib.BytesToString(order.Id), err)
				}
				// No witnessed order is a normal condition, do not log
				continue
			}
			// check if the witnessed order is from a safe block (has sufficient confirmations)
			if wOrder.WitnessedHeight > o.safeHeight {
				o.log.Debugf("Not submitting close order %s: witnessed at height %d, safe height is %d", lib.BytesToString(order.Id), wOrder.WitnessedHeight, o.safeHeight)
				continue
			}
			// check whether this witnessed close order should be submitted in the next proposed block
			if !o.state.shouldSubmit(wOrder, rootHeight, o.config) {
				o.log.Debugf("Not submitting close order %s: LastSubmightHeight %d rootHeight %d", lib.BytesToString(order.Id), wOrder.LastSubmitHeight, rootHeight)
				continue
			}
			// update the last height this order was submitted
			wOrder.LastSubmitHeight = rootHeight
			// update the witnessed order in the store
			err = o.orderStore.WriteOrder(wOrder, types.CloseOrderType)
			if err != nil {
				o.log.Errorf("Failed to write order %s: %v", lib.BytesToString(order.Id), err)
				continue
			}
			o.log.Debugf("Informing controller of witnessed close order %s", wOrder)
			// submit this witnessed close order by returning it in the closeOrders slice
			closeOrders = append(closeOrders, wOrder.OrderId)
		}
	}
	// id, _ := lib.StringToBytes("005361a47f682d6e3af948c1f520c48e2e1701f0")
	// send, _ := lib.StringToBytes("f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	// recv, _ := lib.StringToBytes("45281f3e49287fb12a6721bffab01fb60ee02df9")

	// lockOrders = append(lockOrders, &lib.LockOrder{
	// 	OrderId:             id,
	// 	ChainId:             2,
	// 	BuyerSendAddress:    send,
	// 	BuyerReceiveAddress: recv,
	// 	BuyerChainDeadline:  100000,
	// })
	// if len(lockOrders) > 0 || len(closeOrders) > 0 {
	o.log.Infof("Witnessed %d lock orders and %d close orders, root height %d", len(lockOrders), len(closeOrders), rootHeight)
	// }
	return lockOrders, closeOrders
}
