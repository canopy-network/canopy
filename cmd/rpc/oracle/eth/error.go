package eth

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidTransactionData = errors.New("invalid transaction data")
	ErrNotERC20Transfer       = errors.New("transaction is not an erc20 transfer")
	ErrContractNotFound       = errors.New("contract address not found")
	ErrNilTransaction         = errors.New("transaction is nil")
	ErrTransactionReceipt     = errors.New("failed to get transaction receipt")
	ErrTokenInfo              = errors.New("failed to get token info")
	ErrSourceHeight           = errors.New("ethereum block height lower than expected")
)

// InvalidAddressError represents an error for an invalid ethereum address
type InvalidAddressError struct {
	Address string
}

// Error returns the error message including the invalid address
func (e *InvalidAddressError) Error() string {
	return fmt.Sprintf("invalid address: %s", e.Address)
}
