# Agent Instructions for canoLiq Plugin

This file documents the canoLiq-specific gotchas. The general plugin architecture
(socket protocol, length-prefixed protobuf, CheckTx/DeliverTx pattern) is in
`plugin/go/AGENTS.md` — read that first.

## What canoLiq is

A liquid-staking sub-chain implemented as a sibling Go plugin to `contract/`.
Spec: `docs/plans/canoliq-implementation-plan.md` and the two whitepapers
referenced there.

Single binary, two plugins. `main.go` selects via `CANOPY_PLUGIN_MODE`:

| Value | Plugin |
|---|---|
| unset / `contract` | send tutorial (default) |
| `canoliq` | canoLiq liquid staking |

`CANOLIQ_CONFIG` optionally points at a JSON config file (`Config{ChainId,
DataDirPath, GenesisPath}`).

## Why canoliq has its own Plugin runtime instead of importing contract.Plugin

`contract.Contract.plugin` is unexported, so canoliq cannot drive contract's
FSM connection. `plugin.go` is a near-clone of `contract/plugin.go`. When you
change one you almost certainly need to change the other. The shared types
(`PluginConfig`, `PluginStateReadRequest`, `FSMToPlugin_*` oneof variants,
`KeyForAccount`, `KeyForFeePool`, `Marshal`/`Unmarshal`, etc.) come from the
`contract` package — re-import, do not redefine.

`JoinLenPrefix` is duplicated locally in `state.go` for the same reason
(avoiding an import cycle for trivial key building). If contract's version
changes, mirror it here.

`deliverMessageSend` is also duplicated rather than imported, so the canoliq
binary is a drop-in superset of the send tutorial.

## State key layout

All canoliq keys live under prefix `[]byte{10}`. Canopy core uses 1=accounts,
2=pools, 7=gov; `10` was chosen to leave room and stay unambiguous. Subdomains
are single-byte discriminators inside `JoinLenPrefix` segments — see
`state.go` for the canonical list. Phase 1 used `domainGlobals=1` …
`domainParams=11`; Phase 2 added `domainCliqStake=12`, `domainCliqUnstaking=13`,
`domainProposal=14`, `domainVote=15`, `domainBuybackOrder=16`, `domainSpend=17`,
`domainMultisig=18`, `domainInsurance=19`, `domainStakeIndex=20`. The validator
registry reuses `domainValIncent` + the `indexSingleton` discriminator.

**There is no range-scan.** The FSM only answers point reads, so iteration
requires an explicit index. `VestingIndex`, `ProposalIndex`, `CLIQStakeIndex`,
and the spend index (also a `ProposalIndex`) all exist for this reason. If
you add a collection that needs sweeping, add an index key alongside it.

## Reward sweep mechanics

`reward.go::ProcessRewards` runs in `EndBlock`. The committee pool key
(`contract.KeyForFeePool(chainId)`) is **shared** with transaction-fee
accumulation, so the sweep cannot just take the whole pool — it isolates the
per-block reward delta against `CanoliqGlobals.LastProcessedRewardPool` and
only processes the new inflow.

After the sweep the watermark is reset to the post-sweep pool balance, **not**
to zero, because user-rebate + net portions are credited back into the pool to
back cCNPY redemption. Only the validator/treasury/buyback shares physically
leave the pool (into plugin-owned scalar keys).

Consequences:

- Tests that mix deposit fees with reward injection must pin
  `LastProcessedRewardPool` to the pool's value *after* the deposit so the
  upcoming reward delta is isolated. See `TestCompositeDepositRewardRedeem`.
- A no-inflow block must be a no-op; covered by the third block in
  `TestRewardSweepMultiBlock`.

The validator share is distributed pro-rata across the canoLiq committee
validator set as of Phase 2. The set comes from a plugin-internal
`ValidatorRegistry` singleton (`KeyForValidatorRegistry`) which is
genesis-seedable and mutable via param-change governance. Phase 1.5 will
swap the source for a real Canopy validator-set readback; until then the
registry is the source of truth. **When the registry is empty** the legacy
single-aggregator address (`committeeAggregatorAddr` = 20 bytes of `0xCA`)
holds the entire share — Phase 1 baseline. Don't confuse the two paths in
tests: assert against either the per-validator keys or the aggregator key,
not both.

## Fee math

