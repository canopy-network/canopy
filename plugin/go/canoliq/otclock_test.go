package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// otclock_test.go covers the OTC lock program: lock cCNPY for a tier duration,
// claim the reserved CPLQ at maturity, or cancel early and forfeit it.
//
// The invariants under test, in priority order:
//
//  1. Budget conservation. available + reserved is constant across every
//     lock/claim/cancel, except when a claim pays out (which moves CPLQ out of
//     reserved and into a user balance) or a fund proposal adds to available.
//  2. A matured position can always be paid, because the reward was reserved
//     at lock time and a lock that could not be reserved was rejected.
//  3. Locking does not move globals.TotalCcnpySupply, so the pool exchange
//     rate is unaffected and a locked position keeps appreciating.

// seedCcnpy stores a cCNPY balance at the per-address balance key.
func seedCcnpy(s *fakeStore, addr []byte, amount uint64) {
	s.set(KeyForCCNPYBalance(addr), EncodeUint64(amount))
}

// seedOTCBudget funds the program's unreserved budget directly, standing in
// for a passed ProposalOTCProgramFund.
func seedOTCBudget(s *fakeStore, available uint64) {
	s.set(KeyForOTCBudgetAvailable(), EncodeUint64(available))
}

func readScalarKey(s *fakeStore, key []byte) uint64 {
	return DecodeUint64(s.get(key))
}

func loadOTCLock(t *testing.T, s *fakeStore, addr []byte, id uint64) *contract.OTCLock {
	t.Helper()
	bz := s.get(KeyForOTCLock(addr, id))
	if bz == nil {
		return nil
	}
	lock := new(contract.OTCLock)
	if err := contract.Unmarshal(bz, lock); err != nil {
		t.Fatalf("unmarshal lock: %v", err)
	}
	return lock
}

// otcFixture builds a chain with params, a funded locker, and a funded budget.
func otcFixture(t *testing.T, ccnpy, budget uint64) (*Canoliq, *fakeStore, []byte, *contract.CanoliqParams) {
	t.Helper()
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, TotalCcnpySupply: ccnpy})
	user := addr20(0xA1)
	seedAccount(s, user, 1_000_000)
	seedCcnpy(s, user, ccnpy)
	seedOTCBudget(s, budget)
	c.plugin.setHeight(100)
	return c, s, user, params
}

const (
	testLockAmount = 50_000_000_000  // 50,000 cCNPY, the default minimum
	testBudget     = 500_000_000_000 // 500,000 CPLQ
	testFee        = 10_000
)

// TestOTCRewardMathBothTiers pins the 1:1 quantity conversion at both tiers.
func TestOTCRewardMathBothTiers(t *testing.T) {
	p := DefaultParams()
	if got := otcReward(testLockAmount, otcTierBps(contract.OTCLockTier_OTC_LOCK_90D, p)); got != 2_500_000_000 {
		t.Errorf("90d reward = %d, want 2_500_000_000 (2,500 CPLQ)", got)
	}
	if got := otcReward(testLockAmount, otcTierBps(contract.OTCLockTier_OTC_LOCK_120D, p)); got != 4_000_000_000 {
		t.Errorf("120d reward = %d, want 4_000_000_000 (4,000 CPLQ)", got)
	}
	// Flooring must favor the budget: never round a reward up.
	if got := otcReward(1, 500); got != 0 {
		t.Errorf("dust reward = %d, want 0 (floored)", got)
	}
}

// TestOTCLockDurations pins both tier durations in blocks at the defaults.
func TestOTCLockDurations(t *testing.T) {
	p := DefaultParams()
	if got := otcLockDurationBlocks(contract.OTCLockTier_OTC_LOCK_90D, p); got != 1_296_000 {
		t.Errorf("90d = %d blocks, want 1_296_000", got)
	}
	if got := otcLockDurationBlocks(contract.OTCLockTier_OTC_LOCK_120D, p); got != 1_728_000 {
		t.Errorf("120d = %d blocks, want 1_728_000", got)
	}
	if got := otcLockDurationBlocks(contract.OTCLockTier_OTC_LOCK_UNSPECIFIED, p); got != 0 {
		t.Errorf("unspecified = %d, want 0", got)
	}
	// Nil params must not panic — the helper is called from a handler that
	// always has params, but a zero value should degrade to "unknown tier".
	if got := otcLockDurationBlocks(contract.OTCLockTier_OTC_LOCK_90D, nil); got != 0 {
		t.Errorf("nil params = %d, want 0", got)
	}
	// Terms are governance-tunable: a changed param moves the duration.
	p.OtcTier90Blocks = 45 * blocksPerDay
	if got := otcLockDurationBlocks(contract.OTCLockTier_OTC_LOCK_90D, p); got != 45*blocksPerDay {
		t.Errorf("tuned 90d tier = %d blocks, want %d", got, 45*blocksPerDay)
	}
	if validOTCLockTier(contract.OTCLockTier_OTC_LOCK_UNSPECIFIED) {
		t.Error("OTC_LOCK_UNSPECIFIED must not validate: a zero-valued field is not a choice")
	}
	if validOTCLockTier(contract.OTCLockTier(99)) {
		t.Error("unknown tier must not validate")
	}
}

