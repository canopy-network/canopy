package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/canopy-network/canopy/lib"
)

type OrderType string

const (
	// LockOrderType represents lock order transactions
	LockOrderType OrderType = "lock"
	// CloseOrderType represents close order transactions
	CloseOrderType OrderType = "close"
)

// ProcessingStatus represents the state of block processing
type ProcessingStatus string

const (
	// ProcessingStatusPending indicates block is queued for processing
	ProcessingStatusPending ProcessingStatus = "pending"
	// ProcessingStatusProcessing indicates block is currently being processed
	ProcessingStatusProcessing ProcessingStatus = "processing"
	// ProcessingStatusCompleted indicates block processing completed successfully
	ProcessingStatusCompleted ProcessingStatus = "completed"
	// ProcessingStatusFailed indicates block processing failed
	ProcessingStatusFailed ProcessingStatus = "failed"
)

// BlockProcessingState tracks the processing state of a block
type BlockProcessingState struct {
	// Height is the block height being processed
	Height uint64 `json:"height"`
	// Hash is the block hash for verification
	Hash string `json:"hash"`
	// ParentHash is the parent block hash for chain reorganization detection
	ParentHash string `json:"parentHash"`
	// Status indicates the current processing state
	Status ProcessingStatus `json:"status"`
	// Timestamp when the state was last updated
	Timestamp time.Time `json:"timestamp"`
	// RetryCount tracks how many times processing was attempted
	RetryCount int `json:"retryCount"`
}

type WitnessedOrder struct {
	// OrderId for the enclosed lock or close order
	OrderId lib.HexBytes `json:"orderId"`
	// Witnessed height on the source block chain (ethereum, solana, etc)
	WitnessedHeight uint64 `json:"witnessedHeight"`
	// last canopy root chain height this order was submitted
	LastSubmitHeight uint64 `json:"lastSubmitHeight"`
	// Witnessed lock order
	LockOrder *lib.LockOrder `json:"lockOrder,omitempty"`
	// Witnessed close order
	CloseOrder *lib.CloseOrder `json:"closeOrder,omitempty"`
}

// MarshalJSON emits the canonical json key `lastSubmitHeight`.
func (w WitnessedOrder) MarshalJSON() ([]byte, error) {
	type witnessedOrderJSON struct {
		OrderID          lib.HexBytes    `json:"orderId"`
		WitnessedHeight  uint64          `json:"witnessedHeight"`
		LastSubmitHeight uint64          `json:"lastSubmitHeight"`
		LockOrder        *lib.LockOrder  `json:"lockOrder,omitempty"`
		CloseOrder       *lib.CloseOrder `json:"closeOrder,omitempty"`
	}
	return json.Marshal(witnessedOrderJSON{
		OrderID:          w.OrderId,
		WitnessedHeight:  w.WitnessedHeight,
		LastSubmitHeight: w.LastSubmitHeight,
		LockOrder:        w.LockOrder,
		CloseOrder:       w.CloseOrder,
	})
}

// UnmarshalJSON accepts both `lastSubmitHeight` (canonical) and the legacy typo
// `lastSubmightHeight` for backward compatibility with existing stored orders.
func (w *WitnessedOrder) UnmarshalJSON(data []byte) error {
	type witnessedOrderJSON struct {
		OrderID              lib.HexBytes    `json:"orderId"`
		WitnessedHeight      uint64          `json:"witnessedHeight"`
		LastSubmitHeight     *uint64         `json:"lastSubmitHeight,omitempty"`
		LegacySubmightHeight *uint64         `json:"lastSubmightHeight,omitempty"`
		LockOrder            *lib.LockOrder  `json:"lockOrder,omitempty"`
		CloseOrder           *lib.CloseOrder `json:"closeOrder,omitempty"`
	}
	var dec witnessedOrderJSON
	if err := json.Unmarshal(data, &dec); err != nil {
		return err
	}
	w.OrderId = dec.OrderID
	w.WitnessedHeight = dec.WitnessedHeight
	w.LockOrder = dec.LockOrder
	w.CloseOrder = dec.CloseOrder
	w.LastSubmitHeight = 0
	if dec.LastSubmitHeight != nil {
		w.LastSubmitHeight = *dec.LastSubmitHeight
	} else if dec.LegacySubmightHeight != nil {
		w.LastSubmitHeight = *dec.LegacySubmightHeight
	}
	return nil
}

