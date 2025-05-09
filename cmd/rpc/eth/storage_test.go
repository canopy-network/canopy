package eth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewStorage(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test creating storage with existing directory
	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage with existing directory: %v", err)
	}
	if storage.baseDir != tempDir {
		t.Errorf("Expected baseDir to be %s, got %s", tempDir, storage.baseDir)
	}

	// Test creating storage with non-existing directory
	nonExistingDir := filepath.Join(tempDir, "new-dir")
	storage, err = NewStorage(nonExistingDir)
	if err != nil {
		t.Fatalf("Failed to create storage with non-existing directory: %v", err)
	}

	// Check if directory was created
	if _, err := os.Stat(nonExistingDir); os.IsNotExist(err) {
		t.Errorf("Directory %s was not created", nonExistingDir)
	}
}

func TestWriteAndReadTx(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	blockHeight := uint64(12345)
	txID := "abcdef1234"
	data := "sample transaction data"

	// Test writing transaction
	err = storage.WriteTx(blockHeight, txID, data)
	if err != nil {
		t.Fatalf("Failed to write transaction: %v", err)
	}

	// Check if file exists
	filename := filepath.Join(tempDir, "12345-abcdef1234.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("Transaction file was not created at %s", filename)
	}

	// Test reading transaction
	tx, err := storage.ReadTx(blockHeight, txID)
	if err != nil {
		t.Fatalf("Failed to read transaction: %v", err)
	}
	if tx.Data != data {
		t.Errorf("Expected data %s, got %s", data, tx.Data)
	}

	// Verify file contents directly
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read transaction file: %v", err)
	}

	var fileTx Transaction
	if err := json.Unmarshal(fileBytes, &fileTx); err != nil {
		t.Fatalf("Failed to unmarshal transaction from file: %v", err)
	}

	if fileTx.Data != data {
		t.Errorf("Expected file data %s, got %s", data, fileTx.Data)
	}
}

func TestReadNonExistentTx(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Attempt to read a non-existent transaction
	tx, err := storage.ReadTx(999, "nonexistent")
	if err == nil {
		t.Error("Expected error when reading non-existent transaction, got nil")
	}
	if tx != nil {
		t.Errorf("Expected nil transaction, got %+v", tx)
	}
}

func TestTxExists(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	blockHeight := uint64(12345)
	txID := "abcdef1234"
	data := "sample transaction data"

	// Check non-existent transaction
	if storage.TxExists(blockHeight, txID) {
		t.Error("Expected TxExists to return false for non-existent transaction")
	}

	// Write transaction
	err = storage.WriteTx(blockHeight, txID, data)
	if err != nil {
		t.Fatalf("Failed to write transaction: %v", err)
	}

	// Check if transaction exists
	if !storage.TxExists(blockHeight, txID) {
		t.Error("Expected TxExists to return true for existing transaction")
	}
}

func TestDeleteTx(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	blockHeight := uint64(12345)
	txID := "abcdef1234"
	data := "sample transaction data"

	// Write transaction
	err = storage.WriteTx(blockHeight, txID, data)
	if err != nil {
		t.Fatalf("Failed to write transaction: %v", err)
	}

	// Delete transaction
	err = storage.DeleteTx(blockHeight, txID)
	if err != nil {
		t.Fatalf("Failed to delete transaction: %v", err)
	}

	// Check if transaction still exists
	if storage.TxExists(blockHeight, txID) {
		t.Error("Transaction should have been deleted")
	}

	// Test deleting a non-existent transaction
	err = storage.DeleteTx(999, "nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent transaction, got nil")
	}
}

func TestListTxsForBlock(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	blockHeight := uint64(12345)

	// Test empty block
	txs, err := storage.ListTxsForBlock(blockHeight)
	if err != nil {
		t.Fatalf("Failed to list transactions for empty block: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("Expected 0 transactions for empty block, got %d", len(txs))
	}

	// Write multiple transactions
	expectedTxIDs := []string{"tx1", "tx2", "tx3"}
	for _, txID := range expectedTxIDs {
		err = storage.WriteTx(blockHeight, txID, "data-"+txID)
		if err != nil {
			t.Fatalf("Failed to write transaction %s: %v", txID, err)
		}
	}

	// Write transactions for different block
	err = storage.WriteTx(blockHeight+1, "other-tx", "other-data")
	if err != nil {
		t.Fatalf("Failed to write transaction for different block: %v", err)
	}

	// List transactions for block
	txs, err = storage.ListTxsForBlock(blockHeight)
	if err != nil {
		t.Fatalf("Failed to list transactions: %v", err)
	}

	// Check if we got all expected transaction IDs
	if len(txs) != len(expectedTxIDs) {
		t.Errorf("Expected %d transactions, got %d", len(expectedTxIDs), len(txs))
	}

	// Create a map for easier lookup
	txMap := make(map[string]bool)
	for _, txID := range txs {
		txMap[txID] = true
	}

	// Check that all expected txIDs are in the result
	for _, expectedTxID := range expectedTxIDs {
		if !txMap[expectedTxID] {
			t.Errorf("Expected txID %s not found in result", expectedTxID)
		}
	}
}

func TestConcurrentWrites(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Number of concurrent writes
	const numWrites = 100

	// Use a wait group to wait for all goroutines to finish
	done := make(chan bool)

	// Launch concurrent writes
	for i := 0; i < numWrites; i++ {
		go func(index int) {
			blockHeight := uint64(index / 10) // Group transactions into blocks
			txID := fmt.Sprintf("tx-%d", index)
			data := fmt.Sprintf("data-%d", index)

			err := storage.WriteTx(blockHeight, txID, data)
			if err != nil {
				t.Errorf("Failed to write transaction in goroutine: %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numWrites; i++ {
		<-done
	}

	// Verify that all transactions were written correctly
	for i := 0; i < numWrites; i++ {
		blockHeight := uint64(i / 10)
		txID := fmt.Sprintf("tx-%d", i)
		expectedData := fmt.Sprintf("data-%d", i)

		tx, err := storage.ReadTx(blockHeight, txID)
		if err != nil {
			t.Errorf("Failed to read transaction %s: %v", txID, err)
			continue
		}

		if tx.Data != expectedData {
			t.Errorf("Expected data %s for transaction %s, got %s", expectedData, txID, tx.Data)
		}
	}
}

func TestAtomicWrites(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a read-only directory to test atomic write failure
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0755); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Make the directory read-only after creation
	if err := os.Chmod(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}

	storage, err := NewStorage(readOnlyDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Attempt to write to read-only directory should fail
	err = storage.WriteTx(123, "test-tx", "test-data")
	if err == nil {
		t.Error("Expected write to read-only directory to fail, but it succeeded")
	}

	// Check that no partial files were left behind
	tempFiles, err := filepath.Glob(filepath.Join(readOnlyDir, "*.tmp"))
	if err != nil {
		t.Fatalf("Failed to list temporary files: %v", err)
	}

	if len(tempFiles) > 0 {
		t.Errorf("Found %d temporary files that were not cleaned up", len(tempFiles))
		for _, file := range tempFiles {
			t.Logf("Temporary file: %s", file)
		}
	}
}
