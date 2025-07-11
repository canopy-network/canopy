package fsm

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/canopy-network/canopy/fsm/vm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// HELPER FUNCTIONS FOR CONTRACT OPERATIONS

// generateContractAddress generates a new contract address
func (s *StateMachine) generateContractAddress(sender []byte, codeID uint64) []byte {
	// simple address generation - in production use more sophisticated method
	hash := crypto.Hash(append(sender, vm.EncodeUint64(codeID)...))
	return hash[:crypto.AddressSize]
}

// createContractEnv creates the execution environment for a contract
func (s *StateMachine) createContractEnv(contractAddr []byte) wasmvmtypes.Env {
	// Return a new environment structure for contract execution
	return wasmvmtypes.Env{
		// Create block information section
		Block: wasmvmtypes.BlockInfo{
			// Set the current block height from state machine
			Height: s.height,
			// Set the current timestamp in nanoseconds since Unix epoch
			Time: wasmvmtypes.Uint64(uint64(time.Now().UnixNano())),
			// Set the chain identifier (hardcoded for now)
			ChainID: "canopy-1", // TODO: get from config
		},
		// Create contract information section
		Contract: wasmvmtypes.ContractInfo{
			// Convert contract address bytes to string format
			Address: crypto.NewAddressFromBytes(contractAddr).String(),
		},
		// Create transaction information section
		Transaction: &wasmvmtypes.TransactionInfo{
			// Set transaction index to 0 (placeholder value)
			Index: 0, // TODO: get transaction index
		},
	}
}

// createMessageInfo creates message info for contract execution
func (s *StateMachine) createMessageInfo(sender []byte, funds []*Coin) wasmvmtypes.MessageInfo {
	// Convert funds to wasmvm format
	var wasmFunds []wasmvmtypes.Coin
	for _, fund := range funds {
		wasmFunds = append(wasmFunds, wasmvmtypes.Coin{
			Denom:  fund.Denom,
			Amount: strconv.FormatUint(fund.Amount, 10),
		})
	}
	// return message info
	return wasmvmtypes.MessageInfo{
		Sender: crypto.NewAddressFromBytes(sender).String(),
		Funds:  wasmFunds,
	}
}

// ContractMetadata represents stored contract metadata
type ContractMetadata struct {
	CodeId uint64
	Admin  []byte
	Label  string
}

// storeCode stores WASM code
func (s *StateMachine) storeCode(codeID uint64, uploader []byte, wasmCode []byte) lib.ErrorI {
	key := vm.CodeKey(codeID)
	return s.store.Set(key, wasmCode)
}

// storeContractMetadata stores metadata for a contract instance
func (s *StateMachine) storeContractMetadata(contractAddr []byte, codeID uint64, admin []byte, label string) lib.ErrorI {
	metadata := &ContractMetadata{
		CodeId: codeID,
		Admin:  admin,
		Label:  label,
	}

	data, err := lib.MarshalJSON(metadata)
	if err != nil {
		return lib.NewError(lib.CodeMarshal, "failed to marshal contract metadata", err.Error())
	}

	key := vm.ContractMetadataKey(contractAddr)
	return s.store.Set(key, data)
}

// getContractMetadata retrieves contract metadata
func (s *StateMachine) getContractMetadata(contractAddr []byte) (*ContractMetadata, lib.ErrorI) {
	key := vm.ContractMetadataKey(contractAddr)
	data, err := s.store.Get(key)
	if err != nil {
		return nil, lib.NewError(lib.CodeMarshal, "contract not found", err.Error())
	}

	var metadata ContractMetadata
	if err := lib.UnmarshalJSON(data, &metadata); err != nil {
		return nil, lib.NewError(lib.CodeMarshal, "failed to unmarshal contract metadata", err.Error())
	}

	return &metadata, nil
}

// setContractMetadata updates contract metadata
func (s *StateMachine) setContractMetadata(contractAddr []byte, metadata *ContractMetadata) lib.ErrorI {
	data, err := lib.MarshalJSON(metadata)
	if err != nil {
		return lib.NewError(lib.CodeMarshal, "failed to marshal contract metadata", err.Error())
	}

	key := vm.ContractMetadataKey(contractAddr)
	return s.store.Set(key, data)
}

// GetContractCode returns all contracts from the store under the contracts/code/ key
func (s *StateMachine) GetContractCode() (map[string]lib.HexBytes, lib.ErrorI) {
	contracts := make(map[string]lib.HexBytes)
	// create byte prefix
	prefix := []byte(vm.ContractCodePrefix)
	// create an iterator for keys with the contracts/metadata prefix
	it, err := s.store.Iterator(prefix)
	if err != nil {
		return nil, lib.NewError(lib.CodeMarshal, "failed to create iterator for contracts", err.Error())
	}
	defer it.Close()
	// iterate through all matching keys
	for ; it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()
		fmt.Printf("DEBUG: Found key: %s, value length: %d\n", string(key), len(value))
		// extract contract address from key (remove the prefix)
		codeId := key[len(prefix):]
		// store in result map using hex-encoded address as key
		contracts[hex.EncodeToString(codeId)] = value
	}

	return contracts, nil
}
