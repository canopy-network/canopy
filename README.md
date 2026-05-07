# PRAXIS

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
[![Network](https://img.shields.io/badge/Canopy-Betanet-00ff88)]()
[![Plugin](https://img.shields.io/badge/Plugin-Go-00d4ff)]()
[![Status](https://img.shields.io/badge/Status-Betanet-ffc940)]()

</div>

---

## Overview

**Praxis** is a sovereign prediction market protocol built as a Canopy Nested Chain. It combines the **ADLMSR** logarithmic market scoring rule with the **PORS** (Praxis Optimistic Resolution System), supporting 13 on-chain transaction types across the full market lifecycle — creation, prediction, resolution, and settlement.

The plugin runs as an application-specific blockchain with its own state, token ($PRX), and custom transaction logic.

[![Architecture](https://img.shields.io/badge/architecture-appchain-00ff88)]()
[![Consensus](https://img.shields.io/badge/consensus-NestBFT-00d4ff)]()
[![Signing](https://img.shields.io/badge/signing-BLS12--381-b48eff)]()
[![State](https://img.shields.io/badge/state-key--value-ffc940)]()

---

## Transaction Types

| # | Name | Description |
|---|------|-------------|
| 0 | `send` | Transfer $PRX between accounts |
| 1 | `create_market` | Open a YES/NO prediction market with LMSR liquidity |
| 2 | `submit_prediction` | Purchase shares on YES or NO using LMSR pricing |
| 4 | `claim_winnings` | Claim pro-rata payout from a finalised or cancelled/voided market |
| 5 | `register_resolver` | Stake $PRX to become a registered resolver |
| 6 | `propose_outcome` | Propose the winning outcome after a market expires |
| 7 | `file_dispute` | Challenge a proposed outcome with a bond |
| 8 | `commit_vote` | Panel members submit a blinded vote |
| 9 | `reveal_vote` | Panel members reveal their vote |
| 10 | `tally_votes` | Tally revealed votes and determine dispute outcome |
| 11 | `finalize_market` | Finalise a market — caller receives a 50 PRX bounty |
| 12 | `claim_slash` | Claim the slashed bond of a losing disputer |

---

## Architecture — ADLMSR + PORS

The core LMSR market-maker is augmented by the Praxis Optimistic Resolution System (PORS):

1. A market expires at its configured block height
2. A registered **resolver** proposes the winning outcome
3. A **dispute window** opens — any other resolver can challenge by posting a bond
4. A **panel** of independent resolvers is randomly selected and runs a commit-reveal vote
5. Tallying slashes the losing party and rewards the winner
6. Any account can call `finalize_market` to close the market and earn a **50 PRX bounty**

Proto file-descriptor registration is handled by `z_descriptor.go`, which exposes all proto definitions to the Canopy node during the handshake.

---

## State Model

Praxis state is stored in the Canopy KV store with byte-prefixed keys:

| Prefix | Type | Description |
|--------|------|-------------|
| `0x10` | `MarketState` | Per-market record (status, LMSR quantities, creator, expiry) |
| `0x11` | `PositionState` | Per-bettor position (shares YES/NO, cost paid) |
| `0x12` | `OutcomeState` | Winning outcome and resolution height |
| `0x13` | `ResolverState` | Per-market assigned resolver |
| `0x14` | `TreasuryReserve` | Locked PRX for bounties and bonds |
| `0x16` | `ResolverRecord` | Global resolver profile (stake, RRS score) |
| `0x17` | `ProposalRecord` | Outcome proposal for a market |
| `0x18` | `DisputeRecord` | Dispute details, panel, vote status |
| `0x19` | `VoteCommit` | Blinded vote commitment per panel member |
| `0x1A` | `VoteReveal` | Revealed vote per panel member |
| `0x1B` | `SlashRecord` | Record of a slashed resolver |
| `0x1C` | `PanelEntropyAccum` | Rolling entropy accumulator for panel selection |

> Built-in prefixes `0x01` (Account), `0x02` (Pool), and `0x07` (FeeParams) are preserved.

---

## ADLMSR Parameters

| Constant | Value | Meaning |
|----------|-------|---------|
| `PRECISION_SCALE` | 1,000,000 | Fixed-point scaling for LMSR quantities |
| `MIN_B0` | 1,000,000 | Minimum initial liquidity (1 PRX) |
| `ELEVATED_RISK_THRESHOLD` | 25,000,000,000 | Pool size at which markets become elevated-risk |
| `RESOLUTION_DELAY_BLOCKS` | 100 | Blocks after expiry before resolution can begin |
| `GRACE_PERIOD_BLOCKS` | 200 | Resolution window |
| `CLAIM_GRACE_PERIOD` | 1,000 | Time window to claim winnings |

**LMSR cost function:**

```
C(qYes, qNo) = bEff · ln( exp(qYes/bEff) + exp(qNo/bEff) )
```

**Payout:** overflow-safe pro-rata via `quot·winnerShares + rem·winnerShares / totalWinShares`

### Payout Example

```
YES pool: 600,000 μPRX  (bettor contributed 200,000)
NO pool:  400,000 μPRX  (losing side)

Bettor share of YES pool : 200,000 / 600,000 = 33.3%
Bettor payout            : 200,000 + (400,000 × 33.3%) = 333,333 μPRX
```

---

## Repository Layout

```
plugin/go/
├── main.go
├── chain.json
├── Makefile
├── pluginctl.sh
├── AGENTS.md
├── README.md
├── go.mod / go.sum
│
├── contract/
│   ├── contract.go              ← ContractConfig + lifecycle methods
│   ├── constants.go             ← All named constants
│   ├── error.go                 ← Praxis error codes 100–199
│   ├── helpers.go               ← mulDiv, DeriveMarketId, ComputeCommitHash
│   ├── keys.go                  ← All state key constructors
│   ├── lmsr.go                  ← LMSR cost, trade, payout, min bond
│   ├── height.go                ← Global height with RWMutex
│   ├── plugin.go                ← Socket protocol (never modify)
│   ├── z_descriptor.go          ← Proto file descriptor registration
│   ├── handler_*.go (17 files)  ← CheckTx + DeliverTx for all tx types
│   ├── tx.pb.go                 ← Generated (never edit)
│   ├── event.pb.go              ← Generated
│   ├── account.pb.go            ← Generated
│   └── plugin.pb.go             ← Generated
│
├── crypto/
│   ├── bls.go                   ← BLS12-381 signing (kyber/bdn)
│   └── signing.go               ← GetSignBytes helper
│
├── proto/
│   ├── tx.proto                 ← Message & state definitions
│   ├── account.proto
│   ├── plugin.proto
│   ├── event.proto
│   └── _generate.sh
│
└── tutorial/
    ├── main.go
    ├── go.mod / go.sum
    ├── praxis_test.go           ← Integration test for send + create_market
    └── contract/                ← Tutorial-local generated copies

frontend/
└── index.html                   ← Single-file HTML/JS dashboard
```

---

## Getting Started

### Build

```bash
git clone https://github.com/Makaveli912/canopy.git
cd canopy
git checkout feat/praxis-prediction-markets

# Build Canopy node
go build -o ~/go/bin/canopy ./cmd/main

# Build Praxis plugin
cd plugin/go
GOTOOLCHAIN=local go build -o go-plugin .
```

### Run

```bash
cd ~/canopy
canopy start
```

Watch for:

```
Plugin go started: go-plugin started successfully
Plugin service listening on socket: /tmp/plugin/plugin.sock
```

> **Note:** The node takes ~12–15 seconds to boot before RPC is available on port `50002`.

### Test

```bash
cd plugin/go/tutorial
GOTOOLCHAIN=local go test -v -run TestCreateMarket -timeout 120s
```

---

## Token

| Property | Value |
|----------|-------|
| Name | Praxis |
| Symbol | $PRX |
| Denomination | μPRX (micro-PRX) |
| Chain ID | 1 |
| Network ID | 1 |

---

## Error Codes

| Range | Description |
|-------|-------------|
| 1–14 | Standard Canopy built-in errors |
| 100–199 | Praxis-specific errors (market, position, resolution, dispute, etc.) |

See [`contract/error.go`](plugin/go/contract/error.go) for the full list.

---

## License

MIT — see [LICENSE](LICENSE)

