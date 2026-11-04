package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// reward_attribution_test.go covers which stake canoLiq is allowed to credit
// itself with. The sweep observes block-over-block growth of bonded committee
// stake (reward.go::ProcessRewards); the question this file pins is *whose*
// stake, because Canopy compounds reward into the bond of every committee
// validator, and canoLiq owns at most a few of them.
//
// The answer is Validator.output: a bond contributes to the received reward R
// only when its output address is listed in params.stake_output_addresses.
// Membership on the committee is a separate role and still governs the 15%
// validator-incentive slice — see TestMixedCommitteeCreditsOnlyOwnedGrowth,
// which is the test that holds the two apart.

// liveGlobals is the globals a sweep needs to reach the normal accrual path:
// genesis done, and cCNPY outstanding so the user slice lands in the pool
// rather than taking the ownerless-to-treasury branch.
func liveGlobals(watermark uint64) *contract.CanoliqGlobals {
	return &contract.CanoliqGlobals{
		GenesisComplete:         true,
		TotalCcnpySupply:        testLivePoolCcnpy,
		TotalPooledCnpy:         testLivePoolCcnpy,
		LastProcessedRewardPool: watermark,
	}
}

// sweptState is everything a sweep can credit, read back in one place so a
// test can assert that nothing anywhere moved.
type sweptState struct {
	pooled     uint64
	watermark  uint64
	attributed uint64
	carried    uint64
	treasury   uint64
	insurance  uint64
	buyback    uint64
	validators uint64
}

func readSwept(t *testing.T, s *fakeStore) sweptState {
	t.Helper()
	g := loadGlobals(t, s)
	return sweptState{
		pooled:     g.TotalPooledCnpy,
		watermark:  g.LastProcessedRewardPool,
		attributed: g.LastAttributedReward,
		carried:    g.CarriedReward,
		treasury:   DecodeUint64(s.get(KeyForTreasuryCNPY())),
		insurance:  DecodeUint64(s.get(KeyForInsurancePool())),
		buyback:    DecodeUint64(s.get(KeyForBuybackPool())),
		validators: readAllValidatorIncentives(s),
	}
}

// TestUnownedCommitteeValidatorContributesZeroReward reproduces the mainnet
// committee-29 incident directly.
//
// val-a is a Canopy Foundation validator: on the committee, compounding, and
// growing ~11.90 CNPY per block. canoLiq's pool contributed ~10 CNPY of its
// ~8,235 CNPY bond and its output address is the Foundation's, not canoLiq's.
// The old sweep credited the entire growth to cCNPY holders, taking a 10 CNPY
// deposit to ~562x in forty minutes.
//
// Nothing about that bond may move any canoLiq balance.
func TestUnownedCommitteeValidatorContributesZeroReward(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c) // canoLiq declares an output, but not val-a's

	const (
		valAStake  = 8_235_000_000 // ~8,235 CNPY bonded
		blockGrows = 11_900_000    // ~11.90 CNPY/block, the observed rate
	)
	valA := addr20(0xA1)
	seedGlobals(s, liveGlobals(valAStake))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{{Address: valA, Stake: valAStake}},
	}))
	setCommitteeStakeWithOutput(s, c, valA, valAStake+blockGrows, testForeignOutput)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}

	got := readSwept(t, s)
	if got.pooled != testLivePoolCcnpy {
		t.Errorf("pooled CNPY moved on an unowned bond: got %d want %d (exchange rate must not lift)",
			got.pooled, testLivePoolCcnpy)
	}
	if got.attributed != 0 {
		t.Errorf("attributed reward: got %d want 0", got.attributed)
	}
	if got.treasury != 0 || got.insurance != 0 || got.buyback != 0 {
		t.Errorf("fee split ran on an unowned bond: treasury=%d insurance=%d buyback=%d",
			got.treasury, got.insurance, got.buyback)
	}
	// The watermark tracks owned stake, and canoLiq owns none here.
	if got.watermark != 0 {
		t.Errorf("watermark: got %d want 0 (owned stake, not committee stake)", got.watermark)
	}
	// val-a is still a committee member, so it keeps its registry entry — it is
	// simply not marked owned. Membership and ownership are different roles.
	reg := loadRegistryEntries(t, s)
	if len(reg) != 1 || reg[0].Owned {
		t.Errorf("val-a should stay registered but unowned, got %+v", reg)
	}
}

// TestEmptyStakeOutputSetYieldsZeroReward pins the shipped default. With no
// ownership declared, no bond on the committee is canoLiq's, so a sweep
// attributes nothing however much the committee compounds. This is the state
// every chain is in the moment the fix deploys.
func TestEmptyStakeOutputSetYieldsZeroReward(t *testing.T) {
	c, s := newTestCanoliq()
	// Deliberately no seedStakeOutputParams: DefaultParams ships the set empty.
	val := addr20(0xC0)
	seedGlobals(s, liveGlobals(rewardBaseStake))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: rewardBaseStake}},
	}))
	// Pointed at the address canoLiq *would* own, to prove the set is what
	// gates attribution rather than the address constant.
	setCommitteeStakeWithOutput(s, c, val, rewardBaseStake+1_000_000, testStakeOutput)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}
	if got := readSwept(t, s); got.pooled != testLivePoolCcnpy || got.attributed != 0 {
		t.Errorf("empty ownership set must attribute nothing: pooled=%d attributed=%d",
			got.pooled, got.attributed)
	}
}

