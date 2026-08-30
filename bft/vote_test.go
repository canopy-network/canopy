package bft

import (
	"bytes"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestAddVote(t *testing.T) {
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
			name:   "duplicate voter",
			detail: "a message for this view was already received from this peer",
			preAdd: []*Message{
				{
					Qc: &lib.QuorumCertificate{
						Header: &lib.View{
							Phase: lib.Phase_ELECTION_VOTE,
						},
					},
					Signature: &lib.Signature{
						PublicKey: keys[0].PublicKey().Bytes(),
						Signature: bytes.Repeat([]byte("F"), 96),
					},
				},
			},
			message: &Message{
				Qc: &lib.QuorumCertificate{
					Header: &lib.View{
						Phase: lib.Phase_ELECTION_VOTE,
					},
				},
				Signature: &lib.Signature{
					PublicKey: keys[0].PublicKey().Bytes(),
					Signature: bytes.Repeat([]byte("F"), 96),
				},
			},
			error: "duplicate vote",
		},
		{
			name:   "vote added",
			detail: "this vote message is valid and unique, so no error",
			preAdd: []*Message{},
			message: &Message{
				Qc: &lib.QuorumCertificate{
					Header: &lib.View{
						Phase: lib.Phase_ELECTION_VOTE,
					},
				},
				Signature: &lib.Signature{
					PublicKey: keys[0].PublicKey().Bytes(),
					Signature: bytes.Repeat([]byte("F"), 96),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// initialize a bft object to test with
			consensus := newTestConsensus(t, ElectionVote, 3)
			// pre-add the messages
			for _, add := range test.preAdd {
				require.NoError(t, consensus.bft.AddVote(add))
			}
			// execute the function call
			err := consensus.bft.AddVote(test.message)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error, err)
				return
			}
			// make a convenience variable for the view of the message
			v := test.message.Qc.Header
			// ensure the message was added
			messages := consensus.bft.Votes[v.Round][phaseToString(v.Phase)]
			// calculate the payload
			payload := crypto.HashString(test.message.SignBytes())
			require.EqualExportedValues(t, test.message, messages[payload].Vote)
		})
	}
}

func TestAddVoteAttachments(t *testing.T) {
	consensus := newTestConsensus(t, ElectionVote, 4)
	consensus.bft.Config.RunVDF = true
	newVote := func(i int, vdf *crypto.VDF) *Message {
		return &Message{
			Qc: &QC{Header: &lib.View{Phase: ElectionVote}},
			Signature: &lib.Signature{
				PublicKey: consensus.valKeys[i].PublicKey().Bytes(),
				Signature: bytes.Repeat([]byte("F"), 96),
			},
			Vdf: vdf,
		}
	}
	require.NoError(t, consensus.bft.AddVote(newVote(0, nil)))
	err := consensus.bft.AddVote(newVote(0, &crypto.VDF{Proof: make([]byte, maxVDFElementSize+1), Iterations: 1}))
	require.ErrorContains(t, err, "duplicate vote")
	for i, vdf := range []*crypto.VDF{
		{Proof: make([]byte, maxVDFElementSize+1), Iterations: 1},
		{Output: make([]byte, maxVDFElementSize+1), Iterations: 1},
		{Iterations: maxVDFIterations + 1},
	} {
		vote := newVote(i+1, vdf)
		require.NoError(t, consensus.bft.AddVote(vote))
		require.NotZero(t, consensus.bft.getVoteSet(vote).TotalVotedPower)
	}
	require.Empty(t, consensus.bft.VDFCache)
}
