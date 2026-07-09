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
// rate scaled by RAY (AYIS Section 3, Section 5). BLOCKS_PER_YEAR = 15,768,000
// (2s block time, immutable, AYIS Section 13).
func AnnualRateToPerBlockRateRay(annualRateBps uint64) *big.Int {
	const blocksPerYear = 15_768_000
	numerator := new(big.Int).Mul(big.NewInt(int64(annualRateBps)), RAY)
	numerator.Div(numerator, big.NewInt(bpsScale))
	numerator.Div(numerator, big.NewInt(blocksPerYear))
	return numerator
}
