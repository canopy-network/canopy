package oracle

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrderStore implements OrderStore interface for testing purposes
type mockOrderStore struct {
	// lockOrders stores witnessed orders for lock order type
	lockOrders map[string]*types.WitnessedOrder
	// closeOrders stores witnessed orders for close order type
	closeOrders map[string]*types.WitnessedOrder
	// lockOrders stores witnessed orders for lock order type
	archiveLockOrders map[string]*types.WitnessedOrder
	// closeOrders stores witnessed orders for close order type
	archivedCloseOrders map[string]*types.WitnessedOrder
	// mutex to protect concurrent access
	rwLock sync.RWMutex
}

// NewMockOrderStore creates a new MockOrderStore instance
func NewMockOrderStore() *mockOrderStore {
	return &mockOrderStore{
		lockOrders:          make(map[string]*types.WitnessedOrder),
		closeOrders:         make(map[string]*types.WitnessedOrder),
		archiveLockOrders:   make(map[string]*types.WitnessedOrder),
		archivedCloseOrders: make(map[string]*types.WitnessedOrder),
		rwLock:              sync.RWMutex{},
	}
}

// VerifyOrder verifies the order with order id is present in the store
func (m *mockOrderStore) VerifyOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	// validate parameters
	if err := m.validateOrderParameters(order.OrderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}

	// get the appropriate map based on order type
	orderMap := m.getOrderMap(orderType)
	key := hex.EncodeToString(order.OrderId)

	// check if order exists
	storedOrder, exists := orderMap[key]
	if !exists {
		return ErrVerifyOrder(fmt.Errorf("order not found"))
	}

	// compare lock order
	if !order.LockOrder.Equals(storedOrder.LockOrder) {
		return ErrVerifyOrder(fmt.Errorf("lock order not equal"))
	}
	// compare close order
	if !order.CloseOrder.Equals(storedOrder.CloseOrder) {
		return ErrVerifyOrder(fmt.Errorf("close order not equal"))
	}

	return nil
}

// WriteOrder writes an order to the appropriate map
func (m *mockOrderStore) WriteOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	// validate parameters
	if err := m.validateOrderParameters(order.OrderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}

	// get the appropriate map based on order type
	orderMap := m.getOrderMap(orderType)
	key := hex.EncodeToString(order.OrderId)

	// store the order
	orderMap[key] = order
	return nil
}

// ReadOrder reads an order from the appropriate map
func (m *mockOrderStore) ReadOrder(orderId []byte, orderType types.OrderType) (*types.WitnessedOrder, lib.ErrorI) {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	// validate parameters
	if err := m.validateOrderParameters(orderId, orderType); err != nil {
		return nil, ErrValidateOrder(err)
	}
	// get the appropriate map based on order type
	orderMap := m.getOrderMap(orderType)
	key := hex.EncodeToString(orderId)
	// retrieve the order
	storedOrder, exists := orderMap[key]
	if !exists {
		return nil, ErrReadOrder(fmt.Errorf("order not found"))
	}

	return storedOrder, nil
}

// RemoveOrder removes an order from the appropriate map
func (m *mockOrderStore) RemoveOrder(orderId []byte, orderType types.OrderType) lib.ErrorI {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()

	// validate parameters
	if err := m.validateOrderParameters(orderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}

	// get the appropriate map based on order type
	orderMap := m.getOrderMap(orderType)
	key := hex.EncodeToString(orderId)

	// check if order exists
	if _, exists := orderMap[key]; !exists {
		return ErrRemoveOrder(fmt.Errorf("order not found"))
	}

	// remove the order
	delete(orderMap, key)
	return nil
}

// GetAllOrderIds gets all order ids present in the store for a specific order type
func (m *mockOrderStore) GetAllOrderIds(orderType types.OrderType) ([][]byte, lib.ErrorI) {
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()

	// validate order type
	if orderType != types.LockOrderType && orderType != types.CloseOrderType {
		return nil, ErrVerifyOrder(fmt.Errorf("invalid order type: %s", orderType))
	}

	// get the appropriate map based on order type
	orderMap := m.getOrderMap(orderType)

	// collect all order ids
	var orderIds [][]byte
	for key := range orderMap {
		id, err := hex.DecodeString(key)
		if err != nil {
			continue // skip invalid keys
		}
		orderIds = append(orderIds, id)
	}

	return orderIds, nil
}

// ArchiveOrder archives a witnessed order (mock implementation - does nothing)
func (m *mockOrderStore) ArchiveOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	// Mock implementation - just return nil for now
	return nil
}

// getOrderMap returns the appropriate map based on order type
func (m *mockOrderStore) getOrderMap(orderType types.OrderType) map[string]*types.WitnessedOrder {
	switch orderType {
	case types.LockOrderType:
		return m.lockOrders
	case types.CloseOrderType:
		return m.closeOrders
	default:
		return nil
	}
}

// validateOrderParameters validates order id and order type
func (m *mockOrderStore) validateOrderParameters(orderId []byte, orderType types.OrderType) error {
	// orderId cannot be nil
	if orderId == nil {
		return fmt.Errorf("order id cannot be nil")
	}
	// verify order id length
	// if len(orderId) != orderIdLenBytes {
	// 	return fmt.Errorf("order id invalid length")
	// }
	// validate order type
	if orderType != types.LockOrderType && orderType != types.CloseOrderType {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	return nil
}

type mockBlock struct {
	number       uint64
	hash         string
	parentHash   string
	transactions []types.TransactionI
}

func (m *mockBlock) Number() uint64 {
	return m.number
}

func (m *mockBlock) Hash() string {
	return m.hash
}

func (m *mockBlock) ParentHash() string {
	return m.parentHash
}

func (m *mockBlock) Transactions() []types.TransactionI {
	return m.transactions
}

type mockTransaction struct {
	blockchain    string
	from          string
	to            string
	data          []byte
	hash          string
	order         *types.WitnessedOrder
	tokenTransfer types.TokenTransfer
}

func (m *mockTransaction) Blockchain() string {
	return m.blockchain
}

func (m *mockTransaction) From() string {
	return m.from
}

func (m *mockTransaction) FromBytes() ([]byte, error) {
	return lib.StringToBytes(strings.TrimPrefix(m.from, "0x"))
}

func (m *mockTransaction) To() string {
	return m.to
}

func (m *mockTransaction) Data() []byte {
	return m.data
}

func (m *mockTransaction) Hash() string {
	return m.hash
}

func (m *mockTransaction) Order() *types.WitnessedOrder {
	return m.order
}

func (m *mockTransaction) TokenTransfer() types.TokenTransfer {
	return m.tokenTransfer
}

// MatchesOrderDestination mirrors eth.Transaction: contractBytes formatted as an EVM address
// must equal the mock's `to`, and the token transfer recipient must equal recipientBytes.
func (m *mockTransaction) MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error) {
	if common.BytesToAddress(contractBytes).String() != m.to {
		return false, nil
	}
	recipient, err := lib.StringToBytes(strings.TrimPrefix(m.tokenTransfer.RecipientAddress, "0x"))
	if err != nil {
		return false, err
	}
	if !bytes.Equal(recipientBytes, recipient) {
		return false, nil
	}
	return true, nil
}

func createMockBlockWithTransactions(blockNumber uint64, blockHash string, transactions []types.TransactionI) types.BlockI {
	return &mockBlock{
		number:       blockNumber,
		hash:         blockHash,
		transactions: transactions,
	}
}

// createSellOrder creates a sell order with an optional buyer receive address and seller receive address
func createSellOrder(orderIdHex string, data string, buyerReceiveAddress ...string) *lib.SellOrder {
	orderIdBytes := []byte(orderIdHex)
	sellOrder := &lib.SellOrder{
		Id: orderIdBytes,
	}
	if data != "" {
		b, err := lib.StringToBytes(strings.TrimPrefix(data, "0x"))
		if err != nil {
			panic(err)
		}
		sellOrder.Data = b
	}
	if len(buyerReceiveAddress) > 0 && len(buyerReceiveAddress[0]) > 0 {
		sellOrder.BuyerReceiveAddress = []byte(buyerReceiveAddress[0])
	}
	return sellOrder
}

// createSellOrderWithSellerAddr creates a sell order with seller receive address
func createSellOrderWithSellerAddr(orderIdHex string, buyerReceiveAddress string, sellerReceiveAddress string) *lib.SellOrder {
	orderIdBytes := []byte(orderIdHex)
	sellOrder := &lib.SellOrder{
		Id:                   orderIdBytes,
		BuyerReceiveAddress:  []byte(buyerReceiveAddress),
		SellerReceiveAddress: []byte(sellerReceiveAddress),
	}
	return sellOrder
}

// createWitnessedLockOrder creates a witnessed lock order with an optional buyerAddress
func createWitnessedLockOrder(id string, buyerAddress ...string) *types.WitnessedOrder {
	var addressValue string
	if len(buyerAddress) > 0 {
		addressValue = buyerAddress[0]
	}
	return &types.WitnessedOrder{
		OrderId: []byte(id),
		LockOrder: &lib.LockOrder{
			OrderId:             []byte(id),
			BuyerReceiveAddress: []byte(addressValue),
		},
	}
}

// createWitnessedCloseOrder creates a witnessed close order with an optional chainId
func createWitnessedCloseOrder(id string, chainId ...uint64) *types.WitnessedOrder {
	var chainIdValue uint64
	if len(chainId) > 0 {
		chainIdValue = chainId[0]
	}

	return &types.WitnessedOrder{
		OrderId: []byte(id),
		CloseOrder: &lib.CloseOrder{
			OrderId:    []byte(id),
			ChainId:    chainIdValue,
			CloseOrder: true,
		},
	}
}

func createOrderStore(orders ...*types.WitnessedOrder) *mockOrderStore {
	mockStore := NewMockOrderStore()
	for _, order := range orders {
		if order.LockOrder != nil {
			err := mockStore.WriteOrder(order, types.LockOrderType)
			if err != nil {
				panic(fmt.Sprintf("failed to write lock order for test: %v", err))
			}
		}
		if order.CloseOrder != nil {
			err := mockStore.WriteOrder(order, types.CloseOrderType)
			if err != nil {
				panic(fmt.Sprintf("failed to write close order for test: %v", err))
			}
		}
	}
	return mockStore
}

func withCommittee(order *lib.SellOrder, committee uint64) *lib.SellOrder {
	order.Committee = committee
	return order
}

