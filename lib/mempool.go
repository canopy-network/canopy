package lib

import (
	"maps"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file defines and implements a mempool that maintains an ordered list of 'valid, pending to be included' transactions in memory */

var _ Mempool = &FeeMempool{} // Mempool interface enforcement for FeeMempool implementation

// Mempool interface is a model for a pre-block, in-memory, Transaction store
type Mempool interface {
	Contains(txHash string) bool               // whether the mempool has this transaction already (de-duplicated by hash)
	AddTransactions(tx ...[]byte) (err ErrorI) // insert new unconfirmed transaction
	DeleteTransaction(tx ...[]byte)            // delete unconfirmed transaction
	GetTransactions(maxBytes uint64) [][]byte  // retrieve transactions from the highest fee to lowest

	Clear()              // reset the entire store
	TxCount() int        // number of Transactions in the pool
	TxsBytes() int       // collective number of bytes in the pool
	Iterator() IteratorI // loop through each transaction in the pool
}

// FeeMempool is a Mempool implementation that prioritizes transactions with the highest fees
type FeeMempool struct {
	pool     MempoolTxs    // the actual pool of transactions
	txsBytes int           // collective number of bytes in the pool
	config   MempoolConfig // user configuration of the pool
}

// MempoolTx is a wrapper over Transaction bytes that maintains the fee associated with the bytes
type MempoolTx struct {
	Tx  []byte // transaction bytes
	Fee uint64 // fee associated with the transaction
}

// NewMempool() creates a new FeeMempool instance of a Mempool
func NewMempool(config MempoolConfig) Mempool {
	// if the config drop percentage is set to 0
	if config.DropPercentage == 0 {
		// set the drop percentage to the default mempool config
		config.DropPercentage = DefaultMempoolConfig().DropPercentage
	}
	// return the default mempool
	return &FeeMempool{
		pool:   MempoolTxs{},
		config: config,
	}
}

// AddTransaction() inserts a new unconfirmed Transaction to the Pool and returns if this addition
// requires a recheck of the Mempool due to dropping or re-ordering of the Transactions
func (f *FeeMempool) AddTransactions(txs ...[]byte) (err ErrorI) {
	// create a list of MempoolTxs
	mempoolTxs := make([]MempoolTx, 0, len(txs))
	for _, tx := range txs {
		// ensure the size of the Transaction doesn't exceed the individual limit
		txBytes := len(tx)
		// if the transaction bytes is larger than the max size
		if uint32(txBytes) > f.config.IndividualMaxTxSize {
			// exit with error
			return ErrMaxTxSize()
		}
		// check if the mempool already contains the transaction
		if _, found := f.pool.m[crypto.HashString(tx)]; found {
			continue // skip already contains
		}
		// create a new transaction object reference to ensure a non-nil transaction
		transaction := new(Transaction)
		// populate the object ref with the bytes of the transaction
		if err = Unmarshal(tx, transaction); err != nil {
			return
		}
		// perform basic validations against the tx object
		if err = transaction.CheckBasic(); err != nil {
			return
		}
		// extract the fee from the transaction result
		fee := transaction.Fee
		// if the transaction is a special type: 'certificate result'
		if transaction.MessageType == "certificateResults" {
			// prioritize certificate result transactions by artificially raising the fee 'stored fee'
			fee = math.MaxUint32
		}
		// add to the list
		mempoolTxs = append(mempoolTxs, MempoolTx{Tx: tx, Fee: fee})
		// update the number of bytes
		f.txsBytes += txBytes
	}
	// insert the transactions into the pool
	f.pool.insert(mempoolTxs...)
	// assess if limits are exceeded - if so, drop from the bottom
	var dropped []MempoolTx
	// handle bad config
	if f.config.MaxTransactionCount == 0 {
		f.config.MaxTransactionCount = 1
	}
	// loop until the conditions are satisfied
	for uint32(len(f.pool.s)) >= f.config.MaxTransactionCount || uint64(f.txsBytes) > f.config.MaxTotalBytes {
		// drop percentage is configurable
		dropped = f.pool.drop(f.config.DropPercentage)
		// for each dropped transaction
		for _, d := range dropped {
			// subtract the txsBytes
			f.txsBytes -= len(d.Tx)
		}
	}
	// if any are dropped or re-order happened
	return nil
}

// GetTransactions() returns a list of the Transactions from the pool up to 'max collective Transaction bytes'
func (f *FeeMempool) GetTransactions(maxBytes uint64) (txs [][]byte) {
	// create a variable to track the total transaction byte count
	totalBytes := uint64(0)
	// for each transaction in the pool
	for _, tx := range f.pool.s {
		// get the size of the transaction in bytes
		txSize := len(tx.Tx)
		// add to the total bytes
		totalBytes += uint64(txSize)
		// check to see if the addition of this transaction
		// exceeds the maxBytes limit
		if totalBytes > maxBytes {
			// exit without adding the tx
			return
		}
		// add the tx to the list and increment totalTxs
		txs = append(txs, tx.Tx)
	}
	// exit
	return
}

// Contains() checks if a transaction with the given hash exists in the mempool
func (f *FeeMempool) Contains(txHash string) (contains bool) {
	// check if the hash map contains the transaction hash
	_, contains = f.pool.m[txHash]
	// exit
	return
}

// DeleteTransaction() removes the specified transaction from the mempool
func (f *FeeMempool) DeleteTransaction(tx ...[]byte) {
	// delete the transaction from the pool
	_, deletedBz := f.pool.delete(tx)
	// subtract the from the tx bytes count
	f.txsBytes -= deletedBz
}

// Clear() empties the mempool and resets its state
func (f *FeeMempool) Clear() {
	// reset the memory pool of transactions
	f.pool = MempoolTxs{s: make([]MempoolTx, 0)}
	// reset the bytes count
	f.txsBytes = 0
}

// TxCount() returns the current number of transactions in the mempool
func (f *FeeMempool) TxCount() int { return len(f.pool.s) }

// TxsBytes() returns the total size in bytes of all transactions in the mempool
func (f *FeeMempool) TxsBytes() int {
	// return the number of bytes in the memory pool
	return f.txsBytes
}

// Iterator() creates a new iterator for traversing the transactions in the mempool
func (f *FeeMempool) Iterator() IteratorI {
	// exit with a new mempool iterator
	return NewMempoolIterator(f.pool)
}

var _ IteratorI = &mempoolIterator{} // enforce

// mempoolIterator implements IteratorI using the list of Transactions the index and if the position is valid
type mempoolIterator struct {
	pool  *MempoolTxs // reference to list of Transactions
	index int         // index position
	valid bool        // is the position valid
}

// NewMempoolIterator() initializes a new iterator for the mempool transactions
func NewMempoolIterator(p MempoolTxs) *mempoolIterator {
	pool := p.copy() // copy the pool for safe iteration during a parallel
	return &mempoolIterator{pool: pool, valid: len(pool.s) != 0}
}

// Valid() checks if the iterator is positioned on a valid element
func (m *mempoolIterator) Valid() bool { return m.index < len(m.pool.s) }

// Next() advances the iterator to the next transaction in the pool
func (m *mempoolIterator) Next() { m.index++ }

// Key() returns the transaction at the current iterator position
func (m *mempoolIterator) Key() (key []byte) { return m.pool.s[m.index].Tx }

// Value() returns same as key
func (m *mempoolIterator) Value() (value []byte) { return m.Key() }

// Error() always returns nil, as no errors are tracked by this iterator
func (m *mempoolIterator) Error() error { return nil }

// Close() is a no-op in this iterator, as no resources need to be released
func (m *mempoolIterator) Close() {}

// MempoolTxs is a list of MempoolTxs with a count
type MempoolTxs struct {
	m map[string]struct{}
	s []MempoolTx
}

// insert() batch inserts a number of txs into the list sorted by the highest fee to the lowest fee
func (t *MempoolTxs) insert(txs ...MempoolTx) {
	// combine existing and incoming txs
	combined := append(t.s, txs...)
	// sort by Fee descending
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Fee > combined[j].Fee
	})
	// prepare new map and slice
	newMap := make(map[string]struct{}, len(combined))
	newList := make([]MempoolTx, 0, len(combined))
	// for each tx in the combined array
	for _, tx := range combined {
		hash := crypto.HashString(tx.Tx)
		// skip duplicates
		if _, exists := newMap[hash]; exists {
			continue
		}
		// populate the newMap and newList
		newList = append(newList, tx)
		newMap[hash] = struct{}{}
	}
	// update
	t.s = newList
	t.m = newMap
}

