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
	// closeOrderSubmissions tracks the root height when each close order was submitted
	closeOrderSubmissions map[string]uint64
	// lockOrderSubmissions tracks the root height when each lock order was submitted
	lockOrderSubmissions map[string]uint64
	// safeHeight is the highest block height that has received sufficient confirmations to be considered safe
	safeHeight uint64
	// blockState is the in-memory copy of the last processed block state (mirrors what is on disk)
	blockState *OracleBlockState
	// rwLock protects all oracle state fields from concurrent access
	rwLock sync.RWMutex
	// logger for state management operations
	log lib.LoggerI
}

// NewOracleState creates a new OracleState instance
func NewOracleState(stateSaveFile string, logger lib.LoggerI) *OracleState {
	// Ensure the state save file location exists
	if err := os.MkdirAll(filepath.Dir(stateSaveFile), 0755); err != nil {
		logger.Errorf("[ORACLE-STATE] failed to create directories for %s: %v", stateSaveFile, err)
	}
	s := &OracleState{
		stateSaveFile:         stateSaveFile,
		lockOrderSubmissions:  make(map[string]uint64),
		closeOrderSubmissions: make(map[string]uint64),
		rwLock:                sync.RWMutex{},
		log:                   logger,
	}
	// load any persisted block state from disk once at startup, so the hot path
	// never performs synchronous file I/O under lock (see ValidateSequence/GetLastHeight)
	if state, err := s.readBlockState(); err == nil {
		s.blockState = state
	} else if _, statErr := os.Stat(stateSaveFile); statErr == nil {
		// the file exists but couldn't be read/parsed - that's data loss on restart, not a clean
		// first run, so surface it loudly instead of silently resuming from height 0
		logger.Errorf("[ORACLE-STATE] failed to load existing block state from %s: %v (resuming from height 0)", stateSaveFile, err)
	}
	return s
}

// shouldSubmit determines if the current oracle state allows for submitting this order.
// It performs all submission checks (propose lead time, resubmit delay, lock order cooldown) but is
// READ-ONLY: it records nothing. Submission history is advanced only when an order actually commits
// (see Oracle.CommitCertificate -> recordSubmission), so building a proposal that never commits does
// not throttle a later, legitimate resubmission and does not diverge state from non-proposer nodes.
func (m *OracleState) shouldSubmit(order *types.WitnessedOrder, rootHeight uint64, config lib.OracleConfig) bool {
	// read-only: guard the state fields with a read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	// convert order ID to string for use as map key
	orderIdStr := lib.BytesToString(order.OrderId)
	// propose lead time validation check
	if m.sourceChainHeight < order.WitnessedHeight+config.ProposeDelayBlocks {
		blocksNeeded := (order.WitnessedHeight + config.ProposeDelayBlocks) - m.sourceChainHeight
		m.log.Debugf("[ORACLE-STATE] Order %s held: propose delay (witnessed=%d, need %d more source chain blocks)",
			orderIdStr, order.WitnessedHeight, blocksNeeded)
		return false
	}
	// resubmit delay check
	if rootHeight <= order.LastSubmitHeight+config.OrderResubmitDelayBlocks {
		eligibleAt := order.LastSubmitHeight + config.OrderResubmitDelayBlocks + 1
		m.log.Debugf("[ORACLE-STATE] Order %s held: resubmit delay (lastSubmit=%d, eligible at rootHeight=%d, current=%d)",
			orderIdStr, order.LastSubmitHeight, eligibleAt, rootHeight)
		return false
	}
	// lock order specific time restrictions
	if order.LockOrder != nil {
		// check if this lock order was previously submitted
		if height, exists := m.lockOrderSubmissions[orderIdStr]; exists {
			// test if already submitted at this root height
			if height == rootHeight {
				m.log.Debugf("[ORACLE-STATE] Lock order %s held: already submitted at rootHeight=%d", orderIdStr, rootHeight)
				return false
			}
			// calculate blocks since last submission
			blocksSinceSubmission := rootHeight - height
			// check if enough time has passed
			if blocksSinceSubmission < config.LockOrderCooldownBlocks {
				blocksNeeded := config.LockOrderCooldownBlocks - blocksSinceSubmission
				m.log.Debugf("[ORACLE-STATE] Lock order %s held: cooldown (lastSubmit=%d, need %d more blocks)",
					orderIdStr, height, blocksNeeded)
				return false
			}
			m.log.Debugf("[ORACLE-STATE] Lock order %s submitted at height %d, %d blocks ago, allowing resubmission", orderIdStr, height, blocksSinceSubmission)
		}
	} else if order.CloseOrder != nil {
		if height, exists := m.closeOrderSubmissions[orderIdStr]; exists {
			// test if already submitted at this root height
			if height == rootHeight {
				m.log.Debugf("[ORACLE-STATE] Close order %s held: already submitted at rootHeight=%d", orderIdStr, rootHeight)
				return false
			}
		}
	}
	return true
}

