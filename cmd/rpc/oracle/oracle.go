package oracle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/common"
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
	// context to allow graceful shutdown
	ctx       context.Context
	ctxCancel context.CancelFunc
	// logger
	log lib.LoggerI
	// metrics for telemetry
	metrics *lib.Metrics
}

// NewOracle creates a new Oracle instance
func NewOracle(ctx context.Context, config lib.OracleConfig, blockProvider types.BlockProvider, transactionStore types.OrderStore, logger lib.LoggerI, metrics *lib.Metrics) (*Oracle, error) {
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
		metrics:       metrics,
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
		o.log.Warnf("[ORACLE-REORG] Reorg detected but no last known good height")
		return
	}
	// calculate the rollback height - orders witnessed above this height will be removed
	rollbackHeight := height - o.config.ReorgRollbackBlocks
	o.log.Infof("[ORACLE-REORG] Rolling back orders witnessed above height %d (last height %d - delta %d)", rollbackHeight, height, o.config.ReorgRollbackBlocks)
	// process lock orders first
	o.rollbackOrderType(types.LockOrderType, rollbackHeight)
	// process close orders second
	o.rollbackOrderType(types.CloseOrderType, rollbackHeight)
	// update reorg metrics
	o.metrics.UpdateOracleErrorMetrics(1, 0, 0)
	o.metrics.RecordOracleReorgDepth(o.config.ReorgRollbackBlocks)
	o.log.Infof("[ORACLE-REORG] Reorg rollback completed")
}

// rollbackOrderType removes orders of the specified type that were witnessed above the rollback height
func (o *Oracle) rollbackOrderType(orderType types.OrderType, rollbackHeight uint64) {
	// get all order ids of the specified type from the order store
	orderIds, err := o.orderStore.GetAllOrderIds(orderType)
	if err != nil {
		o.log.Errorf("[ORACLE-REORG] Error getting all %s order ids during rollback: %s", orderType, err.Error())
		return
	}
	// track rollback statistics
	totalOrders := len(orderIds)
	removedCount := 0
	// examine each stored order and remove if witnessed above rollback height
	var storeReadErrors, storeRemoveErrors int
	for _, orderId := range orderIds {
		// read the witnessed order to check its witnessed height
		witnessedOrder, err := o.orderStore.ReadOrder(orderId, orderType)
		if err != nil {
			o.log.Errorf("[ORACLE-REORG] Error reading %s order %x during rollback: %s", orderType, orderId, err.Error())
			storeReadErrors++
			continue
		}
		// check if this order was witnessed above the rollback height
		if witnessedOrder.WitnessedHeight > rollbackHeight {
			// remove the order from the store
			err = o.orderStore.RemoveOrder(orderId, orderType)
			if err != nil {
				o.log.Errorf("[ORACLE-REORG] Error removing %s order %x during rollback: %s", orderType, orderId, err.Error())
				storeRemoveErrors++
				continue
			}
			removedCount++
			o.log.Debugf("[ORACLE-REORG] Removed %s order %x witnessed at height %d", orderType, orderId, witnessedOrder.WitnessedHeight)
		}
	}
	o.log.Infof("[ORACLE-REORG] Rollback processed %d %s orders, removed %d orders witnessed above height %d", totalOrders, orderType, removedCount, rollbackHeight)
	// update pruned orders metric
	if removedCount > 0 {
		o.metrics.UpdateOracleErrorMetrics(0, removedCount, 0)
	}
	// update store error metrics
	if storeReadErrors > 0 || storeRemoveErrors > 0 {
		o.metrics.UpdateOracleStoreErrorMetrics(0, storeReadErrors, storeRemoveErrors)
	}
}

// Start begins listening for blocks from the configured block provider
// syncCh: optional channel to notify when block provider syncs to top (can be nil)
func (o *Oracle) Start(ctx context.Context, syncCh chan<- bool) {
	// log that we're starting the oracle
	o.log.Info("[ORACLE-LIFECYCLE] Starting oracle")
	go func() {
		firstRun := true
		for {
			// an order book must be present to validate incoming orders
			// wait for the controller to set it
			for o.orderBook == nil {
				o.log.Warnf("[ORACLE-LIFECYCLE] Oracle waiting for order book")
				time.Sleep(1 * time.Second)
			}
			// listen for blocks
			// only pass syncCh on the first run to avoid closing it multiple times
			var ch chan<- bool
			if firstRun {
				ch = syncCh
				firstRun = false
			}
			err := o.run(ctx, ch)
			if err == nil {
				// oracle context cancelled
				return
			}
			o.log.Errorf("[ORACLE-LIFECYCLE] Oracle stopped running: %s", err.Error())
			// handle specific error types
			switch err.Code() {
			case CodeBlockSequence:
				// remove current state
				o.state.removeState()
				o.log.Errorf("[ORACLE-LIFECYCLE] Block sequence gap detected, restarting block provider")
			case CodeChainReorg:
				// execute a rollback
				o.reorgRollback()
				// remove current state so oracle restarts from fresh
				o.state.removeState()
				o.log.Errorf("[ORACLE-LIFECYCLE] Chain reorganization detected - oracle rolled back and will reprocess from fresh state")
			default:
				o.log.Errorf("[ORACLE-LIFECYCLE] Oracle unexpected error: %v", err)
			}
		}
	}()
}

