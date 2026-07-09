package contract

import (
	"math/big"
)

// CheckMessageDeposit statelessly validates a 'deposit' message (AYIS
// Section 4.3, ARCM Section 19.2.1b). No state reads here -- market
// existence, Insolvent/frozen/pending-loss admission checks all happen at
// DeliverTx, matching create_market's stateless/stateful split.
func (c *Contract) CheckMessageDeposit(msg *MessageDeposit) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Address) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	// [NOT ENFORCED] AYIS Section 13 states MIN_DEPOSIT = 1 CNPY (governance
	// range 1-100 CNPY), a dust-prevention floor. CNPY is Canopy's own
	// chain-level gas/native token -- it has no pricing relationship to any
	// Arbor market's debt_asset_id, and Arbor does not consume a CNPY price
	// feed anywhere in ARCM/AYIS's oracle design (Section 10). The correct
	// denomination for this floor is NUSD, Arbor's own native USD-pegged
	// stablecoin (an in-protocol NASM module, not a separate nested chain) --
	// but NUSD does not exist yet; NASM coordination is explicitly deferred
	// until Arbor's lending foundation (this file included) is complete.
	// Enforcing MIN_DEPOSIT against CNPY now would silently reject deposits
	// using a foreign, economically meaningless reference point -- actively
	// wrong, not merely incomplete. Left unenforced until NUSD exists and
	// this floor can be correctly restated in NUSD terms. See ErrDepositBelowMinimum.
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Address}}
}

