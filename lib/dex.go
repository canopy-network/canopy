package lib

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/canopy-network/canopy/lib/crypto"
	"strings"
)

var (
	EmptyReceiptsHash                           = []byte(strings.Repeat("F", crypto.HashSize))
	LivenessFallbackBlocks, TriggerModuloBlocks = uint64(10), uint64(5)
)

// Hash() creates a hash representative of the dex batch
func (x *DexBatch) Hash() []byte {
	if x == nil {
		return bytes.Clone(EmptyReceiptsHash)
	}
	if x.IsEmpty() {
		x.ReceiptHash = bytes.Clone(EmptyReceiptsHash)
	}
	bz, _ := Marshal(x.Copy())
	return crypto.Hash(bz)
}

// Copy() returns a copy, omitting non-critical information
func (x *DexBatch) Copy() *DexBatch {
	if x == nil {
		return nil
	}
	return &DexBatch{
		Committee:       x.Committee,
		ReceiptHash:     x.ReceiptHash,
		Orders:          x.Orders,
		Deposits:        x.Deposits,
		Withdrawals:     x.Withdrawals,
		PoolSize:        x.PoolSize,
		CounterPoolSize: 0,
		PoolPoints:      nil,
		TotalPoolPoints: 0,
		//CounterPoolSize: 0,
		//PoolPoints:      nil,
		//TotalPoolPoints: 0,
		Receipts:         x.Receipts,
		LockedHeight:     x.LockedHeight,
		LivenessFallback: x.LivenessFallback,
	}
}

// Copy() makes a deep copy of the limit order
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

// IsEmpty() checks if a locked dex batch is 'logically' empty or not
func (x *DexBatch) IsEmpty() bool {
	if x == nil {
		return true
	}
	return len(x.ReceiptHash) == 0 && len(x.Receipts) == 0 && len(x.Orders) == 0 && len(x.Withdrawals) == 0 && len(x.Deposits) == 0
}

// EnsureNonNil() ensures the slices in the batch are not empty
// NOTE: pool points is the exception because it's omitted unless remote locked batch is pulled due to liveness fallback
func (x *DexBatch) EnsureNonNil() {
	if x.Orders == nil {
		x.Orders = []*DexLimitOrder{}
	}
	if x.Deposits == nil {
		x.Deposits = []*DexLiquidityDeposit{}
	}
	if x.Withdrawals == nil {
		x.Withdrawals = []*DexLiquidityWithdraw{}
	}
	if x.Receipts == nil {
		x.Receipts = []uint64{}
	}
}

// CopyOrders() creates 2 copies of the dex limit orders with hash keys
func (x *DexBatch) CopyOrders(blockHash []byte) (cpy1 []*DexLimitOrderWithKey, cpy2 []*DexLimitOrderWithKey) {
	if x == nil {
		return nil, nil
	}
	cpy1, cpy2 = make([]*DexLimitOrderWithKey, len(x.Orders)), make([]*DexLimitOrderWithKey, len(x.Orders))
	for i, order := range x.Orders {
		// copy 1
		cpy1[i] = &DexLimitOrderWithKey{DexLimitOrder: order.Copy()}
		cpy1[i].HashKey(i, blockHash)
		// copy 2
		cpy2[i] = &DexLimitOrderWithKey{
			DexLimitOrder: order.Copy(),
			Key:           cpy1[i].Key,
		}
	}
	return
}

type DexLimitOrderWithKey struct {
	*DexLimitOrder
	Key string
}

// Key() creates a 'HashKey' with a given block hash
func (x *DexLimitOrderWithKey) HashKey(index int, blockHash []byte) string {
	bz, _ := Marshal(x)

	// convert index to 8-byte big-endian
	idxBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idxBz, uint64(index))

	// preallocate final slice: blockHash + idxBz + bz
	data := make([]byte, 0, len(blockHash)+len(idxBz)+len(bz))
	data = append(data, blockHash...)
	data = append(data, idxBz...)
	data = append(data, bz...)

	x.Key = crypto.HashString(data)
	return x.Key
}

type dexLimitOrder struct {
	AmountForSale   uint64   `json:"amountForSale"`
	RequestedAmount uint64   `json:"requestedAmount"`
	Address         HexBytes `json:"address"`
	OrderId         HexBytes `json:"orderId"`
}

// MarshalJSON() implements the json.Marshal interface for DexLimitOrder
func (x DexLimitOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLimitOrder{
		AmountForSale:   x.AmountForSale,
		RequestedAmount: x.RequestedAmount,
		Address:         x.Address,
		OrderId:         x.OrderId,
	})
}

// UnmarshalJSON() implements the json.Unmarshaller interface for DexLimitOrder
func (x *DexLimitOrder) UnmarshalJSON(b []byte) (err error) {
	d := new(dexLimitOrder)
	if err = json.Unmarshal(b, d); err != nil {
		return err
	}
	*x = DexLimitOrder{
		AmountForSale:   d.AmountForSale,
		RequestedAmount: d.RequestedAmount,
		Address:         d.Address,
		OrderId:         d.OrderId,
	}
	return
}

type dexLiquidityDeposit struct {
	Amount  uint64   `json:"amount"`
	Address HexBytes `json:"address"`
	OrderId HexBytes `json:"orderId"`
}

