# canoLiq mainnet activation — status, blockers, and next steps

_Last updated: 2026-09-23. Written after a deployment/verification session covering committee 29 (chain id 29) post-graduation._

## Context

canoLiq graduated on Canopy mainnet on 2026-09-22 (chain_id: 29, per the Launchpad API). Two validators are currently known to be staked on committee 29:

| Validator | Address | Stake | netAddress | Operator |
|---|---|---|---|---|
| Ours | `5c551fbcaf911f8c78ea505fe844268402f800fb` | 309,067.91 CNPY (~97% of known committee stake) | `tcp://node1.canoliq.org` | us |
| `val-a` | `737ef80e8476450b6970dd6c1e44c93699c8e0c0` | 10,000 CNPY | `tcp://cnlq.val-a.grad.app.canopynetwork.org` | Canopy Foundation |

Both are core-Canopy-only right now — **neither has the canoLiq plugin actually producing real liquid-staking state**. This doc explains why, what we verified, and what's blocking a real activation.

## What we want to do

Activate the canoLiq plugin for real on **our own validator** (`node29` on the box that holds the 309k CNPY stake), since it represents the supermajority of known committee-29 stake and would become the canonical post-activation state the rest of the committee has to catch up to. Coordinate the exact timing with Canopy Foundation so `val-a` (and any future committee-29 validator) converges on the same state rather than forking.

## Key findings this session

### 1. The plugin genuinely writes into core consensus state — confirmed against source, not assumed

Earlier in this session we mistakenly concluded (from a block-hash comparison) that the plugin was read-only and couldn't cause a consensus fork. That was wrong. Verified directly in `fsm/state.go`:

```go
func (s *StateMachine) StateWrite(request *lib.PluginStateWriteRequest) (...) {
    ...
    if err = s.Set(setRequest.Key, setRequest.Value); err != nil { ... }
```

`s.Set` → `s.Store().Set(...)` — the same store that produces `stateRoot`. `pluginWritableCorePrefixes` explicitly allows plugins to write into the `accounts` and `pools` core prefixes. `genesis.go` itself documents canoLiq's state living under store prefix `{20}`. So: **once the plugin's genesis bootstrap runs on a node, that node's `stateRoot` permanently diverges from any peer that hasn't also run it.** This is a real, one-shot, effectively irreversible consensus event — not a config toggle.

### 2. We validated the activation mechanism twice on a zero-stake node — clean, no bugs

Rather than test directly on the real validator, we deliberately activated the plugin on a disposable, zero-stake sync node (no consensus weight, no financial risk) to de-risk the real activation:

