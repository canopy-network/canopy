package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestMaybeBackup(t *testing.T) {
	// use a single base dir so db and backup share the same filesystem, making
	// os.Rename atomic across both paths
	baseDir := t.TempDir()
	dbDir := filepath.Join(baseDir, "db")
	backupDir := filepath.Join(baseDir, "backup")
	// set up chain configs
	config := lib.DefaultConfig()
	config.StoreConfig.BackupInterval = 3
	config.StoreConfig.BackupDirectory = backupDir
	// set up new store with the configs above
	st, e := NewStore(config, dbDir, nil, lib.NewDefaultLogger())
	require.NoError(t, e)
	s := st.(*Store)
	// write a key and commit 3 blocks to reach the backup trigger;
	// Commit() calls MaybeBackup() internally so no explicit call is needed
	key := lib.JoinLenPrefix([]byte("state/"), []byte("value"))
	for i := 1; i <= 3; i++ {
		require.NoError(t, s.Set(key, fmt.Appendf(nil, "v%d", i)))
		_, err := s.Commit()
		require.NoError(t, err)
	}
	require.EqualValues(t, 3, s.Version())
	// wait for the backup goroutine started by Commit() to complete
	require.Eventually(t, func() bool { return !s.backup.Load() },
		1*time.Second, 50*time.Millisecond)
	// backup directory must exist
	_, err := os.Stat(backupDir)
	require.NoError(t, err)
	// height file must reflect the block at which the backup was taken
	heightData, err := os.ReadFile(filepath.Join(backupDir, "height.txt"))
	require.NoError(t, err)
	require.Equal(t, "3", string(heightData))
	// advance more blocks so the live DB diverges from the backup
	for i := 4; i <= 5; i++ {
		require.NoError(t, s.Set(key, fmt.Appendf(nil, "v%d", i)))
		_, err = s.Commit()
		require.NoError(t, err)
	}
	require.EqualValues(t, 5, s.Version())
	require.NoError(t, s.Close())
	// simulate a catastrophic loss of the live DB directory
	require.NoError(t, os.RemoveAll(dbDir))
	// promote the backup to the DB path
	require.NoError(t, os.Rename(backupDir, dbDir))
	// reopen from the backup location — should restore to height 3
	restored, e := NewStore(config, dbDir, nil, lib.NewDefaultLogger())
	require.NoError(t, e)
	defer restored.Close()
	restoredStore := restored.(*Store)
	require.EqualValues(t, 3, restoredStore.Version())
	// value must be what was committed at block 3, not the later diverged state
	restoredVal, err := restored.Get(key)
	require.NoError(t, err)
	require.Equal(t, []byte("v3"), restoredVal)
}

