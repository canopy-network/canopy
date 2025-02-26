package store

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		keyBitSize  int
		preset      *NodeList
		expected    *NodeList
		rootKey     []byte
		targetKey   []byte
		targetValue []byte
	}{
		{
			name: "update and target at 010",
			detail: `BEFORE:   root
							  /    \
						     0      1
		                   /  \    /  \
		                000  010 101  111
		
					AFTER:      root
							  /      \
						     0        1
		                   /  \      /  \
		                000 *010*  101   111
							`,
			keyBitSize:  3,
			rootKey:     []byte{0b10010000}, // arbitrary
			targetKey:   []byte{1},          // hashes to [010]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 0}, // 0
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0
						Key: &key{leastSigBits: []int{0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 2},  // 000
							RightChildKey: []byte{0b10, 1}, // 010
						},
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b111, 0}, // 111
							RightChildKey: []byte{0b101, 0}, // 101
						},
					},
					{ // 000
						Key:  &key{leastSigBits: []int{0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 010
						Key:  &key{leastSigBits: []int{0, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key:  &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
					{ // 101
						Key:  &key{leastSigBits: []int{1, 0, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 0
						Key: &key{leastSigBits: []int{0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 2},  // 000
							RightChildKey: []byte{0b10, 1}, // 010
						},
					},
					{ // 000
						Key:  &key{leastSigBits: []int{0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b111, 0}, // 111
							RightChildKey: []byte{0b101, 0}, // 101
						},
					},
					{ // 010 (updated)
						Key:  &key{leastSigBits: []int{0, 1, 0}},
						Node: lib.Node{Value: []byte("some_value")}, // leaf
					},
					{ // 100 root
						Key: &key{leastSigBits: []int{1, 0, 0}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// NOTE: the tree values on the right side are nulled, so the inputs for the right side are incomplete
								// grandchildren
								input000, input010 := []byte{0b0, 2}, append([]byte{0b10, 1}, crypto.Hash([]byte("some_value"))...)
								// children
								input0 := append([]byte{0b0, 0}, crypto.Hash(append(input000, input010...))...)
								input1 := append([]byte{0b1, 0})
								// root value
								return crypto.Hash(append(input0, input1...))
							}(),
							LeftChildKey:  []byte{0b0, 0}, // 0
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 101
						Key:  &key{leastSigBits: []int{1, 0, 1}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key:  &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
		{
			name: "insert and target at 000010000",
			detail: `BEFORE:   root
							  /    \
						    00     111111111
                           /  \
                   000000000  001111111

					AFTER:     root
							  /    \
						    00     111111111
                           /  \
					   *0000*   001111111
                       /   \
                  000000000 *000010000*
							`,
			keyBitSize:  9,
			rootKey:     []byte{0b10010000, 0}, // arbitrary
			targetKey:   []byte{3},             // hashes to [00001000,0]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{mostSigBytes: []byte{0b10010000}, leastSigBits: []int{0}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 1},             // 00
							RightChildKey: []byte{0b11111111, 0b1, 0}, // 111111111
						},
					},
					{ // 00
						Key: &key{leastSigBits: []int{0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b00000000, 0b0, 0}, // 000000000
							RightChildKey: []byte{0b00111111, 0b1, 0}, // 001111111
						},
					},
					{ // 000000000
						Key:  &key{mostSigBytes: []byte{0b00000000}, leastSigBits: []int{0}},
						Node: lib.Node{}, // leaf
					},
					{ // 001111111
						Key:  &key{mostSigBytes: []byte{0b00111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
					{ // 111111111
						Key:  &key{mostSigBytes: []byte{0b11111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 000000000
						Key:  &key{mostSigBytes: []byte{0b00000000}, leastSigBits: []int{0}},
						Node: lib.Node{}, // leaf
					},
					{ // 00
						Key: &key{leastSigBits: []int{0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3},             // 0000
							RightChildKey: []byte{0b00111111, 0b1, 0}, // 001111111
						},
					},
					{ // 0000
						Key: &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b00000000, 0b0, 0}, // 000000000
							RightChildKey: []byte{0b00001000, 0b0, 0}, // 000010000
						},
					},
					{ // 000010000
						Key:  &key{mostSigBytes: []byte{0b00001000}, leastSigBits: []int{0}},
						Node: lib.Node{Value: []byte("some_value")}, // leaf
					},
					{ // 001111111
						Key:  &key{mostSigBytes: []byte{0b00111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
					{ // root
						Key: &key{mostSigBytes: []byte{0b10010000}, leastSigBits: []int{0}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// great-grandchildren
								in000000000, in000010000 := []byte{0b00000000, 0, 0}, append([]byte{0b00001000, 0, 0}, crypto.Hash([]byte("some_value"))...)
								// grandchildren
								in0000, in001111111 := append([]byte{0b0, 3}, crypto.Hash(append(in000000000, in000010000...))...), []byte{0b00111111, 1, 0}
								// children
								in00, in111111111 := append([]byte{0b0, 1}, crypto.Hash(append(in0000, in001111111...))...), []byte{0b11111111, 1, 0}
								// root value
								return crypto.Hash(append(in00, in111111111...))
							}(),
							LeftChildKey:  []byte{0b0, 1},             // 00
							RightChildKey: []byte{0b11111111, 0b1, 0}, // 111111111
						},
					},
					{ // 111111111
						Key:  &key{mostSigBytes: []byte{0b11111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
		{
			name: "insert and target at 000010000",
			detail: `BEFORE:   root
							  /    \
						    0000    1
								  /   \
								1000   111
									  /   \
								    1110  1111

					AFTER:     root
							  /    \
						    0000    1
								  /   \
								1000 *11*
		                             /  \
							      *1101* 111
                                        /   \
								      1110  1111
							`,
			keyBitSize:  4,
			rootKey:     []byte{0b10010000},
			targetKey:   []byte{2}, // hashes to [1 1 0 1]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b11, 0},   // 11
						},
					},
					{ // 11 (new parent)
						Key: &key{leastSigBits: []int{1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1101, 0}, // 1101
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1001 (root)
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// great-grandchildren
								input1101, input111 := append([]byte{0b1101, 0}, crypto.Hash([]byte("some_value"))...), append([]byte{0b111, 0})
								// grandchildren
								input1000, input11 := []byte{0b1000, 0}, append([]byte{0b11, 0}, crypto.Hash(append(input1101, input111...))...)
								// children
								input0000, input1 := []byte{0b0000, 3}, append([]byte{0b1, 0}, crypto.Hash(append(input1000, input11...))...)
								// root value
								return crypto.Hash(append(input0000, input1...))
							}(),
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 1101 (inserted)
						Key:  &key{leastSigBits: []int{1, 1, 0, 1}},
						Node: lib.Node{},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
		{
			name: "insert and target at 0 1 1 0",
			detail: `BEFORE:   root
							  /    \
						    0000    1
								  /   \
								1000   111
									  /   \
								    1110  1111

					AFTER:     root
							  /     \
                          *01*       1
                           / \      /  \
                       0000 *0110* 1000  111
									   /  \
								     1110   1111
							`,
			keyBitSize:  4,
			rootKey:     []byte{0b10010000},
			targetKey:   []byte{6}, // hashes to [0 1 1 0]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 0 (new parent)
						Key: &key{leastSigBits: []int{0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3},   // 0000
							RightChildKey: []byte{0b110, 1}, // 0110
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 0110 (inserted)
						Key:  &key{leastSigBits: []int{0, 1, 1, 0}},
						Node: lib.Node{},
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1001 (root)
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// grandchildren
								input0000, input0110 := []byte{0b0, 3}, append([]byte{0b110, 1}, crypto.Hash([]byte("some_value"))...)
								// children
								input0, input1 := append([]byte{0b0, 0}, crypto.Hash(append(input0000, input0110...))...), []byte{0b1, 0}
								// root value
								return crypto.Hash(append(input0, input1...))
							}(),
							LeftChildKey:  []byte{0b0, 0}, // 0
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				defer memStore.Close()
				// execute the traversal code
				require.NoError(t, smt.Set(test.targetKey, test.targetValue))
				// create an iterator to check out the values of the store
				it, err := memStore.Iterator(nil)
				require.NoError(t, err)
				defer it.Close()
				// iterate through the database
				for i := 0; it.Valid(); func() { it.Next(); i++ }() {
					got := newNode()
					// convert the value to a node
					require.NoError(t, lib.Unmarshal(it.Value(), &got.Node))
					// convert the key to a node key
					got.Key.fromBytes(it.Key())
					// compare got vs expected
					//fmt.Printf("%08b %v\n", got.Key.mostSigBytes, got.Key.leastSigBits)
					require.Equal(t, test.expected.Nodes[i].Key.bytes(), got.Key.bytes(), fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					require.Equal(t, test.expected.Nodes[i].LeftChildKey, got.LeftChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					require.Equal(t, test.expected.Nodes[i].RightChildKey, got.RightChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					// check root value (this allows quick verification of the hashing up logic without actually needing to fill in and check every value)
					if bytes.Equal(got.Key.bytes(), smt.root.Key.bytes()) {
						require.Equal(t, test.expected.Nodes[i].Value, got.Value)
					}
				}
			}()
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		keyBitSize int
		preset     *NodeList
		expected   *NodeList
		rootKey    []byte
		targetKey  []byte
	}{
		{
			name: "delete with target at 010",
			detail: `BEFORE:   root
							  /    \
						     0      1
		                   /  \    /  \
		                000 *010* 101  111
		
					AFTER:      root
							  /      \
                            000        1
		                              /  \
		                           101   111
							`,
			keyBitSize: 3,
			rootKey:    []byte{0b10010000}, // arbitrary
			targetKey:  []byte{1},          // hashes to [010]
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 0}, // 0
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0
						Key: &key{leastSigBits: []int{0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 2},  // 000
							RightChildKey: []byte{0b10, 1}, // 010
						},
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b111, 0}, // 111
							RightChildKey: []byte{0b101, 0}, // 101
						},
					},
					{ // 000
						Key:  &key{leastSigBits: []int{0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 010
						Key:  &key{leastSigBits: []int{0, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key:  &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
					{ // 101
						Key:  &key{leastSigBits: []int{1, 0, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 000
						Key:  &key{leastSigBits: []int{0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b111, 0}, // 111
							RightChildKey: []byte{0b101, 0}, // 101
						},
					},
					{ // 100 root
						Key: &key{leastSigBits: []int{1, 0, 0}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// NOTE: the tree values on the right side are nulled, so the inputs for the right side are incomplete
								// children
								input000 := append([]byte{0b0, 2})
								input1 := append([]byte{0b1, 0})
								// root value
								return crypto.Hash(append(input000, input1...))
							}(),
							LeftChildKey:  []byte{0b0, 2}, // 000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 101
						Key:  &key{leastSigBits: []int{1, 0, 1}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key:  &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
		{
			name: "Delete and target at 1 0 1 1",
			detail: `BEFORE:   root
							  /    \
						    0000    1
								  /   \
							  *1011*   111
									  /   \
								    1110  1111

					AFTER:     root
							  /     \
                          0000      111
                                    /  \
                                 1110   1111
							`,
			keyBitSize: 4,
			rootKey:    []byte{0b10010000},
			targetKey:  []byte{8}, // hashes to [1 0 1 1]
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1011, 0}, // 1011
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1011
						Key:  &key{leastSigBits: []int{1, 0, 1, 1}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1001 (root)
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							Value: func() []byte {
								// NOTE: 111 hash not updated, so use key only as there's no value preset
								// children
								in0000, in111 := []byte{0, 3}, []byte{0b111, 0}
								// root value
								return crypto.Hash(append(in0000, in111...))
							}(),
							LeftChildKey:  []byte{0b0, 3},   // 0000
							RightChildKey: []byte{0b111, 0}, // 111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
		{
			name: "delete (not exists) and target at 1101",
			detail: `BEFORE:   root     *1101* <- target not exists
							  /    \
						    0000    1
                         		  /   \
								1000   111
									  /   \
								    1110  1111

					After:   root
							  /    \
						    0000    1
								  /   \
								1000   111
									  /   \
								    1110  1111
							`,
			keyBitSize: 4,
			rootKey:    []byte{0b10010000},
			targetKey:  []byte{2}, // hashes to [1 1 0 1]
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1001 (root)
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				defer memStore.Close()
				// execute the traversal code
				require.NoError(t, smt.Delete(test.targetKey))
				// create an iterator to check out the values of the store
				it, err := memStore.Iterator(nil)
				require.NoError(t, err)
				defer it.Close()
				// iterate through the database
				for i := 0; it.Valid(); func() { it.Next(); i++ }() {
					got := newNode()
					// convert the value to a node
					require.NoError(t, lib.Unmarshal(it.Value(), &got.Node))
					// convert the key to a node key
					got.Key.fromBytes(it.Key())
					// compare got vs expected
					//fmt.Printf("%08b %v\n", got.Key.mostSigBytes, got.Key.leastSigBits)
					require.Equal(t, test.expected.Nodes[i].Key.bytes(), got.Key.bytes(), fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					require.Equal(t, test.expected.Nodes[i].LeftChildKey, got.LeftChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					require.Equal(t, test.expected.Nodes[i].RightChildKey, got.RightChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.leastSigBits))
					// check root value (this allows quick verification of the hashing up logic without actually needing to fill in and check every value)
					if bytes.Equal(got.Key.bytes(), smt.root.Key.bytes()) {
						require.Equal(t, test.expected.Nodes[i].Value, got.Value)
					}
				}
			}()
		})
	}
}

func TestTraverse(t *testing.T) {
	tests := []struct {
		name              string
		detail            string
		keyBitSize        int
		preset            *NodeList
		target            *node
		expectedTraversal *NodeList
		expectedCurrent   *node
		rootKey           []byte
	}{
		{
			name: "basic traversal, no preset (Left - 3bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the min hash
                               root
							  /    \
							*000*   111`,
			keyBitSize: 3,
			target:     &node{Key: &key{leastSigBits: []int{0, 0, 0}}},
			expectedTraversal: &NodeList{Nodes: []*node{
				{
					Key: &key{
						leastSigBits: []int{1, 1, 1},
					},
					Node: lib.Node{
						Value: func() []byte {
							// left child key + value
							leftInput := append([]byte{0, 2}, bytes.Repeat([]byte{0}, 20)...)
							// right child key + value
							rightInput := append([]byte{7, 0}, bytes.Repeat([]byte{255}, 20)...)
							// hash ( left + right )
							return crypto.Hash(append(leftInput, rightInput...))
						}(),
						LeftChildKey:  []byte{0, 2},
						RightChildKey: []byte{7, 0},
					},
				},
			}},
			expectedCurrent: &node{
				Key: &key{
					leastSigBits: []int{0, 0, 0},
				},
				Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
			},
		},
		{
			name: "basic traversal, no preset (Right - 3bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the max hash
                               root
							  /    \
							000   *111*`,
			keyBitSize: 3,
			target:     &node{Key: &key{leastSigBits: []int{1, 1, 1}}},
			expectedTraversal: &NodeList{Nodes: []*node{
				{
					Key: &key{
						leastSigBits: []int{1, 1, 1},
					},
					Node: lib.Node{
						Value: func() []byte {
							// left child key + value
							leftInput := append([]byte{0, 2}, bytes.Repeat([]byte{0}, 20)...)
							// right child key + value
							rightInput := append([]byte{7, 0}, bytes.Repeat([]byte{255}, 20)...)
							// hash ( left + right )
							return crypto.Hash(append(leftInput, rightInput...))
						}(),
						LeftChildKey:  []byte{0, 2},
						RightChildKey: []byte{7, 0},
					},
				},
			}},
			expectedCurrent: &node{
				Key: &key{
					leastSigBits: []int{1, 1, 1},
				},
				Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
			},
		},
		{
			name: "basic traversal, no preset (Left - 4bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the min hash
                               root
							  /    \
						   *0000*   1111`,
			keyBitSize: 4,
			target:     &node{Key: &key{leastSigBits: []int{0, 0, 0, 0}}},
			expectedTraversal: &NodeList{Nodes: []*node{
				{
					Key: &key{
						leastSigBits: []int{1, 1, 1, 1},
					},
					Node: lib.Node{
						Value: func() []byte {
							// left child key + value
							leftInput := append([]byte{0, 3}, bytes.Repeat([]byte{0}, 20)...)
							// right child key + value
							rightInput := append([]byte{15, 0}, bytes.Repeat([]byte{255}, 20)...)
							// hash ( left + right )
							return crypto.Hash(append(leftInput, rightInput...))
						}(),
						LeftChildKey:  []byte{0, 3},
						RightChildKey: []byte{15, 0},
					},
				},
			}},
			expectedCurrent: &node{
				Key: &key{
					leastSigBits: []int{0, 0, 0, 0},
				},
				Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
			},
		},
		{
			name: "basic traversal, no preset (Right - 5bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the max hash
                               root
							  /    \
							00000  *11111*`,
			keyBitSize: 5,
			target:     &node{Key: &key{leastSigBits: []int{1, 1, 1, 1, 1}}},
			expectedTraversal: &NodeList{Nodes: []*node{
				{
					Key: &key{
						leastSigBits: []int{1, 1, 1, 1, 1},
					},
					Node: lib.Node{
						Value: func() []byte {
							// left child key + value
							leftInput := append([]byte{0, 4}, bytes.Repeat([]byte{0}, 20)...)
							// right child key + value
							rightInput := append([]byte{31, 0}, bytes.Repeat([]byte{255}, 20)...)
							// hash ( left + right )
							return crypto.Hash(append(leftInput, rightInput...))
						}(),
						LeftChildKey:  []byte{0, 4},
						RightChildKey: []byte{31, 0},
					},
				},
			}},
			expectedCurrent: &node{
				Key: &key{
					leastSigBits: []int{1, 1, 1, 1, 1},
				},
				Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
			},
		},
		{
			name: "traversal with preset and target at 1110",
			detail: `Preset:   root
							  /    \
						    0000    1
								  /   \
								1000   111
									  /   \
								   *1110* 1111
							`,
			keyBitSize: 4,
			target:     &node{Key: &key{leastSigBits: []int{1, 1, 1, 0}}},
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{Value: []byte("some_value")}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
				},
			},
			expectedCurrent: &node{ // 1110
				Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
				Node: lib.Node{Value: []byte("some_value")}, // leaf
			},
			rootKey: []byte{0b10010000},
		},
		{
			name: "traversal with preset and target at 1100",
			detail: `Preset:   root
							  /    \
						    0000    1
								  /   \
								1000  *111*
									  /   \
								    1110 1111
							`,
			keyBitSize: 4,
			target:     &node{Key: &key{leastSigBits: []int{1, 1, 0, 0}}},
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
				},
			},
			expectedCurrent: &node{ // 111
				Key: &key{leastSigBits: []int{1, 1, 1}},
				Node: lib.Node{
					LeftChildKey:  []byte{0b1110, 0}, // 1110
					RightChildKey: []byte{0b1111, 0}, // 1111
				},
			},
			rootKey: []byte{0b10010000},
		},
		{
			name: "traversal with preset and target at 0001",
			detail: `Preset:   root
							  /    \
						    *0000*  1
								  /   \
								1000   111
									  /   \
								    1110 1111
							`,
			keyBitSize: 4,
			target:     &node{Key: &key{leastSigBits: []int{0, 0, 0, 1}}},
			preset: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
					{ // 0000
						Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{Value: []byte("some_value")}, // leaf
					},
					{ // 1
						Key: &key{leastSigBits: []int{1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1000, 0}, // 1000
							RightChildKey: []byte{0b111, 0},  // 111
						},
					},
					{ // 1000
						Key:  &key{leastSigBits: []int{1, 0, 0, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 111
						Key: &key{leastSigBits: []int{1, 1, 1}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b1110, 0}, // 1110
							RightChildKey: []byte{0b1111, 0}, // 1111
						},
					},
					{ // 1110
						Key:  &key{leastSigBits: []int{1, 1, 1, 0}},
						Node: lib.Node{}, // leaf
					},
					{ // 1111
						Key:  &key{leastSigBits: []int{1, 1, 1, 1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					{ // root
						Key: &key{leastSigBits: []int{1, 0, 0, 1}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3}, // 0000
							RightChildKey: []byte{0b1, 0}, // 1
						},
					},
				},
			},
			expectedCurrent: &node{ // 0000
				Key:  &key{leastSigBits: []int{0, 0, 0, 0}},
				Node: lib.Node{Value: []byte("some_value")}, // leaf
			},
			rootKey: []byte{0b10010000},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				defer memStore.Close()
				// set target
				smt.target = test.target
				// execute the traversal code
				require.NoError(t, smt.traverse())
				// compare got vs expected
				require.EqualExportedValues(t, test.expectedCurrent, smt.current)
				require.EqualExportedValues(t, test.expectedTraversal, smt.traversed)
			}()
		})
	}
}

func NewTestSMT(t *testing.T, preset *NodeList, root []byte, keyBitSize int) (*SMT, *TxnWrapper) {
	// create a new memory store to work with
	db, err := badger.OpenManaged(badger.DefaultOptions("").
		WithInMemory(true).WithLoggingLevel(badger.ERROR))
	require.NoError(t, err)
	// make a writable tx that reads from the last height
	tx := db.NewTransactionAt(1, true)
	memStore := NewTxnWrapper(tx, lib.NewDefaultLogger(), stateCommitmentPrefix)
	// if there's no preset - use the default 3 nodes
	if preset == nil {
		if root != nil {
			return NewSMT(root, keyBitSize, memStore), memStore
		}
		return NewSMT(RootKey, keyBitSize, memStore), memStore
	}
	// create the smt
	smt := &SMT{
		store:        memStore,
		keyBitLength: keyBitSize,
	}
	// update root
	smt.root = preset.Nodes[0]
	// preset the nodes
	for _, n := range preset.Nodes {
		// set the node in the db
		require.NoError(t, smt.setNode(n))
	}
	return smt, memStore
}

func TestNewSMT(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		preset   *NodeList
		expected *NodeList
	}{
		{
			name:   "uninitialized tree",
			detail: "the tree is uninitialized - should populate with the default 3 nodes (most_left, root, most_right)",
			preset: nil,
			expected: &NodeList{
				Nodes: []*node{
					{
						Key: &key{
							mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{
							Value: func() []byte {
								// left child key + value
								leftInput := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7}, bytes.Repeat([]byte{0}, 20)...)
								// right child key + value
								rightInput := append([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}, bytes.Repeat([]byte{255}, 20)...)
								// hash ( left + right )
								return crypto.Hash(append(leftInput, rightInput...))
							}(),
							LeftChildKey:  []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7},
							RightChildKey: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0},
						},
					},
					{
						Key: &key{
							mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
					},
				},
			},
		},
		{
			name:   "initialized tree",
			detail: "the tree is initialized - thus it should be the same as preset",
			preset: &NodeList{
				Nodes: []*node{
					{
						Key: &key{
							mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{
							Value: func() []byte {
								// left child key + value
								leftInput := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 7}, bytes.Repeat([]byte{0}, 20)...)
								// right child key + value
								rightInput := append([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}, bytes.Repeat([]byte{255}, 20)...)
								// hash ( left + right )
								return crypto.Hash(append(leftInput, rightInput...))
							}(),
							LeftChildKey:  []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 7},
							RightChildKey: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0},
						},
					},
					{
						Key: &key{
							mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{
						Key: &key{
							mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{
							Value: func() []byte {
								// left child key + value
								leftInput := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 7}, bytes.Repeat([]byte{0}, 20)...)
								// right child key + value
								rightInput := append([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}, bytes.Repeat([]byte{255}, 20)...)
								// hash ( left + right )
								return crypto.Hash(append(leftInput, rightInput...))
							}(),
							LeftChildKey:  []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 7},
							RightChildKey: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0},
						},
					},
					{
						Key: &key{
							mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a new memory store to work with
			memStore, err := NewStoreInMemory(lib.NewDefaultLogger())
			require.NoError(t, err)
			// preset the nodes
			if test.preset != nil {
				for _, n := range test.preset.Nodes {
					// get the bytes for the node to set in the db
					nodeBytes, e := n.bytes()
					require.NoError(t, e)
					// set the node in the db
					require.NoError(t, memStore.Set(n.Key.bytes(), nodeBytes))
				}
			}
			// execute the function call
			_ = NewSMT(RootKey, MaxKeyBitLength, memStore)
			// create an iterator to check out the values of the store
			it, err := memStore.Iterator(nil)
			require.NoError(t, err)
			// iterate through the database
			for i := 0; it.Valid(); func() { it.Next(); i++ }() {
				got := newNode()
				// convert the value to a node
				require.NoError(t, lib.Unmarshal(it.Value(), &got.Node))
				// convert the key to a node key
				got.Key.fromBytes(it.Key())
				// compare got vs expected
				require.EqualExportedValues(t, test.expected.Nodes[i], got)
			}
		})
	}
}

func TestKeyGreatestCommonPrefix(t *testing.T) {
	tests := []struct {
		name    string
		target  *key
		current *key
		gcp     *key
		bitPos  int

		expectedGCP    *key
		expectedBitPos int
	}{
		{
			name: "0000 partial",
			target: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0, 0},
			},
			current: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: nil,
			},
			bitPos: 0,
			expectedGCP: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0},
			},
			expectedBitPos: 1,
		},
		{
			name: "00000001 0111 full",
			target: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1, 1, 1},
			},
			current: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1, 1, 1},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: nil,
			},
			bitPos: 0,
			expectedGCP: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1, 1, 1},
			},
			expectedBitPos: 12,
		},
		{
			name: "11111111 000 full",
			target: &key{
				mostSigBytes: []byte{255},
				leastSigBits: []int{0, 0, 0},
			},
			current: &key{
				mostSigBytes: []byte{255},
				leastSigBits: []int{0, 0, 0},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: nil,
			},
			bitPos: 0,
			expectedGCP: &key{
				mostSigBytes: []byte{255},
				leastSigBits: []int{0, 0, 0},
			},
			expectedBitPos: 11,
		},
		{
			name: "00000001 0111 partial",
			target: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1, 1, 1},
			},
			current: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: nil,
			},
			bitPos: 0,
			expectedGCP: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 1},
			},
			expectedBitPos: 10,
		},
		{
			name: "11111111 000 partial",
			target: &key{
				mostSigBytes: []byte{255},
				leastSigBits: []int{0, 0, 0},
			},
			current: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: nil,
			},
			bitPos: 0,
			expectedGCP: &key{
				mostSigBytes: nil,
				leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expectedBitPos: 8,
		},
		{
			name: "000011 continue",
			target: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0, 0, 1},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 5,
		},
		{
			name: "00000001 000011 continue",
			target: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 13,
		},
		{
			name: "00000001 000011 continue not exact",
			target: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1, 0},
			},
			gcp: &key{
				mostSigBytes: nil,
				leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				mostSigBytes: []byte{1},
				leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 13,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.target.greatestCommonPrefix(&test.bitPos, test.gcp, test.current)
			require.Equal(t, test.expectedGCP, test.gcp)
			require.Equal(t, test.expectedBitPos, test.bitPos)
		})
	}
}

