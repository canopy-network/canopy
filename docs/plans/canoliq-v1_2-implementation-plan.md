# canoLiq v1.2 spec alignment — implementation plan

## Context

A two-agent audit of the canoLiq codebase against `canoLiq_Whitepaper_v1.2.pdf`
and `canoLiq_Tokenomics_v1.2.pdf` confirmed alignment on ~37/39 verifiable items
(token supply, 7 buckets + vesting, fee model, buyback default = burn, insurance
peak-TVL mechanism, vote-escrow lock tiers with lock-end-height enforcement at
`stake.go:354`, 7-tier governance matrix, autonomy graduation, 3 push-alert
conditions, treasury multisig+timelock).

Two **genuine** docs-vs-code gaps remain:

1. **Whitepaper §7 — Restaking optimization** describes a multi-committee
   allocation engine; the codebase has *no* multi-committee logic at all (a
   single-committee `ValidatorRegistry` is the entire stake-management surface).
2. **Whitepaper §9.4 — TVL self-cap "33% of total Canopy stake"** is implemented
   in code as an absolute `tvl_cap_ucnpy` parameter, not a percentage of Canopy
   stake.

User direction: implement restaking now, change the TVL cap to
percentage-of-Canopy-stake, and also roll in three previously-deferred
cleanups (stale insurance README narration; `phase2_test.go:208` copylocks vet
warning; stuck-redemption alert).

A pre-plan investigation confirmed the **Canopy FSM does not need changes**:
`fsm/state.go:679` `StateRead` is generic — it accepts arbitrary keys and
prefix-range iterations and just delegates to the store. The artificial
narrowness sits in the plugin's `contract` package, which mirrors only three
`KeyFor*` helpers (`KeyForAccount`, `KeyForFeeParams`, `KeyForFeePool`).
Mirroring more of `fsm/key.go` in the plugin's contract package is all that's
needed to give the plugin reach into committee/validator/supply state.

Intended outcome: ship a single feature branch landing four phases (D first for
risk-free cleanups; A as foundation; then B and C in order) so the
implementation matches the v1.2 spec and no further docs-vs-code drift remains.

---

## Progress checklist

Tick items as commits land. Each box maps to a coherent work block in the
phase body below.

### Phase D — Independent cleanups ✅ landed on `canoliq-spec-alignment`
- [x] README insurance narration fix (`plugin/go/canoliq/README.md:1166–1168`)
      — commit `deb6ae56`
- [x] `phase2_test.go:208` copylocks vet warning (replace `*params` copy with
      `proto.Clone`) — commit `f140fe1a`
- [x] Stuck-redemption alert: `KeyForMatureRedemption` global index in
      `state.go` (and `deliver.go` write-on-redeem / delete-on-claim plumbing)
      — commit `c203e4b4`
- [x] Stuck-redemption alert: evaluator + debounce in `alerts.go` — commit
      `c203e4b4`
- [x] Stuck-redemption alert: `StuckRedemptionCount` knob on `AlertConfig`
      (config.go JSON-config, not `CanoliqParams` proto — matches the
      existing alert-tuning pattern) — commit `c203e4b4`
- [x] Stuck-redemption tests (threshold crossing, debounce, immature-not-counted,
      end-to-end index lifecycle) — commit `c203e4b4`

### Phase A — Plugin contract package extension ✅ landed on `canoliq-spec-alignment` — commit `ee23a092`
- [x] **Decision (resolved during execution):** generate local protos
      via the plugin's existing protoc pipeline. `proto/account.proto`
      already had a `Pool` wire-compatible with `lib.Pool`; added `Supply`
      next to it with matching field numbers (1–5). Regenerated
      `contract/account.pb.go` via `protoc-gen-go v1.36.11`.
- [x] `KeyForSupply()` (`contract/contract.go`) — singleton at
      `supplyPrefix = []byte{10}` mirroring `fsm/key.go:40,67`.
- [x] `JoinLenPrefix` + `formatUint64` — *already present* in the contract
      package (`contract/plugin.go:343`, `contract/contract.go:287`); no
      porting needed.
- [x] Helper: `readCanopyTotalStake() (uint64, *contract.PluginError)`
      (`canoliq/canopy_state.go`).
