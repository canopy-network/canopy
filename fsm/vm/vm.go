package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/canopy-network/canopy/lib"
)

/*
This code implements a **CosmWasm Virtual Machine (VM) wrapper** for the Canopy blockchain network. Here's a summary:

**Core Purpose**: Manages WebAssembly (WASM) smart contract execution using the CosmWasm runtime.

**Key Features**:
- **Contract Lifecycle**: Store, instantiate, execute, and query WASM smart contracts
- **Code Management**: Maps between code IDs and checksums for contract identification
- **Caching**: In-memory cache for compiled WASM bytecode to improve performance
- **Thread Safety**: Uses RWMutex for concurrent access protection
- **Gas Metering**: Tracks gas usage during contract execution
- **Validation**: Validates WASM bytecode before deployment

**Main Operations**:
- `StoreCode()` - Compiles and stores WASM contracts
- `InstantiateContract()` - Creates new contract instances
- `ExecuteContract()` - Executes contract methods
- `QueryContract()` - Read-only contract queries
- `ValidateContract()` - Validates contract bytecode

The VM acts as a bridge between the blockchain and CosmWasm runtime, providing a clean interface for smart contract operations with proper resource management and error handling.
*/

// VM manages CosmWasm virtual machine instance and provides smart contract execution capabilities
type VM struct {
	vm                *wasmvm.VM
	dataDir           string
	cache             Cache
	supportedFeatures []string
	memoryLimit       uint32
	mu                sync.RWMutex
	checksumToID      map[string]uint64
	idToChecksum      map[uint64][]byte
	nextCodeID        uint64
	logger            lib.LoggerI
}

// Cache provides an interface for caching compiled WASM code
type Cache interface {
	GetCode(codeID uint64) ([]byte, lib.ErrorI)
	SetCode(codeID uint64, code []byte) lib.ErrorI
	HasCode(codeID uint64) bool
	RemoveCode(codeID uint64) lib.ErrorI
}

// Config contains configuration options for the VM
type Config struct {
	DataDir           string   // Directory for VM data storage
	SupportedFeatures []string // List of supported CosmWasm features
	MemoryLimit       uint32   // Memory limit per contract execution in MiB
	CacheSize         uint32   // Size of the WASM code cache
}

// DefaultConfig returns a default VM configuration suitable for Canopy
func DefaultConfig() Config {
	return Config{
		DataDir: "data/wasm",
		SupportedFeatures: []string{
			"iterator",
			"staking",
			"ibc2",
			"cosmwasm_1_1",
			"cosmwasm_1_2",
			"cosmwasm_1_3",
			"cosmwasm_1_4",
			"cosmwasm_2_0",
			"cosmwasm_2_1",
			"cosmwasm_2_2",
			"stargate",
		},
		MemoryLimit: 32,  // 32 MiB memory limit
		CacheSize:   100, // Cache up to 100 compiled contracts
	}
}

// NewVM creates a new CosmWasm VM instance with the given configuration
func NewVM(config Config, logger lib.LoggerI) (*VM, lib.ErrorI) {
	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, ErrCreateDataDir(err)
	}

	// Initialize cache
	cache, err := NewInMemoryCache(config.CacheSize)
	if err != nil {
		return nil, ErrInitCache(err)
	}

	// Create the VM instance
	wasmVMInstance, e := wasmvm.NewVM(
		config.DataDir,
		config.SupportedFeatures,
		config.MemoryLimit,
		true,             // printDebug
		config.CacheSize, // cache capacity
	)
	if e != nil {
		return nil, ErrCreateVM(e)
	}

	return &VM{
		vm:                wasmVMInstance,
		dataDir:           config.DataDir,
		cache:             cache,
		supportedFeatures: config.SupportedFeatures,
		memoryLimit:       config.MemoryLimit,
		logger:            logger,
		checksumToID:      map[string]uint64{},
		idToChecksum:      map[uint64][]byte{},
		mu:                sync.RWMutex{},
	}, nil
}

