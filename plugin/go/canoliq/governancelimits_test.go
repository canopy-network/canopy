package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	govBefore = MainnetGovernanceLimitsHeight - 1_000
	govAfter  = MainnetGovernanceLimitsHeight
)

// newMainnetGov returns a mainnet-profile Canoliq at `height` with default
// params and completed genesis.
func newMainnetGov(t *testing.T, height uint64) (*Canoliq, *fakeStore, *contract.CanoliqParams) {
	t.Helper()
	c, s := newMainnetAt(t, height)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	params, err := c.LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	return c, s, params
}

func paramChangeProposal(t *testing.T, id uint64, next *contract.CanoliqParams, tier *contract.GovernanceTier) *contract.Proposal {
	t.Helper()
	payload, err := anypb.New(&contract.ProposalParamChange{Params: next})
	if err != nil {
		t.Fatal(err)
	}
	return &contract.Proposal{Id: id, Payload: payload, Tier: tier, ActionType: contract.ActionType_ACTION_FEE_CHANGE}
}

func mustFeeBps(t *testing.T, c *Canoliq) uint64 {
	t.Helper()
	p, err := c.LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	return p.FeeBps
}

func TestParamChangeAppliesAfterTierTimelock(t *testing.T) {
	t.Run("before_activation_applies_immediately", func(t *testing.T) {
		c, _, params := newMainnetGov(t, govBefore)
		next := proto.Clone(params).(*contract.CanoliqParams)
		next.FeeBps = 800
		tier := tierFor(params, contract.ActionType_ACTION_FEE_CHANGE)
		if err := c.dispatchPassed(paramChangeProposal(t, 1, next, tier), params, govBefore); err != nil {
			t.Fatal(err)
		}
		if got := mustFeeBps(t, c); got != 800 {
			t.Fatalf("fee_bps = %d, want 800", got)
		}
	})
	t.Run("from_activation_waits_for_timelock", func(t *testing.T) {
		c, s, params := newMainnetGov(t, govAfter)
		next := proto.Clone(params).(*contract.CanoliqParams)
		next.FeeBps = 800
		tier := tierFor(params, contract.ActionType_ACTION_FEE_CHANGE)
		if tier.TimelockBlocks == 0 {
			t.Fatal("fixture needs a non-zero fee-change timelock")
		}
		if err := c.dispatchPassed(paramChangeProposal(t, 1, next, tier), params, govAfter); err != nil {
			t.Fatal(err)
		}
		if got := mustFeeBps(t, c); got != 1200 {
			t.Fatalf("applied before timelock: fee_bps = %d", got)
		}
		due := govAfter + tier.TimelockBlocks
		if err := c.applyPendingParamChanges(due - 1); err != nil {
			t.Fatal(err)
		}
		if got := mustFeeBps(t, c); got != 1200 {
			t.Fatalf("applied one block early: fee_bps = %d", got)
		}
		if err := c.applyPendingParamChanges(due); err != nil {
			t.Fatal(err)
		}
		if got := mustFeeBps(t, c); got != 800 {
			t.Fatalf("not applied at timelock: fee_bps = %d", got)
		}
		if s.get(KeyForPendingParamChange(1)) != nil || s.get(KeyForPendingParamIndex()) != nil {
			t.Fatal("pending record not cleared")
		}
	})
	t.Run("from_activation_emergency_zero_timelock_still_immediate", func(t *testing.T) {
		c, _, params := newMainnetGov(t, govAfter)
		next := proto.Clone(params).(*contract.CanoliqParams)
		next.FeeBps = 1500
		payload, _ := anypb.New(&contract.ProposalEmergency{ParamChange: &contract.ProposalParamChange{Params: next}})
		prop := &contract.Proposal{Id: 2, Payload: payload, Tier: tierFor(params, contract.ActionType_ACTION_EMERGENCY)}
		if err := c.dispatchPassed(prop, params, govAfter); err != nil {
			t.Fatal(err)
		}
		if got := mustFeeBps(t, c); got != 1500 {
			t.Fatalf("fee_bps = %d, want 1500", got)
		}
	})
}