// String returns a formatted string representation of WitnessedOrder
func (w WitnessedOrder) String() string {
	// determine which order type is present
	var orderDetails string
	if w.LockOrder != nil {
		orderDetails = fmt.Sprintf("LockOrder{OrderId:%x ChainId:%d BuyerReceiveAddress:%x BuyerSendAddress:%x BuyerChainDeadline:%d}",
			w.LockOrder.OrderId, w.LockOrder.ChainId, w.LockOrder.BuyerReceiveAddress, w.LockOrder.BuyerSendAddress, w.LockOrder.BuyerChainDeadline)
	} else if w.CloseOrder != nil {
		orderDetails = fmt.Sprintf("CloseOrder{OrderId:%x ChainId:%d CloseOrder:%t}",
			w.CloseOrder.OrderId, w.CloseOrder.ChainId, w.CloseOrder.CloseOrder)
	} else {
		orderDetails = "No order data"
	}
	// return formatted string with order details
	return fmt.Sprintf("Order{ID: %s, WitnessedHeight: %d, %s}",
		w.OrderId, w.WitnessedHeight, orderDetails)
}

// Format implements fmt.Formatter for custom formatting
func (w WitnessedOrder) Format(f fmt.State, verb rune) {
	// handle different format verbs
	switch verb {
	case 'v':
		if f.Flag('+') {
			// detailed format with newlines and indentation
			fmt.Fprintf(f, "WitnessedOrder{\n  OrderId: %x\n  WitnessedHeight: %d\n  LastSubmitHeight: %d\n",
				w.OrderId, w.WitnessedHeight, w.LastSubmitHeight)
			if w.LockOrder != nil {
				fmt.Fprintf(f, "  LockOrder{OrderId:%x ChainId:%d BuyerReceiveAddress:%x BuyerSendAddress:%x BuyerChainDeadline:%d}\n",
					w.LockOrder.OrderId, w.LockOrder.ChainId, w.LockOrder.BuyerReceiveAddress, w.LockOrder.BuyerSendAddress, w.LockOrder.BuyerChainDeadline)
			}
			if w.CloseOrder != nil {
				fmt.Fprintf(f, "  CloseOrder{OrderId:%x ChainId:%d CloseOrder:%t}\n",
					w.CloseOrder.OrderId, w.CloseOrder.ChainId, w.CloseOrder.CloseOrder)
			}
			fmt.Fprint(f, "}")
		} else {
			// use default string representation
			fmt.Fprint(f, w.String())
		}
	case 's':
		// string format uses String() method
		fmt.Fprint(f, w.String())
	default:
		// handle unsupported format verbs
		fmt.Fprintf(f, "%%!%c(WitnessedOrder=%s)", verb, w.String())
	}
}

// BlockI interface represents a blockchain block
type BlockI interface {
	Hash() string
	ParentHash() string
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

// OrderStore defines the methods that are required for order persistence.
type OrderStore interface {
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
	// ArchiveOrder archives a witnessed order to the archive directory for historical retention
	ArchiveOrder(order *WitnessedOrder, orderType OrderType) lib.ErrorI
}

type BlockProvider interface {
	// Start the block provider at height
	Start(ctx context.Context, height uint64)
	// Block returns the channel this provider will send new blocks through
	BlockCh() chan BlockI
	// IsSynced returns whether the provider has synced to the top of the chain
	IsSynced() bool
}

// TokenInfo holds the basic information about an ERC20 token
type TokenInfo struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// String returns a formatted string representation of TokenInfo
func (t TokenInfo) String() string {
	// return formatted string with token information
	return fmt.Sprintf("TokenInfo{Name: %s, Symbol: %s, Decimals: %d}",
		t.Name, t.Symbol, t.Decimals)
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

// String returns a formatted string representation of TokenTransfer
func (t TokenTransfer) String() string {
	// calculate decimal amount for display
	decimalAmount, err := t.DecimalAmount()
	var amountStr string
	if err != nil {
		// fallback to base amount if decimal conversion fails
		amountStr = fmt.Sprintf("BaseAmount: %s (decimal conversion failed: %v)",
			t.TokenBaseAmount.String(), err)
	} else {
		// show both decimal and base amounts
		amountStr = fmt.Sprintf("Amount: %.6f %s (Base: %s)",
			decimalAmount, t.TokenInfo.Symbol, t.TokenBaseAmount.String())
	}
	// return formatted string with transfer details
	return fmt.Sprintf("TokenTransfer{Blockchain: %s, %s, TxID: %s, From: %s, To: %s, Contract: %s}",
		t.Blockchain, amountStr, t.TransactionID, t.SenderAddress,
		t.RecipientAddress, t.ContractAddress)
}
