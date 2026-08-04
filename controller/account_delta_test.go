package controller

import (
	"context"
	"testing"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// the tip cache is keyed by BLOCK height while GetAccountDelta takes a STATE VERSION,
// so a delta cached for block 5 is served for state version 6 (see GetAccountDelta's
// height-convention comment)
func TestGetAccountDelta_TipCacheHit(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	c.accountDeltaCache.put(5, &fsm.AccountDelta{Added: []*fsm.AccountChangeEntry{{Address: []byte("a")}}})
	delta, err := c.GetAccountDelta(context.Background(), 6)
	require.Nil(t, err)
	require.NotNil(t, delta)
	require.Len(t, delta.Added, 1)
	require.Equal(t, []byte("a"), delta.Added[0].Address)
	require.Empty(t, delta.Changed)
	require.Empty(t, delta.Removed)
}

// the cache hit must be served without touching c.FSM at all — a nil FSM here proves the
// lookup happens before any store access, and also guards the height-1 keying: a lookup
// under the wrong key would fall through to the replay path and panic
func TestGetAccountDelta_TipCacheHitDoesNotTouchFSM(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	c.accountDeltaCache.put(9, &fsm.AccountDelta{
		Changed: []*fsm.AccountChangeEntry{{Address: []byte("b"), PrevValue: []byte("x"), FinalValue: []byte("y")}},
		Removed: []*fsm.AccountChangeEntry{{Address: []byte("c"), PrevValue: []byte("z")}},
	})
	require.Nil(t, c.FSM)
	delta, err := c.GetAccountDelta(context.Background(), 10)
	require.Nil(t, err)
	require.NotNil(t, delta)
	require.Empty(t, delta.Added)
	require.Len(t, delta.Changed, 1)
	require.Len(t, delta.Removed, 1)
}

// a cache hit returns fresh slice headers (fsm.AccountDelta.Clone), so a caller appending
// to the result can never write through into the shared tip-cache entry that every other
// reader is served from
func TestGetAccountDelta_TipCacheHitCopiesSliceHeaders(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	c.accountDeltaCache.put(5, &fsm.AccountDelta{Added: []*fsm.AccountChangeEntry{{Address: []byte("a")}}})
	first, err := c.GetAccountDelta(context.Background(), 6)
	require.Nil(t, err)
	// abuse the returned delta the way a careless future caller might
	first.Added = append(first.Added, &fsm.AccountChangeEntry{Address: []byte("rogue")})
	first.Changed = append(first.Changed, &fsm.AccountChangeEntry{Address: []byte("rogue")})
	// a second lookup must be unaffected by the first caller's appends
	second, err := c.GetAccountDelta(context.Background(), 6)
	require.Nil(t, err)
	require.Len(t, second.Added, 1)
	require.Equal(t, []byte("a"), second.Added[0].Address)
	require.Empty(t, second.Changed)
}

// heights below 2 have no committed block to pair with and must be rejected before any
// store or FSM access (matching fsm.IndexerBlob's own height<=1 guard)
func TestGetAccountDelta_RejectsHeightsBelowTwo(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	require.Nil(t, c.FSM)
	for _, height := range []uint64{0, 1} {
		delta, err := c.GetAccountDelta(context.Background(), height)
		require.NotNil(t, err)
		require.Equal(t, lib.CodeWrongBlockHeight, err.Code())
		require.Nil(t, delta)
	}
}
