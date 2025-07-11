package vm

import (
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/btcsuite/btcutil/bech32"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

/*
**Key Components:**

1. **SimpleGasMeter** - Tracks gas consumption with configurable limits and provides basic gas metering functionality

2. **StateBridge** - The main bridge that connects CosmWasm contracts to Canopy's state, managing gas costs and providing access to storage, APIs, and querying

3. **ContractKVStore** - Implements isolated key-value storage for each contract with gas-metered operations (Get/Set/Delete) and iteration capabilities

4. **CanopyQuerier** - Handles various query types from contracts including:
   - Bank queries (balances)
   - Custom Canopy-specific queries (chain info, validators, committees)
   - Staking and Wasm queries (partially implemented)

**Main Features:**
- Gas metering for all storage operations
- Address conversion between human-readable and canonical formats
- Contract storage isolation using prefixed keys
- Iterator support for contract storage traversal
- Extensible query system for blockchain state access

The code provides the foundational infrastructure for running CosmWasm contracts on the Canopy blockchain while maintaining proper gas accounting and state isolation.

*/

// SimpleGasMeter implements a basic gas meter for CosmWasm contracts
type SimpleGasMeter struct {
	limit uint64
	used  uint64
}

// NewSimpleGasMeter creates a new gas meter with the given limit
func NewSimpleGasMeter(limit uint64) *SimpleGasMeter {
	return &SimpleGasMeter{
		limit: limit,
		used:  0,
	}
}

// GasConsumed returns the amount of gas consumed
func (gm *SimpleGasMeter) GasConsumed() uint64 {
	fmt.Printf("DEBUG: GasConsumed() called, returning used gas: %d\n", gm.used)
	return gm.used
}

// GasConsumedToLimit returns the amount of gas consumed up to the limit
func (gm *SimpleGasMeter) GasConsumedToLimit() uint64 {
	fmt.Printf("DEBUG: GasConsumedToLimit() called, used: %d, limit: %d\n", gm.used, gm.limit)
	if gm.used > gm.limit {
		fmt.Printf("DEBUG: Used gas exceeds limit, returning limit: %d\n", gm.limit)
		return gm.limit
	}
	fmt.Printf("DEBUG: Used gas within limit, returning used: %d\n", gm.used)
	return gm.used
}

// Limit returns the gas limit
func (gm *SimpleGasMeter) Limit() uint64 {
	fmt.Printf("DEBUG: Limit() called, returning limit: %d\n", gm.limit)
	return gm.limit
}

// ConsumeGas consumes the given amount of gas
func (gm *SimpleGasMeter) ConsumeGas(amount uint64, descriptor string) error {
	fmt.Println("DEBUG: ConsumeGas() called", amount, descriptor, gm.limit)
	gm.used += amount
	if gm.used > gm.limit {
		return ErrOutOfGas(gm.used, gm.limit)
	}
	return nil
}

// StateBridge connects CosmWasm contracts to the Canopy state machine
// It implements the wasmvm interfaces for KVStore, GoAPI, and Querier
type StateBridge struct {
	store    lib.RWStoreI
	height   uint64
	chainID  uint64
	gasLimit uint64
	gasUsed  uint64
	gasCosts *GasCosts
}

// GasCosts defines gas consumption rates for different operations
type GasCosts struct {
	StorageRead   uint64
	StorageWrite  uint64
	StorageDelete uint64
}

// NewStateBridge creates a new state bridge for contract execution
func NewStateBridge(store lib.RWStoreI, height, chainID, gasLimit uint64, gasCosts *GasCosts) *StateBridge {
	return &StateBridge{
		store:    store,
		height:   height,
		chainID:  chainID,
		gasLimit: gasLimit,
		gasUsed:  0,
		gasCosts: gasCosts,
	}
}

// KVStore implementation for CosmWasm contracts
type ContractKVStore struct {
	bridge       *StateBridge
	contractAddr []byte
	prefix       []byte
}

// NewContractKVStore creates a new KV store for a specific contract
func (sb *StateBridge) NewContractKVStore(contractAddr []byte) *ContractKVStore {
	prefix := makeContractStorageKey(contractAddr)
	return &ContractKVStore{
		bridge:       sb,
		contractAddr: contractAddr,
		prefix:       prefix,
	}
}

// Get retrieves a value from the contract's key-value store
func (kvs *ContractKVStore) Get(key []byte) []byte {
	fmt.Println("kv get", key)
	// Consume gas for storage read operation using configurable gas costs
	if err := kvs.bridge.ConsumeGas(kvs.bridge.gasCosts.StorageRead); err != nil {
		fmt.Println(err)
		return nil // Out of gas
	}

	fullKey := append(kvs.prefix, key...)
	value, err := kvs.bridge.store.Get(fullKey)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return value
}

// Set stores a value in the contract's key-value store
func (kvs *ContractKVStore) Set(key, value []byte) {
	fmt.Println("kv set", key, string(value))
	// Consume gas for storage write operation using configurable gas costs
	if err := kvs.bridge.ConsumeGas(kvs.bridge.gasCosts.StorageWrite); err != nil {
		fmt.Println(err)
		return // Out of gas - cannot complete write
	}

	fullKey := append(kvs.prefix, key...)
	kvs.bridge.store.Set(fullKey, value)
}

// Delete removes a key from the contract's key-value store
func (kvs *ContractKVStore) Delete(key []byte) {
	fmt.Println("kv delete", key)
	// Consume gas for storage delete operation using configurable gas costs
	if err := kvs.bridge.ConsumeGas(kvs.bridge.gasCosts.StorageDelete); err != nil {
		fmt.Println(err)
		return // Out of gas - cannot complete delete
	}

	fullKey := append(kvs.prefix, key...)
	kvs.bridge.store.Delete(fullKey)
}

// Iterator creates an iterator over the contract's key-value store
func (kvs *ContractKVStore) Iterator(start, end []byte) wasmvmtypes.Iterator {
	fmt.Println("iterator", start, end)
	// For simplicity, use prefix iteration with start key
	startKey := append(kvs.prefix, start...)

	iter, err := kvs.bridge.store.Iterator(startKey)
	if err != nil {
		return &ErrorIterator{err: err}
	}

	return &ContractIterator{
		iter:   iter,
		prefix: kvs.prefix,
		end:    append(kvs.prefix, end...),
	}
}

// ReverseIterator creates a reverse iterator over the contract's key-value store
func (kvs *ContractKVStore) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	fmt.Println("reviterator", start, end)
	// For simplicity, use reverse prefix iteration with start key
	startKey := append(kvs.prefix, start...)

	iter, err := kvs.bridge.store.RevIterator(startKey)
	if err != nil {
		return &ErrorIterator{err: err}
	}

	return &ContractIterator{
		iter:   iter,
		prefix: kvs.prefix,
		end:    append(kvs.prefix, end...),
	}
}

