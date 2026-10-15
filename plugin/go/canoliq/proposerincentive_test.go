package canoliq

import (
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// proposerincentive_test.go covers the switch of the validator slice from the
// registry-weighted ledger to each block's proposer (proposerincentive.go).

var (
	rootValidatorA = parityHex("2b6828411e7eb87ab4ab17c6bd6ae685d46e5f96")
	rootValidatorB = parityHex("c849bbf138000000000000000000000000000001")
)

func accountBalance(t *testing.T, s *fakeStore, addr []byte) uint64 {
	t.Helper()
	bz := s.get(contract.KeyForAccount(addr))
	if bz == nil {
		return 0
	}
	a := new(contract.Account)
	if err := contract.Unmarshal(bz, a); err != nil {
		t.Fatal(err)
	}
	return a.Amount
}

type proposerFixture struct {
	t    *testing.T
	c    *Canoliq
	s    *fakeStore
	bond uint64
	val  uint64
	del  uint64
}

const proposerBondGrowth = 1_300_000

func newProposerFixture(t *testing.T) *proposerFixture {
	c, s := newMainnetFixCanoliq(t)
	p := DefaultParams()
	p.StakeOutputAddresses = [][]byte{OwnedBondOutput}
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
	f := &proposerFixture{t: t, c: c, s: s, bond: 1_000_000_000_000, val: 115_213_343_200, del: 37_592_709_978}
	// The owned bond, val-a and a delegate are on the committee from the start;
	// one warm-up block baselines them so later blocks read pure growth.
	f.write()
	f.block(MainnetProposerIncentiveHeight-50, rootValidatorA)
	return f
}

func (f *proposerFixture) write() {
	paritySetValidator(f.t, f.s, fixValA, f.val, false)
	paritySetValidator(f.t, f.s, fixDel, f.del, true)
	setDelegateWithOutput(f.t, f.s, teamOperator, OwnedBondOutput, f.bond)
}

func (f *proposerFixture) block(h uint64, proposer []byte) {
	f.t.Helper()
	f.c.plugin.setHeight(h)
	if r := f.c.BeginBlock(&contract.PluginBeginRequest{Height: h}); r.Error != nil {
		f.t.Fatalf("begin block %d: %v", h, r.Error)
	}
	f.bond += proposerBondGrowth
	f.val += 11_400_000
	f.del += 125_000
	f.write()
	if r := f.c.EndBlock(&contract.PluginEndRequest{Height: h, ProposerAddress: proposer}); r.Error != nil {
		f.t.Fatalf("end block %d: %v", h, r.Error)
	}
}

// validatorShare is the validator slice of one block's reward under the
// fixture's fee params.
func validatorShare(t *testing.T, c *Canoliq, reward uint64) uint64 {
	t.Helper()
	p, err := c.LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	return SplitFee(FeeOnReward(reward, p.FeeBps), &FeeSplitParams{
		UserRebateBps: p.UserRebateBps, TreasuryBps: p.TreasuryBps, ValidatorBps: p.ValidatorBps, BuybackBps: p.BuybackBps,
	}).Validators
}

func TestValidatorSliceUsesLedgerBeforeActivation(t *testing.T) {
	f := newProposerFixture(t)
	for h := MainnetProposerIncentiveHeight - 10; h < MainnetProposerIncentiveHeight; h++ {
		f.block(h, rootValidatorA)
	}
	if got := accountBalance(t, f.s, rootValidatorA); got != 0 {
		t.Fatalf("proposer paid %d before activation", got)
	}
	if readAllValidatorIncentives(f.s) == 0 {
		t.Fatal("ledger did not accrue before activation")
	}
}

func TestActivationMovesLedgerToTreasuryOnce(t *testing.T) {
	f := newProposerFixture(t)
	for h := MainnetProposerIncentiveHeight - 10; h < MainnetProposerIncentiveHeight; h++ {
		f.block(h, rootValidatorA)
	}
	// A stale aggregator entry from the legacy path must move too.
	aggKey := KeyForValidatorIncentives(f.c.committeeAggregatorAddr())
	f.s.set(aggKey, EncodeUint64(7))
	ledger := readAllValidatorIncentives(f.s)
	treasuryBefore := DecodeUint64(f.s.get(KeyForTreasuryCNPY()))

	f.c.plugin.setHeight(MainnetProposerIncentiveHeight)
	if err := f.c.applyMainnetIncentiveMigration(MainnetProposerIncentiveHeight); err != nil {
		t.Fatal(err)
	}
	if got := readAllValidatorIncentives(f.s); got != 0 {
		t.Fatalf("ledger not cleared: %d left", got)
	}
	// The registry shares the key domain and must be left alone.
	if f.s.get(KeyForValidatorRegistry()) == nil || len(loadRegistryEntries(t, f.s)) == 0 {
		t.Fatal("migration deleted the validator registry")
	}
	if got := DecodeUint64(f.s.get(KeyForTreasuryCNPY())); got != treasuryBefore+ledger {
		t.Fatalf("treasury %d, want %d + %d", got, treasuryBefore, ledger)
	}
	// Idempotent, and inert at any other height.
	for _, h := range []uint64{MainnetProposerIncentiveHeight, MainnetProposerIncentiveHeight + 1} {
		if err := f.c.applyMainnetIncentiveMigration(h); err != nil {
			t.Fatal(err)
		}
	}
	if got := DecodeUint64(f.s.get(KeyForTreasuryCNPY())); got != treasuryBefore+ledger {
		t.Fatalf("migration ran twice: treasury %d", got)
	}
}

func TestProposerIsPaidTheValidatorSliceAfterActivation(t *testing.T) {
	f := newProposerFixture(t)
	for h := MainnetProposerIncentiveHeight - 3; h < MainnetProposerIncentiveHeight; h++ {
		f.block(h, rootValidatorA)
	}
	start := readSwept(t, f.s)
	balA0, balB0 := accountBalance(t, f.s, rootValidatorA), accountBalance(t, f.s, rootValidatorB)

	const n = 12
	var wantA, wantB, rewards uint64
	for i := uint64(0); i < n; i++ {
		h := MainnetProposerIncentiveHeight + i
		proposer := rootValidatorA
		if i%3 == 2 {
			proposer = rootValidatorB
		}
		f.block(h, proposer)
		g := loadGlobals(t, f.s)
		share := validatorShare(t, f.c, g.LastAttributedReward)
		rewards += g.LastAttributedReward
		if share == 0 {
			t.Fatalf("block %d: zero validator share", h)
		}
		if proposer[0] == rootValidatorA[0] {
			wantA += share
		} else {
			wantB += share
		}
	}
	if got := accountBalance(t, f.s, rootValidatorA) - balA0; got != wantA {
		t.Fatalf("proposer A paid %d, want %d", got, wantA)
	}
	if got := accountBalance(t, f.s, rootValidatorB) - balB0; got != wantB {
		t.Fatalf("proposer B paid %d, want %d", got, wantB)
	}
	end := readSwept(t, f.s)
	if end.validators != 0 {
		t.Fatalf("ledger accrued %d after activation", end.validators)
	}
	// Conservation: every unit of reward lands somewhere, exactly once.
	distributed := (end.pooled - start.pooled) + (end.treasury - start.treasury - start.validators) +
		(end.insurance - start.insurance) + (end.buyback - start.buyback) + wantA + wantB
	if distributed != rewards {
		t.Fatalf("distributed %d, credited %d", distributed, rewards)
	}
}

func TestSliceGoesToTreasuryWithoutAnEligibleProposer(t *testing.T) {
	for _, tc := range []struct {
		name     string
		proposer []byte
		eject    bool
	}{
		{"no proposer", nil, false},
		{"malformed proposer", []byte{1, 2, 3}, false},
		{"ejected proposer", rootValidatorB, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newProposerFixture(t)
			f.block(MainnetProposerIncentiveHeight, rootValidatorA) // migrate + baseline
			if tc.eject {
				f.s.set(KeyForEjectedValidator(tc.proposer), EncodeUint64(1))
			}
			before := readSwept(t, f.s)
			f.block(MainnetProposerIncentiveHeight+1, tc.proposer)
			after := readSwept(t, f.s)
			g := loadGlobals(t, f.s)
			p, _ := f.c.LoadParams()
			split := SplitFee(FeeOnReward(g.LastAttributedReward, p.FeeBps), &FeeSplitParams{
				UserRebateBps: p.UserRebateBps, TreasuryBps: p.TreasuryBps, ValidatorBps: p.ValidatorBps, BuybackBps: p.BuybackBps,
			})
			insurance := after.insurance - before.insurance
			wantTreasury := split.Treasury - insurance + split.Validators
			if got := after.treasury - before.treasury; got != wantTreasury {
				t.Fatalf("treasury got %d, want %d (validator slice %d)", got, wantTreasury, split.Validators)
			}
			if tc.eject && accountBalance(t, f.s, tc.proposer) != 0 {
				t.Fatal("ejected proposer was paid")
			}
		})
	}
}

func TestProposerIncentiveIsMainnetOnly(t *testing.T) {
	c, _ := newTestCanoliq()
	if c.proposerIncentiveActive(MainnetProposerIncentiveHeight + 1_000) {
		t.Fatal("proposer incentive active on a non-mainnet profile")
	}
	m, _ := newMainnetFixCanoliq(t)
	if m.proposerIncentiveActive(MainnetProposerIncentiveHeight-1) || !m.proposerIncentiveActive(MainnetProposerIncentiveHeight) {
		t.Fatal("mainnet activation boundary wrong")
	}
}
