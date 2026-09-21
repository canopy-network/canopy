package canoliq

import (
	"bytes"
	"encoding/json"
	"log"
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
	c.RedemptionUnstakingBlocks = 30240 // valid window so we reach the placeholder check (M2)
	c.GenesisPath = gp
	err := c.SafetyCheck()
	if err == nil {
		t.Fatalf("testnet safety check accepted placeholder address")
	}
	// The message is broader than it used to be: the check now rejects any
	// unfilled template slot, not only the localnet seed key.
	if !strings.Contains(err.Error(), "placeholder address") {
		t.Fatalf("error should mention the placeholder address: %v", err)
	}
	if !strings.Contains(err.Error(), localnetPlaceholderAddress) {
		t.Fatalf("error should name the offending address: %v", err)
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
	c.RedemptionUnstakingBlocks = 30240 // valid window (M2) so only the address path is exercised
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("mainnet safety check rejected real address: %v", err)
	}
}

func TestSafetyCheckRejectsSmallRedemptionWindowOnNonLocalnet(t *testing.T) {
	// M2: testnet/mainnet must fail closed when redemptionUnstakingBlocks is
	// below the floor (the localnet default of 5 is the common misconfig).
	gp := writeGenesisFixture(t, "0102030405060708090a0b0c0d0e0f1011121314")
	for _, profile := range []string{ProfileTestnet, ProfileMainnet} {
		c := DefaultConfig() // RedemptionUnstakingBlocks defaults to 5
		c.Profile = profile
		c.GenesisPath = gp
		err := c.SafetyCheck()
		if err == nil {
			t.Fatalf("profile=%q accepted redemptionUnstakingBlocks=%d", profile, c.RedemptionUnstakingBlocks)
		}
		if !strings.Contains(err.Error(), "redemptionUnstakingBlocks") {
			t.Fatalf("profile=%q error should name redemptionUnstakingBlocks: %v", profile, err)
		}
	}

	// At the floor it must pass (address is real).
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.RedemptionUnstakingBlocks = minNonLocalnetRedemptionBlocks
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("floor window rejected: %v", err)
	}
}

func TestSafetyCheckLocalnetAllowsSmallRedemptionWindow(t *testing.T) {
	// Localnet keeps the fast 5-block window for tests — the M2 floor must not
	// apply there.
	gp := writeGenesisFixture(t, localnetPlaceholderAddress)
	c := DefaultConfig() // localnet, window 5
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("localnet safety check rejected small window: %v", err)
	}
}

func TestSafetyCheckHandlesPrefixedAddress(t *testing.T) {
	// Recipient with 0x prefix should still be detected as the
	// placeholder; case folding too.
	gp := writeGenesisFixture(t, "0x"+strings.ToUpper(localnetPlaceholderAddress))
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.RedemptionUnstakingBlocks = 30240 // valid window so we reach the placeholder check (M2)
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

// TestBundledTestnetGenesisRefusesPlaceholders replaces an earlier test that
// asserted the opposite: that the bundled testnet template passed SafetyCheck
// *because* its placeholder addresses were merely distinct from the localnet
// seed key. That property was the hole. It meant an operator could copy the
// template, fill in six of seven slots, and boot a chain that minted a real
// CPLQ tranche to an address nobody controls — unrecoverable, since genesis
// runs once. A shipped template must not be bootable as shipped.
func TestBundledTestnetGenesisRefusesPlaceholders(t *testing.T) {
	c := DefaultConfig()
	c.Profile = ProfileTestnet
	c.RedemptionUnstakingBlocks = 30240 // matches canoliq-config.testnet.json (M2 floor)
	c.GenesisPath = "genesis.testnet.json"
	if _, err := os.Stat(c.GenesisPath); err != nil {
		t.Skipf("genesis.testnet.json not present: %v", err)
	}
	err := c.SafetyCheck()
	if err == nil {
		t.Fatal("bundled testnet template booted with placeholders still unfilled")
	}
	if !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("refusal should name the placeholder: %v", err)
	}
	// A filled-in copy of the same template must still pass, so the guard
	// blocks unfilled slots rather than the template shape itself.
	data, rerr := os.ReadFile("genesis.testnet.json")
	if rerr != nil {
		t.Fatalf("read template: %v", rerr)
	}
	filled := strings.NewReplacer(
		"0000000000000000000000000000000000000001", "0102030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000002", "0202030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000003", "0302030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000004", "0402030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000005", "0502030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000006", "0602030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000010", "1002030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000011", "1102030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000012", "1202030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000013", "1302030405060708090a0b0c0d0e0f1011121314",
		"0000000000000000000000000000000000000014", "1402030405060708090a0b0c0d0e0f1011121314",
	).Replace(string(data))
	gp := filepath.Join(t.TempDir(), "genesis.json")
	if werr := os.WriteFile(gp, []byte(filled), 0o600); werr != nil {
		t.Fatalf("write filled template: %v", werr)
	}
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("filled-in template should pass: %v", err)
	}
}

