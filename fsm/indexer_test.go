package fsm

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestDeltaIndexerBlobs_ChangedAddedRemoved(t *testing.T) {
	prevAcc1 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{1}, 20), Amount: 10})
	prevAcc2 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{2}, 20), Amount: 20})
	currAcc2 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{2}, 20), Amount: 21})
	currAcc3 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{3}, 20), Amount: 30})

	prevPool1 := mustMarshalProto(t, &Pool{Id: 1, Amount: 100})
	prevPool2 := mustMarshalProto(t, &Pool{Id: 2, Amount: 200})
	currPool2 := mustMarshalProto(t, &Pool{Id: 2, Amount: 201})
	currPool3 := mustMarshalProto(t, &Pool{Id: 3, Amount: 300})

	prevVal1 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{4}, 20), StakedAmount: 400})
	prevVal2 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{5}, 20), StakedAmount: 500})
	currVal1 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{4}, 20), StakedAmount: 400})
	currVal3 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{6}, 20), StakedAmount: 600})

	prev := &IndexerBlob{
		Block:      mustMarshalProto(t, &lib.BlockResult{}),
		Accounts:   [][]byte{prevAcc1, prevAcc2},
		Pools:      [][]byte{prevPool1, prevPool2},
		Validators: [][]byte{prevVal1, prevVal2},
	}
	curr := &IndexerBlob{
		Block:      mustMarshalProto(t, &lib.BlockResult{}),
		Accounts:   [][]byte{currAcc2, currAcc3},
		Pools:      [][]byte{currPool2, currPool3},
		Validators: [][]byte{currVal1, currVal3},
	}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: curr, Previous: prev})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, currAcc2, currAcc3)
	requireEntriesAsSet(t, delta.Previous.Accounts, prevAcc1, prevAcc2)
	requireEntriesAsSet(t, delta.Current.Pools, currPool2, currPool3)
	requireEntriesAsSet(t, delta.Previous.Pools, prevPool1, prevPool2)
	requireEntriesAsSet(t, delta.Current.Validators, currVal3)
	requireEntriesAsSet(t, delta.Previous.Validators, prevVal2)
	require.True(t, delta.Current.ValidatorsDelta)
	require.True(t, delta.Previous.ValidatorsDelta)
}

func TestDeltaIndexerBlobs_UnchangedEntitiesBecomeEmpty(t *testing.T) {
	acc := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{7}, 20), Amount: 1})
	pool := mustMarshalProto(t, &Pool{Id: 7, Amount: 7})
	val := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{8}, 20), StakedAmount: 8})
	block := mustMarshalProto(t, &lib.BlockResult{})
	current := &IndexerBlob{Block: block, Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}}
	previous := &IndexerBlob{Block: block, Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: current, Previous: previous})
	require.NoError(t, err)
	require.Empty(t, delta.Current.Accounts)
	require.Empty(t, delta.Previous.Accounts)
	require.Empty(t, delta.Current.Pools)
	require.Empty(t, delta.Previous.Pools)
	require.Empty(t, delta.Current.Validators)
	require.Empty(t, delta.Previous.Validators)
}

func TestDeltaIndexerBlobs_ForceIncludeRewardSlashAccounts(t *testing.T) {
	addr := bytes.Repeat([]byte{9}, 20)
	accPrev := mustMarshalProto(t, &Account{Address: addr, Amount: 100})
	accCurr := mustMarshalProto(t, &Account{Address: addr, Amount: 100})

	block := &lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 10},
		Events: []*lib.Event{{
			EventType: string(lib.EventTypeReward),
			Address:   addr,
		}},
	}
	blockBz := mustMarshalProto(t, block)
	emptyBlockBz := mustMarshalProto(t, &lib.BlockResult{})

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current:  &IndexerBlob{Block: blockBz, Accounts: [][]byte{accCurr}},
		Previous: &IndexerBlob{Block: emptyBlockBz, Accounts: [][]byte{accPrev}},
	})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, accCurr)
	requireEntriesAsSet(t, delta.Previous.Accounts, accPrev)
}

