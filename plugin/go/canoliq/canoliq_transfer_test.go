package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// canoliq_transfer_test.go covers MessageCanoliqTransfer: a pure
// internal-balance move of cCNPY between two accounts. Mirrors
// TestCPLQTransferRespectsLiquidBalance (canoliq_test.go) on the cCNPY
// balance prefix instead of CPLQ's.

// TestCanoliqTransferRespectsLiquidBalance: over-balance fails, a
// within-balance transfer moves the exact amount and nothing else.
func TestCanoliqTransferRespectsLiquidBalance(t *testing.T) {
	c, s := newTestCanoliq()
	from := addr20(0x05)
	to := addr20(0x06)
	seedAccount(s, from, 10_000) // CNPY for fee
	s.set(KeyForCCNPYBalance(from), EncodeUint64(500))

	// Over-transfer fails.
	resp := c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 1000},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("transfer of 1000 from balance 500 should fail")
	}

	// Within-balance succeeds.
	resp = c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 200},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("transfer error: %v", resp.Error)
	}
	if readCcnpy(s, from) != 300 || readCcnpy(s, to) != 200 {
		t.Fatalf("post-transfer balances: from=%d to=%d (want 300/200)",
			readCcnpy(s, from), readCcnpy(s, to))
	}
}

// TestCanoliqTransferRejectsSelfTransfer is the regression test for the
// review finding on #37: fromBalKey and toBalKey alias to the same state key
// when from_address == to_address. DeliverMessageCanoliqTransfer reads that
// one key into both fromBal and toBal, then writes both back — the FSM
// applies all Sets before Deletes (fsm/state.go), so the second write wins
// and the balance is destroyed rather than left unchanged: a partial
// self-transfer loses `amount`, a full self-transfer loses everything (the
// toBalKey set is immediately followed by the fromBal==0 branch deleting
// that same key). Both Check and Deliver must reject this outright.
func TestCanoliqTransferRejectsSelfTransfer(t *testing.T) {
	c, s := newTestCanoliq()
	self := addr20(0x0f)
	p := DefaultParams()

	if resp := c.CheckMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: self, ToAddress: self, Amount: 100}, p.CanoliqTransferFee, p,
	); resp.Error == nil {
		t.Error("CheckTx should reject from_address == to_address")
	}

	// Partial: balance must survive at its original value, not amount-less.
	seedAccount(s, self, 10_000)
	s.set(KeyForCCNPYBalance(self), EncodeUint64(500))
	resp := c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: self, ToAddress: self, Amount: 200}, 10_000, p,
	)
	if resp.Error == nil {
		t.Fatal("Deliver should reject a self-transfer rather than silently destroying balance")
	}
	if readCcnpy(s, self) != 500 {
		t.Fatalf("self-transfer must not touch the balance: got %d, want 500", readCcnpy(s, self))
	}

	// Full: the more destructive case (would have zeroed the balance entirely).
	full := addr20(0x10)
	seedAccount(s, full, 10_000)
	s.set(KeyForCCNPYBalance(full), EncodeUint64(500))
	resp = c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: full, ToAddress: full, Amount: 500}, 10_000, p,
	)
	if resp.Error == nil {
		t.Fatal("Deliver should reject a full-balance self-transfer")
	}
	if readCcnpy(s, full) != 500 {
		t.Fatalf("full self-transfer must not zero the balance: got %d, want 500", readCcnpy(s, full))
	}
}

