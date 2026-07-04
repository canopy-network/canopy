# Code Review: Oracle Commit Series (`4aa18637^..HEAD`)

**Date**: 2026-07-04 04:25:23
**Reviewer**: Claude Code (full-review skill)
**Range**: `4aa18637^..HEAD` (20 commits) — oracle/eth cleanup, order-validation fixes, race fixes, metrics refactor, debug endpoint
**Scope**: Go source only (docker volumes, docs, HTML excluded)

## Summary

| Skill | Status | MUST | SHOULD | CAN |
|-------|--------|------|--------|-----|
| go-review | ⚠️ | 1 | 2 | 6 |
| go-testing | ✅ | 0 | 0 | 2 |
| architecture | ✅ | 0 | 0 | 0 |
| observability | ⚠️ | (see MUST) | 0 | 1 |

**Overall**: 1 MUST, 2 SHOULD, ~8 CAN

Overall high quality. The series directly resolves prior-round concerns (data races, blocking channel send, Fatal-in-constructor, error wrapping, dead code). Race fixes and the close-order sender check were traced and confirmed correct. Only one blocking issue: a leftover debug log line.

## MUST Violations (Blocking)

### [LOG] Leftover `DEBUG ... FAILED` error log on a routine reject path
- **File**: `fsm/automatic.go:155`
- **Issue**: `s.log.Errorf("DEBUG HandleCertificateResults FAILED: ...")` ships debug-tagged noise at Error level on a normal validation reject (stale committee height) — a guard that fires routinely, not a fatal.
- **Fix**: Remove the line; if the signal is wanted, downgrade to `Debugf` and drop the "DEBUG ... FAILED" wording.

## SHOULD Violations (Recommended)

### [DESIGN] `DebugOrder` hardcodes `OracleEnabled: true`
- **File**: `cmd/rpc/oracle/oracle.go:816`
- **Issue**: Field is always `true` when the oracle is enabled (carries no info); a nil oracle returns nil, so the caller can't distinguish "disabled" from "no data".
- **Fix**: Have the caller set `OracleEnabled` based on whether the oracle exists, or drop the always-true field.

### [DOC] Dex-batch failure now aborts the whole certificate result
- **File**: `fsm/automatic.go:163-165`
- **Issue**: Returning the `HandleDexBatch` error (vs prior log-and-continue) also skips `HandleCommitteeSwaps`, `HandleCheckpoint`, byzantine handling, and `UpsertCommitteeData`. Intent is sound and stays consensus-safe (deterministic FSM/store errors), but the behavior change is broader than "return the dex error".
- **Fix**: Add a one-line comment documenting that a dex-batch failure intentionally abandons the rest of the receipt.

## CAN Improvements (Optional)

- `cmd/rpc/oracle/state.go:62-64` — `NewOracleState` swallows all `readBlockState` errors as "no prior state"; a corrupt state file is indistinguishable from missing, masking data loss on restart. Log at Warn when the error is not file-not-found.
- `cmd/rpc/oracle/state.go:132-137` — `sourceChainHeight` intentionally not advanced on gap/reorg return paths; asymmetry looks accidental — add a comment.
- `cmd/rpc/query.go:204-259` (`trackClosedOrders`) — An order that closes then reappears is reported in both `Orders` and `ClosedOrders` until the 24h TTL. Delete any `current` id from `s.closedOrders` before building the closed list.
- `cmd/rpc/query.go:234` — `s.closedOrders`/`s.lastOrderIDs` retain live `*lib.SellOrder` pointers from the controller book; snapshot could read stale if backing memory is reused. Copy the order (or surfaced fields) when snapshotting.
- `cmd/rpc/oracle/oracle.go:392` — Buyer-mismatch Warn logs on-chain addresses (`BuyerSendAddress`, `tx.From()`); public data, not secret, but confirm counterparty addresses at Warn are acceptable in prod.
- `block_provider.go:296` — Status-ticker reads `p.nextHeight` unlocked; race-free (single-goroutine owner) but mixed-locking style on the same line invites misuse. Add a comment noting single-goroutine ownership.
- `block_provider.go:539` (`processTransaction`) — `transactionSuccess` receipt RPC called for every witnessed order incl. non-ERC20 self-lock orders; pre-existing, confirm intended.
- `block_provider_test.go:449` — `processBlocksCancelledSend` relies on a 200ms real-time `time.Sleep` to hit cancellation-during-send; timing-dependent, could flake on loaded CI. Prefer a test-controlled signal over wall-clock.
- `transaction_test.go:301` / `known_tokens.go` — Tests couple to another package's unmarshal behavior (`{}` valid, `chainId 0` panics); acceptable, but will silently change meaning if order structs gain required fields.

## Verifications (confirmed correct — not defects)

- **Race fixes (93d4fcc2)**: Hot-path disk I/O moved to in-memory `blockState` cache; all accesses correctly guarded (`ValidateSequence` write-lock; getters read-lock; `saveState`/`removeState` write-lock for pointer swap with disk I/O outside the lock). No copies of the mutex-bearing `OracleState`. No lock-ordering issues. New getters `GetSourceChainHeight`/`SubmissionCounts` fix prior unlocked reads in `updateMetrics`.
- **Close-order sender check (3617b644)**: `validateCloseOrder` is the only path; `BuyerSendAddress == tx.From()` runs before recipient/amount checks with no early-return bypass; case-mismatch is a non-issue (`hex.DecodeString` case-insensitive); test coverage exists.
- **Metrics refactor (6ba897ba)**: `UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{...})` behavior-preserving, all 8 call sites updated, all helpers nil-safe.
- **eth pkg**: `gofmt -l` clean, `go vet` clean, `go test -race` passes (pinned `GOTOOLCHAIN=go1.24.0`). Dead sentinels/imports correctly removed; `%w`/`%v` wrapping correct; consumer-defined `oracle.BlockProvider` interface (3 methods, at consumer) correctly placed.
- **Tests**: `fsm/swap_test.go` table-driven, covers deadline-wins / before-deadline / amount-mismatch.

**Note**: `go build` fails on a pre-existing `cockroachdb/swiss` vs Go 1.26 toolchain mismatch — environmental, unrelated to this diff.
