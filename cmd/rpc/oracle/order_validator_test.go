package oracle

import (
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
)

func TestValidateCloseOrderJsonBytes(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		requiredFields []string
		expectError    bool
	}{
		{
			name:        "valid close order",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"closeOrder":true}`),
			expectError: false,
		},
		{
			name:        "valid close order but closeOrder false",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"closeOrder":false}`),
			expectError: true,
		},
		{
			name:        "valid close order with extra fields",
			input:       []byte(`{"orderId":"12345678901234567890","chain_id":1,"closeOrder":true,"status":"closed"}`),
			expectError: true,
		},
		{
			name:        "invalid JSON",
			input:       []byte(`{"orderId":"123456789012345678901234567890123456789"`),
			expectError: true,
		},
		{
			name:        "empty JSON",
			input:       []byte(`{}`),
			expectError: true,
		},
		{
			name:        "missing orderId",
			input:       []byte(`{"chain_id":1,"closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "missing chain_id",
			input:       []byte(`{"orderId":"12345678901234567890","closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "missing closeOrder",
			input:       []byte(`{"orderId":"12345678901234567890","chain_id":1}`),
			expectError: true,
		},
		{
			name:        "empty orderId",
			input:       []byte(`{"orderId":"","chain_id":1,"closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "null orderId",
			input:       []byte(`{"orderId":null,"chain_id":1,"closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "orderId as number",
			input:       []byte(`{"orderId":1234567890123456789012345678901234567890,"chain_id":1,"closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "wrong length orderId",
			input:       []byte(`{"orderId":"123","chain_id":1,"closeOrder":true}`),
			expectError: true,
		},
		{
			name:        "just a string",
			input:       []byte(`not json`),
			expectError: true,
		},
		{
			name:        "empty byte array",
			input:       []byte(``),
			expectError: true,
		},
		{
			name:        "nil byte array",
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewOrderValidator()
			var err error
			err = validator.ValidateOrderJsonBytes(tt.input, types.CloseOrderType)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateOrderjsonBytes() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateLockOrderJsonBytes(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		requiredFields []string
		expectError    bool
	}{
		{
			name:        "valid lock order",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: false,
		},
		{
			name:        "valid lock order except object as a value",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":{},"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "invalid json - malformed",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890`),
			expectError: true,
		},
		{
			name:        "invalid json - missing closing brace",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1`),
			expectError: true,
		},
		{
			name:        "invalid json - missing quotes",
			input:       []byte(`{orderId:"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "invalid json - trailing comma",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890,}`),
			expectError: true,
		},
		{
			name:        "empty object",
			input:       []byte(`{}`),
			expectError: true,
		},
		{
			name:        "empty json",
			input:       []byte(``),
			expectError: true,
		},
		{
			name:        "null json",
			input:       []byte(`null`),
			expectError: true,
		},
		{
			name:        "array instead of object",
			input:       []byte(`["orderId","1234567890123456789012345678901234567890"]`),
			expectError: true,
		},
		{
			name:        "string instead of object",
			input:       []byte(`"not an object"`),
			expectError: true,
		},
		{
			name:        "number instead of object",
			input:       []byte(`123456`),
			expectError: true,
		},
		{
			name:        "boolean instead of object",
			input:       []byte(`true`),
			expectError: true,
		},
		{
			name:        "missing required field orderId",
			input:       []byte(`{"chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "missing required field chain_id",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "missing required field buyerSendAddress",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "missing required field buyerReceiveAddress",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "missing required field buyerChainDeadline",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890"}`),
			expectError: true,
		},
		{
			name:        "null values for required fields",
			input:       []byte(`{"orderId":null,"chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "empty string values",
			input:       []byte(`{"orderId":"","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "array as field value",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":["0x12345678901234567890"],"buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "nested object in field",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":{"address":"0x12345678901234567890"},"buyerChainDeadline":1234567890}`),
			expectError: true,
		},
		{
			name:        "extra fields with valid required fields",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":1234567890,"extraField":"value"}`),
			expectError: true,
		},
		{
			name:        "zero value for buyer chain deadline",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":0}`),
			expectError: true,
		},
		{
			name:        "negative value for buyer chain deadline",
			input:       []byte(`{"orderId":"1234567890123456789012345678901234567890","chain_id":1,"buyerSendAddress":"0x12345678901234567890","buyerReceiveAddress":"0x12345678901234567890","buyerChainDeadline":-1}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewOrderValidator()
			var err error
			err = validator.ValidateOrderJsonBytes(tt.input, types.LockOrderType)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateOrderjsonBytes() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
