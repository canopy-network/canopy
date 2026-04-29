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
make build
```

The same `go-plugin` binary serves either the send tutorial or canoLiq based
on the `CANOPY_PLUGIN_MODE` environment variable:

| Value | Plugin |
|---|---|
| `` (unset) or `contract` | send tutorial |
| `canoliq` | canoLiq liquid staking |

Optionally point `CANOLIQ_CONFIG` at a JSON file with `chainId`, `dataDirPath`,
and `genesisPath` overrides.

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

The plain `MessageSend` is also accepted so the canoLiq plugin is a drop-in
replacement for the tutorial when the `CANOPY_PLUGIN_MODE=canoliq` binary is
selected.

## State key layout

All canoLiq keys live under prefix `[]byte{10}` to stay clear of Canopy core
prefixes (`1`=accounts, `2`=pools, `7`=gov). See `state.go` for the helper
functions and key composition.

## Logs

```bash
tail -f /tmp/plugin/go-plugin.log
```
