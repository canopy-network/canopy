package store

import (
	"bytes"
	"math/bits"
	"sort"

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
const MaxCacheSize = 1_000_000

type SMT struct {
	// store: an abstraction of the database where the tree is being stored
	store lib.RWStoreI
	// root: the root node
	root *node
	// keyBitLength: the depth of the tree, once set it cannot be changed for a protocol
	keyBitLength int
	// nodeCache: an efficient in-memory cache to avoid marshalling and unmarshalling recent nodes
	nodeCache map[string]*node
	// operations: a list of deferred operations to execute
	operations []*node
	// unsorted_ops: a list of operations that aren't yet sorted
	unsortedOps map[string]*node
	// OpData: data for each operation
	OpData
}

// node wraps protobuf Node with a key
type node struct {
	// Key: the structure that is used to interpret node keys (bytes, fromBytes, etc.)
	Key *key
	// Node: is the structure persisted on disk under the above key bytes
	lib.Node
	// delete: indicates if the deferred operation for the node is 'set or delete'
	delete bool
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
		nodeCache:    make(map[string]*node, MaxCacheSize),
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
	// initialize the operations list
	smt.unsortedOps = make(map[string]*node)
	// prepare for traversal
	smt.reset()
	return
}

// Root() returns the root value of the smt
func (s *SMT) Root() []byte { return bytes.Clone(s.root.Value) }

// Set: insert or update a target
func (s *SMT) Set(k, v []byte) (err lib.ErrorI) {
	// calculate the key and value to upsert
	n := &node{Key: newNodeKey(crypto.Hash(k), s.keyBitLength), Node: lib.Node{Value: crypto.Hash(v)}}
	// check to make sure the target is valid
	if err = s.validateTarget(n); err != nil {
		return err
	}
	// set the node in the deferred list
	s.addOperation(n)
	// exit
	return
}

// set() executes the 'set' logic after traversal
func (s *SMT) set() lib.ErrorI {
	// if current != target key then it is an insert not an update
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
	return s.setNode(s.target)
}

// Delete: removes a target node if exists in the tree
func (s *SMT) Delete(k []byte) (err lib.ErrorI) {
	// calculate the key and value to upsert
	n := &node{Key: newNodeKey(crypto.Hash(k), s.keyBitLength), delete: true}
	// check to make sure the target is valid
	if err = s.validateTarget(n); err != nil {
		return
	}
	// set the node in the deferred list
	s.addOperation(n)
	// exit
	return
}

// delete() executes the 'delete' logic after traversal
func (s *SMT) delete() lib.ErrorI {
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
	return s.delNode(targetBytes)
}

// Commit(): executes the deferred operations in order (left-to-right),
// minimizing the amount of traversals, IOPS, and hash operations
func (s *SMT) Commit() (err lib.ErrorI) {
	s.sortOperations()
	for {
		// if no operations and at root - exit
		if len(s.operations) == 0 && len(s.traversed.Nodes) == 0 {
			return
		}
		// pop head into target
		s.target, s.operations = s.operations[0], s.operations[1:]
		// reset path variables
		s.resetGCP()
		// if not at root
		if len(s.traversed.Nodes) != 0 {
			// update the greatest common prefix and the bit position based on the new current key
			s.target.Key.greatestCommonPrefix(&s.bitPos, s.gcp, s.current.Key)
		}
		// traverse to target
		if err = s.traverse(); err != nil {
			return
		}
		// execute operation
		if !s.target.delete {
			if err = s.set(); err != nil {
				return
			}
		} else {
			if err = s.delete(); err != nil {
				return
			}
		}
		// rehash
		if err = s.rehash(); err != nil {
			return
		}
	}
}

// traverse: navigates the tree downward to locate the target or its closest position
func (s *SMT) traverse() (err lib.ErrorI) {
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
		// defensive nil check
		if s.current == nil {
			return ErrInvalidMerkleTree()
		}
		// load the bytes into the key
		s.current.Key.fromBytes(currentKey)
		// update the greatest common prefix and the bit position based on the new current key
		s.target.Key.greatestCommonPrefix(&s.bitPos, s.gcp, s.current.Key)
		// exit conditions, current != gcp || current == target
		if !s.current.Key.equals(s.gcp) || s.target.Key.equals(s.gcp) {
			return // exit loop
		}
	}
}

