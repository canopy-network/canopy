# Code Review: cmd/rpc/oracle/eth

**Date**: 2026-07-04 02:57:00
**Reviewer**: Claude Code (full-review skill)
**Scope**: block.go, block_provider.go, transaction.go, erc20_token_cache.go, known_tokens.go, error.go (+ tests)

## Summary

| Skill | Status | MUST | SHOULD | CAN |
|-------|--------|------|--------|-----|
| go-review | ❌ | 3 | 3 | 3 |
| go-testing | ⚠️ | 0 | 2 | 0 |
| architecture | ⚠️ | 0 | 1 | 1 |
| observability | ✅ | 0 | 1 | 1 |

**Overall**: 3 MUST, 7 SHOULD, 5 CAN

Notes:
- `gofmt` clean.
- `go vet` / `go test` cannot run in this env — whole module fails to build against `cockroachdb/swiss` under go1.26 toolchain (`undefined: fastrand64/hashFn`). go.mod pins `go 1.24.0`. Toolchain/env issue, not a package defect, but **tests are currently unrunnable here** — could not empirically confirm they pass.

---

## MUST Violations (Blocking)

### [DATA-RACE] `synced` written without the mutex it is read under
- **File**: block_provider.go:277, 319 (writes) vs 142-145 (read)
- **Issue**: `IsSynced()` reads `p.synced` under `heightMu`. But `monitorHeaders` writes `p.synced = false` (277) and `p.synced = true` (319) with **no lock**. `monitorHeaders` runs on the `run` goroutine; `IsSynced()` is called by the consumer goroutine. A mutex held on only one side does not synchronize — this is a data race.
- **Fix**: Lock `heightMu` around both writes, or make `synced` an `atomic.Bool`. Prefer atomic since `nextHeight` is single-goroutine and the mutex is otherwise only guarding this one flag.

### [NO-PRINT] `fmt.Println` on production path
- **File**: transaction.go:141
- **Issue**: `fmt.Println("validated lock order", ...)` fires on every validated lock order. Violates "no fmt.Print* for logging; use zap (lib.LoggerI)". Also leaked debug `fmt.Println` comments at 121-122.
- **Fix**: Remove, or route through the injected logger at Debug. `parseDataForOrders` has no logger; drop the line entirely.

### [CTX-BLOCK] Channel send ignores context cancellation
- **File**: block_provider.go:407
- **Issue**: `p.blockChan <- block` on an **unbuffered** channel. If the consumer stalls, this blocks forever — `timeoutCtx.Done()` and `ctx.Done()` are not selected on during the send. Provider cannot shut down or hit its processBlock time limit while blocked; goroutine leak on cancellation.
- **Fix**:
  ```go
  select {
  case p.blockChan <- block:
  case <-timeoutCtx.Done():
      return next
  }
  ```

---

## SHOULD Violations (Recommended)

### [CONSTRUCTOR] `logger.Fatal` + connection side-effect in constructor
- **File**: block_provider.go:71-98 (Fatal at 75, `ethclient.Dial` at 73)
- **Issue**: `NewEthBlockProvider` dials an Ethereum client at construction and calls `logger.Fatal` on failure (kills process). Constructors should be pure and return `error`. This client is also **separate** from the ones `connect()` creates and is **never Close()d**.
- **Fix**: Return `(*EthBlockProvider, error)`; let caller decide on fatal. Consider lazy-dialing the token-cache client (or share the `connect()` rpcClient).

### [INPUT-STRUCT] 9 positional args on `UpdateEthBlockProviderMetrics`
- **File**: block_provider.go:380,397,414,473,554,568; erc20_token_cache.go:59,97
- **Issue**: `UpdateEthBlockProviderMetrics(0,0,receiptTime,0,0,1,0,0,0)` — positional meaning is opaque and easy to transpose. Violates ">2 args → input struct." Call sites already show the smell (long runs of `0`).
- **Fix**: Pass a struct, or split into focused methods (`RecordReceiptFetch`, `RecordBlockProcessed`, …). Several are already split — finish the job and retire the mega-method.

### [ERR-WRAP] Underlying errors discarded
- **File**: erc20_token_cache.go:67,75,83 (return bare `ErrTokenInfo`); transaction.go swallows unmarshal errors as "normal" (104-113, 143-147)
- **Issue**: `TokenInfo` returns sentinel `ErrTokenInfo` with no `%w` wrap of the RPC error — loses which call/why. `parseDataForOrders` returns validation *and* genuine unmarshal errors identically; caller (`processTransaction:485-490`) treats all as "non-JSON, normal" and never surfaces a real unmarshal failure.
- **Fix**: `fmt.Errorf("%w: %v", ErrTokenInfo, err)`. Distinguish "not an order" (nil) from "malformed order JSON" (real error) in `parseDataForOrders`.

