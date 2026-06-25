# CLIQ Tokenomics v1.2 — Full Detail

Source: `docs/canoLiq_Tokenomics_v1.2.pdf` (9 pages, May 2025).
Subtitle: "CLIQ Token Design & Protocol Economics". Total Supply: **100,000,000
CLIQ — Fixed Supply, No Additional Minting.**

## 1. Token Overview & Design Philosophy

CLIQ is the governance and value-capture token of canoLiq. Deliberately minimal
in emission complexity: **not** a high-emission farming token meant to subsidize
yield. It is a governance asset that captures protocol value through fee-funded
buybacks and aligns long-term holders with protocol success.

| Property | Specification |
|---|---|
| Token Name | CLIQ |
| Total Supply | 100,000,000 (fixed; no minting after genesis) |
| Protocol | canoLiq on Canopy Network |
| Primary Utility | Governance voting + buyback value capture |
| Governance Model | Vote-escrowed lock (longer lock = higher voting weight) |
| Buyback Mechanism | 15% of all protocol fees; default action is burn; DAO may vote quarterly to distribute to locked stakers |
| Secondary Token | cCNPY (yield-bearing liquid staking receipt; distinct from CLIQ) |

## 2. Token Distribution

Prioritizes validator alignment (22%) and community ownership (20% + 15% treasury
+ 6% dev grants = 41%). Team and investor allocations are intentionally below the
community total to reflect a protocol-first governance structure.

| Allocation | Share | Tokens |
|---|---|---|
| Validators & Infra | 22% | 22M |
| Community & Airdrops | 20% | 20M |
| DAO Treasury | 15% | 15M |
| Liquidity Incentives | 15% | 15M |
| Founders & Team | 12% | 12M |
| Strategic Partners | 10% | 10M |
| Dev Grants & Ecosystem | 6% | 6M |

### 2.1 Validators & Infrastructure — 22M CLIQ
Largest allocation; reflects canoLiq's dependency on validator quality.
Validators are professional node operators who run dedicated server infra, bond
their own CNPY as collateral, and accept direct slashing exposure on behalf of
canoLiq users.
- **Vesting:** 3-year linear with 12-month cliff.
- **12-month cliff rationale:** validators run production infra from day one,
  but CLIQ should not unlock until the protocol survives its first full year of
  mainnet (highest-risk period). A 6-month cliff is insufficient.
- **Eligibility:** professional node operators registered with canoLiq, running
  canoLiq committee nodes with verifiable on-chain uptime metrics.
- **Performance-gating:** underperforming validators (below min uptime or
  subject to slashing) may have unvested allocation suspended or forfeited per
  DAO vote.

### 2.2 Community & Airdrops — 20M CLIQ
Distributed to CNPY stakers, early canoLiq depositors, and Canopy ecosystem
participants to bootstrap a broad, decentralized governance base. Uses
snapshot-based linear emission — distinct from Dev Grants' milestone tranches.
- **Snapshot-based:** eligibility determined at protocol launch from Canopy
  staking history and early canoLiq deposit activity.
- **Distribution:** 12-month linear daily emission per qualifying wallet (not
  milestone-gated).
- **Anti-sybil:** per-address caps and minimum CNPY stake requirement to qualify.

### 2.3 DAO Treasury — 15M CLIQ
The protocol's strategic reserve. Governed entirely by CLIQ holders via on-chain
proposals with timelock enforcement.
- **Time-locked multisig:** 3-of-5 signer multisig with 48-hour timelock for
  standard spends, 7-day for large spends (>1M CLIQ equivalent).
- **Authorized uses:** security audits, ecosystem grants, insurance fund
  seeding, cross-chain integrations.
- **Annual budget:** treasury spend above a governance threshold requires full
  DAO vote.

### 2.4 Liquidity Incentives — 15M CLIQ
Incentivizes DEX liquidity for cCNPY/CNPY and CLIQ pairs, reducing slippage.
- **Emission period:** 24 months post-mainnet.
- **Subsidy substitution:** CNPY subsidies from Canopy DAO (see §5) used first;
  CLIQ emissions activate only as needed — expected to reduce actual CLIQ
  emission from this bucket by **30–60%**.
- **DAO-controlled emission rate:** can be accelerated or slowed by governance.

