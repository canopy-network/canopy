package canoliq

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// jsonMarshal is a thin wrapper to keep encoding/json out of the package's
// non-test code while letting tests call a package-local helper.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

// addr20 builds a 20-byte address from a single byte b for terse tests.
func addr20(b byte) []byte {
	out := make([]byte, 20)
	for i := range out {
		out[i] = b
	}
	return out
}

// seedAccount stores a CNPY balance for `addr` in the fake state under the
// canopy core account prefix.
func seedAccount(s *fakeStore, addr []byte, amount uint64) {
	bz, _ := contract.Marshal(&contract.Account{Address: addr, Amount: amount})
	s.set(contract.KeyForAccount(addr), bz)
}

// seedEscrow seeds the CNPY escrow pool (H1). Tests that pre-seed
// globals.TotalPooledCnpy / PendingRedemptionCnpy (simulating prior deposits
// without driving the deposit handler) must also fund escrow to honor the
// invariant escrow.Amount == TotalPooledCnpy + PendingRedemptionCnpy.
func seedEscrow(s *fakeStore, amount uint64) {
	bz, _ := contract.Marshal(&contract.Pool{Amount: amount})
	s.set(KeyForEscrowPool(), bz)
}

// readEscrow returns the current CNPY escrow pool balance.
func readEscrow(s *fakeStore) uint64 {
	bz := s.get(KeyForEscrowPool())
	if bz == nil {
		return 0
	}
	p := new(contract.Pool)
	_ = contract.Unmarshal(bz, p)
	return p.Amount
}

// readAccount returns the CNPY balance for `addr`.
func readAccount(s *fakeStore, addr []byte) uint64 {
	bz := s.get(contract.KeyForAccount(addr))
	if bz == nil {
		return 0
	}
	a := new(contract.Account)
	if err := contract.Unmarshal(bz, a); err != nil {
		return 0
	}
	return a.Amount
}

// readPool returns the CNPY held by the canoLiq committee fee pool.
func readPool(s *fakeStore, chainId uint64) uint64 {
	bz := s.get(contract.KeyForFeePool(chainId))
	if bz == nil {
		return 0
	}
	p := new(contract.Pool)
	if err := contract.Unmarshal(bz, p); err != nil {
		return 0
	}
	return p.Amount
}

// readCcnpy returns the cCNPY balance for `addr` in the fake state.
func readCcnpy(s *fakeStore, addr []byte) uint64 {
	return DecodeUint64(s.get(KeyForCCNPYBalance(addr)))
}

func readCplq(s *fakeStore, addr []byte) uint64 {
	return DecodeUint64(s.get(KeyForCPLQBalance(addr)))
}

func loadGlobals(t *testing.T, s *fakeStore) *contract.CanoliqGlobals {
	t.Helper()
	bz := s.get(KeyForGlobals())
	g := new(contract.CanoliqGlobals)
	if bz != nil {
		if err := contract.Unmarshal(bz, g); err != nil {
			t.Fatalf("unmarshal globals: %v", err)
		}
	}
	return g
}

// TestDepositMintsOneToOneOnFirstDeposit verifies the bootstrap exchange rate.
func TestDepositMintsOneToOneOnFirstDeposit(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x01)
	seedAccount(s, user, 1_000_000+10_000) // 1 CNPY + fee
	resp := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("deposit error: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != 1_000_000 {
		t.Fatalf("cCNPY mint: got %d want 1_000_000", got)
	}
	g := loadGlobals(t, s)
	if g.TotalCcnpySupply != 1_000_000 || g.TotalPooledCnpy != 1_000_000 {
		t.Fatalf("globals: ccnpy=%d pooled=%d", g.TotalCcnpySupply, g.TotalPooledCnpy)
	}
}

// TestDepositSubsequentRatio: with pooled CNPY at 110% of cCNPY supply, a
// 100 uCNPY deposit should mint mulDiv(100, 1000, 1100) = 90 cCNPY (truncated).
func TestDepositSubsequentRatio(t *testing.T) {
	c, s := newTestCanoliq()
	// Manually set globals so the pool is hot before the test deposit.
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1100, GenesisComplete: true}
	bz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), bz)
	user := addr20(0x02)
	seedAccount(s, user, 100+10_000)
	resp := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 100},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("deposit error: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != 90 {
		t.Fatalf("ratio mint: got %d want 90", got)
	}
}

