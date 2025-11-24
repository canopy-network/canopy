package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/units"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout = 10 * time.Second
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
			NetAddress: "localhost:90001",
			PeerMeta:   &lib.PeerMeta{ChainId: 1},
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
	gotMsg := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg.Message, gotMsg))
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
			NetAddress: "localhost:90001",
			PeerMeta:   &lib.PeerMeta{ChainId: 1},
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
	gotMsg := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg.Message, gotMsg))
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
	//require.Error(t, connectStartedNodes(t, n1, n3), "incompatible peers expected")
	defer func() { n1.Stop(); n2.Stop(); n3.Stop() }()
	expectedMsg := &BookPeer{
		Address: &lib.PeerAddress{
			PublicKey:  n1.pub,
			NetAddress: "localhost:90001",
			PeerMeta: &lib.PeerMeta{
				NetworkId: 1,
				ChainId:   lib.CanopyChainId,
				Signature: []byte("1"),
			},
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		require.NoError(t, n1.SendToPeers(lib.Topic_CONSENSUS, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	msg := <-n2.Inbox(lib.Topic_CONSENSUS)
	gotMsg := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg.Message, gotMsg))
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)
}

func TestSendToPeersChunkedPacket(t *testing.T) {
	if maxChunksPerPacket == 256 {
		t.SkipNow()
	}
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
			NetAddress: "localhost:90001",
			PeerMeta: &lib.PeerMeta{
				Signature: bytes.Repeat([]byte("F"), int(maxDataChunkSize)*5),
			},
		},
		ConsecutiveFailedDial: 1,
	}
	go func() {
		require.NoError(t, n1.SendToPeers(lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	msg := <-n2.Inbox(lib.Topic_PEERS_RESPONSE)
	gotMsg := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg.Message, gotMsg))
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
			NetAddress: "localhost:90001",
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
		require.NoError(t, n1.SendToPeers(lib.Topic_CONSENSUS, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	go func() {
		defer wg.Done()
		require.NoError(t, n1.SendToPeers(lib.Topic_CONSENSUS, &PeerBookResponseMessage{Book: []*BookPeer{expectedMsg}}))
		time.AfterFunc(testTimeout, func() { panic("timeout") })
	}()
	wg.Wait()

	msg := <-n2.Inbox(lib.Topic_CONSENSUS)
	gotMsg := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg.Message, gotMsg))
	require.True(t, len(gotMsg.Book) == 1)
	require.Equal(t, expectedMsg.Address.NetAddress, gotMsg.Book[0].Address.NetAddress)
	require.Equal(t, expectedMsg.Address.PublicKey, gotMsg.Book[0].Address.PublicKey)
	require.Equal(t, expectedMsg.ConsecutiveFailedDial, gotMsg.Book[0].ConsecutiveFailedDial)

	msg2 := <-n2.Inbox(lib.Topic_CONSENSUS)
	gotMsg2 := new(PeerBookResponseMessage)
	require.NoError(t, lib.Unmarshal(msg2.Message, gotMsg2))
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
	randomPeerAddress := &lib.PeerAddress{
		PublicKey:  random.Bytes(),
		NetAddress: n4.listener.Addr().String(),
		PeerMeta:   pm,
	}
	n1.book.Add(&BookPeer{
		Address:               randomPeerAddress,
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
		if n1.book.Has(randomPeerAddress) {
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
	}, false, true))
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
	}, false))
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
			unlock := rlockWithTrace("p2p", &n1.mux, n1.log)
			numVals := len(n1.mustConnect)
			unlock()
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
	require.NoError(t, n.SelfSend(n.pub, topic, &PeerBookRequestMessage{}))
	for {
		select {
		case msg := <-n.Inbox(topic):
			require.Equal(t, msg.Sender.Address.PublicKey, n.pub)
			return
		case <-time.After(testTimeout):
			t.Fatal("timeout")
		}
	}
}

