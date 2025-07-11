package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

const (
	testWasmFile = "/home/enielson/go/src/cosmwasm/artifacts/queue.wasm"
)

func createTestVM(t *testing.T) (*VM, func()) {
	tmpDir, err := os.MkdirTemp("", "vm-test-*")
	require.NoError(t, err)

	config := Config{
		DataDir: tmpDir,
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
		MemoryLimit: 32,
		CacheSize:   10,
	}

	vm, libErr := NewVM(config, lib.NewDefaultLogger())
	require.NoError(t, libErr)
	require.NotNil(t, vm)

	cleanup := func() {
		vm.Close()
		os.RemoveAll(tmpDir)
	}

	return vm, cleanup
}

func wasmCodeFromFile(t *testing.T, filename string) []byte {
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", filename, err)
	}

	return content
}

func createTestWasmCode(t *testing.T, content []byte) []byte {
	if content == nil {
		// minimal valid wasm module
		return []byte{
			0x00, 0x61, 0x73, 0x6d, // magic
			0x01, 0x00, 0x00, 0x00, // version
		}
	}
	return content
}

// generateContractAddress generates a new contract address
func generateContractAddress(sender []byte, codeID uint64) []byte {
	// simple address generation - in production use more sophisticated method
	hash := crypto.Hash(append(sender, EncodeUint64(codeID)...))
	return hash[:crypto.AddressSize]
}

// createContractEnv creates the execution environment for a contract
func createContractEnv(height uint64, contractAddr []byte) wasmvmtypes.Env {
	// Return a new environment structure for contract execution
	return wasmvmtypes.Env{
		// Create block information section
		Block: wasmvmtypes.BlockInfo{
			// Set the current block height from state machine
			Height: height,
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
func createMessageInfo(sender []byte, funds []*wasmvmtypes.Coin) wasmvmtypes.MessageInfo {
	// Convert funds to wasmvm format
	var wasmFunds []wasmvmtypes.Coin
	for _, fund := range funds {
		wasmFunds = append(wasmFunds, wasmvmtypes.Coin{
			Denom:  fund.Denom,
			Amount: fund.Amount,
		})
	}
	// return message info
	return wasmvmtypes.MessageInfo{
		Sender: crypto.NewAddressFromBytes(sender).String(),
		Funds:  wasmFunds,
	}
}

func TestStoreCode(t *testing.T) {
	tests := []struct {
		name           string
		wasmCode       []byte
		expectError    bool
		expectedErrMsg string
		setupVM        func() (*VM, func())
	}{
		{
			name:        "valid wasm code",
			wasmCode:    wasmCodeFromFile(t, "/home/enielson/go/src/cosmwasm/artifacts/queue.wasm"),
			expectError: false,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
		},
		{
			name:           "nil wasm code",
			wasmCode:       nil,
			expectError:    true,
			expectedErrMsg: "failed to compile WASM code",
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
		},
		{
			name:           "empty wasm code",
			wasmCode:       []byte{},
			expectError:    true,
			expectedErrMsg: "failed to compile WASM code",
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
		},
		{
			name:           "invalid wasm code",
			wasmCode:       []byte{0x01, 0x02, 0x03},
			expectError:    true,
			expectedErrMsg: "failed to compile WASM code",
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
		},
		{
			name:           "closed vm",
			wasmCode:       createTestWasmCode(t, nil),
			expectError:    true,
			expectedErrMsg: "VM is closed",
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				vm.Close()
				return vm, cleanup
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			codeID, err := vm.StoreCode(tt.wasmCode)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				require.Equal(t, uint64(0), codeID)
			} else {
				require.NoError(t, err)
				require.Greater(t, codeID, uint64(0))

				// verify code was cached
				require.True(t, vm.HasCode(codeID))

				// verify code can be retrieved
				retrievedCode, getErr := vm.GetCode(codeID)
				require.NoError(t, getErr)
				require.Equal(t, tt.wasmCode, retrievedCode)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name           string
		codeID         uint64
		setupVM        func() (*VM, func())
		expectError    bool
		expectedErrMsg string
		expectedCode   []byte
	}{
		{
			name:   "valid code in cache",
			codeID: 1,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			expectError:  false,
			expectedCode: wasmCodeFromFile(t, testWasmFile),
		},
		{
			name:   "non-existent code ID",
			codeID: 999,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError:    true,
			expectedErrMsg: "WASM code not found",
		},
		{
			name:   "zero code ID",
			codeID: 0,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError:    true,
			expectedErrMsg: "WASM code not found",
		},
		{
			name:   "code exists but cache miss",
			codeID: 1,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				// remove from cache to simulate cache miss
				vm.cache.RemoveCode(codeID)
				return vm, cleanup
			},
			expectError:    true,
			expectedErrMsg: "WASM code not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			code, err := vm.GetCode(tt.codeID)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				require.Nil(t, code)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCode, code)
			}
		})
	}
}

// contractExecutionContext holds the common execution context for contract operations
type contractExecutionContext struct {
	env      wasmvmtypes.Env         // contract environment
	info     wasmvmtypes.MessageInfo // message info
	gasLimit uint64
	bridge   *StateBridge
	kvStore  wasmvmtypes.KVStore
	goAPI    wasmvmtypes.GoAPI
	querier  wasmvmtypes.Querier
}

// creates a new in-memory store for testing.
func testStore() (*store.Store, *badger.DB, func()) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR))
	if err != nil {
		panic(err)
	}
	store, err := store.NewStoreWithDB(db, nil, lib.NewDefaultLogger(), true)
	return store, db, func() { store.Close() }
}

