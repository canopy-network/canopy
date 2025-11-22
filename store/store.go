package store

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/sstable"
	"github.com/cockroachdb/pebble/v2/vfs"
)

const (
	maxKeyBytes = 256            // maximum size of a key
	lssVersion  = math.MaxUint64 // the arbitrary version the latest state is written to for optimized queries
)

var (
	latestStatePrefix     = lib.JoinLenPrefix([]byte("s/")) // prefix designated for the LatestStateStore where the most recent blobs of state data are held
	historicStatePrefix   = lib.JoinLenPrefix([]byte("h/")) // prefix designated for the HistoricalStateStore where the historical blobs of state data are held
	stateCommitmentPrefix = lib.JoinLenPrefix([]byte("c/")) // prefix designated for the StateCommitmentStore (immutable, tree DB) built of hashes of state store data
	indexerPrefix         = lib.JoinLenPrefix([]byte("i/")) // prefix designated for indexer (transactions, blocks, and quorum certificates)
	stateCommitIDPrefix   = lib.JoinLenPrefix([]byte("x/")) // prefix designated for the commit ID (height and state merkle root)
	lastCommitIDPrefix    = lib.JoinLenPrefix([]byte("a/")) // prefix designated for the latest commit ID for easy access (latest height and latest state merkle root)

	_ lib.StoreI = &Store{} // enforce the Store interface
)

/*
The Store struct is a high-level abstraction layer built on top of a single PebbleDB instance,
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

The store package contains its own multiversion concurrency control system where all the keys are
managed. PebbleDB is used on top of that to ensure that all writes to the StateStore, StateCommitStore,
Indexer, and CommitIDStore are performed atomically in a single commit operation per height.
Additionally, the Store uses lexicographically ordered prefix keys to facilitate easy and efficient
iteration over stored data.
*/

type Store struct {
	version    uint64        // version of the store
	db         *pebble.DB    // underlying database
	writer     *pebble.Batch // the shared batch writer that allows committing it all at once
	ss         *Txn          // reference to the state store
	sc         *SMT          // reference to the state commitment store
	*Indexer                 // reference to the indexer store
	metrics    *lib.Metrics  // telemetry
	log        lib.LoggerI   // logger
	config     lib.Config    // config
	mu         *sync.Mutex   // mutex for concurrent commits
	compaction atomic.Bool   // atomic boolean for compaction status
	isTxn      bool          // flag indicating if the store is in transaction mode
}

// New() creates a new instance of a StoreI either in memory or an actual disk DB
func New(config lib.Config, metrics *lib.Metrics, l lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	if config.StoreConfig.InMemory {
		return NewStoreInMemory(l)
	}
	return NewStore(config, filepath.Join(config.DataDirPath, config.DBName), metrics, l)
}

// NewStore() creates a new instance of a disk DB˙
func NewStore(config lib.Config, path string, metrics *lib.Metrics, log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	cache := pebble.NewCache(256 << 20) // 256 MB cache
	defer cache.Unref()
	lvl := pebble.LevelOptions{
		BlockSize:      64 << 10, // 64 KB data blocks
		IndexBlockSize: 32 << 10, // 32 KB index blocks
		Compression: func() *sstable.CompressionProfile {
			return sstable.ZstdCompression // biggest compression at the expense of more CPU resources
		},
	}
	db, err := pebble.Open(path, &pebble.Options{
		MemTableSize:          64 << 20,                    // larger memtable to reduce flushes
		L0CompactionThreshold: 6,                           // keep L0 small to avoid read amplification
		L0StopWritesThreshold: 12,                          // stop writes when L0 reaches this size
		MaxOpenFiles:          5000,                        // more file handles
		Cache:                 cache,                       // block cache
		FormatMajorVersion:    pebble.FormatColumnarBlocks, // current format version
		LBaseMaxBytes:         512 << 20,                   // [512MB] maximum size of the LBase level
		Levels: [7]pebble.LevelOptions{
			lvl, lvl, lvl, lvl, lvl, lvl, lvl, // apply same scan-optimized blocks across all levels
		},
		// allows for smaller blocks and more block properties so versions can be more granular
		TargetFileSizes: [7]int64{
			32 << 20,  // L0: 32MB
			64 << 20,  // L1: 64MB
			128 << 20, // L2: 128MB
			128 << 20, // L3: 128MB
			128 << 20, // L4: 128MB
			128 << 20, // L5: 128MB
			128 << 20, // L6: 128MB
		},
		Logger:                  log, // Use project's logger
		BlockPropertyCollectors: []func() pebble.BlockPropertyCollector{newVersionedPropertyCollector},
		// [EXPERIMENTAL] should improve throughput by reducing WAL syncs I/O but may lead to data loss
		// on the worst case (i.e sudden program crash)
		WALMinSyncInterval: func() time.Duration {
			return time.Millisecond * 2
		},
	})
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(config, db, metrics, log)
}

