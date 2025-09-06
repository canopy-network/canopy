package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/canopy-network/canopy/lib"
	"github.com/cockroachdb/pebble/v2"
)

/* versioned_store.go implements a multi-version store in pebble db*/

// key layout: [UserKey][8-byte InvertedVersion]
// value layout: [1-byte Tombstone][ActualValue]
// InvertedVersion = ^version to make newer versions sort first lexicographically

const (
	VersionSize    = 8
	DeadTombstone  = byte(1)
	AliveTombstone = byte(0)
)

// VersionedStore uses inverted version encoding and reverse seeks for maximum performance
type VersionedStore struct {
	db        *pebble.DB
	batch     *pebble.Batch
	version   uint64
	keyBuffer []byte
}

// NewVersionedStore creates a new  versioned store
func NewVersionedStore(db *pebble.DB, batch *pebble.Batch, version uint64) (*VersionedStore, lib.ErrorI) {
	return &VersionedStore{db: db, batch: batch, version: version, keyBuffer: make([]byte, 0, 256)}, nil
}

// Set() stores a key-value pair at the current version
func (vs *VersionedStore) Set(key, value []byte) (err lib.ErrorI) {
	k := vs.makeVersionedKey(key, vs.version)
	v := vs.valueWithTombstone(AliveTombstone, value)
	if e := vs.batch.Set(k, v, nil); e != nil {
		return ErrStoreSet(e)
	}
	return
}

// Delete() marks a key as deleted at the current version
func (vs *VersionedStore) Delete(key []byte) (err lib.ErrorI) {
	k := vs.makeVersionedKey(key, vs.version)
	v := vs.valueWithTombstone(DeadTombstone, nil)
	if e := vs.batch.Set(k, v, nil); e != nil {
		return ErrStoreDelete(e)
	}
	return
}

// Get() retrieves the latest version of a key using reverse seek
func (vs *VersionedStore) Get(key []byte) ([]byte, lib.ErrorI) {
	key, _, err := vs.get(key)
	return key, err
}

// get()  retrieves the latest version of a key using reverse seek
func (vs *VersionedStore) get(key []byte) (value []byte, tombstone byte, err lib.ErrorI) {
	var seekKey = key
	if vs.version != math.MaxUint64 {
		seekKey = vs.makeVersionedKey(key, vs.version+1)
	}
	// create a new iterator
	i, err := vs.newVersionedIterator(key, true)
	if err != nil {
		return nil, 0, err
	}
	defer i.Close()
	// position iterator
	iter := i.iter
	if !i.iter.SeekLT(seekKey) {
		i.iter.SeekGE(key)
	}
	// find latest version ≤ version
	for ; i.iter.Valid(); i.iter.Next() {
		userKey, ver, e := vs.parseVersionedKey(iter.Key())
		if e != nil || !bytes.Equal(userKey, key) || ver > vs.version {
			continue
		}
		// parse the value to extract tombstone and actual value
		tombstone, value = vs.parseValueWithTombstone(iter.Value())
		// exit
		return
	}
	return nil, 0, nil
}

// Commit commits the batch to the database
func (vs *VersionedStore) Commit() (e lib.ErrorI) {
	if err := vs.batch.Commit(&pebble.WriteOptions{Sync: false}); err != nil {
		return ErrCommitDB(err)
	}
	return
}

// Close closes the store and releases resources
func (vs *VersionedStore) Close() lib.ErrorI { return nil }

// Iterator returns an iterator for all keys with the given prefix
func (vs *VersionedStore) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(prefix, false)
}

// RevIterator returns a reverse iterator for all keys with the given prefix
func (vs *VersionedStore) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(prefix, true)
}

// newVersionedIterator creates a new  versioned iterator
func (vs *VersionedStore) newVersionedIterator(prefix []byte, reverse bool) (*VersionedIterator, lib.ErrorI) {
	var (
		err  error
		iter *pebble.Iterator
		opts = &pebble.IterOptions{LowerBound: prefix, UpperBound: prefixEnd(prefix)}
	)
	if vs.batch != nil && vs.batch.Indexed() {
		iter, err = vs.batch.NewIter(opts)
	} else {
		iter, err = vs.db.NewIter(opts)
	}
	if iter == nil || err != nil {
		return nil, ErrStoreGet(fmt.Errorf("failed to create iterator: %v", err))
	}
	return &VersionedIterator{
		iter:    iter,
		store:   vs,
		prefix:  prefix,
		reverse: reverse,
	}, nil
}

// VersionedIterator implements  iteration with single-pass key deduplication
type VersionedIterator struct {
	iter        *pebble.Iterator
	store       *VersionedStore
	prefix      []byte
	reverse     bool
	key         []byte
	value       []byte
	isValid     bool
	initialized bool
	lastUserKey []byte
}

// Valid returns true if the iterator is positioned at a valid entry
func (vi *VersionedIterator) Valid() bool {
	if !vi.initialized {
		vi.first()
	}
	return vi.isValid
}