// TestCanoliqTransferDoesNotTouchPoolAccounting is the point of this
// message's design: a transfer redistributes an existing claim on the pool,
// it must never mint, burn, or otherwise move totalCcnpySupply /
// totalPooledCnpy — those only change via deposit/redeem. If this test ever
// fails, the transfer handler has started interacting with the exact
// accounting #34/#36 exist to protect.
func TestCanoliqTransferDoesNotTouchPoolAccounting(t *testing.T) {
	c, s := newTestCanoliq()
	from := addr20(0x07)
	to := addr20(0x08)
	seedAccount(s, from, 10_000)
	s.set(KeyForCCNPYBalance(from), EncodeUint64(500))
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete:  true,
		TotalCcnpySupply: 500,
		TotalPooledCnpy:  777, // arbitrary — must survive untouched
	})

	resp := c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 200},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("transfer error: %v", resp.Error)
	}
	g := loadGlobals(t, s)
	if g.TotalCcnpySupply != 500 {
		t.Errorf("totalCcnpySupply changed: got %d, want 500 (transfer must not mint/burn)", g.TotalCcnpySupply)
	}
	if g.TotalPooledCnpy != 777 {
		t.Errorf("totalPooledCnpy changed: got %d, want 777 (transfer must not touch the pool)", g.TotalPooledCnpy)
	}
}

// TestCanoliqTransferZeroBalanceKeyDeleted confirms the sender's balance key
// is deleted (not left as a zero-value entry) once fully drained — same
// storage-hygiene convention as redeem and CPLQ transfer.
func TestCanoliqTransferZeroBalanceKeyDeleted(t *testing.T) {
	c, s := newTestCanoliq()
	from := addr20(0x09)
	to := addr20(0x0a)
	seedAccount(s, from, 10_000)
	s.set(KeyForCCNPYBalance(from), EncodeUint64(200))

	resp := c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 200},
		10_000,
		DefaultParams(),
	)
	if resp.Error != nil {
		t.Fatalf("transfer error: %v", resp.Error)
	}
	if s.get(KeyForCCNPYBalance(from)) != nil {
		t.Error("drained sender balance key should be deleted, not left as a zero entry")
	}
	if readCcnpy(s, to) != 200 {
		t.Fatalf("recipient balance: got %d, want 200", readCcnpy(s, to))
	}
}

// TestCanoliqTransferInsufficientCNPYForFee: the fee is paid from the
// sender's native CNPY balance, not their cCNPY — a sender with plenty of
// cCNPY but no CNPY for the fee must be rejected, same as every other
// canoliq message.
func TestCanoliqTransferInsufficientCNPYForFee(t *testing.T) {
	c, s := newTestCanoliq()
	from := addr20(0x0b)
	to := addr20(0x0c)
	seedAccount(s, from, 5_000) // below the 10_000 fee
	s.set(KeyForCCNPYBalance(from), EncodeUint64(1_000_000))

	resp := c.DeliverMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 100},
		10_000,
		DefaultParams(),
	)
	if resp.Error == nil {
		t.Fatal("transfer with insufficient CNPY for the fee should fail")
	}
}

// TestCheckMessageCanoliqTransfer covers the stateless CheckTx validation:
// address shape, zero amount, and fee floor.
func TestCheckMessageCanoliqTransfer(t *testing.T) {
	c, _ := newTestCanoliq()
	from := addr20(0x0d)
	to := addr20(0x0e)
	p := DefaultParams()

	if resp := c.CheckMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from[:19], ToAddress: to, Amount: 100}, p.CanoliqTransferFee, p,
	); resp.Error == nil {
		t.Error("short from_address should be rejected")
	}
	if resp := c.CheckMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 0}, p.CanoliqTransferFee, p,
	); resp.Error == nil {
		t.Error("zero amount should be rejected")
	}
	if resp := c.CheckMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 100}, p.CanoliqTransferFee-1, p,
	); resp.Error == nil {
		t.Error("fee below the configured floor should be rejected")
	}
	resp := c.CheckMessageCanoliqTransfer(
		&contract.MessageCanoliqTransfer{FromAddress: from, ToAddress: to, Amount: 100}, p.CanoliqTransferFee, p,
	)
	if resp.Error != nil {
		t.Fatalf("valid transfer check: %v", resp.Error)
	}
	if string(resp.Recipient) != string(to) {
		t.Errorf("recipient: got %x, want %x", resp.Recipient, to)
	}
	if len(resp.AuthorizedSigners) != 1 || string(resp.AuthorizedSigners[0]) != string(from) {
		t.Errorf("authorized signers: got %v, want [%x]", resp.AuthorizedSigners, from)
	}
}
