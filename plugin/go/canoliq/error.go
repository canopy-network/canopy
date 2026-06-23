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
	codeInsufficientStakedCLIQ
	codeUnstakeNotFound
	codeUnstakeNotMature
	codeProposalNotFound
	codeProposalInactive
	codeProposalNotPassed
	codeProposalAlreadyExecuted
	codeAlreadyVoted
	codeStakeAfterCreation
	codeUnknownProposalPayload
	codeBuybackOrderNotFound
	codeSpendNotFound
	codeSpendNotReady
	codeSpendAlreadyExecuted
	codeNotMultisigSigner
	codeAlreadyApproved
	codeBelowProposalMinStake
	codeInvalidProposalPayload
	codeInsufficientTreasuryCLIQ
	codeInsufficientTreasuryCNPY
	codeInvalidLockTier
	codeStakeLocked
	codeTVLCapExceeded
	codeCanopyStakeUnavailable
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

// ErrInsufficientStakedCLIQ reports an unstake exceeding the active stake.
func ErrInsufficientStakedCLIQ() *contract.PluginError {
	return newError(codeInsufficientStakedCLIQ, "insufficient staked CLIQ")
}

// ErrUnstakeNotFound reports a missing UnstakingCLIQ record.
func ErrUnstakeNotFound() *contract.PluginError {
	return newError(codeUnstakeNotFound, "unstake record not found")
}

// ErrUnstakeNotMature reports a claim before the unbond window has elapsed.
func ErrUnstakeNotMature() *contract.PluginError {
	return newError(codeUnstakeNotMature, "unstake has not yet matured")
}

// ErrProposalNotFound reports a missing Proposal record.
func ErrProposalNotFound() *contract.PluginError {
	return newError(codeProposalNotFound, "proposal not found")
}

// ErrProposalInactive reports a vote on a non-active proposal.
func ErrProposalInactive() *contract.PluginError {
	return newError(codeProposalInactive, "proposal not in active voting window")
}

// ErrProposalNotPassed reports an execute attempt on a non-passed proposal.
func ErrProposalNotPassed() *contract.PluginError {
	return newError(codeProposalNotPassed, "proposal has not passed")
}

// ErrProposalAlreadyExecuted reports a double-execute attempt.
func ErrProposalAlreadyExecuted() *contract.PluginError {
	return newError(codeProposalAlreadyExecuted, "proposal already executed")
}

// ErrAlreadyVoted reports a duplicate vote from the same address.
func ErrAlreadyVoted() *contract.PluginError {
	return newError(codeAlreadyVoted, "address has already voted on this proposal")
}

// ErrStakeAfterCreation reports a vote whose stake postdates the proposal.
func ErrStakeAfterCreation() *contract.PluginError {
	return newError(codeStakeAfterCreation, "voter stake added after proposal creation height")
}

// ErrUnknownProposalPayload reports a payload Any whose type is unsupported.
func ErrUnknownProposalPayload() *contract.PluginError {
	return newError(codeUnknownProposalPayload, "unknown proposal payload type")
}

// ErrBuybackOrderNotFound reports a missing BuybackOrder.
func ErrBuybackOrderNotFound() *contract.PluginError {
	return newError(codeBuybackOrderNotFound, "buyback order not found")
}

// ErrSpendNotFound reports a missing TreasurySpend.
func ErrSpendNotFound() *contract.PluginError {
	return newError(codeSpendNotFound, "treasury spend not found")
}

// ErrSpendNotReady reports an above-threshold spend lacking timelock or
// multisig coverage.
func ErrSpendNotReady() *contract.PluginError {
	return newError(codeSpendNotReady, "treasury spend not ready (timelock or multisig)")
}

// ErrSpendAlreadyExecuted reports a double-execute attempt.
func ErrSpendAlreadyExecuted() *contract.PluginError {
	return newError(codeSpendAlreadyExecuted, "treasury spend already executed")
}

// ErrNotMultisigSigner reports an approval from a non-authorized signer.
func ErrNotMultisigSigner() *contract.PluginError {
	return newError(codeNotMultisigSigner, "address is not a configured multisig signer")
}

// ErrAlreadyApproved reports a duplicate approval from the same signer.
func ErrAlreadyApproved() *contract.PluginError {
	return newError(codeAlreadyApproved, "signer has already approved this spend")
}

// ErrBelowProposalMinStake reports a proposer below the configured minimum.
func ErrBelowProposalMinStake() *contract.PluginError {
	return newError(codeBelowProposalMinStake, "proposer staked CLIQ below minimum")
}

// ErrInvalidProposalPayload reports a malformed or self-inconsistent payload.
func ErrInvalidProposalPayload() *contract.PluginError {
	return newError(codeInvalidProposalPayload, "invalid proposal payload")
}

// ErrInsufficientTreasuryCLIQ reports a treasury_cliq draw beyond available.
func ErrInsufficientTreasuryCLIQ() *contract.PluginError {
	return newError(codeInsufficientTreasuryCLIQ, "insufficient CLIQ in DAO treasury")
}

// ErrInsufficientTreasuryCNPY reports a treasury_canopy draw beyond available.
func ErrInsufficientTreasuryCNPY() *contract.PluginError {
	return newError(codeInsufficientTreasuryCNPY, "insufficient CNPY in DAO treasury")
}

// ErrInvalidLockTier reports a stake with an unknown vote-escrow lock tier.
func ErrInvalidLockTier() *contract.PluginError {
	return newError(codeInvalidLockTier, "invalid vote-escrow lock tier")
}

// ErrStakeLocked reports an unstake before the vote-escrow lock_end_height.
func ErrStakeLocked() *contract.PluginError {
	return newError(codeStakeLocked, "stake is locked until lock_end_height")
}

// ErrTVLCapExceeded reports a deposit that would push total pooled CNPY above
// the percentage TVL cap (mulDiv(canopy_total_stake, tvl_cap_bps, 10000)).
func ErrTVLCapExceeded() *contract.PluginError {
	return newError(codeTVLCapExceeded, "deposit would exceed the TVL cap")
}

// ErrCanopyStakeUnavailable reports that the TVL cap check could not be
// evaluated because the Canopy Supply state was unreadable or staked = 0.
// The deposit fails closed per WP §9.4 (the cap exists to bound systemic
// risk; silently bypassing it would defeat the spec).
func ErrCanopyStakeUnavailable() *contract.PluginError {
	return newError(codeCanopyStakeUnavailable, "Canopy total stake unavailable; TVL cap cannot be enforced — deposit rejected")
}
