package eth

import (
	"errors"
)

var (
	ErrInvalidKey             = errors.New("invalid private key")
	ErrInvalidAddress         = errors.New("invalid address")
	ErrInvalidTransactionData = errors.New("invalid transaction data")
	ErrNotERC20Transfer       = errors.New("transaction is not an erc20 transfer")
	ErrContractNotFound       = errors.New("contract address not found")
	ErrInvalidPrivateKey      = errors.New("invalid private key")
	ErrTransactionFailed      = errors.New("transaction failed")
	ErrGasPriceEstimation     = errors.New("failed to estimate gas price")
	ErrNonceRetrieval         = errors.New("failed to retrieve nonce")
	ErrGasEstimation          = errors.New("failed to estimate gas")
	ErrTransactionSigning     = errors.New("failed to sign transaction")
	ErrTransactionSending     = errors.New("failed to send transaction")
)
