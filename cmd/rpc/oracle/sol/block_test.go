package sol

import (
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
)

func TestBlock_Accessors(t *testing.T) {
	txs := []*Transaction{newTransaction("sig1", "payer", nil)}
	b := newBlock(1234, "blockhashX", "parenthashY", txs)
	if b.Number() != 1234 {
		t.Fatalf("expected slot 1234, got %d", b.Number())
	}
	if b.Hash() != "blockhashX" || b.ParentHash() != "parenthashY" {
		t.Fatalf("hash/parent mismatch: %s %s", b.Hash(), b.ParentHash())
	}
	got := b.Transactions()
	if len(got) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(got))
	}
	var _ types.BlockI = b
	var _ types.TransactionI = got[0]
}
