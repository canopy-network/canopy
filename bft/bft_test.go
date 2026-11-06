package bft

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout = 500 * time.Millisecond
)

func TestStartElectionPhase(t *testing.T) {
	tests := []struct {
		name          string
		numValidators int
	}{
		{
			name:          "two validators",
			numValidators: 2,
		},
		{
			name:          "eight validators",
			numValidators: 8,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Election, test.numValidators)
			expected := SelectProposerWeightedRoundRobin(c.bft.View, c.valSet.ValidatorSet, c.valSet.TotalPower)

			c.bft.StartElectionPhase()

			require.Equal(t, expected, c.bft.ProposerKey)
			select {
			case <-c.cont.sendToReplicasChan:
				t.Fatal("deterministic election must not broadcast candidates")
			default:
			}
		})
	}
}

// TestLegacyVRFProposerModePreservesElection() verifies that legacy VRF mode retains the election phase.
func TestLegacyVRFProposerModePreservesElection(t *testing.T) {
	c := newTestConsensus(t, Election, 6)
	c.bft.Config.ProposerMode = lib.ProposerModeVRF

	go c.bft.StartElectionPhase()
	select {
	case signable := <-c.cont.sendToReplicasChan:
		message, ok := signable.(*Message)
		require.True(t, ok)
		require.NotNil(t, message.Vrf)
		require.NotNil(t, c.bft.SortitionData)
		require.Nil(t, c.bft.ProposerKey)
	case <-time.After(testTimeout):
		t.Fatal("legacy VRF candidate was not broadcast")
	}

	c.bft.StartRound()
	require.Equal(t, Election, c.bft.Phase)
	c.bft.PhaseTimer.Stop()
}

// TestRoundDeadlinesAreTimestampAnchoredAndBackOff() verifies anchored deadlines and round backoff.
func TestRoundDeadlinesAreTimestampAnchoredAndBackOff(t *testing.T) {
	c := newTestConsensus(t, Propose, 4)
	c.bft.Config.RoundRobinBlockIntervalMS = 750
	anchor := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	c.bft.SetHeightAnchor(anchor)

	require.Equal(t, anchor.Add(750*time.Millisecond), c.bft.RoundDeadline(0))
	require.Equal(t, anchor.Add(3*time.Second), c.bft.RoundDeadline(1))
	require.Equal(t, anchor.Add(6750*time.Millisecond), c.bft.RoundDeadline(2))
}

// TestForcedRoundDeadlineUsesRequestedTimeoutSchedule() verifies forced rounds use the requested timeout schedule.
func TestForcedRoundDeadlineUsesRequestedTimeoutSchedule(t *testing.T) {
	c := newTestConsensus(t, Propose, 1)
	c.bft.Round = 2
	c.bft.round.Store(2)
	// Reproduce recovery from a stale finalized timestamp; the old implementation derived round 7 from this anchor and expired it immediately.
	c.bft.SetHeightAnchor(time.Unix(1, 0))
	forceAt := time.Now()
	timeoutRound := uint64(0)
	require.NoError(t, c.bft.ScheduleForceRound(7, forceAt, &timeoutRound))
	select {
	case <-c.bft.forceRoundTimer.C:
	case <-time.After(testTimeout):
		t.Fatal("force-round timer did not fire")
	}
	newViewSent := make(chan struct{})
	go func() {
		<-c.cont.sendToProposerChan
		close(newViewSent)
	}()
	c.bft.applyForcedRound()
	<-newViewSent

	require.Equal(t, forceAt.Add(500*time.Millisecond), c.bft.roundDeadline)
	require.Equal(t, forceAt.Add(2*time.Second), c.bft.RoundDeadline(8))
	require.Equal(t, forceAt.Add(4500*time.Millisecond), c.bft.RoundDeadline(9))
}

// TestValidatedProposalExtendsDeadlineMonotonically() verifies validated proposals only extend deadlines.
func TestValidatedProposalExtendsDeadlineMonotonically(t *testing.T) {
	c := newTestConsensus(t, Propose, 4)
	anchor := time.Now().Add(-250 * time.Millisecond).Truncate(time.Microsecond)
	c.bft.SetHeightAnchor(anchor)
	interval := c.bft.Config.ConsensusConfig.RoundRobinBlockInterval()
	c.bft.roundDeadline = anchor.Add(interval)

	encode := func(at time.Time) []byte {
		bz, err := lib.Marshal(&lib.Block{BlockHeader: &lib.BlockHeader{Time: uint64(at.UnixMicro())}})
		require.NoError(t, err)
		return bz
	}

	c.bft.extendRoundDeadlineFromProposal(encode(anchor.Add(2 * interval)))
	require.Equal(t, anchor.Add(2*interval), c.bft.roundDeadline)

	c.bft.extendRoundDeadlineFromProposal(encode(anchor.Add(interval)))
	require.Equal(t, anchor.Add(2*interval), c.bft.roundDeadline,
		"an older proposal timestamp must not shorten the deadline")
}

// TestProposerAdoptsProducedBlockDeadlineBeforeBroadcast() verifies proposers adopt the produced block deadline before broadcasting.
func TestProposerAdoptsProducedBlockDeadlineBeforeBroadcast(t *testing.T) {
	c := newTestConsensus(t, Propose, 1)
	c.bft.ProposerKey = c.bft.PublicKey
	c.bft.roundDeadline = time.Now().Add(-time.Second)
	proposalTime := time.Now().Add(500 * time.Millisecond).Truncate(time.Microsecond)
	c.cont.proposalTime = uint64(proposalTime.UnixMicro())
	// Model the original round deadline already being selectable while proposal construction holds the controller lock.
	lib.ResetTimer(c.bft.PhaseTimer, 0)
	time.Sleep(time.Millisecond)

	go c.bft.StartProposePhase()
	select {
	case <-c.cont.sendToReplicasChan:
	case <-time.After(testTimeout):
		t.Fatal("proposal was not broadcast")
	}

	require.Equal(t, proposalTime, c.bft.roundDeadline)
	select {
	case <-c.bft.PhaseTimer.C:
		t.Fatal("expired round timer remained armed after proposal production")
	default:
	}
	c.bft.PhaseTimer.Stop()
}

