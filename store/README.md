# Package store

[![Go Reference](https://pkg.go.dev/badge/github.com/canopy/canopy-network.svg)](https://pkg.go.dev/github.com/canopy-network/canopy/store)
[![License](https://img.shields.io/github/license/canopy-network/canopy)](https://github.com/canopy/canopy-network/blob/main/LICENSE)

The `store` package implements the storage layer for the Canopy blockchain, leveraging **[BadgerDB](https://github.com/hypermodeinc/badger)** as its underlying key-value store. It provides structured abstractions for nested transactions, state management, and indexed blockchain data storage. Below is a high-level breakdown of its components:

### **Core Components**  

1. **`Txn`**:  
   The foundational transactional layer. This implements **nested transactions** on top of BadgerDB, enabling atomic operations and rollbacks for complex storage workflows.  

2. **`TxnWrapper` & `SMT`**:  
   - **`TxnWrapper`**: Wraps BadgerDB to conform to the `RWStoreI` interface, providing a simple read/write abstraction with transaction support.  
   - **`SMT`**: An optimized Sparse Merkle Tree implementation backed by BadgerDB. It adheres to the `RWStoreI` interface and enables efficient cryptographic commitment to state data and proof of membership/non-membership of the keys.

3. **`Indexer`**:  
   Built on `TxnWrapper`, this component organizes blockchain data (blocks, transactions, addresses, etc.) using **prefix-based keys**. This design allows efficient iteration and querying of domain-specific data (e.g., "all transactions in block X").  

4. **`Store`**:  
   The top-level struct coordinating the storage layer:  
   - **`Indexer`**: Manages indexed blockchain operations.  
   - **`StateStore`**: Stores raw blockchain state data (blobs) using `TxnWrapper`.  
   - **`StateCommitStore`**: Uses the `SMT` implementation to cryptographically commit hashes of `StateStore` data into the Sparse Merkle Tree, ensuring tamper-evident state verification.  

### **Key Interactions**  
- **Transactions**: `Txn` provides atomicity for operations across BadgerDB.  
- **State Management**: `StateStore` (raw data) and `StateCommitStore` (hashes in SMT) work in tandem to balance performance with cryptographic integrity.  
- **Querying**: The `Indexer`’s prefix-based structure enables fast, type-specific data retrieval.  

This layered design decouples storage concerns while ensuring compatibility with BadgerDB’s performance characteristics and the blockchain’s integrity requirements.  

## BadgerDB simple introduction and its usage on TxnWrapper

## Txn ad hoc nested transactions implementation

## Canopy's Sparse Merkle Tree Implementation

## Indexer operations and prefix usage to optimize iterations

## Store struct and how it adds up all together
