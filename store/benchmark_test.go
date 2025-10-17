package store

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cockroachdb/pebble"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestBenchmark(t *testing.T) {
	fmt.Println("=== COMPARISON: Few Keys, Many Versions ===")
	testPebble(t, 50, 100_000)
	testBadger(t, 50, 100_000)

	fmt.Println("\n=== COMPARISON: Many Keys, Few Versions ===")
	testPebble(t, 100_000, 50)
	testBadger(t, 100_000, 50)
}

func testBadger(t *testing.T, numKeys, numVersions int) {
	db, err := badger.OpenManaged(
		badger.DefaultOptions("./test_badger").
			WithNumVersionsToKeep(math.MaxInt64).
			WithLoggingLevel(badger.ERROR).
			WithMemTableSize(int64(128 * units.MB)),
	)
	defer os.RemoveAll("./test_badger")
	defer db.Close()
	require.NoError(t, err)
	// generate keys
	keys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = crypto.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	// execute writes
	start := time.Now()
	for i := 0; i < numVersions; i++ {
		tx := db.NewWriteBatchAt(uint64(i + 1))
		for j := 0; j < numKeys; j++ {
			require.NoError(t, tx.Set(keys[j], keys[j]))
		}
		require.NoError(t, tx.Flush())
	}
	fmt.Println("BADGER WRITE TIME:", time.Since(start))
	start = time.Now()
	// execute latest iterator
	tx := db.NewTransactionAt(uint64(numVersions), false)
	it := tx.NewIterator(badger.IteratorOptions{Reverse: true})
	require.NoError(t, err)
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		item.Key()
		item.ValueCopy(nil)
		count++
	}
	it.Close()
	tx.Discard()
	fmt.Printf("LATEST ITERATOR (%d VALUES) TIME: %s\n", count, time.Since(start))
	start = time.Now()
	// execute historical iterator
	tx = db.NewTransactionAt(uint64(1), false)
	it = tx.NewIterator(badger.IteratorOptions{
		Reverse: true,
	})
	require.NoError(t, err)
	count = 0
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		item.Key()
		item.ValueCopy(nil)
		count++
	}
	it.Close()
	tx.Discard()
	fmt.Printf("HISTORICAL ITERATOR (%d VALUES) TIME: %s\n\n", count, time.Since(start))
}

func testPebble(t *testing.T, numKeys, numVersions int) {
	fmt.Printf("\n--- OPTIMIZED VersionedStore (PebbleDB): %d keys, %d versions ---\n", numKeys, numVersions)

	cache := pebble.NewCache(512 << 20) // 512MB cache
	defer cache.Unref()
	d, err := pebble.Open("./test_pebble_versioned_opt", &pebble.Options{
		DisableWAL:            false,     // Keep WAL but optimize other settings
		MemTableSize:          512 << 20, // Larger memtable to reduce flushes
		L0CompactionThreshold: 20,        // Delay compaction during bulk writes
		L0StopWritesThreshold: 40,        // Much higher threshold
		MaxOpenFiles:          5000,      // More file handles
		Cache:                 cache,     // Block cache
		FormatMajorVersion:    pebble.FormatNewest,
	})
	require.NoError(t, err)
	defer os.RemoveAll("./test_pebble_versioned_opt")
	defer d.Close()

	// generate keys
	keys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = crypto.Hash([]byte(fmt.Sprintf("%d", i)))
	}

	// execute writes - PROPER VERSIONING: Must commit per version like blockchain
	start := time.Now()

	for version := 1; version <= numVersions; version++ {
		batch := d.NewIndexedBatch()
		vs, err := NewVersionedStore(d, batch, uint64(version))
		require.NoError(t, err)

		for j := 0; j < numKeys; j++ {
			require.NoError(t, vs.Set(keys[j], keys[j]))
		}
		// MUST commit per version for proper blockchain semantics
		require.NoError(t, vs.Commit())
	}
	fmt.Printf("OPTIMIZED PEBBLE WRITE TIME: %s\n", time.Since(start))

	// execute latest iterator
	start = time.Now()
	vs, err := NewVersionedStore(d, d.NewIndexedBatch(), uint64(numVersions))
	require.NoError(t, err)
	it, err := vs.RevIterator(nil)
	require.NoError(t, err)
	count := 0
	for ; it.Valid(); it.Next() {
		it.Key()
		it.Value()
		count++
	}
	it.Close()
	fmt.Printf("OPTIMIZED LATEST ITERATOR (%d VALUES at version %d) TIME: %s\n", count, numVersions, time.Since(start))

	// execute historical iterator (query at version 1 if available)
	if numVersions >= 1 {
		start = time.Now()
		vs, err = NewVersionedStore(d, d.NewIndexedBatch(), uint64(1))
		require.NoError(t, err)
		it, err := vs.RevIterator(nil)
		require.NoError(t, err)
		count = 0
		for ; it.Valid(); it.Next() {
			it.Key()
			it.Value()
			count++
		}
		it.Close()
		fmt.Printf("OPTIMIZED HISTORICAL ITERATOR (%d VALUES at version 1) TIME: %s\n\n", count, time.Since(start))
	}
}
