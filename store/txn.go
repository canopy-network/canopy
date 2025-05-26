package store

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/skl"
)

const (
	// ----------------------------------------------------------------------------------------------------------------
	// BadgerDB garbage collector behavior is not well documented leading to many open issues in their repository
	// However, here is our current understanding based on experimentation
	// ----------------------------------------------------------------------------------------------------------------
	// 1. Manual Keep (Protection)
	//    - `badgerNoDiscardBit` prevents automatic GC of a key version.
	//    - However, it can be manually superseded by a manual removal
	//
	// 2. Manual Remove (Explicit Deletion or Pruning)
	//    - Deleting a key at a higher ts removes earlier versions once `discardTs >= ts`.
	//    - Setting `badgerDiscardEarlierVersions` is similar, except it retains the current version.
	//
	// 3. Auto Remove – Tombstones
	//    - Deleted keys (tombstoned) <= `discardTs` are automatically purged unless protected by `badgerNoDiscardBit`
	//
	// 4. Auto Remove – Set Entries
	//    - For non-deleted (live) keys, Badger retains the number of versions to retain is defined by `KeepNumVersions`.
	//    - Older versions exceeding this count are automatically eligible for GC.
	//
	//   Note:
	// - The first GC pass after updating `discardTs` and flushing memtable is deterministic
	// - Subsequent GC runs are probabilistic, depending on reclaimable space and value log thresholds
	// ----------------------------------------------------------------------------------------------------------------
	// Bits source: https://github.com/hypermodeinc/badger/blob/85389e88bf308c1dc271383b77b67f4ef4a85194/value.go#L37
	badgerMetaFieldName                = "meta"  // badgerDB Entry 'meta' field name
	badgerDiscardEarlierVersions  byte = 1 << 2  // badgerDB 'discard earlier versions' flag
	badgerDeleteBit               byte = 1 << 0  // badgerDB 'tombstoned' flag
	badgerNoDiscardBit            byte = 1 << 3  // badgerDB 'never discard'  bit
	badgerGCRatio                      = .15     // the ratio when badgerDB will run the garbage collector
	badgerSizeFieldName                = "size"  // badgerDB Txn 'size' field name
	badgerCountFieldName               = "count" // badgerDB Txn 'count' field name
	badgerTxnFieldName                 = "txn"   // badgerDB WriteBatch 'txn' field name
	badgerDBMaxBatchScalingFactor      = 0.98425 // through experimentation badgerDB's max transaction scaling factor
)

// TxReaderI() defines the interface to read a TxnTransaction
// Txn implements this itself to allow for nested transactions
type TxnReaderI interface {
	Get(key []byte) ([]byte, lib.ErrorI)
	NewIterator(prefix []byte, reverse bool, allVersions bool) lib.IteratorI
	Discard()
}

// TxnWriterI() defines the interface to write a TxnTransaction
// Txn implements this itself to allow for nested transactions
type TxnWriterI interface {
	Set(key, value []byte) lib.ErrorI
	Delete(key []byte) lib.ErrorI
	SetEntry(entry *badger.Entry) lib.ErrorI
	Write() lib.ErrorI
	Cancel()
}

// enforce the necessary interfaces over Txn
var _ lib.RWStoreI = &Txn{}
var _ TxnReaderI = &Txn{}
var _ TxnWriterI = &Txn{}
var _ lib.IteratorI = &Iterator{}

/*
	Txn acts like a database transaction
	It saves set/del operations in memory and allows the caller to Write() to the parent or Discard()
	When read from, it merges with the parent as if Write() had already been called

	Txn abstraction is necessary due to the inability of BadgerDB to have nested transactions.
	Txns allow an easy rollback of write operations within a single Transaction object, which is necessary
	for ephemeral states and testing the validity of a proposal block / transactions.

	CONTRACT:
	- only safe when writing to another memory store like a badger.Txn() as Write() is not atomic.
	- not thread safe (can't use 1 txn across multiple threads)
	- nil values are supported; deleted values are also set to nil
	- keys must be smaller than 128 bytes
	- Nested txns are supported, but iteration becomes increasingly inefficient
*/

type Txn struct {
	reader TxnReaderI  // memory store to Read() from
	writer TxnWriterI  // memory store to Write() to
	prefix []byte      // prefix for keys in this txn
	logger lib.LoggerI // logger for this txn
	cache  txn
}

