# canoLiq Security Audit & Remediation Plan

## Context

The `feature/security-review` branch carries the entire **canoLiq liquid-staking plugin**
(`plugin/go/canoliq/`, ~45k LOC added vs `main`) — a Canopy Nested-Chain plugin implementing
cCNPY deposits/redemptions, the CPLQ governance token, vote-escrow staking, a DAO treasury
with multisig + timelock, buyback, fee distribution, autonomy-graduation tracking, and a
read-only HTTP query layer.

This audit checks the implementation against the **canoLiq Whitepaper v1.2** and
**Tokenomics v1.2** design intent and against general blockchain-plugin safety. Goal: surface
issues introduced on the branch and concrete improvements, then give a file-level fix plan so
remediation can proceed before any testnet/mainnet deployment.

**Good news up front (verified correct):**
- CPLQ supply = 100M (`genesis.go:15`), 7-bucket split sums to 10000 bps, recipient bps validated (`genesis.go:186`).
- Fee = 12% with 40/30/15/15 split and 5–20% governance bounds (`config.go:241`, `ValidateParams` 309-314).
- Vesting durations match Tokenomics §6 ("cliff included in total"): Validators 12+24, Founders 12+36, Strategic 6+12 (`genesis.localnet.json`/`genesis.testnet.json` + `vesting.go`).
- Per-action governance matrix matches Tokenomics §7 (`defaultGovernanceTiers` 290-300).
- Vote-escrow multipliers match Tokenomics §4.2 (`tierMultipliers` 35-48); lock prevents early unstake (`stake.go:354`); vote snapshot rejects stake added after proposal creation (`governance.go:261`).
- DeliverTx correctly relies on the FSM enforcing `AuthorizedSigners` from CheckTx; multisig-approve re-checks signer set (`treasury.go:208`). No treasury auto-execute bypass (BeginBlock only *queues*).
- Exchange-rate uses `big.Int` mulDiv (no uint64 overflow); deposit/redeem reject zero-mint/zero-owed dust (`deliver.go:76,184`).

---

## Findings (severity-ranked) & Remediation

### H1 — Deposit principal is never escrowed; redemption-claim credits CNPY with no backing debit  *(HIGH / must-fix)*
**Where:** `deliver.go` — `DeliverMessageCanoliqDeposit` (78-114) and `DeliverMessageCanoliqClaimRedemption` (322-360).

On deposit, the user is debited `msg.Amount + fee` (`from.Amount -= deduct`, line 79) but the
committee/escrow pool is written with **fee only**: line 80 does `feePool.Amount += fee`, line 81
computes `escrow.Amount += msg.Amount` into a *separate* struct read from the same key, and line 113
discards it (`_ = escrow`). The principal is therefore removed from the user and credited to **no
pool** — it survives only as a number in `globals.TotalPooledCnpy`. On claim, `from.Amount = from.Amount
- fee + redemption.CnpyAmount` (line 325) credits CNPY back to the user with **no pool debit**.

**Impact:** CNPY conservation is broken inside the plugin. H2 (now resolved — see below) confirms the
Canopy FSM does **not** enforce any conservation/supply invariant on plugin writes, so this manifests as
scenario **(a): silent loss of deposited principal + fabrication of CNPY at redemption
(insolvency/inflation)** — *not* a fail-safe lock. Deposit destroys `msg.Amount` from the user's real
account (debited, credited nowhere); claim credits CNPY back with no pool debit. Both writes succeed
unconditionally because `StateWrite` is a raw passthrough. The in-memory test store likewise does not
enforce conservation, so unit tests pass while real supply silently diverges.

**Fix:**
- On deposit, actually persist the principal into the escrow/committee pool: write
  `feePool.Amount += msg.Amount + fee` (drop the dead `escrow` struct), OR introduce a dedicated
  `KeyForEscrowPool()` distinct from the committee fee pool and credit `msg.Amount` there.
- On `ClaimRedemption`, debit that same pool by `redemption.CnpyAmount` before crediting the user.
- Add an invariant test asserting `Σ user CNPY + pool == constant` across deposit→reward→redeem→claim,
  and reconcile `globals.TotalPooledCnpy` against the real pool balance.
- Decide explicitly how the committee-staked principal relates to this pool (the whitepaper says
  pooled CNPY is delegated on-chain) and document the custody model.

### H2 — Canopy FSM settlement semantics *(HIGH / investigation — RESOLVED 2026-06-24)*
**Where:** `fsm/state.go:731-748` (`StateMachine.StateWrite`) — **in this monorepo, not a separate repo.**

