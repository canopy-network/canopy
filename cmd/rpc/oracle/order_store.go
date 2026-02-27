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
	// storagePath is the directory path for order storage
	storagePath string
	// absStoragePath is the absolute path used for validation
	absStoragePath string
	// absArchivePath is the absolute archive path used for validation
	absArchivePath string
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

	// get absolute path for secure validation
	absStoragePath, err := filepath.Abs(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute storage path: %w", err)
	}
	// clean the path to resolve any .. or . elements
	absStoragePath = filepath.Clean(absStoragePath)

	// create storage directory if it doesn't exist
	if err := os.MkdirAll(absStoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	// create archive directory structure
	archiveDir := filepath.Join(absStoragePath, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}
	// get absolute archive path for validation
	absArchivePath, err := filepath.Abs(archiveDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute archive path: %w", err)
	}
	absArchivePath = filepath.Clean(absArchivePath)
	// create lock and close subdirectories in archive
	lockArchiveDir := filepath.Join(archiveDir, "lock")
	if err := os.MkdirAll(lockArchiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock archive directory: %w", err)
	}
	closeArchiveDir := filepath.Join(archiveDir, "close")
	if err := os.MkdirAll(closeArchiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create close archive directory: %w", err)
	}
	// return new instance
	return &OracleDiskStorage{
		storagePath:    storagePath,
		absStoragePath: absStoragePath,
		absArchivePath: absArchivePath,
		logger:         logger,
		rwLock:         sync.RWMutex{},
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
	// e.logger.Debugf("OrderStore: Wrote %d bytes to %s", len(bz), filePath)
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
	// e.logger.Debugf("OrderStore: Attempting to read %s", filePath)
	// read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrReadOrder(err)
	}
	// e.logger.Debugf("OrderStore: Read %d bytes from %s", len(data), filePath)
	// unmarshal the order
	order := &types.WitnessedOrder{}
	err = json.Unmarshal(data, order)
	if err != nil {
		return nil, ErrUnmarshalOrder(err)
	}
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
	// e.logger.Debugf("OrderStore: Removed %s", filePath)
	return nil
}

// ArchiveOrder archives a witnessed order to the archive directory for historical retention
func (e *OracleDiskStorage) ArchiveOrder(order *types.WitnessedOrder, orderType types.OrderType) lib.ErrorI {
	e.rwLock.Lock()
	defer e.rwLock.Unlock()
	// validate parameters
	if err := e.validateOrderParameters(order.OrderId, orderType); err != nil {
		return ErrValidateOrder(err)
	}
	// marshal order to JSON
	bz, err := json.Marshal(order)
	if err != nil {
		return ErrMarshalOrder(err)
	}
	// build archive file path
	archiveFilePath, err := e.buildArchiveFilePath(order.OrderId, orderType)
	if err != nil {
		return ErrWriteOrder(err)
	}
	// create temporary file for atomic write
	tempPath := archiveFilePath + tempSuffix
	// write data to temporary file
	if err := os.WriteFile(tempPath, bz, 0644); err != nil {
		return ErrWriteOrder(fmt.Errorf("failed to write archive temporary file: %w", err))
	}
	// atomically rename temporary file to final filename
	if err := os.Rename(tempPath, archiveFilePath); err != nil {
		// cleanup temporary file on failure
		os.Remove(tempPath)
		return ErrWriteOrder(fmt.Errorf("failed to rename archive temporary file: %w", err))
	}
	// e.logger.Debugf("OrderStore: Archived %d bytes to %s", len(bz), archiveFilePath)
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
	entries, err := os.ReadDir(e.absStoragePath)
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
				e.logger.Errorf("[ORACLE-STORE] Failed to decode order id in filename: %s", err.Error())
				continue
			}
			orderIds = append(orderIds, id)
		}
	}
	// e.logger.Debugf("OrderStore: All %s IDs %v", orderType, orderIds)
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
	// convert to hex string (orderId is already validated by caller)
	orderIdHex := hex.EncodeToString(orderId)
	// build filename with validated components
	filename := fmt.Sprintf("%s.%s%s", orderIdHex, string(orderType), jsonExtension)
	// use absolute path for security
	filePath := filepath.Join(e.absStoragePath, filename)
	// clean the final path to resolve any remaining path elements
	filePath = filepath.Clean(filePath)
	// ensure the resolved path is within the storage directory using absolute paths
	if !e.isPathWithinDirectory(filePath, e.absStoragePath) {
		return "", fmt.Errorf("path traversal attempt detected: resolved path outside storage directory")
	}
	return filePath, nil
}