// txn internal structure maintains the write operations sorted lexicographically by keys
type txn struct {
	ops       map[string]valueOp // [string(key)] -> set/del operations saved in memory
	sorted    []string           // ops keys sorted lexicographically; needed for iteration
	sortedLen int                // len(sorted)
}

// txn() returns a copy of the current transaction cache
func (t txn) copy() txn {
	sorted := make([]string, t.sortedLen)
	copy(sorted, t.sorted)
	ops := make(map[string]valueOp, t.sortedLen)
	for k, v := range t.ops {
		ops[k] = v
	}
	return txn{
		ops:       ops,
		sorted:    sorted,
		sortedLen: t.sortedLen,
	}
}

// op is the type of operation to be performed on the key
type op uint8

const (
	opDelete    op = iota // delete the key
	opSet                 // set the key
	opTombstone           // tombstone the key (false delete)
	opEntry               // custom badger entry
)

// valueOp has the value portion of the operation and the corresponding operation to perform
type valueOp struct {
	value      []byte        // value of key value pair
	valueEntry *badger.Entry // value of key value pair in case of a custom entry
	op         op            // is operation delete
}

// NewBadgerTxn() creates a new instance of Txn from badger Txn and WriteBatch correspondingly
func NewBadgerTxn(reader *badger.Txn, writer *badger.WriteBatch, prefix []byte, logger lib.LoggerI) *Txn {
	return &Txn{
		reader: BadgerTxnReader{reader, prefix},
		writer: BadgerTxnWriter{writer},
		prefix: prefix,
		logger: logger,
		cache: txn{
			ops:    make(map[string]valueOp),
			sorted: make([]string, 0),
		},
	}
}

// NewTxn() creates a new instance of Txn with the specified reader and writer
func NewTxn(reader TxnReaderI, writer TxnWriterI, prefix []byte, logger lib.LoggerI) *Txn {
	return &Txn{
		reader: reader,
		writer: writer,
		prefix: prefix,
		logger: logger,
		cache: txn{
			ops:    make(map[string]valueOp),
			sorted: make([]string, 0),
		},
	}
}

// Get() retrieves the value for a given key from either the cache operations or the reader store
func (t *Txn) Get(key []byte) ([]byte, lib.ErrorI) {
	// append the prefix to the key
	prefixedKey := lib.Append(t.prefix, key)
	// first retrieve from the in-memory cache
	if v, found := t.cache.ops[lib.BytesToString(prefixedKey)]; found {
		return v.value, nil
	}
	// if not found, retrieve from the parent reader
	return t.reader.Get(prefixedKey)
}

// Set() adds or updates the value for a key in the cache operations
func (t *Txn) Set(key, value []byte) lib.ErrorI {
	t.update(lib.BytesToString(lib.Append(t.prefix, key)), value, opSet)
	return nil
}

// Delete() marks a key for deletion in the cache operations
func (t *Txn) Delete(key []byte) lib.ErrorI {
	t.update(lib.BytesToString(lib.Append(t.prefix, key)), nil, opDelete)
	return nil
}

// Tombstone() removes the key-value pair from the BadgerDB transaction but prevents it from being garbage collected
func (t *Txn) Tombstone(k []byte) lib.ErrorI {
	t.update(lib.BytesToString(lib.Append(t.prefix, k)), nil, opTombstone)
	return nil
}

// SetEntry() adds or updates a custom badger entry in the cache operations
func (t *Txn) SetEntry(entry *badger.Entry) lib.ErrorI {
	t.updateEntry(lib.BytesToString(lib.Append(t.prefix, entry.Key)), entry)
	return nil
}

// update() modifies or adds an operation for a key in the cache operations and maintains the
// lexicographical order.
// NOTE: update() won't modify the key itself, any key prefixing must be done before calling this
func (t *Txn) update(key string, v []byte, opAction op) {
	if _, found := t.cache.ops[key]; !found {
		t.addToSorted(key)
	}
	t.cache.ops[key] = valueOp{value: v, op: opAction}
}

// updateEntry() modifies or adds a custom badger entry in the cache operations and maintains the
// lexicographical order.
// NOTE: updateEntry() won't modify the key itself, any key prefixing must be done before calling this
func (t *Txn) updateEntry(key string, v *badger.Entry) {
	if _, found := t.cache.ops[key]; !found {
		t.addToSorted(key)
	}
	t.cache.ops[key] = valueOp{valueEntry: v, op: opEntry}
}

