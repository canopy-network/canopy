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

// =============================================================================
// KnowBet Error Types
// =============================================================================

func ErrInvalidQuestionText() *PluginError {
	return NewError(15, DefaultModule, "question text is empty or contains invalid content")
}

func ErrInvalidOptionsCount() *PluginError {
	return NewError(16, DefaultModule, "options count must be between 2 and 5")
}

func ErrInvalidOption() *PluginError {
	return NewError(17, DefaultModule, "option text is empty or invalid")
}

func ErrInvalidAnswerHash() *PluginError {
	return NewError(18, DefaultModule, "answer hash must be a valid 64-char hex string (SHA-256)")
}

func ErrStakeTooLow() *PluginError {
	return NewError(19, DefaultModule, "stake amount must be at least 100 KNBT (100,000,000 uknbt)")
}

func ErrInvalidDuration() *PluginError {
	return NewError(20, DefaultModule, "duration must be between 1 hour and 7 days")
}

func ErrQuestionNotFound() *PluginError {
	return NewError(21, DefaultModule, "question not found")
}

func ErrQuestionNotOpen() *PluginError {
	return NewError(22, DefaultModule, "question is not in open state")
}

func ErrBetDeadlinePassed() *PluginError {
	return NewError(23, DefaultModule, "betting deadline has passed")
}

func ErrBetBelowMin() *PluginError {
	return NewError(24, DefaultModule, "bet amount must be at least 10 KNBT (10,000,000 uknbt)")
}

func ErrBetAboveMax() *PluginError {
	return NewError(25, DefaultModule, "bet amount must not exceed 10000 KNBT (10,000,000,000 uknbt)")
}

func ErrCreatorCannotBet() *PluginError {
	return NewError(26, DefaultModule, "question creator cannot bet on their own question")
}

func ErrAlreadyRevealed() *PluginError {
	return NewError(27, DefaultModule, "question answer has already been revealed")
}

func ErrAlreadySettled() *PluginError {
	return NewError(28, DefaultModule, "question has already been settled")
}

func ErrAlreadyRefunded() *PluginError {
	return NewError(29, DefaultModule, "question has already been refunded")
}

func ErrNotQuestionCreator() *PluginError {
	return NewError(30, DefaultModule, "only the question creator can reveal the answer")
}

func ErrAnswerHashMismatch() *PluginError {
	return NewError(31, DefaultModule, "revealed answer does not match the pre-committed hash")
}

func ErrNotRevealed() *PluginError {
	return NewError(32, DefaultModule, "question has not been revealed yet")
}

func ErrRefundPeriodNotElapsed() *PluginError {
	return NewError(33, DefaultModule, "refund period (7 days after close) has not elapsed yet")
}

func ErrSensitiveContent() *PluginError {
	return NewError(34, DefaultModule, "question text or options contain prohibited content")
}

func ErrParticipantAlreadyBetted() *PluginError {
	return NewError(35, DefaultModule, "participant has already placed a bet on this question")
}

func ErrInvalidOptionIndex() *PluginError {
	return NewError(36, DefaultModule, "selected option index is invalid")
}

func ErrNoParticipants() *PluginError {
	return NewError(37, DefaultModule, "no participants to settle or refund")
}

func ErrNoWinners() *PluginError {
	return NewError(38, DefaultModule, "no winners found for this question")
}
