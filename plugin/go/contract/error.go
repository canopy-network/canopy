package contract

import (
	"fmt"
	"reflect"
)

/* This file contains contract level PluginErrors */

const DefaultModule = "plugin"

// NewError() creates a plugin error
func NewError(code uint64, module, message string) *PluginError {
	return &PluginError{Code: code, Module: module, Msg: message}
}

// Error() implements the errors interface
func (p *PluginError) Error() string {
	return fmt.Sprintf("\nModule:  %s\nCode:    %d\nMessage: %s", p.Module, p.Code, p.Msg)
}

func ErrPluginTimeout() *PluginError {
	return NewError(1, DefaultModule, "a plugin timeout occurred")
}

func ErrMarshal(err error) *PluginError {
	return NewError(2, DefaultModule, fmt.Sprintf("marshal() failed with err: %s", err.Error()))
}

func ErrUnmarshal(err error) *PluginError {
	return NewError(3, DefaultModule, fmt.Sprintf("unmarshal() failed with err: %s", err.Error()))
}

func ErrFailedPluginRead(err error) *PluginError {
	return NewError(4, DefaultModule, fmt.Sprintf("a plugin read failed with err: %s", err.Error()))
}

func ErrFailedPluginWrite(err error) *PluginError {
	return NewError(5, DefaultModule, fmt.Sprintf("a plugin write failed with err: %s", err.Error()))
}

func ErrInvalidPluginRespId() *PluginError {
	return NewError(6, DefaultModule, "plugin response id is invalid")
}

func ErrUnexpectedFSMToPlugin(t reflect.Type) *PluginError {
	return NewError(7, DefaultModule, fmt.Sprintf("unexpected FSM to plugin: %v", t))
}

func ErrInvalidFSMToPluginMMessage(t reflect.Type) *PluginError {
	return NewError(8, DefaultModule, fmt.Sprintf("invalid FSM to plugin: %v", t))
}

func ErrInsufficientFunds() *PluginError {
	return NewError(9, DefaultModule, "insufficient funds")
}

func ErrFromAny(err error) *PluginError {
	return NewError(10, DefaultModule, fmt.Sprintf("fromAny() failed with err: %s", err.Error()))
}

func ErrInvalidMessageCast() *PluginError {
	return NewError(11, DefaultModule, "the message cast failed")
}

func ErrInvalidAddress() *PluginError {
	return NewError(12, DefaultModule, "address is invalid")
}

func ErrInvalidAmount() *PluginError {
	return NewError(13, DefaultModule, "amount is invalid")
}

func ErrTxFeeBelowStateLimit() *PluginError {
	return NewError(14, DefaultModule, "tx.fee is below state limit")
}

/* Arbor-owned error codes, starting at 195 per ARCM v3.11/AYIS v1.11 combined
   audit Part I.8 platform-grounding note (codes 1-14 are Canopy-reserved). */

const ArborModule = "arbor"

func ErrInvalidMarketID(err error) *PluginError {
	return NewError(195, ArborModule, fmt.Sprintf("invalid market_id: %s", err.Error()))
}

func ErrMarketAlreadyExists(marketID string) *PluginError {
	return NewError(196, ArborModule, fmt.Sprintf("market %q already exists", marketID))
}

func ErrInvalidAssetTier() *PluginError {
	return NewError(197, ArborModule, "asset_tier must be 0-4 (ARCM Section 3)")
}

func ErrReserveFactorOutOfBounds() *PluginError {
	return NewError(198, ArborModule, "reserve_factor_bps must be 200-3000 (AYIS Section 13)")
}

func ErrTreasuryCutOutOfBounds() *PluginError {
	return NewError(242, ArborModule, "treasury_cut_bps must be 25-150")
}

func ErrUnauthorized() *PluginError {
	return NewError(199, ArborModule, "signer is not authorized for this action")
}

func ErrMarketNotFound(marketID string) *PluginError {
	return NewError(200, ArborModule, fmt.Sprintf("market %q not found", marketID))
}

func ErrUint128EncodingOverflow(value string) *PluginError {
	return NewError(201, ArborModule, fmt.Sprintf("value %s exceeds uint128 range at a DeliverTx-context encode boundary", value))
}

