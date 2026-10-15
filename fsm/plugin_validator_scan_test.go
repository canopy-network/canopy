package fsm

import (
	"encoding/hex"
	"testing"

	"github.com/canopy-network/canopy/lib"
)

// plugin_validator_scan_test.go is the FSM half of a cross-module contract
// test. Plugins discover committee membership by range-scanning
// ValidatorPrefix() and decoding each record with their own partial mirror of
// this package's Validator (plugin/go/proto/validator.proto). The two live in
// separate Go modules, so nothing in either build catches a divergence: a
// field-number change here would surface only as a plugin that observes an
// empty committee, or a zero output address, and silently mis-attributes
// reward.
//
// This file pins the wire bytes this side produces. Its counterpart is
// plugin/go/canoliq/wire_parity_test.go, which decodes the same constants with
// the plugin's mirror. The constants must be updated together.

// validatorWireFixtureHex is a real lib.Validator with address 0x00…01,
// staked_amount 1_000_000_000, committees [404, 1] and compound=true.
const validatorWireFixtureHex = "0a140000000000000000000000000000000000000001208094ebdc032a039403015001"

// validatorWireFixtureWithOutputHex is the same record plus output 0x50…50,
// the field canoLiq keys reward ownership on.
const validatorWireFixtureWithOutputHex = "0a140000000000000000000000000000000000000001208094ebdc032a0394030142145050505050505050505050505050505050505050" + "5001"

// TestPluginValidatorWireFormat pins the bytes a lib.Validator marshals to, so
// that a field renumbering on this side fails here rather than silently in a
// plugin.
func TestPluginValidatorWireFormat(t *testing.T) {
	addr, _ := hex.DecodeString("0000000000000000000000000000000000000001")
	output, _ := hex.DecodeString("5050505050505050505050505050505050505050")

	tests := []struct {
		name string
		val  *Validator
		want string
	}{
		{
			name: "fields the plugin mirror declares",
			val: &Validator{
				Address:      addr,
				StakedAmount: 1_000_000_000,
				Committees:   []uint64{404, 1},
				Compound:     true,
			},
			want: validatorWireFixtureHex,
		},
		{
			name: "with the output address reward ownership keys on",
			val: &Validator{
				Address:      addr,
				StakedAmount: 1_000_000_000,
				Committees:   []uint64{404, 1},
				Output:       output,
				Compound:     true,
			},
			want: validatorWireFixtureWithOutputHex,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bz, err := lib.Marshal(tt.val)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if got := hex.EncodeToString(bz); got != tt.want {
				t.Errorf("wire format changed.\n got: %s\nwant: %s\n"+
					"If this is intentional, update the matching constant in "+
					"plugin/go/canoliq/wire_parity_test.go in the same change.", got, tt.want)
			}
		})
	}
}

// TestPluginValidatorFieldNumbers pins the individual field numbers the plugin
// mirror hard-codes. Marshalling one field at a time makes a renumbering point
// at the field that moved instead of at an opaque byte diff.
func TestPluginValidatorFieldNumbers(t *testing.T) {
	addr, _ := hex.DecodeString("0000000000000000000000000000000000000001")
	tests := []struct {
		field string
		val   *Validator
		want  string
	}{
		{"address = 1", &Validator{Address: addr}, "0a140000000000000000000000000000000000000001"},
		{"staked_amount = 4", &Validator{StakedAmount: 1}, "2001"},
		{"committees = 5", &Validator{Committees: []uint64{404}}, "2a029403"},
		{"unstaking_height = 7", &Validator{UnstakingHeight: 1}, "3801"},
		{"output = 8", &Validator{Output: addr}, "42140000000000000000000000000000000000000001"},
		{"compound = 10", &Validator{Compound: true}, "5001"},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			bz, err := lib.Marshal(tt.val)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if got := hex.EncodeToString(bz); got != tt.want {
				t.Errorf("%s moved: got %s want %s", tt.field, got, tt.want)
			}
		})
	}
}
