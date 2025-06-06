package store

import (
	"crypto/rand"
	"encoding/hex"
	math "math/rand"
	"slices"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func newTxn(t *testing.T, prefix []byte) (*Txn, *badger.DB, *badger.WriteBatch) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	var version uint64 = 1
	reader := db.NewTransactionAt(version, false)
	writer := db.NewWriteBatchAt(version)
	return NewBadgerTxn(reader, writer, prefix, true, lib.NewDefaultLogger()), db, writer
}

func TestNestedTxn(t *testing.T) {
	baseTxn, db, _ := newTxn(t, []byte("1/"))
	defer func() { baseTxn.Close(); db.Close(); baseTxn.Discard() }()
	// create a nested transaction
	nested := NewTxn(baseTxn, baseTxn, []byte("2/"), true, baseTxn.logger)
	// set some values in the nested transaction
	require.NoError(t, nested.Set([]byte("a"), []byte("a")))
	require.NoError(t, nested.Set([]byte("b"), []byte("b")))
	require.NoError(t, nested.Delete([]byte("a")))
	// confirm value is succesfully deleted
	val, err := nested.Get([]byte("a"))
	require.NoError(t, err)
	require.Nil(t, val)
	// confirm value is not visible in the parent transaction
	val, err = baseTxn.Get([]byte("2/b"))
	require.NoError(t, err)
	require.Nil(t, val)
	// commit the nested transaction
	require.NoError(t, nested.Write())
	// check that the changes are visible in the parent transaction
	val, err = baseTxn.Get([]byte("2/b"))
	require.NoError(t, err)
	require.Equal(t, []byte("b"), val)
}

func TestTxnWriteSetGet(t *testing.T) {
	test, db, writer := newTxn(t, []byte("1/"))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	key := []byte("a")
	value := []byte("a")
	require.NoError(t, test.Set(key, value))
	// test get from ops before write()
	val, err := test.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, val)
	// test get from reader before write()
	dbVal, dbErr := test.reader.Get(key)
	require.NoError(t, dbErr)
	require.Nil(t, dbVal)
	require.NoError(t, test.Write())
	require.NoError(t, writer.Flush())
	// test get from reader after write()
	require.Len(t, test.cache.ops, 0)
	val, err = test.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, val)
}

func TestTxnWriteDelete(t *testing.T) {
	test, db, writer := newTxn(t, []byte("1/"))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	// set value and delete in the ops
	require.NoError(t, test.Set([]byte("a"), []byte("a")))
	require.NoError(t, test.Delete([]byte("a")))
	val, err := test.Get([]byte("a"))
	require.NoError(t, err)
	require.Nil(t, val)
	// test get value from reader before write()
	dbVal, dbErr := test.reader.Get([]byte("1/a"))
	require.NoError(t, dbErr)
	require.Nil(t, dbVal)
	// test get value from reader after write()
	require.NoError(t, test.Write())
	require.NoError(t, writer.Flush())
	dbVal, dbErr = test.reader.Get([]byte("1/a"))
	require.NoError(t, dbErr)
	require.Nil(t, dbVal)
}

func TestTxnIterateNilPrefix(t *testing.T) {
	test, db, _ := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	bulkSetKV(t, test, "", "c", "a", "b")
	it1, err := test.Iterator(nil)
	require.NoError(t, err)
	for i := 0; it1.Valid(); it1.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("a"), it1.Key())
			require.Equal(t, []byte("a"), it1.Value())
		case 1:
			require.Equal(t, []byte("b"), it1.Key())
			require.Equal(t, []byte("b"), it1.Value())
		case 2:
			require.Equal(t, []byte("c"), it1.Key())
			require.Equal(t, []byte("c"), it1.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it1.Close()
	it2, err := test.RevIterator(nil)
	require.NoError(t, err)
	for i := 0; it2.Valid(); it2.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("c"), it2.Key())
			require.Equal(t, []byte("c"), it2.Value())
		case 1:
			require.Equal(t, []byte("b"), it2.Key())
			require.Equal(t, []byte("b"), it2.Value())
		case 2:
			require.Equal(t, []byte("a"), it2.Key())
			require.Equal(t, []byte("a"), it2.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it2.Close()
}

