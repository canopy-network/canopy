package eth

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/google/go-cmp/cmp"
)

const (
	// nameFunction is the function signature for name()
	nameFunction = "0x06fdde03"
	// symbolFunction is the function signature for symbol()
	symbolFunction = "0x95d89b41"
	// decimalsFunction is the function signature for decimals()
	decimalsFunction = "0x313ce567"
)

// mockContractCaller implements ContractCaller for testing
type mockContractCaller struct {
	// How many times CallContract was called
	numCalls int
	// map for name responses for each contract address
	names map[string][]byte
	// map for symbol responses for each contract address
	symbols map[string][]byte
	// map for decimal responses for each contract address
	decimals map[string][]byte
}

// CallContract implements ContractCaller interface
func (m *mockContractCaller) CallContract(ctx context.Context, msg ethereum.CallMsg, height *big.Int) ([]byte, error) {
	// increment call counter
	m.numCalls++
	// get contract address as string
	address := strings.ToLower(msg.To.Hex())
	// decode function signature from call data
	if len(msg.Data) < 4 {
		return nil, ErrInvalidTransactionData
	}
	// get function signature
	functionSig := "0x" + lib.BytesToString(msg.Data[:4])
	// return appropriate response based on function signature
	switch functionSig {
	case nameFunction:
		if response, exists := m.names[address]; exists {
			return response, nil
		}
	case symbolFunction:
		if response, exists := m.symbols[address]; exists {
			return response, nil
		}
	case decimalsFunction:
		if response, exists := m.decimals[address]; exists {
			return response, nil
		}
	}
	// return error if no response found
	return nil, ErrContractNotFound
}

// buildContractResponse builds a map with contract address and response data
func buildContractResponse(contract, function, response string) map[string][]byte {
	// create response map
	responseMap := make(map[string][]byte)
	// encode response based on function type
	var encodedResponse []byte
	if function == nameFunction || function == symbolFunction {
		// encode string response
		encodedResponse = encodeString(response)
	} else if function == decimalsFunction {
		// encode uint8 response
		encodedResponse = encodeUint8(response)
	}
	// add response to map
	responseMap[strings.ToLower(contract)] = encodedResponse
	return responseMap
}

// encodeString encodes a string for ethereum contract response
func encodeString(s string) []byte {
	// create 64 byte buffer for offset and length
	result := make([]byte, 64)
	// set offset to 32 (0x20)
	result[31] = 0x20
	// set length
	length := len(s)
	result[63] = byte(length)
	// append string data
	result = append(result, []byte(s)...)
	// pad to 32 byte boundary
	for len(result)%32 != 0 {
		result = append(result, 0)
	}
	return result
}

// encodeUint8 encodes a uint8 for ethereum contract response
func encodeUint8(s string) []byte {
	// create 32 byte buffer
	result := make([]byte, 32)
	// parse string to uint8
	if s == "18" {
		result[31] = 18
	} else if s == "6" {
		result[31] = 6
	}
	return result
}

func TestERC20TokenCache_TokenInfo(t *testing.T) {
	// define test contract addresses
	usdcAddress := "0xa0b86a33e6441e6c7c5c8c8c8c8c8c8c8c8c8c8c"
	invalidAddress := "invalid"
	// define test cases
	tests := []struct {
		name             string
		contractAddress  string
		initialCache     map[string]types.TokenInfo
		mockCaller       *mockContractCaller
		expectedResult   types.TokenInfo
		expectedCache    map[string]types.TokenInfo
		expectedNumCalls int
		expectError      bool
	}{
		{
			name:            "should call contract and return token info for USDC",
			contractAddress: usdcAddress,
			mockCaller: &mockContractCaller{
				names:    buildContractResponse(usdcAddress, nameFunction, "USD Coin"),
				symbols:  buildContractResponse(usdcAddress, symbolFunction, "USDC"),
				decimals: buildContractResponse(usdcAddress, decimalsFunction, "6"),
			},
			expectedResult: types.TokenInfo{
				Name:     "USD Coin",
				Symbol:   "USDC",
				Decimals: 6,
			},
			expectedCache: map[string]types.TokenInfo{
				usdcAddress: {
					Name:     "USD Coin",
					Symbol:   "USDC",
					Decimals: 6,
				},
			},
			expectedNumCalls: 3,
			expectError:      false,
		},
		{
			name: "should return cached token info for USDC",
			initialCache: map[string]types.TokenInfo{
				usdcAddress: {
					Name:     "USD Coin",
					Symbol:   "USDC",
					Decimals: 6,
				},
			},
			contractAddress: usdcAddress,
			mockCaller: &mockContractCaller{
				names:    buildContractResponse(usdcAddress, nameFunction, "USD Coin"),
				symbols:  buildContractResponse(usdcAddress, symbolFunction, "USDC"),
				decimals: buildContractResponse(usdcAddress, decimalsFunction, "6"),
			},
			expectedResult: types.TokenInfo{
				Name:     "USD Coin",
				Symbol:   "USDC",
				Decimals: 6,
			},
			expectedCache: map[string]types.TokenInfo{
				usdcAddress: {
					Name:     "USD Coin",
					Symbol:   "USDC",
					Decimals: 6,
				},
			},
			expectedNumCalls: 0,
			expectError:      false,
		},
		{
			name:            "should return error for invalid address",
			contractAddress: invalidAddress,
			mockCaller: &mockContractCaller{
				names:    make(map[string][]byte),
				symbols:  make(map[string][]byte),
				decimals: make(map[string][]byte),
			},
			expectedResult:   types.TokenInfo{},
			expectedNumCalls: 1,
			expectError:      true,
		},
		{
			name:            "should return error when contract not found",
			contractAddress: "0xC2c86a33E6441E6C7C5c8c8c8c8c8c8c8c8c8c8c",
			mockCaller: &mockContractCaller{
				names:    make(map[string][]byte),
				symbols:  make(map[string][]byte),
				decimals: make(map[string][]byte),
			},
			expectedResult:   types.TokenInfo{},
			expectedNumCalls: 1,
			expectError:      true,
		},
	}
	// run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create new token cache with mock caller
			cache := ERC20TokenCache{
				client: tt.mockCaller,
			}
			// set up initial cache
			if tt.initialCache != nil {
				cache.cache = tt.initialCache
			} else {
				cache.cache = make(map[string]types.TokenInfo)
			}
			// call TokenInfo method
			result, err := cache.TokenInfo(context.Background(), tt.contractAddress)
			// verify CallContract was called the expected number of times
			if tt.mockCaller.numCalls != tt.expectedNumCalls {
				t.Errorf("expected number of calls %d, got %d", tt.expectedNumCalls, tt.mockCaller.numCalls)
			}
			// check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			// check for unexpected error
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Compare the expected cache
			if diff := cmp.Diff(tt.expectedCache, cache.cache); diff != "" {
				t.Errorf("cache mismatch (-expected +actual):\n%s", diff)
			}
			// Compare the expected results
			if diff := cmp.Diff(tt.expectedResult, result); diff != "" {
				t.Errorf("result mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}
