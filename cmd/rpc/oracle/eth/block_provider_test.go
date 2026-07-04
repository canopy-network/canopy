package eth

import (
	"context"
	"errors"
	"log"
	"math/big"
	"sync"
	"testing"
	"time"

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
	// onBlockFetch, if set, is called after a block is fetched; tests use it to deterministically
	// drive events (e.g. cancel the context) once the loop guard has passed and the send is next
	onBlockFetch func(height uint64)
}

func (m *mockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error) {
	height := number.Uint64()
	if block, exists := m.blocks[height]; exists {
		if m.onBlockFetch != nil {
			m.onBlockFetch(height)
		}
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

func TestNewEthBlockProvider_dialError(t *testing.T) {
	cfg := lib.EthBlockProviderConfig{NodeUrl: "not-a-valid-url://%%%"}
	_, err := NewEthBlockProvider(cfg, &mockOrderValidator{}, lib.NewDefaultLogger(), nil)
	if err == nil {
		t.Fatal("expected error from invalid NodeUrl, got nil")
	}
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
	cache := NewERC20TokenCache(&mockEthClient{}, nil)
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
				config:    lib.EthBlockProviderConfig{},
				heightMu:  &sync.Mutex{},
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
		startHeight    *big.Int
		endHeight      *big.Int
		setupBlocks    func() map[uint64]*ethtypes.Block
		expectedBlocks []uint64 // expected block heights sent to channel
		expectedNext   *big.Int // expected return value from processBlocks
	}{
		{
			name:        "no blocks to process - start height higher than end height",
			startHeight: big.NewInt(96),
			endHeight:   big.NewInt(95),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: big.NewInt(96),
		},
		{
			name:        "process single block",
			startHeight: big.NewInt(94),
			endHeight:   big.NewInt(94),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[94] = createEthereumBlock(94, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{94},
			expectedNext:   big.NewInt(95),
		},
		{
			name:        "process multiple blocks",
			startHeight: big.NewInt(102),
			endHeight:   big.NewInt(105),
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
			name:        "start height higher than end height - no processing",
			startHeight: big.NewInt(10),
			endHeight:   big.NewInt(8),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				return make(map[uint64]*ethtypes.Block)
			},
			expectedNext: big.NewInt(10),
		},
		{
			name:        "exact range boundary",
			startHeight: big.NewInt(5),
			endHeight:   big.NewInt(5),
			setupBlocks: func() map[uint64]*ethtypes.Block {
				blocks := make(map[uint64]*ethtypes.Block)
				blocks[5] = createEthereumBlock(5, []*ethtypes.Transaction{})
				return blocks
			},
			expectedBlocks: []uint64{5},
			expectedNext:   big.NewInt(6),
		},
		{
			name:        "process blocks with transactions",
			startHeight: big.NewInt(18),
			endHeight:   big.NewInt(20),
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
				chainId:         1,
				config:          lib.EthBlockProviderConfig{},
				heightMu:        &sync.Mutex{},
			}
			// Call processBlocks with start and end parameters
			resultNext := provider.processBlocks(context.Background(), tt.startHeight, tt.endHeight)
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

			// Verify return value is correct
			if resultNext.Cmp(tt.expectedNext) != 0 {
				t.Errorf("expected return value %s, got %s", tt.expectedNext.String(), resultNext.String())
			}
		})
	}
}

func TestEthBlockProvider_syncedNoRace(t *testing.T) {
	p := &EthBlockProvider{heightMu: &sync.Mutex{}}
	var wg sync.WaitGroup
	wg.Add(2)
	// writer goroutine (simulates monitorHeaders)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			p.setSynced(i%2 == 0)
		}
	}()
	// reader goroutine (simulates consumer calling IsSynced)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = p.IsSynced()
		}
	}()
	wg.Wait()
}

// fakeSubscription is a minimal ethereum.Subscription implementation that lets tests
// fully control Unsubscribe/Err behavior deterministically.
type fakeSubscription struct {
	errCh        chan error
	unsubscribed bool
	unsubCalled  chan struct{}
}

func newFakeSubscription() *fakeSubscription {
	return &fakeSubscription{
		errCh:       make(chan error, 1),
		unsubCalled: make(chan struct{}, 1),
	}
}

func (f *fakeSubscription) Unsubscribe() {
	f.unsubscribed = true
	select {
	case f.unsubCalled <- struct{}{}:
	default:
	}
}

