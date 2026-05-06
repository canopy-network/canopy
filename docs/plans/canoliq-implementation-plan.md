# canoLiq Tokenomics Implementation Plan

## Context

canoLiq is a liquid staking protocol described in two whitepapers (`Cano Liq Whitepaper En v2` and `canoLiq_Tokenomics_v2`). It lets users deposit CNPY, receive a yield-bearing receipt token (cCNPY), and earn staking rewards minus a 12% protocol fee. It also issues CLIQ — a fixed-supply (100M) governance/value-capture token with vesting, treasury, and buyback mechanics.

**Why the work is needed.** The whitepapers describe canoLiq as a sub-chain on the Canopy Network that aligns with Canopy's economic primitives (5% DAO cut, restaking, committee subsidies, halving). None of these primitives need re-implementing — they already exist in `/fsm/committee.go`, `/fsm/gov_params.go`, and the validator/restaking module. What does not exist is the **canoLiq-specific layer**: cCNPY deposit/redeem, the 12% protocol fee with 40/30/15/15 split, CLIQ token + vesting, canoLiq DAO treasury, CLIQ-holder governance, and buyback. This plan adds that layer.

**Intended outcome.** A canoLiq sub-chain that can be subsidized by Canopy's existing committee mechanism, mints cCNPY 1:1 against deposited CNPY, accrues staking rewards via Canopy's existing distribution path, applies the 12% fee with the canonical 40/30/15/15 split inside its plugin, and tracks CLIQ allocations with vesting — all without modifying Canopy core consensus code.

---

## Architecture decision (confirmed with user)

- **Integration model:** sub-chain via Go plugin. canoLiq runs as its own Canopy committee (own `chainId`) with logic implemented as a Go plugin. It reuses Canopy's existing 5% DAO cut, restaking (`Validator.Committees[]`), committee subsidies (`GetSubsidizedCommittees`), and halving emission. No changes to `/fsm/`, `/lib/`, or any Canopy core consensus code.
- **Token model:** dedicated canoLiq state buckets via plugin `StateRead`/`StateWrite`. cCNPY and CLIQ balances live in plugin-owned keys; `Account` proto is untouched.
- **Scope:** phased. Phase 1 = MVP (deposit/redeem, fee split, CLIQ + vesting). Phase 2 = governance + buyback. Phase 3 = monitoring + autonomy graduation.

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
- [x] Define `MessageCLIQTransfer`
- [x] Define `MessageCLIQClaimVested`
- [x] Define `CanoliqGlobals` stored type
- [x] Define `Redemption` stored type
- [x] Define `VestingSchedule` stored type
- [x] Update `plugin/go/proto/_generate.sh` to include `canoliq.proto` (existing glob `./*.proto` already covers it)
- [x] Regenerate Go code (`canoliq.pb.go`)

#### 2. State key schema
- [x] Create `plugin/go/canoliq/state.go`
- [x] Implement `KeyForGlobals()`
- [x] Implement `KeyForCCNPYBalance(addr)`
- [x] Implement `KeyForCLIQBalance(addr)`
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
- [x] `CheckMessageCLIQTransfer` + `DeliverMessageCLIQTransfer`
- [x] `CheckMessageCLIQClaimVested` + `DeliverMessageCLIQClaimVested`

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

#### 5. CLIQ token initialization
- [x] Create `plugin/go/canoliq/vesting.go` (linear vest + cliff)
- [x] Create `plugin/go/canoliq/genesis.json` with bucket recipient addresses
- [x] Implement plugin `Genesis()` to mint 100M CLIQ
- [x] Allocate Validators & Infrastructure bucket (22%)
- [x] Allocate Liquidity Incentives bucket (15%)
- [x] Allocate Community & Airdrops bucket (20%)
- [x] Allocate DAO Treasury bucket (15%)
- [x] Allocate Founders & Core Team bucket (12%, 3yr / 12mo cliff)
- [x] Allocate Strategic Partners bucket (10%, 2yr / 6mo cliff)
- [x] Allocate Plugin & Dev Grants bucket (6%)
- [x] Verify total equals exactly 100,000,000 CLIQ