- [x] Helper: `readCanopySupply() (*contract.Supply, *contract.PluginError)`
      — gives Phase C access to `Supply.committee_staked` (pre-aggregated
      per-committee stake), which makes `iterateCanopyCommittee` and
      `KeyForCommittee`/`KeyForValidator` unnecessary at this stage.
      Deferred to a later phase if per-validator granularity is required.
- [x] Byte-level key parity tests (`contract/keys_test.go`) — cover the
      existing three `KeyFor*` helpers plus the new `KeyForSupply`.
- [x] FakeStore round-trip for `Supply` decode (`canoliq/canopy_state_test.go`)
      — absent path returns `(0, nil)` / `(nil, nil)`; present path
      round-trips `Staked` + repeated `CommitteeStaked`.

**Scope adjustments vs original plan:**
- `KeyForValidator`/`KeyForCommittee`/`KeyForDelegate` and
  `iterateCanopyCommittee` are *not implemented in Phase A*. The Canopy
  `Supply` struct already aggregates per-committee stake via
  `committee_staked` (repeated `Pool`), which is exactly what the Phase C
  restaking allocator needs for the denominator of reward-per-CNPY. If
  Phase C later needs per-validator iteration (e.g. for slashing-risk
  diversification heuristics), those keys can be added then — the work
  is mechanical mirroring of `fsm/key.go:109–122`.

### Phase B — Percentage TVL cap (Whitepaper §9.4) ✅ landed on `canoliq-spec-alignment`
- [x] Proto hard rename: `tvl_cap_ucnpy` → `tvl_cap_bps` in
      `plugin/go/proto/canoliq.proto`; regenerated `contract/canoliq.pb.go`
- [x] Default `TvlCapBps = 3300` (33%) wired in `config.go`
- [x] Deposit check rewritten in `deliver.go` (percentage; fail-closed when
      Supply absent or `staked == 0` → new `ErrCanopyStakeUnavailable`,
      `codeCanopyStakeUnavailable` in `error.go`)
- [x] `/v1/health` rewritten: drops `tvlCapUcnpy`; surfaces `tvlCapBps`,
      `canopyTotalStake`, `tvlCapUcnpyEffective`, `tvlUtilizationBps`.
      Snapshot extended with `CanopyTotalStake`, batched into the existing
      `refreshSnapshot()` round-trip (one extra `KeyForSupply()` read,
      no extra round-trip).
- [x] Tests: existing T3 suite rewritten — boundary, lift via bps raise,
      uncapped, **fail-closed when Supply absent**, fail-closed when
      `staked == 0`, health-surface contract. `newTestCanoliq()` fixture
      seeds a generous default Supply so the 70+ pre-existing
      `DefaultParams()` call sites keep working unchanged; cap-edge tests
      `s.del()` the key to exercise absence.
- [x] Docs: `docs/canoliq-site/docs/advanced/tvl-cap.mdx` rewritten for
      the percentage model with the fail-closed-posture rationale —
      commit `272cf5f3`.
- [x] Commits: split as planned —
      - `2cf750a0` `feat(canoliq): percentage TVL cap (Whitepaper §9.4)`
        (proto + config + deliver + snapshot + query + error + tests)
      - `272cf5f3` `docs(canoliq): rewrite tvl-cap.mdx for percentage model`

### Phase C — Restaking policy + observability (Whitepaper §7) ✅ landed on `canoliq-spec-alignment`

**Scope adjustment vs original plan:** the user picked the "Policy +
observability only" option (see the Phase C scope question). Active
rebalancing (`MessageCanoliqRebalance`, epoch hook,
`applyRestakingDelta`) is deferred — it requires a delegation-routing
primitive not defined in the codebase, and §11 Roadmap does not list it
as a launch deliverable. The `/v1/restaking` surface is observation-only
with drift / under-min / over-max flags so operators can react manually.

- [x] `RestakingPolicyEntry` proto + `CanoliqParams.restaking_policy`
      (proto field 32). Regenerated `contract/canoliq.pb.go`.
- [x] `ValidateParams` checks: empty list OK; non-empty targets sum to
      10000; unique committee ids; per-entry min ≤ max (both > 0).
- [x] `KeyForValidator` added to plugin contract package — mirrors
      `fsm/key.go:121` byte-for-byte. (Originally deferred from Phase A;
      needed here to read per-operator committees.)
- [x] `proto/validator.proto` mirroring `lib.Validator` (only `address`,
      `staked_amount`, `committees` — fields we read).
