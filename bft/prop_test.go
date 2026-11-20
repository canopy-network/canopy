package bft

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddProposal(t *testing.T) {
	// pre-define some validators to test with
	_, keys, _ := newTestValSet(t, 3)
	// define test cases
	tests := []struct {
		name    string
		detail  string
		preAdd  []*Message
		message *Message
		error   string
	}{
		{
			name:   "same phase",
			detail: "a message for this view was already received but it was from a different proposer",
			preAdd: []*Message{
				{
					Header: &lib.View{
						Phase: lib.Phase_ELECTION,
					},
					Signature: &lib.Signature{
						PublicKey: keys[0].PublicKey().Bytes(),
					},
				},
			},
			message: &Message{
				Header: &lib.View{
					Phase: lib.Phase_ELECTION,
				},
				Signature: &lib.Signature{
					PublicKey: keys[1].PublicKey().Bytes(),
				},
			},
		},

		{
			name:   "different phase",
			detail: "a message for this view has not yet been received",
			preAdd: []*Message{
				{
					Header: &lib.View{
						Phase: lib.Phase_ELECTION,
					},
					Signature: &lib.Signature{
						PublicKey: keys[0].PublicKey().Bytes(),
					},
				},
			},
			message: &Message{
				Header: &lib.View{
					Phase: lib.Phase_PROPOSE,
				},
				Signature: &lib.Signature{
					PublicKey: keys[1].PublicKey().Bytes(),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// initialize a bft object to test with
			consensus := newTestConsensus(t, Election, 3)
			// pre-add the messages
			for _, add := range test.preAdd {
				require.NoError(t, consensus.bft.AddProposal(add))
			}
			// execute the function call
			err := consensus.bft.AddProposal(test.message)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error, err)
				return
			}
			// make a convenience variable for the view of the message
			v := test.message.Header
			// ensure the message was added
			messages := consensus.bft.Proposals[v.Round][phaseToString(v.Phase)]
			require.EqualExportedValues(t, test.message, messages[len(messages)-1])
		})
	}
}

func TestGetProposalAfterNewCommitteeResetWithoutElectionCandidate(t *testing.T) {
	b := &BFT{
		Proposals: make(ProposalsForHeight),
	}

	b.ProposalsResetForNewCommittee()

	require.Nil(t, b.getProposal(0, ElectionVote))
}
