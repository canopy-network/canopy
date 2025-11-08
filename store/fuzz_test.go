package store

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/stretchr/testify/require"
)

type TestingOp int

const (
	SetTesting TestingOp = iota
	DelTesting
	GetTesting
	IterateTesting
	WriteTesting
	CommitTesting
)

// for local testing, change to a fixed value
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// var rng = rand.New(rand.NewSource(10))

func TestFuzz(t *testing.T) {
	fs := vfs.NewMem()
	db, err := pebble.Open("", &pebble.Options{
		DisableWAL:            false,
		FS:                    fs,
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
		MaxOpenFiles:          1000,
		FormatMajorVersion:    pebble.FormatNewest,
	})
	require.NoError(t, err)
	// make a writable reader that reads from the last height
	versionedStore := NewVersionedStore(db.NewSnapshot(), db.NewBatch(), 1)
	require.NoError(t, err)
	store, _, cleanup := testStore(t)
	defer cleanup()
	defer db.Close()
	keys := make([]string, 0)
	compareStore := NewTxn(versionedStore, versionedStore, []byte(latestStatePrefix), false, true, 1)
	for range 1000 {
		doRandomOperation(t, store, compareStore, &keys)
	}
}

func TestFuzzTxn(t *testing.T) {
	fs := vfs.NewMem()
	db, err := pebble.Open("", &pebble.Options{
		DisableWAL:            false,
		FS:                    fs,
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
		MaxOpenFiles:          1000,
		FormatMajorVersion:    pebble.FormatNewest,
	})
	require.NoError(t, err)
	// make a writable reader that reads from the last height
	versionedStore := NewVersionedStore(db.NewSnapshot(), db.NewBatch(), 1)
	require.NoError(t, err)
	store, err := NewStoreInMemory(lib.NewDefaultLogger())
	keys := make([]string, 0)
	compareStore := NewTxn(versionedStore, versionedStore, []byte(latestStatePrefix), false, true, 1)
	for range 1000 {
		doRandomOperation(t, store, compareStore, &keys)
	}
	db.Close()
}

func doRandomOperation(t *testing.T, db lib.RWStoreI, compare lib.RWStoreI, keys *[]string) {
	k, v := getRandomBytes(t, rng.Intn(4)+1), getRandomBytes(t, 3)
	switch getRandomOperation(t) {
	case SetTesting:
		testDBSet(t, db, k, v)
		testDBSet(t, compare, k, v)
		*keys = append(*keys, string(k))
		sort.Strings(*keys)
		*keys = deDuplicate(*keys)
	case DelTesting:
		k = randomTestKey(t, k, *keys)
		testDBDelete(t, db, k)
		testDBDelete(t, compare, k)
	case GetTesting:
		k = randomTestKey(t, k, *keys)
		v1, v2 := testDBGet(t, db, k), testDBGet(t, compare, k)
		if !bytes.Equal(v1, v2) {
			fmt.Printf("key=%s db.Get=%s compare.Get=%s\n", k, v1, v2)
		}
		require.Equalf(t, v1, v2, "key=%s db.Get=%s compare.Get=%s", k, v1, v2)
	case IterateTesting:
		testCompareIterators(t, db, compare, *keys)
	case WriteTesting:
		if x, ok := db.(TxnWriterI); ok {
			switch rng.Intn(10) {
			case 0:
				require.NoError(t, x.Commit())
			}
		}
	case CommitTesting:
		if x, ok := db.(lib.StoreI); ok {
			_, err := x.Commit()
			require.NoError(t, err)
		}
	default:
		t.Fatal("invalid op")
	}
}

func deDuplicate(s []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, i := range s {
		if _, value := allKeys[i]; !value {
			allKeys[i] = true
			list = append(list, i)
		}
	}
	return list
}

func getRandomBytes(t *testing.T, n int) []byte {
	bz := make([]byte, n)
	if _, err := rng.Read(bz); err != nil {
		t.Fatal(err)
	}
	return bz
}

func getRandomOperation(_ *testing.T) TestingOp {
	return TestingOp(rng.Intn(6))
}

func randomTestKey(_ *testing.T, k []byte, keys []string) []byte {
	if len(keys) != 0 && rng.Intn(100) < 85 {
		// 85% of time use key already found
		// else default to the random value
		k = []byte(keys[rng.Intn(len(keys))])
	}
	return k
}

func testDBSet(t *testing.T, db lib.WStoreI, k, v []byte) {
	require.NoError(t, db.Set(lib.JoinLenPrefix(k), v))
}

func testDBDelete(t *testing.T, db lib.WStoreI, k []byte) {
	require.NoError(t, db.Delete(lib.JoinLenPrefix(k)))
}

func testDBGet(t *testing.T, db lib.RWStoreI, k []byte) (value []byte) {
	value, err := db.Get(lib.JoinLenPrefix(k))
	require.NoError(t, err)
	return
}

func testCompareIterators(t *testing.T, db lib.RWStoreI, compare lib.RWStoreI, keys []string) {
	var (
		it1, it2 lib.IteratorI
		err      error
	)
	isReverse := rng.Intn(2)
	prefix := lib.JoinLenPrefix(getRandomBytes(t, rng.Intn(4)))
	require.NoError(t, err)
	switch isReverse {
	case 0:
		it1, err = db.Iterator(prefix)
		require.NoError(t, err)
		it2, err = compare.Iterator(prefix)
		require.NoError(t, err)
	case 1:
		it1, err = db.RevIterator(prefix)
		require.NoError(t, err)
		it2, err = compare.RevIterator(prefix)
		require.NoError(t, err)
	}
	defer func() { it1.Close(); it2.Close() }()
	for i := 0; func() bool { return it1.Valid() || it2.Valid() }(); func() { it1.Next(); it2.Next() }() {
		i++
		require.Equal(t, it1.Valid(), it2.Valid(), fmt.Sprintf("it1.valid=%t\ncompare.valid=%t\nisReverse=%d\nprefix=%s\n", it1.Valid(), it2.Valid(), isReverse, prefix))
		require.Equal(t, it1.Key(), it2.Key(), fmt.Sprintf("it1.key=%s\ncompare.key=%s\nisReverse=%d\nprefix=%s\n", it1.Key(), it2.Key(), isReverse, prefix))
		require.Equal(t, it1.Value(), it2.Value(), fmt.Sprintf("it1.value=%s\ncompare.value=%s\nisReverse=%d\nprefix=%s\n", it1.Value(), it2.Value(), isReverse, prefix))
	}
}
