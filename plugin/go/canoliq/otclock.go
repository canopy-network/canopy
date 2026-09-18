package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// otclock.go implements the OTC lock program: lock cCNPY for a fixed tier
// duration, receive a one-off CPLQ reward at maturity.
//
// Three properties drive the whole design:
//
//  1. CPLQ supply is fixed at genesis and nothing mints. Every uCPLQ this
//     program pays already exists, having been moved out of treasury_cplq by a
//     passed ProposalOTCProgramFund. The reward is therefore reserved from a
//     capped budget at lock time, not computed at maturity: a position that
//     cannot be funded is rejected up front, so no matured position can ever
//     outrun what the program can pay.
//
//  2. The lock moves the cCNPY out of the holder's spendable balance into the
//     position record, matching how vesting and vote-escrow lock tokens. That
//     means DeliverMessageCanoliqRedeem needs no change at all — the balance it
//     reads is already reduced. It also means globals.TotalCcnpySupply must NOT
//     move: the cCNPY still exists, it has only changed custody, and leaving
//     the supply alone is exactly what keeps a locked position appreciating at
//     the pool's normal exchange rate for the whole term.
//
//  3. Maturity is user-driven. Nothing scans open positions per block, so the
//     cost of the program to block production stays flat as it grows.
//
// Reward math is a pure quantity conversion at 1:1 micro-units
// (reward_uCPLQ = locked_uccnpy * tier_bps / 10000). There is no price and no
// oracle, preserving the whitepaper's "no external price oracle on the core
// yield path" commitment.

// OTC lock tier durations. Expressed in days against the existing
// blocksPerDay constant (stake.go) rather than via LockTier, which is a closed
// 3/6/12/24-month set whose ordering is load-bearing for the "stronger tier"
// comparison in DeliverMessageCPLQStake and whose only consumers are
// governance vote weight and buyback shares. This program grants neither.
const (
	otcLock90DBlocks  = 90 * blocksPerDay  // 1_296_000
	otcLock120DBlocks = 120 * blocksPerDay // 1_728_000
)

// otcLockDurationBlocks returns the lock duration in blocks for a tier.
// Returns 0 for an unknown tier; callers gate on validOTCLockTier first.
func otcLockDurationBlocks(tier contract.OTCLockTier) uint64 {
	switch tier {
	case contract.OTCLockTier_OTC_LOCK_90D:
		return otcLock90DBlocks
	case contract.OTCLockTier_OTC_LOCK_120D:
		return otcLock120DBlocks
	default:
		return 0
	}
}

// validOTCLockTier whitelists the legal tiers. OTC_LOCK_UNSPECIFIED is
// rejected so a zero-valued field is never silently treated as a real choice.
func validOTCLockTier(tier contract.OTCLockTier) bool {
	switch tier {
	case contract.OTCLockTier_OTC_LOCK_90D, contract.OTCLockTier_OTC_LOCK_120D:
		return true
	default:
		return false
	}
}

// otcTierBps returns the reward rate in basis points for a tier.
func otcTierBps(tier contract.OTCLockTier, params *contract.CanoliqParams) uint64 {
	switch tier {
	case contract.OTCLockTier_OTC_LOCK_90D:
		return params.OtcTier90Bps
	case contract.OTCLockTier_OTC_LOCK_120D:
		return params.OtcTier120Bps
	default:
		return 0
	}
}

// otcReward computes the CPLQ reward for a position. Flooring favors the
// program budget, never the depositor, so rounding can never overcommit.
func otcReward(ccnpyAmount uint64, tierBps uint64) uint64 {
	return mulDiv(ccnpyAmount, tierBps, 10_000)
}

