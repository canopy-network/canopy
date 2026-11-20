package canoliq

import (
	"strings"
	"testing"
)

const realRecipientHex = "859a488ccbd1d9e22658ced19ee66cf0bd47b57a"

func genesisComplete(t *testing.T, c *Canoliq) bool {
	t.Helper()
	g, err := c.LoadGlobals()
	if err != nil {
		t.Fatalf("load globals: %v", err)
	}
	return g.GenesisComplete
}

// With ActivationHeight pinned, genesis runs in exactly that block — never
// before it — so a live validator and a node replaying from height 1 write
// canoLiq state in the same block.
func TestActivationHeightRunsGenesisOnlyAtPinnedBlock(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.GenesisPath = writeGenesisFixture(t, realRecipientHex)
	c.Config.ActivationHeight = 100

	for _, h := range []uint64{1, 50, 99} {
		if err := c.bootstrapGenesisIfNeeded(h); err != nil {
			t.Fatalf("height %d: %v", h, err)
		}
		if genesisComplete(t, c) {
			t.Fatalf("genesis ran at height %d, before activationHeight 100", h)
		}
	}
	if err := c.bootstrapGenesisIfNeeded(100); err != nil {
		t.Fatalf("height 100: %v", err)
	}
	if !genesisComplete(t, c) {
		t.Fatalf("genesis did not run at activationHeight 100")
	}
}

// A pinned block that already passed must not trigger genesis late (that
// would write state at a height other nodes didn't) and must not error
// (that would halt block production) — the plugin just stays inert.
func TestActivationHeightMissedStaysInertWithoutError(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.GenesisPath = writeGenesisFixture(t, realRecipientHex)
	c.Config.ActivationHeight = 100

	if err := c.bootstrapGenesisIfNeeded(150); err != nil {
		t.Fatalf("missed activation must not error, got %v", err)
	}
	if genesisComplete(t, c) {
		t.Fatalf("genesis ran at height 150, after activationHeight 100 had passed")
	}
}

// ActivationHeight 0 keeps the legacy behavior (localnet/devnet): genesis
// runs on the first BeginBlock that has a GenesisPath.
func TestActivationHeightZeroRunsImmediately(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.GenesisPath = writeGenesisFixture(t, realRecipientHex)

	if err := c.bootstrapGenesisIfNeeded(7); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if !genesisComplete(t, c) {
		t.Fatalf("activationHeight 0 should run genesis immediately")
	}
}

func mainnetConfigWithGenesis(t *testing.T) Config {
	c := DefaultConfig()
	c.Profile = ProfileMainnet
	c.ChainId = 29
	c.RedemptionUnstakingBlocks = 30240
	c.GenesisPath = writeGenesisFixture(t, realRecipientHex)
	return c
}

func TestSafetyCheckMainnetRefusesUnpinnedActivation(t *testing.T) {
	c := mainnetConfigWithGenesis(t)
	err := c.SafetyCheck()
	if err == nil || !strings.Contains(err.Error(), "activationHeight=0") {
		t.Fatalf("want activationHeight refusal, got %v", err)
	}
}

func TestSafetyCheckMainnetAcceptsPinnedActivation(t *testing.T) {
	c := mainnetConfigWithGenesis(t)
	c.ActivationHeight = 10_000
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("pinned mainnet config rejected: %v", err)
	}
}

// No genesisPath means the plugin stays inert by design (val-a's current
// setup), so an unpinned mainnet config without one must still start.
func TestSafetyCheckMainnetWithoutGenesisPathNeedsNoActivationHeight(t *testing.T) {
	c := DefaultConfig()
	c.Profile = ProfileMainnet
	c.ChainId = 29
	c.RedemptionUnstakingBlocks = 30240
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("inert mainnet config rejected: %v", err)
	}
}

// End-to-end on the file Canopy will deploy: the real mainnet genesis must
// run cleanly at the pinned height and seed val-a into the validator registry.
func TestMainnetGenesisRunsAtActivationHeight(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.GenesisPath = "genesis.mainnet.json"
	c.Config.ActivationHeight = 500

	if err := c.bootstrapGenesisIfNeeded(499); err != nil {
		t.Fatalf("height 499: %v", err)
	}
	if genesisComplete(t, c) {
		t.Fatalf("mainnet genesis ran before activation height")
	}
	if err := c.bootstrapGenesisIfNeeded(500); err != nil {
		t.Fatalf("mainnet genesis failed at activation height: %v", err)
	}
	if !genesisComplete(t, c) {
		t.Fatalf("mainnet genesis did not complete at activation height")
	}
	reg, err := c.loadValidatorRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if reg == nil || len(reg.Entries) != 1 || hexAddress(reg.Entries[0].Address) != "0x737ef80e8476450b6970dd6c1e44c93699c8e0c0" {
		t.Fatalf("registry not seeded with val-a: %+v", reg)
	}
}