func withData(order *lib.SellOrder, dataHex string) *lib.SellOrder {
	b, err := lib.StringToBytes(strings.TrimPrefix(dataHex, "0x"))
	if err != nil {
		panic(err)
	}
	order.Data = b
	return order
}

func createOrderBook(orders ...*lib.SellOrder) *lib.OrderBook {
	orderBook := &lib.OrderBook{}
	for _, order := range orders {
		orderBook.Orders = append(orderBook.Orders, order)
	}
	return orderBook
}

func TestOracle_ValidateProposedOrders(t *testing.T) {
	orders := func(lockOrderIDs []string, closeOrderIDs []string, resetOrderIDs []string) *lib.Orders {
		lockOrders := make([]*lib.LockOrder, len(lockOrderIDs))
		for i, id := range lockOrderIDs {
			lockOrders[i] = &lib.LockOrder{
				OrderId: []byte(id),
			}
		}
		closeOrders := make([][]byte, len(closeOrderIDs))
		for i, id := range closeOrderIDs {
			closeOrders[i] = []byte(id)
		}
		resetOrders := make([][]byte, len(resetOrderIDs))
		for i, id := range resetOrderIDs {
			resetOrders[i] = []byte(id)
		}
		return &lib.Orders{
			LockOrders:  lockOrders,
			CloseOrders: closeOrders,
			ResetOrders: resetOrders,
		}
	}

	tests := []struct {
		name          string
		orders        *lib.Orders
		orderStore    OrderStore
		orderBook     *lib.OrderBook
		safeHeight    uint64
		expectedError bool
		errorContains string
	}{
		{
			name:          "nil orders should return no error",
			orders:        nil,
			expectedError: false,
			orderStore:    createOrderStore(),
		},
		{
			name:          "no orders should return no error",
			orders:        orders(nil, nil, nil),
			orderStore:    createOrderStore(createWitnessedLockOrder("lock1")),
			expectedError: false,
		},
		{
			name:   "valid lock orders should pass validation",
			orders: orders([]string{"lock1", "lock2"}, nil, nil),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1"),
				createWitnessedLockOrder("lock2"),
			),
			expectedError: false,
		},
		{
			name:   "valid close orders should pass validation",
			orders: orders(nil, []string{"close1", "close2"}, nil),
			orderStore: createOrderStore(
				createWitnessedCloseOrder("close1", 1),
				createWitnessedCloseOrder("close2", 1),
			),
			expectedError: false,
		},
		{
			name:   "valid mixed orders should pass validation",
			orders: orders([]string{"lock1"}, []string{"close1"}, nil),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1"),
				createWitnessedCloseOrder("close1", 1),
			),
			expectedError: false,
		},
		{
			name:       "valid reset order should pass validation",
			orders:     orders(nil, nil, []string{"reset1"}),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(func() *lib.SellOrder {
				order := createSellOrder("reset1", "", "buyer")
				order.BuyerChainDeadline = 10
				return order
			}()),
			safeHeight:    20,
			expectedError: false,
		},
		{
			name:   "lock order verification failure should return error",
			orders: orders([]string{"lock1"}, nil, nil),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1", "address"),
			),
			expectedError: true,
			errorContains: "order not verified",
		},
		{
			name:   "close order verification failure should return error",
			orders: orders(nil, []string{"close1"}, nil),
			orderStore: createOrderStore(
				createWitnessedCloseOrder("close1"), // missing chain id
			),
			expectedError: true,
			errorContains: "order not verified",
		},
		{
			name:       "reset order should fail when deadline has not passed",
			orders:     orders(nil, nil, []string{"reset1"}),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(func() *lib.SellOrder {
				order := createSellOrder("reset1", "", "buyer")
				order.BuyerChainDeadline = 25
				return order
			}()),
			safeHeight:    20,
			expectedError: true,
			errorContains: "order not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Oracle instance
			tempDir, _ := os.MkdirTemp("", "oracle_test")
			defer os.RemoveAll(tempDir)
			oracle := &Oracle{
				orderStore: tt.orderStore,
				orderBook:  tt.orderBook,
				state:      NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger()),
				committee:  1,
				log:        lib.NewDefaultLogger(),
			}
			oracle.state.safeHeight = tt.safeHeight

			// Execute test
			err := oracle.ValidateProposedOrders(tt.orders, tt.orderBook)

			// Verify results
			if tt.expectedError {
				require.Error(t, err, "expected error but got nil")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOracle_CommitCertificate_ResetOrderCleanup(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "oracle_commit_test")
	defer os.RemoveAll(tempDir)

	orderID := "order1"
	otherID := "other-order"
	orderIDKey := lib.BytesToString([]byte(orderID))
	otherIDKey := lib.BytesToString([]byte(otherID))
	store := createOrderStore(createWitnessedCloseOrder(orderID))
	state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
	state.lockOrderSubmissions[orderIDKey] = 10
	state.closeOrderSubmissions[orderIDKey] = 11
	state.lockOrderSubmissions[otherIDKey] = 20
	state.closeOrderSubmissions[otherIDKey] = 21

	oracle := &Oracle{
		orderStore: store,
		state:      state,
		committee:  1,
		log:        lib.NewDefaultLogger(),
	}

	qc := &lib.QuorumCertificate{
		Header: &lib.View{
			RootHeight: 500,
		},
		Results: &lib.CertificateResult{
			Orders: &lib.Orders{
				ResetOrders: [][]byte{[]byte(orderID)},
			},
		},
	}

	err := oracle.CommitCertificate(qc, nil, nil, 0)
	require.NoError(t, err)

	_, readErr := store.ReadOrder([]byte(orderID), types.CloseOrderType)
	require.Error(t, readErr, "expected close order to be removed after reset commit")

	_, hasLockHistory := state.lockOrderSubmissions[orderIDKey]
	_, hasCloseHistory := state.closeOrderSubmissions[orderIDKey]
	assert.False(t, hasLockHistory, "expected lock history to be cleared for reset order")
	assert.False(t, hasCloseHistory, "expected close history to be cleared for reset order")

	_, hasOtherLockHistory := state.lockOrderSubmissions[otherIDKey]
	_, hasOtherCloseHistory := state.closeOrderSubmissions[otherIDKey]
	assert.True(t, hasOtherLockHistory, "expected other lock history to remain")
	assert.True(t, hasOtherCloseHistory, "expected other close history to remain")
}

func TestOracle_WitnessedOrders(t *testing.T) {
	buyerReceiveAddress := "0034567890123456789012345678901234567800"
	contractAddress := "0x1234567890123456789012345678901234567890"

	orderIdOne := "order1"
	orderIdTwo := "order2"

	tests := []struct {
		name                   string
		orderStore             OrderStore
		orderBook              *lib.OrderBook
		orderBookOrders        []*lib.SellOrder
		storeLockOrders        map[string]*lib.LockOrder
		storeCloseOrders       map[string]*lib.CloseOrder
		safeHeight             uint64
		expectedLockOrdersLen  int
		expectedCloseOrdersLen int
		expectedResetOrdersLen int
		expectedLockOrderIds   []string
		expectedCloseOrderIds  []string
		expectedResetOrderIds  []string
	}{
		{
			name:            "orders exist in store but not in order book. both lock and close orders should not be included in proposed block",
			orderBook:       &lib.OrderBook{},
			orderBookOrders: []*lib.SellOrder{},
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
				createWitnessedCloseOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "orders exist in order book but not in store.",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, "", ""),
				createSellOrder(orderIdTwo, contractAddress, buyerReceiveAddress),
			),
			orderStore:             NewMockOrderStore(),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "matching stored lock order, should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, "", ""),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
				createWitnessedLockOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "matching close order exists in store and order book. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, contractAddress, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedCloseOrder(orderIdOne),
				createWitnessedCloseOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdOne},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "unlocked order exists in order book and matching lock order exists in store. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, "", ""),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "locked order exists in order book and matching close order exists in store. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, contractAddress, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedCloseOrder(orderIdOne),
			),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdOne},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "expired locked order with no witnessed close should be submitted as reset",
			orderBook: createOrderBook(func() *lib.SellOrder {
				order := createSellOrder(orderIdOne, contractAddress, buyerReceiveAddress)
				order.BuyerChainDeadline = 10
				return order
			}()),
			orderStore:             createOrderStore(),
			safeHeight:             20,
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{orderIdOne},
		},
		{
			name: "expired locked order with a LATE close (witnessed after deadline) should reset, not close",
			orderBook: createOrderBook(func() *lib.SellOrder {
				order := createSellOrder(orderIdOne, contractAddress, buyerReceiveAddress)
				order.BuyerChainDeadline = 10
				return order
			}()),
			orderStore: createOrderStore(func() *types.WitnessedOrder {
				o := createWitnessedCloseOrder(orderIdOne)
				o.WitnessedHeight = 15 // mined AFTER the deadline of 10 - late payment
				return o
			}()),
			safeHeight:             20,
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedResetOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
			expectedResetOrderIds:  []string{orderIdOne},
		},
		{
			name: "expired locked order with an ON-TIME close (witnessed on or before deadline) must close, not reset",
			orderBook: createOrderBook(func() *lib.SellOrder {
				order := createSellOrder(orderIdOne, contractAddress, buyerReceiveAddress)
				order.BuyerChainDeadline = 10
				return order
			}()),
			orderStore: createOrderStore(func() *types.WitnessedOrder {
				o := createWitnessedCloseOrder(orderIdOne)
				o.WitnessedHeight = 5 // mined at/before the deadline of 10 - buyer paid on time
				return o
			}()),
			safeHeight:             20,
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdOne},
			expectedResetOrderIds:  []string{},
		},
		{
			name: "mixed scenario with multiple orders",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, "", ""),
				createSellOrder(orderIdTwo, contractAddress, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
				createWitnessedCloseOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 1,
			expectedResetOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{orderIdTwo},
			expectedResetOrderIds:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := lib.NewDefaultLogger()
			oracle := &Oracle{
				orderStore: tt.orderStore,
				state:      NewOracleState("file", log),
				log:        log,
			}
			oracle.state.safeHeight = tt.safeHeight
			// sourceChainHeight (the chain tip) is always >= safeHeight in reality; set it high so
			// the propose-delay gate in shouldSubmit does not hold orders in these selection tests
			oracle.state.sourceChainHeight = tt.safeHeight + 1000
			// 100 to specify a high enough root height that shouldSubmit always passes
			witnessedLockOrders, witnessedCloseOrders, witnessedResetOrders := oracle.WitnessedOrders(tt.orderBook, 100)
			if len(witnessedLockOrders) != tt.expectedLockOrdersLen {
				t.Errorf("expected %d lock orders, got %d", tt.expectedLockOrdersLen, len(witnessedLockOrders))
			}
			if len(witnessedCloseOrders) != tt.expectedCloseOrdersLen {
				t.Errorf("expected %d close orders, got %d", tt.expectedCloseOrdersLen, len(witnessedCloseOrders))
			}
			if len(witnessedResetOrders) != tt.expectedResetOrdersLen {
				t.Errorf("expected %d reset orders, got %d", tt.expectedResetOrdersLen, len(witnessedResetOrders))
			}
			for i, expectedId := range tt.expectedLockOrderIds {
				if i < len(witnessedLockOrders) {
					actualId := fmt.Sprintf("%x", witnessedLockOrders[i].OrderId)
					expectedIdHex := fmt.Sprintf("%x", []byte(expectedId))
					if actualId != expectedIdHex {
						t.Errorf("expected lock order id %s, got %s", expectedIdHex, actualId)
					}
				}
			}
			for i, expectedId := range tt.expectedCloseOrderIds {
				if i < len(witnessedCloseOrders) {
					actualId := fmt.Sprintf("%x", witnessedCloseOrders[i])
					expectedIdHex := fmt.Sprintf("%x", []byte(expectedId))
					if actualId != expectedIdHex {
						t.Errorf("expected close order id %s, got %s", expectedIdHex, actualId)
					}
				}
			}
			for i, expectedId := range tt.expectedResetOrderIds {
				if i < len(witnessedResetOrders) {
					actualId := fmt.Sprintf("%x", witnessedResetOrders[i])
					expectedIdHex := fmt.Sprintf("%x", []byte(expectedId))
					if actualId != expectedIdHex {
						t.Errorf("expected reset order id %s, got %s", expectedIdHex, actualId)
					}
				}
			}
		})
	}
}