// buildArchiveFilePath builds a file path for an archived order JSON file
func (e *OracleDiskStorage) buildArchiveFilePath(orderId []byte, orderType types.OrderType) (string, error) {
	// convert to hex string (orderId is already validated by caller)
	orderIdHex := hex.EncodeToString(orderId)
	// build filename with validated components
	filename := fmt.Sprintf("%s.%s%s", orderIdHex, string(orderType), jsonExtension)
	// determine archive subdirectory based on order type
	var archiveSubDir string
	switch orderType {
	case types.LockOrderType:
		archiveSubDir = "lock"
	case types.CloseOrderType:
		archiveSubDir = "close"
	default:
		return "", fmt.Errorf("invalid order type for archive: %s", orderType)
	}
	// validate subdirectory name
	if err := e.validateFilename(archiveSubDir); err != nil {
		return "", fmt.Errorf("invalid archive subdirectory: %w", err)
	}
	// build full archive path using absolute paths
	archiveDir := filepath.Join(e.absArchivePath, archiveSubDir)
	filePath := filepath.Join(archiveDir, filename)
	// clean the final path to resolve any remaining path elements
	filePath = filepath.Clean(filePath)
	// ensure the resolved path is within the archive directory using absolute paths
	if !e.isPathWithinDirectory(filePath, e.absArchivePath) {
		return "", fmt.Errorf("path traversal attempt detected: resolved path outside archive directory")
	}
	return filePath, nil
}

// validateFilename validates a filename component to prevent path traversal
func (e *OracleDiskStorage) validateFilename(filename string) error {
	// check for empty filename
	if filename == "" {
		return errors.New("filename cannot be empty")
	}
	// check for path traversal sequences
	if strings.Contains(filename, "..") {
		return errors.New("filename cannot contain '..' sequences")
	}
	// check for path separators
	if strings.ContainsAny(filename, "/\\") {
		return errors.New("filename cannot contain path separators")
	}
	// check for null bytes
	if strings.Contains(filename, "\x00") {
		return errors.New("filename cannot contain null bytes")
	}
	// check for control characters
	for _, r := range filename {
		if r < 32 || r == 127 {
			return errors.New("filename cannot contain control characters")
		}
	}
	return nil
}

// isPathWithinDirectory securely checks if a path is within a given directory
// This uses absolute paths and proper canonicalization to prevent bypass attempts
func (e *OracleDiskStorage) isPathWithinDirectory(targetPath, allowedDir string) bool {
	// get absolute path of target
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	// clean both paths to resolve symlinks and path elements
	absTarget = filepath.Clean(absTarget)
	allowedDir = filepath.Clean(allowedDir)
	// ensure both paths end with separator for proper prefix checking
	if !strings.HasSuffix(allowedDir, string(filepath.Separator)) {
		allowedDir += string(filepath.Separator)
	}
	if !strings.HasSuffix(absTarget, string(filepath.Separator)) {
		// for files, check if the directory part is within allowed directory
		targetDir := filepath.Dir(absTarget) + string(filepath.Separator)
		return strings.HasPrefix(targetDir, allowedDir)
	}
	// for directories, check direct prefix
	return strings.HasPrefix(absTarget, allowedDir)
}
