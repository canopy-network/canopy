package canoliq

import (
	"testing"
)

// TestM5VirtualOffsetFirstDepositOneToOne confirms the +1 virtual offset still
// mints ~1:1 on the very first deposit (empty pool) — the guard must not change
// the legitimate first-depositor experience.
func TestM5VirtualOffsetFirstDepositOneToOne(t *testing.T) {
	if got := computeMint(1_000_000, 0, 0); got != 1_000_000 {
		t.Fatalf("first deposit mint: got %d want 1_000_000 (1:1)", got)
	}
	// A single uCNPY first deposit mints a single share — but the virtual unit
	// means it never owns 100% of an inflatable empty pool.
	if got := computeMint(1, 0, 0); got != 1 {
		t.Fatalf("dust first deposit mint: got %d want 1", got)
	}
}

// TestM5NoRoundingProfitOnDepositRedeem is the core M5 property: depositing then
// immediately redeeming (no rewards in between) can never return more than was
// deposited. Flooring on both sides plus the virtual offset guarantees a
// deposit/redeem round-trip is value-non-increasing, so a first depositor
// cannot extract value from later depositors via share-price rounding.
func TestM5NoRoundingProfitOnDepositRedeem(t *testing.T) {
	cases := []struct {
		amount, ccnpy, pooled uint64
	}{
		{1, 0, 0},                          // first deposit, 1 uCNPY
		{1_000_000, 0, 0},                  // first deposit, 1 CNPY
		{3, 1_000_000, 9_000_000},          // tiny deposit into a high-rate pool
		{1234, 5_000, 7_777},               // odd ratio
		{999_999, 1_000_000, 1_000_001},    // near-parity
		{50_000_000, 12_345_678, 98_765_4}, // large deposit, lopsided pool
	}
	for _, tc := range cases {
		minted := computeMint(tc.amount, tc.ccnpy, tc.pooled)
		// Redeem the freshly minted shares against the post-deposit pool.
		newCcnpy := tc.ccnpy + minted
		newPooled := tc.pooled + tc.amount
		returned := computeRedeem(minted, newCcnpy, newPooled)
		if returned > tc.amount {
			t.Fatalf("round-trip profit: deposit %d (ccnpy=%d pooled=%d) -> mint %d -> redeem %d > %d",
				tc.amount, tc.ccnpy, tc.pooled, minted, returned, tc.amount)
		}
	}
}
