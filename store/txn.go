package store

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"

	"maps"

	"github.com/google/btree"
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
	SetEntryAt(entry *badger.Entry, version uint64) error
	Flush() error
	Cancel()
}

// enforce the necessary interfaces over Txn
var _ lib.RWStoreI = &Txn{}
var _ TxnReaderI = &Txn{}
var _ TxnWriterI = &Txn{}
var _ lib.IteratorI = &Iterator{}

/*
	Txn acts like a database transaction
	It saves set/del operations in memory and allows the caller to Flush() to the parent or Discard()
	When read from, it merges with the parent as if Flush() had already been called

	Txn abstraction is necessary due to the inability of BadgerDB to have nested transactions.
	Txns allow an easy rollback of write operations within a single Transaction object, which is necessary
	for ephemeral states and testing the validity of a proposal block / transactions.

	CONTRACT:
	- only safe when writing to another memory store like a badger.Txn() as Flush() is not atomic.
	- not thread safe (can't use 1 txn across multiple threads)
	- nil values are supported; deleted values are also set to nil
	- keys must be smaller than 128 bytes
	- Nested txns are supported, but iteration becomes increasingly inefficient
*/

type Txn struct {
	reader       TxnReaderI  // memory store to Read() from
	writer       TxnWriterI  // memory store to Flush() to
	prefix       []byte      // prefix for keys in this txn
	logger       lib.LoggerI // logger for this txn
	sort         bool        // whether to sort the keys in the cache; used for iteration
	writeVersion uint64      // the version to commit the data to
	cache        txn

	// liveWrite enables real-time writing of key-values to the writer as they're set,
	// rather than batching them at commit time. While this adds minimal overhead to
	// individual set operations, it can significantly improve overall performance when
	// calling Flush() to persist data to the writer by spreading the I/O cost over time.
	// Fields supporting live writing functionality
	liveWrite bool
	mu        sync.Mutex
	wg        sync.WaitGroup
	queue     []valueOp
	sender    chan valueOp
	err       error
}

// txn internal structure maintains the write operations sorted lexicographically by keys
type txn struct {
	ops       map[string]valueOp        // [string(key)] -> set/del operations saved in memory
	sorted    *btree.BTreeG[*CacheItem] // sorted btree of keys for fast iteration
	sortedLen int                       // len(sorted)
}

// txn() returns a copy of the current transaction cache
func (t txn) copy() txn {
	ops := make(map[string]valueOp, t.sortedLen)
	maps.Copy(ops, t.ops)
	return txn{
		ops:       ops,
		sorted:    t.sorted.Clone(),
		sortedLen: t.sortedLen,
	}
}

// op is the type of operation to be performed on the key
type op uint8

const (
	opDelete op = iota // delete the key
	opSet              // set the key
	opEntry            // custom badger entry
)

// valueOp has the value portion of the operation and the corresponding operation to perform
type valueOp struct {
	key        []byte        // the key of the key value pair
	value      []byte        // value of key value pair
	valueEntry *badger.Entry // value of key value pair in case of a custom entry
	op         op            // is operation delete
}

// NewBadgerTxn() creates a new instance of Txn from badger Txn and WriteBatch correspondingly
func NewBadgerTxn(reader *badger.Txn, writer *badger.WriteBatch, prefix []byte, sort bool, writeVersion uint64, liveWrite bool, logger lib.LoggerI) *Txn {
	return NewTxn(BadgerTxnReader{reader, prefix}, writer, prefix, sort, writeVersion, liveWrite, logger)
}

// NewTxn() creates a new instance of Txn with the specified reader and writer
func NewTxn(reader TxnReaderI, writer TxnWriterI, prefix []byte, sort bool, version uint64, liveWrite bool, logger lib.LoggerI) *Txn {
	txn := &Txn{
		reader:       reader,
		writer:       writer,
		prefix:       prefix,
		logger:       logger,
		sort:         sort,
		writeVersion: version,
		cache: txn{
			ops:    make(map[string]valueOp),
			sorted: btree.NewG(32, func(a, b *CacheItem) bool { return a.Less(b) }), // need to benchmark this value
		},
	}

	if liveWrite {
		txn.liveWrite = liveWrite
		txn.queue = []valueOp{}
		txn.sender = make(chan valueOp, 1000)
		txn.mu = sync.Mutex{}
		txn.wg = sync.WaitGroup{}
		txn.liveWriter()
	}

	return txn
}

