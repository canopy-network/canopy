package p2p

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cenkalti/backoff/v4"
	"github.com/phuslu/iploc"
	"golang.org/x/net/netutil"
	"google.golang.org/protobuf/proto"
)

/*
	P2P
	- TCP/IP transport [x]
	- Multiplexing [x]
	- Encrypted connection [x]
	- UnPn & nat-pimp auto config [-]
	- DOS mitigation [x]
	- Peer configs: unconditional, num in/out, timeouts [x]
	- Peer list: discover[x], churn[x], share[x]
	- Message dissemination: gossip [x]
*/

const (
	transport               = "tcp"
	dialTimeout             = time.Second
	minPeerTick             = 100 * time.Millisecond
	inboxMonitorInterval    = 5 * time.Second
	defaultMinIOTimeout     = 500 * time.Millisecond
	defaultMaxWriteTimeout  = 2 * time.Second
	defaultMaxReadTimeout   = 4 * time.Second
	dialFailedPeersInterval = 2 * time.Second
)

type P2P struct {
	privateKey             crypto.PrivateKeyI
	listener               net.Listener
	channels               lib.Channels
	meta                   *lib.PeerMeta
	PeerSet                          // active set
	book                   *PeerBook // not active set
	MustConnectsReceiver   chan []*lib.PeerAddress
	maxMembersPerCommittee int
	bannedIPs              []net.IPAddr // banned IPs (non-string)
	config                 lib.Config
	metrics                *lib.Metrics
	log                    lib.LoggerI
	gossip                 bool     // whether gossip mode is active
	failedPeers            sync.Map // peers that have connection errors
}

// New() creates an initialized pointer instance of a P2P object
func New(p crypto.PrivateKeyI, maxMembersPerCommittee uint64, m *lib.Metrics, c lib.Config, l lib.LoggerI) *P2P {
	// initialize the peer book
	peerBook := NewPeerBook(p.PublicKey().Bytes(), c, l)
	// make inbound multiplexed channels
	channels := make(lib.Channels)
	for i := lib.Topic(0); i <= lib.Topic_HEARTBEAT; i++ {
		channels[i] = make(chan *lib.MessageAndMetadata, maxInboxQueueSize)
	}
	// load banned IPs
	var bannedIPs []net.IPAddr
	for _, ip := range c.BannedIPs {
		i, err := net.ResolveIPAddr("", ip)
		if err != nil {
			l.Fatalf(err.Error())
		}
		bannedIPs = append(bannedIPs, *i)
	}
	// derive IO deadlines from config but clamp to short, predictable bounds
	configuredWriteTimeout := time.Duration(2*c.BlockTimeMS()) * time.Millisecond
	WriteTimeout = clampDuration(configuredWriteTimeout, defaultMinIOTimeout, defaultMaxWriteTimeout)
	// ensure read timeout is never shorter than write timeout but still bounded
	ReadTimeout = clampDuration(WriteTimeout*2, WriteTimeout, defaultMaxReadTimeout)
	// set the peer meta
	meta := &lib.PeerMeta{NetworkId: c.NetworkID, ChainId: c.ChainId}
	// return the p2p structure
	return &P2P{
		privateKey:             p,
		channels:               channels,
		metrics:                m,
		config:                 c,
		meta:                   meta.Sign(p),
		PeerSet:                NewPeerSet(c, p, m, l),
		book:                   peerBook,
		MustConnectsReceiver:   make(chan []*lib.PeerAddress, maxChanSize),
		maxMembersPerCommittee: int(maxMembersPerCommittee),
		bannedIPs:              bannedIPs,
		log:                    l,
		failedPeers:            sync.Map{},
	}
}

// Start() begins the P2P service
func (p *P2P) Start() {
	p.log.Info("Starting P2P 🤝 ")
	// Listens for 'must connect peer ids' from the main internal controller
	go p.ListenForMustConnects()
	// Starts the peer address book exchange service
	go p.StartPeerBookService()
	// Listens for external inbound peers
	go p.ListenForInboundPeers(&lib.PeerAddress{NetAddress: p.config.ListenAddress})
	// Dials external outbound peers
	go p.DialForOutboundPeers()
	// Start inbox monitoring
	go p.MonitorInboxStats(inboxMonitorInterval)
	// Start dialing failed peers
	go p.DialFailedPeers(dialFailedPeersInterval)
}

