package eth

import (
	"os"
	"path/filepath"
	"testing"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

// mockLogger implements LoggerI for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string)                          {}
func (m *mockLogger) Info(msg string)                           {}
func (m *mockLogger) Warn(msg string)                           {}
func (m *mockLogger) Error(msg string)                          {}
func (m *mockLogger) Fatal(msg string)                          {}
func (m *mockLogger) Print(msg string)                          {}
func (m *mockLogger) Debugf(format string, args ...interface{}) {}
func (m *mockLogger) Infof(format string, args ...interface{})  {}
func (m *mockLogger) Warnf(format string, args ...interface{})  {}
func (m *mockLogger) Errorf(format string, args ...interface{}) {}
func (m *mockLogger) Fatalf(format string, args ...interface{}) {}
func (m *mockLogger) Printf(format string, args ...interface{}) {}

// TestNewEthDiskStorage tests the constructor for EthDiskStorage
func TestNewEthDiskStorage(t *testing.T) {
	tests := []struct {
		name        string
		storagePath string
		logger      lib.LoggerI
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid parameters",
			storagePath: "test_storage",
			logger:      &mockLogger{},
			wantErr:     false,
		},
		{
			name:        "empty storage path",
			storagePath: "",
			logger:      &mockLogger{},
			wantErr:     true,
			errMsg:      "storage path cannot be empty",
		},
		{
			name:        "nil logger",
			storagePath: "test_storage",
			logger:      nil,
			wantErr:     true,
			errMsg:      "logger cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cleanup test directory if it exists
			if tt.storagePath != "" {
				os.RemoveAll(tt.storagePath)
			}

			// create new storage instance
			storage, err := NewEthDiskStorage(tt.storagePath, tt.logger)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewEthDiskStorage() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewEthDiskStorage() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("NewEthDiskStorage() unexpected error = %v", err)
				return
			}

			// verify storage instance is not nil
			if storage == nil {
				t.Errorf("NewEthDiskStorage() returned nil storage")
			}

			// verify storage path is set correctly
			if storage.storagePath != tt.storagePath {
				t.Errorf("NewEthDiskStorage() storagePath = %v, want %v", storage.storagePath, tt.storagePath)
			}

			// cleanup test directory
			os.RemoveAll(tt.storagePath)
		})
	}
}

// TestEthDiskStorage_WriteOrder tests the WriteOrder method
func TestEthDiskStorage_WriteOrder(t *testing.T) {
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewEthDiskStorage(tempDir, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	tests := []struct {
		name        string
		orderId     string
		orderType   wstypes.OrderType
		blockHeight uint64
		txHash      string
		data        []byte
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid lock order",
			orderId:     "order123",
			orderType:   wstypes.OrderType(LockOrderType),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     false,
		},
		{
			name:        "valid close order",
			orderId:     "order456",
			orderType:   wstypes.OrderType(CloseOrderType),
			blockHeight: 200,
			txHash:      "0xdef456",
			data:        testData,
			wantErr:     false,
		},
		{
			name:        "empty order id",
			orderId:     "",
			orderType:   wstypes.OrderType(LockOrderType),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     true,
			errMsg:      "order id cannot be empty",
		},
		{
			name:        "invalid order type",
			orderId:     "order789",
			orderType:   wstypes.OrderType("invalid"),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     true,
			errMsg:      "invalid order type: invalid",
		},
		{
			name:        "empty transaction hash",
			orderId:     "order789",
			orderType:   wstypes.OrderType(LockOrderType),
			blockHeight: 100,
			txHash:      "",
			data:        testData,
			wantErr:     true,
			errMsg:      "transaction hash cannot be empty",
		},
		{
			name:        "empty data",
			orderId:     "order789",
			orderType:   wstypes.OrderType(LockOrderType),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        []byte{},
			wantErr:     true,
			errMsg:      "order data cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call WriteOrder method
			err := storage.WriteOrder(tt.orderId, tt.orderType, tt.blockHeight, tt.txHash, tt.data)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteOrder() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("WriteOrder() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("WriteOrder() unexpected error = %v", err)
				return
			}

			// verify file was created
			expectedFilename := filepath.Join(tempDir, tt.orderId+"."+string(tt.orderType)+".100."+tt.txHash+".json")
			if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
				t.Errorf("WriteOrder() file was not created: %v", expectedFilename)
			}
		})
	}
}

