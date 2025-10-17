package store

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cockroachdb/pebble"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

// FuzzTestVersionedStore performs comprehensive fuzz testing with 1000 random operations
func FuzzTestVersionedStore(f *testing.F) {
	// Add initial seed inputs - use fixed seeds for reproducible tests
	f.Add(int64(12345))
	f.Add(int64(67890))
	f.Add(int64(11111))

	f.Fuzz(func(t *testing.T, seed int64) {
		// Create deterministic random generator from seed
		rng := rand.New(rand.NewSource(seed))

		// Create temporary directories
		badgerPath := t.TempDir()
		pebblePath := t.TempDir()

		// Setup BadgerDB
		badgerDB, err := badger.OpenManaged(
			badger.DefaultOptions(badgerPath).
				WithNumVersionsToKeep(math.MaxInt64).
				WithLoggingLevel(badger.ERROR).
				WithInMemory(false),
		)
		require.NoError(t, err)
		defer badgerDB.Close()

		// Setup PebbleDB
		cache := pebble.NewCache(64 << 20) // 64MB cache
		defer cache.Unref()
		pebbleDB, err := pebble.Open(pebblePath, &pebble.Options{
			DisableWAL:            false,
			MemTableSize:          32 << 20, // 32MB
			L0CompactionThreshold: 4,
			L0StopWritesThreshold: 12,
			MaxOpenFiles:          1000,
			Cache:                 cache,
			FormatMajorVersion:    pebble.FormatNewest,
		})
		require.NoError(t, err)
		defer pebbleDB.Close()

		// Perform 1000 random operations
		keyPool := []string{"key1", "key2", "key3", "testkey", "longkey", "data", "value"}
		valuePool := []string{"value1", "value2", "data", "test", "content", "info", ""}

		for iteration := 1; iteration <= 1000; iteration++ {
			// Generate random operation parameters
			version := uint64(iteration)
			key := []byte(keyPool[rng.Intn(len(keyPool))])
			value := []byte(valuePool[rng.Intn(len(valuePool))])
			shouldDelete := rng.Float32() < 0.3 // 30% chance of delete operation

			// Skip operations that would create invalid states
			if len(key) == 0 {
				continue
			}

			// Test both stores with identical random operations
			testVersionedStoreOperations(t, badgerDB, pebbleDB, version, key, value, shouldDelete)
		}

		// Final comprehensive read test - verify all keys at latest version
		finalVersion := uint64(1000)
		verifyFinalState(t, badgerDB, pebbleDB.NewSnapshot(), finalVersion, keyPool)
	})
}

// verifyFinalState performs a comprehensive verification of the final database state
func verifyFinalState(t *testing.T, badgerDB *badger.DB, pebbleDB pebble.Reader, version uint64, keys []string) {
	// Create read-only transactions at final version
	badgerReadTxn := badgerDB.NewTransactionAt(version, false)
	defer badgerReadTxn.Discard()
	badgerReader := NewBadgerTxn(badgerReadTxn, nil, nil, false, false, version)
	defer badgerReader.Close()

	pebbleReader, err := NewVersionedStore(pebbleDB, nil, version)
	require.NoError(t, err)
	defer pebbleReader.Close()

	// Test individual key reads
	for _, key := range keys {
		keyBytes := []byte(key)

		val1, err1 := badgerReader.Get(keyBytes)
		val2, err2 := pebbleReader.Get(keyBytes)

		// Both should have same error state
		require.Equal(t, err1 != nil, err2 != nil, "Get error state mismatch for key %s", key)

		if err1 == nil && err2 == nil {
			require.Equal(t, val1, val2, "Value mismatch for key %s: badger=%x, pebble=%x",
				key, val1, val2)
		}
	}

	// Test full iteration to ensure both stores see the same keys
	compareIterators(t, badgerReader, pebbleReader, nil)
}

// Original single-operation fuzz test for backward compatibility
func FuzzVersionedStore(f *testing.F) {
	// Add initial seed inputs
	f.Add(uint64(rand.Intn(999)), []byte("key1"), []byte("value1"), false)
	f.Add(uint64(rand.Intn(999)), []byte("test"), []byte("data"), true)
	f.Add(uint64(rand.Intn(999)), []byte("long_key_with_many_characters"), []byte("some_value_data"), false)

	f.Fuzz(func(t *testing.T, version uint64, key []byte, value []byte, shouldDelete bool) {
		// Skip invalid inputs
		if len(key) == 0 || len(key) > 200 {
			t.Skip("Invalid key length")
		}
		if version > 1000 { // Keep versions reasonable for testing
			t.Skip("Version too large")
		}

		// Create temporary directories
		badgerPath := t.TempDir()
		pebblePath := t.TempDir()

		// Setup BadgerDB
		badgerDB, err := badger.OpenManaged(
			badger.DefaultOptions(badgerPath).
				WithNumVersionsToKeep(math.MaxInt64).
				WithLoggingLevel(badger.ERROR).
				WithInMemory(false),
		)
		require.NoError(t, err)
		defer badgerDB.Close()

		// Setup PebbleDB
		cache := pebble.NewCache(64 << 20) // 64MB cache
		defer cache.Unref()
		pebbleDB, err := pebble.Open(pebblePath, &pebble.Options{
			DisableWAL:            false,
			MemTableSize:          32 << 20, // 32MB
			L0CompactionThreshold: 4,
			L0StopWritesThreshold: 12,
			MaxOpenFiles:          1000,
			Cache:                 cache,
			FormatMajorVersion:    pebble.FormatNewest,
		})
		require.NoError(t, err)
		defer pebbleDB.Close()

		// Test both stores with the same operations
		testVersionedStoreOperations(t, badgerDB, pebbleDB, version, key, value, shouldDelete)
	})
}

