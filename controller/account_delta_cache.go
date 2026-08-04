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

// get() returns the cached delta for a height, if present
func (c *accountDeltaCache) get(height uint64) (*accountDeltaEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[height]
	return entry, ok
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