**Resolved:** `StateWrite` is a raw passthrough — it loops the request's `Sets`/`Deletes` and calls
`s.Set(key, value)` / `s.Delete(key)` with **no token-conservation check, no supply-cap enforcement,
no authority gating**. The plugin can write arbitrary balances to any key, including
`contract.KeyForAccount` (real CNPY accounts) and `KeyForFeePool`.

**Consequences:**
- H1 is confirmed as scenario (a) — **inflation/insolvency**, not a fail-safe lock. Claims will not
  trip a supply check; the plugin's internal accounting silently diverges from real balances.
- Treasury/buyback CPLQ accounting has no FSM-level mint/burn authority backing it — the plugin is
  trusted to keep its own books correct (reinforces L4). Any plugin accounting bug becomes a real
  supply bug with no core-level safety net.

This finding no longer blocks the H1 fix design: the fix must restore conservation entirely within the
plugin, since the FSM will not catch a mistake.

### M1 — Governance quorum & T5 turnout compare boosted vote weight against un-boosted stake snapshot *(MEDIUM)*
**Where:** `governance.go` — `proposalPasses` (434-452, quorum at 442) and turnout accrual (366); weight from `voteWeightFor` (`stake.go:79`).

Vote weight is `stake × lockMultiplier` (up to 4× for 24-month locks), but quorum is checked as
`Σ weightedVotes ≥ quorumBps × SnapshotTotalStaked`, where `SnapshotTotalStaked` is **raw** staked CPLQ
(`globals.TotalStakedCplq`, no multiplier). A single 24-month locker holding ~1.25% of stake produces
5% "turnout," trivially clearing the 5% fee-change quorum. The same mismatch lets `turnoutSumDelta`
exceed 10000 bps (>100%), inflating the T5 autonomy-graduation turnout metric.

**Impact:** quorum becomes far easier to meet than the spec implies; graduation eligibility can be
gamed. Pass-threshold math is internally consistent (weighted yes vs weighted yes+no), so this is a
quorum/turnout-integrity issue, not a tally inversion.

**Fix:** snapshot a **boosted** total at proposal creation (sum of `voteWeightFor` over active stakers)
and use it as the quorum/turnout denominator; or strip the multiplier from the quorum numerator. Clamp
per-proposal turnout to ≤10000 bps before feeding T5. Add tests for both.

### M2 — Redemption unstaking window fails open to ~5 blocks and ignores Canopy's real `UnstakingBlocks` *(MEDIUM)*
**Where:** `deliver.go:201-204` (`window := c.Config.RedemptionUnstakingBlocks; if window == 0 { window = 5 }`); `config.go:184` defaults it to 5 again.

If an operator ships a config without `redemptionUnstakingBlocks`, redemptions mature in ~5 blocks
(~30s), bypassing the cooldown the whitepaper §9.3 relies on for liquidity safety. The value is also a
static config, not Canopy's live `valParams.UnstakingBlocks`.

**Fix:** fail **closed** under `testnet`/`mainnet` profiles — refuse startup (extend `SafetyCheck`) when
`RedemptionUnstakingBlocks` is 0 or implausibly small. Longer term, read `valParams.UnstakingBlocks`
from FSM gov-params (already flagged as TODO at `deliver.go:196`). Keep the 5-block default only for `localnet`.

### M3 — TVL self-cap is disabled by default and modeled as a static absolute, not 33% of network stake *(MEDIUM)*
**Where:** `deliver.go:68` (`params.TvlCapUcnpy > 0` gate); neither genesis JSON sets it; WP §9.4 wants a 33% cap.

`TvlCapUcnpy` defaults to 0 (uncapped) and is absent from both genesis files, so the concentration cap
the whitepaper promises is off. Even when set, an absolute uCNPY ceiling won't track network growth as
§9.4 ("33% of total Canopy network stake") intends.

**Fix:** set a non-zero cap in the testnet/mainnet genesis params; ideally compute the ceiling from live
total network restake (33% bps param) rather than a frozen absolute. At minimum, document the static cap
as a known deviation and have `SafetyCheck` warn when it is 0 on non-localnet.

> ⚠️ **Already implemented on the `canoliq-spec-alignment` branch** (see
> `docs/plans/canoliq-v1_2-implementation-plan.md` Phase B): `tvl_cap_ucnpy` → `tvl_cap_bps`, default
> `3300` (33%), fail-closed when Canopy stake is unavailable. Still valid on *this* branch; if that
> branch merges into `feature/security-review`, M3 auto-resolves — coordinate to avoid double-remediation.

