package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// TestL3TxFeesBypassRewardSplitAndRouteToTreasury covers L3: protocol tx fees
// accrued into the committee pool are excluded from the 12% reward split and
// routed whole to the DAO treasury, instead of being distributed as if they
// were staking reward.
//
// Setup: the committee pool holds 1_010_000 = a 1_000_000 Canopy reward + a
// 10_000 accrued tx fee. Only the 1_000_000 reward is fee-split; the 10_000
// lands in treasury untouched.
func TestL3TxFeesBypassRewardSplitAndRouteToTreasury(t *testing.T) {
	c, s := newTestCanoliq()
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	const reward = 1_000_000
	const txFee = 10_000
	pool := &contract.Pool{Id: c.Config.ChainId, Amount: reward + txFee}
	pBz, _ := contract.Marshal(pool)
	s.set(contract.KeyForFeePool(c.Config.ChainId), pBz)
	s.set(KeyForTxFeeAccrual(), EncodeUint64(txFee))

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("ProcessRewards: %v", err)
	}

	// Reward portion (1_000_000) splits as: fee 120_000 → user 48_000 rebate +
	// treasury 36_000 + validators 18_000 + buyback 18_000; net-to-users 880_000.
	// userSlice = 880_000 + 48_000 = 928_000 (escrow + pooled).
	// Insurance skim = 5% of the 36_000 treasury slice = 1_800.
	// Treasury = (36_000 − 1_800) + 10_000 tx-fee = 44_200.
	g := loadGlobals(t, s)
	if g.TotalPooledCnpy != 928_000 {
		t.Fatalf("pooled (reward userSlice only, tx-fee excluded): got %d want 928_000", g.TotalPooledCnpy)
	}
	if got := readEscrow(s); got != 928_000 {
		t.Fatalf("escrow: got %d want 928_000", got)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != 44_200 {
		t.Fatalf("treasury (reward slice − insurance + tx-fee): got %d want 44_200", got)
	}
	if got := DecodeUint64(s.get(KeyForInsurancePool())); got != 1_800 {
		t.Fatalf("insurance (skim on reward treasury slice only): got %d want 1_800", got)
	}
	if got := DecodeUint64(s.get(KeyForBuybackPool())); got != 18_000 {
		t.Fatalf("buyback: got %d want 18_000", got)
	}
	// Accrual is zeroed after routing.
	if got := DecodeUint64(s.get(KeyForTxFeeAccrual())); got != 0 {
		t.Fatalf("tx-fee accrual not cleared: got %d want 0", got)
	}
}

// TestL3AccrueTxFeeAccumulates checks the central accrual helper sums fees and
// is a no-op for a zero fee.
func TestL3AccrueTxFeeAccumulates(t *testing.T) {
	c, s := newTestCanoliq()
	if err := c.accrueTxFee(10_000); err != nil {
		t.Fatalf("accrueTxFee: %v", err)
	}
	if err := c.accrueTxFee(0); err != nil { // no-op
		t.Fatalf("accrueTxFee(0): %v", err)
	}
	if err := c.accrueTxFee(5_000); err != nil {
		t.Fatalf("accrueTxFee: %v", err)
	}
	if got := DecodeUint64(s.get(KeyForTxFeeAccrual())); got != 15_000 {
		t.Fatalf("accrual: got %d want 15_000", got)
	}
}
