package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// restaking_test.go covers Phase C (WP §7) policy + observability:
//   - RestakingPolicy validation (target weights sum 10000; per-entry min ≤
//     max; duplicate committee ids rejected; empty list OK).
//   - Snapshot-derived allocation: restake semantics (same operator stake
//     counts toward every committee in its committees[]); absent-operator
//     skip; empty registry.
//   - buildRestakingView: weight bps, drift, under-min / over-max flags,
//     PolicyCompliant aggregate flag.
//   - QueryRestaking returns a well-formed view for empty / populated state.

// --- ValidateParams tests ---

// TestValidateRestakingPolicyEmpty: an empty policy is valid (the spec's
// "single-committee fallback" — observation-only mode without targets).
func TestValidateRestakingPolicyEmpty(t *testing.T) {
	p := DefaultParams()
	p.RestakingPolicy = nil
	if err := ValidateParams(p); err != nil {
		t.Fatalf("empty restaking policy should validate, got: %v", err)
	}
}

// TestValidateRestakingPolicyWeightSum: target weights must sum to 10000
// exactly when the policy is non-empty.
func TestValidateRestakingPolicyWeightSum(t *testing.T) {
	cases := []struct {
		name   string
		policy []*contract.RestakingPolicyEntry
		wantOK bool
	}{
		{"single 10000", []*contract.RestakingPolicyEntry{{CommitteeId: 1, TargetWeightBps: 10_000}}, true},
		{"split 5000/5000", []*contract.RestakingPolicyEntry{
			{CommitteeId: 1, TargetWeightBps: 5_000},
			{CommitteeId: 2, TargetWeightBps: 5_000},
		}, true},
		{"under sum 9999", []*contract.RestakingPolicyEntry{
			{CommitteeId: 1, TargetWeightBps: 5_000},
			{CommitteeId: 2, TargetWeightBps: 4_999},
		}, false},
		{"over sum 10001", []*contract.RestakingPolicyEntry{
			{CommitteeId: 1, TargetWeightBps: 5_000},
			{CommitteeId: 2, TargetWeightBps: 5_001},
		}, false},
	}
	for _, tc := range cases {
		p := DefaultParams()
		p.RestakingPolicy = tc.policy
		err := ValidateParams(p)
		if (err == nil) != tc.wantOK {
			t.Errorf("%s: validate err=%v, wantOK=%v", tc.name, err, tc.wantOK)
		}
	}
}

// TestValidateRestakingPolicyDuplicateCommittee: two entries for the same
// committee id are rejected — drift reporting would be ambiguous.
func TestValidateRestakingPolicyDuplicateCommittee(t *testing.T) {
	p := DefaultParams()
	p.RestakingPolicy = []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 5_000},
		{CommitteeId: 1, TargetWeightBps: 5_000},
	}
	if err := ValidateParams(p); err == nil {
		t.Fatal("duplicate committee_id should be rejected")
	}
}

// TestValidateRestakingPolicyMinMax: per-entry min ≤ max (when both set).
func TestValidateRestakingPolicyMinMax(t *testing.T) {
	// min > max → rejected
	p := DefaultParams()
	p.RestakingPolicy = []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MinStakeUcnpy: 1_000, MaxStakeUcnpy: 500},
	}
	if err := ValidateParams(p); err == nil {
		t.Fatal("min > max should be rejected")
	}

	// min set, max=0 → valid (max is optional)
	p2 := DefaultParams()
	p2.RestakingPolicy = []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MinStakeUcnpy: 1_000},
	}
	if err := ValidateParams(p2); err != nil {
		t.Errorf("min set with max=0 should validate: %v", err)
	}

	// min == max → valid (a hard pin)
	p3 := DefaultParams()
	p3.RestakingPolicy = []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MinStakeUcnpy: 500, MaxStakeUcnpy: 500},
	}
	if err := ValidateParams(p3); err != nil {
		t.Errorf("min == max should validate: %v", err)
	}
}

// --- snapshot-derived allocation tests ---
//
// The per-committee exposure map lives on Snapshot.CurrentRestakingAllocation,
// populated inline by refreshSnapshot's Batch 2 (see snapshot.go's
// qCanopyVal branch). These tests seed Canopy state, run refreshSnapshot,
// and assert on the resulting map — exactly the production path.

