package contract

import "testing"

func TestComputeBorrowRate(t *testing.T) {
	cases := []struct {
		name           string
		utilizationBps uint64
		wantBps        uint64
	}{
		{"zero utilization = base rate", 0, 200},
		{"at kink exactly (80%) = base + slope1", 8000, 1000},
		{"half of optimal (40%) = base + half slope1", 4000, 600},
		{"full utilization (100%) = base + slope1 + slope2", 10000, 11000},
		{"just past kink (81%)", 8100, 1500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeBorrowRate(tc.utilizationBps)
			if got != tc.wantBps {
				t.Errorf("ComputeBorrowRate(%d) = %d, want %d", tc.utilizationBps, got, tc.wantBps)
			}
		})
	}
}

func TestComputeUtilizationBps(t *testing.T) {
	cases := []struct {
		name          string
		totalBorrowed uint64
		totalSupplied uint64
		wantBps       uint64
	}{
		{"zero supplied returns zero, not divide-by-zero panic", 500, 0, 0},
		{"zero borrowed", 0, 1000, 0},
		{"50% utilization", 500, 1000, 5000},
		{"100% utilization", 1000, 1000, 10000},
		{"floor rounding, not exact division", 1, 3, 3333},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeUtilizationBps(tc.totalBorrowed, tc.totalSupplied)
			if got != tc.wantBps {
				t.Errorf("ComputeUtilizationBps(%d, %d) = %d, want %d", tc.totalBorrowed, tc.totalSupplied, got, tc.wantBps)
			}
		})
	}
}

func TestAnnualRateToPerBlockRateRay(t *testing.T) {
	// 2% APR (200 bps) should be a small positive per-block rate, well
	// under RAY, and should not panic or return nil/zero for a
	// realistic input.
	rate := AnnualRateToPerBlockRateRay(200)
	if rate.Sign() <= 0 {
		t.Fatalf("expected positive per-block rate for 200bps APR, got %s", rate.String())
	}
	if rate.Cmp(RAY) >= 0 {
		t.Fatalf("per-block rate should be far smaller than RAY (1.0), got %s", rate.String())
	}

	// Zero APR should yield exactly zero.
	zero := AnnualRateToPerBlockRateRay(0)
	if zero.Sign() != 0 {
		t.Fatalf("expected zero per-block rate for 0bps APR, got %s", zero.String())
	}

	// Sanity: higher APR should yield a strictly higher per-block rate.
	low := AnnualRateToPerBlockRateRay(200)
	high := AnnualRateToPerBlockRateRay(30000) // 300%, governance ceiling per ARCM Section 15
	if high.Cmp(low) <= 0 {
		t.Fatalf("expected 300%% APR rate (%s) > 2%% APR rate (%s)", high.String(), low.String())
	}
}
