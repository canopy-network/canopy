package store

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"slices"
	"testing"
	"unsafe"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestMaxTransaction(t *testing.T) {
	tempDirector := os.TempDir()
	db, err := badger.OpenManaged(badger.DefaultOptions(tempDirector).WithMemTableSize(128 * int64(units.MB)).
		WithNumVersionsToKeep(math.MaxInt).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	defer func() { db.Close() }()
	tx := db.NewTransactionAt(1, true)
	require.NoError(t, setBatchOptions(db, maxTransactionSize))
	i := 0
	totalBytes := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(string(debug.Stack()))
			fmt.Println(i)
			fmt.Println("TOTAL_MBs", float64(totalBytes)/1000000)
		}
	}()
	for ; i < 128000; i++ {
		k := numberToBytes(i)
		v := bytes.Repeat([]byte("b"), 1000)
		totalBytes += len(k) + len(v)
		if err = tx.Set(k, v); err != nil {
			fmt.Println(err.Error())
			fmt.Println(i)
			fmt.Println("TOTAL_MBs", float64(totalBytes)/1000000)
			tx.Discard()
			return
		}
	}
	require.NoError(t, tx.CommitAt(1, nil))
	require.NoError(t, FlushMemTable(db))
	require.NoError(t, db.Flatten(1))
}

func numberToBytes(n int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, uint32(n))
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

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

func TestPartitionHeight(t *testing.T) {
	tests := []struct {
		name     string
		height   uint64
		expected uint64
	}{
		{
			name:     "zero",
			height:   0,
			expected: 1,
		},
		{
			name:     "less than partition frequency",
			height:   999,
			expected: 1,
		},
		{
			name:     "greater than partition frequency",
			height:   1001,
			expected: 1000,
		},
		{
			name:     "2x partition frequency",
			height:   2789,
			expected: 2000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, partitionHeight(test.height), test.expected)
		})
	}
}

