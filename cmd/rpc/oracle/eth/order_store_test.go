package eth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

const (
	testId        = "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testId2       = "2222222222222222222222222222222222222222"
	testId3       = "3333333333333333333333333333333333333333"
	nonExistingId = "0000000000000000000000000000000000000000"
)

// TestNewEthDiskStorage tests the constructor for EthDiskStorage
func TestNewEthDiskStorage(t *testing.T) {
	tests := []struct {
		name        string
		storagePath string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid parameters",
			storagePath: "test_storage",
			wantErr:     false,
		},
		{
			name:        "empty storage path",
			storagePath: "",
			wantErr:     true,
			errMsg:      "storage path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cleanup test directory if it exists
			if tt.storagePath != "" {
				os.RemoveAll(tt.storagePath)
			}

			// create new storage instance
			storage, err := NewEthDiskStorage(tt.storagePath, lib.NewDefaultLogger())

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

// TestEthDiskStorage_VerifyOrder tests the VerifyOrder method
func TestEthDiskStorage_VerifyOrder(t *testing.T) {
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewEthDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)
	differentData := []byte(`{"different": "data"}`)

	// write test order first
	err = storage.WriteOrder([]byte(testId), types.OrderType(types.LockOrderType), testData)
	if err != nil {
		t.Fatalf("failed to write test order: %v", err)
	}

	tests := []struct {
		name      string
		orderId   []byte
		orderType types.OrderType
		data      []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid order with matching data",
			orderId:   []byte(testId),
			orderType: types.OrderType(types.LockOrderType),
			data:      testData,
			wantErr:   false,
		},
		{
			name:      "valid order with mismatched data",
			orderId:   []byte(testId),
			orderType: types.OrderType(types.LockOrderType),
			data:      differentData,
			wantErr:   true,
			errMsg:    "order data mismatch: content differs",
		},
		{
			name:      "non-existing order",
			orderId:   []byte(nonExistingId),
			orderType: types.OrderType(types.LockOrderType),
			data:      testData,
			wantErr:   true,
			errMsg:    "failed to read stored order: open " + filepath.Join(tempDir, nonExistingId+".lock.json") + ": no such file or directory",
		},
		{
			name:      "empty order id",
			orderId:   []byte{},
			orderType: types.OrderType(types.LockOrderType),
			data:      testData,
			wantErr:   true,
			errMsg:    "order id invalid length",
		},
		{
			name:      "invalid order type",
			orderId:   []byte(testId),
			orderType: types.OrderType("invalid"),
			data:      testData,
			wantErr:   true,
			errMsg:    "invalid order type: invalid",
		},
		{
			name:      "empty data",
			orderId:   []byte(testId),
			orderType: types.OrderType(types.LockOrderType),
			data:      []byte{},
			wantErr:   true,
			errMsg:    "order data cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call VerifyOrder method
			err := storage.VerifyOrder(tt.orderId, tt.orderType, tt.data)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyOrder() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("VerifyOrder() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("VerifyOrder() unexpected error = %v", err)
				return
			}
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
	storage, err := NewEthDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	tests := []struct {
		name        string
		orderId     []byte
		orderType   types.OrderType
		blockHeight uint64
		txHash      string
		data        []byte
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid lock order",
			orderId:     []byte(testId),
			orderType:   types.OrderType(types.LockOrderType),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     false,
		},
		{
			name:        "valid close order",
			orderId:     []byte(testId2),
			orderType:   types.OrderType(types.CloseOrderType),
			blockHeight: 200,
			txHash:      "0xdef456",
			data:        testData,
			wantErr:     false,
		},
		{
			name:        "empty order id",
			orderId:     []byte{},
			orderType:   types.OrderType(types.LockOrderType),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     true,
			errMsg:      "order id invalid length",
		},
		{
			name:        "invalid order type",
			orderId:     []byte(testId3),
			orderType:   types.OrderType("invalid"),
			blockHeight: 100,
			txHash:      "0xabc123",
			data:        testData,
			wantErr:     true,
			errMsg:      "invalid order type: invalid",
		},
		{
			name:        "empty data",
			orderId:     []byte(testId3),
			orderType:   types.OrderType(types.LockOrderType),
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
			err := storage.WriteOrder(tt.orderId, tt.orderType, tt.data)

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
			expectedFilename := filepath.Join(tempDir, string(tt.orderId)+"."+string(tt.orderType)+".json")
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
	storage, err := NewEthDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	// write test order first
	err = storage.WriteOrder([]byte(testId), types.OrderType(types.LockOrderType), testData)
	if err != nil {
		t.Fatalf("failed to write test order: %v", err)
	}

	tests := []struct {
		name      string
		orderId   []byte
		orderType types.OrderType
		wantData  []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "existing order",
			orderId:   []byte(testId),
			orderType: types.OrderType(types.LockOrderType),
			wantData:  testData,
			wantErr:   false,
		},
		{
			name:      "non-existing order",
			orderId:   []byte(nonExistingId),
			orderType: types.OrderType(types.LockOrderType),
			wantData:  nil,
			wantErr:   true,

			errMsg: "open " + filepath.Join(tempDir, nonExistingId+".lock.json") + ": no such file or directory",
		},
		{
			name:      "empty order id",
			orderId:   []byte{},
			orderType: types.OrderType(types.LockOrderType),
			wantData:  nil,
			wantErr:   true,
			errMsg:    "order id invalid length",
		},
		{
			name:      "invalid order type",
			orderId:   []byte(testId),
			orderType: types.OrderType("invalid"),
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
	storage, err := NewEthDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	tests := []struct {
		name       string
		orderId    []byte
		orderType  types.OrderType
		setupOrder bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "existing order",
			orderId:    []byte(testId),
			orderType:  types.OrderType(types.LockOrderType),
			setupOrder: true,
			wantErr:    false,
		},
		{
			name:       "non-existing order",
			orderId:    []byte(nonExistingId),
			orderType:  types.OrderType(types.LockOrderType),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "failed to remove order file: remove " + filepath.Join(tempDir, nonExistingId) + ".lock.json" + ": no such file or directory",
		},
		{
			name:       "empty order id",
			orderId:    []byte{},
			orderType:  types.OrderType(types.LockOrderType),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "order id invalid length",
		},
		{
			name:       "invalid order type",
			orderId:    []byte(testId),
			orderType:  types.OrderType("invalid"),
			setupOrder: false,
			wantErr:    true,
			errMsg:     "invalid order type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup order if needed
			if tt.setupOrder {
				err := storage.WriteOrder(tt.orderId, tt.orderType, testData)
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
	storage, err := NewEthDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// test data
	testData := []byte(`{"test": "data"}`)

	// setup test orders
	lockOrders := []string{testId, testId2, testId3}
	closeOrders := []string{testId, testId2}

	// write lock orders
	for _, orderId := range lockOrders {
		err := storage.WriteOrder([]byte(orderId), types.OrderType(types.LockOrderType), testData)
		if err != nil {
			t.Fatalf("failed to write lock order: %v", err)
		}
	}

	// write close orders
	for _, orderId := range closeOrders {
		err := storage.WriteOrder([]byte(orderId), types.OrderType(types.CloseOrderType), testData)
		if err != nil {
			t.Fatalf("failed to write close order: %v", err)
		}
	}

	tests := []struct {
		name        string
		orderType   types.OrderType
		expectedIds []string
		expectNil   bool
	}{
		{
			name:        "lock orders",
			orderType:   types.OrderType(types.LockOrderType),
			expectedIds: lockOrders,
			expectNil:   false,
		},
		{
			name:        "close orders",
			orderType:   types.OrderType(types.CloseOrderType),
			expectedIds: closeOrders,
			expectNil:   false,
		},
		{
			name:        "invalid order type",
			orderType:   types.OrderType("invalid"),
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
