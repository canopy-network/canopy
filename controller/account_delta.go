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
	// the block itself is read from the LIVE store: the TimeMachine snapshot is taken at
	// version blockHeight, which by definition does not yet contain block blockHeight
	store, ok := c.FSM.Store().(lib.StoreI)
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
	replayFSM, err := c.FSM.TimeMachine(blockHeight)
	if err != nil {
		return nil, nil, nil, err
	}
	// TimeMachine returns the receiver itself when it cannot build a historical view;
	// discarding that would blow away the LIVE state machine's transaction
	if replayFSM != c.FSM {
		defer replayFSM.Discard()
	}
	// TimeMachine silently clamps a height above the tip, which would otherwise replay
	// the block against the wrong pre-state and return a plausible-looking wrong delta
	if replayFSM.Height() != blockHeight {
		return nil, nil, nil, lib.ErrWrongBlockHeight(replayFSM.Height(), blockHeight)
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