### [TEST-GAP] Core order-detection logic untested
- **File**: transaction.go `parseDataForOrders` (78-181); the covering test is commented out at block_provider_test.go:341-508
- **Issue**: The heart of the package — lock/close order extraction from tx data (self-send lock, zero-amount ERC20 lock, positive-amount close, negative guard) — has **no active test**. `processTransaction`, `transactionSuccess`, `monitorHeaders`, `connect`, `decodeString`, `decodeUint8` also untested.
- **Fix**: Restore/rewrite the commented `checkTransfer`-era table test against current `parseDataForOrders`/`processTransaction`. Add `decodeString` boundary cases (offset/length overflow paths at 140,144,149).

### [TEST-RACE] No evidence of `-race` gate
- **File**: package tests
- **Issue**: Given the confirmed `synced` race, tests must run under `-race` in CI. No indication they do; build currently fails module-wide anyway.
- **Fix**: Add `go test -race ./...` to CI; resolve the go1.24/1.26 toolchain mismatch so the module builds.

### [ARCH] `TokenInfo` duplicated across bounded contexts
- **File**: known_tokens.go:26-32 defines `eth.TokenInfo` while the code uses `types.TokenInfo` everywhere else
- **Issue**: Two `TokenInfo` types; `eth.TokenInfo` + `KnownTokens` map appear unused by the provider. Confusing duplicate, drift risk.
- **Fix**: Delete `eth.TokenInfo`/`KnownTokens` if unused, or fold the reference data into `types`.

---

## CAN Improvements (Optional)

### [DEAD-CODE] Unused/duplicate error vars
- **File**: error.go:9-21
- `ErrInvalidKey` and `ErrInvalidPrivateKey` are identical ("invalid private key"). Many (`ErrGasPriceEstimation`, `ErrNonceRetrieval`, `ErrGasEstimation`, `ErrTransactionSigning/Sending`, `ErrMaxRetries`, `ErrTransactionFailed`) look unused in this package. Prune.

### [DEAD-CODE] Commented-out blocks
- **File**: block_provider_test.go:341-508 (168-line commented test), block_provider.go:123/129/409, transaction.go:121-122
- Remove; git history preserves them.

### [DOC] Missing interface doc comment
- **File**: block_provider.go:50 `OrderValidator` — exported, no doc comment (golint). The sibling interfaces have one.

### [LOG-LEVEL] Expected condition logged at Error
- **File**: block_provider.go:572 `Errorf("... ERC20 transfer failed on-chain, ignoring")`
- A failed on-chain tx is an expected observation, not an oracle error — consider Warn to avoid alert noise.

### [LOG-NOISE] `logAsciiBytes` on every non-order tx
- **File**: block_provider.go:487 → 579
- Called for every tx whose data isn't a valid order (the common case). Debug level so low-risk, but high volume.

---

## Detailed Findings by Skill

### 1. Go Code Standards & Interface Design
Positives: interfaces are small (1-3 methods), consumer-defined, and accepted as params while concrete types are returned (`EthereumRpcClient`, `EthereumWsClient`, `OrderValidator`, `ContractCaller`) — ISP/DIP followed cleanly. `NewTransaction` wraps sender error with `%w`. Context is first param throughout, never stored in structs, timeouts derive from parent ctx (`transactionSuccess`, `callContract`). `heightMu` correctly a pointer.
Issues: the data race, `fmt.Println`, blocking send, Fatal-in-constructor, 9-arg metrics call, and error-wrapping gaps above.

### 2. Testing Standards
Table-driven with `t.Run`, `go-cmp` for diffs, mocks implement the consumer interfaces — good structure. Gaps: the primary order-parsing path is untested (commented out), several functions untested, and no `-race`. Tests are deterministic/hermetic (no network). Module build failure blocks execution here.

### 3. Architecture & Design Patterns
Clean ports-and-adapters: `eth` is an adapter behind `types.BlockProvider`/`types.BlockI`/`types.TransactionI`; ethereum SDK types stay inside the adapter and are mapped to domain types. Dependency direction correct. One smell: duplicate `TokenInfo` type in `known_tokens.go` crossing the `types` boundary.

### 4. Observability
Structured logging via injected `lib.LoggerI`, consistent `[ETH-*]` prefixes, no secrets/PII logged. Metrics are comprehensive (connection state, sync status, lag, reorg, retries-by-attempt, batch size) and all `*lib.Metrics` methods are nil-safe (verified `if m == nil` guards) — so the redundant nil-checks in erc20_token_cache.go:58,96 are harmless but inconsistent with unguarded call sites. Minor: one Error-level log for an expected condition, and the 9-positional-arg metrics API (noted as SHOULD).
