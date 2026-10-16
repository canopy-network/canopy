package contract

import "math/big"

// SumLenderBalancesInMarket computes the exact aggregate current-value sum
// of every lender's balance in a market, in O(1) time, without iterating
// individual LenderPosition records.
//
// [AYIS Section 5.4.2, H2] This is exact, not an approximation: given every
// lender in a market shares the same s_rate and loss_factor,
// sum_i(balance_i) == total_shares_outstanding * s_rate * loss_factor /
// RAY^2 algebraically. Maintaining total_shares_outstanding incrementally
// (MintShares() increments it, RedeemShares() decrements it -- both
// already implemented and guarded, deposit.go/withdraw.go) makes this an
// O(1) read against the {26} SupplyIndexRecord plus the {27} loss_factor,
// rather than an O(n) scan over every lender position in the market.
//
// This is the foundational read Layer 4's remaining pieces
// (ApplyLossFactor, WillExhaustThisBlock) both depend on: it is the
// denominator ApplyLossFactor() compares an incoming bad_debt amount
// against to decide whether a market is merely haircut or fully
// exhausted (AYIS Section 5.4.3), and it is the identical comparison
// WillExhaustThisBlock() (ARCM v3.11.1 Section 9.3b Rule 3, AYIS v1.11.1
// Section 7 Step 8) performs one block earlier as a lookahead, against
// this same value, so that a market's own same-block exhaustion is
// correctly detected before AccrueInterest's Step 8 branch decision.
//
// [AYIS Section 4.3, J2 precedent] Carries the identical BitLen() cast-
// safety guard MintShares()/RedeemShares() already apply to their own
// big.Int -> uint64 casts at this exact type of boundary -- this
// function's own output is a state-writing/decision-making boundary in
// the same sense theirs is, even though it does not itself write state.
func SumLenderBalancesInMarket(c *Contract, marketID string) (uint64, *PluginError) {
	sRate, totalSharesOutstanding, found, pErr := GetSupplyIndex(c, marketID)
	if pErr != nil {
		return 0, pErr
	}
	if !found {
		// Unreachable in practice: create_market always initializes {26}
		// (Section 4.5's zero-init contract, s_rate=RAY,
		// total_shares_outstanding=0). Guarded explicitly rather than
		// assumed, per this project's established standard.
		return 0, ErrMarketNotFound(marketID)
	}

	lossFactor, found, pErr := GetLossFactor(c, marketID)
	if pErr != nil {
		return 0, pErr
	}
	if !found {
		// Unreachable in practice: create_market always initializes {27}
		// to RAY (Section 4.5). Same guard discipline as above.
		return 0, ErrMarketNotFound(marketID)
	}

	balance := new(big.Int).SetUint64(totalSharesOutstanding)
	balance.Mul(balance, sRate)
	balance.Mul(balance, lossFactor)
	balance.Div(balance, new(big.Int).Mul(RAY, RAY))

	// [AYIS Section 4.3, J2] Cast-safety guard on this function's own
	// output, matching MintShares()/RedeemShares()'s existing discipline
	// at the identical big.Int -> uint64 boundary shape.
	if balance.BitLen() > 64 {
		return 0, ErrShareOverflow(marketID, totalSharesOutstanding, balance.String())
	}

	return balance.Uint64(), nil
}
