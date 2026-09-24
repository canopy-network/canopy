package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/contract"
)

func mustAddr(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 20 {
		panic("bad test address " + h)
	}
	return b
}

func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// ProposalParamChange is a full-set replacement: dispatchPassed calls
// SaveParams(p.Params) wholesale, and ValidateParams does not inspect
// tvlCapBps, the graduation thresholds, or insuranceTargetBps, and explicitly
// accepts an empty governance tier list. So any CanoliqParams field this tool
// fails to carry is silently written as its Go zero when the proposal passes —
// a state corruption that no validation rejects.
//
// These tests exist to make that class of bug impossible to reintroduce.

// protoFields returns the real (non-internal) field names of a protobuf struct.
func protoFields(t *testing.T, msg any) []string {
	t.Helper()
	rt := reflect.TypeOf(msg).Elem()
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if strings.HasPrefix(name, "state") || name == "sizeCache" || name == "unknownFields" {
			continue
		}
		out = append(out, name)
	}
	return out
}

// fullyPopulatedParams builds a CanoliqParams with EVERY field set to a
// distinct non-zero value, using reflection so it cannot drift as fields are
// added. A hand-maintained literal is what let the original bug through: the
// nine uncovered fields were zero on both sides of proto.Equal, so the
// round-trip check could not fail for the bug it advertised.
func fullyPopulatedParams(t *testing.T) *contract.CanoliqParams {
	t.Helper()
	p := &contract.CanoliqParams{}
	rv := reflect.ValueOf(p).Elem()
	rt := rv.Type()
	n := uint64(0)
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if strings.HasPrefix(name, "state") || name == "sizeCache" || name == "unknownFields" {
			continue
		}
		f := rv.Field(i)
		if f.Kind() == reflect.Uint64 {
			n++
			f.SetUint(n)
		}
	}
	// Non-scalar fields reflection cannot fill meaningfully.
	p.MultisigSigners = [][]byte{mustAddr("1b6454361f65ac5cc13c6b775692ba3d64cbcb84")}
	p.MultisigThreshold = 1 // must be within [1, len(signers)]
	p.StakeOutputAddresses = [][]byte{mustAddr("c0ffee0000000000000000000000000000000001")}
	p.MaxRewardBpsPerBlock = 100 // reflection fills uint64s, but keep it valid (≤ 10_000)
	p.Governance = []*contract.GovernanceTier{{
		Action:    contract.ActionType_ACTION_FEE_CHANGE,
		QuorumBps: 41, ApprovalBps: 42, TimelockBlocks: 43, VotingPeriodBlocks: 44,
	}}
	p.RestakingPolicy = []*contract.RestakingPolicyEntry{{
		CommitteeId: 45, TargetWeightBps: 10_000, MinStakeUcnpy: 46, MaxStakeUcnpy: 47,
	}}

	// Guard the guard: assert nothing was left at zero, or this test silently
	// stops covering whatever was missed.
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if strings.HasPrefix(name, "state") || name == "sizeCache" || name == "unknownFields" {
			continue
		}
		if rv.Field(i).IsZero() {
			t.Fatalf("field %s left at zero — fullyPopulatedParams must set every field", name)
		}
	}
	return p
}

