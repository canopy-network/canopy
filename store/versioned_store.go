package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/canopy-network/canopy/lib"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/sstable"
)

/* versioned_store.go implements a multi-version store in pebble db*/

// - Encode userKey by escaping 0x00 as 0x00 0xFF.
// - Append a separator 0x00 0x00 after the encoded userKey.
// - Then append 8-byte big-endian inverted version (^version).
// Final layout: [Enc(userKey)][0x00,0x00][^version]
// This guarantees per-userKey contiguity and correct lexicographic separation from longer keys.
const (
	VersionSize    = 8
	DeadTombstone  = byte(1)
	AliveTombstone = byte(0)
	maxVersion     = math.MaxUint64
)

var (
	separator = []byte{0x00, 0x00} // separator between EncUserKey and version
	escByte   = byte(0x00)         // escape byte inside userKey
	escEsc    = byte(0xFF)         // escape continuation
)

// VersionedStore uses inverted version encoding and reverse seeks for maximum performance
type VersionedStore struct {
	db        pebble.Reader
	batch     *pebble.Batch
	closed    bool
	version   uint64
	keyBuffer []byte
}

// NewVersionedStore creates a new  versioned store
func NewVersionedStore(db pebble.Reader, batch *pebble.Batch, version uint64) (*VersionedStore, lib.ErrorI) {
	return &VersionedStore{db: db, batch: batch, version: version, keyBuffer: make([]byte, 0, 256)}, nil
}

// Set() stores a key-value pair at the current version
func (vs *VersionedStore) Set(key, value []byte) (err lib.ErrorI) {
	return vs.SetAt(key, value, vs.version)
}

// SetAt() stores a key-value pair at the given version
func (vs *VersionedStore) SetAt(key, value []byte, version uint64) (err lib.ErrorI) {
	k := vs.makeVersionedKey(key, version)
	v := vs.valueWithTombstone(AliveTombstone, value)
	if e := vs.batch.Set(k, v, nil); e != nil {
		return ErrStoreSet(e)
	}
	return
}

// Delete() marks a key as deleted at the current version
func (vs *VersionedStore) Delete(key []byte) (err lib.ErrorI) {
	return vs.DeleteAt(key, vs.version)
}

// DeleteAt() marks a key as deleted at the given version
func (vs *VersionedStore) DeleteAt(key []byte, version uint64) (err lib.ErrorI) {
	k := vs.makeVersionedKey(key, version)
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

// get() retrieves the latest version of a key using reverse seek
func (vs *VersionedStore) get(key []byte) (value []byte, tombstone byte, err lib.ErrorI) {
	// strictly iterate over this userKey's versions
	lb := encodeUserKey(key)
	var seekKey []byte
	if vs.version != maxVersion {
		// seek to boundary just above snapshot to land on newest visible
		seekKey = vs.makeVersionedKey(key, vs.version+1)
	}
	// create a new iterator
	i, err := vs.newVersionedIterator(lb, true, false)
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
		userKey, ver, e := parseVersionedKey(iter.Key())
		if e != nil || !bytes.Equal(userKey, key) || ver > vs.version {
			continue
		}
		// parse the value to extract tombstone and actual value
		rawValue, valErr := iter.ValueAndErr()
		if valErr != nil {
			return nil, 0, ErrStoreGet(valErr)
		}
		tombstone, value = parseValueWithTombstone(rawValue)
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
func (vs *VersionedStore) Close() lib.ErrorI {
	// prevent panic due to double close
	if vs.closed {
		return nil
	}
	// for write-only versioned store, db may be nil
	if vs.db != nil {
		if err := vs.db.Close(); err != nil {
			return ErrCloseDB(err)
		}
	}
	// for read-only versioned store, batch may be nil
	if vs.batch != nil {
		if err := vs.batch.Close(); err != nil {
			return ErrCloseDB(err)
		}
	}
	vs.closed = true
	return nil
}

// NewIterator is a wrapper around the underlying iterators to conform to the TxnReaderI interface
func (vs *VersionedStore) NewIterator(prefix []byte, reverse bool, allVersions bool) (lib.IteratorI, lib.ErrorI) {
	// Encode raw user-key prefix for correct bounds (no terminator)
	return vs.newVersionedIterator(encodeUserPrefix(prefix), reverse, allVersions)
}

// Iterator returns an iterator for all keys with the given prefix
func (vs *VersionedStore) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(encodeUserPrefix(prefix), false, false)
}

// RevIterator returns a reverse iterator for all keys with the given prefix
func (vs *VersionedStore) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(encodeUserPrefix(prefix), true, false)
}

