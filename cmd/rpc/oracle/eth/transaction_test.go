package eth

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/ethereum/go-ethereum/common"
)

// configurableValidator returns validationErr for LockOrderType and CloseOrderType
// as configured, letting tests drive both the "valid order" and "not an order" paths.
type configurableValidator struct {
	lockErr  error
	closeErr error
}

func (c *configurableValidator) ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error {
	if orderType == types.LockOrderType {
		return c.lockErr
	}
	return c.closeErr
}

func createERC20TransferData(recipient string, amount *big.Int, extra []byte) []byte {
	// Initialize empty byte slice to build the transaction data
	data := make([]byte, 0)
	// Decode the ERC20 transfer method ID from hex string and append to data
	methodIDBytes, _ := hex.DecodeString(erc20TransferMethodID)
	data = append(data, methodIDBytes...)
	// Remove "0x" prefix from recipient address and decode from hex
	recipientBytes, _ := hex.DecodeString(recipient[2:])
	// Create 32-byte padded recipient address (addresses are 20 bytes, padded with 12 zero bytes at start)
	paddedRecipient := make([]byte, 32)
	copy(paddedRecipient[12:], recipientBytes)
	// Print recipient bytes for debugging (should be removed in production)
	// Append padded recipient address to transaction data
	data = append(data, paddedRecipient...)
	// Convert big.Int amount to bytes
	amountBytes := amount.Bytes()
	// Create 32-byte padded amount (right-aligned, padded with zeros at start)
	paddedAmount := make([]byte, 32)
	copy(paddedAmount[32-len(amountBytes):], amountBytes)
	// Append padded amount to transaction data
	data = append(data, paddedAmount...)
	// If extra data is provided, append it to the transaction data
	if extra != nil {
		data = append(data, extra...)
	}
	// Return the complete encoded transaction data
	return data
}

func TestParseERC20Transfer(t *testing.T) {
	recipient1 := common.HexToAddress("0x742d35cc6634c0532925a3b8d0c9e3e0c8b0e8c2").Hex()
	recipient2 := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678").Hex()
	amount1 := big.NewInt(1000000000000000000)
	amount2 := big.NewInt(500000000000000000)
	extraData1 := []byte("extra")
	extraData2 := []byte("more data")

	tests := []struct {
		name              string
		data              []byte
		expectedRecipient string
		expectedAmount    *big.Int
		expectedExtra     []byte
		expectedError     error
	}{
		{
			name:              "valid erc20 transfer with no extra data",
			data:              createERC20TransferData(recipient1, amount1, nil),
			expectedRecipient: recipient1,
			expectedAmount:    amount1,
			expectedExtra:     nil,
			expectedError:     nil,
		},
		{
			name:              "valid erc20 transfer with extra data",
			data:              createERC20TransferData(recipient2, amount2, extraData1),
			expectedRecipient: recipient2,
			expectedAmount:    amount2,
			expectedExtra:     extraData1,
			expectedError:     nil,
		},
		{
			name:              "valid erc20 transfer with longer extra data",
			data:              createERC20TransferData(recipient1, amount2, extraData2),
			expectedRecipient: recipient1,
			expectedAmount:    amount2,
			expectedExtra:     extraData2,
			expectedError:     nil,
		},
		{
			name:              "nil data",
			data:              nil,
			expectedRecipient: "",
			expectedAmount:    nil,
			expectedExtra:     nil,
			expectedError:     ErrNotERC20Transfer,
		},
		{
			name:              "empty data",
			data:              []byte{},
			expectedRecipient: "",
			expectedAmount:    nil,
			expectedExtra:     nil,
			expectedError:     ErrNotERC20Transfer,
		},
		{
			name:              "data too short",
			data:              []byte{0xa9, 0x05, 0x9c, 0xbb, 0x00, 0x00},
			expectedRecipient: "",
			expectedAmount:    nil,
			expectedExtra:     nil,
			expectedError:     ErrNotERC20Transfer,
		},
		{
			name:              "wrong method id",
			data:              append([]byte{0x12, 0x34, 0x56, 0x78}, make([]byte, 64)...),
			expectedRecipient: "",
			expectedAmount:    nil,
			expectedExtra:     nil,
			expectedError:     ErrNotERC20Transfer,
		},
		{
			name:              "exact minimum length",
			data:              createERC20TransferData(recipient1, big.NewInt(0), nil),
			expectedRecipient: recipient1,
			expectedAmount:    big.NewInt(0),
			expectedExtra:     nil,
			expectedError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipient, amount, extra, err := parseERC20Transfer(tt.data)
			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
			if recipient != tt.expectedRecipient {
				t.Errorf("expected recipient %s, got %s", tt.expectedRecipient, recipient)
			}
			if tt.expectedAmount != nil && amount != nil {
				if amount.Cmp(tt.expectedAmount) != 0 {
					t.Errorf("expected amount %s, got %s", tt.expectedAmount.String(), amount.String())
				}
			} else if tt.expectedAmount != amount {
				t.Errorf("expected amount %v, got %v", tt.expectedAmount, amount)
			}
			if len(tt.expectedExtra) != len(extra) {
				t.Errorf("expected extra data length %d, got %d", len(tt.expectedExtra), len(extra))
			}
			for i, b := range tt.expectedExtra {
				if i < len(extra) && extra[i] != b {
					t.Errorf("expected extra data byte %d to be %x, got %x", i, b, extra[i])
				}
			}
		})
	}
}

