package store

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
)

const (
	stateStorePrefix      = "s/" // prefix designated for the StateStore where the actual blobs of state data are held
	stateCommitmentPrefix = "c/" // prefix designated for the StateCommitmentStore (immutable, tree DB) built of hashes of state store data
	indexerPrefix         = "i/" // prefix designated for indexer (transactions, blocks, and quorum certificates)
	stateCommitIDPrefix   = "x/" // prefix designated for the commit ID (height and state merkle root)
	lastCommitIDPrefix    = "a/" // prefix designated for the latest commit ID for easy access (latest height and latest state merkle root)
	maxKeyBytes           = 256  // maximum size of a key
)

var _ lib.StoreI = &Store{} // enforce the Store interface

/*
The Store struct is a high-level abstraction layer built on top of a single BadgerDB instance,
providing four main components for managing blockchain-related data.

1. StateStore: This component is responsible for storing the actual blobs of data that represent
   the state. It acts as the primary data storage layer.

2. StateCommitStore: This component maintains a Sparse Merkle Tree structure, mapping keys
   (hashes) to their corresponding data hashes. It is optimized for blockchain operations,
   allowing efficient proof of existence within the tree and enabling the creation of a single
   'root' hash. This root hash represents the entire state, facilitating easy verification by
   other nodes to ensure consistency between their StateHash and the peer StateHash.

3. Indexer: This component indexes critical blockchain elements, including Quorum Certificates
   by height, Blocks by both height and hash, and Transactions by height.index, hash, sender,
   and recipient. The indexing allows for efficient querying and retrieval of these elements,
   which are essential for blockchain operation.

4. CommitIDStore: This is a smaller abstraction that isolates the 'CommitID' structures, which
   consist of two fields: Version, representing the height or version number, and Root, the root
   hash of the StateCommitStore corresponding to that version. This separation aids in managing
   the state versioning process.

The Store leverages BadgerDB in Managed Mode to maintain historical versions of the state,
allowing for time-travel operations and historical state queries. It uses BadgerDB Transactions
to ensure that all writes to the StateStore, StateCommitStore, Indexer, and CommitIDStore are
performed atomically in a single commit operation per height. Additionally, the Store uses
lexicographically ordered prefix keys to facilitate easy and efficient iteration over stored data.
*/

type Store struct {
	version  uint64      // version of the store
	root     []byte      // root associated with the CommitID at this version
	db       *badger.DB  // underlying database
	writer   *badger.Txn // the batch writer that allows committing it all at once
	ss       *TxnWrapper // reference to the state store
	sc       *SMT        // reference to the state commitment store
	*Indexer             // reference to the indexer store
	log      lib.LoggerI // logger
}

// New() creates a new instance of a StoreI either in memory or an actual disk DB
func New(config lib.Config, l lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	if config.StoreConfig.InMemory {
		return NewStoreInMemory(l)
	}
	return NewStore(filepath.Join(config.DataDirPath, config.DBName), l)
}

// NewStore() creates a new instance of a disk DB
func NewStore(path string, log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	// use badger DB in managed mode to allow easy versioning
	// memTableSize is set to 1.28GB (max) to allow 128MB (10%) of writes in a
	// single batch. It is seemingly unknown why the 10% limit is set
	// https://discuss.dgraph.io/t/discussion-badgerdb-should-offer-arbitrarily-sized-atomic-transactions/8736
	db, err := badger.OpenManaged(badger.DefaultOptions(path).WithNumVersionsToKeep(math.MaxInt).
		WithLoggingLevel(badger.ERROR).WithMemTableSize(int64(1*units.GB + 280*units.MB)))
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(db, log, true)
}

// NewStoreInMemory() creates a new instance of a mem DB
func NewStoreInMemory(log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(db, log, true)
}

// NewStoreWithDB() returns a Store object given a DB and a logger
func NewStoreWithDB(db *badger.DB, log lib.LoggerI, write bool) (*Store, lib.ErrorI) {
	// get the latest CommitID (height and hash)
	id := getLatestCommitID(db, log)
	// make a writable tx that reads from the last height
	writer := db.NewTransactionAt(id.Height, write)
	// return the store object
	return &Store{
		version: id.Height,
		log:     log,
		db:      db,
		writer:  writer,
		ss:      NewTxnWrapper(writer, log, stateStorePrefix),
		sc:      NewDefaultSMT(NewTxnWrapper(writer, log, stateCommitmentPrefix)),
		Indexer: &Indexer{NewTxnWrapper(writer, log, indexerPrefix)},
		root:    id.Root,
	}, nil
}

