# canoLiq Liquid Staking Whitepaper v1.2 — Full Detail

Source: `docs/canoLiq_Whitepaper_v1.2.pdf` (12 pages, May 2025).
Subtitle: "Liquid Staking Protocol on Canopy Network".

## Abstract
canoLiq is a liquid staking protocol built as a **Nested Chain on Canopy
Network**. Users deposit CNPY and receive **cCNPY** — a transferable,
yield-bearing receipt token — while the underlying stake participates in Canopy
committees and earns block rewards. A secondary governance and value-capture
token (**CLIQ**) aligns long-term incentives across users, validators, and the
protocol treasury. Path to full chain sovereignty within the Canopy incubator
model.

## 1. Introduction & Problem Statement
Staking in PoS locks capital: participants sacrifice liquidity for security. In
Canopy, CNPY must be bonded in committees to earn block rewards — creating an
opportunity cost that deters DeFi-oriented users who need composable, liquid
positions. canoLiq abstracts committee staking into a single fungible token
(cCNPY), and leverages Canopy's restaking model — the same bonded CNPY can
simultaneously secure multiple Nested Chains — to maximize yield without extra
capital.

**Key value propositions:** Liquidity (cCNPY freely transferable & composable
while earning yield); Capital Efficiency (restaking compounds rewards on the
same collateral); Governance (CLIQ holders direct parameters, treasury,
validator selection); Canopy-Native (launches with shared security, maturing
toward full sovereignty).

### 1.1 How canoLiq Works: End-to-End
| Step | Who Acts | What Happens |
|---|---|---|
| 1. Deposit | User | Sends CNPY to canoLiq contract; contract mints cCNPY at current exchange rate and returns it. User holds a liquid, yield-bearing position. |
| 2. Delegation | canoLiq Protocol | Pooled CNPY delegated on-chain to whitelisted professional validator operators registered with canoLiq. Operators have bonded their own CNPY as collateral on the Canopy Root Chain and listed canoLiq's committee ID in their stake config. |
| 3. Node Operation | Validator Operators | Each whitelisted validator runs a full instance of the canoLiq Nested Chain software on their own infra, executes NestBFT consensus, produces blocks, and submits Certificate Result Transactions to the Canopy Root Chain each block. |
| 4. Reward Accrual | Canopy Root Chain | Every Root Chain block, new CNPY is minted and sent to the canoLiq committee fund pool (auto-subsidization). Validators divide those rewards per the 70/10/10/10 default split. canoLiq's share accrues in the protocol contract. |
| 5. Fee & Distribution | canoLiq Protocol | Applies its 12% protocol fee to received rewards. 88% flows directly to cCNPY holders via exchange-rate appreciation. The 12% fee is split 40% cCNPY / 30% DAO treasury / 15% validators / 15% CLIQ buyback. |
| 6. Redemption | User | Returns cCNPY to the contract; contract calculates current exchange rate (original CNPY + accrued yield) and initiates unstaking from the committee. Small redemptions served from a liquidity buffer instantly; large redemptions follow Canopy's standard unstaking cooldown. |

### 1.2 Who Runs the Validators?
Professional node operators, **not canoLiq itself**. canoLiq is a protocol, not
an infrastructure company — it does not run servers. It maintains a whitelisted
registry of professional validator operators (e.g. institutional staking
providers already operating on Canopy). These operators:
- Register with canoLiq DAO by bonding their own CNPY collateral on the Canopy
  Root Chain, listing the canoLiq committee ID in their MessageStake transaction.
- Run dedicated server infra with redundant signing keys, uptime monitoring,
  and slashing-protection tooling.
- Accept slashing risk: if they double-sign or go offline beyond the MaxNonSign
  threshold, **their own bonded CNPY is slashed — not users' deposited CNPY
  directly**. The insurance fund compensates users for any shortfall.
- Earn 15% of canoLiq protocol fees as compensation for operational costs and
  slashing risk.
- Are subject to ongoing performance review and can be ejected from the validator
  set by CLIQ governance vote.

At launch, canoLiq partners with **5–10 established Canopy validator operators**
to seed the initial committee. Validator set expansion is governed by CLIQ
holders.

## 2. Canopy Network: Operational Context
Canopy is a recursive blockchain framework: a Root Chain (CNPY) provides shared
security to Nested Chains via opt-in committees of restaking validators.

### 2.1 NestBFT Consensus
Canopy uses **NestBFT**, a BFT consensus mechanism fused with **Proof-of-Age**.
Validators organize into **Committees** — subsets that execute BFT for each
Nested Chain independently. The elected committee leader submits a Certificate
Result Transaction to the Root Chain each block, recording rewards and slashing
decisions.

