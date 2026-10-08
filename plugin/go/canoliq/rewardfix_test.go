package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// rewardfix_test.go covers the mainnet activation of the reward-ownership fix
// (rewardfix.go): what changes at MainnetRewardFixHeight, and the one-time
// correction of the balances the pre-fix sweep inflated.

var (
	fixValA = parityHex("737ef80e8476450b6970dd6c1e44c93699c8e0c0")
	fixDel  = parityHex("725f4f9983aaa3c60cc318bff1a196f86c65eb97")
)

func newMainnetFixCanoliq(t *testing.T) (*Canoliq, *fakeStore) {
	t.Helper()
	c, s := newTestCanoliq()
	cfg := Config{ChainId: 29, DataDirPath: "/tmp/canoliq-test", Profile: ProfileMainnet}
	c.Config = cfg
	c.plugin.config = cfg
	seedParams(t, c, DefaultParams())
	return c, s
}

// seedInflatedMainnetState reproduces committee 29 as the pre-fix sweep left
// it: the two real deposits, one queued redemption, and every protocol
// balance funded out of Canopy Foundation's bond growth.
func seedInflatedMainnetState(t *testing.T, c *Canoliq, s *fakeStore) {
	t.Helper()
	seedGlobals(s, &contract.CanoliqGlobals{
		GenesisComplete:         true,
		TotalCcnpySupply:        fixExpectCcnpySupply,
		TotalPooledCnpy:         113_513_023_501,
		PendingRedemptionCnpy:   fixExpectPending,
		LastProcessedRewardPool: 139_814_328_178,
		PeakTvlUcnpy:            114_143_808_301,
		NextRedemptionId:        fixExpectNextRedeemID,
		CplqTotalSupply:         CPLQTotalSupply,
	})
	seedEscrow(s, 113_513_023_501+fixExpectPending)
	s.set(KeyForCCNPYBalance(fixHolderA), EncodeUint64(fixExpectHolderACcnpy))
	s.set(KeyForCCNPYBalance(fixHolderB), EncodeUint64(fixExpectHolderBCcnpy))
	s.set(KeyForRedemption(fixHolderA, 0), mustMarshal(&contract.Redemption{
		Id: 0, Address: fixHolderA, CnpyAmount: fixExpectRedemptionAmt, UnbondCompleteHeight: 36_578,
	}))
	s.set(KeyForTreasuryCNPY(), EncodeUint64(10_730_623_747))
	s.set(KeyForBuybackPool(), EncodeUint64(2_362_445_243))
	s.set(KeyForInsurancePool(), EncodeUint64(224_160_774))
	s.set(KeyForTxFeeAccrual(), EncodeUint64(40_000))
	s.set(KeyForValidatorIncentives(fixDel), EncodeUint64(748_039_096))
	s.set(KeyForValidatorIncentives(fixValA), EncodeUint64(1_614_406_147))
	s.set(KeyForValidatorRegistry(), mustMarshal(&contract.ValidatorRegistry{
		Entries: []*contract.ValidatorRegistryEntry{
			{Address: fixDel, Stake: 36_224_709_978},
			{Address: fixValA, Stake: 104_269_343_200},
		},
	}))
	paritySetValidator(t, s, fixDel, 36_224_709_978, true)
	paritySetValidator(t, s, fixValA, 104_269_343_200, false)
}

// runBlock drives one BeginBlock / EndBlock pair, growing both foreign bonds
// by their observed per-block rewards in between, the way core compounds them.
func runBlock(t *testing.T, c *Canoliq, s *fakeStore, h uint64, valStake, delStake *uint64) {
	t.Helper()
	c.plugin.setHeight(h)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: h}); r.Error != nil {
		t.Fatalf("begin block %d: %v", h, r.Error)
	}
	*valStake += 11_400_000
	*delStake += 1_425_000
	paritySetValidator(t, s, fixValA, *valStake, false)
	paritySetValidator(t, s, fixDel, *delStake, true)
	if r := c.EndBlock(&contract.PluginEndRequest{Height: h}); r.Error != nil {
		t.Fatalf("end block %d: %v", h, r.Error)
	}
}

