# tx.go - Transaction Processing and Memory Pool Management

This file implements the logic for transaction sending, handling, and memory pooling in the Canopy
blockchain. It's a critical component that manages how transactions flow through the network and are
stored before being included in blocks.

## Overview

The transaction handling system is designed to:

- Process locally generated transactions
- Listen for and validate incoming transactions from the network
- Maintain a memory pool (mempool) of valid transactions
- Gossip valid transactions to peers
- Track failed transactions for reporting
- Prioritize transactions based on fees

## Core Components

### Controller

The Controller manages the overall transaction flow in the blockchain. It:

- Sends locally generated transactions to the network
- Listens for incoming transactions from peers
- Validates transactions before adding them to the mempool
- Manages peer reputation based on transaction validity
- Gossips valid transactions to other peers
- Prevents duplicate transaction processing

### Mempool

The Mempool is a temporary storage area for valid but unconfirmed transactions. It:

- Maintains an ordered list of transactions (typically by fee)
- Validates transactions against the current blockchain state
- Evicts invalid transactions when state changes
- Prioritizes transactions with higher fees
- Handles special transaction types (like certificate results)
- Maintains a txn of transaction results for efficient verification
- Tracks failed transactions for reporting purposes

### Transaction Validation

The validation system ensures only valid transactions enter the mempool:

- Checks if transactions already exist in the blockchain
- Verifies transactions against the current state
- Uses an ephemeral copy of the state machine to validate without affecting the main chain
- Rechecks all mempool transactions when necessary
- Evicts transactions that become invalid due to state changes

## Processes

```mermaid
flowchart TD
    A[Local Transaction] -->|SendTxMsg| B[Controller]
    C[Network Transaction] -->|ListenForTx| B
    B -->|HandleTransaction| D{Already in blockchain?}
    D -->|Yes| E[Reject as Duplicate]
    D -->|No| F{Already in mempool?}
    F -->|Yes| G[Reject as Already Pending]
    F -->|No| H[Mempool]
    H -->|HandleTransaction| I{Valid against state?}
    I -->|No| J[Cache as Failed]
    I -->|Yes| K[Add to Mempool]
    K -->|If needed| L[Recheck Mempool]
    L -->|For each tx| M{Still valid?}
    M -->|No| N[Evict from Mempool]
    M -->|Yes| O[Keep in Mempool]
    B -->|If valid| P[Gossip to Peers]
```

## Component Interactions

The transaction handling system interacts with several other components:

```mermaid
flowchart TD
    A[Controller] <-->|Transaction Handling| B[Mempool]
    A <-->|Network Communication| C[P2P Layer]
    B <-->|State Validation| D[State Machine/FSM]
    A <-->|Transaction Lookup| E[Blockchain Store]
    A <-->|Reputation Management| C
    F[RPC Interface] -->|Transaction Queries| A
```

## Security Features

The transaction handling system includes several security measures:

- Reputation system for peers that penalizes those sending invalid transactions
- Duplicate transaction detection to prevent replay attacks
- Fee-based prioritization to prevent spam
- Transaction validation against current state to prevent invalid state transitions
- Mempool rechecking to ensure only valid transactions remain when state changes
- Caching of failed transactions for monitoring and debugging
- Thread safety through locking mechanisms
