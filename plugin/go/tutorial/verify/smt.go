package verify

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/bits"
)

/*
smt.go is a self-contained, dependency-free port of Canopy's optimized Sparse Merkle Tree proof
verification (store/smt.go). It reconstructs the tree from a proof and checks membership or
non-membership against a trusted root, using the exact same key encoding and hashing (sha256) as the
node, so proofs produced by the node verify here without importing or running it.

Node leaf keys are sha256(rawKey) and leaf values are sha256(value); internal node values are the
hash of (leftKey || leftValue || rightKey || rightValue).
*/

const (
	maxKeyBitLength = 160 // 20-byte (sha256-truncated) leaf keys
	leftChild       = 0
	rightChild      = 1
)

var (
	errInvalidProof = errors.New("invalid merkle proof")
	errReservedKey  = errors.New("reserved key")
	errInvalidTree  = errors.New("invalid merkle tree")
)

// rootKeyBytes is the arbitrary root key (midpoint between min and max), matching store.RootKey.
var rootKeyBytes = []byte{
	0x7F, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
}

// hash() is the global hashing algorithm (sha256), matching lib/crypto.Hash.
func hash(msg []byte) []byte {
	h := sha256.Sum256(msg)
	return h[:]
}

// concat() returns a freshly-allocated concatenation of the parts (avoids append aliasing).
func concat(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// ============================ node ============================

// node mirrors the fields of store's node/lib.Node used during verification.
type node struct {
	Key           *key
	Value         []byte
	LeftChildKey  []byte
	RightChildKey []byte
	Bitmask       int32
}

func newNode() *node { return &node{Key: new(key)} }

func (x *node) copy() *node { c := *x; return &c }

// ============================ smt (verification-only) ============================

type nodeList struct{ nodes []*node }

type smt struct {
	store        map[string]*node
	keyBitLength int
	root         *node
	minKey       *key
	maxKey       *key
	// traversal state
	current   *node
	target    *node
	traversed *nodeList
	gcp       *key
	bitPos    int
	pathBit   int
}

// newSMT() creates a verification SMT over an in-memory node map, initializing an empty tree with
// the reserved min/max border nodes (mirrors store.NewSMT).
func newSMT(rootKey []byte, keyBitLen int, store map[string]*node) *smt {
	s := &smt{
		store:        store,
		keyBitLength: keyBitLen,
		minKey:       newNodeKey(bytes.Repeat([]byte{0}, 20), keyBitLen),
		maxKey:       newNodeKey(bytes.Repeat([]byte{255}, 20), keyBitLen),
	}
	rKey := newNodeKey(bytes.Clone(rootKey), keyBitLen)
	s.root = s.getNode(rKey.bytes())
	if s.root.LeftChildKey == nil {
		s.initializeTree(rKey)
	}
	s.reset()
	return s
}

// initializeTree() ensures the tree always has a root with two border children.
func (s *smt) initializeTree(rootKey *key) {
	minNode := &node{Key: newNodeKey(bytes.Repeat([]byte{0}, 20), s.keyBitLength), Value: bytes.Repeat([]byte{0}, 20)}
	maxNode := &node{Key: newNodeKey(bytes.Repeat([]byte{255}, 20), s.keyBitLength), Value: bytes.Repeat([]byte{255}, 20)}
	s.setNode(minNode)
	s.setNode(maxNode)
	s.root = &node{
		Key:           rootKey,
		LeftChildKey:  minNode.Key.bytes(),
		RightChildKey: maxNode.Key.bytes(),
	}
	s.updateParentValue(s.root)
	s.setNode(s.root)
}

// updateParentValue() recomputes a parent's value from its children (left/right key+value).
func (s *smt) updateParentValue(parent *node) {
	lChild := s.getNode(parent.LeftChildKey)
	rChild := s.getNode(parent.RightChildKey)
	input := make([]byte, lChild.Key.size()+len(lChild.Value)+rChild.Key.size()+len(rChild.Value))
	offset := 0
	n := copy(input[offset:], lChild.Key.bytes())
	offset += n
	n = copy(input[offset:], lChild.Value)
	offset += n
	n = copy(input[offset:], rChild.Key.bytes())
	offset += n
	copy(input[offset:], rChild.Value)
	parent.Value = hash(input)
	if bytes.Equal(parent.Key.bytes(), s.root.Key.bytes()) {
		s.root = parent.copy()
	}
}

func (s *smt) getNode(k []byte) *node {
	if n, ok := s.store[string(k)]; ok && n != nil {
		return n
	}
	return newNode()
}

func (s *smt) setNode(n *node) { s.store[string(n.Key.bytes())] = n }

func (s *smt) reset() {
	s.current = s.root.copy()
	s.traversed = &nodeList{nodes: make([]*node, 0)}
	s.resetGCP()
}

func (s *smt) resetGCP() {
	s.gcp = &key{}
	s.pathBit, s.bitPos = 0, 0
}

// traverse() navigates the tree downward to locate the target or its closest position.
func (s *smt) traverse() error {
	for {
		var currentKey []byte
		s.traversed.nodes = append(s.traversed.nodes, s.current.copy())
		switch s.pathBit = s.target.Key.bitAt(s.bitPos); s.pathBit {
		case leftChild:
			currentKey = s.current.LeftChildKey
		case rightChild:
			currentKey = s.current.RightChildKey
		}
		s.current = s.getNode(currentKey)
		if s.current == nil {
			return errInvalidTree
		}
		s.current.Key.fromBytes(currentKey)
		s.target.Key.greatestCommonPrefix(&s.bitPos, s.gcp, s.current.Key)
		if !s.current.Key.equals(s.gcp) || s.target.Key.equals(s.gcp) {
			return nil
		}
	}
}

// validateTarget() rejects reserved keys (root, min, max).
func (s *smt) validateTarget(n *node) error {
	if bytes.Equal(s.root.Key.bytes(), n.Key.bytes()) {
		return errReservedKey
	}
	if bytes.Equal(s.minKey.bytes(), n.Key.bytes()) {
		return errReservedKey
	}
	if bytes.Equal(s.maxKey.bytes(), n.Key.bytes()) {
		return errReservedKey
	}
	return nil
}

// verifyProof() reconstructs the root hash from the proof and compares it against the provided root,
// then confirms membership (value matches) or non-membership (target absent). Faithful port of
// store.SMT.VerifyProof.
func verifyProof(k, v []byte, validateMembership bool, root []byte, proof []ProofNode, keyBitLength int) (ok bool, err error) {
	// The proof is fully attacker-controlled: a malicious server can send malformed keys/bitmasks
	// that drive the faithful SMT port into out-of-range slice indexing. Treat any such panic as a
	// failed proof rather than crashing the client (defense against a client-side DoS).
	defer func() {
		if r := recover(); r != nil {
			ok, err = false, errInvalidProof
		}
	}()
	proofLen := len(proof)
	if proofLen < 2 {
		return false, errInvalidProof
	}
	h := proof[0].Value
	currentKey := new(key).fromBytes(proof[0].Key)
	memStore := make(map[string]*node)
	s := newSMT(rootKeyBytes, keyBitLength, memStore)
	// set the node being proven (leaf or potential location)
	s.setNode(&node{Value: proof[0].Value, Key: currentKey})
	// reconstruct the tree bottom-up using the proof siblings
	for i := 1; i < proofLen; i++ {
		var parentNode *node
		if proof[i].Bitmask == leftChild {
			h = hash(concat(proof[i].Key, proof[i].Value, currentKey.bytes(), h))
			parentNode = &node{LeftChildKey: proof[i].Key, RightChildKey: currentKey.bytes()}
		} else {
			h = hash(concat(currentKey.bytes(), h, proof[i].Key, proof[i].Value))
			parentNode = &node{LeftChildKey: currentKey.bytes(), RightChildKey: proof[i].Key}
		}
		nodeKey := new(key).fromBytes(proof[i].Key)
		gcp := new(key)
		// NOTE: faithfully mirrors store.VerifyProof (the first comparison is a no-op there too).
		if currentKey.totalBits() < currentKey.totalBits() {
			currentKey.greatestCommonPrefix(new(int), gcp, nodeKey)
		} else {
			nodeKey.greatestCommonPrefix(new(int), gcp, currentKey)
		}
		currentKey = gcp
		parentNode.Value = h
		parentNode.Key = currentKey
		s.setNode(parentNode)
		if i == proofLen-1 {
			s.root = parentNode
		}
	}
	// compare the reconstructed root against the provided (trusted) root
	if !bytes.Equal(h, root) {
		return false, nil
	}
	// traverse the reconstructed tree to confirm presence/absence of the target
	s.target = &node{Key: newNodeKey(hash(k), s.keyBitLength)}
	if err := s.validateTarget(s.target); err != nil {
		return false, err
	}
	s.reset()
	if err := s.traverse(); err != nil {
		return false, err
	}
	nodeExists := s.target.Key.equals(s.gcp)
	if (!nodeExists && validateMembership) || (nodeExists && !validateMembership) {
		return false, nil
	}
	if !nodeExists && !validateMembership {
		return true, nil
	}
	// membership: the proven value must hash to the leaf value
	return bytes.Equal(proof[0].Value, hash(v)), nil
}

// ============================ key ============================
//
// key is a []byte representation of a left-to-right bit-string where the final byte stores the
// number of left-padding zeroes in the preceding data byte. Faithful port of store's key type.

type key struct {
	key      []byte
	bitCount int
	length   int
}

func newNodeKey(data []byte, bitCount int) (k *key) {
	numBytes := (bitCount + 7) / 8
	k = &key{key: make([]byte, numBytes)}
	copy(k.key, data)
	lastByteBits := bitCount % 8
	if lastByteBits == 0 && bitCount > 0 {
		lastByteBits = 8
	}
	k.key[numBytes-1] >>= 8 - lastByteBits
	leftPadding := bits.LeadingZeros8(k.key[numBytes-1]) - (8 - lastByteBits)
	if k.key[numBytes-1] == 0 {
		leftPadding--
	}
	k.key = append(k.key, byte(leftPadding))
	k.bitCount = bitCount
	return k
}

func (k *key) greatestCommonPrefix(bitPos *int, gcp *key, current *key) {
	totalBits := current.totalBits()
	for ; *bitPos < totalBits; *bitPos++ {
		bit1, bit2 := k.bitAt(*bitPos), current.bitAt(*bitPos)
		if bit1 != bit2 {
			break
		}
		gcp.addBit(bit1)
	}
}

func (k *key) bitAt(bitPos int) int {
	byteIndex := bitPos / 8
	byt := k.key[byteIndex]
	size, pos := k.size(), bitPos%8
	if byteIndex != size-2 {
		bitIndex := 7 - pos
		return k.bitAtIndex(byt, bitIndex)
	}
	leftPadding := int(k.key[size-1])
	if (leftPadding == 0 && byt == 0) || pos < leftPadding {
		return 0
	}
	lastByteBitsCount := ((k.totalBits() - 1) % 8) + 1
	shift := lastByteBitsCount - pos - 1
	return int((byt >> shift) & 1)
}

func (k *key) addBit(bit int) {
	bitPos, size := k.totalBits()%8, k.size()
	if bitPos == 0 && k.bitCount != 0 {
		bitPos = 8
	}
	lastByte, leftPadding := k.key[size-2], int(k.key[size-1])
	if k.bitCount != 0 && lastByte == 0 {
		leftPadding++
	}
	if bitPos == 8 {
		k.key = append(k.key, 0)
		lastByte, bitPos, leftPadding, k.length, size = 0, 0, 0, k.length+1, size+1
	}
	k.bitCount++
	k.key[size-2], k.key[size-1] = (lastByte<<1)|byte(bit), byte(leftPadding)
}

func (k *key) bytes() []byte { return k.key }

func (k *key) fromBytes(data []byte) *key {
	k.key = data
	return k
}

func (k *key) totalBits() int {
	if k.bitCount == 0 {
		size := k.size()
		if size == 0 {
			k.key, k.length = []byte{0, 0}, 2
			return 0
		}
		leadingZeroes := int(k.key[size-1])
		dataBytes := k.key[:size-1]
		fullBits := (size - 2) * 8
		bitLen := bits.Len8(dataBytes[size-2])
		if bitLen == 0 {
			bitLen = 1
		}
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

func (k *key) bitAtIndex(b byte, index int) int { return int(b>>index) & 1 }

func (k *key) equals(k2 *key) bool { return bytes.Equal(k.key, k2.key) }

func (k *key) cmp(k2 *key) int {
	var startBit int
	k2TotalBits := k2.totalBits()
	fullBytes := k2TotalBits / 8
	if fullBytes > 0 {
		startBit = fullBytes * 8
		fullBytesCmp := bytes.Compare(k.key[:fullBytes], k2.key[:fullBytes])
		if fullBytesCmp != 0 {
			return fullBytesCmp
		}
	}
	for i := startBit; i < k2TotalBits; i++ {
		b1, b2 := k.bitAt(i), k2.bitAt(i)
		switch {
		case b1 < b2:
			return -1
		case b1 > b2:
			return 1
		}
	}
	return 0
}
