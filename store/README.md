# Package store

[![Go Reference](https://pkg.go.dev/badge/github.com/canopy/canopy-network.svg)](https://pkg.go.dev/github.com/canopy-network/canopy/store)
[![License](https://img.shields.io/github/license/canopy-network/canopy)](https://github.com/canopy/canopy-network/blob/main/LICENSE)

The store package serves as the storage layer for the Canopy blockchain, utilizing
**[BadgerDB](https://github.com/hypermodeinc/badger)** as its underlying key-value store. It offers
structured abstractions for nested transactions, state management, and indexed blockchain data
storage. This document provides a comprehensive overview of the package’s core components and their
functionalities, introducing key concepts while progressively building each component into a
complete storage solution.

## Basic Concepts

### Hashing

Hashing is the process of converting data into an unique "fingerprint". A hash function converts any
data into a fixed-size string of characters, making it useful for data verification and storage.

```mermaid
graph LR
    A["Hello World"] -->|Hash Function| B["68e109f..."]
    C["hello world"] -->|Hash Function| D["5eb63b..."]
```

Key Properties:

- **Same Input, Same Hash**: "Hello" always produces the same hash
- **Small Changes, Different Hash**: "Hello" and "hello" produce completely different hashes
- **One-way**: You can't recreate the original data from its hash
- **Fixed Size**: All hashes have the same length, regardless of input size

In blockchain, hashing is used to create unique IDs for transactions, verify data hasn't changed, and link blocks together.

### Persistent Storage

Persistent storage refers to data storage mechanisms where information persists after power loss or
process termination, unlike volatile memory (RAM) where data is temporary.

Persistent storage systems must balance performance, durability, and consistency requirements. While
RAM provides fast access but loses data on power loss, storage devices like SSDs and HDDs offer
permanence at the cost of slower access speeds. This tradeoff is particularly important in
blockchain systems where transaction history and state changes must be both quickly accessible and
permanently stored while maintaining data integrity across system restarts.

### Key-Value Storage

A key-value storage is like a dictionary where unique keys can be associated with values. It
provides a fast and efficient way to store and retrieve data.

Examples:

- Key: "user_123" → Value: `{name: "John", balance: 100}`
- Key: "settings" → Value: `{theme: "dark", notifications: "on"}`

Benefits:

- Fast lookups ([O(1)](https://en.wikipedia.org/wiki/Time_complexity) in ideal cases)
- Simple to understand and use
- Flexible value storage (can store any type of data)

### Transactions

A transaction is a group of operations that must either all succeed or all fail together. A good
analogy is like transferring money between two bank accounts.

```mermaid
graph LR
    A[Account A: $100<br>Account B: $0] --> B[A Transfers $50 to B] --> C[Account A: $50<br>Account B: $50]
```

If anything fails during the transfer, both accounts should return to their original state. As
clients would be pretty upset if they lost their money.

#### Transaction Properties (ACID)

1. **Atomicity**: All operations in a transaction either succeed or fail together
   - Example: In a money transfer, both the withdrawal and deposit must succeed together

2. **Consistency**: Data remains valid before and after the transaction
   - Example: Total money in all accounts must remain the same after transfers

3. **Isolation**: Multiple transactions don't interfere with each other
   - Example: Two people withdrawing money simultaneously shouldn't cause conflicts

4. **Durability**: Once a transaction is committed, changes are permanent
   - Example: After confirming a transfer, the new balances persist even if the system crashes

#### Nested Transactions

Nested transactions behave like a set of Russian dolls, where each transaction sits inside another:

```mermaid
graph TD
    A[Main Transaction] --> B[Sub-Transaction 1]
    A --> C[Sub-Transaction 2]
    B --> D[Sub-Sub-Transaction]
```

Benefits:

- More granular control over operations
- Ability to rollback partial operations
- Better organization of complex operations

### Why BadgerDB?

**[BadgerDB](https://github.com/hypermodeinc/badger)** is a fast, embeddable, persistent key-value
(KV) database written in pure Go. It's designed to be highly performant for both read and write
operations.

#### Key Features

1. **LSM Tree-based**: Uses a
   [Log-Structured Merge-Tree](https://en.wikipedia.org/wiki/Log-structured_merge-tree)
   architecture, optimized for SSDs
2. **ACID Compliant**: Ensures data consistency through Atomicity, Consistency, Isolation, and
   Durability
3. **Concurrent Access**: Supports multiple readers and a single writer simultaneously
4. **Key-Value Separation**: Stores keys and values separately to improve performance
5. **Transactions**: Native support for both read-only and read-write transactions. Every action in
   BadgerDB happens within a transaction
6. **Iteration**: Provides efficient iteration over key-value pairs that are byte-wise
   lexicographically ordered

### Putting It All Together

In Canopy's storage system:

1. BadgerDB provides the persistent key-value store
2. Transactions ensure data consistency
3. Nested transactions enable complex operations
4. Key-value pairs store blockchain state and data
5. Blockchain data is frequently stored hashed to improve security

This foundation helps understand the more complex features that will be discussed in the following
sections.

## **Store Package Components**

The store package is built from several key components that work together, like building blocks, to
create a complete storage system. It could be represented like a well-organized filing cabinet,
where each component has a specific job in managing and storing data. Each compoent from the
simplest to the most complex is described as follows:

1. **TxnWrapper**: It is a Wrapper around BadgerDB operations:
    - Makes sure all database operations follow the [`RWStoreI`](../lib/store.go) interface
    - Handles basic operations like storing and retrieving data
    - Allows iteration between ranges of data
    - Works like one of the translators between BadgerDB and the rest of Canopy

2. **`Txn`**: The transaction manager
    - Ensures that a group of operations happen together or not all (transactions)
    - Enhances BadgerDB by implementing in-memory nested transactions, to allows to write or discard
      groups of operations (like multiple read/writes) within a single BadgerDB transaction
    - Follows the [`RWStoreI`](../lib/store.go) interface to interact with the database

3. **`SMT`** (Sparse Merkle Tree): A special data structure for proving data existence
   - Organizes data in a tree-like structure
   - Makes it easy to prove whether data exists or doesn't exist
   - Uses smart optimizations to save space and work faster
   - Just like `TxnWrapper` it also follows the [`RWStoreI`](../lib/store.go) interface and
     leverages `TxnWrapper` itself in order to interact with the database

4. **`Indexer`**: The filing system
   - Organizes blockchain data (like blocks and transactions) in an easy-to-find way
   - Uses `TxnWrapper` under the hood to save the data in BadgerDB
   - Uses prefixes (like labels on filing cabinets) to group related data
   - Makes searching through data fast and efficient

5. **`Store`**: The main coordinator
   - Brings all other components together
   - Contains three main parts:
     - `Indexer`: Organizes and indexes blockchain data
     - `StateStore`: Stores the actual blockchain data using `TxnWrapper` under the hood
     - `StateCommitStore`: Stores hashes of the data using `SMT` under the hood
   - Is the only component that commits to the database directly

## **Key Interactions**

```mermaid
graph TD
    Store[Store] --> Indexer[Indexer<br/><i>Indexed data</i>]
    Store --> StateStore[StateStore<br/><i>Raw Data</i>]
    Store --> StateCommitStore[StateCommitStore<br/><i>Hash Data</i>]

    StateStore --> TxnWrapper[TxnWrapper]
    StateCommitStore --> SMT[SMT<br/><i>Sparse Merkle Tree</i>]
    Indexer --> TxnWrapper

    SMT --> TxnWrapper

    TxnWrapper --> BadgerDB[BadgerDB]
    Store --> BadgerDB
```

1. The `Store` manages three main components:
   - `Indexer` (for indexed blockchain data)
   - `StateStore` (for raw data)
   - `StateCommitStore` (for hash data)
   - Commits directly to the database the data on all the components

2. `TxnWrapper` acts as a bridge to BadgerDB:
   - Both `StateStore` and `Indexer` use it directly
   - `SMT` also uses it for data storage
   - Provides consistent way to interact with the database

3. `BadgerDB` serves as the foundation:
   - All data ultimately gets stored here
   - Everything flows through `TxnWrapper` to reach BadgerDB

This architecture ensures each component handles its specific tasks while maintaining consistent data storage and access patterns.

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
    classDef highlightSiblings fill:#ff0000,stroke:#333,stroke-width:2px;
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

Canopy's SMT implementation introduces key optimizations to address traditional SMT limitations:

1. **Optimized Node Structure**:
   - Nil leaf nodes for empty values
   - Parent nodes with single non-nil child are replaced by that child
   - A tree starts and always maintains two root children for consistent operations

2. **Efficient Tree Operations**:
   - Key organization via binary representation and common prefix storage for internal nodes
   - Targeted traversal that only visits relevant tree paths
   - Dynamic node creation/deletion with automatic tree restructuring

3. **Space and Performance**:
   - Eliminates storage of empty branches
   - Reduces hash computation overhead
   - Maintains compact tree structure without compromising security

### Core Algorithm Operations

1. **Tree Traversal**
   - Navigates downward to locate the closest existing node matching the target key's binary path

2. **Modification Operations**

   a. **Upsert (Insert/Update)**
   - Updates node directly if the target node matches current position
   - Otherwise:
     - Creates new parent node using the greatest common prefix between target and current node keys
     - Updates old parent's pointer to reference new parent
     - Sets current and target as children of new parent

   b. **Delete**
   - When target matches current position:
     - Removes current node
     - Updates grandparent to point to current's sibling
     - Removes current's parent node

3. **ReHash**
   - Progressively updates hash values from modified node to root after each operation
   - Ensures cryptographic integrity of tree structure.

#### Example operations

#### Insert 1101

<div style="display: flex;">
<div style="width: 50%;">
Before

```mermaid
graph TD
    root((root)) --> n0000[0000]
    root --> n1[1]
    n1 --> n1000[1000]
    n1 --> n111[111]
    n111 --> n1110[1110]
    n111 --> n1111[1111]
```

</div>
<div style="width: 50%;">
After

```mermaid
graph TD
    root((root)) --> n0000[0000]
    root --> n1[1]
    n1 --> n1000[1000]
    n1 --> n11[11 *new parent*]
    n11 --> n1101[1101 inserted node]
    n11 --> n111[111]
    n111 --> n1110[1110]
    n111 --> n1111[1111]
    %% n1000 --> n1101[1101]

    classDef highlightNewNode fill:#0000ff,stroke:#333;
    class n1101 highlightNewNode;
    classDef highlightRehashedNodes fill:#006622,stroke:#333;
    class n1,n11,root highlightRehashedNodes;

```

</div>
</div>

Steps:

1. **Path Finding**: Navigate down the tree following the binary path of '1101' until reaching the closest existing node ('111')

2. **Position Check**: Determine target node ('1101') doesn't exist at current position

3. **Parent Creation**: Form new parent node with key '11' (greatest common prefix between '1101' and '111')

4. **Restructure**: Update old parent to reference new parent node, making previous node ('111') a child of new parent

5. **Insert**: Add new node ('1101') as the second child of the new parent, maintaining binary tree properties

6. **ReHash**: Progressively updates hash values from the modified node'parent to the root after each
   operation, ensuring cryptographic integrity of tree structure as shown in the diagram in green.

#### Delete 010

<div style="display: flex;">
<div style="width: 50%;">
Before

```mermaid
graph TD;
    root((root)) --> n0["0"]
    root --> n1["1"]
    n0 --> n000["000"]
    n0 --> n010["010"]
    n1 --> n101["101"]
    n1 --> n111["111"]

    classDef highlightDeleteNode fill:#ff0000,stroke:#333;
    class n010 highlightDeleteNode;
```

</div>
<div style="width: 50%;">
After

```mermaid
graph TD;
    root((root)) --> n000["000 new parent"]
    root --> n1["1"]
    n1 --> n101["101"]
    n1 --> n111["111"]

    classDef highlightNewNode fill:#0000ff,stroke:#333;
    class n000 highlightNewNode;
    classDef highlightRehashedNodes fill:#006622,stroke:#333;
    class root highlightRehashedNodes;
```

</div>
</div>

Steps:

1. **Path Finding**: Navigate down the tree following the binary path of '010' until reaching the
   target node

2. **Node Removal**: Remove target node ('010') from tree structure

3. **Parent Update**: Replace parent node '0' in grandparent with target's sibling '000'

4. **Tree Rehash**: Recalculate hash values upward from '000' parent to the root (on this case is the
   same as root) to maintain integrity as shown in the diagram in green.

### Proof Generation and verification

Canopy's SMT implementation supports both proof-of-membership and proof-of-non-membership through
the `GetMerkleProof` and `VerifyProof` methods. These proofs enable verification of whether a
specific key-value pair exists in the tree without requiring access to the complete tree data.

#### Proof Generation (`GetMerkleProof`)

The proof generation process constructs an ordered path from the target leaf node (or the closest
existing node where the key would reside if present) back to the root, by including every sibling
node encountered along each branch of this traversal.

1. **Initial Node**: The first element in the proof is always the target node (for membership
   proofs) or the node at the potential location (for non-membership proofs)

2. **Sibling Collection**: For each level from the leaf to the root:
   1. Record the sibling node's key and value
   2. Store the sibling's position (left=0 or right=1) in the bitmask
      - These siblings are essential for reconstructing parent hashes

#### Proof Verification (`VerifyProof`)

The verification process reconstructs the root hash using the provided proof and compares it with
the known root, if the hashes match, the resulting tree is then used in order to verify the
existence of the requested key-value pair. As a side note, unlike traditional sparse merkle trees,
both the key and value are required in order to generate the parent's hash:

1. **Initial Setup**:
    - Create a temporary in-memory tree
    - Add the first proof node (target/potential location)
    - The first proof node's key and value hash are then used to build the parent of the upcoming node

2. **Hash Reconstruction**:
    - For each sibling in the proof:
      - Use the bitmask to determine sibling position
      - Combine the current hash with the sibling's hash
      - Compute the parent hash using the same function as the main tree
      - Compute the parent's key by calculating the Greatest Common Prefix (GCP) of the current
        node's key and the sibling's key
      - Save the resulting parent node in the temporary tree

3. **Root Hash Validation**:
    - Compare reconstructed root hash with provided root
      - If the hashes do not match, the proof is invalid and the verification fails

4. **Final Verification**:
    - Traverse the temporary tree to verify the existence of the requested key-value pair
      - For membership proofs: Verifies target exists and value matches
      - For non-membership proofs: Confirms target doesn't exist at expected position

## Indexer operations and prefix usage to optimize iterations

## Store struct and how it adds up all together