// Next() advances the iterator to the next entry
func (vi *VersionedIterator) Next() {
	if !vi.initialized {
		vi.first()
		return
	}
	vi.advanceToNextKey()
}

// Key() returns the current key (without version/tombstone suffix)
func (vi *VersionedIterator) Key() []byte {
	if !vi.isValid {
		return nil
	}
	return bytes.Clone(vi.key)
}

// Value() returns the current value
func (vi *VersionedIterator) Value() []byte {
	if !vi.isValid {
		return nil
	}
	return bytes.Clone(vi.value)
}

// Close() closes the iterator
func (vi *VersionedIterator) Close() { _ = vi.iter.Close() }

// first() positions the iterator at the first valid entry
func (vi *VersionedIterator) first() {
	vi.initialized = true
	// seek to proper position
	if !vi.reverse {
		if len(vi.prefix) == 0 {
			vi.iter.First()
		} else {
			vi.iter.SeekGE(vi.prefix)
		}
	} else {
		if len(vi.prefix) == 0 {
			vi.iter.Last()
		} else {
			if !vi.iter.SeekLT(prefixEnd(vi.prefix)) {
				vi.iter.Last()
			}
		}
	}
	// go to the next 'user key'
	vi.advanceToNextKey()
}

// advanceToNextKey() advances to the next unique 'user key'
func (vi *VersionedIterator) advanceToNextKey() {
	vi.isValid, vi.key, vi.value = false, nil, nil
	// while the iterator is valid - step to next key
	for ; vi.iter.Valid(); vi.step() {
		userKey, version, err := vi.store.parseVersionedKey(vi.iter.Key())
		if err != nil || (len(vi.prefix) > 0 && !bytes.HasPrefix(userKey, vi.prefix)) {
			continue
		}
		// skip over the 'previous userKey' to go to the next 'userKey'
		if version > vi.store.version || (vi.lastUserKey != nil && bytes.Equal(userKey, vi.lastUserKey)) {
			continue
		}
		// set as 'previous userKey'
		vi.lastUserKey = bytes.Clone(userKey)
		// Now the iterator's current value is the newest visible version for userKey.
		tomb, val := vi.store.parseValueWithTombstone(vi.iter.Value())
		// skip dead user-keys
		if tomb == DeadTombstone {
			continue
		}
		// set variables
		vi.key, vi.value, vi.isValid = bytes.Clone(userKey), val, true
		// exit
		return
	}
}

// step() increments the iterator to the logical 'next'
func (vi *VersionedIterator) step() {
	if vi.reverse {
		vi.iter.Prev()
	} else {
		vi.iter.Next()
	}
}

// makeVersionedKey() creates a versioned key with inverted version encoding
// k = [UserKey][InvertedVersion]
func (vs *VersionedStore) makeVersionedKey(userKey []byte, version uint64) []byte {
	keyLength := len(userKey) + VersionSize
	vs.keyBuffer = ensureCapacity(vs.keyBuffer, keyLength)
	// copy user key into buffer
	offset := copy(vs.keyBuffer, userKey)
	// use the inverted version (^version) so newer versions sort first
	binary.BigEndian.PutUint64(vs.keyBuffer[offset:], ^version)
	// return a copy to prevent buffer reuse issues
	result := make([]byte, keyLength)
	copy(result, vs.keyBuffer)
	// exit
	return result
}

// parseVersionedKey() extracts components and converts back from inverted version
// k = [UserKey][InvertedVersion]
func (vs *VersionedStore) parseVersionedKey(versionedKey []byte) (userKey []byte, version uint64, err lib.ErrorI) {
	// extract user key (everything between history prefix and suffix)
	userKeyEnd := len(versionedKey) - VersionSize
	// extract the userKey and the version
	userKey, version = versionedKey[:userKeyEnd], binary.BigEndian.Uint64(versionedKey[userKeyEnd:])
	// extract inverted version and convert back to real version
	version = ^version
	// exit
	return
}

// valueWithTombstone() creates a value with tombstone prefix
// v = [1-byte Tombstone][ActualValue]
func (vs *VersionedStore) valueWithTombstone(tombstone byte, value []byte) (v []byte) {
	v = make([]byte, 1+len(value))
	// first byte is tombstone indicator
	v[0] = tombstone
	// the rest is the value
	if len(value) > 0 {
		copy(v[1:], value)
	}
	// exit
	return
}

// parseValueWithTombstone() extracts tombstone and actual value
// v = [1-byte Tombstone][ActualValue]
func (vs *VersionedStore) parseValueWithTombstone(v []byte) (tombstone byte, value []byte) {
	if len(v) == 0 {
		return DeadTombstone, nil
	}
	// extract the value
	if len(v) > 1 {
		value = v[1:]
	}
	// first byte is tombstone indicator
	return v[0], bytes.Clone(value)
}

// ensureCapacity() ensures the buffer has sufficient capacity for the key size (n)
func ensureCapacity(buf []byte, n int) []byte {
	if cap(buf) < n {
		return make([]byte, n, n*2)
	}
	return buf[:n]
}
