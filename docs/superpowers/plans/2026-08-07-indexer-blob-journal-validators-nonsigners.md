# Indexer Blob Journal: Validators & Non-Signers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing account state-change journal (`fsm.IndexerBlobsFromStateChanges`) so validator and non-signer indexer-blob deltas are also sparse, without the correctness bugs and full-scan regressions found in PR 502 (`perf/indexer-journal-entities`).

**Architecture:** Restructure the journal from one row per touched state key to one row per `(version, entity-type)` bucket, cutting per-key storage overhead and turning prefix-filtered reads into a single point read. Fix non-signers by carrying the address (recoverable from the journal key) into the payload instead of relying on the value, and add a `non_signers_delta` flag so empty deltas are unambiguous. Fix validators by force-including reward/slash/lifecycle-event validators directly by key (no index needed — the event already names the key) and moving total-count maintenance out of the per-block commit path into the blob-build path, where it can reuse data already being fetched and fall back to a full scan only when no baseline exists.

**Tech Stack:** Go, Pebble-backed KV store (`store/` package), protobuf (`lib/.proto/*.proto`, regenerated via `lib/.proto/_generate.sh`), testify (`require`).

## Global Constraints

- `Store.Commit()` is the consensus-critical, every-block, every-node path. Do not add reads, writes, or computation to it beyond the minimum journal-bucketing change in Task 2. Totals, force-inclusion, and address handling all belong in the blob-build path (`fsm/indexer.go`), not `store/store.go`.
- Pre-journal heights must keep working exactly as today: `available=false` from `StateChangeKeys` falls back to the legacy full-snapshot comparison. Every new read path needs the same fallback shape.
- `StateChangeKeys(version, prefix)`'s external contract (return value = full raw state keys, `available bool`, `err`) must not change — callers (`fsm.IndexerBlobsFromStateChanges`) must not need to change how they call it for accounts.
- No new external dependencies. Reuse `lib.DecodeLengthPrefixed`, `lib.JoinLenPrefix`, `lib.Append`, `lib.MemHash`, `fsm.AddressFromKey`, and `lib.Marshal`/`lib.Unmarshal` — do not hand-roll parallel utilities for things these already do.
- Proto changes go in `lib/.proto/*.proto` source files, regenerated with `cd lib/.proto && ./_generate.sh` (requires `protoc` and `protoc-go-inject-tag` on PATH) — never hand-edit `*.pb.go` files directly.

---

## File Structure

