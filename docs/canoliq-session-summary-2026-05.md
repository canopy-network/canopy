# canoLiq — Session Summary (May 2026)

This document summarizes the work completed in one working session on the
canoLiq liquid-staking plugin. The session took the project from **localnet
verification** through the **entire testnet feature track (T1–T6)**, all on the
`canoliq` branch.

Everything below is on `canoliq`; the plugin module test suite
(`plugin/go/...`) is green throughout. Commits span `8067fdc2..409f0184`.

---

## 1. Localnet reconciliation (live verification)

Brought up the bundled two-node docker-compose chain and reconciled the reward
fee math against the v1.1 spec **on a live chain**:

- Cumulative reward inflow **R = 216,000,000 uCNPY** reconciled exactly across
  pool / treasury / insurance / buyback / validators (Σ = R).
- Confirmed the **5% insurance skim** (post-F5), not the old 15%.
- Confirmed the effective user yield of **92.8% of R** (88% + 4.8%), matching
  Whitepaper §4.3 / Tokenomics §3.1 to the unit.

Findings & fixes:
- The release plan's "param-change rejected at CheckTx" criterion was
  **mis-specified**: `CheckMessageCLIQProposalCreate` is stateless and never
  unpacks the param payload. The F4 fee bound (5%–20%) is enforced at
  `dispatchPassed` → `ValidateParams` when a proposal *passes*. Added a unit
  test (`TestValidateParamsFeeBpsBounds`) and corrected the plan.
- Fixed a stale comment claiming `insurance_bps=1500` (now 500 post-F5).

## 2. Spec verification (Whitepaper + Tokenomics)

Cross-checked the implementation against both `canoLiq_Whitepaper_v1.1.pdf` and
`canoLiq_Tokenomics_v1.1.pdf`. The reconciliation work matched both documents.