// setupContractExecution handles the common setup for contract instantiation and execution
func setupContractExecution(sender, contractAddr []byte, funds []*wasmvmtypes.Coin, msgName string, minGasLimit uint64) (*contractExecutionContext, lib.ErrorI) {
	// create execution environment
	env := createContractEnv(1, contractAddr)
	info := createMessageInfo(sender, funds)

	gasLimit := uint64(1000000000)

	gasCosts := &GasCosts{
		StorageRead:   1,
		StorageWrite:  2,
		StorageDelete: 3,
	}

	store, _, _ := testStore()
	// create state bridge and related components
	bridge := NewStateBridge(store, 1, 1, gasLimit, gasCosts)
	kvStore := bridge.NewContractKVStore(contractAddr)
	goAPI := bridge.NewCanopyGoAPI()
	querier := bridge.NewCanopyQuerier()

	return &contractExecutionContext{
		env:      env,
		info:     info,
		gasLimit: gasLimit,
		bridge:   bridge,
		kvStore:  kvStore,
		goAPI:    goAPI,
		querier:  querier,
	}, nil
}

func TestInstantiateContract(t *testing.T) {
	tests := []struct {
		name           string
		codeID         uint64
		instantiateMsg []byte
		gasLimit       uint64
		setupVM        func() (*VM, func())
		setupContext   func() (*contractExecutionContext, lib.ErrorI)
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:           "valid contract instantiation",
			codeID:         1,
			instantiateMsg: []byte(`{"count": 0}`),
			gasLimit:       1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "instantiate", 1000000000)
			},
			expectError: false,
		},
		{
			name:           "non-existent code ID",
			codeID:         999,
			instantiateMsg: []byte(`{"count": 0}`),
			gasLimit:       1000000000,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 999)
				return setupContractExecution(sender, contractAddr, nil, "instantiate", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Nil argument: checksum",
		},
		{
			name:           "empty instantiate message",
			codeID:         1,
			instantiateMsg: []byte{},
			gasLimit:       1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "instantiate", 1000000000)
			},
			// expectError:    true,
			expectedErrMsg: "nil checksum",
		},
		{
			name:           "zero gas limit",
			codeID:         1,
			instantiateMsg: []byte(`{"count": 0}`),
			gasLimit:       0,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "instantiate", 0)
			},
			expectError:    true,
			expectedErrMsg: "Out of gas",
		},
		{
			name:           "closed VM",
			codeID:         1,
			instantiateMsg: []byte(`{"count": 0}`),
			gasLimit:       1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				vm.Close()
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "instantiate", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "VM is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			ctx, err := tt.setupContext()
			require.NoError(t, err)

			result, gasUsed, vmErr := vm.InstantiateContract(
				tt.codeID,
				ctx.env,
				ctx.info,
				tt.instantiateMsg,
				ctx.kvStore,
				ctx.goAPI,
				ctx.querier,
				tt.gasLimit,
			)

			if tt.expectError {
				require.Error(t, vmErr)
				require.Contains(t, vmErr.Error(), tt.expectedErrMsg)
				require.Nil(t, result)
				// require.Greater(t, gasUsed, uint64(0))
			} else {
				require.NoError(t, vmErr)
				require.NotNil(t, result)
				require.Greater(t, gasUsed, uint64(0))
			}
		})
	}
}