// TestOTCLockCreateHappyPath verifies the full create-side state transition.
func TestOTCLockCreateHappyPath(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user,
		CcnpyAmount: testLockAmount,
		Tier:        contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params)
	if resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	const wantReward = 2_500_000_000
	// cCNPY left the spendable balance entirely.
	if got := readCcnpy(s, user); got != 0 {
		t.Errorf("ccnpy balance = %d, want 0 (moved into the position)", got)
	}
	lock := loadOTCLock(t, s, user, 1)
	if lock == nil {
		t.Fatal("lock record not written")
	}
	if lock.CcnpyAmount != testLockAmount || lock.RewardCplq != wantReward {
		t.Errorf("lock = (amount %d, reward %d), want (%d, %d)",
			lock.CcnpyAmount, lock.RewardCplq, uint64(testLockAmount), uint64(wantReward))
	}
	if lock.StartHeight != 100 || lock.MatureHeight != 100+1_296_000 {
		t.Errorf("lock heights = (%d, %d), want (100, %d)", lock.StartHeight, lock.MatureHeight, 100+1_296_000)
	}
	// Budget moved from available to reserved, conserving the total.
	avail := readScalarKey(s, KeyForOTCBudgetAvailable())
	resv := readScalarKey(s, KeyForOTCBudgetReserved())
	if avail != testBudget-wantReward || resv != wantReward {
		t.Errorf("budget = (avail %d, resv %d), want (%d, %d)", avail, resv, testBudget-wantReward, wantReward)
	}
	if avail+resv != testBudget {
		t.Errorf("budget not conserved: %d + %d != %d", avail, resv, uint64(testBudget))
	}
	// The exchange rate must be untouched: the cCNPY changed custody, not
	// existence. If this ever fails, every locked position silently stops
	// earning and every unlocked holder gets a windfall.
	if g := loadGlobals(t, s); g.TotalCcnpySupply != testLockAmount {
		t.Errorf("TotalCcnpySupply = %d, want %d (a lock must not burn cCNPY)", g.TotalCcnpySupply, uint64(testLockAmount))
	}
	// Fee was moved into the committee pool, not merely accrued.
	if got := readPool(s, c.Config.ChainId); got != testFee {
		t.Errorf("fee pool = %d, want %d", got, uint64(testFee))
	}
}

// TestOTCLockCreateRejections covers every create-side guard.
func TestOTCLockCreateRejections(t *testing.T) {
	tests := []struct {
		name     string
		amount   uint64
		tier     contract.OTCLockTier
		ccnpy    uint64
		budget   uint64
		wantCode uint64
	}{
		{"below minimum", testLockAmount - 1, contract.OTCLockTier_OTC_LOCK_90D, testLockAmount, testBudget, codeOTCLockBelowMinimum},
		{"invalid tier", testLockAmount, contract.OTCLockTier_OTC_LOCK_UNSPECIFIED, testLockAmount, testBudget, codeInvalidOTCLockTier},
		{"insufficient ccnpy", testLockAmount, contract.OTCLockTier_OTC_LOCK_90D, testLockAmount - 1, testBudget, codeInsufficientCCNPY},
		{"budget exhausted", testLockAmount, contract.OTCLockTier_OTC_LOCK_90D, testLockAmount, 2_499_999_999, codeOTCBudgetExhausted},
		{"budget empty", testLockAmount, contract.OTCLockTier_OTC_LOCK_90D, testLockAmount, 0, codeOTCBudgetExhausted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, s, user, params := otcFixture(t, tc.ccnpy, tc.budget)
			resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
				FromAddress: user,
				CcnpyAmount: tc.amount,
				Tier:        tc.tier,
			}, testFee, params)
			if resp.Error == nil {
				t.Fatal("expected rejection, got success")
			}
			if resp.Error.Code != tc.wantCode {
				t.Errorf("code = %d, want %d (%s)", resp.Error.Code, tc.wantCode, resp.Error.Msg)
			}
			// A rejected lock must leave no trace.
			if lock := loadOTCLock(t, s, user, 1); lock != nil {
				t.Error("rejected lock wrote a position record")
			}
			if resv := readScalarKey(s, KeyForOTCBudgetReserved()); resv != 0 {
				t.Errorf("rejected lock reserved %d", resv)
			}
			if got := readCcnpy(s, user); got != tc.ccnpy {
				t.Errorf("ccnpy = %d, want %d unchanged", got, tc.ccnpy)
			}
		})
	}
}

