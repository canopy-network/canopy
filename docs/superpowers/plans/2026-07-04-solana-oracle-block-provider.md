# Solana Oracle Block Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Solana as a second source chain the Canopy oracle can witness, alongside Ethereum, by implementing a new `sol` package that satisfies the existing chain-agnostic `BlockProvider`/`BlockI`/`TransactionI` interfaces.

**Architecture:** The core oracle package (`oracle.go`) already depends on interfaces, not the concrete `eth` provider. This plan (1) extracts the two remaining Ethereum-specific checks out of `oracle.go` into a new `TransactionI.MatchesOrderDestination` interface method, (2) renames `Eth`-prefixed metric helpers to chain-generic names, (3) adds a `SolBlockProviderConfig`, (4) builds a polling-based `sol` provider (Solana has no usable WebSocket block subscription on managed RPC), and (5) wires provider selection into the CLI. Solana order data travels as a top-level Memo program instruction beside a top-level transfer instruction; the provider scans top-level instructions by program ID.

**Tech Stack:** Go, `github.com/gagliardetto/solana-go` (RPC client + ATA derivation + Memo/System/Token program IDs), existing `github.com/canopy-network/canopy/lib` metrics/config/logging.

**Reference docs (read before starting):**
- `docs/SOLANA_ORACLE_DESIGN.md` — the approved design this plan implements.
- `docs/SOLANA_ORACLE_REQUIREMENTS.md` — client-side constraints (transfer + memo must be top-level, no ALTs) that make single-call parsing valid.
- `cmd/rpc/oracle/eth/` — the reference implementation. Mirror its layout and test style; do NOT copy its WebSocket/receipt/token-cache internals.

**Working directory for all paths:** `~/canopy/canopy/oracle` (this is the `github.com/canopy-network/canopy` module).

**Commit discipline:** one commit per task (or per logical step where noted). Do NOT add `Co-Authored-By: Claude` lines — this repo's convention forbids them.

**Build/test commands used throughout:**
- Build one package: `go build ./cmd/rpc/oracle/...`
- Test one package: `go test ./cmd/rpc/oracle/eth/... -run TestName -v`
- Full oracle tests: `go test ./cmd/rpc/oracle/...`

---

## File Structure

