package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

// createTempDirForState creates a temporary directory for test state files, cleaned up on test exit
func createTempDirForState(t *testing.T) string {
	dir, err := os.MkdirTemp("", "oracle_state_test_*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestOracleState_removeState(t *testing.T) {
	t.Run("clears in-memory cache and removes file from disk", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		block := createTestBlock(1, "0xblock1", "0xparent1")
		require.NoError(t, bsm.saveState(block))

		// sanity: file exists and in-memory cache is populated
		_, statErr := os.Stat(stateFile)
		require.NoError(t, statErr, "state file should exist after saveState")
		require.Equal(t, uint64(1), bsm.GetLastHeight())

		err := bsm.removeState()
		require.NoError(t, err)

		// in-memory cache cleared
		assert.Equal(t, uint64(0), bsm.GetLastHeight())

		// file removed from disk
		_, statErr = os.Stat(stateFile)
		assert.True(t, os.IsNotExist(statErr), "state file should no longer exist")
	})

	t.Run("tolerates file already missing", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "never_created_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		// file was never created - removeState should not error
		err := bsm.removeState()
		assert.NoError(t, err)
		assert.Equal(t, uint64(0), bsm.GetLastHeight())
	})

	t.Run("resets monotonic safeHeight and sourceChainHeight so a reorg re-sync cannot inherit stale height", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		// simulate a synced oracle: high safeHeight/sourceChainHeight and a persisted block
		cfg := lib.OracleConfig{SafeBlockConfirmations: 5}
		bsm.updateSafeHeight(1000, cfg) // safeHeight -> 995
		require.NoError(t, bsm.saveState(createTestBlock(1000, "0xblock1000", "0xparent1000")))
		require.NoError(t, bsm.ValidateSequence(createTestBlock(1001, "0xblock1001", "0xblock1000")))
		require.Equal(t, uint64(995), bsm.GetSafeHeight())
		require.Equal(t, uint64(1001), bsm.GetSourceChainHeight())

		// reorg/gap restart path
		require.NoError(t, bsm.removeState())

		// all derived height state must be cleared (as if a fresh process start)
		assert.Equal(t, uint64(0), bsm.GetLastHeight(), "blockState should be cleared")
		assert.Equal(t, uint64(0), bsm.GetSafeHeight(), "safeHeight must reset, not stay frozen high")
		assert.Equal(t, uint64(0), bsm.GetSourceChainHeight(), "sourceChainHeight must reset")

		// and the monotonic guard must let it rebuild from the re-sync point afterward
		bsm.updateSafeHeight(1000, cfg)
		assert.Equal(t, uint64(995), bsm.GetSafeHeight(), "safeHeight should rebuild after reset")
	})
}

func TestOracleState_GetLastHeight(t *testing.T) {
	t.Run("returns 0 when no prior state", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		assert.Equal(t, uint64(0), bsm.GetLastHeight())
	})

	t.Run("returns blockState.Height when state exists", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		block := createTestBlock(42, "0xblock42", "0xblock41")
		require.NoError(t, bsm.saveState(block))

		assert.Equal(t, uint64(42), bsm.GetLastHeight())
	})
}

// TestOracleState_ConcurrentAccess is a stress test that hammers saveState, removeState,
// GetLastHeight, and ValidateSequence concurrently from many goroutines. It exists to prove
// that rwLock actually serializes access to shared state fields (blockState, sourceChainHeight).
// Run with -race: a broken/missing lock should reliably trigger a data race under this load.
func TestOracleState_ConcurrentAccess(t *testing.T) {
	tempDir := createTempDirForState(t)
	stateFile := filepath.Join(tempDir, "test_state")
	logger := lib.NewDefaultLogger()
	bsm := NewOracleState(stateFile, logger)

	const numGoroutines = 50
	const numIterations = 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4)

	// goroutines calling saveState with monotonically-ish increasing heights per worker
	for g := 0; g < numGoroutines; g++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				height := uint64(worker*numIterations + i + 1)
				block := createTestBlock(height, fmt.Sprintf("0xblock%d", height), fmt.Sprintf("0xblock%d", height-1))
				_ = bsm.saveState(block)
			}
		}(g)
	}

	// goroutines calling GetLastHeight
	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				_ = bsm.GetLastHeight()
			}
		}()
	}

	// goroutines calling ValidateSequence (mostly will error due to gaps/reorgs under concurrent
	// writers, but the point is to exercise the lock, not to assert on the sequencing outcome)
	for g := 0; g < numGoroutines; g++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				height := uint64(worker*numIterations + i + 1)
				block := createTestBlock(height, fmt.Sprintf("0xblock%d", height), fmt.Sprintf("0xblock%d", height-1))
				_ = bsm.ValidateSequence(block)
			}
		}(g)
	}

	// goroutines calling removeState interleaved with the above
	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				_ = bsm.removeState()
			}
		}()
	}

	wg.Wait()

	// final sanity: no panic, and state is in a consistent (readable) condition
	_ = bsm.GetLastHeight()
}