func ErrMarketInsolvent(marketID string) *PluginError {
	return NewError(202, ArborModule, fmt.Sprintf("market %q is insolvent (loss_factor == 0), AYIS Section 4.3 H1", marketID))
}

func ErrMarketIndexOverflowHalted(marketID string) *PluginError {
	return NewError(203, ArborModule, fmt.Sprintf("market %q is index-overflow-halted, ARCM Section 9.3a", marketID))
}

func ErrMarketLayer4Pending(marketID string) *PluginError {
	return NewError(204, ArborModule, fmt.Sprintf("market %q has a pending Layer 4 loss-factor application, ARCM Section 9.2b", marketID))
}

func ErrShareOverflow(marketID string, amount uint64, shares string) *PluginError {
	return NewError(205, ArborModule, fmt.Sprintf("market %q: shares value %s for amount %d exceeds uint64 range, AYIS Section 4.3 J2", marketID, shares, amount))
}

func ErrTotalSharesOverflow(marketID string, current uint64, shares string) *PluginError {
	return NewError(206, ArborModule, fmt.Sprintf("market %q: adding shares %s to total_shares_outstanding %d would overflow uint64, AYIS Section 4.3 L1", marketID, shares, current))
}

func ErrTokenOverflow(marketID string, shares uint64, tokens string) *PluginError {
	return NewError(207, ArborModule, fmt.Sprintf("market %q: token value %s for shares %d exceeds uint64 range, AYIS Section 4.4 J2", marketID, tokens, shares))
}

// ErrCollateralSeizedOverflow guards liquidate_position.go's
// collateralSeized big.Int -> uint64 cast (ARCM Section 8). Reachable if
// an oracle-submitted debtPrice/collateralPrice ratio is extreme enough
// to push ceil(RepayAmount * debtPrice * LIFBps / (collateralPrice *
// 10000)) past 64 bits -- session finding, previously unguarded.
func ErrCollateralSeizedOverflow(marketID string, repayAmount uint64, collateralSeized string) *PluginError {
	return NewError(239, ArborModule, fmt.Sprintf("market %q: collateralSeized value %s for repayAmount %d exceeds uint64 range, ARCM Section 8", marketID, collateralSeized, repayAmount))
}

// ErrBadDebtNativeOverflow guards liquidate_position.go's badDebtNative
// big.Int -> uint64 cast (ARCM Section 9.2, Layer 2 bad-debt path).
// Reachable under the same extreme oracle-price-ratio condition as
// ErrCollateralSeizedOverflow above -- session finding, previously
// disclosed as unguarded rather than fixed.
func ErrBadDebtNativeOverflow(marketID string, badDebtCollateral string, badDebtNative string) *PluginError {
	return NewError(240, ArborModule, fmt.Sprintf("market %q: badDebtNative value %s for badDebtCollateral %s exceeds uint64 range, ARCM Section 9.2", marketID, badDebtNative, badDebtCollateral))
}

func ErrTotalSuppliedOverflow(marketID string, current uint64, amount uint64) *PluginError {
	return NewError(208, ArborModule, fmt.Sprintf("market %q: adding amount %d to total_supplied %d would overflow uint64, ARCM Section 19.2.1b M2", marketID, amount, current))
}

func ErrInsufficientShares(marketID string, have uint64, want uint64) *PluginError {
	return NewError(209, ArborModule, fmt.Sprintf("market %q: position has %d shares, cannot redeem %d", marketID, have, want))
}

func ErrLenderPositionNotFound(marketID string, addr string) *PluginError {
	return NewError(210, ArborModule, fmt.Sprintf("no lender position for address %s in market %q", addr, marketID))
}

func ErrDepositBelowMinimum(amount uint64) *PluginError {
	return NewError(211, ArborModule, fmt.Sprintf("deposit amount %d is below MIN_DEPOSIT, AYIS Section 13", amount))
}

func ErrInsufficientSubmitters(marketID string, have, want int) *PluginError {
	return NewError(212, ArborModule, fmt.Sprintf("market %q: authorized_submitters has %d entries, need at least %d (ARCM Section 10, MinReporters)", marketID, have, want))
}

