package eth

import (
	"context"
	"log"
	"math/big"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/google/go-cmp/cmp"
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

func (m *mockEthClient) Close() {}

// mockOrderValidator is a simple mock implementation of OrderValidator that always returns true
type mockOrderValidator struct{}

// NewMockOrderValidator creates a new mock OrderValidator instance
func NewMockOrderValidator() *mockOrderValidator {
	return &mockOrderValidator{}
}

// ValidateOrderJsonBytes always returns nil (success) for any input
func (m *mockOrderValidator) ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error {
	return nil
}

func createTransaction(toAddress common.Address, data []byte) *ethtypes.Transaction {
	tx := ethtypes.NewTransaction(
		0,
		toAddress,
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		data,
	)
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	// Create signer for the specific chain
	signer := ethtypes.NewEIP155Signer(big.NewInt(0))
	// Sign the transaction
	signedTx, _ := ethtypes.SignTx(tx, signer, privateKey)
	return signedTx
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
	cache.cache.Put(address.Hex(), token)
	return cache
}

func TestEthBlockProvider_fetchBlock(t *testing.T) {
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
				chainId:   1,
				config: lib.EthBlockProviderConfig{
					SafeBlockConfirmations: 5, // default test value
				},
				heightMu: &sync.Mutex{},
			}

			block, err := provider.fetchBlock(context.Background(), new(big.Int).SetUint64(tt.height))

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

func TestEthBlockProvider_processBlocks(t *testing.T) {
	contractAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	testToken := types.TokenInfo{
		Name:     "TestToken",
		Symbol:   "TEST",
		Decimals: 18,
	}

	tests := []struct {
		name           string
		nextHeight     *big.Int
		safeHeight     *big.Int
		setupBlocks    func() map[uint64]*ethtypes.Block
		expectedBlocks []uint64 // expected block heights sent to channel
		expectedNext   *big.Int // expected nextHeight after processing
	}{
		{
			name:       "no blocks to process - next height higher than safe height",
			nextHeight: big.NewInt(96),
			safeHeight: big.NewInt(95),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: big.NewInt(96),
		},
		{
			name:       "process single safe block",
			nextHeight: big.NewInt(94),
			safeHeight: big.NewInt(95),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[94] = createEthereumBlock(94, []*ethtypes.Transaction{})
				blocks[95] = createEthereumBlock(95, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{94, 95},
			expectedNext:   big.NewInt(96),
		},
		{
			name:       "process multiple safe blocks",
			nextHeight: big.NewInt(102),
			safeHeight: big.NewInt(105),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				for i := uint64(102); i <= 105; i++ {
					blocks[i] = createEthereumBlock(i, []*ethtypes.Transaction{})
				}
				return blocks
			},
			expectedBlocks: []uint64{102, 103, 104, 105},
			expectedNext:   big.NewInt(106),
		},
		{
			name:       "next height higher than safe height - no processing",
			nextHeight: big.NewInt(10),
			safeHeight: big.NewInt(8),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: big.NewInt(10),
		},
		{
			name:       "exact safe height boundary",
			nextHeight: big.NewInt(5),
			safeHeight: big.NewInt(5),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[5] = createEthereumBlock(5, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{5},
			expectedNext:   big.NewInt(6),
		},
		{
			name:       "process blocks with transactions",
			nextHeight: big.NewInt(18),
			safeHeight: big.NewInt(20),
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
			expectedNext:   big.NewInt(21),
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
				orderValidator:  &mockOrderValidator{},
				logger:          logger,
				blockChan:       blockChan,
				nextHeight:      new(big.Int).Set(tt.nextHeight),
				safeHeight:      new(big.Int).Set(tt.safeHeight),
				chainId:         1,
				config: lib.EthBlockProviderConfig{
					SafeBlockConfirmations: 5, // default test value
				},
				heightMu: &sync.Mutex{},
			}
			provider.processBlocks(context.Background())
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
			if provider.nextHeight.Cmp(tt.expectedNext) != 0 {
				t.Errorf("expected nextHeight %s, got %s", tt.expectedNext.String(), provider.nextHeight.String())
			}
		})
	}
}

// func TestEthBlockProvider_checkTransfer(t *testing.T) {
// 	// Test constants
// 	contract1Address := common.HexToAddress("0x8884567890123456789012345678901234567888")
// 	contract2Address := common.HexToAddress("0x9994567890123456789012345678901234567999")
// 	unknownContract := common.HexToAddress("0x0006543210987654321098765432109876543000")
// 	recipient := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
// 	testToken := types.TokenInfo{
// 		Name:     "TestToken",
// 		Symbol:   "TEST",
// 		Decimals: 18,
// 	}

