# Oracle Package

## Multi-Oracle Consensus with Validator Voting

The Oracle package provides cross-chain transaction witnessing and validation capabilities for the Canopy blockchain. It implements a chain-agnostic oracle that coordinates between external blockchains (like Ethereum) and a Canopy nested chain (observer chain) running this software to facilitate cross-chain order execution and validation.

The Canopy oracle nested chain employs a witness-based consensus mechanism that combines independent validator nodes with the NestBFT consensus algorithm to ensure reliable attestation of external blockchain transactions.

Each validator node in the committee independently monitors external chains (such as Ethereum) through configurable block providers and witnesses lock/close order transactions. When a relevant transaction is detected, oracle nodes validate it against the current order book and stores any witnessed orders locally.

They participate in the NestBFT consensus protocol where witnessed orders are proposed in blocks and validated against each nodes' witnessed orders. Thie ensures that the required +2/3 supermajority agreement among participating validators before any witnessed order is finalized on the observer chain and reported to the root chain.

## Overview

The Oracle package is designed to handle:
- Witnessing transactions on external blockchains containing Canopy lock & close orders
- Validating and storing witnessed orders in a local order store
- Participating in the BFT consensus process by providing witnessed orders for block proposals
- Synchronizing with the root chain order book to maintain consistency
- Managing persistent state for reliable order processing

## Core Components

### Oracle

The core of the Canopy Oracle system. It manages the overall cross-chain witnessing process, including:
- Receiving blocks from block providers
- Validating witnessed orders against the root chain order book
- Persisting witnessed orders to local storage
- Coordinating with the BFT consensus mechanism
- Maintaining synchronization with root chain order book state

# BlockProvider Integration

The Oracle integrates with external block providers through the `BlockProvider` interface. It provides:
- Real-time block monitoring from external chains
- Transaction parsing and order extraction
- Integration with Oracle's state management for gap detection and reorg handling

# Order Store Management

The Oracle manages witnessed orders through a persistent store that:
- Stores validated lock and close orders separately by type
- Provides atomic read/write operations for order data
- Maintains submission history with timestamps for resubmission logic
- Supports cleanup operations based on root chain order book updates
- Archives processed orders for audit and recovery purposes

## Sequence Diagram

The following sequence diagram illustrates the core interactions in the Oracle package:

```mermaid
sequenceDiagram
    participant SC as Source Chain
    participant BP as BlockProvider
    participant O as Oracle
    participant BFT as BFT
    participant RC as Root Chain

    %% Block retrieval and processing
    Note over SC,O: Source Chain Block Processing
    SC->>BP: New block header received
    BP->>BP: Calculate safe block height
loop Fetch Safe Blocks
    BP->>SC: Fetch block
    SC->>BP: Return block data
    BP->>O: Send block via channel
end
    
    %% Oracle block processing
    O->>O: Validate & write to store
    
    Note over BFT,O: Consensus Participation
    %% BFT consensus integration
    BFT->>O: Request witnessed orders
    O->>O: Check should submit logic
    O->>BFT: Return witnessed orders
    BFT->>BFT: Produces block with witnessed orders
    
    %% Block proposal validation
    BFT->>O: Block proposal validation
    O->>O: Compare proposed vs witnessed orders
    O->>BFT: Return validation result
    BFT->>BFT: Commit Certificate
    
    Note over O,RC: Root Chain Interaction
    %% Root Chain interaction
    BFT->>RC: Certificate Results
    RC->>O: Synchronize order store to order book
```

## Technical Details

### Cross-Chain Transaction Witnessing

The Oracle system uses a block-based monitoring approach to witness transactions on external chains. This is achieved by:

- **Block Provider Integration**: Connects to external blockchain nodes through configurable providers
- **Transaction Parsing**: Extracts Canopy-specific order data from external chain transactions
- **Order Validation**: Performs comprehensive validation against root chain order book data
- **State Persistence**: Maintains reliable state storage for witnessed orders and processing height

The system works like a specialized blockchain monitor that specifically looks for transactions containing Canopy order data, validates them against known orders, and stores them for later use in the consensus process.

State persistence ensures that the Oracle can recover from interruptions without losing witnessed orders or reprocessing previously seen blocks.

### BFT Consensus Integration

The Oracle system uses a dual-phase approach to participate in Byzantine Fault Tolerant consensus:

1. **Proposal Phase**: When acting as a proposer, the Oracle queries its witnessed order store to find orders that should be included in the next block proposal
2. **Validation Phase**: When validating block proposals from other nodes, the Oracle verifies that all proposed orders exist in its local witnessed order store

This ensures that only orders witnessed by a majority of validator nodes are included in the blockchain, providing strong guarantees about cross-chain transaction validity.

### Order Book Synchronization

The Oracle system implements several synchronization mechanisms to maintain consistency with the root chain:

- **Order Book Updates**: Receives periodic updates of the complete root chain order book state
- **Stale Order Cleanup**: Automatically removes witnessed orders that are no longer relevant (locked orders, completed orders)
- **State Validation**: Ensures witnessed orders match current order book entries before including them in block proposals

This synchronization acts like a cache invalidation system, where the Oracle maintains local copies of relevant data but periodically synchronizes with the authoritative root chain state.

## Component Interactions

### 1. Block Processing: External Chain Monitoring

When a new block arrives from an external blockchain, the Oracle performs the following:

- **Block Reception**: Receives blocks through a channel-based interface from the configured BlockProvider
- **Height Persistence**: Saves the current block height to disk before processing to enable recovery
- **Transaction Analysis**: Examines each transaction in the block for Canopy-specific order data
- **Order Validation**: Validates witnessed orders against the current root chain order book
- **Storage Operations**: Persists valid orders to the local order store with appropriate metadata

This process is similar to how a blockchain indexer works, but with specific focus on cross-chain order validation and storage.

### 2. Consensus Participation: BFT Integration

The Oracle participates in the BFT consensus process through two key interfaces:

- **WitnessedOrders**: Called during block proposal to provide witnessed orders that should be included in the next block
- **ValidateProposedOrders**: Called during block validation to verify that proposed orders were actually witnessed by this node

This is analogous to how a validator node participates in consensus, but with specialized logic for cross-chain order validation.

### 3. Root Chain Synchronization: State Management

- **Order Book Updates**: Receives and processes complete order book state from the root chain
- **Cleanup Operations**: Removes witnessed orders that are no longer needed based on root chain state
- **Consistency Validation**: Ensures witnessed orders still match their corresponding root chain entries

## Cross-Chain Security Features

The Oracle system implements several security mechanisms to ensure reliable cross-chain operation:

- **Comprehensive Validation**: Performs strict validation of all witnessed order fields against root chain data

## Configuration

The Oracle accepts configuration through `lib.OracleConfig` with the following parameters:

- `OrderStorePath`: Filesystem path to store witnessed orders (default: `$HOME/.canopy/oracle/orders`)
- `StateSaveFile`: File path for persisting the last processed block height (default: `$HOME/.canopy/oracle/last_block_height.txt`)
- `LogPath`: Path to store oracle logs (default: `$HOME/.canopy/oracle/log`)
- `OrderResubmitDelay`: Number of root blocks to wait before resubmitting an order to consensus (default: `2`)