#### 6. Tests
- [x] Create `plugin/go/canoliq/canoliq_test.go`
- [x] Deposit at 1:1 (first deposit)
- [x] Deposit at correct ratio (subsequent)
- [x] Redeem queues `Redemption` with correct unbond height
- [x] Claim before unbond → error
- [x] Claim after unbond → CNPY arrives in user account
- [x] Reward injection (X=1000) → 928 to pool, 36/18/18 splits
- [x] CLIQ transfer respects vesting (locked → error)
- [x] Vesting linear unlock crosses cliff correctly
- [x] Genesis allocation totals exactly 100M CLIQ
- [x] `DeliverMessageCLIQClaimVested` end-to-end (before cliff / halfway / idempotent / past end)
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
- [x] In-process: vesting tranche claims 0 before cliff, linear after (`::TestVestingLinearUnlock`, `::TestDeliverCLIQClaimVestedFlow`)
- [x] In-process: deposit → reward accrues → redeem returns yield (`::TestCompositeDepositRewardRedeem`)
- [x] Whitepaper §7 reconciliation test passes (X=1000 → 881.6 user yield ± truncation)
- [ ] Coordination: design summary posted to Canopy Discord

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
   surface (deposit/redeem/claim/cliq-transfer/cliq-claim-vested) and the
   Phase 2 surface minus `proposal-create` (vote, stake/unstake/claim,
   buyback-execute, spend-execute, multisig-approve). `proposal-create` is
   deferred because its `Any` payload needs three sub-command variants.

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
- [ ] `MessageSubsidy` proposal funds committee 2's reward pool; canoLiq's
      `EndBlock` observes the inflow and applies the 12% fee on a real chain
      — *not yet directly verified* on the live chain. Committee 2 is
      auto-subsidized via the 33% threshold (both validators opted in), so
      `EndBlock` is observing inflow each block; explicit fee-split
      reconciliation against on-chain state is blocked on a plugin-state
      RPC, which is Phase 3 work.
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
- **Governance weight** = staked CLIQ only, snapshot at proposal-creation height. WP §4: "stake/lock CLIQ for governance power and boosts." Yield boosts are mentioned but undefined — deferred.
- **Treasury spend** = hybrid. Below `treasury_threshold`, a passing CLIQ vote alone executes. Above threshold, vote + N-of-M multisig approvals + `timelock_blocks` delay. WP §9 + Tokenomics §7: "Multisig and timelocks required for treasury spending above thresholds."
- **Per-validator pro-rata** lands here (Phase 1 carryover). Required for the buyback "distribute to stakers" mode and for honest per-validator infra credits.

#### 1. Protobuf and state types
- [x] Add to `plugin/go/proto/canoliq.proto`:
  - [x] `MessageCLIQStake` / `MessageCLIQUnstake` / `MessageCLIQClaimUnstake`
  - [x] `MessageCLIQProposalCreate` / `MessageCLIQVote`
  - [x] `MessageBuybackExecute`
  - [x] `MessageDAOTreasurySpend` / `MessageMultisigApprove`
- [x] New stored types: `CLIQStake`, `UnstakingCLIQ`, `Proposal`, `Vote`, `BuybackOrder`, `MultisigApproval` (plus `CLIQStakeIndex`, `ValidatorRegistry`, `ValidatorRegistryEntry`, `ProposalIndex`)
- [x] `Proposal.payload` typed via `google.protobuf.Any` (resolved with `contract.FromAny`) — equivalent to a oneof for dispatch purposes
- [x] Extend `CanoliqParams`: `insurance_bps`, `treasury_threshold`, `multisig_signers`, `multisig_threshold`, `voting_period_blocks`, `quorum_bps`, `pass_threshold_bps`, `timelock_blocks`, `cliq_unstaking_blocks`, `proposal_fee`, `vote_fee`, `stake_fee`, `multisig_approve_fee`, `min_stake_to_propose`
- [x] Extend `CanoliqGlobals`: `total_staked_cliq`, `next_proposal_id`, `next_buyback_id`, `next_spend_id`, `next_unstake_id`
- [x] Update `Config.SupportedTransactions` / `TransactionTypeUrls`
- [x] Regenerate `canoliq.pb.go`

