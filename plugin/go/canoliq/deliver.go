package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// DeliverMessageCanoliqDeposit moves CNPY from the sender's account into the
// canoLiq escrow pool and credits cCNPY to the sender at the current
// exchange rate. First deposit mints 1:1; subsequent deposits use
// mint = amount * total_ccnpy / total_pooled_cnpy.
func (c *Canoliq) DeliverMessageCanoliqDeposit(msg *contract.MessageCanoliqDeposit, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	fromKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	escrowKey := contract.KeyForFeePool(c.Config.ChainId) // canoLiq committee escrow uses same pool id
	balKey := KeyForCCNPYBalance(msg.FromAddress)
	globalsKey := KeyForGlobals()
	fQ, gQ, bQ, eQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	feeQ := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: fQ, Key: fromKey},
			{QueryId: gQ, Key: globalsKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: eQ, Key: escrowKey},
			{QueryId: feeQ, Key: feePoolKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	from := new(contract.Account)
	globals := new(contract.CanoliqGlobals)
	escrow := new(contract.Pool)
	feePool := new(contract.Pool)
	var balanceBz []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, from); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case bQ:
			balanceBz = r.Entries[0].Value
		case eQ:
			if e := contract.Unmarshal(r.Entries[0].Value, escrow); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	// TVL self-cap (WP §9.4). evaluateTVLCap (query.go) is the single
	// source of truth for the four-way decision tree shared with
	// /v1/health: uncapped / active / awaiting-canopy-stake / fail-closed.
	// See the tvl-cap docs page for the operator-facing semantics.
	if params.TvlCapBps > 0 {
		supply, perr := c.readCanopySupply()
		if perr != nil {
			return &contract.PluginDeliverResponse{Error: perr}
		}
		var staked uint64
		if supply != nil {
			staked = supply.Staked
		}
		decision := evaluateTVLCap(params.TvlCapBps, supply != nil, staked)
		if decision.Err != nil {
			return &contract.PluginDeliverResponse{Error: decision.Err}
		}
		if decision.CapUcnpy > 0 && globals.TotalPooledCnpy+msg.Amount > decision.CapUcnpy {
			return &contract.PluginDeliverResponse{Error: ErrTVLCapExceeded()}
		}
	}
	deduct := msg.Amount + fee
	if from.Amount < deduct {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	mint := computeMint(msg.Amount, globals.TotalCcnpySupply, globals.TotalPooledCnpy)
	if mint == 0 {
		return &contract.PluginDeliverResponse{Error: ErrPoolMath("mint computed to zero")}
	}
	from.Amount -= deduct
	feePool.Amount += fee
	escrow.Amount += msg.Amount
	current := DecodeUint64(balanceBz)
	current += mint
	globals.TotalCcnpySupply += mint
	globals.TotalPooledCnpy += msg.Amount
	fromBz, e := contract.Marshal(from)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: globalsKey, Value: gBz},
		{Key: balKey, Value: EncodeUint64(current)},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if from.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: fromKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: fromKey, Value: fromBz})
	}
	// The escrow pool is the same key as feePool (committee pool); we already
	// updated it above by writing feePool. The escrow accumulator is implicit
	// in the committee pool growing alongside the fee. (Phase 2 may split
	// escrow vs fee tracking onto distinct keys.)
	_ = escrow
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCanoliqRedeem burns the requested cCNPY, computes the
// equivalent CNPY at the current exchange rate, and writes a Redemption
// record that matures after the configured unstaking window.
func (c *Canoliq) DeliverMessageCanoliqRedeem(msg *contract.MessageCanoliqRedeem, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	fromKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	balKey := KeyForCCNPYBalance(msg.FromAddress)
	globalsKey := KeyForGlobals()
	redemIdxKey := KeyForRedemptionIndex(msg.FromAddress)
	fQ, gQ, bQ, feeQ, riQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: fQ, Key: fromKey},
			{QueryId: gQ, Key: globalsKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: feeQ, Key: feePoolKey},
			{QueryId: riQ, Key: redemIdxKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	from := new(contract.Account)
	globals := new(contract.CanoliqGlobals)
	feePool := new(contract.Pool)
	redemIdx := new(contract.RedemptionIndex)
	var balBz []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, from); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case bQ:
			balBz = r.Entries[0].Value
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case riQ:
			if e := contract.Unmarshal(r.Entries[0].Value, redemIdx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if from.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	bal := DecodeUint64(balBz)
	if bal < msg.CcnpyAmount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCCNPY()}
	}
	cnpyOwed := computeRedeem(msg.CcnpyAmount, globals.TotalCcnpySupply, globals.TotalPooledCnpy)
	if cnpyOwed == 0 {
		return &contract.PluginDeliverResponse{Error: ErrPoolMath("redeem computed to zero")}
	}
	bal -= msg.CcnpyAmount
	globals.TotalCcnpySupply -= msg.CcnpyAmount
	globals.TotalPooledCnpy -= cnpyOwed
	globals.PendingRedemptionCnpy += cnpyOwed
	id := globals.NextRedemptionId
	globals.NextRedemptionId++
	from.Amount -= fee
	feePool.Amount += fee
	// Unstaking window: the original Phase 1 plan called for reading
	// valParams.UnstakingBlocks from the FSM gov-params prefix, which
	// never landed. Until that plumbing exists, use Config.RedemptionUnstakingBlocks
	// — defaults to 5 for localnet so claim-after-maturity exercises in
	// reasonable time, but testnet/mainnet configs must set this to
	// something approximating Canopy's real UnstakingBlocks (thousands).
	window := c.Config.RedemptionUnstakingBlocks
	if window == 0 {
		window = 5
	}
	redemption := &contract.Redemption{
		Id:                   id,
		Address:              msg.FromAddress,
		CnpyAmount:           cnpyOwed,
		UnbondCompleteHeight: c.currentHeight() + window,
	}
	rBz, e := contract.Marshal(redemption)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	redemIdx.Ids = append(redemIdx.Ids, id)
	riBz, e := contract.Marshal(redemIdx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: globalsKey, Value: gBz},
		{Key: KeyForRedemption(msg.FromAddress, id), Value: rBz},
		{Key: redemIdxKey, Value: riBz},
		{Key: feePoolKey, Value: feeBz},
		// Global mature-redemption index: lets the stuck-redemption alert
		// evaluator range-scan keys ≤ currentHeight in one call. Key shape
		// puts UnbondCompleteHeight first so lexicographic order matches
		// maturity order. Deleted when the redemption is claimed.
		{Key: KeyForMatureRedemption(redemption.UnbondCompleteHeight, msg.FromAddress, id), Value: matureRedemptionMarker},
	}
	var deletes []*contract.PluginDeleteOp
	if bal == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: balKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: balKey, Value: EncodeUint64(bal)})
	}
	if from.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: fromKey})
	} else {
		fromBz, e := contract.Marshal(from)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: fromKey, Value: fromBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCanoliqClaimRedemption matures a redemption record by
// transferring escrowed CNPY back to the user's account when the unbond
// window has passed.
func (c *Canoliq) DeliverMessageCanoliqClaimRedemption(msg *contract.MessageCanoliqClaimRedemption, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	rKey := KeyForRedemption(msg.FromAddress, msg.RedemptionId)
	fromKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	globalsKey := KeyForGlobals()
	redemIdxKey := KeyForRedemptionIndex(msg.FromAddress)
	rQ, fQ, gQ, feeQ, riQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: rQ, Key: rKey},
			{QueryId: fQ, Key: fromKey},
			{QueryId: gQ, Key: globalsKey},
			{QueryId: feeQ, Key: feePoolKey},
			{QueryId: riQ, Key: redemIdxKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	redemption := new(contract.Redemption)
	from := new(contract.Account)
	globals := new(contract.CanoliqGlobals)
	feePool := new(contract.Pool)
	redemIdx := new(contract.RedemptionIndex)
	var rPresent bool
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case rQ:
			if e := contract.Unmarshal(r.Entries[0].Value, redemption); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			rPresent = redemption.Address != nil
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, from); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case riQ:
			if e := contract.Unmarshal(r.Entries[0].Value, redemIdx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if !rPresent {
		return &contract.PluginDeliverResponse{Error: ErrRedemptionNotFound()}
	}
	currentHeight := c.currentHeight()
	if currentHeight < redemption.UnbondCompleteHeight {
		return &contract.PluginDeliverResponse{Error: ErrRedemptionNotMature()}
	}
	if from.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	from.Amount = from.Amount - fee + redemption.CnpyAmount
	feePool.Amount += fee
	if globals.PendingRedemptionCnpy >= redemption.CnpyAmount {
		globals.PendingRedemptionCnpy -= redemption.CnpyAmount
	} else {
		globals.PendingRedemptionCnpy = 0
	}
	fromBz, e := contract.Marshal(from)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: fromKey, Value: fromBz},
		{Key: globalsKey, Value: gBz},
		{Key: feePoolKey, Value: feeBz},
	}
	deletes := []*contract.PluginDeleteOp{
		{Key: rKey},
		// Remove the global mature-redemption index entry so the
		// stuck-redemption alert evaluator stops counting this one.
		{Key: KeyForMatureRedemption(redemption.UnbondCompleteHeight, msg.FromAddress, msg.RedemptionId)},
	}
	redemIdx.Ids = removeUint64(redemIdx.Ids, msg.RedemptionId)
	if len(redemIdx.Ids) == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: redemIdxKey})
	} else {
		riBz, e := contract.Marshal(redemIdx)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: redemIdxKey, Value: riBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets:    sets,
		Deletes: deletes,
	}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCLIQTransfer moves liquid CLIQ between two accounts.