// seedCanopyValidator writes a lib.Validator-shaped record at the FSM
// validator key, exercising KeyForValidator + the Validator proto.
func seedCanopyValidator(t *testing.T, s *fakeStore, addr []byte, staked uint64, committees []uint64) {
	t.Helper()
	bz, err := contract.Marshal(&contract.Validator{
		Address:      addr,
		StakedAmount: staked,
		Committees:   committees,
	})
	if err != nil {
		t.Fatalf("marshal validator: %v", err)
	}
	s.set(contract.KeyForValidator(addr), bz)
}

// seedCanoliqRegistry writes a ValidatorRegistry singleton to fake state.
func seedCanoliqRegistry(t *testing.T, s *fakeStore, entries ...*contract.ValidatorRegistryEntry) {
	t.Helper()
	bz, err := contract.Marshal(&contract.ValidatorRegistry{Entries: entries})
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	s.set(KeyForValidatorRegistry(), bz)
}

// TestSnapshotRestakeSemantics: an operator on N committees contributes
// its FULL staked_amount to each — that's the point of restaking. Two
// operators on overlapping committees stack.
func TestSnapshotRestakeSemantics(t *testing.T) {
	c, s := newTestCanoliq()
	a := addr20(0xA1)
	b := addr20(0xB2)

	// Operator A: 1000 staked, committees [2, 3] → contributes 1000 to
	// both committees 2 and 3.
	seedCanopyValidator(t, s, a, 1000, []uint64{2, 3})
	// Operator B: 500 staked, committees [3] → contributes 500 to
	// committee 3 only.
	seedCanopyValidator(t, s, b, 500, []uint64{3})
	seedCanoliqRegistry(t, s,
		&contract.ValidatorRegistryEntry{Address: a, Stake: 1000},
		&contract.ValidatorRegistryEntry{Address: b, Stake: 500},
	)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	got := c.plugin.Snapshot().CurrentRestakingAllocation
	if got[2] != 1000 {
		t.Errorf("committee 2 exposure: got %d want 1000", got[2])
	}
	if got[3] != 1500 { // 1000 (A) + 500 (B)
		t.Errorf("committee 3 exposure: got %d want 1500", got[3])
	}
	if len(got) != 2 {
		t.Errorf("unexpected extra committees in map: %+v", got)
	}
}

// TestSnapshotSkipsAbsentOperator: an operator in canoLiq's registry but
// absent from Canopy validators (un-bonded, never registered) contributes
// zero and doesn't error.
func TestSnapshotSkipsAbsentOperator(t *testing.T) {
	c, s := newTestCanoliq()
	known := addr20(0x01)
	unknown := addr20(0x02)
	seedCanopyValidator(t, s, known, 100, []uint64{5})
	// `unknown` deliberately not seeded as a Canopy validator.

	seedCanoliqRegistry(t, s,
		&contract.ValidatorRegistryEntry{Address: known, Stake: 100},
		&contract.ValidatorRegistryEntry{Address: unknown, Stake: 999}, // canoLiq says here, Canopy disagrees
	)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	got := c.plugin.Snapshot().CurrentRestakingAllocation
	if got[5] != 100 {
		t.Errorf("known operator exposure: got %d want 100", got[5])
	}
	if len(got) != 1 {
		t.Errorf("absent operator should not appear: %+v", got)
	}
}

// TestSnapshotEmptyRegistry: nil / empty registry leaves the allocation
// map nil (snapshot only allocates on first non-empty operator decode).
// QueryRestaking handles nil cleanly — see TestQueryRestakingShapeEmpty.
func TestSnapshotEmptyRegistry(t *testing.T) {
	c, s := newTestCanoliq()
	seedCanoliqRegistry(t, s) // explicit empty registry
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	got := c.plugin.Snapshot().CurrentRestakingAllocation
	if len(got) != 0 {
		t.Errorf("expected empty allocation, got %+v", got)
	}
}

// --- buildRestakingView tests ---

