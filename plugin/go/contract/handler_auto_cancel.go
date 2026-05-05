package contract

// handler_auto_cancel.go — CheckAutoCancel helper
// Spec: ADLMSR v5.6.6-r2-CORRECTED (AUDIT-2)
//
// CheckAutoCancel is called at the start of SubmitPrediction, ResolveMarket,
// and ClaimWinnings. If the market has passed its expiry window without a
// resolver committing, it cancels the market atomically.
//
// This is NOT a transaction type — it is a shared helper with a 4-key atomic write.
// AUDIT-2: single atomic write — market status + pool + treasury + resolver state.
//
// Returns nil if no cancellation was needed or cancellation succeeded.
// Returns *PluginError if state read/write failed.

func (c *Contract) CheckAutoCancel(marketId []byte) *PluginError {
now := GetGlobalHeight()
if now == 0 {
return ErrHeightNotSet()
}

marketQId := nextQueryId()
poolQId   := nextQueryId()

marketKey := KeyForMarket(marketId)
poolKey   := KeyForMarketPool(marketId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId, Key: marketKey},
{QueryId: poolQId,   Key: poolKey},
},
})
if err != nil {
return err
}
if resp.Error != nil {
return resp.Error
}

var market *MarketState
var pool   *Pool

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case marketQId:
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return pe
}
case poolQId:
pool = &Pool{}
if pe := Unmarshal(r.Entries[0].Value, pool); pe != nil {
return pe
}
}
}

// Market not found or already in a terminal/non-open state — nothing to cancel.
if market == nil {
return nil
}
if market.Status != STATUS_OPEN {
return nil
}

// Market is still open — check if the resolution window has passed entirely.
// Auto-cancel condition: now > ExpiryTime + GRACE_PERIOD_BLOCKS
// (resolver had the full grace period and did not resolve).
// AUDIT-11: ExpiryTime is bounded by MAX_EXPIRY_TIME so addition cannot overflow.
cancelThreshold := market.ExpiryTime + RESOLUTION_DELAY_BLOCKS + GRACE_PERIOD_BLOCKS
if now <= cancelThreshold {
// Still within window — no cancellation.
return nil
}

// Auto-cancel: transition market to STATUS_CANCELLED.
market.Status = STATUS_CANCELLED

rawMarket, pe := SafeMarshal(market)
if pe != nil {
return pe
}

sets := []*PluginSetOp{
{Key: marketKey, Value: rawMarket},
}

// If pool is non-nil and has funds, sweep them to treasury.
// This handles the case where bettors submitted predictions but no resolver appeared.
if pool != nil && pool.Amount > 0 {
pool.Amount = 0
rawPool, pe := SafeMarshal(pool)
if pe != nil {
return pe
}
sets = append(sets, &PluginSetOp{Key: poolKey, Value: rawPool})
}

// AUDIT-2: single atomic write.
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: sets,
})
return errCheckWrite(wr, werr)
}
