package canoliq

import (
	"bytes"

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
	escrowKey := KeyForEscrowPool()
	gQ, pQ, eQ := qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: gQ, Key: globalsKey},
			{QueryId: pQ, Key: poolKey},
			{QueryId: eQ, Key: escrowKey},
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
	escrow := new(contract.Pool)
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
		case eQ:
			if e := contract.Unmarshal(r.Entries[0].Value, escrow); e != nil {
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
	// L3: the committee pool grows from two sources — Canopy committee rewards
	// and canoLiq's own protocol tx fees (every handler credits its fee here).
	// Only the reward portion is subject to the 12% fee + 40/30/15/15 split; the
	// accrued tx-fee portion is protocol revenue that routes straight to the DAO
	// treasury (see the treasury credit below). Clamp guards a never-expected
	// accrual > pool-growth case.
	txFees := c.readScalar(KeyForTxFeeAccrual())
	if txFees > delta {
		txFees = delta
	}
	rewardDelta := delta - txFees
	fee := FeeOnReward(rewardDelta, params.FeeBps)
	netToUsers := rewardDelta - fee
	split := SplitFee(fee, &FeeSplitParams{
		UserRebateBps: params.UserRebateBps,
		TreasuryBps:   params.TreasuryBps,
		ValidatorBps:  params.ValidatorBps,
		BuybackBps:    params.BuybackBps,
	})
	// User accrual: net rewards plus the user-rebate slice flow into the
	// pooled CNPY backing cCNPY, lifting the cCNPY/CNPY exchange rate.
	userSlice := netToUsers + split.UserRebate
	globals.TotalPooledCnpy += userSlice
	// Advance the peak-TVL high water mark (T4) post-accrual.
	if globals.TotalPooledCnpy > globals.PeakTvlUcnpy {
		globals.PeakTvlUcnpy = globals.TotalPooledCnpy
	}

	// Drain the whole swept reward from the committee pool. Every slice now
	// lives in a plugin-owned key: the user slice moves into the escrow pool
	// (H1 — it backs cCNPY redemptions, kept distinct from this fee pool so it
	// is not re-swept as reward next block); validator/treasury/buyback go to
	// their own keys below.
	if pool.Amount >= delta {
		pool.Amount -= delta
	} else {
		pool.Amount = 0
	}
	// H1: credit the user slice into the escrow pool so cCNPY holders can
	// redeem against real CNPY. Keeps escrow == TotalPooledCnpy + PendingRedemptionCnpy.
	escrow.Amount += userSlice
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
	escrowBz, e := contract.Marshal(escrow)
	if e != nil {
		return e
	}
	sets := []*contract.PluginSetOp{
		{Key: globalsKey, Value: gBz},
		{Key: poolKey, Value: poolBz},
		{Key: escrowKey, Value: escrowBz},
	}
	// Treasury & buyback go into plugin-owned scalar keys. WP §9.2 (slashing
	// risk) prescribes seeding an insurance pool from treasury inflow: skim
	// insurance_bps off the treasury reward slice so every credit auto-routes a
	// fraction. The L3 tx-fee revenue is added to the treasury credit here too
	// but is NOT subject to the insurance skim (it is not committee reward).
	insurance := uint64(0)
	if split.Treasury > 0 && params.InsuranceBps > 0 {
		insurance = mulDiv(split.Treasury, params.InsuranceBps, 10_000)
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
	}
	treasuryDelta := (split.Treasury - insurance) + txFees
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
	// L3: the accrued tx-fees have now been routed to the treasury; zero the
	// accumulator so the next sweep starts fresh.
	if txFees > 0 {
		sets = append(sets, &contract.PluginSetOp{Key: KeyForTxFeeAccrual(), Value: EncodeUint64(0)})
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
	q := qid()
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
	q := qid()
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