### 2.5 Founders & Core Team — 12M CLIQ
Conservative vesting: 4-year linear with 12-month cliff. No early unlocks.
Subject to the same on-chain vesting contracts as all other allocations.

### 2.6 Strategic Partners — 10M CLIQ
Reserved for ecosystem partnerships: Canopy ecosystem projects, CEX listings,
institutional delegators, protocol integrations.
- 18-month linear vesting with 6-month cliff.
- Partner criteria and allocations subject to DAO approval above 500K CLIQ per
  partner.

### 2.7 Developer Grants & Ecosystem — 6M CLIQ
Milestone-based grants to teams building on/integrating with canoLiq (e.g. DeFi
protocols using cCNPY as collateral, tooling, analytics). Milestone-gated —
grants released in tranches upon delivery of agreed deliverables.

## 3. Protocol Fee Model

canoLiq generates revenue by applying a protocol fee to the staking rewards it
receives from Canopy committee participation. This is the **sole** source of
protocol revenue and drives all value accrual to CLIQ holders.

### 3.1 Fee Calculation
```
Protocol Fee = 12% × (Staking Rewards Received by canoLiq from Committee Participation)
Effective user yield = 88% × Rewards + 40% × 12% × Rewards = 92.8% of Rewards Received
```

### 3.2 Fee Distribution
| Recipient | % of Fee | Effective % of Rewards | Mechanism |
|---|---|---|---|
| cCNPY Holders | 40% | 4.8% | Reinvested; increases exchange rate for all cCNPY holders |
| canoLiq DAO Treasury | 30% | 3.6% | Accumulated in treasury pool; DAO-controlled deployment |
| Validators & Infra | 15% | 1.8% | Paid to active committee validators; uptime-weighted |
| CLIQ Buyback | 15% | 1.8% | Open-market CLIQ purchase; **default action is Burn**. DAO may vote quarterly to distribute to locked CLIQ stakers instead. |

### 3.3 Fee Rate Governance
12% default rate is governance-controlled. CLIQ holders may vote to adjust within
protocol-defined bounds of **5% to 20%**. Changes require a standard governance
proposal with a 48-hour timelock.

## 4. CLIQ Value Accrual Mechanics

### 4.1 Buyback Engine
15% of all protocol fees buy CLIQ on the open market. Default action for
purchased CLIQ is to **burn** it (permanently reduces circulating supply).
Quarterly, CLIQ holders may vote to redirect buyback proceeds to locked CLIQ
stakers (as governance yield) instead of burning. Creates persistent buy-side
pressure proportional to protocol TVL regardless of distribution method.