// Stop() stops the P2P service
func (p *P2P) Stop() {
	// it's possible the listener has not yet been initialized before stopping
	if p.listener != nil {
		if err := p.listener.Close(); err != nil {
			p.log.Error(err.Error())
		}
	}
	// gracefully closes all the existing connections
	p.PeerSet.Stop()
}

// ListenForInboundPeers() starts a rate-limited tcp listener service to accept inbound peers
func (p *P2P) ListenForInboundPeers(listenAddress *lib.PeerAddress) {
	ln, er := net.Listen(transport, listenAddress.NetAddress)
	if er != nil {
		p.log.Fatal(ErrFailedListen(er).Error())
	}
	p.log.Infof("Starting net.Listener on tcp://%s", listenAddress.NetAddress)
	p.listener = netutil.LimitListener(ln, p.MaxPossibleInbound())
	// continuous service until program exit
	for {
		// wait for and then accept inbound tcp connection
		c, err := p.listener.Accept()
		if err != nil {
			<-time.After(5 * time.Second)
			p.log.Errorf("Accept error: %v", err)
			continue
		}
		// create a thread to prevent front-of-the-line blocking
		go func(c net.Conn) {
			// ephemeral connections are basic, inbound tcp connections
			defer func() {
				if r := recover(); r != nil {
					p.log.Errorf("panic recovered, err: %s, stack: %s", r, string(debug.Stack()))
				}
			}()
			p.log.Debugf("Received ephemeral connection %s", c.RemoteAddr().String())
			// begin to create a peer address using the inbound tcp conn while filtering any bad ips
			netAddress, e := p.filterBadIPs(c)
			if e != nil {
				p.log.Debugf("Closing ephemeral connection %s", c.RemoteAddr().String())
				_ = c.Close()
				return
			}
			if netAddress == "" {
				p.log.Debugf("Closing ephemeral connection due to no net address %s", c.RemoteAddr().String())
				_ = c.Close()
				return
			}
			// tries to create a full peer from the ephemeral connection and just the net address
			if err = p.AddPeer(c, &lib.PeerInfo{Address: &lib.PeerAddress{NetAddress: netAddress}}, false, false); err != nil {
				p.log.Error(err.Error())
				_ = c.Close()
				return
			}
		}(c)
	}
}

// DialForOutboundPeers() uses the config and peer book to try to max out the outbound peer connections
func (p *P2P) DialForOutboundPeers() {
	// create a tracking variable to ensure not 'over dialing'
	dialing := 0
	getPeerFromString := func(address string) (*lib.PeerAddress, error) {
		// start a peer address structure using the basic configurations
		peer := &lib.PeerAddress{PeerMeta: &lib.PeerMeta{NetworkId: p.meta.NetworkId, ChainId: p.meta.ChainId}}
		// try to populate the peer address using the peer string from the given string
		if err := peer.FromString(address); err != nil {
			return nil, fmt.Errorf("invalid dial peer %s: %s", address, err.Error())
		}
		// exit
		return peer, nil
	}
	// Try to connect to the DialPeers in the config
	for _, peerString := range p.config.DialPeers {
		peerAddress, err := getPeerFromString(peerString)
		if err != nil {
			// log the invalid format
			p.log.Errorf(err.Error())
			// continue with the next
			continue
		}
		// dial in a non-blocking fashion
		go func() {
			// increment dialing
			dialing++
			// dial the peer with exponential backoff
			p.DialWithBackoff(peerAddress, true)
		}()
	}
	// Continuous service until program exit, dial timeout loop frequency for resource break
	for {
		time.Sleep(5 * dialTimeout)
		// for each supported plugin, try to max out peer config by dialing
		func() {
			// exit if maxed out config or none left to dial
			outbound := p.PeerSet.outbound
			if outbound > 0 && outbound+dialing >= p.config.MaxOutbound {
				return
			}
			// try to get a peer to dial
			var peer *lib.PeerAddress
			// first try to get a random peer from the book
			if randPeer := p.book.GetRandom(); randPeer != nil && !p.IsSelf(randPeer.Address) &&
				!p.Has(randPeer.Address.PublicKey) {
				peer = randPeer.Address
			} else if len(p.config.DialPeers) > 0 {
				// otherwise, fallback to config's dial peers
				dialPeer, err := getPeerFromString(p.config.DialPeers[rand.Intn(len(p.config.DialPeers))])
				if err != nil {
					p.log.Errorf(err.Error())
					return
				}
				peer = dialPeer
			} else {
				// no available peers to dial
				return
			}
			p.log.Debugf("Executing P2P Dial for more outbound peers")
			// sequential operation means we'll never be dialing more than 1 peer at a time
			// the peer should be added before the next execution of the loop
			dialing++
			defer func() { dialing-- }()
			if err := p.Dial(peer, false, false); err != nil {
				p.book.AddFailedDialAttempt(peer)
				p.log.Debug(err.Error())
				return
			} else {
				// if succeeded, reset failed attempts
				p.book.ResetFailedDialAttempts(peer)
			}
		}()
	}
}

