package contract

import "math"

// SafeInt64FromUint64 converts a uint64 to a signed int64 safely, returning
// ok=false rather than allowing Go's raw int64(x) conversion to silently
// wrap a value larger than math.MaxInt64 into a negative number.
//
// [ARCM Section 19.3, C1] This closes a real, if narrow, fund-accounting
// corruption path: prior spec versions (through v3.10) called
// applyDebtDelta(market_id, int64(borrow_amount)) directly at each call
// site. A borrow_amount exceeding math.MaxInt64 (~9.22e18, or ~9.22
// billion tokens under this codebase's 9-decimal convention) would wrap to
// a negative int64 under Go's conversion rules -- not a panic, not an
// error, just a wrong sign. applyDebtDelta() would then take its
// delta < 0 branch and DECREASE total_borrowed on what was meant to be a
// borrow that increases it. Implausible at today's realistic amounts, but
// a genuine correctness gap for any future deployment at scale, and the
// exact class of "silently wrong beats loudly rejected" violation this
// codebase's other cast guards (AYIS J2's BitLen() checks, uint128.go's
// EncodeUint128) already treat as non-negotiable.
func SafeInt64FromUint64(u uint64) (result int64, ok bool) {
	if u > math.MaxInt64 {
		return 0, false
	}
	return int64(u), true
}

// applyDebtDelta is the SINGLE mandatory write path for every
// total_borrowed mutation (ARCM Section 19.3, Principle 9/F6): borrow
// (positive delta), repay and liquidation (negative delta). Centralizing
// the overflow/underflow guard HERE, rather than parallel to each call
// site, is itself part of what C1/v3.10 fixes -- a guard written beside
// this function instead of inside it can be silently bypassed by a future
// caller; a guard inside it cannot.
//
// Takes *Market by pointer and mutates market.TotalBorrowed in place;
// caller is responsible for the actual SaveMarket() write, matching this
// codebase's existing GetMarket/mutate/SaveMarket pattern rather than
// this function owning its own state I/O.
// [NEW] clampedFrom is 0 unless the decrement branch below clamps
// market.TotalBorrowed to zero on an oversized decrease -- in which case
// it carries the PRE-CLAMP value, so the caller (repay.go, liquidate_position.go)
// can emit EventTotalBorrowedDustClamp with the real amount clamped away,
// mirroring EventTotalSuppliedDustClamp's fix (ARCM v3.11.1 Section III.6)
// for the identical silent-clamp failure class. This adds one named return
// value; the function's existing *PluginError-only error contract for
// callers that don't care about the clamp (borrow.go's increase branch
// never clamps) is unchanged -- they can simply discard it with _.
func applyDebtDelta(market *Market, marketID string, delta int64) (clampedFrom uint64, pErr *PluginError) {
	if delta > 0 {
		increase := uint64(delta)
		if increase > (^uint64(0) - market.TotalBorrowed) {
			return 0, ErrTotalBorrowedOverflowCentralized(marketID, market.TotalBorrowed, increase)
		}
		market.TotalBorrowed += increase
	} else if delta < 0 {
		// delta is negative and within int64 range by construction (it was
		// only ever produced by a successful SafeInt64FromUint64 call
		// followed by a caller-side negation) -- uint64(-delta) is safe:
		// -delta is positive and representable, since delta != math.MinInt64
		// (a value that only int64(0) is negated INTO, per SafeInt64FromUint64
		// never returning an already-negative result to negate in the first
		// place).
		decrease := uint64(-delta)
		if decrease >= market.TotalBorrowed {
			// [NEW] Capture the pre-clamp value BEFORE zeroing it, same
			// discipline as H4's fix for total_supplied/total_shares_outstanding.
			clampedFrom = market.TotalBorrowed
			market.TotalBorrowed = 0
		} else {
			market.TotalBorrowed -= decrease
		}
	}
	// delta == 0: no-op, matches ARCM's own pseudocode (no else branch for zero).
	return clampedFrom, nil
}
