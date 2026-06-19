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

## 🔑 Data required from the dev team (blocking inputs)

Deployment cannot proceed until the team supplies all of the following. This is
the critical hand-off — everything else is execution.

**Status snapshot (2026-06-18):**

| Section | What | Status |
|---|---|---|
| A | Genesis bucket recipient addresses | ✅ supplied (2026-06-18) |
| B | Multisig signers | ⏳ pending — `genesis.testnet.json` still carries `…b1`–`…b5` placeholders |
| C | Validator registry | ⏳ pending — still `…c01`, `…c02` placeholders |
| D | Chain parameters (`chainId`, `redemptionUnstakingBlocks`) | ⏳ pending — `chainId: 2` / `redemptionUnstakingBlocks: 14400` are the template defaults |
| E | Off-chain coordination facts | ⏳ pending (one item closed: bucket-#2/#3 recipients are controlled distributors, confirmed 2026-06-18) |

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

### B. Multisig signers → `genesis.testnet.json` `params.multisigSigners[]`  ⏳ pending
- N signer addresses (20-byte hex). `multisigThreshold` (default 3) must be ≤ N.
- Optional: a lower `treasuryThreshold` for testnet so the multisig path is
  actually exercised on modest spends.
- *Current placeholders in the file:* `00000000…b1` through `00000000…b5`,
  threshold `3`, treasuryThreshold `1_000_000_000`.

### C. Validator registry → `genesis.testnet.json` `validatorRegistry[]`  ⏳ pending
- Per opted-in committee validator: `address` (matches Canopy `Validator.Address`)
  + `stake` (mirror their `StakedAmount` in uCNPY-equivalent).
- *Optional but strongly recommended* — leaving it empty falls back to a single
  aggregator key and obscures per-validator reward credit.
- *Current placeholders in the file:* `0000…c01` (stake 1B) and `0000…c02` (stake 1B).

### D. Chain parameters → `canoliq-config.testnet.json`  ⏳ pending
- `chainId`: the value **reserved with the Canopy team** (no collision with an
  existing committee). *Current file value:* `2` (template default — confirm
  with the Canopy team whether this is the reserved testnet value).
- `redemptionUnstakingBlocks`: must match the **Canopy testnet's
  `valParams.UnstakingBlocks`**. *Current file value:* `14400` (the template's
  guess — confirm against the live testnet value).

### E. Off-chain coordination facts (not file edits, but gating)
- Confirmation the **chainId is reserved**.  ⏳ pending
- A **`MessageSubsidy` proposal** queued/passed on the Canopy DAO (until it passes,
  `ProcessRewards` is a no-op — no rewards flow).  ⏳ pending
- Confirmation each committee validator has run **`MessageEditStake`** adding the
  chainId to `Validator.Committees[]`.  ⏳ pending
- Webhook URL(s) for alerts (optional) → `CANOLIQ_ALERT_URL` / `alerts` config.  ⏳ pending
- **(Closed 2026-06-18)** Bucket #2 / #3 recipient addresses are controlled
  distributors that honor the off-chain emission schedule (see §A footnote).

---

## Workstream 1 — Wire dev-provided data (once B–D arrive; A already wired)

Edit the two committed files and re-verify invariants. A landed in commit
`f3fa10e2`; B, C, D remain.

- `plugin/go/canoliq/genesis.testnet.json` — ~~replace the 7 bucket addresses~~ ✅,
  the multisig signers, and the validator registry entries; adjust
  `multisigThreshold` / `treasuryThreshold` if requested.
- `plugin/go/canoliq/canoliq-config.testnet.json` — set `chainId` +
  `redemptionUnstakingBlocks`.
- Do **not** touch bucket `bps`, recipient `bps`, or vesting (`cliffMonths`/
  `vestMonths`) — those are spec-fixed and validated (`genesis.go::validateGenesis`
  requires bucket bps and per-bucket recipient bps to each total 10000;
  `config.go::ValidateParams` enforces fee bounds + `multisigThreshold ≤ signers`).
- Update `TestBundledTestnetGenesisIsSafetyCheckClean` expectations only if the
  structure changes (it shouldn't).

**Gate:** `go test ./canoliq/...` green; `SafetyCheck` passes under
`profile=testnet` (no placeholder remains).

## Workstream 2 — Pre-flight verification on a private testnet image

Follow README Phase 2 (lines 323–417). Create the missing compose file (the repo
has none): copy `.docker/compose.yaml` → `.docker/compose.testnet.yaml`, switching
the `CANOLIQ_CONFIG` env to `…/canoliq-config.testnet.json`. The Dockerfile already
bundles both genesis + config variants, so no rebuild logic changes.

- **2.1 Safety banner + check** — boot, confirm the `profile="testnet"` banner and
  that genesis self-bootstraps.
- **2.2 Bucket reconciliation** — assert 100M × 10⁶ uCLIQ distributed exactly per
  bucket/recipient bps; vesting buckets create `VestingSchedule` records, liquid
  buckets credit balances + `cliqCirculatingSupply`.
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
is one-time and irreversible** — the 100M CLIQ mint to bucket addresses cannot be
redone, so Workstreams 1–2 must be signed off first.

---

## Verification

- **Unit:** `cd plugin/go && go test ./canoliq/... ./canoliqctl/...` green.
- **Safety:** plugin boots under `profile=testnet` with the banner and no
  placeholder-refusal error.
- **Reconciliation:** bucket distribution sums to exactly 100M × 10⁶ uCLIQ across
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
| Testnet compose (to create) | `.docker/compose.testnet.yaml` (copy of `.docker/compose.yaml`) |
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