// Close shuts down the VM and cleans up resources
func (vm *VM) Close() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.vm != nil {
		vm.vm.Cleanup()
		vm.vm = nil
	}
	return nil
}

// StoreCode compiles and stores WASM bytecode, returning a code ID
func (vm *VM) StoreCode(wasmCode []byte) (uint64, lib.ErrorI) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.vm == nil {
		return 0, ErrVMClosed()
	}
	// Compile the WASM code
	checksum, gasCost, err := vm.vm.StoreCode(wasmvm.WasmCode(wasmCode), 800000000000)
	if err != nil {
		return 0, ErrStoreCode(err, gasCost)
	}

	// Generate a code ID from the checksum
	codeID := vm.checksumToCodeID(checksum)

	// Cache the compiled code
	if err := vm.cache.SetCode(codeID, wasmCode); err != nil {
		vm.logger.Warnf("Failed to cache WASM code: %v", err)
	}

	vm.logger.Debugf("Stored WASM code with ID: %d, cost: %d", codeID, gasCost)
	return codeID, nil
}

// GetCode retrieves stored WASM bytecode by code ID
func (vm *VM) GetCode(codeID uint64) ([]byte, lib.ErrorI) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	// Try cache first
	if code, err := vm.cache.GetCode(codeID); err == nil {
		return code, nil
	}

	// If not in cache, this would typically load from persistent storage
	// For now, return an error as we need to implement persistent code storage
	return nil, ErrGetCode(codeID)
}

// HasCode checks if WASM code exists for the given code ID
func (vm *VM) HasCode(codeID uint64) bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.cache.HasCode(codeID)
}

// InstantiateContract creates a new contract instance
func (vm *VM) InstantiateContract(
	codeID uint64,
	env wasmvmtypes.Env,
	info wasmvmtypes.MessageInfo,
	msg []byte,
	store wasmvmtypes.KVStore,
	goapi wasmvmtypes.GoAPI,
	querier wasmvmtypes.Querier,
	gasLimit uint64,
) (*wasmvmtypes.ContractResult, uint64, lib.ErrorI) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	// nil vm check
	if vm.vm == nil {
		return nil, 0, ErrVMClosed()
	}
	// convert code ID to checksum
	checksum := vm.codeIDToChecksum(codeID)
	// create a new gas meter
	gasMeter := NewSimpleGasMeter(gasLimit)
	// execute instantiation
	res, gasUsed, err := vm.vm.Instantiate(
		checksum,
		env,
		info,
		msg,
		store,
		goapi,
		querier,
		gasMeter,
		gasLimit,
		wasmvmtypes.UFraction{Numerator: 1, Denominator: 10},
	)
	if err != nil {
		return nil, gasUsed, ErrInstantiateContract(err)
	}

	return res, gasUsed, nil
}

// ExecuteContract executes a message on an existing contract
func (vm *VM) ExecuteContract(
	codeID uint64,
	env wasmvmtypes.Env,
	info wasmvmtypes.MessageInfo,
	msg []byte,
	store wasmvmtypes.KVStore,
	goapi wasmvmtypes.GoAPI,
	querier wasmvmtypes.Querier,
	gasLimit uint64,
) (*wasmvmtypes.ContractResult, uint64, lib.ErrorI) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	// nil vm check
	if vm.vm == nil {
		return nil, 0, ErrVMClosed()
	}
	// convert code ID to checksum
	checksum := vm.codeIDToChecksum(codeID)
	// execute the contract call
	gasMeter := NewSimpleGasMeter(gasLimit)
	res, gasUsed, err := vm.vm.Execute(
		checksum,
		env,
		info,
		msg,
		store,
		goapi,
		querier,
		gasMeter,
		gasLimit,
		wasmvmtypes.UFraction{Numerator: 1, Denominator: 10},
	)
	if err != nil {
		return nil, gasUsed, ErrExecuteContract(err)
	}

	return res, gasUsed, nil
}