// ContractIterator wraps the store iterator and strips the contract prefix
type ContractIterator struct {
	iter   lib.IteratorI
	prefix []byte
	end    []byte
}

// Next moves the iterator to the next key-value pair
func (ci *ContractIterator) Next() {
	ci.iter.Next()
}

// Valid returns true if the iterator is pointing to a valid key-value pair
func (ci *ContractIterator) Valid() bool {
	return ci.iter.Valid()
}

// Key returns the current key with the contract prefix stripped
func (ci *ContractIterator) Key() []byte {
	key := ci.iter.Key()
	if len(key) < len(ci.prefix) {
		return nil
	}
	return key[len(ci.prefix):]
}

// Value returns the current value
func (ci *ContractIterator) Value() []byte {
	return ci.iter.Value()
}

// Close closes the iterator
func (ci *ContractIterator) Close() error {
	ci.iter.Close()
	return nil
}

// Domain returns the domain of the iterator (required by wasmvm Iterator interface)
func (ci *ContractIterator) Domain() ([]byte, []byte) {
	return ci.prefix, ci.end
}

// Error returns any iterator error (required by wasmvm Iterator interface)
func (ci *ContractIterator) Error() error {
	return nil
}

// ErrorIterator implements the Iterator interface for error cases
type ErrorIterator struct {
	err error
}

