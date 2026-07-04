# Handoff: Solana Oracle Block Provider + Build Recovery

**Repo:** `~/canopy/canopy/oracle` (module `github.com/canopy-network/canopy`)
**Branch:** `main`
**Date:** 2026-07-04

---

## Goal

Two threads:
1. **(Done, merged)** Add Solana as a second source chain for the Canopy oracle — a new `cmd/rpc/oracle/sol` package satisfying the existing `BlockProvider`/`BlockI`/`TransactionI` interfaces, alongside the Ethereum provider. Plan: `docs/superpowers/plans/2026-07-04-solana-oracle-block-provider.md`.
2. **(Open)** Write an implementation plan for the **missing unit tests** on `sol/transaction.go` (the parsing engine) — the original request that triggered the build investigation. Not yet written.

---

## Current Progress

### Solana feature — merged to `main`
All 13 plan tasks implemented across commits ending at `37714226`. The `sol` package exists and its tests pass: `error.go`, `mint_cache.go`, `transaction.go`, `block.go`, `block_provider.go` (+ production `realClient` adapter), CLI wiring via `OracleConfig.SourceChain` ("ethereum" default / "solana").

### Build recovery — committed
- `121ec335 fix(oracle): restore dropped WIP definitions to repair broken build`
- `41b8447f build: pin toolchain go1.25.3 in go.mod`

**Verified green** (under stock toolchain — see below):
```
GOTOOLCHAIN=go1.25.3 go build ./...
GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/... ./lib/ ./controller/...
```
All pass: `cmd/rpc/oracle`, `sol`, `eth`, `types`, `lib`, `controller`.

---

## What Happened (root cause — read before touching the build)

The solana feature was originally built on a base commit that was **missing ~1030 lines of uncommitted WIP**. That WIP had been `git stash`ed at the start of the subagent workflow and never restored. It contained definitions the rest of the code depends on:
- `lib/swap.pb.go`: `SellOrder.IsLocked`, `CloseOrder.Equals`, `OrderBook.Copy`
- `lib/util.go`: `AtomicWriteFile`
- `lib/metrics.go`: oracle `Metrics` fields (`OrdersWitnessed`, `SafeHeight`, `SourceChainHeight`, `OrderValidationTime`, etc.)
- `controller/controller.go`: `oracle` field + import; `New()` signature takes `*oracle.Oracle`
- `lib/certificate.go`, `lib/log.go`, `cmd/rpc/{client,query,types}.go`, `controller/block.go`

**Why it went undetected:** `cockroachdb/swiss` (pulled in via `cockroachdb/pebble/v2` → the store layer) fails to compile under the machine's default custom Go toolchain `go1.26.2-X:nodwarf5` (undefined runtime symbols `hashFn`, `fastrand64`, `getRuntimeHasher`). Go stops at that dependency **before** type-checking `lib/`, so every implementer/reviewer subagent hit the swiss wall, reported "build blocked by swiss, code looks correct," and the real compile errors never surfaced. The whole feature merged without the package ever compiling.

**Recovery source:** the dropped work still exists as `stash@{1}` (`WIP on eth-oracle: 4daccba6`). It was reconciled against the branch and restored in `121ec335`.

---

## What Worked

- **`GOTOOLCHAIN=go1.25.3`** — stock 1.25.x compiles `swiss` fine and reaches the real errors. This is the reliable way to build/test locally.
- **Recovering from `stash@{1}` rather than `oracle-bak2`** — the stash was the exact dropped set, more precise than the bak repo.
- **Per-file classification** for the merge: blind-restore stash-only files via `git checkout stash@{1} -- <file>`; hand-merge the few files both the stash and the solana commits touched (`cmd/cli/cli.go`, `cmd/rpc/oracle/oracle_test.go`); skip files already fixed (`lib/metrics.go` was fixed manually).
- **For `oracle_test.go`**: taking the stash's coherent version wholesale, then re-adding the Task-3 `mockTransaction.MatchesOrderDestination` method + `bytes` import, then updating 3 post-refactor error-string expectations. (The branch's own `oracle_test.go` was internally inconsistent — struct literals referenced fields the struct type didn't declare — because it too was never compiled.)

