package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// TestLoadParamsBackfillsFieldsAddedAfterGenesis is the regression test for a
// chain halt.
//
// LoadParams validates on every read and processProposals reads params from
// BeginBlock, so a params record persisted before a required field existed
// would fail ValidateParams on every block and take ApplyBlock down with it.
// Upgrading the plugin binary on a chain whose genesis predates the OTC tier
// terms was enough to trigger it.
func TestLoadParamsBackfillsFieldsAddedAfterGenesis(t *testing.T) {
	c, s := newTestCanoliq()

	// A record as an older build would have written it: every field it knew
	// about is set, and the OTC tier terms (proto fields 36/37) are simply
	// absent, so they decode as zero.
	legacy := DefaultParams()
	legacy.OtcTier90Blocks = 0
	legacy.OtcTier120Blocks = 0
	legacy.OtcMinLockUccnpy = 0
	legacy.CanoliqTransferFee = 0 // field 38, same class: added after some chains had already genesised
	bz, e := contract.Marshal(legacy)
	if e != nil {
		t.Fatalf("marshal legacy params: %v", e)
	}
	// Written directly rather than through SaveParams, which would reject it.
	// That asymmetry is the point: the record is already in state.
	s.set(KeyForParams(), bz)

	got, err := c.LoadParams()
	if err != nil {
		t.Fatalf("LoadParams on a pre-OTC record failed, which halts every block: %v", err)
	}
	d := DefaultParams()
	if got.OtcTier90Blocks != d.OtcTier90Blocks {
		t.Errorf("OtcTier90Blocks = %d, want the default %d", got.OtcTier90Blocks, d.OtcTier90Blocks)
	}
	if got.OtcTier120Blocks != d.OtcTier120Blocks {
		t.Errorf("OtcTier120Blocks = %d, want the default %d", got.OtcTier120Blocks, d.OtcTier120Blocks)
	}
	if got.OtcMinLockUccnpy != d.OtcMinLockUccnpy {
		t.Errorf("OtcMinLockUccnpy = %d, want the default %d", got.OtcMinLockUccnpy, d.OtcMinLockUccnpy)
	}
	if got.CanoliqTransferFee != d.CanoliqTransferFee {
		t.Errorf("CanoliqTransferFee = %d, want the default %d (a pre-existing chain would otherwise get free cCNPY transfers by omission, not governance choice)",
			got.CanoliqTransferFee, d.CanoliqTransferFee)
	}
}

// TestBackfillParamsLeavesDeliberateValuesAlone guards the other direction:
// the backfill must not quietly overwrite what governance actually chose.
func TestBackfillParamsLeavesDeliberateValuesAlone(t *testing.T) {
	tuned := DefaultParams()
	tuned.OtcTier90Blocks = 4_321
	tuned.OtcTier120Blocks = 8_765
	tuned.OtcMinLockUccnpy = 99
	backfillParams(tuned)
	if tuned.OtcTier90Blocks != 4_321 || tuned.OtcTier120Blocks != 8_765 || tuned.OtcMinLockUccnpy != 99 {
		t.Errorf("backfill overwrote configured terms: %d/%d/%d",
			tuned.OtcTier90Blocks, tuned.OtcTier120Blocks, tuned.OtcMinLockUccnpy)
	}

	// Zero is a legitimate tier rate: it is how governance disables one tier
	// while leaving the other running. Backfilling it would silently switch a
	// disabled tier back on.
	off := DefaultParams()
	off.OtcTier90Bps = 0
	backfillParams(off)
	if off.OtcTier90Bps != 0 {
		t.Errorf("backfill re-enabled a tier governance had disabled: OtcTier90Bps = %d", off.OtcTier90Bps)
	}

	// With both tiers off, the minimum position size is meaningless and must
	// stay untouched rather than being resurrected.
	allOff := DefaultParams()
	allOff.OtcTier90Bps, allOff.OtcTier120Bps, allOff.OtcMinLockUccnpy = 0, 0, 0
	backfillParams(allOff)
	if allOff.OtcMinLockUccnpy != 0 {
		t.Errorf("backfill set a minimum for a program with no live tier: %d", allOff.OtcMinLockUccnpy)
	}
}

// TestValidateParamsStillRejectsExplicitBadTerms confirms the backfill did not
// soften validation for values an operator actually supplied.
func TestValidateParamsStillRejectsExplicitBadTerms(t *testing.T) {
	over := DefaultParams()
	over.OtcTier90Blocks = maxOtcTierBlocks + 1
	if err := ValidateParams(over); err == nil {
		t.Error("an out-of-range tier term should still be rejected")
	}
	// ValidateParams is used as a pure validator by SaveParams and by
	// canoliqctl, so it must not backfill its argument.
	zero := DefaultParams()
	zero.OtcTier90Blocks = 0
	if err := ValidateParams(zero); err == nil {
		t.Error("ValidateParams should still reject a zero term rather than silently filling it")
	}
	if zero.OtcTier90Blocks != 0 {
		t.Error("ValidateParams mutated its argument; backfilling belongs on the read path only")
	}
}

// TestValidateParamsStakeOutputAddresses: the ownership set is the lever that
// decides whose stake growth becomes cCNPY yield, so it gets the same shape and
// distinctness checks as the multisig signer set. Empty stays valid — it is the
// default, and it is the safe direction.
func TestValidateParamsStakeOutputAddresses(t *testing.T) {
	good := func() []byte { return addr20(0x50) }
	other := func() []byte { return addr20(0x51) }

	tests := []struct {
		name    string
		addrs   [][]byte
		wantErr bool
	}{
		{"empty is the default and is valid", nil, false},
		{"one well-formed address", [][]byte{good()}, false},
		{"two distinct addresses", [][]byte{good(), other()}, false},
		{"short address", [][]byte{make([]byte, 19)}, true},
		{"long address", [][]byte{make([]byte, 21)}, true},
		{"duplicate address", [][]byte{good(), good()}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParams()
			p.StakeOutputAddresses = tt.addrs
			err := ValidateParams(p)
			if tt.wantErr != (err != nil) {
				t.Errorf("ValidateParams: err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateParamsMaxRewardBps: 10_000 bps is the documented off switch (a
// cap of 100% of owned stake can never bind). Anything above it is the same
// meaning spelled a second way, so it is rejected rather than accepted.
func TestValidateParamsMaxRewardBps(t *testing.T) {
	for _, tc := range []struct {
		bps     uint64
		wantErr bool
	}{{0, false}, {100, false}, {10_000, false}, {10_001, true}} {
		p := DefaultParams()
		p.MaxRewardBpsPerBlock = tc.bps
		if err := ValidateParams(p); tc.wantErr != (err != nil) {
			t.Errorf("bps=%d: err=%v wantErr=%v", tc.bps, err, tc.wantErr)
		}
	}
}
