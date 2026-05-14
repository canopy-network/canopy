# canoLiq v2 → v1.1 Spec-vs-Code Audit

## Context

The canoLiq team revised the protocol spec from **v2** (early draft) to **v1.1** (May 2025).
The current canoliq branch was implemented against the v2 spec and against partial reads of v1.1.
This audit identifies every discrepancy between the v1.1 spec (`canoLiq_Whitepaper_v1.1.pdf` +
`canoLiq_Tokenomics_v1.1.pdf`) and the current code on the `canoliq` branch, organized so the
team can decide what to implement.

**Deliverable**: this audit document. No code changes. Each finding lists file:line, current
value, v1.1 spec value, and recommended fix, sized by effort.

**Resolved interpretation**: per user input, vesting `Duration` columns in v1.1 mean **total
span including cliff**. So `cliffMonths + vestMonths` must equal the documented duration.

---

## Spec Evolution v2 → v1.1 (what changed)

| Area | v2 (old) | v1.1 (new) |
|------|----------|------------|
| Canopy 5% DAO tax | "Canopy takes 5% on-chain prior to committee distribution" | **Removed.** Whitepaper §3.3: *"Canopy does NOT apply a protocol-level DAO tax on top of rewards before distribution."* |
| Founders vesting | "3-year vesting with 6–12 month cliff" | **4-year linear with 12-month cliff** (explicit) |
| Validator vesting | "subject to lockups" (vague) | **3-year linear with 12-month cliff.** Explicit rationale: *"A 6-month cliff is insufficient given the operational commitment required."* |
| Strategic Partners | "subject to lockups" (vague) | **18-month linear with 6-month cliff** |
| Insurance Fund | 1–2% of treasury | **5% of DAO treasury, target 5% of peak TVL within 12 months** |
| Governance | multisig + timelock (generic) | **7-tier matrix** with distinct quorum / approval / timelock per action type |
| Vote-escrow | "stake/lock for boosts" (generic) | **5-tier explicit schedule** (1×–4× voting, base to +75% reward boost) |
| TVL self-cap | not specified | **33% of total Canopy network stake** |
| Fee rate bounds | "subject to governance" | **5%–20%** explicit bounds |
| Autonomy graduation | objective thresholds (vague) | **Concrete numbers**: >$50M TVL, >30 validators, >15% governance turnout, >10k tx/day, >12mo runway |
| Distribution (22/15/20/15/12/10/6) | same | same — no change |
| Fee split (40/30/15/15) | same | same — no change |
| Default fee (12%) | same | same — no change |

---

## Findings

Each finding ranked **P0 (blocking spec violation)** / **P1 (significant gap)** / **P2 (new feature)** / **P3 (cosmetic)**.

### P0 — Blocking spec violations

These items make the code disagree with explicit v1.1 numbers. Each is a one-to-few-line change.

#### F1. Validator vesting cliff is 6 months, spec mandates 12 months
- **Spec**: Tokenomics v1.1 §2.1 — *"3-year linear with 12-month cliff. Rationale for 12-month cliff: ... A 6-month cliff is insufficient given the operational commitment required."*
- **Current**: `plugin/go/canoliq/genesis.localnet.json:6-8` and `genesis.testnet.json:6-9` — `cliffMonths: 6, vestMonths: 24`
- **Fix**: change both files to `cliffMonths: 12, vestMonths: 24` (12mo cliff + 24mo linear = 36mo / 3yr total).
- **Effort**: 2 lines × 2 files.

#### F2. Founders vest duration is 3 years, spec mandates 4 years
- **Spec**: Tokenomics v1.1 §2.5 — *"4-year linear with 12-month cliff. No early unlocks."*
- **Current**: `genesis.localnet.json:55-56` and `genesis.testnet.json:56-57` — `cliffMonths: 12, vestMonths: 24` (= 36mo / 3yr total, matching v2 spec)
- **Fix**: change to `cliffMonths: 12, vestMonths: 36` (12mo cliff + 36mo linear = 48mo / 4yr total).
- **Effort**: 2 lines × 2 files.

