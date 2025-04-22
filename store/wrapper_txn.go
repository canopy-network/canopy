package store

import (
	"bytes"
	"reflect"
	"unsafe"

	"github.com/canopy-network/canopy/lib"
	"github.com/dgraph-io/badger/v4"
)

const (
	// BadgerDB garbage collector behavior is not well documented leading to many open issues in their repository
	// However, here is our current understanding based on experimentation
	// ----------------------------------------------------------------------------------------------------------
	// BadgerDB Garbage Collection Rules listed by priority
	// 1. MergeOp Bit prevents garbage collection of a key regardless
	// 2. Nothing above DiscardTs will be garbage collected
	// 3. Deleting, expiring, and setting ‘Discard earlier versions’ signals the removal of older versions of a key
	// 4. Deleting or expiring a key makes it eligible for GC
	// 5. If KeepNumVersions is set to max int it will prevent automatic GC of older versions
	// ------------------------------------------------------------------------------
	// Bits source: https://github.com/hypermodeinc/badger/blob/85389e88bf308c1dc271383b77b67f4ef4a85194/value.go#L37
	badgerMetaFieldName               = "meta" // badgerDB Entry 'meta' field name
	badgerDiscardEarlierVersions byte = 1 << 2 // badgerDB 'discard earlier versions' flag
	badgerDeleteBit              byte = 1 << 0 // badgerDB 'tombstoned' flag
	badgerNoDiscardBit           byte = 1 << 3 // badgerDB 'never discard'  bit
	badgerGCRatio                     = .15    // the ratio when badgerDB will run the garbage collector
)

// RWStoreI interface enforcement
var _ lib.RWStoreI = &TxnWrapper{}

// TxnWrapper is a wrapper over the badgerDB Txn object that conforms to the RWStoreI interface
type TxnWrapper struct {
	logger lib.LoggerI
	db     *badger.Txn
	prefix []byte
}

// NewTxnWrapper() creates a new TxnWrapper with the provided params
func NewTxnWrapper(db *badger.Txn, logger lib.LoggerI, prefix []byte) *TxnWrapper {
	return &TxnWrapper{
		logger: logger,
		db:     db,
		prefix: prefix,
	}
}

// Get() retrieves the value associated with the key from the BadgerDB transaction
func (t *TxnWrapper) Get(k []byte) ([]byte, lib.ErrorI) {
	item, err := t.db.Get(lib.Append(t.prefix, k))
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

// Set() stores the key-value pair in the BadgerDB transaction
func (t *TxnWrapper) Set(k, v []byte) lib.ErrorI {
	if err := t.db.Set(lib.Append(t.prefix, k), v); err != nil {
		return ErrStoreSet(err)
	}
	return nil
}

// Delete() removes the key-value pair from the BadgerDB transaction
func (t *TxnWrapper) Delete(k []byte) lib.ErrorI {
	if err := t.db.Delete(lib.Append(t.prefix, k)); err != nil {
		return ErrStoreDelete(err)
	}
	return nil
}

// Set() stores the key-value pair in the BadgerDB transaction
func (t *TxnWrapper) SetNonPruneable(k, v []byte) lib.ErrorI {
	// set an entry with a bit that prevents it from being discarded
	if err := t.db.SetEntry(newEntry(lib.Append(t.prefix, k), v, badgerNoDiscardBit)); err != nil {
		return ErrStoreSet(err)
	}
	return nil
}

// DeleteNonPrunable() removes the key-value pair from the BadgerDB transaction but prevents it from being garbage collected
func (t *TxnWrapper) DeleteNonPrunable(k []byte) lib.ErrorI {
	// set an entry with a bit that marks it as deleted and prevents it from being discarded
	if err := t.db.SetEntry(newEntry(lib.Append(t.prefix, k), nil, badgerDeleteBit|badgerNoDiscardBit)); err != nil {
		return ErrStoreDelete(err)
	}
	return nil
}

// Close() discards the current transaction
func (t *TxnWrapper) Close()              { t.db.Discard() }
func (t *TxnWrapper) setDB(p *badger.Txn) { t.db = p }

// Iterator() creates a new iterator for the given prefix in the BadgerDB transaction
func (t *TxnWrapper) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	parent := t.db.NewIterator(badger.IteratorOptions{
		Prefix: lib.Append(t.prefix, prefix),
	})
	parent.Rewind()
	return &Iterator{
		logger: t.logger,
		parent: parent,
		prefix: t.prefix,
	}, nil
}

// RevIterator() creates a new reverse iterator for the given prefix in the BadgerDB transaction
func (t *TxnWrapper) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	newPrefix := lib.Append(t.prefix, prefix)
	parent := t.db.NewIterator(badger.IteratorOptions{
		Reverse: true,
		Prefix:  newPrefix,
	})
	seekLast(parent, newPrefix)
	return &Iterator{
		logger: t.logger,
		parent: parent,
		prefix: t.prefix,
	}, nil
}

// ArchiveIterator() creates a new iterator for all versions under the given prefix in the BadgerDB transaction
func (t *TxnWrapper) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	parent := t.db.NewIterator(badger.IteratorOptions{
		Prefix:      lib.Append(t.prefix, prefix),
		AllVersions: true,
	})
	parent.Rewind()
	return &Iterator{
		logger: t.logger,
		parent: parent,
		prefix: t.prefix,
	}, nil
}

// seekLast() positions the iterator at the last key for the given prefix
func seekLast(it *badger.Iterator, prefix []byte) {
	it.Seek(prefixEnd(prefix))
}

// IteratorI interface enforcement
var _ lib.IteratorI = &Iterator{}

// Iterator implements a wrapper around BadgerDB's iterator but satisfies the IteratorI interface
type Iterator struct {
	logger lib.LoggerI
	parent *badger.Iterator
	prefix []byte
	err    error
}

func (i *Iterator) Valid() bool     { return i.parent.Valid() }
func (i *Iterator) Next()           { i.parent.Next() }
func (i *Iterator) Close()          { i.parent.Close() }
func (i *Iterator) Version() uint64 { return i.parent.Item().Version() }
func (i *Iterator) Deleted() bool   { return i.parent.Item().IsDeletedOrExpired() }
func (i *Iterator) Key() (key []byte) {
	// get the key from the parent
	key = i.parent.Item().Key()
	// make a copy of the key
	c := make([]byte, len(key))
	copy(c, key)
	// remove the prefix and return
	return removePrefix(c, []byte(i.prefix))
}

// removePrefix() removes the prefix from the key
func removePrefix(b, prefix []byte) []byte { return b[len(prefix):] }

// Value() retrieves the current value from the iterator
func (i *Iterator) Value() (value []byte) {
	value, err := i.parent.Item().ValueCopy(nil)
	if err != nil {
		i.err = err
	}
	return
}

var (
	endBytes = bytes.Repeat([]byte{0xFF}, maxKeyBytes+1)
)

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

// setMeta() accesses the private field 'meta' of badgerDB's `Entry`
// badger doesn't yet allow users to explicitly set keys as *do not discard*
// https://github.com/hypermodeinc/badger/issues/2192
func setMeta(e *badger.Entry, value byte) {
	v := reflect.ValueOf(e).Elem()
	f := v.FieldByName(badgerMetaFieldName)
	ptr := unsafe.Pointer(f.UnsafeAddr())
	*(*byte)(ptr) = value
}