func ErrAssetNotInMarket(marketID, assetID string) *PluginError {
	return NewError(213, ArborModule, fmt.Sprintf("market %q does not reference asset %q as either its collateral or debt asset (ARCM Section 10)", marketID, assetID))
}

func ErrCollateralOverflow(marketID string, current uint64, amount uint64) *PluginError {
	return NewError(214, ArborModule, fmt.Sprintf("market %q: adding amount %d to collateral_quantity %d would overflow uint64", marketID, amount, current))
}

func ErrInvalidAssetID(err error) *PluginError {
	return NewError(215, ArborModule, fmt.Sprintf("invalid asset_id: %s", err.Error()))
}

func ErrTierOutOfRange(tier uint32) *PluginError {
	return NewError(216, ArborModule, fmt.Sprintf("tier %d invalid: set_asset_tier accepts 0-3 only (tier 4/Blacklisted cannot be set via this transaction, ARCM Section 3)", tier))
}

func ErrAssetTierNotFound(assetID string) *PluginError {
	return NewError(217, ArborModule, fmt.Sprintf("asset %q has no tier registry entry, cannot be used as collateral (ARCM Section 3, state_keys.go PrefixAssetTier)", assetID))
}

func ErrNoCollateralPosition(marketID string, addr []byte) *PluginError {
	return NewError(218, ArborModule, fmt.Sprintf("no collateral position for address %x in market %q -- deposit_collateral must run before borrow", addr, marketID))
}

func ErrPriceUnavailable(marketID, assetID string) *PluginError {
	return NewError(219, ArborModule, fmt.Sprintf("market %q: no resolvable oracle price for asset %q (ARCM Section 10, quorum/freshness not met)", marketID, assetID))
}

func ErrExceedsMaxLTV(marketID string, requestedDebt, maxBorrow uint64) *PluginError {
	return NewError(220, ArborModule, fmt.Sprintf("market %q: requested total debt %d exceeds max-LTV borrow capacity %d (ARCM Section 4)", marketID, requestedDebt, maxBorrow))
}

func ErrTotalBorrowedOverflow(marketID string, current uint64, amount uint64) *PluginError {
	return NewError(221, ArborModule, fmt.Sprintf("market %q: adding amount %d to total_borrowed/debt %d would overflow uint64", marketID, amount, current))
}

func ErrBorrowerPositionNotFound(marketID string, addr []byte) *PluginError {
	return NewError(222, ArborModule, fmt.Sprintf("no borrower position for address %x in market %q", addr, marketID))
}

func ErrTotalBorrowedUnderflow(marketID string, current uint64, amount uint64) *PluginError {
	return NewError(223, ArborModule, fmt.Sprintf("market %q: subtracting repaid amount %d from total_borrowed %d would underflow", marketID, amount, current))
}

func ErrMarketIndexNotInitialized(marketID string) *PluginError {
	return NewError(224, ArborModule, fmt.Sprintf("market %q: borrow index ({25}) not initialized -- create_market invariant violated", marketID))
}

func ErrAccountBalanceOverflow(addr []byte, current uint64, amount uint64) *PluginError {
	return NewError(225, ArborModule, fmt.Sprintf("account %x: adding amount %d to balance %d would overflow uint64", addr, amount, current))
}

func ErrInt64CastOverflow(context string, value uint64) *PluginError {
	return NewError(226, ArborModule, fmt.Sprintf("%s: value %d exceeds math.MaxInt64, cannot be safely represented as a signed delta (ARCM Section 19.3, C1)", context, value))
}

func ErrTotalBorrowedOverflowCentralized(marketID string, current uint64, increase uint64) *PluginError {
	return NewError(227, ArborModule, fmt.Sprintf("market %q: applyDebtDelta increase %d would overflow total_borrowed %d", marketID, increase, current))
}

func ErrInsufficientCollateral(marketID string, have uint64, want uint64) *PluginError {
	return NewError(228, ArborModule, fmt.Sprintf("market %q: position has %d collateral, cannot withdraw %d", marketID, have, want))
}

