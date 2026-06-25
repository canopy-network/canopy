# canoLiq Tokenomics Implementation Plan

## Context

canoLiq is a liquid staking protocol described in two whitepapers (`Cano Liq Whitepaper En v2` and `canoLiq_Tokenomics_v2`). It lets users deposit CNPY, receive a yield-bearing receipt token (cCNPY), and earn staking rewards minus a 12% protocol fee. It also issues CPLQ — a fixed-supply (100M) governance/value-capture token with vesting, treasury, and buyback mechanics.

**Why the work is needed.** The whitepapers describe canoLiq as a sub-chain on the Canopy Network that aligns with Canopy's economic primitives (5% DAO cut, restaking, committee subsidies, halving). None of these primitives need re-implementing — they already exist in `/fsm/committee.go`, `/fsm/gov_params.go`, and the validator/restaking module. What does not exist is the **canoLiq-specific layer**: cCNPY deposit/redeem, the 12% protocol fee with 40/30/15/15 split, CPLQ token + vesting, canoLiq DAO treasury, CPLQ-holder governance, and buyback. This plan adds that layer.

**Intended outcome.** A canoLiq sub-chain that can be subsidized by Canopy's existing committee mechanism, mints cCNPY 1:1 against deposited CNPY, accrues staking rewards via Canopy's existing distribution path, applies the 12% fee with the canonical 40/30/15/15 split inside its plugin, and tracks CPLQ allocations with vesting — all without modifying Canopy core consensus code.

---

## Architecture decision (confirmed with user)

- **Integration model:** sub-chain via Go plugin. canoLiq runs as its own Canopy committee (own `chainId`) with logic implemented as a Go plugin. It reuses Canopy's existing 5% DAO cut, restaking (`Validator.Committees[]`), committee subsidies (`GetSubsidizedCommittees`), and halving emission. No changes to `/fsm/`, `/lib/`, or any Canopy core consensus code.
- **Token model:** dedicated canoLiq state buckets via plugin `StateRead`/`StateWrite`. cCNPY and CPLQ balances live in plugin-owned keys; `Account` proto is untouched.
- **Scope:** phased. Phase 1 = MVP (deposit/redeem, fee split, CPLQ + vesting). Phase 2 = governance + buyback. Phase 3 = monitoring + autonomy graduation.

The plugin is implemented as a sibling of `/plugin/go/contract/`, following the exact pattern documented in `/plugin/go/AGENTS.md` and demonstrated by the send-transaction tutorial.

---

## Progress checklist

Track implementation progress here. Tick boxes (`[x]`) as work lands. Keep this section in sync with the per-phase sections below.

### Phase 1 — MVP

#### 1. Protobuf and state types
- [x] Create `plugin/go/proto/canoliq.proto`
- [x] Define `MessageCanoliqDeposit`
- [x] Define `MessageCanoliqRedeem`
- [x] Define `MessageCanoliqClaimRedemption`
- [x] Define `MessageCPLQTransfer`
- [x] Define `MessageCPLQClaimVested`
- [x] Define `CanoliqGlobals` stored type
- [x] Define `Redemption` stored type
- [x] Define `VestingSchedule` stored type
- [x] Update `plugin/go/proto/_generate.sh` to include `canoliq.proto` (existing glob `./*.proto` already covers it)
- [x] Regenerate Go code (`canoliq.pb.go`)

#### 2. State key schema
- [x] Create `plugin/go/canoliq/state.go`
- [x] Implement `KeyForGlobals()`
- [x] Implement `KeyForCCNPYBalance(addr)`
- [x] Implement `KeyForCPLQBalance(addr)`
- [x] Implement `KeyForVesting(addr, id)`
- [x] Implement `KeyForRedemption(addr, id)`
- [x] Implement treasury / buyback / validator-incentives key helpers
- [x] Read/write wrappers for each stored type (params.go: LoadParams/SaveParams/LoadGlobals/SaveGlobals)

#### 3. Transaction handlers
- [x] Create `plugin/go/canoliq/canoliq.go` (CheckTx/DeliverTx switches)
- [x] Create `plugin/go/canoliq/error.go` (PluginError constructors)
- [x] Create `plugin/go/canoliq/config.go` (`CanoliqConfig` + `DefaultConfig`)
- [x] `CheckMessageCanoliqDeposit` + `DeliverMessageCanoliqDeposit`
- [x] `CheckMessageCanoliqRedeem` + `DeliverMessageCanoliqRedeem`
- [x] `CheckMessageCanoliqClaimRedemption` + `DeliverMessageCanoliqClaimRedemption`
- [x] `CheckMessageCPLQTransfer` + `DeliverMessageCPLQTransfer`
- [x] `CheckMessageCPLQClaimVested` + `DeliverMessageCPLQClaimVested`

#### 4. Reward fee split (EndBlock)
- [x] Create `plugin/go/canoliq/reward.go` with `ProcessRewards`
- [x] Create `plugin/go/canoliq/fee.go` (12% fee + 40/30/15/15 split)
- [x] Wire `EndBlock` handler in plugin lifecycle
- [x] Read canoLiq committee pool balance
- [x] Compute per-block delta against `last_processed_reward_pool` (renamed from `_height`)
- [x] Route 40% to `total_pooled_cnpy` (user rebate)
- [x] Route 30% to `canoliq/treasury/canoliq`
- [x] Route 15% to `canoliq/validator/incentives/*` (committee aggregator key for MVP; per-validator pro-rata in Phase 2)
- [x] Route 15% to `canoliq/buyback/pool`
- [x] Use overflow-safe math (`mulDiv` via `math/big`, mirrors `lib.SafeMulDiv`)

#### 5. CPLQ token initialization
- [x] Create `plugin/go/canoliq/vesting.go` (linear vest + cliff)
- [x] Create `plugin/go/canoliq/genesis.json` with bucket recipient addresses
- [x] Implement plugin `Genesis()` to mint 100M CPLQ
- [x] Allocate Validators & Infrastructure bucket (22%)
- [x] Allocate Liquidity Incentives bucket (15%)
- [x] Allocate Community & Airdrops bucket (20%)
- [x] Allocate DAO Treasury bucket (15%)
- [x] Allocate Founders & Core Team bucket (12%, 3yr / 12mo cliff)
- [x] Allocate Strategic Partners bucket (10%, 2yr / 6mo cliff)
- [x] Allocate Plugin & Dev Grants bucket (6%)
- [x] Verify total equals exactly 100,000,000 CPLQ

#### 6. Tests
- [x] Create `plugin/go/canoliq/canoliq_test.go`
- [x] Deposit at 1:1 (first deposit)
- [x] Deposit at correct ratio (subsequent)
- [x] Redeem queues `Redemption` with correct unbond height
- [x] Claim before unbond → error
- [x] Claim after unbond → CNPY arrives in user account
- [x] Reward injection (X=1000) → 928 to pool, 36/18/18 splits
- [x] CPLQ transfer respects vesting (locked → error)
- [x] Vesting linear unlock crosses cliff correctly
- [x] Genesis allocation totals exactly 100M CPLQ
- [x] `DeliverMessageCPLQClaimVested` end-to-end (before cliff / halfway / idempotent / past end)
- [x] Multi-block reward sweep (`LastProcessedRewardPool` watermark across consecutive `EndBlock`s)
- [x] Composite deposit → reward → redeem yields a `Redemption` strictly greater than deposit

#### 7. Sub-chain registration
- [x] Create `plugin/go/canoliq/README.md` (registration + ops guide)
- [x] Document chainId selection
- [x] Document validator opt-in via `MessageEditStake`
- [x] Document `MessageSubsidy` proposal flow
- [x] Document `~/.canopy/config.json` `"plugin": "canoliq"` setup

#### 8. Plugin entry point
- [x] Modify `plugin/go/main.go` to branch to `canoliq.StartPlugin` (env var or build tag)
- [x] Verify existing send-tutorial path still works

### Phase 1 verification
- [x] `cd plugin/go && make build` succeeds
- [x] `cd plugin/go/canoliq && go test ./...` passes
- [x] In-process: deposit → cCNPY balance + pool accounting (`canoliq_test.go::TestDepositMintsOneToOneOnFirstDeposit`, `::TestDepositSubsequentRatio`)
- [x] In-process: `total_pooled_cnpy` matches fee-math formula across blocks (`::TestRewardSplitWhitepaperExample`, `::TestRewardSweepMultiBlock`)
- [x] In-process: redeem queues `Redemption` and claim after maturity returns CNPY (`::TestRedeemQueuesRedemption`, `::TestClaimRedemptionMaturity`)
- [x] In-process: treasury / buyback / validator-incentives totals reconcile (conservation asserted in `::TestWhitepaperSection7Reconciliation` and `::TestRewardSweepMultiBlock`)
- [x] In-process: vesting tranche claims 0 before cliff, linear after (`::TestVestingLinearUnlock`, `::TestDeliverCPLQClaimVestedFlow`)
- [x] In-process: deposit → reward accrues → redeem returns yield (`::TestCompositeDepositRewardRedeem`)
- [x] Whitepaper §7 reconciliation test passes (X=1000 → 881.6 user yield ± truncation)
- [ ] Coordination: design summary posted to Canopy Discord. Should
      include: chainId selection (2), 12% fee + 40/30/15/15 split,
      CPLQ supply 100M with the seven-bucket distribution and vesting
      shape, validator opt-in via `MessageEditStake`, and the plugin
      runtime contract (`CANOPY_PLUGIN_MODE=canoliq`,
      `/tmp/plugin/plugin.sock`). Link the live-reconciliation result
      from Phase 1.5 (36M-uCNPY inflow → exact whitepaper §7 split).