// DialFailedPeers intermittently dials must connect peers that have failed while connected due to
// network issues, heartbeat timeouts, or any general connection errors
func (p *P2P) DialFailedPeers(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		p.failedPeers.Range(func(key, rawPeer any) bool {
			peer, ok := rawPeer.(*lib.PeerAddress)
			// invalid peer, remove
			if !ok {
				p.failedPeers.Delete(key)
				return true
			}
			// confirm peer is a must connect, otherwise remove
			if !p.PeerSet.IsMustConnect(peer.PublicKey) {
				p.failedPeers.Delete(key)
				return true
			}
			// skip if already reconnected
			if p.PeerSet.Has(peer.PublicKey) {
				p.failedPeers.Delete(key)
				return true
			}
			// attempt to reconnect to the peer, clear after attempt
			go func(k any, pa *lib.PeerAddress) {
				p.log.Debugf("Dialing %x failed peer", pa.PublicKey)
				p.DialWithBackoff(pa, false)
				// only remove from failed list if successfully reconnected
				if p.PeerSet.Has(pa.PublicKey) {
					p.failedPeers.Delete(k)
				}
			}(key, peer)
			return true
		})
	}
}

// Dial() tries to establish an outbound connection with a peer candidate
func (p *P2P) Dial(address *lib.PeerAddress, disconnect, strictPublicKey bool) lib.ErrorI {
	if p.IsSelf(address) || p.PeerSet.Has(address.PublicKey) {
		return nil
	}
	// only log if not immediate disconnect
	if !disconnect {
		p.log.Debugf("Dialing %s@%s", lib.BytesToString(address.PublicKey), address.NetAddress)
	}
	// try to establish the basic tcp connection
	conn, er := net.DialTimeout(transport, address.NetAddress, dialTimeout)
	if er != nil {
		return ErrFailedDial(er)
	}
	// try to use the basic tcp connection to establish a peer
	return p.AddPeer(conn, &lib.PeerInfo{Address: address, IsOutbound: true}, disconnect, strictPublicKey)
}

// AddPeer() takes an ephemeral tcp connection and an incomplete peerInfo and attempts to
// create a E2E encrypted channel with a fully authenticated peer and save it to
// the peer set and the peer book
func (p *P2P) AddPeer(conn net.Conn, info *lib.PeerInfo, disconnect, strictPublicKey bool) (err lib.ErrorI) {
	// create the e2e encrypted connection while establishing a full peer info object
	connection, err := p.NewConnection(conn)
	if err != nil {
		return err
	}
	// always in case of error close connection
	// first we need to check the error on creation to prevent panics
	defer func() {
		if err != nil {
			p.log.Warn(err.Error())
			connection.Stop()
			pk := connection.p2p.publicKey
			peer, peerErr := p.get(pk)
			// peer was not added, nothing to do
			if peerErr != nil {
				return
			}
			p.PeerSet.Remove(pk, peer.conn.uuid)
		}
	}()
	// log the peer add attempt
	p.log.Debugf("Try Add peer: %s@%s", lib.BytesToString(connection.Address.PublicKey), info.Address.NetAddress)
	// if peer is outbound, ensure the public key matches who we expected to dial
	// this validation should just be done if the peer is from config not the peer book
	if info.IsOutbound && strictPublicKey {
		if !bytes.Equal(connection.Address.PublicKey, info.Address.PublicKey) {
			return ErrMismatchPeerPublicKey(info.Address.PublicKey, connection.Address.PublicKey)
		}
	}
	// overwrite the incomplete peer info with the complete and authenticated info
	info.Address = &lib.PeerAddress{
		PublicKey:  connection.Address.PublicKey,
		NetAddress: info.Address.NetAddress,
		PeerMeta:   connection.Address.PeerMeta,
	}
	// disconnect immediately if prompted by params
	if disconnect {
		p.log.Debugf("Disconnecting from peer %s", lib.BytesToTruncatedString(info.Address.PublicKey))
		connection.Stop()
		return nil
	}
	unlock := lockWithTrace("p2p", &p.mux, p.log)
	// check whether the connection has errors
	if connection.hasError.Load() {
		unlock()
		return
	}

	// check if is must connect
	for _, item := range p.mustConnect {
		if bytes.Equal(item.PublicKey, info.Address.PublicKey) {
			info.IsMustConnect = true
			break
		}
	}
	// check if is trusted
	if slices.Contains(p.config.TrustedPeerIDs, lib.BytesToString(info.Address.PublicKey)) {
		info.IsTrusted = true
	}
	// check if is banned
	for _, item := range p.config.BannedPeerIDs {
		pubKeyString := lib.BytesToString(info.Address.PublicKey)
		if pubKeyString == item {
			unlock()
			return ErrBannedID(pubKeyString)
		}
	}
	bookPeer := &BookPeer{Address: info.Address}
	if err = p.PeerSet.Add(&Peer{
		conn:     connection,
		PeerInfo: info,
	}); err != nil {
		unlock()
		return err
	}
	unlock()
	p.book.Add(bookPeer)
	// add peer to peer set and peer book
	p.log.Infof("Added peer: %s@%s", lib.BytesToString(info.Address.PublicKey), info.Address.NetAddress)
	return
}

