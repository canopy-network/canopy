package contract

import (
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"
)

// CheckMessageWithdraw statelessly validates a 'withdraw' message (AYIS
// Section 4.4, ARCM Section 19.2.1b). No state reads here -- position
// existence, sufficient-shares, and admission checks all happen at
// DeliverTx, matching deposit's stateless/stateful split.
func (c *Contract) CheckMessageWithdraw(msg *MessageWithdraw) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Shares == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageWithdraw handles a 'withdraw' message: AYIS Section 4.4
// RedeemShares() plus ARCM Section 19.2.1b's total_supplied decrement path.
func (c *Contract) DeliverMessageWithdraw(msg *MessageWithdraw, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}

	marketKey := KeyForMarket(msg.MarketId)
	supplyIndexKey := KeyForSupplyIndex(msg.MarketId)
	lossFactorKey := KeyForLossFactor(msg.MarketId)
	lenderPosKey := KeyForLenderPosition(msg.MarketId, msg.Address)
	// [CUSTODY FIX] Inverse of deposit's custody fix: debit the market's
	// escrow pool, credit the withdrawer's real account. See deposit.go's
	// custody-fix comment for the poolPrefix/escrow rationale; unchanged
	// here.
	acctKey := KeyForAccount(msg.Address)
	poolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeSupply)
	poolKey := KeyForFeePool(poolId)

	const (
		qMarket = iota
		qSupplyIndex
		qLossFactor
		qLenderPos
		qAccount
		qPool
	)
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qMarket, Key: marketKey},
			{QueryId: qSupplyIndex, Key: supplyIndexKey},
			{QueryId: qLossFactor, Key: lossFactorKey},
			{QueryId: qLenderPos, Key: lenderPosKey},
			{QueryId: qAccount, Key: acctKey},
			{QueryId: qPool, Key: poolKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if readResp.Error != nil {
		return &PluginDeliverResponse{Error: readResp.Error}
	}

	marketBytes := entryValue(readResp, qMarket)
	if len(marketBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	market := &Market{}
	if uErr := Unmarshal(marketBytes, market); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	if pErr := checkMarketNotPaused(market, msg.MarketId); pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}

	// [ARCM Section 9.2b, C2-confirmed admission set] layer4_pending_count
	// > 0 blocks BOTH deposit and withdraw. index_overflow_halted is
	// DELIBERATELY NOT checked here -- ARCM Section 9.3a's admission set
	// only blocks deposit/borrow (C2, v3.11): RedeemShares() prices a
	// withdrawal against a frozen S_rate that is exactly as valid the block
	// after freezing as it was the block before (AYIS Invariant I14/I15).
	// Blocking withdrawal on this flag was the exact bug C2 closed --
	// do not reintroduce it here.
	if market.Layer4PendingCount > 0 {
		return &PluginDeliverResponse{Error: ErrMarketLayer4Pending(msg.MarketId)}
	}

	lenderPosBytes := entryValue(readResp, qLenderPos)
	if len(lenderPosBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrLenderPositionNotFound(msg.MarketId, hexAddr(msg.Address))}
	}
	lenderPos := &LenderPosition{}
	if uErr := Unmarshal(lenderPosBytes, lenderPos); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}
	if msg.Shares > lenderPos.Shares {
		return &PluginDeliverResponse{Error: ErrInsufficientShares(msg.MarketId, lenderPos.Shares, msg.Shares)}
	}

	supplyIndexBytes := entryValue(readResp, qSupplyIndex)
	if len(supplyIndexBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	sRate, totalSharesOutstanding := DecodeSupplyIndexRecord(supplyIndexBytes)

	lossFactorBytes := entryValue(readResp, qLossFactor)
	if len(lossFactorBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	lossFactor := DecodeUint128(lossFactorBytes)

	// tokens = shares * s_rate * loss_factor / RAY^2, floor (AYIS Section
	// 4.4). Note: MULTIPLIES by loss_factor, does not divide -- unlike
	// MintShares()'s H1 guard, no zero-guard is needed here. A zero
	// loss_factor (fully Insolvent market) correctly produces tokens == 0,
	// not a divide-by-zero: the lender's position is worth nothing, which
	// is the mathematically correct answer for a fully socialized loss
	// (AYIS Section 4.4 commentary, I8/I11 cross-reference).
	tokensBig := new(big.Int).Mul(new(big.Int).SetUint64(msg.Shares), sRate)
	tokensBig.Mul(tokensBig, lossFactor)
	rayRay := new(big.Int).Mul(RAY, RAY)
	tokensBig.Div(tokensBig, rayRay)

	// [AYIS Section 4.4, J2] Cast-safety guard.
	if tokensBig.BitLen() > 64 {
		return &PluginDeliverResponse{Error: ErrTokenOverflow(msg.MarketId, msg.Shares, tokensBig.String())}
	}
	actualWithdrawn := tokensBig.Uint64()

	// [CUSTODY FIX] Debit the market's escrow Pool.Amount, credit the
	// withdrawer's real Account.Amount -- by actualWithdrawn (the real
	// loss_factor-adjusted token payout), not msg.Shares. Prior to this
	// fix, withdraw computed a real payout amount, decremented
	// total_supplied, and updated LenderPosition, but never moved a real
	// token to the withdrawer -- symmetric gap to deposit's.
	acctBytes := entryValue(readResp, qAccount)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}

	poolBytes := entryValue(readResp, qPool)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}

	// actualWithdrawn == 0 is reachable (fully Insolvent market, loss_factor
	// == 0 -- see this file's own comment above tokensBig's computation).
	// debitPoolAmount/creditAccountAmount both handle amount==0 correctly
	// (no-op mutation, no spurious error), so no special-case branch is
	// needed here -- but noted explicitly since a withdrawal that moves
	// zero real tokens while still deleting/updating LenderPosition is a
	// real, correct, if unusual, path through this function.
	if dErr := debitPoolAmount(pool, actualWithdrawn); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	// [FIX] Match fsm/account.go's SetPool() convention: a pool drained to
	// zero is DELETED, not written as an explicit zero-value record. See
	// borrow.go's identical fix for the full rationale.
	var poolBytesOut []byte
	var deletePool bool
	if pool.Amount == 0 {
		deletePool = true
	} else {
		var mErr *PluginError
		poolBytesOut, mErr = Marshal(pool)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
	}

	if cErr := creditAccountAmount(msg.Address, account, actualWithdrawn); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// [AYIS Section 4.4, H4-corrected] total_shares_outstanding decrement:
	// compare-before-subtract, cannot overflow (a decrement, not an
	// increment -- L1's checked-add guard does not apply here, same
	// reasoning as deposit's RedeemShares commentary).
	var newTotalShares uint64
	var events []*Event
	if msg.Shares >= totalSharesOutstanding {
		// [H4] Capture the pre-clamp value BEFORE zeroing it, so the emitted
		// event carries the amount actually clamped away rather than the
		// post-clamp zero (ARCM v3.11.1 Section III.6 / AYIS Section 4.4).
		preClampShares := totalSharesOutstanding
		newTotalShares = 0
		payload := &EventTotalSharesOutstandingDustClamp{
			MarketId:       msg.MarketId,
			SharesRedeemed: msg.Shares,
			PreClampValue:  preClampShares,
		}
		anyMsg, aErr := anypb.New(payload)
		if aErr != nil {
			return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
		}
		events = append(events, &Event{
			EventType: "total_shares_outstanding_dust_clamp",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		})
	} else {
		newTotalShares = totalSharesOutstanding - msg.Shares
	}

	// [ARCM Section 19.2.1b, M2] total_supplied decrement -- uses
	// actualWithdrawn (RedeemShares()'s real token payout), NOT msg.Shares.
	// Shares are aTokens (AYIS Section 4.1); total_supplied is denominated
	// in the market's native asset (same unit as total_borrowed). Crediting
	// this by a shares count would conflate value and quantity, exactly
	// the Principle 8/F2 violation this document already closed elsewhere.
	if actualWithdrawn >= market.TotalSupplied {
		// [H4] Same fix, mirrored for total_supplied.
		preClampSupplied := market.TotalSupplied
		market.TotalSupplied = 0
		payload := &EventTotalSuppliedDustClamp{
			MarketId:        msg.MarketId,
			ActualWithdrawn: actualWithdrawn,
			PreClampValue:   preClampSupplied,
		}
		anyMsg, aErr := anypb.New(payload)
		if aErr != nil {
			return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
		}
		events = append(events, &Event{
			EventType: "total_supplied_dust_clamp",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		})
	} else {
		market.TotalSupplied -= actualWithdrawn
	}

	// Re-encode SupplyIndexRecord ({26}). s_rate itself is unchanged by a
	// withdrawal (only BeginBlock's AccrueInterest moves S_rate) -- same
	// reasoning as deposit.go's mirrored comment.
	sRateEncoded, encErr := EncodeUint128(sRate)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}
	newSupplyIndexBytes := EncodeSupplyIndexRecord(sRateEncoded, newTotalShares)

	// LenderPosition ({24}): decrement shares. Full withdrawal (shares hits
	// zero) deletes the position per AYIS Section 4.4's closing note,
	// rather than leaving a zero-shares record behind.
	lenderPos.Shares -= msg.Shares

	marketBytesOut, mErr := Marshal(market)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	writeReq := &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: marketKey, Value: marketBytesOut},
			{Key: supplyIndexKey, Value: newSupplyIndexBytes},
			{Key: acctKey, Value: acctBytesOut},
		},
	}
	// [FIX] poolKey's write is now conditional (delete at zero, set otherwise --
	// see the debitPoolAmount block above) rather than always a Sets entry.
	// Two independent zero-conditions (pool, lenderPos) can both fire in the
	// same withdrawal (e.g. a full withdrawal that also drains the pool to
	// zero), so both must be able to append to Deletes independently -- using
	// writeReq.Deletes = []*PluginDeleteOp{...} (assignment, not append) for
	// only ONE of them would silently clobber the other if both conditions
	// are true in the same call.
	if deletePool {
		writeReq.Deletes = append(writeReq.Deletes, &PluginDeleteOp{Key: poolKey})
	} else {
		writeReq.Sets = append(writeReq.Sets, &PluginSetOp{Key: poolKey, Value: poolBytesOut})
	}
	if lenderPos.Shares == 0 {
		writeReq.Deletes = append(writeReq.Deletes, &PluginDeleteOp{Key: lenderPosKey})
	} else {
		lenderPosBytesOut, mErr := Marshal(lenderPos)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
		writeReq.Sets = append(writeReq.Sets, &PluginSetOp{Key: lenderPosKey, Value: lenderPosBytesOut})
	}

	writeResp, err := c.plugin.StateWrite(c, writeReq)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{Events: events}
}

// hexAddr formats a raw address for use in error messages.
func hexAddr(addr []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, len(addr)*2)
	for i, b := range addr {
		out[i*2] = hexDigits[b>>4]
		out[i*2+1] = hexDigits[b&0x0f]
	}
	return string(out)
}
