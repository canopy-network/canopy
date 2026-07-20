package contract

// CheckMessageBorrow statelessly validates a 'borrow' message. Market
// existence, Insolvent/layer4-pending gating, collateral-position
// existence, and the max-LTV check all require state, so they run at
// DeliverTx (matching update_price.go's established stateless/stateful split).
func (c *Contract) CheckMessageBorrow(msg *MessageBorrow) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.BorrowAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageBorrow handles 'borrow' per ARCM Section 19.2.1:
//
//	Step 0 -- Insolvent-market + layer4-pending admission block (ARCM v3.7 J1, Section 9.2/19.2)
//	Step 1 -- AccrueInterest (AYIS Section 12.3's mandatory ordering)
//	Step 2 -- consolidate existing debt via ScaledDebt()
//	Step 3 -- pos.borrow_index_at_open = current B_index
//	Step 4 -- market.total_borrowed += borrow_amount (checked)
//
// The max-LTV check (ARCM Section 4) runs before any state mutation.
func (c *Contract) DeliverMessageBorrow(msg *MessageBorrow, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.BorrowAmount == 0 {
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
	if pErr := checkMarketNotDeprecated(market, msg.MarketId); pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}

	// [ARCM v3.7, J1] A market with no real backing liquidity left cannot
	// honestly back new debt.
	if market.Status == MarketStatus_INSOLVENT {
		return &PluginDeliverResponse{Error: ErrMarketInsolvent(msg.MarketId)}
	}
	// [ARCM Section 9.3a, C2-confirmed admission set] index_overflow_halted
	// blocks deposit AND borrow -- unlike Layer4PendingCount (below) and
	// unlike withdraw.go's deliberate non-check of this same flag (see
	// withdraw.go's own comment: RedeemShares() prices safely against a
	// frozen S_rate, but MintShares()-adjacent paths like borrow's own
	// debt/collateral math have no equivalent safety guarantee against a
	// market whose index encoding has already overflowed). This check was
	// missing from the original version of this file -- deposit.go already
	// had it; borrow.go did not, despite ARCM's own admission set naming
	// both deposit and borrow together.
	if market.IndexOverflowHalted {
		return &PluginDeliverResponse{Error: ErrMarketIndexOverflowHalted(msg.MarketId)}
	}
	// [ARCM Section 9.2b, extended to borrow in v3.7 J1]
	if market.Layer4PendingCount > 0 {
		return &PluginDeliverResponse{Error: ErrMarketLayer4Pending(msg.MarketId)}
	}

	// Step 1 -- MUST run before any debt read (AYIS Section 12.3). Safe to
	// call redundantly if BeginBlock already accrued this market this
	// block -- AccrueInterest's own double-accrual guard makes it a no-op.
	if aErr := AccrueInterest(c, msg.MarketId); aErr != nil {
		return &PluginDeliverResponse{Error: aErr}
	}

	// A borrower must have an open collateral position (via
	// deposit_collateral) before borrowing.
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
		return &PluginDeliverResponse{Error: ErrNoCollateralPosition(msg.MarketId, msg.Address)}
	}
	pos := &BorrowerPosition{}
	if uErr := Unmarshal(posBytes, pos); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	// [CUSTODY FIX] borrow computes/checks debt correctly but never moves a
	// real token to the borrower -- symmetric gap to deposit/withdraw's.
	// Debits the market's supply pool (PoolPurposeSupply -- same pool
	// deposit/withdraw use, since borrow draws down what lenders supplied),
	// credits the borrower's real account, by msg.BorrowAmount.
	acctKey := KeyForAccount(msg.Address)
	poolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeSupply)
	poolKey := KeyForFeePool(poolId)
	custodyReadResp, cErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: acctKey},
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

	// Step 2 -- NEVER read pos.DebtPrincipal directly as "current debt."
	currentDebt := ScaledDebt(pos, bIndexNow)

	if msg.BorrowAmount > (^uint64(0) - currentDebt) {
		return &PluginDeliverResponse{Error: ErrTotalBorrowedOverflow(msg.MarketId, currentDebt, msg.BorrowAmount)}
	}
	newPrincipal := currentDebt + msg.BorrowAmount

	// Max-LTV admission check (ARCM Section 4). Tier comes from the {29}
	// registry, NOT Market's own self-declared asset_tier field.
	tier, tierFound, tErr := GetAssetTier(c, market.CollateralAssetId)
	if tErr != nil {
		return &PluginDeliverResponse{Error: tErr}
	}
	if !tierFound {
		return &PluginDeliverResponse{Error: ErrAssetTierNotFound(market.CollateralAssetId)}
	}
	tierParams, tpFound := GetTierParams(tier)
	if !tpFound {
		return &PluginDeliverResponse{Error: ErrAssetTierNotFound(market.CollateralAssetId)}
	}

	collateralPrice, cpFound, cpErr := ResolvePrice(c, market.CollateralAssetId)
	if cpErr != nil {
		return &PluginDeliverResponse{Error: cpErr}
	}
	if !cpFound {
		return &PluginDeliverResponse{Error: ErrPriceUnavailable(msg.MarketId, market.CollateralAssetId)}
	}
	debtPrice, dpFound, dpErr := ResolvePrice(c, market.DebtAssetId)
	if dpErr != nil {
		return &PluginDeliverResponse{Error: dpErr}
	}
	if !dpFound {
		return &PluginDeliverResponse{Error: ErrPriceUnavailable(msg.MarketId, market.DebtAssetId)}
	}

	maxBorrowQty := MaxBorrowQuantity(pos.CollateralQuantity, collateralPrice, tierParams.LTVMaxBps, debtPrice)
	if newPrincipal > maxBorrowQty {
		return &PluginDeliverResponse{Error: ErrExceedsMaxLTV(msg.MarketId, newPrincipal, maxBorrowQty)}
	}

	// Step 3 -- DeliverTx context -> reverting EncodeUint128 (not
	// TryEncodeUint128, which is BeginBlock-only -- see
	// SetBorrowIndexTry's own comment on the split).
	bIndexEncoded, encErr := EncodeUint128(bIndexNow)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}
	pos.DebtPrincipal = newPrincipal
	pos.BorrowIndexAtOpen = bIndexEncoded

	// [CUSTODY FIX] Debit the supply pool, credit the borrower's account,
	// by msg.BorrowAmount. Checked via custody_arith.go's shared pure
	// functions -- debitPoolAmount fails with ErrInsufficientFunds if the
	// pool doesn't actually hold enough real liquidity to back this borrow.
	acctBytes := entryValue(custodyReadResp, 0)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	poolBytes := entryValue(custodyReadResp, 1)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}
	if dErr := debitPoolAmount(pool, msg.BorrowAmount); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	// [FIX] Match fsm/account.go's SetPool() convention: a pool drained to
	// zero is DELETED, not written as an explicit zero-value record.
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
	if cErr := creditAccountAmount(msg.Address, account, msg.BorrowAmount); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	custodyWriteReq := &PluginStateWriteRequest{
		Sets: []*PluginSetOp{{Key: acctKey, Value: acctBytesOut}},
	}
	if deletePool {
		custodyWriteReq.Deletes = []*PluginDeleteOp{{Key: poolKey}}
	} else {
		custodyWriteReq.Sets = append(custodyWriteReq.Sets, &PluginSetOp{Key: poolKey, Value: poolBytesOut})
	}
	custodyWriteResp, cwErr := c.plugin.StateWrite(c, custodyWriteReq)
	if cwErr != nil {
		return &PluginDeliverResponse{Error: cwErr}
	}
	if custodyWriteResp.Error != nil {
		return &PluginDeliverResponse{Error: custodyWriteResp.Error}
	}

	// Step 4 -- [REFACTORED, ARCM Section 19.3, C1] Now routes through the
	// single mandatory applyDebtDelta() write path instead of inline
	// arithmetic. SafeInt64FromUint64 guards the uint64->int64 conversion
	// itself (the exact cast C1 identified as silently wraparound-prone in
	// prior spec versions) before the signed delta ever reaches
	// applyDebtDelta()'s own overflow guard.
	borrowDelta, safeOk := SafeInt64FromUint64(msg.BorrowAmount)
	if !safeOk {
		return &PluginDeliverResponse{Error: ErrInt64CastOverflow("borrow.BorrowAmount", msg.BorrowAmount)}
	}
	if dErr := applyDebtDelta(market, msg.MarketId, borrowDelta); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}

	posBytesOut, mErr := Marshal(pos)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	if sErr := SaveMarket(c, msg.MarketId, market); sErr != nil {
		return &PluginDeliverResponse{Error: sErr}
	}
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{{Key: posKey, Value: posBytesOut}},
	})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}