// recordSubmission advances the local submission history for an order that has actually committed.
// This is the sole writer of the lock/close submission maps; it is driven by CommitCertificate so
// resubmit-delay and lock-order cooldown throttle future retries based on committed submissions
// rather than on speculative proposals that may never reach quorum.
func (m *OracleState) recordSubmission(orderID []byte, orderType types.OrderType, rootHeight uint64) {
	if len(orderID) == 0 {
		return
	}
	orderIdStr := lib.BytesToString(orderID)
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	switch orderType {
	case types.LockOrderType:
		m.lockOrderSubmissions[orderIdStr] = rootHeight
	case types.CloseOrderType:
		m.closeOrderSubmissions[orderIdStr] = rootHeight
	}
}

// ValidateSequence performs sequence validation and reorg detection
func (m *OracleState) ValidateSequence(block types.BlockI) lib.ErrorI {
	// protect oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// verify sequential block processing to detect gaps and chain reorganizations
	lastState := m.blockState
	if lastState == nil {
		m.log.Debugf("[ORACLE-STATE] No previous state found, assuming first block")
		// first block, no validation needed
		m.sourceChainHeight = block.Number()
		return nil
	}
	// note: sourceChainHeight is deliberately NOT advanced on the gap/reorg error returns below -
	// a rejected block must not move the height forward; it only advances on accepted blocks.
	// check for block sequence gaps
	expectedHeight := lastState.Height + 1
	if block.Number() != expectedHeight {
		errorMsg := fmt.Sprintf("expected height %d, got %d", expectedHeight, block.Number())
		m.log.Errorf("[ORACLE-STATE] Block gap detected: %s", errorMsg)
		return ErrBlockSequence(errorMsg)
	}
	// check for chain reorganization by comparing parent hash with last processed block
	if block.ParentHash() != lastState.Hash {
		errorMsg := fmt.Sprintf("parent hash mismatch at height %d: expected %s, got %s",
			block.Number(), lastState.Hash, block.ParentHash())
		m.log.Errorf("[ORACLE-STATE] Chain reorganization detected: %s", errorMsg)
		return ErrChainReorg(errorMsg)
	}
	// save last seen source chain height
	m.sourceChainHeight = block.Number()
	return nil
}

// saveState saves the state after a block has been successfully processed
func (m *OracleState) saveState(block types.BlockI) lib.ErrorI {
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
		m.log.Errorf("[ORACLE-STATE] Failed to marshal block state: %v", err)
		return ErrWriteStateFile(err)
	}
	// m.log.Debugf("Saved block state for height %d", state.Height)
	// write state to file atomically
	if err := lib.AtomicWriteFile(m.stateSaveFile, stateBytes); err != nil {
		m.log.Errorf("[ORACLE-STATE] Failed to write state file: %v", err)
		return ErrWriteStateFile(err)
	}
	// update the in-memory cache to mirror what was just persisted
	m.rwLock.Lock()
	m.blockState = &state
	m.rwLock.Unlock()
	return nil
}

// removeState resets the oracle to a fresh, first-run state: it deletes the on-disk block state
// and clears ALL in-memory derived height state. Both callers (the block-sequence-gap and the
// chain-reorg restart paths) re-sync the block provider from scratch, so any retained height would
// be stale.
//
// safeHeight in particular must be cleared here: it is monotonic (updateSafeHeight only raises it),
// so if left alone after a reorg it stays frozen at the pre-reorg value (~tip - confirmations) for
// the entire ~StartupBlockDepth re-sync. During that window it would claim blocks are safe that
// this pass has not re-processed - most dangerously on the reset path, which gates purely on
// safeHeight vs the deadline with no witnessed evidence. Resetting to 0 is conservative: it holds
// all orders until the first re-processed block re-derives safeHeight from the current pass.
func (m *OracleState) removeState() lib.ErrorI {
	m.rwLock.Lock()
	m.blockState = nil
	m.safeHeight = 0
	m.sourceChainHeight = 0
	m.rwLock.Unlock()
	err := os.Remove(m.stateSaveFile)
	if err != nil && !os.IsNotExist(err) {
		return ErrWriteStateFile(err)
	}
	return nil
}

