# PRAXIS
[![Status](https://img.shields.io/badge/status-live-brightgreen)](https://github.com/Makaveli912/canopy)


<div align="center">

```
██████╗ ██████╗  █████╗ ██╗  ██╗██╗███████╗
██╔══██╗██╔══██╗██╔══██╗╚██╗██╔╝██║██╔════╝
██████╔╝██████╔╝███████║ ╚███╔╝ ██║███████╗
██╔═══╝ ██╔══██╗██╔══██║ ██╔██╗ ██║╚════██║
██║     ██║  ██║██║  ██║██╔╝ ██╗██║███████║
╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚══════╝
```

**On-Chain Prediction Markets on the Canopy Network**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev)
[![Canopy](https://img.shields.io/badge/Canopy-Betanet-00ff88)](https://canopynetwork.org)
[![Plugin](https://img.shields.io/badge/Plugin-Go-00d4ff)](plugin/go)
[![Status](https://img.shields.io/badge/Status-Betanet-ffc940)](https://canopynetwork.org)

</div>

---

## ▶ Praxis is a sovereign prediction market protocol built as a Canopy Nested Chain

Praxis ($PRX) lets anyone create YES/NO prediction markets, stake on outcomes, and claim proportional winnings — entirely on-chain, with no platform extraction and no central authority. It is implemented as a Go plugin on the Canopy Network, meaning it runs as an application-specific blockchain with its own state, its own token, and its own transaction types.

[![architecture](https://img.shields.io/badge/architecture-appchain-00ff88)]()
[![consensus](https://img.shields.io/badge/consensus-NestBFT-00d4ff)]()
[![signing](https://img.shields.io/badge/signing-BLS12--381-b48eff)]()
[![state](https://img.shields.io/badge/state-key--value-ffc940)]()

---

## Overview

Praxis implements four on-chain transaction types:

| Transaction | Description |
|---|---|
| `create_market` | Open a new YES/NO prediction market with a question, resolver, and resolution height |
| `submit_prediction` | Stake tokens on a YES or NO outcome for an open market |
| `resolve_market` | The designated resolver finalises a market with the winning outcome |
| `claim_winnings` | Winners claim their proportional payout from the resolved market pool |

All state is stored on-chain in the plugin's key-value store. No database, no backend, no off-chain oracle required for basic operation.

---

## Architecture

Praxis follows the standard Canopy plugin architecture. The plugin runs as a separate process alongside the Canopy node and communicates over a Unix socket using Protocol Buffers.

```
┌─────────────────────────────────────────┐
│           CANOPY NODE PROCESS           │
│                                         │
│  ┌──────────┐  ┌────────────────────┐   │
│  │ NestBFT  │  │   FSM / Controller │   │
│  │ Consensus│  │  (block lifecycle) │   │
│  └──────────┘  └────────┬───────────┘   │
│                         │ Unix socket   │
└─────────────────────────┼───────────────┘
                          │
          ┌───────────────▼──────────────┐
          │        PRAXIS PLUGIN         │
          │                              │
          │  Genesis()                   │
          │  BeginBlock()                │
          │  CheckTx()   ← validate      │
          │  DeliverTx() ← execute       │
          │  EndBlock()                  │
          │                              │
          │  Transactions:               │
          │  - create_market             │
          │  - submit_prediction         │
          │  - resolve_market            │
          │  - claim_winnings            │
          └──────────────────────────────┘
                          ▲
                          │ HTTP RPC :50002 / :50003
                          │
          ┌───────────────┴──────────────┐
          │     PRAXIS FRONTEND          │
          │  Single-file HTML/JS         │
          │  BLS12-381 signing           │
          │  Hand-encoded protobuf       │
          └──────────────────────────────┘
```

---

## Repository Layout

```
plugin/go/
├── main.go                  # Entry point — calls contract.StartPlugin()
├── chain.json               # Chain metadata: name, symbol, chainId, networkId
├── Makefile                 # Build targets
├── pluginctl.sh             # Plugin lifecycle (start/stop/restart/status)
├── AGENTS.md                # AI assistant context for this plugin
│
├── contract/
│   ├── contract.go          # Application logic — all transaction handlers
│   ├── error.go             # Error codes (built-in 1–14, Praxis 15–16)
│   ├── plugin.go            # Socket protocol — do not modify
│   └── tx.pb.go             # Generated Go structs from tx.proto
│
└── proto/
    ├── tx.proto             # Transaction and state message definitions
    ├── account.proto        # Account and Pool types
    ├── plugin.proto         # FSM communication protocol
    └── _generate.sh         # Regenerates Go structs from .proto files

frontend/
└── index.html               # Single-file frontend — no build step required
```

---

## State Model

Praxis stores all on-chain data in the Canopy key-value store using byte-prefixed keys:

| Prefix | Type | Description |
|---|---|---|
| `0x10` | `Market` | One record per prediction market |
| `0x11` | `MarketCounter` | Singleton — tracks the next market ID |
| `0x12` | `Prediction` | One record per (forecaster, market) pair |

Built-in Canopy prefixes (`0x01` Account, `0x02` Pool, `0x07` FeeParams) are preserved unchanged.

---

## Transaction Types

### create_market

Opens a new YES/NO prediction market. The creator bonds a stake amount and designates a resolver address. The market remains open for predictions until the resolution height is reached.

```protobuf
message MessageCreateMarket {
  bytes  creator_address   = 1;
  string question          = 2;
  string description       = 3;
  bytes  resolver_address  = 4;
  uint64 resolution_height = 5;
  uint64 stake_amount      = 6;
}
```

### submit_prediction

Stakes tokens on a YES (outcome=1) or NO (outcome=2) outcome. Each forecaster may only submit one prediction per market. The staked amount is added to the corresponding pool.

```protobuf
message MessageSubmitPrediction {
  bytes  forecaster_address = 1;
  uint64 market_id          = 2;
  uint32 outcome            = 3;
  uint64 amount             = 4;
}
```

### resolve_market

Finalises the market. Only the designated resolver address may call this. Sets the winning outcome and closes the market to further predictions.

```protobuf
message MessageResolveMarket {
  bytes  resolver_address = 1;
  uint64 market_id        = 2;
  uint32 winning_outcome  = 3;
}
```

### claim_winnings

Pays out a winner's original stake plus their proportional share of the losing pool. Each prediction can only be claimed once.

```protobuf
message MessageClaimWinnings {
  bytes  claimer_address = 1;
  uint64 market_id       = 2;
}
```

Payout formula:
```
payout = stake + (stake × losing_pool) / winning_pool
```

---

## Getting Started

### Prerequisites

- Go 1.24 or later
- `protoc` and `protoc-gen-go` (for proto regeneration only)
- A running Canopy node

See the [Canopy Builder Docs](https://canopynetwork.org) for full prerequisites.

### Build

```bash
# Clone and switch to the Praxis branch
git clone https://github.com/Makaveli912/canopy.git
cd canopy
git checkout feat/praxis-prediction-markets

# Build the Canopy node binary
go build -o ~/go/bin/canopy ./cmd/main

# Build the Praxis plugin binary
cd plugin/go
go build -o go-plugin .
```

### Run

```bash
# From repo root
canopy start
```

Watch for:
```
Plugin go started: go-plugin started successfully
Plugin service listening on socket: /tmp/plugin/plugin.sock
```

### Frontend

```bash
python3 -m http.server 8080 --directory frontend
```

Open `http://localhost:8080`. Go to **Node** → set host to `localhost` → Apply. The green dot confirms connection.

Go to **Signer** → paste your BLS12-381 private key → Load Key. Your address will be auto-derived and filled into all transaction forms.

---

## Payout Model

Praxis uses an AMM-style proportional payout. Winners split the entire losing pool in proportion to their contribution to the winning pool.

```
Example:
  YES pool: 600,000 μPRX (forecaster contributed 200,000)
  NO pool:  400,000 μPRX (losing side)

  Forecaster share of YES pool: 200,000 / 600,000 = 33.3%
  Forecaster payout: 200,000 + (400,000 × 33.3%) = 333,333 μPRX
```

If no one bet on the losing side, the winner's original stake is returned unchanged.

---

## Error Codes

| Code | Name | Description |
|---|---|---|
| 1–14 | Built-in | Standard Canopy plugin errors |
| 15 | `ErrWrongOutcome` | Claimer's prediction did not match the winning outcome |
| 16 | `ErrDuplicatePrediction` | Forecaster already submitted a prediction for this market |

---

## Token

| Property | Value |
|---|---|
| Name | Praxis |
| Symbol | $PRX |
| Denomination | μPRX (micro-PRX) |
| Chain ID | 1 |
| Network ID | 1 |

---

## License

MIT — see [LICENSE](LICENSE)