// CheckMessageOTCLockCreate validates a lock statelessly.
func (c *Canoliq) CheckMessageOTCLockCreate(msg *contract.MessageOTCLockCreate, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.CcnpyAmount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if !validOTCLockTier(msg.Tier) {
		return &contract.PluginCheckResponse{Error: ErrInvalidOTCLockTier()}
	}
	// The minimum is a stateless check because params are available here, and
	// catching dust positions at admission keeps them out of the mempool.
	if msg.CcnpyAmount < params.OtcMinLockUccnpy {
		return &contract.PluginCheckResponse{Error: ErrOTCLockBelowMinimum()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageOTCLockClaim validates a claim statelessly.
func (c *Canoliq) CheckMessageOTCLockClaim(msg *contract.MessageOTCLockClaim, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.ClaimFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageOTCLockCancel validates an early exit statelessly.
func (c *Canoliq) CheckMessageOTCLockCancel(msg *contract.MessageOTCLockCancel, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.ClaimFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// DeliverMessageOTCLockCreate opens a lock position: debits the cCNPY from the
// holder, reserves the tier reward out of the program's unreserved budget, and
// writes the position plus its index entry.
func (c *Canoliq) DeliverMessageOTCLockCreate(msg *contract.MessageOTCLockCreate, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	if !validOTCLockTier(msg.Tier) {
		return &contract.PluginDeliverResponse{Error: ErrInvalidOTCLockTier()}
	}
	if msg.CcnpyAmount < params.OtcMinLockUccnpy {
		return &contract.PluginDeliverResponse{Error: ErrOTCLockBelowMinimum()}
	}
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	ccnpyKey := KeyForCCNPYBalance(msg.FromAddress)
	idxKey := KeyForOTCLockIndex(msg.FromAddress)
	globalsKey := KeyForGlobals()
	availKey := KeyForOTCBudgetAvailable()
	resvKey := KeyForOTCBudgetReserved()
	cQ, fQ, bQ, iQ, gQ, aQ, rQ := qid(), qid(), qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: bQ, Key: ccnpyKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: gQ, Key: globalsKey},
			{QueryId: aQ, Key: availKey},
			{QueryId: rQ, Key: resvKey},
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
	idx := new(contract.OTCLockIndex)
	globals := new(contract.CanoliqGlobals)
	var ccnpyBz, availBz, resvBz []byte
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
			ccnpyBz = r.Entries[0].Value
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case aQ:
			availBz = r.Entries[0].Value
		case rQ:
			resvBz = r.Entries[0].Value
		}
	}
	ccnpyBal := DecodeUint64(ccnpyBz)
	if ccnpyBal < msg.CcnpyAmount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCCNPY()}
	}
	reward := otcReward(msg.CcnpyAmount, otcTierBps(msg.Tier, params))
	available := DecodeUint64(availBz)
	// Reserve up front or reject. This is the guarantee that a matured
	// position always has its payout already set aside.
	if reward == 0 || available < reward {
		return &contract.PluginDeliverResponse{Error: ErrOTCBudgetExhausted()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	ccnpyBal -= msg.CcnpyAmount
	available -= reward
	reserved := DecodeUint64(resvBz) + reward
	height := c.currentHeight()
	globals.NextOtcLockId++
	lockID := globals.NextOtcLockId
	lock := &contract.OTCLock{
		Id:           lockID,
		Address:      msg.FromAddress,
		CcnpyAmount:  msg.CcnpyAmount,
		Tier:         msg.Tier,
		RewardCplq:   reward,
		StartHeight:  height,
		MatureHeight: height + otcLockDurationBlocks(msg.Tier),
	}
	idx.Ids = append(idx.Ids, lockID)
	lockBz, e := contract.Marshal(lock)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	globalsBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	// globals.TotalCcnpySupply is deliberately untouched: the cCNPY moved
	// custody, it was not burned, so the pool's exchange rate is unaffected and
	// the locked position keeps appreciating for the whole term.
	sets := []*contract.PluginSetOp{
		{Key: KeyForOTCLock(msg.FromAddress, lockID), Value: lockBz},
		{Key: idxKey, Value: idxBz},
		{Key: globalsKey, Value: globalsBz},
		{Key: feePoolKey, Value: feeBz},
		{Key: resvKey, Value: EncodeUint64(reserved)},
	}
	deletes := make([]*contract.PluginDeleteOp, 0, 3)
	if ccnpyBal == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: ccnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: ccnpyKey, Value: EncodeUint64(ccnpyBal)})
	}
	if available == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: availKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: availKey, Value: EncodeUint64(available)})
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

