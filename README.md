🌿 Praxis — Prediction Market on Canopy
A fully on-chain YES/NO prediction market built as a Canopy appchain plugin.
Submitted for the Canopy Vibe Coding Contest 2026.
What is Praxis?
Praxis is a sovereign prediction market appchain built on the Canopy Network. Anyone can create a YES/NO question, stake tokens on an outcome, and earn a proportional payout from the losing pool when the market resolves.
Every action — creating a market, placing a bet, resolving an outcome, claiming winnings — is a real on-chain transaction processed by a custom Canopy plugin. No smart contracts. No shared blockspace. Full sovereignty.
How It Works
The Flow
Code
Payout Formula
Code
If you bet 100 PRX on YES and YES wins with a 60/40 split:
Your payout = 100 + (100 × 40/60) = ~167 PRX
Transaction Types
Transaction
Who Signs
What It Does
create_market
Creator
Opens a new YES/NO market with a question, resolver, and resolution block height
submit_prediction
Forecaster
Places a bet of X amount on YES (1) or NO (0)
resolve_market
Resolver
Declares the winning outcome after the resolution height
claim_winnings
Winner
Collects proportional payout from the losing pool
send
Sender
Standard token transfer (built-in)
State Schema
Prefix
Type
Description
0x01
Account
Token balances
0x02
Pool
Fee pool
0x07
FeeParams
Governance fee parameters
0x10
Market
All market data (question, pools, status, winner)
0x11
Prediction
Individual forecaster bets (keyed by marketId + address)
0x12
MarketCounter
Auto-incrementing market ID counter
0x13
ForecasterRecord
Leaderboard stats per forecaster
Files Changed
Code
Running Locally
Prerequisites
Go 1.25+
Git + Make
protoc + protoc-gen-go
Step 1 — Clone and build Canopy
Bash
Step 2 — Regenerate proto files
Bash
Step 3 — Build the plugin
Bash
Step 4 — Configure
Start the node once to generate config files:
Bash
Set the plugin in ~/.canopy/config.json:
Json
Step 5 — Start the chain
Bash
Watch for:
Code
Step 6 — Open the frontend
Bash
Open your browser at:
Code
The green dot in the sidebar confirms the chain is connected.
RPC Endpoints
Port
Purpose
50002
Public RPC — submit transactions, query state, check height
50003
Admin RPC — keystore access, key management (localhost only)
Key endpoints used by the frontend
Code
Signing Transactions
Canopy uses BLS12-381 signatures. Per the builder docs:
Code
Get your private key from the admin keystore:
Bash
Architecture
Code
Technical Notes
Why package-level height variable?
Canopy's plugin.go creates a new Contract instance for every FSM message. This means state set in BeginBlock() would be lost by the time DeliverTx() runs. We solve this with a package-level globalHeight variable protected by sync.RWMutex.
Why GOTOOLCHAIN=local?
The plugin's dependency github.com/drand/kyber requires Go 1.25. Setting GOTOOLCHAIN=local prevents Go from trying to auto-download a newer toolchain, which causes segfaults in constrained environments.
State key design
All state keys use JoinLenPrefix with unique byte prefixes to avoid collisions with built-in keys (0x01 accounts, 0x02 pools, 0x07 fee params). Our custom types start at 0x10.
About
Built by Makaveli912 for the Canopy Vibe Coding Contest 2026.
PR: https://github.com/canopy-network/canopy/pull/375
Branch: feat/praxis-prediction-market
Chain: Canopy Betanet
"Praxis — where prediction meets sovereignty."
