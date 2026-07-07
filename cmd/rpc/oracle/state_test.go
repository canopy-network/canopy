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

func TestOracleState_ValidateSequence(t *testing.T) {
	// Helper function to create a temporary directory for test state files
	createTempDir := func(t *testing.T) string {
		dir, err := os.MkdirTemp("", "oracle_state_test_*")
		require.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(dir) })
		return dir
	}

	// Helper function to create a state manager with test setup
	createStateManager := func(t *testing.T, tempDir string) *OracleState {
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		return NewOracleState(stateFile, logger)
	}

	// Helper function to setup a completed block state
	setupCompletedBlock := func(t *testing.T, bsm *OracleState, height uint64, hash string, parentHash string) {
		block := createTestBlock(height, hash, parentHash)
		err := bsm.saveState(block)
		require.NoError(t, err)
	}

	tests := []struct {
		name         string
		setupState   func(t *testing.T, bsm *OracleState)
		block        types.BlockI
		expectError  bool
		errorCode    lib.ErrorCode
		errorMessage string
	}{
		{
			name: "first block validation should pass",
			setupState: func(t *testing.T, bsm *OracleState) {
				// No setup needed - simulates first run
			},
			block:       createTestBlock(1, "0xblock1", "0xparent1"),
			expectError: false,
		},
		{
			name: "sequential block validation should pass",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:       createTestBlock(2, "0xblock2", "0xblock1"),
			expectError: false,
		},
		{
			name: "block gap should be detected",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(3, "0xblock3", "0xblock2"), // Skipping block 2
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 2, got 3",
		},
		{
			name: "chain reorganization should be detected",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(2, "0xblock2", "0xdifferentparent"), // Wrong parent hash
			expectError:  true,
			errorCode:    CodeChainReorg,
			errorMessage: "parent hash mismatch at height 2: expected 0xblock1, got 0xdifferentparent",
		},
		{
			name: "valid chain continuation after multiple blocks",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:       createTestBlock(6, "0xblock6", "0xblock5"),
			expectError: false,
		},
		{
			name: "gap detection with large height difference",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 1, "0xblock1", "0xparent1")
			},
			block:        createTestBlock(100, "0xblock100", "0xblock99"),
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 2, got 100",
		},
		{
			name: "reorganization with correct height but wrong parent",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 10, "0xblock10", "0xblock9")
			},
			block:        createTestBlock(11, "0xblock11", "0xwrongparent"),
			expectError:  true,
			errorCode:    CodeChainReorg,
			errorMessage: "parent hash mismatch at height 11: expected 0xblock10, got 0xwrongparent",
		},
		{
			name: "backward block should be detected as gap",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:        createTestBlock(3, "0xblock3", "0xblock2"), // Going backwards
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 6, got 3",
		},
		{
			name: "same height block should be detected as gap",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 5, "0xblock5", "0xblock4")
			},
			block:        createTestBlock(5, "0xblock5_alt", "0xblock4"), // Same height, different block
			expectError:  true,
			errorCode:    CodeBlockSequence,
			errorMessage: "expected height 6, got 5",
		},
		{
			name: "empty hash values should still work",
			setupState: func(t *testing.T, bsm *OracleState) {
				setupCompletedBlock(t, bsm, 1, "", "")
			},
			block:       createTestBlock(2, "", ""),
			expectError: false,
		},
		{
			name: "reorg detection with empty parent hash",
			setupState: func(t *testing.T, bsm *OracleState) {
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

func TestOracleState_ValidateSequence_FirstBlockInitializesSourceHeight(t *testing.T) {
	state := NewOracleState(filepath.Join(t.TempDir(), "oracle.state"), lib.NewDefaultLogger())
	block := createTestBlock(42, "0xblock42", "0xblock41")

	require.NoError(t, state.ValidateSequence(block))
	require.Equal(t, uint64(42), state.sourceChainHeight)
}

func TestOracleState_shouldSubmit(t *testing.T) {
	tests := []struct {
		name                     string
		lastSubmitHeight         uint64
		witnessedHeight          uint64
		rootHeight               uint64
		sourceChainHeight        uint64
		orderResubmitDelay       uint64
		proposeLeadTime          uint64
		lockOrderHoldBlocks      uint64
		orderType                string // "lock" or "close" or "none"
		setupPreviousSubmission  bool   // whether to simulate a previous lock order submission
		previousSubmissionHeight uint64
		expected                 bool
	}{
		{
			name:                "propose lead time not passed - should not submit",
			lastSubmitHeight:    40,
			witnessedHeight:     10,
			rootHeight:          60,
			sourceChainHeight:   14, // witnessedHeight(10) + proposeLeadTime(5) = 15, sourceChainHeight(14) < 15
			orderResubmitDelay:  20,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            false,
		},
		{
			name:                "propose lead time exact boundary - should not submit",
			lastSubmitHeight:    40,
			witnessedHeight:     10,
			rootHeight:          60,
			sourceChainHeight:   15, // witnessedHeight(10) + proposeLeadTime(5) = 15, sourceChainHeight(15) >= 15 but still not passed
			orderResubmitDelay:  20,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            false,
		},
		{
			name:                "resubmit delay not reached - should not submit",
			lastSubmitHeight:    40,
			witnessedHeight:     10,
			rootHeight:          55, // 40 + 20 = 60, so 55 <= 60 (delay not reached)
			sourceChainHeight:   16, // witnessedHeight(10) + proposeLeadTime(5) = 15, sourceChainHeight(16) > 15
			orderResubmitDelay:  20,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            false,
		},
		{
			name:                "resubmit delay exact boundary - should not submit",
			lastSubmitHeight:    40,
			witnessedHeight:     10,
			rootHeight:          60, // 40 + 20 = 60, so 60 <= 60 (delay not reached)
			sourceChainHeight:   16,
			orderResubmitDelay:  20,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            false,
		},
		{
			name:                "resubmit delay exceeded - should submit close order",
			lastSubmitHeight:    30,
			witnessedHeight:     10,
			rootHeight:          100, // 30 + 20 = 50, so 100 > 50 (delay exceeded)
			sourceChainHeight:   16,
			orderResubmitDelay:  20,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            true,
		},
		{
			name:                "first submission with all checks passed - should submit",
			lastSubmitHeight:    0,
			witnessedHeight:     10,
			rootHeight:          100,
			sourceChainHeight:   16, // witnessedHeight(10) + proposeLeadTime(5) = 15, sourceChainHeight(16) > 15
			orderResubmitDelay:  10,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "lock",
			expected:            true,
		},
		{
			name:                "zero propose lead time with resubmit delay exceeded - should submit",
			lastSubmitHeight:    40,
			witnessedHeight:     10,
			rootHeight:          80, // 40 + 20 = 60, so 80 > 60 (delay exceeded)
			sourceChainHeight:   11, // witnessedHeight(10) + proposeLeadTime(0) = 10, sourceChainHeight(11) > 10
			orderResubmitDelay:  20,
			proposeLeadTime:     0,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            true,
		},
		{
			name:                "zero delay with propose lead time passed - should submit",
			lastSubmitHeight:    50,
			witnessedHeight:     10,
			rootHeight:          51, // 50 + 0 = 50, so 51 > 50 (delay exceeded)
			sourceChainHeight:   16,
			orderResubmitDelay:  0,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 10,
			orderType:           "close",
			expected:            true,
		},
		{
			name:                "lock order first submission - should submit",
			lastSubmitHeight:    0,
			witnessedHeight:     10,
			rootHeight:          100,
			sourceChainHeight:   16,
			orderResubmitDelay:  10,
			proposeLeadTime:     5,
			lockOrderHoldBlocks: 20,
			orderType:           "lock",
			expected:            true,
		},
		{
			name:                     "lock order resubmission too soon - should not submit",
			lastSubmitHeight:         0,
			witnessedHeight:          10,
			rootHeight:               105,
			sourceChainHeight:        16,
			orderResubmitDelay:       10,
			proposeLeadTime:          5,
			lockOrderHoldBlocks:      20,
			orderType:                "lock",
			setupPreviousSubmission:  true,
			previousSubmissionHeight: 100, // 105 - 100 = 5 blocks, need 20
			expected:                 false,
		},
		{
			name:                     "lock order resubmission after hold time - should submit",
			lastSubmitHeight:         0,
			witnessedHeight:          10,
			rootHeight:               125,
			sourceChainHeight:        16,
			orderResubmitDelay:       10,
			proposeLeadTime:          5,
			lockOrderHoldBlocks:      20,
			orderType:                "lock",
			setupPreviousSubmission:  true,
			previousSubmissionHeight: 100, // 125 - 100 = 25 blocks, need 20, so allowed
			expected:                 true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create state manager with logger
			logger := lib.NewDefaultLogger()
			stateManager := NewOracleState("test_state", logger)
			// Set external chain height for the test
			stateManager.sourceChainHeight = tt.sourceChainHeight
			// Setup previous submission if needed
			if tt.setupPreviousSubmission {
				stateManager.lockOrderSubmissions = make(map[string]uint64)
				orderIdStr := lib.BytesToString([]byte("testorder"))
				stateManager.lockOrderSubmissions[orderIdStr] = tt.previousSubmissionHeight
			}
			// Create config
			config := lib.OracleConfig{
				OrderResubmitDelayBlocks: tt.orderResubmitDelay,
				ProposeDelayBlocks:       tt.proposeLeadTime,
				LockOrderCooldownBlocks:  tt.lockOrderHoldBlocks,
			}
			// Create witnessed order
			order := &types.WitnessedOrder{
				OrderId:          []byte("testorder"),
				LastSubmitHeight: tt.lastSubmitHeight,
				WitnessedHeight:  tt.witnessedHeight,
			}
			// Set order type
			switch tt.orderType {
			case "lock":
				order.LockOrder = &lib.LockOrder{
					OrderId: []byte("testorder"),
				}
			case "close":
				order.CloseOrder = &lib.CloseOrder{
					OrderId: []byte("testorder"),
				}
			}
			// Execute test
			result := stateManager.shouldSubmit(order, tt.rootHeight, config)

			// Verify result
			if result != tt.expected {
				t.Errorf("shouldSubmit() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
