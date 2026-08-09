package store

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// TestGetOrComputeCommittee_CacheHitAvoidsCompute proves a second call for the same
// (chainId, rootHeight) is served from cache - compute (LoadCommittee's stand-in) never runs again.
func TestGetOrComputeCommittee_CacheHitAvoidsCompute(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	const chainId, rootHeight = uint64(1), uint64(100)
	want := &lib.ValidatorSet{TotalPower: 500, NumValidators: 5}

	var computeCalls int32
	compute := func() (*lib.ValidatorSet, lib.ErrorI) {
		atomic.AddInt32(&computeCalls, 1)
		return want, nil
	}

	got, err := store.GetOrComputeCommittee(chainId, rootHeight, "hss", compute)
	require.NoError(t, err)
	require.Equal(t, want, got)

	got, err = store.GetOrComputeCommittee(chainId, rootHeight, "hss", compute)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.EqualValues(t, 1, atomic.LoadInt32(&computeCalls), "second call for the same key should be served from cache")
}

// TestGetOrComputeCommittee_DedupesConcurrentComputation proves N goroutines racing on the
// same uncached (chainId, rootHeight) share one computation - the compute closure runs exactly once.
func TestGetOrComputeCommittee_DedupesConcurrentComputation(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	const goroutines = 20
	const chainId, rootHeight = uint64(1), uint64(7)
	want := &lib.ValidatorSet{TotalPower: 300, NumValidators: 3}

	var computeCalls int32
	var wg sync.WaitGroup
	results := make([]*lib.ValidatorSet, goroutines)
	errs := make([]lib.ErrorI, goroutines)

	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = store.GetOrComputeCommittee(chainId, rootHeight, "hss", func() (*lib.ValidatorSet, lib.ErrorI) {
				atomic.AddInt32(&computeCalls, 1)
				return want, nil
			})
		}(i)
	}
	close(start)
	wg.Wait()

	require.EqualValues(t, 1, atomic.LoadInt32(&computeCalls), "compute closure should run exactly once for concurrent callers of the same key")
	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i])
		require.Equal(t, want, results[i])
	}

	// the result must now be durably cached, independent of singleflight
	cached, err := store.GetOrComputeCommittee(chainId, rootHeight, "hss", func() (*lib.ValidatorSet, lib.ErrorI) {
		t.Fatal("compute should not run again once the result is cached")
		return nil, nil
	})
	require.NoError(t, err)
	require.Equal(t, want, cached)
}

// TestGetOrComputeCommittee_UnrelatedKeysRunInParallel proves dedup is scoped per-key, not a
// global lock - two different (chainId, rootHeight) pairs computed concurrently both progress.
func TestGetOrComputeCommittee_UnrelatedKeysRunInParallel(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// (chainId=1, rootHeight=1)'s compute blocks until released
	go func() {
		defer wg.Done()
		_, err := store.GetOrComputeCommittee(1, 1, "hss", func() (*lib.ValidatorSet, lib.ErrorI) {
			<-release
			return &lib.ValidatorSet{TotalPower: 1, NumValidators: 1}, nil
		})
		require.NoError(t, err)
	}()

	// (chainId=2, rootHeight=1)'s compute must be able to finish even though the first is still blocked
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		_, err := store.GetOrComputeCommittee(2, 1, "hss", func() (*lib.ValidatorSet, lib.ErrorI) {
			return &lib.ValidatorSet{TotalPower: 2, NumValidators: 2}, nil
		})
		require.NoError(t, err)
		close(done)
	}()

	select {
	case <-done:
		// good: the second key completed without waiting on the first's in-flight computation
	case <-time.After(2 * time.Second):
		t.Fatal("second key's compute appears blocked on the first key's in-flight computation")
	}
	close(release)
	wg.Wait()
}
