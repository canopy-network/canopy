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

// Get() retrieves the latest version of a key at or before vs.version
func (vs *VersionedStore) Get(key []byte) ([]byte, lib.ErrorI) {
	val, _, err := vs.get(key)
	return val, err
}

// get() performs SeekGE to ^version and return the first entry.
func (vs *VersionedStore) get(userKey []byte) (value []byte, tombstone byte, err lib.ErrorI) {
	encKey := encodeUserKey(userKey)
	// iterate only over the key's boundary
	it, err := vs.newVersionedIterator(encKey, false, false)
	if err != nil {
		return nil, 0, err
	}
	defer it.Close()
	// build seek key: [Enc(UserKey)][0x00,0x00][^version]
	seekKey := vs.makeVersionedKey(userKey, vs.version)
	iter := it.iter
	if !iter.SeekGE(seekKey) {
		return nil, 0, nil
	}
	// iterator is at [encKey, prefixEnd(encKey)), so no need to re-check encoded key.
	// verify version
	v := parseVersion(iter.Key())
	if v > vs.version {
		return nil, 0, nil
	}
	raw, vErr := iter.ValueAndErr()
	if vErr != nil {
		return nil, 0, ErrStoreGet(vErr)
	}
	// verify tombstone
	tombstone, value = parseValueWithTombstone(raw)
	if tombstone == DeadTombstone {
		return nil, 0, nil
	}
	return value, tombstone, nil
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
		seek:        true,
	}, nil
}