// rehash() recalculate hashes from the current node upwards until
// a) the next operation key is LTE the right sibling's key, after truncating it to the right sibling's key length
// b) traversed list is empty
func (s *SMT) rehash() (err lib.ErrorI) {
	// iterate the traversed list from end to start
	for {
		if len(s.traversed.Nodes) == 0 {
			return
		}
		// select the parent
		parent := s.traversed.Nodes[len(s.traversed.Nodes)-1]
		// set current = parent
		s.current = parent
		// remove the parent from the traverse list
		s.traversed.Pop()
		// exit if next operation is LTE the right sibling
		if len(s.operations) != 0 && s.operations[0].Key.cmp(&key{key: parent.RightChildKey}) <= 0 {
			return
		}
		// calculate its new value
		if err = s.updateParentValue(parent); err != nil {
			return
		}
		// set node in the database
		if err = s.setNode(parent); err != nil {
			return
		}
	}
}

// addOperation() adds a deferred operation to the sorted list
func (s *SMT) addOperation(n *node) { s.unsortedOps[string(n.Key.bytes())] = n }

// sortOperations() sorts the operations
func (s *SMT) sortOperations() {
	s.operations = make([]*node, len(s.unsortedOps))
	// insert all unsorted operations into the slice
	i := 0
	for _, n := range s.unsortedOps {
		s.operations[i] = n
		i++
	}
	// sort the operations
	sort.Slice(s.operations, func(i, j int) bool {
		return s.operations[i].Key.cmp(s.operations[j].Key) < 0
	})
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
	if err := s.updateParentValue(s.root); err != nil {
		panic(err)
	}
	// set the root in store
	if err := s.setNode(s.root); err != nil {
		panic(err)
	}
}

// updateParentValue() updates the value of parent based on its children
func (s *SMT) updateParentValue(parent *node) (err lib.ErrorI) {
	var rightChild, leftChild *node
	// get the left child
	if leftChild, err = s.getNode(parent.LeftChildKey); err != nil {
		return
	}
	// get the left child
	if rightChild, err = s.getNode(parent.RightChildKey); err != nil {
		return
	}
	// create a buffer for the input
	input := make([]byte, leftChild.Key.size()+len(leftChild.Value)+rightChild.Key.size()+len(rightChild.Value))
	offset := 0
	// copy leftChild key
	n := copy(input[offset:], leftChild.Key.bytes())
	offset += n
	// copy leftChild value
	n = copy(input[offset:], leftChild.Value)
	offset += n
	// copy rightChild key
	n = copy(input[offset:], rightChild.Key.bytes())
	offset += n
	// copy rightChild value
	copy(input[offset:], rightChild.Value)
	// concatenate the left and right children values; update the parents value
	parent.Value = crypto.Hash(input)
	// save the updated root value to the structure
	if bytes.Equal(parent.Key.bytes(), s.root.Key.bytes()) {
		s.root = parent.copy()
	}
	return
}

// reset() resets data for each operation
func (s *SMT) reset() {
	s.current, s.traversed = s.root.copy(), &NodeList{Nodes: make([]*node, 0)}
	s.resetGCP()
}

// resetGCP() resets the greatest common prefix and path variables
func (s *SMT) resetGCP() {
	s.gcp = &key{}
	s.pathBit, s.bitPos = 0, 0
}

// setNode() set a node object in a key value database
func (s *SMT) setNode(n *node) lib.ErrorI {
	// check cache max size
	if len(s.nodeCache) >= MaxCacheSize {
		s.nodeCache = make(map[string]*node, MaxCacheSize)
	}
	// set in cache
	s.nodeCache[string(n.Key.bytes())] = n
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
	delete(s.nodeCache, string(key))
	return s.store.Delete(key)
}

