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
// It serves from the live tip cache when available; otherwise it replays the block via
// fsm.ReplayBlock against the snapshot, discarding all writes without ever committing.
func (c *Controller) GetAccountDelta(ctx context.Context, height uint64) (delta *fsm.AccountDelta, err lib.ErrorI) {
	// no committed block pairs with state version 0 or 1
	if height < minAccountDeltaHeight {
		return nil, lib.ErrWrongBlockHeight(height, minAccountDeltaHeight)
	}
	// the block whose application produced state version `height`
	blockHeight := height - 1
	// the cache is nil for a Controller not built by New() (test literals)
	if c.accountDeltaCache != nil {
		// the cached entry is the live shared object (see account_delta_cache.go): Clone()
		// copies the slice headers so no caller can append-through into the shared tip
		// cache; the entry values stay shared -- they are immutable by contract
		if entry, ok := c.accountDeltaCache.get(blockHeight); ok {
			c.Metrics.RecordAccountDeltaTipCacheHit()
			return entry.Clone(), nil
		}
	}
	// everything below is the replay fall-through: after a restart the cache is cold, and
	// anything older than the tip window replays the full block on every request -- the
	// paired hit/miss counters are what make that visible (Metrics methods are nil-safe)
	c.Metrics.RecordAccountDeltaTipCacheMiss()
	c.log.Debugf("account delta for block %d not in tip cache, replaying", blockHeight)
	// capture the live FSM once: commitToStore reassigns c.FSM, and re-reading it would let a
	// concurrent commit slip a different object between the TimeMachine calls and the
	// `!= liveFSM` comparisons below — inverting the Discard guards and potentially
	// discarding the (former) live FSM's store transaction
	liveFSM := c.FSM
	// NEVER read the block or QC through the live store: every commit closes and swaps the
	// live store's readers un-mutexed (store.Reset()), so an RPC goroutine reading its Txn
	// races the commit path. Read through a TimeMachine snapshot instead — the same pattern
	// fsm.IndexerBlob uses ("Use the snapshot store (not the live store)"). The snapshot is
	// taken at state version `height`: block blockHeight's own commit writes the block and
	// its QC at exactly that version, so it is the earliest snapshot containing both (the
	// replay snapshot below, at version blockHeight, by definition contains neither)
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
	// LOAD-BEARING INVARIANT: this reads store.GetQCByHeight(blockHeight) from the
	// version-`height` snapshot, which sees the certificate exactly as block blockHeight's own
	// commit indexed it (controller/block.go:288,410) -- the same qc object whose
	// Results.RootDexBatch the live commit path fed to SetRootDexCache
	// (controller/block.go:274,397). The later CheckAndSetLastCertificate normalization
	// overwrite for blockHeight ("multiple valid versions can exist", controller/block.go:740)
	// is written by block blockHeight+1's commit at version height+1, which this snapshot
	// cannot see -- so the pre-normalization certificate is read deliberately, and no
	// normalization reasoning is needed: whichever valid QC version a peer committed with,
	// its Results are bound by ResultsHash, so the RootDexBatch matches what that commit
	// applied. IndexQC drops only .Block when storing (store/indexer.go), leaving .Results
	// (and .Results.RootDexBatch) intact. If commits ever stopped indexing the certificate
	// they applied with, this would silently go back to being incomplete for nested-chain
	// deltas -- the exact bug this restore was added to fix.
	if blockHeight > 1 {
		qc, qcErr := store.GetQCByHeight(blockHeight)
		if qcErr != nil {
			return nil, qcErr
		}
		// every committed block indexes its certificate in the same commit
		// (controller/block.go:288,410), and a real certificate always has a Header. A missing
		// key unmarshals into an empty, non-nil QC, so a nil Header means the certificate could
		// not be loaded — in which case whether a RootDexBatch existed is unknowable and the
		// delta may be silently incomplete. Fail loudly instead.
		if qc == nil || qc.Header == nil {
			return nil, lib.ErrEmptyQuorumCertificate()
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
	// ReplayBlock applies with skipRoot=true: this result is never committed and its
	// StateRoot is never inspected. Writes land in the snapshot store's in-memory txn and
	// die with the Discard above — nothing on this path calls Commit().
	_, applyResult, err := replayFSM.ReplayBlock(ctx, block, collector)
	if err != nil {
		return nil, err
	}
	// the collector self-poisons on an internal failure instead of aborting the write path
	// (see fsm.AccountChangeCollector) — a poisoned replay collector means the delta is
	// incomplete, so fail loudly rather than serve a silently wrong account set
	if cErr := collector.Err(); cErr != nil {
		return nil, cErr
	}
	// consensus rejects any block containing failed transactions (ApplyAndValidateBlock
	// returns ErrFailedTransactions before the block can be committed), so a committed
	// block never legitimately has them. A failure here means the replay diverged from
	// the original application, which makes the delta wrong — fail loudly rather than
	// return a silently incomplete account set.
	if len(applyResult.Failed) != 0 {
		return nil, lib.ErrFailedTransactions()
	}
	return collector.Results(), nil
}
