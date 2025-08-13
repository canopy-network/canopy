# Phase 1 Implementation Plan: Core EVM State Bridge

## Overview
Phase 1 focuses on creating the core bridge between the Canopy blockchain's key-value store and the Ethereum Virtual Machine's state management requirements. This phase implements the foundational interfaces needed for EVM integration.

## New Files to Create

### 1. `fsm/evm_state.go`
**Primary file implementing the EVM StateDB interface**

#### Types:
```go
type CanopyStateDB struct {
    fsm         *StateMachine
    snapshots   []int
    logs        []*types.Log
    refund      uint64
    accessList  *accessList
    txHash      common.Hash
    blockHash   common.Hash
    txIndex     int
    logSize     uint
}

type accessList struct {
    addresses map[common.Address]int
    slots     []map[common.Hash]int
}

type stateObject struct {
    address  common.Address
    addrHash common.Hash
    data     Account
    code     []byte
    codeHash common.Hash
    storage  map[common.Hash]common.Hash
    dirtyStorage map[common.Hash]common.Hash
    
    // Canopy-specific fields
    canopyAddress crypto.AddressI
    db           *CanopyStateDB
}
```

#### Methods:
```go
// Constructor
func NewCanopyStateDB(fsm *StateMachine, txHash, blockHash common.Hash) *CanopyStateDB

// Account operations
func (db *CanopyStateDB) CreateAccount(addr common.Address)
func (db *CanopyStateDB) SubBalance(addr common.Address, amount *big.Int)
func (db *CanopyStateDB) AddBalance(addr common.Address, amount *big.Int)
func (db *CanopyStateDB) GetBalance(addr common.Address) *big.Int
func (db *CanopyStateDB) GetNonce(addr common.Address) uint64
func (db *CanopyStateDB) SetNonce(addr common.Address, nonce uint64)

// Code operations
func (db *CanopyStateDB) GetCodeHash(addr common.Address) common.Hash
func (db *CanopyStateDB) GetCode(addr common.Address) []byte
func (db *CanopyStateDB) SetCode(addr common.Address, code []byte)
func (db *CanopyStateDB) GetCodeSize(addr common.Address) int

// Storage operations
func (db *CanopyStateDB) GetState(addr common.Address, key common.Hash) common.Hash
func (db *CanopyStateDB) SetState(addr common.Address, key, value common.Hash)
func (db *CanopyStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash

// Refund operations
func (db *CanopyStateDB) AddRefund(gas uint64)
func (db *CanopyStateDB) SubRefund(gas uint64)
func (db *CanopyStateDB) GetRefund() uint64

// State lifecycle
func (db *CanopyStateDB) Exist(addr common.Address) bool
func (db *CanopyStateDB) Empty(addr common.Address) bool
func (db *CanopyStateDB) Suicide(addr common.Address) bool
func (db *CanopyStateDB) HasSuicided(addr common.Address) bool

// Snapshots
func (db *CanopyStateDB) Snapshot() int
func (db *CanopyStateDB) RevertToSnapshot(revid int)

// Access lists (EIP-2930)
func (db *CanopyStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, list types.AccessList)
func (db *CanopyStateDB) AddressInAccessList(addr common.Address) bool
func (db *CanopyStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool)
func (db *CanopyStateDB) AddAddressToAccessList(addr common.Address)
func (db *CanopyStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash)

// Logs
func (db *CanopyStateDB) AddLog(log *types.Log)
func (db *CanopyStateDB) GetLogs(txHash, blockHash common.Hash) []*types.Log

// Finalization
func (db *CanopyStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash
func (db *CanopyStateDB) Finalise(deleteEmptyObjects bool)
func (db *CanopyStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error)

// Helper methods
func (db *CanopyStateDB) getStateObject(addr common.Address) *stateObject
func (db *CanopyStateDB) createObject(addr common.Address) *stateObject
func (db *CanopyStateDB) updateStateObject(obj *stateObject)
func (db *CanopyStateDB) deleteStateObject(obj *stateObject)
func (db *CanopyStateDB) convertAddress(addr common.Address) crypto.AddressI
func (db *CanopyStateDB) convertToCanopyAddress(addr common.Address) crypto.AddressI
func (db *CanopyStateDB) convertFromCanopyAddress(addr crypto.AddressI) common.Address
```

