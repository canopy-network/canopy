package lib

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file contains shared code for peers and messages that are routed by the controller throughout the app */

const (
	DefaultPort = "9000" // default port when not specified
)

// MESSAGE CODE BELOW

// Channels are logical communication paths or streams that operate over a single 'multiplexed' network connection
type Channels map[Topic]chan *MessageAndMetadata

// MessageAndMetadata is a wrapper over a P2P message with information about the sender
type MessageAndMetadata struct {
	Message []byte    // the (proto) payload of the message
	Sender  *PeerInfo // the sender information
}

// PEER ADDRESS CODE BELOW

// Copy() returns a deep clone of the PeerAddress
func (x *PeerAddress) Copy() *PeerAddress {
	// make a destination for a copy of the peer's public key
	pkCopy := make([]byte, len(x.PublicKey))
	// copy the public key to the destination
	copy(pkCopy, x.PublicKey)
	// return a deep copy of the peer address
	return &PeerAddress{
		PublicKey:  pkCopy,
		NetAddress: x.NetAddress,
		PeerMeta:   x.PeerMeta.Copy(),
	}
}

// FromString() creates a new PeerAddress object from string (without meta)
// Peer String example: <some-public-key>@<some-net-address>
func (x *PeerAddress) FromString(stringFromConfig string) (e ErrorI) {
	// split the string from the config file using the @ delimiter
	splitArr := strings.Split(stringFromConfig, "@")
	// if the split isn't length 2 (public key + net address)
	if len(splitArr) != 2 {
		// exit with invalid format
		return ErrInvalidNetAddrString(stringFromConfig)
	}
	// try to extract the public key from the first item of the split
	pubKey, err := crypto.NewPublicKeyFromString(splitArr[0])
	// if an error occurred during the conversion
	if err != nil {
		// exit with invalid public key
		return ErrInvalidNetAddressPubKey(splitArr[0])
	}
	// get the port from the net address
	x.NetAddress = splitArr[1]
	// resolve port automatically if not exists
	if e = ResolveAndReplacePort(&x.NetAddress, x.PeerMeta.ChainId); e != nil {
		return e
	}
	// set the PublicKey
	x.PublicKey = pubKey.Bytes()
	// exit
	return
}

// ResolvePort() executes a network wide protocol for determining what the p2p port of the peer is
// This is useful to allow 1 URL in state to expand to many routing paths for nested-chains
// Example: ResolvePort(CHAIN-ID = 2) with original port 9000 returns 9002
func ResolvePort(oldPort string, chainId uint64) (string, ErrorI) {
	if oldPort != "" {
		return AddToPort(strings.ReplaceAll(oldPort, ":", ""), chainId)
	}
	//TODO review if max chainID should be limited for now to 56,535, combined with defaultPort, or 64,510, combined with the lower bound port (1025)
	//any higher value will return bad port error
	return AddToPort(DefaultPort, chainId)
}

// ResolveAndReplacePort resolves the appropriate port and replaces the port in the net address
func ResolveAndReplacePort(netAddress *string, chainId uint64) (error ErrorI) {
	if netAddress == nil {
		return ErrInvalidNetAddress("<nil>")
	}
	// capture the input
	input := *netAddress
	// replace any prefix
	if strings.HasPrefix(input, "tcp://") {
		input = strings.TrimPrefix(input, "tcp://")
	}
	// extract the host and port
	host, port, err := net.SplitHostPort(input)
	if err != nil {
		// if err isn't about a missing port
		if !strings.Contains(err.Error(), "missing port in address") {
			return ErrInvalidNetAddress(err.Error())
		}
		// resolve the new port
		newPort, e := ResolvePort("", chainId)
		if e != nil {
			return e
		}
		// unwrap IPv6 brackets if present, e.g., "[2001:db8::1]" -> "2001:db8::1"
		*netAddress = net.JoinHostPort(strings.Trim(input, "[]"), newPort)
		return
	}
	// host and port were parsed successfully
	newPort, e := ResolvePort(":"+port, chainId)
	if e != nil {
		return e
	}
	// set the input
	*netAddress = net.JoinHostPort(host, newPort)
	// exit
	return
}

// HasChain() returns if the PeerAddress's PeerMeta has this chain
func (x *PeerAddress) HasChain(id uint64) bool { return x.PeerMeta.ChainId == id }

// peerAddressJSON is the json.Marshaller and json.Unmarshaler representation fo the PeerAddress object
type peerAddressJSON struct {
	PublicKey  HexBytes  `json:"publicKey,omitempty"`
	NetAddress string    `json:"netAddress,omitempty"`
	PeerMeta   *PeerMeta `json:"peerMeta,omitempty"`
}

// MarshalJSON satisfies the json.Marshaller interface for PeerAddress
func (x PeerAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(peerAddressJSON{
		PublicKey:  x.PublicKey,
		NetAddress: x.NetAddress,
		PeerMeta:   x.PeerMeta,
	})
}