// NewStoreInMemory() creates a new instance of a mem DB
func NewStoreInMemory(log lib.LoggerI) (lib.StoreI, lib.ErrorI) {
	db, err := pebble.Open("", &pebble.Options{
		FS:                    vfs.NewMem(),                // memory file system
		L0CompactionThreshold: 20,                          // Delay compaction during bulk writes
		L0StopWritesThreshold: 40,                          // Much higher threshold
		FormatMajorVersion:    pebble.FormatColumnarBlocks, // Current format version
		Logger:                log,                         // use project's logger
		BlockPropertyCollectors: []func() pebble.BlockPropertyCollector{
			func() pebble.BlockPropertyCollector { return newVersionedPropertyCollector() },
		},
	})
	if err != nil {
		return nil, ErrOpenDB(err)
	}
	return NewStoreWithDB(lib.DefaultConfig(), db, nil, log)
}

// NewStoreWithDB() returns a Store object given a DB and a logger
// NOTE: to read the state commit store i.e. for merkle proofs, use NewReadOnly()
func NewStoreWithDB(config lib.Config, db *pebble.DB, metrics *lib.Metrics, log lib.LoggerI) (*Store, lib.ErrorI) {
	// get the latest CommitID (height and hash)
	id := getLatestCommitID(db, log)
	// set the version
	nextVersion, version := id.Height+1, id.Height
	// create a new batch writer for the next version
	writer := db.NewBatch()
	// make a versioned store from the current height and the latest height
	// note: version for the versioned store may be overridden by the SetAt() and DeleteAt() code
	hssStore := NewVersionedStore(db.NewSnapshot(), writer, version)
	lssStore := NewVersionedStore(db.NewSnapshot(), writer, lssVersion)
	// return the store object
	return &Store{
		version:    version,
		log:        log,
		db:         db,
		writer:     writer,
		ss:         NewTxn(lssStore, lssStore, latestStatePrefix, true, true, true, nextVersion),
		Indexer:    &Indexer{NewTxn(hssStore, hssStore, indexerPrefix, false, false, false, nextVersion), config},
		metrics:    metrics,
		config:     config,
		mu:         &sync.Mutex{},
		compaction: atomic.Bool{},
	}, nil
}

// NewReadOnly() returns a store without a writer - meant for historical read only queries
// CONTRACT: Read only stores cannot be copied or written to
func (s *Store) NewReadOnly(queryVersion uint64) (lib.StoreI, lib.ErrorI) {
	var stateReader *Txn
	// make a reader for the specified version
	hssReader := NewVersionedStore(s.db.NewSnapshot(), nil, queryVersion)
	// if the query is for the latest version use the HSS over the LSS
	if s.version == queryVersion {
		lssReader := NewVersionedStore(s.db.NewSnapshot(), nil, lssVersion)
		stateReader = NewTxn(lssReader, nil, latestStatePrefix, false, false, true)
	} else {
		stateReader = NewTxn(hssReader, nil, historicStatePrefix, false, false, true)
	}
	// return the store object
	return &Store{
		version:    queryVersion,
		log:        s.log,
		db:         s.db,
		ss:         stateReader,
		sc:         NewDefaultSMT(NewTxn(hssReader, nil, stateCommitmentPrefix, false, false, true)),
		Indexer:    &Indexer{NewTxn(hssReader, nil, indexerPrefix, false, false, false), s.config},
		metrics:    s.metrics,
		mu:         &sync.Mutex{},
		compaction: atomic.Bool{},
	}, nil
}

