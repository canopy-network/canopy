package crypto

import (
	"crypto/rand"
	"fmt"
	"github.com/stretchr/testify/require"
	math "math/rand"
	"testing"
	"time"
)

func TestKeyBatchFuzz(t *testing.T) {
	for i := 0; i < 100; i++ {
		b := NewBatchVerifier()
		var expectedBadIndices []int
		for j := 0; j < 100; j++ {
			var (
				privateKey PrivateKeyI
				publicKey  PublicKeyI
				signature  []byte
			)
			message := make([]byte, math.Intn(32)+1)
			_, err := rand.Read(message)
			require.NoError(t, err)
			switch math.Intn(4) {
			case 0:
				privateKey, err = NewEd25519PrivateKey()
				require.NoError(t, err)
			case 1:
				privateKey, err = NewSECP256K1PrivateKey()
				require.NoError(t, err)
			case 2:
				privateKey, err = NewETHSECP256K1PrivateKey()
				require.NoError(t, err)
			case 3:
				privateKey, err = NewBLS12381PrivateKey()
				require.NoError(t, err)
			}
			publicKey = privateKey.PublicKey()
			signature = privateKey.Sign(message)
			// 1% chance invalid
			if math.Intn(100) < 1 {
				expectedBadIndices = append(expectedBadIndices, j)
				_, err = rand.Read(signature)
				require.NoError(t, err)
			}
			// add to key
			require.NoError(t, b.Add(publicKey, publicKey.Bytes(), message, signature))
		}
		require.Equal(t, expectedBadIndices, b.Verify())
	}
}

func TestBenchmarkBatch(t *testing.T) {
	DisableCache = true
	var batchAddTime, batchVerifyTime time.Duration
	var linearAddTime, linearVerifyTime time.Duration
	verifyBatch := func(tuples []BatchTuple) bool {
		for _, tuple := range tuples {
			if ok := tuple.PublicKey.VerifyBytes(tuple.Message, tuple.Signature); !ok {
				return false
			}
		}
		return true
	}
	var linearVerify []BatchTuple
	b := NewBatchVerifier()
	for j := 0; j < 200_000; j++ {
		var (
			privateKey PrivateKeyI
			publicKey  PublicKeyI
			signature  []byte
		)
		message := make([]byte, math.Intn(400)+1)
		_, err := rand.Read(message)
		require.NoError(t, err)
		switch 0 { // benchmark whichever key type interested in
		case 0:
			privateKey, err = NewEd25519PrivateKey()
			require.NoError(t, err)
		case 1:
			privateKey, err = NewSECP256K1PrivateKey()
			require.NoError(t, err)
		case 2:
			privateKey, err = NewETHSECP256K1PrivateKey()
			require.NoError(t, err)
		case 3:
			privateKey, err = NewBLS12381PrivateKey()
			require.NoError(t, err)
		}
		publicKey = privateKey.PublicKey()
		signature = privateKey.Sign(message)
		pubKeyBz := publicKey.Bytes()
		s := time.Now()
		require.NoError(t, b.Add(publicKey, pubKeyBz, message, signature))
		batchAddTime += time.Since(s)
		// add to key
		s = time.Now()
		linearVerify = append(linearVerify, BatchTuple{PublicKey: publicKey, Message: message, Signature: signature})
		linearAddTime += time.Since(s)
	}
	s := time.Now()
	require.Equal(t, []int(nil), b.Verify())
	batchVerifyTime = time.Since(s)
	s = time.Now()
	verifyBatch(linearVerify)
	linearVerifyTime = time.Since(s)
	fmt.Printf("BATCH add: %s, verify: %s\n", batchAddTime, batchVerifyTime)
	fmt.Printf("Linear add: %s, verify: %s\n", linearAddTime, linearVerifyTime)
}
