package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

// BlockStateManager manages block processing state, gap detection, and chain reorganization detection
type BlockStateManager struct {
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
	lastProcessedHeight, err := bsm.getLastProcessedHeight()
	if err != nil {
		bsm.log.Warnf("Failed to get last processed height: %v", err)
		// continue processing but log the warning
		return nil
	}

	if lastProcessedHeight == 0 {
		// first block, no validation needed
		bsm.log.Debugf("Processing first block at height %d", block.Number())
		return nil
	}

	// check for block sequence gaps
	expectedHeight := lastProcessedHeight + 1
	if block.Number() != expectedHeight {
		errorMsg := fmt.Sprintf("expected height %d, got %d", expectedHeight, block.Number())
		bsm.log.Errorf("Block gap detected: %s", errorMsg)
		return ErrBlockSequence(errorMsg)
	}

	// check for chain reorganization by comparing parent hash with last processed block
	lastState, err := bsm.readBlockProcessingState()
	if err == nil && lastState.Status == types.ProcessingStatusCompleted {
		if block.ParentHash() != lastState.Hash {
			errorMsg := fmt.Sprintf("parent hash mismatch at height %d: expected %s, got %s",
				block.Number(), lastState.Hash, block.ParentHash())
			bsm.log.Errorf("Chain reorganization detected: %s", errorMsg)
			return ErrChainReorg(errorMsg)
		}
		bsm.log.Debugf("Chain continuity verified: block %d parent hash matches previous block hash", block.Number())
	}

	bsm.log.Debugf("Block sequence verified: processing block %d after %d", block.Number(), lastProcessedHeight)
	return nil
}

// BeginProcessing marks a block as being processed (phase 1 of two-phase commit)
func (bsm *BlockStateManager) BeginProcessing(block types.BlockI) lib.ErrorI {
	return bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), types.ProcessingStatusProcessing)
}

// CompleteProcessing marks a block as successfully processed and updates last processed height (phase 2 of two-phase commit)
func (bsm *BlockStateManager) CompleteProcessing(block types.BlockI) lib.ErrorI {
	// mark block processing as completed
	if err := bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), types.ProcessingStatusCompleted); err != nil {
		return err
	}
	// save last processed height
	return bsm.saveLastProcessedHeight(block.Number())
}

// FailProcessing marks a block as failed processing
func (bsm *BlockStateManager) FailProcessing(block types.BlockI) lib.ErrorI {
	return bsm.saveBlockProcessingState(block.Number(), block.Hash(), block.ParentHash(), types.ProcessingStatusFailed)
}

// GetStartingHeight determines the height to start processing from based on saved state
func (bsm *BlockStateManager) GetStartingHeight() (uint64, lib.ErrorI) {
	// check for incomplete block processing state from previous run
	if processingState, err := bsm.readBlockProcessingState(); err == nil {
		bsm.log.Infof("Found block processing state: height %d, status %s",
			processingState.Height, processingState.Status)
		switch processingState.Status {
		case types.ProcessingStatusProcessing:
			// block was being processed when oracle stopped, retry it
			bsm.log.Warnf("Block %d was being processed when oracle stopped, will retry processing",
				processingState.Height)
			if processingState.Height > 0 {
				return processingState.Height - 1, nil
			}
			return 0, nil
		case types.ProcessingStatusCompleted:
			// block was completed successfully, start from next block
			return processingState.Height, nil
		case types.ProcessingStatusFailed:
			// previous block failed, retry it
			bsm.log.Warnf("Block %d failed processing, will retry", processingState.Height)
			if processingState.Height > 0 {
				return processingState.Height - 1, nil
			}
			return 0, nil
		}
	}

	// no processing state found, check last processed height
	if lastProcessedHeight, err := bsm.getLastProcessedHeight(); err == nil && lastProcessedHeight > 0 {
		bsm.log.Infof("Starting from last processed height: %d", lastProcessedHeight)
		return lastProcessedHeight, nil
	}

	bsm.log.Infof("No previous state found, starting from height 0")
	return 0, nil
}

