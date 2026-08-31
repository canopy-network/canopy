package contract

import (
	"math/big"
	"testing"
)

// maxUint128 is 2^128 - 1, the largest value TryEncodeUint128 must accept.
func maxUint128() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	return max.Sub(max, big.NewInt(1))
}

// twoToThe128 is 2^128 exactly, the smallest value TryEncodeUint128 must
// reject -- the boundary case one bit past maxUint128.
func twoToThe128() *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), 128)
}

func TestTryEncodeUint128(t *testing.T) {
	cases := []struct {
		name   string
		v      *big.Int
		wantOk bool
	}{
		{"zero", big.NewInt(0), true},
		{"one", big.NewInt(1), true},
		{"RAY (1e18)", RAY, true},
		{"max uint128 (2^128 - 1), upper bound", maxUint128(), true},
		{"2^128 exactly rejected, one bit over max", twoToThe128(), false},
		{"negative one rejected", big.NewInt(-1), false},
		{"large negative rejected", new(big.Int).Neg(maxUint128()), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := TryEncodeUint128(tc.v)
			if ok != tc.wantOk {
				t.Errorf("TryEncodeUint128(%s) ok = %v, want %v", tc.v.String(), ok, tc.wantOk)
			}
		})
	}
}

// TestTryEncodeUint128RoundTrip confirms encode-then-decode returns the
// original value exactly, for every in-range case above. This is the
// property every NASM numeric field (vault collateral, debt, SF_index,
// R_nusd) depends on -- a round-trip failure here would silently corrupt
// every value that passes through it.
func TestTryEncodeUint128RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		v    *big.Int
	}{
		{"zero", big.NewInt(0)},
		{"one", big.NewInt(1)},
		{"RAY (1e18)", RAY},
		{"max uint128 (2^128 - 1)", maxUint128()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, ok := TryEncodeUint128(tc.v)
			if !ok {
				t.Fatalf("TryEncodeUint128(%s) unexpectedly returned ok=false", tc.v.String())
			}
			if len(encoded) != 16 {
				t.Fatalf("TryEncodeUint128(%s) returned %d bytes, want 16", tc.v.String(), len(encoded))
			}
			decoded := DecodeUint128(encoded)
			if decoded.Cmp(tc.v) != 0 {
				t.Errorf("round-trip mismatch: encoded %s, decoded %s", tc.v.String(), decoded.String())
			}
		})
	}
}

// TestEncodeUint128RevertsOnOverflow confirms the DeliverTx-context wrapper
// returns a PluginError (for the caller to revert the transaction on),
// never a panic, matching its own doc comment's contract.
func TestEncodeUint128RevertsOnOverflow(t *testing.T) {
	_, pErr := EncodeUint128(twoToThe128())
	if pErr == nil {
		t.Errorf("EncodeUint128(2^128) expected a PluginError, got nil")
	}
}

// TestEncodeUint128SucceedsInRange confirms the wrapper does NOT error for
// a value within range -- the reverting path must not fire on legitimate
// input.
func TestEncodeUint128SucceedsInRange(t *testing.T) {
	encoded, pErr := EncodeUint128(RAY)
	if pErr != nil {
		t.Fatalf("EncodeUint128(RAY) unexpected error: %v", pErr)
	}
	decoded := DecodeUint128(encoded)
	if decoded.Cmp(RAY) != 0 {
		t.Errorf("EncodeUint128(RAY) round-trip mismatch: decoded %s", decoded.String())
	}
}

// TestDecodeUint128ZeroBuffer confirms decoding an all-zero (or empty,
// zero-extended) buffer returns 0 cleanly, not nil and not a panic -- the
// steady-state case every found=false-then-defaulted caller relies on
// (e.g. GetTreasuryNASM's own found=false-is-normal contract in
// state_accessors.go).
func TestDecodeUint128ZeroBuffer(t *testing.T) {
	zero := make([]byte, 16)
	decoded := DecodeUint128(zero)
	if decoded.Sign() != 0 {
		t.Errorf("DecodeUint128(zero buffer) = %s, want 0", decoded.String())
	}
}