// TestRedeemQueuesRedemption verifies cCNPY burn and Redemption record.
func TestRedeemQueuesRedemption(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x03)
	// Seed: 1000 cCNPY, 1000 pooled CNPY, 1000 fee balance, no genesis flag needed.
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	bz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), bz)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	seedAccount(s, user, 10_000)
	resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 400},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("redeem error: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != 600 {
		t.Fatalf("cCNPY remaining: got %d want 600", got)
	}
	g2 := loadGlobals(t, s)
	if g2.PendingRedemptionCnpy != 400 {
		t.Fatalf("pending redemption: got %d want 400", g2.PendingRedemptionCnpy)
	}
	rBz := s.get(KeyForRedemption(user, 0))
	if rBz == nil {
		t.Fatal("redemption record not written at id=0")
	}
	r := new(contract.Redemption)
	_ = contract.Unmarshal(rBz, r)
	if r.CnpyAmount != 400 {
		t.Fatalf("redemption amount: got %d want 400", r.CnpyAmount)
	}
}

// TestClaimBeforeUnbondErrors and TestClaimAfterUnbondSucceeds.
func TestClaimRedemptionMaturity(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x04)
	// Pre-write a redemption that matures at height 100.
	r := &contract.Redemption{Id: 7, Address: user, CnpyAmount: 250, UnbondCompleteHeight: 100}
	bz, _ := contract.Marshal(r)
	s.set(KeyForRedemption(user, 7), bz)
	g := &contract.CanoliqGlobals{PendingRedemptionCnpy: 250}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	seedEscrow(s, 250) // backs the pending redemption (H1)
	seedAccount(s, user, 10_000)

	// Before maturity: error.
	resp := c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 7},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("claim before maturity should error")
	}

	// Advance height past the unbond window.
	c.plugin.setHeight(150)

	resp = c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 7},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("claim after maturity error: %v", resp.Error)
	}
	if readAccount(s, user) != 250 { // 10_000 - 10_000 fee + 250 redemption
		t.Fatalf("user balance after claim: got %d want 250", readAccount(s, user))
	}
	if s.get(KeyForRedemption(user, 7)) != nil {
		t.Fatal("redemption record should be deleted after claim")
	}
}

// TestRewardSplitWhitepaperExample verifies the canonical 12% / 40-30-15-15
// split for a clean X=1000 reward delta. With v1.1 defaults the 30%
// treasury slice is skimmed by insurance_bps=500 (5% of treasury → 0.15%
// of fee) per Tokenomics v1.1 §8. Expected:
//   fee  = 120
//   net  = 880  (to user pool)
//   user-rebate (40% of 120) = 48 (also to user pool) → 928 total to pool
//   treasury (30%) = 36 → 35 after insurance skim (mulDiv(36,500,10000)=1)
//   insurance pool                                   = 1
//   validators (15%) = 18
//   buyback  (15%)   = 18
func TestRewardSplitWhitepaperExample(t *testing.T) {
	c, s := newTestCanoliq()
	// Genesis must be marked complete so the reward sweep runs.
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	// Seed the committee pool with X=1000 uCNPY.
	pool := &contract.Pool{Id: c.Config.ChainId, Amount: 1000}
	pBz, _ := contract.Marshal(pool)
	s.set(contract.KeyForFeePool(c.Config.ChainId), pBz)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}

	g2 := loadGlobals(t, s)
	if g2.TotalPooledCnpy != 928 {
		t.Errorf("total_pooled_cnpy: got %d want 928", g2.TotalPooledCnpy)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != 35 {
		t.Errorf("treasury: got %d want 35 (36 - 1 insurance skim)", got)
	}
	if got := DecodeUint64(s.get(KeyForInsurancePool())); got != 1 {
		t.Errorf("insurance: got %d want 1 (5%% of 36 treasury slice, truncated)", got)
	}
	if got := DecodeUint64(s.get(KeyForBuybackPool())); got != 18 {
		t.Errorf("buyback: got %d want 18", got)
	}
	addr := c.committeeAggregatorAddr()
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(addr))); got != 18 {
		t.Errorf("validators: got %d want 18", got)
	}
	// H1: the committee pool fully drains (1000 in - 1000 swept). The 928 user
	// slice now lands in the escrow pool, so the committee pool and the
	// post-drain LastProcessedRewardPool baseline are both 0.
	if g2.LastProcessedRewardPool != 0 {
		t.Errorf("last_processed_reward_pool: got %d want 0", g2.LastProcessedRewardPool)
	}
	if got := readPool(s, c.Config.ChainId); got != 0 {
		t.Errorf("post-sweep committee pool: got %d want 0", got)
	}
	if got := readEscrow(s); got != 928 {
		t.Errorf("escrow pool: got %d want 928", got)
	}
}

