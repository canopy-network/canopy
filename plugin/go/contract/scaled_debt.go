package contract

import "math/big"

// ScaledDebt computes a borrower position's current owed debt, scaled by
// the market's borrow index growth since the position was opened (AYIS
// Section 6): D(t) = ceil(debt_principal * B_index(t) / borrow_index_at_open).
// Ceiling division favors the protocol (P14) -- the borrower owes slightly
// more, never less, due to rounding. This MUST be used everywhere "current
// debt" is needed; pos.DebtPrincipal alone is only the stored principal at
// last write, not the current scaled amount (ARCM Section 2.2's mandatory
// rule).
//
// No big.Int -> uint64 overflow guard is applied at the final cast here,
// unlike AYIS's MintShares()/RedeemShares()/SumLenderBalancesInMarket().
// This deliberately follows AYIS v1.11 Section 10.6's own explicit
// carve-out: ScaledDebt()'s cast has no 1/loss_factor-style amplification
// path and is bounded by realistic debt-principal magnitudes, and is
// explicitly NOT listed as an at-risk boundary requiring the BitLen() guard.
// If a future review identifies a comparable amplification path on the
// borrower side, the same guard would need to be added here.
func ScaledDebt(pos *BorrowerPosition, bIndexNow *big.Int) uint64 {
if pos.DebtPrincipal == 0 {
return 0
}
borrowIndexAtOpen := DecodeUint128(pos.BorrowIndexAtOpen)

// Defensive-only guard: a zero borrow_index_at_open should never occur
// in practice -- borrow.go always writes this field from a live,
// just-read B_index at the moment a position is opened or consolidated.
// This only protects ScaledDebt() itself from a division-by-zero if a
// position record were ever corrupted or malformed; it is not an
// expected code path.
if borrowIndexAtOpen.Sign() == 0 {
borrowIndexAtOpen = big.NewInt(1)
}

numerator := new(big.Int).Mul(new(big.Int).SetUint64(pos.DebtPrincipal), bIndexNow)
// Ceiling division: (numerator + divisor - 1) / divisor
numerator.Add(numerator, borrowIndexAtOpen)
numerator.Sub(numerator, big.NewInt(1))
numerator.Div(numerator, borrowIndexAtOpen)

return numerator.Uint64()
}