func TestDeltaIndexerBlobs_ForceIncludeValidatorByRewardOutput(t *testing.T) {
	operator := bytes.Repeat([]byte{10}, 20)
	output := bytes.Repeat([]byte{11}, 20)

	valPrev := mustMarshalProto(t, &Validator{Address: operator, Output: output, StakedAmount: 1000, Delegate: true})
	valCurr := mustMarshalProto(t, &Validator{Address: operator, Output: output, StakedAmount: 1000, Delegate: true})

	block := &lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 12},
		Events: []*lib.Event{{
			EventType: string(lib.EventTypeReward),
			Address:   output,
		}},
	}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current:  &IndexerBlob{Block: mustMarshalProto(t, block), Validators: [][]byte{valCurr}},
		Previous: &IndexerBlob{Block: mustMarshalProto(t, &lib.BlockResult{}), Validators: [][]byte{valPrev}},
	})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Validators, valCurr)
	requireEntriesAsSet(t, delta.Previous.Validators, valPrev)
}

func TestDeltaIndexerBlobs_NoPreviousKeepsCurrent(t *testing.T) {
	acc := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{10}, 20), Amount: 42})
	pool := mustMarshalProto(t, &Pool{Id: 42, Amount: 42})
	val := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{11}, 20), StakedAmount: 11})

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current: &IndexerBlob{Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}},
	})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, acc)
	requireEntriesAsSet(t, delta.Current.Pools, pool)
	requireEntriesAsSet(t, delta.Current.Validators, val)
	require.True(t, delta.Current.ValidatorsDelta)
	require.Nil(t, delta.Previous)
}

func TestDeltaIndexerBlobs_KeepsBlockNonSigners(t *testing.T) {
	currentNonSigners := [][]byte{bytes.Repeat([]byte{12}, 20)}
	previousNonSigners := [][]byte{bytes.Repeat([]byte{13}, 20)}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current:  &IndexerBlob{BlockNonSigners: currentNonSigners},
		Previous: &IndexerBlob{BlockNonSigners: previousNonSigners},
	})
	require.NoError(t, err)
	require.Equal(t, currentNonSigners, delta.Current.BlockNonSigners)
	require.Equal(t, previousNonSigners, delta.Previous.BlockNonSigners)
}

func TestMergeChangedBlobKeys_MatchesMapBasedDiff(t *testing.T) {
	// sorted ascending by key, as accountEntries/validatorEntries produce from a Pebble scan
	current := []blobEntry{
		{key: "a", bz: []byte("a1")}, // changed
		{key: "b", bz: []byte("b1")}, // unchanged
		{key: "d", bz: []byte("d1")}, // added
	}
	previous := []blobEntry{
		{key: "a", bz: []byte("a0")},
		{key: "b", bz: []byte("b1")},
		{key: "c", bz: []byte("c1")}, // removed
	}

	currentMap := map[string][]byte{"a": []byte("a1"), "b": []byte("b1"), "d": []byte("d1")}
	previousMap := map[string][]byte{"a": []byte("a0"), "b": []byte("b1"), "c": []byte("c1")}

	gotCurrentChanged, gotPreviousChanged := mergeChangedBlobKeys(current, previous)
	wantCurrentChanged, wantPreviousChanged := changedBlobKeys(currentMap, previousMap)

	require.Equal(t, wantCurrentChanged, gotCurrentChanged)
	require.Equal(t, wantPreviousChanged, gotPreviousChanged)
	require.Equal(t, map[string]struct{}{"a": {}, "d": {}}, gotCurrentChanged)
	require.Equal(t, map[string]struct{}{"a": {}, "c": {}}, gotPreviousChanged)
}

func TestMergeChangedBlobKeys_EmptyPrevious(t *testing.T) {
	current := []blobEntry{{key: "a", bz: []byte("a1")}, {key: "b", bz: []byte("b1")}}
	gotCurrentChanged, gotPreviousChanged := mergeChangedBlobKeys(current, nil)
	require.Equal(t, map[string]struct{}{"a": {}, "b": {}}, gotCurrentChanged)
	require.Empty(t, gotPreviousChanged)
}

