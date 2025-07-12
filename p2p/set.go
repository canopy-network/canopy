package p2p

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
	"slices"
	"sync"
)

const (
	MaxPeerReputation     = 10
	MinimumPeerReputation = -10
)

// PeerSet is the structure that maintains the connections and metadata of connected peers
type PeerSet struct {
	m            map[string]*Peer   // public key -> Peer
	mustConnect  []*lib.PeerAddress // list of peers that must be connected to
	inbound      int                // inbound count
	outbound     int                // outbound count
	sync.RWMutex                    // read / write mutex
	metrics      *lib.Metrics       // telemetry
	config       lib.P2PConfig      // p2p configuration
	publicKey    []byte             // self public key
	logger       lib.LoggerI
}

func NewPeerSet(c lib.Config, priv crypto.PrivateKeyI, metrics *lib.Metrics, logger lib.LoggerI) PeerSet {
	return PeerSet{
		m:           make(map[string]*Peer),
		mustConnect: make([]*lib.PeerAddress, 0),
		inbound:     0,
		outbound:    0,
		RWMutex:     sync.RWMutex{},
		metrics:     metrics,
		config:      c.P2PConfig,
		publicKey:   priv.PublicKey().Bytes(),
		logger:      logger,
	}
}

// Peer is a multiplexed connection + authenticated peer information
type Peer struct {
	conn          *MultiConn // multiplexed tcp connection
	*lib.PeerInfo            // authenticated information of the peer
	stop          sync.Once  // ensures a peer may only be stopped once
}

// Add() introduces a peer to the set
func (ps *PeerSet) Add(p *Peer) (err lib.ErrorI) {
	// check if peer is already added
	pubKey := lib.BytesToString(p.Address.PublicKey)
	if _, found := ps.m[pubKey]; found {
		return ErrPeerAlreadyExists(pubKey)
	}
	// ensure peer is not self
	if bytes.Equal(p.Address.PublicKey, ps.publicKey) {
		return nil
	}
	// if trusted or must connect, don't check inbound/outbound limits nor increment counts
	if p.IsTrusted || p.IsMustConnect {
		ps.set(p)
		return nil
	}
	// check limits
	if p.IsOutbound && ps.outbound >= ps.config.MaxOutbound {
		return ErrMaxOutbound()
	} else if !p.IsOutbound && ps.inbound >= ps.config.MaxInbound {
		return ErrMaxInbound()
	}
	// increment counts
	if p.IsOutbound {
		ps.outbound++
	} else {
		ps.inbound++
	}
	// set the peer
	ps.set(p)
	// update metrics
	ps.metrics.UpdatePeerMetrics(len(ps.m), ps.inbound, ps.outbound)
	return nil
}

// Remove() evicts a peer from the set
func (ps *PeerSet) Remove(publicKey []byte) (peer *Peer, err lib.ErrorI) {
	ps.Lock()
	defer ps.Unlock()
	peer, err = ps.get(publicKey)
	if err != nil {
		return
	}
	ps.remove(peer)
	// update metrics
	ps.metrics.UpdatePeerMetrics(len(ps.m), ps.inbound, ps.outbound)
	return
}

// UpdateMustConnects() updates the list of peers that 'must be connected to'
// Ex. the peers needed to complete committee consensus
func (ps *PeerSet) UpdateMustConnects(mustConnect []*lib.PeerAddress) (toDial []*lib.PeerAddress) {
	ps.Lock()
	defer ps.Unlock()
	ps.mustConnect = mustConnect
	for _, peer := range ps.m {
		if peer.IsMustConnect {
			ps.changeIOCount(true, peer.IsOutbound)
		}
		peer.IsMustConnect = false
	}
	// for each must connect
	for _, peer := range mustConnect {
		// ensure peer is not self
		if bytes.Equal(peer.PublicKey, ps.publicKey) {
			continue
		}
		publicKey := lib.BytesToString(peer.PublicKey)
		// if has peer, just update metadata
		if p, found := ps.m[publicKey]; found {
			ps.m[publicKey].IsMustConnect = true
			ps.changeIOCount(false, p.IsOutbound)
		} else { // else add to 'ToDial' list
			toDial = append(toDial, peer)
		}
	}
	return
}

// ChangeReputation() updates the peer reputation +/- based on the int32 delta
func (ps *PeerSet) ChangeReputation(publicKey []byte, delta int32) {
	ps.Lock()
	defer ps.Unlock()
	peer, err := ps.get(publicKey)
	if err != nil {
		return
	}
	// update the peers reputation
	peer.Reputation += delta
	// enforce maximum peer reputation
	if peer.Reputation >= MaxPeerReputation {
		peer.Reputation = MaxPeerReputation
	}
	// if peer isn't trusted nor is 'must connect' and the reputation is below minimum
	if !peer.IsTrusted && !peer.IsMustConnect && peer.Reputation < MinimumPeerReputation {
		ps.logger.Warnf("Peer %s reputation too low; removing", lib.BytesToTruncatedString(peer.Address.PublicKey))
		peer.stop.Do(func() {
			peer.conn.Stop()
		})
		ps.remove(peer)
		return
	}
	// update the peer
	ps.set(peer)
}

