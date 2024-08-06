package bft

import (
	"github.com/ginchuco/ginchu/lib"
	"github.com/stretchr/testify/require"
	"testing"
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
					Proposal:     []byte("some proposal"),
					ProposalHash: c.c.HashProposal([]byte("some proposal")),
					ProposerKey:  pub,
					Signature: &lib.AggregateSignature{
						Signature: []byte("some aggregate signature"),
						Bitmap:    []byte("some bitmap"),
					},
				},
				HighQc: &QC{
					Header: &lib.View{
						Phase: Precommit,
					},
					Proposal:     []byte("some hqc proposal"),
					ProposalHash: c.c.HashProposal([]byte("some hqc proposal")),
					ProposerKey:  pub,
					Signature: &lib.AggregateSignature{
						Signature: []byte("some hqc aggregate signature"),
						Bitmap:    []byte("some hqc bitmap"),
					},
				},
				LastDoubleSignEvidence: c.newTestDoubleSignEvidence(t),
				BadProposerEvidence:    c.newTestBadProposerEvidence(t),
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
					BadProposerEvidence:    msg.BadProposerEvidence,
				}
				if msg.Qc != nil {
					expectedMsg.Qc = &QC{
						Header:       msg.Qc.Header,
						ProposalHash: msg.Qc.ProposalHash,
						ProposerKey:  msg.Qc.ProposerKey,
						Signature:    msg.Qc.Signature,
					}
					if expectedMsg.Header.Phase == Propose {
						expectedMsg.Qc.Proposal = msg.Qc.Proposal
					}
				}
				expectedSignBytes, err = lib.Marshal(expectedMsg)
			case ElectionVote, ProposeVote, PrecommitVote:
				msg.Header = nil
				expectedSignBytes, err = lib.Marshal(&QC{
					Header:       msg.Qc.Header,
					ProposalHash: msg.Qc.ProposalHash,
					ProposerKey:  msg.Qc.ProposerKey,
				})
			case RoundInterrupt:
				expectedSignBytes, err = lib.Marshal(&Message{Header: msg.Header})
			default:
				t.Fatal("unexpected phase")
			}
			require.NoError(t, err)
			require.Equal(t, expectedSignBytes, msg.SignBytes())
		})
	}
}