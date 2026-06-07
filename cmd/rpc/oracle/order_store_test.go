package oracle

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testId        = "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testId2       = "2222222222222222222222222222222222222222"
	testId3       = "3333333333333333333333333333333333333333"
	nonExistingId = "0000000000000000000000000000000000000000"
)

// TestNewOracleDiskStorage tests the constructor for OracleDiskStorage
func TestNewOracleDiskStorage(t *testing.T) {
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
			storage, err := NewOracleDiskStorage(tt.storagePath, lib.NewDefaultLogger())

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOracleDiskStorage() expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewOracleDiskStorage() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("NewOracleDiskStorage() unexpected error = %v", err)
				return
			}

			// verify storage instance is not nil
			if storage == nil {
				t.Errorf("NewOracleDiskStorage() returned nil storage")
			}

			// verify storage path is set correctly
			if storage.storagePath != tt.storagePath {
				t.Errorf("NewOracleDiskStorage() storagePath = %v, want %v", storage.storagePath, tt.storagePath)
			}

			// cleanup test directory
			os.RemoveAll(tt.storagePath)
		})
	}
}

// TestOracleDiskStorage_VerifyOrder tests the VerifyOrder method
func TestOracleDiskStorage_VerifyOrder(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testIdBytes, _ := hex.DecodeString(testId)
	nonExistingId := "0000000000000000000000000000000000000000"
	nonExistingIdBytes, _ := hex.DecodeString(nonExistingId)

	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// create test witnessed order
	testOrder := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  100,
		LastSubmitHeight: 50,
		LockOrder:        &lib.LockOrder{ /* test lock order fields */ },
		CloseOrder:       &lib.CloseOrder{ /* test close order fields */ },
	}

	// create different order for mismatch test
	differentOrder := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  200,
		LastSubmitHeight: 75,
		LockOrder: &lib.LockOrder{
			ChainId: 1,
		},
		CloseOrder: &lib.CloseOrder{ /* different close order fields */ },
	}

	// write test order first
	err = storage.WriteOrder(testOrder, types.LockOrderType)
	if err != nil {
		t.Fatalf("failed to write test order: %v", err)
	}

	tests := []struct {
		name        string
		order       *types.WitnessedOrder
		orderType   types.OrderType
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid order with matching data",
			order:     testOrder,
			orderType: types.LockOrderType,
			wantErr:   false,
		},
		{
			name:        "valid order with mismatched data",
			order:       differentOrder,
			orderType:   types.LockOrderType,
			wantErr:     true,
			errContains: "not equal",
		},
		{
			name: "non-existing order",
			order: &types.WitnessedOrder{
				OrderId:    nonExistingIdBytes,
				LockOrder:  &lib.LockOrder{},
				CloseOrder: &lib.CloseOrder{},
			},
			orderType:   types.LockOrderType,
			wantErr:     true,
			errContains: "failed to read stored order",
		},
		{
			name: "nil order id",
			order: &types.WitnessedOrder{
				OrderId:    nil,
				LockOrder:  &lib.LockOrder{},
				CloseOrder: &lib.CloseOrder{},
			},
			orderType:   types.LockOrderType,
			wantErr:     true,
			errContains: "cannot be nil",
		},
		{
			name:        "invalid order type",
			order:       testOrder,
			orderType:   types.OrderType("invalid"),
			wantErr:     true,
			errContains: "invalid order type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call VerifyOrder method
			err := storage.VerifyOrder(tt.order, tt.orderType)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyOrder() expected error but got none")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
						t.Errorf("VerifyOrder() error = %v, want error containing %v", err.Error(), tt.errContains)
					}
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

