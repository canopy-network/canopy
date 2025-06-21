package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOrderStore struct {
	lockOrders  map[string][]byte
	closeOrders map[string][]byte

	verifyErr error
}

func (m *mockOrderStore) VerifyOrder(orderId []byte, orderType types.OrderType, data []byte) error {
	orderIdStr := string(orderId)
	if orderType == types.LockOrderType {
		if existingData, exists := m.lockOrders[orderIdStr]; exists {
			if !bytes.Equal(existingData, data) {
				return fmt.Errorf("order data mismatch for lock order %s", orderIdStr)
			}
		} else {
			return fmt.Errorf("lock order %s not found", orderIdStr)
		}
	} else if orderType == types.CloseOrderType {
		if existingData, exists := m.closeOrders[orderIdStr]; exists {
			if !bytes.Equal(existingData, data) {
				fmt.Println(string(existingData), string(data))
				return fmt.Errorf("order data mismatch for close order %s", orderIdStr)
			}
		} else {
			return fmt.Errorf("close order %s not found", orderIdStr)
		}
	}
	return nil
}

func (m *mockOrderStore) WriteOrder(orderId []byte, orderType types.OrderType, data []byte) error {
	if orderType == types.LockOrderType {
		m.lockOrders[string(orderId)] = data
	} else if orderType == types.CloseOrderType {
		m.closeOrders[string(orderId)] = data
	}
	return nil
}

func (m *mockOrderStore) ReadOrder(orderId []byte, orderType types.OrderType) ([]byte, error) {
	if orderType == types.LockOrderType {
		for key, data := range m.lockOrders {
			if bytes.Equal([]byte(key), orderId) {
				return data, nil
			}
		}
		return nil, fmt.Errorf("lock order not found")
	}
	if orderType == types.CloseOrderType {
		for key, data := range m.closeOrders {
			if bytes.Equal([]byte(key), orderId) {
				return data, nil
			}
		}
		return nil, fmt.Errorf("close order not found")
	}
	return nil, fmt.Errorf("unknown order type")
}

func (m *mockOrderStore) RemoveOrder(orderId []byte, orderType types.OrderType) error {
	if orderType == types.LockOrderType {
		for key := range m.lockOrders {
			if bytes.Equal([]byte(key), orderId) {
				delete(m.lockOrders, key)
				break
			}
		}
	} else if orderType == types.CloseOrderType {
		for key := range m.closeOrders {
			if bytes.Equal([]byte(key), orderId) {
				delete(m.closeOrders, key)
				break
			}
		}
	}
	return nil
}

func (m *mockOrderStore) GetAllOrderIds(orderType types.OrderType) [][]byte {
	var ids [][]byte
	if orderType == types.LockOrderType {
		for id := range m.lockOrders {
			ids = append(ids, []byte(id))
		}
	} else if orderType == types.CloseOrderType {
		for id := range m.closeOrders {
			ids = append(ids, []byte(id))
		}
	}
	return ids
}

type mockBlock struct {
	number       uint64
	hash         string
	transactions []types.TransactionI
}

func (m *mockBlock) Number() uint64 {
	return m.number
}