// GetLastHeight returns the last processed source chain height
func (m *OracleState) GetLastHeight() uint64 {
	// protect oracle state fields with read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	// check for previous state from last run (loaded into cache at startup / updated on save)
	if m.blockState != nil {
		m.log.Debugf("[ORACLE-STATE] Found previous block state: height %d", m.blockState.Height)
		// start from the next block after the last successfully processed one
		return m.blockState.Height
	}
	m.log.Debugf("[ORACLE-STATE] no previous state found, returning start height 0")
	return 0
}

// readBlockState reads the oracle state from disk
func (m *OracleState) readBlockState() (*OracleBlockState, lib.ErrorI) {
	// read file contents
	data, err := os.ReadFile(m.stateSaveFile)
	if err != nil {
		m.log.Debugf("[ORACLE-STATE] block state file not found: %v", err)
		return nil, ErrReadStateFile(err)
	}
	// unmarshal JSON data
	var state OracleBlockState
	err = json.Unmarshal(data, &state)
	if err != nil {
		m.log.Errorf("[ORACLE-STATE] failed to unmarshal block state: %v", err)
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
		m.log.Debugf("[ORACLE-STATE] Updating safe height from %d to %d (current height %d, confirmations %d)",
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

// GetSourceChainHeight returns the last seen source chain height
func (m *OracleState) GetSourceChainHeight() uint64 {
	// protect oracle state fields with read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	return m.sourceChainHeight
}

// SubmissionCounts returns the number of tracked lock and close order submissions
func (m *OracleState) SubmissionCounts() (lockCount, closeCount int) {
	// protect oracle state fields with read lock
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	return len(m.lockOrderSubmissions), len(m.closeOrderSubmissions)
}

// PruneHistory removes submission history for orders that are not present in the passed order book
func (m *OracleState) PruneHistory(orderBook *lib.OrderBook) {
	// protect oracle state fields with write lock
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	// handle nil order book case by clearing all history
	if orderBook == nil {
		m.lockOrderSubmissions = make(map[string]uint64)
		m.closeOrderSubmissions = make(map[string]uint64)
		m.log.Infof("[ORACLE-STATE] Order book is nil, cleared all submission history")
		return
	}
	// prune lock order submissions for orders not in order book
	for orderIdStr := range m.lockOrderSubmissions {
		// convert string back to bytes for order book lookup
		orderId, err := lib.StringToBytes(orderIdStr)
		if err != nil {
			m.log.Errorf("[ORACLE-STATE] Failed to convert lock order ID string %s to bytes: %v", orderIdStr, err)
			continue
		}
		// check if order exists in order book
		order, err := orderBook.GetOrder(orderId)
		if err != nil {
			m.log.Errorf("[ORACLE-STATE] Error checking lock order %s in order book: %v", orderIdStr, err)
			continue
		}
		// remove lock order submission for orders not in order book
		if order == nil {
			delete(m.lockOrderSubmissions, orderIdStr)
			m.log.Debugf("[ORACLE-STATE] Pruned lock order submission for order %s (not in order book)", orderIdStr)
		} else {
			m.log.Debugf("[ORACLE-STATE] Not pruning lock order submission for order %s (still in order book)", orderIdStr)
		}
	}
	// prune close order submissions for orders not in order book
	for orderIdStr := range m.closeOrderSubmissions {
		// convert string back to bytes for order book lookup
		orderId, err := lib.StringToBytes(orderIdStr)
		if err != nil {
			m.log.Errorf("[ORACLE-STATE] Failed to convert close order ID string %s to bytes: %v", orderIdStr, err)
			continue
		}
		// check if order exists in order book
		order, err := orderBook.GetOrder(orderId)
		if err != nil {
			m.log.Errorf("[ORACLE-STATE] Error checking close order %s in order book: %v", orderIdStr, err)
			continue
		}
		// remove close order submission for orders not in order book
		if order == nil {
			delete(m.closeOrderSubmissions, orderIdStr)
			m.log.Debugf("[ORACLE-STATE] Pruned close order submission for order %s (not in order book)", orderIdStr)
		} else {
			m.log.Debugf("[ORACLE-STATE] Not pruning close order submission for order %s (still in order book)", orderIdStr)
		}
	}
}

// ClearOrderHistory removes submission history for a specific order ID.
func (m *OracleState) ClearOrderHistory(orderID []byte) {
	if len(orderID) == 0 {
		return
	}
	orderIDStr := lib.BytesToString(orderID)
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	delete(m.lockOrderSubmissions, orderIDStr)
	delete(m.closeOrderSubmissions, orderIDStr)
}