// VersionedIterator implements  iteration with single-pass key deduplication
type VersionedIterator struct {
	iter           *pebble.Iterator
	store          *VersionedStore
	prefix         []byte
	key            []byte
	value          []byte
	reverse        bool
	isValid        bool
	initialized    bool
	allVersions    bool
	lastEncodedKey []byte
	valueBuff      []byte
	seek           bool
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
	// seek is used to take advantage of block property filters to skip
	// sst blocks with versions outside the store version range
	if vi.reverse {
		// position at the last key strictly below UpperBound
		ub := prefixEnd(vi.prefix)
		if !vi.iter.SeekLT(ub) {
			// nothing below ub, iterator invalid
			vi.isValid = false
			return
		}
	} else {
		// position at first key >= LowerBound (prefix)
		if !vi.iter.SeekGE(vi.prefix) {
			vi.isValid = false
			return
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
		// validate just the version
		rawKey := vi.iter.Key()
		version := parseVersion(rawKey)
		if version > vi.store.version {
			// skip over the 'previous userKey' to go to the next 'userKey'
			continue
		}
		// validate encoded key and avoid duplicates
		encodedKey := extractEncodedKey(rawKey)
		if encodedKey == nil || (vi.lastEncodedKey != nil &&
			bytes.Equal(encodedKey, vi.lastEncodedKey)) {
			continue
		}
		// new key found, perform full parsing
		userKey, _, err := parseVersionedKey(rawKey, false)
		if err != nil {
			continue
		}
		// reuse buffer
		vi.lastEncodedKey = ensureCapacity(vi.lastEncodedKey, len(encodedKey))
		copy(vi.lastEncodedKey, encodedKey)
		vi.lastEncodedKey = vi.lastEncodedKey[:len(encodedKey)]
		// extract value
		rawValue, valErr := vi.iter.ValueAndErr()
		if valErr != nil {
			continue
		}
		// reuse buffer
		vi.valueBuff = ensureCapacity(vi.valueBuff, len(rawValue))
		copy(vi.valueBuff, rawValue)
		vi.valueBuff = vi.valueBuff[:len(rawValue)]
		// in reverse mode, when a new key is found, seek to its highest version
		if vi.reverse {
			// build the synthetic seek key for this userKey at the target store.version:
			// [Enc(userKey)][0x00,0x00][^version]
			seekKey := vi.store.makeVersionedKey(userKey, vi.store.version)
			// this lands at the first entry with ^ver >= ^target within this user key,
			// i.e. the greatest original version <= target.
			if vi.iter.SeekGE(seekKey) {
				// Ensure we’re still on the same encoded key (we should be, since we already
				// observed at least one version <= target for this user key).
				if bytes.Equal(extractEncodedKey(vi.iter.Key()), vi.lastEncodedKey) {
					if val, valErr := vi.iter.ValueAndErr(); valErr == nil {
						vi.valueBuff = ensureCapacity(vi.valueBuff, len(val))
						copy(vi.valueBuff, val)
						vi.valueBuff = vi.valueBuff[:len(val)]
					}
				}
			}
		}
		// now the iterator's current value is the newest visible version for userKey.
		tomb, val := parseValueWithTombstone(vi.valueBuff)
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
		// check if is possible to skip versions in reverse
		if vi.seek && vi.iter.Valid() && vi.lastEncodedKey != nil {
			currentEncodedKey := extractEncodedKey(vi.iter.Key())
			// only seek if iterator is still on the same encoded key
			if bytes.Equal(currentEncodedKey, vi.lastEncodedKey) {
				// in reverse, seek backwards to skip all versions of current key
				// remove the separator [0x00 0x00] from [EncodedKey][0x00 0x00] to get [EncodedKey]
				// then SeekLT([EncodedKey]) finds the largest key less than [EncodedKey]
				// since current key versions are [EncodedKey][0x00 0x00][version] > [EncodedKey]
				// this lands on [PreviousKey][0x00 0x00][version], the previous key's highest version
				trimLen := len(vi.lastEncodedKey) - len(separator)
				vi.iter.SeekLT(vi.lastEncodedKey[:trimLen])
				return
			}
		}
		vi.iter.Prev()
	} else {
		// check if is possible to skip versions
		if vi.seek && vi.iter.Valid() && vi.lastEncodedKey != nil {
			currentEncodedKey := extractEncodedKey(vi.iter.Key())
			// only seek if iterator is still on the same encoded key
			if bytes.Equal(currentEncodedKey, vi.lastEncodedKey) {
				// to skip all versions of current key, temporarily
				// increment the lastByte byte of the separator
				// from [EncodedKey][0x00 0x00] to [EncodedKey][0x00 0x01]
				// this skips past all [EncodedKey][0x00 0x00][version] entries
				lastByte := len(vi.lastEncodedKey) - 1
				orig := vi.lastEncodedKey[lastByte]
				vi.lastEncodedKey[lastByte] = 0x01
				vi.iter.SeekGE(vi.lastEncodedKey)
				vi.lastEncodedKey[lastByte] = orig // restore
				return
			}
		}
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
func parseVersionedKey(versionedKey []byte, versionParse bool) (userKey []byte, version uint64,
	err lib.ErrorI) {
	// validate key length
	min := len(separator) + VersionSize
	if len(versionedKey) < min {
		return nil, 0, ErrInvalidKey()
	}
	// validate separator + version size
	encEnd := len(versionedKey) - (len(separator) + VersionSize)
	if encEnd < 0 {
		return nil, 0, ErrInvalidKey()
	}
	// validate separator
	if !bytes.Equal(versionedKey[encEnd:encEnd+len(separator)], separator) {
		return nil, 0, ErrInvalidKey()
	}
	// decode user key
	decoded, derr := decodeUserKey(versionedKey[:encEnd])
	if derr != nil {
		return nil, 0, ErrInvalidKey()
	}
	if versionParse {
		// get the version from the last 8 bytes
		version = parseVersion(versionedKey)
	}
	return decoded, version, nil
}

// parseVersion extracts version directly from the last 8 bytes without full parsing
func parseVersion(versionedKey []byte) uint64 {
	if len(versionedKey) < VersionSize {
		return 0
	}
	// extract inverted version from last 8 bytes
	offset := len(versionedKey) - VersionSize
	return ^binary.BigEndian.Uint64(versionedKey[offset:])
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

// extractEncodedKey returns the key portion without version
// Returns: [Enc(UserKey)][0x00,0x00] (includes separator, excludes version)
// Note: returns a reference, clone if needed
func extractEncodedKey(versionedKey []byte) []byte {
	minLen := len(separator) + VersionSize
	if len(versionedKey) < minLen {
		return nil
	}
	// return everything except the last 8 bytes (version)
	return versionedKey[:len(versionedKey)-VersionSize]
}

// ensureCapacity() ensures the buffer has sufficient capacity for the key size (n)
func ensureCapacity(buf []byte, n int) []byte {
	if cap(buf) < n {
		return make([]byte, n, n*2)
	}
	return buf[:n]
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
	// decode version directly
	version := parseVersion(userKey)
	// ignore invalid keys, math.MaxUint64 is not supported as an upper bound range for the interval
	// collector as is a half range of type [min, max)
	if version == 0 || version == maxVersion {
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
