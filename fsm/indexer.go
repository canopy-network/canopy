package fsm

import (
	"bytes"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/codec"
)

// INDEXER.GO IS ONLY USED FOR CANOPY INDEXING RPC - NOT A CRITICAL PIECE OF THE STATE MACHINE

// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlobs(height uint64) (b *IndexerBlobs, err lib.ErrorI) {
	b = &IndexerBlobs{}
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		b.Previous, err = s.IndexerBlob(height - 1)
		if err != nil {
			return nil, err
		}
	}
	b.Current, err = s.IndexerBlob(height)
	if err != nil {
		return nil, err
	}
	return
}

// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlob(height uint64) (b *IndexerBlob, err lib.ErrorI) {
	if height == 0 || height > s.height {
		height = s.height
	}
	// Height semantics:
	// - `height` is the state version (pre-block-apply for block `height`).
	// - The latest committed block corresponding to that state is `height-1`.
	// This keeps the blob consistent with RPC/state-at-height conventions.
	if height <= 1 {
		// No committed block exists yet to pair with the state snapshot.
		return nil, lib.ErrWrongBlockHeight(0, 1)
	}
	blockHeight := height - 1
	sm, err := s.TimeMachine(height)
	if err != nil {
		return nil, err
	}
	if sm != s {
		defer sm.Discard()
	}
	// Use the snapshot store (not the live store) for all height-based indexer reads.
	st := sm.store.(lib.StoreI)
	// retrieve the block, transactions, and events
	block, err := st.GetBlockByHeight(blockHeight)
	if err != nil {
		return nil, err
	}
	if block == nil || block.BlockHeader == nil {
		return nil, lib.ErrNilBlockHeader()
	}
	if block.BlockHeader.Height == 0 || block.BlockHeader.Height != blockHeight {
		return nil, lib.ErrWrongBlockHeight(block.BlockHeader.Height, blockHeight)
	}
	// use sm for consistent snapshot reads at the requested height
	// retrieve the accounts
	accounts, err := sm.IterateAndAppend(AccountPrefix())
	if err != nil {
		return nil, err
	}
	// retrieve pools
	pools, err := sm.IterateAndAppend(PoolPrefix())
	if err != nil {
		return nil, err
	}
	// retrieve validators
	validators, err := sm.IterateAndAppend(ValidatorPrefix())
	if err != nil {
		return nil, err
	}
	// retrieve dex prices
	dexPrices, err := sm.GetDexPrices()
	if err != nil {
		return nil, err
	}
	// retrieve nonSigners
	nonSigners, err := sm.IterateAndAppend(NonSignerPrefix())
	if err != nil {
		return nil, err
	}
	// retrieve doubleSigners
	doubleSigners, err := st.GetDoubleSignersAsOf(blockHeight)
	if err != nil {
		return nil, err
	}
	// retrieve orders
	orderBooks, err := sm.GetOrderBooks()
	if err != nil {
		return nil, err
	}
	// retrieve params
	params, err := sm.GetParams()
	if err != nil {
		return nil, err
	}
	// retrieve dex batches
	dexBatches, err := sm.IterateAndAppend(lib.JoinLenPrefix(dexPrefix, lockedBatchSegment))
	if err != nil {
		return nil, err
	}
	// retrieve next dex batches
	nextDexBatches, err := sm.IterateAndAppend(lib.JoinLenPrefix(dexPrefix, nextBatchSement))
	if err != nil {
		return nil, err
	}
	// get the CommitteesData bytes under 'committees data prefix'
	committeesData, err := sm.Get(CommitteesDataPrefix())
	if err != nil {
		return nil, err
	}
	// get subsidized committees
	subsidizedCommittees, err := sm.GetSubsidizedCommittees()
	if err != nil {
		return nil, err
	}
	// get retired committees
	retiredCommittees, err := sm.GetRetiredCommittees()
	if err != nil {
		return nil, err
	}
	// get the supply tracker bytes from the state
	supply, err := sm.Get(SupplyPrefix())
	if err != nil {
		return nil, err
	}
	// marshal block to bytes
	blockBz, err := lib.Marshal(block)
	if err != nil {
		return nil, err
	}
	// marshal dex prices to bytes
	var dexPricesBz [][]byte
	for _, price := range dexPrices {
		priceBz, e := lib.Marshal(price)
		if e != nil {
			return nil, e
		}
		dexPricesBz = append(dexPricesBz, priceBz)
	}
	// marshal double signers to bytes
	var doubleSignersBz [][]byte
	for _, doubleSigner := range doubleSigners {
		doubleSignerBz, e := lib.Marshal(doubleSigner)
		if e != nil {
			return nil, e
		}
		doubleSignersBz = append(doubleSignersBz, doubleSignerBz)
	}
	// marshal order books to bytes
	orderBooksBz, err := lib.Marshal(orderBooks)
	if err != nil {
		return nil, err
	}
	// marshal params to bytes
	paramsBz, err := lib.Marshal(params)
	if err != nil {
		return nil, err
	}
	// return the blob
	return &IndexerBlob{
		Block:                blockBz,
		Accounts:             accounts,
		Pools:                pools,
		Validators:           validators,
		DexPrices:            dexPricesBz,
		NonSigners:           nonSigners,
		DoubleSigners:        doubleSignersBz,
		Orders:               orderBooksBz,
		Params:               paramsBz,
		DexBatches:           dexBatches,
		NextDexBatches:       nextDexBatches,
		CommitteesData:       committeesData,
		SubsidizedCommittees: subsidizedCommittees,
		RetiredCommittees:    retiredCommittees,
		Supply:               supply,
	}, nil
}

