# AGENTS.md - AI Agent Guidelines for Canopy Kotlin Plugin

This document provides guidance for AI agents working with the Canopy Kotlin Plugin codebase.

## Project Overview

This is a **Kotlin-based blockchain plugin** for the Canopy network. It communicates with the Canopy FSM (Finite State Machine) via Unix domain sockets using protobuf messages. The plugin handles transaction validation and execution for a token transfer system.

### Key Technologies
- **Language**: Kotlin 2.1+ (JVM 21)
- **Build System**: Gradle 8.x with Kotlin DSL
- **Serialization**: Protocol Buffers (protobuf-java + protobuf-kotlin)
- **Networking**: Unix domain sockets via junixsocket
- **Logging**: kotlin-logging (SLF4J wrapper)
- **Cryptography**: BLS12-381 signatures (for client-side signing in tutorial)

## Project Structure

```
plugin/kotlin/
├── src/main/kotlin/com/canopy/plugin/
│   ├── Main.kt           # Entry point, starts PluginClient
│   ├── Config.kt         # Configuration data class
│   ├── Contract.kt       # Transaction logic (CheckTx, DeliverTx)
│   ├── PluginClient.kt   # Unix socket communication with FSM
│   └── Error.kt          # Error definitions matching Go codes
├── src/main/proto/       # Protobuf definitions
│   ├── account.proto     # Account/Pool state structures
│   ├── event.proto       # Event definitions
│   ├── plugin.proto      # FSM<->Plugin communication protocol
│   └── tx.proto          # Transaction message types
├── tutorial/             # Separate test project (see below)
├── TUTORIAL.md           # How to add new transaction types
└── build.gradle.kts      # Build configuration
```

## Core Architecture

### Plugin Lifecycle

1. **Startup**: `Main.kt` creates `PluginClient` and calls `start()`
2. **Connection**: Plugin connects to Unix socket at `/tmp/plugin/plugin.sock`
3. **Handshake**: Plugin sends `ContractConfig` to FSM, receives `FSMConfig`
4. **Message Loop**: Plugin listens for FSM requests and handles them

### Message Flow

```
FSM (Go) <--Unix Socket--> PluginClient <--> Contract
                              |
                              v
                        State Read/Write (via FSM)
```

### Key Classes

| Class | Purpose |
|-------|---------|
| `PluginClient` | Handles socket communication, message routing |
| `Contract` | Transaction validation and execution logic |
| `ContractConfig` | Plugin configuration (name, supported tx types) |
| `Config` | Runtime configuration (chainId, dataDirPath) |

## Transaction Processing

Each transaction goes through two phases:

### 1. CheckTx (Validation)
- Called for mempool validation
- Statelessly validates transaction structure
- Returns `authorizedSigners` (who must sign) and `recipient`
- Should NOT modify state

### 2. DeliverTx (Execution)
- Called when transaction is included in a block
- Reads current state, validates, applies changes
- Writes new state back to FSM

### Adding New Transaction Types

See `TUTORIAL.md` for complete instructions. Summary:

1. Add message to `tx.proto`
2. Regenerate protos: `./gradlew generateProto`
3. Register in `ContractConfig.SUPPORTED_TRANSACTIONS` and `TRANSACTION_TYPE_URLS`
4. Add `checkMessage*` function for validation
5. Add `deliverMessage*` function for execution
6. Update `fromAny()` to parse the new message type
7. Add cases in `checkTx()` and `deliverTx()` when statements

## State Management

### Key Prefixes
State is stored in a key-value store. Keys are constructed using length-prefixed byte arrays:

```kotlin
private val ACCOUNT_PREFIX = byteArrayOf(1)   // Account balances
private val POOL_PREFIX = byteArrayOf(2)       // Fee pool
private val PARAMS_PREFIX = byteArrayOf(7)     // Parameters (fee params)
```

### State Read/Write Pattern

```kotlin
// 1. Build read request with query IDs
val readRequest = PluginStateReadRequest.newBuilder()
    .addKeys(PluginKeyRead.newBuilder()
        .setQueryId(Random.nextLong())
        .setKey(ByteString.copyFrom(keyBytes))
        .build())
    .build()

// 2. Send to FSM
val readResponse = plugin.stateRead(this, readRequest)

// 3. Parse response by query ID
for (result in readResponse.resultsList) {
    when (result.queryId) {
        myQueryId -> bytes = result.getEntries(0).value.toByteArray()
    }
}

// 4. Parse protobuf
val account = Account.parseFrom(bytes)

// 5. Modify and write back
val writeRequest = PluginStateWriteRequest.newBuilder()
    .addSets(PluginSetOp.newBuilder()
        .setKey(ByteString.copyFrom(keyBytes))
        .setValue(ByteString.copyFrom(newAccount.toByteArray()))
        .build())
    .build()
plugin.stateWrite(this, writeRequest)
```