**Modified (core, chain-agnostic — Tasks 1–7):**
- `cmd/rpc/oracle/types/types.go` — add `MatchesOrderDestination` to `TransactionI`.
- `cmd/rpc/oracle/eth/transaction.go` — implement `MatchesOrderDestination` (reproduces today's exact behavior).
- `cmd/rpc/oracle/oracle.go` — replace two EVM-specific checks in `validateCloseOrder` with one interface call.
- `cmd/rpc/oracle/oracle_test.go` — add `MatchesOrderDestination` to the `mockTransaction` test double.
- `lib/metrics.go` — rename ~27 `Eth`-prefixed metric helper methods to chain-generic names.
- `lib/config.go` — add `SolBlockProviderConfig`, embed in `Config`, add default.
- `cmd/cli/cli.go` — select provider by populated config.
- `go.mod` / `go.sum` — add `github.com/gagliardetto/solana-go`.

**Created (new `sol` package — Tasks 8–13):**
- `cmd/rpc/oracle/sol/error.go` — package sentinel errors.
- `cmd/rpc/oracle/sol/mint_cache.go` — LRU cache of SPL mint decimals, decoded from raw Mint account bytes.
- `cmd/rpc/oracle/sol/transaction.go` — implements `types.TransactionI`; scans top-level instructions.
- `cmd/rpc/oracle/sol/block.go` — implements `types.BlockI`.
- `cmd/rpc/oracle/sol/block_provider.go` — polling loop; implements `oracle.BlockProvider`.
- Matching `_test.go` files for each.

**Dependency order:** Tasks 1–3 (interface extraction) unblock everything. Task 4 (metrics rename) is independent hygiene. Tasks 5–7 (dep + config) unblock the `sol` package. Tasks 8–13 build `sol` bottom-up (errors → mint cache → transaction → block → provider → wiring). Execute in listed order.

---

## Task 1: Add `MatchesOrderDestination` to the `TransactionI` interface

**Files:**
- Modify: `cmd/rpc/oracle/types/types.go:143-151`

This task only changes the interface contract. It will break compilation of `eth` and the oracle test double until Tasks 2–3 land — that is expected and fine within a single logical change. We commit Tasks 1+2+3 together at the end of Task 3 so the tree never stays broken across a commit boundary.

- [ ] **Step 1: Add the method to the interface**

```go
// CURRENT — cmd/rpc/oracle/types/types.go:143-151
// TransactionI interface represents a blockchain transaction
type TransactionI interface {
	Blockchain() string
	From() string
	To() string
	Hash() string
	Order() *WitnessedOrder
	TokenTransfer() TokenTransfer
}

// NEW — cmd/rpc/oracle/types/types.go
// TransactionI interface represents a blockchain transaction
type TransactionI interface {
	Blockchain() string
	From() string
	To() string
	Hash() string
	Order() *WitnessedOrder
	TokenTransfer() TokenTransfer
	// MatchesOrderDestination reports whether this transaction's transfer satisfies the
	// sell order's expected asset/contract (contractBytes) and recipient (recipientBytes),
	// using chain-specific address formatting and/or derivation. Returns (false, nil) for a
	// clean mismatch and (false, err) only when the inputs cannot be interpreted for this chain.
	// Replaces the two Ethereum-specific checks that previously lived in oracle.go's validateCloseOrder.
	MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error)
}
```

- [ ] **Step 2: Verify it fails to build (expected)**

Run: `go build ./cmd/rpc/oracle/...`
Expected: FAIL — `*eth.Transaction` and `*mockTransaction` no longer satisfy `types.TransactionI`. Proceed to Task 2.

---

## Task 2: Implement `MatchesOrderDestination` on `eth.Transaction` (behavior-preserving)

**Files:**
- Modify: `cmd/rpc/oracle/eth/transaction.go` (add method + imports)
- Test: `cmd/rpc/oracle/eth/transaction_test.go` (add regression test)

The Ethereum implementation must reproduce today's exact behavior so Ethereum witnessing is unchanged. Today `oracle.go` does two things we are moving here:
1. Asset/contract check: `common.BytesToAddress(sellOrder.Data).String() == tx.To()` (the ERC20 contract address).
2. Recipient check: the ERC20 transfer recipient (`t.erc20Recipient`, a `0x`-prefixed hex string) decoded to bytes equals `sellOrder.SellerReceiveAddress`.

Note `t.erc20Recipient` is set as `common.BytesToAddress(...).Hex()` in `parseERC20Transfer`, so it is `0x`-prefixed — we strip `0x` before `lib.StringToBytes`, exactly as `oracle.go` does today (`oracle.go:406`).

- [ ] **Step 1: Write the failing test**

```go
// ADD to cmd/rpc/oracle/eth/transaction_test.go

// TestMatchesOrderDestination verifies the eth implementation reproduces the pre-refactor
// oracle.go checks: sellOrder.Data formatted as an EVM address must equal tx.To() (the ERC20
// contract), and the ERC20 transfer recipient must equal sellOrder.SellerReceiveAddress.
func TestMatchesOrderDestination(t *testing.T) {
	// 20-byte contract address and 20-byte recipient address as raw bytes
	contractBytes := common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()
	recipientBytes := common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()

	tx := &Transaction{
		to:             common.BytesToAddress(contractBytes).String(),      // tx recipient == contract
		erc20Recipient: common.BytesToAddress(recipientBytes).Hex(),         // 0x-prefixed, as set by parseERC20Transfer
	}

	// happy path: both match
	ok, err := tx.MatchesOrderDestination(contractBytes, recipientBytes)
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}

	// contract mismatch
	otherContract := common.HexToAddress("0x9999999999999999999999999999999999999999").Bytes()
	ok, err = tx.MatchesOrderDestination(otherContract, recipientBytes)
	if err != nil || ok {
		t.Fatalf("expected contract mismatch (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}

	// recipient mismatch
	otherRecipient := common.HexToAddress("0x8888888888888888888888888888888888888888").Bytes()
	ok, err = tx.MatchesOrderDestination(contractBytes, otherRecipient)
	if err != nil || ok {
		t.Fatalf("expected recipient mismatch (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/eth/ -run TestMatchesOrderDestination -v`
Expected: FAIL — `tx.MatchesOrderDestination undefined`.

- [ ] **Step 3: Add the method and required imports**

The file's current imports are `fmt`, `math/big`, the canopy `types`/`lib`, and go-ethereum `common`/`ethtypes`. Add `bytes` and `strings`.

```go
// CURRENT — cmd/rpc/oracle/eth/transaction.go:3-11
import (
	"fmt"
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// NEW — cmd/rpc/oracle/eth/transaction.go:3-13
import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)
```

Add the method next to the other `*Transaction` methods (e.g. right after `TokenTransfer` at `transaction.go:224`):

```go
// NEW — cmd/rpc/oracle/eth/transaction.go
// MatchesOrderDestination reproduces the pre-refactor Ethereum checks from oracle.go:
// contractBytes (sell order Data) formatted as an EVM address must equal the tx recipient
// (the ERC20 contract), and the ERC20 transfer recipient must equal recipientBytes
// (sell order SellerReceiveAddress). Returns (false, err) only when the transfer recipient
// string cannot be decoded to bytes.
func (t *Transaction) MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error) {
	// asset/contract check: sell order Data formatted as an EVM address must equal tx recipient
	if common.BytesToAddress(contractBytes).String() != t.To() {
		return false, nil
	}
	// recipient check: decode the ERC20 transfer recipient (0x-prefixed hex) to bytes
	recipient, err := lib.StringToBytes(strings.TrimPrefix(t.erc20Recipient, "0x"))
	if err != nil {
		return false, err
	}
	if !bytes.Equal(recipientBytes, recipient) {
		return false, nil
	}
	return true, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rpc/oracle/eth/ -run TestMatchesOrderDestination -v`
Expected: PASS. (`go build ./cmd/rpc/oracle/eth/...` now succeeds; `oracle_test.go` still fails to build — fixed in Task 3.)

---

## Task 3: Refactor `oracle.go` `validateCloseOrder` to call the interface method

**Files:**
- Modify: `cmd/rpc/oracle/oracle.go:372-415` (replace two checks; the buyer/sender + amount checks stay)
- Modify: `cmd/rpc/oracle/oracle_test.go:227-263` (add `MatchesOrderDestination` to `mockTransaction`)

The buyer/sender check (`oracle.go:392-403`) and amount checks (`oracle.go:416-428`) are chain-agnostic and STAY. We remove only: the `sellOrder.Data` vs `tx.To()` check (`oracle.go:373-381`) and the recipient decode+compare (`oracle.go:404-415`). The `common` import may become unused in `oracle.go` after this — Step 4 handles that.

- [ ] **Step 1: Add `MatchesOrderDestination` to the `mockTransaction` test double**

The mock reproduces the eth semantics using its own fields so the existing `validateCloseOrder` table tests keep passing: `to` is the contract, `tokenTransfer.RecipientAddress` is the transfer recipient.

```go
// ADD to cmd/rpc/oracle/oracle_test.go, next to the other mockTransaction methods (after TokenTransfer at line ~263)

// MatchesOrderDestination mirrors eth.Transaction: contractBytes formatted as an EVM address
// must equal the mock's `to`, and the token transfer recipient must equal recipientBytes.
func (m *mockTransaction) MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error) {
	if common.BytesToAddress(contractBytes).String() != m.to {
		return false, nil
	}
	recipient, err := lib.StringToBytes(strings.TrimPrefix(m.tokenTransfer.RecipientAddress, "0x"))
	if err != nil {
		return false, err
	}
	if !bytes.Equal(recipientBytes, recipient) {
		return false, nil
	}
	return true, nil
}
```

Ensure `oracle_test.go` imports `bytes`, `strings`, `github.com/ethereum/go-ethereum/common`, and `github.com/canopy-network/canopy/lib`. Check the existing import block; add any missing. (The test file already uses `common` and `lib` in close-order tests, so likely only confirm.)

- [ ] **Step 2: Replace the two checks in `validateCloseOrder`**

```go
// CURRENT — cmd/rpc/oracle/oracle.go:372-381 (opening of validateCloseOrder)
func (o *Oracle) validateCloseOrder(closeOrder *lib.CloseOrder, sellOrder *lib.SellOrder, tx types.TransactionI) lib.ErrorI {
	// Order data being equal to transaction To address is Ethereum-specific validation
	// TODO move this logic into the block provider

	sellOrderDataHex := common.BytesToAddress(sellOrder.Data).String()
	if sellOrderDataHex != tx.To() {
		o.log.Warnf("[ORACLE-ORDER] close order data mismatch: sellOrderData=%s txRecipient=%s", sellOrderDataHex, tx.To())
		o.metrics.IncrementValidationFailure("close_data_mismatch")
		return ErrOrderValidation("sell order data field does not match transaction recipient")
	}
	// ensure the order ids are a match

// NEW — cmd/rpc/oracle/oracle.go (opening of validateCloseOrder)
func (o *Oracle) validateCloseOrder(closeOrder *lib.CloseOrder, sellOrder *lib.SellOrder, tx types.TransactionI) lib.ErrorI {
	// chain-specific asset + recipient validation, delegated to the block provider's transaction type
	ok, err := tx.MatchesOrderDestination(sellOrder.Data, sellOrder.SellerReceiveAddress)
	if err != nil {
		o.metrics.IncrementValidationFailure("destination_conversion_error")
		return ErrOrderValidation("error validating transaction destination")
	}
	if !ok {
		o.metrics.IncrementValidationFailure("destination_mismatch")
		return ErrOrderValidation("transaction destination does not match sell order")
	}
	// ensure the order ids are a match
```

Then delete the now-redundant recipient block. The `tokenTransfer` variable is still needed for the amount checks, so keep `tokenTransfer := tx.TokenTransfer()` but drop the recipient decode/compare:

```go
// CURRENT — cmd/rpc/oracle/oracle.go:404-415
	// convenience variable
	tokenTransfer := tx.TokenTransfer()
	recipient, err := lib.StringToBytes(strings.TrimPrefix(tokenTransfer.RecipientAddress, "0x"))
	if err != nil {
		o.metrics.IncrementValidationFailure("recipient_conversion_error")
		return ErrOrderValidation("error converting recipient address to bytes")
	}
	// verify the recipient of the transfer was the seller receive address
	if !bytes.Equal(sellOrder.SellerReceiveAddress, recipient) {
		o.metrics.IncrementValidationFailure("recipient_mismatch")
		return ErrOrderValidation("tokens not transferred to sell receive address")
	}
	// ensure transfer amount is not nil

// NEW — cmd/rpc/oracle/oracle.go
	// convenience variable
	tokenTransfer := tx.TokenTransfer()
	// ensure transfer amount is not nil
```

Note: the buyer/sender check between these two edits (`oracle.go:392-403`) is untouched — it still uses `tx.From()`, `strings`, `bytes`, and `lib.StringToBytes`, so those imports remain used in `oracle.go`.

- [ ] **Step 3: Fix imports in `oracle.go`**

The `common` (`github.com/ethereum/go-ethereum/common`) import was only used by the deleted `common.BytesToAddress(sellOrder.Data)` call. If nothing else in `oracle.go` uses `common`, remove that import line. Verify with:

Run: `grep -n "common\." cmd/rpc/oracle/oracle.go`
If no matches: remove the `common` import. If matches remain: leave it.

- [ ] **Step 4: Build and run the full oracle test suite**

Run: `go build ./cmd/rpc/oracle/... && go test ./cmd/rpc/oracle/...`
Expected: PASS. The existing `validateCloseOrder` table tests exercise both match and mismatch paths through the new interface method (via `mockTransaction`), and the eth regression test from Task 2 confirms unchanged Ethereum behavior.

- [ ] **Step 5: Commit Tasks 1–3 together**

```bash
cd ~/canopy/canopy/oracle
git add cmd/rpc/oracle/types/types.go cmd/rpc/oracle/eth/transaction.go cmd/rpc/oracle/eth/transaction_test.go cmd/rpc/oracle/oracle.go cmd/rpc/oracle/oracle_test.go
git commit -m "refactor(oracle): extract chain-specific destination check into TransactionI.MatchesOrderDestination"
```

---

## Task 4: Rename `Eth`-prefixed metric helpers to chain-generic names

**Files:**
- Modify: `lib/metrics.go:1426+` (the `// ========== Eth Block Provider Metrics Helper Functions ==========` section, ~27 methods)
- Modify: `cmd/rpc/oracle/eth/block_provider.go` (all call sites)
- Modify: `cmd/rpc/oracle/eth/erc20_token_cache.go` (any metric call sites)
- Modify: any `eth/*_test.go` referencing these method names

**Decision:** Do the rename (design calls it optional hygiene). Rationale: the `sol` provider will call these same helpers; leaving them `Eth`-prefixed forces `sol` to call `IncrementEthReorgDetected`, which is misleading, and a third chain would need a third copy. The underlying Prometheus gauges/counters are already generically named (e.g. `SetEthConnectionState` wraps `m.ConnectionState`), so only the Go method names change — no metric-name or dashboard impact. Only one chain provider runs per process, so shared gauges have no collision risk.

Rename mapping (method name only; drop the `Eth` token, keep everything else): e.g. `SetEthConnectionState`→`SetConnectionState`, `SetEthSyncStatus`→`SetSyncStatus`, `SetEthBlockHeightLag`→`SetBlockHeightLag`, `SetEthChainHeadHeight`→`SetChainHeadHeight`, `IncrementEthReorgDetected`→`IncrementReorgDetected`, `IncrementEthRPCConnectionAttempt`→`IncrementRPCConnectionAttempt`, `IncrementEthBlockProcessingTimeout`→`IncrementBlockProcessingTimeout`, etc.

**Exceptions — leave these alone:** methods whose *underlying field* also carries the `Eth` name and is genuinely Eth-specific, i.e. `SetEthLastProcessedHeight` (wraps `m.EthLastProcessedHeight`) and `SetEthSafeHeight` (wraps `m.EthSafeHeight`), and the `UpdateEthBlockProvider(lib.EthBlockProviderMetricUpdate{...})` aggregate + its struct. Renaming those touches Prometheus metric registration and is out of scope. The `sol` provider will simply not call the two `Eth*Height` helpers (it sets its own via the generic ones where they exist) — acceptable, since per-chain last/safe height is display-only.

- [ ] **Step 1: Rename the method declarations in `lib/metrics.go`**

For each helper in the `Eth Block Provider Metrics Helper Functions` section EXCEPT the two `*Height` exceptions and the `UpdateEthBlockProvider` aggregate, delete `Eth` from the method name. Example:

```go
// CURRENT — lib/metrics.go
func (m *Metrics) SetEthConnectionState(state int) {
	if m == nil {
		return
	}
	m.ConnectionState.Set(float64(state))
}

// NEW — lib/metrics.go
func (m *Metrics) SetConnectionState(state int) {
	if m == nil {
		return
	}
	m.ConnectionState.Set(float64(state))
}
```

- [ ] **Step 2: Update all call sites in the `eth` package**

Do a mechanical rename across the eth provider. Preview the impact first:

Run: `grep -rn "\.SetEth\|\.IncrementEth\|\.RecordEth" cmd/rpc/oracle/eth/ | grep -v "SetEthLastProcessedHeight\|SetEthSafeHeight\|UpdateEthBlockProvider"`

For each match, drop the `Eth` token from the method name (matching the Step 1 renames). Do NOT touch `SetEthLastProcessedHeight`, `SetEthSafeHeight`, or `UpdateEthBlockProvider` call sites.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./cmd/rpc/oracle/... ./lib/...`
Expected: PASS. If the build reports an undefined `m.SetEth*`/`m.IncrementEth*` method, either a declaration was missed in Step 1 or a call site was missed in Step 2 — reconcile the two lists.

- [ ] **Step 4: Commit**

```bash
git add lib/metrics.go cmd/rpc/oracle/eth/
git commit -m "refactor(metrics): rename Eth-prefixed block-provider metric helpers to chain-generic names"
```

---

## Task 5: Add the `solana-go` dependency and record its actual API

**Files:**
- Modify: `go.mod`, `go.sum`

`github.com/gagliardetto/solana-go` is alpha ("APIs subject to change"). Later tasks call specific symbols; this task pins the version and records the *actual* signatures so later tasks use verified names, not assumptions. A `solana-foundation/solana-go` fork exists — this plan uses `gagliardetto`, the current community standard. Do not switch forks without re-verifying every call site.

- [ ] **Step 1: Add the dependency**

```bash
cd ~/canopy/canopy/oracle
go get github.com/gagliardetto/solana-go@latest
go mod tidy
```

- [ ] **Step 2: Record the actual signatures the plan depends on**

Run each and read the output; if a signature differs from what later tasks assume, note the difference and adapt the later task's code to the real signature.

```bash
go doc github.com/gagliardetto/solana-go/rpc Client.GetSlot
go doc github.com/gagliardetto/solana-go/rpc Client.GetBlockWithOpts
go doc github.com/gagliardetto/solana-go/rpc GetBlockOpts
go doc github.com/gagliardetto/solana-go/rpc Client.GetAccountInfo
go doc github.com/gagliardetto/solana-go/rpc GetBlockResult
go doc github.com/gagliardetto/solana-go/rpc TransactionWithMeta
go doc github.com/gagliardetto/solana-go FindAssociatedTokenAddress
go doc github.com/gagliardetto/solana-go/programs/memo ProgramID
go doc github.com/gagliardetto/solana-go CompiledInstruction
go doc github.com/gagliardetto/solana-go Message
```

Expected anchors (adapt code in later tasks if reality differs):
- `rpc.New(endpoint string) *rpc.Client`
- `(*rpc.Client).GetSlot(ctx, commitment rpc.CommitmentType) (uint64, error)`
- `(*rpc.Client).GetBlockWithOpts(ctx, slot uint64, opts *rpc.GetBlockOpts) (*rpc.GetBlockResult, error)`
- `rpc.GetBlockOpts{Commitment, MaxSupportedTransactionVersion *uint64, TransactionDetails, Rewards *bool}`
- `(*rpc.Client).GetAccountInfo(ctx, account solana.PublicKey) (*rpc.GetAccountInfoResult, error)`; bytes via `res.Value.Data.GetBinary()`
- `solana.FindAssociatedTokenAddress(wallet, mint solana.PublicKey) (solana.PublicKey, uint8, error)`
- Memo program id constant in `programs/memo` (or `solana.MemoProgramID`)
- `solana.CompiledInstruction{ProgramIDIndex uint16, Accounts []uint16, Data solana.Base58}`
- `(*solana.Message).Program(idx uint16) solana.PublicKey` and `(*solana.Message).Account(idx uint16) solana.PublicKey` (or `AccountMetaList`) — confirm the exact accessor names.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build(oracle): add github.com/gagliardetto/solana-go dependency"
```

---

## Task 6: Add `SolBlockProviderConfig` to `lib/config.go`

**Files:**
- Modify: `lib/config.go` (new type + default + embed in `Config`)

Mirror `EthBlockProviderConfig` (`config.go:361-378`). Omit `StartupBlockDepth` (Solana always polls at `finalized` commitment) and the WS URL (polling only). Add `PollIntervalMs`.

- [ ] **Step 1: Add the config type and default**

Place directly after `DefaultEthBlockProviderConfig` (after `config.go:378`):

```go
// NEW — lib/config.go
// SolBlockProviderConfig configures the Solana block provider. Solana polls at `finalized`
// commitment, so there is no WebSocket URL and no startup block-depth heuristic.
type SolBlockProviderConfig struct {
	NodeUrl        string `json:"solNodeUrl"`        // solana http rpc node url (e.g. Alchemy)
	PollIntervalMs int    `json:"solPollIntervalMs"` // poll interval in milliseconds for getSlot(finalized)
	RetryDelay     int    `json:"solRetryDelay"`     // retry delay in seconds for transient rpc failures
}

// DefaultSolBlockProviderConfig returns the default solana block provider configuration
func DefaultSolBlockProviderConfig() SolBlockProviderConfig {
	return SolBlockProviderConfig{
		NodeUrl:        "http://localhost:8899", // solana-test-validator default
		PollIntervalMs: 400,                     // ~1 slot; Solana slot time is ~400ms
		RetryDelay:     5,                       // 5s backoff on transient rpc errors
	}
}
```

- [ ] **Step 2: Embed it in `Config` and wire the default**

```go
// CURRENT — lib/config.go:47-48
	EthBlockProviderConfig `json:"ethBlockProviderConfig"` // ethereum block provider configuration
	OracleConfig           `json:"oracleConfig"`           // oracle configuration

// NEW — lib/config.go
	EthBlockProviderConfig `json:"ethBlockProviderConfig"` // ethereum block provider configuration
	SolBlockProviderConfig `json:"solBlockProviderConfig"` // solana block provider configuration
	OracleConfig           `json:"oracleConfig"`           // oracle configuration
```

```go
// CURRENT — lib/config.go:63-64
		EthBlockProviderConfig: DefaultEthBlockProviderConfig(),
		OracleConfig:           DefaultOracleConfig(),

// NEW — lib/config.go
		EthBlockProviderConfig: DefaultEthBlockProviderConfig(),
		SolBlockProviderConfig: DefaultSolBlockProviderConfig(),
		OracleConfig:           DefaultOracleConfig(),
```

Note: `EthBlockProviderConfig` and `SolBlockProviderConfig` are both embedded structs with no field-name collisions (all fields are chain-prefixed in JSON and uniquely named in Go), so embedding both is safe.

- [ ] **Step 3: Build and test**

Run: `go build ./lib/... && go test ./lib/ -run Config`
Expected: PASS (or no matching tests — build success is the gate).

- [ ] **Step 4: Commit**

```bash
git add lib/config.go
git commit -m "feat(config): add SolBlockProviderConfig"
```

---

## Task 7: Create `sol/error.go`

**Files:**
- Create: `cmd/rpc/oracle/sol/error.go`

- [ ] **Step 1: Write the error file**

```go
// NEW — cmd/rpc/oracle/sol/error.go
package sol

import "errors"

var (
	// ErrSlotSkipped indicates getBlock reported the slot was skipped (no block produced).
	// Solana returns this via jsonrpc codes -32007 (skipped/snapshot) and -32009 (missing in long-term storage).
	ErrSlotSkipped = errors.New("solana slot was skipped")
	// ErrNilTransaction indicates a nil transaction was passed to NewTransaction.
	ErrNilTransaction = errors.New("transaction is nil")
	// ErrMintAccountTooSmall indicates a Mint account's data was shorter than the decimals offset.
	ErrMintAccountTooSmall = errors.New("mint account data too small to decode decimals")
	// ErrSourceSlot indicates the observed finalized slot went backwards below nextSlot.
	ErrSourceSlot = errors.New("solana finalized slot lower than expected")
)
```

- [ ] **Step 2: Build**

Run: `go build ./cmd/rpc/oracle/sol/...`
Expected: PASS (package compiles with only errors defined).

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/error.go
git commit -m "feat(sol): add sol package error definitions"
```

---

## Task 8: Implement `sol/mint_cache.go` — SPL decimals decode + LRU

**Files:**
- Create: `cmd/rpc/oracle/sol/mint_cache.go`
- Test: `cmd/rpc/oracle/sol/mint_cache_test.go`

An SPL Mint account's `decimals` is a `u8` at fixed byte offset **44** (layout: mintAuthority option+key 0–35, supply 36–43, **decimals 44**, ...). Decode it directly from `getAccountInfo` raw bytes — no Metaplex, no contract call. `Name`/`Symbol` are left blank (display-only, never used in validation). Cache decimals per mint with a simple bounded LRU.

**Decision:** decode by fixed offset rather than using solana-go's `token.Mint` deserializer — the offset is verified (design "Verification" table) and avoids depending on a struct layout that may drift in the alpha SDK. One `const mintDecimalsOffset = 44`.

- [ ] **Step 1: Write the failing test**

```go
// NEW — cmd/rpc/oracle/sol/mint_cache_test.go
package sol

import "testing"

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/sol/ -run 'TestDecodeMintDecimals|TestMintCache' -v`
Expected: FAIL — `decodeMintDecimals`, `newMintCache`, `mintDecimalsOffset` undefined.

- [ ] **Step 3: Implement the mint cache**

```go
// NEW — cmd/rpc/oracle/sol/mint_cache.go
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
```

**Verification note:** confirm `res.Value.Data.GetBinary()` is the correct accessor from Task 5's `go doc rpc GetAccountInfoResult` output. If the SDK exposes bytes differently (e.g. `res.GetBinary()`), adapt this one line.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rpc/oracle/sol/ -run 'TestDecodeMintDecimals|TestMintCache' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/oracle/sol/mint_cache.go cmd/rpc/oracle/sol/mint_cache_test.go
git commit -m "feat(sol): add mint decimals decode and LRU cache"
```

---

## Task 9: Implement `sol/transaction.go` — instruction scanning + `TransactionI`

**Files:**
- Create: `cmd/rpc/oracle/sol/transaction.go`
- Test: `cmd/rpc/oracle/sol/transaction_test.go`

This is the core parsing. A witnessed transaction carries a top-level **Memo** instruction (order JSON) and, for close orders, a top-level **transfer** instruction (System Program for native SOL, SPL Token Program for tokens). Scan top-level instructions by program ID (not position — ComputeBudget instructions may be interspaced). Lock order = Memo alone. Close order = Memo + transfer.

**Design decisions (pre-made):**
- `OrderValidator` interface is redeclared locally in `sol` (mirroring `eth`'s local `OrderValidator` at `eth/transaction.go`'s consumer) — same method `ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error`.
- The Solana transaction wrapper is built from already-fetched, decoded data (`*solana.Transaction` + `*rpc.TransactionMeta`), so parsing is pure and unit-testable without RPC. The provider (Task 11) does the fetching and passes decoded instructions in.
- To keep parsing testable without constructing full `*solana.Transaction` values, define a small internal `instruction` struct `{programID solana.PublicKey; accounts []solana.PublicKey; data []byte}` and a `resolveInstructions(*solana.Transaction) ([]instruction, error)` adapter. Tests build `[]instruction` directly; the provider calls `resolveInstructions`.
- `blockchain` identifier constant is `"solana"`.
- Amount + mint for SPL transfers are parsed from the SPL Token `Transfer`/`TransferChecked` instruction data + accounts. For native SOL, from the System `Transfer` instruction (lamports). See Step 3 for the exact byte layout and the pre-made decision on which instruction variants to support.

- [ ] **Step 1: Write the failing test**

```go
// NEW — cmd/rpc/oracle/sol/transaction_test.go
package sol

import (
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/gagliardetto/solana-go"
)

// fakeValidator lets tests drive the "is this valid order JSON" decision per order type.
type fakeValidator struct {
	lockErr  error
	closeErr error
}

func (f *fakeValidator) ValidateOrderJsonBytes(b []byte, ot types.OrderType) error {
	if ot == types.LockOrderType {
		return f.lockErr
	}
	return f.closeErr
}

// A minimal valid lock order JSON is what the real validator would accept; here the fake
// accepts everything, and we assert the parser routes a Memo-only tx to a lock order.
func TestParse_MemoOnly_IsLockOrder(t *testing.T) {
	// NOTE: lockOrderJSON must unmarshal into lib.LockOrder. Use a real serialized lock order
	// fixture (see helper below) so UnmarshalJSON succeeds.
	instrs := []instruction{
		{programID: computeBudgetProgramID, data: []byte{0x02, 0, 0, 0}}, // ignored
		{programID: memoProgramID, data: lockOrderJSONFixture(t)},
	}
	tx := newTestTransaction("sig-lock", instrs, nil)
	err := tx.parseInstructions(&fakeValidator{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tx.Order() == nil || tx.Order().LockOrder == nil {
		t.Fatal("expected a lock order to be parsed from memo-only transaction")
	}
	if tx.Order().CloseOrder != nil {
		t.Fatal("memo-only tx must not produce a close order")
	}
}

func TestParse_NoMemo_IsNoOp(t *testing.T) {
	instrs := []instruction{
		{programID: computeBudgetProgramID, data: []byte{0x02, 0, 0, 0}},
		{programID: solana.SystemProgramID, data: []byte{}},
	}
	tx := newTestTransaction("sig-noop", instrs, nil)
	if err := tx.parseInstructions(&fakeValidator{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tx.Order() != nil {
		t.Fatal("transaction with no memo must not produce an order")
	}
}
```

Add fixture helpers to the test file. `lockOrderJSONFixture` must produce bytes that `(*lib.LockOrder).UnmarshalJSON` accepts — build a `lib.LockOrder`, marshal it, return the bytes:

```go
// ADD to cmd/rpc/oracle/sol/transaction_test.go
import (
	"github.com/canopy-network/canopy/lib"
)

func lockOrderJSONFixture(t *testing.T) []byte {
	t.Helper()
	lo := &lib.LockOrder{
		OrderId:             []byte{0x01, 0x02, 0x03},
		ChainId:             1,
		BuyerReceiveAddress: []byte{0xaa},
		BuyerSendAddress:    []byte{0xbb},
		BuyerChainDeadline:  100,
	}
	b, err := lo.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal lock order fixture: %v", err)
	}
	return b
}
```

**Verify before writing:** run `go doc github.com/canopy-network/canopy/lib LockOrder` to confirm field names (`OrderId`, `ChainId`, `BuyerReceiveAddress`, `BuyerSendAddress`, `BuyerChainDeadline`) and that `MarshalJSON`/`UnmarshalJSON` exist. Adjust the fixture to the real struct if fields differ.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestParse -v`
Expected: FAIL — `instruction`, `memoProgramID`, `computeBudgetProgramID`, `newTestTransaction`, `Transaction.parseInstructions` undefined.

- [ ] **Step 3: Implement `transaction.go`**

Pre-made decisions embedded below:
- **Program IDs** are declared as package vars via `solana.MustPublicKeyFromBase58`. Memo v2: `MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr`. ComputeBudget: `ComputeBudget111111111111111111111111111111`. Token program and System program come from `solana` constants.
- **Transfer parsing scope (native SOL):** System Program `Transfer` instruction is discriminator `2` (u32 LE) followed by `u64 LE lamports`; accounts are `[funding, recipient]`. Support this one variant.
- **Transfer parsing scope (SPL):** support `Transfer` (tag `3`) and `TransferChecked` (tag `12`). For `Transfer`, accounts are `[source, destination, owner, ...]` and amount is `u64 LE` at data[1:9]. For `TransferChecked`, accounts are `[source, mint, destination, owner, ...]` and amount is `u64 LE` at data[1:9]. Destination here is a **token account** (the ATA), which is what `MatchesOrderDestination` compares against.
- **Mint for SPL:** for `TransferChecked` the mint is an account (index 1). For bare `Transfer` the mint is not in the instruction; leave `mint` zero and rely on the design's ATA-derivation match in `MatchesOrderDestination` using the sell order's contract bytes as the mint. (Native SOL has no mint.)

```go
// NEW — cmd/rpc/oracle/sol/transaction.go
package sol

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go"
)