func ErrWithdrawalExceedsHF(marketID string, resultingHFScaled string) *PluginError {
	return NewError(229, ArborModule, fmt.Sprintf("market %q: withdrawal would leave health factor at %s (scaled, liquidatable at <=1000000), ARCM Section 5", marketID, resultingHFScaled))
}

func ErrMarketPoolOverflow(marketID string, current uint64, amount uint64) *PluginError {
	return NewError(230, ArborModule, fmt.Sprintf("market %q: adding amount %d to escrow pool balance %d would overflow uint64", marketID, amount, current))
}

func ErrRepayExceedsDebt(marketID string, currentDebt uint64, repayAmount uint64) *PluginError {
	return NewError(231, ArborModule, fmt.Sprintf("market %q: repay amount %d exceeds current debt %d -- no escrow exists to refund an overpayment, resubmit with amount <= current debt", marketID, repayAmount, currentDebt))
}

func ErrPositionNotLiquidatable(marketID string, hfScaled string) *PluginError {
	return NewError(232, ArborModule, fmt.Sprintf("market %q: position health factor %s (scaled) is above the liquidation threshold (<=1000000), ARCM Section 5", marketID, hfScaled))
}

func ErrRepayExceedsCloseFactor(marketID string, requested, maxAllowed uint64) *PluginError {
	return NewError(233, ArborModule, fmt.Sprintf("market %q: liquidation repay amount %d exceeds close-factor cap %d for this position's HF tier, ARCM Section 7", marketID, requested, maxAllowed))
}

func ErrLiquidationBadDebt(marketID string, badDebt uint64) *PluginError {
	return NewError(234, ArborModule, fmt.Sprintf("market %q: liquidation leaves bad debt %d (debt-asset-native units) -- Layer 2 (reserve fund) checked and insufficient to cover; Layer 3 (protocol treasury) and Layer 4 (lender socialization) not yet implemented, ARCM Section 9.2", marketID, badDebt))
}

func ErrMarketPaused(marketID string) *PluginError {
	return NewError(235, ArborModule, fmt.Sprintf("market %q: paused by governance/risk-committee action -- no deposit/withdraw/borrow/repay/collateral/liquidation activity permitted until resumed, ARCM Section 13/16", marketID))
}

func ErrMarketDeprecated(marketID string) *PluginError {
	return NewError(236, ArborModule, fmt.Sprintf("market %q: permanently deprecated (ARCM Section 19.2) -- no new deposits, borrows, or collateral additions permitted; existing positions may still withdraw, repay, or be liquidated", marketID))
}

func ErrMarketNotPaused(marketID string) *PluginError {
	return NewError(237, ArborModule, fmt.Sprintf("market %q: not currently paused, nothing to resume", marketID))
}

// [NEW] ErrSelfLiquidation -- liquidator and borrower are the same address.
// Without this guard, a borrower could liquidate their own underwater
// position and collect the LIF bonus (ARCM Section 8) risk-free -- that
// bonus exists to compensate an INDEPENDENT liquidator for monitoring and
// execution risk a self-liquidating borrower never bears. repay.go already
// covers the legitimate case of a borrower closing their own position
// without a bonus.
func ErrSelfLiquidation() *PluginError {
	return NewError(238, ArborModule, "liquidator and borrower_address must not be the same address -- self-liquidation is not permitted; use repay to close your own position instead")
}

// ErrScaledDebtOverflow guards ScaledDebt()'s numerator big.Int -> uint64
// cast (AYIS Section 6, ARCM Section 2.2's mandatory scaled-debt rule).
// Previously disclosed as an unguarded, deliberate carve-out in
// scaled_debt.go's own doc comment (no amplification path analogous to
// MintShares()/RedeemShares()/SumLenderBalancesInMarket(), per AYIS
// v1.11 Section 10.6) -- closed here per Arbor Handoff Part 2, item 2.
func ErrScaledDebtOverflow(marketID string, debtPrincipal uint64, scaledDebt string) *PluginError {
	return NewError(241, ArborModule, fmt.Sprintf("market %q: scaled debt value %s for debtPrincipal %d exceeds uint64 range, AYIS Section 6", marketID, scaledDebt, debtPrincipal))
}
