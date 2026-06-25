package canoliq

import (
	"bytes"

	"github.com/canopy-network/go-plugin/contract"
)

// stake.go implements CPLQ staking — locking liquid CPLQ for governance
// weight, queueing unstakes through an unbond window, and claiming matured
// records back to liquid balance.
//
// Voting weight is read against a snapshot taken at proposal creation height
// (see governance.go::voteWeightFor); to enable that snapshot,
// CPLQStake.staked_at_height records the latest stake increase.

// blocksPerMonth approximates a fixed 30-day month at the 6s block time
// (30*24*3600/6). Used to convert lock-tier durations to block counts.
//
// Note (L5): this intentionally differs from the vesting month in genesis.go,
// which derives a calendar month as blocksPerYear/12 (≈438_000, a ~30.4-day
// month). Lock tiers use a round 30-day month so a "12-month lock" is exactly
// 12×30 days regardless of the configured blocksPerYear; multi-year vesting
// uses the calendar-accurate year/12. The ~6_000-block (~10h) monthly delta is
// deliberate, not a bug.
const blocksPerMonth = 432_000

// blocksPerDay approximates a day at the 6s block time (24*3600/6). Bounds the
// T5 daily-transaction rolling window.
const blocksPerDay = 14_400

// tierMultipliers returns the voting multiplier (in bps, 10000 = 1×) and the
// reward-share boost (in bps) for a lock tier (Tokenomics v1.1 §4.2).
//
//	tier      voting×   reward boost
//	NONE      1.0×      base
//	3M        1.5×      +10%
//	6M        2.0×      +25%
//	12M       3.0×      +50%
//	24M       4.0×      +75%
func tierMultipliers(tier contract.LockTier) (voteMultBps, boostBps uint64) {
	switch tier {
	case contract.LockTier_LOCK_3M:
		return 15_000, 1_000
	case contract.LockTier_LOCK_6M:
		return 20_000, 2_500
	case contract.LockTier_LOCK_12M:
		return 30_000, 5_000
	case contract.LockTier_LOCK_24M:
		return 40_000, 7_500
	default: // LOCK_NONE
		return 10_000, 0
	}
}

// lockTierDurationBlocks returns the lock duration in blocks for a tier.
func lockTierDurationBlocks(tier contract.LockTier) uint64 {
	switch tier {
	case contract.LockTier_LOCK_3M:
		return 3 * blocksPerMonth
	case contract.LockTier_LOCK_6M:
		return 6 * blocksPerMonth
	case contract.LockTier_LOCK_12M:
		return 12 * blocksPerMonth
	case contract.LockTier_LOCK_24M:
		return 24 * blocksPerMonth
	default: // LOCK_NONE
		return 0
	}
}

// validLockTier reports whether t is a known LockTier enum value.
func validLockTier(t contract.LockTier) bool {
	switch t {
	case contract.LockTier_LOCK_NONE, contract.LockTier_LOCK_3M, contract.LockTier_LOCK_6M,
		contract.LockTier_LOCK_12M, contract.LockTier_LOCK_24M:
		return true
	default:
		return false
	}
}

// voteWeightFor returns a staker's governance weight: raw stake scaled by the
// lock tier's voting multiplier (T2). Zero for an absent record.
func voteWeightFor(stake *contract.CPLQStake) uint64 {
	if stake == nil || stake.Address == nil {
		return 0
	}
	voteMultBps, _ := tierMultipliers(stake.LockTier)
	return mulDiv(stake.Amount, voteMultBps, 10_000)
}

// boostedStakeTotal sums every active staker's lock-boosted governance weight
// (voteWeightFor). This is the correct quorum/turnout denominator (M1): vote
// tallies are lock-boosted via voteWeightFor, so the snapshot they are measured
// against must be boosted on the same basis — otherwise a single long-lock
// voter clears quorum with a fraction of the raw stake the threshold implies,
// and per-proposal turnout can exceed 100%. Mirrors the staker iteration in
// distributeBuybackToStakers.
func (c *Canoliq) boostedStakeTotal() (uint64, *contract.PluginError) {
	idx, err := c.loadStakeIndex()
	if err != nil {
		return 0, err
	}
	total := uint64(0)
	for _, addr := range idx.Addresses {
		stake, err := c.loadCPLQStake(addr)
		if err != nil {
			return 0, err
		}
		total += voteWeightFor(stake)
	}
	return total, nil
}

