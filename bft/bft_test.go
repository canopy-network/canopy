package bft

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	testTimeout = 500 * time.Millisecond
)

func TestStartElectionPhase(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		selfIsValidator bool
		numValidators   int
		isCandidate     bool
	}{
		{
			name:            "self is leader",
			detail:          `deterministic key set ensures 'self' is an election candidate with a set of 6 Validators`,
			selfIsValidator: true,
			numValidators:   6,
			isCandidate:     true,
		},
		{
			name:            "self is not leader",
			detail:          `deterministic key set ensures 'self' is an not election candidate with a set of 5 Validators`,
			selfIsValidator: true,
			numValidators:   5,
			isCandidate:     false,
		},
		{
			name:            "self is not a validator",
			detail:          `self is not a validator within the deterministic key`,
			selfIsValidator: false,
			numValidators:   3,
			isCandidate:     false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, Election, test.numValidators)
			// use validator 0 as the test replica
			pub, private := c.valKeys[0].PublicKey(), c.valKeys[0]
			if !test.selfIsValidator {
				pk, err := crypto.NewBLS12381PrivateKey()
				require.NoError(t, err)
				c.bft.PublicKey = pk.PublicKey().Bytes()
			}
			// deterministic key set ensures 'self' is not an election candidate with a set of 4 Validators
			go c.bft.StartElectionPhase()
			if !test.isCandidate {
				// if not supposed to be a 'candidate', ensure no ELECTION message sent
				select {
				case <-c.cont.sendToReplicasChan:
					t.Fatal("unexpected message")
				case <-time.After(100 * time.Millisecond):
					return
				}
			} else {
				// if supposed to be a 'candidate', validate the ELECTION message
				expectedView := lib.View{
					Height:     1,
					Round:      0,
					Phase:      Election,
					RootHeight: 1,
					NetworkId:  1,
					ChainId:    lib.CanopyChainId,
				}
				select {
				case <-time.After(testTimeout):
					t.Fatal("timeout")
				case m := <-c.cont.sendToReplicasChan:
					msg, ok := m.(*Message)
					require.True(t, ok)
					require.Equal(t, expectedView, *msg.Header)
					require.Equal(t, pub.Bytes(), msg.Vrf.PublicKey)
					require.Equal(t, private.Sign(lib.FormatInputIntoSeed(c.cont.proposers.Addresses, expectedView.RootHeight, expectedView.Height, expectedView.Round)), msg.Vrf.Signature)
				}
			}
		})
	}
}

func TestStartElectionVotePhase(t *testing.T) {
	tests := []struct {
		name                 string
		detail               string
		numValidators        int
		isElectionCandidate  bool
		noElectionCandidates bool
		hasBE                bool
	}{
		{
			name:                "self is election candidate",
			detail:              `deterministic key set ensures 'self' is an election candidate with a set of 2 Validators`,
			numValidators:       3,
			isElectionCandidate: true,
		},
		{
			name:                "self is not an election candidate",
			detail:              `deterministic key set ensures 'self' is not an election candidate with a set of 5 Validators`,
			numValidators:       5,
			isElectionCandidate: false,
		},
		{
			name:                 "pseudorandom",
			detail:               `didn't 'handle messages' from any candidate, fallback to pseudorandom`,
			numValidators:        8,
			noElectionCandidates: true,
		},
		{
			name:          "byzantine evidence not nil",
			detail:        `testing sending the byzantine evidence to the leader with the leader vote`,
			numValidators: 5,
			hasBE:         true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup
			c := newTestConsensus(t, ElectionVote, test.numValidators)
			switch {
			case test.noElectionCandidates:
				c.setSortitionData(t)
			default:
				c.simElectionPhase(t)
			}
			expectedDoubleSignEvidence := DoubleSignEvidences{}
			if test.hasBE {
				expectedDoubleSignEvidence.Evidence = c.newTestDoubleSignEvidence(t)
				c.bft.ByzantineEvidence = &ByzantineEvidence{
					DSE: expectedDoubleSignEvidence,
				}
			}
			pub, _, expectedView := c.valKeys[0].PublicKey(), c.valKeys[0], lib.View{
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
				if test.isElectionCandidate || test.noElectionCandidates {
					require.Equal(t, msg.Qc.ProposerKey, pub.Bytes())
				} else {
					require.Equal(t, msg.Qc.ProposerKey, c.valKeys[3].PublicKey().Bytes())
				}
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
			name:   "self is not proposer",
			detail: `no election vote quorum certificate received`,
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
				if test.receiveEVQC {
					t.Fatal("timeout")
				}
			case m := <-c.cont.sendToReplicasChan:
				if !test.receiveEVQC {
					t.Fatal("unexpected message")
				}
				msg, ok := m.(*Message)
				require.True(t, ok)
				require.Equal(t, expectedView, *msg.Header)
				require.NotNil(t, msg.Qc)
				require.Equal(t, expectedQCView.Phase, msg.Qc.Header.Phase)
				block, results, e := c.cont.ProduceProposal(nil, nil)
				require.NoError(t, e)
				require.Equal(t, block, msg.Qc.Block)
				require.Equal(t, results.Hash(), msg.Qc.Results.Hash())
				expectedAggSig, err := multiKey.AggregateSignatures()
				require.NoError(t, err)
				require.Equal(t, expectedAggSig, msg.Qc.Signature.Signature)
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
				c.bft.Block, c.bft.Results, _ = c.cont.ProduceProposal(nil, nil)
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
		{
			name:             "received +2/3 prop vote and has election vote DSE stored",
			detail:           `received +2/3 quorum on the precommit votes from replicas and has double sign evidence stored in the form of a conflicting election votes`,
			proposalReceived: true,
			isProposer:       true,
			validProposal:    true,
			hasEVDSE:         true,
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
			Round:      0,
			RootHeight: 1,
			ChainId:    lib.CanopyChainId,
			Phase:      RoundInterrupt,
		})
		require.Equal(t, c.bft.Phase, RoundInterrupt)
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
			if test.hasPeerPacemakerVotes {
				c.simPacemakerPhase(t)
			}
			c.bft.Pacemaker()
			require.Equal(t, uint64(1), c.bft.Height)
			require.Equal(t, test.expectedPacemakerRound, c.bft.Round)
		})
	}
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
			eleLen, evVoteLen, propNil, propVoteLen, precNil, precVoteLen, comNil, expRound, paceLen := 4, 1, false, 1, false, 1, false, uint64(0), 3
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
				msgProposal, hash := c.cont.NewTestBlock(), c.cont.NewTestBlock2()
				if test.samePropInMsg {
					hash = c.cont.NewTestBlockHash()
				}
				err = c.bft.SafeNode(&Message{
					Qc: &QC{
						Results: c.bft.HighQC.Results,
						Block:   msgProposal,
					},
					HighQc: &QC{
						Header:    c.bft.HighQC.Header,
						BlockHash: hash,
					},
				})
			}
			require.Equal(t, test.err, err)
		})
	}
}