#### 2. State key schema
- [x] `KeyForCLIQStake(addr)` (active stake balance)
- [x] `KeyForCLIQUnstaking(addr, id)` (in-flight unbond record)
- [x] `KeyForCLIQStakeIndex()` (active staker enumeration for buyback DISTRIBUTE)
- [x] `KeyForProposal(id)` + `KeyForProposalIndex()` (active proposal id list)
- [x] `KeyForVote(proposalId, voter)`
- [x] `KeyForBuybackOrder(id)`
- [x] `KeyForTreasurySpend(spendId)` + `KeyForSpendIndex()` + `KeyForMultisigApproval(spendId, signer)`
- [x] `KeyForInsurancePool()`
- [x] `KeyForValidatorRegistry()` (singleton registry for per-validator pro-rata)
- [x] Read/write wrappers for each new stored type

#### 3. CLIQ staking
- [x] `stake.go`: stake/unstake/claim handlers
- [x] `Check`/`Deliver MessageCLIQStake` — moves liquid CLIQ → `CLIQStake` record, increments `total_staked_cliq`
- [x] `Check`/`Deliver MessageCLIQUnstake` — debits stake, queues `UnstakingCLIQ` with `mature_height = h + cliq_unstaking_blocks`
- [x] `Check`/`Deliver MessageCLIQClaimUnstake` — matures record, returns CLIQ to liquid balance
- [x] Tests: stake/unstake/claim, unbond before maturity error, double-claim idempotency (`TestCLIQStakeUnstakeClaim`)

#### 4. On-chain governance
- [x] `governance.go`: proposal lifecycle (create / vote / tally / execute)
- [x] `Check`/`Deliver MessageCLIQProposalCreate` — assigns id, snapshots `total_staked_cliq` at creation height, records `expiry_height = h + voting_period_blocks`
- [x] `Check`/`Deliver MessageCLIQVote` — looks up the voter's `CLIQStake` *as of `proposal.creation_height`*, records weighted yes/no/abstain
- [x] BeginBlock hook: scan proposal index, on expiry tally yes/no/abstain weights
- [x] Pass rule: `(yes + no + abstain) >= quorum_bps * snapshot_total_staked / 10000` AND `yes >= pass_threshold_bps * (yes + no) / 10000`
- [x] On pass: execute payload (param change directly; buyback/spend mark as executable downstream)
- [x] On fail/expire: delete proposal + drop from index
- [x] Snapshot mechanism (`CLIQStake.staked_at_height` vs `proposal.creation_height`) defeats flash-stake attacks
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
- [x] Acquire CLIQ at `proposal.price_micro_cnpy_per_cliq`: `cliq_acquired = cnpy_amount * 10^6 / price_micro_cnpy_per_cliq`
- [x] Source CLIQ from `canoliq/treasury/cliq` (DAO 15% bucket)
- [x] Mode `BUYBACK_BURN`: decrement `cliq_total_supply` and `cliq_circulating_supply` by `cliq_acquired`
- [x] Mode `BUYBACK_DISTRIBUTE_STAKERS`: pro-rata credit `CLIQStake` records by stake weight; CNPY moves to `treasury/canoliq`
- [x] Tests: burn (`TestBuybackBurnReducesSupply`), distribute multi-staker (`TestBuybackDistributeStakers`); idempotent re-execute covered in BURN test