func TestShouldPartition(t *testing.T) {
	tests := []struct {
		name            string
		version         uint64
		partitionExists bool
		expected        bool
	}{
		{
			name:     "too early to partition",
			version:  10000,
			expected: false,
		},
		{
			name:            "ready for partition",
			version:         10001,
			partitionExists: false,
			expected:        true,
		},
		{
			name:            "partition already exists",
			version:         10001,
			partitionExists: true,
			expected:        false,
		},
		{
			name:            "not partition boundary",
			version:         10002,
			partitionExists: false,
			expected:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, _, cleanup := testStore(t)
			defer cleanup()

			store.version = test.version
			if test.partitionExists {
				err := store.hss.Set([]byte(partitionExistsKey), []byte(partitionExistsKey))
				require.NoError(t, err)
			}

			result := store.ShouldPartition()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestPartition(t *testing.T) {
	// initialize store
	store, _, cleanup := testStore(t)
	defer cleanup()

	// Set the store version just before the partition boundary (9999)
	store.version = partitionFrequency - 1

	// set up test data at the initial version (9999)
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// insert test data
	for k, v := range testData {
		err := store.Set([]byte(k), []byte(v))
		require.NoError(t, err)
	}

	// Commit the data to finalize the version
	_, err := store.Commit()
	require.NoError(t, err)

	// commit one more time to trigger the partition (10001)
	_, err = store.Commit()
	require.NoError(t, err)

	// check whether the partition should run
	require.True(t, store.ShouldPartition())

	// disable GC for test (as BadgerDB doesn't support GC in In-Memory mode)
	store.isGarbageCollecting.Store(true)
	defer store.isGarbageCollecting.Store(false)

	// perform partition
	store.Partition()

	// calculate partition height
	snapshotHeight := partitionHeight(store.Version())

	// verify partition exists key is set in the correct partition
	reader := store.db.NewTransactionAt(snapshotHeight, false)
	defer reader.Discard()
	writer := store.db.NewWriteBatchAt(0)
	partitionPrefix := historicalPrefix(partitionHeight(store.Version()))
	hss := NewBadgerTxn(reader, writer, []byte(partitionPrefix), false, store.log)
	value, err := hss.Get([]byte(partitionExistsKey))
	require.NoError(t, err)
	require.Equal(t, []byte(partitionExistsKey), value)

	// Verify data in historical partition
	for k, v := range testData {
		value, err := hss.Get([]byte(k))
		require.NoError(t, err)
		require.Equal(t, []byte(v), value)
	}

	discardTs := store.db.MaxVersion()
	require.Equal(t, snapshotHeight+1, discardTs)
}

func TestPartitionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// create temp directory for test db
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test-db")

	// set up db with a configuration that will trigger a GC after multiple partitions
	db, err := badger.OpenManaged(badger.DefaultOptions(dbPath).
		WithNumVersionsToKeep(math.MaxInt).
		WithLoggingLevel(badger.ERROR).
		WithMemTableSize(int64(100 * units.Kilobyte)).
		WithValueThreshold(0).
		WithValueLogFileSize(int64(2 * units.Megabyte)))
	require.NoError(t, err)

	// set up the store
	store, err := NewStoreWithDB(db, nil, lib.NewDefaultLogger(), true)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	// create a large dataset to intentionally trigger garbage collection (GC).
	// based on the current configuration, GC is expected to run after creating 4 partitions
	const datasetSize = partitionFrequency*3 + 1
	const valueSize = 4096 // 4KB values

	// create a buffer large enough for testing
	randValue := make([]byte, valueSize)
	rand.Read(randValue)

	const keyPrefix = "gc-key-"
	// insert large dataset at various heights before partition boundary
	for i := 0; i < int(datasetSize); i++ {
		// use a key to be updated repeatedly to generate versions
		key := fmt.Appendf(nil, "%s%d", keyPrefix, i%100)

		// make an unique value per height to ensure different data
		value := make([]byte, valueSize)
		copy(value, randValue)
		binary.BigEndian.PutUint64(value, uint64(i))

		// set the value in the db
		require.NoError(t, store.Set(key, value))

		// commit regularly to create multiple versions
		_, err := store.Commit()
		require.NoError(t, err)

		if store.ShouldPartition() {
			// perform partition (this one should actually run the GC eventually after multiple partitions)
			store.Partition()
		}
	}

	// calculate last partition height
	snapshotHeight := partitionHeight(store.Version())

	// verify partition exists key is set in the correct partition
	reader := store.db.NewTransactionAt(snapshotHeight, false)
	defer reader.Discard()
	// set up a writer for the partition
	writer := store.db.NewWriteBatchAt(0)
	defer writer.Cancel()
	partitionPrefix := historicalPrefix(partitionHeight(store.Version()))
	hss := NewBadgerTxn(reader, writer, []byte(partitionPrefix), false, store.log)
	value, err := hss.Get([]byte(partitionExistsKey))
	require.NoError(t, err)
	require.Equal(t, []byte(partitionExistsKey), value)

	// verify some keys from the dataset and delete them for the last prune
	deletedKeys := make([]string, 0, 10)
	for i := range 10 {
		key := fmt.Appendf(nil, "%s%d", keyPrefix, i)
		val, err := hss.Get(key)
		require.NoError(t, err)
		require.NotNil(t, val, "Key should exist in historical partition")
		require.NoError(t, store.Delete(key))
		deletedKeys = append(deletedKeys, string(key))
	}

	// Force additional commits until the partition boundary is reached
	for range int(partitionFrequency) {
		_, err = store.Commit()
		require.NoError(t, err)
	}

	// perform one last partition and thus one last GC trigger
	require.True(t, store.ShouldPartition())
	store.Partition()

	// Get the final snapshot height after partition
	finalSnapshotHeight := partitionHeight(store.Version())

	tx := db.NewTransactionAt(finalSnapshotHeight, true)
	archiveStore := NewBadgerTxn(tx, writer, []byte(latestStatePrefix), false, lib.NewDefaultLogger())
	defer archiveStore.Close()

	// use an archive iterator to iterate through the deleted keys
	iterator, err := archiveStore.ArchiveIterator(nil)
	require.NoError(t, err)
	it := iterator.(*Iterator)
	defer it.Close()

	for ; it.Valid() || len(deletedKeys) > 0; it.Next() {
		// check the deleted items are still in the db and are marked as expected
		if idx := slices.Index(deletedKeys, string(it.Key())); idx >= 0 {
			require.True(t, getMeta(it.reader.Item())&badgerDeleteBit != 0)
			deletedKeys = slices.Delete(deletedKeys, idx, idx+1)
		}
	}

	// confirm all the deleted items were found in the archive store
	require.Len(t, deletedKeys, 0)
}

func TestBadgerBitBehavior(t *testing.T) {
	tests := []struct {
		name             string
		nonPruneable     bool
		finalBit         byte
		expectedKeyCount int
	}{
		{
			name:             "discard earlier versions bit",
			finalBit:         badgerDiscardEarlierVersions,
			expectedKeyCount: 1,
		},
		{
			name:             "delete bit",
			finalBit:         badgerDeleteBit,
			expectedKeyCount: 0,
		},
		{
			name:             "discard earlier versions bit",
			finalBit:         badgerDiscardEarlierVersions | badgerNoDiscardBit,
			expectedKeyCount: 1000,
		},
		{
			name:             "delete bit",
			finalBit:         badgerDeleteBit | badgerNoDiscardBit,
			expectedKeyCount: 1000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create temp directory for test db
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "test-db")

			// set up db with a configuration that will trigger a GC after multiple partitions
			db, err := badger.OpenManaged(badger.DefaultOptions(path).WithNumVersionsToKeep(math.MaxInt64).
				WithLoggingLevel(badger.INFO).WithMemTableSize(int64(units.GB)))
			require.NoError(t, err)
			// set up the store
			store, err := NewStoreWithDB(db, nil, lib.NewDefaultLogger(), true)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, store.Close())
			}()

			// insert large dataset at various heights before partition boundary
			k := []byte("my_key")
			iterations := 1000
			for i := 0; i < iterations; i++ {
				// set the value in the db
				require.NoError(t, store.lss.Tombstone(k)) // tombstoned but mark keep

				// commit regularly to create multiple versions
				_, err = store.Commit()
				require.NoError(t, err)
			}
			tx := db.NewTransactionAt(uint64(iterations), true)
			// set an entry with a bit that marks it as deleted and prevents it from being discarded
			require.NoError(t, tx.SetEntry(newEntry(lib.Append([]byte(latestStatePrefix), k), nil, test.finalBit)))
			require.NoError(t, tx.CommitAt(uint64(iterations), nil))

			db.SetDiscardTs(uint64(iterations))

			require.NoError(t, FlushMemTable(db))
			require.NoError(t, db.Flatten(1))

			// use an archive iterator to iterate through the deleted keys
			iterator, err := store.lss.ArchiveIterator(nil)
			require.NoError(t, err)
			it := iterator.(*Iterator)
			defer it.Close()

			numKeys := 0
			for ; it.Valid(); it.Next() {
				numKeys++
			}

			require.Equal(t, test.expectedKeyCount, numKeys)
		})
	}
}