// run runs the main Oracle loop
// - get last height from state manager
// - start block provider
// syncCh: optional channel to notify when block provider syncs to top (can be nil)
func (o *Oracle) run(ctx context.Context, syncCh chan<- bool) lib.ErrorI {
	// create a new context from the existing one
	blockProviderCtx, cancelBlockProvider := context.WithCancel(ctx)
	defer cancelBlockProvider()
	// get the last height processed by the oracle
	if height := o.state.GetLastHeight(); height == 0 { // no height found
		// zero signals the block provider to determine its own starting height
		o.blockProvider.Start(blockProviderCtx, height)
	} else { // height found
		// set the starting height for the block provider
		o.blockProvider.Start(blockProviderCtx, height+1)
	}
	// get the block channel from provider
	blockCh := o.blockProvider.BlockCh()
	// track sync status to avoid duplicate notifications
	lastSyncStatus := false
	// start the main oracle loop
	for {
		select {
		case block, ok := <-blockCh:
			if !ok {
				o.log.Warn("[ORACLE-LIFECYCLE] Block channel closed, stopping oracle")
				return ErrChannelClosed()
			}
			if block == nil {
				o.log.Warn("[ORACLE-BLOCK] received nil block, skipping")
				continue
			}
			// check block for gaps and reorganizations
			if err := o.state.ValidateSequence(block); err != nil {
				o.log.Errorf("[ORACLE-BLOCK] block validation failed for height %d: %v", block.Number(), err)
				return err
			}
			// process the received block
			err := o.processBlock(block)
			// check for processing error
			if err != nil {
				o.log.Errorf("[ORACLE-BLOCK] Failed to process block at height %d: %v", block.Number(), err)
				o.metrics.UpdateOracleErrorMetrics(0, 0, 1)
				continue
			}
			// update safe height after successful block processing
			o.state.updateSafeHeight(block.Number(), o.config)
			// save state after successful block processing
			if err := o.state.saveState(block); err != nil {
				o.log.Errorf("[ORACLE-BLOCK] Failed to save block state for height %d: %v", block.Number(), err)
				o.metrics.UpdateOracleErrorMetrics(0, 0, 1)
				return err
			}
			// close syncCh when provider is synced to top
			if syncCh != nil && !lastSyncStatus {
				if o.blockProvider.IsSynced() {
					lastSyncStatus = true
					close(syncCh)
				}
			}
		case <-ctx.Done():
			// context cancelled, stop the goroutine
			o.log.Info("[ORACLE-LIFECYCLE] Oracle context cancelled, stopping block processing")
			// notify that oracle is no longer synced when shutting down
			if syncCh != nil {
				select {
				case syncCh <- false:
					o.log.Info("[ORACLE-LIFECYCLE] Oracle sync status set to false on shutdown")
				default:
					// channel full or closed, continue shutdown
				}
			}
			return nil
		}
	}
}

// Stop gracefully shuts down the oracle and all oracle components
func (o *Oracle) Stop() {
	if o == nil {
		return
	}
	o.log.Info("[ORACLE-LIFECYCLE] Stopping Oracle")
	// Cancel the context, stopping oracle and oracle components
	o.ctxCancel()
}

