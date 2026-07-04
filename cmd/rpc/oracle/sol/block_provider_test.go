package sol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

// fakeRPC drives the provider deterministically.
type fakeRPC struct {
	slot              uint64
	blocks            map[uint64]*fakeBlock // slot -> block, missing => skipped
	skippedErr        error
	firstAvailable    uint64
	firstAvailableErr error
}

type fakeBlock struct {
	hash, parent string
}

func (f *fakeRPC) GetSlot(ctx context.Context) (uint64, error) { return f.slot, nil }

func (f *fakeRPC) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	if _, ok := f.blocks[slot]; !ok {
		return nil, f.skippedErr // simulate a skipped or pruned slot
	}
	b := f.blocks[slot]
	return newBlock(slot, b.hash, b.parent, nil), nil
}

func (f *fakeRPC) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	return f.firstAvailable, f.firstAvailableErr
}

func (f *fakeRPC) Close() {}

// Note: fakeValidator and errNotOrder are defined in transaction_test.go

// TestRpcErrCode_MatchesRealJsonrpcError guards against the exact bug this fix addresses:
// jsonrpc.RPCError.Code is a struct field, not a method, so a naive `interface{ Code() int }`
// + errors.As check (the original implementation) never matches a real *jsonrpc.RPCError and
// silently misclassifies every pruned/skipped-slot error as a generic transient failure.
func TestRpcErrCode_MatchesRealJsonrpcError(t *testing.T) {
	code, ok := rpcErrCode(&jsonrpc.RPCError{Code: -32001})
	if !ok || code != -32001 {
		t.Fatalf("expected code -32001, ok=true; got code=%d ok=%v", code, ok)
	}
	if !isPrunedSlotErr(&jsonrpc.RPCError{Code: -32001}) {
		t.Fatal("expected -32001 to be classified as a pruned-slot error")
	}
	if !isSkippedSlotErr(&jsonrpc.RPCError{Code: -32007}) {
		t.Fatal("expected -32007 to be classified as a skipped-slot error")
	}
	if isPrunedSlotErr(errors.New("some other transient error")) {
		t.Fatal("a non-jsonrpc error must not be classified as pruned")
	}
}

func TestProvider_SkipsMissingSlot(t *testing.T) {
	// slot 2 is missing (skipped); provider must advance past it and deliver 1 and 3.
	rpc := &fakeRPC{
		slot: 3,
		blocks: map[uint64]*fakeBlock{
			1: {hash: "h1", parent: "h0"},
			3: {hash: "h3", parent: "h2"},
		},
		skippedErr: &jsonrpc.RPCError{Code: -32007},
	}
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{PollIntervalMs: 1, RetryDelay: 1},
		rpc, &fakeValidator{lockErr: errNotOrder, closeErr: errNotOrder},
		lib.NewDefaultLogger(), nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Start(ctx, 1)

	got := make(map[uint64]bool)
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case b := <-p.BlockCh():
			got[b.Number()] = true
		case <-timeout:
			t.Fatalf("timed out; delivered slots: %v", got)
		}
	}
	if !got[1] || !got[3] {
		t.Fatalf("expected slots 1 and 3 delivered, got %v", got)
	}
}

func TestProvider_JumpsPrunedSlotRange(t *testing.T) {
	// slots 1-4 are pruned (BlockCleanedUp); first available is 5, which has a real block.
	// provider must jump straight to 5 in one shot, not retry 1-4 one at a time.
	rpc := &fakeRPC{
		slot: 5,
		blocks: map[uint64]*fakeBlock{
			5: {hash: "h5", parent: "h4"},
		},
		skippedErr:     &jsonrpc.RPCError{Code: -32001},
		firstAvailable: 5,
	}
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{PollIntervalMs: 1, RetryDelay: 1},
		rpc, &fakeValidator{lockErr: errNotOrder, closeErr: errNotOrder},
		lib.NewDefaultLogger(), nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Start(ctx, 1)

	select {
	case b := <-p.BlockCh():
		if b.Number() != 5 {
			t.Fatalf("expected slot 5, got %d", b.Number())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for jumped-to block")
	}
}

func TestProvider_RetriesTransientError(t *testing.T) {
	// slot 1 errors transiently on first GetBlock, succeeds on retry; must not advance past it.
	attempts := 0
	rpc := &countingRPC{
		slot: 1,
		onGetBlock: func(slot uint64) (*Block, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("transient rpc error") // not a skip code
			}
			return newBlock(slot, "h1", "h0", nil), nil
		},
	}
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{PollIntervalMs: 1, RetryDelay: 1},
		rpc, &fakeValidator{lockErr: errNotOrder, closeErr: errNotOrder},
		lib.NewDefaultLogger(), nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Start(ctx, 1)

	select {
	case b := <-p.BlockCh():
		if b.Number() != 1 {
			t.Fatalf("expected slot 1, got %d", b.Number())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for retried block")
	}
	if attempts < 2 {
		t.Fatalf("expected at least 2 GetBlock attempts, got %d", attempts)
	}
}

func TestBackoff_ReturnsTrueAfterDelay(t *testing.T) {
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{RetryDelay: 0}, // 0s: resolves immediately, just exercises the happy path
		&fakeRPC{}, &fakeValidator{}, lib.NewDefaultLogger(), nil,
	)
	if !p.backoff(context.Background()) {
		t.Fatal("expected backoff to return true when ctx is not canceled")
	}
}

func TestBackoff_ReturnsFalseWhenContextCanceled(t *testing.T) {
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{RetryDelay: 5},
		&fakeRPC{}, &fakeValidator{}, lib.NewDefaultLogger(), nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled
	if p.backoff(ctx) {
		t.Fatal("expected backoff to return false when ctx is already canceled")
	}
}

func TestProvider_GetSlotFailure_BacksOffBeforeRetry(t *testing.T) {
	// GetSlot fails every call; provider must not spin the bare poll interval (1ms) forever -
	// backoff(ctx) should be invoked and honor context cancellation promptly.
	calls := 0
	rpc := &countingSlotRPC{
		onGetSlot: func() (uint64, error) {
			calls++
			return 0, errors.New("rpc down")
		},
	}
	p := newSolBlockProviderWithClient(
		lib.SolBlockProviderConfig{PollIntervalMs: 1, RetryDelay: 1},
		rpc, &fakeValidator{lockErr: errNotOrder, closeErr: errNotOrder},
		lib.NewDefaultLogger(), nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(50 * time.Millisecond) // well within the 1s RetryDelay of the first backoff
	cancel()
	time.Sleep(50 * time.Millisecond)
	if calls > 1 {
		t.Fatalf("expected GetSlot backoff to prevent more than 1 call within 50ms, got %d", calls)
	}
}

type countingSlotRPC struct {
	onGetSlot func() (uint64, error)
}

func (c *countingSlotRPC) GetSlot(ctx context.Context) (uint64, error) { return c.onGetSlot() }
func (c *countingSlotRPC) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	return nil, errors.New("unused")
}
func (c *countingSlotRPC) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}
func (c *countingSlotRPC) Close() {}

type countingRPC struct {
	slot       uint64
	onGetBlock func(slot uint64) (*Block, error)
}

func (c *countingRPC) GetSlot(ctx context.Context) (uint64, error) {
	return c.slot, nil
}

func (c *countingRPC) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	return c.onGetBlock(slot)
}

func (c *countingRPC) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (c *countingRPC) Close() {}

var _ = errors.New // keep errors imported if unused above