## Error Handling

Errors are defined in `Error.kt` with codes matching the Go implementation:

```kotlin
fun ErrInsufficientFunds() = PluginError(9, "plugin", "insufficient funds")
fun ErrInvalidAddress() = PluginError(12, "plugin", "address is invalid")
```

Return errors via protobuf:
```kotlin
return PluginCheckResponse.newBuilder()
    .setError(ErrInvalidAddress().toProto())
    .build()
```

## Tutorial Project

The `tutorial/` subdirectory is a separate Gradle project for testing custom transaction types (faucet, reward). It includes:

- **BLS.kt**: BLS12-381 signing with correct DST for Canopy
- **RpcTest.kt**: Integration tests that submit transactions via HTTP RPC
- Full `tx.proto` with all message types

Run tutorial tests:
```bash
cd tutorial
make test-rpc  # Requires running Canopy node
```

## BLS12-381 Signing (Tutorial)

Canopy uses BLS signatures with a specific Domain Separation Tag (DST):

```kotlin
private const val DST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
```

This matches the drand/kyber BDN scheme. The jblst library provides low-level access:

```kotlin
fun sign(secretKey: SecretKey, message: ByteArray): ByteArray {
    val p2 = P2()
    p2.hash_to(message, DST)
    p2.sign_with(secretKey)
    return p2.compress()
}
```

**Critical**: Using the wrong DST will produce valid BLS signatures that Canopy rejects.

## Common Tasks

### Build
```bash
make build
# or
./gradlew build -x test
```

### Run Plugin
```bash
make run
# or
./gradlew run
```

### Generate Protos
```bash
make proto
# or
./gradlew generateProto
```

### Run Tests
```bash
make test
```

## Running with Docker

The Kotlin plugin can be run in a Docker container that includes both Canopy and the plugin.

### Build the Docker Image

From the repository root:

```bash
make docker/plugin PLUGIN=kotlin
```

This builds a Docker image named `canopy-kotlin` that contains:
- The Canopy binary
- The Kotlin plugin fat JAR
- JRE 21 runtime
- Pre-configured `config.json` with `"plugin": "kotlin"`

### Run the Container

```bash
make docker/run-kotlin
```

Or manually with volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-kotlin
```

### Expose Ports for Testing

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-kotlin
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
docker exec -it <container_id> tail -f /tmp/plugin/kotlin-plugin.log
```

## Important Conventions

### Address Format
- Addresses are 20 bytes (derived from SHA256 of public key, first 20 bytes)
- Always validate: `if (msg.address.size() != 20)`

### Amount/Fee
- All amounts are in micro-denomination (uCNPY)
- 1 token = 1,000,000 uCNPY
- Fees are deducted from sender, added to fee pool

### Transaction Type URLs
Must be in format: `type.googleapis.com/types.MessageName`

### Order Matters
`SUPPORTED_TRANSACTIONS` order must match `TRANSACTION_TYPE_URLS` order.

## Debugging Tips

1. **Socket connection issues**: Check if Canopy is running with `plugin: "kotlin"` in config
2. **Proto issues**: Run `./gradlew clean generateProto`
3. **Message not recognized**: Verify type URL in `ContractConfig` and `fromAny()`
4. **Invalid signature**: Check DST matches `"BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"`
5. **State not updating**: Ensure `stateWrite` is called and check for errors

## Files to Modify for Common Changes

| Change | Files |
|--------|-------|
| Add transaction type | `tx.proto`, `Contract.kt` (config, checkTx, deliverTx, fromAny) |
| Add state key type | `Contract.kt` (add key function) |
| Add error code | `Error.kt` |
| Change plugin config | `Contract.kt` (`ContractConfig` object) |
| Modify socket behavior | `PluginClient.kt` |

## Do NOT

- Modify proto files without regenerating (`./gradlew generateProto`)
- Use different DST for BLS signing (must match Canopy server)
- Forget to update `fromAny()` when adding message types
- Skip CheckTx validation (mempool will reject invalid txs)
- Modify state in CheckTx (only DeliverTx should write state)
- Use 0 for queryId (use `Random.nextLong()`)

## Reference

- **Go Plugin**: `plugin/go/` - Reference implementation
- **Go Tutorial**: `plugin/go/TUTORIAL.md` - Parallel documentation
- **Protobuf Definitions**: `lib/.proto/` - Canonical proto files
- **Canopy FSM**: `fsm/` - State machine that calls plugins
