package store

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

const (
	latestStatePrefix     = "s/"           // prefix designated for the LatestStateStore where the most recent blobs of state data are held
	historicStatePrefix   = "h/"           // prefix designated for the HistoricalStateStore where the historical blobs of state data are held
	stateCommitmentPrefix = "c/"           // prefix designated for the StateCommitmentStore (immutable, tree DB) built of hashes of state store data
	indexerPrefix         = "i/"           // prefix designated for indexer (transactions, blocks, and quorum certificates)
	stateCommitIDPrefix   = "x/"           // prefix designated for the commit ID (height and state merkle root)
	lastCommitIDPrefix    = "a/"           // prefix designated for the latest commit ID for easy access (latest height and latest state merkle root)
	maxKeyBytes           = 256            // maximum size of a key
	lssVersion            = math.MaxUint64 // the arbitrary version the latest state is written to for optimized queries
)

var _ lib.StoreI = &Store{} // enforce the Store interface

/*
The Store struct is a high-level abstraction layer built on top of a single BadgerDB instance,
providing four main components for managing blockchain-related data.

1. StateStore: This component is responsible for storing the actual blobs of data that represent
   the state. It acts as the primary data storage layer. This store is divided into 'historical'
   partitions and 'latest' data. This separation allows efficient block processing time while
   minimizing storage de-duplication.

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
	version  uint64             // version of the store
	db       *badger.DB         // underlying database
	writer   *badger.WriteBatch // the shared batch writer that allows committing it all at once
	ss       *Txn               // reference to the state store
	sc       *SMT               // reference to the state commitment store
	*Indexer                    // reference to the indexer store
	metrics  *lib.Metrics       // telemetry
	log      lib.LoggerI        // logger
	config   lib.Config         // config
}

// New() creates a new instance of a StoreI either in memory or an actual disk DB
func New(config lib.Config, metrics *lib.Metrics, l lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	if config.StoreConfig.InMemory {
		return NewStoreInMemory(l)
	}
	return NewStore(config, filepath.Join(config.DataDirPath, config.DBName), metrics, l)
}

// NewStore() creates a new instance of a disk DB
func NewStore(config lib.Config, path string, metrics *lib.Metrics, log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	// use badger DB in managed mode to allow efficient versioning
	db, err := badger.OpenManaged(badger.DefaultOptions(path).WithNumVersionsToKeep(math.MaxInt64).WithLoggingLevel(badger.ERROR).
		WithValueThreshold(1024).WithCompression(options.None).WithNumMemtables(16).WithMemTableSize(256 << 20).
		WithNumLevelZeroTables(10).WithNumLevelZeroTablesStall(20).WithBaseTableSize(128 << 20).WithBaseLevelSize(512 << 20).
		WithNumCompactors(runtime.NumCPU()).WithCompactL0OnClose(true).WithBypassLockGuard(true).WithDetectConflicts(false).WithSyncWrites(true),
	)
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(config, db, metrics, log)
}

// NewStoreInMemory() creates a new instance of a mem DB
func NewStoreInMemory(log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(lib.DefaultConfig(), db, nil, log)
}

// NewStoreWithDB() returns a Store object given a DB and a logger
// NOTE: to read the state commit store i.e. for merkle proofs, use NewReadOnly()
func NewStoreWithDB(config lib.Config, db *badger.DB, metrics *lib.Metrics, log lib.LoggerI) (*Store, lib.ErrorI) {
	// get the latest CommitID (height and hash)
	id := getLatestCommitID(db, log)
	// set the version
	nextVersion, version := id.Height+1, id.Height
	// make a reader from the current height and the latest height
	hssReader := db.NewTransactionAt(version, false)
	lssReader := db.NewTransactionAt(lssVersion, false)
	// create a new batch writer for the next version
	// note: version for WriteBatch may be overridden by the setEntryAt(version) code
	writer := db.NewWriteBatchAt(nextVersion)
	// return the store object
	return &Store{
		version: version,
		log:     log,
		db:      db,
		writer:  writer,
		ss:      NewBadgerTxn(lssReader, writer, []byte(latestStatePrefix), true, true, nextVersion),
		Indexer: &Indexer{NewBadgerTxn(hssReader, writer, []byte(indexerPrefix), false, false, nextVersion), config},
		metrics: metrics,
		config:  config,
	}, nil
}

// NewReadOnly() returns a store without a writer - meant for historical read only queries
// CONTRACT: Read only stores cannot be copied or written to
func (s *Store) NewReadOnly(queryVersion uint64) (lib.StoreI, lib.ErrorI) {
	var stateReader *Txn
	// make a reader for the specified version
	hssReader := s.db.NewTransactionAt(queryVersion, false)
	// if the query is for the latest version use the HSS over the LSS
	if s.version == queryVersion {
		lssReader := s.db.NewTransactionAt(lssVersion, false)
		stateReader = NewBadgerTxn(lssReader, nil, []byte(latestStatePrefix), false, false)
	} else {
		stateReader = NewBadgerTxn(hssReader, nil, []byte(historicStatePrefix), false, false)
	}
	// return the store object
	return &Store{
		version: queryVersion,
		log:     s.log,
		db:      s.db,
		ss:      stateReader,
		sc:      NewDefaultSMT(NewBadgerTxn(hssReader, nil, []byte(stateCommitmentPrefix), false, false)),
		Indexer: &Indexer{NewBadgerTxn(hssReader, nil, []byte(indexerPrefix), false, false), s.config},
		metrics: s.metrics,
	}, nil
}

// Copy() make a copy of the store with a new read/write transaction
// this can be useful for having two simultaneous copies of the store
// ex: Mempool state and FSM state
func (s *Store) Copy() (lib.StoreI, lib.ErrorI) {
	nextVersion := s.version + 1
	// create a comparable writer and reader
	writer, reader, lssReader := s.db.NewWriteBatchAt(nextVersion), s.db.NewTransactionAt(s.version, false), s.db.NewTransactionAt(lssVersion, false)
	// return the store object
	return &Store{
		version: s.version,
		log:     s.log,
		db:      s.db,
		writer:  writer,
		ss:      s.ss.Copy(BadgerTxnReader{lssReader, s.ss.prefix}, writer),
		Indexer: &Indexer{s.Indexer.db.Copy(BadgerTxnReader{reader, s.Indexer.db.prefix}, writer), s.config},
		metrics: s.metrics,
	}, nil
}

// Commit() performs a single atomic write of the current state to all stores.
func (s *Store) Commit() (root []byte, err lib.ErrorI) {
	startTime := time.Now()
	// get the root from the sparse merkle tree at the current state
	root, err = s.Root()
	if err != nil {
		return nil, err
	}
	// update the version (height) number
	s.version++
	// set the new CommitID (to the Transaction not the actual DB)
	if err = s.setCommitID(s.version, root); err != nil {
		return nil, err
	}
	// extract the internal metrics from the badger Txn
	size, entries := getSizeAndCountFromBatch(s.writer)
	// commit the in-memory txn to the badger writer
	if e := s.Flush(); e != nil {
		return nil, e
	}
	// finally commit the entire Transaction to the actual DB under the proper version (height) number
	if e := s.writer.Flush(); e != nil {
		return nil, ErrCommitDB(e)
	}
	// update the metrics once complete
	s.metrics.UpdateStoreMetrics(size, entries, time.Time{}, startTime)
	// reset the writer for the next height
	s.Reset()
	// return the root
	return
}

// Flush() writes the current state to the batch writer without committing it.
func (s *Store) Flush() lib.ErrorI {
	if s.sc != nil {
		if e := s.sc.store.(TxnWriterI).Flush(); e != nil {
			return ErrCommitDB(e)
		}
	}
	if e := s.ss.Flush(); e != nil {
		return ErrCommitDB(e)
	}
	if e := s.Indexer.db.Flush(); e != nil {
		return ErrCommitDB(e)
	}
	return nil
}

// Set() sets the value bytes blob in the LatestStateStore and the HistoricalStateStore
// as well as the value hash in the StateCommitStore referenced by the 'key' and hash('key') respectively
func (s *Store) Set(k, v []byte) lib.ErrorI { return s.ss.Set(k, v) }

// Delete() removes the key-value pair from both the LatestStateStore, HistoricalStateStore, and CommitStore
func (s *Store) Delete(k []byte) lib.ErrorI { return s.ss.Delete(k) }

// Get() returns the value bytes blob from the State Store
func (s *Store) Get(key []byte) ([]byte, lib.ErrorI) { return s.ss.Get(key) }

// Iterator() returns an object for scanning the StateStore starting from the provided prefix.
// The iterator allows forward traversal of key-value pairs that match the prefix.
func (s *Store) Iterator(p []byte) (lib.IteratorI, lib.ErrorI) { return s.ss.Iterator(p) }

// RevIterator() returns an object for scanning the StateStore starting from the provided prefix.
// The iterator allows backward traversal of key-value pairs that match the prefix.
func (s *Store) RevIterator(p []byte) (lib.IteratorI, lib.ErrorI) { return s.ss.RevIterator(p) }

// GetProof() uses the StateCommitStore to prove membership and non-membership
func (s *Store) GetProof(key []byte) ([]*lib.Node, lib.ErrorI) { return s.sc.GetMerkleProof(key) }

// VerifyProof() checks the validity of a member or non-member proof from the StateCommitStore
// by verifying the proof against the provided key, value, and proof data.
func (s *Store) VerifyProof(key, value []byte, validateMembership bool, root []byte, proof []*lib.Node) (bool, lib.ErrorI) {
	return s.sc.VerifyProof(key, value, validateMembership, root, proof)
}

// IncreaseVersion increases the version number of the store without committing any data
func (s *Store) IncreaseVersion() { func() { s.version++; s.sc = nil }() }

// Version() returns the current version number of the Store, representing the height or version
// number of the state. This is used to track the versioning of the state data.
func (s *Store) Version() uint64 { return s.version }

// NewTxn() creates and returns a new transaction for the Store, allowing atomic operations
// on the StateStore, StateCommitStore, Indexer, and CommitIDStore.
func (s *Store) NewTxn() lib.StoreI {
	nextVersion := s.version + 1
	return &Store{
		version: s.version,
		log:     s.log,
		db:      s.db,
		writer:  s.writer,
		ss:      NewTxn(s.ss, s.ss, nil, false, true, nextVersion),
		Indexer: &Indexer{NewTxn(s.Indexer.db, s.Indexer.db, nil, false, true, nextVersion), s.config},
		metrics: s.metrics,
	}
}

// DB() returns the underlying BadgerDB instance associated with the Store, providing access
// to the database for direct operations and management.
func (s *Store) DB() *badger.DB { return s.db }

// Root() retrieves the root hash of the StateCommitStore, representing the current root of the
// Sparse Merkle Tree. This hash is used for verifying the integrity and consistency of the state.
func (s *Store) Root() (root []byte, err lib.ErrorI) {
	// if smt not cached
	if s.sc == nil {
		nextVersion := s.version + 1
		// set up the state commit store
		s.sc = NewDefaultSMT(NewTxn(s.ss.reader.(BadgerTxnReader), s.writer, []byte(stateCommitIDPrefix), false, false, nextVersion))
		// commit the SMT directly using the txn ops
		if err = s.sc.CommitParallel(s.ss.txn.ops); err != nil {
			return nil, err
		}
	}
	// return the root
	return s.sc.Root(), nil
}

// Reset() discard and re-sets the stores writer
func (s *Store) Reset() {
	// create new transactions first before discarding old ones
	newReader := s.db.NewTransactionAt(s.version, false)
	newLSSReader := s.db.NewTransactionAt(lssVersion, false)
	// create a new batch writer for the next version as the version cannot
	// be set at the commit time
	nextVersion := s.version + 1
	newWriter := s.db.NewWriteBatchAt(nextVersion)
	// create all new transaction-dependent objects
	newLSS := NewBadgerTxn(newLSSReader, newWriter, []byte(latestStatePrefix), true, true, nextVersion)
	newIndexer := NewBadgerTxn(newReader, newWriter, []byte(indexerPrefix), false, false, nextVersion)
	// only after creating all new objects, discard old transactions
	s.Discard()
	// update all references
	s.writer = newWriter
	s.ss = newLSS
	s.Indexer.setDB(newIndexer)
}

// Discard() closes the reader and writer
func (s *Store) Discard() {
	s.ss.reader.Discard()
	s.sc = nil
	s.Indexer.db.reader.Discard()
	if s.writer != nil {
		s.writer.Cancel()
	}
}

// Close() discards the writer and closes the database connection
func (s *Store) Close() lib.ErrorI {
	s.Discard()
	if err := s.db.Close(); err != nil {
		return ErrCloseDB(s.db.Close())
	}
	return nil
}

// commitIDKey() returns the key for the commitID at a specific version
func (s *Store) commitIDKey(version uint64) []byte {
	return fmt.Appendf(nil, "%s/%d", stateCommitIDPrefix, version)
}

// getCommitID() retrieves the CommitID value for the specified version from the database
func (s *Store) getCommitID(version uint64) (id lib.CommitID, err lib.ErrorI) {
	var bz []byte
	bz, err = NewTxn(s.Indexer.db.reader, nil, nil, false, false).Get(s.commitIDKey(version))
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
	w := NewTxn(s.Indexer.db.reader, s.writer, nil, false, false, version)
	value, err := lib.Marshal(&lib.CommitID{Height: version, Root: root})
	if err != nil {
		return err
	}
	if err = w.Set([]byte(lastCommitIDPrefix), value); err != nil {
		return err
	}
	if err = w.Set(s.commitIDKey(version), value); err != nil {
		return err
	}
	if e := w.Flush(); e != nil {
		return ErrCommitDB(e)
	}
	return nil
}

// getLatestCommitID() retrieves the latest CommitID from the database
func getLatestCommitID(db *badger.DB, log lib.LoggerI) (id *lib.CommitID) {
	reader := db.NewTransactionAt(math.MaxUint64, false)
	tx := NewBadgerTxn(reader, nil, nil, false, false, 0)
	defer reader.Discard()
	bz, err := tx.Get([]byte(lastCommitIDPrefix))
	if err != nil {
		log.Fatalf("getLatestCommitID() failed with err: %s", err.Error())
	}
	id = new(lib.CommitID)
	if err = lib.Unmarshal(bz, id); err != nil {
		log.Fatalf("unmarshalCommitID() failed with err: %s", err.Error())
	}
	return
}