// TestOTCLockClaimAtMaturity walks create then claim, asserting the payout and
// the circulating-supply bump the L4 invariant requires.
func TestOTCLockClaimAtMaturity(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	const wantReward = 2_500_000_000
	circBefore := loadGlobals(t, s).CplqCirculatingSupply

	// Before maturity the claim must fail, and must not mutate anything.
	c.plugin.setHeight(100 + 1_296_000 - 1)
	resp := c.DeliverMessageOTCLockClaim(&contract.MessageOTCLockClaim{FromAddress: user, LockId: 1}, testFee, params)
	if resp.Error == nil || resp.Error.Code != codeOTCLockNotMature {
		t.Fatalf("premature claim: got %v, want codeOTCLockNotMature", resp.Error)
	}
	if loadOTCLock(t, s, user, 1) == nil {
		t.Fatal("premature claim destroyed the position")
	}

	c.plugin.setHeight(100 + 1_296_000)
	if resp := c.DeliverMessageOTCLockClaim(&contract.MessageOTCLockClaim{FromAddress: user, LockId: 1}, testFee, params); resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != testLockAmount {
		t.Errorf("ccnpy returned = %d, want %d intact", got, uint64(testLockAmount))
	}
	if got := readCplq(s, user); got != wantReward {
		t.Errorf("cplq paid = %d, want %d", got, uint64(wantReward))
	}
	if resv := readScalarKey(s, KeyForOTCBudgetReserved()); resv != 0 {
		t.Errorf("reserved = %d, want 0 after payout", resv)
	}
	// A claim spends the reservation permanently; it does not return to
	// available. Only a cancel does that.
	if avail := readScalarKey(s, KeyForOTCBudgetAvailable()); avail != testBudget-wantReward {
		t.Errorf("available = %d, want %d (a claim must not refund)", avail, testBudget-wantReward)
	}
	if got := loadGlobals(t, s).CplqCirculatingSupply; got != circBefore+wantReward {
		t.Errorf("circulating = %d, want %d (reward left a non-circulating reserve)", got, circBefore+wantReward)
	}
	// Position and index are gone.
	if loadOTCLock(t, s, user, 1) != nil {
		t.Error("claimed position still present")
	}
	if s.get(KeyForOTCLockIndex(user)) != nil {
		t.Error("index key should be deleted when the last position closes")
	}
	// Double claim must fail.
	if resp := c.DeliverMessageOTCLockClaim(&contract.MessageOTCLockClaim{FromAddress: user, LockId: 1}, testFee, params); resp.Error == nil || resp.Error.Code != codeOTCLockNotFound {
		t.Errorf("double claim: got %v, want codeOTCLockNotFound", resp.Error)
	}
}