// addToSorted() inserts a key into the sorted list of operations maintaining lexicographical order
func (t *Txn) addToSorted(key string) {
	i := sort.Search(t.cache.sortedLen, func(i int) bool { return t.cache.sorted[i] >= key })
	t.cache.sorted = append(t.cache.sorted, "")
	copy(t.cache.sorted[i+1:], t.cache.sorted[i:])
	t.cache.sorted[i] = key
	t.cache.sortedLen++
}

// Iterator() returns a new iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it := t.reader.NewIterator(prefix, false, false)
	return newTxnIterator(it, t.cache.copy(), t.prefix, prefix, false), nil
}

// RevIterator() returns a new reverse iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it := t.reader.NewIterator(prefix, true, false)
	return newTxnIterator(it, t.cache, t.prefix, prefix, true), nil
}

// ArchiveIterator() creates a new iterator for all versions under the given prefix in the BadgerDB transaction
func (t *Txn) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return t.reader.NewIterator(prefix, false, true), nil
}

// Discard() clears all in-memory operations and resets the sorted key list
func (t *Txn) Discard() {
	t.cache.ops, t.cache.sorted, t.cache.sortedLen = make(map[string]valueOp), make([]string, 0), 0
}

// Cancel() cancels the current transaction. Any new writes won't be committed
func (t *Txn) Cancel() {
	t.writer.Cancel()
}

// Write() flushes the in-memory operations to the batch writer and clears in-memory changes
func (t *Txn) Write() lib.ErrorI {
	for k, v := range t.cache.ops {
		sk, er := lib.StringToBytes(k)
		if er != nil {
			return er
		}
		switch v.op {
		case opSet:
			if err := t.writer.Set(sk, v.value); err != nil {
				return ErrStoreSet(err)
			}
		case opDelete:
			if err := t.writer.Delete(sk); err != nil {
				return ErrStoreDelete(err)
			}
		case opTombstone:
			// set an entry with a bit that marks it as deleted and prevents it from being discarded
			if err := t.writer.SetEntry(newEntry(sk, nil, badgerDeleteBit|badgerNoDiscardBit)); err != nil {
				return ErrStoreDelete(err)
			}
		case opEntry:
			// set the entry in the batch
			if err := t.writer.SetEntry(v.valueEntry); err != nil {
				return ErrStoreSet(err)
			}
		}
	}
	t.Discard() // clear the in-memory operations after writing

	return nil
}

func (t *Txn) NewIterator(prefix []byte, reverse bool, allVersions bool) lib.IteratorI {
	// Combine the current in-memory cache and parent reader (recursively)
	combinedParentIterator := t.reader.NewIterator(lib.Append(t.prefix, prefix), reverse, allVersions)

	// Create a merged iterator for the parent and in-memory cache
	return newTxnIterator(combinedParentIterator, t.cache, t.prefix, prefix, reverse)
}

// Close() cancels the current transaction. Any new writes will result in an error and a new
// WriteBatch() must be created to write new entries.
func (t *Txn) Close() {
	t.reader.Discard()
	t.writer.Cancel()
}
func (t *Txn) setDBWriter(w *badger.WriteBatch) { t.writer = BadgerTxnWriter{w} }
func (t *Txn) setDBReader(r *badger.Txn)        { t.reader = BadgerTxnReader{r, t.prefix} }

// TXN ITERATOR CODE BELOW

// enforce the Iterator interface
var _ lib.IteratorI = &TxnIterator{}

// TxnIterator is a reversible, merged iterator of the parent and the in-memory operations
type TxnIterator struct {
	parent lib.IteratorI
	txn
	prefix       string
	parentPrefix string
	index        int
	reverse      bool
	invalid      bool
	useTxn       bool
}

// newTxnIterator() initializes a new merged iterator for traversing both the in-memory operations and parent store
func newTxnIterator(parent lib.IteratorI, t txn, parentPrefix, prefix []byte, reverse bool) *TxnIterator {
	return (&TxnIterator{
		parent:       parent,
		txn:          t,
		parentPrefix: lib.BytesToString(parentPrefix),
		prefix:       lib.BytesToString(prefix),
		reverse:      reverse,
	}).First()
}

// First() positions the iterator at the first valid entry based on the traversal direction
func (ti *TxnIterator) First() *TxnIterator {
	if ti.reverse {
		return ti.revSeek() // seek to the end
	}
	return ti.seek() // seek to the beginning
}

