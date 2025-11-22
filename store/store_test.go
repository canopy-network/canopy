package store

import (
	"crypto/rand"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestStoreSetGetDelete(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	key, val := []byte("key"), []byte("val")
	require.NoError(t, store.Set(key, val))
	gotVal, err := store.Get(key)
	require.NoError(t, err)
	require.Equal(t, val, gotVal, fmt.Sprintf("wanted %s got %s", string(val), string(gotVal)))
	require.NoError(t, store.Delete(key))
	gotVal, err = store.Get(key)
	require.NoError(t, err)
	require.NotEqualf(t, gotVal, val, fmt.Sprintf("%s should be delete", string(val)))
	require.NoError(t, store.Close())
}

func TestIteratorCommitBasic(t *testing.T) {
	parent, _, cleanup := testStore(t)
	defer cleanup()
	prefix := "a/"
	expectedVals := []string{prefix + "a", prefix + "b", prefix + "c", prefix + "d", prefix + "e", prefix + "f", prefix + "g", prefix + "i", prefix + "j"}
	expectedValsReverse := []string{prefix + "j", prefix + "i", prefix + "g", prefix + "f", prefix + "e", prefix + "d", prefix + "c", prefix + "b", prefix + "a"}
	bulkSetKV(t, parent, prefix, "a", "c", "e", "g")
	_, err := parent.Commit()
	require.NoError(t, err)
	bulkSetKV(t, parent, prefix, "b", "d", "f", "h", "i", "j")
	require.NoError(t, parent.Delete([]byte(prefix+"h")))
	// forward - ensure cache iterator matches behavior of normal iterator
	cIt, err := parent.Iterator([]byte(prefix))
	require.NoError(t, err)
	validateIterators(t, expectedVals, cIt)
	cIt.Close()
	// backward - ensure cache iterator matches behavior of normal iterator
	rIt, err := parent.RevIterator([]byte(prefix))
	require.NoError(t, err)
	validateIterators(t, expectedValsReverse, rIt)
	rIt.Close()
}

func TestIteratorCommitAndPrefixed(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	prefix := "test/"
	prefix2 := "test2/"
	bulkSetKV(t, store, prefix, "a", "b", "c")
	bulkSetKV(t, store, prefix2, "c", "d", "e")
	it, err := store.Iterator([]byte(prefix))
	require.NoError(t, err)
	validateIterators(t, []string{"test/a", "test/b", "test/c"}, it)
	it.Close()
	it2, err := store.Iterator([]byte(prefix2))
	require.NoError(t, err)
	validateIterators(t, []string{"test2/c", "test2/d", "test2/e"}, it2)
	it2.Close()
	root1, err := store.Commit()
	require.NoError(t, err)
	it3, err := store.RevIterator([]byte(prefix))
	require.NoError(t, err)
	validateIterators(t, []string{"test/c", "test/b", "test/a"}, it3)
	it3.Close()
	root2, err := store.Commit()
	require.NoError(t, err)
	require.Equal(t, root1, root2)
	it4, err := store.RevIterator([]byte(prefix2))
	require.NoError(t, err)
	validateIterators(t, []string{"test2/e", "test2/d", "test2/c"}, it4)
	it4.Close()
}

func testStore(t *testing.T) (*Store, *badger.DB, func()) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR).WithDetectConflicts(false))
	require.NoError(t, err)
	store, err := NewStoreWithDB(lib.DefaultConfig(), db, nil, lib.NewDefaultLogger())
	require.NoError(t, err)
	return store, db, func() { store.Close() }
}

func validateIterators(t *testing.T, expectedKeys []string, iterators ...lib.IteratorI) {
	for _, it := range iterators {
		for i := 0; it.Valid(); func() { i++; it.Next() }() {
			got, wanted := string(it.Key()), expectedKeys[i]
			require.Equal(t, wanted, got, fmt.Sprintf("wanted %s got %s", wanted, got))
		}
	}
}