// TestOTCLockCancelForfeitsReward verifies early exit returns the cCNPY,
// forfeits the CPLQ, and returns the reservation to the unreserved budget.
func TestOTCLockCancelForfeitsReward(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_120D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	circBefore := loadGlobals(t, s).CplqCirculatingSupply

	c.plugin.setHeight(100 + 1_000)
	if resp := c.DeliverMessageOTCLockCancel(&contract.MessageOTCLockCancel{FromAddress: user, LockId: 1}, testFee, params); resp.Error != nil {
		t.Fatalf("cancel: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != testLockAmount {
		t.Errorf("ccnpy returned = %d, want %d intact", got, uint64(testLockAmount))
	}
	if got := readCplq(s, user); got != 0 {
		t.Errorf("cplq paid = %d, want 0 (reward is forfeited)", got)
	}
	// The whole budget is whole again: the forfeited reward returns to
	// available, so the program can fund someone else with it.
	avail := readScalarKey(s, KeyForOTCBudgetAvailable())
	resv := readScalarKey(s, KeyForOTCBudgetReserved())
	if avail != testBudget || resv != 0 {
		t.Errorf("budget after cancel = (avail %d, resv %d), want (%d, 0)", avail, resv, uint64(testBudget))
	}
	if got := loadGlobals(t, s).CplqCirculatingSupply; got != circBefore {
		t.Errorf("circulating = %d, want %d unchanged (a forfeited reward never circulates)", got, circBefore)
	}
}

// TestOTCLockCancelAfterMaturityRefused guards a matured position against a
// mistimed cancel destroying an already-earned reward.
func TestOTCLockCancelAfterMaturityRefused(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	c.plugin.setHeight(100 + 1_296_000)
	resp := c.DeliverMessageOTCLockCancel(&contract.MessageOTCLockCancel{FromAddress: user, LockId: 1}, testFee, params)
	if resp.Error == nil || resp.Error.Code != codeOTCLockMatured {
		t.Fatalf("cancel at maturity: got %v, want codeOTCLockMatured", resp.Error)
	}
	if loadOTCLock(t, s, user, 1) == nil {
		t.Error("refused cancel destroyed the position")
	}
}

// TestOTCLockedCcnpyIsNotRedeemable is the substantive lock check: the locked
// portion must be unreachable through the ordinary redeem path.
func TestOTCLockedCcnpyIsNotRedeemable(t *testing.T) {
	const total = testLockAmount * 2
	c, s, user, params := otcFixture(t, total, testBudget)
	// Back the pool so redeem has CNPY to pay out.
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete: true, TotalCcnpySupply: total, TotalPooledCnpy: total,
	})
	seedEscrow(s, total)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	// Half remains free and must still redeem.
	if resp := c.DeliverMessageCanoliqRedeem(&contract.MessageCanoliqRedeem{
		FromAddress: user, CcnpyAmount: testLockAmount,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("redeeming the free half should succeed: %v", resp.Error)
	}
	// Nothing free is left, so a further redeem of the locked amount fails.
	resp := c.DeliverMessageCanoliqRedeem(&contract.MessageCanoliqRedeem{
		FromAddress: user, CcnpyAmount: testLockAmount,
	}, testFee, params)
	if resp.Error == nil || resp.Error.Code != codeInsufficientCCNPY {
		t.Fatalf("redeeming locked cCNPY: got %v, want codeInsufficientCCNPY", resp.Error)
	}
}

// TestOTCLockMultiplePositionsIndex checks index bookkeeping across several
// positions, including that closing one leaves the others intact.
func TestOTCLockMultiplePositionsIndex(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount*3, testBudget)
	for i := 0; i < 3; i++ {
		if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
			FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
		}, testFee, params); resp.Error != nil {
			t.Fatalf("create %d: %v", i, resp.Error)
		}
	}
	idx := new(contract.OTCLockIndex)
	if err := contract.Unmarshal(s.get(KeyForOTCLockIndex(user)), idx); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}
	if len(idx.Ids) != 3 || idx.Ids[0] != 1 || idx.Ids[2] != 3 {
		t.Fatalf("index ids = %v, want [1 2 3]", idx.Ids)
	}
	// Cancel the middle one; the other two survive and the index shrinks.
	c.plugin.setHeight(100 + 10)
	if resp := c.DeliverMessageOTCLockCancel(&contract.MessageOTCLockCancel{FromAddress: user, LockId: 2}, testFee, params); resp.Error != nil {
		t.Fatalf("cancel: %v", resp.Error)
	}
	idx = new(contract.OTCLockIndex)
	if err := contract.Unmarshal(s.get(KeyForOTCLockIndex(user)), idx); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}
	if len(idx.Ids) != 2 || idx.Ids[0] != 1 || idx.Ids[1] != 3 {
		t.Errorf("index ids = %v, want [1 3]", idx.Ids)
	}
	if loadOTCLock(t, s, user, 1) == nil || loadOTCLock(t, s, user, 3) == nil {
		t.Error("cancelling one position disturbed the others")
	}
	// Budget still balances across the two open positions.
	avail := readScalarKey(s, KeyForOTCBudgetAvailable())
	resv := readScalarKey(s, KeyForOTCBudgetReserved())
	if avail+resv != testBudget {
		t.Errorf("budget not conserved: %d + %d != %d", avail, resv, uint64(testBudget))
	}
	if resv != 2*2_500_000_000 {
		t.Errorf("reserved = %d, want two positions' worth", resv)
	}
}