### 2.2 Economic Model
Each Root Chain block, new CNPY is minted and distributed to auto-subsidized
committees. A committee qualifies for auto-subsidization when its share of total
network restake meets or exceeds the governance-controlled subsidization
threshold (**currently 33%**). Creates natural TVL competition and incentivizes
CNPY holders to delegate to growing Nested Chains.

### 2.3 Reward Distribution (Default 70/10/10/10)
| Recipient | Share | Description |
|---|---|---|
| Block Producer | 70% | Root Chain validator who produced the Nested Chain block |
| Root Chain Delegate | 10% | Randomly selected by stake-weight among delegates for this chain |
| Nested Chain Validator | 10% | Stake-weighted random selection on the Nested Chain |
| Nested Chain Delegate | 10% | Stake-weighted random selection on the Nested Chain |

canoLiq operates **on top of** these mechanics — it does not modify them. Its fee
model captures value from rewards received after Canopy's on-chain distribution.

## 3. Protocol Architecture

### 3.1 How canoLiq Sits Inside Canopy
Deployed as a Nested Chain on Canopy. As a committee member it pools CNPY
deposits and participates in one or more committees, allowing canoLiq to: pool
stake from thousands of users into a single large validator position (lowering
the barrier to earning rewards); optimize restaking (dynamically allocate stake
across committees to maximize yield); abstract complexity (users hold cCNPY with
no need to manage committee membership, cooldowns, or validator ops).

### 3.2 The cCNPY Token
| Property | Detail |
|---|---|
| Minting | 1:1 at current exchange rate on CNPY deposit |
| Redemption | At current exchange rate; subject to committee unstake cooldown |
| Yield Mechanism | Exchange-rate appreciation (auto-compounding net rewards) |
| Transferability | Fully transferable; DeFi-composable |
| Slashing Risk | Proportional exposure to committee slashing events |

### 3.3 Rewards Flow (Step by Step)
Given a gross reward of X CNPY generated by canoLiq's stake exposure in a period:
1. Canopy distributes rewards to the committee pool per the 70/10/10/10 split.
2. canoLiq receives its proportional share (**R**) based on its stake weight in
   the committee.
3. canoLiq applies a 12% protocol fee on R. Fee amount = 0.12 × R.
4. The fee is distributed per the canonical 40/30/15/15 split (see §4).
5. The remaining 88% of R flows to cCNPY holders via exchange-rate appreciation.

**IMPORTANT:** Canopy does not apply a protocol-level DAO tax on top of rewards
before distribution. All economics flow through committee reward pools (§2.2).
The canoLiq 12% fee is applied to rewards received by the protocol.

## 4. Protocol Fee Model

### 4.1 Fee Rate
Default protocol fee is **12% of net rewards** received by canoLiq from committee
participation. Governance-controlled by CLIQ holders; adjustable within bounds
set by the protocol constitution (**proposed range: 5%–20%**).

### 4.2 Fee Distribution
| Recipient | Share of Fee | Amount (of 12%) | Purpose |
|---|---|---|---|
| cCNPY Holders | 40% | 4.8% of R | Reinvested into exchange rate; bonus yield on top of base staking APY |
| canoLiq DAO Treasury | 30% | 3.6% of R | Protocol operations, grants, security, future development |
| Validators & Infra | 15% | 1.8% of R | Operational incentives, uptime bonuses, slashing-risk compensation |
| CLIQ Buyback & Burn | 15% | 1.8% of R | Open-market CLIQ purchases; default action is burn; DAO may vote quarterly to distribute to locked CLIQ stakers instead |

### 4.3 Effective Yield for cCNPY Holders
cCNPY holders benefit from both the base staking yield (88% of received rewards)
and the fee redistribution component (40% of the 12% fee = additional 4.8% of
received rewards). Aggregate yield passed to the exchange rate is 88% + 4.8% =
**92.8% of received rewards**.

## 5. CLIQ Tokenomics
(Mirrors the standalone Tokenomics document — see `reference/tokenomics.md` for
the authoritative full version.)

### 5.1 Overview
CLIQ is the governance and value-capture token. Fixed supply 100,000,000. Not a
reward emission token used to inflate staker yields — its primary roles are
governance participation and value capture via buybacks.