const solanaBlockchain = "solana"

var (
	memoProgramID          = solana.MustPublicKeyFromBase58("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")
	computeBudgetProgramID = solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111")
)

// OrderValidator validates canopy order JSON found in a Memo instruction.
type OrderValidator interface {
	ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error
}

var _ types.TransactionI = &Transaction{}

// instruction is a resolved (program-id + account pubkeys + data) view of one top-level
// instruction, decoupled from solana-go's compiled/indexed form so parsing is pure and testable.
type instruction struct {
	programID solana.PublicKey
	accounts  []solana.PublicKey
	data      []byte
}

// Transaction wraps a Solana transaction and implements types.TransactionI.
type Transaction struct {
	signature string        // base58 transaction signature (used as Hash/TransactionID)
	feePayer  string        // first signer account, used as From()
	instrs    []instruction // resolved top-level instructions

	order *types.WitnessedOrder // parsed order, if any

	// transfer fields, populated for close orders
	isTransfer  bool
	destination string   // token account (SPL) or wallet (native) receiving funds
	mint        string   // SPL mint (base58), empty for native SOL
	amount      *big.Int // base-unit amount
	decimals    uint8    // resolved by the provider via mint cache; 0 for native SOL by convention
}

// From returns the fee payer (first signer).
func (t *Transaction) From() string { return t.feePayer }

