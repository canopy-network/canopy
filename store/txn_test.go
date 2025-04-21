package store

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTxnWriteSetGet(t *testing.T) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	require.NoError(t, err)
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
	require.NoError(t, test.Set([]byte("1/a"), []byte("a")))
	// test get from ops before write()
	val, err := test.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), val)
	// test get from parent before write()
	val, err = parent.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Nil(t, val)
	require.NoError(t, test.Write())
	// test get from parent after write()
	val, err = parent.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), val)
	// test get from ops after write()
	val, err = test.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), val)
}

func TestTxnWriteDelete(t *testing.T) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
	require.NoError(t, test.Set([]byte("1/a"), []byte("a")))
	require.NoError(t, test.Write())
	require.NoError(t, test.Delete([]byte("1/a")))
	val, err := test.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Nil(t, val)
	val, err = parent.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), val)
	require.NoError(t, test.Write())
	val, err = parent.Get([]byte("1/a"))
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestTxnIterateNilPrefix(t *testing.T) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
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
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
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
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
	bulkSetKV(t, parent, "1/", "f", "e", "d")
	bulkSetKV(t, test, "1/", "i", "h", "g")
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
	it1.Close()
	it2, err := test.RevIterator([]byte("1"))
	require.NoError(t, err)
	for i := 0; it2.Valid(); it2.Next() {
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
	it2.Close()
}

func TestTxnIterateMixedWithDeletedValues(t *testing.T) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	test := NewTxn(parent)
	defer func() { parent.Close(); db.Close(); test.Discard() }()
	bulkSetKV(t, parent, "1/", "f", "e", "d")
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

func TestTxnIterate(t *testing.T) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	parent, err := NewStoreInMemory(lib.NewDefaultLogger())
	compare := NewTxnWrapper(db.NewTransactionAt(1, true), lib.NewDefaultLogger(), []byte(latestStatePrefix))
	test := NewTxn(parent)
	defer func() { parent.Close(); compare.Close(); db.Close(); test.Discard() }()
	require.NoError(t, test.Set([]byte("a"), []byte("a")))
	require.NoError(t, compare.Set([]byte("a"), []byte("a")))
	revIt, err := test.RevIterator([]byte(prefixEnd([]byte("a"))))
	require.NoError(t, err)
	revIt.Close()
	revIt, err = compare.RevIterator(prefixEnd([]byte("a")))
	require.NoError(t, err)
	revIt.Close()
}
