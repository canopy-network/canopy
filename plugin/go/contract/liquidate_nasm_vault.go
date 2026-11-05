package contract

import (
	"bytes"
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"
)

// NusdOraclePriceScaled is NUSD's fixed USD price, expressed at the same
// x1e8 scale ARCM's real oracle prices use ({19} PriceRecord). NUSD is not
// an oracle-priced asset -- per NASM Consolidated Spec Section 1.2 ("Peg
// denomination: USD... matches recognizable precedent") and Section 2.2's
// HF_n formula, NUSD represents the USD reference itself, 1:1 by protocol
// definition. mint_nusd.go and burn_nusd.go both already encode this same
// assumption structurally (no ResolvePrice call for NUSD anywhere in
// either file) -- this constant makes that same fixed-$1.00 assumption
// explicit and reusable here, rather than re-deriving it inline.
//
// This is deliberately a constant, not a state read: unlike a real oracle
// asset, there is no PriceRecord for NUSD at {19} to become stale, and none
// should ever be written -- NUSD's price is definitional, not observed.
var NusdOraclePriceScaled uint64 = 1_00000000 // $1.00 at x1e8

// CheckMessageLiquidateNasmVault statelessly validates a
// 'liquidate_nasm_vault' message (NASM Consolidated Spec Section 7). No
// state reads here -- vault existence/ownership, current debt, oracle
// price, and HF_n all require state, so they run at DeliverTx, matching
// liquidate_position.go's own CheckMessageLiquidatePosition split exactly.
func (c *Contract) CheckMessageLiquidateNasmVault(msg *MessageLiquidateNasmVault) *PluginCheckResponse {
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidVaultID(err)}
	}
	if len(msg.Liquidator) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.RepayAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Liquidator}}
}