// To returns the transfer destination token/wallet account (empty for lock orders).
func (t *Transaction) To() string { return t.destination }

// Hash returns the transaction signature.
func (t *Transaction) Hash() string { return t.signature }

// Blockchain returns the chain identifier.
func (t *Transaction) Blockchain() string { return solanaBlockchain }

// Order returns the parsed witnessed order, if any.
func (t *Transaction) Order() *types.WitnessedOrder { return t.order }

// clearOrder resets parsed state (mirrors eth's clearOrder for retry paths).
func (t *Transaction) clearOrder() {
	t.order = nil
	t.isTransfer = false
}

// TokenTransfer returns the generic token transfer view.
func (t *Transaction) TokenTransfer() types.TokenTransfer {
	return types.TokenTransfer{
		Blockchain:       solanaBlockchain,
		TokenInfo:        types.TokenInfo{Decimals: t.decimals},
		TransactionID:    t.signature,
		SenderAddress:    t.feePayer,
		RecipientAddress: t.destination,
		TokenBaseAmount:  t.amount,
		ContractAddress:  t.mint,
	}
}

// findInstruction returns the first top-level instruction whose program id matches, or nil.
func (t *Transaction) findInstruction(programID solana.PublicKey) *instruction {
	for i := range t.instrs {
		if t.instrs[i].programID.Equals(programID) {
			return &t.instrs[i]
		}
	}
	return nil
}

