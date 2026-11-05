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
// [FIXED] Guarded -- see ErrScaledDebtOverflow. The v1.11-era disclosed
// carve-out reasoning (no amplification path analogous to
// MintShares()/RedeemShares()/SumLenderBalancesInMarket()) was a design
// assumption, not a proven bound; closed per Arbor Handoff Part 2, item 2,
// matching the identical BitLen() > 64 guard pattern already used in
// deposit.go, withdraw.go, and liquidate_position.go.
func ScaledDebt(pos *BorrowerPosition, bIndexNow *big.Int) (uint64, *PluginError) {
	if pos.DebtPrincipal == 0 {
		return 0, nil
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

	// [NEW] Cast-safety guard -- same pattern as deposit.go's sharesBig
	// guard, withdraw.go's tokensBig guard, and liquidate_position.go's
	// collateralSeized/badDebtNative guards.
	if numerator.BitLen() > 64 {
		return 0, ErrScaledDebtOverflow(pos.MarketId, pos.DebtPrincipal, numerator.String())
	}
	return numerator.Uint64(), nil
}
