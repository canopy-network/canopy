package fsm

import "github.com/canopy-network/canopy/lib"

// INDEXER.GO IS ONLY USED FOR CANOPY INDEXING RPC - NOT A CRITICAL PIECE OF THE STATE MACHINE

/*
	TODO
  - /v1/gov/poll — poll snapshots via cli.Poll in app/indexer/activity/poll.go.
  - /v1/gov/proposals — proposal snapshots via cli.Proposals in app/indexer/activity/proposals.go.
*/

// IndexerBlob() retrieves the protobuf blobs for a blockchain indexer
func (s *StateMachine) IndexerBlobs(height uint64) (b *IndexerBlobs, err lib.ErrorI) {
	b = &IndexerBlobs{}
	if height > 1 {
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
