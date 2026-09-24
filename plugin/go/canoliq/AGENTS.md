# Agent Instructions for canoLiq Plugin

This file documents the canoLiq-specific gotchas. The general plugin architecture
(socket protocol, length-prefixed protobuf, CheckTx/DeliverTx pattern) is in
`plugin/go/AGENTS.md` — read that first.

## What canoLiq is

A liquid-staking sub-chain implemented as a sibling Go plugin to `contract/`.
Spec: the canoLiq v1.2 papers (captured in the `canoliq-papers` skill; source
PDFs in repo-root `docs/`). Rollout work is tracked in
`docs/plans/canoliq-release-plan.md`.

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

All canoliq keys live under prefix `[]byte{20}`. Canopy core reserves the
single-byte range 1-15 (1=accounts, 2=pools, 7=gov, 10=supply, …); writing
under any of those makes `FSM.StateWrite` panic, so canoliq uses `20` — outside
the reserved range — and declares it in `CanoliqConfig.CustomStatePrefixes` so
the handshake validates it. Subdomains
are single-byte discriminators inside `JoinLenPrefix` segments — see
`state.go` for the canonical list. Phase 1 used `domainGlobals=1` …
`domainParams=11`; Phase 2 added `domainCplqStake=12`, `domainCplqUnstaking=13`,
`domainProposal=14`, `domainVote=15`, `domainBuybackOrder=16`, `domainSpend=17`,
`domainMultisig=18`, `domainInsurance=19`, `domainStakeIndex=20`. The validator
registry reuses `domainValIncent` + the `indexSingleton` discriminator.

**Prefer an explicit index over a range-scan.** `PluginStateReadRequest`
does carry `Ranges` (prefix iteration, honored by `fsm/state.go::StateRead`),
but every entry crosses the unix socket, so it is only worth it when the whole
prefix is the answer — the stuck-redemption alert (`MatureRedemptionPrefix`),
the ejection tombstones (`EjectedValidatorPrefix`), and the live-validator scan
in `registry.go`. For anything read by key, keep the explicit index:
`VestingIndex`, `ProposalIndex`, `CPLQStakeIndex`, and the spend index (also a
`ProposalIndex`) all exist for this reason.

## Reward sweep mechanics

`reward.go::ProcessRewards` runs in `EndBlock`. It does **not** sweep the
committee fee pool: Canopy distributes and zeroes `KeyForFeePool(chainId)` in
its own `EndBlock`, before this hook runs, compounding the reward into the
bonded stake of the committee's validators. The sweep therefore observes the
block-over-block *growth of that bonded stake* and isolates it against
`CanoliqGlobals.LastProcessedRewardPool`, which means "last observed aggregate
**canoLiq-owned** stake", not a pool balance and not the committee total. Read
`ProcessRewards`' doc comment before changing any of it.

### Membership and ownership are two different roles

This is the single thing to get right in this file, and the thing that was
wrong on mainnet.

- **Membership** is `Validator.committees[]` containing `chainId`. It decides
  who shares the 15% validator-incentive slice, pro-rata by stake, in
  `distributeValidatorShare`. Every committee member qualifies, because that
  slice pays operators for running the committee regardless of whose CNPY sits
  in their bond.
- **Ownership** is `Validator.output` appearing in
  `params.stake_output_addresses`. It decides the received reward `R`. Canopy
  returns a bond and its early-withdrawal rewards to `output`
  (`fsm/committee.go::DistributeCommitteeReward`), so `output` is the record's
  economic beneficiary. Every other bond on the committee is an operator
  earning on their own collateral (WP §1.1: operators bond their own CNPY, and
  it is their own CNPY that gets slashed).

Summing growth over the whole committee credits cCNPY holders with other
people's yield. On mainnet committee 29 that took a 10 CNPY deposit to ~562x in
forty minutes off an 8,235 CNPY bond canoLiq did not own — and because the
phantom reward lands in the escrow pool, the escrow guard in claim-redemption
passes and the payout becomes an un-backed account credit. WP §3.3 is explicit
that `R` is canoLiq's stake-weighted share.

`ValidatorRegistryEntry.Owned` persists the verdict rather than leaving each
consumer to re-derive it, so the snapshot, `/v1/restaking` and the alerts
cannot drift from what the sweep acted on.

Two invariants that live in docs, not code, because Canopy cannot express them:

1. A listed output address may only receive records whose **entire** bond is
   canoLiq principal. One `output` per record means a mixed bond (operator
   collateral plus pool CNPY) pointed at a listed address reproduces the same
   over-attribution with a different trigger.