// TestEthDiskStorage_ReadOrder tests the ReadOrder method
func TestEthDiskStorage_ReadOrder(t *testing.T) {
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewEthDiskStorage(tempDir, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	// write test order first
	err = storage.WriteOrder("order123", wstypes.OrderType(LockOrderType), 100, "0xabc123", testData)
	if err != nil {
		t.Fatalf("failed to write test order: %v", err)
	}

	tests := []struct {
		name      string
		orderId   string
		orderType wstypes.OrderType
		wantData  []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "existing order",
			orderId:   "order123",
			orderType: wstypes.OrderType(LockOrderType),
			wantData:  testData,
			wantErr:   false,
		},
		{
			name:      "non-existing order",
			orderId:   "nonexistent",
			orderType: wstypes.OrderType(LockOrderType),
			wantData:  nil,
			wantErr:   true,
			errMsg:    "order not found: nonexistent.lock",
		},
		{
			name:      "empty order id",
			orderId:   "",
			orderType: wstypes.OrderType(LockOrderType),
			wantData:  nil,
			wantErr:   true,
			errMsg:    "order id cannot be empty",
		},
		{
			name:      "invalid order type",
			orderId:   "order123",
			orderType: wstypes.OrderType("invalid"),
			wantData:  nil,
			wantErr:   true,
			errMsg:    "invalid order type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call ReadOrder method
			data, err := storage.ReadOrder(tt.orderId, tt.orderType)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadOrder() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ReadOrder() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("ReadOrder() unexpected error = %v", err)
				return
			}

			// verify data matches expected
			if string(data) != string(tt.wantData) {
				t.Errorf("ReadOrder() data = %v, want %v", string(data), string(tt.wantData))
			}
		})
	}
}

// TestEthDiskStorage_RemoveOrder tests the RemoveOrder method
func TestEthDiskStorage_RemoveOrder(t *testing.T) {
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewEthDiskStorage(tempDir, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	tests := []struct {
		name       string
		orderId    string
		orderType  wstypes.OrderType
		setupOrder bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "existing order",
			orderId:    "order123",
			orderType:  wstypes.OrderType(LockOrderType),
			setupOrder: true,
			wantErr:    false,
		},
		{
			name:       "non-existing order",
			orderId:    "nonexistent",
			orderType:  wstypes.OrderType(LockOrderType),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "order not found for removal: nonexistent.lock",
		},
		{
			name:       "empty order id",
			orderId:    "",
			orderType:  wstypes.OrderType(LockOrderType),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "order id cannot be empty",
		},
		{
			name:       "invalid order type",
			orderId:    "order123",
			orderType:  wstypes.OrderType("invalid"),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "invalid order type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup order if needed
			if tt.setupOrder {
				err := storage.WriteOrder(tt.orderId, tt.orderType, 100, "0xabc123", testData)
				if err != nil {
					t.Fatalf("failed to setup test order: %v", err)
				}
			}

			// call RemoveOrder method
			err := storage.RemoveOrder(tt.orderId, tt.orderType)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveOrder() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("RemoveOrder() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("RemoveOrder() unexpected error = %v", err)
				return
			}

			// verify file was removed
			_, err = storage.ReadOrder(tt.orderId, tt.orderType)
			if err == nil {
				t.Errorf("RemoveOrder() file still exists after removal")
			}
		})
	}
}

// TestEthDiskStorage_GetAllOrderIds tests the GetAllOrderIds method
func TestEthDiskStorage_GetAllOrderIds(t *testing.T) {
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewEthDiskStorage(tempDir, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	// setup test orders
	lockOrders := []string{"lock1", "lock2", "lock3"}
	closeOrders := []string{"close1", "close2"}

	// write lock orders
	for _, orderId := range lockOrders {
		err := storage.WriteOrder(orderId, wstypes.OrderType(LockOrderType), 100, "0xabc123", testData)
		if err != nil {
			t.Fatalf("failed to write lock order: %v", err)
		}
	}

	// write close orders
	for _, orderId := range closeOrders {
		err := storage.WriteOrder(orderId, wstypes.OrderType(CloseOrderType), 100, "0xdef456", testData)
		if err != nil {
			t.Fatalf("failed to write close order: %v", err)
		}
	}

	tests := []struct {
		name        string
		orderType   wstypes.OrderType
		expectedIds []string
		expectNil   bool
	}{
		{
			name:        "lock orders",
			orderType:   wstypes.OrderType(LockOrderType),
			expectedIds: lockOrders,
			expectNil:   false,
		},
		{
			name:        "close orders",
			orderType:   wstypes.OrderType(CloseOrderType),
			expectedIds: closeOrders,
			expectNil:   false,
		},
		{
			name:        "invalid order type",
			orderType:   wstypes.OrderType("invalid"),
			expectedIds: nil,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call GetAllOrderIds method
			orderIds := storage.GetAllOrderIds(tt.orderType)

			// check if nil expected
			if tt.expectNil {
				if orderIds != nil {
					t.Errorf("GetAllOrderIds() expected nil but got %v", orderIds)
				}
				return
			}

			// verify count matches expected
			if len(orderIds) != len(tt.expectedIds) {
				t.Errorf("GetAllOrderIds() count = %v, want %v", len(orderIds), len(tt.expectedIds))
				return
			}

			// convert byte slices to strings for comparison
			actualIds := make([]string, len(orderIds))
			for i, id := range orderIds {
				actualIds[i] = string(id)
			}

			// verify all expected ids are present
			for _, expectedId := range tt.expectedIds {
				found := false
				for _, actualId := range actualIds {
					if actualId == expectedId {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetAllOrderIds() missing expected id: %v", expectedId)
				}
			}
		})
	}
}