func TestParamChangeEnforcesGovernanceFloors(t *testing.T) {
	mutations := map[string]func(p *contract.CanoliqParams){
		"zero_scalar_quorum": func(p *contract.CanoliqParams) { p.QuorumBps = 0 },
		"zero_tier_quorum":   func(p *contract.CanoliqParams) { p.Governance[0].QuorumBps = 0 },
		"short_voting":       func(p *contract.CanoliqParams) { p.VotingPeriodBlocks = 10 },
		"short_tier_voting":  func(p *contract.CanoliqParams) { p.Governance[0].VotingPeriodBlocks = 10 },
		"low_approval":       func(p *contract.CanoliqParams) { p.PassThresholdBps = 1 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			// Governance knobs are sensitive, so vote on the strictest tier to
			// isolate the floor check.
			for _, h := range []uint64{govBefore, govAfter} {
				c, _, params := newMainnetGov(t, h)
				next := proto.Clone(params).(*contract.CanoliqParams)
				mutate(next)
				err := c.dispatchPassed(paramChangeProposal(t, 1, next, strictestTier(params)), params, h)
				if h == govBefore && err != nil {
					t.Fatalf("before activation: %v", err)
				}
				if h == govAfter && (err == nil || err.Code != codeInvalidParams) {
					t.Fatalf("from activation: want invalid params, got %v", err)
				}
			}
		})
	}
}

func TestSensitiveParamChangeRequiresStrictestTier(t *testing.T) {
	mutate := func(p *contract.CanoliqParams) { p.TreasuryThreshold = 1 }

	t.Run("dispatch_on_ordinary_tier", func(t *testing.T) {
		for _, h := range []uint64{govBefore, govAfter} {
			c, _, params := newMainnetGov(t, h)
			next := proto.Clone(params).(*contract.CanoliqParams)
			mutate(next)
			err := c.dispatchPassed(paramChangeProposal(t, 1, next, tierFor(params, contract.ActionType_ACTION_FEE_CHANGE)), params, h)
			if h == govBefore && err != nil {
				t.Fatalf("before activation: %v", err)
			}
			if h == govAfter && err == nil {
				t.Fatal("from activation: sensitive change on an ordinary tier accepted")
			}
		}
	})

	t.Run("create_records_strictest_tier", func(t *testing.T) {
		c, s, params := newMainnetGov(t, govAfter)
		proposer := addr20(0x31)
		seedAccount(s, proposer, 1_000_000)
		seedCPLQStake(t, s, proposer, 10_000_000)
		next := proto.Clone(params).(*contract.CanoliqParams)
		mutate(next)
		payload, _ := anypb.New(&contract.ProposalParamChange{Params: next})
		if r := c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{FromAddress: proposer, Payload: payload}, 10_000, params); r.Error != nil {
			t.Fatal(r.Error)
		}
		prop := loadProposal(t, s, 1)
		if !tierAtLeast(prop.Tier, strictestTier(params)) {
			t.Fatalf("tier not upgraded: %+v", prop.Tier)
		}
		if prop.ExpiryHeight != govAfter+strictestTier(params).VotingPeriodBlocks {
			t.Fatalf("expiry %d", prop.ExpiryHeight)
		}
		// And it is accepted at dispatch.
		if err := c.dispatchPassed(prop, params, govAfter); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("emergency_cannot_touch_sensitive", func(t *testing.T) {
		c, s, params := newMainnetGov(t, govAfter)
		proposer := addr20(0x31)
		seedAccount(s, proposer, 1_000_000)
		seedCPLQStake(t, s, proposer, 10_000_000)
		next := proto.Clone(params).(*contract.CanoliqParams)
		mutate(next)
		payload, _ := anypb.New(&contract.ProposalEmergency{ParamChange: &contract.ProposalParamChange{Params: next}})
		r := c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{FromAddress: proposer, Payload: payload}, 10_000, params)
		if r.Error == nil {
			t.Fatal("emergency sensitive change accepted at create")
		}
		prop := &contract.Proposal{Id: 9, Payload: payload, Tier: strictestTier(params)}
		if err := c.dispatchPassed(prop, params, govAfter); err == nil {
			t.Fatal("emergency sensitive change accepted at dispatch")
		}
	})
}

