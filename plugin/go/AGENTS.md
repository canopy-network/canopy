# Praxis Plugin — AI Agent Context

This file provides context for AI coding assistants working on the Praxis
prediction-market plugin for the Canopy Network.

## What This Is

Praxis is an on-chain YES/NO prediction market protocol built as a Canopy
Nested Chain plugin. It is written in Go and implements the Canopy plugin
interface (the five lifecycle methods: Genesis, BeginBlock, CheckTx,
DeliverTx, EndBlock).

## Repository Layout

```
plugin/go/
├── main.go              # Entry point — calls contract.StartPlugin(). Do not modify.
├── chain.json           # Chain metadata (name, symbol, chainId, networkId)
├── Makefile             # Build targets — run `make build` before `canopy start`
├── pluginctl.sh         # Plugin lifecycle script (start/stop/restart/status)
├── crypto/              # BLS12-381 signing helpers (key loading, sign-bytes, address derivation)
│
├── contract/
│   ├── plugin.go        # Socket protocol, StartPlugin(), StateRead/StateWrite. Do not modify.
│   ├── contract.go      # ALL application logic lives here — this is the file to edit
│   └── error.go         # Plugin error codes. Built-in: 1–14. Praxis: 15–29.
│
└── proto/
    ├── tx.proto         # Transaction and state message definitions
    ├── account.proto    # Account and Pool types (built-in, do not modify)
    ├── plugin.proto     # FSM communication protocol (do not modify)
    ├── event.proto      # Event types (do not modify)
    └── _generate.sh     # Run this after editing tx.proto to regenerate Go structs
```

## Transaction Types

| Name                | Type URL                                          | Signer            |
|---------------------|---------------------------------------------------|-------------------|
| `send`              | `type.googleapis.com/types.MessageSend`           | from_address      |
| `create_market`     | `type.googleapis.com/types.MessageCreateMarket`   | creator_address   |
| `submit_prediction` | `type.googleapis.com/types.MessageSubmitPrediction`| forecaster_address|
| `resolve_market`    | `type.googleapis.com/types.MessageResolveMarket`  | resolver_address  |
| `claim_winnings`    | `type.googleapis.com/types.MessageClaimWinnings`  | claimer_address   |

## State Key Prefixes

| Prefix | Type          | Key function            |
|--------|---------------|-------------------------|
| 0x01   | Account       | KeyForAccount(addr)     |
| 0x02   | Pool          | KeyForFeePool(chainId)  |
| 0x07   | FeeParams     | KeyForFeeParams()       |
| 0x10   | Market        | KeyForMarket(id)        |
| 0x11   | MarketCounter | KeyForMarketCounter()   |
| 0x12   | Prediction    | KeyForPrediction(addr, marketId) |

**Important**: All keys are built with `JoinLenPrefix` which length-prefixes
each segment. This means `KeyForMarket(1)` starts with bytes `[0x01, 0x10, ...]`
not `[0x10, ...]`. When querying state by prefix, use `0110` (hex), not `10`.

## Critical Rules (from Canopy Plugin Spec)

1. `SupportedTransactions[i]` MUST match `TransactionTypeUrls[i]` exactly.
   A mismatch causes silent transaction misrouting.

2. `CheckTx` is stateless. It CANNOT write state. It may read state minimally
   (the fee params read is acceptable). Do not add state reads for business
   logic — those belong in `DeliverTx`.

3. Sign bytes = `proto.Marshal(txWithSignatureFieldNil)`. Never sign JSON.

4. If `DeliverTx` returns an error, the transaction is still included in the
   block and the fee is still charged. `CheckTx` must catch everything
   recoverable.

5. `PluginDeliverRequest` does not carry a block height. Capture it in
   `BeginBlock` via `c.currentHeight = req.Height`.

6. Always batch-read all required keys in a single `StateRead` call, then
   batch-write all changes in a single `StateWrite` call.

7. When building composite state keys with `append`, always allocate a new
   slice. `append(existingSlice, ...)` may mutate the original if it has
   spare capacity.

## Adding a New Transaction Type

1. Add the message definition to `proto/tx.proto`
2. Run `proto/_generate.sh` to regenerate Go structs
3. Add the name to `ContractConfig.SupportedTransactions` at index N
4. Add the type URL to `ContractConfig.TransactionTypeUrls` at index N (same index)
5. Add a `case *MessageXxx:` to the `CheckTx` switch and implement `CheckMessageXxx`
6. Add a `case *MessageXxx:` to the `DeliverTx` switch and implement `DeliverMessageXxx`
7. Add a `KeyForXxx` function using an unused byte prefix (> 0x12 for Praxis)
8. If you need a new error, add it to `error.go` starting at code 30+

## Prompts That Work Well

**Adding a transaction type:**
"Using the Canopy Go plugin pattern in contract.go, add a [tx_name] transaction.
The proto message fields are: [list fields]. Validate [conditions] in CheckTx.
In DeliverTx, read [keys], apply [logic], write [keys] back. Use prefix 0x13."

**State schema:**
"I need state keys for [type] in a Canopy plugin. Existing prefixes in use:
0x01–0x02, 0x07, 0x10–0x12. Design JoinLenPrefix-based keys that don't collide."

**Frontend encoding:**
"Write a JavaScript hand-encoder for [MessageName] with fields [list with types].
Field numbers must match tx.proto exactly. Use the varintField/bytesField/stringField
helpers already in the frontend."
