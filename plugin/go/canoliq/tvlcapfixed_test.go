package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// tvlcapfixed_test.go covers the switch from the percentage TVL cap to the
// fixed mainnet ceiling at MainnetFixedTvlCapHeight (tvlcapfixed.go).

// mainnetDepositAt runs one deposit on a mainnet-profile plugin at height h,
// with the chain-29 stake and pool given, under the 33% default cap.
func mainnetDepositAt(t *testing.T, h, chainStake, pooled, amount uint64) (*contract.PluginDeliverResponse, *fakeStore) {
	t.Helper()
	c, s := newMainnetFixCanoliq(t)
	user := addr20(0x01)
	seedCanopySupply(t, s, chainStake)
	seedGlobals(s, hotPoolGlobals(pooled))
	seedAccount(s, user, amount+1_000_000)
	c.plugin.setHeight(h)
	r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: amount}, 10_000, cappedParams(3_300))
	return r, s
}

// Before the height the percentage cap still governs: blocks that already
// ran must replay exactly as they did.
func TestFixedCapNotActiveBeforeHeight(t *testing.T) {
	// 33% of 2M = 660K cap; 650K pooled + 20K deposit is over it.
	r, s := mainnetDepositAt(t, MainnetFixedTvlCapHeight-1, 2_000_000_000_000, 650_000_000_000, 20_000_000_000)
	if r.Error == nil || r.Error.Code != codeTVLCapExceeded {
		t.Fatalf("pre-height deposit over the percentage cap: got %v, want codeTVLCapExceeded", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 650_000_000_000 {
		t.Fatalf("pooled changed on rejection: %d", g.TotalPooledCnpy)
	}
}

func TestFixedCapReplacesPercentageCapAtHeight(t *testing.T) {
	r, s := mainnetDepositAt(t, MainnetFixedTvlCapHeight, 2_000_000_000_000, 650_000_000_000, 20_000_000_000)
	if r.Error != nil {
		t.Fatalf("deposit under the fixed ceiling rejected: %v", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != 670_000_000_000 {
		t.Fatalf("pooled after deposit: %d", g.TotalPooledCnpy)
	}
}

func TestFixedCapBoundary(t *testing.T) {
	const pooled = MainnetFixedTvlCapUcnpy - 5_000_000
	// Exactly to the ceiling → accepted.
	if r, _ := mainnetDepositAt(t, MainnetFixedTvlCapHeight, 2_000_000_000_000, pooled, 5_000_000); r.Error != nil {
		t.Fatalf("deposit to exactly 50M rejected: %v", r.Error)
	}
	// One uCNLQ past it → rejected, state unchanged.
	r, s := mainnetDepositAt(t, MainnetFixedTvlCapHeight, 2_000_000_000_000, pooled, 5_000_001)
	if r.Error == nil || r.Error.Code != codeTVLCapExceeded {
		t.Fatalf("deposit past 50M: got %v, want codeTVLCapExceeded", r.Error)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != pooled {
		t.Fatalf("pooled changed on rejection: %d", g.TotalPooledCnpy)
	}
}

// The fixed ceiling does not depend on reading the chain's stake.
func TestFixedCapIgnoresChainStake(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	user := addr20(0x02)
	seedGlobals(s, hotPoolGlobals(1_000_000))
	seedAccount(s, user, 10_000_000)
	c.plugin.setHeight(MainnetFixedTvlCapHeight)
	// No Supply record: the percentage cap would fail closed.
	if r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000}, 10_000, cappedParams(3_300)); r.Error != nil {
		t.Fatalf("deposit without a Supply record: %v", r.Error)
	}
}

func TestFixedCapOnlyOnMainnet(t *testing.T) {
	c, s := newTestCanoliq()
	user := addr20(0x03)
	seedCanopySupply(t, s, 2_000_000_000_000)
	seedGlobals(s, hotPoolGlobals(650_000_000_000))
	seedAccount(s, user, 30_000_000_000)
	c.plugin.setHeight(MainnetFixedTvlCapHeight + 1_000)
	r := c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 20_000_000_000}, 10_000, cappedParams(3_300))
	if r.Error == nil || r.Error.Code != codeTVLCapExceeded {
		t.Fatalf("non-mainnet profile should keep the percentage cap: got %v", r.Error)
	}
}

func TestHealthReportsFixedCap(t *testing.T) {
	for _, tc := range []struct {
		h       uint64
		status  string
		cap     uint64
		utilBps uint64
	}{
		{MainnetFixedTvlCapHeight - 1, TVLCapStatusActive, 660_000_000_000, 9_848},
		{MainnetFixedTvlCapHeight, TVLCapStatusFixed, MainnetFixedTvlCapUcnpy, 130},
	} {
		c, s := newMainnetFixCanoliq(t)
		seedCanopySupply(t, s, 2_000_000_000_000)
		seedParams(t, c, cappedParams(3_300))
		seedGlobals(s, &contract.CanoliqGlobals{TotalPooledCnpy: 650_000_000_000, GenesisComplete: true})
		if err := c.refreshSnapshot(tc.h); err != nil {
			t.Fatalf("refreshSnapshot: %v", err)
		}
		h := c.plugin.QueryHealth()
		if h.TVLCapStatus != tc.status || h.TVLCapUcnpyEffective != tc.cap || h.TVLUtilizationBps != tc.utilBps {
			t.Fatalf("height %d: status %q cap %d util %d", tc.h, h.TVLCapStatus, h.TVLCapUcnpyEffective, h.TVLUtilizationBps)
		}
	}
}