// parseInstructions scans top-level instructions for a Memo (order JSON) and, for close orders,
// a transfer instruction. No order found is not an error (returns nil, order stays nil).
func (t *Transaction) parseInstructions(v OrderValidator) error {
	memo := t.findInstruction(memoProgramID)
	if memo == nil {
		return nil // not an order transaction
	}
	// try lock order first (no transfer required)
	if v.ValidateOrderJsonBytes(memo.data, types.LockOrderType) == nil {
		lo := &lib.LockOrder{}
		if err := lo.UnmarshalJSON(memo.data); err != nil {
			return fmt.Errorf("failed to unmarshal lock order json: %w", err)
		}
		t.order = &types.WitnessedOrder{OrderId: lo.OrderId, LockOrder: lo}
		return nil
	}
	// try close order (requires a transfer instruction)
	if v.ValidateOrderJsonBytes(memo.data, types.CloseOrderType) == nil {
		co := &lib.CloseOrder{}
		if err := co.UnmarshalJSON(memo.data); err != nil {
			return fmt.Errorf("failed to unmarshal close order json: %w", err)
		}
		t.order = &types.WitnessedOrder{OrderId: co.OrderId, CloseOrder: co}
		if err := t.parseTransfer(); err != nil {
			return err
		}
		return nil
	}
	// memo present but not a canopy order - normal condition
	return nil
}

// parseTransfer locates and decodes the funds-transfer instruction (SPL token or native SOL).
func (t *Transaction) parseTransfer() error {
	// SPL token transfer takes precedence when present
	if spl := t.findInstruction(solana.TokenProgramID); spl != nil {
		return t.parseSPLTransfer(spl)
	}
	// otherwise native SOL transfer
	if sys := t.findInstruction(solana.SystemProgramID); sys != nil {
		return t.parseNativeTransfer(sys)
	}
	return fmt.Errorf("close order transaction has no transfer instruction")
}

// parseSPLTransfer decodes SPL Token `Transfer` (tag 3) or `TransferChecked` (tag 12).
func (t *Transaction) parseSPLTransfer(in *instruction) error {
	if len(in.data) < 9 {
		return fmt.Errorf("spl transfer data too short")
	}
	tag := in.data[0]
	amount := binary.LittleEndian.Uint64(in.data[1:9])
	switch tag {
	case 3: // Transfer: accounts [source, destination, owner]
		if len(in.accounts) < 2 {
			return fmt.Errorf("spl transfer missing accounts")
		}
		t.destination = in.accounts[1].String()
		// mint not present in bare Transfer; left empty, matched via ATA derivation
	case 12: // TransferChecked: accounts [source, mint, destination, owner]
		if len(in.accounts) < 3 {
			return fmt.Errorf("spl transferChecked missing accounts")
		}
		t.mint = in.accounts[1].String()
		t.destination = in.accounts[2].String()
	default:
		return fmt.Errorf("unsupported spl token instruction tag %d", tag)
	}
	t.amount = new(big.Int).SetUint64(amount)
	t.isTransfer = true
	return nil
}

// parseNativeTransfer decodes a System Program `Transfer` (discriminator 2, u64 lamports).
func (t *Transaction) parseNativeTransfer(in *instruction) error {
	if len(in.data) < 12 {
		return fmt.Errorf("system transfer data too short")
	}
	if binary.LittleEndian.Uint32(in.data[0:4]) != 2 {
		return fmt.Errorf("not a system transfer instruction")
	}
	if len(in.accounts) < 2 {
		return fmt.Errorf("system transfer missing accounts")
	}
	lamports := binary.LittleEndian.Uint64(in.data[4:12])
	t.destination = in.accounts[1].String() // recipient wallet
	t.amount = new(big.Int).SetUint64(lamports)
	t.mint = "" // native SOL, no mint
	t.isTransfer = true
	return nil
}
```

Add the `MatchesOrderDestination` method (ATA derivation). This is the load-bearing recipient check:

```go
// NEW — cmd/rpc/oracle/sol/transaction.go (append)

// MatchesOrderDestination verifies the transfer went to the seller's expected destination.
// For SPL transfers, the expected destination is the Associated Token Account derived from
// (recipient wallet, mint) — recipientBytes is the seller's WALLET pubkey and contractBytes is
// the mint. For native SOL transfers, the destination is compared directly to recipientBytes.
func (t *Transaction) MatchesOrderDestination(contractBytes, recipientBytes []byte) (bool, error) {
	if !t.isTransfer {
		return false, nil
	}
	recipient := solana.PublicKeyFromBytes(recipientBytes)
	// native SOL: no mint/contract, compare wallet directly
	if len(contractBytes) == 0 {
		return recipient.String() == t.destination, nil
	}
	mint := solana.PublicKeyFromBytes(contractBytes)
	ata, _, err := solana.FindAssociatedTokenAddress(recipient, mint)
	if err != nil {
		return false, err
	}
	return ata.String() == t.destination, nil
}
```

Finally add the test-only constructor used by the tests (place in `transaction.go`, not the test file, so both the provider and tests share it; or in a `_test.go` helper — put it in `transaction.go` for reuse by the provider path):

```go
// NEW — cmd/rpc/oracle/sol/transaction.go (append)

// newTransaction builds a Transaction from resolved instructions and identity fields.
func newTransaction(signature, feePayer string, instrs []instruction) *Transaction {
	return &Transaction{signature: signature, feePayer: feePayer, instrs: instrs}
}
```

And the test helper alias in the test file:

```go
// ADD to cmd/rpc/oracle/sol/transaction_test.go

// newTestTransaction is a thin wrapper matching the test call sites.
func newTestTransaction(sig string, instrs []instruction, _ interface{}) *Transaction {
	return newTransaction(sig, "FeePayer1111111111111111111111111111111111", instrs)
}
```

**Verification note:** Task 5 must have confirmed `solana.FindAssociatedTokenAddress`, `solana.PublicKeyFromBytes`, `solana.TokenProgramID`, `solana.SystemProgramID`, and `(PublicKey).Equals`/`.String()` exist with these signatures. If `FindAssociatedTokenAddress` lives in a subpackage (e.g. `programs/associated-token-account`), import that and adjust the call.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestParse -v`
Expected: PASS.

- [ ] **Step 5: Add close-order + native/SPL transfer + ATA-match tests**

