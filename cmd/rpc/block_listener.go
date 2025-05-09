// Create a type that does the following:
// - Is called BlockListener and has a constructor called NewBlockListener
// - Constructor accepts channel of type BlockI, a TransactionStore and a lib.LoggerI for logging
// - BlockLister receives BlockI messages over the channel
// - Iterates over the transactions in the received block
// -- For transactions where the To and From address are equal, attempt to unmarshal the transaction data into a LockOrder. If that succeeds, persist the transaction to disk
// -

// Use this interface to write transactions to disk:
// type TransactionStorer interface {
//  WriteTx(blockHeight uint64, txID string, data string) error
// }

// Other notes:
// - package name is rpc
// START
package rpc

import (
	"encoding/json"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

// TransactionStorer defines the interface for writing transaction data to disk
type TransactionStorer interface {
	WriteTx(blockHeight uint64, txID string, data string) error
}

// BlockListener listens for incoming blocks and processes their transactions
type BlockListener struct {
	blockCh chan wstypes.BlockI // Channel to receive blocks
	txStore TransactionStorer   // Store for persisting transactions
	logger  lib.LoggerI         // Logger for output messages
	stopCh  chan struct{}       // Channel to signal stopping
	doneCh  chan struct{}       // Channel to signal completion of stopping
}

// NewBlockListener creates a new BlockListener
func NewBlockListener(blockCh chan wstypes.BlockI, txStore TransactionStorer, logger lib.LoggerI) *BlockListener {
	// Create a new BlockListener instance
	bl := &BlockListener{
		blockCh: blockCh,
		txStore: txStore,
		logger:  logger,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	// Log that the block listener was created
	logger.Info("Block listener created")

	// Return the new block listener
	return bl
}

// listen starts the block listening process
func (bl *BlockListener) Start() {
	// Log that the listener is starting
	bl.logger.Info("Block listener starting")

	// Continue listening until stopped
	for {
		select {
		case block := <-bl.blockCh:
			// Process the received block
			bl.processBlock(block)

		case <-bl.stopCh:
			// Log that the listener is stopping
			bl.logger.Info("Block listener stopping")

			// Signal that the listener has stopped
			close(bl.doneCh)

			return
		}
	}
}

// processBlock processes a single block
func (bl *BlockListener) processBlock(block wstypes.BlockI) {
	// Log the block being processed
	bl.logger.Infof("Processing block %d with hash %s", block.Number(), block.Hash())

	// Get all transactions in the block
	txs := block.Transactions()

	// Log the number of transactions found
	bl.logger.Debugf("Found %d transactions in block %d", len(txs), block.Number())

	// Iterate through all transactions
	for _, tx := range txs {
		// Process each transaction
		bl.procesTransaction(block.Number(), tx)
	}

	// Log that block processing is complete
	bl.logger.Debugf("Finished processing block %d", block.Number())
}

// procesTransaction processes a single transaction
func (bl *BlockListener) procesTransaction(blockHeight uint64, tx wstypes.TransactionI) {
	// Get the sender and recipient addresses
	from := tx.From()
	to := tx.To()

	// Log the transaction details
	bl.logger.Debugf("Processing transaction %s: from=%s, to=%s", tx.Hash(), from, to)

	// Check if the sender and recipient are the same
	if from != to {
		// Skip if addresses aren't equal
		bl.logger.Debugf("Skipping transaction %s: sender and recipient addresses are different", tx.Hash())
		return
	}

	// Log that we found a self-transaction
	bl.logger.Debugf("Found self-transaction %s, attempting to unmarshal as LockOrder", tx.Hash())

	// Get the transaction data
	data := tx.Data()

	// Create a LockOrder instance to unmarshal into
	var lockOrder lib.LockOrder

	// Attempt to unmarshal the transaction data
	err := json.Unmarshal(data, &lockOrder)
	if err != nil {
		// Log the unmarshal error
		bl.logger.Warnf("Failed to unmarshal transaction %s data as LockOrder: %v", tx.Hash(), err)
		return
	}

	// Log successful unmarshal
	bl.logger.Infof("Successfully unmarshaled transaction %s as LockOrder", tx.Hash())

	// Marshal back to a string for storage
	dataStr, err := json.Marshal(lockOrder)
	if err != nil {
		// Log the marshal error
		bl.logger.Errorf("Failed to marshal LockOrder back to JSON for storage: %v", err)
		return
	}

	// Store the transaction
	txID := tx.Hash()
	bl.logger.Infof("Persisting transaction %s to disk", txID)

	// Write the transaction to the store
	err = bl.txStore.WriteTx(blockHeight, txID, string(dataStr))
	if err != nil {
		// Log the storage error
		bl.logger.Errorf("Failed to write transaction %s to store: %v", txID, err)
		return
	}

	// Log successful storage
	bl.logger.Infof("Successfully persisted transaction %s", txID)
}

// Stop stops the listener and waits for it to complete
func (bl *BlockListener) Stop() {
	// Send the stop signal
	bl.logger.Info("Sending stop signal to block listener")
	close(bl.stopCh)

	// Wait for the listener to stop
	<-bl.stopCh

	bl.logger.Info("Block listener stopped")
}
