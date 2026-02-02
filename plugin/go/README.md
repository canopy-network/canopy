# Send Transaction Flow Analysis

## Overview

This document analyzes the complete flow of a send transaction in the Canopy blockchain plugin template. The plugin implements a Unix socket-based communication system between smart contracts and the Canopy FSM (Finite State Machine).

## Architecture Components

### Key Files
- `main.go`: Entry point starting the plugin with graceful shutdown
- `contract/plugin.go`: Socket communication and plugin lifecycle management
- `contract/contract.go`: Core contract logic and transaction processing
- `contract/error.go`: Plugin-specific error definitions
- `proto/*.proto`: Protocol buffer definitions for communication

## Complete Transaction Flow

### 1. Plugin Initialization (main.go:11-18)

```
main() → StartPlugin(DefaultConfig()) → Socket connection to FSM
```

The plugin connects to the Canopy FSM via Unix socket (`plugin.sock`):
- Attempts connection every second until successful
- Creates Plugin instance with configuration
- Starts background listener for FSM messages
- Performs handshake with FSM

### 2. Socket Communication Setup (plugin.go:34-68)

```
StartPlugin() → net.Dial("unix", sockPath) → ListenForInbound() → Handshake()
```

- **Socket Path**: `/tmp/plugin/plugin.sock` (default data directory)
- **Protocol**: Length-prefixed protobuf messages
- **Handshake**: Exchanges plugin configuration with FSM
- **Concurrent Handling**: Each message processed in separate goroutine

### 3. Transaction Reception (plugin.go:122-167)

```
FSM → Unix Socket → ListenForInbound() → Route by message type
```

When FSM sends a transaction:
1. Plugin receives length-prefixed protobuf message
2. Creates new Contract instance with FSM context
3. Routes message based on type (`FSMToPlugin_Check` or `FSMToPlugin_Deliver`)
4. Processes request concurrently in goroutine

### 4. Transaction Validation - CheckTx (contract.go:38-72)

```
CheckTx() → Validate Fee → Parse Message → CheckMessageSend()
```

**Fee Validation**:
- Reads fee parameters from state: `KeyForFeeParams()` (contract.go:42)
- Verifies `tx.fee >= minFees.SendFee` (contract.go:57)
- Returns error if fee too low: `ErrTxFeeBelowStateLimit()`

**Message Parsing**:
- Deserializes `tx.msg` from protobuf Any type (contract.go:61)
- Type-switches to handle `MessageSend` (contract.go:66-68)

**Send Message Validation** (contract.go:96-111):
- **From Address**: Must be exactly 20 bytes (contract.go:98)
- **To Address**: Must be exactly 20 bytes (contract.go:102) 
- **Amount**: Must be greater than 0 (contract.go:106)
- Returns authorized signers: `[][]byte{msg.FromAddress}`

### 5. Transaction Execution - DeliverTx (contract.go:74-88)

```
DeliverTx() → Parse Message → DeliverMessageSend()
```

Routes to `DeliverMessageSend()` with fee parameter for state modifications.

### 6. Send Transaction Processing (contract.go:114-212)

**State Reading** (contract.go:124-147):
```
StateRead() → FSM via Socket → Returns account balances and fee pool
```

Batch read operation for:
- **Fee Pool**: `KeyForFeePool(chainId)` with prefix `[]byte{2}`
- **From Account**: `KeyForAccount(fromAddress)` with prefix `[]byte{1}` 
- **To Account**: `KeyForAccount(toAddress)` with prefix `[]byte{1}`

**Balance Validation** (contract.go:149-164):
- Calculates total deduction: `amount + fee`
- Checks sender balance: `from.Amount >= amountToDeduct`
- Returns `ErrInsufficientFunds()` if insufficient

**Balance Updates** (contract.go:166-187):
- **Self-transfer optimization**: Uses same account object if `fromKey == toKey`
- **Sender**: `from.Amount -= (msg.Amount + fee)`
- **Recipient**: `to.Amount += msg.Amount`  
- **Fee Pool**: `feePool.Amount += fee`

**State Writing** (contract.go:189-211):
```
StateWrite() → FSM via Socket → Commits state changes
```

Two write patterns:
- **Account Deletion**: If sender balance reaches 0, delete sender account (contract.go:191-198)
- **Normal Update**: Update all three entities (contract.go:199-207)

### 7. State Key Structure

**Account Storage**:
- Prefix: `[]byte{1}`
- Key: `JoinLenPrefix(accountPrefix, address)`
- 20-byte addresses only

**Fee Pool Storage**:
- Prefix: `[]byte{2}` 
- Key: `JoinLenPrefix(poolPrefix, formatUint64(chainId))`

**Fee Parameters**:
- Prefix: `[]byte{7}`
- Key: `JoinLenPrefix(paramsPrefix, []byte("/f/"))`

### 8. Socket Communication Protocol (plugin.go:239-292)

**Message Format**:
```
[4-byte length prefix][protobuf message bytes]
```

**Request/Response Pattern**:
- **Sync Requests**: Plugin waits for FSM response with 10-second timeout
- **Async Handling**: FSM requests processed concurrently
- **Request Correlation**: Unique request IDs track pending operations
- **Error Handling**: Timeout cleanup and error propagation

### 9. Error Handling

**Validation Errors** (error.go):
- `ErrInvalidAddress()` - Code 12: Non-20-byte addresses
- `ErrInvalidAmount()` - Code 13: Zero amounts  
- `ErrTxFeeBelowStateLimit()` - Code 14: Insufficient fees
- `ErrInsufficientFunds()` - Code 9: Balance too low

**System Errors**:
- Socket communication failures
- Protobuf serialization errors
- State read/write timeouts
- Plugin response correlation errors

## Transaction Lifecycle Summary

1. **Plugin Startup**: Connect to FSM via Unix socket
2. **Transaction Receipt**: FSM sends transaction via socket
3. **Validation Phase**: CheckTx validates fee, addresses, and amount
4. **Execution Phase**: DeliverTx reads state, validates balance, updates accounts
5. **State Persistence**: Write updated balances and fee pool to FSM
6. **Response**: Return success/error to FSM via socket

## Key Features

- **Concurrent Processing**: Multiple transactions handled simultaneously
- **State Management**: Efficient batch reads/writes with FSM
- **Error Recovery**: Comprehensive error handling and timeouts
- **Account Lifecycle**: Automatic cleanup of zero-balance accounts
- **Fee Collection**: Transparent fee pooling for network sustainability

## Performance Characteristics

- **Socket Communication**: Low-latency Unix domain sockets
- **Batch Operations**: Multiple state operations in single FSM call
- **Memory Efficiency**: Length-prefixed messaging avoids buffering issues
- **Concurrent Safety**: Thread-safe request correlation and state management