# Canopy Wallet Starter v3 (Config-First, Wizard + Payload)

- EVM address validation (0x optional) via `viem`
- 15-minute session-unlock (RAM-only)
- Fees from POST /v1/query/params (`selector: fee.sendFee`)
- Extended manifest:
  - Field `rules`, `help`, `placeholder`, `prefix/suffix`, `colSpan`, `tab`
  - `form.layout.grid` + optional `aside`
  - `confirm.showPayload` to reveal raw payload
  - `steps[]` for wizard flows

## Run
pnpm i
pnpm dev
# open http://localhost:5173/?action=Send
# or http://localhost:5173/?action=Stake