// TestOTCBudgetCapIsHard drains the budget and confirms the program stops
// accepting locks rather than promising CPLQ it cannot pay.
func TestOTCBudgetCapIsHard(t *testing.T) {
	// Budget covers exactly two minimum 90d positions.
	const budget = 2 * 2_500_000_000
	c, s, user, params := otcFixture(t, testLockAmount*3, budget)
	for i := 0; i < 2; i++ {
		if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
			FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
		}, testFee, params); resp.Error != nil {
			t.Fatalf("create %d: %v", i, resp.Error)
		}
	}
	if avail := readScalarKey(s, KeyForOTCBudgetAvailable()); avail != 0 {
		t.Fatalf("available = %d, want 0 after draining", avail)
	}
	resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params)
	if resp.Error == nil || resp.Error.Code != codeOTCBudgetExhausted {
		t.Fatalf("lock past the cap: got %v, want codeOTCBudgetExhausted", resp.Error)
	}
	// Every reserved uCPLQ is still claimable: that is the point of the cap.
	if resv := readScalarKey(s, KeyForOTCBudgetReserved()); resv != budget {
		t.Errorf("reserved = %d, want the whole budget %d", resv, uint64(budget))
	}
}

// TestFundOTCProgramMovesTreasuryCplq covers the governance funding path and
// the conservation between treasury and program budget.
func TestFundOTCProgramMovesTreasuryCplq(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, CplqTotalSupply: CPLQTotalSupply})
	s.set(KeyForTreasuryCPLQ(), EncodeUint64(1_000_000_000_000)) // 1M CPLQ
	c.plugin.setHeight(10)

	if err := c.fundOTCProgram(&contract.ProposalOTCProgramFund{Amount: testBudget}); err != nil {
		t.Fatalf("fund: %v", err)
	}
	if got := readScalarKey(s, KeyForTreasuryCPLQ()); got != 1_000_000_000_000-testBudget {
		t.Errorf("treasury = %d, want %d", got, uint64(1_000_000_000_000-testBudget))
	}
	if got := readScalarKey(s, KeyForOTCBudgetAvailable()); got != testBudget {
		t.Errorf("program budget = %d, want %d", got, uint64(testBudget))
	}
	// Funding moves between two non-circulating reserves, so nothing enters
	// circulation and nothing mints.
	g := loadGlobals(t, s)
	if g.CplqCirculatingSupply != 0 {
		t.Errorf("circulating = %d, want 0 (funding is a reserve-to-reserve move)", g.CplqCirculatingSupply)
	}
	if g.CplqTotalSupply != CPLQTotalSupply {
		t.Errorf("total supply = %d, want %d unchanged", g.CplqTotalSupply, CPLQTotalSupply)
	}
	// Over-funding clamps to whatever the treasury still holds rather than
	// erroring. This runs from dispatchPassed inside BeginBlock, where a
	// returned error aborts the block and then repeats forever, so a
	// state-dependent rejection here is a chain halt.
	remaining := readScalarKey(s, KeyForTreasuryCPLQ())
	if err := c.fundOTCProgram(&contract.ProposalOTCProgramFund{Amount: 1_000_000_000_000}); err != nil {
		t.Fatalf("over-fund should clamp, not fail: %v", err)
	}
	if got := readScalarKey(s, KeyForTreasuryCPLQ()); got != 0 {
		t.Errorf("treasury after clamped over-fund = %d, want 0 (fully drained)", got)
	}
	if got := readScalarKey(s, KeyForOTCBudgetAvailable()); got != testBudget+remaining {
		t.Errorf("program budget = %d, want %d (first fund plus the clamped remainder)", got, uint64(testBudget)+remaining)
	}
	// An empty treasury makes funding a no-op, still without an error.
	if err := c.fundOTCProgram(&contract.ProposalOTCProgramFund{Amount: 500}); err != nil {
		t.Fatalf("fund against an empty treasury should no-op, not fail: %v", err)
	}
	if got := readScalarKey(s, KeyForOTCBudgetAvailable()); got != testBudget+remaining {
		t.Errorf("budget moved on an empty-treasury fund: got %d", got)
	}
	// A zero-amount payload is still malformed, which is a payload error
	// rather than a state-dependent one, so it stays an error.
	if err := c.fundOTCProgram(&contract.ProposalOTCProgramFund{Amount: 0}); err == nil {
		t.Error("zero-amount fund proposal should be refused")
	}
}

