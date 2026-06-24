# canoLiq — Testnet Deployment Readiness Plan

## Context

The canoLiq plugin's testnet feature track (T1–T6) is code-complete and the
plugin module test suite is green. However, the plugin is **not deployment-ready**:
`genesis.testnet.json` and `canoliq-config.testnet.json` still carry TODO
placeholder addresses, the new T1–T6 features have only in-process (`fakeStore`)
test coverage (never exercised on a live multi-node chain), and the plan-required
governance security review hasn't run.

This plan closes those gaps. The bulk of the remaining work is **not new code** —
it's collecting real values from the dev team, wiring them into the genesis/config
files, validating end-to-end on a private testnet image, and a security pass. The
existing README already documents the deployment *mechanics* (Phases 0–5 in
`plugin/go/canoliq/README.md` lines 206–519); this plan organizes the readiness
work and pins down exactly what data the devs must supply.

**Decisions (confirmed with user):**
- I wire the dev-provided data into the files once values are supplied; devs only
  provide values + review.
- Known code gaps (stuck-redemption alert, `phase2_test.go:208` vet warning, stale
  insurance README line) are **out-of-scope follow-ups**, not testnet blockers.
- I perform a governance security **self-review** pass *and* flag for an external
  reviewer.

---

## ✅ Testnet upload checklist (at a glance)

Single-source tracker for everything needed to launch on testnet. Detail and
rationale live in the §A–E and Workstream sections below; this is the summary.
Last re-run: 2026-06-24 (build green, testnet genesis safety-check clean).

**Part 1 — Files & data — DONE (verified)**
- [x] §A — 7 bucket recipient addresses wired (bps sum 10000, no placeholder)
- [x] §B — 5 multisig signers, `multisigThreshold: 3` (3-of-5), `treasuryThreshold: 50M uCPLQ`
- [x] §C — `validatorRegistry: []` (empty, single-aggregator fallback for first boot)
- [x] §D — `chainId: 42`, `redemptionUnstakingBlocks: 30240`
- [x] §D — chainId 42 reservation confirmed with Canopy team
- [x] `.docker/compose.testnet.yaml` created and `docker compose config`-valid
- [x] Economic params verified spec-faithful vs v1.2 (fee 12% / 5–20%, 40/30/15/15,
      buyback, lock multipliers, all 7 governance tiers, insurance, graduation, vesting)
- [x] `go test ./canoliq/... ./canoliqctl/...` green; testnet genesis safety-check clean

**Part 2 — External coordination — BLOCKING (only Canopy/operators can do) 🔴**
- [ ] Fund committee pool 42 so `ProcessRewards` is not a no-op — simplest testnet
      path: self-funded `MessageSubsidy` (`canopy admin tx-subsidy <sender> <amount> 42 <opcode>`, no DAO vote). See §E.
- [ ] Each committee operator runs `MessageEditStake` adding chainId `42` to their
      `Validator.Committees[]` (no membership → no consensus, no rewards). See §E.
- [ ] *(Optional, T6)* Alert webhook URL → `CANOLIQ_ALERT_URL`. See §E.

**Part 3 — Recommended before a meaningful testnet (decisions) 🟡**
- [ ] Populate `validatorRegistry[]` with real **operator** validator addresses (the
      same ones doing `MessageEditStake` — **not** the §B signers). Needed for WS3 T1
      validator-eject coverage + per-validator reward credit. See §C.
- [ ] TVL-cap decision — genesis ships uncapped (`TvlCapUcnpy: 0`); spec §9.4 wants a
      33%-of-network-stake cap. Set at genesis or via param-change (T3).

**Part 4 — Execution / verification (run on the image) ⚙️**
- [ ] WS2 pre-flight: safety banner + self-bootstrap → bucket reconciliation (exactly
      100M CPLQ) → deposit→redeem→claim smoke → multisig rehearsal
- [ ] WS3 live T1–T6 run-through on the multi-node image; capture run-through log
- [ ] WS4 governance security self-review (`governance.go` / `treasury.go` / `alerts.go`)

