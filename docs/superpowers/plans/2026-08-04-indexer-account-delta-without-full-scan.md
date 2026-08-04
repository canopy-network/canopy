# Eliminate Full Account Scan in IndexerBlob — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `IndexerBlob`'s full `IterateAndAppend(AccountPrefix())` scan (~1.35M accounts, done twice per request) with a per-block touched-account capture, so account deltas cost O(accounts touched by that block) instead of O(total accounts), for both live blocks and arbitrary historical/backfill requests.

**Architecture:** A new `AccountChangeCollector` hooks the FSM's generic `Set`/`Delete` methods (filtered to `AccountPrefix()`), capturing each touched account's pre-block value (via one `Get` on first touch) and its final value. `ApplyBlock` gains two optional parameters — `collector` (nil-safe) and `skipRoot` (skips `store.Root()` for throwaway replay calls). The live/consensus commit path attaches a collector and populates an in-memory tip cache; any other request (historical, backfill, evicted tip) falls back to replaying `ApplyBlock` with `skipRoot=true` against a `TimeMachine(height-1)` snapshot, never committing. Both paths produce the same `[][]byte` shape `IndexerBlob.Accounts`/`DeltaIndexerBlobs` already use, so the RPC response format is unchanged.

**Tech Stack:** Go, canopy-core FSM/store/controller/RPC layers, existing `testify` test conventions (`fsm/indexer_test.go`, `fsm/state_test.go`).

**Spec:** `docs/superpowers/specs/2026-08-04-indexer-account-delta-without-full-scan-design.md`

---

## Testing Conventions Note

Confirmed existing test helpers this plan reuses directly: `newTestStateMachine(t)` and
`newTestAddress(t, ...)` (`fsm/state_test.go:400,429`), and the table-driven `TestApplyBlock`
(`fsm/state_test.go:97-289`, the current sole caller of `ApplyBlock` in tests). Several later
tasks reference additional fixture-construction names (e.g. `newTestApplyBlockFixture`,
`buildMultiBlockFixtureWithAccountChanges`, `newTestControllerWithBlock`) that are **not**
confirmed to exist — they are descriptive placeholders for "however the nearest existing test
in that file already builds this kind of fixture." Every such step says so explicitly and
instructs grepping/reading that file first. Do not create a new shared helper under one of
these placeholder names without first confirming no equivalent already exists — extend the
existing pattern (e.g. `TestDeltaIndexerBlobs_ChangedAddedRemoved`'s inline setup at
`fsm/indexer_test.go:11`) inline in the new test instead, if that's how the file already does it.

---

## File Structure

- **Create** `fsm/account_change.go` — `AccountChangeCollector`, `AccountChangeEntry`, classification logic. No dependency on `StateMachine` — takes a plain lookup callback, so it's unit-testable standalone.
- **Create** `fsm/account_change_test.go` — collector unit tests.
- **Modify** `fsm/state.go` — add `accountCollector` field to `StateMachine`; hook `Set`/`Delete`; add `bytes` import; change `ApplyBlock` signature (`collector`, `skipRoot` params); conditionally skip `store.Root()`.
- **Modify** `fsm/state_test.go:272` — update the one existing `ApplyBlock` call site.
- **Modify** `controller/tx.go:309` — update the mempool `ApplyBlock` call site (pass `nil, false` — no behavior change).
- **Modify** `controller/block.go` — update `ApplyAndValidateBlock`'s `ApplyBlock` call site; thread a `collector` param through `ApplyAndValidateBlock`, `CommitCertificate`, `CommitCertificateParallel`; add a tip cache populated in `commitToStore`; add `GetAccountDelta(ctx, height)` on `Controller` for the lazy/replay path.
- **Create** `controller/account_delta_cache.go` — the in-memory tip cache (`accountDeltaTipCache`), small FIFO similar in spirit to `cmd/rpc/types.go`'s `indexerBlobCache` but simpler (fixed small size, no `current`/`delta` distinction).
- **Create** `controller/account_delta_cache_test.go`.
- **Modify** `lib/config.go` — add `RPCConfig.ServeIndexerBlobsLive bool`, default `true` in `DefaultRPCConfig()`.
- **Modify** `fsm/indexer.go` — add `skipAccounts bool` param to `IndexerBlob`; update `IndexerBlobs` (plural wrapper) and its test call sites.
- **Modify** `cmd/rpc/query.go` — `IndexerBlobsCached` calls `IndexerBlob(ctx, height, true)` (skip accounts), fetches the account delta via `s.controller.GetAccountDelta(ctx, height)`, and overrides `blobDelta.Current.Accounts`/`blobDelta.Previous.Accounts` after `DeltaIndexerBlobs` runs.
- **Modify** `fsm/indexer_test.go`, `cmd/rpc/query_test.go` — update/add tests for the new signatures and the account-delta override.

---

## Task 1: `AccountChangeCollector` — classification logic

**Files:**
- Create: `fsm/account_change.go`
- Test: `fsm/account_change_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// fsm/account_change_test.go
package fsm

import (
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
	getPrev := func(key []byte) ([]byte, lib.ErrorI) { return nil, lib.ErrUnmarshal(nil) }
	c := NewAccountChangeCollector(getPrev)
	key := append(append([]byte{}, AccountPrefix()...), []byte("addr1")...)
	err := c.RecordSet(key, []byte("v1"))
	require.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./fsm/... -run TestAccountChangeCollector -v`
Expected: FAIL — `NewAccountChangeCollector`/`RecordSet`/`RecordDelete`/`Results` undefined.

- [ ] **Step 3: Write the implementation**

```go
// fsm/account_change.go
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
	address := append([]byte(nil), key[len(AccountPrefix()):]...)
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./fsm/... -run TestAccountChangeCollector -v`
Expected: PASS (7 tests)

- [ ] **Step 5: Commit**

```bash
git add fsm/account_change.go fsm/account_change_test.go
git commit -m "feat(fsm): add AccountChangeCollector for per-block account delta capture"
```

---

## Task 2: Hook `StateMachine.Set`/`Delete`

**Files:**
- Modify: `fsm/state.go:25-41` (struct), `fsm/state.go:658-666` (Set/Get/Delete), imports
- Test: `fsm/state_test.go`

- [ ] **Step 1: Write the failing test**

```go
// fsm/state_test.go — add near other StateMachine unit tests
func TestStateMachine_SetHooksAccountCollector(t *testing.T) {
	sm := newTestStateMachine(t) // existing test helper that constructs a StateMachine w/ in-memory store
	collector := NewAccountChangeCollector(sm.Get)
	sm.accountCollector = collector
	address := crypto.NewAddressFromBytes(newTestAddress(t).Bytes()) // existing test helper for a valid address
	require.NoError(t, sm.Set(KeyForAccount(address), []byte("v1")))
	added, changed, removed := collector.Results()
	require.Len(t, added, 1)
	require.Empty(t, changed)
	require.Empty(t, removed)
	require.Equal(t, address.Bytes(), added[0].Address)
}

func TestStateMachine_SetDoesNotHookNonAccountKeys(t *testing.T) {
	sm := newTestStateMachine(t)
	collector := NewAccountChangeCollector(sm.Get)
	sm.accountCollector = collector
	require.NoError(t, sm.Set(KeyForPool(1), []byte("v1")))
	added, changed, removed := collector.Results()
	require.Empty(t, added)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

func TestStateMachine_SetNilCollectorIsNoOp(t *testing.T) {
	sm := newTestStateMachine(t)
	require.Nil(t, sm.accountCollector)
	address := crypto.NewAddressFromBytes(newTestAddress(t).Bytes())
	require.NoError(t, sm.Set(KeyForAccount(address), []byte("v1"))) // must not panic
}
```