// TestParseERC20Transfer_USDT verifies that USDT transfers parse identically to standard ERC20 transfers
// USDT has a non-standard implementation (returns void instead of bool from transfer()),
// but this doesn't affect parsing since we parse the INPUT data, not return values
func TestParseERC20Transfer_USDT(t *testing.T) {
	// Use USDT mainnet contract address
	usdtContract := USDTMainnet.Hex()
	// USDT uses 6 decimals, so 1 USDT = 1,000,000 base units
	oneUSDT := big.NewInt(1000000)
	recipient := common.HexToAddress("0x742d35cc6634c0532925a3b8d0c9e3e0c8b0e8c2").Hex()
	orderJSON := []byte(`{"order_id":"abc123"}`)

	tests := []struct {
		name              string
		amount            *big.Int
		extraData         []byte
		expectedRecipient string
		expectedAmount    *big.Int
		expectedExtra     []byte
	}{
		{
			name:              "USDT transfer with order data",
			amount:            oneUSDT,
			extraData:         orderJSON,
			expectedRecipient: recipient,
			expectedAmount:    oneUSDT,
			expectedExtra:     orderJSON,
		},
		{
			name:              "USDT transfer without extra data",
			amount:            big.NewInt(5000000), // 5 USDT
			extraData:         nil,
			expectedRecipient: recipient,
			expectedAmount:    big.NewInt(5000000),
			expectedExtra:     nil,
		},
		{
			name:              "USDT zero amount transfer (lock order)",
			amount:            big.NewInt(0),
			extraData:         orderJSON,
			expectedRecipient: recipient,
			expectedAmount:    big.NewInt(0),
			expectedExtra:     orderJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create ERC20 transfer data (same format for USDT and standard ERC20)
			data := createERC20TransferData(tt.expectedRecipient, tt.amount, tt.extraData)

			// Parse the transfer
			parsedRecipient, parsedAmount, parsedExtra, err := parseERC20Transfer(data)

			// Verify parsing succeeds
			if err != nil {
				t.Errorf("USDT transfer parsing failed: %v", err)
			}

			// Verify recipient matches
			if parsedRecipient != tt.expectedRecipient {
				t.Errorf("expected recipient %s, got %s", tt.expectedRecipient, parsedRecipient)
			}

			// Verify amount matches
			if parsedAmount.Cmp(tt.expectedAmount) != 0 {
				t.Errorf("expected amount %s, got %s", tt.expectedAmount.String(), parsedAmount.String())
			}

			// Verify extra data matches
			if len(parsedExtra) != len(tt.expectedExtra) {
				t.Errorf("expected extra data length %d, got %d", len(tt.expectedExtra), len(parsedExtra))
			}
			for i, b := range tt.expectedExtra {
				if parsedExtra[i] != b {
					t.Errorf("expected extra data byte %d to be %x, got %x", i, b, parsedExtra[i])
				}
			}

			// Log success - USDT parsing works identically to standard ERC20
			t.Logf("✓ USDT transfer parsed successfully (contract: %s, amount: %s)", usdtContract, parsedAmount.String())
		})
	}
}

