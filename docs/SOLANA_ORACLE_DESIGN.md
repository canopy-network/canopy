# Solana Oracle Block Provider — Design

Date: 2026-07-04
Status: Approved for planning
Verified: 2026-07-04 (web verification of critical Solana claims — see "Verification" below), by Claude Opus 4.8 (1M context)

## Goal

Add Solana as a second source chain the Canopy oracle can witness, alongside the
existing Ethereum support, with minimal or no changes to the chain-agnostic core
(`oracle.go`, `state.go`, `order_store.go`, `order_validator.go`).

## Summary of the evaluation

The core oracle package is already well-abstracted: `oracle.go` depends on the
`types.BlockProvider` interface (`types/types.go:172-179`), not on the concrete
`eth.BlockProvider`. Adding a chain is fundamentally "write a new package that
implements `BlockProvider`/`BlockI`/`TransactionI`."

**Decision: copy the `eth/` package's interface-conformance pattern and
one-package-per-chain layout — do not copy its internals line-by-line.** Three of
`eth/block_provider.go`'s internal mechanics are specifically shaped by Ethereum
constraints that don't hold for Solana as scoped here:

1. A WebSocket-subscription-driven run loop (`connect()`/`monitorHeaders()`) —
   Solana's equivalent (`blockSubscribe`) is officially "unstable" in Agave, off
   by default, and not exposed by managed RPC providers like Alchemy (this
   deployment's target). The Solana provider polls instead.
2. A live contract-call token cache (`erc20_token_cache.go`, 3 RPC calls per
   unknown token) — an SPL mint's decimals are a fixed-offset byte directly in
   the Mint account's raw data, decodable from a single `getAccountInfo` call
   with no contract-call round trip.
3. A manual block-depth safety margin (`StartupBlockDepth`) — Solana exposes
   finality directly as an RPC `commitment` parameter (`finalized`), so no
   depth-based heuristic is needed.

There are exactly two Ethereum-specific leaks into the otherwise chain-agnostic
`oracle.go`, both flagged by the original author with a TODO (`oracle.go:326`).
Both must be extracted regardless of which second chain is added — they block
*any* second `BlockProvider` implementation, not just Solana.

## Changes to the core package (chain-agnostic, one-time)

The first three subsections below (`types.go`, `oracle.go`, `config.go`/`cli.go`)
are **required** — the Solana provider cannot compile or be constructed without
them. The `metrics.go` rename is **optional naming hygiene**, not a blocker: a
`sol` provider could call the existing `Eth`-prefixed methods today with no
functional bug, since only one chain provider ever runs per oracle process. It's
included here because it's cheap to do while touching this code, not because
anything depends on it.

### `types/types.go` — new `TransactionI` method

```go
// TransactionI (existing interface, add one method)
type TransactionI interface {
    Blockchain() string
    From() string
    To() string
    Hash() string
    Order() *WitnessedOrder
    TokenTransfer() TokenTransfer
    // MatchesOrderDestination reports whether this transaction's transfer
    // satisfies the sell order's expected contract/mint (contractBytes) and
    // recipient (recipientBytes), using chain-specific address formatting
    // and/or derivation. Replaces the two Ethereum-specific checks that
    // previously lived in oracle.go's validateCloseOrder.
    MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error)
}
```

### `oracle.go` — remove the two EVM-specific checks

`validateCloseOrder` (`oracle.go:324-369`) currently does two things that only
make sense for Ethereum:

1. Compares `sellOrder.Data` (formatted as a 20-byte EVM address) against
   `tx.To()` (the ERC20 contract address) — this is a "which asset/contract does
   this order accept" check, not a recipient check.
2. Compares `tokenTransfer.RecipientAddress` (assumed `0x`-prefixed hex) against
   `sellOrder.SellerReceiveAddress`.

Both get replaced by one call:

```go
ok, err := tx.MatchesOrderDestination(sellOrder.Data, sellOrder.SellerReceiveAddress)
if err != nil {
    o.metrics.IncrementValidationFailure("destination_conversion_error")
    return ErrOrderValidation("error validating transaction destination")
}
if !ok {
    o.metrics.IncrementValidationFailure("destination_mismatch")
    return ErrOrderValidation("transaction destination does not match sell order")
}
```

`eth.Transaction.MatchesOrderDestination` reproduces today's exact behavior
(EVM address formatting + hex comparison) so Ethereum behavior is unchanged.

### `lib/config.go` — new `SolBlockProviderConfig`

Mirrors `EthBlockProviderConfig` (`config.go:341-358`), embedded into `Config`
alongside it. Fields: `NodeUrl` (Alchemy HTTP RPC endpoint), `PollIntervalMs`,
`RetryDelay`, `StartupBlockDepth` is **not** carried over — Solana always polls
at `finalized` commitment instead.

### `cmd/cli/cli.go` — select provider by config

Replace the hardcoded `eth.NewEthBlockProvider(...)` call (`cli.go:140`) with a
switch on which chain's config is populated, constructing `sol.NewSolBlockProvider(...)`
when targeting Solana. Only one provider is ever constructed per oracle process
(a node witnesses exactly one source chain).

### `lib/metrics.go` — rename `Eth`-prefixed metric methods

The ~27 methods under `// ========== Eth Block Provider Metrics Helper Functions ==========`
(`metrics.go:1320` on) wrap generically-named underlying Prometheus gauges/counters
(e.g. `SetEthConnectionState` wraps `m.ConnectionState`) — the `Eth` prefix is only
in the Go method name, not the underlying metric. Since only one chain provider
runs per oracle process, there's no collision risk in sharing the same
gauges/counters across chains. Rename the methods to chain-generic names
(`SetConnectionState`, `IncrementRPCConnectionAttempt`, etc.) so a `sol` provider
doesn't have to call something named `IncrementEthReorgDetected`, and so a third
chain doesn't require a third copy-pasted set of metric methods.

## New package: `cmd/rpc/oracle/sol/`

```
sol/
  block_provider.go   — polling loop against the configured RPC endpoint
  block.go            — wraps a Solana block, implements types.BlockI
  transaction.go       — wraps a Solana transaction, implements types.TransactionI
  mint_cache.go        — LRU cache of decimals per mint, decoded from raw Mint account data
  error.go
```

RPC client: `github.com/gagliardetto/solana-go` — the de facto standard Go SDK
for Solana (analogous to `go-ethereum`'s `ethclient`), actively maintained,
provides `rpc.Client` with `GetSlot`/`GetBlock`/`GetAccountInfo` and the account
deserialization helpers needed for Mint account decoding.

### Order encoding

Lock/close order JSON travels as a **Memo program instruction**
(`MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr`), a separate top-level instruction
in the same transaction as the funds transfer (System Program transfer for
native SOL, SPL Token transfer for tokens). Both instructions land or fail
together since Solana transactions are all-or-nothing.

- **Lock order**: Memo instruction alone is sufficient — no transfer instruction
  required, since a lock order doesn't move funds. This is cleaner than
  Ethereum's "self-sent transaction" convention (`t.To() == t.From()` in
  `eth/transaction.go:93`), which Solana doesn't need since Memo doesn't require
  a transfer at all.
- **Close order**: Memo instruction (order JSON) + transfer instruction (the
  actual payout) as sibling top-level instructions.

Parsing scans the transaction's top-level instruction list by program ID (not
position — real transactions carry ComputeBudget instructions too) to find the
Memo instruction and the transfer instruction independently.