// Close() closes the merged iterator
func (ti *TxnIterator) Close() { ti.parent.Close() }

// Next() advances the iterator to the next entry, choosing between in-memory and parent store entries
func (ti *TxnIterator) Next() {
	// if parent is not usable any more then txn.Next()
	// if txn is not usable any more then parent.Next()
	if !ti.parent.Valid() {
		ti.txnNext()
		return
	}
	if ti.txnInvalid() {
		ti.parent.Next()
		return
	}
	// compare the keys of the in memory option and the parent option
	switch ti.compare(ti.txnKey(), ti.parent.Key()) {
	case 1: // use parent
		ti.parent.Next()
	case 0: // use both
		ti.parent.Next()
		ti.txnNext()
	case -1: // use txn
		ti.txnNext()
	}
}

// Key() returns the current key from either the in-memory operations or the parent store
func (ti *TxnIterator) Key() []byte {
	if ti.useTxn {
		return ti.txnKey()
	}
	return ti.parent.Key()
}

// Value() returns the current value from either the in-memory operations or the parent store
func (ti *TxnIterator) Value() []byte {
	if ti.useTxn {
		return ti.txnValue().value
	}
	return ti.parent.Value()
}

// Valid() checks if the current position of the iterator is valid, considering both the parent and in-memory entries
func (ti *TxnIterator) Valid() bool {
	for {
		if !ti.parent.Valid() {
			// only using cache; call txn.next until invalid or !deleted
			ti.txnFastForward()
			ti.useTxn = true
			break
		}
		if ti.txnInvalid() {
			// parent is valid; txn is not
			ti.useTxn = false
			break
		}
		// both are valid; key comparison matters
		cKey, pKey := ti.txnKey(), ti.parent.Key()
		switch ti.compare(cKey, pKey) {
		case 1: // use parent
			ti.useTxn = false
		case 0: // when equal txn shadows parent
			if ti.txnValue().op == opDelete || ti.txnValue().op == opTombstone {
				ti.parent.Next()
				ti.txnNext()
				continue
			}
			ti.useTxn = true
		case -1: // use txn
			if ti.txnValue().op == opDelete || ti.txnValue().op == opTombstone {
				ti.txnNext()
				continue
			}
			ti.useTxn = true
		}
		break
	}
	return !ti.txnInvalid() || ti.parent.Valid()
}

// txnFastForward() skips over deleted entries in the in-memory operations
// return when invalid or !deleted
func (ti *TxnIterator) txnFastForward() {
	for {
		if ti.txnInvalid() || !(ti.txnValue().op == opDelete || ti.txnValue().op == opTombstone) {
			return
		}
		ti.txnNext()
	}
}

// txnInvalid() determines if the current in-memory entry is invalid
func (ti *TxnIterator) txnInvalid() bool {
	if ti.invalid {
		return ti.invalid
	}
	ti.invalid = true
	if ti.reverse {
		if ti.index < 0 {
			return ti.invalid
		}
	} else {
		if ti.index >= ti.sortedLen {
			return ti.invalid
		}
	}
	if !strings.HasPrefix(ti.sorted[ti.index], ti.parentPrefix+ti.prefix) {
		return ti.invalid
	}
	ti.invalid = false
	return ti.invalid
}

// txnKey() returns the key of the current in-memory operation
func (ti *TxnIterator) txnKey() []byte {
	bz, _ := lib.StringToBytes(strings.TrimPrefix(ti.sorted[ti.index], ti.parentPrefix))
	return bz
}

// txnValue() returns the value of the current in-memory operation
func (ti *TxnIterator) txnValue() valueOp { return ti.ops[ti.sorted[ti.index]] }

// compare() compares two byte slices, adjusting for reverse iteration if needed
func (ti *TxnIterator) compare(a, b []byte) int {
	if ti.reverse {
		return bytes.Compare(a, b) * -1
	}
	return bytes.Compare(a, b)
}

// txnNext() advances the index of the in-memory operations based on the iteration direction
func (ti *TxnIterator) txnNext() {
	if ti.reverse {
		ti.index--
	} else {
		ti.index++
	}
}

// seek() positions the iterator at the first entry that matches or exceeds the prefix.
func (ti *TxnIterator) seek() *TxnIterator {
	ti.index = sort.Search(ti.sortedLen, func(i int) bool {
		return ti.sorted[i] >= (ti.parentPrefix + ti.prefix)
	})
	return ti
}