// TestCopySeededNodeCacheReducesMisses verifies that a store Copy() reference-shares the
// parent's SMT nodeCache as a read-only seed, so Root() on the copy serves tree-node reads
// from the in-memory seed instead of Pebble.
func TestCopySeededNodeCacheReducesMisses(t *testing.T) {
	parent, _, cleanup := testStore(t)
	defer cleanup()

	// write several keys and commit so the parent's SMT nodeCache gets warmed
	key1 := lib.JoinLenPrefix([]byte("acct/"), []byte("alice"))
	key2 := lib.JoinLenPrefix([]byte("acct/"), []byte("bob"))
	key3 := lib.JoinLenPrefix([]byte("acct/"), []byte("carol"))
	require.NoError(t, parent.Set(key1, []byte("100")))
	require.NoError(t, parent.Set(key2, []byte("200")))
	require.NoError(t, parent.Set(key3, []byte("300")))
	// Commit() persists to Pebble (cold-comparison copy below needs disk-resident state),
	// then a follow-up Set+Root warms parent.sc.nodeCache by traversing the on-disk tree
	// to insert a new key. (Root() alone after Commit has no ops to process.)
	_, err := parent.Commit()
	require.NoError(t, err)
	warmKey := lib.JoinLenPrefix([]byte("acct/"), []byte("warm"))
	require.NoError(t, parent.Set(warmKey, []byte("0")))
	_, errR := parent.Root()
	require.NoError(t, errR)
	require.True(t, parent.IsRootCached())
	warmCacheSize := len(parent.sc.nodeCache)
	require.Positive(t, warmCacheSize, "parent nodeCache should be populated after Set+Root")

	// Copy() should reference-share the parent's nodeCache via seedCache.
	// crucially: it must be the SAME map (pointer equality), not a copy.
	copyStore, errI := parent.Copy()
	require.NoError(t, errI)
	cs := copyStore.(*Store)
	require.NotNil(t, cs.seedCache, "copy should carry a seed reference")
	require.Len(t, cs.seedCache, warmCacheSize, "seed should be the parent's full nodeCache")
	// prove true reference-sharing (no copy): mutating the parent's nodeCache must be
	// visible through the seed. would fail if Copy() allocated a new map.
	parent.sc.nodeCache["__test_sentinel__"] = nil
	_, ok := cs.seedCache["__test_sentinel__"]
	require.True(t, ok, "mutation on parent's nodeCache must be visible through shared seed reference")
	delete(parent.sc.nodeCache, "__test_sentinel__")

	// apply a new write on the copy (simulates mempool apply_block)
	key4 := lib.JoinLenPrefix([]byte("acct/"), []byte("dave"))
	require.NoError(t, cs.Set(key4, []byte("400")))

	// Root() on the copy should move seedCache onto the ephemeral SMT as parentNodeCache
	_, errI = cs.Root()
	require.NoError(t, errI)
	require.Nil(t, cs.seedCache, "Store.seedCache should be released after Root()")
	require.NotNil(t, cs.sc.parentNodeCache, "SMT.parentNodeCache should hold the seed after Root()")

	// cache hits should be non-zero (seeded nodes were reused via parentNodeCache fallback)
	require.Positive(t, cs.sc.stats.NodeCacheHits,
		"parentNodeCache fallback should produce cache hits during Root()")

	totalReads := cs.sc.stats.NodeCacheHits + cs.sc.stats.NodeCacheMisses
	missRate := float64(cs.sc.stats.NodeCacheMisses) / float64(totalReads)
	t.Logf("seeded copy: reads=%d hits=%d misses=%d miss_rate=%.1f%%",
		totalReads, cs.sc.stats.NodeCacheHits, cs.sc.stats.NodeCacheMisses, missRate*100)
	require.Less(t, missRate, 0.5, "seeded copy miss rate should be below 50%%")

	// cold copy (no seed) for comparison: commit so Pebble has tree nodes, but skip the
	// post-commit Root() — sc stays nil so Copy() captures no seed and the ephemeral
	// must read everything from disk.
	parent2, _, cleanup2 := testStore(t)
	defer cleanup2()
	require.NoError(t, parent2.Set(key1, []byte("100")))
	require.NoError(t, parent2.Set(key2, []byte("200")))
	require.NoError(t, parent2.Set(key3, []byte("300")))
	_, err2 := parent2.Commit()
	require.NoError(t, err2)
	require.False(t, parent2.IsRootCached(), "skip Root() so the cold copy gets no seed")
	coldCopy, errI := parent2.Copy()
	require.NoError(t, errI)
	cc := coldCopy.(*Store)
	require.Nil(t, cc.seedCache, "cold copy (parent had no Commit) should have no seed")
	require.NoError(t, cc.Set(key4, []byte("400")))
	_, errI = cc.Root()
	require.NoError(t, errI)
	require.Nil(t, cc.sc.parentNodeCache, "cold copy should not have a parentNodeCache")
	coldMissRate := float64(cc.sc.stats.NodeCacheMisses) / float64(cc.sc.stats.NodeCacheHits+cc.sc.stats.NodeCacheMisses)
	t.Logf("cold copy:   reads=%d hits=%d misses=%d miss_rate=%.1f%%",
		cc.sc.stats.NodeCacheHits+cc.sc.stats.NodeCacheMisses,
		cc.sc.stats.NodeCacheHits, cc.sc.stats.NodeCacheMisses, coldMissRate*100)
	require.Greater(t, coldMissRate, missRate,
		"cold copy should have a higher miss rate than the seeded copy")
}

// TestCopySeedSurvivesMaxCacheSizeWipe verifies that the MaxCacheSize wipe in setNode()
// only clears the ephemeral SMT's own nodeCache and leaves the parentNodeCache seed intact.
// This is the PERF-1 fix: with the prior design (seed copied into nodeCache) the wipe
// would erase the entire seed on the first setNode call when the cache was near MaxCacheSize.
func TestCopySeedSurvivesMaxCacheSizeWipe(t *testing.T) {
	parent, _, cleanup := testStore(t)
	defer cleanup()
	// warm parent SMT (Root, not Commit — Commit resets sc and erases the warm cache).
	key1 := lib.JoinLenPrefix([]byte("acct/"), []byte("alice"))
	require.NoError(t, parent.Set(key1, []byte("100")))
	_, err := parent.Root()
	require.NoError(t, err)

	copyStore, errI := parent.Copy()
	require.NoError(t, errI)
	cs := copyStore.(*Store)
	require.NoError(t, cs.Set(lib.JoinLenPrefix([]byte("acct/"), []byte("bob")), []byte("200")))
	_, errI = cs.Root()
	require.NoError(t, errI)
	require.NotNil(t, cs.sc.parentNodeCache, "seed must be present after Root()")

	// simulate the MaxCacheSize wipe path: force setNode() to reset the local cache
	cs.sc.nodeCache = make(map[string]*node)
	// the seed must still be intact and queryable
	require.NotEmpty(t, cs.sc.parentNodeCache, "parentNodeCache must survive a local cache wipe")

	// pick any key from the seed and verify getNode finds it via the seed fallback
	var sampleKey string
	for k := range cs.sc.parentNodeCache {
		sampleKey = k
		break
	}
	require.NotEmpty(t, sampleKey, "seed must contain at least one node")
	statsBefore := cs.sc.stats.NodeCacheHits
	n, errI := cs.sc.getNode([]byte(sampleKey))
	require.NoError(t, errI)
	require.NotNil(t, n)
	require.Equal(t, statsBefore+1, cs.sc.stats.NodeCacheHits,
		"getNode must register a cache hit served from parentNodeCache")
	require.Equal(t, 1, cs.sc.stats.NodeCacheHitsSeed,
		"seed-served hits must be counted separately for observability")
}

