package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// t3_tvlcap_test.go covers the self-imposed percentage TVL cap (WP §9.4):
// deposit accepted at exactly the computed cap, rejected past it; cap
// lifting via params or via observing more Canopy stake; the uncapped
// default (TvlCapBps = 0); /v1/health exposing the effective cap and
// utilization; and the fail-closed posture when Canopy total stake is
// unavailable.

// hotPoolGlobals returns 1:1-rate globals with the given pooled CNPY so a
// deposit mints an equal amount of cCNPY.
func hotPoolGlobals(pooled uint64) *contract.CanoliqGlobals {
	return &contract.CanoliqGlobals{
		TotalCcnpySupply: pooled,
		TotalPooledCnpy:  pooled,
		GenesisComplete:  true,
	}
}

// seedCanopySupply writes a Supply singleton at contract.KeyForSupply() so
// the deposit / health paths can observe Canopy total stake.
func seedCanopySupply(t *testing.T, s *fakeStore, staked uint64) {
	t.Helper()
	bz, err := contract.Marshal(&contract.Supply{Staked: staked})
	if err != nil {
		t.Fatalf("marshal supply: %v", err)
	}
	s.set(contract.KeyForSupply(), bz)
}

// cappedParams returns DefaultParams with TvlCapBps overridden.
func cappedParams(bps uint64) *contract.CanoliqParams {
	p := DefaultParams()
	p.TvlCapBps = bps
	return p
}

// TestT3DepositCapBoundary: a deposit landing exactly on the computed
// cap is allowed; one uCNPY more is rejected.
// Total Canopy stake 100M × bps 3300 → effective cap 33M. Pooled 25M +
// deposit 8M = exactly 33M → accepted; deposit 8M + 1 → rejected.
func TestT3DepositCapBoundary(t *testing.T) {
	const (
		canopyStake uint64 = 100_000_000
		pooled      uint64 = 25_000_000
		capBps      uint64 = 3_300 // 33%
		// effective cap = 100_000_000 * 3300 / 10_000 = 33_000_000
	)
	user := addr20(0x01)

	// Exactly to cap (25M + 8M == 33M) → accepted.
	c, s := newTestCanoliq()
	seedCanopySupply(t, s, canopyStake)
	seedGlobals(s, hotPoolGlobals(pooled))
	seedAccount(s, user, 100_000_000)
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 8_000_000}, 10_000, cappedParams(capBps)); r.Error != nil {
		t.Fatalf("deposit to exact cap should succeed: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 33_000_000 {
		t.Errorf("pooled after deposit: got %d want 33_000_000", g.TotalPooledCnpy)
	}

	// One uCNPY above cap → rejected.
	c2, s2 := newTestCanoliq()
	seedCanopySupply(t, s2, canopyStake)
	seedGlobals(s2, hotPoolGlobals(pooled))
	seedAccount(s2, user, 100_000_000)
	r := c2.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 8_000_001}, 10_000, cappedParams(capBps))
	if r.Error == nil {
		t.Fatal("deposit above cap should be rejected")
	}
	if r.Error.Code != codeTVLCapExceeded {
		t.Fatalf("expected codeTVLCapExceeded, got %d", r.Error.Code)
	}
	// State unchanged on rejection.
	if g := loadGlobals(t, s2); g.TotalPooledCnpy != pooled {
		t.Errorf("pooled should be unchanged after rejection: got %d want %d", g.TotalPooledCnpy, pooled)
	}
}

// TestT3LiftCapReenablesDeposits: a deposit rejected at the current cap
// is accepted once the cap bps is raised (governance lift, per WP §9.4
// "pending ecosystem maturation and governance approval to lift").
func TestT3LiftCapReenablesDeposits(t *testing.T) {
	const (
		canopyStake uint64 = 100_000_000
		pooled      uint64 = 32_000_000
	)
	user := addr20(0x02)
	c, s := newTestCanoliq()
	seedCanopySupply(t, s, canopyStake)
	seedGlobals(s, hotPoolGlobals(pooled))
	seedAccount(s, user, 100_000_000)

	// 32M + 2M = 34M > 33% cap (33M) → rejected.
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 2_000_000}, 10_000, cappedParams(3_300)); r.Error == nil {
		t.Fatal("deposit above 33% cap should be rejected")
	}
	// Lift cap to 40% → effective cap 40M → 32M + 2M = 34M ≤ 40M → accepted.
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 2_000_000}, 10_000, cappedParams(4_000)); r.Error != nil {
		t.Fatalf("deposit after lifting cap should succeed: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 34_000_000 {
		t.Errorf("pooled after lifted-cap deposit: got %d want 34_000_000", g.TotalPooledCnpy)
	}
}