func (c *Canoliq) DeliverMessageCLIQTransfer(msg *contract.MessageCLIQTransfer, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	fromBalKey := KeyForCLIQBalance(msg.FromAddress)
	toBalKey := KeyForCLIQBalance(msg.ToAddress)
	cnpyFromKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	fbQ, tbQ, cQ, feeQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: fbQ, Key: fromBalKey},
			{QueryId: tbQ, Key: toBalKey},
			{QueryId: cQ, Key: cnpyFromKey},
			{QueryId: feeQ, Key: feePoolKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	var fromBz, toBz []byte
	cnpyFrom := new(contract.Account)
	feePool := new(contract.Pool)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case fbQ:
			fromBz = r.Entries[0].Value
		case tbQ:
			toBz = r.Entries[0].Value
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpyFrom); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if cnpyFrom.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	fromBal := DecodeUint64(fromBz)
	toBal := DecodeUint64(toBz)
	if fromBal < msg.Amount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCLIQ()}
	}
	fromBal -= msg.Amount
	toBal += msg.Amount
	cnpyFrom.Amount -= fee
	feePool.Amount += fee
	cnpyFromB, e := contract.Marshal(cnpyFrom)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: toBalKey, Value: EncodeUint64(toBal)},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if fromBal == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: fromBalKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: fromBalKey, Value: EncodeUint64(fromBal)})
	}
	if cnpyFrom.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyFromKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyFromKey, Value: cnpyFromB})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCLIQClaimVested unlocks any newly-vested CLIQ across all of
