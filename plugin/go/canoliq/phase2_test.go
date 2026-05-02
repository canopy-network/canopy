package canoliq

import (
	"bytes"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// phase2_test.go covers the governance, staking, buyback, treasury, and
// per-validator pro-rata behaviors added in Phase 2. The fakeStore harness
// from fakeplugin_test.go is shared with Phase 1 tests.

// seedParams writes the supplied CanoliqParams to state. Used to swap
// DefaultParams for a test-specific tuning (e.g., shorter voting period).
func seedParams(t *testing.T, c *Canoliq, params *contract.CanoliqParams) {
	t.Helper()
	if err := c.SaveParams(params); err != nil {
		t.Fatalf("save params: %v", err)
	}
}

// seedCLIQ stores liquid CLIQ at the per-address balance key.
func seedCLIQ(s *fakeStore, addr []byte, amount uint64) {
	s.set(KeyForCLIQBalance(addr), EncodeUint64(amount))
}

// seedGlobals merges fields onto whatever globals already exist in state.
func seedGlobals(s *fakeStore, g *contract.CanoliqGlobals) {
	bz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), bz)
}

func loadStake(t *testing.T, s *fakeStore, addr []byte) *contract.CLIQStake {
	t.Helper()
	bz := s.get(KeyForCLIQStake(addr))
	stake := new(contract.CLIQStake)
	if bz != nil {
		if err := contract.Unmarshal(bz, stake); err != nil {
			t.Fatalf("unmarshal stake: %v", err)
		}
	}
	return stake
}

func loadProposal(t *testing.T, s *fakeStore, id uint64) *contract.Proposal {
	t.Helper()
	bz := s.get(KeyForProposal(id))
	if bz == nil {
		return nil
	}
	p := new(contract.Proposal)
	if err := contract.Unmarshal(bz, p); err != nil {
		t.Fatalf("unmarshal proposal: %v", err)
	}
	return p
}

func loadProposalIndex(s *fakeStore) *contract.ProposalIndex {
	bz := s.get(KeyForProposalIndex())
	idx := new(contract.ProposalIndex)
	if bz != nil {
		_ = contract.Unmarshal(bz, idx)
	}
	return idx
}

func loadOrder(t *testing.T, s *fakeStore, proposalID uint64) *contract.BuybackOrder {
	t.Helper()
	bz := s.get(KeyForBuybackOrder(proposalID))
	if bz == nil {
		return nil
	}
	order := new(contract.BuybackOrder)
	if err := contract.Unmarshal(bz, order); err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	return order
}

func loadSpend(t *testing.T, s *fakeStore, id uint64) *contract.TreasurySpend {
	t.Helper()
	bz := s.get(KeyForTreasurySpend(id))
	if bz == nil {
		return nil
	}
	spend := new(contract.TreasurySpend)
	if err := contract.Unmarshal(bz, spend); err != nil {
		t.Fatalf("unmarshal spend: %v", err)
	}
	return spend
}

// === §3 staking ===

