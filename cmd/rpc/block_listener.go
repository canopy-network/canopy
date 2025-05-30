package rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

// BlockListener listens for new blocks and processes transactions
type BlockListener struct {
	blockProvider    wstypes.BlockProvider
	orderBookReader  RootChainOrderBookReader
	transactionStore OrderStorer
	logger           lib.LoggerI
	lastBlockHeight  uint64
	blockHeightFile  string
}

// OrderStorer persists a transaction
type OrderStorer interface {
	WriteOrder(orderId string, orderType wstypes.OrderType, blockHeight uint64, txHash string, data []byte) error
}

// RootChainOrderBookReader gets the order book from the root chain
type RootChainOrderBookReader interface {
	GetOrderBook() (*lib.OrderBook, lib.ErrorI)
}

// NewBlockListener creates a new BlockListener
func NewBlockListener(
	blockProvider wstypes.BlockProvider,
	orderBookReader RootChainOrderBookReader,
	transactionStore OrderStorer,
	logger lib.LoggerI,
) *BlockListener {
	// create a new block listener
	bl := &BlockListener{
		blockProvider:    blockProvider,
		orderBookReader:  orderBookReader,
		transactionStore: transactionStore,
		logger:           logger,
		blockHeightFile:  "last_block_height.txt",
	}

	// read the last seen block height from disk
	height, err := bl.readLastBlockHeight()
	if err != nil {
		// if there's an error, log it and start from block 0
		bl.logger.Warnf("ETHSWAP failed to read last block height: %v, starting from block 0", err)
		height = 0
	}

	// set the next block to be processed
	bl.lastBlockHeight = height
	bl.blockProvider.SetNext(height)
	bl.logger.Infof("ETHSWAP block listener initialized, starting from block height: %d", height)

	return bl
}

// readLastBlockHeight reads the last seen block height from disk
func (bl *BlockListener) readLastBlockHeight() (uint64, error) {
	// read the file containing the last block height
	data, err := os.ReadFile(bl.blockHeightFile)
	if err != nil {
		// if the file doesn't exist, return 0
		if os.IsNotExist(err) {
			bl.logger.Info("last block height file not found, starting from block 0")
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read last block height file: %w", err)
	}

	// convert the data to a uint64
	heightStr := string(data)
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last block height: %w", err)
	}

	return height, nil
}

// writeLastBlockHeight writes the last seen block height to disk
func (bl *BlockListener) writeLastBlockHeight(height uint64) error {
	// ensure the directory exists
	dir := filepath.Dir(bl.blockHeightFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for block height file: %w", err)
	}

	// write the height to the file
	heightStr := strconv.FormatUint(height, 10)
	if err := os.WriteFile(bl.blockHeightFile, []byte(heightStr), 0644); err != nil {
		return fmt.Errorf("failed to write last block height: %w", err)
	}

	return nil
}

// Start begins listening for new blocks
func (bl *BlockListener) Start() {
	bl.logger.Info("starting block listener")

	// get the block channel from the provider
	blockCh := bl.blockProvider.BlockCh()

	// listen for new blocks
	for block := range blockCh {
		bl.processBlock(block)
	}
}

// processBlock processes a single block
func (bl *BlockListener) processBlock(block wstypes.BlockI) {
	// get the block height
	height := block.Number()
	bl.logger.Infof("ETHSWAP processing block at height: %d %d", height, len(block.Transactions()))

	// persist the block height to disk
	if err := bl.writeLastBlockHeight(height); err != nil {
		bl.logger.Errorf("ETHSWAP failed to write block height: %v", err)
	}

	// update the last block height
	bl.lastBlockHeight = height

	// process all transactions in the block
	for _, tx := range block.Transactions() {
		bl.processTransaction(tx, height)
	}
}

// processTransaction processes a single transaction
func (bl *BlockListener) processTransaction(tx wstypes.TransactionI, blockHeight uint64) {
	// get the from and to addresses
	from := tx.From()
	to := tx.To()
	txHash := tx.Hash()

	bl.logger.Debugf("processing transaction %s from %s to %s", txHash, from, to)

	// check if this is a token transfer
	transfer := tx.TokenTransfer()
	if transfer.TransactionID != "" {
		bl.processTransfer(transfer, tx, blockHeight)
	}

	// check if this is a self-transaction (from and to are the same)
	if from == to {
		bl.processSelfTransaction(tx, blockHeight)
	}
}