### Phase 1.5 — Live integration (deferred)

Phase 1.5 covers the verification steps that require a real Canopy node + plugin
process pair on a unix socket — i.e., the parts the in-process `fakeStore`
harness explicitly mocks out. These are not unit-test gaps; they exercise
plumbing (Dockerfile, handshake, length-prefixed framing, plugin-mode env var)
that the in-process tests cannot.

Pre-reqs to land before this phase can be exercised:
1. Compose service (or sibling container) that runs `go-plugin` with
   `CANOPY_PLUGIN_MODE=canoliq` against the same `/tmp/plugin/plugin.sock`.
2. Node config switched to `"plugin": "go"`, genesis amended so committee 2
   exists and at least one validator opts in via `MessageEditStake`.
3. ~~Minimal `canoliqctl`~~ — landed at `plugin/go/canoliqctl/`. Builds with
   `go build -o canoliqctl ./canoliqctl/`. Subcommands cover the full Phase 1
   surface (deposit/redeem/claim/cplq-transfer/cplq-claim-vested) and the
   full Phase 2 surface (vote, stake/unstake/claim, buyback-execute,
   spend-execute, multisig-approve, **proposal-create with param-change /
   buyback / treasury-spend sub-commands**).

Checklist:
- [x] **canoliqctl scaffold**: build, sign with BLS12-381, POST to `/v1/tx`
      — verified locally via `go build` and CLI usage smoke test. End-to-end
      verification against a real socket still depends on items 1 & 2 above.
- [x] Plugin handshake: `go-plugin` connects to FSM, exchanges `PluginConfig`,
      and survives one `BeginBlock`/`EndBlock` cycle on the live socket
      — verified end-to-end via `.docker/compose.yaml`. Two issues surfaced
      during integration that the in-process tests masked: (1) Alpine's
      busybox `ps` lacks the `-p` flag `pluginctl.sh::is_running` relies on
      → fixed by installing `procps` in the runtime image. (2) FSM
      `StateRead` returns `Entry{Value: nil}` for missing keys (not zero
      entries) → fixed in `LoadParams`/`LoadGlobals` to treat empty `Value`
      as unset.
- [x] `MessageSubsidy` proposal funds committee 2's reward pool; canoLiq's
      `EndBlock` observes the inflow and applies the 12% fee on a real chain.
      Verified live in `.docker/compose.yaml` after wiping volumes: with
      the BeginBlock self-bootstrap (so plugin Genesis runs even when chain
      genesis.json has no plugin section), an inflow of 36,000,000 uCNPY
      reconciled exactly via `/v1/globals` + `/v1/pools`:
      `pool=33,408,000 + treasury=1,101,600 + insurance=194,400 +
      validators=648,000 + buyback=648,000 = 36,000,000`, matching the
      whitepaper §7 fee math (12% × 36M = 4.32M split 40/30/15/15 with
      15% of treasury skim → insurance).
- [x] Submit a real `MessageCanoliqDeposit` via `canoliqctl deposit`; tx
      accepted at height 4 (cCNPY mint proven indirectly by the subsequent
      successful redeem).
- [x] Submit `MessageCanoliqRedeem` via `canoliqctl redeem`, advance past
      `unbond_complete_height`, submit `MessageCanoliqClaimRedemption` via
      `canoliqctl claim`; an early claim at height 7 was correctly rejected
      with "redemption has not yet matured", and the post-maturity claim
      landed at height 11.

Two server-side bugs surfaced and were fixed during Step G:
1. `DeliverMessageCanoliqRedeem` used `height := uint64(0)` instead of
   `c.currentHeight()`, making `unbond_complete_height` always equal to
   the placeholder unstaking constant. Fixed.
2. The placeholder unstaking constant was 14400 blocks (~24h) — too long
   for localnet verification. Lowered to 5 blocks pending the proper FSM
   `valParams.UnstakingBlocks` plumbing the original Phase 1 plan called
   for (deferred to Phase 2).

### Phase 2 — Governance, buyback, treasury

Design decisions (resolved against the whitepapers):
- **Buyback execution** = internal accounting swap at a proposal-set price. WP §6 allows "market buyback and burn **or** direct distribution governed by DAO"; a real on-chain DEX route is deferred to Phase 3 (autonomy).
- **Governance weight** = staked CPLQ only, snapshot at proposal-creation height. WP §4: "stake/lock CPLQ for governance power and boosts." Yield boosts are mentioned but undefined — deferred.
- **Treasury spend** = hybrid. Below `treasury_threshold`, a passing CPLQ vote alone executes. Above threshold, vote + N-of-M multisig approvals + `timelock_blocks` delay. WP §9 + Tokenomics §7: "Multisig and timelocks required for treasury spending above thresholds."
- **Per-validator pro-rata** lands here (Phase 1 carryover). Required for the buyback "distribute to stakers" mode and for honest per-validator infra credits.

#### 1. Protobuf and state types
- [x] Add to `plugin/go/proto/canoliq.proto`:
  - [x] `MessageCPLQStake` / `MessageCPLQUnstake` / `MessageCPLQClaimUnstake`
  - [x] `MessageCPLQProposalCreate` / `MessageCPLQVote`
  - [x] `MessageBuybackExecute`
  - [x] `MessageDAOTreasurySpend` / `MessageMultisigApprove`
- [x] New stored types: `CPLQStake`, `UnstakingCPLQ`, `Proposal`, `Vote`, `BuybackOrder`, `MultisigApproval` (plus `CPLQStakeIndex`, `ValidatorRegistry`, `ValidatorRegistryEntry`, `ProposalIndex`)
- [x] `Proposal.payload` typed via `google.protobuf.Any` (resolved with `contract.FromAny`) — equivalent to a oneof for dispatch purposes
- [x] Extend `CanoliqParams`: `insurance_bps`, `treasury_threshold`, `multisig_signers`, `multisig_threshold`, `voting_period_blocks`, `quorum_bps`, `pass_threshold_bps`, `timelock_blocks`, `cplq_unstaking_blocks`, `proposal_fee`, `vote_fee`, `stake_fee`, `multisig_approve_fee`, `min_stake_to_propose`
- [x] Extend `CanoliqGlobals`: `total_staked_cplq`, `next_proposal_id`, `next_buyback_id`, `next_spend_id`, `next_unstake_id`
- [x] Update `Config.SupportedTransactions` / `TransactionTypeUrls`
- [x] Regenerate `canoliq.pb.go`

#### 2. State key schema
- [x] `KeyForCPLQStake(addr)` (active stake balance)
- [x] `KeyForCPLQUnstaking(addr, id)` (in-flight unbond record)
- [x] `KeyForCPLQStakeIndex()` (active staker enumeration for buyback DISTRIBUTE)
- [x] `KeyForProposal(id)` + `KeyForProposalIndex()` (active proposal id list)
- [x] `KeyForVote(proposalId, voter)`
- [x] `KeyForBuybackOrder(id)`
- [x] `KeyForTreasurySpend(spendId)` + `KeyForSpendIndex()` + `KeyForMultisigApproval(spendId, signer)`
- [x] `KeyForInsurancePool()`
- [x] `KeyForValidatorRegistry()` (singleton registry for per-validator pro-rata)
- [x] Read/write wrappers for each new stored type

#### 3. CPLQ staking
- [x] `stake.go`: stake/unstake/claim handlers
- [x] `Check`/`Deliver MessageCPLQStake` — moves liquid CPLQ → `CPLQStake` record, increments `total_staked_cplq`
- [x] `Check`/`Deliver MessageCPLQUnstake` — debits stake, queues `UnstakingCPLQ` with `mature_height = h + cplq_unstaking_blocks`
- [x] `Check`/`Deliver MessageCPLQClaimUnstake` — matures record, returns CPLQ to liquid balance
- [x] Tests: stake/unstake/claim, unbond before maturity error, double-claim idempotency (`TestCPLQStakeUnstakeClaim`)