// TestCLIQStakeUnstakeClaim walks one staker through the lifecycle:
// liquid → stake → unstake → claim. Verifies CLIQStake balance, total
// staked tracker, unstake record, and that claim returns CLIQ to liquid
// only after mature_height has elapsed.
func TestCLIQStakeUnstakeClaim(t *testing.T) {
	c, s := newTestCanoliq()
	staker := addr20(0x21)
	const amount uint64 = 5_000_000
	seedParams(t, c, DefaultParams())
	seedAccount(s, staker, 100_000) // CNPY for fees
	seedCLIQ(s, staker, amount)

	// Stake 5 CLIQ.
	resp := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: staker, Amount: amount}, 10_000, DefaultParams())
	if resp.Error != nil {
		t.Fatalf("stake: %v", resp.Error)
	}
	stake := loadStake(t, s, staker)
	if stake.Amount != amount {
		t.Fatalf("stake amount: got %d want %d", stake.Amount, amount)
	}
	g := loadGlobals(t, s)
	if g.TotalStakedCliq != amount {
		t.Fatalf("total staked: got %d want %d", g.TotalStakedCliq, amount)
	}
	if liq := readCliq(s, staker); liq != 0 {
		t.Errorf("liquid CLIQ should be drained: got %d", liq)
	}

	// Set height past expected unstake mature.
	c.plugin.setHeight(10)

	// Unstake 5 CLIQ. mature_height = 10 + cliq_unstaking_blocks.
	resp = c.DeliverMessageCLIQUnstake(&contract.MessageCLIQUnstake{FromAddress: staker, Amount: amount}, 10_000, DefaultParams())
	if resp.Error != nil {
		t.Fatalf("unstake: %v", resp.Error)
	}
	g = loadGlobals(t, s)
	if g.TotalStakedCliq != 0 {
		t.Errorf("total staked post-unstake: got %d want 0", g.TotalStakedCliq)
	}
	// Unstake id was assigned 0 (first unstake → NextUnstakeId++ from 0).
	uBz := s.get(KeyForCLIQUnstaking(staker, 0))
	if uBz == nil {
		t.Fatal("unstake record missing")
	}
	unstake := new(contract.UnstakingCLIQ)
	if err := contract.Unmarshal(uBz, unstake); err != nil {
		t.Fatalf("unmarshal unstake: %v", err)
	}
	if unstake.Amount != amount {
		t.Errorf("unstake amount: got %d want %d", unstake.Amount, amount)
	}
	if unstake.MatureHeight != 10+DefaultParams().CliqUnstakingBlocks {
		t.Errorf("mature height: got %d", unstake.MatureHeight)
	}

	// Claim before mature → error.
	resp = c.DeliverMessageCLIQClaimUnstake(&contract.MessageCLIQClaimUnstake{FromAddress: staker, UnstakeId: 0}, 10_000, DefaultParams())
	if resp.Error == nil {
		t.Fatal("expected ErrUnstakeNotMature, got nil")
	}

	// Advance height past maturity, claim succeeds.
	c.plugin.setHeight(unstake.MatureHeight + 1)
	resp = c.DeliverMessageCLIQClaimUnstake(&contract.MessageCLIQClaimUnstake{FromAddress: staker, UnstakeId: 0}, 10_000, DefaultParams())
	if resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}
	if liq := readCliq(s, staker); liq != amount {
		t.Errorf("liquid CLIQ post-claim: got %d want %d", liq, amount)
	}
	if s.get(KeyForCLIQUnstaking(staker, 0)) != nil {
		t.Error("unstake record should be deleted after claim")
	}
}

// === §4 governance ===

// shortGovParams returns DefaultParams() tuned for fast test cycles.
func shortGovParams() *contract.CanoliqParams {
	p := DefaultParams()
	p.VotingPeriodBlocks = 5
	p.CliqUnstakingBlocks = 5
	p.QuorumBps = 3300
	p.PassThresholdBps = 5001
	p.MinStakeToPropose = 1_000_000
	return p
}

// TestProposalParamChangeRoundTrip drives a full proposal lifecycle for a
// param change: stake → propose → vote → tally → param applied.
func TestProposalParamChangeRoundTrip(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	proposer := addr20(0x31)
	seedAccount(s, proposer, 500_000)
	seedCLIQ(s, proposer, 10_000_000)

	// Stake 10 CLIQ.
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: proposer, Amount: 10_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("stake: %v", r.Error)
	}

	// Build a param-change payload that flips fee_bps from 1200 → 800.
	newParams := *params
	newParams.FeeBps = 800
	payload, err := anypb.New(&contract.ProposalParamChange{Params: &newParams})
	if err != nil {
		t.Fatalf("anypb new: %v", err)
	}
	c.plugin.setHeight(20)
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{
		FromAddress: proposer,
		Payload:     payload,
		Description: "lower fee to 8%",
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create proposal: %v", r.Error)
	}
	prop := loadProposal(t, s, 1)
	if prop == nil {
		t.Fatal("proposal not stored")
	}
	if prop.SnapshotTotalStaked != 10_000_000 {
		t.Errorf("snapshot: got %d want 10_000_000", prop.SnapshotTotalStaked)
	}
	if prop.ExpiryHeight != 25 {
		t.Errorf("expiry height: got %d want 25", prop.ExpiryHeight)
	}

	// Vote yes with full stake.
	if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{
		FromAddress: proposer,
		ProposalId:  1,
		Choice:      contract.VoteChoice_VOTE_YES,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("vote: %v", r.Error)
	}
	prop = loadProposal(t, s, 1)
	if prop.YesWeight != 10_000_000 {
		t.Errorf("yes weight: got %d want 10_000_000", prop.YesWeight)
	}

	// Re-vote rejected.
	if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{
		FromAddress: proposer,
		ProposalId:  1,
		Choice:      contract.VoteChoice_VOTE_YES,
	}, 10_000, params); r.Error == nil {
		t.Fatal("expected ErrAlreadyVoted")
	}

	// Advance past expiry; BeginBlock tally should pass + apply param change.
	c.plugin.setHeight(30)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: 30}); r.Error != nil {
		t.Fatalf("begin block: %v", r.Error)
	}
	if loadProposal(t, s, 1) != nil {
		t.Error("proposal should be deleted post-tally")
	}
	if idx := loadProposalIndex(s); len(idx.Ids) != 0 {
		t.Errorf("proposal index should be empty, got %v", idx.Ids)
	}
	// Verify params actually mutated.
	got, err2 := c.LoadParams()
	if err2 != nil {
		t.Fatalf("load params: %v", err2)
	}
	if got.FeeBps != 800 {
		t.Errorf("fee_bps after param change: got %d want 800", got.FeeBps)
	}
}