func TestOracleState_updateSafeHeight(t *testing.T) {
	t.Run("monotonic: safe height only increases, never decreases", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		config := lib.OracleConfig{SafeBlockConfirmations: 5}

		// first call: height 100 - 5 confirmations = safe height 95
		bsm.updateSafeHeight(100, config)
		assert.Equal(t, uint64(95), bsm.GetSafeHeight())

		// second call with a lower current height (e.g. from a stale/late reorg-related call):
		// 50 - 5 = 45, which is lower than current safe height 95 - must NOT decrease
		bsm.updateSafeHeight(50, config)
		assert.Equal(t, uint64(95), bsm.GetSafeHeight())

		// call with a higher height should advance it
		bsm.updateSafeHeight(200, config)
		assert.Equal(t, uint64(195), bsm.GetSafeHeight())

		// another lower call after the increase still must not decrease
		bsm.updateSafeHeight(150, config)
		assert.Equal(t, uint64(195), bsm.GetSafeHeight())
	})

	t.Run("confirmation depth subtraction math", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		config := lib.OracleConfig{SafeBlockConfirmations: 10}

		bsm.updateSafeHeight(30, config)
		assert.Equal(t, uint64(20), bsm.GetSafeHeight())
	})

	t.Run("currentBlockHeight less than confirmations clamps to 0, no underflow", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		config := lib.OracleConfig{SafeBlockConfirmations: 100}

		// currentBlockHeight (5) < confirmations (100) - must not underflow uint64, clamps to 0
		bsm.updateSafeHeight(5, config)
		assert.Equal(t, uint64(0), bsm.GetSafeHeight())
	})

	t.Run("currentBlockHeight exactly equal to confirmations clamps to 0", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		bsm := NewOracleState(stateFile, logger)

		config := lib.OracleConfig{SafeBlockConfirmations: 10}

		// currentBlockHeight == confirmations: code uses strict ">" so this falls into the else
		// branch (0), not 0 via subtraction - verifying the boundary condition explicitly
		bsm.updateSafeHeight(10, config)
		assert.Equal(t, uint64(0), bsm.GetSafeHeight())
	})
}

func TestOracleState_saveState_AtomicWriteFailure(t *testing.T) {
	// skip when running as root - root bypasses directory permission bits, which would make
	// this test's failure-injection mechanism a no-op (false negative rather than false pass)
	if os.Geteuid() == 0 {
		t.Skip("cannot exercise permission-denied write failure as root")
	}

	tempDir := createTempDirForState(t)
	logger := lib.NewDefaultLogger()

	// let NewOracleState create the directory normally (it calls os.MkdirAll on construction),
	// then strip write permission from that directory afterwards so AtomicWriteFile's
	// os.CreateTemp(dir, ...) call fails with permission denied on the actual write attempt.
	targetDir := filepath.Join(tempDir, "state_dir")
	stateFile := filepath.Join(targetDir, "test_state")

	bsm := NewOracleState(stateFile, logger)

	require.NoError(t, os.Chmod(targetDir, 0555))
	t.Cleanup(func() { os.Chmod(targetDir, 0755) }) // restore so t.Cleanup can remove tempDir

	block := createTestBlock(1, "0xblock1", "0xparent1")
	err := bsm.saveState(block)

	require.Error(t, err, "saveState should fail when AtomicWriteFile cannot create its temp file")
	assert.Equal(t, CodeWriteStateFile, err.Code())

	// per the actual saveState implementation, the in-memory cache is only updated AFTER a
	// successful AtomicWriteFile call - since the write failed, the cache must remain untouched
	// (still nil / height 0), i.e. the on-disk write failure does not leave the in-memory state
	// out of sync with disk.
	assert.Equal(t, uint64(0), bsm.GetLastHeight())
}

func TestOracleState_readBlockState_CorruptFile(t *testing.T) {
	t.Run("corrupt JSON returns ErrParseState", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()

		// write malformed JSON directly to the state file path before constructing OracleState
		require.NoError(t, os.WriteFile(stateFile, []byte("{not valid json!!"), 0644))

		bsm := NewOracleState(stateFile, logger)

		// NewOracleState should recover gracefully (per its documented behavior: log loudly but
		// resume from height 0) rather than propagating the error to the caller/panicking
		assert.Equal(t, uint64(0), bsm.GetLastHeight())

		// directly exercise readBlockState to verify the sentinel error code
		state, err := bsm.readBlockState()
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Equal(t, CodeParseHeight, err.Code())
	})

	t.Run("missing file returns ErrReadStateFile and NewOracleState starts fresh with no error surfaced to caller", func(t *testing.T) {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state_missing")
		logger := lib.NewDefaultLogger()

		// file was never created
		bsm := NewOracleState(stateFile, logger)

		// starts fresh, no panic, height 0
		assert.Equal(t, uint64(0), bsm.GetLastHeight())

		// readBlockState itself still returns an error for the missing file (constructor
		// swallows/logs it, but the underlying call is not silent about the missing file)
		state, err := bsm.readBlockState()
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Equal(t, CodeReadStateFile, err.Code())
	})
}