// NewReadOnly() returns a store without a writer - meant for historical read only queries
func (s *Store) NewReadOnly(version uint64) (lib.StoreI, lib.ErrorI) {
	// make a reader for the specified version
	reader := s.db.NewTransactionAt(version, false)
	// return the store object
	return &Store{
		version: version,
		log:     s.log,
		db:      s.db,
		writer:  reader,
		ss:      NewTxnWrapper(reader, s.log, stateStorePrefix),
		sc:      NewDefaultSMT(NewTxnWrapper(reader, s.log, stateCommitmentPrefix)),
		Indexer: &Indexer{NewTxnWrapper(reader, s.log, indexerPrefix)},
		root:    bytes.Clone(s.root),
	}, nil
}

// Copy() make a copy of the store with a new read/write transaction
// this can be useful for having two simultaneous copies of the store
// ex: Mempool state and FSM state
func (s *Store) Copy() (lib.StoreI, lib.ErrorI) {
	writer := s.db.NewTransactionAt(s.version, true)
	return &Store{
		version: s.version,
		log:     s.log,
		db:      s.db,
		writer:  writer,
		ss:      NewTxnWrapper(writer, s.log, stateStorePrefix),
		sc:      NewDefaultSMT(NewTxnWrapper(writer, s.log, stateCommitmentPrefix)),
		Indexer: &Indexer{NewTxnWrapper(writer, s.log, indexerPrefix)},
		root:    bytes.Clone(s.root),
	}, nil
}

// Commit() performs a single atomic write of the current state to all stores.
func (s *Store) Commit() (root []byte, err lib.ErrorI) {
	// update the version (height) number
	s.version++
	// get the root from the sparse merkle tree at the current state
	s.root = s.sc.Root()
	// set the new CommitID (to the Transaction not the actual DB)
	if err = s.setCommitID(s.version, s.root); err != nil {
		return nil, err
	}
	// finally commit the entire Transaction to the actual DB under the proper version (height) number
	if e := s.writer.CommitAt(s.version, nil); e != nil {
		return nil, ErrCommitDB(e)
	}
	// reset the writer for the next height
	s.resetWriter()
	// return the root
	return bytes.Clone(s.root), nil
}

// Get() returns the value bytes blob from the State Store
func (s *Store) Get(key []byte) ([]byte, lib.ErrorI) { return s.ss.Get(key) }

// Set() sets the value bytes blob in the StateStore and the value hash in the StateCommitStore
// referenced by the 'key' and hash('key') respectively
func (s *Store) Set(k, v []byte) lib.ErrorI {
	if err := s.ss.Set(k, v); err != nil {
		return err
	}
	return s.sc.Set(k, v)
}

// Delete() removes the key-value pair from both the State and CommitStore
func (s *Store) Delete(k []byte) lib.ErrorI {
	if err := s.ss.Delete(k); err != nil {
		return err
	}
	return s.sc.Delete(k)
}

// GetProof() uses the StateCommitStore to prove membership and non-membership
func (s *Store) GetProof(key []byte) ([]*lib.Node, lib.ErrorI) {
	return s.sc.GetMerkleProof(key)
}

// VerifyProof() checks the validity of a member or non-member proof from the StateCommitStore
// by verifying the proof against the provided key, value, and proof data.
func (s *Store) VerifyProof(key, value []byte, validateMembership bool, root []byte, proof []*lib.Node) (bool, lib.ErrorI) {
	return s.sc.VerifyProof(key, value, validateMembership, root, proof)
}

// Iterator() returns an object for scanning the StateStore starting from the provided prefix.
// The iterator allows forward traversal of key-value pairs that match the prefix.
func (s *Store) Iterator(p []byte) (lib.IteratorI, lib.ErrorI) { return s.ss.Iterator(p) }

// RevIterator() returns an object for scanning the StateStore starting from the provided prefix.
// The iterator allows backward traversal of key-value pairs that match the prefix.
func (s *Store) RevIterator(p []byte) (lib.IteratorI, lib.ErrorI) { return s.ss.RevIterator(p) }

// Version() returns the current version number of the Store, representing the height or version
// number of the state. This is used to track the versioning of the state data.
func (s *Store) Version() uint64 { return s.version }

// NewTxn() creates and returns a new transaction for the Store, allowing atomic operations
// on the StateStore, StateCommitStore, Indexer, and CommitIDStore.
func (s *Store) NewTxn() lib.StoreTxnI { return NewTxn(s) }

// DB() returns the underlying BadgerDB instance associated with the Store, providing access
// to the database for direct operations and management.
func (s *Store) DB() *badger.DB { return s.db }

// Root() retrieves the root hash of the StateCommitStore, representing the current root of the
// Sparse Merkle Tree. This hash is used for verifying the integrity and consistency of the state.
func (s *Store) Root() (root []byte, err lib.ErrorI) { return s.sc.Root(), nil }

func (s *Store) Reset() {
	s.resetWriter()
}

// Discard() closes the writer
func (s *Store) Discard() {
	s.writer.Discard()
}

