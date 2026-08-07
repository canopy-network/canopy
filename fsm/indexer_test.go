package fsm

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
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

func TestResolveValidatorTotals_UsesPersistedBaselineWhenAvailable(t *testing.T) {
	sm := newTestStateMachine(t)
	st := sm.store.(lib.StoreI)

	require.NoError(t, st.SetValidatorTotals(4, &lib.ValidatorTotals{ValidatorsActive: 7}))

	// current/previous entries are irrelevant here since no validator changed - the fallback
	// full scan must NOT run when height-1's baseline is available; if it did, it would see
	// zero validators in this empty test store and totals would come back 0, not 7.
	totals, err := sm.resolveValidatorTotals(st, 5, nil, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(7), totals.ValidatorsActive)
}

func TestResolveValidatorTotals_FallsBackToFullScanWhenNoBaseline(t *testing.T) {
	sm := newTestStateMachine(t)
	st := sm.store.(lib.StoreI)
	require.NoError(t, sm.SetParams(DefaultParams()))

	v := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{0x51}, crypto.AddressSize)})
	require.NoError(t, sm.SetValidator(&Validator{Address: bytes.Repeat([]byte{0x51}, crypto.AddressSize)}))
	_, err := st.Commit()
	require.NoError(t, err)

	totals, err := sm.resolveValidatorTotals(st, 2, [][]byte{v}, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(1), totals.ValidatorsActive)

	cached, available, err := st.GetValidatorTotals(2)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, uint32(1), cached.ValidatorsActive)
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

// TestIndexerBlobsFromStateChanges_NonSignerAddressRoundTrips is a regression for the
// journal path returning raw NonSigner state values as-is: NonSigner's wire encoding
// carries only Counter/ChainCounters (see KeyForNonSigner's write path in byzantine.go),
// never the Address -- that field is reconstructed from the storage key on every other
// read path. The selective (journal) fetch must do the same reconstruction, or every
// non-signer delta the indexer serves comes back with an empty address.
func TestIndexerBlobsFromStateChanges_NonSignerAddressRoundTrips(t *testing.T) {
	log := lib.NewDefaultLogger()
	config := lib.DefaultConfig()
	config.StoreConfig.StateChangeJournalEnabled = true
	db, err := store.NewStoreInMemory(log, config)
	require.NoError(t, err)
	defer db.Close()

	sm := StateMachine{
		store:             db,
		ProtocolVersion:   0,
		NetworkID:         1,
		height:            2,
		slashTracker:      NewSlashTracker(),
		proposeVoteConfig: AcceptAllProposals,
		Config: lib.Config{
			MainConfig:         lib.DefaultMainConfig(),
			StateMachineConfig: lib.DefaultStateMachineConfig(),
		},
		events: new(lib.EventsTracker),
		log:    log,
		cache: &cache{
			accounts: make(map[uint64]*Account),
			pools:    make(map[uint64]*Pool),
		},
	}
	now := uint64(time.Now().UnixMicro())

	// version 1: genesis params, no block to pair with yet.
	require.NoError(t, sm.SetParams(DefaultParams()))
	_, err = db.Commit()
	require.NoError(t, err)

	// version 2: pairs with block 1.
	require.NoError(t, sm.SetParams(DefaultParams()))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 1,
			Hash:   crypto.Hash([]byte("block-1")),
			Time:   now,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)

	// version 3: pairs with block 2, and journals the non-signer increment.
	kg := newTestKeyGroup(t, 0)
	address := kg.Address.Bytes()
	require.NoError(t, sm.SetParams(DefaultParams()))
	require.NoError(t, sm.IncrementNonSigners(0, [][]byte{kg.PublicKey.Bytes()}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 2,
			Hash:   crypto.Hash([]byte("block-2")),
			Time:   now + 1,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	sm.height = 3

	blobs, available, err := sm.IndexerBlobsFromStateChanges(context.Background(), 3)
	require.NoError(t, err)
	require.True(t, available)
	require.True(t, blobs.Current.NonSignersDelta)
	require.Len(t, blobs.Current.NonSigners, 1)

	ns := new(NonSigner)
	require.NoError(t, lib.Unmarshal(blobs.Current.NonSigners[0], ns))
	require.Equal(t, address, ns.Address)
}
