package eth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Storage handles transaction persistence
type Storage struct {
	baseDir string
}

// NewStorage creates a new storage instance
func NewStorage(baseDir string) (*Storage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		baseDir: baseDir,
	}, nil
}

// Transaction represents the simplified transaction structure we're storing
type StoredTransaction struct {
	Data string `json:"data"`
}

// WriteTx writes a transaction's data to disk in an atomic way
func (s *Storage) WriteTx(blockHeight uint64, txID string, data string) error {
	// Create filename in the format "<height>-<txid>.json"
	filename := fmt.Sprintf("%d-%s.json", blockHeight, txID)
	filepath := filepath.Join(s.baseDir, filename)

	// Create temporary filepath for atomic write
	tempFilepath := filepath + ".tmp"

	// Prepare transaction with only the data field
	tx := StoredTransaction{
		Data: data,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Write to temporary file
	if err := ioutil.WriteFile(tempFilepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temp file to final file (atomic operation)
	if err := os.Rename(tempFilepath, filepath); err != nil {
		// Try to clean up the temporary file
		os.Remove(tempFilepath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// ReadTx reads a transaction from disk
func (s *Storage) ReadTx(blockHeight uint64, txID string) (*Transaction, error) {
	// Create filename in the format "<height>-<txid>.json"
	filename := fmt.Sprintf("%d-%s.json", blockHeight, txID)
	filepath := filepath.Join(s.baseDir, filename)

	// Read file
	jsonData, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	// Unmarshal JSON
	var tx Transaction
	if err := json.Unmarshal(jsonData, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &tx, nil
}

// TxExists checks if a transaction file exists
func (s *Storage) TxExists(blockHeight uint64, txID string) bool {
	filename := fmt.Sprintf("%d-%s.json", blockHeight, txID)
	filepath := filepath.Join(s.baseDir, filename)

	_, err := os.Stat(filepath)
	return err == nil
}

// DeleteTx removes a transaction file
func (s *Storage) DeleteTx(blockHeight uint64, txID string) error {
	filename := fmt.Sprintf("%d-%s.json", blockHeight, txID)
	filepath := filepath.Join(s.baseDir, filename)

	return os.Remove(filepath)
}

// ListTxsForBlock returns all transaction IDs for a given block height
func (s *Storage) ListTxsForBlock(blockHeight uint64) ([]string, error) {
	pattern := fmt.Sprintf("%d-*.json", blockHeight)
	matches, err := filepath.Glob(filepath.Join(s.baseDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	// Extract transaction IDs from filenames
	txIDs := make([]string, 0, len(matches))
	prefix := fmt.Sprintf("%d-", blockHeight)

	for _, match := range matches {
		basename := filepath.Base(match)
		// Remove the prefix and .json suffix to get the txID
		txID := basename[len(prefix) : len(basename)-5]
		txIDs = append(txIDs, txID)
	}

	return txIDs, nil
}
