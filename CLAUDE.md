# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

### Building
```bash
make build/canopy           # Build just the canopy binary
make build/canopy-full      # Build canopy + wallet + explorer
make build/wallet           # Build wallet frontend (Next.js)
make build/explorer         # Build explorer frontend (Next.js)
```

### Testing
```bash
make test/all              # Run all Go tests
make test/fuzz             # Run fuzz tests individually
go test ./fsm -run TestSpecificFunction  # Run specific test
```

### Development
```bash
make dev/deps              # Install dependencies (go mod vendor)
make docker/up             # Run containerized localnet
make docker/logs           # View container logs
```

## Architecture Overview

Canopy is a blockchain implementation built on a recursive architecture with the following core modules:

### Controller (`controller/`)
Central coordinator that manages communication between all blockchain components. Acts as the system bus connecting FSM, BFT consensus, P2P networking, and storage.

### Finite State Machine (`fsm/`)
Core state transition logic that processes transactions and maintains blockchain state. Key components:
- **State Management**: Handles accounts, validators, governance, swaps
- **Message Processing**: Routes transaction types (send, stake, create order, etc.)
- **Ethereum Compatibility**: Translates Ethereum RLP transactions to native message types via selector pattern
- **Contract System**: Uses pseudo-contract addresses for built-in functionality (CNPY token, staking, trading)

### BFT Consensus (`bft/`)
Byzantine Fault Tolerant consensus mechanism for block agreement. Handles voting, proposals, and evidence collection.

### P2P Network (`p2p/`)
Encrypted peer-to-peer communication layer for node discovery and message propagation.

### Storage (`store/`)
Persistent data layer with transaction support, indexing, and Sparse Merkle Tree implementation for state verification.

### Library (`lib/`)
Shared utilities including cryptography (BLS, secp256k1, Ed25519), blockchain primitives, and common interfaces.

## Key Implementation Details

### Transaction Processing Flow
1. Transactions enter through Controller
2. FSM processes via `ApplyTransaction()` method
3. Message routing in `HandleMessage()` based on type
4. State changes persisted through store interface

### Protocol Buffers
Extensive use of protobuf for serialization across all modules. Generated files have `.pb.go` suffix.

### Multi-Language Components
- **Go**: Core blockchain logic

## Common Development Patterns

### Error Handling
Uses custom error interface (`lib.ErrorI`) throughout the codebase rather than standard Go errors.

### State Management
All state changes go through `StateMachine.Set()` and `StateMachine.Get()` with key-value store backing.

### Message Types
New transaction types require:
1. Protobuf definition
2. Message handler in FSM
3. Routing entry in `HandleMessage()`

### RPC and CLI Development

#### RPC Server Endpoints
- **Admin endpoints**: `cmd/rpc/admin.go` - Add new HTTP endpoints here
- **Query endpoints**: `cmd/rpc/query.go` - Add query-related HTTP endpoints here

#### RPC Client Commands
- **Client code**: `cmd/rpc/client.go` - When adding HTTP endpoints, add corresponding client commands here

#### CLI Commands
- **CLI directory**: `cmd/cli/` - Add new CLI commands in files within this directory: query.go and admin.go
- When adding a CLI command it will usually have RPC client and RPC server endpoints. You must update the RPC client and RPC server when implementing new CLI commands.

### Testing
Each module has comprehensive test coverage. Use `_test.go` files and follow existing patterns for new tests.

### Plans & Tasks
- **Plan directory**: `.claude/plans/` - Project planning, feature plans
- **Tasks directory**: `.claude/tasks/` - Tasks
- **Current Task**: `.claude/tasks/current/` - The current task is in this directory

### Coding Style
#### Comments
- Each line of code should have a comment explaining what it does, even if it is simple
- Each function should have a detailed comment. Multi-line if required.
- All configuration structs should have a comment explaining that fields use

#### Formatting
- **Spacing**: There should be no blank link between lines of code.