// DeliverMessageOTCLockClaim closes a matured position, returning the cCNPY
// and paying the reserved CPLQ in the same transaction.
func (c *Canoliq) DeliverMessageOTCLockClaim(msg *contract.MessageOTCLockClaim, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	st, resp := c.loadOTCLockContext(msg.FromAddress, msg.LockId)
	if resp != nil {
		return resp
	}
	if c.currentHeight() < st.lock.MatureHeight {
		return &contract.PluginDeliverResponse{Error: ErrOTCLockNotMature()}
	}
	if st.cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	st.cnpy.Amount -= fee
	st.feePool.Amount += fee
	// cCNPY returns to the spendable balance; the reserved CPLQ is paid.
	ccnpyBal := st.ccnpyBal + st.lock.CcnpyAmount
	cplqBal := st.cplqBal + st.lock.RewardCplq
	reserved := saturatingSub(st.reserved, st.lock.RewardCplq)
	sets, deletes, e := c.finalizeOTCLock(st, msg.FromAddress, msg.LockId, ccnpyBal, reserved)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets = append(sets, &contract.PluginSetOp{Key: KeyForCPLQBalance(msg.FromAddress), Value: EncodeUint64(cplqBal)})
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	// The reward leaves a non-circulating reserve for a liquid balance, so it
	// enters circulation here. Mirrors applySpend and ClaimVested. Cancel does
	// not bump, because a forfeited reward never reaches circulation.
	if st.lock.RewardCplq > 0 {
		if err := c.bumpCirculating(st.lock.RewardCplq); err != nil {
			return &contract.PluginDeliverResponse{Error: err}
		}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageOTCLockCancel exits a position early: the cCNPY is returned
// intact and the whole reward is forfeited back to the unreserved budget.
func (c *Canoliq) DeliverMessageOTCLockCancel(msg *contract.MessageOTCLockCancel, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	st, resp := c.loadOTCLockContext(msg.FromAddress, msg.LockId)
	if resp != nil {
		return resp
	}
	// Refuse to cancel an already-matured position. The reward is earned by
	// then, so cancelling would destroy it for nothing when a claim would pay.
	if c.currentHeight() >= st.lock.MatureHeight {
		return &contract.PluginDeliverResponse{Error: ErrOTCLockMatured()}
	}
	if st.cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	st.cnpy.Amount -= fee
	st.feePool.Amount += fee
	ccnpyBal := st.ccnpyBal + st.lock.CcnpyAmount
	reserved := saturatingSub(st.reserved, st.lock.RewardCplq)
	available := st.available + st.lock.RewardCplq
	sets, deletes, e := c.finalizeOTCLock(st, msg.FromAddress, msg.LockId, ccnpyBal, reserved)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets = append(sets, &contract.PluginSetOp{Key: KeyForOTCBudgetAvailable(), Value: EncodeUint64(available)})
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// otcLockContext is the state both close paths read. Claim and cancel differ
// only in the maturity gate and where the reward goes, so the read and the
// common write staging are shared rather than duplicated.
type otcLockContext struct {
	cnpy      *contract.Account
	feePool   *contract.Pool
	lock      *contract.OTCLock
	idx       *contract.OTCLockIndex
	ccnpyBal  uint64
	cplqBal   uint64
	available uint64
	reserved  uint64
}

// loadOTCLockContext batches the single StateRead both close paths need. A
// non-nil response is a terminal error the caller must return as-is.
func (c *Canoliq) loadOTCLockContext(addr []byte, lockID uint64) (*otcLockContext, *contract.PluginDeliverResponse) {
	cnpyKey := contract.KeyForAccount(addr)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	lockKey := KeyForOTCLock(addr, lockID)
	ccnpyKey := KeyForCCNPYBalance(addr)
	cplqKey := KeyForCPLQBalance(addr)
	idxKey := KeyForOTCLockIndex(addr)
	availKey := KeyForOTCBudgetAvailable()
	resvKey := KeyForOTCBudgetReserved()
	cQ, fQ, lQ, bQ, pQ, iQ, aQ, rQ := qid(), qid(), qid(), qid(), qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: lQ, Key: lockKey},
			{QueryId: bQ, Key: ccnpyKey},
			{QueryId: pQ, Key: cplqKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: aQ, Key: availKey},
			{QueryId: rQ, Key: resvKey},
		},
	})
	if err != nil {
		return nil, &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return nil, &contract.PluginDeliverResponse{Error: resp.Error}
	}
	st := &otcLockContext{
		cnpy:    new(contract.Account),
		feePool: new(contract.Pool),
		lock:    new(contract.OTCLock),
		idx:     new(contract.OTCLockIndex),
	}
	present := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, st.cnpy); e != nil {
				return nil, &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, st.feePool); e != nil {
				return nil, &contract.PluginDeliverResponse{Error: e}
			}
		case lQ:
			if e := contract.Unmarshal(r.Entries[0].Value, st.lock); e != nil {
				return nil, &contract.PluginDeliverResponse{Error: e}
			}
			// Same presence convention as the unstake path: an absent key
			// unmarshals to a zero record, so the address field is the tell.
			present = st.lock.Address != nil
		case bQ:
			st.ccnpyBal = DecodeUint64(r.Entries[0].Value)
		case pQ:
			st.cplqBal = DecodeUint64(r.Entries[0].Value)
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, st.idx); e != nil {
				return nil, &contract.PluginDeliverResponse{Error: e}
			}
		case aQ:
			st.available = DecodeUint64(r.Entries[0].Value)
		case rQ:
			st.reserved = DecodeUint64(r.Entries[0].Value)
		}
	}
	if !present {
		return nil, &contract.PluginDeliverResponse{Error: ErrOTCLockNotFound()}
	}
	return st, nil
}

