package oracle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

const (
	// heightFileName is the file name for storing last seen block height
	heightFileName = "eth_last_height.txt"
	// canopyDir is the directory name for canopy data
	canopyDir = ".canopy"
)

// Oracle listens for new blocks and processes transactions
type Oracle struct {
	blockProvider types.BlockProvider
	orderStore    types.OrderStoreI
	orderBook     *lib.OrderBook
	logger        lib.LoggerI
}

// NewOracle creates a new Oracle instance
func NewOracle(blockProvider types.BlockProvider, transactionStore types.OrderStoreI, logger lib.LoggerI) *Oracle {
	// create new oracle instance
	o := &Oracle{
		blockProvider: blockProvider,
		orderStore:    transactionStore,
		logger:        logger,
	}
	// read last seen block height from disk
	height, err := o.readLastSeenHeight()
	if err != nil {
		logger.Errorf("Failed to read last seen height: %v", err)
		// return and allow default behaviour from provider
		return o
	}
	// instruct the provider to start at the next block
	blockProvider.SetHeight(height + 1)
	// return new oracle instance
	return o
}

// Start begins listening for blocks synchronously
func (o *Oracle) Start() {
	// log that we're starting the oracle
	o.logger.Info("Starting oracle")
	// get the block channel from provider
	blockCh := o.blockProvider.BlockCh()
	// listen for blocks
	go func() {
		for block := range blockCh {
			// process the received block
			o.processBlock(block)
		}
	}()
}

// processBlock handles processing of a single block
func (o *Oracle) processBlock(block types.BlockI) {
	// log that we received a new block
	o.logger.Infof("Received block %d with hash %s", block.Number(), block.Hash())
	// persist block height to disk
	if err := o.persistBlockHeight(block.Number()); err != nil {
		o.logger.Errorf("Failed to persist block height: %v", err)
	}
	// get all transactions from the block
	transactions := block.Transactions()
	// iterate through each transaction
	for _, tx := range transactions {
		// process the transaction
		o.processTransaction(tx, block.Number())
	}
}

// processTransaction handles processing of a single transaction
func (o *Oracle) processTransaction(tx types.TransactionI, blockHeight uint64) error {
	switch {
	case tx.To() == tx.From():
		// process as potential lock order
		if err := o.processLockOrderTransaction(tx, blockHeight); err != nil {
			o.logger.Errorf("Failed to process lock order transaction: %s", err.Error())
			return err
		}
	case tx.TokenTransfer().TransactionID != "":
		// print the transfer details
		// process as potential close order
		if err := o.processCloseOrderTransaction(tx, blockHeight, tx.TokenTransfer()); err != nil {
			o.logger.Errorf("Failed to process close order transaction: %s", err.Error())
			return err
		}
	}
	return nil
}

// processLockOrderTransaction processes transactions that might be lock orders
func (o *Oracle) processLockOrderTransaction(tx types.TransactionI, blockHeight uint64) lib.ErrorI {
	// attempt to unmarshal transaction data into lock order
	var lockOrder lib.LockOrder
	err := json.Unmarshal(tx.Data(), &lockOrder)
	if err != nil {
		return types.ErrUnmarshalOrder(err)
	}
	// verify order id exists in order book
	if !o.verifyOrderExists(lockOrder.OrderId) {
		// log that order id not found and return error
		return types.ErrOrderNotFoundInBook(lib.BytesToString(lockOrder.OrderId))
	}
	// marshal lock order to json
	data, err := lockOrder.MarshalJSON()
	if err != nil {
		return types.ErrMarshalOrder(err)
	}
	// persist lock order to disk
	err = o.orderStore.WriteOrder(lockOrder.OrderId, types.LockOrderType, data)
	if err != nil {
		return types.ErrPersistOrder(err)
	}
	// log successful persistence
	o.logger.Infof("Successfully persisted lock order %s", lockOrder.OrderId)
	return nil
}