#### 7. DAO treasury spend
- [x] `treasury.go`: `MessageDAOTreasurySpend` triggers a queued spend by proposal id
- [x] On vote pass: `queueTreasurySpend` writes `TreasurySpend` with `executable_height = h + (timelock_blocks if amount > treasury_threshold else 0)`
- [x] `Check`/`Deliver MessageMultisigApprove` — records signer's approval per `spend_id`; signer must be in `multisig_signers`
- [x] Execution: `current_height >= executable_height` AND (`amount <= threshold` OR `approvals >= multisig_threshold`)
- [x] Source from `treasury/canoliq` (CNPY) or `treasury/cliq` (CLIQ); credit recipient liquid balance
- [x] Tests: below-threshold (`TestTreasurySpendBelowThreshold`), above-threshold full path including timelock + missing approvals + replay (`TestTreasurySpendAboveThresholdRequiresTimelockAndMultisig`), non-signer rejection (`TestMultisigApprovalRejectsNonSigner`)

#### 8. Insurance fund auto-routing
- [x] `DefaultParams()` adds `insurance_bps` (default 1500 = 15% of incoming treasury credit, ≈ 1.5% of fee — within WP §11)
- [x] `ProcessRewards`: skim `insurance_bps` off `split.Treasury` before crediting `treasury/canoliq`
- [x] New scalar key `canoliq/insurance/pool`
- [x] `ValidateParams` allows `insurance_bps <= 10000`
- [x] Tests: full conservation including insurance (`TestInsuranceConservationFullSplit`); pre-existing reward tests updated to include the insurance line

#### 9. Tests (`plugin/go/canoliq/*_test.go`)
- [x] CLIQ stake / unstake / claim lifecycle
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
- [x] In-process: buyback burn reduces `cliq_total_supply`; distribute credits stakers
- [x] In-process: above-threshold spend rejected pre-timelock and pre-multisig-quorum, accepted once both met
- [x] In-process: insurance pool grows by `insurance_bps * fee * treasury_bps / 10000^2` per block; full conservation holds
- [x] In-process: per-validator infra credits split proportionally to committee stake

### Phase 3 — Monitoring & autonomy graduation

#### 1. Read-only query layer (HTTP-in-plugin)

The plugin process owns all canoLiq-prefixed state. Phase 1.5 left
fee-split reconciliation against the live chain blocked on a way to
read that state from outside the plugin. Phase 3 §1 fills that gap by
running an HTTP server inside the plugin process, alongside the FSM
unix socket. All routes are read-only — they call `StateRead` only.

##### 1a. Query helpers (`plugin/go/canoliq/query.go`)
- [x] `QueryGlobals()` → `*contract.CanoliqGlobals`
- [x] `QueryParams()` → `*contract.CanoliqParams`
- [x] `QueryAccount(addr)` → composite view: CNPY, cCNPY, liquid CLIQ,
      staked CLIQ, validator-incentive accrual, vesting schedules
      (with cumulative unlocked-to-date). Pending unstakes and open
      redemptions are *not* in the composite view because there is no
      per-address index — callers fetch individual records by id via
      the dedicated routes.
- [x] `QueryPools()` → treasury (CNPY/CLIQ), buyback, insurance,
      validator-incentives ledger (iterate `ValidatorRegistry` →
      per-address `KeyForValidatorIncentives`; falls back to legacy
      aggregator address when registry is empty)
- [x] `QueryProposal(id)` → `*contract.Proposal` (or 404)
- [x] `QueryProposalIndex()` → list of active proposal ids
- [x] `QueryVote(proposalId, voter)` → `*contract.Vote` (or 404)
- [x] `QueryBuybackOrder(id)` → `*contract.BuybackOrder`
- [x] `QueryTreasurySpend(id)` → `*contract.TreasurySpend`
- [x] `QuerySpendIndex()` → list of pending spend ids
- [x] `QueryMultisigApprovals(spendId)` → list of approvals filtered
      against current `params.multisig_signers`
- [x] `QueryValidatorRegistry()` → `*contract.ValidatorRegistry`
- [x] `QueryRedemption(addr, id)` → `*contract.Redemption` (or 404)
- [x] `QueryVesting(addr)` → list of `VestingSchedule` + cumulative
      unlocked-at-current-height (uses `Plugin.CurrentHeight()`)

##### 1b. HTTP server (`plugin/go/canoliq/rpc.go`)
- [x] `net/http` mux gated by `Config.RpcAddress` (or env
      `CANOLIQ_RPC_ADDR`); empty disables the listener so the Phase 1
      binary surface is preserved.