func (m *mockBlock) Hash() string {
	return m.hash
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

func TestOracle_processLockOrderTransaction(t *testing.T) {
	validOrderId := []byte("test-order-id")
	validOrderIdHex := string(validOrderId)
	validLockOrder := lib.LockOrder{
		OrderId:             validOrderId,
		BuyerReceiveAddress: []byte("buyer-receive-address"),
		BuyerSendAddress:    []byte("buyer-send-address"),
	}
	validLockOrderData, _ := json.Marshal(validLockOrder)

	invalidOrderId := []byte("invalid-order-id")
	invalidLockOrder := lib.LockOrder{
		OrderId:             invalidOrderId,
		BuyerReceiveAddress: []byte("buyer-receive-address"),
		BuyerSendAddress:    []byte("buyer-send-address"),
	}
	invalidLockOrderData, _ := json.Marshal(invalidLockOrder)

	validSellOrder := &lib.SellOrder{
		Id:                   validOrderId,
		Committee:            1,
		AmountForSale:        1000,
		RequestedAmount:      500,
		SellerReceiveAddress: []byte("seller-receive-address"),
		BuyerSendAddress:     nil,
		BuyerReceiveAddress:  nil,
	}

	validOrderBook := &lib.OrderBook{
		ChainId: 1,
		Orders:  []*lib.SellOrder{validSellOrder},
	}

	emptyOrderBook := &lib.OrderBook{
		ChainId: 1,
		Orders:  []*lib.SellOrder{},
	}

	tests := []struct {
		name            string
		transaction     types.TransactionI
		orderBook       *lib.OrderBook
		orderBookError  lib.ErrorI
		expectedError   bool
		expectedOrderId string
	}{
		{
			name: "successfully processes valid lock order transaction",
			transaction: &mockTransaction{
				blockchain: "ethereum",
				from:       "0x123",
				to:         "0x123",
				data:       validLockOrderData,
				hash:       "tx-hash-1",
			},
			orderBook:       validOrderBook,
			orderBookError:  nil,
			expectedError:   false,
			expectedOrderId: validOrderIdHex,
		},
		{
			name: "fails when order id not found in order book",
			transaction: &mockTransaction{
				blockchain: "ethereum",
				from:       "0x123",
				to:         "0x123",
				data:       invalidLockOrderData,
				hash:       "tx-hash-2",
			},
			orderBook:       emptyOrderBook,
			orderBookError:  nil,
			expectedError:   true,
			expectedOrderId: "",
		},
		{
			name: "fails when order book reader returns error",
			transaction: &mockTransaction{
				blockchain: "ethereum",
				from:       "0x123",
				to:         "0x123",
				data:       validLockOrderData,
				hash:       "tx-hash-3",
			},
			orderBook:       nil,
			orderBookError:  lib.NewError(1, "test", "order book error"),
			expectedError:   true,
			expectedOrderId: "",
		},
		{
			name: "fails when transaction data is invalid json",
			transaction: &mockTransaction{
				blockchain: "ethereum",
				from:       "0x123",
				to:         "0x123",
				data:       []byte("invalid json"),
				hash:       "tx-hash-4",
			},
			orderBook:       validOrderBook,
			orderBookError:  nil,
			expectedError:   true,
			expectedOrderId: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorer := &mockOrderStore{
				lockOrders: map[string][]byte{},
			}

			bl := &Oracle{
				orderStore: mockStorer,
				orderBook:  tt.orderBook,
				logger:     lib.NewDefaultLogger(),
			}

			err := bl.processLockOrderTransaction(tt.transaction, 0)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			if !tt.expectedError && tt.expectedOrderId != "" {
				_, exists := mockStorer.lockOrders[tt.expectedOrderId]
				if !exists {
					fmt.Println(tt.expectedOrderId, mockStorer.lockOrders)
					t.Errorf("expected order to be stored with id %s", tt.expectedOrderId)
				}
			}
		})
	}
}