func bulkSetKV(t *testing.T, store lib.WStoreI, prefix string, keyValue ...string) {
	for _, kv := range keyValue {
		require.NoError(t, store.Set([]byte(prefix+kv), []byte(kv)))
	}
}

func TestDoublyNestedTxn(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	// set initial value to the store
	store.Set([]byte("base"), []byte("base"))
	// create a nested transaction
	nested := store.NewTxn()
	// set nested value
	nested.Set([]byte("nested"), []byte("nested"))
	// retrieve parent key
	value, err := nested.Get([]byte("base"))
	require.NoError(t, err)
	require.Equal(t, []byte("base"), value)
	// create a doubly nested transaction
	doublyNested := nested.NewTxn()
	// set doubly nested value
	doublyNested.Set([]byte("doublyNested"), []byte("doublyNested"))
	// commit doubly nested transaction
	err = doublyNested.Flush()
	// retrieve grandparent key
	value, err = doublyNested.Get([]byte("base"))
	require.NoError(t, err)
	require.Equal(t, []byte("base"), value)
	require.NoError(t, err)
	// verify value can be retrieved from nested the store but
	// not from the store itself
	value, err = nested.Get([]byte("doublyNested"))
	require.NoError(t, err)
	require.Equal(t, []byte("doublyNested"), value)
	value, err = store.Get([]byte("doublyNested"))
	require.NoError(t, err)
	require.Nil(t, value)
	// commit nested transaction
	err = nested.Flush()
	require.NoError(t, err)
	// verify both nested and doubly nested values can be retrieved from the store
	value, err = store.Get([]byte("nested"))
	require.NoError(t, err)
	require.Equal(t, []byte("nested"), value)
	value, err = store.Get([]byte("doublyNested"))
	require.NoError(t, err)
	require.Equal(t, []byte("doublyNested"), value)
}

// BadgerDB forced key deletion test

const (
	NumKeys   = 100_000
	KeySize   = units.KB
	ValueSize = units.KB
)

type KVPair struct {
	Key   []byte
	Value []byte
}