// seedCPLQStake writes a stake record directly and adds it to the index.
func seedCPLQStake(t *testing.T, s *fakeStore, addr []byte, amount uint64) {
	t.Helper()
	bz, _ := contract.Marshal(&contract.CPLQStake{Address: addr, Amount: amount})
	s.set(KeyForCPLQStake(addr), bz)
	idx := new(contract.CPLQStakeIndex)
	if b := s.get(KeyForCPLQStakeIndex()); b != nil {
		_ = contract.Unmarshal(b, idx)
	}
	idx.Addresses = append(idx.Addresses, addr)
	ib, _ := contract.Marshal(idx)
	s.set(KeyForCPLQStakeIndex(), ib)
}

func TestTallyAppliesQuorumFloor(t *testing.T) {
	prop := &contract.Proposal{
		SnapshotTotalStaked: 1_000_000,
		YesWeight:           1, // 0.0001% turnout
		Tier:                &contract.GovernanceTier{QuorumBps: 0, ApprovalBps: 0},
	}
	for _, h := range []uint64{govBefore, govAfter} {
		c, _, params := newMainnetGov(t, h)
		got := c.proposalPassesAt(prop, params, h)
		if h == govBefore && !got {
			t.Fatal("before activation: legacy tally changed")
		}
		if h == govAfter && got {
			t.Fatal("from activation: passed below the quorum floor")
		}
	}
}

func TestProposalVotingPeriodFloor(t *testing.T) {
	for _, h := range []uint64{govBefore, govAfter} {
		c, s, params := newMainnetGov(t, h)
		params.VotingPeriodBlocks = 5
		for _, tier := range params.Governance {
			tier.VotingPeriodBlocks = 5
		}
		params.CplqUnstakingBlocks = 5
		seedParams(t, c, params)
		proposer := addr20(0x31)
		seedAccount(s, proposer, 1_000_000)
		seedCPLQStake(t, s, proposer, 10_000_000)
		next := proto.Clone(params).(*contract.CanoliqParams)
		next.FeeBps = 800
		payload, _ := anypb.New(&contract.ProposalParamChange{Params: next})
		if r := c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{FromAddress: proposer, Payload: payload}, 10_000, params); r.Error != nil {
			t.Fatal(r.Error)
		}
		want := h + 5
		if h == govAfter {
			want = h + MinGovernanceVotingPeriodBlocks
		}
		if got := loadProposal(t, s, 1).ExpiryHeight; got != want {
			t.Fatalf("height %d: expiry %d, want %d", h, got, want)
		}
	}
}