// TestBuildViewDriftAndCompliance covers: weight bps from observed total,
// drift bps against policy target, under-min / over-max detection, and
// the PolicyCompliant aggregate flag.
func TestBuildViewDriftAndCompliance(t *testing.T) {
	policy := []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 5_000, MinStakeUcnpy: 200, MaxStakeUcnpy: 800},
		{CommitteeId: 2, TargetWeightBps: 5_000},
	}
	observed := map[uint64]uint64{1: 400, 2: 600} // total 1000 → 40% / 60%

	v := buildRestakingView(policy, observed)

	if v.TotalExposureUcnpy != 1000 {
		t.Errorf("total exposure: got %d want 1000", v.TotalExposureUcnpy)
	}
	if len(v.Allocations) != 2 {
		t.Fatalf("alloc count: got %d want 2", len(v.Allocations))
	}
	// committee 1: 400/1000 = 4000 bps, target 5000 → drift -1000
	a1 := v.Allocations[0]
	if a1.CommitteeID != 1 || a1.WeightBps != 4_000 || a1.TargetBps != 5_000 || a1.DriftBps != -1000 {
		t.Errorf("committee 1: %+v", a1)
	}
	if a1.UnderMin || a1.OverMax {
		t.Errorf("committee 1 in [200, 800]: under=%v over=%v", a1.UnderMin, a1.OverMax)
	}
	// committee 2: 600/1000 = 6000 bps, target 5000 → drift +1000
	a2 := v.Allocations[1]
	if a2.CommitteeID != 2 || a2.WeightBps != 6_000 || a2.TargetBps != 5_000 || a2.DriftBps != 1000 {
		t.Errorf("committee 2: %+v", a2)
	}
	// No bound violation → compliant.
	if !v.PolicyCompliant {
		t.Error("expected PolicyCompliant=true (drift only, no min/max violation)")
	}
}

// TestBuildViewUnderMinTripsCompliance: observed below min trips the
// aggregate non-compliant flag.
func TestBuildViewUnderMinTripsCompliance(t *testing.T) {
	policy := []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MinStakeUcnpy: 500},
	}
	v := buildRestakingView(policy, map[uint64]uint64{1: 100})
	if !v.Allocations[0].UnderMin {
		t.Error("expected UnderMin=true (observed 100 < min 500)")
	}
	if v.PolicyCompliant {
		t.Error("expected PolicyCompliant=false")
	}
}

// TestBuildViewOverMaxTripsCompliance: observed above max trips the
// aggregate non-compliant flag.
func TestBuildViewOverMaxTripsCompliance(t *testing.T) {
	policy := []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MaxStakeUcnpy: 500},
	}
	v := buildRestakingView(policy, map[uint64]uint64{1: 999})
	if !v.Allocations[0].OverMax {
		t.Error("expected OverMax=true (observed 999 > max 500)")
	}
	if v.PolicyCompliant {
		t.Error("expected PolicyCompliant=false")
	}
}

// TestBuildViewObservedWithoutPolicy: committees that have exposure but
// no policy entry still appear in Allocations (no target, no drift),
// so operators can see drift-from-empty-policy too.
func TestBuildViewObservedWithoutPolicy(t *testing.T) {
	v := buildRestakingView(nil, map[uint64]uint64{7: 100, 9: 50})
	if len(v.Allocations) != 2 {
		t.Fatalf("alloc count: got %d want 2", len(v.Allocations))
	}
	for _, a := range v.Allocations {
		if a.TargetBps != 0 || a.DriftBps != 0 {
			t.Errorf("no-policy alloc should have target=0 drift=0: %+v", a)
		}
	}
	if !v.PolicyCompliant {
		t.Error("empty policy is always compliant (no constraints)")
	}
}

// TestBuildViewPolicyWithoutObservation: a policy entry for a committee
// canoLiq has zero exposure to surfaces as a 0-stake / under-min entry,
// not silently dropped.
func TestBuildViewPolicyWithoutObservation(t *testing.T) {
	policy := []*contract.RestakingPolicyEntry{
		{CommitteeId: 5, TargetWeightBps: 10_000, MinStakeUcnpy: 100},
	}
	v := buildRestakingView(policy, map[uint64]uint64{})
	if len(v.Allocations) != 1 {
		t.Fatalf("expected 1 alloc, got %d", len(v.Allocations))
	}
	a := v.Allocations[0]
	if a.StakeUcnpy != 0 || a.WeightBps != 0 || !a.UnderMin {
		t.Errorf("zero-exposure entry: %+v", a)
	}
}