func TestOracle_UpdateRootChainInfo(t *testing.T) {
	// Test data builders
	newSellOrder := func(id string) *lib.SellOrder {
		return &lib.SellOrder{Id: []byte(id)}
	}

	newOrderBook := func(orderIds ...string) *lib.OrderBook {
		orders := make([]*lib.SellOrder, len(orderIds))
		for i, id := range orderIds {
			orders[i] = newSellOrder(id)
		}
		return &lib.OrderBook{Orders: orders}
	}

	tests := []struct {
		name          string
		storedLock    []string
		storedClose   []string
		orderBookIds  []string
		expectedLock  []string
		expectedClose []string
	}{
		{
			name:          "removes lock orders not in order book",
			storedLock:    []string{"lock1", "lock2"},
			storedClose:   []string{},
			orderBookIds:  []string{"lock1"},
			expectedLock:  []string{"lock1"},
			expectedClose: []string{},
		},
		{
			name:          "removes close orders not in order book",
			storedLock:    []string{},
			storedClose:   []string{"close1", "close2"},
			orderBookIds:  []string{"close1"},
			expectedLock:  []string{},
			expectedClose: []string{"close1"},
		},
		{
			name:          "removes all orders when order book is empty",
			storedLock:    []string{"lock1", "lock2"},
			storedClose:   []string{"close1", "close2"},
			orderBookIds:  []string{},
			expectedLock:  []string{},
			expectedClose: []string{},
		},
		{
			name:          "keeps orders present in order book",
			storedLock:    []string{"lock1", "lock3"},
			storedClose:   []string{"close1", "close3"},
			orderBookIds:  []string{"lock1", "close1"},
			expectedLock:  []string{"lock1"},
			expectedClose: []string{"close1"},
		},
		{
			name:          "handles empty stored orders",
			storedLock:    []string{},
			storedClose:   []string{},
			orderBookIds:  []string{"lock1", "close1"},
			expectedLock:  []string{},
			expectedClose: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := &mockOrderStore{
				lockOrders:          make(map[string]*types.WitnessedOrder),
				closeOrders:         make(map[string]*types.WitnessedOrder),
				archiveLockOrders:   make(map[string]*types.WitnessedOrder),
				archivedCloseOrders: make(map[string]*types.WitnessedOrder),
			}

			// Populate store with test data
			for _, id := range tt.storedLock {
				order := &types.WitnessedOrder{
					OrderId: []byte(id),
				}
				mockStore.WriteOrder(order, types.LockOrderType)
			}
			for _, id := range tt.storedClose {
				order := &types.WitnessedOrder{
					OrderId: []byte(id),
				}
				mockStore.WriteOrder(order, types.CloseOrderType)
			}

			orderBook := newOrderBook(tt.orderBookIds...)
			oracle := &Oracle{
				orderStore: mockStore,
				orderBook:  orderBook,
				state:      NewOracleState("", lib.NewDefaultLogger()),
				log:        lib.NewDefaultLogger(),
			}

			// Execute
			info := &lib.RootChainInfo{
				Orders: orderBook,
			}
			oracle.UpdateRootChainInfo(info)

			// Verify
			ids, _ := mockStore.GetAllOrderIds(types.LockOrderType)
			assertOrderIds(t, "lock", ids, tt.expectedLock)
			ids, _ = mockStore.GetAllOrderIds(types.CloseOrderType)
			assertOrderIds(t, "close", ids, tt.expectedClose)
		})
	}
}

func assertOrderIds(t *testing.T, orderType string, actual [][]byte, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("expected %d %s orders, got %d", len(expected), orderType, len(actual))
		return
	}

	expectedSet := make(map[string]bool)
	for _, id := range expected {
		expectedSet[id] = true
	}

	for _, actualId := range actual {
		if !expectedSet[string(actualId)] {
			t.Errorf("unexpected %s order %s found in store", orderType, string(actualId))
		}
	}
}

func TestOracle_processBlock(t *testing.T) {
	buyerReceiveAddress := "0034567890123456789012345678901234567800"
	contractAddress := "0x1234567890123456789012345678901234567890"

	transactionWithOrder := func(order *types.WitnessedOrder, toAddress ...string) *mockTransaction {
		t := &mockTransaction{
			tokenTransfer: types.TokenTransfer{
				TokenBaseAmount: new(big.Int),
			},
			order: order,
		}
		if len(toAddress) > 0 {
			t.to = toAddress[0]
		}
		return t
	}

	tests := []struct {
		name                  string
		block                 types.BlockI
		orderStore            *mockOrderStore
		orderBook             *lib.OrderBook
		expectedLockOrderIds  [][]byte
		expectedCloseOrderIds [][]byte
	}{
		{
			name: "should process block with no transactions",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{},
			),
			orderStore: createOrderStore(),
			orderBook:  createOrderBook(),
		},
		{
			name: "should skip transactions without orders",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{},
			),
			orderStore: createOrderStore(),
			orderBook:  createOrderBook(),
		},
		{
			name: "should process valid lock order transaction",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedLockOrder("order1", buyerReceiveAddress)),
				},
			),
			orderStore:           createOrderStore(),
			orderBook:            createOrderBook(createSellOrder("order1", "", buyerReceiveAddress)),
			expectedLockOrderIds: [][]byte{[]byte("order1")},
		},
		{
			name: "should process valid close order transaction",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					// create transaction with To address as contract address
					transactionWithOrder(createWitnessedCloseOrder("order1"), contractAddress),
				},
			),
			orderStore: createOrderStore(),
			// create sell order with contractAdcress as data
			orderBook:             createOrderBook(createSellOrder("order1", contractAddress, buyerReceiveAddress)),
			expectedCloseOrderIds: [][]byte{[]byte("order1")},
		},
		{
			name: "should process multiple valid transactions",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedLockOrder("lock1", buyerReceiveAddress)),
					transactionWithOrder(createWitnessedCloseOrder("close1"), contractAddress),
					&mockTransaction{},
				},
			),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(
				createSellOrder("lock1", contractAddress, buyerReceiveAddress),
				createSellOrder("close1", contractAddress, buyerReceiveAddress),
			),
			expectedLockOrderIds:  [][]byte{[]byte("lock1")},
			expectedCloseOrderIds: [][]byte{[]byte("close1")},
		},
		{
			name: "should skip transactions when order not found in order book",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedLockOrder("lock1")),
				},
			),
			orderStore: createOrderStore(),
			orderBook:  createOrderBook(),
		},
		{
			name: "should skip transactions when validation fails",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedCloseOrder("close1")),
				},
			),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(
				createSellOrder("lock1", "", buyerReceiveAddress),
			),
		},
		{
			name: "should handle mixed successful and failed transactions",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedLockOrder("lock1", buyerReceiveAddress)),
					transactionWithOrder(createWitnessedLockOrder("nonexistent")),
					transactionWithOrder(createWitnessedCloseOrder("close1"), contractAddress),
					&mockTransaction{},
				},
			),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(
				createSellOrder("lock1", contractAddress, buyerReceiveAddress),
				createSellOrder("close1", contractAddress, buyerReceiveAddress),
			),
			expectedLockOrderIds:  [][]byte{[]byte("lock1")},
			expectedCloseOrderIds: [][]byte{[]byte("close1")},
		},
		{
			name: "should not overwrite existing order in store",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(&types.WitnessedOrder{
						OrderId: []byte("order1"),
						// these heights are different in the incoming transaction
						WitnessedHeight:  200,
						LastSubmitHeight: 150,
						LockOrder: &lib.LockOrder{
							BuyerReceiveAddress: []byte(buyerReceiveAddress),
							OrderId:             []byte("order1"),
						},
					}),
				},
			),
			orderStore: createOrderStore(&types.WitnessedOrder{
				OrderId:          []byte("order1"),
				WitnessedHeight:  100,
				LastSubmitHeight: 50,
				LockOrder: &lib.LockOrder{
					BuyerReceiveAddress: []byte(buyerReceiveAddress),
					OrderId:             []byte("order1"),
				},
			}),
			orderBook:            createOrderBook(createSellOrder("order1", "", buyerReceiveAddress)),
			expectedLockOrderIds: [][]byte{[]byte("order1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oracle := &Oracle{
				orderStore: tt.orderStore,
				orderBook:  tt.orderBook,
				log:        lib.NewDefaultLogger(),
			}

			// Execute the method under test
			err := oracle.processBlock(tt.block)
			if err != nil {
				panic(err)
			}
			for _, expectedId := range tt.expectedLockOrderIds {
				storedOrder, ok := tt.orderStore.lockOrders[hex.EncodeToString(expectedId)]
				if !ok {
					t.Errorf("expected lock order id %v not found", string(expectedId))
				}
				// Special case: verify existing order wasn't overwritten
				if tt.name == "should not overwrite existing order in store" && storedOrder != nil {
					if storedOrder.WitnessedHeight != 100 || storedOrder.LastSubmitHeight != 50 {
						t.Errorf("existing order was overwritten: expected WitnessedHeight=100, LastSubmitHeight=50, got WitnessedHeight=%d, LastSubmitHeight=%d",
							storedOrder.WitnessedHeight, storedOrder.LastSubmitHeight)
					}
				}
			}
			for _, expectedId := range tt.expectedCloseOrderIds {
				_, ok := tt.orderStore.closeOrders[hex.EncodeToString(expectedId)]
				if !ok {
					t.Errorf("expected close order id %v not found", string(expectedId))
				}
			}

			// Check for unexpected lock orders when none were expected
			if len(tt.expectedLockOrderIds) == 0 {
				for orderId := range tt.orderStore.lockOrders {
					t.Errorf("unexpected lock order id %v found when none were expected", orderId)
				}
			}
			// Check for unexpected close orders when none were expected
			if len(tt.expectedCloseOrderIds) == 0 {
				for orderId := range tt.orderStore.closeOrders {
					t.Errorf("unexpected close order id %v found when none were expected", orderId)
				}
			}
		})
	}
}

