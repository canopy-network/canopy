# AGENTS.md - Python Plugin for Canopy Blockchain

This document provides context for AI agents working with the Canopy Python plugin codebase.

## Overview

This is a **Python implementation of a Canopy blockchain plugin** that extends the Finite State Machine (FSM) with custom transaction logic. The plugin communicates with the Canopy node via Unix socket using length-prefixed protobuf messages.

## Project Structure

```
plugin/python/
├── contract/                 # Core plugin logic
│   ├── __init__.py          # Package exports (start_plugin, default_config, etc.)
│   ├── contract.py          # Transaction handlers (CheckTx, DeliverTx)
│   ├── error.py             # Error types matching Go implementation
│   ├── plugin.py            # Socket communication with FSM
│   └── proto/               # Protobuf definitions and generated code
│       ├── __init__.py      # Proto type exports
│       ├── account.proto    # Account and Pool state types
│       ├── event.proto      # Event types
│       ├── plugin.proto     # FSM <-> Plugin communication protocol
│       ├── tx.proto         # Transaction and message types
│       └── *_pb2.py         # Generated Python protobuf code
├── tests/                   # Unit tests (pytest)
├── tutorial/                # Tutorial project for custom transactions
│   ├── proto/               # Proto files with faucet/reward messages
│   ├── rpc_test.py          # RPC integration test
│   └── requirements.txt     # Tutorial dependencies
├── main.py                  # Entry point - starts the plugin
├── Makefile                 # Build, test, proto generation commands
├── pyproject.toml           # Python project configuration
├── CUSTOMIZE.md             # Quick guide for adding transactions
└── TUTORIAL.md              # Step-by-step tutorial for new tx types
```

## Key Files

### `contract/contract.py`
The main contract implementation. Contains:
- `CONTRACT_CONFIG` - Plugin registration (name, supported transactions, type URLs)
- `Contract` class with lifecycle methods:
  - `genesis()` - Initialize state at height 0
  - `begin_block()` - Called at start of each block
  - `check_tx()` - Stateless transaction validation (async)
  - `deliver_tx()` - Execute transaction and modify state (async)
  - `end_block()` - Called at end of each block
- Transaction handlers: `_check_message_*` and `_deliver_message_*`
- State key generation: `key_for_account()`, `key_for_fee_pool()`, `key_for_fee_params()`

### `contract/plugin.py`
Socket communication layer:
- `Plugin` class - Manages Unix socket connection to FSM
- `Config` / `default_config()` - Plugin configuration
- `state_read()` / `state_write()` - Async state operations
- Length-prefixed protobuf message encoding (4-byte big-endian prefix)

### `contract/error.py`
Error types matching Go implementation:
- `PluginError` class with code, module, and message
- Factory functions: `err_insufficient_funds()`, `err_invalid_address()`, etc.

### `contract/proto/tx.proto`
Transaction message definitions. Currently supports:
- `Transaction` - Main transaction wrapper (must match `lib.Transaction` for signing)
- `MessageSend` - Standard token transfer
- `Signature` - BLS12-381 signature
- `FeeParams` - Fee configuration

## Architecture Patterns

### State Management
- State is stored in a key-value database on the FSM side
- Plugin reads/writes state via `state_read()` and `state_write()` async methods
- Keys are generated with length-prefixed byte arrays (see `join_len_prefix()`)
- State prefixes: `0x01` = accounts, `0x02` = pools, `0x07` = params

### Transaction Flow
1. FSM sends `check` request → Plugin validates statelessly, returns authorized signers
2. FSM sends `deliver` request → Plugin reads state, applies changes, writes state
3. All state changes are atomic within a transaction

### Async Design
- Plugin uses `asyncio` for concurrent request handling
- `check_tx()` and `deliver_tx()` are async methods
- Multiple transactions can be processed concurrently

## Common Tasks

### Adding a New Transaction Type

1. **Define protobuf message** in `contract/proto/tx.proto`:
   ```protobuf
   message MessageMyTx {
     bytes sender_address = 1;
     uint64 amount = 2;
   }
   ```