- [x] Route table:
  - `GET /v1/globals`
  - `GET /v1/params`
  - `GET /v1/account/{addr}` (hex address, with or without `0x`)
  - `GET /v1/pools`
  - `GET /v1/proposals` (active id list)
  - `GET /v1/proposal/{id}`
  - `GET /v1/vote/{id}/{voter}`
  - `GET /v1/buyback/{id}`
  - `GET /v1/spends` (pending id list)
  - `GET /v1/spend/{id}`
  - `GET /v1/spend/{id}/approvals`
  - `GET /v1/validators`
  - `GET /v1/redemption/{addr}/{id}`
  - `GET /v1/vesting/{addr}`
  - `GET /v1/health` (liveness — `{height, genesisComplete, chainId}`)
- [x] Encode responses as JSON; proto `@gotags` keep field names
      consistent with `canoliqctl`.
- [x] 404 with `{"error":"..."}` on missing keys; 400 on malformed
      address; 405 on non-GET; 500 on plugin-internal error mapped
      from `*contract.PluginError` (address-shape errors map to 400).
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
      (`rpc_test.go::TestRPCHealthAndGlobals`,
      `TestRPCParamsRoundTrip`,
      `TestRPCAccountComposite`, `TestRPCProposalLifecycle`,
      `TestRPCSpendAndApprovals`, `TestRPCBuybackOrder`,
      `TestRPCRedemptionAndVesting`).
- [x] 400 on malformed address (`TestRPCAccountRejectsBadAddress`),
      405 on non-GET (`TestRPCMethodNotAllowed`), and an
      empty-addr-disables-server case
      (`TestRPCStartRPCServerEmptyAddrDisabled`).
- [x] Conservation cross-check: `TestRPCPoolsConservationAfterReward`
      injects a 950-uCNPY post-DAO reward, runs `ProcessRewards`, and
      asserts the `/v1/pools` view + `globals.TotalPooledCnpy` sum
      back to the seeded delta — the same conservation equation as
      `TestInsuranceConservationFullSplit` but observed through the
      HTTP layer.

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
- [ ] Live integration: hit `/v1/globals` and `/v1/pools` against
      the running canoliq plugin in `.docker/compose.yaml` and
      reconcile a fee split — closes the open Phase 1.5 item.

#### 2. Alerting hooks (deferred)
- [ ] Threshold-driven webhook for buyback pool drain rate
- [ ] Stuck-redemption alert (queue length above threshold)
- [ ] Validator-incentives starvation alert (registry stake skew)

#### 3. Graduation snapshot / migration tooling (deferred)
Snapshot tooling for whitepaper §10 thresholds (TVL, validator
count, DAO maturity).

---

## Repo layout

New plugin directory (parallel to `/plugin/go/contract/`):

```
plugin/go/canoliq/
  canoliq.go         # CheckTx/DeliverTx switch and message handlers
  state.go           # Key helpers + read/write wrappers for cCNPY, CLIQ, vesting, treasury
  fee.go             # 12% fee application and 40/30/15/15 split
  vesting.go         # Vesting schedule application (founders, validators, partners)
  reward.go          # EndBlock hook: observe canoLiq pool, apply fee, distribute net to cCNPY holders
  governance.go      # (Phase 2) CLIQ-holder voting state and tally
  buyback.go         # (Phase 2) CNPY-treasury → CLIQ buyback
  error.go           # PluginError constructors
  config.go          # CanoliqConfig + DefaultConfig (registers SupportedTransactions/TypeUrls)
  canoliq_test.go    # Unit tests
plugin/go/proto/
  canoliq.proto      # New tx types and stored types (added alongside tx.proto)
plugin/go/main.go    # MODIFY: branch to start either contract.StartPlugin or canoliq.StartPlugin
                     # via env var or build tag; existing send tutorial still works.
```

---

## Phase 1 — MVP: deposit, redeem, fee split, CLIQ, vesting

### 1. Protobuf message and state types

