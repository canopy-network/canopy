package fsm

import (
	"bytes"

	"github.com/canopy-network/canopy/lib"
)

// AccountChangeEntry is one account touched during a single ApplyBlock call.
// PrevValue is nil if the account did not exist at height-1 (the pre-block baseline).
// FinalValue is nil if the account was deleted by the end of the block.
type AccountChangeEntry struct {
	Address    []byte
	PrevValue  []byte
	FinalValue []byte
}

// AccountChangeCollector accumulates every account write/delete during a single
// ApplyBlock call, keyed by address, classifying each into added/changed/removed
// relative to the pre-block (height-1) baseline. It captures the baseline value on
// first touch only, before any of this block's own writes shadow it — safe by
// construction even if a store Get transparently overlays in-progress writes,
// because on first touch nothing has been written for that key yet this block.
type AccountChangeCollector struct {
	getPrevValue func(key []byte) ([]byte, lib.ErrorI)
	entries      map[string]*AccountChangeEntry
}

// NewAccountChangeCollector takes a lookup callback (StateMachine.Get bound to the
// FSM instance being hooked) used to fetch each touched account's pre-block value.
func NewAccountChangeCollector(getPrevValue func(key []byte) ([]byte, lib.ErrorI)) *AccountChangeCollector {
	return &AccountChangeCollector{
		getPrevValue: getPrevValue,
		entries:      make(map[string]*AccountChangeEntry),
	}
}

// RecordSet records an account write. key includes AccountPrefix(); value is the
// account's already-marshalled bytes (matches SetAccount's `bz` argument to Set).
func (c *AccountChangeCollector) RecordSet(key, value []byte) lib.ErrorI {
	entry, err := c.entryFor(key)
	if err != nil {
		return err
	}
	entry.FinalValue = append([]byte(nil), value...)
	return nil
}

// RecordDelete records an account deletion. key includes AccountPrefix().
func (c *AccountChangeCollector) RecordDelete(key []byte) lib.ErrorI {
	entry, err := c.entryFor(key)
	if err != nil {
		return err
	}
	entry.FinalValue = nil
	return nil
}

func (c *AccountChangeCollector) entryFor(key []byte) (*AccountChangeEntry, lib.ErrorI) {
	addrKey := string(key)
	if entry, ok := c.entries[addrKey]; ok {
		return entry, nil
	}
	prevValue, err := c.getPrevValue(key)
	if err != nil {
		return nil, err
	}
	// key is AccountPrefix() followed by a length-prefixed address segment
	// (KeyForAccount uses lib.JoinLenPrefix(accountPrefix, addr.Bytes())), so skip
	// AccountPrefix() plus the address segment's own 1-byte length prefix.
	address := append([]byte(nil), key[len(AccountPrefix())+1:]...)
	entry := &AccountChangeEntry{Address: address, PrevValue: prevValue}
	c.entries[addrKey] = entry
	return entry, nil
}

// Results returns every touched account classified as added (didn't exist at
// height-1), changed (existed with a different value), or removed (existed at
// height-1, deleted by end of block). An account touched but net-unchanged
// (e.g. set then deleted within the same block, or set to its existing value)
// is dropped from all three lists.
func (c *AccountChangeCollector) Results() (added, changed, removed []*AccountChangeEntry) {
	for _, e := range c.entries {
		switch {
		case e.PrevValue == nil && e.FinalValue == nil:
			continue
		case e.PrevValue == nil:
			added = append(added, e)
		case e.FinalValue == nil:
			removed = append(removed, e)
		case !bytes.Equal(e.PrevValue, e.FinalValue):
			changed = append(changed, e)
		}
	}
	return
}