// revSeek() positions the iterator at the last entry that matches the prefix in reverse order.
func (ti *TxnIterator) revSeek() *TxnIterator {
	bz, _ := lib.StringToBytes(ti.parentPrefix + ti.prefix)
	endPrefix := lib.BytesToString(prefixEnd(bz))
	ti.index = sort.Search(ti.sortedLen, func(i int) bool {
		return ti.sorted[i] >= endPrefix
	}) - 1
	return ti
}

// Iterator implements a wrapper around BadgerDB's iterator with the in-memory store but satisfies the IteratorI interface
type Iterator struct {
	reader *badger.Iterator
	prefix []byte
	err    error
}

func (i *Iterator) Valid() bool {
	valid := i.reader.Valid()
	return valid
}
func (i *Iterator) Next()           { i.reader.Next() }
func (i *Iterator) Close()          { i.reader.Close() }
func (i *Iterator) Version() uint64 { return i.reader.Item().Version() }
func (i *Iterator) Deleted() bool   { return i.reader.Item().IsDeletedOrExpired() }
func (i *Iterator) Key() (key []byte) {
	// get the key from the parent
	key = i.reader.Item().Key()
	// make a copy of the key
	c := make([]byte, len(key))
	copy(c, key)
	// remove the prefix and return
	return removePrefix(c, []byte(i.prefix))
}

// Value() retrieves the current value from the iterator
func (i *Iterator) Value() (value []byte) {
	value, err := i.reader.Item().ValueCopy(nil)
	if err != nil {
		i.err = err
	}
	return
}

// BADGERDB TXNWRITER AND TXNREADER INTERFACES IMPLEMENTATION BELOW

// Enforce interface implementations
var _ TxnReaderI = &BadgerTxnReader{}
var _ TxnWriterI = &BadgerTxnWriter{}

type BadgerTxnReader struct {
	*badger.Txn
	prefix []byte
}

func (r BadgerTxnReader) Get(key []byte) ([]byte, lib.ErrorI) {
	item, err := r.Txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, ErrStoreGet(err)
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, ErrStoreGet(err)
	}
	return val, nil
}

func (r BadgerTxnReader) NewIterator(prefix []byte, reverse bool, allVersions bool) lib.IteratorI {
	newPrefix := lib.Append(r.prefix, prefix)
	it := r.Txn.NewIterator(badger.IteratorOptions{
		Prefix:      newPrefix,
		Reverse:     reverse,
		AllVersions: allVersions,
	})
	if !reverse {
		it.Rewind()
	} else {
		seekLast(it, newPrefix)
	}
	return &Iterator{
		reader: it,
		prefix: r.prefix,
	}
}

func (r BadgerTxnReader) Discard() {
	r.Txn.Discard()
}

type BadgerTxnWriter struct {
	*badger.WriteBatch
}

func (w BadgerTxnWriter) Set(key, value []byte) lib.ErrorI {
	err := w.WriteBatch.Set(key, value)
	if err != nil {
		return ErrStoreSet(err)
	}
	return nil
}

func (w BadgerTxnWriter) Delete(key []byte) lib.ErrorI {
	err := w.WriteBatch.Delete(key)
	if err != nil {
		return ErrStoreDelete(err)
	}
	return nil
}

func (w BadgerTxnWriter) SetEntry(entry *badger.Entry) lib.ErrorI {
	err := w.WriteBatch.SetEntry(entry)
	if err != nil {
		return ErrStoreSet(err)
	}
	return nil
}

func (w BadgerTxnWriter) Write() lib.ErrorI {
	err := w.WriteBatch.Flush()
	if err != nil {
		return ErrFlushBatch(err)
	}
	return nil
}

func (w BadgerTxnWriter) Cancel() {
	w.WriteBatch.Cancel()
}

var (
	endBytes = bytes.Repeat([]byte{0xFF}, maxKeyBytes+1)
)

// removePrefix() removes the prefix from the key
func removePrefix(b, prefix []byte) []byte { return b[len(prefix):] }

// prefixEnd() returns the end key for a given prefix by appending max possible bytes
func prefixEnd(prefix []byte) []byte {
	return lib.Append(prefix, endBytes)
}

// newEntry() creates a new badgerDB entry
func newEntry(key, value []byte, meta byte) (e *badger.Entry) {
	e = &badger.Entry{Key: key, Value: value}
	setMeta(e, meta)
	return
}