`fee.go::SplitFee` divides the fee into UserRebate / Treasury / Validators /
Buyback by bps. Integer truncation residual is **added to the treasury** so
the four parts sum to `feeAmount` exactly. `ValidateParams` enforces that the
four split bps fields total 10_000, plus Phase 2 invariants:
`insurance_bps ≤ 10_000`, `quorum_bps ≤ 10_000`, `pass_threshold_bps ≤ 10_000`,
`multisig_threshold ≤ len(multisig_signers)` when signers present, and
`cliq_unstaking_blocks ≥ voting_period_blocks` (so a voter cannot
stake → vote → unstake → unwind before tally).

`mulDiv` uses `math/big` for overflow safety — mirrors `lib.SafeMulDiv` in
Canopy core. Always use it for `(a*b)/c` over uint64 amounts.

## Whitepaper §7 reconciliation

The whitepaper's worked example assumes Canopy applies the 5% DAO cut
**upstream** of the plugin. canoliq's pool sees only `0.95 * X` already. Any
test that pins the whitepaper number (e.g., `TestWhitepaperSection7Reconciliation`)
must seed the pool with the post-DAO amount, not gross X. Don't double-apply
the DAO cut inside the plugin.

For X=1000 / 12% fee / 40-30-15-15 split with truncation **and Phase 2
defaults including `insurance_bps=1500`**, the reference output is:
yield=881, treasury=30, insurance=5, validators=17, buyback=17, sum=950.
Pre-Phase-2 baseline (insurance off) was treasury=35; the 5 uCNPY
delta is the auto-skim into `canoliq/insurance/pool`. Conservation
includes the insurance line — any new test that asserts the equation
`yield + treasury + insurance + validators + buyback == post-DAO` will
fail if you forget it.

## Genesis

`Canoliq.Genesis` is **idempotent** — re-runs short-circuit on
`globals.GenesisComplete`. The genesis source order is
`req.GenesisJson` → `Config.GenesisPath` → error. Tests inject via
`PluginGenesisRequest.GenesisJson`; production injects via the configured path.

Bucket bps must sum to 10_000, **and** recipients within each bucket must
also sum to 10_000. Both are validated by `validateGenesis`. Liquid tranches
(`cliffMonths==0 && vestMonths==0`) credit `KeyForCLIQBalance` directly;
otherwise a `VestingSchedule` is written and an entry is appended to
`VestingIndex`.

`CLIQTotalSupply = 100_000_000 * 1_000_000` (uCLIQ, 6-decimal parity with
uCNPY). Don't change this unit without auditing every fee/transfer path.

## Governance lifecycle (Phase 2)

`governance.go` implements proposal create / vote / tally / execute. A few
non-obvious things:

- **Voting weight is snapshotted by stake-time, not by proposal-time read.**
  The `CLIQStake` record carries `staked_at_height`. At vote delivery the
  handler rejects votes whose `staked_at_height > proposal.creation_height`.
  This defeats flash-stake without storing per-(proposal, voter) snapshot
  balances. Side-effect: a staker who *increases* stake after a proposal
  opens will have the new (post-creation) `staked_at_height` and lose
  voting eligibility on that proposal. That is intentional — re-staking
  cleanly resets eligibility.
- **Total staked snapshot is taken at create.** `Proposal.snapshot_total_staked`
  is `globals.total_staked_cliq` at the proposal's creation height. Quorum
  divides against that, not against the value at tally time. Don't change
  the snapshot semantics without auditing `proposalPasses`.
- **Tally cleanup deletes the proposal but not its votes.** Vote records are
  per-(proposal_id, voter); without a per-proposal voter index they cannot
  be enumerated. Stale vote keys are inert (no proposal to look them up by)
  and cheap. If you need to GC them, add a per-proposal voter index *first*.
- **Param-change payloads are full-set replacement.** `ProposalParamChange`
  carries a complete `CanoliqParams`, not a delta. This keeps `ValidateParams`
  invariants (split bps total, threshold ≤ signers, …) checkable in one shot.
- **`MessageCLIQProposalCreate.Payload` is a `google.protobuf.Any`.** Use
  `anypb.New(typed)` to build it; `unwrapPayload(any)` resolves it back via
  `contract.FromAny`. Only `ProposalParamChange`, `ProposalBuyback`, and
  `ProposalTreasurySpend` are accepted — any other type is rejected at
  create *and* at tally.