// testVersionedStoreOperations tests both stores with identical operations and compares results
func testVersionedStoreOperations(t *testing.T, badgerDB *badger.DB, pebbleDB *pebble.DB, version uint64, key []byte, value []byte, shouldDelete bool) {
	// Create BadgerDB transaction
	badgerTxn := badgerDB.NewWriteBatchAt(version)
	defer badgerTxn.Cancel()

	badgerStore := NewBadgerTxn(badgerDB.NewTransactionAt(version-1, false), badgerTxn, nil, false, false, version)
	defer badgerStore.Close()

	// Create PebbleDB versioned store
	pebbleBatch := pebbleDB.NewBatch()
	defer pebbleBatch.Close()

	pebbleSnapshot := pebbleDB.NewSnapshot()
	pebbleStore, err := NewVersionedStore(pebbleSnapshot, pebbleBatch, version)
	require.NoError(t, err)
	defer pebbleStore.Close()

	// Perform identical operations on both stores
	if shouldDelete {
		// First set a value, then delete it
		err1 := badgerStore.Set(key, value)
		err2 := pebbleStore.Set(key, value)
		require.Equal(t, err1, err2, "Set operations should have same error state")

		if err1 == nil && err2 == nil {
			err1 = badgerStore.Delete(key)
			err2 = pebbleStore.Delete(key)
			require.Equal(t, err1, err2, "Delete operations should have same error state")
		}
	} else {
		err1 := badgerStore.Set(key, value)
		err2 := pebbleStore.Set(key, value)
		require.Equal(t, err1, err2, "Set operations should have same error state")
	}

	// Flush both stores
	if badgerStore.Flush() == nil && pebbleStore.Commit() == nil {
		badgerTxn.Flush()
		// Read back and compare
		compareStoreReads(t, badgerDB, pebbleDB.NewSnapshot(), version, key, shouldDelete, value)
	}
}

// compareStoreReads compares read operations between BadgerDB and PebbleDB
func compareStoreReads(t *testing.T, badgerDB *badger.DB, pebbleDB pebble.Reader, version uint64, key []byte, shouldDelete bool, expectedValue []byte) {
	// Create read-only transactions
	badgerReadTxn := badgerDB.NewTransactionAt(version, false)
	defer badgerReadTxn.Discard()

	badgerReader := NewBadgerTxn(badgerReadTxn, nil, nil, false, false, version)
	defer badgerReader.Close()

	pebbleReader, err := NewVersionedStore(pebbleDB, nil, version)
	require.NoError(t, err)
	defer pebbleReader.Close()

	// Test Get operations
	val1, err1 := badgerReader.Get(key)
	val2, err2 := pebbleReader.Get(key)

	// Both should have same error state
	if err1 != nil && err2 != nil {
		// Both have errors - acceptable for this fuzz test
		return
	}

	require.Equal(t, err1 != nil, err2 != nil, "Both stores should have same error state for Get")

	if shouldDelete {
		// After deletion, both should return nil
		require.Nil(t, val1, "BadgerDB should return nil for deleted key")
		require.Nil(t, val2, "PebbleDB should return nil for deleted key")
	} else {
		// Both should return the same value
		require.Equal(t, val1, val2, "Both stores should return same value")
		if val1 != nil && val2 != nil {
			require.Equal(t, expectedValue, val1, "BadgerDB should return expected value")
			require.Equal(t, expectedValue, val2, "PebbleDB should return expected value")
		}
	}

	// Test iteration
	compareIterators(t, badgerReader, pebbleReader, nil)
}

// compareIterators compares iterator behavior between two stores
func compareIterators(t *testing.T, store1, store2 interface {
	Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI)
	RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI)
}, prefix []byte) {

	// Test forward iteration
	iter1, err1 := store1.Iterator(prefix)
	iter2, err2 := store2.Iterator(prefix)

	if err1 != nil || err2 != nil {
		// If either has an error, skip iteration comparison
		if iter1 != nil {
			iter1.Close()
		}
		if iter2 != nil {
			iter2.Close()
		}
		return
	}

	defer iter1.Close()
	defer iter2.Close()

	// Compare iteration results
	for iter1.Valid() && iter2.Valid() {
		key1, key2 := iter1.Key(), iter2.Key()
		val1, val2 := iter1.Value(), iter2.Value()

		require.True(t, bytes.Equal(key1, key2), "Iterator keys should match")
		require.True(t, bytes.Equal(val1, val2), "Iterator values should match")

		iter1.Next()
		iter2.Next()
	}

	// Both iterators should be invalid at the same time
	require.Equal(t, iter1.Valid(), iter2.Valid(), "Iterators should end at same time")
}