### 2. `fsm/evm_chain.go`
**ChainContext interface implementation**

#### Types:
```go
type CanopyChainContext struct {
    fsm *StateMachine
}

type CanopyBlockHeader struct {
    Number     uint64
    Time       uint64
    Hash       common.Hash
    ParentHash common.Hash
    // Additional header fields as needed
}
```

#### Methods:
```go
// Constructor
func NewCanopyChainContext(fsm *StateMachine) *CanopyChainContext

// ChainContext interface
func (cc *CanopyChainContext) GetHeader(hash common.Hash, number uint64) *types.Header

// Helper methods
func (cc *CanopyChainContext) convertToEthHeader(canopyHeader *CanopyBlockHeader) *types.Header
func (cc *CanopyChainContext) getCurrentHeader() *types.Header
func (cc *CanopyChainContext) getHeaderByNumber(number uint64) (*types.Header, error)
func (cc *CanopyChainContext) getHeaderByHash(hash common.Hash) (*types.Header, error)
```

### 3. `fsm/evm_config.go`
**EVM configuration and parameters**

#### Types:
```go
type CanopyEVMConfig struct {
    ChainConfig *params.ChainConfig
    VMConfig    vm.Config
    GasLimit    uint64
    BaseFee     *big.Int
    
    // Canopy-specific configuration
    NetworkID        uint32
    ProtocolVersion  uint64
    EVMEnabled       bool
    MaxCodeSize      int
    MaxInitCodeSize  int
}

type GasConfig struct {
    GasLimit    uint64
    GasPrice    *big.Int
    BaseFee     *big.Int
    GasTipCap   *big.Int
    GasFeeCap   *big.Int
}
```

#### Methods:
```go
// Constructor
func NewCanopyEVMConfig(networkID uint32, protocolVersion uint64) *CanopyEVMConfig

// Configuration methods
func (cfg *CanopyEVMConfig) GetChainConfig() *params.ChainConfig
func (cfg *CanopyEVMConfig) GetVMConfig() vm.Config
func (cfg *CanopyEVMConfig) IsEVMEnabled() bool
func (cfg *CanopyEVMConfig) GetGasLimit() uint64
func (cfg *CanopyEVMConfig) GetBaseFee() *big.Int

// Gas configuration
func (cfg *CanopyEVMConfig) NewGasConfig(gasLimit uint64, gasPrice *big.Int) *GasConfig
func (cfg *CanopyEVMConfig) CalculateBaseFee(parentGasUsed, parentGasLimit uint64) *big.Int
func (cfg *CanopyEVMConfig) ValidateGasParams(gasLimit uint64, gasPrice *big.Int) error

// Chain configuration helpers
func (cfg *CanopyEVMConfig) GetForkBlock(forkName string) *big.Int
func (cfg *CanopyEVMConfig) IsForked(blockNumber uint64, forkName string) bool
func (cfg *CanopyEVMConfig) GetActivePrecompiles(blockNumber uint64) []common.Address

// Canopy-specific methods
func (cfg *CanopyEVMConfig) ConvertCanopyFeeToGas(canopyFee uint64) uint64
func (cfg *CanopyEVMConfig) ConvertGasToCanopyFee(gas uint64) uint64
func (cfg *CanopyEVMConfig) GetMaxTransactionSize() int
func (cfg *CanopyEVMConfig) GetMaxContractSize() int
```

## Modified Files

### 1. `fsm/key.go`
**Add new key prefixes for EVM data**

