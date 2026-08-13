package canoliq

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// The genesis tvlCapBps knob exists so long-lived development committees can
// run uncapped. It is a *uint64 rather than a uint64 because 0 — "uncapped" —
// is exactly the value operators need to set, and the `!= 0` convention the
// other GenesisParamsJSON fields use cannot express it. These tests pin the
// three-way distinction that pointer buys (absent / explicit zero / explicit
// non-zero) and the profile whitelist that keeps it out of production.

func genesisWithParams(p *GenesisParamsJSON) *GenesisFile {
	gf := miniGenesis()
	gf.Params = p
	return gf
}

func paramsAfterGenesis(t *testing.T, c *Canoliq, gf *GenesisFile) *contract.CanoliqParams {
	t.Helper()
	resp := c.Genesis(&contract.PluginGenesisRequest{GenesisJson: mustJSON(t, gf)})
	if resp.Error != nil {
		t.Fatalf("genesis: %v", resp.Error)
	}
	got, err := c.LoadParams()
	if err != nil {
		t.Fatalf("load params: %v", err)
	}
	return got
}

// Absent tvlCapBps must fall back to the whitepaper default, not to the zero
// value of the JSON field. This is the regression the pointer prevents: a plain
// uint64 would read as 0 for every genesis file that omits the key and silently
// uncap every chain.
func TestGenesisTvlCapAbsentKeepsDefault(t *testing.T) {
	c, _ := newTestCanoliq()
	got := paramsAfterGenesis(t, c, genesisWithParams(&GenesisParamsJSON{MultisigThreshold: 3}))
	if got.TvlCapBps != 3300 {
		t.Fatalf("TvlCapBps: got %d want 3300 (DefaultParams)", got.TvlCapBps)
	}
}

// No params block at all takes the DefaultParams path rather than
// paramsFromJSON, so cover it separately.
func TestGenesisTvlCapNoParamsBlockKeepsDefault(t *testing.T) {
	c, _ := newTestCanoliq()
	got := paramsAfterGenesis(t, c, miniGenesis())
	if got.TvlCapBps != 3300 {
		t.Fatalf("TvlCapBps: got %d want 3300 (DefaultParams)", got.TvlCapBps)
	}
}

// The point of the knob: an explicit 0 must survive into state as 0, and must
// read as "uncapped" rather than fail-closed — otherwise deposits would still
// be rejected, the opposite of the intent.
func TestGenesisTvlCapExplicitZeroUncaps(t *testing.T) {
	c, _ := newTestCanoliq()
	c.Config.Profile = ProfileDevnet
	zero := uint64(0)
	got := paramsAfterGenesis(t, c, genesisWithParams(&GenesisParamsJSON{TvlCapBps: &zero}))
	if got.TvlCapBps != 0 {
		t.Fatalf("TvlCapBps: got %d want 0 (explicit uncapped)", got.TvlCapBps)
	}
	if d := evaluateTVLCap(got.TvlCapBps, true, 1_000_000); d.Status != TVLCapStatusUncapped {
		t.Fatalf("status: got %q want %q", d.Status, TVLCapStatusUncapped)
	}
	// Uncapped must not fail closed when Canopy supply is absent either.
	if d := evaluateTVLCap(got.TvlCapBps, false, 0); d.Status != TVLCapStatusUncapped || d.Err != nil {
		t.Fatalf("uncapped must not fail closed with Canopy supply absent: %+v", d)
	}
}

// A non-zero override must still work, so a devnet can pick a deliberately low
// ceiling to exercise the T3 rejection path instead of turning the cap off.
func TestGenesisTvlCapExplicitNonZeroOverrides(t *testing.T) {
	c, _ := newTestCanoliq()
	low := uint64(25)
	got := paramsAfterGenesis(t, c, genesisWithParams(&GenesisParamsJSON{TvlCapBps: &low}))
	if got.TvlCapBps != 25 {
		t.Fatalf("TvlCapBps: got %d want 25", got.TvlCapBps)
	}
	d := evaluateTVLCap(got.TvlCapBps, true, 1_000_000)
	if d.Status != TVLCapStatusActive || d.CapUcnpy != 2_500 {
		t.Fatalf("decision: %+v want active cap=2500", d)
	}
}

// Both development profiles may ship an uncapped genesis. devnet is the one
// that matters here: it is the shared, hosted chain the team tests against.
func TestGenesisTvlCapZeroAllowedOnDevProfiles(t *testing.T) {
	for _, profile := range []string{ProfileLocalnet, ProfileDevnet} {
		c, _ := newTestCanoliq()
		c.Config.Profile = profile
		zero := uint64(0)
		got := paramsAfterGenesis(t, c, genesisWithParams(&GenesisParamsJSON{TvlCapBps: &zero}))
		if got.TvlCapBps != 0 {
			t.Fatalf("profile=%q TvlCapBps: got %d want 0", profile, got.TvlCapBps)
		}
	}
}

// testnet and mainnet must refuse to boot uncapped. The unset and unrecognized
// cases matter most: isDevProfile is a whitelist precisely so a profile nobody
// anticipated fails closed with the cap enforced.
func TestGenesisTvlCapZeroRefusedOutsideDevProfiles(t *testing.T) {
	for _, profile := range []string{ProfileMainnet, ProfileTestnet, "", "staging", "Devnet", "dev", "DEVNET"} {
		c, _ := newTestCanoliq()
		c.Config.Profile = profile
		zero := uint64(0)
		resp := c.Genesis(&contract.PluginGenesisRequest{
			GenesisJson: mustJSON(t, genesisWithParams(&GenesisParamsJSON{TvlCapBps: &zero})),
		})
		if resp.Error == nil {
			t.Fatalf("profile=%q: expected genesis to be refused", profile)
		}
		if want := ErrUncappedOutsideDevProfile(profile); resp.Error.Code != want.Code {
			t.Fatalf("profile=%q error code: got %d want %d", profile, resp.Error.Code, want.Code)
		}
	}
}

