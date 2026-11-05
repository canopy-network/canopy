package bft

import (
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSignBytes(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		phase  Phase
	}{
		{
			name:   "election",
			detail: "validates sign bytes of an election message",
			phase:  Election,
		},
		{
			name:   "election-vote",
			detail: "validates sign bytes of an election-vote message",
			phase:  ElectionVote,
		},
		{
			name:   "propose",
			detail: "validates sign bytes of a propose message",
			phase:  Propose,
		},
		{
			name:   "propose-vote",
			detail: "validates sign bytes of a propose-vote message",
			phase:  ProposeVote,
		},
		{
			name:   "precommit",
			detail: "validates sign bytes of a precommit message",
			phase:  Precommit,
		},
		{
			name:   "precommit-vote",
			detail: "validates sign bytes of a precommit-vote message",
			phase:  PrecommitVote,
		},
		{
			name:   "commit",
			detail: "validates sign bytes of a commit message",
			phase:  Commit,
		},
		{
			name:   "pacemaker",
			detail: "validates sign bytes of a pacemaker message",
			phase:  RoundInterrupt,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, test.phase, 3)
			pub := c.valKeys[0].PublicKey().Bytes()
			results := &lib.CertificateResult{
				RewardRecipients: &lib.RewardRecipients{
					PaymentPercents: []*lib.PaymentPercents{{Address: []byte("some address"), Percent: 100, ChainId: lib.CanopyChainId}},
					NumberOfSamples: 0,
				},
			}
			msg := &Message{
				Header: &lib.View{
					Height: 1,
					Round:  0,
					Phase:  test.phase,
				},
				Vrf: &lib.Signature{
					PublicKey: pub,
					Signature: []byte("some vrf"),
				},
				Qc: &QC{
					Header: &lib.View{
						Height: 1,
						Round:  0,
						Phase:  test.phase,
					},
					Block:       []byte("some proposal"),
					BlockHash:   crypto.Hash([]byte("some proposal")),
					Results:     results,
					ResultsHash: results.Hash(),
					ProposerKey: pub,
					Signature: &lib.AggregateSignature{
						Signature: []byte("some aggregate signature"),
						Bitmap:    []byte("some bitmap"),
					},
				},
				HighQc: &QC{
					Header: &lib.View{
						Phase: Precommit,
					},
					BlockHash:   []byte("some hqc proposal"),
					Block:       crypto.Hash([]byte("some hqc proposal")),
					ProposerKey: pub,
					Signature: &lib.AggregateSignature{
						Signature: []byte("some hqc aggregate signature"),
						Bitmap:    []byte("some hqc bitmap"),
					},
				},
				LastDoubleSignEvidence: c.newTestDoubleSignEvidence(t),
				Signature: &lib.Signature{
					PublicKey: []byte("some omitted pubkey"),
					Signature: []byte("some omitted signature"),
				},
			}
			var expectedSignBytes []byte
			var err error
			switch test.phase {
			case Election, Propose, Precommit, Commit:
				expectedMsg := &Message{
					Header:                 msg.Header,
					Vrf:                    msg.Vrf,
					HighQc:                 msg.HighQc,
					LastDoubleSignEvidence: msg.LastDoubleSignEvidence,
				}
				if msg.Qc != nil {
					expectedMsg.Qc = &QC{
						Header:      msg.Qc.Header,
						BlockHash:   msg.Qc.BlockHash,
						ResultsHash: msg.Qc.ResultsHash,
						ProposerKey: msg.Qc.ProposerKey,
						Signature:   msg.Qc.Signature,
					}
				}
				expectedSignBytes, err = lib.Marshal(expectedMsg)
			case ElectionVote, ProposeVote, PrecommitVote:
				msg.Header = nil
				expectedSignBytes = (&QC{
					Header:      msg.Qc.Header,
					BlockHash:   msg.Qc.BlockHash,
					ResultsHash: msg.Qc.ResultsHash,
					ProposerKey: msg.Qc.ProposerKey,
				}).SignBytes()
			case RoundInterrupt:
				expectedSignBytes, err = lib.Marshal(&Message{Qc: &QC{Header: msg.Qc.Header}})
			default:
				t.Fatal("unexpected phase")
			}
			require.NoError(t, err)
			require.Equal(t, expectedSignBytes, msg.SignBytes())
		})
	}
}

