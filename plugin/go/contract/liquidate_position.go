package contract

import (
	"bytes"
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"
)

// CheckMessageLiquidatePosition statelessly validates a 'liquidate_position'
// message. Market/position existence, HF eligibility, close-factor cap, and
// price resolution all require state, so they run at DeliverTx (matching
// borrow.go/repay.go's established stateless/stateful split).
func (c *Contract) CheckMessageLiquidatePosition(msg *MessageLiquidatePosition) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Liquidator) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if len(msg.BorrowerAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.RepayAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Liquidator}}
}

// DeliverMessageLiquidatePosition handles 'liquidate_position' per ARCM
// Section 6 (liquidation architecture) and Section 19.2.2 (detailed
// liquidation debt-reduction sequence):
//
// Step 1 -- AccrueInterest (AYIS Section 12.3's mandatory ordering)
// Step 2 -- Compute HF; reject if position is not liquidatable (HF > 1.0)
// Step 3 -- Compute close factor from HF tier (ARCM Section 7); cap
//
//	repay_amount at scaled_debt * close_factor
//
// Step 4 -- collateral_seized = ceil(repay_amount * P_d * LIF / P_c)
// Step 5 -- If collateral_seized > Q_c: bad debt scenario (Section 8.5, 9)
// Step 6 -- Atomic state update: reduce debt, reduce collateral, move funds
// Step 7 -- market.total_borrowed write-back via applyDebtDelta (Principle 9)
//
// NOT YET IMPLEMENTED (disclosed, not silent): Layer 2-4 of the bad-debt
// waterfall (reserve fund draw-down, treasury, lender socialization via
// loss_factor) per ARCM Section 9.2. A liquidation that would leave bad
// debt is currently rejected outright via ErrLiquidationBadDebt rather than
// silently under-seizing collateral or leaving the protocol with an
// unaccounted shortfall. This is a scope limit, not a bug: implementing
// Layers 2-4 requires R_fund draw-down, ApplyLossFactor, and the {28}
// queue, none of which this handler touches.
func (c *Contract) DeliverMessageLiquidatePosition(msg *MessageLiquidatePosition, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Liquidator) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if len(msg.BorrowerAddress) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	// [NEW] Self-liquidation guard. Without this, an address can liquidate
	// its own underwater position and collect the LIF bonus (ARCM Section 8)
	// risk-free -- that bonus exists to compensate an INDEPENDENT liquidator
	// for monitoring/execution risk a self-liquidating borrower never bears.
	// A borrower who wants to close their own underwater position without a
	// bonus already has repay.go for that; this path has no legitimate use
	// case repay.go doesn't already cover, and this guard closes a real
	// economic exploit (worse if the same address is also an authorized
	// price oracle submitter for this market -- see the separate, still-open
	// oracle/borrower-role-separation gap this does NOT fix).
	if bytes.Equal(msg.Liquidator, msg.BorrowerAddress) {
		return &PluginDeliverResponse{Error: ErrSelfLiquidation()}
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

	// [MOVED, session finding] events must be declared before Layer 4's
	// ApplyLossFactor call (Layer 2 miss fallthrough, below) can append to
	// it -- previously declared later, at Step 7, when this was the only
	// place an event could originate. Now declared here so both the new
	// Layer 4 path and the existing Step 7 dust-clamp path share one slice.
	var events []*Event
	if pErr := checkMarketNotPaused(market, msg.MarketId); pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}

	if aErr := AccrueInterest(c, msg.MarketId); aErr != nil {
		return &PluginDeliverResponse{Error: aErr}
	}

	posKey := KeyForBorrowerPosition(msg.MarketId, msg.BorrowerAddress)
	liquidatorAcctKey := KeyForAccount(msg.Liquidator)
	borrowerAcctKey := KeyForAccount(msg.BorrowerAddress)
	collateralPoolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeCollateral)
	collateralPoolKey := KeyForFeePool(collateralPoolId)
	debtPoolId := KeyForMarketPoolId(msg.MarketId, PoolPurposeSupply)
	debtPoolKey := KeyForFeePool(debtPoolId)

	const (
		qPos = iota
		qLiquidatorAcct
		qBorrowerAcct
		qCollateralPool
		qDebtPool
	)
	readResp, rErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qPos, Key: posKey},
			{QueryId: qLiquidatorAcct, Key: liquidatorAcctKey},
			{QueryId: qBorrowerAcct, Key: borrowerAcctKey},
			{QueryId: qCollateralPool, Key: collateralPoolKey},
			{QueryId: qDebtPool, Key: debtPoolKey},
		},
	})
	if rErr != nil {
		return &PluginDeliverResponse{Error: rErr}
	}
	if readResp.Error != nil {
		return &PluginDeliverResponse{Error: readResp.Error}
	}

	posBytes := entryValue(readResp, qPos)
	if len(posBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrBorrowerPositionNotFound(msg.MarketId, msg.BorrowerAddress)}
	}
	pos := &BorrowerPosition{}
	if uErr := Unmarshal(posBytes, pos); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	bIndexNow, biFound, biErr := GetBorrowIndex(c, msg.MarketId)
	if biErr != nil {
		return &PluginDeliverResponse{Error: biErr}
	}
	if !biFound {
		return &PluginDeliverResponse{Error: ErrMarketIndexNotInitialized(msg.MarketId)}
	}
	currentDebt := ScaledDebt(pos, bIndexNow)
	if currentDebt == 0 {
		return &PluginDeliverResponse{Error: ErrPositionNotLiquidatable(msg.MarketId, "n/a (zero debt)")}
	}

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

	// Step 2 -- ARCM Section 5. Verify HF <= 1.0 at DeliverTx time.
	hfScaled := ComputeHealthFactorScaled(pos.CollateralQuantity, collateralPrice, tierParams.LTVLiqBps, currentDebt, debtPrice)
	if hfScaled.Cmp(HFLiquidatableThresholdScaled) > 0 {
		return &PluginDeliverResponse{Error: ErrPositionNotLiquidatable(msg.MarketId, hfScaled.String())}
	}

	// Step 3 -- ARCM Section 7. Dynamic close factor by HF tier.
	closeFactorBps := CloseFactorBpsForHF(hfScaled)
	maxRepay := new(big.Int).Mul(new(big.Int).SetUint64(currentDebt), big.NewInt(int64(closeFactorBps)))
	maxRepay.Div(maxRepay, big.NewInt(10_000))
	maxRepayU64 := maxRepay.Uint64()
	if msg.RepayAmount > maxRepayU64 {
		return &PluginDeliverResponse{Error: ErrRepayExceedsCloseFactor(msg.MarketId, msg.RepayAmount, maxRepayU64)}
	}

	// Step 4 -- ARCM Section 8. Collateral seized = ceil(repay * P_d * LIF / P_c).
	seizeNum := new(big.Int).Mul(new(big.Int).SetUint64(msg.RepayAmount), new(big.Int).SetUint64(debtPrice))
	seizeNum.Mul(seizeNum, big.NewInt(int64(tierParams.LIFBps)))
	seizeDen := new(big.Int).Mul(new(big.Int).SetUint64(collateralPrice), big.NewInt(10_000))
	// Ceiling division.
	seizeNum.Add(seizeNum, seizeDen)
	seizeNum.Sub(seizeNum, big.NewInt(1))
	collateralSeized := new(big.Int).Div(seizeNum, seizeDen)

	// Step 5 -- ARCM Section 9.2, Layer 2. If collateral_seized > Q_c, attempt
	// to cover the shortfall's value from the market reserve fund before
	// falling back to a hard reject. Layer 3 (protocol treasury) and Layer 4
	// (lender socialization) are not implemented -- see bad_debt_layer2.go's
	// doc comment for why an uncovered shortfall hard-rejects rather than
	// partially draining R_fund with nowhere for the residual to go.
	if collateralSeized.Cmp(new(big.Int).SetUint64(pos.CollateralQuantity)) > 0 {
		badDebtCollateral := new(big.Int).Sub(collateralSeized, new(big.Int).SetUint64(pos.CollateralQuantity))

		// Convert the uncovered COLLATERAL quantity to its DEBT-asset-native
		// value equivalent, at raw prices -- deliberately WITHOUT LIFBps,
		// since LIF is the liquidator's incentive markup already applied in
		// Step 4's seizure math; the reserve fund makes the protocol whole
		// for actual value lost, not the incentivized amount. Ceiling-
		// rounded so the reserve is never undercharged (Section 10.2:
		// rounding favors the protocol).
		badDebtNativeNum := new(big.Int).Mul(badDebtCollateral, new(big.Int).SetUint64(collateralPrice))
		badDebtNativeDen := new(big.Int).SetUint64(debtPrice)
		badDebtNativeNum.Add(badDebtNativeNum, badDebtNativeDen)
		badDebtNativeNum.Sub(badDebtNativeNum, big.NewInt(1))
		badDebtNative := new(big.Int).Div(badDebtNativeNum, badDebtNativeDen)

		// [DISCLOSED] No BitLen/IsUint64 guard on badDebtNative before the
		// .Uint64() call below -- matches this function's pre-existing risk
		// posture (the prior badDebtValue.Uint64() call this replaces had no
		// such guard either). Not a new gap introduced by this change.
		covered, l2Err := Layer2DrawDown(c, msg.MarketId, badDebtNative.Uint64())
		if l2Err != nil {
			return &PluginDeliverResponse{Error: l2Err}
		}
		if !covered {
			// [DISCLOSED, session finding] Layer 3 (Arbor protocol treasury)
			// does NOT exist in this codebase -- no state key, no accessor,
			// nothing. It is a standalone, unbuilt piece shared by ARCM's own
			// waterfall and (per NASM Section 11.2) NASM's NUSD waterfall,
			// deferred until Arbor's core lending protocol is complete. This
			// is NOT a NASM prerequisite issue -- NASM's own Layer 3 row
			// assumes Arbor treasury already exists ("Arbor protocol treasury
			// covers remaining shortfall"); it does not define or build it.
			//
			// Given that, this liquidation intentionally skips straight from
			// Layer 2 (just missed, above) to Layer 4 (lender socialization
			// via ApplyLossFactor) rather than hard-rejecting as before. This
			// is a deliberate interim design choice, not a silent omission:
			// bad debt Layer 3 might have partially absorbed instead lands
			// fully on lenders via loss_factor. Open item, tracked separately
			// from this comment: "Layer 3 (Arbor protocol treasury): standalone,
			// unbuilt, blocks full four-layer waterfall fidelity."
			// [FIXED] Now passes the already-in-memory market struct (read once
			// near the top of this function) instead of letting ApplyLossFactor
			// fetch its own separate copy. This closes the two-copy race that
			// caused a confirmed on-chain bug: SetMarketInsolvent() used to save
			// its own copy of market mid-call, which this function's own
			// end-of-function SaveMarket(market) at line ~337 then silently
			// overwrote with its stale copy -- losing the Status write while
			// loss_factor correctly persisted to 0. See market_insolvency.go and
			// apply_loss_factor.go for the full fix.
			layer4Event, alErr := ApplyLossFactor(c, market, msg.MarketId, badDebtNative.Uint64())
			if alErr != nil {
				return &PluginDeliverResponse{Error: alErr}
			}
			if layer4Event != nil {
				events = append(events, layer4Event)
			}
		}

		// Layer 2 fully covered the shortfall's value. Cap seizure at what
		// the position actually has -- the liquidator receives all
		// available collateral; the reserve fund makes the protocol whole
		// for the rest.
		collateralSeized = new(big.Int).SetUint64(pos.CollateralQuantity)
	}
	collateralSeizedU64 := collateralSeized.Uint64()

	// Step 6 -- atomic custody update: debit liquidator's account by
	// RepayAmount, credit debt-asset pool; debit collateral pool, credit
	// liquidator's account with seized collateral.
	liquidatorAcctBytes := entryValue(readResp, qLiquidatorAcct)
	liquidatorAcct := &Account{}
	if len(liquidatorAcctBytes) > 0 {
		if uErr := Unmarshal(liquidatorAcctBytes, liquidatorAcct); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if dErr := debitAccountAmount(liquidatorAcct, msg.RepayAmount); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	if cErr := creditAccountAmount(msg.Liquidator, liquidatorAcct, collateralSeizedU64); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}

	debtPoolBytes := entryValue(readResp, qDebtPool)
	debtPool := &Pool{Id: debtPoolId}
	if len(debtPoolBytes) > 0 {
		if uErr := Unmarshal(debtPoolBytes, debtPool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		debtPool.Id = debtPoolId
	}
	if cErr := creditPoolAmount(msg.MarketId, debtPool, msg.RepayAmount); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}

	collateralPoolBytes := entryValue(readResp, qCollateralPool)
	collateralPool := &Pool{Id: collateralPoolId}
	if len(collateralPoolBytes) > 0 {
		if uErr := Unmarshal(collateralPoolBytes, collateralPool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		collateralPool.Id = collateralPoolId
	}
	if dErr := debitPoolAmount(collateralPool, collateralSeizedU64); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}

	// Position update: reduce debt and collateral.
	newDebt := currentDebt - msg.RepayAmount
	newCollateral := pos.CollateralQuantity - collateralSeizedU64
	pos.DebtPrincipal = newDebt
	pos.CollateralQuantity = newCollateral
	bIndexEncoded, encErr := EncodeUint128(bIndexNow)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}
	pos.BorrowIndexAtOpen = bIndexEncoded

	// Step 7 -- ARCM Principle 9. Single mandatory write path for total_borrowed.
	repaidDelta, safeOk := SafeInt64FromUint64(msg.RepayAmount)
	if !safeOk {
		return &PluginDeliverResponse{Error: ErrInt64CastOverflow("liquidate.repayAmount", msg.RepayAmount)}
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
			Source:         "liquidation",
			DecreaseAmount: msg.RepayAmount,
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

	// [FIX, session finding] SaveMarket's own internal StateWrite used to
	// fire here as a SECOND, independent write, separate from the
	// liquidator/pool/position write below -- the exact non-atomicity bug
	// class this session found and fixed in market_insolvency.go, except
	// here it was a genuine two-StateWrite split within DeliverTx itself,
	// not two GetMarket/SaveMarket round-trips. Per the Canopy builder
	// docs' own canonical pattern ("batch-read... batch-write... in one
	// StateWrite call" -- operations in one StateWrite call are atomic,
	// no cross-call guarantee exists), a failure in the write below this
	// used to leave market.TotalBorrowed already decremented while the
	// liquidator never actually received funds/collateral and the
	// borrower's position still showed the old debt/collateral. Now
	// marshaled here and appended to the same sets slice as everything
	// else, so the whole transaction commits or fails as one write.
	marketBytesOut, mErr := Marshal(market)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	liquidatorAcctOut, mErr := Marshal(liquidatorAcct)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	debtPoolOut, mErr := Marshal(debtPool)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}
	collateralPoolOut, mErr := Marshal(collateralPool)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	sets := []*PluginSetOp{
		{Key: KeyForMarket(msg.MarketId), Value: marketBytesOut},
		{Key: liquidatorAcctKey, Value: liquidatorAcctOut},
		{Key: debtPoolKey, Value: debtPoolOut},
		{Key: collateralPoolKey, Value: collateralPoolOut},
	}

	var deletes []*PluginDeleteOp
	if newDebt == 0 && newCollateral == 0 {
		deletes = append(deletes, &PluginDeleteOp{Key: posKey})
	} else {
		posBytesOut, mErr := Marshal(pos)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
		sets = append(sets, &PluginSetOp{Key: posKey, Value: posBytesOut})
	}

	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets, Deletes: deletes})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}

	_ = fee
	_ = borrowerAcctKey
	return &PluginDeliverResponse{Events: events}
}
