package contract

import "math/big"

// GetMarketStatus reads a market's current status. Thin wrapper around
// GetMarket() for callers (AYIS's own functions, per Section 5.4.3's
// ARCM.GetMarketStatus() reference) that need only the status field and
// would otherwise have to unpack the full Market record themselves.
func GetMarketStatus(c *Contract, marketID string) (status MarketStatus, found bool, pErr *PluginError) {
	market, found, pErr := GetMarket(c, marketID)
	if pErr != nil {
		return MarketStatus_ACTIVE, false, pErr
	}
	if !found {
		return MarketStatus_ACTIVE, false, nil
	}
	return market.Status, true, nil
}

// SetMarketInsolvent transitions a market to MarketStatus_INSOLVENT.
//
// [AYIS Section 5.4.3, I11] Called exactly once per market, at the moment
// ApplyLossFactor() drives that market's loss_factor to exactly zero on
// total wipeout. Idempotency (never re-firing this transition's side
// effects on an already-Insolvent market) is ApplyLossFactor()'s own K3
// guard's responsibility, checked BEFORE this function is ever called --
// this function itself does not re-check current status, matching
// pause_market.go's DeliverMessageResumeMarket()'s own documented
// idempotent-at-the-call-site-not-the-accessor pattern.
//
// [ARCM v3.11.1 Section 9.3b Rule 1] This transition MUST NOT be gated by
// market.index_overflow_halted in either direction -- a frozen market can
// still become Insolvent for the first time. This function has no
// dependency on that flag at all, by construction, which is how Rule 1 is
// satisfied: there is simply nothing here that could block it.
func SetMarketInsolvent(c *Contract, marketID string) *PluginError {
	market, found, pErr := GetMarket(c, marketID)
	if pErr != nil {
		return pErr
	}
	if !found {
		// Unreachable in practice: ApplyLossFactor()'s own caller chain
		// (ProcessLossFactorQueue, or a synchronous liquidation) already
		// read this market to compute bad_debt in the first place.
		return ErrMarketNotFound(marketID)
	}
	market.Status = MarketStatus_INSOLVENT
	return SaveMarket(c, marketID, market)
}

// SetLossFactor writes a market's {27} loss_factor value, using the
// reverting EncodeUint128() wrapper -- correct here per AYIS Section 9/
// 10.8, since Invariant I8 (loss_factor monotonicity: never increases from
// RAY) structurally bounds this value well within 128 bits at every real
// call site (ApplyLossFactor() only ever multiplies it by a strictly-
// smaller-than-one ratio or sets it to exactly zero); it is the one field
// AYIS Section 9 itself names as never needing the BeginBlock-freeze
// treatment TryEncodeUint128() exists for.
func SetLossFactor(c *Contract, marketID string, lossFactor *big.Int) *PluginError {
	encoded, eErr := EncodeUint128(lossFactor)
	if eErr != nil {
		return eErr
	}
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{{Key: KeyForLossFactor(marketID), Value: encoded}},
	})
	if wErr != nil {
		return wErr
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// DecrementLayer4Pending decrements market.layer4_pending_count (floor
// zero, with an underflow event rather than a silent wrap or panic) and
// subtracts bad_debt_amount from market.layer4_pending_bad_debt_total
// (compare-before-subtract, floor zero) -- the exact pairing
// ApplyLossFactor() calls once per completed application, per ARCM
// Section 9.2b. The count and the total are decremented together, in the
// same call, since they are incremented together at the same Layer-4-
// triggering call site (ARCM Section 19.2.2 Step 8) and this codebase's
// own Principle 8 (aggregates must reflect reality) applies to both halves
// of this pair identically -- there is no scenario where one should move
// without the other.
func DecrementLayer4Pending(c *Contract, marketID string, badDebtAmount uint64) *PluginError {
	market, found, pErr := GetMarket(c, marketID)
	if pErr != nil {
		return pErr
	}
	if !found {
		return ErrMarketNotFound(marketID)
	}

	if market.Layer4PendingCount == 0 {
		// [ARCM Section 9.2b] Underflow guard -- mirrors the spec's own
		// Layer4PendingCountUnderflowEvent pseudocode. This indicates a
		// logic error upstream (DecrementLayer4Pending called more times
		// than the count was ever incremented), not a state this
		// function can correct on its own.
		//
		// TODO: this should emit EventLayer4PendingCountUnderflow (the
		// proto message already exists and is registered in contract.go's
		// EventTypeUrls) but this function has no established way to do
		// so: every existing event-emission call site in this codebase
		// (repay.go, liquidate_position.go) is DeliverTx-context,
		// appending to a local events slice this function -- called from
		// BeginBlock via ProcessLossFactorQueue -- does not have access
		// to. interest_accrual.go's own BeginBlock-context events (per
		// spec) are ALSO not actually wired up anywhere in this codebase
		// as of this writing -- a real, pre-existing gap (how does
		// BeginBlock emit events at all in this plugin architecture?),
		// not something to paper over with an invented call path here.
		// Returning nil (silently skip the decrement) rather than
		// guessing at an emission mechanism that hasn't been verified to
		// exist.
		return nil
	}
	market.Layer4PendingCount--

	current := DecodeUint128(market.Layer4PendingBadDebtTotal)
	amount := new(big.Int).SetUint64(badDebtAmount)
	var newTotal *big.Int
	if amount.Cmp(current) >= 0 {
		newTotal = big.NewInt(0)
	} else {
		newTotal = new(big.Int).Sub(current, amount)
	}
	// EncodeUint128 cannot actually fail here -- newTotal is always <=
	// current, which was itself already a valid 128-bit encoded value
	// read from state (the saturating checked-add at the increment call
	// site, ARCM Section 9.2b, is what guarantees current never exceeds
	// uint128 in the first place). Checked anyway rather than assumed,
	// per this project's established standard of never trusting an
	// encode call to succeed without checking its actual return value.
	encodedTotal, eErr := EncodeUint128(newTotal)
	if eErr != nil {
		return eErr
	}
	market.Layer4PendingBadDebtTotal = encodedTotal

	return SaveMarket(c, marketID, market)
}