Add to `plugin/go/proto/canoliq.proto`. Regenerate via `plugin/go/proto/_generate.sh`.

**New transaction messages:**

| Message | Fields | Purpose |
|---|---|---|
| `MessageCanoliqDeposit` | `from_address`, `amount` (uCNPY) | User deposits CNPY → mints cCNPY at current exchange rate |
| `MessageCanoliqRedeem` | `from_address`, `ccnpy_amount` | Burns cCNPY, queues CNPY redemption respecting committee unstaking cooldown |
| `MessageCanoliqClaimRedemption` | `from_address`, `redemption_id` | Withdraws matured CNPY redemption to user |
| `MessageCLIQTransfer` | `from_address`, `to_address`, `amount` | Transfers liquid (vested) CLIQ between accounts |
| `MessageCLIQClaimVested` | `from_address` | Moves any newly-unlocked CLIQ from vesting bucket → liquid CLIQ balance |

**New stored types:**

| Type | Fields | Notes |
|---|---|---|
| `CanoliqGlobals` | `total_ccnpy_supply`, `total_pooled_cnpy`, `pending_redemption_cnpy`, `last_processed_reward_height`, `cliq_total_supply`, `cliq_circulating_supply` | Singleton, key `globals` |
| `Redemption` | `id`, `address`, `cnpy_amount`, `unbond_complete_height` | Per-id record |
| `VestingSchedule` | `address`, `total_amount`, `cliff_height`, `start_height`, `end_height`, `claimed_amount` | Linear vest after cliff; one per allocation tranche |

### 2. State key schema (plugin-owned)

All keys are namespaced under a canoLiq prefix to avoid collision with the existing `[]byte{1}` (account) / `[]byte{2}` (pool) / `[]byte{7}` (gov) prefixes called out in `plugin/go/AGENTS.md` lines 50–54.

```
canoliq/globals                              → CanoliqGlobals
canoliq/ccnpy/balance/{addr}                 → uint64
canoliq/cliq/balance/{addr}                  → uint64       (liquid)
canoliq/cliq/vesting/{addr}/{schedule_id}    → VestingSchedule
canoliq/redemption/{addr}/{redemption_id}    → Redemption
canoliq/treasury/canoliq                     → uint64       (CNPY held)
canoliq/treasury/cliq                        → uint64       (CLIQ held by DAO)
canoliq/buyback/pool                         → uint64       (CNPY earmarked for buyback)
canoliq/validator/incentives/{addr}          → uint64       (CNPY accrued for infra)
```

Key helper file: `plugin/go/canoliq/state.go` — exposes `KeyForCCNPYBalance(addr)`, `KeyForCLIQBalance(addr)`, `KeyForVesting(addr, id)`, `KeyForRedemption(addr, id)`, `KeyForGlobals()`, etc. Mirror the helper style of `contract/contract.go` (`KeyForAccount`, `KeyForFeePool`, `KeyForFeeParams`).

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

**CLIQ transfer / vested claim:** standard balance update + vesting linear-unlock formula (`unlocked = total * (height - start) / (end - start)` clamped after cliff).

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

### 5. CLIQ token initialization

CLIQ is a fixed 100,000,000 supply token, minted once at canoLiq genesis with the distribution from whitepaper §5 / tokenomics §2:

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
- `CanoliqGlobals.cliq_total_supply = 100_000_000 * 10^6` (using uCLIQ micro-units for parity with uCNPY).
- For each bucket, either set `canoliq/cliq/balance/{addr}` directly (unlocked tranches) or write a `VestingSchedule` (vested tranches). Bucket recipient addresses come from a JSON config file `plugin/go/canoliq/genesis.json` (mirror of `plugin/go/chain.json`) — keeps founders/partners addresses out of code.

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
- CLIQ transfer respects vesting (cannot transfer locked).
- Vesting linear unlock crosses cliff correctly.
- Genesis allocation totals exactly 100M CLIQ.

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
| `MessageCLIQStake` | `from_address`, `amount` | Locks liquid CLIQ → stake balance; counts toward governance weight |
| `MessageCLIQUnstake` | `from_address`, `amount` | Debits stake, queues unbond record maturing after `cliq_unstaking_blocks` |
| `MessageCLIQClaimUnstake` | `from_address`, `unstake_id` | Returns matured CLIQ to liquid balance |
| `MessageCLIQProposalCreate` | `from_address`, `payload` (oneof: param_change, buyback, treasury_spend) | Opens a proposal; assigns id; snapshots `total_staked_cliq` |
| `MessageCLIQVote` | `from_address`, `proposal_id`, `choice` (yes/no/abstain) | Records weighted vote using staker's snapshot weight at proposal creation height |
| `MessageBuybackExecute` | `from_address`, `proposal_id` | Idempotently runs an approved buyback proposal |
| `MessageDAOTreasurySpend` | `from_address`, `proposal_id` | Idempotently runs an approved spend (subject to threshold + timelock) |
| `MessageMultisigApprove` | `from_address` (signer), `spend_id` | Per-signer approval for above-threshold spend |