// A field present in paramsJSON but never assigned in toContract would pass a
// name-only check. Round-tripping a fully-populated message catches that too,
// because the dropped field comes back zero against a non-zero source.
func TestParamsRoundTripWithEveryFieldPopulated(t *testing.T) {
	src := fullyPopulatedParams(t)
	bz, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw paramsJSON
	if err := json.Unmarshal(bz, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dst, err := raw.toContract()
	if err != nil {
		t.Fatalf("toContract: %v", err)
	}
	if !proto.Equal(src, dst) {
		t.Fatalf("a field is declared in paramsJSON but not threaded through toContract:\n src=%+v\n dst=%+v", src, dst)
	}
}

// If someone adds a field to CanoliqParams and forgets paramsJSON, this fails
// — which is the whole point. It is cheaper than discovering it as a wiped
// governance matrix on a live chain.
func TestParamsJSONCoversEveryParamField(t *testing.T) {
	want := protoFields(t, &contract.CanoliqParams{})
	have := map[string]bool{}
	rt := reflect.TypeOf(paramsJSON{})
	for i := 0; i < rt.NumField(); i++ {
		have[rt.Field(i).Name] = true
	}
	var missing []string
	for _, f := range want {
		if !have[f] {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("paramsJSON is missing %d CanoliqParams field(s): %v\n"+
			"a param-change would write these as zero — add them to paramsJSON AND toContract",
			len(missing), missing)
	}
}

// The workflow operators actually use is: GET /v1/params > params.json, edit
// one field, submit. That round-trip must be lossless, or editing the TVL cap
// silently resets everything the decoder drops.
func TestParamsRoundTripFromQueryOutputIsLossless(t *testing.T) {
	orig := canoliq.DefaultParams()
	// DefaultParams leaves MultisigSigners nil; a real chain has them, and they
	// are the field most likely to break a round-trip because /v1/params emits
	// [][]byte as base64 while the genesis convention is hex.
	orig.MultisigSigners = [][]byte{
		mustAddr("1b6454361f65ac5cc13c6b775692ba3d64cbcb84"),
		mustAddr("2ea35a0ef4ef34c58c176ae0ad6b6a4fbf33346f"),
		mustAddr("b749e623b0b3dc8a6fce9f89c539d3351e92a6c5"),
	}
	// Same base64-vs-hex hazard, and losing this one silently switches reward
	// attribution off rather than merely mangling a signer list.
	orig.StakeOutputAddresses = [][]byte{
		mustAddr("c0ffee0000000000000000000000000000000001"),
	}
	orig.MultisigThreshold = 2

	// Exactly what GET /v1/params writes: encoding/json over the proto struct.
	dumped, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "params.json")
	if err := os.WriteFile(path, dumped, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := loadParamsFromFile(path)
	if err != nil {
		t.Fatalf("loadParamsFromFile: %v", err)
	}
	if !proto.Equal(orig, got) {
		t.Fatalf("round-trip lost data:\n original: %+v\n  reloaded: %+v", orig, got)
	}
}

// The specific edit the devnet needs: dump, set tvlCapBps to 0, submit. Only
// that field may differ afterwards.
func TestParamsRoundTripUncapEditChangesOnlyTheCap(t *testing.T) {
	orig := canoliq.DefaultParams()
	dumped, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(dumped, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw["tvlCapBps"] = 0
	edited, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "params.json")
	if err := os.WriteFile(path, edited, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := loadParamsFromFile(path)
	if err != nil {
		t.Fatalf("loadParamsFromFile: %v", err)
	}
	if got.TvlCapBps != 0 {
		t.Fatalf("TvlCapBps: got %d want 0", got.TvlCapBps)
	}
	// Everything else must survive. Governance tiers are the sharpest check:
	// they were wiped entirely before this fix, silently downgrading every
	// future proposal to the scalar quorum/threshold knobs.
	if len(got.Governance) != len(orig.Governance) {
		t.Fatalf("governance tiers: got %d want %d (wiping these changes every future vote)",
			len(got.Governance), len(orig.Governance))
	}
	if got.GraduationMinValidators != orig.GraduationMinValidators {
		t.Fatalf("GraduationMinValidators: got %d want %d", got.GraduationMinValidators, orig.GraduationMinValidators)
	}
	if got.InsuranceTargetBps != orig.InsuranceTargetBps {
		t.Fatalf("InsuranceTargetBps: got %d want %d", got.InsuranceTargetBps, orig.InsuranceTargetBps)
	}
	// Restore the one edited field and the whole message must match.
	got.TvlCapBps = orig.TvlCapBps
	if !proto.Equal(orig, got) {
		t.Fatalf("edit touched more than tvlCapBps:\n original: %+v\n  reloaded: %+v", orig, got)
	}
}

// Signers must decode from either encoding, since the genesis convention is hex
// and /v1/params emits base64.
func TestDecodeSignerAcceptsHexAndBase64(t *testing.T) {
	addr := mustAddr("1b6454361f65ac5cc13c6b775692ba3d64cbcb84")
	for name, in := range map[string]string{
		"hex":        "1b6454361f65ac5cc13c6b775692ba3d64cbcb84",
		"hex-0x":     "0x1b6454361f65ac5cc13c6b775692ba3d64cbcb84",
		"base64":     "G2RUNh9lrFzBPGt3VpK6PWTLy4Q=",
		"base64-alt": base64Std(addr),
	} {
		got, err := decodeSigner(in)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(got) != string(addr) {
			t.Fatalf("%s: got %x want %x", name, got, addr)
		}
	}
	if _, err := decodeSigner("not-an-address"); err == nil {
		t.Fatalf("expected an error for garbage input")
	}
	// Wrong length must be rejected in both encodings.
	if _, err := decodeSigner("deadbeef"); err == nil {
		t.Fatalf("expected an error for a short hex address")
	}
}
