package rpc

import (
	"bytes"
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

	got, bz, err := server.IndexerBlobsCached(3)
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
	require.Len(t, entry.current.Accounts, 2)
	require.Same(t, got, entry.deltaBlobs)
	require.Equal(t, bz, entry.deltaBytes)

	gotAgain, bzAgain, err := server.IndexerBlobsCached(3)
	require.NoError(t, err)
	require.Same(t, entry.deltaBlobs, gotAgain)
	require.Equal(t, entry.deltaBytes, bzAgain)
}

func TestIndexerBlobsCached_RetainsOnlyLatestFullSnapshot(t *testing.T) {
	server := newTestIndexerBlobServerWithHeights(t, 4)

	got3, _, err := server.IndexerBlobsCached(3)
	require.NoError(t, err)
	require.NotNil(t, got3)

	entry3, ok := server.indexerBlobCache.get(3)
	require.True(t, ok)
	require.NotNil(t, entry3)
	require.NotNil(t, entry3.current)

	got4, _, err := server.IndexerBlobsCached(4)
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

func TestAccountQueryReturnsVestingBreakdown(t *testing.T) {
	server := newTestIndexerBlobServer(t)
	sm := server.controller.FSM
	address := crypto.NewAddress(bytes.Repeat([]byte{0x33}, crypto.AddressSize))

	require.NoError(t, sm.SetAccount(&fsm.Account{
		Address:            address.Bytes(),
		Amount:             150,
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
	require.Equal(t, uint64(150), got.TotalAmount)
	require.Equal(t, uint64(110), got.SpendableAmount)
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

	require.NoError(t, sm.SetAccount(&fsm.Account{Address: liquid.Bytes(), Amount: 25}))
	require.NoError(t, sm.SetAccount(&fsm.Account{
		Address:            vested.Bytes(),
		Amount:             150,
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
	require.Equal(t, uint64(25), amounts[liquid.String()].TotalAmount)
	require.Equal(t, uint64(25), amounts[liquid.String()].SpendableAmount)
	require.Zero(t, amounts[liquid.String()].VestedAmount)
	require.Zero(t, amounts[liquid.String()].LockedAmount)

	vestedAccount, ok := amounts[vested.String()]
	require.True(t, ok)
	require.Equal(t, uint64(110), vestedAccount.Amount)
	require.Equal(t, uint64(150), vestedAccount.TotalAmount)
	require.Equal(t, uint64(110), vestedAccount.SpendableAmount)
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

	return &Server{
		controller:       &controller.Controller{FSM: sm},
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}
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