Illustrative annual buyback (8% assumed APY; applies to net rewards received
after Canopy's on-chain distribution):

| TVL (CNPY) | Annual Fees to Protocol | Annual CLIQ Buyback (15%) |
|---|---|---|
| $5M | $48,000 | $7,200 |
| $25M | $240,000 | $36,000 |
| $100M | $960,000 | $144,000 |
| $500M | $4,800,000 | $720,000 |

### 4.2 Vote-Escrow Boost
| Lock Duration | Voting Multiplier | Reward Share Boost |
|---|---|---|
| No lock | 1× | Base rate |
| 3 months | 1.5× | +10% |
| 6 months | 2× | +25% |
| 12 months | 3× | +50% |
| 24 months | 4× | +75% |

## 5. Canopy Subsidy Integration

### 5.1 The Bootstrap Dilemma
Every new DeFi protocol needs liquidity to be useful but needs to be useful to
attract liquidity. Standard solution (liquidity mining) pays early users in the
protocol's own token, flooding the market with freshly minted CLIQ before the
protocol has proven value. canoLiq instead uses **CNPY — subsidized from Canopy
DAO — as the bootstrap incentive**. Early users are still rewarded, but in CNPY,
not newly printed CLIQ. The 15M CLIQ liquidity allocation then stretches over
24+ months instead of front-loading the first few months.

> **Why this matters for CLIQ holders:** every CLIQ NOT emitted in months 1–6 is
> CLIQ not sold by mercenary farmers. Subsidy substitution is the primary
> mechanism by which canoLiq's liquidity incentive budget preserves CLIQ value —
> central to the tokenomics working as designed, not a secondary strategy.

### 5.2 Auto-Subsidization Qualification
Canopy automatically distributes newly minted CNPY to committees that hold
**≥33% of total network restake** (the auto-subsidization threshold). canoLiq's
goal is to qualify from day one by coordinating with launch validators to reach
that threshold at mainnet, triggering CNPY emission to the committee pool from
block one with zero additional CLIQ required.

### 5.3 Canopy DAO Manual Subsidy Proposal
In parallel, canoLiq will submit a formal manual subsidy proposal to Canopy DAO
within 30 days of mainnet launch:
- **Proposed duration:** 6–12 months, or until TVL is self-sustaining.
- **Proposed tranches:** tied to TVL milestones — **$500K, $2M, and $10M** in
  CNPY deposits.
- **Commitment:** monthly on-chain reporting of TVL, subsidy utilization, and
  protocol fee revenue to Canopy DAO as a condition of ongoing subsidy.

| Scenario | Effective CLIQ Emission (Liquidity Bucket, 24 mo.) | Impact on CLIQ Supply |
|---|---|---|
| No subsidy approved | 15M CLIQ (full allocation, front-loaded) | Heavy early sell pressure; runway exhausted by month 18 |
| 6-month CNPY subsidy | ~8–10M CLIQ over 24 months | ~30–45% reduction in early emission; runway extended |
| 12-month CNPY subsidy | ~5–8M CLIQ over 24 months | ~50–60% reduction; pool active well beyond year 2 |

## 6. Vesting & Lock Schedule Summary

| Allocation | Start | Cliff | Duration | Schedule |
|---|---|---|---|---|
| Validators & Infra | TGE | 12 months | 3 years | Linear monthly after cliff |
| Community & Airdrops | TGE | None | 12 months | Snapshot-based; linear daily emission |
| DAO Treasury | TGE | N/A | Perpetual | DAO-governed release |
| Liquidity Incentives | TGE | None | 24 months | DAO-controlled emission rate |
| Founders & Team | TGE | 12 months | 4 years | Linear monthly after cliff |
| Strategic Partners | TGE | 6 months | 18 months | Linear monthly after cliff |
| Dev Grants | Per milestone | Per grant | Per grant | Milestone-gated tranches |

TGE = Token Generation Event (mainnet launch day). Cliff is included within the
total vesting period — e.g. Validators: 12-month cliff then 24 months linear =
36 months total.

## 7. Governance Framework

CLIQ is the sole governance token. All protocol parameter changes, treasury
actions, and strategic decisions require CLIQ holder votes.

| Governance Action | Required Quorum | Approval Threshold | Timelock |
|---|---|---|---|
| Fee rate adjustment | 5% of circulating supply | Simple majority (51%) | 48 hours |
| Treasury spend (small) | 5% | 51% | 48 hours |
| Treasury spend (large, >1M CLIQ equiv.) | 10% | 67% | 7 days |
| Emergency security action | 8% | 67% | 24 hours (fast-track) |
| Validator ejection | 5% | 51% | 48 hours |
| Protocol upgrade | 10% | 67% | 7 days |
| Autonomy graduation vote | 15% | 75% | 14 days |

## 8. Token Risk Considerations

- **Smart contract risk:** bugs could cause loss of deposited CNPY or incorrect
  fee accounting. Two independent audits mitigate but do not eliminate.
- **Slashing risk:** canoLiq's CNPY stake is exposed to committee slashing;
  cCNPY and CLIQ holders bear this proportionally. Insurance fund of 5% of DAO
  treasury — accumulated progressively from fee income over first 12 months,
  target 5% of peak TVL — provides protection. Validators also bond their own
  CNPY as first line of defense.
- **CLIQ liquidity risk:** early CLIQ markets may be illiquid; large sellers may
  face slippage. The buyback program (default: burn) gives structural buy-side
  support but no price guarantees.
- **Governance centralization:** a single entity accumulating >33% of circulating
  CLIQ can influence most votes. Quadratic / conviction voting may be explored
  post-launch.
- **Canopy dependency:** canoLiq economics are deeply tied to Canopy's block
  reward schedule, committee rules, and subsidy decisions.

## 9. Disclaimer
Informational only; not financial/legal/investment advice. Tokenomics subject to
change prior to and following mainnet launch. CLIQ and cCNPY involve significant
risk.
