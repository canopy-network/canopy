package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// TestL4TreasuryCplqSpendBumpsCirculating covers L4: paying CPLQ out of the DAO
// treasury returns it to circulation, so CplqCirculatingSupply must rise by the
// spend amount while CplqTotalSupply is unchanged, and the supply invariant
// (circulating <= total <= 100M) must hold afterward.
func TestL4TreasuryCplqSpendBumpsCirculating(t *testing.T) {
	c, s := newTestCanoliq()

	const treasuryCplq = 10_000_000
	const spendAmt = 4_000_000
	const startCirculating = 50_000_000_000_000 // 50M CPLQ already circulating
	recipient := addr20(0x77)

	s.set(KeyForTreasuryCPLQ(), EncodeUint64(treasuryCplq))
	seedGlobals(s, &contract.CanoliqGlobals{
		CplqTotalSupply:       CPLQTotalSupply,
		CplqCirculatingSupply: startCirculating,
	})

	spend := &contract.TreasurySpend{
		Id:       1,
		Executed: false,
		Payload: &contract.ProposalTreasurySpend{
			Recipient:    recipient,
			Amount:       spendAmt,
			Denomination: contract.SpendDenomination_SPEND_CPLQ,
		},
	}
	if err := c.applySpend(spend); err != nil {
		t.Fatalf("applySpend: %v", err)
	}

	// Recipient credited; treasury debited.
	if got := DecodeUint64(s.get(KeyForCPLQBalance(recipient))); got != spendAmt {
		t.Fatalf("recipient CPLQ: got %d want %d", got, spendAmt)
	}
	if got := DecodeUint64(s.get(KeyForTreasuryCPLQ())); got != treasuryCplq-spendAmt {
		t.Fatalf("treasury CPLQ: got %d want %d", got, treasuryCplq-spendAmt)
	}

	g := loadGlobals(t, s)
	if g.CplqCirculatingSupply != startCirculating+spendAmt {
		t.Fatalf("circulating: got %d want %d", g.CplqCirculatingSupply, startCirculating+spendAmt)
	}
	if g.CplqTotalSupply != CPLQTotalSupply {
		t.Fatalf("total supply changed: got %d want %d", g.CplqTotalSupply, CPLQTotalSupply)
	}
	// Supply invariant: circulating <= total <= 100M.
	if g.CplqCirculatingSupply > g.CplqTotalSupply {
		t.Fatalf("invariant broken: circulating %d > total %d", g.CplqCirculatingSupply, g.CplqTotalSupply)
	}
	if g.CplqTotalSupply > CPLQTotalSupply {
		t.Fatalf("invariant broken: total %d > 100M %d", g.CplqTotalSupply, CPLQTotalSupply)
	}
}
