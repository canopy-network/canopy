package lib

import (
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cockroachdb/pebble/v2"
)

/* This file contains persistence module interfaces that are used throughout the app */

// StoreI defines the interface for interacting with blockchain storage
type StoreI interface {
	RWStoreI                                     // reading and writing
	ProveStoreI                                  // proving membership / non-membership
	RWIndexerI                                   // reading and writing indexer
	NewTxn() StoreI                              // wrap the store in a discardable nested store
	Root() ([]byte, ErrorI)                      // get the merkle root from the store
	DB() *pebble.DB                              // retrieve the underlying badger db
	Version() uint64                             // access the height of the store
	Copy() (StoreI, ErrorI)                      // make a clone of the store
	NewReadOnly(version uint64) (StoreI, ErrorI) // historical read only version of the store
	Commit() (root []byte, err ErrorI)           // save the store and increment the height
	Discard()                                    // discard the underlying writer
	Reset()                                      // reset the underlying writer
	Close() ErrorI                               // gracefully stop the database
	Flush() ErrorI                               // flush all operations to the underlying 'writer' without committing
	IncreaseVersion()                            // increment the version of the store
}

// ReadOnlyStoreI defines a Read-Only interface for accessing the blockchain storage including membership and non-membership proofs
type ReadOnlyStoreI interface {
	ProveStoreI
	RStoreI
	RIndexerI
}

// RWStoreI defines the Read/Write interface for basic db CRUD operations
type RWStoreI interface {
	RStoreI
	WStoreI
}

// RWIndexerI defines the Read/Write interface for indexing operations
type RWIndexerI interface {
	WIndexerI
	RIndexerI
}

// WIndexerI defines the write interface for the indexing operations
type WIndexerI interface {
	IndexQC(qc *QuorumCertificate) ErrorI                          // save a quorum certificate by height
	IndexTx(result *TxResult) ErrorI                               // save a tx by hash, height.index, sender, and recipient
	IndexBlock(b *BlockResult) ErrorI                              // save a block by hash and height
	IndexDoubleSigner(address []byte, height uint64) ErrorI        // save a double signer for a height
	IndexCheckpoint(chainId uint64, checkpoint *Checkpoint) ErrorI // save a checkpoint for a committee chain
	DeleteTxsForHeight(height uint64) ErrorI                       // deletes all transactions for a height
	DeleteBlockForHeight(height uint64) ErrorI                     // deletes a block and transaction data for a height
	DeleteQCForHeight(height uint64) ErrorI                        // deletes a certificate for a height
}

// RIndexerI defines the read interface for the indexing operations
type RIndexerI interface {
	GetTxByHash(hash []byte) (*TxResult, ErrorI)                                                  // get the tx by the Transaction hash
	GetTxsByHeight(height uint64, newestToOldest bool, p PageParams) (*Page, ErrorI)              // get Transactions for a height
	GetTxsBySender(address crypto.AddressI, newestToOldest bool, p PageParams) (*Page, ErrorI)    // get Transactions for a sender
	GetTxsByRecipient(address crypto.AddressI, newestToOldest bool, p PageParams) (*Page, ErrorI) // get Transactions for a recipient
	GetBlockByHash(hash []byte) (*BlockResult, ErrorI)                                            // get a block by hash
	GetBlockByHeight(height uint64) (*BlockResult, ErrorI)                                        // get a block by height
	GetBlockHeaderByHeight(height uint64) (*BlockResult, ErrorI)                                  // get a block by height without transactions
	GetBlocks(p PageParams) (*Page, ErrorI)                                                       // get a page of blocks within the page params
	GetQCByHeight(height uint64) (*QuorumCertificate, ErrorI)                                     // get certificate for a height
	GetDoubleSigners() ([]*DoubleSigner, ErrorI)                                                  // all double signers in the indexer
	IsValidDoubleSigner(address []byte, height uint64) (bool, ErrorI)                             // get if the DoubleSigner is already set for a height
	GetCheckpoint(chainId, height uint64) (blockHash HexBytes, err ErrorI)                        // get the checkpoint block hash for a certain committee and height combination
	GetMostRecentCheckpoint(chainId uint64) (checkpoint *Checkpoint, err ErrorI)                  // get the most recent checkpoint for a committee
	GetAllCheckpoints(chainId uint64) (checkpoints []*Checkpoint, err ErrorI)                     // export all checkpoints for a committee
}

// WStoreI defines an interface for basic write operations
type WStoreI interface {
	Set(key, value []byte) ErrorI // set value bytes referenced by key bytes
	Delete(key []byte) ErrorI
}

// WStoreI defines an interface for basic read operations
type RStoreI interface {
	Get(key []byte) ([]byte, ErrorI)               // access value bytes using key bytes
	Iterator(prefix []byte) (IteratorI, ErrorI)    // iterate through the data one KV pair at a time in lexicographical order
	RevIterator(prefix []byte) (IteratorI, ErrorI) // iterate through the date on KV pair at a time in reverse lexicographical order
}

// ProveStoreI defines an interface
type ProveStoreI interface {
	GetProof(key []byte) (proof []*Node, err ErrorI) // Get gets the bytes for a compact merkle proof
	VerifyProof(key, value []byte, validateMembership bool,
		root []byte, proof []*Node) (valid bool, err ErrorI) // VerifyProof validates the merkle proof
}

// IteratorI defines an interface for iterating over key-value pairs in a data store
type IteratorI interface {
	Valid() bool           // if the item the iterator is pointing at is valid
	Next()                 // move to next item
	Key() (key []byte)     // retrieve key
	Value() (value []byte) // retrieve value
	Close()                // close the iterator when done, ensuring proper resource management
}
