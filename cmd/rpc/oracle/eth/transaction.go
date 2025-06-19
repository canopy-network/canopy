package eth

import (
	"errors"
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	// ethereumBlockchain represents the ethereum blockchain identifier
	ethereumBlockchain = "ethereum"
)

var _ types.TransactionI = &Transaction{} // Ensures *Transaction implements TransactionI

// Transaction represents an ethereum transaction that implements TransactionI
type Transaction struct {
	// tx holds the underlying ethereum transaction
	tx *ethtypes.Transaction
	// tokenTransfer holds the parsed token transfer information
	tokenTransfer types.TokenTransfer
	// extraData holds any additional data beyond standard transaction
	extraData []byte
}

// NewTransaction creates a new Transaction instance from an ethereum transaction
func NewTransaction(tx *ethtypes.Transaction) (*Transaction, error) {
	// create new transaction wrapper
	transaction := &Transaction{
		tx: tx,
	}
	// return the transaction
	return transaction, nil
}

// Blockchain returns the blockchain identifier
func (t *Transaction) Blockchain() string {
	// return ethereum blockchain identifier
	return ethereumBlockchain
}

// From returns the sender address of the transaction in uppercase
func (t *Transaction) From() string {
	// extract sender address using latest signer
	if from, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(big.NewInt(31337)), t.tx); err == nil {
		return from.Hex()
	}
	// return empty string if extraction fails
	return ""
}

// To returns the recipient address of the transaction in uppercase
func (t *Transaction) To() string {
	// check if transaction has a recipient
	if t.tx.To() != nil {
		return t.tx.To().Hex()
	}
	// return empty string for contract creation transactions
	return ""
}

// Data returns the transaction data or extra data if available
func (t *Transaction) Data() []byte {
	// return extra data if it exists
	if t.extraData != nil {
		return t.extraData
	}
	// return original transaction data
	return t.tx.Data()
}

// Hash returns the transaction hash
func (t *Transaction) Hash() string {
	// return transaction hash as hex string
	return t.tx.Hash().Hex()
}

// TokenTransfer returns the token transfer information
func (t *Transaction) TokenTransfer() types.TokenTransfer {
	// return the token transfer data
	return t.tokenTransfer
}

// populateTokenTransfer populates the internal tokenTransfer struct
func (t *Transaction) populateTokenTransfer(tokenInfo types.TokenInfo, recipient string, amount *big.Int) error {
	if t == nil {
		return errors.New("transaction is nil")
	}
	if amount == nil {
		return errors.New("amount is nil")
	}
	if amount.Sign() < 0 {
		return errors.New("amount cannot be negative")
	}
	if recipient == "" {
		return errors.New("recipient address cannot be empty")
	}

	// calculate decimal-adjusted amount
	decimals := big.NewInt(int64(tokenInfo.Decimals))
	divisor := new(big.Int).Exp(big.NewInt(10), decimals, nil)
	if divisor.Cmp(big.NewInt(0)) == 0 {
		return errors.New("divisor cannot be zero")
	}

	decimalAmount := new(big.Float).SetInt(amount)
	decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(divisor))
	tokenAmount, accuracy := decimalAmount.Float64()
	if accuracy != big.Exact && accuracy != big.Below && accuracy != big.Above {
		return errors.New("failed to convert decimal amount to float64")
	}

	hash := t.Hash()
	from := t.From()
	to := t.To()

	// populate token transfer struct
	t.tokenTransfer = types.TokenTransfer{
		Blockchain:       ethereumBlockchain,
		TokenInfo:        tokenInfo,
		TransactionID:    hash,
		SenderAddress:    from,
		RecipientAddress: recipient,
		TokenAmount:      tokenAmount,
		TokenBaseAmount:  amount.Uint64(),
		ContractAddress:  to,
	}

	return nil
}