func (f *fakeSubscription) Err() <-chan error {
	return f.errCh
}

// mockEthWsClient implements EthereumWsClient for tests, letting the test control the
// header channel and subscription (or subscription error) returned to monitorHeaders.
type mockEthWsClient struct {
	sub       ethereum.Subscription
	subErr    error
	headerCh  chan<- *ethtypes.Header
	subscribe func(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error)
}

func (m *mockEthWsClient) SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
	if m.subscribe != nil {
		return m.subscribe(ctx, ch)
	}
	m.headerCh = ch
	return m.sub, m.subErr
}

func (m *mockEthWsClient) Close() {}

// TestEthBlockProvider_monitorHeaders_ReorgDetected exercises the reorg-detection branch:
// when a header arrives whose number is lower than nextHeight (i.e. nextHeight is ahead of
// the reported chain head), monitorHeaders must unsubscribe and return ErrSourceHeight.
func TestEthBlockProvider_monitorHeaders_ReorgDetected(t *testing.T) {
	sub := newFakeSubscription()
	headerCh := make(chan *ethtypes.Header, 1)
	wsClient := &mockEthWsClient{
		sub: sub,
		subscribe: func(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
			// capture the channel passed by monitorHeaders so the test can push a header into it
			go func() {
				h := <-headerCh
				ch <- h
			}()
			return sub, nil
		},
	}
	provider := &EthBlockProvider{
		wsClient:   wsClient,
		logger:     lib.NewDefaultLogger(),
		nextHeight: big.NewInt(100), // ahead of the header we're about to deliver
		heightMu:   &sync.Mutex{},
	}
	// header number (99) is lower than nextHeight (100) -> reorg condition
	headerCh <- &ethtypes.Header{Number: big.NewInt(99)}

	errCh := make(chan error, 1)
	go func() {
		errCh <- provider.monitorHeaders(context.Background())
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrSourceHeight) {
			t.Errorf("expected ErrSourceHeight, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHeaders did not return on reorg detection")
	}
	select {
	case <-sub.unsubCalled:
	case <-time.After(time.Second):
		t.Error("expected Unsubscribe to be called on reorg detection")
	}
}

// TestEthBlockProvider_monitorHeaders_WsClientNil verifies monitorHeaders fails fast
// with a descriptive error when the websocket client was never established.
func TestEthBlockProvider_monitorHeaders_WsClientNil(t *testing.T) {
	provider := &EthBlockProvider{
		logger:   lib.NewDefaultLogger(),
		heightMu: &sync.Mutex{},
	}
	err := provider.monitorHeaders(context.Background())
	if err == nil {
		t.Fatal("expected error when wsClient is nil, got nil")
	}
}

