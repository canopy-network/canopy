package lib

import (
	"bytes"
	"encoding/binary"
	"github.com/canopy-network/canopy/lib/crypto"
	"strings"
)

// Hash() creates a hash representative of the dex batch
func (x *DexBatch) Hash() []byte {
	if x.IsEmpty() {
		return []byte(strings.Repeat("F", crypto.HashSize))
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
	return x.ReceiptHash == nil && len(x.Receipts) == 0 && len(x.Orders) == 0 && len(x.Withdraws) == 0 && len(x.Deposits) == 0
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