// TestWhitepaperSection7Reconciliation pins the end-to-end yield math from
// the canoLiq Whitepaper v1.1 §7 worked example. Under v1.1, Canopy does NOT
// apply a protocol-level DAO tax on top of rewards before distribution
// (WP §3.3), so canoLiq receives its committee share R directly. With a
// non-round R=950 exercising integer truncation:
//
//	R              = 950
//	fee (12%)      = 114
//	net to users   = 836
//	rebate (40%)   = 45  (114*4000/10000 = 45.6 → 45 truncated)
//	user yield     = 836 + 45 = 881  (= 0.88 * R = 836.0 + rebate)
//
// Effective user yield matches Tokenomics v1.1 §4.1 "Effective user yield =
// 88% × Rewards Received."
func TestWhitepaperSection7Reconciliation(t *testing.T) {
	c, s := newTestCanoliq()
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	const R = 950 // canoLiq's committee share — non-round to exercise truncation

	pool := &contract.Pool{Id: c.Config.ChainId, Amount: R}
	pBz, _ := contract.Marshal(pool)
	s.set(contract.KeyForFeePool(c.Config.ChainId), pBz)

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}

	g2 := loadGlobals(t, s)
	const wantUserYield = 881
	if g2.TotalPooledCnpy != wantUserYield {
		t.Errorf("user yield: got %d want %d (Tokenomics v1.1 §4.1: 0.88 * R with truncation)", g2.TotalPooledCnpy, wantUserYield)
	}
	// Sanity: fee = 114, treasury = 34 (114*3000/10000 = 34.2 truncated, plus
	// any residual from rounding the splits goes to treasury). insurance_bps=500
	// skims 5% off the treasury credit per Tokenomics v1.1 §8.
	const wantFee = 114
	const wantTreasuryRaw = 34 + 1 // 35 before insurance skim
	const wantInsurance = 1        // mulDiv(35, 500, 10000) = 1
	const wantTreasury = wantTreasuryRaw - wantInsurance
	const wantValidators = 17 // 114*1500/10000 = 17.1 → 17
	const wantBuyback = 17    // ditto
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != wantTreasury {
		t.Errorf("treasury: got %d want %d", got, wantTreasury)
	}
	if got := DecodeUint64(s.get(KeyForInsurancePool())); got != wantInsurance {
		t.Errorf("insurance: got %d want %d", got, wantInsurance)
	}
	if got := DecodeUint64(s.get(KeyForBuybackPool())); got != wantBuyback {
		t.Errorf("buyback: got %d want %d", got, wantBuyback)
	}
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr()))); got != wantValidators {
		t.Errorf("validators: got %d want %d", got, wantValidators)
	}
	// Conservation: R = userYield + treasury + insurance + validators + buyback.
	total := g2.TotalPooledCnpy +
		DecodeUint64(s.get(KeyForTreasuryCNPY())) +
		DecodeUint64(s.get(KeyForInsurancePool())) +
		DecodeUint64(s.get(KeyForBuybackPool())) +
		DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr())))
	if total != R {
		t.Errorf("conservation: %d (yield %d + treasury %d + insurance %d + buyback %d + validators %d) want %d",
			total, g2.TotalPooledCnpy,
			DecodeUint64(s.get(KeyForTreasuryCNPY())),
			DecodeUint64(s.get(KeyForInsurancePool())),
			DecodeUint64(s.get(KeyForBuybackPool())),
			DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr()))),
			R)
	}
	_ = wantFee
}

// TestVestingLinearUnlock verifies vesting math at three sample points.
func TestVestingLinearUnlock(t *testing.T) {
	s := &contract.VestingSchedule{
		TotalAmount:  1_000_000,
		CliffHeight:  100,
		StartHeight:  100,
		EndHeight:    200,
	}
	cases := []struct {
		height uint64
		want   uint64
	}{
		{50, 0},          // before cliff
		{100, 0},         // exactly at cliff: 0/100 elapsed
		{150, 500_000},   // halfway
		{200, 1_000_000}, // saturated
		{300, 1_000_000}, // beyond end stays saturated
	}
	for _, tc := range cases {
		got := unlockedAmount(s, tc.height)
		if got != tc.want {
			t.Errorf("unlockedAmount@%d: got %d want %d", tc.height, got, tc.want)
		}
	}
}

