# byzantine.go - Byzantine Fault Handling in Canopy Blockchain

This file implements the mechanisms for detecting and penalizing byzantine (faulty or malicious) behavior in the Canopy blockchain network. It focuses on two main types of byzantine behavior: validators who fail to sign quorum certificates (non-signers) and validators who sign conflicting blocks at the same height (double-signers).

## Overview

The byzantine handling system is designed to:
- Track validators who don't participate in consensus (non-signers)
- Detect validators who sign multiple conflicting blocks (double-signers)
- Slash (penalize by burning tokens) validators who exhibit byzantine behavior
- Remove validators from committees when their behavior threatens network security
- Maintain the integrity of the blockchain by enforcing penalties for misbehavior

## Core Components

### Byzantine Detection

The system identifies two types of byzantine behavior:
- Non-signers: Validators who fail to sign quorum certificates
- Double-signers: Validators who sign conflicting blocks at the same height

The detection happens during the processing of each quorum certificate, where the system checks which validators should have signed but didn't, and also processes any reports of double-signing.

### Slashing Mechanism

When validators exhibit byzantine behavior, they are penalized through "slashing" - a process that burns a percentage of their staked tokens. The system implements:
- Different slash percentages for different types of violations
- Maximum slash limits per committee per block
- Tracking of slashes to prevent excessive penalties

### Non-Signer Tracking

The system maintains a counter for each validator who fails to sign quorum certificates:
- Increments a counter each time a validator fails to sign
- Periodically checks if any validator has exceeded the maximum allowed non-signs
- Slashes and resets validators who exceed the threshold

### Double-Signer Handling

For validators who sign conflicting blocks:
- Validates reported double-signing evidence
- Indexes double-signers by address and block height
- Applies more severe slashing penalties than for non-signing

### Validator Management

After slashing, the system may:
- Force validators to begin unstaking if their behavior is severe
- Remove validators from committees
- Update validator stake amounts
- Track validators across different committees

### Slashing Process

The slashing process follows these steps:

1. **Validation**: Ensure the validator exists and the slashing is authorized
2. **Calculation**: Determine the percentage of tokens to slash based on the type of violation
3. **Tracking**: Update the slash tracker to ensure no committee exceeds maximum slash per block
4. **Token Burning**: Reduce the validator's stake and the total token supply
5. **Committee Updates**: Potentially remove the validator from committees
6. **State Updates**: Update the validator's information in the state machine

This process ensures that penalties are applied fairly and consistently, while preventing excessive slashing that could destabilize the network.

### Non-Signer Window

The system uses a "non-sign window" concept:
- Tracks non-signing over a configurable number of blocks
- When the end of a window is reached, validators exceeding the maximum non-signs are slashed
- After slashing, the non-signer tracking is reset for the next window
- In protocol v2, reset semantics are intentionally single-settlement-per-window:
  the first chain processed during reset performs settlement, then all non-signer
  evidence for that window is erased before later chains are processed.

This approach allows for occasional missed signatures (which might happen due to network issues) while still penalizing persistent non-participation.

### Slash Tracker

The SlashTracker is a specialized component that:
- Maps validator addresses to committees to slash percentages
- Ensures no committee can slash a validator beyond the maximum allowed percentage in a single block
- Prevents excessive penalties that could unfairly harm validators

### Quorum Certificate Processing

When a quorum certificate (QC) is processed:
1. The system extracts the list of validators who should have signed but didn't
2. It increments the non-signing counter for each of these validators
3. If the current block marks the end of a non-sign window, it checks if any validators exceeded the threshold
4. It processes any double-signer reports included in the QC

This integration with the consensus mechanism ensures that byzantine behavior is detected and penalized as part of normal blockchain operation.

### Validator State Management

The byzantine handling system interacts closely with validator management:
1. When validators are slashed, their stake amounts are reduced
2. If stake falls to zero, the validator is removed entirely
3. Validators may be removed from committees based on their behavior
4. The system tracks which validators are part of which committees to ensure proper slashing

This ensures that the validator set remains healthy and that byzantine validators cannot continue to harm the network.

### Evidence Expiration

The system implements an evidence expiration mechanism:
1. Evidence of byzantine behavior is only valid for a certain period
2. The minimum evidence height is calculated based on the unstaking period
3. Evidence older than this height is considered expired and not processed

This prevents old evidence from being used maliciously against validators who may have already been penalized or left the network.