// QueryContract queries a contract without modifying state
func (vm *VM) QueryContract(
	codeID uint64,
	env wasmvmtypes.Env,
	msg []byte,
	store wasmvmtypes.KVStore,
	goapi wasmvmtypes.GoAPI,
	querier wasmvmtypes.Querier,
	gasLimit uint64,
) (*wasmvmtypes.QueryResult, uint64, lib.ErrorI) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.vm == nil {
		return nil, 0, ErrVMClosed()
	}
	// Convert code ID to checksum
	checksum := vm.codeIDToChecksum(codeID)
	// Execute the query
	gasMeter := NewSimpleGasMeter(gasLimit)
	res, gasUsed, err := vm.vm.Query(
		wasmvmtypes.Checksum(checksum),
		env,
		msg,
		store,
		goapi,
		querier,
		gasMeter,
		gasLimit,
		wasmvmtypes.UFraction{Numerator: 1, Denominator: 10},
	)
	if err != nil {
		return nil, gasUsed, ErrQueryContract(err)
	}

	return res, gasUsed, nil
}

// ValidateContract validates WASM bytecode without storing it
func (vm *VM) ValidateContract(wasmCode []byte) lib.ErrorI {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.vm == nil {
		return ErrVMClosed()
	}
	// create a temporary directory for validation
	tempDir := filepath.Join(vm.dataDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return ErrCreateTempDir(err)
	}
	defer os.RemoveAll(tempDir)
	// create a temporary VM for validation
	tempVM, err := wasmvm.NewVM(
		tempDir,
		vm.supportedFeatures,
		vm.memoryLimit,
		false,
		10000,
	)
	if err != nil {
		return ErrCreateTempVM(err)
	}
	defer tempVM.Cleanup()
	// Try to compile the code
	_, _, err = tempVM.StoreCode(wasmvm.WasmCode(wasmCode), 800000000000)
	if err != nil {
		return ErrValidateContract(err)
	}

	return nil
}

// checksumToCodeID converts a wasmvm checksum to a code ID
// Uses a persistent mapping to ensure consistent and reversible conversions
func (vm *VM) checksumToCodeID(checksum wasmvmtypes.Checksum) uint64 {
	// Check if checksum already has an assigned ID
	if id, exists := vm.checksumToID[string(checksum)]; exists {
		return id
	}

	// Generate new ID using atomic counter to ensure uniqueness
	newID := atomic.AddUint64(&vm.nextCodeID, 1)

	// Store bidirectional mapping
	vm.checksumToID[string(checksum)] = newID
	vm.idToChecksum[newID] = make([]byte, len(checksum))
	copy(vm.idToChecksum[newID], checksum)

	return newID
}

// codeIDToChecksum converts a code ID back to a wasmvm checksum
// Uses the stored mapping to ensure accurate reverse conversion
func (vm *VM) codeIDToChecksum(codeID uint64) wasmvmtypes.Checksum {
	// Retrieve checksum from mapping
	if checksum, exists := vm.idToChecksum[codeID]; exists {
		return checksum
	}

	// Return nil if mapping doesn't exist
	fmt.Printf("Code ID %d not found in mapping\n", codeID)
	return nil
}

// Add these fields to the VM struct:
// nextCodeID uint64 // atomic counter for generating unique IDs
// checksumToID map[string]uint64 // maps checksum bytes to code IDs
// idToChecksum map[uint64][]byte // maps code IDs back to checksums

// GetMetrics returns VM performance metrics
func (vm *VM) GetMetrics() map[string]interface{} {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return map[string]interface{}{
		"data_dir":           vm.dataDir,
		"supported_features": vm.supportedFeatures,
		"memory_limit":       vm.memoryLimit,
		"cache_size":         vm.cache,
	}
}