// liveWriter is a background goroutine that writes the operations to the writer as soon
// as they are received
func (t *Txn) liveWriter() {
	// processQueue processes the next operation in the queue
	processQueue := func() {
		if len(t.queue) > 0 {
			valueOp := t.queue[0]
			t.queue = t.queue[1:]
			if err := t.processOperation(valueOp); err != nil {
				t.err = errors.Join(t.err, err)
			}
			t.wg.Done()
		}
	}
	go func() {
		for {
			// check the queue for operations
			t.mu.Lock()
			processQueue()
			t.mu.Unlock()
			// wait for the next operation to be sent to the channel
			valueOp, ok := <-t.sender
			if !ok {
				break
			}
			t.processOperation(valueOp)
			t.wg.Done()
		}
		// process any remaining operations in the queue
		t.mu.Lock()
		for len(t.queue) > 0 {
			processQueue()
		}
		t.mu.Unlock()
	}()
}

// sendToLiveWriter sends a value operation to the live writer channel and queues it
// if the channel is full. This allow operations to be non blocking
func (t *Txn) sendToLiveWriter(valueOp valueOp) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.wg.Add(1)
	// try sending to the channel
	select {
	case t.sender <- valueOp:
	default:
		// send to queue if channel is full
		t.queue = append(t.queue, valueOp)
	}
}

// Get() retrieves the value for a given key from either the cache operations or the reader store
func (t *Txn) Get(key []byte) ([]byte, lib.ErrorI) {
	// append the prefix to the key
	prefixedKey := lib.Append(t.prefix, key)
	// first retrieve from the in-memory cache
	if v, found := t.cache.ops[string(prefixedKey)]; found {
		return v.value, nil
	}
	// if not found, retrieve from the parent reader
	return t.reader.Get(prefixedKey)
}

// Set() adds or updates the value for a key in the cache operations
func (t *Txn) Set(key, value []byte) lib.ErrorI {
	t.update(lib.Append(t.prefix, key), value, opSet)
	return nil
}

// Delete() marks a key for deletion in the cache operations
func (t *Txn) Delete(key []byte) lib.ErrorI {
	t.update(lib.Append(t.prefix, key), nil, opDelete)
	return nil
}

// SetEntry() adds or updates a custom badger entry in the cache operations
func (t *Txn) SetEntryAt(entry *badger.Entry, _ uint64) error {
	t.updateEntry(lib.Append(t.prefix, entry.Key), entry)
	return nil
}

// update() modifies or adds an operation for a key in the cache operations and maintains the
// lexicographical order.
// NOTE: update() won't modify the key itself, any key prefixing must be done before calling this
func (t *Txn) update(key []byte, v []byte, opAction op) {
	k := string(key)
	if t.sort {
		if _, found := t.cache.ops[k]; !found && t.sort {
			t.addToSorted(string(key))
		}
	}
	valueOp := valueOp{key: key, value: v, op: opAction}
	t.cache.ops[k] = valueOp

	if t.liveWrite {
		t.sendToLiveWriter(valueOp)
	}
}

// updateEntry() modifies or adds a custom badger entry in the cache operations and maintains the
// lexicographical order.
// NOTE: updateEntry() won't modify the key itself, any key prefixing must be done before calling this
func (t *Txn) updateEntry(key []byte, v *badger.Entry) {
	k := string(key)
	if t.sort {
		if _, found := t.cache.ops[k]; !found && t.sort {
			t.addToSorted(string(key))
		}
	}

	valueOp := valueOp{key: key, value: v.Value, valueEntry: v, op: opEntry}
	t.cache.ops[k] = valueOp

	if t.liveWrite {
		t.sendToLiveWriter(valueOp)
	}
}

// addToSorted() inserts a key into the sorted list of operations maintaining lexicographical order
func (t *Txn) addToSorted(key string) {
	t.cache.sortedLen++
	t.cache.sorted.ReplaceOrInsert(&CacheItem{Key: key, Exists: true})
}

