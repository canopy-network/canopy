package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/contract"
)

// proposal_create_test.go covers the per-payload helpers and the Any
// type URL correctness so the FSM-side resolver
// (governance.go::unwrapPayload → contract.FromAny) can dispatch each
// payload type without hand-running the full submit path.

func TestParseBuybackMode(t *testing.T) {
	cases := []struct {
		in   string
		want contract.BuybackMode
	}{
		{"burn", contract.BuybackMode_BUYBACK_BURN},
		{"BURN", contract.BuybackMode_BUYBACK_BURN},
		{"distribute", contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS},
		{"distribute-stakers", contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS},
		{"distribute_stakers", contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS},
	}
	for _, tc := range cases {
		got, err := parseBuybackMode(tc.in)
		if err != nil {
			t.Errorf("parseBuybackMode(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseBuybackMode(%q): got %v want %v", tc.in, got, tc.want)
		}
	}
	if _, err := parseBuybackMode("not-a-mode"); err == nil {
		t.Errorf("parseBuybackMode should reject unknown mode")
	}
}

func TestParseSpendDenomination(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want contract.SpendDenomination
	}{
		{"cnpy", contract.SpendDenomination_SPEND_CNPY},
		{"CNPY", contract.SpendDenomination_SPEND_CNPY},
		{"cplq", contract.SpendDenomination_SPEND_CPLQ},
	} {
		got, err := parseSpendDenomination(tc.in)
		if err != nil {
			t.Errorf("parseSpendDenomination(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("parseSpendDenomination(%q): got %v want %v", tc.in, got, tc.want)
		}
	}
	if _, err := parseSpendDenomination("usdc"); err == nil {
		t.Errorf("parseSpendDenomination should reject unknown denom")
	}
}

func TestParseDescriptionFlag(t *testing.T) {
	for _, tc := range []struct {
		name           string
		in             []string
		wantPositional []string
		wantDesc       string
	}{
		{
			name:           "trailing flag",
			in:             []string{"alice", "1", "2", "burn", "--description", "lower fee"},
			wantPositional: []string{"alice", "1", "2", "burn"},
			wantDesc:       "lower fee",
		},
		{
			name:           "embedded equals",
			in:             []string{"alice", "1", "--description=lower fee"},
			wantPositional: []string{"alice", "1"},
			wantDesc:       "lower fee",
		},
		{
			name:           "short flag",
			in:             []string{"alice", "-d", "x"},
			wantPositional: []string{"alice"},
			wantDesc:       "x",
		},
		{
			name:           "no flag",
			in:             []string{"alice", "1", "2"},
			wantPositional: []string{"alice", "1", "2"},
			wantDesc:       "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, desc := parseDescriptionFlag(tc.in)
			if !reflect.DeepEqual(pos, tc.wantPositional) {
				t.Errorf("positional: got %v want %v", pos, tc.wantPositional)
			}
			if desc != tc.wantDesc {
				t.Errorf("description: got %q want %q", desc, tc.wantDesc)
			}
		})
	}
}

func TestParamsJSONToContract(t *testing.T) {
	// Mirrors the genesis params block — flat JSON with hex multisig
	// signers — and asserts each scalar field flows through unchanged.
	raw := paramsJSON{
		FeeBps:              1500,
		UserRebateBps:       4000,
		TreasuryBps:         3000,
		ValidatorBps:        1500,
		BuybackBps:          1500,
		DepositFee:          10000,
		InsuranceBps:        2000,
		TreasuryThreshold:   500_000_000,
		MultisigSigners:     []string{"0x" + strings.Repeat("aa", 20), strings.Repeat("bb", 20)},
		MultisigThreshold:   2,
		VotingPeriodBlocks:  100_800,
		QuorumBps:           3300,
		PassThresholdBps:    5001,
		TimelockBlocks:      28_800,
		CplqUnstakingBlocks: 100_800,
		MinStakeToPropose:   1_000_000,
	}
	p, err := raw.toContract()
	if err != nil {
		t.Fatalf("toContract: %v", err)
	}
	if p.FeeBps != 1500 || p.InsuranceBps != 2000 || p.MultisigThreshold != 2 {
		t.Fatalf("scalar fields: %+v", p)
	}
	if len(p.MultisigSigners) != 2 ||
		!bytes.Equal(p.MultisigSigners[0], bytes.Repeat([]byte{0xaa}, 20)) ||
		!bytes.Equal(p.MultisigSigners[1], bytes.Repeat([]byte{0xbb}, 20)) {
		t.Fatalf("multisig signers: %x", p.MultisigSigners)
	}
}

func TestParamsJSONRejectsBadHex(t *testing.T) {
	raw := paramsJSON{MultisigSigners: []string{"not-hex"}}
	if _, err := raw.toContract(); err == nil {
		t.Errorf("expected hex decode error")
	}
	raw = paramsJSON{MultisigSigners: []string{strings.Repeat("aa", 19)}} // 19 bytes
	if _, err := raw.toContract(); err == nil {
		t.Errorf("expected length error")
	}
}

