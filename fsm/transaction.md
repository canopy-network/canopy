# transaction.go - Transaction Processing in the Canopy Blockchain

This file contains the core transaction handling logic for the Canopy blockchain's state machine. It defines how transactions are processed, validated, and applied to the blockchain state.

## Overview

The transaction handling system is designed to:
- Process and validate incoming transactions
- Check transaction signatures and authorization
- Prevent replay attacks and hash collisions
- Apply transaction changes to the blockchain state
- Create various types of transactions for different operations

## Core Components

### Transaction Processing

The transaction processing system handles the lifecycle of a transaction from validation to execution. It ensures that transactions are properly formed, authorized, and can be safely applied to the blockchain state. This includes checking signatures, validating message payloads, and ensuring sufficient fees are provided.

When a transaction is submitted to the blockchain, it goes through several validation steps before being applied to the state. This ensures that only valid transactions are processed, maintaining the integrity of the blockchain.

### Transaction Validation

Transaction validation is a multi-step process that ensures transactions meet all requirements before being accepted. This includes basic validation of the transaction structure, replay protection, signature verification, and fee validation.

The validation process is crucial for maintaining blockchain security. It prevents malicious actors from submitting invalid transactions or replaying previously executed transactions, which could potentially disrupt the blockchain's operation or lead to double-spending.

### Transaction Creation

The file provides numerous helper functions for creating different types of transactions. These functions make it easier to construct properly formatted transactions for various operations like sending tokens, staking, changing parameters, and more.

Each transaction type serves a specific purpose in the blockchain ecosystem, allowing users to perform different actions like transferring tokens, participating in consensus through staking, or modifying blockchain parameters through governance.

## Technical Details

### Transaction Lifecycle

When a transaction is submitted to the blockchain, it goes through the following process:

1. **Validation (CheckTx)**: The transaction is validated to ensure it meets all requirements:
   - Basic structure validation
   - Replay protection checks
   - Message payload validation
   - Signature verification
   - Fee validation

2. **Fee Deduction**: If validation passes, fees are deducted from the sender's account to pay for transaction processing.

3. **Message Handling**: The specific message payload is processed, which may update various aspects of the blockchain state.

4. **Result Creation**: A transaction result is created, containing information about the processed transaction.

This process ensures that only valid transactions are applied to the blockchain state, maintaining the integrity and security of the system.

### Replay Protection

Canopy uses a unique approach to prevent replay attacks and hash collisions:

- Instead of using a traditional sequence number, Canopy uses a combination of timestamp and created block height.
- The timestamp adds microsecond-level entropy to the transaction hash, making hash collisions extremely unlikely.
- The created block height must be within an acceptable range of the current height (±4320 blocks). Assuming 20s block times, this is roughly 24 hours and allows safe pruning of old transaction data.

This approach is "prune-friendly," meaning that nodes don't need to store the entire transaction history to prevent replay attacks, which helps keep the blockchain more efficient.

### Signature Verification

Transaction signatures are verified using public key cryptography:

1. The transaction is serialized into a canonical byte representation.
2. The signature is verified against these bytes using the sender's public key.
3. The system checks if the signer is authorized to perform the requested action.

For certain message types (like stake operations or parameter changes), additional information is populated during signature verification, such as the signer field or proposal hash.

### Transaction and State Machine

Transactions interact with the state machine to update the blockchain state:

- The state machine provides the context for transaction validation and execution.
- Transactions read from and write to the state machine's store.
- The state machine maintains the current blockchain height, network ID, and chain ID, which are used for transaction validation.

This interaction ensures that transactions are processed in a consistent and deterministic manner, maintaining the integrity of the blockchain state.

### Transaction and Fee System

Transactions must include sufficient fees to be processed:

- Each message type has a minimum required fee set in the blockchain parameters.
- The transaction validation process checks if the provided fee meets or exceeds this minimum.
- Fees are deducted from the sender's account before the transaction is processed.

This fee system helps prevent spam and ensures that transaction processors (validators) are compensated for their work.

### Transaction and Message Types

Transactions are containers for different types of messages:

- Each transaction contains a specific message type (like Send, Stake, ChangeParameter, etc.).
- The message determines what action the transaction will perform.
- Different message types have different validation rules and effects on the blockchain state.

The file provides helper functions for creating various types of transactions, each with a specific message type for operations like sending tokens, staking, changing parameters, and more.

## Security & Integrity Mechansisms

- **Signature Verification**: All transactions must be properly signed by an authorized sender.
- **Replay Protection**: The system prevents transaction replay attacks using timestamps and block heights.
- **Fee Requirements**: Minimum fees help prevent spam and denial-of-service attacks.
- **Authorization Checks**: The system verifies that transaction signers are authorized to perform the requested actions.
- **Hash Collision Prevention**: Microsecond-level timestamps add entropy to transaction hashes, making collisions extremely unlikely.