// TestVoteSnapshotRejectsLateStake confirms voters who stake AFTER proposal
// creation are rejected (zero weight isn't even tallied — the stake-after
// check rejects the tx).
func TestVoteSnapshotRejectsLateStake(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	proposer := addr20(0x41)
	flashStaker := addr20(0x42)
	seedAccount(s, proposer, 500_000)
	seedAccount(s, flashStaker, 500_000)
	seedCLIQ(s, proposer, 5_000_000)
	seedCLIQ(s, flashStaker, 5_000_000)
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: proposer, Amount: 5_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("proposer stake: %v", r.Error)
	}
	c.plugin.setHeight(10)
	payload, _ := anypb.New(&contract.ProposalParamChange{Params: shortGovParams()})
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{
		FromAddress: proposer,
		Payload:     payload,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create: %v", r.Error)
	}
	// Flash-stake AFTER proposal creation.
	c.plugin.setHeight(11)
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: flashStaker, Amount: 5_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("flash stake: %v", r.Error)
	}
	// Flash staker tries to vote — snapshot guard must reject.
	if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{
		FromAddress: flashStaker,
		ProposalId:  1,
		Choice:      contract.VoteChoice_VOTE_NO,
	}, 10_000, params); r.Error == nil {
		t.Fatal("expected ErrStakeAfterCreation")
	}
}

// TestProposalQuorumMiss confirms a proposal with too few participating votes
// fails on tally despite a yes-majority.
func TestProposalQuorumMiss(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	proposer := addr20(0x51)
	seedAccount(s, proposer, 500_000)
	// Globals show 100M total staked but only the proposer's 1M votes — far
	// below quorum.
	seedGlobals(s, &contract.CanoliqGlobals{TotalStakedCliq: 100_000_000})
	seedCLIQ(s, proposer, 1_000_000)
	if r := c.DeliverMessageCLIQStake(&contract.MessageCLIQStake{FromAddress: proposer, Amount: 1_000_000}, 10_000, params); r.Error != nil {
		t.Fatalf("stake: %v", r.Error)
	}
	c.plugin.setHeight(10)
	payload, _ := anypb.New(&contract.ProposalParamChange{Params: params})
	if r := c.DeliverMessageCLIQProposalCreate(&contract.MessageCLIQProposalCreate{
		FromAddress: proposer,
		Payload:     payload,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("create: %v", r.Error)
	}
	if r := c.DeliverMessageCLIQVote(&contract.MessageCLIQVote{
		FromAddress: proposer,
		ProposalId:  1,
		Choice:      contract.VoteChoice_VOTE_YES,
	}, 10_000, params); r.Error != nil {
		t.Fatalf("vote: %v", r.Error)
	}
	c.plugin.setHeight(20)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: 20}); r.Error != nil {
		t.Fatalf("begin block: %v", r.Error)
	}
	got, _ := c.LoadParams()
	// Param change must NOT apply; FeeBps stays at the seeded 1200.
	if got.FeeBps != 1200 {
		t.Errorf("fee_bps after failed proposal: got %d want 1200", got.FeeBps)
	}
}

// === §6 buyback ===