// TestCPLQTransferRespectsLiquidBalance ensures transfers fail when the
// requested amount exceeds the liquid balance, and succeed when it does not.
func TestCPLQTransferRespectsLiquidBalance(t *testing.T) {
	c, s := newTestCanoliq()
	from := addr20(0x05)
	to := addr20(0x06)
	seedAccount(s, from, 10_000)             // CNPY for fee
	s.set(KeyForCPLQBalance(from), EncodeUint64(500))

	// Over-transfer fails.
	resp := c.DeliverMessageCPLQTransfer(
		&contract.MessageCPLQTransfer{FromAddress: from, ToAddress: to, Amount: 1000},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("transfer of 1000 from balance 500 should fail")
	}

	// Within-balance succeeds.
	resp = c.DeliverMessageCPLQTransfer(
		&contract.MessageCPLQTransfer{FromAddress: from, ToAddress: to, Amount: 200},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("transfer error: %v", resp.Error)
	}
	if readCplq(s, from) != 300 || readCplq(s, to) != 200 {
		t.Fatalf("post-transfer balances: from=%d to=%d (want 300/200)",
			readCplq(s, from), readCplq(s, to))
	}
}

// TestGenesisAllocationTotals: after running genesis, sum of all liquid
// balances + sum of all VestingSchedule.TotalAmount must equal CPLQTotalSupply.
func TestGenesisAllocationTotals(t *testing.T) {
	c, s := newTestCanoliq()
	gf := miniGenesis()
	// Inject the genesis JSON via PluginGenesisRequest.GenesisJson.
	gfBytes := mustJSON(t, gf)
	resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: gfBytes})
	if resp.Error != nil {
		t.Fatalf("genesis: %v", resp.Error)
	}
	g := loadGlobals(t, s)
	if g.CplqTotalSupply != CPLQTotalSupply {
		t.Fatalf("cplq_total_supply: got %d want %d", g.CplqTotalSupply, CPLQTotalSupply)
	}
	// Sum allocations across the store.
	var allocated uint64
	for k, v := range s.data {
		if hasPrefix(k, KeyForCPLQBalance(nil)) {
			allocated += DecodeUint64(v)
			continue
		}
		if hasPrefix(k, JoinLenPrefix(canoliqPrefix, domainVesting)) {
			vs := new(contract.VestingSchedule)
			if err := contract.Unmarshal(v, vs); err == nil {
				allocated += vs.TotalAmount
			}
		}
	}
	if allocated != CPLQTotalSupply {
		t.Fatalf("allocated total: got %d want %d", allocated, CPLQTotalSupply)
	}
}

// TestBeginBlockSelfBootstrapsGenesis verifies that BeginBlock runs the
// plugin Genesis when Config.GenesisPath is set and globals.GenesisComplete
// is false. This is the path used in production when the FSM does not
// dispatch a PluginGenesisRequest (chain genesis.json has no canoliq
// plugin section). Without it, ProcessRewards bails as a no-op forever.
func TestBeginBlockSelfBootstrapsGenesis(t *testing.T) {
	c, s := newTestCanoliq()
	gfBytes := mustJSON(t, miniGenesis())
	tmp, err := os.CreateTemp(t.TempDir(), "genesis*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := tmp.Write(gfBytes); err != nil {
		t.Fatalf("write genesis: %v", err)
	}
	tmp.Close()
	c.Config.GenesisPath = tmp.Name()
	// Sanity: globals starts un-initialized.
	if g := loadGlobals(t, s); g.GenesisComplete {
		t.Fatalf("precondition: genesis already complete")
	}
	resp := c.BeginBlock(&contract.PluginBeginRequest{Height: 1})
	if resp.Error != nil {
		t.Fatalf("BeginBlock: %v", resp.Error)
	}
	g := loadGlobals(t, s)
	if !g.GenesisComplete {
		t.Fatalf("genesis did not run; globals=%+v", g)
	}
	if g.CplqTotalSupply != CPLQTotalSupply {
		t.Fatalf("supply: got %d want %d", g.CplqTotalSupply, CPLQTotalSupply)
	}
	// Re-run is idempotent.
	resp = c.BeginBlock(&contract.PluginBeginRequest{Height: 2})
	if resp.Error != nil {
		t.Fatalf("idempotent BeginBlock: %v", resp.Error)
	}
	if g2 := loadGlobals(t, s); g2.CplqTotalSupply != CPLQTotalSupply {
		t.Fatalf("idempotent supply changed: %d", g2.CplqTotalSupply)
	}
}

