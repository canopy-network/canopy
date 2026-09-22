package canoliq

import (
	"bytes"

	"github.com/canopy-network/go-plugin/contract"
)

// ProcessRewards is the EndBlock hook that observes canoLiq's received
// committee reward and applies the 12% protocol fee with the canonical
// 40/30/15/15 split.
//
// Observation source — canoLiq's committee stake, NOT the committee fee pool.
// Canopy funds the committee reward pool (KeyForFeePool(chainId)) in BeginBlock
// and then fully distributes + zeroes it in EndBlock's DistributeCommitteeRewards,
// which runs *before* this plugin EndBlock hook. So the pool is always 0 by the
// time we look — it cannot be swept. Instead, Canopy compounds each block's
// committee reward into the bonded StakedAmount of the committee's validators
// (DistributeCommitteeReward, Compound=true). We therefore observe the
// block-over-block growth of canoLiq's committee validator stake as the
// received reward R, and store the last observed aggregate in
// globals.last_processed_reward_pool (repurposed to mean "last observed
// committee stake"). canoLiq's own protocol tx-fees are tracked separately in
// KeyForTxFeeAccrual and route straight to the DAO treasury.
//
// The observed member set is reconciled against Canopy's live committee
// membership on every call (registry.go::syncCommitteeRegistry) so validators
// that join or leave committee `chainId` post-genesis are accounted for. The
// reward is the *lesser* of two independent estimates of that growth:
//
//   - per-validator: Σ growth of members already registered last block, which
//     ignores the bond a newly admitted member brings with it, and
//   - aggregate: the growth of the whole member set against the watermark,
//     which ignores a stale per-validator baseline (e.g. the first sync after
//     an upgrade, where the seeded genesis weights are not real stakes).
//
// Each estimate is exact except in the case the other one covers, so the min
// is right whenever either is, and conservative (under-credits for one block)
// when membership churns in both directions at once.
func (c *Canoliq) ProcessRewards(req *contract.PluginEndRequest) *contract.PluginError {
	params, err := c.LoadParams()
	if err != nil {
		return err
	}
	if params.FeeBps == 0 {
		return nil
	}
	globalsKey := KeyForGlobals()
	escrowKey := KeyForEscrowPool()
	gQ, eQ := qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: gQ, Key: globalsKey},
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
	// Reconcile the committee member set with Canopy's live validator records
	// and observe the growth of their bonded stake.
	obs, err := c.syncCommitteeRegistry()
	if err != nil {
		return err
	}
	registryOp, e := registrySetOp(obs.registry)
	if e != nil {
		return e
	}
	observedStake := obs.total
	baseline := globals.LastProcessedRewardPool
	// rewardDelta is pure Canopy committee reward — see the min() rationale in
	// the doc comment above.
	rewardDelta := obs.reward
	if observedStake <= baseline {
		// An unstake / slash / departure shrank the aggregate position: take no
		// reward this block and let the watermark reset to the new level.
		rewardDelta = 0
	} else if aggregate := observedStake - baseline; aggregate < rewardDelta {
		rewardDelta = aggregate
	}
	// Seed-and-return when there is no growth to distribute. baseline == 0 is
	// the first observation ever (fresh node / post-upgrade): adopt the current
	// stake as the watermark so pre-existing bonded stake is not mistaken for
	// one giant reward. Either way the peak-TVL high-water mark (T4) still
	// advances and the reconciled registry is still persisted.
	if baseline == 0 || rewardDelta == 0 {
		if globals.TotalPooledCnpy > globals.PeakTvlUcnpy {
			globals.PeakTvlUcnpy = globals.TotalPooledCnpy
		}
		globals.LastProcessedRewardPool = observedStake
		gBz, err := contract.Marshal(globals)
		if err != nil {
			return err
		}
		if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
			Sets: []*contract.PluginSetOp{{Key: globalsKey, Value: gBz}, registryOp},
		}); err != nil {
			return err
		}
		return nil
	}
	// L3: canoLiq's own protocol tx-fees accrue in their own scalar (every
	// handler credits it) and route straight to the DAO treasury. They are
	// protocol revenue, not committee reward, so they are NOT part of the 12%
	// fee + 40/30/15/15 split applied to rewardDelta.
	txFees := c.readScalar(KeyForTxFeeAccrual())
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
	// ...unless there is no cCNPY in existence to lift. TotalPooledCnpy is the
	// denominator of the exchange rate, so crediting it while TotalCcnpySupply
	// is zero manufactures pooled CNPY that no share has a claim on and drives
	// computeMint's ratio toward zero:
	//
	//   mint = amount * (total_ccnpy + 1) / (total_pooled + 1)
	//
	// Once accrued reward alone outgrows a realistic deposit, every deposit
	// computes mint == 0 and is rejected, and nothing can lift TotalCcnpySupply
	// off zero to break the deadlock. That is the canoliq-98803 / chain-404
	// failure, and it needs no chain reset: a full redeem leaves
	// TotalCcnpySupply at zero with dust in the pool (computeRedeem floors in
	// the pool's favour by design), after which ordinary reward accrual walks
	// the rate off a cliff on its own.
	//
	// So route an ownerless slice to the DAO treasury instead. The CNPY stays
	// protocol-held and conserved, it simply accrues where it has a defined
	// owner. reconcileOrphanedPoolOnDevnet cleans up chains already carrying an
	// orphaned pool, but it is dev-profile-only by design; this is the
	// prevention half, and it runs everywhere because it never rewrites an
	// existing balance. The instant any real deposit mints cCNPY, this branch
	// stops firing and accrual returns to the pool untouched.
	ownerless := uint64(0)
	if globals.TotalCcnpySupply == 0 {
		ownerless = userSlice
		userSlice = 0
	}
	globals.TotalPooledCnpy += userSlice
	// Advance the peak-TVL high water mark (T4) post-accrual.
	if globals.TotalPooledCnpy > globals.PeakTvlUcnpy {
		globals.PeakTvlUcnpy = globals.TotalPooledCnpy
	}

	// H1: credit the user slice into the escrow pool so cCNPY holders can
	// redeem against real CNPY. Keeps escrow == TotalPooledCnpy + PendingRedemptionCnpy.
	// (The reward CNPY itself lives in the bonded committee stake; escrow becomes
	// a claim on it, redeemable once the position is unbonded.)
	//
	// An ownerless slice is deliberately NOT credited here: it is not backing
	// any cCNPY, and adding it would break that invariant. It lands in the
	// treasury scalar below instead, so physical CNPY is still conserved.
	escrow.Amount += userSlice
	// Record the observed committee stake so the next block isolates only the
	// fresh growth (reward) as delta.
	globals.LastProcessedRewardPool = observedStake

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
		{Key: escrowKey, Value: escrowBz},
		registryOp,
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
	// ownerless carries the user slice that had no cCNPY to back it (see above).
	treasuryDelta := (split.Treasury - insurance) + txFees + ownerless
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
		valSets, err := c.distributeValidatorShare(split.Validators, obs.registry)
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