// --- Snapshot integration ---

// TestSnapshotPopulatesRestakingAllocation: refreshSnapshot reads the
// per-operator Canopy Validators added to Batch 2 and builds
// CurrentRestakingAllocation in one pass — no extra round-trip.
//
// Note: CurrentRestakingAllocation is a Go map (iteration order
// undefined). Deterministic ordering for /v1/restaking comes from
// buildRestakingView's sort on the union of committee ids — see
// TestBuildViewDriftAndCompliance for that contract.
func TestSnapshotPopulatesRestakingAllocation(t *testing.T) {
	c, s := newTestCanoliq()

	a := addr20(0xA1)
	b := addr20(0xB2)
	seedCanopyValidator(t, s, a, 1_000_000, []uint64{2, 7})
	seedCanopyValidator(t, s, b, 500_000, []uint64{7})

	registry := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: a, Stake: 1_000_000},
		{Address: b, Stake: 500_000},
	}}
	regBz, err := contract.Marshal(registry)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	s.set(KeyForValidatorRegistry(), regBz)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	snap := c.plugin.Snapshot()
	if snap.CurrentRestakingAllocation == nil {
		t.Fatal("expected CurrentRestakingAllocation populated, got nil")
	}
	if snap.CurrentRestakingAllocation[2] != 1_000_000 {
		t.Errorf("committee 2: got %d want 1_000_000", snap.CurrentRestakingAllocation[2])
	}
	if snap.CurrentRestakingAllocation[7] != 1_500_000 {
		t.Errorf("committee 7: got %d want 1_500_000", snap.CurrentRestakingAllocation[7])
	}
}

// TestQueryRestakingShapeEmpty: with empty policy + empty allocation the
// response is well-formed (PolicyCompliant=true, zero total, empty
// allocations).
func TestQueryRestakingShapeEmpty(t *testing.T) {
	c, _ := newTestCanoliq()
	if err := c.refreshSnapshot(0); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	v := c.plugin.QueryRestaking()
	if v == nil {
		t.Fatal("QueryRestaking returned nil")
	}
	if v.TotalExposureUcnpy != 0 {
		t.Errorf("expected zero exposure, got %d", v.TotalExposureUcnpy)
	}
	if !v.PolicyCompliant {
		t.Error("empty state should report compliant")
	}
	if len(v.Allocations) != 0 {
		t.Errorf("expected no allocations, got %d", len(v.Allocations))
	}
}

// TestQueryRestakingShapeWithPolicy: with a configured policy and
// observed exposure, /v1/restaking reports the full surface end-to-end
// (policy passes through, drift computed, compliance derived).
func TestQueryRestakingShapeWithPolicy(t *testing.T) {
	c, s := newTestCanoliq()
	a := addr20(0xA1)
	seedCanopyValidator(t, s, a, 1000, []uint64{1})
	registry := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: a, Stake: 1000}}}
	regBz, _ := contract.Marshal(registry)
	s.set(KeyForValidatorRegistry(), regBz)

	params := DefaultParams()
	params.RestakingPolicy = []*contract.RestakingPolicyEntry{
		{CommitteeId: 1, TargetWeightBps: 10_000, MaxStakeUcnpy: 500},
	}
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	v := c.plugin.QueryRestaking()
	if v.TotalExposureUcnpy != 1000 {
		t.Errorf("total exposure: got %d want 1000", v.TotalExposureUcnpy)
	}
	if len(v.Policy) != 1 {
		t.Fatalf("policy: got %d entries want 1", len(v.Policy))
	}
	if len(v.Allocations) != 1 {
		t.Fatalf("allocations: got %d entries want 1", len(v.Allocations))
	}
	if !v.Allocations[0].OverMax {
		t.Error("expected OverMax=true (1000 > 500)")
	}
	if v.PolicyCompliant {
		t.Error("expected PolicyCompliant=false (OverMax tripped)")
	}
}
