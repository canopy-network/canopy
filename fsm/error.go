package fsm

import (
	"fmt"

	"github.com/canopy-network/canopy/lib"
)

// This file defines error objects for the State Machine module

func ErrReadGenesisFile(err error) lib.ErrorI {
	return lib.NewError(lib.CodeReadGenesisFile, lib.StateMachineModule, fmt.Sprintf("read genesis file failed with err: %s", err.Error()))
}

func ErrUnmarshalGenesis(err error) lib.ErrorI {
	return lib.NewError(lib.CodeUnmarshalGenesis, lib.StateMachineModule, fmt.Sprintf("unmarshal genesis failed with err: %s", err.Error()))
}

func ErrUnauthorizedTx() lib.ErrorI {
	return lib.NewError(lib.CodeUnauthorizedTx, lib.StateMachineModule, "unauthorized tx")
}

func ErrInvalidTxMessage() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidTxMessage, lib.StateMachineModule, "invalid transaction message")
}

func ErrTxFeeBelowStateLimit() lib.ErrorI {
	return lib.NewError(lib.CodeFeeBelowState, lib.StateMachineModule, "tx.fee is below state limit")
}

func ErrRejectProposal() lib.ErrorI {
	return lib.NewError(lib.CodeRejectProposal, lib.StateMachineModule, "proposal rejected")
}

func ErrAddressEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeAddressEmpty, lib.StateMachineModule, "address is empty")
}

func ErrRecipientAddressEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeRecipientAddressEmpty, lib.StateMachineModule, "recipient address is empty")
}

func ErrOutputAddressEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeOutputAddressEmpty, lib.StateMachineModule, "output address is empty")
}

func ErrOutputAddressSize() lib.ErrorI {
	return lib.NewError(lib.CodeOutputAddressSize, lib.StateMachineModule, "output address size is invalid")
}

func ErrAddressSize() lib.ErrorI {
	return lib.NewError(lib.CodeAddressSize, lib.StateMachineModule, "address size is invalid")
}

func ErrInvalidNetAddressLen() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidNetAddressLen, lib.StateMachineModule, "net address has invalid length")
}

func ErrRecipientAddressSize() lib.ErrorI {
	return lib.NewError(lib.CodeRecipientAddressSize, lib.StateMachineModule, "recipient address size is invalid")
}

func ErrInvalidAmount() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidAmount, lib.StateMachineModule, "amount is invalid")
}

func ErrPublicKeyEmpty() lib.ErrorI {
	return lib.NewError(lib.CodePubKeyEmpty, lib.StateMachineModule, "public key is empty")
}

func ErrPublicKeySize() lib.ErrorI {
	return lib.NewError(lib.CodePubKeySize, lib.StateMachineModule, "public key size is invalid")
}

func ErrParamKeyEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeParamKeyEmpty, lib.StateMachineModule, "the parameter key is empty")
}

func ErrParamValueEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeParamValEmpty, lib.StateMachineModule, "the parameter value is empty")
}

func ErrInvalidProposalHash() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidProposalHash, lib.StateMachineModule, "the parameter value is empty")
}

func ErrUnknownMessage(x lib.MessageI) lib.ErrorI {
	return lib.NewError(lib.CodeUnknownMsg, lib.StateMachineModule, fmt.Sprintf("message %T is unknown", x))
}

func ErrInsufficientFunds() lib.ErrorI {
	return lib.NewError(lib.CodeInsufficientFunds, lib.StateMachineModule, "insufficient funds")
}

func ErrInsufficientSupply() lib.ErrorI {
	return lib.NewError(lib.CodeInsufficientSupply, lib.StateMachineModule, "insufficient supply")
}

func ErrValidatorExists() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorExists, lib.StateMachineModule, "validator exists")
}

func ErrValidatorNotExists() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorNotExists, lib.StateMachineModule, "validator does not exist")
}

func ErrValidatorUnstaking() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorUnstaking, lib.StateMachineModule, "validator is unstaking")
}

func ErrValidatorPaused() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorPaused, lib.StateMachineModule, "validator paused")
}

func ErrValidatorNotPaused() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorNotPaused, lib.StateMachineModule, "validator not paused")
}

func ErrEmptyConsParams() lib.ErrorI {
	return lib.NewError(lib.CodeEmptyConsParams, lib.StateMachineModule, "consensus params empty")
}

func ErrEmptyValParams() lib.ErrorI {
	return lib.NewError(lib.CodeEmptyValParams, lib.StateMachineModule, "validator params empty")
}

func ErrEmptyFeeParams() lib.ErrorI {
	return lib.NewError(lib.CodeEmptyFeeParams, lib.StateMachineModule, "fee params empty")
}

func ErrEmptyGovParams() lib.ErrorI {
	return lib.NewError(lib.CodeEmptyGovParams, lib.StateMachineModule, "governance params empty")
}

func ErrUnknownParam() lib.ErrorI {
	return lib.NewError(lib.CodeUnknownParam, lib.StateMachineModule, "unknown param")
}

func ErrUnknownParamSpace() lib.ErrorI {
	return lib.NewError(lib.CodeUnknownParamSpace, lib.StateMachineModule, "unknown param space")
}

func ErrUnknownParamType(t any) lib.ErrorI {
	return lib.NewError(lib.CodeUnknownParamType, lib.StateMachineModule, fmt.Sprintf("unknown param type %T", t))
}
func ErrInvalidSignature() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidSignature, lib.StateMachineModule, "invalid signature")
}

func ErrEmptySignature() lib.ErrorI {
	return lib.NewError(lib.CodeEmptySignature, lib.StateMachineModule, "empty signature")
}

