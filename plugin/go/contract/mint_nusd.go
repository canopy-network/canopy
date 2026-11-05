package contract

import "strconv"

// CheckMessageMintNusd statelessly validates a 'mint_nusd' message (NASM
// Consolidated Spec Section 4.1). No state reads here -- vault_id
// collision, tier eligibility, oracle price, and the HF_n check all
// require state, so they run at DeliverTx, matching every other handler's
// stateless/stateful split (update_price.go, create_market.go).
func (c *Contract) CheckMessageMintNusd(msg *MessageMintNusd) *PluginCheckResponse {
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidVaultID(err)}
	}
	if err := ValidateAssetID(msg.CollateralAssetId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Owner) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.CollateralQuantity == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.NusdAmountRequested == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Owner}}
}

// DeliverMessageMintNusd handles a 'mint_nusd' message per NASM
// Consolidated Spec Section 4.1's seven-step mint flow:
//
// Step 1 -- verify collateral_asset is NASM Tier N-0 or N-1 (ResolveNasmTier)
// Step 2 -- [DISCLOSED GAP] tier concentration cap -- NOT enforced yet,
//
// see this function's own comment below and MessageMintNusd's
// proto doc comment for why.
//
// Step 3 -- compute HF_n post-mint using NASM tier LTV_n_liq
// Step 4 -- reject if HF_n post-mint would be <= 1.0
// Step 5 -- snapshot SF_index(t) at vault open
// Step 6 -- mint nusd_amount_requested to sender; increase vault's
//
// nusd_principal; increase NUSD total_supply
//
// Step 7 -- atomic: all steps succeed or the entire transaction reverts
//
// Collateral custody follows deposit_collateral's established real-token-
// movement pattern (debit owner's Account, credit a Pool), using
// PoolPurposeNasmVault -- NOT PoolPurposeCollateral -- to keep NASM's
// collateral custody structurally isolated from ARCM lending's own
// collateral pools (see pool_id.go's own doc comment on why reusing
// PoolPurposeCollateral here would risk a real vault_id/market_id pool-id
// collision).
//
// NUSD issuance credits NusdBalance, NOT Account.Amount -- see
// NusdBalance's own doc comment in arbor_state.proto for why these must be
// two structurally independent balances.
func (c *Contract) DeliverMessageMintNusd(msg *MessageMintNusd, fee uint64) *PluginDeliverResponse {
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidVaultID(err)}
	}
	if err := ValidateAssetID(msg.CollateralAssetId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Owner) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.CollateralQuantity == 0 || msg.NusdAmountRequested == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}

	// Vault-ID collision check -- a NASM vault, like a Market
	// (create_market.go), must never be silently overwritten by a second
	// mint_nusd naming an already-open vault_id. A single existence read
	// suffices; the read result being non-empty is itself the check.
	_, vaultFound, vErr := GetNasmVault(c, msg.VaultId)
	if vErr != nil {
		return &PluginDeliverResponse{Error: vErr}
	}
	if vaultFound {
		return &PluginDeliverResponse{Error: ErrVaultAlreadyExists(msg.VaultId)}
	}

	// Step 1 -- NASM Spec Section 3.1 tier eligibility. found=false covers
	// all three ineligibility cases (no {29} entry, ARCM Tier 2/3, or an
	// asset structurally absent from {29} as Tier 0/1) -- see
	// ResolveNasmTier's own doc comment. nasmParams (LTV/LIF) is no longer
	// consumed directly here -- CalcMaxMintableNusd (Step 3/4, below)
	// re-resolves it internally as part of its own self-contained
	// calculation, so only tier eligibility itself is checked at this
	// call site.
	nasmTier, _, tierFound, tErr := ResolveNasmTier(c, msg.CollateralAssetId)
	if tErr != nil {
		return &PluginDeliverResponse{Error: tErr}
	}
	if !tierFound {
		return &PluginDeliverResponse{Error: ErrNasmTierIneligible(msg.CollateralAssetId)}
	}

	// Step 2 -- NASM Spec Section 3.3's per-tier mint concentration cap.
	// [GAP CLOSED] Previously disclosed as unenforced pending burn_nusd's
	// existence (see nasm_tier_backing.go's own doc comment for the full
	// dependency history) -- burn_nusd now exists, closing that dependency.
	// CheckTierConcentrationCap is read-only; the actual accumulator
	// increase (applyTierBackingDelta) happens later, after every other
	// mint guard has passed, matching this codebase's established
	// verify-before-mutate ordering.
	if ccErr := CheckTierConcentrationCap(c, nasmTier, msg.NusdAmountRequested); ccErr != nil {
		return &PluginDeliverResponse{Error: ccErr}
	}

	// Oracle price is resolved inside CalcMaxMintableNusd below (it does
	// its own ResolvePrice call) -- no separate price read needed here
	// anymore now that the HF_n math has moved into that shared function.
	// ErrNasmPriceUnavailable is still raised correctly if the price is
	// missing: CalcMaxMintableNusd returns found=false in that case,
	// handled just below.

	// Step 3/4 -- HF_n = (V_nc * LTV_n_liq) / D_nusd, NASM Spec Section 2.2.
	// [DEDUPLICATED] This math now lives in nasm_tier.go's
	// CalcMaxMintableNusd -- see that function's own doc comment for the
	// full scaling derivation (V_nc's x1e8 oracle scale vs.
	// nusd_amount_requested's x1e6 precision, floor rounding throughout).
	// CalcMaxMintableNusd does its own internal price/tier resolution, so
	// this call intentionally duplicates the ResolvePrice/ResolveNasmTier
	// reads already done above rather than threading their results
	// through as extra parameters -- keeps the shared function's
	// signature simple and independently callable (exactly what the new
	// RPC route needs) at the cost of one harmless extra state read on
	// this call path.
	maxNusdAtHF1, _, mmFound, mmErr := CalcMaxMintableNusd(c, msg.CollateralAssetId, msg.CollateralQuantity)
	if mmErr != nil {
		return &PluginDeliverResponse{Error: mmErr}
	}
	if !mmFound {
		return &PluginDeliverResponse{Error: ErrNasmPriceUnavailable(msg.VaultId, msg.CollateralAssetId)}
	}

	if maxNusdAtHF1 <= msg.NusdAmountRequested {
		return &PluginDeliverResponse{Error: ErrNasmHealthFactorTooLow(msg.VaultId, msg.NusdAmountRequested, strconv.FormatUint(maxNusdAtHF1, 10))}
	}

	// Step 5 -- snapshot SF_index(t) at vault open (NASM Spec Section 6.3).
	// found=false at genesis means SF_index has never been initialized --
	// MUST default to RAY (the multiplicative identity), never zero, per
	// GetStabilityFeeIndex's own doc comment (matches AYIS's
	// SupplyIndexRecord genesis convention, create_market.go's s_rate=RAY
	// precedent).
	sfIndexNow, sfFound, sfErr := GetStabilityFeeIndex(c)
	if sfErr != nil {
		return &PluginDeliverResponse{Error: sfErr}
	}
	if !sfFound {
		sfIndexNow = RAY
	}
	sfIndexEncoded, encErr := EncodeUint128(sfIndexNow)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}

	// Collateral custody -- debit owner's real Account, credit the NASM
	// vault's own escrow Pool, checked via custody_arith.go's shared
	// functions (debitAccountAmount fails with ErrInsufficientFunds if the
	// owner doesn't actually hold enough real balance to lock).
	acctKey := KeyForAccount(msg.Owner)
	poolId := KeyForMarketPoolId(msg.VaultId, PoolPurposeNasmVault)
	poolKey := KeyForFeePool(poolId)
	nusdBalKey := KeyForNusdBalance(msg.Owner)

	const (
		qAccount = iota
		qPool
		qNusdBal
		qSupply
	)
	custodyReadResp, cErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qAccount, Key: acctKey},
			{QueryId: qPool, Key: poolKey},
			{QueryId: qNusdBal, Key: nusdBalKey},
			{QueryId: qSupply, Key: KeyForNusdSupply()},
		},
	})
	if cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	if custodyReadResp.Error != nil {
		return &PluginDeliverResponse{Error: custodyReadResp.Error}
	}

	acctBytes := entryValue(custodyReadResp, qAccount)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if dErr := debitAccountAmount(account, msg.CollateralQuantity); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	poolBytes := entryValue(custodyReadResp, qPool)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}
	if cErr := creditPoolAmount(msg.VaultId, pool, msg.CollateralQuantity); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	poolBytesOut, mErr := Marshal(pool)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Step 6 -- NUSD issuance. Credits NusdBalance (NOT Account.Amount --
	// see NusdBalance's own doc comment), and increments the global
	// NusdSupply total_supply counter in the same atomic write.
	nusdBalBytes := entryValue(custodyReadResp, qNusdBal)
	nusdBal := &NusdBalance{Address: msg.Owner}
	if len(nusdBalBytes) > 0 {
		if uErr := Unmarshal(nusdBalBytes, nusdBal); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		nusdBal.Address = msg.Owner
	}
	if cErr := creditNusdBalance(msg.Owner, nusdBal, msg.NusdAmountRequested); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	nusdBalBytesOut, mErr := Marshal(nusdBal)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	supplyBytes := entryValue(custodyReadResp, qSupply)
	supply := &NusdSupply{}
	if len(supplyBytes) > 0 {
		if uErr := Unmarshal(supplyBytes, supply); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if msg.NusdAmountRequested > (^uint64(0) - supply.TotalSupply) {
		return &PluginDeliverResponse{Error: ErrAccountBalanceOverflow(msg.Owner, supply.TotalSupply, msg.NusdAmountRequested)}
	}
	supply.TotalSupply += msg.NusdAmountRequested
	supplyBytesOut, mErr := Marshal(supply)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Build the new NasmVault record ({30}). owner is mutable going
	// forward (vault transferability, NASM Spec Section 5.2), but starts
	// as msg.Owner at mint time.
	vault := &NasmVault{
		VaultId:            msg.VaultId,
		Owner:              msg.Owner,
		CollateralAssetId:  msg.CollateralAssetId,
		CollateralQuantity: msg.CollateralQuantity,
		NusdPrincipal:      msg.NusdAmountRequested,
		SfIndexAtOpen:      sfIndexEncoded,
		NasmTier:           uint32(nasmTier), // snapshotted at mint, see NasmVault.nasm_tier's own proto doc comment
	}
	vaultBytesOut, mErr := Marshal(vault)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// [NEW] NASM Spec Section 3.3: apply the actual tier-backing accumulator
	// increase, now that CheckTierConcentrationCap (Step 2, above) has
	// already verified this mint would not breach the cap. Read standalone
	// (not batched into custodyReadResp) -- matches GetGovernanceParams'
	// own standalone-read precedent for a small, low-frequency, single
	// global record, rather than adding a fifth query to a batch that
	// exists for the higher-frequency custody fields above it.
	tierBacking, tbFound, tbErr := GetNasmTierBacking(c)
	if tbErr != nil {
		return &PluginDeliverResponse{Error: tbErr}
	}
	if !tbFound {
		tierBacking = &NasmTierBacking{}
	}
	mintAmountI64, tbOk := SafeInt64FromUint64(msg.NusdAmountRequested)
	if !tbOk {
		return &PluginDeliverResponse{Error: ErrNasmTierBackingOverflow(nasmTier, 0, msg.NusdAmountRequested)}
	}
	if _, tbdErr := applyTierBackingDelta(tierBacking, nasmTier, mintAmountI64); tbdErr != nil {
		return &PluginDeliverResponse{Error: tbdErr}
	}
	tierBackingBytesOut, mErr := Marshal(tierBacking)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Step 7 -- single atomic StateWrite for the whole transaction, per
	// the non-atomic-split bug class already found and fixed this project
	// in liquidate_position.go/borrow.go (see borrow.go's own comment on
	// this exact discipline).
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForNasmVault(msg.VaultId), Value: vaultBytesOut},
			{Key: acctKey, Value: acctBytesOut},
			{Key: poolKey, Value: poolBytesOut},
			{Key: nusdBalKey, Value: nusdBalBytesOut},
			{Key: KeyForNusdSupply(), Value: supplyBytesOut},
			{Key: KeyForNasmTierBacking(), Value: tierBackingBytesOut},
		},
	})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}
