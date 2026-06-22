# canoLiq v1.1 — Whitepaper vs. Tokenomics Discrepancy Report

**Date:** 2026-05-22
**Documents compared:**
- `canoLiq_Whitepaper_v1.1.pdf` (Technical Whitepaper, Version 1.1, May 2025)
- `canoLiq_Tokenomics_v1.1.pdf` (CLIQ Token Design & Protocol Economics, v1.1, May 2025)

> **Status (verified 2026-06-18, closed 2026-06-18): All findings resolved.**
> All four doc-vs-doc discrepancies are fixed in v1.2 — see
> [§ Resolution in v1.2](#resolution-in-v12-verified-2026-06-18) for the per-item
> verification against `canoLiq_Whitepaper_v1.2.pdf` and `canoLiq_Tokenomics_v1.2.pdf`.
> The Community & Airdrops genesis-vesting gap noted in
> [§ Additional code/doc gap](#additional-code-doc-gap-community--airdrops) was a
> distributor-vs-genesis question, not a doc-vs-doc one; the dev team has confirmed
> that the bucket #3 recipient is a controlled distributor that enforces the
> 12-month linear emission, so no docs or genesis change is required.

---

## Material discrepancy

### 1. Validators & Infrastructure vesting cliff — 6 months vs. 12 months

| Document | Location | Cliff |
|---|---|---|
| Whitepaper | §5.2 Distribution table | "3-year linear; **6-month cliff**" |
| Tokenomics | §2.1 and §6 summary table | "3-year linear with **12-month cliff**" |

This is a direct contradiction and the most significant finding. The Tokenomics doc
does not merely state 12 months — §2.1 includes a dedicated rationale bullet explicitly
arguing that **"A 6-month cliff is insufficient given the operational commitment
required."** The two documents therefore actively disagree on the design choice, not
just a transcription error. One document must be corrected.

---

## Moderate discrepancy

### 2. Community & Airdrops vesting type — "Milestone-based" vs. snapshot-based linear

| Document | Location | Description |
|---|---|---|
| Whitepaper | §5.2 table | "**Milestone-based**; 12-month distribution" |
| Tokenomics | §2.2 / §6 table | "Snapshot-based... 12-month **linear** emission" / "Linear daily" |

The 12-month period agrees, but the mechanism conflicts: the whitepaper labels it
"Milestone-based" (the same label it applies to Dev Grants), while the Tokenomics doc
describes a snapshot-based linear/daily emission. This appears to be a copy-paste error
in the whitepaper's table — the Tokenomics description is internally consistent and more
detailed.

---

## Minor discrepancies

### 3. 6-month subsidy emission reduction — 30–45% vs. 33–45%

- Whitepaper §6 scenario table: "**30**–45% reduction"
- Tokenomics §5 scenario table: "~**33**–45% reduction"

Same upper bound, different lower bound. (Tokenomics §2.4 separately states "30–60%",
which is consistent with both figures.)

### 4. Buyback default mechanism

- Whitepaper §8.1 governance table lists the buyback **Default** as "Burn" (with
  burn-vs-distribute by quarterly DAO vote).
- Tokenomics states only "burn or distribute per DAO vote" and never names a default.

Not strictly contradictory, but the Tokenomics doc omits the stated default. Worth aligning.

---

## Verified consistent (no conflict)

- Total supply 100M fixed; full distribution table (22/20/15/15/12/10/6%) and token amounts
- Fee rate 12%, bounds 5–20%, 48h timelock; fee split 40/30/15/15 and the 92.8% effective yield math
- Founders (4yr / 12mo cliff), Strategic Partners (18mo / 6mo cliff), DAO Treasury, Liquidity (24mo), Dev Grants
- Auto-subsidization threshold (33%), manual subsidy tranches ($500K / $2M / $10M), 6–12 month duration
- Insurance fund (5% of treasury, target 5% of peak TVL, accumulated over first 12 months)
- Governance quorums / thresholds / timelocks (Tokenomics §7 is more granular but does not conflict with the whitepaper)

---

## Codebase reconciliation

Checked against the on-chain vesting buckets in `plugin/go/canoliq/genesis.testnet.json`
and `genesis.localnet.json` (both identical). The vesting convention in
`genesis.go` is: total window = `cliffMonths + vestMonths`, with linear unlock between
the cliff and the end height.

| Bucket | Code (cliff / vest) | Total window | Whitepaper | Tokenomics |
|---|---|---|---|---|
| **Validators & Infrastructure** | **12mo cliff + 24mo vest** | 36mo (3yr) | 3yr, **6mo cliff** ❌ | 3yr, **12mo cliff** ✅ |
| Founders & Core Team | 12mo + 36mo | 48mo (4yr) | 4yr, 12mo cliff ✅ | 4yr, 12mo cliff ✅ |
| Strategic Partners | 6mo + 12mo | 18mo | 18mo, 6mo cliff ✅ | 18mo, 6mo cliff ✅ |

**The implementation resolves discrepancy #1 in favor of the Tokenomics doc:** the
validator bucket uses a **12-month cliff**. The whitepaper's "6-month cliff" is the
outlier, contradicted by both the Tokenomics doc and the code. Fix should land in the
whitepaper.

<a id="additional-code-doc-gap-community--airdrops"></a>
### Additional code/doc gap (Community & Airdrops)

`Community & Airdrops` and `Liquidity Incentives` are both configured
`cliffMonths: 0, vestMonths: 0`, which per `genesis.go:231` mints the full allocation
**liquid at TGE** with no on-chain vesting. Both documents describe a 12-month
distribution (Community) and 24-month emission (Liquidity).

- **Liquidity Incentives:** plausibly fine — docs describe "DAO-controlled emission," so
  tokens held in a controller/distributor account is not a contradiction.
- **Community & Airdrops:** the docs' "12-month linear / linear daily" emission is **not**
  enforced by the genesis vesting mechanism. It must be handled by the distributor
  address receiving the bucket. Confirm this so the airdrop is not unintentionally fully
  unlocked at launch.
  - **Resolved 2026-06-18:** dev team confirmed the bucket #3 recipient in
    `genesis.testnet.json` is a controlled distributor that enforces the 12-month
    linear emission. No genesis or doc change required.

## Recommended actions

> ⚠️ **Superseded** — actions #1–#3 below were the v1.1 recommendations and have been
> applied in v1.2 (see [§ Resolution in v1.2](#resolution-in-v12-verified-2026-06-18)).
> Kept for historical context.

1. **#1 (validator cliff)** requires a decision — the documents contradict each other and the
   Tokenomics doc argues against the whitepaper's value. Pick 6 or 12 months and align both
   documents (and verify the codebase genesis vesting matches).
2. **#2** is almost certainly a labeling slip in the whitepaper; update its §5.2 table to
   "Snapshot-based linear; 12-month distribution".
3. **#3 / #4** are small wording/number alignments.

---

<a id="resolution-in-v12-verified-2026-06-18"></a>
## Resolution in v1.2 (verified 2026-06-18)

**Documents checked:**
- `canoLiq_Whitepaper_v1.2.pdf` (Technical Whitepaper, Version 1.2, May 2025)
- `canoLiq_Tokenomics_v1.2.pdf` (CLIQ Token Design & Protocol Economics, v1.2, May 2025)

| # | Severity | v1.1 finding | v1.2 status |
|---|----------|---------------------------------------------------------|-------------|
| 1 | Material | Validator cliff: WP 6mo vs Tok 12mo                     | ✅ Resolved — WP corrected to 12mo |
| 2 | Moderate | Community: WP "Milestone-based" vs Tok snapshot-linear  | ✅ Resolved — WP corrected to snapshot-based linear |
| 3 | Minor    | 6mo subsidy: WP 30–45% vs Tok 33–45%                    | ✅ Resolved — both now say 30–45% |
| 4 | Minor    | Buyback default: WP "Burn" vs Tok unstated              | ✅ Resolved — Tok now names "Burn" in three places |

### #1 — Validator cliff (Material) ✅

- **Whitepaper v1.2 §5.2 table (p. 7):** "Validators & Infrastructure … 3-year vesting;
  **12-month cliff**".
- **Whitepaper v1.2 §5.3 rationale (p. 7):** "The **12-month cliff (not 6 months)** ensures
  validators remain committed through the highest-risk first year of mainnet operation
  before any tokens unlock." The corrected value is explicitly called out.
- **Tokenomics v1.2 §2.1 (p. 3):** unchanged — "3-year linear with 12-month cliff" with the
  "A 6-month cliff is insufficient" rationale intact.
- **Code:** `plugin/go/canoliq/genesis.testnet.json` bucket #1 = `cliffMonths: 12,
  vestMonths: 24` (36mo total = 3yr). Matches both docs.

### #2 — Community & Airdrops mechanism (Moderate) ✅

- **Whitepaper v1.2 §5.2 table (p. 6):** "Community & Airdrops … **Snapshot-based linear;
  12-month distribution**".
- **Whitepaper v1.2 §5.3 rationale (p. 7):** "Community & Airdrops (20%) uses a
  **snapshot-based linear emission** … **This is distinct from milestone-based grants
  (used only for Dev Grants)**." The label collision with Dev Grants is explicitly
  disambiguated.
- **Tokenomics v1.2 §2.2 (p. 3) and §6 summary (p. 7):** unchanged — "snapshot-based
  linear emission" / "Snapshot-based; linear daily emission".

### #3 — 6-month subsidy reduction range (Minor) ✅

- **Whitepaper v1.2 §6 scenario table (p. 9):** "Partial subsidy (6 mo.) → ~**30–45%**
  reduction".
- **Tokenomics v1.2 §5.3 scenario table (p. 7):** "6-month CNPY subsidy → ~**30–45%**
  reduction in early CLIQ emission". Tokenomics moved from `33–45%` to match the
  whitepaper.

### #4 — Buyback default (Minor) ✅

The Tokenomics doc, which previously never named the default, now states it in three places:

- **§1 Overview table (p. 2):** "15% of all protocol fees; **default action is burn**;
  DAO may vote quarterly to distribute to locked stakers".
- **§3.2 fee distribution table (p. 5):** "Open-market CLIQ purchase; **default action
  is Burn**. DAO may vote quarterly to distribute to locked CLIQ stakers instead of
  burning."
- **§4.1 Buyback Engine (p. 5):** "The **default action** for purchased CLIQ is to
  **burn** it, reducing circulating supply permanently."

Whitepaper v1.2 §8.1 (p. 9) is unchanged: Buyback Mechanism default = "Burn".

### Community & Airdrops genesis-vesting gap — closed 2026-06-18

The Community & Airdrops bucket carries `cliffMonths: 0, vestMonths: 0` in
`plugin/go/canoliq/genesis.testnet.json`, which means the on-chain vesting mechanism
does *not* enforce the 12-month linear emission described by both v1.2 docs (see the
Tokenomics §6 summary on p. 7). That schedule must therefore be enforced off-chain by
whatever address holds the bucket.

**Dev confirmation (2026-06-18):** the address now populated for bucket #3
(`7d941def…e478` in `genesis.testnet.json`) is a controlled distributor that enforces
the 12-month linear emission. The on-chain `0/0` is therefore intentional and not a
doc/code conflict.

No further action required for the v1.1 ↔ v1.2 reconciliation. This whole report
can be considered closed.

---

## v1.2 → code alignment closure (2026-06-22)

After the v1.1 ↔ v1.2 doc reconciliation closed above, a follow-up audit compared
**v1.2 against the implementation** (`docs/canoliq-v1_2-implementation-plan.md`).
That audit found ~37 of 39 verifiable spec points already implemented and two
genuine docs-vs-code gaps:

1. **§9.4 Concentration Risk** — spec said "self-impose a TVL cap of 33% of total
   Canopy network stake"; code shipped an absolute `tvl_cap_ucnpy` parameter
   (T3 / commit `648e7a38`).
2. **§7 Restaking Optimization** — spec described a multi-committee allocation
   engine; code had no multi-committee surface at all (single-committee
   `ValidatorRegistry` only).

Both gaps are now closed on branch `canoliq-spec-alignment`.

### §9.4 percentage TVL cap — closed

| Aspect | Status |
|---|---|
| Spec phrasing | "self-impose a TVL cap of 33% of total Canopy network stake pending ecosystem maturation and governance approval to lift this cap" |
| Implementation | Phase B commits `2cf750a0` (code) + `272cf5f3` (docs) |
| Plugin reach | New `KeyForSupply()` + `readCanopyTotalStake()` (Phase A foundation, commit `ee23a092`) read `lib.Supply.staked` live |
| Default | `TvlCapBps = 3300` (= 33%); `0` = uncapped; governance-tunable |
| Safety | Fail-closed when `Supply` is unreadable or `staked == 0` — `ErrCanopyStakeUnavailable` rejects the deposit rather than silently bypass the cap |
| Surface | `/v1/health` exposes `tvlCapBps`, `canopyTotalStake`, `tvlCapUcnpyEffective`, `tvlUtilizationBps` |

The cap is now live policy enforcement at every deposit, computed against the
current Canopy total stake, exactly as §9.4 calls for.

### §7 Restaking — closed in **policy + observability scope**

| Aspect | Status |
|---|---|
| Spec phrasing | "canoLiq will implement a restaking optimization engine to... dynamically allocate canoLiq's stake to maximize aggregate APY... [under] governance-controlled allocation policy, including minimum and maximum stake per committee." |
| Implementation | Phase C commits `ef44cd0d` (KeyForValidator + Validator proto) + `b0097d32` (engine) + `7336f1e5` + `b354bca6` (docs) |
| What's live | Governance-declared `RestakingPolicy` (target weight + min/max per committee); per-committee exposure observation via `lib.Validator.committees[]`; drift + under-min + over-max + `policyCompliant` flag; `/v1/restaking` surface |
| What's deferred | Active rebalancing (atomically re-routing pool delegations between operators). Requires a delegation-routing primitive not yet defined in the codebase; §11 Roadmap does not list it as a launch deliverable. Operators act on drift signals manually for now (e.g. ejecting underperforming operators via `ACTION_VALIDATOR_EJECT`). |

The spec's *policy mechanism* — declare per-committee targets / min / max, surface
compliance — is implemented faithfully. The *automation* side (the protocol itself
re-routing in response to drift) is documented as a deferred future workstream in
`docs/canoliq-site/docs/advanced/restaking.mdx`, with the rationale visible to
operators reading `/v1/restaking`'s drift signals.

### Related cleanups landed under the same branch

The spec-alignment work also took the opportunity to land three small
docs/test-hygiene fixes the readiness doc had previously listed as "out of scope":

- **Stale insurance README narration** (`plugin/go/canoliq/README.md` — commit
  `deb6ae56`). `default 1500 = 15% of treasury slice` → `default 500 = 5%`,
  matching `config.go:252` and Whitepaper §9.2.
- **`phase2_test.go:208` copylocks vet warning** (commit `f140fe1a`).
  `proto.Clone` replaces a direct struct copy that triggered `go vet`'s
  copylocks check.
- **Stuck-redemption alert** (commit `c203e4b4`). Fourth T6 push-alert
  condition: fires when the count of mature-but-unclaimed redemptions
  exceeds a governance threshold (default 10). Not directly mandated by
  any v1.2 section but rounds out the T6 alert surface; out-of-scope flag
  on the readiness doc removed.

### Final state

`canoliq-spec-alignment` ships **10 commits** closing both v1.2 docs-vs-code gaps
plus three independent cleanups. All four phases (D, A, B, C) are committed; the
implementation-plan document (`docs/canoliq-v1_2-implementation-plan.md`) records
the per-phase commit hashes. The v1.2 whitepaper and tokenomics text are
unchanged — the *code* now matches what the docs promised.

This appendix closes both audits: v1.1 ↔ v1.2 (above) and v1.2 ↔ code (here).
