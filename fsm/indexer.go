package fsm

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/codec"
	"github.com/canopy-network/canopy/lib/crypto"
)

// INDEXER.GO IS ONLY USED FOR CANOPY INDEXING RPC - NOT A CRITICAL PIECE OF THE STATE MACHINE

// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlobs(ctx context.Context, height uint64) (b *IndexerBlobs, err lib.ErrorI) {
	b = &IndexerBlobs{}
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		b.Previous, err = s.IndexerBlob(ctx, height-1)
		if err != nil {
			return nil, err
		}
	}
	b.Current, err = s.IndexerBlob(ctx, height)
	if err != nil {
		return nil, err
	}
	return
}

// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlob(ctx context.Context, height uint64) (b *IndexerBlob, err lib.ErrorI) {
	return s.indexerBlob(ctx, height, nil, nil, nil, false, false)
}

// IndexerBlobsFromStateChanges builds an account-sparse delta using the state
// keys journaled for height. available=false means the height predates the
// journal and callers must use the legacy full-snapshot comparison.
func (s *StateMachine) IndexerBlobsFromStateChanges(ctx context.Context, height uint64) (b *IndexerBlobs, available bool, err lib.ErrorI) {
	// Honor an already-cancelled/disconnected caller before doing any store work --
	// this path is fast, not free, and skips the per-iteration ctx checks that
	// IterateAndAppend relies on in the legacy full-snapshot path below.
	if cErr := ctx.Err(); cErr != nil {
		return nil, false, lib.ErrCancelled(cErr)
	}
	if height == 0 || height > s.height {
		height = s.height
	}
	// At the genesis boundary there is no previous blob to compare against, so
	// the indexer contract requires the complete current account snapshot. The
	// height journal only contains changes made by block 1 and cannot represent
	// genesis accounts that remained untouched.
	if height == 2 {
		return nil, false, nil
	}
	st, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, false, nil
	}
	accountKeys, available, err := st.StateChangeKeys(height, AccountPrefix())
	if err != nil || !available {
		return nil, available, err
	}
	validatorKeys, _, err := st.StateChangeKeys(height, ValidatorPrefix())
	if err != nil {
		return nil, true, err
	}
	nonSignerKeys, _, err := st.StateChangeKeys(height, NonSignerPrefix())
	if err != nil {
		return nil, true, err
	}

	b = &IndexerBlobs{}
	b.Current, err = s.indexerBlob(ctx, height, accountKeys, validatorKeys, nonSignerKeys, true, true)
	if err != nil {
		return nil, true, err
	}
	// Reward/slash events can require an unchanged account in the delta. Use
	// the current block's event addresses for both snapshots so the previous
	// side cannot look like an unrelated account deletion.
	eventAccountKeys, eventErr := rewardSlashAccountStateKeys(b.Current.Block)
	if eventErr != nil {
		return nil, true, eventErr
	}
	accountKeys = append(accountKeys, eventAccountKeys...)
	if height > 2 {
		b.Previous, err = s.indexerBlob(ctx, height-1, accountKeys, validatorKeys, nonSignerKeys, true, false)
		if err != nil {
			return nil, true, err
		}
	}
	// Previously uninstrumented: query.go's delta_compute timer only ever wrapped the
	// legacy branch's DeltaIndexerBlobs call, leaving the journal path's own delta
	// cost invisible. Labeling this "journal" is what actually answers "how good is
	// the journal path" instead of assuming it from the legacy-only number.
	deltaComputeStart := time.Now()
	b, err = DeltaIndexerBlobs(b)
	s.Metrics.ObserveIndexerBlobStep("delta_compute", "journal", "n_a", deltaComputeStart)
	return b, true, err
}

