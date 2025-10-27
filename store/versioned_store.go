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

// get()  retrieves the latest version of a key using reverse seek
func (vs *VersionedStore) get(key []byte) (value []byte, tombstone byte, err lib.ErrorI) {
	var seekKey = key
	if vs.version != math.MaxUint64 {
		seekKey = vs.makeVersionedKey(key, vs.version+1)
	}
	// create a new iterator
	i, err := vs.newVersionedIterator(key, true, false)
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
		tombstone, value = parseValueWithTombstone(iter.Value())
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
	var (
		err  error
		iter *pebble.Iterator
		opts = &pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: prefixEnd(prefix),
			KeyTypes:   pebble.IterKeyTypePointsOnly,
			PointKeyFilters: []pebble.BlockPropertyFilter{
				newVersionedFilter(vs.version),
			},
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
		if err != nil || (len(vi.prefix) > 0 && !bytes.HasPrefix(userKey, vi.prefix)) {
			continue
		}
		// skip over the 'previous userKey' to go to the next 'userKey'
		if version > vi.store.version || (vi.lastUserKey != nil && bytes.Equal(userKey, vi.lastUserKey)) {
			continue
		}
		// in reverse mode, when a new key is found, seek to its highest version
		if vi.reverse {
			// clone user key as is currently a reference to the key in the iterator
			userKey = bytes.Clone(userKey)
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
		// set as 'previous userKey'
		vi.lastUserKey = bytes.Clone(userKey)
		// Now the iterator's current value is the newest visible version for userKey.
		tomb, val := parseValueWithTombstone(vi.iter.Value())
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
// The caller should not modify the returned userKey slice as it points to the original buffer,
// instead it should make a copy if needed.
func parseVersionedKey(versionedKey []byte) (userKey []byte, version uint64, err lib.ErrorI) {
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

// encodeMinMax serializes [min, maxExclusive) as 16 bytes big-endian.
func encodeMinMax(min, max uint64) []byte {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], min)
	binary.BigEndian.PutUint64(buf[8:16], max)
	return buf[:]
}

// decodeMinMax parses a 16-byte [min, maxExclusive) big-endian property blob.
func decodeMinMax(prop []byte) (min, max uint64, ok bool) {
	if len(prop) != 16 {
		return 0, 0, false
	}
	return binary.BigEndian.Uint64(prop[0:8]), binary.BigEndian.Uint64(prop[8:16]), true
}

const blockPropertyName = "canopy.mvcc.version.range"

var _ pebble.BlockPropertyCollector = &versionedCollector{} // force BlockPropertyCollector interface

// versionedCollector implements pebble.BlockPropertyCollector.
//
// It observes each point key added to a data block while building an sstable,
// extracts the REAL (uninverted) MVCC version from the trailing 8 bytes,
// updates per-data-block min/max, and then emits a property when the data block
// is finished. It also rolls up those properties for index/table scopes.
//
// Note: range keys are explictly ignored; they are handled by separate mechanisms
// and do not contribute to point-key data block properties.
type versionedCollector struct {
	// per-data-block state (for the currently building block).
	blockMin, blockMax uint64 // [bMin, bMax) over REAL versions
	blockHas           bool   // true if we’ve seen at least one point key in the current block
	// per-index-block roll-up. These aggregate properties over finished data blocks
	// that belong to the current index block. Emitted at FinishIndexBlock.
	indexMin, indexMax uint64
	indexHas           bool
	// per-table roll-up. Aggregates over finished index blocks. Emitted at FinishTable.
	tableMin, tableMax uint64
	tableHas           bool
}

// NewVersionedCollector constructs a new collector instance.
// Pebble will call this per sstable writer (i.e., per flush/compaction output).
func NewVersionedCollector() *versionedCollector { return &versionedCollector{} }

func (vc *versionedCollector) Name() string {
	return blockPropertyName
}

// AddPointKey is called for each point key appended to the current data block.
// We:
// - Extract the suffix (last 8 bytes) as an inverted version.
// - Invert it back (~) to get the REAL version.
// - Update the current data block’s [min,max) interval.
func (vc *versionedCollector) AddPointKey(k sstable.InternalKey, _ []byte) error {
	userKey := k.UserKey
	// ignore malformed user keys to not fail the sstable build
	if len(userKey) < VersionSize {
		return nil
	}
	// extract the version from the key
	_, version, err := parseVersionedKey(userKey)
	// ignore malformed user keys to not fail the sstable build
	if err != nil {
		return nil
	}
	// prevent max version from exceeding the maximum possible version
	nextVersion := version
	if version != math.MaxUint64 {
		nextVersion++
	}
	// set data block interval for a new block
	if !vc.blockHas {
		vc.blockMin, vc.blockMax, vc.blockHas = version, nextVersion, true
		return nil
	}
	// update an existing data block interval
	if version < vc.blockMin {
		vc.blockMin = version
	}
	if nextVersion > vc.blockMax {
		vc.blockMax = nextVersion
	}
	return nil
}

// AddRangeKeys is invoked for range keys added to the sstable.
// Currently not used on this store.
func (*versionedCollector) AddRangeKeys(_ sstable.Span) error { return nil }

// FinishDataBlock is called when the current data block completes.
// It stores the version interval metadata for the block.
func (c *versionedCollector) FinishDataBlock(buf []byte) ([]byte, error) {
	// Roll into table-level accumulation for FinishTable().
	if c.blockHas {
		// set table block interval for a new table
		if !c.tableHas {
			c.tableMin, c.tableMax, c.tableHas = c.blockMin, c.blockMax, true
		} else {
			// update an existing table block interval
			if c.blockMin < c.tableMin {
				c.tableMin = c.blockMin
			}
			if c.blockMax > c.tableMax {
				c.tableMax = c.blockMax
			}
		}
	}
	// emit block property. If the block had no point keys, encode a default
	// sentinel that the filter will interpret as “no intersection with any query.”
	prop := encodeMinMax(0, 0)
	if c.blockHas {
		prop = encodeMinMax(c.blockMin, c.blockMax)
	}
	return prop, nil
}

// AddPrevDataBlockToIndexBlock is called when the previously finished data block
// is logically added to the current index block.
//
// This is used hook to roll the finished block’s [min,max) into the index-level
// aggregation and then reset the per-data-block state for the next block.
func (c *versionedCollector) AddPrevDataBlockToIndexBlock() {
	if c.blockHas {
		if !c.indexHas {
			c.indexMin, c.indexMax, c.indexHas = c.blockMin, c.blockMax, true
		} else {
			if c.blockMin < c.indexMin {
				c.indexMin = c.blockMin
			}
			if c.blockMax > c.indexMax {
				c.indexMax = c.blockMax
			}
		}
	}
	// Reset per-block state for the next data block.
	c.blockMin, c.blockMax, c.blockHas = 0, 0, false
}

// FinishIndexBlock is called when the current index block completes.
func (c *versionedCollector) FinishIndexBlock(buf []byte) ([]byte, error) {
	prop := encodeMinMax(0, 0)
	if c.indexHas {
		prop = encodeMinMax(c.indexMin, c.indexMax)
	}
	// Reset index state for the next index block.
	c.indexMin, c.indexMax, c.indexHas = 0, 0, false
	return prop, nil
}

// FinishTable is called when the sstable completes.
func (c *versionedCollector) FinishTable(buf []byte) ([]byte, error) {
	prop := encodeMinMax(math.MaxUint64, 0)
	if c.tableHas {
		prop = encodeMinMax(c.tableMin, c.tableMax)
	}
	return prop, nil
}

// AddCollectedWithSuffixReplacement is not supported
func (vc *versionedCollector) AddCollectedWithSuffixReplacement(oldProp []byte, oldSuffix,
	newSuffix []byte) error {
	return nil
}

// SupportsSuffixReplacement sets the support for suffix replacement.
func (vc *versionedCollector) SupportsSuffixReplacement() bool {
	return false
}

var _ sstable.BlockPropertyFilter = (*versionedFilter)(nil)

type versionedFilter struct {
	limit uint64 // target version limit
}

func newVersionedFilter(limit uint64) *versionedFilter {
	return &versionedFilter{
		limit: limit,
	}
}

// SyntheticSuffixIntersects is not supported
func (vf *versionedFilter) SyntheticSuffixIntersects(prop []byte, suffix []byte) (bool, error) {
	return false, nil
}

// Name returns the sstable property name this filter applies to.
func (vf *versionedFilter) Name() string {
	return blockPropertyName
}

func (f *versionedFilter) Intersects(prop []byte) (bool, error) {
	min, max, ok := decodeMinMax(prop)
	if !ok {
		// be conservative: without a valid property, don’t prune.
		return true, nil
	}
	// recognize empty/sentinel ranges (as emitted by the collector for empty blocks).
	if min == 0 && max == 0 || min > max {
		return false, nil
	}
	// Admit if the block’s minimum version is <= limit.
	// rationale: If min > limit, then ALL versions in this block are > limit, so skip.
	return min <= f.limit, nil
}