- `lib/.proto/store.proto` — add `ValidatorTotals` message (storage-only, not part of the public `IndexerBlob` wire format)
- `lib/.proto/indexer.proto` — add `non_signers_delta` field to `IndexerBlob`
- `lib/store.go` — add `GetValidatorTotals`/`SetValidatorTotals` to `RIndexerI`/`WIndexerI`
- `store/indexer.go` — replace per-key journal write/read (`indexStateChangeKeys`, `StateChangeKeys`'s iterator) with per-type bucketing; add `GetValidatorTotals`/`SetValidatorTotals`
- `store/store.go` — `recordStateChangeKeys` sources from the sorted btree and buckets by type instead of cloning every op key individually
- `store/store_test.go` — extend `TestStateChangeKeys` to cover multiple entity types sharing one version
- `store/indexer_test.go` — new tests for `GetValidatorTotals`/`SetValidatorTotals`
- `fsm/indexer.go` — `IndexerBlobsFromStateChanges` collects validator + non-signer journal keys; `indexerBlob` fetches validators/non-signers selectively, force-includes event-named validators, populates non-signer addresses, computes/caches totals
- `fsm/indexer_test.go` — tests for the above
- `cmd/rpc/query_test.go` — end-to-end test covering edit-stake + reward + finish-unstaking in one block
- `canopy-indexer` repo, `internal/indexer/convert_byzantine.go` and `convert_blob.go` — honor `non_signers_delta`

---

### Task 1: `ValidatorTotals` storage message and store-level persistence

**Files:**
- Modify: `lib/.proto/store.proto`
- Modify: `lib/store.go:61-62` (interface additions)
- Modify: `store/indexer.go`
- Test: `store/indexer_test.go` (new file)

**Interfaces:**
- Produces: `lib.ValidatorTotals` struct (fields: `ValidatorsActive, ValidatorsPaused, ValidatorsUnstaking, DelegatesActive, DelegatesPaused, DelegatesUnstaking uint32`)
- Produces: `(t *Indexer) GetValidatorTotals(version uint64) (totals *lib.ValidatorTotals, available bool, err lib.ErrorI)`
- Produces: `(t *Indexer) SetValidatorTotals(version uint64, totals *lib.ValidatorTotals) lib.ErrorI`

- [ ] **Step 1: Add the `ValidatorTotals` message to `store.proto`**

Append to `lib/.proto/store.proto`:

```protobuf
// ValidatorTotals is a storage-only record of validator/delegate status counts at a
// specific version. It is not part of the IndexerBlob wire format - it's the journal's
// cached baseline so indexerBlob() doesn't have to re-scan every validator on every request.
message ValidatorTotals {
  uint32 validators_active = 1;
  uint32 validators_paused = 2;
  uint32 validators_unstaking = 3;
  uint32 delegates_active = 4;
  uint32 delegates_paused = 5;
  uint32 delegates_unstaking = 6;
}
```

- [ ] **Step 2: Regenerate protobuf code**

Run: `cd lib/.proto && ./_generate.sh`
Expected: `lib/store.pb.go` now contains a `ValidatorTotals` struct with the six `uint32` fields and generated getters. Run `git diff --stat lib/store.pb.go` to confirm only the expected file changed.

- [ ] **Step 3: Add a new prefix and write the read/write methods in `store/indexer.go`**

Add to the prefix block near the top of `store/indexer.go` (after `stateChangePrefix`):

```go
validatorTotalsPrefix = []byte{15} // validator/delegate status totals at a committed version
```

Add near `StateChangeKeys`:

```go
// GetValidatorTotals returns the persisted validator/delegate status totals at version,
// or available=false if nothing has been persisted for that version yet.
func (t *Indexer) GetValidatorTotals(version uint64) (totals *lib.ValidatorTotals, available bool, err lib.ErrorI) {
	bz, err := t.db.Get(t.key(validatorTotalsPrefix, t.encodeBigEndian(version), nil))
	if err != nil || len(bz) == 0 {
		return nil, false, err
	}
	totals = new(lib.ValidatorTotals)
	if err = lib.Unmarshal(bz, totals); err != nil {
		return nil, false, err
	}
	return totals, true, nil
}

// SetValidatorTotals persists validator/delegate status totals for version.
func (t *Indexer) SetValidatorTotals(version uint64, totals *lib.ValidatorTotals) lib.ErrorI {
	bz, err := lib.Marshal(totals)
	if err != nil {
		return err
	}
	return t.db.Set(t.key(validatorTotalsPrefix, t.encodeBigEndian(version), nil), bz)
}
```

- [ ] **Step 4: Add both methods to the `RIndexerI`/`WIndexerI` interfaces in `lib/store.go`**

In `RIndexerI` (next to the existing `StateChangeKeys` line, `lib/store.go:63`):

```go
GetValidatorTotals(version uint64) (totals *ValidatorTotals, available bool, err ErrorI) // get persisted validator/delegate totals at a version
```

In `WIndexerI`:

```go
SetValidatorTotals(version uint64, totals *ValidatorTotals) ErrorI // persist validator/delegate totals at a version
```

- [ ] **Step 5: Write the failing test**

Create `store/indexer_test.go`:

```go
package store

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestValidatorTotals(t *testing.T) {
	st, _, cleanup := testStore(t)
	defer cleanup()

	_, available, err := st.GetValidatorTotals(1)
	require.NoError(t, err)
	require.False(t, available)

	want := &lib.ValidatorTotals{
		ValidatorsActive:    3,
		ValidatorsPaused:    1,
		ValidatorsUnstaking: 0,
		DelegatesActive:     2,
		DelegatesPaused:     0,
		DelegatesUnstaking:  1,
	}
	require.NoError(t, st.SetValidatorTotals(1, want))

	got, available, err := st.GetValidatorTotals(1)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, want.ValidatorsActive, got.ValidatorsActive)
	require.Equal(t, want.ValidatorsPaused, got.ValidatorsPaused)
	require.Equal(t, want.ValidatorsUnstaking, got.ValidatorsUnstaking)
	require.Equal(t, want.DelegatesActive, got.DelegatesActive)
	require.Equal(t, want.DelegatesPaused, got.DelegatesPaused)
	require.Equal(t, want.DelegatesUnstaking, got.DelegatesUnstaking)

	_, available, err = st.GetValidatorTotals(2)
	require.NoError(t, err)
	require.False(t, available)
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./store/... -run TestValidatorTotals -v`
Expected: FAIL — `st.GetValidatorTotals` undefined (interface/method not yet wired if any step above was skipped; otherwise this should already PASS if steps 1-4 were done first — if so, skip to step 7)

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./store/... -run TestValidatorTotals -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add lib/.proto/store.proto lib/store.pb.go lib/store.go store/indexer.go store/indexer_test.go
git commit -m "add persisted ValidatorTotals record to the indexer partition"
```

---

### Task 2: Restructure the state-change journal to per-type buckets

**Files:**
- Modify: `store/indexer.go:32,42-98`
- Modify: `store/store.go:268-272,308-320`
- Test: `store/store_test.go:30-71`

**Interfaces:**
- Consumes: nothing new — reads `s.ss.txn.sorted` (`*btree.BTreeG[*CacheItem]`, existing field, populated because `s.ss` is constructed with `sort: true`)
- Produces: `StateChangeKeys(version uint64, prefix []byte) (keys [][]byte, available bool, err lib.ErrorI)` — **same signature and same return contract as today** (full raw state keys). Internal implementation changes; no caller changes required.

- [ ] **Step 1: Write the failing test — multiple entity types touched in one version**

Extend `store/store_test.go`'s `TestStateChangeKeys` (replace the existing function body with this, which adds a validator-prefix key alongside the existing account keys and asserts both prefixes are independently readable):

```go
func TestStateChangeKeys(t *testing.T) {
	st, _, cleanup := testStore(t)
	defer cleanup()
	require.False(t, st.config.StoreConfig.StateChangeJournalEnabled)
	st.config.StoreConfig.StateChangeJournalEnabled = true

	accountPrefix := lib.JoinLenPrefix([]byte{1})
	validatorPrefix := lib.JoinLenPrefix([]byte{3})
	accountA := lib.JoinLenPrefix([]byte{1}, []byte("account-a"))
	accountB := lib.JoinLenPrefix([]byte{1}, []byte("account-b"))
	validatorX := lib.JoinLenPrefix([]byte{3}, []byte("validator-x"))
	otherKey := lib.JoinLenPrefix([]byte{2}, []byte("other"))

	_, available, err := st.StateChangeKeys(1, accountPrefix)
	require.NoError(t, err)
	require.False(t, available)

	require.NoError(t, st.Set(accountA, []byte("a1")))
	require.NoError(t, st.Set(validatorX, []byte("v1")))
	require.NoError(t, st.Set(otherKey, []byte("other")))
	_, err = st.Commit()
	require.NoError(t, err)

	keys, available, err := st.StateChangeKeys(1, accountPrefix)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, [][]byte{accountA}, keys)

	valKeys, available, err := st.StateChangeKeys(1, validatorPrefix)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, [][]byte{validatorX}, valKeys)

	require.NoError(t, st.Delete(accountA))
	require.NoError(t, st.Set(accountB, []byte("b1")))
	_, err = st.Commit()
	require.NoError(t, err)

	keys, available, err = st.StateChangeKeys(2, accountPrefix)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, [][]byte{accountA, accountB}, keys)

	// version 2 touched no validators - the bucket row simply doesn't exist
	valKeys, available, err = st.StateChangeKeys(2, validatorPrefix)
	require.NoError(t, err)
	require.True(t, available)
	require.Empty(t, valKeys)

	_, err = st.Commit()
	require.NoError(t, err)
	keys, available, err = st.StateChangeKeys(3, accountPrefix)
	require.NoError(t, err)
	require.True(t, available)
	require.Empty(t, keys)
}
```

- [ ] **Step 2: Run test to verify it fails or passes on old behavior**

Run: `go test ./store/... -run TestStateChangeKeys -v`
Expected: PASS with the current per-key implementation too (this test doesn't yet exercise anything the old code can't do — it's here to pin behavior before the rewrite). Confirm it passes now, so a later regression is unambiguous.

- [ ] **Step 3: Replace the write path in `store/indexer.go`**

Replace `indexStateChangeKeys` (`store/indexer.go:83-93`) with:

```go
// indexStateChanges records the commit marker even when ops is empty, then writes one
// row per entity type actually touched this version - keyed by [stateChangePrefix][version][typeHeader],
// valued with the concatenated, already length-prefixed remainders of every touched key of
// that type. ops must already be in ascending key order (see Store.recordStateChangeKeys) -
// DeltaIndexerBlobs' merge-walk in fsm/indexer.go depends on that order downstream.
func (t *Indexer) indexStateChanges(version uint64, ops []valueOp) lib.ErrorI {
	versionPrefix := t.stateChangeVersionPrefix(version)
	if err := t.db.Set(versionPrefix, stateChangeMarker); err != nil {
		return err
	}
	var currentHeader []byte
	var value []byte
	flush := func() lib.ErrorI {
		if currentHeader == nil {
			return nil
		}
		return t.db.Set(lib.Append(versionPrefix, currentHeader), value)
	}
	for _, op := range ops {
		if len(op.key) < 2 {
			continue // malformed/non-entity key, skip
		}
		header := op.key[:2] // [length byte][type-prefix byte], e.g. AccountPrefix()/ValidatorPrefix()
		if currentHeader == nil || !bytes.Equal(header, currentHeader) {
			if err := flush(); err != nil {
				return err
			}
			currentHeader, value = header, nil
		}
		value = append(value, op.key[2:]...) // remainder is already self-length-prefixed
	}
	return flush()
}
```

- [ ] **Step 4: Replace the read path in `store/indexer.go`**

Replace the body of `StateChangeKeys` (`store/indexer.go:53-78`) with:

```go
func (t *Indexer) StateChangeKeys(version uint64, prefix []byte) (keys [][]byte, available bool, err lib.ErrorI) {
	versionPrefix := t.stateChangeVersionPrefix(version)
	marker, err := t.db.Get(versionPrefix)
	if err != nil || len(marker) == 0 {
		return nil, false, err
	}
	// prefix here is e.g. AccountPrefix()/ValidatorPrefix()/NonSignerPrefix() - already the
	// exact 2-byte [length][type] header used as the bucket row's key suffix.
	blob, err := t.db.Get(lib.Append(versionPrefix, prefix))
	if err != nil {
		return nil, false, err
	}
	if len(blob) == 0 {
		return nil, true, nil // available, nothing of this type touched
	}
	for _, remainder := range lib.DecodeLengthPrefixed(blob) {
		keys = append(keys, lib.Append(prefix, lib.JoinLenPrefix(remainder)))
	}
	return keys, true, nil
}
```

- [ ] **Step 5: Update `Store.recordStateChangeKeys` to source sorted ops and call the new writer**

Replace `store/store.go:308-320`:

```go
// recordStateChangeKeys snapshots the pending state transaction before Flush clears it,
// in ascending key order (required for indexStateChanges' per-type bucketing and for the
// downstream merge-walk in DeltaIndexerBlobs). Values are already available from the
// versioned state store, so the journal only needs keys.
func (s *Store) recordStateChangeKeys(version uint64) lib.ErrorI {
	s.ss.txn.l.Lock()
	ops := make([]valueOp, 0, len(s.ss.txn.ops))
	s.ss.txn.sorted.Ascend(func(item *CacheItem) bool {
		ops = append(ops, s.ss.txn.ops[item.HashedKey])
		return true
	})
	s.ss.txn.l.Unlock()
	return s.Indexer.indexStateChanges(version, ops)
}
```

- [ ] **Step 6: Run the test suite**

Run: `go test ./store/... -run TestStateChangeKeys -v`
Expected: PASS

Run: `go test ./store/...`
Expected: all existing store tests still PASS (this confirms the rewrite didn't break `TestIteratorCommitBasic` or anything else that shares `Store.Commit()`)

- [ ] **Step 7: Commit**

```bash
git add store/indexer.go store/store.go store/store_test.go
git commit -m "restructure state-change journal to per-type buckets instead of per-key rows"
```

---

### Task 3: Add `non_signers_delta` to the `IndexerBlob` proto

**Files:**
- Modify: `lib/.proto/indexer.proto`

**Interfaces:**
- Produces: `NonSignersDelta bool` field on the generated `fsm.IndexerBlob` struct (field 24 — next free tag after `block_non_signers = 23`)

- [ ] **Step 1: Add the field**

In `lib/.proto/indexer.proto`, inside `message IndexerBlob`, after `repeated bytes block_non_signers = 23;`:

```protobuf
  // non_signers_delta: true when non_signers is a sparse changed-set (journal path) rather
  // than a full snapshot (legacy path) - mirrors validators_delta.
  bool non_signers_delta = 24;