#### F3. Strategic Partners total span 24mo, spec mandates 18mo
- **Spec**: Tokenomics v1.1 §2.6 — *"18-month linear vesting with 6-month cliff"*. Summary table §6: Cliff 6 mo, Duration 18 months.
- **Current**: `genesis.localnet.json:67-68` and `genesis.testnet.json:68-69` — `cliffMonths: 6, vestMonths: 18` (= 24mo total).
- **Fix**: change to `cliffMonths: 6, vestMonths: 12` (6mo cliff + 12mo linear = 18mo total).
- **Effort**: 2 lines × 2 files.
- **Note**: this is the one finding whose direction depends on the interpretation choice. Under user's chosen reading ("Duration includes cliff"), current code is wrong.

#### F4. Protocol fee bounds (5%–20%) not enforced
- **Spec**: Tokenomics v1.1 §3.3 — *"CLIQ holders may vote to adjust within the protocol-defined bounds of 5% to 20%."* Whitepaper §4.1 same.
- **Current**: `plugin/go/canoliq/config.go:243` — only checks `p.FeeBps > 10_000` (blocks >100%, allows 0–100%).
- **Fix**: in `ValidateParams`, add `if p.FeeBps < 500 || p.FeeBps > 2000 { return ErrInvalidParams() }`. A `param-change` proposal violating these bounds should be rejected.
- **Effort**: ~3 lines in `config.go:239-268`.

#### F5. Insurance bps target is 15% of treasury slice, spec mandates 5% of DAO treasury
- **Spec**: Tokenomics v1.1 §8 — *"An insurance fund of 5% of DAO treasury — accumulated progressively from protocol fee income over the first 12 months, with a target of 5% of peak TVL."* Whitepaper §9.2 same.
- **Current**: `plugin/go/canoliq/config.go:219` — `InsuranceBps: 1500` (15% of the treasury fee slice ≈ 1.5% of fee total).
- **Ambiguity**: the v1.1 *"5% of DAO treasury"* could be read as (a) skim 5% of every treasury inflow into insurance, or (b) accumulate until insurance reserve = 5% of treasury balance. (b) is a periodic-target, not a continuous skim — would require new state.
- **Fix (minimal)**: change `InsuranceBps: 1500 → 500` to match reading (a). Add a TODO referencing the "target 5% of peak TVL" cap that needs to be a separate periodic check.
- **Effort**: 1 line; plus a peak-TVL tracker is P2 work.

---

### P1 — Significant gaps (no behavior, but spec calls them out)

#### F6. AGENTS.md and test comments bake in v2's "5% Canopy pre-cut" narrative
- **Spec**: Whitepaper v1.1 §3.3 — *"⚠ IMPORTANT: Canopy does NOT apply a protocol-level DAO tax on top of rewards before distribution."* Tokenomics v1.1 §4.1 *"Effective user yield = 88% × Rewards Received."*
- **Current behaviour**: `plugin/go/canoliq/reward.go`, `fee.go`, `EndBlock` math — all operate on inflows as-received, no `0.95` factor. **Production code is correct under v1.1.**
- **Current docs/tests**:
  - `plugin/go/canoliq/AGENTS.md:101-116` — entire "Whitepaper §7 reconciliation" section assumes Canopy 5% pre-cut.
  - `plugin/go/canoliq/canoliq_test.go:258, 265, 288` — comments reference `0.88 * 0.95 * X` and `(whitepaper §7: 0.88 * 0.95 * X with truncation)`.
  - `plugin/go/canoliq/rpc_test.go:133` — comment references `post-DAO 0.95X`.
- **Fix**: rewrite the AGENTS.md section to describe the v1.1 model: canoLiq receives `R` directly from committee distribution, applies 12% fee on `R`, user yield = `0.88 * R`. Update test comments to drop the `0.95` factor — the test inputs (1000 uCNPY into the pool) stay the same, but the narrative becomes "given canoLiq's received share R=1000" rather than "given gross X=1053 with 0.95X reaching the pool."
- **Effort**: ~30 lines of comment changes across 3 files. No test code logic changes needed (numbers stay the same once you reframe the input as `R` directly).

#### F7. Multi-tier governance not implemented — single global threshold for all proposals
- **Spec**: Tokenomics v1.1 §7 defines a 7-row table:
  | Action | Quorum | Approval | Timelock |
  |---|---|---|---|
  | Fee rate adjustment | 5% | 51% | 48h |
  | Treasury spend (small) | 5% | 51% | 48h |
  | Treasury spend (large, >1M CLIQ) | 10% | 67% | 7d |
  | Emergency security action | 8% | 67% | 24h fast-track |
  | Validator ejection | 5% | 51% | 48h |
  | Protocol upgrade | 10% | 67% | 7d |
  | Autonomy graduation | 15% | 75% | 14d |
