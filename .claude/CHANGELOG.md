# Changelog

## [Unreleased]

### Improvements
#### Oracle Submission History Pruning
- Enhanced `PruneHistory()` method in Oracle state management
- Implemented selective submission history pruning
- Prevents unnecessary order resubmissions for valid orders
- Reduces state inconsistency in cross-chain order processing
- Adds detailed logging for pruning operations
- Maintains thread-safe pruning with existing mutex protection

### Security
#### Order Store Rollback Vulnerability Fix
- Addressed state inconsistency vulnerability in Oracle submission tracking
- Added `o.state.ResetSubmissionHistory()` call in `reorgRollback()` method
- Ensures complete cleanup of submission history during chain reorganizations
- Prevents stale submission tracking that could block legitimate order resubmissions

### Modified
#### Oracle Test Helpers Enhancement
- Improved `transactionWithOrder` helper function in `cmd/rpc/oracle/oracle_test.go`
- Added optional `To` parameter support for more flexible test case configuration
- Enhanced `createSellOrder` test helper with robust hex data conversion using `StringToBytes`
- Updated test cases to use proper contract addresses for close order validation
- Improved test data setup to match Oracle validation requirements

#### Oracle Close Order Validation
- Updated `TestOracle_validateCloseOrder` in Oracle RPC tests
- Added new validation logic for token transfer recipient address
- Introduced test cases for:
  - Verifying token transfer recipient matches sell order's seller receive address
  - Handling invalid hex address conversion
- Improved test coverage for close order validation

#### Ethereum Oracle Block Provider
- Simplified `EthBlockProvider` struct by removing `safeHeight` field
- Updated `processBlocks` method test in `cmd/rpc/oracle/eth/block_provider_test.go`
  - Changed test signature to include `startHeight` and `endHeight` parameters
  - Updated test parameters and expectations
  - Removed obsolete `TestEthBlockProvider_updateHeights` test
- Test modifications ensure correct method behavior without using internal `safeHeight` tracking

### Files Modified
- `cmd/rpc/oracle/eth/block_provider_test.go`
- `cmd/rpc/oracle/oracle_test.go`
- `cmd/rpc/oracle/oracle.go`
- `cmd/rpc/oracle/state.go`

### Impact
- Simplified block processing logic
- Improved test coverage
- Removed unnecessary internal state tracking
- Enhanced Oracle close order validation with stricter address matching
- Added comprehensive test scenarios for close order validation
- Increased Oracle state management resilience during chain reorganizations
- Prevented potential order submission blocking scenarios

### SafeBlock Refactoring
#### Oracle State Management
- Moved SafeBlock confirmation logic from EthBlockProvider to OracleState
- Implemented thread-safe SafeBlock handling with RWMutex protection
- Removed redundant SafeBlockConfirmations field from EthBlockProviderConfig
- Consolidated SafeBlock processing logic into a single, efficient implementation

#### Performance Improvements
- Block provider now delivers all available blocks immediately
- Eliminated artificial block delivery delays
- Reduced mutex operations in block processing
- Maintained monotonic height guarantee for safe blocks

### Files Modified
- `lib/config.go`
- `cmd/rpc/oracle/eth/block_provider.go`
- `cmd/rpc/oracle/eth/block_provider_test.go`
- `cmd/rpc/oracle/oracle.go`
- `cmd/rpc/oracle/state.go`

### Technical Highlights
- Clear architectural separation between block delivery and safe block confirmation
- Backward-compatible configuration
- Improved system performance and reliability