func TestOracle_processCloseOrderTransaction(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	validOrderId := []byte(testId)
	validOrderIdHex := string(validOrderId)
	validCloseOrder := lib.CloseOrder{
		OrderId: validOrderId,
	}
	validCloseOrderData, _ := json.Marshal(validCloseOrder)

	invalidOrderId := []byte("invalid-order-id")
	invalidCloseOrder := lib.CloseOrder{
		OrderId: invalidOrderId,
	}
	invalidCloseOrderData, _ := json.Marshal(invalidCloseOrder)

	ethTokenInfo := types.TokenInfo{
		Name:     "Ethereum",
		Symbol:   "ETH",
		Decimals: 18,
	}

	validTokenTransfer := types.TokenTransfer{
		Blockchain:       "ethereum",
		TokenInfo:        ethTokenInfo,
		TransactionID:    "tx-123",
		SenderAddress:    "0x456",
		RecipientAddress: "0x789",
		TokenAmount:      1.5,
		TokenBaseAmount:  1000,
		ContractAddress:  "0xabc",
	}

	mismatchTokenTransfer := types.TokenTransfer{
		Blockchain:       "ethereum",
		TokenInfo:        ethTokenInfo,
		TransactionID:    "tx-456",
		SenderAddress:    "0x456",
		RecipientAddress: "0x789",
		TokenAmount:      2.0,
		TokenBaseAmount:  2000,
		ContractAddress:  "0xabc",
	}

	validSellOrder := &lib.SellOrder{
		Id:                   validOrderId,
		Committee:            1,
		AmountForSale:        1500,
		RequestedAmount:      1000,
		SellerReceiveAddress: []byte("seller-receive-address"),
		BuyerSendAddress:     nil,
		BuyerReceiveAddress:  nil,
	}

	validOrderBook := &lib.OrderBook{
		ChainId: 1,
		Orders:  []*lib.SellOrder{validSellOrder},
	}

	emptyOrderBook := &lib.OrderBook{
		ChainId: 1,
		Orders:  []*lib.SellOrder{},
	}

	tests := []struct {
		name            string
		transaction     types.TransactionI
		tokenTransfer   types.TokenTransfer
		orderBook       *lib.OrderBook
		orderBookError  lib.ErrorI
		expectedError   bool
		expectedOrderId string
	}{
		{
			name: "successfully processes valid close order transaction",
			transaction: &mockTransaction{
				blockchain:    "ethereum",
				from:          "0x456",
				to:            "0x789",
				data:          validCloseOrderData,
				hash:          "tx-hash-1",
				tokenTransfer: validTokenTransfer,
			},
			tokenTransfer:   validTokenTransfer,
			orderBook:       validOrderBook,
			orderBookError:  nil,
			expectedError:   false,
			expectedOrderId: validOrderIdHex,
		},
		{
			name: "fails when order id not found in order book",
			transaction: &mockTransaction{
				blockchain:    "ethereum",
				from:          "0x456",
				to:            "0x789",
				data:          invalidCloseOrderData,
				hash:          "tx-hash-2",
				tokenTransfer: validTokenTransfer,
			},
			tokenTransfer:   validTokenTransfer,
			orderBook:       emptyOrderBook,
			orderBookError:  nil,
			expectedError:   true,
			expectedOrderId: "",
		},
		{
			name: "fails when token transfer amount does not match order amount",
			transaction: &mockTransaction{
				blockchain:    "ethereum",
				from:          "0x456",
				to:            "0x789",
				data:          validCloseOrderData,
				hash:          "tx-hash-3",
				tokenTransfer: mismatchTokenTransfer,
			},
			tokenTransfer:   mismatchTokenTransfer,
			orderBook:       validOrderBook,
			orderBookError:  nil,
			expectedError:   true,
			expectedOrderId: "",
		},
		{
			name: "fails when order book reader returns error",
			transaction: &mockTransaction{
				blockchain:    "ethereum",
				from:          "0x456",
				to:            "0x789",
				data:          validCloseOrderData,
				hash:          "tx-hash-4",
				tokenTransfer: validTokenTransfer,
			},
			tokenTransfer:   validTokenTransfer,
			orderBook:       nil,
			orderBookError:  lib.NewError(1, "test", "order book error"),
			expectedError:   true,
			expectedOrderId: "",
		},
		{
			name: "fails when transaction data is invalid json",
			transaction: &mockTransaction{
				blockchain:    "ethereum",
				from:          "0x456",
				to:            "0x789",
				data:          []byte("invalid json"),
				hash:          "tx-hash-5",
				tokenTransfer: validTokenTransfer,
			},
			tokenTransfer:   validTokenTransfer,
			orderBook:       validOrderBook,
			orderBookError:  nil,
			expectedError:   true,
			expectedOrderId: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorer := &mockOrderStore{
				closeOrders: map[string][]byte{},
			}

			bl := &Oracle{
				orderStore: mockStorer,
				orderBook:  tt.orderBook,
				logger:     lib.NewDefaultLogger(),
			}

			err := bl.processCloseOrderTransaction(tt.transaction, 0, tt.tokenTransfer)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			if !tt.expectedError && tt.expectedOrderId != "" {
				_, exists := mockStorer.closeOrders[tt.expectedOrderId]
				if !exists {
					t.Errorf("expected order to be stored with id %s", tt.expectedOrderId)
				}
			}
		})
	}
}

