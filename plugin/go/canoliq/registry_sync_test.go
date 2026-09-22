package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// registry_sync_test.go covers the per-block reconciliation of the
// ValidatorRegistry against Canopy's live committee membership
// (registry.go::syncCommitteeRegistry, driven from ProcessRewards).

// loadRegistry reads the persisted ValidatorRegistry as a map of
// address → recorded stake.
func loadRegistry(t *testing.T, s *fakeStore) map[string]uint64 {
	t.Helper()
	bz := s.get(KeyForValidatorRegistry())
	out := make(map[string]uint64)
	if bz == nil {
		return out
	}
	reg := new(contract.ValidatorRegistry)
	if err := contract.Unmarshal(bz, reg); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	for _, e := range reg.Entries {
		out[string(e.Address)] = e.Stake
	}
	return out
}

// TestSyncAdmitsPostGenesisValidator is the regression test for the reported
// bug: a validator that stakes to this committee *after* genesis was never
// added to the registry, so ProcessRewards observed nothing and the
// cCNPY/CNPY exchange rate stayed pinned at 1.0 no matter how much stake
// compounded. The first sweep must adopt the validator (distributing
// nothing), and the next sweep must distribute its compounded growth.
func TestSyncAdmitsPostGenesisValidator(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: testLivePoolCcnpy})
	// No registry at all — genesis carried no validatorRegistry block.
	val := addr20(0xC0)
	setCommitteeStake(s, c, val, 1_000_000_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	reg := loadRegistry(t, s)
	if got, ok := reg[string(val)]; !ok || got != 1_000_000_000 {
		t.Fatalf("post-genesis validator must be admitted at its live stake: got %d present=%t", got, ok)
	}
	g := loadGlobals(t, s)
	if g.TotalPooledCnpy != 0 {
		t.Fatalf("admission must not distribute: pooled got %d want 0", g.TotalPooledCnpy)
	}
	if g.LastProcessedRewardPool != 1_000_000_000 {
		t.Fatalf("watermark: got %d want 1_000_000_000", g.LastProcessedRewardPool)
	}

	// Canopy compounds 1_000_000 of committee reward into the bonded position.
	setCommitteeStake(s, c, val, 1_000_000_000+1_000_000)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 2}); err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	g2 := loadGlobals(t, s)
	// fee=120_000; net=880_000; user rebate=48_000 → user slice 928_000.
	if g2.TotalPooledCnpy != 928_000 {
		t.Fatalf("compounded reward must reach cCNPY holders: pooled got %d want 928_000", g2.TotalPooledCnpy)
	}
	if reg2 := loadRegistry(t, s); reg2[string(val)] != 1_001_000_000 {
		t.Fatalf("registry stake must re-baseline to live: got %d", reg2[string(val)])
	}
}

// TestSyncNewMemberBondIsNotReward guards the hazard the sync introduces: a
// validator joining the committee lifts the aggregate committee stake by its
// entire bond, which an aggregate-only diff would mint to cCNPY holders as
// yield. Only the already-registered member's growth counts.
func TestSyncNewMemberBondIsNotReward(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: testLivePoolCcnpy})
	v1, v2 := addr20(0xC0), addr20(0xC1)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: v1, Stake: rewardBaseStake}}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = rewardBaseStake
	s.set(KeyForGlobals(), mustMarshal(g))
	// v1 compounds 1_000_000 of reward; v2 joins the committee with a huge bond.
	setCommitteeStake(s, c, v1, rewardBaseStake+1_000_000)
	setCommitteeStake(s, c, v2, 5_000_000_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	if g2.TotalPooledCnpy != 928_000 {
		t.Fatalf("newcomer's bond must not read as reward: pooled got %d want 928_000", g2.TotalPooledCnpy)
	}
	if g2.LastProcessedRewardPool != rewardBaseStake+1_000_000+5_000_000_000 {
		t.Fatalf("watermark must cover the whole member set: got %d", g2.LastProcessedRewardPool)
	}
	if got := loadRegistry(t, s)[string(v2)]; got != 5_000_000_000 {
		t.Fatalf("newcomer must be baselined at its live stake: got %d", got)
	}
}

