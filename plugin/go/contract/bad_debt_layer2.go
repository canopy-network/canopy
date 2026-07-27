package contract

import "math/big"

// bad_debt_layer2.go implements Layer 2 of ARCM Section 9.2's four-layer
// bad-debt waterfall: reserve fund draw-down.
//
// DESIGN DECISION, all-or-nothing gate (not a partial drain + residual
// pass-through): ARCM Section 9.2's own Layer 2 row states the mechanism
// literally as "if bad_debt <= R_fund[market]: R_fund -= bad_debt" -- this
// is a binary gate, not a partial-cover-then-continue step.
//
// [STALE COMMENT CORRECTED] This block previously claimed Layer 4
// (SumLenderBalancesInMarket, ApplyLossFactor, EnqueueLossFactorApplication,
// ProcessLossFactorQueue) "do not exist yet" and that "no treasury
// key/accessor exists." As of this correction, verified directly against
// the real files:
//   - Layer 4 (SumLenderBalancesInMarket, ApplyLossFactor,
//     EnqueueLossFactorApplication, PeekLossFactorQueue,
//     DequeueLossFactorApplication): EXIST and ARE wired in --
//     liquidate_position.go calls ApplyLossFactor on a Layer 2 miss.
//     ProcessLossFactorQueue (BeginBlock drain) is still unbuilt.
//   - Treasury (T_fund): key/accessors NOW EXIST (PrefixTreasury,
//     KeyForTreasury, GetTreasury, SetTreasuryTry, SetTreasury in
//     state_keys.go / state_accessors.go), added after this file's
//     creation. Layer 3 itself -- an actual draw-down function analogous
//     to this file's own Layer2DrawDown, and a caller wiring it into the
//     waterfall between Layer 2 and Layer 4 -- is STILL NOT BUILT. Having
//     accessors is not the same as having Layer 3; do not assume Layer 3
//     is wired just because T_fund can now be read and written.
//
// A partial drain here would leave R_fund silently weakened against a
// FUTURE bad-debt event while THIS event's shortfall has nowhere real to
// go and nothing tracking it -- the exact "looks done but silently isn't"
// failure mode this codebase's own established discipline (see
// liquidate_position.go's ErrLiquidationBadDebt disclosure) exists to
// avoid. Full-cover-or-noop is therefore the correct choice given the
// current state of this codebase, not merely the literal-spec-reading
// choice -- when Layer 3 is eventually built, this function's contract
// (covered bool, R_fund only mutated on full cover) can remain unchanged;
// only the caller's handling of covered == false needs to gain a Layer 3
// hand-off instead of falling straight through to Layer 4.
//
// UNIT CONTRACT: badDebtNative MUST already be converted to the market's
// debt-asset native units (same unit R_fund itself is stored in -- see
// this session's verified re-derivation: repay.go's Insolvent-routing leg
// and interest_accrual.go's Step 10 reserve_cut both credit R_fund in
// native debt-asset units, never USD). This function does NOT perform any
// price conversion and does NOT call ResolvePrice -- callers own their own
// conversion and their own rounding-direction choice (Section 10.2 assigns
// rounding per-formula, not globally), matching how liquidate_position.go's
// Step 4/5 already do their own inline price math rather than delegating
// it to a shared helper. See liquidate_position.go for a caller computing
// badDebtNative from a collateral-quantity shortfall via
// collateralPrice/debtPrice, ceiling-rounded so the reserve is never
// undercharged.
//
// ENCODING CONTEXT: this function is DeliverTx-context only (called from
// liquidate_position.go's Step 5, a real transaction that can revert) --
// it uses SetReserveFund (the reverting EncodeUint128 wrapper), NOT
// SetReserveFundTry (BeginBlock-context, freezes the market instead of
// reverting). Do not call this from BeginBlock code without re-deriving
// which encoding response applies, per Principle 14.
//
// Layer2DrawDown attempts to fully cover badDebtNative from
// R_fund[marketID]. Returns covered=true and mutates R_fund (write
// persisted) only if R_fund >= badDebtNative. Returns covered=false and
// leaves R_fund completely untouched (no read-then-abandoned-write; the
// comparison happens before any write is attempted) if R_fund is
// insufficient. pErr is non-nil only for a genuine state-layer failure
// (read error, encode-overflow revert, write error) -- insufficient
// coverage is a normal, expected return value (covered=false), not a
// PluginError, since it is the caller's decision what to do next (as of
// this file's creation: liquidate_position.go treats it as a hard reject,
// matching Layer 1's existing ErrLiquidationBadDebt behavior).
func Layer2DrawDown(c *Contract, marketID string, badDebtNative uint64) (covered bool, pErr *PluginError) {
	rFund, found, err := GetReserveFund(c, marketID)
	if err != nil {
		return false, err
	}
	if !found {
		rFund = big.NewInt(0)
	}

	badDebtBig := new(big.Int).SetUint64(badDebtNative)
	if rFund.Cmp(badDebtBig) < 0 {
		// R_fund insufficient to fully cover -- all-or-nothing gate (see
		// doc comment above). R_fund is NOT mutated; no write is attempted.
		return false, nil
	}

	newRFund := new(big.Int).Sub(rFund, badDebtBig)
	if wErr := SetReserveFund(c, marketID, newRFund); wErr != nil {
		return false, wErr
	}
	return true, nil
}
