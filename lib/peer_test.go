package lib

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"

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
		expected   map[string]struct{}
	}{
		{
			name:   "exists",
			detail: "not added as message already exists in cache",
			cache: MessageCache{
				queue: list.New(),
				deDupe: &DeDuplicator[string]{m: map[string]struct{}{
					func() string {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return crypto.HashString(bz)
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
			expected: map[string]struct{}{
				func() string {
					bz, _ := Marshal(&StringWrapper{Value: "b"})
					return crypto.HashString(bz)
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
				deDupe: &DeDuplicator[string]{m: map[string]struct{}{
					func() string {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return crypto.HashString(bz)
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
			expected: map[string]struct{}{
				func() string {
					bz, _ := Marshal(&StringWrapper{Value: "b"})
					return crypto.HashString(bz)
				}(): {},
				func() string {
					bz, _ := Marshal(&StringWrapper{Value: "c"})
					return crypto.HashString(bz)
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
				deDupe: &DeDuplicator[string]{m: map[string]struct{}{
					func() string {
						bz, _ := Marshal(&StringWrapper{Value: "b"})
						return crypto.HashString(bz)
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
			expected: map[string]struct{}{
				func() string {
					bz, _ := Marshal(&StringWrapper{Value: "c"})
					return crypto.HashString(bz)
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
				log:             NewDefaultLogger(),
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
				log:             NewDefaultLogger(),
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
				log:             NewDefaultLogger(),
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
			detail: "valid url",
			s:      newTestPublicKey(t).String() + "@tcp://0.0.0.0:8080",
			expected: &PeerAddress{
				PeerMeta:   &PeerMeta{ChainId: 1},
				PublicKey:  newTestPublicKeyBytes(t),
				NetAddress: "0.0.0.0:8081",
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

func TestResolveAndReplacePort(t *testing.T) {
	tests := []struct {
		name        string
		netaddr     string
		expected    string
		chainId     uint64
		expectedErr string
	}{
		{
			name:        "invalid net address: bad port",
			netaddr:     "tcp://:badport",
			expected:    "",
			chainId:     1,
			expectedErr: "port not numerical",
		},
		{
			name:     "no port",
			netaddr:  "tcp://example.com",
			expected: "example.com:9001",
			chainId:  1,
		},

		{
			name:     "no port chain 2",
			netaddr:  "tcp://example.com",
			expected: "example.com:9002",
			chainId:  2,
		},
		{
			name:     "with port",
			netaddr:  "tcp://example.com:9000",
			expected: "example.com:9001",
			chainId:  1,
		},
		{
			name:     "with port chain 2",
			netaddr:  "tcp://example.com:9000",
			expected: "example.com:9002",
			chainId:  2,
		},
		{
			name:     "with ip",
			netaddr:  "tcp://9.9.9.9:9000",
			expected: "9.9.9.9:9001",
			chainId:  1,
		},
		{
			name:     "with ip big chain id",
			netaddr:  "tcp://9.9.9.9:9000",
			expected: "9.9.9.9:10000",
			chainId:  1000,
		},
		{
			name:     "with large port",
			netaddr:  "example.com:60000",
			expected: "example.com:60001",
			chainId:  1,
		},
		{
			name:     "with large port chain 2",
			netaddr:  "tcp://example.com:60000",
			expected: "example.com:60002",
			chainId:  2,
		},
		{
			name:        "exceeds port limit",
			netaddr:     "tcp://example.com:70000",
			expected:    "",
			chainId:     2,
			expectedErr: "max port exceeded",
		},

		{
			name:     "IPV6",
			netaddr:  "[2001:db8::1]:7000",
			expected: "[2001:db8::1]:7001",
			chainId:  1,
		},
		{
			name:     "IPV6 no port",
			netaddr:  "tcp://[2001:db8::1]",
			expected: "[2001:db8::1]:9002",
			chainId:  2,
		},
		{
			name:     "with port chain 20",
			netaddr:  "tcp://example.com:9000",
			expected: "example.com:9020",
			chainId:  20,
		},
		{
			name:        "with port chain 20 on port lower than 1024",
			netaddr:     "tcp://example.com:80",
			expected:    "",
			chainId:     20,
			expectedErr: fmt.Sprintf("port must be greater than %d", MinAllowedPort),
		},
		{
			name:     "with port chain 5",
			netaddr:  "tcp://example.com:1025",
			expected: "example.com:1030",
			chainId:  5,
		},
		{
			name:     "default with chain 56,535 ",
			netaddr:  "tcp://example.com",
			expected: "example.com:65535",
			chainId:  56535,
		},
		{
			name:        "default with chain 56,536 ",
			netaddr:     "tcp://example.com",
			expected:    "",
			chainId:     56536,
			expectedErr: "max port exceeded",
		},
		{
			name:     "port 1025 with chain 64,510 ",
			netaddr:  "tcp://example.com:1025",
			expected: "example.com:65535",
			chainId:  64510,
		},
		{
			name:        "port 1025 with chain 64,511 ",
			netaddr:     "tcp://example.com:1025",
			expected:    "",
			chainId:     64511,
			expectedErr: "max port exceeded",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ResolveAndReplacePort(&test.netaddr, test.chainId)
			require.Equal(t, err == nil, test.expectedErr == "")
			if err != nil {
				require.Contains(t, err.Error(), test.expectedErr)
				return
			}
			require.Equal(t, test.expected, test.netaddr)
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

func TestValidatorTCPProxy_ForwardsTraffic(t *testing.T) {
	logger := NewDefaultLogger()

	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	backendAddr := backendLn.Addr().String()
	backendDone := make(chan struct{})

	go func() {
		defer close(backendDone)
		conn, acceptErr := backendLn.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

		buf := make([]byte, 4)
		if _, readErr := io.ReadFull(conn, buf); readErr != nil {
			return
		}
		_, _ = conn.Write(buf)
	}()

	frontendLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	frontendPort := frontendLn.Addr().(*net.TCPAddr).Port
	require.NoError(t, frontendLn.Close())

	proxy := NewValidatorTCPProxy(map[uint64]string{uint64(frontendPort): backendAddr}, logger)
	require.NoError(t, proxy.Start())
	t.Cleanup(proxy.Stop)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", frontendPort), time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))

	msg := []byte("ping")
	_, err = conn.Write(msg)
	require.NoError(t, err)

	resp := make([]byte, len(msg))
	_, err = io.ReadFull(conn, resp)
	require.NoError(t, err)
	require.Equal(t, msg, resp)

	require.NoError(t, backendLn.Close())
	<-backendDone
}
