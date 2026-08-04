package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestIndexerBlobs_IgnoresLegacyDeltaField(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	req := httptest.NewRequest(http.MethodPost, IndexerBlobsRoutePath, bytes.NewBufferString(`{"height":3,"delta":false}`))
	rec := httptest.NewRecorder()

	server.IndexerBlobs(rec, req, nil)

	require.Equal(t, http.StatusOK, rec.Code)

	got := new(fsm.IndexerBlobs)
	require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), got))
	require.NotNil(t, got.Current)
	require.Len(t, got.Current.Accounts, 1)
	require.NotNil(t, got.Previous)
}

func TestIndexerBlobsCached_CachesDeltaResponsesOnly(t *testing.T) {
	server := newTestIndexerBlobServer(t)

	got, bz, err := server.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, bz)
	require.Len(t, got.Current.Accounts, 1)
	require.NotNil(t, got.Previous)

	entry, ok := server.indexerBlobCache.get(3)
	require.True(t, ok)
	require.NotNil(t, entry)
	require.NotNil(t, entry.current)
	require.NotNil(t, entry.deltaBlobs)
	require.NotEmpty(t, entry.deltaBytes)
	// the cached full snapshot no longer carries accounts at all: IndexerBlob is now called
	// with skipAccounts=true, which is the whole point of the fast path -- the ~1.35M-account
	// scan that used to fill this field is exactly what was eliminated
	require.Empty(t, entry.current.Accounts)
	require.Same(t, got, entry.deltaBlobs)
	require.Equal(t, bz, entry.deltaBytes)

	gotAgain, bzAgain, err := server.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.Same(t, entry.deltaBlobs, gotAgain)
	require.Equal(t, entry.deltaBytes, bzAgain)
}

func TestIndexerBlobsCached_RetainsOnlyLatestFullSnapshot(t *testing.T) {
	server := newTestIndexerBlobServerWithHeights(t, 4)

	got3, _, err := server.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, got3)

	entry3, ok := server.indexerBlobCache.get(3)
	require.True(t, ok)
	require.NotNil(t, entry3)
	require.NotNil(t, entry3.current)

	got4, _, err := server.IndexerBlobsCached(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, got4)
	require.NotNil(t, got4.Previous)

	entry3, ok = server.indexerBlobCache.get(3)
	require.True(t, ok)
	require.NotNil(t, entry3)
	require.Nil(t, entry3.current)
	require.NotNil(t, entry3.deltaBlobs)
	require.NotEmpty(t, entry3.deltaBytes)

	entry4, ok := server.indexerBlobCache.get(4)
	require.True(t, ok)
	require.NotNil(t, entry4)
	require.NotNil(t, entry4.current)
	require.NotNil(t, entry4.deltaBlobs)
	require.NotEmpty(t, entry4.deltaBytes)
}

// the account delta on the response must come from Controller.GetAccountDelta, not from
// diffing two full account scans: added+changed land on Current as their FINAL values, and
// changed+removed land on Previous as their PREVIOUS values.
func TestIndexerBlobsCached_UsesAccountDeltaFastPath(t *testing.T) {
	server := newTestIndexerBlobServerWithHeights(t, 4)
	addrA := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))
	addrB := crypto.NewAddress(bytes.Repeat([]byte{0x22}, crypto.AddressSize))

	// state version 4 is produced by block 3, whose seeded delta is "A and B both changed"
	got, _, err := server.IndexerBlobsCached(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.NotNil(t, got.Previous)

	// current side: the post-block values
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 125),
		mustMarshalAccount(t, addrB.Bytes(), 75),
	}, got.Current.Accounts)
	// previous side: the pre-block values for the same two accounts
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 50),
	}, got.Previous.Accounts)

	// proof this did not come from the old full-scan diff: neither underlying snapshot blob
	// carries accounts anymore, so a diff of them could only have produced an empty result
	entry, ok := server.indexerBlobCache.get(4)
	require.True(t, ok)
	require.Empty(t, entry.current.Accounts)
}