// validateOrder ensures the witnessed order passes basic sanity checks, then validates any lock or close orders with more specific functions
func (o *Oracle) validateOrder(tx types.TransactionI, sellOrder *lib.SellOrder) lib.ErrorI {
	// get witnessed order from transaction
	order := tx.Order()
	if order == nil {
		o.metrics.IncrementValidationFailure("order_nil")
		return ErrOrderValidation("witnessed order cannot be nil")
	}
	// convenience variables
	hasLock := order.LockOrder != nil
	hasClose := order.CloseOrder != nil
	// witnessed order must contain either a lock or close order
	if !hasLock && !hasClose {
		o.metrics.IncrementValidationFailure("missing_order_type")
		return ErrOrderValidation("witnessed order must contain either lock or close order")
	}
	// witnessed order cannot contain both a lock or close order
	if hasLock && hasClose {
		o.metrics.IncrementValidationFailure("both_order_types")
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
		o.metrics.IncrementValidationFailure("lock_id_mismatch")
		return ErrOrderValidation("lock order ID does not match sell order ID")
	}
	if lockOrder.ChainId != sellOrder.Committee {
		o.metrics.IncrementValidationFailure("lock_chain_mismatch")
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

	sellOrderDataHex := common.BytesToAddress(sellOrder.Data).String()
	if sellOrderDataHex != tx.To() {
		fmt.Println(sellOrderDataHex, tx.To())
		o.metrics.IncrementValidationFailure("close_data_mismatch")
		return ErrOrderValidation("sell order data field does not match transaction recipient")
	}
	// ensure the order ids are a match
	if !bytes.Equal(closeOrder.OrderId, sellOrder.Id) {
		o.metrics.IncrementValidationFailure("close_id_mismatch")
		return ErrOrderValidation("close order ID does not match sell order ID")
	}
	// ensure the chain and committee are a match
	if closeOrder.ChainId != sellOrder.Committee {
		o.metrics.IncrementValidationFailure("close_chain_mismatch")
		return ErrOrderValidation("close order chain ID does not match sell order committee")
	}
	// convenience variable
	tokenTransfer := tx.TokenTransfer()
	recipient, err := lib.StringToBytes(strings.TrimPrefix(tokenTransfer.RecipientAddress, "0x"))
	if err != nil {
		o.metrics.IncrementValidationFailure("recipient_conversion_error")
		return ErrOrderValidation("error converting recipient address to bytes")
	}
	// verify the recipient of the transfer was the seller receive address
	if !bytes.Equal(sellOrder.SellerReceiveAddress, recipient) {
		o.metrics.IncrementValidationFailure("recipient_mismatch")
		return ErrOrderValidation("tokens not transferred to sell receive address")
	}
	// ensure transfer amount is not nil
	// TODO validate further fields here?
	if tokenTransfer.TokenBaseAmount == nil {
		o.metrics.IncrementValidationFailure("amount_nil")
		return ErrOrderValidation("token transfer amount cannot be nil")
	}
	// ensure the correct amount was transferred
	if tokenTransfer.TokenBaseAmount.Uint64() != sellOrder.RequestedAmount {
		o.metrics.IncrementValidationFailure("amount_mismatch")
		return ErrOrderValidation(fmt.Sprintf("transfer amount %d does not match requested amount %d",
			tokenTransfer.TokenBaseAmount.Uint64(), sellOrder.RequestedAmount))
	}
	return nil
}

// processBlock processes a block received from the source chain
// examines any witnessed orders in the block, validates them, and writes them to the order store
// any orders that are not present in the order book, or fail validation, are dropped and not saved to the order store
func (o *Oracle) processBlock(block types.BlockI) lib.ErrorI {
	// track block processing start time for metrics
	startTime := time.Now()
	defer func() {
		// update block processing metrics
		o.metrics.UpdateOracleBlockMetrics(time.Since(startTime))
	}()
	// lock order book for reading
	o.orderBookMu.RLock()
	defer o.orderBookMu.RUnlock()
	// log that we received a new block
	if len(block.Transactions()) > 0 {
		o.log.Infof("[ORACLE-BLOCK] Received block %s at height %d (%d transactions)", block.Hash(), block.Number(), len(block.Transactions()))
	}
	// initialize metrics counters for this block
	var witnessed, validated, rejected int
	var notInOrderbook, duplicate, archived int
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
			o.log.Errorf("[ORACLE-ORDER] Error getting order from order book: %s", orderErr.Error())
			return orderErr
		}
		// the order book returns a nil order if no order was found
		// this should not happen under normal circumstances but is not an error
		if canopyOrder == nil {
			// log a warning and continue processing transactions
			o.log.Warnf("[ORACLE-ORDER] Order %s not found in order book", lib.BytesToString(order.OrderId))
			rejected++
			notInOrderbook++
			continue
		}
		// increment witnessed orders counter
		witnessed++
		// validate the order that was witnessed against the order found in the order book
		validationStart := time.Now()
		if err := o.validateOrder(tx, canopyOrder); err != nil {
			// log a warning and continue processing transactions
			o.log.Warnf("[ORACLE-ORDER] %s", err.Error())
			rejected++
			continue
		}
		// order validation succeeded
		validated++
		validationTime := time.Since(validationStart)
		// determine order type
		orderType := types.LockOrderType
		if order.CloseOrder != nil {
			orderType = types.CloseOrderType
		}
		// check if the witnessed order already exists in store
		_, err := o.orderStore.ReadOrder(order.OrderId, orderType)
		if err == nil {
			o.log.Warnf("[ORACLE-ORDER] Order %s already exists in store, skipping new order", lib.BytesToString(order.OrderId))
			// order exists, skip writing
			// this prevents newer orders from overwriting older orders
			duplicate++
			continue
		}
		err = o.orderStore.WriteOrder(order, orderType)
		if err != nil {
			o.log.Errorf("[ORACLE-ORDER] Failed to write order %s: %v", lib.BytesToString(order.OrderId), err)
			o.metrics.UpdateOracleStoreErrorMetrics(1, 0, 0)
			return err
		}
		// write order to archive
		err = o.orderStore.ArchiveOrder(order, orderType)
		if err != nil {
			o.log.Errorf("[ORACLE-ORDER] Failed to archive order %s: %v", lib.BytesToString(order.OrderId), err)
			o.metrics.UpdateOracleStoreErrorMetrics(1, 0, 0)
			return err
		}
		archived++
		// update order metrics for this successful write
		o.metrics.UpdateOracleOrderMetrics(0, 0, 0, 0, validationTime)
		o.log.Debugf("[ORACLE-ORDER] Wrote order %s %s to store", order, orderType)
	}
	// update order metrics for this block with counters
	o.metrics.UpdateOracleOrderMetrics(witnessed, validated, 0, rejected, 0)
	// update lifecycle metrics
	o.metrics.UpdateOracleLifecycleMetrics(notInOrderbook, duplicate, archived, 0, 0)
	// update state and store metrics
	o.updateMetrics()
	return nil
}