// processTransfer processes a token transfer transaction
func (bl *BlockListener) processTransfer(transfer wstypes.TokenTransfer, tx wstypes.TransactionI, blockHeight uint64) {
	// log the transfer details
	bl.logger.Infof("token transfer detected: %+v", transfer)

	// attempt to unmarshal the transaction data into a CloseOrder
	var closeOrder lib.CloseOrder
	if err := json.Unmarshal(tx.Data(), &closeOrder); err != nil {
		bl.logger.Debugf("failed to unmarshal transaction data as CloseOrder: %v", err)
		return
	}

	// verify the order exists in the order book
	orderBook, err := bl.orderBookReader.GetOrderBook()
	if err != nil {
		bl.logger.Errorf("failed to get order book: %v", err)
		return
	}

	// check if the order exists
	order, err := orderBook.GetOrder(closeOrder.OrderId)
	if err != nil || order == nil {
		bl.logger.Warnf("order %s not found in order book", hex.EncodeToString(closeOrder.OrderId))
		return
	}

	// persist the close order
	orderId := hex.EncodeToString(closeOrder.OrderId)
	data, e := json.Marshal(&closeOrder)
	if e != nil {
		bl.logger.Errorf("failed to marshal close order: %v", err)
		return
	}

	// write the order to storage
	if err := bl.transactionStore.WriteOrder(orderId, "close_order", blockHeight, tx.Hash(), data); err != nil {
		bl.logger.Errorf("failed to write close order: %v", err)
		return
	}

	bl.logger.Infof("persisted close order %s at block %d", orderId, blockHeight)
}

// processSelfTransaction processes a transaction where the sender and receiver are the same
func (bl *BlockListener) processSelfTransaction(tx wstypes.TransactionI, blockHeight uint64) {
	// attempt to unmarshal the transaction data into a LockOrder
	var lockOrder lib.LockOrder
	if err := json.Unmarshal(tx.Data(), &lockOrder); err != nil {
		bl.logger.Debugf("failed to unmarshal transaction data as LockOrder: %v", err)
		return
	}

	// verify the order exists in the order book
	orderBook, err := bl.orderBookReader.GetOrderBook()
	if err != nil {
		bl.logger.Errorf("failed to get order book: %v", err)
		return
	}

	// check if the order exists
	order, err := orderBook.GetOrder(lockOrder.OrderId)
	if err != nil || order == nil {
		bl.logger.Warnf("order %s not found in order book", hex.EncodeToString(lockOrder.OrderId))
		return
	}

	// persist the lock order
	orderId := hex.EncodeToString(lockOrder.OrderId)
	data, e := json.Marshal(&lockOrder)
	if e != nil {
		bl.logger.Errorf("failed to marshal lock order: %v", err)
		return
	}

	// write the order to storage
	if err := bl.transactionStore.WriteOrder(orderId, "lock_order", blockHeight, tx.Hash(), data); err != nil {
		bl.logger.Errorf("failed to write lock order: %v", err)
		return
	}

	bl.logger.Infof("persisted lock order %s at block %d", orderId, blockHeight)
}

// CreateLockOrder iterates through the order book and creates a lock order for the first sell order it finds
func (bl *BlockListener) CreateLockOrder() *lib.LockOrder {
	// Log that we're creating a lock order
	bl.logger.Info("Creating lock order from first available sell order")

	// Get the order book
	orderBook, err := bl.orderBookReader.GetOrderBook()
	if err != nil {
		// Log the error and return if we couldn't get the order book
		bl.logger.Errorf("Failed to get order book: %v", err)
		return nil
	}

	for _, order := range orderBook.Orders {
		// Order already locked
		if order.BuyerSendAddress != nil {
			continue
		}
		order := &lib.LockOrder{
			OrderId:             order.Id,
			BuyerReceiveAddress: []byte("12345678901234567890"),
			BuyerSendAddress:    []byte("12345678901234567890"),
		}
		return order
	}
	return nil
}
