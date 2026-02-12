package lib

import (
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/known/anypb"
	"testing"
	"time"
)

// NOTE: pages are covered in the FSM module

func TestMarshalUnmarshal(t *testing.T) {
	// create a proto structure to test with
	expected := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// convert to bytes
	gotBytes, err := Marshal(expected)
	require.NoError(t, err)
	// create a structure to unmarshal to
	got := new(Signature)
	// populate structure with bytes
	require.NoError(t, Unmarshal(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestJSONMarshalUnmarshal(t *testing.T) {
	// create a proto structure to test with
	expected := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// convert to bytes
	gotBytes, err := MarshalJSON(expected)
	require.NoError(t, err)
	// create a structure to unmarshal to
	got := new(Signature)
	// populate structure with bytes
	require.NoError(t, UnmarshalJSON(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestJSONMarshalUnmarshalIndent(t *testing.T) {
	// create a proto structure to test with
	expected := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// convert to bytes
	gotBytes, err := MarshalJSONIndent(expected)
	require.NoError(t, err)
	// create a structure to unmarshal to
	got := new(Signature)
	// populate structure with bytes
	require.NoError(t, UnmarshalJSON(gotBytes, got))
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestAnyPb(t *testing.T) {
	// create a proto structure to test with
	expected := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// convert to anypb
	a, err := NewAny(expected)
	require.NoError(t, err)
	// convert back to structure from anypb
	got, err := FromAny(a)
	require.NoError(t, err)
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
}

func TestBytesToStringConversion(t *testing.T) {
	expected := newTestPublicKeyBytes(t)
	// convert hex string to bytes
	converted := BytesToString(expected)
	// convert string to bytes
	got, err := StringToBytes(converted)
	require.NoError(t, err)
	// compare got vs expected
	require.Equal(t, expected, got)
}

func TestBytesToTruncatedString(t *testing.T) {
	expected := newTestPublicKeyBytes(t)[:10]
	// convert hex string to bytes
	converted := BytesToTruncatedString(newTestPublicKeyBytes(t))
	// convert string to bytes
	got, err := StringToBytes(converted)
	require.NoError(t, err)
	// compare got vs expected
	require.Equal(t, expected, got)
}

func TestMerkleTree(t *testing.T) {
	tests := []struct {
		name         string
		detail       string
		items        [][]byte
		expectedRoot []byte
		expectedTree [][]byte
	}{
		{
			name:         "empty input",
			detail:       "there are no items",
			items:        nil,
			expectedRoot: []byte{},
			expectedTree: [][]byte{},
		},
		{
			name:         "one item",
			detail:       "there's one entry in the tree",
			items:        [][]byte{[]byte("a")},
			expectedRoot: crypto.Hash([]byte("a")),
			expectedTree: [][]byte{
				crypto.Hash([]byte("a")),
			},
		},
		{
			name:   "multi-item",
			detail: "there are multiple items added",
			items: [][]byte{
				[]byte("a"),
				[]byte("b"),
				[]byte("c"),
				[]byte("d"),
			},
			expectedTree: [][]byte{
				crypto.Hash([]byte("a")),
				crypto.Hash([]byte("b")),
				crypto.Hash([]byte("c")),
				crypto.Hash([]byte("d")),
				crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
				crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("d"))...)),
				crypto.Hash(append(
					crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
					crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("d"))...))...,
				)),
			},
			expectedRoot: crypto.Hash(
				append(
					crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
					crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("d"))...))...,
				),
			),
		},
		{
			name:   "'non power of two' multi-item",
			detail: "there are multiple items, but not enough for a perfect po2 tree",
			items: [][]byte{
				[]byte("a"),
				[]byte("b"),
				[]byte("c"),
			},
			expectedTree: [][]byte{
				crypto.Hash([]byte("a")),
				crypto.Hash([]byte("b")),
				crypto.Hash([]byte("c")),
				nil,
				crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
				crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("c"))...)),
				crypto.Hash(append(
					crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
					crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("c"))...))...,
				)),
			},
			expectedRoot: crypto.Hash(
				append(
					crypto.Hash(append(crypto.Hash([]byte("a")), crypto.Hash([]byte("b"))...)),
					crypto.Hash(append(crypto.Hash([]byte("c")), crypto.Hash([]byte("c"))...))...,
				),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, tree, err := MerkleTree(test.items)
			require.NoError(t, err)
			require.Equal(t, test.expectedRoot, root)
			require.Equal(t, test.expectedTree, tree)
		})
	}
}

