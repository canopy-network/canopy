package vm

import (
	"encoding/json"
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/canopy-network/canopy/lib/crypto"
)

// Querier implementation for CosmWasm contracts
type CanopyQuerier struct {
	bridge *StateBridge
}

// Ensure CanopyQuerier implements wasmvmtypes.Querier
var _ wasmvmtypes.Querier = &CanopyQuerier{}

// Query handles queries from CosmWasm contracts to the Canopy blockchain state
func (q *CanopyQuerier) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	fmt.Println("query", request, gasLimit)
	switch {
	case request.Bank != nil:
		return q.queryBank(*request.Bank)
	case request.Custom != nil:
		return q.queryCustom(request.Custom)
	case request.Staking != nil:
		return q.queryStaking(*request.Staking)
	case request.Wasm != nil:
		return q.queryWasm(*request.Wasm)
	default:
		return nil, ErrUnsupportedQuery("unknown")
	}
}

// GasConsumed returns the gas consumed by the querier
func (q *CanopyQuerier) GasConsumed() uint64 {
	return q.bridge.GasUsed()
}

// queryBank handles bank-related queries (account balances, etc.)
func (q *CanopyQuerier) queryBank(request wasmvmtypes.BankQuery) ([]byte, error) {
	switch {
	case request.Balance != nil:
		return q.queryBalance(request.Balance.Address, request.Balance.Denom)
	case request.AllBalances != nil:
		return q.queryAllBalances(request.AllBalances.Address)
	default:
		return nil, ErrUnsupportedQuery("bank")
	}
}

// queryBalance returns the balance of a specific denomination for an address
func (q *CanopyQuerier) queryBalance(address, denom string) ([]byte, error) {
	// Convert address string to canonical format
	_, err := crypto.NewAddressFromString(address)
	if err != nil {
		return nil, ErrInvalidAddress(address, err)
	}

	// For Canopy, we only have CNPY tokens, so ignore denom for now
	// In a production system, you'd validate the denom

	// Query account balance from state
	// This would need to be implemented by accessing the account state
	// For now, return a placeholder
	response := wasmvmtypes.BalanceResponse{
		Amount: wasmvmtypes.Coin{
			Denom:  "cnpy",
			Amount: "0", // Would query actual balance here
		},
	}

	return json.Marshal(response)
}

// queryAllBalances returns all token balances for an address
func (q *CanopyQuerier) queryAllBalances(address string) ([]byte, error) {
	// For Canopy, we only have CNPY tokens
	balance, err := q.queryBalance(address, "cnpy")
	if err != nil {
		return nil, err
	}

	var balanceResp wasmvmtypes.BalanceResponse
	if err := json.Unmarshal(balance, &balanceResp); err != nil {
		return nil, err
	}

	response := wasmvmtypes.AllBalancesResponse{
		Amount: []wasmvmtypes.Coin{balanceResp.Amount},
	}

	return json.Marshal(response)
}

// queryCustom handles custom queries specific to Canopy
func (q *CanopyQuerier) queryCustom(request json.RawMessage) ([]byte, error) {
	// Parse the custom query
	var customQuery struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(request, &customQuery); err != nil {
		return nil, ErrInvalidCustomQuery(err)
	}

	switch customQuery.Type {
	case "chain_info":
		return q.queryChainInfo()
	case "validator_info":
		return q.queryValidatorInfo(customQuery.Data)
	case "committee_info":
		return q.queryCommitteeInfo(customQuery.Data)
	default:
		return nil, ErrUnsupportedQuery(customQuery.Type)
	}
}

// queryChainInfo returns basic chain information
func (q *CanopyQuerier) queryChainInfo() ([]byte, error) {
	response := map[string]interface{}{
		"chain_id": q.bridge.chainID,
		"height":   q.bridge.height,
	}
	return json.Marshal(response)
}

// queryValidatorInfo returns validator information
func (q *CanopyQuerier) queryValidatorInfo(data json.RawMessage) ([]byte, error) {
	// This would query validator information from the state machine
	// Implementation depends on how validator data is stored
	return json.Marshal(map[string]interface{}{
		"message": "validator query not implemented yet",
	})
}

// queryCommitteeInfo returns committee information
func (q *CanopyQuerier) queryCommitteeInfo(data json.RawMessage) ([]byte, error) {
	// This would query committee information from the state machine
	// Implementation depends on how committee data is stored
	return json.Marshal(map[string]interface{}{
		"message": "committee query not implemented yet",
	})
}

// queryStaking handles staking-related queries
func (q *CanopyQuerier) queryStaking(request wasmvmtypes.StakingQuery) ([]byte, error) {
	// Staking queries would be implemented based on Canopy's validator/committee system
	return nil, ErrQueryNotImplemented("staking")
}

// queryWasm handles wasm-related queries (calling other contracts)
func (q *CanopyQuerier) queryWasm(request wasmvmtypes.WasmQuery) ([]byte, error) {
	switch {
	case request.Smart != nil:
		return q.querySmartContract(request.Smart.ContractAddr, request.Smart.Msg)
	case request.Raw != nil:
		return q.queryRawContract(request.Raw.ContractAddr, request.Raw.Key)
	default:
		return nil, ErrUnsupportedQuery("wasm")
	}
}

// querySmartContract executes a query on another smart contract
func (q *CanopyQuerier) querySmartContract(contractAddr string, msg json.RawMessage) ([]byte, error) {
	// This would execute a query on another contract
	// For now, return not implemented
	return nil, ErrQueryNotImplemented("smart contract")
}

// queryRawContract reads raw storage from another contract
func (q *CanopyQuerier) queryRawContract(contractAddr string, key []byte) ([]byte, error) {
	// Convert address and create storage key
	addr, err := crypto.NewAddressFromString(contractAddr)
	if err != nil {
		return nil, ErrInvalidAddress(contractAddr, err)
	}

	// Create storage key for the other contract
	storageKey := makeContractStorageKey(addr.Bytes())
	fullKey := append(storageKey, key...)

	// Query the storage
	value, err := q.bridge.store.Get(fullKey)
	if err != nil {
		return nil, ErrContractStorageQuery(err)
	}

	return value, nil
}
