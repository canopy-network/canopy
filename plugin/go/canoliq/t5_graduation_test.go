package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// t5_graduation_test.go covers autonomy-graduation tracking (T5 / WP §10):
// the five metrics + composite eligibility, the passed-proposal and turnout
// counters at tally, the rolling daily-tx window, and per-tx counting.

// gradParams returns DefaultParams with low graduation thresholds for tests.
func gradParams() *contract.CanoliqParams {
	p := DefaultParams()
	p.GraduationMinTvlUcnpy = 1000
	p.GraduationMinValidators = 2
	p.GraduationMinTurnoutBps = 1000
	p.GraduationMinDailyTx = 5
	p.GraduationMinRunwayMonths = 12
	return p
}

// seedGradAllMet seeds an all-thresholds-met state and refreshes the snapshot.
func seedGradAllMet(t *testing.T, c *Canoliq, s *fakeStore) {
	t.Helper()
	seedParams(t, c, gradParams())
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete:    true,
		TotalPooledCnpy:    2000, // > 1000
		LastDailyTxCount:   10,   // > 5
		TurnoutSumBps:      3000,
		TurnoutSampleCount: 1, // avg 3000 > 1000
	})
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: addr20(0x01), Stake: 1}, {Address: addr20(0x02), Stake: 1}, {Address: addr20(0x03), Stake: 1}, // 3 > 2
	}}))
	s.set(KeyForTreasuryCNPY(), EncodeUint64(1_000_000)) // burn 0 → runway infinite
	if err := c.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
}

func TestT5EligibleWhenAllMet(t *testing.T) {
	c, s := newTestCanoliq()
	seedGradAllMet(t, c, s)
	v := c.plugin.QueryGraduation()
	if !v.Eligible {
		t.Fatalf("should be eligible when all met; metrics=%+v", v.Metrics)
	}
	if len(v.Metrics) != 5 {
		t.Fatalf("want 5 metrics, got %d", len(v.Metrics))
	}
	for _, m := range v.Metrics {
		if !m.Met {
			t.Errorf("metric %s should be met: value=%d threshold=%d", m.Name, m.Value, m.Threshold)
		}
	}
}

// TestT5EligibleFlipsPerMetric knocks each metric below its threshold in turn
// and confirms eligibility flips to false.
func TestT5EligibleFlipsPerMetric(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(g *contract.CanoliqGlobals, s *fakeStore)
	}{
		{"tvl", func(g *contract.CanoliqGlobals, s *fakeStore) { g.TotalPooledCnpy = 500 }},
		{"validators", func(g *contract.CanoliqGlobals, s *fakeStore) {
			s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{{Address: addr20(0x01), Stake: 1}}}))
		}},
		{"turnout", func(g *contract.CanoliqGlobals, s *fakeStore) { g.TurnoutSumBps = 500; g.TurnoutSampleCount = 1 }},
		{"daily_tx", func(g *contract.CanoliqGlobals, s *fakeStore) { g.LastDailyTxCount = 1 }},
		{"runway", func(g *contract.CanoliqGlobals, s *fakeStore) {
			g.TreasurySpentTotal = 1000
			s.set(KeyForTreasuryCNPY(), EncodeUint64(10)) // 10 / (1000/1mo) = 0 months
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s := newTestCanoliq()
			seedGradAllMet(t, c, s)
			g := loadGlobals(t, s)
			tc.break_(g, s)
			seedGlobals(s, g)
			// runway uses height for elapsed months; refresh at 1 month so burn bites.
			if err := c.refreshSnapshot(blocksPerMonth); err != nil {
				t.Fatalf("refresh: %v", err)
			}
			v := c.plugin.QueryGraduation()
			if v.Eligible {
				t.Errorf("breaking %s should make ineligible; metrics=%+v", tc.name, v.Metrics)
			}
		})
	}
}