func TestOracle_ValidateProposedOrders(t *testing.T) {
	buildOrders := func(lockOrderIDs []string, closeOrderIDs []string) *lib.Orders {
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

	setupStore := func(lockOrderIds []string, closeOrderIds []string) *mockOrderStore {
		store := &mockOrderStore{
			lockOrders:  make(map[string][]byte),
			closeOrders: make(map[string][]byte),
		}
		for _, orderId := range lockOrderIds {
			order := &lib.LockOrder{
				OrderId: []byte(orderId),
			}
			marshalled, _ := json.Marshal(order)
			store.lockOrders[orderId] = marshalled
		}
		for _, orderId := range closeOrderIds {
			order := &lib.CloseOrder{
				OrderId:    []byte(orderId),
				CloseOrder: true,
			}
			marshalled, _ := json.Marshal(order)
			store.closeOrders[orderId] = marshalled
		}
		return store
	}

	emptyOrders := &lib.Orders{
		LockOrders:  []*lib.LockOrder{},
		CloseOrders: [][]byte{},
	}

	tests := []struct {
		name          string
		orders        *lib.Orders
		store         *mockOrderStore
		expectedError bool
		errorContains string
	}{
		{
			name:          "nil orders should return nil",
			orders:        nil,
			store:         setupStore(nil, []string{}),
			expectedError: false,
		},
		{
			name:          "empty orders should return nil",
			orders:        emptyOrders,
			store:         setupStore([]string{"lock1"}, nil),
			expectedError: false,
		},
		{
			name:          "valid lock orders should pass validation",
			orders:        buildOrders([]string{"lock1", "lock2"}, nil),
			store:         setupStore([]string{"lock1", "lock2"}, nil),
			expectedError: false,
		},
		{
			name:          "valid close orders should pass validation",
			orders:        buildOrders(nil, []string{"close1", "close2"}),
			store:         setupStore(nil, []string{"close1", "close2"}),
			expectedError: false,
		},
		{
			name:          "valid mixed orders should pass validation",
			orders:        buildOrders([]string{"lock1"}, []string{"close1"}),
			store:         setupStore([]string{"lock1"}, []string{"close1"}),
			expectedError: false,
		},
		{
			name:          "lock order verification failure should return error",
			orders:        buildOrders([]string{"lock1"}, nil),
			store:         setupStore(nil, []string{}),
			expectedError: true,
			errorContains: "order not verified",
		},
		{
			name:          "close order verification failure should return error",
			orders:        buildOrders(nil, []string{"close1"}),
			store:         setupStore(nil, []string{}),
			expectedError: true,
			errorContains: "order not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Oracle instance
			oracle := &Oracle{
				orderStore: tt.store,
				logger:     lib.NewDefaultLogger(),
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

func createLockOrder(orderIdHex string) *lib.LockOrder {
	orderIdBytes := []byte(orderIdHex)
	lockOrder := &lib.LockOrder{
		OrderId: orderIdBytes,
	}
	return lockOrder
}

func createCloseOrder(orderIdHex string) *lib.CloseOrder {
	orderIdBytes := []byte(orderIdHex)
	closeOrder := &lib.CloseOrder{
		OrderId:    orderIdBytes,
		CloseOrder: true,
	}
	return closeOrder
}

func createSellOrder(orderIdHex string, isLocked bool) *lib.SellOrder {
	orderIdBytes := []byte(orderIdHex)
	sellOrder := &lib.SellOrder{
		Id: orderIdBytes,
	}
	if isLocked {
		sellOrder.BuyerSendAddress = []byte("buyer_address")
	}
	return sellOrder
}

func createOrderBook(orders []*lib.SellOrder) *lib.OrderBook {
	return &lib.OrderBook{
		ChainId: 1,
		Orders:  orders,
	}
}

func TestOracle_GatherWitnessedOrders(t *testing.T) {
	orderIdOne := "order1"
	orderIdTwo := "order2"
	orderIdThree := "order3"
	orderIdFour := "order4"
	orderIdFive := "order5"
	orderIdSix := "order6"
	orderIdSeven := "order7"
	orderIdEight := "order8"
	unlockedSellOrderOne := createSellOrder(orderIdOne, false)
	lockedSellOrderThree := createSellOrder(orderIdThree, true)
	unlockedSellOrderFive := createSellOrder(orderIdFive, false)
	lockedSellOrderSix := createSellOrder(orderIdSix, true)
	unlockedSellOrderSeven := createSellOrder(orderIdSeven, false)
	lockedSellOrderEight := createSellOrder(orderIdEight, true)
	lockOrderOne := createLockOrder(orderIdOne)
	lockOrderTwo := createLockOrder(orderIdTwo)
	closeOrderThree := createCloseOrder(orderIdThree)
	closeOrderFour := createCloseOrder(orderIdFour)
	lockOrderFive := createLockOrder(orderIdFive)
	closeOrderSix := createCloseOrder(orderIdSix)
	tests := []struct {
		name                   string
		orderBookOrders        []*lib.SellOrder
		storeLockOrders        map[string]*lib.LockOrder
		storeCloseOrders       map[string]*lib.CloseOrder
		expectedLockOrdersLen  int
		expectedCloseOrdersLen int
		expectedLockOrderIds   []string
		expectedCloseOrderIds  []string
	}{
		{
			name:                   "orders exist in store but not in order book. both lock and close orders should not be included in proposed block",
			orderBookOrders:        []*lib.SellOrder{},
			storeLockOrders:        map[string]*lib.LockOrder{orderIdOne: lockOrderOne},
			storeCloseOrders:       map[string]*lib.CloseOrder{orderIdThree: closeOrderThree},
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
		},
		{
			name:                   "orders exist in order book but not in store. both lock and close orders no order should be included in proposed block",
			orderBookOrders:        []*lib.SellOrder{unlockedSellOrderOne, lockedSellOrderThree},
			storeLockOrders:        map[string]*lib.LockOrder{},
			storeCloseOrders:       map[string]*lib.CloseOrder{},
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{},
		},
		{
			name:                   "duplicate locked order exists in store and order book. should not be included",
			orderBookOrders:        []*lib.SellOrder{unlockedSellOrderOne},
			storeLockOrders:        map[string]*lib.LockOrder{orderIdOne: lockOrderOne, orderIdTwo: lockOrderTwo},
			storeCloseOrders:       map[string]*lib.CloseOrder{},
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdOne},
			expectedCloseOrderIds:  []string{},
		},
		{
			name:                   "duplicate close order exists in store and order book. should not be included",
			orderBookOrders:        []*lib.SellOrder{lockedSellOrderThree},
			storeLockOrders:        map[string]*lib.LockOrder{},
			storeCloseOrders:       map[string]*lib.CloseOrder{orderIdThree: closeOrderThree, orderIdFour: closeOrderFour},
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdThree},
		},
		{
			name:                   "unlocked order exists in order book and matching lock order exists in store. should be included",
			orderBookOrders:        []*lib.SellOrder{unlockedSellOrderFive},
			storeLockOrders:        map[string]*lib.LockOrder{orderIdFive: lockOrderFive},
			storeCloseOrders:       map[string]*lib.CloseOrder{},
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 0,
			expectedLockOrderIds:   []string{orderIdFive},
			expectedCloseOrderIds:  []string{},
		},
		{
			name:                   "locked order exists in order book and matching close order exists in store. should be included",
			orderBookOrders:        []*lib.SellOrder{lockedSellOrderSix},
			storeLockOrders:        map[string]*lib.LockOrder{},
			storeCloseOrders:       map[string]*lib.CloseOrder{orderIdSix: closeOrderSix},
			expectedLockOrdersLen:  0,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{},
			expectedCloseOrderIds:  []string{orderIdSix},
		},
		{
			name:                   "mixed scenario with multiple orders",
			orderBookOrders:        []*lib.SellOrder{unlockedSellOrderSeven, lockedSellOrderEight},
			storeLockOrders:        map[string]*lib.LockOrder{orderIdSeven: createLockOrder(orderIdSeven)},
			storeCloseOrders:       map[string]*lib.CloseOrder{orderIdEight: createCloseOrder(orderIdEight)},
			expectedLockOrdersLen:  1,
			expectedCloseOrdersLen: 1,
			expectedLockOrderIds:   []string{orderIdSeven},
			expectedCloseOrderIds:  []string{orderIdEight},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockOrderStore{
				lockOrders:  map[string][]byte{},
				closeOrders: map[string][]byte{},
			}
			for orderId, lockOrder := range tt.storeLockOrders {
				lockOrderBytes, _ := json.Marshal(lockOrder)
				mockStore.WriteOrder([]byte(orderId), types.LockOrderType, lockOrderBytes)
			}
			for orderId, closeOrder := range tt.storeCloseOrders {
				closeOrderBytes, _ := json.Marshal(closeOrder)
				mockStore.WriteOrder([]byte(orderId), types.CloseOrderType, closeOrderBytes)
			}
			orderBook := createOrderBook(tt.orderBookOrders)
			oracle := &Oracle{
				orderStore: mockStore,
				logger:     lib.NewDefaultLogger(),
			}
			witnessedLockOrders, witnessedCloseOrders := oracle.GatherWitnessedOrders(orderBook)
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

func TestOracle_OrderBookUpdate(t *testing.T) {
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
				lockOrders:  make(map[string][]byte),
				closeOrders: make(map[string][]byte),
			}

			// Populate store with test data
			for _, id := range tt.storedLock {
				mockStore.WriteOrder([]byte(id), types.LockOrderType, []byte("data"))
			}
			for _, id := range tt.storedClose {
				mockStore.WriteOrder([]byte(id), types.CloseOrderType, []byte("data"))
			}

			oracle := &Oracle{
				orderStore: mockStore,
				logger:     lib.NewDefaultLogger(),
			}

			// Execute
			oracle.OrderBookUpdate(newOrderBook(tt.orderBookIds...))

			// Verify
			assertOrderIds(t, "lock", mockStore.GetAllOrderIds(types.LockOrderType), tt.expectedLock)
			assertOrderIds(t, "close", mockStore.GetAllOrderIds(types.CloseOrderType), tt.expectedClose)
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
