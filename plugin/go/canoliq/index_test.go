package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// loadRedemptionIndex reads the per-address redemption index from fake state.
// Returns nil when the key is absent (which is the empty-set encoding).
func loadRedemptionIndex(t *testing.T, s *fakeStore, addr []byte) *contract.RedemptionIndex {
	t.Helper()
	bz := s.get(KeyForRedemptionIndex(addr))
	if bz == nil {
		return nil
	}
	idx := new(contract.RedemptionIndex)
	if err := contract.Unmarshal(bz, idx); err != nil {
		t.Fatalf("unmarshal redemption index: %v", err)
	}
	return idx
}

// loadUnstakingIndex reads the per-address unstake index from fake state.
func loadUnstakingIndex(t *testing.T, s *fakeStore, addr []byte) *contract.UnstakingIndex {
	t.Helper()
	bz := s.get(KeyForUnstakingIndex(addr))
	if bz == nil {
		return nil
	}
	idx := new(contract.UnstakingIndex)
	if err := contract.Unmarshal(bz, idx); err != nil {
		t.Fatalf("unmarshal unstaking index: %v", err)
	}
	return idx
}

// TestRedemptionIndexAppendOnRedeem covers the write-side invariant: each
// successful redeem appends its new redemption id to the per-address index.
func TestRedemptionIndexAppendOnRedeem(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x10)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	seedAccount(s, user, 100_000)

	for i := 0; i < 3; i++ {
		resp := c.DeliverMessageCanoliqRedeem(
			&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 100},
			10_000, DefaultParams(),
		)
		if resp.Error != nil {
			t.Fatalf("redeem %d: %v", i, resp.Error)
		}
	}

	idx := loadRedemptionIndex(t, s, user)
	if idx == nil {
		t.Fatal("redemption index not written")
	}
	want := []uint64{0, 1, 2}
	if len(idx.Ids) != len(want) {
		t.Fatalf("index len: got %d want %d (ids=%v)", len(idx.Ids), len(want), idx.Ids)
	}
	for i, id := range want {
		if idx.Ids[i] != id {
			t.Errorf("idx[%d]: got %d want %d", i, idx.Ids[i], id)
		}
	}
}

// TestRedemptionIndexRemoveOnClaim covers the matching write-side invariant
// on the claim path: the matured id is removed from the index, and when the
// last id is claimed the index key itself is deleted (empty-set encoding).
func TestRedemptionIndexRemoveOnClaim(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x11)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	seedAccount(s, user, 100_000)

	if resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 100},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}

	// Advance past maturity, claim, and the index must end empty.
	c.plugin.setHeight(100)
	if resp := c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 0},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}

	if idx := loadRedemptionIndex(t, s, user); idx != nil {
		t.Fatalf("redemption index should be deleted when empty, got %+v", idx)
	}
}

// TestRedemptionIndexOutOfOrderClaims verifies that partial / out-of-order
// claims keep the index consistent: claiming a middle id drops just that
// entry; claiming the rest empties the index.
func TestRedemptionIndexOutOfOrderClaims(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x12)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	seedAccount(s, user, 1_000_000)

	for i := 0; i < 3; i++ {
		if resp := c.DeliverMessageCanoliqRedeem(
			&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 100},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("redeem %d: %v", i, resp.Error)
		}
	}
	c.plugin.setHeight(100)

	// Claim id 1 (the middle one). Index should retain [0, 2].
	if resp := c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 1},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim middle: %v", resp.Error)
	}
	idx := loadRedemptionIndex(t, s, user)
	if idx == nil || len(idx.Ids) != 2 || idx.Ids[0] != 0 || idx.Ids[1] != 2 {
		t.Fatalf("after middle claim: got %+v want [0 2]", idx)
	}

	// Claim id 2, then id 0. Order doesn't matter; index must end empty.
	for _, id := range []uint64{2, 0} {
		if resp := c.DeliverMessageCanoliqClaimRedemption(
			&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: id},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("claim %d: %v", id, resp.Error)
		}
	}
	if idx := loadRedemptionIndex(t, s, user); idx != nil {
		t.Fatalf("index should be deleted after all claims, got %+v", idx)
	}
}

// TestUnstakingIndexAppendOnUnstake covers the analogous write-side
// invariant for unstakes.
func TestUnstakingIndexAppendOnUnstake(t *testing.T) {
	c, s := newTestCanoliq()
	staker := addr20(0x20)
	seedParams(t, c, DefaultParams())
	seedAccount(s, staker, 1_000_000)
	seedCPLQ(s, staker, 30_000_000)

	if resp := c.DeliverMessageCPLQStake(
		&contract.MessageCPLQStake{FromAddress: staker, Amount: 30_000_000},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("stake: %v", resp.Error)
	}

	for i := 0; i < 3; i++ {
		if resp := c.DeliverMessageCPLQUnstake(
			&contract.MessageCPLQUnstake{FromAddress: staker, Amount: 1_000_000},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("unstake %d: %v", i, resp.Error)
		}
	}

	idx := loadUnstakingIndex(t, s, staker)
	if idx == nil {
		t.Fatal("unstaking index not written")
	}
	want := []uint64{0, 1, 2}
	if len(idx.Ids) != len(want) {
		t.Fatalf("index len: got %d want %d (ids=%v)", len(idx.Ids), len(want), idx.Ids)
	}
	for i, id := range want {
		if idx.Ids[i] != id {
			t.Errorf("idx[%d]: got %d want %d", i, idx.Ids[i], id)
		}
	}
}