// DeliverMessageLiquidateNasmVault handles 'liquidate_nasm_vault' per NASM
// Consolidated Spec Section 7 and MessageLiquidateNasmVault's own proto doc
// comment (arbor.proto): reuses ARCM's ComputeHealthFactorScaled and
// CloseFactorBpsForHF UNCHANGED, substitutes ScaledNusdDebt for ScaledDebt,
// and substitutes NASM's own tighter tier params for ARCM's lending tier
// table. The collateral-seizure formula itself is ARCM's own, unmodified,
// from liquidate_position.go's Step 4 -- the only substitution within it is
// NUSD's debt-side price, which is NusdOraclePriceScaled (a fixed $1.00
// constant) rather than a ResolvePrice lookup, since NUSD has no {19} price
// record and is USD-pegged 1:1 by protocol definition (Section 1.2).
//
// repay_amount is debited from the LIQUIDATOR's own NusdBalance (NOT
// Account.Amount), mirroring burn_nusd.go's custody split exactly --
// liquidating a vault burns NUSD out of circulation via NusdSupply, the
// same as any other debt reduction (proto doc comment, arbor.proto).
//
// PHASE 1 SCOPE LIMIT, DISCLOSED (see ErrNasmLiquidationBadDebt's own doc
// comment, error.go): a liquidation whose required collateral seizure would
// exceed the vault's own locked collateral_quantity is hard-rejected rather
// than partially seizing and leaving an unaccounted shortfall. NASM's own
// bad-debt waterfall (R_nusd draw-down, Arbor treasury fallback, Spec
// Section 11.2) is not yet built -- no R_nusd accessor exists anywhere in
// this codebase as of this writing (confirmed: no {33} key, no
// GetNusdReserve/SetNusdReserve function). This function does NOT emit a
// WaterfallEvent for that reason: there is no covered waterfall step here
// to log, only a hard rejection on the uncovered path. It emits
// EventNasmVaultLiquidated instead, reporting the liquidation itself.
func (c *Contract) DeliverMessageLiquidateNasmVault(msg *MessageLiquidateNasmVault, fee uint64) *PluginDeliverResponse {
	var nasmWaterfallSets []*PluginSetOp
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidVaultID(err)}
	}
	if len(msg.Liquidator) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.RepayAmount == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}

	// Step 1 -- vault must exist. Ownership is read from the vault record
	// itself (vault.Owner), NOT from any message field -- NASM vault
	// ownership is mutable (Spec Section 5.2's transferability), unlike
	// ARCM's BorrowerPosition, which liquidate_position.go addresses via
	// msg.BorrowerAddress directly. The self-liquidation guard below
	// therefore must also compare against vault.Owner, read fresh from
	// state, not any liquidator-supplied claim about who owns it.
	vault, vaultFound, vErr := GetNasmVault(c, msg.VaultId)
	if vErr != nil {
		return &PluginDeliverResponse{Error: vErr}
	}
	if !vaultFound {
		return &PluginDeliverResponse{Error: ErrNasmVaultNotFound(msg.VaultId)}
	}

	// [Self-liquidation guard] Mirrors liquidate_position.go's own
	// ErrSelfLiquidation guard exactly, using ErrSelfLiquidationNasm
	// (error.go). Without this, a vault owner could liquidate their own
	// underwater vault and collect the LIF bonus risk-free -- the owner
	// already has burn_nusd for closing their own position without one.
	if bytes.Equal(msg.Liquidator, vault.Owner) {
		return &PluginDeliverResponse{Error: ErrSelfLiquidationNasm(msg.VaultId)}
	}

	// Step 2 -- tier params. Re-resolved here (not trusted from mint time)
	// since ARCM's {29} registry is the single source of truth and could
	// have changed since this vault was opened (nasm_tier.go's own doc
	// comment on why NASM never caches tier classification locally).
	_, nasmParams, tierFound, tErr := ResolveNasmTier(c, vault.CollateralAssetId)
	if tErr != nil {
		return &PluginDeliverResponse{Error: tErr}
	}
	if !tierFound {
		return &PluginDeliverResponse{Error: ErrNasmTierIneligible(vault.CollateralAssetId)}
	}

	// Batched read: liquidator's NusdBalance, liquidator's Account (to
	// receive seized collateral), the vault's own collateral escrow Pool,
	// NusdSupply, and the current SF_index -- matches burn_nusd.go's own
	// batched-read shape exactly, one round-trip for every value this
	// function needs.
	liquidatorNusdBalKey := KeyForNusdBalance(msg.Liquidator)
	liquidatorAcctKey := KeyForAccount(msg.Liquidator)
	poolId := KeyForMarketPoolId(msg.VaultId, PoolPurposeNasmVault)
	poolKey := KeyForFeePool(poolId)

	const (
		qLiquidatorNusdBal = iota
		qLiquidatorAcct
		qPool
		qSupply
		qSfIndex
	)
	readResp, rErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qLiquidatorNusdBal, Key: liquidatorNusdBalKey},
			{QueryId: qLiquidatorAcct, Key: liquidatorAcctKey},
			{QueryId: qPool, Key: poolKey},
			{QueryId: qSupply, Key: KeyForNusdSupply()},
			{QueryId: qSfIndex, Key: KeyForStabilityFeeIndex()},
		},
	})
	if rErr != nil {
		return &PluginDeliverResponse{Error: rErr}
	}
	if readResp.Error != nil {
		return &PluginDeliverResponse{Error: readResp.Error}
	}

	// Step 3 -- current SF_index(t), same found=false-defaults-to-RAY
	// convention as burn_nusd.go and mint_nusd.go.
	var sfIndexNow *big.Int
	sfIndexBytes := entryValue(readResp, qSfIndex)
	if len(sfIndexBytes) == 0 {
		sfIndexNow = RAY
	} else {
		sf := &StabilityFeeIndex{}
		if uErr := Unmarshal(sfIndexBytes, sf); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		sfIndexNow = DecodeUint128(sf.SfIndex)
	}

	currentDebt, sdErr := ScaledNusdDebt(vault, sfIndexNow)
	if sdErr != nil {
		return &PluginDeliverResponse{Error: sdErr}
	}
	if currentDebt == 0 {
		return &PluginDeliverResponse{Error: ErrNasmVaultNotLiquidatable(msg.VaultId, "n/a (zero debt)")}
	}

	// Oracle price for the COLLATERAL asset only -- the shared {19} cache,
	// ARCM Section 10, NASM Spec Section 7's "Fully shared" integration.
	// NUSD's own "price" is NusdOraclePriceScaled (fixed, no lookup) --
	// see that constant's own doc comment above.
	collateralPrice, priceFound, pErr := ResolvePrice(c, vault.CollateralAssetId)
	if pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}
	if !priceFound {
		return &PluginDeliverResponse{Error: ErrNasmPriceUnavailable(msg.VaultId, vault.CollateralAssetId)}
	}

	// [NEW] NASM Spec Section 9.2. Different rationale than burn_nusd's own
	// identical check (that file's own comment): liquidation isn't gated
	// here as a punitive pause -- it's gated because HF_n and the seizure
	// amount CANNOT be safely computed against a price this untrustworthy,
	// regardless of whether liquidation itself would otherwise be
	// desirable during an emergency (protecting the protocol from further
	// bad debt). An untrustworthy price makes the underlying math itself
	// unsound, not merely risky.
	untrustworthy, ouErr := OracleUntrustworthy(c, vault.CollateralAssetId)
	if ouErr != nil {
		return &PluginDeliverResponse{Error: ouErr}
	}
	if untrustworthy {
		return &PluginDeliverResponse{Error: ErrOracleUntrustworthy(msg.VaultId, vault.CollateralAssetId)}
	}

	// Step 4 -- HF_n, via ARCM's own ComputeHealthFactorScaled, UNCHANGED,
	// per MessageLiquidateNasmVault's own proto doc comment. debt price is
	// NusdOraclePriceScaled (fixed $1.00), not a lookup.
	hfScaled := ComputeHealthFactorScaled(vault.CollateralQuantity, collateralPrice, nasmParams.LTVLiqBps, currentDebt, NusdOraclePriceScaled)
	if hfScaled.Cmp(HFLiquidatableThresholdScaled) > 0 {
		return &PluginDeliverResponse{Error: ErrNasmVaultNotLiquidatable(msg.VaultId, hfScaled.String())}
	}

	// Step 5 -- dynamic close factor, via ARCM's own CloseFactorBpsForHF,
	// UNCHANGED, per the same proto doc comment.
	closeFactorBps := CloseFactorBpsForHF(hfScaled)
	maxRepay := new(big.Int).Mul(new(big.Int).SetUint64(currentDebt), big.NewInt(int64(closeFactorBps)))
	maxRepay.Div(maxRepay, big.NewInt(10_000))
	maxRepayU64 := maxRepay.Uint64()
	if msg.RepayAmount > maxRepayU64 {
		return &PluginDeliverResponse{Error: ErrRepayExceedsCloseFactorNasm(msg.VaultId, msg.RepayAmount, maxRepayU64)}
	}

	// Step 6 -- collateral seized = ceil(repay * P_d * LIF / P_c), ARCM's
	// own unmodified seizure formula from liquidate_position.go's Step 4,
	// with P_d = NusdOraclePriceScaled (fixed) in place of a ResolvePrice
	// lookup for the debt asset.
	seizeNum := new(big.Int).Mul(new(big.Int).SetUint64(msg.RepayAmount), new(big.Int).SetUint64(NusdOraclePriceScaled))
	seizeNum.Mul(seizeNum, big.NewInt(int64(nasmParams.LIFBps)))
	seizeDen := new(big.Int).Mul(new(big.Int).SetUint64(collateralPrice), big.NewInt(10_000))
	// Ceiling division.
	seizeNum.Add(seizeNum, seizeDen)
	seizeNum.Sub(seizeNum, big.NewInt(1))
	collateralSeized := new(big.Int).Div(seizeNum, seizeDen)

	if collateralSeized.BitLen() > 64 {
		return &PluginDeliverResponse{Error: ErrCollateralSeizedOverflow(msg.VaultId, msg.RepayAmount, collateralSeized.String())}
	}
	collateralSeizedU64 := collateralSeized.Uint64()

	// Step 7 -- bad-debt path. PHASE 1 SCOPE LIMIT lifted: Layer3DrawDownNASM
	// (bad_debt_layer3.go) already existed, reading/writing R_nusd at {41}
	// via GetTreasuryNASM/SetTreasuryNASM -- this function's own prior doc
	// comment claiming "no R_nusd accessor exists anywhere in this codebase"
	// predates cross-referencing bad_debt_layer3.go and was incorrect as of
	// commit 80d68889; corrected here rather than carried forward.
	//
	// Mirrors liquidate_position.go's own Layer 2/3 badDebtNative derivation
	// exactly: raw prices, deliberately WITHOUT LIFBps (LIF is the
	// liquidator's incentive markup already applied in Step 6's seizure
	// math; R_nusd makes the protocol whole for actual value lost, not the
	// incentivized amount). Ceiling-rounded, protocol-favouring.
	if collateralSeizedU64 > vault.CollateralQuantity {
		badDebtCollateral := new(big.Int).Sub(
			new(big.Int).SetUint64(collateralSeizedU64),
			new(big.Int).SetUint64(vault.CollateralQuantity),
		)
		badDebtNativeNum := new(big.Int).Mul(badDebtCollateral, new(big.Int).SetUint64(collateralPrice))
		badDebtNativeDen := new(big.Int).SetUint64(NusdOraclePriceScaled)
		badDebtNativeNum.Add(badDebtNativeNum, badDebtNativeDen)
		badDebtNativeNum.Sub(badDebtNativeNum, big.NewInt(1))
		badDebtNative := new(big.Int).Div(badDebtNativeNum, badDebtNativeDen)

		// Same 64-bit overflow guard as liquidate_position.go's own
		// badDebtNative check, applied before use in Layer3DrawDownNASM.
		if badDebtNative.BitLen() > 64 {
			return &PluginDeliverResponse{Error: ErrBadDebtNativeOverflow(msg.VaultId, badDebtCollateral.String(), badDebtNative.String())}
		}
		badDebtNativeU64 := badDebtNative.Uint64()

		covered, newTFund, l3Err := Layer3DrawDownNASM(c, msg.VaultId, badDebtNativeU64)
		if l3Err != nil {
			return &PluginDeliverResponse{Error: l3Err}
		}
		if !covered {
			// R_nusd insufficient to fully cover -- all-or-nothing gate, same
			// contract as Layer3DrawDownArbor/Layer2DrawDown. No partial
			// seizure; original hard-reject remains the correct fallback
			// (NASM Spec Section 11.2 -- Layer 4 not yet built for NASM).
			return &PluginDeliverResponse{Error: ErrNasmLiquidationBadDebt(msg.VaultId, collateralSeizedU64, vault.CollateralQuantity)}
		}

		// Covered. Log a WaterfallEvent ({42}) for this NASM Layer 3
		// draw-down -- same durable-log pattern this session added for
		// ARCM's own waterfall. seq=0: NASM has only one waterfall layer
		// wired so far (unlike ARCM's 0/1/2 for Layer2/3/4), so no
		// collision risk within a single liquidation.
		wfSetOp, wfErr := BuildWaterfallEventSetOp(c.plugin.CurrentHeight(), 0, &WaterfallEvent{
			BlockHeight:      c.plugin.CurrentHeight(),
			MarketId:         msg.VaultId,
			Layer:            "layer3_nasm",
			EventType:        "nasm_treasury_draw_down",
			BadDebt:          badDebtNativeU64,
			RemainingBalance: newTFund.String(),
		})
		if wfErr != nil {
			return &PluginDeliverResponse{Error: wfErr}
		}
		nasmWaterfallSets = append(nasmWaterfallSets, wfSetOp)

		// Layer 3 fully covered the shortfall's value. Cap seizure at what
		// the vault actually has -- the liquidator receives all available
		// collateral; R_nusd makes the protocol whole for the rest. Mirrors
		// liquidate_position.go's identical post-waterfall reassignment.
		collateralSeizedU64 = vault.CollateralQuantity
	}

	// Step 8 -- custody. Liquidator's NusdBalance is debited by
	// RepayAmount (burns NUSD, mirrors burn_nusd.go's custody split, per
	// the proto doc comment's own explicit mandate); NusdSupply.TotalSupply
	// decreases by the same amount; the vault's own collateral escrow Pool
	// is debited by collateralSeizedU64; the liquidator's real Account is
	// credited with the seized collateral.
	liquidatorNusdBalBytes := entryValue(readResp, qLiquidatorNusdBal)
	liquidatorNusdBal := &NusdBalance{Address: msg.Liquidator}
	if len(liquidatorNusdBalBytes) > 0 {
		if uErr := Unmarshal(liquidatorNusdBalBytes, liquidatorNusdBal); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		liquidatorNusdBal.Address = msg.Liquidator
	}
	if msg.RepayAmount > liquidatorNusdBal.Amount {
		return &PluginDeliverResponse{Error: ErrNusdInsufficientBalance(msg.VaultId, msg.RepayAmount, liquidatorNusdBal.Amount)}
	}
	if dErr := debitNusdBalance(liquidatorNusdBal, msg.RepayAmount); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	liquidatorNusdBalBytesOut, mErr := Marshal(liquidatorNusdBal)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	supplyBytes := entryValue(readResp, qSupply)
	supply := &NusdSupply{}
	if len(supplyBytes) > 0 {
		if uErr := Unmarshal(supplyBytes, supply); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if supply.TotalSupply < msg.RepayAmount {
		// Unreachable in practice, matches burn_nusd.go's own identical
		// guard and reasoning: total_supply is the sum of every vault's
		// own debt, so a single vault's repay can never exceed it.
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	supply.TotalSupply -= msg.RepayAmount
	supplyBytesOut, mErr := Marshal(supply)
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
	if dErr := debitPoolAmount(pool, collateralSeizedU64); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	// Pool drained to zero is deleted, matching burn_nusd.go's own
	// [FIX]-annotated convention.
	var poolBytesOut []byte
	var deletePool bool
	if pool.Amount == 0 {
		deletePool = true
	} else {
		var pmErr *PluginError
		poolBytesOut, pmErr = Marshal(pool)
		if pmErr != nil {
			return &PluginDeliverResponse{Error: pmErr}
		}
	}

	liquidatorAcctBytes := entryValue(readResp, qLiquidatorAcct)
	liquidatorAcct := &Account{}
	if len(liquidatorAcctBytes) > 0 {
		if uErr := Unmarshal(liquidatorAcctBytes, liquidatorAcct); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if cErr := creditAccountAmount(msg.Liquidator, liquidatorAcct, collateralSeizedU64); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	liquidatorAcctBytesOut, mErr := Marshal(liquidatorAcct)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Step 9 -- update vault: reduce debt and collateral, re-snapshot
	// sf_index_at_open (mirrors burn_nusd.go's Steps 9/10 exactly). Full
	// closure (newDebt == 0) deletes the vault record entirely.
	newDebt := currentDebt - msg.RepayAmount
	newCollateral := vault.CollateralQuantity - collateralSeizedU64
	vaultClosed := newDebt == 0

	// [NEW] NASM Spec Section 3.3: decrement the tier-backing accumulator
	// by msg.RepayAmount (the same figure NusdSupply.total_supply is
	// decremented by, directly below), for the SAME tier this vault was
	// minted into -- vault.NasmTier, mirroring burn_nusd.go's identical
	// decrement exactly (see that file's own comment for the full
	// rationale on why this reads the snapshotted tier, not a live
	// re-resolution).
	tierBacking, tbFound, tbErr := GetNasmTierBacking(c)
	if tbErr != nil {
		return &PluginDeliverResponse{Error: tbErr}
	}
	if !tbFound {
		tierBacking = &NasmTierBacking{}
	}
	repayAmountI64, tbOk := SafeInt64FromUint64(msg.RepayAmount)
	if !tbOk {
		return &PluginDeliverResponse{Error: ErrNasmTierBackingOverflow(uint8(vault.NasmTier), 0, msg.RepayAmount)}
	}
	if _, tbdErr := applyTierBackingDelta(tierBacking, uint8(vault.NasmTier), -repayAmountI64); tbdErr != nil {
		return &PluginDeliverResponse{Error: tbdErr}
	}
	tierBackingBytesOut, mErr := Marshal(tierBacking)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	sets := []*PluginSetOp{
		{Key: liquidatorNusdBalKey, Value: liquidatorNusdBalBytesOut},
		{Key: KeyForNusdSupply(), Value: supplyBytesOut},
		{Key: liquidatorAcctKey, Value: liquidatorAcctBytesOut},
		{Key: KeyForNasmTierBacking(), Value: tierBackingBytesOut},
	}
	var deletes []*PluginDeleteOp
	if deletePool {
		deletes = append(deletes, &PluginDeleteOp{Key: poolKey})
	} else {
		sets = append(sets, &PluginSetOp{Key: poolKey, Value: poolBytesOut})
	}

	if vaultClosed {
		deletes = append(deletes, &PluginDeleteOp{Key: KeyForNasmVault(msg.VaultId)})
	} else {
		sfIndexEncoded, encErr := EncodeUint128(sfIndexNow)
		if encErr != nil {
			return &PluginDeliverResponse{Error: encErr}
		}
		vault.NusdPrincipal = newDebt
		vault.CollateralQuantity = newCollateral
		vault.SfIndexAtOpen = sfIndexEncoded
		vaultBytesOut, mErr := Marshal(vault)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
		sets = append(sets, &PluginSetOp{Key: KeyForNasmVault(msg.VaultId), Value: vaultBytesOut})
	}

	// Event -- EventNasmVaultLiquidated (arbor_events.proto), registered in
	// contract.go's EventTypeUrls. Reports the liquidation itself, not a
	// waterfall step (see this function's own doc comment above).
	eventPayload := &EventNasmVaultLiquidated{
		VaultId:            msg.VaultId,
		Liquidator:         msg.Liquidator,
		RepayAmount:        msg.RepayAmount,
		CollateralSeized:   collateralSeizedU64,
		RemainingVaultDebt: newDebt,
		VaultClosed:        vaultClosed,
	}
	anyMsg, aErr := anypb.New(eventPayload)
	if aErr != nil {
		return &PluginDeliverResponse{Error: ErrMarshal(aErr)}
	}
	events := []*Event{
		{
			EventType: "nasm_vault_liquidated",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		},
	}

	// Step 10 -- single atomic StateWrite, matching burn_nusd.go's own
	// discipline.
	sets = append(sets, nasmWaterfallSets...)
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets, Deletes: deletes})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}

	_ = fee
	return &PluginDeliverResponse{Events: events}
}