// ValidateProposedOrders verifies that the passed orders are all present in the local order store.
// This is called when the BFT module validates a block proposal to ensure that each order
// in the proposed block is an exact match for an order in the witnessed order store.
func (o *Oracle) ValidateProposedOrders(orders *lib.Orders, rootOrderBook *lib.OrderBook) lib.ErrorI {
	// oracle is disabled
	if o == nil {
		return nil
	}
	// handle nil orders case
	if orders == nil {
		o.log.Error("[ORACLE-VALIDATE] Proposal orders == nil, unable to validate orders")
		return nil
	}
	// skip validation and logging when no orders are present
	if len(orders.LockOrders) == 0 && len(orders.CloseOrders) == 0 && len(orders.ResetOrders) == 0 {
		return nil
	}
	// get current safe height for validation
	safeHeight := o.state.GetSafeHeight()
	// entry log with context
	o.log.Debugf("[ORACLE-VALIDATE] Validating proposal: %d lock orders, %d close orders, %d reset orders, safeHeight=%d",
		len(orders.LockOrders), len(orders.CloseOrders), len(orders.ResetOrders), safeHeight)
	// validate each lock order against the witnessed order store
	for _, lock := range orders.LockOrders {
		orderId := lib.BytesToString(lock.OrderId)
		// get order from order store
		witnessedOrder, err := o.orderStore.ReadOrder(lock.OrderId, types.LockOrderType)
		if err != nil {
			o.log.Warnf("[ORACLE-VALIDATE] Lock order %s rejected: not found in store", orderId)
			return ErrOrderNotVerified(orderId, err)
		}
		// check if the witnessed order is from a safe block (has sufficient confirmations)
		if witnessedOrder.WitnessedHeight > safeHeight {
			o.log.Warnf("[ORACLE-VALIDATE] Lock order %s rejected: not safe (witnessed=%d, safe=%d, need %d more blocks)",
				orderId, witnessedOrder.WitnessedHeight, safeHeight, witnessedOrder.WitnessedHeight-safeHeight)
			return ErrOrderNotVerified(orderId, errors.New("order witnessed above safe height"))
		}
		// compare orderbook order and witnessed order
		if !lock.Equals(witnessedOrder.LockOrder) {
			o.log.Warnf("[ORACLE-VALIDATE] Lock order %s rejected: order data mismatch", orderId)
			return ErrOrderNotVerified(orderId, errors.New("lock order unequal"))
		}
		o.log.Infof("[ORACLE-VALIDATE] Lock order %s valid (witnessed=%d)", orderId, witnessedOrder.WitnessedHeight)
	}
	// validate each close order against the witnessed order store
	for _, orderId := range orders.CloseOrders {
		orderIdStr := lib.BytesToString(orderId)
		// get the witnessed order
		witnessedOrder, err := o.orderStore.ReadOrder(orderId, types.CloseOrderType)
		if err != nil {
			o.log.Warnf("[ORACLE-VALIDATE] Close order %s rejected: not found in store", orderIdStr)
			return ErrOrderNotVerified(orderIdStr, err)
		}
		// check if the witnessed order is from a safe block (has sufficient confirmations)
		if witnessedOrder.WitnessedHeight > safeHeight {
			o.log.Warnf("[ORACLE-VALIDATE] Close order %s rejected: not safe (witnessed=%d, safe=%d, need %d more blocks)",
				orderIdStr, witnessedOrder.WitnessedHeight, safeHeight, witnessedOrder.WitnessedHeight-safeHeight)
			return ErrOrderNotVerified(orderIdStr, errors.New("order witnessed above safe height"))
		}
		// construct close order for comparison
		order := lib.CloseOrder{
			OrderId:    orderId,
			ChainId:    o.committee,
			CloseOrder: true,
		}
		// compare orderbook order and witnessed order
		if !order.Equals(witnessedOrder.CloseOrder) {
			o.log.Warnf("[ORACLE-VALIDATE] Close order %s rejected: order data mismatch", orderIdStr)
			return ErrOrderNotVerified(orderIdStr, errors.New("close order unequal"))
		}
		o.log.Infof("[ORACLE-VALIDATE] Close order %s valid (witnessed=%d)", orderIdStr, witnessedOrder.WitnessedHeight)
	}
	// validate each reset order against the current order book and source-chain safe height
	for _, orderId := range orders.ResetOrders {
		if rootOrderBook == nil {
			o.log.Warnf("[ORACLE-VALIDATE] Reset order %s rejected: root order book snapshot is nil", lib.BytesToString(orderId))
			return ErrNilOrderBook()
		}
		orderIDStr := lib.BytesToString(orderId)
		order, err := rootOrderBook.GetOrder(orderId)
		if err != nil {
			o.log.Warnf("[ORACLE-VALIDATE] Reset order %s rejected: error loading order from order book: %v", orderIDStr, err)
			return ErrOrderNotVerified(orderIDStr, err)
		}
		if order == nil {
			o.log.Warnf("[ORACLE-VALIDATE] Reset order %s rejected: order missing from order book", orderIDStr)
			return ErrOrderNotVerified(orderIDStr, errors.New("order not found in order book"))
		}
		if !order.IsLocked() {
			o.log.Warnf("[ORACLE-VALIDATE] Reset order %s rejected: order is not locked", orderIDStr)
			return ErrOrderNotVerified(orderIDStr, errors.New("order is not locked"))
		}
		if safeHeight <= order.BuyerChainDeadline {
			o.log.Warnf("[ORACLE-VALIDATE] Reset order %s rejected: deadline not yet passed (deadline=%d, safe=%d)",
				orderIDStr, order.BuyerChainDeadline, safeHeight)
			return ErrOrderNotVerified(orderIDStr, errors.New("order deadline not passed"))
		}
		o.log.Infof("[ORACLE-VALIDATE] Reset order %s valid (deadline=%d, safe=%d)", orderIDStr, order.BuyerChainDeadline, safeHeight)
	}
	// summary log
	o.log.Infof("[ORACLE-VALIDATE] Validated %d lock orders, %d close orders, and %d reset orders",
		len(orders.LockOrders), len(orders.CloseOrders), len(orders.ResetOrders))
	return nil
}

