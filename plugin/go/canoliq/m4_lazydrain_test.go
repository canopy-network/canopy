package canoliq

import (
	"testing"
)

// TestM4LazyDrainCapPerBlock verifies the M4 cap: a single EndBlock drain
// fulfills at most maxLazyDrainPerBlock queries, so an unauthenticated RPC
// burst cannot force the whole queue (up to lazyQueueCapacity) of serial
// StateReads inside one consensus EndBlock. The remainder stays buffered for
// subsequent blocks.
func TestM4LazyDrainCapPerBlock(t *testing.T) {
	c, _ := newTestCanoliq()
	c.plugin.pendingQueries = make(chan *lazyQuery, lazyQueueCapacity)

	const enqueued = maxLazyDrainPerBlock + 5
	for i := 0; i < enqueued; i++ {
		c.plugin.pendingQueries <- &lazyQuery{
			kind:   lazyKindAccount,
			addr:   addr20(byte(i + 1)),
			result: make(chan lazyResult, 1), // buffered so fulfillLazy never blocks
		}
	}

	// First drain fulfills exactly the cap; the rest remain queued.
	c.drainLazyQueries()
	if remaining := len(c.plugin.pendingQueries); remaining != enqueued-maxLazyDrainPerBlock {
		t.Fatalf("after first drain: %d queued, want %d (cap=%d)",
			remaining, enqueued-maxLazyDrainPerBlock, maxLazyDrainPerBlock)
	}

	// A second block drains the remainder (well under the cap).
	c.drainLazyQueries()
	if remaining := len(c.plugin.pendingQueries); remaining != 0 {
		t.Fatalf("after second drain: %d still queued, want 0", remaining)
	}
}