#### New Constants:
```go
// EVM-related key prefixes
var (
    contractCodePrefix    = []byte{0x0C}    // Contract bytecode storage
    contractStoragePrefix = []byte{0x0D}    // Contract storage slots
    contractNoncePrefix   = []byte{0x0E}    // Contract account nonces
    transactionLogPrefix  = []byte{0x0F}    // Transaction logs/events
    suicidePrefix         = []byte{0x10}    // Suicided contract addresses
)
```

#### New Key Generation Functions:
```go
// Contract code keys
func KeyForContractCode(addr common.Address) []byte
func KeyForContractStorage(addr common.Address, slot common.Hash) []byte
func KeyForContractNonce(addr common.Address) []byte

// Transaction log keys
func KeyForTransactionLog(txHash common.Hash, logIndex uint) []byte
func KeyForBlockLogs(blockHash common.Hash) []byte

// Suicide tracking
func KeyForSuicide(addr common.Address) []byte

// Helper functions
func ContractCodePrefix() []byte
func ContractStoragePrefix() []byte
func TransactionLogPrefix() []byte
```

### 2. `fsm/account.go`
**Extend Account struct for EVM compatibility**

#### Extended Account Type:
```go
type Account struct {
    Address  []byte // address: the short version of a public key
    Amount   uint64 // amount: the balance of funds the account has
    Nonce    uint64 // nonce: transaction count for the account
    CodeHash []byte // codeHash: hash of contract code (if contract account)
}
```

#### New Methods:
```go
// EVM-specific account methods
func (s *StateMachine) GetAccountNonce(address crypto.AddressI) (uint64, lib.ErrorI)
func (s *StateMachine) SetAccountNonce(address crypto.AddressI, nonce uint64) lib.ErrorI
func (s *StateMachine) GetAccountCodeHash(address crypto.AddressI) ([]byte, lib.ErrorI)
func (s *StateMachine) SetAccountCodeHash(address crypto.AddressI, codeHash []byte) lib.ErrorI
func (s *StateMachine) IsContractAccount(address crypto.AddressI) (bool, lib.ErrorI)
func (s *StateMachine) CreateContractAccount(address crypto.AddressI, codeHash []byte) lib.ErrorI
```

### 3. `fsm/error.go`
**Add EVM-specific error types**

#### New Error Functions:
```go
// EVM-specific errors
func ErrInvalidContractAddress() lib.ErrorI
func ErrContractCodeNotFound() lib.ErrorI
func ErrInvalidCodeHash() lib.ErrorI
func ErrContractStorageError() lib.ErrorI
func ErrInvalidStorageKey() lib.ErrorI
func ErrEVMExecutionFailed() lib.ErrorI
func ErrInvalidGasLimit() lib.ErrorI
func ErrInvalidGasPrice() lib.ErrorI
func ErrInsufficientGas() lib.ErrorI
func ErrInvalidSnapshot() lib.ErrorI
func ErrAccessListError() lib.ErrorI
```

## Implementation Task Breakdown

### Task 1: Setup and Dependencies (Day 1)
- [X] Add Ethereum Go dependencies to go.mod
- [X] Create basic file structure for new files
- [X] Set up import statements and basic type definitions

### Task 2: Key Management Extension (Day 1-2)
- [ ] Implement new key prefixes in `fsm/key.go`
- [ ] Add key generation functions for EVM data
- [ ] Write unit tests for key generation functions

### Task 3: Account Model Extension (Day 2-3)
- [ ] Extend Account struct with nonce and code hash fields
- [ ] Implement EVM-specific account methods in `fsm/account.go`
- [ ] Add account serialization/deserialization for new fields
- [ ] Write unit tests for extended account functionality

### Task 4: Error Handling Setup (Day 3)
- [ ] Add EVM-specific error types in `fsm/error.go`
- [ ] Implement error messages and error codes
- [ ] Write unit tests for error handling

### Task 5: EVM Configuration (Day 4-5)
- [ ] Implement `CanopyEVMConfig` struct in `fsm/evm_config.go`
- [ ] Set up chain configuration for Canopy network
- [ ] Implement gas configuration and conversion methods
- [ ] Add configuration validation methods
- [ ] Write unit tests for configuration functionality

