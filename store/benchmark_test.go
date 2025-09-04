package store

import (
	"fmt"
	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cockroachdb/pebble/v2"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
	"math"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

var numKeys, numVersions = 10_000_000, 2

func TestT(t *testing.T) {
	d, err := pebble.Open("./test_pebble", &pebble.Options{
		DisableWAL:               true,
		MemTableSize:             8 << 20, // 8MB
		MaxConcurrentCompactions: func() int { return 8 },
		//L0CompactionThreshold:    16,
		FormatMajorVersion: pebble.FormatNewest,
	})
	defer d.Close()
	var db *VersionedStore
	// generate keys
	keys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = crypto.Hash([]byte(fmt.Sprintf("%d", i)))
	}
	// execute writes
	start := time.Now()
	for i := 0; i < numVersions; i++ {
		db, err = NewVersionedStore(nil, d.NewBatch(), uint64(i), false)
		require.NoError(t, err)
		for j := 0; j < numKeys; j++ {
			require.NoError(t, db.Set(keys[j], keys[j]))
		}
		require.NoError(t, db.Commit())
	}
	fmt.Println("WRITE TIME:", time.Since(start))
	f, err := os.Create("pebble.prof")
	if err != nil {
		t.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()
	start = time.Now()
	// execute latest iterator
	db, err = NewVersionedStore(d, d.NewBatch(), uint64(numVersions), false)
	require.NoError(t, err)
	it, err := db.Iterator(nil)
	require.NoError(t, err)
	count := 0
	for ; it.Valid(); it.Next() {
		it.Key()
		it.Value()
		count++
	}
	it.Close()
	fmt.Printf("LATEST ITERATOR (%d VALUES) TIME: %s\n", count, time.Since(start))
	start = time.Now()
	// execute historical iterator
	db, err = NewVersionedStore(d, d.NewBatch(), uint64(1), false)
	require.NoError(t, err)
	it, err = db.Iterator(nil)
	require.NoError(t, err)
	count = 0
	for ; it.Valid(); it.Next() {
		it.Key()
		it.Value()
		count++
	}
	it.Close()
	fmt.Printf("HISTORICAL ITERATOR (%d VALUES) TIME: %s\n", count, time.Since(start))
}

func TestT2(t *testing.T) {
	db, err := badger.OpenManaged(
		badger.DefaultOptions("./test_badger").
			WithNumVersionsToKeep(math.MaxInt64).
			WithLoggingLevel(badger.ERROR).
			WithMemTableSize(int64(128 * units.MB)),
	)
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
	fmt.Println("WRITE TIME:", time.Since(start))
	start = time.Now()
	// execute latest iterator
	tx := db.NewTransactionAt(uint64(numVersions), false)
	it := tx.NewIterator(badger.IteratorOptions{})
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
	it = tx.NewIterator(badger.IteratorOptions{})
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
	fmt.Printf("HISTORICAL ITERATOR (%d VALUES) TIME: %s\n", count, time.Since(start))
	start = time.Now()
}
