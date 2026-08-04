package fsm

import (
	"bytes"

	"github.com/canopy-network/canopy/lib"
)

// accountKeyPrefix is AccountPrefix() computed once: the hooks in StateMachine.Set/Delete
// run this prefix compare on every write of a live block application, and AccountPrefix()
// allocates a fresh slice per call (lib.JoinLenPrefix). Safe to hoist because the source
// accountPrefix byte slice is a package var that is never mutated.
var accountKeyPrefix = AccountPrefix()

// AccountChangeEntry is one account touched during a single ApplyBlock call.
// PrevValue is nil if the account did not exist at height-1 (the pre-block baseline).
// FinalValue is nil if the account was deleted by the end of the block.
type AccountChangeEntry struct {
	Address    []byte
	PrevValue  []byte
	FinalValue []byte
}

// AccountDelta is one block's classified account changes: added (didn't exist at
// height-1), changed (existed with a different value), and removed (existed at height-1,
// deleted by end of block). One unit so the three identically-typed slices can't be
// transposed across a call boundary.
type AccountDelta struct {
	Added   []*AccountChangeEntry
	Changed []*AccountChangeEntry
	Removed []*AccountChangeEntry
}

// AccountChangeCollector accumulates every account write/delete during a single
// ApplyBlock call, keyed by address, classifying each against the pre-block (height-1)
// baseline. The baseline is captured on first touch only — before any of this block's own
// writes can shadow it. Only ever runs during a throwaway replay (StateMachine.ReplayBlock),
// never during a real commit, so a failure here fails one RPC request, not consensus.
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
	// key is accountKeyPrefix followed by a length-prefixed address segment
	// (KeyForAccount uses lib.JoinLenPrefix(accountPrefix, addr.Bytes())), so skip
	// the prefix plus the address segment's own 1-byte length prefix. Checked before the
	// baseline read so a malformed key fails without a wasted store Get.
	if len(key) < len(accountKeyPrefix)+1 {
		return nil, ErrInvalidKey(key)
	}
	prevValue, err := c.getPrevValue(key)
	if err != nil {
		return nil, err
	}
	address := append([]byte(nil), key[len(accountKeyPrefix)+1:]...)
	entry := &AccountChangeEntry{Address: address, PrevValue: prevValue}
	c.entries[addrKey] = entry
	return entry, nil
}

// Results classifies every touched account into added/changed/removed. An account
// touched but net-unchanged (set then deleted, or set to its existing value) is
// dropped from all three lists.
func (c *AccountChangeCollector) Results() *AccountDelta {
	delta := new(AccountDelta)
	for _, e := range c.entries {
		switch {
		case e.PrevValue == nil && e.FinalValue == nil:
			continue
		case e.PrevValue == nil:
			delta.Added = append(delta.Added, e)
		case e.FinalValue == nil:
			delta.Removed = append(delta.Removed, e)
		case !bytes.Equal(e.PrevValue, e.FinalValue):
			delta.Changed = append(delta.Changed, e)
		}
	}
	return delta
}