// CommitCertificate is executed after the quorum agrees on a block
func (o *Oracle) CommitCertificate(qc *lib.QuorumCertificate, block *lib.Block, blockResult *lib.BlockResult, ts uint64) (err lib.ErrorI) {
	// oracle is disabled
	if o == nil {
		return nil
	}
	// Update the last submit height for all lock orders in this certificate
	for _, order := range qc.Results.Orders.LockOrders {
		// get order from order store
		wOrder, err := o.orderStore.ReadOrder(order.OrderId, types.LockOrderType)
		if err != nil {
			o.log.Warnf("[ORACLE-COMMIT] Unable to find order %s in order store", lib.BytesToString(order.OrderId))
			return ErrOrderNotVerified(lib.BytesToString(order.OrderId), err)
		}
		// update the last height this order was submitted
		// TODO is this the proper way to get the root height?
		wOrder.LastSubmitHeight = qc.Header.RootHeight
		// save this update to disk
		err = o.orderStore.WriteOrder(wOrder, types.LockOrderType)
		if err != nil {
			o.log.Errorf("[ORACLE-COMMIT] Failed to write order %s: %v", lib.BytesToString(order.OrderId), err)
			o.metrics.UpdateOracleStoreErrorMetrics(1, 0, 0)
			continue
		}
		o.log.Infof("[ORACLE-COMMIT] Updated last submit height for lock order %s: %d", lib.BytesToString(order.OrderId), qc.Header.RootHeight)
	}
	// Update the last submit height for all close orders in this certificate
	for _, orderId := range qc.Results.Orders.CloseOrders {
		// get order from order store
		wOrder, err := o.orderStore.ReadOrder(orderId, types.CloseOrderType)
		if err != nil {
			o.log.Warnf("[ORACLE-COMMIT] Unable to find order %s in order store", lib.BytesToString(orderId))
			return ErrOrderNotVerified(lib.BytesToString(orderId), err)
		}
		// update the last height this order was submitted
		// TODO is this the proper way to get the root height?
		wOrder.LastSubmitHeight = qc.Header.RootHeight
		// save this update to disk
		err = o.orderStore.WriteOrder(wOrder, types.CloseOrderType)
		if err != nil {
			o.log.Errorf("[ORACLE-COMMIT] Failed to write close order %s: %v", lib.BytesToString(orderId), err)
			o.metrics.UpdateOracleStoreErrorMetrics(1, 0, 0)
			continue
		}
		o.log.Infof("[ORACLE-COMMIT] Updated last submit height for close order %s: %d", lib.BytesToString(orderId), qc.Header.RootHeight)
	}
	// Reset orders are deterministic (deadline-based), so clear stale local close-order witness state.
	for _, orderId := range qc.Results.Orders.ResetOrders {
		orderIDStr := lib.BytesToString(orderId)
		if _, err := o.orderStore.ReadOrder(orderId, types.CloseOrderType); err == nil {
			if e := o.orderStore.RemoveOrder(orderId, types.CloseOrderType); e != nil {
				o.log.Warnf("[ORACLE-COMMIT] Failed removing stale close order %s after reset: %v", orderIDStr, e)
			} else {
				o.log.Infof("[ORACLE-COMMIT] Removed stale close order %s after reset", orderIDStr)
			}
		}
		o.state.ClearOrderHistory(orderId)
	}

	// update lifecycle metrics for committed orders
	o.metrics.UpdateOracleLifecycleMetrics(0, 0, 0, len(qc.Results.Orders.LockOrders), len(qc.Results.Orders.CloseOrders))
	return
}