- [x] `canoliq/canopy_state.go::readCanopyValidator(addr)` reader helper.
- [x] `canoliq/restaking.go`: `observeCurrentAllocation`,
      `buildRestakingView` (drift, under-min, over-max, PolicyCompliant
      aggregate). Restaking semantics implemented per §1.1 / §7:
      operator stake counts toward every committee in its `committees[]`.
- [x] `Snapshot.CurrentRestakingAllocation` populated inside the existing
      Batch 2 of `refreshSnapshot` — one extra read per registered
      operator, no extra round-trip.
- [x] `Plugin.QueryRestaking()` + `RestakingView` JSON type.
- [x] `/v1/restaking` route wired in `rpc.go`.
- [x] Tests (`restaking_test.go`): 15 cases —
      4 ValidateParams (empty, weight-sum, dup committee, min/max),
      3 observation (restake semantics, absent operator skip, empty),
      5 view assembly (drift+compliance, under-min, over-max,
      observed-without-policy, policy-without-observation),
      3 integration (snapshot population, empty query, populated query).
- [x] Docs: `docs/canoliq-site/docs/advanced/restaking.mdx` (honest about
      observation-only scope), sidebar entry, `api/endpoints.mdx` updated
      with the new route + the Phase B `/v1/health` shape change (which
      had been left stale on that page).
- [x] Commits: shipped as the 4-way split —
      - `ef44cd0d` `feat(canoliq): plugin contract — KeyForValidator + Validator proto`
        (also corrects `@gotags` JSON tags across the .pb.go files —
        Phases A + B used raw `protoc` and skipped the inject-tag step)
      - `b0097d32` `feat(canoliq): restaking policy + per-committee observation (WP §7)`
        (proto + ValidateParams + restaking.go + snapshot + query + rpc + 15 tests)
      - `7336f1e5` `docs(canoliq): document restaking policy + /v1/restaking`
        (`advanced/restaking.mdx` + sidebar entry)
      - `b354bca6` `docs(canoliq): refresh endpoints.mdx — /v1/health shape + /v1/restaking`
        (Phase B left this page stale on the cap shape; this commit
        catches it up alongside adding the new route)

**Deferred to a future "active rebalancing" workstream** (not in scope here):
`MessageCanoliqRebalance`, epoch hook in `ProcessRewards` /
`BeginBlock`, `applyRestakingDelta`, governance round-trip under
`ACTION_PROTOCOL_UPGRADE` for live policy enforcement, `canoliqctl
restaking-status`, multi-committee fakeStore reward-skew assertion.

### Final verification + closure 🟡 partly landed, closure-appendix uncommitted
- [x] `cd plugin/go && go vet ./canoliq/...` clean (verified after every phase)
- [x] `cd plugin/go && go test ./canoliq/... ./canoliqctl/... ./contract/...`
      green (verified after every phase)
- [~] Manual fakeStore smoke: deposit → redeem → observe `/v1/restaking`
      and `/v1/health.tvl_cap_ucnpy_effective` — ran ad-hoc on
      2026-06-22 via a one-off `TestSpecAlignmentSmoke` (then scratched
      per user decision: the per-phase unit tests already cover the
      same paths individually and a persistent smoke duplicates that).
      The ad-hoc run confirmed end-to-end: percentage cap surfaces
      `tvlCapBps=3300` / `tvlCapUcnpyEffective=33000000` against
      `canopyTotalStake=100000000`; restake double-counting works
      (total observed exposure 2.5M from 1.5M of operator stake spread
      across overlapping committees); stuck-redemption index entry is
      written by the redeem path. Workstream-2 of the readiness doc
      still owns the cross-process smoke on a real compose image.
- [x] Appendix to `docs/canoliq-whitepaper-tokenomics-discrepancies.md`
      noting Whitepaper §7 and §9.4 implementation gaps closed
      — commit `dceb8884`.

---

## Phase D — Independent cleanups (land first, smallest risk)

These don't depend on anything else and can ship in one or three small commits
before the heavier phases. Reduces noise during Phase A–C reviews.