// TestObsoleteElectionProposalIsRejectedBeforeValidation() verifies deterministic mode rejects obsolete election proposals early.
func TestObsoleteElectionProposalIsRejectedBeforeValidation(t *testing.T) {
	c := newTestConsensus(t, Election, 3)
	msg := &Message{Header: c.view(Election)}

	require.True(t, msg.IsProposerMessage())
	err := c.bft.HandleMessage(msg)
	require.Error(t, err)
	require.Equal(t, lib.CodeWrongPhase, err.Code())
	require.Empty(t, c.bft.Proposals[0][phaseToString(Election)])
}

func TestProposerMessageQCRejectsWrongRootHeightForJustification(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)

	// local view root height is 1 (from newTestConsensus); craft a QC for a different root height.
	const otherRootHeight = uint64(2)

	blk, blkHash, results, resHash := c.proposal(t)
	qc := &QC{
		Header: &lib.View{
			NetworkId:  lib.CanopyMainnetNetworkId,
			ChainId:    lib.CanopyChainId,
			Height:     1,
			Round:      0,
			RootHeight: otherRootHeight,
			Phase:      ElectionVote, // Propose justification QC
		},
		Block:       blk,
		BlockHash:   blkHash,
		Results:     results,
		ResultsHash: resHash,
		ProposerKey: c.valKeys[0].PublicKey().Bytes(),
	}

	// Build a +2/3 majority aggregate signature for the QC.
	sb := qc.SignBytes()
	mk := c.valSet.MultiKey.Copy()
	for idx := 0; idx < len(c.valKeys)-1; idx++ { // omit last signer (still +2/3 for this test valset)
		require.NoError(t, mk.AddSigner(c.valKeys[idx].Sign(sb), idx))
	}
	aggSig, err := mk.AggregateSignatures()
	require.NoError(t, err)
	qc.Signature = &lib.AggregateSignature{Signature: aggSig, Bitmap: mk.Bitmap()}

	msg := &Message{
		Header: c.view(Propose, 0), // local root height (1)
		Qc:     qc,
	}
	require.NoError(t, msg.Sign(c.valKeys[0]))

	errI := c.bft.HandleMessage(msg)
	require.Error(t, errI)
	require.ErrorContains(t, errI, "wrong phase")
}

func TestProposerMessageQCRejectsWrongRootHeightForPartialQC(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)

	// local view root height is 1 (from newTestConsensus); craft a QC for a different root height.
	const otherRootHeight = uint64(2)

	blk, blkHash, results, resHash := c.proposal(t)
	qc := &QC{
		Header: &lib.View{
			NetworkId:  lib.CanopyMainnetNetworkId,
			ChainId:    lib.CanopyChainId,
			Height:     1,
			Round:      0,
			RootHeight: otherRootHeight,
			Phase:      ElectionVote,
		},
		Block:       blk,
		BlockHash:   blkHash,
		Results:     results,
		ResultsHash: resHash,
		ProposerKey: c.valKeys[0].PublicKey().Bytes(),
	}

	// Build a partial (non +2/3) aggregate signature for the QC.
	sb := qc.SignBytes()
	mk := c.valSet.MultiKey.Copy()
	require.NoError(t, mk.AddSigner(c.valKeys[0].Sign(sb), 0)) // one signer only => partial QC
	aggSig, err := mk.AggregateSignatures()
	require.NoError(t, err)
	qc.Signature = &lib.AggregateSignature{Signature: aggSig, Bitmap: mk.Bitmap()}

	msg := &Message{
		Header: c.view(Propose, 0),
		Qc:     qc,
	}
	require.NoError(t, msg.Sign(c.valKeys[0]))

	require.ErrorContains(t, c.bft.HandleMessage(msg), "wrong phase")
}

func TestHandleMessageNilSignatureReturnsError(t *testing.T) {
	c := newTestConsensus(t, ProposeVote, 3)
	msg := &Message{
		Qc: &QC{
			Header: c.view(ProposeVote, 0),
		},
	}
	errI := c.bft.HandleMessage(msg)
	require.Error(t, errI)
	require.Equal(t, ErrPartialSignatureEmpty().Code(), errI.Code())
}

