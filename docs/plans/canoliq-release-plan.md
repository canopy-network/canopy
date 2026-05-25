# canoLiq v1.1 Release Plan

This document supersedes `docs/plans/canoliq-implementation-plan.md` (written
against the v2 spec) and absorbs the v1.1 spec audit in
`docs/plans/canoliq-1-1-implementation_plan.md`. It reorganizes remaining work
by **release target** — localnet, testnet, mainnet — rather than by the
historical phase numbering, so it's clear at every point what blocks the next
milestone.

## Context

canoLiq is a liquid staking protocol implemented as a Canopy sub-chain Go
plugin (no Canopy core consensus changes). The plugin:

- mints cCNPY 1:1 against deposited CNPY, accruing staking rewards from
  Canopy's committee distribution path,
- applies a **12% protocol fee** with a **40 / 30 / 15 / 15** split
  (users / canoLiq treasury / validator incentives / buyback pool),
- mints a fixed **100 M CLIQ** supply at genesis with the seven-bucket
  distribution and per-bucket vesting,
- runs governance, buyback, treasury spend, and an HTTP query/alert
  surface inside the plugin process.

**Spec source of truth:** `canoLiq_Whitepaper_v1.1.pdf` +
`canoLiq_Tokenomics_v1.1.pdf` (May 2025). The earlier v2 draft is referenced
only when describing what changed.

