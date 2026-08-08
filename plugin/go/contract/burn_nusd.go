package contract

import (
	"bytes"
	"math/big"
)

// CheckMessageBurnNusd statelessly validates a 'burn_nusd' message (NASM
// Consolidated Spec Section 4.2). No state reads here -- vault existence,
// ownership, current debt, oracle price, and balance checks all require
// state, so they run at DeliverTx, matching mint_nusd.go's own split.
func (c *Contract) CheckMessageBurnNusd(msg *MessageBurnNusd) *PluginCheckResponse {
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidVaultID(err)}
	}
	if len(msg.Sender) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.NusdAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Sender}}
}

// DeliverMessageBurnNusd handles a 'burn_nusd' message per NASM
// Consolidated Spec Section 4.2's eleven-step burn flow. Owner-scoped
// only (Step 1) -- there is no code path by which NUSD burned by holder A
// can release collateral from a vault owned by holder B (Invariant N-I8).
//
// Compare-before-subtract (Step 6): msg.NusdAmount MAY exceed the vault's
// actual current debt. Only min(msg.NusdAmount, current_debt) is ever
// actually debited from the sender's NusdBalance or subtracted from
// NusdSupply.total_supply -- this is NOT the refund-by-minting risk
// ARCM's repay.go's ErrRepayExceedsDebt was fixed to close (see
// MessageBurnNusd's own proto doc comment); it only ever debits a balance
// the sender already holds, never creates value.
//
// AccrueStabilityFee is NOT called again here -- unlike AccrueInterest's
// defensive per-market re-call in borrow.go (needed because a market
// created mid-block might not yet have been covered by that block's
// BeginBlock pass), NASM's stability fee accrual is a SINGLE global step
// that always runs in BeginBlock before any DeliverTx in the same block,
// with no per-vault "might not be covered yet" gap -- a redundant call
// here would be pure overhead, not a correctness requirement.
func (c *Contract) DeliverMessageBurnNusd(msg *MessageBurnNusd, fee uint64) *PluginDeliverResponse {
	if err := ValidateVaultID(msg.VaultId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidVaultID(err)}
	}
	if len(msg.Sender) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.NusdAmount == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}

	// Step 1 -- vault must exist and sender must be its current owner.
	vault, vaultFound, vErr := GetNasmVault(c, msg.VaultId)
	if vErr != nil {
		return &PluginDeliverResponse{Error: vErr}
	}
	if !vaultFound {
		return &PluginDeliverResponse{Error: ErrNasmVaultNotFound(msg.VaultId)}
	}
	if !bytes.Equal(vault.Owner, msg.Sender) {
		return &PluginDeliverResponse{Error: ErrNotVaultOwner(msg.VaultId)}
	}

	// Step 2 -- sender must hold >= nusd_amount. Read here (not batched
	// with the custody read below) since we need the balance value itself
	// for this explicit check, not just to mutate it later.
	nusdBalKey := KeyForNusdBalance(msg.Sender)
	acctKey := KeyForAccount(msg.Sender)
	poolId := KeyForMarketPoolId(msg.VaultId, PoolPurposeNasmVault)
	poolKey := KeyForFeePool(poolId)

	const (
		qNusdBal = iota
		qAccount
		qPool
		qSupply
		qSfIndex
	)
	readResp, rErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qNusdBal, Key: nusdBalKey},
			{QueryId: qAccount, Key: acctKey},
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

	nusdBalBytes := entryValue(readResp, qNusdBal)
	nusdBal := &NusdBalance{Address: msg.Sender}
	if len(nusdBalBytes) > 0 {
		if uErr := Unmarshal(nusdBalBytes, nusdBal); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		nusdBal.Address = msg.Sender
	}
	if msg.NusdAmount > nusdBal.Amount {
		return &PluginDeliverResponse{Error: ErrNusdInsufficientBalance(msg.VaultId, msg.NusdAmount, nusdBal.Amount)}
	}

	// Step 3/4 -- current SF_index(t). found=false at genesis defaults to
	// RAY, matching GetStabilityFeeIndex's own doc comment (in practice,
	// AccrueStabilityFee's BeginBlock step means this should never
	// actually be found=false by the time any burn_nusd runs, but the
	// same defensive default is applied here as everywhere else this
	// index is read).
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

	// Oracle price read -- shared {19} cache, ARCM Section 10, NASM Spec
	// Section 7's "Fully shared" integration.
	collateralPrice, priceFound, pErr := ResolvePrice(c, vault.CollateralAssetId)
	if pErr != nil {
		return &PluginDeliverResponse{Error: pErr}
	}
	if !priceFound {
		return &PluginDeliverResponse{Error: ErrNasmPriceUnavailable(msg.VaultId, vault.CollateralAssetId)}
	}

	// Step 6 -- compare BEFORE subtracting (NASM Spec Section 4.2 Step 6's
	// own explicit mandate, matching ARCM/AYIS's identical repay-path
	// discipline). currentDebt and msg.NusdAmount are both unsigned.
	var newDebt uint64
	var burnedAmount uint64
	fullClosure := msg.NusdAmount >= currentDebt
	if fullClosure {
		newDebt = 0
		burnedAmount = currentDebt
		// refund_amount (msg.NusdAmount - currentDebt) is simply never
		// debited in the first place -- see this function's own doc
		// comment on why this differs structurally from a refund-by-
		// minting design. Nothing further to do for the excess.
	} else {
		newDebt = currentDebt - msg.NusdAmount
		burnedAmount = msg.NusdAmount
	}

	// Step 7 -- burn burnedAmount from sender's NusdBalance; decrease
	// NusdSupply.total_supply by the same amount.
	if dErr := debitNusdBalance(nusdBal, burnedAmount); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	nusdBalBytesOut, mErr := Marshal(nusdBal)
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
	if supply.TotalSupply < burnedAmount {
		// Unreachable in practice: total_supply is the sum of every
		// vault's own nusd_principal-derived debt, so a single vault's
		// burnedAmount can never exceed the global total. Guarded
		// explicitly per this project's standard rather than assumed.
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	supply.TotalSupply -= burnedAmount
	supplyBytesOut, mErr := Marshal(supply)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Step 8 -- release collateral_released = burnedAmount / price
	// (floor, protocol-favouring, NASM Spec Section 14.1), up to the
	// vault's own locked collateral_quantity. Unit reconciliation mirrors
	// mint_nusd's own HF formula, inverted: burnedAmount is 1e6-precision
	// NUSD/USD; price is USD-per-unit x1e8; the x100 gap (1e8/1e6) is
	// folded into the same division as mint_nusd's own comment derives.
	collateralReleased := new(big.Int).SetUint64(burnedAmount)
	collateralReleased.Mul(collateralReleased, big.NewInt(100))
	collateralReleased.Div(collateralReleased, new(big.Int).SetUint64(collateralPrice))

	var collateralReleasedU64 uint64
	if collateralReleased.BitLen() > 64 {
		collateralReleasedU64 = ^uint64(0)
	} else {
		collateralReleasedU64 = collateralReleased.Uint64()
	}
	// Cap at the vault's own locked amount -- never release more than was
	// ever locked, regardless of price movement since mint (NASM Spec
	// Section 4.2 Step 8: "up to the vault's locked amount").
	if collateralReleasedU64 > vault.CollateralQuantity {
		collateralReleasedU64 = vault.CollateralQuantity
	}
	// Full closure additionally releases ALL remaining collateral (Step
	// 10), which may exceed the price-derived collateralReleasedU64 if
	// the collateral's price has risen since mint (less collateral is
	// needed to cover the same USD debt than was originally locked).
	if fullClosure {
		collateralReleasedU64 = vault.CollateralQuantity
	}

	// Collateral custody -- debit the NASM vault's own escrow Pool,
	// credit the owner's real Account, the reverse of mint_nusd's own
	// custody movement.
	poolBytes := entryValue(readResp, qPool)
	pool := &Pool{Id: poolId}
	if len(poolBytes) > 0 {
		if uErr := Unmarshal(poolBytes, pool); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
		pool.Id = poolId
	}
	if dErr := debitPoolAmount(pool, collateralReleasedU64); dErr != nil {
		return &PluginDeliverResponse{Error: dErr}
	}
	// A pool drained to zero is deleted, matching borrow.go's own
	// [FIX]-annotated convention (fsm/account.go's SetPool() behavior).
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

	acctBytes := entryValue(readResp, qAccount)
	account := &Account{}
	if len(acctBytes) > 0 {
		if uErr := Unmarshal(acctBytes, account); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	}
	if cErr := creditAccountAmount(msg.Sender, account, collateralReleasedU64); cErr != nil {
		return &PluginDeliverResponse{Error: cErr}
	}
	acctBytesOut, mErr := Marshal(account)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// [NEW] NASM Spec Section 3.3: decrement the tier-backing accumulator
	// by burnedAmount (the same SF-scaled, already-computed figure
	// NusdSupply.total_supply is decremented by, directly above) for the
	// SAME tier this vault was minted into -- vault.NasmTier, snapshotted
	// at mint_nusd time, NOT re-resolved live (see NasmVault.nasm_tier's
	// own proto doc comment for why re-resolving would risk permanent
	// accumulator drift on a post-mint asset reclassification).
	tierBacking, tbFound, tbErr := GetNasmTierBacking(c)
	if tbErr != nil {
		return &PluginDeliverResponse{Error: tbErr}
	}
	if !tbFound {
		// Unreachable in practice -- a vault cannot exist without a prior
		// mint_nusd having already created this record -- guarded explicitly
		// rather than assumed, per this project's standard.
		tierBacking = &NasmTierBacking{}
	}
	burnAmountI64, tbOk := SafeInt64FromUint64(burnedAmount)
	if !tbOk {
		return &PluginDeliverResponse{Error: ErrNasmTierBackingOverflow(uint8(vault.NasmTier), 0, burnedAmount)}
	}
	if _, tbdErr := applyTierBackingDelta(tierBacking, uint8(vault.NasmTier), -burnAmountI64); tbdErr != nil {
		return &PluginDeliverResponse{Error: tbdErr}
	}
	tierBackingBytesOut, mErr := Marshal(tierBacking)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	// Steps 9/10 -- re-snapshot sf_index_at_open, update nusd_principal
	// and collateral_quantity, or delete the vault entirely on full
	// closure.
	sets := []*PluginSetOp{
		{Key: nusdBalKey, Value: nusdBalBytesOut},
		{Key: KeyForNusdSupply(), Value: supplyBytesOut},
		{Key: acctKey, Value: acctBytesOut},
		{Key: KeyForNasmTierBacking(), Value: tierBackingBytesOut},
	}
	var deletes []*PluginDeleteOp
	if deletePool {
		deletes = append(deletes, &PluginDeleteOp{Key: poolKey})
	} else {
		sets = append(sets, &PluginSetOp{Key: poolKey, Value: poolBytesOut})
	}

	if newDebt == 0 {
		// Step 10 -- full closure: delete the vault record entirely.
		deletes = append(deletes, &PluginDeleteOp{Key: KeyForNasmVault(msg.VaultId)})
	} else {
		sfIndexEncoded, encErr := EncodeUint128(sfIndexNow)
		if encErr != nil {
			return &PluginDeliverResponse{Error: encErr}
		}
		vault.NusdPrincipal = newDebt
		vault.CollateralQuantity -= collateralReleasedU64
		vault.SfIndexAtOpen = sfIndexEncoded
		vaultBytesOut, mErr := Marshal(vault)
		if mErr != nil {
			return &PluginDeliverResponse{Error: mErr}
		}
		sets = append(sets, &PluginSetOp{Key: KeyForNasmVault(msg.VaultId), Value: vaultBytesOut})
	}

	// Step 11 -- single atomic StateWrite, matching mint_nusd.go's own
	// discipline.
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets, Deletes: deletes})
	if wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}
