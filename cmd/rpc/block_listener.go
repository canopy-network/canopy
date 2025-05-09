// Create a type that does the following:
// - Is called BlockListener and has a constructor called NewBlockListener
// - Constructor accepts channel of type BlockI, a TransactionStore and a lib.LoggerI for logging
// - BlockLister receives BlockI messages over the channel
// - Iterates over the transactions in the received block
// -- For transactions where the To and From address are equal, validate the data field is proper JSON and if so, store the transaction
// --

// Use this interface to write transactions to disk:
// type TransactionStorer interface {
//  WriteTx(blockHeight uint64, txID string, data string) error
// }

// Use these imports
// "github.com/ethereum/go-ethereum/core/types"
// "github.com/ethereum/go-ethereum/ethclient"
// "github.com/ethereum/go-ethereum/rpc"

// Other notes:
// - package name is rpc
package rpc

import (
	"encoding/json"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

// TransactionStorer defines an interface for storing transaction data
type TransactionStorer interface {
	WriteTx(blockHeight uint64, txID string, data string) error
}

// BlockListener listens for new blocks and processes their transactions
type BlockListener struct {
	blockCh  chan wstypes.BlockI
	txStorer TransactionStorer
	logger   lib.LoggerI
}

// NewBlockListener creates a new BlockListener instance
func NewBlockListener(blockCh chan wstypes.BlockI, txStorer TransactionStorer, logger lib.LoggerI) *BlockListener {
	// Log the creation of the block listener
	logger.Info("Creating new BlockListener")

	return &BlockListener{
		blockCh:  blockCh,
		txStorer: txStorer,
		logger:   logger,
	}
}

// Start begins listening for blocks on the channel
func (bl *BlockListener) Start() {
	bl.logger.Info("Starting BlockListener")

	// Start a goroutine to process blocks
	go bl.processBlocks()
}

// processBlocks handles incoming blocks from the channel
func (bl *BlockListener) processBlocks() {
	bl.logger.Debug("Entered processBlocks method")

	for block := range bl.blockCh {
		bl.logger.Infof("Received block #%d with hash %s", block.Number(), block.Hash())
		bl.processTransactions(block)
	}

	bl.logger.Info("BlockListener channel closed, exiting processBlocks")
}

// processTransactions iterates through all transactions in a block
func (bl *BlockListener) processTransactions(block wstypes.BlockI) {
	bl.logger.Debugf("Processing %d transactions from block #%d", len(block.Transactions()), block.Number())

	for _, tx := range block.Transactions() {
		bl.logger.Debugf("Processing transaction: %s", tx.Hash())

		// Check if sender and receiver are the same
		if tx.From() == tx.To() {
			bl.logger.Debugf("Found transaction where From (%s) = To (%s)", tx.From(), tx.To())
			bl.processEqualAddressTx(block.Number(), tx)
		}
	}
}

// processEqualAddressTx handles transactions where From = To
func (bl *BlockListener) processEqualAddressTx(blockHeight uint64, tx wstypes.TransactionI) {
	// Check if data is valid JSON
	if isValidJson(tx.Data()) {
		bl.logger.Infof("Valid JSON found in transaction %s", tx.Hash())

		// Convert data to string for storage
		dataStr := string(tx.Data())

		// Store the transaction
		err := bl.txStorer.WriteTx(blockHeight, tx.Hash(), dataStr)
		if err != nil {
			bl.logger.Errorf("Failed to store transaction %s: %v", tx.Hash(), err)
			return
		}

		bl.logger.Infof("Stored transaction %s from block #%d", tx.Hash(), blockHeight)
	} else {
		bl.logger.Warnf("Transaction %s data is not valid JSON", tx.Hash())
	}
}

// isValidJson checks if the given data is valid JSON
func isValidJson(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}
