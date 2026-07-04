# Solana Oracle `sol` Package Missing Unit Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the unit-test coverage gaps in `cmd/rpc/oracle/sol` (transaction parsing, mint-decimals caching, and instruction resolution) identified in `HANDOFF.md`, so the parsing engine that was merged without ever compiling now has a real safety net.

**Architecture:** All tests are **characterization tests for already-implemented, already-passing code** — no production code changes are needed or expected. Every new test function must go GREEN on first run. If any new test goes RED, STOP: that indicates a real bug in `transaction.go`, `mint_cache.go`, or `block_provider.go`, not a bad test — do not "fix" the test to make it pass, investigate and report back before continuing the plan.

**Tech Stack:** Go 1.25 (`GOTOOLCHAIN=go1.25.3` — required locally, see `HANDOFF.md`), `testing`, `github.com/gagliardetto/solana-go`, `github.com/gagliardetto/solana-go/rpc`.

---

## Before You Start

Every test run in this plan **must** set `GOTOOLCHAIN=go1.25.3`, or the build fails on the `cockroachdb/swiss` dependency before your new test ever runs (see `HANDOFF.md` "What Happened"). All `go test` commands below already include this.

Existing files you will be editing:
- `cmd/rpc/oracle/sol/transaction_test.go` — add SPL/native transfer parsing, `parseInstructions`, `parseTransfer`, `MatchesOrderDestination`, and `TokenTransfer`/accessor tests here (it already has the `fakeValidator`, `newTestTransaction`, `u64LE`, `lockOrderJSONFixture`, `closeOrderJSONFixture`, `errNotOrder` helpers you'll reuse).
- `cmd/rpc/oracle/sol/mint_cache_test.go` — add `Decimals()` fetch/cache/error tests here (it already has `makeMintData`).
- `cmd/rpc/oracle/sol/resolve_test.go` — add out-of-range and empty-instruction tests here.

No new files are created.

---

### Task 1: `parseSPLTransfer` — TransferChecked + edge cases

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add the TransferChecked happy-path test and the four edge-case tests**

Append to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestParseSPLTransfer_TransferChecked(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	dest := solana.NewWallet().PublicKey()
	owner := solana.NewWallet().PublicKey()
	data := append([]byte{12}, u64LE(750)...)
	in := &instruction{
		programID: solana.TokenProgramID,
		accounts:  []solana.PublicKey{source, mint, dest, owner},
		data:      data,
	}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !tx.isTransfer {
		t.Fatal("expected isTransfer=true")
	}
	if tx.mint != mint.String() {
		t.Fatalf("expected mint %s, got %s", mint.String(), tx.mint)
	}
	if tx.destination != dest.String() {
		t.Fatalf("expected destination %s, got %s", dest.String(), tx.destination)
	}
	if tx.amount.Uint64() != 750 {
		t.Fatalf("expected amount 750, got %s", tx.amount)
	}
}

func TestParseSPLTransfer_DataTooShort(t *testing.T) {
	in := &instruction{programID: solana.TokenProgramID, data: []byte{3, 1, 2}}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err == nil {
		t.Fatal("expected error for undersized spl transfer data")
	}
}

func TestParseSPLTransfer_Tag3_MissingAccounts(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	data := append([]byte{3}, u64LE(1)...)
	in := &instruction{programID: solana.TokenProgramID, accounts: []solana.PublicKey{source}, data: data}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err == nil {
		t.Fatal("expected error for tag 3 with <2 accounts")
	}
}

func TestParseSPLTransfer_Tag12_MissingAccounts(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	data := append([]byte{12}, u64LE(1)...)
	in := &instruction{programID: solana.TokenProgramID, accounts: []solana.PublicKey{source, mint}, data: data}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err == nil {
		t.Fatal("expected error for tag 12 with <3 accounts")
	}
}

func TestParseSPLTransfer_UnsupportedTag(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	dest := solana.NewWallet().PublicKey()
	owner := solana.NewWallet().PublicKey()
	data := append([]byte{99}, u64LE(1)...)
	in := &instruction{
		programID: solana.TokenProgramID,
		accounts:  []solana.PublicKey{source, dest, owner},
		data:      data,
	}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err == nil {
		t.Fatal("expected error for unsupported spl token instruction tag")
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestParseSPLTransfer_' -v`
Expected: all 5 tests PASS. If any FAIL, stop — this means `parseSPLTransfer` has a real bug; do not alter the test to force a pass, investigate `transaction.go` first.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover parseSPLTransfer TransferChecked path and edge cases"
```

---

### Task 2: `parseNativeTransfer` — happy path + edge cases

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add native transfer tests**

Append to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestParseNativeTransfer_HappyPath(t *testing.T) {
	from := solana.NewWallet().PublicKey()
	to := solana.NewWallet().PublicKey()
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 2)
	binary.LittleEndian.PutUint64(data[4:12], 12345)
	in := &instruction{programID: solana.SystemProgramID, accounts: []solana.PublicKey{from, to}, data: data}
	tx := &Transaction{}
	if err := tx.parseNativeTransfer(in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !tx.isTransfer {
		t.Fatal("expected isTransfer=true")
	}
	if tx.destination != to.String() {
		t.Fatalf("expected destination %s, got %s", to.String(), tx.destination)
	}
	if tx.amount.Uint64() != 12345 {
		t.Fatalf("expected amount 12345, got %s", tx.amount)
	}
	if tx.mint != "" {
		t.Fatalf("expected empty mint for native transfer, got %s", tx.mint)
	}
}

func TestParseNativeTransfer_DataTooShort(t *testing.T) {
	in := &instruction{programID: solana.SystemProgramID, data: make([]byte, 8)}
	tx := &Transaction{}
	if err := tx.parseNativeTransfer(in); err == nil {
		t.Fatal("expected error for undersized system transfer data")
	}
}

func TestParseNativeTransfer_WrongDiscriminator(t *testing.T) {
	from := solana.NewWallet().PublicKey()
	to := solana.NewWallet().PublicKey()
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 5) // not 2 (Transfer)
	binary.LittleEndian.PutUint64(data[4:12], 100)
	in := &instruction{programID: solana.SystemProgramID, accounts: []solana.PublicKey{from, to}, data: data}
	tx := &Transaction{}
	if err := tx.parseNativeTransfer(in); err == nil {
		t.Fatal("expected error for wrong discriminator")
	}
}

func TestParseNativeTransfer_MissingAccounts(t *testing.T) {
	from := solana.NewWallet().PublicKey()
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 2)
	binary.LittleEndian.PutUint64(data[4:12], 100)
	in := &instruction{programID: solana.SystemProgramID, accounts: []solana.PublicKey{from}, data: data}
	tx := &Transaction{}
	if err := tx.parseNativeTransfer(in); err == nil {
		t.Fatal("expected error for missing accounts")
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestParseNativeTransfer_' -v`
Expected: all 4 tests PASS. If any FAIL, stop and investigate `parseNativeTransfer` in `transaction.go` before proceeding.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover parseNativeTransfer happy path and edge cases"
```

---

### Task 3: `parseTransfer` — SPL precedence and no-transfer error

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add precedence and error tests**

Append to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestParseTransfer_SPLTakesPrecedenceOverNative(t *testing.T) {
	splSource := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	splDest := solana.NewWallet().PublicKey()
	splOwner := solana.NewWallet().PublicKey()
	splData := append([]byte{12}, u64LE(1)...)

	nativeFrom := solana.NewWallet().PublicKey()
	nativeTo := solana.NewWallet().PublicKey()
	nativeData := make([]byte, 12)
	binary.LittleEndian.PutUint32(nativeData[0:4], 2)
	binary.LittleEndian.PutUint64(nativeData[4:12], 999)

	tx := &Transaction{
		instrs: []instruction{
			{programID: solana.SystemProgramID, accounts: []solana.PublicKey{nativeFrom, nativeTo}, data: nativeData},
			{programID: solana.TokenProgramID, accounts: []solana.PublicKey{splSource, mint, splDest, splOwner}, data: splData},
		},
	}
	if err := tx.parseTransfer(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// mint is only ever set by the SPL path — a non-empty mint proves SPL, not native, was chosen.
	if tx.mint != mint.String() {
		t.Fatalf("expected SPL transfer to take precedence, got mint=%q destination=%q", tx.mint, tx.destination)
	}
	if tx.destination != splDest.String() {
		t.Fatalf("expected spl destination %s, got %s", splDest.String(), tx.destination)
	}
}

func TestParseTransfer_NoTransferInstruction_Errors(t *testing.T) {
	tx := &Transaction{instrs: []instruction{{programID: memoProgramID, data: closeOrderJSONFixture(t)}}}
	if err := tx.parseTransfer(); err == nil {
		t.Fatal("expected error when no transfer instruction is present")
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestParseTransfer_' -v`
Expected: both tests PASS. If FAIL, stop and investigate `parseTransfer` in `transaction.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover parseTransfer SPL precedence and no-transfer error"
```

---

### Task 4: `parseInstructions` — malformed JSON and error propagation

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add malformed-JSON, no-match, and propagation tests**

Append to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestParseInstructions_LockValidates_UnmarshalFails(t *testing.T) {
	instrs := []instruction{{programID: memoProgramID, data: []byte("not valid json")}}
	tx := newTestTransaction("sig-bad-lock", instrs, nil)
	v := &fakeValidator{lockErr: nil, closeErr: errNotOrder}
	if err := tx.parseInstructions(v); err == nil {
		t.Fatal("expected unmarshal error when lock JSON validates but is malformed")
	}
}

func TestParseInstructions_CloseValidates_UnmarshalFails(t *testing.T) {
	instrs := []instruction{{programID: memoProgramID, data: []byte("not valid json")}}
	tx := newTestTransaction("sig-bad-close", instrs, nil)
	v := &fakeValidator{lockErr: errNotOrder, closeErr: nil}
	if err := tx.parseInstructions(v); err == nil {
		t.Fatal("expected unmarshal error when close JSON validates but is malformed")
	}
}

func TestParseInstructions_MemoPresent_NeitherTypeValidates_IsNoOp(t *testing.T) {
	instrs := []instruction{{programID: memoProgramID, data: []byte("just a note")}}
	tx := newTestTransaction("sig-plain-memo", instrs, nil)
	v := &fakeValidator{lockErr: errNotOrder, closeErr: errNotOrder}
	if err := tx.parseInstructions(v); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tx.Order() != nil {
		t.Fatal("expected no order when memo matches neither order type")
	}
}

func TestParseInstructions_CloseValidated_TransferParseFails_Propagates(t *testing.T) {
	instrs := []instruction{{programID: memoProgramID, data: closeOrderJSONFixture(t)}}
	tx := newTestTransaction("sig-close-no-transfer", instrs, nil)
	v := &fakeValidator{lockErr: errNotOrder, closeErr: nil}
	if err := tx.parseInstructions(v); err == nil {
		t.Fatal("expected parseTransfer error to propagate when close order has no transfer instruction")
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestParseInstructions_' -v`
Expected: all 4 tests PASS. If FAIL, stop and investigate `parseInstructions` in `transaction.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover parseInstructions malformed-json and error-propagation paths"
```

---

### Task 5: `MatchesOrderDestination` — non-transfer, mismatch, and full integration

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add the remaining `MatchesOrderDestination` cases**

Append to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestMatchesOrderDestination_NotATransfer_ReturnsFalseNil(t *testing.T) {
	tx := &Transaction{isTransfer: false}
	ok, err := tx.MatchesOrderDestination(nil, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatal("expected false when transaction is not a transfer")
	}
}

func TestMatchesOrderDestination_NativeSOL_Mismatch(t *testing.T) {
	actualRecipient := solana.NewWallet().PublicKey()
	expectedRecipient := solana.NewWallet().PublicKey()
	tx := &Transaction{isTransfer: true, destination: actualRecipient.String()}
	ok, err := tx.MatchesOrderDestination(nil, expectedRecipient.Bytes())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatal("expected mismatch for different native SOL recipient")
	}
}

func TestMatchesOrderDestination_TransferChecked_ATA_Integration(t *testing.T) {
	source := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	recipientWallet := solana.NewWallet().PublicKey()
	owner := solana.NewWallet().PublicKey()
	expectedATA, _, err := solana.FindAssociatedTokenAddress(recipientWallet, mint)
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte{12}, u64LE(42)...)
	in := &instruction{
		programID: solana.TokenProgramID,
		accounts:  []solana.PublicKey{source, mint, expectedATA, owner},
		data:      data,
	}
	tx := &Transaction{}
	if err := tx.parseSPLTransfer(in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	ok, err := tx.MatchesOrderDestination(mint.Bytes(), recipientWallet.Bytes())
	if err != nil || !ok {
		t.Fatalf("expected TransferChecked destination to match derived ATA, ok=%v err=%v", ok, err)
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestMatchesOrderDestination_' -v`
Expected: all 5 tests (2 pre-existing + 3 new) PASS. If any new one FAILs, stop and investigate `MatchesOrderDestination` in `transaction.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover MatchesOrderDestination non-transfer, mismatch, and full ATA integration"
```

---

### Task 6: `TokenTransfer()` field mapping and simple accessors

**Files:**
- Modify: `cmd/rpc/oracle/sol/transaction_test.go`

- [ ] **Step 1: Add `TokenTransfer()` and accessor tests**

`TestTokenTransfer_FieldMapping` needs `math/big` for `big.Int`, which isn't imported yet. Update the import block:

```go
// CURRENT — cmd/rpc/oracle/sol/transaction_test.go:1-11
package sol

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go"
)

// NEW — cmd/rpc/oracle/sol/transaction_test.go
package sol

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go"
)
```

Then append the tests to `cmd/rpc/oracle/sol/transaction_test.go`:

```go
func TestTokenTransfer_FieldMapping(t *testing.T) {
	tx := newTransaction("sig-transfer", "FeePayer1111111111111111111111111111111111", nil)
	tx.destination = "Dest1111111111111111111111111111111111111"
	tx.mint = "Mint1111111111111111111111111111111111111"
	tx.amount = new(big.Int).SetUint64(555)
	tx.decimals = 6

	got := tx.TokenTransfer()
	if got.Blockchain != solanaBlockchain {
		t.Fatalf("expected blockchain %q, got %q", solanaBlockchain, got.Blockchain)
	}
	if got.TokenInfo.Decimals != 6 {
		t.Fatalf("expected decimals 6, got %d", got.TokenInfo.Decimals)
	}
	if got.TransactionID != "sig-transfer" {
		t.Fatalf("expected transaction id sig-transfer, got %s", got.TransactionID)
	}
	if got.SenderAddress != "FeePayer1111111111111111111111111111111111" {
		t.Fatalf("expected sender FeePayer1111111111111111111111111111111111, got %s", got.SenderAddress)
	}
	if got.RecipientAddress != tx.destination {
		t.Fatalf("expected recipient %s, got %s", tx.destination, got.RecipientAddress)
	}
	if got.TokenBaseAmount.Uint64() != 555 {
		t.Fatalf("expected amount 555, got %s", got.TokenBaseAmount)
	}
	if got.ContractAddress != tx.mint {
		t.Fatalf("expected contract address %s, got %s", tx.mint, got.ContractAddress)
	}
}

func TestTransaction_Accessors(t *testing.T) {
	tx := newTransaction("sig-accessors", "FeePayer1111111111111111111111111111111111", nil)
	tx.destination = "Dest1111111111111111111111111111111111111"
	if tx.From() != "FeePayer1111111111111111111111111111111111" {
		t.Fatalf("unexpected From(): %s", tx.From())
	}
	if tx.To() != "Dest1111111111111111111111111111111111111" {
		t.Fatalf("unexpected To(): %s", tx.To())
	}
	if tx.Hash() != "sig-accessors" {
		t.Fatalf("unexpected Hash(): %s", tx.Hash())
	}
	if tx.Blockchain() != solanaBlockchain {
		t.Fatalf("unexpected Blockchain(): %s", tx.Blockchain())
	}
	if tx.Order() != nil {
		t.Fatal("expected nil Order() before any parse")
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestTokenTransfer_FieldMapping|TestTransaction_Accessors' -v`
Expected: both tests PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/transaction_test.go
git commit -m "test(oracle/sol): cover TokenTransfer field mapping and Transaction accessors"
```

---

### Task 7: `resolveInstructions` — out-of-range and empty cases

**Files:**
- Modify: `cmd/rpc/oracle/sol/resolve_test.go`

- [ ] **Step 1: Add out-of-range and empty-instructions tests**

Append to `cmd/rpc/oracle/sol/resolve_test.go`:

```go
func TestResolveInstructions_ProgramIDIndexOutOfRange(t *testing.T) {
	msg := &solana.Message{
		AccountKeys: []solana.PublicKey{solana.NewWallet().PublicKey()},
		Instructions: []solana.CompiledInstruction{
			{ProgramIDIndex: 5, Accounts: nil, Data: solana.Base58([]byte("x"))},
		},
	}
	if _, err := resolveInstructions(msg); err == nil {
		t.Fatal("expected error for out-of-range program id index")
	}
}

func TestResolveInstructions_AccountIndexOutOfRange(t *testing.T) {
	prog := memoProgramID
	msg := &solana.Message{
		AccountKeys: []solana.PublicKey{prog},
		Instructions: []solana.CompiledInstruction{
			{ProgramIDIndex: 0, Accounts: []uint16{7}, Data: solana.Base58([]byte("x"))},
		},
	}
	if _, err := resolveInstructions(msg); err == nil {
		t.Fatal("expected error for out-of-range account index")
	}
}

func TestResolveInstructions_Empty(t *testing.T) {
	msg := &solana.Message{
		AccountKeys:  []solana.PublicKey{solana.NewWallet().PublicKey()},
		Instructions: nil,
	}
	got, err := resolveInstructions(msg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 resolved instructions, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestResolveInstructions_' -v`
Expected: all 4 tests (1 pre-existing + 3 new) PASS. If any new one FAILs, stop and investigate `resolveInstructions` in `block_provider.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/resolve_test.go
git commit -m "test(oracle/sol): cover resolveInstructions out-of-range and empty cases"
```

---

### Task 8: `mintCache.Decimals()` — fetch/cache/error paths

**Files:**
- Modify: `cmd/rpc/oracle/sol/mint_cache_test.go`

- [ ] **Step 1: Add a fake `AccountFetcher` and the `Decimals()` tests**

Append to `cmd/rpc/oracle/sol/mint_cache_test.go`. Add the two new imports (`context`, `errors`, `github.com/gagliardetto/solana-go`, `github.com/gagliardetto/solana-go/rpc`) to the existing `import ( "testing" )` block, making it:

```go
// CURRENT — cmd/rpc/oracle/sol/mint_cache_test.go:1-3
package sol

import "testing"

// NEW — cmd/rpc/oracle/sol/mint_cache_test.go
package sol

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)
```

Then append:

```go
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
```

- [ ] **Step 2: Run the new tests**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -run 'TestMintCache_Decimals_' -v`
Expected: all 4 tests PASS. If any FAIL, stop and investigate `mintCache.Decimals()` in `mint_cache.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/rpc/oracle/sol/mint_cache_test.go
git commit -m "test(oracle/sol): cover mintCache.Decimals fetch, cache-hit, and error paths"
```

---

### Task 9: Full package verification

**Files:** none (verification only)

- [ ] **Step 1: Run the entire `sol` package test suite**

Run: `GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/sol/... -v`
Expected: PASS, with all tests from Tasks 1-8 (27 new) plus the 11 pre-existing tests, all green (38 total `func Test` bodies).

- [ ] **Step 2: Run the broader oracle test suite to confirm no regressions**

Run: `GOTOOLCHAIN=go1.25.3 go build ./... && GOTOOLCHAIN=go1.25.3 go test ./cmd/rpc/oracle/... ./lib/ ./controller/...`
Expected: PASS (matches the green baseline recorded in `HANDOFF.md`).

- [ ] **Step 3: Update `HANDOFF.md`**

Mark item 1 under "Next Steps" as done, noting the commits from Tasks 1-8. No code block needed — this is a documentation edit.

---

## Coverage Cross-Check (against `HANDOFF.md` gap list)

| Gap from HANDOFF.md | Task |
|---|---|
| `parseSPLTransfer`: TransferChecked, undersized, missing accounts (tag 3 / 12), unsupported tag | Task 1 |
| `parseNativeTransfer`: happy path, undersized, wrong discriminator, missing accounts | Task 2 |
| `parseTransfer`: SPL precedence, close-with-no-transfer error | Task 3 |
| `parseInstructions`: lock/close UnmarshalJSON failure, memo-no-match no-op, transfer-parse-error propagation | Task 4 |
| `MatchesOrderDestination`: isTransfer false, native mismatch, TransferChecked ATA integration | Task 5 |
| `TokenTransfer()` field mapping, accessors | Task 6 |
| `resolveInstructions`: out-of-range indices, empty | Task 7 |
| `mintCache.Decimals()`: miss→fetch→cache→hit, fetcher error, nil result | Task 8 |