- First with our own custom-built image (`canoliq-committee29:latest`, built from this repo's `plugin/go/Dockerfile` post-fix).
- Then again with the **official** `canopynetwork/canopy-plugin-go:latest` image (matching what our real validator's `node29` container actually runs), to catch any image-specific quirks before touching the real validator.

Both times: clean bootstrap, zero `SafetyCheck` errors (real multisig signers, real bucket addresses, correct chainId — all pass), real `StateWrite` calls succeed. The node then gets `unequal block hash` and gets stuck retrying the same height forever — **expected**, since it's the only node with the plugin's state and has no stake to make its version canonical. No panics, no crash loops, no data corruption. The mechanism itself is solid.

Genesis file was independently verified byte-identical against the authoritative one in `chains/prod/29-canoliq-32/resources.yaml` at [commit `a89151e`](https://github.com/canopy-network/graduated-chains/commit/a89151ed3ef0beb3242ab090655f6f7cbc608e07) — not a guess, a real diff.

### 3. `val-a`'s prod deployment is missing the plugin-mode env vars entirely — separate from the Dockerfile bug we already fixed

`plugin/go/main.go` gates everything on `CANOPY_PLUGIN_MODE`:

```go
switch mode {
case "", "contract":
    log.Println("starting plugin in 'contract' mode (send tutorial)")   // default, no-op stub
case "canoliq":
    log.Println("starting plugin in 'canoliq' mode (liquid staking)")   // real logic, needs CANOLIQ_CONFIG
```

Canopy's own dev deployment (`404-canoliq-98803`) sets `CANOPY_PLUGIN_MODE=canoliq`, `CANOLIQ_CONFIG=/etc/canoliq/canoliq-config.json`, `CANOLIQ_RPC_ADDR=:8587`, and mounts a ConfigMap with the plugin's config+genesis at `/etc/canoliq`. **Their prod deployment (`29-canoliq-32`, `val-a`) sets none of the three env vars and has no such ConfigMap.** Result: `val-a`'s plugin has been running in the generic stub mode the whole time — not because of the Dockerfile bug we reported (that only explains why the files would be *missing if requested*), but because prod's own manifest never asks for canoliq mode in the first place.

Practical implication: our Dockerfile fix (merged, released as `plugin-go-v2026.266.22`, picked up by `val-a`'s `pluginAutoUpdate`) does **not** by itself risk activating real canoliq logic on `val-a` — the env vars are the actual gate, and prod's manifest doesn't set them. This lowers the urgency of the "uncoordinated auto-activation" risk we flagged to Canopy earlier in this session, though it's still correct to fix before anyone flips the switch deliberately.

## Current blockers, in order

1. **Activating on our own `node29` is the next real step and hasn't happened yet.** Everything above was deliberately tested on a disposable node first. `node29`'s config/genesis files and env-var wiring are prepared and proven to work via the identical test; only the actual go-ahead on the staked validator is pending.
2. **`validatorRegistry` is empty** in `genesis.mainnet.json`. Per its own comment: this routes the entire 15% validator-incentive slice to a synthetic, unspendable aggregator key. Needs seeding with real committee-29 validator addresses before/at activation.
3. **`tvlCapBps` is not explicitly set** in the mainnet genesis params — needs an explicit governance decision on the real cap rather than relying on whatever default applies.
4. **PR #36** (`supply_pool_desync` alert) is still open, unmerged.
5. **Coordination with Canopy Foundation** on: (a) confirming the three env vars above before flipping `val-a` to real canoliq mode, (b) an agreed activation height/time so `val-a` doesn't fork away from whatever state our validator establishes, (c) pull access to `ghcr.io/canopy-network/graduator-builds` so we're not permanently building a different artifact than what they run in prod.

## Suggestions for this repo

- **Add a startup guard for the silent-stub-mode gap.** Right now, `profile=mainnet` with `CANOPY_PLUGIN_MODE` unset (or `"contract"`) starts up cleanly with zero indication anything canoliq-specific is missing — as we found, this can persist unnoticed in a real production deployment. Consider: if `SafetyCheck` sees `profile=mainnet` but the running mode isn't `"canoliq"`, log a loud, impossible-to-miss warning (or refuse to start, mirroring the existing placeholder-address guards).
- **Document the three env vars as a first-class deployment requirement**, e.g. in `plugin/go/README.md` or `TUTORIAL.md`, explicitly calling out that a `profile=mainnet` deployment without `CANOPY_PLUGIN_MODE=canoliq` + `CANOLIQ_CONFIG` will silently run the tutorial stub with no error — exactly the gap that let `val-a` run stub-mode in prod without anyone noticing.
- **Seed `validatorRegistry`** in `genesis.mainnet.json` before activation — currently `[]`.
- **Set an explicit `tvlCapBps`** in `genesis.mainnet.json`'s params rather than leaving it to whatever default `DefaultParams()` supplies.
- **Get PR #36 merged** so the `supply_pool_desync` alert is live before real funds start moving through the buckets/treasury.

## Verification trail (for anyone re-checking this)

- Genesis byte-diff: `chains/prod/29-canoliq-32/resources.yaml` @ `a89151e` vs `plugin/go/canoliq/genesis.mainnet.json` and vs the state-derived `canoliq-genesis-REAL.json` — all semantically identical.
- `SafetyCheck` source read in full (`plugin/go/canoliq/config.go`) — confirmed every guard (profile, chainId, redemption window, genesis readability, placeholder addresses, placeholder multisig, uncapped TVL) passes cleanly with current `main`.
- `StateWrite` path read in full (`fsm/state.go`) — confirmed plugin writes reach the same store as core consensus.
- Two independent activation dry-runs on a zero-stake node (custom image, then official `canopynetwork/canopy-plugin-go:latest`) — both clean, both diverge as expected, neither crashes.
