# canoLiq v1.1 — Whitepaper vs. Tokenomics Discrepancy Report

**Date:** 2026-05-22
**Documents compared:**
- `canoLiq_Whitepaper_v1.1.pdf` (Technical Whitepaper, Version 1.1, May 2025)
- `canoLiq_Tokenomics_v1.1.pdf` (CLIQ Token Design & Protocol Economics, v1.1, May 2025)

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

## Recommended actions

1. **#1 (validator cliff)** requires a decision — the documents contradict each other and the
   Tokenomics doc argues against the whitepaper's value. Pick 6 or 12 months and align both
   documents (and verify the codebase genesis vesting matches).
2. **#2** is almost certainly a labeling slip in the whitepaper; update its §5.2 table to
   "Snapshot-based linear; 12-month distribution".
3. **#3 / #4** are small wording/number alignments.
