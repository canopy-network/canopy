package store

import (
	"bytes"
	"sync"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"

	"github.com/google/btree"
)

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

const (
	opDelete op = iota // delete the key
	opSet              // set the key
	opEntry            // custom badger entry
)

// Txn is an in memory database transaction
type Txn struct {
	reader       TxnReaderI // memory store to Read() from
	writer       TxnWriterI // memory store to Flush() to
	prefix       []byte     // prefix for keys in this txn
	state        bool       // whether the flush should go to the HSS and LSS
	sort         bool       // whether to sort the keys in the txn; used for iteration
	writeVersion uint64     // the version to commit the data to
	txn          *txn       // the internal structure maintaining the write operations
}

// txn internal structure maintains the write operations sorted lexicographically by keys
type txn struct {
	ops    map[uint64]valueOp        // [string(key)] -> set/del operations saved in memory
	sorted *btree.BTreeG[*CacheItem] // sorted btree of keys for fast iteration
	rbuf   []byte                    // a buffer to re-use for reads

	l sync.Mutex // thread safety
}

// valueOp has the value portion of the operation and the corresponding operation to perform
type valueOp struct {
	key     []byte // the key of the key value pair
	value   []byte // value of key value pair
	version uint64 // version of the key value pair
	op      op     // is operation delete
}

// op is the type of operation to be performed on the key
type op uint8

// TxReaderI() defines the interface to read a TxnTransaction
// Txn implements this itself to allow for nested transactions
type TxnReaderI interface {
	Get(key []byte) ([]byte, lib.ErrorI)
	NewIterator(prefix []byte, reverse bool, allVersions bool) (lib.IteratorI, lib.ErrorI)
	Close() lib.ErrorI
}

// TxnWriterI() defines the interface to write a TxnTransaction
// Txn implements this itself to allow for nested transactions
type TxnWriterI interface {
	SetAt(key, value []byte, version uint64) lib.ErrorI
	DeleteAt(key []byte, version uint64) lib.ErrorI
	Commit() lib.ErrorI
	Close() lib.ErrorI
}

// NewBadgerTxn() creates a new instance of Txn from badger Txn and WriteBatch correspondingly
// func NewBadgerTxn(reader *badger.Txn, writer *badger.WriteBatch, prefix []byte, state, sort bool, writeVersion ...uint64) *Txn {
// 	var writerI TxnWriterI
// 	// prevent nil interface containing nil value
// 	if writer != nil {
// 		writerI = writer
// 	}
// 	// exit with the new transaction
// 	return NewTxn(BadgerTxnReader{reader, prefix}, writerI, prefix, state, sort, writeVersion...)
// }

// NewTxn() creates a new instance of Txn with the specified reader and writer
func NewTxn(reader TxnReaderI, writer TxnWriterI, prefix []byte, state, sort bool, version ...uint64) *Txn {
	var v uint64
	if len(version) != 0 {
		v = version[0]
	}
	return &Txn{
		reader:       reader,
		writer:       writer,
		prefix:       prefix,
		state:        state,
		sort:         sort,
		writeVersion: v,
		txn: &txn{
			ops:    make(map[uint64]valueOp),
			sorted: btree.NewG(32, func(a, b *CacheItem) bool { return a.Less(b) }), // need to benchmark this value
			l:      sync.Mutex{},
		},
	}
}

// Get() retrieves the value for a given key from either the txn operations or the reader store
func (t *Txn) Get(key []byte) ([]byte, lib.ErrorI) {
	t.txn.l.Lock()
	defer t.txn.l.Unlock()
	// first retrieve from the in-memory txn
	if v, found := t.txn.ops[lib.MemHash(key)]; found {
		return v.value, nil
	}
	// if not found, retrieve from the parent reader
	return t.reader.Get(lib.AppendWithBuffer(&t.txn.rbuf, t.prefix, key))
}

// Set() adds or updates the value for a key in the txn operations
func (t *Txn) Set(key, value []byte) lib.ErrorI {
	return t.update(key, value, t.writeVersion, opSet)
}

// Delete() marks a key for deletion in the txn operations
func (t *Txn) Delete(key []byte) lib.ErrorI {
	return t.update(key, nil, t.writeVersion, opDelete)
}

// SetAt() adds or updates the value for a key in the txn operations with a specific version
func (t *Txn) SetAt(key, value []byte, version uint64) lib.ErrorI {
	return t.update(key, value, version, opSet)
}

// DeleteAt() marks a key for deletion in the txn operations with a specific version
func (t *Txn) DeleteAt(key []byte, version uint64) lib.ErrorI {
	return t.update(key, nil, version, opDelete)
}

// update() modifies or adds an operation to the txn
func (t *Txn) update(key []byte, val []byte, version uint64, opAction op) (e lib.ErrorI) {
	hashedKey := lib.MemHash(key)
	t.txn.l.Lock()
	defer t.txn.l.Unlock()
	if _, found := t.txn.ops[hashedKey]; !found && t.sort {
		t.addToSorted(key, hashedKey)
	}
	t.txn.ops[hashedKey] = valueOp{key: key, value: val, version: version, op: opAction}
	return
}