- **README insurance narration** (`plugin/go/canoliq/README.md:1166–1168`).
  Currently reads `default 1500 = 15% of treasury slice → ≈1.5% of fee`;
  correct is `default 500 = 5% of treasury slice → ≈0.5% of fee` (matches
  `config.go:252` `InsuranceBps: 500` and Whitepaper §9.2). Also drop the
  "Phase 3 will add slashing-reimbursement disbursement; Phase 2 only seeds the
  pool" clause — T4 (`cfeb00fc`) already shipped peak-TVL tracking and
  target-driven auto-off.
- **`phase2_test.go:208` copylocks vet warning**. The line
  `newParams := *params` copies a struct containing a `sync.Mutex` (or proto
  state lock). Replace with `proto.Clone(params).(*contract.CanoliqParams)` so
  the param-change payload exercises a fresh struct without the lock copy.
- **Stuck-redemption alert** (fourth condition, deferred during T6). Requires a
  global mature-unclaimed-redemption index keyed by `mature_height`. New work:
  - `stake.go`: maintain a `KeyForMatureRedemption(height, id)` write on the
    unstake `MatureHeight` and delete on claim. Reuse existing `UnstakingIndex`
    per-address for cheap lookup; the new index is global, ordered by
    `mature_height` for the alert evaluator.
  - `alerts.go`: add a `stuck_redemption` evaluator that range-scans the index
    up to `currentHeight()` and fires when the count of mature-unclaimed
    redemptions exceeds a governance threshold (proposal: 10 by default; `crit`
    severity).
  - `config.go` / `canoliq.proto`: new `StuckRedemptionThreshold` field,
    default 10.
  - Tests: cover the threshold boundary, debounce against the existing alert
    debounce, and ensure an in-flight claim removes the index entry.

**Verification (Phase D):** `cd plugin/go && go vet ./canoliq/...` clean (vet
warning gone); `go test ./canoliq/...` green; new tests for stuck-redemption
added; README diff matches v1.2 spec.

---

## Phase A — Extend the plugin contract package

Mirror the `fsm/key.go` helpers needed by B and C in
`plugin/go/contract/contract.go`. **No Canopy FSM changes.**

**Mirror these prefixes and helpers** (from
`/Users/sokrato/Developer/Blockchain/canopy/fsm/key.go:30–122`):

- `validatorPrefix = []byte{3}`, `KeyForValidator(addr)`
- `committeePrefix = []byte{4}`, `KeyForCommittee(chainId, addr, stake)`,
  `CommitteePrefix(chainId)`
- `supplyPrefix = []byte{10}`, `KeyForSupply()` (for total network stake
  aggregate)
- `delegatePrefix = []byte{11}`, `KeyForDelegate(chainId, addr, stake)`,
  `DelegatePrefix(chainId)` (only if needed by restaking allocator)

Port the small utilities `lib.JoinLenPrefix` and `formatUint64` (BigEndian
8-byte) — these are 5-line helpers; copying them is safer than taking a `lib/`
dependency the plugin contract package doesn't already have.

**Unmarshal targets:** the proto types `Validator` and `Supply` come from `lib/`
(the FSM uses `lib.Validator`, etc.). The plugin's `contract` package either
(a) generates its own copy from the same `.proto` files, or (b) imports `lib/`.
Pick whichever the existing plugin code already does for symmetric types —
inspect `plugin/go/contract/contract.go` and the proto wiring to confirm the
convention before choosing.

**Optional helpers on `Canoliq`** (`plugin/go/canoliq/canoliq.go` or new
`canopy_state.go`):

- `readCanopyTotalStake() (uint64, error)` — single `StateRead` of
  `KeyForSupply()`.
- `iterateCanopyCommittee(chainId uint64) ([]CommitteeMember, error)` —
  `Ranges`-based `StateRead` over `CommitteePrefix(chainId)`.