// FlushMemTable() ensures badgerDB is flushing its mem table before running flatten
// IMPORTANT - discardTs must be set before this
func FlushMemTable(db *badger.DB) lib.ErrorI {
	// get random 32 bytes
	randomPrefix := make([]byte, 32)
	if _, err := rand.Read(randomPrefix); err != nil {
		return ErrReadBytes(err)
	}
	// create a new transaction to write to the database
	tx := db.NewTransactionAt(math.MaxUint64, true)
	// write the random prefix to the database
	if err := tx.Set(randomPrefix, nil); err != nil {
		return ErrSetEntry(err)
	}
	// commit the transaction
	if err := tx.CommitAt(math.MaxUint64, nil); err != nil {
		return ErrCommitDB(err)
	}
	// call drop prefix which triggers the mempool flush
	// NOTE: this only works if an actual prefix exists
	if err := db.DropPrefix(randomPrefix); err != nil {
		return ErrFlushMemTable(err)
	}
	return nil
}

// setBatchOptions() updates unexported badgerDB batch options
func setBatchOptions(db *badger.DB, batchSize int64) error {
	// Access the DB struct's Options field using reflection
	v := reflect.ValueOf(db).Elem()

	// Access the 'opt' field of DB struct using unsafe
	optField := v.FieldByName("opt")
	if !optField.IsValid() {
		return fmt.Errorf("unable to access 'opt' field")
	}

	// Use unsafe.Pointer to get a pointer to the unexported field
	optPtr := unsafe.Pointer(optField.UnsafeAddr())
	optStruct := reflect.NewAt(optField.Type(), optPtr).Elem()

	maxBatchSizeField := optStruct.FieldByName("maxBatchSize")
	maxBatchCountField := optStruct.FieldByName("maxBatchCount")
	// Now, use unsafe.Pointer to modify the unexported fields directly
	batchSizePtr := unsafe.Pointer(maxBatchSizeField.UnsafeAddr())
	batchCountPtr := unsafe.Pointer(maxBatchCountField.UnsafeAddr())

	// Set values using unsafe pointer manipulation
	*(*int64)(batchSizePtr) = batchSize
	*(*int64)(batchCountPtr) = batchSize / int64(skl.MaxNodeSize)

	return nil
}

// setMeta() accesses the private field 'meta' of badgerDB's `Entry`
// badger doesn't yet allow users to explicitly set keys as *do not discard*
// https://github.com/hypermodeinc/badger/issues/2192
func setMeta(e *badger.Entry, value byte) {
	v := reflect.ValueOf(e).Elem()
	f := v.FieldByName(badgerMetaFieldName)
	ptr := unsafe.Pointer(f.UnsafeAddr())
	*(*byte)(ptr) = value
}

// getTxnFromBatch() accesses the private field 'size/count' of badgerDB's `Txn` inside a 'WriteBatch'
// badger doesn't yet allow users to access this info - though it allows users to avoid
// TxnTooBig errors
func getSizeAndCountFromBatch(batch *badger.WriteBatch) (size, count int64) {
	v := reflect.ValueOf(batch).Elem()
	f := v.FieldByName(badgerTxnFieldName)
	if f.Kind() != reflect.Ptr || f.IsNil() {
		return 0, 0
	}
	// f.Pointer() is the uintptr of the actual *Txn
	txPtr := (*badger.Txn)(unsafe.Pointer(f.Pointer()))
	return getSizeAndCount(txPtr)
}

// getSizeAndCount() accesses the private field 'size/count' of badgerDB's `Txn`
// badger doesn't yet allow users to access this info - though it allows users to avoid
// TxnTooBig errors
func getSizeAndCount(txn *badger.Txn) (size, count int64) {
	v := reflect.ValueOf(txn).Elem()
	sizeF, countF := v.FieldByName(badgerSizeFieldName), v.FieldByName(badgerCountFieldName)
	if !sizeF.IsValid() || !countF.IsValid() {
		return 0, 0
	}
	sizePtr, countPtr := unsafe.Pointer(sizeF.UnsafeAddr()), unsafe.Pointer(countF.UnsafeAddr())
	size, count = *(*int64)(sizePtr), *(*int64)(countPtr)
	return
}

// seekLast() positions the iterator at the last key for the given prefix
func seekLast(it *badger.Iterator, prefix []byte) {
	it.Seek(prefixEnd(prefix))
}