// finalize stages the writes common to claim and cancel: return the cCNPY,
// settle the fee, drop the position and its index entry, and write back the
// reserved budget. The caller appends its own reward-specific ops.
func (c *Canoliq) finalizeOTCLock(st *otcLockContext, addr []byte, lockID uint64, ccnpyBal, reserved uint64) ([]*contract.PluginSetOp, []*contract.PluginDeleteOp, *contract.PluginError) {
	feeBz, e := contract.Marshal(st.feePool)
	if e != nil {
		return nil, nil, e
	}
	cnpyBz, e := contract.Marshal(st.cnpy)
	if e != nil {
		return nil, nil, e
	}
	sets := []*contract.PluginSetOp{
		{Key: KeyForCCNPYBalance(addr), Value: EncodeUint64(ccnpyBal)},
		{Key: contract.KeyForFeePool(c.Config.ChainId), Value: feeBz},
		{Key: KeyForOTCBudgetReserved(), Value: EncodeUint64(reserved)},
	}
	deletes := []*contract.PluginDeleteOp{{Key: KeyForOTCLock(addr, lockID)}}
	if st.cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: contract.KeyForAccount(addr)})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: contract.KeyForAccount(addr), Value: cnpyBz})
	}
	idxKey := KeyForOTCLockIndex(addr)
	st.idx.Ids = removeUint64(st.idx.Ids, lockID)
	if len(st.idx.Ids) == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: idxKey})
	} else {
		idxBz, e := contract.Marshal(st.idx)
		if e != nil {
			return nil, nil, e
		}
		sets = append(sets, &contract.PluginSetOp{Key: idxKey, Value: idxBz})
	}
	return sets, deletes, nil
}

// saturatingSub subtracts without underflowing. The reserved counter should
// never go negative by construction, but a floor here means a corrupted or
// hand-seeded counter degrades instead of wrapping to a huge number.
func saturatingSub(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}

// fundOTCProgram moves CPLQ from the DAO treasury into the OTC lock program's
// unreserved budget. Called from dispatchPassed when a ProposalOTCProgramFund
// passes, which is the only way the program is ever funded.
//
// Nothing mints here and nothing enters circulation: treasury CPLQ and program
// CPLQ are both non-circulating reserves, so CplqCirculatingSupply is
// deliberately untouched. It rises only when a matured position is claimed.
//
// All three scalars are read in one batch and written from locally computed
// values. readScalar sees committed state, not pending set ops, and the FSM
// applies sets last-write-wins, so re-reading a key already staged in this
// batch would silently drop the earlier write. That is precisely the bug this
// change fixes in buyback.go.
func (c *Canoliq) fundOTCProgram(p *contract.ProposalOTCProgramFund) *contract.PluginError {
	if p == nil || p.Amount == 0 {
		return ErrInvalidProposalPayload()
	}
	treasuryKey := KeyForTreasuryCPLQ()
	availKey := KeyForOTCBudgetAvailable()
	tQ, aQ := qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: tQ, Key: treasuryKey},
			{QueryId: aQ, Key: availKey},
		},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	var treasury, available uint64
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case tQ:
			treasury = DecodeUint64(r.Entries[0].Value)
		case aQ:
			available = DecodeUint64(r.Entries[0].Value)
		}
	}
	if treasury < p.Amount {
		return ErrInsufficientTreasuryCPLQ()
	}
	treasury -= p.Amount
	available += p.Amount
	sets := []*contract.PluginSetOp{{Key: availKey, Value: EncodeUint64(available)}}
	deletes := make([]*contract.PluginDeleteOp, 0, 1)
	if treasury == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: treasuryKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: treasuryKey, Value: EncodeUint64(treasury)})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return e
	}
	return nil
}
