package lib

import (
	"encoding/json"
	"testing"
)

func TestDexBatch_Hash(t *testing.T) {
	batch := &DexBatch{Committee: 1}
	hash := batch.Hash()

	if len(hash) == 0 {
		t.Error("expected non-empty hash")
	}
}

func TestDexBatch_Hash_Nil(t *testing.T) {
	var batch *DexBatch
	hash := batch.Hash()

	if len(hash) == 0 {
		t.Error("expected non-empty hash for nil batch")
	}
}

func TestDexBatch_Copy(t *testing.T) {
	original := &DexBatch{
		Committee:    1,
		PoolSize:     1000,
		LockedHeight: 100,
	}

	copy := original.Copy()

	if copy.Committee != original.Committee {
		t.Errorf("expected committee %d, got %d", original.Committee, copy.Committee)
	}
	if copy.PoolSize != original.PoolSize {
		t.Errorf("expected poolSize %d, got %d", original.PoolSize, copy.PoolSize)
	}
}

func TestDexBatch_IsEmpty(t *testing.T) {
	batch := &DexBatch{}
	if !batch.IsEmpty() {
		t.Error("expected empty batch to return true")
	}

	batch.Orders = []*DexLimitOrder{{AmountForSale: 100}}
	if batch.IsEmpty() {
		t.Error("expected non-empty batch to return false")
	}
}

func TestDexBatch_EnsureNonNil(t *testing.T) {
	batch := &DexBatch{}
	batch.EnsureNonNil()

	if batch.Orders == nil {
		t.Error("expected Orders to be initialized")
	}
	if batch.Deposits == nil {
		t.Error("expected Deposits to be initialized")
	}
	if batch.Withdraws == nil {
		t.Error("expected Withdraws to be initialized")
	}
	if batch.Receipts == nil {
		t.Error("expected Receipts to be initialized")
	}
}

func TestDexLimitOrder_Copy(t *testing.T) {
	original := &DexLimitOrder{
		AmountForSale:   1000,
		RequestedAmount: 2000,
		Address:         []byte("test-address"),
	}

	copy := original.Copy()

	if copy.AmountForSale != original.AmountForSale {
		t.Errorf("expected amountForSale %d, got %d", original.AmountForSale, copy.AmountForSale)
	}
	if copy.RequestedAmount != original.RequestedAmount {
		t.Errorf("expected requestedAmount %d, got %d", original.RequestedAmount, copy.RequestedAmount)
	}
}

func TestDexLimitOrderWithKey_HashKey(t *testing.T) {
	order := &DexLimitOrderWithKey{
		DexLimitOrder: &DexLimitOrder{AmountForSale: 1000},
	}

	key := order.HashKey(0, []byte("blockhash"))

	if key == "" {
		t.Error("expected non-empty key")
	}
	if order.Key != key {
		t.Error("expected key to be set on order")
	}
}

func TestDexLimitOrder_MarshalJSON(t *testing.T) {
	order := DexLimitOrder{
		AmountForSale:   1000,
		RequestedAmount: 2000,
		Address:         []byte("test"),
	}

	data, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestDexLimitOrder_UnmarshalJSON(t *testing.T) {
	jsonData := `{"amountForSale":1000,"requestedAmount":2000,"address":"74657374"}`

	var order DexLimitOrder
	err := json.Unmarshal([]byte(jsonData), &order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.AmountForSale != 1000 {
		t.Errorf("expected amountForSale 1000, got %d", order.AmountForSale)
	}
	if order.RequestedAmount != 2000 {
		t.Errorf("expected requestedAmount 2000, got %d", order.RequestedAmount)
	}
}

func TestDexLiquidityDeposit_MarshalJSON(t *testing.T) {
	deposit := DexLiquidityDeposit{
		Amount:  1000,
		Address: []byte("test"),
	}

	data, err := json.Marshal(deposit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestDexLiquidityWithdraw_MarshalJSON(t *testing.T) {
	withdraw := DexLiquidityWithdraw{
		Percent: 50,
		Address: []byte("test"),
	}

	data, err := json.Marshal(withdraw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestDexBatch_MarshalJSON(t *testing.T) {
	batch := DexBatch{
		Committee: 1,
		PoolSize:  1000,
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestPoolPoints_MarshalJSON(t *testing.T) {
	points := PoolPoints{
		Address: []byte("test"),
		Points:  100,
	}

	data, err := json.Marshal(points)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}