## What Didn't Work

- **`toolchain go1.25.3` in go.mod does NOT fix local builds.** Go's toolchain directive only selects a toolchain **≥** the running one; the default `go1.26.2-X` is treated as *newer* than 1.25.3, so Go ignores the directive and keeps using the broken 1.26.2. The directive is kept only to document intent and help CI/older environments. **You must use the `GOTOOLCHAIN=go1.25.3` env var to actually downgrade.**
- **Piecemeal patching of `oracle_test.go`** — chasing one compile error at a time was a dead end because the file was coherently broken in many places; wholesale replace-then-reapply was correct.

---

## Next Steps

1. **DONE (2026-07-04).** Missing-tests plan written and fully executed via `superpowers:subagent-driven-development` on branch `sol-transaction-tests` (worktree `.worktrees/sol-transaction-tests`). Plan: `docs/superpowers/plans/2026-07-04-sol-transaction-missing-tests.md`. All 8 coverage-gap tasks implemented, each independently reviewed (spec + quality, all Approved, no fixes needed) and committed:
   - `parseSPLTransfer` TransferChecked + edge cases — commit `746727f9`
   - `parseNativeTransfer` happy path + edge cases — commit `bac94e6c`
   - `parseTransfer` SPL precedence + no-transfer error — commit `b93b9a45`
   - `parseInstructions` malformed-JSON + error-propagation — commit `c9986264`
   - `MatchesOrderDestination` non-transfer/mismatch/ATA integration — commit `e98cecb6`
   - `TokenTransfer()` field mapping + accessors — commit `f3ff3648`
   - `resolveInstructions` out-of-range + empty cases — commit `a5ef8422`
   - `mintCache.Decimals()` fetch/cache/error paths — commit `2941f27b`

   `sol` package now has 38 tests (up from 11), all green: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -v`. Broader regression check also green: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/... ./lib/ ./controller/...`. Branch not yet merged to `main` — pending final whole-branch review.

   **New environment quirk found:** `GOTOOLCHAIN=go1.25.3 go build ./...` fails in a fresh worktree checkout with `pattern all:web/explorer/dist: no matching files found` (`cmd/web/explorer/dist` doesn't exist and isn't checked in). This did NOT reproduce on `main` in the original checkout — likely a stale build-cache hit there masking the same underlying gap. Scoped builds (`go build ./cmd/rpc/oracle/... ./lib/... ./controller/...`) are unaffected and succeed. Not caused by this work; worth a separate investigation into whether the frontend dist assets need a build step documented somewhere.

2. **CI toolchain guard.** Ensure CI runs with `GOTOOLCHAIN=go1.25.3` (or a stock Go) so a swiss-masked build can never hide real compile errors again. Consider documenting the custom-toolchain caveat in the repo README/CONTRIBUTING.

3. **Optional recovery leftover.** The stash's `oracle_test.go` had a richer per-case `orderBook` version of `TestOracle_ValidateProposedOrders`; the merged version passes `nil` as the order book at `oracle_test.go` line ~492 (`ValidateProposedOrders(tt.orders, nil)`). If you want that table test to exercise book-cross validation, port the stash's per-case `orderBook` field. Not required for green.

4. **Drop the stash once satisfied:** `git stash drop stash@{1}` (kept as a safety net; verify nothing else missing first).

5. **Unrelated known failure:** `cmd/auto-update` `TestGetLatestPluginReleaseFiltersByPluginStream` fails — pre-existing, untouched by this work, not oracle-related.

---

## Key Commits (main)
```
41b8447f build: pin toolchain go1.25.3 in go.mod
121ec335 fix(oracle): restore dropped WIP definitions to repair broken build
37714226 fix(sol): resolve test compilation issues, uncomment metrics, remove hygiene files
... (13 solana feature commits back to b71abfa3)
4daccba6 chore(docker): ...   ← base the feature was (incorrectly) built on
```
