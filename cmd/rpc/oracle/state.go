package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

// OracleBlockState represents the last processed block
type OracleBlockState struct {
	// Height is the last successfully processed block height
	Height uint64 `json:"height"`
	// Hash is the block hash for verification
	Hash string `json:"hash"`
	// ParentHash is the parent block hash for chain reorganization detection
	ParentHash string `json:"parentHash"`
	// Timestamp when the block was processed
	Timestamp time.Time `json:"timestamp"`
}

// OracleState manages block processing state, gap detection, and chain reorganization detection
type OracleState struct {
	// sourceChainHeight is the last seen height for the source chain
	sourceChainHeight uint64
	// stateSaveFile is the base path for state files
	stateSaveFile string
	// submissionHistory tracks orders that have been submitted at specific root heights
	submissionHistory map[string]map[uint64]bool
	// lockOrderSubmissions tracks the root height when each lock order ID was first successfully submitted
	lockOrderSubmissions map[string]uint64
	// safeHeight is the highest block height that has received sufficient confirmations to be considered safe
	safeHeight uint64
	// rwLock protects all oracle state fields from concurrent access
	rwLock sync.RWMutex
	// logger for state management operations
	log lib.LoggerI
}

// NewOracleState creates a new OracleState instance
func NewOracleState(stateSaveFile string, logger lib.LoggerI) *OracleState {
	// Ensure the state save file location exists
	if err := os.MkdirAll(filepath.Dir(stateSaveFile), 0755); err != nil {
		logger.Errorf("failed to create directories for %s: %w", stateSaveFile, err)
	}
	return &OracleState{
		stateSaveFile:        stateSaveFile,
		submissionHistory:    make(map[string]map[uint64]bool),
		lockOrderSubmissions: make(map[string]uint64),
		rwLock:               sync.RWMutex{},
		log:                  logger,
	}
}

// shouldSubmit determines if the current oracle state allows for submitting this order
// Performs all submission checks including lead time, resubmit delay, lock order restrictions, and history tracking
func (m *OracleState) shouldSubmit(order *types.WitnessedOrder, rootHeight uint64, config lib.OracleConfig) bool {
	// protect all oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// convert order ID to string for use as map key
	orderIdStr := lib.BytesToString(order.OrderId)
	// propose lead time validation check
	if m.sourceChainHeight < order.WitnessedHeight+config.ProposeLeadBlocks {
		m.log.Warnf("Propose lead time has not passed, not submitting order %s", order.OrderId)
		return false
	}
	// resubmit delay check
	if rootHeight <= order.LastSubmitHeight+config.OrderResubmitDelay {
		m.log.Warnf("Block resubmit height has not passed, not submitting order %s", order.OrderId)
		return false
	}
	// lock order specific time restrictions
	if order.LockOrder != nil {
		// check if this lock order was previously submitted
		if submittedHeight, exists := m.lockOrderSubmissions[orderIdStr]; exists {
			// calculate blocks since last submission
			blocksSinceSubmission := rootHeight - submittedHeight
			// check if enough time has passed
			if blocksSinceSubmission < config.LockOrderHoldBlocks {
				m.log.Debugf("Lock order %s submitted at height %d, only %d blocks ago (need %d), not allowing resubmission",
					orderIdStr, submittedHeight, blocksSinceSubmission, config.LockOrderHoldBlocks)
				return false
			}
			m.log.Debugf("Lock order %s submitted at height %d, %d blocks ago, allowing resubmission",
				orderIdStr, submittedHeight, blocksSinceSubmission)
		}
		// record the submission height for this lock order
		m.lockOrderSubmissions[orderIdStr] = rootHeight
	}
	// check if we have submission history for this order
	if orderHeights, exists := m.submissionHistory[orderIdStr]; exists {
		// check if this order was already submitted at this root height
		if orderHeights[rootHeight] {
			m.log.Debugf("Order %s already submitted at root height %d", orderIdStr, rootHeight)
			return false
		}
	} else {
		// initialize submission history for this order
		m.submissionHistory[orderIdStr] = make(map[uint64]bool)
	}
	// record that we are submitting this order at this root height
	m.submissionHistory[orderIdStr][rootHeight] = true
	m.log.Debugf("Allowing submission of order %s at root height %d", orderIdStr, rootHeight)
	return true
}

// ValidateSequence performs sequence validation and reorg detection
func (m *OracleState) ValidateSequence(block types.BlockI) lib.ErrorI {
	// protect oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// verify sequential block processing to detect gaps and chain reorganizations
	lastState, err := m.readBlockState()
	if err != nil {
		m.log.Debugf("No previous state found, assuming first block")
		// first block, no validation needed
		return nil
	}
	// check for block sequence gaps
	expectedHeight := lastState.Height + 1
	if block.Number() != expectedHeight {
		errorMsg := fmt.Sprintf("expected height %d, got %d", expectedHeight, block.Number())
		m.log.Errorf("Block gap detected: %s", errorMsg)
		return ErrBlockSequence(errorMsg)
	}
	// check for chain reorganization by comparing parent hash with last processed block
	if block.ParentHash() != lastState.Hash {
		errorMsg := fmt.Sprintf("parent hash mismatch at height %d: expected %s, got %s",
			block.Number(), lastState.Hash, block.ParentHash())
		m.log.Errorf("Chain reorganization detected: %s", errorMsg)
		return ErrChainReorg(errorMsg)
	}
	// save last seen source chain height
	m.sourceChainHeight = block.Number()
	return nil
}

