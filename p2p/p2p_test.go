package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout = 3 * time.Second
)

func TestConnection(t *testing.T) {
	_, _, cleanup := newTestP2PPair(t)
	cleanup()
}

func TestMultiSendRec(t *testing.T) {
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "pipe",
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		require.NoError(t, n1.SendTo(n2.pub, lib.Topic_TX, &PeerBookRequestMessage{}))
		require.NoError(t, n1.SendTo(n2.pub, lib.Topic_CONSENSUS, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	<-n2.Inbox(lib.Topic_TX)
	msg := <-n2.Inbox(lib.Topic_CONSENSUS)
	gotMsg, ok := msg.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)
}

func TestSendToRand(t *testing.T) {
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "pipe",
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		peerInfo, err := n1.SendToRandPeer(lib.Topic_CONSENSUS, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}})
		require.NoError(t, err)
		require.Equal(t, peerInfo.Address.PublicKey, n2.pub)
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	msg := <-n2.Inbox(lib.Topic_CONSENSUS)
	gotMsg, ok := msg.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)
}

func TestSendToPeers(t *testing.T) {
	n1 := newStartedTestP2PNode(t)
	n2 := newTestP2PNode(t)
	n2.meta.ChainId = 1
	startTestP2PNode(t, n2)
	n1.UpdateMustConnects([]*lib.PeerAddress{n2.ID()})
	n3 := newTestP2PNode(t)
	n3.meta.ChainId = 2
	startTestP2PNode(t, n3)
	require.NoError(t, connectStartedNodes(t, n1, n2), "compatible peers")
	require.Error(t, connectStartedNodes(t, n1, n3), "incompatible peers expected")
	defer func() { n1.Stop(); n2.Stop(); n3.Stop() }()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "pipe",
			PeerMeta: &lib.PeerMeta{
				Signature: []byte("1"),
			},
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		require.NoError(t, n1.SendToPeers(lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	msg := <-n2.Inbox(lib.Topic_PEERS_RESPONSE)
	gotMsg, ok := msg.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)
}

func TestSendToPeersChunkedPacket(t *testing.T) {
	n1 := newStartedTestP2PNode(t)
	n2 := newTestP2PNode(t)
	n2.meta.ChainId = 1
	startTestP2PNode(t, n2)
	n1.UpdateMustConnects([]*lib.PeerAddress{n2.ID()})
	n3 := newTestP2PNode(t)
	n3.meta.ChainId = 2
	startTestP2PNode(t, n3)
	// n1.
	require.NoError(t, connectStartedNodes(t, n1, n2), "compatible peers")
	require.Error(t, connectStartedNodes(t, n1, n3), "incompatible peers expected")
	defer func() { n1.Stop(); n2.Stop(); n3.Stop() }()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "pipe",
			PeerMeta: &lib.PeerMeta{
				Signature: bytes.Repeat([]byte("F"), maxDataChunkSize*5),
			},
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		require.NoError(t, n1.SendToPeers(lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	msg := <-n2.Inbox(lib.Topic_PEERS_RESPONSE)
	gotMsg, ok := msg.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)
}

func TestSendToPeersMultipleMessages(t *testing.T) {
	n1 := newStartedTestP2PNode(t)
	n2 := newTestP2PNode(t)
	n2.meta.ChainId = 1
	startTestP2PNode(t, n2)
	n1.UpdateMustConnects([]*lib.PeerAddress{n2.ID()})
	n3 := newTestP2PNode(t)
	n3.meta.ChainId = 2
	startTestP2PNode(t, n3)
	require.NoError(t, connectStartedNodes(t, n1, n2), "compatible peers")
	require.Error(t, connectStartedNodes(t, n1, n3), "incompatible peers expected")
	defer func() { n1.Stop(); n2.Stop(); n3.Stop() }()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "pipe",
			PeerMeta: &lib.PeerMeta{
				Signature: []byte("1"),
			},
		},
		ConsecutiveFailedDial: 1,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		require.NoError(t, n1.SendToPeers(lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	go func() {
		defer wg.Done()
		require.NoError(t, n1.SendToPeers(lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	wg.Wait()

	msg := <-n2.Inbox(lib.Topic_PEERS_RESPONSE)
	gotMsg, ok := msg.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)

	msg2 := <-n2.Inbox(lib.Topic_PEERS_RESPONSE)
	gotMsg2, ok := msg2.Message.(*PeerBookResponseMessage)
	require.True(t, ok)
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg2.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg2.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg2.Book[0].ConsecutiveFailedDial)
}

func TestDialReceive(t *testing.T) {
	n1, n2 := newStartedTestP2PNode(t), newStartedTestP2PNode(t)
	defer func() { n1.Stop(); n2.Stop() }()
	connectStartedNodes(t, n1, n2)
}

func TestStart(t *testing.T) {
	n2, n3, n4 := newTestP2PNodeWithConfig(t, newTestP2PConfig(t), true), newTestP2PNodeWithConfig(t, newTestP2PConfig(t), true), newTestP2PNodeWithConfig(t, newTestP2PConfig(t), true)
	n3.log, n2.log = lib.NewNullLogger(), lib.NewNullLogger()
	startTestP2PNode(t, n2)
	startTestP2PNode(t, n3)
	startTestP2PNode(t, n4)
	c := newTestP2PConfig(t)
	// test dial peers
	c.DialPeers = []string{fmt.Sprintf("%s@%s", lib.BytesToString(n2.pub), n2.listener.Addr().String())}
	n1 := newTestP2PNodeWithConfig(t, c)
	// test churn process
	private, _ := crypto.NewBLS12381PrivateKey()
	random := private.PublicKey()
	pm := &lib.PeerMeta{
		NetworkId: 1,
		ChainId:   1,
	}
	n1.book.Add(&BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  random.Bytes(),
			NetAddress: n4.listener.Addr().String(),
			PeerMeta:   pm,
		},
		ConsecutiveFailedDial: MaxFailedDialAttempts - 1,
	})
	// test validator receiver
	n1.MustConnectsReceiver <- []*lib.PeerAddress{{
		PublicKey:  n2.pub,
		NetAddress: n2.listener.Addr().String(),
		PeerMeta:   pm,
	}}
	test := func() (ok bool, reason string) {
		peerInfo, _ := n1.GetPeerInfo(n2.pub)
		if peerInfo == nil {
			return false, "n2 not found"
		}
		if !peerInfo.IsMustConnect {
			return false, "n2 not validator"
		}
		if n1.book.Has(random.Bytes()) {
			return false, "n1 did not churn peer book"
		}
		n3PI, _ := n1.GetPeerInfo(n3.pub)
		if n3PI == nil {
			return false, "n3 not found"
		}
		if n3PI.IsOutbound {
			return false, "n3 incorrectly marked as outbound"
		}
		return true, ""
	}
	startTestP2PNode(t, n1)
	defer func() { n1.Stop(); n2.Stop(); n3.Stop() }()
	// test listener
	require.NoError(t, n3.Dial(&lib.PeerAddress{
		PublicKey:  n1.pub,
		NetAddress: n1.listener.Addr().String(),
		PeerMeta:   pm,
	}, false))
	for {
		select {
		default:
			if ok, _ := test(); ok {
				return
			}
		case <-time.After(testTimeout):
			_, reason := test()
			t.Fatal(reason)
		}
	}
}

func TestDialDisconnect(t *testing.T) {
	n1, n2 := newStartedTestP2PNode(t), newStartedTestP2PNode(t)
	defer func() { n1.Stop(); n2.Stop() }()
	require.NoError(t, n1.DialAndDisconnect(&lib.PeerAddress{
		PublicKey:  n2.pub,
		NetAddress: n2.listener.Addr().String(),
	}))
	_, err := n1.PeerSet.GetPeerInfo(n2.pub)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "not found"))
}