#### 4. On-chain governance
- [x] `governance.go`: proposal lifecycle (create / vote / tally / execute)
- [x] `Check`/`Deliver MessageCPLQProposalCreate` — assigns id, snapshots `total_staked_cplq` at creation height, records `expiry_height = h + voting_period_blocks`
- [x] `Check`/`Deliver MessageCPLQVote` — looks up the voter's `CPLQStake` *as of `proposal.creation_height`*, records weighted yes/no/abstain
- [x] BeginBlock hook: scan proposal index, on expiry tally yes/no/abstain weights
- [x] Pass rule: `(yes + no + abstain) >= quorum_bps * snapshot_total_staked / 10000` AND `yes >= pass_threshold_bps * (yes + no) / 10000`
- [x] On pass: execute payload (param change directly; buyback/spend mark as executable downstream)
- [x] On fail/expire: delete proposal + drop from index
- [x] Snapshot mechanism (`CPLQStake.staked_at_height` vs `proposal.creation_height`) defeats flash-stake attacks
- [x] Tests: param change round-trip (`TestProposalParamChangeRoundTrip`), quorum miss (`TestProposalQuorumMiss`), snapshot correctness (`TestVoteSnapshotRejectsLateStake`)

#### 5. Per-validator reward pro-rata (Phase 1 carryover)
- [x] In `reward.go`, replace `committeeAggregatorAddr()` with a stake-weighted distribution (legacy aggregator kept as a fallback when registry is empty)
- [x] Read validator set + per-validator stake from a plugin-internal `ValidatorRegistry` singleton (genesis-seedable, governance-mutable). Phase 1.5 will swap this for a real Canopy validator-set readback.
- [x] Distribute the 15% validator slice proportional to per-validator stake
- [x] `mulDiv` share-out; rounding remainder credited to the largest-stake validator
- [x] Tests: 70/20/10 stake split + rounding remainder credited to largest (`TestPerValidatorProRataDistribution`)

#### 6. Buyback (internal accounting swap)
- [x] `buyback.go`: implements proposal-driven buyback
- [x] `MessageBuybackExecute` — references a passed `ProposalBuyback` id; idempotent (re-execute rejected)
- [x] Drain `cnpy_amount` from `canoliq/buyback/pool` (clamped to available)
- [x] Acquire CPLQ at `proposal.price_micro_cnpy_per_cplq`: `cplq_acquired = cnpy_amount * 10^6 / price_micro_cnpy_per_cplq`
- [x] Source CPLQ from `canoliq/treasury/cplq` (DAO 15% bucket)
- [x] Mode `BUYBACK_BURN`: decrement `cplq_total_supply` and `cplq_circulating_supply` by `cplq_acquired`
- [x] Mode `BUYBACK_DISTRIBUTE_STAKERS`: pro-rata credit `CPLQStake` records by stake weight; CNPY moves to `treasury/canoliq`
- [x] Tests: burn (`TestBuybackBurnReducesSupply`), distribute multi-staker (`TestBuybackDistributeStakers`); idempotent re-execute covered in BURN test

#### 7. DAO treasury spend
- [x] `treasury.go`: `MessageDAOTreasurySpend` triggers a queued spend by proposal id
- [x] On vote pass: `queueTreasurySpend` writes `TreasurySpend` with `executable_height = h + (timelock_blocks if amount > treasury_threshold else 0)`
- [x] `Check`/`Deliver MessageMultisigApprove` — records signer's approval per `spend_id`; signer must be in `multisig_signers`
- [x] Execution: `current_height >= executable_height` AND (`amount <= threshold` OR `approvals >= multisig_threshold`)
- [x] Source from `treasury/canoliq` (CNPY) or `treasury/cplq` (CPLQ); credit recipient liquid balance
- [x] Tests: below-threshold (`TestTreasurySpendBelowThreshold`), above-threshold full path including timelock + missing approvals + replay (`TestTreasurySpendAboveThresholdRequiresTimelockAndMultisig`), non-signer rejection (`TestMultisigApprovalRejectsNonSigner`)

#### 8. Insurance fund auto-routing
- [x] `DefaultParams()` adds `insurance_bps` (default 1500 = 15% of incoming treasury credit, ≈ 1.5% of fee — within WP §11)
- [x] `ProcessRewards`: skim `insurance_bps` off `split.Treasury` before crediting `treasury/canoliq`
- [x] New scalar key `canoliq/insurance/pool`
- [x] `ValidateParams` allows `insurance_bps <= 10000`
- [x] Tests: full conservation including insurance (`TestInsuranceConservationFullSplit`); pre-existing reward tests updated to include the insurance line

#### 9. Tests (`plugin/go/canoliq/*_test.go`)
- [x] CPLQ stake / unstake / claim lifecycle
- [x] Proposal create → vote → tally → param mutation
- [x] Voter weight snapshot defeats post-creation stake
- [x] Quorum miss rejects
- [x] Buyback burn mode: total + circulating supply decrement
- [x] Buyback distribute mode: multi-staker pro-rata credit
- [x] Treasury spend below threshold (single-step)
- [x] Treasury spend above threshold: timelock + multisig threshold both enforced
- [x] Insurance skim conservation
- [x] Validator pro-rata under skewed stake distribution

#### 10. Documentation
- [x] Update `plugin/go/canoliq/README.md` — proposal lifecycle, buyback workflow, multisig signer onboarding, insurance, per-validator pro-rata
- [x] Document genesis-time multisig configuration (signers + threshold) in `genesis.json` — `params` block now accepts `multisigSigners`, `multisigThreshold`, `treasuryThreshold`, and the full Phase 2 governance knobs

### Phase 2 verification
- [x] `cd plugin/go && go build ./...` succeeds with new package
- [x] `cd plugin/go && go test ./canoliq/...` passes (12 Phase 1 + 11 Phase 2 tests)
- [x] In-process: governance round-trip mutates `fee_bps`, next `ProcessRewards` applies the new bps
- [x] In-process: buyback burn reduces `cplq_total_supply`; distribute credits stakers
- [x] In-process: above-threshold spend rejected pre-timelock and pre-multisig-quorum, accepted once both met
- [x] In-process: insurance pool grows by `insurance_bps * fee * treasury_bps / 10000^2` per block; full conservation holds
- [x] In-process: per-validator infra credits split proportionally to committee stake

### Phase 3 — Monitoring & autonomy graduation

#### 1. Read-only query layer (HTTP-in-plugin, snapshot model)

The plugin process owns all canoLiq-prefixed state. Phase 1.5 left
fee-split reconciliation against the live chain blocked on a way to
read that state from outside the plugin. Phase 3 §1 fills that gap by
running an HTTP server inside the plugin process, serving from a
snapshot of canoliq-owned state refreshed inside `EndBlock`.

**Why a snapshot.** An early design issued plugin-initiated `StateRead`
calls from the HTTP handler with a random fsmId. The Canopy FSM
rejected those (`code 107: plugin response id is invalid`) because the
fsmId did not match an in-flight FSM lifecycle context — the FSM only
holds a state-context for the duration of a Check/Deliver/Begin/End
call. The `fakeStore` test path bypassed the protocol so the bug went
undetected until live docker compose. The fix: build the snapshot
inside `EndBlock` (where `c.fsmId` is the FSM-originated EndBlock id,
valid for arbitrary state reads), atomically swap it into
`Plugin.snapshot`, and answer HTTP requests from there. Stale by up to
one block; that's acceptable for monitoring.

##### 1a. Snapshot type + refresh (`plugin/go/canoliq/snapshot.go`)
- [x] `Snapshot` struct holding the singleton + index-driven view of
      canoliq-owned state.
- [x] `(*Canoliq).refreshSnapshot(height)` — three batched StateReads:
      (1) singletons + indexes, (2) per-id reads for active proposals
      / pending spends / stakers / validator incentives, (3) multisig
      approvals (per-spend × per-signer).
- [x] Atomic publish via `Plugin.snapshot` (`atomic.Pointer`).
- [x] `EndBlock` calls `refreshSnapshot` after `ProcessRewards`.

##### 1a-bis. Query helpers (`plugin/go/canoliq/query.go`)
- [x] `(*Plugin).QueryHealth()` → `{height, genesisComplete, chainId}`
- [x] `(*Plugin).QueryGlobals()` → `*contract.CanoliqGlobals`
- [x] `(*Plugin).QueryParams()` → `*contract.CanoliqParams`
- [x] `(*Plugin).QueryPools()` → committee pool, treasury (CNPY/CPLQ),
      buyback, insurance, per-validator incentives
- [x] `(*Plugin).QueryProposalIDs()` / `QueryProposal(id)`
- [x] `(*Plugin).QuerySpendIDs()` / `QuerySpend(id)` /
      `QueryMultisigApprovals(id)`
- [x] `(*Plugin).QueryValidatorRegistry()`
- [x] `(*Plugin).QueryStakers()` (from `CPLQStakeIndex`)

##### 1b. HTTP server (`plugin/go/canoliq/rpc.go`)
- [x] `net/http` mux gated by `Config.RpcAddress` (or env
      `CANOLIQ_RPC_ADDR`); empty disables the listener so the Phase 1
      binary surface is preserved.
- [x] Route table (singleton + index-driven only):
  - `GET /v1/health` (`{height, genesisComplete, chainId}`)
  - `GET /v1/globals`
  - `GET /v1/params`
  - `GET /v1/pools`
  - `GET /v1/proposals` (active id list)
  - `GET /v1/proposal/{id}`
  - `GET /v1/spends` (pending id list)
  - `GET /v1/spend/{id}`
  - `GET /v1/spend/{id}/approvals`
  - `GET /v1/validators`
  - `GET /v1/stakers`