// SaveProcessedBlock saves the state after a block has been successfully processed
func (m *OracleState) SaveProcessedBlock(block types.BlockI) lib.ErrorI {
	// create the simple block state
	state := OracleBlockState{
		Height:     block.Number(),
		Hash:       block.Hash(),
		ParentHash: block.ParentHash(),
		Timestamp:  time.Now(),
	}
	// marshal state to JSON
	stateBytes, err := json.Marshal(state)
	if err != nil {
		m.log.Errorf("Failed to marshal block state: %v", err)
		return ErrWriteStateFile(err)
	}
	// m.log.Debugf("Saved block state for height %d", state.Height)
	// write state to file atomically
	if err := lib.AtomicWriteFile(m.stateSaveFile, stateBytes); err != nil {
		m.log.Errorf("Failed to write state file: %v", err)
		return ErrWriteStateFile(err)
	}
	return nil
}

// GetLastHeight returns the last processed source chain height
func (m *OracleState) GetLastHeight() uint64 {
	// protect oracle state fields with read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	// check for previous state from last run
	if state, err := m.readBlockState(); err == nil {
		m.log.Infof("Found previous block state: height %d", state.Height)
		// start from the next block after the last successfully processed one
		return state.Height
	}
	m.log.Infof("no previous state found, returning start height 0")
	return 0
}

// readBlockState reads the oracle state from disk
func (m *OracleState) readBlockState() (*OracleBlockState, lib.ErrorI) {
	// read file contents
	data, err := os.ReadFile(m.stateSaveFile)
	if err != nil {
		m.log.Debugf("block state file not found: %v", err)
		return nil, ErrReadStateFile(err)
	}
	// unmarshal JSON data
	var state OracleBlockState
	err = json.Unmarshal(data, &state)
	if err != nil {
		m.log.Errorf("failed to unmarshal block state: %v", err)
		return nil, ErrParseState(err)
	}
	return &state, nil
}

// updateSafeHeight calculates and updates the safe block height with monotonic guarantee
// The safe height can only increase, never decrease, providing stability during reorgs
func (m *OracleState) updateSafeHeight(currentBlockHeight uint64, config lib.OracleConfig) {
	// calculate new safe height by subtracting required confirmations
	var newSafeHeight uint64
	if currentBlockHeight > config.SafeBlockConfirmations {
		newSafeHeight = currentBlockHeight - config.SafeBlockConfirmations
	} else {
		// handle startup case where current height is less than required confirmations
		newSafeHeight = 0
	}
	// protect oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// only update if the new safe height is higher (monotonic property)
	if newSafeHeight > m.safeHeight {
		m.log.Debugf("Updating safe height from %d to %d (current height %d, confirmations %d)",
			m.safeHeight, newSafeHeight, currentBlockHeight, config.SafeBlockConfirmations)
		m.safeHeight = newSafeHeight
	}
}

// GetSafeHeight returns the current safe block height
func (m *OracleState) GetSafeHeight() uint64 {
	// protect oracle state fields with read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	return m.safeHeight
}

// PruneHistory removes submission history for orders that are not present in the passed order book
func (m *OracleState) PruneHistory(orderBook *lib.OrderBook) {
	// protect oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// handle nil order book case by clearing all history
	if orderBook == nil {
		m.submissionHistory = make(map[string]map[uint64]bool)
		m.lockOrderSubmissions = make(map[string]uint64)
		m.log.Infof("Order book is nil, cleared all submission history")
		return
	}
	// prune submission history for orders not in order book
	for orderIdStr := range m.submissionHistory {
		// convert string back to bytes for order book lookup
		orderId, err := lib.StringToBytes(orderIdStr)
		if err != nil {
			m.log.Errorf("Failed to convert order ID string %s to bytes: %v", orderIdStr, err)
			continue
		}
		// check if order exists in order book
		order, err := orderBook.GetOrder(orderId)
		if err != nil {
			m.log.Errorf("Error checking order %s in order book: %v", orderIdStr, err)
			continue
		}
		// remove submission history for orders not in order book
		if order == nil {
			delete(m.submissionHistory, orderIdStr)
			m.log.Debugf("Pruned submission history for order %s (not in order book)", orderIdStr)
		} else {
			m.log.Debugf("Preserved submission history for order %s (still in order book)", orderIdStr)
		}
	}
	// prune lock order submissions for orders not in order book
	for orderIdStr := range m.lockOrderSubmissions {
		// convert string back to bytes for order book lookup
		orderId, err := lib.StringToBytes(orderIdStr)
		if err != nil {
			m.log.Errorf("Failed to convert lock order ID string %s to bytes: %v", orderIdStr, err)
			continue
		}
		// check if order exists in order book
		order, err := orderBook.GetOrder(orderId)
		if err != nil {
			m.log.Errorf("Error checking lock order %s in order book: %v", orderIdStr, err)
			continue
		}
		// remove lock order submission for orders not in order book
		if order == nil {
			delete(m.lockOrderSubmissions, orderIdStr)
			m.log.Debugf("Pruned lock order submission for order %s (not in order book)", orderIdStr)
		} else {
			m.log.Debugf("Preserved lock order submission for order %s (still in order book)", orderIdStr)
		}
	}
}