func TestTxnIterateBasic(t *testing.T) {
	test, db, _ := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	bulkSetKV(t, test, "0/", "c", "a", "b")
	bulkSetKV(t, test, "1/", "f", "d", "e")
	bulkSetKV(t, test, "2/", "i", "h", "g")
	it1, err := test.Iterator([]byte("1/"))
	require.NoError(t, err)
	for i := 0; it1.Valid(); it1.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("1/d"), it1.Key())
			require.Equal(t, []byte("d"), it1.Value())
		case 1:
			require.Equal(t, []byte("1/e"), it1.Key())
			require.Equal(t, []byte("e"), it1.Value())
		case 2:
			require.Equal(t, []byte("1/f"), it1.Key())
			require.Equal(t, []byte("f"), it1.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it1.Close()
	it2, err := test.RevIterator([]byte("2"))
	require.NoError(t, err)
	for i := 0; it2.Valid(); it2.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("2/i"), it2.Key())
			require.Equal(t, []byte("i"), it2.Value())
		case 1:
			require.Equal(t, []byte("2/h"), it2.Key())
			require.Equal(t, []byte("h"), it2.Value())
		case 2:
			require.Equal(t, []byte("2/g"), it2.Key())
			require.Equal(t, []byte("g"), it2.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it2.Close()
}

func TestTxnIterateMixed(t *testing.T) {
	test, db, writer := newTxn(t, []byte("s/"))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	// first write to the memory cache and flush it
	bulkSetKV(t, test, "1/", "f", "e", "d")
	require.NoError(t, test.Write())
	require.NoError(t, writer.Flush())
	// now add new entries to the memory cache.
	// Since the reader and writer are on the same version,
	// it's possible to read the previously flushed data
	// without creating a new transaction
	bulkSetKV(t, test, "1/", "i", "h", "g")
	// confirm that the only data in the memory cache are the last 3 entries
	require.Len(t, test.cache.ops, 3)
	it1, err := test.Iterator([]byte(""))
	require.NoError(t, err)
	var i int
	for ; it1.Valid(); it1.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("1/d"), it1.Key())
			require.Equal(t, []byte("d"), it1.Value())
		case 1:
			require.Equal(t, []byte("1/e"), it1.Key())
			require.Equal(t, []byte("e"), it1.Value())
		case 2:
			require.Equal(t, []byte("1/f"), it1.Key())
			require.Equal(t, []byte("f"), it1.Value())
		case 3:
			require.Equal(t, []byte("1/g"), it1.Key())
			require.Equal(t, []byte("g"), it1.Value())
		case 4:
			require.Equal(t, []byte("1/h"), it1.Key())
			require.Equal(t, []byte("h"), it1.Value())
		case 5:
			require.Equal(t, []byte("1/i"), it1.Key())
			require.Equal(t, []byte("i"), it1.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	require.Equal(t, 6, i, "not all iterator cases tested")
	it1.Close()
	it2, err := test.RevIterator([]byte(""))
	require.NoError(t, err)
	i = 0
	for ; it2.Valid(); it2.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("1/i"), it2.Key())
			require.Equal(t, []byte("i"), it2.Value())
		case 1:
			require.Equal(t, []byte("1/h"), it2.Key())
			require.Equal(t, []byte("h"), it2.Value())
		case 2:
			require.Equal(t, []byte("1/g"), it2.Key())
			require.Equal(t, []byte("g"), it2.Value())
		case 3:
			require.Equal(t, []byte("1/f"), it2.Key())
			require.Equal(t, []byte("f"), it2.Value())
		case 4:
			require.Equal(t, []byte("1/e"), it2.Key())
			require.Equal(t, []byte("e"), it2.Value())
		case 5:
			require.Equal(t, []byte("1/d"), it2.Key())
			require.Equal(t, []byte("d"), it2.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	require.Equal(t, 6, i, "not all reverse iterator cases tested")
	it2.Close()
}