// TestAccountDelta_MatchesOldFullScanAndDiff is the differential regression test for the
// account-delta fast path: it runs the OLD path (full AccountPrefix scan of both heights,
// diffed by DeltaIndexerBlobs) and the NEW path (single-block ApplyBlock replay with an
// AccountChangeCollector, assembled the way cmd/rpc's overrideAccountDelta assembles it)
// against the SAME committed chain, and requires byte-for-byte, order-for-order equality of
// both delta sides. Neither side's expectation is hand-seeded — both are computed from the
// chain, so a divergence in classification, in value selection (FinalValue vs PrevValue),
// in the reward/slash force-include, or in ordering fails the test.
//
// LIMITATION (deliberate, documented): this chain runs on the ROOT chain (Config.ChainId ==
// lib.CanopyChainId), so it does NOT exercise the nested-chain root-DEX-batch path. On a
// nested chain, fsm/dex.go's HandleDexBatch early-returns when s.cache.rootDexBatch is nil,
// which a fresh TimeMachine snapshot always is; controller.GetAccountDelta compensates by
// calling SetRootDexCache from the committed certificate before replaying. That restoration
// lives in the controller package and cannot be reached from an fsm-package test, so the
// equivalence proven here covers root-chain blocks only.
func TestAccountDelta_MatchesOldFullScanAndDiff(t *testing.T) {
	sm, targetHeight := newTestAccountDeltaChain(t)
	ctx := context.Background()

	// OLD PATH: full account scan at both heights, diffed by DeltaIndexerBlobs
	oldCurrent, err := sm.IndexerBlob(ctx, targetHeight, false)
	require.NoError(t, err)
	oldPrevious, err := sm.IndexerBlob(ctx, targetHeight-1, false)
	require.NoError(t, err)
	oldDelta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: oldCurrent, Previous: oldPrevious})
	require.NoError(t, err)

	// NEW PATH: replay the single block whose application moved state from targetHeight-1 to
	// targetHeight, capturing touched accounts. Mirrors controller.GetAccountDelta's replay
	// branch (block read from the live store, pre-state from TimeMachine(blockHeight),
	// collector baselined on the snapshot, skipRoot=true, never committed).
	blockHeight := targetHeight - 1
	blockResult, err := sm.store.(lib.StoreI).GetBlockByHeight(blockHeight)
	require.NoError(t, err)
	require.NotNil(t, blockResult)
	block, err := blockResult.ToBlock()
	require.NoError(t, err)
	replaySM, err := sm.TimeMachine(blockHeight)
	require.NoError(t, err)
	require.NotSame(t, sm, replaySM, "TimeMachine must produce a real snapshot, not the live FSM")
	defer replaySM.Discard()
	require.Equal(t, blockHeight, replaySM.Height())
	collector := NewAccountChangeCollector(replaySM.Get)
	_, applyResult, err := replaySM.ApplyBlock(ctx, block, false, collector, true)
	require.NoError(t, err)
	require.Empty(t, applyResult.Failed)
	added, changed, removed := collector.Results()

	// sanity-check the fixture actually produced the scenarios it claims to, so a chain that
	// silently stopped exercising them can't turn this into a vacuous empty-vs-empty compare
	require.NotEmpty(t, added, "fixture must produce at least one brand-new account")
	require.NotEmpty(t, changed, "fixture must produce at least one changed account")
	require.NotEmpty(t, removed, "fixture must produce at least one removed (zeroed) account")

	// ...and that the force-include path has real work to do: the block must emit a
	// reward event naming an address the collector never saw written, so both assemblies
	// have to read that account back rather than derive it from the replay
	forced, err := RewardSlashAccountKeys(oldCurrent.Block)
	require.NoError(t, err)
	require.NotEmpty(t, forced, "fixture must emit a reward/slash event")
	touched := make(map[string]struct{})
	for _, e := range append(append(append([]*AccountChangeEntry{}, added...), changed...), removed...) {
		touched[string(e.Address)] = struct{}{}
	}
	for address := range forced {
		require.NotContains(t, touched, address,
			"the forced address must be untouched by the block, or the read-back never runs")
	}

	newCurrent := testAccountSide(added, changed, true)
	require.NoError(t, testForceIncludeAccounts(sm, newCurrent, oldCurrent.Block, targetHeight))
	newPrevious := testAccountSide(changed, removed, false)
	require.NoError(t, testForceIncludeAccounts(sm, newPrevious, oldCurrent.Block, targetHeight-1))

	// exact ordered equality: both paths are ascending-by-address (the old side comes from an
	// ascending Pebble prefix scan, the new one from sortedAccountEntries), so ElementsMatch
	// would silently tolerate an ordering regression that changes the cached response bytes.
	require.Equal(t, oldDelta.Current.Accounts, testSortedAccountEntries(newCurrent))
	require.Equal(t, oldDelta.Previous.Accounts, testSortedAccountEntries(newPrevious))
}