// TestEthBlockProvider_monitorHeaders_SubscribeError verifies the subscription-error path
// returns the underlying error without panicking.
func TestEthBlockProvider_monitorHeaders_SubscribeError(t *testing.T) {
	wantErr := errors.New("subscribe boom")
	wsClient := &mockEthWsClient{subErr: wantErr}
	provider := &EthBlockProvider{
		wsClient:   wsClient,
		logger:     lib.NewDefaultLogger(),
		nextHeight: big.NewInt(1),
		heightMu:   &sync.Mutex{},
	}
	err := provider.monitorHeaders(context.Background())
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// TestEthBlockProvider_monitorHeaders_ContextCancelled verifies a context cancellation while
// waiting on headers causes Unsubscribe to be called and ctx.Err() to be returned.
func TestEthBlockProvider_monitorHeaders_ContextCancelled(t *testing.T) {
	sub := newFakeSubscription()
	wsClient := &mockEthWsClient{sub: sub}
	provider := &EthBlockProvider{
		wsClient:   wsClient,
		logger:     lib.NewDefaultLogger(),
		nextHeight: big.NewInt(1),
		heightMu:   &sync.Mutex{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- provider.monitorHeaders(ctx)
	}()
	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHeaders did not return on context cancellation")
	}
	select {
	case <-sub.unsubCalled:
	case <-time.After(time.Second):
		t.Error("expected Unsubscribe to be called on context cancellation")
	}
}

// TestEthBlockProvider_monitorHeaders_SubscriptionErrChan verifies that an error surfaced on
// sub.Err() causes monitorHeaders to unsubscribe and propagate the error.
func TestEthBlockProvider_monitorHeaders_SubscriptionErrChan(t *testing.T) {
	sub := newFakeSubscription()
	wsClient := &mockEthWsClient{sub: sub}
	provider := &EthBlockProvider{
		wsClient:   wsClient,
		logger:     lib.NewDefaultLogger(),
		nextHeight: big.NewInt(1),
		heightMu:   &sync.Mutex{},
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- provider.monitorHeaders(context.Background())
	}()
	wantErr := errors.New("subscription dropped")
	sub.errCh <- wantErr
	select {
	case err := <-errCh:
		if !errors.Is(err, wantErr) {
			t.Errorf("expected %v, got %v", wantErr, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("monitorHeaders did not return on subscription error")
	}
	select {
	case <-sub.unsubCalled:
	case <-time.After(time.Second):
		t.Error("expected Unsubscribe to be called on subscription error")
	}
}

// TestEthBlockProvider_transactionSuccess covers the receipt-fetch error path along with
// both receipt status outcomes (success and failure).
func TestEthBlockProvider_transactionSuccess(t *testing.T) {
	toAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")

	tests := []struct {
		name          string
		setupReceipts func(hash common.Hash) map[common.Hash]*ethtypes.Receipt
		expectSuccess bool
		expectErr     error
	}{
		{
			name: "receipt fetch error",
			setupReceipts: func(hash common.Hash) map[common.Hash]*ethtypes.Receipt {
				// no receipt registered -> mockEthClient.TransactionReceipt returns ethereum.NotFound
				return map[common.Hash]*ethtypes.Receipt{}
			},
			expectSuccess: false,
			expectErr:     ErrTransactionReceipt,
		},
		{
			name: "receipt status success",
			setupReceipts: func(hash common.Hash) map[common.Hash]*ethtypes.Receipt {
				return map[common.Hash]*ethtypes.Receipt{
					hash: {Status: TransactionStatusSuccess},
				}
			},
			expectSuccess: true,
			expectErr:     nil,
		},
		{
			name: "receipt status failure",
			setupReceipts: func(hash common.Hash) map[common.Hash]*ethtypes.Receipt {
				return map[common.Hash]*ethtypes.Receipt{
					hash: {Status: 0},
				}
			},
			expectSuccess: false,
			expectErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ethTx := createTransaction(toAddress, []byte("data"))
			tx, err := NewTransaction(ethTx, 1)
			if err != nil {
				t.Fatalf("NewTransaction: %v", err)
			}
			mockClient := &mockEthClient{
				receipts: tt.setupReceipts(ethTx.Hash()),
			}
			provider := &EthBlockProvider{
				rpcClient: mockClient,
				logger:    lib.NewDefaultLogger(),
				heightMu:  &sync.Mutex{},
			}
			success, err := provider.transactionSuccess(context.Background(), tx)
			if success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, success)
			}
			if tt.expectErr == nil {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else if !errors.Is(err, tt.expectErr) {
				t.Errorf("expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}

// retryingReceiptClient wraps mockEthClient, failing the first N TransactionReceipt calls
// with ethereum.NotFound (triggering ErrTransactionReceipt) before succeeding.
type retryingReceiptClient struct {
	*mockEthClient
	hash       common.Hash
	failFirstN int
	calls      int
	onCall     func()
}

func (r *retryingReceiptClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error) {
	r.calls++
	if r.onCall != nil {
		r.onCall()
	}
	if r.calls <= r.failFirstN {
		return nil, ethereum.NotFound
	}
	return &ethtypes.Receipt{Status: TransactionStatusSuccess}, nil
}

// TestEthBlockProvider_processBlockTransactions_RetryableThenSuccess verifies that a
// receipt-fetch error (retryable per ErrTransactionReceipt) causes a retry with backoff,
// and a subsequent successful receipt fetch completes processing without error.
func TestEthBlockProvider_processBlockTransactions_RetryableThenSuccess(t *testing.T) {
	toAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	transferData := createERC20TransferData("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", big.NewInt(1000000000000000000), []byte(`{"order_id":"abc"}`))
	ethTx := createTransaction(toAddress, transferData)
	tx, err := NewTransaction(ethTx, 1)
	if err != nil {
		t.Fatalf("NewTransaction: %v", err)
	}
	block := &Block{transactions: []*Transaction{tx}}

	var receiptCalls int
	rpc := &retryingReceiptClient{
		mockEthClient: &mockEthClient{},
		hash:          ethTx.Hash(),
		failFirstN:    1,
		onCall:        func() { receiptCalls++ },
	}
	provider := &EthBlockProvider{
		rpcClient:      rpc,
		orderValidator: &mockOrderValidator{},
		erc20TokenCache: setupTokenCache(toAddress, types.TokenInfo{
			Name: "Test", Symbol: "TST", Decimals: 18,
		}),
		logger:   lib.NewDefaultLogger(),
		heightMu: &sync.Mutex{},
	}
	start := time.Now()
	err = provider.processBlockTransactions(context.Background(), block)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receiptCalls < 2 {
		t.Errorf("expected at least 2 receipt calls (1 failure + 1 retry), got %d", receiptCalls)
	}
	// first backoff is 1<<0 == 1 second
	if elapsed < time.Second {
		t.Errorf("expected backoff delay of at least 1s before retry, elapsed %v", elapsed)
	}
}

// TestEthBlockProvider_processBlockTransactions_ExhaustedRetries verifies that when every
// attempt fails with a retryable error, processing exhausts all attempts (metrics/log the
// exhaustion) but still returns nil overall - exhaustion of a single transaction does not
// fail the whole block's processing.
//
// Note: processTransaction can only return ErrTransactionReceipt (receipt fetch failure) or
// ErrTokenInfo (token info fetch failure) as errors - both are retryable per the `!errors.Is`
// check in processBlockTransactions. There is no reachable non-retryable error out of
// processTransaction without modifying production code, so the "non-retryable breaks
// immediately" case is not independently testable; this test instead exercises full
// exhaustion of maxTransactionProcessAttempts.
func TestEthBlockProvider_processBlockTransactions_ExhaustedRetries(t *testing.T) {
	toAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	transferData := createERC20TransferData("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", big.NewInt(1000000000000000000), []byte(`{"order_id":"abc"}`))
	ethTx := createTransaction(toAddress, transferData)
	tx, err := NewTransaction(ethTx, 1)
	if err != nil {
		t.Fatalf("NewTransaction: %v", err)
	}
	block := &Block{transactions: []*Transaction{tx}}

	rpc := &retryingReceiptClient{
		mockEthClient: &mockEthClient{},
		hash:          ethTx.Hash(),
		failFirstN:    maxTransactionProcessAttempts, // fail every attempt
	}
	provider := &EthBlockProvider{
		rpcClient:      rpc,
		orderValidator: &mockOrderValidator{},
		logger:         lib.NewDefaultLogger(),
		heightMu:       &sync.Mutex{},
	}
	err = provider.processBlockTransactions(context.Background(), block)
	// processBlockTransactions always returns nil at the top level; exhaustion is logged/metriced
	// per-transaction rather than surfaced as a return error.
	if err != nil {
		t.Errorf("expected nil error (exhaustion handled internally), got %v", err)
	}
	if rpc.calls != maxTransactionProcessAttempts {
		t.Errorf("expected %d receipt fetch attempts, got %d", maxTransactionProcessAttempts, rpc.calls)
	}
}

// TestEthBlockProvider_processBlockTransactions_CtxCancelledDuringBackoff verifies that if
// the context is cancelled while waiting in the exponential backoff, processBlockTransactions
// returns ctx.Err() immediately rather than continuing to retry.
func TestEthBlockProvider_processBlockTransactions_CtxCancelledDuringBackoff(t *testing.T) {
	toAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	transferData := createERC20TransferData("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", big.NewInt(1000000000000000000), []byte(`{"order_id":"abc"}`))
	ethTx := createTransaction(toAddress, transferData)
	tx, err := NewTransaction(ethTx, 1)
	if err != nil {
		t.Fatalf("NewTransaction: %v", err)
	}
	block := &Block{transactions: []*Transaction{tx}}

	rpc := &retryingReceiptClient{
		mockEthClient: &mockEthClient{},
		hash:          ethTx.Hash(),
		failFirstN:    maxTransactionProcessAttempts, // always fail, so it will hit backoff before attempt 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	provider := &EthBlockProvider{
		rpcClient:      rpc,
		orderValidator: &mockOrderValidator{},
		logger:         lib.NewDefaultLogger(),
		heightMu:       &sync.Mutex{},
	}
	// cancel shortly after starting so cancellation lands during the 1s backoff sleep
	// following the first failed attempt.
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	done := make(chan error, 1)
	go func() {
		done <- provider.processBlockTransactions(ctx, block)
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("processBlockTransactions did not return promptly on context cancellation")
	}
}

// TestEthBlockProvider_fetchBlock_SkipsMalformedTransaction verifies that a transaction which
// fails NewTransaction wrapping (e.g. a contract-creation tx with no recipient address) is
// logged and skipped rather than failing the entire block fetch.
func TestEthBlockProvider_fetchBlock_SkipsMalformedTransaction(t *testing.T) {
	recipientAddress := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	goodTx := createTransaction(recipientAddress, []byte("regular data"))
	// a contract-creation transaction has a nil `to`, which fails common.IsHexAddress("")
	// inside NewTransaction, returning *InvalidAddressError - this is the "malformed" tx path.
	badTx := ethtypes.NewContractCreation(1, big.NewInt(0), 21000, big.NewInt(1000000000), []byte("create"))
	privateKey, kerr := crypto.GenerateKey()
	if kerr != nil {
		t.Fatalf("failed to generate key: %v", kerr)
	}
	signer := ethtypes.NewEIP155Signer(big.NewInt(0))
	signedBadTx, serr := ethtypes.SignTx(badTx, signer, privateKey)
	if serr != nil {
		t.Fatalf("failed to sign contract-creation tx: %v", serr)
	}

	mockClient := &mockEthClient{
		blocks: map[uint64]*ethtypes.Block{
			7: createEthereumBlock(7, []*ethtypes.Transaction{goodTx, signedBadTx}),
		},
	}
	provider := &EthBlockProvider{
		rpcClient: mockClient,
		logger:    lib.NewDefaultLogger(),
		chainId:   1,
		config:    lib.EthBlockProviderConfig{},
		heightMu:  &sync.Mutex{},
	}

	block, err := provider.fetchBlock(context.Background(), big.NewInt(7))
	if err != nil {
		t.Fatalf("expected fetchBlock to succeed despite malformed tx, got error: %v", err)
	}
	if block == nil {
		t.Fatal("expected block, got nil")
	}
	// only the well-formed transaction should have been kept
	txs := block.Transactions()
	if len(txs) != 1 {
		t.Fatalf("expected 1 surviving transaction, got %d", len(txs))
	}
	if txs[0].Hash() != goodTx.Hash().Hex() {
		t.Errorf("expected surviving tx hash %s, got %s", goodTx.Hash().Hex(), txs[0].Hash())
	}
}

// TestNewBlock_NilInput verifies NewBlock rejects a nil ethereum block with lib.ErrNilBlock().
func TestNewBlock_NilInput(t *testing.T) {
	block, err := NewBlock(nil)
	if block != nil {
		t.Errorf("expected nil block, got %+v", block)
	}
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	wantErr := lib.ErrNilBlock()
	if err.Error() != wantErr.Error() {
		t.Errorf("expected error %q, got %q", wantErr.Error(), err.Error())
	}
}

func TestEthBlockProvider_processBlocksCancelledSend(t *testing.T) {
	mockClient := &mockEthClient{
		blocks: map[uint64]*ethtypes.Block{
			50: createEthereumBlock(50, []*ethtypes.Transaction{}),
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	// cancel the moment block 50 is fetched: the top-of-loop guard for this iteration has already
	// passed, so cancellation deterministically lands on the channel send (unbuffered, no receiver)
	// rather than racing a wall-clock timer.
	mockClient.onBlockFetch = func(uint64) { cancel() }
	provider := &EthBlockProvider{
		rpcClient:      mockClient,
		orderValidator: &mockOrderValidator{},
		logger:         lib.NewDefaultLogger(),
		blockChan:      make(chan types.BlockI), // unbuffered, no receiver
		chainId:        1,
		config:         lib.EthBlockProviderConfig{},
		heightMu:       &sync.Mutex{},
	}

	done := make(chan *big.Int, 1)
	go func() {
		done <- provider.processBlocks(ctx, big.NewInt(50), big.NewInt(50))
	}()

	select {
	case next := <-done:
		// block 50 was never delivered, so next height stays at 50 for retry
		if next.Cmp(big.NewInt(50)) != 0 {
			t.Errorf("expected next height 50 (unsent block retried), got %s", next.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("processBlocks blocked on channel send despite cancelled context")
	}
}