// delete() batch deletes a number of transactions
func (t *MempoolTxs) delete(txs [][]byte) (deleted []MempoolTx, deletedBz int) {
	if len(txs) == 0 || len(t.s) == 0 {
		return nil, 0
	}
	// build a lookup set for hashes to delete
	toDelete := make(map[string]struct{}, len(txs))
	for _, h := range txs {
		toDelete[crypto.HashString(h)] = struct{}{}
	}
	// allocate new slice
	newList := make([]MempoolTx, 0, len(t.s))
	newMap := make(map[string]struct{}, len(t.s))
	// for each tx in the current slice
	for _, tx := range t.s {
		// create the hash
		hash := crypto.HashString(tx.Tx)
		// if should delete
		if _, found := toDelete[hash]; found {
			// add to deleted list
			deleted = append(deleted, tx)
			// add to deleted bytes
			deletedBz += len(tx.Tx)
			// continue
			continue
		}
		// add to new list and map
		newList = append(newList, tx)
		newMap[hash] = struct{}{}
	}
	// update
	t.s = newList
	t.m = newMap
	// exit
	return
}

// drop() removes the bottom (the lowest fee) X percent of Transactions
func (t *MempoolTxs) drop(percent int) (dropped []MempoolTx) {
	// calculate the percent using integer division
	numDrop := (len(t.s)*percent)/100 + 1
	// avoid slicing out of bounds
	if numDrop >= len(t.s) {
		numDrop = len(t.s)
	}
	// save the evicted list
	dropped = t.s[len(t.s)-numDrop:]
	// update the list with what's not evicted
	t.s = t.s[:len(t.s)-numDrop]
	// rebuild map
	t.m = make(map[string]struct{}, len(t.s))
	for _, tx := range t.s {
		t.m[crypto.HashString(tx.Tx)] = struct{}{}
	}
	// exit
	return
}