func (s *StateMachine) indexerBlob(ctx context.Context, height uint64, accountKeys, validatorKeys, nonSignerKeys [][]byte, selective, includeBlockEventAccounts bool) (b *IndexerBlob, err lib.ErrorI) {
	// path labels every ObserveIndexerBlobStep call below so journal-sourced and
	// legacy-sourced calls to the same step (accounts_iterate above all) are no
	// longer indistinguishable in the aggregate -- selective is exactly the
	// journal/legacy branch signal already threaded through this function.
	path := "legacy"
	if selective {
		path = "journal"
	}
	if height == 0 || height > s.height {
		height = s.height
	}
	// store.Store.NewReadOnly routes state reads through the LSS (Latest State Store)
	// only when the requested height equals the store's current live version; every
	// other height -- including tip-1, the "previous" side of a live headscan request
	// -- goes through the HSS (Historic State Store) instead. This is the actual
	// mechanism behind steps that show a fast/slow split despite having no journal at
	// all (validators_iterate, block_non_signers_get): it isn't proximity-based cache
	// warmth or "how old is this height," it's this one deterministic check.
	tier := "hss"
	if height == s.height {
		tier = "lss"
	}
	s.Metrics.RecordIndexerBlobTier(tier)
	// Height semantics:
	// - `height` is the state version (pre-block-apply for block `height`).
	// - The latest committed block corresponding to that state is `height-1`.
	// This keeps the blob consistent with RPC/state-at-height conventions.
	if height <= 1 {
		// No committed block exists yet to pair with the state snapshot.
		return nil, lib.ErrWrongBlockHeight(0, 1)
	}
	blockHeight := height - 1
	stepStart := time.Now()
	sm, err := s.TimeMachine(height)
	s.Metrics.ObserveIndexerBlobStep("time_machine", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	if sm != s {
		defer sm.Discard()
	}
	// Use the snapshot store (not the live store) for all height-based indexer reads.
	st := sm.store.(lib.StoreI)
	// retrieve the block, transactions, and events
	stepStart = time.Now()
	block, err := st.GetBlockByHeight(blockHeight)
	s.Metrics.ObserveIndexerBlobStep("block_fetch", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	if block == nil || block.BlockHeader == nil {
		return nil, lib.ErrNilBlockHeader()
	}
	if block.BlockHeader.Height == 0 || block.BlockHeader.Height != blockHeight {
		return nil, lib.ErrWrongBlockHeight(block.BlockHeader.Height, blockHeight)
	}
	// marshal block to bytes -- moved up from below so both the account event-key
	// logic and the validator force-include logic can use it.
	blockBz, err := lib.Marshal(block)
	if err != nil {
		return nil, err
	}
	// use sm for consistent snapshot reads at the requested height
	// retrieve either the complete account snapshot (legacy path) or only the
	// keys touched by the requested commit (journal path).
	stepStart = time.Now()
	var accounts [][]byte
	if !selective {
		accounts, err = sm.IterateAndAppend(ctx, AccountPrefix())
	} else {
		if includeBlockEventAccounts {
			eventAccountKeys, eventErr := rewardSlashAccountStateKeys(blockBz)
			if eventErr != nil {
				return nil, eventErr
			}
			accountKeys = append(accountKeys, eventAccountKeys...)
		}
		accounts, err = sm.valuesForStateKeys(accountKeys, AccountPrefix())
	}
	s.Metrics.ObserveIndexerBlobStep("accounts_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve pools
	stepStart = time.Now()
	pools, err := sm.IterateAndAppend(ctx, PoolPrefix())
	s.Metrics.ObserveIndexerBlobStep("pools_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve validators
	stepStart = time.Now()
	var validators [][]byte
	if !selective {
		validators, err = sm.IterateAndAppend(ctx, ValidatorPrefix())
	} else {
		forced, forceErr := validatorForceKeysByAddress(blockBz)
		if forceErr != nil {
			return nil, forceErr
		}
		validators, err = sm.valuesForStateKeys(append(validatorKeys, forced...), ValidatorPrefix())
	}
	s.Metrics.ObserveIndexerBlobStep("validators_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve dex prices
	stepStart = time.Now()
	dexPrices, err := sm.GetDexPrices()
	s.Metrics.ObserveIndexerBlobStep("dex_prices_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve nonSigners
	stepStart = time.Now()
	var nonSigners [][]byte
	if !selective {
		nonSigners, err = sm.IterateAndAppend(ctx, NonSignerPrefix())
	} else {
		for _, key := range nonSignerKeys {
			addr, addrErr := AddressFromKey(key)
			if addrErr != nil {
				return nil, addrErr
			}
			bz, getErr := sm.Get(key)
			if getErr != nil {
				return nil, getErr
			}
			if bz == nil {
				continue
			}
			ns := new(NonSigner)
			if unmarshalErr := lib.Unmarshal(bz, ns); unmarshalErr != nil {
				return nil, unmarshalErr
			}
			ns.Address = addr.Bytes()
			nsBz, marshalErr := lib.Marshal(ns)
			if marshalErr != nil {
				return nil, marshalErr
			}
			nonSigners = append(nonSigners, nsBz)
		}
	}
	s.Metrics.ObserveIndexerBlobStep("non_signers_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve doubleSigners
	stepStart = time.Now()
	doubleSigners, err := st.GetDoubleSignersAsOf(blockHeight)
	s.Metrics.ObserveIndexerBlobStep("double_signers_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve per-block non-signers from the committed QC for this block
	stepStart = time.Now()
	blockNonSigners, err := sm.blockNonSignerAddresses(blockHeight)
	s.Metrics.ObserveIndexerBlobStep("block_non_signers_get", path, tier, stepStart)
	if err != nil {
		blockNonSigners = nil
	}
	// retrieve orders
	stepStart = time.Now()
	orderBooks, err := sm.GetOrderBooks()
	s.Metrics.ObserveIndexerBlobStep("order_books_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve params
	stepStart = time.Now()
	params, err := sm.GetParams()
	s.Metrics.ObserveIndexerBlobStep("params_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve dex batches
	stepStart = time.Now()
	dexBatches, err := sm.IterateAndAppend(ctx, lib.JoinLenPrefix(dexPrefix, lockedBatchSegment))
	s.Metrics.ObserveIndexerBlobStep("dex_batches_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// retrieve next dex batches
	stepStart = time.Now()
	nextDexBatches, err := sm.IterateAndAppend(ctx, lib.JoinLenPrefix(dexPrefix, nextBatchSement))
	s.Metrics.ObserveIndexerBlobStep("next_dex_batches_iterate", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// get the CommitteesData bytes under 'committees data prefix'
	stepStart = time.Now()
	committeesData, err := sm.Get(CommitteesDataPrefix())
	s.Metrics.ObserveIndexerBlobStep("committees_data_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// get subsidized committees
	stepStart = time.Now()
	subsidizedCommittees, err := sm.GetSubsidizedCommittees()
	s.Metrics.ObserveIndexerBlobStep("subsidized_committees_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// get retired committees
	stepStart = time.Now()
	retiredCommittees, err := sm.GetRetiredCommittees()
	s.Metrics.ObserveIndexerBlobStep("retired_committees_get", path, tier, stepStart)
	if err != nil {
		return nil, err
	}
	// get the supply tracker bytes from the state
	stepStart = time.Now()
	supply, err := sm.Get(SupplyPrefix())
	s.Metrics.ObserveIndexerBlobStep("supply_get", path, tier, stepStart)
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
	// journal path resolves totals from a persisted incremental baseline (full scan fallback
	// only if none exists); legacy path always computes them from the full validator set.
	var totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking uint32
	var totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking uint32
	if !selective {
		totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking,
			totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking, err = validatorTotals(validators)
		if err != nil {
			return nil, err
		}
	} else {
		// sm (not s) is this function's height-scoped snapshot - the fallback scan must use it
		// too, or a non-head height's first fallback reads the live head's validators, not height's.
		var totals *lib.ValidatorTotals
		var totalsErr lib.ErrorI
		if includeBlockEventAccounts {
			// Current blob: fetch each touched validator's height-1 status too - without
			// it, a validator that changes status gets counted in both categories instead of moving between them.
			forced, forceErr := validatorForceKeysByAddress(blockBz)
			if forceErr != nil {
				return nil, forceErr
			}
			touchedKeys := append(append([][]byte{}, validatorKeys...), forced...)
			previousValidators, prevErr := s.previousValidatorEntries(height, touchedKeys)
			if prevErr != nil {
				return nil, prevErr
			}
			totals, totalsErr = sm.resolveValidatorTotals(st, height, validators, previousValidators)
		} else {
			// Previous blob reuses the Current call's validator keys (for DeltaIndexerBlobs' value diff) -
			// wrong set for a totals diff, so read back the already-resolved totals instead.
			totals, totalsErr = sm.totalsAtHeight(st, height)
		}
		if totalsErr != nil {
			return nil, totalsErr
		}
		totalValidatorsActive, totalValidatorsPaused, totalValidatorsUnstaking =
			totals.ValidatorsActive, totals.ValidatorsPaused, totals.ValidatorsUnstaking
		totalDelegatesActive, totalDelegatesPaused, totalDelegatesUnstaking =
			totals.DelegatesActive, totals.DelegatesPaused, totals.DelegatesUnstaking
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
		BlockNonSigners:          blockNonSigners,
		NonSignersDelta:          selective,
	}, nil
}

// valuesForStateKeys() returns the byte array for state keys
func (s *StateMachine) valuesForStateKeys(keys [][]byte, prefix []byte) ([][]byte, lib.ErrorI) {
	values := make([][]byte, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if !bytes.HasPrefix(key, prefix) {
			continue
		}
		keyString := string(key)
		if _, ok := seen[keyString]; ok {
			continue
		}
		seen[keyString] = struct{}{}
		value, err := s.Get(key)
		if err != nil {
			return nil, err
		}
		if value != nil {
			values = append(values, value)
		}
	}
	return values, nil
}

func (s *StateMachine) blockNonSignerAddresses(blockHeight uint64) ([][]byte, lib.ErrorI) {
	if blockHeight <= 1 {
		return nil, nil
	}
	qc, err := s.LoadCertificateHashesOnly(blockHeight)
	if err != nil {
		return nil, err
	}
	if qc == nil || qc.Header == nil || qc.Signature == nil {
		return nil, nil
	}
	committee, err := s.LoadCommittee(qc.Header.ChainId, qc.Header.RootHeight)
	if err != nil {
		return nil, err
	}
	if committee.ValidatorSet == nil {
		return nil, nil
	}
	nonSignerPubKeys, _, err := qc.GetNonSigners(committee.ValidatorSet)
	if err != nil {
		return nil, err
	}
	addresses := make([][]byte, 0, len(nonSignerPubKeys))
	for _, pubKeyBytes := range nonSignerPubKeys {
		pubKey, e := crypto.NewPublicKeyFromBytes(pubKeyBytes)
		if e != nil {
			return nil, lib.ErrPubKeyFromBytes(e)
		}
		addresses = append(addresses, pubKey.Address().Bytes())
	}
	return addresses, nil
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
	// currentAccounts/previousAccounts come from a Pebble prefix scan over AccountPrefix(),
	// so both are already ascending by address bytes - accountEntryKey extracts that same
	// address as the raw key, so entry order matches key order and the merge walk is safe.
	currentAccounts, currentAccountMap, err := accountEntries(out.Current.Accounts)
	if err != nil {
		return nil, err
	}
	previousAccounts, previousAccountMap, err := accountEntries(previous.Accounts)
	if err != nil {
		return nil, err
	}
	currentAccountKeys, previousAccountKeys := mergeChangedBlobKeys(currentAccounts, previousAccounts)
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
	// NOT converted to the sorted merge walk: poolEntryKey extracts Pool.Id's raw varint
	// wire bytes, but the Pebble storage key (KeyForPool) encodes Id big-endian - the two
	// orderings diverge at varint length boundaries (e.g. id 16383 vs 16384), so entry order
	// here does not reliably match key order. Map-based diff stays correct regardless of order.
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
	// Same order guarantee as accounts: validatorEntries' key is the unmarshalled
	// validator.Address, matching KeyForValidator's raw address bytes.
	currentValidators, currentValidatorMap, currentOutputIndex, err := validatorEntries(out.Current.Validators)
	if err != nil {
		return nil, err
	}
	previousValidators, previousValidatorMap, previousOutputIndex, err := validatorEntries(previous.Validators)
	if err != nil {
		return nil, err
	}
	currentValidatorKeys, previousValidatorKeys := mergeChangedBlobKeys(currentValidators, previousValidators)
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

// entryKeyOrZero extracts field #1 as a map key. In proto3, absent scalar
// fields represent the zero value on the wire (Pool{Id:0}, Account{Address:nil}),
// so codec.ErrFieldNotFound is a legal zero-value signal, not a parse error.
// Real proto-parse errors (invalid tags, buffer overruns) still surface.
func entryKeyOrZero(entry []byte) (string, error) {
	field, err := codec.GetRawProtoField(entry, 1)
	if err != nil {
		if errors.Is(err, codec.ErrFieldNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(field), nil
}

func accountEntryKey(entry []byte) (string, error) {
	return entryKeyOrZero(entry) // Account.address
}

func poolEntryKey(entry []byte) (string, error) {
	return entryKeyOrZero(entry) // Pool.id
}

// mergeChangedBlobKeys() is changedBlobKeys' sorted-input equivalent: current and previous
// must each be ascending by key (true for a Pebble prefix-scan result - see call sites).
// A two-pointer merge walk finds the same added/changed/removed keys as the map-based
// version without hashing every key into a map[string][]byte first.
func mergeChangedBlobKeys(current, previous []blobEntry) (map[string]struct{}, map[string]struct{}) {
	currentChanged := make(map[string]struct{})
	previousChanged := make(map[string]struct{})
	i, j := 0, 0
	for i < len(current) && j < len(previous) {
		c, p := current[i], previous[j]
		switch {
		case c.key < p.key:
			currentChanged[c.key] = struct{}{}
			i++
		case c.key > p.key:
			previousChanged[p.key] = struct{}{}
			j++
		default:
			if !bytes.Equal(c.bz, p.bz) {
				currentChanged[c.key] = struct{}{}
				previousChanged[p.key] = struct{}{}
			}
			i++
			j++
		}
	}
	for ; i < len(current); i++ {
		currentChanged[current[i].key] = struct{}{}
	}
	for ; j < len(previous); j++ {
		previousChanged[previous[j].key] = struct{}{}
	}
	return currentChanged, previousChanged
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

// rewardSlashAccountStateKeys() returns account storage keys referenced by reward/slash events
// Sparse journal reads use them to include event accounts even when their state is unchanged
func rewardSlashAccountStateKeys(blockBz []byte) ([][]byte, lib.ErrorI) {
	addresses, err := rewardSlashAccountKeys(blockBz)
	if err != nil {
		return nil, err
	}
	keys := make([][]byte, 0, len(addresses))
	for address := range addresses {
		keys = append(keys, lib.JoinLenPrefix(accountPrefix, []byte(address)))
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

// validatorForceKeysByAddress force-includes validators from lifecycle/reward events by
// their own address - reward events name the operator address directly (committee.go's DistributeCommitteeReward), so no output resolution is needed.
func validatorForceKeysByAddress(blockBz []byte) ([][]byte, lib.ErrorI) {
	keys := make([][]byte, 0)
	if len(blockBz) == 0 {
		return keys, nil
	}
	block := new(lib.BlockResult)
	if err := lib.Unmarshal(blockBz, block); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	for _, event := range block.Events {
		if event == nil || len(event.Address) == 0 {
			continue
		}
		switch event.EventType {
		case string(lib.EventTypeReward), string(lib.EventTypeSlash),
			string(lib.EventTypeAutoPause), string(lib.EventTypeAutoBeginUnstaking),
			string(lib.EventTypeFinishUnstaking):
			addr := string(event.Address)
			if _, ok := seen[addr]; ok {
				continue
			}
			seen[addr] = struct{}{}
			keys = append(keys, lib.JoinLenPrefix(validatorPrefix, event.Address))
		}
	}
	return keys, nil
}

// resolveValidatorTotals derives height's totals from the persisted baseline at height-1
// plus this blob's validator transitions, falling back to a full scan when no baseline exists (e.g. after a restart or cache eviction).
func (s *StateMachine) resolveValidatorTotals(st lib.StoreI, height uint64, current, previous [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	return st.GetOrComputeValidatorTotals(height, func() (*lib.ValidatorTotals, lib.ErrorI) {
		baseline, available, err := st.GetValidatorTotals(height - 1)
		if err != nil {
			return nil, err
		}
		if !available {
			full, fullErr := s.fullValidatorSnapshotForTotals()
			if fullErr != nil {
				return nil, fullErr
			}
			return totalsFromFullScan(full)
		}
		return applyValidatorTransitions(baseline, current, previous)
	})
}

// fullValidatorSnapshotForTotals does the one-time full scan used only when no baseline
// exists yet for height-1.
func (s *StateMachine) fullValidatorSnapshotForTotals() ([][]byte, lib.ErrorI) {
	return s.IterateAndAppend(context.Background(), ValidatorPrefix())
}

// previousValidatorEntries returns the given keys' values as of height-1, so
// applyValidatorTransitions can see each validator's status before height's changes, not just after.
func (s *StateMachine) previousValidatorEntries(height uint64, keys [][]byte) ([][]byte, lib.ErrorI) {
	if height < 2 || len(keys) == 0 {
		return nil, nil
	}
	prevSM, err := s.TimeMachine(height - 1)
	if err != nil {
		return nil, err
	}
	if prevSM != s {
		defer prevSM.Discard()
	}
	return prevSM.valuesForStateKeys(keys, ValidatorPrefix())
}

// totalsAtHeight reads back height's already-persisted totals (full-scanning once if none
// exist) - unlike resolveValidatorTotals it never diffs, since Previous's fetched validators belong to a different height.
func (s *StateMachine) totalsAtHeight(st lib.StoreI, height uint64) (*lib.ValidatorTotals, lib.ErrorI) {
	return st.GetOrComputeValidatorTotals(height, func() (*lib.ValidatorTotals, lib.ErrorI) {
		full, err := s.fullValidatorSnapshotForTotals()
		if err != nil {
			return nil, err
		}
		return totalsFromFullScan(full)
	})
}

func totalsFromFullScan(validators [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	active, paused, unstaking, delActive, delPaused, delUnstaking, err := validatorTotals(validators)
	if err != nil {
		return nil, err
	}
	return &lib.ValidatorTotals{
		ValidatorsActive: active, ValidatorsPaused: paused, ValidatorsUnstaking: unstaking,
		DelegatesActive: delActive, DelegatesPaused: delPaused, DelegatesUnstaking: delUnstaking,
	}, nil
}

// applyValidatorTransitions diffs current/previous against the baseline; a previous entry
// missing from current is a deletion (e.g. EventFinishUnstaking) - decrement only.
func applyValidatorTransitions(baseline *lib.ValidatorTotals, current, previous [][]byte) (*lib.ValidatorTotals, lib.ErrorI) {
	totals := &lib.ValidatorTotals{
		ValidatorsActive: baseline.ValidatorsActive, ValidatorsPaused: baseline.ValidatorsPaused,
		ValidatorsUnstaking: baseline.ValidatorsUnstaking, DelegatesActive: baseline.DelegatesActive,
		DelegatesPaused: baseline.DelegatesPaused, DelegatesUnstaking: baseline.DelegatesUnstaking,
	}
	prevByAddr, err := validatorStatusByAddress(previous)
	if err != nil {
		return nil, err
	}
	currByAddr, err := validatorStatusByAddress(current)
	if err != nil {
		return nil, err
	}
	for addr, curr := range currByAddr {
		old, hadOld := prevByAddr[addr]
		if hadOld {
			decrementBucket(totals, old)
		}
		incrementBucket(totals, curr)
	}
	for addr, old := range prevByAddr {
		if _, stillPresent := currByAddr[addr]; !stillPresent {
			decrementBucket(totals, old) // deleted (e.g. finished unstaking) - decrement only
		}
	}
	return totals, nil
}

type validatorStatus struct {
	unstaking, paused, delegate bool
}

func validatorStatusByAddress(entries [][]byte) (map[string]validatorStatus, lib.ErrorI) {
	out := make(map[string]validatorStatus, len(entries))
	for _, entry := range entries {
		v := new(Validator)
		if err := lib.Unmarshal(entry, v); err != nil {
			return nil, lib.ErrUnmarshal(err)
		}
		out[string(v.Address)] = validatorStatus{
			unstaking: v.UnstakingHeight > 0,
			paused:    v.UnstakingHeight == 0 && v.MaxPausedHeight > 0,
			delegate:  v.Delegate,
		}
	}
	return out, nil
}

func incrementBucket(t *lib.ValidatorTotals, s validatorStatus) {
	switch {
	case s.unstaking:
		t.ValidatorsUnstaking++
		if s.delegate {
			t.DelegatesUnstaking++
		}
	case s.paused:
		t.ValidatorsPaused++
		if s.delegate {
			t.DelegatesPaused++
		}
	default:
		t.ValidatorsActive++
		if s.delegate {
			t.DelegatesActive++
		}
	}
}

func decrementBucket(t *lib.ValidatorTotals, s validatorStatus) {
	switch {
	case s.unstaking:
		t.ValidatorsUnstaking--
		if s.delegate {
			t.DelegatesUnstaking--
		}
	case s.paused:
		t.ValidatorsPaused--
		if s.delegate {
			t.DelegatesPaused--
		}
	default:
		t.ValidatorsActive--
		if s.delegate {
			t.DelegatesActive--
		}
	}
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
		BlockNonSigners:          src.BlockNonSigners,
		NonSignersDelta:          src.NonSignersDelta,
	}
}

// nilSafeBlob() normalizes a nil blob to an empty blob for helper calls.
func nilSafeBlob(blob *IndexerBlob) *IndexerBlob {
	if blob != nil {
		return blob
	}
	return &IndexerBlob{}
}