func ErrTxSignBytes(err error) lib.ErrorI {
	return lib.NewError(lib.CodeTxSignBytes, lib.StateMachineModule, fmt.Sprintf("tx.SignBytes() failed with err: %s", err.Error()))
}

func ErrInvalidProtocolVersion() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidProtocolVersion, lib.StateMachineModule, "invalid protocol version")
}

func ErrInvalidKey(key []byte) lib.ErrorI {
	return lib.NewError(lib.CodeInvalidDBKey, lib.StateMachineModule, fmt.Sprintf("invalid key: %s", key))
}

func ErrInvalidParam(paramName string) lib.ErrorI {
	return lib.NewError(lib.CodeInvalidParam, lib.StateMachineModule, fmt.Sprintf("invalid param: %s", paramName))
}

func ErrWrongStoreType() lib.ErrorI {
	return lib.NewError(lib.CodeWrongStoreType, lib.StateMachineModule, "wrong store type")
}

func ErrMaxBlockSize() lib.ErrorI {
	return lib.NewError(lib.CodeMaxBlockSize, lib.StateMachineModule, "max block size")
}

func ErrInvalidBlockRange() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidBlockRange, lib.StateMachineModule, "proposal block range is invalid")
}

func ErrInvalidPublicKey(err error) lib.ErrorI {
	return lib.NewError(lib.CodeInvalidPublicKey, lib.StateMachineModule, "public key is invalid: "+err.Error())
}

func ErrInvalidNumCommittees() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidNumCommittees, lib.StateMachineModule, "committees length is invalid")
}

func ErrValidatorIsADelegate() lib.ErrorI {
	return lib.NewError(lib.CodeValidatorIsADelegate, lib.StateMachineModule, "validator is a delegate")
}

func ErrInvalidChainId() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidChainId, lib.StateMachineModule, "invalid chain id")
}

func ErrInvalidNumOfSamples() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidNumberOfSamples, lib.StateMachineModule, "invalid number of samples")
}

func ErrInvalidCertificateResults() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidCertificateResults, lib.StateMachineModule, "invalid certificate results")
}

func ErrEmptyCertificateResults() lib.ErrorI {
	return lib.NewError(lib.CodeEmptyCertificateResults, lib.StateMachineModule, "empty certificate results")
}

func ErrMismatchCertResults() lib.ErrorI {
	return lib.NewError(lib.CodeMismatchCertResults, lib.StateMachineModule, "the certificate results generated does not match the compare")
}

func ErrInvalidSubisdy() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidSubsidy, lib.StateMachineModule, "invalid subsidy")
}

func ErrInvalidOpcode() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidOpcode, lib.StateMachineModule, "invalid opcode")
}

func ErrNonSubsidizedCommittee() lib.ErrorI {
	return lib.NewError(lib.CodeNonSubsidizedCommittee, lib.StateMachineModule, "non subsidized committee")
}

func ErrMinimumOrderSize() lib.ErrorI {
	return lib.NewError(lib.CodeMinimumOrderSize, lib.StateMachineModule, "minimum order size")
}

func ErrInvalidLockOrder() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidLockOrder, lib.StateMachineModule, "lock order invalid")
}

func InvalidSellOrder() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidSellOrder, lib.StateMachineModule, "sell order invalid")
}

func ErrInvalidCloseOrder() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidCloseOrder, lib.StateMachineModule, "close order invalid")
}

func ErrDuplicateLockOrder() lib.ErrorI {
	return lib.NewError(lib.CodeDuplicateLockOrder, lib.StateMachineModule, "lock order is a duplicate")
}

func ErrInvalidBuyerDeadline() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidBuyerDeadline, lib.StateMachineModule, "lock order deadline height is invalid")
}

func ErrInvalidCheckpoint() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidCheckpoint, lib.StateMachineModule, "checkpoint is invalid")
}

func ErrInvalidStartPollHeight() lib.ErrorI {
	return lib.NewError(lib.CodeStartPollHeight, lib.StateMachineModule, "start poll height is invalid")
}

func ErrSlashNonExistentValidator() lib.ErrorI {
	return lib.NewError(lib.CodeSlashNonValidator, lib.StateMachineModule, "cannot slash non-existent validator")
}

func ErrInvalidRLPTx(err error) lib.ErrorI {
	return lib.NewError(lib.CodeInvalidRLPTx, lib.StateMachineModule, fmt.Sprintf("rlp tx decode failed: %s", err.Error()))
}

func ErrInvalidERC20Tx(err error) lib.ErrorI {
	return lib.NewError(lib.CodeInvalidERC20Tx, lib.StateMachineModule, fmt.Sprintf("erc20 decode failed: %s", err.Error()))
}

func ErrNotEmpty() lib.ErrorI {
	return lib.NewError(lib.CodeErrNotEmpty, lib.StateMachineModule, "a field that should be empty isn't")
}

func ErrStakeBelowMininum() lib.ErrorI {
	return lib.NewError(lib.CodeStakeBelowMinimum, lib.StateMachineModule, "stake below minimum")
}

func ErrMismatchDexBatchReceipt() lib.ErrorI {
	return lib.NewError(lib.CodeMismatchDexBatchReceipt, lib.StateMachineModule, "the dex batch receipt doesn't correspond to the last batch")
}

func ErrInvalidLiquidityPool() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidLiquidityPool, lib.StateMachineModule, "the dex liquidity pool amount is invalid")
}

func ErrMaxDexBatchSize() lib.ErrorI {
	return lib.NewError(lib.CodeMaxDexBatchSize, lib.StateMachineModule, "the dex batch size exceeds the global max")
}
