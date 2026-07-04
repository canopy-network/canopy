# Oracle Block Provider - Known Issues

## Sync State Flaws

Of the four flaws originally documented here, **two are now resolved in code** (Flaw 1
and Flaw 4), one was always benign (Flaw 2), and one minor observability gap remains
(Flaw 3, low impact). The sync-state handling in `eth/block_provider.go` is now
correct and free of data races.

### Flaw 1: `synced` Never Resets on Reconnect — RESOLVED

`monitorHeaders` now resets the sync flag to `false` on every (re)subscription, so a
reconnect after a disconnect correctly re-enters the syncing state until `nextHeight`
catches back up to the chain tip.

```go
// log successful subscription
p.logger.Info("[ETH-WS] subscribed to new headers")
// reset sync state for reconnection scenarios
p.setSynced(false)
p.metrics.SetEthSyncStatus(1) // syncing
```

**Fix:** `setSynced(false)` is called immediately after `SubscribeNewHead` succeeds,
before the header-processing loop begins. The reconnect scenario (synced, disconnect,
reconnect behind the tip) now shows `synced=false` until catch-up completes.

**Location:** `eth/block_provider.go:284-287` (in `monitorHeaders()`), driven by the
`run()` reconnect loop at `eth/block_provider.go:176-221`

---

### Flaw 2: Sync Check Only Happens on Exact Match — BENIGN (unchanged)

```go
// not synced to top
if !p.IsSynced() {
    // check for source chain sync
    if p.nextHeight.Cmp(header.Number) == 0 {
        // we've caught up to the latest block, mark as synced
        p.setSynced(true)
```

This only triggers when `nextHeight` **exactly equals** the header number. But after
`processBlocks`, `nextHeight` is set to `end + 1`.

So if header 100 arrives and we process up to 100, `nextHeight` becomes 101. The sync
check compares 101 == 100, which is false. We only become synced when header 101
arrives and `nextHeight` is still 101.

This is **correct behavior** but could be confusing - there's a 1-block delay in the
sync flag.

**Location:** `eth/block_provider.go:324-332`

---

### Flaw 3: No "Catching Up" State — STILL OPEN (low impact)

There's still no distinction between:
- Never synced (initial catch-up)
- Was synced, now behind (reconnect catch-up)

The struct carries a single boolean `synced bool` field, so both cases collapse to
`synced=false`. Note that with Flaw 1 resolved, the reconnect-catch-up case now at
least reports `synced=false` correctly (previously it stayed `true`); the remaining
gap is purely observability — a caller cannot tell *which kind* of catch-up is in
progress.

```go
// EthBlockProvider provides ethereum blocks through a channel
type EthBlockProvider struct {
    ...
    synced          bool                       // flag indicating synced to top
    heightMu        *sync.Mutex                // mutex around next height
    ...
}
```

**Location:** `eth/block_provider.go:57-70` - `EthBlockProvider` struct (`synced` field
at line 67)

---

### Flaw 4: Race Condition Potential — RESOLVED

`synced` is now read and written exclusively under `heightMu` on both sides.
`monitorHeaders` no longer touches `p.synced` directly; it reads via `IsSynced()` and
writes via `setSynced()`, both of which take the mutex:

```go
// IsSynced returns whether the block provider has synced to the top of the chain
func (p *EthBlockProvider) IsSynced() bool {
    p.heightMu.Lock()
    defer p.heightMu.Unlock()
    return p.synced
}

// setSynced safely updates the synced flag under heightMu
func (p *EthBlockProvider) setSynced(v bool) {
    p.heightMu.Lock()
    p.synced = v
    p.heightMu.Unlock()
}
```

In `monitorHeaders` the read is `if !p.IsSynced()` and the write is `p.setSynced(true)`
(and the reconnect reset `p.setSynced(false)`), so all accesses are serialized by the
mutex and the data race is eliminated.

**Fix:** introduced `setSynced()` helper and routed all `synced` reads/writes through
`IsSynced()`/`setSynced()`.

**Location:** `eth/block_provider.go:141-155` (`IsSynced` / `setSynced`) and
`eth/block_provider.go:325-329` (guarded read/write in `monitorHeaders`)