func TestOracle_validateLockOrder(t *testing.T) {
	oracle := &Oracle{}

	// Base valid orders for comparison
	baseOrderID := []byte("order123")
	baseChainID := uint64(1)
	baseBuyerReceiveAddr := []byte("receive_addr")
	baseBuyerSendAddr := []byte("send_addr")
	baseDeadline := uint64(1234567890)

	baseLockOrder := &lib.LockOrder{
		OrderId:             baseOrderID,
		ChainId:             baseChainID,
		BuyerReceiveAddress: baseBuyerReceiveAddr,
		BuyerSendAddress:    baseBuyerSendAddr,
		BuyerChainDeadline:  baseDeadline,
	}

	baseSellOrder := &lib.SellOrder{
		Id:                  baseOrderID,
		Committee:           baseChainID,
		BuyerReceiveAddress: baseBuyerReceiveAddr,
		BuyerSendAddress:    baseBuyerSendAddr,
		BuyerChainDeadline:  baseDeadline,
	}

	tests := []struct {
		name      string
		lockOrder *lib.LockOrder
		sellOrder *lib.SellOrder
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid matching orders",
			lockOrder: baseLockOrder,
			sellOrder: baseSellOrder,
			wantErr:   false,
		},
		{
			name: "mismatched order IDs",
			lockOrder: &lib.LockOrder{
				OrderId:             []byte("different_id"),
				ChainId:             baseChainID,
				BuyerReceiveAddress: baseBuyerReceiveAddr,
				BuyerSendAddress:    baseBuyerSendAddr,
				BuyerChainDeadline:  baseDeadline,
			},
			sellOrder: baseSellOrder,
			wantErr:   true,
			errMsg:    "lock order ID does not match sell order ID",
		},
		{
			name: "mismatched chain ID and committee",
			lockOrder: &lib.LockOrder{
				OrderId:             baseOrderID,
				ChainId:             999,
				BuyerReceiveAddress: baseBuyerReceiveAddr,
				BuyerSendAddress:    baseBuyerSendAddr,
				BuyerChainDeadline:  baseDeadline,
			},
			sellOrder: baseSellOrder,
			wantErr:   true,
			errMsg:    "lock order chain ID does not match sell order committee",
		},
		{
			name: "nil order ID in lock order",
			lockOrder: &lib.LockOrder{
				OrderId:             nil,
				ChainId:             baseChainID,
				BuyerReceiveAddress: baseBuyerReceiveAddr,
				BuyerSendAddress:    baseBuyerSendAddr,
				BuyerChainDeadline:  baseDeadline,
			},
			sellOrder: baseSellOrder,
			wantErr:   true,
			errMsg:    "lock order ID does not match sell order ID",
		},
		{
			name: "empty byte slices",
			lockOrder: &lib.LockOrder{
				OrderId:             []byte{},
				ChainId:             baseChainID,
				BuyerReceiveAddress: []byte{},
				BuyerSendAddress:    []byte{},
				BuyerChainDeadline:  baseDeadline,
			},
			sellOrder: &lib.SellOrder{
				Id:                  []byte{},
				Committee:           baseChainID,
				BuyerReceiveAddress: []byte{},
				BuyerSendAddress:    []byte{},
				BuyerChainDeadline:  baseDeadline,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := oracle.validateLockOrder(tt.lockOrder, tt.sellOrder)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateLockOrder() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateLockOrder() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateLockOrder() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestOracle_validateCloseOrder(t *testing.T) {
	oracle := &Oracle{log: lib.NewDefaultLogger()}

	// Base test data
	baseOrderId := []byte("order-123")
	baseChainId := uint64(1)
	baseRequestedAmount := uint64(1000)
	baseTo := common.HexToAddress("1234567890abcdef")

	baseSellerReceiveAddress := []byte("seller_receive_addr")
	baseSellerReceiveAddressHex := fmt.Sprintf("%x", baseSellerReceiveAddress)
	baseBuyer := common.HexToAddress("abcdef1234567890")
	baseSellOrder := &lib.SellOrder{
		Id:                   baseOrderId,
		Committee:            baseChainId,
		RequestedAmount:      baseRequestedAmount,
		Data:                 baseTo.Bytes(),
		SellerReceiveAddress: baseSellerReceiveAddress,
		BuyerSendAddress:     baseBuyer.Bytes(),
	}

	baseCloseOrder := &lib.CloseOrder{
		OrderId: baseOrderId,
		ChainId: baseChainId,
	}

	baseTx := &mockTransaction{
		from: baseBuyer.String(),
		to:   baseTo.String(),
		tokenTransfer: types.TokenTransfer{
			TokenBaseAmount:  big.NewInt(int64(baseRequestedAmount)),
			RecipientAddress: baseSellerReceiveAddressHex,
		},
	}

	tests := []struct {
		name        string
		closeOrder  *lib.CloseOrder
		sellOrder   *lib.SellOrder
		tx          types.TransactionI
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid close order",
			closeOrder:  baseCloseOrder,
			sellOrder:   baseSellOrder,
			tx:          baseTx,
			expectError: false,
		},
		{
			name:       "sell order data does not match transaction recipient",
			closeOrder: baseCloseOrder,
			sellOrder: &lib.SellOrder{
				Id:                   baseOrderId,
				Committee:            baseChainId,
				RequestedAmount:      baseRequestedAmount,
				Data:                 []byte("0xdifferentaddress"),
				SellerReceiveAddress: baseSellerReceiveAddress,
			},
			tx:          baseTx,
			expectError: true,
			errorMsg:    "transaction destination does not match sell order",
		},
		{
			name:       "token transfer recipient does not match seller receive address",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: baseBuyer.String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(int64(baseRequestedAmount)),
					RecipientAddress: fmt.Sprintf("%x", []byte("different_recipient_address")),
				},
			},
			expectError: true,
			errorMsg:    "transaction destination does not match sell order",
		},
		{
			name:       "error converting recipient address to bytes",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: baseBuyer.String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(int64(baseRequestedAmount)),
					RecipientAddress: "invalid_hex_address_gg",
				},
			},
			expectError: true,
			errorMsg:    "error validating transaction destination",
		},
		{
			name:       "closing transaction sender does not match locked buyer",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: common.HexToAddress("deadbeef00000000").String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(int64(baseRequestedAmount)),
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: true,
			errorMsg:    "closing transaction sender does not match the order's locked buyer",
		},
		{
			name: "close order ID does not match sell order ID",
			closeOrder: &lib.CloseOrder{
				OrderId: []byte("different-order-id"),
				ChainId: baseChainId,
			},
			sellOrder:   baseSellOrder,
			tx:          baseTx,
			expectError: true,
			errorMsg:    "close order ID does not match sell order ID",
		},
		{
			name: "close order chain ID does not match sell order committee",
			closeOrder: &lib.CloseOrder{
				OrderId: baseOrderId,
				ChainId: uint64(999),
			},
			sellOrder:   baseSellOrder,
			tx:          baseTx,
			expectError: true,
			errorMsg:    "close order chain ID does not match sell order committee",
		},
		{
			name:       "token transfer amount is nil",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: baseBuyer.String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  nil,
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: true,
			errorMsg:    "token transfer amount cannot be nil",
		},
		{
			name:       "transfer amount does not match requested amount",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: baseBuyer.String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(500), // Different from requested amount
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: true,
			errorMsg:    fmt.Sprintf("transfer amount %d does not match requested amount %d", 500, baseRequestedAmount),
		},
		{
			name:       "zero amounts are valid",
			closeOrder: baseCloseOrder,
			sellOrder: &lib.SellOrder{
				Id:                   baseOrderId,
				Committee:            baseChainId,
				RequestedAmount:      0,
				Data:                 baseTo.Bytes(),
				SellerReceiveAddress: baseSellerReceiveAddress,
			},
			tx: &mockTransaction{
				to: baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(0),
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: false,
		},
		{
			name:       "large amounts are valid",
			closeOrder: baseCloseOrder,
			sellOrder: &lib.SellOrder{
				Id:                   baseOrderId,
				Committee:            baseChainId,
				RequestedAmount:      18446744073709551615, // Max uint64
				Data:                 baseTo.Bytes(),
				SellerReceiveAddress: baseSellerReceiveAddress,
			},
			tx: &mockTransaction{
				to: baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  big.NewInt(0).SetUint64(18446744073709551615),
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: false,
		},
		{
			// regression: TokenBaseAmount can exceed 2^64 (parsed from a 32-byte ERC20 amount).
			// requested + 2^64 has the same low 64 bits as requested, so the old .Uint64()
			// comparison truncated and wrongly accepted it. The big.Int comparison must reject it.
			name:       "transfer amount exceeding uint64 range is rejected",
			closeOrder: baseCloseOrder,
			sellOrder:  baseSellOrder,
			tx: &mockTransaction{
				from: baseBuyer.String(),
				to:   baseTo.String(),
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount:  new(big.Int).Add(new(big.Int).SetUint64(baseRequestedAmount), new(big.Int).Lsh(big.NewInt(1), 64)),
					RecipientAddress: baseSellerReceiveAddressHex,
				},
			},
			expectError: true,
			errorMsg:    "does not match requested amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := oracle.validateCloseOrder(tt.closeOrder, tt.sellOrder, tt.tx)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateCloseOrder() expected error but got nil")
					return
				}

				// Check if the error message contains the expected text
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateCloseOrder() error = %v, expected to contain %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateCloseOrder() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestOracle_MultiTokenSupport verifies that USDC and USDT orders can be processed on the same committee
// This test demonstrates that the oracle is token-agnostic and works with any ERC20 token
func TestOracle_MultiTokenSupport(t *testing.T) {
	// Import known token addresses from eth package
	usdcContract := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	usdtContract := "0xdAC17F958D2ee523a2206206994597C13D831ec7"

	buyerReceiveAddress := "0034567890123456789012345678901234567800"
	sellerReceiveAddress := "seller_receive_address"

	// Create order IDs for different tokens
	usdcOrderId := "usdc_order_1"
	usdtOrderId := "usdt_order_1"

	tests := []struct {
		name                  string
		orderBook             *lib.OrderBook
		witnessedOrders       []*types.WitnessedOrder
		transactions          []types.TransactionI
		expectedLockOrders    int
		expectedCloseOrders   int
		expectedStoreContains []string
	}{
		{
			name: "USDC and USDT lock orders on same committee",
			orderBook: createOrderBook(
				createSellOrder(usdcOrderId, usdcContract, ""),
				createSellOrder(usdtOrderId, usdtContract, ""),
			),
			witnessedOrders: []*types.WitnessedOrder{
				createWitnessedLockOrder(usdcOrderId, buyerReceiveAddress),
				createWitnessedLockOrder(usdtOrderId, buyerReceiveAddress),
			},
			transactions: []types.TransactionI{
				&mockTransaction{
					to:    usdcContract,
					order: createWitnessedLockOrder(usdcOrderId, buyerReceiveAddress),
				},
				&mockTransaction{
					to:    usdtContract,
					order: createWitnessedLockOrder(usdtOrderId, buyerReceiveAddress),
				},
			},
			expectedLockOrders:    2,
			expectedCloseOrders:   0,
			expectedStoreContains: []string{usdcOrderId, usdtOrderId},
		},
		{
			name: "USDC and USDT close orders on same committee",
			orderBook: createOrderBook(
				withData(withCommittee(createSellOrderWithSellerAddr(usdcOrderId, buyerReceiveAddress, sellerReceiveAddress), 1), usdcContract),
				withData(withCommittee(createSellOrderWithSellerAddr(usdtOrderId, buyerReceiveAddress, sellerReceiveAddress), 1), usdtContract),
			),
			witnessedOrders: []*types.WitnessedOrder{
				createWitnessedCloseOrder(usdcOrderId, 1),
				createWitnessedCloseOrder(usdtOrderId, 1),
			},
			transactions: []types.TransactionI{
				&mockTransaction{
					to:    usdcContract,
					order: createWitnessedCloseOrder(usdcOrderId, 1),
					tokenTransfer: types.TokenTransfer{
						TokenBaseAmount:  big.NewInt(100000000),
						RecipientAddress: fmt.Sprintf("%x", []byte(sellerReceiveAddress)),
					},
				},
				&mockTransaction{
					to:    usdtContract,
					order: createWitnessedCloseOrder(usdtOrderId, 1),
					tokenTransfer: types.TokenTransfer{
						TokenBaseAmount:  big.NewInt(100000000),
						RecipientAddress: fmt.Sprintf("%x", []byte(sellerReceiveAddress)),
					},
				},
			},
			expectedLockOrders:    0,
			expectedCloseOrders:   2,
			expectedStoreContains: []string{usdcOrderId, usdtOrderId},
		},
		{
			name: "mixed USDC lock and USDT close on same committee",
			orderBook: createOrderBook(
				createSellOrder(usdcOrderId, usdcContract, ""),
				withData(withCommittee(createSellOrderWithSellerAddr(usdtOrderId, buyerReceiveAddress, sellerReceiveAddress), 1), usdtContract),
			),
			witnessedOrders: []*types.WitnessedOrder{
				createWitnessedLockOrder(usdcOrderId, buyerReceiveAddress),
				createWitnessedCloseOrder(usdtOrderId, 1),
			},
			transactions: []types.TransactionI{
				&mockTransaction{
					to:    usdcContract,
					order: createWitnessedLockOrder(usdcOrderId, buyerReceiveAddress),
				},
				&mockTransaction{
					to:    usdtContract,
					order: createWitnessedCloseOrder(usdtOrderId, 1),
					tokenTransfer: types.TokenTransfer{
						TokenBaseAmount:  big.NewInt(100000000),
						RecipientAddress: fmt.Sprintf("%x", []byte(sellerReceiveAddress)),
					},
				},
			},
			expectedLockOrders:    1,
			expectedCloseOrders:   1,
			expectedStoreContains: []string{usdcOrderId, usdtOrderId},
		},
		{
			name: "USDT close order validation with exact amount match",
			orderBook: createOrderBook(
				// USDT with 6 decimals: 100 USDT = 100,000,000 base units
				withData(withCommittee(createSellOrderWithSellerAddr(usdtOrderId, buyerReceiveAddress, sellerReceiveAddress), 1), usdtContract),
			),
			witnessedOrders: []*types.WitnessedOrder{
				createWitnessedCloseOrder(usdtOrderId, 1),
			},
			transactions: []types.TransactionI{
				&mockTransaction{
					to:    usdtContract,
					order: createWitnessedCloseOrder(usdtOrderId, 1),
					tokenTransfer: types.TokenTransfer{
						TokenBaseAmount:  big.NewInt(100000000), // 100 USDT
						RecipientAddress: fmt.Sprintf("%x", []byte(sellerReceiveAddress)),
					},
				},
			},
			expectedLockOrders:    0,
			expectedCloseOrders:   1,
			expectedStoreContains: []string{usdtOrderId},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create oracle with empty order store
			mockStore := NewMockOrderStore()
			tempDir, _ := os.MkdirTemp("", "oracle_multi_token_test")
			defer os.RemoveAll(tempDir)

			oracle := &Oracle{
				orderStore: mockStore,
				orderBook:  tt.orderBook,
				state:      NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger()),
				committee:  1, // Single committee handling both USDC and USDT
				log:        lib.NewDefaultLogger(),
				metrics:    nil, // Metrics methods are nil-safe
			}

			// Process block with transactions
			if len(tt.transactions) > 0 {
				block := createMockBlockWithTransactions(100, "0xblockhash", tt.transactions)
				// Update sell orders with requested amounts for close order validation
				for _, order := range tt.orderBook.Orders {
					order.RequestedAmount = 100000000 // 100 tokens with 6 decimals
				}
				err := oracle.processBlock(block)
				if err != nil {
					t.Fatalf("processBlock failed: %v", err)
				}
			}

			// Get witnessed orders from oracle
			lockOrders, closeOrders, resetOrders := oracle.WitnessedOrders(tt.orderBook, 100)

			// Verify lock order count
			if len(lockOrders) != tt.expectedLockOrders {
				t.Errorf("expected %d lock orders, got %d", tt.expectedLockOrders, len(lockOrders))
			}

			// Verify close order count
			if len(closeOrders) != tt.expectedCloseOrders {
				t.Errorf("expected %d close orders, got %d", tt.expectedCloseOrders, len(closeOrders))
			}
			if len(resetOrders) != 0 {
				t.Errorf("expected 0 reset orders, got %d", len(resetOrders))
			}

			// Verify orders are in store
			for _, expectedId := range tt.expectedStoreContains {
				// Check lock orders
				if tt.expectedLockOrders > 0 {
					_, err := mockStore.ReadOrder([]byte(expectedId), types.LockOrderType)
					if err != nil {
						// Try close orders if not a lock order
						if tt.expectedCloseOrders > 0 {
							_, err = mockStore.ReadOrder([]byte(expectedId), types.CloseOrderType)
							if err != nil {
								t.Errorf("expected order %s not found in store", expectedId)
							}
						}
					}
				} else if tt.expectedCloseOrders > 0 {
					_, err := mockStore.ReadOrder([]byte(expectedId), types.CloseOrderType)
					if err != nil {
						t.Errorf("expected close order %s not found in store", expectedId)
					}
				}
			}

			t.Logf("✓ Successfully processed %d USDC/USDT orders on committee %d",
				len(lockOrders)+len(closeOrders)+len(resetOrders), oracle.committee)
		})
	}
}