func TestOnPeerError(t *testing.T) {
	n1, n2, cleanup := newTestP2PPair(t)
	defer cleanup()
	n2PeerAddress := &lib.PeerAddress{
		PublicKey: n2.pub,
		NetAddress: "pipe" +
			"",
		PeerMeta: &lib.PeerMeta{
			NetworkId: 1,
			ChainId:   1,
		},
	}
	_, found := n1.book.getIndex(n2PeerAddress)
	require.True(t, found)
	peer, err := n1.PeerSet.get(n2.pub)
	require.NoError(t, err)
	n1.OnPeerError(errors.New(""), n2.pub, "", peer.conn.uuid)
	_, err = n1.PeerSet.get(n2.pub)
	require.Error(t, err)
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
	if int(maxDataChunkSize) > int(1*units.MB) {
		t.SkipNow()
	}
	a, err := lib.NewAny(&Packet{
		StreamId: lib.Topic_INVALID,
		Eof:      true,
		Bytes:    bytes.Repeat([]byte("F"), int(maxDataChunkSize)),
	})
	require.NoError(t, err)
	envelope := &Envelope{Payload: a}
	maxPacket, _ := lib.Marshal(envelope)
	require.EqualValues(t, len(maxPacket), int(maxPacketSize))
}

func connectStartedNodes(t *testing.T, n1, n2 testP2PNode) error {
	if err := n1.Dial(&lib.PeerAddress{
		PublicKey:  n2.pub,
		NetAddress: n2.listener.Addr().String(),
	}, false, true); err != nil {
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
	n1PeerAddress := &lib.PeerAddress{
		PublicKey:  n2.pub,
		NetAddress: c2.RemoteAddr().String(),
		PeerMeta: &lib.PeerMeta{
			ChainId: 0,
		},
	}
	n2PeerAddress := &lib.PeerAddress{
		PublicKey:  n1.pub,
		NetAddress: c1.RemoteAddr().String(),
		PeerMeta:   &lib.PeerMeta{ChainId: 0},
	}
	go func() {
		require.NoError(t, n1.AddPeer(c2, &lib.PeerInfo{Address: n1PeerAddress}, false, true))
		wg.Done()
	}()
	require.NoError(t, n2.AddPeer(c1, &lib.PeerInfo{Address: n2PeerAddress},
		false, true))
	wg.Wait()
	n1.peerAddress = n1PeerAddress
	n2.peerAddress = n2PeerAddress
	require.True(t, n1.PeerSet.Has(n2.pub))
	require.True(t, n2.PeerSet.Has(n1.pub))
	return
}

type testP2PNode struct {
	*P2P
	priv        crypto.PrivateKeyI
	peerAddress *lib.PeerAddress
	pub         []byte
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
	n.peerAddress = &lib.PeerAddress{
		PublicKey:  n.pub,
		NetAddress: "localhost:90001",
		PeerMeta:   &lib.PeerMeta{ChainId: 1},
	}
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
	config.DataDirPath = os.TempDir()
	return config
}

func TestGetInboxStats(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		setup    func(*P2P)
		expected map[lib.Topic]int
	}{
		{
			name:   "empty inboxes",
			detail: "all inboxes are empty",
			setup:  func(p *P2P) {},
			expected: map[lib.Topic]int{
				lib.Topic_CONSENSUS:      0,
				lib.Topic_BLOCK:          0,
				lib.Topic_BLOCK_REQUEST:  0,
				lib.Topic_TX:             0,
				lib.Topic_PEERS_RESPONSE: 0,
				lib.Topic_PEERS_REQUEST:  0,
			},
		},
		{
			name:   "single message in TX inbox",
			detail: "one transaction message queued",
			setup: func(p *P2P) {
				// Create a fake peer info
				peerInfo := &lib.PeerInfo{
					Address: &lib.PeerAddress{
						PublicKey:  []byte("test-peer"),
						NetAddress: "localhost:9001",
						PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
					},
				}
				// Send a message to TX channel
				txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte("test-tx")}}
				msgBytes, _ := lib.Marshal(txMsg)
				p.channels[lib.Topic_TX] <- &lib.MessageAndMetadata{
					Message: msgBytes,
					Sender:  peerInfo,
				}
			},
			expected: map[lib.Topic]int{
				lib.Topic_CONSENSUS:      0,
				lib.Topic_BLOCK:          0,
				lib.Topic_BLOCK_REQUEST:  0,
				lib.Topic_TX:             1,
				lib.Topic_PEERS_RESPONSE: 0,
				lib.Topic_PEERS_REQUEST:  0,
			},
		},
		{
			name:   "multiple messages in different inboxes",
			detail: "messages spread across multiple topics",
			setup: func(p *P2P) {
				peerInfo := &lib.PeerInfo{
					Address: &lib.PeerAddress{
						PublicKey:  []byte("test-peer"),
						NetAddress: "localhost:9001",
						PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
					},
				}

				// Add 3 messages to TX
				txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte("test-tx")}}
				msgBytes, _ := lib.Marshal(txMsg)
				for i := 0; i < 3; i++ {
					p.channels[lib.Topic_TX] <- &lib.MessageAndMetadata{
						Message: msgBytes,
						Sender:  peerInfo,
					}
				}

				// Add 2 messages to BLOCK
				blockMsg := &lib.BlockMessage{ChainId: 1, MaxHeight: 100}
				blockBytes, _ := lib.Marshal(blockMsg)
				for i := 0; i < 2; i++ {
					p.channels[lib.Topic_BLOCK] <- &lib.MessageAndMetadata{
						Message: blockBytes,
						Sender:  peerInfo,
					}
				}

				// Add 1 message to PEERS_RESPONSE
				peerBookMsg := &PeerBookResponseMessage{Book: []*BookPeer{}}
				peerBookBytes, _ := lib.Marshal(peerBookMsg)
				p.channels[lib.Topic_PEERS_RESPONSE] <- &lib.MessageAndMetadata{
					Message: peerBookBytes,
					Sender:  peerInfo,
				}
			},
			expected: map[lib.Topic]int{
				lib.Topic_CONSENSUS:      0,
				lib.Topic_BLOCK:          2,
				lib.Topic_BLOCK_REQUEST:  0,
				lib.Topic_TX:             3,
				lib.Topic_PEERS_RESPONSE: 1,
				lib.Topic_PEERS_REQUEST:  0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a test P2P node
			p2pNode := newTestP2PNode(t)
			defer p2pNode.Stop()

			// Setup test scenario
			test.setup(p2pNode.P2P)

			// Give a small delay for messages to be queued
			time.Sleep(10 * time.Millisecond)

			// Execute function call
			stats := p2pNode.GetInboxStats()

			// Verify results
			require.Equal(t, test.expected, stats, test.detail)
		})
	}
}