func TestFlushMemtable(t *testing.T) {
	// create temp directory for test db
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test-db")

	// set up db with a configuration that will trigger a GC after multiple partitions
	db, err := badger.OpenManaged(badger.DefaultOptions(path).WithNumVersionsToKeep(math.MaxInt64).
		WithLoggingLevel(badger.INFO).WithMemTableSize(int64(units.GB)))
	require.NoError(t, err)
	// set up the store
	store, err := NewStoreWithDB(db, nil, lib.NewDefaultLogger(), true)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	// insert large dataset at various heights before partition boundary
	var k []byte
	keys := [][]byte{
		bytes.Repeat([]byte("0"), 32),
		bytes.Repeat([]byte("1"), 32),
		bytes.Repeat([]byte("2"), 32),
		bytes.Repeat([]byte("3"), 32),
		bytes.Repeat([]byte("4"), 32),
		bytes.Repeat([]byte("5"), 32),
		bytes.Repeat([]byte("6"), 32),
		bytes.Repeat([]byte("7"), 32),
		bytes.Repeat([]byte("8"), 32),
		bytes.Repeat([]byte("9"), 32),
	}
	iterations := 1000
	for i := 1; i < iterations; i++ {
		// if not the first iteration
		if i != 0 {
			// delete the last key in the db
			require.NoError(t, store.Delete(k))
		}
		// use a key to be updated repeatedly to generate versions
		k = keys[i%10]

		// set the value in the db
		require.NoError(t, store.Set(k, nil))

		// commit regularly to create multiple versions
		_, err = store.Commit()
		require.NoError(t, err)
	}
	db.SetDiscardTs(uint64(iterations - 1))
	require.NoError(t, FlushMemTable(db))
	require.NoError(t, db.Flatten(32))

	// use an archive iterator to iterate through the deleted keys
	iterator, err := store.lss.ArchiveIterator(nil)
	require.NoError(t, err)
	it := iterator.(*Iterator)
	defer it.Close()

	numKeys := 0
	for ; it.Valid(); it.Next() {
		numKeys++
	}

	require.Equal(t, 1, numKeys)
}

func TestHistoricalPrefix(t *testing.T) {
	tests := []struct {
		name     string
		height   uint64
		expected []byte
	}{
		{
			name:     "zero",
			height:   0,
			expected: append([]byte(historicStatePrefix), binary.BigEndian.AppendUint64(nil, 1)...),
		},
		{
			name:     "less than partition frequency",
			height:   999,
			expected: append([]byte(historicStatePrefix), binary.BigEndian.AppendUint64(nil, 1)...),
		},
		{
			name:     "greater than partition frequency",
			height:   1001,
			expected: append([]byte(historicStatePrefix), binary.BigEndian.AppendUint64(nil, 1000)...),
		},
		{
			name:     "2x partition frequency",
			height:   2794,
			expected: append([]byte(historicStatePrefix), binary.BigEndian.AppendUint64(nil, 2000)...),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, []byte(historicalPrefix(test.height)), test.expected)
		})
	}
}

func testStore(t *testing.T) (*Store, *badger.DB, func()) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	store, err := NewStoreWithDB(db, nil, lib.NewDefaultLogger(), true)
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

func getMeta(e *badger.Item) (value byte) {
	v := reflect.ValueOf(e).Elem()
	f := v.FieldByName(badgerMetaFieldName)
	ptr := unsafe.Pointer(f.UnsafeAddr())
	return *(*byte)(ptr)
}