// ArchiveIterator returns an iterator for all keys with the given prefix
// TODO: Currently not working, VersionedIterator must be modified to support archive iteration
func (vs *VersionedStore) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(encodeUserPrefix(prefix), false, true)
}

// newVersionedIterator creates a new  versioned iterator
func (vs *VersionedStore) newVersionedIterator(prefix []byte, reverse bool, allVersions bool) (*VersionedIterator, lib.ErrorI) {
	// use property filter if possible
	var filters []pebble.BlockPropertyFilter
	if vs.version != maxVersion {
		filters = []pebble.BlockPropertyFilter{
			newTargetWindowFilter(0, vs.version),
		}
	}
	var (
		err  error
		iter *pebble.Iterator
		opts = &pebble.IterOptions{
			LowerBound:      prefix,
			UpperBound:      prefixEnd(prefix),
			KeyTypes:        pebble.IterKeyTypePointsOnly,
			PointKeyFilters: filters,
			UseL6Filters:    false,
		}
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
		iter:        iter,
		store:       vs,
		prefix:      prefix,
		reverse:     reverse,
		allVersions: allVersions,
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
	allVersions bool
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
	if vi.reverse {
		vi.iter.Last()
	} else {
		vi.iter.First()
	}
	// go to the next 'user key'
	vi.advanceToNextKey()
}

// advanceToNextKey() advances to the next unique 'user key'
func (vi *VersionedIterator) advanceToNextKey() {
	vi.isValid, vi.key, vi.value = false, nil, nil
	// while the iterator is valid - step to next key
	for ; vi.iter.Valid(); vi.step() {
		userKey, version, err := parseVersionedKey(vi.iter.Key())
		// skip over the 'previous userKey' to go to the next 'userKey'
		if err != nil || version > vi.store.version || (vi.lastUserKey != nil &&
			bytes.Equal(userKey, vi.lastUserKey)) {
			continue
		}
		// in reverse mode, when a new key is found, seek to its highest version
		if vi.reverse {
			for vi.iter.Prev() {
				prevUserKey, prevVersion, err := parseVersionedKey(vi.iter.Key())
				if err != nil || !bytes.Equal(userKey, prevUserKey) || prevVersion > vi.store.version {
					break
				}
			}
			vi.iter.Next()
			if version > vi.store.version {
				continue
			}
		}
		// reuse buffer if capacity is sufficient
		vi.lastUserKey = ensureCapacity(vi.lastUserKey, len(userKey))
		copy(vi.lastUserKey, userKey)
		vi.lastUserKey = vi.lastUserKey[:len(userKey)]
		// extract value
		rawValue, valErr := vi.iter.ValueAndErr()
		if valErr != nil {
			continue
		}
		// now the iterator's current value is the newest visible version for userKey.
		tomb, val := parseValueWithTombstone(rawValue)
		// skip dead user-keys
		if tomb == DeadTombstone {
			continue
		}
		// reuse buffer if capacity is sufficient
		vi.key = ensureCapacity(vi.key, len(userKey))
		copy(vi.key, userKey)
		vi.key, vi.value, vi.isValid = vi.key[:len(userKey)], val, true
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
// k = [Enc(UserKey)][0x00,0x00][InvertedVersion]
func (vs *VersionedStore) makeVersionedKey(userKey []byte, version uint64) []byte {
	encodedKey := encodeUserKey(userKey)
	keyLength := len(encodedKey) + VersionSize
	vs.keyBuffer = ensureCapacity(vs.keyBuffer, keyLength)
	// copy user key into buffer
	offset := copy(vs.keyBuffer, encodedKey)
	// use the inverted version (^version) so newer versions sort first
	binary.BigEndian.PutUint64(vs.keyBuffer[offset:], ^version)
	// return a copy to prevent buffer reuse issues
	result := make([]byte, keyLength)
	copy(result, vs.keyBuffer)
	// exit
	return result
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

// parseVersionedKey() extracts components and converts back from inverted version
// k = [Enc(UserKey)][0x00,0x00][InvertedVersion]
func parseVersionedKey(versionedKey []byte) (userKey []byte, version uint64, err lib.ErrorI) {
	min := len(separator) + VersionSize
	if len(versionedKey) < min {
		return nil, 0, ErrInvalidKey()
	}
	encEnd := len(versionedKey) - (len(separator) + VersionSize)
	if encEnd < 0 {
		return nil, 0, ErrInvalidKey()
	}
	// verify terminator
	if !bytes.Equal(versionedKey[encEnd:encEnd+len(separator)], separator) {
		return nil, 0, ErrInvalidKey()
	}
	// decode user key
	decoded, derr := decodeUserKey(versionedKey[:encEnd])
	if derr != nil {
		return nil, 0, ErrInvalidKey()
	}
	// invert version
	version = ^binary.BigEndian.Uint64(versionedKey[encEnd+len(separator):])
	return decoded, version, nil
}

// parseValueWithTombstone() extracts tombstone and actual value
// v = [1-byte Tombstone][ActualValue]
func parseValueWithTombstone(v []byte) (tombstone byte, value []byte) {
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

// encodeUserKey escapes 0x00 as 0x00 0xFF and appends 0x00 0x00 terminator.
func encodeUserKey(u []byte) []byte {
	// worst-case growth: every byte is 0x00 => 2x, plus 2 bytes terminator.
	out := make([]byte, 0, len(u)*2+2)
	for _, b := range u {
		if b == escByte {
			out = append(out, escByte, escEsc)
		} else {
			out = append(out, b)
		}
	}
	return append(out, separator...)
}

// decodeUserKey reverses encodeUserKey.
// 'enc' must NOT include the 0x00 0x00 terminator.
func decodeUserKey(enc []byte) ([]byte, error) {
	out := make([]byte, 0, len(enc))
	for i := 0; i < len(enc); i++ {
		b := enc[i]
		if b != escByte {
			out = append(out, b)
			continue
		}
		// b == 0x00: must be an escaped 0x00 => next byte must be 0xFF
		i++
		if i >= len(enc) {
			return nil, fmt.Errorf("unterminated escape")
		}
		if enc[i] != escEsc {
			return nil, fmt.Errorf("invalid escape sequence 0x00 0x%02x", enc[i])
		}
		out = append(out, escByte)
	}
	return out, nil
}

// encodeUserPrefix escapes 0x00 as 0x00 0xFF without appending the 0x00 0x00 terminator.
// Use this to build iterator bounds for user-key prefix scans.
func encodeUserPrefix(p []byte) []byte {
	out := make([]byte, 0, len(p)*2)
	for _, b := range p {
		if b == escByte {
			out = append(out, escByte, escEsc)
		} else {
			out = append(out, b)
		}
	}
	return out
}

// BlockPropertyCollector / BlockPropertyFilter code below

const blockPropertyName = "canopy.mvcc.version.range"

// versionedCollector implements the IntervalMapper interface through which an user can
// define the mapping between keys and intervals by mapping keys to [version, version+1) using the
// version bytes. This helps iteration as it allows for efficient range queries on versioned data by
// only checking the SST tables and blocks that may contain the required versioned data.
type versionedCollector struct{}

// enforce interface implementation
var _ sstable.IntervalMapper = versionedCollector{}

// MapPointKey adds a versioned key to the interval collector.
func (versionedCollector) MapPointKey(key pebble.InternalKey, _ []byte) (sstable.BlockInterval, error) {
	userKey := key.UserKey
	if len(userKey) < VersionSize {
		// ignore malformed keys
		return sstable.BlockInterval{}, nil
	}
	// Decode inverted suffix directly. Avoid any higher-level parser here.
	_, version, err := parseVersionedKey(userKey)
	// ignore invalid keys, math.MaxUint64 is not supported as an upper bound range for the interval
	// collector as is a half range of type [min, max)
	if err != nil || version == maxVersion {
		return sstable.BlockInterval{}, nil
	}
	// set the interval for the key
	return sstable.BlockInterval{Lower: version, Upper: version + 1}, nil
}

// MapRangeKeys implements sstable.IntervalMapper for range keys.
// Not implemented as the versioned store does not support range keys.
func (versionedCollector) MapRangeKeys(span sstable.Span) (sstable.BlockInterval, error) {
	return sstable.BlockInterval{}, nil
}

// newVersionedPropertyCollector returns a BlockPropertyCollector that records per-block
// [minVersion, maxVersionExclusive) using the interval mapper.
func newVersionedPropertyCollector() pebble.BlockPropertyCollector {
	return sstable.NewBlockIntervalCollector(
		blockPropertyName,
		versionedCollector{},
		nil,
	)
}

// newTargetWindowFilter builds a filter to admit blocks/tables that may contain
// any low <= version <= high. It uses the interval [low, high+1).
func newTargetWindowFilter(low, high uint64) sstable.BlockPropertyFilter {
	return sstable.NewBlockIntervalFilter(
		blockPropertyName,
		low,
		high+1,
		nil,
	)
}
