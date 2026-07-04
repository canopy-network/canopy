# Code Review: oracle package (`cmd/rpc/oracle/`)

**Date**: 2026-07-04 03:56:41
**Reviewer**: Claude Code (full-review skill)
**Scope**: `cmd/rpc/oracle/` top level only (oracle.go, state.go, order_store.go, order_validator.go, error.go). Excludes `eth/` and `types/` subpackages.

## Summary

| Skill | Status | MUST | SHOULD | CAN |
|-------|--------|------|--------|-----|
| go-review | ❌ | 3 | 3 | 2 |
| go-testing | ⚠️ | 0 | 2 | 1 |
| architecture | ⚠️ | 0 | 2 | 1 |
| observability | ❌ | 1 | 1 | 1 |

**Overall**: 4 MUST violations, 8 SHOULD violations, 5 CAN suggestions

---

## MUST Violations (Blocking)

### [RACE-1] `o.orderBook` read without holding `orderBookMu`
- **File**: oracle.go:156 (`for o.orderBook == nil`)
- **Issue**: `Start` goroutine reads `o.orderBook` in a spin loop with no lock. `UpdateRootChainInfo` writes `o.orderBook` under `orderBookMu.Lock()` (oracle.go:677). Concurrent unsynchronized read/write = data race.
- **Fix**: Read behind `orderBookMu.RLock()`, or gate readiness with a `chan struct{}` / `atomic.Bool` set once the first order book arrives.

### [RACE-2] `o.state.sourceChainHeight` + submission maps read without state lock
- **File**: oracle.go:757 (`DebugOrder`), oracle.go:910-913 (`updateMetrics`)
- **Issue**: Both reach directly into `OracleState` unexported fields (`sourceChainHeight`, `len(lockOrderSubmissions)`, `len(closeOrderSubmissions)`) from the oracle without taking `state.rwLock`. `ValidateSequence` writes `sourceChainHeight` under the write lock (state.go:145); `shouldSubmit`/`PruneHistory` mutate the maps under lock. Racy read of `sourceChainHeight` and — worse — racy `len()` on a map being written concurrently.
- **Fix**: Add `GetSourceChainHeight()` and submission-count getters on `OracleState` that take `rwLock.RLock()`. Never touch state internals from `Oracle` directly.

### [OBS-1 / LOG-1] Leftover `fmt.Println` debug statement
- **File**: oracle.go:330 (`fmt.Println(sellOrderDataHex, tx.To())`)
- **Issue**: Raw stdout print in `validateCloseOrder` on the close-data-mismatch path. Violates zap-only logging, pollutes stdout, fires on every mismatch.
- **Fix**: Remove, or `o.log.Warnf("[ORACLE-ORDER] close data mismatch: sellData=%s to=%s", sellOrderDataHex, tx.To())`.

### [LOG-2] `%w` verb passed to `Errorf` (printf), not `fmt.Errorf`
- **File**: state.go:49 (`logger.Errorf("...for %s: %w", stateSaveFile, err)`)
- **Issue**: `%w` is only valid in `fmt.Errorf`. In a printf-style logger it renders `%!w(...)`. `go vet` flags this.
- **Fix**: Use `%v` (or `%s` with `err.Error()`).

---

## SHOULD Violations (Recommended)

### [ERR-1] `ErrOrderNotVerified` drops its `s` (orderId) argument
- **File**: error.go:108-110
- **Issue**: Signature `ErrOrderNotVerified(s string, err error)` ignores `s`; message is only `"order not verified: "+err.Error()`. Every caller passes the orderId (oracle.go:498, 504, 520, …) expecting it in the message — the id is silently lost.
- **Fix**: `"order not verified ("+s+"): "+err.Error()`.

### [ERR-2] `ErrOrderValidation` allocates a rich message but callers also increment separate metric labels — fine; but `ErrAmountMismatch`/`ErrOrderNotFoundInOrderBook` are dead code
- **File**: error.go:96-106 (and `ErrCreateDirectory`, `ErrGetHomeDirectory`, `ErrContextCancelled`, `ErrParseState` partially)
- **Issue**: Several constructors/codes defined but unreferenced by the package. Dead surface area.
- **Fix**: Remove unused error constructors + codes, or wire them where relevant.

### [LEAK-1] Startup spin loop ignores context cancellation
- **File**: oracle.go:156-159
- **Issue**: `for o.orderBook == nil { sleep(1s) }` has no `ctx.Done()` check. If shutdown happens before an order book is ever set, this goroutine leaks and logs a warning every second forever.
- **Fix**: `select { case <-ctx.Done(): return; case <-time.After(time.Second): }` in the wait loop.

