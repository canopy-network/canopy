package lib

import (
	"bytes"
	"math"
)

func (x *DexLimitOrder) MinAsk() float64 {
	return float64(x.RequestedAmount) / float64(x.AmountForSale)
}

func (x *DexLimitOrder) MapKey() uint64 {
	bz, _ := Marshal(x)
	return MemHash(bz)
}

func (x *DexLimitOrder) SurplusMetric() float64 {
	return math.Pow(float64(x.AmountForSale), 2) / x.MinAsk()
}

func (x *DexLimitOrder) Copy() *DexLimitOrder {
	// defensive
	if x == nil {
		return nil
	}
	return &DexLimitOrder{
		AmountForSale:   x.AmountForSale,
		RequestedAmount: x.RequestedAmount,
		Address:         bytes.Clone(x.Address),
	}
}

func (x *DexBatch) IsEmpty() bool {
	if x == nil {
		return true
	}
	return len(x.Receipts) == 0 && len(x.Orders) == 0
}

func (x *DexBatch) Copy() *DexBatch {
	if x == nil {
		return nil
	}
	orders := make([]*DexLimitOrder, len(x.Orders))
	for i, order := range x.Orders {
		orders[i] = order.Copy()
	}
	return &DexBatch{Orders: orders}
}

func (x *DexLimitOrder) Equals(y *DexLimitOrder) bool {
	// defensive
	if x == nil || y == nil {
		return false
	}
	// compare requested amounts
	if x.RequestedAmount != y.RequestedAmount {
		return false
	}
	// compare amount for sale
	if x.AmountForSale != y.AmountForSale {
		return false
	}
	// compare addresses
	return bytes.Equal(x.Address, y.Address)
}
