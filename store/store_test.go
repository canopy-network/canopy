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
	key, val := lib.JoinLenPrefix([]byte("key")), []byte("val")
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
	lengthPrefix := lib.JoinLenPrefix([]byte(prefix))
	expectedKeys := []string{"a", "b", "c", "d", "e", "f", "g", "i", "j"}
	expectedKeysReverse := []string{"j", "i", "g", "f", "e", "d", "c", "b", "a"}
	bulkSetPrefixedKV(t, parent, prefix, "a", "c", "e", "g")
	_, err := parent.Commit()
	require.NoError(t, err)
	bulkSetPrefixedKV(t, parent, prefix, "b", "d", "f", "h", "i", "j")
	require.NoError(t, parent.Delete(lib.JoinLenPrefix([]byte(prefix), []byte("h"))))
	// forward - ensure cache iterator matches behavior of normal iterator
	cIt, err := parent.Iterator(lengthPrefix)
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix), expectedKeys, cIt)
	cIt.Close()
	// backward - ensure cache iterator matches behavior of normal iterator
	rIt, err := parent.RevIterator(lengthPrefix)
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix), expectedKeysReverse, rIt)
	rIt.Close()
}

func TestIteratorCommitAndPrefixed(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	prefix := "test/"
	lengthPrefix := lib.JoinLenPrefix([]byte(prefix))
	prefix2 := "test2/"
	lengthPrefix2 := lib.JoinLenPrefix([]byte(prefix2))
	bulkSetPrefixedKV(t, store, prefix, "a", "b", "c")
	bulkSetPrefixedKV(t, store, prefix2, "c", "d", "e")
	it, err := store.Iterator([]byte(lengthPrefix))
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix), []string{"a", "b", "c"}, it)
	it.Close()
	it2, err := store.Iterator(lengthPrefix2)
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix2), []string{"c", "d", "e"}, it2)
	it2.Close()
	root1, err := store.Commit()
	require.NoError(t, err)
	it3, err := store.RevIterator(lengthPrefix)
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix), []string{"c", "b", "a"}, it3)
	it3.Close()
	root2, err := store.Commit()
	require.NoError(t, err)
	require.Equal(t, root1, root2)
	it4, err := store.RevIterator(lengthPrefix2)
	require.NoError(t, err)
	validateIterators(t, string(lengthPrefix2), []string{"e", "d", "c"}, it4)
	it4.Close()
}

func TestDoublyNestedTxn(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	// set initial value to the store
	baseKey := lib.JoinLenPrefix([]byte("base"))
	nestedKey := lib.JoinLenPrefix([]byte("nested"))
	doublyNestedKey := lib.JoinLenPrefix([]byte("doublyNested"))
	store.Set(baseKey, baseKey)
	// create a nested transaction
	nested := store.NewTxn()
	// set nested value
	nested.Set(nestedKey, nestedKey)
	// retrieve parent key
	value, err := nested.Get(baseKey)
	require.NoError(t, err)
	require.Equal(t, baseKey, value)
	// create a doubly nested transaction
	doublyNested := nested.NewTxn()
	// set doubly nested value
	doublyNested.Set(doublyNestedKey, doublyNestedKey)
	// commit doubly nested transaction
	err = doublyNested.Flush()
	// retrieve grandparent key
	value, err = doublyNested.Get(baseKey)
	require.NoError(t, err)
	require.Equal(t, baseKey, value)
	require.NoError(t, err)
	// verify value can be retrieved from nested the store but
	// not from the store itself
	value, err = nested.Get(doublyNestedKey)
	require.NoError(t, err)
	require.Equal(t, doublyNestedKey, value)
	value, err = store.Get(doublyNestedKey)
	require.NoError(t, err)
	require.Nil(t, value)
	// commit nested transaction
	err = nested.Flush()
	require.NoError(t, err)
	// verify both nested and doubly nested values can be retrieved from the store
	value, err = store.Get(nestedKey)
	require.NoError(t, err)
	require.Equal(t, nestedKey, value)
	value, err = store.Get(doublyNestedKey)
	require.NoError(t, err)
	require.Equal(t, doublyNestedKey, value)
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

func validateIterators(t *testing.T, prefix string, expectedKeys []string, iterators ...lib.IteratorI) {
	for _, it := range iterators {
		for i := 0; it.Valid(); func() { i++; it.Next() }() {
			got, wanted := string(it.Key()), prefix+string(lib.JoinLenPrefix([]byte(expectedKeys[i])))
			require.Equal(t, wanted, got, fmt.Sprintf("wanted %s got %s", wanted, got))
		}
	}
}

// bulkSetPrefixedKV sets multiple single segment length prefixed key-value pairs in the store
func bulkSetPrefixedKV(t *testing.T, store lib.WStoreI, prefix string, keyValue ...string) {
	for _, kv := range keyValue {
		if len(prefix) > 0 {
			require.NoError(t, store.Set(lib.JoinLenPrefix([]byte(prefix), []byte(kv)), []byte(kv)))
		} else {
			require.NoError(t, store.Set(lib.JoinLenPrefix([]byte(kv)), []byte(kv)))
		}
	}
}

func bulkSetKV(t *testing.T, store lib.WStoreI, keyValue ...[]byte) {
	for _, kv := range keyValue {
		require.NoError(t, store.Set(kv, kv))
	}
}
