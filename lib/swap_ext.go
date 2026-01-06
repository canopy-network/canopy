package lib

// IsLocked returns true if the SellOrder has a buyer (is locked)
func (x *SellOrder) IsLocked() bool {
	if x == nil {
		return false
	}
	if x.BuyerReceiveAddress == nil {
		return false
	}
	return true
}

// Copy returns a reference to a clone of the SellOrder
func (s *SellOrder) Copy() *SellOrder {
	if s == nil {
		return nil
	}
	return &SellOrder{
		Id:                   append([]byte(nil), s.Id...),
		Committee:            s.Committee,
		Data:                 append([]byte(nil), s.Data...),
		AmountForSale:        s.AmountForSale,
		RequestedAmount:      s.RequestedAmount,
		SellerReceiveAddress: append([]byte(nil), s.SellerReceiveAddress...),
		BuyerSendAddress:     append([]byte(nil), s.BuyerSendAddress...),
		BuyerReceiveAddress:  append([]byte(nil), s.BuyerReceiveAddress...),
		BuyerChainDeadline:   s.BuyerChainDeadline,
		SellersSendAddress:   append([]byte(nil), s.SellersSendAddress...),
	}
}

// Copy returns a reference to a clone of the OrderBook
func (o *OrderBook) Copy() *OrderBook {
	if o == nil {
		return nil
	}
	ordersCopy := make([]*SellOrder, len(o.Orders))
	for i, order := range o.Orders {
		ordersCopy[i] = order.Copy()
	}
	return &OrderBook{
		ChainId: o.ChainId,
		Orders:  ordersCopy,
	}
}
