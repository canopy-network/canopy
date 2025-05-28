package eth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/canopy-network/canopy/lib"
)

// EthDiskStorage implements OrderStoreI interface for storing Ethereum orders on disk
type EthDiskStorage struct {
	storagePath string      // Path where order files will be stored
	logger      lib.LoggerI // Logger for verbose logging
}

// NewEthDiskStorage creates a new EthDiskStorage instance
func NewEthDiskStorage(storagePath string, logger lib.LoggerI) (*EthDiskStorage, error) {
	// Log the initialization
	logger.Infof("Initializing EthDiskStorage with path: %s", storagePath)

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		// Log the error
		logger.Errorf("Failed to create storage directory: %v", err)
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Log successful initialization
	logger.Info("EthDiskStorage initialized successfully")

	// Return the new instance
	return &EthDiskStorage{
		storagePath: storagePath,
		logger:      logger,
	}, nil
}

// WriteOrder writes an order to disk
func (s *EthDiskStorage) WriteOrder(orderId string, orderType string, blockHeight uint64, txHash string, data []byte) error {
	// Log the write operation
	s.logger.Debugf("Writing order: id=%s, type=%s, blockHeight=%d, txHash=%s", orderId, orderType, blockHeight, txHash)

	// Construct the filename
	filename := fmt.Sprintf("%s.%s.%d.%s.json", orderId, orderType, blockHeight, txHash)
	filepath := filepath.Join(s.storagePath, filename)

	// Create a temporary file for atomic write
	tempFilePath := filepath + ".tmp"
	s.logger.Debugf("Creating temporary file: %s", tempFilePath)

	// Write data to temporary file
	if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
		s.logger.Errorf("Failed to write to temporary file: %v", err)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Rename temporary file to final filename (atomic operation)
	s.logger.Debugf("Renaming temporary file to: %s", filepath)
	if err := os.Rename(tempFilePath, filepath); err != nil {
		s.logger.Errorf("Failed to rename temporary file: %v", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	// Log successful write
	s.logger.Infof("Successfully wrote order to: %s", filepath)
	return nil
}

// ReadOrder reads an order from disk
func (s *EthDiskStorage) ReadOrder(orderId string, orderType string) ([]byte, error) {
	// Log the read operation

	// Find the file matching the orderId and orderType
	pattern := fmt.Sprintf("%s.%s.*.*.json", orderId, orderType)
	matches, err := filepath.Glob(filepath.Join(s.storagePath, pattern))

	// Check for glob error
	if err != nil {
		return nil, fmt.Errorf("error searching for order files: %w", err)
	}

	// Check if any files were found
	if len(matches) == 0 {
		return nil, fmt.Errorf("order not found: id=%s, type=%s", orderId, orderType)
	}

	// If multiple files found, use the most recent one (assuming higher block height is more recent)
	if len(matches) > 1 {
		s.logger.Warnf("Multiple files found for order id=%s, type=%s, using the first one", orderId, orderType)
	}

	// Read the file
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read order file: %w", err)
	}

	// Log successful read
	s.logger.Infof("Successfully read order: %s", matches[0])
	return data, nil
}

// RemoveOrder removes an order from disk
func (s *EthDiskStorage) RemoveOrder(orderId string, orderType string) error {
	// Log the remove operation
	s.logger.Debugf("Removing order: id=%s, type=%s", orderId, orderType)

	// Find the file matching the orderId and orderType
	pattern := fmt.Sprintf("%s.%s.*.*.json", orderId, orderType)
	matches, err := filepath.Glob(filepath.Join(s.storagePath, pattern))

	// Check for glob error
	if err != nil {
		s.logger.Errorf("Error searching for order files: %v", err)
		return fmt.Errorf("error searching for order files: %w", err)
	}

	// Check if any files were found
	if len(matches) == 0 {
		s.logger.Warnf("No order found for removal: id=%s, type=%s", orderId, orderType)
		return fmt.Errorf("order not found for removal: id=%s, type=%s", orderId, orderType)
	}

	// Remove all matching files
	for _, file := range matches {
		s.logger.Debugf("Removing file: %s", file)
		if err := os.Remove(file); err != nil {
			s.logger.Errorf("Failed to remove file %s: %v", file, err)
			return fmt.Errorf("failed to remove file %s: %w", file, err)
		}
	}

	// Log successful removal
	s.logger.Infof("Successfully removed %d order file(s) for id=%s, type=%s", len(matches), orderId, orderType)
	return nil
}

// GetAllOrderIds gets all order ids present in the store for a specific order type
func (s *EthDiskStorage) GetAllOrderIds(orderType string) [][]byte {
	// Log the operation
	s.logger.Debugf("Getting all order IDs for type: %s", orderType)

	// Find all files matching the order type
	pattern := fmt.Sprintf("*.%s.*.*.json", orderType)
	matches, err := filepath.Glob(filepath.Join(s.storagePath, pattern))

	// Check for glob error
	if err != nil {
		s.logger.Errorf("Error searching for order files: %v", err)
		return [][]byte{}
	}

	// Extract order IDs from filenames
	orderIds := make([][]byte, 0, len(matches))
	for _, file := range matches {
		// Get just the filename without the path
		filename := filepath.Base(file)

		// Extract order ID from filename
		parts := strings.Split(filename, ".")
		if len(parts) >= 5 {
			orderId := parts[0]
			orderIds = append(orderIds, []byte(orderId))
		} else {
			s.logger.Warnf("Filename does not match expected format: %s", filename)
		}
	}

	// Log the result
	s.logger.Infof("Found %d order IDs for type %s", len(orderIds), orderType)
	return orderIds
}