func TestUint64PercentageDiv(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		dividend uint64
		divisor  uint64
		expected uint64
	}{
		{
			name:     "100 / 0",
			detail:   "divide by 0",
			dividend: 100,
			divisor:  0,
			expected: 0,
		},
		{
			name:     "0 / 100",
			detail:   "0 divided by 100",
			dividend: 0,
			divisor:  100,
			expected: 0,
		},
		{
			name:     "100/100",
			detail:   "100 divided by 100",
			dividend: 100,
			divisor:  100,
			expected: 100,
		},
		{
			name:     "101%",
			detail:   "can't go above 100%",
			dividend: 500,
			divisor:  100,
			expected: 100,
		},
		{
			name:     "66%",
			detail:   "2 divided by 3",
			dividend: 2,
			divisor:  3,
			expected: 66,
		},
		{
			name:     "33%",
			detail:   "1 divided by 3",
			dividend: 1,
			divisor:  3,
			expected: 33,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, Uint64PercentageDiv(test.dividend, test.divisor))
		})
	}
}

func TestUint64Percentage(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		amount   uint64
		percent  uint64
		expected uint64
	}{
		{
			name:     "0%",
			detail:   "0% percent of 100",
			amount:   100,
			percent:  0,
			expected: 0,
		},
		{
			name:     "100%",
			detail:   "100% percent of 100",
			amount:   100,
			percent:  100,
			expected: 100,
		},
		{
			name:     "101%",
			detail:   "can't go above 100%",
			amount:   100,
			percent:  100,
			expected: 100,
		},
		{
			name:     "33%",
			detail:   "33% of 100",
			amount:   100,
			percent:  33,
			expected: 33,
		},
		{
			name:     "66%",
			detail:   "66% of 100",
			amount:   100,
			percent:  66,
			expected: 66,
		},
		{
			name:     "99%",
			detail:   "99% of 1",
			amount:   1,
			percent:  99,
			expected: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, Uint64Percentage(test.amount, test.percent))
		})
	}
}

func TestUint64ReducePercentage(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		amount   uint64
		percent  uint64
		expected uint64
	}{
		{
			name:     "0%",
			detail:   "0% percent of 100",
			amount:   100,
			percent:  0,
			expected: 100,
		},
		{
			name:     "100%",
			detail:   "100% percent of 100",
			amount:   100,
			percent:  100,
			expected: 0,
		},
		{
			name:     "101%",
			detail:   "can't go above 100%",
			amount:   100,
			percent:  100,
			expected: 0,
		},
		{
			name:     "33%",
			detail:   "33% of 100",
			amount:   100,
			percent:  33,
			expected: 67,
		},
		{
			name:     "66%",
			detail:   "66% of 100",
			amount:   100,
			percent:  66,
			expected: 34,
		},
		{
			name:     "99%",
			detail:   "99% of 1",
			amount:   1,
			percent:  99,
			expected: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, Uint64ReducePercentage(test.amount, test.percent))
		})
	}
}

func TestValidURLInput(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"localhost", true},                     // valid hostname
		{"example.com", true},                   // valid hostname
		{"192.168.1.1", true},                   // valid IPv4
		{"[2001:db8::1]", true},                 // valid IPv6
		{"127.0.0.1", true},                     // valid IPv4
		{"[::1]", true},                         // valid IPv6
		{"tcp://localhost", true},               // valid hostname with tcp://
		{"tcp://example.com", true},             // valid hostname with tcp://
		{"tcp://192.168.1.1", true},             // valid IPv4 with tcp://
		{"tcp://[2001:db8::1]", true},           // valid IPv6 with tcp://
		{"subdomain.example.com", true},         // valid subdomain
		{"", false},                             // empty input
		{"tcp://", false},                       // missing hostname/IP
		{"192.168.1.1:443", true},               // valid (port allowed)
		{"tcp://localhost:8080", true},          // valid (port allowed)
		{"localhost:8080", true},                // valid (port allowed)
		{"tcp://192.168.1.1:80", true},          // valid (port allowed)
		{"example.com:1234", true},              // valid (port allowed)
		{"localhost/extra", false},              // invalid (sub-path not allowed)
		{"example.com/path/to/resource", false}, // invalid (sub-path not allowed)
		{"192.168.1.1/resource", false},         // invalid (sub-path not allowed)
		{"[::1]/path", false},                   // invalid (sub-path not allowed)
		{"example.com#", false},                 // invalid (fragment not allowed)
		{"example.com?", false},                 // invalid (query string not allowed)
		{"example.com/path#fragment", false},    // invalid (path and fragment not allowed)
		{"sub_domain.example.com", false},       // invalid (underscore not allowed in hostname)
		{"example..com", false},                 // invalid (double dot in hostname)
		{"example.com/", false},                 // invalid (trailing slash not allowed)
		{"192.168.1.1/", false},                 // invalid (trailing slash not allowed)
		{"[2001:db8::1]/", false},               // invalid (trailing slash not allowed)
		{"exa mple.com", false},                 // invalid (space in hostname)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, ValidNetURLInput(test.name))
		})
	}
}

