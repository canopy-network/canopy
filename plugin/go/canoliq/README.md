# canoLiq plugin

This package implements the canoLiq liquid-staking sub-chain as a Canopy
plugin. It accepts CNPY deposits, mints cCNPY at the current exchange rate,
applies a 12% protocol fee on staking rewards (split 40/30/15/15 between
users, the canoLiq DAO treasury, validator infrastructure, and CPLQ buyback),
and tracks CPLQ — the fixed-supply (100M) governance token — with cliff +
linear-vesting schedules.

The plugin is a sibling of `plugin/go/contract/`; it reuses the proto types
generated in `contract` package and runs its own FSM connection.

> **New to Canopy or liquid staking?** The conceptual reference — what canoLiq
> is, how the tokens and governance work, and the full API — lives in the
> documentation site under `docs/canoliq-site/` (run `yarn start` there, or read
> the `.mdx` sources directly). This README is the operator/developer runbook.
> Remaining roadmap work is tracked in `docs/plans/canoliq-release-plan.md`.

## Deployment profiles

canoLiq ships two pre-built deployment profiles. The plugin's
`Config.Profile` field selects which is active and unlocks
profile-specific safety behavior at startup.

| Profile | Config file | Genesis file | Notes |
|---|---|---|---|
| `localnet` | `canoliq-config.localnet.json` | `genesis.localnet.json` | Placeholder addresses (every bucket → single localnet key), 5-block redemption window |
| `testnet` | `canoliq-config.testnet.json` | `genesis.testnet.json` | TODO addresses (must be filled in before deploy), 14400-block redemption window template |

The plugin **refuses to start** with `profile=testnet` (or `mainnet`)
if any genesis bucket recipient still equals the well-known localnet
placeholder address. This catches the most common foot-gun: shipping
the localnet genesis into a real environment by mistake.

The bundled docker compose pins `localnet`. To run against a real
testnet, edit `genesis.testnet.json` with real recipient addresses
and override `CANOLIQ_CONFIG=/app/plugin/go/canoliq/canoliq-config.testnet.json`
in your runtime environment — see "Configuring a testnet deployment"
below.

## Quick start (localnet via Docker)

The fastest path to a running canoLiq chain on your machine — two
Canopy nodes plus a plugin process per node, built and signed by the
bundled `.docker/compose.yaml`. The validator set is pre-seeded so
both nodes are committee-2 (canoLiq) participants out of the box; the
plugin self-bootstraps Genesis on its first `BeginBlock` so 100M CPLQ
supply is minted (to the localnet placeholder address) before you can
hit the RPC.

Prerequisites: Docker (Compose v2) and `curl`. Optional: `jq` for the
JSON examples (or pipe through `python3 -m json.tool`).

### 1. Build the images

From repo root:

```bash
cd .docker
docker compose build
```

This compiles `go-plugin`, `canoliq`, and the canopy node binary into
two images, one per node. First build takes a couple of minutes;
incremental rebuilds are seconds because Go module caching.

### 2. Start the nodes

```bash
docker compose up -d
```

Both nodes start, the plugin attaches to its FSM unix socket
(`/tmp/plugin/plugin.sock` inside the container), Genesis runs once,
and rewards begin flowing into the canoLiq committee pool.

### 3. Wait for plugin Genesis

The plugin's `BeginBlock` self-bootstraps Genesis when chain
genesis.json carries no plugin section (which the bundled localnet
does not). Poll until `genesisComplete: true`:

```bash
until [ "$(curl -fsS http://127.0.0.1:8587/v1/health 2>/dev/null \
  | grep -o '"genesisComplete":true')" ]; do sleep 2; done
echo "ready"
```

This typically takes 6–12s (one to two blocks).

### 4. Probe the read-only RPC

The plugin exposes a read-only HTTP query surface on host port 8587
(node-1) and 8588 (node-2). Both serve the same routes; pick either.

```bash
curl -sS http://127.0.0.1:8587/v1/health   | jq
curl -sS http://127.0.0.1:8587/v1/globals  | jq
curl -sS http://127.0.0.1:8587/v1/params   | jq
curl -sS http://127.0.0.1:8587/v1/pools    | jq
```

After a few blocks of subsidy you should see non-zero `treasuryCnpy`,
`insurancePool`, `buybackPool`, and `validatorIncentives` reflecting
the 12% fee with the canonical 40/30/15/15 split (15% of treasury skim
→ insurance). Conservation: `globals.totalPooledCnpy + treasuryCnpy +
insurancePool + buybackPool + Σ validatorIncentives` equals the
cumulative post-DAO inflow.

The complete route list (snapshot-served + lazy-fulfilled) is in
[Read-only HTTP query layer](#read-only-http-query-layer-phase-3)
below.

### 5. Build `canoliqctl` (for tx submission)

The HTTP query layer is read-only. To submit transactions (deposit,
redeem, stake, vote, etc.) build the operator CLI from the host:

```bash
cd ../plugin/go
go build -o canoliqctl ./canoliqctl/
```

### 6. Submit a deposit

`canoliqctl` fetches the BLS12-381 signer key from the node's admin
keystore (`/v1/admin/keystore-get`), signs, and POSTs the envelope to
`/v1/tx`. It needs the keystore password.

```bash
export CANOLIQCTL_PASSWORD=test   # adjust for your keystore
./canoliqctl deposit <address-hex> 1000000
```

`<address-hex>` is the 40-char hex address for the keystore entry you
want to draft from. The default `--rpc-url=http://localhost:50002`
and `--admin-url=http://localhost:50003` line up with node-1's
exposed ports.

### 7. Verify the tx landed

Re-query the address through the lazy account route:

```bash
curl -sS http://127.0.0.1:8587/v1/account/0x<address-hex> | jq
```

You should see `ccnpy` non-zero (depositor received cCNPY) and `cnpy`
debited by `amount + fee`. The route blocks up to one block (~6s)
while EndBlock fulfills the lookup.

### 8. Tear down

```bash
cd ../../.docker
docker compose down
```

State persists in `.docker/volumes/node_*/canopy/` between restarts.

### Resetting to a fresh chain

Plugin Genesis runs once and short-circuits on subsequent boots
(idempotent on `globals.GenesisComplete`). To force a clean re-run
(regenerates 100M CPLQ supply, zeroes pools, replays subsidies from
height 1):

```bash
docker compose down
rm -rf volumes/node_*/canopy volumes/node_*/logs
rm -f  volumes/node_*/book.json volumes/node_*/polls.json
docker compose build   # if you changed code
docker compose up -d
```

Keep the other files (`config.json`, `genesis.json`, `keystore.json`,
`validator_key.json`, `proposals.json`) — those are config/identity,
not mutable state.

### Ports

| Service | node-1 | node-2 |
|---|---:|---:|
| Wallet | 50000 | 40000 |
| Explorer | 50001 | 40001 |
| RPC | 50002 | 40002 |
| Admin RPC | 50003 | 40003 |
| canoLiq plugin RPC | 8587 | 8588 |
| Debug pprof | 6060 | 6061 |
| Metrics | 9090 | 9091 |
| TCP P2P | 9001 | 9002 |

### Common pitfalls

- **`/v1/health` returns `genesisComplete: false` forever.** Either
  the plugin can't read its `genesis.json` (check `CANOLIQ_CONFIG`
  env var inside the container) or `Config.GenesisPath` is empty. The
  bundled compose sets both correctly; check
  `docker compose logs node-1 | grep -i canoliq` for plugin errors.
- **Empty reply from `/v1/...`.** Plugin process probably crashed.
  Check `docker compose logs node-1 | grep -iE 'fatal|panic|code:.*107'`.
  A `code 107: plugin response id is invalid` indicates a
  freestanding StateRead from outside an FSM lifecycle window — this
  was an early-design bug fixed by the snapshot model; if it
  reappears, see `AGENTS.md` and the saved memory.
- **Lazy routes time out (504).** Chain has stalled — `EndBlock` is
  not firing. Check that committee 2 has at least one validator
  opted in (the bundled genesis seeds two; if you've edited it,
  confirm `MessageEditStake` ran).

## Testnet deployment

Testnet rollout follows the same six-phase shape as production but
with the rigor dialed down where the cost of a mistake is recoverable:
audits are encouraged but not blocking, multisig signers can be
team-controlled, and "spin up a fresh testnet" is a viable recovery
path. The localnet quick-start above mints 100M CPLQ to one
placeholder key and uses a 5-block redemption window — both wrong
for any shared chain. The testnet profile keeps the same plugin
binary but loads a distinct genesis + runtime config, with safety
checks that refuse to start until placeholders are replaced.

### Phase 0 — Readiness gate

Lighter than production but skip these at your peril — most of them
are coordination tasks that can't be undone after first-block:

- [ ] **chainId reserved with the Canopy team.** No collision with
      any existing committee on the target testnet.