> **Known doc conflict — validator cliff (verified 2026-05-21):** Whitepaper
> §5.2 mis-states the Validators & Infrastructure cliff as **6 months**.
> Tokenomics §2.1 governs and is explicit: **3-year linear with a 12-month
> cliff**, and it directly rejects the shorter figure ("A 6-month cliff is
> insufficient given the operational commitment required"); §6's summary
> table likewise lists 12 months. The genesis files use the correct 12-month
> cliff (`cliffMonths: 12, vestMonths: 24`). For vesting numbers, Tokenomics
> is authoritative over the whitepaper's §5.2 summary table.

**Vesting `Duration` interpretation (resolved with user):** Duration columns
in v1.1 mean **total span including cliff**. So `cliffMonths + vestMonths`
must equal the documented duration.

---

## What's already landed

The Phase 1 MVP, Phase 1.5 live-socket integration, Phase 2 governance /
buyback / treasury / insurance, and Phase 3 §1 (HTTP query layer) + §1.1
(per-address lazy routes) are all complete and verified. The full landed
inventory lives in the prior plan; the relevant pre-existing facts for the
remaining work are:

- **Plugin handshake & live reward sweep** — verified in
  `.docker/compose.yaml`. Reward inflow reconciles exactly across
  pool / treasury / insurance / validator-incentives / buyback per the
  v1.1 fee math (WP §3.3–§4.3, Tokenomics §3). Re-verified live 2026-05-21
  at cumulative R = 216 M uCNPY with the corrected 5 % insurance skim
  (see Localnet L1 verification).
- **Tx surface** covers Phase 1 (deposit / redeem / claim / cliq-transfer /
  cliq-claim-vested) and Phase 2 (vote / stake / unstake / claim /
  buyback-execute / spend-execute / multisig-approve / proposal-create with
  param-change / buyback / treasury-spend sub-commands).
- **Read surface** — `/v1/health`, `/v1/globals`, `/v1/params`, `/v1/pools`,
  `/v1/proposals` (+ `/v1/proposal/{id}`), `/v1/spends` (+
  `/v1/spend/{id}/approvals`), `/v1/validators`, `/v1/stakers`,
  `/v1/account/{addr}` (composite), `/v1/vesting/{addr}`,
  `/v1/redemption/{addr}/{id}`, `/v1/vote/{id}/{voter}`,
  `/v1/buyback/{id}`. Lazy-queue model: HTTP routes that need arbitrary
  state reads queue into `EndBlock` (where `c.fsmId` is valid for
  StateReads), so one-block worst-case latency.
- **ValidatorRegistry** — singleton, genesis-seedable, governance-mutable;
  drives per-validator pro-rata pay-out of the 15 % validator slice.

Two non-obvious constraints surfaced live and must not regress:

1. The Canopy FSM rejects plugin-initiated `StateRead` calls whose `fsmId`
   is not from an in-flight FSM lifecycle call (`code 107: plugin response
   id is invalid`). Any background / HTTP / timer goroutine that needs to
   read state must be served from `Plugin.snapshot` (refreshed in
   `EndBlock`) or queued onto the lazy-fulfill channel — never from a
   minted random fsmId.
2. The Canopy FSM only dispatches `PluginGenesisRequest` when chain
   `genesis.json` carries a plugin section for the plugin's id. The
   bundled docker compose chain genesis has no canoliq section, so the
   plugin must **self-bootstrap genesis from `BeginBlock`**
   (`bootstrapGenesisIfNeeded` in `canoliq.go`, reading
   `Config.GenesisPath`).

## Spec evolution v2 → v1.1 (what changed)

| Area | v2 (old) | v1.1 (new) |
|------|----------|------------|
| Canopy 5 % DAO tax | "Canopy takes 5 % on-chain prior to committee distribution" | **Removed.** WP §3.3 — "Canopy does NOT apply a protocol-level DAO tax on top of rewards before distribution." |
| Founders vesting | "3-yr vest, 6–12 mo cliff" | **4-yr linear, 12-mo cliff** |
| Validator vesting | "subject to lockups" | **3-yr linear, 12-mo cliff** (12-mo cliff explicitly motivated) |
| Strategic Partners | "subject to lockups" | **18-mo linear, 6-mo cliff** |
| Insurance Fund | 1–2 % of treasury | **5 % of DAO treasury, target 5 % of peak TVL within 12 mo** |
| Governance | multisig + timelock (generic) | **7-tier matrix** with distinct quorum / approval / timelock per action type |
| Vote-escrow | "stake/lock for boosts" (generic) | **5-tier explicit schedule** (1×–4× voting, base to +75 % reward boost) |
| TVL self-cap | not specified | **33 % of total Canopy network stake** |
| Fee rate bounds | "subject to governance" | **5 %–20 %** explicit bounds |
| Autonomy graduation | objective thresholds (vague) | **Concrete numbers**: > $50 M TVL, > 30 validators, > 15 % governance turnout, > 10 k tx/day, > 12 mo runway |
| Distribution (22/15/20/15/12/10/6) | same | same — no change |
| Fee split (40/30/15/15) | same | same — no change |
| Default fee (12 %) | same | same — no change |

Production reward / fee code is already correct under v1.1 (it operates on
inflows as-received with no `0.95X` factor). The remaining gaps are in
genesis numbers, doc/test narrative, governance richness, vote-escrow, caps,
and autonomy.

---

# Localnet release

Localnet release = "the docker-compose chain reconciles exactly against the
v1.1 spec." Everything here is small in code-cost; the point is to get the
on-chain numbers right and remove every trace of v2 from the docs/tests.

## L1. v1.1 numeric corrections (audit Wave 1)

Genesis JSON + one validator path. All P0 blocking spec violations. ~1 day.

- [x] **F1.** `plugin/go/canoliq/genesis.localnet.json:6-8` and
      `genesis.testnet.json:6-9` — validator vesting `cliffMonths: 6,
      vestMonths: 24` → `cliffMonths: 12, vestMonths: 24` (Tokenomics
      v1.1 §2.1: 3-yr linear, 12-mo cliff).
- [x] **F2.** `genesis.localnet.json:55-56` and `genesis.testnet.json:56-57`
      — founders `cliffMonths: 12, vestMonths: 24` → `cliffMonths: 12,
      vestMonths: 36` (Tokenomics v1.1 §2.5: 4-yr linear, 12-mo cliff).
- [x] **F3.** `genesis.localnet.json:67-68` and `genesis.testnet.json:68-69`
      — strategic partners `cliffMonths: 6, vestMonths: 18` → `cliffMonths:
      6, vestMonths: 12` (Tokenomics v1.1 §2.6 summary table: 18-mo total
      span; under "Duration includes cliff" reading the current code is
      wrong).
- [x] **F4.** `plugin/go/canoliq/config.go::ValidateParams` (~line 243)
      currently only rejects `FeeBps > 10_000`. Add `if p.FeeBps < 500 ||
      p.FeeBps > 2000 { return ErrInvalidParams() }`. Any `param-change`
      proposal violating this must fail validation, not pass through.
      (Tokenomics v1.1 §3.3 / WP §4.1: 5 %–20 %.)
- [x] **F5.** `config.go:219` `InsuranceBps: 1500 → 500` (matches the "5 %
      of DAO treasury inflow" reading of Tokenomics §8). Track the
      5 %-of-peak-TVL *cap* as a peak-TVL tracker in **T4** (testnet wave);
      for now this is the continuous-skim correction only.

### L1 verification
- [x] `cd plugin/go && go test ./canoliq/...` — vesting unit tests assert
      against configured `cliffMonths`/`vestMonths`, so F1–F3 should pass
      unchanged. F4/F5 may need one or two fixture updates. (Three reward
      tests updated to match the new 5 % insurance skim arithmetic;
      no other fixtures touched.)
- [x] Spin localnet via `.docker/compose.yaml`, let subsidy rewards flow,
      hit `/v1/pools` + `/v1/globals` and assert the split uses the
      corrected 5 % insurance bps. **Verified live (2026-05-21):** a
      cumulative R = 216 M-uCNPY inflow reconciles exactly —
      pool 200.448 M / treasury-net 7.3872 M / insurance 0.3888 M /
      buyback 3.888 M / validators 3.888 M (Σ = 216 M). Insurance is the
      v1.1 5 %-of-treasury skim (388 800), **not** the old 15 % (1 166 400).
      Param-change bound: the F4 5 %–20 % rejection is **not** enforced at
      `CheckTx` — `CheckMessageCLIQProposalCreate` is stateless and never
      unpacks the param payload. `ValidateParams` (rejecting FeeBps 2500)
      runs in `dispatchPassed` when a proposal *passes*, so an out-of-bound
      param-change is queued but cannot apply. Covered by the new
      `TestValidateParamsFeeBpsBounds` unit test rather than a 100 800-block
      live vote.

## L2. Doc / narrative cleanup (audit Wave 2)

No behaviour change; removes the v2 "0.95 X Canopy pre-cut" narrative that
will otherwise contradict every line of v1.1 doc anyone reads.

- [x] **F6.** Rewrite `plugin/go/canoliq/AGENTS.md:101-116` (the
      "Whitepaper §7 reconciliation" section) to describe the v1.1 model:
      canoLiq receives `R` directly from committee distribution, applies
      12 % fee on `R`, user yield = `0.88 × R`. Update test comments in
      `canoliq_test.go:258, 265, 288` and `rpc_test.go:133` to drop the
      `0.95` factor — **test inputs do not change** (the 1000 uCNPY input
      stays the same, but the narrative becomes "given R = 1000" rather
      than "given gross X = 1053 with 0.95X reaching the pool").
- [x] **F14.** Rename genesis bucket `"Plugin & Dev Grants"` → `"Developer
      Grants & Ecosystem"` in `genesis.localnet.json:78`,
      `genesis.testnet.json:79` (Tokenomics v1.1 §6).
- [x] **F15.** Rename `"Liquidity Incentives (Farming)"` → `"Liquidity
      Incentives"` in `genesis.localnet.json:17`, `genesis.testnet.json:18`
      (v1.1 drops the "(Farming)" suffix).

### L2 verification
- [x] `cd plugin/go && go test ./canoliq/...` — nothing should reference
      the old bucket names by string match. (README production-template
      block also updated to v1.1 vesting numbers + new bucket names.)

## L3. Per-address collection indexes (Phase 3 §1.1-bis carryover)

`/v1/account/{addr}` cannot list pending redemptions or unstakes today —
there is no per-address index that names those records. This blocks the
"stuck redemption" alert condition (T6 below) and the export round-trip
(M1).

### Proto + state
- [x] Added typed `RedemptionIndex` and `UnstakingIndex` proto messages
      (mirrors the established `VestingIndex` / `ProposalIndex` /
      `CLIQStakeIndex` pattern; named-type readability over wire-format
      reuse of `VestingIndex.ScheduleIds`). Two new key helpers:
      `KeyForRedemptionIndex(addr)` (domain byte 21),
      `KeyForUnstakingIndex(addr)` (domain byte 22), plus a
      `removeUint64` helper.

### Write-side maintenance
- [x] `DeliverMessageCanoliqRedeem` appends to `KeyForRedemptionIndex(addr)`.
- [x] `DeliverMessageCanoliqClaimRedemption` removes the matured id from
      the index alongside the existing record delete; deletes the index
      key entirely when the last id is claimed.
- [x] `DeliverMessageCLIQUnstake` appends to `KeyForUnstakingIndex(addr)`.
- [x] `DeliverMessageCLIQClaimUnstake` removes the matured id; deletes
      the index key when empty.

### Read-side
- [x] Extended `buildAccountView` to load both indexes, batch-read the
      records, and add `Redemptions []*contract.Redemption` and
      `Unstakes []*contract.UnstakingCLIQ` to `AccountView`.
- [ ] Optional dedicated routes: `/v1/account/{addr}/redemptions`,
      `/v1/account/{addr}/unstakes` (lazy queue, same pattern as vesting).
      Deferred — composite `/v1/account/{addr}` already exposes both
      lists; standalone routes are nice-to-have and can land alongside
      T6 stuck-redemption alerts if needed.

### L3 tests
- [x] Index append/remove invariants under all four flows
      (`TestRedemptionIndexAppendOnRedeem`, `TestRedemptionIndexRemoveOnClaim`,
      `TestRedemptionIndexOutOfOrderClaims`,
      `TestUnstakingIndexAppendOnUnstake`, `TestUnstakingIndexRemoveOnClaim`,
      `TestUnstakingIndexOutOfOrderClaims`). Index ends empty / deleted
      when no records remain.
- [x] `buildAccountView` reflects the new lists in correct order
      (`TestAccountViewIncludesRedemptionsAndUnstakes`).
- [x] Idempotency: `removeUint64` is a no-op when the id is absent
      (`TestRemoveUint64Idempotent`).

## Localnet exit criteria
- [x] L1 + L2 + L3 all landed and tested.
- [x] `.docker/compose.yaml` chain reconciles its reward inflow against
      v1.1 fee math. **Verified live 2026-05-21** (cumulative R = 216 M
      uCNPY; 5 % insurance skim confirmed; conservation exact).
- [x] `/v1/account/{addr}` returns redemption + unstake collections.
- [x] No production-path file references "0.95 X" or "DAO 5 % pre-cut" or
      "Liquidity Incentives (Farming)" or "Plugin & Dev Grants".
- [ ] Coordination message to Canopy Discord (from old plan): chainId
      selection, fee + split, supply + distribution, validator opt-in,
      plugin runtime contract (`CANOPY_PLUGIN_MODE=canoliq`,
      `/tmp/plugin/plugin.sock`).

---

# Testnet release

Testnet release = "the v1.1 spec is enforced end-to-end with safety rails
and an alert surface." Features here are net-new behaviour that earlier
audits and the old plan deferred. Multi-week.

## T1. Per-action governance matrix (audit F7 + F12 + F13)

Adds a 7-row per-action matrix from Tokenomics v1.1 §7 on top of today's
one-size-fits-all `quorum_bps / pass_threshold_bps / timelock_blocks`
(kept as the fallback — see Proto + state). F12 (validator ejection) and
F13 (emergency fast-track) fall out for free once T1 is in.

**Status: landed and tested (2026-05-22).** Full suite green; see the
checked items below for the as-built notes.

| Action | Quorum | Approval | Timelock |
|---|---|---|---|
| Fee rate adjustment | 5 % | 51 % | 48 h |
| Treasury spend (small) | 5 % | 51 % | 48 h |
| Treasury spend (large, > 1 M CLIQ) | 10 % | 67 % | 7 d |
| Emergency security action | 8 % | 67 % | 24 h fast-track |
| Validator ejection | 5 % | 51 % | 48 h |
| Protocol upgrade | 10 % | 67 % | 7 d |
| Autonomy graduation | 15 % | 75 % | 14 d |

### Proto + state
- [x] `proto/canoliq.proto` — added `ActionType` enum (all 7 values) and
      `GovernanceTier{action, quorum_bps, approval_bps, timelock_blocks,
      voting_period_blocks}`. Regenerated via `proto/_generate.sh`.
- [x] Added `ProposalValidatorEject{validator_address}`,
      `ProposalEmergency{description, param_change}`,
      `ProposalProtocolUpgrade{version, payload}`. (Resolved by `FromAny`
      via the global proto registry — no oneof; `Proposal.payload` stays a
      generic `Any`, matching the existing param-change / buyback / spend
      pattern.)
- [x] `CanoliqParams` — **decision (with user): additive + fallback rather
      than the literal "replace".** Added `repeated GovernanceTier
      governance = 24`; kept the scalar `quorum_bps / pass_threshold_bps /
      timelock_blocks / voting_period_blocks` as the fallback when a tier is
      unset or unmatched (e.g. buyback). Lower blast radius, backward-compatible
      with stored params, per-action enforcement fully delivered. `DefaultParams`
      seeds all seven from Tokenomics §7; `ValidateParams` checks every tier
      (known action, unique, bps ≤ 10000). `Proposal` also gained `action_type`
      + a snapshotted `tier`.

### Behaviour
- [x] `DeliverMessageCLIQProposalCreate` infers `ActionType` via
      `actionTypeForPayload` (treasury small/large split at the 1M-CLIQ
      boundary) and snapshots the resolved tier + voting period into the
      `Proposal` record.
- [x] `proposalPasses` reads quorum + approval from the proposal's recorded
      tier (scalar fallback for nil-tier / pre-T1 proposals).
- [x] `queueTreasurySpend` uses the proposal-recorded tier timelock (small
      48 h, large 7 d, emergency 0); legacy nil-tier path unchanged. Multisig
      gating stays independent (amount vs `treasury_threshold`).
- [x] **F12** — `dispatchPassed` for `ProposalValidatorEject` calls
      `ejectValidator`: removes the address from `ValidatorRegistry` and
      deletes its accrued `validator/incentives/{addr}`. Idempotent (no-op if
      absent) so a passed eject can never halt BeginBlock; future sweeps
      redistribute pro-rata over survivors.
- [x] **F13** — `ProposalEmergency` runs on the fast-track tier (24h vote,
      0 timelock); an optional `param_change` is validated + applied
      immediately on pass. `ProposalProtocolUpgrade` is recorded only (no
      on-chain dispatch).

### CLI
- [x] `canoliqctl proposal-create` gained `validator-eject` and `emergency`
      sub-commands; existing sub-commands are auto-tagged plugin-side via
      `actionTypeForPayload` (no CLI change needed for tagging).

### T1 tests
- [x] Tier quorum / approval / timelock independently enforced
      (`TestT1ProposalPassesUsesTier`, `TestT1TreasuryTimelockFromTier`,
      `TestT1DefaultTiersMatchSpec`, `TestT1ActionTypeInference`).
- [x] Validator ejection: pre-ejection split 9/9, post-ejection the survivor
      takes the full validator share, ejected gets nothing
      (`TestT1ValidatorEjectSkipsRewards`).
- [x] Emergency 24 h fast-track vs the same diff as a 7 d param-change
      (`TestT1CreateSnapshotsTierAndVotingPeriod`); emergency param diff
      applies with zero timelock (`TestT1EmergencyParamDiffApplied`).
- [x] Mixed-flight: a fee-change (51%) and a large treasury spend (67%) in
      the same window with an identical 60% yes ratio — the fee-change passes
      and applies, the large spend fails, both cleaned up
      (`TestT1MixedFlightIndependentTally`).

## T2. Vote-escrow lock multipliers (audit F8)

Tokenomics v1.1 §4.2:

| Lock | Voting × | Reward boost |
|---|---|---|
| None | 1× | base |
| 3 mo | 1.5× | +10 % |
| 6 mo | 2× | +25 % |
| 12 mo | 3× | +50 % |
| 24 mo | 4× | +75 % |

**Status: landed and tested (2026-05-22).** Full suite green.

### Proto + state
- [x] Extended `CLIQStake` with `lock_tier` (`LockTier` enum: LOCK_NONE /
      LOCK_3M / LOCK_6M / LOCK_12M / LOCK_24M) and `lock_end_height`.
- [x] `MessageCLIQStake` gained `lock_tier`; voting weight, unstake
      eligibility, and reward boost derive from it.
- [x] `tierMultipliers()` returns `(voteMultBps, boostBps)` (10000 = 1×);
      `lockTierDurationBlocks()` converts tiers to blocks
      (`blocksPerMonth = 432_000` at 6s); `validLockTier()` range-checks.

### Behaviour
- [x] `voteWeightFor(stake)` (new in `stake.go`) scales raw stake by the
      tier `voteMultBps`; the vote handler uses it.
- [x] `BUYBACK_DISTRIBUTE_STAKERS` (`distributeBuybackToStakers`) applies
      tier `boostBps` to each staker's effective weight; rounding remainder
      goes to the largest-stake LOCK_24M staker (fallback: largest overall).
      **Deviation from plan:** the boost is *not* applied to
      `distributeValidatorShare` — that path pays the 15 % slice to committee
      validators (`ValidatorRegistry`), not CLIQ stakers, so the §4.2
      vote-escrow boost doesn't belong there. Vote-escrow boost is a
      CLIQ-staker reward, hence the buyback-distribution path only.
- [x] Stake aggregation rule: locks only ever **strengthen** — a higher tier
      raises the record and pushes `lock_end_height` out; adding LOCK_NONE to
      a locked record leaves the lock intact (added tokens inherit it).
- [x] `DeliverMessageCLIQUnstake` rejects with `ErrStakeLocked` when
      `lock_tier != LOCK_NONE && current_height < lock_end_height`. (Enforced
      in Deliver, not the stateless Check.) Tier range checked in
      `CheckMessageCLIQStake` (`ErrInvalidLockTier`).

### CLI
- [x] `canoliqctl cliq-stake` gained `--lock {none,3m,6m,12m,24m}`
      (`parseLockFlag` / `parseLockTier`).

### T2 tests (`t2_voteescrow_test.go`)
- [x] Tier resolver values + durations; `validLockTier` rejects out-of-range
      (`TestT2TierMultipliers`).
- [x] Voting weight scales with tier; a LOCK_24M staker out-votes 3 ×
      LOCK_NONE stakers (`TestT2VoteWeightForScalesWithTier`,
      `TestT2VoteTallyAppliesMultiplier` — yes 4X vs no 3X end-to-end).
- [x] Reward boost: LOCK_NONE 100 + LOCK_12M 100 → 100 / 150 (1.5×), exact
      conservation to `cliqAcquired` (`TestT2RewardBoostInBuybackDistribution`).
      (Conservation target is `cliqAcquired`, not `split.Validators` — the
      boost lives in the buyback-distribution path per the deviation above.)
- [x] Pre-lock-end unstake rejected, post-lock-end accepted
      (`TestT2UnstakeLockGate`); lock-strengthen aggregation
      (`TestT2StakeLockOnlyStrengthens`).

## T3. TVL self-cap (audit F9)

WP v1.1 §9.4: "canoLiq will self-impose a TVL cap of 33 % of total Canopy
network stake pending ecosystem maturation and governance approval to lift
this cap."

The plugin cannot query Canopy's global stake directly without an FSM
helper. Two options; recommend the second for speed:

- (a) plumb a new `GetTotalNetworkStake` request from plugin → FSM
      (Canopy-side change, slow);
- (b) add `tvl_cap_uCnpy` to `CanoliqParams`, governance-tunable, and have
      the DAO update it as Canopy stake grows.

**Status: landed and tested (2026-05-25).** Took option (b) — a
governance-tunable `tvl_cap_ucnpy` param. Full suite green.

### Behaviour
- [x] `CanoliqParams.tvl_cap_ucnpy uint64` (proto field 25; 0 = uncapped;
      default 0). No `ValidateParams` rule needed — uint64 can't be negative.
- [x] `DeliverMessageCanoliqDeposit` rejects with `ErrTVLCapExceeded` when
      `cap > 0 && total_pooled_cnpy + amount > cap`. (Deliver-only; CheckTx is
      stateless.)
- [x] `/v1/health` (`HealthView` / `QueryHealth`) surfaces `tvlCapUcnpy` and
      `tvlUtilizationBps` (= pooled / cap in bps; 0 when uncapped).

### T3 tests (`t3_tvlcap_test.go`)
- [x] Deposit at exactly cap accepted, one uCNPY above rejected with state
      unchanged (`TestT3DepositCapBoundary`); lifting the cap re-enables
      (`TestT3LiftCapReenablesDeposits`); uncapped allows any deposit
      (`TestT3UncappedAllowsLargeDeposit`); health surfaces cap + utilization
      (`TestT3HealthSurfacesCapAndUtilization`).

## T4. Insurance fund peak-TVL tracking (audit F10)

Continuous skim is in (post-F5). The v1.1 spec also requires a target of
**5 % of peak TVL** within 12 months of mainnet. Implement as a periodic
gate: once reserve ≥ 5 % of peak TVL, the skim turns off until peak TVL
grows past the next threshold.

**Status: landed and tested (2026-05-25).** Full suite green.

### Proto + state
- [x] Added `CanoliqGlobals.peak_tvl_ucnpy` (field 14) and
      `CanoliqParams.insurance_target_bps` (field 26; `uint64` not `uint32` —
      matches the other bps fields). `DefaultParams` seeds
      `insurance_target_bps = 500` (5 %).

### Behaviour
- [x] Peak advanced inside `ProcessRewards` (the EndBlock reward hook):
      `peak_tvl = max(peak_tvl, total_pooled_cnpy)` on both the fresh-delta
      path (post-accrual) and the no-delta path — the latter also seeds peak
      from the current pool on a pre-T4 node.
- [x] `ProcessRewards`: when `insurance_target_bps > 0` and
      `insurance_pool >= mulDiv(peak_tvl, insurance_target_bps, 10_000)`, the
      skim is set to 0 so the full treasury slice (incl. the would-be
      insurance amount) stays in `treasury/canoliq` — conservation holds.
      `insurance_target_bps = 0` disables the gate (skim always on).
- [x] `/v1/pools` surfaces `peakTvlUcnpy`, `insuranceTargetUcnpy`,
      `insuranceFundedBps` (= `insurance_pool / target` in bps).

### T4 tests (`t4_insurance_test.go`)
- [x] Skim active below target (`TestT4SkimActiveBelowTarget`); off at/above
      target (`TestT4SkimOffAtTargetConserves`); resumes after peak grows
      (`TestT4SkimResumesAfterPeakGrows`).
- [x] Conservation: skim-off redirects the amount into the treasury, total
      equals the reward delta (`TestT4SkimOffAtTargetConserves`).
- [x] Migration: pre-T4 node (`peak_tvl_ucnpy = 0`) initializes peak from the
      current pool on the first sweep, even with no fresh delta
      (`TestT4PeakInitializesFromPoolOnMigration`).

## T5. Autonomy graduation tracking (audit F11)

T1 landed the `ACTION_AUTONOMY_GRADUATE` **ActionType + governance tier**
(15 % / 75 % / 14 d, seeded in `defaultGovernanceTiers`). The graduation
**payload** (`ProposalAutonomyGraduate`), its create path, and dispatch are
still T5/M3 work — `actionTypeForPayload` has no case for it yet, so a
graduation proposal cannot be created until then. T5 adds the threshold
tracking and surface so the DAO knows when to vote.

Whitepaper §10 thresholds:

| Metric | Threshold |
|---|---|
| TVL | > $50 M (uCNPY equivalent at oracle price; for now, a flat uCNPY threshold) |
| Active validators | > 30 |
| Governance turnout | > 15 % |
| Daily transactions | > 10 k |
| Runway (treasury / monthly burn) | > 12 mo |

### Proto + state
- [ ] `CanoliqParams` gains `graduation_min_tvl_ucnpy`,
      `graduation_min_validators`, `graduation_min_turnout_bps`,
      `graduation_min_daily_tx`, `graduation_min_runway_months`. All
      governance-tunable; defaults from §10.
- [ ] `CanoliqGlobals` gains `passed_proposal_count` (incremented in
      `dispatchPassed`), `daily_tx_count_window` (rolling counter), and
      `last_window_close_height` for turnout / tx-volume measurement.
- [ ] `GraduationStatus` on `Snapshot`.

### Behaviour
- [ ] `BeginBlock` advances the rolling window; each handler increments
      `daily_tx_count_window`.
- [ ] `dispatchPassed` advances `passed_proposal_count`.
- [ ] New route `GET /v1/graduation` returns each metric, its threshold,
      the ratio, and a composite `eligible bool`.

### T5 tests
- [ ] Each metric crosses threshold independently; `eligible` flips only
      when all five are met.
- [ ] Passed-proposal counter advances on pass; failed proposals don't
      increment.
- [ ] Rolling window resets exactly at boundary blocks.

## T6. Alerting hooks (old plan Phase 3 §2)

WP §11 "monitoring dashboards & alerts for validator behaviour, committee
health, and TVL movement." Push surface for unattended monitoring; pull
surface (RPC) is already in.

### Webhook delivery (`plugin/go/canoliq/alerts.go`)
- [ ] `AlertConfig` struct — `WebhookURL`, `AuthHeader` (optional),
      `Enabled`, per-kind `MinIntervalBlocks` debounce, payload `Format`
      (json | slack | discord), 5 s POST timeout.
- [ ] `Config.Alerts *AlertConfig` plumbed via JSON + `CANOLIQ_ALERT_URL`
      env override; empty disables (mirrors `RpcAddress`).
- [ ] Dispatcher fans out POSTs on a goroutine so `EndBlock` never blocks
      on network IO; failures logged at WARN.
- [ ] Deduplication: per-kind last-fired-height entry under
      `KeyForAlertState(kind)`; skip when `current_height - last_fired <
      MinIntervalBlocks`. Resolution event clears the watermark.

### Conditions
- [ ] **Buyback drain rate.** Rolling 100-block window; fire when window
      drain > `drain_alert_bps` (default 50 %).
- [ ] **Stuck redemption queue.** Mature unclaimed redemption count >
      `stuck_redemption_threshold`. **Depends on L3 indexes** + a global
      `StuckRedemptionIndex` of unclaimed mature ids — block this condition
      on those landing.
- [ ] **Validator-incentive starvation.** `max_validator_stake /
      total_committee_stake > concentration_alert_bps` (default 66 %).
- [ ] **TVL drop.** Rolling-window drop > N % in M blocks.

### Payload + tests + docs
- [ ] Envelope: `{kind, height, severity (warn|crit), message, details:
      {schemaVersion, ...}}`. Slack / Discord adapters convert.
- [ ] `httptest.Server` mock receiver; each kind fires once with expected
      payload under a seeded condition. Debounce: same condition seeded
      across consecutive `EndBlock`s fires once. Resilience: webhook 500
      doesn't stall `EndBlock`. Threshold edges: at-threshold doesn't fire,
      just-above fires.
- [ ] `README.md` payload schema + config knobs + webhook setup.
      `AGENTS.md` design rationale (push vs pull, dedup-in-state, goroutine
      dispatcher).

## Testnet exit criteria
- [ ] T1 + T2 + T3 + T4 + T5 + T6 landed and tested.
- [ ] Public testnet docker compose runs through:
      - tiered proposal round-trips (fee change, validator ejection,
        emergency, large treasury spend),
      - tier-boosted staker reward distribution,
      - TVL cap rejection at the configured ceiling,
      - insurance skim auto-off at target, auto-on as peak TVL grows,
      - `/v1/graduation` populated,
      - alert webhook receives buyback-drain + concentration warn events.
- [ ] Security review of governance flow (focus on T1 multi-tier dispatch +
      T2 lock-tier weight) before testnet bring-up.

---

# Mainnet release

Mainnet release = "an external network can rely on this." The bar is
operational: graduation tooling, on-chain DEX route for buyback so price
isn't proposal-set, audit + bug bounty sign-off, runbooks.

## M1. State export tooling (old plan Phase 3 §3b)

- [ ] Define `GenesisExport` proto extending `GenesisFile` with live state
      slots: per-address cCNPY / liquid CLIQ balances, `CLIQStake` records
      (with `staked_at_height` and tier), `UnstakingCLIQ` records,
      `Redemption` records, `VestingSchedule.ClaimedAmount`, treasury /
      buyback / insurance scalars, validator registry, active proposals /
      spends / votes, full `CanoliqParams` (incl. T5 graduation knobs, T3
      cap, T4 target), `globals.PassedProposalCount`, `peak_tvl_ucnpy`.
      Versioned via `export_schema_version`.
- [ ] Walk all canoliq-prefixed keys via the existing indexes + L3's
      per-address indexes (so this hard-depends on L3 landing). Encode
      addresses as hex.
- [ ] Determinism: same height + state → identical bytes. Sort collections
      by stable keys.
- [ ] CLI `canoliqctl export-genesis --rpc <url> --output
      genesis-export.json` reads via the RPC surface — snapshot for
      singletons + indexes, lazy queue for per-address data.
- [ ] Tests: round-trip — seed a small chain via `fakeStore`, export,
      re-import into a fresh `fakeStore`, assert state-tree equivalence
      (every canoliq-prefixed key matches).

## M2. State import (old plan Phase 3 §3c)

- [ ] Extend `runGenesis` to detect `GenesisExport` payloads (via
      `export_schema_version` presence) and seed *all* live state, not just
      the bucket distribution. Treat bucket math as already applied — the
      export captures post-distribution state.
- [ ] Validation: `cliq_total_supply` preserved (liquid + stake + unstaking
      + remaining-vesting = cap); treasury balances non-negative;
      per-validator stake totals match registry; `CLIQStakeIndex` matches
      the set of stake records; `peak_tvl_ucnpy >= total_pooled_cnpy`.
      Refuse to import if any check fails.
- [ ] Live test: export from a non-trivial localnet → restart fresh docker
      compose with the exported genesis → query the new node and assert
      identical `/v1/...` responses (modulo height, which restarts at 1).

## M3. Graduation coordination flow (old plan Phase 3 §3d)

- [ ] `ACTION_AUTONOMY_GRADUATE` dispatch (when passed): marks
      `globals.GraduationLockedHeight = h + grace_blocks` and stops
      accepting new deposits. Redemptions and vesting claims continue
      through the grace window.
- [ ] After `GraduationLockedHeight`, deposits return `ErrGraduationLocked`.
      Redemptions force-mature so users can exit before export.
- [ ] Cross-chain coordination doc (separate from the whitepaper):
      pre-graduation Canopy DAO proposal to deprecate the canoLiq committee;
      post-export, publish export hash on Canopy and the new L1;
      cCNPY/CLIQ holders' continuity guarantee (same addresses, same
      balances, same vesting schedules).
- [ ] Tests: pre-graduation → all flows work; post-lock pre-grace →
      deposits rejected, redemptions accepted; post-grace → export captures
      a fully quiesced state.

## M4. Real on-chain DEX buyback route

Phase 2's buyback executes against a proposal-set price (internal accounting
swap). WP §6 explicitly allows that, but a market-priced route is preferable
for mainnet so DAO mispricing is impossible.