// TestDeliverCPLQClaimVestedFlow exercises the full vesting-claim handler:
// before-cliff returns NothingToClaim, halfway claims the linear share, an
// idempotent second call at the same height also returns NothingToClaim, and a
// final call past end_height drains the schedule. Closes the gap left by
// TestVestingLinearUnlock, which only covered the unlockedAmount math helper.
func TestDeliverCPLQClaimVestedFlow(t *testing.T) {
	c, s := newTestCanoliq()
	holder := addr20(0x07)
	seedAccount(s, holder, 100_000) // covers fees on the two successful claims

	sched := &contract.VestingSchedule{
		Address:     holder,
		TotalAmount: 1_000_000,
		CliffHeight: 100,
		StartHeight: 100,
		EndHeight:   200,
	}
	sBz, _ := contract.Marshal(sched)
	s.set(KeyForVesting(holder, 0), sBz)

	idx := &contract.VestingIndex{ScheduleIds: []uint64{0}}
	iBz, _ := contract.Marshal(idx)
	s.set(KeyForVestingIndex(holder), iBz)

	// Before cliff: nothing unlocked.
	c.plugin.setHeight(50)
	resp := c.DeliverMessageCPLQClaimVested(
		&contract.MessageCPLQClaimVested{FromAddress: holder},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("claim before cliff should error")
	}
	if got := readCplq(s, holder); got != 0 {
		t.Fatalf("liquid CPLQ before cliff: got %d want 0", got)
	}
	if got := readAccount(s, holder); got != 100_000 {
		t.Fatalf("CNPY untouched on failed claim: got %d want 100_000", got)
	}

	// Halfway through the vesting span (height 150 of [100,200]): expect 500_000.
	c.plugin.setHeight(150)
	resp = c.DeliverMessageCPLQClaimVested(
		&contract.MessageCPLQClaimVested{FromAddress: holder},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("halfway claim error: %v", resp.Error)
	}
	if got := readCplq(s, holder); got != 500_000 {
		t.Fatalf("halfway liquid CPLQ: got %d want 500_000", got)
	}
	if got := readAccount(s, holder); got != 90_000 {
		t.Fatalf("CNPY after fee: got %d want 90_000", got)
	}

	// Same height, second call: claimed_amount already covers what's unlocked.
	resp = c.DeliverMessageCPLQClaimVested(
		&contract.MessageCPLQClaimVested{FromAddress: holder},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("idempotent claim at same height should error (nothing to claim)")
	}

	// Past end_height: remaining 500_000 unlocks.
	c.plugin.setHeight(250)
	resp = c.DeliverMessageCPLQClaimVested(
		&contract.MessageCPLQClaimVested{FromAddress: holder},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("past-end claim error: %v", resp.Error)
	}
	if got := readCplq(s, holder); got != 1_000_000 {
		t.Fatalf("final liquid CPLQ: got %d want 1_000_000", got)
	}

	// claimed_amount must reflect the saturation.
	persisted := new(contract.VestingSchedule)
	_ = contract.Unmarshal(s.get(KeyForVesting(holder, 0)), persisted)
	if persisted.ClaimedAmount != 1_000_000 {
		t.Fatalf("schedule claimed_amount: got %d want 1_000_000", persisted.ClaimedAmount)
	}

	// Globals must track circulating supply.
	g := loadGlobals(t, s)
	if g.CplqCirculatingSupply != 1_000_000 {
		t.Fatalf("circulating CPLQ: got %d want 1_000_000", g.CplqCirculatingSupply)
	}
}

