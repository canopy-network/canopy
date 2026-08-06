package contract

import "math/big"

// state_accessors.go centralizes read/write logic for state records that
// are read by more than one caller shape (a single DeliverTx handler AND
// AccrueInterest's per-market BeginBlock/DeliverTx-shared logic, per AYIS
// Section 12.3's Accrual Ordering Contract). create_market.go, deposit.go,
// and withdraw.go each still inline their own market read/write, matching
// their existing, already-verified style -- these helpers are NOT a
// retrofit of that code, only the shared surface new callers (starting
// with AccrueInterest) build on, to avoid re-deriving the same
// empty-value guards and error handling at every new call site.

// GetMarket reads and unmarshals a Market record by ID. found=false with a
// nil error means the key simply does not exist -- callers decide what that
// means in their own context (DeliverTx: ErrMarketNotFound; BeginBlock
// range-walk: normally unreachable since the key came from the range read
// itself, so treat as a genuine anomaly, not a rejected transaction).
func GetMarket(c *Contract, marketID string) (market *Market, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForMarket(marketID)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	marketBytes := entryValue(readResp, 0)
	if len(marketBytes) == 0 {
		return nil, false, nil
	}
	m := &Market{}
	if uErr := Unmarshal(marketBytes, m); uErr != nil {
		return nil, false, uErr
	}
	return m, true, nil
}

// SaveMarket marshals and writes a Market record. Caller owns mutating the
// struct before calling this; this function performs exactly one state
// write and nothing else, matching deposit.go's single-responsibility
// write-site pattern.
func SaveMarket(c *Contract, marketID string, market *Market) *PluginError {
	marketBytes, mErr := Marshal(market)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForMarket(marketID), Value: marketBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// GetBorrowIndex reads and decodes B_index ({25}) for a market. found=false
// with a nil error means the key does not exist -- unreachable in practice
// since create_market always writes it (matching Section 4.5's
// initialization contract), but guarded explicitly rather than assumed,
// per this project's established standard.
func GetBorrowIndex(c *Contract, marketID string) (bIndex *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForBorrowIndex(marketID)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	return DecodeUint128(raw), true, nil
}

// SetBorrowIndexTry writes B_index using TryEncodeUint128 -- the
// BeginBlock-context-safe path (AYIS Section 3.2, M1). Returns ok=false
// (with no error) if the value would not fit in 128 bits; the caller MUST
// respond by freezing the single affected market (index_overflow_halted),
// never by treating ok=false as a PluginError to propagate as a revert --
// there is no transaction to revert in this context (Principle 14).
func SetBorrowIndexTry(c *Contract, marketID string, bIndex *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(bIndex)
	if !encOk {
		return false, nil
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForBorrowIndex(marketID), Value: encoded},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// GetSupplyIndex reads and decodes the SupplyIndexRecord ({26}) for a
// market, returning both s_rate and total_shares_outstanding per AYIS
// Section 9's 24-byte layout (M3). found=false with a nil error means the
// key does not exist.
func GetSupplyIndex(c *Contract, marketID string) (sRate *big.Int, totalSharesOutstanding uint64, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForSupplyIndex(marketID)},
		},
	})
	if err != nil {
		return nil, 0, false, err
	}
	if readResp.Error != nil {
		return nil, 0, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, 0, false, nil
	}
	sRate, totalSharesOutstanding = DecodeSupplyIndexRecord(raw)
	return sRate, totalSharesOutstanding, true, nil
}

