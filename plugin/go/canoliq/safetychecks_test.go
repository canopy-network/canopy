package canoliq

import (
	"math"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

const rangeTestFee = uint64(10_000)

// newMainnetAt returns a mainnet-profile Canoliq at `height` with default
// params saved.
func newMainnetAt(t *testing.T, height uint64) (*Canoliq, *fakeStore) {
	t.Helper()
	c, s := newTestCanoliq()
	c.Config.Profile = ProfileMainnet
	c.plugin.config.Profile = ProfileMainnet
	c.plugin.setHeight(height)
	seedParams(t, c, DefaultParams())
	return c, s
}

func isRangeErr(err *contract.PluginError) bool {
	return err != nil && err.Code == codeAmountOutOfRange
}

// rangeCase is one guarded deliver path: setup seeds state at the edge of the
// uint64 range, run delivers the message. Before activation the legacy
// result (wantPre) must hold; from activation the tx is rejected with
// wantPost and leaves state untouched.
type rangeCase struct {
	name    string
	setup   func(t *testing.T, c *Canoliq, s *fakeStore)
	run     func(c *Canoliq) *contract.PluginDeliverResponse
	wantPre func(err *contract.PluginError) bool
	// wantPost is the expected code from activation; 0 means codeAmountOutOfRange.
	wantPost uint64
}

func preOK(err *contract.PluginError) bool { return err == nil }

func runRangeCases(t *testing.T, cases []rangeCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name+"/before_activation", func(t *testing.T) {
			c, s := newMainnetAt(t, MainnetSafetyChecksHeight-1)
			tc.setup(t, c, s)
			resp := tc.run(c)
			if isRangeErr(resp.Error) || !tc.wantPre(resp.Error) {
				t.Fatalf("legacy behaviour changed before activation: %v", resp.Error)
			}
		})
		t.Run(tc.name+"/from_activation", func(t *testing.T) {
			c, s := newMainnetAt(t, MainnetSafetyChecksHeight)
			tc.setup(t, c, s)
			before := parityStateHash(s)
			resp := tc.run(c)
			want := tc.wantPost
			if want == 0 {
				want = codeAmountOutOfRange
			}
			if resp.Error == nil || resp.Error.Code != want {
				t.Fatalf("want error code %d, got %v", want, resp.Error)
			}
			if parityStateHash(s) != before {
				t.Fatal("rejected tx modified state")
			}
		})
	}
}

func TestDeliverRejectsAmountFeeOverflow(t *testing.T) {
	a, b := addr20(0xA1), addr20(0xB2)
	edgeAmount := uint64(math.MaxUint64) - rangeTestFee + 2
	runRangeCases(t, []rangeCase{
		{
			name: "deposit",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedGlobals(s, &contract.CanoliqGlobals{TotalCcnpySupply: 1_000_000, TotalPooledCnpy: 1_000_000, GenesisComplete: true})
				seedEscrow(s, 1_000_000)
				seedAccount(s, a, 2*rangeTestFee)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: a, Amount: edgeAmount}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "send",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, rangeTestFee+1)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.deliverMessageSend(&contract.MessageSend{FromAddress: a, ToAddress: b, Amount: edgeAmount}, rangeTestFee)
			},
			wantPre: preOK,
		},
	})
}