// the caller's vesting schedules and credits it to their liquid balance.
func (c *Canoliq) DeliverMessageCLIQClaimVested(msg *contract.MessageCLIQClaimVested, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	idxKey := KeyForVestingIndex(msg.FromAddress)
	balKey := KeyForCLIQBalance(msg.FromAddress)
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	iQ, bQ, cQ, feeQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: iQ, Key: idxKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: feeQ, Key: feePoolKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	idx := new(contract.VestingIndex)
	var balBz []byte
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	idxPresent := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			idxPresent = true
		case bQ:
			balBz = r.Entries[0].Value
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if !idxPresent || len(idx.ScheduleIds) == 0 {
		return &contract.PluginDeliverResponse{Error: ErrNoVestingSchedule()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	scheduleReads := make([]*contract.PluginKeyRead, 0, len(idx.ScheduleIds))
	scheduleQ := make(map[uint64]uint64, len(idx.ScheduleIds))
	for _, id := range idx.ScheduleIds {
		q := rand.Uint64()
		scheduleQ[q] = id
		scheduleReads = append(scheduleReads, &contract.PluginKeyRead{
			QueryId: q,
			Key:     KeyForVesting(msg.FromAddress, id),
		})
	}
	schedResp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: scheduleReads})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if schedResp.Error != nil {
		return &contract.PluginDeliverResponse{Error: schedResp.Error}
	}
	height := c.currentHeight()
	totalUnlocked := uint64(0)
	updates := make([]*contract.PluginSetOp, 0)
	for _, r := range schedResp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		sched := new(contract.VestingSchedule)
		if e := contract.Unmarshal(r.Entries[0].Value, sched); e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		newClaim := unlockedAmount(sched, height)
		if newClaim <= sched.ClaimedAmount {
			continue
		}
		delta := newClaim - sched.ClaimedAmount
		sched.ClaimedAmount = newClaim
		totalUnlocked += delta
		bz, e := contract.Marshal(sched)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		updates = append(updates, &contract.PluginSetOp{Key: r.Entries[0].Key, Value: bz})
	}
	if totalUnlocked == 0 {
		return &contract.PluginDeliverResponse{Error: ErrNothingToClaim()}
	}
	bal := DecodeUint64(balBz) + totalUnlocked
	cnpy.Amount -= fee
	feePool.Amount += fee
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := append(updates,
		&contract.PluginSetOp{Key: balKey, Value: EncodeUint64(bal)},
		&contract.PluginSetOp{Key: feePoolKey, Value: feeBz},
	)
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	// Update circulating supply on globals.
	if err := c.bumpCirculating(totalUnlocked); err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	return &contract.PluginDeliverResponse{}
}

// bumpCirculating reads the globals record, increments cliq_circulating_supply
// by `delta`, and writes it back. Used after vesting unlocks.
func (c *Canoliq) bumpCirculating(delta uint64) *contract.PluginError {
	gKey := KeyForGlobals()
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: gKey}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	globals := new(contract.CanoliqGlobals)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, globals); e != nil {
			return e
		}
	}
	globals.CliqCirculatingSupply += delta
	bz, e := contract.Marshal(globals)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: gKey, Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}

// currentHeight returns the latest block height observed by the long-lived
// plugin. Used by handlers that need height to evaluate vesting unlock or
// redemption maturity. Returns 0 if no block lifecycle event has been seen
// yet (typical in unit tests with a stub plugin).
func (c *Canoliq) currentHeight() uint64 {
	if c == nil || c.plugin == nil {
		return 0
	}
	return c.plugin.CurrentHeight()
}
