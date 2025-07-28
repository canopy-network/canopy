package oracle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestBlock creates a test block with all required fields
func createTestBlock(number uint64, hash string, parentHash string) types.BlockI {
	return &mockBlock{
		number:       number,
		hash:         hash,
		parentHash:   parentHash,
		transactions: []types.TransactionI{},
	}
}

func TestBlockStateManager_ValidateSequence(t *testing.T) {
	// Helper function to create a temporary directory for test state files
	createTempDir := func(t *testing.T) string {
		dir, err := os.MkdirTemp("", "oracle_state_test_*")
		require.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(dir) })
		return dir
	}

	// Helper function to create a state manager with test setup
	createStateManager := func(t *testing.T, tempDir string) *BlockStateManager {
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		return NewBlockStateManager(stateFile, logger)
	}

	// Helper function to setup a completed block state
	setupCompletedBlock := func(t *testing.T, bsm *BlockStateManager, height uint64, hash string, parentHash string) {
		err := bsm.saveBlockProcessingState(height, hash, parentHash, types.ProcessingStatusCompleted)
		require.NoError(t, err)
	}

	tests := []struct {
		name         string
		setupState   func(t *testing.T, bsm *BlockStateManager)
		block        types.BlockI
		expectError  bool
		errorCode    lib.ErrorCode
		errorMessage string
	}{
		{
			name: "first block validation should pass",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				// No setup needed - simulates first run
			},
			block:       createTestBlock(1, "0xblock1", "0xparent1"),
			expectError: false,
		},
		{
			name: "sequential block validation should pass",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:       createTestBlock(2, "0xblock2", "0xblock1"),
			expectError: false,
		},
		{
			name: "block gap should be detected",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(3, "0xblock3", "0xblock2"), // Skipping block 2
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 2, got 3",
		},
		{
			name: "chain reorganization should be detected",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(2, "0xblock2", "0xdifferentparent"), // Wrong parent hash
			expectError:  true,
			errorCode:    CodeChainReorg,
			errorMessage: "parent hash mismatch at height 2: expected 0xblock1, got 0xdifferentparent",
		},
		{
			name: "valid chain continuation after multiple blocks",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:       createTestBlock(6, "0xblock6", "0xblock5"),
			expectError: false,
		},
		{
			name: "gap detection with large height difference",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(100, "0xblock100", "0xblock99"),
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 2, got 100",
		},
		{
			name: "reorganization with correct height but wrong parent",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 10, "0xblock10", "0xblock9")
			},
			block:        createTestBlock(11, "0xblock11", "0xwrongparent"),
			expectError:  true,
			errorCode:    CodeChainReorg,
			errorMessage: "parent hash mismatch at height 11: expected 0xblock10, got 0xwrongparent",
		},
		{
			name: "backward block should be detected as gap",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:        createTestBlock(3, "0xblock3", "0xblock2"), // Going backwards
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 6, got 3",
		},
		{
			name: "same height block should be detected as gap",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:        createTestBlock(5, "0xblock5_alt", "0xblock4"), // Same height, different block
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 6, got 5",
		},
		{
			name: "empty hash values should still work",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "", "")
			},
			block:       createTestBlock(2, "", ""),
			expectError: false,
		},
		{
			name: "reorg detection with empty parent hash",
			setupState: func(t *testing.T, bsm *BlockStateManager) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(2, "0xblock2", ""), // Empty parent hash
			expectError:  true,
			errorCode:    CodeChainReorg,
			errorMessage: "parent hash mismatch at height 2: expected 0xblock1, got ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and state manager for this test
			tempDir := createTempDir(t)
			bsm := createStateManager(t, tempDir)

			// Setup initial state if needed
			if tt.setupState != nil {
				tt.setupState(t, bsm)
			}

			// Execute the test
			err := bsm.ValidateSequence(tt.block)

			// Verify results
			if tt.expectError {
				require.Error(t, err, "expected error but got nil")

				// Check error code if specified
				if tt.errorCode != 0 {
					assert.Equal(t, tt.errorCode, err.Code(), "unexpected error code")
				}

				// Check error message if specified
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage, "error message does not contain expected text")
				}
			} else {
				require.NoError(t, err, "unexpected error: %v", err)
			}
		})
	}
}
