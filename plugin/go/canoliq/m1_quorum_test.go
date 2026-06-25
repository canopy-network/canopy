package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestM1SnapshotUsesBoostedStakeTotal proves the M1 fix: a created proposal's
// quorum/turnout denominator is the lock-boosted stake total (voteWeightFor
// summed over all stakers), not the raw TotalStakedCplq. Vote tallies are
// boosted by the same multiplier, so denominator and numerator now share a
// basis and a long-lock voter can no longer clear quorum with a fraction of the
// raw stake the threshold implies.
func TestM1SnapshotUsesBoostedStakeTotal(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{})

	locker := addr20(0x61) // 24-month lock -> 4x voting weight
	plain := addr20(0x62)  // no lock -> 1x
	seedAccount(s, locker, 500_000)
	seedAccount(s, plain, 500_000)
	seedCPLQ(s, locker, 1_000_000)
	seedCPLQ(s, plain, 3_000_000)
	if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{
		FromAddress: locker, Amount: 1_000_000, LockTier: contract.LockTier_LOCK_24M,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("locker stake: %v", r.Error)
	}
	if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{
		FromAddress: plain, Amount: 3_000_000,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("plain stake: %v", r.Error)
	}

	// Raw total = 4M; boosted = 4*1M (locker) + 1*3M (plain) = 7M.
	const wantBoosted = 4_000_000 + 3_000_000
	if got, err := c.boostedStakeTotal(); err != nil || got != wantBoosted {
		t.Fatalf("boostedStakeTotal: got %d err %v, want %d", got, err, wantBoosted)
	}

	c.plugin.setHeight(10)
	payload, _ := anypb.New(&contract.ProposalParamChange{Params: params})
	if r := c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{
		FromAddress: locker, Payload: payload,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create: %v", r.Error)
	}
	prop := loadProposal(t, s, 1)
	if prop.SnapshotTotalStaked != wantBoosted {
		t.Fatalf("proposal snapshot: got %d want boosted %d (raw would be 4_000_000)",
			prop.SnapshotTotalStaked, wantBoosted)
	}
}

// TestM1TurnoutClampedToOneHundredPercent verifies the defensive per-proposal
// turnout clamp: even if a proposal's recorded vote weight exceeds its snapshot
// (e.g. an under-counted snapshot), the turnout contribution to the T5
// graduation metric is capped at 10000 bps.
func TestM1TurnoutClampedToOneHundredPercent(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	// Hand-craft an expired proposal whose tallied weight is double its
	// snapshot — turnout would be 20000 bps without the clamp. The split
	// (mostly NO) fails the pass threshold so it is not dispatched (no payload),
	// but turnout is recorded regardless of outcome.
	prop := &contract.Proposal{
		Id:                  1,
		CreationHeight:      1,
		ExpiryHeight:        5,
		SnapshotTotalStaked: 1_000_000,
		YesWeight:           500_000,
		NoWeight:            1_500_000,
		Status:              contract.ProposalStatus_PROPOSAL_ACTIVE,
	}
	pBz, _ := contract.Marshal(prop)
	s.set(KeyForProposal(1), pBz)
	idx := &contract.ProposalIndex{Ids: []uint64{1}}
	iBz, _ := contract.Marshal(idx)
	s.set(KeyForProposalIndex(), iBz)

	c.plugin.setHeight(10)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: 10}); r.Error != nil {
		t.Fatalf("begin block: %v", r.Error)
	}

	g := loadGlobals(t, s)
	if g.TurnoutSumBps != 10_000 {
		t.Fatalf("turnout sum: got %d want 10000 (clamped)", g.TurnoutSumBps)
	}
	if g.TurnoutSampleCount != 1 {
		t.Fatalf("turnout sample count: got %d want 1", g.TurnoutSampleCount)
	}
}
