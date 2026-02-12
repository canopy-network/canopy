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
	// calculate validator/delegator status totals from the full snapshot
	totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking,
		totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking, err := validatorTotals(validators)
	if err != nil {
		return nil, err
	}
	// return the blob
	return &IndexerBlob{
		Block:                    blockBz,
		Accounts:                 accounts,
		Pools:                    pools,
		Validators:               validators,
		DexPrices:                dexPricesBz,
		NonSigners:               nonSigners,
		DoubleSigners:            doubleSignersBz,
		Orders:                   orderBooksBz,
		Params:                   paramsBz,
		DexBatches:               dexBatches,
		NextDexBatches:           nextDexBatches,
		CommitteesData:           committeesData,
		SubsidizedCommittees:     subsidizedCommittees,
		RetiredCommittees:        retiredCommittees,
		Supply:                   supply,
		TotalValidatorsActive:    totalValidatorsActive,
		TotalValidatorsPaused:    totalValidatorsPaused,
		TotalValidatorsUnstaking: totalValidatorsUnstaking,
		TotalDelegatesActive:     totalDelegatesActive,
		TotalDelegatesPaused:     totalDelegatesPaused,
		TotalDelegatesUnstaking:  totalDelegatesUnstaking,
	}, nil
}

// DeltaIndexerBlobs returns a clone of blobs where account, pool, and validator payloads
// are reduced to changed/added/removed entries. Other entities remain full snapshots.
func DeltaIndexerBlobs(blobs *IndexerBlobs) (*IndexerBlobs, lib.ErrorI) {
	if blobs == nil {
		return nil, nil
	}
	out := cloneIndexerBlobs(blobs)
	if out == nil || out.Current == nil {
		return out, nil
	}
	previous := nilSafeBlob(out.Previous)

	// accounts: changed+added in current, changed+removed in previous
	currentAccounts, currentAccountMap, err := accountEntries(out.Current.Accounts)
	if err != nil {
		return nil, err
	}
	previousAccounts, previousAccountMap, err := accountEntries(previous.Accounts)
	if err != nil {
		return nil, err
	}
	currentAccountKeys, previousAccountKeys := changedBlobKeys(currentAccountMap, previousAccountMap)
	forcedAccountKeys, err := rewardSlashAccountKeys(out.Current.Block)
	if err != nil {
		return nil, err
	}
	forceIncludeKeys(currentAccountKeys, previousAccountKeys, currentAccountMap, previousAccountMap, forcedAccountKeys)
	out.Current.Accounts = selectBlobEntries(currentAccounts, currentAccountKeys)
	if out.Previous != nil {
		out.Previous.Accounts = selectBlobEntries(previousAccounts, previousAccountKeys)
	}

	// pools: changed+added in current, changed+removed in previous
	currentPools, currentPoolMap, err := poolEntries(out.Current.Pools)
	if err != nil {
		return nil, err
	}
	previousPools, previousPoolMap, err := poolEntries(previous.Pools)
	if err != nil {
		return nil, err
	}
	currentPoolKeys, previousPoolKeys := changedBlobKeys(currentPoolMap, previousPoolMap)
	out.Current.Pools = selectBlobEntries(currentPools, currentPoolKeys)
	if out.Previous != nil {
		out.Previous.Pools = selectBlobEntries(previousPools, previousPoolKeys)
	}

	// validators: changed+added in current, changed+removed in previous
	currentValidators, currentValidatorMap, currentOutputIndex, err := validatorEntries(out.Current.Validators)
	if err != nil {
		return nil, err
	}
	previousValidators, previousValidatorMap, previousOutputIndex, err := validatorEntries(previous.Validators)
	if err != nil {
		return nil, err
	}
	currentValidatorKeys, previousValidatorKeys := changedBlobKeys(currentValidatorMap, previousValidatorMap)
	forcedValidatorKeys, err := validatorForceKeys(out.Current.Block, currentOutputIndex, previousOutputIndex)
	if err != nil {
		return nil, err
	}
	forceIncludeKeys(currentValidatorKeys, previousValidatorKeys, currentValidatorMap, previousValidatorMap, forcedValidatorKeys)
	out.Current.Validators = selectBlobEntries(currentValidators, currentValidatorKeys)
	out.Current.ValidatorsDelta = true
	if out.Previous != nil {
		out.Previous.Validators = selectBlobEntries(previousValidators, previousValidatorKeys)
		out.Previous.ValidatorsDelta = true
	}
	return out, nil
}

type blobEntry struct {
	key string
	bz  []byte
}

func accountEntries(entries [][]byte) ([]blobEntry, map[string][]byte, lib.ErrorI) {
	return entriesByKey(entries, accountEntryKey)
}

func poolEntries(entries [][]byte) ([]blobEntry, map[string][]byte, lib.ErrorI) {
	return entriesByKey(entries, poolEntryKey)
}

func validatorEntries(entries [][]byte) ([]blobEntry, map[string][]byte, map[string][]string, lib.ErrorI) {
	out := make([]blobEntry, 0, len(entries))
	entriesByAddress := make(map[string][]byte, len(entries))
	outputToValidator := make(map[string][]string)
	for _, entry := range entries {
		validator := new(Validator)
		if err := lib.Unmarshal(entry, validator); err != nil {
			return nil, nil, nil, err
		}
		key := string(validator.Address)
		out = append(out, blobEntry{key: key, bz: entry})
		entriesByAddress[key] = entry
		if len(validator.Output) > 0 {
			outputToValidator[string(validator.Output)] = append(outputToValidator[string(validator.Output)], key)
		}
	}
	return out, entriesByAddress, outputToValidator, nil
}