// TestRewardSweepMultiBlock pins down that LastProcessedRewardPool functions
// as a per-block watermark so consecutive EndBlock invocations operate on the
// fresh delta only — the property a real localnet would exercise across blocks
// but was previously only tested at a single sweep.
func TestRewardSweepMultiBlock(t *testing.T) {
	c, s := newTestCanoliq()
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)

	setPool := func(amount uint64) {
		p := &contract.Pool{Id: c.Config.ChainId, Amount: amount}
		bz, _ := contract.Marshal(p)
		s.set(contract.KeyForFeePool(c.Config.ChainId), bz)
	}
	addToPool := func(delta uint64) {
		p := new(contract.Pool)
		_ = contract.Unmarshal(s.get(contract.KeyForFeePool(c.Config.ChainId)), p)
		p.Amount += delta
		bz, _ := contract.Marshal(p)
		s.set(contract.KeyForFeePool(c.Config.ChainId), bz)
	}

	// Block 1: 1000 inflow → fee 120 → user share 928, treasury 36, val 18, buyback 18.
	setPool(1000)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("block 1: %v", err)
	}
	g1 := loadGlobals(t, s)
	// H1: the committee pool fully drains each block; the user slice (928) now
	// lives in the escrow pool, and LastProcessedRewardPool returns to its
	// post-drain baseline (0).
	if g1.TotalPooledCnpy != 928 || g1.LastProcessedRewardPool != 0 {
		t.Fatalf("block 1 globals: pooled=%d last=%d (want 928/0)",
			g1.TotalPooledCnpy, g1.LastProcessedRewardPool)
	}
	if got := readEscrow(s); got != 928 {
		t.Fatalf("block 1 escrow: got %d want 928", got)
	}

	// Block 2: +500 fresh inflow on top of the post-sweep pool. Delta must
	// isolate to 500, NOT the cumulative 1428.
	addToPool(500)
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 2}); err != nil {
		t.Fatalf("block 2: %v", err)
	}
	g2 := loadGlobals(t, s)
	// fee=60, net=440, rebate=24 → block 2 user share = 464; cumulative = 928+464.
	if g2.TotalPooledCnpy != 1392 {
		t.Fatalf("block 2 pooled: got %d want 1392", g2.TotalPooledCnpy)
	}
	if g2.LastProcessedRewardPool != 0 {
		t.Fatalf("block 2 last-processed: got %d want 0", g2.LastProcessedRewardPool)
	}
	if got := readEscrow(s); got != 1392 {
		t.Fatalf("block 2 escrow: got %d want 1392", got)
	}
	// Per-block under insurance_bps=500: treasury 36→35 + insurance 1; block 2
	// treasury 18→18 + insurance 0 (mulDiv(18,500,10000)=0.9 → 0). Cumulative:
	// treasury 53, insurance 1, validators 27, buyback 27.
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != 53 {
		t.Errorf("treasury cumulative: got %d want 53", got)
	}
	if got := DecodeUint64(s.get(KeyForInsurancePool())); got != 1 {
		t.Errorf("insurance cumulative: got %d want 1", got)
	}
	if got := DecodeUint64(s.get(KeyForBuybackPool())); got != 27 {
		t.Errorf("buyback cumulative: got %d want 27", got)
	}
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr()))); got != 27 {
		t.Errorf("validators cumulative: got %d want 27", got)
	}

	// Block 3: no inflow. Pool == watermark, so the sweep is a no-op for users
	// and accumulators must NOT advance.
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 3}); err != nil {
		t.Fatalf("block 3: %v", err)
	}
	g3 := loadGlobals(t, s)
	if g3.TotalPooledCnpy != 1392 {
		t.Fatalf("block 3 pooled drift: got %d want 1392", g3.TotalPooledCnpy)
	}
	if got := readEscrow(s); got != 1392 {
		t.Fatalf("block 3 escrow drift: got %d want 1392", got)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != 53 {
		t.Errorf("block 3 treasury drift: got %d want 53", got)
	}
}