func (ei *ErrorIterator) Next()                    {}
func (ei *ErrorIterator) Valid() bool              { return false }
func (ei *ErrorIterator) Key() []byte              { return nil }
func (ei *ErrorIterator) Value() []byte            { return nil }
func (ei *ErrorIterator) Close() error             { return ei.err }
func (ei *ErrorIterator) Domain() ([]byte, []byte) { return nil, nil }
func (ei *ErrorIterator) Error() error             { return ei.err }

// NewCanopyGoAPI creates a new GoAPI implementation for Canopy
// A GoAPI implementation provides a bridge between WebAssembly (WASM) smart contracts
// and the host blockchain environment. It exposes core blockchain functionality
// that contracts need to interact with the underlying system.
func (sb *StateBridge) NewCanopyGoAPI() wasmvmtypes.GoAPI {
	// Return a GoAPI struct that implements the required interface methods
	// for address handling operations that WASM contracts can call
	return wasmvmtypes.GoAPI{
		// HumanizeAddress converts internal address representation to human-readable format
		HumanizeAddress: sb.humanizeAddress,
		// CanonicalizeAddress converts human-readable address to internal canonical format
		CanonicalizeAddress: sb.canonicalizeAddress,
		// ValidateAddress checks if an address string is valid according to blockchain rules
		ValidateAddress: sb.validateAddress,
	}
}

// validateAddress validates an address format
func (sb *StateBridge) validateAddress(addr string) (uint64, error) {
	if len(addr) == 0 {
		return 0, fmt.Errorf("address cannot be empty")
	}

	// Check if address has valid bech32 format
	hrp, decoded, err := bech32.Decode(addr)
	if err != nil {
		return 0, fmt.Errorf("invalid bech32 address: %w", err)
	}

	// Validate human-readable part (HRP) if needed
	if hrp == "" {
		return 0, fmt.Errorf("invalid address: empty human-readable part")
	}

	// Convert 5-bit groups to 8-bit bytes
	converted, err := bech32.ConvertBits(decoded, 5, 8, false)
	if err != nil {
		return 0, fmt.Errorf("invalid address: failed to convert bits: %w", err)
	}

	// Validate decoded data length (common validation)
	if len(converted) == 0 {
		return 0, fmt.Errorf("invalid address: empty decoded data")
	}

	return 0, nil
}

// canonicalizeAddress converts a human-readable address to canonical form
func (sb *StateBridge) canonicalizeAddress(humanAddr string) ([]byte, uint64, error) {
	// Parse the human-readable address
	addr, err := crypto.NewAddressFromString(humanAddr)
	if err != nil {
		return nil, 0, ErrInvalidAddress(humanAddr, err)
	}

	// Return canonical address (the raw bytes)
	return addr.Bytes(), 0, nil
}

// humanizeAddress converts a canonical address to human-readable form
func (sb *StateBridge) humanizeAddress(canonicalAddr []byte) (string, uint64, error) {
	// Convert bytes to address
	addr := crypto.NewAddressFromBytes(canonicalAddr)

	// Return human-readable format
	return addr.String(), 0, nil
}

// GasUsed returns the amount of gas used by the bridge
func (sb *StateBridge) GasUsed() uint64 {
	fmt.Println("gasused", sb.gasUsed)
	return sb.gasUsed
}

// ConsumeGas adds to the gas usage counter
func (sb *StateBridge) ConsumeGas(amount uint64) error {
	fmt.Println("consume gas", amount)
	sb.gasUsed += amount
	if sb.gasUsed > sb.gasLimit {
		return ErrOutOfGas(sb.gasUsed, sb.gasLimit)
	}
	return nil
}

// NewCanopyQuerier creates a new Querier implementation for Canopy
func (sb *StateBridge) NewCanopyQuerier() *CanopyQuerier {
	return &CanopyQuerier{bridge: sb}
}

// Helper function to create storage keys for contracts
func makeContractStorageKey(contractAddr []byte) []byte {
	prefix := []byte("contracts/state/")
	return append(prefix, contractAddr...)
}