- **Current**: `plugin/go/canoliq/config.go:223-226` — one global `QuorumBps: 3300`, `PassThresholdBps: 5001`, `TimelockBlocks: 28_800`. Treasury already has small/large path via `TreasuryThreshold` (`config.go:220`, `treasury.go:32-35`) but uses the same quorum/approval for both.
- **Fix scope**: requires (a) tagging each proposal with an `ActionType` enum, (b) parameter struct per type, (c) `governance.go` reading the right threshold per proposal, (d) genesis-loadable defaults. Validator ejection, emergency, autonomy graduation, protocol upgrade as proposal types may not exist at all yet — need new payload definitions in `proto/canoliq.proto`.
- **Effort**: multi-day feature. Roughly: proto changes + 200-400 lines across `governance.go`, `treasury.go`, `config.go`, `canoliqctl/proposal_create.go`, plus tests.

#### F8. Vote-escrow lock multipliers not implemented
- **Spec**: Tokenomics v1.1 §4.2:
  | Lock | Voting × | Reward boost |
  |---|---|---|
  | None | 1× | base |
  | 3mo | 1.5× | +10% |
  | 6mo | 2× | +25% |
  | 12mo | 3× | +50% |
  | 24mo | 4× | +75% |
- **Current**: `plugin/go/canoliq/governance.go:246-252` (`voteWeightFor`) — voting weight = raw staked CLIQ amount, no lock multiplier. Reward boost not applied anywhere in `reward.go`.
- **Fix scope**: add lock duration to `CLIQStake` (proto change), tier resolver function, multiplier in `voteWeightFor`, reward-share boost path in `reward.go::distributeValidatorShare` and buyback distribution. Needs unstake-when-lock-expires semantics + tests for boundary cases.
- **Effort**: multi-day feature. ~300-500 lines + proto changes + tests.

#### F9. TVL self-cap (33% of Canopy network stake) not enforced
- **Spec**: Whitepaper v1.1 §9.4 — *"canoLiq will self-impose a TVL cap of 33% of total Canopy network stake pending ecosystem maturation and governance approval to lift this cap."*
- **Current**: no enforcement in deposit path.
- **Fix scope**: depends on whether the plugin can query total Canopy stake. If yes, add check in `canoliq.go::Deposit` to reject new deposits when `pool/total_canopy_stake ≥ 33%`. If no, add a governance-set cap parameter and check against that.
- **Effort**: 1 day if the network-stake query exists; 2-3 days if a governance-set ceiling has to be added instead.

---

### P2 — Net-new v1.1 features

#### F10. Insurance fund "5% peak TVL" target with periodic tracking
- **Spec**: Tokenomics v1.1 §8, Whitepaper §9.2 — *"target of 5% of peak TVL within 12 months of mainnet."*
- **Current**: continuous skim via `InsuranceBps` (see F5) but no peak-TVL tracker and no cap-when-target-met logic.
- **Fix scope**: add `PeakTvl` to globals, update on every `EndBlock`, gate insurance skim off when reserve ≥ 5% of peak TVL.
- **Effort**: 1-2 days + state migration consideration.

#### F11. Autonomy graduation criteria + proposal type
- **Spec**: Whitepaper v1.1 §10 — concrete table of 5 thresholds (TVL, validator count, governance turnout, tx/day, runway). Tokenomics §7 — autonomy graduation vote: 15% quorum, 75% approval, 14d timelock.
- **Current**: no graduation proposal type, no threshold tracking.
- **Fix scope**: new proposal payload, threshold-check helper, mostly informational (graduation itself is cross-chain coordination, not an on-chain primitive).
- **Effort**: 1-2 days for the on-chain part.

#### F12. Validator ejection proposal type
- **Spec**: Tokenomics v1.1 §7 — *"Validator ejection: 5% quorum, 51% approval, 48h timelock."* Whitepaper §1.2: *"Are subject to ongoing performance review and can be ejected from the validator set by CLIQ governance vote."*
- **Current**: validator registry exists (`genesis.go:130`), but no ejection proposal path.
- **Fix scope**: new proposal payload type, eject-validator-from-registry handler.
- **Effort**: 1 day.