// TestVersionedStoreBasicOperations tests basic operations with known inputs
func TestVersionedStoreBasicOperations(t *testing.T) {
	// Create temporary directories
	badgerPath := t.TempDir()
	pebblePath := t.TempDir()

	// Setup BadgerDB
	badgerDB, err := badger.OpenManaged(
		badger.DefaultOptions(badgerPath).
			WithNumVersionsToKeep(math.MaxInt64).
			WithLoggingLevel(badger.ERROR),
	)
	require.NoError(t, err)
	defer badgerDB.Close()

	// Setup PebbleDB
	cache := pebble.NewCache(64 << 20)
	defer cache.Unref()
	pebbleDB, err := pebble.Open(pebblePath, &pebble.Options{
		DisableWAL:         false,
		MemTableSize:       32 << 20,
		Cache:              cache,
		FormatMajorVersion: pebble.FormatNewest,
	})
	require.NoError(t, err)
	defer pebbleDB.Close()

	// Test scenarios
	testCases := []struct {
		name    string
		version uint64
		key     []byte
		value   []byte
		delete  bool
	}{
		{"simple_set", 1, []byte("key1"), []byte("value1"), false},
		{"simple_delete", 2, []byte("key1"), []byte("value1"), true},
		{"empty_value", 3, []byte("key2"), []byte(""), false},
		{"large_key", 4, crypto.Hash([]byte("large_key_test")), []byte("test"), false},
		{"large_value", 5, []byte("small"), make([]byte, 1024), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testVersionedStoreOperations(t, badgerDB, pebbleDB, tc.version, tc.key, tc.value, tc.delete)
		})
	}
}

// TestVersionedStoreMultiVersion tests version handling
func TestVersionedStoreMultiVersion(t *testing.T) {
	pebblePath := t.TempDir()
	badgerPath := t.TempDir()

	// Setup PebbleDB
	cache := pebble.NewCache(64 << 20)
	defer cache.Unref()
	pebbleDB, err := pebble.Open(pebblePath, &pebble.Options{
		Cache:              cache,
		FormatMajorVersion: pebble.FormatNewest,
	})
	require.NoError(t, err)
	defer pebbleDB.Close()

	// Setup BadgerDB
	badgerDB, err := badger.OpenManaged(
		badger.DefaultOptions(badgerPath).
			WithNumVersionsToKeep(math.MaxInt64).
			WithLoggingLevel(badger.ERROR),
	)
	require.NoError(t, err)
	defer badgerDB.Close()

	key := []byte("version_test_key")

	// Write multiple versions
	for version := uint64(1); version <= 5; version++ {
		value := []byte(fmt.Sprintf("value_at_version_%d", version))

		// BadgerDB
		badgerTxn := badgerDB.NewWriteBatchAt(version)
		badgerStore := NewBadgerTxn(badgerDB.NewTransactionAt(version-1, false), badgerTxn, nil, false, false, version)
		require.NoError(t, badgerStore.Set(key, value))
		require.NoError(t, badgerStore.Flush())
		require.NoError(t, badgerTxn.Flush())
		badgerStore.Close()

		// PebbleDB
		pebbleBatch := pebbleDB.NewBatch()
		pebbleStore, err := NewVersionedStore(pebbleDB.NewSnapshot(), pebbleBatch, version)
		require.NoError(t, err)
		require.NoError(t, pebbleStore.Set(key, value))
		require.NoError(t, pebbleStore.Commit())
		pebbleStore.Close()
	}

	// Read from each version
	for version := uint64(1); version <= 5; version++ {
		expectedValue := fmt.Appendf(nil, "value_at_version_%d", version)

		// BadgerDB read
		badgerReadTxn := badgerDB.NewTransactionAt(version, false)
		badgerReader := NewBadgerTxn(badgerReadTxn, nil, nil, false, false, version)
		val1, err := badgerReader.Get(key)
		require.NoError(t, err)
		badgerReader.Close()

		// PebbleDB read
		pebbleReader, err := NewVersionedStore(pebbleDB.NewSnapshot(), nil, version)
		require.NoError(t, err)
		val2, err := pebbleReader.Get(key)
		require.NoError(t, err)
		pebbleReader.Close()

		// Values should match
		require.Equal(t, expectedValue, val1, "BadgerDB should return correct version")
		require.Equal(t, expectedValue, val2, "PebbleDB should return correct version")
		require.Equal(t, val1, val2, "Both stores should return same value for version %d", version)
	}
}
