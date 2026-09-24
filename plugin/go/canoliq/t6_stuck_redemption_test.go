package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// t6_stuck_redemption_test.go covers the fourth T6 alert: stuck_redemption.
// The evaluator range-scans the global mature-redemption index (written by
// DeliverMessageCanoliqRedeem and deleted by DeliverMessageCanoliqClaimRedemption)
// and fires when the count of entries whose mature_height ≤ currentHeight
// exceeds StuckRedemptionCount (default 10).

// seedMatureRedemptionEntries directly writes the global mature-redemption
// index entries — bypasses the full deposit/redeem dance because the alert
// only cares about index presence, not the underlying Redemption records.
func seedMatureRedemptionEntries(s *fakeStore, entries []struct {
	matureHeight uint64
	addr         []byte
	id           uint64
}) {
	for _, e := range entries {
		s.set(KeyForMatureRedemption(e.matureHeight, e.addr, e.id), matureRedemptionMarker)
	}
}

// stuckCfg returns an AlertConfig with the threshold overridden so the
// tests can exercise the boundary without seeding hundreds of entries.
func stuckCfg(threshold uint64) *AlertConfig {
	return &AlertConfig{StuckRedemptionCount: threshold}
}

// evalAlertsAt advances the height, refreshes the snapshot, and runs the
// full alert evaluation. Unlike evalAt (t6_alerts_test.go) it does not
// reseed buyback/TVL — those alerts will no-op without baselines.
func evalAlertsAt(t *testing.T, c *Canoliq, height uint64) {
	t.Helper()
	c.plugin.setHeight(height)
	if err := c.refreshSnapshot(height); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := c.evaluateAlerts(height); err != nil {
		t.Fatalf("evaluateAlerts: %v", err)
	}
}

// TestT6StuckRedemptionFires: more mature unclaimed redemptions than the
// threshold triggers exactly one crit alert with count + threshold details.
func TestT6StuckRedemptionFires(t *testing.T) {
	c, s, got := newAlertTest(t)
	c.plugin.config.Alerts = stuckCfg(3)

	// Four entries all mature by height 10 (threshold is 3 → 4 > 3 fires).
	seedMatureRedemptionEntries(s, []struct {
		matureHeight uint64
		addr         []byte
		id           uint64
	}{
		{5, addr20(0x01), 1},
		{6, addr20(0x02), 2},
		{7, addr20(0x03), 3},
		{8, addr20(0x04), 4},
	})

	evalAlertsAt(t, c, 10)
	if countKind(*got, AlertStuckRedemption) != 1 {
		t.Fatalf("expected 1 stuck-redemption alert, got %d", countKind(*got, AlertStuckRedemption))
	}
	e := (*got)[len(*got)-1]
	if e.Severity != severityCrit {
		t.Errorf("severity: got %s want %s", e.Severity, severityCrit)
	}
	if e.Details["count"] != uint64(4) {
		t.Errorf("count: got %v want 4", e.Details["count"])
	}
	if e.Details["thresholdCount"] != uint64(3) {
		t.Errorf("thresholdCount: got %v want 3", e.Details["thresholdCount"])
	}
}

// TestT6StuckRedemptionThresholdEdge: exactly threshold entries does not
// fire (`count > threshold` semantics); one more does.
func TestT6StuckRedemptionThresholdEdge(t *testing.T) {
	// Exactly threshold → no fire.
	c, s, got := newAlertTest(t)
	c.plugin.config.Alerts = stuckCfg(3)
	seedMatureRedemptionEntries(s, []struct {
		matureHeight uint64
		addr         []byte
		id           uint64
	}{
		{1, addr20(0x01), 1},
		{1, addr20(0x02), 2},
		{1, addr20(0x03), 3},
	})
	evalAlertsAt(t, c, 10)
	if n := countKind(*got, AlertStuckRedemption); n != 0 {
		t.Errorf("at-threshold should not fire, got %d", n)
	}

	// Threshold + 1 → fires.
	c2, s2, got2 := newAlertTest(t)
	c2.plugin.config.Alerts = stuckCfg(3)
	seedMatureRedemptionEntries(s2, []struct {
		matureHeight uint64
		addr         []byte
		id           uint64
	}{
		{1, addr20(0x01), 1},
		{1, addr20(0x02), 2},
		{1, addr20(0x03), 3},
		{1, addr20(0x04), 4},
	})
	evalAlertsAt(t, c2, 10)
	if n := countKind(*got2, AlertStuckRedemption); n != 1 {
		t.Errorf("just-above-threshold should fire, got %d", n)
	}
}