func TestDeliverRejectsBalanceOverflow(t *testing.T) {
	a, b := addr20(0xA1), addr20(0xB2)
	near := uint64(math.MaxUint64 - 5)
	runRangeCases(t, []rangeCase{
		{
			name: "send_recipient",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				seedAccount(s, b, near)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.deliverMessageSend(&contract.MessageSend{FromAddress: a, ToAddress: b, Amount: 10}, rangeTestFee)
			},
			wantPre: preOK,
		},
		{
			name: "send_fee_pool",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				bz, _ := contract.Marshal(&contract.Pool{Id: c.Config.ChainId, Amount: near})
				s.set(contract.KeyForFeePool(c.Config.ChainId), bz)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.deliverMessageSend(&contract.MessageSend{FromAddress: a, ToAddress: b, Amount: 10}, rangeTestFee)
			},
			wantPre: preOK,
		},
		{
			name: "redeem_pending",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedGlobals(s, &contract.CanoliqGlobals{TotalCcnpySupply: 1_000_000, TotalPooledCnpy: 1_000_000, PendingRedemptionCnpy: near, GenesisComplete: true})
				seedAccount(s, a, 1_000_000)
				seedCcnpy(s, a, 1_000_000)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCanoliqRedeem(&contract.MessageCanoliqRedeem{FromAddress: a, CcnpyAmount: 500_000}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "claim_redemption",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedGlobals(s, &contract.CanoliqGlobals{PendingRedemptionCnpy: 2 * rangeTestFee, GenesisComplete: true})
				seedEscrow(s, 2*rangeTestFee)
				seedAccount(s, a, near)
				bz, _ := contract.Marshal(&contract.Redemption{Id: 0, Address: a, CnpyAmount: 2 * rangeTestFee})
				s.set(KeyForRedemption(a, 0), bz)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCanoliqClaimRedemption(&contract.MessageCanoliqClaimRedemption{FromAddress: a, RedemptionId: 0}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "ccnpy_transfer",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				seedCcnpy(s, a, 100)
				seedCcnpy(s, b, near)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCanoliqTransfer(&contract.MessageCanoliqTransfer{FromAddress: a, ToAddress: b, Amount: 10}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "cplq_transfer",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				seedCPLQ(s, a, 100)
				seedCPLQ(s, b, near)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCPLQTransfer(&contract.MessageCPLQTransfer{FromAddress: a, ToAddress: b, Amount: 10}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "cplq_stake_total",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedGlobals(s, &contract.CanoliqGlobals{TotalStakedCplq: near, GenesisComplete: true})
				seedAccount(s, a, 1_000_000)
				seedCPLQ(s, a, 100)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCPLQStake(&contract.MessageCPLQStake{FromAddress: a, Amount: 10}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "cplq_claim_unstake",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				seedCPLQ(s, a, near)
				bz, _ := contract.Marshal(&contract.UnstakingCPLQ{Id: 0, Address: a, Amount: 10, MatureHeight: 1})
				s.set(KeyForCPLQUnstaking(a, 0), bz)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageCPLQClaimUnstake(&contract.MessageCPLQClaimUnstake{FromAddress: a, UnstakeId: 0}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
		{
			name: "treasury_spend_recipient",
			setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
				seedAccount(s, a, 1_000_000)
				seedAccount(s, b, near)
				s.set(KeyForTreasuryCNPY(), EncodeUint64(1_000))
				spend := &contract.TreasurySpend{Id: 1, ProposalId: 9,
					Payload: &contract.ProposalTreasurySpend{Recipient: b, Amount: 100, Denomination: contract.SpendDenomination_SPEND_CNPY}}
				bz, _ := contract.Marshal(spend)
				s.set(KeyForTreasurySpend(1), bz)
				ib, _ := contract.Marshal(&contract.ProposalIndex{Ids: []uint64{1}})
				s.set(KeyForSpendIndex(), ib)
			},
			run: func(c *Canoliq) *contract.PluginDeliverResponse {
				return c.DeliverMessageDAOTreasurySpend(&contract.MessageDAOTreasurySpend{FromAddress: a, ProposalId: 9}, rangeTestFee, DefaultParams())
			},
			wantPre: preOK,
		},
	})
}

func TestDeliverDepositTvlCapComparisonCannotWrap(t *testing.T) {
	a := addr20(0xA1)
	// amount+fee fits, but pooled+amount does not.
	amount := uint64(math.MaxUint64 - rangeTestFee)
	runRangeCases(t, []rangeCase{{
		name: "fixed_cap",
		setup: func(t *testing.T, c *Canoliq, s *fakeStore) {
			seedGlobals(s, &contract.CanoliqGlobals{TotalCcnpySupply: 1_000_000, TotalPooledCnpy: 1_000_000, GenesisComplete: true})
			seedEscrow(s, 1_000_000)
			seedAccount(s, a, 1_000_000)
		},
		run: func(c *Canoliq) *contract.PluginDeliverResponse {
			return c.DeliverMessageCanoliqDeposit(&contract.MessageCanoliqDeposit{FromAddress: a, Amount: amount}, rangeTestFee, DefaultParams())
		},
		// Legacy: the wrapped sum passes the cap and the balance check rejects.
		wantPre:  func(err *contract.PluginError) bool { return err != nil && err.Code == codeInsufficientCNPY },
		wantPost: codeTVLCapExceeded,
	}})
}

func TestDeliverRejectsTxFeeAccrualOverflow(t *testing.T) {
	for _, tc := range []struct {
		height  uint64
		wantErr bool
	}{{MainnetSafetyChecksHeight - 1, false}, {MainnetSafetyChecksHeight, true}} {
		c, s := newMainnetAt(t, tc.height)
		s.set(KeyForTxFeeAccrual(), EncodeUint64(math.MaxUint64-5))
		err := c.accrueTxFee(rangeTestFee)
		if (err != nil) != tc.wantErr || (err != nil && !isRangeErr(err)) {
			t.Fatalf("height %d: got %v, wantErr=%v", tc.height, err, tc.wantErr)
		}
	}
}

func TestCheckRejectsAmountFeeOverflowAndFeeAboveMaximum(t *testing.T) {
	a, b := addr20(0xA1), addr20(0xB2)
	mk := func(t *testing.T, msg *contract.MessageSend, fee uint64) *contract.PluginCheckRequest {
		any, err := anypb.New(msg)
		if err != nil {
			t.Fatal(err)
		}
		return &contract.PluginCheckRequest{Tx: &contract.Transaction{Msg: any, Fee: fee}}
	}
	cases := []struct {
		name     string
		amount   uint64
		fee      uint64
		wantCode uint64
	}{
		{"amount_plus_fee", math.MaxUint64 - rangeTestFee + 1, rangeTestFee, codeAmountOutOfRange},
		{"fee_above_maximum", 1, MaxTxFeeUcnpy + 1, codeFeeAboveMaximum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newMainnetAt(t, MainnetSafetyChecksHeight-1)
			if r := c.CheckTx(mk(t, &contract.MessageSend{FromAddress: a, ToAddress: b, Amount: tc.amount}, tc.fee)); r.Error != nil {
				t.Fatalf("before activation: %v", r.Error)
			}
			c, _ = newMainnetAt(t, MainnetSafetyChecksHeight)
			r := c.CheckTx(mk(t, &contract.MessageSend{FromAddress: a, ToAddress: b, Amount: tc.amount}, tc.fee))
			if r.Error == nil || r.Error.Code != tc.wantCode {
				t.Fatalf("from activation: want code %d, got %v", tc.wantCode, r.Error)
			}
			// A normal tx is still accepted.
			if r := c.CheckTx(mk(t, &contract.MessageSend{FromAddress: a, ToAddress: b, Amount: 1_000}, rangeTestFee)); r.Error != nil {
				t.Fatalf("normal tx rejected: %v", r.Error)
			}
		})
	}
}

func TestSafetyChecksActiveFromGenesisOffMainnet(t *testing.T) {
	c, _ := newTestCanoliq()
	for _, p := range []string{ProfileLocalnet, ProfileDevnet, ProfileTestnet} {
		c.Config.Profile = p
		if !c.safetyChecksActive(0) {
			t.Fatalf("profile %s: checks should be active from genesis", p)
		}
	}
	c.Config.Profile = ProfileMainnet
	if c.safetyChecksActive(MainnetSafetyChecksHeight-1) || !c.safetyChecksActive(MainnetSafetyChecksHeight) {
		t.Fatal("mainnet activation boundary wrong")
	}
}