// TestMixedCommitteeCreditsOnlyOwnedGrowth is the test that holds the two roles
// apart. One owned bond and one foreign bond both grow. Only the owned growth
// may reach cCNPY holders — but the foreign validator must still draw its
// pro-rata slice of the 15% validator incentive, because that slice pays
// operators for running the committee regardless of whose CNPY is bonded.
func TestMixedCommitteeCreditsOnlyOwnedGrowth(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)

	const (
		ownedBase   = 4_000_000_000
		foreignBase = 6_000_000_000
		ownedGrowth = 2_000_000 // the only reward canoLiq may claim
		alienGrowth = 9_000_000 // must reach nothing
	)
	owned, alien := addr20(0xC0), addr20(0xC1)
	seedGlobals(s, liveGlobals(ownedBase))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{
			{Address: owned, Stake: ownedBase, Owned: true},
			{Address: alien, Stake: foreignBase},
		},
	}))
	setCommitteeStake(s, c, owned, ownedBase+ownedGrowth)
	setCommitteeStakeWithOutput(s, c, alien, foreignBase+alienGrowth, testForeignOutput)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}

	got := readSwept(t, s)
	if got.attributed != ownedGrowth {
		t.Fatalf("attributed reward: got %d want %d (only the owned bond's growth)",
			got.attributed, ownedGrowth)
	}
	// Conservation over the attributed reward alone.
	sum := (got.pooled - testLivePoolCcnpy) + got.treasury + got.insurance + got.buyback + got.validators
	if sum != ownedGrowth {
		t.Errorf("conservation: %d distributed, want %d", sum, ownedGrowth)
	}
	// Watermark follows owned stake only.
	if want := uint64(ownedBase + ownedGrowth); got.watermark != want {
		t.Errorf("watermark: got %d want %d", got.watermark, want)
	}
	// The foreign operator still earns its committee-weighted incentive slice.
	if alienShare := DecodeUint64(s.get(KeyForValidatorIncentives(alien))); alienShare == 0 {
		t.Error("unowned committee member must still share the validator-incentive slice; " +
			"membership and ownership are separate roles")
	}
}

// TestUpgradeTransitionFromCommitteeWatermark is the live-chain transition.
//
// Before the fix, last_processed_reward_pool held aggregate *committee* stake.
// After it, the same field means aggregate *owned* stake, and on a chain with
// nothing declared that is 0. The first block therefore sees a watermark far
// above the observed stake. It must credit nothing and write the watermark
// down, and the block after must be a clean no-op — no windfall, no stall.
func TestUpgradeTransitionFromCommitteeWatermark(t *testing.T) {
	c, s := newTestCanoliq()
	const staleCommitteeWatermark = 8_235_000_000 // what the old binary left

	valA := addr20(0xA1)
	seedGlobals(s, liveGlobals(staleCommitteeWatermark))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{{Address: valA, Stake: staleCommitteeWatermark}},
	}))
	setCommitteeStakeWithOutput(s, c, valA, staleCommitteeWatermark+11_900_000, testForeignOutput)

	// Block 1: the shrink branch absorbs the stale watermark.
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("block 1: %v", err)
	}
	first := readSwept(t, s)
	if first.pooled != testLivePoolCcnpy {
		t.Errorf("block 1 credited a windfall: pooled %d want %d", first.pooled, testLivePoolCcnpy)
	}
	if first.watermark != 0 {
		t.Errorf("block 1 watermark: got %d want 0", first.watermark)
	}

	// Block 2: still nothing owned, still nothing to do, and no stall.
	setCommitteeStakeWithOutput(s, c, valA, staleCommitteeWatermark+23_800_000, testForeignOutput)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 2}); err != nil {
		t.Fatalf("block 2: %v", err)
	}
	if second := readSwept(t, s); second != first {
		t.Errorf("block 2 should be a no-op: got %+v want %+v", second, first)
	}
}

// TestAddingStakeOutputAddressDoesNotCreditExistingBond covers the day
// governance declares an output address for a bond that has been sitting on the
// committee all along. Its whole principal must not read as one block of
// reward; only growth after the flip counts.
func TestAddingStakeOutputAddressDoesNotCreditExistingBond(t *testing.T) {
	c, s := newTestCanoliq()
	const bond = 5_000_000_000

	val := addr20(0xC0)
	seedGlobals(s, liveGlobals(0))
	setCommitteeStake(s, c, val, bond) // output is canoLiq's...

	// ...but the ownership set is still empty, so block 1 observes nothing.
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("block 1: %v", err)
	}
	if got := readSwept(t, s); got.watermark != 0 || got.pooled != testLivePoolCcnpy {
		t.Fatalf("pre-declaration block credited something: %+v", got)
	}

	// Governance declares the address. The bond is already in the registry at
	// its current stake, so its per-validator delta this block is zero.
	seedStakeOutputParams(t, c)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 2}); err != nil {
		t.Fatalf("flip block: %v", err)
	}
	flip := readSwept(t, s)
	if flip.pooled != testLivePoolCcnpy {
		t.Errorf("the flip block credited the pre-existing bond: pooled %d want %d",
			flip.pooled, testLivePoolCcnpy)
	}
	if flip.watermark != bond {
		t.Errorf("flip block watermark: got %d want %d (adopt the bond, credit none of it)",
			flip.watermark, bond)
	}

	// The block after, ordinary growth accrues normally.
	const growth = 1_000_000
	setCommitteeStake(s, c, val, bond+growth)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 3}); err != nil {
		t.Fatalf("growth block: %v", err)
	}
	if got := readSwept(t, s); got.attributed != growth {
		t.Errorf("growth block attributed: got %d want %d", got.attributed, growth)
	}
}

