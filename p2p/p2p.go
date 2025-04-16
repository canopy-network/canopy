package p2p

import (
	"bytes"
	"context"
	"fmt"
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

const transport, dialTimeout = "tcp", time.Second

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
	log                    lib.LoggerI
}

// New() creates an initialized pointer instance of a P2P object
func New(p crypto.PrivateKeyI, maxMembersPerCommittee uint64, c lib.Config, l lib.LoggerI) *P2P {
	// initialize the peer book
	peerBook := NewPeerBook(p.PublicKey().Bytes(), c, l)
	// make inbound multiplexed channels
	channels := make(lib.Channels)
	for i := lib.Topic(0); i < lib.Topic_INVALID; i++ {
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
	// set the read/write timeout to be 2 x the block time
	ReadWriteTimeout = time.Duration(2*c.BlockTimeMS()) * time.Millisecond
	// set the peer meta
	meta := &lib.PeerMeta{ChainId: c.ChainId}
	// return the p2p structure
	return &P2P{
		privateKey:             p,
		channels:               channels,
		config:                 c,
		meta:                   meta.Sign(p),
		PeerSet:                NewPeerSet(c, p, l),
		book:                   peerBook,
		MustConnectsReceiver:   make(chan []*lib.PeerAddress, maxChanSize),
		maxMembersPerCommittee: int(maxMembersPerCommittee),
		bannedIPs:              bannedIPs,
		log:                    l,
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
			return
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
			if err = p.AddPeer(c, &lib.PeerInfo{Address: &lib.PeerAddress{NetAddress: netAddress}}, false); err != nil {
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
	// Try to connect to the DialPeers in the config
	for _, peerString := range p.config.DialPeers {
		// start a peer address structure using the basic configurations
		peerAddress := &lib.PeerAddress{PeerMeta: &lib.PeerMeta{NetworkId: p.meta.NetworkId, ChainId: p.meta.ChainId}}
		// try to populate the peer address using the peer string from the config
		if err := peerAddress.FromString(peerString); err != nil {
			// log the invalid format
			p.log.Errorf("invalid dial peer %s: %s", peerString, err.Error())
			// continue with the next
			continue
		}
		// dial in a non-blocking fashion
		go func() {
			// increment dialing
			dialing++
			// dial the peer with exponential backoff
			p.DialWithBackoff(peerAddress)
		}()
	}
	// Continuous service until program exit, dial timeout loop frequency for resource break
	for {
		time.Sleep(5 * dialTimeout)
		// for each supported plugin, try to max out peer config by dialing
		func() {
			// exit if maxed out config or none left to dial
			if (p.PeerSet.outbound+dialing >= p.config.MaxOutbound) || p.book.GetBookSize() == 0 {
				return
			}
			p.log.Debugf("Executing P2P Dial for more outbound peers")
			// get random peer for chain
			rand := p.book.GetRandom()
			if rand == nil || p.IsSelf(rand.Address) || p.Has(rand.Address.PublicKey) {
				return
			}
			// sequential operation means we'll never be dialing more than 1 peer at a time
			// the peer should be added before the next execution of the loop
			dialing++
			defer func() { dialing-- }()
			if err := p.Dial(rand.Address, false); err != nil {
				p.log.Debug(err.Error())
				return
			}
		}()
	}
}

// Dial() tries to establish an outbound connection with a peer candidate
func (p *P2P) Dial(address *lib.PeerAddress, disconnect bool) lib.ErrorI {
	if p.IsSelf(address) || p.PeerSet.Has(address.PublicKey) {
		return nil
	}
	p.log.Debugf("Dialing %s@%s", lib.BytesToString(address.PublicKey), address.NetAddress)
	// try to establish the basic tcp connection
	conn, er := net.DialTimeout(transport, address.NetAddress, dialTimeout)
	if er != nil {
		return ErrFailedDial(er)
	}
	// try to use the basic tcp connection to establish a peer
	return p.AddPeer(conn, &lib.PeerInfo{Address: address, IsOutbound: true}, disconnect)
}

// AddPeer() takes an ephemeral tcp connection and an incomplete peerInfo and attempts to
// create a E2E encrypted channel with a fully authenticated peer and save it to
// the peer set and the peer book
func (p *P2P) AddPeer(conn net.Conn, info *lib.PeerInfo, disconnect bool) (err lib.ErrorI) {
	// create the e2e encrypted connection while establishing a full peer info object
	connection, err := p.NewConnection(conn)
	if err != nil {
		return err
	}
	// replace the peer's port with the resolved port
	if err = lib.ResolveAndReplacePort(&info.Address.NetAddress, p.config.ChainId); err != nil {
		p.log.Error(err.Error())
		return
	}
	// log the peer add attempt
	p.log.Debugf("Try Add peer: %s@%s", lib.BytesToString(connection.Address.PublicKey), info.Address.NetAddress)
	// if peer is outbound, ensure the public key matches who we expected to dial
	if info.IsOutbound {
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
		connection.Stop()
		return nil
	}
	p.Lock()
	defer p.Unlock()
	// check if is must connect
	for _, item := range p.mustConnect {
		if bytes.Equal(item.PublicKey, info.Address.PublicKey) {
			info.IsMustConnect = true
			break
		}
	}
	// check if is trusted
	for _, item := range p.config.TrustedPeerIDs {
		if item == lib.BytesToString(info.Address.PublicKey) {
			info.IsTrusted = true
			break
		}
	}
	// check if is banned
	for _, item := range p.config.BannedPeerIDs {
		pubKeyString := lib.BytesToString(info.Address.PublicKey)
		if pubKeyString == item {
			return ErrBannedID(pubKeyString)
		}
	}
	// add peer to peer set and peer book
	p.log.Infof("Adding peer: %s@%s", lib.BytesToString(info.Address.PublicKey), info.Address.NetAddress)
	p.book.Add(&BookPeer{Address: info.Address})
	if err = p.PeerSet.Add(&Peer{
		conn:     connection,
		PeerInfo: info,
		stop:     sync.Once{},
	}); err != nil {
		connection.Stop()
	}
	return
}

// DialWithBackoff() dials the peer with exponential backoff retry
func (p *P2P) DialWithBackoff(peerInfo *lib.PeerAddress) {
	dialAndLog := func() (err error) {
		if err = p.Dial(peerInfo, false); err != nil {
			p.log.Errorf("Dial %s@%s failed: %s", lib.BytesToString(peerInfo.PublicKey), peerInfo.NetAddress, err.Error())
		}
		return
	}
	opts := backoff.NewExponentialBackOff()
	opts.InitialInterval = 5 * time.Second
	opts.MaxElapsedTime = time.Minute
	_ = backoff.Retry(dialAndLog, opts)
}

// DialAndDisconnect() dials the peer but disconnects once a fully authenticated connection is established
func (p *P2P) DialAndDisconnect(a *lib.PeerAddress) lib.ErrorI { return p.Dial(a, true) }

// OnPeerError() callback to P2P when a peer errors
func (p *P2P) OnPeerError(err error, publicKey []byte, remoteAddr string) {
	p.log.Warn(PeerError(publicKey, remoteAddr, err))
	// ignore error: peer may have disconnected before added
	peer, _ := p.PeerSet.Remove(publicKey)
	if peer != nil {
		peer.stop.Do(peer.conn.Stop)
	}
}

// NewStreams() creates map of streams for the multiplexing architecture
func (p *P2P) NewStreams() (streams map[lib.Topic]*Stream) {
	streams = make(map[lib.Topic]*Stream)
	for i := lib.Topic(0); i < lib.Topic_INVALID; i++ {
		streams[i] = &Stream{
			topic:        i,
			msgAssembler: make([]byte, 0, maxMessageSize),
			sendQueue:    make(chan *Packet, maxStreamSendQueueSize),
			inbox:        p.Inbox(i),
			logger:       p.log,
		}
	}
	return
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
		p.Inbox(topic) <- (&lib.MessageAndMetadata{
			Message: proto.Clone(payload),
			Sender:  &lib.PeerInfo{Address: &lib.PeerAddress{PublicKey: fromPublicKey}},
		}).WithHash()
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
			go p.DialWithBackoff(val)
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
	"PR", // Puerto Rico
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