**Part 5 — Cutover (point of no return) 🚀**
- [ ] WS5: hash-anchor final genesis + config
- [ ] Distribute image to validators (`CANOPY_PLUGIN_MODE=canoliq` + testnet `CANOLIQ_CONFIG`)
- [ ] Verify on real chain: `/v1/health.genesisComplete`, `/v1/validators` matches seeded
      set, `/v1/pools.committeePool` growing

> ⚠️ **First-block genesis is one-time and irreversible** — the 100M CPLQ mint to
> bucket addresses cannot be redone. Parts 2–4 must be signed off before Part 5.

**Critical path:** the only hard external blockers are the two Part 2 items. WS4 and
the TVL-cap decision can be done now (no live testnet needed); WS2/WS3 need a running image.

---

## 🔑 Data required from the dev team (blocking inputs)

Deployment cannot proceed until the team supplies all of the following. This is
the critical hand-off — everything else is execution.

**Status snapshot (2026-06-18):**

| Section | What | Status |
|---|---|---|
| A | Genesis bucket recipient addresses | ✅ supplied (2026-06-18) |
| B | Multisig signers | ✅ supplied (2026-06-18) — 3-of-5; `treasuryThreshold` lowered to 50M uCPLQ |
| C | Validator registry | ✅ shipped empty (2026-06-18) — single-aggregator fallback for first bring-up; populate before mainnet |
| D | Chain parameters (`chainId`, `redemptionUnstakingBlocks`) | ✅ supplied 2026-06-24 — `chainId: 42` (reserved with the Canopy team, confirmed 2026-06-24 — see §E); `redemptionUnstakingBlocks: 30240` matched to Canopy's official `valParams.UnstakingBlocks` |
| E | Off-chain coordination facts | 🟡 partial — closed: bucket-#2/#3 distributors (2026-06-18), chainId `42` reservation (2026-06-24); still pending: fund committee pool 42 (self-funded `MessageSubsidy` is the simplest testnet path — no DAO vote), per-validator `MessageEditStake`, alert webhook URL(s) |

### A. Genesis bucket recipient addresses → `genesis.testnet.json` `buckets[].recipients[].address`  ✅ supplied 2026-06-18
One 20-byte hex address per bucket (7 total). Must be team-controlled testnet
wallets; **none may equal** the localnet placeholder `851e90…d123` (safety check
rejects it). The dev team should document who controls each:

| # | Bucket | bps | On-chain vesting | Status / Address |
|---|---|---|---|---|
| 1 | Validators & Infrastructure | 2200 | 12mo cliff / 3yr | ✅ `83e993da…58cb` |
| 2 | Liquidity Incentives | 1500 | none (off-chain ⚠️) | ✅ `5c9de695…4206` |
| 3 | Community & Airdrops | 2000 | none (off-chain ⚠️) | ✅ `7d941def…e478` |
| 4 | DAO Treasury (canoLiq) | 1500 | none | ✅ `69679898…b4bf` |
| 5 | Founders & Core Team | 1200 | 12mo cliff / 4yr | ✅ `4a0b0aa3…4d22` |
| 6 | Strategic Partners & Integrations | 1000 | 6mo cliff / 18mo | ✅ `8e2c25c9…6d24` |
| 7 | Developer Grants & Ecosystem | 600 | none | ✅ `81be5fbd…8f87` |

⚠️ **Buckets #2 and #3 carry an off-chain schedule.** Their `0/0` on-chain
vesting is intentional — the v1.2 tokenomics schedule is enforced by the bucket
recipient, not by the genesis vesting mechanism (24-month DAO-controlled
emission for Liquidity Incentives; 12-month snapshot-based linear emission for
Community & Airdrops). Dev team confirmed (2026-06-18) that the addresses above
for buckets #2 and #3 are controlled distributors that honor those schedules;
see [the discrepancy report](./canoliq-whitepaper-tokenomics-discrepancies.md)
for the audit trail.