- [ ] **`MessageSubsidy` proposal queued or passed** on the Canopy
      DAO so the canoLiq committee pool will fund. Until it passes,
      `ProcessRewards` is a no-op.
- [ ] **Validator opt-in confirmed.** Every validator that should be
      on the canoLiq committee has run `MessageEditStake` adding
      `chainId` to its `Validator.Committees[]`. This is the sole
      source of truth: the plugin reconciles its `validatorRegistry`
      against live committee membership every block, so opt-ins after
      genesis are picked up on the next block.
- [ ] **Bucket recipient addresses identified.** Test wallets are
      fine, but they must not be the localnet placeholder
      (`851e90…d123`) — the safety check refuses that automatically.
      Document who controls each bucket so the team can recover keys
      if a tester rotates out.
- [ ] **Multisig signers chosen.** A test signer set is acceptable
      (e.g., five team members each with their own keystore). Pick
      a threshold (default 3) and document the password-recovery
      story.
- [ ] **Redemption window decided.** Set
      `redemptionUnstakingBlocks` to match the testnet's
      `valParams.UnstakingBlocks`. The template ships with `14400`
      (~24h at 6s blocks); raise or lower based on the actual
      testnet's block time and unstaking length.
- [ ] **Discord summary posted** with chainId, fee/split, CPLQ
      bucket distribution, plugin runtime contract.
- [ ] **Monitoring sketched out.** Even on testnet you want a
      uptime check on `/v1/health` per node — silent failure during
      a multi-day soak test wastes everyone's time.

### Phase 1 — Build the testnet genesis + config

Treat both files as **release artifacts** even on testnet — easier
to debug a soak failure six days in if you can pin to a specific
file pair than if someone hand-edited a file on the live host.

#### 1.1. Edit `genesis.testnet.json`

Replace each bucket `address` (currently `0000…0001` … `0000…0007`)
with the real testnet bucket-owner address. Within a bucket,
recipient bps must sum to 10000; the seven bucket bps must also sum
to 10000. The validator and partner buckets carry vesting
(`cliffMonths > 0` or `vestMonths > 0`) — leave those numbers
alone unless governance has a reason to alter them.

In the same file's `params` block:

- Swap `multisigSigners` from the `00000000…b1`–`00000000…b5`
  placeholders to real testnet signer addresses. Hex with or without
  `0x` prefix is accepted.
- Adjust `multisigThreshold` if you've listed more or fewer than
  five signers. The constraint is `multisigThreshold ≤
  len(multisigSigners)`; `ValidateParams` rejects the inverse.
- `treasuryThreshold` defaults to `1000000000` uCNPY (1k CNPY-eq).
  Below this, treasury spends execute single-step on a passing
  vote; above, multisig + timelock kicks in. Lower the threshold
  during testnet soak so multisig flow gets exercised on smaller
  spends.

In the same file's `validatorRegistry` block:

- Optional. Each entry is `(address, stake)` for a real opted-in
  testnet validator, with stake mirroring that validator's
  `Validator.StakedAmount`. The first `EndBlock` overwrites the block
  with live committee membership either way, so an empty registry is
  fine — it just means the aggregator key holds the validator slice
  until the first validator opts in.

#### 1.2. Edit `canoliq-config.testnet.json`

```json
{
  "profile": "testnet",
  "chainId": <reserved-chainId>,
  "dataDirPath": "/tmp/plugin",
  "genesisPath": "/app/plugin/go/canoliq/genesis.testnet.json",
  "rpcAddress": "0.0.0.0:8587",
  "redemptionUnstakingBlocks": <Canopy-testnet-valParams-UnstakingBlocks>
}
```

Two notes specific to testnet (production differs):

- **`rpcAddress: "0.0.0.0:8587"`** is acceptable on a private testnet
  behind a VPN or firewall. On a shared testnet exposed to the
  internet, prefer `127.0.0.1:8587` and front it with a reverse
  proxy + rate limit (the lazy routes can block a goroutine for up
  to one block).
- **`redemptionUnstakingBlocks` is profile-bound, not protocol-bound.**
  Pick the value that matches the chain you're deploying against;
  there's no "testnet default" because testnets vary.

#### 1.3. Hash-anchor (optional but recommended)

```bash
sha256sum plugin/go/canoliq/genesis.testnet.json
sha256sum plugin/go/canoliq/canoliq-config.testnet.json
```

Drop the hashes into the Discord announcement so signers and
validators can verify they're running the same files.

### Phase 2 — Pre-flight verification

Always exercise the production-shaped flow on a private chain
before the actual testnet — the tooling is identical and the cost
of a typo on testnet is "spin up a fresh chainId" which is annoying.

#### 2.1. Confirm safety banner + check

Build and run the plugin briefly to confirm the startup banner
reports your edits and the safety check passes:

```bash
cd plugin/go && go build -o go-plugin .
CANOPY_PLUGIN_MODE=canoliq \
  CANOLIQ_CONFIG=$PWD/canoliq/canoliq-config.testnet.json \
  ./go-plugin
# Expect first log line:
# canoliq: profile="testnet" chainId=… genesis="…testnet.json" rpc="0.0.0.0:8587" redemptionUnstakingBlocks=…
# If any placeholder address remains, the process exits with:
# canoliq: refusing to start profile="testnet" with localnet placeholder
# address in bucket "<name>" (set real bucket recipient addresses in …)
```

The plugin will block trying to connect to the FSM socket — that's
fine for the safety-check verification. Kill with Ctrl-C.

#### 2.2. Bucket reconciliation on a private chain

Bring up a local testnet image (copy `.docker/compose.yaml` to
`compose.testnet.yaml` and switch the `CANOLIQ_CONFIG` line to point
at `canoliq-config.testnet.json`):

```bash
docker compose -f .docker/compose.testnet.yaml up -d --build
until curl -fsS http://127.0.0.1:8587/v1/health 2>/dev/null \
  | grep -q '"genesisComplete":true'; do sleep 2; done
```

Verify each bucket recipient got the expected balance, summing to
exactly 100M × 10⁶ uCPLQ:

```bash
for ADDR in <validators> <liquidity> <community> <dao> <founders> <partners> <grants>; do
  echo "=== 0x$ADDR ==="
  curl -sS http://127.0.0.1:8587/v1/account/0x$ADDR | jq '{cplqLiquid, vestings: .vestings | map({totalAmount: .schedule.totalAmount, cliffHeight: .schedule.cliffHeight})}'
done
```

Expected totals (in uCPLQ):

| Bucket | uCPLQ | Liquid | Vesting |
|---|---:|:---:|:---:|
| Validators | 22,000,000,000,000 | — | ✓ |
| Liquidity | 15,000,000,000,000 | ✓ | — |
| Community | 20,000,000,000,000 | ✓ | — |
| DAO Treasury | 15,000,000,000,000 | ✓ | — |
| Founders | 12,000,000,000,000 | — | ✓ |
| Partners | 10,000,000,000,000 | — | ✓ |
| Grants | 6,000,000,000,000 | ✓ | — |

#### 2.3. Lifecycle smoke test

```bash
# Deposit
./canoliqctl --rpc-url http://localhost:50002 deposit <test-acct> 1000000

# Redeem (queues with the testnet-configured unstaking window)
./canoliqctl redeem <test-acct> 250000

# Wait redemptionUnstakingBlocks blocks; then claim
./canoliqctl claim <test-acct> 0

# Stake CPLQ + propose a no-op param-change to verify governance
./canoliqctl cplq-stake <test-acct> 5000000
./canoliqctl proposal-create param-change <test-acct> ./params-noop.json \
  --description "testnet smoke test"
./canoliqctl vote <test-acct> 1 yes
# Wait votingPeriodBlocks; verify the proposal tallies + executes via /v1/proposals
```

Anything that fails here will fail on the actual testnet; fix it
before cutover.

#### 2.4. Multisig flow rehearsal (lighter than production)

Submit one above-`treasuryThreshold` `proposal-create
treasury-spend`, vote it through, walk it across the timelock +
multisig path. You only need to confirm two things on testnet:

1. Below-threshold approvals → 503 / clear error.
2. Execution after both timelock and quorum succeeds.

Production rehearses all four failure modes (pre-timelock rejection,
replay rejection, etc.); testnet can defer those to the actual
testnet soak.

### Phase 3 — Cutover

Testnet supports only the fresh-chain path — there's no in-place
migration tool yet (Phase 3 §3 of the implementation plan).

1. **Confirm Canopy DAO state.** chainId reserved, `MessageSubsidy`
   proposal passed (or about to pass), validators opted in.