// TestNextHeightMessageWaitsForBlockReset() verifies next-height messages remain buffered until the block reset.
func TestNextHeightMessageWaitsForBlockReset(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	msg := &Message{Header: c.view(Propose)}
	msg.Header.Height++
	require.NoError(t, msg.Sign(c.valKeys[0]))

	// The message is not validated against height 1 and lost; it is retained until the block listener advances consensus to its height.
	require.NoError(t, c.bft.HandleMessage(msg))
	require.Len(t, c.bft.futureMsgs, 1)
	require.Equal(t, proto.Size(msg), c.bft.futureBytes)
	require.Empty(t, c.bft.Proposals[msg.Header.Round][phaseToString(msg.Header.Phase)])
	require.Nil(t, c.bft.takeFutureMessages(c.bft.Height))
	require.Len(t, c.bft.futureMsgs, 1)

	c.bft.Height = msg.Header.Height
	deferred := c.bft.takeFutureMessages(c.bft.Height)
	require.Equal(t, []*Message{msg}, deferred)
	require.Empty(t, c.bft.futureMsgs)
	require.Zero(t, c.bft.futureBytes)
}

// TestNextHeightMessageRejectedWhenFutureBufferIsFull() verifies the future-message buffer remains bounded.
func TestNextHeightMessageRejectedWhenFutureBufferIsFull(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	c.bft.futureBytes = maxFutureMessageBytes
	msg := &Message{Header: c.view(Propose)}
	msg.Header.Height++
	require.NoError(t, msg.Sign(c.valKeys[0]))

	require.Error(t, c.bft.HandleMessage(msg))
	require.Empty(t, c.bft.futureMsgs)
}

func TestNextHeightMessageFromNonMemberIsNotDeferred(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	key, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	msg := &Message{Header: c.view(Propose)}
	msg.Header.Height++
	require.NoError(t, msg.Sign(key))

	require.Error(t, c.bft.HandleMessage(msg))
	require.Empty(t, c.bft.futureMsgs)
	require.Zero(t, c.bft.futureBytes)
}

func TestNextHeightMessagesAreBoundedPerSigner(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	for round := uint64(0); round < maxFutureMessagesPerSigner; round++ {
		msg := &Message{Header: c.view(Propose, round)}
		msg.Header.Height++
		require.NoError(t, msg.Sign(c.valKeys[0]))
		require.NoError(t, c.bft.HandleMessage(msg))
	}
	msg := &Message{Header: c.view(Propose, maxFutureMessagesPerSigner)}
	msg.Header.Height++
	require.NoError(t, msg.Sign(c.valKeys[0]))

	require.Error(t, c.bft.HandleMessage(msg))
	require.Len(t, c.bft.futureMsgs, maxFutureMessagesPerSigner)
}

// TestNextHeightPacemakerWaitsForResetBeforeRoundWindowCheck() verifies next-height pacemaker messages wait for reset before round validation.
func TestNextHeightPacemakerWaitsForResetBeforeRoundWindowCheck(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	c.bft.Round = 5
	c.bft.round.Store(5)
	msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, 0)}}
	msg.Qc.Header.Height++
	require.NoError(t, msg.Sign(c.valKeys[0]))

	require.NoError(t, c.bft.HandleMessage(msg))
	require.Equal(t, []*Message{msg}, c.bft.futureMsgs)
}

// TestLegacyModeDoesNotBufferNextHeightMessage() verifies legacy mode does not use deterministic next-height buffering.
func TestLegacyModeDoesNotBufferNextHeightMessage(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	c.bft.Config.ProposerMode = lib.ProposerModeVRF
	msg := &Message{Header: c.view(Propose)}
	msg.Header.Height++
	require.NoError(t, msg.Sign(c.valKeys[0]))

	require.Error(t, c.bft.HandleMessage(msg))
	require.Empty(t, c.bft.futureMsgs)
}

