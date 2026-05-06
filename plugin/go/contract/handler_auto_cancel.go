package contract

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

if market == nil {
return nil
}
if market.Status != STATUS_OPEN {
return nil
}

cancelThreshold := market.ExpiryTime + RESOLUTION_DELAY_BLOCKS + GRACE_PERIOD_BLOCKS
if now <= cancelThreshold {
return nil
}

market.Status = STATUS_CANCELLED

rawMarket, pe := SafeMarshal(market)
if pe != nil {
return pe
}

sets := []*PluginSetOp{
{Key: marketKey, Value: rawMarket},
}

if pool != nil && pool.Amount > 0 {
pool.Amount = 0
rawPool, pe := SafeMarshal(pool)
if pe != nil {
return pe
}
sets = append(sets, &PluginSetOp{Key: poolKey, Value: rawPool})
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: sets,
})
return errCheckWrite(wr, werr)
}