// DeltaIndexerBlobs returns a clone of blobs where account payloads are reduced
// to changed/added/removed entries. Pools, validators, and other entities remain full snapshots.
func DeltaIndexerBlobs(blobs *IndexerBlobs) (*IndexerBlobs, lib.ErrorI) {
	// no payload to reduce
	if blobs == nil {
		return nil, nil
	}
	// clone first to avoid mutating cache-backed or shared pointers
	out := cloneIndexerBlobs(blobs)
	// if current is nil there is nothing to diff
	if out == nil || out.Current == nil {
		return out, nil
	}
	// key current/previous by account address
	currentAccounts, currentAccountMap, err := accountEntries(out.Current.Accounts)
	if err != nil {
		return nil, err
	}
	previousAccounts, previousAccountMap, err := accountEntries(nilSafeBlob(out.Previous).Accounts)
	if err != nil {
		return nil, err
	}
	// include changed, added, and removed account keys
	currentAccountKeys, previousAccountKeys := changedAccountKeys(currentAccountMap, previousAccountMap)
	// force include reward/slash accounts so downstream reward/slash attribution never misses rows
	forcedAccountKeys, err := rewardSlashAccountKeys(out.Current.Block)
	if err != nil {
		return nil, err
	}
	forceIncludeAccounts(currentAccountKeys, previousAccountKeys, currentAccountMap, previousAccountMap, forcedAccountKeys)
	// rebuild account slices from selected key sets
	out.Current.Accounts = selectAccounts(currentAccounts, currentAccountKeys)
	if out.Previous != nil {
		out.Previous.Accounts = selectAccounts(previousAccounts, previousAccountKeys)
	}
	// pools, validators and all other entities remain full snapshots
	return out, nil
}

type accountEntry struct {
	key string
	bz  []byte
}

// accountEntries() builds:
//   - an ordered entry slice (preserves original order)
//   - an address->entry map for O(1) diff lookups
func accountEntries(entries [][]byte) ([]accountEntry, map[string][]byte, lib.ErrorI) {
	list := make([]accountEntry, 0, len(entries))
	entryMap := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		key, err := accountEntryKey(entry)
		if err != nil {
			return nil, nil, lib.ErrUnmarshal(err)
		}
		list = append(list, accountEntry{key: key, bz: entry})
		entryMap[key] = entry
	}
	return list, entryMap, nil
}

// accountEntryKey() extracts Account.address (field 1) without unmarshalling the full message.
func accountEntryKey(entry []byte) (string, error) {
	field, err := codec.GetRawProtoField(entry, 1)
	if err != nil {
		return "", err
	}
	return string(field), nil
}

