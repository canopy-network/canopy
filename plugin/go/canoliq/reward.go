package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// ProcessRewards is the EndBlock hook that observes the canoLiq committee
// reward pool, isolates this block's reward delta against the last sweep,
// and applies the 12% protocol fee with the canonical 40/30/15/15 split.
//
// The committee pool is the same Canopy pool key used for transaction fees
// (KeyForFeePool(chainId)), since Canopy mints subsidies into that pool.
// To avoid double-counting fee revenue from canoLiq's own fee_pool
// accumulation, the function compares the pool against
// last_processed_reward_pool stored on globals and only sweeps the delta.
func (c *Canoliq) ProcessRewards(req *contract.PluginEndRequest) *contract.PluginError {
	params, err := c.LoadParams()
	if err != nil {
		return err
	}
	if params.FeeBps == 0 {
		return nil
	}
	globalsKey := KeyForGlobals()
	poolKey := contract.KeyForFeePool(c.Config.ChainId)
	gQ, pQ := rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: gQ, Key: globalsKey},
			{QueryId: pQ, Key: poolKey},
		},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	globals := new(contract.CanoliqGlobals)
	pool := new(contract.Pool)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return e
			}
		case pQ:
			if e := contract.Unmarshal(r.Entries[0].Value, pool); e != nil {
				return e
			}
		}
	}
	if !globals.GenesisComplete {
		// Genesis has not run yet; nothing to do.
		return nil
	}
	if pool.Amount <= globals.LastProcessedRewardPool {
		// No fresh reward delta this block.
		globals.LastProcessedRewardPool = pool.Amount
		return c.SaveGlobals(globals)
	}
	delta := pool.Amount - globals.LastProcessedRewardPool
	fee := FeeOnReward(delta, params.FeeBps)
	netToUsers := delta - fee
	split := SplitFee(fee, &FeeSplitParams{
		UserRebateBps: params.UserRebateBps,
		TreasuryBps:   params.TreasuryBps,
		ValidatorBps:  params.ValidatorBps,
		BuybackBps:    params.BuybackBps,
	})
	// User accrual: net rewards plus the user-rebate slice flow into the
	// pooled CNPY backing cCNPY, lifting the cCNPY/CNPY exchange rate.
	globals.TotalPooledCnpy += netToUsers + split.UserRebate

	// Drain the swept reward from the committee pool. The remainder
	// (validator + treasury + buyback shares) lives in plugin-owned keys.
	if pool.Amount >= delta {
		pool.Amount -= delta
	} else {
		pool.Amount = 0
	}
	// Re-credit the user-rebate + net portion back into the pool so cCNPY
	// holders can redeem against it. Validator/treasury/buyback portions are
	// removed from the committee pool by the subtraction above.
	pool.Amount += netToUsers + split.UserRebate
	// Record the post-sweep pool balance so the next block isolates only
	// fresh subsidy/fee inflows as delta.
	globals.LastProcessedRewardPool = pool.Amount

	poolBz, e := contract.Marshal(pool)
	if e != nil {
		return e
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return e
	}
	sets := []*contract.PluginSetOp{
		{Key: globalsKey, Value: gBz},
		{Key: poolKey, Value: poolBz},
	}
	// Treasury & buyback go into plugin-owned scalar keys.
	if split.Treasury > 0 {
		treasuryKey := KeyForTreasuryCNPY()
		sets = append(sets, &contract.PluginSetOp{
			Key:   treasuryKey,
			Value: EncodeUint64(c.readScalar(treasuryKey) + split.Treasury),
		})
	}
	if split.Buyback > 0 {
		buybackKey := KeyForBuybackPool()
		sets = append(sets, &contract.PluginSetOp{
			Key:   buybackKey,
			Value: EncodeUint64(c.readScalar(buybackKey) + split.Buyback),
		})
	}
	if split.Validators > 0 {
		// MVP: park validator share at a single committee-wide aggregator key.
		// A future iteration will distribute pro-rata across the committee
		// validator set once that lookup is available.
		valKey := KeyForValidatorIncentives(c.committeeAggregatorAddr())
		sets = append(sets, &contract.PluginSetOp{
			Key:   valKey,
			Value: EncodeUint64(c.readScalar(valKey) + split.Validators),
		})
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets}); err != nil {
		return err
	}
	_ = req
	return nil
}

// readScalar is a small convenience for reading a uint64 stored under `key`,
// returning 0 if absent. Used by ProcessRewards to read-modify-write
// treasury/buyback/validator accumulators in one batch.
func (c *Canoliq) readScalar(key []byte) uint64 {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: key}},
	})
	if err != nil || resp == nil || resp.Error != nil {
		return 0
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return 0
	}
	return DecodeUint64(resp.Results[0].Entries[0].Value)
}

// committeeAggregatorAddr returns a synthetic "address" used to aggregate
// validator-incentive accruals at the committee level. Since the canoLiq
// MVP does not yet introspect the validator set from the FSM, this single
// address holds the accrued share until Phase 2 wires per-validator splits.
func (c *Canoliq) committeeAggregatorAddr() []byte {
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = 0xCA
	}
	return addr
}