func TestKeyDecode(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedKey *key
	}{
		{
			name: "0",
			data: []byte{0, 0},
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0},
			},
		},
		{
			name: "1",
			data: []byte{1, 0},
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{1},
			},
		},
		{
			name: "00",
			data: []byte{0, 1},
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0},
			},
		},
		{
			name: "000",
			data: []byte{0, 2},
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 0},
			},
		},
		{
			name: "001",
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 1},
			},
			data: []byte{1, 2},
		},
		{
			name: "001",
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 1},
			},
			data: []byte{1, 2},
		},
		{
			name: "10101",
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{1, 0, 1, 0, 1},
			},
			data: []byte{21, 0},
		},
		{
			name: "00010101",
			expectedKey: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 0, 1, 0, 1, 0, 1},
			},
			data: []byte{21, 3},
		},
		{
			name: "00000000 001",
			expectedKey: &key{
				mostSigBytes: []byte{0},
				leastSigBits: []int{0, 0, 1},
			},
			data: []byte{0, 1, 2},
		},
		{
			name: "00000000 11111111 001",
			expectedKey: &key{
				mostSigBytes: []byte{0, 255},
				leastSigBits: []int{0, 0, 1},
			},
			data: []byte{0, 255, 1, 2},
		},
		{
			name: "00000101 11111111 101",
			expectedKey: &key{
				mostSigBytes: []byte{5, 255},
				leastSigBits: []int{1, 0, 1},
			},
			data: []byte{5, 255, 5, 0},
		},
		{
			name: "00000101 11111111",
			expectedKey: &key{
				mostSigBytes: []byte{5},
				leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			data: []byte{5, 255, 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := new(key)
			got.fromBytes(test.data)
			require.Equal(t, test.expectedKey, got)
		})
	}
}