### [ARCH-1] Consumer interfaces defined in `types/`, not at the consumer
- **File**: types/types.go:157 (`OrderStore`), :172 (`BlockProvider`), consumed in oracle.go:37-39
- **Issue**: Go convention (and the interface-design standard) is to define interfaces where they are consumed. `Oracle` consumes `OrderStore`/`BlockProvider` but they live in a shared `types` package, coupling consumer and implementations to a third package.
- **Fix**: Define the consumed interfaces in package `oracle`; let `eth`/store adapters satisfy them structurally.

### [ARCH-2] `Oracle` reaches into `OracleState` private fields
- **File**: oracle.go:757, 910-913
- **Issue**: Same access as RACE-2, viewed as design: breaks encapsulation of `OracleState`. State should expose intent-revealing, lock-guarded accessors.
- **Fix**: Add getters (also resolves RACE-2).

### [PERF-1] `readBlockState` does disk I/O while holding the write lock, every block
- **File**: state.go:119-147 (`ValidateSequence` → `readBlockState` → `os.ReadFile`), also GetLastHeight
- **Issue**: Each block re-reads + re-unmarshals the state file from disk under `rwLock.Lock()`. Serializes all state access behind synchronous file I/O on the hot path.
- **Fix**: Keep last block state in memory (already persisted via `saveState`); read disk only once at startup.

---

## CAN Improvements (Optional)

### [NIL-1] Inconsistent `o.metrics` nil-guarding
- **File**: oracle.go — `UpdateRootChainInfo` guards `o.metrics == nil` (652-653) but `processBlock`, `reorgRollback`, `run` call `o.metrics.*` unguarded. If `metrics` can be nil, these panic; if it never is, the guard is noise. Pick one contract.

### [CMT-1] `CommitCertificate` named return `err` is never assigned
- **File**: oracle.go:575, 635 — write failures `continue` without setting `err`; function always returns nil. Either propagate or drop the named return + doc that partial failure is tolerated.

### [TEST-1] Consolidate duplicated error constructors / magic strings
- error.go has two module namespaces with overlapping numeric codes (`OracleModule` 1-17, `OrderStoreModule` 1-5) — acceptable but easy to misread; a comment block clarifying the split would help.

---

## Detailed Findings by Skill

### 1. Go Code Standards & Interface Design
- `gofmt -l`: clean (no files need formatting).
- `go vet ./cmd/rpc/oracle/`: **could not run** — build fails in dependency `github.com/cockroachdb/swiss` (undefined `fastrand64`/`hashFn`) under Go 1.26.2. Environmental toolchain/dependency mismatch, **not** an oracle-package defect, but it means vet/tests are not currently executable here. Recommend pinning a supported Go version or bumping the `swiss` dep.
- Data races RACE-1, RACE-2 are the headline correctness issues.
- Interfaces returned as concrete types ✓; functions accept interfaces ✓; interfaces are small (BlockI/TransactionI/OrderStore) ✓ — but located in `types/` (ARCH-1).
- Error wrapping: package uses `lib.NewError` with string concatenation rather than `%w`; consistent with the codebase's `lib.ErrorI` pattern, acceptable.

### 2. Testing Standards
- Table-driven with `t.Run`: present across all four test files ✓.
- Good hand-rolled mocks (`mockOrderStore`, `mockBlock`, `mockTransaction`) ✓.
- **No `-race` / concurrency tests** despite the package's heavy use of goroutines, `sync.RWMutex`, and shared state — the very races above would surface under `go test -race`. (SHOULD)
- **No `t.Parallel()`**, no goroutine/WaitGroup stress on `OracleState` or `OracleDiskStorage` (both mutex-protected, untested for concurrency). (SHOULD)
- Tests not runnable in current env (dependency build break) — verify on a supported toolchain. (CAN)

### 3. Architecture & Design Patterns
- Clean chain-agnostic core: `Oracle` depends on `BlockProvider`/`OrderStore` ports, eth specifics injected — good hexagonal shape. Lingering Ethereum leak: `validateCloseOrder` does `common.BytesToAddress` + `tx.To()` comparison (oracle.go:325-333) with an in-code `TODO move this logic into the block provider`. (ARCH, noted)
- Consumer interfaces misplaced in `types/` (ARCH-1).
- Encapsulation break into `OracleState` internals (ARCH-2).

### 4. Observability
- Strong structured, tagged logging throughout (`[ORACLE-*]` prefixes), good level discipline.
- **`fmt.Println` leftover** (OBS-1) is a hard violation.
- `%w` in `Errorf` (LOG-2).
- Metrics coverage is thorough (block, order, lifecycle, submission, height, store-error). Nil-guard inconsistency (NIL-1).
- No sensitive data logged ✓ (order ids/addresses only).

---

## Recommended fix order
1. RACE-1, RACE-2 (correctness under concurrency)
2. OBS-1 `fmt.Println`, LOG-2 `%w`
3. ERR-1 dropped orderId
4. Add `-race` concurrency tests to lock in the fixes
5. PERF-1, ARCH-1/2, cleanup items
