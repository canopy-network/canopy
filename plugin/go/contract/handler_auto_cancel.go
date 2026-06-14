package contract

// CheckAutoCancel checks if the market should be auto-cancelled and returns
// the updated MarketState if so (status set to STATUS_CANCELLED), or nil if not.
// NO StateWrite is performed here — the caller must include the updated market
// in its own atomic StateWrite to avoid multiple writes per deliver context.
func (c *Contract) CheckAutoCancel(marketId []byte) (*MarketState, *PluginError) {
now := GetGlobalHeight()
if now == 0 {
return nil, ErrHeightNotSet()
}

marketQId := nextQueryId()
marketKey := KeyForMarket(marketId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId, Key: marketKey},
},
})
if err != nil {
return nil, err
}
if resp.Error != nil {
return nil, resp.Error
}

var market *MarketState
for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
if r.QueryId == marketQId {
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return nil, pe
}
}
}

if market == nil {
return nil, nil
}
if market.Status != STATUS_OPEN {
return nil, nil
}

resolutionDelay := RESOLUTION_DELAY_BLOCKS
gracePeriod     := GRACE_PERIOD_BLOCKS
if TEST_MODE {
resolutionDelay = TEST_RESOLUTION_DELAY
gracePeriod     = TEST_GRACE_PERIOD
}
cancelThreshold := market.ExpiryTime + resolutionDelay + gracePeriod
if now <= cancelThreshold {
return nil, nil
}

// Return the cancelled market — caller writes it atomically with other state.
market.Status = STATUS_CANCELLED
return market, nil
}