// SetSupplyIndexTry writes the SupplyIndexRecord using TryEncodeUint128 for
// s_rate's own encoding -- the BeginBlock-context-safe path (AYIS Section
// 4.6, M1). totalSharesOutstanding is NOT modified by AccrueInterest (only
// a deposit/withdraw changes share count, per AYIS Section 4.2 -- interest
// accrual updates s_rate only) so it is passed through unchanged by the
// caller, not recomputed here. Returns ok=false (no error) if s_rate would
// not fit in 128 bits; caller MUST freeze the market, matching
// SetBorrowIndexTry's contract exactly.
func SetSupplyIndexTry(c *Contract, marketID string, sRate *big.Int, totalSharesOutstanding uint64) (ok bool, pErr *PluginError) {
	sRateEncoded, encOk := TryEncodeUint128(sRate)
	if !encOk {
		return false, nil
	}
	recordBytes := EncodeSupplyIndexRecord(sRateEncoded, totalSharesOutstanding)
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForSupplyIndex(marketID), Value: recordBytes},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// GetLossFactor reads and decodes loss_factor ({27}) for a market.
// found=false with a nil error means the key does not exist -- unreachable
// in practice since create_market always writes it (AYIS Section 4.5, G2),
// guarded explicitly rather than assumed.
func GetLossFactor(c *Contract, marketID string) (lossFactor *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForLossFactor(marketID)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	return DecodeUint128(raw), true, nil
}

// GetReserveFund reads and decodes R_fund ({18}) for a market. found=false
// with a nil error means the key does not exist -- unreachable in practice
// once create_market initializes it (mirrors GetBorrowIndex's guard style).
func GetReserveFund(c *Contract, marketID string) (rFund *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForReserveFund(marketID)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	return DecodeUint128(raw), true, nil
}

