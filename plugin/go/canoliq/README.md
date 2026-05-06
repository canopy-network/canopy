# canoLiq plugin

This package implements the canoLiq liquid-staking sub-chain as a Canopy
plugin. It accepts CNPY deposits, mints cCNPY at the current exchange rate,
applies a 12% protocol fee on staking rewards (split 40/30/15/15 between
users, the canoLiq DAO treasury, validator infrastructure, and CLIQ buyback),
and tracks CLIQ — the fixed-supply (100M) governance token — with cliff +
linear-vesting schedules.

The plugin is a sibling of `plugin/go/contract/`; it reuses the proto types
generated in `contract` package and runs its own FSM connection. See the full
implementation plan at `docs/plans/canoliq-implementation-plan.md`.

## Building

```bash
cd plugin/go
make build                        # builds the plugin process: ./go-plugin
go build -o canoliqctl ./canoliqctl/   # builds the operator CLI: ./canoliqctl
```

The same `go-plugin` binary serves either the send tutorial or canoLiq based
on the `CANOPY_PLUGIN_MODE` environment variable:

| Value | Plugin |
|---|---|
| `` (unset) or `contract` | send tutorial |
| `canoliq` | canoLiq liquid staking |

Optionally point `CANOLIQ_CONFIG` at a JSON file with `chainId`, `dataDirPath`,
and `genesisPath` overrides.

## canoliqctl — operator CLI

`canoliqctl` builds, signs, and submits canoLiq plugin transactions to a
running Canopy node. It uses the node's admin keystore (`/v1/admin/keystore-get`)
to fetch the BLS12-381 signer key and POSTs the signed envelope to `/v1/tx`.

Global flags (also configurable via env vars `CANOLIQCTL_RPC_URL`,
`CANOLIQCTL_ADMIN_URL`, `CANOLIQCTL_NETWORK_ID`, `CANOLIQCTL_CHAIN_ID`,
`CANOLIQCTL_FEE`, `CANOLIQCTL_PASSWORD`):

```
--rpc-url      node query RPC (default http://localhost:50002)
--admin-url    node admin RPC, hosts the keystore (default http://localhost:50003)
--network-id   Canopy network id (default 1)
--chain-id     canoLiq committee chain id (default 2)
--fee          tx fee in uCNPY (default 10000)
--password     keystore password — required
```

Phase 1 worked example (deposit → redeem → claim once unbond matures):

```bash
export CANOLIQCTL_PASSWORD=hunter2
./canoliqctl deposit alice 1000000           # 1 CNPY → cCNPY
./canoliqctl redeem  alice 250000            # burn 0.25 cCNPY, queue redemption
# advance past unbond_complete_height (Canopy's UnstakingBlocks param)
./canoliqctl claim   alice 0                 # claim redemption #0
```

Phase 2 commands (governance, staking, buyback, treasury):

```bash
./canoliqctl cliq-stake          alice 5000000
./canoliqctl cliq-unstake        alice 1000000
./canoliqctl cliq-claim-unstake  alice 0
./canoliqctl vote                alice <proposal-id> yes
./canoliqctl buyback-execute     alice <proposal-id>
./canoliqctl spend-execute       alice <proposal-id>
./canoliqctl multisig-approve    signer1 <spend-id>
./canoliqctl cliq-transfer       alice <to-hex> 1000000
./canoliqctl cliq-claim-vested   alice
```

`proposal-create` is intentionally not yet wired: its payload is a
`google.protobuf.Any` carrying one of three sub-types (param_change, buyback,
treasury_spend), each with a distinct argument surface. Until that lands,
construct proposals via in-process tests or hand-built JSON. The `vote`,
`buyback-execute`, and `spend-execute` commands work against any proposal id
regardless of how it was created.

## Registering canoLiq as a Canopy committee

1. **Pick a `chainId`.** Ensure it does not collide with an existing committee
   in your Canopy instance. The default is `2`; override via `CANOLIQ_CONFIG`.
2. **Validators opt in.** Each participating validator submits a
   `MessageEditStake` adding the canoLiq `chainId` to its
   `Validator.Committees[]` list. This is Canopy's existing restaking flow —
   no fork required.
3. **Request a subsidy.** Submit a `MessageSubsidy` proposal so the Canopy DAO
   funds the canoLiq committee reward pool. The plugin's `EndBlock` hook
   reads from this pool and applies the 12% fee.
4. **Run the plugin.** Set `~/.canopy/config.json` `"plugin": "canoliq"` (or
   start the binary directly with `CANOPY_PLUGIN_MODE=canoliq`). On first
   boot the plugin runs `Genesis` once, minting the 100M CLIQ supply to the
   recipients in `genesis.json` according to the bucket weights and vesting
   schedules.