// UpdateRootChainInfo examines the new root chain order book and prunes the local order store.
// The method performs the following operations:
//   - saves the order book for use in processBlocks
//   - removes lock orders from the store when corresponding sell orders are locked on the root chain
//   - removes lock/close orders when their corresponding sell orders are no longer present
func (o *Oracle) UpdateRootChainInfo(info *lib.RootChainInfo) {
	// oracle is disabled
	if o == nil {
		return
	}
	// track timing for metrics
	startTime := time.Now()
	// track pruned orders and store errors for metrics
	var ordersPruned, storeRemoveErrors int
	defer func() {
		if o.metrics == nil {
			return
		}
		elapsed := time.Since(startTime)
		o.metrics.OrderBookUpdateTime.Observe(elapsed.Seconds())
		o.metrics.RootChainSyncTime.Observe(elapsed.Seconds())
		if ordersPruned > 0 {
			o.metrics.UpdateOracleErrorMetrics(0, ordersPruned, 0)
		}
		if storeRemoveErrors > 0 {
			o.metrics.UpdateOracleStoreErrorMetrics(0, 0, storeRemoveErrors)
		}
	}()
	// lock order book while updating it and updating order store
	o.orderBookMu.Lock()
	defer o.orderBookMu.Unlock()
	// remove history for orders no longer in order book
	o.state.PruneHistory(info.Orders)
	// log a warning for a nil order book
	if info.Orders == nil {
		o.log.Warn("[ORACLE-ORDERBOOK] Order book from root chain was nil")
		return
	}
	// copy and save order book
	o.orderBook = info.Orders.Copy()
	for _, order := range o.orderBook.Orders {
		o.log.Infof("[ORACLE-ORDERBOOK] ORDER %s", formatSellOrder(order))
	}
	// get all lock orders from the order store
	storedOrders, err := o.orderStore.GetAllOrderIds(types.LockOrderType)
	if err != nil {
		o.log.Errorf("[ORACLE-ORDERBOOK] Error getting all order ids: %s", err.Error())
		return
	}
	// examine stored lock orders and remove any not present in the order book
	for _, id := range storedOrders {
		// o.log.Debugf("UpdateRootChainInfo checking stored lock order %x for removal", id)
		// attempt to get stored lock order from order book
		order, err := o.orderBook.GetOrder(id)
		if err != nil {
			o.log.Errorf("[ORACLE-ORDERBOOK] Error getting order from order book: %s", err.Error())
			continue
		}
		// remove lock order from store if one of the following conditions is met:
		//   - corresponding sell order was not found in the root chain order book
		//   - root chain sell order is locked
		switch {
		case order == nil:
			o.log.Infof("[ORACLE-ORDERBOOK] Order %x no longer in order book, removing lock order from store", id)
		case order.BuyerSendAddress != nil:
			o.log.Infof("[ORACLE-ORDERBOOK] Order %x is locked in order book, removing lock order from store", order.Id)
		default:
			// neither condition was met, do not remove this order
			// continue processing remaining stored orders
			continue
		}
		// remove lock order from the store
		err = o.orderStore.RemoveOrder(id, types.LockOrderType)
		if err != nil {
			o.log.Errorf("[ORACLE-ORDERBOOK] Error removing order from order store: %s", err.Error())
			storeRemoveErrors++
		} else {
			ordersPruned++
		}
	}
	// get all close orders from the order store
	storedOrders, err = o.orderStore.GetAllOrderIds(types.CloseOrderType)
	if err != nil {
		o.log.Errorf("[ORACLE-ORDERBOOK] Error getting all order ids: %s", err.Error())
		return
	}
	// examine every stored close order and remove it if is no long present in the order book
	for _, id := range storedOrders {
		// o.log.Debugf("UpdateRootChainInfo checking stored close order %x for removal", id)
		// attempt to get stored close order from order book
		order, err := o.orderBook.GetOrder(id)
		if err != nil {
			o.log.Errorf("[ORACLE-ORDERBOOK] Error getting order from order book: %s", err.Error())
			continue
		}
		// remove close order from store if it was not found in the order book
		if order == nil {
			o.log.Infof("[ORACLE-ORDERBOOK] Removing close order %x from store", id)
			err := o.orderStore.RemoveOrder(id, types.CloseOrderType)
			if err != nil {
				o.log.Errorf("[ORACLE-ORDERBOOK] Error removing order from order store: %s", err.Error())
				storeRemoveErrors++
			} else {
				ordersPruned++
			}
		}
	}
}