// SetReserveFundTry writes R_fund using TryEncodeUint128 -- the
// BeginBlock-context-safe path (ARCM Section 9.3, "interest" source leg).
// Returns ok=false (no error) if the value would not fit in 128 bits; the
// caller MUST respond by freezing the single affected market
// (IndexOverflowHalted), never by treating ok=false as a PluginError.
// This does NOT cover the DeliverTx-context legs (repay/liquidation
// principal routing) that ARCM Section 9.3 also specifies -- those require
// the reverting EncodeUint128 wrapper instead, and are out of scope here
// since repay/liquidate_position are not yet implemented.
func SetReserveFundTry(c *Contract, marketID string, rFund *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(rFund)
	if !encOk {
		return false, nil
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForReserveFund(marketID), Value: encoded},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// SetReserveFund writes R_fund using the reverting EncodeUint128 wrapper --
// the DeliverTx-context-safe path (ARCM Section 9.3, repay/liquidation
// principal-routing legs; ARCM Section 9.2 Layer 2 draw-down). Unlike
// SetReserveFundTry (BeginBlock-context, freezes the market on overflow),
// a DeliverTx caller has a real transaction to revert, so an out-of-range
// value here reverts the whole transaction via EncodeUint128's own
// PluginError return, per Principle 14's context-dependent response rule.
func SetReserveFund(c *Contract, marketID string, rFund *big.Int) *PluginError {
	encoded, encErr := EncodeUint128(rFund)
	if encErr != nil {
		return encErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForReserveFund(marketID), Value: encoded},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// [REVERSED] GetTreasury (single shared T_fund) is replaced by two
// distinct functions below -- see PrefixTreasuryArbor/PrefixTreasuryNASM's
// comment in state_keys.go for the full reversal reasoning.
//
// GetGovernanceParams reads and unmarshals the {22} GovernanceParams
// record -- a single global struct, matching Market's convention: one
// record, read-modify-write per change, not one state key per field. Unlike
// GetMarket, found=false with a nil error is the EXPECTED steady state
// before any governance parameter has ever been set (no create_market-
// equivalent initializes this record) -- callers must treat a zero-value
// GovernanceParams{} as the correct default in that case, mirroring
// GetTreasuryArbor's own found=false-is-normal contract, not GetMarket's
// found=false-is-an-error contract.
func GetGovernanceParams(c *Contract) (params *GovernanceParams, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForGovernanceParams()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	p := &GovernanceParams{}
	if uErr := Unmarshal(raw, p); uErr != nil {
		return nil, false, uErr
	}
	return p, true, nil
}

// SaveGovernanceParams marshals and writes the {22} GovernanceParams
// record. Caller owns read-modify-write: call GetGovernanceParams first
// (treating found=false as a zero-value GovernanceParams{} starting point,
// per GetGovernanceParams's own doc comment), mutate the field(s) being
// changed, then call this -- matching SaveMarket's single-responsibility
// write-site pattern exactly.
func SaveGovernanceParams(c *Contract, params *GovernanceParams) *PluginError {
	paramsBytes, mErr := Marshal(params)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForGovernanceParams(), Value: paramsBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// GetTreasuryArbor reads and decodes Arbor's own T_fund ({40}) -- the
// protocol treasury. Unlike every accessor above, this is NOT market-keyed:
// T_fund is a single global uint128 balance. found=false with a nil error
// means the key does not exist -- expected before T_fund's first write (no
// create_market-equivalent initializes a global value), unlike
// GetReserveFund's per-market unreachable-in-practice guard style.
func GetTreasuryArbor(c *Contract) (tFund *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForTreasuryArbor()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	return DecodeUint128(raw), true, nil
}

// GetTreasuryNASM is the NASM/NUSD-owned analog of GetTreasuryArbor above --
// same shape, reads {41} instead of {40}, fully isolated.
func GetTreasuryNASM(c *Contract) (tFund *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForTreasuryNASM()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	return DecodeUint128(raw), true, nil
}

// [REVERSED] SetTreasuryTry (single shared T_fund) is replaced by two
// distinct functions below -- same reversal as GetTreasury above.
//
// SetTreasuryArborTry writes Arbor's own T_fund using TryEncodeUint128 --
// the BeginBlock-context-safe path, mirroring SetReserveFundTry's contract
// exactly. Returns ok=false (no error) if the value would not fit in 128
// bits; the caller MUST respond with a protocol-level freeze signal
// appropriate to a global accumulator, never by treating ok=false as a
// PluginError to propagate as a revert -- there is no transaction to
// revert in BeginBlock context (Principle 14).
func SetTreasuryArborTry(c *Contract, tFund *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(tFund)
	if !encOk {
		return false, nil
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForTreasuryArbor(), Value: encoded},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// SetTreasuryNASMTry is the NASM/NUSD-owned analog of SetTreasuryArborTry
// above -- same shape, writes {41} instead of {40}, fully isolated.
func SetTreasuryNASMTry(c *Contract, tFund *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(tFund)
	if !encOk {
		return false, nil
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForTreasuryNASM(), Value: encoded},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// [REVERSED] SetTreasury (single shared T_fund) is replaced by two
// distinct functions below -- same reversal as GetTreasury/SetTreasuryTry
// above. Now used by Layer3DrawDown's split (bad_debt_layer3.go).
//
// SetTreasuryArbor writes Arbor's own T_fund using the reverting
// EncodeUint128 wrapper -- the DeliverTx-context-safe path, mirroring
// SetReserveFund's contract exactly. A DeliverTx caller has a real
// transaction to revert, so an out-of-range value here reverts the whole
// transaction via EncodeUint128's own PluginError return, per Principle
// 14's context-dependent response rule.
func SetTreasuryArbor(c *Contract, tFund *big.Int) *PluginError {
	encoded, encErr := EncodeUint128(tFund)
	if encErr != nil {
		return encErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForTreasuryArbor(), Value: encoded},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// SetTreasuryNASM is the NASM/NUSD-owned analog of SetTreasuryArbor above --
// same shape, writes {41} instead of {40}, fully isolated. No current
// caller exists yet -- NASM's own waterfall is not yet built.
func SetTreasuryNASM(c *Contract, tFund *big.Int) *PluginError {
	encoded, encErr := EncodeUint128(tFund)
	if encErr != nil {
		return encErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForTreasuryNASM(), Value: encoded},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// ─────────────────────────────────────────────
// NASM (NUSD / Arbor Stability Module) accessors -- {30} NasmVault, {31}
// NusdSupply, {32} StabilityFeeIndex, {34} RwaYieldVaultPosition. See
// arbor_state.proto's NASM message block and state_keys.go's NASM prefix
// block for full design rationale.
// ─────────────────────────────────────────────

// GetNasmVault reads and unmarshals a {30} NasmVault record by vault_id.
// found=false with a nil error means no vault exists at that vault_id --
// the expected state both before mint_nusd first creates one, and after
// burn_nusd deletes it on full closure (NASM Spec Section 4.2 Step 10).
// Matches GetMarket's found-is-meaningful contract exactly, not
// GetGovernanceParams/GetTreasuryNASM's found-false-is-normal-singleton
// contract -- a NasmVault is a real, individually-created-and-destroyed
// record, not a global default-zero value.
func GetNasmVault(c *Contract, vaultID string) (vault *NasmVault, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForNasmVault(vaultID)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	v := &NasmVault{}
	if uErr := Unmarshal(raw, v); uErr != nil {
		return nil, false, uErr
	}
	return v, true, nil
}

// SaveNasmVault marshals and writes a {30} NasmVault record. Caller owns
// read-modify-write: call GetNasmVault first, mutate, then call this --
// matching SaveMarket's single-responsibility write-site pattern.
func SaveNasmVault(c *Contract, vault *NasmVault) *PluginError {
	vaultBytes, mErr := Marshal(vault)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForNasmVault(vault.VaultId), Value: vaultBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// DeleteNasmVault removes a {30} NasmVault record entirely -- called on
// full closure (NASM Spec Section 4.2 Step 10: "If new_debt == 0: release
// all remaining collateral and delete the vault record"). A dedicated
// delete function, not a zero-value SaveNasmVault call, since an absent
// key and an explicit zero-value record must not be conflated at read
// sites downstream (same principle as create_market.go's zero-init
// comment for Layer4PendingBadDebtTotal).
func DeleteNasmVault(c *Contract, vaultID string) *PluginError {
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Deletes: []*PluginDeleteOp{
			{Key: KeyForNasmVault(vaultID)},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// GetNusdSupply reads and unmarshals the {31} NusdSupply record -- a
// single global struct, matching GovernanceParams' convention. found=false
// with a nil error is the expected steady state before mint_nusd's first
// call ever writes it; callers MUST treat a zero-value NusdSupply{} (i.e.
// total_supply=0) as correct in that case, mirroring
// GetGovernanceParams/GetTreasuryNASM's found=false-is-normal contract.
func GetNusdSupply(c *Contract) (supply *NusdSupply, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForNusdSupply()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	s := &NusdSupply{}
	if uErr := Unmarshal(raw, s); uErr != nil {
		return nil, false, uErr
	}
	return s, true, nil
}

// SaveNusdSupply marshals and writes the {31} NusdSupply record. Caller
// owns read-modify-write, matching SaveGovernanceParams' pattern exactly.
func SaveNusdSupply(c *Contract, supply *NusdSupply) *PluginError {
	supplyBytes, mErr := Marshal(supply)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForNusdSupply(), Value: supplyBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// GetStabilityFeeIndex reads and decodes the {32} StabilityFeeIndex record
// -- a single global RAY-scaled uint128 value, reusing AYIS's B_index/
// S_rate accrual pattern (NASM Spec Section 6.2). found=false with a nil
// error means the index has never been initialized -- callers at genesis
// MUST treat this as RAY (1e18), matching AYIS's SupplyIndexRecord
// initialization convention (create_market.go's s_rate=RAY precedent),
// NOT as zero -- an uninitialized multiplicative index must default to the
// identity value, not the zero value.
func GetStabilityFeeIndex(c *Contract) (sfIndex *big.Int, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForStabilityFeeIndex()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	sf := &StabilityFeeIndex{}
	if uErr := Unmarshal(raw, sf); uErr != nil {
		return nil, false, uErr
	}
	return DecodeUint128(sf.SfIndex), true, nil
}

// SetStabilityFeeIndexTry writes the {32} StabilityFeeIndex using
// TryEncodeUint128 -- the BeginBlock-context-safe path (NASM Spec Section
// 6.5: SF_index accrual runs in BeginBlock, alongside AYIS.AccrueInterest).
// Mirrors SetTreasuryArborTry's contract exactly: returns ok=false (no
// error) on overflow; the caller MUST respond with a protocol-level freeze
// signal, never treat ok=false as a PluginError to propagate as a revert --
// there is no transaction to revert in BeginBlock context (Principle 14).
func SetStabilityFeeIndexTry(c *Contract, sfIndex *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(sfIndex)
	if !encOk {
		return false, nil
	}
	sfBytes, mErr := Marshal(&StabilityFeeIndex{SfIndex: encoded})
	if mErr != nil {
		return false, mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForStabilityFeeIndex(), Value: sfBytes},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}

// GetRwaYieldVaultPosition reads and unmarshals a {34} RwaYieldVaultPosition
// record by depositor address. found=false with a nil error means the
// depositor has no RYV position yet -- expected before their first
// deposit_yield_vault call. Matches GetLenderPosition's presumed
// per-address-keyed contract (address-keyed, not global).
func GetRwaYieldVaultPosition(c *Contract, addr []byte) (position *RwaYieldVaultPosition, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForRwaYieldVaultPosition(addr)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	p := &RwaYieldVaultPosition{}
	if uErr := Unmarshal(raw, p); uErr != nil {
		return nil, false, uErr
	}
	return p, true, nil
}

// SaveRwaYieldVaultPosition marshals and writes a {34} RwaYieldVaultPosition
// record. Caller owns read-modify-write.
func SaveRwaYieldVaultPosition(c *Contract, position *RwaYieldVaultPosition) *PluginError {
	posBytes, mErr := Marshal(position)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForRwaYieldVaultPosition(position.Depositor), Value: posBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// GetNusdBalance reads and unmarshals a {35} NusdBalance record by address.
// found=false with a nil error means the holder has no NUSD balance yet --
// the expected state before their first mint_nusd call (or before ever
// receiving a transfer, once transfer_nusd exists -- see NusdBalance's own
// doc comment on that disclosed gap). Matches LenderPosition's
// found-is-meaningful, address-keyed contract.
func GetNusdBalance(c *Contract, addr []byte) (balance *NusdBalance, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForNusdBalance(addr)},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	b := &NusdBalance{}
	if uErr := Unmarshal(raw, b); uErr != nil {
		return nil, false, uErr
	}
	return b, true, nil
}

// SaveNusdBalance marshals and writes a {35} NusdBalance record. Caller
// owns read-modify-write, matching SaveNasmVault/SaveLenderPosition's
// single-responsibility write-site pattern.
func SaveNusdBalance(c *Contract, balance *NusdBalance) *PluginError {
	balBytes, mErr := Marshal(balance)
	if mErr != nil {
		return mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForNusdBalance(balance.Address), Value: balBytes},
		},
	})
	if err != nil {
		return err
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// creditNusdBalance adds amount to balance.Amount. Checked-add, mirrors
// custody_arith.go's creditAccountAmount exactly, but operating on
// NusdBalance.Amount instead of Account.Amount -- see NusdBalance's own
// doc comment for why these must be two structurally independent balances,
// never the same field. Pure function, no state I/O -- caller (mint_nusd)
// owns the surrounding GetNusdBalance/SaveNusdBalance read-modify-write,
// matching creditAccountAmount's own division of responsibility.
func creditNusdBalance(addr []byte, balance *NusdBalance, amount uint64) *PluginError {
	if amount > (^uint64(0) - balance.Amount) {
		return ErrAccountBalanceOverflow(addr, balance.Amount, amount)
	}
	balance.Amount += amount
	return nil
}

// debitNusdBalance subtracts amount from balance.Amount. Compare-before-
// subtract, mirrors custody_arith.go's debitAccountAmount exactly. Returns
// ErrInsufficientFunds() on shortfall -- will be the primary guard
// burn_nusd uses to confirm a caller holds enough NUSD before burning it.
func debitNusdBalance(balance *NusdBalance, amount uint64) *PluginError {
	if balance.Amount < amount {
		return ErrInsufficientFunds()
	}
	balance.Amount -= amount
	return nil
}

// GetStabilityFeeIndexRecord reads and unmarshals the full {32}
// StabilityFeeIndex record (sf_index AND last_accrual_block), for callers
// that need both fields -- AccrueStabilityFee's own use, distinct from
// GetStabilityFeeIndex above, which returns only the decoded sf_index
// value for simpler callers (mint_nusd/burn_nusd) that don't need
// last_accrual_block. found=false means the index has never been
// initialized -- callers MUST treat this as sf_index=RAY (the
// multiplicative identity, matching GetStabilityFeeIndex's own doc
// comment) and last_accrual_block=the current block (first accrual ever,
// zero elapsed blocks, matching Market's own first-accrual convention).
func GetStabilityFeeIndexRecord(c *Contract) (record *StabilityFeeIndex, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForStabilityFeeIndex()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	sf := &StabilityFeeIndex{}
	if uErr := Unmarshal(raw, sf); uErr != nil {
		return nil, false, uErr
	}
	return sf, true, nil
}

// SetStabilityFeeIndexRecordTry writes the full {32} StabilityFeeIndex
// record (sf_index AND last_accrual_block) using TryEncodeUint128 for
// sf_index -- the BeginBlock-context-safe path (NASM Spec Section 6.5:
// SF_index accrual runs in BeginBlock). Mirrors SetStabilityFeeIndexTry's
// contract exactly: returns ok=false (no error) on sf_index overflow; the
// caller (AccrueStabilityFee) MUST respond by leaving the index at its
// last successfully-committed value rather than treating ok=false as a
// PluginError to propagate -- there is no transaction to revert in
// BeginBlock context (Principle 14).
// SetStabilityFeeIndexRecordTry now also persists remainderRay (uint128,
// RAY-scaled) alongside sf_index/last_accrual_block -- added when
// AccrueStabilityFee was wired to actually credit {41} from NusdSupply's
// aggregate debt (NASM Spec Section 6.4), mirroring
// Market.InterestRemainderRay's identical dust-carry fix.
func SetStabilityFeeIndexRecordTry(c *Contract, sfIndex *big.Int, lastAccrualBlock uint64, remainderRay *big.Int) (ok bool, pErr *PluginError) {
	encoded, encOk := TryEncodeUint128(sfIndex)
	if !encOk {
		return false, nil
	}
	remainderEncoded, remOk := TryEncodeUint128(remainderRay)
	if !remOk {
		// Structurally unreachable -- a remainder (numerator mod RAY) can
		// never itself reach or exceed RAY, so it can never fail to fit in
		// 128 bits. Checked anyway, matching interest_accrual.go's identical
		// remainder-encode check rather than assuming the invariant.
		return false, nil
	}
	sfBytes, mErr := Marshal(&StabilityFeeIndex{SfIndex: encoded, LastAccrualBlock: lastAccrualBlock, RemainderRay: remainderEncoded})
	if mErr != nil {
		return false, mErr
	}
	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForStabilityFeeIndex(), Value: sfBytes},
		},
	})
	if err != nil {
		return false, err
	}
	if writeResp.Error != nil {
		return false, writeResp.Error
	}
	return true, nil
}