// ExecuteMsg represents the execute message variants
type ExecuteMsg struct {
	// Enqueue will add some value to the end of list
	Enqueue *EnqueueMsg `json:"enqueue,omitempty"`
	// Dequeue will remove value from start of the list
	Dequeue *struct{} `json:"dequeue,omitempty"`
}

type EnqueueMsg struct {
	Value int32 `json:"value"`
}

func TestExecuteContract(t *testing.T) {
	msgBytes := func(value int32) []byte {
		msg, e := json.Marshal(&ExecuteMsg{
			Enqueue: &EnqueueMsg{
				Value: value,
			},
		})
		if e != nil {
			t.Errorf("Error unmarshalling: %v", e)
		}
		return msg
	}

	tests := []struct {
		name           string
		codeID         uint64
		executeMsg     []byte
		gasLimit       uint64
		setupVM        func() (*VM, func())
		setupContext   func() (*contractExecutionContext, lib.ErrorI)
		expectError    bool
		expectedErrMsg string
		expectedKV     map[string]string // expected key-value pairs to check in storage
	}{
		{
			name:       "valid contract execution",
			codeID:     1,
			executeMsg: msgBytes(1),
			gasLimit:   1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "execute", 1000000000)
			},
			expectError: false,
			expectedKV: map[string]string{
				string([]byte{0, 0, 0, 0}): `{"value":1}`,
			},
		},
		{
			name:       "non-existent code ID",
			codeID:     999,
			executeMsg: msgBytes(1),
			gasLimit:   1000000000,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 999)
				return setupContractExecution(sender, contractAddr, nil, "execute", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Nil argument: checksum",
		},
		{
			name:       "empty execute message",
			codeID:     1,
			executeMsg: []byte{},
			gasLimit:   1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "execute", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Error parsing into type",
		},
		{
			name:       "invalid JSON message",
			codeID:     1,
			executeMsg: []byte(`{"invalid_json": `),
			gasLimit:   1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "execute", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Error parsing into type",
		},
		{
			name:       "zero gas limit",
			codeID:     1,
			executeMsg: msgBytes(1),
			gasLimit:   0,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "execute", 0)
			},
			expectError:    true,
			expectedErrMsg: "Out of gas",
		},
		{
			name:       "closed VM",
			codeID:     1,
			executeMsg: msgBytes(1),
			gasLimit:   1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				vm.Close()
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "execute", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "VM is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			ctx, err := tt.setupContext()
			require.NoError(t, err)

			result, gasUsed, vmErr := vm.ExecuteContract(
				tt.codeID,
				ctx.env,
				ctx.info,
				tt.executeMsg,
				ctx.kvStore,
				ctx.goAPI,
				ctx.querier,
				tt.gasLimit,
			)
			if result != nil && tt.expectError { // handle errors present in the result from the contract
				require.Contains(t, result.Err, tt.expectedErrMsg)
			} else if tt.expectError { // handle standard error return value
				require.Error(t, vmErr)
				require.Contains(t, vmErr.Error(), tt.expectedErrMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, vmErr)
				require.NotNil(t, result)
				require.Greater(t, gasUsed, uint64(0))
				v := ctx.kvStore.Get([]byte{0, 0, 0, 0})
				fmt.Println("v", string(v))
				// Check expected key-value pairs
				if tt.expectedKV != nil {
					for key, expectedValue := range tt.expectedKV {
						actualValue := ctx.kvStore.Get([]byte(key))
						require.Equal(t, expectedValue, string(actualValue))
					}
				}
			}
		})
	}
}

// QueryMsg represents the query message variants
type QueryMsg struct {
	// List will return the current state of the queue
	List *struct{} `json:"list,omitempty"`
	// Sum will return the sum of all values in the queue
	Sum *struct{} `json:"sum,omitempty"`
}

