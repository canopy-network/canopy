package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// t1_governance_test.go covers the per-action governance matrix (T1):
// ActionType inference, tier resolution + fallback, tier-snapshotted voting
// period / quorum / approval / timelock, and the F12 (validator eject) +
// F13 (emergency fast-track) dispatch paths.

// TestT1ActionTypeInference pins the payload → ActionType classification,
// including the small/large treasury-spend split at the 1M CLIQ boundary.
func TestT1ActionTypeInference(t *testing.T) {
	cases := []struct {
		name    string
		payload interface{}
		want    contract.ActionType
	}{
		{"param-change", &contract.ProposalParamChange{Params: DefaultParams()}, contract.ActionType_ACTION_FEE_CHANGE},
		{"buyback-unmapped", &contract.ProposalBuyback{CnpyAmount: 1, PriceMicroCnpyPerCliq: 1}, contract.ActionType_ACTION_UNKNOWN},
		{"spend-small-cnpy", &contract.ProposalTreasurySpend{Amount: 5_000_000_000, Denomination: contract.SpendDenomination_SPEND_CNPY}, contract.ActionType_ACTION_TREASURY_SPEND_SMALL},
		{"spend-small-cliq-at-threshold", &contract.ProposalTreasurySpend{Amount: largeSpendCliqThreshold, Denomination: contract.SpendDenomination_SPEND_CLIQ}, contract.ActionType_ACTION_TREASURY_SPEND_SMALL},
		{"spend-large-cliq", &contract.ProposalTreasurySpend{Amount: largeSpendCliqThreshold + 1, Denomination: contract.SpendDenomination_SPEND_CLIQ}, contract.ActionType_ACTION_TREASURY_SPEND_LARGE},
		{"validator-eject", &contract.ProposalValidatorEject{ValidatorAddress: addr20(0x01)}, contract.ActionType_ACTION_VALIDATOR_EJECT},
		{"emergency", &contract.ProposalEmergency{Description: "x"}, contract.ActionType_ACTION_EMERGENCY},
		{"protocol-upgrade", &contract.ProposalProtocolUpgrade{Version: "v2"}, contract.ActionType_ACTION_PROTOCOL_UPGRADE},
	}
	for _, tc := range cases {
		if got := actionTypeForPayload(tc.payload); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

// TestT1DefaultTiersMatchSpec asserts every Tokenomics §7 row is present with
// the documented quorum / approval / timelock, and that tierFor falls back to
// nil for an unmatched action.
func TestT1DefaultTiersMatchSpec(t *testing.T) {
	p := DefaultParams()
	want := map[contract.ActionType][3]uint64{ // {quorumBps, approvalBps, timelockBlocks}
		contract.ActionType_ACTION_FEE_CHANGE:           {500, 5100, blocks48h},
		contract.ActionType_ACTION_TREASURY_SPEND_SMALL: {500, 5100, blocks48h},
		contract.ActionType_ACTION_TREASURY_SPEND_LARGE: {1000, 6700, blocks7d},
		contract.ActionType_ACTION_EMERGENCY:            {800, 6700, 0},
		contract.ActionType_ACTION_VALIDATOR_EJECT:      {500, 5100, blocks48h},
		contract.ActionType_ACTION_PROTOCOL_UPGRADE:     {1000, 6700, blocks7d},
		contract.ActionType_ACTION_AUTONOMY_GRADUATE:    {1500, 7500, blocks14d},
	}
	if len(p.Governance) != len(want) {
		t.Fatalf("tier count: got %d want %d", len(p.Governance), len(want))
	}
	for action, w := range want {
		tier := tierFor(p, action)
		if tier == nil {
			t.Errorf("%v: no tier", action)
			continue
		}
		if tier.QuorumBps != w[0] || tier.ApprovalBps != w[1] || tier.TimelockBlocks != w[2] {
			t.Errorf("%v: got q=%d a=%d tl=%d want q=%d a=%d tl=%d",
				action, tier.QuorumBps, tier.ApprovalBps, tier.TimelockBlocks, w[0], w[1], w[2])
		}
	}
	// Emergency is the only 24h-vote tier; the rest use the 7d window.
	if tierFor(p, contract.ActionType_ACTION_EMERGENCY).VotingPeriodBlocks != blocks24h {
		t.Errorf("emergency voting period should be 24h (%d)", blocks24h)
	}
	if got := ValidateParams(p); got != nil {
		t.Errorf("DefaultParams tiers should validate: %v", got)
	}
}

// TestT1ProposalPassesUsesTier shows quorum and approval are read from the
// snapshotted tier, not the scalar knobs — and that a nil tier falls back to
// the scalars.
func TestT1ProposalPassesUsesTier(t *testing.T) {
	// Scalar knobs deliberately lax so any tier effect is visible.
	params := &contract.CanoliqParams{QuorumBps: 0, PassThresholdBps: 0}

	// Fee tier: 5% quorum / 51% approval. 520 yes / 480 no of 1000 staked.
	feeTier := &contract.GovernanceTier{Action: contract.ActionType_ACTION_FEE_CHANGE, QuorumBps: 500, ApprovalBps: 5100}
	autoTier := &contract.GovernanceTier{Action: contract.ActionType_ACTION_AUTONOMY_GRADUATE, QuorumBps: 1500, ApprovalBps: 7500}

	base := func(tier *contract.GovernanceTier, yes, no uint64) *contract.Proposal {
		return &contract.Proposal{SnapshotTotalStaked: 1000, YesWeight: yes, NoWeight: no, Tier: tier}
	}

	// 52% yes passes the 51% tier but fails the 75% tier.
	if !proposalPasses(base(feeTier, 520, 480), params) {
		t.Error("520/480 should pass the 51% fee tier")
	}
	if proposalPasses(base(autoTier, 520, 480), params) {
		t.Error("520/480 should fail the 75% autonomy tier")
	}
	// Quorum: 100 votes of 1000 clears 5% but misses 15%.
	if !proposalPasses(base(feeTier, 100, 0), params) {
		t.Error("100/1000 should clear the 5% quorum")
	}
	if proposalPasses(base(autoTier, 100, 0), params) {
		t.Error("100/1000 should miss the 15% quorum")
	}
	// Nil tier falls back to scalar knobs (0/0 → any non-empty yes passes).
	if !proposalPasses(&contract.Proposal{SnapshotTotalStaked: 1000, YesWeight: 1, Tier: nil}, params) {
		t.Error("nil tier should fall back to scalar knobs")
	}
}

// stakeProposer seeds a proposer with CNPY + staked CLIQ above the proposal
// minimum and returns its address.
func stakeProposer(t *testing.T, c *Canoliq, s *fakeStore, params *contract.CanoliqParams, b byte) []byte {
	t.Helper()
	proposer := addr20(b)
	seedAccount(s, proposer, 1_000_000)
	seedCLIQ(s, proposer, 10_000_000)
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: proposer, Amount: 10_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("stake: %v", r.Error)
	}
	return proposer
}

// TestT1CreateSnapshotsTierAndVotingPeriod drives proposal creation through
// the real handler and asserts the recorded ActionType + tier + expiry. The
// emergency fast-track uses the 24h window; an equivalent param diff submitted
// as a normal param-change keeps the 7d window.
func TestT1CreateSnapshotsTierAndVotingPeriod(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	proposer := stakeProposer(t, c, s, params, 0x31)
	c.plugin.setHeight(20)

	// Emergency proposal → 24h voting window, ACTION_EMERGENCY tier.
	emPayload, err := anypb.New(&contract.ProposalEmergency{Description: "halt"})
	if err != nil {
		t.Fatalf("anypb: %v", err)
	}
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{
		FromAddress: proposer, Payload: emPayload, Description: "halt",
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create emergency: %v", r.Error)
	}
	em := loadProposal(t, s, 1)
	if em.ActionType != contract.ActionType_ACTION_EMERGENCY {
		t.Errorf("emergency action type: got %v", em.ActionType)
	}
	if em.Tier == nil || em.ExpiryHeight != 20+blocks24h {
		t.Errorf("emergency expiry: got %d want %d", em.ExpiryHeight, 20+blocks24h)
	}

	// Param-change with the same intent → 7d window, ACTION_FEE_CHANGE tier.
	pc := DefaultParams()
	pc.FeeBps = 800
	pcPayload, err := anypb.New(&contract.ProposalParamChange{Params: pc})
	if err != nil {
		t.Fatalf("anypb: %v", err)
	}
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{
		FromAddress: proposer, Payload: pcPayload, Description: "lower fee",
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create param-change: %v", r.Error)
	}
	change := loadProposal(t, s, 2)
	if change.ActionType != contract.ActionType_ACTION_FEE_CHANGE {
		t.Errorf("param-change action type: got %v", change.ActionType)
	}
	if change.ExpiryHeight != 20+blocks7d {
		t.Errorf("param-change expiry: got %d want %d", change.ExpiryHeight, 20+blocks7d)
	}
}

// TestT1TreasuryTimelockFromTier shows the recorded tier drives the spend
// timelock: a small spend gets 48h, a large (>1M CLIQ) spend gets 7d.
func TestT1TreasuryTimelockFromTier(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	small := &contract.Proposal{Id: 1, Tier: tierFor(params, contract.ActionType_ACTION_TREASURY_SPEND_SMALL)}
	if err := c.queueTreasurySpend(small, &contract.ProposalTreasurySpend{
		Recipient: addr20(0xaa), Amount: 100, Denomination: contract.SpendDenomination_SPEND_CNPY,
	}, params, 1000); err != nil {
		t.Fatalf("queue small: %v", err)
	}
	if got := loadSpend(t, s, 1).ExecutableHeight; got != 1000+blocks48h {
		t.Errorf("small spend timelock: got %d want %d", got, 1000+blocks48h)
	}

	large := &contract.Proposal{Id: 2, Tier: tierFor(params, contract.ActionType_ACTION_TREASURY_SPEND_LARGE)}
	if err := c.queueTreasurySpend(large, &contract.ProposalTreasurySpend{
		Recipient: addr20(0xbb), Amount: largeSpendCliqThreshold + 1, Denomination: contract.SpendDenomination_SPEND_CLIQ,
	}, params, 1000); err != nil {
		t.Fatalf("queue large: %v", err)
	}
	if got := loadSpend(t, s, 2).ExecutableHeight; got != 1000+blocks7d {
		t.Errorf("large spend timelock: got %d want %d", got, 1000+blocks7d)
	}
}

// TestT1ValidatorEjectSkipsRewards (F12) confirms a passed eject removes the
// validator from the registry and clears its accrued incentives, after which
// reward sweeps distribute only across the survivors.
func TestT1ValidatorEjectSkipsRewards(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	v1, v2 := addr20(0x01), addr20(0x02)
	registry := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: v1, Stake: 500}, {Address: v2, Stake: 500},
	}}
	s.set(KeyForValidatorRegistry(), mustMarshal(registry))
	// Pre-ejection: both validators share the first sweep (X=1000 → validators=18 → 9/9).
	s.set(contract.KeyForFeePool(c.Config.ChainId), mustMarshal(&contract.Pool{Id: c.Config.ChainId, Amount: 1000}))
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(v1))); got != 9 {
		t.Errorf("pre-eject v1 share: got %d want 9", got)
	}

	// Eject v1 via dispatchPassed.
	ejectPayload, _ := anypb.New(&contract.ProposalValidatorEject{ValidatorAddress: v1})
	prop := &contract.Proposal{Id: 7, Payload: ejectPayload, Status: contract.ProposalStatus_PROPOSAL_PASSED}
	if err := c.dispatchPassed(prop, params, 2); err != nil {
		t.Fatalf("dispatch eject: %v", err)
	}
	// v1 removed from registry, accrued incentives cleared.
	reg := new(contract.ValidatorRegistry)
	if err := contract.Unmarshal(s.get(KeyForValidatorRegistry()), reg); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	if len(reg.Entries) != 1 || string(reg.Entries[0].Address) != string(v2) {
		t.Fatalf("registry after eject: %+v", reg.Entries)
	}
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(v1))); got != 0 {
		t.Errorf("v1 incentives after eject: got %d want 0", got)
	}

	// Post-ejection sweep: drive a clean delta of 1000 (the pool sits at 928
	// after sweep 1: 880 net + 48 user-rebate re-credited). validators=18, all
	// to the lone survivor v2.
	g := loadGlobals(t, s)
	s.set(contract.KeyForFeePool(c.Config.ChainId), mustMarshal(&contract.Pool{Id: c.Config.ChainId, Amount: g.LastProcessedRewardPool + 1000}))
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 3}); err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(v1))); got != 0 {
		t.Errorf("v1 should receive nothing post-eject, got %d", got)
	}
	v2after := DecodeUint64(s.get(KeyForValidatorIncentives(v2)))
	if v2after != 9+18 {
		t.Errorf("v2 should hold its pre-eject 9 plus the full post-eject 18, got %d", v2after)
	}
}

