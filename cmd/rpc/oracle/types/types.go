package types

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
	Data() []byte
	Hash() string
	TokenTransfer() TokenTransfer
}

type OrderType string

const (
	// LockOrderType represents lock order transactions
	LockOrderType = "lock"
	// CloseOrderType represents close order transactions
	CloseOrderType = "close"
)

// OrderStoreI defines the methods that are required for order persistence.
type OrderStoreI interface {
	// VerifyOrder verifies the byte data of a stored order
	VerifyOrder(orderId []byte, orderType OrderType, data []byte) error

	// WriteOrder writes an order
	WriteOrder(orderId []byte, orderType OrderType, data []byte) error

	// ReadOrder reads an order
	ReadOrder(orderId []byte, orderType OrderType) ([]byte, error)

	// RemoveOrder removes an order
	RemoveOrder(orderId []byte, orderType OrderType) error

	// GetAllOrderIds gets all order ids present in the store
	GetAllOrderIds(orderType OrderType) [][]byte
}

type BlockProvider interface {
	// SetHeight sets the next block to be provided
	SetHeight(height uint64)

	// Block returns the channel this provider will send new blocks through
	BlockCh() chan BlockI
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
	TransactionID    string  // Unique identifier for the transaction
	SenderAddress    string  // Address of the sender
	RecipientAddress string  // Address of the recipient
	TokenAmount      float64 // Amount of tokens transferred decimal-adjusted
	TokenBaseAmount  uint64  // Amount of tokens transferred represented in base units
	ContractAddress  string  // Mint address or contract address of the token
}
