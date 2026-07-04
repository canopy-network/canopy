package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
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

// newTestServer builds a *Server with just the fields trackClosedOrders touches
// (closedOrdersMu, lastOrderIDs, closedOrders). It deliberately leaves controller nil -
// trackClosedOrders never calls s.controller, so this is safe.
func newTestServer() *Server {
	return &Server{
		lastOrderIDs: make(map[string]*lib.SellOrder),
		closedOrders: make(map[string]*closedOracleOrder),
	}
}

func sellOrder(id string) *lib.SellOrder {
	return &lib.SellOrder{Id: []byte(id)}
}

// Gap 1: an order present last call but missing from live now gets snapshotted into
// closedOrders with a fresh, non-zero ClosedAt.
func TestTrackClosedOrders_NewlyClosedOrderIsSnapshotted(t *testing.T) {
	s := newTestServer()

	// first call: order1 is live
	s.trackClosedOrders([]*lib.SellOrder{sellOrder("order1")}, 10)

	before := time.Now()
	// second call: order1 has disappeared from the live book
	responses := s.trackClosedOrders(nil, 10)
	after := time.Now()

	if len(responses) != 1 {
		t.Fatalf("expected 1 closed order, got %d", len(responses))
	}
	if responses[0].ClosedAt == nil {
		t.Fatal("expected ClosedAt to be set")
	}
	closedAt := *responses[0].ClosedAt
	if closedAt.IsZero() {
		t.Fatal("expected ClosedAt to be non-zero")
	}
	if closedAt.Before(before) || closedAt.After(after) {
		t.Fatalf("expected ClosedAt %v to be between %v and %v", closedAt, before, after)
	}

	s.closedOrdersMu.Lock()
	entry, ok := s.closedOrders[lib.BytesToString([]byte("order1"))]
	s.closedOrdersMu.Unlock()
	if !ok {
		t.Fatal("expected order1 to be present in closedOrders map")
	}
	if entry.ClosedAt.IsZero() {
		t.Fatal("expected map entry ClosedAt to be non-zero")
	}
}

// Gap 2: an order already recorded as closed, still missing from live, is not re-added or
// re-timestamped - ClosedAt must be unchanged across calls.
func TestTrackClosedOrders_AlreadyClosedOrderNotRetimestamped(t *testing.T) {
	s := newTestServer()

	s.trackClosedOrders([]*lib.SellOrder{sellOrder("order1")}, 10)
	first := s.trackClosedOrders(nil, 10)
	if len(first) != 1 {
		t.Fatalf("expected 1 closed order after first close, got %d", len(first))
	}
	firstClosedAt := *first[0].ClosedAt

	// sleep a tiny amount so that if the code incorrectly re-stamps ClosedAt, the test would
	// have a chance to observe a different timestamp
	time.Sleep(2 * time.Millisecond)

	second := s.trackClosedOrders(nil, 10)
	if len(second) != 1 {
		t.Fatalf("expected 1 closed order after second call, got %d", len(second))
	}
	secondClosedAt := *second[0].ClosedAt

	if !firstClosedAt.Equal(secondClosedAt) {
		t.Fatalf("expected ClosedAt to remain unchanged, got %v then %v", firstClosedAt, secondClosedAt)
	}
}

// Gap 3: an order that was closed and then reappears in the live order book gets deleted from
// closedOrders (regression test for the reopen bug fix).
func TestTrackClosedOrders_ReopenedOrderIsRemovedFromClosed(t *testing.T) {
	s := newTestServer()

	s.trackClosedOrders([]*lib.SellOrder{sellOrder("order1")}, 10)
	closedResp := s.trackClosedOrders(nil, 10)
	if len(closedResp) != 1 {
		t.Fatalf("expected order1 to be closed, got %d closed orders", len(closedResp))
	}

	// order1 reappears in the live book
	reopenedResp := s.trackClosedOrders([]*lib.SellOrder{sellOrder("order1")}, 10)
	if len(reopenedResp) != 0 {
		t.Fatalf("expected 0 closed orders once order1 reappeared, got %d", len(reopenedResp))
	}

	s.closedOrdersMu.Lock()
	_, stillClosed := s.closedOrders[lib.BytesToString([]byte("order1"))]
	s.closedOrdersMu.Unlock()
	if stillClosed {
		t.Fatal("expected order1 to be removed from closedOrders after reappearing live")
	}
}

