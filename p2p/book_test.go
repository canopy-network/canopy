package p2p

import (
	"bytes"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestStartPeerBookService(t *testing.T) {
	isTest = true
	defer func() { isTest = false }()
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	n3, n4 := newTestP2PNode(t), newTestP2PNode(t)
	n1.book.Add(&BookPeer{Address: n3.ID()})
	n2.book.Add(&BookPeer{Address: n4.ID()})
	n1.StartPeerBookService()
	n2.StartPeerBookService()
	for {
		select {
		case <-time.NewTicker(time.Millisecond * 100).C:
			bp := n1.GetBookPeers()
			if len(bp) < 3 {
				continue
			}
			hasN4 := false
			for _, peer := range bp {
				if bytes.Equal(peer.Address.PublicKey, n4.pub) {
					hasN4 = true
					break
				}
			}
			require.True(t, hasN4)
			bp = n2.GetBookPeers()
			if len(bp) < 3 {
				continue
			}
			hasN3 := false
			for _, peer := range bp {
				if bytes.Equal(peer.Address.PublicKey, n3.pub) {
					hasN3 = true
					break
				}
			}
			require.True(t, hasN3)
			return
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
}

func TestGetRandom(t *testing.T) {
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	require.Nil(t, n1.book.GetRandom())
	require.Nil(t, n1.book.GetRandom())
	n1.book.Add(&BookPeer{Address: &lib.PeerAddress{
		PublicKey:  n2.pub,
		NetAddress: "",
		PeerMeta:   &lib.PeerMeta{ChainId: 1},
	}})
	got := n1.book.GetRandom()
	require.Equal(t, got.Address.PublicKey, n2.pub)
	got = n1.book.GetRandom()
	require.Equal(t, got.Address.PublicKey, n2.pub)
}

func TestGetAll(t *testing.T) {
	n1, n2, n3 := newTestP2PNode(t), newTestP2PNode(t), newTestP2PNode(t)
	require.Len(t, n1.book.GetAll(), 0)
	n1.book.Add(&BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "localhost:90001",
			PeerMeta:   &lib.PeerMeta{ChainId: 1},
		},
	})
	got := n1.book.GetAll()
	require.Len(t, got, 1)
	require.Equal(t, got[0].Address.PublicKey, n2.pub)
	n1.book.Add(&BookPeer{Address: &lib.PeerAddress{PublicKey: n3.pub, NetAddress: "localhost:90001"}})
	got = n1.book.GetAll()
	require.Len(t, got, 2)
	require.True(t, n1.book.Has(n3.pub))
	require.True(t, n1.book.Has(n2.pub))
}

func TestAddRemoveHas(t *testing.T) {
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	require.Len(t, n1.book.GetAll(), 0)
	n1.book.Add(&BookPeer{Address: &lib.PeerAddress{PublicKey: n2.pub, NetAddress: "localhost:90001"}})
	require.True(t, n1.book.Has(n2.pub))
	n1.book.Remove(n2.pub)
	require.False(t, n1.book.Has(n2.pub))
}

func TestAddFailedDialAttempt(t *testing.T) {
	startConsecutiveFailedDialAttempt := int32(3)
	n1, n2, n3 := newTestP2PNode(t), newTestP2PNode(t), newTestP2PNode(t)
	require.Len(t, n1.book.GetAll(), 0)
	n1.book.Add(&BookPeer{
		Address:               &lib.PeerAddress{PublicKey: n2.pub, PeerMeta: &lib.PeerMeta{ChainId: 1}, NetAddress: "localhost:90001"},
		ConsecutiveFailedDial: startConsecutiveFailedDialAttempt,
	})
	peer := n1.book.GetRandom()
	require.Equal(t, peer.Address.PublicKey, n2.pub)
	require.Equal(t, peer.ConsecutiveFailedDial, startConsecutiveFailedDialAttempt)
	n1.book.AddFailedDialAttempt(n3.pub)
	peer = n1.book.GetRandom()
	require.Equal(t, peer.Address.PublicKey, n2.pub)
	require.Equal(t, peer.ConsecutiveFailedDial, startConsecutiveFailedDialAttempt)
	n1.book.AddFailedDialAttempt(n2.pub)
	peer = n1.book.GetRandom()
	require.Equal(t, peer.Address.PublicKey, n2.pub)
	require.Equal(t, peer.ConsecutiveFailedDial, startConsecutiveFailedDialAttempt+1)
	n1.book.AddFailedDialAttempt(n2.pub)
	require.False(t, n1.book.Has(n2.pub))
}

func TestResetFailedDialAttempt(t *testing.T) {
	startConsecutiveFailedDialAttempt := int32(4)
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	require.Len(t, n1.book.GetAll(), 0)
	n1.book.Add(&BookPeer{
		Address:               &lib.PeerAddress{PublicKey: n2.pub, PeerMeta: &lib.PeerMeta{ChainId: 1}, NetAddress: "localhost:90001"},
		ConsecutiveFailedDial: startConsecutiveFailedDialAttempt,
	})
	peer := n1.book.GetRandom()
	require.Equal(t, peer.Address.PublicKey, n2.pub)
	require.Equal(t, peer.ConsecutiveFailedDial, startConsecutiveFailedDialAttempt)
	n1.book.ResetFailedDialAttempts(n2.pub)
	require.True(t, n1.book.Has(n2.pub))
	peer = n1.book.GetRandom()
	require.Equal(t, peer.Address.PublicKey, n2.pub)
	require.Equal(t, peer.ConsecutiveFailedDial, int32(0))
}

// Has() returns if the
func (p *PeerBook) Has(publicKey []byte) bool {
	p.l.Lock()
	defer p.l.Unlock()
	_, found := p.getIndex(publicKey)
	return found
}