// MarshalJSON() implements the json.Marshal interface for DexLiquidityDeposit
func (x DexLiquidityDeposit) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLiquidityDeposit{
		Amount:  x.Amount,
		Address: x.Address,
		OrderId: x.OrderId,
	})
}

// UnmarshalJSON() implements the json.Unmarshaller interface for DexLiquidityDeposit
func (x *DexLiquidityDeposit) UnmarshalJSON(b []byte) (err error) {
	d := new(dexLiquidityDeposit)
	if err = json.Unmarshal(b, d); err != nil {
		return err
	}
	*x = DexLiquidityDeposit{
		Amount:  d.Amount,
		Address: d.Address,
		OrderId: d.OrderId,
	}
	return
}

type dexLiquidityWithdraw struct {
	Percent uint64   `json:"percent"`
	Address HexBytes `json:"address"`
	OrderId HexBytes `json:"orderId"`
}

// MarshalJSON() implements the json.Marshal interface for dexLiquidityWithdraw
func (x DexLiquidityWithdraw) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLiquidityWithdraw{
		Percent: x.Percent,
		Address: x.Address,
		OrderId: x.OrderId,
	})
}

// UnmarshalJSON() implements the json.Unmarshaller interface for dexLiquidityWithdraw
func (x *DexLiquidityWithdraw) UnmarshalJSON(b []byte) (err error) {
	d := new(dexLiquidityWithdraw)
	if err = json.Unmarshal(b, d); err != nil {
		return err
	}
	*x = DexLiquidityWithdraw{
		Percent: d.Percent,
		Address: d.Address,
		OrderId: d.OrderId,
	}
	return
}

type dexBatch struct {
	Committee        uint64                  `json:"committee"`
	ReceiptHash      HexBytes                `json:"receiptHash"`
	Orders           []*DexLimitOrder        `json:"orders"`
	Deposits         []*DexLiquidityDeposit  `json:"deposits"`
	Withdraws        []*DexLiquidityWithdraw `json:"withdraws"`
	CounterPoolSize  uint64                  `json:"counterPoolSize"`
	PoolSize         uint64                  `json:"poolSize"`
	PoolPoints       []*PoolPoints           `json:"poolPoints"`
	TotalPoolPoints  uint64                  `json:"totalPoolPoints"`
	Receipts         []uint64                `json:"receipts"`
	LockedHeight     uint64                  `json:"locked_height"`
	LivenessFallback bool                    `json:"livenessFallback"`
}

// MarshalJSON() implements the json.Marshal interface for dex batch
func (x DexBatch) MarshalJSON() ([]byte, error) {
	x.EnsureNonNil()
	return json.Marshal(dexBatch{
		Committee:        x.Committee,
		ReceiptHash:      x.ReceiptHash,
		Orders:           x.Orders,
		Deposits:         x.Deposits,
		Withdraws:        x.Withdrawals,
		PoolSize:         x.PoolSize,
		CounterPoolSize:  x.CounterPoolSize,
		PoolPoints:       x.PoolPoints,
		TotalPoolPoints:  x.TotalPoolPoints,
		Receipts:         x.Receipts,
		LockedHeight:     x.LockedHeight,
		LivenessFallback: x.LivenessFallback,
	})
}

// UnmarshalJSON() implements the json.Unmarshaller interface for dex batch
func (x *DexBatch) UnmarshalJSON(b []byte) (err error) {
	d := new(dexBatch)
	if err = json.Unmarshal(b, d); err != nil {
		return err
	}
	*x = DexBatch{
		Committee:        d.Committee,
		ReceiptHash:      d.ReceiptHash,
		Orders:           d.Orders,
		Deposits:         d.Deposits,
		Withdrawals:      d.Withdraws,
		CounterPoolSize:  d.CounterPoolSize,
		PoolSize:         d.PoolSize,
		PoolPoints:       d.PoolPoints,
		TotalPoolPoints:  d.TotalPoolPoints,
		Receipts:         d.Receipts,
		LockedHeight:     d.LockedHeight,
		LivenessFallback: d.LivenessFallback,
	}
	x.EnsureNonNil()
	return
}

type poolPoints struct {
	Address HexBytes `json:"address"`
	Points  uint64   `json:"points"`
}

// MarshalJSON() is the json.Marshaller implementation for the PoolPoints object
func (x PoolPoints) MarshalJSON() ([]byte, error) {
	return json.Marshal(poolPoints{
		Address: x.Address,
		Points:  x.Points,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the PoolPoints object
func (x *PoolPoints) UnmarshalJSON(bz []byte) (err error) {
	a := new(poolPoints)
	if err = json.Unmarshal(bz, a); err != nil {
		return err
	}
	*x = PoolPoints{
		Address: a.Address,
		Points:  a.Points,
	}
	return
}

// DexPrice implements a structure to represent the dex price
type DexPrice struct {
	LocalChainId  uint64 `json:"chainId"`
	RemoteChainId uint64 `json:"remoteChainId"`
	LocalPool     uint64 `json:"localPool"`
	RemotePool    uint64 `json:"remotePool"`
	E6ScaledPrice uint64 `json:"e6ScaledPrice"`
}
