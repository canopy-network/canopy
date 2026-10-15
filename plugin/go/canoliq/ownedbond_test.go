package canoliq

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/encoding/protowire"
)

// ownedbond_test.go covers the owned-bond activation (ownedbond.go): the
// keyless output joins stake_output_addresses at MainnetOwnedBondHeight, and
// from then on exactly that bond's growth, and no foreign growth, lifts cCNPY.

var teamOperator = parityHex("d5c74cd0eba543f4fc4ad92a541250e29838d82d")

// setDelegateWithOutput writes a core delegate record on committee 29 with an
// explicit output, byte-compatible with core's Validator encoding.
func setDelegateWithOutput(t *testing.T, s *fakeStore, addr, output []byte, stake uint64) {
	t.Helper()
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendBytes(b, addr)
	b = protowire.AppendTag(b, 4, protowire.VarintType)
	b = protowire.AppendVarint(b, stake)
	b = protowire.AppendTag(b, 5, protowire.BytesType)
	b = protowire.AppendBytes(b, protowire.AppendVarint(nil, 29))
	b = protowire.AppendTag(b, 8, protowire.BytesType)
	b = protowire.AppendBytes(b, output)
	b = protowire.AppendTag(b, 9, protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	b = protowire.AppendTag(b, 10, protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	s.set(contract.KeyForValidator(addr), b)
}

func TestOwnedBondOutputIsTheDocumentedKeylessHash(t *testing.T) {
	sum := sha256.Sum256([]byte("canoliq/owned-reward-bond/v1"))
	if want := hex.EncodeToString(sum[:20]); hex.EncodeToString(OwnedBondOutput) != want {
		t.Fatalf("OwnedBondOutput %x is not sha256(\"canoliq/owned-reward-bond/v1\")[:20] = %s", OwnedBondOutput, want)
	}
}

// TestMainnetOwnedBondActivation walks committee 29 in its post-correction
// state across MainnetOwnedBondHeight. The team's bond is already on the
// committee and growing; val-a and 725f keep compounding too.
func TestMainnetOwnedBondActivation(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	// Params as stored on chain today: the record predates the two new fields.
	p := DefaultParams()
	p.MaxRewardBpsPerBlock = 0
	seedParams(t, c, p)
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete:       true,
		TotalCcnpySupply:      fixCcnpySupply,
		TotalPooledCnpy:       fixPooled,
		PendingRedemptionCnpy: fixPending,
		PeakTvlUcnpy:          fixPooled,
		NextRedemptionId:      1,
	})
	seedEscrow(s, fixEscrow)

	const bondGrowth = 1_300_000 // ~96% of the ~1.425 CNLQ delegate cut
	valStake, delStake, bond := uint64(115_213_343_200), uint64(37_592_709_978), uint64(1_000_000_000_000)
	block := func(h uint64) {
		t.Helper()
		c.plugin.setHeight(h)
		if r := c.BeginBlock(&contract.PluginBeginRequest{Height: h}); r.Error != nil {
			t.Fatalf("begin block %d: %v", h, r.Error)
		}
		valStake += 11_400_000
		delStake += 125_000
		bond += bondGrowth
		paritySetValidator(t, s, fixValA, valStake, false)
		paritySetValidator(t, s, fixDel, delStake, true)
		setDelegateWithOutput(t, s, teamOperator, OwnedBondOutput, bond)
		if r := c.EndBlock(&contract.PluginEndRequest{Height: h}); r.Error != nil {
			t.Fatalf("end block %d: %v", h, r.Error)
		}
	}

	// The bond joins the committee a few blocks before activation.
	for h := MainnetOwnedBondHeight - 5; h < MainnetOwnedBondHeight; h++ {
		block(h)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != fixPooled {
		t.Fatalf("bond credited before activation: pooled=%d", g.TotalPooledCnpy)
	}

	// Activation block: the address is added, and the pre-existing bond is
	// only baselined, never read as one giant reward.
	block(MainnetOwnedBondHeight)
	params, err := c.LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.StakeOutputAddresses) != 1 || !bytes.Equal(params.StakeOutputAddresses[0], OwnedBondOutput) {
		t.Fatalf("stake_output_addresses: %x", params.StakeOutputAddresses)
	}
	if params.MaxRewardBpsPerBlock != 100 {
		t.Fatalf("max_reward_bps_per_block: got %d want 100", params.MaxRewardBpsPerBlock)
	}
	if g := loadGlobals(t, s); g.TotalPooledCnpy != fixPooled {
		t.Fatalf("activation block credited the existing bond as reward: pooled=%d", g.TotalPooledCnpy)
	}
	for _, e := range loadRegistryEntries(t, s) {
		if bytes.Equal(e.Address, teamOperator) != e.Owned {
			t.Errorf("registry owned flag for %x: %v", e.Address, e.Owned)
		}
	}

	// Every block after: exactly the bond's growth is distributed, split per
	// the fee params, and foreign growth reaches nothing.
	const n = 20
	for h := MainnetOwnedBondHeight + 1; h <= MainnetOwnedBondHeight+n; h++ {
		block(h)
	}
	got := readSwept(t, s)
	distributed := (got.pooled - fixPooled) + got.treasury + got.insurance + got.buyback + got.validators
	if distributed != n*bondGrowth {
		t.Fatalf("distributed %d over %d blocks, want exactly the bond's growth %d", distributed, n, n*bondGrowth)
	}
	if got.pooled <= fixPooled {
		t.Fatal("cCNPY exchange rate did not lift")
	}
	if esc := readEscrow(s); esc != got.pooled+fixPending {
		t.Fatalf("escrow %d != pooled %d + pending %d", esc, got.pooled, fixPending)
	}
}

// TestMainnetOwnedBondActivationIsIdempotent: re-running the activation (or
// finding the address already listed) must not duplicate it.
func TestMainnetOwnedBondActivationIsIdempotent(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
	for i := 0; i < 2; i++ {
		if err := c.applyMainnetOwnedBond(MainnetOwnedBondHeight); err != nil {
			t.Fatal(err)
		}
	}
	params, err := c.LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.StakeOutputAddresses) != 1 {
		t.Fatalf("stake_output_addresses duplicated: %x", params.StakeOutputAddresses)
	}
	if err := c.applyMainnetOwnedBond(MainnetOwnedBondHeight + 1); err != nil {
		t.Fatal(err)
	}
}
