package sol

import (
	"container/list"
	"context"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// mintDecimalsOffset is the byte offset of the u8 `decimals` field in an SPL Mint account.
// Layout: mintAuthority (COption<Pubkey>) 0-35, supply (u64) 36-43, decimals (u8) 44, ...
const mintDecimalsOffset = 44

// tokenAccountMintOffset is the byte offset of the `mint` Pubkey field in an SPL Token Account.
// Layout: mint (Pubkey) 0-31, owner (Pubkey) 32-63, amount (u64) 64-71, ...
const tokenAccountMintOffset = 0
const tokenAccountMintLen = 32

// AccountFetcher is the minimal RPC surface the mint cache needs (satisfied by *rpc.Client);
// declared as an interface so tests can inject a fake without a live RPC endpoint.
type AccountFetcher interface {
	GetAccountInfo(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error)
}

// decodeMintDecimals extracts the decimals byte from raw SPL Mint account data.
func decodeMintDecimals(data []byte) (uint8, error) {
	if len(data) <= mintDecimalsOffset {
		return 0, ErrMintAccountTooSmall
	}
	return data[mintDecimalsOffset], nil
}

// mintCache is a bounded LRU of mint base58 string -> decimals.
type mintCache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List               // front = most recently used
	items    map[string]*list.Element // mint -> element
}

type mintEntry struct {
	mint     string
	decimals uint8
}

func newMintCache(capacity int) *mintCache {
	if capacity < 1 {
		capacity = 1
	}
	return &mintCache{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *mintCache) get(mint string) (uint8, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[mint]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*mintEntry).decimals, true
	}
	return 0, false
}

func (c *mintCache) put(mint string, decimals uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[mint]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*mintEntry).decimals = decimals
		return
	}
	el := c.ll.PushFront(&mintEntry{mint: mint, decimals: decimals})
	c.items[mint] = el
	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*mintEntry).mint)
		}
	}
}

// decodeTokenAccountMint extracts the mint pubkey from raw SPL Token Account data.
func decodeTokenAccountMint(data []byte) (solana.PublicKey, error) {
	if len(data) < tokenAccountMintOffset+tokenAccountMintLen {
		return solana.PublicKey{}, ErrTokenAccountTooSmall
	}
	return solana.PublicKeyFromBytes(data[tokenAccountMintOffset : tokenAccountMintOffset+tokenAccountMintLen]), nil
}

// resolveTokenAccountMint fetches a token account and returns its mint. Used for bare SPL
// `Transfer` instructions (tag 3), which don't carry the mint inline unlike `TransferChecked`.
// Not cached here — each token account is looked up once per transaction, unlike mints which
// recur across many transactions.
func resolveTokenAccountMint(ctx context.Context, fetcher AccountFetcher, tokenAccount solana.PublicKey) (solana.PublicKey, error) {
	res, err := fetcher.GetAccountInfo(ctx, tokenAccount)
	if err != nil {
		return solana.PublicKey{}, err
	}
	if res == nil || res.Value == nil {
		return solana.PublicKey{}, ErrTokenAccountTooSmall
	}
	return decodeTokenAccountMint(res.Value.Data.GetBinary())
}

// Decimals returns the decimals for a mint, fetching + caching on a miss.
func (c *mintCache) Decimals(ctx context.Context, fetcher AccountFetcher, mint solana.PublicKey) (uint8, error) {
	key := mint.String()
	if d, ok := c.get(key); ok {
		return d, nil
	}
	res, err := fetcher.GetAccountInfo(ctx, mint)
	if err != nil {
		return 0, err
	}
	if res == nil || res.Value == nil {
		return 0, ErrMintAccountTooSmall
	}
	d, err := decodeMintDecimals(res.Value.Data.GetBinary())
	if err != nil {
		return 0, err
	}
	c.put(key, d)
	return d, nil
}
