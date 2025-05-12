package rpc

import (
	"context"
	"encoding/json"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

// TransactionStorer defines the interface for storing transactions
type TransactionStorer interface {
	WriteTx(blockHeight uint64, txID string, data string) error
}

// BlockListener listens for new blocks and processes their transactions
type BlockListener struct {
	blockCh <-chan wstypes.BlockI // Channel to receive blocks
	txStore TransactionStorer     // Store for persisting transactions
	logger  lib.LoggerI           // Logger for verbose logging
	ctx     context.Context       // Context for cancellation
	cancel  context.CancelFunc    // Function to cancel the context
}

// NewBlockListener creates a new BlockListener
func NewBlockListener(blockCh <-chan wstypes.BlockI, txStore TransactionStorer, logger lib.LoggerI) *BlockListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &BlockListener{
		blockCh: blockCh,
		txStore: txStore,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins listening for blocks
func (bl *BlockListener) Start() {
	bl.logger.Info("Starting BlockListener")
	go bl.listen()
}

// Stop halts the block listener
func (bl *BlockListener) Stop() {
	bl.logger.Info("Stopping BlockListener")
	bl.cancel()
}

// listen is the main loop that processes incoming blocks
func (bl *BlockListener) listen() {
	for {
		select {
		case <-bl.ctx.Done():
			// Context was cancelled, exit the loop
			bl.logger.Info("BlockListener stopped")
			return
		case block := <-bl.blockCh:
			// Process the received block
			bl.processBlock(block)
		}
	}
}

// processBlock handles a single block and its transactions
func (bl *BlockListener) processBlock(block wstypes.BlockI) {
	blockHeight := block.Number()
	bl.logger.Infof("Processing block #%d with %d transactions", blockHeight, len(block.Transactions()))

	// Iterate through all transactions in the block
	for _, tx := range block.Transactions() {
		bl.processTx(blockHeight, tx)
	}
}

// processTx handles a single transaction
func (bl *BlockListener) processTx(blockHeight uint64, tx wstypes.TransactionI) {
	// Get the from and to addresses
	from := tx.From()
	to := tx.To()

	// Check if from and to addresses are the same
	if from == to {
		bl.logger.Debugf("Found self-transaction: %s", tx.Hash())

		// Try to unmarshal the transaction data into a LockOrder
		var lockOrder lib.LockOrder
		txData := tx.Data()

		err := json.Unmarshal([]byte(txData), &lockOrder)
		if err != nil {
			// Not a valid LockOrder, skip this transaction
			bl.logger.Debugf("Transaction %s is not a valid LockOrder: %v", tx.Hash(), err)
			return
		}

		// Valid LockOrder, persist the transaction
		bl.logger.Infof("Found valid LockOrder in transaction %s", tx.Hash())
		err = bl.txStore.WriteTx(blockHeight, tx.Hash(), string(txData))
		if err != nil {
			bl.logger.Errorf("Failed to write transaction %s to store: %v", tx.Hash(), err)
		} else {
			bl.logger.Infof("Successfully stored transaction %s", tx.Hash())
		}
	}
}
