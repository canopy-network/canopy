package contract

// CheckMessageDepositCollateral statelessly validates a 'deposit_collateral'
// message (ARCM Section 19.2, borrower-side). No state reads here -- market
// existence is checked at DeliverTx, matching deposit.go's stateless/stateful
// split.
func (c *Contract) CheckMessageDepositCollateral(msg *MessageDepositCollateral) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Quantity == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageDepositCollateral handles 'deposit_collateral': increases
// BorrowerPosition.CollateralQuantity at {17} (ARCM Section 19.2).
//
// [ARCM Section 19.2, transaction inventory] deposit_collateral carries NO
// admission gate -- it is grouped with repay, not with deposit/borrow.
// Specifically: NOT blocked by market.status == Insolvent (a lender-side
// wipeout has no bearing on a borrower adding collateral to their own
// position -- if anything this improves that position's HF), NOT blocked by
// index_overflow_halted (Section 9.3a's admission set is deposit/borrow
// only), NOT blocked by layer4_pending_count > 0 (Section 9.2b's gate is
// deposit/withdraw/borrow only), and NOT blocked by oracle staleness --
// Section 16 (Emergency Mode) explicitly ALLOWS collateral additions during
// staleness, the same as repayments. Unlike withdraw_collateral, this
// transaction never decreases HF, so none of the safety reasons for gating
// those other transaction types apply here.
func (c *Contract) DeliverMessageDepositCollateral(msg *MessageDepositCollateral, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}

	marketKey := KeyForMarket(msg.MarketId)
	borrowerPosKey := KeyForBorrowerPosition(msg.MarketId, msg.Address)
	// [CUSTODY FIX] The depositor's real account and the market's
	// COLLATERAL escrow pool -- deliberately PoolPurposeCollateral, not
	// PoolPurposeSupply. create_market.go never validates
	// CollateralAssetId != DebtAssetId, so even a same-asset market must
	// keep these two custody domains structurally separate.
	acctKey := KeyForAccount(msg.Address)
	poolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeCollateral)
	poolKey := KeyForFeePool(poolId)

	// Batched read: Market (existence check only), this address's existing
	// BorrowerPosition ({17}, may not exist yet), the depositor's Account,
	// and the market's collateral escrow Pool. One round trip.
	const (
		qMarket = iota
		qBorrowerPos
		qAccount
		qPool
	)
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qMarket, Key: marketKey},
			{QueryId: qBorrowerPos, Key: borrowerPosKey},
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
	// [DEPRECATE-MARKET, added alongside deprecate_market.go] Unlike
	// PAUSED (see this function's other admission-gate note: pause is
	// deliberately NOT checked here, since adding safety-margin collateral
	// to an existing position is a defensive action a pause shouldn't
	// block), DEPRECATED is checked. The reasoning is different, not
	// inconsistent: a deprecated market is permanently winding down --
	// deposit and borrow are blocked forever, so there is no future
	// borrow this collateral could ever protect. Allowing a deposit here
	// would not create risk, but it WOULD strand real funds in a market
	// with no path to ever using them productively again -- the deposit
	// itself becomes the harm, not a defense against one.
	market := &Market{}
	if uErr := Unmarshal(marketBytes, market); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}
	if pErr := checkMarketNotDeprecated(market, msg.MarketId); pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}

	borrowerPosBytes := entryValue(readResp, qBorrowerPos)
	borrowerPos := &BorrowerPosition{}
	if len(borrowerPosBytes) > 0 {
		if uErr := Unmarshal(borrowerPosBytes, borrowerPos); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	} else {
		borrowerPos.MarketId = msg.MarketId
		borrowerPos.Address = msg.Address
		// DebtPrincipal and BorrowIndexAtOpen remain at their zero values --
		// a position with collateral but no debt yet is a valid, ordinary
		// state (the borrower has not called 'borrow' yet).
	}

	// Checked-add guard -- same idiom as deposit.go's total_supplied /
	// total_shares_outstanding guards (ARCM Section 19.2.1b, M2 pattern).
	if msg.Quantity > (^uint64(0) - borrowerPos.CollateralQuantity) {
		return &PluginDeliverResponse{Error: ErrCollateralOverflow(msg.MarketId, borrowerPos.CollateralQuantity, msg.Quantity)}
	}
	borrowerPos.CollateralQuantity += msg.Quantity

	borrowerPosBytesOut, mErr := Marshal(borrowerPos)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// [CUSTODY FIX] Debit depositor's real Account.Amount, credit the
	// market's collateral escrow Pool.Amount, by msg.Quantity.
	acctBytes := entryValue(readResp, qAccount)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if dErr := debitAccountAmount(account, msg.Quantity); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	poolBytes := entryValue(readResp, qPool)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}
	if cErr := creditPoolAmount(msg.MarketId, pool, msg.Quantity); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	poolBytesOut, mErr := Marshal(pool)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: borrowerPosKey, Value: borrowerPosBytesOut},
			{Key: acctKey, Value: acctBytesOut},
			{Key: poolKey, Value: poolBytesOut},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}

