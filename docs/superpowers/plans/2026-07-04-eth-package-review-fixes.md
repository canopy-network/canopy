# ETH Oracle Package Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 3 MUST, 7 SHOULD, and 5 CAN findings from the full-review of `cmd/rpc/oracle/eth` — data race, debug print, blocking channel send, constructor error handling, metrics API, error wrapping, test gaps, and dead code.

**Architecture:** All changes are confined to the `eth` package plus one call site in `cmd/cli/cli.go` and one struct addition in `lib/metrics.go`. No behavior change to block-processing semantics except making the channel send cancellable and the constructor fallible. Linear execution — most edits touch `block_provider.go`, so parallel work would only create merge conflicts.

**Tech Stack:** Go 1.24, `github.com/ethereum/go-ethereum`, `lib.LoggerI` (zap), `lib.Metrics` (Prometheus), `sync.Mutex`, table-driven tests with `github.com/google/go-cmp/cmp`.

---

## Environment Setup (do this once before any task)

The module does **not** build under the machine's default Go toolchain (go1.26.2): dependency `github.com/cockroachdb/swiss` uses runtime `//go:linkname` symbols (`fastrand64`, `hashFn`) that were removed after go1.24. `go.mod` pins `go 1.24.0`.

**Every `go build`/`go test`/`go vet` command in this plan MUST be prefixed with `GOTOOLCHAIN=go1.24.0`.** This was verified: `GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/` → `ok ... 1.081s`.

Do **not** commit a `toolchain` directive to `go.mod` — that changes the toolchain for the entire monorepo and is out of scope for this review. Export it in your shell for the session instead:

```bash
export GOTOOLCHAIN=go1.24.0
cd /home/enielson/canopy/canopy/oracle
```

Confirm the baseline is green before starting:

Run: `go test -race ./cmd/rpc/oracle/eth/`
Expected: `ok  github.com/canopy-network/canopy/cmd/rpc/oracle/eth`

---

## Task 1: Fix data race on `synced` (MUST)

`p.synced` is written in `monitorHeaders` (block_provider.go:277, 319) with **no lock**, but read in `IsSynced()` (142-145) **under `heightMu`**. `monitorHeaders` runs on the provider's `run` goroutine; `IsSynced()` is called by the consumer goroutine. A one-sided lock does not synchronize → data race. Route every read and write of `synced` through `heightMu`.

**Design decision:** Keep the existing `heightMu` mutex (rather than switching to `atomic.Bool`) — it is already a field, the test files construct the struct with `heightMu: &sync.Mutex{}` literals, and a `setSynced` helper is the smallest correct change.

**Files:**
- Modify: `cmd/rpc/oracle/eth/block_provider.go`
- Test: `cmd/rpc/oracle/eth/block_provider_test.go`

- [ ] **Step 1: Write the failing test**

Add to the end of `cmd/rpc/oracle/eth/block_provider_test.go`:

```go
func TestEthBlockProvider_syncedNoRace(t *testing.T) {
	p := &EthBlockProvider{heightMu: &sync.Mutex{}}
	var wg sync.WaitGroup
	wg.Add(2)
	// writer goroutine (simulates monitorHeaders)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			p.setSynced(i%2 == 0)
		}
	}()
	// reader goroutine (simulates consumer calling IsSynced)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = p.IsSynced()
		}
	}()
	wg.Wait()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run TestEthBlockProvider_syncedNoRace`
Expected: FAIL — compile error `p.setSynced undefined (type *EthBlockProvider has no field or method setSynced)`

- [ ] **Step 3: Add the `setSynced` helper**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:139-146
// IsSynced returns whether the block provider has synced to the top of the chain
func (p *EthBlockProvider) IsSynced() bool {
	// lock height mutex to safely read synced state
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	// return current sync status
	return p.synced
}

// NEW — cmd/rpc/oracle/eth/block_provider.go (insert directly after IsSynced)
// IsSynced returns whether the block provider has synced to the top of the chain
func (p *EthBlockProvider) IsSynced() bool {
	// lock height mutex to safely read synced state
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	// return current sync status
	return p.synced
}

// setSynced safely updates the synced flag under heightMu
func (p *EthBlockProvider) setSynced(v bool) {
	p.heightMu.Lock()
	p.synced = v
	p.heightMu.Unlock()
}
```

- [ ] **Step 4: Route the production writes and the ticker read through the lock**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:276-278
	// reset sync state for reconnection scenarios
	p.synced = false
	p.metrics.SetEthSyncStatus(1) // syncing

// NEW — cmd/rpc/oracle/eth/block_provider.go:276-278
	// reset sync state for reconnection scenarios
	p.setSynced(false)
	p.metrics.SetEthSyncStatus(1) // syncing
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:284-287
		case <-statusTicker.C:
			// periodic status update
			p.logger.Infof("[ETH-SYNC] status: nextHeight=%s, synced=%v", p.nextHeight.String(), p.synced)

// NEW — cmd/rpc/oracle/eth/block_provider.go:284-287
		case <-statusTicker.C:
			// periodic status update
			p.logger.Infof("[ETH-SYNC] status: nextHeight=%s, synced=%v", p.nextHeight.String(), p.IsSynced())
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:314-323
			// not synced to top
			if !p.synced {
				// check for source chain sync
				if p.nextHeight.Cmp(header.Number) == 0 {
					// we've caught up to the latest block, mark as synced
					p.synced = true
					p.logger.Infof("[ETH-SYNC] synced at height %s", p.nextHeight.String())
					p.metrics.SetEthSyncStatus(2) // synced
				}
			}

// NEW — cmd/rpc/oracle/eth/block_provider.go:314-323
			// not synced to top
			if !p.IsSynced() {
				// check for source chain sync
				if p.nextHeight.Cmp(header.Number) == 0 {
					// we've caught up to the latest block, mark as synced
					p.setSynced(true)
					p.logger.Infof("[ETH-SYNC] synced at height %s", p.nextHeight.String())
					p.metrics.SetEthSyncStatus(2) // synced
				}
			}
```

