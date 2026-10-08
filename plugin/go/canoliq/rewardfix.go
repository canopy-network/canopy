package canoliq

import (
	"bytes"
	"log"

	"github.com/canopy-network/go-plugin/contract"
)

// rewardfix.go pins the reward-ownership fix to a block height on mainnet and
// carries the one-time state correction that runs at that height.
//
// Why a height at all
// -------------------
// Mainnet committee 29 ran the pre-fix reward sweep from its activation block
// (4679) onwards: every block credited the whole committee's stake growth —
// Canopy Foundation's own validator and delegate bonds — to cCNPY holders.
// Those blocks are final. Any node that syncs committee 29 from genesis
// re-executes them with the plugin binary it runs *now*, so the fix must leave
// every earlier block byte-identical and switch on at one height that every
// node agrees on. TestMainnetPreFixReplayParity holds the first half of that
// against hashes captured from plugin-go-v2026.266.23.
//
// Other profiles have no such history: localnet, devnet and testnet chains
// either start fresh on this binary or are reset, so the fix is active from
// genesis there.
//
// What changes at MainnetRewardFixHeight
// --------------------------------------
//   - reward attribution follows ownership (registry.go), with the per-block
//     plausibility clamp and carried remainder (reward.go);
//   - an "awaiting-canopy-stake" TVL cap rejects deposits (deliver.go);
//   - the reward-attribution / owned-stake alerts start evaluating (alerts.go);
//   - applyMainnetRewardCorrection rewrites the balances the pre-fix sweep
//     inflated (below).

// MainnetRewardFixHeight is the committee-29 block at which the reward fix
// activates. It must be above the height at which every committee-29 node
// has upgraded to this release; pick it with margin for pluginAutoUpdate.
const MainnetRewardFixHeight uint64 = 13_950

// rewardFixActive reports whether the reward-ownership fix governs `height`.
func (c *Canoliq) rewardFixActive(height uint64) bool {
	if c.Config.Profile != ProfileMainnet {
		return true
	}
	return height >= MainnetRewardFixHeight
}

// The correction below is written against the exact committee-29 state the
// pre-fix sweep left behind, read from the chain while it was paused for
// maintenance. Only two addresses ever held cCNPY, and every uCNPY of real
// principal is accounted for:
//
//	111e0583…5e61  deposited  10.000000 CNLQ, holds 8 cCNPY, redemption #0 of 2 cCNPY
//	9ddb1fe7…85e3  deposited 508.500661 CNLQ, holds 0.719475 cCNPY
//
// The correction restores a 1:1 exchange rate on that principal: each
// depositor's claim is exactly what they put in, the queued redemption pays
// back the 2 CNLQ its 2 cCNPY stood for, and the escrow holds 518.500661 CNLQ
// — the sum of both deposits, which is what core debited from their accounts.
// Every balance the sweep funded out of foreign bond growth (treasury, buyback,
// insurance, validator incentives, the tx-fee accrual) goes to zero; none of
// it was ever paid out (no treasury spend, buyback, or claim has executed).
var (
	fixHolderA = mustHex20("111e0583347c901fa3ccf3bc1de4840349565e61")
	fixHolderB = mustHex20("9ddb1fe7e9539e08e035118f15f50b2c089485e3")
)

const (
	// Pre-correction state the correction expects to find. Anything else means
	// the chain moved in a way this release did not anticipate, and the
	// correction declines to run rather than overwrite it.
	fixExpectCcnpySupply   = 8_719_475
	fixExpectPending       = 1_942_025_331
	fixExpectNextRedeemID  = 1
	fixExpectHolderACcnpy  = 8_000_000
	fixExpectHolderBCcnpy  = 719_475
	fixExpectRedemptionAmt = 1_942_025_331

	// Corrected state.
	fixHolderACcnpy  = 8_000_000   // 8 of the 10 CNLQ deposited, 2 are queued
	fixHolderBCcnpy  = 508_500_661 // the full 508.500661 CNLQ deposited
	fixRedemptionAmt = 2_000_000   // 2 cCNPY at 1:1
	fixCcnpySupply   = fixHolderACcnpy + fixHolderBCcnpy
	fixPooled        = fixCcnpySupply
	fixPending       = fixRedemptionAmt
	fixEscrow        = fixPooled + fixPending // 518.500661 CNLQ of real principal
	fixRedemptionID  = 0
)

