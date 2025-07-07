package eth

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
)

func TestValidOrderJson(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		requiredFields []string
		expectError    bool
	}{
		{
			name:           "valid close order",
			input:          []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    false,
		},
		{
			name:           "valid close order with extra fields",
			input:          []byte(`{"orderId":"12345678901234567890","chain_id":1,"closeOrder":true,"status":"closed"}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			input:          []byte(`{"orderId":"123456789012345678901234567890123456789"`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "empty JSON",
			input:          []byte(`{}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "missing orderId",
			input:          []byte(`{"chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "missing chain_id",
			input:          []byte(`{"orderId":"12345678901234567890","closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "missing closeOrder",
			input:          []byte(`{"orderId":"12345678901234567890","chain_id":1}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "empty orderId",
			input:          []byte(`{"orderId":"","chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "null orderId",
			input:          []byte(`{"orderId":null,"chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "orderId as number",
			input:          []byte(`{"orderId":1234567890123456789012345678901234567890,"chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "wrong length orderId",
			input:          []byte(`{"orderId":"123","chain_id":1,"closeOrder":true}`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "just a string",
			input:          []byte(`not json`),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "empty byte array",
			input:          []byte(``),
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "nil byte array",
			input:          nil,
			requiredFields: closeOrderRequiredFields,
			expectError:    true,
		},
		{
			name:           "valid lock order",
			input:          []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x123","buyerReceiveAddress":"0x456","buyerChainDeadline":1234567890}`),
			requiredFields: lockOrderRequiredFields,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validOrderJson(tt.input, tt.requiredFields)
			if (err != nil) != tt.expectError {
				t.Errorf("validOrderJson() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestUnmarshalValidatedLockOrder(t *testing.T) {
	// Valid 20-byte addresses (40 hex characters)
	validAddress1 := "0123456789abcdef0123456789abcdef01234567"
	validAddress2 := "fedcba9876543210fedcba9876543210fedcba98"
	// Valid 40-character order ID
	validOrderID := "1234567890123456789012345678901234567890"

	tests := []struct {
		name        string
		input       []byte
		expectError bool
		validate    func(*testing.T, *lib.LockOrder)
	}{
		{
			name: "valid buyer address lengths",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"buyerSendAddress": "` + validAddress1 + `",
				"buyerReceiveAddress": "` + validAddress2 + `",
				"buyerChainDeadline": 1234567890
			}`),
			expectError: false,
			validate: func(t *testing.T, lo *lib.LockOrder) {
				if len(lo.BuyerSendAddress) != 20 {
					t.Errorf("Expected BuyerSendAddress length 20, got %d", len(lo.BuyerSendAddress))
				}
				if len(lo.BuyerReceiveAddress) != 20 {
					t.Errorf("Expected BuyerReceiveAddress length 20, got %d", len(lo.BuyerReceiveAddress))
				}
			},
		},
		{
			name: "invalid buyerSendAddress length - too short",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"buyerSendAddress": "123",
				"buyerReceiveAddress": "` + validAddress2 + `",
				"buyerChainDeadline": 1234567890
			}`),
			expectError: true,
		},
		{
			name: "invalid buyerSendAddress length - too long",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"buyerSendAddress": "` + validAddress1 + `1234567890",
				"buyerReceiveAddress": "` + validAddress2 + `",
				"buyerChainDeadline": 1234567890
			}`),
			expectError: true,
		},
		{
			name: "invalid buyerReceiveAddress length - too short",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"buyerSendAddress": "` + validAddress1 + `",
				"buyerReceiveAddress": "123",
				"buyerChainDeadline": 1234567890
			}`),
			expectError: true,
		},
		{
			name: "invalid buyerReceiveAddress length - too long",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"buyerSendAddress": "` + validAddress1 + `",
				"buyerReceiveAddress": "` + validAddress2 + `1234567890",
				"buyerChainDeadline": 1234567890
			}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unmarshalValidatedLockOrder(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if result != nil {
					t.Errorf("Expected nil result when error occurs, got %+v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Errorf("Expected valid result but got nil")
				} else if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestUnmarshalValidatedCloseOrder(t *testing.T) {
	// Valid 40-character order ID
	validOrderID := "1234567890123456789012345678901234567890"

	tests := []struct {
		name        string
		input       []byte
		expectError bool
		validate    func(*testing.T, *lib.CloseOrder)
	}{
		{
			name: "valid close order",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"closeOrder": true
			}`),
			expectError: false,
			validate: func(t *testing.T, co *lib.CloseOrder) {
				if !co.CloseOrder {
					t.Errorf("Expected CloseOrder to be true, got %v", co.CloseOrder)
				}
			},
		},
		{
			name: "invalid closeOrder field - false",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1,
				"closeOrder": false
			}`),
			expectError: true,
		},
		{
			name: "missing closeOrder field",
			input: []byte(`{
				"orderId": "` + validOrderID + `",
				"chain_id": 1
			}`),
			expectError: true,
		},
		{
			name: "invalid orderId length",
			input: []byte(`{
				"orderId": "123",
				"chain_id": 1,
				"closeOrder": true
			}`),
			expectError: true,
		},
		{
			name: "missing orderId field",
			input: []byte(`{
				"chain_id": 1,
				"closeOrder": true
			}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unmarshalValidatedCloseOrder(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if result != nil {
					t.Errorf("Expected nil result when error occurs, got %+v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Errorf("Expected valid result but got nil")
				} else if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}
