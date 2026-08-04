package controller

import (
	"bytes"
	"testing"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

/*
This file covers the live-commit capture pipeline: accountCollectorForLiveCommit (the gate
deciding whether a commit captures account changes at all) and cacheAccountDelta (the only
writer of the tip cache). A wrong gate silently degrades every fast-path response to replay
-- or worse, captures during sync -- so the gating is pinned by table tests.
*/

// the collector must be created only for a live commit on a node configured to serve
// indexer blobs -- never while syncing (nothing is polling yet), never with the flag off
func TestAccountCollectorForLiveCommit(t *testing.T) {
	// a real FSM so the returned collector's baseline lookup (c.FSM.Get) actually works
	base := newBareReplayController(t, 1, nil)
	tests := []struct {
		name          string
		syncing       bool
		serveLive     bool
		wantCollector bool
	}{
		{name: "live commit with flag on captures", syncing: false, serveLive: true, wantCollector: true},
		{name: "syncing never captures", syncing: true, serveLive: true, wantCollector: false},
		{name: "flag off never captures", syncing: false, serveLive: false, wantCollector: false},
		{name: "syncing with flag off never captures", syncing: true, serveLive: false, wantCollector: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Controller{
				FSM:    base.FSM,
				Config: lib.Config{RPCConfig: lib.RPCConfig{ServeIndexerBlobsLive: test.serveLive}},
			}
			collector := c.accountCollectorForLiveCommit(test.syncing)
			if !test.wantCollector {
				require.Nil(t, collector)
				return
			}
			require.NotNil(t, collector)
			// the collector must be usable: its baseline read goes through c.FSM.Get
			address := bytes.Repeat([]byte{0xAA}, crypto.AddressSize)
			collector.RecordSet(fsm.KeyForAccount(crypto.NewAddressFromBytes(address)), []byte("account-bytes"))
			require.NoError(t, collector.Err())
			delta := collector.Results()
			require.NotNil(t, delta)
			require.Len(t, delta.Added, 1)
			require.Equal(t, address, delta.Added[0].Address)
		})
	}
}

// a nil collector means no collector ran for this height (sync, pre-calculated block
// result, or flag off) -- caching an empty, authoritative-looking entry would silently
// serve a wrong delta forever
func TestCacheAccountDelta_NilCollectorCachesNothing(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	c.cacheAccountDelta(5, nil)
	_, ok := c.accountDeltaCache.get(5)
	require.False(t, ok)
	c.accountDeltaCache.mu.RLock()
	require.Empty(t, c.accountDeltaCache.entries)
	c.accountDeltaCache.mu.RUnlock()
}

// the cache is nil for a Controller not built by New() -- a caching miss must never take
// down a commit
func TestCacheAccountDelta_NilCacheDoesNotPanic(t *testing.T) {
	c := &Controller{}
	collector := fsm.NewAccountChangeCollector(func([]byte) ([]byte, lib.ErrorI) { return nil, nil })
	address := bytes.Repeat([]byte{0xBB}, crypto.AddressSize)
	collector.RecordSet(fsm.KeyForAccount(crypto.NewAddressFromBytes(address)), []byte("account-bytes"))
	require.NotPanics(t, func() { c.cacheAccountDelta(7, collector) })
}

// a healthy collector's results must land in the cache under the BLOCK height it ran for
func TestCacheAccountDelta_HealthyCollectorEntryRetrievable(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0)}
	collector := fsm.NewAccountChangeCollector(func([]byte) ([]byte, lib.ErrorI) { return nil, nil })
	address := bytes.Repeat([]byte{0xCC}, crypto.AddressSize)
	value := []byte("account-bytes")
	collector.RecordSet(fsm.KeyForAccount(crypto.NewAddressFromBytes(address)), value)
	c.cacheAccountDelta(9, collector)
	delta, ok := c.accountDeltaCache.get(9)
	require.True(t, ok)
	require.NotNil(t, delta)
	require.Len(t, delta.Added, 1)
	require.Equal(t, address, delta.Added[0].Address)
	require.Equal(t, value, delta.Added[0].FinalValue)
	// keyed by block height only -- no off-by-one neighbor entries
	_, ok = c.accountDeltaCache.get(8)
	require.False(t, ok)
	_, ok = c.accountDeltaCache.get(10)
	require.False(t, ok)
}

// a poisoned collector was disabled mid-block and its set is incomplete: caching it would
// serve a silently wrong delta for this height forever -- it must be skipped (and the
// height then falls back to on-demand replay)
func TestCacheAccountDelta_PoisonedCollectorNotCached(t *testing.T) {
	c := &Controller{accountDeltaCache: newAccountDeltaCache(0), log: lib.NewDefaultLogger()}
	// the baseline read fails on first touch, poisoning the collector
	collector := fsm.NewAccountChangeCollector(func([]byte) ([]byte, lib.ErrorI) {
		return nil, lib.ErrNilBlockHeader()
	})
	address := bytes.Repeat([]byte{0xDD}, crypto.AddressSize)
	collector.RecordSet(fsm.KeyForAccount(crypto.NewAddressFromBytes(address)), []byte("account-bytes"))
	require.Error(t, collector.Err())
	c.cacheAccountDelta(11, collector)
	_, ok := c.accountDeltaCache.get(11)
	require.False(t, ok, "a poisoned collector's incomplete delta must never be cached")
	c.accountDeltaCache.mu.RLock()
	require.Empty(t, c.accountDeltaCache.entries)
	c.accountDeltaCache.mu.RUnlock()
}
