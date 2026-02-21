package oracle

import (
	"fmt"
	"strings"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/xeipuuv/gojsonschema"
)

const (
	// length of valid order id strings
	orderIdLen = 40
	// length of Canopy addresses in bytes
	canopyAddressLenBytes = 20
	// JSON schema for lock order validation
	lockOrderSchema = `{
		"type": "object",
		"properties": {
			"orderId": {
				"type": "string",
				"minLength": 40,
				"maxLength": 40
			},
			"chain_id": {
				"type": "integer"
			},
			"buyerSendAddress": {
				"type": "string"
			},
			"buyerReceiveAddress": {
				"type": "string"
			},
			"buyerChainDeadline": {
				"type": "integer",
				"minimum": 1
			}
		},
		"required": ["orderId", "chain_id", "buyerSendAddress", "buyerReceiveAddress", "buyerChainDeadline"],
		"additionalProperties": false
	}`
	// JSON schema for close order validation
	closeOrderSchema = `{
		"type": "object",
		"properties": {
			"orderId": {
				"type": "string",
				"minLength": 40,
				"maxLength": 40
			},
			"chain_id": {
				"type": "integer"
			},
			"closeOrder": {
				"type": "boolean",
				"const": true
			}
		},
		"required": ["orderId", "chain_id", "closeOrder"],
		"additionalProperties": false
	}`
)

// OrderValidator provides validation functionality for lock and close orders
type OrderValidator struct {
	// JSON schema loaders for validation
	lockOrderSchemaLoader  gojsonschema.JSONLoader
	closeOrderSchemaLoader gojsonschema.JSONLoader
}

// NewOrderValidator creates a new OrderValidator instance with predefined schemas
func NewOrderValidator() *OrderValidator {
	return &OrderValidator{
		lockOrderSchemaLoader:  gojsonschema.NewStringLoader(lockOrderSchema),
		closeOrderSchemaLoader: gojsonschema.NewStringLoader(closeOrderSchema),
	}
}

// validateOrderJsonSchema validates JSON using gojsonschema
func (v *OrderValidator) ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error {
	var schemaLoader gojsonschema.JSONLoader
	switch orderType {
	case types.LockOrderType:
		// use pre-created schema loader
		schemaLoader = v.lockOrderSchemaLoader
	case types.CloseOrderType:
		// use pre-created schema loader
		schemaLoader = v.closeOrderSchemaLoader
	default:
		return fmt.Errorf("error validating json bytes: Invalid order type")
	}
	// create document loader
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)
	// validate document against schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}
	// check if validation passed
	if !result.Valid() {
		var errorMessages []string
		for _, desc := range result.Errors() {
			errorMessages = append(errorMessages, desc.String())
		}
		return fmt.Errorf("schema validation failed: %s", strings.Join(errorMessages, "; "))
	}
	return nil
}
