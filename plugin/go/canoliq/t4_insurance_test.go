package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// t4_insurance_test.go covers insurance-fund peak-TVL tracking (T4): the
// per-block insurance skim turns off once the reserve reaches its target
// (insurance_target_bps of peak TVL) and back on as peak TVL grows, with the
// redirected amount conserved into the treasury.

// runSweep sets the committee pool to lastProcessed+delta and runs one reward
// sweep at the given height.
func runSweep(t *testing.T, c *Canoliq, s *fakeStore, lastProcessed, delta, height uint64) {
	t.Helper()
	s.set(contract.KeyForFeePool(c.Config.ChainId), mustMarshal(&contract.Pool{Id: c.Config.ChainId, Amount: lastProcessed + delta}))
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: height}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}
}

func insurancePool(s *fakeStore) uint64 { return DecodeUint64(s.get(KeyForInsurancePool())) }
func treasuryCnpy(s *fakeStore) uint64  { return DecodeUint64(s.get(KeyForTreasuryCNPY())) }

// For delta = 1_000_000 and the default 12% / 30% / 5% params:
//
//	fee = 120_000; treasury slice = 36_000; insurance skim = 1_800;
//	treasury net (skim on) = 34_200; treasury net (skim off) = 36_000.
const (
	t4Delta         uint64 = 1_000_000
	t4TreasurySlice uint64 = 36_000
	t4InsuranceSkim uint64 = 1_800
)

// TestT4SkimActiveBelowTarget: with the reserve below target, the skim runs.
func TestT4SkimActiveBelowTarget(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	// peak 10M → target 500k; reserve starts at 0 (below target).
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, PeakTvlUcnpy: 10_000_000})
	runSweep(t, c, s, 0, t4Delta, 1)
	if got := insurancePool(s); got != t4InsuranceSkim {
		t.Errorf("insurance skim should run below target: got %d want %d", got, t4InsuranceSkim)
	}
	if got := treasuryCnpy(s); got != t4TreasurySlice-t4InsuranceSkim {
		t.Errorf("treasury net (skim on): got %d want %d", got, t4TreasurySlice-t4InsuranceSkim)
	}
}

// TestT4SkimOffAtTargetConserves: at/above target the skim stops and the
// would-be insurance amount is conserved into the treasury.
func TestT4SkimOffAtTargetConserves(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	// peak 10M → target 500k; seed reserve exactly at target.
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, PeakTvlUcnpy: 10_000_000})
	s.set(KeyForInsurancePool(), EncodeUint64(500_000))

	runSweep(t, c, s, 0, t4Delta, 1)

	if got := insurancePool(s); got != 500_000 {
		t.Errorf("insurance should be unchanged at target: got %d want 500_000", got)
	}
	if got := treasuryCnpy(s); got != t4TreasurySlice {
		t.Errorf("treasury should receive the full slice when skim is off: got %d want %d", got, t4TreasurySlice)
	}
	// Conservation: every uCNPY of the delta is accounted for.
	g := loadGlobals(t, s)
	pooledIncrease := g.TotalPooledCnpy // started at 0
	buyback := DecodeUint64(s.get(KeyForBuybackPool()))
	validators := DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr())))
	insuranceIncrease := insurancePool(s) - 500_000 // 0
	total := pooledIncrease + treasuryCnpy(s) + buyback + validators + insuranceIncrease
	if total != t4Delta {
		t.Errorf("conservation: pooled %d + treasury %d + buyback %d + validators %d + insuranceΔ %d = %d, want %d",
			pooledIncrease, treasuryCnpy(s), buyback, validators, insuranceIncrease, total, t4Delta)
	}
}

// TestT4SkimResumesAfterPeakGrows: a reserve at target for the current peak
// resumes skimming once peak TVL grows past the next threshold.
func TestT4SkimResumesAfterPeakGrows(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, PeakTvlUcnpy: 10_000_000})
	s.set(KeyForInsurancePool(), EncodeUint64(500_000)) // == 5% of 10M

	// Sweep 1: reserve at target → skim off.
	runSweep(t, c, s, 0, t4Delta, 1)
	if got := insurancePool(s); got != 500_000 {
		t.Fatalf("skim should be off at target: insurance got %d want 500_000", got)
	}

	// Grow TVL to 20M (simulating deposits); next sweep raises peak → target
	// 1M, reserve 500k now below it → skim resumes.
	g := loadGlobals(t, s)
	g.TotalPooledCnpy = 20_000_000
	seedGlobals(s, g)
	lastProcessed := g.LastProcessedRewardPool
	runSweep(t, c, s, lastProcessed, t4Delta, 2)

	if got := loadGlobals(t, s).PeakTvlUcnpy; got < 20_000_000 {
		t.Errorf("peak should have grown to >= 20M: got %d", got)
	}
	if got := insurancePool(s); got != 500_000+t4InsuranceSkim {
		t.Errorf("skim should resume after peak grows: got %d want %d", got, 500_000+t4InsuranceSkim)
	}
}

// TestT4PeakInitializesFromPoolOnMigration: a pre-T4 node (peak_tvl_ucnpy = 0)
// seeds the high water mark from the current pool on the first sweep, even
// when there is no fresh reward delta.
func TestT4PeakInitializesFromPoolOnMigration(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete: true, TotalPooledCnpy: 5_000_000,
		PeakTvlUcnpy: 0, LastProcessedRewardPool: 1_000,
	})
	// No fresh delta: pool == lastProcessed.
	runSweep(t, c, s, 0, 1_000, 1)
	if got := loadGlobals(t, s).PeakTvlUcnpy; got != 5_000_000 {
		t.Errorf("peak should initialize from current pool: got %d want 5_000_000", got)
	}
}