// TestT1EmergencyParamDiffNoTimelock (F13) confirms an emergency param diff
// applies immediately on pass (the emergency tier carries a zero timelock).
func TestT1EmergencyParamDiffApplied(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	newParams := DefaultParams()
	newParams.FeeBps = 1500
	emPayload, _ := anypb.New(&contract.ProposalEmergency{
		Description: "raise fee under stress",
		ParamChange: &contract.ProposalParamChange{Params: newParams},
	})
	prop := &contract.Proposal{Id: 3, Payload: emPayload, Status: contract.ProposalStatus_PROPOSAL_PASSED}
	if err := c.dispatchPassed(prop, params, 5); err != nil {
		t.Fatalf("dispatch emergency: %v", err)
	}
	got, err := c.LoadParams()
	if err != nil {
		t.Fatalf("load params: %v", err)
	}
	if got.FeeBps != 1500 {
		t.Errorf("emergency param diff not applied: fee_bps got %d want 1500", got.FeeBps)
	}
	_ = s
}

// TestT1MixedFlightIndependentTally runs two proposals of different tiers in
// the same voting window with an identical 60% yes ratio: the fee-change
// (51% tier) passes and applies; the large treasury spend (67% tier) fails.
func TestT1MixedFlightIndependentTally(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams() // tier voting periods shortened to 5 blocks
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	// Two stakers create a 6M / 4M split = 60% yes on every proposal.
	a := addr20(0x41)
	b := addr20(0x42)
	seedAccount(s, a, 1_000_000)
	seedAccount(s, b, 1_000_000)
	seedCLIQ(s, a, 6_000_000)
	seedCLIQ(s, b, 4_000_000)
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: a, Amount: 6_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("stake a: %v", r.Error)
	}
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: b, Amount: 4_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("stake b: %v", r.Error)
	}

	c.plugin.setHeight(10)
	// Proposal 1: fee-change (ACTION_FEE_CHANGE, 51% approval).
	feeParams := shortGovParams()
	feeParams.FeeBps = 800
	feePayload, _ := anypb.New(&contract.ProposalParamChange{Params: feeParams})
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{FromAddress: a, Payload: feePayload}, 10_000, params); r.Error != nil {
		t.Fatalf("create fee-change: %v", r.Error)
	}
	// Proposal 2: large CLIQ treasury spend (ACTION_TREASURY_SPEND_LARGE, 67%).
	largePayload, _ := anypb.New(&contract.ProposalTreasurySpend{
		Recipient: addr20(0xcc), Amount: largeSpendCliqThreshold + 1, Denomination: contract.SpendDenomination_SPEND_CLIQ,
	})
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{FromAddress: a, Payload: largePayload}, 10_000, params); r.Error != nil {
		t.Fatalf("create large-spend: %v", r.Error)
	}

	// Same 60% yes ratio on both proposals.
	for _, id := range []uint64{1, 2} {
		if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{FromAddress: a, ProposalId: id, Choice: contract.VoteChoice_VOTE_YES}, 10_000, params); r.Error != nil {
			t.Fatalf("vote a on %d: %v", id, r.Error)
		}
		if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{FromAddress: b, ProposalId: id, Choice: contract.VoteChoice_VOTE_NO}, 10_000, params); r.Error != nil {
			t.Fatalf("vote b on %d: %v", id, r.Error)
		}
	}

	// Advance past expiry (10 + 5) and tally both in one BeginBlock.
	c.plugin.setHeight(20)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: 20}); r.Error != nil {
		t.Fatalf("begin block: %v", r.Error)
	}

	// Fee-change passed at 60% ≥ 51% and applied.
	got, _ := c.LoadParams()
	if got.FeeBps != 800 {
		t.Errorf("fee-change should have passed: fee_bps got %d want 800", got.FeeBps)
	}
	// Large spend failed at 60% < 67%: no spend queued.
	if sp := s.get(KeyForTreasurySpend(1)); sp != nil {
		t.Error("large treasury spend should have failed (no spend record)")
	}
	// Both proposals cleaned up regardless of outcome.
	if idx := loadProposalIndex(s); len(idx.Ids) != 0 {
		t.Errorf("proposal index should be empty post-tally, got %v", idx.Ids)
	}
}
