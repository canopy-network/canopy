package contract

// checkMarketNotPaused is the single shared admission guard for
// MarketStatus_PAUSED, called by every handler that creates or changes
// debt/fund exposure in a market: deposit, withdraw, borrow, repay,
// withdraw_collateral, liquidate_position.
//
// DELIBERATELY NOT called by deposit_collateral. collateral.go's own
// existing comment on that handler states "none of Market's flags gate
// deposit_collateral" -- confirmed correct on inspection, not an oversight
// this function should silently fix. Depositing collateral with no
// corresponding borrow creates no new debt, no new protocol liability, and
// no new exposure; it is a strictly defensive action (a borrower adding
// safety margin to an existing position). Blocking it during a pause would
// actively harm a borrower trying to protect against liquidation risk
// during the exact kind of stress event a pause is meant to respond to,
// with no offsetting safety benefit. This mirrors, in spirit, ARCM Section
// 9.2 J1's and Section 9.3a's own reasoning for why Insolvent and
// index_overflow_halted leave certain exit/defensive paths open rather
// than blocking everything uniformly -- applied here to a case those two
// sections don't cover, since Section 13 itself doesn't specify pause's
// admission set.
//
// DESIGN DECISION, DISCLOSED: for every OTHER handler in the list above,
// ARCM Section 13 states pause/resume is a risk-committee emergency action
// but does not itself specify which transaction types a paused market
// blocks. In the absence of a stated narrower rule for these six, this
// implementation takes the conservative reading: a governance-triggered
// pause is an emergency "stop new exposure and fund movement until this
// market is understood" action. Revisit if a future ARCM version defines
// pause's admission set explicitly.
func checkMarketNotPaused(market *Market, marketID string) *PluginError {
	if market.Status == MarketStatus_PAUSED {
		return ErrMarketPaused(marketID)
	}
	return nil
}