New stored types:

| Type | Fields |
|---|---|
| `CLIQStake` | `address`, `amount` |
| `UnstakingCLIQ` | `id`, `address`, `amount`, `mature_height` |
| `Proposal` | `id`, `proposer`, `creation_height`, `expiry_height`, `snapshot_total_staked`, `payload` (oneof), `status` (active/passed/failed/executed) |
| `Vote` | `proposal_id`, `voter`, `choice`, `weight` |
| `ProposalParamChange` | `params` (full new `CanoliqParams`) — full-set replacement keeps `ValidateParams` semantics |
| `ProposalBuyback` | `cnpy_amount`, `price_uCnpyPerCliq`, `mode` (BURN / DISTRIBUTE_STAKERS) |
| `ProposalTreasurySpend` | `recipient`, `amount`, `denomination` (CNPY / CLIQ), `source` (treasury_canopy / treasury_cliq) |
| `TreasurySpend` | `id`, `proposal_id`, `executable_height`, `payload`, `executed` |
| `MultisigApproval` | `spend_id`, `signer`, `height` |

Extend `CanoliqParams` with the governance-tunable knobs (defaults in parens):

- `insurance_bps` (1500) — share of treasury credit redirected to insurance pool. WP §11 calls for 1–2% of treasury, equivalent to 1500 bps of the 30% treasury slice.
- `treasury_threshold` (1_000_000_000 uCNPY) — spend amount above which multisig + timelock kick in
- `multisig_signers` (repeated bytes) — set of authorized signer addresses; configured at genesis, mutable via param-change vote
- `multisig_threshold` (3) — minimum approvals for above-threshold spends
- `voting_period_blocks` (~7d at 6s blocks = 100_800)
- `quorum_bps` (3300) — fraction of snapshot staked CLIQ that must participate
- `pass_threshold_bps` (5001) — `yes` vs `(yes + no)` to pass; just-above-50% by default
- `timelock_blocks` (~48h at 6s = 28_800)
- `cliq_unstaking_blocks` (~7d at 6s = 100_800)
- `proposal_fee`, `vote_fee`, `stake_fee` — minimum tx fees

Extend `CanoliqGlobals`: `total_staked_cliq`, `next_proposal_id`, `next_buyback_id`, `next_spend_id`, `next_unstake_id`.

### 2. State key schema

```
canoliq/cliq/stake/{addr}                    → CLIQStake
canoliq/cliq/unstaking/{addr}/{id}           → UnstakingCLIQ
canoliq/proposal/{id}                        → Proposal
canoliq/proposal/index                       → list of active proposal ids
canoliq/vote/{proposal_id}/{voter}           → Vote
canoliq/buyback/order/{id}                   → BuybackOrder (post-execution receipt)
canoliq/spend/{id}                           → TreasurySpend
canoliq/multisig/approval/{spend_id}/{signer} → MultisigApproval
canoliq/insurance/pool                       → uint64 (CNPY held)
```

### 3. CLIQ staking

