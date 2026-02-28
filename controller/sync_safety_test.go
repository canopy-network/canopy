package controller

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	"github.com/stretchr/testify/require"
)

func newTestQCForSyncCheckpoint(t *testing.T, networkID, chainID, height uint64) *lib.QuorumCertificate {
	t.Helper()
	blk := &lib.Block{BlockHeader: &lib.BlockHeader{
		Height:             height,
		NetworkId:          uint32(networkID),
		Time:               uint64(time.Now().UnixMicro()),
		NumTxs:             0,
		TotalTxs:           0,
		TotalVdfIterations: 0,
		LastBlockHash:      crypto.Hash([]byte("last")),
		StateRoot:          crypto.Hash([]byte("state")),
		TransactionRoot:    crypto.Hash([]byte("txroot")),
		ValidatorRoot:      crypto.Hash([]byte("vroot")),
		NextValidatorRoot:  crypto.Hash([]byte("nextvroot")),
		ProposerAddress:    bytes.Repeat([]byte{0xAA}, crypto.AddressSize),
	}}
	blockHash, err := blk.Hash()
	require.NoError(t, err)
	blockBz, err := lib.Marshal(blk)
	require.NoError(t, err)
	results := &lib.CertificateResult{
		RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{
				ChainId: chainID,
				Address: bytes.Repeat([]byte{0xBB}, crypto.AddressSize),
				Percent: 100,
			}},
		},
	}
	qc := &lib.QuorumCertificate{
		Header: &lib.View{
			Height:    height,
			NetworkId: networkID,
			ChainId:   chainID,
			Phase:     lib.Phase_PRECOMMIT_VOTE,
		},
		Block:       blockBz,
		BlockHash:   blockHash,
		Results:     results,
		ResultsHash: results.Hash(),
		Signature: &lib.AggregateSignature{
			Signature: bytes.Repeat([]byte{0x11}, crypto.BLS12381SignatureSize),
			Bitmap:    []byte{0x01},
		},
	}
	require.NoError(t, qc.CheckBasic())
	return qc
}

func TestHandlePeerBlockSyncCheckpointMismatchReturnsError(t *testing.T) {
	const networkID, chainID = uint64(1), uint64(1)
	height := uint64(CheckpointFrequency)
	qc := newTestQCForSyncCheckpoint(t, networkID, chainID, height)
	checkpoint := crypto.Hash([]byte("expected-checkpoint"))
	require.False(t, bytes.Equal(qc.BlockHash, checkpoint))
	c := &Controller{
		FSM:    &fsm.StateMachine{},
		Config: lib.DefaultConfig(),
		log:    lib.NewDefaultLogger(),
		checkpoints: map[uint64]map[uint64]lib.HexBytes{
			chainID: {height: checkpoint},
		},
	}
	c.Config.ChainId = chainID
	c.Config.P2PConfig.NetworkID = networkID
	_, err := c.HandlePeerBlock(&lib.BlockMessage{ChainId: chainID, BlockAndCertificate: qc}, true)
	require.Equal(t, fsm.ErrInvalidCheckpoint(), err)
}

func TestSyncingDoneDoesNotFatalOnVDFMismatch(t *testing.T) {
	c := &Controller{
		FSM: &fsm.StateMachine{},
		log: lib.NewDefaultLogger(),
	}
	require.True(t, c.syncingDone(0, 1))
}

func TestAssembleSyncingPeersNoEmptyPrefix(t *testing.T) {
	peers := []*lib.PeerInfo{
		{Address: &lib.PeerAddress{PublicKey: []byte{0x01}}},
		{Address: &lib.PeerAddress{PublicKey: []byte{0x02}}},
	}
	syncingPeers := assembleSyncingPeers(peers)
	require.Len(t, syncingPeers, 2)
	require.NotEmpty(t, syncingPeers[0])
	require.NotEmpty(t, syncingPeers[1])
}

func TestGetRandomAllowedPeerSkipsEmptyID(t *testing.T) {
	limiter := lib.NewLimiter(1000, 1000, 1, "test", lib.NewDefaultLogger())
	selected := getRandomAllowedPeer([]string{"", "peer-1"}, limiter)
	require.Equal(t, "peer-1", selected)
}

func TestSingleNodeNetworkReturnsErrorNoFatal(t *testing.T) {
	c := &Controller{
		FSM:   &fsm.StateMachine{},
		log:   lib.NewDefaultLogger(),
		Mutex: &sync.Mutex{},
	}
	_, err := c.singleNodeNetwork()
	require.Error(t, err)
}

func TestShouldGossipPacemakerContinuesLocalHandling(t *testing.T) {
	c := &Controller{
		PublicKey: []byte{0xAA},
		P2P:       &p2p.P2P{},
	}
	c.P2P.SetGossipMode(true)
	msg := &bft.Message{
		Qc: &bft.QC{
			Header: &lib.View{Phase: bft.RoundInterrupt},
		},
	}
	gossip, exit := c.ShouldGossip(msg)
	require.True(t, gossip)
	require.False(t, exit)
}