- [ ] **Step 5: Run tests under the race detector**

Run: `GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/`
Expected: PASS, no `DATA RACE` output.

- [ ] **Step 6: Commit**

```bash
git add cmd/rpc/oracle/eth/block_provider.go cmd/rpc/oracle/eth/block_provider_test.go
git commit -m "fix(oracle/eth): guard synced flag with heightMu to remove data race"
```

---

## Task 2: Remove debug `fmt.Println` / commented prints (MUST)

`transaction.go:141` calls `fmt.Println("validated lock order", ...)` on every validated lock order — logging via `fmt` is prohibited (must use `lib.LoggerI`, and `parseDataForOrders` has no logger). Also remove the dead commented `fmt.Println` lines at 121-122. `parseDataForOrders` has no other `fmt` usage, so the `"fmt"` import stays (still used by `fmt.Errorf`).

**Files:**
- Modify: `cmd/rpc/oracle/eth/transaction.go`

- [ ] **Step 1: Delete the debug print and dead comments**

```go
// CURRENT — cmd/rpc/oracle/eth/transaction.go:116-141
	recipient, amount, data, err := parseERC20Transfer(txData)
	if err != nil {
		// not an erc20 transfer - normal condition
		return nil
	}
	// fmt.Println("checking", t.from, recipient, amount, string(data), err)
	// fmt.Println("checking", t.from == recipient, amount.Cmp(big.NewInt(0)))
	// all Canopy swap ERC20 transfers have aux data
	if len(data) == 0 {
		// no data to process - not a canopy swap ERC20 transfer
		return nil
	}

	switch amount.Cmp(big.NewInt(0)) {
	case 0: // zero amount - potential lock order
		// test for self-sent ERC20 transfers, lock orders must be self-sent
		if t.from != recipient {
			break
		}
		// attempt to validate a lock order
		err = orderValidator.ValidateOrderJsonBytes(data, types.LockOrderType)
		if err != nil {
			// erc20 transaction did not contain canopy lock order json - normal condition
			return err
		}
		fmt.Println("validated lock order", t.from, recipient, amount, string(data), err)
		order := &lib.LockOrder{}

// NEW — cmd/rpc/oracle/eth/transaction.go:116-141
	recipient, amount, data, err := parseERC20Transfer(txData)
	if err != nil {
		// not an erc20 transfer - normal condition
		return nil
	}
	// all Canopy swap ERC20 transfers have aux data
	if len(data) == 0 {
		// no data to process - not a canopy swap ERC20 transfer
		return nil
	}

	switch amount.Cmp(big.NewInt(0)) {
	case 0: // zero amount - potential lock order
		// test for self-sent ERC20 transfers, lock orders must be self-sent
		if t.from != recipient {
			break
		}
		// attempt to validate a lock order
		err = orderValidator.ValidateOrderJsonBytes(data, types.LockOrderType)
		if err != nil {
			// erc20 transaction did not contain canopy lock order json - normal condition
			return err
		}
		order := &lib.LockOrder{}
```

- [ ] **Step 2: Verify it builds and `fmt` is still used**

Run: `GOTOOLCHAIN=go1.24.0 go build ./cmd/rpc/oracle/eth/`
Expected: builds cleanly (no "imported and not used: fmt" — `fmt.Errorf` remains at lines ~104, 146, 173).

