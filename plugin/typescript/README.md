# Canopy Plugin Express.js Port

This is an Express.js port of the Go Canopy blockchain plugin, maintaining 100% functional compatibility with the original implementation.

## Overview

The plugin implements a "send" transaction functionality that communicates with the Canopy FSM (Finite State Machine) via Unix socket connections using length-prefixed protobuf messages.

## Project Structure

```
expressjs/
├── src/
│   ├── proto/           # Generated protobuf bindings
│   ├── socket-client.js # Unix socket communication with FSM
│   ├── config.js        # Configuration management
│   ├── proto-utils.js   # Protobuf utility functions
│   └── server.js        # Express.js server (coming in Phase 3)
├── proto/               # Protobuf definition files
├── test/                # Test files
├── package.json         # Node.js dependencies and scripts
└── TODO.md             # Implementation roadmap
```

## Dependencies

- **express**: Web framework for HTTP API endpoints
- **protobufjs**: Protocol buffer implementation for JavaScript
- **protobufjs-cli**: Command-line tools for protobuf code generation

## Scripts

- `npm start`: Start HTTP server with FSM integration
- `npm run start:plugin`: Start as Unix socket plugin only (matches Go behavior)
- `npm run dev`: Run server with nodemon for development
- `npm run demo`: Run API demonstration script
- `npm test`: Run comprehensive test suite (27 tests)
- `npm run build:proto`: Generate JavaScript protobuf bindings
- `npm run lint`: Run ESLint

## Phase 1 Complete ✅

The following infrastructure components have been implemented:

## Phase 2 Complete ✅

Contract logic and transaction processing have been fully implemented:

## Phase 3 Complete ✅

Express.js HTTP server with REST API endpoints has been implemented:

### HTTP Server Components

#### Express Server (`src/server.js`)

- ✅ Complete PluginServer class with HTTP API
- ✅ Blockchain lifecycle endpoints (genesis, begin-block, check-tx, deliver-tx, end-block)
- ✅ Transaction simulation endpoint (works without FSM)
- ✅ Information endpoints (health, status, info)
- ✅ Graceful startup and shutdown
- ✅ FSM integration with socket client

#### Middleware Stack (`src/middleware.js`)

- ✅ Request logging and validation
- ✅ Error handling with plugin error code mapping
- ✅ Response formatting with request correlation
- ✅ Timeout handling (10-second timeout matching Go)
- ✅ CORS support for development
- ✅ Health check integration

#### API Testing (`test/server.test.js`)

- ✅ Comprehensive test suite for all endpoints
- ✅ Transaction simulation validation
- ✅ Error handling verification
- ✅ Request/response format testing

#### Demo Script (`demo.js`)

- ✅ Interactive API demonstration
- ✅ Shows transaction simulation without FSM
- ✅ Error handling examples

### Contract Logic Components

#### Contract Class (`src/contract.js`)

- ✅ Complete Contract class matching Go struct behavior
- ✅ Genesis, BeginBlock, EndBlock lifecycle methods (empty like Go)
- ✅ CheckTx validation with fee parameter checking
- ✅ DeliverTx execution with state modifications
- ✅ CheckMessageSend with identical validation rules
- ✅ DeliverMessageSend with exact balance update logic

#### Error Handling (`src/errors.js`)

- ✅ PluginError class matching Go struct
- ✅ All 14 error types with identical codes and messages
- ✅ Error formatting and protobuf conversion

#### State Key Management (`src/keys.js`)

- ✅ Key prefixes: account[1], pool[2], params[7]
- ✅ KeyForAccount, KeyForFeePool, KeyForFeeParams functions
- ✅ Binary address and amount validation
- ✅ Length-prefixed key generation (JoinLenPrefix)

#### Main Entry Point (`src/main.js`)

- ✅ Plugin startup matching Go's main.go
- ✅ Graceful shutdown with signal handling
- ✅ Default configuration setup

### Core Infrastructure

- ✅ Node.js project structure with proper package.json
- ✅ Directory structure (src/, proto/, test/)
- ✅ Protobuf integration with code generation
- ✅ Unix socket client with length-prefixed messaging protocol

### Key Components

#### SocketClient (`src/socket-client.js`)

- Unix socket connection with retry logic (matches Go's ticker behavior)
- Length-prefixed protobuf message handling
- Async request/response correlation system
- 10-second timeout handling (matches Go timeout)
- FSM handshake process
- Message routing for Genesis, BeginBlock, CheckTx, DeliverTx, EndBlock

#### Configuration (`src/config.js`)

- Default configuration (ChainId=1, DataDir="/tmp/plugin/")
- JSON file loading and saving
- Matches Go's Config struct behavior

#### Protocol Buffer Utilities (`src/proto-utils.js`)

- Marshal/Unmarshal functions matching Go behavior
- FromAny type conversion for message polymorphism
- JoinLenPrefix for binary key generation
- FormatUint64 for big-endian number encoding

## Transaction Processing ✅

The Express.js implementation now provides **100% functional compatibility** with the Go version:

### Send Transaction Flow

1. **Validation (CheckTx)**: Fee validation, address format (20 bytes), amount > 0
2. **Execution (DeliverTx)**: Balance checks, state updates, fee collection
3. **State Management**: Batch reads/writes, account deletion optimization
4. **Error Handling**: Identical error codes and messages

### Key Features Implemented

- ✅ **Account Management**: 20-byte addresses, balance tracking
- ✅ **Fee Processing**: Parameter-driven fees, fee pool collection
- ✅ **Self-Transfer Optimization**: Same logic as Go version
- ✅ **Account Cleanup**: Zero-balance account deletion
- ✅ **Binary Key Generation**: Length-prefixed, identical format
- ✅ **Comprehensive Testing**: 14 passing tests validating core logic

## API Endpoints

### Information & Health

- `GET /health` - Health check with FSM connection status
- `GET /status` - Detailed server status and metrics
- `GET /info` - Plugin configuration and metadata

### Blockchain Lifecycle (requires FSM)

- `POST /genesis` - Process genesis block
- `POST /begin-block` - Handle block start
- `POST /check-tx` - Validate transaction
- `POST /deliver-tx` - Execute transaction
- `POST /end-block` - Handle block end

### Development & Testing

- `POST /simulate-tx` - Simulate transaction validation (no FSM required)

## Quick Start

```bash
# Install dependencies
npm install

# Run tests
npm test

# Start HTTP server (requires FSM running)
npm start

# Or start as Unix socket plugin only
npm run start:plugin

# Run API demonstration
npm run demo
```

## Architecture

This implementation maintains the exact same:

- **Message Flow**: FSM ↔ Unix Socket ↔ Plugin Logic
- **Protocol**: Length-prefixed protobuf messages
- **State Management**: Batch read/write operations
- **Error Handling**: Same error codes and messages
- **Transaction Processing**: Identical validation and execution logic

The main difference is using Express.js for HTTP endpoints and JavaScript async patterns instead of Go's channel-based concurrency.