### M4 — Unauthenticated RPC lazy-query path injects external load into the consensus EndBlock *(MEDIUM)*
**Where:** `rpc.go` per-address routes → `lazy_query.go` `enqueueLazy`/`drainLazyQueries`; drained synchronously in `EndBlock` (`canoliq.go:191`). Bind address is operator-set via `CANOLIQ_RPC_ADDR` (`main.go:47`) with no auth.

Each `/v1/account|vesting|redemption|vote|buyback` request enqueues a query that `EndBlock` fulfills with
a synchronous FSM `StateRead` round-trip. Up to `lazyQueueCapacity = 256` queries can be pending, so an
unauthenticated client can force ~256 extra serial state reads per block inside the consensus-critical
EndBlock, slowing block production. The server is off by default but can be bound to `0.0.0.0`.

**Fix:** cap the number of lazy queries drained per block (e.g. 8–16) and shed the rest; document that
RPC must bind to loopback / behind an authenticated proxy; consider moving per-address reads fully off
the consensus path (a side reader over a read-only state snapshot). Add a per-IP rate limit.

### M5 — Share-price math has no first-depositor / inflation guard *(LOW–MEDIUM)*
**Where:** `fee.go:10` (`computeMint`) — floor `mulDiv(amount, totalCcnpy, totalPooled)`; first deposit mints 1:1.

`computeMint` has no virtual-shares offset and no minimum-initial-deposit floor. Once H1 is fixed and the
pool holds real CNPY, a high exchange rate (after rewards accrue against a tiny share supply) causes
later small deposits to lose value to floor-division rounding — the classic ERC-4626 first-depositor
precision problem.

**Mitigating fact (lowers severity):** the *active* donation-inflation attack is **not** available here —
`globals.TotalPooledCnpy` is a plugin-internal counter incremented only by deposits and `ProcessRewards`,
not a readable account balance an attacker can inflate by direct transfer. So the residual risk is passive
rounding precision loss, partially covered already by the zero-mint rejection (`deliver.go:76`). Becomes
relevant only after H1 restores a real backing pool.

**Fix:** adopt a virtual-shares/offset on the exchange-rate math (ERC-4626 style), or seed the pool at
genesis / enforce a minimum initial deposit so the share price can't start at a manipulable 1:1 against a
near-empty pool. Add a rounding-loss bound test.

### L1 — `ValidateParams` allows `MultisigThreshold = 0` while signers are configured *(LOW–MEDIUM)*
**Where:** `config.go:324` only checks `threshold > len(signers)`. `treasury.go:173` gates with `approvals < threshold`.

A passed param-change setting `MultisigThreshold = 0` (with signers present) makes the
`RequiresMultisig` gate pass with **zero** approvals, defeating the multisig on large treasury spends.
Requires governance to enact, so defense-in-depth.

**Fix:** in `ValidateParams`, require `MultisigThreshold >= 1` whenever `len(MultisigSigners) > 0` (and
consider a sane floor like ⌈signers/2⌉). Add a unit test.

### L2 — `math/rand` used for `QueryId` throughout consensus handlers *(LOW / hygiene)*
**Where:** every `Deliver*`/`Process*`/`load*` uses `rand.Uint64()` for read-correlation IDs; stray `_ = rand.Uint64()` at `genesis.go:287`.

Functionally low-risk (global rand is concurrency-safe; collisions across ~8 keys are astronomically
unlikely), but using a PRNG for correlation IDs in deterministic consensus code is a smell and a latent
mis-routing bug.

**Fix:** replace with a deterministic per-request counter (e.g. an incrementing field on `Canoliq`).
Remove the dead `rand` call in genesis.

### L3 — Fee pool conflates protocol tx-fees with committee subsidies *(LOW / design)*
**Where:** `reward.go:19-103` — `ProcessRewards` treats *any* growth of `KeyForFeePool(chainId)` since the last sweep as reward delta subject to the 12% fee + 40/30/15/15 split, but every handler also adds tx fees to that same pool.

Protocol tx-fee income is therefore re-distributed as if it were staking reward (88% back to cCNPY
holders, etc.), double-handling the fee model described in WP §3.3/§4.

**Fix:** separate the committee-reward pool key from the tx-fee accumulator, or subtract accumulated
tx-fees from the delta before applying the protocol fee. Document the intended treatment of tx-fee revenue.

### L4 — Circulating-supply bookkeeping gaps & buyback realism *(LOW / design)*
**Where:** `treasury.go:354-388` (treasury **CPLQ** spend credits recipient but never bumps `CplqCirculatingSupply`); `buyback.go` executes at a governance-set `PriceMicroCnpyPerCplq` against treasury CPLQ (internal accounting swap, Phase-3 market route deferred); burn decrements both supplies (`buyback.go:112-131`).

