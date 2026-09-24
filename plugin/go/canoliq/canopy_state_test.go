package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// canopy_state_test.go covers readCanopySupply: a Supply seeded at
// contract.KeyForSupply() round-trips through readCanopySupply for both
// the absent (key not in state) and present paths. The deposit handler's
// fail-closed branch (Supply absent) and accept branch (Supply present
// with Staked=0) are pinned by the T3 suite (t3_tvlcap_test.go); these
// tests only verify the reader contract.

func TestReadCanopySupplyAbsent(t *testing.T) {
	c, s := newTestCanoliq()
	// Clear the fixture's default Supply to exercise the absent path.
	s.del(contract.KeyForSupply())

	supply, err := c.readCanopySupply()
	if err != nil {
		t.Fatalf("readCanopySupply: %v", err)
	}
	if supply != nil {
		t.Errorf("absent supply: got %+v want nil", supply)
	}
}

func TestReadCanopySupplyPresent(t *testing.T) {
	c, s := newTestCanoliq()

	// Seed a Supply directly at the Canopy key. The deposit handler
	// (Phase B) reads .Staked; future restaking automation (Phase C
	// 'active rebalancing', deferred) could read .CommitteeStaked.
	supply := &contract.Supply{
		Total:         100_000_000_000_000, // 100M CNPY × 1e6 uCNPY
		Staked:        30_000_000_000_000,  // 30%
		DelegatedOnly: 5_000_000_000_000,
		CommitteeStaked: []*contract.Pool{
			{Id: 2, Amount: 12_000_000_000_000}, // canoLiq committee
			{Id: 3, Amount: 18_000_000_000_000}, // another committee
		},
	}
	bz, err := contract.Marshal(supply)
	if err != nil {
		t.Fatalf("marshal supply: %v", err)
	}
	s.set(contract.KeyForSupply(), bz)

	got, perr := c.readCanopySupply()
	if perr != nil {
		t.Fatalf("readCanopySupply: %v", perr)
	}
	if got == nil {
		t.Fatal("readCanopySupply: got nil, want populated")
	}
	if got.Total != supply.Total {
		t.Errorf("Total: got %d want %d", got.Total, supply.Total)
	}
	if got.Staked != supply.Staked {
		t.Errorf("Staked: got %d want %d", got.Staked, supply.Staked)
	}
	if got.DelegatedOnly != supply.DelegatedOnly {
		t.Errorf("DelegatedOnly: got %d want %d", got.DelegatedOnly, supply.DelegatedOnly)
	}
	if len(got.CommitteeStaked) != len(supply.CommitteeStaked) {
		t.Fatalf("CommitteeStaked len: got %d want %d", len(got.CommitteeStaked), len(supply.CommitteeStaked))
	}
	for i, want := range supply.CommitteeStaked {
		if got.CommitteeStaked[i].Id != want.Id || got.CommitteeStaked[i].Amount != want.Amount {
			t.Errorf("CommitteeStaked[%d]: got {%d, %d} want {%d, %d}", i,
				got.CommitteeStaked[i].Id, got.CommitteeStaked[i].Amount, want.Id, want.Amount)
		}
	}
}
