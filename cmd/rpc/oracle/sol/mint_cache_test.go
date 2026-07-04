package sol

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// makeMintData builds a byte slice with `decimals` at offset 44, mimicking an SPL Mint account.
func makeMintData(decimals byte) []byte {
	data := make([]byte, 82) // SPL Mint account is 82 bytes
	data[mintDecimalsOffset] = decimals
	return data
}

func TestDecodeMintDecimals(t *testing.T) {
	data := makeMintData(6)
	got, err := decodeMintDecimals(data)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 6 {
		t.Fatalf("expected 6 decimals, got %d", got)
	}

	// too-small data must error, not panic
	if _, err := decodeMintDecimals(make([]byte, 10)); err == nil {
		t.Fatal("expected error for undersized mint data")
	}
}

// makeTokenAccountData builds a byte slice with `mint` at offset 0, mimicking an SPL Token Account.
func makeTokenAccountData(mint solana.PublicKey) []byte {
	data := make([]byte, 165) // SPL Token Account is 165 bytes
	copy(data[tokenAccountMintOffset:], mint.Bytes())
	return data
}

func TestDecodeTokenAccountMint(t *testing.T) {
	mint := solana.NewWallet().PublicKey()
	got, err := decodeTokenAccountMint(makeTokenAccountData(mint))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.Equals(mint) {
		t.Fatalf("expected mint %s, got %s", mint, got)
	}

	// too-small data must error, not panic
	if _, err := decodeTokenAccountMint(make([]byte, 10)); err == nil {
		t.Fatal("expected error for undersized token account data")
	}
}

func TestResolveTokenAccountMint_HappyPath(t *testing.T) {
	mint := solana.NewWallet().PublicKey()
	fetcher := &fakeAccountFetcher{
		result: &rpc.GetAccountInfoResult{
			Value: &rpc.Account{Data: rpc.DataBytesOrJSONFromBytes(makeTokenAccountData(mint))},
		},
	}
	got, err := resolveTokenAccountMint(context.Background(), fetcher, solana.NewWallet().PublicKey())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.Equals(mint) {
		t.Fatalf("expected mint %s, got %s", mint, got)
	}
}

func TestResolveTokenAccountMint_FetcherError(t *testing.T) {
	fetcher := &fakeAccountFetcher{err: errors.New("rpc unavailable")}
	if _, err := resolveTokenAccountMint(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher fails")
	}
}

func TestResolveTokenAccountMint_NilResult(t *testing.T) {
	fetcher := &fakeAccountFetcher{result: nil}
	if _, err := resolveTokenAccountMint(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher returns nil result")
	}
}

func TestResolveTokenAccountMint_NilValue(t *testing.T) {
	fetcher := &fakeAccountFetcher{result: &rpc.GetAccountInfoResult{Value: nil}}
	if _, err := resolveTokenAccountMint(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher returns a result with a nil Value")
	}
}

func TestMintCache_HitMiss(t *testing.T) {
	c := newMintCache(2)
	mintA := "MintAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	mintB := "MintBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

	if _, ok := c.get(mintA); ok {
		t.Fatal("expected miss on empty cache")
	}
	c.put(mintA, 6)
	if d, ok := c.get(mintA); !ok || d != 6 {
		t.Fatalf("expected hit d=6, got d=%d ok=%v", d, ok)
	}
	// exceed capacity: mintA should evict when a third distinct entry is added
	c.put(mintB, 9)
	c.put("MintCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC", 0)
	if _, ok := c.get(mintA); ok {
		t.Fatal("expected mintA evicted after exceeding capacity")
	}
}

// fakeAccountFetcher lets tests control GetAccountInfo's result/error and count calls,
// so cache-hit behavior (no refetch) can be asserted.
type fakeAccountFetcher struct {
	result *rpc.GetAccountInfoResult
	err    error
	calls  int
}

func (f *fakeAccountFetcher) GetAccountInfo(_ context.Context, _ solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	f.calls++
	return f.result, f.err
}

func TestMintCache_Decimals_MissThenFetchThenCacheHit(t *testing.T) {
	fetcher := &fakeAccountFetcher{
		result: &rpc.GetAccountInfoResult{
			Value: &rpc.Account{Data: rpc.DataBytesOrJSONFromBytes(makeMintData(6))},
		},
	}
	c := newMintCache(4)
	mint := solana.NewWallet().PublicKey()

	got, err := c.Decimals(context.Background(), fetcher, mint)
	if err != nil {
		t.Fatalf("unexpected err on miss: %v", err)
	}
	if got != 6 {
		t.Fatalf("expected decimals 6, got %d", got)
	}
	if fetcher.calls != 1 {
		t.Fatalf("expected 1 fetch call after miss, got %d", fetcher.calls)
	}

	got, err = c.Decimals(context.Background(), fetcher, mint)
	if err != nil {
		t.Fatalf("unexpected err on hit: %v", err)
	}
	if got != 6 {
		t.Fatalf("expected decimals 6 on cache hit, got %d", got)
	}
	if fetcher.calls != 1 {
		t.Fatalf("expected no refetch on cache hit, got %d calls", fetcher.calls)
	}
}

func TestMintCache_Decimals_FetcherError(t *testing.T) {
	fetcher := &fakeAccountFetcher{err: errors.New("rpc unavailable")}
	c := newMintCache(4)
	if _, err := c.Decimals(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher fails")
	}
}

func TestMintCache_Decimals_NilResult(t *testing.T) {
	fetcher := &fakeAccountFetcher{result: nil}
	c := newMintCache(4)
	if _, err := c.Decimals(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher returns nil result")
	}
}

func TestMintCache_Decimals_NilValue(t *testing.T) {
	fetcher := &fakeAccountFetcher{result: &rpc.GetAccountInfoResult{Value: nil}}
	c := newMintCache(4)
	if _, err := c.Decimals(context.Background(), fetcher, solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected error when fetcher returns a result with a nil Value")
	}
}