// DeliverMessageDeposit handles a 'deposit' message: AYIS Section 4.3
// MintShares() plus ARCM Section 19.2.1b's total_supplied write path.
func (c *Contract) DeliverMessageDeposit(msg *MessageDeposit, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}

	marketKey := KeyForMarket(msg.MarketId)
	supplyIndexKey := KeyForSupplyIndex(msg.MarketId)
	lossFactorKey := KeyForLossFactor(msg.MarketId)
	lenderPosKey := KeyForLenderPosition(msg.MarketId, msg.Address)

	// Batched read: Market, SupplyIndexRecord ({26}), loss_factor ({27}),
	// and this address's existing LenderPosition ({24}, may not exist yet --
	// first deposit for this address in this market). One round trip,
	// matching create_market's existence-check pattern.
	const (
		qMarket = iota
		qSupplyIndex
		qLossFactor
		qLenderPos
	)
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qMarket, Key: marketKey},
			{QueryId: qSupplyIndex, Key: supplyIndexKey},
			{QueryId: qLossFactor, Key: lossFactorKey},
			{QueryId: qLenderPos, Key: lenderPosKey},
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
	market := &Market{}
	if uErr := Unmarshal(marketBytes, market); uErr != nil {
		return &PluginDeliverResponse{Error: uErr}
	}

	// Admission checks, in order: index_overflow_halted (ARCM Section 9.3a --
	// deposit IS in this flag's blocked set, unlike withdraw post-C2) then
	// layer4_pending_count > 0 (ARCM Section 9.2b -- blocks deposit AND
	// withdraw). Insolvent is checked below via loss_factor.Sign()==0
	// (AYIS Section 4.3 H1), which is MintShares()'s own guard rather than a
	// separate admission-layer check -- the two documents enforce the same
	// underlying condition from their own respective sides.
	if market.IndexOverflowHalted {
		return &PluginDeliverResponse{Error: ErrMarketIndexOverflowHalted(msg.MarketId)}
	}
	if market.Layer4PendingCount > 0 {
		return &PluginDeliverResponse{Error: ErrMarketLayer4Pending(msg.MarketId)}
	}

	supplyIndexBytes := entryValue(readResp, qSupplyIndex)
	if len(supplyIndexBytes) == 0 {
		// Unreachable in practice: create_market always writes {26}. Guarded
		// explicitly per this project's standard of never assuming a read
		// succeeded silently.
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	sRate, totalSharesOutstanding := DecodeSupplyIndexRecord(supplyIndexBytes)

	lossFactorBytes := entryValue(readResp, qLossFactor)
	if len(lossFactorBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	lossFactor := DecodeUint128(lossFactorBytes)

	// [AYIS Section 4.3, H1] Defense-in-depth guard: a market whose
	// loss_factor has been driven to exactly zero (total lender wipeout,
	// AYIS Section 5.4.3) is Insolvent. Dividing by it below would panic;
	// this guard makes the rejection explicit and named instead.
	if lossFactor.Sign() == 0 {
		return &PluginDeliverResponse{Error: ErrMarketInsolvent(msg.MarketId)}
	}

	// shares = amount * RAY * RAY / (s_rate * loss_factor), floor (AYIS
	// Section 4.3, G1 -- loss-adjusted redemption value, not S_rate alone).
	numerator := new(big.Int).Mul(new(big.Int).SetUint64(msg.Amount), RAY)
	numerator.Mul(numerator, RAY)
	denominator := new(big.Int).Mul(sRate, lossFactor)
	sharesBig := new(big.Int).Div(numerator, denominator)

	// [AYIS Section 4.3, J2] Cast-safety guard -- proves sharesBig fits in
	// 64 bits before any further use.
	if sharesBig.BitLen() > 64 {
		return &PluginDeliverResponse{Error: ErrShareOverflow(msg.MarketId, msg.Amount, sharesBig.String())}
	}
	sharesMinted := sharesBig.Uint64()

	// [AYIS Section 4.3, L1] Accumulator-safety guard -- DIFFERENT property
	// from J2 above. J2 proves sharesMinted itself is representable; this
	// proves ADDING it to the existing total_shares_outstanding accumulator
	// does not overflow that accumulator. Both required; neither implies
	// the other.
	if sharesMinted > (^uint64(0) - totalSharesOutstanding) {
		return &PluginDeliverResponse{Error: ErrTotalSharesOverflow(msg.MarketId, totalSharesOutstanding, sharesBig.String())}
	}
	newTotalShares := totalSharesOutstanding + sharesMinted

	// [ARCM Section 19.2.1b, M2] total_supplied write path -- checked-add,
	// runs alongside the shares accumulator above. Denominated in the
	// market's native asset (same unit as total_borrowed), not shares.
	if msg.Amount > (^uint64(0) - market.TotalSupplied) {
		return &PluginDeliverResponse{Error: ErrTotalSuppliedOverflow(msg.MarketId, market.TotalSupplied, msg.Amount)}
	}
	market.TotalSupplied += msg.Amount

	// Re-encode SupplyIndexRecord ({26}) with the new total_shares_outstanding.
	// s_rate itself is unchanged by a deposit (AYIS Section 4.2 -- only
	// BeginBlock's AccrueInterest updates S_rate); re-encode the same
	// sRate bytes the read returned, DeliverTx-context revert-on-failure
	// wrapper since sRate was already known-valid from state.
	sRateEncoded, encErr := EncodeUint128(sRate)
	if encErr != nil {
		return &PluginDeliverResponse{Error: encErr}
	}
	newSupplyIndexBytes := EncodeSupplyIndexRecord(sRateEncoded, newTotalShares)

	// LenderPosition ({24}): create new or update existing shares total.
	// deposit_block updates to the current block on every deposit (matches
	// AYIS Section 5.1's field semantics -- most recent deposit activity).
	lenderPosBytes := entryValue(readResp, qLenderPos)
	lenderPos := &LenderPosition{}
	if len(lenderPosBytes) > 0 {
		if uErr := Unmarshal(lenderPosBytes, lenderPos); uErr != nil {
			return &PluginDeliverResponse{Error: uErr}
		}
	} else {
		lenderPos.MarketId = msg.MarketId
		lenderPos.Address = msg.Address
	}
	lenderPos.Shares += sharesMinted
	lenderPos.DepositBlock = c.plugin.CurrentHeight()
	lenderPosBytesOut, mErr := Marshal(lenderPos)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	marketBytesOut, mErr := Marshal(market)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: marketKey, Value: marketBytesOut},
			{Key: supplyIndexKey, Value: newSupplyIndexBytes},
			{Key: lenderPosKey, Value: lenderPosBytesOut},
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

// entryValue extracts the raw value bytes for a given query_id from a
// PluginStateReadResponse, or nil if the key had no entry (does not exist).
// Small helper to avoid repeating the same nested-slice-index defensive
// checks at every read site above.
func entryValue(resp *PluginStateReadResponse, queryId uint64) []byte {
	for _, result := range resp.Results {
		if result.QueryId == queryId {
			if len(result.Entries) > 0 {
				return result.Entries[0].Value
			}
			return nil
		}
	}
	return nil
}