func TestKeyEncode(t *testing.T) {
	tests := []struct {
		name            string
		key             *key
		expectedEncoded []byte
	}{
		{
			name: "0",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0},
			},
			expectedEncoded: []byte{0, 0},
		},
		{
			name: "1",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{1},
			},
			expectedEncoded: []byte{1, 0},
		},
		{
			name: "00",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0},
			},
			expectedEncoded: []byte{0, 1},
		},
		{
			name: "000",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 0},
			},
			expectedEncoded: []byte{0, 2},
		},
		{
			name: "001",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 1},
			},
			expectedEncoded: []byte{1, 2},
		},
		{
			name: "10101",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{1, 0, 1, 0, 1},
			},
			expectedEncoded: []byte{21, 0},
		},
		{
			name: "00010101",
			key: &key{
				mostSigBytes: []byte{},
				leastSigBits: []int{0, 0, 0, 1, 0, 1, 0, 1},
			},
			expectedEncoded: []byte{21, 3},
		},
		{
			name: "00000000 001",
			key: &key{
				mostSigBytes: []byte{0},
				leastSigBits: []int{0, 0, 1},
			},
			expectedEncoded: []byte{0, 1, 2},
		},
		{
			name: "00000000 11111111 001",
			key: &key{
				mostSigBytes: []byte{0, 255},
				leastSigBits: []int{0, 0, 1},
			},
			expectedEncoded: []byte{0, 255, 1, 2},
		},
		{
			name: "00000101 11111111 101",
			key: &key{
				mostSigBytes: []byte{5, 255},
				leastSigBits: []int{1, 0, 1},
			},
			expectedEncoded: []byte{5, 255, 5, 0},
		},
		{
			name: "00000101 11111111",
			key: &key{
				mostSigBytes: []byte{5},
				leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1, 1},
			},
			expectedEncoded: []byte{5, 255, 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.key.bytes()
			require.Equal(t, test.expectedEncoded, got)
		})
	}
}

