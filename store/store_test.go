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

func TestRollback(t *testing.T) {
	st, db, cleanup := testStore(t)
	defer cleanup()
	key := lib.JoinLenPrefix([]byte("state/"), []byte("balance"))

	require.NoError(t, st.Set(key, []byte("v1")))
	_, err := st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 1, st.Version())

	require.NoError(t, st.Set(key, []byte("v2")))
	_, err = st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 2, st.Version())

	require.NoError(t, st.Set(key, []byte("v3")))
	_, err = st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 3, st.Version())

	require.NoError(t, st.Rollback(1))
	require.EqualValues(t, 1, st.Version())

	value, err := st.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte("v1"), value)

	// Querying a higher historical version after rollback should not leak old future state.
	rolledBackReadOnly, err := st.NewReadOnly(2)
	require.NoError(t, err)
	defer rolledBackReadOnly.Discard()
	value, err = rolledBackReadOnly.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte("v1"), value)

	// Re-opening from DB should restore the rolled back height from the latest commit pointer.
	reopened, err := NewStoreWithDB(lib.DefaultConfig(), db, nil, lib.NewDefaultLogger())
	require.NoError(t, err)
	defer reopened.Discard()
	require.EqualValues(t, 1, reopened.Version())
	value, err = reopened.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte("v1"), value)
}

func TestRollbackSelectiveStateRestore(t *testing.T) {
	st, _, cleanup := testStore(t)
	defer cleanup()

	stableKey := lib.JoinLenPrefix([]byte("state/"), []byte("stable"))
	revertedKey := lib.JoinLenPrefix([]byte("state/"), []byte("reverted"))
	futureKey := lib.JoinLenPrefix([]byte("state/"), []byte("future"))

	require.NoError(t, st.Set(stableKey, []byte("stable-v1")))
	require.NoError(t, st.Set(revertedKey, []byte("reverted-v1")))
	_, err := st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 1, st.Version())

	require.NoError(t, st.Set(revertedKey, []byte("reverted-v2")))
	require.NoError(t, st.Set(futureKey, []byte("future-v2")))
	_, err = st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 2, st.Version())

	require.NoError(t, st.Set(revertedKey, []byte("reverted-v3")))
	require.NoError(t, st.Delete(futureKey))
	_, err = st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 3, st.Version())

	require.NoError(t, st.Rollback(1))
	require.EqualValues(t, 1, st.Version())

	value, err := st.Get(stableKey)
	require.NoError(t, err)
	require.Equal(t, []byte("stable-v1"), value)

	value, err = st.Get(revertedKey)
	require.NoError(t, err)
	require.Equal(t, []byte("reverted-v1"), value)

	value, err = st.Get(futureKey)
	require.NoError(t, err)
	require.Nil(t, value)

	// Rollback must remove future-only keys from latest-state iteration, not just return nil on Get().
	statePrefix := lib.JoinLenPrefix([]byte("state/"))
	it, err := st.Iterator(statePrefix)
	require.NoError(t, err)
	defer it.Close()
	seen := make(map[string]struct{})
	for ; it.Valid(); it.Next() {
		seen[string(it.Key())] = struct{}{}
	}
	require.Len(t, seen, 2)
	require.Contains(t, seen, string(stableKey))
	require.Contains(t, seen, string(revertedKey))
	require.NotContains(t, seen, string(futureKey))
}

func TestGetDoesNotMatchPrefixSuperset(t *testing.T) {
	st, _, cleanup := testStore(t)
	defer cleanup()

	parentKey := lib.JoinLenPrefix([]byte("a"))
	childKey := lib.JoinLenPrefix([]byte("a"), []byte("b"))

	require.NoError(t, st.Set(childKey, []byte("child-v1")))
	_, err := st.Commit()
	require.NoError(t, err)

	value, err := st.Get(parentKey)
	require.NoError(t, err)
	require.Nil(t, value)

	value, err = st.Get(childKey)
	require.NoError(t, err)
	require.Equal(t, []byte("child-v1"), value)
}

func TestRollbackWithPrefixOverlappingKeys(t *testing.T) {
	st, _, cleanup := testStore(t)
	defer cleanup()

	parentKey := lib.JoinLenPrefix([]byte("a"))
	childKey := lib.JoinLenPrefix([]byte("a"), []byte("b"))

	require.NoError(t, st.Set(childKey, []byte("child-v1")))
	_, err := st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 1, st.Version())

	require.NoError(t, st.Set(parentKey, []byte("parent-v2")))
	_, err = st.Commit()
	require.NoError(t, err)
	require.EqualValues(t, 2, st.Version())

	require.NoError(t, st.Rollback(1))
	require.EqualValues(t, 1, st.Version())

	value, err := st.Get(parentKey)
	require.NoError(t, err)
	require.Nil(t, value)

	value, err = st.Get(childKey)
	require.NoError(t, err)
	require.Equal(t, []byte("child-v1"), value)
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
