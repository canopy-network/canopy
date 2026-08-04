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
	require.NoError(t, c.RecordSet(key, []byte("v1")))
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
	require.NoError(t, c.RecordSet(key, []byte("new")))
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
	require.NoError(t, c.RecordSet(key, []byte("same")))
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
	require.NoError(t, c.RecordDelete(key))
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
	require.NoError(t, c.RecordSet(key, []byte("v1")))
	require.NoError(t, c.RecordDelete(key))
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
	require.NoError(t, c.RecordSet(key, []byte("intermediate")))
	require.NoError(t, c.RecordSet(key, []byte("final")))
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

// TestAccountChangeCollector_GetPrevErrorPropagates: the collector only ever runs during a
// throwaway on-demand replay (never a real block commit, see StateMachine.ApplyBlock), so
// an internal failure is a plain propagated error like any other Set/Delete failure.
func TestAccountChangeCollector_GetPrevErrorPropagates(t *testing.T) {
	getPrevErr := lib.ErrUnmarshal(errors.New("test"))
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, getPrevErr }
	c := NewAccountChangeCollector(getPrev)
	address := newTestAddress(t)
	key := KeyForAccount(address)
	err := c.RecordSet(key, []byte("v1"))
	require.Error(t, err)
	require.Equal(t, getPrevErr.Error(), err.Error())
}

// TestAccountChangeCollector_TooShortKeyErrorsWithoutPanic guards entryFor's slice bound:
// a key that carries AccountPrefix() but is too short to also contain the address segment's
// length-prefix byte must return an error, not panic.
func TestAccountChangeCollector_TooShortKeyErrorsWithoutPanic(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, nil }
	c := NewAccountChangeCollector(getPrev)
	// exactly AccountPrefix() with nothing appended -- too short to contain the address
	// segment's own 1-byte length prefix, let alone an address.
	key := append([]byte(nil), AccountPrefix()...)
	var err lib.ErrorI
	require.NotPanics(t, func() {
		err = c.RecordSet(key, []byte("v1"))
	})
	require.Error(t, err)
}