// changedAccountKeys() returns the include sets for current and previous:
//   - current includes changed+added keys
//   - previous includes changed+removed keys
func changedAccountKeys(current, previous map[string][]byte) (map[string]struct{}, map[string]struct{}) {
	currentChanged := make(map[string]struct{})
	previousChanged := make(map[string]struct{})
	// changed or added in current
	for key, currentEntry := range current {
		if previousEntry, ok := previous[key]; !ok || !bytes.Equal(currentEntry, previousEntry) {
			currentChanged[key] = struct{}{}
		}
	}
	// changed or removed from previous
	for key, previousEntry := range previous {
		if currentEntry, ok := current[key]; !ok || !bytes.Equal(currentEntry, previousEntry) {
			previousChanged[key] = struct{}{}
		}
	}

	return currentChanged, previousChanged
}

// selectAccounts() keeps original entry order while filtering to the provided key set.
func selectAccounts(entries []accountEntry, include map[string]struct{}) [][]byte {
	selected := make([][]byte, 0, len(include))
	seen := make(map[string]struct{}, len(include))
	for _, entry := range entries {
		if _, ok := include[entry.key]; !ok {
			continue
		}
		if _, dup := seen[entry.key]; dup {
			continue
		}
		selected = append(selected, entry.bz)
		seen[entry.key] = struct{}{}
	}
	return selected
}

// forceIncludeAccounts() injects required account keys into current/previous include sets when present.
func forceIncludeAccounts(
	currentInclude, previousInclude map[string]struct{},
	current, previous map[string][]byte,
	keys map[string]struct{},
) {
	for key := range keys {
		if _, ok := current[key]; ok {
			currentInclude[key] = struct{}{}
		}
		if _, ok := previous[key]; ok {
			previousInclude[key] = struct{}{}
		}
	}
}

// rewardSlashAccountKeys() finds reward/slash event addresses in the current block.
// These addresses are always included in account deltas to preserve event attribution.
func rewardSlashAccountKeys(blockBz []byte) (map[string]struct{}, lib.ErrorI) {
	keys := make(map[string]struct{})
	if len(blockBz) == 0 {
		return keys, nil
	}
	block := new(lib.BlockResult)
	if err := lib.Unmarshal(blockBz, block); err != nil {
		return nil, err
	}
	for _, event := range block.Events {
		if event == nil || len(event.Address) == 0 {
			continue
		}
		switch event.EventType {
		case string(lib.EventTypeReward), string(lib.EventTypeSlash):
			keys[string(event.Address)] = struct{}{}
		}
	}
	return keys, nil
}

// cloneIndexerBlobs() clones the top-level current/previous wrapper.
func cloneIndexerBlobs(src *IndexerBlobs) *IndexerBlobs {
	if src == nil {
		return nil
	}
	return &IndexerBlobs{
		Current:  cloneIndexerBlob(src.Current),
		Previous: cloneIndexerBlob(src.Previous),
	}
}

// cloneIndexerBlob() performs a lightweight structural copy.
// The underlying byte payloads are shared read-only; delta logic replaces only
// Accounts slice headers on the clone so cached snapshots remain untouched.
func cloneIndexerBlob(src *IndexerBlob) *IndexerBlob {
	if src == nil {
		return nil
	}
	return &IndexerBlob{
		Block:                src.Block,
		Accounts:             src.Accounts,
		Pools:                src.Pools,
		Validators:           src.Validators,
		DexPrices:            src.DexPrices,
		NonSigners:           src.NonSigners,
		DoubleSigners:        src.DoubleSigners,
		Orders:               src.Orders,
		Params:               src.Params,
		DexBatches:           src.DexBatches,
		NextDexBatches:       src.NextDexBatches,
		CommitteesData:       src.CommitteesData,
		SubsidizedCommittees: src.SubsidizedCommittees,
		RetiredCommittees:    src.RetiredCommittees,
		Supply:               src.Supply,
	}
}

// nilSafeBlob() normalizes a nil blob to an empty blob for helper calls.
func nilSafeBlob(blob *IndexerBlob) *IndexerBlob {
	if blob != nil {
		return blob
	}
	return &IndexerBlob{}
}