// with ServeIndexerBlobsLive disabled, IndexerBlobsCached must fall all the way back to the
// old full-scan-and-diff behavior: both blobs get skipAccounts=false (full account scan),
// GetAccountDelta is never called, and overrideAccountDelta is skipped -- DeltaIndexerBlobs's
// own diff of the two fully-scanned snapshots is what serves the account delta. This is the
// operator's config-only kill switch back to the pre-fast-path behavior.
func TestIndexerBlobsCached_FlagDisabledUsesFullScanPath(t *testing.T) {
	server := newTestIndexerBlobServerWithHeights(t, 4)
	server.config.ServeIndexerBlobsLive = false
	addrA := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))
	addrB := crypto.NewAddress(bytes.Repeat([]byte{0x22}, crypto.AddressSize))

	got, _, err := server.IndexerBlobsCached(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.NotNil(t, got.Previous)

	// same accounts, same values as the fast path produces for this fixture -- proving the
	// full-scan-and-diff path reaches the identical result, not that it merely runs without
	// erroring
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 125),
		mustMarshalAccount(t, addrB.Bytes(), 75),
	}, got.Current.Accounts)
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 50),
	}, got.Previous.Accounts)

	// proof this came from the full scan, not the fast path: the cached full snapshot DOES
	// carry accounts now (the opposite of TestIndexerBlobsCached_UsesAccountDeltaFastPath's
	// assertion), because skipAccounts was false for both IndexerBlob calls
	entry, ok := server.indexerBlobCache.get(4)
	require.True(t, ok)
	require.NotEmpty(t, entry.current.Accounts)
}

// an added account has no previous-side entry and a removed account has no current-side
// entry, matching DeltaIndexerBlobs's existing convention
func TestIndexerBlobsCached_AddedAccountsOmittedFromPreviousSide(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	addrB := crypto.NewAddress(bytes.Repeat([]byte{0x22}, crypto.AddressSize))

	// state version 3 is produced by block 2, whose seeded delta is "B added, nothing else"
	got, _, err := server.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.NotNil(t, got.Previous)

	require.Equal(t, [][]byte{mustMarshalAccount(t, addrB.Bytes(), 50)}, got.Current.Accounts)
	require.Empty(t, got.Previous.Accounts)
}

// NOTE: the account-side assembly helpers (accountSide, sortedAccountEntries, the
// force-include read-back) live in fsm next to DeltaIndexerBlobs and are tested there
// (fsm/indexer_test.go: TestAccountSide_DoesNotMutateOrAliasInputs,
// TestAccountSide_SortsByAddressAscending, TestAccountDelta_MatchesOldFullScanAndDiff);
// this package only orchestrates fsm.AssembleAccountDeltaSides onto the response blobs.

// I-2. At height 2 there is no previous blob, and DeltaIndexerBlobs' documented behaviour is
// to keep Current's FULL account set -- with nothing to diff against, every account counts as
// added (pinned by fsm's TestDeltaIndexerBlobs_NoPreviousKeepsCurrent). The fast path cannot
// reproduce that, since a collector only reports what block 1 wrote, so this one height falls
// back to the full scan to keep the response byte-identical. It happens once in a chain's
// life, so the scan cost is irrelevant.
func TestIndexerBlobsCached_HeightTwoKeepsFullAccountSet(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	addrA := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))

	got, _, err := server.IndexerBlobsCached(context.Background(), 2)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.Nil(t, got.Previous)

	// state@2 holds only addrA, and it is present in full despite no account delta existing
	require.Equal(t, [][]byte{mustMarshalAccount(t, addrA.Bytes(), 100)}, got.Current.Accounts)

	// and the cached snapshot for this height carries the full set too, unlike height >= 3
	entry, ok := server.indexerBlobCache.get(2)
	require.True(t, ok)
	require.Len(t, entry.current.Accounts, 1)
}