- [ ] **Step 3: Run tests**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/rpc/oracle/eth/transaction.go
git commit -m "fix(oracle/eth): remove debug fmt.Println from order parsing"
```

---

## Task 3: Make block channel send cancellable (MUST)

`processBlocks` sends on the **unbuffered** `p.blockChan` at block_provider.go:407 with a bare `p.blockChan <- block`. If the consumer stalls, this blocks forever — neither `timeoutCtx.Done()` (the processBlock time limit) nor parent `ctx.Done()` (shutdown) can interrupt it, leaking the goroutine. Guard the send with a `select` on `timeoutCtx.Done()` (which is derived from the parent `ctx`, so it also fires on shutdown). On cancellation, return `next` (the current, not-yet-sent height) so the block is retried next round.

**Files:**
- Modify: `cmd/rpc/oracle/eth/block_provider.go`
- Test: `cmd/rpc/oracle/eth/block_provider_test.go`

- [ ] **Step 1: Write the failing test**

This test uses a **buffer-size-0** channel and a cancelled context; before the fix the send blocks forever and the test times out, after the fix it returns promptly at the start height. Add to `block_provider_test.go`:

```go
func TestEthBlockProvider_processBlocksCancelledSend(t *testing.T) {
	mockClient := &mockEthClient{
		blocks: map[uint64]*ethtypes.Block{
			50: createEthereumBlock(50, []*ethtypes.Transaction{}),
		},
	}
	provider := &EthBlockProvider{
		rpcClient:      mockClient,
		orderValidator: &mockOrderValidator{},
		logger:         lib.NewDefaultLogger(),
		blockChan:      make(chan types.BlockI), // unbuffered, no receiver
		chainId:        1,
		config:         lib.EthBlockProviderConfig{},
		heightMu:       &sync.Mutex{},
	}
	// cancel immediately: the send must not block forever
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan *big.Int, 1)
	go func() {
		done <- provider.processBlocks(ctx, big.NewInt(50), big.NewInt(50))
	}()

	select {
	case next := <-done:
		// block 50 was never delivered, so next height stays at 50 for retry
		if next.Cmp(big.NewInt(50)) != 0 {
			t.Errorf("expected next height 50 (unsent block retried), got %s", next.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("processBlocks blocked on channel send despite cancelled context")
	}
}
```

Add `"time"` to the test file's imports if not already present (it is not currently imported in `block_provider_test.go`):

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider_test.go:3-18
import (
	"context"
	"log"
	"math/big"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/google/go-cmp/cmp"
)

// NEW — cmd/rpc/oracle/eth/block_provider_test.go:3-19
import (
	"context"
	"log"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/google/go-cmp/cmp"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run TestEthBlockProvider_processBlocksCancelledSend -timeout 30s`
Expected: FAIL — `processBlocks blocked on channel send despite cancelled context`

- [ ] **Step 3: Guard the send with select**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:406-416
		// send block through channel
		p.blockChan <- block
		// log successful block processing
		// p.logger.Infof("eth block provider sent safe block at height %d through channel", next)
		// update counters
		blocksProcessed++
		transactionsProcessed += len(block.transactions)
		// update metrics with current block data
		p.metrics.UpdateEthBlockProviderMetrics(fetchTime, txProcessTime, 0, 0, 0, 0, 1, len(block.transactions), 0)
		// increment height for next iteration
		next.Add(next, big.NewInt(1))

// NEW — cmd/rpc/oracle/eth/block_provider.go:406-421
		// send block through channel, aborting if the context is cancelled or the
		// processBlock time limit is hit so a stalled consumer cannot block shutdown
		select {
		case p.blockChan <- block:
		case <-timeoutCtx.Done():
			p.logger.Warnf("[ETH-BLOCK] context done before sending block %d, will retry", next)
			p.metrics.IncrementEthBlockProcessingTimeout()
			return next
		}
		// update counters
		blocksProcessed++
		transactionsProcessed += len(block.transactions)
		// update metrics with current block data
		p.metrics.UpdateEthBlockProviderMetrics(fetchTime, txProcessTime, 0, 0, 0, 0, 1, len(block.transactions), 0)
		// increment height for next iteration
		next.Add(next, big.NewInt(1))
```

- [ ] **Step 4: Run tests (with race)**

Run: `GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/ -timeout 60s`
Expected: PASS — new test returns promptly with next=50; existing `TestEthBlockProvider_processBlocks` still passes (its channel is buffered size 10, so the send never blocks).

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/oracle/eth/block_provider.go cmd/rpc/oracle/eth/block_provider_test.go
git commit -m "fix(oracle/eth): make block channel send respect context cancellation"
```

---

## Task 4: Constructor returns error instead of Fatal (SHOULD)

`NewEthBlockProvider` (block_provider.go:71-98) calls `ethclient.Dial` at construction and `logger.Fatal` on failure — a constructor should not kill the process, and the dialed client is never `Close()`d and is separate from the ones `connect()` creates. Change the signature to return `(*EthBlockProvider, error)` and let the single caller decide. The only caller is `cmd/cli/cli.go:140`, which already uses the `x, e := ...; if e != nil { l.Fatal(e.Error()) }` pattern.

**Design decision:** Keep dialing the token-cache client in the constructor (do not restructure to share `connect()`'s client — that is a larger change out of scope). Just surface the error instead of calling Fatal.

**Files:**
- Modify: `cmd/rpc/oracle/eth/block_provider.go:71-98`
- Modify: `cmd/cli/cli.go:140`
- Test: `cmd/rpc/oracle/eth/block_provider_test.go`

- [ ] **Step 1: Write the failing test**

Add to `block_provider_test.go`. A malformed URL makes `ethclient.Dial` fail without any network:

```go
func TestNewEthBlockProvider_dialError(t *testing.T) {
	cfg := lib.EthBlockProviderConfig{NodeUrl: "not-a-valid-url://%%%"}
	_, err := NewEthBlockProvider(cfg, &mockOrderValidator{}, lib.NewDefaultLogger(), nil)
	if err == nil {
		t.Fatal("expected error from invalid NodeUrl, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run TestNewEthBlockProvider_dialError`
Expected: FAIL — compile error: `NewEthBlockProvider(...) used as value` / assignment mismatch (constructor returns 1 value, test wants 2).

- [ ] **Step 3: Change the constructor signature**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:70-98
// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(config lib.EthBlockProviderConfig, orderValidator OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) *EthBlockProvider {
	// create an ethereum client for the token cache
	ethClient, ethErr := ethclient.Dial(config.NodeUrl)
	if ethErr != nil {
		logger.Fatal("[ETH-CONN] " + ethErr.Error())
	}
	// create a new erc20 token cache
	tokenCache := NewERC20TokenCache(ethClient, metrics)
	// create the block output channel, this is unbuffered so the provider
	// halts processing until the receiver is ready to process more blocks
	ch := make(chan types.BlockI)
	// create new provider instance
	p := &EthBlockProvider{
		config:          config,
		blockChan:       ch,
		erc20TokenCache: tokenCache,
		logger:          logger,
		orderValidator:  orderValidator,
		nextHeight:      big.NewInt(0),
		chainId:         config.EVMChainId,
		synced:          false,
		heightMu:        &sync.Mutex{},
		metrics:         metrics,
	}
	// log provider creation
	p.logger.Infof("[ETH-CONN] created block provider with rpc: %s, ws: %s, chain id: %d", p.config.NodeUrl, p.config.NodeWSUrl, p.chainId)
	return p
}

// NEW — cmd/rpc/oracle/eth/block_provider.go:70-98
// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(config lib.EthBlockProviderConfig, orderValidator OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) (*EthBlockProvider, error) {
	// create an ethereum client for the token cache
	ethClient, ethErr := ethclient.Dial(config.NodeUrl)
	if ethErr != nil {
		return nil, fmt.Errorf("[ETH-CONN] failed to dial token-cache client: %w", ethErr)
	}
	// create a new erc20 token cache
	tokenCache := NewERC20TokenCache(ethClient, metrics)
	// create the block output channel, this is unbuffered so the provider
	// halts processing until the receiver is ready to process more blocks
	ch := make(chan types.BlockI)
	// create new provider instance
	p := &EthBlockProvider{
		config:          config,
		blockChan:       ch,
		erc20TokenCache: tokenCache,
		logger:          logger,
		orderValidator:  orderValidator,
		nextHeight:      big.NewInt(0),
		chainId:         config.EVMChainId,
		synced:          false,
		heightMu:        &sync.Mutex{},
		metrics:         metrics,
	}
	// log provider creation
	p.logger.Infof("[ETH-CONN] created block provider with rpc: %s, ws: %s, chain id: %d", p.config.NodeUrl, p.config.NodeWSUrl, p.chainId)
	return p, nil
}
```

(`fmt` is already imported in block_provider.go.)

- [ ] **Step 4: Update the caller in cli.go**

```go
// CURRENT — cmd/cli/cli.go:139-140
		// create the ethereum block provider
		ethBlockProvider := eth.NewEthBlockProvider(config.EthBlockProviderConfig, orderValidator, oracleLogger, metrics)