## Genesis configuration

`genesis.json` lists CLIQ allocation buckets and per-recipient weights. The
sum of bucket bps must be `10000`; recipients within a bucket must also sum
to `10000`. Buckets with `cliffMonths == 0 && vestMonths == 0` mint to a
liquid CLIQ balance immediately; otherwise the plugin writes a
`VestingSchedule` and the recipient must call `MessageCLIQClaimVested` to
unlock vested CLIQ. Update the placeholder hex addresses (`...a1` … `...a7`)
before running mainnet.

## Transaction reference

| Tx | Effect |
|---|---|
| `MessageCanoliqDeposit` | Deposits CNPY → mints cCNPY |
| `MessageCanoliqRedeem` | Burns cCNPY → queues `Redemption` (matures after the unstaking window) |
| `MessageCanoliqClaimRedemption` | Withdraws a matured `Redemption` to the user's CNPY account |
| `MessageCLIQTransfer` | Transfers liquid (vested) CLIQ |
| `MessageCLIQClaimVested` | Unlocks newly-vested CLIQ across all of the caller's vesting schedules |
| `MessageCLIQStake` | Locks liquid CLIQ for governance weight |
| `MessageCLIQUnstake` | Queues an unbond record; voting weight drops immediately |
| `MessageCLIQClaimUnstake` | Returns matured CLIQ to the liquid balance |
| `MessageCLIQProposalCreate` | Opens a governance proposal (param change \| buyback \| treasury spend) |
| `MessageCLIQVote` | Votes yes/no/abstain on an active proposal |
| `MessageBuybackExecute` | Triggers a passed buyback proposal (BURN or DISTRIBUTE_STAKERS) |
| `MessageDAOTreasurySpend` | Triggers a passed treasury spend (timelock + multisig above threshold) |
| `MessageMultisigApprove` | Per-signer approval of an above-threshold spend |

The plain `MessageSend` is also accepted so the canoLiq plugin is a drop-in
replacement for the tutorial when the `CANOPY_PLUGIN_MODE=canoliq` binary is
selected.

## Governance lifecycle

CLIQ holders stake CLIQ for governance weight (and yield boosts in a future
release). All material protocol parameters — fee bps, the 40/30/15/15 split,
buyback mechanics, multisig membership, validator-onboarding criteria — flow
through the same proposal pipeline.

1. **Stake.** `MessageCLIQStake` locks liquid CLIQ; `staked_at_height` is
   recorded on the `CLIQStake` record. Unstake decrements voting weight
   immediately and queues an unbond record maturing after
   `cliq_unstaking_blocks` (default ~7 days).
2. **Propose.** `MessageCLIQProposalCreate` accepts a typed `google.protobuf.Any`
   payload — one of `ProposalParamChange`, `ProposalBuyback`, or
   `ProposalTreasurySpend`. The proposer must hold ≥ `min_stake_to_propose`.
   Total staked CLIQ is snapshotted into `Proposal.snapshot_total_staked`.
3. **Vote.** `MessageCLIQVote` casts a yes/no/abstain weighted by the voter's
   `CLIQStake.amount` *as of the proposal's creation height*. Stake added
   after creation is rejected (defeats flash-stake attacks).
4. **Tally.** On `BeginBlock` after `expiry_height`, the plugin tallies
   weights and applies the rules:
   - **Quorum**: `yes + no + abstain ≥ quorum_bps × snapshot_total_staked / 10000`
     (default 33%).
   - **Pass threshold**: `yes ≥ pass_threshold_bps × (yes + no) / 10000`
     (default 50%+1).
5. **Execute on pass.** Param-change payloads update `CanoliqParams` immediately
   and are observed by the next `ProcessRewards` sweep. Buyback and
   treasury-spend payloads write self-contained `BuybackOrder` / `TreasurySpend`
   records that subsequent triggers (`MessageBuybackExecute`,
   `MessageDAOTreasurySpend`) drain.

## Buyback workflow