// copy() returns a shallow copy of the MempoolTxs
func (t *MempoolTxs) copy() *MempoolTxs {
	// allocate a destination
	dst := make([]MempoolTx, len(t.s))
	dstM := make(map[string]struct{}, len(t.s))
	// shallow copy the source to destination
	copy(dst, t.s)
	maps.Copy(dstM, t.m)
	// exit with copy
	return &MempoolTxs{
		s: dst,
		m: dstM,
	}
}

// FAILED TX CACHE CODE BELOW

// FailedTxCache is a cache of failed transactions that is used to inform the user of the failure
type FailedTxCache struct {
	cache                  map[string]*FailedTx // map tx hashes to errors
	disallowedMessageTypes []string             // reject all transactions that are of these types
	l                      sync.Mutex           // a lock for thread safety
}

// NewFailedTxCache returns a new FailedTxCache
func NewFailedTxCache(disallowedMessageTypes ...string) (cache *FailedTxCache) {
	// initialize the failed transactions cache
	cache = &FailedTxCache{
		cache:                  map[string]*FailedTx{},
		l:                      sync.Mutex{},
		disallowedMessageTypes: disallowedMessageTypes,
	}
	// start the cleaning service
	go cache.StartCleanService()
	// exit with the cache
	return
}

// NewFailedTx() attempts to create a new failed transaction from bytes
func NewFailedTx(txBytes []byte, txErr error) *FailedTx {
	addressString := "unknown"
	// create a new transaction object reference to ensure a non nil result
	tx := new(Transaction)
	// populate the new object reference using the transaction bytes
	_ = Unmarshal(txBytes, tx)
	// if the signature is empty
	if tx.Signature != nil {
		// get the public key object from the bytes of the signature
		pubKey, err := crypto.NewPublicKeyFromBytes(tx.Signature.PublicKey)
		if err == nil {
			addressString = pubKey.Address().String()
		}
	}
	return &FailedTx{
		Transaction: tx,
		Hash:        crypto.HashString(txBytes),
		Address:     addressString,
		Error:       txErr,
		timestamp:   time.Now(),
		bytes:       txBytes,
	}
}

