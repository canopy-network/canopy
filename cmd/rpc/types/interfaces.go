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

// OrderStoreI defines the methods that are required for order persistence.
type OrderStoreI interface {
	// WriteOrder writes an order
	WriteOrder(orderId string, orderType OrderType, blockHeight uint64, txHash string, data []byte) error

	// ReadOrder reads an order
	ReadOrder(orderId string, orderType OrderType) ([]byte, error)

	// RemoveOrder removes an order
	RemoveOrder(orderId string, orderType OrderType) error

	// GetAllOrderIds gets all order ids present in the store
	GetAllOrderIds(orderType OrderType) [][]byte
}

type BlockProvider interface {
	// SetNext sets the next block to be provided
	SetNext(height uint64)

	// GetBlockByHeight gets the block at the specified height
	GetBlockByHeight(height uint64) (BlockI, error)

	// Block returns the channel this provider will send new blocks through
	BlockCh() chan BlockI
}

// TokenTransfer represents a generic token transfer across different blockchains.
type TokenTransfer struct {
	Blockchain       string  // Name of the blockchain (e.g., Ethereum, Solana, Binance Smart Chain)
	TransactionID    string  // Unique identifier for the transaction
	SenderAddress    string  // Address of the sender
	RecipientAddress string  // Address of the recipient
	TokenSymbol      string  // Symbol of the token (e.g., ETH, SOL, USDT)
	TokenAmount      float64 // Amount of tokens transferred
	TokenDecimals    int     // Number of decimals the token uses
	ContractAddress  string  // Mint address or contract address of the token
}