// NEW — cmd/cli/cli.go:139-143
		// create the ethereum block provider
		ethBlockProvider, e := eth.NewEthBlockProvider(config.EthBlockProviderConfig, orderValidator, oracleLogger, metrics)
		if e != nil {
			l.Fatal(e.Error())
		}
```

Note: `e` and `l` are already in scope at this point in cli.go (used identically at lines 133-135 for `oracle.NewOracleDiskStorage`). Verify `e` is reused, not re-declared with `:=` conflict — the block at 143-146 (`o, e = oracle.NewOracle(...)`) already reuses `e`, so `e` exists; if the compiler reports "no new variables on left side of :=", change `ethBlockProvider, e :=` to declare only the new var: `var e2 error; ethBlockProvider, e2 := ...` is wrong — instead read the surrounding scope and reuse the existing `e` with `=` if already declared above, or keep `:=` since `ethBlockProvider` is new (Go allows `:=` when at least one var on the left is new). `ethBlockProvider` is new here, so `:=` is valid.

- [ ] **Step 5: Run tests + build the cli**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ && GOTOOLCHAIN=go1.24.0 go build ./cmd/cli/`
Expected: tests PASS, cli builds cleanly.

- [ ] **Step 6: Commit**

```bash
git add cmd/rpc/oracle/eth/block_provider.go cmd/rpc/oracle/eth/block_provider_test.go cmd/cli/cli.go
git commit -m "refactor(oracle/eth): return error from NewEthBlockProvider instead of Fatal"
```

---

## Task 5: Replace 9-positional-arg metrics method with a named-field struct (SHOULD)

`UpdateEthBlockProviderMetrics(blockFetchTime, transactionProcessTime, receiptFetchTime time.Duration, cacheHits, cacheMisses, connectionErrors, blocksProcessed, transactionsProcessed, retries int)` is called at 8 sites with long runs of `0` — positionally opaque and transposition-prone (violates ">2 args → input struct"). Add a struct with named fields and a method that takes it; migrate all call sites; delete the old method.

**Design decision:** Put the struct in `lib` next to `Metrics` (the metrics type lives there and the struct is part of its API). Name it `EthBlockProviderMetricUpdate`. Keep the same nil-receiver guard behavior.

**Files:**
- Modify: `lib/metrics.go:1235` (the method)
- Modify: `cmd/rpc/oracle/eth/block_provider.go` (call sites: 380, 397, 414, 473, 554, 568)
- Modify: `cmd/rpc/oracle/eth/erc20_token_cache.go` (call sites: 59, 97)

- [ ] **Step 1: Read the current method body to preserve its logic exactly**

Run: `GOTOOLCHAIN=go1.24.0 sed -n '1235,1322p' lib/metrics.go`
Expected: the full body of `UpdateEthBlockProviderMetrics`. Note every field it observes/adds so the struct-based version is logically identical.

- [ ] **Step 2: Add the struct and new method, keep the old one delegating (temporary)**

Insert immediately **before** the current `func (m *Metrics) UpdateEthBlockProviderMetrics(` at lib/metrics.go:1235:

```go
// NEW — lib/metrics.go (insert before UpdateEthBlockProviderMetrics)

// EthBlockProviderMetricUpdate carries a batch of eth block-provider metric
// deltas. Zero-valued fields are no-ops, matching the previous positional API.
type EthBlockProviderMetricUpdate struct {
	BlockFetchTime         time.Duration
	TransactionProcessTime time.Duration
	ReceiptFetchTime       time.Duration
	CacheHits              int
	CacheMisses            int
	ConnectionErrors       int
	BlocksProcessed        int
	TransactionsProcessed  int
	Retries                int
}

// UpdateEthBlockProvider records a batch of eth block-provider metrics using
// named fields. Prefer this over the positional UpdateEthBlockProviderMetrics.
func (m *Metrics) UpdateEthBlockProvider(u EthBlockProviderMetricUpdate) {
	if m == nil {
		return
	}
	m.UpdateEthBlockProviderMetrics(
		u.BlockFetchTime, u.TransactionProcessTime, u.ReceiptFetchTime,
		u.CacheHits, u.CacheMisses, u.ConnectionErrors,
		u.BlocksProcessed, u.TransactionsProcessed, u.Retries,
	)
}
```

Rationale for delegating rather than moving the body: keeps this task low-risk and reviewable; the old method is deleted in Step 5 after all callers migrate.

- [ ] **Step 3: Migrate the 8 call sites to the named-field form**

Each edit maps positional args → named fields. The positional order is `(blockFetchTime, transactionProcessTime, receiptFetchTime, cacheHits, cacheMisses, connectionErrors, blocksProcessed, transactionsProcessed, retries)`.

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:380
			p.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 0, 1, blocksProcessed, transactionsProcessed, retries)
// NEW
			p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{
				ConnectionErrors:      1,
				BlocksProcessed:       blocksProcessed,
				TransactionsProcessed: transactionsProcessed,
				Retries:               retries,
			})
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:397
			p.metrics.UpdateEthBlockProviderMetrics(fetchTime, 0, 0, 0, 0, 1, blocksProcessed, transactionsProcessed, retries)
// NEW
			p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{
				BlockFetchTime:        fetchTime,
				ConnectionErrors:      1,
				BlocksProcessed:       blocksProcessed,
				TransactionsProcessed: transactionsProcessed,
				Retries:               retries,
			})
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:414
		p.metrics.UpdateEthBlockProviderMetrics(fetchTime, txProcessTime, 0, 0, 0, 0, 1, len(block.transactions), 0)
