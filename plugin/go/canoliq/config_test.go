package canoliq

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// config_test.go covers the deployment-profile machinery added to support
// localnet/testnet/mainnet separation: NewConfigFromFile defaults,
// SafetyCheck behavior, and the configurable redemption unstaking window
// flowing through DeliverMessageCanoliqRedeem.

func TestNewConfigFromFileNormalizesProfileAndUnstakingBlocks(t *testing.T) {
	// Empty profile + zero RedemptionUnstakingBlocks should normalize to
	// localnet defaults so old config files keep working.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	mustWrite(t, path, `{"chainId":7,"dataDirPath":"/tmp/x"}`)
	c, err := NewConfigFromFile(path)
	if err != nil {
		t.Fatalf("NewConfigFromFile: %v", err)
	}
	if c.Profile != ProfileLocalnet {
		t.Fatalf("profile: got %q want %q", c.Profile, ProfileLocalnet)
	}
	if c.RedemptionUnstakingBlocks != 5 {
		t.Fatalf("RedemptionUnstakingBlocks: got %d want 5", c.RedemptionUnstakingBlocks)
	}
	if c.ChainId != 7 || c.DataDirPath != "/tmp/x" {
		t.Fatalf("explicit fields: %+v", c)
	}
}

func TestSafetyCheckLocalnetSkipsPlaceholderAddress(t *testing.T) {
	// The placeholder 851e90… is the localnet seed key; localnet profile
	// must accept it without complaint.
	gp := writeGenesisFixture(t, localnetPlaceholderAddress)
	c := DefaultConfig()
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("localnet safety check rejected placeholder: %v", err)
	}
}

func TestSafetyCheckTestnetRefusesPlaceholderAddress(t *testing.T) {
	// Same fixture, but profile=testnet — must refuse with a message that
	// names the offending bucket.
	gp := writeGenesisFixture(t, localnetPlaceholderAddress)
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.GenesisPath = gp
	err := c.SafetyCheck()
	if err == nil {
		t.Fatalf("testnet safety check accepted placeholder address")
	}
	if !strings.Contains(err.Error(), "localnet placeholder") {
		t.Fatalf("error should mention 'localnet placeholder': %v", err)
	}
	if !strings.Contains(err.Error(), "Liquidity") {
		// Fixture's only bucket is named "Liquidity" — confirm the
		// error pinpoints the offending bucket so operators know what
		// to edit.
		t.Fatalf("error should name the offending bucket: %v", err)
	}
}

func TestSafetyCheckTestnetAcceptsRealAddress(t *testing.T) {
	gp := writeGenesisFixture(t, "0102030405060708090a0b0c0d0e0f1011121314")
	c := DefaultConfig()
	c.Profile = ProfileMainnet
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("mainnet safety check rejected real address: %v", err)
	}
}

func TestSafetyCheckHandlesPrefixedAddress(t *testing.T) {
	// Recipient with 0x prefix should still be detected as the
	// placeholder; case folding too.
	gp := writeGenesisFixture(t, "0x"+strings.ToUpper(localnetPlaceholderAddress))
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err == nil {
		t.Fatalf("safety check missed 0x-prefixed uppercase placeholder")
	}
}

func TestRedemptionWindowFromConfig(t *testing.T) {
	c, s := newTestCanoliq()
	c.Config.RedemptionUnstakingBlocks = 1234
	user := addr20(0xAB)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000, GenesisComplete: true}
	s.set(KeyForGlobals(), mustMarshal(g))
	s.set(KeyForCCNPYBalance(user), EncodeUint64(500))
	seedAccount(s, user, 100_000)
	c.plugin.setHeight(42)

	resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 250},
		10_000, DefaultParams())
	if resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}
	// Redemption id is globals.NextRedemptionId before the increment.
	red := new(contract.Redemption)
	bz := s.get(KeyForRedemption(user, 0))
	if err := contract.Unmarshal(bz, red); err != nil {
		t.Fatalf("unmarshal redemption: %v", err)
	}
	if red.UnbondCompleteHeight != 42+1234 {
		t.Fatalf("UnbondCompleteHeight: got %d want %d", red.UnbondCompleteHeight, 42+1234)
	}
}

func TestRedemptionWindowFallsBackTo5WhenZero(t *testing.T) {
	c, s := newTestCanoliq()
	c.Config.RedemptionUnstakingBlocks = 0 // simulate stale config without the field
	user := addr20(0xAC)
	g := &contract.CanoliqGlobals{TotalCcnpySupply: 1000, TotalPooledCnpy: 1000, GenesisComplete: true}
	s.set(KeyForGlobals(), mustMarshal(g))
	s.set(KeyForCCNPYBalance(user), EncodeUint64(500))
	seedAccount(s, user, 100_000)
	c.plugin.setHeight(100)

	resp := c.DeliverMessageCanoliqRedeem(
		&contract.MessageCanoliqRedeem{FromAddress: user, CcnpyAmount: 250},
		10_000, DefaultParams())
	if resp.Error != nil {
		t.Fatalf("redeem: %v", resp.Error)
	}
	red := new(contract.Redemption)
	if err := contract.Unmarshal(s.get(KeyForRedemption(user, 0)), red); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if red.UnbondCompleteHeight != 105 {
		t.Fatalf("expected fallback window of 5 blocks → height 105, got %d",
			red.UnbondCompleteHeight)
	}
}

func TestBundledTestnetGenesisIsSafetyCheckClean(t *testing.T) {
	// The committed plugin/go/canoliq/genesis.testnet.json must pass
	// safety check under profile=testnet — even though its addresses are
	// TODO placeholders, none of them is the localnet placeholder.
	// Tests run from the package dir, so the file is in cwd.
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.GenesisPath = "genesis.testnet.json"
	if _, err := os.Stat(c.GenesisPath); err != nil {
		t.Skipf("genesis.testnet.json not present: %v", err)
	}
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("bundled testnet template should pass safety check: %v", err)
	}
}

// --- helpers ---

// writeGenesisFixture writes a minimal one-bucket GenesisFile with the
// given recipient address. Buckets total 10000 bps (the validator
// requires it).
func writeGenesisFixture(t *testing.T, recipientHex string) string {
	t.Helper()
	gf := GenesisFile{
		BlocksPerYear: 5_256_000,
		Buckets: []GenesisBucket{
			{
				Name: "Liquidity",
				Bps:  10_000,
				Recipients: []GenesisAllocation{
					{Address: recipientHex, Bps: 10_000},
				},
			},
		},
	}
	bz, err := json.Marshal(gf)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(path, bz, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