// TestCommitteeResetCollectsRoundZeroNewViews() verifies committee resets collect round-zero new-view votes.
func TestCommitteeResetCollectsRoundZeroNewViews(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	c.cont.rootHeight = 1
	locked := c.setupTestableHighQC(t, false, true)
	c.bft.HighQC = locked
	c.bft.NewHeight(true)
	c.bft.SetHeightAnchor(time.Now())

	go c.bft.StartRound()
	select {
	case signable := <-c.cont.sendToProposerChan:
		msg := signable.(*Message)
		require.Equal(t, ElectionVote, msg.Qc.Header.Phase)
		require.Same(t, locked, msg.HighQc)
	case <-time.After(testTimeout):
		t.Fatal("committee reset did not send a round-zero new-view vote")
	}

	c.bft.ProposerKey = c.bft.PublicKey
	c.bft.StartProposePhase()
	select {
	case <-c.cont.sendToReplicasChan:
		t.Fatal("committee-reset proposer used the one-signer round-zero shortcut")
	default:
	}

	c.simElectionVotePhase(t, 0, false, false, false, 0)
	go c.bft.StartProposePhase()
	select {
	case signable := <-c.cont.sendToReplicasChan:
		msg := signable.(*Message)
		require.Same(t, locked, msg.HighQc)
		signers, _, err := msg.Qc.Signature.GetSigners(c.bft.ValidatorSet)
		require.NoError(t, err)
		require.Len(t, signers, 2)
	case <-time.After(testTimeout):
		t.Fatal("committee-reset proposer did not propose after new-view quorum")
	}
	c.bft.PhaseTimer.Stop()
}

// TestCommitteeResetDoesNotAcceptRoundZeroOneSignerProof() verifies committee resets reject a round-zero single-signer proof.
func TestCommitteeResetDoesNotAcceptRoundZeroOneSignerProof(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	c.bft.ProposerKey = c.bft.PublicKey
	vote, signature, err := c.bft.roundZeroProposerProof()
	require.NoError(t, err)
	block, blockHash, results, resultsHash := c.proposal(t)
	msg := &Message{
		Header: c.view(Propose),
		Qc: &QC{
			Header:      vote.Qc.Header,
			Block:       block,
			BlockHash:   blockHash,
			Results:     results,
			ResultsHash: resultsHash,
			ProposerKey: c.bft.PublicKey,
			Signature:   signature,
		},
	}
	require.NoError(t, msg.Sign(c.bft.PrivateKey))
	c.bft.roundZeroNewView = true
	params, errI := c.bft.GetValidateMessageParams(msg)
	require.NoError(t, errI)
	partial, errI := c.bft.CheckProposerMessage(msg, params)
	require.NoError(t, errI)
	require.True(t, partial)

	c.bft.roundZeroNewView = false
	params, errI = c.bft.GetValidateMessageParams(msg)
	require.NoError(t, errI)
	partial, errI = c.bft.CheckProposerMessage(msg, params)
	require.NoError(t, errI)
	require.False(t, partial)
}

// TestTimeoutQuorumRequiresTwoThirdsInOneExactRound() verifies timeout quorums cannot combine votes from different rounds.
func TestTimeoutQuorumRequiresTwoThirdsInOneExactRound(t *testing.T) {
	c := newTestConsensus(t, Propose, 4)
	add := func(validator, round int) {
		msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, uint64(round))}}
		require.NoError(t, msg.Sign(c.valKeys[validator]))
		key := fmt.Sprintf("%d/%s", round, lib.BytesToString(msg.Signature.PublicKey))
		c.bft.PacemakerMessages[key] = msg
	}

	add(0, 0)
	add(1, 0)
	add(2, 1)
	add(3, 1)
	_, ok := c.bft.TimeoutQuorumRound()
	require.False(t, ok, "votes from different rounds must not be combined")

	add(2, 0)
	round, ok := c.bft.TimeoutQuorumRound()
	require.True(t, ok)
	require.Equal(t, uint64(0), round)
}

// TestTimeoutCatchUpRequiresDistinctOneThirdPower() verifies catch-up evidence requires distinct one-third voting power.
func TestTimeoutCatchUpRequiresDistinctOneThirdPower(t *testing.T) {
	c := newTestConsensus(t, RoundInterrupt, 4)
	add := func(validator int, round uint64) {
		msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, round)}}
		require.NoError(t, msg.Sign(c.valKeys[validator]))
		key := fmt.Sprintf("%d/%s", round, lib.BytesToString(msg.Signature.PublicKey))
		c.bft.PacemakerMessages[key] = msg
	}

	add(0, 3)
	add(0, 4)
	_, ok := c.bft.TimeoutCatchUpRound()
	require.False(t, ok, "one validator must not be counted once per round")

	add(1, 3)
	round, ok := c.bft.TimeoutCatchUpRound()
	require.True(t, ok)
	require.Equal(t, uint64(3), round)
}