// testAccountSide / testForceIncludeAccounts / testSortedAccountEntries mirror
// accountSide(), forceIncludeAccounts() and sortedAccountEntries() in cmd/rpc/query.go. They
// are duplicated rather than imported because fsm cannot import cmd/rpc (import cycle); keep
// them in step with that file or this test stops proving what the RPC layer actually serves.
func testAccountSide(a, b []*AccountChangeEntry, useFinal bool) map[string][]byte {
	side := make(map[string][]byte, len(a)+len(b))
	collect := func(entries []*AccountChangeEntry) {
		for _, e := range entries {
			if e == nil {
				continue
			}
			value := e.FinalValue
			if !useFinal {
				value = e.PrevValue
			}
			if value == nil {
				continue
			}
			side[string(e.Address)] = append([]byte(nil), value...)
		}
	}
	collect(a)
	collect(b)
	return side
}

func testForceIncludeAccounts(sm *StateMachine, side map[string][]byte, currentBlockBz []byte, version uint64) lib.ErrorI {
	forced, err := RewardSlashAccountKeys(currentBlockBz)
	if err != nil {
		return err
	}
	missing := make(map[string]struct{}, len(forced))
	for address := range forced {
		if _, ok := side[address]; !ok {
			missing[address] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	entries, err := sm.AccountEntriesAtVersion(version, missing)
	if err != nil {
		return err
	}
	for address, bz := range entries {
		side[address] = bz
	}
	return nil
}

func testSortedAccountEntries(side map[string][]byte) [][]byte {
	addresses := make([]string, 0, len(side))
	for address := range side {
		addresses = append(addresses, address)
	}
	slices.Sort(addresses)
	out := make([][]byte, 0, len(addresses))
	for _, address := range addresses {
		out = append(out, side[address])
	}
	return out
}

// newTestAccountDeltaChain builds a REAL committed chain by running ApplyBlock + Commit for
// each block, starting from newTestApplyBlockFixture (fsm/state_test.go), and returns the
// state machine plus the target state version to diff.
//
// Height convention: state version V means blocks 0..V-1 are applied, so the delta between
// state@V-1 and state@V is produced by block V-1, and IndexerBlob(V) pairs state@V with
// block V-1. newTestApplyBlockFixture leaves sm.height=3 but store version 2 (it commits
// hand-written state rather than applying a block), so the helper commits once more to align
// them before driving real blocks.
//
// The chain applies blocks 3, 4 and 5. Block 5 -- the one the returned target height (6)
// diffs -- is the interesting one; it exercises:
//   - a brand-new account: first receipt at an address that has never held funds (added)
//   - a balance change to an existing account (changed), for both a sender and a recipient
//   - an account touched but net-unchanged: a zero-fee self-send, which writes the account
//     twice and lands on its original bytes (must be dropped from BOTH sides)
//   - an account zeroed out: it sends its entire balance, so SetAccount's Amount==0 &&
//     Nonce==0 branch deletes the key (removed)
//   - a reward event naming an address that block 5 never writes (the certificate pays
//     validator #3, which is non-compounding, so the tokens land on its OUTPUT account
//     instead) — the reward/slash force-include has to read that account back on both sides
func newTestAccountDeltaChain(t *testing.T) (*StateMachine, uint64) {
	t.Helper()
	fixture, _ := newTestApplyBlockFixture(t)
	sm := &fixture
	st := sm.store.(lib.StoreI)
	// newTestApplyBlockFixture hand-sets sm.NetworkID rather than deriving it from Config, but
	// TimeMachine builds its snapshot FSM through newStateMachine, which sets
	// NetworkID = Config.P2PConfig.NetworkID. Left as-is, every transaction in the replayed
	// block would fail the replay's network-id check -- a fixture artifact, not a production
	// one (a live node derives both from the same config). Align them here -- on the config
	// side, since a network id of 0 is rejected outright as "nil network id".
	sm.Config.P2PConfig.NetworkID = uint64(sm.NetworkID)
	// align the store version with sm.height (see the height convention note above)
	_, e := st.Commit()
	require.NoError(t, e)
	require.Equal(t, sm.height, st.Version())

	// send fee of zero, so the net-unchanged self-send below really is net-unchanged
	require.NoError(t, sm.UpdateParam("fee", ParamSendFee, &lib.UInt64Wrapper{Value: 0}))
	// funder pays for everything downstream
	require.NoError(t, sm.AccountAdd(newTestAddress(t), 10_000_000))
	// block 3: no transactions -- it exists so block 4's BeginBlock has a real committed
	// predecessor block and certificate to load
	testApplyAndCommitBlock(t, sm, nil)

	// block 4: fund the accounts block 5 will change / zero / self-send
	testApplyAndCommitBlock(t, sm, [][]byte{
		testSendTxBytes(t, sm, 0, newTestAddress(t, 4), 5_000), // net-unchanged self-sender
		testSendTxBytes(t, sm, 0, newTestAddress(t, 5), 3_000), // zeroed out in block 5
		testSendTxBytes(t, sm, 0, newTestAddress(t, 6), 2_000), // changed in block 5
		testSendTxBytes(t, sm, 0, newTestAddress(t, 3), 4_000), // reward-event recipient; untouched by block 5
	})

	// block 5: the measured block
	brandNew := crypto.NewAddressFromBytes(bytes.Repeat([]byte{0xAB}, crypto.AddressSize))
	testApplyAndCommitBlock(t, sm, [][]byte{
		testSendTxBytes(t, sm, 0, brandNew, 1_500),             // brandNew added; kg0 changed
		testSendTxBytes(t, sm, 6, newTestAddress(t, 7), 1_000), // kg6 changed; kg7 added (never funded)
		testSendTxBytes(t, sm, 4, newTestAddress(t, 4), 5_000), // kg4 self-send: touched, net-unchanged
		testSendTxBytes(t, sm, 5, newTestAddress(t, 6), 3_000), // kg5 sends its whole balance -> removed
	})

	// sm.height is now 6 and the store holds versions up to 6 with blocks 2..5 indexed, so
	// both IndexerBlob(6) (state@6 + block 5) and IndexerBlob(5) (state@5 + block 4) resolve
	return sm, sm.height
}

// TestAccountDelta_PoolAndValidatorWritesDoNotLeakIntoAccountCollector proves that when a
// REAL applied block writes POOL and VALIDATOR entries alongside ACCOUNT entries, the
// AccountChangeCollector captures only the account addresses.
// TestStateMachine_SetDoesNotHookNonAccountKeys (fsm/state_test.go) already proves prefix
// isolation at the Set() call level with a single hand-written write; this test's distinct
// value is the integration level -- pool and validator writes here happen as a side effect of
// the real BeginBlock/EndBlock committee-reward machinery (ApplyBlock), not a direct Set call.
func TestAccountDelta_PoolAndValidatorWritesDoNotLeakIntoAccountCollector(t *testing.T) {
	sm, targetHeight, validatorAddress := newTestPoolAndValidatorTouchingChain(t)
	ctx := context.Background()

	blockHeight := targetHeight - 1
	blockResult, err := sm.store.(lib.StoreI).GetBlockByHeight(blockHeight)
	require.NoError(t, err)
	require.NotNil(t, blockResult)
	block, err := blockResult.ToBlock()
	require.NoError(t, err)
	replaySM, err := sm.TimeMachine(blockHeight)
	require.NoError(t, err)
	defer replaySM.Discard()

	// snapshot pre-state DAO pool balance and validator #3's stake from the SAME snapshot the
	// replay below runs against, before ApplyBlock mutates it
	prePool, err := replaySM.GetPool(lib.DAOPoolID)
	require.NoError(t, err)
	preValidator, err := replaySM.GetValidator(crypto.NewAddressFromBytes(validatorAddress))
	require.NoError(t, err)

	collector := NewAccountChangeCollector(replaySM.Get)
	_, applyResult, err := replaySM.ApplyBlock(ctx, block, false, collector, true)
	require.NoError(t, err)
	require.Empty(t, applyResult.Failed)

	// verify -- don't assume -- that the block genuinely wrote both a pool and a validator
	// entry. The DAO pool is minted into every block unconditionally (FundCommitteeRewardPools,
	// never debited back down), so a strict increase proves a real SetPool call happened, not
	// just a Set(sameValue) no-op. Validator #3 was made compounding by the fixture below, so
	// its committee reward in this block updates StakedAmount via UpdateValidatorStake ->
	// SetValidator instead of the default non-compounding AccountAdd-to-output path -- a strict
	// increase proves that write happened too. If the fixture regressed to stop touching either,
	// this would fail instead of silently asserting nothing.
	postPool, err := replaySM.GetPool(lib.DAOPoolID)
	require.NoError(t, err)
	require.Greater(t, postPool.Amount, prePool.Amount,
		"fixture must actually mint into the DAO pool during the measured block")
	postValidator, err := replaySM.GetValidator(crypto.NewAddressFromBytes(validatorAddress))
	require.NoError(t, err)
	require.Greater(t, postValidator.StakedAmount, preValidator.StakedAmount,
		"fixture must actually write validator #3's stake (compounding reward) in the measured block")

	added, changed, removed := collector.Results()
	all := append(append(append([]*AccountChangeEntry{}, added...), changed...), removed...)
	require.NotEmpty(t, all, "fixture must also touch at least one account, or this test can't distinguish account entries from a leak")

	for _, e := range all {
		// weak alone (a validator address is also crypto.AddressSize bytes), but catches a
		// pool-prefix leak: pool keys carry no address-shaped payload at all, so a
		// bytes.HasPrefix(k, PoolPrefix()) regression would produce garbage-length
		// "addresses" here rather than passing silently
		require.Len(t, e.Address, crypto.AddressSize, "captured entries must be account-sized addresses")

		// catches a validator-prefix leak specifically, which the length check above cannot:
		// validator #3's address is proven (above) to have been written in this very block, so
		// if the hook regressed to also match bytes.HasPrefix(k, ValidatorPrefix()), that
		// address would show up here -- assert it never does
		require.NotEqual(t, validatorAddress, e.Address, "validator address must not leak into the account collector")

		// positive check: every captured address must round-trip to the real KeyForAccount
		// entry, proving it genuinely is an account key rather than merely address-shaped bytes
		// living under some other prefix
		key := KeyForAccount(crypto.NewAddressFromBytes(e.Address))
		bz, gErr := replaySM.Get(key)
		require.NoError(t, gErr)
		if e.FinalValue != nil {
			require.Equal(t, e.FinalValue, bz, "captured added/changed entry must match the live KeyForAccount value")
		} else {
			require.Empty(t, bz, "captured removed entry must correspond to an actually-deleted account key")
		}
	}
}

// newTestPoolAndValidatorTouchingChain builds a REAL committed chain (same base pattern as
// newTestAccountDeltaChain: newTestApplyBlockFixture, then ApplyBlock+IndexQC+IndexBlock+Commit
// per block via testApplyAndCommitBlock) whose measured block (the last one applied) writes an
// ACCOUNT (a send tx), a POOL (the automatic committee-reward mint that runs unconditionally
// every block in BeginBlock, see FundCommitteeRewardPools) and a VALIDATOR (a *compounding*
// committee-reward payout, which routes through UpdateValidatorStake -> SetValidator instead of
// the default non-compounding AccountAdd-to-output path -- see DistributeCommitteeReward).
//
// Validator #3 is the certificate's reward recipient (see testAccountDeltaQC) and is
// non-compounding by default in newTestApplyBlockFixture, so it is flipped to Compound=true
// here directly via SetValidator. That flip happens BEFORE block 4 is applied/committed so it
// lands in the state@4 snapshot that block 5's replay (TimeMachine(4) + ApplyBlock) reads --
// flipping it after block 4's commit would either land the write in block 4 itself or be lost
// (an uncommitted edit merged into block 5's own commit isn't visible to a version-4 snapshot).
func newTestPoolAndValidatorTouchingChain(t *testing.T) (sm *StateMachine, targetHeight uint64, validatorAddress []byte) {
	t.Helper()
	fixture, _ := newTestApplyBlockFixture(t)
	sm = &fixture
	st := sm.store.(lib.StoreI)
	sm.Config.P2PConfig.NetworkID = uint64(sm.NetworkID)
	_, e := st.Commit()
	require.NoError(t, e)
	require.Equal(t, sm.height, st.Version())

	require.NoError(t, sm.UpdateParam("fee", ParamSendFee, &lib.UInt64Wrapper{Value: 0}))
	require.NoError(t, sm.AccountAdd(newTestAddress(t), 10_000_000))
	// block 3: no transactions -- exists so block 4's BeginBlock has a real committed
	// predecessor block and certificate to load
	testApplyAndCommitBlock(t, sm, nil)

	validator3, err := sm.GetValidator(newTestAddress(t, 3))
	require.NoError(t, err)
	require.False(t, validator3.Compound, "fixture assumption: validator #3 starts non-compounding")
	validator3.Compound = true
	require.NoError(t, sm.SetValidator(validator3))

	// block 4: no new transactions of its own -- indexes the certificate (paying validator #3)
	// that block 5's BeginBlock consumes
	testApplyAndCommitBlock(t, sm, nil)

	// block 5: the measured block. A send tx touches accounts; BeginBlock's automatic DAO pool
	// mint and validator #3's now-compounding committee reward (from block 4's certificate)
	// touch a pool and a validator in the same pass.
	testApplyAndCommitBlock(t, sm, [][]byte{
		testSendTxBytes(t, sm, 0, newTestAddress(t, 7), 1_000),
	})

	return sm, sm.height, newTestAddressBytes(t, 3)
}

// testApplyAndCommitBlock applies a block at sm.height with the given transactions, indexes
// the resulting BlockResult and a signed certificate for that height, commits the store, and
// advances the state machine -- the same order controller.CommitCertificate uses (apply,
// IndexQC, IndexBlock, Commit, re-init FSM at the new version).
func testApplyAndCommitBlock(t *testing.T, sm *StateMachine, txs [][]byte) {
	t.Helper()
	st := sm.store.(lib.StoreI)
	height := sm.height
	block := &lib.Block{
		BlockHeader: &lib.BlockHeader{
			Height:          height,
			Time:            uint64(time.Date(2024, 02, 01, 0, 0, 0, 0, time.UTC).UnixMicro()) + height*1_000_000,
			ProposerAddress: newTestAddressBytes(t),
		},
		Transactions: txs,
	}
	header, result, err := sm.ApplyBlock(context.Background(), block, false, nil, false)
	require.NoError(t, err)
	for _, f := range result.Failed {
		t.Logf("FAILED TX at height %d: %s", height, f.Error.Error())
	}
	require.Empty(t, result.Failed, "block at height %d had failed transactions", height)
	require.NoError(t, st.IndexQC(testAccountDeltaQC(t, sm, height)))
	require.NoError(t, st.IndexBlock(&lib.BlockResult{
		BlockHeader:  header,
		Transactions: result.Results,
		Events:       result.Events,
	}))
	_, e := st.Commit()
	require.NoError(t, e)
	sm.height++
	sm.Reset()
	require.Equal(t, sm.height, st.Version())
}

// testAccountDeltaQC builds a certificate for `height` signed by 3 of the fixture's 4
// validators -- mirroring the QC newTestApplyBlockFixture indexes for height 2, but with the
// chain ids it omits (see below). BeginBlock loads the previous height's certificate, so
// every applied block needs one indexed for its own height.
func testAccountDeltaQC(t *testing.T, sm *StateMachine, height uint64) *lib.QuorumCertificate {
	t.Helper()
	qc := &lib.QuorumCertificate{
		// ChainId is what keys the committee data HandleCertificateResults updates and
		// DistributeCommitteeRewards later pays out from -- without it the rewards land on
		// committee 0, whose pool is never subsidized, and no reward event is ever emitted
		Header: &lib.View{Height: height, ChainId: lib.CanopyChainId},
		// PaymentPercents.ChainId must be set: CommitteeData.Combine drops every stub whose
		// ChainId doesn't match, so an unset one silently yields no reward and no event.
		// The recipient is validator #3, which takes part in no transaction below and is
		// non-compounding -- so the reward is written to the validator's OUTPUT account (key
		// group 0) while the reward EVENT names validator #3. That is exactly the
		// force-include case: an address the collector never sees written, which both delta
		// sides must nonetheless carry.
		Results: &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{
				Address: newTestAddressBytes(t, 3),
				Percent: 100,
				ChainId: lib.CanopyChainId,
			}},
		}},
	}
	committee, err := sm.GetCommitteeMembers(lib.CanopyChainId)
	require.NoError(t, err)
	mk := committee.MultiKey.Copy()
	for i := 0; i < 3; i++ {
		privateKey := newTestKeyGroup(t, i).PrivateKey
		for j, pubKey := range mk.PublicKeys() {
			if privateKey.PublicKey().Equals(pubKey) {
				require.NoError(t, mk.AddSigner(privateKey.Sign(qc.SignBytes()), j))
			}
		}
	}
	aggSig, e := mk.AggregateSignatures()
	require.NoError(t, e)
	qc.Signature = &lib.AggregateSignature{Signature: aggSig, Bitmap: mk.Bitmap()}
	return qc
}