// Copy() make a copy of the store with a new read/write transaction
// this can be useful for having two simultaneous copies of the store
// ex: Mempool state and FSM state
func (s *Store) Copy() (lib.StoreI, lib.ErrorI) {
	// create a comparable writer and reader
	writer := s.db.NewBatch()
	reader := NewVersionedStore(s.db.NewSnapshot(), writer, s.version)
	lssReader := NewVersionedStore(s.db.NewSnapshot(), writer, lssVersion)
	// return the store object
	return &Store{
		version:    s.version,
		log:        s.log,
		db:         s.db,
		writer:     writer,
		ss:         s.ss.Copy(lssReader, lssReader),
		Indexer:    &Indexer{s.Indexer.db.Copy(reader, reader), s.config},
		metrics:    s.metrics,
		mu:         &sync.Mutex{},
		compaction: atomic.Bool{},
	}, nil
}

// Commit() performs a single atomic write of the current state to all stores.
func (s *Store) Commit() (root []byte, err lib.ErrorI) {
	// nested transactions should only flush changes to the parent transaction, not the database
	if s.isTxn {
		return nil, ErrCommitDB(fmt.Errorf("nested transactions are not supported"))
	}
	s.mu.Lock()         // lock commit op
	defer s.mu.Unlock() // unlock commit op
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
	// commit the in-memory txn to the pebbleDB batch
	if e := s.Flush(); e != nil {
		return nil, e
	}
	// extract the internal metrics from the pebble batch
	size, count := len(s.writer.Repr()), s.writer.Count()
	// finally commit the entire Transaction to the actual DB under the proper version (height) number
	if err := s.db.Apply(s.writer, pebble.NoSync); err != nil {
		return nil, ErrCommitDB(err)
	}
	// update the metrics once complete
	s.metrics.UpdateStoreMetrics(int64(size), int64(count), time.Time{}, startTime)
	// reset the writer for the next height
	s.Reset()
	// compact if necessary
	s.MaybeCompact()
	// return the root
	return
}

// Flush() writes the current state to the batch writer without actually committing it
func (s *Store) Flush() lib.ErrorI {
	if s.sc != nil {
		if e := s.sc.store.(TxnWriterI).Commit(); e != nil {
			return ErrCommitDB(e)
		}
	}
	if e := s.ss.Commit(); e != nil {
		return ErrCommitDB(e)
	}
	if e := s.Indexer.db.Commit(); e != nil {
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
		ss:      NewTxn(s.ss, s.ss, nil, false, true, true, nextVersion),
		Indexer: &Indexer{NewTxn(s.Indexer.db, s.Indexer.db, nil, false, true, false, nextVersion), s.config},
		metrics: s.metrics,
		mu:      s.mu,
		isTxn:   true,
	}
}

// DB() returns the underlying PebbleDB instance associated with the Store, providing access
// to the database for direct operations and management.
func (s *Store) DB() *pebble.DB { return s.db }

// Root() retrieves the root hash of the StateCommitStore, representing the current root of the
// Sparse Merkle Tree. This hash is used for verifying the integrity and consistency of the state.
func (s *Store) Root() (root []byte, err lib.ErrorI) {
	// if smt not cached
	if s.sc == nil {
		nextVersion := s.version + 1
		// set up the state commit store
		s.sc = NewDefaultSMT(NewTxn(s.ss.reader, s.ss.writer, stateCommitIDPrefix, false, false, true, nextVersion))
		// commit the SMT directly using the txn ops
		if err = s.sc.Commit(s.ss.txn.ops); err != nil {
			return nil, err
		}
	}
	// return the root
	return s.sc.Root(), nil
}

