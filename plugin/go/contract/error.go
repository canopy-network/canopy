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
return NewError(200, ArborModule, fmt.Sprintf("invalid asset_id: %s", err.Error()))
}

func ErrTierOutOfRange(tier uint32) *PluginError {
return NewError(201, ArborModule, fmt.Sprintf("tier %d invalid: set_asset_tier accepts 0-3 only (tier 4/Blacklisted cannot be set via this transaction, ARCM Section 3)", tier))
}