// reconcileOrphanedPoolOnDevnet zeroes TotalPooledCnpy (and the escrow pool
// backing it) on isDevProfile profiles when TotalCcnpySupply is still zero —
// i.e. ProcessRewards has been compounding committee reward into the pool
// (see the "User accrual" comment above) for a chain that has never had a
// single real deposit mint any cCNPY against it.
//
// computeMint's exchange rate is amount*(totalCcnpy+1)/(totalPooled+1); with
// totalCcnpy==0 that reduces to amount/(totalPooled+1). Once accrued reward
// alone outgrows any realistic single deposit, every new deposit computes
// mint=0 and is rejected by CheckMessageCanoliqDeposit ("pool math error:
// mint computed to zero") — permanently, since nothing can grow
// TotalCcnpySupply off zero to break the deadlock. A long-lived devnet
// committee earns reward from having a staked validator alone, with no
// deposit activity required to get there, so this is reachable purely by
// leaving a devnet chain running (the same underlying pattern as the
// tvlCapBps devnet override next door).
//
// Safe to run every block: it only ever mutates state while
// TotalCcnpySupply is exactly zero, i.e. before any real depositor holds a
// claim on the pool. The instant a real deposit mints the first cCNPY, this
// becomes permanently inert (TotalCcnpySupply != 0 short-circuits above) —
// it can never touch a balance any user holds a claim against.
//
// Resets the escrow pool in lockstep to preserve the
// escrow == TotalPooledCnpy + PendingRedemptionCnpy invariant (state.go),
// and the TVL high-water mark so it does not keep pointing at an accrual
// nothing backs anymore.
func (c *Canoliq) reconcileOrphanedPoolOnDevnet() *contract.PluginError {
	if !isDevProfile(c.Config.Profile) {
		return nil
	}
	globals, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if globals.TotalCcnpySupply != 0 || globals.TotalPooledCnpy == 0 {
		return nil
	}
	escrowKey := KeyForEscrowPool()
	eq := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: eq, Key: escrowKey}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	escrow := new(contract.Pool)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, escrow); e != nil {
			return e
		}
	}
	// PendingRedemptionCnpy must already be 0 here — nothing could have been
	// redeemed with zero cCNPY ever minted — but compute from it rather than
	// assume, so the invariant holds even if that ever stops being true.
	escrow.Amount = globals.PendingRedemptionCnpy
	globals.TotalPooledCnpy = 0
	globals.PeakTvlUcnpy = 0
	eBz, e := contract.Marshal(escrow)
	if e != nil {
		return e
	}
	if err := c.SaveGlobals(globals); err != nil {
		return err
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: escrowKey, Value: eBz}},
	}); err != nil {
		return err
	}
	return nil
}

// validatorOnCommittee reports whether val is a member of the given committee.
func validatorOnCommittee(val *contract.Validator, chainId uint64) bool {
	for _, id := range val.Committees {
		if id == chainId {
			return true
		}
	}
	return false
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
//
// `registry` is the set reconciled by this block's syncCommitteeRegistry, not
// a re-read of state: the reconciled copy is only written at the end of
// ProcessRewards, so a re-read here would pay out against last block's
// membership.
func (c *Canoliq) distributeValidatorShare(share uint64, registry *contract.ValidatorRegistry) ([]*contract.PluginSetOp, *contract.PluginError) {
	if share == 0 {
		return nil, nil
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
//
// The removal is also recorded as a tombstone (KeyForEjectedValidator) because
// the registry is reconciled against Canopy's live committee membership every
// block — the ejected operator is still a Canopy validator bonded to this
// committee, so without the tombstone the next sync would re-admit it and the
// ejection would last exactly one block. Its stake keeps earning Canopy
// committee reward; canoLiq simply stops observing that stake and stops
// crediting it.
func (c *Canoliq) ejectValidator(addr []byte) *contract.PluginError {
	registry, err := c.loadValidatorRegistry()
	if err != nil {
		return err
	}
	// Always clear any accrued incentives for the ejected address.
	deletes := []*contract.PluginDeleteOp{{Key: KeyForValidatorIncentives(addr)}}
	sets := []*contract.PluginSetOp{{
		Key:   KeyForEjectedValidator(addr),
		Value: EncodeUint64(c.plugin.CurrentHeight()),
	}}
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