// GetPeerInfo() returns a copy of the authenticated information from the peer structure
func (ps *PeerSet) GetPeerInfo(publicKey []byte) (*lib.PeerInfo, lib.ErrorI) {
	ps.RLock()
	defer ps.RUnlock()
	peer, err := ps.get(publicKey)
	if err != nil {
		return nil, err
	}
	return peer.PeerInfo.Copy(), nil
}

// PeerCount() returns the total number of peers
func (ps *PeerSet) PeerCount() int {
	ps.RLock()
	defer ps.RUnlock()
	return len(ps.m)
}

// IsMustConnect() checks if a peer is on the must-connect list
func (ps *PeerSet) IsMustConnect(publicKey []byte) bool {
	ps.RLock()
	defer ps.RUnlock()
	// check if is must connect
	for _, item := range ps.mustConnect {
		if bytes.Equal(item.PublicKey, publicKey) {
			return true
		}
	}
	return false
}

// GetAllInfos() returns the information on connected peers and the total inbound / outbound counts
func (ps *PeerSet) GetAllInfos() (res []*lib.PeerInfo, numInbound, numOutbound int) {
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		if p.IsOutbound {
			numOutbound++
		} else {
			numInbound++
		}
		res = append(res, p.PeerInfo.Copy())
	}
	return
}

// SendToRandPeer() sends a message to any random peer on the list
func (ps *PeerSet) SendToRandPeer(topic lib.Topic, msg proto.Message) (*lib.PeerInfo, lib.ErrorI) {
	bz, err := lib.Marshal(msg)
	if err != nil {
		return nil, err
	}
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		return p.PeerInfo, ps.send(p, topic, bz)
	}
	return nil, nil
}

// SendTo() sends a message to a specific peer based on their public key
func (ps *PeerSet) SendTo(publicKey []byte, topic lib.Topic, msg proto.Message) lib.ErrorI {
	bz, err := lib.Marshal(msg)
	if err != nil {
		return err
	}
	ps.RLock()
	defer ps.RUnlock()
	peer, err := ps.get(publicKey)
	if err != nil {
		return err
	}
	return ps.send(peer, topic, bz)
}

// SendToPeers() sends a message to all peers
func (ps *PeerSet) SendToPeers(topic lib.Topic, msg proto.Message, excludeKeys ...string) lib.ErrorI {
	bz, err := lib.Marshal(msg)
	if err != nil {
		return err
	}
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		// exclude specific public keys to send to
		if slices.Contains(excludeKeys, lib.BytesToString(p.Address.PublicKey)) {
			continue
		}
		// send to peer
		if err = ps.send(p, topic, bz); err != nil {
			return err
		}
	}
	return nil
}

// Has() returns if the set has a peer with a specific public key
func (ps *PeerSet) Has(publicKey []byte) bool {
	ps.RLock()
	defer ps.RUnlock()
	pubKey := lib.BytesToString(publicKey)
	_, found := ps.m[pubKey]
	return found
}

// Stop() stops the entire peer set
func (ps *PeerSet) Stop() {
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		p.stop.Do(p.conn.Stop)
	}
}

// send() sends a message to a specific peer object
func (ps *PeerSet) send(peer *Peer, topic lib.Topic, bz []byte) lib.ErrorI {
	ps.logger.Debugf("Sending %s message to %s", topic, lib.BytesToTruncatedString(peer.Address.PublicKey))
	peer.conn.Send(topic, bz)
	return nil
}

// remove() decrements the in/out counters, and deletes it from the set
func (ps *PeerSet) remove(peer *Peer) {
	if !peer.IsTrusted && !peer.IsMustConnect {
		if peer.IsOutbound {
			ps.outbound--
		} else {
			ps.inbound--
		}
	}
	ps.del(peer.PeerInfo.Address.PublicKey)
}

// changeIOCount() increments or decrements numInbound and numOutbound
func (ps *PeerSet) changeIOCount(increment, outbound bool) {
	if outbound {
		if increment {
			ps.outbound++
		} else {
			ps.outbound--
		}
	} else {
		if increment {
			ps.inbound++
		} else {
			ps.inbound--
		}
	}
	ps.metrics.UpdatePeerMetrics(len(ps.m), ps.inbound, ps.outbound)
}

// map based CRUD operations below
func (ps *PeerSet) set(p *Peer)          { ps.m[lib.BytesToString(p.Address.PublicKey)] = p }
func (ps *PeerSet) del(publicKey []byte) { delete(ps.m, lib.BytesToString(publicKey)) }
func (ps *PeerSet) get(publicKey []byte) (*Peer, lib.ErrorI) {
	pub := lib.BytesToString(publicKey)
	peer, ok := ps.m[pub]
	if !ok {
		return nil, ErrPeerNotFound(pub)
	}
	return peer, nil
}