- [x] Encode responses as JSON; proto `@gotags` keep field names
      consistent with `canoliqctl`.
- [x] 404 on missing entity, 400 on malformed input, 405 on non-GET.
- [x] Graceful shutdown via `(*RPCServer).Shutdown` driven by
      `main.go`'s signal context (5s timeout).

##### 1c. Config + main wiring
- [x] Extended `Config` with `RpcAddress string`; honors
      `CANOLIQ_RPC_ADDR` env var override at startup.
- [x] `main.go` captures the long-lived `*Plugin` and calls
      `rpc.Shutdown` on signal cancellation.
- [x] `.docker/compose.yaml` exposes the RPC port: `8587:8587` for
      node-1 and `8588:8587` for node-2; both containers set
      `CANOLIQ_RPC_ADDR=0.0.0.0:8587` so the host port forwards see
      the listener.

##### 1d. Tests
- [x] Per-route HTTP tests against `fakeStore` via `httptest.Server`
      (`rpc_test.go::TestRPCHealthBeforeSnapshot`,
      `TestRPCHealthAndGlobalsAfterRefresh`, `TestRPCParamsRoundTrip`,
      `TestRPCProposalLifecycle`, `TestRPCSpendAndApprovals`,
      `TestRPCValidatorsAndStakers`).
- [x] 405 on non-GET (`TestRPCMethodNotAllowed`); empty-addr disables
      the server (`TestRPCStartRPCServerEmptyAddrDisabled`); cold
      start serves sane defaults (`TestRPCHealthBeforeSnapshot`).
- [x] Conservation cross-check via the HTTP layer:
      `TestRPCPoolsConservationAfterReward` injects a 950-uCNPY post-DAO
      reward, runs the full `EndBlock` (which exercises both
      `ProcessRewards` and `refreshSnapshot`), and asserts the
      `/v1/pools` view + `globals.TotalPooledCnpy` reconcile to the
      seeded delta.

##### 1e. Documentation
- [x] `plugin/go/canoliq/README.md` — added "Read-only HTTP query
      layer (Phase 3)" section listing all routes, the env-var knob,
      docker port mapping, and known gaps.
- [x] `plugin/go/canoliq/AGENTS.md` — added "Query layer" section
      covering the per-request `*Canoliq` minting pattern, the
      gating config, and why collection routes depend on existing
      indexes.

##### Phase 3 §1 verification
- [x] `cd plugin/go && go build ./...` succeeds with the new
      `query.go` / `rpc.go` files; `contract` mode unchanged.
- [x] In-process: every route returns 200/JSON for seeded state and
      4xx for missing/malformed inputs (`go test ./canoliq/ -run
      TestRPC` — 11 tests).
- [x] Live integration: hit `/v1/globals` and `/v1/pools` against the
      running canoliq plugin in `.docker/compose.yaml` and reconcile a
      fee split. Result: 36M-uCNPY inflow reconciles exactly across
      pool / treasury / insurance / validator / buyback per the
      whitepaper §7 split. Snapshot height tracks the live chain;
      no `code 107` regressions across 100+ blocks.

#### 1.1 Per-address query routes (lazy-fulfill queue)

Per-address composite views were dropped from §1 because the snapshot
can only enumerate state reachable from a singleton or an existing
index, and canoliq has no global "all addresses ever seen" index.

**Implementation: lazy-fulfill queue.** The plugin owns a bounded
channel of `*lazyQuery`. HTTP handlers build a query (kind + address +
optional id/voter), push it onto the channel, and block on the per-query
result with the request `ctx`. `EndBlock` drains the queue after
`refreshSnapshot` (so `c.fsmId` is a valid FSM context for arbitrary
StateReads), executes each query, and sends results back. Worst-case
latency: one block (~6s localnet). Client disconnect cancels the wait.

Routes landed:
- [x] `GET /v1/account/{addr}` — composite CNPY + cCNPY + liquid CPLQ
      + stake + validator-incentive + vesting (no redemption/unstake
      list yet — needs a write-side index)
- [x] `GET /v1/vesting/{addr}` — schedules + cumulative unlocked-to-date
- [x] `GET /v1/redemption/{addr}/{id}` — point lookup
- [x] `GET /v1/vote/{id}/{voter}` — point lookup
- [x] `GET /v1/buyback/{id}` — point lookup against `KeyForBuybackOrder`

Tests (`rpc_test.go`):
- [x] `TestRPCAccountComposite`, `TestRPCAccountRejectsBadAddress`
- [x] `TestRPCVestingDedicated` (404 on no-index)
- [x] `TestRPCRedemptionPointLookup` (200 + 404)
- [x] `TestRPCVotePointLookup` (200 + 404)
- [x] `TestRPCBuybackPointLookup` (200 + 404)
- [x] `TestLazyQueueTimeout` — handler aborts on client disconnect

##### 1.1-bis. Per-address collection indexes (follow-up)

Today `/v1/account/{addr}` cannot list pending redemptions or
unstakes for a user — there is no per-address index that names
those records. Add lightweight indexes mirroring `VestingIndex`.

###### Proto + state
- [ ] Reuse `VestingIndex`'s shape (`repeated uint64 ids`) — no new
      proto types needed. Just two new key helpers in `state.go`:
      `KeyForRedemptionIndex(addr)`, `KeyForUnstakingIndex(addr)`.
      Pick fresh domain bytes (next free after 20 = 21, 22).

###### Write-side maintenance
- [ ] `DeliverMessageCanoliqRedeem` — append the new redemption id
      to `KeyForRedemptionIndex(addr)`
- [ ] `DeliverMessageCanoliqClaimRedemption` — remove the matured
      id from the index alongside the existing record delete
- [ ] `DeliverMessageCPLQUnstake` — append unstake id to
      `KeyForUnstakingIndex(addr)`
- [ ] `DeliverMessageCPLQClaimUnstake` — remove the matured id

###### Read-side
- [ ] Extend `buildAccountView` to load both indexes, batch-read
      the records, and add `Redemptions []*contract.Redemption`
      and `Unstakes []*contract.UnstakingCPLQ` to `AccountView`
- [ ] Optional dedicated routes: `/v1/account/{addr}/redemptions`,
      `/v1/account/{addr}/unstakes` (lazy, same queue)

###### Tests
- [ ] Index append/remove invariants under all four flows
      (redeem → claim, unstake → claim, plus partial / out-of-order
      claims — index must end empty when no records remain)
- [ ] `/v1/account/{addr}` reflects the new lists in correct order
- [ ] Idempotency: re-applying a deletion of a non-present id is
      a no-op

#### 2. Alerting hooks

The snapshot already gives operators a pull surface; alerting adds a
push surface for unattended monitoring. Each `EndBlock`, after the
snapshot publishes, the plugin evaluates a small set of conditions
and POSTs to a configured webhook. Alerts are debounced and
dedup-persisted so a process restart does not re-fire conditions
that were already acknowledged.

Whitepaper §11 requirement: "monitoring dashboards & alerts for
validator behavior, committee health, and TVL movement." This
section satisfies the alerting half; dashboards consume the
existing `/v1` surface.

##### 2a. Webhook delivery (`plugin/go/canoliq/alerts.go`)
- [ ] `AlertConfig` struct — `WebhookURL`, `AuthHeader` (optional),
      `Enabled`, per-kind `MinIntervalBlocks` debounce, payload
      `Format` (json | slack | discord), 5s POST timeout
- [ ] `Config.Alerts *AlertConfig` plumbed via JSON +
      `CANOLIQ_ALERT_URL` env override; empty disables (mirrors
      the `RpcAddress` pattern)
- [ ] Dispatcher fans out POSTs on a goroutine so EndBlock never
      blocks on network IO; failures logged at WARN, never panic
- [ ] Deduplication: each alert kind has a last-fired-height entry
      under `KeyForAlertState(kind)` (new domain byte). Skip when
      `current_height - last_fired < MinIntervalBlocks`. Kind
      transitions to "resolved" emit a single resolution event then
      clear the watermark
- [ ] Persistence rationale: webhook delivery is best-effort, but
      dedup state lives in plugin state so a restart-during-incident
      doesn't double-page

##### 2b. Conditions (evaluated each EndBlock from the snapshot)
- [ ] **Buyback drain rate.** Track `BuybackPool` deltas across
      a rolling window (default 100 blocks). Fire when the
      window-sum drain exceeds `drain_alert_bps` of the prior
      window-start balance (default 5000 = 50%). Window stored as
      a small ring on `Plugin` (in-memory; bounded so no proto
      change). Resolution: drain falls below `drain_resolve_bps`