// TestT3UncappedAllowsLargeDeposit: TvlCapBps = 0 imposes no ceiling and
// short-circuits the Supply read entirely (works even with no Supply seeded).
func TestT3UncappedAllowsLargeDeposit(t *testing.T) {
	user := addr20(0x03)
	c, s := newTestCanoliq()
	// Note: no seedCanopySupply call — uncapped means the Supply read is
	// skipped, so absence is irrelevant.
	seedGlobals(s, hotPoolGlobals(1_000_000))
	seedAccount(s, user, 1_000_000_000)
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 900_000_000}, 10_000, cappedParams(0)); r.Error != nil {
		t.Fatalf("uncapped deposit should succeed: %v", r.Error)
	}
}

// TestT3FailClosedOnAbsentSupply: with TvlCapBps > 0 and the Canopy Supply
// singleton ABSENT from state, deposits are rejected with
// codeCanopyStakeUnavailable. WP §9.4 makes the cap a safety floor; if
// the cap-policy state itself hasn't initialized we'd rather reject than
// silently bypass.
func TestT3FailClosedOnAbsentSupply(t *testing.T) {
	user := addr20(0x04)
	c, s := newTestCanoliq()
	// Delete the fixture's default Supply to exercise the absent path.
	s.del(contract.KeyForSupply())
	seedGlobals(s, hotPoolGlobals(10_000_000))
	seedAccount(s, user, 100_000_000)

	r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000}, 10_000, cappedParams(3_300))
	if r.Error == nil {
		t.Fatal("deposit with absent Supply should be rejected (fail-closed)")
	}
	if r.Error.Code != codeCanopyStakeUnavailable {
		t.Fatalf("expected codeCanopyStakeUnavailable, got %d", r.Error.Code)
	}
	// State must be unchanged.
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 10_000_000 {
		t.Errorf("pooled changed on fail-closed reject: got %d want 10_000_000", g.TotalPooledCnpy)
	}
}

// TestT3AcceptsWhenCapTruncatesToZero: integer-truncation edge — at
// very low Canopy stake, mulDiv(staked, bps, 10000) truncates to 0
// (e.g. Staked=1 with bps=3300 → cap=0). Treated same as Staked=0:
// accept this block; the cap re-engages once Canopy stake grows past
// the truncation point. Without this guard, Staked=0 would accept
// while Staked=1..3 would reject everything — an asymmetric quirk.
func TestT3AcceptsWhenCapTruncatesToZero(t *testing.T) {
	user := addr20(0x06)
	c, s := newTestCanoliq()
	// Staked=1, bps=3300 → mulDiv = 1*3300/10000 = 0 (integer truncation).
	seedCanopySupply(t, s, 1)
	seedGlobals(s, hotPoolGlobals(10_000_000))
	seedAccount(s, user, 100_000_000)

	r := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000},
		10_000, cappedParams(3_300),
	)
	if r.Error != nil {
		t.Fatalf("deposit at truncate-to-zero cap should be accepted: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 11_000_000 {
		t.Errorf("pooled after deposit: got %d want 11_000_000", g.TotalPooledCnpy)
	}

	// Sanity: once Staked is large enough for capUcnpy > 0, the cap
	// engages normally. Staked=10000, bps=3300 → cap=3300. A 1_000
	// deposit (pooled 11_000_000 + 1_000) far exceeds 3300, so rejected.
	c2, s2 := newTestCanoliq()
	seedCanopySupply(t, s2, 10_000)
	seedGlobals(s2, hotPoolGlobals(10_000_000))
	seedAccount(s2, user, 100_000_000)

	r2 := c2.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000},
		10_000, cappedParams(3_300),
	)
	if r2.Error == nil || r2.Error.Code != codeTVLCapExceeded {
		t.Errorf("cap should engage once it truncates to non-zero; got err=%v", r2.Error)
	}
}

