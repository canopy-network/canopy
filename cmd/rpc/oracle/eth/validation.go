package eth

import (
	"encoding/json"
	"errors"

	"github.com/canopy-network/canopy/lib"
)

var (
	closeOrderRequiredFields = []string{"orderId", "chain_id", "closeOrder"}
	lockOrderRequiredFields  = []string{"orderId", "chain_id", "buyerSendAddress", "buyerReceiveAddress", "buyerChainDeadline"}
)

const (
	// length of valid order ids, in bytes
	orderIdLen = 40
	// length of Canopy addresses in bytes
	canopyAddressLenBytes = 20
)

// validOrderJson validates JSON format, required fields and order id length
func validOrderJson(jsonBytes []byte, requiredFields []string) error {
	var rawData map[string]any
	// jsonBytes is valid JSON
	err := json.Unmarshal(jsonBytes, &rawData)
	if err != nil {
		return err
	}
	// every required field is present
	for _, field := range requiredFields {
		if _, exists := rawData[field]; !exists {
			return errors.New("required fields not present")
		}
	}
	// only required fields were present, no other ones
	if len(rawData) != len(requiredFields) {
		return errors.New("Incorrect field count in JSON data")
	}
	// attempt to coerce orderId field data to string
	orderId, ok := rawData["orderId"].(string)
	if !ok {
		return errors.New("orderId must be a string")
	}
	// compare byte lengths
	if len(orderId) != orderIdLen {
		return errors.New("Invalid order id length")
	}
	return nil
}

// unmarshalValidatedLockOrder validates and unmarshals lock order byte data
func unmarshalValidatedLockOrder(data []byte) (*lib.LockOrder, error) {
	err := validOrderJson(data, lockOrderRequiredFields)
	if err != nil {
		return nil, err
	}
	lockOrder := &lib.LockOrder{}
	// unmarshal the validated json data
	err = lockOrder.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	if len(lockOrder.BuyerReceiveAddress) != canopyAddressLenBytes {
		return nil, errors.New("BuyerReceiveAddress invalid length")
	}
	if len(lockOrder.BuyerSendAddress) != canopyAddressLenBytes {
		return nil, errors.New("BuyerSendAddress invalid length")
	}
	// return validated lock order
	return lockOrder, nil
}

// unmarshalValidatedCloseOrder validates and unmarshals close order byte data
func unmarshalValidatedCloseOrder(data []byte) (*lib.CloseOrder, error) {
	err := validOrderJson(data, closeOrderRequiredFields)
	if err != nil {
		return nil, err
	}
	closeOrder := &lib.CloseOrder{}
	// unmarshal the validated json data
	err = closeOrder.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}

	if closeOrder.CloseOrder == false {
		return nil, errors.New("Invalid Close Order: CloseOrder field set to false")
	}
	// return validated close order
	return closeOrder, nil
}
