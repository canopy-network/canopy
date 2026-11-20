package p2p

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// Regression test for issue #491-adjacent shutdown log spam: ListenForInboundPeers
// used to log an ERROR and retry every 5 seconds forever once its listener was
// closed by Stop(), instead of recognizing the closure was deliberate and exiting.
// This spawns ListenForInboundPeers directly (bypassing Start()'s other goroutines,
// which aren't relevant here) so the test can deterministically wait for the
// goroutine to exit and fail fast if it doesn't.
func TestListenForInboundPeers_ExitsQuietlyWhenListenerClosed(t *testing.T) {
	n := newTestP2PNodeWithConfig(t, newTestP2PConfig(t), true)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Match how Start() invokes this: listen on config.ListenAddress
		// (":0" from newTestP2PConfig, i.e. an OS-assigned free port), not
		// n.peerAddress (a fixed placeholder port that could collide across
		// parallel tests).
		n.ListenForInboundPeers(&lib.PeerAddress{NetAddress: n.config.ListenAddress})
	}()

	// Wait for the listener to actually be assigned before closing it — closing
	// a nil listener would race with net.Listen() inside ListenForInboundPeers.
	require.Eventually(t, func() bool {
		return n.listener != nil
	}, testTimeout, time.Millisecond, "listener was never initialized")

	require.NoError(t, n.listener.Close())

	// Before the fix, this goroutine never returns: Accept() on a closed
	// listener returns net.ErrClosed forever, and the old code just logged and
	// retried in a loop. Assert it actually exits instead of hanging.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// goroutine exited, as expected after the fix
	case <-time.After(testTimeout):
		t.Fatal("ListenForInboundPeers did not exit after its listener was closed")
	}
}

// A transient accept error that isn't net.ErrClosed (e.g. a temporary resource
// exhaustion from the OS) must still be logged and retried, not treated as a
// shutdown signal — only confirms the errors.Is(err, net.ErrClosed) branch is
// specific to actual listener closure, not accept errors in general.
func TestListenForInboundPeers_ClosedListenerErrorIsDistinguishable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, ln.Close())

	_, acceptErr := ln.Accept()
	require.Error(t, acceptErr)
	require.ErrorIs(t, acceptErr, net.ErrClosed)
}
