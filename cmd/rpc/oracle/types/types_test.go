package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWitnessedOrderJSON_UsesCanonicalLastSubmitHeightKey(t *testing.T) {
	order := WitnessedOrder{
		OrderId:          []byte("order1"),
		WitnessedHeight:  10,
		LastSubmitHeight: 99,
	}
	bz, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	jsonStr := string(bz)
	if !strings.Contains(jsonStr, `"lastSubmitHeight":99`) {
		t.Fatalf("expected canonical key in json, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "lastSubmightHeight") {
		t.Fatalf("unexpected legacy typo key in json: %s", jsonStr)
	}
}

func TestWitnessedOrderJSON_BackwardCompatibleUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
		want uint64
	}{
		{
			name: "canonical key",
			json: `{"orderId":"6f7264657231","witnessedHeight":10,"lastSubmitHeight":7}`,
			want: 7,
		},
		{
			name: "legacy typo key",
			json: `{"orderId":"6f7264657231","witnessedHeight":10,"lastSubmightHeight":11}`,
			want: 11,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var order WitnessedOrder
			if err := json.Unmarshal([]byte(tt.json), &order); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if order.LastSubmitHeight != tt.want {
				t.Fatalf("LastSubmitHeight = %d, want %d", order.LastSubmitHeight, tt.want)
			}
		})
	}
}
