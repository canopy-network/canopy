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

// Key Layout: per-segment length prefix + version
// User key (path-like) is encoded as a sequence of length-prefixed segments where
// every non-final path component retains its trailing '/' and the final component
// is stored without a trailing '/'.

// Example path: "s/a/account"
// Segments (with trailing slash for non-final components):
//   "s/"  "a/"  "account"
// Encoded (via lib.JoinLenPrefix):
//   [2]"s/" [2]"a/" [7]"account"
// Composite on-disk key:
//   [2]"s/" [2]"a/" [7]"account" [^version]

// Version suffix:
//   8-byte big-endian bitwise-inverted version (^version) so that newer versions sort first.

// Prefix iteration:
//   For prefix "s/" -> segments "s/" -> encoded: [2]"s/"
//     LowerBound = encodedPrefix
//     UpperBound = prefixEnd(encodedPrefix)
//   For prefix "s/a/" -> segments "s/" "a/" -> encoded: [2]"s/" [2]"a/"
//     Same LB/UB pattern.
// All descendants (regardless of length of remaining segments) lie within that single contiguous range.

// Accepted input for Set/Get/Delete/Iterator:
//   - Fully encoded length-prefixed stream (e.g. "\x02s/\x02a/\x07account")
//   - Keys MUST be fully encoded length-prefixed streams. Otherwise a panic will occur.
//   - Maximum key length is 255 bytes.
//   - Use lib.JoinLenPrefix to encode keys.

// Final layout: [length-prefixed-key][^version]

// Performance properties from this layout:
//   - Single-range prefix scans at any depth.
//   - Fast skipping of all versions of a user key (SeekGE with 0xFF fill or SeekLT).
//   - Block property filtering on version ranges.
//   - Inverted version ordering for latest-first retrieval per key.
const (
	VersionSize    = 8
	DeadTombstone  = byte(1)
	AliveTombstone = byte(0)
	maxVersion     = math.MaxUint64
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
	// create (and validate) key to seek: [length-prefixed-key][^version]
	seekKey := vs.makeVersionedKey(userKey, vs.version)
	// iterate only over the key's boundary
	it, err := vs.newVersionedIterator(userKey, false, false)
	if err != nil {
		return nil, 0, err
	}
	defer it.Close()
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
	return bytes.Clone(value), tombstone, nil
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
	return vs.newVersionedIterator(prefix, reverse, allVersions)
}

// Iterator returns an iterator for all keys with the given prefix
func (vs *VersionedStore) Iterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(prefix, false, false)
}

// RevIterator returns a reverse iterator for all keys with the given prefix
func (vs *VersionedStore) RevIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(prefix, true, false)
}

// ArchiveIterator returns an iterator for all keys with the given prefix
// TODO: Currently not working, VersionedIterator must be modified to support archive iteration
func (vs *VersionedStore) ArchiveIterator(prefix []byte) (lib.IteratorI, lib.ErrorI) {
	return vs.newVersionedIterator(prefix, false, true)
}

// newVersionedIterator creates a new  versioned iterator
func (vs *VersionedStore) newVersionedIterator(prefix []byte, reverse bool, allVersions bool) (*VersionedIterator, lib.ErrorI) {
	// validate prefix
	_ = lib.DecodeLengthPrefixed(prefix)
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
		// [EXPERIMENTAL] test of whether linear prev/next is faster than seeking
		// on a small number of keys
		seek: vs.version != maxVersion,
	}, nil
}

// VersionedIterator implements  iteration with single-pass key deduplication
type VersionedIterator struct {
	iter        *pebble.Iterator
	store       *VersionedStore
	prefix      []byte
	key         []byte
	value       []byte
	reverse     bool
	isValid     bool
	initialized bool
	allVersions bool
	lastUserKey []byte
	valueBuff   []byte
	seek        bool
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
		// possibly new key found, perform full parsing
		userKey, _, err := parseVersionedKey(rawKey, false)
		if err != nil {
			continue
		}
		// validate userKey and avoid duplicates
		if userKey == nil || (vi.lastUserKey != nil &&
			bytes.Equal(userKey, vi.lastUserKey)) {
			continue
		}
		// copy userKey as is currently a reference
		userKey = bytes.Clone(userKey)
		// reuse buffer
		vi.lastUserKey = ensureCapacity(vi.lastUserKey, len(userKey))
		copy(vi.lastUserKey, userKey)
		vi.lastUserKey = vi.lastUserKey[:len(userKey)]
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
			// [length-prefixed-key][^version]
			seekKey := vi.store.makeVersionedKey(userKey, vi.store.version)
			// this lands at the first entry with ^ver >= ^target within this user key,
			// i.e. the greatest original version <= target.
			if vi.iter.SeekGE(seekKey) {
				// ensure the iterator is still on the same key (it should be, since at least
				// one version <= target for this user key was already observed).
				k, _, _ := parseVersionedKey(vi.iter.Key(), false)
				if bytes.Equal(k, vi.lastUserKey) {
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
	// attempt to skip versions if seeking is enabled
	if vi.seek && vi.iter.Valid() && vi.lastUserKey != nil {
		currentKey, _, err := parseVersionedKey(vi.iter.Key(), false)
		if err != nil {
			vi.isValid = false
			return
		}
		// only seek if iterator is still on the same encoded key
		if bytes.Equal(currentKey, vi.lastUserKey) {
			if vi.reverse {
				// seek backwards to skip all versions of current key
				vi.iter.SeekLT(vi.lastUserKey)
			} else {
				// seek over the prefix end to skip all versions of current key
				vi.iter.SeekGE(prefixEnd(vi.lastUserKey))
			}
			return
		}
	}
	// normal iteration
	if vi.reverse {
		vi.iter.Prev()
	} else {
		vi.iter.Next()
	}
}

// makeVersionedKey() creates a versioned key with inverted version encoding
// k = [length-prefixed-key][InvertedVersion]
func (vs *VersionedStore) makeVersionedKey(userKey []byte, version uint64) []byte {
	// validate key is length-prefixed
	_ = lib.DecodeLengthPrefixed(userKey)
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
// k = [length-prefixed-key][InvertedVersion]
// The caller should not modify the returned userKey slice as it points to the original buffer,
// instead it should make a copy if needed.
func parseVersionedKey(versionedKey []byte, getVersion bool) (userKey []byte,
	version uint64, err lib.ErrorI) {
	// extract user key (everything between history prefix and suffix)
	userKeyEnd := len(versionedKey) - VersionSize
	// extract the userKey and the version
	userKey = versionedKey[:userKeyEnd]
	if !getVersion {
		return
	}
	version = binary.BigEndian.Uint64(versionedKey[userKeyEnd:])
	// extract inverted version and convert back to real version
	version = ^version
	// exit
	return
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
// The caller should not modify the returned userKey slice as it points to the original buffer,
// instead it should make a copy if needed.
func parseValueWithTombstone(v []byte) (tombstone byte, value []byte) {
	if len(v) == 0 {
		return DeadTombstone, nil
	}
	// extract the value
	if len(v) > 1 {
		value = v[1:]
	}
	// first byte is tombstone indicator
	return v[0], value
}

// ensureCapacity() ensures the buffer has sufficient capacity for the key size (n)
func ensureCapacity(buf []byte, n int) []byte {
	if cap(buf) < n {
		return make([]byte, n, n*2)
	}
	return buf[:n]
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