// processCloseOrderTransaction processes transactions that might be close orders
func (o *Oracle) processCloseOrderTransaction(tx types.TransactionI, blockHeight uint64, transfer types.TokenTransfer) lib.ErrorI {
	// attempt to unmarshal transaction data into close order
	var closeOrder lib.CloseOrder
	err := json.Unmarshal(tx.Data(), &closeOrder)
	if err != nil {
		return types.ErrUnmarshalOrder(err)
	}
	// verify order id exists in order book
	if !o.verifyOrderExists(closeOrder.OrderId) {
		// log that order id not found and return error
		return types.ErrOrderNotFoundInBook(lib.BytesToString(closeOrder.OrderId))
	}
	// find the order in the order book
	order, orderErr := o.orderBook.GetOrder(closeOrder.OrderId)
	if orderErr != nil || order == nil {
		// log error getting specific order and return error
		o.logger.Errorf("Failed to get order from order book: %v", orderErr)
		return types.ErrGetOrderBook(orderErr)
	}
	// verify base transfer amount equals order amount
	if transfer.TokenBaseAmount != order.RequestedAmount {
		// log amount mismatch and return error
		o.logger.Warnf("Transfer amount %d does not match order amount %d", transfer.TokenBaseAmount, order.RequestedAmount)
		return types.ErrAmountMismatch(transfer.TokenBaseAmount, order.RequestedAmount)
	}
	// marshal close order to json
	data, err := closeOrder.MarshalJSON()
	if err != nil {
		return types.ErrMarshalOrder(err)
	}
	// persist close order to disk
	err = o.orderStore.WriteOrder(closeOrder.OrderId, types.CloseOrderType, data)
	if err != nil {
		return types.ErrPersistOrder(err)
	}
	// log successful persistence
	o.logger.Infof("Successfully persisted close order %s", string(closeOrder.OrderId))
	return nil
}

// verifyOrderExists checks if an order exists in the order book
func (o *Oracle) verifyOrderExists(orderId []byte) bool {
	order, _ := o.orderBook.GetOrder(orderId)
	if order == nil {
		return false
	}
	return true
}

// readLastSeenHeight reads the last seen block height from disk
func (o *Oracle) readLastSeenHeight() (uint64, lib.ErrorI) {
	// get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, types.ErrGetHomeDirectory(err)
	}
	// construct file path
	filePath := filepath.Join(homeDir, canopyDir, heightFileName)
	// read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		// log that file doesn't exist and return 0 with no error (this is expected)
		o.logger.Infof("Last height file not found, starting from 0: %v", err)
		return 0, nil
	}
	// parse height from file contents
	heightStr := strings.TrimSpace(string(data))
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		return 0, types.ErrParseHeight(err)
	}
	// log the height we read
	o.logger.Infof("Read last seen height: %d", height)
	return height, nil
}

// persistBlockHeight saves the block height to disk
func (o *Oracle) persistBlockHeight(height uint64) lib.ErrorI {
	// get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// log error and return error
		o.logger.Errorf("Failed to get home directory: %v", err)
		return types.ErrGetHomeDirectory(err)
	}
	// construct directory path
	dirPath := filepath.Join(homeDir, canopyDir)
	// create directory if it doesn't exist
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		// log error creating directory and return error
		o.logger.Errorf("Failed to create directory: %v", err)
		return types.ErrCreateDirectory(err)
	}
	// construct file path
	filePath := filepath.Join(dirPath, heightFileName)
	// convert height to string
	heightStr := strconv.FormatUint(height, 10)
	// write height to file
	err = os.WriteFile(filePath, []byte(heightStr), 0644)
	if err != nil {
		// log error writing file and return error
		o.logger.Errorf("Failed to write height to file: %v", err)
		return types.ErrWriteHeightFile(err)
	}
	// log successful write
	o.logger.Debugf("Persisted block height: %d", height)
	return nil
}

// ValidateProposedOrders verifies that the passed orders are all present in the local order store
func (o *Oracle) ValidateProposedOrders(orders *lib.Orders) lib.ErrorI {
	// if the orders are empty
	if orders == nil {
		return nil
	}
	// no orders to validate
	if len(orders.LockOrders) == 0 && len(orders.CloseOrders) == 0 {
		return nil
	}
	// validate lock orders
	for _, lock := range orders.LockOrders {
		o.logger.Infof("Verifying lock order %s", lib.BytesToString(lock.OrderId))
		bz, err := json.Marshal(lock)
		if err != nil {
			return lib.ErrJSONMarshal(err)
		}

		// verify a matching order exists in the order store
		err = o.orderStore.VerifyOrder(lock.OrderId, types.LockOrderType, bz)
		if err != nil {
			return types.ErrOrderNotVerified(lib.BytesToString(lock.OrderId), err)
		}
	}
	// validate close orders
	for _, orderId := range orders.CloseOrders {
		o.logger.Infof("Verifying close order %s", lib.BytesToString(orderId))
		order := lib.CloseOrder{
			OrderId:    orderId,
			ChainId:    0,
			CloseOrder: true,
		}
		bz, err := order.MarshalJSON()
		if err != nil {
			return lib.ErrJSONMarshal(err)
		}

		// verify a matching order exists in the order store
		err = o.orderStore.VerifyOrder(orderId, types.CloseOrderType, bz)
		if err != nil {
			return types.ErrOrderNotVerified(lib.BytesToString(orderId), err)
		}
	}
	o.logger.Info("Validated off chain orders successfully")
	return nil
}

