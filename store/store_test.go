package store

import (
	"fmt"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/stretchr/testify/require"
)

func TestStoreSetGetDelete(t *testing.T) {
	store, _, _ := testStore(t)
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

func testStore(t *testing.T) (*Store, *pebble.DB, func()) {
	fs := vfs.NewMem()
	db, err := pebble.Open("", &pebble.Options{
		DisableWAL:            false,
		FS:                    fs,
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
		MaxOpenFiles:          1000,
		FormatMajorVersion:    pebble.FormatNewest,
	})
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
