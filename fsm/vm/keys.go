package vm

import (
	"encoding/binary"
)

// Storage key prefixes for contract-related data
const (
	// ContractCodePrefix stores compiled WASM bytecode
	ContractCodePrefix = "contracts/code/"
	// ContractMetadataPrefix stores contract metadata (admin, code_id, etc.)
	ContractMetadataPrefix = "contracts/metadata/"
	// ContractStatePrefix stores contract state data
	ContractStatePrefix = "contracts/state/"
	// ContractInstancePrefix stores contract instance info
	ContractInstancePrefix = "contracts/instances/"
	// ContractSequenceKey stores the next contract address sequence number
	ContractSequenceKey = "contracts/sequence"
	// CodeSequenceKey stores the next code ID sequence number
	CodeSequenceKey = "contracts/code_sequence"
)

// Storage key generators for contract-related data

// CodeKey creates a storage key for WASM bytecode
func CodeKey(codeID uint64) []byte {
	return append([]byte(ContractCodePrefix), EncodeUint64(codeID)...)
}

// ContractMetadataKey creates a storage key for contract metadata
func ContractMetadataKey(contractAddr []byte) []byte {
	return append([]byte(ContractMetadataPrefix), contractAddr...)
}

// ContractStateKey creates a storage key for contract state
func ContractStateKey(contractAddr []byte, key []byte) []byte {
	prefix := append([]byte(ContractStatePrefix), contractAddr...)
	prefix = append(prefix, '/')
	return append(prefix, key...)
}

// ContractInstanceKey creates a storage key for contract instance data
func ContractInstanceKey(contractAddr []byte) []byte {
	return append([]byte(ContractInstancePrefix), contractAddr...)
}

// SequenceKey returns the key for the contract address sequence counter
func SequenceKey() []byte {
	return []byte(ContractSequenceKey)
}

// CodeSequenceKeyBytes returns the key for the code ID sequence counter
func CodeSequenceKeyBytes() []byte {
	return []byte(CodeSequenceKey)
}

// Helper functions for encoding/decoding keys and values

// EncodeUint64 encodes a uint64 as bytes for storage
func EncodeUint64(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}

// DecodeUint64 decodes a uint64 from bytes
func DecodeUint64(bytes []byte) uint64 {
	if len(bytes) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bytes)
}

// makeContractStorageKeyInternal creates a prefixed key for contract storage
// This is used by the bridge to isolate contract storage
func makeContractStorageKeyInternal(contractAddr []byte) []byte {
	return ContractStateKey(contractAddr, nil)[:len(ContractStatePrefix)+len(contractAddr)+1]
}

// KeyRange represents a range of keys for iteration
type KeyRange struct {
	Start []byte
	End   []byte
}

// NewKeyRange creates a new key range
func NewKeyRange(start, end []byte) KeyRange {
	return KeyRange{
		Start: start,
		End:   end,
	}
}

// ContractStateRange returns a key range for all state of a specific contract
func ContractStateRange(contractAddr []byte) KeyRange {
	prefix := append([]byte(ContractStatePrefix), contractAddr...)
	prefix = append(prefix, '/')

	start := prefix
	end := append(prefix, 0xFF) // 0xFF is higher than any valid key byte

	return KeyRange{
		Start: start,
		End:   end,
	}
}

// AllContractsRange returns a key range for all contract-related keys
func AllContractsRange() KeyRange {
	start := []byte("contracts/")
	end := []byte("contracts0") // Next prefix after "contracts/"

	return KeyRange{
		Start: start,
		End:   end,
	}
}

// CodeRange returns a key range for all WASM code
func CodeRange() KeyRange {
	start := []byte(ContractCodePrefix)
	end := []byte(ContractCodePrefix[:len(ContractCodePrefix)-1] + "0") // Next prefix

	return KeyRange{
		Start: start,
		End:   end,
	}
}

// MetadataRange returns a key range for all contract metadata
func MetadataRange() KeyRange {
	start := []byte(ContractMetadataPrefix)
	end := []byte(ContractMetadataPrefix[:len(ContractMetadataPrefix)-1] + "0") // Next prefix

	return KeyRange{
		Start: start,
		End:   end,
	}
}
