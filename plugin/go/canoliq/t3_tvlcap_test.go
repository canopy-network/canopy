package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// t3_tvlcap_test.go covers the self-imposed TVL cap (T3 / WP §9.4): deposit
// rejection at the ceiling, lifting the cap via params, the uncapped default,
// and the /v1/health surfacing.

// hotPoolGlobals returns 1:1-rate globals with the given pooled CNPY so a
// deposit mints an equal amount of cCNPY.
func hotPoolGlobals(pooled uint64) *contract.CanoliqGlobals {
	return &contract.CanoliqGlobals{
		TotalCcnpySupply: pooled,
		TotalPooledCnpy:  pooled,
		GenesisComplete:  true,
	}
}

func cappedParams(cap uint64) *contract.CanoliqParams {
	p := DefaultParams()
	p.TvlCapUcnpy = cap
	return p
}

// TestT3DepositAtExactCapAccepted: a deposit landing exactly on the cap is
// allowed; one uCNPY more is rejected.
func TestT3DepositCapBoundary(t *testing.T) {
	const cap, pooled uint64 = 1_000_000, 600_000
	user := addr20(0x01)

	// Exactly to cap (600k + 400k == 1M) → accepted.
	c, s := newTestCanoliq()
	seedGlobals(s, hotPoolGlobals(pooled))
	seedAccount(s, user, 1_000_000)
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 400_000}, 10_000, cappedParams(cap)); r.Error != nil {
		t.Fatalf("deposit to exact cap should succeed: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != cap {
		t.Errorf("pooled after deposit: got %d want %d", g.TotalPooledCnpy, cap)
	}

	// One uCNPY above cap → rejected.
	c2, s2 := newTestCanoliq()
	seedGlobals(s2, hotPoolGlobals(pooled))
	seedAccount(s2, user, 1_000_000)
	r := c2.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 400_001}, 10_000, cappedParams(cap))
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

// TestT3LiftCapReenablesDeposits: a deposit rejected at the current cap is
// accepted once the cap param is raised.
func TestT3LiftCapReenablesDeposits(t *testing.T) {
	const pooled uint64 = 1_000_000
	user := addr20(0x02)
	c, s := newTestCanoliq()
	seedGlobals(s, hotPoolGlobals(pooled))
	seedAccount(s, user, 1_000_000)

	// At cap → rejected.
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 500_000}, 10_000, cappedParams(pooled)); r.Error == nil {
		t.Fatal("deposit at cap should be rejected")
	}
	// Lift the cap → accepted.
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 500_000}, 10_000, cappedParams(2_000_000)); r.Error != nil {
		t.Fatalf("deposit after lifting cap should succeed: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 1_500_000 {
		t.Errorf("pooled after lifted-cap deposit: got %d want 1_500_000", g.TotalPooledCnpy)
	}
}

// TestT3UncappedAllowsLargeDeposit: cap 0 imposes no ceiling.
func TestT3UncappedAllowsLargeDeposit(t *testing.T) {
	user := addr20(0x03)
	c, s := newTestCanoliq()
	seedGlobals(s, hotPoolGlobals(1_000_000))
	seedAccount(s, user, 1_000_000_000)
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 900_000_000}, 10_000, cappedParams(0)); r.Error != nil {
		t.Fatalf("uncapped deposit should succeed: %v", r.Error)
	}
}

// TestT3HealthSurfacesCapAndUtilization checks /v1/health exposes the cap and
// utilization bps (and reports 0 utilization when uncapped).
func TestT3HealthSurfacesCapAndUtilization(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, cappedParams(1_000_000))
	seedGlobals(s, &contract.CanoliqGlobals{TotalPooledCnpy: 250_000, GenesisComplete: true})
	if err := c.refreshSnapshot(5); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	h := c.plugin.QueryHealth()
	if h.TVLCapUcnpy != 1_000_000 {
		t.Errorf("tvlCapUcnpy: got %d want 1_000_000", h.TVLCapUcnpy)
	}
	if h.TVLUtilizationBps != 2_500 { // 250k / 1M = 25%
		t.Errorf("tvlUtilizationBps: got %d want 2500", h.TVLUtilizationBps)
	}

	// Uncapped → utilization 0.
	seedParams(t, c, cappedParams(0))
	if err := c.refreshSnapshot(6); err != nil {
		t.Fatalf("refreshSnapshot: %v", err)
	}
	h = c.plugin.QueryHealth()
	if h.TVLCapUcnpy != 0 || h.TVLUtilizationBps != 0 {
		t.Errorf("uncapped health: cap=%d util=%d want 0/0", h.TVLCapUcnpy, h.TVLUtilizationBps)
	}
}