func entriesByKey(entries [][]byte, keyExtractor func([]byte) (string, error)) ([]blobEntry, map[string][]byte, lib.ErrorI) {
	out := make([]blobEntry, 0, len(entries))
	entryMap := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		key, err := keyExtractor(entry)
		if err != nil {
			return nil, nil, lib.ErrUnmarshal(err)
		}
		out = append(out, blobEntry{key: key, bz: entry})
		entryMap[key] = entry
	}
	return out, entryMap, nil
}

func accountEntryKey(entry []byte) (string, error) {
	field, err := codec.GetRawProtoField(entry, 1) // Account.address
	if err != nil {
		return "", err
	}
	return string(field), nil
}

func poolEntryKey(entry []byte) (string, error) {
	field, err := codec.GetRawProtoField(entry, 1) // Pool.id
	if err != nil {
		return "", err
	}
	return string(field), nil
}

func changedBlobKeys(current, previous map[string][]byte) (map[string]struct{}, map[string]struct{}) {
	currentChanged := make(map[string]struct{})
	previousChanged := make(map[string]struct{})
	for key, currentEntry := range current {
		if previousEntry, ok := previous[key]; !ok || !bytes.Equal(currentEntry, previousEntry) {
			currentChanged[key] = struct{}{}
		}
	}
	for key, previousEntry := range previous {
		if currentEntry, ok := current[key]; !ok || !bytes.Equal(currentEntry, previousEntry) {
			previousChanged[key] = struct{}{}
		}
	}
	return currentChanged, previousChanged
}

func selectBlobEntries(entries []blobEntry, include map[string]struct{}) [][]byte {
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

func forceIncludeKeys(
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

// validatorForceKeys() includes validators tied to lifecycle/reward events.
func validatorForceKeys(blockBz []byte, currentOutputIndex, previousOutputIndex map[string][]string) (map[string]struct{}, lib.ErrorI) {
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
		eventKey := string(event.Address)
		switch event.EventType {
		case string(lib.EventTypeReward):
			keys[eventKey] = struct{}{}
			for _, validatorKey := range currentOutputIndex[eventKey] {
				keys[validatorKey] = struct{}{}
			}
			for _, validatorKey := range previousOutputIndex[eventKey] {
				keys[validatorKey] = struct{}{}
			}
		case string(lib.EventTypeSlash),
			string(lib.EventTypeAutoPause),
			string(lib.EventTypeAutoBeginUnstaking),
			string(lib.EventTypeFinishUnstaking):
			keys[eventKey] = struct{}{}
		}
	}
	return keys, nil
}

func validatorTotals(validators [][]byte) (
	totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking uint32,
	totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking uint32,
	err lib.ErrorI,
) {
	for _, entry := range validators {
		validator := new(Validator)
		if err = lib.Unmarshal(entry, validator); err != nil {
			return
		}
		if validator.UnstakingHeight > 0 {
			totalValidatorsUnstaking++
			if validator.Delegate {
				totalDelegatesUnstaking++
			}
			continue
		}
		if validator.MaxPausedHeight > 0 {
			totalValidatorsPaused++
			if validator.Delegate {
				totalDelegatesPaused++
			}
			continue
		}
		totalValidatorsActive++
		if validator.Delegate {
			totalDelegatesActive++
		}
	}
	return
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
// Accounts/Pools/Validators slice headers on the clone so cached snapshots remain untouched.
func cloneIndexerBlob(src *IndexerBlob) *IndexerBlob {
	if src == nil {
		return nil
	}
	return &IndexerBlob{
		Block:                    src.Block,
		Accounts:                 src.Accounts,
		Pools:                    src.Pools,
		Validators:               src.Validators,
		DexPrices:                src.DexPrices,
		NonSigners:               src.NonSigners,
		DoubleSigners:            src.DoubleSigners,
		Orders:                   src.Orders,
		Params:                   src.Params,
		DexBatches:               src.DexBatches,
		NextDexBatches:           src.NextDexBatches,
		CommitteesData:           src.CommitteesData,
		SubsidizedCommittees:     src.SubsidizedCommittees,
		RetiredCommittees:        src.RetiredCommittees,
		Supply:                   src.Supply,
		TotalValidatorsActive:    src.TotalValidatorsActive,
		TotalValidatorsPaused:    src.TotalValidatorsPaused,
		TotalValidatorsUnstaking: src.TotalValidatorsUnstaking,
		ValidatorsDelta:          src.ValidatorsDelta,
		TotalDelegatesActive:     src.TotalDelegatesActive,
		TotalDelegatesPaused:     src.TotalDelegatesPaused,
		TotalDelegatesUnstaking:  src.TotalDelegatesUnstaking,
	}
}

// nilSafeBlob() normalizes a nil blob to an empty blob for helper calls.
func nilSafeBlob(blob *IndexerBlob) *IndexerBlob {
	if blob != nil {
		return blob
	}
	return &IndexerBlob{}
}