If `newTestStateMachine`/`newTestAddress` helpers don't already exist under those exact names, grep `fsm/*_test.go` for the actual construction helper used by nearby `StateMachine` tests (e.g. `newTestSM`, `NewTestStateMachine`) and use that name instead — do not invent a new helper without checking first.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./fsm/... -run TestStateMachine_Set -v`
Expected: FAIL — `sm.accountCollector` undefined (compile error).

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — fsm/state.go:1-14 (import block)
package fsm

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// NEW — fsm/state.go
package fsm

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)
```

```go
// CURRENT — fsm/state.go:25-41
type StateMachine struct {
	store lib.RWStoreI

	ProtocolVersion    uint64                                  // the version of the protocol this node is running
	NetworkID          uint32                                  // the id of the network this node is configured to be on
	height             uint64                                  // the 'version' of the state based on number of blocks currently on
	totalVDFIterations uint64                                  // the number of 'verifiable delay iterations' in the blockchain up to this version
	slashTracker       *SlashTracker                           // tracks total slashes across multiple blocks
	proposeVoteConfig  GovProposalVoteConfig                   // the configuration of how the state machine behaves with governance proposals
	Config             lib.Config                              // the main configuration as defined by the 'config.json' file
	Metrics            *lib.Metrics                            // the telemetry module
	events             *lib.EventsTracker                      // a simple event tracker for 'per-transaction' events
	log                lib.LoggerI                             // the logger for standard output and debugging
	cache              *cache                                  // the state machine cache
	LastValidatorSet   map[uint64]map[uint64]*lib.ValidatorSet // reference to the last validator set saved in the controller
	Plugin             *lib.Plugin                             // extensible plugin for the FSM
}

// NEW — fsm/state.go
type StateMachine struct {
	store lib.RWStoreI

	ProtocolVersion    uint64                                  // the version of the protocol this node is running
	NetworkID          uint32                                  // the id of the network this node is configured to be on
	height             uint64                                  // the 'version' of the state based on number of blocks currently on
	totalVDFIterations uint64                                  // the number of 'verifiable delay iterations' in the blockchain up to this version
	slashTracker       *SlashTracker                           // tracks total slashes across multiple blocks
	proposeVoteConfig  GovProposalVoteConfig                   // the configuration of how the state machine behaves with governance proposals
	Config             lib.Config                              // the main configuration as defined by the 'config.json' file
	Metrics            *lib.Metrics                            // the telemetry module
	events             *lib.EventsTracker                      // a simple event tracker for 'per-transaction' events
	log                lib.LoggerI                             // the logger for standard output and debugging
	cache              *cache                                  // the state machine cache
	LastValidatorSet   map[uint64]map[uint64]*lib.ValidatorSet // reference to the last validator set saved in the controller
	Plugin             *lib.Plugin                             // extensible plugin for the FSM
	accountCollector   *AccountChangeCollector                 // set only during an ApplyBlock call that wants account-touch capture
}
```

```go
// CURRENT — fsm/state.go:658-666
// Set() upserts a key-value pair under a key
func (s *StateMachine) Set(k, v []byte) (err lib.ErrorI) { return s.Store().Set(k, v) }

// Get() retrieves a key-value pair under a key
// NOTE: returns (nil, nil) if no value is found for that key
func (s *StateMachine) Get(key []byte) (bz []byte, err lib.ErrorI) { return s.Store().Get(key) }

// Delete() deletes a key-value pair under a key
func (s *StateMachine) Delete(key []byte) lib.ErrorI { return s.Store().Delete(key) }

// NEW — fsm/state.go
// Set() upserts a key-value pair under a key
func (s *StateMachine) Set(k, v []byte) (err lib.ErrorI) {
	if s.accountCollector != nil && bytes.HasPrefix(k, AccountPrefix()) {
		if err = s.accountCollector.RecordSet(k, v); err != nil {
			return err
		}
	}
	return s.Store().Set(k, v)
}

// Get() retrieves a key-value pair under a key
// NOTE: returns (nil, nil) if no value is found for that key
func (s *StateMachine) Get(key []byte) (bz []byte, err lib.ErrorI) { return s.Store().Get(key) }