func TestActiveProposalLimits(t *testing.T) {
	create := func(c *Canoliq, params *contract.CanoliqParams, who []byte) *contract.PluginError {
		payload, _ := anypb.New(&contract.ProposalProtocolUpgrade{})
		return c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{FromAddress: who, Payload: payload}, 10_000, params).Error
	}
	t.Run("per_proposer", func(t *testing.T) {
		for _, h := range []uint64{govBefore, govAfter} {
			c, s, params := newMainnetGov(t, h)
			p := addr20(0x31)
			seedAccount(s, p, 1_000_000)
			seedCPLQStake(t, s, p, 10_000_000)
			for i := 0; i < MaxActiveProposalsPerProposer; i++ {
				if err := create(c, params, p); err != nil {
					t.Fatal(err)
				}
			}
			err := create(c, params, p)
			if h == govBefore && err != nil {
				t.Fatalf("before activation: %v", err)
			}
			if h == govAfter && (err == nil || err.Code != codeTooManyProposals) {
				t.Fatalf("from activation: got %v", err)
			}
		}
	})
	t.Run("global", func(t *testing.T) {
		c, s, params := newMainnetGov(t, govAfter)
		ids := make([]uint64, MaxActiveProposals)
		for i := range ids {
			ids[i] = uint64(1000 + i)
		}
		ib, _ := contract.Marshal(&contract.ProposalIndex{Ids: ids})
		s.set(KeyForProposalIndex(), ib)
		p := addr20(0x31)
		seedAccount(s, p, 1_000_000)
		seedCPLQStake(t, s, p, 10_000_000)
		if err := create(c, params, p); err == nil || err.Code != codeTooManyProposals {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCPLQStakeMinimum(t *testing.T) {
	for _, h := range []uint64{govBefore, govAfter} {
		c, s, params := newMainnetGov(t, h)
		a := addr20(0x41)
		seedAccount(s, a, 1_000_000)
		seedCPLQ(s, a, 10_000_000)
		r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 1}, 10_000, params)
		if h == govBefore && r.Error != nil {
			t.Fatalf("before activation: %v", r.Error)
		}
		if h == govAfter && (r.Error == nil || r.Error.Code != codeStakeBelowMinimum) {
			t.Fatalf("from activation: got %v", r.Error)
		}
	}
	// Partial unstake that would leave dust.
	for _, h := range []uint64{govBefore, govAfter} {
		c, s, params := newMainnetGov(t, h)
		a := addr20(0x42)
		seedAccount(s, a, 1_000_000)
		seedCPLQStake(t, s, a, 2*MinCplqStakeUcplq)
		r := c.DeliverMessageCPLQUnstake(&contract.MessageCPLQUnstake{FromAddress: a, Amount: 2*MinCplqStakeUcplq - 1}, 10_000, params)
		if h == govBefore && r.Error != nil {
			t.Fatalf("before activation: %v", r.Error)
		}
		if h == govAfter && (r.Error == nil || r.Error.Code != codeStakeBelowMinimum) {
			t.Fatalf("from activation: got %v", r.Error)
		}
	}
}

func TestBoostedStakeRunningTotalMatchesWalk(t *testing.T) {
	c, s, params := newMainnetGov(t, govAfter)
	a, b := addr20(0x51), addr20(0x52)
	for _, x := range [][]byte{a, b} {
		seedAccount(s, x, 1_000_000)
		seedCPLQ(s, x, 100_000_000)
	}
	// A stake that predates activation: the first update must fold it in.
	seedCPLQStake(t, s, addr20(0x53), 7_000_000)
	steps := []func() *contract.PluginDeliverResponse{
		func() *contract.PluginDeliverResponse {
			return c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 10_000_000}, 10_000, params)
		},
		func() *contract.PluginDeliverResponse {
			return c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: b, Amount: 20_000_000, LockTier: contract.LockTier_LOCK_12M}, 10_000, params)
		},
		func() *contract.PluginDeliverResponse {
			return c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 5_000_000, LockTier: contract.LockTier_LOCK_3M}, 10_000, params)
		},
		func() *contract.PluginDeliverResponse {
			return c.DeliverMessageCPLQUnstake(&contract.MessageCPLQUnstake{FromAddress: addr20(0x53), Amount: 7_000_000}, 10_000, params)
		},
	}
	seedAccount(s, addr20(0x53), 1_000_000)
	for i, step := range steps {
		if r := step(); r.Error != nil {
			t.Fatalf("step %d: %v", i, r.Error)
		}
		walk, err := c.boostedStakeTotal()
		if err != nil {
			t.Fatal(err)
		}
		if got := DecodeUint64(s.get(KeyForBoostedStakeTotal())); got != walk {
			t.Fatalf("step %d: running total %d, walk %d", i, got, walk)
		}
	}
}

