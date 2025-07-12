package crypto

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/allegro/bigcache/v3"
	oasisEd25519 "github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	"sort"
	"sync"
	"time"
)

/* This file implements a multithreaded, batch signature verifier over SECP256K1, ED25519, and BDN (BLS) Keys */

var (
	DisableCache      = false
	SignatureCache, _ = bigcache.New(context.Background(), bigcache.Config{
		Shards:             1024,             // Good for parallelism (use 2x GOMAXPROCS as rule of thumb)
		LifeWindow:         5 * time.Minute,  // How long an entry stays in cache
		CleanWindow:        20 * time.Second, // How often to clean expired entries
		MaxEntriesInWindow: 1_00_000,         // Number of items you want in the cache
		MaxEntrySize:       1000,             // Slightly overestimate to account for key/metadata (in bytes)
		HardMaxCacheSize:   250,              // Total MB
		Verbose:            false,
	})
)

// BatchVerifier is an efficient, multi-threaded, batch verifier for many common keys
type BatchVerifier struct {
	ed25519      [8][]BatchTuple // 8 lists of ED25519 tuples
	ethSecp256k1 [8][]BatchTuple // 8 lists of ETHSECP256K1 tuples
	secp256k1    [8][]BatchTuple // 8 lists of SECP256K1 tuples
	bls12381     [8][]BatchTuple // 8 lists of BLS tuples
	count        int             // number of signatures in the batch
	noOp         bool            // no op disables all batch verifier operations
}

// BatchTuple is a convenient structure to validate the batch
type BatchTuple struct {
	PublicKey PublicKeyI // the public key associated with the batch tuple
	Message   []byte     // the unique message payload
	Signature []byte     // the digital signature that corresponds to the message and public key
	index     int        // the master index of the batch tuple among all threads
}

// NewBatchVerifier() constructs a batch verifier
func NewBatchVerifier(noOp ...bool) (b *BatchVerifier) {
	// if no operation enabled
	if noOp != nil {
		// skip initialization and configure as 'no op' mode
		return &BatchVerifier{noOp: true}
	}
	// setup capacity per thread
	capacity := 20_000
	// initialize the batch verifier
	b = new(BatchVerifier)
	// create 8 different batch tuple lists for each key type
	for i := 0; i < 8; i++ {
		b.ed25519[i] = make([]BatchTuple, 0, capacity)
		b.ethSecp256k1[i] = make([]BatchTuple, 0, capacity)
		b.secp256k1[i] = make([]BatchTuple, 0, capacity)
		b.bls12381[i] = make([]BatchTuple, 0, capacity)
	}
	return
}

// Add() adds a tuple to the batch verifier
func (b *BatchVerifier) Add(pk PublicKeyI, publicKey, message, signature []byte) (err error) {
	// if no-op enabled
	if b.noOp {
		// exit
		return nil
	}
	// initialize the batch tuple object and append to the proper list
	t := BatchTuple{PublicKey: pk, Message: message, Signature: signature, index: b.count}
	listIdx := b.count % 8
	// depending on the public key length
	switch len(publicKey) {
	case Ed25519PubKeySize:
		b.ed25519[listIdx] = append(b.ed25519[listIdx], t)
	case ETHSECP256K1PubKeySize, ETHSECP256K1PubKeySize + 1:
		b.ethSecp256k1[listIdx] = append(b.ethSecp256k1[listIdx], t)
	case SECP256K1PubKeySize:
		b.secp256k1[listIdx] = append(b.secp256k1[listIdx], t)
	case BLS12381PubKeySize:
		b.bls12381[listIdx] = append(b.bls12381[listIdx], t)
	default:
		return fmt.Errorf("unrecognized public key format")
	}
	// increment the global count
	b.count++
	// exit
	return
}

