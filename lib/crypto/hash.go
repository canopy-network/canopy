package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

const (
	HashSize = sha256.Size
)

var (
	MaxHash = bytes.Repeat([]byte{0xFF}, HashSize)
)

/*
	Hash is a function that takes an input message and returns a fixed-size string of bytes that is unique to the input
    to produce a short, fixed-length representation of the data, which can be used for various applications like data
    integrity checks
*/

// Hasher() returns the global hashing algorithm used
func Hasher() hash.Hash { return sha256.New() }

// Hash() executes the global hashing algorithm on input bytes
func Hash(msg []byte) []byte {
	h := sha256.Sum256(msg)
	return h[:]
}

// ShortHash() executes the global hashing algorithm on input bytes
// and truncates the output to 20 bytes
func ShortHash(msg []byte) []byte {
	h := sha256.Sum256(msg)
	return h[:20]
}

// ShortHashString() returns the hex byte version of a short hash
func ShortHashString(msg []byte) string { return hex.EncodeToString(ShortHash(msg)) }

// HashString() returns the hex byte version of a hash
func HashString(msg []byte) string { return hex.EncodeToString(Hash(msg)) }

// MerkleTree creates a merkle tree from a slice of bytes. A
// linear slice was chosen since it uses about half as much memory as a tree
// example: items = {a, b, c, d} -> store = {H(a), H(b), H(c), H(d), H(H(a),H(b)), H(H(c),H(d)), H(H(H(a),H(b)),H(H(c),H(d))) }
func MerkleTree(items [][]byte) (root []byte, store [][]byte, err error) {
	if len(items) == 0 {
		return []byte{}, [][]byte{}, nil
	}
	// calculate how many entries are required to hold the binary merkle
	// tree as a linear array and create a slice of that size.
	offset := nextPowerOfTwo(len(items))
	// calculate the length of the tree
	size := offset*2 - 1
	// initialize the store to populate the tree with
	store = make([][]byte, size)
	// create the base hashes and populate the slice with them.
	for i, item := range items {
		store[i] = Hash(item)
	}
	// offset index = after the last transaction and adjusted to the next power of two.
	for i := 0; i < size-1; i += 2 {
		switch {
		// normal case, parent = hash(Concat(left, right))
		default:
			store[offset] = Hash(concat(store[i], store[i+1]))

		// no left or right child, so the parent is going to be nil
		case store[i] == nil:
			store[offset] = nil

		// no right child, parent = hash(Concat(left, left))
		case store[i+1] == nil:
			store[offset] = Hash(concat(store[i], store[i]))
		}
		offset++
	}
	return store[size-1], store, nil
}

// nextPowerOfTwo() calculates the smallest power of 2 that is greater than or equal to the input value
func nextPowerOfTwo(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

// concat() concatenates two byte slices
func concat(a, b []byte) []byte {
	out := make([]byte, len(a)+len(b))
	copy(out, a)
	copy(out[len(a):], b)
	return out
}