// addToSorted() inserts a key into the sorted list of operations maintaining lexicographical order
func (t *Txn) addToSorted(key []byte, hashedKey uint64) {
	t.txn.sorted.ReplaceOrInsert(&CacheItem{Key: key, HashedKey: hashedKey, Exists: true})
}

// Iterator() returns a new iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it, err := t.reader.NewIterator(lib.Append(t.prefix, prefix), false, false)
	if err != nil {
		return nil, err
	}
	return newTxnIterator(it, t.txn.copy(), prefix, t.prefix, false), nil
}

// RevIterator() returns a new reverse iterator for merged iteration of both the in-memory operations and parent store with the given prefix
func (t *Txn) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	it, err := t.reader.NewIterator(lib.Append(t.prefix, prefix), true, false)
	if err != nil {
		return nil, err
	}
	return newTxnIterator(it, t.txn.copy(), prefix, t.prefix, true), nil
}

// ArchiveIterator() creates a new iterator for all versions under the given prefix in the BadgerDB transaction
func (t *Txn) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return t.reader.NewIterator(lib.Append(t.prefix, prefix), false, true)
}

// Discard() clears all in-memory operations and resets the sorted key list
func (t *Txn) Discard() {
	t.txn.sorted.Clear(false)
	t.txn.ops = make(map[uint64]valueOp)
}

// Closes() Closes the writer. Any new writes will result in error and possibly panic depending on the implementation.
func (t *Txn) Cancel() {
	if t.writer != nil {
		t.writer.Close()
	}
}