func TestBitsToBytes(t *testing.T) {
	tests := []struct {
		name     string
		ba       []int
		expected byte
	}{
		{
			name:     "nil bit array to 0 byte",
			ba:       nil,
			expected: byte(0b00000000),
		},
		{
			name:     "0101",
			ba:       []int{0, 1, 0, 1},
			expected: byte(0b00000101),
		},
		{
			name:     "1010",
			ba:       []int{1, 0, 1, 0},
			expected: byte(0b00001010),
		},
		{
			name:     "11011",
			ba:       []int{1, 1, 0, 1, 1},
			expected: byte(0b00011011),
		},
		{
			name:     "11111111",
			ba:       []int{1, 1, 1, 1, 1, 1, 1, 1},
			expected: byte(0b11111111),
		},
		{
			name:     "00000000",
			ba:       []int{0, 0, 0, 0, 0, 0, 0, 0},
			expected: byte(0b00000000),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := new(key)
			got := k.bitsToByte(test.ba)
			require.Equal(t, test.expected, got, fmt.Sprintf("Expected: %8b, Got: %8b\n", test.expected, got))
		})
	}
}

func TestBytesToBits(t *testing.T) {
	tests := []struct {
		name          string
		byt           byte
		leadingZeroes int
		expected      []int
	}{
		{
			name:          "zero",
			byt:           0,
			leadingZeroes: 0,
			expected:      []int{0},
		},
		{
			name:          "0001",
			byt:           byte(0b1),
			leadingZeroes: 3,
			expected:      []int{0, 0, 0, 1},
		},
		{
			name:          "01011",
			byt:           byte(0b1011),
			leadingZeroes: 1,
			expected:      []int{0, 1, 0, 1, 1},
		},
		{
			name:          "00100100",
			byt:           byte(0b00100100),
			leadingZeroes: 2,
			expected:      []int{0, 0, 1, 0, 0, 1, 0, 0},
		},
		{
			name:          "00000000",
			byt:           byte(0b00000000),
			leadingZeroes: 7,
			expected:      []int{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:          "11111111",
			byt:           byte(0b11111111),
			leadingZeroes: 0,
			expected:      []int{1, 1, 1, 1, 1, 1, 1, 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := new(key)
			got := k.byteToBits(test.byt, test.leadingZeroes)
			require.Equal(t, test.expected, got, fmt.Sprintf("Expected: %v, Got: %v\n", test.expected, got))
		})
	}
}

func TestStoreProof(t *testing.T) {
	threeLeavesPreset := &NodeList{
		Nodes: []*node{
			{ // 1000
				Key:  &key{leastSigBits: []int{1, 0, 0, 0}}, // leaf
				Node: lib.Node{},
			},
			{ // 1110
				Key: &key{leastSigBits: []int{1, 1, 1, 0}}, // leaf
				Node: lib.Node{
					Value: []byte("some_value"),
				},
			},
			{ // 1111
				Key:  &key{leastSigBits: []int{1, 1, 1, 1}}, // leaf
				Node: lib.Node{},
			},
		},
	}

	fourLeavesPreset := &NodeList{
		Nodes: []*node{
			{ // 0001
				Key: &key{leastSigBits: []int{0, 0, 0, 1}}, // leaf
				Node: lib.Node{
					Value: []byte("some_value"),
				},
			},
			{ // 0010
				Key:  &key{leastSigBits: []int{0, 0, 1, 0}}, // leaf
				Node: lib.Node{},
			},
			{ // 1000
				Key:  &key{leastSigBits: []int{1, 1, 1, 0}}, // leaf
				Node: lib.Node{},
			},
			{ // 1111
				Key: &key{leastSigBits: []int{1, 1, 1, 1}}, // leaf
				Node: lib.Node{
					Value: []byte("some_value"),
				},
			},
		},
	}

	tests := []struct {
		name               string
		detail             string
		keyBitSize         int
		target             *node
		preset             *NodeList
		rootKey            []byte
		valid              bool
		validateMembership bool
		proofErr           error
	}{
		{
			name: "valid proof of membership with target at 1110",
			detail: `Preset:      root
		                         /    \
							   0000    1
		                         /      \
		                       1000     111
		                                /   \
		                            *1110*  1111
							`,
			keyBitSize:         4,
			preset:             threeLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{1, 1, 1, 0}}, Node: lib.Node{Value: []byte("some_value")}},
			validateMembership: true,
			valid:              true,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "valid proof of non membership with target at 2101",
			detail: `Preset:      root
		                         /    \
		                       0000    1
		                         /      \
		                       1000     111
		                                /   \
		                             1110  1111
							`,
			keyBitSize:         4,
			preset:             threeLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{2, 1, 0, 1}}, Node: lib.Node{Value: []byte("some_value")}},
			validateMembership: false,
			valid:              true,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "invalid proof of non membership with target at 1111 (key exist, values differ)",
			detail: `Preset:      root
		                         /    \
		                       0000    1
		                         /      \
		                       1000     111
		                                /   \
		                             1110  *1111*
							`,
			keyBitSize:         4,
			preset:             threeLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{1, 1, 1, 1}}, Node: lib.Node{Value: []byte("some_value")}},
			validateMembership: true,
			valid:              false,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "invalid proof of non membership with target at 1111 (key exists)",
			detail: `Preset:        root
		                         /        \
		                       0           1
		                     /  \         /   \
		                   0001 0010    1110  1111
							`,
			keyBitSize:         4,
			preset:             fourLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{1, 1, 1, 1}}, Node: lib.Node{Value: []byte("some_value")}},
			validateMembership: false,
			valid:              false,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "valid proof of membership with target at 0001",
			detail: `Preset:        root
		                         /        \
		                       0           1
		                     /  \         /   \
		                   0001 0010    1110  1111
							`,
			keyBitSize:         4,
			preset:             fourLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{0, 0, 0, 1}}, Node: lib.Node{Value: []byte("some_value")}},
			validateMembership: true,
			valid:              true,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "invalid proof of non membership with target at 0001 (values differ)",
			detail: `Preset:        root
		                         /        \
		                       0           1
		                     /  \         /   \
		                   0001 0010    1110  1111
							`,
			keyBitSize:         4,
			preset:             fourLeavesPreset,
			target:             &node{Key: &key{leastSigBits: []int{0, 0, 0, 1}}, Node: lib.Node{Value: []byte("some_value1")}},
			validateMembership: true,
			valid:              false,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "invalid proof of membership with default smt root value as target (gcp is never equal to root)",
			detail: `Preset:       *root*
		                         /        \
		                       min          max
							`,
			keyBitSize:         4,
			target:             &node{Key: newNodeKey(bytes.Clone(RootKey), MaxKeyBitLength), Node: lib.Node{}},
			validateMembership: true,
			valid:              false,
			rootKey:            []byte("a_random_root_key"),
		},
		{
			name: "invalid proof of membership with default smt max value value as target (default min-max keys are not hashed)",
			detail: `Preset:        root
		                         /        \
		                       min          *max*
							`,
			keyBitSize:         MaxKeyBitLength,
			target:             &node{Key: newNodeKey(bytes.Repeat([]byte{255}, 20), MaxKeyBitLength), Node: lib.Node{}},
			validateMembership: true,
			valid:              false,
			rootKey:            nil,
		},
		{
			name: "Attempt to verify the root key on the default tree",
			detail: `Preset:        root
		                         /        \
		                       min          max
							`,
			keyBitSize:         4,
			target:             &node{Key: &key{leastSigBits: []int{1, 1, 1, 0}}, Node: lib.Node{Value: []byte("some_value1")}},
			validateMembership: true,
			valid:              false,
			proofErr:           ErrReserveKeyWrite("root"),
			rootKey:            nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// VerifyProof uses the same default root for the in-memory store, so we
			//  are setting the test root key as the global RootKey to use the same
			// root for the test
			if test.rootKey != nil {
				tempRootKey := RootKey
				RootKey = test.rootKey
				defer func() { RootKey = tempRootKey }()
			}

			// create the smt
			smt, memStore := NewTestSMT(t, nil, test.rootKey, test.keyBitSize)
			defer memStore.Close()

			// preset the nodes manually to trigger rehashing
			if test.preset != nil {
				for _, n := range test.preset.Nodes {
					// set the node in the db
					require.NoError(t, smt.Set(n.Key.bytes(), n.Value))
				}
			}

			// generate the merkle proof
			proof, err := smt.GetMerkleProof(test.target.Key.bytes())

			// validate proof results
			if test.proofErr != nil {
				require.Equal(t, test.proofErr, err)
				return
			}

			require.NoError(t, err)
			// verify the proof
			valid, err := smt.VerifyProof(test.target.Key.bytes(), test.target.Value,
				test.validateMembership, smt.Root(), proof)

			// validate results
			require.NoError(t, err)
			require.Equal(t, test.valid, valid)
		})
	}
}
