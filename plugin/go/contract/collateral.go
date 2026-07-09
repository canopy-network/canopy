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

	// Batched read: Market (existence check only) and this address's
	// existing BorrowerPosition ({17}, may not exist yet -- first collateral
	// deposit for this address in this market). One round trip, matching
	// deposit.go's pattern.
	const (
		qMarket = iota
		qBorrowerPos
	)
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qMarket, Key: marketKey},
			{QueryId: qBorrowerPos, Key: borrowerPosKey},
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
	// Market is read only to confirm existence -- no field of it is
	// consulted or mutated by this transaction (see admission-gate note
	// above: none of Market's flags gate deposit_collateral).

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

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: borrowerPosKey, Value: borrowerPosBytesOut},
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
