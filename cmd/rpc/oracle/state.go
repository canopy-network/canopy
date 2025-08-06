package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

// OracleBlockState represents the simple state of the last processed block
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
	// submissionHistory tracks orders that have been submitted at specific root heights to prevent duplicate submissions
	submissionHistory map[string]map[uint64]bool
	// lockOrderSubmissions tracks the root height when each lock order ID was first successfully submitted
	lockOrderSubmissions map[string]uint64
	// logger for state management operations
	log lib.LoggerI
}

// NewOracleState creates a new OracleState instance
func NewOracleState(stateSaveFile string, logger lib.LoggerI) *OracleState {
	return &OracleState{
		stateSaveFile:        stateSaveFile,
		log:                  logger,
		submissionHistory:    make(map[string]map[uint64]bool),
		lockOrderSubmissions: make(map[string]uint64),
	}
}

// shouldSubmit determines if the current oracle state allows for submitting this order
// Performs all submission checks including lead time, resubmit delay, lock order restrictions, and history tracking
func (m *OracleState) shouldSubmit(order *types.WitnessedOrder, rootHeight uint64, config lib.OracleConfig) bool {
	// convert order ID to string for use as map key
	orderIdStr := lib.BytesToString(order.OrderId)
	// CHECK 1: Propose lead time validation
	if m.sourceChainHeight < order.WitnessedHeight+config.ProposeLeadTime {
		m.log.Warnf("Propose lead time has not passed, not submitting order %s", order.OrderId)
		return false
	}
	// CHECK 2: Resubmit delay validation
	if rootHeight <= order.LastSubmitHeight+config.OrderResubmitDelay {
		m.log.Warnf("Block resubmit height has not passed, not submitting order %s", order.OrderId)
		return false
	}
	// CHECK 3: Lock order specific time restrictions
	if order.LockOrder != nil {
		// check if this lock order was previously submitted
		if submittedHeight, exists := m.lockOrderSubmissions[orderIdStr]; exists {
			// calculate blocks since last submission
			blocksSinceSubmission := rootHeight - submittedHeight
			// check if enough time has passed
			if blocksSinceSubmission < config.LockOrderHoldTime {
				m.log.Debugf("Lock order %s submitted at height %d, only %d blocks ago (need %d), not allowing resubmission",
					orderIdStr, submittedHeight, blocksSinceSubmission, config.LockOrderHoldTime)
				return false
			}
			m.log.Debugf("Lock order %s submitted at height %d, %d blocks ago, allowing resubmission",
				orderIdStr, submittedHeight, blocksSinceSubmission)
		}
		// record the submission height for this lock order
		m.lockOrderSubmissions[orderIdStr] = rootHeight
	}
	// CHECK 4: General submission history tracking
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

// ValidateSequence performs comprehensive block validation including gap detection and reorg detection
func (m *OracleState) ValidateSequence(block types.BlockI) lib.ErrorI {
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
	m.log.Debugf("Block sequence verified: processing block %d after %d", block.Number(), lastState.Height)
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
	return m.atomicWriteFile(m.stateSaveFile, stateBytes)
}

// GetLastHeight returns the last processed source chain height
func (m *OracleState) GetLastHeight() (uint64, lib.ErrorI) {
	// check for previous state from last run
	if state, err := m.readBlockState(); err == nil {
		m.log.Infof("Found previous block state: height %d", state.Height)
		// start from the next block after the last successfully processed one
		return state.Height, nil
	}
	m.log.Infof("No previous state found, returning start height 0")
	return 0, nil
}

// readBlockState reads the simple block state from disk
func (m *OracleState) readBlockState() (*OracleBlockState, lib.ErrorI) {
	// read file contents
	data, err := os.ReadFile(m.stateSaveFile)
	if err != nil {
		m.log.Debugf("Block state file not found: %v", err)
		return nil, ErrReadStateFile(err)
	}
	// unmarshal JSON data
	var state OracleBlockState
	err = json.Unmarshal(data, &state)
	if err != nil {
		m.log.Errorf("Failed to unmarshal block state: %v", err)
		return nil, ErrParseState(err)
	}
	return &state, nil
}

// atomicWriteFile writes data to a file atomically using write-and-move pattern
func (m *OracleState) atomicWriteFile(filePath string, data []byte) lib.ErrorI {
	// create temporary file in the same directory as the target file
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_oracle_state_*")
	if err != nil {
		m.log.Errorf("Failed to create temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	tempFilePath := tempFile.Name()
	// ensure temporary file is cleaned up if something goes wrong
	defer func() {
		tempFile.Close()
		os.Remove(tempFilePath)
	}()
	// write data to temporary file
	_, err = tempFile.Write(data)
	if err != nil {
		m.log.Errorf("Failed to write to temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// sync to ensure data is written to disk
	err = tempFile.Sync()
	if err != nil {
		m.log.Errorf("Failed to sync temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// close temporary file before rename
	err = tempFile.Close()
	if err != nil {
		m.log.Errorf("Failed to close temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// atomically move temporary file to final destination
	err = os.Rename(tempFilePath, filePath)
	if err != nil {
		m.log.Errorf("Failed to rename temporary file to final destination: %v", err)
		return ErrWriteStateFile(err)
	}
	return nil
}