// WitnessedOrders returns witnessed orders that match orders in the order book
// When the block proposer produces a block proposal it uses the orders returned here to build the proposed block
// TODO watch for conflicts while syncing ethereum block, prooducer might resubmit order
func (o *Oracle) WitnessedOrders(orderBook *lib.OrderBook, rootHeight uint64) ([]*lib.LockOrder, [][]byte, [][]byte) {
	lockOrders := []*lib.LockOrder{}
	closeOrders := [][]byte{}
	resetOrders := [][]byte{}
	// oracle is disabled
	if o == nil {
		return lockOrders, closeOrders, resetOrders
	}
	// get current heights for context
	safeHeight := o.state.GetSafeHeight()
	// track statistics for summary log
	var stats struct {
		lockChecked, lockSubmitting, lockHeldSafe, lockHeldDelay     int
		closeChecked, closeSubmitting, closeHeldSafe, closeHeldDelay int
		resetChecked, resetSubmitting                                int
	}
	// entry log with key context
	// o.log.Debugf("[ORACLE-SUBMIT] Checking orders: orderBook=%d orders, rootHeight=%d, safeHeight=%d",
	// 	len(orderBook.Orders), rootHeight, safeHeight)
	// loop through the order book searching the order store for lock/close orders witnessed by this node
	for _, order := range orderBook.Orders {
		orderId := lib.BytesToString(order.Id)
		// process unlocked sell order
		if !order.IsLocked() {
			stats.lockChecked++
			// try to find a lock order
			wOrder, err := o.orderStore.ReadOrder(order.Id, types.LockOrderType)
			if err != nil {
				if err.Code() != CodeReadOrder {
					o.log.Errorf("[ORACLE-SUBMIT] Failed to read lock order %s: %v", orderId, err)
				}
				continue
			}
			// check if the witnessed order is from a safe block (has sufficient confirmations)
			if wOrder.WitnessedHeight > safeHeight {
				blocksUntilSafe := wOrder.WitnessedHeight - safeHeight
				o.log.Debugf("[ORACLE-SUBMIT] Lock order %s held: awaiting safe height (witnessed=%d, safe=%d, need %d more blocks)",
					orderId, wOrder.WitnessedHeight, safeHeight, blocksUntilSafe)
				stats.lockHeldSafe++
				continue
			}
			// check whether this witnessed lock order should be submitted in the next proposed block
			if !o.state.shouldSubmit(wOrder, rootHeight, o.config) {
				// shouldSubmit logs the specific reason internally
				stats.lockHeldDelay++
				continue
			}
			o.log.Infof("[ORACLE-SUBMIT] Submitting lock order %s (witnessed=%d)", orderId, wOrder.WitnessedHeight)
			stats.lockSubmitting++
			// submit this witnessed lock order by returning it in the lockOrders slice
			lockOrders = append(lockOrders, wOrder.LockOrder)
		} else {
			stats.resetChecked++
			if order.BuyerChainDeadline > 0 && safeHeight > order.BuyerChainDeadline {
				stats.resetSubmitting++
				o.log.Infof("[ORACLE-SUBMIT] Submitting reset order %s (deadline=%d, safe=%d)",
					orderId, order.BuyerChainDeadline, safeHeight)
				resetOrders = append(resetOrders, order.Id)
				continue
			}
			stats.closeChecked++
			// process locked orders - look for witnessed close orders
			wOrder, err := o.orderStore.ReadOrder(order.Id, types.CloseOrderType)
			if err != nil {
				if err.Code() != CodeReadOrder {
					o.log.Errorf("[ORACLE-SUBMIT] Failed to read close order %s: %v", orderId, err)
				}
				// No witnessed order is a normal condition, do not log
				continue
			}
			// check if the witnessed order is from a safe block (has sufficient confirmations)
			if wOrder.WitnessedHeight > safeHeight {
				blocksUntilSafe := wOrder.WitnessedHeight - safeHeight
				o.log.Debugf("[ORACLE-SUBMIT] Close order %s held: awaiting safe height (witnessed=%d, safe=%d, need %d more blocks)",
					orderId, wOrder.WitnessedHeight, safeHeight, blocksUntilSafe)
				stats.closeHeldSafe++
				continue
			}
			// check whether this witnessed close order should be submitted in the next proposed block
			if !o.state.shouldSubmit(wOrder, rootHeight, o.config) {
				// shouldSubmit logs the specific reason internally
				stats.closeHeldDelay++
				continue
			}
			// update the last height this order was submitted
			wOrder.LastSubmitHeight = rootHeight
			// update the witnessed order in the store
			err = o.orderStore.WriteOrder(wOrder, types.CloseOrderType)
			if err != nil {
				o.log.Errorf("[ORACLE-SUBMIT] Failed to write close order %s: %v", orderId, err)
				o.metrics.UpdateOracleStoreErrorMetrics(1, 0, 0)
				continue
			}
			o.log.Infof("[ORACLE-SUBMIT] Submitting close order %s (witnessed=%d)", orderId, wOrder.WitnessedHeight)
			stats.closeSubmitting++
			// submit this witnessed close order by returning it in the closeOrders slice
			closeOrders = append(closeOrders, wOrder.OrderId)
		}
	}
	// summary log with full picture
	o.log.Infof("[ORACLE-SUBMIT] rootHeight=%d: lock[checked=%d submitting=%d heldSafe=%d heldDelay=%d] close[checked=%d submitting=%d heldSafe=%d heldDelay=%d] reset[checked=%d submitting=%d]",
		rootHeight,
		stats.lockChecked, stats.lockSubmitting, stats.lockHeldSafe, stats.lockHeldDelay,
		stats.closeChecked, stats.closeSubmitting, stats.closeHeldSafe, stats.closeHeldDelay,
		stats.resetChecked, stats.resetSubmitting)
	// update submitted orders metrics
	totalSubmitted := stats.lockSubmitting + stats.closeSubmitting + stats.resetSubmitting
	o.metrics.UpdateOracleOrderMetrics(0, 0, totalSubmitted, 0, 0)
	// update submission tracking metrics
	totalHeldSafe := stats.lockHeldSafe + stats.closeHeldSafe
	o.metrics.UpdateOracleSubmissionMetrics(totalHeldSafe, 0, 0, 0, 0)
	// update orders awaiting confirmation gauge
	if o.metrics != nil && o.metrics.OrdersAwaitingConfirmation != nil {
		o.metrics.OrdersAwaitingConfirmation.Set(float64(totalHeldSafe))
	}
	return lockOrders, closeOrders, resetOrders
}