- [ ] Define a `BUYBACK_DEX` mode that routes through Canopy's DEX / swap
      surface (`fsm/dex.go` or `fsm/swap.go` per pre-existing design note).
      Source the CLIQ via the on-chain DEX rather than `treasury/cliq`.
- [ ] Slippage protection: max-slippage-bps field on `ProposalBuyback`;
      abort and refund the pool if breached.
- [ ] Plumbing: this requires a plugin → FSM tx-side helper (the plugin
      currently only reads FSM state, not writes via FSM tx). The cleanest
      path is to have the executor submit a `MessageSwap`-equivalent through
      the same surface used by `MessageSubsidy` — confirm with Canopy core
      before sizing.

## M5. Operational sign-off

- [ ] Security audit covering the full v1.1 governance / staking / buyback
      / treasury surface (T1 + T2 + M4 are highest risk).
- [ ] Bug bounty program (WP §9.1). Operational only — link from
      `README.md`.
- [ ] Migration runbook in `docs/` — step-by-step for the operations team:
      pause deposits, wait grace, run export, verify hash, hand off to
      new-L1 genesis, deprecate committee.
- [ ] Live mainnet bring-up plan: validator opt-in list, initial
      `multisig_signers`, initial `MessageSubsidy` size, initial
      `tvl_cap_ucnpy`.

