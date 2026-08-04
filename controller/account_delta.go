package controller

import (
	"context"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// minAccountDeltaHeight is the lowest state version an account delta exists for.
// A delta is the difference between state@height-1 and state@height, so height must
// be at least 2 for both sides of the comparison to name a real committed state.
const minAccountDeltaHeight = 2

// GetAccountDelta returns the accounts added/changed/removed between state version
// height-1 and state version height.
//
// HEIGHT CONVENTION (deliberately matches fsm.IndexerBlob, NOT the plan's snippet):
// `height` is a STATE VERSION — the same value cmd/rpc's IndexerBlobsCached passes to
// FSM.IndexerBlob, i.e. c.FSM.Height() semantics (StateMachine.Initialize sets
// s.height = store.Version(), and reads "the previous block" as s.Height()-1). State
// version H therefore means "blocks 0..H-1 are applied", so the single block whose
// application moves state from H-1 to H is block H-1. That block is what gets replayed
// here, against a TimeMachine(H-1) snapshot — TimeMachine(v) yields the state with
// blocks 0..v-1 applied, which is exactly the pre-state of block H-1 and exactly the
// s.height a live FSM has while applying it.
//
// The tip cache is keyed by BLOCK height (commitToStore passes block.BlockHeader.Height),
// so the lookup uses height-1 as well.
//
// It serves from the live tip cache when available; otherwise it replays ApplyBlock with
// skipRoot=true against the snapshot, discarding all writes without ever committing.
func (c *Controller) GetAccountDelta(ctx context.Context, height uint64) (added, changed, removed []*fsm.AccountChangeEntry, err lib.ErrorI) {
	// no committed block pairs with state version 0 or 1
	if height < minAccountDeltaHeight {
		return nil, nil, nil, lib.ErrWrongBlockHeight(height, minAccountDeltaHeight)
	}
	// the block whose application produced state version `height`
	blockHeight := height - 1
	// the cache is nil for a Controller not built by New() (test literals)
	if c.accountDeltaCache != nil {
		// READ-ONLY: entry is the live shared cached object (see account_delta_cache.go);
		// return its slices as-is and never mutate, append to, or transform them here
		if entry, ok := c.accountDeltaCache.get(blockHeight); ok {
			return entry.added, entry.changed, entry.removed, nil
		}
	}
	// capture the live FSM once: commitToStore reassigns c.FSM, and re-reading it would let a
	// concurrent commit slip a different object between the TimeMachine call and the
	// `replayFSM != liveFSM` comparison below — inverting the Discard guard and potentially
	// discarding the (former) live FSM's store transaction
	liveFSM := c.FSM
	// the block itself is read from the LIVE store: the TimeMachine snapshot is taken at
	// version blockHeight, which by definition does not yet contain block blockHeight
	store, ok := liveFSM.Store().(lib.StoreI)
	if !ok {
		return nil, nil, nil, fsm.ErrWrongStoreType()
	}
	blockResult, err := store.GetBlockByHeight(blockHeight)
	if err != nil {
		return nil, nil, nil, err
	}
	if blockResult == nil || blockResult.BlockHeader == nil {
		return nil, nil, nil, lib.ErrNilBlockHeader()
	}
	if blockResult.BlockHeader.Height != blockHeight {
		return nil, nil, nil, lib.ErrWrongBlockHeight(blockResult.BlockHeader.Height, blockHeight)
	}
	// GetBlockByHeight returns a *lib.BlockResult; ApplyBlock needs a *lib.Block
	block, err := blockResult.ToBlock()
	if err != nil {
		return nil, nil, nil, err
	}
	// snapshot the pre-block state
	replayFSM, err := liveFSM.TimeMachine(blockHeight)
	if err != nil {
		return nil, nil, nil, err
	}
	// TimeMachine returns the receiver itself when it cannot build a historical view;
	// discarding that would blow away the LIVE state machine's transaction
	if replayFSM != liveFSM {
		defer replayFSM.Discard()
	}
	// TimeMachine silently clamps a height above the tip, which would otherwise replay
	// the block against the wrong pre-state and return a plausible-looking wrong delta
	if replayFSM.Height() != blockHeight {
		return nil, nil, nil, lib.ErrWrongBlockHeight(replayFSM.Height(), blockHeight)
	}
	// restore the root DEX cache exactly as the committing path does (controller/block.go:274
	// and :397, `if qc.Results != nil && qc.Results.RootDexBatch != nil { SetRootDexCache(...) }`).
	// Without this, on a NESTED chain HandleDexBatch's `remoteBatch = s.cache.rootDexBatch`
	// (fsm/dex.go:75-80) reads the fresh snapshot's nil cache and the function silently returns
	// at fsm/dex.go:87 — no error, no failed tx, and with skipRoot=true no state-root mismatch —
	// so every account the DEX batch would have moved is missing from the delta.
	//
	// :222's `c.getDexRootBatch(rcBuildHeight)` is deliberately NOT mirrored: that is
	// ValidateProposal's pre-consensus path, which reaches out to the root chain. This function
	// replays the COMMIT path, which uses the batch embedded in the certificate.
	//
	// BeginBlock returns before any certificate handling when s.Height() <= 1
	// (fsm/automatic.go:29-31), so the cache cannot matter for block 1 and no QC is loaded.
	//
	// LOAD-BEARING INVARIANT: this reads store.GetQCByHeight(blockHeight) directly, bypassing
	// ApplyAndValidateBlock and therefore CheckAndSetLastCertificate -- which exists precisely
	// to normalize the indexed QC for height-1 because "multiple valid versions can exist"
	// (controller/block.go:701, :715-716). That's correct here for a non-obvious reason:
	// committing block blockHeight+1 calls CheckAndSetLastCertificate(candidate) with
	// candidate.Height == blockHeight+1, which IndexQC's candidate.LastQuorumCertificate --
	// i.e. block blockHeight+1's OWN embedded copy of the QC for blockHeight -- overwriting
	// whatever was indexed for blockHeight before. So the QC this function reads for blockHeight
	// is already normalized as long as block blockHeight+1 has been committed, which is
	// guaranteed here: GetAccountDelta only replays blocks old enough to be committed already
	// (the live tip cache, checked above, serves anything more recent). And
	// LoadCertificateHashesOnly nullifies only .Block, leaving .Results (and .Results.RootDexBatch)
	// intact -- EqualPayloads compares ResultsHash, not the full Results struct, so the
	// normalization check never forces Results to nil. That's why the RootDexBatch restore below
	// reads the right batch. If block headers ever stopped carrying Results, this would silently
	// go back to being incomplete for nested-chain deltas -- the exact bug this restore was added
	// to fix.
	if blockHeight > 1 {
		qc, qcErr := store.GetQCByHeight(blockHeight)
		if qcErr != nil {
			return nil, nil, nil, qcErr
		}
		// every committed block indexes its certificate in the same commit
		// (controller/block.go:288,410), and a real certificate always has a Header. A missing
		// key unmarshals into an empty, non-nil QC, so a nil Header means the certificate could
		// not be loaded — in which case whether a RootDexBatch existed is unknowable and the
		// delta may be silently incomplete. Fail loudly instead.
		if qc == nil || qc.Header == nil {
			return nil, nil, nil, lib.ErrEmptyQuorumCertificate()
		}
		// a nil Results or nil RootDexBatch is NORMAL (root chain, and any block whose
		// certificate carries no root batch). The live path leaves the cache nil there too —
		// c.FSM.Reset() clears cache.rootDexBatch — so a fresh snapshot already matches and
		// erroring here would be a false positive.
		if qc.Results != nil && qc.Results.RootDexBatch != nil {
			replayFSM.SetRootDexCache(qc.Results.RootDexBatch)
		}
	}
	// the collector's baseline lookup must read the snapshot, not the live state
	collector := fsm.NewAccountChangeCollector(replayFSM.Get)
	// skipRoot=true: this result is never committed and its StateRoot is never inspected.
	// Writes land in the snapshot store's in-memory txn and die with the Discard above —
	// nothing on this path calls Commit().
	_, applyResult, err := replayFSM.ApplyBlock(ctx, block, false, collector, true)
	if err != nil {
		return nil, nil, nil, err
	}
	// consensus rejects any block containing failed transactions (ApplyAndValidateBlock
	// returns ErrFailedTransactions before the block can be committed), so a committed
	// block never legitimately has them. A failure here means the replay diverged from
	// the original application, which makes the delta wrong — fail loudly rather than
	// return a silently incomplete account set.
	if len(applyResult.Failed) != 0 {
		return nil, nil, nil, lib.ErrFailedTransactions()
	}
	added, changed, removed = collector.Results()
	return added, changed, removed, nil
}
