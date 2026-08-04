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
	require.Len(t, withoutFunderAccount(t, got.Current.Accounts), 1)
	require.NotNil(t, got.Previous)
}

func TestIndexerBlobsCached_CachesDeltaResponsesOnly(t *testing.T) {
	server := newTestIndexerBlobServer(t)

	got, bz, err := server.IndexerBlobsCached(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, bz)
	require.Len(t, withoutFunderAccount(t, got.Current.Accounts), 1)
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
	}, withoutFunderAccount(t, got.Current.Accounts))
	// previous side: the pre-block values for the same two accounts
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 50),
	}, withoutFunderAccount(t, got.Previous.Accounts))

	// proof this did not come from the old full-scan diff: neither underlying snapshot blob
	// carries accounts anymore, so a diff of them could only have produced an empty result
	entry, ok := server.indexerBlobCache.get(4)
	require.True(t, ok)
	require.Empty(t, entry.current.Accounts)
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

	require.Equal(t, [][]byte{mustMarshalAccount(t, addrB.Bytes(), 50)}, withoutFunderAccount(t, got.Current.Accounts))
	require.Empty(t, withoutFunderAccount(t, got.Previous.Accounts))
}

// NOTE: the account-side assembly helpers (accountSide, sortedAccountEntries, the
// force-include read-back) live in fsm next to DeltaIndexerBlobs and are tested there
// (fsm/indexer_test.go: TestAccountSide_DoesNotMutateOrAliasInputs,
// TestAccountSide_SortsByAddressAscending, TestAccountDelta_MatchesOldFullScanAndDiff);
// this package only orchestrates fsm.AssembleAccountDeltaSides onto the response blobs.

// At height 2 there is no previous blob and DeltaIndexerBlobs keeps Current's full
// account set. The fast path cannot reproduce that (a collector only reports what block 1
// wrote), so that one height falls back to the full scan to stay byte-identical.
func TestIndexerBlobsCached_HeightTwoKeepsFullAccountSet(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	addrA := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))

	got, _, err := server.IndexerBlobsCached(context.Background(), 2)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.Nil(t, got.Previous)

	// state@2 holds addrA (plus the funder, funded one version earlier for later blocks'
	// real sends), and both are present in full despite no account delta existing
	require.Equal(t, [][]byte{mustMarshalAccount(t, addrA.Bytes(), 100)}, withoutFunderAccount(t, got.Current.Accounts))

	// and the cached snapshot for this height carries the full set too, unlike height >= 3
	entry, ok := server.indexerBlobCache.get(2)
	require.True(t, ok)
	require.Len(t, withoutFunderAccount(t, entry.current.Accounts), 1)
}

