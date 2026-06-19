# canoLiq v1.1 — Whitepaper vs. Tokenomics Discrepancy Report

**Date:** 2026-05-22
**Documents compared:**
- `canoLiq_Whitepaper_v1.1.pdf` (Technical Whitepaper, Version 1.1, May 2025)
- `canoLiq_Tokenomics_v1.1.pdf` (CLIQ Token Design & Protocol Economics, v1.1, May 2025)

> **Status (verified 2026-06-18): All four doc-vs-doc discrepancies are resolved in v1.2.**
> See [§ Resolution in v1.2](#resolution-in-v12-verified-2026-06-18) at the end of this
> document for the per-item verification against `canoLiq_Whitepaper_v1.2.pdf` and
> `canoLiq_Tokenomics_v1.2.pdf`. The Community & Airdrops genesis-vesting gap noted in
> [§ Additional code/doc gap](#additional-code-doc-gap-community--airdrops) is *not* a
> docs issue and remains open pending dev-team confirmation that the bucket recipient
> is a controlled distributor.

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

### Still open (not a doc-vs-doc issue)

The Community & Airdrops genesis-vesting gap (see
[§ Additional code/doc gap](#additional-code-doc-gap-community--airdrops)) is unchanged
in v1.2 and cannot be fixed by docs alone. The v1.2 Tokenomics §6 summary still lists
the bucket as Cliff "None" / Duration "12 months / Snapshot-based; linear daily emission",
which is internally consistent with an off-chain distributor enforcing the schedule.
The on-chain genesis vesting (`cliffMonths: 0, vestMonths: 0`) is fine **only if** the
bucket recipient is a controlled distributor address. Confirm with the dev team that
this is the case for the testnet genesis address now populated for bucket #3.
