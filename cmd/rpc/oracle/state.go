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

// ProcessingStatus represents the state of block processing
type ProcessingStatus string

const (
	// ProcessingStatusPending indicates block is queued for processing
	ProcessingStatusPending ProcessingStatus = "pending"
	// ProcessingStatusProcessing indicates block is currently being processed
	ProcessingStatusProcessing ProcessingStatus = "processing"
	// ProcessingStatusCompleted indicates block processing completed successfully
	ProcessingStatusCompleted ProcessingStatus = "completed"
	// ProcessingStatusFailed indicates block processing failed
	ProcessingStatusFailed ProcessingStatus = "failed"
)

// BlockStateManager manages block processing state, gap detection, and chain reorganization detection
type BlockStateManager struct {
	// externalChainHeight is the last seen height for the source chain
	externalChainHeight uint64
	// stateSaveFile is the base path for state files
	stateSaveFile string
	// logger for state management operations
	log lib.LoggerI
}

// NewBlockStateManager creates a new BlockStateManager instance
func NewBlockStateManager(stateSaveFile string, logger lib.LoggerI) *BlockStateManager {
	return &BlockStateManager{
		stateSaveFile: stateSaveFile,
		log:           logger,
	}
}

// ValidateBlock performs comprehensive block validation including gap detection and reorg detection
func (bsm *BlockStateManager) ValidateBlock(block types.BlockI) lib.ErrorI {
	// verify sequential block processing to detect gaps and chain reorganizations
	lastState, err := bsm.readBlockProcessingState()
	if err != nil {
		bsm.log.Debugf("No previous processing state found, assuming first block")
		// first block, no validation needed
		return nil
	}

	if lastState.Status != ProcessingStatusCompleted {
		// last block wasn't completed, no validation needed for retry
		bsm.log.Debugf("Last block wasn't completed, no gap validation needed")
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
	// bsm.log.Debugf("Chain continuity verified: block %d parent hash matches previous block hash", block.Number())

	bsm.log.Debugf("Block sequence verified: processing block %d after %d", block.Number(), lastState.Height)
	// save last seen source chain height
	bsm.externalChainHeight = block.Number()
	return nil
}

// BeginProcessing marks a block as being processed (phase 1 of two-phase commit)
func (bsm *BlockStateManager) BeginProcessing(block types.BlockI) lib.ErrorI {
	return bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), ProcessingStatusProcessing)
}

// CompleteProcessing marks a block as successfully processed (phase 2 of two-phase commit)
func (bsm *BlockStateManager) CompleteProcessing(block types.BlockI) lib.ErrorI {
	// mark block processing as completed
	return bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), ProcessingStatusCompleted)
}

// FailProcessing marks a block as failed processing
func (bsm *BlockStateManager) FailProcessing(block types.BlockI) lib.ErrorI {
	return bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), ProcessingStatusFailed)
}

// GetStartingHeight determines the height to start processing from based on saved state
func (bsm *BlockStateManager) GetStartingHeight() (uint64, lib.ErrorI) {
	// check for incomplete block processing state from previous run
	if state, err := bsm.readBlockProcessingState(); err == nil {
		bsm.log.Infof("Found block processing state: height %d, status %s", state.Height, state.Status)
		// handle the different processing states
		switch state.Status {
		case ProcessingStatusProcessing:
			// block was being processed when oracle stopped, retry it
			bsm.log.Warnf("Block %d was being processed when oracle stopped, will retry processing",
				state.Height)
			if state.Height > 0 {
				return state.Height - 1, nil
			}
			return 0, nil
		case ProcessingStatusCompleted:
			// block was completed successfully, start from next block
			return state.Height, nil
		case ProcessingStatusFailed:
			// previous block failed, retry it
			bsm.log.Warnf("Block %d failed processing, will retry", state.Height)
			if state.Height > 0 {
				return state.Height - 1, nil
			}
			return 0, nil
		}
	}
	bsm.log.Infof("No previous state found, returning start height 0")
	return 0, nil
}

// saveBlockProcessingState saves the block processing state to disk
func (bsm *BlockStateManager) saveBlockProcessingState(height uint64, hash string, parentHash string, status ProcessingStatus) lib.ErrorI {
	// create block processing state struct
	state := types.BlockProcessingState{
		Height:     height,
		Hash:       hash,
		ParentHash: parentHash,
		Status:     status,
		Timestamp:  time.Now(),
	}
	// read existing state to preserve retry count if updating existing block
	if existingState, err := bsm.readBlockProcessingState(); err == nil && existingState.Height == height {
		state.RetryCount = existingState.RetryCount
		if status == ProcessingStatusFailed {
			state.RetryCount++
		}
	}
	// marshal state to JSON
	stateBytes, err := json.Marshal(state)
	if err != nil {
		bsm.log.Errorf("Failed to marshal block processing state: %v", err)
		return ErrWriteStateFile(err)
	}
	// bsm.log.Infof("Saved block processing state height %d: %s", state.Height, state.Status)
	// write state to file atomically
	return bsm.atomicWriteFile(bsm.stateSaveFile, stateBytes)
}

// readBlockProcessingState reads the block processing state from disk
func (bsm *BlockStateManager) readBlockProcessingState() (*types.BlockProcessingState, lib.ErrorI) {
	// read file contents
	data, err := os.ReadFile(bsm.stateSaveFile)
	if err != nil {
		bsm.log.Infof("Block processing state file not found: %v", err)
		return nil, ErrReadStateFile(err)
	}
	// unmarshal JSON data
	var state types.BlockProcessingState
	err = json.Unmarshal(data, &state)
	if err != nil {
		bsm.log.Errorf("Failed to unmarshal block processing state: %v", err)
		return nil, ErrParseState(err)
	}
	return &state, nil
}

// atomicWriteFile writes data to a file atomically using write-and-move pattern
func (bsm *BlockStateManager) atomicWriteFile(filePath string, data []byte) lib.ErrorI {
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
