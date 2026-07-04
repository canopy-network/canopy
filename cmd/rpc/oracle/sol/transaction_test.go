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

var errNotOrder = errors.New("not an order")

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