// Gap 4: trackClosedOrders must defensively copy the order it snapshots - mutating the
// original *lib.SellOrder after the snapshot must not affect the stored copy.
func TestTrackClosedOrders_SnapshotIsDefensiveCopy(t *testing.T) {
	s := newTestServer()

	original := sellOrder("order1")
	original.AmountForSale = 100

	s.trackClosedOrders([]*lib.SellOrder{original}, 10)
	s.trackClosedOrders(nil, 10) // order1 disappears and gets snapshotted

	// mutate the original after it has been snapshotted
	original.AmountForSale = 999999

	s.closedOrdersMu.Lock()
	entry, ok := s.closedOrders[lib.BytesToString([]byte("order1"))]
	s.closedOrdersMu.Unlock()
	if !ok {
		t.Fatal("expected order1 to be present in closedOrders map")
	}
	if entry.Order == original {
		t.Fatal("expected stored order to be a distinct copy, not the same pointer as original")
	}
	if entry.Order.AmountForSale != 100 {
		t.Fatalf("expected stored copy to be unaffected by later mutation, got AmountForSale=%d", entry.Order.AmountForSale)
	}
}

// Gap 5: an entry whose ClosedAt predates closedOrderRetention is pruned on a subsequent call -
// it must not appear in the returned slice nor remain in the map. No sleeping: ClosedAt is
// manually backdated.
func TestTrackClosedOrders_ExpiredEntryIsPruned(t *testing.T) {
	s := newTestServer()

	s.trackClosedOrders([]*lib.SellOrder{sellOrder("order1")}, 10)
	s.trackClosedOrders(nil, 10) // order1 closes

	// manually backdate ClosedAt beyond the retention window
	s.closedOrdersMu.Lock()
	s.closedOrders[lib.BytesToString([]byte("order1"))].ClosedAt = time.Now().Add(-closedOrderRetention - time.Minute)
	s.closedOrdersMu.Unlock()

	responses := s.trackClosedOrders(nil, 10)
	if len(responses) != 0 {
		t.Fatalf("expected expired entry to be pruned from returned slice, got %d entries", len(responses))
	}

	s.closedOrdersMu.Lock()
	_, stillPresent := s.closedOrders[lib.BytesToString([]byte("order1"))]
	s.closedOrdersMu.Unlock()
	if stillPresent {
		t.Fatal("expected expired entry to be pruned from the map")
	}
}

// Gap 6: the returned slice is sorted by ClosedAt descending (most recently closed first).
func TestTrackClosedOrders_SortedByClosedAtDescending(t *testing.T) {
	s := newTestServer()

	now := time.Now()

	s.closedOrdersMu.Lock()
	s.closedOrders["a"] = &closedOracleOrder{Order: sellOrder("a"), ClosedAt: now.Add(-3 * time.Hour)}
	s.closedOrders["b"] = &closedOracleOrder{Order: sellOrder("b"), ClosedAt: now.Add(-1 * time.Hour)}
	s.closedOrders["c"] = &closedOracleOrder{Order: sellOrder("c"), ClosedAt: now.Add(-2 * time.Hour)}
	s.closedOrdersMu.Unlock()

	responses := s.trackClosedOrders(nil, 10)
	if len(responses) != 3 {
		t.Fatalf("expected 3 closed orders, got %d", len(responses))
	}

	for i := 0; i < len(responses)-1; i++ {
		if responses[i].ClosedAt.Before(*responses[i+1].ClosedAt) {
			t.Fatalf("expected descending ClosedAt order, but entry %d (%v) is before entry %d (%v)",
				i, *responses[i].ClosedAt, i+1, *responses[i+1].ClosedAt)
		}
	}

	// double-check the exact expected order: b (most recent), c, a (oldest)
	wantOrder := []string{"b", "c", "a"}
	for i, id := range wantOrder {
		got := string(responses[i].OrderId)
		if got != id {
			t.Fatalf("expected responses[%d] to have OrderId %q, got %q", i, id, got)
		}
	}
}

// Gap 7: nil/empty live input with nothing previously tracked returns a non-panicking, empty
// slice.
func TestTrackClosedOrders_EmptyInputReturnsEmptySlice(t *testing.T) {
	s := newTestServer()

	responses := s.trackClosedOrders(nil, 10)
	if len(responses) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(responses))
	}

	responses = s.trackClosedOrders([]*lib.SellOrder{}, 10)
	if len(responses) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(responses))
	}
}

// Gap 8: concurrent calls to trackClosedOrders on the same *Server with overlapping order sets
// must not race. Run with `go test -race`.
func TestTrackClosedOrders_ConcurrentAccessNoRace(t *testing.T) {
	s := newTestServer()

	const goroutines = 20
	const itersPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < itersPerGoroutine; i++ {
				// build an overlapping but varying set of orders per goroutine/iteration
				live := []*lib.SellOrder{
					sellOrder("shared-order"),
					sellOrder(string(rune('A' + (g % 26)))),
				}
				if i%2 == 0 {
					live = live[:1]
				}
				_ = s.trackClosedOrders(live, uint64(i))
			}
		}(g)
	}
	wg.Wait()
}