// TestOTCCheckHandlers covers the stateless admission guards.
func TestOTCCheckHandlers(t *testing.T) {
	c, _ := newTestCanoliq()
	p := DefaultParams()
	user := addr20(0xA1)

	if r := c.CheckMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, p); r.Error != nil {
		t.Fatalf("valid create rejected: %v", r.Error)
	}
	cases := []struct {
		name     string
		msg      *contract.MessageOTCLockCreate
		fee      uint64
		wantCode uint64
	}{
		{"short address", &contract.MessageOTCLockCreate{FromAddress: []byte{1}, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D}, testFee, codeInvalidAddress},
		{"zero amount", &contract.MessageOTCLockCreate{FromAddress: user, CcnpyAmount: 0, Tier: contract.OTCLockTier_OTC_LOCK_90D}, testFee, codeInvalidAmount},
		{"bad tier", &contract.MessageOTCLockCreate{FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier(9)}, testFee, codeInvalidOTCLockTier},
		{"dust", &contract.MessageOTCLockCreate{FromAddress: user, CcnpyAmount: 1, Tier: contract.OTCLockTier_OTC_LOCK_90D}, testFee, codeOTCLockBelowMinimum},
		{"fee too low", &contract.MessageOTCLockCreate{FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D}, 1, codeFeeBelowMinimum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := c.CheckMessageOTCLockCreate(tc.msg, tc.fee, p)
			if r.Error == nil || r.Error.Code != tc.wantCode {
				t.Errorf("got %v, want code %d", r.Error, tc.wantCode)
			}
		})
	}
	if r := c.CheckMessageOTCLockClaim(&contract.MessageOTCLockClaim{FromAddress: user, LockId: 1}, 1, p); r.Error == nil || r.Error.Code != codeFeeBelowMinimum {
		t.Errorf("claim fee floor: got %v", r.Error)
	}
	if r := c.CheckMessageOTCLockCancel(&contract.MessageOTCLockCancel{FromAddress: []byte{1}, LockId: 1}, testFee, p); r.Error == nil || r.Error.Code != codeInvalidAddress {
		t.Errorf("cancel address check: got %v", r.Error)
	}
}

// TestOTCLockDispatch confirms both switches route the new messages. Wiring
// only one produces a message that clears the mempool then fails at delivery.
func TestOTCLockDispatch(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	_ = params
	msgs := []proto.Message{
		&contract.MessageOTCLockCreate{FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D},
		&contract.MessageOTCLockClaim{FromAddress: user, LockId: 1},
		&contract.MessageOTCLockCancel{FromAddress: user, LockId: 1},
	}
	for _, m := range msgs {
		payload, err := anypb.New(m)
		if err != nil {
			t.Fatalf("to any: %v", err)
		}
		tx := &contract.Transaction{Msg: payload, Fee: testFee}
		check := c.CheckTx(&contract.PluginCheckRequest{Tx: tx})
		if check.Error != nil && check.Error.Code == codeUnsupportedMessage {
			t.Errorf("%T not wired into CheckTx", m)
		}
		deliver := c.dispatchDeliver(&contract.PluginDeliverRequest{Tx: tx})
		if deliver.Error != nil && deliver.Error.Code == codeUnsupportedMessage {
			t.Errorf("%T not wired into dispatchDeliver", m)
		}
	}
	_ = s
}

// === Defect-fix coverage ===
//
// The three tests below pin the defects this change fixes. All three were
// silently broken before and none had coverage, which is how they survived.