### 5.2 Distribution
| Allocation | % of Supply | Tokens | Vesting |
|---|---|---|---|
| Validators & Infrastructure | 22% | 22,000,000 | 3-year linear; 12-month cliff |
| Community & Airdrops | 20% | 20,000,000 | Snapshot-based linear; 12-month distribution |
| DAO Treasury (canoLiq) | 15% | 15,000,000 | DAO-controlled; time-locked multisig |
| Liquidity Incentives | 15% | 15,000,000 | Emitted over 24 months; governed by DAO |
| Founders & Core Team | 12% | 12,000,000 | 4-year vesting; 12-month cliff |
| Strategic Partners | 10% | 10,000,000 | 18-month linear; 6-month cliff |
| Developer Grants & Ecosystem | 6% | 6,000,000 | Milestone-based grants |

### 5.3 Design Rationale
Validators get the largest single allocation (22%) because security, uptime, and
committee performance depend on validator quality; the 12-month (not 6-month)
cliff keeps them committed through the highest-risk first year. Community (20%)
uses snapshot-based linear emission (distinct from milestone-based Dev Grants).
Liquidity (15%) sized to support DEX liquidity without excessive inflation; CNPY
subsidies (§6) reduce the need to emit it all early. Founders vest over 4 years
(longer than typical 3-year) to align with long-term success. Treasury (15%) in
a time-locked multisig as strategic reserve.

### 5.4 CLIQ Value Accrual
Three mechanisms: (1) **Buyback pressure** — 15% of all protocol fees buy CLIQ on
the open market, default action burn (deflationary), DAO may redirect to locked
stakers quarterly. (2) **Governance premium** — locked CLIQ stakers get boosted
voting power and may receive a share of buyback CLIQ rather than it being burned
(DAO vote required). (3) **Protocol ownership** — CLIQ holders govern the most
critical lever (the fee rate), making CLIQ a claim on future protocol cash flows.

## 6. Canopy Subsidy Strategy

### 6.1 The Bootstrap Problem
New DeFi protocols face a chicken-and-egg liquidity problem. Standard solution
("liquidity mining") pays early users in the native token (CLIQ), flooding the
market with freshly minted tokens and creating sell pressure. canoLiq instead
uses **CNPY — subsidized from Canopy DAO** — as the bootstrap incentive; same
user behavior (deposit early, provide liquidity) with far less inflation pressure
on CLIQ.

> **Concrete example:** Without subsidies, attracting $5M TVL in month 1 might
> need 500,000 CLIQ as farming rewards (0.5% of supply in month 1 alone). With
> CNPY subsidies, Canopy DAO seeds the committee reward pool; early depositors
> earn bonus CNPY yield on top of base staking APY, and canoLiq emits zero
> additional CLIQ for the same TVL outcome. The 15M CLIQ liquidity allocation
> then stretches across 24+ months instead of front-loading.

### 6.2 Auto-Subsidization Mechanics
Every Root Chain block, newly minted CNPY flows automatically to committees that
hold **≥33% of total network restake** (the auto-subsidization threshold).
canoLiq's goal is to qualify from day one by working with launch validators to
reach that stake threshold immediately at mainnet.

### 6.3 Requesting Canopy DAO Manual Subsidies
In addition to auto-subsidization, any party can manually subsidize a committee
via a **MessageSubsidy** transaction. canoLiq plans to submit a formal proposal
to Canopy DAO requesting manual CNPY subsidies during bootstrap:
- **Proposed duration:** 6–12 months, or until TVL milestones are self-sustaining.
- **Proposed tranches:** linked to TVL milestones (e.g. CNPY subsidy tranche 1 at
  $500K TVL, tranche 2 at $2M TVL, tranche 3 at $10M TVL).
- **Transparency:** all subsidy flows on-chain and publicly auditable; monthly
  reporting to Canopy DAO.
- **Alignment:** subsidies requested specifically as bootstrap support, not a
  permanent operating dependency.

| Scenario | CLIQ Emitted (Liquidity Bucket) | Result |
|---|---|---|
| No subsidy | 15M CLIQ over 24 months (front-loaded) | Heavy early sell pressure on CLIQ |
| Partial subsidy (6 mo.) | ~8–10M CLIQ over 24 months | ~30–45% reduction in early CLIQ emission |
| Full subsidy (12 mo.) | ~5–8M CLIQ over 24 months | ~50–60% reduction; extended incentive runway |

