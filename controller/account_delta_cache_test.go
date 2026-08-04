package controller

import (
	"testing"

	"github.com/canopy-network/canopy/fsm"
	"github.com/stretchr/testify/require"
)

func TestAccountDeltaCache_PutAndGet(t *testing.T) {
	c := newAccountDeltaCache(4)
	entry := &accountDeltaEntry{
		added:   []*fsm.AccountChangeEntry{{Address: []byte("a")}},
		changed: nil,
		removed: nil,
	}
	c.put(10, entry)
	got, ok := c.get(10)
	require.True(t, ok)
	require.Equal(t, entry, got)
}

func TestAccountDeltaCache_MissReturnsFalse(t *testing.T) {
	c := newAccountDeltaCache(4)
	_, ok := c.get(999)
	require.False(t, ok)
}

func TestAccountDeltaCache_EvictsOldestBeyondCapacity(t *testing.T) {
	c := newAccountDeltaCache(2)
	c.put(1, &accountDeltaEntry{})
	c.put(2, &accountDeltaEntry{})
	c.put(3, &accountDeltaEntry{})
	_, ok := c.get(1)
	require.False(t, ok, "height 1 should have been evicted")
	_, ok = c.get(2)
	require.True(t, ok)
	_, ok = c.get(3)
	require.True(t, ok)
}

// re-putting an existing height must overwrite in place without consuming a second
// FIFO slot, otherwise a re-commit of the same height would evict a live neighbor
func TestAccountDeltaCache_OverwriteExistingHeightDoesNotEvict(t *testing.T) {
	c := newAccountDeltaCache(2)
	first := &accountDeltaEntry{added: []*fsm.AccountChangeEntry{{Address: []byte("a")}}}
	second := &accountDeltaEntry{added: []*fsm.AccountChangeEntry{{Address: []byte("b")}}}
	c.put(1, first)
	c.put(2, &accountDeltaEntry{})
	c.put(1, second)
	got, ok := c.get(1)
	require.True(t, ok)
	require.Equal(t, second, got, "height 1 should hold the overwritten entry")
	_, ok = c.get(2)
	require.True(t, ok, "height 2 must not be evicted by an in-place overwrite")
}

// a non-positive capacity must fall back to the default rather than evicting every put
func TestAccountDeltaCache_NonPositiveCapacityUsesDefault(t *testing.T) {
	c := newAccountDeltaCache(0)
	require.Equal(t, defaultAccountDeltaCacheEntries, c.maxEntries)
	c.put(1, &accountDeltaEntry{})
	_, ok := c.get(1)
	require.True(t, ok)
}
