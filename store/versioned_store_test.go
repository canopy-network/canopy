package store

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestMakeVersionedKeyRoundTrip(t *testing.T) {
	vs := &VersionedStore{}
	userKey := lib.JoinLenPrefix([]byte("params"), []byte("cons"))
	version := uint64(42)

	got := vs.makeVersionedKey(userKey, version)
	require.Len(t, got, len(userKey)+VersionSize)

	decodedKey, decodedVersion, err := parseVersionedKey(got, true)
	require.NoError(t, err)
	require.True(t, bytes.Equal(userKey, decodedKey))
	require.Equal(t, version, decodedVersion)
}

func TestMakeVersionedKeyConcurrent(t *testing.T) {
	vs := &VersionedStore{}
	keys := [][]byte{
		lib.JoinLenPrefix([]byte("a")),
		lib.JoinLenPrefix([]byte("params"), []byte("cons")),
		lib.JoinLenPrefix([]byte("root"), []byte("chain"), []byte("height"), []byte("latest")),
		lib.JoinLenPrefix([]byte("x"), []byte("y"), []byte("z"), []byte("very-long-segment-for-contention")),
	}

	var failed atomic.Bool
	const goroutines = 32
	const iterations = 10000

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := keys[(id+i)%len(keys)]
				version := uint64(id*iterations + i)

				encoded := vs.makeVersionedKey(key, version)
				if len(encoded) != len(key)+VersionSize {
					failed.Store(true)
					return
				}

				decodedKey, decodedVersion, err := parseVersionedKey(encoded, true)
				if err != nil || decodedVersion != version || !bytes.Equal(decodedKey, key) {
					failed.Store(true)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	require.False(t, failed.Load(), "concurrent makeVersionedKey produced malformed keys")
}

