package controller

import (
	"testing"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/p2p"
	"github.com/stretchr/testify/require"
)

// TestShouldGossipRejectsObsoleteElectionProposal() verifies obsolete election proposals are not gossiped in deterministic mode.
func TestShouldGossipRejectsObsoleteElectionProposal(t *testing.T) {
	network := new(p2p.P2P)
	network.SetGossipMode(true)
	config := lib.DefaultConfig()
	config.ProposerMode = lib.ProposerModeStakeWeightedRoundRobin
	c := &Controller{P2P: network, Config: config}
	msg := &bft.Message{
		Header: &lib.View{Phase: bft.Election},
		// Extra fields must not let an ELECTION header masquerade as another
		// message class and reach gossip.
		Qc: &bft.QC{Header: &lib.View{Phase: bft.RoundInterrupt}},
	}

	gossip, exit := c.ShouldGossip(msg)
	require.False(t, gossip)
	require.True(t, exit)
}