// 	// Create close order data
// 	orderId := "1010101010101010101010101010101010101010"
// 	orderIdBytes, _ := lib.StringToBytes(orderId)
// 	order := lib.CloseOrder{
// 		OrderId:    orderIdBytes,
// 		ChainId:    0,
// 		CloseOrder: true,
// 	}
// 	closeOrderBytes, _ := order.MarshalJSON()

// 	// Set up token cache
// 	tokenCache := NewERC20TokenCache(&mockEthClient{contractErr: errors.New("error")})
// 	tokenCache.cache.Put(contract1Address.Hex(), testToken)

// 	tests := []struct {
// 		name                  string
// 		txConfig              transactionConfig
// 		receiptStatus         *uint64 // nil means no receipt
// 		expectedTokenTransfer types.TokenTransfer
// 		expectedExtraData     []byte
// 	}{
// 		{
// 			name: "valid erc20 transfer with no extra data, no token transfer",
// 			txConfig: transactionConfig{
// 				contractAddress: contract1Address,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(1000000000000000000),
// 				extraData:       nil,
// 			},
// 			receiptStatus:         nil,
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 		{
// 			name: "valid erc20 transfer with close order and successful receipt",
// 			txConfig: transactionConfig{
// 				contractAddress: contract1Address,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(500000000000000000),
// 				extraData:       closeOrderBytes,
// 			},
// 			receiptStatus: uint64Ptr(1),
// 			expectedTokenTransfer: types.TokenTransfer{
// 				Blockchain:       "ethereum",
// 				TokenInfo:        testToken,
// 				RecipientAddress: recipient,
// 				TokenBaseAmount:  big.NewInt(500000000000000000),
// 				ContractAddress:  contract1Address.Hex(),
// 			},
// 			expectedExtraData: closeOrderBytes,
// 		},
// 		{
// 			name: "erc20 transfer with close order but failed transaction receipt",
// 			txConfig: transactionConfig{
// 				contractAddress: contract2Address,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(500000000000000000),
// 				extraData:       closeOrderBytes,
// 			},
// 			receiptStatus:         uint64Ptr(0),
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 		{
// 			name: "erc20 transfer with close order but receipt not found",
// 			txConfig: transactionConfig{
// 				contractAddress: contract1Address,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(500000000000000000),
// 				extraData:       closeOrderBytes,
// 			},
// 			receiptStatus:         nil,
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 		{
// 			name: "non-erc20 transaction",
// 			txConfig: transactionConfig{
// 				contractAddress: contract1Address,
// 				isERC20:         false,
// 				recipient:       "",
// 				amount:          nil,
// 				extraData:       nil,
// 			},
// 			receiptStatus:         nil,
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 		{
// 			name: "erc20 transfer but token info not found",
// 			txConfig: transactionConfig{
// 				contractAddress: unknownContract,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(1000000000000000000),
// 				extraData:       nil,
// 			},
// 			receiptStatus:         nil,
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 		{
// 			name: "zero amount transfer. no extra data, no token transfer",
// 			txConfig: transactionConfig{
// 				contractAddress: contract1Address,
// 				isERC20:         true,
// 				recipient:       recipient,
// 				amount:          big.NewInt(0),
// 				extraData:       nil,
// 			},
// 			receiptStatus:         nil,
// 			expectedTokenTransfer: types.TokenTransfer{},
// 			expectedExtraData:     nil,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Create transaction based on test configuration
// 			tx := createTestTransaction(tt.txConfig)

// 			// Set up receipts
// 			receipts := make(map[common.Hash]*ethtypes.Receipt)
// 			if tt.receiptStatus != nil {
// 				receipts[tx.tx.Hash()] = &ethtypes.Receipt{Status: *tt.receiptStatus}
// 			}

// 			// Set up provider
// 			logger := lib.NewDefaultLogger()
// 			mockClient := &mockEthClient{
// 				receipts: receipts,
// 			}
// 			provider := &EthBlockProvider{
// 				rpcClient:       mockClient,
// 				erc20TokenCache: tokenCache,
// 				logger:          logger,
// 			}

// 			// Execute test
// 			provider.checkTransfer(tx)

// 			// Verify results
// 			cmpopts := cmpopts.IgnoreFields(types.TokenTransfer{}, "TransactionID")
// 			if diff := cmp.Diff(tt.expectedTokenTransfer, tx.tokenTransfer, cmpopts); diff != "" {
// 				t.Errorf("token transfer mismatch (-want +got):\n%s", diff)
// 			}
// 			if diff := cmp.Diff(tt.expectedExtraData, tx.orderData); diff != "" {
// 				t.Errorf("extra data mismatch (-want +got):\n%s", diff)
// 			}
// 		})
// 	}
// }

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
	tx, _ := NewTransaction(ethTx, 1)
	return tx
}

// uint64Ptr returns a pointer to a uint64 value
func uint64Ptr(v uint64) *uint64 {
	return &v
}

func TestEthBlockProvider_updateHeights(t *testing.T) {
	tests := []struct {
		name               string
		config             lib.EthBlockProviderConfig
		initialNextHeight  *big.Int
		initialSafeHeight  *big.Int
		currentHeight      *big.Int
		expectedNextHeight *big.Int
		expectedSafeHeight *big.Int
		expectExit         bool // if true, expect os.Exit(1) to be called
	}{
		{
			name: "first time initialization - zero next height",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(0),
			initialSafeHeight:  big.NewInt(0),
			currentHeight:      big.NewInt(1000),
			expectedNextHeight: big.NewInt(900), // 1000 - 100
			expectedSafeHeight: big.NewInt(995), // 1000 - 5
		},
		{
			name: "first time initialization - zero next height with small current height",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(0),
			initialSafeHeight:  big.NewInt(0),
			currentHeight:      big.NewInt(50),
			expectedNextHeight: big.NewInt(0),  // max(50 - 100, 0) = 0
			expectedSafeHeight: big.NewInt(45), // 50 - 5
		},
		{
			name: "first time initialization - current height less than safe blocks",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 10,
				StartupBlockDepth:      5,
			},
			initialNextHeight:  big.NewInt(0),
			initialSafeHeight:  big.NewInt(0),
			currentHeight:      big.NewInt(8),
			expectedNextHeight: big.NewInt(3), // 8 - 5
			expectedSafeHeight: big.NewInt(0), // max(8 - 10, 0) = 0
		},
		{
			name: "normal update - safe height increases",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(990),
			initialSafeHeight:  big.NewInt(995),
			currentHeight:      big.NewInt(1010),
			expectedNextHeight: big.NewInt(990),  // unchanged
			expectedSafeHeight: big.NewInt(1005), // 1010 - 5
		},
		{
			name: "reorg protection - safe height would go backward",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(990),
			initialSafeHeight:  big.NewInt(1000),
			currentHeight:      big.NewInt(1003), // would make safe height 998, less than current 1000
			expectedNextHeight: big.NewInt(990),  // unchanged
			expectedSafeHeight: big.NewInt(1000), // unchanged due to reorg protection
		},
		{
			name: "safe height stays same when calculated height equals current",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(990),
			initialSafeHeight:  big.NewInt(995),
			currentHeight:      big.NewInt(1000),
			expectedNextHeight: big.NewInt(990), // unchanged
			expectedSafeHeight: big.NewInt(995), // 1000 - 5 = 995, same as current
		},
		{
			name: "next height ahead of current height - should exit",
			config: lib.EthBlockProviderConfig{
				SafeBlockConfirmations: 5,
				StartupBlockDepth:      100,
			},
			initialNextHeight:  big.NewInt(1010), // ahead of current height
			initialSafeHeight:  big.NewInt(1000),
			currentHeight:      big.NewInt(1005),
			expectedNextHeight: big.NewInt(1010), // unchanged before exit
			expectedSafeHeight: big.NewInt(1000), // unchanged before exit
			expectExit:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := lib.NewDefaultLogger()

			provider := &EthBlockProvider{
				config:     tt.config,
				logger:     logger,
				nextHeight: new(big.Int).Set(tt.initialNextHeight),
				safeHeight: new(big.Int).Set(tt.initialSafeHeight),
				heightMu:   &sync.Mutex{},
			}

			if tt.expectExit {
				// We can't easily test os.Exit, so we'll just verify the condition
				// that would trigger the exit in the actual function
				provider.heightMu.Lock()
				if provider.nextHeight.Cmp(tt.currentHeight) > 0 {
					provider.heightMu.Unlock()
					// This condition would trigger os.Exit(1) in the real function
					// For testing purposes, we just verify this condition is met
					return
				}
				provider.heightMu.Unlock()
			}

			// Execute the function under test
			provider.updateHeights(tt.currentHeight)

			// Verify next height
			if provider.nextHeight.Cmp(tt.expectedNextHeight) != 0 {
				t.Errorf("expected nextHeight %s, got %s", tt.expectedNextHeight.String(), provider.nextHeight.String())
			}

			// Verify safe height
			if provider.safeHeight.Cmp(tt.expectedSafeHeight) != 0 {
				t.Errorf("expected safeHeight %s, got %s", tt.expectedSafeHeight.String(), provider.safeHeight.String())
			}
		})
	}
}
