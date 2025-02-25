package lib

import (
	"container/list"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
)

// MESSAGE CODE BELOW

// Channels are logical communication paths or streams that operate over a single 'multiplexed' network connection
type Channels map[Topic]chan *MessageAndMetadata

// MessageAndMetadata is a wrapper over a P2P message with information about the sender and the hash of the message
// for easy de-duplication at the module level
type MessageAndMetadata struct {
	Message proto.Message
	Hash    []byte
	Sender  *PeerInfo
}

// WithHash() fills the hash field with the cryptographic hash of the message (used for de-duplication)
func (x *MessageAndMetadata) WithHash() *MessageAndMetadata {
	bz, _ := MarshalJSON(x.Message)
	x.Hash = crypto.Hash(bz)
	return x
}

// MessageCache is a simple p2p message de-duplicator that protects redundancy in the p2p network
type MessageCache struct {
	queue   *list.List
	m       map[string]struct{}
	maxSize int
}

// NewMessageCache() initializes and returns a new MessageCache instance
func NewMessageCache() *MessageCache {
	return &MessageCache{
		queue:   list.New(),
		m:       map[string]struct{}{},
		maxSize: 10000,
	}
}

// Add inserts a new message into the cache if it doesn't already exist
// It removes the oldest message if the cache is full
func (c *MessageCache) Add(msg *MessageAndMetadata) (ok bool) {
	k := BytesToString(msg.Hash)
	if _, found := c.m[k]; found {
		return false
	}
	if c.queue.Len() >= c.maxSize {
		e := c.queue.Back()
		message := e.Value.(*MessageAndMetadata)
		delete(c.m, BytesToString(message.Hash))
		c.queue.Remove(e)
	}
	c.m[k] = struct{}{}
	c.queue.PushFront(msg)
	return true
}

// MESSAGE LIMITERS BELOW

// SimpleLimiter ensures the number of requests don't exceed
// a total limit and a limit per requester during a timeframe
type SimpleLimiter struct {
	requests        map[string]int
	totalRequests   int
	maxPerRequester int
	maxRequests     int
	reset           *time.Ticker
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
	if l.totalRequests >= l.maxRequests {
		return false, true
	}
	if count := l.requests[requester]; count >= l.maxPerRequester {
		return true, false
	}
	l.requests[requester]++
	l.totalRequests++
	return
}

// Reset() clears the requests and resets the total request count
func (l *SimpleLimiter) Reset() {
	l.requests = map[string]int{}
	l.totalRequests = 0
}

// TimeToReset() returns the channel that signals when the limiter may be reset
// This channel is called by the time.Ticker() set in NewLimiter
func (l *SimpleLimiter) TimeToReset() <-chan time.Time {
	return l.reset.C
}

// PEER ADDRESS CODE BELOW

// Copy() returns a clone of the PeerAddress
func (x *PeerAddress) Copy() *PeerAddress {
	pkCopy := make([]byte, len(x.PublicKey))
	copy(pkCopy, x.PublicKey)
	return &PeerAddress{
		PublicKey:  pkCopy,
		NetAddress: x.NetAddress,
		PeerMeta:   x.PeerMeta.Copy(),
	}
}

// FromString() creates a new PeerAddress object from string (without meta)
// Peer String example: <some-public-key>@<some-net-address>
func (x *PeerAddress) FromString(s string) (e ErrorI) {
	arr := strings.Split(s, "@")
	if len(arr) != 2 {
		return ErrInvalidNetAddrString(s)
	}
	pubKey, err := crypto.NewPublicKeyFromString(arr[0])
	if err != nil {
		return ErrInvalidNetAddressPubKey(arr[0])
	}
	u, er := url.Parse(arr[1])
	if er != nil || u.Hostname() == "" {
		return ErrInvalidNetAddress(s)
	}
	port := u.Port()
	// resolve port automatically if not exists
	// port definition exists everywhere except for in state
	if port == "" {
		port, e = ResolvePort(x.PeerMeta.ChainId)
		if e != nil {
			return
		}
	}
	// ensure the port starts with a colon
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	x.NetAddress = strings.ReplaceAll(u.Hostname(), "tcp://", "") + port
	x.PublicKey = pubKey.Bytes()
	return
}

// ResolvePort() executes a network wide protocol for determining what the p2p port of the peer is
// This is useful to allow 1 URL in state to expand to many different routing paths for nested-chains
// Example: ResolvePort(CHAIN-ID = 2) returns 9002
func ResolvePort(chainId uint64) (string, ErrorI) {
	return AddToPort(":9000", chainId)
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
func (x *PeerAddress) UnmarshalJSON(bz []byte) error {
	j := new(peerAddressJSON)
	if err := json.Unmarshal(bz, j); err != nil {
		return err
	}
	x.PublicKey, x.NetAddress, x.PeerMeta = j.PublicKey, j.NetAddress, j.PeerMeta
	return nil
}

// PEER META CODE BELOW

// Sign() adds a digital signature to the PeerMeta for remote public key verification
func (x *PeerMeta) Sign(key crypto.PrivateKeyI) *PeerMeta {
	x.Signature = key.Sign(x.SignBytes())
	return x
}

// SignBytes() returns the canonical byte representation used to digitally sign the bytes
func (x *PeerMeta) SignBytes() []byte {
	sig := x.Signature
	x.Signature = nil
	bz, _ := Marshal(x)
	x.Signature = sig
	return bz
}

// Copy() returns a reference to a clone of the PeerMeta
func (x *PeerMeta) Copy() *PeerMeta {
	if x == nil {
		return nil
	}
	return &PeerMeta{
		NetworkId: x.NetworkId,
		ChainId:   x.ChainId,
		Signature: slices.Clone(x.Signature),
	}
}

// PEER INFO CODE BELOW

// Copy() returns a reference to a clone of the PeerInfo
func (x *PeerInfo) Copy() *PeerInfo {
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
