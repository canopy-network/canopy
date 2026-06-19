package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// canopy_state_test.go covers the Phase-A readers: a Supply seeded at
// contract.KeyForSupply() decodes correctly through readCanopySupply and
// readCanopyTotalStake, including the absence path (Supply not set yet).

func TestReadCanopyTotalStakeAbsent(t *testing.T) {
	c, _ := newTestCanoliq()

	stake, err := c.readCanopyTotalStake()
	if err != nil {
		t.Fatalf("readCanopyTotalStake: %v", err)
	}
	if stake != 0 {
		t.Errorf("absent supply: got %d want 0", stake)
	}

	supply, err := c.readCanopySupply()
	if err != nil {
		t.Fatalf("readCanopySupply: %v", err)
	}
	if supply != nil {
		t.Errorf("absent supply: got %+v want nil", supply)
	}
}

func TestReadCanopyTotalStakePresent(t *testing.T) {
	c, s := newTestCanoliq()

	// Seed a Supply directly at the Canopy key. The percentage TVL cap
	// (Phase B) reads .Staked; the restaking optimizer (Phase C) reads
	// .CommitteeStaked entries.
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

	// Total stake should match what we seeded.
	stake, perr := c.readCanopyTotalStake()
	if perr != nil {
		t.Fatalf("readCanopyTotalStake: %v", perr)
	}
	if stake != supply.Staked {
		t.Errorf("readCanopyTotalStake: got %d want %d", stake, supply.Staked)
	}

	// Full Supply should round-trip including the repeated CommitteeStaked
	// — Phase C's restaking allocator reads this aggregate.
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