// DialWithBackoff() dials the peer with exponential backoff retry
func (p *P2P) DialWithBackoff(peerInfo *lib.PeerAddress, strictPublicKey bool) {
	dialAndLog := func() (err error) {
		if err = p.Dial(peerInfo, false, strictPublicKey); err != nil {
			p.log.Warnf("Dial %s@%s failed: %s", lib.BytesToString(peerInfo.PublicKey), peerInfo.NetAddress, err.Error())
		}
		return
	}
	opts := backoff.NewExponentialBackOff()
	opts.InitialInterval = 5 * time.Second
	opts.MaxElapsedTime = time.Minute
	_ = backoff.Retry(dialAndLog, opts)
}

// DialAndDisconnect() dials the peer but disconnects once a fully authenticated connection is established
func (p *P2P) DialAndDisconnect(a *lib.PeerAddress, strictPublicKey bool) lib.ErrorI {
	p.log.Debugf("DialAndDisconnect %s@%s", lib.BytesToString(a.PublicKey), a.NetAddress)
	return p.Dial(a, true, strictPublicKey)
}

// OnPeerError() callback to P2P when a peer errors
func (p *P2P) OnPeerError(err error, publicKey []byte, remoteAddr string, uuid uint64) {
	p.log.Warn(PeerError(publicKey, remoteAddr, err))
	// ignore error: peer may have disconnected before added
	if err = p.PeerSet.Remove(publicKey, uuid); err != nil {
		p.log.Errorf("Remove error: %s", err.Error())
	}
	// add to failed peers using the configured address from mustConnects
	// (remoteAddr may be an ephemeral port for inbound connections)
	// NOTE: caller already holds PeerSet.mux, so access mustConnect directly
	netAddr := remoteAddr
	idx := slices.IndexFunc(p.mustConnect, func(peer *lib.PeerAddress) bool {
		return bytes.Equal(peer.PublicKey, publicKey)
	})
	if idx >= 0 {
		netAddr = p.mustConnect[idx].NetAddress
	}
	p.failedPeers.Store(string(publicKey), &lib.PeerAddress{
		PublicKey:  publicKey,
		NetAddress: netAddr,
	})
}

// NewStreams() creates map of streams for the multiplexing architecture
func (p *P2P) NewStreams() (streams map[lib.Topic]*Stream) {
	streams = make(map[lib.Topic]*Stream, lib.Topic_INVALID+1)
	for i := range lib.Topic_INVALID {
		if i == lib.Topic_HEARTBEAT {
			continue
		}
		streams[i] = &Stream{
			topic:        i,
			msgAssembler: make([]byte, 0),
			sendQueue:    make(chan *PacketWithTiming, maxStreamSendQueueSize),
			inbox:        p.Inbox(i),
			logger:       p.log,
		}
	}
	// reserved stream for heartbeats (not forwarded to application inbox)
	streams[lib.Topic_HEARTBEAT] = &Stream{
		topic:        lib.Topic_HEARTBEAT,
		msgAssembler: make([]byte, 0),
		sendQueue:    make(chan *PacketWithTiming, maxStreamSendQueueSize),
		inbox:        nil,
		logger:       p.log,
	}
	return
}

// cleanup releases used memory in stream
func (s *Stream) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.msgAssembler = nil // Release the buffer
	close(s.sendQueue)   // Close send channel
}

