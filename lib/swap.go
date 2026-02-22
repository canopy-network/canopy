package lib

import (
	"bytes"
	"encoding/json"
)

/* This file implements 'sell order book' logic for token swaps that is used throughout the app */

// GetOrder() retrieves a sell order from the OrderBook
func (x *OrderBook) GetOrder(orderId []byte) (order *SellOrder, err ErrorI) {
	// ensure non-nil
	if x == nil {
		return nil, ErrEmptyOrderBook()
	}
	// for each order in the book
	for _, order = range x.Orders {
		// if order is found
		if bytes.Equal(order.Id, orderId) {
			// exit with order
			return
		}
	}
	return nil, nil
}

// Empty() indicates whether the sell order is null
func (x *SellOrder) Empty() bool {
	return x == nil || x.SellersSendAddress == nil
}

// jsonSellOrder is the json.Marshaller and json.Unmarshaler implementation for the SellOrder object
type jsonSellOrder struct {
	Id                   HexBytes `json:"id,omitempty"`                   // the unique identifier of the order
	Committee            uint64   `json:"committee,omitempty"`            // the id of the committee that is in-charge of escrow for the swap
	Data                 HexBytes `json:"data,omitempty"`                 // generic data for the swap to allow additional functionality
	AmountForSale        uint64   `json:"amountForSale,omitempty"`        // amount of CNPY for sale
	RequestedAmount      uint64   `json:"requestedAmount,omitempty"`      // amount of 'token' to receive
	SellerReceiveAddress HexBytes `json:"sellerReceiveAddress,omitempty"` // the external chain address to receive the 'token'
	BuyerSendAddress     HexBytes `json:"buyerSendAddress,omitempty"`     // the send address from the buyer
	BuyerReceiveAddress  HexBytes `json:"buyerReceiveAddress,omitempty"`  // the buyers address to receive the 'coin'
	BuyerChainDeadline   uint64   `json:"buyerChainDeadline,omitempty"`   // the external chain height deadline to send the 'tokens' to SellerReceiveAddress
	SellersSellAddress   HexBytes `json:"sellersSendAddress,omitempty"`   // the address of seller who is selling the 'coin'
}

// MarshalJSON() is the json.Marshaller implementation for the SellOrder object
func (x SellOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSellOrder{
		Id:                   x.Id,
		Committee:            x.Committee,
		Data:                 x.Data,
		AmountForSale:        x.AmountForSale,
		RequestedAmount:      x.RequestedAmount,
		SellerReceiveAddress: x.SellerReceiveAddress,
		BuyerSendAddress:     x.BuyerSendAddress,
		BuyerReceiveAddress:  x.BuyerReceiveAddress,
		BuyerChainDeadline:   x.BuyerChainDeadline,
		SellersSellAddress:   x.SellersSendAddress,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the SellOrder object
func (x *SellOrder) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non nil result
	j := new(jsonSellOrder)
	// populate the json object using the json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying sell order using the json object
	*x = SellOrder{
		Id:                   j.Id,
		Committee:            j.Committee,
		Data:                 j.Data,
		AmountForSale:        j.AmountForSale,
		RequestedAmount:      j.RequestedAmount,
		SellerReceiveAddress: j.SellerReceiveAddress,
		BuyerSendAddress:     j.BuyerSendAddress,
		BuyerReceiveAddress:  j.BuyerReceiveAddress,
		BuyerChainDeadline:   j.BuyerChainDeadline,
		SellersSendAddress:   j.SellersSellAddress,
	}
	// exit
	return
}

// MarshalJSON() is the json.Marshaller implementation for the OrderBooks object
func (x OrderBooks) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.OrderBooks)
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the OrderBooks object
func (x *OrderBooks) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object ref to ensure a non nil result
	jsonOrderBooks := new([]*OrderBook)
	// populate the object using json bytes
	if err = json.Unmarshal(jsonBytes, jsonOrderBooks); err != nil {
		// exit
		return
	}
	// populate the underlying object using the json object
	*x = OrderBooks{
		OrderBooks: *jsonOrderBooks,
	}
	// exit
	return
}

// SellOrders is a slice of SellOrder pointers that implements the Pageable interface
type SellOrders []*SellOrder

// Len returns the number of orders in the slice
func (s *SellOrders) Len() int { return len(*s) }

// New returns a new empty SellOrders instance (satisfies Pageable interface)
func (s *SellOrders) New() Pageable { return &SellOrders{} }
