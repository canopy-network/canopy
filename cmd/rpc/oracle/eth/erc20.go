package eth

import (
	"math/big"

	"github.com/canopy-network/canopy/lib"
)

const (
	// erc20TransferMethodID is the method signature for ERC20 transfer function
	erc20TransferMethodID = "a9059cbb"
	// erc20TransferDataLength is the expected length of ERC20 transfer data (4 bytes method + 32 bytes address + 32 bytes amount)
	erc20TransferDataLength = 68
)

// parseERC20Transfer parses the transaction data looking for ERC20 transfers and any extra data beyond the standard transfer call
func parseERC20Transfer(data []byte) (recipientAddress string, amount *big.Int, extraData []byte, err error) {
	// check if data is long enough to contain a valid ERC20 transfer
	if len(data) < erc20TransferDataLength {
		return "", nil, nil, ErrNotERC20Transfer
	}
	// extract method signature from first 4 bytes
	methodID := lib.BytesToString(data[:4])
	// verify this is an ERC20 transfer method call
	if methodID != erc20TransferMethodID {
		return "", nil, nil, ErrNotERC20Transfer
	}
	// extract recipient address from bytes 4-36 (32 bytes, but address is only last 20 bytes)
	recipientBytes := data[16:36]
	recipientAddress = "0x" + lib.BytesToString(recipientBytes)
	// extract amount from bytes 36-68 (32 bytes)
	amountBytes := data[36:68]
	amount = new(big.Int).SetBytes(amountBytes)
	// check if there is extra data beyond the standard transfer call
	if len(data) > erc20TransferDataLength {
		extraData = data[erc20TransferDataLength:]
	}
	return recipientAddress, amount, extraData, nil
}
