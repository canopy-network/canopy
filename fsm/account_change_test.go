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
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordSet(key, []byte("v1")))
	added, changed, removed := c.Results()
	require.Len(t, added, 1)
	require.Empty(t, changed)
	require.Empty(t, removed)
	require.Equal(t, []byte("addr1"), added[0].Address)
	require.Equal(t, []byte("v1"), added[0].FinalValue)
	require.Nil(t, added[0].PrevValue)
}

func TestAccountChangeCollector_ExistingAccountIsChanged(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("old"), nil }
	c := NewAccountChangeCollector(getPrev)
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordSet(key, []byte("new")))
	added, changed, removed := c.Results()
	require.Empty(t, added)
	require.Len(t, changed, 1)
	require.Empty(t, removed)
	require.Equal(t, []byte("old"), changed[0].PrevValue)
	require.Equal(t, []byte("new"), changed[0].FinalValue)
}

func TestAccountChangeCollector_SetSameValueIsNotChanged(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("same"), nil }
	c := NewAccountChangeCollector(getPrev)
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordSet(key, []byte("same")))
	added, changed, removed := c.Results()
	require.Empty(t, added)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

func TestAccountChangeCollector_DeleteExistingAccountIsRemoved(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return []byte("old"), nil }
	c := NewAccountChangeCollector(getPrev)
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordDelete(key))
	added, changed, removed := c.Results()
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
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordSet(key, []byte("v1")))
	require.NoError(t, c.RecordDelete(key))
	added, changed, removed := c.Results()
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
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	require.NoError(t, c.RecordSet(key, []byte("intermediate")))
	require.NoError(t, c.RecordSet(key, []byte("final")))
	require.Equal(t, 1, getPrevCalls, "baseline should only be fetched once, on first touch")
	added, changed, removed := c.Results()
	require.Empty(t, added)
	require.Len(t, changed, 1)
	require.Empty(t, removed)
	require.Equal(t, []byte("baseline"), changed[0].PrevValue)
	require.Equal(t, []byte("final"), changed[0].FinalValue)
}

func TestAccountChangeCollector_GetPrevError(t *testing.T) {
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, lib.ErrUnmarshal(errors.New("test")) }
	c := NewAccountChangeCollector(getPrev)
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	err := c.RecordSet(key, []byte("v1"))
	require.Error(t, err)
}