// Delete() deletes a key-value pair under a key
func (s *StateMachine) Delete(key []byte) lib.ErrorI {
	if s.accountCollector != nil && bytes.HasPrefix(key, AccountPrefix()) {
		if err := s.accountCollector.RecordDelete(key); err != nil {
			return err
		}
	}
	return s.Store().Delete(key)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./fsm/... -run TestStateMachine_Set -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Run the full fsm test suite to check nothing else broke**

Run: `go test ./fsm/...`
Expected: PASS (nil `accountCollector` is the default zero value, so every existing `Set`/`Delete` caller is unaffected)

- [ ] **Step 6: Commit**

```bash
git add fsm/state.go fsm/state_test.go
git commit -m "feat(fsm): hook Set/Delete to capture account writes via AccountChangeCollector"
```

---

## Task 3: `ApplyBlock` signature — add `collector`/`skipRoot`

**Files:**
- Modify: `fsm/state.go:140-253` (`ApplyBlock`)
- Modify: `fsm/state_test.go:272`
- Modify: `controller/tx.go:309`
- Modify: `controller/block.go:524` (and its enclosing `ApplyAndValidateBlock`, see Task 4)

- [ ] **Step 1: Write the failing test**

```go
// fsm/state_test.go — add near the existing ApplyBlock test
func TestApplyBlock_SkipRootLeavesStateRootNil(t *testing.T) {
	sm, block := newTestApplyBlockFixture(t) // adapt to whatever the existing ApplyBlock test's setup helper is named — see fsm/state_test.go:~200-272 for the current fixture construction
	header, result, err := sm.ApplyBlock(context.Background(), block, false, nil, true)
	require.NoError(t, err)
	require.Empty(t, result.Failed)
	require.Nil(t, header.StateRoot)
	require.NotNil(t, header.Hash) // header must still hash successfully with an empty state root
}

func TestApplyBlock_CollectorCapturesTouchedAccounts(t *testing.T) {
	sm, block := newTestApplyBlockFixture(t) // same fixture as above; block must contain at least one send-type tx so an account is actually touched
	collector := NewAccountChangeCollector(sm.Get)
	_, result, err := sm.ApplyBlock(context.Background(), block, false, collector, false)
	require.NoError(t, err)
	require.Empty(t, result.Failed)
	added, changed, removed := collector.Results()
	require.NotEmpty(t, append(append(added, changed...), removed...), "collector should have captured at least one touched account")
}

func TestApplyBlock_NilCollectorSkipRootFalseIsUnchanged(t *testing.T) {
	sm, block := newTestApplyBlockFixture(t)
	header, result, err := sm.ApplyBlock(context.Background(), block, false, nil, false)
	require.NoError(t, err)
	require.Empty(t, result.Failed)
	require.NotNil(t, header.StateRoot) // unchanged default behavior
}
```

`newTestApplyBlockFixture` is a placeholder name for whatever the *existing* table-driven `ApplyBlock` test at `fsm/state_test.go:~200-272` uses to construct `sm`/`test.block` — before writing this step, read that test in full and reuse its actual setup (do not introduce a parallel fixture).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./fsm/... -run TestApplyBlock_ -v`
Expected: FAIL — compile error, `ApplyBlock` called with 5 args against the current 3-arg signature.

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — fsm/state.go:140
func (s *StateMachine) ApplyBlock(ctx context.Context, b *lib.Block, allowOversize bool) (header *lib.BlockHeader, r *lib.ApplyBlockResults, err lib.ErrorI) {

// NEW — fsm/state.go
// ApplyBlock processes a given block, updating the state machine's state accordingly
// collector, if non-nil, captures every account touched during this call (see
// AccountChangeCollector) — used for both live capture and on-demand replay.
// skipRoot, if true, skips store.Root() (the SMT/Merkle computation) — for throwaway
// replay calls whose caller never inspects header.StateRoot or commits the result.
func (s *StateMachine) ApplyBlock(ctx context.Context, b *lib.Block, allowOversize bool, collector *AccountChangeCollector, skipRoot bool) (header *lib.BlockHeader, r *lib.ApplyBlockResults, err lib.ErrorI) {
```

```go
// CURRENT — fsm/state.go:149-157 (right after the signature, before BeginBlock)
	// define vars to track the bytes of the transaction results and the size of a block
	r = new(lib.ApplyBlockResults)
	// cast the store to a StoreI, as only the writable store main 'apply blocks'
	store, ok := s.Store().(lib.StoreI)
	// casting fails, exit with error
	if !ok {
		return nil, nil, ErrWrongStoreType()
	}
	// automated execution at the 'beginning of a block'
	beginBlockStartTime := time.Now()

// NEW — fsm/state.go
	// define vars to track the bytes of the transaction results and the size of a block
	r = new(lib.ApplyBlockResults)
	// cast the store to a StoreI, as only the writable store main 'apply blocks'
	store, ok := s.Store().(lib.StoreI)
	// casting fails, exit with error
	if !ok {
		return nil, nil, ErrWrongStoreType()
	}
	// attach the account-change collector for the duration of this call only
	s.accountCollector = collector
	defer func() { s.accountCollector = nil }()
	// automated execution at the 'beginning of a block'
	beginBlockStartTime := time.Now()
```

```go
// CURRENT — fsm/state.go:203-218
	// calculate the merkle root of the state database to enable consensus on the result of the state after applying the block
	rootWasCached := false
	if cacheAwareStore, ok := store.(rootCacheStateStore); ok {
		rootWasCached = cacheAwareStore.IsRootCached()
	}
	rootStartTime := time.Time{}
	if !rootWasCached {
		rootStartTime = time.Now()
	}
	stateRoot, err := store.Root()
	if err != nil {
		return nil, nil, err
	}
	if !rootStartTime.IsZero() {
		s.Metrics.UpdateFSMApplyBlockRootTime(rootStartTime)
	}

// NEW — fsm/state.go
	// calculate the merkle root of the state database to enable consensus on the result of the state after applying the block
	// skipRoot=true is only ever passed by a throwaway replay call (see Task 6) whose
	// caller never commits and never inspects header.StateRoot/header.Hash for consensus
	// purposes — skipping this is what removes the dominant cost of a replay-only call.
	var stateRoot []byte
	if !skipRoot {
		rootWasCached := false
		if cacheAwareStore, ok := store.(rootCacheStateStore); ok {
			rootWasCached = cacheAwareStore.IsRootCached()
		}
		rootStartTime := time.Time{}
		if !rootWasCached {
			rootStartTime = time.Now()
		}
		stateRoot, err = store.Root()
		if err != nil {
			return nil, nil, err
		}
		if !rootStartTime.IsZero() {
			s.Metrics.UpdateFSMApplyBlockRootTime(rootStartTime)
		}
	}
```

Now update the three call sites:

```go
// CURRENT — fsm/state_test.go:272
			header, result, e := sm.ApplyBlock(context.Background(), test.block, false)

// NEW — fsm/state_test.go
			header, result, e := sm.ApplyBlock(context.Background(), test.block, false, nil, false)
```

```go
// CURRENT — controller/tx.go:309
	block.BlockHeader, result, err = m.FSM.ApplyBlock(ctx, block, true)

// NEW — controller/tx.go
	block.BlockHeader, result, err = m.FSM.ApplyBlock(ctx, block, true, nil, false)
```

`controller/block.go:524` is updated in Task 4 together with `ApplyAndValidateBlock`'s new `collector` parameter — do not edit it here in isolation.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./fsm/... -run TestApplyBlock_ -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Build the whole repo to catch any other call sites**

Run: `go build ./...`
Expected: fails only on `controller/block.go:524` (fixed in Task 4). If it fails anywhere else, that's a call site the earlier grep missed — read it, decide `nil, false` (no behavior change) unless it's part of this feature's live/replay wiring, and add it to this task's diff before proceeding.

- [ ] **Step 6: Commit**

```bash
git add fsm/state.go fsm/state_test.go controller/tx.go
git commit -m "feat(fsm): add collector/skipRoot params to ApplyBlock"
```

---

## Task 4: `ApplyAndValidateBlock` — thread `collector` through, fix build

**Files:**
- Modify: `controller/block.go:512-563` (`ApplyAndValidateBlock`), and its two call sites at `controller/block.go:272` (`CommitCertificate`) and `controller/block.go:387` (`CommitCertificateParallel`)

- [ ] **Step 1: Write the failing test**

```go
// controller/block_test.go — add near existing ApplyAndValidateBlock coverage, reusing
// whatever test-controller fixture those tests already use (grep for
// "func TestApplyAndValidateBlock" or "func Test.*CommitCertificate" first and match its
// existing setup pattern — do not invent a new fixture)
func TestApplyAndValidateBlock_PassesCollectorThrough(t *testing.T) {
	c, block := newTestControllerWithBlock(t) // match existing fixture helper name
	collector := fsm.NewAccountChangeCollector(c.FSM.Get)
	_, err := c.ApplyAndValidateBlock(block, true, collector)
	require.NoError(t, err)
	added, changed, removed := collector.Results()
	require.NotEmpty(t, append(append(added, changed...), removed...))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./controller/... -run TestApplyAndValidateBlock_PassesCollectorThrough -v`
Expected: FAIL — compile error, `ApplyAndValidateBlock` doesn't take a 3rd argument yet.

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — controller/block.go:513
func (c *Controller) ApplyAndValidateBlock(block *lib.Block, commit bool) (b *lib.BlockResult, err lib.ErrorI) {

// NEW — controller/block.go
// ApplyAndValidateBlock() plays the block against the state machine which returns a result
// that is compared against the candidate block header. collector, if non-nil, is passed
// through to ApplyBlock to capture touched accounts for the live indexer-blob tip cache.
func (c *Controller) ApplyAndValidateBlock(block *lib.Block, commit bool, collector *fsm.AccountChangeCollector) (b *lib.BlockResult, err lib.ErrorI) {
```

```go
// CURRENT — controller/block.go:524
	compare, results, err := c.FSM.ApplyBlock(context.Background(), block, false)

// NEW — controller/block.go
	compare, results, err := c.FSM.ApplyBlock(context.Background(), block, false, collector, false)
```

```go
// CURRENT — controller/block.go:272 (inside CommitCertificate)
		// apply the block against the state machine
		blockResult, err = c.ApplyAndValidateBlock(block, true)
		if err != nil {
			// exit with error
			return
		}

// NEW — controller/block.go
		// apply the block against the state machine, attaching a live account-change
		// collector when this node is configured to serve indexer blobs and isn't syncing
		// (see Task 5 for accountCollectorForLiveCommit and Task 6 for tip-cache populate)
		liveCollector := c.accountCollectorForLiveCommit(syncing)
		blockResult, err = c.ApplyAndValidateBlock(block, true, liveCollector)
		if err != nil {
			// exit with error
			return
		}
```

```go
// CURRENT — controller/block.go:387 (inside CommitCertificateParallel)
		if blockResult, err = c.ApplyAndValidateBlock(block, true); err != nil {
			// exit with error
			return
		}

// NEW — controller/block.go
		liveCollector := c.accountCollectorForLiveCommit(syncing)
		if blockResult, err = c.ApplyAndValidateBlock(block, true, liveCollector); err != nil {
			// exit with error
			return
		}
```

`accountCollectorForLiveCommit` doesn't exist yet — it's defined in Task 5. This task will not build in isolation; that's expected and resolved by the end of Task 5. Do not skip Step 4 below for this reason — Step 4 for this task runs against the *unit test*, which only needs `ApplyAndValidateBlock`'s new signature, not `accountCollectorForLiveCommit`. Stub it minimally so `controller` package compiles for this task's own test run:

```go
// TEMPORARY stub for this task only — replaced by Task 5's real implementation.
// controller/block.go — add near ApplyAndValidateBlock
func (c *Controller) accountCollectorForLiveCommit(syncing bool) *fsm.AccountChangeCollector {
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/... -run TestApplyAndValidateBlock_PassesCollectorThrough -v`
Expected: PASS

- [ ] **Step 5: Build the whole repo**

Run: `go build ./...`
Expected: PASS (no remaining `ApplyBlock`/`ApplyAndValidateBlock` call-site mismatches)

- [ ] **Step 6: Commit**

```bash
git add controller/block.go controller/block_test.go
git commit -m "feat(controller): thread AccountChangeCollector through ApplyAndValidateBlock"
```

---

## Task 5: Config flag — `ServeIndexerBlobsLive`

**Files:**
- Modify: `lib/config.go:121-164`

- [ ] **Step 1: Write the failing test**

```go
// lib/config_test.go — add near other DefaultRPCConfig assertions (grep for
// "func TestDefaultRPCConfig" first; if it doesn't exist, add a new small test)
func TestDefaultRPCConfig_ServeIndexerBlobsLiveDefaultsTrue(t *testing.T) {
	cfg := DefaultRPCConfig()
	require.True(t, cfg.ServeIndexerBlobsLive)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lib/... -run TestDefaultRPCConfig_ServeIndexerBlobsLiveDefaultsTrue -v`
Expected: FAIL — `ServeIndexerBlobsLive` undefined on `RPCConfig`.

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — lib/config.go:121-137
type RPCConfig struct {
	WalletPort                 string `json:"walletPort"`                 // the port where the web wallet is hosted
	ExplorerPort               string `json:"explorerPort"`               // the port where the block explorer is hosted
	RPCPort                    string `json:"rpcPort"`                    // the port where the rpc server is hosted
	AdminPort                  string `json:"adminPort"`                  // the port where the admin rpc server is hosted
	ProfilingPort              string `json:"profilingPort"`              // the port where the pprof profiling server is hosted
	RPCUrl                     string `json:"rpcURL"`                     // the url where the rpc server is hosted
	AdminRPCUrl                string `json:"adminRPCUrl"`                // the url where the admin rpc server is hosted
	TimeoutS                   int    `json:"timeoutS"`                   // the rpc request timeout in seconds
	IndexerBlobCacheEntries    int    `json:"indexerBlobCacheEntries"`    // number of cached indexer blobs to keep in memory
	MaxRCSubscribers           int    `json:"maxRCSubscribers"`           // max total root-chain subscribers
	MaxRCSubscribersPerChain   int    `json:"maxRCSubscribersPerChain"`   // max root-chain subscribers per chain id
	RCSubscriberReadLimitBytes int64  `json:"rcSubscriberReadLimitBytes"` // max bytes allowed in a single ws message from a subscriber
	RCSubscriberWriteTimeoutMS int    `json:"rcSubscriberWriteTimeoutMS"` // ws write timeout for publishing root-chain info
	RCSubscriberPongWaitS      int    `json:"rcSubscriberPongWaitS"`      // time to wait for pong responses
	RCSubscriberPingPeriodS    int    `json:"rcSubscriberPingPeriodS"`    // how often to ping subscribers
}

// NEW — lib/config.go
type RPCConfig struct {
	WalletPort                 string `json:"walletPort"`                 // the port where the web wallet is hosted
	ExplorerPort               string `json:"explorerPort"`               // the port where the block explorer is hosted
	RPCPort                    string `json:"rpcPort"`                    // the port where the rpc server is hosted
	AdminPort                  string `json:"adminPort"`                  // the port where the admin rpc server is hosted
	ProfilingPort              string `json:"profilingPort"`              // the port where the pprof profiling server is hosted
	RPCUrl                     string `json:"rpcURL"`                     // the url where the rpc server is hosted
	AdminRPCUrl                string `json:"adminRPCUrl"`                // the url where the admin rpc server is hosted
	TimeoutS                   int    `json:"timeoutS"`                   // the rpc request timeout in seconds
	IndexerBlobCacheEntries    int    `json:"indexerBlobCacheEntries"`    // number of cached indexer blobs to keep in memory
	ServeIndexerBlobsLive      bool   `json:"serveIndexerBlobsLive"`      // capture account changes during live block commit for the indexer-blobs tip cache, instead of only on-demand replay
	MaxRCSubscribers           int    `json:"maxRCSubscribers"`           // max total root-chain subscribers
	MaxRCSubscribersPerChain   int    `json:"maxRCSubscribersPerChain"`   // max root-chain subscribers per chain id
	RCSubscriberReadLimitBytes int64  `json:"rcSubscriberReadLimitBytes"` // max bytes allowed in a single ws message from a subscriber
	RCSubscriberWriteTimeoutMS int    `json:"rcSubscriberWriteTimeoutMS"` // ws write timeout for publishing root-chain info
	RCSubscriberPongWaitS      int    `json:"rcSubscriberPongWaitS"`      // time to wait for pong responses
	RCSubscriberPingPeriodS    int    `json:"rcSubscriberPingPeriodS"`    // how often to ping subscribers
}
```

```go
// CURRENT — lib/config.go:145-164 (DefaultRPCConfig)
func DefaultRPCConfig() RPCConfig {
	return RPCConfig{
		WalletPort:                 "50000",                    // find the wallet on localhost:50000
		ExplorerPort:               "50001",                    // find the explorer on localhost:50001
		RPCPort:                    "50002",                    // the rpc is served on localhost:50002
		AdminPort:                  "50003",                    // the admin rpc is served on localhost:50003
		ProfilingPort:              "6060",                     // the pprof profiling server is served on localhost:6060
		RPCUrl:                     "http://localhost:50002",   // use a local rpc by default
		AdminRPCUrl:                "http://localhost:50003",   // use a local admin rpc by default
		TimeoutS:                   30,                         // the rpc timeout is 30 seconds
		IndexerBlobCacheEntries:    64,                         // cache the most recent indexer blobs
		MaxRCSubscribers:           512,                        // limit total root-chain subscribers
		MaxRCSubscribersPerChain:   128,                        // limit subscribers per chain id
		RCSubscriberReadLimitBytes: int64(64 * units.Kilobyte), // cap inbound ws message sizes
		RCSubscriberWriteTimeoutMS: 10000,                      // 10s write deadline for publishes
		RCSubscriberPongWaitS:      60,                         // 60s pong wait
		RCSubscriberPingPeriodS:    50,                         // 50s ping interval
	}
}

// NEW — lib/config.go
func DefaultRPCConfig() RPCConfig {
	return RPCConfig{
		WalletPort:                 "50000",                    // find the wallet on localhost:50000
		ExplorerPort:               "50001",                    // find the explorer on localhost:50001
		RPCPort:                    "50002",                    // the rpc is served on localhost:50002
		AdminPort:                  "50003",                    // the admin rpc is served on localhost:50003
		ProfilingPort:              "6060",                     // the pprof profiling server is served on localhost:6060
		RPCUrl:                     "http://localhost:50002",   // use a local rpc by default
		AdminRPCUrl:                "http://localhost:50003",   // use a local admin rpc by default
		TimeoutS:                   30,                         // the rpc timeout is 30 seconds
		IndexerBlobCacheEntries:    64,                         // cache the most recent indexer blobs
		ServeIndexerBlobsLive:      true,                       // capture account changes live by default
		MaxRCSubscribers:           512,                        // limit total root-chain subscribers
		MaxRCSubscribersPerChain:   128,                        // limit subscribers per chain id
		RCSubscriberReadLimitBytes: int64(64 * units.Kilobyte), // cap inbound ws message sizes
		RCSubscriberWriteTimeoutMS: 10000,                      // 10s write deadline for publishes
		RCSubscriberPongWaitS:      60,                         // 60s pong wait
		RCSubscriberPingPeriodS:    50,                         // 50s ping interval
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./lib/... -run TestDefaultRPCConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add lib/config.go lib/config_test.go
git commit -m "feat(config): add ServeIndexerBlobsLive RPC config flag"
```

---

## Task 6: Tip cache + live capture wiring

**Files:**
- Create: `controller/account_delta_cache.go`
- Create: `controller/account_delta_cache_test.go`
- Modify: `controller/block.go` — replace Task 4's stub `accountCollectorForLiveCommit`, populate the cache in `commitToStore`

- [ ] **Step 1: Write the failing tests**

```go
// controller/account_delta_cache_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/... -run TestAccountDeltaCache -v`
Expected: FAIL — `newAccountDeltaCache`/`accountDeltaEntry` undefined.

- [ ] **Step 3: Write the implementation**

```go
// controller/account_delta_cache.go
package controller

import (
	"sync"

	"github.com/canopy-network/canopy/fsm"
)

// defaultAccountDeltaCacheEntries bounds the in-memory tip cache. It only needs to
// cover the handful of most-recent heights an indexer polling live blocks would
// realistically request before falling back to on-demand replay.
const defaultAccountDeltaCacheEntries = 16

// accountDeltaEntry is one height's classified account changes, produced either by
// the live AccountChangeCollector attached during commit, or by an on-demand replay.
type accountDeltaEntry struct {
	added, changed, removed []*fsm.AccountChangeEntry
}

// accountDeltaCache is a small fixed-capacity FIFO cache of recent heights' account
// deltas. It is a pure accelerator with no correctness dependency: a reorged-out
// height's cached entry is simply never requested again, so no invalidation logic
// is needed.
type accountDeltaCache struct {
	mu         sync.RWMutex
	maxEntries int
	entries    map[uint64]*accountDeltaEntry
	order      []uint64
}

func newAccountDeltaCache(maxEntries int) *accountDeltaCache {
	if maxEntries <= 0 {
		maxEntries = defaultAccountDeltaCacheEntries
	}
	return &accountDeltaCache{
		maxEntries: maxEntries,
		entries:    make(map[uint64]*accountDeltaEntry),
		order:      make([]uint64, 0, maxEntries),
	}
}

func (c *accountDeltaCache) get(height uint64) (*accountDeltaEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[height]
	return entry, ok
}

func (c *accountDeltaCache) put(height uint64, entry *accountDeltaEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[height]; ok {
		c.entries[height] = entry
		return
	}
	c.entries[height] = entry
	c.order = append(c.order, height)
	if len(c.order) > c.maxEntries {
		evictHeight := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, evictHeight)
	}
}
```

Now wire it into `Controller`. First confirm the `Controller` struct's field block (grep `type Controller struct` in `controller/controller.go` before editing) and add `accountDeltaCache *accountDeltaCache`, initialized in the constructor alongside other fields. Since the exact constructor body wasn't part of this plan's research, the engineer must:

```bash
grep -n "type Controller struct" -A 40 controller/controller.go
grep -n "func New(" controller/controller.go
```

Add the field to the struct and `accountDeltaCache: newAccountDeltaCache(0),` (0 → default 16) to the constructor's returned `&Controller{...}` literal, matching the style of the other fields already initialized there.

Replace Task 4's stub:

```go
// CURRENT — controller/block.go (Task 4's temporary stub)
func (c *Controller) accountCollectorForLiveCommit(syncing bool) *fsm.AccountChangeCollector {
	return nil
}

// NEW — controller/block.go
// accountCollectorForLiveCommit returns a fresh AccountChangeCollector for the live
// commit path only — never during sync/replay, where nothing is polling yet and
// eagerly classifying every replayed block would be pure waste.
func (c *Controller) accountCollectorForLiveCommit(syncing bool) *fsm.AccountChangeCollector {
	if syncing || !c.Config.RPCConfig.ServeIndexerBlobsLive {
		return nil
	}
	return fsm.NewAccountChangeCollector(c.FSM.Get)
}
```

Populate the cache right after commit succeeds. `commitToStore` is the single primitive both `CommitCertificate` and `CommitCertificateParallel` converge on:

```go
// CURRENT — controller/block.go:494-508
// commitToStore() atomically writes the ephemeral batch to disk and sets up the FSM for the next height
func (c *Controller) commitToStore(storeI lib.StoreI, qc *lib.QuorumCertificate, height uint64) (err lib.ErrorI) {
	// log the start of the commit
	c.log.Debug("Committing to store")
	// atomically write all from the ephemeral database batch to the actual database
	if _, err = storeI.Commit(); err != nil {
		// exit with error
		return err
	}
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Plugin, c.Metrics, c.log)
	return err
}

// NEW — controller/block.go
// commitToStore() atomically writes the ephemeral batch to disk and sets up the FSM for the next height
func (c *Controller) commitToStore(storeI lib.StoreI, qc *lib.QuorumCertificate, height uint64, collector *fsm.AccountChangeCollector) (err lib.ErrorI) {
	// log the start of the commit
	c.log.Debug("Committing to store")
	// atomically write all from the ephemeral database batch to the actual database
	if _, err = storeI.Commit(); err != nil {
		// exit with error
		return err
	}
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Plugin, c.Metrics, c.log)
	if err != nil {
		return err
	}
	if collector != nil {
		added, changed, removed := collector.Results()
		c.accountDeltaCache.put(height, &accountDeltaEntry{added: added, changed: changed, removed: removed})
	}
	return nil
}
```

`commitToStore` gains a `collector` parameter — thread it from its two call sites:

```go
// CURRENT — controller/block.go:306 (inside CommitCertificate, non-parallel path)
	if _, err = storeI.Commit(); err != nil {
		// exit with error
		return err
	}
```

This non-parallel `CommitCertificate` path (block.go:248-358) doesn't call `commitToStore` today — it inlines the commit (line 306) and FSM setup (line 313) directly rather than sharing `commitToStore` with the parallel path. Apply the equivalent change inline here instead of introducing a shared call: after line 313's `c.FSM, err = fsm.New(...)` and its error check, add:

```go
// CURRENT — controller/block.go:311-317
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), block.BlockHeader.Height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Plugin, c.Metrics, c.log)
	if err != nil {
		// exit with error
		return err
	}

// NEW — controller/block.go
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), block.BlockHeader.Height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Plugin, c.Metrics, c.log)
	if err != nil {
		// exit with error
		return err
	}
	if liveCollector != nil {
		added, changed, removed := liveCollector.Results()
		c.accountDeltaCache.put(block.BlockHeader.Height, &accountDeltaEntry{added: added, changed: changed, removed: removed})
	}
```

(`liveCollector` here is the same variable introduced in Task 4's edit to this function.)

```go
// CURRENT — controller/block.go:409-410 (inside CommitCertificateParallel, sync branch)
	if syncing {
		return c.commitToStore(storeI, qc, block.BlockHeader.Height)
	}

// NEW — controller/block.go
	if syncing {
		return c.commitToStore(storeI, qc, block.BlockHeader.Height, nil) // never live during sync
	}
```

```go
// CURRENT — controller/block.go:430 (inside CommitCertificateParallel, live branch, eg.Go closure)
		if err = c.commitToStore(storeI, qc, block.BlockHeader.Height); err != nil {
			// exit with error
			return err
		}

// NEW — controller/block.go
		if err = c.commitToStore(storeI, qc, block.BlockHeader.Height, liveCollector); err != nil {
			// exit with error
			return err
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/... -run TestAccountDeltaCache -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Build and run the full controller suite**

Run: `go build ./... && go test ./controller/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add controller/account_delta_cache.go controller/account_delta_cache_test.go controller/block.go controller/controller.go
git commit -m "feat(controller): populate account-delta tip cache on live commit"
```

---

## Task 7: `Controller.GetAccountDelta` — tip-cache hit or on-demand replay

**Files:**
- Modify: `controller/block.go` (or a new `controller/account_delta.go` if `block.go` is already large — check its current line count first with `wc -l controller/block.go`; if it's over ~600 lines, create the new file instead to keep responsibilities separated)
- Test: alongside whichever file is chosen

- [ ] **Step 1: Write the failing test**

```go
// controller/account_delta_test.go (or block_test.go, matching Step 3's file choice)
func TestGetAccountDelta_TipCacheHit(t *testing.T) {
	c := newTestController(t) // match whatever existing controller test fixture is used elsewhere in this package
	c.accountDeltaCache.put(5, &accountDeltaEntry{added: []*fsm.AccountChangeEntry{{Address: []byte("a")}}})
	added, changed, removed, err := c.GetAccountDelta(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, added, 1)
	require.Empty(t, changed)
	require.Empty(t, removed)
}

func TestGetAccountDelta_ReplayOnCacheMiss(t *testing.T) {
	c, block := newTestControllerWithCommittedBlock(t) // committed at some height H containing a send tx, matching existing fixtures
	// no cache entry for H — force the replay path
	added, changed, removed, err := c.GetAccountDelta(context.Background(), block.BlockHeader.Height)
	require.NoError(t, err)
	require.NotEmpty(t, append(append(added, changed...), removed...))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/... -run TestGetAccountDelta -v`
Expected: FAIL — `GetAccountDelta` undefined.

- [ ] **Step 3: Write the implementation**

```go
// controller/account_delta.go
package controller

import (
	"context"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// GetAccountDelta returns the accounts added/changed/removed by the block at height.
// It serves from the live tip cache when available; otherwise it replays ApplyBlock
// with skipRoot=true against a TimeMachine(height-1) snapshot, discarding all writes
// without ever committing — the same no-commit pattern already used by mempool checks
// (controller/tx.go).
func (c *Controller) GetAccountDelta(ctx context.Context, height uint64) (added, changed, removed []*fsm.AccountChangeEntry, err lib.ErrorI) {
	if entry, ok := c.accountDeltaCache.get(height); ok {
		return entry.added, entry.changed, entry.removed, nil
	}
	store, ok := c.FSM.Store().(lib.StoreI)
	if !ok {
		return nil, nil, nil, fsm.ErrWrongStoreType()
	}
	block, loadErr := store.GetBlockByHeight(height)
	if loadErr != nil {
		return nil, nil, nil, loadErr
	}
	if block == nil || block.BlockHeader == nil {
		return nil, nil, nil, lib.ErrNilBlockHeader()
	}
	replayFSM, tmErr := c.FSM.TimeMachine(height - 1)
	if tmErr != nil {
		return nil, nil, nil, tmErr
	}
	if replayFSM != c.FSM {
		defer replayFSM.Discard()
	}
	collector := fsm.NewAccountChangeCollector(replayFSM.Get)
	_, applyResult, applyErr := replayFSM.ApplyBlock(ctx, block, false, collector, true)
	if applyErr != nil {
		return nil, nil, nil, applyErr
	}
	if len(applyResult.Failed) != 0 {
		return nil, nil, nil, lib.ErrFailedTransactions()
	}
	added, changed, removed = collector.Results()
	return added, changed, removed, nil
}
```

Before finalizing this step, verify the exact names/signatures of `lib.StoreI.GetBlockByHeight`, `StateMachine.Discard`, `ErrWrongStoreType`, and `lib.ErrNilBlockHeader`/`lib.ErrFailedTransactions` against the current source (`grep -n "func.*GetBlockByHeight\|func.*Discard\|func ErrWrongStoreType\|func ErrNilBlockHeader\|func ErrFailedTransactions"` across `lib/` and `fsm/`) — this plan's earlier research confirmed `GetBlockByHeight` and `ErrNilBlockHeader`/`ErrWrongStoreType` are used exactly this way inside `fsm/indexer.go` (`st.GetBlockByHeight(blockHeight)`, `ErrNilBlockHeader()`, `ErrWrongStoreType()`), so these should match, but confirm `Discard`'s exact receiver/behavior (referenced in `fsm/indexer.go:55`: `defer sm.Discard()`) before relying on it here.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/... -run TestGetAccountDelta -v`
Expected: PASS

- [ ] **Step 5: Build and run the full controller suite**

Run: `go build ./... && go test ./controller/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add controller/account_delta.go controller/account_delta_test.go
git commit -m "feat(controller): add GetAccountDelta with tip-cache/replay fallback"
```

---

## Task 8: Wire into `IndexerBlob` — skip the full account scan

**Files:**
- Modify: `fsm/indexer.go:16-32` (`IndexerBlobs` wrapper), `fsm/indexer.go:34-79` (`IndexerBlob`, accounts step only)
- Modify: `fsm/state_test.go` or `fsm/indexer_test.go` — any existing direct callers of `IndexerBlob`/`IndexerBlobs`

- [ ] **Step 1: Write the failing test**

```go
// fsm/indexer_test.go — add near existing coverage
func TestIndexerBlob_SkipAccountsLeavesAccountsNil(t *testing.T) {
	sm := newTestStateMachineWithCommittedBlocks(t, 3) // match whatever existing fixture builds a multi-block committed chain for indexer tests — see setup used by TestDeltaIndexerBlobs_* in this same file
	blob, err := sm.IndexerBlob(context.Background(), 3, true)
	require.NoError(t, err)
	require.Nil(t, blob.Accounts)
}

func TestIndexerBlob_NoSkipStillScansAccounts(t *testing.T) {
	sm := newTestStateMachineWithCommittedBlocks(t, 3)
	blob, err := sm.IndexerBlob(context.Background(), 3, false)
	require.NoError(t, err)
	// unchanged existing behavior — full scan still populates Accounts
	require.NotNil(t, blob) // exact non-nil-Accounts assertion depends on whether fixture accounts exist; adapt to match existing TestDeltaIndexerBlobs_* fixtures' expectations
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./fsm/... -run TestIndexerBlob_ -v`
Expected: FAIL — compile error, `IndexerBlob` called with 3 args against the current 2-arg signature.

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — fsm/indexer.go:34-35
// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlob(ctx context.Context, height uint64) (b *IndexerBlob, err lib.ErrorI) {

// NEW — fsm/indexer.go
// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer.
// skipAccounts, if true, leaves Accounts nil instead of doing the full
// IterateAndAppend(AccountPrefix()) scan — used when the caller (IndexerBlobsCached)
// is going to source the account delta separately via Controller.GetAccountDelta.
func (s *StateMachine) IndexerBlob(ctx context.Context, height uint64, skipAccounts bool) (b *IndexerBlob, err lib.ErrorI) {
```

```go
// CURRENT — fsm/indexer.go:72-79
	// use sm for consistent snapshot reads at the requested height
	// retrieve the accounts
	stepStart = time.Now()
	accounts, err := sm.IterateAndAppend(ctx, AccountPrefix())
	s.Metrics.ObserveIndexerBlobStep("accounts_iterate", stepStart)
	if err != nil {
		return nil, err
	}

// NEW — fsm/indexer.go
	// use sm for consistent snapshot reads at the requested height
	// retrieve the accounts, unless the caller is sourcing them separately (see skipAccounts doc)
	var accounts [][]byte
	if !skipAccounts {
		stepStart = time.Now()
		accounts, err = sm.IterateAndAppend(ctx, AccountPrefix())
		s.Metrics.ObserveIndexerBlobStep("accounts_iterate", stepStart)
		if err != nil {
			return nil, err
		}
	}
```

```go
// CURRENT — fsm/indexer.go:16-32
func (s *StateMachine) IndexerBlobs(ctx context.Context, height uint64) (b *IndexerBlobs, err lib.ErrorI) {
	b = &IndexerBlobs{}
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		b.Previous, err = s.IndexerBlob(ctx, height-1)
		if err != nil {
			return nil, err
		}
	}
	b.Current, err = s.IndexerBlob(ctx, height)
	if err != nil {
		return nil, err
	}
	return
}

// NEW — fsm/indexer.go
func (s *StateMachine) IndexerBlobs(ctx context.Context, height uint64) (b *IndexerBlobs, err lib.ErrorI) {
	b = &IndexerBlobs{}
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		b.Previous, err = s.IndexerBlob(ctx, height-1, false)
		if err != nil {
			return nil, err
		}
	}
	b.Current, err = s.IndexerBlob(ctx, height, false)
	if err != nil {
		return nil, err
	}
	return
}
```

Before running tests, grep every other call site of `.IndexerBlob(` (not `IndexerBlobs`) across the repo — this plan's research found it's called from `fsm/indexer.go:22,27` (just updated above) and `cmd/rpc/query.go:638,652` (updated in Task 9). Re-run the grep now in case anything else references it (e.g. other test files) and update those call sites to pass `false` (preserve current full-scan behavior) unless they're part of this feature's own wiring.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./fsm/... -run TestIndexerBlob_ -v`
Expected: PASS

- [ ] **Step 5: Build and run the full fsm suite**

Run: `go build ./... && go test ./fsm/...`
Expected: fails only on `cmd/rpc/query.go` (fixed in Task 9)

- [ ] **Step 6: Commit**

```bash
git add fsm/indexer.go fsm/indexer_test.go
git commit -m "feat(fsm): add skipAccounts to IndexerBlob"
```

---

## Task 9: Wire into `IndexerBlobsCached` — override Accounts with the fast path

**Files:**
- Modify: `cmd/rpc/query.go:617-695` (`IndexerBlobsCached`)
- Test: `cmd/rpc/query_test.go`

- [ ] **Step 1: Write the failing test**

```go
// cmd/rpc/query_test.go — add near existing IndexerBlobsCached coverage
func TestIndexerBlobsCached_UsesAccountDeltaFastPath(t *testing.T) {
	s := newTestServerWithCommittedBlocks(t, 3) // match existing fixture used by TestIndexerBlobsCached_* in this file
	deltaBlobs, _, err := s.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, deltaBlobs.Current)
	// Accounts on the response must come from GetAccountDelta, not a full scan —
	// assert against whatever the test fixture's known committed accounts are,
	// matching the assertion style of the existing TestIndexerBlobsCached_* tests
	// in this file (read them first to match expected fixture values exactly).
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/... -run TestIndexerBlobsCached_UsesAccountDeltaFastPath -v`
Expected: FAIL — either a compile error (if `IndexerBlob` call sites in this file aren't updated yet) or an assertion failure (accounts still come from the old scan).

- [ ] **Step 3: Write the implementation**

```go
// CURRENT — cmd/rpc/query.go:636-657
	coldStart := time.Now()
	defer s.controller.Metrics.RecordIndexerBlobCacheMiss(coldStart)
	current, err := s.controller.FSM.IndexerBlob(ctx, height)
	if err != nil {
		return nil, nil, err
	}

	var previous *fsm.IndexerBlob
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		if cachedPrev, ok := s.indexerBlobCache.getCurrent(height - 1); ok {
			s.controller.Metrics.RecordIndexerBlobPreviousReuseHit()
			previous = cachedPrev
		} else {
			s.controller.Metrics.RecordIndexerBlobPreviousReuseMiss()
			prev, prevErr := s.controller.FSM.IndexerBlob(ctx, height-1)
			if prevErr != nil {
				return nil, nil, prevErr
			}
			previous = prev
		}
	}

// NEW — cmd/rpc/query.go
	coldStart := time.Now()
	defer s.controller.Metrics.RecordIndexerBlobCacheMiss(coldStart)
	current, err := s.controller.FSM.IndexerBlob(ctx, height, true)
	if err != nil {
		return nil, nil, err
	}

	var previous *fsm.IndexerBlob
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		if cachedPrev, ok := s.indexerBlobCache.getCurrent(height - 1); ok {
			s.controller.Metrics.RecordIndexerBlobPreviousReuseHit()
			previous = cachedPrev
		} else {
			s.controller.Metrics.RecordIndexerBlobPreviousReuseMiss()
			prev, prevErr := s.controller.FSM.IndexerBlob(ctx, height-1, true)
			if prevErr != nil {
				return nil, nil, prevErr
			}
			previous = prev
		}
	}

	accountDeltaStart := time.Now()
	added, changed, removed, deltaErr := s.controller.GetAccountDelta(ctx, height)
	s.controller.Metrics.ObserveIndexerBlobStep("account_delta_get", accountDeltaStart)
	if deltaErr != nil {
		return nil, nil, deltaErr
	}
```

Then, right after `blobDelta, err := fsm.DeltaIndexerBlobs(blobs)` succeeds, override the account fields it computed (which will be empty, since both `current.Accounts` and `previous.Accounts` were left nil above) with the fast-path result. `added`/`changed` become `Current.Accounts` (their final marshalled bytes — matches `DeltaIndexerBlobs`'s existing convention that removed accounts are omitted from `Current.Accounts` entirely); `changed`/`removed` become `Previous.Accounts` (their pre-block bytes):

```go
// CURRENT — cmd/rpc/query.go:673-679
	deltaComputeStart := time.Now()
	blobDelta, err := fsm.DeltaIndexerBlobs(blobs)
	s.controller.Metrics.ObserveIndexerBlobStep("delta_compute", deltaComputeStart)
	if err != nil {
		return nil, nil, err
	}

// NEW — cmd/rpc/query.go
	deltaComputeStart := time.Now()
	blobDelta, err := fsm.DeltaIndexerBlobs(blobs)
	s.controller.Metrics.ObserveIndexerBlobStep("delta_compute", deltaComputeStart)
	if err != nil {
		return nil, nil, err
	}
	if blobDelta.Current != nil {
		blobDelta.Current.Accounts = accountBytes(added, changed, true) // added + changed final values
	}
	if blobDelta.Previous != nil {
		blobDelta.Previous.Accounts = accountBytes(changed, removed, false) // changed + removed previous values
	}
```

Add the small helper next to `IndexerBlobsCached`:

```go
// cmd/rpc/query.go — new helper near IndexerBlobsCached
// accountBytes builds the [][]byte wire shape IndexerBlob.Accounts already uses,
// from AccountChangeCollector entries. useFinal selects FinalValue (for the current
// side) vs PrevValue (for the previous side) — matches DeltaIndexerBlobs's existing
// convention where an added account has no previous-side entry and a removed
// account has no current-side entry.
func accountBytes(a, b []*fsm.AccountChangeEntry, useFinal bool) [][]byte {
	out := make([][]byte, 0, len(a)+len(b))
	for _, e := range append(append([]*fsm.AccountChangeEntry{}, a...), b...) {
		if useFinal {
			if e.FinalValue != nil {
				out = append(out, e.FinalValue)
			}
		} else {
			if e.PrevValue != nil {
				out = append(out, e.PrevValue)
			}
		}
	}
	return out
}
```

Before finalizing, re-check `fsm.IndexerBlob`/`fsm.IndexerBlobs`/`fsm.DeltaIndexerBlobs`'s exact field types for `Current`/`Previous`/`Accounts` (`grep -n "type IndexerBlob struct\|type IndexerBlobs struct" fsm/*.go` — these are protobuf-generated, likely in a `*.pb.go` file) to confirm `Accounts` really is `[][]byte` and not a different wrapper type; this plan's research treated it as `[][]byte` based on `Accounts: accounts` where `accounts, err := sm.IterateAndAppend(...)`, but confirm `IterateAndAppend`'s return type directly before writing `accountBytes`'s signature.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rpc/... -run TestIndexerBlobsCached_UsesAccountDeltaFastPath -v`
Expected: PASS

- [ ] **Step 5: Build and run the full test suite**

Run: `go build ./... && go test ./...`
Expected: PASS. Pay particular attention to any other pre-existing `cmd/rpc/query_test.go` tests that assert exact `Accounts` byte contents from the old full-scan path (e.g. `TestIndexerBlobs_IgnoresLegacyDeltaField`, `TestIndexerBlobsCached_CachesDeltaResponsesOnly`, `TestIndexerBlobsCached_RetainsOnlyLatestFullSnapshot`) — these may need their expected `Accounts` values updated to route through `GetAccountDelta`'s replay path instead, since the fixture's committed blocks won't have a live tip-cache entry unless the test explicitly commits them through the live path.

- [ ] **Step 6: Commit**

```bash
git add cmd/rpc/query.go cmd/rpc/query_test.go
git commit -m "feat(rpc): source IndexerBlobsCached account deltas from the fast path"
```

---

## Task 10: Differential regression test — old scan vs new path agree

Note: the spec's testing section named the golden-reference dataset infra (`task:1371`) as
the vehicle for this check. That infra was confirmed to live entirely in `canopy-indexer`
(`internal/indexer/characterization_golden_test.go`, `tests/golden/staging/`), not in
canopy-core — this plan is scoped to canopy-core only, so it substitutes an in-repo
differential test instead, built the same way `fsm/indexer_test.go`'s existing tests are.
A cross-repo golden-reference check (running canopy-indexer's harness against a node running
this change) is a reasonable follow-up once this ships, not part of this plan.

**Files:**
- Modify: `fsm/indexer_test.go`

- [ ] **Step 1: Write the test**

```go
// fsm/indexer_test.go
func TestAccountDelta_MatchesOldFullScanAndDiff(t *testing.T) {
	// Build a short chain (3-4 blocks) with a mix of: a brand-new account (first
	// send to a fresh address), a balance update to an existing account, a nonce-only
	// update, and an account zeroed out to trigger SetAccount's delete path — reuse
	// whatever block-building fixture TestDeltaIndexerBlobs_ChangedAddedRemoved (line
	// 11 of this file) already uses, extended with these specific tx types.
	sm, targetHeight := buildMultiBlockFixtureWithAccountChanges(t)

	// OLD PATH: full scan + DeltaIndexerBlobs
	oldCurrent, err := sm.IndexerBlob(context.Background(), targetHeight, false)
	require.NoError(t, err)
	oldPrevious, err := sm.IndexerBlob(context.Background(), targetHeight-1, false)
	require.NoError(t, err)
	oldDelta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: oldCurrent, Previous: oldPrevious})
	require.NoError(t, err)

	// NEW PATH: replay via ApplyBlock(skipRoot=true) + AccountChangeCollector
	block, err := sm.store.(lib.StoreI).GetBlockByHeight(targetHeight)
	require.NoError(t, err)
	replaySM, err := sm.TimeMachine(targetHeight - 1)
	require.NoError(t, err)
	collector := NewAccountChangeCollector(replaySM.Get)
	_, _, err = replaySM.ApplyBlock(context.Background(), block, false, collector, true)
	require.NoError(t, err)
	added, changed, removed := collector.Results()

	newCurrentAccounts := accountBytesForTest(added, changed, true)
	newPreviousAccounts := accountBytesForTest(changed, removed, false)

	require.ElementsMatch(t, oldDelta.Current.Accounts, newCurrentAccounts)
	require.ElementsMatch(t, oldDelta.Previous.Accounts, newPreviousAccounts)
}

// accountBytesForTest mirrors cmd/rpc/query.go's accountBytes helper (kept as a
// separate small copy here rather than an import, since fsm can't import cmd/rpc).
func accountBytesForTest(a, b []*AccountChangeEntry, useFinal bool) [][]byte {
	out := make([][]byte, 0, len(a)+len(b))
	for _, e := range append(append([]*AccountChangeEntry{}, a...), b...) {
		if useFinal {
			if e.FinalValue != nil {
				out = append(out, e.FinalValue)
			}
		} else {
			if e.PrevValue != nil {
				out = append(out, e.PrevValue)
			}
		}
	}
	return out
}
```

`buildMultiBlockFixtureWithAccountChanges` is a placeholder name — before writing this step, read `TestDeltaIndexerBlobs_ChangedAddedRemoved` (fsm/indexer_test.go:11) in full to see exactly how it constructs a `StateMachine` with committed blocks, and extend that same pattern rather than inventing a new one.

- [ ] **Step 2: Run test**

Run: `go test ./fsm/... -run TestAccountDelta_MatchesOldFullScanAndDiff -v`
Expected: PASS. If it fails, the mismatch is a real bug in either `AccountChangeCollector`'s classification or the `accountBytes` override wiring — do not adjust the test to match incorrect output; find and fix the actual discrepancy.

- [ ] **Step 3: Commit**

```bash
git add fsm/indexer_test.go
git commit -m "test(fsm): differential test confirming account delta fast path matches full-scan-and-diff"
```

---

## Task 11: Reward/slash and prefix-leak regression tests

**Files:**
- Modify: `fsm/indexer_test.go`

- [ ] **Step 1: Write the tests**

```go
// fsm/indexer_test.go
func TestAccountDelta_CapturesRewardSlashAccountsWithoutForceInclude(t *testing.T) {
	// Build a block whose reward/slash processing (BeginBlock/EndBlock) touches an
	// account that no direct transaction in the block touches — reuse the fixture
	// pattern from TestDeltaIndexerBlobs_ForceIncludeRewardSlashAccounts (line 70),
	// which already constructs exactly this scenario for the old force-include path.
	sm, targetHeight, rewardedAddress := buildRewardSlashFixture(t)
	block, err := sm.store.(lib.StoreI).GetBlockByHeight(targetHeight)
	require.NoError(t, err)
	replaySM, err := sm.TimeMachine(targetHeight - 1)
	require.NoError(t, err)
	collector := NewAccountChangeCollector(replaySM.Get)
	_, _, err = replaySM.ApplyBlock(context.Background(), block, false, collector, true)
	require.NoError(t, err)
	added, changed, removed := collector.Results()
	var found bool
	for _, e := range append(append(added, changed...), removed...) {
		if bytes.Equal(e.Address, rewardedAddress) {
			found = true
		}
	}
	require.True(t, found, "reward/slash-touched account must be captured without rewardSlashAccountKeys force-include")
}

func TestAccountDelta_PoolAndValidatorWritesDoNotLeakIntoAccountCollector(t *testing.T) {
	sm, targetHeight := buildFixtureWithPoolAndValidatorWrites(t) // any block that updates a pool and a validator alongside an account
	block, err := sm.store.(lib.StoreI).GetBlockByHeight(targetHeight)
	require.NoError(t, err)
	replaySM, err := sm.TimeMachine(targetHeight - 1)
	require.NoError(t, err)
	collector := NewAccountChangeCollector(replaySM.Get)
	_, _, err = replaySM.ApplyBlock(context.Background(), block, false, collector, true)
	require.NoError(t, err)
	added, changed, removed := collector.Results()
	for _, e := range append(append(added, changed...), removed...) {
		require.Len(t, e.Address, 20, "only account addresses should appear (adjust length to the actual address byte size used elsewhere in this file)")
	}
}
```

Both fixture helpers are placeholders — build them by adapting the closest existing fixture in this file (`TestDeltaIndexerBlobs_ForceIncludeRewardSlashAccounts` for the first, any pool/validator-touching test setup elsewhere in the `fsm` package for the second).

- [ ] **Step 2: Run tests**

Run: `go test ./fsm/... -run TestAccountDelta_ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add fsm/indexer_test.go
git commit -m "test(fsm): reward/slash capture and prefix-leak regression tests"
```

---

## Task 12: Full suite + metrics sanity check

- [ ] **Step 1: Run the entire repo test suite**

Run: `go build ./... && go test ./...`
Expected: PASS, no regressions.

- [ ] **Step 2: Manual sanity check of the metrics claim**

Start a local node against a small test chain (or reuse whatever local-cluster deploy path is already documented for this repo — check `HANDOFF.md`/component notes before improvising one). Issue a few `IndexerBlobs` RPC requests for both a live tip height and a historical height, and confirm via the `canopy_indexer_blob_step_time` metric (`ObserveIndexerBlobStep`) that `accounts_iterate` no longer appears for those requests (since `skipAccounts=true` is now always passed), while `account_delta_get` (the new step added in Task 9) does appear and completes quickly.

- [ ] **Step 3: Commit any leftover fixups**

```bash
git add -A
git commit -m "chore: final fixups after full suite run"
```

---

## Explicitly out of scope (per spec)

- Pools and validators — untouched, still full-scan-and-diff via `DeltaIndexerBlobs`.
- Durable/persisted change-sets, pruning/compaction interaction, `Rollback()`/reorg handling — the tip cache is pure in-memory, no durability.
- Throttling or dedup-caching for concurrent lazy replays under heavy backfill load.
- `rewardSlashAccountKeys`/`validatorForceKeys` removal — Task 11 proves the force-include is now redundant for accounts, but the dead code itself is left in place (it's harmless no-op work on an empty account-side diff branch) rather than deleted in this plan, to keep the diff focused on the new mechanism. Removing it is a small, safe follow-up once this ships and is observed working in production.