Stake locks liquid CLIQ into a per-address `CLIQStake` record and increments `globals.total_staked_cliq`. Unstake debits the record, writes `UnstakingCLIQ{mature_height: h + cliq_unstaking_blocks}`, and decrements `total_staked_cliq` immediately so unstaked CLIQ has zero voting weight from the moment of unstake. Claim moves matured records back to the liquid balance.

The unstaking window must be ≥ `voting_period_blocks` so a voter cannot stake → vote → unstake → unwind their position before tally. With defaults both at 7d this is satisfied.

### 4. On-chain governance (`governance.go`)

Proposal lifecycle:

1. **Create** (`MessageCLIQProposalCreate`) — caller must hold ≥ some minimum stake (read from params); plugin assigns `next_proposal_id`, snapshots `globals.total_staked_cliq` into `proposal.snapshot_total_staked`, sets `expiry_height = h + voting_period_blocks`, appends to proposal index.
2. **Vote** (`MessageCLIQVote`) — voter's weight = the `CLIQStake` balance **at `proposal.creation_height`**. The simplest implementation: voters must have staked *before* creation (the snapshot uses the *current* `CLIQStake` reading and rejects votes whose `staked_at_height > proposal.creation_height`). Each address votes once per proposal.
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
3. Compute `cliq_acquired = cnpy_amount * 10^6 / price_uCnpyPerCliq` using `mulDiv`.
4. Source the CLIQ from `canoliq/treasury/cliq` (the DAO 15% bucket holds the supply for buybacks).
5. **Mode = BURN**: decrement `globals.cliq_total_supply` and `globals.cliq_circulating_supply` by `cliq_acquired`; deduct from `treasury/cliq`; the drained CNPY is treated as paid into `treasury/canoliq`.
6. **Mode = DISTRIBUTE_STAKERS**: iterate `CLIQStake` records (or use a stake-index key for efficiency), credit each staker `mulDiv(cliq_acquired, stake[s], total_staked_cliq)`; rounding remainder to the largest staker.
7. Mark `BuybackOrder.executed = true` so re-execute is a no-op.

Why this is faithful to WP §6: the whitepaper describes "market buyback and burn or direct distribution governed by DAO" — Phase 2 implements the governance + accounting spine. A real on-chain market route (via `fsm/dex.go` or `fsm/swap.go`) requires a relayer to bridge plugin state ↔ FSM tx, which is precisely the Phase 1.5 / Phase 3 plumbing. Internal swap with a vote-set price preserves the economic shape (CNPY ⇨ CLIQ extraction) without that dependency.

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
- Buyback BURN reduces `cliq_total_supply` + `cliq_circulating_supply` by exactly `cliq_acquired`.
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

## Phase 3 — Monitoring, autonomy graduation (outline)

Deferred. Includes: RPC endpoints for TVL / pool health / validator uptime / vesting status; alerting hooks; graduation tooling that snapshots state for migration to a standalone L1 once whitepaper §10 thresholds are met (TVL, validator count, DAO maturity).

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
   - Vesting unlock math + claim handler — `TestVestingLinearUnlock`, `TestDeliverCLIQClaimVestedFlow`.
   - Composite deposit → reward → redeem yield — `TestCompositeDepositRewardRedeem`.
4. **Whitepaper §7 reconciliation:** for total reward `X = 1000`, expect `0.95X = 950` to canoLiq pool, `fee = 114`, `net = 836` to users + `45.6` rebate from 40% of fee = `881.6` user yield (`0.88 * 0.95 * X` modulo integer truncation). Asserted by `TestWhitepaperSection7Reconciliation`.
5. **Coordination:** before submitting upstream, post a design summary to Canopy Discord per `CONTRIBUTING.md` "coordinate bigger changes" guidance — even though Phase 1 touches only `plugin/go/`, registering a new sub-chain is operationally significant.

Live-socket integration (plugin handshake, real `EndBlock` over unix socket,
on-chain `MessageSubsidy`/`MessageCanoliqDeposit` round-trips) is tracked in
**Phase 1.5** above and is gated on a compose service for the plugin process
plus a minimal `canoliqctl` for tx submission — neither exists yet.