```

- [ ] **Step 2: Regenerate**

Run: `cd lib/.proto && ./_generate.sh`
Expected: `fsm/indexer.pb.go` gains a `NonSignersDelta bool` field and `GetNonSignersDelta()` getter. `git diff --stat` should show only `fsm/indexer.pb.go` changed.

- [ ] **Step 3: Commit**

```bash
git add lib/.proto/indexer.proto fsm/indexer.pb.go
git commit -m "add non_signers_delta to the IndexerBlob proto"
```

---

### Task 4: Selective validators and non-signers in `indexerBlob()`

**Files:**
- Modify: `fsm/indexer.go:42-95` (`IndexerBlobsFromStateChanges`)
- Modify: `fsm/indexer.go:97-334` (`indexerBlob`)
- Test: `fsm/indexer_test.go`

**Interfaces:**
- Consumes: `st.StateChangeKeys(height, ValidatorPrefix())`, `st.StateChangeKeys(height, NonSignerPrefix())` (Task 2's rewritten implementation, same signature as accounts already use)
- Consumes: `AddressFromKey(k []byte) (crypto.AddressI, lib.ErrorI)` (existing, `fsm/key.go:114`)
- Produces: `validatorForceKeysByAddress(blockBz []byte) ([][]byte, lib.ErrorI)` — new, direct-key-only version of reward/slash/lifecycle force-inclusion (no output index)

- [ ] **Step 1: Write the failing test — non-signer address survives the journal round trip**

Add to `fsm/indexer_test.go`:

```go
func TestIndexerBlobsFromStateChanges_NonSignerAddressRoundTrips(t *testing.T) {
	sm, cleanup := newTestStateMachine(t) // use the same test-SM helper the existing indexer_test.go tests use
	defer cleanup()
	require.NoError(t, sm.SetParams(DefaultParams()))
	address := bytes.Repeat([]byte{0x41}, crypto.AddressSize)

	require.NoError(t, sm.IncrementNonSigners(0, [][]byte{mustPubKeyFor(t, address)}))
	commitAtHeight(t, sm, 1) // existing test helper pattern: commit + advance height

	blobs, available, err := sm.IndexerBlobsFromStateChanges(context.Background(), 2)
	require.NoError(t, err)
	require.True(t, available)
	require.True(t, blobs.Current.NonSignersDelta)
	require.Len(t, blobs.Current.NonSigners, 1)

	ns := new(NonSigner)
	require.NoError(t, lib.Unmarshal(blobs.Current.NonSigners[0], ns))
	require.Equal(t, address, ns.Address)
}
```

(If `newTestStateMachine`/`commitAtHeight`/`mustPubKeyFor` don't already exist in `fsm/indexer_test.go`'s test file, use whatever setup helper the file's other `TestIndexerBlobsFromStateChanges_*` or `TestDeltaIndexerBlobs_*` tests already use — check the top of `fsm/indexer_test.go` before writing this step for the actual helper names in this codebase.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./fsm/... -run TestIndexerBlobsFromStateChanges_NonSignerAddressRoundTrips -v`
Expected: FAIL — `ns.Address` is empty (current code returns raw `IterateAndAppend` values, which never carry the address)