// TestUnstakingIndexRemoveOnClaim covers the matching write-side invariant
// on claim: matured id removed; empty index key deleted.
func TestUnstakingIndexRemoveOnClaim(t *testing.T) {
	c, s := newTestCanoliq()
	staker := addr20(0x21)
	seedParams(t, c, DefaultParams())
	seedAccount(s, staker, 1_000_000)
	seedCPLQ(s, staker, 10_000_000)

	if resp := c.DeliverMessageCPLQStake(
		&contract.MessageCPLQStake{FromAddress: staker, Amount: 10_000_000},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("stake: %v", resp.Error)
	}
	if resp := c.DeliverMessageCPLQUnstake(
		&contract.MessageCPLQUnstake{FromAddress: staker, Amount: 5_000_000},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("unstake: %v", resp.Error)
	}

	// Advance past the unstake maturity window and claim.
	c.plugin.setHeight(DefaultParams().CplqUnstakingBlocks + 10)
	if resp := c.DeliverMessageCPLQClaimUnstake(
		&contract.MessageCPLQClaimUnstake{FromAddress: staker, UnstakeId: 0},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}

	if idx := loadUnstakingIndex(t, s, staker); idx != nil {
		t.Fatalf("unstaking index should be deleted when empty, got %+v", idx)
	}
}

// TestUnstakingIndexOutOfOrderClaims mirrors TestRedemptionIndexOutOfOrderClaims.
func TestUnstakingIndexOutOfOrderClaims(t *testing.T) {
	c, s := newTestCanoliq()
	staker := addr20(0x22)
	seedParams(t, c, DefaultParams())
	seedAccount(s, staker, 1_000_000)
	seedCPLQ(s, staker, 30_000_000)

	if resp := c.DeliverMessageCPLQStake(
		&contract.MessageCPLQStake{FromAddress: staker, Amount: 30_000_000},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("stake: %v", resp.Error)
	}
	for i := 0; i < 3; i++ {
		if resp := c.DeliverMessageCPLQUnstake(
			&contract.MessageCPLQUnstake{FromAddress: staker, Amount: 1_000_000},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("unstake %d: %v", i, resp.Error)
		}
	}

	c.plugin.setHeight(DefaultParams().CplqUnstakingBlocks + 10)
	if resp := c.DeliverMessageCPLQClaimUnstake(
		&contract.MessageCPLQClaimUnstake{FromAddress: staker, UnstakeId: 1},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim middle: %v", resp.Error)
	}
	idx := loadUnstakingIndex(t, s, staker)
	if idx == nil || len(idx.Ids) != 2 || idx.Ids[0] != 0 || idx.Ids[1] != 2 {
		t.Fatalf("after middle claim: got %+v want [0 2]", idx)
	}

	for _, id := range []uint64{2, 0} {
		if resp := c.DeliverMessageCPLQClaimUnstake(
			&contract.MessageCPLQClaimUnstake{FromAddress: staker, UnstakeId: id},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("claim %d: %v", id, resp.Error)
		}
	}
	if idx := loadUnstakingIndex(t, s, staker); idx != nil {
		t.Fatalf("index should be deleted after all claims, got %+v", idx)
	}
}

// TestRemoveUint64Idempotent covers the helper: removing an id that's not
// in the slice is a no-op. The claim path doesn't reach this case under
// normal flow, but the helper must not panic or rewrite the slice.
func TestRemoveUint64Idempotent(t *testing.T) {
	in := []uint64{1, 2, 3}
	out := removeUint64(in, 99)
	if len(out) != len(in) {
		t.Fatalf("len changed: got %d want %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("out[%d]: got %d want %d", i, out[i], in[i])
		}
	}
	// Empty input.
	if got := removeUint64(nil, 5); got != nil {
		t.Errorf("nil input → nil output, got %v", got)
	}
}

// TestAccountViewIncludesRedemptionsAndUnstakes verifies the lazy-query
// account composite picks up both new collections in correct order.
func TestAccountViewIncludesRedemptionsAndUnstakes(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x30)
	seedParams(t, c, DefaultParams())
	seedAccount(s, user, 10_000_000)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	seedCPLQ(s, user, 20_000_000)

	// Two redeems + two unstakes (after staking).
	for i := 0; i < 2; i++ {
		if resp := c.DeliverMessageCanoliqRedeem(
			&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 100},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("redeem %d: %v", i, resp.Error)
		}
	}
	if resp := c.DeliverMessageCPLQStake(
		&contract.MessageCPLQStake{FromAddress: user, Amount: 20_000_000},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("stake: %v", resp.Error)
	}
	for i := 0; i < 2; i++ {
		if resp := c.DeliverMessageCPLQUnstake(
			&contract.MessageCPLQUnstake{FromAddress: user, Amount: 1_000_000},
			10_000, DefaultParams(),
		); resp.Error != nil {
			t.Fatalf("unstake %d: %v", i, resp.Error)
		}
	}

	view, err := c.buildAccountView(user)
	if err != nil {
		t.Fatalf("buildAccountView: %v", err)
	}
	if len(view.Redemptions) != 2 {
		t.Fatalf("redemptions: got %d want 2 (%+v)", len(view.Redemptions), view.Redemptions)
	}
	if view.Redemptions[0].Id != 0 || view.Redemptions[1].Id != 1 {
		t.Errorf("redemption ids: got %d,%d want 0,1",
			view.Redemptions[0].Id, view.Redemptions[1].Id)
	}
	if len(view.Unstakes) != 2 {
		t.Fatalf("unstakes: got %d want 2 (%+v)", len(view.Unstakes), view.Unstakes)
	}
	if view.Unstakes[0].Id != 0 || view.Unstakes[1].Id != 1 {
		t.Errorf("unstake ids: got %d,%d want 0,1",
			view.Unstakes[0].Id, view.Unstakes[1].Id)
	}
}
