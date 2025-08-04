package p2p

import (
	"net"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestNewAuthConnectionPool(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	// check minimum fields
	require.NotNil(t, pool)
	require.NotNil(t, pool.pool)
	require.NotNil(t, pool.cleanupTicker)
	require.Equal(t, logger, pool.logger)
	// Clean up
	pool.Close()
}

func TestAuthConnectionPool(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	defer pool.Close()
	// create test nodes
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	// create test connection
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	// create test multi connection
	multiConn := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "test:1234",
		},
		Reputation: 5,
	}
	// test Put operation
	success := pool.Put(multiConn, peerInfo)
	require.True(t, success)
	// test Get operation
	retrievedConn, retrievedInfo, found := pool.Get(n2.pub)
	require.True(t, found)
	require.Equal(t, multiConn, retrievedConn)
	require.Equal(t, peerInfo.Address.PublicKey, retrievedInfo.Address.PublicKey)
	require.Equal(t, peerInfo.Reputation, retrievedInfo.Reputation)
	// connection should be marked as pooled and in use
	require.True(t, multiConn.pooled.Load())
	// test Get for non-existent peer
	nonExistentKey := []byte("nonexistent")
	_, _, found = pool.Get(nonExistentKey)
	require.False(t, found)
}

func TestAuthConnectionPoolDuplicateConnection(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	defer pool.Close()
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	// create two connections for the same peer
	c1, c2 := net.Pipe()
	c3, c4 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	defer c3.Close()
	defer c4.Close()
	multiConn1 := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	multiConn2 := newTestMultiConnMock(t, n2.pub, c3, n1.P2P)
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "test:1234",
		},
	}
	// put first connection
	success1 := pool.Put(multiConn1, peerInfo)
	require.True(t, success1)
	// put second connection for same peer (should fail if first is not in use)
	success2 := pool.Put(multiConn2, peerInfo)
	require.False(t, success2)
	// get the first connection (marks it as in use)
	retrievedConn, _, found := pool.Get(n2.pub)
	require.True(t, found)
	require.Equal(t, multiConn1, retrievedConn)
	// putting second connection should succeed since first is in use
	success3 := pool.Put(multiConn2, peerInfo)
	require.True(t, success3)
}

func TestAuthConnectionPoolExpiredConnection(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	defer pool.Close()
	// create test nodes
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	multiConn := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "test:1234",
		},
	}
	// put connection
	success := pool.Put(multiConn, peerInfo)
	require.True(t, success)
	// manually set disconnectAt to past the grace period
	pool.mu.Lock()
	authConn := pool.pool[lib.BytesToString(n2.pub)]
	authConn.mu.Lock()
	authConn.disconnectAt = time.Now().Add(-2 * reconnectGracePeriod)
	authConn.mu.Unlock()
	pool.mu.Unlock()
	// try to get the expired connection
	_, _, found := pool.Get(n2.pub)
	require.False(t, found)
}

func TestAuthConnectionPoolCleanup(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	defer pool.Close()
	// create test nodes
	n1, n2, n3 := newTestP2PNode(t), newTestP2PNode(t), newTestP2PNode(t)
	// create connections
	c1, c2 := net.Pipe()
	c3, c4 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	defer c3.Close()
	defer c4.Close()
	multiConn1 := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	multiConn2 := newTestMultiConnMock(t, n3.pub, c3, n1.P2P)
	peerInfo1 := &lib.PeerInfo{
		Address: &lib.PeerAddress{PublicKey: n2.pub, NetAddress: "test:1234"},
	}
	peerInfo2 := &lib.PeerInfo{
		Address: &lib.PeerAddress{PublicKey: n3.pub, NetAddress: "test:5678"},
	}
	// put both connections
	require.True(t, pool.Put(multiConn1, peerInfo1))
	require.True(t, pool.Put(multiConn2, peerInfo2))
	// verify both are in pool
	require.Len(t, pool.pool, 2)
	// mark first connection as expired
	authConn1 := pool.pool[lib.BytesToString(n2.pub)]
	authConn1.disconnectAt = time.Now().Add(-2 * reconnectGracePeriod)
	authConn1.inUse = false
	// run cleanup
	pool.cleanup()
	// first connection should be removed, second should remain
	_, found1After := pool.pool[lib.BytesToString(n2.pub)]
	_, found2After := pool.pool[lib.BytesToString(n3.pub)]

	require.False(t, found1After)
	require.True(t, found2After)
}

func TestAuthConnectionPoolClose(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	// create connections
	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)
	c1, c2 := net.Pipe()
	defer c2.Close()
	multiConn := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "test:1234",
		},
	}
	// put connection
	success := pool.Put(multiConn, peerInfo)
	require.True(t, success)
	// close pool
	pool.Close()
	// verify pool is empty
	require.Empty(t, pool.pool)
	// verify connection is closed
	require.True(t, isConnClosed(multiConn.conn))
}

func TestIsConnClosed(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	// open connection
	closed := isConnClosed(c1)
	require.False(t, closed)
	// closed connection
	c3, c4 := net.Pipe()
	c3.Close()
	c4.Close()
	closed = isConnClosed(c3)
	require.True(t, closed)
}

func TestAuthConnectionPool_GetAfterConnectionInUse(t *testing.T) {
	logger := lib.NewNullLogger()
	pool := NewAuthConnectionPool(logger)
	defer pool.Close()

	n1, n2 := newTestP2PNode(t), newTestP2PNode(t)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	multiConn := newTestMultiConnMock(t, n2.pub, c1, n1.P2P)
	peerInfo := &lib.PeerInfo{
		Address: &lib.PeerAddress{
			PublicKey:  n2.pub,
			NetAddress: "test:1234",
		},
	}

	// Put connection
	success := pool.Put(multiConn, peerInfo)
	require.True(t, success, "Put should succeed")

	// Get connection (marks as in use)
	_, _, found := pool.Get(n2.pub)
	require.True(t, found, "Should find connection")

	// Try to get again (should fail because it's in use)
	_, _, found2 := pool.Get(n2.pub)
	require.False(t, found2, "Should not find in-use connection")
}