func TestTransaction_parseDataForOrders(t *testing.T) {
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	// lib.LockOrder / lib.CloseOrder UnmarshalJSON target private structs whose
	// fields are all optional (json:",omitempty" or no presence requirement),
	// so "{}" is sufficient valid JSON for both order types.
	lockJSON := []byte(`{}`)
	closeJSON := []byte(`{}`)

	tests := []struct {
		name        string
		to          common.Address
		data        []byte
		selfSend    bool
		validator   OrderValidator
		wantOrder   bool
		wantIsERC20 bool
	}{
		{
			name:      "self-sent lock order in raw data",
			to:        other, // overridden to self via selfSend below
			data:      lockJSON,
			selfSend:  true,
			validator: &configurableValidator{lockErr: nil},
			wantOrder: true,
		},
		{
			name:        "empty data -> no order",
			to:          other,
			data:        []byte{},
			validator:   &mockOrderValidator{},
			wantOrder:   false,
			wantIsERC20: false,
		},
		{
			name:        "oversized data -> no order",
			to:          other,
			data:        make([]byte, maxTransactionDataSize+1),
			validator:   &mockOrderValidator{},
			wantOrder:   false,
			wantIsERC20: false,
		},
		{
			name:        "erc20 close order (positive amount)",
			to:          other,
			data:        createERC20TransferData(other.Hex(), big.NewInt(1000000), closeJSON),
			validator:   &configurableValidator{closeErr: nil},
			wantOrder:   true,
			wantIsERC20: true,
		},
		{
			name:        "erc20 transfer, not an order",
			to:          other,
			data:        createERC20TransferData(other.Hex(), big.NewInt(1000000), []byte("not json")),
			validator:   &configurableValidator{closeErr: ErrInvalidTransactionData},
			wantOrder:   false,
			wantIsERC20: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ethTx := createTransaction(tt.to, tt.data)
			// NewTransaction uses ethtypes.LatestSignerForChainID which panics on
			// chainId 0 (see block_provider_test.go's own use of chainId 1 for the
			// same createTransaction helper); the brief's suggested chainId 0 is
			// incorrect for this go-ethereum version.
			tx, err := NewTransaction(ethTx, 1)
			if err != nil {
				t.Fatalf("NewTransaction: %v", err)
			}
			if tt.selfSend {
				// From() is derived from the transaction signature (createTransaction
				// signs with a freshly generated key), so it can't be made to match a
				// fixed `to` address ahead of time. Override `to` directly instead to
				// deterministically exercise the self-sent branch.
				tx.to = tx.from
			}
			_ = tx.parseDataForOrders(tt.validator)
			if (tx.order != nil) != tt.wantOrder {
				t.Errorf("order presence = %v, want %v", tx.order != nil, tt.wantOrder)
			}
			if tx.isERC20 != tt.wantIsERC20 {
				t.Errorf("isERC20 = %v, want %v", tx.isERC20, tt.wantIsERC20)
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "too short", data: make([]byte, 10), want: ""},
		{name: "offset out of range", data: func() []byte {
			b := make([]byte, 64)
			b[31] = 0xff // offset = 255, beyond len
			return b
		}(), want: ""},
		{name: "valid short string", data: func() []byte {
			b := make([]byte, 96)
			b[31] = 0x20 // offset 32
			b[63] = 0x03 // length 3
			copy(b[64:], []byte("abc"))
			return b
		}(), want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeString(tt.data); got != tt.want {
				t.Errorf("decodeString = %q, want %q", got, tt.want)
			}
		})
	}
}