// TestBuybackBurnReducesSupply seeds a treasury_cliq and buyback pool, runs
// a passed BURN proposal end-to-end, and asserts that:
//   - cliq_total_supply and cliq_circulating_supply are decremented.
//   - buyback pool is drained.
//   - treasury_cliq is debited.
//   - treasury_cnpy is credited.
func TestBuybackBurnReducesSupply(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	const treasuryCLIQ uint64 = 1_000_000_000_000 // 1M CLIQ in uCLIQ — well above the buyback acquisition
	const buybackPoolAmt uint64 = 1_000_000_000
	s.set(KeyForTreasuryCLIQ(), EncodeUint64(treasuryCLIQ))
	s.set(KeyForBuybackPool(), EncodeUint64(buybackPoolAmt))
	seedGlobals(s, &contract.CanoliqGlobals{
		CliqTotalSupply:       100_000_000_000_000, // 100M CLIQ in uCLIQ
		CliqCirculatingSupply: 100_000_000_000_000,
	})

	// Seed a passed BuybackOrder directly (governance.dispatchPassed path is
	// covered by TestProposalParamChangeRoundTrip; here we focus on execution).
	const proposalID uint64 = 7
	const cnpyAmount uint64 = 500_000_000
	const price uint64 = 1_000_000 // 1 CNPY per CLIQ
	order := &contract.BuybackOrder{
		ProposalId: proposalID,
		Mode:       contract.BuybackMode_BUYBACK_BURN,
		Payload: &contract.ProposalBuyback{
			CnpyAmount:            cnpyAmount,
			PriceMicroCnpyPerCliq: price,
			Mode:                  contract.BuybackMode_BUYBACK_BURN,
		},
	}
	bz, _ := contract.Marshal(order)
	s.set(KeyForBuybackOrder(proposalID), bz)

	trigger := addr20(0x91)
	seedAccount(s, trigger, 100_000)
	resp := c.DeliverMessageBuybackExecute(&contract.MessageBuybackExecute{
		FromAddress: trigger,
		ProposalId:  proposalID,
	}, 10_000, params)
	if resp.Error != nil {
		t.Fatalf("execute: %v", resp.Error)
	}
	const cliqAcquired = cnpyAmount * 1_000_000 / price // = 500_000_000

	g := loadGlobals(t, s)
	if g.CliqTotalSupply != 100_000_000_000_000-cliqAcquired {
		t.Errorf("total supply: got %d want %d", g.CliqTotalSupply, 100_000_000_000_000-cliqAcquired)
	}
	if g.CliqCirculatingSupply != 100_000_000_000_000-cliqAcquired {
		t.Errorf("circulating supply: got %d want %d", g.CliqCirculatingSupply, 100_000_000_000_000-cliqAcquired)
	}
	if got := DecodeUint64(s.get(KeyForBuybackPool())); got != buybackPoolAmt-cnpyAmount {
		t.Errorf("buyback pool drain: got %d want %d", got, buybackPoolAmt-cnpyAmount)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != cnpyAmount {
		t.Errorf("treasury_cnpy credit: got %d want %d", got, cnpyAmount)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCLIQ())); got != treasuryCLIQ-cliqAcquired {
		// Note: with cliqAcquired = 500M and treasuryCLIQ = 50M, this fails —
		// adjust the seed.
		t.Errorf("treasury_cliq debit: got %d want %d", got, treasuryCLIQ-cliqAcquired)
	}

	// Idempotency: re-execute is rejected.
	resp = c.DeliverMessageBuybackExecute(&contract.MessageBuybackExecute{
		FromAddress: trigger,
		ProposalId:  proposalID,
	}, 10_000, params)
	if resp.Error == nil {
		t.Error("re-execute should be rejected")
	}
}

