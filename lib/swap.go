package lib

import (
	"encoding/json"
)

// AddOrder() adds a sell order to the OrderBook
func (x *OrderBook) AddOrder(order *SellOrder) (id uint64) {
	// if there's an empty slot, fill it with the sell order
	for i, slot := range x.Orders {
		if slot == nil {
			id = uint64(i)
			order.Id = id
			x.Orders[i] = order
			return
		}
	}
	// if there's no empty slots, add the sell order to the
	id = uint64(len(x.Orders))
	order.Id = id
	x.Orders = append(x.Orders, order)
	return
}

// LockOrder() adds a recipient address and deadline height to the order to 'claim' the order and prevent others from 'claiming it'
func (x *OrderBook) LockOrder(orderId int, buyersReceiveAddress, buyersSendAddress []byte, buyerChainDeadlineHeight uint64) ErrorI {
	order, err := x.GetOrder(orderId)
	if err != nil {
		return err
	}
	if order.BuyerReceiveAddress != nil {
		return ErrOrderAlreadyAccepted()
	}
	order.BuyerReceiveAddress, order.BuyerSendAddress, order.BuyerChainDeadline = buyersReceiveAddress, buyersSendAddress, buyerChainDeadlineHeight
	x.Orders[orderId] = order
	return nil
}

// ResetOrder() removes a recipient address and the deadline height from the order to 'un-claim' the order
func (x *OrderBook) ResetOrder(orderId int) ErrorI {
	order, err := x.GetOrder(orderId)
	if err != nil {
		return err
	}
	order.BuyerReceiveAddress, order.BuyerSendAddress, order.BuyerChainDeadline = nil, nil, 0
	x.Orders[orderId] = order
	return nil
}

// UpdateOrder() updates a sell order to the OrderBook, passing a nil `order` is effectively a delete operation
func (x *OrderBook) UpdateOrder(orderId int, order *SellOrder) (err ErrorI) {
	numOfOrderSlots := len(x.Orders)
	if orderId >= numOfOrderSlots {
		return ErrOrderNotFound(orderId)
	}
	// if deleting from the end, shrink the slice
	if order == nil && orderId == numOfOrderSlots-1 {
		x.Orders = x.Orders[:numOfOrderSlots-1]
		// continue shrinking the slice if nil entries are at the end
		for i := numOfOrderSlots - 2; i >= 0; i-- {
			if x.Orders[i] != nil {
				break
			}
			x.Orders = x.Orders[:i]
		}
		return
	}
	// if not deleting from the end of the slice,
	// simply replace the order
	x.Orders[orderId] = order
	return
}

// GetOrder() retrieves a sell order from the OrderBook
func (x *OrderBook) GetOrder(orderId int) (order *SellOrder, err ErrorI) {
	numOfOrderSlots := len(x.Orders)
	if orderId >= numOfOrderSlots || x.Orders[orderId] == nil {
		return nil, ErrOrderNotFound(orderId)
	}
	order = x.Orders[orderId]
	return
}

// jsonSellOrder is the json.Marshaller and json.Unmarshaler implementation for the SellOrder object
type jsonSellOrder struct {
	Id                   uint64   `json:"id,omitempty"`                   // the unique identifier of the order
	Committee            uint64   `json:"committee,omitempty"`            // the id of the committee that is in-charge of escrow for the swap
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
func (x *SellOrder) UnmarshalJSON(bz []byte) error {
	j := new(jsonSellOrder)
	if err := json.Unmarshal(bz, j); err != nil {
		return err
	}
	*x = SellOrder{
		Id:                   j.Id,
		Committee:            j.Committee,
		AmountForSale:        j.AmountForSale,
		RequestedAmount:      j.RequestedAmount,
		SellerReceiveAddress: j.SellerReceiveAddress,
		BuyerSendAddress:     j.BuyerSendAddress,
		BuyerReceiveAddress:  j.BuyerReceiveAddress,
		BuyerChainDeadline:   j.BuyerChainDeadline,
		SellersSendAddress:   j.SellersSellAddress,
	}
	return nil
}

// MarshalJSON() is the json.Marshaller implementation for the OrderBooks object
func (x OrderBooks) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.OrderBooks)
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the OrderBooks object
func (x *OrderBooks) UnmarshalJSON(bz []byte) error {
	jsonOrderBooks := new([]*OrderBook)
	if err := json.Unmarshal(bz, jsonOrderBooks); err != nil {
		return err
	}
	*x = OrderBooks{
		OrderBooks: *jsonOrderBooks,
	}
	return nil
}