// writeParamsFile dumps params exactly as `GET /v1/params` would, optionally
// applying edits, and returns the path. This is the workflow operators follow.
func writeParamsFile(t *testing.T, params *contract.CanoliqParams, edit func(map[string]any)) string {
	t.Helper()
	bz, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(bz, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if edit != nil {
		edit(raw)
	}
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "params.json")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadParamsFromFile(t *testing.T) {
	src := canoliq.DefaultParams()
	src.FeeBps = 800
	p, err := loadParamsFromFile(writeParamsFile(t, src, nil))
	if err != nil {
		t.Fatalf("loadParamsFromFile: %v", err)
	}
	if p.FeeBps != 800 {
		t.Fatalf("FeeBps: got %d want 800", p.FeeBps)
	}
}

// A param-change is a full-set replacement, so an incomplete file must be
// rejected rather than zero-filled. This test previously asserted the opposite:
// it fed a file missing 12 keys and expected it to load, which is exactly how
// an operator would silently uncap the TVL ceiling or zero the multisig
// threshold.
func TestLoadParamsFromFileRejectsIncompleteFile(t *testing.T) {
	path := writeParamsFile(t, canoliq.DefaultParams(), func(raw map[string]any) {
		delete(raw, "tvlCapBps")
		delete(raw, "governance")
	})
	_, err := loadParamsFromFile(path)
	if err == nil {
		t.Fatalf("expected an incomplete params file to be rejected")
	}
	for _, want := range []string{"tvlCapBps", "governance", "missing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error should name %q, got: %v", want, err)
		}
	}
}

// Omitting multisigSigners AND multisigThreshold together is the sharpest case:
// ValidateParams only sanity-checks the threshold when signers are present, so
// a zero threshold would sail through and treasury.go's
// `approvals < threshold` gate would pass with zero approvals.
func TestLoadParamsFromFileRejectsMissingMultisigPair(t *testing.T) {
	path := writeParamsFile(t, canoliq.DefaultParams(), func(raw map[string]any) {
		delete(raw, "multisigSigners")
		delete(raw, "multisigThreshold")
	})
	if _, err := loadParamsFromFile(path); err == nil {
		t.Fatalf("expected rejection: a zero multisig threshold disables the treasury gate")
	}
}

// A misspelled key would otherwise be dropped silently, leaving the real field
// at zero — the same silent-zeroing failure by a different route.
func TestLoadParamsFromFileRejectsUnknownKey(t *testing.T) {
	path := writeParamsFile(t, canoliq.DefaultParams(), func(raw map[string]any) {
		delete(raw, "tvlCapBps")
		raw["tvlCap"] = 0 // typo
	})
	if _, err := loadParamsFromFile(path); err == nil {
		t.Fatalf("expected a misspelled key to be rejected")
	}
}

// Pre-validate locally: the plugin only runs ValidateParams at dispatchPassed,
// so an invalid payload would otherwise burn a full voting period before
// failing to apply.
func TestLoadParamsFromFileRejectsInvalidParams(t *testing.T) {
	path := writeParamsFile(t, canoliq.DefaultParams(), func(raw map[string]any) {
		raw["feeBps"] = 9000 // outside the [500, 2000] band
	})
	err := loadParamsFromFile2(t, path)
	if err == nil {
		t.Fatalf("expected ValidateParams to reject feeBps=9000 before submission")
	}
	if !strings.Contains(err.Error(), "ValidateParams") {
		t.Fatalf("error should mention ValidateParams, got: %v", err)
	}
}

func loadParamsFromFile2(t *testing.T, path string) error {
	t.Helper()
	_, err := loadParamsFromFile(path)
	return err
}

// TestProposalPayloadAnyRoundTrip asserts each payload kind survives
// anypb.New + UnmarshalNew with the right concrete type. This is the
// guarantee the FSM relies on — wrong TypeUrl → unknown payload error.
func TestProposalPayloadAnyRoundTrip(t *testing.T) {
	cases := []struct {
		name        string
		build       func() proto.Message
		wantTypeURL string
	}{
		{
			name: "param-change",
			build: func() proto.Message {
				return &contract.ProposalParamChange{
					Params: &contract.CanoliqParams{FeeBps: 1234},
				}
			},
			wantTypeURL: "type.googleapis.com/types.ProposalParamChange",
		},
		{
			name: "buyback",
			build: func() proto.Message {
				return &contract.ProposalBuyback{
					CnpyAmount:            500_000,
					PriceMicroCnpyPerCplq: 1_500_000,
					Mode:                  contract.BuybackMode_BUYBACK_BURN,
				}
			},
			wantTypeURL: "type.googleapis.com/types.ProposalBuyback",
		},
		{
			name: "treasury-spend",
			build: func() proto.Message {
				return &contract.ProposalTreasurySpend{
					Recipient:    bytes.Repeat([]byte{0xcc}, 20),
					Amount:       9_999,
					Denomination: contract.SpendDenomination_SPEND_CNPY,
				}
			},
			wantTypeURL: "type.googleapis.com/types.ProposalTreasurySpend",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := anypb.New(tc.build())
			if err != nil {
				t.Fatalf("anypb.New: %v", err)
			}
			if payload.TypeUrl != tc.wantTypeURL {
				t.Fatalf("TypeUrl: got %q want %q", payload.TypeUrl, tc.wantTypeURL)
			}
			// UnmarshalNew gives us back the concrete proto type — same
			// path the FSM resolver takes through contract.FromAny.
			msg, err := payload.UnmarshalNew()
			if err != nil {
				t.Fatalf("UnmarshalNew: %v", err)
			}
			if !proto.Equal(msg, tc.build()) {
				t.Fatalf("round-trip diverged: got %+v", msg)
			}
		})
	}
}

