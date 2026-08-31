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

## Adding Custom RPC Endpoints

A plugin can expose its own read-only HTTP endpoints for chain-specific data (e.g. `/v1/query/faucets`, `/v1/query/rewards`). The plugin owns its HTTP server entirely — Canopy never needs to know about these routes. See `TUTORIAL.md` "Step 5b: Expose Custom RPC Endpoints" for the full walkthrough.

1. **Persist queryable records during `DeliverTx`** — write a small protobuf record to state under a plugin-owned key prefix (e.g. `Faucet` under `[]byte{100}` via `KeyForFaucet(addr)`). Endpoints can only return data that lives in state.
2. **Declare the prefix** — add every custom record prefix to `ContractConfig.CustomStatePrefixes`. Canopy validates this at handshake and **panics before the plugin starts** if a prefix collides with a core-reserved prefix (`1-15`). See "Key Prefixes" below.
3. **Register routes** in the skeleton `contract/rpc.go` `StartRPCServer()` on the shared `http.ServeMux` (add as many as you want). The skeleton server registers no routes by default and is already started from `main.go` with `go plugin.StartRPCServer()`.
4. **Back each handler with `QueryState`** — the detached, read-only query path: `p.QueryState(height, read)` returns raw key/value state at a historical height (`0` = latest committed). Use a single-key read (`KeyForFaucet(addr)`) for one record, or a range read over the prefix (`FaucetPrefix()`) to list all records. Decode the raw bytes into your own protobuf type and shape the JSON response however you like.
5. **Listen address** comes from the `rpcAddress` config field (default `0.0.0.0:50010`).

## State Management

### Key Prefixes
- `[]byte{1}` - Account storage (shared with core; plugins may interoperate)
- `[]byte{2}` - Pool storage (shared with core; plugins may interoperate)
- `[]byte{7}` - Governance parameters
- `[]byte{100}` / `[]byte{101}` - Example plugin-owned custom records (faucet/reward) **added by the tutorial**, not part of the base plugin

> **Avoid prefix collisions:** the plugin shares Canopy's FSM keyspace. Canopy reserves the single-byte prefixes `1-15` for its own state (accounts, pools, validators, committees, params, …). Custom plugin records **must** use prefixes outside that range (e.g. `100`, `101`). Declare them in `ContractConfig.CustomStatePrefixes`; Canopy panics at handshake — before the plugin starts — if any declared prefix collides, and the core FSM additionally rejects any write to a reserved prefix.

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
make build          # Builds to plugin/go/go-plugin
```

## Running with Docker

The Go plugin can be run in a Docker container that includes both Canopy and the plugin.

### Build the Docker Image

From the repository root:

```bash
make docker/plugin PLUGIN=go
```

This builds a Docker image named `canopy-go` that contains:
- The Canopy binary
- The Go plugin binary and control script
- Pre-configured `config.json` with `"plugin": "go"`

### Run the Container

```bash
make docker/run-go
```

Or manually with volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-go
```

### Expose Ports for Testing

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-go
```

| Port | Service |
|------|---------|
| 50002 | RPC API (transactions, queries) |
| 50003 | Admin RPC (keystore operations) |

Now you can run tests from your host machine that connect to `localhost:50002`.

### View Logs

```bash
# Get the container ID
docker ps

# View Canopy logs
docker exec -it <container_id> tail -f /root/.canopy/logs/log

# View plugin logs
docker exec -it <container_id> tail -f /tmp/plugin/go-plugin.log
```

## Running with Canopy

1. Add `"plugin": "go"` to `~/.canopy/config.json`
2. Start Canopy: `~/go/bin/canopy start`
3. Plugin auto-starts and connects via Unix socket

## Testing

```bash
cd plugin/go
make test
```

`make test` runs both the transaction integration tests (`TestPluginTransactions`) and the custom RPC endpoints test (`TestPluginCustomRPCEndpoints`). Equivalent direct invocation:

```bash
cd plugin/go/tutorial
go test -v -run 'TestPluginTransactions|TestPluginCustomRPCEndpoints' -timeout 600s
```

Requires Canopy running with the plugin enabled, faucet/reward transactions implemented, and the plugin's RPC server reachable on port `50010` for the custom RPC test.

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
