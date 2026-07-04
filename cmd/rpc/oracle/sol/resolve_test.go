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
			{ProgramIDIndex: 0, Accounts: []uint16{1}, Data: solana.Base58([]byte("hello"))},
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
