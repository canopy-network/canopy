package canoliq

import (
	"testing"
)

// seedOrphanedPool puts a Canoliq into the exact state
// reconcileOrphanedPoolOnDevnet exists for: genesis already ran, committee
// reward has compounded into the pool, but TotalCcnpySupply is still zero
// because no real deposit has ever minted cCNPY against it.
func seedOrphanedPool(t *testing.T, c *Canoliq, s *fakeStore, pooled uint64) {
	t.Helper()
	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	g.GenesisComplete = true
	g.TotalCcnpySupply = 0
	g.TotalPooledCnpy = pooled
	g.PeakTvlUcnpy = pooled
	if err := c.SaveGlobals(g); err != nil {
		t.Fatalf("save globals: %v", err)
	}
	seedEscrow(s, pooled)
}

// This is the exact deadlock reconcileOrphanedPoolOnDevnet exists to break:
// weeks of committee reward accrual with zero real deposits leaves
// TotalPooledCnpy far larger than any realistic single deposit, so
// computeMint(amount, 0, hugePooled) floors to zero for every amount up to
// ~hugePooled itself and every deposit is rejected forever.
func TestReconcileOrphanedPoolOnDevnetZeroesPool(t *testing.T) {
	const orphaned = 14_908_429_287_755 // ~14.9M CNPY, observed on canoliq-98803
	c, s := newTestCanoliq()
	c.Config.Profile = ProfileDevnet
	seedOrphanedPool(t, c, s, orphaned)

	if err := c.bootstrapGenesisIfNeeded(); err != nil {
		t.Fatalf("bootstrapGenesisIfNeeded: %v", err)
	}

	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	if g.TotalPooledCnpy != 0 {
		t.Fatalf("TotalPooledCnpy: got %d want 0", g.TotalPooledCnpy)
	}
	if g.PeakTvlUcnpy != 0 {
		t.Fatalf("PeakTvlUcnpy: got %d want 0 (stale high-water mark must not survive the reconcile)", g.PeakTvlUcnpy)
	}
	if got := readEscrow(s); got != 0 {
		t.Fatalf("escrow: got %d want 0 (must track TotalPooledCnpy + PendingRedemptionCnpy)", got)
	}

	// The deadlock is actually broken: a realistic deposit now mints ~1:1
	// again, matching computeMint's documented first-deposit behavior.
	const deposit = 20_000_000 // 20 CNPY
	if mint := computeMint(deposit, g.TotalCcnpySupply, g.TotalPooledCnpy); mint == 0 {
		t.Fatalf("computeMint(%d, 0, 0) = 0 — deposit still rejected after reconcile", deposit)
	}
}

// Once a real deposit has minted any cCNPY, TotalCcnpySupply != 0 and the
// reconcile must never fire again — it must not be possible for this to
// touch a balance a real depositor holds a claim against.
func TestReconcileOrphanedPoolOnDevnetInertAfterRealDeposit(t *testing.T) {
	c, s := newTestCanoliq()
	c.Config.Profile = ProfileDevnet
	seedOrphanedPool(t, c, s, 14_908_429_287_755)

	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	// Simulate a real deposit having landed: some cCNPY now exists.
	g.TotalCcnpySupply = 5_000_000
	if err := c.SaveGlobals(g); err != nil {
		t.Fatalf("save globals: %v", err)
	}

	if err := c.bootstrapGenesisIfNeeded(); err != nil {
		t.Fatalf("bootstrapGenesisIfNeeded: %v", err)
	}

	got, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals (after): %v", err)
	}
	if got.TotalPooledCnpy != 14_908_429_287_755 {
		t.Fatalf("TotalPooledCnpy: got %d want unchanged 14908429287755 — a real depositor's backing must never be touched",
			got.TotalPooledCnpy)
	}
	if got.TotalCcnpySupply != 5_000_000 {
		t.Fatalf("TotalCcnpySupply: got %d want unchanged 5000000", got.TotalCcnpySupply)
	}
}

// Symmetric with the tvlCapBps override: testnet/mainnet never get an
// automatic pool reconcile. An orphaned pool there needs a human decision,
// not a silent state rewrite.
func TestReconcileOrphanedPoolOnDevnetSkippedOffDevProfiles(t *testing.T) {
	for _, profile := range []string{ProfileTestnet, ProfileMainnet} {
		c, s := newTestCanoliq()
		c.Config.Profile = profile
		seedOrphanedPool(t, c, s, 14_908_429_287_755)

		if err := c.bootstrapGenesisIfNeeded(); err != nil {
			t.Fatalf("profile=%q: bootstrapGenesisIfNeeded: %v", profile, err)
		}

		g, err := c.LoadGlobals()
		if err != nil {
			t.Fatalf("profile=%q: load globals: %v", profile, err)
		}
		if g.TotalPooledCnpy != 14_908_429_287_755 {
			t.Fatalf("profile=%q: TotalPooledCnpy: got %d want unchanged 14908429287755", profile, g.TotalPooledCnpy)
		}
	}
}

// A healthy pool (TotalPooledCnpy == 0, nothing accrued yet) must not be
// touched — there is nothing to reconcile.
func TestReconcileOrphanedPoolOnDevnetNoOpOnEmptyPool(t *testing.T) {
	c, s := newTestCanoliq()
	c.Config.Profile = ProfileDevnet
	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	g.GenesisComplete = true
	if err := c.SaveGlobals(g); err != nil {
		t.Fatalf("save globals: %v", err)
	}

	if err := c.bootstrapGenesisIfNeeded(); err != nil {
		t.Fatalf("bootstrapGenesisIfNeeded: %v", err)
	}
	if got := readEscrow(s); got != 0 {
		t.Fatalf("escrow: got %d want 0", got)
	}
}

// PendingRedemptionCnpy, if somehow non-zero alongside an orphaned pool,
// must survive the reconcile in the escrow balance — the invariant is
// escrow == TotalPooledCnpy + PendingRedemptionCnpy, not escrow == 0.
func TestReconcileOrphanedPoolOnDevnetPreservesPendingRedemption(t *testing.T) {
	c, s := newTestCanoliq()
	c.Config.Profile = ProfileDevnet
	seedOrphanedPool(t, c, s, 14_908_429_287_755)

	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	g.PendingRedemptionCnpy = 42
	seedEscrow(s, g.TotalPooledCnpy+g.PendingRedemptionCnpy)
	if err := c.SaveGlobals(g); err != nil {
		t.Fatalf("save globals: %v", err)
	}

	if err := c.bootstrapGenesisIfNeeded(); err != nil {
		t.Fatalf("bootstrapGenesisIfNeeded: %v", err)
	}

	got, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals (after): %v", err)
	}
	if got.TotalPooledCnpy != 0 {
		t.Fatalf("TotalPooledCnpy: got %d want 0", got.TotalPooledCnpy)
	}
	if escrow := readEscrow(s); escrow != got.PendingRedemptionCnpy {
		t.Fatalf("escrow: got %d want %d (== TotalPooledCnpy(0) + PendingRedemptionCnpy)", escrow, got.PendingRedemptionCnpy)
	}
}