**Tests:** Plugin contract package gets unit tests for the byte-level key
construction (compare against the FSM's `key.go` output for known inputs).
End-to-end smoke test: in a fakeStore harness, seed a known `KeyForSupply()`
value, confirm the plugin reads it back.

**Verification (Phase A):** `cd plugin/go && go test ./contract/... ./canoliq/...`
green; key-construction tests verify byte-for-byte parity with FSM keys.

---

## Phase B — Percentage TVL cap (Whitepaper §9.4)

**Proto / config (`plugin/go/proto/canoliq.proto`, `config.go`):**

- Replace `tvl_cap_ucnpy` (absolute) with `tvl_cap_bps` (percentage of total
  Canopy stake, in bps).
- Default `TvlCapBps = 3300` (33% per spec).
- `0` continues to mean "uncapped".
- Migration: existing genesis files (localnet, testnet) had
  `tvl_cap_ucnpy: 0` so default behavior is unchanged. Drop the `TvlCapUcnpy`
  field; absolute caps were never the spec.

**Deposit check** (`deliver.go::DeliverMessageCanoliqDeposit`, current lines
66–70):

```go
if params.TvlCapBps > 0 {
    totalCanopyStake, err := c.readCanopyTotalStake()  // Phase A helper
    if err != nil { return … }
    cap := mulDiv(totalCanopyStake, params.TvlCapBps, 10_000)
    if globals.TotalPooledCnpy + msg.Amount > cap {
        return ErrTVLCapExceeded()
    }
}
```

**Error / query:**

- `ErrTVLCapExceeded()` unchanged (`error.go:259`).
- `query.go`: `/v1/health` should expose `tvl_cap_ucnpy_effective` (the computed
  cap at the current height) so operators can see the live threshold; existing
  `tvl_cap_ucnpy` field on the response is renamed/repurposed.

**Tests:** New unit tests covering: cap=0 (uncapped); deposit at exactly 33%
boundary; deposit past 33%; behavior when Canopy total stake is unavailable
(`readCanopyTotalStake` returns error — fail-closed: reject the deposit with a
clear error, do not silently accept).

**Docs:** `docs/canoliq-site/docs/advanced/tvl-cap.mdx` — rewrite for the
percentage model. Note governance can tune `tvl_cap_bps`; default 3300 (33%);
`0` = uncapped.

**Verification (Phase B):** Phase A tests + new percentage-cap tests pass;
manual deposit through `canoliqctl` against a fakeStore image hits the new path.

---

## Phase C — Restaking optimization engine (Whitepaper §7)

The biggest piece. The whitepaper specifies four properties: identify high-yield
committees (reward-per-CNPY), manage slashing risk (diversification), balance
liquidity (keep enough in canoLiq's own committee for redemption demand),
governance-controlled allocation policy (min/max stake per committee).

**Proto / config additions:**

- `RestakingPolicy` proto type:
  `{committee_id: uint64, min_stake_ucnpy: uint64, max_stake_ucnpy: uint64, target_weight_bps: uint64}`
  — repeated in `CanoliqParams.restaking_policy`.
- `MessageCanoliqRebalance` — governance-only message that triggers a rebalance
  (or, alternative: trigger on epoch boundary every `N` blocks). Recommend
  on-epoch automatic + on-demand by governance proposal.

**State additions (`canoliq.proto::CanoliqGlobals`):**

- `current_allocation` — repeated `{committee_id, allocated_ucnpy}` snapshot so
  `/v1/restaking` can report current state.

**New module: `plugin/go/canoliq/restaking.go`:**

- `evaluateRestakingPolicy(ctx)` — for each policy entry, read the Canopy
  committee state via Phase A helpers; compute reward-per-CNPY (from observed
  rewards over a trailing window) and current canoLiq allocation; produce a
  proposed reallocation respecting min/max bounds and target weights.
- `applyRestakingDelta(delta)` — write back via `StateWrite`, recording the
  rebalance event for `/v1/restaking`.
- Integration: at every Nth block (configurable, e.g.
  `BlocksPerRestakingEpoch = 7200` ≈ 12h at 6s blocks) in `ProcessRewards` (or
  a new `BeginBlock` hook), evaluate and apply.

**Backward compat:**

- Genesis without a `restaking_policy` block → single-committee fallback
  (current behavior). The optimizer no-ops if the policy is empty.
- Existing `ValidatorRegistry` per-validator pro-rata reward logic continues to
  operate on canoLiq's own committee; the restaking engine governs *how much*
  canoLiq stakes per committee, not how rewards within canoLiq's committee are
  distributed.

**Tests:**

- `restaking_test.go`: policy validation (sum of target weights = 10000;
  min ≤ max); allocation respects min/max; reallocation invariants (total
  allocated = total bonded); epoch trigger fires every N blocks; on-demand via
  governance proposal.
- Multi-committee fakeStore harness: seed two Canopy committees with different
  reward-per-CNPY, confirm the optimizer skews allocation toward the
  higher-yield committee within policy bounds.
- Governance round-trip: a `RestakingPolicy` change proposal under
  `ACTION_PROTOCOL_UPGRADE` (10%/67%/7d) passes and updates the live policy.

**Docs:**

- New `docs/canoliq-site/docs/advanced/restaking.mdx` — explain the engine, the
  policy params, the epoch cadence, and the safety guarantees (no allocation
  outside policy bounds).
- Sidebar update.

**Out-of-scope within Phase C** (deferred to a follow-up): live oracle for
inter-committee reward rates beyond simple trailing-window observation;
cross-committee slashing arbitrage. These are mainnet hardening items; testnet
ships with the trailing-window heuristic.

**Verification (Phase C):** All Phase A/B/C tests green; multi-committee
fakeStore harness shows expected allocation skew; `canoliqctl restaking-status`
reports the current allocation matching the policy.

---

## Critical files

Plan touches these (representative; not exhaustive):

| File | Phase | Purpose |
|---|---|---|
| `plugin/go/canoliq/README.md` | D | Insurance narration fix |
| `plugin/go/canoliq/phase2_test.go` | D | `proto.Clone` to kill copylocks |
| `plugin/go/canoliq/alerts.go`, `stake.go`, `config.go` | D | Stuck-redemption alert + index |
| `plugin/go/contract/contract.go` | A | New `KeyFor*` helpers mirroring `fsm/key.go:103–122` |
| `plugin/go/canoliq/canoliq.go` (or new `canopy_state.go`) | A | `readCanopyTotalStake`, `iterateCanopyCommittee` |
| `plugin/go/proto/canoliq.proto` | B, C | `tvl_cap_bps` rename; `RestakingPolicy`; `current_allocation` |
| `plugin/go/canoliq/deliver.go` | B | Percentage TVL cap check |
| `plugin/go/canoliq/restaking.go` (NEW) | C | Optimizer + epoch trigger |
| `docs/canoliq-site/docs/advanced/{tvl-cap,restaking}.mdx` | B, C | Docs alignment |
| `plugin/go/canoliq/genesis.{localnet,testnet}.json` | B | Drop `tvl_cap_ucnpy` if present; defaults flow from `config.go` |

## Files explicitly NOT modified

- `fsm/state.go`, `fsm/key.go`, anything else in `fsm/` — Phase A confirmed
  unnecessary: `StateRead` is already generic.
- `genesis.testnet.json` bucket addresses, multisig signers, validator
  registry — those are testnet-deployment-readiness territory, not
  spec-alignment.

## Verification (end-to-end)

1. `cd /Users/sokrato/Developer/Blockchain/canopy/plugin/go && go vet ./canoliq/...`
   → clean (no copylocks).
2. `cd /Users/sokrato/Developer/Blockchain/canopy/plugin/go && go test ./canoliq/... ./canoliqctl/... ./contract/...`
   → all green.
3. `TestBundledTestnetGenesisIsSafetyCheckClean` + governance + alert tests
   pass without modification (Phase D, no genesis structure change).
4. New tests in scope:
   - `contract` key-construction parity vs FSM (byte-for-byte).
   - Percentage TVL cap boundary + fail-closed behavior on
     `readCanopyTotalStake` error.
   - Restaking policy validation; allocation invariants; multi-committee
     fakeStore reward-skew.
   - Stuck-redemption threshold crossing + claim-removes-index.
5. Manual smoke against a private testnet image (Workstream 2 of the readiness
   doc): deposit, redeem, observe `/v1/restaking` and
   `/v1/health.tvl_cap_ucnpy_effective`.
6. Whitepaper §7 and §9.4 phrasing left unchanged; the code now matches.
7. Update `docs/canoliq-whitepaper-tokenomics-discrepancies.md` with a
   "v1.2 → code alignment closure" appendix noting the two implementation gaps
   closed.

## Sequencing recommendation

- **Land Phase D first** as 3 small commits on the existing `canoliq` branch
  (or a dedicated `canoliq/spec-alignment` branch through the now-familiar
  feature-branch + PR flow).
- **Phase A** is the foundation for B and C — one commit, well-tested.
- **Phase B** lands next (small, contained).
- **Phase C** is the largest; split into proto/config commit + engine commit +
  tests + docs commit.
- Total estimated effort: D ≈ 1 session, A ≈ 1 session, B ≈ 1 session, C ≈ 2–3
  sessions.
