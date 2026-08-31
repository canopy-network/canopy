# AGENTS.md - Canopy TypeScript Plugin

This file provides context for AI coding assistants working on the Canopy TypeScript plugin.

## Project Overview

This is a **TypeScript plugin** for the Canopy blockchain that extends the Finite State Machine (FSM) functionality. The plugin communicates with the Canopy node via Unix socket using length-prefixed protobuf messages.

### Key Concepts

- **Plugin Architecture**: The plugin runs as a separate Node.js process and communicates with the Canopy FSM via a Unix socket (`plugin.sock`)
- **Protobuf Communication**: All messages between the plugin and FSM are protobuf-encoded with 4-byte length prefixes (big-endian)
- **Transaction Processing**: The plugin handles `CheckTx` (stateless validation) and `DeliverTx` (state application) for custom transaction types
- **State Access**: The plugin reads/writes blockchain state via `StateRead` and `StateWrite` RPC calls to the FSM

## Directory Structure

```
plugin/typescript/
├── src/
│   ├── main.ts                 # Entry point - starts the plugin
│   ├── contract/
│   │   ├── contract.ts         # Transaction handlers (Check/Deliver)
│   │   ├── plugin.ts           # Socket communication and FSM interface
│   │   ├── error.ts            # Error types and constructors
│   │   └── index.ts            # Re-exports
│   └── proto/
│       ├── types.ts            # Type re-exports from generated code
│       ├── descriptors.ts      # File descriptor protos for registration
│       ├── index.js            # Generated protobuf code (CommonJS)
│       ├── index.cjs           # Copy for ESM compatibility
│       └── index.d.ts          # Generated TypeScript definitions
├── proto/                      # Source .proto files
│   ├── tx.proto                # Transaction message definitions
│   ├── plugin.proto            # Plugin<->FSM communication messages
│   ├── account.proto           # Account state messages
│   └── event.proto             # Event messages
├── scripts/
│   └── generate-descriptors.cjs # Generates proto descriptors
├── tutorial/                   # Separate test project for new tx types
│   ├── src/rpc_test.ts         # RPC integration tests
│   ├── proto/                  # Proto files with test tx types
│   └── package.json
├── TUTORIAL.md                 # Guide for adding new transaction types
├── CUSTOMIZE.md                # General customization guide
└── package.json
```

## Key Files

### `src/contract/contract.ts`

Contains the core contract logic:

- **`ContractConfig`**: Registers supported transaction types with the FSM
  - `supportedTransactions`: Array of transaction names (e.g., `["send"]`)
  - `transactionTypeUrls`: Corresponding protobuf type URLs
  - Order must match between these arrays

- **`Contract` class**: Synchronous contract methods
  - `Genesis()`: Initial state setup
  - `BeginBlock()`: Called at block start
  - `EndBlock()`: Called at block end
  - `CheckMessage*()`: Stateless validation for each message type

- **`ContractAsync` class**: Async methods for state operations
  - `CheckTx()`: Validates transaction (reads state for fee check)
  - `DeliverTx()`: Applies transaction to state
  - `DeliverMessage*()`: State mutation for each message type

- **Key functions**: `KeyForAccount()`, `KeyForFeeParams()`, `KeyForFeePool()`
  - Generate state database keys with length-prefixed encoding

### `src/contract/plugin.ts`

Socket communication layer:

- **`Plugin` class**: Manages connection to FSM
  - `Handshake()`: Initial config exchange
  - `StateRead()`: Read state from FSM
  - `StateWrite()`: Write state to FSM
  - `ListenForInbound()`: Process incoming messages

- **`FromAny()`**: Decodes `google.protobuf.Any` to typed messages
  - Add new message type cases here when extending

- **Message protocol**: 4-byte length prefix (big-endian) + protobuf bytes

### `src/contract/error.ts`

Standardized error types:

- `IPluginError`: Interface with `code`, `module`, `msg`
- Error constructors: `ErrInvalidAddress()`, `ErrInsufficientFunds()`, etc.
- Module name: `"plugin"` for all errors

## Common Tasks

### Adding a New Transaction Type

See `TUTORIAL.md` for the complete guide. Summary:

1. Add message to `proto/tx.proto`
2. Run `npm run build:proto` and `npm run build:descriptors`
3. Add to `ContractConfig.supportedTransactions` and `transactionTypeUrls`
4. Add case in `FromAny()` function
5. Add `CheckMessage*()` method to `Contract` class
6. Add case in `ContractAsync.CheckTx()` switch
7. Add `DeliverMessage*()` method to `ContractAsync` class
8. Add case in `ContractAsync.DeliverTx()` switch

### Building the Plugin

Using Makefile (recommended):
```bash
make build-all       # Full rebuild (install + proto + descriptors + TypeScript)
make build           # TypeScript compilation only
make build-proto     # Regenerate protobuf code only
make build-descriptors  # Regenerate descriptor file only
```

Using npm directly:
```bash
npm run build:all    # Full rebuild (proto + descriptors + TypeScript)
npm run build:proto  # Regenerate protobuf code only
npm run build:descriptors  # Regenerate descriptor file only
npm run build        # TypeScript compilation only
```

### Running the Plugin

The plugin is started by Canopy when configured with `"plugin": "typescript"` in `~/.canopy/config.json`.

For development:
```bash
make dev             # Run with nodemon for hot reload
make run             # Run compiled output
# or
npm run dev          # Run with nodemon for hot reload
npm start            # Run compiled output
```

### Running with Docker

The TypeScript plugin can be run in a Docker container that includes both Canopy and the plugin.

#### Build the Docker Image

From the repository root:

```bash
make docker/plugin PLUGIN=typescript
```

This builds a Docker image named `canopy-typescript` that contains:
- The Canopy binary
- The TypeScript plugin (compiled with all proto descriptors)
- Node.js 20 runtime
- Pre-configured `config.json` with `"plugin": "typescript"`

#### Run the Container

```bash
make docker/run-typescript
```

Or manually with volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-typescript
```

#### Expose Ports for Testing

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-typescript
```

| Port | Service |
|------|---------|
| 50002 | RPC API (transactions, queries) |
| 50003 | Admin RPC (keystore operations) |

Now you can run tests from your host machine that connect to `localhost:50002`.

#### View Logs

```bash
# Get the container ID
docker ps

# View Canopy logs
docker exec -it <container_id> tail -f /root/.canopy/logs/log

# View plugin logs
docker exec -it <container_id> tail -f /tmp/plugin/typescript-plugin.log
```

### Running Tests

Run the integration tests from the plugin directory:

```bash
cd plugin/typescript
make test
```

`make test` runs both the transaction integration tests and the custom RPC endpoints test (`npm test && npm test -- custom` in the `tutorial/` subdirectory). Equivalent direct invocation:

```bash
cd tutorial
npm install
npm test             # transaction tests
npm test -- custom   # custom RPC endpoints test
```

Requires Canopy running with the plugin enabled, faucet/reward transactions implemented, and the plugin's RPC server reachable on port `50010` for the custom RPC test.

## Adding Custom RPC Endpoints

A plugin can expose its own read-only HTTP endpoints for chain-specific data (e.g. `/v1/query/faucets`, `/v1/query/rewards`). The plugin owns its HTTP server entirely — Canopy never needs to know about these routes. See `TUTORIAL.md` "Step 5b: Expose Custom RPC Endpoints" for the full walkthrough.

1. **Persist queryable records during `DeliverTx`** — write a small protobuf record to state under a plugin-owned key prefix (e.g. `Faucet` under `[100]` via `KeyForFaucet(addr)` in `src/contract/contract.ts`). Endpoints can only return data that lives in state.
2. **Declare the prefix** — add every custom record prefix to `ContractConfig.customStatePrefixes` (e.g. `customStatePrefixes: [faucetPrefix, rewardPrefix]`). Canopy validates this at handshake and **panics before the plugin starts** if a prefix collides with a core-reserved prefix (`1-15`). See "Key Prefixes" below.
3. **Register routes** — the base plugin already ships a skeleton `src/contract/rpc.ts` whose `StartRPCServer()` runs the node `http` server with **no routes registered by default**, and it is already started from `src/main.ts` with `StartRPCServer(plugin)`. Just add your routes+handlers (backed by `queryState`) to the existing `http.createServer` callback in `StartRPCServer()` (add as many as you want).
4. **Back each handler with `queryState`** — the detached, read-only query path on the `Plugin` class: `plugin.queryState(height, read)` returns raw key/value state at a historical height (`0` = latest committed). Use a single-key read (`KeyForFaucet(addr)`) for one record, or a range read over the prefix (`FaucetPrefix()`) to list all records. Decode the raw bytes into your own protobuf type and shape the JSON response however you like.
5. **Listen address** comes from the `rpcAddress` config field (default `0.0.0.0:50010`).