// UnmarshalJSON satisfies the json.Unmarshlaer interface for PeerAddress
func (x *PeerAddress) UnmarshalJSON(jsonBytes []byte) (err error) {
	// make a new json object reference to ensure a non nil result
	j := new(peerAddressJSON)
	// populate the json object using the json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying object using the json object
	x.PublicKey, x.NetAddress, x.PeerMeta = j.PublicKey, j.NetAddress, j.PeerMeta
	// exit
	return
}

// PEER META CODE BELOW

// Sign() adds a digital signature to the PeerMeta for remote public key verification
func (x *PeerMeta) Sign(key crypto.PrivateKeyI) *PeerMeta {
	// sign the peer meta and populate the signature field
	x.Signature = key.Sign(x.SignBytes())
	// return the meta
	return x
}

// SignBytes() returns the canonical byte representation used to digitally sign the bytes
func (x *PeerMeta) SignBytes() (signBytes []byte) {
	// save the signature in a temporary variable
	temp := x.Signature
	// nullify the signature
	x.Signature = nil
	// convert the structure into proto bytes
	signBytes, _ = Marshal(x)
	// set the signature back into the object
	x.Signature = temp
	// exit
	return
}

// Copy() returns a reference to a clone of the PeerMeta
func (x *PeerMeta) Copy() *PeerMeta {
	// if the peer meta is nil, return nil
	if x == nil {
		return nil
	}
	// exit deep copy of the peer meta
	return &PeerMeta{
		NetworkId: x.NetworkId,
		ChainId:   x.ChainId,
		Signature: slices.Clone(x.Signature),
	}
}

// PEER INFO CODE BELOW

// Copy() returns a reference to a clone of the PeerInfo
func (x *PeerInfo) Copy() *PeerInfo {
	// exit with a deep copy of the peer info
	return &PeerInfo{
		Address:       x.Address.Copy(),
		IsOutbound:    x.IsOutbound,
		IsMustConnect: x.IsMustConnect,
		IsTrusted:     x.IsTrusted,
		Reputation:    x.Reputation,
	}
}

// HasChain() returns if the PeerInfo has a chain under the PeerAddresses' PeerMeta
func (x *PeerInfo) HasChain(id uint64) bool { return x.Address.HasChain(id) }

// MarshalJSON satisfies the json.Marshaller interface for PeerInfo
func (x PeerInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(peerInfoJSON{
		Address:       x.Address,
		IsOutbound:    x.IsOutbound,
		IsValidator:   x.IsMustConnect,
		IsMustConnect: x.IsMustConnect,
		IsTrusted:     x.IsTrusted,
		Reputation:    x.Reputation,
	})
}

// peerInfoJSON is the json marshaller and unmarshaler representation of PeerInfo
type peerInfoJSON struct {
	Address       *PeerAddress `json:"address"`
	IsOutbound    bool         `json:"isOutbound"`
	IsValidator   bool         `json:"isValidator"`
	IsMustConnect bool         `json:"isMustConnect"`
	IsTrusted     bool         `json:"isTrusted"`
	Reputation    int32        `json:"reputation"`
}

// MessageCache is a simple p2p message de-duplicator that protects redundancy in the p2p network
type MessageCache struct {
	queue   *list.List            // a FIFO list of MessageAndMetadata
	deDupe  *DeDuplicator[string] // the O(1) de-duplicator
	maxSize int                   // the max size before evicting the oldest
}

// NewMessageCache() initializes and returns a new MessageCache instance
func NewMessageCache() *MessageCache {
	return &MessageCache{
		queue:   list.New(),
		deDupe:  NewDeDuplicator[string](),
		maxSize: 10000,
	}
}

// Add inserts a new message into the cache if it doesn't already exist
// It removes the oldest message if the cache is full
func (c *MessageCache) Add(msg *MessageAndMetadata) (ok bool) {
	// create a key for the message
	key := crypto.HashString(msg.Message)
	// check / add to the de-duplicator to ensure no duplicates
	if c.deDupe.Found(key) {
		// exit with 'already found'
		return false
	}
	// add the new message to the front
	c.queue.PushFront(msg)
	// if the queue size is exceeded
	if c.queue.Len() > c.maxSize {
		// get the oldest element
		e := c.queue.Back()
		// cast it to a MessageAndMetadata
		message := e.Value.(*MessageAndMetadata)
		// create a key for the message
		toDeleteKey := crypto.HashString(message.Message)
		// delete it from the underlying de-duplicator
		c.deDupe.Delete(toDeleteKey)
		// remove it from the queue
		c.queue.Remove(e)
	}
	// exit with 'added'
	return true
}