#### F13. Emergency security action (fast-track 24h proposal)
- **Spec**: Tokenomics v1.1 §7 — *"Emergency security action: 8% quorum, 67% approval, 24h fast-track."*
- **Current**: no emergency path; all proposals use the standard `VotingPeriodBlocks` and `TimelockBlocks`.
- **Fix scope**: requires per-proposal timing parameters (overlaps F7).
- **Effort**: comes mostly free once F7 is done.

---

### P3 — Cosmetic / informational

#### F14. Genesis bucket name "Plugin & Dev Grants" — spec calls it "Developer Grants & Ecosystem"
- **Current**: `genesis.localnet.json:78`, `genesis.testnet.json:79`.
- **Effort**: rename; or leave as-is and update spec terminology comment.

#### F15. Genesis bucket name "Liquidity Incentives (Farming)" — spec calls it "Liquidity Incentives"
- **Current**: `genesis.localnet.json:17`, `genesis.testnet.json:18`. The "(Farming)" suffix tracks v2 naming; v1.1 drops it.
- **Effort**: rename.

---

## Recommended Fix Tiers (suggested cut points)

The team can implement these in waves. From cheapest-to-most-correct upward:

- **Wave 1 — Genesis & validation (half-day)**: F1, F2, F3, F4, F5. All are 1-3 line changes. Adds bounds-checking and brings genesis numbers to v1.1.
- **Wave 2 — Doc cleanup (half-day)**: F6, F14, F15. Removes v2 narrative from AGENTS.md / test comments / bucket names. Behaviour unchanged.
- **Wave 3 — Per-action governance + ejection (1-2 weeks)**: F7, F12, F13 together. F7 is the umbrella feature; ejection and emergency fall out naturally.
- **Wave 4 — Vote-escrow (1 week)**: F8. Proto change + tier resolver + boost in reward path + redesign of stake/unstake flow.
- **Wave 5 — Caps & autonomy (3-5 days)**: F9, F10, F11. Mostly state-tracking + check logic.

---

## Critical files for any of the above fixes

| Concern | File |
|---|---|
| Genesis numeric values | `plugin/go/canoliq/genesis.localnet.json`, `genesis.testnet.json` |
| Param defaults & validation | `plugin/go/canoliq/config.go` (esp. `DefaultParams`, `ValidateParams`) |
| Reward math (the part that's already correct) | `plugin/go/canoliq/reward.go`, `fee.go` |
| Doc/test narrative | `plugin/go/canoliq/AGENTS.md`, `canoliq_test.go`, `rpc_test.go` |
| Governance proposal handling | `plugin/go/canoliq/governance.go`, `treasury.go`, `proto/canoliq.proto` |
| Staking & voting weight | `plugin/go/canoliq/stake.go`, `governance.go::voteWeightFor` |
| Deposit/redeem (where TVL cap lands) | `plugin/go/canoliq/canoliq.go`, `deposit.go`, `redeem.go` |
| CLI surface | `plugin/go/canoliqctl/proposal_create.go` and siblings |

## Verification (when fixes are eventually implemented)

- **Wave 1**: run `go test ./plugin/go/canoliq/...` — existing tests should pass after the F1–F3 changes because vesting unit tests assert against the configured `cliffMonths`/`vestMonths`, not hardcoded month counts. F4/F5 may require updating one or two test fixtures.
- **Wave 2**: no behavioural test changes; doc proofread plus `go test ./plugin/go/canoliq/...` to confirm nothing referenced the renamed buckets by string match.
- **Waves 3–5**: each item should ship with new tests covering the new threshold/multiplier/proposal type, plus an updated `phase2_test.go` (or a new `phase3_test.go`) demonstrating end-to-end flow.

## Open items / not covered

- **Snapshot-based airdrop eligibility** (Tokenomics §2.2) — outside the contract surface; off-chain process.
- **Quadratic / conviction voting** (Tokenomics §8 risk mitigation) — explicitly noted in spec as *"may be explored post-launch"*; not actionable now.
- **Audit / bug bounty** (Whitepaper §9.1) — operational, not code.
- **Canopy DAO subsidy proposal submission** (Tokenomics §5.3) — off-chain governance work for Canopy DAO, not canoLiq code.
