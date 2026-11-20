package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// t2_voteescrow_test.go covers vote-escrow lock multipliers (T2): the tier
// resolver, lock-scaled voting weight, lock-scaled reward boost in the
// buyback distribution, and the unstake lock gate.

// TestT2TierMultipliers pins the Tokenomics §4.2 voting multipliers, reward
// boosts, and lock durations, and that validLockTier rejects out-of-range.
func TestT2TierMultipliers(t *testing.T) {
	cases := []struct {
		tier      contract.LockTier
		voteMult  uint64
		boost     uint64
		durMonths uint64
	}{
		{contract.LockTier_LOCK_NONE, 10_000, 0, 0},
		{contract.LockTier_LOCK_3M, 15_000, 1_000, 3},
		{contract.LockTier_LOCK_6M, 20_000, 2_500, 6},
		{contract.LockTier_LOCK_12M, 30_000, 5_000, 12},
		{contract.LockTier_LOCK_24M, 40_000, 7_500, 24},
	}
	for _, tc := range cases {
		gotVote, gotBoost := tierMultipliers(tc.tier)
		if gotVote != tc.voteMult || gotBoost != tc.boost {
			t.Errorf("%v: got vote=%d boost=%d want vote=%d boost=%d", tc.tier, gotVote, gotBoost, tc.voteMult, tc.boost)
		}
		if got := lockTierDurationBlocks(tc.tier); got != tc.durMonths*blocksPerMonth {
			t.Errorf("%v: duration got %d want %d", tc.tier, got, tc.durMonths*blocksPerMonth)
		}
		if !validLockTier(tc.tier) {
			t.Errorf("%v: should be a valid tier", tc.tier)
		}
	}
	if validLockTier(contract.LockTier(99)) {
		t.Error("out-of-range lock tier should be invalid")
	}
}

// TestT2VoteWeightForScalesWithTier checks the per-tier weight scaling and
// that a LOCK_24M staker out-votes three LOCK_NONE stakers of equal stake.
func TestT2VoteWeightForScalesWithTier(t *testing.T) {
	const x = 1_000_000
	none := &contract.CPLQStake{Address: addr20(0x01), Amount: x, LockTier: contract.LockTier_LOCK_NONE}
	max := &contract.CPLQStake{Address: addr20(0x02), Amount: x, LockTier: contract.LockTier_LOCK_24M}
	if got := voteWeightFor(none); got != x {
		t.Errorf("LOCK_NONE weight: got %d want %d", got, x)
	}
	if got := voteWeightFor(max); got != 4*x {
		t.Errorf("LOCK_24M weight: got %d want %d", got, 4*x)
	}
	if voteWeightFor(max) <= 3*voteWeightFor(none) {
		t.Error("one LOCK_24M staker should out-vote three LOCK_NONE stakers of equal stake")
	}
	if voteWeightFor(nil) != 0 || voteWeightFor(&contract.CPLQStake{}) != 0 {
		t.Error("absent stake should carry zero weight")
	}
}

