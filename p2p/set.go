package p2p

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
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
	inbound      map[uint64]int     // inbound count
	outbound     map[uint64]int     // outbound count
	sync.RWMutex                    // read / write mutex
	config       lib.P2PConfig      // p2p configuration
	publicKey    []byte             // self public key
	logger       lib.LoggerI
}

func NewPeerSet(c lib.Config, priv crypto.PrivateKeyI, logger lib.LoggerI) PeerSet {
	inbound, outbound := make(map[uint64]int), make(map[uint64]int)
	for _, p := range c.Plugins {
		inbound[p.ID], outbound[p.ID] = 0, 0
	}
	return PeerSet{
		m:           make(map[string]*Peer),
		mustConnect: make([]*lib.PeerAddress, 0),
		inbound:     inbound,
		outbound:    outbound,
		RWMutex:     sync.RWMutex{},
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
	ps.Lock()
	defer ps.Unlock()
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
	// use a ptr to reference the inbound / outbound counters
	peerCounts, maxErr := new(map[uint64]int), lib.ErrorI(nil)
	if p.IsOutbound {
		peerCounts, maxErr = &ps.outbound, ErrMaxOutbound()
	} else {
		peerCounts, maxErr = &ps.inbound, ErrMaxInbound()
	}
	// limit inbound / outbound on non-trusted & non-must-connects
	// for each chain the peer supports
	for _, chain := range p.Address.PeerMeta.Chains {
		// check if below limit and self hasChain
		if count, selfHasChain := (*peerCounts)[chain]; selfHasChain && count < ps.config.MaxOutbound {
			// increment counts
			for _, c := range p.Address.PeerMeta.Chains {
				(*peerCounts)[c]++
			}
			// set the peer
			ps.set(p)
			return nil
		}
	}
	// all chains are at or above limit
	return maxErr
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
	return
}

// UpdateMustConnects() updates the list of peers that 'must be connected to'
// Ex. the peers needed to complete committee consensus
func (ps *PeerSet) UpdateMustConnects(mustConnect []*lib.PeerAddress) (toDial []*lib.PeerAddress) {
	ps.Lock()
	defer ps.Unlock()
	ps.mustConnect = mustConnect
	for _, peer := range ps.m {
		peer.IsMustConnect = false
		ps.changeIOCount(false, peer.IsOutbound, peer.PeerInfo.Address.PeerMeta)
	}
	// for each must connect
	for _, peer := range mustConnect {
		// ensure peer is not self
		if bytes.Equal(peer.PublicKey, ps.publicKey) {
			return nil
		}
		publicKey := lib.BytesToString(peer.PublicKey)
		// if has peer, just update metadata
		if p, found := ps.m[publicKey]; found {
			ps.m[publicKey].IsMustConnect = true
			ps.changeIOCount(true, p.IsOutbound, p.PeerInfo.Address.PeerMeta)
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
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		return p.Copy(), ps.send(p, topic, msg)
	}
	return nil, nil
}

// SendTo() sends a message to a specific peer based on their public key
func (ps *PeerSet) SendTo(publicKey []byte, topic lib.Topic, msg proto.Message) lib.ErrorI {
	ps.RLock()
	defer ps.RUnlock()
	peer, err := ps.get(publicKey)
	if err != nil {
		return err
	}
	return ps.send(peer, topic, msg)
}

// SendToChainPeers() sends a message to all peers with the chainId
func (ps *PeerSet) SendToChainPeers(chainId uint64, topic lib.Topic, msg proto.Message, excludeBadRep ...bool) lib.ErrorI {
	ps.RLock()
	defer ps.RUnlock()
	for _, p := range ps.m {
		if p.HasChain(chainId) {
			if len(excludeBadRep) == 1 && excludeBadRep[0] {
				if p.Reputation < MinimumPeerReputation {
					continue
				}
			}
			if err := ps.send(p, topic, msg); err != nil {
				return err
			}
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
func (ps *PeerSet) send(peer *Peer, topic lib.Topic, msg proto.Message) lib.ErrorI {
	a, err := lib.NewAny(msg)
	if err != nil {
		return err
	}
	ps.logger.Debugf("Sending %s message to %s", topic, lib.BytesToTruncatedString(peer.Address.PublicKey))
	peer.conn.Send(topic, &Envelope{Payload: a})
	return nil
}

// remove() decrements the in/out counters, and deletes it from the set
func (ps *PeerSet) remove(peer *Peer) {
	if !peer.IsTrusted && !peer.IsMustConnect {
		for _, chain := range peer.Address.PeerMeta.Chains {
			if peer.IsOutbound {
				if _, selfHasChain := ps.outbound[chain]; selfHasChain {
					ps.outbound[chain]--
				}
			} else {
				if _, selfHasChain := ps.inbound[chain]; selfHasChain {
					ps.inbound[chain]--
				}
			}
		}
	}
	ps.del(peer.PeerInfo.Address.PublicKey)
}

// changeIOCount() increments or decrements numInbound and numOutbound for each chain in the PeerMeta
func (ps *PeerSet) changeIOCount(increment, outbound bool, meta *lib.PeerMeta) {
	for _, c := range meta.Chains {
		if outbound {
			if increment {
				ps.outbound[c]++
			} else {
				ps.outbound[c]--
			}
		} else {
			if increment {
				ps.inbound[c]++
			} else {
				ps.inbound[c]--
			}
		}
	}
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