### B. Multisig signers → `genesis.testnet.json` `params.multisigSigners[]`  ✅ supplied 2026-06-18
- N signer addresses (20-byte hex). `multisigThreshold` (default 3) must be ≤ N.
- Optional: a lower `treasuryThreshold` for testnet so the multisig path is
  actually exercised on modest spends.
- *Wired values:* five signer addresses (`1b6454…cb84`, `2ea35a…346f`, `b749e6…a6c5`,
  `08f564…6a4b`, `1b3894…6a44`); `multisigThreshold: 3` (3-of-5); `treasuryThreshold:
  50_000_000` uCPLQ (lowered from the 1B template default so the multisig branch
  is exercised on realistic testnet spends).

### C. Validator registry → `genesis.testnet.json` `validatorRegistry[]`  ✅ shipped empty 2026-06-18
- Per opted-in committee validator: `address` (matches Canopy `Validator.Address`)
  + `stake` (mirror their `StakedAmount` in uCNPY-equivalent).
- *Optional but strongly recommended* — leaving it empty falls back to a single
  aggregator key and obscures per-validator reward credit.
- *Decision (2026-06-18):* ship `validatorRegistry: []` for first testnet bring-up.
  The single-aggregator fallback is defensible while validator details are still
  being locked, but must be populated before the WS3 T1 validator-eject scenarios
  carry meaningful coverage, and **must** be populated before mainnet.
- ⚠️ **These are NOT the §B multisig signers.** The two rosters model different
  roles and must stay distinct (no code couples them):
  - §B `multisigSigners[]` = the **DAO treasury council** (cold/hardware keys that
    authorize treasury spends, 3-of-5; consumed in `treasury.go`/`snapshot.go`).
  - §C `validatorRegistry[]` = the **professional node operators** (hot keys running
    committee infrastructure, bearing slashing risk; their addresses match each
    operator's Canopy `Validator.Address` and must be the same operators that run
    the §E `MessageEditStake` to join committee 42; consumed in `reward.go` for the
    15% validator slice, plus `graduation.go`/`alerts.go`).
  WP §1.2 is explicit that validators are independent operators "not canoLiq
  itself"; reusing signer addresses here would concentrate treasury control and
  validator economics in the same hands (the §8 centralization risk) and drag a
  cold treasury key into a hot-key threat model. Fill the registry with operator
  validator addresses, not the §B signers.

### D. Chain parameters → `canoliq-config.testnet.json`  ✅ supplied 2026-06-24
- `chainId`: the value **reserved with the Canopy team** (no collision with an
  existing committee). *Current file value:* `42` (wired 2026-06-24). Reservation
  with the Canopy team confirmed 2026-06-24 (no committee collision) — see §E.
- `redemptionUnstakingBlocks`: must match the **Canopy testnet's
  `valParams.UnstakingBlocks`**. *Current file value:* `30240` — Canopy's official
  `unstakingBlocks` (supplied 2026-06-24), replacing the `14400` template guess.

### E. Off-chain coordination facts (not file edits, but gating)
- Confirmation the **chainId is reserved**.  ✅ confirmed 2026-06-24 — chainId `42`
  reserved with the Canopy team (no committee collision).