func TestTxnIterateMixedWithDeletedValues(t *testing.T) {
	test, db, writer := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	// first write to the db writer and flush it
	bulkSetKV(t, test, "1/", "f", "e", "d")
	require.NoError(t, test.Write())
	require.NoError(t, writer.Flush())
	// now add new entries to the memory cache.
	// Since the reader and writer are on the same version,
	// it's possible to read the previously flushed data
	// without creating a new transaction
	// add the values to the memory cache
	bulkSetKV(t, test, "1/", "h", "g", "f")
	require.NoError(t, test.Delete([]byte("1/f"))) // shared and shadowed
	require.NoError(t, test.Delete([]byte("1/d"))) // first
	require.NoError(t, test.Delete([]byte("1/h"))) // last
	it1, err := test.Iterator([]byte("1/"))
	require.NoError(t, err)
	for i := 0; it1.Valid(); it1.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("1/e"), it1.Key())
			require.Equal(t, []byte("e"), it1.Value())
		case 1:
			require.Equal(t, []byte("1/g"), it1.Key())
			require.Equal(t, []byte("g"), it1.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it1.Close()
	it2, err := test.RevIterator([]byte("1"))
	require.NoError(t, err)
	for i := 0; it2.Valid(); it2.Next() {
		switch i {
		case 0:
			require.Equal(t, []byte("1/g"), it2.Key())
			require.Equal(t, []byte("g"), it2.Value())
		case 1:
			require.Equal(t, []byte("1/e"), it2.Key())
			require.Equal(t, []byte("e"), it2.Value())
		default:
			t.Fatal("too many iterations")
		}
		i++
	}
	it2.Close()
}

func TestIteratorBasic(t *testing.T) {
	test, db, writer := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	expectedVals := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	expectedValsReverse := []string{"h", "g", "f", "e", "d", "c", "b", "a"}
	bulkSetKV(t, test, "", expectedVals...)
	require.NoError(t, test.Write())
	require.NoError(t, writer.Flush())
	it, err := test.Iterator(nil)
	require.NoError(t, err)
	defer it.Close()
	validateIterators(t, expectedVals, it)
	rIt, err := test.RevIterator(nil)
	require.NoError(t, err)
	defer rIt.Close()
	validateIterators(t, expectedValsReverse, rIt)
}

func TestIteratorWithDelete(t *testing.T) {
	expectedVals := []string{"a", "b", "c", "d", "e", "f", "g"}
	test, db, _ := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	bulkSetKV(t, test, "", expectedVals...)
	for range 10 {
		randomindex := math.Intn(len(expectedVals))
		require.NoError(t, test.Delete([]byte(expectedVals[randomindex])))
		expectedVals = slices.Delete(expectedVals, randomindex, randomindex+1)
		cIt, err := test.Iterator(nil)
		require.NoError(t, err)
		validateIterators(t, expectedVals, cIt)
		cIt.Close()
		add := make([]byte, 1)
		_, er := rand.Read(add)
		require.NoError(t, er)
		expectedVals = append(expectedVals, hex.EncodeToString(add))
	}
}

func TestTxnIterateWithDeleteDuringIteration(t *testing.T) {
	test, db, _ := newTxn(t, []byte(""))
	defer func() { test.Close(); db.Close(); test.Discard() }()
	// set up initial data
	bulkSetKV(t, test, "", "a", "b", "c", "d", "e")
	// get an iterator
	it, err := test.Iterator(nil)
	require.NoError(t, err)
	defer it.Close()
	// track seen keys
	seen := make(map[string]int)
	// first iteration to get some keys
	keysToDelete := make([][]byte, 0)
	count := 0
	for ; it.Valid() && count < 3; it.Next() {
		key := string(it.Key())
		seen[key]++
		keysToDelete = append(keysToDelete, []byte(key))
		count++
	}
	// delete those keys
	for _, key := range keysToDelete {
		require.NoError(t, test.Delete(key))
	}
	// Continue iterating - shouldn't see deleted keys again
	for ; it.Valid(); it.Next() {
		key := string(it.Key())
		seen[key]++
		// Each key should be seen exactly once
		require.Equal(t, 1, seen[key], "key %s was seen multiple times", key)
	}
	// Verify all keys were seen exactly once
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		require.Equal(t, 1, seen[k], "key %s was not iterated or was seen multiple times", k)
	}
}