// NEW
		p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{
			BlockFetchTime:         fetchTime,
			TransactionProcessTime: txProcessTime,
			BlocksProcessed:        1,
			TransactionsProcessed:  len(block.transactions),
		})
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:473
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 0, 0, 0, 0, retryCount)
// NEW
		p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{Retries: retryCount})
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:554
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, receiptTime, 0, 0, 1, 0, 0, 0)
// NEW
		p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{
			ReceiptFetchTime: receiptTime,
			ConnectionErrors: 1,
		})
```

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:568
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, receiptTime, 0, 0, 0, 0, 0, 0)
// NEW
		p.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{ReceiptFetchTime: receiptTime})
```

```go
// CURRENT — cmd/rpc/oracle/eth/erc20_token_cache.go:58-60
		if m.metrics != nil {
			m.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 1, 0, 0, 0, 0, 0)
		}
// NEW  (the nil guard is now redundant — UpdateEthBlockProvider is nil-safe — but keep it to minimize churn)
		if m.metrics != nil {
			m.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{CacheHits: 1})
		}
```

```go
// CURRENT — cmd/rpc/oracle/eth/erc20_token_cache.go:96-98
	if m.metrics != nil {
		m.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 1, 0, 0, 0, 0)
	}
// NEW
	if m.metrics != nil {
		m.metrics.UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{CacheMisses: 1})
	}
```

- [ ] **Step 4: Verify no positional callers remain**

Run: `GOTOOLCHAIN=go1.24.0 grep -rn "UpdateEthBlockProviderMetrics" cmd/ lib/`
Expected: matches only inside `lib/metrics.go` (the old method definition + the delegating call in `UpdateEthBlockProvider`). No matches under `cmd/`.

- [ ] **Step 5: Delete the old positional method and inline its body into the new one**

Move the original body from `UpdateEthBlockProviderMetrics` into `UpdateEthBlockProvider` (replacing the delegating call), then delete `UpdateEthBlockProviderMetrics`. Concretely: replace the delegating body written in Step 2 with the real implementation from Step 1 (reading fields off `u.` instead of the positional params), then remove the `func (m *Metrics) UpdateEthBlockProviderMetrics(...) { ... }` block entirely.

Run afterward: `GOTOOLCHAIN=go1.24.0 grep -rn "UpdateEthBlockProviderMetrics" cmd/ lib/`
Expected: **no** matches.

- [ ] **Step 6: Build the whole module + run tests**

Run: `GOTOOLCHAIN=go1.24.0 go build ./... && GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/`
Expected: builds cleanly, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add lib/metrics.go cmd/rpc/oracle/eth/block_provider.go cmd/rpc/oracle/eth/erc20_token_cache.go
git commit -m "refactor(oracle/eth): replace 9-arg metrics call with named-field struct"
```

---

## Task 6: Wrap discarded errors in TokenInfo (SHOULD)

`ERC20TokenCache.TokenInfo` (erc20_token_cache.go:64-84) returns the bare sentinel `ErrTokenInfo` on each RPC failure, discarding which call failed and why. Wrap the underlying error with `%w` so logs carry context while callers can still `errors.Is(err, ErrTokenInfo)` (the retry check in `processBlockTransactions:444` relies on this).

**Files:**
- Modify: `cmd/rpc/oracle/eth/erc20_token_cache.go:63-84`
- Test: `cmd/rpc/oracle/eth/erc20_token_cache_test.go`

- [ ] **Step 1: Write the failing test**

Add to `erc20_token_cache_test.go`. Verify the wrapped error still satisfies `errors.Is(err, ErrTokenInfo)`:

```go
func TestERC20TokenCache_TokenInfo_WrapsSentinel(t *testing.T) {
	cache := NewERC20TokenCache(&mockContractCaller{
		names:    make(map[string][]byte),
		symbols:  make(map[string][]byte),
		decimals: make(map[string][]byte),
	}, nil)
	_, err := cache.TokenInfo(context.Background(), "0xC2c86a33E6441E6C7C5c8c8c8c8c8c8c8c8c8c8c")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTokenInfo) {
		t.Errorf("expected error to wrap ErrTokenInfo, got %v", err)
	}
}
```

Add `"errors"` to the test file imports:

```go
// CURRENT — cmd/rpc/oracle/eth/erc20_token_cache_test.go:3-13
import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/google/go-cmp/cmp"
)

// NEW — cmd/rpc/oracle/eth/erc20_token_cache_test.go:3-14
import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/google/go-cmp/cmp"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run TestERC20TokenCache_TokenInfo_WrapsSentinel`
Expected: FAIL — `expected error to wrap ErrTokenInfo` (current code returns bare `ErrTokenInfo`, which `errors.Is` matches — wait, bare sentinel DOES satisfy Is; the test must assert the *underlying* cause is present). Adjust: the point is to preserve context AND identity. Rewrite the test assertion to require both `errors.Is(err, ErrTokenInfo)` and that the message contains the underlying cause, so it fails on the bare-sentinel version:

```go
	if !errors.Is(err, ErrTokenInfo) {
		t.Errorf("expected error to wrap ErrTokenInfo, got %v", err)
	}
	if !strings.Contains(err.Error(), "contract address not found") {
		t.Errorf("expected wrapped underlying cause in message, got %q", err.Error())
	}
```

(`strings` is already imported in this test file.) With the bare-sentinel current code, the message is just `"failed to get token info"` → the `strings.Contains` assertion fails.

- [ ] **Step 3: Wrap the underlying errors**

