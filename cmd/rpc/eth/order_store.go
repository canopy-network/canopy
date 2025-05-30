package eth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
)

const (
	// LockOrderType represents lock order transactions
	LockOrderType = "lock"
	// CloseOrderType represents close order transactions
	CloseOrderType = "close"
)

// EthDiskStorage implements OrderStoreI interface for disk-based storage
type EthDiskStorage struct {
	// storagePath is the directory path where orders are stored
	storagePath string
	// logger is used for logging operations
	logger lib.LoggerI
}

// NewEthDiskStorage creates a new EthDiskStorage instance
func NewEthDiskStorage(storagePath string, logger lib.LoggerI) (*EthDiskStorage, error) {
	// validate storage path is not empty
	if storagePath == "" {
		return nil, fmt.Errorf("storage path cannot be empty")
	}
	// create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	// validate logger is not nil
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	// return new instance
	return &EthDiskStorage{
		storagePath: storagePath,
		logger:      logger,
	}, nil
}

// WriteOrder writes an order to disk with atomic operation
func (e *EthDiskStorage) WriteOrder(orderId string, orderType wstypes.OrderType, blockHeight uint64, txHash string, data []byte) error {
	// validate order id is not empty
	if orderId == "" {
		return fmt.Errorf("order id cannot be empty")
	}
	// validate order type
	if string(orderType) != LockOrderType && string(orderType) != CloseOrderType {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	// validate transaction hash is not empty
	if txHash == "" {
		return fmt.Errorf("transaction hash cannot be empty")
	}
	// validate data is not empty
	if len(data) == 0 {
		return fmt.Errorf("order data cannot be empty")
	}
	// construct filename
	filename := fmt.Sprintf("%s.%s.%d.%s.json", orderId, orderType, blockHeight, txHash)
	// construct full file path
	filePath := filepath.Join(e.storagePath, filename)
	// construct temporary file path for atomic write
	tempPath := filePath + ".tmp"
	// write data to temporary file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	// atomically rename temporary file to final file
	if err := os.Rename(tempPath, filePath); err != nil {
		// cleanup temporary file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}

// ReadOrder reads an order from disk
func (e *EthDiskStorage) ReadOrder(orderId string, orderType wstypes.OrderType) ([]byte, error) {
	// validate order id is not empty
	if orderId == "" {
		return nil, fmt.Errorf("order id cannot be empty")
	}
	// validate order type
	if string(orderType) != LockOrderType && string(orderType) != CloseOrderType {
		return nil, fmt.Errorf("invalid order type: %s", orderType)
	}
	// find matching file in storage directory
	files, err := os.ReadDir(e.storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}
	// search for file matching order id and type
	for _, file := range files {
		// skip directories
		if file.IsDir() {
			continue
		}
		// check if filename matches pattern
		parts := strings.Split(file.Name(), ".")
		if len(parts) >= 2 && parts[0] == orderId && parts[1] == string(orderType) {
			// construct full file path
			filePath := filepath.Join(e.storagePath, file.Name())
			// read file contents
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read order file: %w", err)
			}
			return data, nil
		}
	}
	// return error if order not found
	return nil, fmt.Errorf("order not found: %s.%s", orderId, orderType)
}

// RemoveOrder removes an order from disk
func (e *EthDiskStorage) RemoveOrder(orderId string, orderType wstypes.OrderType) error {
	// validate order id is not empty
	if orderId == "" {
		return fmt.Errorf("order id cannot be empty")
	}
	// validate order type
	if string(orderType) != LockOrderType && string(orderType) != CloseOrderType {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	// find matching file in storage directory
	files, err := os.ReadDir(e.storagePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}
	// search for file matching order id and type
	for _, file := range files {
		// skip directories
		if file.IsDir() {
			continue
		}
		// check if filename matches pattern
		parts := strings.Split(file.Name(), ".")
		if len(parts) >= 2 && parts[0] == orderId && parts[1] == string(orderType) {
			// construct full file path
			filePath := filepath.Join(e.storagePath, file.Name())
			// remove file
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to remove order file: %w", err)
			}
			return nil
		}
	}
	// return error if order not found
	return fmt.Errorf("order not found for removal: %s.%s", orderId, orderType)
}

// GetAllOrderIds gets all order ids present in the store for given order type
func (e *EthDiskStorage) GetAllOrderIds(orderType wstypes.OrderType) [][]byte {
	// validate order type
	if string(orderType) != LockOrderType && string(orderType) != CloseOrderType {
		return nil
	}
	// read storage directory
	files, err := os.ReadDir(e.storagePath)
	if err != nil {
		return nil
	}
	// collect order ids
	var orderIds [][]byte
	// iterate through files
	for _, file := range files {
		// skip directories
		if file.IsDir() {
			continue
		}
		// check if filename matches pattern
		parts := strings.Split(file.Name(), ".")
		if len(parts) >= 2 && parts[1] == string(orderType) {
			// add order id to result
			orderIds = append(orderIds, []byte(parts[0]))
		}
	}
	return orderIds
}