// TestAttributionClampCarriesExcessForward covers the plausibility clamp. The
// excess is carried in the watermark rather than discarded, because a
// nested-delegate lottery win is legitimately lumpy and truncating would
// quietly destroy real user yield.
func TestAttributionClampCarriesExcessForward(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)

	const (
		bond   = 1_000_000_000
		capBps = 100        // 1% of owned stake per block
		spike  = 25_000_000 // 2.5% — 2.5 blocks' worth of cap
	)
	val := addr20(0xC0)
	seedGlobals(s, liveGlobals(bond))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: bond, Owned: true}},
	}))
	setCommitteeStake(s, c, val, bond+spike)

	// Cap is computed against observed (post-growth) owned stake.
	wantFirst := mulDiv(bond+spike, capBps, 10_000)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("block 1: %v", err)
	}
	first := readSwept(t, s)
	if first.attributed != wantFirst {
		t.Fatalf("clamped credit: got %d want %d", first.attributed, wantFirst)
	}
	// The watermark advances fully; the deferred remainder rides in
	// CarriedReward so the watermark keeps meaning exactly one thing.
	if first.watermark != bond+spike {
		t.Errorf("watermark: got %d want %d", first.watermark, bond+spike)
	}
	if wantCarried := spike - wantFirst; first.carried != wantCarried {
		t.Errorf("carried remainder: got %d want %d", first.carried, wantCarried)
	}

	// Drain the remainder over following blocks with no further growth. Nothing
	// may be lost: the total credited must reach the full spike.
	total := first.attributed
	for h := uint64(2); h <= 10; h++ {
		if err := c.ProcessRewards(&contract.PluginEndRequest{Height: h}); err != nil {
			t.Fatalf("block %d: %v", h, err)
		}
		total += loadGlobals(t, s).LastAttributedReward
	}
	if total != spike {
		t.Errorf("carried-forward reward lost value: credited %d of %d", total, spike)
	}
	if carried := loadGlobals(t, s).CarriedReward; carried != 0 {
		t.Errorf("carry should be fully drained: got %d want 0", carried)
	}
}

// TestAttributionClampOffAtFullBps pins the documented off switch. 10_000 bps
// is a cap of 100% of owned stake, which can never bind because the attributed
// reward is itself bounded by that stake's growth. Zero is NOT the off switch:
// backfillParams treats it as "unset" and restores the default.
func TestAttributionClampOffAtFullBps(t *testing.T) {
	c, s := newTestCanoliq()
	seedStakeOutputParams(t, c)
	disableRewardClamp(t, c)

	const (
		bond  = 1_000_000_000
		spike = 25_000_000
	)
	val := addr20(0xC0)
	seedGlobals(s, liveGlobals(bond))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{{Address: val, Stake: bond, Owned: true}},
	}))
	setCommitteeStake(s, c, val, bond+spike)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}
	got := readSwept(t, s)
	if got.attributed != spike {
		t.Errorf("clamp at 10_000 bps must not bind: got %d want %d", got.attributed, spike)
	}
	if got.watermark != bond+spike {
		t.Errorf("watermark: got %d want %d", got.watermark, bond+spike)
	}
}

// TestZeroClampBpsBackfillsToDefault guards the sentinel directly: a params
// record written before max_reward_bps_per_block existed decodes it as zero,
// and zero must not mean "no clamp" — that is the safety net on the very bug
// the field exists for.
func TestZeroClampBpsBackfillsToDefault(t *testing.T) {
	p := DefaultParams()
	p.MaxRewardBpsPerBlock = 0
	backfillParams(p)
	if p.MaxRewardBpsPerBlock != DefaultParams().MaxRewardBpsPerBlock {
		t.Errorf("zero clamp bps must backfill to the default, got %d", p.MaxRewardBpsPerBlock)
	}
}

// loadRegistryEntries reads the persisted registry as entries, so tests can
// assert on the Owned flag as well as the stake.
func loadRegistryEntries(t *testing.T, s *fakeStore) []*contract.ValidatorRegistryEntry {
	t.Helper()
	bz := s.get(KeyForValidatorRegistry())
	if bz == nil {
		return nil
	}
	reg := new(contract.ValidatorRegistry)
	if err := contract.Unmarshal(bz, reg); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	return reg.Entries
}