// TestMainnetRewardFixCorrectsInflatedState walks committee 29 across the
// activation height: the block before still credits foreign growth (it must,
// to replay as produced), the activation block restores the real principal,
// and no block after it credits anything.
func TestMainnetRewardFixCorrectsInflatedState(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedInflatedMainnetState(t, c, s)
	valStake, delStake := uint64(104_269_343_200), uint64(36_224_709_978)

	runBlock(t, c, s, MainnetRewardFixHeight-1, &valStake, &delStake)
	if g := loadGlobals(t, s); g.TotalPooledCnpy <= 113_513_023_501 {
		t.Fatalf("pre-activation block must still run the pre-fix sweep: pooled=%d", g.TotalPooledCnpy)
	}

	runBlock(t, c, s, MainnetRewardFixHeight, &valStake, &delStake)
	g := loadGlobals(t, s)
	if g.TotalCcnpySupply != fixCcnpySupply || g.TotalPooledCnpy != fixPooled || g.PendingRedemptionCnpy != fixPending {
		t.Fatalf("corrected globals: supply=%d pooled=%d pending=%d", g.TotalCcnpySupply, g.TotalPooledCnpy, g.PendingRedemptionCnpy)
	}
	if got := readEscrow(s); got != 518_500_661 || got != g.TotalPooledCnpy+g.PendingRedemptionCnpy {
		t.Fatalf("escrow %d must equal the 518.500661 CNLQ deposited and back pooled+pending", got)
	}
	if a, b := readCcnpy(s, fixHolderA), readCcnpy(s, fixHolderB); a != 8_000_000 || b != 508_500_661 {
		t.Fatalf("holder balances: 111e=%d 9ddb=%d", a, b)
	}
	red := new(contract.Redemption)
	if err := contract.Unmarshal(s.get(KeyForRedemption(fixHolderA, 0)), red); err != nil {
		t.Fatal(err)
	}
	if red.CnpyAmount != 2_000_000 || red.UnbondCompleteHeight != 36_578 {
		t.Fatalf("redemption #0: amount=%d unbond=%d", red.CnpyAmount, red.UnbondCompleteHeight)
	}
	for name, key := range map[string][]byte{
		"treasury":       KeyForTreasuryCNPY(),
		"buyback":        KeyForBuybackPool(),
		"insurance":      KeyForInsurancePool(),
		"tx-fee accrual": KeyForTxFeeAccrual(),
	} {
		if v := DecodeUint64(s.get(key)); v != 0 {
			t.Errorf("%s: got %d want 0", name, v)
		}
	}
	if v := readAllValidatorIncentives(s); v != 0 {
		t.Errorf("validator incentives: got %d want 0", v)
	}
	for _, e := range loadRegistryEntries(t, s) {
		if e.Owned {
			t.Errorf("foreign bond %x marked owned", e.Address)
		}
	}

	// Many blocks of foreign compounding later, nothing has been credited.
	for h := MainnetRewardFixHeight + 1; h <= MainnetRewardFixHeight+50; h++ {
		runBlock(t, c, s, h, &valStake, &delStake)
	}
	g = loadGlobals(t, s)
	if g.TotalPooledCnpy != fixPooled || g.LastAttributedReward != 0 || readEscrow(s) != 518_500_661 {
		t.Fatalf("post-fix block credited foreign growth: pooled=%d attributed=%d escrow=%d",
			g.TotalPooledCnpy, g.LastAttributedReward, readEscrow(s))
	}
	if v := DecodeUint64(s.get(KeyForTreasuryCNPY())) + readAllValidatorIncentives(s) +
		DecodeUint64(s.get(KeyForBuybackPool())); v != 0 {
		t.Fatalf("post-fix block funded protocol balances: %d", v)
	}
}

// TestMainnetRewardCorrectionSkipsUnexpectedState: the correction writes fixed
// values, so it must refuse to run over any state it was not written against —
// here a third deposit that landed before the activation height.
func TestMainnetRewardCorrectionSkipsUnexpectedState(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedInflatedMainnetState(t, c, s)
	g := loadGlobals(t, s)
	g.TotalCcnpySupply += 1_000
	seedGlobals(s, g)
	valStake, delStake := uint64(104_269_343_200), uint64(36_224_709_978)

	runBlock(t, c, s, MainnetRewardFixHeight, &valStake, &delStake)
	if got := readCcnpy(s, fixHolderB); got != fixExpectHolderBCcnpy {
		t.Fatalf("correction ran over unexpected state: 9ddb=%d", got)
	}
	// The fix itself is still active: foreign growth is not credited.
	before := loadGlobals(t, s).TotalPooledCnpy
	runBlock(t, c, s, MainnetRewardFixHeight+1, &valStake, &delStake)
	if after := loadGlobals(t, s).TotalPooledCnpy; after != before {
		t.Fatalf("fix inactive after skipped correction: pooled %d -> %d", before, after)
	}
}

// TestMainnetTVLAwaitingStakeGatedOnFixHeight: with Canopy Supply readable but
// Staked == 0, a deposit was accepted uncapped before the fix (and must still
// replay that way) and is rejected from the activation height on.
func TestMainnetTVLAwaitingStakeGatedOnFixHeight(t *testing.T) {
	for _, tc := range []struct {
		height uint64
		accept bool
	}{
		{MainnetRewardFixHeight - 1, true},
		{MainnetRewardFixHeight, false},
	} {
		c, s := newMainnetFixCanoliq(t)
		paritySetSupply(t, s, 0)
		seedGlobals(s, &contract.CanoliqGlobals{GenesisComplete: true})
		user := addr20(0x01)
		seedAccount(s, user, 2_000_000)
		c.plugin.setHeight(tc.height)
		dep, _ := anypb.New(&contract.MessageCanoliqDeposit{FromAddress: user, Amount: 1_000_000})
		r := c.DeliverTx(&contract.PluginDeliverRequest{Tx: &contract.Transaction{Msg: dep, Fee: 10_000}})
		if accepted := r.Error == nil; accepted != tc.accept {
			t.Errorf("height %d: accepted=%v want %v (err=%v)", tc.height, accepted, tc.accept, r.Error)
		}
	}
}

// TestNonMainnetProfilesRunTheFixFromGenesis: only mainnet has pre-fix history
// to preserve.
func TestNonMainnetProfilesRunTheFixFromGenesis(t *testing.T) {
	for _, p := range []string{"", ProfileLocalnet, ProfileDevnet, ProfileTestnet} {
		c := &Canoliq{Config: Config{Profile: p}}
		if !c.rewardFixActive(1) {
			t.Errorf("profile %q: fix must be active from genesis", p)
		}
	}
	c := &Canoliq{Config: Config{Profile: ProfileMainnet}}
	if c.rewardFixActive(MainnetRewardFixHeight-1) || !c.rewardFixActive(MainnetRewardFixHeight) {
		t.Error("mainnet: fix must switch on exactly at MainnetRewardFixHeight")
	}
}