func TestOracleState_PruneHistory(t *testing.T) {
	newTestState := func(t *testing.T) *OracleState {
		tempDir := createTempDirForState(t)
		stateFile := filepath.Join(tempDir, "test_state")
		logger := lib.NewDefaultLogger()
		return NewOracleState(stateFile, logger)
	}

	t.Run("nil order book clears all history", func(t *testing.T) {
		bsm := newTestState(t)
		bsm.lockOrderSubmissions["lock1"] = 10
		bsm.closeOrderSubmissions["close1"] = 20

		bsm.PruneHistory(nil)

		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 0, lockCount)
		assert.Equal(t, 0, closeCount)
	})

	t.Run("removes submissions for orders no longer in order book, keeps present ones", func(t *testing.T) {
		bsm := newTestState(t)

		presentLockID := []byte("present-lock-order")
		staleLockID := []byte("stale-lock-order")
		presentCloseID := []byte("present-close-order")
		staleCloseID := []byte("stale-close-order")

		bsm.lockOrderSubmissions[lib.BytesToString(presentLockID)] = 100
		bsm.lockOrderSubmissions[lib.BytesToString(staleLockID)] = 50
		bsm.closeOrderSubmissions[lib.BytesToString(presentCloseID)] = 200
		bsm.closeOrderSubmissions[lib.BytesToString(staleCloseID)] = 75

		// order book only contains the "present" orders - the "stale" ones are absent,
		// which per PruneHistory's actual logic (order == nil from GetOrder) means "not in
		// order book" and should be pruned
		orderBook := &lib.OrderBook{
			Orders: []*lib.SellOrder{
				{Id: presentLockID},
				{Id: presentCloseID},
			},
		}

		bsm.PruneHistory(orderBook)

		_, lockStillTracked := bsm.lockOrderSubmissions[lib.BytesToString(presentLockID)]
		_, staleLockTracked := bsm.lockOrderSubmissions[lib.BytesToString(staleLockID)]
		_, closeStillTracked := bsm.closeOrderSubmissions[lib.BytesToString(presentCloseID)]
		_, staleCloseTracked := bsm.closeOrderSubmissions[lib.BytesToString(staleCloseID)]

		assert.True(t, lockStillTracked, "present lock order submission should remain")
		assert.False(t, staleLockTracked, "stale lock order submission should be pruned")
		assert.True(t, closeStillTracked, "present close order submission should remain")
		assert.False(t, staleCloseTracked, "stale close order submission should be pruned")

		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 1, lockCount)
		assert.Equal(t, 1, closeCount)
	})

	t.Run("empty order book (no orders) prunes everything", func(t *testing.T) {
		bsm := newTestState(t)
		bsm.lockOrderSubmissions[lib.BytesToString([]byte("some-lock"))] = 10
		bsm.closeOrderSubmissions[lib.BytesToString([]byte("some-close"))] = 20

		orderBook := &lib.OrderBook{Orders: []*lib.SellOrder{}}
		bsm.PruneHistory(orderBook)

		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 0, lockCount)
		assert.Equal(t, 0, closeCount)
	})
}

func TestOracleState_Getters(t *testing.T) {
	tempDir := createTempDirForState(t)
	stateFile := filepath.Join(tempDir, "test_state")
	logger := lib.NewDefaultLogger()
	bsm := NewOracleState(stateFile, logger)

	t.Run("before any data", func(t *testing.T) {
		assert.Equal(t, uint64(0), bsm.GetSourceChainHeight())
		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 0, lockCount)
		assert.Equal(t, 0, closeCount)
	})

	t.Run("after one mutation", func(t *testing.T) {
		block := createTestBlock(1, "0xblock1", "0xparent1")
		require.NoError(t, bsm.ValidateSequence(block))
		assert.Equal(t, uint64(1), bsm.GetSourceChainHeight())

		// submission history is advanced by recordSubmission (on commit), not by shouldSubmit
		bsm.recordSubmission([]byte("order-a"), types.LockOrderType, 10)

		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 1, lockCount)
		assert.Equal(t, 0, closeCount)
	})

	t.Run("after multiple mutations", func(t *testing.T) {
		bsm.recordSubmission([]byte("order-b"), types.CloseOrderType, 10)

		block2 := createTestBlock(2, "0xblock2", "0xblock1")
		require.NoError(t, bsm.ValidateSequence(block2))

		assert.Equal(t, uint64(2), bsm.GetSourceChainHeight())
		lockCount, closeCount := bsm.SubmissionCounts()
		assert.Equal(t, 1, lockCount)
		assert.Equal(t, 1, closeCount)
	})
}
