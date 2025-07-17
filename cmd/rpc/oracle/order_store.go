package oracle

import (
	"encoding/hex"
	"encoding/json"
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
)

// OracleDiskStorage implements OrderStore interface for Ethereum order storage
type OracleDiskStorage struct {
	// storagePath is the directory path where orders are stored
	storagePath string
	// logger is used for logging operations
	logger lib.LoggerI
	// mutex to protect concurrent access
	rwLock sync.RWMutex
}

// NewOracleDiskStorage creates a new OracleDiskStorage instance
func NewOracleDiskStorage(storagePath string, logger lib.LoggerI) (*OracleDiskStorage, error) {
	// validate storage path is not empty
	if storagePath == "" {
		return nil, fmt.Errorf("storage path cannot be empty")
	}
	// validate logger is not nil
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if strings.HasPrefix(storagePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		storagePath = filepath.Join(home, storagePath[2:])
	}

	// create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	// return new instance
	return &OracleDiskStorage{
		storagePath: storagePath,
		logger:      logger,
		rwLock:      sync.RWMutex{},
	}, nil
}

// VerifyOrder verifies the order with order id is present in the store
// this verifies the lock order or close order fields of the witnessed order, ignoring the other fields
func (e *OracleDiskStorage) VerifyOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	// validate parameters
	if err := e.validateOrderParameters(order.OrderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}
	// read the stored order
	storedOrder, err := e.ReadOrder(order.OrderId, orderType)
	if err != nil {
		return ErrVerifyOrder(fmt.Errorf("failed to read stored order: %w", err))
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

// WriteOrder writes an order to disk with atomic write operation
func (e *OracleDiskStorage) WriteOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	// validate parameters
	if err := e.validateOrderParameters(order.OrderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}
	// build file path
	bz, err := json.Marshal(order)
	if err != nil {
		return ErrMarshalOrder(err)
	}
	filePath, err := e.buildFilePath(order.OrderId, orderType)
	if err != nil {
		return ErrWriteOrder(err)
	}
	// create temporary file for atomic write
	tempPath := filePath + tempSuffix
	// write data to temporary file
	if err := os.WriteFile(tempPath, bz, 0644); err != nil {
		return ErrWriteOrder(fmt.Errorf("failed to write temporary file: %w", err))
	}
	// atomically rename temporary file to final filename
	if err := os.Rename(tempPath, filePath); err != nil {
		// cleanup temporary file on failure
		os.Remove(tempPath)
		return ErrWriteOrder(fmt.Errorf("failed to rename temporary file: %w", err))
	}
	return nil
}

// ReadOrder reads an order from disk
func (e *OracleDiskStorage) ReadOrder(orderId []byte, orderType types.OrderType) (*types.WitnessedOrder, lib.ErrorI) {
	e.rwLock.RLock()
	defer e.rwLock.RUnlock()
	// validate parameters
	if err := e.validateOrderParameters(orderId, orderType); err != nil {
		return nil, ErrValidateOrder(err)
	}
	// build file path
	filePath, err := e.buildFilePath(orderId, orderType)
	if err != nil {
		return nil, ErrReadOrder(err)
	}
	// read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrReadOrder(err)
	}
	// unmarshal the order
	order := &types.WitnessedOrder{}
	err = json.Unmarshal(data, order)
	return order, nil
}

// RemoveOrder removes an order from disk
func (e *OracleDiskStorage) RemoveOrder(orderId []byte, orderType types.OrderType) lib.ErrorI {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	// validate parameters
	if err := e.validateOrderParameters(orderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}
	// build file path
	filePath, err := e.buildFilePath(orderId, orderType)
	if err != nil {
		return ErrRemoveOrder(err)
	}
	// remove the file
	if err := os.Remove(filePath); err != nil {
		return ErrRemoveOrder(err)
	}
	return nil
}

// GetAllOrderIds gets all order ids present in the store for a specific order type
func (e *OracleDiskStorage) GetAllOrderIds(orderType types.OrderType) ([][]byte, lib.ErrorI) {
	e.rwLock.RLock()
	defer e.rwLock.RUnlock()
	// validate order type
	if orderType != types.LockOrderType && orderType != types.CloseOrderType {
		return nil, ErrVerifyOrder(fmt.Errorf("invalid order type: %s", orderType))
	}
	// read directory contents
	entries, err := os.ReadDir(e.storagePath)
	if err != nil {
		return nil, ErrReadOrder(err)
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
			id, err := hex.DecodeString(orderId)
			if err != nil {
				e.logger.Errorf("Failed to decode order id in filename: %s", err.Error())
				continue
			}
			orderIds = append(orderIds, id)
		}
	}
	return orderIds, nil
}

func (e *OracleDiskStorage) validateOrderParameters(orderId []byte, orderType types.OrderType) error {
	// orderId cannot be nil
	if orderId == nil {
		return errors.New("order id cannot be nil")
	}
	if len(orderId) == 0 {
		return errors.New("order id invalid length")
	}
	// validate order type
	if orderType != types.LockOrderType && orderType != types.CloseOrderType {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	return nil
}

// buildFilePath builds a file path for an order JSON file
func (e *OracleDiskStorage) buildFilePath(orderId []byte, orderType types.OrderType) (string, error) {
	filename := fmt.Sprintf("%s.%s%s", hex.EncodeToString(orderId), string(orderType), jsonExtension)
	filePath := filepath.Join(e.storagePath, filename)

	// Ensure the path is within the storage directory
	if !strings.HasPrefix(filePath, e.storagePath) {
		return "", fmt.Errorf("invalid file path")
	}
	return filePath, nil
}