2. An owned bond must set `compound=true` and must not be unstaking, or Canopy
   pays its reward to the output address as liquid CNPY where stake-growth
   observation cannot see it. `AlertOwnedBondNotCompounding` catches this. Do
   **not** try to capture that reward by diffing the output account's balance —
   it also moves for transfers and gas, and treating those as reward
   reintroduces the same class of bug.

**An empty ownership set means `R = 0`, and that is correct** whenever the
protocol has no staked position. Under-crediting truthfully beats minting yield
against stake it does not own. `AlertOwnedStakeMissing` keeps that from being a
silent default. It is reachable, not permanent: a `delegate=true` record with
`output` set to a canoLiq address makes it non-zero. Canopy has the delegation
primitive (`lib.Validator.delegate`, `fsm/committee.go::GetDelegates`); what is
missing is a signable identity for the escrow pool, which is plugin state under
prefix `{20}`.

### The rest of the sweep

The observed member set is the plugin's `ValidatorRegistry` singleton
(`KeyForValidatorRegistry`), reconciled against Canopy's live committee
membership on every call — see `registry.go`, which also explains why the
observation is per-validator (a newcomer's bond must not read as reward) and
why governance ejection needs the `KeyForEjectedValidator` tombstone
(`domainEjected`) to survive the next sync. The genesis `validatorRegistry`
block is a bootstrap convenience only; the first `EndBlock` overwrites it.

After the `min(per-validator, aggregate)` estimate, a plausibility clamp bounds
one block to `params.max_reward_bps_per_block` of owned stake. The excess is
**deferred in `globals.carried_reward`, not discarded** — nested-delegate
rewards arrive by stake-weighted lottery, so a legitimate win is lumpy and
truncating would quietly destroy real yield. It has to be an explicit field:
the registry re-baselines every member at its live stake each block, so the
watermark alone cannot carry it and the `min()` would zero it on the next
block. `0` means "unset" and backfills to the default, because a params record
written before the field existed decodes it as zero and must not inherit "no
clamp"; `10_000` is the deliberate off switch, since a cap of 100% of owned
stake can never bind.

Consequences:

- Tests seed a live `contract.Validator` record per member (`setCommitteeStake`)
  *and* pin `LastProcessedRewardPool`; a registry entry with no matching live
  validator record is dropped by the sync. See `seedReward`.
- Tests that expect reward must ALSO declare ownership in the params held in
  **state** — `ProcessRewards` calls `LoadParams()`, not whatever a test passes
  to a `Deliver*` handler. `seedStakeOutputParams` does that, and amends rather
  than replaces so it composes with tests that tuned other fields.
  `setCommitteeStakeWithOutput` builds the unowned case.
- Tests with synthetic ratios (1000 uCNPY of reward compounded into a 1000
  uCNPY bond) trip the clamp. `disableRewardClamp` turns it off and says why.
- A no-growth block must be a no-op; covered by the third block in
  `TestRewardSweepMultiBlock`.

The validator share is distributed pro-rata over that same set, weighted by
live `StakedAmount` and **ignoring `Owned`** — that is the membership role, see
above. **When the registry is empty** the legacy single-aggregator address
(`committeeAggregatorAddr` = 20 bytes of `0xCA`) holds the entire share — Phase
1 baseline. Don't confuse the two paths in tests: assert against either the
per-validator keys or the aggregator key, not both.

## Fee math

`fee.go::SplitFee` divides the fee into UserRebate / Treasury / Validators /
Buyback by bps. Integer truncation residual is **added to the treasury** so
the four parts sum to `feeAmount` exactly. `ValidateParams` enforces that the
four split bps fields total 10_000, plus Phase 2 invariants:
`insurance_bps ≤ 10_000`, `quorum_bps ≤ 10_000`, `pass_threshold_bps ≤ 10_000`,
`multisig_threshold ≤ len(multisig_signers)` when signers present, and
`cplq_unstaking_blocks ≥ voting_period_blocks` (so a voter cannot
stake → vote → unstake → unwind before tally).

`mulDiv` uses `math/big` for overflow safety — mirrors `lib.SafeMulDiv` in
Canopy core. Always use it for `(a*b)/c` over uint64 amounts.

## Whitepaper §7 reconciliation

Under Whitepaper v1.2 §3.3, Canopy does **not** apply a protocol-level DAO
tax on top of rewards before distribution. canoliq's pool sees the full
committee share `R` directly. Any test that pins the whitepaper number
(e.g., `TestWhitepaperSection7Reconciliation`) seeds the pool with `R`.
Effective user yield per Tokenomics v1.2 §4.1 is `0.88 × R` (modulo
truncation), since the plugin applies its 12% fee on `R` with no upstream
cut to back out.

For `R=950` / 12% fee / 40-30-15-15 split with truncation **and v1.2
defaults including `insurance_bps=500`** (5% of treasury slice per
Tokenomics §8), the reference output is: yield=881, treasury=34,
insurance=1, validators=17, buyback=17, sum=950. Conservation includes
the insurance line — any new test that asserts the equation
`yield + treasury + insurance + validators + buyback == R` will fail
if you forget it.

## Genesis

`Canoliq.Genesis` is **idempotent** — re-runs short-circuit on
`globals.GenesisComplete`. The genesis source order is
`req.GenesisJson` → `Config.GenesisPath` → error. Tests inject via
`PluginGenesisRequest.GenesisJson`; production injects via the configured
path.

**The Canopy FSM does not currently dispatch `PluginGenesisRequest` for
the canoliq plugin** because chain genesis.json carries no plugin section
for plugin id 2. Without a trigger, plugin Genesis would never run and
`ProcessRewards` is a permanent no-op (it bails on
`!globals.GenesisComplete`). To fix this, `BeginBlock` calls
`bootstrapGenesisIfNeeded` on every block: if `GenesisComplete=false` and
`Config.GenesisPath` is set, it runs `runGenesis(nil)` which loads the
configured file. This means the plugin self-bootstraps on its first
BeginBlock after deploy. Tests with empty `Config.GenesisPath` skip the
branch cleanly. See `TestBeginBlockSelfBootstrapsGenesis`.

Bucket bps must sum to 10_000, **and** recipients within each bucket must
also sum to 10_000. Both are validated by `validateGenesis`. Liquid tranches
(`cliffMonths==0 && vestMonths==0`) credit `KeyForCPLQBalance` directly;
otherwise a `VestingSchedule` is written and an entry is appended to
`VestingIndex`.

`CPLQTotalSupply = 100_000_000 * 1_000_000` (uCPLQ, 6-decimal parity with
uCNPY). Don't change this unit without auditing every fee/transfer path.

## Governance lifecycle (Phase 2)

`governance.go` implements proposal create / vote / tally / execute. A few
non-obvious things:

- **Voting weight is snapshotted by stake-time, not by proposal-time read.**
  The `CPLQStake` record carries `staked_at_height`. At vote delivery the
  handler rejects votes whose `staked_at_height > proposal.creation_height`.
  This defeats flash-stake without storing per-(proposal, voter) snapshot
  balances. Side-effect: a staker who *increases* stake after a proposal
  opens will have the new (post-creation) `staked_at_height` and lose
  voting eligibility on that proposal. That is intentional — re-staking
  cleanly resets eligibility.
- **Total staked snapshot is taken at create.** `Proposal.snapshot_total_staked`
  is `globals.total_staked_cplq` at the proposal's creation height. Quorum
  divides against that, not against the value at tally time. Don't change
  the snapshot semantics without auditing `proposalPasses`.
- **Tally cleanup deletes the proposal but not its votes.** Vote records are
  per-(proposal_id, voter); without a per-proposal voter index they cannot
  be enumerated. Stale vote keys are inert (no proposal to look them up by)
  and cheap. If you need to GC them, add a per-proposal voter index *first*.
- **Param-change payloads are full-set replacement.** `ProposalParamChange`
  carries a complete `CanoliqParams`, not a delta. This keeps `ValidateParams`
  invariants (split bps total, threshold ≤ signers, …) checkable in one shot.
- **`MessageCPLQProposalCreate.Payload` is a `google.protobuf.Any`.** Use
  `anypb.New(typed)` to build it; `unwrapPayload(any)` resolves it back via
  `contract.FromAny`. Only `ProposalParamChange`, `ProposalBuyback`, and
  `ProposalTreasurySpend` are accepted — any other type is rejected at
  create *and* at tally.

## Buyback (Phase 2)

`buyback.go::DeliverMessageBuybackExecute` consumes a passed
`BuybackOrder` keyed by proposal id. The order is **self-contained** — it
embeds the original `ProposalBuyback` payload (`cnpy_amount`,
`price_micro_cnpy_per_cplq`, `mode`) — because `dispatchPassed` deletes the
source `Proposal` record at tally cleanup. Don't rely on the proposal still
being readable when execute runs.

Modes:

- `BUYBACK_BURN` decrements `globals.cplq_total_supply` and
  `globals.cplq_circulating_supply` by `cplq_acquired`. CNPY moves from
  `buyback/pool` to `treasury/canoliq`; CPLQ disappears (no recipient).
- `BUYBACK_DISTRIBUTE_STAKERS` iterates `CPLQStakeIndex.Addresses`, computes
  `mulDiv(cplqAcquired, stake[s], totalStake)` per staker, credits liquid
  CPLQ balances, and adds the rounding remainder to the largest-stake
  staker. Empty staker set → re-credit `treasury/cplq` (no-op buyback).

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
`insurance_bps=500` (5% of treasury slice ≈ 0.5% of fee) — matches Tokenomics
§8 / WP §9.2 ("insurance fund of 5% of DAO treasury"). The insurance pool is a passive accumulator in
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

`MessageCPLQClaimVested` reads the `VestingIndex` first, then issues a second
batch read for every schedule listed. Two FSM round-trips per claim is
intentional — needed because we cannot range-scan.

## Query layer (Phase 3 §1)

`rpc.go` runs a small `net/http` mux **inside the plugin process** so
operators can read plugin-owned state without going through the FSM.
All routes are read-only.

### Why a snapshot, not freestanding reads

The Canopy FSM enforces that every plugin-initiated `StateRead` carries
a request ID from an in-flight FSM-originated lifecycle call
(`CheckTx`/`Deliver`/`Begin`/`End`). The FSM uses that ID to look up its
side of the request context. Outside that window the context does not
exist and the FSM rejects the read with `state_machine code 107` and
tears down the unix socket. **An earlier design that minted random
fsmIds for HTTP requests crashed the node every time** — the
`fakeStore` test path bypasses the protocol so the bug only appeared
under live docker integration. See
`memory/feedback_plugin_stateread_constraint.md`.

The current design builds a `Snapshot` (`snapshot.go`) inside
`EndBlock`, atomically swaps it into `Plugin.snapshot`
(`atomic.Pointer[Snapshot]`), and serves HTTP queries from that frozen
value. No plugin↔FSM round-trip per request, no concurrency on `pending`,
and no shared-state contention. Snapshots are stale by up to one block.

### What's snapshotted

Only state reachable from a singleton or an existing index:

- Singletons: `CanoliqGlobals`, `CanoliqParams`, committee pool,
  treasury CNPY/CPLQ, buyback pool, insurance pool.
- `ValidatorRegistry` → per-validator incentive accruals.
- `ProposalIndex` → active `Proposal` records.
- `SpendIndex` → pending `TreasurySpend` records.
- `CPLQStakeIndex` → active `CPLQStake` records.
- For each pending spend × each `MultisigSigners` entry → live
  `MultisigApproval` records.

### Lazy-fulfilled per-address routes (Phase 3 §1.1)

Per-address routes (`/v1/account/{addr}`, `/v1/vesting/{addr}`,
`/v1/redemption/{addr}/{id}`, `/v1/vote/{id}/{voter}`,
`/v1/buyback/{id}`) cannot be snapshotted because canoliq has no
global "all addresses ever seen" index. Instead, `lazy_query.go`
implements a queue: HTTP handlers build a `*lazyQuery`, push it onto
`Plugin.pendingQueries`, and block on the per-query result channel
(with the request context). `EndBlock` drains the queue *after*
`refreshSnapshot` (so `c.fsmId` is still a valid FSM context for
state reads), reading each address's records in turn and sending
results back.

Worst-case HTTP latency = one block (~6s on localnet). Client
context cancellation (`r.Context()`) aborts the wait promptly so
disconnected requests don't pin a goroutine. Queue capacity is
`lazyQueueCapacity = 256`; saturated → 503. Drain timeout =
`lazyQueryTimeout = 15s` (≈2.5× block time); exceeded → 504.

When adding a new lazy route:

1. Add a new `lazyKindX` constant.
2. Add a `readX` helper in `lazy_query.go` that does the StateRead
   work. Stay synchronous; the drain is single-threaded.
3. Wire it into `fulfillLazy`'s switch.
4. Add a route handler in `rpc.go` that builds the `*lazyQuery` and
   passes `r.Context()` to `enqueueLazy`.

Don't try to satisfy a lazy query from outside `EndBlock` — the FSM
will reject the StateRead with code 107 (see
`memory/feedback_plugin_stateread_constraint.md`).

### Adding a new route

1. If the entity has a singleton or index already, add it to
   `Snapshot` + `refreshSnapshot` and write a `Plugin.QueryX` accessor
   in `query.go`.
2. If not, add the index on the *write side* first (mirroring how
   `ProposalIndex` / `SpendIndex` / `CPLQStakeIndex` are maintained).
   Snapshot enumeration is bounded by the index size; never try to
   sweep state from a route handler.

JSON encoding piggybacks on `@gotags` on the proto types so the wire
shape matches `canoliqctl`. `google.protobuf.Any` fields (e.g.
`Proposal.Payload`) serialize as `{typeUrl, value}` — opaque to JSON
consumers but sufficient for reconciliation.

## Push alerts (T6)

`alerts.go` adds a *push* path on top of the *pull* RPC surface. Design points
a reviewer should hold onto:

- **Push vs pull.** The RPC layer already exposes everything for polling.
  Alerts exist only for the handful of events an operator must react to
  unattended (pool draining, stake concentration, TVL crash). Don't add an
  alert for something a dashboard poll already covers well.
- **Goroutine dispatcher.** `fireAlert` spawns a goroutine for the webhook
  POST (5s timeout) so a slow/dead receiver can never stall `EndBlock` — i.e.
  never stall consensus. Delivery is best-effort: failures log at WARN and are
  swallowed. Never make alert delivery block or error a lifecycle call.
- **Dedup in state, not memory.** The per-kind watermark
  (`KeyForAlertState(kind)` → `AlertState`) lives on-chain so debounce survives
  restarts and is deterministic across nodes. `applyAlert` fires only when
  `last_fired==0` or `height-last_fired >= minInterval`, and clears the
  watermark on resolution so the next occurrence pages immediately.
- **Evaluation is deterministic; delivery is not.** `evaluateAlerts` runs in
  `EndBlock` after `refreshSnapshot`, reading the snapshot + alert state — all
  deterministic, same on every node. Only the POST is non-deterministic, and it
  has no state effect. Keep it that way: never let webhook success/failure feed
  back into state.
- **Tumbling windows.** Drain/drop conditions re-anchor a baseline every
  `windowBlocks` rather than keeping a per-block ring buffer — cheaper, and good
  enough for an alert. `rollWindow` returns `true` on the anchor block so the
  caller skips comparing against a just-set baseline.
- **Test seam.** `Plugin.alertHook` (nil in production) receives alerts
  synchronously so condition/dedup tests are deterministic; the real HTTP path
  (`postAlert`, format adapters, 500-resilience) is tested separately against
  `httptest.Server`.
- **Deferred:** the stuck-redemption condition needs a global
  mature-unclaimed-redemption index (L3 only added per-address indexes). Blocked
  on that landing.

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
(1–14). Phase 1 occupies 100–116; Phase 2 extends through 117–135 (CPLQ
stake/unstake, governance, buyback, treasury, multisig, insurance). Always
use the constructor functions; never build `*PluginError` literals directly
so the module field stays consistent.

## Common bugs caught in review

- **Forgetting to pin the watermark** after a deposit when also injecting a
  reward in the same test → reward sweep treats the deposit fee as reward
  inflow and yields are off.
- **Confusing committee membership with stake ownership.** Being on committee
  `chainId` does not make a bond canoLiq's; `Validator.output` does. Crediting
  membership is the mainnet committee-29 bug. The inverse mistake is just as
  wrong: skipping unowned members in `distributeValidatorShare` would stop
  paying operators for work they actually do.
- **Seeding a reward test without declaring ownership in state.**
  `ProcessRewards` reads params via `LoadParams()`, so params built in a test
  variable and handed to a `Deliver*` call are invisible to it — the sweep sees
  the default empty set and attributes nothing. Use `seedStakeOutputParams`.
- **Setting `GenesisComplete=true` but skipping `SaveParams`** → handlers
  that call `LoadParams()` get `DefaultParams` instead of the genesis-time
  override.
- **Using `len(r.Entries) == 0` as "key absent"** is correct, but follow-up
  code that unmarshals the (empty) bytes into a proto produces a zero-value
  struct — make sure that is the intended semantic for that field. The cCNPY
  balance path deliberately uses `DecodeUint64` which returns 0 for nil/short.
- **Account proto vs scalar uint64**. CNPY (in `Account`) is a protobuf
  message; cCNPY/CPLQ balances are bare 8-byte big-endian uint64. Don't
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
- **Voting weight from the wrong source.** Always read `CPLQStake`, never
  `KeyForCPLQBalance` (liquid). Liquid CPLQ has zero governance weight by
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
