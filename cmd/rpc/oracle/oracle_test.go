package oracle

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
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

func createMockBlockWithTransactions(blockNumber uint64, blockHash string, transactions []types.TransactionI) types.BlockI {
	return &mockBlock{
		number:       blockNumber,
		hash:         blockHash,
		transactions: transactions,
	}
}

// createSellOrder creates a sell order with an optional buyer receive address
func createSellOrder(orderIdHex string, buyerReceiveAddress ...string) *lib.SellOrder {
	orderIdBytes := []byte(orderIdHex)
	sellOrder := &lib.SellOrder{
		Id: orderIdBytes,
	}
	if len(buyerReceiveAddress) > 0 && len(buyerReceiveAddress[0]) > 0 {
		sellOrder.BuyerReceiveAddress = []byte(buyerReceiveAddress[0])
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

func createOrderBook(orders ...*lib.SellOrder) *lib.OrderBook {
	orderBook := &lib.OrderBook{}
	for _, order := range orders {
		orderBook.Orders = append(orderBook.Orders, order)
	}
	return orderBook
}

func TestOracle_ValidateProposedOrders(t *testing.T) {
	orders := func(lockOrderIDs []string, closeOrderIDs []string) *lib.Orders {
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
		return &lib.Orders{
			LockOrders:  lockOrders,
			CloseOrders: closeOrders,
		}
	}

	tests := []struct {
		name          string
		orders        *lib.Orders
		orderStore    types.OrderStore
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
			orders:        orders(nil, nil),
			orderStore:    createOrderStore(createWitnessedLockOrder("lock1")),
			expectedError: false,
		},
		{
			name:   "valid lock orders should pass validation",
			orders: orders([]string{"lock1", "lock2"}, nil),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1"),
				createWitnessedLockOrder("lock2"),
			),
			expectedError: false,
		},
		{
			name:   "valid close orders should pass validation",
			orders: orders(nil, []string{"close1", "close2"}),
			orderStore: createOrderStore(
				createWitnessedCloseOrder("close1", 1),
				createWitnessedCloseOrder("close2", 1),
			),
			expectedError: false,
		},
		{
			name:   "valid mixed orders should pass validation",
			orders: orders([]string{"lock1"}, []string{"close1"}),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1"),
				createWitnessedCloseOrder("close1", 1),
			),
			expectedError: false,
		},
		{
			name:   "lock order verification failure should return error",
			orders: orders([]string{"lock1"}, nil),
			orderStore: createOrderStore(
				createWitnessedLockOrder("lock1", "address"),
			),
			expectedError: true,
			errorContains: "order not verified",
		},
		{
			name:   "close order verification failure should return error",
			orders: orders(nil, []string{"close1"}),
			orderStore: createOrderStore(
				createWitnessedCloseOrder("close1"), // missing chain id
			),
			expectedError: true,
			errorContains: "order not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Oracle instance
			oracle := &Oracle{
				orderStore: tt.orderStore,
				committee:  1,
				log:        lib.NewDefaultLogger(),
			}

			// Execute test
			err := oracle.ValidateProposedOrders(tt.orders)

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

func TestOracle_WitnessedOrders(t *testing.T) {
	buyerReceiveAddress := "buyer_recv_address"

	orderIdOne := "order1"
	orderIdTwo := "order2"

	notLocked := ""

	tests := []struct {
		name                   string
		orderStore             types.OrderStore
		orderBook              *lib.OrderBook
		orderBookOrders        []*lib.SellOrder
		storeLockOrders        map[string]*lib.LockOrder
		storeCloseOrders       map[string]*lib.CloseOrder
		expectedLockOrdersLen  int
		expectedCloseOrdersLen int
		expectedLockOrderIds   []string
		expectedCloseOrderIds  []string
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
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
		},
		{
			name: "orders exist in order book but not in store.",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, notLocked),
				createSellOrder(orderIdTwo, "buyerReceiveAddres"),
			),
			orderStore:             NewMockOrderStore(),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
		},
		{
			name: "matching stored lock order, should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, notLocked),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
				createWitnessedLockOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{},
		},
		{
			name: "matching close order exists in store and order book. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedCloseOrder(orderIdOne),
				createWitnessedCloseOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdOne},
		},
		{
			name: "unlocked order exists in order book and matching lock order exists in store. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, notLocked),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{},
		},
		{
			name: "locked order exists in order book and matching close order exists in store. should be included",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedCloseOrder(orderIdOne),
			),
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdOne},
		},
		{
			name: "mixed scenario with multiple orders",
			orderBook: createOrderBook(
				createSellOrder(orderIdOne, notLocked),
				createSellOrder(orderIdTwo, buyerReceiveAddress),
			),
			orderStore: createOrderStore(
				createWitnessedLockOrder(orderIdOne),
				createWitnessedCloseOrder(orderIdTwo),
			),
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{orderIdTwo},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := lib.NewDefaultLogger()
			oracle := &Oracle{
				orderStore:   tt.orderStore,
				stateManager: NewOracleState("file", log),
				log:          log,
			}
			// 100 to specify a high enough root height that shouldSubmit always passes
			witnessedLockOrders, witnessedCloseOrders := oracle.WitnessedOrders(tt.orderBook, 100)
			if len(witnessedLockOrders) != tt.expectedLockOrdersLen {
				t.Errorf("expected %d lock orders, got %d", tt.expectedLockOrdersLen, len(witnessedLockOrders))
			}
			if len(witnessedCloseOrders) != tt.expectedCloseOrdersLen {
				t.Errorf("expected %d close orders, got %d", tt.expectedCloseOrdersLen, len(witnessedCloseOrders))
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
	buyerReceiveAddress := "buyer_recv_address"

	transactionWithOrder := func(order *types.WitnessedOrder) *mockTransaction {
		return &mockTransaction{
			tokenTransfer: types.TokenTransfer{
				TokenBaseAmount: new(big.Int),
			},
			order: order,
		}
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
			orderBook:            createOrderBook(createSellOrder("order1", buyerReceiveAddress)),
			expectedLockOrderIds: [][]byte{[]byte("order1")},
		},
		{
			name: "should process valid close order transaction",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedCloseOrder("order1")),
				},
			),
			orderStore:            createOrderStore(),
			orderBook:             createOrderBook(createSellOrder("order1", buyerReceiveAddress)),
			expectedCloseOrderIds: [][]byte{[]byte("order1")},
		},
		{
			name: "should process multiple valid transactions",
			block: createMockBlockWithTransactions(
				100,
				"0xblockhash",
				[]types.TransactionI{
					transactionWithOrder(createWitnessedLockOrder("lock1", buyerReceiveAddress)),
					transactionWithOrder(createWitnessedCloseOrder("close1")),
					&mockTransaction{},
				},
			),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(
				createSellOrder("lock1", buyerReceiveAddress),
				createSellOrder("close1", buyerReceiveAddress),
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
				createSellOrder("lock1", buyerReceiveAddress),
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
					transactionWithOrder(createWitnessedCloseOrder("close1")),
					&mockTransaction{},
				},
			),
			orderStore: createOrderStore(),
			orderBook: createOrderBook(
				createSellOrder("lock1", buyerReceiveAddress),
				createSellOrder("close1", buyerReceiveAddress),
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
			orderBook:            createOrderBook(createSellOrder("order1", buyerReceiveAddress)),
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
	oracle := &Oracle{}

	// Base test data
	baseOrderId := []byte("order-123")
	baseChainId := uint64(1)
	baseRequestedAmount := uint64(1000)
	baseTo := "1234567890abcdef"

	baseData, _ := lib.StringToBytes(baseTo)

	baseSellOrder := &lib.SellOrder{
		Id:              baseOrderId,
		Committee:       baseChainId,
		RequestedAmount: baseRequestedAmount,
		Data:            baseData,
	}

	baseCloseOrder := &lib.CloseOrder{
		OrderId: baseOrderId,
		ChainId: baseChainId,
	}

	baseTx := &mockTransaction{
		to: "0x" + baseTo,
		tokenTransfer: types.TokenTransfer{
			TokenBaseAmount: big.NewInt(int64(baseRequestedAmount)),
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
				Id:              baseOrderId,
				Committee:       baseChainId,
				RequestedAmount: baseRequestedAmount,
				Data:            []byte("0xdifferentaddress"),
			},
			tx:          baseTx,
			expectError: true,
			errorMsg:    "sell order data field does not match transaction recipient",
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
				to: baseTo,
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount: nil,
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
				to: baseTo,
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount: big.NewInt(500), // Different from requested amount
				},
			},
			expectError: true,
			errorMsg:    fmt.Sprintf("transfer amount %d does not match requested amount %d", 500, baseRequestedAmount),
		},
		{
			name:       "zero amounts are valid",
			closeOrder: baseCloseOrder,
			sellOrder: &lib.SellOrder{
				Id:              baseOrderId,
				Committee:       baseChainId,
				RequestedAmount: 0,
				Data:            baseData,
			},
			tx: &mockTransaction{
				to: baseTo,
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount: big.NewInt(0),
				},
			},
			expectError: false,
		},
		{
			name:       "large amounts are valid",
			closeOrder: baseCloseOrder,
			sellOrder: &lib.SellOrder{
				Id:              baseOrderId,
				Committee:       baseChainId,
				RequestedAmount: 18446744073709551615, // Max uint64
				Data:            baseData,
			},
			tx: &mockTransaction{
				to: baseTo,
				tokenTransfer: types.TokenTransfer{
					TokenBaseAmount: big.NewInt(0).SetUint64(18446744073709551615),
				},
			},
			expectError: false,
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