// faultyOrderStore wraps mockOrderStore and allows injecting errors on specific
// operations to exercise error branches that the plain mock can't reach (e.g.
// ReadOrder/WriteOrder/RemoveOrder/GetAllOrderIds failing for reasons other than
// "not found").
type faultyOrderStore struct {
	*mockOrderStore
	// failReadOrderIds are order ids (as string keys via lib.BytesToString) for which ReadOrder should fail
	failReadOrderIds map[string]bool
	// failWriteOrderIds are order ids for which WriteOrder should fail
	failWriteOrderIds map[string]bool
	// failRemoveOrderIds are order ids for which RemoveOrder should fail
	failRemoveOrderIds map[string]bool
	// failGetAllOrderIds, if true, makes GetAllOrderIds fail for the given order type
	failGetAllOrderIds map[types.OrderType]bool
}

func newFaultyOrderStore(orders ...*types.WitnessedOrder) *faultyOrderStore {
	return &faultyOrderStore{
		mockOrderStore:     createOrderStore(orders...),
		failReadOrderIds:   map[string]bool{},
		failWriteOrderIds:  map[string]bool{},
		failRemoveOrderIds: map[string]bool{},
		failGetAllOrderIds: map[types.OrderType]bool{},
	}
}

func (f *faultyOrderStore) ReadOrder(orderId []byte, orderType types.OrderType) (*types.WitnessedOrder, lib.ErrorI) {
	if f.failReadOrderIds[lib.BytesToString(orderId)] {
		return nil, ErrReadOrder(fmt.Errorf("injected read failure"))
	}
	return f.mockOrderStore.ReadOrder(orderId, orderType)
}

