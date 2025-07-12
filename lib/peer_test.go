package lib

import (
	"container/list"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/runtime/protoimpl"
)

func TestMessageCacheAdd(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		cache      MessageCache
		toAdd      *MessageAndMetadata
		expectedOk bool
		expected   map[uint64]struct{}
	}{
		{
			name:   "exists",
			detail: "not added as message already exists in cache",
			cache: MessageCache{
				queue: list.New(),
				deDupe: &DeDuplicator[uint64]{m: map[uint64]struct{}{
					func() uint64 {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return MemHash(bz)
					}(): {},
				}},
				maxSize: 2,
			},
			toAdd: &MessageAndMetadata{
				Message: func() []byte {
					bz, err := Marshal(&StringWrapper{Value: "b"})
					require.NoError(t, err)
					return bz
				}(),
			},
			expected: map[uint64]struct{}{
				func() uint64 {
					bz, _ := Marshal(&StringWrapper{Value: "b"})
					return MemHash(bz)
				}(): {},
			},
			expectedOk: false,
		},
		{
			name:   "ok",
			detail: "added without eviction",
			cache: MessageCache{
				queue: func() (l *list.List) {
					l = list.New()
					l.PushFront(&MessageAndMetadata{
						Message: func() []byte {
							bz, err := Marshal(&StringWrapper{Value: "b"})
							require.NoError(t, err)
							return bz
						}(),
					})
					return
				}(),
				deDupe: &DeDuplicator[uint64]{m: map[uint64]struct{}{
					func() uint64 {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return MemHash(bz)
					}(): {},
				}},
				maxSize: 2,
			},
			toAdd: &MessageAndMetadata{
				Message: func() []byte {
					bz, err := Marshal(&StringWrapper{Value: "c"})
					require.NoError(t, err)
					return bz
				}(),
			},
			expected: map[uint64]struct{}{
				func() uint64 {
					bz, _ := Marshal(&StringWrapper{Value: "b"})
					return MemHash(bz)
				}(): {},
				func() uint64 {
					bz, _ := Marshal(&StringWrapper{Value: "c"})
					return MemHash(bz)
				}(): {},
			},
			expectedOk: true,
		},
		{
			name:   "max size",
			detail: "added and evicted the old",
			cache: MessageCache{
				queue: func() (l *list.List) {
					l = list.New()
					l.PushFront(&MessageAndMetadata{
						Message: func() []byte {
							bz, err := Marshal(&StringWrapper{Value: "b"})
							require.NoError(t, err)
							return bz
						}(),
					})
					return
				}(),
				deDupe: &DeDuplicator[uint64]{m: map[uint64]struct{}{
					func() uint64 {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return MemHash(bz)
					}(): {},
				}},
				maxSize: 1,
			},
			toAdd: &MessageAndMetadata{
				Message: func() []byte {
					bz, err := Marshal(&StringWrapper{Value: "c"})
					require.NoError(t, err)
					return bz
				}(),
			},
			expected: map[uint64]struct{}{
				func() uint64 {
					bz, _ := Marshal(&StringWrapper{Value: "c"})
					return MemHash(bz)
				}(): {},
			},
			expectedOk: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			require.Equal(t, test.expectedOk, test.cache.Add(test.toAdd))
			// compare got vs expected
			require.Equal(t, test.expected, test.cache.deDupe.Map())
		})
	}
}

func TestSimpleLimiterNewRequest(t *testing.T) {
	tests := []struct {
		name                   string
		detail                 string
		limiter                SimpleLimiter
		requester              string
		expectedRequesterBlock bool
		expectedTotalBlock     bool
	}{
		{
			name:   "requester block",
			detail: "max for a requester exceeded",
			limiter: SimpleLimiter{
				requests: map[string]int{
					"a": 1,
				},
				totalRequests:   1,
				maxPerRequester: 1,
				maxRequests:     3,
			},
			requester:              "a",
			expectedRequesterBlock: true,
			expectedTotalBlock:     false,
		},
		{
			name:   "all block",
			detail: "max for all requesters exceeded",
			limiter: SimpleLimiter{
				requests: map[string]int{
					"b": 1,
				},
				totalRequests:   1,
				maxPerRequester: 1,
				maxRequests:     1,
			},
			requester:              "b",
			expectedRequesterBlock: false,
			expectedTotalBlock:     true,
		},
		{
			name:   "no block",
			detail: "no limits exceeded",
			limiter: SimpleLimiter{
				requests:        map[string]int{},
				totalRequests:   0,
				maxPerRequester: 1,
				maxRequests:     1,
			},
			requester:              "b",
			expectedRequesterBlock: false,
			expectedTotalBlock:     false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			gotRequesterBlock, gotTotalBlock := test.limiter.NewRequest(test.requester)
			// check got vs expected
			require.Equal(t, test.expectedRequesterBlock, gotRequesterBlock)
			require.Equal(t, test.expectedTotalBlock, gotTotalBlock)
		})
	}
}