```go
// CURRENT — cmd/rpc/oracle/eth/erc20_token_cache.go:63-84
	// fetch name from contract
	nameBytes, err := m.callContract(ctx, contractAddress, erc20NameFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("name")
		return types.TokenInfo{}, ErrTokenInfo
	}
	// decode name from bytes
	name := decodeString(nameBytes)
	// fetch symbol from contract
	symbolBytes, err := m.callContract(ctx, contractAddress, erc20SymbolFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("symbol")
		return types.TokenInfo{}, ErrTokenInfo
	}
	// decode symbol from bytes
	symbol := decodeString(symbolBytes)
	// fetch decimals from contract
	decimalsBytes, err := m.callContract(ctx, contractAddress, erc20DecimalsFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("decimals")
		return types.TokenInfo{}, ErrTokenInfo
	}

// NEW — cmd/rpc/oracle/eth/erc20_token_cache.go:63-84
	// fetch name from contract
	nameBytes, err := m.callContract(ctx, contractAddress, erc20NameFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("name")
		return types.TokenInfo{}, fmt.Errorf("%w: name: %v", ErrTokenInfo, err)
	}
	// decode name from bytes
	name := decodeString(nameBytes)
	// fetch symbol from contract
	symbolBytes, err := m.callContract(ctx, contractAddress, erc20SymbolFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("symbol")
		return types.TokenInfo{}, fmt.Errorf("%w: symbol: %v", ErrTokenInfo, err)
	}
	// decode symbol from bytes
	symbol := decodeString(symbolBytes)
	// fetch decimals from contract
	decimalsBytes, err := m.callContract(ctx, contractAddress, erc20DecimalsFunction)
	if err != nil {
		m.metrics.IncrementEthTokenInfoFetchError("decimals")
		return types.TokenInfo{}, fmt.Errorf("%w: decimals: %v", ErrTokenInfo, err)
	}
```

Add `"fmt"` to the file's imports:

```go
// CURRENT — cmd/rpc/oracle/eth/erc20_token_cache.go:3-14
import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// NEW — cmd/rpc/oracle/eth/erc20_token_cache.go:3-15
import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)
```

- [ ] **Step 4: Run tests**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run 'TokenInfo'`
Expected: PASS — both the new wrap test and the existing `TestERC20TokenCache_TokenInfo` (which only checks `err != nil` for error cases, still satisfied).

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/oracle/eth/erc20_token_cache.go cmd/rpc/oracle/eth/erc20_token_cache_test.go
git commit -m "fix(oracle/eth): wrap underlying cause in ErrTokenInfo"
```

---

## Task 7: Test the core order-detection logic (SHOULD)

`parseDataForOrders` (transaction.go:78-181) — the heart of the package (lock/close order extraction) — has **no active test**; the covering test was commented out at block_provider_test.go:341-508. Add a focused table-driven test against the current API, plus boundary tests for `decodeString`.

**Design decision:** Test `parseDataForOrders` directly through `NewTransaction` + a controllable `OrderValidator`, not through the whole `processBlocks` path. The existing `mockOrderValidator` always returns nil; add a configurable validator so both "valid order" and "not an order" branches are covered.

**Files:**
- Test: `cmd/rpc/oracle/eth/transaction_test.go`

- [ ] **Step 1: Add a configurable order validator mock**

Append to `transaction_test.go`:

```go
// configurableValidator returns validationErr for LockOrderType and CloseOrderType
// as configured, letting tests drive both the "valid order" and "not an order" paths.
type configurableValidator struct {
	lockErr  error
	closeErr error
}

func (c *configurableValidator) ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error {
	if orderType == types.LockOrderType {
		return c.lockErr
	}
	return c.closeErr
}
```

Add `"github.com/canopy-network/canopy/cmd/rpc/oracle/types"` to the transaction_test.go imports:

```go
// CURRENT — cmd/rpc/oracle/eth/transaction_test.go:3-9
import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// NEW — cmd/rpc/oracle/eth/transaction_test.go:3-10
import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/ethereum/go-ethereum/common"
)
```

- [ ] **Step 2: Write the failing test for parseDataForOrders**

The lock/close order JSON must be real enough for `order.UnmarshalJSON` to succeed after validation passes. Use minimal valid JSON that `lib.LockOrder`/`lib.CloseOrder` unmarshal accepts.

First discover the exact accepted JSON shape:

Run: `GOTOOLCHAIN=go1.24.0 grep -rn "func (x \*LockOrder) UnmarshalJSON\|func (x \*CloseOrder) UnmarshalJSON\|jsonLockOrder\|jsonCloseOrder" lib/`
Expected: locates the JSON struct tags for lock/close orders so the test payload matches. Use the discovered field names in the payloads below (the `{...}` placeholders MUST be replaced with real fields before writing the test — do not leave them as-is).

```go
func TestTransaction_parseDataForOrders(t *testing.T) {
	self := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	// REPLACE with the real minimal JSON accepted by lib.LockOrder/lib.CloseOrder
	// (discovered in Step 2's grep). Example placeholders:
	lockJSON := []byte(`{ ...valid lock order fields... }`)
	closeJSON := []byte(`{ ...valid close order fields... }`)

	tests := []struct {
		name         string
		to           common.Address
		data         []byte
		validator    OrderValidator
		wantOrder    bool
		wantIsERC20  bool
	}{
		{
			name:      "self-sent lock order in raw data",
			to:        self, // NewTransaction sets from==to only when signer matches; see note below
			data:      lockJSON,
			validator: &configurableValidator{lockErr: nil},
			wantOrder: true,
		},
		{
			name:        "empty data -> no order",
			to:          other,
			data:        []byte{},
			validator:   &mockOrderValidator{},
			wantOrder:   false,
			wantIsERC20: false,
		},
		{
			name:        "oversized data -> no order",
			to:          other,
			data:        make([]byte, maxTransactionDataSize+1),
			validator:   &mockOrderValidator{},
			wantOrder:   false,
			wantIsERC20: false,
		},
		{
			name:        "erc20 close order (positive amount)",
			to:          other,
			data:        createERC20TransferData(other.Hex(), big.NewInt(1000000), closeJSON),
			validator:   &configurableValidator{closeErr: nil},
			wantOrder:   true,
			wantIsERC20: true,
		},
		{
			name:        "erc20 transfer, not an order",
			to:          other,
			data:        createERC20TransferData(other.Hex(), big.NewInt(1000000), []byte("not json")),
			validator:   &configurableValidator{closeErr: ErrInvalidTransactionData},
			wantOrder:   false,
			wantIsERC20: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ethTx := createTransaction(tt.to, tt.data)
			tx, err := NewTransaction(ethTx, 0) // chainId 0 matches createTransaction's EIP155Signer(0)
			if err != nil {
				t.Fatalf("NewTransaction: %v", err)
			}
			_ = tx.parseDataForOrders(tt.validator)
			if (tx.order != nil) != tt.wantOrder {
				t.Errorf("order presence = %v, want %v", tx.order != nil, tt.wantOrder)
			}
			if tx.isERC20 != tt.wantIsERC20 {
				t.Errorf("isERC20 = %v, want %v", tx.isERC20, tt.wantIsERC20)
			}
		})
	}
}
```