func TestProposalCreateOuterMessageMarshals(t *testing.T) {
	// The outer MessageCPLQProposalCreate is marshaled by SubmitPluginTx;
	// confirm it preserves payload + description across a marshal round
	// trip so we don't accidentally drop fields between proto versions.
	payload, err := anypb.New(&contract.ProposalBuyback{
		CnpyAmount: 1, PriceMicroCnpyPerCplq: 1, Mode: contract.BuybackMode_BUYBACK_BURN,
	})
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	original := &contract.MessageCPLQProposalCreate{
		FromAddress: bytes.Repeat([]byte{0x01}, 20),
		Payload:     payload,
		Description: "test description",
	}
	bz, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := new(contract.MessageCPLQProposalCreate)
	if err := proto.Unmarshal(bz, got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Description != "test description" {
		t.Fatalf("description: got %q", got.Description)
	}
	if got.Payload == nil || got.Payload.TypeUrl != payload.TypeUrl {
		t.Fatalf("payload TypeUrl: got %+v", got.Payload)
	}
}

// guard against the JSON shape drifting from the proto's @gotags. If a
// new param field is added on the proto side, this test will fail
// reminding the developer to update paramsJSON.
func TestParamsJSONShapeMatchesProto(t *testing.T) {
	// Encode a fully-populated CanoliqParams via std json (it honors
	// @gotags) and decode into paramsJSON. All fields must round-trip.
	//
	// Every field below must carry a NON-ZERO value. This test previously
	// populated only the 22 fields paramsJSON already covered, which made it
	// unable to fail for the bug it advertises: the nine it omitted were zero
	// on both sides, so proto.Equal compared zero to zero and passed while
	// param-changes silently wiped the governance matrix and the graduation
	// thresholds. TestParamsJSONCoversEveryParamField (reflection over the
	// proto type) is the structural guard; this one is the value check, and it
	// only works if nothing here is left at its zero value.
	src := &contract.CanoliqParams{
		FeeBps: 1, UserRebateBps: 2, TreasuryBps: 3, ValidatorBps: 4, BuybackBps: 5,
		DepositFee: 6, RedeemFee: 7, ClaimFee: 8, CplqTransferFee: 9,
		InsuranceBps: 10, TreasuryThreshold: 11, MultisigThreshold: 12,
		VotingPeriodBlocks: 13, QuorumBps: 14, PassThresholdBps: 15,
		TimelockBlocks: 16, CplqUnstakingBlocks: 17, ProposalFee: 18,
		VoteFee: 19, StakeFee: 20, MultisigApproveFee: 21, MinStakeToPropose: 22,
		TvlCapBps: 23, InsuranceTargetBps: 24,
		GraduationMinTvlUcnpy: 25, GraduationMinValidators: 26,
		GraduationMinTurnoutBps: 27, GraduationMinDailyTx: 28,
		GraduationMinRunwayMonths: 29,
		Governance: []*contract.GovernanceTier{{
			Action:    contract.ActionType_ACTION_FEE_CHANGE,
			QuorumBps: 30, ApprovalBps: 31,
			TimelockBlocks: 32, VotingPeriodBlocks: 33,
		}},
		RestakingPolicy: []*contract.RestakingPolicyEntry{{
			CommitteeId: 34, TargetWeightBps: 35,
			MinStakeUcnpy: 36, MaxStakeUcnpy: 37,
		}},
	}
	bz, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw paramsJSON
	if err := json.Unmarshal(bz, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	dst, err := raw.toContract()
	if err != nil {
		t.Fatalf("toContract: %v", err)
	}
	// proto.Equal checks every field including ones we might have
	// forgotten to thread through paramsJSON.
	if !proto.Equal(src, dst) {
		t.Fatalf("paramsJSON drops a field — round-trip diverged.\n  src=%+v\n  dst=%+v", src, dst)
	}
}