// Reset() discard and re-sets the stores writer
func (s *Store) Reset() {
	// create a new batch for the next version
	nextVersion := s.version + 1
	newWriter := s.db.NewBatch()
	// create new versioned stores first before discarding old ones
	newLSSStore := NewVersionedStore(s.db.NewSnapshot(), newWriter, lssVersion)
	newStore := NewVersionedStore(s.db.NewSnapshot(), newWriter, s.version)
	// create all new transaction-dependent objects
	newLSS := NewTxn(newLSSStore, newStore, latestStatePrefix, true, true, true, nextVersion)
	newIndexer := NewTxn(newStore, newStore, indexerPrefix, false, false, false, nextVersion)
	// only after creating all new objects, discard old transactions
	s.Discard()
	// update all references
	s.writer = newWriter
	s.ss = newLSS
	s.Indexer.setDB(newIndexer)
}

// Discard() closes the reader and writer
func (s *Store) Discard() {
	if s.isTxn {
		s.ss.Discard()
		s.Indexer.db.Discard()
		return
	}
	s.ss.Close()
	s.sc = nil
	s.Indexer.db.Close()
	if s.writer != nil {
		s.writer.Close()
	}
}

// Close() discards the writer and closes the database connection
func (s *Store) Close() lib.ErrorI {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Discard()
	if err := s.db.Flush(); err != nil {
		return ErrCloseDB(fmt.Errorf("flush error: %v", err))
	}
	if err := s.db.Close(); err != nil {
		return ErrCloseDB(err)
	}
	return nil
}

// commitIDKey() returns the key for the commitID at a specific version
func (s *Store) commitIDKey(version uint64) []byte {
	return append(stateCommitIDPrefix, lib.JoinLenPrefix(fmt.Appendf(nil, "%d", version))...)
}

// getCommitID() retrieves the CommitID value for the specified version from the database
func (s *Store) getCommitID(version uint64) (id lib.CommitID, err lib.ErrorI) {
	var bz []byte
	bz, err = NewTxn(s.Indexer.db.reader, nil, nil, false, false, true).Get(s.commitIDKey(version))
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
	// prepare the commit ID value
	value, err := lib.Marshal(&lib.CommitID{Height: version, Root: root})
	if err != nil {
		return err
	}
	vs := NewVersionedStore(nil, s.writer, version)
	if err = vs.SetAt(lastCommitIDPrefix, value, lssVersion); err != nil {
		return err
	}
	if err = vs.SetAt(s.commitIDKey(version), value, version); err != nil {
		return err
	}
	return nil
}

// getLatestCommitID() retrieves the latest CommitID from the database
func getLatestCommitID(db *pebble.DB, log lib.LoggerI) (id *lib.CommitID) {
	reader := db.NewSnapshot()
	defer reader.Close()
	vs := NewVersionedStore(reader, nil, lssVersion)
	tx := NewTxn(vs, nil, nil, false, false, true, 0)
	bz, err := tx.Get(lastCommitIDPrefix)
	if err != nil {
		log.Fatalf("getLatestCommitID() failed with err: %s", err.Error())
	}
	id = new(lib.CommitID)
	if err = lib.Unmarshal(bz, id); err != nil {
		log.Fatalf("unmarshalCommitID() failed with err: %s", err.Error())
	}
	return
}

// MaybeCompact() checks if it is time to compact the LSS and HSS respectively
func (s *Store) MaybeCompact() {
	// check if the current version is a multiple of the cleanup block interval
	compactionInterval := s.config.StoreConfig.LSSCompactionInterval
	version := s.Version()
	if compactionInterval > 0 && version%compactionInterval == 0 {
		go func() {
			// compactions are not allowed to run concurrently to not intertwine with the keys
			if s.compaction.Load() {
				s.log.Debugf("key compaction skipped [%d]: already in progress", version)
				return
			}
			s.compaction.Store(true)
			defer s.compaction.Store(false)
			// perform HSS compaction every 4th compaction
			hssCompaction := (version/compactionInterval)%4 == 0
			// trigger compaction of store keys
			if err := s.Compact(version, hssCompaction); err != nil {
				s.log.Errorf("key compaction failed: %s", err)
			}
		}()
	}
}

