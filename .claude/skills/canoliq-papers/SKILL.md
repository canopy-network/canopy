---
name: canoliq-papers
description: >-
  Canonical specs for the canoLiq protocol from its official v1.2 papers — the
  CPLQ Tokenomics document and the Liquid Staking Whitepaper. Use whenever the
  user references "the canoliq papers", asks about canoLiq tokenomics, CPLQ or
  cCNPY tokens, token distribution/vesting, the protocol fee model, buyback,
  vote-escrow, governance thresholds, Canopy subsidies, restaking, validators,
  risk framework, or autonomy graduation. Reference PROACTIVELY whenever there
  is any doubt about a canoLiq spec, number, or design decision instead of
  guessing — the numbers here are authoritative.
---

# canoLiq Papers (v1.2, May 2025)

Authoritative reference for canoLiq protocol specs. Two source documents:

- **CPLQ Tokenomics v1.2** — `docs/canoLiq_Tokenomics_v1.2.pdf` (9 pages)
- **Liquid Staking Whitepaper v1.2** — `docs/canoLiq_Whitepaper_v1.2.pdf` (12 pages)

> **Ticker note:** the governance token's ticker is **CPLQ**. The v1.2 source
> PDFs predate the rename and still print the *old* ticker **CLIQ** — they are
> binary and pending regeneration (see `docs/canoliq-pdf-ticker-rename-checklist.md`).
> `CLIQ` and `CPLQ` refer to the same token; this skill uses the current `CPLQ`
> to match the codebase. When quoting a PDF verbatim, expect to see `CLIQ`.

When the user says "based on the canoliq papers…", or any canoLiq spec/number is
in question, treat the figures below as canonical. The two papers are
consistent with each other; cite the document + section when it matters. For
deeper narrative beyond the figures here, read:

- `reference/tokenomics.md` — full section-by-section tokenomics detail
- `reference/whitepaper.md` — full section-by-section whitepaper detail

## What canoLiq is

Liquid staking protocol built as a **Nested Chain on Canopy Network**. Users
deposit **CNPY** and receive **cCNPY** (transferable, yield-bearing receipt
token) while the underlying stake participates in Canopy committees and earns
block rewards. **CPLQ** is the separate governance + value-capture token.

**Three tokens — do not conflate:**
- **CNPY** — Canopy Network's native token (deposited by users; bonded by validators).
- **cCNPY** — canoLiq's yield-bearing liquid staking receipt. Minted 1:1 at the
  current exchange rate on deposit; appreciates via exchange-rate growth
  (auto-compounding). Fully transferable / DeFi-composable.
- **CPLQ** — governance + value-capture token. **Fixed supply 100,000,000; no
  minting after genesis.** NOT a high-emission farming token.

## Core numbers (most-cited)

| Spec | Value |
|---|---|
| CPLQ total supply | 100,000,000 (fixed, no post-genesis minting) |
| Protocol fee rate | 12% default; governance-controlled bounds **5%–20%** |
| Fee → cCNPY holders | 40% of fee = 4.8% of rewards |
| Fee → DAO Treasury | 30% of fee = 3.6% of rewards |
| Fee → Validators & Infra | 15% of fee = 1.8% of rewards |
| Fee → CPLQ Buyback | 15% of fee = 1.8% of rewards (**default: burn**) |
| Effective cCNPY yield | 88% + 4.8% = **92.8% of rewards received** |
| Canopy committee reward split | **70/10/10/10** (producer / root delegate / nested validator / nested delegate) |
| canoLiq fee distribution split | **40/30/15/15** (cCNPY / treasury / validators / buyback) |
| Auto-subsidization threshold | committee holds **≥33%** of total network restake |
| TVL self-cap | 33% of total Canopy network stake (pending governance to lift) |

## Token distribution (100M CPLQ)

| Allocation | Share | Tokens | Vesting (start = TGE / mainnet launch) |
|---|---|---|---|
| Validators & Infrastructure | 22% | 22M | 12-mo cliff, then linear monthly → **3 years total** |
| Community & Airdrops | 20% | 20M | No cliff; snapshot-based linear daily over **12 months** |
| DAO Treasury | 15% | 15M | Perpetual; DAO-governed release (time-locked multisig) |
| Liquidity Incentives | 15% | 15M | No cliff; DAO-controlled emission over **24 months** |
| Founders & Core Team | 12% | 12M | 12-mo cliff, then linear monthly → **4 years total** |
| Strategic Partners | 10% | 10M | 6-mo cliff, then linear monthly → **18 months total** |
| Dev Grants & Ecosystem | 6% | 6M | Milestone-gated tranches (per grant) |

