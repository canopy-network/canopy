# Feature Plan: Geth EVM Integration for Solidity Smart Contracts

## Overview
- **Problem**: Canopy blockchain currently only supports limited pseudo-contract functionality through RLP translation. Users cannot deploy and execute arbitrary Solidity smart contracts.
- **Solution**: Integrate the Geth EVM (Ethereum Virtual Machine) to enable full Solidity smart contract execution capability within the Canopy blockchain.
- **Target Users**: End users wanting to deploy and interact with Solidity smart contracts on Canopy

## Technical Impact

### Components Affected
- **FSM (Primary)**: Core EVM integration, new transaction types, state management bridge
- **Store**: EVM state storage, contract code storage, transaction logs
- **Controller**: EVM transaction routing and execution coordination
- **RPC/CLI**: New endpoints for contract deployment and interaction
- **Library**: Gas calculation utilities, EVM-specific cryptographic functions

### New Dependencies
- **github.com/ethereum/go-ethereum/core/vm**: Core EVM implementation
- **github.com/ethereum/go-ethereum/common**: Ethereum common types
- **github.com/ethereum/go-ethereum/core/types**: Ethereum transaction types
- **github.com/ethereum/go-ethereum/params**: EVM configuration parameters

### API Changes
- **New Transaction Types**: Contract deployment, contract call transactions
- **New RPC Endpoints**: `eth_sendTransaction`, `eth_call`, `eth_getCode`, `eth_estimateGas`
- **New CLI Commands**: `deploy-contract`, `call-contract`, `get-contract-code`
- **Enhanced State Queries**: Contract storage queries, event log retrieval

## Implementation Phases

### Phase 1 - Core EVM State Bridge (Week 1-2)
- [ ] Implement `StateDB` interface in `fsm/evm_state.go`
  - [ ] Bridge Canopy key-value store to EVM state operations
  - [ ] Account balance management (`GetBalance`, `AddBalance`, `SubBalance`)
  - [ ] Nonce management (`GetNonce`, `SetNonce`)
  - [ ] Contract code storage (`GetCode`, `SetCode`, `GetCodeHash`)
  - [ ] Contract storage operations (`GetState`, `SetState`, `GetCommittedState`)
  - [ ] Snapshot and revert functionality for transaction rollback
- [ ] Implement `ChainContext` interface in `fsm/evm_chain.go`
  - [ ] Block header retrieval from Canopy block format
  - [ ] Chain configuration management
- [ ] Create EVM configuration in `fsm/evm_config.go`
  - [ ] Define Canopy chain parameters and EVM fork configurations
  - [ ] Set up gas pricing and limits

### Phase 2 - Transaction Processing Integration (Week 3-4)
- [ ] Implement `Message` interface in `fsm/evm_message.go`
  - [ ] Wrap Canopy transactions for EVM execution
  - [ ] Handle contract deployment vs contract call transactions
  - [ ] Gas limit and pricing integration
- [ ] Extend `HandleMessage()` in `fsm/message.go`
  - [ ] Add `CONTRACT_DEPLOY` and `CONTRACT_CALL` message types
  - [ ] Route to EVM execution handlers
- [ ] Create EVM execution logic in `fsm/evm_executor.go`
  - [ ] Contract deployment transaction handling
  - [ ] Contract call transaction handling
  - [ ] Gas consumption tracking and refunds
  - [ ] Transaction receipt generation with logs

### Phase 3 - Storage and State Management (Week 5)
- [ ] Extend storage key structure in `fsm/key.go`
  - [ ] Add `contractCodePrefix` for contract bytecode storage
  - [ ] Add `contractStoragePrefix` for contract state storage
  - [ ] Add `transactionLogPrefix` for event log storage
- [ ] Implement EVM state persistence in `fsm/evm_storage.go`
  - [ ] Contract bytecode storage and retrieval
  - [ ] Contract storage trie management
  - [ ] Event log indexing and querying
- [ ] Add state root calculation for EVM state
  - [ ] Integrate with existing Sparse Merkle Tree implementation
  - [ ] Ensure state root includes EVM state changes

### Phase 4 - RPC and CLI Integration (Week 6)
- [ ] Add EVM RPC endpoints in `cmd/rpc/evm.go`
  - [ ] `eth_sendTransaction` - Deploy/call contracts
  - [ ] `eth_call` - Read-only contract calls
  - [ ] `eth_getCode` - Retrieve contract bytecode
  - [ ] `eth_estimateGas` - Gas estimation for transactions
  - [ ] `eth_getLogs` - Query contract event logs
