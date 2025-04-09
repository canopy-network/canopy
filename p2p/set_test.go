// nolint:all
package p2p

import (
	"net"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestPeerSetAddGetDel(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	require.True(t, len(n1.PeerSet.m) == 0)
	expected := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "pipe",
		},
		IsOutbound:    true,
		IsMustConnect: true,
		IsTrusted:     true,
		Reputation:    10,
	}
	c1, c2 := net.Pipe()
	defer c2.Close()
	require.NoError(t, n1.PeerSet.Add(&Peer{PeerInfo: expected, conn: newTestMultiConnMock(t, n2.pub, c1, n1.P2P)}))
	require.True(t, len(n1.PeerSet.m) == 1)
	got, err := n1.PeerSet.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.Equal(t, *expected, *got)
	_, err = n1.PeerSet.Remove(n2.pub)
	require.NoError(t, err)
	_, err = n1.PeerSet.Remove(n2.pub)
	require.Error(t, err)
	_, err = n1.PeerSet.GetPeerInfo(n2.pub)
	require.Error(t, err)
	require.True(t, len(n1.PeerSet.m) == 0)
}

func TestUpdateMustConnects(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()
	n1, n2, n3 := newTestP2PNode(t), newTestP2PNode(t), newTestP2PNode(t)
	require.NoError(t, n1.Add(&Peer{
		PeerInfo: &lib.PeerInfo{Address: n2.ID()},
	}))
	toDial := n1.UpdateMustConnects([]*lib.PeerAddress{
		n2.ID(),
		n3.ID(),
	})
	require.True(t, len(toDial) == 1)
	require.Equal(t, toDial[0].PublicKey, n3.pub)
	peerInfo, err := n1.PeerSet.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.True(t, peerInfo.IsMustConnect)
	_, err = n1.PeerSet.GetPeerInfo(n1.pub)
	require.Error(t, err)
}

func TestChangeReputation(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	n1.UpdateMustConnects([]*lib.PeerAddress{{PublicKey: n2.pub}})
	peerInfo, err := n1.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.True(t, peerInfo.IsMustConnect)
	require.True(t, peerInfo.Reputation == 0)
	n1.ChangeReputation(n2.pub, -11)
	peerInfo, err = n1.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.True(t, peerInfo.Reputation == -11)
	_, err = n1.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	n1.PeerSet.m[lib.BytesToString(peerInfo.Address.PublicKey)].IsMustConnect = false
	n1.ChangeReputation(n2.pub, 0)
	_, err = n1.GetPeerInfo(n2.pub)
	require.Error(t, err)
}

func TestGetAllInfosAndBookPeers(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()
	n1, n2, cleanup := newTestP2PPair(t)
	n3 := newTestP2PNode(t)
	_, c := net.Pipe()
	require.NoError(t, n1.Add(&Peer{
		conn: newTestMultiConnMock(t, n3.pub, c, n1.P2P),
		PeerInfo: &lib.PeerInfo{Address: &lib.PeerAddress{PublicKey: n3.pub, PeerMeta: &lib.PeerMeta{
			ChainId: 1,
		}}, IsOutbound: true},
	}))
	defer func() { cleanup() }()
	infos, numInbound, numOutbound := n1.GetAllInfos()
	require.True(t, len(infos) == 2)
	require.True(t, numInbound == 1)
	require.True(t, numOutbound == 1)
	for _, i := range infos {
		if i.IsOutbound {
			require.Equal(t, n3.pub, i.Address.PublicKey)
		} else {
			require.Equal(t, n2.pub, i.Address.PublicKey)
		}
	}
	bp := n1.GetBookPeers()
	require.Len(t, bp, 1)
	require.Equal(t, bp[0].Address.PublicKey, n2.pub)
}

func newTestMultiConnMock(_ *testing.T, peerPubKey []byte, conn net.Conn, p *P2P) *MultiConn {
	return &MultiConn{
		conn: conn,
		Address: &lib.PeerAddress{
			PublicKey:  peerPubKey,
			NetAddress: "",
			PeerMeta: &lib.PeerMeta{
				ChainId: 1,
			},
		},
		streams:       p.NewStreams(),
		quitSending:   make(chan struct{}, maxChanSize),
		quitReceiving: make(chan struct{}, maxChanSize),
		onError:       func(err error, bytes []byte, s string) { p.log.Error(err.Error()) },
		error:         sync.Once{},
		p2p:           p,
		log:           p.log,
	}
}
