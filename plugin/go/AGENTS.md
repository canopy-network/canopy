# Agent Instructions for Go Plugin

This document provides context for AI agents working with the Canopy Go plugin codebase.

## Overview

This is a **Go plugin for the Canopy blockchain** that communicates with the Canopy FSM (Finite State Machine) via Unix sockets. The plugin implements custom transaction types and state management for a nested blockchain.

## Architecture

### Communication Pattern
- **Protocol**: Length-prefixed protobuf messages over Unix socket (`/tmp/plugin/plugin.sock`)
- **Flow**: FSM ↔ Plugin via `FSMToPlugin` and `PluginToFSM` protobuf messages
- **Lifecycle**: Plugin connects → Handshake → Receives tx requests → Responds with results

### Key Components

| Directory | Purpose |
|-----------|---------|
| `contract/` | Core contract logic, protobuf generated code, plugin communication |
| `crypto/` | BLS12-381 signing utilities |
| `proto/` | Protobuf definitions (`.proto` files) |
| `tutorial/` | Test project with pre-built faucet/reward transaction examples |

### Important Files

- `contract/contract.go` - Main contract logic with `CheckTx` and `DeliverTx` handlers
- `contract/plugin.go` - Socket communication and plugin lifecycle
- `contract/*.pb.go` - Generated protobuf code (do not edit manually)
- `proto/tx.proto` - Transaction message definitions
- `main.go` - Entry point

## Transaction Flow

1. **CheckTx**: Stateless validation (fee check, address validation, returns authorized signers)
2. **DeliverTx**: Stateful execution (reads state, applies changes, writes state)

## Adding New Transaction Types

Follow the pattern in `TUTORIAL.md`:

1. Add message to `proto/tx.proto`
2. Run `proto/_generate.sh` to regenerate Go code
3. Register in `ContractConfig.SupportedTransactions` and `TransactionTypeUrls`
4. Add `case` in `CheckTx` switch → implement `CheckMessage<Type>`
5. Add `case` in `DeliverTx` switch → implement `DeliverMessage<Type>`

## State Management

### Key Prefixes
- `[]byte{1}` - Account storage
- `[]byte{2}` - Pool storage  
- `[]byte{7}` - Governance parameters

### State Operations
```go
// Read state
c.plugin.StateRead(c, &PluginStateReadRequest{Keys: []*PluginKeyRead{...}})

// Write state
c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: [...], Deletes: [...]})
```

## Cryptography

- **Signature scheme**: BLS12-381 (not Ed25519)
- **Address derivation**: First 20 bytes of SHA256(publicKey)
- **Sign bytes**: Deterministic protobuf marshaling of Transaction (without signature field)

## Building

```bash
cd plugin/go
make build          # Builds to ~/go/bin/go-plugin
```

## Running with Canopy

1. Add `"plugin": "go"` to `~/.canopy/config.json`
2. Start Canopy: `~/go/bin/canopy start`
3. Plugin auto-starts and connects via Unix socket

## Testing

```bash
cd plugin/go/tutorial
go test -v -run TestPluginTransactions -timeout 120s
```

Requires Canopy running with the plugin enabled and faucet/reward transactions implemented.

## Code Conventions

- **Error handling**: Return `*PluginError` structs, use error functions from `error.go`
- **Protobuf**: Use `Marshal`/`Unmarshal` helpers from `contract/plugin.go`
- **Logging**: Use `log.Printf` for debugging (logs to `/tmp/plugin/go-plugin.log`)
- **QueryIds**: Use `rand.Uint64()` to correlate batch state read requests

## Common Patterns

### CheckTx Template
```go
func (c *Contract) CheckMessage<Type>(msg *Message<Type>) *PluginCheckResponse {
    // Validate addresses (must be 20 bytes)
    if len(msg.Address) != 20 {
        return &PluginCheckResponse{Error: ErrInvalidAddress()}
    }
    // Validate amount
    if msg.Amount == 0 {
        return &PluginCheckResponse{Error: ErrInvalidAmount()}
    }
    // Return authorized signers
    return &PluginCheckResponse{
        Recipient:         msg.RecipientAddress,
        AuthorizedSigners: [][]byte{msg.SignerAddress},
    }
}
```

### DeliverTx Template
```go
func (c *Contract) DeliverMessage<Type>(msg *Message<Type>, fee uint64) *PluginDeliverResponse {
    // 1. Generate query IDs
    queryId := rand.Uint64()
    
    // 2. Read state
    response, err := c.plugin.StateRead(c, &PluginStateReadRequest{...})
    
    // 3. Unmarshal accounts
    account := new(Account)
    Unmarshal(bytes, account)
    
    // 4. Apply business logic
    account.Amount += msg.Amount
    
    // 5. Marshal and write state
    bytes, _ = Marshal(account)
    c.plugin.StateWrite(c, &PluginStateWriteRequest{...})
    
    return &PluginDeliverResponse{}
}
```

## Debugging

- **Plugin logs**: `tail -f /tmp/plugin/go-plugin.log`
- **Canopy logs**: `tail -f ~/.canopy/logs/log`
- **Common errors**:
  - `"message name X is unknown"` → Transaction not registered in `ContractConfig`
  - `"invalid signature"` → Sign bytes mismatch, check protobuf serialization
  - Balance not updating → Check `DeliverTx` is being called, wait for block finalization

## Dependencies

Key external packages:
- `google.golang.org/protobuf` - Protobuf serialization
- `github.com/drand/kyber` + `github.com/drand/kyber-bls12381` - BLS signing
