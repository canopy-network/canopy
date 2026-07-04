package sol

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
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

// newTestTransaction is a thin wrapper matching the test call sites.
func newTestTransaction(sig string, instrs []instruction, _ interface{}) *Transaction {
	return newTransaction(sig, "FeePayer1111111111111111111111111111111111", instrs)
}

// Helper to create byte array for u64 little-endian encoding
func u64LE(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

var errNotOrder = errors.New("not an order")

// lockOrderJSONFixture produces bytes that (*lib.LockOrder).UnmarshalJSON accepts
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

// closeOrderJSONFixture produces bytes that (*lib.CloseOrder).UnmarshalJSON accepts
func closeOrderJSONFixture(t *testing.T) []byte {
	t.Helper()
	co := &lib.CloseOrder{
		OrderId:    []byte{0x01, 0x02, 0x03},
		ChainId:    1,
		CloseOrder: true,
	}
	b, err := co.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal close order fixture: %v", err)
	}
	return b
}

// A minimal valid lock order JSON is what the real validator would accept; here the fake
// accepts everything, and we assert the parser routes a Memo-only tx to a lock order.
func TestParse_MemoOnly_IsLockOrder(t *testing.T) {
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