// testSendTxBytes builds a signed send transaction from test key group `fromKeyGroup`, valid
// at the state machine's current height, with a zero fee (see newTestAccountDeltaChain).
func testSendTxBytes(t *testing.T, sm *StateMachine, fromKeyGroup int, to crypto.AddressI, amount uint64) []byte {
	t.Helper()
	kg := newTestKeyGroup(t, fromKeyGroup)
	tx, err := NewSendTransaction(kg.PrivateKey, to, amount, uint64(sm.NetworkID), sm.Config.ChainId, 0, sm.height, "")
	require.NoError(t, err)
	bz, err := lib.Marshal(tx)
	require.NoError(t, err)
	return bz
}

func mustMarshalProto(t *testing.T, message any) []byte {
	t.Helper()
	bz, err := lib.Marshal(message)
	require.NoError(t, err)
	return bz
}

func requireEntriesAsSet(t *testing.T, got [][]byte, expected ...[]byte) {
	t.Helper()
	gotSet := make(map[string]int, len(got))
	for _, entry := range got {
		gotSet[string(entry)]++
	}
	expSet := make(map[string]int, len(expected))
	for _, entry := range expected {
		expSet[string(entry)]++
	}
	require.Equal(t, expSet, gotSet)
}

// TestIndexerBlob_SkipAccountsLeavesAccountsNil and TestIndexerBlob_NoSkipStillScansAccounts
// use newTestApplyBlockFixture (fsm/state_test.go) - the real helper used elsewhere in this
// package to build a StateMachine with a committed block (height 2, indexed via IndexBlock)
// and a funded account (via AccountAdd), then sm.height set to 3 and the store committed.
// That leaves the state machine ready for IndexerBlob(ctx, 3, ...) without needing to
// actually call ApplyBlock: the fixture's committed state at height 3 already pairs with
// the indexed block at height 2 (blockHeight = height - 1), and its funded account guarantees
// the accounts scan (when not skipped) has at least one real entry to assert on.
func TestIndexerBlob_SkipAccountsLeavesAccountsNil(t *testing.T) {
	sm, _ := newTestApplyBlockFixture(t)
	blob, err := sm.IndexerBlob(context.Background(), 3, true)
	require.NoError(t, err)
	require.Nil(t, blob.Accounts)
}

