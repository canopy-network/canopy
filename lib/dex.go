package lib

import (
	"bytes"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
)

func (x *DexLimitOrder) MinAsk() float64 {
	return float64(x.RequestedAmount) / float64(x.AmountForSale)
}

func (x *DexLimitOrder) MapKey(blockHash []byte) string {
	bz, _ := Marshal(x)
	return crypto.HashString(append(blockHash, bz...))
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
	return len(x.Receipts) == 0 && len(x.Orders) == 0 && len(x.Withdraws) == 0 && len(x.Deposits) == 0
}

func (x *DexBatch) EnsureNonNil() {
	if x.Orders == nil {
		x.Orders = []*DexLimitOrder{}
	}
	if x.Deposits == nil {
		x.Deposits = []*DexLiquidityDeposit{}
	}
	if x.Withdraws == nil {
		x.Withdraws = []*DexLiquidityWithdraw{}
	}
	if x.Receipts == nil {
		x.Receipts = []bool{}
	}
}

func (x *DexBatch) CopyOrders() []*DexLimitOrder {
	if x == nil {
		return nil
	}
	orders := make([]*DexLimitOrder, len(x.Orders))
	for i, order := range x.Orders {
		orders[i] = order.Copy()
	}
	return orders
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
