# Praxis V2 Roadmap

19 items · 3 waves · Active development

---

## Wave 1 — Critical (Build Now)

### Security & Integrity

**1. Dispute Window After Resolution**
Add a `dispute_resolution` transaction. After a market is resolved, open a 24-hour (N block) dispute window. Any address can challenge the outcome by staking a dispute bond. If the dispute succeeds, the original resolver is slashed and the market is re-opened.
Tags: `Security` `New Tx Type`

**2. Resolution Window Enforcement**
Block `resolve_market` calls before the market's `resolution_height` is reached. Currently any resolver can finalise early. Add a height check in `DeliverResolveMarket`.
Tags: `Security` `Critical`

**3. Creator Self-Resolution Restriction**
Prevent the market creator from also being the resolver. This is the single most obvious manipulation vector. One-line check in `CheckCreateMarket`.
Tags: `Security` `Critical`

**4. Invalid Market Handling — Void + Refund**
Add an `INVALID` resolution outcome. When a market resolves as INVALID, return all stakes to the original forecasters with no fee deduction. Required for markets where the question becomes unanswerable.
Tags: `Security` `New Tx Type`

**5. Resolver Stake Slashing**
Require resolvers to bond a stake when designated. If a dispute succeeds against their resolution, slash the bond and redistribute to the disputing address. Aligns resolver incentives with honest outcomes.
Tags: `Security` `Protocol`

### Revenue

**6. Losing Pool Cut — Protocol Treasury Fee**
Take 2–3% of every losing pool before paying out winners. Route it to a hardcoded treasury address in `contract.go`. This is how Polymarket makes most of its revenue. At scale this is the highest ROI change in the entire roadmap.
Tags: `Revenue` `High Priority`

**7. Volume Fee on Predictions**
Charge a small percentage (0.5–1%) of every prediction stake on submission, separate from the transaction fee. Prediction volume is always much higher than market creation volume. Compounds revenue every time anyone bets on any market.
Tags: `Revenue`

---

## Wave 2 — Core Features

**8. Treasury Address in contract.go**
Hardcode a protocol treasury address as a constant in `contract.go`. All protocol fee cuts route here automatically. Should be a multisig or governance-controlled account for transparency.
Tags: `Revenue` `Protocol`

**9. Market Creation Bond**
Charge a returnable bond on market creation — returned to the creator if the market resolves cleanly, slashed if the resolver misbehaves or the market is abandoned. Deters spam and aligns creator incentives with good market hygiene.
Tags: `Revenue` `Security`

**10. Position Withdrawal — Exit Before Resolution**
Add a `withdraw_prediction` transaction that lets forecasters exit their position before the market resolves. Apply a withdrawal penalty (e.g. 10%) to discourage gaming — penalty goes to the remaining pool. Polymarket's most-used feature after betting itself.
Tags: `New Tx Type` `UX`

**11. Market Categories and Tags**
Add a `category` string field to `MessageCreateMarket`. Categories: Crypto, Politics, Sports, Science, Entertainment, Other. Enables filtering on the Markets page and makes large market sets navigable.
Tags: `Proto Change` `UX`

---

## Wave 3 — Growth & Polish

**12. Leaderboard — Top Forecasters**
Rank addresses by total profit, accuracy rate, and number of markets predicted. Query all prediction transactions, aggregate by address, and display on a dedicated Leaderboard page. Builds competition and social proof.
Tags: `Growth` `Frontend`

**13. My Positions Page**
A personal dashboard showing all open predictions, pending claims, resolved markets, and total P&L. Query transactions by the loaded address and display a personal history. Makes the protocol feel like a real trading interface.
Tags: `UX` `Frontend`

**14. Market Search**
Client-side search over loaded markets by question text, category, or creator address. No new RPC endpoint needed — filter the already-loaded market list. Simple but essential once there are more than 20 markets.
Tags: `UX` `Frontend`

**15. Rate Limiting — Cap Market Creation Per Address**
Limit each address to N market creations per block or per epoch. Prevents a single actor from flooding the chain with spam markets. Implement as a counter in `DeliverCreateMarket` state.
Tags: `Security` `Protocol`

---

## Competitive Analysis

| Protocol | Sovereign Chain | No Oracle | Position Exit | Dispute | Revenue Model |
|---|---|---|---|---|---|
| **Praxis v1** | ✓ | ✓ | ✗ | ✗ | Fees only |
| Polymarket | ✗ | ✗ | ✓ | ✓ | 2% volume fee |
| Augur | ✗ | ✗ | ✓ | ✓ | Reporter fees |
| Manifold | ✗ | ✓ | ✓ | ✗ | Token model |
| **Praxis v2 target** | ✓ | ✓ | ✓ | ✓ | Pool cut + volume |

---

*Praxis ($PRX) · Canopy Nested Chain · Prediction Markets Protocol*