2. **Distribute the production-shaped image** to validator hosts:
   each canopy node runs with
   `CANOPY_PLUGIN_MODE=canoliq` and
   `CANOLIQ_CONFIG=/app/plugin/go/canoliq/canoliq-config.testnet.json`.
   Two paths:
   - **Direct:** set the env vars on the canopy process directly.
   - **Docker:** copy `.docker/compose.yaml` to
     `compose.testnet.yaml`, swap the `CANOLIQ_CONFIG` line, bring
     up with `docker compose -f compose.testnet.yaml up -d`. The
     Dockerfile already bundles both genesis + config variants, so
     no rebuild needed.
3. **First-block bootstrap.** Plugin's `BeginBlock` self-bootstrap
   mints 100M CPLQ to the testnet bucket addresses on first block
   observation. There is no second chance — if the genesis is
   wrong, you reserve a new chainId and start over.
4. **Verify on real chain.** Re-run §2.2 reconciliation against
   the testnet RPC. Confirm `/v1/health.genesisComplete=true`,
   `/v1/validators` matches the seeded set, `/v1/pools.committeePool`
   grows as the subsidy accrues.

### Phase 4 — Day-2 operations

Mostly identical to production:

- **Routine queries.** Poll `/v1/health` (every 30s),
  `/v1/globals` + `/v1/pools` (every 5m), `/v1/proposals` +
  `/v1/spends` (every block).
- **Routine governance.** Every parameter change, buyback, and
  treasury spend goes through `canoliqctl proposal-create`. The CLI
  needs the proposer's keystore password — testnet can use a shared
  service account; production cannot.
