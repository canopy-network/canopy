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

// AccountDelta is one block's classified account changes: the accounts added (didn't
// exist at height-1), changed (existed with a different value), and removed (existed at
// height-1, deleted by end of block). It travels as one unit from the collector through
// the controller's tip cache to the RPC layer so the three identically-typed slices can
// never be transposed across a call boundary.
type AccountDelta struct {
	Added   []*AccountChangeEntry
	Changed []*AccountChangeEntry
	Removed []*AccountChangeEntry
}

// Clone returns a copy of the delta with fresh slice headers, so an append on the result
// can never write through into the receiver's backing arrays (e.g. the shared tip-cache
// entry). The *AccountChangeEntry values themselves stay shared -- they are immutable by
// contract.
func (d *AccountDelta) Clone() *AccountDelta {
	if d == nil {
		return nil
	}
	return &AccountDelta{
		Added:   append([]*AccountChangeEntry(nil), d.Added...),
		Changed: append([]*AccountChangeEntry(nil), d.Changed...),
		Removed: append([]*AccountChangeEntry(nil), d.Removed...),
	}
}

// AccountChangeCollector accumulates every account write/delete during a single
// ApplyBlock call, keyed by address, classifying each into added/changed/removed
// relative to the pre-block (height-1) baseline. It captures the baseline value on
// first touch only, before any of this block's own writes shadow it — safe by
// construction even if a store Get transparently overlays in-progress writes,
// because on first touch nothing has been written for that key yet this block.
//
// The collector is SELF-POISONING: it hooks the consensus write path
// (StateMachine.Set/Delete), so an internal failure — a baseline read error, a
// malformed key — must never abort the store write of an already-agreed block. The
// first internal error is recorded instead, all further collection is disabled, and
// RecordSet/RecordDelete return silently. Consumers MUST check Err() before using
// Results(): a poisoned collector's set is incomplete and must never be cached or
// served as a delta.
type AccountChangeCollector struct {
	getPrevValue func(key []byte) ([]byte, lib.ErrorI)
	entries      map[string]*AccountChangeEntry
	err          lib.ErrorI // first internal error; non-nil disables all further collection (see Err)
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
// An internal failure poisons the collector (see Err) rather than being returned,
// so the consensus store write this hook precedes always proceeds.
func (c *AccountChangeCollector) RecordSet(key, value []byte) {
	// a poisoned collector is disabled for the rest of the block
	if c.err != nil {
		return
	}
	entry, err := c.entryFor(key)
	if err != nil {
		c.poison(err)
		return
	}
	entry.FinalValue = append([]byte(nil), value...)
}

// RecordDelete records an account deletion. key includes AccountPrefix().
// An internal failure poisons the collector (see Err) rather than being returned,
// so the consensus store write this hook precedes always proceeds.
func (c *AccountChangeCollector) RecordDelete(key []byte) {
	// a poisoned collector is disabled for the rest of the block
	if c.err != nil {
		return
	}
	entry, err := c.entryFor(key)
	if err != nil {
		c.poison(err)
		return
	}
	entry.FinalValue = nil
}

// poison records the collector's first internal error and disables all further
// collection; the already-collected entries are dropped so a poisoned collector
// can never leak a partial set through Results()
func (c *AccountChangeCollector) poison(err lib.ErrorI) {
	c.err = err
	c.entries = make(map[string]*AccountChangeEntry)
}

// Err returns the first internal error the collector hit, or nil. A non-nil value
// means collection was disabled mid-block: the delta for that block is incomplete
// and must never be cached or served — consumers check this before Results().
func (c *AccountChangeCollector) Err() lib.ErrorI { return c.err }

func (c *AccountChangeCollector) entryFor(key []byte) (*AccountChangeEntry, lib.ErrorI) {
	addrKey := string(key)
	if entry, ok := c.entries[addrKey]; ok {
		return entry, nil
	}
	// key is accountKeyPrefix followed by a length-prefixed address segment
	// (KeyForAccount uses lib.JoinLenPrefix(accountPrefix, addr.Bytes())), so skip
	// the prefix plus the address segment's own 1-byte length prefix.
	//
	// This runs on the consensus write path (via StateMachine.Set/Delete's collector
	// hook, see fsm/state.go), so a malformed key here must produce an error -- which
	// RecordSet/RecordDelete turn into a poison, not a consensus failure -- rather than
	// a slice-bounds panic; panicking would abort ApplyBlock mid block commit and take
	// the node down instead of just failing indexing. Checked BEFORE the baseline read
	// so an invalid key fails without a wasted store Get.
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

// Results returns every touched account classified as added (didn't exist at
// height-1), changed (existed with a different value), or removed (existed at
// height-1, deleted by end of block). An account touched but net-unchanged
// (e.g. set then deleted within the same block, or set to its existing value)
// is dropped from all three lists. Returns nil for a poisoned collector.
func (c *AccountChangeCollector) Results() *AccountDelta {
	// a poisoned collector's set is incomplete; never surface it (see Err)
	if c.err != nil {
		return nil
	}
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