## Code Patterns

### State Keys

Keys are length-prefixed byte arrays:

```typescript
const accountPrefix = Buffer.from([1]);
const poolPrefix = Buffer.from([2]);
const paramsPrefix = Buffer.from([7]);

function KeyForAccount(addr: Uint8Array): Uint8Array {
    return JoinLenPrefix(accountPrefix, Buffer.from(addr));
}
```

#### Key Prefixes

- `[1]` - Account storage (shared with core; plugins may interoperate)
- `[2]` - Pool storage (shared with core; plugins may interoperate)
- `[7]` - Governance parameters
- `[100]` / `[101]` - Example plugin-owned custom records (faucet/reward) **added by the tutorial**, not part of the base plugin

> **Avoid prefix collisions:** the plugin shares Canopy's FSM keyspace. Canopy reserves the single-byte prefixes `1-15` for its own state (accounts, pools, validators, committees, params, …). Custom plugin records **must** use prefixes outside that range (e.g. `100`, `101`). Declare them in `ContractConfig.customStatePrefixes`; Canopy panics at handshake — before the plugin starts — if any declared prefix collides, and the core FSM additionally rejects any write to a reserved prefix.

### Reading State

```typescript
const [response, readErr] = await contract.plugin.StateRead(contract, {
    keys: [
        { queryId: Long.fromNumber(randomId), key: keyBytes },
    ],
});
// Response has response.results[].entries[].value
```

### Writing State

```typescript
const [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
    sets: [{ key: keyBytes, value: valueBytes }],
    deletes: [{ key: deleteKeyBytes }],  // optional
});
```

### Error Handling

Always return errors in response objects:

```typescript
if (error) {
    return { error: ErrInvalidAddress() };
}
return { recipient: addr, authorizedSigners: [signer] };
```

### Working with Long

Protobuf uint64 values may be `Long` or `number`:

```typescript
const amount = Long.isLong(msg.amount) 
    ? msg.amount 
    : Long.fromNumber(msg.amount as number || 0);
```

## Protobuf Notes

- Generated code uses CommonJS format (`index.js`)
- ESM compatibility via `index.cjs` copy
- Field names in TypeScript use camelCase (e.g., `fromAddress`)
- Field names in proto files use snake_case (e.g., `from_address`)
- `google.protobuf.Any` uses `type_url` (snake_case) for encoding

## Testing Notes

- Tests require a running Canopy node with the TypeScript plugin enabled
- Use `@noble/curves` for BLS12-381 signing in tests
- Transaction signatures use G2 signatures (longSignatures)
- DST: `BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_`

## Dependencies

**Runtime:**
- `long`: 64-bit integer support for protobuf
- `protobufjs`: Runtime protobuf encoding/decoding

**Development:**
- `protobufjs-cli`: Proto compilation (`pbjs`, `pbts`)
- `typescript`: Type checking and compilation
- `@types/node`: Node.js type definitions

**Tutorial only:**
- `@noble/curves`: BLS12-381 cryptography for test signing
- `tsx`: TypeScript execution for tests

## Configuration

The plugin reads config from `chain.json` in the data directory (default: `/tmp/plugin/`):

```typescript
interface Config {
    ChainId: number;      // Chain identifier
    DataDirPath: string;  // Path to plugin data directory
}
```

## Socket Protocol

1. **Connection**: Plugin connects to `{DataDirPath}/plugin.sock`
2. **Handshake**: Plugin sends config, FSM responds with FSM config
3. **Request/Response**: FSM sends requests, plugin responds
4. **Message Format**: `[4-byte length BE][protobuf bytes]`

Message types (`FSMToPlugin` / `PluginToFSM`):
- `config`: Configuration exchange (handshake)
- `genesis`: Genesis state import/export
- `begin`: BeginBlock
- `check`: CheckTx
- `deliver`: DeliverTx
- `end`: EndBlock
- `stateRead`: State read request/response
- `stateWrite`: State write request/response
