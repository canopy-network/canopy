package eth

import (
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// mockEthereumServer tracks received data for verification
type mockEthereumServer struct {
	receivedData []byte
	responses    map[string]interface{}
}

// createUSDCTransferData creates ABI encoded transfer data with optional extra data
func createUSDCTransferData(to string, amount *big.Int, extraData []byte) []byte {
	// create method signature for transfer(address,uint256)
	methodSig, _ := hex.DecodeString(erc20TransferMethodID)

	// pad address to 32 bytes
	toAddr := common.HexToAddress(to)
	paddedAddr := common.LeftPadBytes(toAddr.Bytes(), 32)

	// pad amount to 32 bytes
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)

	// combine all data
	data := append(methodSig, paddedAddr...)
	data = append(data, paddedAmount...)

	// append extra data if provided
	if len(extraData) > 0 {
		data = append(data, extraData...)
	}

	return data
}

func TestParseERC20Transfer(t *testing.T) {
	// usdc contract address for testing
	usdcAddress := common.HexToAddress("0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e")
	unknownAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	tests := []struct {
		name           string
		to             *common.Address
		data           []byte
		expectedError  error
		expectedSymbol string
		expectedAmount float64
		expectedExtra  []byte
	}{
		{
			name:           "valid usdc transfer without extra data",
			to:             &usdcAddress,
			data:           createUSDCTransferData("0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6", big.NewInt(1000000), nil),
			expectedError:  nil,
			expectedSymbol: "USDC",
			expectedAmount: 1.0,
			expectedExtra:  nil,
		},
		{
			name:           "valid usdc transfer with extra data",
			to:             &usdcAddress,
			data:           createUSDCTransferData("0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6", big.NewInt(2000000), []byte("extra")),
			expectedError:  nil,
			expectedSymbol: "USDC",
			expectedAmount: 2.0,
			expectedExtra:  []byte("extra"),
		},
		{
			name:          "transaction with no data",
			to:            &usdcAddress,
			data:          []byte{},
			expectedError: ErrNotERC20Transfer,
		},
		{
			name:          "transaction with insufficient data",
			to:            &usdcAddress,
			data:          []byte{0xa9, 0x05, 0x9c},
			expectedError: ErrNotERC20Transfer,
		},
		{
			name:          "transaction with wrong method signature",
			to:            &usdcAddress,
			data:          append([]byte{0x12, 0x34, 0x56, 0x78}, make([]byte, 64)...),
			expectedError: ErrNotERC20Transfer,
		},
		{
			name:          "transaction with incomplete transfer data",
			to:            &usdcAddress,
			data:          append([]byte{0xa9, 0x05, 0x9c, 0xbb}, make([]byte, 32)...),
			expectedError: ErrInvalidTransactionData,
		},
		{
			name:          "transaction to unknown contract",
			to:            &unknownAddress,
			data:          createUSDCTransferData("0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6", big.NewInt(1000000), nil),
			expectedError: ErrContractNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock transaction
			tx := types.NewTransaction(0, *tt.to, big.NewInt(0), 21000, big.NewInt(1000000000), tt.data)

			// call function under test
			tokenTransfer, extraData, err := ParseERC20Transfer(tx)

			// verify error expectation
			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			// verify no error occurred
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// verify token symbol
			if tokenTransfer.TokenSymbol != tt.expectedSymbol {
				t.Errorf("expected symbol %s, got %s", tt.expectedSymbol, tokenTransfer.TokenSymbol)
			}

			// verify token amount
			if tokenTransfer.TokenAmount != tt.expectedAmount {
				t.Errorf("expected amount %f, got %f", tt.expectedAmount, tokenTransfer.TokenAmount)
			}

			// verify blockchain field
			if tokenTransfer.Blockchain != "Ethereum" {
				t.Errorf("expected blockchain Ethereum, got %s", tokenTransfer.Blockchain)
			}

			// verify transaction id
			if tokenTransfer.TransactionID != tx.Hash().Hex() {
				t.Errorf("expected transaction id %s, got %s", tx.Hash().Hex(), tokenTransfer.TransactionID)
			}

			// verify extra data
			if len(tt.expectedExtra) == 0 && len(extraData) != 0 {
				t.Errorf("expected no extra data, got %v", extraData)
			} else if len(tt.expectedExtra) > 0 {
				if string(extraData) != string(tt.expectedExtra) {
					t.Errorf("expected extra data %v, got %v", tt.expectedExtra, extraData)
				}
			}
		})
	}
}

