package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// reward_observe_test.go covers the reward-observation model: ProcessRewards
// derives canoLiq's received committee reward from the block-over-block growth
// of its bonded committee stake (summed over the ValidatorRegistry), not from
// the committee fee pool (which Canopy drains to committee members every block
// before this hook runs). See reward.go::ProcessRewards.

// TestObserveFirstRunSeedsBaselineNoDistribution: on the very first observation
// (baseline 0) the pre-existing bonded stake must be adopted as the baseline
// and NOT counted as one giant reward — nothing is distributed and the exchange
// rate stays put.
func TestObserveFirstRunSeedsBaselineNoDistribution(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true}) // LastProcessedRewardPool == 0
	// A committee validator already holds a large bonded position.
	val := addr20(0xC0)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: rewardBaseStake}}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	setCommitteeStake(s, c, val, 5_000_000_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g := loadGlobals(t, s)
	if g.TotalPooledCnpy != 0 {
		t.Fatalf("first observation must not distribute: pooled got %d want 0", g.TotalPooledCnpy)
	}
	if g.LastProcessedRewardPool != 5_000_000_000 {
		t.Fatalf("baseline should seed to current observed stake: got %d want 5_000_000_000", g.LastProcessedRewardPool)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != 0 {
		t.Fatalf("no treasury credit on first observation: got %d", got)
	}
}

// TestObserveStakeGrowthDistributesReward: once a baseline exists, a stake
// increase of N is treated as reward N and lifts the exchange rate via the 12%
// fee + 40/30/15/15 split.
func TestObserveStakeGrowthDistributesReward(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: testLivePoolCcnpy})
	// seedReward records rewardBaseStake as the baseline and grows the position
	// by N; ProcessRewards should observe exactly N.
	const N = 1_000_000
	seedReward(t, s, c, N)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g := loadGlobals(t, s)
	// fee=120_000; net=880_000; user rebate=48_000 → user slice 928_000.
	if g.TotalPooledCnpy != 928_000 {
		t.Fatalf("user yield: got %d want 928_000", g.TotalPooledCnpy)
	}
	if g.LastProcessedRewardPool != rewardBaseStake+N {
		t.Fatalf("watermark: got %d want %d", g.LastProcessedRewardPool, rewardBaseStake+N)
	}
	// Conservation: N == userYield + treasury + insurance + buyback + validators.
	total := g.TotalPooledCnpy +
		DecodeUint64(s.get(KeyForTreasuryCNPY())) +
		DecodeUint64(s.get(KeyForInsurancePool())) +
		DecodeUint64(s.get(KeyForBuybackPool())) +
		readAllValidatorIncentives(s)
	if total != N {
		t.Fatalf("conservation: got %d want %d", total, N)
	}
}

// TestObserveStakeDecreaseResetsBaselineNoDistribution: an unstake/slash that
// shrinks the observed position below the baseline yields no reward and resets
// the baseline to the new (lower) level so growth resumes cleanly.
func TestObserveStakeDecreaseResetsBaselineNoDistribution(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	val := addr20(0xC0)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: rewardBaseStake}}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	// Baseline recorded at a high level; the live position has since shrunk.
	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = 2_000_000_000
	s.set(KeyForGlobals(), mustMarshal(g))
	setCommitteeStake(s, c, val, 1_500_000_000) // below baseline

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	if g2.TotalPooledCnpy != 0 {
		t.Fatalf("stake decrease must not distribute: pooled got %d want 0", g2.TotalPooledCnpy)
	}
	if g2.LastProcessedRewardPool != 1_500_000_000 {
		t.Fatalf("baseline should reset to new observed stake: got %d want 1_500_000_000", g2.LastProcessedRewardPool)
	}
}

// TestObserveExcludesValidatorsOffCommittee: registry entries whose live
// Canopy validator is not a member of this committee are excluded from the
// observed stake, so their stake does not inflate the reward delta.
func TestObserveExcludesValidatorsOffCommittee(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: testLivePoolCcnpy})
	onCommittee, offCommittee := addr20(0xC0), addr20(0xC1)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: onCommittee, Stake: rewardBaseStake},
		{Address: offCommittee, Stake: rewardBaseStake},
	}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	// on-committee validator: baseline + growth of N.
	const N = 1_000_000
	setCommitteeStake(s, c, onCommittee, rewardBaseStake+N)
	// off-committee validator: large stake, but bonded to a DIFFERENT committee,
	// so it must not contribute to the observation.
	other := &contract.Validator{Address: offCommittee, StakedAmount: 9_000_000_000, Committees: []uint64{c.Config.ChainId + 1}}
	s.set(contract.KeyForValidator(offCommittee), mustMarshal(other))

	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = rewardBaseStake // only the on-committee validator's baseline
	s.set(KeyForGlobals(), mustMarshal(g))

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	// Only N is observed (off-committee stake excluded) → user slice 928_000.
	if g2.TotalPooledCnpy != 928_000 {
		t.Fatalf("off-committee stake must be excluded: pooled got %d want 928_000", g2.TotalPooledCnpy)
	}
	if g2.LastProcessedRewardPool != rewardBaseStake+N {
		t.Fatalf("watermark should reflect on-committee stake only: got %d want %d", g2.LastProcessedRewardPool, rewardBaseStake+N)
	}
}