Cliff is *included within* total vesting (e.g. Validators = 12-mo cliff + 24-mo
linear = 36 months). Community/Liquidity have no cliff; Dev Grants is the only
milestone-gated bucket (distinct from snapshot-based Community).

## Fee model

`Protocol Fee = 12% × (staking rewards received by canoLiq from committee participation)`

Rewards flow: gross reward X → Canopy distributes per 70/10/10/10 → canoLiq
receives its stake-weighted share **R** → 12% fee = 0.12×R → fee split 40/30/15/15
→ remaining 88% of R flows to cCNPY holders via exchange-rate appreciation.
Fee is applied to rewards *received* (after Canopy's on-chain distribution);
Canopy applies no protocol-level DAO tax on top before distribution. Fee rate
change requires a standard governance proposal with a 48-hour timelock.

## Value accrual & vote-escrow

- **Buyback engine:** 15% of all protocol fees buy CPLQ on the open market.
  Default action is **burn**; the DAO may vote *quarterly* to redirect buyback
  CPLQ to locked stakers instead of burning.
- **Vote-escrow lock schedule** (longer lock = more voting weight + bigger reward share):

| Lock | Voting multiplier | Reward share boost |
|---|---|---|
| No lock | 1× | base |
| 3 months | 1.5× | +10% |
| 6 months | 2× | +25% |
| 12 months | 3× | +50% |
| 24 months | 4× | +75% |

## Governance thresholds

| Action | Quorum | Approval | Timelock |
|---|---|---|---|
| Fee rate adjustment | 5% | 51% | 48h |
| Treasury spend (small) | 5% | 51% | 48h |
| Treasury spend (large, >1M CPLQ equiv) | 10% | 67% | 7 days |
| Emergency security action | 8% | 67% | 24h (fast-track) |
| Validator ejection | 5% | 51% | 48h |
| Protocol upgrade | 10% | 67% | 7 days |
| Autonomy graduation vote | 15% | 75% | 14 days |

DAO Treasury is a **3-of-5 multisig**: 48h timelock standard spends, 7-day for
large spends (>1M CPLQ equiv).

## Validators

Run by **professional node operators, not canoLiq** (canoLiq is a protocol, not
an infra company). Whitelisted registry. Operators register with canoLiq DAO,
bond their **own CNPY** as collateral on the Canopy Root Chain (list canoLiq's
committee ID in their MessageStake), run dedicated infra, and execute NestBFT.
On slashing, the operator's *own* bonded CNPY is slashed — not users' deposited
CNPY directly; the insurance fund compensates shortfalls. Operators earn 15% of
protocol fees and can be ejected by CPLQ vote. **At launch: 5–10 established
Canopy validator operators** seed the initial committee.

## Canopy subsidy strategy (bootstrap)

CPLQ liquidity emission is back-stopped by **CNPY subsidies from Canopy DAO**,
used first so CPLQ emits only as needed — avoiding early sell pressure from
mercenary farmers. Auto-subsidization triggers when canoLiq's committee holds
≥33% of total network restake. canoLiq will also submit a **manual subsidy
proposal** to Canopy DAO within 30 days of mainnet, tranches tied to TVL
milestones (**$500K / $2M / $10M**). Subsidy outcomes on the 15M liquidity
bucket: no subsidy → full 15M front-loaded (runway exhausted ~month 18);
6-mo subsidy → ~8–10M over 24mo (30–45% reduction); 12-mo → ~5–8M (50–60% reduction).

## Risk framework (key mitigations)

- **Slashing:** insurance fund = 5% of DAO treasury, seeded progressively from
  fee income over first 12 months, **target 5% of peak TVL**. Validators bond
  own CNPY as first line of defense. Canopy `MaxSlashPerCommittee` caps slashes.
- **Smart contract:** two independent audits + formal verification of core
  accounting invariants + public bug bounty at mainnet.
- **Concentration:** self-imposed TVL cap of 33% of total Canopy stake.
- **Oracle/price:** cCNPY/CNPY exchange rate computed entirely on-chain — no
  external price oracle, so no oracle-manipulation risk on the core yield path.
- **Governance centralization:** a single entity >33% of circulating CPLQ could
  sway most votes; quadratic/conviction voting may be explored post-launch.

## Autonomy graduation (Nested Chain → sovereign chain)

Thresholds: TVL > $50M USD in CNPY; > 30 active independent validators; > 15%
average governance turnout; > 10,000 tx/day sustained 30 days; > 12 months
treasury runway. Requires cross-chain governance coordination (Canopy DAO +
canoLiq DAO).