```go
// ADD to cmd/rpc/oracle/sol/transaction_test.go

func closeOrderJSONFixture(t *testing.T) []byte {
	t.Helper()
	co := &lib.CloseOrder{
		OrderId:   []byte{0x01, 0x02, 0x03},
		ChainId:   1,
		CloseOrder: true,
	}
	b, err := co.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal close order fixture: %v", err)
	}
	return b
}

func TestParse_MemoPlusSPLTransfer_IsCloseOrder(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	destATA := solana.NewWallet().PublicKey()
	owner := solana.NewWallet().PublicKey()
	// SPL Transfer (tag 3), amount 500 LE u64
	data := append([]byte{3}, u64LE(500)...)
	instrs := []instruction{
		{programID: memoProgramID, data: closeOrderJSONFixture(t)},
		{programID: solana.TokenProgramID, accounts: []solana.PublicKey{source, destATA, owner}, data: data},
	}
	// close order fixture accepts close type only
	v := &fakeValidator{lockErr: errNotOrder, closeErr: nil}
	tx := newTestTransaction("sig-close", instrs, nil)
	if err := tx.parseInstructions(v); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tx.Order() == nil || tx.Order().CloseOrder == nil {
		t.Fatal("expected close order")
	}
	if tx.To() != destATA.String() {
		t.Fatalf("expected destination %s, got %s", destATA.String(), tx.To())
	}
	if tx.TokenTransfer().TokenBaseAmount.Uint64() != 500 {
		t.Fatalf("expected amount 500, got %s", tx.TokenTransfer().TokenBaseAmount)
	}
}

func TestMatchesOrderDestination_SPL_ATA(t *testing.T) {
	wallet := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	ata, _, err := solana.FindAssociatedTokenAddress(wallet, mint)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Transaction{isTransfer: true, destination: ata.String()}
	ok, err := tx.MatchesOrderDestination(mint.Bytes(), wallet.Bytes())
	if err != nil || !ok {
		t.Fatalf("expected ATA match, ok=%v err=%v", ok, err)
	}
	// wrong wallet -> mismatch
	ok, _ = tx.MatchesOrderDestination(mint.Bytes(), solana.NewWallet().PublicKey().Bytes())
	if ok {
		t.Fatal("expected mismatch for wrong wallet")
	}
}

func TestMatchesOrderDestination_NativeSOL(t *testing.T) {
	wallet := solana.NewWallet().PublicKey()
	tx := &Transaction{isTransfer: true, destination: wallet.String()}
	ok, err := tx.MatchesOrderDestination(nil, wallet.Bytes())
	if err != nil || !ok {
		t.Fatalf("expected native match, ok=%v err=%v", ok, err)
	}
}
```

Add the small test helpers (`u64LE`, `errNotOrder`) to the test file:

```go
// ADD to cmd/rpc/oracle/sol/transaction_test.go
import "encoding/binary"
import "errors"

var errNotOrder = errors.New("not an order")

func u64LE(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}
```

**Verify before writing:** confirm `solana.NewWallet().PublicKey()` exists (it does in gagliardetto). Confirm `lib.CloseOrder` field names via `go doc github.com/canopy-network/canopy/lib CloseOrder`; adjust the fixture accordingly.

- [ ] **Step 6: Run all sol transaction tests**

Run: `go test ./cmd/rpc/oracle/sol/ -run 'TestParse|TestMatches' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction.go cmd/rpc/oracle/sol/transaction_test.go
git commit -m "feat(sol): parse memo/transfer instructions and implement TransactionI"
```

---

## Task 10: Implement `sol/block.go` — `BlockI`

**Files:**
- Create: `cmd/rpc/oracle/sol/block.go`
- Test: `cmd/rpc/oracle/sol/block_test.go`

Mirror `eth/block.go`. A Solana block's `Number()` is its **slot**. `Hash()` is the block's `blockhash`; `ParentHash()` is `previousBlockhash`. The oracle's reorg logic uses these, but under `finalized` commitment Solana blocks do not reorg, so these are informational.

- [ ] **Step 1: Write the failing test**

```go
// NEW — cmd/rpc/oracle/sol/block_test.go
package sol

import (
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
)

func TestBlock_Accessors(t *testing.T) {
	txs := []*Transaction{newTransaction("sig1", "payer", nil)}
	b := newBlock(1234, "blockhashX", "parenthashY", txs)
	if b.Number() != 1234 {
		t.Fatalf("expected slot 1234, got %d", b.Number())
	}
	if b.Hash() != "blockhashX" || b.ParentHash() != "parenthashY" {
		t.Fatalf("hash/parent mismatch: %s %s", b.Hash(), b.ParentHash())
	}
	got := b.Transactions()
	if len(got) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(got))
	}
	var _ types.BlockI = b
	var _ types.TransactionI = got[0]
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestBlock_Accessors -v`
Expected: FAIL — `newBlock` undefined.

- [ ] **Step 3: Implement `block.go`**

```go
// NEW — cmd/rpc/oracle/sol/block.go
package sol

import "github.com/canopy-network/canopy/cmd/rpc/oracle/types"

var _ types.BlockI = &Block{}

// Block wraps a Solana block (identified by slot) and implements types.BlockI.
type Block struct {
	slot         uint64
	hash         string // blockhash
	parentHash   string // previousBlockhash
	transactions []*Transaction
}

// newBlock creates a Block from decoded fields.
func newBlock(slot uint64, hash, parentHash string, txs []*Transaction) *Block {
	return &Block{slot: slot, hash: hash, parentHash: parentHash, transactions: txs}
}

// Hash returns the block hash (blockhash).
func (b *Block) Hash() string { return b.hash }

// ParentHash returns the previous block hash.
func (b *Block) ParentHash() string { return b.parentHash }

// Number returns the slot number.
func (b *Block) Number() uint64 { return b.slot }

// Transactions returns the block's transactions as interface values.
func (b *Block) Transactions() []types.TransactionI {
	txs := make([]types.TransactionI, len(b.transactions))
	for i, tx := range b.transactions {
		txs[i] = tx
	}
	return txs
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestBlock_Accessors -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/oracle/sol/block.go cmd/rpc/oracle/sol/block_test.go
git commit -m "feat(sol): implement BlockI wrapper"
```

---

## Task 11: Implement `sol/block_provider.go` — polling loop + `BlockProvider`

**Files:**
- Create: `cmd/rpc/oracle/sol/block_provider.go`
- Test: `cmd/rpc/oracle/sol/block_provider_test.go`

Mirror `eth/block_provider.go`'s *shape* (channel-based, `Start`/`BlockCh`/`IsSynced`, unbuffered block channel, timeout-bounded batch processing) but replace the WS run loop with a polling loop. Define a small `SolanaRpcClient` interface (like `EthereumRpcClient`) so tests inject a mock without a live endpoint.

**Pre-made decisions:**
- Poll `GetSlot(finalized)` every `PollIntervalMs`. For each slot in `[nextSlot, currentSlot]`, call `GetBlockWithOpts(slot, finalized, maxSupportedTransactionVersion=0)`.
- **Skipped slot:** if the RPC error's jsonrpc code is `-32007` or `-32009`, treat as skipped — log, advance `nextSlot` past it, do NOT retry. Any other error: log, retry same slot after `RetryDelay`, do NOT advance.
- **Skip detection:** type-assert the error to the SDK's jsonrpc error type recorded in Task 5, read `.Code`. If that type is not readily assertable, fall back to matching the decimal codes in `err.Error()`. Encapsulate this in one helper `isSkippedSlotErr(err) bool` so it is the single place to adjust.
- Build `sol.Transaction` values via `resolveInstructions` (converts solana-go compiled instructions + account keys into `[]instruction`), then `parseInstructions`. Resolve SPL decimals via the mint cache for transactions that carry an SPL close-order transfer (decimals are display-only; failure to fetch is logged and non-fatal — set decimals 0).
- Success filter: include a transaction only if its `meta.Err == nil`.

- [ ] **Step 1: Write the failing test (skipped-slot advance + normal processing)**

```go
// NEW — cmd/rpc/oracle/sol/block_provider_test.go
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
	slot        uint64
	blocks      map[uint64]*fakeBlock // slot -> block, missing => skipped
	skippedErr  error
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

var _ = errors.New // keep errors imported if unused above
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestProvider_SkipsMissingSlot -v`
Expected: FAIL — `newSolBlockProviderWithClient`, `SolBlockProvider`, `SolanaRpcClient` undefined.

- [ ] **Step 3: Implement `block_provider.go`**

The test injects a client whose `GetBlock` already returns a `*Block` (fetch + wrap combined) so the loop is testable without decoding real solana-go blocks. Define the injectable interface at that boundary; the production client (Task 12 wires the real `*rpc.Client` behind an adapter that decodes blocks).