// OrderBookUpdate removes orders from the store that are not longer present in the order book
func (o *Oracle) OrderBookUpdate(orderBook *lib.OrderBook) {
	o.orderBook = orderBook
	// get all lock orders from the order store
	storedOrders := o.orderStore.GetAllOrderIds(types.LockOrderType)
	o.logger.Debugf("TransactionStore UpdateRootChainInfo %d stored lock orders", len(storedOrders))
	// examine every stored lock order and remove it if is no long present in the order book
	for _, id := range storedOrders {
		// attempt to get stored lock order from order book
		order, err := orderBook.GetOrder(id)
		if err != nil {
			o.logger.Errorf("Error getting order from order book: %s", err.Error())
			continue
		}
		// remove lock order from store if it was not found in the order book
		if order == nil {
			o.logger.Infof("TransactionStore Removing lock order %x from store", id)
			err := o.orderStore.RemoveOrder(id, types.LockOrderType)
			if err != nil {
				o.logger.Errorf("Error removing order from order store: %s", err.Error())
			}
		}
	}

	// get all close orders from the order store
	storedOrders = o.orderStore.GetAllOrderIds(types.CloseOrderType)
	o.logger.Debugf("TransactionStore UpdateRootChainInfo %d stored close orders", len(storedOrders))
	// examine every stored close order and remove it if is no long present in the order book
	for _, id := range storedOrders {
		// attempt to get stored close order from order book
		order, err := orderBook.GetOrder(id)
		if err != nil {
			o.logger.Errorf("Error getting order from order book: %s", err.Error())
			continue
		}
		// remove close order from store if it was not found in the order book
		if order == nil {
			o.logger.Infof("Removing close order %x from store", id)
			err := o.orderStore.RemoveOrder(id, types.CloseOrderType)
			if err != nil {
				o.logger.Errorf("Error removing order from order store: %s", err.Error())
			}
		}
	}
}

// GatherWitnessedOrders finds any witnessed orders that match orders in the order book
func (o *Oracle) GatherWitnessedOrders(orderBook *lib.OrderBook) ([]*lib.LockOrder, [][]byte) {
	lockOrders := []*lib.LockOrder{}
	closeOrders := [][]byte{}

	// iterate over the order book, looking in the order store for any lock/close orders this node has witnessed
	for _, order := range orderBook.Orders {
		if order.BuyerSendAddress == nil {
			// process unlocked orders - look for witnessed lock orders
			orderBytes, err := o.orderStore.ReadOrder(order.Id, types.LockOrderType)
			if err != nil {
				// No witnessed order is a normal condition, do not log
				continue
			}
			// unmarshal lock order
			var lockOrder lib.LockOrder
			err = json.Unmarshal(orderBytes, &lockOrder)
			if err != nil {
				o.logger.Errorf("Error reading lock order: %v", err)
				continue
			}
			o.logger.Debugf("Witnessed lock order %x", lockOrder.OrderId)
			// include witnessed lock order
			lockOrders = append(lockOrders, &lockOrder)
		} else {
			// process locked orders - look for witnessed close orders
			orderBytes, err := o.orderStore.ReadOrder(order.Id, types.CloseOrderType)
			if err != nil {
				// No witnessed order is a normal condition, do not log
				continue
			}
			// unmarshal close order
			var closeOrder lib.CloseOrder
			err = json.Unmarshal(orderBytes, &closeOrder)
			if err != nil {
				o.logger.Errorf("Error reading close order: %v", err)
				continue
			}
			o.logger.Debugf("Witnessed close order %x", closeOrder.OrderId)
			// include witnessed close order
			closeOrders = append(closeOrders, closeOrder.OrderId)
		}
	}

	o.logger.Infof("Witnessed %d lock orders and %d close orders", len(lockOrders), len(closeOrders))
	return lockOrders, closeOrders
}