Transaction success comes directly from `getTransaction`'s `meta.err` field
(null = success) — no separate receipt call needed, unlike Ethereum's two-step
fetch-block-then-fetch-receipt pattern (`transactionSuccess()` in
`eth/block_provider.go:539`).

**Client-side constraints** (documented separately in
`docs/SOLANA_ORACLE_REQUIREMENTS.md`): the transfer and Memo instructions must
be top-level (not invoked via CPI), and transactions must not use Address
Lookup Tables. Both are enforceable because the swap client is a bespoke
Canopy-built tool, not a third-party wallet/DEX integration.

### Recipient matching: `SellerReceiveAddress` is a wallet pubkey

Following standard Solana practice (every wallet/exchange/dApp takes a wallet
address, never a specific token account, since a wallet has one address
regardless of asset, and a token account may not exist yet), `SellerReceiveAddress`
stores the seller's **wallet pubkey**, exactly as it stores a plain address for
Ethereum today.

`sol.Transaction.MatchesOrderDestination(contractBytes, recipientBytes)`:
1. Derives the expected Associated Token Account from `(recipientBytes, contractBytes)`
   — the standard `getAssociatedTokenAddress(owner, mint)` PDA derivation, pure
   and deterministic, no RPC call.
2. Compares the derived ATA against the transfer instruction's actual
   destination token account.