// TestCompositeDepositRewardRedeem walks one user through the full lifecycle:
// deposit → reward injection → process → redeem. Asserts that the queued
// Redemption pays back strictly more CNPY than was deposited, proving yield
// actually flows to cCNPY holders end-to-end. This is the closest in-process
// analog to the localnet "deposit, accrue, redeem" smoke test.
func TestCompositeDepositRewardRedeem(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x10)
	// Seed enough CNPY for the deposit + its 10_000 fee + a later redeem fee.
	seedAccount(s, user, 1_000_000+10_000+10_000)

	g := &contract.CanoliqGlobals{GenesisComplete: true}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)

	// 1) Deposit 1_000_000 → mints 1_000_000 cCNPY at 1:1.
	if resp := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000},
		10_000,
		DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("deposit: %v", resp.Error)
	}
	if got := readCcnpy(s, user); got != 1_000_000 {
		t.Fatalf("post-deposit cCNPY: got %d want 1_000_000", got)
	}

	// 2) Pin the watermark to the current pool balance so the upcoming reward
	//    delta is isolated from any side effects of the deposit (its 10_000
	//    fee accrued into the same committee pool key).
	gAfterDeposit := loadGlobals(t, s)
	gAfterDeposit.LastProcessedRewardPool = readPool(s, c.Config.ChainId)
	gBz2, _ := contract.Marshal(gAfterDeposit)
	s.set(KeyForGlobals(), gBz2)

	// 3) Inject a 1_000_000 reward into the committee pool (simulates a
	//    MessageSubsidy crediting the canoLiq committee).
	pool := new(contract.Pool)
	_ = contract.Unmarshal(s.get(contract.KeyForFeePool(c.Config.ChainId)), pool)
	pool.Amount += 1_000_000
	pBz, _ := contract.Marshal(pool)
	s.set(contract.KeyForFeePool(c.Config.ChainId), pBz)

	// 4) Process rewards. Per the canonical 12% / 40-30-15-15 split:
	//    fee=120_000, net=880_000, rebate=48_000 → user share = 928_000.
	//    TotalPooledCnpy: 1_000_000 → 1_928_000.
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("rewards: %v", err)
	}
	gPostReward := loadGlobals(t, s)
	if gPostReward.TotalPooledCnpy != 1_928_000 {
		t.Fatalf("post-reward pooled: got %d want 1_928_000", gPostReward.TotalPooledCnpy)
	}

	// 5) Redeem all 1_000_000 cCNPY at the new exchange rate.
	if resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 1_000_000},
		10_000,
		DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}

	// The Redemption record must reflect the accrued yield: 1:1 redemption
	// before reward would be 1_000_000; after reward, the same cCNPY claims
	// ~1_928_000 CNPY (1 uCNPY retained as M5 virtual-offset dust).
	rBz := s.get(KeyForRedemption(user, 0))
	if rBz == nil {
		t.Fatal("redemption record not written")
	}
	r := new(contract.Redemption)
	_ = contract.Unmarshal(rBz, r)
	if r.CnpyAmount <= 1_000_000 {
		t.Fatalf("expected redemption > deposit (yield), got %d", r.CnpyAmount)
	}
	if r.CnpyAmount != 1_927_999 {
		t.Fatalf("redemption amount: got %d want 1_927_999", r.CnpyAmount)
	}
}

// hasPrefix is a small helper for TestGenesisAllocationTotals' iteration.
// `keyPrefix` is the canonical prefix as built by JoinLenPrefix; both keys
// share the same first len(keyPrefix) - len(addr) bytes.
func hasPrefix(key string, keyPrefix []byte) bool {
	if len(key) < len(keyPrefix) {
		return false
	}
	for i, b := range keyPrefix {
		// Stop comparing once we hit the address-length byte (variable). For
		// our key layout, the prefix up to the canonical fixed segments is
		// constant; the address segment that follows starts with byte 20
		// (length=20) and we don't need to match beyond it for prefix.
		if i >= len(key) {
			return false
		}
		if key[i] != b {
			return false
		}
	}
	return true
}

// miniGenesis constructs an in-memory genesis with seven distinct addresses,
// matching the canonical 22/15/20/15/12/10/6 split. Useful for keeping the
// test self-contained without reading genesis.json from disk.
func miniGenesis() *GenesisFile {
	rec := func(addr byte) []GenesisAllocation {
		return []GenesisAllocation{{Address: hex.EncodeToString(addr20(addr)), Bps: 10000}}
	}
	return &GenesisFile{
		BlocksPerYear: 5_256_000,
		Buckets: []GenesisBucket{
			{Name: "validators", Bps: 2200, CliffMonths: 6, VestMonths: 24, Recipients: rec(0xa1)},
			{Name: "liquidity", Bps: 1500, CliffMonths: 0, VestMonths: 0, Recipients: rec(0xa2)},
			{Name: "community", Bps: 2000, CliffMonths: 0, VestMonths: 0, Recipients: rec(0xa3)},
			{Name: "treasury", Bps: 1500, CliffMonths: 0, VestMonths: 0, Recipients: rec(0xa4)},
			{Name: "founders", Bps: 1200, CliffMonths: 12, VestMonths: 24, Recipients: rec(0xa5)},
			{Name: "partners", Bps: 1000, CliffMonths: 6, VestMonths: 18, Recipients: rec(0xa6)},
			{Name: "grants", Bps: 600, CliffMonths: 0, VestMonths: 0, Recipients: rec(0xa7)},
		},
	}
}

// mustJSON marshals to JSON, failing the test on error.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	bz, err := jsonMarshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bz
}