## Buyback (Phase 2)

`buyback.go::DeliverMessageBuybackExecute` consumes a passed
`BuybackOrder` keyed by proposal id. The order is **self-contained** — it
embeds the original `ProposalBuyback` payload (`cnpy_amount`,
`price_micro_cnpy_per_cliq`, `mode`) — because `dispatchPassed` deletes the
source `Proposal` record at tally cleanup. Don't rely on the proposal still
being readable when execute runs.

Modes:

- `BUYBACK_BURN` decrements `globals.cliq_total_supply` and
  `globals.cliq_circulating_supply` by `cliq_acquired`. CNPY moves from
  `buyback/pool` to `treasury/canoliq`; CLIQ disappears (no recipient).
- `BUYBACK_DISTRIBUTE_STAKERS` iterates `CLIQStakeIndex.Addresses`, computes
  `mulDiv(cliqAcquired, stake[s], totalStake)` per staker, credits liquid
  CLIQ balances, and adds the rounding remainder to the largest-stake
  staker. Empty staker set → re-credit `treasury/cliq` (no-op buyback).

Idempotency is via `BuybackOrder.executed`. Re-execute is rejected with
`ErrProposalAlreadyExecuted`. Tests rely on this — don't relax it.

## Treasury spend + multisig + timelock (Phase 2)

`treasury.go::queueTreasurySpend` runs from `dispatchPassed` and writes a
`TreasurySpend` with `executable_height = h + (timelock_blocks if amount >
treasury_threshold else 0)` and `requires_multisig = amount >
treasury_threshold`. The decision is **frozen at queue time** — later raising
the threshold via governance does not re-classify queued spends.

Above-threshold gating in `DeliverMessageDAOTreasurySpend`:

1. `current_height >= executable_height` (timelock elapsed).
2. `count(approvals) >= multisig_threshold` where each approval comes from
   a signer in `params.multisig_signers` *at execute time*. Approvals from
   signers later removed from the params set are ignored — `countMultisigApprovals`
   only counts approvals whose signer is currently authorized.

`MessageMultisigApprove` rejects non-signers up-front. The approval record
is per-(spend_id, signer); duplicate approvals from the same signer error.
Idempotency on the spend itself is via `TreasurySpend.executed`.

## Insurance auto-routing (Phase 2)

`ProcessRewards` skims `mulDiv(split.Treasury, params.InsuranceBps, 10_000)`
into `canoliq/insurance/pool` before crediting `treasury/canoliq`. Default
`insurance_bps=1500` (15% of treasury slice ≈ 1.5% of fee) — within WP §11's
"1–2% of treasury" framing. The insurance pool is a passive accumulator in
Phase 2; slashing-reimbursement disbursement is Phase 3.

When extending the reward sweep, **always update the conservation equation**
in tests: `treasury + insurance + buyback + validators + user_rebate +
net_to_users == delta`.

## Vesting

`vesting.go::unlockedAmount` is the cumulative-unlock function. Returns 0
before cliff, linearly interpolates between `StartHeight` and `EndHeight`,
saturates at `TotalAmount` after end. Degenerate schedule
(`EndHeight <= StartHeight`) returns the full amount once past the cliff —
used for "instant unlock at cliff" tranches.

`MessageCLIQClaimVested` reads the `VestingIndex` first, then issues a second
batch read for every schedule listed. Two FSM round-trips per claim is
intentional — needed because we cannot range-scan.

## Query layer (Phase 3)

`rpc.go` runs a small `net/http` mux **inside the plugin process** so
operators can read plugin-owned state without going through the FSM.
All routes are read-only — they go through `query.go` helpers, which
issue point `StateRead` calls and never write.

The HTTP server is gated by `Config.RpcAddress` (or env
`CANOLIQ_RPC_ADDR`). Empty disables it. `StartPlugin` returns the
long-lived `*Plugin` so `main.go` can drive `RPCServer.Shutdown`
during graceful exit.

Per-HTTP-request context is built by `(*RPCServer).queryContext()`,
which mints a fresh `*Canoliq` with `rand.Uint64()` as `fsmId`. This
matters because `Plugin.pending` is keyed by request id; FSM-originated
ids are guaranteed unique by the FSM, but plugin-originated reads
(state lookups inside an HTTP handler) need their own unique id to
avoid response-channel collisions under concurrency.