// A reward or slash event often names an account the block never writes (compounding
// rewards and slashes move stake only), so the collector never sees it — yet the full-scan
// path force-included it on both sides. This fixture is built so the named accounts are
// never written; a reward that actually credited an account would not catch the regression.
func TestIndexerBlobsCached_ForceIncludesUnwrittenRewardSlashAccounts(t *testing.T) {
	server, addrA, addrB, addrC := newTestIndexerBlobServerWithRewardSlashEvents(t)

	got, _, err := server.IndexerBlobsCached(context.Background(), 4)
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	require.NotNil(t, got.Previous)
	current := withoutFunderAccount(t, got.Current.Accounts)
	previous := withoutFunderAccount(t, got.Previous.Accounts)

	// A (compounding reward) and C (slash) were never written by block 3, so the collector
	// never saw them -- only B actually changed. All three must still be present.
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 75),
		mustMarshalAccount(t, addrC.Bytes(), 100),
	}, current)
	require.Equal(t, [][]byte{
		mustMarshalAccount(t, addrA.Bytes(), 100),
		mustMarshalAccount(t, addrB.Bytes(), 50),
		mustMarshalAccount(t, addrC.Bytes(), 100),
	}, previous)

	// the force-include rule emits an unchanged account on both sides with IDENTICAL bytes
	require.Equal(t, current[0], previous[0], "A: reward, unwritten")
	require.Equal(t, current[2], previous[2], "C: slash, unwritten")
	// ...while an account that genuinely changed keeps its own per-side value
	require.NotEqual(t, current[1], previous[1], "B: genuinely changed")
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
	funder := rpcTestKeyGroup(t, 3)
	now := uint64(time.Now().UnixMicro())

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)

	// block 1 and block 2 establish A, B and C in state (hand-set: GetAccountDelta only ever
	// replays blockHeight > 1, so blocks 1-2's application is never replayed). Committee/
	// validators land in block 1's commit (version 2): block 3 is the first real ApplyBlock,
	// with sm.height=3, so its BeginBlock reads LoadCommittee(chainId, s.Height()-1=2) --
	// one version earlier than block 3's own state, matching where this commits them.
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 100}))
	setUpRPCTestValidators(t, sm)
	require.NoError(t, sm.AccountAdd(funder.Address, 10_000_000))
	require.NoError(t, sm.UpdateParam("fee", fsm.ParamSendFee, &lib.UInt64Wrapper{Value: 0}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 1, Hash: crypto.Hash([]byte("rs-block-1")), Time: now},
	}))
	_, err = db.Commit()
	require.NoError(t, err)

	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrB.Bytes(), Amount: 50}))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrC.Bytes(), Amount: 100}))
	// block 3's BeginBlock loads "last certificate" via LoadCertificate(s.Height()-1=2), and
	// HandleCertificateResults' non-signer calculation needs a real Signature too.
	placeholderQC2 := &lib.QuorumCertificate{
		Header:      &lib.View{Height: 2, ChainId: lib.CanopyChainId},
		ProposerKey: rpcTestKeyGroup(t, 0).PublicKey.Bytes(),
		Results:     &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{}},
	}
	signRPCTestQC(t, sm, placeholderQC2)
	require.NoError(t, db.IndexQC(placeholderQC2))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 2, Hash: crypto.Hash([]byte("rs-block-2")), Time: now + 1},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 3)
	sm.Reset()

	// block 3: a real ApplyBlock moves B's balance via a send from the funder; A and C are
	// never written, matching the compounding-reward / stake-slash shape where the reward
	// lands on STAKE, not the account. The reward/slash events are then appended by hand --
	// force-include reads events from the INDEXED block, not from real reward execution, so
	// this proves the read-back without needing real committee reward/slash machinery to fire.
	applyAndCommitRPCTestBlock(t, sm, db, [][]byte{
		rpcTestSendTx(t, sm, funder, addrB, 25),
	}, []*lib.Event{
		{EventType: string(lib.EventTypeReward), Address: addrA.Bytes()},
		{EventType: string(lib.EventTypeSlash), Address: addrC.Bytes()},
	})

	ctrl := &controller.Controller{FSM: sm, Config: sm.Config}

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
	funder := rpcTestKeyGroup(t, 3)
	now := uint64(time.Now().UnixMicro())

	// committee/validators must be committed at version 1: BeginBlock's LoadCommittee(chainId,
	// s.Height()-1) reads a HISTORICAL snapshot at that version, so with sm.height=2 (set
	// below) when the first real block applies, the committee must already exist at version 1
	// -- one commit earlier than block 1's own state.
	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	setUpRPCTestValidators(t, sm)
	require.NoError(t, sm.AccountAdd(funder.Address, 10_000_000))
	require.NoError(t, sm.UpdateParam("fee", fsm.ParamSendFee, &lib.UInt64Wrapper{Value: 0}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)

	// block 1: hand-set genesis-like state -- GetAccountDelta only ever replays blockHeight
	// > 1 (useAccountDelta requires state version > 2), so block 1's application is never
	// replayed and can stay a bare-header fixture.
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addrA.Bytes(), Amount: 100}))
	// block 2's BeginBlock loads "last certificate" via LoadCertificate(s.Height()-1=1), and
	// HandleCertificateResults' non-signer calculation needs a real Signature too.
	placeholderQC1 := &lib.QuorumCertificate{
		Header:      &lib.View{Height: 1, ChainId: lib.CanopyChainId},
		ProposerKey: rpcTestKeyGroup(t, 0).PublicKey.Bytes(),
		Results:     &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{}},
	}
	signRPCTestQC(t, sm, placeholderQC1)
	require.NoError(t, db.IndexQC(placeholderQC1))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 1,
			Hash:   crypto.Hash([]byte("block-1")),
			Time:   now,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)
	sm.Reset()

	// block 2 (state 2 -> 3): a real ApplyBlock creates B via a send from the funder. A is
	// never touched, so a collector correctly omits it (equivalent to the old fixture's
	// "rewritten with an identical value" case, without needing a no-op write to prove it).
	applyAndCommitRPCTestBlock(t, sm, db, [][]byte{
		rpcTestSendTx(t, sm, funder, addrB, 50),
	}, nil)

	if height >= 4 {
		// block 3 (state 3 -> 4): both A and B change value
		applyAndCommitRPCTestBlock(t, sm, db, [][]byte{
			rpcTestSendTx(t, sm, funder, addrA, 25),
			rpcTestSendTx(t, sm, funder, addrB, 25),
		}, nil)
	}

	ctrl := &controller.Controller{FSM: sm, Config: sm.Config}

	return &Server{
		controller:       ctrl,
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}
}