- **Plugin upgrades** are expected to be more frequent on testnet
  (it's where you find bugs). Each upgrade: tag a release, validators
  restart their plugin process. Canopy node stays up; plugin
  re-attaches to the FSM socket and resumes from stored state.

### Phase 5 — Incident response

Testnet's escape hatch — "spin up a fresh testnet chainId" — is
genuinely available and changes how you handle several incident
classes:

- **Compromised signer.** Submit `proposal-create param-change`
  removing the compromised key from `multisigSigners`. New signer
  set takes effect immediately on pass.
- **Suspicious proposal.** No admin override. Multisig + timelock
  is the defense — signers can refuse to approve, the timelock
  buys coordination time. Same as production.
- **Plugin crashes / unix-socket disconnect.** Restart the plugin
  with the same config. On reconnect it re-handshakes with the FSM
  and resumes; snapshot is empty until the next `EndBlock`. No
  state is lost.
- **State corruption suspected.** On testnet the right answer is
  often "spin up a fresh chainId rather than spend a week
  diagnosing". Document what you learned in the post-mortem so
  production benefits.
- **Stuck redemptions.** Check `/v1/redemption/{addr}/{id}` for the
  ones you know about; until §1.1-bis lands there's no enumeration
  surface, so users have to surface stuck ids themselves.

### Testnet deployment checklist

Paste-ready for a release ticket:

```
[ ] chainId reserved with Canopy team
[ ] MessageSubsidy proposal queued/passed
[ ] Validators opted into committee via MessageEditStake
[ ] Validator registry block matches opted-in set (bootstrap only; synced per block)
[ ] genesis.testnet.json reviewed; bucket addresses replaced
[ ] params block: multisigSigners + threshold replaced
[ ] canoliq-config.testnet.json: chainId + redemptionUnstakingBlocks set
[ ] Hashes of both files recorded
[ ] Local safety-check verification passed (banner + safety check)
[ ] Private-chain bucket reconciliation: 100M × 10⁶ uCPLQ exact
[ ] Private-chain lifecycle smoke test: deposit → redeem → claim
[ ] Private-chain governance smoke test: propose → vote → execute
[ ] Private-chain multisig rehearsal: above-threshold flow
[ ] Discord announcement posted (chainId, fee/split, hashes)
[ ] Monitoring + on-call sketched
```

### Profile safety check details

`SafetyCheck` runs immediately after the startup banner. Under
`profile=testnet` or `mainnet` it parses the genesis file and refuses
to proceed if any bucket recipient address (case- and prefix-folded)
matches the localnet placeholder
`851e90eaef1fa27debaee2c2591503bdeec1d123`. The check does not
validate that addresses are *correct* — only that they're not the
known-bad localnet seed. Genesis-schema validation (bps sums,
required fields) runs separately inside `validateGenesis`.

Localnet (or empty) profiles skip the check entirely.

## Production deployment

Production rollout is the testnet flow + audit + key-management
discipline + Canopy-DAO coordination + day-2 ops. The plugin code is
the same binary; the rigor goes into the inputs (addresses, signers,
chainId, redemption window) and the surrounding process. This section
is the runbook a release captain follows, in order.

### Phase 0 — Readiness gate (do not start without these)

Stop and resolve any unchecked item. Each one is a real foot-gun in
production:

- [ ] **Audit complete.** Whitepaper §11 calls this out: "Audits &
      formal verification: mandatory before mainnet launch." Run a
      canoliq-specific audit, not just a Canopy-core audit. The
      tokenomics math (12% fee + 40/30/15/15 split, insurance skim,
      vesting curves, governance snapshot semantics, multisig +
      timelock for above-threshold spends) is the high-value target.
- [ ] **Multisig key ceremony executed.** Each signer holds their key
      in cold storage; signers are operationally independent (no two
      under the same physical/legal authority). Document signer
      identities, public addresses, and recovery procedures off-chain.
- [ ] **Bucket recipient addresses finalized.** All seven buckets must
      point at multisig wallets or time-lock contracts owned by the
      named beneficiary class — never an EOA, never the same address
      twice, never the localnet placeholder. The safety check refuses
      `851e90…d123` automatically; you still have to verify the *real*
      addresses are correct.
- [ ] **Validator opt-in confirmed.** Every validator that should be
      on the canoLiq committee has run `MessageEditStake` adding the
      target chainId to its `Validator.Committees[]`. The plugin
      reconciles `validatorRegistry` against live committee membership
      every block, so this is the sole source of truth.
- [ ] **chainId reserved with the Canopy team.** No collision with
      any existing committee.
- [ ] **`MessageSubsidy` proposal queued or passed** on the Canopy
      DAO so the canoLiq committee pool will fund on first block.
- [ ] **Redemption window matches Canopy.** Set
      `redemptionUnstakingBlocks` to the live chain's
      `valParams.UnstakingBlocks` (typically thousands of blocks).
      Document the value with a citation to the chain it was read
      from.
- [ ] **Discord summary posted** announcing chainId, fee/split, CPLQ
      supply + bucket distribution, multisig signer set, plugin
      runtime contract.
- [ ] **Monitoring + on-call rotation in place.** At minimum: a
      uptime check on `/v1/health` per node, an alert when
      `genesisComplete` flips false (impossible normally → indicates
      state corruption), and a periodic snapshot of `/v1/pools` for
      anomaly review. Phase 3 §2 alerting is not required to ship,
      but if you don't have it, an operator must be paid to read
      `/v1/pools` daily.
- [ ] **Disaster recovery plan written.** What do you do if (a) a
      multisig signer is compromised, (b) the plugin crashes
      mid-block, (c) a buyback executes at a stale price, (d) you
      need to roll back? Decisions made in advance are decisions
      made well.

### Phase 1 — Build the production genesis + config

Treat both files as **release artifacts**: version-controlled, code-
reviewed, hash-anchored. Don't edit on the live host.

#### 1.1. Make a `genesis.production.json`

Copy `genesis.testnet.json` to `genesis.production.json` and edit:

```json
{
  "blocksPerYear": 5256000,
  "buckets": [
    { "name": "Validators & Infrastructure", "bps": 2200, "cliffMonths": 12, "vestMonths": 24,
      "recipients": [{ "address": "<multisig-validators>",   "bps": 10000 }]},
    { "name": "Liquidity Incentives",        "bps": 1500, "cliffMonths": 0,  "vestMonths": 0,
      "recipients": [{ "address": "<multisig-liquidity>",    "bps": 10000 }]},
    { "name": "Community & Airdrops",        "bps": 2000, "cliffMonths": 0,  "vestMonths": 0,
      "recipients": [{ "address": "<multisig-community>",    "bps": 10000 }]},
    { "name": "DAO Treasury",                "bps": 1500, "cliffMonths": 0,  "vestMonths": 0,
      "recipients": [{ "address": "<multisig-dao>",          "bps": 10000 }]},
    { "name": "Founders & Core Team",        "bps": 1200, "cliffMonths": 12, "vestMonths": 36,
      "recipients": [{ "address": "<timelock-founders>",     "bps": 10000 }]},
    { "name": "Strategic Partners",          "bps": 1000, "cliffMonths": 6,  "vestMonths": 12,
      "recipients": [{ "address": "<timelock-partners>",     "bps": 10000 }]},
    { "name": "Developer Grants & Ecosystem","bps": 600,  "cliffMonths": 0,  "vestMonths": 0,
      "recipients": [{ "address": "<multisig-grants>",       "bps": 10000 }]}
  ],
  "params": {
    "multisigSigners": [
      "<signer-1>", "<signer-2>", "<signer-3>",
      "<signer-4>", "<signer-5>", "<signer-6>", "<signer-7>"
    ],
    "multisigThreshold": 4,
    "treasuryThreshold": 1000000000
  },
  "validatorRegistry": [
    { "address": "<validator-1>", "stake": <validator-1-stakedAmount> },
    { "address": "<validator-2>", "stake": <validator-2-stakedAmount> }
  ]
}
```

Production multisig usually wants more signers and a higher threshold
than the localnet 5-of-3 example — 7-of-4 or 9-of-5 is typical. The
threshold must be ≤ `len(multisigSigners)`; `ValidateParams` rejects
the inverse.

Recipients within a bucket can be split across multiple addresses
(e.g., founders divided across N partners) — bps within a bucket must
sum to 10000 just like at the top level.

#### 1.2. Make a `canoliq-config.production.json`

```json
{
  "profile": "mainnet",
  "chainId": <reserved-chainId>,
  "dataDirPath": "/tmp/plugin",
  "genesisPath": "/app/plugin/go/canoliq/genesis.production.json",
  "rpcAddress": "127.0.0.1:8587",
  "redemptionUnstakingBlocks": <Canopy-valParams-UnstakingBlocks>
}
```

Key choices:

- **`profile: "mainnet"`** — activates the same safety check as
  `testnet` (refuses the localnet placeholder address) plus signals
  intent in the startup banner.
- **`rpcAddress: "127.0.0.1:8587"`** — bind to loopback only.
  Production should expose the query layer behind a reverse proxy
  with rate limiting + TLS, *not* directly to the internet. The lazy
  routes will block the calling goroutine for up to one block under
  load — an unauthenticated public endpoint is a trivial DoS surface.
- **`redemptionUnstakingBlocks`** — must match Canopy's
  `valParams.UnstakingBlocks` so cCNPY redemptions and Canopy's
  validator unbonding are consistent. Mismatch → users either
  redeem before the unbonding window (slashing exposure) or wait
  longer than necessary (UX regression).

#### 1.3. Hash-anchor both artifacts

```bash
sha256sum plugin/go/canoliq/genesis.production.json
sha256sum plugin/go/canoliq/canoliq-config.production.json
```

Publish the hashes alongside the Discord announcement so signers,
validators, and any third-party reviewer can independently verify the
files match what the team agreed to.

### Phase 2 — Pre-flight verification

Stand up a private chain with the production genesis before touching
the real Canopy network. This catches typos in addresses, signers,
and chainId that would otherwise create a permanent on-chain mistake.

#### 2.1. Build with the production files

Add a temporary `compose.production.yaml` that mounts the production
config:

```yaml
# Same shape as .docker/compose.yaml; only the env line changes.
environment:
  - CANOPY_PLUGIN_MODE=canoliq
  - CANOLIQ_CONFIG=/app/plugin/go/canoliq/canoliq-config.production.json
  - CANOLIQ_RPC_ADDR=0.0.0.0:8587
```

Make sure the Dockerfile is updated to also `COPY` the
`genesis.production.json` and `canoliq-config.production.json` into
the image. Build:

```bash
docker compose -f .docker/compose.production.yaml build
docker compose -f .docker/compose.production.yaml up -d
```

#### 2.2. Confirm safety banner + check

```bash
docker compose -f .docker/compose.production.yaml logs node-1 | grep -i canoliq
```

Expect a banner line:
```
canoliq: profile="mainnet" chainId=<your-chainId> genesis="…production.json" rpc="0.0.0.0:8587" redemptionUnstakingBlocks=<your-window>
```

If the safety check rejects the genesis (placeholder address, invalid
hex), the process exits before binding to the FSM socket — fix the
file and rebuild. **Do not** edit files on the live host to make the
check pass; rebuild from source.

#### 2.3. Reconcile the bucket distribution

Wait for `genesisComplete: true`, then verify every bucket recipient
got the expected balance (uCPLQ):

```bash
for ADDR in <multisig-validators> <multisig-liquidity> <multisig-community> \
            <multisig-dao> <timelock-founders> <timelock-partners> <multisig-grants>; do
  echo "=== 0x$ADDR ==="
  curl -sS http://127.0.0.1:8587/v1/account/0x$ADDR | jq '{cplqLiquid, vestings: .vestings | map({totalAmount: .schedule.totalAmount, cliffHeight: .schedule.cliffHeight, endHeight: .schedule.endHeight})}'
done
```

Expected totals (in uCPLQ, where 1 CPLQ = 10⁶ uCPLQ):

| Bucket | uCPLQ | Liquid | Vesting |
|---|---:|:---:|:---:|
| Validators | 22,000,000,000,000 | — | ✓ (24mo) |
| Liquidity | 15,000,000,000,000 | ✓ | — |
| Community | 20,000,000,000,000 | ✓ | — |
| DAO Treasury | 15,000,000,000,000 | ✓ | — |
| Founders | 12,000,000,000,000 | — | ✓ (24mo + 12mo cliff) |
| Partners | 10,000,000,000,000 | — | ✓ (18mo + 6mo cliff) |
| Grants | 6,000,000,000,000 | ✓ | — |
| **Total** | **100,000,000,000,000** | | |

Sum the per-bucket totals across `/v1/account/{addr}` results and
confirm exactly 100M × 10⁶ uCPLQ. Off-by-one in the sum means a
recipient list misallocates and the genesis is wrong.

#### 2.4. Smoke-test the full lifecycle

Use canoliqctl on the private chain to exercise:

```bash
# Deposit
./canoliqctl --rpc-url http://localhost:50002 \
  deposit <test-account> 1000000

# Redeem (will queue with the production unstaking window)
./canoliqctl redeem <test-account> 250000

# Wait redemptionUnstakingBlocks blocks, then claim
./canoliqctl claim <test-account> 0

# Stake CPLQ + propose a no-op param-change to verify governance
./canoliqctl cplq-stake <test-account> 5000000
./canoliqctl proposal-create param-change <test-account> ./params-noop.json \
  --description "production smoke test"
./canoliqctl vote <test-account> 1 yes
# Wait votingPeriodBlocks; verify proposal tallies + executes
```

Anything that fails on the private chain will fail on production —
fix it now, not later.

#### 2.5. Multisig flow rehearsal

Stand up the multisig signers on the private chain (real signing keys,
not test keys). Run an above-`treasuryThreshold` `proposal-create
treasury-spend`, vote it through, and walk it across the timelock +
multisig path:

```bash
# After proposal passes:
./canoliqctl --password=<signer-1-pw> multisig-approve <signer-1> <spend-id>
./canoliqctl --password=<signer-2-pw> multisig-approve <signer-2> <spend-id>
./canoliqctl --password=<signer-3-pw> multisig-approve <signer-3> <spend-id>
./canoliqctl --password=<signer-4-pw> multisig-approve <signer-4> <spend-id>
# 4-of-7 reached; wait for executable_height (timelock_blocks past pass);
# then execute
./canoliqctl --password=<exec-pw> spend-execute <executor> <proposal-id>
```

Verify (a) below-threshold approvals are rejected with a clear error,
(b) execution before timelock is rejected, (c) execution after both
timelock and quorum succeeds, (d) re-execution returns
`spend already executed`. Only after all four flows are confirmed
should the production multisig signers be considered ready.

### Phase 3 — Cutover

Once Phase 0–2 are green, the actual production cutover is short.
Two patterns depending on your existing infrastructure:

#### Path A — fresh chain (recommended for new sub-chains)

1. **Reserve the chainId** with the Canopy team. Confirm no other
   committee uses it.
2. **Land `MessageSubsidy`** on the Canopy DAO so the committee pool
   funds when the first block lands.
3. **Deploy the production image** to validator hosts (canoliq plugin
   only — no canopy code change). Each validator runs the canopy node
   binary with `CANOPY_PLUGIN_MODE=canoliq` and
   `CANOLIQ_CONFIG=/path/to/canoliq-config.production.json`.
4. **Validator opt-in** via `MessageEditStake` adding chainId. Opt-ins
   that land later are reconciled into the registry on the next block,
   but until the first one lands there is no committee stake to observe
   and no reward to distribute.
5. **First-block bootstrap.** The plugin's `BeginBlock` self-bootstrap
   runs on first observation, mints 100M CPLQ to the production
   bucket addresses, and seeds the validator registry. There is no
   second chance — if the genesis file is wrong, you start over from
   step 1 with a new chainId. (This is why Phase 2 is non-negotiable.)
6. **Verify.** Run `Phase 2.3` reconciliation against the real chain.
   Verify `/v1/health.genesisComplete=true`, `/v1/validators` matches
   the seeded set, and `/v1/pools.committeePool` is growing as the
   subsidy accrues.

#### Path B — migrating an existing canoLiq deployment

This implies you already have canoLiq state on the chain (running
under `profile=localnet` or similar dev config). **There is currently
no in-place migration tool** (Phase 3 §3 of the implementation plan
covers this). Until then: treat existing state as throwaway, take a
fresh chainId, deploy as Path A.

### Phase 4 — Day-2 operations

#### Routine queries

The full read surface lives at `:8587/v1/...` — see
[Read-only HTTP query layer](#read-only-http-query-layer-phase-3) for
the route list. Production dashboards typically poll:

- `/v1/health` (every 30s) — liveness
- `/v1/globals` (every 5m) — `totalPooledCnpy`, CPLQ supply,
  `pendingRedemptionCnpy`
- `/v1/pools` (every 5m) — fee-split health, treasury / buyback /
  insurance balances
- `/v1/proposals` (every block) — active governance
- `/v1/spends` (every block) — pending treasury spends

#### Routine governance

Every parameter change, buyback, and treasury spend goes through
`canoliqctl proposal-create`. The CLI requires the proposer's
keystore password — use a CI service account or a hardware-key
signer; never hardcode passwords.

For above-threshold treasury spends, the multisig signers each run
`canoliqctl multisig-approve` from their own host with their own
keystore. Track which signer has signed via
`/v1/spend/{id}/approvals`.

#### Upgrades

Plugin binary upgrades (e.g., bug fixes, new routes) are coordinated
with the validator set:

1. Audit the diff. Re-run the test suite + safety checks against the
   production genesis.
2. Tag a release; publish the image hash.
3. Validators upgrade their plugin process — canopy node stays up
   throughout because the plugin is a separate process. The plugin
   re-attaches to the FSM socket on restart and resumes from the
   stored state.

Param changes go through governance, never via config edit. The only
config knobs that don't have a governance-mutable equivalent are
`profile`, `chainId`, `dataDirPath`, `genesisPath` — all of which
are immutable post-deploy.

### Phase 5 — Incident response

#### A multisig signer is compromised

1. Submit `proposal-create param-change` with a new
   `multisigSigners` list excluding the compromised key.
2. Vote it through. The new signer set takes effect immediately on
   pass — `countMultisigApprovals` filters approvals against the
   *current* signer set, so any pending approvals from the removed
   signer become inert (see `AGENTS.md`, "Treasury spend").

#### Suspicious buyback or treasury spend

Both go through governance. There is no admin override. The defense
is the multisig + timelock — if a malicious proposal passes,
multisig signers can refuse to approve the resulting spend, and the
timelock buys time to coordinate.

#### Plugin crashes / unix-socket disconnect

The plugin is a separate process; restart it with the same config.
On reconnect it re-handshakes with the FSM and resumes. Snapshot is
empty until the next `EndBlock` (cold start serves zeros — see
"Snapshot model"); HTTP queries during that window return defaults.
No state is lost — everything lives in the FSM-managed state DB.

If `Plugin.snapshot` returns stale data after a long outage, the
fix is one block of activity — `EndBlock` will refresh.

#### State corruption suspected

Plugin-owned state is point-read only; suspicious values surface in
`/v1/globals` or `/v1/pools` first. Cross-check by enumerating
buckets via the address routes and summing CPLQ. If the supply
doesn't equal 100M × 10⁶ minus circulating, escalate to a Canopy
core engineer — the FSM's state DB is the canonical source of truth.

There is no rollback button. Recovery requires either (a) a
governance-passed `param_change` that compensates for the
divergence, (b) a Canopy-core state-machine fix, or (c) graduation
to a new chain (Phase 3 §3 — not yet built). Build a snapshot
export tool *before* you need it.

#### cCNPY supply desynced from pooled CNPY

The above covers CPLQ supply arithmetic; this is the separate
cCNPY/pooled-CNPY case. `totalCcnpySupply` can fall out of step with
`totalPooledCnpy` — a full redeem drains supply to 0 while pooled
retains dust (deliberate, see `fee.go:22-25`), and a desync
originating below the plugin (a base-chain reset, a rollback) can
do the same at any supply level. PR #34 stops normal reward accrual
from creating or worsening this once it's reached zero, but a
below-the-plugin desync is invisible to the plugin itself.

The `supply_pool_desync` alert (T6, `alerts.go`) is the detection
side: every `EndBlock` it asks `computeMint` what a reference 1 CNPY
deposit would mint right now and fires `crit` if that's below
`supplyDesyncFloorBps` (default 1%) of fair. Check `/v1/globals` for
`totalCcnpySupply` and `totalPooledCnpy` directly to confirm.

Recovery is the same shape as CPLQ state corruption — no rollback
button. On testnet, prefer a fresh chainId over diagnosing. On
mainnet: (a) a governance `param_change` that compensates (there is
currently no purpose-built proposal type for this specific
divergence — `ACTION_PROTOCOL_UPGRADE` is the closest existing
lever), or (b) escalate to a Canopy core engineer if the divergence
traces to base-chain state rather than anything the plugin wrote.

### Production deployment checklist

A condensed pre-flight you can paste into a release ticket:

```
[ ] Audit complete; signoff on file
[ ] genesis.production.json reviewed + sha256 hash published
[ ] canoliq-config.production.json reviewed + sha256 hash published
[ ] All seven bucket addresses verified against named beneficiaries
[ ] Multisig signers identified, keys in cold storage, threshold set
[ ] redemptionUnstakingBlocks matches Canopy live valParams
[ ] chainId reserved with Canopy team
[ ] MessageSubsidy proposal queued/passed
[ ] Validators opted into committee via MessageEditStake
[ ] Validator registry block matches opted-in set (bootstrap only; synced per block)
[ ] Private-chain bucket reconciliation: 100M × 10⁶ uCPLQ exact
[ ] Private-chain lifecycle smoke test: deposit → redeem → claim
[ ] Private-chain governance smoke test: propose → vote → execute
[ ] Private-chain multisig rehearsal: 4 flows verified
[ ] Discord announcement posted with hashes + chainId
[ ] Monitoring + on-call rotation live
[ ] Disaster recovery runbook reviewed by team
[ ] Image tagged + signed; deployment artifacts immutable
```

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
--chain-id     chain id of the NODE being submitted to (default 1)
--fee          tx fee in uCNPY (default 10000)
--password     keystore password — required
--no-wait      print the tx hash and exit without confirming the outcome
--wait-timeout how long to wait for inclusion or rejection (default 45s)
```

> **Commands wait for a real outcome by default.** Submitting is not succeeding: `/v1/tx` admits
> a transaction on `CheckBasic` alone — no chain-id check, no signature check, no state — so it
> returns a hash for transactions the next mempool re-check will reject and drop. `canoliqctl`
> therefore polls `tx-by-hash` for inclusion and `failed-txs` for rejection, printing the block a
> transaction landed in or exiting non-zero with the node's own reason. Pass `--no-wait` for the
> old fire-and-forget behaviour.
>
> **Global flags must precede the positional arguments** — `canoliqctl deposit --no-wait <address>
> <amount>`, not `… <address> <amount> --no-wait`. Go's flag parser stops at the first
> non-flag argument, so a trailing flag is silently ignored. This applies to every global flag,
> not just `--no-wait`.

> **`--chain-id` is the node's chain id, not the canoLiq committee id.** A transaction is signed
> for the chain it is submitted to; `fsm/transaction.go::CheckReplay` rejects a mismatch with
> `ErrWrongChainId` before the plugin is consulted. The committee id lives in the *plugin* config
> (`canoliq-config.*.json`) and only scopes fee pools and committee membership — testnet pairs
> node chain 1 with committee 42, mainnet with committee 19. Passing the committee id here fails
> silently: `/v1/tx` runs only `CheckBasic`, so it returns a hash, and the mempool re-check then
> drops the transaction with nothing printed.

> **If every pool reads zero and `/v1/health` shows `genesisComplete: false`, check `genesisPath`
> first.** The plugin self-bootstraps genesis from `BeginBlock` when the node's `genesis.json`
> carries no canoLiq section, but only when `genesisPath` is set. Left empty, it skips silently
> and the plugin runs forever uninitialized with `ProcessRewards` a no-op — a chain that looks
> healthy while canoLiq does nothing. It now logs a one-line warning when that happens.
>
> The config named by `CANOLIQ_CONFIG` is read **once at plugin startup** and is not reloaded per
> block, so fixing the JSON requires restarting the plugin process. Outside localnet, a
> `genesisPath` that is set but unreadable or malformed now refuses to start rather than
> deferring the error to a per-block failure.

Phase 1 worked example (deposit → redeem → claim once unbond matures):

```bash
export CANOLIQCTL_PASSWORD=hunter2
# Commands identify the signer by hex address, not a keystore nickname.
ADDR=851e90eaef1fa27debaee2c2591503bdeec1d123
./canoliqctl deposit $ADDR 1000000           # 1 CNPY → cCNPY
./canoliqctl redeem  $ADDR 250000            # burn 0.25 cCNPY, queue redemption
# advance past unbond_complete_height (Canopy's UnstakingBlocks param)
./canoliqctl claim   $ADDR 0                 # claim redemption #0
```

Phase 2 commands (governance, staking, buyback, treasury):

```bash
./canoliqctl cplq-stake          $ADDR 5000000              # optional: --lock <none|3m|6m|12m|24m>
./canoliqctl cplq-unstake        $ADDR 1000000
./canoliqctl cplq-claim-unstake  $ADDR 0
./canoliqctl vote                $ADDR <proposal-id> yes
./canoliqctl buyback-execute     $ADDR <proposal-id>
./canoliqctl spend-execute       $ADDR <proposal-id>
./canoliqctl multisig-approve    <signer-address> <spend-id>
./canoliqctl cplq-transfer       $ADDR <to-hex> 1000000
./canoliqctl cplq-claim-vested   $ADDR
```

The optional `--lock <tier>` on `cplq-stake` commits the stake to a vote-escrow
lock (`3m`/`6m`/`12m`/`24m`, default `none`), raising its voting multiplier (up
to 4×) and buyback reward boost. Locks only ever strengthen on re-stake, and the
stake cannot be unstaked until its lock height. See the docs site
(`docs/canoliq-site/docs/tokenomics/vote-escrow.mdx`) for the multiplier table.

### Creating proposals

`proposal-create` dispatches `MessageCPLQProposalCreate` with a
`google.protobuf.Any` payload. The CLI wires five payload types
(`param-change`, `buyback`, `treasury-spend`, `validator-eject`, `emergency`);
a sixth, `ProposalProtocolUpgrade`, is defined in proto but not yet exposed as a
CLI subcommand. The proposal's payload determines its governance
[action type / tier](../../../docs/canoliq-site/docs/governance/governance-tiers.mdx).
The proposer must hold ≥ `min_stake_to_propose` CPLQ at creation height.

```bash
# 1. Param change — full-set CanoliqParams replacement (loaded from JSON)
./canoliqctl proposal-create param-change $ADDR ./new-params.json \
    --description "lower fee from 12% to 8%"

# 2. Buyback — CNPY → CPLQ extraction at a vote-set price
./canoliqctl proposal-create buyback $ADDR 100000000 1500000 burn \
    --description "Q4 buyback and burn"
# args: cnpy-amount  price-uCNPY-per-CPLQ  mode (burn|distribute)

# 3. Treasury spend — transfer from canoliq treasury to a recipient
./canoliqctl proposal-create treasury-spend $ADDR 0xabc...123 50000000 cnpy \
    --description "infrastructure grant"
# args: recipient-hex  amount  denomination (cnpy|cplq)

# 4. Validator eject — remove a validator from the committee registry (F12)
./canoliqctl proposal-create validator-eject $ADDR 0xdef...456 \
    --description "eject unresponsive validator"
# args: validator-hex

# 5. Emergency — security-critical fast-track action with optional param diff (F13)
./canoliqctl proposal-create emergency $ADDR ./emergency-params.json \
    --description "freeze deposits pending investigation"
# args: optional params-json-file (omit for a signalling-only emergency)

# 6. OTC program fund — move CPLQ from treasury_cplq into the OTC lock program
#    budget. The only path by which the program is funded; nothing mints.
./canoliqctl proposal-create otc-program-fund $ADDR 500000000000 \
    --description "fund the OTC lock program with 500,000 CPLQ"
# args: amount-uCPLQ (capped at the treasury CPLQ balance at execution time)
```

The `param-change` JSON file uses the same shape as the `params` block
in `genesis.localnet.json` / `genesis.testnet.json`; copy that block to
a file, edit, and pass the path. `multisigSigners` are accepted as hex
strings (with or without `0x` prefix), exactly like the genesis files.

`canoliqctl` validates the file before submitting: it rejects one that
omits any key (a param-change replaces the whole set, so a missing key
would be written as zero rather than left alone) and runs the plugin's
own `ValidateParams` locally. A bad bps split or a
`multisigThreshold > len(multisigSigners)` fails at the terminal, not
after a voting period.

That check lives in the CLI, not on the chain. A proposal submitted
through raw RPC is only type-checked at creation and first meets
`ValidateParams` at execution, where an invalid payload is dropped after
burning a full voting period.

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
   boot the plugin runs `Genesis` once, minting the 100M CPLQ supply to the
   recipients in `genesis.json` according to the bucket weights and vesting
   schedules.

## Genesis configuration

`genesis.json` lists CPLQ allocation buckets and per-recipient weights. The
sum of bucket bps must be `10000`; recipients within a bucket must also sum
to `10000`. Buckets with `cliffMonths == 0 && vestMonths == 0` mint to a
liquid CPLQ balance immediately; otherwise the plugin writes a
`VestingSchedule` and the recipient must call `MessageCPLQClaimVested` to
unlock vested CPLQ. Update the placeholder hex addresses (`...a1` … `...a7`)
before running mainnet.

⚠️ **Buckets with `0/0` are not all "no schedule".** For *Liquidity Incentives*
and *Community & Airdrops* the on-chain `0/0` is intentional: the tokenomics
schedule (24-month DAO-controlled emission for Liquidity, 12-month snapshot-based
linear emission for Community & Airdrops) is enforced **off-chain by the bucket
recipient** — a distributor contract or controlled multisig — not by the genesis
vesting mechanism. The recipient address you supply for these two buckets must
honor that schedule; otherwise the full allocation goes liquid at TGE. The other
`0/0` buckets (*DAO Treasury*, *Dev Grants & Ecosystem*) intentionally have no
schedule — Treasury is governed by timelock-multisig spend rules, and Dev Grants
release in milestone-gated tranches at the DAO's discretion.

## Transaction reference

| Tx | Effect |
|---|---|
| `MessageCanoliqDeposit` | Deposits CNPY → mints cCNPY |
| `MessageCanoliqRedeem` | Burns cCNPY → queues `Redemption` (matures after the unstaking window) |
| `MessageCanoliqClaimRedemption` | Withdraws a matured `Redemption` to the user's CNPY account |
| `MessageCPLQTransfer` | Transfers liquid (vested) CPLQ |
| `MessageCPLQClaimVested` | Unlocks newly-vested CPLQ across all of the caller's vesting schedules |
| `MessageCPLQStake` | Locks liquid CPLQ for governance weight |
| `MessageCPLQUnstake` | Queues an unbond record; voting weight drops immediately |
| `MessageCPLQClaimUnstake` | Returns matured CPLQ to the liquid balance |
| `MessageCPLQProposalCreate` | Opens a governance proposal (param change \| buyback \| treasury spend) |
| `MessageCPLQVote` | Votes yes/no/abstain on an active proposal |
| `MessageBuybackExecute` | Triggers a passed buyback proposal (BURN or DISTRIBUTE_STAKERS) |
| `MessageDAOTreasurySpend` | Triggers a passed treasury spend (timelock + multisig above threshold) |
| `MessageMultisigApprove` | Per-signer approval of an above-threshold spend |
| `MessageOTCLockCreate` | Locks cCNPY for a 90d/120d term and reserves its CPLQ reward from the program budget |
| `MessageOTCLockClaim` | Releases a matured lock: returns the cCNPY and pays the reserved CPLQ |
| `MessageOTCLockCancel` | Exits a lock early: returns the cCNPY, forfeits the whole CPLQ reward |

The plain `MessageSend` is also accepted so the canoLiq plugin is a drop-in
replacement for the tutorial when the `CANOPY_PLUGIN_MODE=canoliq` binary is
selected.

## OTC lock program

Term-lock for cCNPY holders: lock cCNPY for a fixed tier, receive a one-off
CPLQ reward at maturity.

| Tier | Duration | Blocks | Reward | Param |
|---|---|---|---|---|
| `OTC_LOCK_90D` | 90 days | 1,296,000 | 5% | `otcTier90Bps` |
| `OTC_LOCK_120D` | 120 days | 1,728,000 | 8% | `otcTier120Bps` |

The reward is a **1:1 micro-unit quantity conversion**,
`reward_uCPLQ = locked_uccnpy * tier_bps / 10000`. There is no price and no
oracle, which keeps the whitepaper's "no external price oracle on the core
yield path" commitment intact. At the default minimum of 50,000 cCNPY a 90-day
position reserves 2,500 CPLQ.

### Funding

CPLQ supply is fixed at genesis and nothing mints, so every uCPLQ the program
pays already exists. The budget is filled by a passed `ProposalOTCProgramFund`,
which moves CPLQ from `treasury_cplq` into `canoliq/otc_budget/available`.

That path depends on the treasury actually holding CPLQ, which before this
change it never did: `applyGenesisBuckets` only wrote per-address balances, so
`treasury_cplq` read zero forever and both of its consumers — `SPEND_CPLQ`
treasury spends and every `MessageBuybackExecute` — failed their balance guard
on any real chain. A genesis bucket now credits it directly:

```json
{ "name": "DAO Treasury (canoLiq)", "bps": 1500, "destination": "treasury" }
```

A treasury-destined bucket takes no recipients, cannot vest, and does **not**
count toward `cplq_circulating_supply` — treasury holdings are not circulating.

### Budget accounting

Two scalars under domain 30: `available` (unreserved) and `reserved`
(committed to open positions).

| Event | available | reserved | circulating |
|---|---|---|---|
| Fund proposal passes | `+= amount` | — | unchanged |
| Lock created | `-= reward` | `+= reward` | unchanged |
| Claim at maturity | — | `-= reward` | `+= reward` |
| Early exit | `+= reward` | `-= reward` | unchanged |

Reserving at lock time, and rejecting the lock when `available < reward`, is
what makes the cap hard: no matured position can outrun what the program can
pay. The circulating bump on claim is required by the L4 supply invariant,
since the reward leaves a non-circulating reserve for a liquid balance. A
forfeited reward never circulates, so cancel must not bump.

### Lock mechanics

A lock **moves** the cCNPY out of the holder's balance into the position
record, the same way vesting and vote-escrow lock tokens, rather than
annotating the balance. Two consequences worth knowing:

- `DeliverMessageCanoliqRedeem` needs no change. The balance it reads is
  already reduced, so locked cCNPY is simply not there to redeem.
- `globals.total_ccnpy_supply` is deliberately **untouched**. The cCNPY still
  exists, it has only changed custody. Leaving the supply alone is exactly what
  keeps a locked position appreciating at the pool's normal exchange rate for
  the whole term. Decrementing it would silently stop locked positions earning
  and hand every unlocked holder a windfall.

Maturity is user-driven: nothing scans open positions per block, so the cost of
the program to block production stays flat as it grows.

`Cancel` is a separate message from `Claim`, and is refused at or after
`mature_height`. By then the reward is earned, so cancelling would destroy it
for nothing when a claim would pay. Forfeiture should always be deliberate.

### Tiers are not `LockTier`

`LockTier` is a closed 3/6/12/24-month set whose only consumers are governance
vote weight and buyback distribution shares, neither of which this program
grants. Its ordering is also load-bearing — `DeliverMessageCPLQStake` uses
`msg.LockTier > stake.LockTier` as "stronger tier" — so appending a 4-month
value would rank it above 24 months. `OTCLockTier` is a separate enum.

### CLI

```
canoliqctl otc-lock <address> <ccnpy-amount> <90d|120d>
canoliqctl otc-lock-claim <address> <lock-id>
canoliqctl otc-lock-cancel <address> <lock-id>   # forfeits the CPLQ reward
```

Open positions appear under `otcLocks` on `GET /v1/account/{addr}`; the program
budget appears as `otcBudgetAvailable` / `otcBudgetReserved` on `GET /v1/pools`.

## Governance lifecycle

CPLQ holders stake CPLQ for governance weight (and yield boosts in a future
release). All material protocol parameters — fee bps, the 40/30/15/15 split,
buyback mechanics, multisig membership, validator-onboarding criteria — flow
through the same proposal pipeline.

1. **Stake.** `MessageCPLQStake` locks liquid CPLQ; `staked_at_height` is
   recorded on the `CPLQStake` record. Unstake decrements voting weight
   immediately and queues an unbond record maturing after
   `cplq_unstaking_blocks` (default ~7 days).
2. **Propose.** `MessageCPLQProposalCreate` accepts a typed `google.protobuf.Any`
   payload — one of `ProposalParamChange`, `ProposalBuyback`, or
   `ProposalTreasurySpend`. The proposer must hold ≥ `min_stake_to_propose`.
   Total staked CPLQ is snapshotted into `Proposal.snapshot_total_staked`.
3. **Vote.** `MessageCPLQVote` casts a yes/no/abstain weighted by the voter's
   `CPLQStake.amount` *as of the proposal's creation height*. Stake added
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

A passed `ProposalBuyback` carries `cnpy_amount`, `price_micro_cnpy_per_cplq`,
and `mode`. `MessageBuybackExecute`:

- Drains up to `cnpy_amount` from `canoliq/buyback/pool` and credits
  `canoliq/treasury/canoliq` by the same.
- Computes `cplq_acquired = cnpy_amount * 1_000_000 / price_micro_cnpy_per_cplq`
  and debits `canoliq/treasury/cplq` (DAO 15% bucket).
- **BURN**: decrements `globals.cplq_total_supply` and
  `globals.cplq_circulating_supply` by `cplq_acquired`.
- **DISTRIBUTE_STAKERS**: pro-rata credits all active CPLQ stakers' liquid
  balances; rounding remainder credited to the largest staker.

Re-execution is a no-op (`BuybackOrder.executed` flag).

## Treasury spend workflow

`ProposalTreasurySpend` carries `recipient`, `amount`, and `denomination`
(CNPY or CPLQ). Below `treasury_threshold` the spend executes as a single
step; above threshold it requires:

- A `timelock_blocks` delay before `executable_height` is reached.
- ≥ `multisig_threshold` of `multisig_signers` recording approvals via
  `MessageMultisigApprove`.

Initial multisig signer set is configured in `genesis.json`; subsequent
membership and threshold changes flow through a `ProposalParamChange`.

## Insurance fund

`ProcessRewards` skims `insurance_bps` (default `500` = 5% of the treasury
slice → ≈0.5% of fee) into `canoliq/insurance/pool` per WP §9.2. The skim
auto-disables once the pool reaches the `insurance_target_bps` target
(default 500 = 5% of `peak_tvl_ucnpy`), tracked from T4 onward.

## Committee validator registry

The plugin-internal `ValidatorRegistry` singleton is the member set the 15%
validator-incentive pro-rata runs against, and the record of which of those
members' bonds canoLiq actually owns. It is **reconciled against Canopy's live
committee membership on every block**, inside `EndBlock` before the sweep
(`registry.go::syncCommitteeRegistry`): the plugin range-reads every
`lib.Validator` record and keeps those whose `Committees[]` contains this
chain id — the same derivation the FSM uses (`getValidatorSet`), and the
only one available since protocol v2 stopped writing per-committee index
keys.

So a validator that opts in post-genesis with `tx-stake` /
`MessageEditStake` starts earning on the next block, and one that drops
the committee stops, with no genesis edit or governance vote. Each entry's
`stake` is refreshed to the validator's live `StakedAmount`, which serves
as both the pro-rata weight and the next block's per-validator reward
baseline.

### Membership pays the incentive slice; ownership earns the reward

Being on the committee and being canoLiq's stake are different things, and the
registry tracks them separately.

Every member, owned or not, draws its stake-weighted share of the 15%
validator-incentive slice. That slice pays operators for running the committee,
which they do regardless of whose CNPY is bonded in their record.

Only **owned** members contribute to the reward `R` that lifts the cCNPY
exchange rate. A member is owned when its Canopy `Validator.output` address
appears in `params.stakeOutputAddresses`. Canopy returns a bond and its
early-withdrawal rewards to `output`, so that address is the record's economic
beneficiary; an operator's bond is their own collateral, and per WP §1.1 it is
their own CNPY that gets slashed, not depositors'.

Crediting every member's growth to cCNPY holders hands them other people's
yield. On mainnet committee 29 it took a 10 CNPY deposit to ~562x in forty
minutes. WP §3.3 is explicit that `R` is canoLiq's stake-weighted share.

**`stakeOutputAddresses` ships empty, which means `R = 0`.** That is the
correct reading when the protocol owns no bonded position, and the safe
direction regardless: under-crediting truthfully beats minting yield against
stake canoLiq does not own. The `owned_stake_missing` alert fires when the
committee is earning and canoLiq owns none of it, so this cannot pass unnoticed.

To start earning, two things have to be true, and neither is checkable on
chain:

1. A Canopy validator or delegate record exists whose **entire** bond is
   canoLiq principal, with `output` set to an address canoLiq controls. Canopy
   gives each record one `output`, so a mixed bond — operator collateral plus
   pool CNPY — pointed at a listed address re-creates the same
   over-attribution.
2. That record sets `compound=true` and is not unstaking. Otherwise Canopy pays
   its reward to the output address as liquid CNPY, which the stake-growth
   observation cannot see; the `owned_bond_not_compounding` alert flags it, and
   the CNPY is recoverable by hand but not through the exchange rate.

Then add the address with a param-change proposal. Note that a param change is
a **full-set replacement**: carry the existing addresses forward, or the
omitted ones stop earning.

Two further consequences worth knowing:

- **A newly admitted member's bond is not reward.** The observation sums
  per-validator growth over members that were already registered, so
  joining the committee with a large bond does not credit that bond to
  cCNPY holders. See `ProcessRewards`' doc comment for the aggregate cap
  that covers the mirror-image case (a stale per-validator baseline).
- **Ejection is a tombstone.** Because the registry is rebuilt from live
  membership, a passed `validator-eject` proposal (F12) also writes
  `KeyForEjectedValidator(addr)`; the sync skips tombstoned addresses.
  The operator keeps its Canopy stake and keeps earning Canopy committee
  reward — canoLiq simply stops observing and crediting it. There is no
  un-eject message: re-admitting requires clearing the tombstone.

The `validatorRegistry` block in `genesis.localnet.json` /
`genesis.testnet.json` still seeds the registry at genesis, but it is now
only a bootstrap convenience — the first `EndBlock` overwrites it with
live membership:

```json
"validatorRegistry": [
  { "address": "851e90eaef1fa27debaee2c2591503bdeec1d123", "stake": 1000000000 },
  { "address": "02cd4e5eb53ea665702042a6ed6d31d616054dc5", "stake": 1000000000 }
]
```

When the registry is **empty** — no validator has opted into the committee
yet — the legacy aggregator key
(`KeyForValidatorIncentives(committeeAggregatorAddr)`) holds the full
share; there is nothing to observe, so no reward is distributed either.

The genesis block seeds `stake` and membership only. Ownership is never seeded
from `validatorRegistry`: it is recomputed every block from `output` against
`params.stakeOutputAddresses`, so the way to grant it is the params list, not
the registry.

## State key layout

All canoLiq keys live under prefix `[]byte{20}` to stay clear of Canopy core
prefixes, which reserve the single-byte range `1`-`15` (`1`=accounts, `2`=pools,
`7`=gov, `10`=supply, …). Writing under a reserved prefix makes `FSM.StateWrite`
panic, so canoLiq uses `20` and declares it in
`CanoliqConfig.CustomStatePrefixes`. See `state.go` for the helper functions and
key composition.

## Read-only HTTP query layer (Phase 3)

The plugin owns all canoLiq-prefixed state and is the only process that
can read it. To expose plugin state to operators and dashboards, a small
read-only HTTP server runs **inside the plugin process** alongside the
FSM unix socket.

Enable it by setting `CANOLIQ_RPC_ADDR` (or `Config.RpcAddress`) to a
listen address. Empty/unset disables the server.

```bash
export CANOLIQ_RPC_ADDR=127.0.0.1:8587
CANOPY_PLUGIN_MODE=canoliq ./go-plugin
```

The bundled Docker compose binds the plugin RPC to `0.0.0.0:8587` inside
each container; host ports are `8587` (node-1) and `8588` (node-2).

### Snapshot model

The Canopy FSM rejects plugin-initiated `StateRead` calls whose request
ID is not from an in-flight FSM lifecycle call. Freestanding reads from
an HTTP handler therefore cannot work. Instead, the plugin builds a
**snapshot** of canoliq-owned state inside `EndBlock` (where the
FSM-originated request ID is valid) and HTTP handlers serve from that
frozen snapshot — no plugin↔FSM round-trip per request. The snapshot is
swapped atomically (`sync/atomic.Pointer[Snapshot]`).

Consequence: query responses are **stale by up to one block**. For
liquid-staking monitoring, that is acceptable — operators care about
trends, not single-block precision.

Cold start (before the first `EndBlock`) returns sane defaults: zero
height, `genesisComplete=false`, `DefaultParams()`.

### Routes (all `GET`)

Snapshot-served (sub-millisecond, stale by ≤1 block):

| Path | Returns |
|---|---|
| `/v1/health` | `{height, genesisComplete, chainId}` |
| `/v1/globals` | `CanoliqGlobals` (singleton accounting record) |
| `/v1/params` | `CanoliqParams` (governance-tunable knobs) |
| `/v1/pools` | committee pool, treasury (CNPY/CPLQ), buyback, insurance, per-validator incentives |
| `/v1/proposals` | `{ids: [active proposal ids]}` |
| `/v1/proposal/{id}` | full `Proposal` record (404 if not in active set) |
| `/v1/spends` | `{ids: [pending spend ids]}` |
| `/v1/spend/{id}` | `TreasurySpend` record |
| `/v1/spend/{id}/approvals` | multisig approvals filtered to the *current* signer set |
| `/v1/validators` | `ValidatorRegistry` (committee snapshot used for pro-rata) |
| `/v1/stakers` | `{stakers: [{address, amount, stakedAtHeight}]}` from `CPLQStakeIndex` |
| `/v1/graduation` | five autonomy-graduation metrics + composite `eligible` (T5) |
| `/v1/restaking` | per-Canopy-committee restaking policy + observed exposure + drift/compliance |

Lazy-fulfilled per-address (latency: up to one block ≈ 6s on localnet):

| Path | Returns |
|---|---|
| `/v1/account/{addr}` | composite: CNPY + cCNPY + liquid CPLQ + stake + validator-incentive + vesting |
| `/v1/vesting/{addr}` | every vesting schedule with cumulative unlocked-to-date |
| `/v1/redemption/{addr}/{id}` | one redemption record |
| `/v1/vote/{id}/{voter}` | one vote record |
| `/v1/buyback/{id}` | post-execution `BuybackOrder` |

Errors: `400` malformed input, `404` missing entity, `405` non-`GET`,
`500` internal error, `503` lazy queue saturated, `504` lazy drain
timed out (chain stalled).

The lazy routes block the HTTP request until the next `EndBlock`
drains the query queue. Client disconnects (e.g. via `ctx` timeout)
cancel the wait promptly. Pending unstakes and per-address
redemption listings are *not yet* exposed — they'd need new
write-side indexes; tracked as future work.

## Push alerts (webhooks)

The RPC surface above is *pull* (poll it from a dashboard). For unattended
monitoring the plugin can also *push* alerts to a webhook (Slack, Discord,
or any JSON receiver) when an on-chain condition trips. Disabled by default.

Enable by setting `CANOLIQ_ALERT_URL` (or the `alerts` block in
`CANOLIQ_CONFIG`):

```bash
export CANOLIQ_ALERT_URL=https://hooks.slack.com/services/...
```

Full config block (all knobs optional except `webhookUrl`):

```json
"alerts": {
  "webhookUrl": "https://hooks.slack.com/services/...",
  "authHeader": "Bearer <token>",          // optional Authorization header
  "format": "slack",                         // json (default) | slack | discord
  "windowBlocks": 100,                       // tumbling window for drain/drop
  "drainAlertBps": 5000,                     // buyback drain > 50% over window
  "concentrationAlertBps": 6600,             // one validator > 66% of stake
  "tvlDropBps": 2000,                        // TVL drop > 20% over window
  "defaultMinIntervalBlocks": 100,           // debounce between re-fires
  "minIntervalBlocks": { "tvl_drop": 50 },   // per-kind debounce override
  "stuckRedemptionCount": 10,                // mature unclaimed redemptions
  "supplyDesyncFloorBps": 100                // reference-deposit mint floor, 1%
}
```

### Conditions

| Kind | Severity | Fires when |
|---|---|---|
| `buyback_drain` | warn | buyback pool drains > `drainAlertBps` within a `windowBlocks` window |
| `validator_concentration` | warn | one committee validator holds > `concentrationAlertBps` of total stake |
| `tvl_drop` | crit | total pooled CNPY drops > `tvlDropBps` within a `windowBlocks` window |
| `stuck_redemption` | crit | mature unclaimed redemptions exceed `stuckRedemptionCount` |
| `supply_pool_desync` | crit | a reference 1 CNPY deposit would mint < `supplyDesyncFloorBps` of fair cCNPY — see [State corruption suspected](#state-corruption-suspected) |

Drain/drop use a **tumbling** window: the baseline re-anchors every
`windowBlocks` blocks (not a true sliding window — simpler, no per-block
ring buffer). Concentration, stuck-redemption, and supply-pool-desync are
all instantaneous state checks, not rate-of-change.

### Payload

The canonical JSON envelope:

```json
{
  "kind": "buyback_drain",
  "height": 12345,
  "severity": "warn",
  "message": "buyback pool draining faster than threshold",
  "details": { "schemaVersion": 1, "baseline": 1000, "current": 400, "dropBps": 6000, "thresholdBps": 5000 }
}
```

`format: "slack"` sends `{"text": "[warn] canoLiq buyback_drain @ h12345: ..."}`;
`format: "discord"` sends the same string as `{"content": "..."}`.

### Delivery semantics

- **Non-blocking:** POSTs run on a goroutine with a 5s timeout, so a slow or
  dead receiver never stalls `EndBlock` / consensus. Failures log at WARN.
- **Debounced:** each kind re-fires at most once per `minIntervalBlocks`. The
  watermark lives in state (`KeyForAlertState(kind)`) so it survives restarts.
- **Auto-resolving:** when a condition clears, its watermark resets, so the
  next occurrence fires immediately.

## Logs

```bash
tail -f /tmp/plugin/go-plugin.log
```