// TestDelNodeShadowsParentSeed proves the CORRECT-1 fix: a delNode against a key that
// lives only in the inherited parentNodeCache must shadow the seed so subsequent
// getNode calls within the same batch see the deletion, not the stale parent value.
func TestDelNodeShadowsParentSeed(t *testing.T) {
	parent, _, cleanup := testStore(t)
	defer cleanup()
	// warm parent SMT with one key. Root() (not Commit) keeps parent.sc populated.
	k := lib.JoinLenPrefix([]byte("acct/"), []byte("alice"))
	require.NoError(t, parent.Set(k, []byte("100")))
	_, err := parent.Root()
	require.NoError(t, err)
	require.Positive(t, len(parent.sc.nodeCache))

	copyStore, errI := parent.Copy()
	require.NoError(t, errI)
	cs := copyStore.(*Store)
	// force Root() so the SMT picks up parentNodeCache from seedCache
	require.NoError(t, cs.Set(lib.JoinLenPrefix([]byte("acct/"), []byte("bob")), []byte("200")))
	_, errI = cs.Root()
	require.NoError(t, errI)
	require.NotNil(t, cs.sc.parentNodeCache)

	// pick any key the parent put in the seed (we want to test the delete-shadow path on it).
	var seedKey string
	for kk := range cs.sc.parentNodeCache {
		seedKey = kk
		break
	}
	require.NotEmpty(t, seedKey)
	// sanity: getNode returns the seed node before deletion.
	pre, errPre := cs.sc.getNode([]byte(seedKey))
	require.NoError(t, errPre)
	require.NotNil(t, pre, "pre-delete read should hit the seed")

	// delete and verify subsequent getNode does NOT return the stale seed entry.
	require.NoError(t, cs.sc.delNode([]byte(seedKey)))
	n, err2 := cs.sc.getNode([]byte(seedKey))
	require.NoError(t, err2)
	require.Nil(t, n, "delNode must shadow the parentNodeCache entry — getNode should return nil after deletion")

	// the tombstone itself should NOT count as a seed hit on the metric.
	hitsSeedBefore := cs.sc.stats.NodeCacheHitsSeed
	_, _ = cs.sc.getNode([]byte(seedKey))
	require.Equal(t, hitsSeedBefore, cs.sc.stats.NodeCacheHitsSeed,
		"tombstone reads must not increment NodeCacheHitsSeed (no fall-through to seed)")
}

// TestCopySeedConcurrentParentActivity exercises the design's central concurrency claim:
// the parent's nodeCache map captured at Copy() time must remain safe to read from the
// ephemeral copy WITHOUT any external lock, even while the parent continues committing
// new blocks. Production reality: an external mutex (Mempool.L in the controller)
// serializes Copy and Commit on the parent — but the ephemeral's subsequent Root()
// (which reads parentNodeCache) runs OUTSIDE that lock. The claim is that this is safe
// because each parent Commit allocates a fresh sc.nodeCache map; the OLD map referenced
// by the ephemeral becomes orphaned in the parent and is never written to again.
// Run under -race; the race detector catches any violation.
func TestCopySeedConcurrentParentActivity(t *testing.T) {
	parent, _, cleanup := testStore(t)
	defer cleanup()
	require.NoError(t, parent.Set(lib.JoinLenPrefix([]byte("seed/"), []byte("k0")), []byte("v0")))
	_, err := parent.Commit()
	require.NoError(t, err)

	// extLock simulates the controller's Mempool.L — it serializes parent.Commit and
	// parent.Copy, but ephemeral operations on the COPY run lock-free.
	var extLock sync.Mutex
	const iterations = 30
	var stop atomic.Bool
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; !stop.Load() && i < iterations; i++ {
			extLock.Lock()
			key := lib.JoinLenPrefix([]byte("parent/"), []byte(fmt.Sprintf("k%d", i)))
			if err := parent.Set(key, []byte(fmt.Sprintf("v%d", i))); err != nil {
				extLock.Unlock()
				return
			}
			if _, err := parent.Commit(); err != nil {
				extLock.Unlock()
				return
			}
			extLock.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; !stop.Load() && i < iterations; i++ {
			extLock.Lock()
			cp, errI := parent.Copy()
			extLock.Unlock()
			if errI != nil {
				return
			}
			// Run ephemeral Root() OUTSIDE the external lock. This is the actual
			// production pattern: the heavy CheckMempool/Root work runs without
			// blocking parent commits.
			cs := cp.(*Store)
			key := lib.JoinLenPrefix([]byte("ephem/"), []byte(fmt.Sprintf("k%d", i)))
			if err := cs.Set(key, []byte("x")); err != nil {
				cs.Discard()
				return
			}
			if _, errI := cs.Root(); errI != nil {
				cs.Discard()
				return
			}
			cs.Discard()
		}
	}()

	wg.Wait()
	stop.Store(true)
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