// CheckMessageCPLQStake validates a stake request statelessly.
func (c *Canoliq) CheckMessageCPLQStake(msg *contract.MessageCPLQStake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if !validLockTier(msg.LockTier) {
		return &contract.PluginCheckResponse{Error: ErrInvalidLockTier()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCPLQUnstake validates an unstake request statelessly.
func (c *Canoliq) CheckMessageCPLQUnstake(msg *contract.MessageCPLQUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCPLQClaimUnstake validates a claim_unstake request statelessly.
func (c *Canoliq) CheckMessageCPLQClaimUnstake(msg *contract.MessageCPLQClaimUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// DeliverMessageCPLQStake debits the sender's liquid CPLQ balance, credits
// the per-address CPLQStake record, and increments globals.total_staked_cplq.
// Multiple stakes from the same address aggregate; staked_at_height tracks
// the most recent increase so the snapshot guard can reject post-creation
// stake additions.
func (c *Canoliq) DeliverMessageCPLQStake(msg *contract.MessageCPLQStake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	balKey := KeyForCPLQBalance(msg.FromAddress)
	stakeKey := KeyForCPLQStake(msg.FromAddress)
	idxKey := KeyForCPLQStakeIndex()
	gKey := KeyForGlobals()
	cQ, fQ, bQ, sQ, iQ, gQ := qid(), qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: gQ, Key: gKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	stake := new(contract.CPLQStake)
	idx := new(contract.CPLQStakeIndex)
	globals := new(contract.CanoliqGlobals)
	var balBz []byte
	stakePresent := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case bQ:
			balBz = r.Entries[0].Value
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			stakePresent = stake.Address != nil
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	bal := DecodeUint64(balBz)
	if bal < msg.Amount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCPLQ()}
	}
	bal -= msg.Amount
	cnpy.Amount -= fee
	feePool.Amount += fee
	height := c.currentHeight()
	newLockEnd := height + lockTierDurationBlocks(msg.LockTier)
	if !stakePresent {
		stake = &contract.CPLQStake{
			Address:       msg.FromAddress,
			Amount:        msg.Amount,
			StakedAtHeight: height,
			LockTier:      msg.LockTier,
			LockEndHeight: newLockEnd,
		}
		idx.Addresses = appendStakerIfMissing(idx.Addresses, msg.FromAddress)
	} else {
		stake.Amount += msg.Amount
		stake.StakedAtHeight = height
		// Locks only ever strengthen: a higher tier raises the record's tier
		// and a later end pushes lock_end_height out. Adding LOCK_NONE to an
		// existing lock leaves both untouched (added tokens inherit the lock).
		if msg.LockTier > stake.LockTier {
			stake.LockTier = msg.LockTier
		}
		if newLockEnd > stake.LockEndHeight {
			stake.LockEndHeight = newLockEnd
		}
	}
	globals.TotalStakedCplq += msg.Amount
	stakeBz, e := contract.Marshal(stake)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: stakeKey, Value: stakeBz},
		{Key: idxKey, Value: idxBz},
		{Key: gKey, Value: gBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if bal == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: balKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: balKey, Value: EncodeUint64(bal)})
	}
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCPLQUnstake debits the staker's CPLQStake by `amount`, queues
// an UnstakingCPLQ record maturing at h + cplq_unstaking_blocks, and
// decrements globals.total_staked_cplq immediately. The unstaked tokens
// carry zero voting weight from this height onward.
func (c *Canoliq) DeliverMessageCPLQUnstake(msg *contract.MessageCPLQUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	stakeKey := KeyForCPLQStake(msg.FromAddress)
	idxKey := KeyForCPLQStakeIndex()
	gKey := KeyForGlobals()
	unstIdxKey := KeyForUnstakingIndex(msg.FromAddress)
	cQ, fQ, sQ, iQ, gQ, uiQ := qid(), qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: gQ, Key: gKey},
			{QueryId: uiQ, Key: unstIdxKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	stake := new(contract.CPLQStake)
	idx := new(contract.CPLQStakeIndex)
	globals := new(contract.CanoliqGlobals)
	unstIdx := new(contract.UnstakingIndex)
	stakePresent := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			stakePresent = stake.Address != nil
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case uiQ:
			if e := contract.Unmarshal(r.Entries[0].Value, unstIdx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if !stakePresent || stake.Amount < msg.Amount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientStakedCPLQ()}
	}
	// Vote-escrow lock: locked stake cannot unstake until lock_end_height (T2).
	if stake.LockTier != contract.LockTier_LOCK_NONE && c.currentHeight() < stake.LockEndHeight {
		return &contract.PluginDeliverResponse{Error: ErrStakeLocked()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	stake.Amount -= msg.Amount
	cnpy.Amount -= fee
	feePool.Amount += fee
	if globals.TotalStakedCplq >= msg.Amount {
		globals.TotalStakedCplq -= msg.Amount
	} else {
		globals.TotalStakedCplq = 0
	}
	id := globals.NextUnstakeId
	globals.NextUnstakeId++
	mature := c.currentHeight() + params.CplqUnstakingBlocks
	unstake := &contract.UnstakingCPLQ{
		Id:           id,
		Address:      msg.FromAddress,
		Amount:       msg.Amount,
		MatureHeight: mature,
	}
	uBz, e := contract.Marshal(unstake)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	unstIdx.Ids = append(unstIdx.Ids, id)
	unstIdxBz, e := contract.Marshal(unstIdx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: KeyForCPLQUnstaking(msg.FromAddress, id), Value: uBz},
		{Key: unstIdxKey, Value: unstIdxBz},
		{Key: gKey, Value: gBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if stake.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: stakeKey})
		idx.Addresses = removeStaker(idx.Addresses, msg.FromAddress)
		idxBz, e := contract.Marshal(idx)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: idxKey, Value: idxBz})
	} else {
		sBz, e := contract.Marshal(stake)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: stakeKey, Value: sBz})
	}
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCPLQClaimUnstake matures an UnstakingCPLQ record by returning
// the CPLQ to the staker's liquid balance once mature_height has passed.
func (c *Canoliq) DeliverMessageCPLQClaimUnstake(msg *contract.MessageCPLQClaimUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	uKey := KeyForCPLQUnstaking(msg.FromAddress, msg.UnstakeId)
	balKey := KeyForCPLQBalance(msg.FromAddress)
	unstIdxKey := KeyForUnstakingIndex(msg.FromAddress)
	cQ, fQ, uQ, bQ, uiQ := qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: uQ, Key: uKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: uiQ, Key: unstIdxKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	unstake := new(contract.UnstakingCPLQ)
	unstIdx := new(contract.UnstakingIndex)
	var balBz []byte
	uPresent := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case uQ:
			if e := contract.Unmarshal(r.Entries[0].Value, unstake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			uPresent = unstake.Address != nil
		case bQ:
			balBz = r.Entries[0].Value
		case uiQ:
			if e := contract.Unmarshal(r.Entries[0].Value, unstIdx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if !uPresent {
		return &contract.PluginDeliverResponse{Error: ErrUnstakeNotFound()}
	}
	if c.currentHeight() < unstake.MatureHeight {
		return &contract.PluginDeliverResponse{Error: ErrUnstakeNotMature()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	bal := DecodeUint64(balBz) + unstake.Amount
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: balKey, Value: EncodeUint64(bal)},
		{Key: feePoolKey, Value: feeBz},
	}
	deletes := []*contract.PluginDeleteOp{{Key: uKey}}
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	unstIdx.Ids = removeUint64(unstIdx.Ids, msg.UnstakeId)
	if len(unstIdx.Ids) == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: unstIdxKey})
	} else {
		uiBz, e := contract.Marshal(unstIdx)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: unstIdxKey, Value: uiBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// loadCPLQStake reads a per-address CPLQStake record. Returns (nil, nil)
// when the staker has no record, distinguishable from an empty stake.
func (c *Canoliq) loadCPLQStake(addr []byte) (*contract.CPLQStake, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForCPLQStake(addr)}},
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
	stake := new(contract.CPLQStake)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, stake); e != nil {
		return nil, e
	}
	return stake, nil
}

// loadStakeIndex reads the singleton stake index. Returns an empty index when
// no stakers exist yet.
func (c *Canoliq) loadStakeIndex() (*contract.CPLQStakeIndex, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForCPLQStakeIndex()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	idx := new(contract.CPLQStakeIndex)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
			return nil, e
		}
	}
	return idx, nil
}

func appendStakerIfMissing(addrs [][]byte, addr []byte) [][]byte {
	for _, a := range addrs {
		if bytes.Equal(a, addr) {
			return addrs
		}
	}
	return append(addrs, addr)
}

func removeStaker(addrs [][]byte, addr []byte) [][]byte {
	out := addrs[:0]
	for _, a := range addrs {
		if bytes.Equal(a, addr) {
			continue
		}
		out = append(out, a)
	}
	return out
}