// TestBuybackDistributeStakers credits stakers pro-rata. Two stakers at
// 70 / 30 receive 70 / 30 of the acquired CLIQ.
func TestBuybackDistributeStakers(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	seedParams(t, c, params)
	a := addr20(0xa1)
	b := addr20(0xb1)
	const stakeA, stakeB uint64 = 7_000_000, 3_000_000
	s.set(KeyForCLIQStake(a), mustMarshal(&contract.CLIQStake{Address: a, Amount: stakeA, StakedAtHeight: 1}))
	s.set(KeyForCLIQStake(b), mustMarshal(&contract.CLIQStake{Address: b, Amount: stakeB, StakedAtHeight: 1}))
	idx := &contract.CLIQStakeIndex{Addresses: [][]byte{a, b}}
	s.set(KeyForCLIQStakeIndex(), mustMarshal(idx))
	seedGlobals(s, &contract.CanoliqGlobals{TotalStakedCliq: stakeA + stakeB})

	const treasuryCLIQ uint64 = 100_000_000
	const buybackPoolAmt uint64 = 100_000_000
	s.set(KeyForTreasuryCLIQ(), EncodeUint64(treasuryCLIQ))
	s.set(KeyForBuybackPool(), EncodeUint64(buybackPoolAmt))

	const proposalID uint64 = 11
	const cnpyAmount uint64 = 10_000_000
	const price uint64 = 1_000_000
	order := &contract.BuybackOrder{
		ProposalId: proposalID,
		Mode:       contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS,
		Payload: &contract.ProposalBuyback{
			CnpyAmount:            cnpyAmount,
			PriceMicroCnpyPerCliq: price,
			Mode:                  contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS,
		},
	}
	s.set(KeyForBuybackOrder(proposalID), mustMarshal(order))

	trigger := addr20(0xcc)
	seedAccount(s, trigger, 100_000)
	resp := c.DeliverMessageBuybackExecute(&contract.MessageBuybackExecute{
		FromAddress: trigger,
		ProposalId:  proposalID,
	}, 10_000, params)
	if resp.Error != nil {
		t.Fatalf("execute: %v", resp.Error)
	}
	cliqAcquired := cnpyAmount * 1_000_000 / price // 10M
	wantA := cliqAcquired * stakeA / (stakeA + stakeB)
	wantB := cliqAcquired - wantA // remainder credited to largest staker (a) — but mulDiv is exact here

	if got := readCliq(s, a); got != wantA {
		t.Errorf("staker A credit: got %d want %d", got, wantA)
	}
	if got := readCliq(s, b); got != wantB {
		t.Errorf("staker B credit: got %d want %d", got, wantB)
	}
	if got := readCliq(s, a) + readCliq(s, b); got != cliqAcquired {
		t.Errorf("conservation (A+B): got %d want %d", got, cliqAcquired)
	}
}

// === §7 treasury ===

// TestTreasurySpendBelowThreshold runs a sub-threshold spend through the
// proposal route and verifies it executes immediately (no timelock, no
// multisig).
func TestTreasurySpendBelowThreshold(t *testing.T) {
	c, s := newTestCanoliq()
	params := shortGovParams()
	params.TreasuryThreshold = 1_000_000_000
	seedParams(t, c, params)
	const treasury uint64 = 5_000_000_000
	s.set(KeyForTreasuryCNPY(), EncodeUint64(treasury))

	recipient := addr20(0x71)
	const amount uint64 = 500_000_000 // below threshold
	payload := &contract.ProposalTreasurySpend{
		Recipient:    recipient,
		Amount:       amount,
		Denomination: contract.SpendDenomination_SPEND_CNPY,
	}
	prop := &contract.Proposal{Id: 5, CreationHeight: 1, ExpiryHeight: 2}
	if err := c.queueTreasurySpend(prop, payload, params, 10); err != nil {
		t.Fatalf("queue spend: %v", err)
	}
	g := loadGlobals(t, s)
	spendID := g.NextSpendId
	spend := loadSpend(t, s, spendID)
	if spend.RequiresMultisig {
		t.Fatalf("below-threshold spend should not require multisig")
	}
	if spend.ExecutableHeight != 10 {
		t.Errorf("executable_height: got %d want 10 (no timelock)", spend.ExecutableHeight)
	}
	trigger := addr20(0x99)
	seedAccount(s, trigger, 100_000)
	c.plugin.setHeight(10)
	resp := c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{
		FromAddress: trigger,
		ProposalId:  prop.Id,
	}, 10_000, params)
	if resp.Error != nil {
		t.Fatalf("execute: %v", resp.Error)
	}
	if got := readAccount(s, recipient); got != amount {
		t.Errorf("recipient balance: got %d want %d", got, amount)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCNPY())); got != treasury-amount {
		t.Errorf("treasury debit: got %d want %d", got, treasury-amount)
	}
}