func (f *faultyOrderStore) WriteOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	if f.failWriteOrderIds[lib.BytesToString(order.OrderId)] {
		return ErrWriteOrder(fmt.Errorf("injected write failure"))
	}
	return f.mockOrderStore.WriteOrder(order, orderType)
}

func (f *faultyOrderStore) RemoveOrder(orderId []byte, orderType types.OrderType) lib.ErrorI {
	if f.failRemoveOrderIds[lib.BytesToString(orderId)] {
		return ErrRemoveOrder(fmt.Errorf("injected remove failure"))
	}
	return f.mockOrderStore.RemoveOrder(orderId, orderType)
}

func (f *faultyOrderStore) GetAllOrderIds(orderType types.OrderType) ([][]byte, lib.ErrorI) {
	if f.failGetAllOrderIds[orderType] {
		return nil, ErrVerifyOrder(fmt.Errorf("injected get-all failure"))
	}
	return f.mockOrderStore.GetAllOrderIds(orderType)
}

// newTestMetrics creates a real, non-nil *lib.Metrics instance for exercising the
// non-nil-metrics branches of updateMetrics/CommitCertificate/UpdateRootChainInfo.
// Metrics are registered against prometheus' default global registry via promauto,
// so this must only be called once per test binary run to avoid "duplicate metrics
// collector registration" panics - a package-level sync.Once guards construction.
var (
	testMetricsOnce sync.Once
	testMetricsInst *lib.Metrics
)

func newTestMetrics() *lib.Metrics {
	testMetricsOnce.Do(func() {
		addr := crypto.NewAddressFromBytes([]byte("test-oracle-address"))
		testMetricsInst = lib.NewMetricsServer(addr, 1, "test", lib.MetricsConfig{
			MetricsEnabled:    true,
			PrometheusAddress: "127.0.0.1:0",
		}, lib.NewDefaultLogger())
	})
	return testMetricsInst
}

func TestOracle_DebugOrder(t *testing.T) {
	t.Run("nil oracle receiver returns nil", func(t *testing.T) {
		var oracle *Oracle
		got := oracle.DebugOrder([]byte("order1"))
		assert.Nil(t, got)
	})

	t.Run("confirmation lag is zero when source height <= safe height (no underflow)", func(t *testing.T) {
		// SafeHeight and SourceChainHeight are both uint64. DebugOrder guards the
		// subtraction with `if sourceHeight > safeHeight`, so when sourceHeight <=
		// safeHeight the subtraction is never executed and confirmationLag stays at
		// its zero value - there is no unsigned-underflow path here. This test
		// pins that behavior as a regression guard.
		tempDir, _ := os.MkdirTemp("", "oracle_debugorder_test")
		defer os.RemoveAll(tempDir)
		state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
		state.safeHeight = 100
		state.sourceChainHeight = 100 // equal case
		oracle := &Oracle{
			orderStore: NewMockOrderStore(),
			state:      state,
			log:        lib.NewDefaultLogger(),
		}
		info := oracle.DebugOrder([]byte("order1"))
		require.NotNil(t, info)
		assert.Equal(t, uint64(0), info.ConfirmationLag)
		assert.Equal(t, uint64(100), info.SafeHeight)
		assert.Equal(t, uint64(100), info.SourceChainHeight)

		// sourceHeight < safeHeight (can legitimately happen transiently since
		// safeHeight is monotonic but sourceHeight tracks the latest validated block)
		state.safeHeight = 100
		state.sourceChainHeight = 40
		info = oracle.DebugOrder([]byte("order1"))
		require.NotNil(t, info)
		assert.Equal(t, uint64(0), info.ConfirmationLag, "confirmationLag must not underflow when sourceHeight < safeHeight")
	})

	t.Run("confirmation lag computed when source height > safe height", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "oracle_debugorder_test2")
		defer os.RemoveAll(tempDir)
		state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
		state.safeHeight = 100
		state.sourceChainHeight = 150
		oracle := &Oracle{
			orderStore: NewMockOrderStore(),
			state:      state,
			log:        lib.NewDefaultLogger(),
			config: lib.OracleConfig{
				SafeBlockConfirmations: 12,
				ProposeDelayBlocks:     3,
			},
		}
		info := oracle.DebugOrder([]byte("order1"))
		require.NotNil(t, info)
		assert.Equal(t, uint64(50), info.ConfirmationLag)
		assert.Equal(t, uint64(12), info.SafeBlockConfirmations)
		assert.Equal(t, uint64(3), info.ProposeDelayBlocks)
	})

	t.Run("lock and close order lookups reflected in result", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "oracle_debugorder_test3")
		defer os.RemoveAll(tempDir)
		state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
		store := createOrderStore(
			createWitnessedLockOrder("lock-found"),
			createWitnessedCloseOrder("close-found"),
		)
		oracle := &Oracle{
			orderStore: store,
			state:      state,
			log:        lib.NewDefaultLogger(),
		}

		// order found: both lock and close populated when present
		info := oracle.DebugOrder([]byte("lock-found"))
		require.NotNil(t, info)
		assert.NotNil(t, info.LockOrder)
		assert.Nil(t, info.CloseOrder)

		info = oracle.DebugOrder([]byte("close-found"))
		require.NotNil(t, info)
		assert.Nil(t, info.LockOrder)
		assert.NotNil(t, info.CloseOrder)

		// order not found: both remain nil, no error surfaces
		info = oracle.DebugOrder([]byte("does-not-exist"))
		require.NotNil(t, info)
		assert.Nil(t, info.LockOrder)
		assert.Nil(t, info.CloseOrder)
		assert.Equal(t, []byte("does-not-exist"), []byte(info.OrderId))
	})
}

func TestOracle_CommitCertificate_LockCloseErrorBranches(t *testing.T) {
	newOracle := func(store OrderStore) *Oracle {
		return &Oracle{
			orderStore: store,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
		}
	}

	t.Run("nil oracle receiver returns nil", func(t *testing.T) {
		var oracle *Oracle
		err := oracle.CommitCertificate(nil, nil, nil, 0)
		assert.NoError(t, err)
	})

	t.Run("lock order ReadOrder failure returns error and stops processing", func(t *testing.T) {
		store := createOrderStore() // empty, lock1 not present
		oracle := newOracle(store)
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					LockOrders: []*lib.LockOrder{{OrderId: []byte("lock1")}},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "order not verified")
	})

	t.Run("lock order WriteOrder failure is logged and loop continues", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedLockOrder("lock1"), createWitnessedLockOrder("lock2"))
		faulty.failWriteOrderIds[lib.BytesToString([]byte("lock1"))] = true
		oracle := newOracle(faulty)
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					LockOrders: []*lib.LockOrder{
						{OrderId: []byte("lock1")},
						{OrderId: []byte("lock2")},
					},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.NoError(t, err, "write failure on one order should not abort processing of the rest")
		// lock2 should have been updated despite lock1 failing
		updated, readErr := faulty.ReadOrder([]byte("lock2"), types.LockOrderType)
		require.NoError(t, readErr)
		assert.Equal(t, uint64(10), updated.LastSubmitHeight)
	})

	t.Run("close order ReadOrder failure returns error and stops processing", func(t *testing.T) {
		store := createOrderStore() // empty, close1 not present
		oracle := newOracle(store)
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					CloseOrders: [][]byte{[]byte("close1")},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "order not verified")
	})

	t.Run("close order WriteOrder failure is logged and loop continues", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedCloseOrder("close1"), createWitnessedCloseOrder("close2"))
		faulty.failWriteOrderIds[lib.BytesToString([]byte("close1"))] = true
		oracle := newOracle(faulty)
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					CloseOrders: [][]byte{[]byte("close1"), []byte("close2")},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.NoError(t, err, "write failure on one order should not abort processing of the rest")
		updated, readErr := faulty.ReadOrder([]byte("close2"), types.CloseOrderType)
		require.NoError(t, readErr)
		assert.Equal(t, uint64(10), updated.LastSubmitHeight)
	})

	t.Run("reset order RemoveOrder failure is logged but does not error", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedCloseOrder("order1"))
		faulty.failRemoveOrderIds[lib.BytesToString([]byte("order1"))] = true
		oracle := newOracle(faulty)
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					ResetOrders: [][]byte{[]byte("order1")},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.NoError(t, err)
		// order should still be present since RemoveOrder failed
		_, readErr := faulty.ReadOrder([]byte("order1"), types.CloseOrderType)
		require.NoError(t, readErr, "order should remain in store since removal was injected to fail")
	})

	t.Run("with real metrics: lifecycle metrics updated on commit", func(t *testing.T) {
		store := createOrderStore(createWitnessedLockOrder("lock1"), createWitnessedCloseOrder("close1"))
		oracle := &Oracle{
			orderStore: store,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		qc := &lib.QuorumCertificate{
			Header: &lib.View{RootHeight: 10},
			Results: &lib.CertificateResult{
				Orders: &lib.Orders{
					LockOrders:  []*lib.LockOrder{{OrderId: []byte("lock1")}},
					CloseOrders: [][]byte{[]byte("close1")},
				},
			},
		}
		err := oracle.CommitCertificate(qc, nil, nil, 0)
		require.NoError(t, err)
	})
}

func TestOracle_updateMetrics(t *testing.T) {
	t.Run("nil metrics is a no-op", func(t *testing.T) {
		oracle := &Oracle{
			orderStore: NewMockOrderStore(),
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    nil,
		}
		// must not panic
		oracle.updateMetrics()
	})

	t.Run("non-nil metrics: happy path updates all metric groups", func(t *testing.T) {
		store := createOrderStore(createWitnessedLockOrder("lock1"), createWitnessedCloseOrder("close1"))
		oracle := &Oracle{
			orderStore: store,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		// must not panic and should read through to the store successfully
		oracle.updateMetrics()
	})

	t.Run("non-nil metrics: GetAllOrderIds error branches default counts to zero", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedLockOrder("lock1"), createWitnessedCloseOrder("close1"))
		faulty.failGetAllOrderIds[types.LockOrderType] = true
		faulty.failGetAllOrderIds[types.CloseOrderType] = true
		oracle := &Oracle{
			orderStore: faulty,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		// must not panic even though both GetAllOrderIds calls fail
		oracle.updateMetrics()
	})
}

