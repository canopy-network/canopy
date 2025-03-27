# Package store

[![Go Reference](https://pkg.go.dev/badge/github.com/canopy/canopy-network.svg)](https://pkg.go.dev/github.com/canopy-network/canopy/store)
[![License](https://img.shields.io/github/license/canopy-network/canopy)](https://github.com/canopy/canopy-network/blob/main/LICENSE)

The `store` package implements the storage layer for the Canopy blockchain, leveraging
**[BadgerDB](https://github.com/hypermodeinc/badger)** as its underlying key-value store. It
provides structured abstractions for nested transactions, state management, and indexed blockchain
data storage.

## **Core Components**

1. **`Txn`**: The foundational transactional layer. This implements ad-hoc **nested transactions** on top
   of BadgerDB, enabling atomic operations and rollbacks for complex storage workflows.

2. **`TxnWrapper` & `SMT`**:
   - **`TxnWrapper`**: Wraps BadgerDB to conform to the `RWStoreI` interface, providing a simple
     read/write abstraction with transaction support.
   - **`SMT`**: An optimized Sparse Merkle Tree implementation backed by BadgerDB. It adheres to the
     `RWStoreI` interface and enables efficient cryptographic commitment to state data and proof of
     membership/non-membership of the keys.

3. **`Indexer`**: Built on `TxnWrapper`, this component organizes blockchain data (blocks,
   transactions, addresses, etc.) using **prefix-based keys**. This design allows efficient
   iteration and querying of domain-specific data (e.g., "all transactions in block X").

4. **`Store`**: The top-level struct coordinating the storage layer:
   - **`Indexer`**: Manages indexed blockchain operations.
   - **`StateStore`**: Stores raw blockchain state data (blobs) using `TxnWrapper`.
   - **`StateCommitStore`**: Uses the `SMT` implementation to cryptographically commit hashes of
     `StateStore` data into the Sparse Merkle Tree, ensuring tamper-evident state verification.

## **Key Interactions**

- **Transactions**: `Txn` provides atomicity for operations across BadgerDB.
- **State Management**: `StateStore` (raw data) and `StateCommitStore` (hashes in SMT) work in
  tandem to balance performance with cryptographic integrity.
- **Querying**: The `Indexer`’s prefix-based structure enables fast, type-specific data retrieval.

This layered design decouples storage concerns while ensuring compatibility with BadgerDB’s
performance characteristics and the blockchain’s integrity requirements.

## Understanding BadgerDB and Transaction Management in Canopy

**[BadgerDB](https://github.com/hypermodeinc/badger)** is a fast, embeddable, persistent key-value
(KV) database written in pure Go. It's designed to be highly performant for both read and write
operations.

### Key Features

1. **LSM Tree-based**: Uses a Log-Structured Merge-Tree architecture, optimized for SSDs
2. **ACID Compliant**: Ensures data consistency through Atomicity, Consistency, Isolation, and
   Durability
3. **Concurrent Access**: Supports multiple readers and a single writer simultaneously
4. **Key-Value Separation**: Stores keys and values separately to improve performance
5. **Transactions**: Native support for both read-only and read-write transactions
6. **Iteration**: Provides efficient iteration over key-value pairs that are byte-wise
   lexicographically ordered

### TxnWrapper

BadgerDB's native transaction system provides atomic operations through its `Txn` type, supporting
both read-only and read-write transactions. Each transaction works with a consistent snapshot of the
database, ensuring data integrity during concurrent operations.

The `TxnWrapper` in Canopy builds upon this foundation by providing a clean abstraction layer over
BadgerDB's transaction system. It serves two main purposes:

1. **Interface Compliance**: Implements the `RWStoreI` interface, establishing a consistent contract
   for all storage operations within Canopy. This standardization ensures that different components
   can interact with the storage layer uniformly.

2. **Transaction Management**: Encapsulates BadgerDB's transaction handling, supporting both
   read-only and read-write operations. This wrapper simplifies transaction management by:
   - Providing a cleaner API for common database operations
   - Extending support of BadgerDB's iterator functionality
   - Returning errors in a consistent manner based on the project's error handling conventions

The main operations `TxnWrapper` is set to support according to the `RWStoreI` interface are:

- `Get(key []byte) ([]byte, ErrorI)`: Retrieves the value associated with the given key.
- `Set(key, value []byte) ErrorI`: Sets the value for the given key.
- `Delete(key []byte) ErrorI`: Deletes the value associated with the given key.
- `Iterator(prefix []byte) (IteratorI, ErrorI)`: Creates an iterator over byte-wise
  lexicographically sorted key-value pairs that start with the given prefix.
- `RevIterator(prefix []byte) (IteratorI, ErrorI)`: Creates a reverse iterator over byte-wise
  lexicographically sorted key-value pairs that start with the given prefix.

Note that all operations within `TxnWrapper` are neither committed nor rolled back directly, as
`TxnWrapper` operates within the broader transaction scope managed by the `Store` struct, which
handles the final commit or rollback decisions.

#### Key prefixing

All keys in `TxnWrapper` are automatically prefixed with a unique identifier (e.g., "s/" for state
store, "c/" for commitment store) to achieve two main purposes:

1. **Data Isolation**: Each component (`StateStore`, `StateCommitStore`, `Indexer`) maintains its own
   prefix-based namespace, preventing key collisions in the shared BadgerDB instance.

2. **Efficient Iteration**: Since BadgerDB stores keys in lexicographical order, prefixes enable
   efficient range queries within specific components. For example, iterating through all state
   store entries ("s/...") without touching commitment store data ("c/...").

These prefixes are only used internally and never exposed to the user.

## Txn ad hoc nested transactions implementation

## Canopy's Optimized Sparse Merkle Tree

A [Merkle Tree](https://en.wikipedia.org/wiki/Merkle_tree) is a data structure that enables
efficient and secure verification of large data sets. It works by hashing pairs of nodes and
progressively combining them upwards to create a single "root" hash that cryptographically commits
to all the data below it. A Sparse Merkle Tree (SMT) is a specialized variant that efficiently
handles sparse key-value mappings, making it particularly useful for storing state data and proof
the existence or non-existence of keys.

### Traditional Sparse Merkle Tree

```mermaid
graph TD
    Root --> H0
    Root --> H1
    H0 --> H00
    H0 --> H01
    H1 --> H10
    H1 --> H11
    H00 --> L0[Value 1]
    H00 --> L1[Empty]
    H01 --> L2[Empty]
    H01 --> L3[Empty]
    H10 --> L4[Empty]
    H10 --> L5[Empty]
    H11 --> L6[Empty]
    H11 --> L7[Empty]

    classDef highlight fill:#0000ff,stroke:#333,stroke-width:2px;
    class Root,H0,H00,L0 highlight;
    classDef highlightSiblings fill:#cc2900,stroke:#333,stroke-width:2px;
    class H01,H1,L1 highlightSiblings;
```

In a traditional Sparse Merkle Tree, as illustrated above, each level splits into two child nodes,
creating a binary tree structure. The blue nodes show the path to `Value 1`, while the red nodes
represent the sibling hashes needed for proof verification. For example, to prove the existence of
`Value 1`, we need:

1. The hash of `Value 1` itself
2. Its immediate sibling hash (L1)
3. The sibling hash at the next level (H01)
4. And finally, H1 to reconstruct the root

This approach has significant limitations:

- **Storage overhead**: Even for sparse data, the tree maintains placeholder nodes for empty
  branches
- **Computational cost**: Proof generation requires multiple hash computations along the path
- **Scalability challenges**: As the tree grows, both storage and computational requirements
  increase exponentially

These limitations become particularly problematic in blockchain environments where efficiency and
scalability are crucial. Canopy's SMT implementation introduces several optimizations to address
these challenges while maintaining the security properties of traditional Sparse Merkle Trees.

### Canopy's Sparse Merkle Tree

Some of the key optimizations of Canopy's SMT are:

- **Sparse Structure**: Keys are organized by their binary representation, with internal nodes storing
  common prefixes to reduce redundant paths

- **Optimized Traversals**: Operations like insertion, deletion, and lookup focus only on the relevant
  parts of the tree, minimizing unnecessary traversal of empty nodes

- **Key-Value Operations**: Supports upserts and deletions by dynamically creating or removing nodes
  while maintaining the Merkle tree structure

## Indexer operations and prefix usage to optimize iterations

## Store struct and how it adds up all together
