package lib

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/canopy-network/canopy/lib/crypto"
	"strings"
)

var (
	EmptyReceiptsHash = []byte(strings.Repeat("F", crypto.HashSize))
)

// Hash() creates a hash representative of the dex batch
func (x *DexBatch) Hash() []byte {
	if x.IsEmpty() {
		return bytes.Clone(EmptyReceiptsHash)
	}
	bz, _ := Marshal(x)
	return crypto.Hash(bz)
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
	return len(x.ReceiptHash) == 0 && len(x.Receipts) == 0 && len(x.Orders) == 0 && len(x.Withdraws) == 0 && len(x.Deposits) == 0
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
	if x.Withdraws == nil {
		x.Withdraws = []*DexLiquidityWithdraw{}
	}
	if x.Receipts == nil {
		x.Receipts = []bool{}
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
}

// MarshalJSON() implements the json.Marshal interface for DexLimitOrder
func (x DexLimitOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLimitOrder{
		AmountForSale:   x.AmountForSale,
		RequestedAmount: x.RequestedAmount,
		Address:         x.Address,
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
	}
	return
}

type dexLiquidityDeposit struct {
	Amount  uint64   `json:"amount"`
	Address HexBytes `json:"address"`
}

// MarshalJSON() implements the json.Marshal interface for DexLiquidityDeposit
func (x DexLiquidityDeposit) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLiquidityDeposit{
		Amount:  x.Amount,
		Address: x.Address,
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
	}
	return
}

type dexLiquidityWithdraw struct {
	Percent uint64   `json:"percent"`
	Address HexBytes `json:"address"`
}

// MarshalJSON() implements the json.Marshal interface for dexLiquidityWithdraw
func (x DexLiquidityWithdraw) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexLiquidityWithdraw{
		Percent: x.Percent,
		Address: x.Address,
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
	}
	return
}

type dexBatch struct {
	Committee       uint64                  `json:"committee"`
	ReceiptHash     HexBytes                `json:"receiptHash"`
	Orders          []*DexLimitOrder        `json:"orders"`
	Deposits        []*DexLiquidityDeposit  `json:"deposits,omitempty"`
	Withdraws       []*DexLiquidityWithdraw `json:"withdraws,omitempty"`
	PoolSize        uint64                  `json:"poolSize"`
	PoolPoints      []*PoolPoints           `json:"poolPoints"`
	TotalPoolPoints uint64                  `json:"totalPoolPoints"`
	Receipts        []bool                  `json:"receipts,omitempty"`
	LockedHeight    uint64                  `json:"locked_height,omitempty"`
}

// MarshalJSON() implements the json.Marshal interface for dex batch
func (x DexBatch) MarshalJSON() ([]byte, error) {
	return json.Marshal(dexBatch{
		Committee:       x.Committee,
		ReceiptHash:     x.ReceiptHash,
		Orders:          x.Orders,
		Deposits:        x.Deposits,
		Withdraws:       x.Withdraws,
		PoolSize:        x.PoolSize,
		PoolPoints:      x.PoolPoints,
		TotalPoolPoints: x.TotalPoolPoints,
		Receipts:        x.Receipts,
		LockedHeight:    x.LockedHeight,
	})
}

// UnmarshalJSON() implements the json.Unmarshaller interface for dex batch
func (x *DexBatch) UnmarshalJSON(b []byte) (err error) {
	d := new(dexBatch)
	if err = json.Unmarshal(b, d); err != nil {
		return err
	}
	*x = DexBatch{
		Committee:       d.Committee,
		ReceiptHash:     d.ReceiptHash,
		Orders:          d.Orders,
		Deposits:        d.Deposits,
		Withdraws:       d.Withdraws,
		PoolSize:        d.PoolSize,
		PoolPoints:      d.PoolPoints,
		TotalPoolPoints: d.TotalPoolPoints,
		Receipts:        d.Receipts,
		LockedHeight:    d.LockedHeight,
	}
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