### Task 6: Chain Context Implementation (Day 5-6)
- [ ] Implement `CanopyChainContext` in `fsm/evm_chain.go`
- [ ] Create header conversion methods
- [ ] Implement `GetHeader` method
- [ ] Add helper methods for header management
- [ ] Write unit tests for chain context functionality

### Task 7: StateDB Foundation (Day 6-8)
- [ ] Implement basic `CanopyStateDB` struct in `fsm/evm_state.go`
- [ ] Add address conversion helper methods
- [ ] Implement state object management
- [ ] Add snapshot and rollback infrastructure
- [ ] Write unit tests for foundation functionality

### Task 8: Account Operations (Day 8-10)
- [ ] Implement balance operations (GetBalance, AddBalance, SubBalance)
- [ ] Implement nonce operations (GetNonce, SetNonce)
- [ ] Implement account existence checks (Exist, Empty)
- [ ] Implement account creation (CreateAccount)
- [ ] Write unit tests for account operations

### Task 9: Code Operations (Day 10-12)
- [ ] Implement contract code storage (SetCode, GetCode)
- [ ] Implement code hash operations (GetCodeHash)
- [ ] Implement code size operations (GetCodeSize)
- [ ] Add code validation and limits
- [ ] Write unit tests for code operations

### Task 10: Storage Operations (Day 12-14)
- [ ] Implement contract storage operations (GetState, SetState)
- [ ] Implement committed state operations (GetCommittedState)
- [ ] Add storage key-value persistence
- [ ] Implement storage change tracking
- [ ] Write unit tests for storage operations

### Task 11: Advanced Features (Day 14-16)
- [ ] Implement access list functionality (EIP-2930)
- [ ] Implement suicide/selfdestruct tracking
- [ ] Implement gas refund tracking
- [ ] Implement transaction log management
- [ ] Write unit tests for advanced features

### Task 12: Integration Testing (Day 16-18)
- [ ] Write integration tests for StateDB with existing FSM
- [ ] Test state persistence and retrieval
- [ ] Test snapshot and rollback functionality
- [ ] Test concurrent access patterns
- [ ] Performance testing and optimization

### Task 13: Documentation and Cleanup (Day 18-20)
- [ ] Add comprehensive code documentation
- [ ] Write usage examples
- [ ] Create developer documentation
- [ ] Code review and cleanup
- [ ] Final testing and validation

## Dependencies and Prerequisites

### External Dependencies:
- `github.com/ethereum/go-ethereum/core/vm` - EVM implementation
- `github.com/ethereum/go-ethereum/common` - Common Ethereum types
- `github.com/ethereum/go-ethereum/core/types` - Transaction and block types
- `github.com/ethereum/go-ethereum/params` - Network parameters

### Internal Dependencies:
- Existing FSM StateMachine implementation
- Current key-value store interface
- Account management system
- Transaction processing framework

## Success Criteria

At the end of Phase 1, we should have:
- [ ] Complete StateDB interface implementation
- [ ] Working ChainContext for block header access
- [ ] Proper EVM configuration for Canopy network
- [ ] Extended account model with EVM fields
- [ ] Comprehensive test coverage (>80%)
- [ ] All tests passing
- [ ] Documentation for all new interfaces
- [ ] Ready for Phase 2 transaction processing integration

## Risk Mitigation

### Technical Risks:
- **Address Format Compatibility**: Ensure proper conversion between Canopy and Ethereum address formats
- **Storage Key Conflicts**: Prevent key collisions between EVM and native Canopy storage
- **Performance Impact**: Monitor state access performance with new abstraction layer
- **Memory Usage**: Implement efficient caching and cleanup for state objects

### Mitigation Strategies:
- Extensive unit testing for address conversions
- Use distinct key prefixes for EVM data
- Benchmark state operations and optimize critical paths
- Implement proper resource cleanup and garbage collection