// resetWriter() closes the writer, and creates a new writer, and sets the writer to the 3 main abstractions
func (s *Store) resetWriter() {
	s.writer.Discard()
	s.writer = s.db.NewTransactionAt(s.version, true)
	s.ss.setDB(s.writer)
	s.sc = NewDefaultSMT(NewTxnWrapper(s.writer, s.log, stateCommitmentPrefix))
	s.Indexer.setDB(NewTxnWrapper(s.writer, s.log, indexerPrefix))
}

// commitIDKey() returns the key for the commitID at a specific version
func (s *Store) commitIDKey(version uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", stateCommitIDPrefix, version))
}

// getCommitID() retrieves the CommitID value for the specified version from the database
func (s *Store) getCommitID(version uint64) (id lib.CommitID, err lib.ErrorI) {
	var bz []byte
	bz, err = NewTxnWrapper(s.writer, s.log, "").Get(s.commitIDKey(version))
	if err != nil {
		return
	}
	if err = lib.Unmarshal(bz, &id); err != nil {
		return
	}
	return
}

// setCommitID() stores the CommitID for the specified version and root in the database
func (s *Store) setCommitID(version uint64, root []byte) lib.ErrorI {
	w := NewTxnWrapper(s.writer, s.log, "")
	value, err := lib.Marshal(&lib.CommitID{
		Height: version,
		Root:   root,
	})
	if err != nil {
		return err
	}
	if err = w.Set([]byte(lastCommitIDPrefix), value); err != nil {
		return err
	}
	k := s.commitIDKey(version)
	return w.Set(k, value)
}

// getLatestCommitID() retrieves the latest CommitID from the database
func getLatestCommitID(db *badger.DB, log lib.LoggerI) (id *lib.CommitID) {
	tx := NewTxnWrapper(db.NewTransactionAt(math.MaxUint64, false), log, "")
	defer tx.Close()
	id = new(lib.CommitID)
	bz, err := tx.Get([]byte(lastCommitIDPrefix))
	if err != nil {
		log.Fatalf("getLatestCommitID() failed with err: %s", err.Error())
	}
	if err = lib.Unmarshal(bz, id); err != nil {
		log.Fatalf("unmarshalCommitID() failed with err: %s", err.Error())
	}
	return
}

// close() discards the writer and closes the database connection
func (s *Store) Close() lib.ErrorI {
	s.Discard()
	if err := s.db.Close(); err != nil {
		return ErrCloseDB(s.db.Close())
	}
	return nil
}

// UnsafeRollback() allows a rollback of the database to a previous height
// Since blockchains are immutable, this should only be used in testing or
// crisis situations
func (s *Store) UnsafeRollback(rollbackToHeight uint64) {
	// TODO test and fix this functionality
	for i := s.Version(); i > rollbackToHeight; i-- {
		s.deleteAllKeysAtVersion(i)
	}
	s.version = rollbackToHeight
	s.resetWriter()
}

// Prune() removes older heights of the blockchain database for a 'lighter weight node'
// NOTE: this is not recommended and at-least breaks peer syncing from genesis and could
// be dangerous for nested-chains syncing from the root-chain
func (s *Store) Prune(highestPruneHeight uint64) {
	// TODO incomplete, pruning breaks the initializeTree design which attempts to write
	//   on a read only store. Need to ensure that if pruned - don't accept historical
	//   queries beyond the prune height. Also consider not starting at (0) but start at
	//   first non-pruned height
	for i := uint64(0); i <= highestPruneHeight; i++ {
		s.deleteAllKeysAtVersion(i)
	}
	s.resetWriter()
}

// deleteAllKeysAtVersion() deletes all items at a certain height of the blockchain
func (s *Store) deleteAllKeysAtVersion(version uint64) {
	tx := s.db.NewTransactionAt(version, false)
	// only get the keys (no values)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	// iterate through all the keys
	it := tx.NewIterator(opts)
	// save the keys to delete
	keysToDelete := make([][]byte, 0)
	for it.Rewind(); it.Valid(); it.Next() {
		keysToDelete = append(keysToDelete, it.Item().KeyCopy(nil))
	}
	it.Close()
	// discard the read only tx
	tx.Discard()
	// delete in a new transaction
	tx = s.db.NewTransactionAt(version, true)
	for _, k := range keysToDelete {
		// delete each key
		if err := tx.Delete(k); err != nil {
			panic(err)
		}
	}
	// commit at the specific version
	if err := tx.CommitAt(version, nil); err != nil {
		panic(err)
	}
	// get the root from the sparse merkle tree at the current state
	s.root = s.sc.Root()
	// set the new CommitID (to the Transaction not the actual DB)
	if err := s.setCommitID(s.version, s.root); err != nil {
		panic(err)
	}
}
