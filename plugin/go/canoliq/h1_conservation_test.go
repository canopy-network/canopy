package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// physicalCnpy sums every key that actually holds CNPY in the plugin's books:
// the listed user accounts, the escrow pool, the committee fee pool, and the
// treasury / insurance / buyback / validator-incentive accumulators. cCNPY and
// CPLQ are excluded (they are not CNPY). globals.TotalPooledCnpy is an
// accounting figure, not a holding, so it is excluded here and checked
// separately via the escrow invariant.
func physicalCnpy(s *fakeStore, c *Canoliq, users ...[]byte) uint64 {
	total := uint64(0)
	for _, u := range users {
		total += readAccount(s, u)
	}
	total += readEscrow(s)
	total += readPool(s, c.Config.ChainId) // committee fee pool
	total += DecodeUint64(s.get(KeyForTreasuryCNPY()))
	total += DecodeUint64(s.get(KeyForBuybackPool()))
	total += DecodeUint64(s.get(KeyForInsurancePool()))
	total += readAllValidatorIncentives(s)
	return total
}

// assertEscrowInvariant pins the H1 backing invariant:
// escrow.Amount == TotalPooledCnpy + PendingRedemptionCnpy.
func assertEscrowInvariant(t *testing.T, s *fakeStore) {
	t.Helper()
	g := loadGlobals(t, s)
	want := g.TotalPooledCnpy + g.PendingRedemptionCnpy
	if got := readEscrow(s); got != want {
		t.Fatalf("escrow invariant broken: escrow=%d, want TotalPooled(%d)+Pending(%d)=%d",
			got, g.TotalPooledCnpy, g.PendingRedemptionCnpy, want)
	}
}

// TestH1CnpyConservationLifecycle is the H1 regression guard: CNPY is conserved
// across deposit -> reward -> redeem -> claim. The only balance that may change
// the physical CNPY total is an external reward inflow; every internal step
// (deposit, redeem, claim) must merely relocate CNPY between keys. The escrow
// invariant is re-checked after each step.
//
// Before the H1 fix, deposit debited the user but credited no pool (the
// principal vanished) and claim credited CNPY with no backing debit, so the
// physical total silently diverged. This test fails closed against any
// regression of that bug.
func TestH1CnpyConservationLifecycle(t *testing.T) {
	c, s := newTestCanoliq()
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)

	user := addr20(0x42)
	const initial = 10_000_000
	seedAccount(s, user, initial)

	// Baseline: all CNPY is in the user account.
	if got := physicalCnpy(s, c, user); got != initial {
		t.Fatalf("baseline physical CNPY: got %d want %d", got, initial)
	}
	assertEscrowInvariant(t, s)

	const fee = 10_000

	// 1) Deposit 1 CNPY. Conserved: principal -> escrow, fee -> committee pool.
	const depAmt = 1_000_000
	if resp := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: depAmt}, fee, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("deposit: %v", resp.Error)
	}
	if got := physicalCnpy(s, c, user); got != initial {
		t.Fatalf("post-deposit physical CNPY: got %d want %d", got, initial)
	}
	assertEscrowInvariant(t, s)

	// 2) Reward: committee reward R compounds into the bonded committee stake,
	// observed by ProcessRewards. Physical total rises by exactly R (mirrored
	// across the plugin's accounting keys); the user slice lands in escrow.
	const reward = 500_000
	seedReward(t, s, c, reward)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("rewards: %v", err)
	}
	if got := physicalCnpy(s, c, user); got != initial+reward {
		t.Fatalf("post-reward physical CNPY: got %d want %d", got, initial+reward)
	}
	assertEscrowInvariant(t, s)

	// 3) Redeem all cCNPY. Conserved: only the redeem fee moves; escrow unchanged
	// (the owed CNPY shifts from TotalPooled to PendingRedemption).
	ccnpy := readCcnpy(s, user)
	if resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: ccnpy}, fee, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}
	if got := physicalCnpy(s, c, user); got != initial+reward {
		t.Fatalf("post-redeem physical CNPY: got %d want %d", got, initial+reward)
	}
	assertEscrowInvariant(t, s)

	// 4) Claim after maturity. Conserved: escrow -> user (minus claim fee to pool).
	c.plugin.setHeight(1_000_000)
	if resp := c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 0}, fee, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}
	if got := physicalCnpy(s, c, user); got != initial+reward {
		t.Fatalf("post-claim physical CNPY: got %d want %d", got, initial+reward)
	}
	assertEscrowInvariant(t, s)

	// All cCNPY redeemed and claimed: pending must be fully cleared. A few uCNPY
	// of M5 virtual-offset dust may remain in escrow by design (the offset never
	// lets the pool drain to a manipulable empty state); the escrow invariant
	// above already confirms escrow == TotalPooled + Pending throughout.
	gFinal := loadGlobals(t, s)
	if gFinal.PendingRedemptionCnpy != 0 {
		t.Fatalf("pending redemption after full claim: got %d want 0", gFinal.PendingRedemptionCnpy)
	}
	if dust := readEscrow(s); dust > 2 {
		t.Fatalf("escrow after full claim: got %d, want <= 2 (virtual-offset dust)", dust)
	}
}