// TestValidateParamsFeeBpsBounds pins the F4 fee-rate bound (Tokenomics
// v1.2 §3.3 / WP §4.1: 5%–20%). ValidateParams must reject any FeeBps
// outside [500, 2000] and accept the edges. This is the enforcement a
// passed param-change proposal hits in dispatchPassed before SaveParams.
func TestValidateParamsFeeBpsBounds(t *testing.T) {
	cases := []struct {
		feeBps uint64
		ok     bool
	}{
		{499, false},  // just below 5%
		{500, true},   // 5% floor
		{1200, true},  // default 12%
		{2000, true},  // 20% ceiling
		{2001, false}, // just above 20%
		{2500, false}, // 25% — the localnet reconciliation case
	}
	for _, tc := range cases {
		p := DefaultParams()
		p.FeeBps = tc.feeBps
		err := ValidateParams(p)
		if tc.ok && err != nil {
			t.Errorf("FeeBps=%d: want accepted, got %v", tc.feeBps, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("FeeBps=%d: want rejected, got nil", tc.feeBps)
		}
	}
}

// TestValidateParamsMultisigThreshold covers L1: with signers configured the
// threshold must be in [1, len(signers)] — a zero threshold (which would
// defeat the multisig gate) and an over-count are both rejected.
func TestValidateParamsMultisigThreshold(t *testing.T) {
	signers := [][]byte{{1}, {2}, {3}}
	cases := []struct {
		name      string
		signers   [][]byte
		threshold uint64
		ok        bool
	}{
		{"zero threshold with signers rejected", signers, 0, false},
		{"threshold above signer count rejected", signers, 4, false},
		{"valid 2-of-3 accepted", signers, 2, true},
		{"full 3-of-3 accepted", signers, 3, true},
		{"zero threshold with no signers accepted", nil, 0, true},
	}
	for _, tc := range cases {
		p := DefaultParams()
		p.MultisigSigners = tc.signers
		p.MultisigThreshold = tc.threshold
		err := ValidateParams(p)
		if tc.ok && err != nil {
			t.Errorf("%s: want accepted, got %v", tc.name, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("%s: want rejected, got nil", tc.name)
		}
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

// TestSafetyCheckRejectsUnrecognizedProfile pins the fail-closed behavior. An
// unset profile used to skip every guard below it, which is silent and exactly
// backwards: the less a deployment has declared about itself, the more checks
// it should get, not fewer. A typo'd profile lands here too.
func TestSafetyCheckRejectsUnrecognizedProfile(t *testing.T) {
	gp := writeGenesisFixture(t, "0102030405060708090a0b0c0d0e0f1011121314")
	for _, profile := range []string{"", "mainet", "prod", "LOCALNET"} {
		t.Run("profile="+profile, func(t *testing.T) {
			c := DefaultConfig()
			c.Profile = profile
			c.RedemptionUnstakingBlocks = 30240
			c.GenesisPath = gp
			err := c.SafetyCheck()
			if err == nil {
				t.Fatalf("profile %q was accepted", profile)
			}
			if !strings.Contains(err.Error(), "unrecognized profile") {
				t.Fatalf("error should name the unrecognized profile: %v", err)
			}
		})
	}
	// The four known profiles still pass.
	for _, profile := range []string{ProfileLocalnet, ProfileDevnet, ProfileTestnet, ProfileMainnet} {
		c := DefaultConfig()
		c.Profile = profile
		c.RedemptionUnstakingBlocks = 30240
		c.GenesisPath = gp
		if err := c.SafetyCheck(); err != nil {
			t.Errorf("profile %q rejected: %v", profile, err)
		}
	}
}

// TestSafetyCheckRejectsTemplatePlaceholders covers the hole that let a
// half-filled template boot. The shipped testnet template used seven distinct
// fake addresses precisely so the old single-address check would pass, so
// copying it forward and missing one slot minted a real tranche to an address
// nobody controls. Genesis is one-shot, so that is unrecoverable.
func TestSafetyCheckRejectsTemplatePlaceholders(t *testing.T) {
	for _, addr := range []string{
		"0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"000000000000000000000000000000000000FFFF",
		localnetPlaceholderAddress,
	} {
		if !isTemplatePlaceholder(addr) {
			t.Errorf("isTemplatePlaceholder(%q) = false, want true", addr)
		}
	}
	for _, addr := range []string{
		"0102030405060708090a0b0c0d0e0f1011121314",
		"851e90eaef1fa27debaee2c2591503bdeec1d124", // one digit off the localnet key
	} {
		if isTemplatePlaceholder(addr) {
			t.Errorf("isTemplatePlaceholder(%q) = true, want false", addr)
		}
	}
	// End to end: a template placeholder in a bucket refuses to boot.
	gp := writeGenesisFixture(t, "0000000000000000000000000000000000000003")
	c := DefaultConfig()
	c.Profile = ProfileMainnet
	c.RedemptionUnstakingBlocks = 30240
	c.GenesisPath = gp
	err := c.SafetyCheck()
	if err == nil {
		t.Fatal("mainnet accepted a template placeholder address")
	}
	if !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("error should name the placeholder: %v", err)
	}
}

// TestShippedTemplatesRefuseToBoot is the regression test for the whole class:
// both shipped non-localnet templates must be unbootable as committed. If
// someone fills one in and commits it, this fails loudly, which is the point.
func TestShippedTemplatesRefuseToBoot(t *testing.T) {
	for _, path := range []string{"genesis.testnet.json", "genesis.mainnet.json"} {
		t.Run(path, func(t *testing.T) {
			c := DefaultConfig()
			c.Profile = ProfileMainnet
			c.ChainId = mainnetCommitteeId
			c.RedemptionUnstakingBlocks = 30240
			c.GenesisPath = path
			err := c.SafetyCheck()
			if err == nil {
				t.Fatalf("%s booted with placeholders still in place", path)
			}
			// Assert *why* it refused. SafetyCheck has several failure modes and
			// more have been added since; without this the test would keep
			// passing on an unreadable path or an unset chain id and silently
			// stop covering the placeholders it exists to catch.
			if !strings.Contains(err.Error(), "placeholder") {
				t.Fatalf("%s refused for the wrong reason (want a placeholder rejection): %v", path, err)
			}
		})
	}
}

// TestSafetyCheckRejectsUnsetChainId covers the committee id being left at its
// template placeholder of 0. Nothing else validates ChainId, and the failure it
// causes is silent rather than loud: no validator declares committee 0, so the
// registry reconciles to empty, the validator-incentive slice routes to the
// synthetic aggregator key, and every fee-pool read is scoped to the wrong
// pool. Localnet is exempt, as it is for every other guard.
func TestSafetyCheckRejectsUnsetChainId(t *testing.T) {
	gp := writeGenesisFixture(t, "0102030405060708090a0b0c0d0e0f1011121314")
	for _, profile := range []string{ProfileDevnet, ProfileTestnet, ProfileMainnet} {
		t.Run("profile="+profile, func(t *testing.T) {
			c := DefaultConfig()
			c.Profile = profile
			c.ChainId = 0
			c.RedemptionUnstakingBlocks = 30240
			c.GenesisPath = gp
			err := c.SafetyCheck()
			if err == nil {
				t.Fatal("chainId=0 was accepted outside localnet")
			}
			if !strings.Contains(err.Error(), "chainId=0") {
				t.Fatalf("error should name the offending field, got: %v", err)
			}
		})
	}
	// Localnet is deliberately exempt so a bare DefaultConfig still boots.
	c := DefaultConfig()
	c.Profile = ProfileLocalnet
	c.ChainId = 0
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("localnet must stay exempt: %v", err)
	}
	// A real committee id passes.
	c = DefaultConfig()
	c.Profile = ProfileMainnet
	c.ChainId = 19
	c.RedemptionUnstakingBlocks = 30240
	c.GenesisPath = gp
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("chainId=19 should pass: %v", err)
	}
}

// mainnetCommitteeId is canoLiq's committee id on Canopy mainnet, confirmed
// against the registration. It is the one value in the mainnet config that
// cannot be derived from anything else in the repo, and it is load-bearing:
// every fee-pool key is scoped by it and committee membership is matched on
// it, so a wrong value fails silently rather than loudly (see
// TestSafetyCheckRejectsUnsetChainId).
const mainnetCommitteeId = 19

// TestMainnetConfigCarriesCommitteeId pins the shipped mainnet config to the
// registered committee id. The genesis it points at is still a placeholder
// template that refuses to boot (TestShippedTemplatesRefuseToBoot), but genesis
// is one-shot, so a regression here must fail in CI rather than on a live
// mainnet.
func TestMainnetConfigCarriesCommitteeId(t *testing.T) {
	c, err := NewConfigFromFile("canoliq-config.mainnet.json")
	if err != nil {
		t.Fatalf("load mainnet config: %v", err)
	}
	if c.ChainId != mainnetCommitteeId {
		t.Fatalf("mainnet committee id = %d, want %d", c.ChainId, mainnetCommitteeId)
	}
}

// TestSafetyCheckRejectsBrokenGenesisPath covers a genesisPath that is set but
// does not resolve, or resolves to something unparseable. Both used to return
// nil on the theory that runGenesis would report it later with a better
// message. It does — but only per-block from BeginBlock, once the node is
// already running, and genesis is one-shot.
func TestSafetyCheckRejectsBrokenGenesisPath(t *testing.T) {
	malformed := filepath.Join(t.TempDir(), "genesis.json")
	mustWrite(t, malformed, `{"buckets": [ this is not json`)

	cases := []struct {
		name, path, wantSubstr string
	}{
		{"missing file", filepath.Join(t.TempDir(), "does-not-exist.json"), "unreadable genesisPath"},
		{"malformed json", malformed, "malformed genesis"},
	}
	for _, tc := range cases {
		for _, profile := range []string{ProfileDevnet, ProfileTestnet, ProfileMainnet} {
			t.Run(tc.name+"/"+profile, func(t *testing.T) {
				c := DefaultConfig()
				c.Profile = profile
				c.ChainId = 19
				c.RedemptionUnstakingBlocks = 30240
				c.GenesisPath = tc.path
				err := c.SafetyCheck()
				if err == nil {
					t.Fatalf("%s was accepted under profile=%q", tc.name, profile)
				}
				if !strings.Contains(err.Error(), tc.wantSubstr) {
					t.Fatalf("error should say %q, got: %v", tc.wantSubstr, err)
				}
				if !strings.Contains(err.Error(), tc.path) {
					t.Fatalf("error should name the offending path, got: %v", err)
				}
			})
		}
	}

	// Localnet stays exempt, as it is for every other guard.
	c := DefaultConfig()
	c.Profile = ProfileLocalnet
	c.GenesisPath = filepath.Join(t.TempDir(), "does-not-exist.json")
	if err := c.SafetyCheck(); err != nil {
		t.Fatalf("localnet must stay exempt: %v", err)
	}
}

// TestSafetyCheckAllowsEmptyGenesisPath pins the deliberate hole in the guard
// above. An empty path is legitimate when the canoLiq section was merged into
// the node's own genesis.json, because the FSM then dispatches it as a
// PluginGenesisRequest. That is indistinguishable at startup from a deployment
// that forgot the setting, so it must not fail here — bootstrapGenesisIfNeeded
// warns at runtime instead.
func TestSafetyCheckAllowsEmptyGenesisPath(t *testing.T) {
	for _, profile := range []string{ProfileDevnet, ProfileTestnet, ProfileMainnet} {
		c := DefaultConfig()
		c.Profile = profile
		c.ChainId = 19
		c.RedemptionUnstakingBlocks = 30240
		c.GenesisPath = ""
		if err := c.SafetyCheck(); err != nil {
			t.Fatalf("profile=%q: an empty genesisPath must not fail at startup: %v", profile, err)
		}
	}
}

// TestBootstrapWarnsOnceWhenGenesisPathMissing covers the runtime half of the
// fix. A deployment whose genesisPath is unset and whose node genesis carries
// no canoLiq section sits at genesis_complete=false forever with every pool at
// zero and ProcessRewards a no-op. That produced no signal at all, so the only
// symptom was a chain that looked healthy while canoLiq quietly did nothing.
//
// BeginBlock must keep succeeding (the merged-genesis case is legitimate, and
// tests drive BeginBlock without a genesis file), but it must say so — once,
// since a per-block warning would bury the line it is meant to surface.
func TestBootstrapWarnsOnceWhenGenesisPathMissing(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.GenesisPath = ""

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	for h := uint64(1); h <= 5; h++ {
		if resp := c.BeginBlock(&contract.PluginBeginRequest{Height: h}); resp.Error != nil {
			t.Fatalf("BeginBlock at height %d: %v", h, resp.Error)
		}
	}

	out := buf.String()
	if n := strings.Count(out, "no genesisPath configured"); n != 1 {
		t.Fatalf("warning fired %d times across 5 blocks, want exactly 1:\n%s", n, out)
	}
	// The message has to carry the fix, not just the symptom: the config is
	// read once at startup, so editing it does not self-correct per block.
	for _, want := range []string{"CANOLIQ_CONFIG", "restart"} {
		if !strings.Contains(out, want) {
			t.Errorf("warning should mention %q so the reader knows what to do:\n%s", want, out)
		}
	}
}