## 7. Restaking Optimization
Canopy's restaking model lets the same bonded CNPY simultaneously secure multiple
Nested Chains. canoLiq will implement a restaking optimization engine to:
identify high-yield committees (monitor reward per CNPY staked across active
committees, dynamically allocate to maximize aggregate APY); manage slashing risk
(diversify stake to limit exposure to any single committee's slashing events);
balance liquidity (maintain enough stake in the canoLiq committee itself to
support redemptions without large cooldown delays); governance-controlled (CLIQ
holders vote on restaking allocation policy including min/max stake per committee).

## 8. Governance

### 8.1 Scope of CLIQ Governance
| Parameter | Default | Governance Bounds |
|---|---|---|
| Protocol Fee Rate | 12% | 5%–20% |
| Fee Distribution (40/30/15/15) | As specified | Adjustable per component within ranges |
| Buyback Mechanism | Burn | Burn vs. distribute to locked stakers; DAO vote per quarter |
| Validator Onboarding | Committee vote | Criteria set by DAO |
| Restaking Allocation | Optimization engine | Max/min per committee; DAO-controlled |
| Subsidy Proposals | Canopy DAO submission | canoLiq DAO approves first |
| Treasury Spending | Multisig + timelock | Thresholds set by DAO |

### 8.2 Voting Mechanics
- CLIQ can be staked and time-locked for boosted voting power (longer lock =
  higher multiplier).
- Proposals require a minimum quorum (proposed: 5% of circulating supply).
- Emergency proposals (security-critical) may use a 24-hour fast-track with a
  higher approval threshold (67%).
- All treasury transactions above a governance-defined threshold require a
  timelock delay (proposed: 48 hours for small, 7 days for large).

## 9. Risk Framework
- **9.1 Smart Contract Risk:** two independent audits before mainnet, plus formal
  verification of core accounting invariants (exchange-rate monotonicity, fee
  distribution correctness). Public bug bounty launched alongside mainnet.
- **9.2 Slashing Risk:** canoLiq's stake is exposed to committee slashing for
  double-signing and non-signing. Mitigations: insurance fund (5% of DAO
  treasury, seeded progressively from fee income over first 12 months, target 5%
  of peak TVL — compensates cCNPY holders for losses); validator diversification
  (no single failure affects more than a fraction of TVL); real-time monitoring
  (on-chain alerts for non-signing with rapid governance-triggered ejection);
  slashing cap (Canopy `MaxSlashPerCommittee` parameter limits slashes per
  committee per block — a protocol-level circuit breaker).
- **9.3 Liquidity Risk:** cCNPY redemptions subject to Canopy's unstaking
  cooldown. canoLiq maintains a liquidity buffer (CNPY in reserve) to honor small
  redemptions instantly; large redemptions follow standard cooldown.
- **9.4 Concentration Risk:** a single protocol controlling a large fraction of
  Canopy's stake could pose systemic risk. canoLiq self-imposes a **TVL cap of
  33% of total Canopy network stake**, pending ecosystem maturation and
  governance approval to lift.
- **9.5 Oracle / Price Risk:** the cCNPY/CNPY exchange rate is computed entirely
  on-chain from protocol state — no external price oracles, eliminating oracle
  manipulation risk for the core yield mechanism.

## 10. Autonomy Graduation
canoLiq begins as a Canopy Nested Chain (shared security) and may graduate to a
fully sovereign chain (its own Security Root) when it meets objective thresholds:

| Graduation Criterion | Target Threshold |
|---|---|
| Total Value Locked (TVL) | > 50,000,000 USD in CNPY |
| Active Validators | > 30 independent validators |
| DAO Participation | > 15% average turnout on governance proposals |
| On-Chain Activity | > 10,000 transactions/day sustained for 30 days |
| Treasury Reserves | > 12 months of operational runway |

Graduation is subject to cross-chain governance coordination (Canopy DAO +
canoLiq DAO) to preserve historical finality and seamless UX during migration.

## 11. Roadmap (90-Day Launch Execution)
| Phase | Timeline | Deliverables |
|---|---|---|
| Foundation | Days 1–30 | Smart contract development; fee model implementation (post-committee split); CLIQ vesting contracts; testnet deployment |
| Audit & Subsidy | Days 30–60 | Independent audit #1 & #2; Canopy DAO subsidy proposal submission; bug bounty launch; multisig treasury setup |
| Mainnet Launch | Days 60–90 | Mainnet deployment; cCNPY minting live; CLIQ distribution begins; liquidity pools seeded; governance portal live |

## 12. Disclaimer
Technical/conceptual document, informational only; not financial/legal/investment
advice. CLIQ and cCNPY involve significant risk including loss of principal.
Design subject to change before and after mainnet based on audit findings,
governance decisions, and evolving Canopy Network economics.