// TestT2VoteTallyAppliesMultiplier drives a full proposal vote: one LOCK_24M
// yes-voter and three LOCK_NONE no-voters of equal stake — the boosted yes
// weight (4X) exceeds the combined no weight (3X).
func TestT2VoteTallyAppliesMultiplier(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})

	const x = 2_000_000
	yes := addr20(0x21)
	no1, no2, no3 := addr20(0x31), addr20(0x32), addr20(0x33)
	stakeAt := func(addr []byte, tier contract.LockTier) {
		seedAccount(s, addr, 1_000_000)
		seedCPLQ(s, addr, x)
		if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: addr, Amount: x, LockTier: tier}, 10_000, params); r.Error != nil {
			t.Fatalf("stake %x: %v", addr[0], r.Error)
		}
	}
	c.plugin.setHeight(1)
	stakeAt(yes, contract.LockTier_LOCK_24M)
	stakeAt(no1, contract.LockTier_LOCK_NONE)
	stakeAt(no2, contract.LockTier_LOCK_NONE)
	stakeAt(no3, contract.LockTier_LOCK_NONE)

	c.plugin.setHeight(10)
	payload, _ := anypb.New(&contract.ProposalParamChange{Params: shortGovParams()})
	if r := c.DeliverMessageCPLQProposalCreate(&contract.MessageCPLQProposalCreate{FromAddress: yes, Payload: payload}, 10_000, params); r.Error != nil {
		t.Fatalf("create: %v", r.Error)
	}
	vote := func(addr []byte, choice contract.VoteChoice) {
		if r := c.DeliverMessageCPLQVote(&contract.MessageCPLQVote{FromAddress: addr, ProposalId: 1, Choice: choice}, 10_000, params); r.Error != nil {
			t.Fatalf("vote %x: %v", addr[0], r.Error)
		}
	}
	vote(yes, contract.VoteChoice_VOTE_YES)
	vote(no1, contract.VoteChoice_VOTE_NO)
	vote(no2, contract.VoteChoice_VOTE_NO)
	vote(no3, contract.VoteChoice_VOTE_NO)

	prop := loadProposal(t, s, 1)
	if prop.YesWeight != 4*x {
		t.Errorf("yes weight: got %d want %d (4× LOCK_24M)", prop.YesWeight, 4*x)
	}
	if prop.NoWeight != 3*x {
		t.Errorf("no weight: got %d want %d (3× LOCK_NONE)", prop.NoWeight, 3*x)
	}
	if prop.YesWeight <= prop.NoWeight {
		t.Error("boosted LOCK_24M yes should out-weigh three LOCK_NONE no votes")
	}
}

// TestT2RewardBoostInBuybackDistribution seeds a LOCK_NONE 100 + LOCK_12M 100
// staker pair and confirms the LOCK_12M staker receives 1.5× the LOCK_NONE
// staker's share, with exact conservation.
func TestT2RewardBoostInBuybackDistribution(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)

	none := addr20(0xa1)
	locked := addr20(0xb1)
	const stake uint64 = 100
	s.set(KeyForCPLQStake(none), mustMarshal(&contract.CPLQStake{Address: none, Amount: stake, StakedAtHeight: 1, LockTier: contract.LockTier_LOCK_NONE}))
	s.set(KeyForCPLQStake(locked), mustMarshal(&contract.CPLQStake{Address: locked, Amount: stake, StakedAtHeight: 1, LockTier: contract.LockTier_LOCK_12M, LockEndHeight: 9_999_999}))
	s.set(KeyForCPLQStakeIndex(), mustMarshal(&contract.CPLQStakeIndex{Addresses: [][]byte{none, locked}}))
	seedGlobals(s, &contract.CanoliqGlobals{TotalStakedCplq: 2 * stake})

	// boosted weights: none=100, locked=150, total=250; acquire 250.
	const treasuryCPLQ, buybackPoolAmt uint64 = 1_000_000, 1_000_000
	s.set(KeyForTreasuryCPLQ(), EncodeUint64(treasuryCPLQ))
	s.set(KeyForBuybackPool(), EncodeUint64(buybackPoolAmt))
	const proposalID, cnpyAmount, price uint64 = 7, 250, 1_000_000
	s.set(KeyForBuybackOrder(proposalID), mustMarshal(&contract.BuybackOrder{
		ProposalId: proposalID,
		Mode:       contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS,
		Payload:    &contract.ProposalBuyback{CnpyAmount: cnpyAmount, PriceMicroCnpyPerCplq: price, Mode: contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS},
	}))
	trigger := addr20(0xcc)
	seedAccount(s, trigger, 1_000_000)
	if r := c.DeliverMessageBuybackExecute(&contract.MessageBuybackExecute{FromAddress: trigger, ProposalId: proposalID}, 10_000, params); r.Error != nil {
		t.Fatalf("execute: %v", r.Error)
	}
	cplqAcquired := cnpyAmount * 1_000_000 / price // 250
	gotNone := readCplq(s, none)
	gotLocked := readCplq(s, locked)
	if gotNone != 100 || gotLocked != 150 {
		t.Errorf("boosted distribution: none=%d locked=%d want 100/150", gotNone, gotLocked)
	}
	if gotLocked*2 != gotNone*3 { // 150/100 == 1.5
		t.Errorf("LOCK_12M should get 1.5× the LOCK_NONE share: none=%d locked=%d", gotNone, gotLocked)
	}
	if gotNone+gotLocked != cplqAcquired {
		t.Errorf("conservation: %d+%d != %d", gotNone, gotLocked, cplqAcquired)
	}
}

