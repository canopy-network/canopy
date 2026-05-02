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
3. Minimal `canoliqctl` (or RPC handlers in `cmd/rpc/admin.go`) so canoLiq tx
   types can be built, signed (BLS12-381), and posted to `/v1/tx`.

Checklist:
- [ ] Plugin handshake: `go-plugin` connects to FSM, exchanges `PluginConfig`,
      and survives one `BeginBlock`/`EndBlock` cycle on the live socket
- [ ] `MessageSubsidy` proposal funds committee 2's reward pool; canoLiq's
      `EndBlock` observes the inflow and applies the 12% fee on a real chain
- [ ] Submit a real `MessageCanoliqDeposit`; verify cCNPY balance and pool
      growth via plugin-state RPC
- [ ] Submit `MessageCanoliqRedeem`, advance past `unbond_complete_height`,
      submit `MessageCanoliqClaimRedemption`; verify CNPY back in user account

### Phase 2 — Governance, buyback, treasury (deferred)
- [ ] `MessageCLIQVote` (parameter governance)
- [ ] `MessageCLIQStake` / `MessageCLIQUnstake`
- [ ] `MessageBuybackExecute` (drains buyback pool via Canopy DEX)
- [ ] `MessageDAOTreasurySpend` (multisig-gated)
- [ ] Insurance fund auto-routing (1–2% of treasury credits)

### Phase 3 — Monitoring & autonomy graduation (deferred)
- [ ] RPC endpoints (TVL, pool health, validator uptime, vesting status)
- [ ] Alerting hooks
- [ ] Graduation snapshot / migration tooling

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

## Phase 2 — Governance, buyback, treasury operations (outline)

Deferred. New tx types (sketch):

- `MessageCLIQVote` — propose/vote on parameter changes (fee bps, buyback frequency, validator-incentives split). Quorum = stake-weighted CLIQ.
- `MessageCLIQStake` / `MessageCLIQUnstake` — lock CLIQ for governance weight + boosts (whitepaper §4).
- `MessageBuybackExecute` — DAO-approved swap: drains `canoliq/buyback/pool` CNPY, buys CLIQ from the existing Canopy DEX (`fsm/dex.go`), either burns or distributes to CLIQ stakers per active vote.
- `MessageDAOTreasurySpend` — multisig-gated spend from `canoliq/treasury/canoliq` or `canoliq/treasury/cliq` above a threshold (whitepaper §9).
- Insurance fund: 1–2% of treasury auto-routed into `canoliq/insurance` pool on each treasury credit (whitepaper §11).

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
