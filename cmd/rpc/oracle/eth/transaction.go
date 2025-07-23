package eth

import (
	"fmt"
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	// ethereumBlockchain represents the ethereum blockchain identifier
	ethereumBlockchain = "ethereum"
	// erc20TransferMethodID is the method signature for ERC20 transfer function
	erc20TransferMethodID = "a9059cbb"
	// erc20TransferDataLength is the expected length of ERC20 transfer data (4 bytes method + 32 bytes address + 32 bytes amount)
	erc20TransferDataLength = 68
	// maxTransactionDataSize is the maximum allowed size for transaction data to prevent memory exhaustion
	maxTransactionDataSize = 1024
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
func NewTransaction(ethTx *ethtypes.Transaction, chainId uint64) (*Transaction, error) {
	// check nil ethTx
	if ethTx == nil {
		return nil, ErrNilTransaction
	}
	// create new tx wrapper
	tx := &Transaction{
		tx: ethTx,
	}
	// check if transaction has a recipient
	if ethTx.To() != nil {
		// set to address
		tx.to = ethTx.To().Hex()
	}
	// validate address format
	if !common.IsHexAddress(tx.to) {
		return nil, ErrInvalidAddress
	}
	// extract sender address using latest signer
	from, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(big.NewInt(int64(chainId))), ethTx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sender address: %w", err)
	}
	// set from address
	tx.from = from.Hex()
	// return transaction
	return tx, nil
}

// parseDataForOrders examines the transaction input data looking for canopy orders
func (t *Transaction) parseDataForOrders(orderValidator OrderValidator) error {
	// get ethereum transaction data
	txData := t.tx.Data()
	// check for transaction data
	if len(txData) == 0 {
		// no transaction data to process
		return nil
	}
	// test txData size
	if len(txData) > maxTransactionDataSize {
		// large transactions are not expected from Canopy swap clients
		return nil
	}
	// test for self-sent transactions
	if t.To() == t.From() {
		// canopy swap clients will place lock order json in transaction data
		err := orderValidator.ValidateOrderJsonBytes(txData, types.LockOrderType)
		if err != nil {
			// self-sent transaction did not contain canopy lock order json - normal condition
			return nil
		}
		order := &lib.LockOrder{}
		// unmarshal the validated json data
		err = order.UnmarshalJSON(txData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal lock order json: %w", err)
		}
		// create and store witnessed order
		t.order = &types.WitnessedOrder{
			OrderId:   order.OrderId,
			LockOrder: order,
		}
		// lock order found - no error
		return nil
	}
	// Canopy swap embeds lock and close orders in data trailing
	// standard ERC20 ABI encoded function call data
	recipient, amount, data, err := parseERC20Transfer(txData)
	if err != nil {
		// not an erc20 transfer - normal condition
		return nil
	}
	// all Canopy swap ERC20 transfers have aux data
	if len(data) == 0 {
		// no data to process - not a canopy swap ERC20 transfer
		return nil
	}
	// test for self-sent ERC20 transfers
	if t.from != recipient {
		// not self sent - not a canopy swap ERC20 transfer
		return nil
	}
	// fmt.Println("checking", t.from, recipient, amount, string(data), err)
	// fmt.Println("checking", t.from == recipient, amount == new(big.Int).SetUint64(0))

	switch amount.Cmp(big.NewInt(0)) {
	case 0: // zero amount - potential lock order
		// attempt to validate a lock order
		err = orderValidator.ValidateOrderJsonBytes(data, types.LockOrderType)
		if err != nil {
			// erc20 transaction did not contain canopy lock order json - normal condition
			return nil
		}
		fmt.Println("validated lock order", t.from, recipient, amount, string(data), err)
		order := &lib.LockOrder{}
		// unmarshal the validated json data
		err = order.UnmarshalJSON(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal validated lock order json: %w", err)
		}
		// create witnessed order
		t.order = &types.WitnessedOrder{
			OrderId:   order.OrderId,
			LockOrder: order,
		}
	case 1: // positive amount - potential close order
		// attempt to validate a close order
		err = orderValidator.ValidateOrderJsonBytes(data, types.CloseOrderType)
		if err != nil {
			// erc20 transaction did not contain canopy close order json - normal condition
			return nil
		}
		order := &lib.CloseOrder{}
		// unmarshal the validated json data
		err = order.UnmarshalJSON(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal validated close order json: %w", err)
		}
		// create witnessed order
		t.order = &types.WitnessedOrder{
			OrderId:    order.OrderId,
			CloseOrder: order,
		}
	}
	// set erc20 flag
	t.isERC20 = true
	// store erc20 fields
	t.erc20Recipient = recipient
	t.erc20Amount = amount
	return nil
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

// clearOrder clears order and transfer data
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

// parseERC20Transfer parses the transaction data looking for ERC20 transfers and any auxiliary data beyond the standard transfer call
func parseERC20Transfer(data []byte) (recipientAddress string, amount *big.Int, auxData []byte, err error) {
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
		auxData = data[erc20TransferDataLength:]
	}
	return recipientAddress, amount, auxData, nil
}