// Add() adds a failed transaction with its error to the cache
func (f *FailedTxCache) Add(failed *FailedTx) (added bool) {
	// lock for thread safety
	f.l.Lock()
	// unlock when the function completes
	defer f.l.Unlock()
	// add a new 'failed tx' type to the cache
	f.cache[failed.Hash] = failed
	// exit with 'added'
	return true
}

// Get() returns the failed transaction associated with its hash
func (f *FailedTxCache) Get(txHash string) (failedTx *FailedTx, found bool) {
	// lock for thread safety
	f.l.Lock()
	// unlock when the function completes
	defer f.l.Unlock()
	// get the failed tx from the cache
	failedTx, found = f.cache[txHash]
	// if not found in the cache
	if !found {
		// exit with not found
		return
	}
	// exit
	return
}

// GetFailedForAddress() returns all the failed transactions in the cache for a given address
func (f *FailedTxCache) GetFailedForAddress(address string) (failedTxs []*FailedTx) {
	// lock for thread safety
	f.l.Lock()
	// unlock when the function completes
	defer f.l.Unlock()
	// for each failed transaction in the cache
	for _, failed := range f.cache {
		// if the address matches
		if failed.Address == address {
			// add to the list
			failedTxs = append(failedTxs, failed)
		}
	}
	// exit
	return
}

// Remove() removes a transaction hash from the cache
func (f *FailedTxCache) Remove(txHashes ...string) {
	// lock for thread safety
	f.l.Lock()
	// unlock when function completes
	defer f.l.Unlock()
	// for each transaction hash
	for _, hash := range txHashes {
		// remove it from the memory cache
		delete(f.cache, hash)
	}
}

// StartCleanService() periodically removes transactions from the cache that are older than 5 minutes
func (f *FailedTxCache) StartCleanService() {
	// every minute until app stops
	for range time.Tick(time.Minute) {
		// wrap in a function to use 'defer'
		func() {
			// lock for thread safety
			f.l.Lock()
			// unlock when iteration completes
			defer f.l.Unlock()
			// for each in the cache
			for hash, tx := range f.cache {
				// if the 'time since' is greater than 5 minutes
				if time.Since(tx.timestamp) >= 5*time.Minute {
					// remove it from the cache
					delete(f.cache, hash)
				}
			}
		}()
	}
}

// GetBytes() returns the raw tx bytes for the failed tx
func (f *FailedTx) GetBytes() []byte { return f.bytes }

// FailedTx contains a failed transaction and its error
type FailedTx struct {
	Transaction *Transaction `json:"transaction,omitempty"` // the transaction object that failed
	Hash        string       `json:"txHash,omitempty"`      // the hash of the transaction object
	Address     string       `json:"address,omitempty"`     // the address that sent the transaction
	Error       error        `json:"error,omitempty"`       // the error that occurred
	timestamp   time.Time    // the time when the failure was recorded
	bytes       []byte       // raw bytes of the transaction
}

type FailedTxs []*FailedTx // a list of failed transactions

// ensure failed txs implements the pageable interface
var _ Pageable = &FailedTxs{}

// implement pageable interface
func (t *FailedTxs) Len() int      { return len(*t) }
func (t *FailedTxs) New() Pageable { return &FailedTxs{} }