// TestGenesisTreasuryDestinationFundsScalar is the core fix. Before this, the
// DAO Treasury tranche credited a per-address CPLQ balance and nothing ever
// wrote KeyForTreasuryCPLQ, so the scalar read zero forever and both of its
// consumers — SPEND_CPLQ treasury spends and every buyback — failed their
// balance guard on any real chain.
func TestGenesisTreasuryDestinationFundsScalar(t *testing.T) {
	c, s := newTestCanoliq()
	gf := miniGenesis()
	for i := range gf.Buckets {
		if gf.Buckets[i].Name == "treasury" {
			gf.Buckets[i].Destination = BucketDestTreasury
			gf.Buckets[i].Recipients = nil
		}
	}
	if resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)}); resp.Error != nil {
		t.Fatalf("genesis: %v", resp.Error)
	}
	// 1500 bps of 100M CPLQ = 15M CPLQ.
	const wantTreasury = 15_000_000 * 1_000_000
	if got := readScalarKey(s, KeyForTreasuryCPLQ()); got != wantTreasury {
		t.Errorf("treasury_cplq = %d, want %d", got, uint64(wantTreasury))
	}
	// The tranche must not also land at an address.
	if got := readCplq(s, addr20(0xa4)); got != 0 {
		t.Errorf("treasury tranche also credited an address: %d", got)
	}
	// Treasury holdings are not circulating. miniGenesis's liquid-at-TGE
	// tranches are liquidity (1500) + community (2000) + grants (600) = 4100
	// bps = 41M CPLQ. If the treasury tranche were wrongly counted this would
	// read 56M, so the exact figure is the assertion that matters.
	const wantCirculating = 41_000_000 * 1_000_000
	g := loadGlobals(t, s)
	if g.CplqCirculatingSupply != wantCirculating {
		t.Errorf("circulating = %d, want %d (treasury tranche must be excluded)", g.CplqCirculatingSupply, uint64(wantCirculating))
	}
	// And the funding path can now actually draw on it, which is the point.
	if err := c.fundOTCProgram(&contract.ProposalOTCProgramFund{Amount: testBudget}); err != nil {
		t.Fatalf("fund from genesis-seeded treasury: %v", err)
	}
	if got := readScalarKey(s, KeyForOTCBudgetAvailable()); got != testBudget {
		t.Errorf("program budget = %d, want %d", got, uint64(testBudget))
	}
}

// TestGenesisRejectsMalformedAddresses covers the validation gap that made a
// mis-sized address mint a whole tranche to an unspendable key. hex.DecodeString
// accepts any even-length string, so length has to be checked separately, and
// genesis is one-shot so the mistake is unrecoverable.
func TestGenesisRejectsMalformedAddresses(t *testing.T) {
	cases := []struct {
		name string
		addr string
	}{
		{"nineteen bytes", "00112233445566778899aabbccddeeff001122"},
		{"twenty one bytes", "00112233445566778899aabbccddeeff00112233aa"},
		{"empty", ""},
		{"odd length", "abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestCanoliq()
			gf := miniGenesis()
			gf.Buckets[0].Recipients = []GenesisAllocation{{Address: tc.addr, Bps: 10000}}
			resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
			if resp.Error == nil {
				t.Fatalf("genesis accepted a malformed address %q", tc.addr)
			}
		})
	}
	// A treasury-destined bucket carrying recipients is a config error, not a
	// silently ignored field.
	c, _ := newTestCanoliq()
	gf := miniGenesis()
	gf.Buckets[3].Destination = BucketDestTreasury
	if resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)}); resp.Error == nil {
		t.Error("treasury bucket with recipients should be refused")
	}
}

// TestValidateParamsRejectsDuplicateSigners covers the multisig gap:
// countMultisigApprovals tallies by iterating the signer list and probing one
// key per entry, so a duplicated address contributes its single approval once
// per occurrence. Five slots holding four distinct keys would let one holder
// cast two of the three approvals a large spend needs.
func TestValidateParamsRejectsDuplicateSigners(t *testing.T) {
	p := DefaultParams()
	p.MultisigSigners = [][]byte{addr20(0xb1), addr20(0xb2), addr20(0xb1), addr20(0xb4), addr20(0xb5)}
	p.MultisigThreshold = 3
	if err := ValidateParams(p); err == nil {
		t.Fatal("duplicate multisig signers accepted")
	}
	p.MultisigSigners = [][]byte{addr20(0xb1), addr20(0xb2), addr20(0xb3), addr20(0xb4), addr20(0xb5)}
	if err := ValidateParams(p); err != nil {
		t.Fatalf("five distinct signers rejected: %v", err)
	}
}

