package canoliq

import (
	"bytes"
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
		// No fresh reward delta this block. Still advance the peak-TVL high
		// water mark (T4) — also seeds it from the current pool on a pre-T4
		// node whose peak_tvl_ucnpy is still zero.
		if globals.TotalPooledCnpy > globals.PeakTvlUcnpy {
			globals.PeakTvlUcnpy = globals.TotalPooledCnpy
		}
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
	// Advance the peak-TVL high water mark (T4) post-accrual.
	if globals.TotalPooledCnpy > globals.PeakTvlUcnpy {
		globals.PeakTvlUcnpy = globals.TotalPooledCnpy
	}

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
	// Treasury & buyback go into plugin-owned scalar keys. WP §11 prescribes
	// 1–2% of the treasury feed into an insurance pool: skim insurance_bps
	// off the treasury slice so every credit auto-routes a fraction.
	if split.Treasury > 0 {
		insurance := uint64(0)
		if params.InsuranceBps > 0 {
			insurance = mulDiv(split.Treasury, params.InsuranceBps, 10_000)
		}
		// T4: once the reserve reaches its target (insurance_target_bps of peak
		// TVL), stop skimming — the would-be insurance amount stays in the
		// treasury so the fee-conservation invariant still holds. A target of 0
		// disables the gate (skim always on).
		if params.InsuranceTargetBps > 0 {
			target := mulDiv(globals.PeakTvlUcnpy, params.InsuranceTargetBps, 10_000)
			if c.readScalar(KeyForInsurancePool()) >= target {
				insurance = 0
			}
		}
		treasuryDelta := split.Treasury - insurance
		if treasuryDelta > 0 {
			treasuryKey := KeyForTreasuryCNPY()
			sets = append(sets, &contract.PluginSetOp{
				Key:   treasuryKey,
				Value: EncodeUint64(c.readScalar(treasuryKey) + treasuryDelta),
			})
		}
		if insurance > 0 {
			insuranceKey := KeyForInsurancePool()
			sets = append(sets, &contract.PluginSetOp{
				Key:   insuranceKey,
				Value: EncodeUint64(c.readScalar(insuranceKey) + insurance),
			})
		}
	}
	if split.Buyback > 0 {
		buybackKey := KeyForBuybackPool()
		sets = append(sets, &contract.PluginSetOp{
			Key:   buybackKey,
			Value: EncodeUint64(c.readScalar(buybackKey) + split.Buyback),
		})
	}
	if split.Validators > 0 {
		valSets, err := c.distributeValidatorShare(split.Validators)
		if err != nil {
			return err
		}
		sets = append(sets, valSets...)
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
// validator-incentive accruals when the canoLiq committee validator set is
// unknown (empty registry). Phase 2 prefers ValidatorRegistry-driven
// pro-rata in distributeValidatorShare; this address is the legacy fallback.
func (c *Canoliq) committeeAggregatorAddr() []byte {
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = 0xCA
	}
	return addr
}

// distributeValidatorShare splits the validator-incentive slice across the
// canoLiq committee validator set proportional to per-validator stake. Empty
// registry falls back to a single committee-wide aggregator key — the
// Phase 1 behavior — so Phase 1 tests continue to pass unchanged. Rounding
// remainder is credited to the largest-stake validator so the credited
// total exactly equals the input share.
func (c *Canoliq) distributeValidatorShare(share uint64) ([]*contract.PluginSetOp, *contract.PluginError) {
	if share == 0 {
		return nil, nil
	}
	registry, err := c.loadValidatorRegistry()
	if err != nil {
		return nil, err
	}
	if registry == nil || len(registry.Entries) == 0 {
		// Legacy aggregator path.
		key := KeyForValidatorIncentives(c.committeeAggregatorAddr())
		return []*contract.PluginSetOp{
			{Key: key, Value: EncodeUint64(c.readScalar(key) + share)},
		}, nil
	}
	totalStake := uint64(0)
	largestIdx := 0
	for i, e := range registry.Entries {
		totalStake += e.Stake
		if e.Stake > registry.Entries[largestIdx].Stake {
			largestIdx = i
		}
	}
	if totalStake == 0 {
		key := KeyForValidatorIncentives(c.committeeAggregatorAddr())
		return []*contract.PluginSetOp{
			{Key: key, Value: EncodeUint64(c.readScalar(key) + share)},
		}, nil
	}
	credits := make([]uint64, len(registry.Entries))
	allocated := uint64(0)
	for i, e := range registry.Entries {
		credits[i] = mulDiv(share, e.Stake, totalStake)
		allocated += credits[i]
	}
	if allocated < share {
		credits[largestIdx] += share - allocated
	}
	sets := make([]*contract.PluginSetOp, 0, len(registry.Entries))
	for i, e := range registry.Entries {
		if credits[i] == 0 {
			continue
		}
		key := KeyForValidatorIncentives(e.Address)
		sets = append(sets, &contract.PluginSetOp{
			Key:   key,
			Value: EncodeUint64(c.readScalar(key) + credits[i]),
		})
	}
	return sets, nil
}

// ejectValidator removes addr from the committee registry and clears its
// accrued validator-incentive balance (F12). Idempotent: a no-op when addr is
// absent so a passed eject proposal can never halt BeginBlock. Future reward
// sweeps redistribute pro-rata over the remaining registry entries, so the
// ejected validator simply stops receiving a share.
func (c *Canoliq) ejectValidator(addr []byte) *contract.PluginError {
	registry, err := c.loadValidatorRegistry()
	if err != nil {
		return err
	}
	// Always clear any accrued incentives for the ejected address.
	deletes := []*contract.PluginDeleteOp{{Key: KeyForValidatorIncentives(addr)}}
	var sets []*contract.PluginSetOp
	if registry != nil {
		kept := registry.Entries[:0]
		for _, e := range registry.Entries {
			if bytes.Equal(e.Address, addr) {
				continue
			}
			kept = append(kept, e)
		}
		registry.Entries = kept
		bz, e := contract.Marshal(registry)
		if e != nil {
			return e
		}
		sets = append(sets, &contract.PluginSetOp{Key: KeyForValidatorRegistry(), Value: bz})
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); err != nil {
		return err
	}
	return nil
}

// loadValidatorRegistry reads the singleton validator registry. Returns
// (nil, nil) when absent.
func (c *Canoliq) loadValidatorRegistry() (*contract.ValidatorRegistry, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForValidatorRegistry()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	reg := new(contract.ValidatorRegistry)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, reg); e != nil {
		return nil, e
	}
	return reg, nil
}