// TestTreasurySpendAboveThresholdRequiresTimelockAndMultisig drives an
// above-threshold spend through queue → reject pre-timelock → reject without
// approvals → execute once both met.
func TestTreasurySpendAboveThresholdRequiresTimelockAndMultisig(t *testing.T) {
	c, s := newTestCanoliq()
	signers := [][]byte{addr20(0xa0), addr20(0xa1), addr20(0xa2), addr20(0xa3), addr20(0xa4)}
	params := shortGovParams()
	params.TreasuryThreshold = 1_000_000_000
	params.TimelockBlocks = 100
	params.MultisigSigners = signers
	params.MultisigThreshold = 3
	seedParams(t, c, params)
	const treasury uint64 = 50_000_000_000
	s.set(KeyForTreasuryCNPY(), EncodeUint64(treasury))

	recipient := addr20(0x88)
	const amount uint64 = 5_000_000_000 // above threshold
	payload := &contract.ProposalTreasurySpend{
		Recipient:    recipient,
		Amount:       amount,
		Denomination: contract.SpendDenomination_SPEND_CNPY,
	}
	prop := &contract.Proposal{Id: 9, CreationHeight: 1, ExpiryHeight: 2}
	c.plugin.setHeight(10)
	if err := c.queueTreasurySpend(prop, payload, params, 10); err != nil {
		t.Fatalf("queue: %v", err)
	}
	g := loadGlobals(t, s)
	spendID := g.NextSpendId
	spend := loadSpend(t, s, spendID)
	if !spend.RequiresMultisig {
		t.Fatal("above-threshold spend must require multisig")
	}
	if spend.ExecutableHeight != 110 {
		t.Errorf("executable_height: got %d want 110 (10 + timelock 100)", spend.ExecutableHeight)
	}

	trigger := addr20(0x77)
	seedAccount(s, trigger, 1_000_000)

	// Pre-timelock: rejected.
	c.plugin.setHeight(50)
	resp := c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{
		FromAddress: trigger,
		ProposalId:  prop.Id,
	}, 10_000, params)
	if resp.Error == nil {
		t.Error("expected ErrSpendNotReady (pre-timelock)")
	}

	// Past timelock but no approvals: rejected.
	c.plugin.setHeight(120)
	resp = c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{
		FromAddress: trigger,
		ProposalId:  prop.Id,
	}, 10_000, params)
	if resp.Error == nil {
		t.Error("expected ErrSpendNotReady (no multisig)")
	}

	// Add 3 approvals from valid signers.
	for i := 0; i < 3; i++ {
		seedAccount(s, signers[i], 1_000_000)
		r := c.DeliverMessageMultisigApprove(&contract.MessageMultisigApprove{
			FromAddress: signers[i],
			SpendId:     spendID,
		}, 10_000, params)
		if r.Error != nil {
			t.Fatalf("approve %d: %v", i, r.Error)
		}
	}

	// Now executes.
	resp = c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{
		FromAddress: trigger,
		ProposalId:  prop.Id,
	}, 10_000, params)
	if resp.Error != nil {
		t.Fatalf("execute: %v", resp.Error)
	}
	if got := readAccount(s, recipient); got != amount {
		t.Errorf("recipient balance: got %d want %d", got, amount)
	}

	// Re-execute rejected.
	resp = c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{
		FromAddress: trigger,
		ProposalId:  prop.Id,
	}, 10_000, params)
	if resp.Error == nil {
		t.Error("re-execute should be rejected")
	}
}

// TestMultisigApprovalRejectsNonSigner verifies only configured signers can
// approve.
func TestMultisigApprovalRejectsNonSigner(t *testing.T) {
	c, s := newTestCanoliq()
	signers := [][]byte{addr20(0xa0)}
	params := shortGovParams()
	params.MultisigSigners = signers
	params.MultisigThreshold = 1
	seedParams(t, c, params)
	// Seed a TreasurySpend record directly so the approve handler has a target.
	spend := &contract.TreasurySpend{Id: 1, ProposalId: 99, RequiresMultisig: true, Payload: &contract.ProposalTreasurySpend{Recipient: addr20(0xff), Amount: 1, Denomination: contract.SpendDenomination_SPEND_CNPY}}
	s.set(KeyForTreasurySpend(1), mustMarshal(spend))

	rogue := addr20(0xee)
	seedAccount(s, rogue, 100_000)
	resp := c.DeliverMessageMultisigApprove(&contract.MessageMultisigApprove{
		FromAddress: rogue,
		SpendId:     1,
	}, 10_000, params)
	if resp.Error == nil {
		t.Fatal("expected ErrNotMultisigSigner")
	}
}