// TestTimeoutCatchUpHealsAdjacentRoundSplit() verifies catch-up evidence heals validators split across adjacent rounds.
func TestTimeoutCatchUpHealsAdjacentRoundSplit(t *testing.T) {
	c := newTestConsensus(t, RoundInterrupt, 2)
	msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, 1)}}
	require.NoError(t, msg.Sign(c.valKeys[1]))
	key := fmt.Sprintf("1/%s", lib.BytesToString(msg.Signature.PublicKey))
	c.bft.PacemakerMessages[key] = msg

	round, ok := c.bft.TimeoutCatchUpRound()
	require.True(t, ok)
	require.Equal(t, uint64(1), round,
		"a round-zero validator must be able to join the round-one validator")
}

func TestStartElectionVotePhase(t *testing.T) {
	tests := []struct {
		name          string
		numValidators int
		hasBE         bool
	}{
		{
			name:          "three validators",
			numValidators: 3,
		},
		{
			name:          "eight validators",
			numValidators: 8,
		},
		{
			name:          "byzantine evidence not nil",
			numValidators: 5,
			hasBE:         true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, ElectionVote, test.numValidators)
			expectedProposer := SelectProposerWeightedRoundRobin(c.bft.View, c.valSet.ValidatorSet, c.valSet.TotalPower)
			c.bft.ProposerKey = expectedProposer
			expectedDoubleSignEvidence := DoubleSignEvidences{}
			if test.hasBE {
				expectedDoubleSignEvidence.Evidence = c.newTestDoubleSignEvidence(t)
				c.bft.ByzantineEvidence = &ByzantineEvidence{
					DSE: expectedDoubleSignEvidence,
				}
			}
			expectedView := lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				RootHeight: 1,
				Round:      0,
				Phase:      ElectionVote,
			}
			go c.bft.StartElectionVotePhase()
			select {
			case <-time.After(testTimeout):
				t.Fatal("timeout")
			case m := <-c.cont.sendToProposerChan:
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.NotNil(t, msg.Qc)
				require.Equal(t, *msg.Qc.Header, expectedView)
				require.Equal(t, expectedProposer, msg.Qc.ProposerKey)
				if test.hasBE {
					require.NotNil(t, msg.LastDoubleSignEvidence)
					require.Equal(t, expectedDoubleSignEvidence.Evidence, msg.LastDoubleSignEvidence)
				}
			}
		})
	}
}

func TestStartProposePhase(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		receiveEVQC    bool
		hasBE          bool
		hasLivenessHQC bool
		hasSafetyHQC   bool
	}{
		{
			name:   "round zero deterministic proposer",
			detail: `no network new-view quorum certificate required`,
		},
		{
			name:        "self is proposer",
			detail:      `election vote quorum certificate received`,
			receiveEVQC: true,
		},
		{
			name:        "self is proposer with byzantine evidence",
			detail:      `election vote quorum certificate received and there was byzantine evidence attached to the QC`,
			receiveEVQC: true,
			hasBE:       true,
		},
		{
			name:           "self is proposer with existing liveness highQC", // liveness vs safety doesn't matter for the proposer, a valid precommit qc from the same height is a lock
			detail:         `election vote quorum certificate received and there was a highQC (passes liveness rule) attached to the QC`,
			receiveEVQC:    true,
			hasLivenessHQC: true,
		},
		{
			name:         "self is proposer with existing safety highQC", // liveness vs safety doesn't matter for the proposer, a valid precommit qc from the same height is a lock
			detail:       `election vote quorum certificate received and there was a highQC (passes safety rule) attached to the QC`,
			receiveEVQC:  true,
			hasSafetyHQC: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, Propose, 3)
			c.bft.ProposerKey = c.bft.PublicKey
			var multiKey crypto.MultiPublicKeyI
			if test.receiveEVQC {
				multiKey = c.simElectionVotePhase(t, 0, test.hasBE, test.hasLivenessHQC, test.hasSafetyHQC, 0)
			}
			expectedView, expectedQCView := lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				Round:      0,
				Phase:      Propose,
				RootHeight: 1,
			}, lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				RootHeight: 1,
				Round:      0,
				Phase:      ElectionVote,
			}
			go c.bft.StartProposePhase()
			select {
			case <-time.After(testTimeout):
				t.Fatal("timeout")
			case m := <-c.cont.sendToReplicasChan:
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.Equal(t, expectedView, *msg.Header)
				require.NotNil(t, msg.Qc)
				require.Equal(t, expectedQCView.Phase, msg.Qc.Header.Phase)
				_, block, results, e := c.cont.ProduceProposal(nil, nil)
				require.NoError(t, e)
				require.Equal(t, block, msg.Qc.Block)
				require.Equal(t, results.Hash(), msg.Qc.Results.Hash())
				if test.receiveEVQC {
					expectedAggSig, err := multiKey.AggregateSignatures()
					require.NoError(t, err)
					require.Equal(t, expectedAggSig, msg.Qc.Signature.Signature)
				} else {
					signers, _, err := msg.Qc.Signature.GetSigners(c.bft.ValidatorSet)
					require.NoError(t, err)
					require.Equal(t, [][]byte{c.bft.PublicKey}, signers)
				}
				if test.hasBE {
					expectedDSE := c.newTestDoubleSignEvidence(t)
					require.EqualExportedValues(t, expectedDSE[0].VoteA, msg.LastDoubleSignEvidence[0].VoteA)
					require.EqualExportedValues(t, expectedDSE[0].VoteB, msg.LastDoubleSignEvidence[0].VoteB)
				}
				if test.hasLivenessHQC || test.hasSafetyHQC {
					require.Equal(t, msg.Qc.BlockHash, c.setupTestableHighQC(t, test.hasLivenessHQC, true).BlockHash)
					require.Equal(t, msg.Qc.ResultsHash, c.setupTestableHighQC(t, test.hasLivenessHQC, true).ResultsHash)
				}
			}
		})
	}
}

