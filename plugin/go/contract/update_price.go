package contract

import "bytes"

// CheckMessageUpdatePrice statelessly validates an 'update_price' message
// (ARCM Section 10). No state reads here -- authorization against the
// market's authorized_submitters list requires a state read, so it happens
// at DeliverTx, matching every other handler's stateless/stateful split.
func (c *Contract) CheckMessageUpdatePrice(msg *MessageUpdatePrice) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.AssetId) == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.Price == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if len(msg.Submitter) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Submitter}}
}

// DeliverMessageUpdatePrice handles an 'update_price' message: ARCM Section
// 10's permissioned price feed. Writes ONE submitter's reading as its own
// PriceRecord ({19}, keyed by asset_id+submitter) -- quorum and deviation
// checks are read-side concerns (see design note in state_keys.go's
// KeyForPriceRecord), not enforced here. A write is never rejected for
// disagreeing with other submitters' readings; only later CONSUMPTION of
// an insufficient or too-divergent price set is blocked, at the point
// something tries to use it.
func (c *Contract) DeliverMessageUpdatePrice(msg *MessageUpdatePrice, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.AssetId) == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if len(msg.Submitter) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}

	market, found, err := GetMarket(c, msg.MarketId)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}

	// asset_id must actually belong to this market -- otherwise a submitter
	// authorized for market X could post a price for an unrelated asset Y
	// merely by naming market X in market_id and Y in asset_id.
	if msg.AssetId != market.CollateralAssetId && msg.AssetId != market.DebtAssetId {
		return &PluginDeliverResponse{Error: ErrAssetNotInMarket(msg.MarketId, msg.AssetId)}
	}

	// Real authorization check -- msg.Submitter must appear in THIS
	// market's authorized_submitters list (set once at create_market,
	// immutable -- see create_market.go). Linear scan: this list is
	// bounded by practical submitter-set size (small, human-managed), not
	// by market count or chain history, so this is not the kind of
	// unbounded scan the market_id field was added specifically to avoid.
	authorized := false
	for _, addr := range market.AuthorizedSubmitters {
		if bytes.Equal(addr, msg.Submitter) {
			authorized = true
			break
		}
	}
	if !authorized {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	record := &PriceRecord{
		AssetId:       msg.AssetId,
		Submitter:     msg.Submitter,
		Price:         msg.Price,
		ConfidenceBps: msg.ConfidenceBps,
		BlockHeight:   c.plugin.CurrentHeight(),
	}
	recordBytes, mErr := Marshal(record)
	if mErr != nil {
		return &PluginDeliverResponse{Error: mErr}
	}

	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForPriceRecord(msg.AssetId, msg.Submitter), Value: recordBytes},
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