func TestGetInboxStatsThreadSafety(t *testing.T) {
	// Create test P2P node
	p2pNode := newTestP2PNode(t)
	defer p2pNode.Stop()

	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  []byte("test-peer"),
			NetAddress: "localhost:9001",
			PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
		},
	}

	// Concurrently send messages and read stats
	done := make(chan bool)
	errs := make(chan error, 100)

	// Writer goroutines - send messages
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte(fmt.Sprintf("tx-%d-%d", id, j))}}
				msgBytes, err := lib.Marshal(txMsg)
				if err != nil {
					errs <- err
					return
				}
				p2pNode.channels[lib.Topic_TX] <- &lib.MessageAndMetadata{
					Message: msgBytes,
					Sender:  peerInfo,
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Reader goroutines - read stats
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				stats := p2pNode.GetInboxStats()
				// Just verify it doesn't panic and returns a valid map
				require.NotNil(t, stats)
				require.GreaterOrEqual(t, len(stats), 6)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case err := <-errs:
			t.Fatal(err)
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	// Verify no errs occurred
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
		// No errs - good
	}
}

func TestMonitorInboxStats(t *testing.T) {
	// Create test P2P node
	p2pNode := newTestP2PNode(t)
	defer p2pNode.Stop()

	// Start monitoring in background
	done := make(chan bool)
	go func() {
		// Run for 3 seconds (3 monitoring cycles)
		time.Sleep(3 * time.Second)
		done <- true
	}()

	// Start the monitor (it will run until test completes)
	go p2pNode.MonitorInboxStats(1 * time.Second)

	// Simulate message flow
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  []byte("test-peer"),
			NetAddress: "localhost:9001",
			PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
		},
	}

	// sending some messages
	go func() {
		for i := 0; i < 10; i++ {
			txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte(fmt.Sprintf("tx-%d", i))}}
			msgBytes, _ := lib.Marshal(txMsg)
			p2pNode.channels[lib.Topic_TX] <- &lib.MessageAndMetadata{
				Message: msgBytes,
				Sender:  peerInfo,
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	<-done
}