// TestGenesisValidatorRegistrySeedsState confirms that a genesis file
// carrying a validatorRegistry block writes the expected
// ValidatorRegistry record under KeyForValidatorRegistry. The next
// reward sweep then routes the validator slice via the registry rather
// than the legacy aggregator address.
func TestGenesisValidatorRegistrySeedsState(t *testing.T) {
	c, s := newTestCanoliq()
	gf := miniGenesis()
	gf.ValidatorRegistry = []GenesisValidatorRegistryEntry{
		{Address: hex.EncodeToString(addr20(0xb1)), Stake: 7_000_000},
		{Address: "0x" + hex.EncodeToString(addr20(0xb2)), Stake: 3_000_000},
	}
	resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
	if resp.Error != nil {
		t.Fatalf("genesis: %v", resp.Error)
	}
	bz := s.get(KeyForValidatorRegistry())
	if len(bz) == 0 {
		t.Fatalf("validator registry not written to state")
	}
	got := new(contract.ValidatorRegistry)
	if err := contract.Unmarshal(bz, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entries: got %d want 2", len(got.Entries))
	}
	if got.Entries[0].Stake != 7_000_000 || got.Entries[1].Stake != 3_000_000 {
		t.Fatalf("stakes: %+v", got.Entries)
	}
	// 0x-prefix accepted on the second address but stripped during seed.
	if len(got.Entries[1].Address) != 20 {
		t.Fatalf("address[1] should be 20 raw bytes, got %d", len(got.Entries[1].Address))
	}
}

// TestGenesisValidatorRegistryEmptyKeepsLegacyPath confirms a genesis
// with no validatorRegistry block doesn't write the key — preserving
// the legacy aggregator fallback for Phase 1 deployments that don't
// care about per-validator pro-rata yet.
func TestGenesisValidatorRegistryEmptyKeepsLegacyPath(t *testing.T) {
	c, s := newTestCanoliq()
	gf := miniGenesis()
	resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
	if resp.Error != nil {
		t.Fatalf("genesis: %v", resp.Error)
	}
	if bz := s.get(KeyForValidatorRegistry()); len(bz) != 0 {
		t.Fatalf("expected no registry key when genesis omits it; got %d bytes", len(bz))
	}
}

// TestGenesisValidatorRegistryRejectsBadAddress confirms malformed
// hex / wrong-length addresses fail genesis cleanly so a typo in the
// registry block doesn't silently corrupt the per-validator ledger.
func TestGenesisValidatorRegistryRejectsBadAddress(t *testing.T) {
	c, _ := newTestCanoliq()
	gf := miniGenesis()
	gf.ValidatorRegistry = []GenesisValidatorRegistryEntry{
		{Address: "0xnope", Stake: 1},
	}
	resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
	if resp.Error == nil {
		t.Fatalf("expected genesis to reject malformed validator address")
	}

	// Wrong-length (19 bytes) also rejected.
	c, _ = newTestCanoliq()
	gf = miniGenesis()
	gf.ValidatorRegistry = []GenesisValidatorRegistryEntry{
		{Address: hex.EncodeToString(make([]byte, 19)), Stake: 1},
	}
	resp = c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
	if resp.Error == nil {
		t.Fatalf("expected genesis to reject 19-byte validator address")
	}
}

// TestBundledLocalnetGenesisSeedsTwoValidators verifies the committed
// genesis.localnet.json carries the two real localnet validator
// addresses (851e90… and 02cd4e…), each with stake matching the chain
// genesis. This anchors the live-docker reconciliation to the registry
// path rather than the legacy aggregator.
func TestBundledLocalnetGenesisSeedsTwoValidators(t *testing.T) {
	data, err := os.ReadFile("genesis.localnet.json")
	if err != nil {
		t.Skipf("genesis.localnet.json not present: %v", err)
	}
	gf := new(GenesisFile)
	if err := json.Unmarshal(data, gf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(gf.ValidatorRegistry) != 2 {
		t.Fatalf("expected 2 seed validators in localnet genesis, got %d", len(gf.ValidatorRegistry))
	}
	wantAddrs := map[string]bool{
		"851e90eaef1fa27debaee2c2591503bdeec1d123": false,
		"02cd4e5eb53ea665702042a6ed6d31d616054dc5": false,
	}
	for _, e := range gf.ValidatorRegistry {
		if _, ok := wantAddrs[e.Address]; !ok {
			t.Errorf("unexpected validator %q in localnet seed", e.Address)
			continue
		}
		wantAddrs[e.Address] = true
		if e.Stake != 1_000_000_000 {
			t.Errorf("validator %q stake: got %d want 1_000_000_000", e.Address, e.Stake)
		}
	}
	for addr, seen := range wantAddrs {
		if !seen {
			t.Errorf("missing validator %q in localnet seed", addr)
		}
	}
}
