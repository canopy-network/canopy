package contract

import (
	"bytes"
	"testing"
)

// keys_test.go covers byte-level parity for the state-database keys this
// plugin constructs. Plugin and Canopy FSM (fsm/key.go) must agree
// byte-for-byte — a one-byte mismatch makes every cross-state read miss.
//
// JoinLenPrefix layout: each segment is encoded as [1-byte length][bytes].
// A singleton key (KeyForFeeParams, KeyForSupply) is just the segment(s)
// with no trailing payload.

func TestKeyForAccount(t *testing.T) {
	addr := []byte{0xAA, 0xBB, 0xCC}
	// [1=accountPrefix][prefix=01][3=addr_len][addr=AABBCC]
	want := []byte{0x01, 0x01, 0x03, 0xAA, 0xBB, 0xCC}
	if got := KeyForAccount(addr); !bytes.Equal(got, want) {
		t.Errorf("KeyForAccount: got %x want %x", got, want)
	}
}

func TestKeyForFeeParams(t *testing.T) {
	// [1=paramsPrefix][prefix=07][3=tag_len][tag="/f/"]
	want := []byte{0x01, 0x07, 0x03, '/', 'f', '/'}
	if got := KeyForFeeParams(); !bytes.Equal(got, want) {
		t.Errorf("KeyForFeeParams: got %x want %x", got, want)
	}
}

func TestKeyForFeePool(t *testing.T) {
	// chainId=2 → [1=poolPrefix][02][8=uint64_len][00..00 02]
	want := []byte{0x01, 0x02, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	if got := KeyForFeePool(2); !bytes.Equal(got, want) {
		t.Errorf("KeyForFeePool(2): got %x want %x", got, want)
	}
	// chainId at the high end exercises the big-endian encoding.
	want2 := []byte{0x01, 0x02, 0x08, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if got := KeyForFeePool(0x0102030405060708); !bytes.Equal(got, want2) {
		t.Errorf("KeyForFeePool(big): got %x want %x", got, want2)
	}
}

// TestKeyForSupply verifies parity with Canopy's fsm.SupplyPrefix() (and
// fsm/key.go's supplyPrefix = []byte{10}). The supply record is a
// singleton at the prefix itself — no trailing segments.
func TestKeyForSupply(t *testing.T) {
	// [1=supplyPrefix_len][prefix=0A]
	want := []byte{0x01, 0x0A}
	if got := KeyForSupply(); !bytes.Equal(got, want) {
		t.Errorf("KeyForSupply: got %x want %x", got, want)
	}
}

// TestKeyForValidator verifies parity with fsm/key.go:121's KeyForValidator
// (validatorPrefix = []byte{3}). The record key is the operator address
// length-prefixed after the prefix byte.
func TestKeyForValidator(t *testing.T) {
	addr := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	// [1=validatorPrefix_len][prefix=03][4=addr_len][addr=DEADBEEF]
	want := []byte{0x01, 0x03, 0x04, 0xDE, 0xAD, 0xBE, 0xEF}
	if got := KeyForValidator(addr); !bytes.Equal(got, want) {
		t.Errorf("KeyForValidator: got %x want %x", got, want)
	}
}
