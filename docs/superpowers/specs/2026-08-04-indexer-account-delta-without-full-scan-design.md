# Eliminate the full account scan in IndexerBlob

## Problem

`IndexerBlob(height)` (`fsm/indexer.go:35`) currently produces its account delta by doing a
full `IterateAndAppend(ctx, AccountPrefix())` scan of the *entire* account keyspace (~1.35M
accounts) at both `height` and `height-1`, then `DeltaIndexerBlobs` (`fsm/indexer.go:279+`)
diffs the two full lists down to changed/added/removed entries before the RPC layer ever
serializes a response. This is O(total accounts) work, twice, on every `IndexerBlobs` request,
regardless of how many accounts a block actually touched.

Downstream, canopy-indexer never uses a full account list either — the `accounts` table write
(`insertAccounts`, `pkg/db/postgres/chain/inserts.go:1264+`), the `_latest` skip-cache, and
`block_summary.NumAccounts` all operate purely on the delta set already produced by
`DeltaIndexerBlobs`. So the full scan exists only to be thrown away.

This is orthogonal to (not a replacement for) `task:1375` (replacing the map-based diff in
`DeltaIndexerBlobs` with a sorted merge walk), which stays useful regardless as a correctness
cross-check and for whatever full-scan usage remains.

## Goal

Eliminate the full account-keyspace scan entirely — for both the live (tip) case and
arbitrary historical/backfill requests — by capturing the set of accounts a block actually
touched at write time (live) or via minimal replay (historical), instead of scanning and
diffing the whole keyspace.

Scope: **accounts only**. Pools and validators keep using the existing full-scan-and-diff path
unchanged — their entry counts are small enough that the scan cost isn't a problem, and
`poolEntryKey`'s wire-format ordering already complicates reusing the sorted-merge approach
used for accounts (see `task:1375`).

## Design

### Core idea

`ApplyBlock` (`fsm/state.go:140`) already knows, in real time, every account it writes —
`BeginBlock` → `ApplyTransactions` → `EndBlock`, with every account mutation flowing through
the FSM's generic key-value `Set`/`Delete` methods. Hooking those calls captures the touched
account set as a side effect of execution that's already happening, live, for every block —
no separate scan needed. For requests where no live execution ever happened (historical /
backfill heights), the exact same `ApplyBlock` can be re-run as a throwaway, non-committing
call against a `TimeMachine(height-1)` snapshot to recover the same information — a pattern
already used elsewhere (mempool/ephemeral tx checks call `ApplyBlock` without ever committing:
`controller/tx.go:309`).

### `ApplyBlock` signature change

Add two optional parameters to `ApplyBlock` (Go has no native optional params, so this is
either a small trailing options struct or new trailing args with all existing call sites
updated to pass zero-values, preserving current behavior exactly):

- `collector *AccountChangeCollector` — nil by default. When non-nil, the account-write hook
  (below) records into it.
- `skipRoot bool` — false by default. When true, skips validator-merkle-root computation and
  `store.Root()` (`fsm/state.go:212`, `store/store.go:520`), which today has no skip path and
  needs one added. `store.Root()` is a distinct, separable step — it runs after all account
  writes are already flushed to the txn ops log, so skipping it doesn't touch write logic.
  **`skipRoot=true` must never reach a caller that consumes the header hash** — confirmed the
  only such caller is the consensus `ApplyAndValidateBlock` path
  (`controller/block.go:524`), which always passes `false`.

Existing call sites (consensus commit, mempool checks) pass `collector=nil, skipRoot=false`
and see no behavior change.

### `AccountChangeCollector`