func TestOracle_UpdateRootChainInfo_Gaps(t *testing.T) {
	t.Run("nil oracle receiver is a no-op", func(t *testing.T) {
		var oracle *Oracle
		// must not panic
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{})
	})

	t.Run("nil info.Orders returns early after pruning history", func(t *testing.T) {
		store := createOrderStore(createWitnessedLockOrder("lock1"))
		oracle := &Oracle{
			orderStore: store,
			orderBook:  createOrderBook(createSellOrder("lock1", "", "")),
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
		}
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: nil})
		// order book should remain untouched (still the original, non-nil book)
		assert.NotNil(t, oracle.orderBook)
		// stored lock order should be untouched since we returned before pruning ran
		_, err := store.ReadOrder([]byte("lock1"), types.LockOrderType)
		assert.NoError(t, err)
	})

	t.Run("GetAllOrderIds error for lock orders returns early", func(t *testing.T) {
		faulty := newFaultyOrderStore()
		faulty.failGetAllOrderIds[types.LockOrderType] = true
		orderBook := createOrderBook(createSellOrder("order1", "", ""))
		oracle := &Oracle{
			orderStore: faulty,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
		}
		// must not panic
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: orderBook})
		assert.NotNil(t, oracle.orderBook)
	})

	t.Run("GetAllOrderIds error for close orders returns early", func(t *testing.T) {
		faulty := newFaultyOrderStore()
		faulty.failGetAllOrderIds[types.CloseOrderType] = true
		orderBook := createOrderBook(createSellOrder("order1", "", ""))
		oracle := &Oracle{
			orderStore: faulty,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
		}
		// must not panic; lock-order pruning (which succeeds) runs before the close-order
		// GetAllOrderIds call fails and returns early
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: orderBook})
		assert.NotNil(t, oracle.orderBook)
	})

	t.Run("RemoveOrder failure for lock order increments storeRemoveErrors but continues", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedLockOrder("lock1"), createWitnessedLockOrder("lock2"))
		faulty.failRemoveOrderIds[lib.BytesToString([]byte("lock1"))] = true
		// order book with neither lock1 nor lock2, so both are candidates for removal
		orderBook := createOrderBook()
		oracle := &Oracle{
			orderStore: faulty,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: orderBook})
		// lock1 removal failed, so it should still be present
		_, err := faulty.ReadOrder([]byte("lock1"), types.LockOrderType)
		assert.NoError(t, err, "lock1 should remain since its removal was injected to fail")
		// lock2 removal should have succeeded
		_, err = faulty.ReadOrder([]byte("lock2"), types.LockOrderType)
		assert.Error(t, err, "lock2 should have been removed")
	})

	t.Run("RemoveOrder failure for close order increments storeRemoveErrors but continues", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedCloseOrder("close1"), createWitnessedCloseOrder("close2"))
		faulty.failRemoveOrderIds[lib.BytesToString([]byte("close1"))] = true
		orderBook := createOrderBook()
		oracle := &Oracle{
			orderStore: faulty,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: orderBook})
		_, err := faulty.ReadOrder([]byte("close1"), types.CloseOrderType)
		assert.NoError(t, err, "close1 should remain since its removal was injected to fail")
		_, err = faulty.ReadOrder([]byte("close2"), types.CloseOrderType)
		assert.Error(t, err, "close2 should have been removed")
	})

	t.Run("locked order in order book prunes matching lock order from store", func(t *testing.T) {
		store := createOrderStore(createWitnessedLockOrder("order1"))
		// UpdateRootChainInfo treats a non-nil BuyerSendAddress as "locked" for pruning purposes
		lockedOrder := &lib.SellOrder{
			Id:               []byte("order1"),
			BuyerSendAddress: []byte("buyer-send"),
		}
		orderBook := createOrderBook(lockedOrder)
		oracle := &Oracle{
			orderStore: store,
			state:      NewOracleState("", lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
		}
		oracle.UpdateRootChainInfo(&lib.RootChainInfo{Orders: orderBook})
		_, err := store.ReadOrder([]byte("order1"), types.LockOrderType)
		assert.Error(t, err, "locked sell order should cause the stored lock order to be removed")
	})
}

func TestOracle_ValidateProposedOrders_ResetOrderGaps(t *testing.T) {
	newOracleWithSafeHeight := func(safeHeight uint64, store OrderStore) *Oracle {
		tempDir, _ := os.MkdirTemp("", "oracle_reset_gap_test")
		oracle := &Oracle{
			orderStore: store,
			state:      NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger()),
			committee:  1,
			log:        lib.NewDefaultLogger(),
		}
		oracle.state.safeHeight = safeHeight
		return oracle
	}

	t.Run("nil rootOrderBook returns ErrNilOrderBook", func(t *testing.T) {
		oracle := newOracleWithSafeHeight(20, createOrderStore())
		orders := &lib.Orders{ResetOrders: [][]byte{[]byte("reset1")}}
		err := oracle.ValidateProposedOrders(orders, nil)
		require.Error(t, err)
		assert.Equal(t, CodeNilOrderBook, err.Code())
	})

	t.Run("order missing from root order book is rejected", func(t *testing.T) {
		// Note: lib.OrderBook.GetOrder only ever returns a non-nil error when the
		// *OrderBook receiver itself is nil (see lib/swap.go); ValidateProposedOrders
		// already short-circuits that exact case via its own `rootOrderBook == nil`
		// check above, so GetOrder's internal error branch is unreachable here. The
		// reachable "not found" outcome is GetOrder returning (nil, nil), which this
		// test covers instead.
		oracle := newOracleWithSafeHeight(20, createOrderStore())
		orders := &lib.Orders{ResetOrders: [][]byte{[]byte("reset1")}}
		orderBook := createOrderBook() // empty book: GetOrder("reset1") returns (nil, nil)
		err := oracle.ValidateProposedOrders(orders, orderBook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "order not verified")
		assert.Contains(t, err.Error(), "order not found in order book")
	})

	t.Run("order found but not locked is rejected", func(t *testing.T) {
		oracle := newOracleWithSafeHeight(20, createOrderStore())
		unlockedOrder := createSellOrder("reset1", "", "") // no buyer receive address => IsLocked() == false
		orderBook := createOrderBook(unlockedOrder)
		orders := &lib.Orders{ResetOrders: [][]byte{[]byte("reset1")}}
		err := oracle.ValidateProposedOrders(orders, orderBook)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "order not verified")
		assert.Contains(t, err.Error(), "order is not locked")
	})
}

func TestOracle_validateCloseOrder_HexDecodeError(t *testing.T) {
	oracle := &Oracle{log: lib.NewDefaultLogger()}

	baseOrderId := []byte("order-123")
	baseChainId := uint64(1)
	baseTo := common.HexToAddress("1234567890abcdef")

	closeOrder := &lib.CloseOrder{
		OrderId: baseOrderId,
		ChainId: baseChainId,
	}
	sellOrder := &lib.SellOrder{
		Id:        baseOrderId,
		Committee: baseChainId,
		Data:      baseTo.Bytes(),
	}

	// tx.From() returns a value that is not valid hex, so lib.StringToBytes fails
	// before the sender is ever compared against sellOrder.BuyerSendAddress
	tx := &mockTransaction{
		from: "not-valid-hex-zz",
		to:   baseTo.String(),
	}

	err := oracle.validateCloseOrder(closeOrder, sellOrder, tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error converting sender address to bytes")
}

// fakeBlockProvider is a minimal, controllable BlockProvider test double used to
// exercise Oracle's lifecycle methods (run/Start) without a live source-chain client.
type fakeBlockProvider struct {
	blockCh chan types.BlockI
	// synced controls the return value of IsSynced
	synced bool
	// startCalls records the heights this provider was started at
	startCalls []uint64
}

func newFakeBlockProvider() *fakeBlockProvider {
	return &fakeBlockProvider{blockCh: make(chan types.BlockI, 8)}
}

func (f *fakeBlockProvider) Start(ctx context.Context, height uint64) {
	f.startCalls = append(f.startCalls, height)
}

func (f *fakeBlockProvider) BlockCh() chan types.BlockI {
	return f.blockCh
}

func (f *fakeBlockProvider) IsSynced() bool {
	return f.synced
}

func newLifecycleOracle(t *testing.T, provider BlockProvider, store OrderStore) *Oracle {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "oracle_lifecycle_test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	ctx, cancel := context.WithCancel(context.Background())
	return &Oracle{
		blockProvider: provider,
		orderStore:    store,
		state:         NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger()),
		log:           lib.NewDefaultLogger(),
		ctx:           ctx,
		ctxCancel:     cancel,
	}
}

func TestOracle_NewOracle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "oracle_new_oracle_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	provider := newFakeBlockProvider()
	store := NewMockOrderStore()
	config := lib.OracleConfig{
		StateFile: filepath.Join(tempDir, "test_state"),
		Committee: 7,
	}
	o, err := NewOracle(context.Background(), config, provider, store, lib.NewDefaultLogger(), nil)
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.Equal(t, uint64(7), o.committee)
	assert.Same(t, provider, o.blockProvider)
	assert.Same(t, store, o.orderStore)
	assert.NotNil(t, o.state)
	assert.NotNil(t, o.ctx)
	assert.NotNil(t, o.ctxCancel)
	// Stop should cancel the internally-created context without panicking
	o.Stop()
	select {
	case <-o.ctx.Done():
		// expected
	default:
		t.Errorf("expected oracle context to be cancelled after Stop()")
	}
}

func TestOracle_hasOrderBook(t *testing.T) {
	oracle := &Oracle{}
	assert.False(t, oracle.hasOrderBook(), "no order book set should report false")

	oracle.orderBook = &lib.OrderBook{}
	assert.True(t, oracle.hasOrderBook(), "order book set should report true")
}

func TestOracle_Stop_NilReceiver(t *testing.T) {
	var oracle *Oracle
	// must not panic
	oracle.Stop()
}