// saveBlockProcessingState saves the block processing state to disk
func (bsm *BlockStateManager) saveBlockProcessingState(height uint64, hash string, parentHash string, status types.ProcessingStatus) lib.ErrorI {
	// create processing state file path by appending .state to the main state file
	stateFile := bsm.stateSaveFile + ".state"
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
		if status == types.ProcessingStatusFailed {
			state.RetryCount++
		}
	}
	// marshal state to JSON
	stateBytes, err := json.Marshal(state)
	if err != nil {
		bsm.log.Errorf("Failed to marshal block processing state: %v", err)
		return ErrWriteHeightFile(err)
	}
	// bsm.log.Infof("Saved block processing state height %d: %s", state.Height, state.Status)
	// write state to file atomically
	return bsm.atomicWriteFile(stateFile, stateBytes)
}

// readBlockProcessingState reads the block processing state from disk
func (bsm *BlockStateManager) readBlockProcessingState() (*types.BlockProcessingState, lib.ErrorI) {
	// create processing state file path by appending .state to the main state file
	stateFile := bsm.stateSaveFile + ".state"
	// read file contents
	data, err := os.ReadFile(stateFile)
	if err != nil {
		bsm.log.Infof("Block processing state file not found: %v", err)
		return nil, ErrParseHeight(err)
	}
	// unmarshal JSON data
	var state types.BlockProcessingState
	err = json.Unmarshal(data, &state)
	if err != nil {
		bsm.log.Errorf("Failed to unmarshal block processing state: %v", err)
		return nil, ErrParseHeight(err)
	}
	return &state, nil
}

// getLastProcessedHeight returns the height of the last successfully processed block
func (bsm *BlockStateManager) getLastProcessedHeight() (uint64, lib.ErrorI) {
	// create last processed height file path
	lastProcessedFile := bsm.stateSaveFile + ".last"
	// read file contents
	data, err := os.ReadFile(lastProcessedFile)
	if err != nil {
		if os.IsNotExist(err) {
			// no last processed height file means we haven't processed any blocks yet
			return 0, nil
		}
		return 0, ErrParseHeight(err)
	}
	// parse height from file contents
	heightStr := strings.TrimSpace(string(data))
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		return 0, ErrParseHeight(err)
	}
	return height, nil
}

// saveLastProcessedHeight saves the height of the last successfully processed block
func (bsm *BlockStateManager) saveLastProcessedHeight(height uint64) lib.ErrorI {
	// create last processed height file path
	lastProcessedFile := bsm.stateSaveFile + ".last"
	// convert height to string
	heightStr := strconv.FormatUint(height, 10)
	// write height to file atomically
	return bsm.atomicWriteFile(lastProcessedFile, []byte(heightStr))
}

// atomicWriteFile writes data to a file atomically using write-and-move pattern
func (bsm *BlockStateManager) atomicWriteFile(filePath string, data []byte) lib.ErrorI {
	// create temporary file in the same directory as the target file
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_oracle_state_*")
	if err != nil {
		bsm.log.Errorf("Failed to create temporary file: %v", err)
		return ErrWriteHeightFile(err)
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
		return ErrWriteHeightFile(err)
	}
	// sync to ensure data is written to disk
	err = tempFile.Sync()
	if err != nil {
		bsm.log.Errorf("Failed to sync temporary file: %v", err)
		return ErrWriteHeightFile(err)
	}
	// close temporary file before rename
	err = tempFile.Close()
	if err != nil {
		bsm.log.Errorf("Failed to close temporary file: %v", err)
		return ErrWriteHeightFile(err)
	}
	// atomically move temporary file to final destination
	err = os.Rename(tempFilePath, filePath)
	if err != nil {
		bsm.log.Errorf("Failed to rename temporary file to final destination: %v", err)
		return ErrWriteHeightFile(err)
	}
	return nil
}