func TestConnectValidator(t *testing.T) {
	n1, n2 := newStartedTestP2PNode(t), newStartedTestP2PNode(t)
	defer func() { n1.Stop(); n2.Stop() }()
	n1.MustConnectsReceiver <- []*lib.PeerAddress{
		{
			PublicKey:  n2.pub,
			NetAddress: n2.listener.Addr().String(),
			PeerMeta:   n2.meta,
		},
	}
out:
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			n1.RLock()
			numVals := len(n1.mustConnect)
			n1.RUnlock()
			if numVals != 0 {
				break out
			}
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
	peer, err := n1.PeerSet.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.True(t, peer.IsOutbound)
	require.True(t, peer.IsMustConnect)
}

func TestSelfSend(t *testing.T) {
	topic := lib.Topic_CONSENSUS
	n := newStartedTestP2PNode(t)
	expected := (&lib.MessageAndMetadata{
		Message: &PeerBookRequestMessage{},
		Sender: &lib.PeerInfo{
			Address: &lib.PeerAddress{
				PublicKey:  n.pub,
				NetAddress: "",
			},
		},
	}).WithHash()
	require.NoError(t, n.SelfSend(n.pub, topic, &PeerBookRequestMessage{}))
	for {
		select {
		case msg := <-n.Inbox(topic):
			require.Equal(t, msg.Sender.Address.PublicKey, n.pub)
			require.Equal(t, msg.Hash, expected.Hash)
			return
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
}

func TestOnPeerError(t *testing.T) {
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	_, found := n1.book.getIndex(n2.pub)
	require.True(t, found)
	peer, err := n1.PeerSet.get(n2.pub)
	require.NoError(t, err)
	n1.OnPeerError(errors.New(""), n2.pub, "")
	_, err = n1.PeerSet.get(n2.pub)
	require.Error(t, err)
	_, e := peer.conn.conn.Read(make([]byte, 8))
	require.Error(t, e)
}

func TestNewStreams(t *testing.T) {
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	streams := n1.NewStreams()
	peer, err := n1.PeerSet.get(n2.pub)
	require.NoError(t, err)
	for i, s := range streams {
		ps := peer.conn.streams[i]
		require.Equal(t, ps.topic, s.topic)
		require.Equal(t, ps.inbox, s.inbox)
	}
}

func TestIsSelf(t *testing.T) {
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	require.True(t, n1.IsSelf(&lib.PeerAddress{PublicKey: n1.pub}))
	require.False(t, n1.IsSelf(&lib.PeerAddress{PublicKey: n2.pub}))
	require.True(t, n2.IsSelf(&lib.PeerAddress{PublicKey: n2.pub}))
	require.False(t, n2.IsSelf(&lib.PeerAddress{PublicKey: n1.pub}))
}

func TestID(t *testing.T) {
	n := newTestP2PNode(t)
	want := &lib.PeerAddress{
		PublicKey:  n.pub,
		NetAddress: n.config.ExternalAddress,
	}
	got := n.ID()
	require.Equal(t, want.PublicKey, got.PublicKey)
	require.Equal(t, want.NetAddress, got.NetAddress)
}

func TestMaxPacketSize(t *testing.T) {
	a, err := lib.NewAny(&Packet{
		StreamId: lib.Topic_INVALID,
		Eof:      true,
		Bytes:    bytes.Repeat([]byte("F"), maxDataChunkSize),
	})
	require.NoError(t, err)
	envelope := &Envelope{Payload: a}
	maxPacket, _ := lib.Marshal(envelope)
	require.Equal(t, len(maxPacket), maxPacketSize)
}

func connectStartedNodes(t *testing.T, n1, n2 testP2PNode) error {
	if err := n1.Dial(&lib.PeerAddress{
		PublicKey:  n2.pub,
		NetAddress: n2.listener.Addr().String(),
	}, false); err != nil {
		return err
	}
	peer, err := n1.PeerSet.GetPeerInfo(n2.pub)
	require.NoError(t, err)
	require.True(t, peer.IsOutbound)
o2:
	for {
		select {
		default:
			if n2.PeerSet.Has(n1.pub) {
				break o2
			}
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
	peer, err = n2.PeerSet.GetPeerInfo(n1.pub)
	require.NoError(t, err)
	require.False(t, peer.IsOutbound)
	return nil
}

func newStartedTestP2PNode(t *testing.T) testP2PNode {
	n := newTestP2PNode(t)
	return startTestP2PNode(t, n)
}

func startTestP2PNode(t *testing.T, n testP2PNode) testP2PNode {
	n.Start()
	for {
		select {
		default:
			if n.listener != nil {
				return n
			}
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
}

func newTestP2PPair(t *testing.T) (n1, n2 testP2PNode, cleanup func()) {
	n1, n2 = newTestP2PNode(t), newTestP2PNode(t)
	c1, c2 := net.Pipe()
	pipeTO := time.Now().Add(time.Second)
	err := c1.SetReadDeadline(pipeTO)
	require.NoError(t, err)
	err = c2.SetReadDeadline(pipeTO)
	require.NoError(t, err)
	cleanup = func() { n1.Stop(); n2.Stop() }
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		require.NoError(t, n1.AddPeer(c2, &lib.PeerInfo{Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: c2.RemoteAddr().String(),
			PeerMeta: &lib.PeerMeta{
				ChainId: 0,
			},
		}}, false))
		wg.Done()
	}()
	require.NoError(t, n2.AddPeer(c1, &lib.PeerInfo{Address: &lib.PeerAddress{
		PublicKey:  n1.pub,
		NetAddress: c1.RemoteAddr().String(),
		PeerMeta:   &lib.PeerMeta{ChainId: 0},
	}},
		false))
	wg.Wait()
	require.True(t, n1.PeerSet.Has(n2.pub))
	require.True(t, n2.PeerSet.Has(n1.pub))
	return
}

type testP2PNode struct {
	*P2P
	priv crypto.PrivateKeyI
	pub  []byte
}

func newTestP2PNode(t *testing.T) (n testP2PNode) {
	return newTestP2PNodeWithConfig(t, newTestP2PConfig(t))
}

func newTestP2PNodeWithConfig(t *testing.T, c lib.Config, noLog ...bool) (n testP2PNode) {
	var err error
	n.priv, err = crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	n.pub = n.priv.PublicKey().Bytes()
	require.NoError(t, err)
	logger := lib.NewDefaultLogger()
	if len(noLog) == 1 && noLog[0] == true {
		logger = lib.NewNullLogger()
	}
	n.P2P = New(n.priv, 1, nil, c, logger)
	return
}

func newTestP2PConfig(_ *testing.T) lib.Config {
	config := lib.DefaultConfig()
	config.ChainId = lib.CanopyChainId
	config.ListenAddress = ":0"
	return config
}