- [ ] **Step 3: Extend journal key collection in `IndexerBlobsFromStateChanges`**

Replace `fsm/indexer.go:63-76`:

```go
	accountKeys, available, err := st.StateChangeKeys(height, AccountPrefix())
	if err != nil || !available {
		return nil, available, err
	}
	validatorKeys, _, err := st.StateChangeKeys(height, ValidatorPrefix())
	if err != nil {
		return nil, true, err
	}
	nonSignerKeys, _, err := st.StateChangeKeys(height, NonSignerPrefix())
	if err != nil {
		return nil, true, err
	}

	b = &IndexerBlobs{}
	b.Current, err = s.indexerBlob(ctx, height, accountKeys, validatorKeys, nonSignerKeys, true, true)
	if err != nil {
		return nil, true, err
	}
```

Update the `Previous` call a few lines below it (currently `s.indexerBlob(ctx, height-1, accountKeys, true, false)`) to match the new parameter list:

```go
	if height > 2 {
		b.Previous, err = s.indexerBlob(ctx, height-1, accountKeys, validatorKeys, nonSignerKeys, true, false)
		if err != nil {
			return nil, true, err
		}
	}
```

Update `IndexerBlob`'s legacy-path call site (`fsm/indexer.go:36`, inside `func (s *StateMachine) IndexerBlob`) to pass `nil, nil` for the two new parameters: `s.indexerBlob(ctx, height, nil, nil, nil, false, false)`.

- [ ] **Step 4: Update `indexerBlob`'s signature and validator/non-signer fetch**

Change the signature (`fsm/indexer.go:97`):

```go
func (s *StateMachine) indexerBlob(ctx context.Context, height uint64, accountKeys, validatorKeys, nonSignerKeys [][]byte, selective, includeBlockEventAccounts bool) (b *IndexerBlob, err lib.ErrorI) {
```

Replace the validator fetch (`fsm/indexer.go:186-191`):

```go
	stepStart = time.Now()
	var validators [][]byte
	if !selective {
		validators, err = sm.IterateAndAppend(ctx, ValidatorPrefix())
	} else {
		forced, forceErr := validatorForceKeysByAddress(blockBz)
		if forceErr != nil {
			return nil, forceErr
		}
		validators, err = sm.valuesForStateKeys(append(validatorKeys, forced...), ValidatorPrefix())
	}
	s.Metrics.ObserveIndexerBlobStep("validators_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
```

Note: `blockBz` is marshaled a few lines below the validator fetch today (`fsm/indexer.go:265-270`, in the `includeBlockEventAccounts` block). Move that marshal earlier, right after `block` is fetched and validated (`fsm/indexer.go:135-140`), so it's available for both the account event-key logic and this new validator force-include logic:

```go
	blockBz, err := lib.Marshal(block)
	if err != nil {
		return nil, err
	}
```//and delete the later, now-duplicate marshal further down in the function.

Replace the non-signer fetch (`fsm/indexer.go:200-206`):

```go
	stepStart = time.Now()
	var nonSigners [][]byte
	if !selective {
		nonSigners, err = sm.IterateAndAppend(ctx, NonSignerPrefix())
	} else {
		for _, key := range nonSignerKeys {
			addr, addrErr := AddressFromKey(key)
			if addrErr != nil {
				return nil, addrErr
			}
			bz, getErr := sm.Get(key)
			if getErr != nil {
				return nil, getErr
			}
			if bz == nil {
				continue
			}
			ns := new(NonSigner)
			if unmarshalErr := lib.Unmarshal(bz, ns); unmarshalErr != nil {
				return nil, unmarshalErr
			}
			ns.Address = addr.Bytes()
			nsBz, marshalErr := lib.Marshal(ns)
			if marshalErr != nil {
				return nil, marshalErr
			}
			nonSigners = append(nonSigners, nsBz)
		}
	}
	s.Metrics.ObserveIndexerBlobStep("non_signers_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
```

Set `NonSignersDelta` in the returned `&IndexerBlob{...}` literal (`fsm/indexer.go:317-334`), alongside the other `Total*`/`ValidatorsDelta`-style fields:

```go
		NonSignersDelta: selective,
```

- [ ] **Step 5: Write `validatorForceKeysByAddress` — the direct-key-only replacement for the old output-index approach**

Add near `validatorForceKeys` (`fsm/indexer.go:670`):

```go
// validatorForceKeysByAddress includes validators tied to lifecycle/reward events by
// forcing the event's own address into the fetch set. This works because reward events
// name the validator's own operator address directly (see committee.go's
// DistributeCommitteeReward, which only emits EventReward for an address that is already
// a valid GetValidator() key) - there is no need to resolve an output address separately;
// the payout address is available on the force-included validator's own Output field.
func validatorForceKeysByAddress(blockBz []byte) ([][]byte, lib.ErrorI) {
	keys := make([][]byte, 0)
	if len(blockBz) == 0 {
		return keys, nil
	}
	block := new(lib.BlockResult)
	if err := lib.Unmarshal(blockBz, block); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	for _, event := range block.Events {
		if event == nil || len(event.Address) == 0 {
			continue
		}
		switch event.EventType {
		case string(lib.EventTypeReward), string(lib.EventTypeSlash),
			string(lib.EventTypeAutoPause), string(lib.EventTypeAutoBeginUnstaking),
			string(lib.EventTypeFinishUnstaking):
			addr := string(event.Address)
			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}
			keys = append(keys, lib.JoinLenPrefix(validatorPrefix, event.Address))
		}
	}
	return keys, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./fsm/... -run TestIndexerBlobsFromStateChanges_NonSignerAddressRoundTrips -v`
Expected: PASS

- [ ] **Step 7: Run the full fsm test suite**

Run: `go test ./fsm/...`
Expected: all PASS — this catches any existing `DeltaIndexerBlobs`/`IndexerBlobsFromStateChanges` tests broken by the signature change

- [ ] **Step 8: Commit**

```bash
git add fsm/indexer.go fsm/indexer_test.go
git commit -m "make validators and non-signers sparse in the journal path"
```

---

### Task 5: Validator totals computed at blob-build time, not commit time

**Files:**
- Modify: `fsm/indexer.go:311-316` (`validatorTotals(validators)` call site)
- Test: `fsm/indexer_test.go`

**Interfaces:**
- Consumes: `st.GetValidatorTotals(height)`, `st.SetValidatorTotals(height, totals)` (Task 1)
- Consumes: `validatorTotals(validators [][]byte) (...)` (existing, `fsm/indexer.go:703`, kept as the fallback)
- Produces: `(s *StateMachine) resolveValidatorTotals(st lib.StoreI, height uint64, current, previous [][]byte) (*lib.ValidatorTotals, lib.ErrorI)`

- [ ] **Step 1: Write the failing test — totals come from the persisted baseline, not a full scan, when available**

Add to `fsm/indexer_test.go`:

```go
func TestResolveValidatorTotals_UsesPersistedBaselineWhenAvailable(t *testing.T) {
	sm, cleanup := newTestStateMachine(t)
	defer cleanup()
	st := sm.Store().(lib.StoreI)

	require.NoError(t, st.SetValidatorTotals(4, &lib.ValidatorTotals{ValidatorsActive: 7}))

	// current/previous entries are irrelevant here since no validator changed - the fallback
	// full scan must NOT run when height-1's baseline is available; if it did, it would see
	// zero validators in this empty test store and totals would come back 0, not 7.
	totals, err := sm.resolveValidatorTotals(st, 5, nil, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(7), totals.ValidatorsActive)
}

func TestResolveValidatorTotals_FallsBackToFullScanWhenNoBaseline(t *testing.T) {
	sm, cleanup := newTestStateMachine(t)
	defer cleanup()
	st := sm.Store().(lib.StoreI)
	require.NoError(t, sm.SetParams(DefaultParams()))

	v := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{0x51}, crypto.AddressSize)})
	require.NoError(t, sm.SetValidator(&Validator{Address: bytes.Repeat([]byte{0x51}, crypto.AddressSize)}))
	commitAtHeight(t, sm, 1)

	totals, err := sm.resolveValidatorTotals(st, 2, [][]byte{v}, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(1), totals.ValidatorsActive)

	cached, available, err := st.GetValidatorTotals(2)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, uint32(1), cached.ValidatorsActive)
}
```

