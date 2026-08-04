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
	c.accountDeltaCache.put(5, &accountDeltaEntry{added: []*fsm.AccountChangeEntry{{Address: []byte("a")}}})
	added, changed, removed, err := c.GetAccountDelta(context.Background(), 6)
	require.Nil(t, err)
	require.Len(t, added, 1)
	require.Equal(t, []byte("a"), added[0].Address)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

// the cache hit must be served without touching c.FSM at all — a nil FSM here proves the
// lookup happens before any store access, and also guards the height-1 keying: a lookup
// under the wrong key would fall through to the replay path and panic
func TestGetAccountDelta_TipCacheHitDoesNotTouchFSM(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	c.accountDeltaCache.put(9, &accountDeltaEntry{
		changed: []*fsm.AccountChangeEntry{{Address: []byte("b"), PrevValue: []byte("x"), FinalValue: []byte("y")}},
		removed: []*fsm.AccountChangeEntry{{Address: []byte("c"), PrevValue: []byte("z")}},
	})
	require.Nil(t, c.FSM)
	added, changed, removed, err := c.GetAccountDelta(context.Background(), 10)
	require.Nil(t, err)
	require.Empty(t, added)
	require.Len(t, changed, 1)
	require.Len(t, removed, 1)
}

// heights below 2 have no committed block to pair with and must be rejected before any
// store or FSM access (matching fsm.IndexerBlob's own height<=1 guard)
func TestGetAccountDelta_RejectsHeightsBelowTwo(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	require.Nil(t, c.FSM)
	for _, height := range []uint64{0, 1} {
		added, changed, removed, err := c.GetAccountDelta(context.Background(), height)
		require.NotNil(t, err)
		require.Equal(t, lib.CodeWrongBlockHeight, err.Code())
		require.Empty(t, added)
		require.Empty(t, changed)
		require.Empty(t, removed)
	}
}