// C-1 regression. A reward or slash event names an account that very often is NOT written by
// the block: a COMPOUNDING validator's reward is routed to UpdateValidatorStake
// (fsm/committee.go:223) with no AccountPrefix write at all, and a slash moves stake only
// (fsm/byzantine.go:359,386). The old full-scan path force-included those accounts on BOTH
// sides with identical bytes because its maps held every account in state
// (fsm/indexer.go:325-329, pinned by fsm's TestDeltaIndexerBlobs_ForceIncludeRewardSlashAccounts).
// The collector only ever sees WRITTEN accounts, so without an explicit read-back the fast
// path would silently emit a strictly smaller Accounts set on nearly every block that pays or
// slashes a validator.
//
// A test whose reward actually credits an account would NOT catch this -- the account would be
// written, so the collector would report it anyway. This fixture is built so the named
// accounts are never written.
func TestIndexerBlobsCached_ForceIncludesUnwrittenRewardSlashAccounts(t *testing.T) {
	server, addrA, addrB, addrC := newTestIndexerBlobServerWithRewardSlashEvents(t)

	got, _, err := server.IndexerBlobsCached(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.NotNil(t, got.Previous)

	// A (compounding reward) and C (slash) were never written by block 3, so the collector
	// never saw them -- only B actually changed. All three must still be present.
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 75),
		mustMarshalAccount(t, addrC.Bytes(), 100),
	}, got.Current.Accounts)
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 50),
		mustMarshalAccount(t, addrC.Bytes(), 100),
	}, got.Previous.Accounts)

	// the force-include rule emits an unchanged account on both sides with IDENTICAL bytes
	require.Equal(t, got.Current.Accounts[0], got.Previous.Accounts[0], "A: reward, unwritten")
	require.Equal(t, got.Current.Accounts[2], got.Previous.Accounts[2], "C: slash, unwritten")
	// ...while an account that genuinely changed keeps its own per-side value
	require.NotEqual(t, got.Current.Accounts[1], got.Previous.Accounts[1], "B: genuinely changed")
}

// newTestIndexerBlobServerWithRewardSlashEvents builds a 4-height fixture where block 3 emits
// a reward event for addrA and a slash event for addrC, but writes NEITHER account -- exactly
// the compounding-reward / stake-slash shape. Only addrB's account changes (50 -> 75).
func newTestIndexerBlobServerWithRewardSlashEvents(t *testing.T) (_ *Server, addrA, addrB, addrC crypto.AddressI) {
	t.Helper()

	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	addrA = crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))
	addrB = crypto.NewAddress(bytes.Repeat([]byte{0x22}, crypto.AddressSize))
	addrC = crypto.NewAddress(bytes.Repeat([]byte{0x33}, crypto.AddressSize))
	now := uint64(time.Now().UnixMicro())

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)

	// block 1 and block 2 establish A, B and C in state
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 100}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 1, Hash: crypto.Hash([]byte("rs-block-1")), Time: now},
	}))
	_, err = db.Commit()
	require.NoError(t, err)

	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrB.Bytes(), Amount: 50}))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrC.Bytes(), Amount: 100}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 2, Hash: crypto.Hash([]byte("rs-block-2")), Time: now + 1},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 3)

	// block 3: B's balance moves; A is rewarded and C is slashed, but both rewards land on
	// STAKE, so neither account is written
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrB.Bytes(), Amount: 75}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 3, Hash: crypto.Hash([]byte("rs-block-3")), Time: now + 2},
		Events: []*lib.Event{
			{EventType: string(lib.EventTypeReward), Address: addrA.Bytes()},
			{EventType: string(lib.EventTypeSlash), Address: addrC.Bytes()},
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 4)

	ctrl := &controller.Controller{FSM: sm}
	// what a live collector would have produced for block 3: B only
	ctrl.SeedAccountDeltaCache(3, &fsm.AccountDelta{Changed: []*fsm.AccountChangeEntry{{
		Address:    addrB.Bytes(),
		PrevValue:  mustMarshalAccount(t, addrB.Bytes(), 50),
		FinalValue: mustMarshalAccount(t, addrB.Bytes(), 75),
	}}})

	return &Server{
		controller:       ctrl,
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}, addrA, addrB, addrC
}

func TestAccountQueryReturnsVestingBreakdown(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	sm := server.controller.FSM
	address := crypto.NewAddress(bytes.Repeat([]byte{0x33}, crypto.AddressSize))

	require.NoError(t, sm.SetAccount(&fsm.Account{
		Address:            address.Bytes(),
		Amount:             150,
		Nonce:              7,
		VestingAmount:      100,
		VestingStartHeight: 1,
		VestingCliffHeight: 2,
		VestingEndHeight:   6,
	}))
	_, err := sm.Store().(lib.StoreI).Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, sm.Store().(lib.StoreI).Version())

	req := httptest.NewRequest(http.MethodPost, AccountRoutePath, bytes.NewBufferString(
		`{"height":0,"address":"`+address.String()+`"}`,
	))
	rec := httptest.NewRecorder()

	server.Account(rec, req, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	var got AccountView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, address.Bytes(), []byte(got.Address))
	require.Equal(t, uint64(110), got.Amount)
	require.Equal(t, uint64(7), got.Nonce)
	require.Equal(t, uint64(150), got.TotalAmount)
	require.Equal(t, uint64(60), got.VestedAmount)
	require.Equal(t, uint64(40), got.LockedAmount)
	require.Equal(t, uint64(100), got.VestingAmount)
	require.Equal(t, uint64(1), got.VestingStartHeight)
	require.Equal(t, uint64(2), got.VestingCliffHeight)
	require.Equal(t, uint64(6), got.VestingEndHeight)
}