One genuine conflict surfaced and was documented:
- **Validator vesting cliff** — Whitepaper §5.2 says *6-month* cliff; Tokenomics
  §2.1/§6 say *12-month* and explicitly reject 6 months ("A 6-month cliff is
  insufficient…"). The genesis files use the correct 12-month cliff. Tokenomics
  governs for vesting numbers.
- Captured in `docs/canoliq-whitepaper-tokenomics-discrepancies.md` and a note
  in the release plan's "Spec source of truth" section.

---

## 3. Testnet feature track (T1–T6)

All six testnet features from `docs/plans/canoliq-release-plan.md` landed with
tests and plan updates.

### T1 — Per-action governance matrix
*Commit `8f3eb20e`.*

Replaced the one-size-fits-all governance knobs with the 7-tier matrix from
Tokenomics §7 (per-action quorum / approval / timelock / voting period).

- **Proto:** `ActionType` enum, `GovernanceTier`, three new proposal payloads
  (`ProposalValidatorEject`, `ProposalEmergency`, `ProposalProtocolUpgrade`);
  `Proposal` gained `action_type` + a snapshotted `tier`.
- **Decision (with user): additive + fallback** rather than the plan's literal
  "replace" — kept the scalar knobs as the fallback for unmatched actions /
  pre-T1 proposals. Lower blast radius, backward-compatible.
- **Behaviour:** proposal creation infers the action and snapshots the tier;
  tally reads tier quorum/approval; treasury timelock comes from the tier.
  **F12** validator-eject dispatch (idempotent registry removal + incentive
  clearing); **F13** emergency fast-track with optional immediate param diff.
- **CLI:** `proposal-create validator-eject` and `emergency` subcommands.
- **Tests:** `t1_governance_test.go` (8) — tier resolution, tier-driven tally +
  timelock, validator-eject reward skip, emergency fast-track, mixed-flight.

### T2 — Vote-escrow lock multipliers
*Commit `a9fe6aad`.*

Longer CLIQ locks grant higher voting weight + reward boost (Tokenomics §4.2:
1×–4× voting, +0/10/25/50/75% boost).

- **Proto:** `LockTier` enum; `CLIQStake.lock_tier` + `lock_end_height`;
  `MessageCLIQStake.lock_tier`.
- **Behaviour:** vote weight = raw stake × tier multiplier; buyback
  `DISTRIBUTE_STAKERS` applies the boost (remainder to largest LOCK_24M staker);
  locks only ever strengthen on re-stake; unstake gated until `lock_end_height`.
- **Deviation (documented):** the reward boost applies only to the staker
  buyback-distribution path, **not** `distributeValidatorShare` — that path pays
  the 15% slice to committee validators, not CLIQ stakers.
- **CLI:** `cliq-stake --lock {none,3m,6m,12m,24m}`.
- **Tests:** `t2_voteescrow_test.go` (6).

### T3 — TVL self-cap
*Commit `648e7a38`.*

WP §9.4 self-imposed TVL ceiling (33% of network stake), as a governance-tunable
param (plan option b).

- **Proto:** `CanoliqParams.tvl_cap_ucnpy` (0 = uncapped).
- **Behaviour:** deposits rejected with `ErrTVLCapExceeded` above the cap;
  `/v1/health` surfaces `tvlCapUcnpy` + `tvlUtilizationBps`.
- **Tests:** `t3_tvlcap_test.go` (4).

### T4 — Insurance fund peak-TVL tracking
*Commit `cfeb00fc`.*

WP §9.2 reserve target of 5% of peak TVL — the per-block insurance skim turns
off at target and back on as peak TVL grows.

- **Proto:** `CanoliqGlobals.peak_tvl_ucnpy`; `CanoliqParams.insurance_target_bps`
  (default 500).
- **Behaviour:** `ProcessRewards` advances the peak high-water mark (seeds it on
  a pre-T4 node) and gates the skim against the target, rerouting the would-be
  insurance amount to the treasury (conservation holds); `/v1/pools` surfaces
  `peakTvlUcnpy`, `insuranceTargetUcnpy`, `insuranceFundedBps`.
- **Tests:** `t4_insurance_test.go` (4) — skim on/off/resume, conservation,
  migration init.

### T5 — Autonomy graduation tracking
*Commit `a268efac`.*

Surfaces the five WP §10 graduation thresholds so the DAO knows when to vote.

- **Proto:** 5 `graduation_min_*` params (§10 defaults) + 7 globals counters
  (passed-proposal count, daily-tx window, turnout running average, treasury
  burn).
- **Behaviour:** `DeliverTx` wrapper counts successful txs; `BeginBlock` rolls
  the daily-tx window; `processProposals` records passed count + per-proposal
  turnout; `applySpend` tracks CNPY burn for runway.
- **Surface:** `GET /v1/graduation` — each metric (value/threshold/met) +
  composite `eligible`.
- **Tests:** `t5_graduation_test.go` (9).

### T6 — Push-alert webhooks
*Commit `409f0184`.*

WP §11 unattended monitoring — push alerts to a webhook (Slack/Discord/JSON)
when an on-chain condition trips.

- **Config/proto:** `AlertConfig` (URL, auth, format, debounce, window,
  thresholds); `Config.Alerts` via JSON + `CANOLIQ_ALERT_URL`; `AlertState`
  proto + `KeyForAlertState(kind)`.
- **Dispatcher:** goroutine POST (5s timeout) so `EndBlock` never blocks;
  json/slack/discord adapters; on-chain dedup watermark with auto-resolution.
- **Conditions:** buyback drain + TVL drop (tumbling window); validator
  concentration (instantaneous). **Stuck-redemption deferred** — needs a global
  mature-unclaimed-redemption index that doesn't exist yet.
- **Tests:** `t6_alerts_test.go` (8) — firing, debounce, resolution, threshold
  edges, `httptest.Server` delivery for all formats, auth, 500-resilience.
- **Docs:** README config/conditions/payload sections; AGENTS rationale.

---

## 4. Files touched (new)

| File | Purpose |
|---|---|
| `plugin/go/canoliq/graduation.go` | T5 counters + `/v1/graduation` query |
| `plugin/go/canoliq/alerts.go` | T6 webhook dispatcher + conditions |
| `plugin/go/canoliq/t1_governance_test.go` … `t6_alerts_test.go` | per-feature tests |
| `docs/canoliq-whitepaper-tokenomics-discrepancies.md` | spec conflict report |

Plus edits across `proto/canoliq.proto` (regenerated `contract/canoliq.pb.go`),
`config.go`, `governance.go`, `treasury.go`, `reward.go`, `stake.go`, `deliver.go`,
`query.go`, `rpc.go`, `canoliq.go`, the `canoliqctl` CLI, `README.md`, `AGENTS.md`,
and `docs/plans/canoliq-release-plan.md`.

## 5. Status & what's next

**Testnet feature track T1–T6: complete and tested.** Remaining testnet exit
criteria are operational, not code:

- Public testnet docker-compose run-through (tiered proposals, tier-boosted
  rewards, TVL-cap rejection, insurance auto-off, `/v1/graduation`, alert
  events).
- Security review of the governance flow (T1 multi-tier dispatch + T2 lock-tier
  weight) before bring-up.
- Localnet leftover: the Canopy Discord coordination message.

Then the **mainnet track (M1–M5)**: state export/import tooling, graduation
coordination flow, on-chain DEX buyback route, and operational sign-off.

### Known follow-ups noted during the session
- Stuck-redemption alert condition (blocked on a global stuck index).
- A pre-existing `go vet` copylocks warning at `phase2_test.go:208`
  (`newParams := *params`) — predates this work; a one-line `proto.Clone` fix.
- README "Insurance fund" section still narrates `insurance_bps=1500` (stale
  post-F5) — cosmetic doc drift, not code.
