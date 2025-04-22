package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync/atomic"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
)

const (
	latestStatePrefix     = "s/"          // prefix designated for the LatestStateStore where the most recent blobs of state data are held
	historicStatePrefix   = "h/"          // prefix designated for the HistoricalStateStore where the historical blobs of state data are held
	stateCommitmentPrefix = "c/"          // prefix designated for the StateCommitmentStore (immutable, tree DB) built of hashes of state store data
	indexerPrefix         = "i/"          // prefix designated for indexer (transactions, blocks, and quorum certificates)
	stateCommitIDPrefix   = "x/"          // prefix designated for the commit ID (height and state merkle root)
	lastCommitIDPrefix    = "a/"          // prefix designated for the latest commit ID for easy access (latest height and latest state merkle root)
	partitionExistsKey    = "e/"          // to check if partition exists
	partitionFrequency    = uint64(10000) // blocks
	maxKeyBytes           = 256           // maximum size of a key
)

var _ lib.StoreI = &Store{} // enforce the Store interface

/*
The Store struct is a high-level abstraction layer built on top of a single BadgerDB instance,
providing four main components for managing blockchain-related data.

1. StateStore: This component is responsible for storing the actual blobs of data that represent
   the state. It acts as the primary data storage layer. This store is divided into 'historical'
   partitions and 'latest' data. This separation allows efficient iteration, fast snapshot access,
   and safe pruning of older state without impacting current performance.

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
	version             uint64       // version of the store
	root                []byte       // root associated with the CommitID at this version
	db                  *badger.DB   // underlying database
	writer              *badger.Txn  // the batch writer that allows committing it all at once
	lss                 *TxnWrapper  // reference to the 'latest' state store
	hss                 *TxnWrapper  // references the 'historical' state store
	sc                  *SMT         // reference to the state commitment store
	*Indexer                         // reference to the indexer store
	useHistorical       bool         // signals to use the historical state store for query
	isGarbageCollecting atomic.Bool  // protect garbage collector (only 1 at a time)
	metrics             *lib.Metrics // telemetry
	log                 lib.LoggerI  // logger
}

// New() creates a new instance of a StoreI either in memory or an actual disk DB
func New(config lib.Config, metrics *lib.Metrics, l lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	if config.StoreConfig.InMemory {
		return NewStoreInMemory(l)
	}
	return NewStore(filepath.Join(config.DataDirPath, config.DBName), metrics, l)
}

// NewStore() creates a new instance of a disk DB
func NewStore(path string, metrics *lib.Metrics, log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	// use badger DB in managed mode to allow easy versioning
	// memTableSize is set to 1.28GB (max) to allow 128MB (10%) of writes in a
	// single batch. It is seemingly unknown why the 10% limit is set
	// https://discuss.dgraph.io/t/discussion-badgerdb-should-offer-arbitrarily-sized-atomic-transactions/8736
	db, err := badger.OpenManaged(badger.DefaultOptions(path).WithNumVersionsToKeep(math.MaxInt64).
		WithLoggingLevel(badger.ERROR).WithMemTableSize(int64(1*units.GB + 280*units.MB)))
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(db, metrics, log, true)
}

// NewStoreInMemory() creates a new instance of a mem DB
func NewStoreInMemory(log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(db, nil, log, true)
}

// NewStoreWithDB() returns a Store object given a DB and a logger
func NewStoreWithDB(db *badger.DB, metrics *lib.Metrics, log lib.LoggerI, write bool) (*Store, lib.ErrorI) {
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
		lss:     NewTxnWrapper(writer, log, []byte(latestStatePrefix)),
		hss:     NewTxnWrapper(writer, log, historicalPrefix(id.Height)),
		sc:      NewDefaultSMT(NewTxnWrapper(writer, log, []byte(stateCommitmentPrefix))),
		Indexer: &Indexer{NewTxnWrapper(writer, log, []byte(indexerPrefix))},
		metrics: metrics,
		root:    id.Root,
	}, nil
}

// NewReadOnly() returns a store without a writer - meant for historical read only queries
func (s *Store) NewReadOnly(queryVersion uint64) (lib.StoreI, lib.ErrorI) {
	// create a variable to signal if the historical state store should be utilized
	var useHistorical bool
	// if the query version is older than the partition frequency
	if queryVersion+1 < partitionHeight(s.version) {
		useHistorical = true
	}
	// make a reader for the specified version
	reader := s.db.NewTransactionAt(queryVersion, false)
	// return the store object
	return &Store{
		version:       queryVersion,
		log:           s.log,
		db:            s.db,
		writer:        reader,
		lss:           NewTxnWrapper(reader, s.log, []byte(latestStatePrefix)),
		hss:           NewTxnWrapper(reader, s.log, historicalPrefix(queryVersion)),
		sc:            NewDefaultSMT(NewTxnWrapper(reader, s.log, []byte(stateCommitmentPrefix))),
		Indexer:       &Indexer{NewTxnWrapper(reader, s.log, []byte(indexerPrefix))},
		useHistorical: useHistorical,
		metrics:       s.metrics,
		root:          bytes.Clone(s.root),
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
		lss:     NewTxnWrapper(writer, s.log, []byte(latestStatePrefix)),
		hss:     NewTxnWrapper(writer, s.log, historicalPrefix(s.version)),
		sc:      NewDefaultSMT(NewTxnWrapper(writer, s.log, []byte(stateCommitmentPrefix))),
		Indexer: &Indexer{NewTxnWrapper(writer, s.log, []byte(indexerPrefix))},
		metrics: s.metrics,
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

// PARTITIONING CODE BELOW

// ShouldPartition() determines if it is time to partition
func (s *Store) ShouldPartition() (timeToPartition bool) {
	// check if it's time to partition (1001, 2001, 3001...)
	if (s.version-partitionHeight(s.version))%(partitionFrequency/10) != 1 {
		return false
	}
	// get the partition exists value from the store at a particular historical partition prefix
	value, err := s.hss.Get([]byte(partitionExistsKey))
	if err != nil {
		s.log.Errorf("ShouldPartition() failed with err: %s", err.Error())
		return false
	}
	// check if the value is set <key is the value>
	timeToPartition = !bytes.Equal(value, []byte(partitionExistsKey))
	// log the result
	if !timeToPartition {
		s.log.Debug("Not partitioning, partition key already exists")
	} else {
		s.log.Debug("Should partition! No partition key exists")
	}
	return
}

// Partition()
//  1. SNAPSHOT: for each key in the state store, copy them in the new historical partition to ensure
//     each partition has a complete version of the state at the border -- allowing older partitions to
//     safely be pruned
//  2. PRUNE: Drop all versions in the LSS older than the latest keys @ partition height
func (s *Store) Partition() {
	// create a copy of the store for multi-thread safety
	sCopy, err := s.Copy()
	if err != nil {
		s.log.Errorf(err.Error())
		return
	}
	// cast to a type Store reference for lower level functionality
	sc := sCopy.(*Store)
	// log the beginning of the process
	sc.log.Info("Started partition process ✂️")
	if err = func() lib.ErrorI {
		// calculate the partition height
		snapshotHeight := partitionHeight(sc.Version())
		// create a reader at the partition height
		reader := sc.db.NewTransactionAt(snapshotHeight, false)
		defer reader.Discard()
		// create a managed batch to do the 'writing'
		writer := sc.db.NewWriteBatchAt(snapshotHeight)
		defer writer.Cancel()
		// generate the historical partition prefix
		partitionPrefix := historicalPrefix(snapshotHeight)
		// get the latest store in a transaction wrapper
		lss := NewTxnWrapper(reader, sc.log, []byte(latestStatePrefix))
		// create an iterator that traverses the entire latest state store at the partition height
		iterator, er := lss.ArchiveIterator(nil)
		if er != nil {
			return er
		}
		// convert the iterator to a type Iterator for lower level functionality
		it := iterator.(*Iterator)
		defer it.Close()
		// set a signal the partition was successfully created <only written if batch succeeds>
		if e := writer.SetEntryAt(&badger.Entry{Key: lib.Append(partitionPrefix, []byte(partitionExistsKey)), Value: []byte(partitionExistsKey)}, snapshotHeight); e != nil {
			return ErrSetEntry(e)
		}
		// create a variable to de-duplicate the calls
		deDuplicator := lib.NewDeDuplicator[string]()
		// for each key in the state at the partition height
		for ; it.Valid(); it.Next() {
			// get the key from the iterator item
			k, v := it.Key(), it.Value()
			// skip items that are already marked for Garbage Collection
			if deDuplicator.Found(lib.BytesToString(k)) {
				continue
			}
			// if the item is 'deleted'
			if it.Deleted() {
				// set the item as deleted at the partition height and discard earlier versions
				if e := writer.SetEntryAt(newEntry(lib.Append([]byte(latestStatePrefix), k), nil, badgerDeleteBit|badgerDiscardEarlierVersions), snapshotHeight); e != nil {
					return ErrSetEntry(e)
				}
			} else {
				// set item in the historical partition
				if e := writer.SetEntryAt(newEntry(lib.Append(partitionPrefix, k), v, badgerNoDiscardBit), snapshotHeight); e != nil {
					return ErrSetEntry(e)
				}
				// re-write the latest version with the 'discard' flag set
				if e := writer.SetEntryAt(newEntry(lib.Append([]byte(latestStatePrefix), k), v, badgerDiscardEarlierVersions), snapshotHeight); e != nil {
					return ErrSetEntry(e)
				}
			}
		}
		// commit the batch
		if e := writer.Flush(); e != nil {
			return ErrFlushBatch(e)
		}
		// if the partition height is past the partition frequency, set the discardTs at the partition height-1
		if snapshotHeight > partitionFrequency {
			sc.db.SetDiscardTs(snapshotHeight - 2)
		}
		// if the GC isn't already running
		if !s.isGarbageCollecting.Swap(true) {
			// unset isGarbageCollecting once complete
			defer s.isGarbageCollecting.Store(false)
			// trigger garbage collector to prune keys
			if gcErr := sc.db.RunValueLogGC(badgerGCRatio); gcErr != nil {
				if errors.Is(gcErr, badger.ErrNoRewrite) {
					sc.log.Debugf("%v - this is normal", gcErr)
					// don't return an error here - this is an expected condition
				} else {
					return ErrGarbageCollectDB(gcErr)
				}
			}
		}
		// exit
		return nil
	}(); err != nil {
		if err != badger.ErrNoRewrite {
			sc.log.Warn(err.Error()) // nothing collected
		} else {
			sc.log.Errorf("Partitioning failed with error: %s", err.Error())
		}
	}
	sc.log.Info("Partitioning complete ✅")
}

// Get() returns the value bytes blob from the State Store
func (s *Store) Get(key []byte) ([]byte, lib.ErrorI) {
	// if reading from a historical partition
	if s.useHistorical {
		return s.hss.Get(key)
	}
	// if reading from the latest
	return s.lss.Get(key)
}

// Set() sets the value bytes blob in the LatestStateStore and the HistoricalStateStore
// as well as the value hash in the StateCommitStore referenced by the 'key' and hash('key') respectively
func (s *Store) Set(k, v []byte) lib.ErrorI {
	// set in the state store @ latest
	if err := s.lss.Set(k, v); err != nil {
		return err
	}
	// set in the state store @ historical partition
	if err := s.hss.SetNonPruneable(k, v); err != nil {
		return err
	}
	// set in the state commit store
	return s.sc.Set(k, v)
}

// Delete() removes the key-value pair from both the LatestStateStore, HistoricalStateStore, and CommitStore
func (s *Store) Delete(k []byte) lib.ErrorI {
	// delete from the state store @ latest
	if err := s.lss.Delete(k); err != nil {
		return err
	}
	// delete from the state store @ historical partition
	if err := s.hss.DeleteNonPrunable(k); err != nil {
		return err
	}
	// delete from the state commit store
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
func (s *Store) Iterator(p []byte) (lib.IteratorI, lib.ErrorI) {
	// if reading from a historical partition
	if s.useHistorical {
		return s.hss.Iterator(p)
	}
	// if reading from latest
	return s.lss.Iterator(p)
}

// RevIterator() returns an object for scanning the StateStore starting from the provided prefix.
// The iterator allows backward traversal of key-value pairs that match the prefix.
func (s *Store) RevIterator(p []byte) (lib.IteratorI, lib.ErrorI) {
	// if reading from a historical partition
	if s.useHistorical {
		return s.hss.RevIterator(p)
	}
	// if reading from latest
	return s.lss.RevIterator(p)
}

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

// Reset() discard and re-sets the stores writer
func (s *Store) Reset() {
	s.resetWriter()
}

// Discard() closes the writer
func (s *Store) Discard() {
	s.writer.Discard()
}

// Close() discards the writer and closes the database connection
func (s *Store) Close() lib.ErrorI {
	s.Discard()
	if err := s.db.Close(); err != nil {
		return ErrCloseDB(s.db.Close())
	}
	return nil
}

// resetWriter() closes the writer, and creates a new writer, and sets the writer to the 3 main abstractions
func (s *Store) resetWriter() {
	s.writer.Discard()
	s.writer = s.db.NewTransactionAt(s.version, true)
	s.lss = NewTxnWrapper(s.writer, s.log, []byte(latestStatePrefix))
	s.hss = NewTxnWrapper(s.writer, s.log, historicalPrefix(s.version))
	s.sc = NewDefaultSMT(NewTxnWrapper(s.writer, s.log, []byte(stateCommitmentPrefix)))
	s.Indexer.setDB(NewTxnWrapper(s.writer, s.log, []byte(indexerPrefix)))
}

// commitIDKey() returns the key for the commitID at a specific version
func (s *Store) commitIDKey(version uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", stateCommitIDPrefix, version))
}

// getCommitID() retrieves the CommitID value for the specified version from the database
func (s *Store) getCommitID(version uint64) (id lib.CommitID, err lib.ErrorI) {
	var bz []byte
	bz, err = NewTxnWrapper(s.writer, s.log, nil).Get(s.commitIDKey(version))
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
	w := NewTxnWrapper(s.writer, s.log, nil)
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

// historicalPrefix() calculates the prefix for a particular historical partition given the block height
func historicalPrefix(height uint64) []byte {
	return append([]byte(historicStatePrefix), binary.BigEndian.AppendUint64(nil, partitionHeight(height))...)
}

// partitionHeight() returns the height of the partition given some height
// (ex. 45000 -> 40000 and 57550 -> 50000)
func partitionHeight(height uint64) uint64 {
	if height < partitionFrequency {
		return 1 // not 0
	}
	return (height / partitionFrequency) * partitionFrequency
}

// getLatestCommitID() retrieves the latest CommitID from the database
func getLatestCommitID(db *badger.DB, log lib.LoggerI) (id *lib.CommitID) {
	tx := NewTxnWrapper(db.NewTransactionAt(math.MaxUint64, false), log, nil)
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
