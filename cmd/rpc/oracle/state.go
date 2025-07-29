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

// OracleStateManager manages block processing state, gap detection, and chain reorganization detection
type OracleStateManager struct {
	// externalChainHeight is the last seen height for the source chain
	externalChainHeight uint64
	// stateSaveFile is the base path for state files
	stateSaveFile string
	// logger for state management operations
	log lib.LoggerI
}

// NewOracleStateManager creates a new OracleStateManager instance
func NewOracleStateManager(stateSaveFile string, logger lib.LoggerI) *OracleStateManager {
	return &OracleStateManager{
		stateSaveFile: stateSaveFile,
		log:           logger,
	}
}

// shouldSubmit determines if the current oracle state allows for submitting this order
func (bsm *OracleStateManager) shouldSubmit(order *types.WitnessedOrder, rootHeight uint64) bool {
	// if an order has already been submitted for this rootHeight, return false

	return true
}

// ValidateSequence performs comprehensive block validation including gap detection and reorg detection
func (bsm *OracleStateManager) ValidateSequence(block types.BlockI) lib.ErrorI {
	// verify sequential block processing to detect gaps and chain reorganizations
	lastState, err := bsm.readBlockState()
	if err != nil {
		bsm.log.Debugf("No previous state found, assuming first block")
		// first block, no validation needed
		return nil
	}

	// check for block sequence gaps
	expectedHeight := lastState.Height + 1
	if block.Number() != expectedHeight {
		errorMsg := fmt.Sprintf("expected height %d, got %d", expectedHeight, block.Number())
		bsm.log.Errorf("Block gap detected: %s", errorMsg)
		return ErrBlockSequence(errorMsg)
	}

	// check for chain reorganization by comparing parent hash with last processed block
	if block.ParentHash() != lastState.Hash {
		errorMsg := fmt.Sprintf("parent hash mismatch at height %d: expected %s, got %s",
			block.Number(), lastState.Hash, block.ParentHash())
		bsm.log.Errorf("Chain reorganization detected: %s", errorMsg)
		return ErrChainReorg(errorMsg)
	}

	bsm.log.Debugf("Block sequence verified: processing block %d after %d", block.Number(), lastState.Height)
	// save last seen source chain height
	bsm.externalChainHeight = block.Number()
	return nil
}

// SaveProcessedBlock saves the state after a block has been successfully processed
func (bsm *OracleStateManager) SaveProcessedBlock(block types.BlockI) lib.ErrorI {
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
		bsm.log.Errorf("Failed to marshal block state: %v", err)
		return ErrWriteStateFile(err)
	}
	bsm.log.Debugf("Saved block state for height %d", state.Height)
	// write state to file atomically
	return bsm.atomicWriteFile(bsm.stateSaveFile, stateBytes)
}

// GetStartingHeight determines the height to start processing from based on saved state
func (bsm *OracleStateManager) GetStartingHeight() (uint64, lib.ErrorI) {
	// check for previous state from last run
	if state, err := bsm.readBlockState(); err == nil {
		bsm.log.Infof("Found previous block state: height %d", state.Height)
		// start from the next block after the last successfully processed one
		return state.Height, nil
	}
	bsm.log.Infof("No previous state found, returning start height 0")
	return 0, nil
}

// readBlockState reads the simple block state from disk
func (bsm *OracleStateManager) readBlockState() (*OracleBlockState, lib.ErrorI) {
	// read file contents
	data, err := os.ReadFile(bsm.stateSaveFile)
	if err != nil {
		bsm.log.Debugf("Block state file not found: %v", err)
		return nil, ErrReadStateFile(err)
	}
	// unmarshal JSON data
	var state OracleBlockState
	err = json.Unmarshal(data, &state)
	if err != nil {
		bsm.log.Errorf("Failed to unmarshal block state: %v", err)
		return nil, ErrParseState(err)
	}
	return &state, nil
}

// atomicWriteFile writes data to a file atomically using write-and-move pattern
func (bsm *OracleStateManager) atomicWriteFile(filePath string, data []byte) lib.ErrorI {
	// create temporary file in the same directory as the target file
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_oracle_state_*")
	if err != nil {
		bsm.log.Errorf("Failed to create temporary file: %v", err)
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
		bsm.log.Errorf("Failed to write to temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// sync to ensure data is written to disk
	err = tempFile.Sync()
	if err != nil {
		bsm.log.Errorf("Failed to sync temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// close temporary file before rename
	err = tempFile.Close()
	if err != nil {
		bsm.log.Errorf("Failed to close temporary file: %v", err)
		return ErrWriteStateFile(err)
	}
	// atomically move temporary file to final destination
	err = os.Rename(tempFilePath, filePath)
	if err != nil {
		bsm.log.Errorf("Failed to rename temporary file to final destination: %v", err)
		return ErrWriteStateFile(err)
	}
	return nil
}