func TestProposalCleanupDeletesVotes(t *testing.T) {
	for _, h := range []uint64{govBefore, govAfter} {
		c, s, _ := newMainnetGov(t, h)
		voters := [][]byte{addr20(0x61), addr20(0x62)}
		for _, v := range voters {
			s.set(KeyForVote(7, v), []byte{1})
		}
		other := KeyForVote(8, voters[0])
		s.set(other, []byte{1})
		if err := c.cleanupProposal(&contract.Proposal{Id: 7}); err != nil {
			t.Fatal(err)
		}
		for _, v := range voters {
			present := s.get(KeyForVote(7, v)) != nil
			if h == govBefore && !present {
				t.Fatal("before activation: votes deleted")
			}
			if h == govAfter && present {
				t.Fatal("from activation: vote left behind")
			}
		}
		if s.get(other) == nil {
			t.Fatal("another proposal's vote was deleted")
		}
	}
}

func TestTreasurySpendToCallerKeepsPayout(t *testing.T) {
	const start, amount, fee = uint64(50_000), uint64(1_000_000), uint64(10_000)
	for _, h := range []uint64{govBefore, govAfter} {
		c, s, params := newMainnetGov(t, h)
		rcpt := addr20(0x77)
		seedAccount(s, rcpt, start)
		s.set(KeyForTreasuryCNPY(), EncodeUint64(5_000_000))
		spend := &contract.TreasurySpend{Id: 1, ProposalId: 9,
			Payload: &contract.ProposalTreasurySpend{Recipient: rcpt, Amount: amount, Denomination: contract.SpendDenomination_SPEND_CNPY}}
		bz, _ := contract.Marshal(spend)
		s.set(KeyForTreasurySpend(1), bz)
		ib, _ := contract.Marshal(&contract.ProposalIndex{Ids: []uint64{1}})
		s.set(KeyForSpendIndex(), ib)
		if r := c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{FromAddress: rcpt, ProposalId: 9}, fee, params); r.Error != nil {
			t.Fatal(r.Error)
		}
		got := readAccount(s, rcpt)
		if h == govBefore && got != start-fee {
			t.Fatalf("before activation: legacy balance changed: %d", got)
		}
		if h == govAfter && got != start-fee+amount {
			t.Fatalf("from activation: balance %d, want %d", got, start-fee+amount)
		}
	}
}

func TestHandlersUseRequestHeightFromActivation(t *testing.T) {
	c, _ := newMainnetAt(t, MainnetGovernanceLimitsHeight+50)
	c.reqHeight = MainnetGovernanceLimitsHeight - 1
	if got := c.currentHeight(); got != MainnetGovernanceLimitsHeight+50 {
		t.Fatalf("before activation: got %d, want the plugin height", got)
	}
	c.reqHeight = MainnetGovernanceLimitsHeight + 10
	if got := c.currentHeight(); got != MainnetGovernanceLimitsHeight+10 {
		t.Fatalf("from activation: got %d, want the request height", got)
	}
	c.reqHeight = 0
	if got := c.currentHeight(); got != MainnetGovernanceLimitsHeight+50 {
		t.Fatalf("no request height: got %d", got)
	}
}

func TestUnlockedAmountCliffBeforeStart(t *testing.T) {
	sched := &contract.VestingSchedule{TotalAmount: 1_000, CliffHeight: 10, StartHeight: 100, EndHeight: 200}
	for _, h := range []uint64{govBefore, govAfter} {
		c, _ := newMainnetAt(t, h)
		got := c.unlockedAmountAt(sched, 50)
		if h == govBefore && got != unlockedAmount(sched, 50) {
			t.Fatal("before activation: legacy result changed")
		}
		if h == govAfter && got != 0 {
			t.Fatalf("from activation: unlocked %d before start", got)
		}
	}
}
