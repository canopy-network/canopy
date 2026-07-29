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