// Verify() returns the indices of bad signatures (if any)
func (b *BatchVerifier) Verify() (badIndices []int) {
	// initialize sync variables
	var wg sync.WaitGroup
	var mutex sync.Mutex
	// if no op enabled - exit
	if b.noOp {
		return nil
	}
	// for each 'thread'
	for i := 0; i < 8; i++ {
		// capture loop variable (important)
		idx := i
		// increment wait group
		wg.Add(1)
		// execute verification
		go func() {
			defer wg.Done()
			if bad := b.verifyAll(idx); len(bad) > 0 {
				mutex.Lock()
				badIndices = append(badIndices, bad...)
				mutex.Unlock()
			}
		}()
	}
	// wait until each thread completes
	wg.Wait()
	// if there exists bad indices
	if len(badIndices) != 0 {
		// sort them
		sort.Ints(badIndices)
	}
	return
}

// verifyAll() verifies a group of signatures and returns a list of bad signatures
func (b *BatchVerifier) verifyAll(idx int) (badIndices []int) {
	// callback function that verifies each tuple 1 by 1
	verifyBatch := func(tuples []BatchTuple) {
		// for each tuple
		for _, tuple := range tuples {
			// execute verification
			if ok := tuple.PublicKey.VerifyBytes(tuple.Message, tuple.Signature); ok {
				// if valid signature - add to cache
				SignatureCache.Set(tuple.Key(), []byte{0})
			} else {
				// if invalid signature add to bad list
				badIndices = append(badIndices, tuple.index)
			}
		}
		return
	}
	// verify ed25519
	if len(b.ed25519[idx]) != 0 {
		// leverage a batch verifier
		verifier := oasisEd25519.NewBatchVerifier()
		// initialize optimization variables
		cacheKeys, notInCache := make([]string, 0, len(b.ed25519[idx])), make(map[int]struct{})
		for i, t := range b.ed25519[idx] {
			// store the keys for the signature cache for optimal re-use
			cacheKeys = append(cacheKeys, t.Key())
			// if not found in the cache
			if _, notFoundErr := SignatureCache.Get(cacheKeys[i]); notFoundErr != nil {
				// save the index within this ed25519 list
				notInCache[i] = struct{}{}
				// add to the batch verifier
				verifier.Add(t.PublicKey.Bytes(), t.Message, t.Signature)
			}
		}
		// if any keys not in the cache
		if len(notInCache) != 0 {
			// verify the batch
			if !verifier.VerifyBatchOnly(rand.Reader) {
				// if batch verification fails, check each signature individually
				verifyBatch(b.ed25519[idx])
			} else {
				// if batch succeeded, just populate the cache
				for i := range notInCache {
					_ = SignatureCache.Set(cacheKeys[i], []byte{0})
				}
			}
		}
	}
	// verify ethSecp256k1
	verifyBatch(b.ethSecp256k1[idx])
	// verify secp256k1
	verifyBatch(b.secp256k1[idx])
	// verify bls12381
	verifyBatch(b.bls12381[idx])
	// all valid
	return
}

// Key() returns a unique string key for the cache
func (bt *BatchTuple) Key() string {
	// get the public key bytes
	pk := bt.PublicKey.Bytes()
	// calculate the total length of the key
	totalLen := len(pk) + len(bt.Message) + len(bt.Signature)
	// create the buffer and offset variables
	b, offset := make([]byte, totalLen), 0
	// copy pubkey in first part
	copy(b[offset:], pk)
	offset += len(pk)
	// copy message in second part
	copy(b[offset:], bt.Message)
	offset += len(bt.Message)
	// copy signature in third part
	copy(b[offset:], bt.Signature)
	// return string version
	return string(b)
}

// CheckCache() is a convenience function that checks the signature cache for a combination
// and returns a callback for the caller to easily add the signature to the cache
func CheckCache(pk PublicKeyI, msg, sig []byte) (found bool, addToCache func()) {
	// if cache disabled - no op
	if DisableCache {
		return false, func() {}
	}
	// create a tuple
	cacheTuple := BatchTuple{PublicKey: pk, Message: msg, Signature: sig}
	// get the key from the tuple
	key := cacheTuple.Key()
	// generate the 'add to cache' callback function
	addToCache = func() { SignatureCache.Set(key, []byte{0}) }
	// attempt to get the tuple from the cache
	_, notFoundErr := SignatureCache.Get(key)
	// check if found
	found = notFoundErr == nil
	// exit
	return
}