- **Committee reward pool (chainId 42) funded** so `ProcessRewards` is not a no-op
  — until CNPY flows into the pool, no rewards distribute.  ⏳ pending. Three
  non-exclusive paths:
  - **Self-funded `MessageSubsidy`** (CLI `admin tx-subsidy <sender> <amount> 42
    <opcode>`) — a plain transaction that debits the sender and credits pool 42
    (`fsm/message.go::HandleMessageSubsidy`). **No Canopy DAO vote required.** This
    is the simplest testnet unblock: seed pool 42 from a team wallet.
  - **Auto-subsidization** — protocol auto-mints CNPY to any committee holding ≥33%
    of total network restake (WP §6.2); requires launch validators to push chainId
    42 over the threshold. canoLiq's day-one goal, but not required for testnet.
  - **Canopy DAO treasury subsidy** — a `tx-dao-transfer` governance *proposal*
    (CLI `admin tx-dao-transfer`, subject to Canopy's DAO vote) for sustained
    funding; this is the "formal proposal to Canopy DAO" of WP §6.3. Matters more
    for mainnet than for testnet bring-up.
- Confirmation each committee validator has run **`MessageEditStake`** adding the
  chainId to `Validator.Committees[]`.  ⏳ pending
- Webhook URL(s) for alerts (optional) → `CANOLIQ_ALERT_URL` / `alerts` config.  ⏳ pending
- **(Closed 2026-06-18)** Bucket #2 / #3 recipient addresses are controlled
  distributors that honor the off-chain emission schedule (see §A footnote).

---

## Workstream 1 — Wire dev-provided data (A–D wired)

Edit the two committed files and re-verify invariants. A landed in commit
`f3fa10e2`; B + C landed alongside this doc update; D (chainId `42` +
redemptionUnstakingBlocks `30240`) wired 2026-06-24.

- `plugin/go/canoliq/genesis.testnet.json` — ~~replace the 7 bucket addresses~~ ✅,
  ~~the multisig signers~~ ✅, ~~and the validator registry entries~~ ✅ (shipped empty);
  ~~adjust `multisigThreshold` / `treasuryThreshold` if requested~~ ✅.
- `plugin/go/canoliq/canoliq-config.testnet.json` — ~~set `chainId` +
  `redemptionUnstakingBlocks`~~ ✅ (`42` / `30240`).
- Do **not** touch bucket `bps`, recipient `bps`, or vesting (`cliffMonths`/
  `vestMonths`) — those are spec-fixed and validated (`genesis.go::validateGenesis`
  requires bucket bps and per-bucket recipient bps to each total 10000;
  `config.go::ValidateParams` enforces fee bounds + `multisigThreshold ≤ signers`).
- Update `TestBundledTestnetGenesisIsSafetyCheckClean` expectations only if the
  structure changes (it shouldn't).

**Gate:** `go test ./canoliq/...` green; `SafetyCheck` passes under
`profile=testnet` (no placeholder remains).

## Workstream 2 — Pre-flight verification on a private testnet image

Follow README Phase 2 (lines 323–417). The compose file now exists:
`.docker/compose.testnet.yaml` (created 2026-06-24) mirrors `.docker/compose.yaml`
with `CANOLIQ_CONFIG` pointed at `…/canoliq-config.testnet.json`. The Dockerfile
already bundles both genesis + config variants, so no rebuild logic changes.

- **2.1 Safety banner + check** — boot, confirm the `profile="testnet"` banner and
  that genesis self-bootstraps.
- **2.2 Bucket reconciliation** — assert 100M × 10⁶ uCPLQ distributed exactly per
  bucket/recipient bps; vesting buckets create `VestingSchedule` records, liquid
  buckets credit balances + `cplqCirculatingSupply`.
- **2.3 Lifecycle smoke** — deposit → redeem → claim after
  `redemptionUnstakingBlocks`; confirm `/v1/account/{addr}` lists redemptions.
- **2.4 Multisig rehearsal** — below-threshold spend rejected; execution succeeds
  after timelock + approvals.

## Workstream 3 — Live T1–T6 feature run-through (the real gap)

The testnet exit criteria require these exercised on a live chain, not just unit
tests. On the `compose.testnet.yaml` image, drive via `canoliqctl` + `/v1/*`:

- **T1** — tiered proposal round-trips: fee-change (5%/51%/48h), validator-eject,
  emergency fast-track (24h), large treasury spend (10%/67%/7d). Confirm each tier's
  quorum/approval/timelock enforced live.
- **T2** — stake with `--lock 12m`/`24m`; confirm boosted vote weight and that a
  `BUYBACK_DISTRIBUTE_STAKERS` execution boosts locked stakers; unstake rejected
  before `lock_end_height`.
- **T3** — set a low `tvl_cap_ucnpy` via param-change; deposit at cap accepted,
  above rejected; `/v1/health` shows utilization.
- **T4** — observe insurance skim auto-off once reserve hits 5% of peak TVL;
  `/v1/pools` shows `peakTvlUcnpy` / `insuranceFundedBps`.
- **T5** — `/v1/graduation` populated; counters advance (passed proposals, daily-tx
  window, turnout).
- **T6** — point `CANOLIQ_ALERT_URL` at a mock receiver (or real Slack/Discord);
  trigger buyback-drain + validator-concentration; confirm POSTs + debounce.

Capture results in a run-through log; fix any divergence from the unit-test
behaviour before sign-off.

## Workstream 4 — Governance security self-review (+ flag external)

Run a focused security pass (the `security-review` skill) over the highest-risk
surface, then write up findings and recommend an independent review before mainnet:
- T1 multi-tier dispatch (`governance.go` tally/`dispatchPassed`, tier snapshot,
  `actionTypeForPayload` small/large boundary, validator-eject idempotency).
- T2 lock-tier weighting (`voteWeightFor`, unstake lock gate, buyback boost
  conservation).
- Treasury multisig + timelock (`treasury.go`), and the alert evaluation's
  state-determinism (`alerts.go`).

## Workstream 5 — Cutover coordination

Per README Phase 3 (lines 419–444), once E is confirmed: hash-anchor the final
genesis + config, distribute the image to validators with
`CANOPY_PLUGIN_MODE=canoliq` + `CANOLIQ_CONFIG=…/canoliq-config.testnet.json`,
and verify on the real chain (`/v1/health.genesisComplete`, `/v1/validators`
matches the seeded set, `/v1/pools.committeePool` growing). **First-block genesis
is one-time and irreversible** — the 100M CPLQ mint to bucket addresses cannot be
redone, so Workstreams 1–2 must be signed off first.

---

## Verification

- **Unit:** `cd plugin/go && go test ./canoliq/... ./canoliqctl/...` green.
- **Safety:** plugin boots under `profile=testnet` with the banner and no
  placeholder-refusal error.
- **Reconciliation:** bucket distribution sums to exactly 100M × 10⁶ uCPLQ across
  recipients (Workstream 2.2).
- **Live features:** the Workstream 3 run-through log shows each T1–T6 behaviour on
  a multi-node chain matching unit-test expectations.
- **Exit criteria** (`docs/plans/canoliq-release-plan.md` "Testnet exit criteria"):
  tiered proposals, tier-boosted rewards, TVL-cap rejection, insurance auto-off,
  `/v1/graduation`, alert events — all observed live.

## Critical files

| Concern | File |
|---|---|
| Genesis values to fill | `plugin/go/canoliq/genesis.testnet.json` |
| Chain config to fill | `plugin/go/canoliq/canoliq-config.testnet.json` |
| Testnet compose | `.docker/compose.testnet.yaml` ✅ (created 2026-06-24) |
| Safety check / param validation | `plugin/go/canoliq/config.go` (`SafetyCheck`, `ValidateParams`) |
| Genesis invariants | `plugin/go/canoliq/genesis.go` (`validateGenesis`, `applyGenesisBuckets`) |
| Deploy guide (reference, don't duplicate) | `plugin/go/canoliq/README.md` §"Testnet deployment" |
| Exit criteria | `docs/plans/canoliq-release-plan.md` |

## Out of scope (follow-ups, not testnet blockers)

- Stuck-redemption alert condition (needs a global mature-unclaimed-redemption
  index; T6 shipped the other three conditions).
- Pre-existing `go vet` copylocks warning at `phase2_test.go:208`.
- Stale `README.md` "Insurance fund" narration (`insurance_bps=1500` → should read
  500 post-F5).
- Independent/external security audit (recommended before *mainnet*; testnet uses
  the self-review pass).