// TestT5PassedProposalAndTurnoutCounters tallies a passing and a failing
// proposal in one block: passed_proposal_count advances by 1 (only the pass),
// turnout is recorded for both.
func TestT5PassedProposalAndTurnoutCounters(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, shortGovParams()) // scalar quorum 3300 / threshold 5001
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	payload, _ := anypb.New(&contract.ProposalParamChange{Params: DefaultParams()})
	pass := &contract.Proposal{Id: 1, ExpiryHeight: 10, SnapshotTotalStaked: 1000, YesWeight: 600, NoWeight: 100, Payload: payload, Status: contract.ProposalStatus_PROPOSAL_ACTIVE}
	fail := &contract.Proposal{Id: 2, ExpiryHeight: 10, SnapshotTotalStaked: 1000, YesWeight: 100, Payload: payload, Status: contract.ProposalStatus_PROPOSAL_ACTIVE}
	s.set(KeyForProposal(1), mustMarshal(pass))
	s.set(KeyForProposal(2), mustMarshal(fail))
	s.set(KeyForProposalIndex(), mustMarshal(&contract.ProposalIndex{Ids: []uint64{1, 2}}))

	c.plugin.setHeight(20)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: 20}); r.Error != nil {
		t.Fatalf("begin block: %v", r.Error)
	}
	g := loadGlobals(t, s)
	if g.PassedProposalCount != 1 {
		t.Errorf("passed_proposal_count: got %d want 1", g.PassedProposalCount)
	}
	if g.TurnoutSampleCount != 2 {
		t.Errorf("turnout_sample_count: got %d want 2", g.TurnoutSampleCount)
	}
	// turnout: 700/1000 = 7000 bps + 100/1000 = 1000 bps = 8000.
	if g.TurnoutSumBps != 8000 {
		t.Errorf("turnout_sum_bps: got %d want 8000", g.TurnoutSumBps)
	}
}

// TestT5RollingWindowReset checks the window anchors on first call, holds
// before the boundary, and resets exactly at it.
func TestT5RollingWindowReset(t *testing.T) {
	c, s := newTestCanoliq()

	// Init: anchors LastWindowCloseHeight to the current height.
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	if err := c.advanceGraduationWindow(50); err != nil {
		t.Fatalf("advance init: %v", err)
	}
	if g := loadGlobals(t, s); g.LastWindowCloseHeight != 50 {
		t.Fatalf("window should anchor at 50, got %d", g.LastWindowCloseHeight)
	}

	// Just before the boundary: no reset.
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true, LastWindowCloseHeight: 100, DailyTxCountWindow: 42})
	if err := c.advanceGraduationWindow(100 + blocksPerDay - 1); err != nil {
		t.Fatalf("advance pre-boundary: %v", err)
	}
	if g := loadGlobals(t, s); g.DailyTxCountWindow != 42 || g.LastDailyTxCount != 0 {
		t.Fatalf("should not reset before boundary: window=%d last=%d", g.DailyTxCountWindow, g.LastDailyTxCount)
	}

	// Exactly at the boundary: reset, recording the completed count.
	if err := c.advanceGraduationWindow(100 + blocksPerDay); err != nil {
		t.Fatalf("advance boundary: %v", err)
	}
	g := loadGlobals(t, s)
	if g.LastDailyTxCount != 42 || g.DailyTxCountWindow != 0 || g.LastWindowCloseHeight != 100+blocksPerDay {
		t.Errorf("boundary reset wrong: last=%d window=%d close=%d", g.LastDailyTxCount, g.DailyTxCountWindow, g.LastWindowCloseHeight)
	}
}

// TestT5DeliverTxCountsOnSuccessOnly confirms a successful delivery advances
// the window counter and a failed one does not.
func TestT5DeliverTxCountsOnSuccessOnly(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	user := addr20(0x01)
	seedAccount(s, user, 1_010_000) // amount + fee

	depAny, _ := anypb.New(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000})
	req := &contract.PluginDeliverRequest{Tx: &contract.Transaction{Msg: depAny, Fee: 10_000}}
	if r := c.DeliverTx(req); r.Error != nil {
		t.Fatalf("deposit deliver: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.DailyTxCountWindow != 1 {
		t.Fatalf("successful tx should count: got %d want 1", g.DailyTxCountWindow)
	}

	// A failing tx (insufficient balance) must not advance the counter.
	poor := addr20(0x02)
	seedAccount(s, poor, 100)
	depAny2, _ := anypb.New(&contract.MessageCanoliqDeposit{FromAddress: poor, Amount: 1_000_000})
	req2 := &contract.PluginDeliverRequest{Tx: &contract.Transaction{Msg: depAny2, Fee: 10_000}}
	if r := c.DeliverTx(req2); r.Error == nil {
		t.Fatal("expected the under-funded deposit to fail")
	}
	if g := loadGlobals(t, s); g.DailyTxCountWindow != 1 {
		t.Errorf("failed tx should not count: got %d want 1", g.DailyTxCountWindow)
	}
}