// Compact deletes all entries marked for compaction on the given prefix range.
// it iterates over the prefix, deletes tombstone entries, and performs DB compaction
func (s *Store) Compact(version uint64, compactHSS bool) lib.ErrorI {
	// first compaction: latest state  keys
	startPrefix, endPrefix := latestStatePrefix, prefixEnd(latestStatePrefix)
	// track current time and version
	now := time.Now()
	s.log.Debugf("key compaction started at height %d", version)
	// create a timeout to limit the duration of the compaction process
	// TODO: timeout was chosen arbitrarily, should update the number once multiple tests are run
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// create a batch to set the keys to be removed by the compaction
	batch := s.db.NewBatch()
	defer batch.Close()
	// set metrics to log
	total, toDelete := 0, 0
	// collect and delete tombstone entries
	err := s.withIterator(startPrefix, endPrefix, func(it *pebble.Iterator) {
		tombstone, _ := parseValueWithTombstone(it.Value())
		if tombstone == DeadTombstone {
			batch.Delete(it.Key(), pebble.NoSync)
			toDelete++
		} else {
			total++
		}
	})
	if err != nil {
		return ErrCommitDB(err)
	}
	// if nothing to delete, skip compaction
	if batch.Empty() {
		s.log.Debugf("key compaction finished [%d], no values to delete", version)
		return nil
	}
	// commit the batch
	s.mu.Lock() // lock commit op
	// check if batch's db closed field is not nil using reflection, this is due to a possible issue
	// on pebbleDB where the batch may come as nil due to their internal sync.Pool implementation.
	// The issue is currently being investigated and this may be removed in the future.
	// Issue: https://github.com/cockroachdb/pebble/issues/5563
	batchValue := reflect.ValueOf(batch).Elem()
	batchDBField := batchValue.FieldByName("db")
	if !batchDBField.IsValid() || batchDBField.IsNil() {
		s.mu.Unlock() // unlock commit op
		return ErrCommitDB(fmt.Errorf("db field is nil"))
	}
	dbValue := batchDBField.Elem()
	closedField := dbValue.FieldByName("closed")
	if !closedField.IsValid() || closedField.IsNil() {
		s.mu.Unlock() // unlock commit op
		return ErrCommitDB(fmt.Errorf("closed field is nil"))
	}
	if err := batch.Commit(pebble.Sync); err != nil {
		s.mu.Unlock() // unlock commit op
		return ErrCommitDB(err)
	}
	s.mu.Unlock() // unlock commit op
	batchTime := time.Since(now)
	// perform a flush to ensure memtables are flushed to disk
	if err := s.db.Flush(); err != nil {
		return ErrCommitDB(fmt.Errorf("failed to flush memtables: %v", err))
	}
	// flush and compact the range
	if err := s.db.Compact(ctx, startPrefix, endPrefix, true); err != nil {
		return ErrCommitDB(err)
	}
	lssTime := time.Since(now)
	s.log.Debugf("key compaction finished [LSS] [%d], total keys: %d, deleted: %d batch: %s elapsed: %s",
		version, total, toDelete, batchTime, lssTime)
	// second compaction: historic state keys
	if compactHSS {
		startPrefix, endPrefix = historicStatePrefix, prefixEnd(historicStatePrefix)
		hssTime := time.Now()
		if err := s.db.Compact(ctx, startPrefix, endPrefix, false); err != nil {
			return ErrCommitDB(err)
		}
		// log results
		s.log.Debugf("key compaction finished [HSS] [%d] time: %s, total time: %s", version,
			time.Since(hssTime), time.Since(now))
	}
	return nil
}

// withIterator iterates over the keys in the given range with the provided callback function.
func (s *Store) withIterator(startPrefix, endPrefix []byte, cb func(it *pebble.Iterator)) lib.ErrorI {
	db := s.db.NewSnapshot()
	defer db.Close()
	it, err := db.NewIter(&pebble.IterOptions{LowerBound: startPrefix, UpperBound: endPrefix})
	if err != nil {
		return ErrOpenDB(err)
	}
	defer it.Close()
	for it.First(); it.Valid(); it.Next() {
		cb(it)
	}
	return nil
}