func TestStartProposeVotePhase(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		safetyLocked   bool
		livenessLocked bool
		invalidHighQC  bool
		validProposal  bool
		hasBE          bool
	}{
		{
			name:          "proposal is valid",
			detail:        `replica received a valid proposal from the leader`,
			validProposal: true,
		},
		{
			name:   "proposal is invalid",
			detail: `replica received an invalid proposal from the leader`,
		},
		{
			name:          "proposal is valid with BE",
			detail:        `proposal is valid and there's byzantine evidence attached to the message`,
			validProposal: true,
			hasBE:         true,
		},
		{
			name:          "proposal is valid and replica is locked",
			detail:        `a locked replica received a valid proposal from the leader, lock is bypassed using safety`,
			validProposal: true,
			safetyLocked:  true,
		},
		{
			name:           "proposal is valid and replica is locked",
			detail:         `a locked replica received a valid proposal from the leader, lock is bypassed using liveness`,
			validProposal:  true,
			livenessLocked: true,
		},
		{
			name:          "replica is locked and highQC doesn't pass safety or liveness",
			detail:        `a locked replica received a valid proposal with an invalid highQC justification, lock is not bypassed`,
			invalidHighQC: true,
			validProposal: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, ProposeVote, 3)
			highQC, be := (*QC)(nil), ByzantineEvidence{}
			if test.safetyLocked || test.livenessLocked || test.invalidHighQC {
				highQC = c.setupTestableHighQC(t, test.livenessLocked, !test.invalidHighQC)
			}
			if test.hasBE {
				be.DSE.Evidence = c.newTestDoubleSignEvidence(t)
			}
			_, results := c.simProposePhase(t, 0, test.validProposal, be, highQC, 0)
			expectedView := lib.View{
				Height:     1,
				Round:      0,
				Phase:      ProposeVote,
				RootHeight: 1,
				NetworkId:  1,
				ChainId:    lib.CanopyChainId,
			}
			// valid proposal
			go c.bft.StartProposeVotePhase()
			select {
			case <-time.After(testTimeout):
				if test.validProposal && !test.invalidHighQC {
					t.Fatal("timeout")
				}
			case m := <-c.cont.sendToProposerChan:
				if !test.validProposal || test.invalidHighQC {
					t.Fatal("unexpected message")
				}
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.NotNil(t, msg.Qc)
				require.Equal(t, *msg.Qc.Header, expectedView)
				require.Equal(t, c.cont.NewTestBlockHash(), msg.Qc.BlockHash)
				require.Equal(t, results.Hash(), msg.Qc.ResultsHash)
				if test.hasBE {
					require.NotNil(t, be.DSE)
				}
				require.Equal(t, c.bft.ByzantineEvidence.DSE.Evidence, be.DSE.Evidence)
			}
		})
	}
}

func TestStartPrecommitPhase(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		has23MajPropVote bool
		isProposer       bool
	}{
		{
			name:   "not proposer",
			detail: `self is not the proposer`,
		},
		{
			name:       "didn't received +2/3 prop vote",
			detail:     `did not receive +2/3 quorum on the propose votes from replicas`,
			isProposer: true,
		},
		{
			name:             "received +2/3 prop vote",
			detail:           `received +2/3 quorum on the propose votes from replicas`,
			isProposer:       true,
			has23MajPropVote: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c, multiKey, blockHash, resultsHash := newTestConsensus(t, Precommit, 3), crypto.MultiPublicKeyI(nil), []byte(nil), []byte(nil)
			expectedView, expectedQCView := lib.View{
				Height:     1,
				Round:      0,
				Phase:      Precommit,
				RootHeight: 1,
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
			}, lib.View{
				Height:     1,
				Round:      0,
				Phase:      ProposeVote,
				RootHeight: 1,
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
			}
			if test.has23MajPropVote {
				multiKey, blockHash, resultsHash = c.simProposeVotePhase(t, test.isProposer, true, 0)
			}
			go c.bft.StartPrecommitPhase()
			select {
			case <-time.After(testTimeout):
				if test.has23MajPropVote {
					t.Fatal("timeout")
				}
			case m := <-c.cont.sendToReplicasChan:
				if !test.has23MajPropVote {
					t.Fatal("unexpected message")
				}
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.NotNil(t, msg.Qc)
				require.Equal(t, msg.Qc.Header.Phase, expectedQCView.Phase)
				require.Equal(t, *msg.Header, expectedView)
				require.Equal(t, blockHash, msg.Qc.BlockHash)
				require.Equal(t, resultsHash, msg.Qc.ResultsHash)
				expectedAggSig, err := multiKey.AggregateSignatures()
				require.NoError(t, err)
				require.Equal(t, expectedAggSig, msg.Qc.Signature.Signature)
			}
		})
	}
}

