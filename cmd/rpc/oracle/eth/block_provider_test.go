package eth

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type mockEthClient struct {
	blocks      map[uint64]*ethtypes.Block
	receipts    map[common.Hash]*ethtypes.Receipt
	contractErr error
}

func (m *mockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error) {
	height := number.Uint64()
	if block, exists := m.blocks[height]; exists {
		return block, nil
	}
	return nil, ethereum.NotFound
}

func (m *mockEthClient) CallContract(ctx context.Context, msg ethereum.CallMsg, height *big.Int) ([]byte, error) {
	return nil, m.contractErr
}

func (m *mockEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error) {
	if block, exists := m.receipts[txHash]; exists {
		return block, nil
	}
	return nil, ethereum.NotFound
}
func createTransaction(toAddress common.Address, data []byte) *ethtypes.Transaction {
	return ethtypes.NewTransaction(
		0,
		toAddress,
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		data,
	)
}

func createEthereumBlock(height uint64, txs []*ethtypes.Transaction) *ethtypes.Block {
	header := &ethtypes.Header{
		Number:     big.NewInt(int64(height)),
		GasLimit:   8000000,
		GasUsed:    21000,
		Time:       1234567890,
		Difficulty: big.NewInt(1000),
	}
	body := &ethtypes.Body{
		Transactions: txs,
	}
	triedb := ethtrie.NewEmpty(nil)
	return ethtypes.NewBlock(header, body, nil, triedb)
}

func setupTokenCache(address common.Address, token types.TokenInfo) *ERC20TokenCache {
	cache := NewERC20TokenCache(&mockEthClient{})
	cache.cache[address.Hex()] = token
	return cache
}

func TestEthBlockProvider_getBlockByHeight(t *testing.T) {
	recipientAddress := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"

	tests := []struct {
		name          string
		height        uint64
		setupBlocks   func() map[uint64]*ethtypes.Block
		expectedError bool
		expectedTxs   int
	}{
		{
			name:   "block with no transactions",
			height: 1,
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[1] = createEthereumBlock(1, []*ethtypes.Transaction{})
				return blocks
			},
			expectedError: false,
			expectedTxs:   0,
		},
		{
			name:   "block with regular transaction",
			height: 2,
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				tx := createTransaction(common.HexToAddress(recipientAddress), []byte("regular data"))
				blocks[2] = createEthereumBlock(2, []*ethtypes.Transaction{tx})
				return blocks
			},
			expectedError: false,
			expectedTxs:   1,
		},
		{
			name:   "block with multiple transactions",
			height: 3,
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				tx1 := createTransaction(common.HexToAddress(recipientAddress), []byte("data1"))
				tx2 := createTransaction(common.HexToAddress(recipientAddress), []byte("data2"))
				blocks[3] = createEthereumBlock(3, []*ethtypes.Transaction{tx1, tx2})
				return blocks
			},
			expectedError: false,
			expectedTxs:   2,
		},
		{
			name:   "block not found",
			height: 999,
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedError: true,
			expectedTxs:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockEthClient{
				blocks: tt.setupBlocks(),
			}
			logger := lib.NewDefaultLogger()

			provider := &EthBlockProvider{
				rpcClient: mockClient,
				logger:    logger,
			}

			block, err := provider.getBlockByHeight(tt.height)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if block == nil {
				t.Errorf("expected block but got nil")
				return
			}

			if block.Number() != tt.height {
				t.Errorf("expected block number %d, got %d", tt.height, block.Number())
			}

			transactions := block.Transactions()
			if len(transactions) != tt.expectedTxs {
				t.Errorf("expected %d transactions, got %d", tt.expectedTxs, len(transactions))
			}
		})
	}
}