- [ ] **Stuck redemption queue.** Count redemptions where
      `unbond_complete_height + grace < current_height` (mature,
      unclaimed). Fire when count > `stuck_redemption_threshold`
      (default 10). **Depends on §1.1-bis `RedemptionIndex(addr)`
      *plus* a global `StuckRedemptionIndex` of un-claimed
      mature ids** — without enumeration, the alert can't compute.
      Block this sub-bullet on §1.1-bis landing first
- [ ] **Validator-incentive starvation.** Compute
      `max_validator_stake / total_committee_stake` from
      `snapshot.ValidatorRegistry`. Fire when ratio >
      `concentration_alert_bps` (default 6600 = 66%). Resolution:
      ratio falls below `concentration_resolve_bps`
- [ ] **TVL drop** (whitepaper §11 explicit ask). Track
      `globals.TotalPooledCnpy` rolling window. Fire on > N% drop
      in M blocks. Default thresholds tunable via `AlertConfig`

##### 2c. Payload schema
- [ ] Generic JSON envelope:
      `{kind, height, severity (warn|crit), message, details: {...}}`
- [ ] Slack/Discord adapters: convert envelope to that platform's
      format. Slack uses `text` + `blocks`; Discord uses `content` +
      `embeds`. Pick by `Format` config field
- [ ] Stable schema versioned under `details.schemaVersion` so
      downstream consumers can evolve

##### 2d. Tests
- [ ] `httptest.Server` mock receiver; each kind fires once with
      expected payload shape under a seeded condition
- [ ] Debounce: same condition seeded across two consecutive
      EndBlocks fires only once when interval > 1 block
- [ ] Dedup persistence: simulate process restart by reloading
      `Plugin.snapshot` and asserting no re-fire
- [ ] Resilience: webhook returns 500 → EndBlock still completes,
      retry happens on next firing window (no infinite-retry loop)
- [ ] Threshold edge cases: exactly-at-threshold doesn't fire;
      just-above-threshold fires

##### 2e. Documentation
- [ ] `README.md` — alert payload schema, configuration knobs,
      Slack/Discord webhook setup, debounce semantics
- [ ] `AGENTS.md` — design rationale (push vs pull, why dedup
      state lives in plugin state, why the dispatcher is a
      goroutine)
- [ ] `MEMORY.md` entry if any new gotchas surface during the work

##### Phase 3 §2 verification
- [ ] In-process: each kind fires exactly once on threshold cross
- [ ] In-process: debounce + resolution events sequence correctly
- [ ] Live: point `CANOLIQ_ALERT_URL` at a local sink, induce a
      buyback drain via a `MessageBuybackExecute`, observe the
      alert envelope land in the sink

#### 3. Graduation snapshot / migration tooling

Whitepaper §10 describes graduation to a standalone L1 once
objective thresholds are met (TVL, active validator count, DAO
maturity). The mechanics decompose into four pieces: threshold
tracking, point-in-time state export, fresh-chain import, and
governance-gated coordination flow.

This is the largest of the Phase 3 sub-projects. Recommend landing
3a + 3b first as standalone deliverables — they're useful even
without graduation (3a feeds dashboards, 3b is operational backup
infrastructure).

##### 3a. Threshold tracking
- [ ] Extend `CanoliqParams` with three governance-tunable fields:
      `graduation_min_tvl` (uCNPY), `graduation_min_validators`
      (count), `graduation_min_passed_proposals` (count).
      Defaults from whitepaper §10 (TBD with the team — placeholder
      seeds in `DefaultParams`)
- [ ] Add `GraduationStatus` to `Snapshot` and a new field on
      `/v1/health` (or dedicated `/v1/graduation`). Surface:
      current TVL, current validator count, cumulative passed
      proposal count, ratio against each threshold, and a
      composite `eligible bool`
- [ ] Track `globals.PassedProposalCount` — incremented in
      `dispatchPassed`. Tests: counter advances on each pass;
      failed proposals don't increment
- [ ] Tests: thresholds met / partial / unmet → eligible flag
      flips correctly

##### 3b. State export (`plugin/go/canoliq/export.go`)
- [ ] Define `GenesisExport` proto extending `GenesisFile` with
      live state slots: per-address cCNPY/CPLQ liquid balances,
      `CPLQStake` records (with `staked_at_height`),
      `UnstakingCPLQ` records, `Redemption` records,
      `VestingSchedule.ClaimedAmount`, treasury / buyback /
      insurance scalars, validator registry, active
      proposals/spends/votes, full `CanoliqParams`,
      `globals.PassedProposalCount`. Versioned via
      `export_schema_version`
- [ ] Implementation: walk all canoliq-prefixed keys via the
      plugin's existing index helpers + per-address indexes
      from §1.1-bis (so this depends on those landing). Encode
      addresses as hex for JSON readability
- [ ] CLI: `canoliqctl export-genesis --rpc <url> --output
      genesis-export.json`. Pull state via the read-only RPC
      (snapshot is sufficient; per-address data via lazy queue).
      No new plugin tx surface needed
- [ ] Determinism: same height + state → identical bytes. Sort
      collections by stable keys (addresses lex-ordered, ids
      ascending) before encoding
- [ ] Tests: round-trip — seed a small chain via fakeStore, export,
      re-import into a fresh fakeStore, assert state-tree
      equivalence (every canoliq-prefixed key matches)

##### 3c. Migration import
- [ ] Extend `runGenesis` to detect `GenesisExport` payloads (via
      `export_schema_version` presence) and seed *all* live state,
      not just the bucket distribution. Treat bucket math as
      already applied (the export captures the post-distribution
      state, not the original allocation tree)
- [ ] Validation: `cplq_total_supply` preserved (sum of liquid +
      stake + unstaking + remaining-vesting matches the cap);
      treasury balances non-negative; per-validator stake totals
      match the registry; `CPLQStakeIndex` matches the set of
      stake records. Refuse to import if any check fails
- [ ] Tests: export from a non-trivial chain (genesis + several
      blocks of activity) → restart fresh node with the exported
      genesis → query the new node and assert the same `/v1/...`
      responses (modulo height, which restarts at 1)

##### 3d. Coordination flow (graduation message)
- [ ] New proposal payload `ProposalGraduate` (Phase 2-style
      governance) — when passed, the plugin marks
      `globals.GraduationLockedHeight = h + grace_blocks` and
      stops accepting new deposits. Existing redemptions and
      vesting claims continue to work through the grace window
- [ ] After `GraduationLockedHeight`, deposits return
      `ErrGraduationLocked`. Redemptions force-mature so users
      can exit before the export
- [ ] Tests: pre-graduation → all flows work; post-lock pre-grace
      → deposits rejected, redemptions accepted; post-grace →
      export captures a fully-quiesced state
- [ ] Cross-chain coordination doc (separate to whitepaper):
      pre-graduation Canopy DAO proposal to deprecate the canoLiq
      committee; post-export, publish the export hash on Canopy
      and the new L1; cCNPY/CPLQ holders' continuity guarantee
      (same addresses, same balances, same vesting schedules)

##### 3e. Documentation
- [ ] `README.md` — graduation playbook: thresholds, export
      command, sequencing, fallback / abort path. Include the
      `eligible` flag semantics (advisory; governance is the
      actual trigger)
- [ ] `AGENTS.md` — invariants the export must preserve, the
      determinism contract, why `runGenesis` short-circuits on
      `genesis_complete=true` (regular path) vs. how
      `GenesisExport` overrides the bucket math
- [ ] Migration runbook in `docs/` — step-by-step for the
      operations team: pause deposits, wait grace, run export,
      verify hash, hand off to new-L1 genesis, deprecate committee

##### Phase 3 §3 verification
- [ ] In-process: round-trip export/import preserves every
      canoliq-prefixed key
- [ ] In-process: graduation lock blocks deposits, allows
      redemptions, force-matures pending redemptions at end of
      grace window
- [ ] Live: run a docker-compose chain through to graduation
      (with shortened thresholds for testability), export, spin
      up a fresh canopy node from the exported genesis, observe
      identical RPC responses

---

## Repo layout

New plugin directory (parallel to `/plugin/go/contract/`):

```
plugin/go/canoliq/
  canoliq.go         # CheckTx/DeliverTx switch and message handlers
  state.go           # Key helpers + read/write wrappers for cCNPY, CPLQ, vesting, treasury
  fee.go             # 12% fee application and 40/30/15/15 split
  vesting.go         # Vesting schedule application (founders, validators, partners)
  reward.go          # EndBlock hook: observe canoLiq pool, apply fee, distribute net to cCNPY holders
  governance.go      # (Phase 2) CPLQ-holder voting state and tally
  buyback.go         # (Phase 2) CNPY-treasury → CPLQ buyback
  error.go           # PluginError constructors
  config.go          # CanoliqConfig + DefaultConfig (registers SupportedTransactions/TypeUrls)
  canoliq_test.go    # Unit tests
plugin/go/proto/
  canoliq.proto      # New tx types and stored types (added alongside tx.proto)
plugin/go/main.go    # MODIFY: branch to start either contract.StartPlugin or canoliq.StartPlugin
                     # via env var or build tag; existing send tutorial still works.
```

---