func TestStartPrecommitVotePhase(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		proposalReceived bool
		validProposal    bool
		isProposer       bool
	}{
		{
			name:   "no proposal received",
			detail: `no proposal was received`,
		},
		{
			name:             "sender not proposer",
			detail:           `sender is not the set proposer`,
			proposalReceived: true,
		},
		{
			name:             "proposer sent invalid proposal",
			detail:           `the proposer sent a proposal that did not correspond with the block set in the propose phase`,
			proposalReceived: true,
			isProposer:       true,
		},
		{
			name:             "received +2/3 prop vote",
			detail:           `received +2/3 quorum on the propose votes from replicas`,
			proposalReceived: true,
			isProposer:       true,
			validProposal:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, PrecommitVote, 3)
			var (
				block   []byte
				results *lib.CertificateResult
			)
			if test.proposalReceived {
				block, results = c.simPrecommitPhase(t, 0)
			}
			if !test.validProposal {
				c.bft.Block = c.cont.NewTestBlock2() // mismatched proposals
				c.bft.BlockHash = c.cont.NewTestBlockHash2()
			}
			if !test.isProposer {
				c.bft.ProposerKey = []byte("some other proposer")
			}
			expectedView := lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				Round:      0,
				RootHeight: 1,
				Phase:      PrecommitVote,
			}
			go c.bft.StartPrecommitVotePhase()
			select {
			case <-time.After(testTimeout):
				if test.validProposal {
					t.Fatal("timeout")
				}
			case m := <-c.cont.sendToProposerChan:
				if !test.validProposal {
					t.Fatal("unexpected message received")
				}
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.NotNil(t, msg.Qc)
				require.Equal(t, *msg.Qc.Header, expectedView)
				require.Equal(t, c.cont.NewTestBlockHash(), msg.Qc.BlockHash)
				require.Equal(t, results.Hash(), msg.Qc.ResultsHash)
				require.Equal(t, c.bft.HighQC.BlockHash, msg.Qc.BlockHash)
				require.Equal(t, c.bft.HighQC.Block, block)
				require.Equal(t, c.bft.HighQC.ResultsHash, msg.Qc.ResultsHash)
			}
		})
	}
}

func TestStartCommitPhase(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		has23MajPropVote bool
		isProposer       bool
	}{
		{
			name:   "not proposer",
			detail: `self is not the proposer`,
		},
		{
			name:       "didn't received +2/3 prop vote",
			detail:     `did not receive +2/3 quorum on the propose votes from replicas`,
			isProposer: true,
		},
		{
			name:             "received +2/3 prop vote",
			detail:           `received +2/3 quorum on the propose votes from replicas`,
			isProposer:       true,
			has23MajPropVote: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, Commit, 3)
			multiKey, blockHash, resultsHash := crypto.MultiPublicKeyI(nil), []byte(nil), []byte(nil)
			expectedView, expectedQCView := lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				Round:      0,
				RootHeight: 1,
				Phase:      Commit,
			}, lib.View{
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Height:     1,
				Round:      0,
				RootHeight: 1,
				Phase:      PrecommitVote,
			}
			if !test.isProposer {
				c.bft.ProposerKey = []byte("some other proposer")
			}
			if test.has23MajPropVote {
				multiKey, blockHash, resultsHash = c.simPrecommitVotePhase(t, 0)
				_, c.bft.Block, c.bft.Results, _ = c.cont.ProduceProposal(nil, nil)
			}
			go c.bft.StartCommitPhase()
			// receive the commit message
			select {
			case <-time.After(testTimeout):
				if test.has23MajPropVote {
					t.Fatal("timeout")
				}
			case m := <-c.cont.sendToReplicasChan:
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.NotNil(t, msg.Qc)
				require.Equal(t, msg.Qc.Header.Phase, expectedQCView.Phase)
				require.Equal(t, *msg.Header, expectedView)
				require.Equal(t, blockHash, msg.Qc.BlockHash)
				require.Equal(t, resultsHash, msg.Qc.ResultsHash)
				expectedAggSig, err := multiKey.AggregateSignatures()
				require.NoError(t, err)
				require.Equal(t, expectedAggSig, msg.Qc.Signature.Signature)
			}
		})
	}
}

func TestStartCommitProcessPhase(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		proposalReceived bool
		validProposal    bool
		isProposer       bool
		hasPartialQCDSE  bool
		hasEVDSE         bool
	}{
		{
			name:   "no proposal received",
			detail: `no proposal was received`,
		},
		{
			name:             "sender not proposer",
			detail:           `sender is not the set proposer`,
			proposalReceived: true,
		},
		{
			name:             "proposer sent invalid proposal",
			detail:           `the proposer sent a proposal that did not correspond with the block set in the propose phase`,
			proposalReceived: true,
			isProposer:       true,
		},
		{
			name:             "received +2/3 prop vote",
			detail:           `received +2/3 quorum on the precommit votes from replicas`,
			proposalReceived: true,
			isProposer:       true,
			validProposal:    true,
		},
		{
			name:             "received +2/3 prop vote and has partial qc DSE stored",
			detail:           `received +2/3 quorum on the precommit votes from replicas and has double sign evidence stored in the form of a conflicting partial qc`,
			proposalReceived: true,
			isProposer:       true,
			validProposal:    true,
			hasPartialQCDSE:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, CommitProcess, 3)
			if test.hasPartialQCDSE {
				c.simPrecommitPhase(t, 1)
				c.newPartialQCDoubleSign(t, Precommit)
			}
			if test.hasEVDSE {
				c.simProposePhase(t, 1, true, ByzantineEvidence{}, nil, 1)
				c.newElectionVoteDoubleSign(t)
			}
			c.bft.Round++
			multiKey, block, results := crypto.MultiPublicKeyI(nil), []byte(nil), &lib.CertificateResult{}
			if !test.isProposer {
				c.bft.ProposerKey = []byte("some other proposer")
			}
			if !test.validProposal {
				c.bft.Block = []byte("some other proposal")
			}
			if test.proposalReceived {
				multiKey, block, results = c.simCommitPhase(t, 1, 1)
			}
			expectedQCView := lib.View{
				Height:     1,
				Round:      1,
				RootHeight: 1,
				NetworkId:  lib.CanopyMainnetNetworkId,
				ChainId:    lib.CanopyChainId,
				Phase:      PrecommitVote,
			}
			if test.hasEVDSE {
				c.bft.ProposerKey = c.valKeys[1].PublicKey().Bytes()
			}
			go c.bft.StartCommitProcessPhase()
			select {
			case <-time.After(testTimeout):
				if test.validProposal {
					t.Fatal("timeout")
				}
			case qc := <-c.cont.gossipCertChan:
				require.Equal(t, qc.Header.Phase, expectedQCView.Phase)
				require.Equal(t, qc.Block, block)
				require.Equal(t, c.cont.NewTestBlockHash(), qc.BlockHash)
				require.Equal(t, results.Hash(), qc.ResultsHash)
				require.NotNil(t, qc.Signature)
				expectedAggSig, err := multiKey.AggregateSignatures()
				require.NoError(t, err)
				require.Equal(t, expectedAggSig, qc.Signature.Signature)
				if test.hasPartialQCDSE || test.hasEVDSE {
					require.Len(t, c.bft.ByzantineEvidence.DSE.Evidence, 1)
				}
			}
		})
	}
}

