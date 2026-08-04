// fsm/account_change_test.go
package fsm

import (
	"errors"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestAccountChangeCollector_NewAccountIsAdded(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, nil } // didn't exist before
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("v1"))
	require.Nil(t, c.Err())
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Len(t, added, 1)
	require.Empty(t, changed)
	require.Empty(t, removed)
	require.Equal(t, address.Bytes(), added[0].Address)
	require.Equal(t, []byte("v1"), added[0].FinalValue)
	require.Nil(t, added[0].PrevValue)
}

func TestAccountChangeCollector_ExistingAccountIsChanged(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("old"), nil }
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("new"))
	require.Nil(t, c.Err())
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Empty(t, added)
	require.Len(t, changed, 1)
	require.Empty(t, removed)
	require.Equal(t, []byte("old"), changed[0].PrevValue)
	require.Equal(t, []byte("new"), changed[0].FinalValue)
}

func TestAccountChangeCollector_SetSameValueIsNotChanged(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("same"), nil }
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("same"))
	require.Nil(t, c.Err())
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Empty(t, added)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

func TestAccountChangeCollector_DeleteExistingAccountIsRemoved(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("old"), nil }
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordDelete(key)
	require.Nil(t, c.Err())
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Empty(t, added)
	require.Empty(t, changed)
	require.Len(t, removed, 1)
	require.Equal(t, []byte("old"), removed[0].PrevValue)
	require.Nil(t, removed[0].FinalValue)
}

func TestAccountChangeCollector_SetThenDeleteSameBlockIsNetNoOp(t *testing.T) {
	// account existed before this block (SetAccount's zero-balance path deletes an
	// account that was created and zeroed out within the same block).
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, nil }
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("v1"))
	c.RecordDelete(key)
	require.Nil(t, c.Err())
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Empty(t, added)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

func TestAccountChangeCollector_MultipleTouchesUseHeight1Baseline(t *testing.T) {
	// two writes to the same account within a block must classify against the
	// pre-block baseline, not the mid-block intermediate value.
	getPrevCalls := 0
	getPrev := func(key []byte) ([]byte, lib.ErrorI) {
		getPrevCalls++
		return []byte("baseline"), nil
	}
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("intermediate"))
	c.RecordSet(key, []byte("final"))
	require.Equal(t, 1, getPrevCalls, "baseline should only be fetched once, on first touch")
	delta := c.Results()
	require.NotNil(t, delta)
	added, changed, removed := delta.Added, delta.Changed, delta.Removed
	require.Empty(t, added)
	require.Len(t, changed, 1)
	require.Empty(t, removed)
	require.Equal(t, []byte("baseline"), changed[0].PrevValue)
	require.Equal(t, []byte("final"), changed[0].FinalValue)
}

// TestAccountChangeCollector_GetPrevErrorPoisons guards the self-poisoning contract:
// the collector hooks the consensus write path, so an internal error must never
// propagate out of RecordSet/RecordDelete (which would abort committing a block the
// network already agreed on). Instead the first error poisons the collector: Err()
// reports it, all further collection is disabled, and Results() surfaces nothing.
func TestAccountChangeCollector_GetPrevErrorPoisons(t *testing.T) {
	getPrevErr := lib.ErrUnmarshal(errors.New("test"))
	getPrevCalls := 0
	getPrev := func(key []byte) ([]byte, lib.ErrorI) {
		getPrevCalls++
		return nil, getPrevErr
	}
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	c.RecordSet(key, []byte("v1"))
	require.NotNil(t, c.Err())
	require.Equal(t, getPrevErr.Error(), c.Err().Error())
	// collection is disabled after poisoning: no further baseline reads happen and the
	// first error is retained
	c.RecordSet(key, []byte("v2"))
	c.RecordDelete(key)
	require.Equal(t, 1, getPrevCalls, "a poisoned collector must not keep reading baselines")
	require.Equal(t, getPrevErr.Error(), c.Err().Error())
	// a poisoned collector's set is incomplete and must never be surfaced
	require.Nil(t, c.Results())
}

// TestAccountChangeCollector_PoisonDropsPartialResults guards that entries collected
// BEFORE the poisoning error are dropped too — a partial set served as a delta would
// be silently wrong.
func TestAccountChangeCollector_PoisonDropsPartialResults(t *testing.T) {
	getPrevCalls := 0
	getPrev := func(key []byte) ([]byte, lib.ErrorI) {
		getPrevCalls++
		if getPrevCalls > 1 {
			return nil, lib.ErrUnmarshal(errors.New("transient store read failure"))
		}
		return nil, nil
	}
	c := NewAccountChangeCollector(getPrev)
	c.RecordSet(KeyForAccount(newTestAddress(t)), []byte("v1"))
	require.Nil(t, c.Err())
	c.RecordSet(KeyForAccount(newTestAddress(t, 1)), []byte("v2"))
	require.NotNil(t, c.Err())
	require.Nil(t, c.Results())
}

// TestAccountChangeCollector_TooShortKeyPoisonsWithoutPanic guards entryFor's slice bound:
// a key that carries AccountPrefix() but is too short to also contain the address segment's
// length-prefix byte must poison the collector, not panic. This runs on the consensus write
// path (via StateMachine.Set/Delete's collector hook), so a panic here would abort
// ApplyBlock mid block commit instead of just failing indexing.
func TestAccountChangeCollector_TooShortKeyPoisonsWithoutPanic(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, nil }
	c := NewAccountChangeCollector(getPrev)
	// exactly AccountPrefix() with nothing appended -- too short to contain the address
	// segment's own 1-byte length prefix, let alone an address.
	key := append([]byte(nil), AccountPrefix()...)
	require.NotPanics(t, func() {
		c.RecordSet(key, []byte("v1"))
	})
	require.NotNil(t, c.Err())
}