// Iterator() returns a new iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it := t.reader.NewIterator(prefix, false, false)
	return newTxnIterator(it, t.cache.copy(), t.prefix, prefix, false), nil
}

// RevIterator() returns a new reverse iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it := t.reader.NewIterator(prefix, true, false)
	return newTxnIterator(it, t.cache.copy(), t.prefix, prefix, true), nil
}

// ArchiveIterator() creates a new iterator for all versions under the given prefix in the BadgerDB transaction
func (t *Txn) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return t.reader.NewIterator(prefix, false, true), nil
}

// Discard() clears all in-memory operations and resets the sorted key list
func (t *Txn) Discard() {
	t.cache.sorted.Clear(false)
	t.cache.ops, t.cache.sortedLen = make(map[string]valueOp), 0
}

// Cancel() cancels the current transaction and clears the live writer queue if enabled.
// Any new writes won't be committed
func (t *Txn) Cancel() {
	if t.liveWrite {
		t.liveWrite = false
		close(t.sender)
		// clear the queue and decrement the wait group for each pending operation
		t.mu.Lock()
		pendingOps := len(t.queue)
		t.queue = make([]valueOp, 0)
		t.mu.Unlock()
		// decrement waitgroup for all pending operations
		for range pendingOps {
			t.wg.Done()
		}
	}
	if t.writer != nil {
		// close the writer
		t.writer.Cancel()
	}
}

// Flush() flushes the in-memory operations to the batch writer and clears in-memory changes
func (t *Txn) Flush() error {
	if t.liveWrite {
		// set live writer to false to prevent further writes
		t.liveWrite = false
		// close sender channel to signal end of operations
		close(t.sender)
		// wait for all pending operations to complete
		t.wg.Wait()
		// clear the in-memory operations after writing
		t.Discard()
		// return any errors encountered during the write process
		if t.err != nil {
			return t.err.(lib.ErrorI)
		}
		// no need to process t.cache.ops again since they have already been applied
		return nil
	}
	// flush all operations to the writer
	for _, v := range t.cache.ops {
		if err := t.processOperation(v); err != nil {
			return err
		}
	}
	// clear the in-memory operations after writing
	t.Discard()
	// exit
	return nil
}

func (t *Txn) processOperation(vOp valueOp) lib.ErrorI {
	switch vOp.op {
	case opSet:
		// set an entry with a bit that marks it as deleted and prevents it from being discarded
		if err := t.writer.SetEntryAt(&badger.Entry{Key: vOp.key, Value: vOp.value}, t.writeVersion); err != nil {
			return ErrStoreDelete(err)
		}
	case opDelete:
		// set an entry with a bit that marks it as deleted and prevents it from being discarded
		if err := t.writer.SetEntryAt(newEntry(vOp.key, nil, badgerDeleteBit|badgerNoDiscardBit), t.writeVersion); err != nil {
			return ErrStoreDelete(err)
		}
	case opEntry:
		// set the entry in the batch
		if err := t.writer.SetEntryAt(vOp.valueEntry, t.writeVersion); err != nil {
			return ErrStoreSet(err)
		}
	}

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
	t.Cancel()
}

// TXN ITERATOR CODE BELOW

// enforce the Iterator interface
var _ lib.IteratorI = &TxnIterator{}

// TxnIterator is a reversible, merged iterator of the parent and the in-memory operations
type TxnIterator struct {
	parent lib.IteratorI
	tree   *BTreeIterator
	txn
	hasNext      bool
	prefix       string
	parentPrefix string
	index        int
	reverse      bool
	invalid      bool
	useTxn       bool
}