A per-`ApplyBlock`-call accumulator: `map[address]{prevValue *Value, finalValue *Value}`
(`prevValue == nil` means the address didn't exist at `height-1`).

- On first touch of an address within the block, look up its value against the `height-1`
  baseline (the pre-block state) to capture `prevValue`. This is correct by construction:
  it's the first read for that key in this execution, before this block's own writes shadow
  it — no separate "is this mid-block or pre-block" tracking needed.
- On every touch (first or subsequent), update `finalValue`.
- At block end, walk the map once: `prevValue == nil` → added; `finalValue` is a delete →
  removed; otherwise, `prevValue != finalValue` → changed (else drop — touched but unchanged
  is possible if a tx re-writes an identical value, and shouldn't be reported as a delta).

Because `BeginBlock`/`EndBlock` (where reward/slash processing happens) go through the same
hooked `Set`/`Delete` calls as normal tx execution, reward/slash-touched accounts are captured
automatically. **`DeltaIndexerBlobs`'s `rewardSlashAccountKeys` force-include
(`fsm/indexer.go:511`) becomes dead code for accounts** and can be removed once this ships.

### Hook placement

The hook lives in the FSM's generic `Set(key, value)` / `Delete(key)` methods — the ones
`SetAccount`, `SetPool`, `SetValidator`, etc. all funnel through — not in `SetAccount()`
specifically. This catches any account write regardless of which higher-level function issued
it, and reuses the same op-tagging (`opSet`/`opDelete`) pattern already established by
`valueOp`/`collectLssDeleteKeys` (`store/txn.go:30-32,57-62`, `store/store.go:813-826`) at the
adjacent store layer:

```go
if collector != nil && bytes.HasPrefix(key, AccountPrefix()) {
    collector.Record(key, op, value)
}
```

Address is recovered by stripping `AccountPrefix()` from `key`, rather than being passed in as
a struct field.

### Two call paths

1. **Live** — the real, committing `ApplyBlock()` call (consensus path) passes a live
   `collector` when the node is configured to serve `IndexerBlobs` (config-gated per node, not
   global — nodes that don't opt in see zero added cost beyond a nil check per write).
   `skipRoot=false` always, since this call still needs a real header for consensus. After
   commit, the collector's finalized added/changed/removed list is held in a small in-memory
   tip cache (a handful of recent heights) for the RPC handler to serve immediately.
   This capture only runs when the node is live/caught-up — **not during sync/replay**, where
   `ApplyBlock` runs in a tight catch-up loop over blocks nothing is polling yet.

2. **Lazy/replay** — on an `IndexerBlobs` request for height `H`: check the tip cache first;
   on miss (arbitrary historical height, backfill, sync-replayed heights, or an evicted tip
   entry), call `ApplyBlock(ctx, block(H), allowOversize, collector, skipRoot=true)` against a
   `TimeMachine(H-1)`-derived store, then simply never call `Commit()` — mirroring the existing
   no-commit mempool pattern. Requires the raw block (tx list) for `H` to still be available;
   existing evidence (`indexerPrefix` compaction handling in `store.go`) suggests block/tx/QC
   data is retained indefinitely, but this is an assumption worth confirming before relying on
   it broadly.

Either path produces the same shape: a list of `{address, op, prevValue?, finalValue}` for
accounts actually touched by that block — directly consumable wherever
`DeltaIndexerBlobs`'s account output is consumed today.

## Out of scope

- Pools and validators — unchanged, keep full-scan-and-diff.
- Durable/persisted change-sets — this design has no new persisted state and no interaction
  with pruning, compaction, or `Rollback()`/reorg handling. The tip cache is a pure in-memory
  accelerator with no correctness dependency (a reorged-out block's cache entry is simply
  never requested again).
- Throttling/caching for concurrent lazy replays under heavy backfill load — flagged as a
  follow-up if it becomes a real bottleneck, not solved here.

## Testing

- **Unit**: `AccountChangeCollector` classification — new account, balance-only change,
  nonce-only change, zero-out/delete, multiple writes to the same account within one block
  (must classify against the `height-1` baseline, not mid-block state).
- **Prefix-filter test**: pool/validator/param writes in the same block must never leak into
  the account collector's output.
- **Golden-reference regression**: run both the current full-scan-and-diff path and the new
  hook-based path (via existing golden-reference dataset infra, `task:1371`) across a range of
  real historical heights and assert identical added/changed/removed sets before this replaces
  the old path for accounts in production.
- **Reward/slash regression**: confirm accounts touched only via `BeginBlock`/`EndBlock`
  reward/slash logic are captured now that `rewardSlashAccountKeys` force-include is removed.
- **Perf**: before/after comparison via the existing `ObserveIndexerBlobStep`/`accounts_iterate`
  metrics, confirming the scan step disappears from the hot path for both live and replay
  requests.

## Open questions / risks to confirm during implementation

- Confirm block/tx/QC retention is actually indefinite (or at least covers whatever height
  range `IndexerBlobs` backfill callers realistically request) before treating the
  `skipRoot=true` replay path as universally available.
- Confirm no other code path writes account keys outside the FSM's generic `Set`/`Delete`
  (the prefix-filtered hook assumes there isn't one).
