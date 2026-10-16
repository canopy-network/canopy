package contract

import "math/big"

// ARCM Section 14 -- Interest Rate Model. Governance-bounded parameters,
// currently hardcoded to their launch defaults (ARCM Section 15's table)
// since the governance-parameter-store read path ({22}) does not exist yet
// -- same category of gap as MIN_DEPOSIT (AYIS Section 13), flagged
// explicitly rather than silently assumed permanent. Values in bps.
const (
	uOptimalBps = 8000  // 80%, ARCM Section 15 default
	baseRateBps = 200   // 2%
	slope1Bps   = 800   // 8%
	slope2Bps   = 10000 // 100%
	bpsScale    = 10000
)

// ComputeBorrowRate implements ARCM Section 14's kinked utilization formula.
// utilizationBps = total_borrowed / total_supplied, in bps (0-10000+,
// though utilization above 100% should not occur if total_supplied's write
// path, ARCM Section 19.2.1b, is correctly enforced elsewhere). Returns
// annual_rate_bps.
//
// if U <= U_optimal: borrow_rate = BASE_RATE + (U / U_optimal) * SLOPE_1
// else:              borrow_rate = BASE_RATE + SLOPE_1 + ((U - U_optimal) / (1 - U_optimal)) * SLOPE_2
func ComputeBorrowRate(utilizationBps uint64) uint64 {
	if utilizationBps <= uOptimalBps {
		return baseRateBps + (utilizationBps*slope1Bps)/uOptimalBps
	}
	excessUtilization := utilizationBps - uOptimalBps
	remainingRange := uint64(bpsScale - uOptimalBps)
	return baseRateBps + slope1Bps + (excessUtilization*slope2Bps)/remainingRange
}

// ComputeUtilizationBps returns total_borrowed / total_supplied in bps,
// floor-rounded (AYIS Section 10.2's rounding table). Returns 0 if
// total_supplied is 0, matching AYIS Section 7 Step 3's explicit skip
// condition -- the caller is responsible for checking total_supplied == 0
// separately if it needs to skip accrual entirely rather than compute a
// meaningless 0% utilization.
func ComputeUtilizationBps(totalBorrowed, totalSupplied uint64) uint64 {
	if totalSupplied == 0 {
		return 0
	}
	return (totalBorrowed * bpsScale) / totalSupplied
}

// AnnualRateToPerBlockRateRay converts an annual rate in bps to a per-block
// rate scaled by RAY (AYIS Section 3, Section 5).
//
// [CORRECTED] AYIS Section 13 states BLOCKS_PER_YEAR = 15,768,000, derived
// from an assumed 2-second block time (365*24*3600/2). Arbor's actual,
// confirmed block time is 20 seconds, not 2 -- a 10x discrepancy. Left as
// originally spec'd, this constant would have understated every accrued
// interest amount by roughly 10x (per-block rate 10x too small, applied
// at a block cadence 10x slower than assumed -- both errors compound in
// the same direction). Corrected here to the true value:
// 365 * 24 * 3600 / 20 = 1,576,800. AYIS's own spec text still states the
// old 2s-derived figure -- this is a known, disclosed code/spec mismatch;
// the spec document itself should be corrected to match in a future
// revision, but the live constant here is what actually governs accrual,
// so it takes priority over the stale spec figure.
func AnnualRateToPerBlockRateRay(annualRateBps uint64) *big.Int {
	const blocksPerYear = 1_576_800 // 20s block time: 365*24*3600/20
	numerator := new(big.Int).Mul(big.NewInt(int64(annualRateBps)), RAY)
	numerator.Div(numerator, big.NewInt(bpsScale))
	numerator.Div(numerator, big.NewInt(blocksPerYear))
	return numerator
}
