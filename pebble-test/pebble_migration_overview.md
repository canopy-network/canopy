# Canopy Pebble Migration Overview

## Executive Summary

**Recommendation**: Migrate Canopy from Badger to Pebble using the Option-1 LSS/HSS layout for improved performance and memory efficiency.

**Key Benefits**:
- Significant allocation efficiency improvements (2-4 allocs/op vs 1000+ allocs/op for large scans)
- Maintains all existing functionality including versioning, historical reads, and proofs
- Better compaction behavior and memory management at scale

**Migration Feasibility**: ✅ **Fully Viable**
- No functional gaps between Badger and Pebble for Canopy's use cases
- Primary blockers are API-level, not capability-level
- Clean migration path with minimal code changes required

**Main Technical Blockers**:
1. **Type Leakage**: `lib.StoreI.DB() *badger.DB` exposes Badger types directly
2. **API Coupling**: Badger-specific transaction APIs in `store/txn.go`
3. **Metrics Implementation**: Reflection-based access to Badger internals

**Solution**: Implement a thin engine abstraction layer that decouples Canopy from Badger-specific APIs while preserving all existing functionality.

---

## Current Badger Implementation Analysis

### Core Architecture

**Store Abstraction** (`store/store.go`):
- Opens Badger in managed mode with MVCC enabled
- Implements dual-write pattern: Latest State Store (LSS) and Historical State Store (HSS)
- Uses versioning: `lssVersion = math.MaxUint64` for latest, block height for historical
- Key prefixes: `s/` for latest state, `h/` for historical state

**Transaction Layer** (`store/txn.go`):
- Merged in-memory transaction with Badger as parent store
- Uses Badger-specific APIs: `NewTransactionAt()`, `NewWriteBatchAt()`
- Iterator support: forward, reverse, and archive (all-versions) iteration

### Badger-Specific Dependencies

**Type Leakage**:
- `lib.StoreI.DB() *badger.DB` exposes Badger types directly
- Used by `cmd/rpc/query.go` (lines 739-750) for store cloning

**API Coupling**:
- `TxnWriterI.SetEntryAt(entry *badger.Entry, version uint64)` requires Badger entries
- Meta bit usage: `badgerDeleteBit|badgerNoDiscardBit` for delete handling
- `valueOp.entry *badger.Entry` ties operations to Badger types

**Metrics Implementation**:
- Reflection-based access to Badger private fields
- Functions: `getSizeAndCountFromBatch()`, `getSizeAndCount()`

### Engine-Agnostic Components

**Indexer Layer** (`store/indexer.go`):
- Uses transaction-level `Iterator` and `RevIterator`
- Operates over length-prefixed key spaces
- Only requires forward/reverse prefix scan capability

**SMT Layer** (`store/smt.go`):
- Relies solely on `lib.RWStoreI` interface (Set/Get/Delete)
- No engine-specific dependencies

---

## Functional Requirements for Pebble Migration

### Core Capabilities Required

**Versioning and Historical Access**:
- Latest state reads via `s/` prefix
- Historical state reads via height-bounded range scans over `h/`
- Consistent snapshots at specific block heights

**Iteration Support**:
- Forward and reverse prefix iteration with lexicographical ordering
- Archive-style iteration for historical data analysis
- Stable iteration behavior across transaction boundaries

**Transaction Semantics**:
- Merged iterator behavior (in-memory + persistent state)
- Atomic commits across all store segments (state, indexer, commit)
- Batch operations for performance

**Delete and Garbage Collection**:
- Tombstone support for latest state deletions
- Historical delete markers at specific heights
- Efficient pruning of historical data by height ranges

**Proof Generation**:
- SMT operations via `RWStoreI` interface must remain unchanged
- No impact on cryptographic proof generation