**Note for the implementer:** the "self-sent" case requires `t.To() == t.From()`. `From()` is derived from the signature (`createTransaction` generates a random key), so `to` will NOT equal the random `from`. To exercise the self-sent branch deterministically, set the fields directly after construction instead of relying on address match:

```go
			// for the self-sent case only:
			tx.to = tx.from
```

Restructure the self-sent test case to apply this override (e.g. add a `selfSend bool` field to the table and, when true, do `tx.to = tx.from` before calling `parseDataForOrders`). Prefer this over trying to make signer-derived addresses collide.

- [ ] **Step 3: Run test to verify it fails, then iterate on JSON payloads**

Run: `GOTOOLCHAIN=go1.24.0 go test ./cmd/rpc/oracle/eth/ -run TestTransaction_parseDataForOrders -v`
Expected: initially FAILS on the order cases until `lockJSON`/`closeJSON` are real payloads that `UnmarshalJSON` accepts. Iterate the payloads (from Step 2's grep) until the `wantOrder: true` cases pass.

- [ ] **Step 4: Add decodeString boundary tests**

Append to `transaction_test.go` (these hit the overflow guards at erc20_token_cache.go:140, 144, 149):

```go
func TestDecodeString(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "too short", data: make([]byte, 10), want: ""},
		{name: "offset out of range", data: func() []byte {
			b := make([]byte, 64)
			b[31] = 0xff // offset = 255, beyond len
			return b
		}(), want: ""},
		{name: "valid short string", data: func() []byte {
			b := make([]byte, 96)
			b[31] = 0x20 // offset 32
			b[63] = 0x03 // length 3
			copy(b[64:], []byte("abc"))
			return b
		}(), want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeString(tt.data); got != tt.want {
				t.Errorf("decodeString = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 5: Run all package tests with race**

Run: `GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/rpc/oracle/eth/transaction_test.go
git commit -m "test(oracle/eth): cover parseDataForOrders and decodeString"
```

---

## Task 8: Delete dead code — duplicate TokenInfo, unused errors, commented blocks (SHOULD + CAN)

`known_tokens.go` defines `eth.TokenInfo` + `KnownTokens` + `USDCMainnet`/`USDTMainnet` that are **unused outside the package** (verified: no references under `cmd/` or `lib/` except the eth package's own tests) and duplicate `types.TokenInfo` — a bounded-context smell. Several `error.go` sentinels are unused, and there is a large commented-out test block.

**Design decision:** `USDTMainnet.Hex()` is referenced in `transaction_test.go:155` (`TestParseERC20Transfer_USDT`). So do **not** delete `USDTMainnet`. Delete `eth.TokenInfo`, `KnownTokens`, and `USDCMainnet` (unreferenced). Verify each deletion with a grep before removing.

**Files:**
- Modify: `cmd/rpc/oracle/eth/known_tokens.go`
- Modify: `cmd/rpc/oracle/eth/error.go`
- Modify: `cmd/rpc/oracle/eth/block_provider_test.go` (remove commented block)

- [ ] **Step 1: Confirm what is actually referenced**

Run: `GOTOOLCHAIN=go1.24.0 grep -rn "KnownTokens\|eth\.TokenInfo\|USDCMainnet\|USDTMainnet\b" cmd/ lib/`
Expected: `USDTMainnet` referenced in `transaction_test.go`; `KnownTokens`, `USDCMainnet`, and `eth.TokenInfo` (the local struct) referenced **nowhere** except their own definitions. If any of these three show a real usage, keep them and note it.

- [ ] **Step 2: Trim known_tokens.go to only what's used**

```go
// CURRENT — cmd/rpc/oracle/eth/known_tokens.go (entire file)
package eth

import "github.com/ethereum/go-ethereum/common"

// Known ERC20 token contracts on Ethereum mainnet
// ... (USDCMainnet, USDTMainnet, TokenInfo struct, KnownTokens map)

// NEW — cmd/rpc/oracle/eth/known_tokens.go (entire file)
package eth

import "github.com/ethereum/go-ethereum/common"

// USDTMainnet is the USDT token contract on Ethereum mainnet (6 decimals).
// USDT is a non-standard ERC20 (transfer() returns void), but the oracle observes
// transactions passively and validates via receipt status, so this does not matter.
// Retained as a reference address used by tests.
var USDTMainnet = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
```

- [ ] **Step 3: Confirm unused error sentinels, then remove them**

Run: `GOTOOLCHAIN=go1.24.0 grep -rEn "ErrInvalidKey|ErrInvalidPrivateKey|ErrTransactionFailed|ErrGasPriceEstimation|ErrNonceRetrieval|ErrGasEstimation|ErrTransactionSigning|ErrTransactionSending|ErrMaxRetries" cmd/ lib/`
Expected: no references (all defined-but-unused). For any that ARE referenced, keep them. Remove only the confirmed-unused ones:

```go
// CURRENT — cmd/rpc/oracle/eth/error.go:8-25
var (
	ErrInvalidKey             = errors.New("invalid private key")
	ErrInvalidTransactionData = errors.New("invalid transaction data")
	ErrNotERC20Transfer       = errors.New("transaction is not an erc20 transfer")
	ErrContractNotFound       = errors.New("contract address not found")
	ErrInvalidPrivateKey      = errors.New("invalid private key")
	ErrTransactionFailed      = errors.New("transaction failed")
	ErrGasPriceEstimation     = errors.New("failed to estimate gas price")
	ErrNonceRetrieval         = errors.New("failed to retrieve nonce")
	ErrGasEstimation          = errors.New("failed to estimate gas")
	ErrTransactionSigning     = errors.New("failed to sign transaction")
	ErrTransactionSending     = errors.New("failed to send transaction")
	ErrNilTransaction         = errors.New("transaction is nil")
	ErrMaxRetries             = errors.New("maximum retries reached")
	ErrTransactionReceipt     = errors.New("failed to get transaction receipt")
	ErrTokenInfo              = errors.New("failed to get token info")
	ErrSourceHeight           = errors.New("ethereum block height lower than expected")
)

// NEW — cmd/rpc/oracle/eth/error.go:8-16 (keep only the referenced sentinels)
var (
	ErrInvalidTransactionData = errors.New("invalid transaction data")
	ErrNotERC20Transfer       = errors.New("transaction is not an erc20 transfer")
	ErrContractNotFound       = errors.New("contract address not found")
	ErrNilTransaction         = errors.New("transaction is nil")
	ErrTransactionReceipt     = errors.New("failed to get transaction receipt")
	ErrTokenInfo              = errors.New("failed to get token info")
	ErrSourceHeight           = errors.New("ethereum block height lower than expected")
)
```

(Keep `InvalidAddressError` and its `Error()` method below — used in transaction.go:65.)

- [ ] **Step 4: Delete the commented-out test block**

Remove the entire commented block `cmd/rpc/oracle/eth/block_provider_test.go:341-508` (the `// func TestEthBlockProvider_checkTransfer(...)` through its closing `// }`). Also remove the now-orphaned `transactionConfig` type and `createTestTransaction`/`uint64Ptr` helpers (511-537) **only if** grep shows they are unused after Task 7:

Run: `GOTOOLCHAIN=go1.24.0 grep -rn "createTestTransaction\|transactionConfig\|uint64Ptr" cmd/rpc/oracle/eth/`
Expected: if only their own definitions match, delete them; if Task 7's test used any, keep those.

- [ ] **Step 5: Build + test**

Run: `GOTOOLCHAIN=go1.24.0 go build ./cmd/rpc/oracle/eth/ && GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/`
Expected: builds cleanly (no "declared and not used"), tests PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/rpc/oracle/eth/known_tokens.go cmd/rpc/oracle/eth/error.go cmd/rpc/oracle/eth/block_provider_test.go
git commit -m "chore(oracle/eth): remove dead TokenInfo dup, unused errors, commented tests"
```

---

## Task 9: Polish — interface doc, log levels (CAN)

Minor low-risk cleanups: add the missing doc comment on the exported `OrderValidator` interface, and downgrade the expected on-chain-failure log from Error to Warn so it doesn't trip alerting.

**Files:**
- Modify: `cmd/rpc/oracle/eth/block_provider.go:50, 572`

- [ ] **Step 1: Add OrderValidator doc comment**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:50-52
type OrderValidator interface {
	ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error
}

// NEW — cmd/rpc/oracle/eth/block_provider.go:50-53
// OrderValidator validates canopy order JSON found embedded in ethereum transaction data.
type OrderValidator interface {
	ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error
}
```

- [ ] **Step 2: Downgrade expected-failure log level**

```go
// CURRENT — cmd/rpc/oracle/eth/block_provider.go:572
	p.logger.Errorf("[ETH-TX] tx %s ERC20 transfer failed on-chain, ignoring", txHashStr)

// NEW — cmd/rpc/oracle/eth/block_provider.go:572
	p.logger.Warnf("[ETH-TX] tx %s ERC20 transfer failed on-chain, ignoring", txHashStr)
```

- [ ] **Step 3: Build + test + vet**

Run: `GOTOOLCHAIN=go1.24.0 go vet ./cmd/rpc/oracle/eth/ && GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/`
Expected: vet clean (the earlier `cockroachdb/swiss` vet failure is resolved by the pinned toolchain), tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/rpc/oracle/eth/block_provider.go
git commit -m "docs(oracle/eth): document OrderValidator; warn (not error) on expected on-chain failure"
```

---

## Final Verification

- [ ] **Full package gate**

Run: `GOTOOLCHAIN=go1.24.0 gofmt -l cmd/rpc/oracle/eth/ && GOTOOLCHAIN=go1.24.0 go vet ./cmd/rpc/oracle/eth/ && GOTOOLCHAIN=go1.24.0 go test -race ./cmd/rpc/oracle/eth/`
Expected: `gofmt -l` prints nothing, vet clean, tests PASS.

- [ ] **Confirm all review findings addressed**

Cross-check against `ai/reviews/eth-package-review-20260704-025700.md`:
- MUST: data race (Task 1), fmt.Println (Task 2), blocking send (Task 3) ✓
- SHOULD: constructor Fatal (Task 4), 9-arg metrics (Task 5), error wrapping (Task 6), test gaps + no -race (Task 7 + toolchain), TokenInfo dup (Task 8) ✓
- CAN: unused errors + commented code (Task 8), OrderValidator doc + log level (Task 9) ✓
- Residual (intentionally not changed, documented here): `logAsciiBytes` debug-level volume on every non-order tx is left as-is (debug level, low risk); `parseDataForOrders` treating validator failure as "not an order" is intentional design — behavior unchanged, only the discarded TokenInfo cause was wrapped.

---

## Notes

- **Component tracking:** Per project CLAUDE.md, oracle work should be tracked under a Canopy component (there is no dedicated `eth-oracle` component yet — `solana-bridge` is the closest bridge/oracle component). Load the appropriate component and log these fixes as completed tasks when the branch merges.
- The `GOTOOLCHAIN=go1.24.0` requirement is an environment quirk of this machine (go1.26 installed) — CI pinned to go1.24 will not need it. If CI also runs go1.26, raise the `cockroachdb/swiss` incompatibility separately; it is out of scope for this review.