func TestRoundInterrupt(t *testing.T) {
	// setup
	c := newTestConsensus(t, Propose, 3)
	c.bft.Config.RoundInterruptTimeoutMS = 123
	c.bft.Round = 2
	go c.bft.RoundInterrupt()
	select {
	case <-time.After(testTimeout):
		t.Fatal("timeout")
	case m := <-c.cont.sendToReplicasChan:
		msg, ok := m.(*Message)
		require.True(t, ok)
		require.EqualExportedValues(t, msg.Qc.Header, &lib.View{
			NetworkId:  lib.CanopyMainnetNetworkId,
			Height:     1,
			Round:      2,
			RootHeight: 1,
			ChainId:    lib.CanopyChainId,
			Phase:      RoundInterrupt,
		})
		require.Equal(t, c.bft.Phase, RoundInterrupt)
		require.Equal(t, 123, c.bft.Config.RoundInterruptTimeoutMS,
			"round interrupt must not overwrite the configured pacemaker window")
		require.Equal(t, 123*time.Millisecond, c.bft.WaitTime(RoundInterrupt, c.bft.Round))
	}
}

func TestPacemaker(t *testing.T) {
	tests := []struct {
		name                   string
		detail                 string
		hasPeerPacemakerVotes  bool
		has13MajVote           bool
		expectedPacemakerRound uint64
	}{
		{
			name:                   "no peer pacemaker votes",
			detail:                 "no peer pacemaker votes received, simply increment round",
			expectedPacemakerRound: 1,
		},
		{
			name:                   "received peer pacemaker votes",
			detail:                 "peer pacemaker votes received, highest +2/3 at round 1",
			hasPeerPacemakerVotes:  true,
			expectedPacemakerRound: 1,
		},
		{
			name:                   "received peer pacemaker votes which caused a round fast forward",
			detail:                 "peer pacemaker votes received, highest +2/3 at round 3",
			hasPeerPacemakerVotes:  true,
			has13MajVote:           true,
			expectedPacemakerRound: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			numValidators := 1
			if test.has13MajVote {
				numValidators = 2
			}
			c := newTestConsensus(t, Propose, numValidators)
			c.bft.Config.ProposerMode = lib.ProposerModeVRF
			if test.hasPeerPacemakerVotes {
				c.simPacemakerPhase(t)
			}
			c.bft.Pacemaker()
			require.Equal(t, test.expectedPacemakerRound, c.bft.Round)
		})
	}
}

func TestPacemakerRequiresMoreThanOneThird(t *testing.T) {
	c := newTestConsensus(t, Propose, 2)
	c.bft.Config.ProposerMode = lib.ProposerModeVRF
	c.bft.ValidatorSet.ValidatorSet.ValidatorSet[0].VotingPower = 67
	c.bft.ValidatorSet.ValidatorSet.ValidatorSet[1].VotingPower = 33
	c.bft.ValidatorSet.TotalPower = 100
	msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, 3)}}
	require.NoError(t, msg.Sign(c.valKeys[1]))
	require.NoError(t, c.bft.HandleMessage(msg))

	c.bft.Pacemaker()
	require.Equal(t, uint64(1), c.bft.Round)
}

func TestScheduleForceRound(t *testing.T) {
	c := newTestConsensus(t, Propose, 1)
	c.bft.Round = 2
	c.bft.round.Store(2)
	lockedQC := &QC{}
	c.bft.HighQC = lockedQC

	timeoutRound := uint64(0)
	require.NoError(t, c.bft.ScheduleForceRound(7, time.Now().Add(time.Hour), &timeoutRound))
	c.bft.applyForcedRound()
	require.Equal(t, Propose, c.bft.Phase)
	require.Equal(t, uint64(2), c.bft.Round)

	require.NoError(t, c.bft.ScheduleForceRound(7, time.Now(), &timeoutRound))
	select {
	case <-c.bft.forceRoundTimer.C:
	case <-time.After(testTimeout):
		t.Fatal("force-round timer did not fire")
	}
	newViewSent := make(chan struct{})
	go func() {
		<-c.cont.sendToProposerChan
		close(newViewSent)
	}()
	c.bft.applyForcedRound()
	<-newViewSent
	require.Equal(t, uint64(7), c.bft.CurrentRound())
	require.Equal(t, Propose, c.bft.Phase)
	require.Same(t, lockedQC, c.bft.HighQC)
	require.Equal(t, time.Duration(c.bft.Config.ElectionTimeoutMS)*time.Millisecond, c.bft.WaitTime(Election, c.bft.Round+1))
	require.Error(t, c.bft.ScheduleForceRound(7, time.Now(), nil))
	c.bft.NewHeight()
	require.Nil(t, c.bft.forcedTimeoutRound)
	c.bft.PhaseTimer.Stop()
}