// newTxnIterator() initializes a new merged iterator for traversing both the in-memory operations and parent store
func newTxnIterator(parent lib.IteratorI, t txn, parentPrefix, prefix []byte, reverse bool) *TxnIterator {
	tree := NewBTreeIterator(t.sorted.Clone(),
		&CacheItem{
			Key: string(lib.Append(parentPrefix, prefix)),
		},
		reverse)

	return (&TxnIterator{
		parent:       parent,
		tree:         tree,
		txn:          t,
		parentPrefix: string(parentPrefix),
		prefix:       string(prefix),
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
			if ti.txnValue().op == opDelete {
				ti.parent.Next()
				ti.txnNext()
				continue
			}
			ti.useTxn = true
		case -1: // use txn
			if ti.txnValue().op == opDelete {
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
		if ti.txnInvalid() || !(ti.txnValue().op == opDelete) {
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
	current := ti.tree.Current()
	if current == nil || current.Key == "" {
		ti.invalid = true
		return ti.invalid
	}
	if !strings.HasPrefix(ti.tree.Current().Key, ti.parentPrefix+ti.prefix) {
		return ti.invalid
	}
	ti.invalid = false
	return ti.invalid
}

// txnKey() returns the key of the current in-memory operation
func (ti *TxnIterator) txnKey() []byte {
	return []byte(strings.TrimPrefix(ti.tree.Current().Key, ti.parentPrefix))
}

// txnValue() returns the value of the current in-memory operation
func (ti *TxnIterator) txnValue() valueOp {
	return ti.ops[ti.tree.Current().Key]
}

// compare() compares two byte slices, adjusting for reverse iteration if needed
func (ti *TxnIterator) compare(a, b []byte) int {
	if ti.reverse {
		return bytes.Compare(a, b) * -1
	}
	return bytes.Compare(a, b)
}

// txnNext() advances the index of the in-memory operations based on the iteration direction
func (ti *TxnIterator) txnNext() {
	ti.hasNext = ti.tree.HasNext()
	ti.tree.Next()
}

// seek() positions the iterator at the first entry that matches or exceeds the prefix.
func (ti *TxnIterator) seek() *TxnIterator {
	ti.tree.Move(&CacheItem{
		Key: ti.parentPrefix + ti.prefix,
	})
	return ti
}

// revSeek() positions the iterator at the last entry that matches the prefix in reverse order.
func (ti *TxnIterator) revSeek() *TxnIterator {
	bz := []byte(ti.parentPrefix + ti.prefix)
	endPrefix := string(prefixEnd(bz))
	ti.tree.Move(&CacheItem{
		Key: endPrefix,
	})
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
	badgerDeleteBit               byte = 1 << 0  // badgerDB 'tombstoned' flag
	badgerNoDiscardBit            byte = 1 << 3  // badgerDB 'never discard'  bit
	badgerSizeFieldName                = "size"  // badgerDB Txn 'size' field name
	badgerCountFieldName               = "count" // badgerDB Txn 'count' field name
	badgerTxnFieldName                 = "txn"   // badgerDB WriteBatch 'txn' field name
	badgerDBMaxBatchScalingFactor      = 0.98425 // through experimentation badgerDB's max transaction scaling factor
)

// Enforce interface implementations
var _ TxnReaderI = &BadgerTxnReader{}

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

func (r BadgerTxnReader) Discard() { r.Txn.Discard() }

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
func seekLast(it *badger.Iterator, prefix []byte) { it.Seek(prefixEnd(prefix)) }

var (
	endBytes = bytes.Repeat([]byte{0xFF}, maxKeyBytes+1)
)

// removePrefix() removes the prefix from the key
func removePrefix(b, prefix []byte) []byte { return b[len(prefix):] }

// prefixEnd() returns the end key for a given prefix by appending max possible bytes
func prefixEnd(prefix []byte) []byte { return lib.Append(prefix, endBytes) }

// newEntry() creates a new badgerDB entry
func newEntry(key, value []byte, meta byte) (e *badger.Entry) {
	e = &badger.Entry{Key: key, Value: value}
	setMeta(e, meta)
	return
}

// BTREE ITERATOR CODE BELOW

type CacheItem struct {
	Key    string
	Exists bool
}

func (ti CacheItem) Less(than *CacheItem) bool {
	// compare the keys lexicographically
	return ti.Key < than.Key
}

// BTreeIterator provides external iteration over a btree
type BTreeIterator struct {
	tree    *btree.BTreeG[*CacheItem] // the btree to iterate over
	current *CacheItem                // current item in the iteration
	reverse bool                      // whether the iteration is in reverse order
}

// NewBTreeIterator() creates a new iterator starting at the closest item to the given key
func NewBTreeIterator(tree *btree.BTreeG[*CacheItem], start *CacheItem, reverse bool) *BTreeIterator {
	// create a new BTreeIterator
	bt := &BTreeIterator{
		tree:    tree,
		reverse: reverse,
	}
	// if no start item is provided, set the iterator to the first or last item based on the direction
	if start == nil || start.Key == "" {
		if reverse {
			val, _ := tree.Max()
			bt.current = val
		} else {
			val, _ := tree.Min()
			bt.current = val
		}
		return bt
	}
	// otherwise, move the iterator to that item
	bt.Move(start)
	return bt
}

// Move() moves the iterator to the given key or the closest item if the key is not found
func (bi *BTreeIterator) Move(item *CacheItem) {
	// reset the current item
	bi.current = nil
	// try to get an exact match
	if exactMatch, ok := bi.tree.Get(item); ok {
		bi.current = exactMatch
		return
	}
	// if no exact match, find the closest item based on the direction of iteration
	if bi.reverse {
		bi.current = &CacheItem{Key: item.Key + string(endBytes)}
		bi.current = bi.prev()
	} else {
		bi.current = &CacheItem{Key: item.Key}
		bi.current = bi.next()
	}
}

// Current() returns the current item in the iteration
func (bi *BTreeIterator) Current() *CacheItem {
	// if current is nil, return an empty Item to avoid nil pointer dereference
	if bi.current == nil {
		return &CacheItem{Key: "", Exists: false}
	}
	return bi.current
}

// Next() advances to the next item in the tree
func (bi *BTreeIterator) Next() *CacheItem {
	// check if current exist, otherwise the iterator is invalid
	if bi.current == nil {
		return nil
	}
	// go to the next item based on the direction of iteration
	if bi.reverse {
		bi.current = bi.prev()
	} else {
		bi.current = bi.next()
	}
	// return the current item which is the possible next item in the iteration
	return bi.Current()
}

// next() finds the next item in the tree based on the current item
func (bi *BTreeIterator) next() *CacheItem {
	var nextItem *CacheItem
	var found bool
	// find the next item
	bi.tree.AscendGreaterOrEqual(bi.current, func(item *CacheItem) bool {
		nextItem = item
		if nextItem.Key != bi.current.Key {
			found = true
			return false
		}
		return true
	})
	// if the item found, return it
	if found {
		return nextItem
	}
	// no next item
	return nil
}

// Prev() back towards the previous item in the tree
func (bi *BTreeIterator) Prev() *CacheItem {
	// check if current exist, otherwise the iterator is invalid
	if bi.current == nil {
		return nil
	}
	// go to the previous item based on the direction of iteration
	if bi.reverse {
		bi.current = bi.next()
	} else {
		bi.current = bi.prev()
	}
	// return the current item which is the possible previous item in the iteration
	return bi.Current()
}

// next() finds the previous item in the tree based on the current item
func (bi *BTreeIterator) prev() *CacheItem {
	var prevItem *CacheItem
	var found bool
	// find the previous item
	bi.tree.DescendLessOrEqual(bi.current, func(item *CacheItem) bool {
		prevItem = item
		if prevItem.Less(bi.current) {
			found = true
			return false
		}
		return true
	})
	// if the item found, return it
	if found {
		return prevItem
	}
	// no previous item
	return nil
}

// HasNext() returns true if there are more items after current
func (bi *BTreeIterator) HasNext() bool {
	if bi.reverse {
		return bi.hasPrev()
	}
	return bi.hasNext()
}

// hasNext() checks if there is a next item in the iteration
func (bi *BTreeIterator) hasNext() bool {
	if bi.current == nil {
		return false
	}
	return bi.next() != nil
}

// HasPrev() returns true if there are items before current
func (bi *BTreeIterator) HasPrev() bool {
	if bi.reverse {
		return bi.hasNext()
	}
	return bi.hasPrev()
}

// hasPrev() checks if there is a previous item in the iteration
func (bi *BTreeIterator) hasPrev() bool {
	if bi.current == nil {
		return false
	}
	return bi.prev() != nil
}