// // ----------------------------------------------------------------------------------------------------------------
// // BadgerDB garbage collector behavior is not well documented leading to many open issues in their repository
// // However, here is our current understanding based on experimentation
// // ----------------------------------------------------------------------------------------------------------------
// // 1. Manual Keep (Protection)
// //    - `badgerNoDiscardBit` prevents automatic GC of a key version.
// //    - However, it can be manually superseded by a manual removal
// //
// // 2. Manual Remove (Explicit Deletion or Pruning)
// //    - Deleting a key at a higher ts removes earlier versions once `discardTs >= ts`.
// //    - Setting `badgerDiscardEarlierVersions` is similar, except it retains the current version.
// //
// // 3. Auto Remove – Tombstones
// //    - Deleted keys (tombstoned) <= `discardTs` are automatically purged unless protected by `badgerNoDiscardBit`
// //
// // 4. Auto Remove – Set Entries
// //    - For non-deleted (live) keys, Badger retains the number of versions to retain is defined by `KeepNumVersions`.
// //    - Older versions exceeding this count are automatically eligible for GC.
// //
// //   Note:
// // - The first GC pass after updating `discardTs` and flushing memtable is deterministic
// // - Subsequent GC runs are probabilistic, depending on reclaimable space and value log thresholds
// // ----------------------------------------------------------------------------------------------------------------
// // Bits source: https://github.com/hypermodeinc/badger/blob/85389e88bf308c1dc271383b77b67f4ef4a85194/value.go#L37
func TestBadgerDBForcedDeletion(t *testing.T) {
	// cleanup
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR).WithDetectConflicts(false))
	require.NoError(t, err)
	// IMPORTANT
	// create a slice to hold keys and values
	kvs := make([]KVPair, NumKeys)
	// create a write batch to simulate lss store
	lssWriter := db.NewWriteBatchAt(math.MaxUint64)
	for i := range NumKeys {
		kvs[i] = RandKeyValue(t)
	}
	// write to latest store
	for i := range NumKeys {
		require.NoError(t, lssWriter.Set(kvs[i].Key, kvs[i].Value))
	}
	// HACK (ONLY WAY TO DO IT!)
	triggerEvict := []byte("some_random_trigger_key_")
	require.NoError(t, lssWriter.Set(triggerEvict, nil))
	// flush the batch
	require.NoError(t, lssWriter.Flush())
	// start benchmark
	s := time.Now()
	// create a new write batch to simulate lss store
	lssWriter = db.NewWriteBatchAt(math.MaxUint64)
	// delete 50%
	for i := range NumKeys / 2 {
		//require.NoError(t, lssWriter.Delete(triggerEvict)) // this also works but we do deletes simlar to below
		e := &badger.Entry{Key: kvs[i].Key}
		setMeta(e, badgerDeleteBit|badgerNoDiscardBit)
		require.NoError(t, lssWriter.SetEntryAt(e, math.MaxUint64))
	}
	require.NoError(t, lssWriter.Flush())
	// EVICTION LOGIC - Two-phase approach to handle badgerNoDiscardBit
	// Phase 1: Clear badgerNoDiscardBit from deleted entries to allow eviction
	// NOTE: This may NOT work on long running db instances that
	// have already been LSM flattened
	clearNoDiscardBitFromDeletedEntries(t, db, kvs[:NumKeys/2])
	// Phase 2: Standard eviction logic
	db.SetDiscardTs(math.MaxUint64)
	require.NoError(t, db.DropPrefix(triggerEvict))
	db.SetDiscardTs(0)
	// create a reader to checkout the status of the database
	tx := db.NewTransactionAt(math.MaxUint64, false)
	defer tx.Discard()
	// return all versions to not pseudo-skip 'tombstoned keys'
	it := tx.NewIterator(badger.IteratorOptions{AllVersions: true})
	defer it.Close()
	// measure count
	count := 0
	countDeleted := 0
	for it.Rewind(); it.Valid(); it.Next() {
		count++
		if it.Item().IsDeletedOrExpired() {
			countDeleted++
		}
	}
	require.Equal(t, NumKeys/2, count)
	// finish benchmark
	t.Log("Total time:", time.Since(s), "with", count, "keys")
	// ensure safe to write again even at a lower version
	// create a new write batch to simulate lss store
	lssWriter = db.NewWriteBatchAt(1)
	// delete 50%
	for i := range NumKeys / 2 {
		require.NoError(t, lssWriter.Set(kvs[i].Key, kvs[i].Value))
	}
	require.NoError(t, lssWriter.Flush())
}

// clearNoDiscardBitFromDeletedEntries clears the badgerNoDiscardBit from entries that are marked for deletion
// This allows them to be properly garbage collected during the eviction process
func clearNoDiscardBitFromDeletedEntries(t *testing.T, db *badger.DB, deletedKeys []KVPair) {
	// create a new write batch to modify the entries
	lssWriter := db.NewWriteBatchAt(math.MaxUint64)

	// directly rewrite the deleted entries with only badgerDeleteBit (no badgerNoDiscardBit)
	for _, kv := range deletedKeys {
		e := &badger.Entry{Key: kv.Key}
		// set only the delete bit, removing the no-discard bit protection
		setMeta(e, badgerDeleteBit)
		require.NoError(t, lssWriter.SetEntryAt(e, math.MaxUint64))
	}

	// flush the changes
	require.NoError(t, lssWriter.Flush())
}

func RandKeyValue(t *testing.T) KVPair {
	k, v := make([]byte, KeySize), make([]byte, ValueSize)
	rand.Read(k)
	rand.Read(v)
	return KVPair{k, v}
}