Inconsistent circulating-supply accounting and a governance-priced (non-market) buyback. Acknowledged as
Phase-3 deferrals in comments, but worth tracking before mainnet value flows.

**Fix:** bump `CplqCirculatingSupply` on treasury-CPLQ spend; add an invariant test
(`circulating ≤ total`, both ≤ 100M); gate buyback price against a sanity band or real route before mainnet.

### L5 — Doc/comment drift & minor inconsistencies *(LOW / cleanup)*
- `treasury.go:17` comment mentions a "BeginBlock auto-execute" of spends that does not exist — remove to avoid implying an un-gated path.
- `reward.go:117`/`config.go:253` cite "WP §11" for the insurance fund; v1.2 places it at §9.2 — fix the reference. *(Already fixed on `canoliq-spec-alignment` per the v1.2 plan's Phase D, along with the stuck-redemption alert and README insurance narration; coordinate the merge.)*
- `blocksPerMonth` differs between genesis (`blocksPerYear/12 = 438,000`, `genesis.go:219`) and staking/lock tiers (`432,000`, `stake.go:20`); reconcile or document why vesting vs lock use different month lengths.

---

## Critical files (for remediation)
- `plugin/go/canoliq/deliver.go` — H1 escrow/claim accounting, M2 redemption window.
- `plugin/go/canoliq/governance.go` + `stake.go` — M1 quorum/turnout denominator.
- `plugin/go/canoliq/config.go` — M2/M3 `SafetyCheck` fail-closed, L1 `ValidateParams` floor.
- `plugin/go/canoliq/reward.go` — L3 fee/reward separation, §-ref fix.
- `plugin/go/canoliq/lazy_query.go` + `rpc.go` + `main.go` — M4 per-block drain cap / bind guidance.
- `plugin/go/canoliq/treasury.go` / `buyback.go` — L4 supply bookkeeping, L5 comment.
- `plugin/go/canoliq/fee.go` — M5 share-price guard (virtual shares / min initial deposit).
- `genesis.testnet.json` — M3 TVL cap param.
- `fsm/state.go:731-748` (this monorepo) — H2 settlement path; **investigation closed**, no change needed there (the fix lives in the plugin).

## Verification
- **Unit/invariant tests** (the suite is `go test ./plugin/go/canoliq/...`): add
  (1) CNPY-conservation test across deposit→reward→redeem→claim (H1);
  (2) quorum/turnout test with mixed lock tiers asserting denominator parity and ≤100% turnout (M1);
  (3) `ValidateParams` rejecting `threshold=0` with signers (L1);
  (4) `SafetyCheck` rejecting non-localnet profiles with zero redemption window / zero TVL cap (M2/M3).
- **H2:** ✅ done — `fsm/state.go:731-748` confirmed to be a raw passthrough (no conservation). Remaining:
  after the H1 fix, reproduce a deposit→reward→redeem→claim cycle on localnet and confirm real account +
  pool balances reconcile (they will not today).
- **M4:** load-test the RPC per-address routes while a localnet produces blocks; confirm block interval
  does not stretch under a burst and that the per-block drain cap holds.
- **Regression:** full `go test ./...` for the plugin module plus a localnet deposit/redeem/stake/
  vote/treasury-spend smoke run via `canoliqctl` after each fix.

## Notes
- No code has been changed by this audit. Findings are ordered by severity; H1 is the blocker for any
  value-bearing deployment (H2, which it depended on, is now resolved — the FSM offers no safety net).
- All line numbers reference the current `feature/security-review` checkout.
- **Verification status (re-checked 2026-06-24 against live code):** H1, H2, M1, M2, M3, M4, L3, L4, and
  the M5 share-math gap were confirmed against the current branch. M4 (`drainLazyQueries` drains the whole
  256-deep queue synchronously in EndBlock, `lazy_query.go:141` / `canoliq.go:191`), L3 (deposit adds tx
  fee to the same `KeyForFeePool` that `ProcessRewards` sweeps as reward delta), and L4 (`CplqCirculatingSupply`
  is touched only in `buyback.go`, never on treasury CPLQ spend) all verified accurate.
- **Cross-branch note:** M3 and the L5 §9.2 reference (plus stuck-redemption / insurance items) are already
  implemented on `canoliq-spec-alignment` (`docs/plans/canoliq-v1_2-implementation-plan.md`). They remain
  valid here; merging that branch first auto-resolves them.
- **Scope:** this audit covers the `plugin/go/canoliq/` package and its FSM settlement touchpoint
  (`fsm/state.go`). Not separately reviewed: the host-side `canoliqctl` CLI (`plugin/go/canoliqctl/`),
  generated `*.pb.go`, and the read-only docs site.