// TestT2UnstakeLockGate rejects an unstake before lock_end_height and accepts
// it after.
func TestT2UnstakeLockGate(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	a := addr20(0x51)
	const amount uint64 = 5_000_000
	s.set(KeyForCPLQStake(a), mustMarshal(&contract.CPLQStake{
		Address: a, Amount: amount, StakedAtHeight: 1,
		LockTier: contract.LockTier_LOCK_3M, LockEndHeight: 1_000,
	}))
	s.set(KeyForCPLQStakeIndex(), mustMarshal(&contract.CPLQStakeIndex{Addresses: [][]byte{a}}))
	seedGlobals(s, &contract.CanoliqGlobals{TotalStakedCplq: amount})
	seedAccount(s, a, 1_000_000)

	// Before lock end → rejected.
	c.plugin.setHeight(500)
	if r := c.DeliverMessageCPLQUnstake(&contract.MessageCPLQUnstake{FromAddress: a, Amount: 1_000_000}, 10_000, params); r.Error == nil {
		t.Fatal("expected ErrStakeLocked before lock_end_height")
	} else if r.Error.Code != codeStakeLocked {
		t.Fatalf("expected codeStakeLocked, got %d", r.Error.Code)
	}

	// After lock end → accepted.
	c.plugin.setHeight(1_001)
	if r := c.DeliverMessageCPLQUnstake(&contract.MessageCPLQUnstake{FromAddress: a, Amount: 1_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("unstake after lock end: %v", r.Error)
	}
	if got := loadStake(t, s, a).Amount; got != amount-1_000_000 {
		t.Errorf("stake after unstake: got %d want %d", got, amount-1_000_000)
	}
}

// TestT2StakeLockOnlyStrengthens confirms a higher tier raises the record and
// adding LOCK_NONE to a locked record leaves the lock intact.
func TestT2StakeLockOnlyStrengthens(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	a := addr20(0x61)
	seedAccount(s, a, 1_000_000)
	seedCPLQ(s, a, 30_000_000)

	c.plugin.setHeight(100)
	// First stake: LOCK_NONE.
	if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 10_000_000, LockTier: contract.LockTier_LOCK_NONE}, 10_000, params); r.Error != nil {
		t.Fatalf("stake none: %v", r.Error)
	}
	// Strengthen to LOCK_12M.
	if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 10_000_000, LockTier: contract.LockTier_LOCK_12M}, 10_000, params); r.Error != nil {
		t.Fatalf("stake 12m: %v", r.Error)
	}
	st := loadStake(t, s, a)
	if st.LockTier != contract.LockTier_LOCK_12M {
		t.Errorf("tier after strengthen: got %v want LOCK_12M", st.LockTier)
	}
	wantEnd := uint64(100) + lockTierDurationBlocks(contract.LockTier_LOCK_12M)
	if st.LockEndHeight != wantEnd {
		t.Errorf("lock_end after strengthen: got %d want %d", st.LockEndHeight, wantEnd)
	}

	// Adding LOCK_NONE must not weaken the existing 12M lock.
	if r := c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 10_000_000, LockTier: contract.LockTier_LOCK_NONE}, 10_000, params); r.Error != nil {
		t.Fatalf("stake none again: %v", r.Error)
	}
	st = loadStake(t, s, a)
	if st.LockTier != contract.LockTier_LOCK_12M || st.LockEndHeight != wantEnd {
		t.Errorf("lock weakened by LOCK_NONE add: tier=%v end=%d", st.LockTier, st.LockEndHeight)
	}
	if st.Amount != 30_000_000 {
		t.Errorf("amount: got %d want 30_000_000", st.Amount)
	}
}
