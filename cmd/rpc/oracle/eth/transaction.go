package eth

import (
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	// ethereumBlockchain represents the ethereum blockchain identifier
	ethereumBlockchain = "ethereum"
	// erc20TransferMethodID is the method signature for ERC20 transfer function
	erc20TransferMethodID = "a9059cbb"
	// erc20TransferDataLength is the expected length of ERC20 transfer data (4 bytes method + 32 bytes address + 32 bytes amount)
	erc20TransferDataLength = 68
)

var _ types.TransactionI = &Transaction{} // Ensures *Transaction implements TransactionI

// Transaction represents an ethereum transaction that implements TransactionI
type Transaction struct {
	// tx holds the underlying ethereum transaction
	tx *ethtypes.Transaction
	// to address
	to string
	// signer address (transaction from address)
	from string
	// tokenInfo holds erc20 token info
	tokenInfo types.TokenInfo
	// isERC20 stores whether a valid ERC20 transfer function id was detected, and transaction data is of sufficient length
	isERC20 bool
	// erc20Amount stores the amount of the erc20 token transferred
	erc20Amount *big.Int
	// erc20Recipient is the recipient of the erc20 transfer
	erc20Recipient string
	// order contains the witnessed order and height
	order *types.WitnessedOrder
	// orderData contains the validated bytes of a canopy lock or close order
	orderData []byte
}

// NewTransaction creates a new Transaction instance from an ethereum transaction
func NewTransaction(ethTx *ethtypes.Transaction) (*Transaction, error) {
	// create new tx wrapper
	tx := &Transaction{
		tx: ethTx,
	}
	// check if transaction has a recipient
	if ethTx.To() != nil {
		// set to address
		tx.to = ethTx.To().Hex()
	}
	// return transaction
	return tx, nil
}

// parseDataForOrders examines the transaction input data looking for canopy orders, returning whether an order was found
func (t *Transaction) parseDataForOrders() {
	// get ethereum transaction data
	txData := t.tx.Data()
	// check for transaction data
	if txData == nil || len(txData) == 0 {
		// no transaction data to process
		return
	}
	// test for self-sent transactions
	if t.To() == t.From() {
		// check for a lock order in the transaction data
		order, err := unmarshalValidatedLockOrder(txData)
		if err != nil {
			return
		}
		// create witnessed order
		t.order = &types.WitnessedOrder{
			OrderId:   order.OrderId,
			LockOrder: order,
		}
		return
	}
	// try to get ERC20 data
	recipient, amount, data, err := parseERC20Transfer(txData)
	if err != nil {
		// not an erc20 transfer
		return
	}
	// check for a valid close order in this data
	order, err := unmarshalValidatedCloseOrder(data)
	if err != nil {
		return
	}
	// set erc20 flag
	t.isERC20 = true
	// create witnessed order
	t.order = &types.WitnessedOrder{
		OrderId:    order.OrderId,
		CloseOrder: order,
	}
	// store erc20 fields
	t.erc20Recipient = recipient
	t.erc20Amount = amount
	// set erc20
	t.isERC20 = true
}

// Blockchain returns the blockchain identifier
func (t *Transaction) Blockchain() string {
	// return ethereum blockchain identifier
	return ethereumBlockchain
}

// From returns the sender address of the transaction
func (t *Transaction) From() string {
	return t.from
}

// To returns the recipient address of the transaction
func (t *Transaction) To() string {
	return t.to
}

// Order returns the witnessed order
func (t *Transaction) Order() *types.WitnessedOrder {
	return t.order
}

// Hash returns the transaction hash
func (t *Transaction) Hash() string {
	// return transaction hash as hex string
	return t.tx.Hash().Hex()
}

// clearOrder resets order and transfer data
func (t *Transaction) clearOrder() {
	t.order = nil
	t.isERC20 = false
}

// TokenTransfer returns the token transfer information
func (t *Transaction) TokenTransfer() types.TokenTransfer {
	return types.TokenTransfer{
		Blockchain:       ethereumBlockchain,
		TokenInfo:        t.tokenInfo,
		TransactionID:    t.Hash(),
		SenderAddress:    t.From(),
		RecipientAddress: t.erc20Recipient,
		TokenBaseAmount:  t.erc20Amount,
		ContractAddress:  t.To(),
	}
}

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