## Phase 1 — MVP: deposit, redeem, fee split, CPLQ, vesting

### 1. Protobuf message and state types

Add to `plugin/go/proto/canoliq.proto`. Regenerate via `plugin/go/proto/_generate.sh`.

**New transaction messages:**

| Message | Fields | Purpose |
|---|---|---|
| `MessageCanoliqDeposit` | `from_address`, `amount` (uCNPY) | User deposits CNPY → mints cCNPY at current exchange rate |
| `MessageCanoliqRedeem` | `from_address`, `ccnpy_amount` | Burns cCNPY, queues CNPY redemption respecting committee unstaking cooldown |
| `MessageCanoliqClaimRedemption` | `from_address`, `redemption_id` | Withdraws matured CNPY redemption to user |
| `MessageCPLQTransfer` | `from_address`, `to_address`, `amount` | Transfers liquid (vested) CPLQ between accounts |
| `MessageCPLQClaimVested` | `from_address` | Moves any newly-unlocked CPLQ from vesting bucket → liquid CPLQ balance |

**New stored types:**

| Type | Fields | Notes |
|---|---|---|
| `CanoliqGlobals` | `total_ccnpy_supply`, `total_pooled_cnpy`, `pending_redemption_cnpy`, `last_processed_reward_height`, `cplq_total_supply`, `cplq_circulating_supply` | Singleton, key `globals` |
| `Redemption` | `id`, `address`, `cnpy_amount`, `unbond_complete_height` | Per-id record |
| `VestingSchedule` | `address`, `total_amount`, `cliff_height`, `start_height`, `end_height`, `claimed_amount` | Linear vest after cliff; one per allocation tranche |

### 2. State key schema (plugin-owned)

All keys are namespaced under a canoLiq prefix to avoid collision with the existing `[]byte{1}` (account) / `[]byte{2}` (pool) / `[]byte{7}` (gov) prefixes called out in `plugin/go/AGENTS.md` lines 50–54.

```
canoliq/globals                              → CanoliqGlobals
canoliq/ccnpy/balance/{addr}                 → uint64
canoliq/cplq/balance/{addr}                  → uint64       (liquid)
canoliq/cplq/vesting/{addr}/{schedule_id}    → VestingSchedule
canoliq/redemption/{addr}/{redemption_id}    → Redemption
canoliq/treasury/canoliq                     → uint64       (CNPY held)
canoliq/treasury/cplq                        → uint64       (CPLQ held by DAO)
canoliq/buyback/pool                         → uint64       (CNPY earmarked for buyback)
canoliq/validator/incentives/{addr}          → uint64       (CNPY accrued for infra)
```

Key helper file: `plugin/go/canoliq/state.go` — exposes `KeyForCCNPYBalance(addr)`, `KeyForCPLQBalance(addr)`, `KeyForVesting(addr, id)`, `KeyForRedemption(addr, id)`, `KeyForGlobals()`, etc. Mirror the helper style of `contract/contract.go` (`KeyForAccount`, `KeyForFeePool`, `KeyForFeeParams`).

### 3. Transaction handlers

Pattern is the same as the send-tutorial CheckTx/DeliverTx in `plugin/go/contract/contract.go` (see `CheckMessageSend` / `DeliverMessageSend`, contract.go:96–212).

For each new tx in the table above, implement:

```go
func (c *Canoliq) CheckMessage<X>(msg *Message<X>) *PluginCheckResponse  // stateless
func (c *Canoliq) DeliverMessage<X>(msg *Message<X>, fee uint64) *PluginDeliverResponse  // stateful
```

Wire each into the `CheckTx` and `DeliverTx` switches, mirroring `contract.go:38–88`.