// TestT3AcceptsWhenSupplyPresentButStakedZero: Supply present but
// .Staked == 0 means Canopy is up and tracking, just nobody has staked
// yet (legitimate fresh-network state). The cap is uncapped this block
// — deposits go through. The cap re-engages automatically once staking
// begins. (Rejecting here would brick canoLiq on every fresh genesis.)
func TestT3AcceptsWhenSupplyPresentButStakedZero(t *testing.T) {
	user := addr20(0x05)
	c, s := newTestCanoliq()
	seedCanopySupply(t, s, 0)
	seedGlobals(s, hotPoolGlobals(10_000_000))
	seedAccount(s, user, 100_000_000)

	r := c.DeliverMessageCanoliqDeposit(
		&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000},
		10_000, cappedParams(3_300),
	)
	if r.Error != nil {
		t.Fatalf("deposit with Supply.Staked=0 should be accepted: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 11_000_000 {
		t.Errorf("pooled after deposit: got %d want 11_000_000", g.TotalPooledCnpy)
	}
}

// TestT3HealthSurfacesEffectiveCap exercises the /v1/health rewrite:
// effective cap = mulDiv(canopy_total_stake, tvl_cap_bps, 10000),
// utilization computed against the effective cap, and zero values when
// uncapped or when Canopy stake is missing.
func TestT3HealthSurfacesEffectiveCap(t *testing.T) {
	c, s := newTestCanoliq()
	seedCanopySupply(t, s, 100_000_000) // canopy stake 100M
	seedParams(t, c, cappedParams(3_300))
	seedGlobals(s, &contract.CanoliqGlobals{TotalPooledCnpy: 8_250_000, GenesisComplete: true})
	if err := c.refreshSnapshot(5); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	h := c.plugin.QueryHealth()
	if h.TVLCapBps != 3_300 {
		t.Errorf("tvlCapBps: got %d want 3300", h.TVLCapBps)
	}
	if h.CanopyTotalStake != 100_000_000 {
		t.Errorf("canopyTotalStake: got %d want 100_000_000", h.CanopyTotalStake)
	}
	if h.TVLCapUcnpyEffective != 33_000_000 { // 100M * 3300 / 10000
		t.Errorf("tvlCapUcnpyEffective: got %d want 33_000_000", h.TVLCapUcnpyEffective)
	}
	if h.TVLUtilizationBps != 2_500 { // 8.25M / 33M = 25%
		t.Errorf("tvlUtilizationBps: got %d want 2500", h.TVLUtilizationBps)
	}

	// Uncapped → cap fields zero, utilization zero.
	seedParams(t, c, cappedParams(0))
	if err := c.refreshSnapshot(6); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	h = c.plugin.QueryHealth()
	if h.TVLCapBps != 0 || h.TVLCapUcnpyEffective != 0 || h.TVLUtilizationBps != 0 {
		t.Errorf("uncapped health: bps=%d eff=%d util=%d want 0/0/0",
			h.TVLCapBps, h.TVLCapUcnpyEffective, h.TVLUtilizationBps)
	}

	// Capped but Canopy stake absent → effective cap reports zero
	// (deposit path would fail closed; health surface signals "unknown").
	c2, s2 := newTestCanoliq()
	s2.del(contract.KeyForSupply())
	seedParams(t, c2, cappedParams(3_300))
	seedGlobals(s2, &contract.CanoliqGlobals{TotalPooledCnpy: 1_000, GenesisComplete: true})
	if err := c2.refreshSnapshot(1); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	h2 := c2.plugin.QueryHealth()
	if h2.TVLCapBps != 3_300 {
		t.Errorf("capBps should still be reported: got %d", h2.TVLCapBps)
	}
	if h2.CanopyTotalStake != 0 || h2.TVLCapUcnpyEffective != 0 {
		t.Errorf("absent supply: stake=%d eff=%d want both 0", h2.CanopyTotalStake, h2.TVLCapUcnpyEffective)
	}
}
