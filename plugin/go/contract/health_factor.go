package contract

import "math/big"

// ComputeHealthFactorScaled implements ARCM Section 5's exact formula:
//
//HF_scaled = (Q_c * P_c * LTV_liq_bps * 1_000_000) / (Q_d * P_d * 10_000)
//
// A position is liquidatable when HF_scaled <= 1_000_000 (i.e. HF <= 1.0).
// math/big throughout per ARCM Section 17.4 -- Q_c * P_c * LTV_liq_bps
// alone can exceed uint64 range for realistic collateral quantities
// against high-priced assets.
//
// PRECONDITION: qDebt and pDebt must both be > 0. HF is undefined (not
// "infinitely healthy", not zero) when there is no debt -- callers MUST
// check for zero debt themselves and skip calling this function entirely
// in that case, treating "no debt" as trivially safe for any collateral
// action rather than routing it through this formula. This mirrors
// MaxBorrowQuantity's existing precondition style (tier_params.go).
func ComputeHealthFactorScaled(qCollateral, pCollateral, ltvLiqBps, qDebt, pDebt uint64) *big.Int {
if qDebt == 0 || pDebt == 0 {
// Should never be reached given the documented precondition; return
// a maximally-safe sentinel (an enormous HF) rather than divide by
// zero, consistent with this codebase's "never panic" discipline
// (ARCM P12) even at a call site that should be unreachable.
return new(big.Int).Lsh(big.NewInt(1), 256)
}
numerator := new(big.Int).Mul(new(big.Int).SetUint64(qCollateral), new(big.Int).SetUint64(pCollateral))
numerator.Mul(numerator, new(big.Int).SetUint64(ltvLiqBps))
numerator.Mul(numerator, big.NewInt(1_000_000))
denominator := new(big.Int).Mul(new(big.Int).SetUint64(qDebt), new(big.Int).SetUint64(pDebt))
denominator.Mul(denominator, big.NewInt(10_000))
return new(big.Int).Div(numerator, denominator)
}

// HFLiquidatableThresholdScaled is the exact ARCM Section 5 boundary: a
// position with HF_scaled <= this value is liquidatable. Exported as a
// named constant rather than a bare 1_000_000 literal at every call site,
// so its meaning is self-documenting wherever it's compared against.
var HFLiquidatableThresholdScaled = big.NewInt(1_000_000)