2. **Regenerate protos**: `make proto`

3. **Register in CONTRACT_CONFIG** (`contract/contract.py`):
   ```python
   "supported_transactions": ["send", "mytx"],
   "transaction_type_urls": [
       "type.googleapis.com/types.MessageSend",
       "type.googleapis.com/types.MessageMyTx",
   ],
   ```

4. **Add check handler**:
   ```python
   def _check_message_mytx(self, msg: MessageMyTx) -> PluginCheckResponse:
       # Validate addresses, amounts
       # Return authorized_signers
   ```

5. **Add deliver handler**:
   ```python
   async def _deliver_message_mytx(self, msg: MessageMyTx, fee: int) -> PluginDeliverResponse:
       # Read state, apply changes, write state
   ```

6. **Wire up in check_tx/deliver_tx switch statements**

### Running Tests

```bash
# Unit tests
make test

# With coverage
make test-cov

# Tutorial integration test (requires running Canopy node)
cd tutorial && make test
```

### Running with Docker

The Python plugin can be run in a Docker container that includes both Canopy and the plugin.

#### Build the Docker Image

From the repository root:

```bash
make docker/plugin PLUGIN=python
```

This builds a Docker image named `canopy-python` that contains:
- The Canopy binary
- The Python plugin with virtual environment
- Python 3.12 runtime
- Pre-configured `config.json` with `"plugin": "python"`

#### Run the Container

```bash
make docker/run-python
```

Or manually with volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-python
```

#### Expose Ports for Testing

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-python
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
docker exec -it <container_id> tail -f /tmp/plugin/python-plugin.log
```

### Regenerating Protobuf Code

```bash
make proto
```

This runs `grpc_tools.protoc` and fixes relative imports in generated files.

## Dependencies

Core:
- `protobuf>=4.21.0,<5.0.0` - Protocol buffer serialization
- `fastapi>=0.104.0` - HTTP server (not used in socket mode)
- `uvicorn>=0.24.0` - ASGI server
- `pydantic>=2.5.0` - Data validation
- `structlog>=23.2.0` - Structured logging

For signing (tutorial):
- `blspy>=2.0.0` - BLS12-381 signatures (use `BasicSchemeMPL`)

## Important Conventions

### Proto Field Naming
- Proto uses `snake_case` (e.g., `from_address`)
- JSON uses `camelCase` via `@gotags` annotations (e.g., `fromAddress`)
- Python generated code uses `snake_case`

### Error Handling
- Use `PluginError` with appropriate error code
- Error codes must match Go implementation for compatibility
- Return errors via response objects, don't raise in handlers

### State Keys
- Always use `join_len_prefix()` for consistent key generation
- Keys are length-prefixed byte arrays for safe concatenation
- Format uint64 as big-endian bytes with `format_uint64()`

### Transaction Signing
- BLS12-381 with G2 signatures (96 bytes)
- Use `BasicSchemeMPL` from `blspy` (not Aug or Pop schemes)
- Sign protobuf-encoded Transaction bytes (without signature field)
- See `tutorial/rpc_test.py` for complete signing implementation

## Communication Protocol

Plugin connects to FSM via Unix socket at `{data_dir_path}/plugin.sock`.

Message format:
```
[4 bytes: length (big-endian)] [N bytes: protobuf message]
```

Request/Response types defined in `plugin.proto`:
- `FSMToPlugin` - Messages from FSM to plugin
- `PluginToFSM` - Messages from plugin to FSM

Handshake sequence:
1. Plugin sends `PluginConfig` with supported transactions
2. FSM responds with `PluginFSMConfig`
3. Plugin ready to receive block/tx requests

## Debugging

- Plugin logs to stdout with `logging` module
- Check Canopy logs at `~/.canopy/logs/`
- Plugin-specific logs at `/tmp/plugin/python-plugin.log` (if configured)
- Use `logger.debug()` for transaction-level logging

## Related Files

- `plugin/go/` - Go plugin implementation (reference)
- `lib/codec.go` - Canopy's transaction encoding (must match for signing)
- `fsm/` - Finite State Machine that invokes plugins