func TestPacemakerMessageSignatureBindsClaimedRound(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, 0)}}
	require.NoError(t, msg.Sign(c.valKeys[1]))

	msg.Qc.Header.Round = 1

	errI := c.bft.HandleMessage(msg)
	require.Error(t, errI)
	require.Equal(t, ErrInvalidPartialSignature().Code(), errI.Code())
}

// TestPacemakerMessageAllowsArbitraryHigherRoundCatchUp() verifies authenticated pacemaker evidence permits higher-round catch-up.
func TestPacemakerMessageAllowsArbitraryHigherRoundCatchUp(t *testing.T) {
	c := newTestConsensus(t, Propose, 4)
	for _, validator := range []int{1, 2} {
		msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, 3)}}
		require.NoError(t, msg.Sign(c.valKeys[validator]))
		require.NoError(t, c.bft.HandleMessage(msg))
	}

	round, ok := c.bft.TimeoutCatchUpRound()
	require.True(t, ok)
	require.Equal(t, uint64(3), round)
	c.bft.SetHeightAnchor(time.Now())
	newViewSent := make(chan struct{})
	go func() {
		<-c.cont.sendToProposerChan
		close(newViewSent)
	}()
	c.bft.Controller.Lock()
	c.bft.advanceReadyPhases()
	c.bft.Controller.Unlock()
	<-newViewSent
	require.Equal(t, uint64(3), c.bft.Round)
	c.bft.PhaseTimer.Stop()
}

// TestPacemakerMessageRetentionIsBoundedAcrossRounds() verifies retained pacemaker votes remain bounded across rounds.
func TestPacemakerMessageRetentionIsBoundedAcrossRounds(t *testing.T) {
	c := newTestConsensus(t, Propose, 1)
	for round := uint64(0); round < 100; round++ {
		msg := &Message{Qc: &QC{Header: c.view(RoundInterrupt, round)}}
		require.NoError(t, msg.Sign(c.valKeys[0]))
		require.NoError(t, c.bft.HandleMessage(msg))
		require.LessOrEqual(t, len(c.bft.PacemakerMessages), maxPacemakerMessagesPerSigner)
	}
	rounds := make([]uint64, 0, maxPacemakerMessagesPerSigner)
	for _, msg := range c.bft.PacemakerMessages {
		rounds = append(rounds, msg.Qc.Header.Round)
	}
	require.ElementsMatch(t, []uint64{98, 99}, rounds)
}

func TestProposerMessageRejectsInvalidHighQC(t *testing.T) {
	c := newTestConsensus(t, Propose, 3)
	justifyPropose := c.simElectionVotePhase(t, 0, false, false, false, 0)
	aggSig, err := justifyPropose.AggregateSignatures()
	require.NoError(t, err)

	blk, blkHash, results, resHash := c.proposal(t)
	highQC := c.setupTestableHighQC(t, false, true)
	highQC.Signature.Signature = append([]byte{}, highQC.Signature.Signature...)
	highQC.Signature.Signature[0] ^= 0xFF

	msg := &Message{
		Header: c.view(Propose, 0),
		Qc: &QC{
			Header:      c.view(ElectionVote, 0),
			Block:       blk,
			BlockHash:   blkHash,
			Results:     results,
			ResultsHash: resHash,
			ProposerKey: c.valKeys[0].PublicKey().Bytes(),
			Signature: &lib.AggregateSignature{
				Signature: aggSig,
				Bitmap:    justifyPropose.Bitmap(),
			},
		},
		HighQc: highQC,
	}
	require.NoError(t, msg.Sign(c.valKeys[0]))

	errI := c.bft.HandleMessage(msg)
	require.Error(t, errI)
	require.Equal(t, lib.CodeInvalidAggregateSignature, errI.Code())
}

func TestProposerMessageRejectsPostElectionRelay(t *testing.T) {
	c := newTestConsensus(t, Precommit, 3)
	c.simPrecommitPhase(t, 0)
	msg := c.bft.Proposals[0][phaseToString(Precommit)][0]
	require.NoError(t, msg.Sign(c.valKeys[1]))

	err := c.bft.HandleMessage(msg)
	require.Error(t, err)
	require.Equal(t, lib.CodeInvalidSigner, err.Code())
}
