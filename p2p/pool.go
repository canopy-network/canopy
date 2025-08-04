package p2p

import (
	"encoding/hex"
	"net"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
)

const (
	cleanupInterval      = 1 * time.Minute // how often to cleanup expired connections
	reconnectGracePeriod = 1 * time.Minute // time the node can attempt to reconnect without losing the connection
)

// AuthenticatedConnectionPool manages pooled authenticated connections after handshake
// This provides connection reuse while maintaining strict security through peer identity verification
// Only one connection per authenticated peer is kept
type AuthenticatedConnectionPool struct {
	pool          map[string]*AuthenticatedConn // [peer_public_key_hex] -> single connection per peer
	cleanupTicker *time.Ticker                  // periodic cleanup timer
	lastCleanup   time.Time                     // last time cleanup was performed
	logger        lib.LoggerI                   // logger instance for logging
	mu            sync.RWMutex                  // thread access
}

// AuthenticatedConn wraps a fully authenticated MultiConn for pooling
// Only connections that have completed handshake and peer verification are pooled
type AuthenticatedConn struct {
	conn         *MultiConn    // the authenticated connection
	info         *lib.PeerInfo // peer information
	peerKey      string        // hex-encoded peer public key (pool identifier)
	netAddress   string        // network address for reference
	inUse        bool          // whether connection is currently in use
	disconnectAt time.Time     // when the connection was returned to pool (disconnected)
	mu           sync.Mutex    // thread access
}

// NewAuthConnectionPool creates a new authenticated connection pool
func NewAuthConnectionPool(logger lib.LoggerI) *AuthenticatedConnectionPool {
	pool := &AuthenticatedConnectionPool{
		pool:          make(map[string]*AuthenticatedConn),
		cleanupTicker: time.NewTicker(cleanupInterval),
		lastCleanup:   time.Now(),
		logger:        logger,
		mu:            sync.RWMutex{},
	}
	return pool
}

// Get retrieves an idle authenticated connection for the given peer public key
func (p *AuthenticatedConnectionPool) Get(peerPublicKey []byte) (*MultiConn, *lib.PeerInfo, bool) {
	// cleanup stale connections
	defer func() {
		if time.Since(p.lastCleanup) > cleanupInterval {
			go p.cleanup()
		}
	}()
	// hex encode peer public key
	peerKeyHex := hex.EncodeToString(peerPublicKey)
	// p.logger.Debugf("Getting authenticated connection for peer %s", peerKeyHex)
	// retrieve authenticated connection from pool
	p.mu.RLock()
	authConn, ok := p.pool[peerKeyHex]
	p.mu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	// safe locking
	authConn.mu.Lock()
	defer authConn.mu.Unlock()
	// check if connection is alive and not used
	if isConnClosed(authConn.conn.conn) || authConn.inUse {
		return nil, nil, false
	}
	// check if connection is stale
	if time.Since(authConn.disconnectAt) > reconnectGracePeriod {
		return nil, nil, false
	}
	// check if connection is still healthy
	if !p.isConnHealthy(authConn) {
		return nil, nil, false
	}
	// mark connection as in use
	authConn.inUse = true
	// mark the connection as pooled
	authConn.conn.pooled.Store(true)
	// p.logger.Debugf("Reusing authenticated connection for peer %s", peerKeyHex)
	return authConn.conn, authConn.info.Copy(), true
}

// Put returns an authenticated connection to the pool for reuse
// The connection must be fully authenticated and healthy
func (p *AuthenticatedConnectionPool) Put(conn *MultiConn, peerInfo *lib.PeerInfo) bool {
	// retrieve connection
	poolConn, peerKeyHex, exists := p.get(conn)
	if peerKeyHex == "" {
		return false
	}
	// check if connection is healthy for pooling
	if !conn.isHealthyForPooling() {
		return false
	}
	// safe locking
	p.mu.Lock()
	defer p.mu.Unlock()
	// check if there's already a connection for this peer
	if exists {
		poolConn.mu.Lock()
		// if existing connection is not in use, prefer to keep the existing one
		if !poolConn.inUse {
			poolConn.mu.Unlock()
			return false
		}
		poolConn.mu.Unlock()
	}
	// create new authenticated connection entry
	authConn := &AuthenticatedConn{
		conn:         conn,
		info:         peerInfo.Copy(),
		peerKey:      peerKeyHex,
		netAddress:   conn.Address.NetAddress,
		inUse:        false,
		disconnectAt: time.Now(),
	}
	p.pool[peerKeyHex] = authConn
	// p.logger.Debugf("Pooled authenticated connection for peer %s", peerKeyHex)
	return true
}

// Close shuts down the connection pool and closes all connections
func (p *AuthenticatedConnectionPool) Close() {
	p.cleanupTicker.Stop()
	p.mu.Lock()
	defer p.mu.Unlock()
	// Close all authenticated connections
	for _, authConn := range p.pool {
		authConn.mu.Lock()
		if authConn.conn != nil {
			authConn.conn.Stop()
		}
		authConn.mu.Unlock()
	}
	// clear pools map
	p.pool = make(map[string]*AuthenticatedConn)
}

// get retrieves an authenticated connection from the pool with basic validation
func (p *AuthenticatedConnectionPool) get(conn *MultiConn) (*AuthenticatedConn, string, bool) {
	// nil check
	if conn == nil || conn.Address == nil {
		return nil, "", false
	}
	// convert the key to hex
	peerKeyHex := hex.EncodeToString(conn.Address.PublicKey)
	// retrieve authenticated connection from pool
	p.mu.RLock()
	authConn, exists := p.pool[peerKeyHex]
	p.mu.RUnlock()
	// exit
	return authConn, peerKeyHex, exists
}

// isConnHealthy checks if an authenticated connection is still usable
func (p *AuthenticatedConnectionPool) isConnHealthy(authConn *AuthenticatedConn) bool {
	// check if the underlying MultiConn is still healthy
	if authConn.conn == nil {
		return false
	}
	// check if MultiConn thinks it's healthy for pooling
	exist := authConn.conn.isHealthyForPooling()
	return exist
}

// cleanup removes unused idle connections
func (p *AuthenticatedConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for peerKey, authConn := range p.pool {
		authConn.mu.Lock()
		// remove from pool if not in use and idle for too long
		shouldRemove := !authConn.inUse && now.Sub(authConn.disconnectAt) > reconnectGracePeriod
		if shouldRemove {
			// remove from pool
			delete(p.pool, peerKey)
			// close connection
			if authConn.conn != nil {
				authConn.conn.Stop()
			}
		}
		authConn.mu.Unlock()
	}
	// update last cleanup time
	p.lastCleanup = now
}

// isConnClosed checks if a connection is closed or broken
func isConnClosed(conn net.Conn) bool {
	// set a very short deadline
	conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	// reset deadline after checking
	defer conn.SetReadDeadline(time.Time{})
	// try to read one byte
	one := make([]byte, 0)
	_, err := conn.Read(one)
	if err != nil {
		// check for specific error types
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// connection is alive, just no data
			return false
		}
		// connection is closed or broken
		return true
	}
	// got data, connection is alive
	return false
}
