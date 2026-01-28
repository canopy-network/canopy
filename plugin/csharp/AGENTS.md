# AGENTS.md - C# Plugin for Canopy

This document provides guidance for AI agents working with the Canopy C# plugin codebase.

## Project Overview

This is a **C# plugin for the Canopy blockchain** that implements smart contract logic for a nested chain. The plugin communicates with the Canopy node via Unix sockets using Protocol Buffers for message serialization.

### Key Concepts

- **Plugin Architecture**: The plugin runs as a separate process and communicates with Canopy via Unix socket IPC
- **Transaction Types**: The base plugin supports "send" transactions; additional types (faucet, reward) are demonstrated in the tutorial
- **BLS12-381 Signatures**: Canopy uses BLS signatures with a custom DST (`BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_`) compatible with drand/kyber
- **Protobuf**: All messages are serialized using Protocol Buffers

## Project Structure

```
plugin/csharp/
├── src/CanopyPlugin/
│   ├── contract.cs          # Main contract logic (CheckTx, DeliverTx)
│   └── plugin.cs            # Plugin server and socket communication
├── proto/                    # Protobuf definitions
│   ├── tx.proto             # Transaction messages (MessageSend, etc.)
│   ├── plugin.proto         # Plugin protocol messages
│   ├── account.proto        # Account state
│   └── event.proto          # Event types
├── tutorial/                 # Tutorial project with extended features
│   ├── Crypto/BLSCrypto.cs  # BLS signing utilities
│   ├── RpcTest.cs           # Integration tests
│   └── CanopyPlugin.Tutorial.csproj
├── CanopyPlugin.csproj      # Main project file
├── CanopyPlugin.sln         # Solution file
├── Makefile                 # Build and test commands
├── TUTORIAL.md              # Guide for adding new transaction types
└── global.json              # .NET SDK version (8.0)
```

## Key Files

### `src/CanopyPlugin/contract.cs`
The main contract implementation containing:
- `ContractConfig` - Plugin configuration (name, ID, supported transactions, type URLs)
- `Contract` class with:
  - `CheckTxAsync()` - Stateless transaction validation
  - `DeliverTxAsync()` - Transaction execution with state changes
  - `Genesis()`, `BeginBlock()`, `EndBlock()` - Lifecycle hooks
- State key generation methods (`KeyForAccount`, `KeyForFeePool`, etc.)
- Error factory methods

### `src/CanopyPlugin/plugin.cs`
Plugin server that:
- Listens on Unix socket at `/tmp/plugin/{plugin_name}.sock`
- Handles the FSM (Finite State Machine) protocol with Canopy
- Manages state read/write operations

### `proto/tx.proto`
Defines transaction message types:
- `Transaction` - Wrapper with signature, fee, timestamps
- `MessageSend` - Transfer tokens between addresses
- `MessageReward` - Mint tokens (admin pays fee)
- `MessageFaucet` - Mint tokens (for testing, no balance check)

## Build Commands

```bash
# Restore dependencies
make restore

# Build the plugin
make build

# Run the plugin (starts socket server)
make run

# Run tutorial tests (requires Canopy node running)
make test-tutorial

# Clean build artifacts
make clean

# Format code
make format

# Lint code
make lint
```

## Running with Docker

The C# plugin can be run in a Docker container that includes both Canopy and the plugin.

### Build the Docker Image

From the repository root:

```bash
make docker/plugin PLUGIN=csharp
```

This builds a Docker image named `canopy-csharp` that contains:
- The Canopy binary
- The C# plugin DLL
- .NET 8 runtime
- Pre-configured `config.json` with `"plugin": "csharp"`

### Run the Container

```bash
make docker/run-csharp
```

Or manually with volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-csharp
```

### Expose Ports for Testing

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-csharp
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
docker exec -it <container_id> tail -f /tmp/plugin/csharp-plugin.log
```

## Development Guidelines

### Adding New Transaction Types

See `TUTORIAL.md` for the complete guide. Summary:

1. **Define protobuf message** in `proto/tx.proto`
2. **Register in ContractConfig**:
   - Add to `SupportedTransactions` array
   - Add type URL to `TransactionTypeUrls` array (same order!)
3. **Add CheckTx handler** - Stateless validation
4. **Add DeliverTx handler** - State mutation logic
5. **Rebuild** with `make build`

### Transaction Handler Pattern

