package controller

import (
	"sync"

	"github.com/canopy-network/canopy/fsm"
)

// defaultAccountDeltaCacheEntries bounds the in-memory tip cache. It only needs to
// cover the handful of most-recent heights an indexer polling live blocks would
// realistically request before falling back to on-demand replay.
const defaultAccountDeltaCacheEntries = 16

// accountDeltaEntry is one height's classified account changes, produced either by
// the live AccountChangeCollector attached during commit, or by an on-demand replay.
//
// Once put() into the cache, an entry and everything it points at is IMMUTABLE and
// shared by every reader — the cache's mutex protects the map, not the entry contents.
// Callers must never mutate a retrieved entry, its slices, or the *fsm.AccountChangeEntry
// values inside them. Build a fresh entry instead.
type accountDeltaEntry struct {
	added, changed, removed []*fsm.AccountChangeEntry
}

// accountDeltaCache is a small fixed-capacity FIFO cache of recent heights' account
// deltas. It is a pure accelerator with no correctness dependency: a reorged-out
// height's cached entry is simply never requested again, so no invalidation logic
// is needed.
type accountDeltaCache struct {
	mu         sync.RWMutex
	maxEntries int
	entries    map[uint64]*accountDeltaEntry
	order      []uint64
}

// newAccountDeltaCache() creates a FIFO cache holding at most maxEntries heights
// (a non-positive maxEntries falls back to defaultAccountDeltaCacheEntries)
func newAccountDeltaCache(maxEntries int) *accountDeltaCache {
	if maxEntries <= 0 {
		maxEntries = defaultAccountDeltaCacheEntries
	}
	return &accountDeltaCache{
		maxEntries: maxEntries,
		entries:    make(map[uint64]*accountDeltaEntry),
		order:      make([]uint64, 0, maxEntries),
	}
}

// get() returns the cached delta for a height, if present.
//
// READ-ONLY: the returned entry is the live cached object, not a copy, and is shared with
// every other reader. Do not mutate it, do not append to added/changed/removed (an append
// to a full slice reallocates but an append to a slice with spare capacity would write
// through into the cached data), and do not modify the *fsm.AccountChangeEntry values it
// points at. Copy first if a caller needs to transform the results.
func (c *accountDeltaCache) get(height uint64) (*accountDeltaEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[height]
	return entry, ok
}

// SeedAccountDeltaCache() installs a block height's classified account changes into the tip
// cache directly, bypassing a live commit. It exists so callers outside this package — the
// cmd/rpc indexer-blob tests in particular — can exercise GetAccountDelta's cache-hit path
// without standing up a full validator set and certificate chain for the replay path.
//
// TEST-ONLY. It has no production caller and must not acquire one. It lazily assigns
// c.accountDeltaCache, and that pointer field is read UNSYNCHRONIZED by GetAccountDelta from
// every RPC goroutine (the cache's RWMutex protects the map, not the field). It is therefore
// only safe to call during setup, before the Controller begins serving — never concurrently
// with GetAccountDelta.
//
// The seeded slices become part of the shared, immutable cache entry: the caller must not
// mutate them afterwards (see this file's doc comments on read-only entries).
func (c *Controller) SeedAccountDeltaCache(blockHeight uint64, added, changed, removed []*fsm.AccountChangeEntry) {
	if c.accountDeltaCache == nil {
		c.accountDeltaCache = newAccountDeltaCache(0)
	}
	c.accountDeltaCache.put(blockHeight, &accountDeltaEntry{added: added, changed: changed, removed: removed})
}

// put() stores a height's delta, evicting the oldest height once at capacity
func (c *accountDeltaCache) put(height uint64, entry *accountDeltaEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// overwrite in place so a re-commit of the same height doesn't consume a second
	// FIFO slot and evict a live neighbor
	if _, ok := c.entries[height]; ok {
		c.entries[height] = entry
		return
	}
	c.entries[height] = entry
	c.order = append(c.order, height)
	if len(c.order) > c.maxEntries {
		evictHeight := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, evictHeight)
	}
}