// TestSyncStaleBaselineCappedByAggregate covers the upgrade path: a registry
// seeded by hand at genesis carries share-out weights that need not match real
// bonded stake, so the first per-validator diff after this code ships can be
// wildly overstated. The aggregate growth (measured against the watermark,
// which was always summed from live stake) caps it.
func TestSyncStaleBaselineCappedByAggregate(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: testLivePoolCcnpy})
	val := addr20(0xC0)
	// Genesis weight of 1 vs. a live bond of rewardBaseStake.
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: 1}}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = rewardBaseStake // accurate: summed from live stake
	s.set(KeyForGlobals(), mustMarshal(g))
	setCommitteeStake(s, c, val, rewardBaseStake+1_000_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	if g2.TotalPooledCnpy != 928_000 {
		t.Fatalf("stale genesis weight must not inflate the reward: pooled got %d want 928_000", g2.TotalPooledCnpy)
	}
}

// TestSyncPerValidatorShrinkIsNotReward: a member whose stake shrank
// (unstake / slash) contributes no reward and is re-baselined at the lower
// level, so the next block's growth is measured from there.
func TestSyncPerValidatorShrinkIsNotReward(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	val := addr20(0xC0)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: 2_000_000_000}}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = 2_000_000_000
	s.set(KeyForGlobals(), mustMarshal(g))
	setCommitteeStake(s, c, val, 1_500_000_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	if g2.TotalPooledCnpy != 0 {
		t.Fatalf("shrink must not distribute: pooled got %d want 0", g2.TotalPooledCnpy)
	}
	if g2.LastProcessedRewardPool != 1_500_000_000 {
		t.Fatalf("watermark should reset: got %d want 1_500_000_000", g2.LastProcessedRewardPool)
	}
	if got := loadRegistry(t, s)[string(val)]; got != 1_500_000_000 {
		t.Fatalf("entry should re-baseline to the lower stake: got %d", got)
	}
}

// TestSyncDropsDepartedValidator: a validator that leaves committee `chainId`
// (edit-stake dropping it from Committees[]) is dropped from the registry, so
// it stops receiving a share of the validator-incentive slice.
func TestSyncDropsDepartedValidator(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	stays, leaves := addr20(0xC0), addr20(0xC1)
	reg := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: stays, Stake: rewardBaseStake},
		{Address: leaves, Stake: rewardBaseStake},
	}}
	s.set(KeyForValidatorRegistry(), mustMarshal(reg))
	setCommitteeStake(s, c, stays, rewardBaseStake)
	// `leaves` is still a live Canopy validator, but on a different committee.
	s.set(contract.KeyForValidator(leaves), mustMarshal(&contract.Validator{
		Address: leaves, StakedAmount: rewardBaseStake, Committees: []uint64{c.Config.ChainId + 1},
	}))
	g := loadGlobals(t, s)
	g.LastProcessedRewardPool = 2 * rewardBaseStake
	s.set(KeyForGlobals(), mustMarshal(g))

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	after := loadRegistry(t, s)
	if _, ok := after[string(leaves)]; ok {
		t.Fatalf("validator off the committee must be dropped from the registry")
	}
	if _, ok := after[string(stays)]; !ok {
		t.Fatalf("committee member must stay registered")
	}
}

// TestSyncKeepsEjectedValidatorOut: the registry is rebuilt from live
// membership every block, so a governance ejection (F12) only sticks because
// of the tombstone ejectValidator writes. Without it the very next sweep
// would re-admit the ejected operator.
func TestSyncKeepsEjectedValidatorOut(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	v1, v2 := addr20(0xC0), addr20(0xC1)
	setCommitteeStake(s, c, v1, rewardBaseStake)
	setCommitteeStake(s, c, v2, rewardBaseStake)

	if err := c.ejectValidator(v1); err != nil {
		t.Fatalf("eject: %v", err)
	}
	// v1 is still a bonded, committee-listed Canopy validator.
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}
	after := loadRegistry(t, s)
	if _, ok := after[string(v1)]; ok {
		t.Fatalf("ejected validator must not be re-admitted by the sync")
	}
	if _, ok := after[string(v2)]; !ok {
		t.Fatalf("non-ejected committee member must be registered")
	}
	// Its stake is also out of the observation, so the watermark covers v2 only.
	if got := loadGlobals(t, s).LastProcessedRewardPool; got != rewardBaseStake {
		t.Fatalf("ejected stake must be out of the observation: watermark got %d want %d", got, rewardBaseStake)
	}
}