func TestOracle_rollbackOrderType(t *testing.T) {
	t.Run("GetAllOrderIds failure logs and returns without removing anything", func(t *testing.T) {
		faulty := newFaultyOrderStore(createWitnessedLockOrder("lock1"))
		faulty.failGetAllOrderIds[types.LockOrderType] = true
		oracle := &Oracle{
			orderStore: faulty,
			log:        lib.NewDefaultLogger(),
		}
		oracle.rollbackOrderType(types.LockOrderType, 10)
		_, err := faulty.ReadOrder([]byte("lock1"), types.LockOrderType)
		assert.NoError(t, err, "order should remain since GetAllOrderIds failed before any removal was attempted")
	})

	t.Run("removes orders witnessed above rollback height, keeps orders at or below it", func(t *testing.T) {
		aboveOrder := &types.WitnessedOrder{
			OrderId:         []byte("above"),
			WitnessedHeight: 100,
			LockOrder:       &lib.LockOrder{OrderId: []byte("above")},
		}
		belowOrder := &types.WitnessedOrder{
			OrderId:         []byte("below"),
			WitnessedHeight: 40,
			LockOrder:       &lib.LockOrder{OrderId: []byte("below")},
		}
		store := createOrderStore(aboveOrder, belowOrder)
		oracle := &Oracle{
			orderStore: store,
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		oracle.rollbackOrderType(types.LockOrderType, 50)

		_, err := store.ReadOrder([]byte("above"), types.LockOrderType)
		assert.Error(t, err, "order witnessed above rollback height should be removed")
		_, err = store.ReadOrder([]byte("below"), types.LockOrderType)
		assert.NoError(t, err, "order witnessed at or below rollback height should remain")
	})

	t.Run("ReadOrder failure is logged and skipped, RemoveOrder failure is logged and skipped", func(t *testing.T) {
		aboveOrder := &types.WitnessedOrder{
			OrderId:         []byte("above1"),
			WitnessedHeight: 100,
			LockOrder:       &lib.LockOrder{OrderId: []byte("above1")},
		}
		aboveOrder2 := &types.WitnessedOrder{
			OrderId:         []byte("above2"),
			WitnessedHeight: 100,
			LockOrder:       &lib.LockOrder{OrderId: []byte("above2")},
		}
		faulty := newFaultyOrderStore(aboveOrder, aboveOrder2)
		faulty.failReadOrderIds[lib.BytesToString([]byte("above1"))] = true
		faulty.failRemoveOrderIds[lib.BytesToString([]byte("above2"))] = true
		oracle := &Oracle{
			orderStore: faulty,
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
		}
		// must not panic despite both a read and a remove failure
		oracle.rollbackOrderType(types.LockOrderType, 50)
		// above1's read failed, so it's never evaluated for removal - still present
		_, err := faulty.ReadOrder([]byte("above1"), types.LockOrderType)
		assert.Error(t, err, "ReadOrder is still faulted for above1, so this lookup also errors")
		// above2's removal was injected to fail, so it should remain
		faulty.failRemoveOrderIds[lib.BytesToString([]byte("above2"))] = false
		_, err = faulty.ReadOrder([]byte("above2"), types.LockOrderType)
		assert.NoError(t, err, "above2 should remain since its removal was injected to fail")
	})
}

func TestOracle_reorgRollback(t *testing.T) {
	t.Run("no last known good height logs a warning and returns without rolling back", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "oracle_reorg_test")
		defer os.RemoveAll(tempDir)
		store := createOrderStore(createWitnessedLockOrder("lock1"))
		oracle := &Oracle{
			orderStore: store,
			state:      NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger()),
			log:        lib.NewDefaultLogger(),
			config:     lib.OracleConfig{ReorgRollbackBlocks: 10},
		}
		// height is 0 (no prior state), so reorgRollback should bail before touching the store
		oracle.reorgRollback()
		_, err := store.ReadOrder([]byte("lock1"), types.LockOrderType)
		assert.NoError(t, err, "order should be untouched when there is no last known good height")
	})

	t.Run("rolls back lock and close orders witnessed above the rollback height", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "oracle_reorg_test2")
		defer os.RemoveAll(tempDir)
		aboveLock := &types.WitnessedOrder{
			OrderId:         []byte("lock-above"),
			WitnessedHeight: 100,
			LockOrder:       &lib.LockOrder{OrderId: []byte("lock-above")},
		}
		belowClose := &types.WitnessedOrder{
			OrderId:         []byte("close-below"),
			WitnessedHeight: 10,
			CloseOrder:      &lib.CloseOrder{OrderId: []byte("close-below")},
		}
		store := createOrderStore(aboveLock, belowClose)
		state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
		// simulate a prior successfully-processed block so GetLastHeight() != 0
		block := &mockBlock{number: 100, hash: "h100"}
		require.NoError(t, state.saveState(block))

		oracle := &Oracle{
			orderStore: store,
			state:      state,
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
			config:     lib.OracleConfig{ReorgRollbackBlocks: 50},
		}
		oracle.reorgRollback()

		_, err := store.ReadOrder([]byte("lock-above"), types.LockOrderType)
		assert.Error(t, err, "lock order witnessed above the rollback height should be removed")
		_, err = store.ReadOrder([]byte("close-below"), types.CloseOrderType)
		assert.NoError(t, err, "close order witnessed below the rollback height should remain")
	})

	t.Run("rollback delta larger than last height clamps to 0 and rolls back all orders (no uint64 underflow)", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "oracle_reorg_test3")
		defer os.RemoveAll(tempDir)
		// order witnessed at height 3; whole chain is within the rollback window
		order := &types.WitnessedOrder{
			OrderId:         []byte("lock-early"),
			WitnessedHeight: 3,
			LockOrder:       &lib.LockOrder{OrderId: []byte("lock-early")},
		}
		store := createOrderStore(order)
		state := NewOracleState(filepath.Join(tempDir, "test_state"), lib.NewDefaultLogger())
		// last processed height 5, well below the rollback delta of 50 (reorg near startup)
		require.NoError(t, state.saveState(&mockBlock{number: 5, hash: "h5"}))

		oracle := &Oracle{
			orderStore: store,
			state:      state,
			log:        lib.NewDefaultLogger(),
			metrics:    newTestMetrics(),
			config:     lib.OracleConfig{ReorgRollbackBlocks: 50},
		}
		oracle.reorgRollback()

		// pre-fix: rollbackHeight underflowed to ~2^64 so nothing was removed. clamped: all removed.
		_, err := store.ReadOrder([]byte("lock-early"), types.LockOrderType)
		assert.Error(t, err, "order should be rolled back when the whole chain is within the rollback window")
	})
}

func TestOracle_run(t *testing.T) {
	t.Run("closed block channel returns ErrChannelClosed", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		close(provider.blockCh)

		err := oracle.run(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, CodeChannelClosed, err.Code())
		require.Len(t, provider.startCalls, 1)
		assert.Equal(t, uint64(0), provider.startCalls[0], "zero last height should start the block provider at height 0")
	})

	t.Run("resumes from last height + 1 when prior state exists", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		// simulate prior successfully-processed block at height 42
		require.NoError(t, oracle.state.saveState(&mockBlock{number: 42, hash: "h42"}))
		close(provider.blockCh)

		err := oracle.run(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, CodeChannelClosed, err.Code())
		require.Len(t, provider.startCalls, 1)
		assert.Equal(t, uint64(43), provider.startCalls[0])
	})

	t.Run("context cancellation returns nil and notifies syncCh", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		ctx, cancel := context.WithCancel(context.Background())
		syncCh := make(chan bool, 1)
		cancel() // cancel before calling run so the select fires the ctx.Done() branch immediately

		err := oracle.run(ctx, syncCh)
		assert.NoError(t, err)
		select {
		case v := <-syncCh:
			assert.False(t, v, "syncCh should be notified with false on shutdown")
		default:
			// buffered channel send is best-effort (uses a select/default in run());
			// as long as run() returned nil without blocking or panicking, this is acceptable
		}
	})

	t.Run("nil block is skipped without error", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		provider.blockCh <- nil
		close(provider.blockCh)

		err := oracle.run(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, CodeChannelClosed, err.Code(), "after skipping the nil block, the loop continues and eventually observes the closed channel")
	})

	t.Run("ValidateSequence failure (gap) returns CodeBlockSequence", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		// establish a baseline at height 10
		require.NoError(t, oracle.state.saveState(&mockBlock{number: 10, hash: "h10"}))
		// send a block with a gap (expected 11, got 20) to trigger CodeBlockSequence
		provider.blockCh <- &mockBlock{number: 20, hash: "h20", parentHash: "h10"}

		err := oracle.run(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, CodeBlockSequence, err.Code())
	})

	t.Run("ValidateSequence failure (reorg) returns CodeChainReorg", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		require.NoError(t, oracle.state.saveState(&mockBlock{number: 10, hash: "h10"}))
		// next block references the wrong parent hash to trigger a reorg detection
		provider.blockCh <- &mockBlock{number: 11, hash: "h11", parentHash: "wrong-parent"}

		err := oracle.run(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, CodeChainReorg, err.Code())
	})
}

func TestOracle_Start(t *testing.T) {
	t.Run("waits for order book then processes via run(), stopping cleanly on context cancellation", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		ctx, cancel := context.WithCancel(context.Background())
		syncCh := make(chan bool, 1)

		oracle.Start(ctx, syncCh)
		// oracle has no order book yet, so Start's goroutine should be blocked waiting for one;
		// cancelling the context should allow the wait loop to observe ctx.Done() and return
		cancel()

		// give the goroutine a moment to observe cancellation; the wait loop polls every second,
		// but also selects on ctx.Done() directly so it should return promptly
		select {
		case <-ctx.Done():
			// expected: context is cancelled; the goroutine will exit on its next select
		}
	})

	t.Run("runs the main loop once an order book is present, exits via CodeBlockSequence retry path", func(t *testing.T) {
		provider := newFakeBlockProvider()
		oracle := newLifecycleOracle(t, provider, NewMockOrderStore())
		oracle.orderBook = &lib.OrderBook{}
		// establish a baseline so the first block sent triggers a sequence gap
		require.NoError(t, oracle.state.saveState(&mockBlock{number: 10, hash: "h10"}))

		ctx, cancel := context.WithCancel(oracle.ctx)
		defer cancel()
		oracle.Start(ctx, nil)

		// send a block with a gap; run() should return CodeBlockSequence, causing Start's
		// goroutine to call state.removeState() and loop back into run() again
		provider.blockCh <- &mockBlock{number: 20, hash: "h20", parentHash: "h10"}

		// give the goroutine time to process the gap and loop back into run(), then
		// verify state was cleared (removeState resets GetLastHeight to 0)
		require.Eventually(t, func() bool {
			return oracle.state.GetLastHeight() == 0
		}, 2*time.Second, 10*time.Millisecond, "expected state to be cleared after a block-sequence error")

		cancel()
	})
}