```csharp
// CheckTx - Stateless validation
private PluginCheckResponse CheckMessageXxx(MessageXxx msg)
{
    // Validate addresses (must be 20 bytes)
    if (msg.SomeAddress.Length != 20)
        return new PluginCheckResponse { Error = ErrInvalidAddress() };
    
    // Validate amounts
    if (msg.Amount == 0)
        return new PluginCheckResponse { Error = ErrInvalidAmount() };
    
    // Return authorized signers
    return new PluginCheckResponse
    {
        Recipient = msg.RecipientAddress,
        AuthorizedSigners = { msg.SignerAddress }
    };
}

// DeliverTx - State mutation
private async Task<PluginDeliverResponse> DeliverMessageXxxAsync(MessageXxx msg, ulong fee)
{
    // 1. Read current state
    var response = await Plugin.StateReadAsync(this, new PluginStateReadRequest { ... });
    
    // 2. Check for errors
    if (response.Error != null)
        return new PluginDeliverResponse { Error = response.Error };
    
    // 3. Unmarshal accounts
    var account = new Account();
    account.MergeFrom(bytes);
    
    // 4. Apply state changes
    account.Amount += msg.Amount;
    
    // 5. Write state
    var writeResp = await Plugin.StateWriteAsync(this, new PluginStateWriteRequest { ... });
    return new PluginDeliverResponse { Error = writeResp.Error };
}
```

### State Keys

State keys are constructed using length-prefixed encoding:

```csharp
// Account key: 0x01 prefix + address
public static byte[] KeyForAccount(byte[] addr) => JoinLenPrefix(AccountPrefix, addr);

// Fee pool key: 0x02 prefix + chain ID
public static byte[] KeyForFeePool(ulong chainId) => JoinLenPrefix(PoolPrefix, FormatUInt64(chainId));

// Fee params key: 0x07 prefix + "/f/"
public static byte[] KeyForFeeParams() => JoinLenPrefix(ParamsPrefix, Encoding.UTF8.GetBytes("/f/"));
```

### BLS Signatures

The plugin uses BLS12-381 with a custom DST for compatibility with Canopy:

```csharp
// DST must match exactly
private const string DST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_";

// Sign bytes = protobuf-encoded Transaction WITHOUT signature field
var signBytes = tx.ToByteArray();  // signature field must be null/empty
var signature = BLSCrypto.Sign(secretKey, signBytes);
```

### Error Handling

Use the error factory methods in `Contract` class:

```csharp
ErrInvalidAddress()      // Code 12
ErrInvalidAmount()       // Code 13
ErrInsufficientFunds()   // Code 9
ErrInvalidMessageCast()  // Code 11
ErrTxFeeBelowStateLimit() // Code 14
```

## Testing

### Prerequisites
1. Canopy node running with C# plugin enabled (`"plugin": "csharp"` in config)
2. Plugin socket available at `/tmp/plugin/csharp-plugin.sock`

### Running Tests
```bash
cd plugin/csharp
make test-tutorial
```

### Test Structure
The tutorial tests (`tutorial/RpcTest.cs`) demonstrate:
1. Creating accounts in keystore
2. Sending faucet transaction (mint tokens)
3. Sending transfer transaction
4. Sending reward transaction
5. Verifying balances

## RPC Endpoints

Tests communicate with Canopy via HTTP RPC:

| Endpoint | Port | Purpose |
|----------|------|---------|
| Query RPC | 50002 | `height`, `account`, `txs-by-sender`, `failed-txs`, `tx` submission |
| Admin RPC | 50003 | `keystore-new-key`, `keystore-get` |

## Common Issues

### "message name xxx is unknown"
- Check `ContractConfig.SupportedTransactions` includes the message name
- Check `ContractConfig.TransactionTypeUrls` includes the type URL
- Verify order matches between the two arrays

### Invalid signature errors
- Ensure signing protobuf bytes, not JSON
- Verify Transaction structure matches (all fields in correct order)
- Check DST matches exactly

### Plugin not connecting
- Check socket path `/tmp/plugin/csharp-plugin.sock`
- Verify plugin is running (`make run`)
- Check Canopy config has `"plugin": "csharp"`

### Build errors with test files
- The main `.csproj` excludes `test/` and `tutorial/` directories
- If new test folders are added, update the exclusion in `CanopyPlugin.csproj`

## Dependencies

- **.NET 8.0** - Target framework
- **Google.Protobuf** - Protocol Buffers runtime
- **Grpc.Tools** - Protobuf code generation
- **Nethermind.Crypto.Bls** - BLS12-381 native library (blst)

## Configuration

### `global.json`
```json
{
  "sdk": {
    "version": "8.0.100",
    "rollForward": "latestFeature"
  }
}
```

### Plugin Config in Canopy
Edit `~/.canopy/config.json`:
```json
{
  "plugin": "csharp",
  ...
}
```

## Useful Commands

```bash
# Check .NET SDK version
dotnet --version

# List available tests
dotnet test tutorial/CanopyPlugin.Tutorial.csproj --list-tests

# Run specific test
dotnet test --filter "TestPluginTransactions"

# Verbose test output
dotnet test --logger "console;verbosity=detailed"

# Check plugin logs
tail -f /tmp/plugin/csharp-plugin.log
```