**Operational Requirements**:
- Configurable durability (equivalent to Badger's `SyncWrites`)
- Efficient compaction and memory management
- Engine-neutral metrics collection

---

## Pebble Option-1 Implementation Strategy

### Data Layout Design

**Latest State Store (LSS)**: `s/<userKey>`
- Direct key-value mapping for current state
- Uses Pebble's native delete tombstones

**Historical State Store (HSS)**: `h/<heightBigEndian(8B)>/<userKey>`
- Height-encoded keys for historical data
- Enables efficient range queries by height

**Key Encoding Functions** (see `pebble-test/keys.go`):
```go
// Latest state key encoding
func EncodeLatestKey(userKey []byte) []byte

// Historical state key encoding  
func EncodeHistoricalKey(height uint64, userKey []byte) []byte

// Height-based range bounds
func boundsForHeight(height uint64) ([]byte, []byte)
func boundsForHeightRange(min, max uint64) ([]byte, []byte)
```

### Operation Mapping

| Operation | Badger (Current) | Pebble (Option-1) |
|-----------|------------------|-------------------|
| **Latest Read** | `NewTransactionAt(MaxUint64)` + `s/` prefix | Bounded iterator `["s/", "t/")` |
| **Historical Read** | `NewTransactionAt(height)` + `h/` prefix | Bounded iterator `boundsForHeight(height)` |
| **Latest Write** | `SetEntryAt(entry, MaxUint64)` to `s/` | `Set(s/<key>, value)` |
| **Historical Write** | `SetEntryAt(entry, height)` to `h/` | `Set(h/<height>/<key>, value)` |
| **Latest Delete** | Meta bits: `badgerDeleteBit\|badgerNoDiscardBit` | `Delete(s/<key>)` (native tombstone) |
| **Historical Delete** | Meta bits at height | `Set(h/<height>/<key>, tombstone_marker)` |
| **Reverse Iteration** | `IteratorOptions{Reverse: true}` | `NewIter()` + `Last()` + `Prev()` |
| **Archive Iteration** | `IteratorOptions{AllVersions: true}` | Range scan over `h/` with optional filtering |

### Transaction and Durability

**Batch Operations**:
- Use `pebble.Batch` for atomic commits
- Dual-write pattern: LSS + HSS operations in single batch
- Commit with `pebble.Sync` to match Badger's `SyncWrites(true)`

**Consistency Model**:
- Maintain existing merged in-memory transaction behavior
- Engine abstraction preserves transaction semantics
- Pebble snapshots for consistent read-only views when needed

---

## Engine Abstraction Design

### Interface Definition

```go
// Engine abstraction interface
type Engine interface {
    Open(path string, options EngineOptions) error
    Close() error
    NewBatch() Batch
    NewReader(version uint64) Reader
    NewSnapshot() Snapshot
}

// Batch operations
type Batch interface {
    SetAt(prefix []byte, key []byte, value []byte, version uint64) error
    DeleteAt(prefix []byte, key []byte, version uint64) error
    Commit(sync bool) error
    Close() error
}

// Reader operations  
type Reader interface {
    NewIterator(prefix []byte, reverse bool) Iterator
    NewBoundedIterator(lower, upper []byte, reverse bool) Iterator
    Get(key []byte) ([]byte, error)
    Close() error
}

// Iterator interface
type Iterator interface {
    Valid() bool
    Key() []byte
    Value() []byte
    Next() error
    Prev() error
    Seek(key []byte) error
    Close() error
}
```

### Implementation Strategy

**Badger Engine Adapter**:
- Wraps existing Badger functionality
- Maintains current MVCC behavior
- Preserves meta bit handling for compatibility

**Pebble Engine Implementation**:
- Implements Option-1 LSS/HSS layout
- Uses bounded iteration for historical queries
- Handles delete markers explicitly

**Key Benefits**:
- Preserves existing algorithm logic in `store/txn.go` and `store/indexer.go`
- Only concrete Reader/Writer implementations change
- Enables A/B testing and gradual migration

---

## Required API Changes

### 1. Eliminate Type Leakage

**Current Problem**:
```go
// lib/store.go - exposes Badger types
type StoreI interface {
    DB() *badger.DB  // ❌ Hard type dependency
    // ... other methods
}
```

**Solution**:
```go
// Replace with engine-neutral methods
type StoreI interface {
    CloneReadOnly(version uint64) StoreI
    CloneRW() StoreI
    Engine() Engine  // Opaque engine interface
    // ... other methods
}
```

**Update Sites**:
- `cmd/rpc/query.go` (lines 739-752): Use `CloneReadOnly()` instead of `DB()`

### 2. Engine-Neutral Transaction API

**Current Problem**:
```go
// store/txn.go - Badger-specific entry handling
type TxnWriterI interface {
    SetEntryAt(entry *badger.Entry, version uint64) error  // ❌ Badger types
}
```

**Solution**:
```go
// Engine-neutral transaction API
type TxnWriterI interface {
    SetAt(key []byte, value []byte, version uint64) error
    DeleteAt(key []byte, version uint64) error
}
```

### 3. Archive Iterator Semantics

**Current**: Uses Badger's `AllVersions: true` for MVCC iteration
**New**: Scan HSS ranges with optional user-key filtering

```go
// Keep method signature, change implementation
func (t *Txn) ArchiveIterator(prefix []byte) Iterator {
    // Pebble: scan h/<prefix> range instead of MVCC versions
}
```

---

## Migration Roadmap

### Phase 1: Engine Abstraction (No Runtime Changes)
1. **Introduce Engine Interface**
   - Define engine abstraction interfaces
   - Create Badger adapter implementation
   - Ensure zero runtime behavior changes

2. **Remove Type Leakage**
   - Replace `lib.StoreI.DB()` with `CloneReadOnly()`/`CloneRW()`
   - Update `cmd/rpc/query.go` call sites
   - Maintain backward compatibility

3. **Engine-Neutral Transaction API**
   - Replace `SetEntryAt()` with `SetAt()`/`DeleteAt()`
   - Update `store/txn.go` implementation
   - Preserve existing meta bit behavior in Badger adapter

### Phase 2: Pebble Integration
1. **Implement Pebble Engine**
   - Create Pebble engine with Option-1 layout
   - Implement LSS/HSS dual-write pattern
   - Add configuration switch (build tags or runtime config)

2. **Engine-Neutral Metrics**
   - Replace reflection-based metrics with counters
   - Track operations at transaction layer
   - Provide engine-agnostic metric collection

3. **Validation and Testing**
   - Extend test suite to cover both engines
   - Benchmark performance parity
   - Validate functional correctness

### Phase 3: Production Migration
1. **Deployment Strategy**
   - Separate data directories for rollback capability
   - Configuration-based engine selection
   - Gradual rollout with monitoring

2. **Data Migration** (if needed)
   - Export/import tooling for existing networks
   - Resync option for acceptable downtime scenarios
   - Validation of migrated data integrity

---

## Validation Plan

### Correctness Testing
- **Core Operations**: Set/Get/Delete across both engines with identical results
- **Iteration**: Forward/reverse prefix scans with consistent ordering
- **Historical Queries**: Height-bounded reads with exact data parity
- **SMT Operations**: Root hash and proof generation consistency
- **Indexer Functionality**: All existing indexer tests must pass
- **GC Operations**: Historical pruning by height ranges

### Performance Benchmarks
- **Memory Efficiency**: Validate 2-4 allocs/op vs 1000+ allocs/op improvements
- **Write Performance**: Ensure durability parity with `SyncWrites` equivalent
- **Read Performance**: Compare latest and historical read latencies
- **Compaction Behavior**: Monitor LSM tree efficiency at scale

### Integration Testing
- **End-to-End**: Full node operation with Pebble engine
- **Network Compatibility**: Ensure state root consistency across engines
- **Upgrade/Downgrade**: Validate migration and rollback procedures

---

## Risk Assessment and Mitigations

### Technical Risks

**Type Leakage Dependencies**:
- *Risk*: Hidden Badger type usage in downstream code
- *Mitigation*: Comprehensive grep audit + compilation testing

**MVCC Semantics Differences**:
- *Risk*: Code relying on Badger's internal MVCC for same-key versions
- *Mitigation*: Archive iterator provides HSS-based equivalent with grouping helpers

**Delete Behavior Changes**:
- *Risk*: Different tombstone/retention semantics
- *Mitigation*: Explicit HSS delete markers + range-based GC provide better control

**Metrics Divergence**:
- *Risk*: Loss of engine-specific performance insights
- *Mitigation*: Engine-neutral counters + Pebble-specific metrics via separate interface

### Operational Risks

**Migration Complexity**:
- *Risk*: Data corruption during engine transition
- *Mitigation*: Separate data directories + comprehensive validation + rollback capability

**Performance Regression**:
- *Risk*: Unexpected performance characteristics
- *Mitigation*: Extensive benchmarking + gradual rollout + monitoring

**Configuration Management**:
- *Risk*: Engine selection complexity
- *Mitigation*: Clear configuration options + documentation + default behaviors

---

## Implementation Checklist

### Code Changes Required

- [ ] **Engine Abstraction Layer**
  - [ ] Define engine interfaces (`Engine`, `Batch`, `Reader`, `Iterator`)
  - [ ] Implement Badger adapter with existing behavior
  - [ ] Create Pebble engine with Option-1 layout

- [ ] **API Refactoring**
  - [ ] Remove `lib.StoreI.DB()` method
  - [ ] Add `CloneReadOnly()` and `CloneRW()` methods
  - [ ] Update `cmd/rpc/query.go` call sites
  - [ ] Replace `TxnWriterI.SetEntryAt()` with engine-neutral methods

- [ ] **Metrics Replacement**
  - [ ] Remove reflection-based Badger metrics
  - [ ] Implement transaction-layer counters
  - [ ] Add engine-neutral metric collection

- [ ] **Configuration System**
  - [ ] Add engine selection configuration
  - [ ] Implement build tags or runtime switching
  - [ ] Document configuration options

### Testing Requirements

- [ ] **Unit Tests**
  - [ ] Engine interface compliance tests
  - [ ] Badger adapter correctness tests
  - [ ] Pebble engine correctness tests
  - [ ] Cross-engine consistency tests

- [ ] **Integration Tests**
  - [ ] Full store operation tests
  - [ ] Historical query validation
  - [ ] SMT proof consistency tests
  - [ ] Indexer functionality tests

- [ ] **Performance Tests**
  - [ ] Memory allocation benchmarks
  - [ ] Read/write latency comparisons
  - [ ] Large scan efficiency tests
  - [ ] Compaction behavior analysis

### Documentation Updates

- [ ] **Technical Documentation**
  - [ ] Engine abstraction design docs
  - [ ] Migration procedure documentation
  - [ ] Configuration reference
  - [ ] Performance comparison results

- [ ] **Operational Guides**
  - [ ] Deployment procedures
  - [ ] Monitoring and alerting setup
  - [ ] Troubleshooting guide
  - [ ] Rollback procedures

---

## Conclusion

The migration from Badger to Pebble is **technically feasible and highly beneficial** for Canopy. The Option-1 LSS/HSS layout provides:

1. **Significant Performance Improvements**: 2-4 allocs/op vs 1000+ allocs/op for large scans
2. **Full Functional Parity**: All existing capabilities preserved
3. **Clean Migration Path**: Engine abstraction enables smooth transition
4. **Operational Benefits**: Better compaction, memory management, and GC control

**Key Success Factors**:
- Implement engine abstraction layer first to decouple from Badger types
- Maintain existing transaction and iteration semantics
- Comprehensive testing across both engines
- Gradual migration with rollback capabilities

The primary blockers are API-level coupling rather than fundamental capability gaps, making this migration both achievable and worthwhile for Canopy's long-term performance and scalability goals.
