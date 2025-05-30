package eth

import (
	"math/big"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/ethereum/go-ethereum/core/types"
)

// Transaction represents an Ethereum transaction that implements TransactionI
type Transaction struct {
	tx            *types.Transaction    // the underlying ethereum transaction
	from          string                // cached from address
	tokenTransfer wstypes.TokenTransfer // token transfer data if this is an erc20 transfer
	extraData     []byte                // extra data from erc20 transfer
}

// NewTransaction creates a new Transaction from an ethereum transaction
func NewTransaction(tx *types.Transaction) (*Transaction, error) {
	// create a new transaction object
	transaction := &Transaction{
		tx: tx,
	}

	// extract the from address
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(31337)), tx)
	if err != nil {
		return nil, err
	}
	transaction.from = from.Hex()

	// check if this is an erc20 transfer
	tokenTransfer, extraData, err := ParseERC20Transfer(tx)
	if err == nil && tokenTransfer.TokenSymbol != "" {
		// this is an erc20 transfer
		transaction.tokenTransfer = tokenTransfer
		transaction.extraData = extraData
	}

	return transaction, err
}

// Blockchain returns the blockchain name
func (t *Transaction) Blockchain() string {
	return "Ethereum"
}

// From returns the sender address
func (t *Transaction) From() string {
	return t.from
}

// To returns the recipient address
func (t *Transaction) To() string {
	if t.tx.To() == nil {
		return "" // contract creation transaction
	}
	return t.tx.To().Hex()
}

// Data returns the transaction data
func (t *Transaction) Data() []byte {
	// if this is an erc20 transfer, return the extra data instead
	if t.extraData != nil {
		return t.extraData
	}
	// otherwise return the original transaction data
	return t.tx.Data()
}

// Hash returns the transaction hash
func (t *Transaction) Hash() string {
	return t.tx.Hash().Hex()
}

// TokenTransfer returns the token transfer information
func (t *Transaction) TokenTransfer() wstypes.TokenTransfer {
	return t.tokenTransfer
}