// applyMainnetRewardCorrection runs once, in the BeginBlock of
// MainnetRewardFixHeight on a mainnet profile. It is deterministic: it reads
// only consensus state and writes fixed values, so every node that reaches the
// height produces the same result.
//
// It never returns an error for a state mismatch. Halting committee 29 over a
// bookkeeping correction would be worse than the inflation it corrects, and
// the fix itself (R = 0 for unowned stake) is already live at this height
// either way; a mismatch is logged loudly for a follow-up release instead.
func (c *Canoliq) applyMainnetRewardCorrection(height uint64) *contract.PluginError {
	if c.Config.Profile != ProfileMainnet || height != MainnetRewardFixHeight {
		return nil
	}
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if !g.GenesisComplete {
		return nil
	}
	redKey := KeyForRedemption(fixHolderA, fixRedemptionID)
	balAKey, balBKey := KeyForCCNPYBalance(fixHolderA), KeyForCCNPYBalance(fixHolderB)
	qA, qB, qR, qReg, qE := qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: qA, Key: balAKey},
			{QueryId: qB, Key: balBKey},
			{QueryId: qR, Key: redKey},
			{QueryId: qReg, Key: KeyForValidatorRegistry()},
			{QueryId: qE, Key: KeyForEscrowPool()},
		},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	var balA, balB uint64
	var red *contract.Redemption
	reg := new(contract.ValidatorRegistry)
	escrow := new(contract.Pool)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		v := r.Entries[0].Value
		switch r.QueryId {
		case qA:
			balA = DecodeUint64(v)
		case qB:
			balB = DecodeUint64(v)
		case qR:
			red = new(contract.Redemption)
			if e := contract.Unmarshal(v, red); e != nil {
				return e
			}
		case qReg:
			if e := contract.Unmarshal(v, reg); e != nil {
				return e
			}
		case qE:
			if e := contract.Unmarshal(v, escrow); e != nil {
				return e
			}
		}
	}
	if g.TotalCcnpySupply != fixExpectCcnpySupply ||
		g.PendingRedemptionCnpy != fixExpectPending ||
		g.NextRedemptionId != fixExpectNextRedeemID ||
		balA != fixExpectHolderACcnpy || balB != fixExpectHolderBCcnpy ||
		red == nil || red.CnpyAmount != fixExpectRedemptionAmt || !bytes.Equal(red.Address, fixHolderA) {
		log.Printf("canoliq: ERROR reward correction at height %d skipped: pre-state differs from what this "+
			"release expects (supply=%d pending=%d nextRedemption=%d holderA=%d holderB=%d redemption=%v). "+
			"The reward fix is active; balances were left untouched for a follow-up release.",
			height, g.TotalCcnpySupply, g.PendingRedemptionCnpy, g.NextRedemptionId, balA, balB, red)
		return nil
	}

	g.TotalCcnpySupply = fixCcnpySupply
	g.TotalPooledCnpy = fixPooled
	g.PendingRedemptionCnpy = fixPending
	g.PeakTvlUcnpy = fixPooled
	// The watermark now means aggregate *owned* stake, and canoLiq owns none.
	g.LastProcessedRewardPool = 0
	g.LastAttributedReward = 0
	g.CarriedReward = 0
	red.CnpyAmount = fixRedemptionAmt
	escrow.Amount = fixEscrow

	gBz, e := contract.Marshal(g)
	if e != nil {
		return e
	}
	redBz, e := contract.Marshal(red)
	if e != nil {
		return e
	}
	escrowBz, e := contract.Marshal(escrow)
	if e != nil {
		return e
	}
	sets := []*contract.PluginSetOp{
		{Key: KeyForGlobals(), Value: gBz},
		{Key: KeyForEscrowPool(), Value: escrowBz},
		{Key: balAKey, Value: EncodeUint64(fixHolderACcnpy)},
		{Key: balBKey, Value: EncodeUint64(fixHolderBCcnpy)},
		{Key: redKey, Value: redBz},
	}
	deletes := []*contract.PluginDeleteOp{
		{Key: KeyForTreasuryCNPY()},
		{Key: KeyForBuybackPool()},
		{Key: KeyForInsurancePool()},
		{Key: KeyForTxFeeAccrual()},
		{Key: KeyForValidatorIncentives(c.committeeAggregatorAddr())},
	}
	for _, entry := range reg.Entries {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: KeyForValidatorIncentives(entry.Address)})
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); err != nil {
		return err
	}
	log.Printf("canoliq: reward correction applied at height %d: pooled=%d supply=%d pending=%d escrow=%d",
		height, fixPooled, fixCcnpySupply, fixPending, fixEscrow)
	return nil
}

func mustHex20(h string) []byte {
	b, err := decodeGenesisAddress(h)
	if err != nil || len(b) != 20 {
		panic("canoliq: bad fixed address " + h)
	}
	return b
}