Phase 2 ships an internal accounting swap (whitepaper §6 allows "market
buyback and burn or direct distribution governed by DAO"). A real on-chain
DEX route is deferred to Phase 3.

A passed `ProposalBuyback` carries `cnpy_amount`, `price_micro_cnpy_per_cliq`,
and `mode`. `MessageBuybackExecute`:

- Drains up to `cnpy_amount` from `canoliq/buyback/pool` and credits
  `canoliq/treasury/canoliq` by the same.
- Computes `cliq_acquired = cnpy_amount * 1_000_000 / price_micro_cnpy_per_cliq`
  and debits `canoliq/treasury/cliq` (DAO 15% bucket).
- **BURN**: decrements `globals.cliq_total_supply` and
  `globals.cliq_circulating_supply` by `cliq_acquired`.
- **DISTRIBUTE_STAKERS**: pro-rata credits all active CLIQ stakers' liquid
  balances; rounding remainder credited to the largest staker.

Re-execution is a no-op (`BuybackOrder.executed` flag).

## Treasury spend workflow

`ProposalTreasurySpend` carries `recipient`, `amount`, and `denomination`
(CNPY or CLIQ). Below `treasury_threshold` the spend executes as a single
step; above threshold it requires:

- A `timelock_blocks` delay before `executable_height` is reached.
- ≥ `multisig_threshold` of `multisig_signers` recording approvals via
  `MessageMultisigApprove`.

Initial multisig signer set is configured in `genesis.json`; subsequent
membership and threshold changes flow through a `ProposalParamChange`.

## Insurance fund

`ProcessRewards` skims `insurance_bps` (default 1500 = 15% of the treasury
slice → ≈1.5% of fee) into `canoliq/insurance/pool` per WP §11. Phase 3 will
add slashing-reimbursement disbursement; Phase 2 only seeds the pool.

## Per-validator pro-rata

The 15% validator-incentive slice is now distributed proportionally across
the canoLiq committee validator set, sourced from a plugin-internal
`ValidatorRegistry` singleton. Phase 1.5 will replace the registry source
with a real Canopy validator-set readback. When the registry is empty the
legacy aggregator key (`KeyForValidatorIncentives(committeeAggregatorAddr)`)
holds the full share — Phase 1 baseline behavior.

## State key layout

All canoLiq keys live under prefix `[]byte{10}` to stay clear of Canopy core
prefixes (`1`=accounts, `2`=pools, `7`=gov). See `state.go` for the helper
functions and key composition.

## Read-only HTTP query layer (Phase 3)

The plugin owns all canoLiq-prefixed state and is the only process that can
read it. To make plugin state visible to operators, dashboards, and
`canoliqctl`, a small read-only HTTP server runs **inside the plugin
process** alongside the FSM unix socket. All routes are read-only — they
never write state.

Enable it by setting `CANOLIQ_RPC_ADDR` (or `Config.RpcAddress`) to a
listen address. Empty/unset disables the server (the Phase 1 binary surface
is preserved).

```bash
export CANOLIQ_RPC_ADDR=127.0.0.1:8587
CANOPY_PLUGIN_MODE=canoliq ./go-plugin
```

The bundled Docker compose binds the plugin RPC to `0.0.0.0:8587` inside
each node container; the host port forwards are `8587` (node-1) and
`8588` (node-2).

### Routes (all `GET`)

| Path | Returns |
|---|---|
| `/v1/health` | `{height, genesisComplete, chainId}` |
| `/v1/globals` | `CanoliqGlobals` (singleton accounting record) |
| `/v1/params` | `CanoliqParams` (governance-tunable knobs) |
| `/v1/pools` | committee pool, treasury (CNPY/CLIQ), buyback, insurance, per-validator incentives |
| `/v1/account/{addr}` | composite per-address view: CNPY + cCNPY + liquid CLIQ + stake + validator-incentive + vesting |
| `/v1/proposals` | active proposal id list |
| `/v1/proposal/{id}` | full `Proposal` record |
| `/v1/vote/{id}/{voter}` | vote cast by `voter` on proposal `id` |
| `/v1/buyback/{id}` | post-execution `BuybackOrder` for a passed buyback proposal |
| `/v1/spends` | pending treasury-spend id list |
| `/v1/spend/{id}` | `TreasurySpend` record |
| `/v1/spend/{id}/approvals` | multisig approvals filtered to the *current* signer set |
| `/v1/validators` | `ValidatorRegistry` (canoLiq committee snapshot used for pro-rata) |
| `/v1/redemption/{addr}/{id}` | a queued cCNPY → CNPY redemption record |
| `/v1/vesting/{addr}` | every vesting schedule for an address with cumulative unlocked-to-date |

Address parameters accept `0x`-prefixed or bare 40-character hex.

Errors: `400` on malformed address, `404` on missing entity, `405` on
non-`GET`, `500` on plugin-internal failure (with a JSON error body).

### Known gaps

There are no per-address indexes for redemptions or unstaking-CLIQ
records. To list those for an address, the caller needs the id (returned
by the originating tx). A future iteration can add per-address indexes if
operator demand emerges.

## Logs

```bash
tail -f /tmp/plugin/go-plugin.log
```