func TestIndexerBlob_NoSkipStillScansAccounts(t *testing.T) {
	sm, _ := newTestApplyBlockFixture(t)
	blob, err := sm.IndexerBlob(context.Background(), 3, false)
	require.NoError(t, err)
	// the fixture funds an account via AccountAdd before committing, so the full scan
	// (skipAccounts=false) must find and return it - a meaningful assertion rather than
	// a vacuous non-nil check on the blob itself.
	require.NotEmpty(t, blob.Accounts, "full scan with skipAccounts=false must still populate Accounts")
}

// TestDeltaIndexerBlobs_ZeroValuePoolAndAccount is a regression for the
// proto3-zero-value bug that permanently 400s /v1/query/indexer-blobs when
// chain state contains a Pool{Id:0} or Account{Address:nil}. Because proto3
// omits zero-valued scalar fields from the wire, marshalling Pool{Id:0,Amount:5000}
// produces bytes with only field #2 (Amount). Pre-fix, poolEntryKey called
// codec.GetRawProtoField(entry, 1) and returned "field number 1 not found",
// which bubbled up as lib.ErrUnmarshal and HTTP 400 on every affected height.
// Post-fix, the missing field is treated as an empty (zero-value) key.
func TestDeltaIndexerBlobs_ZeroValuePoolAndAccount(t *testing.T) {
	zeroPool := mustMarshalProto(t, &Pool{Id: 0, Amount: 5000})
	normalPool := mustMarshalProto(t, &Pool{Id: 32768, Amount: 25_000_000_000_000})
	zeroAcc := mustMarshalProto(t, &Account{Address: nil, Amount: 1})
	normalAcc := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{7}, 20), Amount: 42})

	curr := &IndexerBlob{
		Block:    mustMarshalProto(t, &lib.BlockResult{}),
		Accounts: [][]byte{zeroAcc, normalAcc},
		Pools:    [][]byte{zeroPool, normalPool},
	}
	prev := &IndexerBlob{
		Block:    mustMarshalProto(t, &lib.BlockResult{}),
		Accounts: [][]byte{zeroAcc, normalAcc},
		Pools:    [][]byte{zeroPool, normalPool},
	}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: curr, Previous: prev})
	require.NoError(t, err, "zero-value Pool.Id / Account.Address must not error")
	require.NotNil(t, delta)
}
