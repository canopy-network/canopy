package store

import (
	"bytes"
	"slices"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// =====================================================
// SMT: An optimized sparse Merkle tree
// =====================================================
//
// This is an optimized sparse Merkle tree (SMT) designed for key-value storage.
// It combines properties of prefix trees and Merkle trees to efficiently handle
// sparse datasets and cryptographic integrity.
//
//  - Sparse Structure: Keys are organized by their binary representation,
//     with internal nodes storing common prefixes to reduce redundant paths
//
//  - Merkle Hashing: Each node stores a hash derived from its children, enabling
//     cryptographic proofs for efficient verification of data integrity
//
//  - Optimized Traversals: Operations like insertion, deletion, and lookup focus
//     only on the relevant parts of the tree, minimizing unnecessary traversal of empty nodes
//
//  - Key-Value Operations: Supports upserts and deletions by dynamically creating
//     or removing nodes while maintaining the Merkle tree structure
//
// OPTIMIZATIONS OVER REGULAR SMT:
// 1. Any leaf nodes without values are set to nil. A parent node is also nil if both children are nil
// 2. If a parent has exactly one non-nil child, replace the parent with the non-nil child
// 3. A tree always starts with two children: (0x0...) and (FxF...), and a Root
//
// ALGORITHM:
//	1. Tree Traversal
//	    - Navigate down the tree to set *current* to the closest node based on the binary of the target key
//	2.a Upsert (Insert or Update)
//	    - If the target already matches *current*: Update the existing node
//	    - Otherwise: Create a new node to represent the parent of the target node and *current*
//	    - Replace the pointer to *current* within its old parent with the new parent
//	    - Assign the *current* node and the target as children of the new parent
//	2.b Delete
//	    - If the target matches *current*:
//	      - Delete *current*
//	      - Replace the pointer to *current's parent* within *current's grandparent* with the *current's* sibling
//	      - Delete *current's* parent node
//	3. ReHash
//	    - Update hash values for all ancestor nodes of the modified node, moving upward until the root
//
// Examples:
//
//      INSERT 1101                 DELETE 010
//
//                     BEFORE
//         root                        root
//        /    \                     /      \
//      0000    1                 *0*        1
//            /   \               / \       /  \
//          1000  111          000 *010*  101  111
//               /   \
//             1110  1111
//
//
//                       AFTER
//         root                        root
//        /    \                     /      \
//      0000    1                  000       1
//            /   \                         /  \
//          1000 *11*                     101   111
//               /  \
//           *1101*  111
//                  /   \
//                1110  1111
//
// =====================================================

const MaxKeyBitLength = 160 // the maximum leaf key bits (20 bytes)

type SMT struct {
	// store: an abstraction of the database where the tree is being stored
	store lib.RWStoreI
	// root: the root node
	root *node
	// keyBitLength: the depth of the tree, once set it cannot be changed for a protocol
	keyBitLength int

	// OpData: data for each operation
	OpData
}

// node wraps protobuf Node with a key
type node struct {
	// Key: the structure that is used to interpret node keys (bytes, fromBytes, etc.)
	Key *key
	// Node: is the structure persisted on disk under the above key bytes
	lib.Node
}

// OpData: data for each operation (set, delete)
type OpData struct {
	// gcp: The greatest common prefix between the Target and Current’s keys, representing the shared path
	gcp *key
	// bitPos: The bit position of the bit after the gcp in target_key
	bitPos int
	// pathBit: The bit at bitPos
	pathBit int
	// target: the node that is being added or deleted (or its ID)
	target *node
	// current: the current selected node
	current *node
	// traversed: a descending list of traversed nodes from root to parent of current
	traversed *NodeList
}

const (
	// leftChild: enum identifier of left child (0)
	leftChild = iota
	// leftChild: enum identifier of right child (1)
	rightChild
)

// NewDefaultSMT() creates a new abstraction fo the SMT object using default parameters
func NewDefaultSMT(store lib.RWStoreI) (smt *SMT) {
	return NewSMT(RootKey, MaxKeyBitLength, store)
}

// NewSMT() creates a new abstraction of the SMT object
func NewSMT(rootKey []byte, keyBitLen int, store lib.RWStoreI) (smt *SMT) {
	var err lib.ErrorI
	// create a new smt object
	smt = &SMT{
		store:        store,
		keyBitLength: keyBitLen,
	}
	// ensure the root key is the proper length based on the bit count
	rKey := newNodeKey(bytes.Clone(rootKey), keyBitLen)
	// get the root from the store
	smt.root, err = smt.getNode(rKey.bytes())
	if err != nil {
		panic(err)
	}
	// if the root is empty, initialize with min and max node
	if smt.root.LeftChildKey == nil {
		smt.initializeTree(rKey)
	}
	return
}

// Root() returns the root value of the smt
func (s *SMT) Root() []byte { return bytes.Clone(s.root.Value) }

// Set: insert or update a target
func (s *SMT) Set(k, v []byte) lib.ErrorI {
	// calculate the key and value to upsert
	s.target = &node{Key: newNodeKey(crypto.Hash(k), s.keyBitLength), Node: lib.Node{Value: crypto.Hash(v)}}
	// check to make sure the target is valid
	if err := s.validateTarget(); err != nil {
		return err
	}
	// navigates the tree downward
	if err := s.traverse(); err != nil {
		return err
	}
	// if gcp != target key then it is an insert not an update
	if !s.target.Key.equals(s.gcp) {
		// create a new node (new parent of current and target)
		newParent := newNode()
		newParent.Key = s.gcp
		// get the parent (soon to be grandparent) of current
		oldParent := s.traversed.Parent()
		// calculate current's bytes by encoding
		currentBytes, targetBytes := s.current.Key.bytes(), s.target.Key.bytes()
		// replace the reference to Current in its parent with the new parent
		oldParent.replaceChild(currentBytes, newParent.Key.bytes())
		// set current and target as children of new parent
		// NOTE: the old parent is now the grandparent of target and current
		switch s.pathBit = s.target.Key.bitAt(s.bitPos); s.pathBit {
		case 0:
			newParent.setChildren(targetBytes, currentBytes)
		case 1:
			newParent.setChildren(currentBytes, targetBytes)
		}
		// add new node to traversed list, as it's the new parent for current and target
		// and should come after the grandparent (previously parent)
		s.traversed.Nodes = append(s.traversed.Nodes, newParent.copy())
	}
	// set the node in the database
	if err := s.setNode(s.target); err != nil {
		return err
	}
	// finish with a rehashing of the tree
	return s.rehash()
}

// Delete: removes a target node if exists in the tree
func (s *SMT) Delete(k []byte) lib.ErrorI {
	// calculate the key and value to upsert
	s.target = &node{Key: newNodeKey(crypto.Hash(k), s.keyBitLength)}
	// check to make sure the target is valid
	if err := s.validateTarget(); err != nil {
		return err
	}
	// navigates the tree downward
	if err := s.traverse(); err != nil {
		return err
	}
	// if gcp != target key then there is no delete because the node does not exist
	if !s.target.Key.equals(s.gcp) {
		return nil
	}
	// calculate target key bytes
	targetBytes := s.target.Key.bytes()
	// get the parent and grandparent
	parent, grandparent := s.traversed.Parent(), s.traversed.GrandParent()
	// get the sibling of the target
	sibling, _ := parent.getOtherChild(targetBytes)
	// replace the parent reference with the sibling in the grandparent
	grandparent.replaceChild(parent.Key.bytes(), sibling)
	// delete the parent from the database and remove it from the traversal array
	if err := s.delNode(parent.Key.bytes()); err != nil {
		return err
	}
	// remove the parent from the traversed list
	s.traversed.Pop()
	// delete the target from the database
	if err := s.delNode(targetBytes); err != nil {
		return err
	}
	// finish with a rehashing of the tree
	return s.rehash()
}

// traverse: navigates the tree downward to locate the target or its closest position
func (s *SMT) traverse() (err lib.ErrorI) {
	s.reset()
	// execute main loop
	for {
		var currentKey []byte
		// add current to traversed
		s.traversed.Nodes = append(s.traversed.Nodes, s.current.copy())
		// decide to move left or right based on the bit-value of the key
		switch s.pathBit = s.target.Key.bitAt(s.bitPos); s.pathBit {
		case 0: // move down to the left
			currentKey = s.current.LeftChildKey
		case 1: // move down to the right
			currentKey = s.current.RightChildKey
		}
		// load current node from the store
		s.current, err = s.getNode(currentKey)
		if err != nil {
			return
		}
		if s.current == nil {
			return ErrInvalidMerkleTree()
		}
		// load the bytes into the key
		s.current.Key.fromBytes(currentKey)
		// update the greatest common prefix and the bit position based on the new current key
		s.target.Key.greatestCommonPrefix(&s.bitPos, s.gcp, s.current.Key)
		// exit conditions
		if !s.current.Key.equals(s.gcp) || s.target.Key.equals(s.gcp) {
			return // exit loop
		}
	}
}

// rehash() recalculate hashes from the current node upwards
func (s *SMT) rehash() lib.ErrorI {
	// create a convenience variable for the max index of the array
	maxIdx := len(s.traversed.Nodes) - 1
	// iterate the traversed list from end to start
	for i := maxIdx; i >= 0; i-- {
		// child stores the cached from the parent that was traversed
		var child *node
		// select the parent
		parent := s.traversed.Nodes[i]
		// get a child from the traversed list if possible
		if i != maxIdx {
			child = s.traversed.Nodes[i+1]
		}
		// calculate its new value
		if err := s.updateParentValue(parent, child); err != nil {
			return err
		}
		// set node in the database
		if err := s.setNode(parent); err != nil {
			return err
		}
	}
	return nil
}

// initializeTree() ensures the tree always has a root with two children
// this allows the logic to be without root edge cases for insert and delete
func (s *SMT) initializeTree(rootKey *key) {
	// create a min and max node, this enables no edge cases for root
	minNode := &node{Key: newNodeKey(bytes.Repeat([]byte{0}, 20), s.keyBitLength), Node: lib.Node{Value: bytes.Repeat([]byte{0}, 20)}}
	maxNode := &node{Key: newNodeKey(bytes.Repeat([]byte{255}, 20), s.keyBitLength), Node: lib.Node{Value: bytes.Repeat([]byte{255}, 20)}}
	// set min and max node in the database
	if err := s.setNode(minNode); err != nil {
		panic(err)
	}
	if err := s.setNode(maxNode); err != nil {
		panic(err)
	}
	// update root
	s.root = &node{
		Key: rootKey,
		Node: lib.Node{
			LeftChildKey:  minNode.Key.bytes(),
			RightChildKey: maxNode.Key.bytes(),
		},
	}
	// update the root's value
	if err := s.updateParentValue(s.root, minNode); err != nil {
		panic(err)
	}
	// set the root in store
	if err := s.setNode(s.root); err != nil {
		panic(err)
	}
}

// updateParentValue() updates the value of parent based on its children
func (s *SMT) updateParentValue(parent, child *node) (err lib.ErrorI) {
	var rightChild, leftChild *node
	// if there's a child from the traversed list
	if child != nil {
		// determine if it's the right or left child for the parent
		if bytes.Equal(parent.RightChildKey, child.Key.bytes()) {
			rightChild = child
		} else {
			leftChild = child
		}
	}
	// calculate the left child input
	leftChildInput, err := s.childInput(parent.LeftChildKey, leftChild)
	if err != nil {
		return err
	}
	// calculate the right child input
	rightChildInput, err := s.childInput(parent.RightChildKey, rightChild)
	if err != nil {
		return err
	}
	// concatenate the left and right children values; update the parents value
	parent.Value = crypto.Hash(append(leftChildInput, rightChildInput...))
	// save the updated root value to the structure
	if bytes.Equal(parent.Key.bytes(), s.root.Key.bytes()) {
		s.root = parent.copy()
	}
	return
}

// childInput() returns key + value of the child, retrieving the node from the db if needed
func (s *SMT) childInput(childKey []byte, child *node) (input []byte, err lib.ErrorI) {
	// if the child is not populated
	if child == nil {
		// get the child from the database
		child, err = s.getNode(childKey)
		if err != nil {
			return
		}
	}
	// return key + value
	return append(childKey, child.Value...), nil
}

// reset() resets data for each operation
func (s *SMT) reset() {
	s.current, s.gcp = s.root.copy(), &key{}
	s.pathBit, s.bitPos = 0, 0
	s.traversed = &NodeList{Nodes: make([]*node, 0)}
}

// setNode() set a node object in a key value database
func (s *SMT) setNode(n *node) lib.ErrorI {
	// convert the node object to bytes
	nodeBytes, err := n.bytes()
	if err != nil {
		return err
	}
	// set the bytes under the key in the store
	return s.store.Set(n.Key.bytes(), nodeBytes)
}

// delNode() remove a node from the database given its unique identifier
func (s *SMT) delNode(key []byte) lib.ErrorI {
	return s.store.Delete(key)
}

// getNode() retrieves a node object from the database
func (s *SMT) getNode(key []byte) (n *node, err lib.ErrorI) {
	// initialize a reference to a node object
	n = newNode()
	// get the bytes of the node from the kv store
	nodeBytes, err := s.store.Get(key)
	if err != nil || nodeBytes == nil {
		return
	}
	// convert the node bytes into a node object
	if err = lib.Unmarshal(nodeBytes, n); err != nil {
		return
	}
	// set the key in the node for convenience
	n.Key.fromBytes(key)
	return
}

// validateTarget() checks the target to ensure it's not a reserved key like root, minimum or maximum
func (s *SMT) validateTarget() lib.ErrorI {
	if bytes.Equal(s.root.Key.bytes(), s.target.Key.bytes()) {
		return ErrReserveKeyWrite("root")
	}
	if bytes.Equal(newNodeKey(bytes.Repeat([]byte{0}, 20), s.keyBitLength).bytes(), s.target.Key.bytes()) {
		return ErrReserveKeyWrite("minimum")
	}
	if bytes.Equal(newNodeKey(bytes.Repeat([]byte{255}, 20), s.keyBitLength).bytes(), s.target.Key.bytes()) {
		return ErrReserveKeyWrite("maximum")
	}
	return nil
}

// GetMerkleProof() returns the merkle proof-of-membership for a given key if it exists,
// and the proof of non-membership otherwise
func (s *SMT) GetMerkleProof(k []byte) ([]*lib.Node, lib.ErrorI) {
	// calculate the key and value to traverse
	s.target = &node{Key: newNodeKey(crypto.Hash(k), s.keyBitLength)}
	// check to make sure the target is valid
	if err := s.validateTarget(); err != nil {
		return nil, err
	}
	// make the slice to store the leaf nodes and the intermediate sibling nodes
	proof := make([]*lib.Node, 0)
	// navigates the tree downward
	if err := s.traverse(); err != nil {
		return nil, err
	}
	// add the target node as the initial value of the proof
	proof = append(proof, &lib.Node{
		Key:   s.current.Key.bytes(),
		Value: s.current.Value,
	})
	// Add current to the list of traversed nodes. For membership proofs, traversed nodes include the
	// path to the target node. For non-membership proofs, the potential insertion location is
	// included instead, this is used for proof verification as the binary key (required for parent
	// hash calculation) is not externally known.
	s.traversed.Nodes = append(s.traversed.Nodes, s.current.copy())
	// traverse the nodes back up to the root to generate the proof
	for i := len(s.traversed.Nodes) - 1; i > 0; i-- {
		// get the current node and its parent
		node := s.traversed.Nodes[i]
		parent := s.traversed.Nodes[i-1]
		// use the parent and the current node itself in order to get its sibling
		siblingKey, order := parent.getOtherChild(node.Key.bytes())
		siblingNode, err := s.getNode(siblingKey)
		// check whether the sibling node actually exists
		if err != nil {
			return nil, err
		}
		// add the sibling node to the proof slice
		proof = append(proof, &lib.Node{
			Key:     siblingKey,
			Value:   siblingNode.Value,
			Bitmask: int32(order),
		})
	}
	// return the proof
	return proof, nil
}

// VerifyProof verifies a Sparse Merkle Tree proof for a given value
// reconstructing the root hash and comparing it against the provided root hash
// depending on the proof type (membership or non-membership)
func (s *SMT) VerifyProof(k []byte, v []byte, validateMembership bool, root []byte, proof []*lib.Node) (bool, lib.ErrorI) {
	// shorthand for the length of the proof slice
	proofLen := len(proof)
	// the proof slice must contain at least two nodes: the leaf node and its sibling
	if proofLen < 2 {
		return false, ErrInvalidMerkleTreeProof()
	}
	// The target is always the first value in the proof. For membership
	// proofs, it represents the actual value being verified. For non-membership proofs,
	// it indicates the potential location of the node. The initial root hash
	// can be constructed using this value.
	hash := proof[0].Value
	// currentKey is the key of the sibling node at any given height, it is used to
	// calculate the parent node's key by finding the greatest common prefix (GCP)
	// of the current node's and its sibling's keys
	currentKey := new(key).fromBytes(proof[0].Key)
	// create a new in-memory store to reconstruct the tree
	memStore, err := NewStoreInMemory(lib.NewDefaultLogger())
	if err != nil {
		return false, err
	}
	// Reconstruct a similar Merkle tree using the proof nodes. This allows to traverse
	// the tree again to verify if the given key and value are included in the tree or
	// to confirm proof-of-non-membership if the key is absent.
	smt := NewSMT(RootKey, s.keyBitLength, memStore)
	// set the node being proven in the new tree
	if err := smt.setNode(&node{
		Node: lib.Node{
			Value: proof[0].Value,
			Key:   currentKey.bytes(),
		},
		Key: currentKey,
	}); err != nil {
		return false, err
	}
	// reconstruct the tree from the bottom up using the proof slice
	for i := 1; i < proofLen; i++ {
		// parentNode will be calculated based on the current node and its sibling
		var parentNode *node
		// calculate the hash of the parent node based on the bitmask of the sibling
		if proof[i].Bitmask == leftChild {
			// build the parent node hash based on the left sibling of the given node
			hash = crypto.Hash(
				append(append(proof[i].Key, proof[i].Value...),
					append(currentKey.bytes(), hash...)...),
			)
			// set the parent node's children
			parentNode = &node{
				Node: lib.Node{
					LeftChildKey:  proof[i].Key,
					RightChildKey: currentKey.bytes(),
				},
			}
		} else {
			// build the parent node hash based on the right sibling of the given node
			hash = crypto.Hash(
				append(append(currentKey.bytes(), hash...),
					append(proof[i].Key, proof[i].Value...)...),
			)
			// set the parent node's children
			parentNode = &node{
				Node: lib.Node{
					LeftChildKey:  currentKey.bytes(),
					RightChildKey: proof[i].Key,
				},
			}
		}
		// calculate the key of the parent node by finding the greatest common prefix
		// (GCP) of their children
		nodeKey := new(key).fromBytes(proof[i].Key)
		gcp := new(key)
		// calculate the GCP between the node and the sibling based on the length of
		// the least significant bits to avoid out of bounds errors
		if len(nodeKey.leastSigBits) < len(currentKey.leastSigBits) {
			currentKey.greatestCommonPrefix(new(int), gcp, nodeKey)
		} else {
			nodeKey.greatestCommonPrefix(new(int), gcp, currentKey)
		}
		// update the current key to the parent key
		currentKey = gcp
		// set the parent node's value, which is the hash of its children
		parentNode.Value = hash
		// set the parent node's key, which is the gcp of its children
		parentNode.Key = currentKey
		// add the parent node to the new tree
		if err := smt.setNode(parentNode); err != nil {
			return false, err
		}
		// set the root of the new tree, as the tree is being reconstructed from
		// the bottom up, the last node in the proof slice will be the root
		if i == proofLen-1 {
			smt.root = parentNode
		}
	}
	// compare the calculated root hash against the provided root hash
	if !bytes.Equal(hash, root) {
		return false, nil
	}
	// calculate the key to traverse the tree
	smt.target = &node{Key: newNodeKey(crypto.Hash(k), smt.keyBitLength)}
	// make sure the target is valid
	if err := smt.validateTarget(); err != nil {
		return false, err
	}
	// navigates the tree downward
	if err := smt.traverse(); err != nil {
		return false, err
	}
	// Verify whether the key exists in the tree and what kind of proof is being validated
	// (membership or non-membership).
	// if the key does not exist in the tree and the proof is for membership or
	// if the key exists in the tree and the proof is for non-membership, return false
	nodeExists := smt.target.Key.equals(smt.gcp)
	if (!nodeExists && validateMembership) || (nodeExists && !validateMembership) {
		return false, nil
	}
	// if the key does not exist in the tree and the proof is for non-membership, return true
	if !nodeExists && !validateMembership {
		return true, nil
	}
	// Verify if the value matches the provided one. This step confirms the
	// proof-of-non-membership, as the intermediate nodes are built using the
	// children's keys and values. A mismatch in values indicates that the Merkle
	// root could not have been derived from this data.
	return bytes.Equal(proof[0].Value, crypto.Hash(v)), nil
}

// NODE KEY CODE BELOW

/*
Understanding Node Keys:
	Node keys are compact representations of key bit sequences, truncated to fit the specified `KeyBitLength`
	KV databases typically store keys as sequences of bytes, but the specified key length for prefix nodes
    might not align with a whole number of bytes. Without the extra byte, there is no way to differentiate
    between keys that share the same initial bytes but differ in bit-length. The extra byte encodes the number
    of leading zero bits in the last byte of the key. This ensures that the database can reconstruct the
    original bit-level representation of the key.

	Example:
	KeyBitLength = 24
	Input:  []byte{255, 255, 255}
	Input Bits:  []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	Output: []byte{255, 255, 255, 0}
	Explanation: 0 leading zero bits in the last byte (255 is 0b11111111)

	KeyBitLength = 9
	Input:  []byte{255, 255}
	Input Bits:  []int{1, 1, 1, 1, 1, 1, 1, 1, 1}
	Output: []byte{255, 1, 0}
	Explanation: The last byte (255) has 0 leading zero bits.
				 However, the key is truncated to 9 bits: 11111111 1,
				 and the last byte (1) has 0 leading zero bits

	KeyBitLength = 10
	Input:  []byte{1, 0}
	Input Bits:  []int{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}
	Output: []byte{1, 0, 1}
	Explanation: The last byte (0) has 1 leading zero bit (00)

*/

// key is the structure used to interpret node keys, it splits the byte array into most significant bytes
// and a final least significant byte represented as a bit array
// Example: []byte{3, 3, 1} = key{mostSigBytes=[]byte{3, 3}, leastSigBits=[]int{0,0,0,0,0,0,0,1}
type key struct {
	// mostSigBytes: is the full most significant bytes of the key (left to right) without the LSB
	mostSigBytes []byte
	// leastSigBits: is the bits of the least significant byte
	leastSigBits []int
	// TODO cache bytes
}

// newNodeKey() creates a brand-new key from a hash value
// NOTE: This is not used to covert key bytes into a key object
func newNodeKey(data []byte, bitCount int) (k *key) {
	// create a new key object
	k = new(key)
	// determine the number of full bytes and remaining bits to extract
	fullBytes, remainingBits := bitCount/8, bitCount%8
	// if no remaining bits, adjust to always include last bits (even if full byte is complete)
	if remainingBits == 0 {
		fullBytes, remainingBits = fullBytes-1, 8
	}
	// extract the most significant full bytes from the data
	k.mostSigBytes = data[:fullBytes]
	// extract the last bits from the next byte, if it exists
	if fullBytes < len(data) {
		lastByte := data[fullBytes]
		for i := 7; i >= 8-remainingBits; i-- {
			k.leastSigBits = append(k.leastSigBits, k.bitAtIndex(lastByte, i))
		}
	}
	return k
}

// greatestCommonPrefix() calculates the greatest common prefix (GCP) between the current key and another key.
// - Starts at a given bit position (`bitPos`).
// - Continues until bits differ or there are no more bits in the `current` key.
// CONTRACT: `current`'s size is always less than or equal to the target (`k`).
func (k *key) greatestCommonPrefix(bitPos *int, gcp *key, current *key) {
	totalBits := current.totalBits()
	// traverse both byte slices bit by bit starting at bit position
	for ; *bitPos < totalBits; *bitPos++ {
		// get the bits for target and current at current bit position
		bit1, bit2 := k.bitAt(*bitPos), current.bitAt(*bitPos)
		if bit1 != bit2 {
			break
		}
		// if the bits match, add to the common prefix
		gcp.addBit(bit1)
	}
}

// bitAt() returns the bit value <0 or 1> at a 0 indexed position left to right (MSB)
// ex 1: [0,1,1,1]: bitPos=0 returns 0 and bitPos=1 returns 1
// ex 2: [1,0,0,0,0,0,0,0], [1,0,0]: bitPos=8 returns 1 and bitPos=9 returns 0
func (k *key) bitAt(bitPos int) int {
	// calculate the byte index
	byteIndex := bitPos / 8
	// if within most significant bytes
	if byteIndex < len(k.mostSigBytes) {
		// calculate the new bit index using MSB logic
		bitIndex := 7 - (bitPos % 8)
		// get the byte at byte index
		byt := k.mostSigBytes[byteIndex]
		// use bitwise to retrieve the bit value
		return k.bitAtIndex(byt, bitIndex)
	}
	// if within the least significant bits
	return k.leastSigBits[bitPos%8]
}

// addBit() adds a bit to the key
func (k *key) addBit(bit int) {
	// if least significant bits is full
	if len(k.leastSigBits) == 8 {
		// convert it to a byte and add it to the most sig bytes
		k.mostSigBytes = append(k.mostSigBytes, k.bitsToByte(k.leastSigBits))
		// reset the least sig bits
		k.leastSigBits = nil
	}
	// prepend the bit to the least sig bits
	k.leastSigBits = append(k.leastSigBits, bit)
}

// bytes() encodes a key object to bytes preserving the leading zero
// information needed for prefix keys ex. (0010, 001, 00)
func (k *key) bytes() []byte {
	var leadingZeroes int
	// iterate through all the bits
	for _, bit := range k.leastSigBits {
		// exit loop if bit is 1
		if bit == 1 {
			break
		}
		// increment leading zeroes if bit is 0
		leadingZeroes++
	}
	// if all bits are zero, decrement the leadingZeroes count
	// because the lastByte will inherently count for 1 zero
	if leadingZeroes == len(k.leastSigBits) {
		leadingZeroes--
	}
	// convert the last bits back to byte
	lastByte := k.bitsToByte(k.leastSigBits)
	// encoding = most_significant_bytes + least_significant_byte + leading_zeroes_in_LSB
	return append(append(k.mostSigBytes, lastByte), byte(leadingZeroes))
}

// fromBytes() creates a new key object from existing encoded key bytes
func (k *key) fromBytes(data []byte) *key {
	keyLength := len(data)
	// mostSigBytes: full bit bytes going left to right excluding the last byte
	k.mostSigBytes = data[:keyLength-2]
	// lastByte: last value byte of the key
	lastByte := data[keyLength-2]
	// leadingZeroes: the actual last byte contains the number of leading zeroes (if any)
	leadingZeroes := data[keyLength-1]
	// convert the final byte to bits
	k.leastSigBits = k.byteToBits(lastByte, int(leadingZeroes))

	return k
}

// bitsToBytes() converts an array of bits to a byte
// example: []int{1 0 1 1} --> Byte: byte(0b00001011)
func (k *key) bitsToByte(bits []int) (b byte) {
	maxIdx := len(bits) - 1
	// iterate through the bits array from LSB to MSB
	for i, bit := range bits {
		// shift the bit to its correct position and set it in the byte using a bitwise OR operation.
		b |= byte(bit) << (maxIdx - i)
	}
	return
}

// byteToBits() converts a byte to a bit array given some leading zeroes
// Example: b=byte(1), leadingZeroes=3 --> []int{0, 0, 0, 1}
func (k *key) byteToBits(b byte, leadingZeroes int) (bits []int) {
	// handle leading zero count
	bits = make([]int, leadingZeroes)
	// if b == 0, it inherently has 1 zero in the bit array
	// it may have more if leading zeroes is != 0 ex. 000
	if b == 0 {
		leadingZeroes++
		bits = append(bits, 0)
	}
	// convert the final byte to bits
	for i := 7 - leadingZeroes; i >= 0; i-- {
		// get bit at index
		bit := k.bitAtIndex(b, i)
		// ensure no other leading zeroes are added
		// example: byte(1) and leadingZeroes=2 = []int{0,0,1}
		if bit == 0 && len(bits) == leadingZeroes {
			continue // ignore any other leading zeroes
		}
		// add to the bit array
		bits = append(bits, int(bit))
	}
	return
}

// bitAtIndex() returns a bit at an index (0 indexed and left to right) within a byte
func (k *key) bitAtIndex(b byte, index int) int { return int(b>>index) & 1 }

// totalBits() returns the total number of bits in a key
func (k *key) totalBits() int { return len(k.mostSigBytes)*8 + len(k.leastSigBits) }

// equals() returns true if two key objects are equivalent
func (k *key) equals(k2 *key) bool {
	return bytes.Equal(k.mostSigBytes, k2.mostSigBytes) &&
		slices.Equal(k.leastSigBits, k2.leastSigBits)
}

// NODE CODE BELOW

// newNode() is a constructor for the node object
func newNode() (n *node) {
	n = new(node)
	n.Key = new(key)
	return
}

// bytes() returns the marshalled node
func (x *node) bytes() ([]byte, lib.ErrorI) {
	// convert the object into bytes
	// NOTE: the `key` will not be marshalled as
	// it's excluded from the Node protobuf structure
	return lib.Marshal(x)
}

// setChildren() sets the children of a node in its structure
func (x *node) setChildren(leftKey, rightKey []byte) {
	x.LeftChildKey, x.RightChildKey = leftKey, rightKey
}

// getOtherChild() returns the sibling for the child key passed and which child it is
func (x *node) getOtherChild(childKey []byte) ([]byte, byte) {
	switch {
	case bytes.Equal(x.LeftChildKey, childKey):
		return x.RightChildKey, rightChild
	case bytes.Equal(x.RightChildKey, childKey):
		return x.LeftChildKey, leftChild
	}
	panic("no child node was a match for getOtherChild")
}

// replaceChild() replaces the child reference with a new key
func (x *node) replaceChild(oldKey, newKey []byte) {
	switch {
	case bytes.Equal(x.LeftChildKey, oldKey):
		x.LeftChildKey = newKey
		return
	case bytes.Equal(x.RightChildKey, oldKey):
		x.RightChildKey = newKey
		return
	}
	panic("no child node was replaced")
}

// copy() returns a deep copy of the node
func (x *node) copy() *node {
	return &node{
		Key: &key{
			mostSigBytes: slices.Clone(x.Key.mostSigBytes),
			leastSigBits: slices.Clone(x.Key.leastSigBits),
		},
		Node: lib.Node{
			Value:         slices.Clone(x.Value),
			LeftChildKey:  slices.Clone(x.LeftChildKey),
			RightChildKey: slices.Clone(x.RightChildKey),
		},
	}
}

// NODE LIST CODE BELOW

// NodeList defines a list of nodes, used for traversal
type NodeList struct {
	Nodes []*node
}

// Parent() returns the parent of the last node traversed (current)
func (n *NodeList) Parent() *node { return n.Nodes[len(n.Nodes)-1] }

// GrandParent() returns the grandparent of the last node traversed (current)
func (n *NodeList) GrandParent() *node { return n.Nodes[len(n.Nodes)-2] }

// Pop() removes the node from the list
func (n *NodeList) Pop() { n.Nodes = n.Nodes[:len(n.Nodes)-1] }

// RootKey() value is arbitrary, but it happens to be right in the middle of Min and Max Hash for abstract cleanliness
var (
	RootKey = []byte{
		0x7F, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
	}
)
