package controller

import (
	"bytes"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// TestNextProposalTimeSkipsOldSlotsArithmetically() verifies stale proposal slots are skipped arithmetically.
func TestNextProposalTimeSkipsOldSlotsArithmetically(t *testing.T) {
	interval := 750 * time.Millisecond
	leadTime := 225 * time.Millisecond
	previous := uint64(time.Unix(1, 0).UnixMicro())
	now := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	got := nextProposalTime(previous, now, interval, leadTime)
	minimum := now.Add(leadTime)

	require.False(t, got.Before(minimum))
	intervalMicros := uint64(interval / time.Microsecond)
	require.Zero(t, (uint64(got.UnixMicro())-(previous+intervalMicros))%intervalMicros)
}

// TestNextProposalTimeUsesImmediateNextSlotWhenReady() verifies the immediate next slot is used when sufficient lead time remains.
func TestNextProposalTimeUsesImmediateNextSlotWhenReady(t *testing.T) {
	now := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	previous := uint64(now.UnixMicro())
	config := lib.DefaultConsensusConfig()
	interval := config.RoundRobinBlockInterval()
	require.True(t, now.Add(interval).Equal(nextProposalTime(previous, now, interval, config.BlockConsensusLeadTime())))
}

// TestValidateProposalTimeRequiresMonotonicTimestamp() verifies proposal timestamps increase monotonically.
func TestValidateProposalTimeRequiresMonotonicTimestamp(t *testing.T) {
	previous := &lib.BlockResult{BlockHeader: &lib.BlockHeader{Time: 1_000}}
	require.Error(t, validateProposalTime(999, previous))
	require.Error(t, validateProposalTime(1_000, previous))
	require.NoError(t, validateProposalTime(1_001, previous))
	require.Error(t, validateProposalTime(1_001, nil))
}

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
		log:       lib.NewNullLogger(),
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