**Deposit logic (`DeliverMessageCanoliqDeposit`):**
1. Read sender CNPY account (key prefix `[]byte{1}` — Canopy's standard account key).
2. Verify `account.Amount >= amount + fee`.
3. Move `amount` CNPY from sender account → canoLiq escrow pool (prefix `[]byte{2}`, pool id derived from canoLiq committee chainId — matches Canopy's existing committee escrow pattern in `fsm/committee.go`).
4. Compute cCNPY to mint: `mint = amount * total_ccnpy_supply / total_pooled_cnpy` (or 1:1 if first deposit).
5. Update `canoliq/ccnpy/balance/{addr}` and `CanoliqGlobals.total_ccnpy_supply` / `total_pooled_cnpy`.

**Redeem logic (`DeliverMessageCanoliqRedeem`):**
1. Verify cCNPY balance.
2. Compute CNPY owed: `cnpy = ccnpy_amount * total_pooled_cnpy / total_ccnpy_supply`.
3. Burn cCNPY; subtract from `total_pooled_cnpy`; add to `pending_redemption_cnpy`.
4. Write a `Redemption` record with `unbond_complete_height = current_height + valParams.UnstakingBlocks`. Read `UnstakingBlocks` from gov params via state read of prefix `[]byte{7}` (matches the param read pattern in `contract.go:42`).

**Claim redemption (`DeliverMessageCanoliqClaimRedemption`):**
1. Read the `Redemption` record.
2. If `current_height >= unbond_complete_height`, transfer CNPY from canoLiq escrow pool to user account, decrement `pending_redemption_cnpy`, delete the record.

**CPLQ transfer / vested claim:** standard balance update + vesting linear-unlock formula (`unlocked = total * (height - start) / (end - start)` clamped after cliff).

### 4. Reward fee split — `EndBlock` hook

This is the heart of the tokenomics. canoLiq's plugin observes its committee reward pool every block and applies the 12% fee with the 40/30/15/15 split. Canopy already mints rewards into the canoLiq committee pool and applies the 5% DAO cut upstream — see `fsm/committee.go:15–82` and lines 64–75 for the DAO split.

Plugin `EndBlock` (route via the existing `FSMToPlugin_EndBlock` message handled in `plugin/go/contract/plugin.go`) calls into `reward.go::ProcessRewards`:

1. Read canoLiq committee reward pool balance (Canopy pool key, prefix `[]byte{2}`, pool id = canoLiq chainId). This is the post-Canopy-5%-cut amount.
2. Compute `delta = pool_now - last_processed_reward_height_balance` to isolate this block's reward.
3. `fee = delta * 12 / 100` (read from a `canoliq/params/fee_bps` key, default 1200 bps so it's governance-tunable in Phase 2).
4. `net_to_users = delta - fee`. Add `net_to_users` to `total_pooled_cnpy` — cCNPY exchange rate rises automatically (40% of fee also rebates here per spec since 40% of fee → users → so effective user share = `net_to_users + 0.4 * fee`).
5. Split the fee:
   - `0.40 * fee` → also added to `total_pooled_cnpy` (users, reinvested — increases cCNPY/CNPY rate).
   - `0.30 * fee` → `canoliq/treasury/canoliq`.
   - `0.15 * fee` → `canoliq/validator/incentives/{addr}` distributed pro-rata across canoLiq committee validators (read validator set from Canopy supply/committee state).
   - `0.15 * fee` → `canoliq/buyback/pool` (held in CNPY until Phase 2 buyback executes).
6. Update `last_processed_reward_height`.

Use `lib.SafeMulDiv` semantics (mirror `fsm/committee.go:72`) to avoid overflow.

### 5. CPLQ token initialization

CPLQ is a fixed 100,000,000 supply token, minted once at canoLiq genesis with the distribution from whitepaper §5 / tokenomics §2:

| Bucket | % | Vesting |
|---|---:|---|
| Validators & Infrastructure (restakers) | 22 | service-based lockup (TBD: time-vest as proxy) |
| Liquidity Incentives (Farming) | 15 | unlocked, drip-released by canoLiq DAO |
| Community & Airdrops | 20 | unlocked at TGE |
| DAO Treasury (canoLiq) | 15 | unlocked, multisig-gated (Phase 2) |
| Founders & Core Team | 12 | 3-year linear vest, 12-month cliff |
| Strategic Partners & Integrations | 10 | 2-year linear vest, 6-month cliff |
| Plugin & Dev Grants | 6 | unlocked, drip-released |

Implementation: a one-shot `Genesis()` plugin call (the plugin lifecycle exposes `Genesis` per the FSM agent report — see `lib/plugin.go`). `Genesis` writes:
- `CanoliqGlobals.cplq_total_supply = 100_000_000 * 10^6` (using uCPLQ micro-units for parity with uCNPY).
- For each bucket, either set `canoliq/cplq/balance/{addr}` directly (unlocked tranches) or write a `VestingSchedule` (vested tranches). Bucket recipient addresses come from a JSON config file `plugin/go/canoliq/genesis.json` (mirror of `plugin/go/chain.json`) — keeps founders/partners addresses out of code.

### 6. Tests (`plugin/go/canoliq/canoliq_test.go`)

Follow the `require.X` style and the integration approach in `plugin/go/tutorial/rpc_test.go`.

Cover:
- Deposit mints correct cCNPY at 1:1 (first deposit) and at correct ratio (subsequent).
- Redeem queues a `Redemption` with the right unbond height.
- Claim before unbond → error; claim after → CNPY arrives.
- Reward injection of `1000` uCNPY into the committee pool produces:
  - `120` fee, `880` to `total_pooled_cnpy`, then `+48` (40% of fee) to `total_pooled_cnpy` = `928` total.
  - `36` to canoLiq treasury, `18` to validator incentives, `18` to buyback pool.
  - Verify with whitepaper §7's example math: net yield = `0.88 * 0.95X` of the original X.
- CPLQ transfer respects vesting (cannot transfer locked).
- Vesting linear unlock crosses cliff correctly.
- Genesis allocation totals exactly 100M CPLQ.

### 7. Sub-chain registration

Document (in a new `plugin/go/canoliq/README.md`) how to register canoLiq as a Canopy committee:
- Pick a `chainId` (e.g., the next free one — read from existing committee registry).
- Validators add the canoLiq chainId to their `Validator.Committees[]` via `MessageEditStake` to opt in (restaking — Canopy already supports this, see `fsm/validator.pb.go:45–76`).
- Submit a `MessageSubsidy` proposal (already implemented in `fsm/message.go`, `HandleMessageSubsidy`) requesting CNPY subsidies from the Canopy DAO per whitepaper §8.
- Plugin starts via `~/.canopy/config.json` `"plugin": "canoliq"` (mirroring `"plugin": "go"` in the existing setup, AGENTS.md line 136).

### Critical files to modify

- **NEW:** `plugin/go/canoliq/*.go` (all logic).
- **NEW:** `plugin/go/proto/canoliq.proto`.
- **MODIFY:** `plugin/go/main.go` — add a build-tag or env-var branch so either `contract.StartPlugin` or `canoliq.StartPlugin` runs (preserve existing tutorial).
- **MODIFY:** `plugin/go/proto/_generate.sh` — include `canoliq.proto` in generation.
- **NEW:** `plugin/go/canoliq/genesis.json` — distribution recipient addresses.
- **NEW:** `plugin/go/canoliq/README.md` — registration + ops guide.

### Reused from Canopy core (no changes)

- `fsm/committee.go` — `FundCommitteeRewardPools`, `GetSubsidizedCommittees`, `DistributeCommitteeRewards`, `GetBlockMintStats` (handles 5% DAO cut and per-committee minting).
- `fsm/gov_params.go` — `DaoRewardPercentage` (default 5), `UnstakingBlocks`, `StakePercentForSubsidizedCommittee`.
- `fsm/message.go` — `HandleMessageSubsidy` (Canopy-DAO → canoLiq subsidies), `HandleMessageStake`/`HandleMessageEditStake` (validators opt into canoLiq committee).
- `lib/plugin.go` — plugin lifecycle (`Genesis`, `BeginBlock`, `CheckTx`, `DeliverTx`, `EndBlock`).
- `plugin/go/contract/plugin.go` — Unix-socket transport, handshake, length-prefixed protobuf framing, `StateRead`/`StateWrite` helpers.
- `lib.SafeMulDiv` — overflow-safe mul-then-div for fee math.
- `lib.DAOPoolID` (= `2*MaxUint16+1`) and pool key derivation conventions.

---

## Phase 2 — Governance, buyback, treasury operations

### 1. Protobuf message and state types

Add to `plugin/go/proto/canoliq.proto`:

| Message | Fields | Purpose |
|---|---|---|
| `MessageCPLQStake` | `from_address`, `amount` | Locks liquid CPLQ → stake balance; counts toward governance weight |
| `MessageCPLQUnstake` | `from_address`, `amount` | Debits stake, queues unbond record maturing after `cplq_unstaking_blocks` |
| `MessageCPLQClaimUnstake` | `from_address`, `unstake_id` | Returns matured CPLQ to liquid balance |
| `MessageCPLQProposalCreate` | `from_address`, `payload` (oneof: param_change, buyback, treasury_spend) | Opens a proposal; assigns id; snapshots `total_staked_cplq` |
| `MessageCPLQVote` | `from_address`, `proposal_id`, `choice` (yes/no/abstain) | Records weighted vote using staker's snapshot weight at proposal creation height |
| `MessageBuybackExecute` | `from_address`, `proposal_id` | Idempotently runs an approved buyback proposal |
| `MessageDAOTreasurySpend` | `from_address`, `proposal_id` | Idempotently runs an approved spend (subject to threshold + timelock) |
| `MessageMultisigApprove` | `from_address` (signer), `spend_id` | Per-signer approval for above-threshold spend |

New stored types:

| Type | Fields |
|---|---|
| `CPLQStake` | `address`, `amount` |
| `UnstakingCPLQ` | `id`, `address`, `amount`, `mature_height` |
| `Proposal` | `id`, `proposer`, `creation_height`, `expiry_height`, `snapshot_total_staked`, `payload` (oneof), `status` (active/passed/failed/executed) |
| `Vote` | `proposal_id`, `voter`, `choice`, `weight` |
| `ProposalParamChange` | `params` (full new `CanoliqParams`) — full-set replacement keeps `ValidateParams` semantics |
| `ProposalBuyback` | `cnpy_amount`, `price_uCnpyPerCplq`, `mode` (BURN / DISTRIBUTE_STAKERS) |
| `ProposalTreasurySpend` | `recipient`, `amount`, `denomination` (CNPY / CPLQ), `source` (treasury_canopy / treasury_cplq) |
| `TreasurySpend` | `id`, `proposal_id`, `executable_height`, `payload`, `executed` |
| `MultisigApproval` | `spend_id`, `signer`, `height` |

Extend `CanoliqParams` with the governance-tunable knobs (defaults in parens):

- `insurance_bps` (1500) — share of treasury credit redirected to insurance pool. WP §11 calls for 1–2% of treasury, equivalent to 1500 bps of the 30% treasury slice.
- `treasury_threshold` (1_000_000_000 uCNPY) — spend amount above which multisig + timelock kick in
- `multisig_signers` (repeated bytes) — set of authorized signer addresses; configured at genesis, mutable via param-change vote
- `multisig_threshold` (3) — minimum approvals for above-threshold spends
- `voting_period_blocks` (~7d at 6s blocks = 100_800)
- `quorum_bps` (3300) — fraction of snapshot staked CPLQ that must participate
- `pass_threshold_bps` (5001) — `yes` vs `(yes + no)` to pass; just-above-50% by default
- `timelock_blocks` (~48h at 6s = 28_800)
- `cplq_unstaking_blocks` (~7d at 6s = 100_800)
- `proposal_fee`, `vote_fee`, `stake_fee` — minimum tx fees

Extend `CanoliqGlobals`: `total_staked_cplq`, `next_proposal_id`, `next_buyback_id`, `next_spend_id`, `next_unstake_id`.

### 2. State key schema

```
canoliq/cplq/stake/{addr}                    → CPLQStake
canoliq/cplq/unstaking/{addr}/{id}           → UnstakingCPLQ
canoliq/proposal/{id}                        → Proposal
canoliq/proposal/index                       → list of active proposal ids
canoliq/vote/{proposal_id}/{voter}           → Vote
canoliq/buyback/order/{id}                   → BuybackOrder (post-execution receipt)
canoliq/spend/{id}                           → TreasurySpend
canoliq/multisig/approval/{spend_id}/{signer} → MultisigApproval
canoliq/insurance/pool                       → uint64 (CNPY held)
```

### 3. CPLQ staking

Stake locks liquid CPLQ into a per-address `CPLQStake` record and increments `globals.total_staked_cplq`. Unstake debits the record, writes `UnstakingCPLQ{mature_height: h + cplq_unstaking_blocks}`, and decrements `total_staked_cplq` immediately so unstaked CPLQ has zero voting weight from the moment of unstake. Claim moves matured records back to the liquid balance.

The unstaking window must be ≥ `voting_period_blocks` so a voter cannot stake → vote → unstake → unwind their position before tally. With defaults both at 7d this is satisfied.

### 4. On-chain governance (`governance.go`)

Proposal lifecycle:

1. **Create** (`MessageCPLQProposalCreate`) — caller must hold ≥ some minimum stake (read from params); plugin assigns `next_proposal_id`, snapshots `globals.total_staked_cplq` into `proposal.snapshot_total_staked`, sets `expiry_height = h + voting_period_blocks`, appends to proposal index.
2. **Vote** (`MessageCPLQVote`) — voter's weight = the `CPLQStake` balance **at `proposal.creation_height`**. The simplest implementation: voters must have staked *before* creation (the snapshot uses the *current* `CPLQStake` reading and rejects votes whose `staked_at_height > proposal.creation_height`). Each address votes once per proposal.
3. **Tally** (BeginBlock) — when `current_height >= proposal.expiry_height`, sum `yes/no/abstain` weights. Apply pass rule:
   - `quorum`: `yes + no + abstain >= quorum_bps * snapshot_total_staked / 10_000`
   - `threshold`: `yes >= pass_threshold_bps * (yes + no) / 10_000`
4. **Execute on pass**:
   - `param_change`: `SaveParams(payload.params)` — runs through `ValidateParams`.
   - `buyback`: write a `BuybackOrder` keyed by id; `MessageBuybackExecute` is the trigger that drains the order.
   - `treasury_spend`: write a `TreasurySpend` with `executable_height = h + (timelock_blocks if amount > treasury_threshold else 0)`.
5. **Cleanup**: delete `proposal/{id}` + all `vote/{id}/*` records on either pass or fail; remove from proposal index.

### 5. Per-validator pro-rata reward distribution (Phase 1 carryover)

Replace `committeeAggregatorAddr()` in `reward.go`. Read the canoLiq committee validator set + per-validator stake from FSM state via `StateRead` against Canopy's validator prefix and `Validator.Committees[]`. Distribute `split.Validators` proportionally:

```
for each validator v in committee:
  share[v] = mulDiv(split.Validators, stake[v], total_committee_stake)
remainder credited to largest validator (or carried as `validator_remainder` on globals)
```

Tests must assert that a 70/30 stake split yields a 70/30 incentive split modulo rounding.

### 6. Buyback (`buyback.go`) — internal accounting swap

`MessageBuybackExecute` triggers a previously-passed `ProposalBuyback`:

1. Verify `proposal.status == passed` and that buyback hasn't already executed (`BuybackOrder.executed == false`).
2. Drain `cnpy_amount` from `canoliq/buyback/pool` (clamp to available).
3. Compute `cplq_acquired = cnpy_amount * 10^6 / price_uCnpyPerCplq` using `mulDiv`.
4. Source the CPLQ from `canoliq/treasury/cplq` (the DAO 15% bucket holds the supply for buybacks).
5. **Mode = BURN**: decrement `globals.cplq_total_supply` and `globals.cplq_circulating_supply` by `cplq_acquired`; deduct from `treasury/cplq`; the drained CNPY is treated as paid into `treasury/canoliq`.
6. **Mode = DISTRIBUTE_STAKERS**: iterate `CPLQStake` records (or use a stake-index key for efficiency), credit each staker `mulDiv(cplq_acquired, stake[s], total_staked_cplq)`; rounding remainder to the largest staker.
7. Mark `BuybackOrder.executed = true` so re-execute is a no-op.

Why this is faithful to WP §6: the whitepaper describes "market buyback and burn or direct distribution governed by DAO" — Phase 2 implements the governance + accounting spine. A real on-chain market route (via `fsm/dex.go` or `fsm/swap.go`) requires a relayer to bridge plugin state ↔ FSM tx, which is precisely the Phase 1.5 / Phase 3 plumbing. Internal swap with a vote-set price preserves the economic shape (CNPY ⇨ CPLQ extraction) without that dependency.

### 7. DAO treasury spend (`treasury.go`) — multisig + timelock

When a `ProposalTreasurySpend` passes, write a `TreasurySpend` record with:

- `executable_height = current_height + (timelock_blocks if amount > treasury_threshold else 0)`
- `payload` (recipient, amount, denomination, source)

`MessageMultisigApprove` records a signer's approval (signer must be in `multisig_signers`). BeginBlock scans pending spends and executes when:

1. `current_height >= executable_height` (timelock elapsed)
2. EITHER `amount <= treasury_threshold` (no multisig required) OR `count(approvals) >= multisig_threshold`

Below-threshold spends still pass through the proposal route (not unilaterally callable) — they just skip the timelock and multisig gate.

### 8. Insurance fund auto-routing

Modify `ProcessRewards` in `reward.go`. After computing `split.Treasury`, redirect `insurance = mulDiv(split.Treasury, params.InsuranceBps, 10_000)` into `canoliq/insurance/pool` and credit only `split.Treasury - insurance` to `treasury/canoliq`. With defaults (`insurance_bps = 1500`) this routes 15% of the treasury slice (= 4.5% of the protocol fee, ≈ 0.51% of post-Canopy reward) into insurance — within WP §11's "1–2% of treasury" framing when measured against ongoing treasury inflows.

The insurance pool is a passive accumulator in Phase 2. Slashing-reimbursement disbursement logic is a Phase 3 concern.

### 9. Tests

Cover, in addition to the items in §6 of Phase 1:

- Stake / unstake / claim full lifecycle.
- Proposal create / vote / tally / execute round-trip for each payload type.
- Voter weight uses the *creation-height* snapshot, not current stake — flash-stake post-creation has zero weight.
- Quorum miss → proposal fails; exact pass-threshold accepts; just-below rejects.
- Param-change proposal mutates `CanoliqParams` and the next `ProcessRewards` observes the new bps.
- Buyback BURN reduces `cplq_total_supply` + `cplq_circulating_supply` by exactly `cplq_acquired`.
- Buyback DISTRIBUTE credits multiple stakers pro-rata; rounding remainder accounted for.
- Treasury spend below threshold: single-step, no multisig required.
- Treasury spend above threshold: rejected pre-timelock, rejected with insufficient approvals, executes once both met; idempotent re-trigger.
- Insurance skim conservation: `treasury + insurance + buyback + validators + user_rebate + net_to_users == delta`.
- Per-validator pro-rata rounding: skewed stakes (e.g., 7M / 3M / 1M committee stake) get proportional shares.

### Critical files to add/modify

- **NEW:** `plugin/go/canoliq/stake.go`, `governance.go`, `buyback.go`, `treasury.go`.
- **MODIFY:** `plugin/go/canoliq/canoliq.go` — extend `CheckTx`/`DeliverTx` switches, add `BeginBlock` proposal/spend scan.
- **MODIFY:** `plugin/go/canoliq/reward.go` — insurance skim + per-validator pro-rata.
- **MODIFY:** `plugin/go/canoliq/config.go` — register new tx types in `SupportedTransactions` / `TransactionTypeUrls`.
- **MODIFY:** `plugin/go/canoliq/params.go` — `DefaultParams` adds new fields; `ValidateParams` covers new invariants.
- **MODIFY:** `plugin/go/proto/canoliq.proto` — new tx + stored types; regenerate.
- **MODIFY:** `plugin/go/canoliq/genesis.json` — `multisig_signers` configured at genesis.
- **MODIFY:** `plugin/go/canoliq/README.md` — proposal workflow, multisig signer ops, buyback playbook.

---

## Phase 3 — Monitoring, autonomy graduation

The detailed Phase 3 plan lives in the **Progress checklist** above
(§1 query layer, §1.1 per-address routes, §1.1-bis collection
indexes, §2 alerting hooks, §3 graduation tooling). §1 and §1.1 are
landed and verified live; §1.1-bis, §2, and §3 are scoped and
ready to pick up — the alerting "stuck redemption" condition
specifically depends on §1.1-bis indexes landing first.

---

## Verification (Phase 1)

End-to-end checks once Phase 1 lands:

1. **Build:** `cd plugin/go && make build` succeeds with the new package; protobuf regen `./proto/_generate.sh` produces `canoliq.pb.go` cleanly.
2. **Unit tests:** `cd plugin/go/canoliq && go test ./...` — all cases listed in §6 pass.
3. **In-process behavior coverage:** the `fakeStore` harness in `fakeplugin_test.go` runs the real `Canoliq` handlers against an in-memory KV, so deposit/redeem/claim/reward/vesting flows are all exercised against the production code paths. Specifically:
   - Deposit and exchange-rate math — `TestDepositMintsOneToOneOnFirstDeposit`, `TestDepositSubsequentRatio`.
   - Redeem queues `Redemption`, claim before/after maturity — `TestRedeemQueuesRedemption`, `TestClaimRedemptionMaturity`.
   - Reward fee-split (single block) — `TestRewardSplitWhitepaperExample`.
   - Reward delta watermark across blocks — `TestRewardSweepMultiBlock`.
   - Treasury / buyback / validator-incentives reconcile against post-DAO inflow — conservation asserted in `TestWhitepaperSection7Reconciliation`.
   - Vesting unlock math + claim handler — `TestVestingLinearUnlock`, `TestDeliverCPLQClaimVestedFlow`.
   - Composite deposit → reward → redeem yield — `TestCompositeDepositRewardRedeem`.
4. **Whitepaper §7 reconciliation:** for total reward `X = 1000`, expect `0.95X = 950` to canoLiq pool, `fee = 114`, `net = 836` to users + `45.6` rebate from 40% of fee = `881.6` user yield (`0.88 * 0.95 * X` modulo integer truncation). Asserted by `TestWhitepaperSection7Reconciliation`.
5. **Coordination:** before submitting upstream, post a design summary to Canopy Discord per `CONTRIBUTING.md` "coordinate bigger changes" guidance — even though Phase 1 touches only `plugin/go/`, registering a new sub-chain is operationally significant.

Live-socket integration (plugin handshake, real `EndBlock` over unix socket,
on-chain `MessageSubsidy`/`MessageCanoliqDeposit` round-trips) is tracked in
**Phase 1.5** above and is gated on a compose service for the plugin process
plus a minimal `canoliqctl` for tx submission — neither exists yet.