// IsSelf() returns if the peer address public key equals the self public key
func (p *P2P) IsSelf(a *lib.PeerAddress) bool {
	return bytes.Equal(p.privateKey.PublicKey().Bytes(), a.PublicKey)
}

// SelfSend() executes an internal pipe send to self
func (p *P2P) SelfSend(fromPublicKey []byte, topic lib.Topic, payload proto.Message) lib.ErrorI {
	p.log.Debugf("Self sending %s message", topic)
	// non blocking
	go func() {
		bz, _ := lib.Marshal(payload)
		m := &lib.MessageAndMetadata{
			Message: bz,
			Sender:  &lib.PeerInfo{Address: &lib.PeerAddress{PublicKey: fromPublicKey}},
		}
		select {
		case p.Inbox(topic) <- m:
		default:
			p.log.Errorf("CRITICAL: Inbox %s queue full in self send", lib.Topic_name[int32(topic)])
			p.log.Error("Dropping all messages")
			// drain inbox
			func() {
				for {
					select {
					case <-p.Inbox(topic):
						// drop
					default:
						// channel is empty now
						return
					}
				}
			}()
		}
	}()
	return nil
}

// MaxPossiblePeers() sums the MaxIn, MaxOut, MaxCommitteeConnects and trusted peer IDs
func (p *P2P) MaxPossiblePeers() int {
	return (p.config.MaxInbound + p.config.MaxOutbound + p.maxMembersPerCommittee) + len(p.config.TrustedPeerIDs)
}

// MaxPossibleInbound() sums the MaxIn, MaxCommitteeConnects and trusted peer IDs
func (p *P2P) MaxPossibleInbound() int {
	return (p.config.MaxInbound + p.maxMembersPerCommittee) + len(p.config.TrustedPeerIDs)
}

// MaxPossibleOutbound() sums the MaxIn, MaxCommitteeConnects and trusted peer IDs
func (p *P2P) MaxPossibleOutbound() int {
	return (p.config.MaxOutbound + p.maxMembersPerCommittee) + len(p.config.TrustedPeerIDs)
}

// Inbox() is a getter for the multiplexed stream with a specific topic
func (p *P2P) Inbox(topic lib.Topic) chan *lib.MessageAndMetadata { return p.channels[topic] }

// ListenForMustConnects() is an internal listener that receives 'must connect peers' updates from the controller
func (p *P2P) ListenForMustConnects() {
	for mustConnect := range p.MustConnectsReceiver {
		// UpdateMustConnects() removes connections that are already established
		for _, val := range p.UpdateMustConnects(mustConnect) {
			go p.DialWithBackoff(val, false)
		}
	}
}

// ID() returns the self peer address
func (p *P2P) ID() *lib.PeerAddress {
	return &lib.PeerAddress{
		PublicKey:  p.privateKey.PublicKey().Bytes(),
		NetAddress: p.config.ExternalAddress,
		PeerMeta:   p.meta,
	}
}

// WaitForMinimumPeers() doesn't return until the minimum peer count is reached
// This may be useful when coordinating network starts
func (p *P2P) WaitForMinimumPeers() {
	ticker := time.NewTicker(minPeerTick)
	defer ticker.Stop()
	// every 'tick'
	for range ticker.C {
		// if reached the minimum number of peers
		if p.PeerCount() >= p.config.MinimumPeersToStart {
			// exit
			return
		}
	}
}

// clampDuration bounds d between min and max (inclusive).
func clampDuration(d, min, max time.Duration) time.Duration {
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}

var blockedCountries = []string{
	"AF", // Afghanistan
	"BY", // Belarus
	"CF", // Central African Republic
	"CU", // Cuba
	"IR", // Iran
	"KP", // North Korea
	"LB", // Lebanon
	"LY", // Libya
	"ML", // Mali
	"NI", // Nicaragua
	"RU", // Russia
	"SD", // Sudan
	"SS", // South Sudan
	"SY", // Syria
	"VE", // Venezuela
	"YE", // Yemen
	"ZW", // Zimbabwe
}