## Mainnet exit criteria
- [ ] M1 + M2 + M3 + M4 + M5 landed.
- [ ] Round-trip export/import preserves every canoliq-prefixed key.
- [ ] Graduation lock blocks deposits, allows redemptions, force-matures
      pending redemptions at end of grace.
- [ ] DEX-mode buyback executes against a live Canopy DEX with slippage
      protection.
- [ ] External audit report on file; bounty program live.

---

# Critical files for any of the above

| Concern | File |
|---|---|
| Genesis numeric values | `plugin/go/canoliq/genesis.localnet.json`, `genesis.testnet.json` |
| Param defaults & validation | `plugin/go/canoliq/config.go` (`DefaultParams`, `ValidateParams`) |
| Reward math (already correct under v1.1) | `plugin/go/canoliq/reward.go`, `fee.go` |
| Doc / test narrative | `plugin/go/canoliq/AGENTS.md`, `canoliq_test.go`, `rpc_test.go` |
| Governance proposal handling | `plugin/go/canoliq/governance.go`, `treasury.go`, `proto/canoliq.proto` |
| Staking & voting weight (T2 multipliers land here) | `plugin/go/canoliq/stake.go`, `governance.go::voteWeightFor` |
| Deposit / redeem (T3 TVL cap lands here) | `plugin/go/canoliq/canoliq.go`, `deposit.go`, `redeem.go` |
| Snapshot / RPC | `plugin/go/canoliq/snapshot.go`, `query.go`, `rpc.go` |
| Alerts (T6) | `plugin/go/canoliq/alerts.go` (new) |
| Export / import (M1 / M2) | `plugin/go/canoliq/export.go` (new), `canoliq.go::runGenesis` |
| CLI surface | `plugin/go/canoliqctl/proposal_create.go` and siblings |

# Open items / not covered

- **Snapshot-based airdrop eligibility** (Tokenomics §2.2) — outside the
  contract surface; off-chain process.
- **Quadratic / conviction voting** (Tokenomics §8 risk mitigation) — spec
  notes "may be explored post-launch"; not actionable now.
- **Canopy DAO subsidy proposal submission** (Tokenomics §5.3) — off-chain
  governance work for the Canopy DAO, not canoLiq code.
- **Real-USD TVL oracle** for T5 graduation threshold — currently a flat
  uCNPY threshold; price-oracle plumbing is its own track.
