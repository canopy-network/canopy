package types

import (
	"context"
	"errors"
	"math/big"

	"github.com/canopy-network/canopy/lib"
)

const (
	// LockOrderType represents lock order transactions
	LockOrderType = OrderType("lock")
	// CloseOrderType represents close order transactions
	CloseOrderType = OrderType("close")
)

type WitnessedOrder struct {
	// OrderId for the enclosed lock or close order
	OrderId lib.HexBytes `json:"orderId"`
	// witnessed height on the source block chain (ethereum, solana, etc)
	WitnessedHeight uint64 `json:"witnessedHeight"`
	// last canopy root chain height this order was submitted
	LastSubmitHeight uint64 `json:"lastSubmightHeight"`
	// Witnessed lock order
	LockOrder *lib.LockOrder `json:"lockOrder"`
	// Witnessed close order
	CloseOrder *lib.CloseOrder `json:"closeOrder"`
}

// BlockI interface represents a blockchain block
type BlockI interface {
	Hash() string
	Number() uint64
	Transactions() []TransactionI
}

// TransactionI interface represents a blockchain transaction
type TransactionI interface {
	Blockchain() string
	From() string
	To() string
	Hash() string
	Order() *WitnessedOrder
	TokenTransfer() TokenTransfer
}

type OrderType string

// OrderStoreI defines the methods that are required for order persistence.
type OrderStoreI interface {
	// VerifyOrder verifies the byte data of a stored order
	VerifyOrder(order *WitnessedOrder, orderType OrderType) lib.ErrorI
	// WriteOrder writes an order
	WriteOrder(order *WitnessedOrder, orderType OrderType) lib.ErrorI
	// ReadOrder reads a witnessed order
	ReadOrder(orderId []byte, orderType OrderType) (*WitnessedOrder, lib.ErrorI)
	// RemoveOrder removes an order
	RemoveOrder(order []byte, orderType OrderType) lib.ErrorI
	// GetAllOrderIds gets all order ids present in the store
	GetAllOrderIds(orderType OrderType) ([][]byte, lib.ErrorI)
}

type BlockProvider interface {
	// SetHeight sets the next block to be provided
	SetHeight(height *big.Int)
	// Block returns the channel this provider will send new blocks through
	BlockCh() chan BlockI
	// Start starts the block provider
	Start(ctx context.Context)
}

// TokenInfo holds the basic information about an ERC20 token
type TokenInfo struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// TokenTransfer represents a generic token transfer across different blockchains.
type TokenTransfer struct {
	Blockchain       string // Name of the blockchain (e.g., Ethereum, Solana, Binance Smart Chain)
	TokenInfo        TokenInfo
	TransactionID    string   // Unique identifier for the transaction
	SenderAddress    string   // Address of the sender
	RecipientAddress string   // Address of the recipient
	TokenBaseAmount  *big.Int // Amount of tokens transferred represented in base units
	ContractAddress  string   // Mint address or contract address of the token
}

// Amount returns the decimal-adjusted token transfer amount
func (t TokenTransfer) DecimalAmount() (float64, error) {
	// calculate decimal-adjusted amount
	decimals := big.NewInt(int64(t.TokenInfo.Decimals))
	divisor := new(big.Int).Exp(big.NewInt(10), decimals, nil)
	if divisor.Cmp(big.NewInt(0)) == 0 {
		return 0, errors.New("divisor cannot be zero")
	}
	decimalAmount := new(big.Float).SetInt(t.TokenBaseAmount)
	decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(divisor))
	tokenAmount, accuracy := decimalAmount.Float64()
	if accuracy != big.Exact && accuracy != big.Below && accuracy != big.Above {
		return 0, errors.New("failed to convert decimal amount to float64")
	}
	return tokenAmount, nil
}
