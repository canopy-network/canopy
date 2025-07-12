package lib

import (
	"container/list"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file contains shared code for peers and messages that are routed by the controller throughout the app */

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
	// attempt to extract the net address from the second part of the split array
	netAddress, er := url.Parse(splitArr[1])
	// if an error occurred during the parsing
	if er != nil || netAddress.Hostname() == "" {
		// exit with 'invalid net address'
		return ErrInvalidNetAddress(stringFromConfig)
	}
	// get the port from the net address
	port := netAddress.Port()
	// resolve port automatically if not exists
	// port definition exists everywhere except for in state
	if port == "" {
		// resolve the port
		port, e = ResolvePort(x.PeerMeta.ChainId)
		// if an error occurred resolving the port
		if e != nil {
			// exit
			return
		}
	}
	// ensure the port starts with a colon
	if !strings.HasPrefix(port, ":") {
		// add the colon if not exists
		port = ":" + port
	}
	// remove the transport prefix (if exists) and set the NetAddress
	x.NetAddress = strings.ReplaceAll(netAddress.Hostname(), "tcp://", "") + port
	// set the PublicKey
	x.PublicKey = pubKey.Bytes()
	// exit
	return
}

// ResolvePort() executes a network wide protocol for determining what the p2p port of the peer is
// This is useful to allow 1 URL in state to expand to many routing paths for nested-chains
// Example: ResolvePort(CHAIN-ID = 2) returns 9002
func ResolvePort(chainId uint64) (string, ErrorI) {
	return AddToPort(":9000", chainId)
}

// ResolveAndReplacePort() resolves the appropriate port and replaces the port in the net address
func ResolveAndReplacePort(netAddress *string, chainId uint64) ErrorI {
	// resolve the port using the network wide protocol
	newPort, err := ResolvePort(chainId)
	if err != nil {
		return err
	}
	// remove the colon
	newPort = strings.Replace(newPort, ":", "", 1)
	// find the index of the final colon in the address
	i := strings.LastIndex(*netAddress, ":")
	// if no colon found
	if i == -1 {
		// return the original address with the new port appended
		*netAddress = *netAddress + ":" + newPort
		// exit with no error
		return nil
	}
	// return the address up to and including the colon, then the new port
	*netAddress = (*netAddress)[:i+1] + newPort
	// exit with no error
	return nil
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
	deDupe  *DeDuplicator[uint64] // the O(1) de-duplicator
	maxSize int                   // the max size before evicting the oldest
}

// NewMessageCache() initializes and returns a new MessageCache instance
func NewMessageCache() *MessageCache {
	return &MessageCache{
		queue:   list.New(),
		deDupe:  NewDeDuplicator[uint64](),
		maxSize: 10000,
	}
}

// Add inserts a new message into the cache if it doesn't already exist
// It removes the oldest message if the cache is full
func (c *MessageCache) Add(msg *MessageAndMetadata) (ok bool) {
	// create a key for the message
	key := MemHash(msg.Message)
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
		toDeleteKey := MemHash(message.Message)
		// delete it from the underlying de-duplicator
		c.deDupe.Delete(toDeleteKey)
		// remove it from the queue
		c.queue.Remove(e)
	}
	// exit with 'added'
	return true
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
