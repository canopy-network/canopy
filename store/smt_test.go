package store

import (
	"bytes"
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"strconv"
	"testing"

	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestFuzzMultiSet(t *testing.T) {
	iterations := 1000
	// create a new SMT
	smt1, memStore := NewTestSMT(t, nil, nil, 160)
	unsortedOps := make(map[uint64]valueOp)
	unsortedOps2 := make(map[uint64]valueOp)
	// close the store when done
	defer memStore.Close()
	// create a compare SMT
	smt2, memStore2 := NewTestSMT(t, nil, nil, 160)
	// close the store when done
	defer memStore2.Close()
	var keys [][]byte
	for range iterations {
		// load 32 random bytes
		random := make([]byte, 32)
		_, err := rand.Read(random)
		require.NoError(t, err)
		// 50% of the time do a set
		if mathrand.Intn(2) == 0 {
			keys = append(keys, random)
			unsortedOps[lib.MemHash(random)] = valueOp{key: random, value: random, op: opSet}
			unsortedOps2[lib.MemHash(random)] = valueOp{key: random, value: random, op: opSet}
		} else {
			toDelete := random
			if mathrand.Intn(2) == 0 {
				if len(keys) != 0 {
					// choose a random key
					idx := mathrand.Intn(len(keys))
					toDelete = keys[idx]
					// remove it from the keys slice
					keys = append(keys[:idx], keys[idx+1:]...)
				}
			}
			// 50% of the time do a delete
			unsortedOps[lib.MemHash(toDelete)] = valueOp{key: toDelete, op: opDelete}
			unsortedOps2[lib.MemHash(toDelete)] = valueOp{key: toDelete, op: opDelete}
		}
		// for smt 2 commit everytime
		require.NoError(t, smt2.Commit(unsortedOps2))
	}
	// commit smt 1
	require.NoError(t, smt1.CommitParallel(unsortedOps))
	// compare roots between the two smts
	require.Equal(t, smt1.Root(), smt2.Root())
}

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
			name: "insert and target at 1110",
			detail: `BEFORE    root
	                             /   \
							    0000  1111

				          AFTER     root
			                        /  \
							    0000   111
	                                 /  \
								   *1110*  1111
	                            `,
			keyBitSize:  4,
			rootKey:     []byte{0b10010000}, // arbitrary
			targetKey:   []byte{5},          // hashes to [1110]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1100", nil, "0000", "1111"), // root
					newTestNode("0000", nil, "", ""),         // leaf
					newTestNode("1111", nil, "", ""),         // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode( // root
						"1100",
						func() []byte {
							// grandchildren
							input1110 := append(keyBytesFromStr("1110"), crypto.Hash([]byte("some_value"))...)
							input1111 := append(keyBytesFromStr("1111"), []byte{}...)
							//  children
							input0000 := append(keyBytesFromStr("0000"), []byte{}...)
							input111 := append(keyBytesFromStr("111"), crypto.Hash(append(input1110, input1111...))...)
							// root
							return crypto.Hash(append(input0000, input111...))
						}(),
						"0000", "111"),
					newTestNode("1110", []byte("some_value"), "", ""), // new leaf
					newTestNode("1111", nil, "", ""),                  // leaf
				},
			},
		},
		{
			name: "insert and target at 011",
			detail: `BEFORE    root
			                           / \
			                          0   1
								     / \
							       010 001

				          AFTER    root
			                        / \
			                       0   1
								  / \
							     01  001
							    / \
							  010 *011*
			                          `,
			keyBitSize:  3,
			rootKey:     []byte{0b10010000}, // arbitrary
			targetKey:   []byte{6},          // hashes to [011]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "001", "010"),
					newTestNode("1", nil, "", ""),   // leaf
					newTestNode("001", nil, "", ""), // leaf
					newTestNode("010", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0", nil, "001", "01"),
					newTestNode("1", nil, "", ""), // leaf
					newTestNode("01", nil, "010", "011"),
					newTestNode("001", nil, "", ""),                  // leaf
					newTestNode("010", nil, "", ""),                  // leaf
					newTestNode("011", []byte("some_value"), "", ""), // new leaf
					newTestNode( // root
						"110",
						func() []byte {
							// great-grandchildren
							input010 := append(keyBytesFromStr("010"), []byte{}...)
							input011 := append(keyBytesFromStr("011"), crypto.Hash([]byte("some_value"))...)
							// grandchildren
							input01 := append(keyBytesFromStr("01"), crypto.Hash(append(input010, input011...))...)
							input001 := append(keyBytesFromStr("001"), []byte{}...)
							//  children
							input0 := append(keyBytesFromStr("0"), crypto.Hash(append(input001, input01...))...)
							input1 := append(keyBytesFromStr("1"), []byte{}...)
							// root
							return crypto.Hash(append(input0, input1...))
						}(),
						"0", "1"),
				},
			},
		},
		{
			name: "update and target at 101",
			detail: `BEFORE    root
			                           / \
			                          0   1
			                              / \
			                             10  11
			                            / \
			                           100 101

				          AFTER    root
			                        / \
			                       0   1
			                          / \
			                         10  11
			                         / \
			                      100 *101*
			                          `,
			keyBitSize:  3,
			rootKey:     []byte{0b10010000}, // arbitrary
			targetKey:   []byte{8},          // hashes to [101]
			targetValue: []byte("some_value"),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "", ""),     // leaf
					newTestNode("1", nil, "10", "111"),
					newTestNode("10", nil, "100", "101"),
					newTestNode("111", nil, "", ""), // leaf
					newTestNode("100", nil, "", ""), // leaf
					newTestNode("101", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0", nil, "", ""), // leaf
					newTestNode("1", nil, "10", "111"),
					newTestNode("10", nil, "100", "101"),
					newTestNode("100", nil, "", ""),                  // leaf
					newTestNode("101", []byte("some_value"), "", ""), // updated
					newTestNode( // root
						"110",
						func() []byte {
							// great-grandchildren
							input100 := append(keyBytesFromStr("100"), []byte{}...)
							input101 := append(keyBytesFromStr("101"), crypto.Hash([]byte("some_value"))...)
							// grandchildren
							input10 := append(keyBytesFromStr("10"), crypto.Hash(append(input100, input101...))...)
							input111 := append(keyBytesFromStr("111"), []byte{}...)
							//  children
							input1 := append(keyBytesFromStr("1"), crypto.Hash(append(input10, input111...))...)
							input0 := append(keyBytesFromStr("0"), []byte{}...)
							// root
							return crypto.Hash(append(input0, input1...))
						}(),
						"0", "1"),
					newTestNode("111", nil, "", ""), // leaf
				},
			},
		},
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
					newTestNode(
						"100", nil, "0", "1",
					),
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "111", "101"),
					newTestNode("000", nil, "", ""), // leaf
					newTestNode("010", nil, "", ""), // leaf
					newTestNode("111", nil, "", ""), // leaf
					newTestNode("101", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0", nil, "000", "010"),
					newTestNode("000", nil, "", ""),
					newTestNode("1", nil, "111", "101"),
					newTestNode("010", []byte("some_value"), "", ""), // updated
					newTestNode( // root
						"100",
						func() []byte {
							// NOTE: the tree values on the right side are nulled, so the inputs for the right side are incomplete
							// grandchildren
							input000, input010 := keyBytesFromStr("000"), append(keyBytesFromStr("010"), crypto.Hash([]byte("some_value"))...)
							// children
							input0 := append(keyBytesFromStr("0"), crypto.Hash(append(input000, input010...))...)
							input1 := append(keyBytesFromStr("1"), []byte{}...)
							// root value
							return crypto.Hash(append(input0, input1...))
						}(),
						"0",
						"1",
					),
					newTestNode("101", nil, "", ""),
					newTestNode("111", nil, "", ""),
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
					newTestNode("1001", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1000", "11"),
					// new parent
					newTestNode("11", nil, "1101", "111"),
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("1001", func() []byte {
						// great-grandchildren
						input1101, input111 := append(
							keyBytesFromStr("1101"),
							crypto.Hash([]byte("some_value"))...),
							append(keyBytesFromStr("111"), []byte{}...)
						// grandchildren
						input1000, input11 := keyBytesFromStr("1000"),
							append(keyBytesFromStr("11"), crypto.Hash(append(input1101, input111...))...)
						// children
						input0000, input1 := keyBytesFromStr("0000"),
							append(keyBytesFromStr("1"), crypto.Hash(append(input1000, input11...))...)
						// root value
						return crypto.Hash(append(input0000, input1...))
					}(),
						"0000",
						"1"),
					newTestNode("1101", nil, "", ""), // leaf
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
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

						AFTER:         root
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
					newTestNode("1001", nil, "0000", "1"),
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0", nil, "0000", "0110"), // new parent
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("0110", nil, "", ""), // inserted
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("1001", // root
						func() []byte {
							// grandchildren
							input0000, input0110 := keyBytesFromStr("0000"), append(keyBytesFromStr("0110"), crypto.Hash([]byte("some_value"))...)
							// children
							input0, input1 := append(keyBytesFromStr("0"), crypto.Hash(append(input0000, input0110...))...), keyBytesFromStr("1")
							// root value
							return crypto.Hash(append(input0, input1...))
						}(), "0", "1"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
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
						Key: &key{key: []byte{0b10010000, 0, 0}},
						//Key: &key{mostSigBytes: []byte{0b10010000}, leastSigBits: []int{0}}, // arbitrary
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 1},             // 00
							RightChildKey: []byte{0b11111111, 0b1, 0}, // 111111111
						},
					},
					{ // 00
						Key: &key{key: []byte{0, 1}},
						//Key: &key{leastSigBits: []int{0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b00000000, 0b0, 0}, // 000000000
							RightChildKey: []byte{0b00111111, 0b1, 0}, // 001111111
						},
					},
					{ // 000000000
						Key: &key{key: []byte{0b00000000, 0, 0}},
						//Key:  &key{mostSigBytes: []byte{0b00000000}, leastSigBits: []int{0}},
						Node: lib.Node{}, // leaf
					},
					{ // 001111111
						Key: &key{key: []byte{0b00111111, 1, 0}},
						//Key:  &key{mostSigBytes: []byte{0b00111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
					{ // 111111111
						Key: &key{key: []byte{0b11111111, 1, 0}},
						//Key:  &key{mostSigBytes: []byte{0b11111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{ // 000000000
						Key: &key{key: []byte{0b00000000, 0, 0}},
						//Key:  &key{mostSigBytes: []byte{0b00000000}, leastSigBits: []int{0}},
						Node: lib.Node{}, // leaf
					},
					{ // 00
						Key: &key{key: []byte{0b0, 1}},
						//Key: &key{leastSigBits: []int{0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b0, 3},             // 0000
							RightChildKey: []byte{0b00111111, 0b1, 0}, // 001111111
						},
					},
					{ // 0000
						Key: &key{key: []byte{0b0, 3}},
						//Key: &key{leastSigBits: []int{0, 0, 0, 0}},
						Node: lib.Node{
							LeftChildKey:  []byte{0b00000000, 0b0, 0}, // 000000000
							RightChildKey: []byte{0b00001000, 0b0, 0}, // 000010000
						},
					},
					{ // 000010000
						Key: &key{key: []byte{0b00001000, 0, 0}},
						//Key:  &key{mostSigBytes: []byte{0b00001000}, leastSigBits: []int{0}},
						Node: lib.Node{Value: []byte("some_value")}, // leaf
					},
					{ // 001111111
						Key: &key{key: []byte{0b00111111, 1, 0}},
						//Key:  &key{mostSigBytes: []byte{0b00111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
					{ // root
						Key: &key{key: []byte{0b10010000, 0, 0}},
						//Key: &key{mostSigBytes: []byte{0b10010000}, leastSigBits: []int{0}}, // arbitrary
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
						Key: &key{key: []byte{0b11111111, 1, 0}},
						//Key:  &key{mostSigBytes: []byte{0b11111111}, leastSigBits: []int{1}},
						Node: lib.Node{}, // leaf
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				// create a new SMT
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				// close the store when done
				defer memStore.Close()
				// execute the traversal code
				unsortedOps := map[uint64]valueOp{}
				unsortedOps[lib.MemHash(test.targetKey)] = valueOp{key: test.targetKey, value: test.targetValue, op: opSet}
				// commit the set
				require.NoError(t, smt.Commit(unsortedOps))
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
					require.Equal(t, test.expected.Nodes[i].Key.bytes(), got.Key.bytes(), fmt.Sprintf("Key Iteration: %d on node %v", i, got.Key.bytes()))
					require.Equal(t, test.expected.Nodes[i].LeftChildKey, got.LeftChildKey, fmt.Sprintf("Left Child Key Iteration: %d on node %v", i, got.Key.key))
					require.Equal(t, test.expected.Nodes[i].RightChildKey, got.RightChildKey, fmt.Sprintf("Right Child Key Iteration: %d on node %v", i, got.Key.key))
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
			name: "delete with target at 110",
			detail: `BEFORE:   root
								  /    \
							 	000    11
			                          /  \
			                        110  111

						AFTER:      root
								   /    \
	                           000    111
								`,
			keyBitSize: 3,
			rootKey:    []byte{0b10010000}, // arbitrary
			targetKey:  []byte{2},          // hashes to [110]
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("100", nil, "000", "11"), // root
					newTestNode("000", nil, "", ""),      // leaf
					newTestNode("11", nil, "110", "111"),
					newTestNode("110", nil, "", ""), // leaf
					newTestNode("111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("000", nil, "", ""), // leaf
					newTestNode("100",
						func() []byte { // root
							// children
							input000 := append(keyBytesFromStr("000"), []byte{}...)
							input111 := append(keyBytesFromStr("111"), []byte{}...)
							// root value
							return crypto.Hash(append(input000, input111...))
						}(), "000", "111"),
					newTestNode("111", nil, "", ""), // leaf
				},
			},
		},
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
					newTestNode("100", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "111", "101"),
					newTestNode("000", nil, "", ""), // leaf
					newTestNode("010", nil, "", ""), // leaf
					newTestNode("111", nil, "", ""), // leaf
					newTestNode("101", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("000", nil, "", ""),     // leaf
					newTestNode("1", nil, "111", "101"), // leaf
					newTestNode("100",
						func() []byte { // root
							// NOTE: the tree values on the right side are nulled, so the inputs for the right side are incomplete
							// children
							input000 := append(keyBytesFromStr("000"), []byte{}...)
							input1 := append(keyBytesFromStr("1"), []byte{}...)
							// root value
							return crypto.Hash(append(input000, input1...))
						}(), "000", "1"),
					newTestNode("101", nil, "", ""), // leaf
					newTestNode("111", nil, "", ""), // leaf
				},
			},
		},
		{
			name: "Delete and target at 1 1 1 0",
			detail: `BEFORE:   root
									  /    \
								    0000    1
										  /   \
									    1011   111
											  /   \
										   *1110* 1111

							AFTER:     root
									  /     \
				                      0000       1
				                                /  \
				                             1011  1111
									`,
			keyBitSize: 4,
			rootKey:    []byte{0b10010000},
			targetKey:  []byte{4}, // hashes to [1 1 1 0]
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"),
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1011", "111"),
					newTestNode("1011", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1011", "1111"),
					newTestNode("1001", // root
						func() []byte {
							// grandChildren
							input1011 := keyBytesFromStr("1011")
							input1111 := keyBytesFromStr("1111")
							// children
							input0000 := keyBytesFromStr("0000")
							// 1 needs to be rehashed as now it has a new child node
							input1 := append(keyBytesFromStr("1"), crypto.Hash(append(input1011, input1111...))...)
							// root value
							return crypto.Hash(append(input0000, input1...))
						}(), "0000", "1"),
					newTestNode("1011", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
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
					newTestNode("1001", nil, "0000", "1"),
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1011", "111"),
					newTestNode("1011", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1001", // root
						func() []byte {
							// NOTE: 111 hash not updated, so use key only as there's no value preset
							// children
							in0000, in111 := keyBytesFromStr("0000"), keyBytesFromStr("111")
							// root value
							return crypto.Hash(append(in0000, in111...))
						}(), "0000", "111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
		},
		{
			name: "delete (not exists) and target at 011",
			detail: `BEFORE:       root     *011* <- target not exists
								  /     \
							    0        10
							  /   \     /  \
						    001   010  100  101

						After:     root
								  /     \
							    0        10
							  /   \     /  \
						    001   010  100  101
								`,
			keyBitSize: 3,
			rootKey:    []byte{0b10010000},
			targetKey:  []byte{6}, // hashes to [011]
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "10"), // root
					newTestNode("0", nil, "001", "010"),
					newTestNode("001", nil, "", ""), // leaf
					newTestNode("10", nil, "100", "101"),
					newTestNode("010", nil, "", ""), // leaf
					newTestNode("100", nil, "", ""), // leaf
					newTestNode("101", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0", nil, "001", "010"),
					newTestNode("001", nil, "", ""), // leaf
					newTestNode("10", nil, "100", "101"),
					newTestNode("010", nil, "", ""), // leaf
					newTestNode("100", nil, "", ""), // leaf
					newTestNode("101", nil, "", ""), // leaf
					newTestNode("110", func() []byte {
						// NOTE: nothing was updated - but rehash() is executed regardless so values must be updated
						// NOTE: there's no values in the nodes so just use keys
						in001, in010 := keyBytesFromStr("001"), keyBytesFromStr("010")
						in0, in10 := append(keyBytesFromStr("0"), crypto.Hash(append(in001, in010...))...), keyBytesFromStr("10")
						// root value
						return crypto.Hash(append(in0, in10...))
					}(), "0", "10"), // root
				},
			},
		},
		{
			name: "delete (not exists) and target at 1101",
			detail: `BEFORE:       root     *1101* <- target not exists
								  /    \
							    0000     1
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
					newTestNode("1001", nil, "0000", "1"),
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					newTestNode("0000", nil, "", ""), // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("111", nil, "1110", "1111"), // leaf
					newTestNode("1000", nil, "", ""),        // leaf
					newTestNode("1001", func() []byte {
						// NOTE: nothing was updated - but rehash() is executed regardless so values must be updated
						// NOTE: there's no values in the nodes so just use keys
						in1000, in111 := keyBytesFromStr("1000"), keyBytesFromStr("111")
						in0000, in1 := keyBytesFromStr("0000"), append(keyBytesFromStr("1"), crypto.Hash(append(in1000, in111...))...)
						// root value
						return crypto.Hash(append(in0000, in1...))
					}(), "0000", "1"), // root
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				// create a new SMT
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				// close the store when done
				defer memStore.Close()
				unsortedOps := map[uint64]valueOp{}
				unsortedOps[lib.MemHash(test.targetKey)] = valueOp{key: test.targetKey, op: opSet}
				// commit the result
				require.NoError(t, smt.Commit(unsortedOps))
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
					require.Equal(t, test.expected.Nodes[i].Key.bytes(), got.Key.bytes(), fmt.Sprintf("Iteration: %d on node %v", i, got.Key.key))
					require.Equal(t, test.expected.Nodes[i].LeftChildKey, got.LeftChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.key))
					require.Equal(t, test.expected.Nodes[i].RightChildKey, got.RightChildKey, fmt.Sprintf("Iteration: %d on node %v", i, got.Key.key))
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
			target:     newTestNode("000", nil, "", ""),
			expectedTraversal: &NodeList{Nodes: []*node{
				newTestNode("111", func() []byte { // root
					// left child key + value
					leftInput := append(keyBytesFromStr("000"), bytes.Repeat([]byte{0}, 20)...)
					// right child key + value
					rightInput := append(keyBytesFromStr("111"), bytes.Repeat([]byte{255}, 20)...)
					// hash ( left + right )
					return crypto.Hash(append(leftInput, rightInput...))
				}(),
					"000", "111"),
			}},
			expectedCurrent: newTestNode("000", bytes.Repeat([]byte{0}, 20), "", ""),
		},
		{
			name: "basic traversal, no preset (Right - 3bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the max hash
                             root
							  /    \
							000   *111*`,
			keyBitSize: 3,
			target:     newTestNode("111", nil, "", ""),
			expectedTraversal: &NodeList{Nodes: []*node{
				newTestNode("111", func() []byte {
					// left child key + value
					leftInput := append(keyBytesFromStr("000"), bytes.Repeat([]byte{0}, 20)...)
					// right child key + value
					rightInput := append(keyBytesFromStr("111"), bytes.Repeat([]byte{255}, 20)...)
					// hash ( left + right )
					return crypto.Hash(append(leftInput, rightInput...))
				}(), "000", "111"),
			}},
			expectedCurrent: newTestNode("111", bytes.Repeat([]byte{255}, 20), "", ""),
		},
		{
			name: "basic traversal, no preset (Left - 4bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the min hash
                             root
							  /    \
						   *0000*   1111`,
			keyBitSize: 4,
			target:     newTestNode("0000", nil, "", ""),
			expectedTraversal: &NodeList{Nodes: []*node{
				newTestNode("1111",
					func() []byte {
						// left child key + value
						leftInput := append(keyBytesFromStr("0000"), bytes.Repeat([]byte{0}, 20)...)
						// right child key + value
						rightInput := append(keyBytesFromStr("1111"), bytes.Repeat([]byte{255}, 20)...)
						// hash ( left + right )
						return crypto.Hash(append(leftInput, rightInput...))
					}(), "0000", "1111"),
			}},
			expectedCurrent: newTestNode("0000", bytes.Repeat([]byte{0}, 20), "", ""),
		},
		{
			name: "basic traversal, no preset (Right - 5bit)",
			detail: `there's no preset - so traversed should only have root and the current should be the max hash
                             root
							  /    \
							00000  *11111*`,
			keyBitSize: 5,
			target:     newTestNode("11111", nil, "", ""),
			expectedTraversal: &NodeList{Nodes: []*node{
				newTestNode("11111",
					func() []byte {
						// left child key + value
						leftInput := append([]byte{0, 4}, bytes.Repeat([]byte{0}, 20)...)
						// right child key + value
						rightInput := append([]byte{31, 0}, bytes.Repeat([]byte{255}, 20)...)
						// hash ( left + right )
						return crypto.Hash(append(leftInput, rightInput...))
					}(), "00000", "11111"),
			}},
			expectedCurrent: newTestNode("11111", bytes.Repeat([]byte{255}, 20), "", ""),
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
			target:     newTestNode("1110", nil, "", ""),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", []byte("some_value"), "", ""), // leaf
					newTestNode("1111", nil, "", ""),                  // leaf
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"), // root
					newTestNode("1", nil, "1000", "111"),
					newTestNode("111", nil, "1110", "1111"),
				},
			},
			expectedCurrent: newTestNode("1110", []byte("some_value"), "", ""),
			rootKey:         []byte{0b10010000},
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
			target:     &node{Key: &key{key: []byte{0b1100, 0}}},
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"), // root
					newTestNode("1", nil, "1000", "111"),
				},
			},
			expectedCurrent: newTestNode("111", nil, "1110", "1111"),
			rootKey:         []byte{0b10010000},
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
			target:     &node{Key: &key{key: []byte{0b1, 3}}},
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"),             // root
					newTestNode("0000", []byte("some_value"), "", ""), // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					newTestNode("1001", nil, "0000", "1"), // root
				},
			},
			expectedCurrent: newTestNode("0000", []byte("some_value"), "", ""),
			rootKey:         []byte{0b10010000},
		},
		{
			name: "traversal with preset and target at 010",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000 *010*  101 111
							`,
			keyBitSize: 3,
			target:     newTestNode("010", nil, "", ""),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "101", "111"),
					newTestNode("000", nil, "", ""),                  // leaf
					newTestNode("010", []byte("some_value"), "", ""), // leaf
					newTestNode("101", nil, "", ""),                  // leaf
					newTestNode("111", nil, "", ""),                  // leaf
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
				},
			},
			expectedCurrent: newTestNode("010", []byte("some_value"), "", ""),
			rootKey:         []byte{0b10010000},
		},
		{
			name: "traversal with preset and target at 101",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000   010 *101* 111
							`,
			keyBitSize: 3,
			target:     newTestNode("101", nil, "", ""),
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "101", "111"),
					newTestNode("000", nil, "", ""),                  // leaf
					newTestNode("010", nil, "", ""),                  // leaf
					newTestNode("101", []byte("some_value"), "", ""), // leaf
					newTestNode("111", nil, "", ""),                  // leaf
				},
			},
			expectedTraversal: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("1", nil, "101", "111"),
				},
			},
			expectedCurrent: newTestNode("101", []byte("some_value"), "", ""),
			rootKey:         []byte{0b10010000},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				// create a new SMT
				smt, memStore := NewTestSMT(t, test.preset, nil, test.keyBitSize)
				// close the store when done
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
							key: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0b0, 7},
							//mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							//leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							key: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
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
							key: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
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
							key: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0b0, 7},
							//mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							//leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							key: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
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
							key: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)},
					},
				},
			},
			expected: &NodeList{
				Nodes: []*node{
					{
						Key: &key{
							key: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0b0, 7},
							//mostSigBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							//leastSigBits: []int{0, 0, 0, 0, 0, 0, 0, 0},
						},
						Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)},
					},
					{
						Key: &key{
							key: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
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
							key: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0b11111111, 0},
							//mostSigBytes: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
							//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
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
					require.NoError(t, memStore.Set(lib.JoinLenPrefix(n.Key.bytes()), nodeBytes))
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
		shouldPanic    bool
	}{
		{
			name: "0000 partial",
			target: &key{
				key: []byte{0, 3},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0},
			},
			current: &key{
				key: []byte{0, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{0},
			},
			gcp:    &key{},
			bitPos: 0,
			expectedGCP: &key{
				key: []byte{0, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{0},
			},
			expectedBitPos: 1,
		},
		{
			name: "00000001 0111 full",
			target: &key{
				key: []byte{byte(1), 0b111, 1},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 1, 1, 1},
			},
			current: &key{
				key: []byte{byte(1), 0b111, 1},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 1, 1, 1},
			},
			gcp:    &key{},
			bitPos: 0,
			expectedGCP: &key{
				key: []byte{byte(1), 0b111, 1},
			},
			expectedBitPos: 12,
		},
		{
			name: "11111111 000 full",
			target: &key{
				key: []byte{255, 0, 2},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{0, 0, 0},
			},
			current: &key{
				key: []byte{255, 0, 2},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{0, 0, 0},
			},
			gcp:    &key{},
			bitPos: 0,
			expectedGCP: &key{
				key: []byte{255, 0, 2},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{0, 0, 0},
			},
			expectedBitPos: 11,
		},
		{
			name: "00000001 0111 partial",
			target: &key{
				key: []byte{1, 0b111, 1},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 1, 1, 1},
			},
			current: &key{
				key: []byte{1, 0b1, 1},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 1},
			},
			gcp:    &key{},
			bitPos: 0,
			expectedGCP: &key{
				key: []byte{1, 0b1, 1},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 1},
			},
			expectedBitPos: 10,
		},
		{
			name: "11111111 000 partial",
			target: &key{
				key: []byte{255, 0, 2},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{0, 0, 0},
			},
			current: &key{
				key: []byte{255, 0},
				//mostSigBytes: []byte{},
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			gcp:    &key{},
			bitPos: 0,
			expectedGCP: &key{
				key: []byte{255, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expectedBitPos: 8,
		},
		{
			name: "000011 continue",
			target: &key{
				key: []byte{0b11, 4},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				key: []byte{0b1, 4},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0, 1},
			},
			gcp: &key{
				key: []byte{0b0, 2},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				key: []byte{0b1, 4},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 5,
		},
		{
			name: "00000001 000011 continue",
			target: &key{
				key: []byte{0b1, 0b11, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				key: []byte{0b1, 0b1, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1},
			},
			gcp: &key{
				key: []byte{0b0, 2},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				key: []byte{0b1, 0b1, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 13,
		},
		{
			name: "00000001 000011 continue not exact",
			target: &key{
				key: []byte{0b1, 0b11, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1, 1},
			},
			current: &key{
				key: []byte{0b1, 0b10, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1, 0},
			},
			gcp: &key{
				key: []byte{0b0, 2},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0},
			},
			bitPos: 3,
			expectedGCP: &key{
				key: []byte{0b1, 0b1, 4},
				//mostSigBytes: []byte{1},
				//leastSigBits: []int{0, 0, 0, 0, 1},
			},
			expectedBitPos: 13,
		},
		{
			name: "0 001 panic current greater than target",
			target: &key{
				key: []byte{0, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{0},
			},
			current: &key{
				key: []byte{1, 2},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 1},
			},
			bitPos:         1,
			shouldPanic:    true,
			expectedBitPos: 2,
		},
		{
			name:        "nil nil panic nil current and target",
			target:      nil,
			current:     nil,
			shouldPanic: true,
		},
		{
			name: "0000 nil panic nil current",
			target: &key{
				key: []byte{0, 3},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0},
			},
			current:     nil,
			shouldPanic: true,
		},
		{
			name: "nil 0000 panic nil target",
			current: &key{
				key: []byte{0, 3},
				//mostSigBytes: nil,
				//leastSigBits: []int{0, 0, 0, 0},
			},
			target:      nil,
			shouldPanic: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					// check whether the test should panic
					require.Equal(t, r != nil, test.shouldPanic)
				}
			}()
			for i := 0; i < test.expectedGCP.totalBits(); i++ {
				if i%8 == 0 && i != 0 {
					fmt.Print("_")
				}
				fmt.Printf("%d", test.expectedGCP.bitAt(i))
			}
			fmt.Println("_")
			// Compare the greatest common prefix between the target and the current key
			test.target.greatestCommonPrefix(&test.bitPos, test.gcp, test.current)
			// Compare the results
			require.Equal(t, test.expectedGCP.bytes(), test.gcp.bytes())
			require.Equal(t, test.expectedBitPos, test.bitPos)
		})
	}
}

func TestStoreProof(t *testing.T) {
	tests := []struct {
		name               string
		detail             string
		keyBitSize         int
		preset             *NodeList
		valid              bool
		validateMembership bool
		proofErr           error
		targetKey          []byte
		targetValue        []byte
	}{
		{
			name: "valid proof of membership with target at 010",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000  *010* 101 111
							`,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "101", "111"),
					newTestNode("000", nil, "", ""),                  // leaf
					newTestNode("010", []byte("some_value"), "", ""), // leaf
					newTestNode("101", nil, "", ""),                  // leaf
					newTestNode("111", nil, "", ""),                  // leaf
				},
			},
			keyBitSize:         3,
			validateMembership: true,
			valid:              true,
			targetKey:          []byte{1}, // hashes to [010]
			targetValue:        []byte("some_value"),
		},
		{
			name: "valid proof of non-membership with target at 011",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000  *010* 101 111
							`,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "101", "111"),
					newTestNode("000", nil, "", ""),                  // leaf
					newTestNode("010", []byte("some_value"), "", ""), // leaf
					newTestNode("101", nil, "", ""),                  // leaf
					newTestNode("111", nil, "", ""),                  // leaf
				},
			},
			keyBitSize:         3,
			validateMembership: false,
			valid:              true,
			targetKey:          []byte{6}, // hashes to [011]
			targetValue:        []byte("some_value"),
		},
		{
			name: "invalid proof of membership with target at 010 (key exist, values differ)",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000  *010* 101 111
							`,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("110", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "101", "111"),
					newTestNode("000", nil, "", ""),                  // leaf
					newTestNode("010", []byte("some_value"), "", ""), // leaf
					newTestNode("101", nil, "", ""),                  // leaf
					newTestNode("111", nil, "", ""),                  // leaf
				},
			},
			keyBitSize:         3,
			validateMembership: true,
			valid:              false,
			targetKey:          []byte{1}, // hashes to [010]
			targetValue:        []byte("wrong_value"),
		},
		{
			name: "invalid proof of non membership with target at 110 (key exists)",
			detail: `Preset:   root
							  /    \
						     0       1
						   /  \     /  \
					    000   010 100 *110*
							`,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("001", nil, "0", "1"), // root
					newTestNode("0", nil, "000", "010"),
					newTestNode("1", nil, "100", "110"),
					newTestNode("000", nil, "", ""), // leaf
					newTestNode("010", []byte("some_value"), "", ""),
					newTestNode("100", nil, "", ""), // leaf
					newTestNode("110", nil, "", ""), // leaf
				},
			},
			keyBitSize:         3,
			validateMembership: false,
			valid:              false,
			targetKey:          []byte{2}, // hashes to [110]
			targetValue:        []byte(""),
		},
		{
			name: "valid proof of membership with target at 1000",
			detail: `Preset:      root
		                         /    \
		                       0000    1
		                             /  \
							    *1000*  111
		                                /   \
		                              1110  1111
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", []byte("some_value"), "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""), // leaf
					newTestNode("1111", nil, "", ""), // leaf
				},
			},
			targetKey:          []byte{20}, // hashes to [1 0 0 0]
			targetValue:        []byte("some_value"),
			validateMembership: true,
			valid:              true,
		},
		{
			name: "invalid proof of membership with target at 1110 (key exist, values differ)",
			detail: `Preset:      root
		                         /    \
		                       0000    1
		                             /  \
							      1000   11
		                                /   \
		                              1100  *1110*
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "11"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("11", nil, "1100", "1110"),
					newTestNode("1100", nil, "", ""),                  // leaf
					newTestNode("1110", []byte("some_value"), "", ""), // leaf
				},
			},

			targetKey:          []byte{4}, // hashes to [1 1 1 0]
			targetValue:        []byte("wrong_value"),
			validateMembership: true,
			valid:              false,
		},
		{
			name: "valid proof of non membership with target at 1001",
			detail: `Preset:      root
		                         /    \
		                       0000    1
		                             /  \
							      1000  111
		                                /   \
		                              1110  1111 (does not exist)
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1"), // root
					newTestNode("0000", nil, "", ""),      // leaf
					newTestNode("1", nil, "1000", "111"),
					newTestNode("1000", nil, "", ""), // leaf
					newTestNode("111", nil, "1110", "1111"),
					newTestNode("1110", nil, "", ""),                  // leaf
					newTestNode("1111", []byte("some_value"), "", ""), // leaf
				},
			},

			targetKey:          []byte{13}, // hashes to [1 0 0 1]
			targetValue:        []byte("wrong_value"),
			validateMembership: false,
			valid:              true,
		},
		{
			name: "attempt to verify a root key",
			detail: `Preset:        root (*1011*)
		                         /        \
		                       0000       1111
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1111"), // root
				},
			},
			targetKey:          []byte{8}, // hashes to [1 0 1 1]
			targetValue:        []byte("some_value"),
			validateMembership: false,
			valid:              false,
			proofErr:           ErrReserveKeyWrite("root"),
		},
		{
			name: "attempt to verify a key minimum",
			detail: `Preset:        root (1011)
		                         /        \
						      *0000*     1111
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1111"), // root
				},
			},
			targetKey:   []byte{3}, // hashes to [0 0 0 0]
			targetValue: []byte(""),
			proofErr:    ErrReserveKeyWrite("minimum"),
		},
		{
			name: "attempt to verify a key maximum",
			detail: `Preset:        root (1011)
		                         /        \
		                       0000     *1111*
							`,
			keyBitSize: 4,
			preset: &NodeList{
				Nodes: []*node{
					newTestNode("1011", nil, "0000", "1111"), // root
				},
			},
			targetKey:   []byte{18}, // hashes to [1 1 1 1]
			targetValue: []byte(""),
			proofErr:    ErrReserveKeyWrite("maximum"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// preset must have at least one node in order to set expected root to verify
			require.True(t, len(test.preset.Nodes) > 0, "preset must have at least one node")
			// only preset the root so the smt is created with the root key
			rootPreset := &NodeList{
				Nodes: []*node{
					test.preset.Nodes[0],
				},
			}
			// create the smt
			smt, memStore := NewTestSMT(t, rootPreset, nil, test.keyBitSize)
			// close the store when done
			defer memStore.Close()
			// preset the nodes manually to trigger rehashing
			for _, n := range test.preset.Nodes[1:] {
				// set the node in the db manually (value is hashed before as a normal Set operation would)
				n.Value = crypto.Hash(n.Value)
				require.NoError(t, smt.setNode(n))
				// set the target node as the node just created
				smt.target = n
				smt.reset()
				require.NoError(t, smt.traverse())
				// traverse to the node that was just set
				require.Equal(t, n, smt.target)
				// rehash the tree from the newly created node
				require.NoError(t, smt.rehash())
			}
			// generate the merkle proof
			proof, err := smt.GetMerkleProof(test.targetKey)
			if test.proofErr != nil {
				require.Equal(t, test.proofErr, err)
				return
			}
			// validate proof results
			require.Equal(t, test.proofErr, err)
			// verify the proof
			valid, err := smt.VerifyProof(test.targetKey, test.targetValue,
				test.validateMembership, smt.Root(), proof)
			// validate results
			require.NoError(t, err)
			require.Equal(t, test.valid, valid)
		})
	}
}

func TestKeyCmp(t *testing.T) {
	tests := []struct {
		name     string
		k1       *key
		k2       *key
		expected int
	}{
		{
			name: "equal",
			k1: &key{
				key: []byte{255, 0, 2},
			},
			k2: &key{
				key: []byte{255, 0, 2},
			},
			expected: 0, // equal
		},
		{
			name: "more left padding k1 but equal",
			k1: &key{
				key: []byte{255, 0, 3},
			},
			k2: &key{
				key: []byte{255, 0, 2},
			},
			expected: 0, // equal
		},
		{
			name: "equal full bytes",
			k1: &key{
				key: []byte{255, 255, 255, 0},
				//mostSigBytes: []byte{0b11111110},
				//leastSigBits: []int{0, 0, 0},
			},
			k2: &key{
				key: []byte{255, 255, 255, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expected: 0, // equal
		},
		{
			name: "longer k1 but greater",
			k1: &key{
				key: []byte{0b11111111, 0, 2},
				//mostSigBytes: []byte{0b11111111},
				//leastSigBits: []int{0, 0, 0},
			},
			k2: &key{
				key: []byte{0b11111110, 0},
				//mostSigBytes: nil,
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 0},
			},
			expected: 1, // greater
		},
		{
			name: "longer k1 with full bytes but equal",
			k1: &key{
				key: []byte{255, 255, 0, 2},
				//mostSigBytes: []byte{255, 0b11111111},
				//leastSigBits: []int{0, 0, 0},
			},
			k2: &key{
				key: []byte{255, 255, 0},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expected: 0, // equal
		},
		{
			name: "longer k1 with full bytes but less",
			k1: &key{
				key: []byte{255, 0b11111110, 0, 2},
				//mostSigBytes: []byte{255,0b11111110},
				//leastSigBits: []int{0, 0, 0},
			},
			k2: &key{
				key: []byte{255, 0b11111111, 0},
				//mostSigBytes: []byte{255},
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expected: -1, // less
		},
		{
			name: "longer k1 with full bytes but greater",
			k1: &key{
				key: []byte{255, 0b11111111, 0, 2},
				//mostSigBytes: []byte{255,0b11111111},
				//leastSigBits: []int{0, 0, 0},
			},
			k2: &key{
				key: []byte{254, 0b11111111, 0},
				//mostSigBytes: []byte{254},
				//leastSigBits: []int{1, 1, 1, 1, 1, 1, 1, 1},
			},
			expected: 1, // greater
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.k1.cmp(test.k2))
		})
	}
}

func FuzzKeyDecodeEncode(f *testing.F) {
	// seed corpus
	tests := []struct {
		data []byte
	}{
		// seed input comes from TestKeyEncode
		{[]byte{0, 0}},
		{[]byte{1, 0}},
		{[]byte{0, 1}},
		{[]byte{0, 0, 0}},
		{[]byte{0, 1, 0, 1, 0, 1}},
		{[]byte{5, 255, 5, 0}},
	}
	for _, test := range tests {
		// add the seed to the fuzz test
		f.Add(test.data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// skip invalid test
		if len(data) < 2 {
			t.Skip("Skipping test: key encode requires a minimum of two bytes")
		}
		// create a new key from the fuzz data
		newKey := new(key).fromBytes(data)
		// convert the new key back to bytes
		bytesFromKey := newKey.bytes()
		// compare the resulting bytes against the fuzz data
		require.Equal(t, bytesFromKey, data)
	})
}

func NewTestSMT(t *testing.T, preset *NodeList, root []byte, keyBitSize int) (*SMT, *Txn) {
	// create a new memory store to work with
	fs := vfs.NewMem()
	db, err := pebble.Open("", &pebble.Options{
		DisableWAL:            false,
		FS:                    fs,
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
		MaxOpenFiles:          1000,
		FormatMajorVersion:    pebble.FormatNewest,
	})
	require.NoError(t, err)
	// make a writable reader that reads from the last height
	versionedStore := NewVersionedStore(db.NewSnapshot(), db.NewBatch(), 1)
	require.NoError(t, err)
	memStore := NewTxn(versionedStore, versionedStore, []byte(stateCommitmentPrefix), false, false, 1)
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
		nodeCache:    make(map[string]*node, MaxCacheSize),
		unsortedOps:  make(map[string]*node),
		OpData:       OpData{},
		minKey:       newNodeKey(bytes.Repeat([]byte{byte(0)}, 20), keyBitSize),
		maxKey:       newNodeKey(bytes.Repeat([]byte{byte(255)}, 20), keyBitSize),
	}
	// update root
	smt.root = preset.Nodes[0]
	// preset the nodes
	for _, n := range preset.Nodes {
		// set the node in the dbz
		require.NoError(t, smt.setNode(n))
	}
	// prepare for traversal
	smt.reset()
	return smt, memStore
}

// newTestNode creates a new node with the given key, value, left and right child keys
func newTestNode(k string, value []byte, leftChildKey, rightChildKey string) *node {
	// create the key bytes for the left child
	leftKey := keyBytesFromStr(leftChildKey)
	// create the key bytes for the right child
	rightKey := keyBytesFromStr(rightChildKey)
	// create the node
	return &node{
		Key: &key{
			key: keyBytesFromStr(k),
		},
		Node: lib.Node{
			Value:         value,
			LeftChildKey:  leftKey,
			RightChildKey: rightKey,
		},
	}
}

// keyBytesFromStr converts a string of binary bits to a byte slice
// CONTRACT: str must be < than 8 bits
func keyBytesFromStr(str string) []byte {
	// create a new key from the byts
	var byts []byte
	if str != "" {
		leftPadding := 0
		for _, ch := range str {
			if ch != '0' {
				break
			}
			leftPadding++
		}
		if leftPadding == len(str) {
			leftPadding--
		}
		val, _ := strconv.ParseUint(str, 2, 8)
		// convert the bits to bytes now producing the key
		byts = []byte{byte(val), byte(leftPadding)}
	}
	// return the key bytes
	return byts
}
