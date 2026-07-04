# Oracle Block Provider - Known Issues

## Sync State Flaws

### Flaw 1: `synced` Never Resets on Reconnect

When the websocket disconnects and `monitorHeaders` returns an error, the `run()` loop reconnects. But `synced` stays `true` even though:
- We might have missed blocks during the disconnect
- `nextHeight` could now be behind the chain tip again

**Example:**
1. Provider synced at height 24,000,000
2. Websocket disconnects for 5 minutes (~25 blocks)
3. Reconnects - `nextHeight` is still 24,000,001 but chain is at 24,000,025
4. `synced` is still `true` even though we're 24 blocks behind

**Location:** `eth/block_provider.go` - `run()` loop and `monitorHeaders()`

---

### Flaw 2: Sync Check Only Happens on Exact Match

```go
if p.nextHeight.Cmp(header.Number) == 0 {
    p.synced = true
}
```

This only triggers when `nextHeight` **exactly equals** the header number. But after `processBlocks`, `nextHeight` is set to `end + 1`.

So if header 100 arrives and we process up to 100, `nextHeight` becomes 101. The sync check compares 101 == 100, which is false. We only become synced when header 101 arrives and `nextHeight` is still 101.

This is actually **correct behavior** but could be confusing - there's a 1-block delay in the sync flag.

**Location:** `eth/block_provider.go:282-289`

---

### Flaw 3: No "Catching Up" State

There's no distinction between:
- Never synced (initial catch-up)
- Was synced, now behind (reconnect catch-up)

Both show `synced=false` or `synced=true` (if it was ever synced).

**Location:** `eth/block_provider.go` - `EthBlockProvider` struct

---

### Flaw 4: Race Condition Potential

`synced` is read/written without the mutex in `monitorHeaders`:
```go
if !p.synced {           // read without lock
    if p.nextHeight.Cmp(header.Number) == 0 {
        p.synced = true  // write without lock
```

But `IsSynced()` uses the mutex:
```go
func (p *EthBlockProvider) IsSynced() bool {
    p.heightMu.Lock()
    defer p.heightMu.Unlock()
    return p.synced
}
```

If another goroutine calls `IsSynced()` while `monitorHeaders` is writing, there's a data race.

**Location:** `eth/block_provider.go:140-146` and `eth/block_provider.go:282-289`