(As in Task 4 Step 1: check `fsm/indexer_test.go`'s existing helpers for the actual names of `newTestStateMachine`/`commitAtHeight`/`SetValidator` before writing this — use what's already there rather than inventing new setup code.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./fsm/... -run TestResolveValidatorTotals -v`
Expected: FAIL — `sm.resolveValidatorTotals` undefined

- [ ] **Step 3: Implement `resolveValidatorTotals`**

Add near `validatorTotals` (`fsm/indexer.go:703`):

```go
// resolveValidatorTotals returns the validator/delegate status totals at height, computed
// incrementally from the persisted baseline at height-1 plus the status transitions visible
// in the current/previous validator entries already fetched for this blob. Falls back to a
// full scan (same cost as the pre-journal path) only when no baseline exists yet - this
// happens at most once per node, the first time a journal-path blob is requested after this
// feature goes live; every height after that reads/writes the incremental baseline.
func (s *StateMachine) resolveValidatorTotals(st lib.StoreI, height uint64, current, previous [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	baseline, available, err := st.GetValidatorTotals(height - 1)
	if err != nil {
		return nil, err
	}
	var totals *lib.ValidatorTotals
	if !available {
		full, fullErr := s.fullValidatorSnapshotForTotals()
		if fullErr != nil {
			return nil, fullErr
		}
		totals, err = totalsFromFullScan(full)
		if err != nil {
			return nil, err
		}
	} else {
		totals, err = applyValidatorTransitions(baseline, current, previous)
		if err != nil {
			return nil, err
		}
	}
	if err := st.SetValidatorTotals(height, totals); err != nil {
		return nil, err
	}
	return totals, nil
}

// fullValidatorSnapshotForTotals does the one-time full scan used only when no baseline
// exists yet for height-1.
func (s *StateMachine) fullValidatorSnapshotForTotals() ([][]byte, lib.ErrorI) {
	return s.IterateAndAppend(context.Background(), ValidatorPrefix())
}

func totalsFromFullScan(validators [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	active, paused, unstaking, delActive, delPaused, delUnstaking, err := validatorTotals(validators)
	if err != nil {
		return nil, err
	}
	return &lib.ValidatorTotals{
		ValidatorsActive: active, ValidatorsPaused: paused, ValidatorsUnstaking: unstaking,
		DelegatesActive: delActive, DelegatesPaused: delPaused, DelegatesUnstaking: delUnstaking,
	}, nil
}

// applyValidatorTransitions diffs the current/previous validator entries already fetched
// for this blob against the persisted baseline. previous entries not present in current
// represent deletions (e.g. EventFinishUnstaking) - decrement only, no increment.
func applyValidatorTransitions(baseline *lib.ValidatorTotals, current, previous [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	totals := &lib.ValidatorTotals{
		ValidatorsActive: baseline.ValidatorsActive, ValidatorsPaused: baseline.ValidatorsPaused,
		ValidatorsUnstaking: baseline.ValidatorsUnstaking, DelegatesActive: baseline.DelegatesActive,
		DelegatesPaused: baseline.DelegatesPaused, DelegatesUnstaking: baseline.DelegatesUnstaking,
	}
	prevByAddr, err := validatorStatusByAddress(previous)
	if err != nil {
		return nil, err
	}
	currByAddr, err := validatorStatusByAddress(current)
	if err != nil {
		return nil, err
	}
	for addr, curr := range currByAddr {
		old, hadOld := prevByAddr[addr]
		if hadOld {
			decrementBucket(totals, old)
		}
		incrementBucket(totals, curr)
	}
	for addr, old := range prevByAddr {
		if _, stillPresent := currByAddr[addr]; !stillPresent {
			decrementBucket(totals, old) // deleted (e.g. finished unstaking) - decrement only
		}
	}
	return totals, nil
}

type validatorStatus struct {
	unstaking, paused, delegate bool
}

func validatorStatusByAddress(entries [][]byte) (map[string]validatorStatus, lib.ErrorI) {
	out := make(map[string]validatorStatus, len(entries))
	for _, entry := range entries {
		v := new(Validator)
		if err := lib.Unmarshal(entry, v); err != nil {
			return nil, lib.ErrUnmarshal(err)
		}
		out[string(v.Address)] = validatorStatus{
			unstaking: v.UnstakingHeight > 0,
			paused:    v.UnstakingHeight == 0 && v.MaxPausedHeight > 0,
			delegate:  v.Delegate,
		}
	}
	return out, nil
}

func incrementBucket(t *lib.ValidatorTotals, s validatorStatus) {
	switch {
	case s.unstaking:
		t.ValidatorsUnstaking++
		if s.delegate {
			t.DelegatesUnstaking++
		}
	case s.paused:
		t.ValidatorsPaused++
		if s.delegate {
			t.DelegatesPaused++
		}
	default:
		t.ValidatorsActive++
		if s.delegate {
			t.DelegatesActive++
		}
	}
}

func decrementBucket(t *lib.ValidatorTotals, s validatorStatus) {
	switch {
	case s.unstaking:
		t.ValidatorsUnstaking--
		if s.delegate {
			t.DelegatesUnstaking--
		}
	case s.paused:
		t.ValidatorsPaused--
		if s.delegate {
			t.DelegatesPaused--
		}
	default:
		t.ValidatorsActive--
		if s.delegate {
			t.DelegatesActive--
		}
	}
}
```

- [ ] **Step 4: Wire it into `indexerBlob`, replacing the direct `validatorTotals(validators)` call**

Replace `fsm/indexer.go:311-316`:

```go
	var totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking uint32
	var totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking uint32
	if !selective {
		totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking,
			totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking, err = validatorTotals(validators)
		if err != nil {
			return nil, err
		}
	} else {
		totals, totalsErr := s.resolveValidatorTotals(st, height, validators, nil)
		if totalsErr != nil {
			return nil, totalsErr
		}
		totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking =
			totals.ValidatorsActive, totals.ValidatorsPaused, totals.ValidatorsUnstaking
		totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking =
			totals.DelegatesActive, totals.DelegatesPaused, totals.DelegatesUnstaking
	}
```

Note: this call site is inside `indexerBlob`, which is called once for `Current` and once for `Previous`. The `previous` argument to `resolveValidatorTotals` should be the *other* call's validator set so deletions are detected — pass `nil` here for the initial wiring (matches the test in Step 1, which only exercises the single-height case) and revisit in Task 6's end-to-end test, where the finish-unstaking case needs both sides. If Task 6 finds this insufficient, thread the previous height's validator set through `IndexerBlobsFromStateChanges` into the `Current` call instead of computing totals inside each independent `indexerBlob` invocation — check the Task 6 test's result before deciding which shape is needed.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./fsm/... -run TestResolveValidatorTotals -v`
Expected: PASS

- [ ] **Step 6: Run the full fsm test suite**

Run: `go test ./fsm/...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add fsm/indexer.go fsm/indexer_test.go
git commit -m "compute validator totals in indexerBlob() with a persisted incremental baseline"
```

---

### Task 6: End-to-end integration test — edit-stake + reward + finish-unstaking in one block

**Files:**
- Modify: `cmd/rpc/query_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1-5

- [ ] **Step 1: Write the test**

Add to `cmd/rpc/query_test.go` (replacing PR 502's `TestIndexerBlobsCached_JournalPathUsesValidatorAndNonSignerKeys`, which seeded a fake address into the non-signer value — see Global Constraints and this repo's PR 502 review for why that test masked the bug):

```go
func TestIndexerBlobsCached_JournalPathHandlesValidatorsAndNonSigners(t *testing.T) {
	server := newTestIndexerBlobServerWithHeights(t, 4)
	sm := server.controller.FSM
	db := sm.Store().(lib.StoreI)

	editStakeValidator := crypto.NewAddress(bytes.Repeat([]byte{0x31}, crypto.AddressSize))
	unstakingValidator := crypto.NewAddress(bytes.Repeat([]byte{0x32}, crypto.AddressSize))
	nonSignerAddress := crypto.NewAddress(bytes.Repeat([]byte{0x33}, crypto.AddressSize))
	rewardOutput := bytes.Repeat([]byte{0x34}, crypto.AddressSize)

	require.NoError(t, sm.SetValidator(&fsm.Validator{Address: editStakeValidator.Bytes(), Output: rewardOutput}))
	require.NoError(t, sm.SetValidator(&fsm.Validator{Address: unstakingValidator.Bytes(), UnstakingHeight: 4}))
	require.NoError(t, sm.IncrementNonSigners(0, [][]byte{mustPubKeyFor(t, nonSignerAddress.Bytes())}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{BlockHeader: &lib.BlockHeader{
		Height: 4, Hash: crypto.Hash([]byte("block-4-seed")), Time: uint64(time.Now().UnixMicro()),
	}}))
	_, err := db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 5)

	seeded, _, err := server.IndexerBlobsCached(5)
	require.NoError(t, err)
	require.Len(t, seeded.Current.Validators, 2)
	require.Len(t, seeded.Current.NonSigners, 1)
	require.Equal(t, uint32(1), seeded.Current.TotalValidatorsActive)
	require.Equal(t, uint32(1), seeded.Current.TotalValidatorsUnstaking)

	// height 5: editStakeValidator gets a reward (output unchanged), unstakingValidator
	// finishes unstaking (deleted)
	require.NoError(t, sm.SetValidator(&fsm.Validator{Address: editStakeValidator.Bytes(), Output: rewardOutput, Committees: []uint64{1}}))
	require.NoError(t, sm.DeleteValidator(&fsm.Validator{Address: unstakingValidator.Bytes(), UnstakingHeight: 4}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{BlockHeader: &lib.BlockHeader{
		Height: 5, Hash: crypto.Hash([]byte("block-5-reward-and-unstake")), Time: uint64(time.Now().UnixMicro()),
	}, Events: []*lib.Event{{EventType: string(lib.EventTypeReward), Address: editStakeValidator.Bytes()}}}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 6)

	rewarded, _, err := server.IndexerBlobsCached(6)
	require.NoError(t, err)
	require.Len(t, rewarded.Current.Validators, 1) // editStakeValidator only - it changed and was force-included
	require.Len(t, rewarded.Previous.Validators, 2) // both present at height 5
	require.Equal(t, uint32(1), rewarded.Current.TotalValidatorsActive)
	require.Equal(t, uint32(0), rewarded.Current.TotalValidatorsUnstaking)

	var got fsm.Validator
	require.NoError(t, lib.Unmarshal(rewarded.Current.Validators[0], &got))
	require.Equal(t, rewardOutput, got.Output)

	require.True(t, rewarded.Current.NonSignersDelta)
	require.Empty(t, rewarded.Current.NonSigners)
}
```

(`mustPubKeyFor` needs to construct something `crypto.NewPublicKeyFromBytes`/`IncrementNonSigners` accepts as a pubkey whose `.Address()` equals the target address — check how existing tests in this file or `fsm/byzantine_test.go` construct a non-signer pubkey fixture, and reuse that helper rather than writing a new one.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/... -run TestIndexerBlobsCached_JournalPathHandlesValidatorsAndNonSigners -v`
Expected: FAIL initially if Task 5's `previous` threading (Step 4's open question) wasn't resolved — the `TotalValidatorsUnstaking` assertion at height 6 needs the deletion to be visible, which requires `resolveValidatorTotals` to see the previous height's validator set. Fix by threading the previous validator set from `IndexerBlobsFromStateChanges` into the `Current` blob's totals resolution before re-running.

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./cmd/rpc/... -run TestIndexerBlobsCached_JournalPathHandlesValidatorsAndNonSigners -v`
Expected: PASS

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/query_test.go
git commit -m "add end-to-end test for journal-path validators and non-signers"
```

---

### Task 7: `canopy-indexer` consumer honors `non_signers_delta`

**Files:**
- Modify: `canopy-indexer/internal/indexer/convert_byzantine.go:11-52` (`convertNonSignersWithChangeDetection`)
- Modify: `canopy-indexer/internal/indexer/convert_block.go:143-144,277-279` (caller wiring)
- Test: `canopy-indexer/internal/indexer/convert_byzantine_test.go`

**Interfaces:**
- Consumes: `blob.NonSignersDelta bool` (Task 3's new field, available via `fsm.IndexerBlob` once `canopy-indexer` picks up the updated `canopy` dependency)

- [ ] **Step 1: Write the failing test**

Add to `canopy-indexer/internal/indexer/convert_byzantine_test.go` (create if it doesn't exist, following the existing table-test style in `indexer_test.go`'s `TestNonSignMap`-style tests referenced earlier in this repo):

```go
func TestConvertNonSignersWithChangeDetection_SparseDeltaSkipsResetDetection(t *testing.T) {
	current := []*fsm.NonSigner{{Address: []byte("addr-a"), Counter: 2}}
	// sparse/delta mode: previous is empty not because everyone reset, but because only
	// "addr-a" was touched this height - Phase 2's reset-detection must not run here.
	results, newCount := convertNonSignersWithChangeDetection(current, nil, 10, time.Now(), true)
	require.Equal(t, uint32(1), newCount)
	require.Len(t, results, 1)
	require.Equal(t, "addr-a", results[0].Address... /* whatever addr encoding this repo uses, e.g. lib.BytesToString */)
}

func TestConvertNonSignersWithChangeDetection_FullSnapshotStillDetectsResets(t *testing.T) {
	previous := []*fsm.NonSigner{{Address: []byte("addr-a"), Counter: 2}}
	// full-snapshot mode (delta=false): addr-a missing from current means it reset.
	results, _ := convertNonSignersWithChangeDetection(nil, previous, 10, time.Now(), false)
	require.Len(t, results, 1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/indexer/... -run TestConvertNonSignersWithChangeDetection -v`
Expected: FAIL — `convertNonSignersWithChangeDetection` doesn't accept a 5th `delta bool` parameter yet

- [ ] **Step 3: Update the function signature and guard Phase 2 on the flag**

Replace the signature and Phase 2 block in `convert_byzantine.go`:

```go
func convertNonSignersWithChangeDetection(
	current, previous []*fsm.NonSigner, height uint64, blockTime time.Time, delta bool,
) (results []*indexermodels.ValidatorNonSigningInfo, newCount uint32) {
	prevMap, currMap := nonSignMap(previous), nonSignMap(current)

	addRow := func(addr string, counter uint64) {
		results = append(results, &indexermodels.ValidatorNonSigningInfo{
			Address: addr, MissedBlocksCount: counter, LastSignedHeight: height, Height: height, HeightTime: blockTime,
		})
	}

	for _, curr := range current {
		addr := lib.BytesToString(curr.Address)
		var changed bool
		prev := prevMap[addr]
		if prev == nil {
			newCount++
		}
		if prev == nil || curr.Counter != prev.Counter {
			changed = true
		}
		if changed {
			addRow(addr, curr.Counter)
		}
	}

	// Phase 2 (existed at H-1, missing at H = reset) is only valid against a full snapshot.
	// In sparse/delta mode, "missing at H" just means "not touched at H" - it says nothing
	// about whether the non-signer still exists.
	if !delta {
		for _, prev := range previous {
			addr := lib.BytesToString(prev.Address)
			if _, exists := currMap[addr]; !exists {
				addRow(addr, prev.Counter)
			}
		}
	}

	return
}
```

- [ ] **Step 4: Update the call site in `convert_block.go`**

At `convert_block.go:277-279`, pass the flag through from the blob:

```go
result.ValidatorNonSigningInfo, result.NewNonSigningInfos = convertNonSignersWithChangeDetection(
	data.NonSignersCurrent, data.NonSignersPrevious, height, blockTime, blob.Current.NonSignersDelta,
)
```

(Confirm `blob`/`data` naming matches what's already in scope at this call site — read the surrounding function in `convert_block.go` before editing, since the exact variable holding the raw `*fsm.IndexerBlob` may differ from `blob`.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/indexer/... -run TestConvertNonSignersWithChangeDetection -v`
Expected: PASS

- [ ] **Step 6: Run the full indexer test suite**

Run: `go test ./internal/indexer/...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/indexer/convert_byzantine.go internal/indexer/convert_block.go internal/indexer/convert_byzantine_test.go
git commit -m "honor non_signers_delta: skip reset-detection on sparse journal deltas"
```

---

## Self-Review Notes

- **Spec coverage:** journal restructure (Task 2), non-signer address+delta fix (Tasks 3-4), validator selective fetch + reward force-inclusion without an index (Task 4), validator totals moved off the commit path with fallback (Task 1, 5), end-to-end proof (Task 6), consumer update (Task 7). All items from the design discussion are covered.
- **Open design decision flagged, not hidden:** Task 5 Step 4 and Task 6 Step 2 call out explicitly that `previous`-side threading for totals (needed to detect deletions like finish-unstaking) may require restructuring how `resolveValidatorTotals` is called from `indexerBlob` vs `IndexerBlobsFromStateChanges` — this is a real open question the code will surface, not a placeholder; Task 6's test is what proves which shape is correct.
- **Type consistency:** `lib.ValidatorTotals` (Task 1) is used with identical field names throughout Tasks 5-6. `NonSignersDelta` (Task 3) matches between the proto, Task 4's blob construction, and Task 6/7's assertions. `validatorForceKeysByAddress` (Task 4) is distinct from the existing `validatorForceKeys` (kept, unused by the new path, safe to leave for now since `DeltaIndexerBlobs` still calls it for the legacy/pool-diffing path).