// The guard must be specific to the uncapped case — a normal capped genesis
// still has to boot on every profile, including production.
func TestGenesisNonZeroCapAllowedOnEveryProfile(t *testing.T) {
	for _, profile := range []string{ProfileMainnet, ProfileTestnet, ProfileDevnet, ProfileLocalnet, ""} {
		c, _ := newTestCanoliq()
		c.Config.Profile = profile
		capBps := uint64(3300)
		got := paramsAfterGenesis(t, c, genesisWithParams(&GenesisParamsJSON{TvlCapBps: &capBps}))
		if got.TvlCapBps != 3300 {
			t.Fatalf("profile=%q TvlCapBps: got %d want 3300", profile, got.TvlCapBps)
		}
	}
}

// Omitting the key entirely must boot everywhere — the guard keys on the
// resulting cap value, so a default-capped genesis is unaffected by profile.
func TestGenesisTvlCapAbsentBootsOnEveryProfile(t *testing.T) {
	for _, profile := range []string{ProfileMainnet, ProfileTestnet, ProfileDevnet, ProfileLocalnet, ""} {
		c, _ := newTestCanoliq()
		c.Config.Profile = profile
		got := paramsAfterGenesis(t, c, miniGenesis())
		if got.TvlCapBps != 3300 {
			t.Fatalf("profile=%q TvlCapBps: got %d want 3300", profile, got.TvlCapBps)
		}
	}
}

// devnet buys the uncapped-genesis affordance WITHOUT giving up the
// non-localnet guard rails. That trade is the whole reason it is a distinct
// profile rather than telling operators to run a shared chain as localnet:
// localnet skips SafetyCheck entirely, devnet must not.
func TestDevnetProfileStillEnforcesSafetyCheck(t *testing.T) {
	tooFast := Config{Profile: ProfileDevnet, ChainId: 404, RedemptionUnstakingBlocks: 5}
	if err := tooFast.SafetyCheck(); err == nil {
		t.Fatalf("devnet must enforce the redemptionUnstakingBlocks minimum")
	}
	local := Config{Profile: ProfileLocalnet, ChainId: 2, RedemptionUnstakingBlocks: 5}
	if err := local.SafetyCheck(); err != nil {
		t.Fatalf("localnet must stay exempt from SafetyCheck: %v", err)
	}
	ok := Config{Profile: ProfileDevnet, ChainId: 404, RedemptionUnstakingBlocks: 30240}
	if err := ok.SafetyCheck(); err != nil {
		t.Fatalf("devnet with a valid redemption window: %v", err)
	}
}

// runGenesis runs from BeginBlock, so its refusal surfaces as every block
// failing to apply rather than as a startup error. SafetyCheck catches the same
// condition before the node starts producing, which is the diagnostic an
// operator actually sees.
func TestSafetyCheckRejectsUncappedGenesisOffDevProfiles(t *testing.T) {
	write := func(t *testing.T, capBps *uint64) string {
		t.Helper()
		gf := miniGenesis()
		gf.Params = &GenesisParamsJSON{TvlCapBps: capBps}
		bz, err := json.Marshal(gf)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		p := filepath.Join(t.TempDir(), "genesis.json")
		if err := os.WriteFile(p, bz, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return p
	}
	zero := uint64(0)
	capped := uint64(3300)

	for _, profile := range []string{ProfileTestnet, ProfileMainnet} {
		cfg := Config{Profile: profile, RedemptionUnstakingBlocks: 30240, GenesisPath: write(t, &zero)}
		err := cfg.SafetyCheck()
		if err == nil {
			t.Fatalf("profile=%q: SafetyCheck must reject an uncapped genesis at startup", profile)
		}
		if !strings.Contains(err.Error(), "tvlCapBps=0") {
			t.Fatalf("profile=%q: error should name the offending field, got: %v", profile, err)
		}
	}
	// devnet is allowed to be uncapped and must still start.
	dev := Config{Profile: ProfileDevnet, RedemptionUnstakingBlocks: 30240, GenesisPath: write(t, &zero)}
	if err := dev.SafetyCheck(); err != nil {
		t.Fatalf("devnet must be allowed an uncapped genesis: %v", err)
	}
	// A capped genesis starts everywhere.
	for _, profile := range []string{ProfileTestnet, ProfileMainnet, ProfileDevnet} {
		cfg := Config{Profile: profile, RedemptionUnstakingBlocks: 30240, GenesisPath: write(t, &capped)}
		if err := cfg.SafetyCheck(); err != nil {
			t.Fatalf("profile=%q with a capped genesis: %v", profile, err)
		}
	}
}

// isDevProfile is the single whitelist the genesis guard keys on; pin its
// membership directly so a future edit cannot widen it silently.
func TestIsDevProfileMembership(t *testing.T) {
	for _, p := range []string{ProfileLocalnet, ProfileDevnet} {
		if !isDevProfile(p) {
			t.Fatalf("%q must be a dev profile", p)
		}
	}
	for _, p := range []string{ProfileTestnet, ProfileMainnet, "", "staging", "Devnet", "dev", "DEVNET"} {
		if isDevProfile(p) {
			t.Fatalf("%q must NOT be a dev profile", p)
		}
	}
}
