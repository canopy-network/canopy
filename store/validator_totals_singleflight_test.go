package store

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// TestGetOrComputeValidatorTotals_DedupesConcurrentComputation proves N goroutines racing
// on the same uncached version share one computation - the compute closure runs exactly once.
func TestGetOrComputeValidatorTotals_DedupesConcurrentComputation(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	const goroutines = 20
	const version = uint64(7)
	want := &lib.ValidatorTotals{ValidatorsActive: 3, DelegatesActive: 1}

	var computeCalls int32
	var wg sync.WaitGroup
	results := make([]*lib.ValidatorTotals, goroutines)
	errs := make([]lib.ErrorI, goroutines)

	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = store.GetOrComputeValidatorTotals(version, func() (*lib.ValidatorTotals, lib.ErrorI) {
				atomic.AddInt32(&computeCalls, 1)
				return want, nil
			})
		}(i)
	}
	close(start)
	wg.Wait()

	require.EqualValues(t, 1, atomic.LoadInt32(&computeCalls), "compute closure should run exactly once for concurrent callers of the same version")
	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i])
		require.Equal(t, want, results[i])
	}

	// the result must now be durably cached, independent of singleflight
	cached, available, err := store.GetValidatorTotals(version)
	require.NoError(t, err)
	require.True(t, available)
	require.Equal(t, want, cached)
}

// TestGetOrComputeValidatorTotals_UnrelatedVersionsRunInParallel proves dedup is scoped
// per-version, not a global lock - two different versions computed concurrently both progress.
func TestGetOrComputeValidatorTotals_UnrelatedVersionsRunInParallel(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// version 1's compute blocks until released
	go func() {
		defer wg.Done()
		_, err := store.GetOrComputeValidatorTotals(1, func() (*lib.ValidatorTotals, lib.ErrorI) {
			<-release
			return &lib.ValidatorTotals{ValidatorsActive: 1}, nil
		})
		require.NoError(t, err)
	}()

	// version 2's compute must be able to finish even though version 1's is still blocked
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		_, err := store.GetOrComputeValidatorTotals(2, func() (*lib.ValidatorTotals, lib.ErrorI) {
			return &lib.ValidatorTotals{ValidatorsActive: 2}, nil
		})
		require.NoError(t, err)
		close(done)
	}()

	select {
	case <-done:
		// good: version 2 completed without waiting on version 1
	case <-time.After(2 * time.Second):
		t.Fatal("version 2's compute appears blocked on version 1's in-flight computation")
	}
	close(release)
	wg.Wait()
}
