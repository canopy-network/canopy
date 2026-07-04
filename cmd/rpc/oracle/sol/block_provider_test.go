package sol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// fakeRPC drives the provider deterministically.
type fakeRPC struct {
	slot       uint64
	blocks     map[uint64]*fakeBlock // slot -> block, missing => skipped
	skippedErr error
}

type fakeBlock struct {
	hash, parent string
}

func (f *fakeRPC) GetSlot(ctx context.Context) (uint64, error) { return f.slot, nil }

func (f *fakeRPC) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	if _, ok := f.blocks[slot]; !ok {
		return nil, f.skippedErr // simulate a skipped slot
	}
	b := f.blocks[slot]
	return newBlock(slot, b.hash, b.parent, nil), nil
}

func (f *fakeRPC) Close() {}

// fakeValidator is a minimal OrderValidator for testing.
type fakeValidator struct {
	lockErr  error
	closeErr error
}

func (f *fakeValidator) ValidateOrderJsonBytes(jsonBytes []byte, orderType string) error {
	return nil
}

var (
	errNotOrder = errors.New("not an order")
)

func TestProvider_SkipsMissingSlot(t *testing.T) {
	// slot 2 is missing (skipped); provider must advance past it and deliver 1 and 3.
	rpc := &fakeRPC{
		slot: 3,
		blocks: map[uint64]*fakeBlock{
			1: {hash: "h1", parent: "h0"},
			3: {hash: "h3", parent: "h2"},
		},
		skippedErr: &skipErr{code: -32007},
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

// skipErr is a test stand-in for the SDK jsonrpc error carrying a code.
type skipErr struct{ code int }

func (e *skipErr) Error() string { return "skipped" }
func (e *skipErr) Code() int     { return e.code }

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

func (c *countingRPC) Close() {}

var _ = errors.New // keep errors imported if unused above