// Close() cancels the current transaction. Any new writes will result in an error and a new
// WriteBatch() must be created to write new entries.
func (t *Txn) Close() lib.ErrorI {
	if t.reader != nil {
		if err := t.reader.Close(); err != nil {
			return err
		}
	}
	// the same underlying struct can satisfy both interfaces, on such case Close() is only called
	// once to prevent double closing
	if any(t.reader) == any(t.writer) { // compares the pointers to check if they are the same
		return nil
	}
	if t.writer != nil {
		if err := t.writer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Commit() commits the in-memory operations to the batch writer and clears in-memory changes
func (t *Txn) Commit() (err lib.ErrorI) {
	t.txn.l.Lock()
	defer func() { t.txn.l.Unlock(); t.Discard() }()
	for version, prefix := range t.flushTo() {
		if err = t.flush(prefix, version); err != nil {
			return err
		}
	}
	// exit
	return nil
}

// flushTo() returns the prefixes to flush to; state uses special logic to flush
func (t *Txn) flushTo() map[uint64][]byte {
	if t.state {
		return map[uint64][]byte{lssVersion: []byte(latestStatePrefix), t.writeVersion: []byte(historicStatePrefix)}
	}
	return map[uint64][]byte{t.writeVersion: t.prefix}
}

// flush() flushes all operations to the underlying writer with a prefix if given
func (t *Txn) flush(prefix []byte, writeVersion uint64) (err lib.ErrorI) {
	for _, v := range t.txn.ops {
		if err = t.write(prefix, writeVersion, v); err != nil {
			return err
		}
	}
	return
}

// write() writes a value operation to the underlying writer
func (t *Txn) write(prefix []byte, writeVersion uint64, op valueOp) lib.ErrorI {
	k := op.key
	if prefix != nil {
		k = lib.Append(prefix, k)
	}
	switch op.op {
	case opSet:
		if err := t.writer.SetAt(k, op.value, writeVersion); err != nil {
			return ErrStoreSet(err)
		}
	case opDelete:
		if err := t.writer.DeleteAt(k, writeVersion); err != nil {
			return ErrStoreDelete(err)
		}
	}
	return nil
}

// NewIterator() creates a merged iterator with the reader and writer
func (t *Txn) NewIterator(prefix []byte, reverse bool, allVersions bool) (lib.IteratorI, lib.ErrorI) {
	// create an iterator for the parent
	parentIterator, err := t.reader.NewIterator(lib.Append(t.prefix, prefix), reverse, allVersions)
	if err != nil {
		return nil, err
	}
	// create a merged iterator for the parent and in-memory txn
	return newTxnIterator(parentIterator, t.txn, prefix, t.prefix, reverse), nil
}

// Copy creates a new Txn with the same configuration and txn as the original
func (t *Txn) Copy(reader TxnReaderI, writer TxnWriterI) *Txn {
	return &Txn{
		reader:       reader,
		writer:       writer,
		prefix:       t.prefix,
		state:        t.state,
		sort:         t.sort,
		writeVersion: t.writeVersion,
		txn:          t.txn.copy(),
	}
}

// txn() returns a copy of the current transaction txn
func (t *txn) copy() *txn {
	t.l.Lock()
	defer t.l.Unlock()
	ops := make(map[uint64]valueOp, len(t.ops))
	for k, o := range t.ops {
		ops[k] = o.copy()
	}
	return &txn{
		ops:    ops,
		sorted: t.sorted.Clone(),
		l:      sync.Mutex{},
	}
}

// copy() deep copies a value operation
func (o valueOp) copy() valueOp {
	return valueOp{
		key:     bytes.Clone(o.key),
		value:   bytes.Clone(o.value),
		version: o.version,
		op:      o.op,
	}
}

// TXN ITERATOR CODE BELOW

// enforce the Iterator interface
var _ lib.IteratorI = &TxnIterator{}

// TxnIterator is a reversible, merged iterator of the parent and the in-memory operations
type TxnIterator struct {
	parent lib.IteratorI
	tree   *BTreeIterator
	*txn
	hasNext      bool
	prefix       []byte
	parentPrefix []byte
	reverse      bool
	invalid      bool
	useTxn       bool
}

// newTxnIterator() initializes a new merged iterator for traversing both the in-memory operations and parent store
func newTxnIterator(parent lib.IteratorI, t *txn, prefix []byte, parentPrefix []byte, reverse bool) *TxnIterator {
	tree := NewBTreeIterator(t.sorted.Clone(),
		&CacheItem{
			Key: prefix,
		},
		reverse)

	return (&TxnIterator{
		parent:       parent,
		tree:         tree,
		txn:          t,
		prefix:       prefix,
		parentPrefix: parentPrefix,
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
	switch ti.compare(ti.txnKey(), removePrefix(ti.parent.Key(), ti.parentPrefix)) {
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
	return removePrefix(ti.parent.Key(), ti.parentPrefix)
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
			// only using txn; call txn.next until invalid or !deleted
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
		cKey, pKey := ti.txnKey(), removePrefix(ti.parent.Key(), ti.parentPrefix)
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

// Close() closes the merged iterator
func (ti *TxnIterator) Close() { ti.parent.Close() }

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
	if current == nil || len(current.Key) == 0 {
		ti.invalid = true
		return ti.invalid
	}
	if !bytes.HasPrefix(current.Key, ti.prefix) {
		return ti.invalid
	}
	ti.invalid = false
	return ti.invalid
}

// txnKey() returns the key of the current in-memory operation
func (ti *TxnIterator) txnKey() []byte {
	return ti.tree.Current().Key
}

// txnValue() returns the value of the current in-memory operation
func (ti *TxnIterator) txnValue() valueOp {
	ti.l.Lock()
	defer ti.l.Unlock()
	return ti.ops[ti.tree.Current().HashedKey]
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
	ti.tree.Move(&CacheItem{Key: ti.prefix})
	return ti
}

// revSeek() positions the iterator at the last entry that matches the prefix in reverse order.
func (ti *TxnIterator) revSeek() *TxnIterator {
	ti.tree.Move(&CacheItem{Key: prefixEnd(ti.prefix)})
	return ti
}

// Iterator implements a wrapper around BadgerDB's iterator with the in-memory store but satisfies the IteratorI interface
type Iterator struct {
	reader *badger.Iterator
	prefix []byte
	err    error
}

func (i *Iterator) Valid() bool     { return i.reader.Valid() }
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

// removePrefix() removes the prefix from the key
func removePrefix(b, prefix []byte) []byte { return b[len(prefix):] }

// prefixEnd() returns the end key for a given prefix by appending max possible bytes
func prefixEnd(prefix []byte) []byte { return lib.Append(prefix, endBytes) }

var endBytes = bytes.Repeat([]byte{0xFF}, maxKeyBytes+1)

// BTREE ITERATOR CODE BELOW

type CacheItem struct {
	HashedKey uint64
	Key       []byte
	Exists    bool
}

// Less() compares the keys lexicographically
func (ti CacheItem) Less(than *CacheItem) bool { return bytes.Compare(ti.Key, than.Key) < 0 }

// BTreeIterator provides external iteration over a btree
type BTreeIterator struct {
	tree    *btree.BTreeG[*CacheItem] // the btree to iterate over
	current *CacheItem                // current item in the iteration
	reverse bool                      // whether the iteration is in reverse order
}

// NewBTreeIterator() creates a new iterator starting at the closest item to the given key
func NewBTreeIterator(tree *btree.BTreeG[*CacheItem], start *CacheItem, reverse bool) (bt *BTreeIterator) {
	// create a new BTreeIterator
	bt = &BTreeIterator{
		tree:    tree,
		reverse: reverse,
	}
	// if no start item is provided, set the iterator to the first or last item based on the direction
	if start == nil || len(start.Key) == 0 {
		if reverse {
			bt.current, _ = tree.Max()
		} else {
			bt.current, _ = tree.Min()
		}
		return
	}
	// otherwise, move the iterator to that item
	bt.Move(start)
	return
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
		bi.current = &CacheItem{Key: append(item.Key, endBytes...)}
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
		return &CacheItem{Key: []byte{}, Exists: false}
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
		if !bytes.Equal(nextItem.Key, bi.current.Key) {
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

// prev() finds the previous item in the tree based on the current item
func (bi *BTreeIterator) prev() *CacheItem {
	var prev *CacheItem
	bi.tree.DescendLessOrEqual(bi.current, func(item *CacheItem) bool {
		if item.Less(bi.current) {
			prev = item
			return false // stop iteration
		}
		return true // continue iteration
	})
	return prev
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
