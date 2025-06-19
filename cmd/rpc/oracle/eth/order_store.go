package eth

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

const (
	// tempSuffix is the suffix used for temporary files during atomic writes
	tempSuffix = ".tmp"
	// jsonExtension is the file extension for JSON files
	jsonExtension = ".json"
	// order id length
	orderIdLength = 40
)

// EthDiskStorage implements OrderStoreI interface for Ethereum order storage
type EthDiskStorage struct {
	// storagePath is the directory path where orders are stored
	storagePath string
	// logger is used for logging operations
	logger lib.LoggerI
	// mutex to protect concurrent access
	rwLock sync.RWMutex
}

// NewEthDiskStorage creates a new EthDiskStorage instance
func NewEthDiskStorage(storagePath string, logger lib.LoggerI) (*EthDiskStorage, error) {
	// validate storage path is not empty
	if storagePath == "" {
		return nil, fmt.Errorf("storage path cannot be empty")
	}
	// validate logger is not nil
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	// create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	// return new instance
	return &EthDiskStorage{
		storagePath: storagePath,
		logger:      logger,
		rwLock:      sync.RWMutex{},
	}, nil
}

// VerifyOrder verifies the order with order id is present in the store by comparing the byte data
func (e *EthDiskStorage) VerifyOrder(orderId []byte, orderType types.OrderType, data []byte) error {
	// validate order id
	if err := e.validateOrderId(orderId); err != nil {
		return err
	}
	// validate order type
	if err := e.validateOrderType(orderType); err != nil {
		return err
	}
	// validate data is not nil
	if data == nil {
		return fmt.Errorf("order data cannot be nil")
	}
	// validate data is not empty
	if len(data) == 0 {
		return fmt.Errorf("order data cannot be empty")
	}
	// read the stored order data
	storedData, err := e.ReadOrder(orderId, orderType)
	if err != nil {
		return fmt.Errorf("failed to read stored order: %w", err)
	}
	// compare the provided data with stored data
	if !bytes.Equal(data, storedData) {
		return fmt.Errorf("order data mismatch: content differs")
	}
	return nil
}

// WriteOrder writes an order to disk with atomic write operation
func (e *EthDiskStorage) WriteOrder(orderId []byte, orderType types.OrderType, data []byte) error {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	// validate order id
	if err := e.validateOrderId(orderId); err != nil {
		return err
	}
	// validate order type
	if err := e.validateOrderType(orderType); err != nil {
		return err
	}
	// validate data is not empty
	if len(data) == 0 {
		return fmt.Errorf("order data cannot be empty")
	}
	// build file path
	filePath, err := e.buildFilePath(orderId, orderType)
	if err != nil {
		return err
	}
	// check if file already exists to prevent duplicates
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("order with id %s and type %s already exists", lib.BytesToString(orderId), orderType)
	}
	// create temporary file for atomic write
	tempPath := filePath + tempSuffix
	// write data to temporary file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	// atomically rename temporary file to final filename
	if err := os.Rename(tempPath, filePath); err != nil {
		// cleanup temporary file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}

// ReadOrder reads an order from disk
func (e *EthDiskStorage) ReadOrder(orderId []byte, orderType types.OrderType) ([]byte, error) {
	e.rwLock.RLock()
	defer e.rwLock.RUnlock()
	// validate order id
	if err := e.validateOrderId(orderId); err != nil {
		return nil, err
	}
	// validate order type
	if err := e.validateOrderType(orderType); err != nil {
		return nil, err
	}
	// build file path
	filePath, err := e.buildFilePath(orderId, orderType)
	if err != nil {
		return nil, err
	}
	// read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RemoveOrder removes an order from disk
func (e *EthDiskStorage) RemoveOrder(orderId []byte, orderType types.OrderType) error {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	// validate order id
	if err := e.validateOrderId(orderId); err != nil {
		return err
	}
	// validate order type
	if err := e.validateOrderType(orderType); err != nil {
		return err
	}
	// build file path
	filePath, err := e.buildFilePath(orderId, orderType)
	if err != nil {
		return err
	}
	// remove the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove order file: %w", err)
	}
	return nil
}

// GetAllOrderIds gets all order ids present in the store for a specific order type
func (e *EthDiskStorage) GetAllOrderIds(orderType types.OrderType) [][]byte {
	e.rwLock.RLock()
	defer e.rwLock.RUnlock()
	// validate order type
	if err := e.validateOrderType(orderType); err != nil {
		return nil
	}
	// read directory contents
	entries, err := os.ReadDir(e.storagePath)
	if err != nil {
		return nil
	}
	// collect order ids for the specified type
	var orderIds [][]byte
	orderTypeSuffix := fmt.Sprintf(".%s%s", string(orderType), jsonExtension)
	// iterate through directory entries
	for _, entry := range entries {
		// skip directories
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		// check if filename matches the order type pattern
		if strings.HasSuffix(filename, orderTypeSuffix) {
			// extract order id from filename
			orderId := strings.TrimSuffix(filename, orderTypeSuffix)
			orderIds = append(orderIds, []byte(orderId))
		}
	}
	return orderIds
}

// validateOrderId validates the order id
func (e *EthDiskStorage) validateOrderId(orderId []byte) error {
	if orderId == nil {
		return errors.New("order id cannot be nil")
	}
	if len(orderId) != orderIdLength {
		return errors.New("order id invalid length")
	}
	return nil
}

// validateOrderType validates the order type
func (e *EthDiskStorage) validateOrderType(orderType types.OrderType) error {
	if orderType != types.LockOrderType && orderType != types.CloseOrderType {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	return nil
}

// buildFilePath builds a file path for an order JSON file
func (e *EthDiskStorage) buildFilePath(orderId []byte, orderType types.OrderType) (string, error) {
	filename := fmt.Sprintf("%s.%s%s", string(orderId), string(orderType), jsonExtension)
	filePath := filepath.Join(e.storagePath, filename)

	// Ensure the path is within the storage directory
	if !strings.HasPrefix(filePath, e.storagePath) {
		return "", fmt.Errorf("invalid file path")
	}
	return filePath, nil
}