// CheckMessageWithdrawCollateral statelessly validates a
// 'withdraw_collateral' message.
func (c *Contract) CheckMessageWithdrawCollateral(msg *MessageWithdrawCollateral) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Quantity == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageWithdrawCollateral handles 'withdraw_collateral'.
//
// [CUSTODY FIX, session finding] The comment previously here claimed this
// handler was "bookkeeping-only... no Account.Amount fund transfer occurs
// on either side" -- stale relative to the code below it, which debits the
// market's collateral escrow pool and credits the withdrawer's real
// account, mirroring deposit_collateral's own (already-real) custody move.
// Whichever prior session wired this up did not update this header
// comment to match; corrected here rather than left to mislead a future
// reader into believing no real funds move on this path.
//
// Admission gates deliberately NOT applied here, matching
// deposit_collateral's own established reasoning (see that function's
// comment): NOT gated by market.status == Insolvent (a lender-side
// wipeout doesn't prevent a borrower from managing their own collateral
// position -- the real protection against an unsafe withdrawal is the
// health-factor check below, not market-wide status), NOT gated by
// index_overflow_halted or layer4_pending_count (ARCM's admission-set
// text for both names the LENDER-side 'deposit'/'withdraw'/'borrow'
// message types, not the borrower-side deposit_collateral/
// withdraw_collateral pair). The health-factor check IS the safety gate
// for this transaction, and it is unconditional -- it applies regardless
// of any market-wide flag.
//
// [DISCLOSED GAP] Emergency Mode (ARCM Section 16, P15: "collateral
// withdrawals... BLOCKED unconditionally during staleness") is NOT
// enforced here. Emergency Mode has no implementation anywhere in this
// codebase yet ({21} is a reserved key prefix only) -- this handler
// cannot honor a rule that doesn't exist yet to be read. Flagged
// explicitly rather than silently omitted.
func (c *Contract) DeliverMessageWithdrawCollateral(msg *MessageWithdrawCollateral, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.Quantity == 0 {
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

	posKey := KeyForBorrowerPosition(msg.MarketId, msg.Address)
	// [CUSTODY FIX] Inverse of deposit_collateral: debit the market's
	// collateral escrow pool, credit the withdrawer's real account. Read
	// alongside the existing BorrowerPosition query in one batch.
	acctKey := KeyForAccount(msg.Address)
	poolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeCollateral)
	poolKey := KeyForFeePool(poolId)
	const (
		qPos = iota
		qAccount
		qPool
	)
	posReadResp, rErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qPos, Key: posKey},
			{QueryId: qAccount, Key: acctKey},
			{QueryId: qPool, Key: poolKey},
		},
	})
	if rErr != nil {
		return &PluginDeliverResponse{Error: rErr}
	}
	if posReadResp.Error != nil {
		return &PluginDeliverResponse{Error: posReadResp.Error}
	}
	posBytes := entryValue(posReadResp, qPos)
	if len(posBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrNoCollateralPosition(msg.MarketId, msg.Address)}
	}
	pos := &BorrowerPosition{}
	if uErr := Unmarshal(posBytes, pos); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	if msg.Quantity > pos.CollateralQuantity {
		return &PluginDeliverResponse{Error: ErrInsufficientCollateral(msg.MarketId, pos.CollateralQuantity, msg.Quantity)}
	}

	// AccrueInterest MUST run before any debt read (AYIS Section 12.3).
	if aErr := AccrueInterest(c, msg.MarketId); aErr != nil {
		return &PluginDeliverResponse{Error: aErr}
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
	newCollateralQty := pos.CollateralQuantity - msg.Quantity

	// Only enforce the health-factor check when there is actual debt open
	// against this position (ComputeHealthFactorScaled's own documented
	// precondition). Zero debt means any withdrawal amount up to the full
	// collateral balance is unconditionally safe.
	if currentDebt > 0 {
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

		resultingHF := ComputeHealthFactorScaled(newCollateralQty, collateralPrice, tierParams.LTVLiqBps, currentDebt, debtPrice)
		if resultingHF.Cmp(HFLiquidatableThresholdScaled) <= 0 {
			return &PluginDeliverResponse{Error: ErrWithdrawalExceedsHF(msg.MarketId, resultingHF.String())}
		}
	}

	// [CUSTODY FIX] Debit the market's collateral pool, credit the
	// withdrawer's real account, by msg.Quantity -- runs AFTER the HF
	// check above passes, so a withdrawal that would leave the position
	// liquidatable is rejected before any custody or position state
	// mutates, matching this file's existing guard-before-mutation order.
	acctBytes := entryValue(posReadResp, qAccount)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	poolBytes := entryValue(posReadResp, qPool)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}
	if dErr := debitPoolAmount(pool, msg.Quantity); dErr != nil {
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
	if cErr := creditAccountAmount(msg.Address, account, msg.Quantity); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	pos.CollateralQuantity = newCollateralQty

	// [FIX, session finding] Custody (account credit + pool debit/delete)
	// and the borrower's position used to commit via TWO independent
	// StateWrite calls -- the same non-atomicity bug class found and fixed
	// this session in liquidate_position.go, borrow.go, and repay.go. Per
	// the Canopy builder docs' canonical pattern (batch-read, batch-write,
	// ONE StateWrite call is atomic -- no cross-call guarantee exists), a
	// failure in the second write used to leave the withdrawer already
	// credited with real funds while pos.CollateralQuantity still showed
	// the old, larger amount -- overstating remaining collateral and
	// potentially permitting a further borrow or passing an HF check
	// against collateral no longer actually held. Now one bundled sets/
	// deletes pair, one StateWrite call for the whole transaction.
	sets := []*PluginSetOp{{Key: acctKey, Value: acctBytesOut}}
	var deletes []*PluginDeleteOp
	if deletePool {
		deletes = append(deletes, &PluginDeleteOp{Key: poolKey})
	} else {
		sets = append(sets, &PluginSetOp{Key: poolKey, Value: poolBytesOut})
	}

	// [Matches repay.go's dual-zero-delete convention] Only delete the
	// record when BOTH collateral and debt are zero; otherwise always
	// re-save. Since currentDebt > 0 already forced a real HF check above
	// (which would have rejected any withdrawal down to zero collateral
	// against open debt, as an HF of zero collateral / positive debt is
	// definitionally liquidatable), reaching newCollateralQty == 0 here
	// with currentDebt > 0 should be unreachable -- this guard exists for
	// the same defense-in-depth reason repay.go's mirror check does.
	if newCollateralQty == 0 && currentDebt == 0 {
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
	return &PluginDeliverResponse{}
}
