package canoliq

import (
	"encoding/hex"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// wire_parity_test.go is the plugin half of a cross-module contract test. The
// canoLiq registry sync (registry.go) decodes raw lib.Validator records that
// the Canopy FSM wrote, using contract.Validator — a partial mirror declaring
// address=1, staked_amount=4, committees=5, unstaking_height=7, output=8 and
// compound=10, and relying on proto3 to skip the rest. Nothing in the build
// catches a divergence: the two live in separate Go modules, and a
// field-number change would surface only as a plugin that observes an empty
// committee and silently distributes no reward — or worse, one that reads a
// zero output and mis-attributes reward ownership.
//
// The fixture below is produced by the FSM's own marshaller and pinned by
// fsm/plugin_validator_scan_test.go::TestPluginValidatorWireFormat. The two
// constants must be updated together.

// validatorWireFixtureHex mirrors fsm/plugin_validator_scan_test.go's constant
// of the same name: a real lib.Validator with address 0x00…01, staked_amount
// 1_000_000_000, committees [404, 1] and compound=true.
const validatorWireFixtureHex = "0a140000000000000000000000000000000000000001208094ebdc032a039403015001"

// validatorWireFixtureWithOutputHex mirrors the FSM constant of the same name:
// the record above plus output 0x50…50. Reward attribution keys on this field,
// so it needs its own fixture rather than riding along on the first.
const validatorWireFixtureWithOutputHex = "0a140000000000000000000000000000000000000001208094ebdc032a0394030142145050505050505050505050505050505050505050" + "5001"

// TestValidatorWireFormatDecodesInPlugin decodes FSM-produced bytes with the
// plugin's own Validator message and asserts every field the reward
// observation depends on survives the crossing.
func TestValidatorWireFormatDecodesInPlugin(t *testing.T) {
	bz, err := hex.DecodeString(validatorWireFixtureHex)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	val := new(contract.Validator)
	if e := contract.Unmarshal(bz, val); e != nil {
		t.Fatalf("unmarshal FSM validator record: %v", e)
	}
	wantAddr := "0000000000000000000000000000000000000001"
	if got := hex.EncodeToString(val.Address); got != wantAddr {
		t.Errorf("address: got %s want %s", got, wantAddr)
	}
	if val.StakedAmount != 1_000_000_000 {
		t.Errorf("stakedAmount: got %d want 1_000_000_000", val.StakedAmount)
	}
	if len(val.Committees) != 2 || val.Committees[0] != 404 || val.Committees[1] != 1 {
		t.Errorf("committees: got %v want [404 1]", val.Committees)
	}
	// The membership predicate the sync actually calls.
	if !validatorOnCommittee(val, 404) {
		t.Errorf("validatorOnCommittee(404) must be true for an FSM record listing 404")
	}
	if validatorOnCommittee(val, 405) {
		t.Errorf("validatorOnCommittee(405) must be false")
	}
	// compound=10 and output=8 are what reward attribution reads. The first
	// fixture has always encoded compound; the mirror just could not see it.
	if !val.Compound {
		t.Errorf("compound: got false want true — field 10 is not crossing the module boundary")
	}
	if len(val.Output) != 0 {
		t.Errorf("output: got %x want empty for the no-output fixture", val.Output)
	}
	if val.UnstakingHeight != 0 {
		t.Errorf("unstakingHeight: got %d want 0", val.UnstakingHeight)
	}
}

// TestValidatorOutputDecodesInPlugin is the ownership half: reward attribution
// is keyed entirely on Validator.output, so a field-8 divergence would read as
// "canoLiq owns nothing" (silent zero yield) or, if it collided with another
// field, as ownership of a bond that is not canoLiq's.
func TestValidatorOutputDecodesInPlugin(t *testing.T) {
	bz, err := hex.DecodeString(validatorWireFixtureWithOutputHex)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	val := new(contract.Validator)
	if e := contract.Unmarshal(bz, val); e != nil {
		t.Fatalf("unmarshal FSM validator record: %v", e)
	}
	wantOutput := "5050505050505050505050505050505050505050"
	if got := hex.EncodeToString(val.Output); got != wantOutput {
		t.Errorf("output: got %s want %s", got, wantOutput)
	}
	// The other fields must be untouched by the added one.
	if val.StakedAmount != 1_000_000_000 {
		t.Errorf("stakedAmount: got %d want 1_000_000_000", val.StakedAmount)
	}
	if !val.Compound {
		t.Errorf("compound: got false want true")
	}
}

// TestCommitteeValidatorsDecodesFSMRecords runs the same FSM-produced bytes
// through the sync's own decode+filter path, the way a range read delivers
// them, so the fixture exercises registry.go rather than just the proto.
func TestCommitteeValidatorsDecodesFSMRecords(t *testing.T) {
	bz, err := hex.DecodeString(validatorWireFixtureHex)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	entries := []*contract.PluginStateEntry{
		{Key: contract.KeyForValidator(addr20(0x01)), Value: bz},
	}
	members, e := committeeValidators(entries, 404)
	if e != nil {
		t.Fatalf("committeeValidators: %v", e)
	}
	if len(members) != 1 {
		t.Fatalf("committee member count: got %d want 1", len(members))
	}
	if members[0].StakedAmount != 1_000_000_000 {
		t.Errorf("observed stake: got %d want 1_000_000_000", members[0].StakedAmount)
	}
	// Same record, different committee: filtered out.
	off, e := committeeValidators(entries, 405)
	if e != nil {
		t.Fatalf("committeeValidators(405): %v", e)
	}
	if len(off) != 0 {
		t.Fatalf("off-committee record must be filtered: got %d", len(off))
	}
}
