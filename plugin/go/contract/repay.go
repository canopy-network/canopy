package contract

import (
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"
)

// CheckMessageRepay statelessly validates a 'repay' message.
func (c *Contract) CheckMessageRepay(msg *MessageRepay) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.RepayAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageRepay handles 'repay' per ARCM Section 19.2.1. NOT blocked
// by Insolvent status (Section 9.2/19.2) -- borrowers must always be able
// to discharge debt. For an Insolvent market, the repaid-funds leg routes
// to R_fund[market_id] instead of general custody (ARCM v3.7 K1, Section 9.3(2)).
func (c *Contract) DeliverMessageRepay(msg *MessageRepay, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.RepayAmount == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}

	market, found, err := GetMarket(c, msg.MarketId)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	if pErr := checkMarketNotPaused(market, msg.MarketId); pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}

	var events []*Event
	aiEvent, aErr := AccrueInterest(c, msg.MarketId)
	if aErr != nil {
		return &PluginDeliverResponse{Error: aErr}
	}
	if aiEvent != nil {
		events = append(events, aiEvent)
	}

	posKey := KeyForBorrowerPosition(msg.MarketId, msg.Address)
	posReadResp, rErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: 0, Key: posKey}},
	})
	if rErr != nil {
		return &PluginDeliverResponse{Error: rErr}
	}
	if posReadResp.Error != nil {
		return &PluginDeliverResponse{Error: posReadResp.Error}
	}
	posBytes := entryValue(posReadResp, 0)
	if len(posBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrBorrowerPositionNotFound(msg.MarketId, msg.Address)}
	}
	pos := &BorrowerPosition{}
	if uErr := Unmarshal(posBytes, pos); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	// [CUSTODY FIX] repay computes actualRepaid correctly but never debits
	// a real token from the repayer for that leg. Read the repayer's
	// account and the market's supply pool up front.
	custodyAcctKey := KeyForAccount(msg.Address)
	poolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeSupply)
	poolKey := KeyForFeePool(poolId)
	custodyReadResp, cErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: custodyAcctKey},
			{QueryId: 1, Key: poolKey},
		},
	})
	if cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	if custodyReadResp.Error != nil {
		return &PluginDeliverResponse{Error: custodyReadResp.Error}
	}

	bIndexNow, biFound, biErr := GetBorrowIndex(c, msg.MarketId)
	if biErr != nil {
		return &PluginDeliverResponse{Error: biErr}
	}
	if !biFound {
		return &PluginDeliverResponse{Error: ErrMarketIndexNotInitialized(msg.MarketId)}
	}

	currentDebt, sdErr := ScaledDebt(pos, bIndexNow)
	if sdErr != nil {
		return &PluginDeliverResponse{Error: sdErr}
	}

	// [CUSTODY FIX, replaces unbacked refund mint] No FSM-level escrow of
	// msg.RepayAmount exists (confirmed: DeliverTx is a bare type-switch,
	// no pre-debit). The prior refund path credited account.Amount for any
	// overpayment with no corresponding debit anywhere in the transaction --
	// live unauthenticated token minting, exploitable by any address with an
	// open borrower position. Since there is nothing to refund out of
	// custody, overpayment must be rejected outright rather than "refunded."
	if msg.RepayAmount > currentDebt {
		return &PluginDeliverResponse{Error: ErrRepayExceedsDebt(msg.MarketId, currentDebt, msg.RepayAmount)}
	}
	var newDebt, actualRepaid uint64
	if msg.RepayAmount == currentDebt {
		newDebt = 0
		actualRepaid = currentDebt
	} else {
		newDebt = currentDebt - msg.RepayAmount
		actualRepaid = msg.RepayAmount
	}

	// [CUSTODY FIX] Debit the repayer's real account by actualRepaid
	// (always, regardless of market status). Only actualRepaid is owed --
	// any amount above currentDebt was already rejected above via
	// ErrRepayExceedsDebt, so actualRepaid is exactly what should leave
	// custody on the debit side.
	var sets []*PluginSetOp
	var custodyAcctBytesOut []byte
	var custodyPoolBytesOut []byte
	if actualRepaid > 0 {
		acctBytes := entryValue(custodyReadResp, 0)
		account := &Account{}
		if len(acctBytes) > 0 {
			if uErr := Unmarshal(acctBytes, account); uErr != nil {
				return &PluginDeliverResponse{Error: uErr}
			}
		}
		if dErr := debitAccountAmount(account, actualRepaid); dErr != nil {
			return &PluginDeliverResponse{Error: dErr}
		}
		var mErr *PluginError
		custodyAcctBytesOut, mErr = Marshal(account)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}

		// [Branch-dependent destination] Only credit the supply pool when the
		// market is NOT Insolvent -- the Insolvent branch below already routes
		// actualRepaid's value to R_fund ({18}, ARCM-owned); crediting the
		// supply pool here too would double-count the same real debit.
		if market.Status != MarketStatus_INSOLVENT {
			poolBytes := entryValue(custodyReadResp, 1)
			pool := &Pool{Id: poolId}
			if len(poolBytes) > 0 {
				if uErr := Unmarshal(poolBytes, pool); uErr != nil {
					return &PluginDeliverResponse{Error: uErr}
				}
				pool.Id = poolId
			}
			if cErr := creditPoolAmount(msg.MarketId, pool, actualRepaid); cErr != nil {
				return &PluginDeliverResponse{Error: cErr}
			}
			custodyPoolBytesOut, mErr = Marshal(pool)
			if mErr != nil {
				return &PluginDeliverResponse{Error: mErr}
			}
		}
	}

	bIndexEncoded, encErr := EncodeUint128(bIndexNow)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}
	pos.DebtPrincipal = newDebt
	pos.BorrowIndexAtOpen = bIndexEncoded

	// [REFACTORED, ARCM Section 19.3, C1] Now routes through the single
	// mandatory applyDebtDelta() write path (mirrors borrow.go's identical
	// refactor) instead of a standalone compare-before-subtract beside it.
	// A negative delta correctly reproduces the exact same
	// compare-before-subtract/zero-clamp behaviour applyDebtDelta()'s own
	// delta < 0 branch implements -- this is not new underflow logic, only
	// its relocation into the one function every total_borrowed mutation
	// (borrow, repay, liquidation) is meant to share.
	if actualRepaid > 0 {
		repaidDelta, safeOk := SafeInt64FromUint64(actualRepaid)
		if !safeOk {
			return &PluginDeliverResponse{Error: ErrInt64CastOverflow("repay.actualRepaid", actualRepaid)}
		}
		clampedFrom, dErr := applyDebtDelta(market, msg.MarketId, -repaidDelta)
		if dErr != nil {
			return &PluginDeliverResponse{Error: dErr}
		}
		if clampedFrom > 0 {
			// [NEW] applyDebtDelta's decrement branch clamped TotalBorrowed to
			// zero -- emit the dust-clamp event with the real pre-clamp value,
			// mirroring withdraw.go's H4 fix for total_supplied.
			payload := &EventTotalBorrowedDustClamp{
				MarketId:       msg.MarketId,
				Source:         "repay",
				DecreaseAmount: actualRepaid,
				PreClampValue:  clampedFrom,
			}
			anyMsg, aErr := anypb.New(payload)
			if aErr != nil {
				return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
			}
			events = append(events, &Event{
				EventType: "total_borrowed_dust_clamp",
				Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
			})
		}

		// [FIX, session finding] Custody (account debit + pool credit),
		// R_fund routing, market.TotalBorrowed, and the borrower's position
		// used to commit via up to FOUR independent StateWrite calls -- the
		// same non-atomicity bug class found and fixed this session in
		// liquidate_position.go and borrow.go. Per the Canopy builder docs'
		// canonical pattern (batch-read, batch-write, ONE StateWrite call is
		// atomic -- no cross-call guarantee exists), a failure partway
		// through used to leave some of these committed and others not --
		// e.g. the repayer's account already debited while
		// market.TotalBorrowed still showed the old, higher debt. All four
		// are now accumulated into one sets/deletes pair, committed in a
		// single StateWrite at the end of this function.
		sets = append(sets, &PluginSetOp{Key: custodyAcctKey, Value: custodyAcctBytesOut})
		if custodyPoolBytesOut != nil {
			sets = append(sets, &PluginSetOp{Key: poolKey, Value: custodyPoolBytesOut})
		}
	}

	// [ARCM v3.7, K1] Insolvent-market routing. DeliverTx context ->
	// reverting EncodeUint128 (SetReserveFundTry's own comment flags this
	// exact leg as out of its Try/BeginBlock-only scope).
	if market.Status == MarketStatus_INSOLVENT && actualRepaid > 0 {
		rFund, rfFound, rfErr := GetReserveFund(c, msg.MarketId)
		if rfErr != nil {
			return &PluginDeliverResponse{Error: rfErr}
		}
		if !rfFound {
			rFund = big.NewInt(0)
		}
		newRFund := new(big.Int).Add(rFund, new(big.Int).SetUint64(actualRepaid))
		rFundEncoded, rfEncErr := EncodeUint128(newRFund)
		if rfEncErr != nil {
			return &PluginDeliverResponse{Error: rfEncErr}
		}
		sets = append(sets, &PluginSetOp{Key: KeyForReserveFund(msg.MarketId), Value: rFundEncoded})
	}

	marketBytesOut, mErr := Marshal(market)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	sets = append(sets, &PluginSetOp{Key: KeyForMarket(msg.MarketId), Value: marketBytesOut})

	// [FIX] A BorrowerPosition record holds BOTH collateral_quantity and
	// debt_principal (arbor_state.pb.go). Deleting the whole record on
	// newDebt == 0 alone would silently destroy the borrower's remaining
	// collateral as a side effect of paying off their debt -- a real
	// fund-safety bug caught live on devnet (a full repay after borrow
	// left ErrNoCollateralPosition on the next borrow attempt). The
	// record is only ever deleted when BOTH debt and collateral are zero;
	// otherwise it is always re-saved, even when newDebt == 0, so
	// surviving collateral is preserved for future withdrawal or borrow.
	var deletes []*PluginDeleteOp
	if newDebt == 0 && pos.CollateralQuantity == 0 {
		deletes = append(deletes, &PluginDeleteOp{Key: posKey})
	} else {
		posBytesOut, mErr := Marshal(pos)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
		sets = append(sets, &PluginSetOp{Key: posKey, Value: posBytesOut})
	}

	_ = fee

	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets, Deletes: deletes})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}

	return &PluginDeliverResponse{Events: events}
}