// TestRefreshRootChainInfoPublishesCommitteeSnapshotAtomically() verifies root-chain committee snapshots are published atomically.
func TestRefreshRootChainInfoPublishesCommitteeSnapshotAtomically(t *testing.T) {
	c := newTestConsensus(t, Propose, 1)
	oldValidators := c.bft.ValidatorSet.ValidatorSet
	oldCommitteeData := &lib.CommitteeData{LastRootHeightUpdated: 7}
	c.bft.RootHeight = 7
	c.bft.CommitteeData = oldCommitteeData

	newValidators, _, _ := newTestValSet(t, 2)
	newCommitteeData := &lib.CommitteeData{LastRootHeightUpdated: 8}
	c.cont.rootHeight = 8
	c.cont.valSet[lib.CanopyChainId] = newValidators
	c.cont.committeeData = newCommitteeData

	c.cont.loadCommitteeErr = lib.ErrNoValidators()
	c.bft.RefreshRootChainInfo()
	require.Equal(t, uint64(7), c.bft.RootHeight)
	require.Same(t, oldValidators, c.bft.ValidatorSet.ValidatorSet)
	require.Same(t, oldCommitteeData, c.bft.CommitteeData)

	c.cont.loadCommitteeErr = nil
	c.cont.loadCommitteeDataErr = lib.ErrNoValidators()
	c.bft.RefreshRootChainInfo()
	require.Equal(t, uint64(7), c.bft.RootHeight)
	require.Same(t, oldValidators, c.bft.ValidatorSet.ValidatorSet)
	require.Same(t, oldCommitteeData, c.bft.CommitteeData)

	c.cont.loadCommitteeDataErr = nil
	c.bft.RefreshRootChainInfo()
	require.Equal(t, uint64(8), c.bft.RootHeight)
	require.Same(t, newValidators.ValidatorSet, c.bft.ValidatorSet.ValidatorSet)
	require.Same(t, newCommitteeData, c.bft.CommitteeData)
}

func TestPhaseHas23Maj(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	// 75% votes
	c.simElectionVotePhase(t, 0, false, false, false, 0)
	require.True(t, c.bft.PhaseHas23Maj())
	// 50% votes
	c.simProposeVotePhase(t, false, false, 0)
	c.bft.Phase = Precommit
	require.False(t, c.bft.PhaseHas23Maj())
	// 0% votes
	c.bft.Phase = Commit
	require.False(t, c.bft.PhaseHas23Maj())
}

func TestCheckProposerAndBlock(t *testing.T) {
	tests := []struct {
		name          string
		detail        string
		validProposer bool
		validProposal bool
	}{
		{
			name:          "valid proposer and proposal",
			detail:        "message has a proposer and proposal that corresponds to the local state",
			validProposer: true,
			validProposal: true,
		},
		{
			name:          "valid proposer and invalid proposal",
			detail:        "message has a proposer that corresponds to the local state but the proposal does not",
			validProposer: true,
		},
		{
			name:          "valid proposal and invalid proposer",
			detail:        "message has a proposal that corresponds to the local state but the proposer does not",
			validProposal: true,
		},
		{
			name:   "invalid proposal and invalid proposer",
			detail: "message doesn't have a proposal nor a proposer that corresponds to the local state",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Precommit, 1)
			c.bft.Block, c.bft.ProposerKey, c.bft.Results = c.cont.NewTestBlock(), bytes.Repeat([]byte("F"), crypto.BLS12381PubKeySize), &lib.CertificateResult{
				RewardRecipients: &lib.RewardRecipients{
					PaymentPercents: []*lib.PaymentPercents{{
						Address: crypto.Hash([]byte("some address"))[:20],
						Percent: 100,
						ChainId: lib.CanopyChainId,
					}},
				},
			}
			messageHash, messageProposer := c.cont.NewTestBlockHash2(), []byte("some other proposer")
			if test.validProposal {
				messageHash = c.cont.NewTestBlockHash()
			}
			if test.validProposer {
				messageProposer = c.bft.ProposerKey
			}
			msg := &Message{
				Qc: &lib.QuorumCertificate{
					BlockHash:   messageHash,
					ResultsHash: c.bft.Results.Hash(),
				},
				Signature: &lib.Signature{
					PublicKey: messageProposer,
				},
			}
			require.Equal(t, !(test.validProposer && test.validProposal), c.bft.CheckProposerAndProposal(msg))
		})
	}
}

func TestNewRound(t *testing.T) {
	tests := []struct {
		name      string
		detail    string
		newRound  bool
		newHeight bool
	}{
		{
			name:   "not new round",
			detail: "new round was not called",
		},
		{
			name:     "new round only",
			detail:   "new round was called, but not a new height",
			newRound: true,
		},
		{
			name:      "new round / height",
			detail:    "new height = new round 0",
			newHeight: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Election, 4)
			c.simElectionPhase(t)
			c.simProposePhase(t, 0, true, ByzantineEvidence{}, nil, 0)
			c.simPrecommitPhase(t, 0)
			c.simCommitPhase(t, 0, 0)
			c.simPacemakerPhase(t)
			eleLen, evVoteLen, propNil, propVoteLen, precNil, precVoteLen, comNil, expRound, paceLen := 0, 1, false, 1, false, 1, false, uint64(0), 3
			if test.newRound || test.newHeight {
				eleLen, evVoteLen, propNil, propVoteLen, precNil, precVoteLen, comNil, expRound = 0, 0, true, 0, true, 0, true, 1
			}
			if test.newRound {
				c.bft.NewRound(false)
			}
			if test.newHeight {
				expRound, paceLen = 0, 0
				c.bft.NewHeight()
				c.bft.NewRound(true)
			}
			require.Equal(t, c.bft.Round, expRound)
			require.Len(t, c.bft.Proposals[expRound][phaseToString(Election)], eleLen)
			require.Len(t, c.bft.Votes[expRound][phaseToString(ElectionVote)], evVoteLen)
			require.Equal(t, c.bft.Proposals[expRound][phaseToString(Propose)] == nil, propNil)
			require.Len(t, c.bft.Votes[expRound][phaseToString(ProposeVote)], propVoteLen)
			require.Equal(t, c.bft.Proposals[expRound][phaseToString(Precommit)] == nil, precNil)
			require.Len(t, c.bft.Votes[expRound][phaseToString(PrecommitVote)], precVoteLen)
			require.Equal(t, c.bft.Proposals[expRound][phaseToString(Commit)] == nil, comNil)
			require.Len(t, c.bft.PacemakerMessages, paceLen)
		})
	}
}