- [ ] Add EVM client functions in `cmd/rpc/client.go`
  - [ ] Contract deployment client functions
  - [ ] Contract interaction client functions
  - [ ] Contract query client functions
- [ ] Add CLI commands in `cmd/cli/evm.go`
  - [ ] `deploy-contract` - Deploy Solidity contracts
  - [ ] `call-contract` - Execute contract functions
  - [ ] `query-contract` - Read contract state
  - [ ] `get-contract-logs` - Retrieve event logs

### Phase 5 - Testing and Validation (Week 7-8)
- [ ] Unit tests for EVM integration
  - [ ] StateDB interface implementation tests
  - [ ] Contract deployment and execution tests
  - [ ] Gas calculation and consumption tests
  - [ ] State persistence and rollback tests
- [ ] Integration tests with existing Canopy functionality
  - [ ] EVM transactions alongside native Canopy transactions
  - [ ] Fee payment in CNPY tokens for contract execution
  - [ ] Validator consensus on EVM state changes
- [ ] End-to-end testing
  - [ ] Deploy sample Solidity contracts (ERC20, simple storage)
  - [ ] Test contract interactions via RPC and CLI
  - [ ] Performance benchmarking vs native transactions

### Phase 6 - Documentation and Examples (Week 9)
- [ ] Code documentation
  - [ ] Interface documentation for EVM integration
  - [ ] Gas calculation and pricing documentation
  - [ ] State management architecture documentation
- [ ] User guides
  - [ ] Solidity contract deployment guide
  - [ ] Contract interaction examples
  - [ ] Gas optimization best practices
- [ ] API documentation
  - [ ] RPC endpoint specifications
  - [ ] CLI command reference
  - [ ] Error handling and troubleshooting

## Risk Mitigation

### High Risk
- **State Consistency**: EVM state must remain consistent with Canopy consensus
  - *Mitigation*: Implement atomic transactions and comprehensive rollback mechanisms
  - *Mitigation*: Extensive testing of state transitions and edge cases
- **Gas DoS Attacks**: Malicious contracts could consume excessive gas
  - *Mitigation*: Implement proper gas limits and pricing mechanisms
  - *Mitigation*: Monitor gas usage patterns and implement circuit breakers
- **Performance Impact**: EVM execution may slow down block processing
  - *Mitigation*: Benchmark EVM vs native transaction performance
  - *Mitigation*: Implement execution timeouts and resource limits

### Medium Risk
- **Memory Usage**: Contract storage could grow large over time
  - *Mitigation*: Implement efficient storage pruning mechanisms
  - *Mitigation*: Monitor storage growth and implement limits if needed
- **Compatibility Issues**: Geth EVM version compatibility with Canopy
  - *Mitigation*: Pin to specific Geth version and thoroughly test upgrades
  - *Mitigation*: Implement version detection and compatibility checks

### Low Risk
- **Transaction Ordering**: EVM transactions mixed with native transactions
  - *Mitigation*: Ensure deterministic transaction ordering in blocks
- **Fee Calculation**: Gas pricing integration with CNPY token fees
  - *Mitigation*: Implement clear gas-to-CNPY conversion mechanisms

## Success Criteria
- [ ] Users can deploy Solidity contracts via RPC/CLI
- [ ] Users can interact with deployed contracts (read/write operations)
- [ ] Contract state persists correctly across blocks
- [ ] Gas consumption is calculated accurately
- [ ] EVM transactions integrate seamlessly with existing Canopy functionality
- [ ] Performance impact is within acceptable limits (<20% slowdown)
- [ ] All tests pass including edge cases and error conditions

## Dependencies and Prerequisites
- **Existing Ethereum Compatibility**: Canopy already has RLP transaction support and signature verification
- **Key-Value Store**: Current store interface is sufficient for EVM state storage
- **Account Management**: Existing account system can be extended for contract accounts
- **Gas Infrastructure**: Need to integrate EVM gas model with existing fee structure

## Next Steps
1. **Review and Approval**: Get team approval for this implementation plan
2. **Environment Setup**: Ensure Geth dependencies are properly imported
3. **Phase 1 Implementation**: Start with StateDB interface implementation
4. **Iterative Development**: Implement and test each phase incrementally
5. **Community Testing**: Deploy to testnet for community validation