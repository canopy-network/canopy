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
// height-1 and state version height, as one fsm.AccountDelta.
//
// `height` is a state version (same convention as fsm.IndexerBlob): version H means
// blocks 0..H-1 are applied, so the block replayed here is block H-1, against a
// TimeMachine(H-1) snapshot — exactly the pre-state a live FSM had while applying it.
// The replay is throwaway: writes are discarded, nothing is committed or cached.
func (c *Controller) GetAccountDelta(ctx context.Context, height uint64) (delta *fsm.AccountDelta, err lib.ErrorI) {
	// no committed block pairs with state version 0 or 1
	if height < minAccountDeltaHeight {
		return nil, lib.ErrWrongBlockHeight(height, minAccountDeltaHeight)
	}
	// the block whose application produced state version `height`
	blockHeight := height - 1
	// capture the live FSM once: commitToStore reassigns c.FSM, and re-reading it could
	// invert the `!= liveFSM` Discard guards below across a concurrent commit
	liveFSM := c.FSM
	// never read the block or QC through the live store — commits swap its readers
	// un-mutexed (store.Reset()). Read through a snapshot at version `height`, the
	// earliest one containing both the block and its QC
	readFSM, err := liveFSM.TimeMachine(height)
	if err != nil {
		return nil, err
	}
	// TimeMachine returns the receiver itself when it cannot build a historical view;
	// discarding that would blow away the LIVE state machine's transaction
	if readFSM != liveFSM {
		defer readFSM.Discard()
	}
	// TimeMachine silently clamps a height above the tip, which would otherwise read a
	// different height's block — and, when the receiver itself came back, fall through to
	// the live store this snapshot exists to avoid
	if readFSM.Height() != height {
		return nil, lib.ErrWrongBlockHeight(readFSM.Height(), height)
	}
	// the snapshot store serves both the block fetch here and the QC fetch below
	store, ok := readFSM.Store().(lib.StoreI)
	if !ok {
		return nil, fsm.ErrWrongStoreType()
	}
	blockResult, err := store.GetBlockByHeight(blockHeight)
	if err != nil {
		return nil, err
	}
	if blockResult == nil || blockResult.BlockHeader == nil {
		return nil, lib.ErrNilBlockHeader()
	}
	if blockResult.BlockHeader.Height != blockHeight {
		return nil, lib.ErrWrongBlockHeight(blockResult.BlockHeader.Height, blockHeight)
	}
	// GetBlockByHeight returns a *lib.BlockResult; ApplyBlock needs a *lib.Block
	block, err := blockResult.ToBlock()
	if err != nil {
		return nil, err
	}
	// snapshot the pre-block state
	replayFSM, err := liveFSM.TimeMachine(blockHeight)
	if err != nil {
		return nil, err
	}
	// TimeMachine returns the receiver itself when it cannot build a historical view;
	// discarding that would blow away the LIVE state machine's transaction
	if replayFSM != liveFSM {
		defer replayFSM.Discard()
	}
	// TimeMachine silently clamps a height above the tip, which would otherwise replay
	// the block against the wrong pre-state and return a plausible-looking wrong delta
	if replayFSM.Height() != blockHeight {
		return nil, lib.ErrWrongBlockHeight(replayFSM.Height(), blockHeight)
	}
	// restore the root DEX cache from the committed certificate, exactly as the commit path
	// does. Without it, on a nested chain HandleDexBatch reads the fresh snapshot's nil
	// cache and silently no-ops, dropping every account the DEX batch moved from the delta.
	// The pre-normalization certificate this snapshot sees is fine: Results are bound by
	// ResultsHash, so its RootDexBatch matches what the commit applied. BeginBlock skips
	// certificate handling entirely at heights <= 1, so no QC is loaded there.
	if blockHeight > 1 {
		qc, qcErr := store.GetQCByHeight(blockHeight)
		if qcErr != nil {
			return nil, qcErr
		}
		// a missing key unmarshals into an empty, non-nil QC, so a nil Header means the
		// certificate could not be loaded — whether a RootDexBatch existed is then
		// unknowable and the delta may be silently incomplete. Fail loudly instead.
		if qc == nil || qc.Header == nil {
			return nil, lib.ErrEmptyQuorumCertificate()
		}
		// nil Results/RootDexBatch is normal (root chain, or no root batch in the
		// certificate) — the live path leaves the cache nil there too
		if qc.Results != nil && qc.Results.RootDexBatch != nil {
			replayFSM.SetRootDexCache(qc.Results.RootDexBatch)
		}
	}
	// the collector's baseline lookup must read the snapshot, not the live state
	collector := fsm.NewAccountChangeCollector(replayFSM.Get)
	// ReplayBlock applies with skipRoot=true: the result is never committed and its
	// StateRoot never inspected; writes die with the Discard above
	_, applyResult, err := replayFSM.ReplayBlock(ctx, block, collector)
	if err != nil {
		return nil, err
	}
	// a committed block never has failed transactions (consensus rejects them), so a
	// failure here means the replay diverged and the delta is wrong — fail loudly
	if len(applyResult.Failed) != 0 {
		return nil, lib.ErrFailedTransactions()
	}
	return collector.Results(), nil
}
