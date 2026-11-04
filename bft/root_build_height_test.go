package bft

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canopy-network/canopy/lib"
)

func TestIsLockedReproposal(t *testing.T) {
	locked := &QC{BlockHash: []byte("block"), ResultsHash: []byte("results")}
	tests := []struct {
		name     string
		detail   string
		highQC   *QC
		msg      *Message
		expected bool
	}{
		{name: "not locked", detail: "a replica without a lock can't receive a reproposal", msg: &Message{HighQc: locked}},
		{name: "no justification", detail: "a proposal without a HighQC isn't a reproposal", highQC: locked, msg: &Message{}},
		{name: "same proposal", detail: "the proposal matches the lock", highQC: locked, msg: &Message{HighQc: &lib.QuorumCertificate{BlockHash: []byte("block"), ResultsHash: []byte("results")}}, expected: true},
		{name: "different block", detail: "a different block isn't a reproposal", highQC: locked, msg: &Message{HighQc: &lib.QuorumCertificate{BlockHash: []byte("other"), ResultsHash: []byte("results")}}},
		{name: "different results", detail: "different results aren't a reproposal", highQC: locked, msg: &Message{HighQc: &lib.QuorumCertificate{BlockHash: []byte("block"), ResultsHash: []byte("other")}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := &BFT{HighQC: test.highQC}
			require.Equal(t, test.expected, b.isLockedReproposal(test.msg), test.detail)
		})
	}
}