// TestOracleDiskStorage_WriteOrder tests the WriteOrder method
func TestOracleDiskStorage_WriteOrder(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testId2 := "2222222222222222222222222222222222222222"
	testId3 := "3333333333333333333333333333333333333333"
	testIdBytes, _ := hex.DecodeString(testId)
	testId2Bytes, _ := hex.DecodeString(testId2)
	testId3Bytes, _ := hex.DecodeString(testId3)
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name        string
		order       *types.WitnessedOrder
		orderType   types.OrderType
		wantErr     bool
		errContains string
	}{
		{
			name: "valid lock order",
			order: &types.WitnessedOrder{
				OrderId:          testIdBytes,
				WitnessedHeight:  100,
				LastSubmitHeight: 50,
				LockOrder:        &lib.LockOrder{},
				CloseOrder:       &lib.CloseOrder{},
			},
			orderType: types.LockOrderType,
			wantErr:   false,
		},
		{
			name: "valid close order",
			order: &types.WitnessedOrder{
				OrderId:          testId2Bytes,
				WitnessedHeight:  200,
				LastSubmitHeight: 150,
				LockOrder:        &lib.LockOrder{},
				CloseOrder:       &lib.CloseOrder{},
			},
			orderType: types.CloseOrderType,
			wantErr:   false,
		},
		{
			name: "empty order id",
			order: &types.WitnessedOrder{
				OrderId:          []byte{},
				WitnessedHeight:  100,
				LastSubmitHeight: 50,
				LockOrder:        &lib.LockOrder{},
				CloseOrder:       &lib.CloseOrder{},
			},
			orderType:   types.LockOrderType,
			wantErr:     true,
			errContains: "order id invalid length",
		},
		{
			name: "invalid order type",
			order: &types.WitnessedOrder{
				OrderId:          testId3Bytes,
				WitnessedHeight:  100,
				LastSubmitHeight: 50,
				LockOrder:        &lib.LockOrder{},
				CloseOrder:       &lib.CloseOrder{},
			},
			orderType:   types.OrderType("invalid"),
			wantErr:     true,
			errContains: "invalid order type: invalid",
		},
		{
			name: "duplicate order",
			order: &types.WitnessedOrder{
				OrderId:          testIdBytes,
				WitnessedHeight:  300,
				LastSubmitHeight: 250,
				LockOrder:        &lib.LockOrder{},
				CloseOrder:       &lib.CloseOrder{},
			},
			orderType: types.LockOrderType,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call WriteOrder method
			err := storage.WriteOrder(tt.order, tt.orderType)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteOrder() expected error but got none")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
						t.Errorf("WriteOrder() error = %v, want error containing %v", err.Error(), tt.errContains)
					}
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("WriteOrder() unexpected error = %v", err)
				return
			}

			// verify file was created
			expectedFilename, _ := storage.buildFilePath(tt.order.OrderId, tt.orderType)
			if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
				t.Errorf("WriteOrder() file was not created: %v", expectedFilename)
			}
			o, err := storage.ReadOrder(tt.order.OrderId, tt.orderType)
			if err != nil {
				t.Errorf("ReadOrder() unexpected error = %v", err)
			}
			// Compare the retrieved order with the expected order
			if diff := cmp.Diff(tt.order, o,
				cmpopts.IgnoreUnexported(lib.LockOrder{}),
				cmpopts.IgnoreUnexported(lib.CloseOrder{}),
			); diff != "" {
				t.Errorf("ReadOrder() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOracleDiskStorage_ReadOrder(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testIdBytes, _ := hex.DecodeString(testId)
	nonExistingId := "0000000000000000000000000000000000000000"
	nonExistingIdBytes, _ := hex.DecodeString(nonExistingId)

	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// create test witnessed order
	testOrder := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  12345,
		LastSubmitHeight: 67890,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}

	// write test order first
	err = storage.WriteOrder(testOrder, types.LockOrderType)
	if err != nil {
		t.Fatalf("failed to write test order: %v", err)
	}

	tests := []struct {
		name        string
		orderId     []byte
		orderType   types.OrderType
		wantOrder   *types.WitnessedOrder
		wantErr     bool
		errContains string
	}{
		{
			name:      "existing order",
			orderId:   testIdBytes,
			orderType: types.LockOrderType,
			wantOrder: testOrder,
			wantErr:   false,
		},
		{
			name:        "non-existing order",
			orderId:     nonExistingIdBytes,
			orderType:   types.LockOrderType,
			wantOrder:   nil,
			wantErr:     true,
			errContains: "no such file or directory",
		},
		{
			name:        "empty order id",
			orderId:     []byte{},
			orderType:   types.LockOrderType,
			wantOrder:   nil,
			wantErr:     true,
			errContains: "order id invalid length",
		},
		{
			name:        "invalid order type",
			orderId:     testIdBytes,
			orderType:   types.OrderType("invalid"),
			wantOrder:   nil,
			wantErr:     true,
			errContains: "invalid order type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// call ReadOrder method
			order, err := storage.ReadOrder(tt.orderId, tt.orderType)

			// check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadOrder() expected error but got none")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
						t.Errorf("ReadOrder() error = %v, want error containing %v", err.Error(), tt.errContains)
					}
				}
				return
			}

			// check no error expected
			if err != nil {
				t.Errorf("ReadOrder() unexpected error = %v", err)
				return
			}

			// verify order matches expected
			if order == nil {
				t.Errorf("ReadOrder() returned nil order")
				return
			}

			if string(order.OrderId) != string(tt.wantOrder.OrderId) {
				t.Errorf("ReadOrder() OrderId = %v, want %v", hex.EncodeToString(order.OrderId), hex.EncodeToString(tt.wantOrder.OrderId))
			}

			if order.WitnessedHeight != tt.wantOrder.WitnessedHeight {
				t.Errorf("ReadOrder() WitnessedHeight = %v, want %v", order.WitnessedHeight, tt.wantOrder.WitnessedHeight)
			}

			if order.LastSubmitHeight != tt.wantOrder.LastSubmitHeight {
				t.Errorf("ReadOrder() LastSubmitHeight = %v, want %v", order.LastSubmitHeight, tt.wantOrder.LastSubmitHeight)
			}
		})
	}
}