func TestGetInboxStatsPerformance(t *testing.T) {
	// Create test P2P node
	p2pNode := newTestP2PNode(t)
	defer p2pNode.Stop()

	// Fill channels with messages
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  []byte("test-peer"),
			NetAddress: "localhost:9001",
			PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
		},
	}

	// Add 1000 messages to each channel
	for topic := lib.Topic_CONSENSUS; topic < lib.Topic_INVALID; topic++ {
		for i := 0; i < 1000; i++ {
			txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte("test")}}
			msgBytes, _ := lib.Marshal(txMsg)
			p2pNode.channels[topic] <- &lib.MessageAndMetadata{
				Message: msgBytes,
				Sender:  peerInfo,
			}
		}
	}

	// Measure performance of GetInboxStats
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		stats := p2pNode.GetInboxStats()
		require.NotNil(t, stats)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// GetInboxStats should be very fast (< 1 ms per call)
	require.Less(t, avgTime, time.Millisecond,
		fmt.Sprintf("GetInboxStats too slow: %v per call", avgTime))

	t.Logf("GetInboxStats performance: %v per call (%d iterations)", avgTime, iterations)
}

func TestInboxStatsWithFullChannel(t *testing.T) {
	// Create test P2P node
	p2pNode := newTestP2PNode(t)
	defer p2pNode.Stop()

	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  []byte("test-peer"),
			NetAddress: "localhost:9001",
			PeerMeta:   &lib.PeerMeta{NetworkId: 1, ChainId: 1},
		},
	}

	// Fill TX channel to capacity (500,000)
	// For testing, use a smaller number to avoid timeout
	fillAmount := 1000

	for i := 0; i < fillAmount; i++ {
		txMsg := &lib.TxMessage{ChainId: 1, Txs: [][]byte{[]byte(fmt.Sprintf("tx-%d", i))}}
		msgBytes, _ := lib.Marshal(txMsg)
		p2pNode.channels[lib.Topic_TX] <- &lib.MessageAndMetadata{
			Message: msgBytes,
			Sender:  peerInfo,
		}
	}

	// Get stats
	stats := p2pNode.GetInboxStats()

	// Verify TX channel count
	require.Equal(t, fillAmount, stats[lib.Topic_TX])

	// Calculate percentage
	percentage := float64(stats[lib.Topic_TX]) / float64(maxInboxQueueSize) * 100

	t.Logf("TX channel: %d messages (%.4f%% full)", stats[lib.Topic_TX], percentage)

	// Verify other channels are empty
	for topic := lib.Topic_CONSENSUS; topic < lib.Topic_INVALID; topic++ {
		if topic != lib.Topic_TX {
			require.Equal(t, 0, stats[topic])
		}
	}
}