func mustMarshalAccount(t *testing.T, address []byte, amount uint64) []byte {
	t.Helper()
	bz, err := lib.Marshal(&fsm.Account{Address: address, Amount: amount})
	require.NoError(t, err)
	return bz
}

// withoutFunderAccount strips the funder key group's own account entry (its balance
// legitimately changes on every real send in these fixtures) so assertions can compare
// against just the test-relevant addresses.
func withoutFunderAccount(t *testing.T, entries [][]byte) [][]byte {
	t.Helper()
	funder := rpcTestKeyGroup(t, 3).Address.Bytes()
	out := make([][]byte, 0, len(entries))
	for _, e := range entries {
		acc := new(fsm.Account)
		require.NoError(t, lib.Unmarshal(e, acc))
		if bytes.Equal(acc.Address, funder) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func newTestRPCStateMachine(t *testing.T, db lib.StoreI, log lib.LoggerI) *fsm.StateMachine {
	t.Helper()

	sm := &fsm.StateMachine{
		ProtocolVersion: 1,
		NetworkID:       1,
		Config: lib.Config{
			MainConfig:         lib.DefaultMainConfig(),
			StateMachineConfig: lib.DefaultStateMachineConfig(),
		},
	}
	sm.Config.P2PConfig.NetworkID = 1

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
	// ApplyBlock touches both the account and the pool caches, so both maps must exist
	for _, name := range []string{"accounts", "pools"} {
		mapField := cacheValue.Elem().FieldByName(name)
		require.True(t, mapField.IsValid(), name)
		reflect.NewAt(mapField.Type(), unsafe.Pointer(mapField.UnsafeAddr())).Elem().Set(reflect.MakeMap(mapField.Type()))
	}
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(cacheValue)
}

// rpcTestBLSKeys are the same deterministic BLS12381 keys used throughout the fsm/controller
// test fixtures, so validator/committee/QC-signing behavior mirrors already-proven fixtures.
var rpcTestBLSKeys = []string{
	"01553a101301cd7019b78ffa1186842dd93923e563b8ae22e2ab33ae889b23ee",
	"1b6b244fbdf614acb5f0d00a2b56ffcbe2aa23dabd66365dffcd3f06491ae50a",
	"2ee868f74134032eacba191ca529115c64aa849ac121b75ca79b37420a623036",
	"3e3ab94c10159d63a12cb26aca4b0e76070a987d49dd10fc5f526031e05801da",
}

func rpcTestKeyGroup(t *testing.T, i int) *crypto.KeyGroup {
	t.Helper()
	key, err := crypto.StringToBLS12381PrivateKey(rpcTestBLSKeys[i])
	require.NoError(t, err)
	return crypto.NewKeyGroup(key)
}

// setUpRPCTestValidators configures a 4-validator committee (needed for ApplyBlock's
// validator-root calculation and BeginBlock's committee lookups) with no stake movement of
// its own -- fixtures that need real account-touching blocks build sends from a separately
// funded key-group address, since addrA/addrB/addrC are plain byte patterns with no known
// private key and can only ever be send RECIPIENTS.
func setUpRPCTestValidators(t *testing.T, sm *fsm.StateMachine) {
	t.Helper()
	supply := &fsm.Supply{}
	for i := 0; i < 4; i++ {
		kg := rpcTestKeyGroup(t, i)
		require.NoError(t, sm.SetValidators([]*fsm.Validator{{
			Address:      kg.Address.Bytes(),
			PublicKey:    kg.PublicKey.Bytes(),
			StakedAmount: 100,
			Committees:   []uint64{lib.CanopyChainId},
			Output:       kg.Address.Bytes(),
		}}, supply))
		require.NoError(t, sm.SetCommitteeMember(kg.Address, lib.CanopyChainId, 100))
	}
	require.NoError(t, sm.SetSupply(supply))
}

// signRPCTestQC signs qc with 3 of the 4 committee validators set up by setUpRPCTestValidators
func signRPCTestQC(t *testing.T, sm *fsm.StateMachine, qc *lib.QuorumCertificate) {
	t.Helper()
	committee, err := sm.GetCommitteeMembers(lib.CanopyChainId)
	require.NoError(t, err)
	mk := committee.MultiKey.Copy()
	for i := 0; i < 3; i++ {
		privateKey := rpcTestKeyGroup(t, i).PrivateKey
		for j, pubKey := range mk.PublicKeys() {
			if privateKey.PublicKey().Equals(pubKey) {
				require.NoError(t, mk.AddSigner(privateKey.Sign(qc.SignBytes()), j))
			}
		}
	}
	aggSig, e := mk.AggregateSignatures()
	require.NoError(t, e)
	qc.Signature = &lib.AggregateSignature{Signature: aggSig, Bitmap: mk.Bitmap()}
}

// rpcTestSendTx builds a signed zero-fee send transaction from the funder key group, valid
// at the state machine's current height
func rpcTestSendTx(t *testing.T, sm *fsm.StateMachine, funder *crypto.KeyGroup, to crypto.AddressI, amount uint64) []byte {
	t.Helper()
	tx, err := fsm.NewSendTransaction(funder.PrivateKey, to, amount, uint64(sm.NetworkID), sm.Config.ChainId, 0, sm.Height(), "")
	require.NoError(t, err)
	bz, mErr := lib.Marshal(tx)
	require.NoError(t, mErr)
	return bz
}

// applyAndCommitRPCTestBlock applies a block at sm.Height() with the given transactions,
// indexes the resulting BlockResult (with extraEvents appended, e.g. synthetic reward/slash
// events a test wants force-included without triggering real reward execution) and a signed
// certificate for that height, commits, and advances the state machine -- the same order
// controller.CommitCertificate uses.
func applyAndCommitRPCTestBlock(t *testing.T, sm *fsm.StateMachine, db lib.StoreI, txs [][]byte, extraEvents []*lib.Event) {
	t.Helper()
	height := sm.Height()
	block := &lib.Block{
		BlockHeader: &lib.BlockHeader{
			Height:          height,
			Time:            uint64(time.Now().UnixMicro()) + height*1_000_000,
			ProposerAddress: rpcTestKeyGroup(t, 0).Address.Bytes(),
		},
		Transactions: txs,
	}
	header, result, err := sm.ApplyBlock(context.Background(), block, false, nil, false)
	require.NoError(t, err)
	for _, f := range result.Failed {
		t.Logf("FAILED TX at height %d: %s", height, f.Error.Error())
	}
	require.Empty(t, result.Failed, "block at height %d had failed transactions", height)
	qc := &lib.QuorumCertificate{
		Header:      &lib.View{Height: height, ChainId: lib.CanopyChainId},
		ProposerKey: rpcTestKeyGroup(t, 0).PublicKey.Bytes(),
		Results:     &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{}}, // BeginBlock's HandleCertificateResults requires non-nil Results
	}
	signRPCTestQC(t, sm, qc)
	require.NoError(t, db.IndexQC(qc))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader:  header,
		Transactions: result.Results,
		Events:       append(result.Events, extraEvents...),
	}))
	_, e := db.Commit()
	require.NoError(t, e)
	setFSMHeight(t, sm, height+1)
	sm.Reset()
	require.Equal(t, sm.Height(), db.Version())
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