3. For native SOL transfers (no mint/contract involved), compares `recipientBytes`
   directly against the System Program transfer's destination — no derivation
   needed, since native transfers go straight to a wallet pubkey.

### Token metadata: no Metaplex dependency

`TokenInfo.Name`/`Symbol` are display-only (`types.go:224-241`, feed the
`String()` log formatter) — validation compares raw base-unit amounts directly
(`oracle.go:363`) and never uses decimals or name/symbol for correctness. Given
that, and given not all SPL mints register Metaplex Token Metadata (our own
test mint from `docker/sol-init.sh` has none), `mint_cache.go` fetches only
`Decimals` via a direct `getAccountInfo` + fixed-offset decode of the Mint
account, and leaves `Name`/`Symbol` blank. Revisit only if a real
observability need for token names emerges.

### Confirmation strategy: `finalized` commitment, no depth tracking

All `getSlot`/`getBlock` calls request `finalized` commitment. No
`StartupBlockDepth`-equivalent config exists for Solana — finality is a native
RPC parameter, not a heuristic.

### New-block detection: polling, no WebSocket subscription

Alchemy (the target RPC provider) does not expose `blockSubscribe` (Agave marks
it "unstable," off by default, requiring a validator flag managed providers
don't set). The provider polls `getSlot(commitment=finalized)` on a configurable
interval (`PollIntervalMs`) and fetches `getBlock` for any new finalized slots.

### Skipped slots

Solana's slot sequence has gaps — a leader can miss its turn and produce no
block for that slot at all. `getBlock` on a skipped slot returns a distinct RPC
error, not an empty block. The polling loop treats this specific error as "skip
this slot, advance past it" — distinct from a transient RPC error, which retries
the same slot with backoff (mirroring `eth`'s retry behavior for genuine
failures, but not applying that retry to skipped slots, which `eth`'s
one-height-per-block model never had to distinguish).

## Data flow (poll loop)

1. `cli.go` constructs `sol.NewSolBlockProvider(config.SolBlockProviderConfig, orderValidator, logger, metrics)`
   and passes it to `oracle.NewOracle(...)` as `types.BlockProvider`, same wiring
   shape as today's `eth` path.
2. `Start(ctx, height)` sets `nextSlot = height` (or computed from `finalized`
   minus nothing, since there's no startup depth) and spawns the poll loop.
3. Each tick: `GetSlot(finalized)` → `currentSlot`. If `currentSlot < nextSlot`,
   nothing to do (should not happen under `finalized` commitment; log a warning
   if it does, since finalized slots are monotonic).
4. For each slot in `[nextSlot, currentSlot]`:
   - `GetBlock(slot, commitment=finalized, maxSupportedTransactionVersion=0)`.
   - Skipped-slot error → advance past it, do not retry.
   - Other RPC error → retry with backoff, do not advance `nextSlot`.
   - Wrap the block into `sol.Block`; for each transaction with `meta.err == nil`,
     wrap into `sol.Transaction`, scanning instructions for Memo (→ `Order()`)
     and transfer (→ `TokenTransfer()`).
   - Send the block through `blockChan` (unbuffered, same backpressure design
     as `eth`).
5. `IsSynced()` reports true once there's no backlog between `nextSlot` and the
   last observed `currentSlot`.

## Testing

Mirror the existing `eth` test structure:
- `block_provider_test.go` — mock the RPC client interface (mirroring
  `EthereumRpcClient`/`EthereumWsClient` in `eth/block_provider.go:37-48`),
  covering: normal block processing, skipped-slot handling, transient RPC
  error retry/backoff, transaction-processing timeout budget.
- `transaction_test.go` — fixture transactions covering: Memo-only (lock order),
  Memo + SPL transfer (close order), Memo + native SOL transfer, transactions
  with ComputeBudget instructions interspersed, transactions with no Memo
  instruction (non-order transactions, should parse as no-op).
- `mint_cache_test.go` — Mint account decode correctness, LRU cache hit/miss.
- New `oracle_test.go` cases exercising `MatchesOrderDestination` through both
  `eth.Transaction` (regression — behavior must be unchanged) and a fake
  chain-agnostic transaction type, to confirm `oracle.go`'s refactored
  `validateCloseOrder` calls the interface method correctly.

## Verification

Critical Solana technical claims in this doc were verified against current
public documentation on 2026-07-04 by Claude Opus 4.8 (1M context). All
load-bearing claims held; no factual corrections were required.

| Claim | Source | Verdict |
|-------|--------|---------|
| `blockSubscribe` is unstable, off by default, requires `--rpc-pubsub-enable-block-subscription` + `--enable-rpc-transaction-history` | Agave / Solana RPC docs | Confirmed exactly |
| Alchemy does not expose `blockSubscribe` (supports only `accountSubscribe`, `programSubscribe`, `logsSubscribe`, `signatureSubscribe`, `rootSubscribe`, `slotSubscribe`) | Alchemy Subscription API docs | Confirmed |
| SPL Mint `decimals` is a `u8` at fixed byte offset 44, decodable from one `getAccountInfo` | `spl-token` `Mint` layout (mintAuthority 4–35, supply 36–43, **decimals 44**, freezeAuthority 45–76) | Confirmed, offset 44 |
| Memo program ID `MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr` | Official `spl-memo` | Confirmed exactly |
| Skipped slot returns a distinct `getBlock` RPC error | Confirmed — **two** codes: `-32007` (skipped / snapshot jump) and `-32009` (missing in long-term storage) | Confirmed (see note 1) |
| ATA derivation is deterministic, off-curve PDA, no RPC needed | Associated Token Program (seeds: owner, token program, mint) | Confirmed |
| `finalized` is a native commitment RPC parameter | Solana RPC docs | Confirmed |
| `github.com/gagliardetto/solana-go` provides `GetSlot`/`GetBlock`/`GetAccountInfo` | pkg.go.dev / repo | Confirmed (see note 2) |

**Note 1 — skipped-slot detection.** There are two skip-indicating error codes,
`-32007` and `-32009`; the poll loop's skip handling (§ "Skipped slots",
"Data flow" step 4) must match on **both**, not just one. Additionally, these
codes can also fire transiently when an RPC node is overwhelmed or rebooting, so
the "advance past, never retry" rule could theoretically skip a genuine block on
a provider hiccup. Under `finalized` commitment on Alchemy this is unlikely, but
the skip path should emit a log line for observability.

**Note 2 — Go SDK.** A `solana-foundation/solana-go` fork now exists alongside
`gagliardetto/solana-go`. Gagliardetto remains the widely-used community
standard (latest tag v1.16.0, alpha, unaudited, "APIs subject to change"); the
foundation fork may become canonical. Confirm the import path before pinning.

## Explicitly out of scope (see `docs/SOLANA_ORACLE_REQUIREMENTS.md`)

- CPI / inner-instruction scanning (transfer or Memo invoked via
  cross-program invocation).
- Address Lookup Table resolution.
- Metaplex Token Metadata (name/symbol) fetching.
- WebSocket-based block subscription.

Revisit if the swap client stops being a bespoke Canopy-built tool, or if
Alchemy (or a future provider) exposes `blockSubscribe`.
