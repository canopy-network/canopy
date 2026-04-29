package canoliq

import (
	"fmt"

	"github.com/canopy-network/go-plugin/contract"
)

// Module is the error module identifier reported in PluginError records.
const Module = "canoliq"

// Error code space starts at 100 to avoid colliding with contract package codes (1–14).
const (
	codeInvalidAddress = iota + 100
	codeInvalidAmount
	codeInsufficientCNPY
	codeInsufficientCCNPY
	codeInsufficientCLIQ
	codeInsufficientLockedCLIQ
	codeRedemptionNotFound
	codeRedemptionNotMature
	codeNoVestingSchedule
	codeNothingToClaim
	codeFeeBelowMinimum
	codeInvalidParams
	codeGenesisAlreadyRun
	codeGenesisNotRun
	codePoolMath
	codeUnsupportedMessage
	codeStateUnmarshal
)

// newError constructs a PluginError stamped with the canoLiq module.
func newError(code uint64, msg string) *contract.PluginError {
	return &contract.PluginError{Code: code, Module: Module, Msg: msg}
}

// ErrInvalidAddress reports a malformed (non-20-byte) address.
func ErrInvalidAddress() *contract.PluginError {
	return newError(codeInvalidAddress, "address must be 20 bytes")
}

// ErrInvalidAmount reports a zero or otherwise invalid amount.
func ErrInvalidAmount() *contract.PluginError {
	return newError(codeInvalidAmount, "amount must be greater than zero")
}

// ErrInsufficientCNPY reports a CNPY balance that does not cover amount + fee.
func ErrInsufficientCNPY() *contract.PluginError {
	return newError(codeInsufficientCNPY, "insufficient CNPY balance")
}

// ErrInsufficientCCNPY reports a cCNPY balance below the requested redemption.
func ErrInsufficientCCNPY() *contract.PluginError {
	return newError(codeInsufficientCCNPY, "insufficient cCNPY balance")
}

// ErrInsufficientCLIQ reports a liquid CLIQ balance below the requested transfer.
func ErrInsufficientCLIQ() *contract.PluginError {
	return newError(codeInsufficientCLIQ, "insufficient liquid CLIQ balance")
}

// ErrInsufficientLockedCLIQ reports an attempt to use locked vesting CLIQ.
func ErrInsufficientLockedCLIQ() *contract.PluginError {
	return newError(codeInsufficientLockedCLIQ, "CLIQ is still locked under vesting schedule")
}

// ErrRedemptionNotFound reports a missing redemption record.
func ErrRedemptionNotFound() *contract.PluginError {
	return newError(codeRedemptionNotFound, "redemption record not found")
}

// ErrRedemptionNotMature reports a claim before the unbonding window expires.
func ErrRedemptionNotMature() *contract.PluginError {
	return newError(codeRedemptionNotMature, "redemption has not yet matured")
}

// ErrNoVestingSchedule reports the absence of any vesting schedule for the caller.
func ErrNoVestingSchedule() *contract.PluginError {
	return newError(codeNoVestingSchedule, "no vesting schedule for address")
}

// ErrNothingToClaim reports a claim that would unlock zero CLIQ.
func ErrNothingToClaim() *contract.PluginError {
	return newError(codeNothingToClaim, "no vested CLIQ available to claim")
}

// ErrFeeBelowMinimum reports a tx fee under the configured minimum.
func ErrFeeBelowMinimum() *contract.PluginError {
	return newError(codeFeeBelowMinimum, "tx fee below minimum")
}

// ErrInvalidParams reports invalid CanoliqParams (e.g., split bps mismatch).
func ErrInvalidParams() *contract.PluginError {
	return newError(codeInvalidParams, "invalid canoLiq parameters")
}

// ErrGenesisAlreadyRun reports a duplicate genesis call.
func ErrGenesisAlreadyRun() *contract.PluginError {
	return newError(codeGenesisAlreadyRun, "canoLiq genesis already executed")
}

// ErrGenesisNotRun reports state operations attempted before genesis.
func ErrGenesisNotRun() *contract.PluginError {
	return newError(codeGenesisNotRun, "canoLiq genesis has not yet executed")
}

// ErrPoolMath wraps an overflow or divide-by-zero in pool exchange-rate math.
func ErrPoolMath(detail string) *contract.PluginError {
	return newError(codePoolMath, fmt.Sprintf("pool math error: %s", detail))
}

// ErrUnsupportedMessage reports an unrecognized canoLiq message type.
func ErrUnsupportedMessage() *contract.PluginError {
	return newError(codeUnsupportedMessage, "unsupported canoLiq message type")
}

// ErrStateUnmarshal wraps a stored value that fails to deserialize.
func ErrStateUnmarshal(err error) *contract.PluginError {
	return newError(codeStateUnmarshal, fmt.Sprintf("state unmarshal failed: %s", err))
}