// === §5 per-validator pro-rata ===

// TestPerValidatorProRataDistribution exercises the new stake-weighted
// validator-incentive distribution. Three validators stake 70 / 20 / 10
// receive 70 / 20 / 10 of the validator-share credit, with rounding remainder
// attributed to the largest stake.
func TestPerValidatorProRataDistribution(t *testing.T) {
	c, s := newTestCanoliq()
	params := DefaultParams()
	seedParams(t, c, params)
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	seedGlobals(s, g)
	// Seed validator registry with 70/20/10 weights.
	v1, v2, v3 := addr20(0x01), addr20(0x02), addr20(0x03)
	registry := &contract.ValidatorRegistry{Entries: []*contract.ValidatorRegistryEntry{
		{Address: v1, Stake: 700},
		{Address: v2, Stake: 200},
		{Address: v3, Stake: 100},
	}}
	s.set(KeyForValidatorRegistry(), mustMarshal(registry))

	// Reward sweep with X=1000 → fee=120, validators=18.
	pool := &contract.Pool{Id: c.Config.ChainId, Amount: 1000}
	s.set(contract.KeyForFeePool(c.Config.ChainId), mustMarshal(pool))
	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}
	// Expected per-validator credits with rounding: 18*700/1000 = 12,
	// 18*200/1000 = 3, 18*100/1000 = 1; allocated 16, remainder 2 → largest (v1) gets 14.
	got1 := DecodeUint64(s.get(KeyForValidatorIncentives(v1)))
	got2 := DecodeUint64(s.get(KeyForValidatorIncentives(v2)))
	got3 := DecodeUint64(s.get(KeyForValidatorIncentives(v3)))
	if got1+got2+got3 != 18 {
		t.Errorf("conservation: %d+%d+%d=%d want 18", got1, got2, got3, got1+got2+got3)
	}
	if got1 < got2 || got2 < got3 {
		t.Errorf("ordering broken: %d %d %d", got1, got2, got3)
	}
	// Aggregator key should NOT have been used.
	if got := DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr()))); got != 0 {
		t.Errorf("aggregator should be zero with registry present, got %d", got)
	}
}

// === conservation including insurance ===

// TestInsuranceConservationFullSplit pins the full Phase 2 conservation
// equation: delta == userYield + treasury + insurance + buyback + validators.
func TestInsuranceConservationFullSplit(t *testing.T) {
	c, s := newTestCanoliq()
	g := &contract.CanoliqGlobals{GenesisComplete: true}
	seedGlobals(s, g)
	const X = 950 // post-DAO inflow
	pool := &contract.Pool{Id: c.Config.ChainId, Amount: X}
	s.set(contract.KeyForFeePool(c.Config.ChainId), mustMarshal(pool))

	if err := c.ProcessRewards(&contract.PluginEndRequest{Height: 1}); err != nil {
		t.Fatalf("process rewards: %v", err)
	}
	g2 := loadGlobals(t, s)
	yield := g2.TotalPooledCnpy
	treasury := DecodeUint64(s.get(KeyForTreasuryCNPY()))
	insurance := DecodeUint64(s.get(KeyForInsurancePool()))
	buyback := DecodeUint64(s.get(KeyForBuybackPool()))
	validators := DecodeUint64(s.get(KeyForValidatorIncentives(c.committeeAggregatorAddr())))
	total := yield + treasury + insurance + buyback + validators
	if total != X {
		t.Errorf("conservation: yield %d + treasury %d + insurance %d + buyback %d + validators %d = %d, want %d",
			yield, treasury, insurance, buyback, validators, total, X)
	}
	if insurance == 0 {
		t.Error("insurance should be non-zero with default insurance_bps=1500")
	}
}

func mustMarshal(m interface{ Reset() }) []byte {
	bz, err := contract.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bz
}

// keep bytes import live across compile gates that may strip helpers.
var _ = bytes.Equal