func TestAccountsQueryReturnsVestingBreakdowns(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	sm := server.controller.FSM
	liquid := crypto.NewAddress(bytes.Repeat([]byte{0x44}, crypto.AddressSize))
	vested := crypto.NewAddress(bytes.Repeat([]byte{0x55}, crypto.AddressSize))

	require.NoError(t, sm.SetAccount(&fsm.Account{Address: liquid.Bytes(), Amount: 25, Nonce: 3}))
	require.NoError(t, sm.SetAccount(&fsm.Account{
		Address:            vested.Bytes(),
		Amount:             150,
		Nonce:              8,
		VestingAmount:      100,
		VestingStartHeight: 1,
		VestingCliffHeight: 2,
		VestingEndHeight:   6,
	}))
	_, err := sm.Store().(lib.StoreI).Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, sm.Store().(lib.StoreI).Version())

	req := httptest.NewRequest(http.MethodPost, AccountsRoutePath, bytes.NewBufferString(`{"height":0,"pageNumber":1,"perPage":20}`))
	rec := httptest.NewRecorder()

	server.Accounts(rec, req, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Results []AccountView `json:"results"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.NotEmpty(t, got.Results)

	amounts := make(map[string]AccountView, len(got.Results))
	for _, account := range got.Results {
		amounts[crypto.NewAddressFromBytes(account.Address).String()] = account
	}
	require.Equal(t, uint64(25), amounts[liquid.String()].Amount)
	require.Equal(t, uint64(3), amounts[liquid.String()].Nonce)
	require.Equal(t, uint64(25), amounts[liquid.String()].TotalAmount)
	require.Zero(t, amounts[liquid.String()].VestedAmount)
	require.Zero(t, amounts[liquid.String()].LockedAmount)

	vestedAccount, ok := amounts[vested.String()]
	require.True(t, ok)
	require.Equal(t, uint64(110), vestedAccount.Amount)
	require.Equal(t, uint64(8), vestedAccount.Nonce)
	require.Equal(t, uint64(150), vestedAccount.TotalAmount)
	require.Equal(t, uint64(60), vestedAccount.VestedAmount)
	require.Equal(t, uint64(40), vestedAccount.LockedAmount)
	require.Equal(t, uint64(100), vestedAccount.VestingAmount)
	require.Equal(t, uint64(1), vestedAccount.VestingStartHeight)
	require.Equal(t, uint64(2), vestedAccount.VestingCliffHeight)
	require.Equal(t, uint64(6), vestedAccount.VestingEndHeight)
}

func newTestIndexerBlobServer(t *testing.T) *Server {
	t.Helper()
	return newTestIndexerBlobServerWithHeights(t, 3)
}

func newTestIndexerBlobServerWithHeights(t *testing.T, height uint64) *Server {
	t.Helper()

	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	addrA := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))
	addrB := crypto.NewAddress(bytes.Repeat([]byte{0x22}, crypto.AddressSize))
	now := uint64(time.Now().UnixMicro())

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 100}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 1,
			Hash:   crypto.Hash([]byte("block-1")),
			Time:   now,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 100}))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrB.Bytes(), Amount: 50}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 2,
			Hash:   crypto.Hash([]byte("block-2")),
			Time:   now + 1,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 3)

	if height >= 4 {
		require.NoError(t, sm.SetParams(fsm.DefaultParams()))
		require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 125}))
		require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrB.Bytes(), Amount: 75}))
		require.NoError(t, db.IndexBlock(&lib.BlockResult{
			BlockHeader: &lib.BlockHeader{
				Height: 3,
				Hash:   crypto.Hash([]byte("block-3")),
				Time:   now + 2,
			},
		}))
		_, err = db.Commit()
		require.NoError(t, err)
		setFSMHeight(t, sm, 4)
	}

	ctrl := &controller.Controller{FSM: sm}

	// IndexerBlobsCached now sources account deltas from Controller.GetAccountDelta rather
	// than from a full account scan of both heights. GetAccountDelta's replay fallback is not
	// reachable from this fixture -- these blocks are bare headers with no certificate and the
	// state has no validator set, so an ApplyBlock replay fails with "no validators in the
	// set". Seed the tip cache instead with exactly the deltas a live AccountChangeCollector
	// would have produced for the state transitions the fixture performs above, so the tests
	// exercise the real GetAccountDelta cache-hit path (including its block-height keying).
	//
	// cache key is BLOCK height; state version N is produced by block N-1.
	//
	// block 2 (state 2 -> 3): B is created; A is rewritten with an identical value, which a
	// collector classifies as unchanged and therefore omits.
	ctrl.SeedAccountDeltaCache(2, &fsm.AccountDelta{Added: []*fsm.AccountChangeEntry{
		{Address: addrB.Bytes(), FinalValue: mustMarshalAccount(t, addrB.Bytes(), 50)},
	}})
	if height >= 4 {
		// block 3 (state 3 -> 4): both A and B change value
		ctrl.SeedAccountDeltaCache(3, &fsm.AccountDelta{Changed: []*fsm.AccountChangeEntry{
			{
				Address:    addrA.Bytes(),
				PrevValue:  mustMarshalAccount(t, addrA.Bytes(), 100),
				FinalValue: mustMarshalAccount(t, addrA.Bytes(), 125),
			},
			{
				Address:    addrB.Bytes(),
				PrevValue:  mustMarshalAccount(t, addrB.Bytes(), 50),
				FinalValue: mustMarshalAccount(t, addrB.Bytes(), 75),
			},
		}})
	}

	return &Server{
		controller:       ctrl,
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
		// matches lib.DefaultRPCConfig()'s default: the account-delta fast path is on
		// unless an operator explicitly turns it off.
		config: lib.Config{RPCConfig: lib.RPCConfig{ServeIndexerBlobsLive: true}},
	}
}

func mustMarshalAccount(t *testing.T, address []byte, amount uint64) []byte {
	t.Helper()
	bz, err := lib.Marshal(&fsm.Account{Address: address, Amount: amount})
	require.NoError(t, err)
	return bz
}

func newTestRPCStateMachine(t *testing.T, db lib.StoreI, log lib.LoggerI) *fsm.StateMachine {
	t.Helper()

	sm := &fsm.StateMachine{
		ProtocolVersion: 0,
		NetworkID:       1,
		Config: lib.Config{
			MainConfig:         lib.DefaultMainConfig(),
			StateMachineConfig: lib.DefaultStateMachineConfig(),
		},
	}

	setUnexportedField(t, sm, "store", db)
	setUnexportedField(t, sm, "height", uint64(2))
	setUnexportedField(t, sm, "slashTracker", fsm.NewSlashTracker())
	setUnexportedField(t, sm, "proposeVoteConfig", fsm.AcceptAllProposals)
	setUnexportedField(t, sm, "events", new(lib.EventsTracker))
	setUnexportedField(t, sm, "log", log)
	setFSMCache(t, sm)

	return sm
}

func setFSMHeight(t *testing.T, sm *fsm.StateMachine, height uint64) {
	t.Helper()
	setUnexportedField(t, sm, "height", height)
}

func setFSMCache(t *testing.T, sm *fsm.StateMachine) {
	t.Helper()

	field := reflect.ValueOf(sm).Elem().FieldByName("cache")
	require.True(t, field.IsValid())

	cacheValue := reflect.New(field.Type().Elem())
	accounts := cacheValue.Elem().FieldByName("accounts")
	reflect.NewAt(accounts.Type(), unsafe.Pointer(accounts.UnsafeAddr())).Elem().Set(reflect.MakeMap(accounts.Type()))
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(cacheValue)
}

func setUnexportedField(t *testing.T, target any, name string, value any) {
	t.Helper()

	field := reflect.ValueOf(target).Elem().FieldByName(name)
	require.True(t, field.IsValid(), name)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func TestIndexerBlobsCached_CancelledContextReturnsErrCancelled(t *testing.T) {
	server := newTestIndexerBlobServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, bz, err := server.IndexerBlobsCached(ctx, 3)
	require.NotNil(t, err)
	require.Equal(t, lib.CodeCancelled, err.Code())
	require.Nil(t, got)
	require.Nil(t, bz)

	_, ok := server.indexerBlobCache.get(3)
	require.False(t, ok, "a cancelled scan must not populate the cache")
}