func TestQueryContract(t *testing.T) {
	queryMsgBytes := func(query string) []byte {
		var msg QueryMsg
		switch query {
		case "list":
			msg.List = &struct{}{}
		case "sum":
			msg.Sum = &struct{}{}
		}
		msgBytes, err := json.Marshal(&msg)
		if err != nil {
			t.Errorf("Error marshalling query message: %v", err)
		}
		return msgBytes
	}

	tests := []struct {
		name           string
		codeID         uint64
		queryMsg       []byte
		gasLimit       uint64
		setupVM        func() (*VM, func())
		setupContext   func() (*contractExecutionContext, lib.ErrorI)
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:     "valid contract query",
			codeID:   1,
			queryMsg: queryMsgBytes("list"),
			gasLimit: 1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "query", 1000000000)
			},
			expectError: false,
		},
		{
			name:     "non-existent code ID",
			codeID:   999,
			queryMsg: queryMsgBytes("list"),
			gasLimit: 1000000000,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 999)
				return setupContractExecution(sender, contractAddr, nil, "query", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Nil argument: checksum",
		},
		{
			name:     "empty query message",
			codeID:   1,
			queryMsg: []byte{},
			gasLimit: 1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "query", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Error parsing into type",
		},
		{
			name:     "invalid JSON message",
			codeID:   1,
			queryMsg: []byte(`{"invalid_json": `),
			gasLimit: 1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "query", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "Error parsing into type",
		},
		{
			name:     "zero gas limit",
			codeID:   1,
			queryMsg: queryMsgBytes("list"),
			gasLimit: 0,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "query", 0)
			},
			expectError:    true,
			expectedErrMsg: "Out of gas",
		},
		{
			name:     "closed VM",
			codeID:   1,
			queryMsg: queryMsgBytes("list"),
			gasLimit: 1000000000,
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				wasmCode := wasmCodeFromFile(t, testWasmFile)
				codeID, err := vm.StoreCode(wasmCode)
				require.NoError(t, err)
				require.Equal(t, uint64(1), codeID)
				vm.Close()
				return vm, cleanup
			},
			setupContext: func() (*contractExecutionContext, lib.ErrorI) {
				sender := []byte("test-sender")
				contractAddr := generateContractAddress(sender, 1)
				return setupContractExecution(sender, contractAddr, nil, "query", 1000000000)
			},
			expectError:    true,
			expectedErrMsg: "VM is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			ctx, err := tt.setupContext()
			require.NoError(t, err)

			result, gasUsed, vmErr := vm.QueryContract(
				tt.codeID,
				ctx.env,
				tt.queryMsg,
				ctx.kvStore,
				ctx.goAPI,
				ctx.querier,
				tt.gasLimit,
			)

			if result != nil && tt.expectError { // handle errors present in the result from the contract
				require.Contains(t, result.Err, tt.expectedErrMsg)
			} else if tt.expectError {
				require.Error(t, vmErr)
				require.Contains(t, vmErr.Error(), tt.expectedErrMsg)
				require.Nil(t, result)
			} else {
				require.NoError(t, vmErr)
				require.NotNil(t, result)
				require.Greater(t, gasUsed, uint64(0))
			}
		})
	}
}

func TestValidateContract(t *testing.T) {
	tests := []struct {
		name           string
		wasmCode       []byte
		setupVM        func() (*VM, func())
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:     "valid wasm code",
			wasmCode: wasmCodeFromFile(t, testWasmFile),
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError: false,
		},
		{
			name:     "nil wasm code",
			wasmCode: nil,
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError:    true,
			expectedErrMsg: "WASM validation failed",
		},
		{
			name:     "empty wasm code",
			wasmCode: []byte{},
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError:    true,
			expectedErrMsg: "WASM validation failed",
		},
		{
			name:     "invalid wasm code",
			wasmCode: []byte{0x01, 0x02, 0x03},
			setupVM: func() (*VM, func()) {
				return createTestVM(t)
			},
			expectError:    true,
			expectedErrMsg: "WASM validation failed",
		},
		{
			name:     "closed VM",
			wasmCode: wasmCodeFromFile(t, testWasmFile),
			setupVM: func() (*VM, func()) {
				vm, cleanup := createTestVM(t)
				vm.Close()
				return vm, cleanup
			},
			expectError:    true,
			expectedErrMsg: "VM is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm, cleanup := tt.setupVM()
			defer cleanup()

			err := vm.ValidateContract(tt.wasmCode)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