func TestLengthedPrefix(t *testing.T) {
	tests := []struct {
		name     string
		segments [][]byte
	}{
		{
			name:     "empty key",
			segments: nil,
		},
		{
			name: "one segment",
			segments: [][]byte{
				[]byte("a"),
			},
		},
		{
			name: "two fixed length segments",
			segments: [][]byte{
				[]byte("ab"),
				[]byte("cd"),
			},
		},
		{
			name: "three variable length segments",
			segments: [][]byte{
				[]byte("abcdef"),
				[]byte("ghijk"),
				[]byte("l"),
			},
		},
		{
			name: "zero length injected",
			segments: [][]byte{
				[]byte("abcdef"),
				[]byte(""),
				[]byte("l"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DecodeLengthPrefixed(JoinLenPrefix(test.segments...))
			require.Equal(t, test.segments, got)
		})
	}
}

func TestResetTimer(t *testing.T) {
	timer := NewTimer()
	duration := 100 * time.Millisecond

	ResetTimer(timer, duration)

	// wait for the timer to fire
	select {
	case <-timer.C:
	case <-time.After(150 * time.Millisecond):
		t.Error("timer did not fire after ResetTimer")
	}
}

func TestStopTimer(t *testing.T) {
	timer := time.NewTimer(1 * time.Second)
	StopTimer(timer)

	// ensure the timer channel is drained
	select {
	case <-timer.C:
		// timer channel was drained successfully
	default:
	}
}

func TestUnmarshalRejectsUnknownBlockFields(t *testing.T) {
	block := &Block{BlockHeader: &BlockHeader{NetworkId: 1}}
	bz, err := Marshal(block)
	require.NoError(t, err)

	withUnknown := appendUnknownField(bz)
	err = Unmarshal(withUnknown, &Block{})
	require.Error(t, err)

	require.NoError(t, Unmarshal(bz, &Block{}))
}

func TestUnmarshalRejectsUnknownTransactionFields(t *testing.T) {
	tx := &Transaction{
		MessageType:   "noop",
		Msg:           &anypb.Any{},
		Signature:     &Signature{},
		Time:          1,
		CreatedHeight: 1,
		NetworkId:     1,
		ChainId:       1,
	}
	bz, err := Marshal(tx)
	require.NoError(t, err)

	withUnknown := appendUnknownField(bz)
	err = Unmarshal(withUnknown, &Transaction{})
	require.Error(t, err)

	require.NoError(t, Unmarshal(bz, &Transaction{}))
}

func appendUnknownField(b []byte) []byte {
	unknown := protowire.AppendTag(nil, 9999, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 1)
	return append(b, unknown...)
}

func TestUnmarshalRejectsOversizedLists(t *testing.T) {
	block := &Block{BlockHeader: &BlockHeader{NetworkId: 1}}
	for i := 0; i < protoMaxListLen+1; i++ {
		block.Transactions = append(block.Transactions, []byte{1})
	}
	bz, err := Marshal(block)
	require.NoError(t, err)
	require.Error(t, Unmarshal(bz, &Block{}))
}

func TestUnmarshalPreflightRejectsHugeFieldLengths(t *testing.T) {
	block := &Block{BlockHeader: &BlockHeader{NetworkId: 1}}
	bz, err := Marshal(block)
	require.NoError(t, err)

	extra := protowire.AppendTag(nil, 2, protowire.BytesType)
	extra = protowire.AppendVarint(extra, protoMaxFieldBytes+1)
	bz = append(bz, extra...)

	require.Error(t, Unmarshal(bz, &Block{}))
}