// TestOracleDiskStorage_RemoveOrder tests the RemoveOrder method
func TestOracleDiskStorage_RemoveOrder(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testIdBytes, _ := hex.DecodeString(testId)
	nonExistingId := "0000000000000000000000000000000000000000"
	nonExistingIdBytes, _ := hex.DecodeString(nonExistingId)
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// create test witnessed order
	testOrder := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  100,
		LastSubmitHeight: 50,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}

	tests := []struct {
		name        string
		orderId     []byte
		orderType   types.OrderType
		setupOrder  bool
		wantErr     bool
		errContains string
	}{
		{
			name:       "existing order",
			orderId:    testIdBytes,
			orderType:  types.OrderType(types.LockOrderType),
			setupOrder: true,
			wantErr:    false,
		},
		{
			name:        "non-existing order",
			orderId:     nonExistingIdBytes,
			orderType:   types.OrderType(types.LockOrderType),
			setupOrder:  false,
			wantErr:     true,
			errContains: "no such file or directory",
		},
		{
			name:        "empty order id",
			orderId:     []byte{},
			orderType:   types.LockOrderType,
			setupOrder:  false,
			wantErr:     true,
			errContains: "order id invalid length",
		},
		{
			name:        "invalid order type",
			orderId:     testIdBytes,
			orderType:   types.OrderType("invalid"),
			setupOrder:  false,
			wantErr:     true,
			errContains: "invalid order type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup order if needed
			if tt.setupOrder {
				err := storage.WriteOrder(testOrder, tt.orderType)
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
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
						t.Errorf("RemoveOrder() error = %v, want error containing %v", err.Error(), tt.errContains)
					}
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

// TestOracleDiskStorage_GetAllOrderIds tests the GetAllOrderIds method
func TestOracleDiskStorage_GetAllOrderIds(t *testing.T) {
	testId := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b"
	testId2 := "2222222222222222222222222222222222222222"
	testId3 := "3333333333333333333333333333333333333333"
	testIdBytes, _ := hex.DecodeString(testId)
	testId2Bytes, _ := hex.DecodeString(testId2)
	testId3Bytes, _ := hex.DecodeString(testId3)
	// create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "eth_storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// create test witnessed orders
	lockOrder1 := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  100,
		LastSubmitHeight: 50,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}
	lockOrder2 := &types.WitnessedOrder{
		OrderId:          testId2Bytes,
		WitnessedHeight:  200,
		LastSubmitHeight: 150,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}
	lockOrder3 := &types.WitnessedOrder{
		OrderId:          testId3Bytes,
		WitnessedHeight:  300,
		LastSubmitHeight: 250,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}
	closeOrder1 := &types.WitnessedOrder{
		OrderId:          testIdBytes,
		WitnessedHeight:  100,
		LastSubmitHeight: 50,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}
	closeOrder2 := &types.WitnessedOrder{
		OrderId:          testId2Bytes,
		WitnessedHeight:  200,
		LastSubmitHeight: 150,
		LockOrder:        &lib.LockOrder{},
		CloseOrder:       &lib.CloseOrder{},
	}

	// setup test orders
	lockOrders := []*types.WitnessedOrder{lockOrder1, lockOrder2, lockOrder3}
	lockOrderIds := [][]byte{testIdBytes, testId2Bytes, testId3Bytes}
	closeOrders := []*types.WitnessedOrder{closeOrder1, closeOrder2}
	closeOrderIds := [][]byte{testIdBytes, testId2Bytes}

	// write lock orders
	for _, order := range lockOrders {
		err := storage.WriteOrder(order, types.LockOrderType)
		if err != nil {
			t.Fatalf("failed to write lock order: %v", err)
		}
	}

	// write close orders
	for _, order := range closeOrders {
		err := storage.WriteOrder(order, types.CloseOrderType)
		if err != nil {
			t.Fatalf("failed to write close order: %v", err)
		}
	}

	tests := []struct {
		name        string
		orderType   types.OrderType
		expectedIds [][]byte
		expectNil   bool
	}{
		{
			name:        "lock orders",
			orderType:   types.OrderType(types.LockOrderType),
			expectedIds: lockOrderIds,
			expectNil:   false,
		},
		{
			name:        "close orders",
			orderType:   types.OrderType(types.CloseOrderType),
			expectedIds: closeOrderIds,
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
			orderIds, _ := storage.GetAllOrderIds(tt.orderType)
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
			// verify all expected ids are present
			for _, expectedId := range tt.expectedIds {
				found := false
				for _, actualId := range orderIds {
					if bytes.Equal(actualId, expectedId) {
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

func TestOracleDiskStorage_GetAllOrderIds_UsesAbsolutePathAfterCwdChange(t *testing.T) {
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	root := t.TempDir()
	baseDir := filepath.Join(root, "base")
	otherDir := filepath.Join(root, "other")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))
	require.NoError(t, os.MkdirAll(otherDir, 0o755))
	require.NoError(t, os.Chdir(baseDir))

	storage, err := NewOracleDiskStorage("relative-orders", lib.NewDefaultLogger())
	require.NoError(t, err)

	orderID, err := hex.DecodeString(testId)
	require.NoError(t, err)
	order := &types.WitnessedOrder{
		OrderId:         orderID,
		WitnessedHeight: 1,
		LockOrder:       &lib.LockOrder{OrderId: orderID},
	}
	require.NoError(t, storage.WriteOrder(order, types.LockOrderType))

	ids, getErr := storage.GetAllOrderIds(types.LockOrderType)
	require.NoError(t, getErr)
	require.Len(t, ids, 1)

	require.NoError(t, os.Chdir(otherDir))

	ids, getErr = storage.GetAllOrderIds(types.LockOrderType)
	require.NoError(t, getErr)
	require.Len(t, ids, 1)

	readOrder, readErr := storage.ReadOrder(orderID, types.LockOrderType)
	require.NoError(t, readErr)
	require.NotNil(t, readOrder)
}

func TestOracleDiskStorage_ArchiveOrder(t *testing.T) {
	tempDir := t.TempDir()
	logger := lib.NewDefaultLogger()
	storage, err := NewOracleDiskStorage(tempDir, logger)
	require.NoError(t, err)

	orderIdHex := "53ecc91b68aba0e82ba09fbf205e4f81cc44b92b" // Use hex string
	orderId, _ := hex.DecodeString(orderIdHex)
	order := &types.WitnessedOrder{
		OrderId:         orderId,
		WitnessedHeight: 100,
		LockOrder: &lib.LockOrder{
			OrderId:             orderId,
			BuyerReceiveAddress: []byte("test_buyer_address"),
		},
	}

	tests := []struct {
		name      string
		order     *types.WitnessedOrder
		orderType types.OrderType
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid lock order archive",
			order:     order,
			orderType: types.LockOrderType,
			wantErr:   false,
		},
		{
			name: "valid close order archive",
			order: &types.WitnessedOrder{
				OrderId:         orderId,
				WitnessedHeight: 200,
				CloseOrder: &lib.CloseOrder{
					OrderId:    orderId,
					ChainId:    1,
					CloseOrder: true,
				},
			},
			orderType: types.CloseOrderType,
			wantErr:   false,
		},
		{
			name: "nil order id",
			order: &types.WitnessedOrder{
				OrderId: nil,
			},
			orderType: types.LockOrderType,
			wantErr:   true,
			errMsg:    "order id cannot be nil",
		},
		{
			name:      "invalid order type",
			order:     order,
			orderType: types.OrderType("invalid"),
			wantErr:   true,
			errMsg:    "invalid order type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.ArchiveOrder(tt.order, tt.orderType)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)

				// Verify archive file exists in correct location
				var expectedDir string
				switch tt.orderType {
				case types.LockOrderType:
					expectedDir = "archive/lock"
				case types.CloseOrderType:
					expectedDir = "archive/close"
				}

				filename := fmt.Sprintf("%s.%s.json", hex.EncodeToString(tt.order.OrderId), string(tt.orderType))
				archivePath := filepath.Join(tempDir, expectedDir, filename)

				// Check that archive file exists
				_, err := os.Stat(archivePath)
				assert.NoError(t, err, "archive file should exist")

				// Verify archive file contents
				data, err := os.ReadFile(archivePath)
				require.NoError(t, err)

				var archivedOrder types.WitnessedOrder
				err = json.Unmarshal(data, &archivedOrder)
				require.NoError(t, err)

				assert.Equal(t, tt.order.OrderId, archivedOrder.OrderId)
				assert.Equal(t, tt.order.WitnessedHeight, archivedOrder.WitnessedHeight)
			}
		})
	}
}
