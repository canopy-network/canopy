package controller

import (
	"bytes"
	"reflect"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// failGetStore is a store whose Get() always fails, used to force GetAccount() and
// GetValidator() to return a nil result alongside an error, mirroring a store read failure.
type failGetStore struct {
	lib.RWStoreI
}

// Get() always returns an error so callers exercise their error handling paths.
func (failGetStore) Get([]byte) ([]byte, lib.ErrorI) {
	// return a store 'get' error to stand in for a database read failure
	return nil, lib.NewError(lib.CodeStoreGet, lib.StorageModule, "forced get failure")
}

// TestUpdateTelemetryHandlesGetAccountError ensures UpdateTelemetry() does not panic when
// GetAccount() returns a nil account together with an error (for example a store read failure).
func TestUpdateTelemetryHandlesGetAccountError(t *testing.T) {
	// build a state machine backed by a store whose Get() always fails
	sm := newFailGetStateMachine(t)
	// build a minimal controller wired to the failing state machine (nil Metrics is safe)
	c := &Controller{
		Address:   bytes.Repeat([]byte{1}, crypto.AddressSize),
		FSM:       sm,
		isSyncing: &atomic.Bool{},
	}
	// build a minimal block and quorum certificate for the telemetry update
	block := &lib.Block{BlockHeader: &lib.BlockHeader{}}
	qc := &lib.QuorumCertificate{}
	// the telemetry update must tolerate a failing account lookup without panicking
	require.NotPanics(t, func() {
		c.UpdateTelemetry(qc, block, 0)
	})
}

// newFailGetStateMachine builds a minimal StateMachine backed by a store whose Get() always fails.
func newFailGetStateMachine(t *testing.T) *fsm.StateMachine {
	t.Helper()
	// create an empty state machine
	sm := &fsm.StateMachine{}
	// wire in the failing store through the exported setter
	sm.SetStore(failGetStore{})
	// the cache is an unexported field with no setter; allocate it via reflection so that
	// GetAccount()'s cache lookup does not nil-panic before it reaches the failing store
	field := reflect.ValueOf(sm).Elem().FieldByName("cache")
	require.True(t, field.IsValid())
	// allocate a fresh cache value
	cacheValue := reflect.New(field.Type().Elem())
	// initialize the 'accounts' map so the cache lookup is a clean miss
	accounts := cacheValue.Elem().FieldByName("accounts")
	reflect.NewAt(accounts.Type(), unsafe.Pointer(accounts.UnsafeAddr())).Elem().Set(reflect.MakeMap(accounts.Type()))
	// set the cache field on the state machine
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(cacheValue)
	return sm
}
