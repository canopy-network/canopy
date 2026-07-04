package sol

import "testing"

// makeMintData builds a byte slice with `decimals` at offset 44, mimicking an SPL Mint account.
func makeMintData(decimals byte) []byte {
	data := make([]byte, 82) // SPL Mint account is 82 bytes
	data[mintDecimalsOffset] = decimals
	return data
}

func TestDecodeMintDecimals(t *testing.T) {
	data := makeMintData(6)
	got, err := decodeMintDecimals(data)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 6 {
		t.Fatalf("expected 6 decimals, got %d", got)
	}

	// too-small data must error, not panic
	if _, err := decodeMintDecimals(make([]byte, 10)); err == nil {
		t.Fatal("expected error for undersized mint data")
	}
}

func TestMintCache_HitMiss(t *testing.T) {
	c := newMintCache(2)
	mintA := "MintAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	mintB := "MintBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

	if _, ok := c.get(mintA); ok {
		t.Fatal("expected miss on empty cache")
	}
	c.put(mintA, 6)
	if d, ok := c.get(mintA); !ok || d != 6 {
		t.Fatalf("expected hit d=6, got d=%d ok=%v", d, ok)
	}
	// exceed capacity: mintA should evict when a third distinct entry is added
	c.put(mintB, 9)
	c.put("MintCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC", 0)
	if _, ok := c.get(mintA); ok {
		t.Fatal("expected mintA evicted after exceeding capacity")
	}
}