// getNode() retrieves a node object from the database
func (s *SMT) getNode(key []byte) (n *node, err lib.ErrorI) {
	// check cache
	n, found := s.nodeCache[string(key)]
	if found {
		return n, nil
	}
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
func (s *SMT) validateTarget(n *node) lib.ErrorI {
	if bytes.Equal(s.root.Key.bytes(), n.Key.bytes()) {
		return ErrReserveKeyWrite("root")
	}
	if bytes.Equal(newNodeKey(bytes.Repeat([]byte{0}, 20), s.keyBitLength).bytes(), n.Key.bytes()) {
		return ErrReserveKeyWrite("minimum")
	}
	if bytes.Equal(newNodeKey(bytes.Repeat([]byte{255}, 20), s.keyBitLength).bytes(), n.Key.bytes()) {
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
	if err := s.validateTarget(s.target); err != nil {
		return nil, err
	}
	// make the slice to store the leaf nodes and the intermediate sibling nodes
	proof := make([]*lib.Node, 0)
	// reset the traversal variables
	s.reset()
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
		if currentKey.totalBits() < currentKey.totalBits() {
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
	if err := smt.validateTarget(smt.target); err != nil {
		return false, err
	}
	// reset the traversal variables
	smt.reset()
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
- Node keys are []byte representation of 'left-to-right' bit-strings where the last byte is metadata
that stores the number of left padding zeroes in the final data byte.

The padding byte is needed in order to support varying length bit-strings:

Examples:
  - []byte{255, 0}       = 11111111           where bit-length = 8
  - []byte{0, 0b1011, 2} = 00000000 001011    where bit-length = 14
  - []byte{0, 0b1, 4}    = 00000000 00001     where bit-length = 13
  - []byte{0, 1}         = 00                 where bit-length = 2
  - []byte{0, 0}         = 0                  where bit-length = 1

An alternative structure to this design would've been storing the bit-length in the final byte
however storing the bit length might require two bytes for keys > length of 255
*/

// key is the structure used to cache data about node keys for optimal performance
type key struct {
	key      []byte // the actual node key bytes
	bitCount int    // cached bit count
	length   int    // cached num of bytes in the node key
}

// newNodeKey() given data and a bitCount, this function treats bytes like a continuous bit string
// a) it truncates the bits beyond bitcount
// b) it stores the left padding of the final byte in an appended meta byte at the end
//
// Examples:
//
//   - data = []byte{0b00001000, 0b00100010}, bitCount = 11
//     => []byte{0b00001000, 0b00000001, 2}
//     Final 3 bits are meaningful ('001'), compacted to 0b00000001 with 2 leading zeros.
//
//   - data = []byte{0b00001000}, bitCount = 4
//     => []byte{0b00000000, 3}
//     Final 4 bits are all 0s, compacted to 0b00000000 with 3 leading zeros.
//
//   - data = []byte{0b11110000, 0b10101001}, bitCount = 13
//     => []byte{0b11110000, 0b00010101, 0}
//     Final 5 bits are '10101', compacted to 0b00010101 with 0 leading zeros.
func newNodeKey(data []byte, bitCount int) (k *key) {
	// count the number of bytes
	numBytes := (bitCount + 7) / 8
	// create a new key object
	k = &key{key: make([]byte, numBytes)}
	// copy the data into that key object without extra bytes
	copy(k.key, data)
	// calculate the bits in the final byte
	lastByteBits := bitCount % 8
	if lastByteBits == 0 && bitCount > 0 {
		lastByteBits = 8
	}
	// zero out junk bits
	k.key[numBytes-1] >>= 8 - lastByteBits
	// calculate left padding (number of leading zeroes)
	leftPadding := bits.LeadingZeros8(k.key[numBytes-1]) - (8 - lastByteBits)
	if k.key[numBytes-1] == 0 {
		leftPadding--
	}
	// append meta byte
	k.key = append(k.key, byte(leftPadding))
	// set bit count
	k.bitCount = bitCount
	// exit with key
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
	// get the byte at byte index
	byt := k.key[byteIndex]
	// get the length of the key
	size, pos := k.size(), bitPos%8
	// if in the bit is NOT in the last data byte
	if byteIndex != size-2 {
		// calculate the new bit index using MSB logic
		bitIndex := 7 - pos
		// use bitwise to retrieve the bit value
		return k.bitAtIndex(byt, bitIndex)
	}
	// if the byte is fully zero or the bit is within the left-padding region, return 0
	leftPadding := int(k.key[size-1])
	if (leftPadding == 0 && byt == 0) || pos < leftPadding {
		return 0
	}
	// compute how many bits are meaningful in the final byte (1-based count)
	lastByteBitsCount := ((k.totalBits() - 1) % 8) + 1
	// calculate how far to right-shift to align the target bit to the least significant bit
	shift := lastByteBitsCount - pos - 1
	// shift the byte and mask to isolate and return the desired bit (0 or 1)
	return int((byt >> shift) & 1)
}

// addBit() appends a single bit (0 or 1) to the key and returns a new key
func (k *key) addBit(bit int) {
	// total byte length, including metaByte
	bitPos, size := k.totalBits()%8, k.size()
	if bitPos == 0 && k.bitCount != 0 {
		bitPos = 8
	}
	// get the last data byte, get the zero bit count
	lastByte, leftPadding := k.key[size-2], int(k.key[size-1])
	// inc leading zeroes if bit is zero and the last byte is all zeroes
	if k.bitCount != 0 && lastByte == 0 {
		leftPadding++
	}
	// if the current lastByte is full, append a new byte
	if bitPos == 8 {
		// add byte to the end
		k.key = append(k.key, 0)
		// reset last byte, bit position and left padding, increment size
		lastByte, bitPos, leftPadding, k.length, size = 0, 0, 0, k.length+1, size+1
	}
	// increment the bit count
	k.bitCount++
	// append the new bit to the end of the last byte and update key
	k.key[size-2], k.key[size-1] = (lastByte<<1)|byte(bit), byte(leftPadding)
}

// bytes() encodes a key object to bytes
func (k *key) bytes() []byte { return k.key }

// fromBytes() creates a new key object from existing encoded key bytes
func (k *key) fromBytes(data []byte) *key {
	k.key = data
	return k
}

// totalBits() returns the number of meaningful bits in the key
func (k *key) totalBits() int {
	if k.bitCount == 0 {
		// total byte length, including metaByte
		size := k.size()
		// initialize if necessary
		if size == 0 {
			k.key, k.length = []byte{0, 0}, 2
			return 0
		}
		// metaByte = number of leading 0s which is encoded in the *final* byte
		leadingZeroes := int(k.key[size-1])
		// slice without the metaByte
		dataBytes := k.key[:size-1]
		// bits from all full data bytes except the last
		fullBits := (size - 2) * 8
		// number of significant bits in the last byte
		bitLen := bits.Len8(dataBytes[size-2])
		// ensure only '0' still counts as 1 bit
		if bitLen == 0 {
			bitLen = 1
		}
		// finally set the bit count
		k.bitCount = fullBits + leadingZeroes + bitLen
	}
	return k.bitCount
}

func (k *key) size() int {
	if k.length == 0 {
		k.length = len(k.key)
	}
	return k.length
}

// bitAtIndex() returns a bit at an index (0 indexed and left to right) within a byte
func (k *key) bitAtIndex(b byte, index int) int { return int(b>>index) & 1 }

// equals() returns true if two key objects are equivalent
func (k *key) equals(k2 *key) bool { return bytes.Equal(k.key, k2.key) }

// cmp() compares two node keys, returning -1 for less, 0 for equal, and 1 for greater than
// - contract: `k2`'s size is always less than or equal to (`k`)
func (k *key) cmp(k2 *key) int {
	var startBit int
	// get the totalBits of k2
	k2TotalBits := k2.totalBits()
	// calculate the number of full bytes
	fullBytes := k2TotalBits / 8
	// if k2 has full bytes
	if fullBytes > 0 {
		// calculate start bit
		startBit = fullBytes * 8
		// compare the full bytes
		fullBytesCmp := bytes.Compare(k.key[:fullBytes], k2.key[:fullBytes])
		// if full bytes aren't equal
		if fullBytesCmp != 0 {
			return fullBytesCmp
		}
	}
	// do bitwise comparison for the remaining bits
	for i := startBit; i < k2TotalBits; i++ {
		// compare bit by bit in the final data byte
		b1, b2 := k.bitAt(i), k2.bitAt(i)
		switch {
		// less
		case b1 < b2:
			return -1
		// greater
		case b1 > b2:
			return 1
		}
	}
	// exit
	return 0
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

// copy() returns a shallow copy of the node
func (x *node) copy() *node { return &(*x) }

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
