package contract

import (
	"math"
	"math/big"
	"testing"
)

// TestScaledDebt_NormalCase is a regression guard confirming the new
// BitLen() guard (session fix, Handoff Part 2 item 2) does not interfere
// with any realistic, in-range value -- mirroring the live devnet read
// against borrow-test-01 (debtPrincipal 294002 -> currentDebt 294760).
func TestScaledDebt_NormalCase(t *testing.T) {
	borrowIndexAtOpen, encErr := EncodeUint128(big.NewInt(1_000_000_000_000_000_000)) // 1 RAY
	if encErr != nil {
		t.Fatalf("unexpected encode error: %v", encErr)
	}
	pos := &BorrowerPosition{
		MarketId:          "test-market",
		DebtPrincipal:     294002,
		BorrowIndexAtOpen: borrowIndexAtOpen,
	}
	// bIndexNow slightly above 1 RAY, simulating some accrued interest.
	bIndexNow := big.NewInt(0)
	bIndexNow.SetString("1002580000000000000", 10) // ~1.00258 RAY

	got, sdErr := ScaledDebt(pos, bIndexNow)
	if sdErr != nil {
		t.Fatalf("unexpected overflow error on normal-magnitude input: %v", sdErr)
	}
	if got == 0 {
		t.Errorf("ScaledDebt() = 0, want a nonzero value close to debtPrincipal")
	}
	if got < pos.DebtPrincipal {
		t.Errorf("ScaledDebt() = %d, want >= debtPrincipal %d (interest accrual should not decrease debt)", got, pos.DebtPrincipal)
	}
}

// TestScaledDebt_OverflowGuardFires is a deliberately extreme, artificially
// constructed case -- analogous in spirit to ARCM/AYIS's own worked
// examples (Section 14.9/14.10) -- that forces numerator.BitLen() > 64 to
// confirm the guard added this session actually fires, not merely that it
// stays quiet for normal values (already confirmed by the case above and
// by live devnet reads). No realistic borrower position can reach this
// state under the codebase's existing invariants; borrowIndexAtOpen is
// forced to an artificial value of 1 (instead of a real ~1 RAY snapshot)
// specifically to manufacture the overflow.
func TestScaledDebt_OverflowGuardFires(t *testing.T) {
	borrowIndexAtOpen, encErr := EncodeUint128(big.NewInt(1)) // artificial, not realistic
	if encErr != nil {
		t.Fatalf("unexpected encode error: %v", encErr)
	}
	pos := &BorrowerPosition{
		MarketId:          "overflow-test-market",
		DebtPrincipal:     math.MaxUint64,
		BorrowIndexAtOpen: borrowIndexAtOpen,
	}
	// bIndexNow at a normal ~1 RAY scale -- the overflow comes entirely
	// from the artificially tiny borrowIndexAtOpen divisor above, not
	// from an unrealistic B_index.
	bIndexNow := big.NewInt(1_000_000_000_000_000_000)

	got, sdErr := ScaledDebt(pos, bIndexNow)
	if sdErr == nil {
		t.Fatalf("expected ErrScaledDebtOverflow, got nil error and value %d", got)
	}
	if got != 0 {
		t.Errorf("on overflow, want got == 0, got %d", got)
	}
	t.Logf("guard fired correctly: %v", sdErr)
}