func TestLimiterReset(t *testing.T) {
	// initialize limiter
	limiter := SimpleLimiter{
		requests:      map[string]int{"a": 1},
		totalRequests: 1,
		reset:         time.NewTicker(500 * time.Millisecond),
	}
	// wait for reset ticker
out:
	for {
		select {
		case <-limiter.TimeToReset():
			limiter.Reset()
			break out
		case <-time.Tick(time.Second):
			t.Fatal("timeout")
		}
	}
	// validate reset
	require.Len(t, limiter.requests, 0)
	require.Equal(t, limiter.totalRequests, 0)
}

func TestPeerAddressFromString(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		s        string
		error    string
		expected *PeerAddress
	}{
		{
			name:   "bad format",
			detail: "the address string is missing an @ sign",
			s:      "https://wrong-format.com",
			error:  "invalid net address string",
		},
		{
			name:   "bad public key",
			detail: "the address string is missing an @ sign",
			s:      newTestAddress(t).String() + "@tcp://0.0.0.0:8080",
			error:  "invalid net address public key",
		},
		{
			name:   "port resolution",
			detail: "the address string is missing an @ sign",
			s:      newTestPublicKey(t).String() + "@tcp://0.0.0.0",
			expected: &PeerAddress{
				PeerMeta:   &PeerMeta{ChainId: 1},
				PublicKey:  newTestPublicKeyBytes(t),
				NetAddress: "0.0.0.0:9001",
			},
		},
		{
			name:   "valid url in string",
			detail: "the address string is missing an @ sign",
			s:      newTestPublicKey(t).String() + "@tcp://0.0.0.0:8080",
			expected: &PeerAddress{
				PeerMeta:   &PeerMeta{ChainId: 1},
				PublicKey:  newTestPublicKeyBytes(t),
				NetAddress: "0.0.0.0:8080",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a peer address object
			peerAddress := new(PeerAddress)
			peerAddress.PeerMeta = &PeerMeta{ChainId: 1}
			// execute the function call
			err := peerAddress.FromString(test.s)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error, err)
				return
			}
			// check got vs expected
			require.EqualExportedValues(t, test.expected, peerAddress)
		})
	}
}

func TestHasChain(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		peerAddress PeerAddress
		chain       uint64
		has         bool
	}{
		{
			name:   "peer isn't on the chain",
			detail: "peer meta doesn't contain the chain id",
			peerAddress: PeerAddress{
				PeerMeta: &PeerMeta{
					ChainId: 1,
				},
			},
			chain: 0,
			has:   false,
		},
		{
			name:   "peer is on the chain",
			detail: "peer meta contains the chain id",
			peerAddress: PeerAddress{
				PeerMeta: &PeerMeta{
					ChainId: 3,
				},
			},
			chain: 3,
			has:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			require.Equal(t, test.has, test.peerAddress.HasChain(test.chain))
		})
	}
}

func TestPeerAddressCopy(t *testing.T) {
	expected := &PeerAddress{
		PublicKey:  newTestPublicKeyBytes(t),
		NetAddress: "8.8.8.8:8080",
		PeerMeta: &PeerMeta{
			NetworkId: 1,
			ChainId:   1,
			Signature: []byte("sig"),
		},
	}
	// make a copy
	got := expected.Copy()
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestPeerAddressJSON(t *testing.T) {
	expected := &PeerAddress{
		PublicKey:  newTestPublicKeyBytes(t),
		NetAddress: "8.8.8.8:8080",
		PeerMeta: &PeerMeta{
			NetworkId: 1,
			ChainId:   1,
			Signature: []byte("sig"),
		},
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(PeerAddress)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestPeerMetaJSON(t *testing.T) {
	expected := &PeerMeta{
		NetworkId: 1,
		ChainId:   1,
		Signature: []byte("sig"),
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(PeerMeta)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestPeerMetaCopy(t *testing.T) {
	expected := &PeerMeta{
		NetworkId: 1,
		ChainId:   1,
		Signature: []byte("sig"),
	}
	// make a copy
	got := expected.Copy()
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestPeerInfoJSON(t *testing.T) {
	expected := &PeerInfo{
		Address: &PeerAddress{
			state:         protoimpl.MessageState{},
			sizeCache:     0,
			unknownFields: nil,
			PublicKey:     newTestPublicKeyBytes(t),
			NetAddress:    "8.8.8.8:8080",
			PeerMeta: &PeerMeta{
				NetworkId: 1,
				ChainId:   1,
				Signature: []byte("sig"),
			},
		},
		IsOutbound:    true,
		IsMustConnect: true,
		IsTrusted:     true,
		Reputation:    1,
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(PeerInfo)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestPeerInfoCopy(t *testing.T) {
	expected := &PeerInfo{
		Address: &PeerAddress{
			state:         protoimpl.MessageState{},
			sizeCache:     0,
			unknownFields: nil,
			PublicKey:     newTestPublicKeyBytes(t),
			NetAddress:    "8.8.8.8:8080",
			PeerMeta: &PeerMeta{
				NetworkId: 1,
				ChainId:   1,
				Signature: []byte("sig"),
			},
		},
		IsOutbound:    true,
		IsMustConnect: true,
		IsTrusted:     true,
		Reputation:    1,
	}
	// make a copy
	got := expected.Copy()
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}