func TestEthBlockProvider_processSafeBlocks(t *testing.T) {
	contractAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	testToken := types.TokenInfo{
		Name:     "TestToken",
		Symbol:   "TEST",
		Decimals: 18,
	}

	tests := []struct {
		name                   string
		currentHeight          uint64
		nextHeight             uint64
		setupBlocks            func() map[uint64]*ethtypes.Block
		expectedTokenTransfers map[uint64]map[string]types.TokenTransfer
		expectedBlocks         []uint64 // expected block heights sent to channel
		expectedNext           uint64   // expected nextHeight after processing
	}{
		{
			name:          "no blocks to process - next height too high",
			currentHeight: 100,
			nextHeight:    96, // safe height would be 95 (100-5)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: 96,
		},
		{
			name:          "process single safe block",
			currentHeight: 100,
			nextHeight:    94, // safe height would be 95 (100-5)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[94] = createEthereumBlock(94, []*ethtypes.Transaction{})
				blocks[95] = createEthereumBlock(95, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{94, 95},
			expectedNext:   96,
		},
		{
			name:          "process multiple safe blocks",
			currentHeight: 110,
			nextHeight:    102, // safe height would be 105 (110-5)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				for i := uint64(102); i <= 105; i++ {
					blocks[i] = createEthereumBlock(i, []*ethtypes.Transaction{})
				}
				return blocks
			},
			expectedBlocks: []uint64{102, 103, 104, 105},
			expectedNext:   106,
		},
		{
			name:          "initialize next height when zero",
			currentHeight: 50,
			nextHeight:    0, // should be initialized to safe height (45)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[45] = createEthereumBlock(45, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{45},
			expectedNext:   46,
		},
		{
			name:          "current height too low for safe blocks",
			currentHeight: 6, // safe height would be 1 (6-5)
			nextHeight:    2, // higher than safe height, so no blocks processed
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: 2,
		},
		{
			name:          "exact safe height boundary",
			currentHeight: 10,
			nextHeight:    5, // safe height would be exactly 5 (10-5)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[5] = createEthereumBlock(5, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{5},
			expectedNext:   6,
		},
		{
			name:          "process blocks with transactions",
			currentHeight: 25,
			nextHeight:    18, // safe height would be 20 (25-5)
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				// Create blocks with some transactions
				transferData := createERC20TransferData("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", big.NewInt(1000000000000000000), []byte{})
				tx := createTransaction(contractAddress, transferData)
				blocks[18] = createEthereumBlock(18, []*ethtypes.Transaction{tx})
				blocks[19] = createEthereumBlock(19, []*ethtypes.Transaction{})
				blocks[20] = createEthereumBlock(20, []*ethtypes.Transaction{tx})
				return blocks
			},
			expectedBlocks: []uint64{18, 19, 20},
			expectedNext:   21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockEthClient{
				blocks: tt.setupBlocks(),
			}
			logger := lib.NewDefaultLogger()
			tokenCache := setupTokenCache(contractAddress, testToken)
			// Create a buffered channel to capture sent blocks
			blockChan := make(chan types.BlockI, 10)
			provider := &EthBlockProvider{
				rpcClient:       mockClient,
				erc20TokenCache: tokenCache,
				logger:          logger,
				blockChan:       blockChan,
				nextHeight:      tt.nextHeight,
			}
			provider.processSafeBlocks(tt.currentHeight)
			// Collect all blocks sent to channel
			var receivedBlocks []types.BlockI
			close(blockChan) // Close channel to stop range loop
			for block := range blockChan {
				receivedBlocks = append(receivedBlocks, block)
			}

			// Extract block numbers for comparison
			var receivedBlockNumbers []uint64
			for _, block := range receivedBlocks {
				receivedBlockNumbers = append(receivedBlockNumbers, block.Number())
			}

			// Verify expected blocks were sent
			if diff := cmp.Diff(tt.expectedBlocks, receivedBlockNumbers); diff != "" {
				t.Errorf("sent blocks mismatch (-want +got):\n%s", diff)
			}

			// Verify nextHeight was updated correctly
			if provider.nextHeight != tt.expectedNext {
				t.Errorf("expected nextHeight %d, got %d", tt.expectedNext, provider.nextHeight)
			}
		})
	}
}

func TestValidateCloseOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid close order",
			input:    []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"closeOrder":true}`),
			expected: true,
		},
		{
			name:     "valid close order with extra fields",
			input:    []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"closeOrder":true,"status":"closed","timestamp":"2023-01-01"}`),
			expected: false,
		},
		{
			name:     "invalid JSON",
			input:    []byte(`{"orderId":"123456789012345678901234567890123456789"`),
			expected: false,
		},
		{
			name:     "empty JSON",
			input:    []byte(`{}`),
			expected: false,
		},
		{
			name:     "missing orderId",
			input:    []byte(`{"chain_id":1,"closeOrder":true}`),
			expected: false,
		},
		{
			name:     "missing chain_id",
			input:    []byte(`{"orderId":"1234567890123456789012345678901234567890","closeOrder":true}`),
			expected: false,
		},
		{
			name:     "missing closeOrder",
			input:    []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1}`),
			expected: false,
		},
		{
			name:     "empty orderId",
			input:    []byte(`{"orderId":"","chain_id":1,"closeOrder":true}`),
			expected: false,
		},
		{
			name:     "null orderId",
			input:    []byte(`{"orderId":null,"chain_id":1,"closeOrder":true}`),
			expected: false,
		},
		{
			name:     "orderId as number",
			input:    []byte(`{"orderId":1234567890123456789012345678901234567890,"chain_id":1,"closeOrder":true}`),
			expected: false,
		},
		{
			name:     "wrong length orderId",
			input:    []byte(`{"orderId":"123","chain_id":1,"closeOrder":true}`),
			expected: false,
		},
		{
			name:     "just a string",
			input:    []byte(`not json`),
			expected: false,
		},
		{
			name:     "empty byte array",
			input:    []byte(``),
			expected: false,
		},
		{
			name:     "nil byte array",
			input:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCloseOrder(tt.input)
			if result != tt.expected {
				t.Errorf("validateCloseOrder() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEthBlockProvider_checkTransfer(t *testing.T) {
	// Test constants
	contract1Address := common.HexToAddress("0x8884567890123456789012345678901234567888")
	contract2Address := common.HexToAddress("0x9994567890123456789012345678901234567999")
	unknownContract := common.HexToAddress("0x0006543210987654321098765432109876543000")
	recipient := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	testToken := types.TokenInfo{
		Name:     "TestToken",
		Symbol:   "TEST",
		Decimals: 18,
	}

	// Create close order data
	orderId := "1010101010101010101010101010101010101010"
	orderIdBytes, _ := lib.StringToBytes(orderId)
	order := lib.CloseOrder{
		OrderId:    orderIdBytes,
		ChainId:    0,
		CloseOrder: true,
	}
	closeOrderBytes, _ := order.MarshalJSON()

	// Set up token cache
	tokenCache := NewERC20TokenCache(&mockEthClient{contractErr: errors.New("error")})
	tokenCache.cache[contract1Address.Hex()] = testToken

	tests := []struct {
		name                  string
		txConfig              transactionConfig
		receiptStatus         *uint64 // nil means no receipt
		expectedTokenTransfer types.TokenTransfer
		expectedExtraData     []byte
	}{
		{
			name: "valid erc20 transfer with no extra data, no token transfer",
			txConfig: transactionConfig{
				contractAddress: contract1Address,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(1000000000000000000),
				extraData:       nil,
			},
			receiptStatus:         nil,
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
		{
			name: "valid erc20 transfer with close order and successful receipt",
			txConfig: transactionConfig{
				contractAddress: contract1Address,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(500000000000000000),
				extraData:       closeOrderBytes,
			},
			receiptStatus: uint64Ptr(1),
			expectedTokenTransfer: types.TokenTransfer{
				Blockchain:       "ethereum",
				TokenInfo:        testToken,
				RecipientAddress: recipient,
				TokenAmount:      0.5,
				TokenBaseAmount:  500000000000000000,
				ContractAddress:  contract1Address.Hex(),
			},
			expectedExtraData: closeOrderBytes,
		},
		{
			name: "erc20 transfer with close order but failed transaction receipt",
			txConfig: transactionConfig{
				contractAddress: contract2Address,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(500000000000000000),
				extraData:       closeOrderBytes,
			},
			receiptStatus:         uint64Ptr(0),
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
		{
			name: "erc20 transfer with close order but receipt not found",
			txConfig: transactionConfig{
				contractAddress: contract1Address,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(500000000000000000),
				extraData:       closeOrderBytes,
			},
			receiptStatus:         nil,
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
		{
			name: "non-erc20 transaction",
			txConfig: transactionConfig{
				contractAddress: contract1Address,
				isERC20:         false,
				recipient:       "",
				amount:          nil,
				extraData:       nil,
			},
			receiptStatus:         nil,
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
		{
			name: "erc20 transfer but token info not found",
			txConfig: transactionConfig{
				contractAddress: unknownContract,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(1000000000000000000),
				extraData:       nil,
			},
			receiptStatus:         nil,
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
		{
			name: "zero amount transfer. no extra data, no token transfer",
			txConfig: transactionConfig{
				contractAddress: contract1Address,
				isERC20:         true,
				recipient:       recipient,
				amount:          big.NewInt(0),
				extraData:       nil,
			},
			receiptStatus:         nil,
			expectedTokenTransfer: types.TokenTransfer{},
			expectedExtraData:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create transaction based on test configuration
			tx := createTestTransaction(tt.txConfig)

			// Set up receipts
			receipts := make(map[common.Hash]*ethtypes.Receipt)
			if tt.receiptStatus != nil {
				receipts[tx.tx.Hash()] = &ethtypes.Receipt{Status: *tt.receiptStatus}
			}

			// Set up provider
			logger := lib.NewDefaultLogger()
			mockClient := &mockEthClient{
				receipts: receipts,
			}
			provider := &EthBlockProvider{
				rpcClient:       mockClient,
				erc20TokenCache: tokenCache,
				logger:          logger,
			}

			// Execute test
			provider.checkTransfer(tx)

			// Verify results
			cmpopts := cmpopts.IgnoreFields(types.TokenTransfer{}, "TransactionID")
			if diff := cmp.Diff(tt.expectedTokenTransfer, tx.tokenTransfer, cmpopts); diff != "" {
				t.Errorf("token transfer mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expectedExtraData, tx.extraData); diff != "" {
				t.Errorf("extra data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// transactionConfig defines the configuration for creating test transactions
type transactionConfig struct {
	contractAddress common.Address
	isERC20         bool
	recipient       string
	amount          *big.Int
	extraData       []byte
}

// createTestTransaction creates a transaction based on the provided configuration
func createTestTransaction(config transactionConfig) *Transaction {
	var data []byte

	if config.isERC20 {
		data = createERC20TransferData(config.recipient, config.amount, config.extraData)
	} else {
		data = []byte("not_erc20_data")
	}

	ethTx := createTransaction(config.contractAddress, data)
	tx, _ := NewTransaction(ethTx)
	return tx
}

// uint64Ptr returns a pointer to a uint64 value
func uint64Ptr(v uint64) *uint64 {
	return &v
}