func TestGetPhaseWaitTime(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		phase            Phase
		round            uint64
		expectedWaitTime time.Duration
	}{
		{
			name:             "election phase wait time",
			detail:           "the wait time for election",
			phase:            Election,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().ElectionTimeoutMS) * time.Millisecond,
		},
		{
			name:             "election vote phase wait time",
			detail:           "the wait time for election vote",
			phase:            ElectionVote,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().ElectionVoteTimeoutMS) * time.Millisecond,
		},
		{
			name:             "propose phase wait time",
			detail:           "the wait time for proposal phase",
			phase:            Propose,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().ProposeTimeoutMS) * time.Millisecond,
		},
		{
			name:             "propose vote phase wait time",
			detail:           "the wait time for proposal vote phase",
			phase:            ProposeVote,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().ProposeVoteTimeoutMS) * time.Millisecond,
		},
		{
			name:             "precommit phase wait time",
			detail:           "the wait time for precommit phase",
			phase:            Precommit,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().PrecommitTimeoutMS) * time.Millisecond,
		},
		{
			name:             "precommit vote phase wait time",
			detail:           "the wait time for precommit vote phase",
			phase:            PrecommitVote,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().PrecommitVoteTimeoutMS) * time.Millisecond,
		},
		{
			name:             "commit phase wait time",
			detail:           "the wait time for commit phase",
			phase:            Commit,
			round:            0,
			expectedWaitTime: time.Duration(lib.DefaultConfig().CommitTimeoutMS) * time.Millisecond,
		},
		{
			name:             "propose phase wait time with round 3",
			detail:           "the wait time for round interrupt phase",
			phase:            Propose,
			round:            3,
			expectedWaitTime: time.Duration(lib.DefaultConfig().ProposeTimeoutMS) * time.Millisecond * (6 + 1),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Election, 1)
			require.Equal(t, c.bft.WaitTime(test.phase, test.round), test.expectedWaitTime)
		})
	}
}

func TestSafeNode(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		samePropInMsg    bool
		unlockBySafety   bool
		unlockByLiveness bool
		err              lib.ErrorI
	}{
		{
			name:   "high qc doesn't justify the proposal",
			detail: "high qc contains a different proposal than the one in the message",
			err:    ErrMismatchedProposals(),
		},
		{
			name:          "high qc fails safe node predicate",
			detail:        "high qc does not satisfy safety nor liveness",
			samePropInMsg: true,
			err:           ErrFailedSafeNodePredicate(),
		},
		{
			name:           "high qc unlocks with safety",
			detail:         "high qc does satisfy the safety portion of the safe node predicate (same proposal)",
			samePropInMsg:  true,
			unlockBySafety: true,
		},
		{
			name:             "high qc unlocks with liveness",
			detail:           "high qc does satisfy the liveness portion of the safe node predicate (higher round)",
			samePropInMsg:    true,
			unlockByLiveness: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, c2 := newTestConsensus(t, PrecommitVote, 4), newTestConsensus(t, PrecommitVote, 4)
			c2.bft.Round++
			c2.simPrecommitPhase(t, 1) // higher lock
			go c2.bft.StartPrecommitVotePhase()
			<-c2.cont.sendToProposerChan
			c.simPrecommitPhase(t, 0) // lock
			go c.bft.StartPrecommitVotePhase()
			<-c.cont.sendToProposerChan
			var err lib.ErrorI
			switch {
			case test.unlockBySafety:
				err = c.bft.SafeNode(&Message{
					Qc: &lib.QuorumCertificate{
						Results: c.bft.Results,
						Block:   c.bft.HighQC.Block,
					},
					HighQc: c.bft.HighQC,
				})
			case test.unlockByLiveness:
				err = c.bft.SafeNode(&Message{
					Qc: &lib.QuorumCertificate{
						Results: c2.bft.Results,
						Block:   c2.bft.HighQC.Block,
					},
					HighQc: c2.bft.HighQC,
				})
			default:
				msgProposal, hash, resultsHash := c.cont.NewTestBlock(), c.cont.NewTestBlockHash2(), []byte(nil)
				if test.samePropInMsg {
					msgProposal = c.cont.NewTestBlock2()
					hash = c.cont.NewTestBlockHash2()
					resultsHash = c.bft.HighQC.Results.Hash()
				}
				err = c.bft.SafeNode(&Message{
					Qc: &QC{
						Results: c.bft.HighQC.Results,
						Block:   msgProposal,
					},
					HighQc: &QC{
						Header:      c.bft.HighQC.Header,
						BlockHash:   hash,
						ResultsHash: resultsHash,
					},
				})
			}
			require.Equal(t, test.err, err)
		})
	}
}