// NewHeightTracker() detects if a node has fallen out of sync based on peer info
type NewHeightTracker struct {
	Peers  map[string]struct{} // peers (peer + height) that claimed new height with a valid block
	Blocks map[string]uint64   // pre-validated blocks showing new height
	syncCB func()              // the callback to start syncing
	l      LoggerI
}

// NewBlockTracker() constructs a NewHeightTracker
func NewBlockTracker(syncCB func(), l LoggerI) (n *NewHeightTracker) {
	return &NewHeightTracker{
		Peers:  make(map[string]struct{}),
		Blocks: make(map[string]uint64),
		syncCB: syncCB,
		l:      l,
	}
}

// Add() records a 'new height' for a sender and height
func (n *NewHeightTracker) Add(sender, message []byte, height uint64, peerCount int) (outOfSync bool) {
	// create key
	newHeightKey := fmt.Sprintf("%s/%d", BytesToString(sender), height)
	// add to peers
	n.Peers[newHeightKey] = struct{}{}
	// add to blocks
	n.Blocks[crypto.HashString(message)] = height
	// if num of peers claiming new height is >= 1/3 of total peers
	if float64(len(n.Peers)) >= float64(peerCount)/float64(3) {
		// reset the counter
		n.Reset()
		// log out of sync
		n.l.Error("Detected node has fallen out of sync")
		// start syncing
		go n.syncCB()
		// set 'out of sync'
		outOfSync = true
	}
	return
}

// AddIfHas() records a 'new height' for a sender and height only if it already contains this block
func (n *NewHeightTracker) AddIfHas(sender, message []byte, peerCount int) (outOfSync bool) {
	//	if already has this block
	if height, found := n.Blocks[crypto.HashString(message)]; found {
		outOfSync = n.Add(sender, message, height, peerCount)
	}
	// exit
	return
}

// Reset() resets the new height tracker
func (n *NewHeightTracker) Reset() {
	n.Peers, n.Blocks = make(map[string]struct{}), make(map[string]uint64)
}

// MESSAGE LIMITERS BELOW

// SimpleLimiter ensures the number of requests don't exceed
// a total limit and a limit per requester during a timeframe
type SimpleLimiter struct {
	requests        map[string]int // [requester_id] -> number_of_requests
	totalRequests   int            // total requests from all requesters
	maxPerRequester int            // config: max requests per requester
	maxRequests     int            // config: max total requests
	reset           *time.Ticker   // a timer that indicates the caller to 'reset' the limiter
}

// NewLimiter() returns a new instance of SimpleLimiter with
// - max requests per requester
// - max total requests
// - how often to reset the limiter
func NewLimiter(maxPerRequester, maxRequests, resetWindowS int) *SimpleLimiter {
	return &SimpleLimiter{
		requests:        map[string]int{},
		maxPerRequester: maxPerRequester,
		maxRequests:     maxRequests,
		reset:           time.NewTicker(time.Duration(resetWindowS) * time.Second),
	}
}

// NewRequest() processes a new request and checks if the requester or total requests should be blocked
func (l *SimpleLimiter) NewRequest(requester string) (requesterBlock, totalBlock bool) {
	// if the total requests exceed the max requests
	if l.totalRequests >= l.maxRequests {
		// exit with 'block every requester'
		return false, true
	}
	// if the count of requests for this requester is larger than the max per requester
	if count := l.requests[requester]; count >= l.maxPerRequester {
		// exit with 'block this requester'
		return true, false
	}
	// add to the requests for this requester
	l.requests[requester]++
	// add to the total requests
	l.totalRequests++
	// exit
	return
}

// Reset() clears the requests and resets the total request count
func (l *SimpleLimiter) Reset() {
	// reset the requester -> requestCounts
	l.requests = map[string]int{}
	// reset the total requests
	l.totalRequests = 0
}

// TimeToReset() returns the channel that signals when the limiter may be reset
// This channel is called by the time.Ticker() set in NewLimiter
func (l *SimpleLimiter) TimeToReset() <-chan time.Time { return l.reset.C }

// Equals() compares the equality of two peer addresses
func (x *PeerAddress) Equals(y *PeerAddress) bool {
	if x == nil || y == nil {
		return false
	}
	// check if public keys are not equal
	if !bytes.Equal(x.PublicKey, y.PublicKey) {
		return false
	}
	// check if net address is not equal
	if x.NetAddress != y.NetAddress {
		return false
	}
	// check if peer metas are equal
	return x.PeerMeta.Equals(y.PeerMeta)
}

// Equals() compares the equality of two peer metas
func (x *PeerMeta) Equals(y *PeerMeta) bool {
	if x == nil || y == nil {
		return false
	}
	// if network ids aren't equal
	if x.NetworkId != y.NetworkId {
		return false
	}
	// if chain ids are not equal
	if x.ChainId != y.ChainId {
		return false
	}
	return true
}