// filterBadIPs() returns the net address string and blocks any undesirable ip addresses
func (p *P2P) filterBadIPs(conn net.Conn) (netAddress string, e lib.ErrorI) {
	remoteAddr := conn.RemoteAddr()
	tcpAddr, ok := remoteAddr.(*net.TCPAddr)
	if !ok {
		return "", ErrNonTCPAddress()
	}
	host := tcpAddr.IP.String()
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return "", ErrIPLookup(err)
	}
	for _, ip := range ips {
		for _, bannedIP := range p.bannedIPs {
			if ip.IP.Equal(bannedIP.IP) {
				return "", ErrBannedIP(ip.String())
			}
		}
		originCountry := iploc.Country(ip.IP)
		if slices.Contains(blockedCountries, originCountry) {
			return "", ErrBannedCountry(originCountry)
		}
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", tcpAddr.Port)), nil
}

// catchPanic() is a programmatic safeguard against panics within the caller
func (p *P2P) catchPanic() {
	if r := recover(); r != nil {
		p.log.Error(string(debug.Stack()))
	}
}

// MonitorInboxStats continuously monitors and logs inbox channel depths
// without blocking message processing. Safe to run as a goroutine.
func (p *P2P) MonitorInboxStats(interval time.Duration) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			p.log.Errorf("MonitorInboxStats panic: %v, stack: %s", r, string(debug.Stack()))
		}
	}()
	p.log.Infof("Starting inbox monitoring with interval: %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	tickCount := 0
	for range ticker.C {
		tickCount++
		// Log heartbeat every 10 ticks to prove it's running
		if tickCount%10 == 0 {
			p.log.Debugf("Inbox monitor heartbeat: tick #%d", tickCount)
		}
		// Update queue depth metrics for prometheus
		p.UpdateQueueDepthMetrics()
		// Collect stats without blocking
		stats := p.GetInboxStats()
		// Calculate total messages across all inboxes
		totalMessages := 0
		for _, count := range stats {
			totalMessages += count
		}
		// Log even when idle every 60 seconds to confirm monitoring is active
		if totalMessages == 0 {
			if tickCount%4 == 0 { // Every 60 seconds with 15s interval
				p.log.Debugf("Inbox Stats: All inboxes empty (monitoring active)")
			}
			continue
		}
		// Log summary
		p.log.Infof("Inbox Stats: Total=%d msgs across %d topics", totalMessages, len(stats))
		// Log details for non-empty inboxes
		for topic, count := range stats {
			if count > 0 {
				percentage := float64(count) / float64(maxInboxQueueSize) * 100

				if percentage > 50 {
					p.log.Warnf("  ⚠️  %s: %d msgs (%.1f%% full)", lib.Topic_name[int32(topic)], count, percentage)
				} else {
					p.log.Infof("  ✓ %s: %d msgs (%.1f%% full)", lib.Topic_name[int32(topic)], count, percentage)
				}
			}
		}
	}
	p.log.Warnf("MonitorInboxStats exited unexpectedly")
}

// GetInboxStats returns the current message count for each inbox channel
// This operation is non-blocking and safe to call concurrently
func (p *P2P) GetInboxStats() map[lib.Topic]int {
	stats := make(map[lib.Topic]int)
	// len() on channels is non-blocking and thread-safe
	for topic, ch := range p.channels {
		stats[topic] = len(ch)
	}
	return stats
}

// UpdateQueueDepthMetrics updates prometheus metrics for send and inbox queue depths
// by iterating through all peer connections and aggregating queue sizes
func (p *P2P) UpdateQueueDepthMetrics() {
	if p.metrics == nil {
		return
	}
	// Track send queue depths by aggregating across all peers
	sendQueueDepths := make(map[lib.Topic]int)
	// Get all peers and check their stream send queues
	p.PeerSet.mux.RLock()
	for _, peer := range p.PeerSet.m {
		if peer.conn != nil && peer.conn.streams != nil {
			for topic, stream := range peer.conn.streams {
				if stream != nil && stream.sendQueue != nil {
					sendQueueDepths[topic] += len(stream.sendQueue)
				}
			}
		}
	}
	p.PeerSet.mux.RUnlock()
	// Update send queue depth metrics
	for topic, depth := range sendQueueDepths {
		p.metrics.SendQueueDepth.WithLabelValues(lib.Topic_name[int32(topic)]).Set(float64(depth))
	}
	// Update inbox queue depth metrics
	for topic, ch := range p.channels {
		p.metrics.InboxQueueDepth.WithLabelValues(lib.Topic_name[int32(topic)]).Set(float64(len(ch)))
	}
}

// SetGossipMode sets the gossip mode for the P2P instance
func (p *P2P) SetGossipMode(gossip bool) {
	p.gossip = gossip
}

// GossipMode returns the current gossip mode for the P2P instance
func (p *P2P) GossipMode() bool {
	return p.gossip
}