// TestT6StuckRedemptionImmatureNotCounted: entries whose mature_height is
// still in the future are not counted toward the alert.
func TestT6StuckRedemptionImmatureNotCounted(t *testing.T) {
	c, s, got := newAlertTest(t)
	c.plugin.config.Alerts = stuckCfg(2)

	// Three entries, but only one matured at height=10.
	seedMatureRedemptionEntries(s, []struct {
		matureHeight uint64
		addr         []byte
		id           uint64
	}{
		{5, addr20(0x01), 1},   // mature
		{20, addr20(0x02), 2},  // immature at h=10
		{100, addr20(0x03), 3}, // immature at h=10
	})

	evalAlertsAt(t, c, 10)
	if n := countKind(*got, AlertStuckRedemption); n != 0 {
		t.Errorf("immature entries should not be counted; expected no fire, got %d", n)
	}

	// Advance past maturity of the second and third entries → fires.
	evalAlertsAt(t, c, 200)
	if n := countKind(*got, AlertStuckRedemption); n != 1 {
		t.Errorf("after maturation 3 > 2 should fire; got %d", n)
	}
}

// TestT6StuckRedemptionDebounceAndResolution: the alert fires once, stays
// quiet within the min-interval window, then re-fires after resolution
// (count back at/below threshold) clears the watermark.
func TestT6StuckRedemptionDebounceAndResolution(t *testing.T) {
	c, s, got := newAlertTest(t)
	c.plugin.config.Alerts = stuckCfg(2)

	// 3 mature entries → fires.
	seedMatureRedemptionEntries(s, []struct {
		matureHeight uint64
		addr         []byte
		id           uint64
	}{
		{1, addr20(0x01), 1},
		{1, addr20(0x02), 2},
		{1, addr20(0x03), 3},
	})

	evalAlertsAt(t, c, 10) // fires
	evalAlertsAt(t, c, 11) // within debounce → no re-fire
	if n := countKind(*got, AlertStuckRedemption); n != 1 {
		t.Fatalf("debounce: expected 1 fire, got %d", n)
	}

	// Resolve: drop count to threshold (2) — watermark clears.
	s.del(KeyForMatureRedemption(1, addr20(0x03), 3))
	evalAlertsAt(t, c, 12)
	if n := countKind(*got, AlertStuckRedemption); n != 1 {
		t.Fatalf("at-threshold after delete should not re-fire; got %d", n)
	}

	// Re-cross by adding a new mature entry → fires (watermark was cleared).
	s.set(KeyForMatureRedemption(1, addr20(0x05), 5), matureRedemptionMarker)
	evalAlertsAt(t, c, 13)
	if n := countKind(*got, AlertStuckRedemption); n != 2 {
		t.Fatalf("post-resolution re-cross: expected 2 total fires, got %d", n)
	}
}

// TestStuckRedemptionIndexLifecycle: the full redeem → claim flow writes
// the global mature-redemption index entry on redeem and deletes it on
// claim. End-to-end coverage of the deliver.go plumbing that backs the
// alert evaluator.
func TestStuckRedemptionIndexLifecycle(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x21)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000}
	gBz, _ := contract.Marshal(g)
	s.set(KeyForGlobals(), gBz)
	s.set(KeyForCCNPYBalance(user), EncodeUint64(1000))
	seedEscrow(s, 1000) // backs the pre-seeded TotalPooledCnpy (H1)
	seedAccount(s, user, 100_000)

	// Redeem → mature-redemption index entry written.
	if resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 100},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}

	// Recover the redemption record so we know its UnbondCompleteHeight.
	bz := s.get(KeyForRedemption(user, 0))
	if bz == nil {
		t.Fatal("redemption record not written")
	}
	red := new(contract.Redemption)
	if err := contract.Unmarshal(bz, red); err != nil {
		t.Fatalf("unmarshal redemption: %v", err)
	}
	mkey := KeyForMatureRedemption(red.UnbondCompleteHeight, user, 0)
	if got := s.get(mkey); got == nil {
		t.Fatal("mature-redemption index entry not written on redeem")
	}

	// Advance past maturity and claim → index entry deleted.
	c.plugin.setHeight(100)
	if resp := c.DeliverMessageCanoliqClaimRedemption(
		&contract.MessageCanoliqClaimRedemption{FromAddress: user, RedemptionId: 0},
		10_000, DefaultParams(),
	); resp.Error != nil {
		t.Fatalf("claim: %v", resp.Error)
	}
	if got := s.get(mkey); got != nil {
		t.Fatalf("mature-redemption index entry should be deleted on claim, got %x", got)
	}
}