func TestTransferERC20WithData(t *testing.T) {
	tests := []struct {
		name            string
		contractAddress string
		receiveAddress  string
		privateKey      string
		transferAmount  int
		data            []byte
		expectedError   error
		setupMock       func(*mockEthereumServer)
	}{
		{
			name:            "valid transfer without extra data",
			contractAddress: "0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e",
			receiveAddress:  "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
			privateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			transferAmount:  1000000,
			data:            nil,
			expectedError:   nil,
			setupMock: func(mock *mockEthereumServer) {
				// setup standard responses for ethereum client calls
				mock.responses["eth_getTransactionCount"] = "0x0"
				mock.responses["eth_gasPrice"] = "0x3b9aca00"
				mock.responses["eth_estimateGas"] = "0x5208"
				mock.responses["net_version"] = "1"
				mock.responses["eth_sendRawTransaction"] = "0x1234567890abcdef"
			},
		},
		{
			name:            "valid transfer with extra data",
			contractAddress: "0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e",
			receiveAddress:  "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
			privateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			transferAmount:  2000000,
			data:            []byte("test extra data"),
			expectedError:   nil,
			setupMock: func(mock *mockEthereumServer) {
				// setup standard responses for ethereum client calls
				mock.responses["eth_getTransactionCount"] = "0x0"
				mock.responses["eth_gasPrice"] = "0x3b9aca00"
				mock.responses["eth_estimateGas"] = "0x5208"
				mock.responses["net_version"] = "1"
				mock.responses["eth_sendRawTransaction"] = "0x1234567890abcdef"
			},
		},
		{
			name:            "invalid private key",
			contractAddress: "0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e",
			receiveAddress:  "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
			privateKey:      "invalid",
			transferAmount:  1000000,
			data:            nil,
			expectedError:   ErrInvalidPrivateKey,
			setupMock:       func(mock *mockEthereumServer) {},
		},
		{
			name:            "invalid contract address",
			contractAddress: "invalid",
			receiveAddress:  "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
			privateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			transferAmount:  1000000,
			data:            nil,
			expectedError:   ErrInvalidAddress,
			setupMock:       func(mock *mockEthereumServer) {},
		},
		{
			name:            "invalid receive address",
			contractAddress: "0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e",
			receiveAddress:  "invalid",
			privateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			transferAmount:  1000000,
			data:            nil,
			expectedError:   ErrInvalidAddress,
			setupMock:       func(mock *mockEthereumServer) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock ethereum server
			mock := &mockEthereumServer{
				responses: make(map[string]interface{}),
			}
			tt.setupMock(mock)

			// create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// capture received data for verification
				body := make([]byte, r.ContentLength)
				r.Body.Read(body)
				mock.receivedData = body

				// return appropriate response based on method
				response := `{"jsonrpc":"2.0","id":1,"result":"0x0"}`
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(response))
			}))
			defer server.Close()

			// create ethereum client connected to mock server
			rpcClient, err := rpc.Dial(server.URL)
			if err != nil {
				t.Fatalf("failed to create rpc client: %v", err)
			}
			client := ethclient.NewClient(rpcClient)

			// call function under test
			err = TransferERC20WithData(client, tt.contractAddress, tt.receiveAddress, tt.privateKey, tt.transferAmount, tt.data)

			// verify error expectation
			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			// verify no error occurred for successful cases
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// verify that data was sent to mock server
			if len(mock.receivedData) == 0 {
				t.Error("expected data to be sent to mock server")
			}

			// verify extra data was included if provided
			if len(tt.data) > 0 {
				// check that received data contains our extra data
				if !strings.Contains(string(mock.receivedData), hex.EncodeToString(tt.data)) {
					t.Errorf("expected extra data %s to be included in request", hex.EncodeToString(tt.data))
				}
			}
		})
	}
}