Collection routes depend on the existing indexes — `ProposalIndex`,
`SpendIndex`, `VestingIndex`, `CLIQStakeIndex`, `ValidatorRegistry`.
Anything that lacks an index (per-address redemption list, per-address
unstake list) is a point lookup by id, *not* a sweep. If you want a
new collection route, add the index alongside it on the write path
first — the FSM does not support range scans.

JSON encoding piggybacks on `@gotags` already attached to the proto
types, so the wire format matches what `canoliqctl` already produces.
Fields that are `google.protobuf.Any` (e.g. `Proposal.Payload`)
serialize as `{typeUrl, value}` — opaque to JSON consumers but
sufficient for reconciliation.

## Per-request Canoliq, long-lived Plugin

Every inbound FSM lifecycle message creates a fresh `*Canoliq` carrying the
request's `fsmId`. Concurrent requests do not share state. Block height is
tracked on the long-lived `*Plugin` and surfaced via `Plugin.CurrentHeight()`,
because `DeliverTx` requests do not carry a height — only `Begin`/`End` do.
`setHeight` is monotonic; out-of-order updates do not regress.

## fakeStore test hook

`Plugin.fakeStore` is a non-nil-only-in-tests field. When set, `StateRead`/
`StateWrite` answer from the in-memory map instead of the unix-socket FSM.
The hook interface is in `plugin.go`; the implementation is in
`fakeplugin_test.go` so it never ships in release binaries (Go excludes
`*_test.go` from `go build`).

`newTestCanoliq()` is the standard test entry point. Pre-seed via the
returned `*fakeStore` (account, pool, params, globals helpers in
`canoliq_test.go`). Set height via `c.plugin.setHeight(h)`.

## Error codes

`error.go` codes start at 100 to avoid colliding with `contract` package codes
(1–14). Phase 1 occupies 100–116; Phase 2 extends through 117–135 (CLIQ
stake/unstake, governance, buyback, treasury, multisig, insurance). Always
use the constructor functions; never build `*PluginError` literals directly
so the module field stays consistent.

## Common bugs caught in review

- **Forgetting to pin the watermark** after a deposit when also injecting a
  reward in the same test → reward sweep treats the deposit fee as reward
  inflow and yields are off.
- **Setting `GenesisComplete=true` but skipping `SaveParams`** → handlers
  that call `LoadParams()` get `DefaultParams` instead of the genesis-time
  override.
- **Using `len(r.Entries) == 0` as "key absent"** is correct, but follow-up
  code that unmarshals the (empty) bytes into a proto produces a zero-value
  struct — make sure that is the intended semantic for that field. The cCNPY
  balance path deliberately uses `DecodeUint64` which returns 0 for nil/short.
- **Account proto vs scalar uint64**. CNPY (in `Account`) is a protobuf
  message; cCNPY/CLIQ balances are bare 8-byte big-endian uint64. Don't
  cross the streams.
- **Treating `BuybackOrder` as a thin receipt.** Phase 2 stores the full
  `ProposalBuyback` payload on the order so execute is independent of the
  proposal's lifetime. If you ever introduce another deferred-execute path
  (governance → trigger), self-contain the artifact the same way — the
  proposal record is gone after tally.
- **Counting stale multisig approvals.** `countMultisigApprovals` filters
  against the *current* signer set. If you add a fast path that just counts
  approval keys without re-checking signer membership, you reintroduce the
  "removed signer can still satisfy threshold" bug.
- **Voting weight from the wrong source.** Always read `CLIQStake`, never
  `KeyForCLIQBalance` (liquid). Liquid CLIQ has zero governance weight by
  design.
- **Assuming `multisig_signers` is non-empty.** `DefaultParams()` ships
  with an empty signer set and `multisig_threshold=3`. Above-threshold
  spends are blocked until governance (or genesis) populates the set.
  `ValidateParams` allows the empty/threshold combination so tests don't
  trip; in production this is intentional inertia.

## Build & test

```bash
cd plugin/go
make build                              # builds go-plugin (both modes)
go test ./canoliq/... -v                # in-process tests, no FSM needed
CANOPY_PLUGIN_MODE=canoliq ./go-plugin  # run the canoliq plugin against a real FSM
```

The in-process test suite is the fast feedback loop. Localnet/Docker is only
needed for end-to-end checks against the real FSM socket.