// updateMetrics updates various oracle metrics with current state information
func (o *Oracle) updateMetrics() {
	// exit if empty
	if o.metrics == nil {
		return
	}
	// get current state metrics
	safeHeight := o.state.GetSafeHeight()
	lastHeight := o.state.GetLastHeight()
	sourceHeight := o.state.sourceChainHeight
	// get submission history sizes
	lockOrderSubmissionsSize := len(o.state.lockOrderSubmissions)
	closeOrderSubmissionsSize := len(o.state.closeOrderSubmissions)
	// get order store counts
	lockOrderIds, err := o.orderStore.GetAllOrderIds(types.LockOrderType)
	lockOrders := 0
	if err == nil {
		lockOrders = len(lockOrderIds)
	}
	closeOrderIds, err := o.orderStore.GetAllOrderIds(types.CloseOrderType)
	closeOrders := 0
	if err == nil {
		closeOrders = len(closeOrderIds)
	}
	// update height metrics
	o.metrics.UpdateOracleHeightMetrics(lastHeight, safeHeight, sourceHeight, 0)
	// update state metrics
	o.metrics.UpdateOracleStateMetrics(safeHeight, sourceHeight, lockOrderSubmissionsSize, closeOrderSubmissionsSize)
	// update store metrics
	o.metrics.UpdateOracleStoreMetrics(lockOrders, closeOrders)
}

// formatSellOrder formats a SellOrder with hex-encoded byte fields for logging
func formatSellOrder(o *lib.SellOrder) string {
	return fmt.Sprintf(
		"Id:%x Committee:%d Data:%x AmountForSale:%d RequestedAmount:%d SellerReceive:%x SellerSend:%x",
		o.Id, o.Committee, o.Data, o.AmountForSale, o.RequestedAmount,
		o.SellerReceiveAddress, o.SellersSendAddress,
	)
}