```go
// NEW — cmd/rpc/oracle/sol/block_provider.go
package sol

import (
	"context"
	"errors"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle"
	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

const processBlocksTimeLimitS = 12

var _ oracle.BlockProvider = &SolBlockProvider{}

// SolanaRpcClient is the minimal RPC surface the provider needs. The production adapter (see cli
// wiring) fetches a raw block and decodes it into *Block; tests inject a fake.
type SolanaRpcClient interface {
	GetSlot(ctx context.Context) (uint64, error)
	GetBlock(ctx context.Context, slot uint64) (*Block, error)
	Close()
}

// coder is implemented by the SDK's jsonrpc error and by the test skipErr; used to read the code.
type coder interface{ Code() int }

// isSkippedSlotErr reports whether err indicates a skipped slot (jsonrpc -32007 or -32009).
func isSkippedSlotErr(err error) bool {
	var c coder
	if errors.As(err, &c) {
		code := c.Code()
		return code == -32007 || code == -32009
	}
	return false
}

// SolBlockProvider polls a Solana RPC endpoint at finalized commitment and delivers blocks.
type SolBlockProvider struct {
	config    lib.SolBlockProviderConfig
	client    SolanaRpcClient
	validator OrderValidator
	logger    lib.LoggerI
	metrics   *lib.Metrics
	blockChan chan types.BlockI
	nextSlot  uint64
	synced    bool
}

// newSolBlockProviderWithClient builds a provider around an injected client (used by tests + cli adapter).
func newSolBlockProviderWithClient(cfg lib.SolBlockProviderConfig, client SolanaRpcClient, v OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) *SolBlockProvider {
	return &SolBlockProvider{
		config:    cfg,
		client:    client,
		validator: v,
		logger:    logger,
		metrics:   metrics,
		blockChan: make(chan types.BlockI), // unbuffered: backpressure like eth
	}
}

// BlockCh returns the block delivery channel.
func (p *SolBlockProvider) BlockCh() chan types.BlockI { return p.blockChan }

// IsSynced reports whether the provider has caught up to the finalized tip.
func (p *SolBlockProvider) IsSynced() bool { return p.synced }

// Start begins the poll loop at the given slot.
func (p *SolBlockProvider) Start(ctx context.Context, height uint64) {
	p.nextSlot = height
	p.logger.Info("[SOL-CONN] starting solana block provider")
	go p.run(ctx)
}

func (p *SolBlockProvider) run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.config.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	defer p.client.Close()
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("[SOL-CONN] shutting down solana block provider")
			return
		case <-ticker.C:
		}
		current, err := p.client.GetSlot(ctx)
		if err != nil {
			p.logger.Errorf("[SOL-RPC] GetSlot failed: %v", err)
			continue
		}
		p.metrics.SetChainHeadHeight(current)
		if current < p.nextSlot {
			// under finalized commitment slots are monotonic; log and wait
			p.logger.Warnf("[SOL-SYNC] finalized slot %d below nextSlot %d", current, p.nextSlot)
			continue
		}
		p.synced = false
		p.processSlots(ctx, current)
		if p.nextSlot > current {
			p.synced = true
		}
	}
}

// processSlots delivers blocks for [nextSlot, current], honoring skip vs retry semantics.
func (p *SolBlockProvider) processSlots(ctx context.Context, current uint64) {
	timeoutCtx, cancel := context.WithTimeout(ctx, processBlocksTimeLimitS*time.Second)
	defer cancel()
	for p.nextSlot <= current {
		select {
		case <-timeoutCtx.Done():
			return
		default:
		}
		block, err := p.client.GetBlock(timeoutCtx, p.nextSlot)
		if err != nil {
			if isSkippedSlotErr(err) {
				p.logger.Infof("[SOL-BLOCK] slot %d skipped, advancing", p.nextSlot)
				p.nextSlot++ // advance past skipped slot, do not retry
				continue
			}
			p.logger.Errorf("[SOL-RPC] GetBlock(%d) failed: %v; retrying", p.nextSlot, err)
			return // retry same slot next tick, do not advance
		}
		select {
		case p.blockChan <- block:
		case <-timeoutCtx.Done():
			return // do not advance; retry this slot
		}
		p.nextSlot++
	}
}
```

**Verification note:** `p.metrics.SetChainHeadHeight` must exist after Task 4's rename. If Task 4 was skipped, change it to `SetEthChainHeadHeight` or drop the metric call.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestProvider_SkipsMissingSlot -v`
Expected: PASS.

- [ ] **Step 5: Add a transient-error retry test**

```go
// ADD to cmd/rpc/oracle/sol/block_provider_test.go

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

func (c *countingRPC) GetSlot(ctx context.Context) (uint64, error)         { return c.slot, nil }
func (c *countingRPC) GetBlock(ctx context.Context, slot uint64) (*Block, error) { return c.onGetBlock(slot) }
func (c *countingRPC) Close()                                              {}
```

- [ ] **Step 6: Run all provider tests**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestProvider -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/rpc/oracle/sol/block_provider.go cmd/rpc/oracle/sol/block_provider_test.go
git commit -m "feat(sol): polling block provider with skip/retry slot handling"
```

---

## Task 12: Production RPC adapter — decode real Solana blocks into `*Block`

**Files:**
- Modify: `cmd/rpc/oracle/sol/block_provider.go` (add `NewSolBlockProvider` + real client adapter)
- Test: `cmd/rpc/oracle/sol/resolve_test.go` (instruction resolution from a compiled message)

Task 11's `SolanaRpcClient.GetBlock` returns a decoded `*Block`. The production adapter wraps `*rpc.Client`: fetches the raw block via `GetBlockWithOpts`, iterates transactions with `meta.Err == nil`, resolves each transaction's compiled instructions into `[]instruction`, runs `parseInstructions`, resolves SPL decimals via the mint cache, and builds `*Block`.

**Pre-made decision:** `resolveInstructions` maps each `solana.CompiledInstruction` to `instruction{programID, accounts, data}` using the message's account key list: `programID = message.Account(instr.ProgramIDIndex)`, `accounts[i] = message.Account(instr.Accounts[i])`, `data = instr.Data` (base58-decoded bytes). Confirm the exact accessor names from Task 5.

- [ ] **Step 1: Write the failing test for instruction resolution**

Because constructing a full `*solana.Transaction` is verbose, test `resolveInstructions` against a hand-built `*solana.Message` with two accounts and one instruction.

```go
// NEW — cmd/rpc/oracle/sol/resolve_test.go
package sol

import (
	"testing"

	"github.com/gagliardetto/solana-go"
)

func TestResolveInstructions(t *testing.T) {
	prog := memoProgramID
	acct := solana.NewWallet().PublicKey()
	msg := &solana.Message{
		AccountKeys: []solana.PublicKey{prog, acct},
		Instructions: []solana.CompiledInstruction{
			{ProgramIDIndex: 0, Accounts: []uint16{1}, Data: []byte("hello")},
		},
	}
	got, err := resolveInstructions(msg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(got))
	}
	if !got[0].programID.Equals(prog) {
		t.Fatalf("program id mismatch")
	}
	if len(got[0].accounts) != 1 || !got[0].accounts[0].Equals(acct) {
		t.Fatalf("account resolution mismatch")
	}
	if string(got[0].data) != "hello" {
		t.Fatalf("data mismatch: %q", got[0].data)
	}
}
```

**Verify before writing:** `go doc github.com/gagliardetto/solana-go Message` and `CompiledInstruction`. The field `Data` is likely `solana.Base58` (a `[]byte` alias); the literal `Data: []byte("hello")` must be convertible — if `Data` is typed `solana.Base58`, write `Data: solana.Base58([]byte("hello"))`. Confirm `AccountKeys`/`Instructions` field names and the `.Account(idx)` accessor. Adapt this test and `resolveInstructions` to the real API.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rpc/oracle/sol/ -run TestResolveInstructions -v`
Expected: FAIL — `resolveInstructions` undefined.

- [ ] **Step 3: Implement `resolveInstructions` and the production provider**

```go
// ADD to cmd/rpc/oracle/sol/block_provider.go (imports: add solana + rpc + fmt)

// resolveInstructions converts a message's compiled instructions into resolved instructions
// using the message's static account key list. Requires transactions with all accounts listed
// inline (no Address Lookup Tables — see docs/SOLANA_ORACLE_REQUIREMENTS.md).
func resolveInstructions(msg *solana.Message) ([]instruction, error) {
	out := make([]instruction, 0, len(msg.Instructions))
	for _, ci := range msg.Instructions {
		if int(ci.ProgramIDIndex) >= len(msg.AccountKeys) {
			return nil, fmt.Errorf("program id index %d out of range", ci.ProgramIDIndex)
		}
		accts := make([]solana.PublicKey, 0, len(ci.Accounts))
		for _, ai := range ci.Accounts {
			if int(ai) >= len(msg.AccountKeys) {
				return nil, fmt.Errorf("account index %d out of range", ai)
			}
			accts = append(accts, msg.AccountKeys[ai])
		}
		out = append(out, instruction{
			programID: msg.AccountKeys[ci.ProgramIDIndex],
			accounts:  accts,
			data:      []byte(ci.Data),
		})
	}
	return out, nil
}
```

Then the production client adapter + public constructor. This is the code that talks to the real SDK; keep it thin so all logic stays in the tested pieces:

```go
// ADD to cmd/rpc/oracle/sol/block_provider.go

// realClient adapts *rpc.Client to SolanaRpcClient, decoding raw blocks into *Block.
type realClient struct {
	rpc       *rpc.Client
	validator OrderValidator
	mints     *mintCache
	logger    lib.LoggerI
}

func (c *realClient) GetSlot(ctx context.Context) (uint64, error) {
	return c.rpc.GetSlot(ctx, rpc.CommitmentFinalized)
}

func (c *realClient) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	maxVer := uint64(0)
	res, err := c.rpc.GetBlockWithOpts(ctx, slot, &rpc.GetBlockOpts{
		Commitment:                     rpc.CommitmentFinalized,
		MaxSupportedTransactionVersion: &maxVer,
	})
	if err != nil {
		return nil, err // caller classifies skip vs transient
	}
	txs := make([]*Transaction, 0, len(res.Transactions))
	for i := range res.Transactions {
		twm := res.Transactions[i]
		if twm.Meta != nil && twm.Meta.Err != nil {
			continue // failed transaction, ignore
		}
		solTx, err := twm.GetTransaction()
		if err != nil || solTx == nil {
			c.logger.Warnf("[SOL-TX] failed to decode transaction in slot %d: %v", slot, err)
			continue
		}
		instrs, err := resolveInstructions(&solTx.Message)
		if err != nil {
			c.logger.Warnf("[SOL-TX] failed to resolve instructions in slot %d: %v", slot, err)
			continue
		}
		feePayer := ""
		if len(solTx.Message.AccountKeys) > 0 {
			feePayer = solTx.Message.AccountKeys[0].String()
		}
		sig := ""
		if len(solTx.Signatures) > 0 {
			sig = solTx.Signatures[0].String()
		}
		tx := newTransaction(sig, feePayer, instrs)
		if err := tx.parseInstructions(c.validator); err != nil {
			c.logger.Warnf("[SOL-TX] parse error in slot %d tx %s: %v", slot, sig, err)
			tx.clearOrder()
			continue
		}
		if tx.Order() == nil {
			continue // not an order transaction
		}
		tx.order.WitnessedHeight = slot
		// resolve SPL decimals (display-only; non-fatal on failure)
		if tx.isTransfer && tx.mint != "" {
			if mintPK, e := solana.PublicKeyFromBase58(tx.mint); e == nil {
				if d, e2 := c.mints.Decimals(ctx, c.rpc, mintPK); e2 == nil {
					tx.decimals = d
				}
			}
		}
		txs = append(txs, tx)
	}
	return newBlock(slot, res.Blockhash.String(), res.PreviousBlockhash.String(), txs), nil
}

func (c *realClient) Close() {} // *rpc.Client has no persistent connection to close

// NewSolBlockProvider constructs a production Solana block provider.
func NewSolBlockProvider(cfg lib.SolBlockProviderConfig, v OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) (*SolBlockProvider, error) {
	client := &realClient{
		rpc:       rpc.New(cfg.NodeUrl),
		validator: v,
		mints:     newMintCache(1024),
		logger:    logger,
	}
	logger.Infof("[SOL-CONN] created solana block provider with rpc: %s", cfg.NodeUrl)
	return newSolBlockProviderWithClient(cfg, client, v, logger, metrics), nil
}
```

**Verification note (critical — adapt to Task 5 findings):** the field/method names `res.Transactions`, `twm.Meta.Err`, `twm.GetTransaction()`, `solTx.Message`, `solTx.Signatures`, `res.Blockhash`, `res.PreviousBlockhash`, `rpc.CommitmentFinalized`, `rpc.GetBlockOpts.MaxSupportedTransactionVersion` are the load-bearing SDK names. Confirm each against `go doc` output from Task 5 and fix any that differ before building. This is the single riskiest task for API drift.

- [ ] **Step 4: Build and run the full sol package**

Run: `go build ./cmd/rpc/oracle/sol/... && go test ./cmd/rpc/oracle/sol/...`
Expected: PASS. If the build fails on an SDK symbol, reconcile against Task 5's recorded signatures.

- [ ] **Step 5: Commit**

```bash
git add cmd/rpc/oracle/sol/block_provider.go cmd/rpc/oracle/sol/resolve_test.go
git commit -m "feat(sol): production RPC adapter decoding solana blocks into BlockI"
```

---

## Task 13: Wire provider selection into the CLI

**Files:**
- Modify: `cmd/cli/cli.go:133-141`

Select the provider by which config is populated. **Decision:** a non-empty `SolBlockProviderConfig.NodeUrl` that differs from the eth default selects Solana; otherwise Ethereum. Since only one provider runs per process, gate on an explicit signal: add a boolean or infer from config. The cleanest explicit signal without a new config field is: if `config.SolBlockProviderConfig.NodeUrl != ""` AND the operator intends Solana. To avoid ambiguity (both have defaults), introduce a small explicit selector.

**Pre-made decision:** add one field `SourceChain string json:"oracleSourceChain"` to `OracleConfig` (values `"ethereum"` (default) or `"solana"`), and switch on it. This is unambiguous and self-documenting. (If the reviewer prefers inferring from config presence, that is a follow-up; explicit is safer.)

- [ ] **Step 1: Add `SourceChain` to `OracleConfig`**

```go
// CURRENT — lib/config.go (end of OracleConfig struct, after SafeBlockConfirmations)
	SafeBlockConfirmations   uint64 `json:"safeBlockConfirmations"`   // number of block confirmations required before considering a block safe
}

// NEW — lib/config.go
	SafeBlockConfirmations   uint64 `json:"safeBlockConfirmations"`   // number of block confirmations required before considering a block safe
	SourceChain              string `json:"oracleSourceChain"`        // which source chain to witness: "ethereum" (default) or "solana"
}
```

Set the default in `DefaultOracleConfig` (find it near the other defaults):

Run: `grep -n "func DefaultOracleConfig" lib/config.go`
Then add `SourceChain: "ethereum",` to the returned struct literal.

- [ ] **Step 2: Switch on `SourceChain` in `cli.go`**

```go
// CURRENT — cmd/cli/cli.go:132-141
		// create the ethereum block provider
		ethBlockProvider, e := eth.NewEthBlockProvider(config.EthBlockProviderConfig, orderValidator, oracleLogger, metrics)
		if e != nil {
			l.Fatal(e.Error())
		}

		// create an absolute path for the state save file
		config.OracleConfig.StateFile = filepath.Join(oracleRoot, config.OracleConfig.StateFile)
		// create a new oracle instance and pass the ethereum block provider with shared context
		o, e = oracle.NewOracle(ctx, config.OracleConfig, ethBlockProvider, oracleStorage, oracleLogger, metrics)

// NEW — cmd/cli/cli.go
		// select the source-chain block provider by configuration
		var blockProvider oracle.BlockProvider
		switch config.OracleConfig.SourceChain {
		case "solana":
			blockProvider, e = sol.NewSolBlockProvider(config.SolBlockProviderConfig, orderValidator, oracleLogger, metrics)
		default: // "ethereum" or unset
			blockProvider, e = eth.NewEthBlockProvider(config.EthBlockProviderConfig, orderValidator, oracleLogger, metrics)
		}
		if e != nil {
			l.Fatal(e.Error())
		}

		// create an absolute path for the state save file
		config.OracleConfig.StateFile = filepath.Join(oracleRoot, config.OracleConfig.StateFile)
		// create a new oracle instance and pass the selected block provider with shared context
		o, e = oracle.NewOracle(ctx, config.OracleConfig, blockProvider, oracleStorage, oracleLogger, metrics)
```

Add the `sol` import to `cli.go`'s import block:

```go
	"github.com/canopy-network/canopy/cmd/rpc/oracle/sol"
```

(Confirm the existing `eth` import path in `cli.go` and mirror it for `sol`.)

- [ ] **Step 3: Build everything**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Run the full oracle + lib test suites**

Run: `go test ./cmd/rpc/oracle/... ./lib/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/config.go cmd/cli/cli.go
git commit -m "feat(oracle): select source-chain block provider via OracleConfig.SourceChain"
```

---

## Task 14: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Vet and full build**

Run: `go vet ./cmd/rpc/oracle/... ./lib/... && go build ./...`
Expected: no vet warnings, clean build.

- [ ] **Step 2: Full test run for touched packages**

Run: `go test ./cmd/rpc/oracle/... ./lib/...`
Expected: all PASS.

- [ ] **Step 3: Confirm interface conformance is compile-checked**

Verify these lines exist (they cause a compile error if the type ever stops satisfying the interface):
- `cmd/rpc/oracle/sol/transaction.go`: `var _ types.TransactionI = &Transaction{}`
- `cmd/rpc/oracle/sol/block.go`: `var _ types.BlockI = &Block{}`
- `cmd/rpc/oracle/sol/block_provider.go`: `var _ oracle.BlockProvider = &SolBlockProvider{}`

Run: `grep -rn "var _ " cmd/rpc/oracle/sol/`
Expected: all three present.

- [ ] **Step 4: Manual smoke test note (optional, requires a Solana RPC endpoint)**

Set `oracleSourceChain: "solana"` and `solNodeUrl` to a devnet/local `solana-test-validator` endpoint in the node config, start the oracle, and confirm the log line `[SOL-CONN] starting solana block provider` appears and slots advance without error spam. This is out of scope for automated CI but validates the poll loop against a real endpoint.

---

## Out of Scope (do NOT implement — see `docs/SOLANA_ORACLE_REQUIREMENTS.md`)

- CPI / inner-instruction scanning (`meta.innerInstructions`).
- Address Lookup Table resolution.
- Metaplex Token Metadata (name/symbol) fetching.
- WebSocket `blockSubscribe`.

Revisit these only if the swap client stops being a bespoke Canopy tool, or the RPC provider begins exposing `blockSubscribe`.