// TestBuybackRefundDoesNotMint covers the defect that becomes reachable the
// moment the treasury is funded: the void path re-read treasury_cplq with
// readScalar, which sees committed state and therefore missed the caller's
// pending deduction. Appended after that deduction and applied last-write-wins,
// the net effect was `original + acquired` — a silent CPLQ mint.
func TestBuybackRefundDoesNotMint(t *testing.T) {
	const (
		treasury = 1_000_000_000
		acquired = 250_000_000
	)
	// With no stakers, distribute mode voids the buyback and refunds. The
	// refund must restore exactly the original balance, never more.
	c, _ := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	sets, err := c.distributeBuybackToStakers(acquired, treasury-acquired)
	if err != nil {
		t.Fatalf("distribute: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("expected one refund op, got %d", len(sets))
	}
	got := DecodeUint64(sets[0].Value)
	if got != treasury {
		t.Errorf("refund wrote %d, want %d — anything higher is a mint", got, uint64(treasury))
	}
}

// TestOTCLockTermChangeDoesNotMoveOpenPositions covers the consequence of
// making the tier terms governance-tunable: a term change must apply to new
// positions only. MatureHeight is written once at creation, so an open position
// keeps the maturity it was sold, and governance cannot shorten or extend a
// lock somebody already agreed to.
func TestOTCLockTermChangeDoesNotMoveOpenPositions(t *testing.T) {
	c, s, user, params := otcFixture(t, testLockAmount, testBudget)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user,
		CcnpyAmount: testLockAmount,
		Tier:        contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create: %v", resp.Error)
	}
	before := loadOTCLock(t, s, user, 1)
	if before == nil {
		t.Fatal("lock record not written")
	}
	wantMature := uint64(100 + 1_296_000)
	if before.MatureHeight != wantMature {
		t.Fatalf("MatureHeight = %d, want %d", before.MatureHeight, wantMature)
	}

	// Governance shortens the 90-day tier to 10 days. The open position must
	// not move, and a claim before its original maturity must still be refused.
	params.OtcTier90Blocks = 10 * blocksPerDay
	if got := loadOTCLock(t, s, user, 1).MatureHeight; got != wantMature {
		t.Errorf("MatureHeight after term change = %d, want %d (unchanged)", got, wantMature)
	}
	c.plugin.setHeight(100 + 10*blocksPerDay + 1) // past the NEW term, not the old
	resp := c.DeliverMessageOTCLockClaim(&contract.MessageOTCLockClaim{
		FromAddress: user, LockId: 1,
	}, testFee, params)
	if resp.Error == nil {
		t.Fatal("claim succeeded against the shortened term; an open position must keep its original maturity")
	}

	// A position opened after the change gets the new, shorter term.
	seedCcnpy(s, user, testLockAmount)
	if resp := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user,
		CcnpyAmount: testLockAmount,
		Tier:        contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params); resp.Error != nil {
		t.Fatalf("create after term change: %v", resp.Error)
	}
	second := loadOTCLock(t, s, user, 2)
	if second == nil {
		t.Fatal("second lock not written")
	}
	if got, want := second.MatureHeight-second.StartHeight, uint64(10*blocksPerDay); got != want {
		t.Errorf("new position term = %d blocks, want %d", got, want)
	}
}

// TestOTCZeroTierRateIsNotReportedAsExhaustedBudget covers the diagnostic
// split. A tier rate of zero and a budget too small to cover the reward used
// to produce the same ErrOTCBudgetExhausted, which sent operators looking at
// program funding for a tier governance had deliberately disabled.
func TestOTCZeroTierRateIsNotReportedAsExhaustedBudget(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	params.OtcTier90Bps = 0 // disable the 90-day tier, leave 120-day running
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, CplqTotalSupply: CPLQTotalSupply})
	user := addr20(0xA1)
	seedAccount(s, user, 1_000_000)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(testLockAmount*4))
	s.set(KeyForOTCBudgetAvailable(), EncodeUint64(testBudget))
	c.plugin.setHeight(10)

	r := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_90D,
	}, testFee, params)
	if r.Error == nil {
		t.Fatal("a lock on a zero-rate tier should be refused")
	}
	if r.Error.Code != codeOTCTierRateZero {
		t.Errorf("got code %d (%s), want codeOTCTierRateZero: a disabled tier is not an exhausted budget",
			r.Error.Code, r.Error.Msg)
	}

	// The other tier is unaffected, which is the whole point of allowing a
	// zero rate.
	if r := c.DeliverMessageOTCLockCreate(&contract.MessageOTCLockCreate{
		FromAddress: user, CcnpyAmount: testLockAmount, Tier: contract.OTCLockTier_OTC_LOCK_120D,
	}, testFee, params); r.Error != nil {
		t.Errorf("the 120-day tier should still work: %v", r.Error)
	}
}
