package contract

const (
bountyAmount uint64 = 50_000_000
bondAmount   uint64 = 100_000_000
)

func (c *Contract) CheckMessageResolveMarket(msg *MessageResolveMarket) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageResolveMarket(msg *MessageResolveMarket, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId   := nextQueryId()
resolverQId := nextQueryId()
outcomeQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: resolverQId, Key: KeyForResolverState(msg.MarketId)},
{QueryId: outcomeQId,  Key: KeyForOutcome(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market   *MarketState
var resolver *ResolverState
var outcomeRaw []byte

for _, r := range resp.Results {
switch r.QueryId {
case marketQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case resolverQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrNoResolverRegistered()}
}
resolver = &ResolverState{}
if pe := Unmarshal(r.Entries[0].Value, resolver); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case outcomeQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
outcomeRaw = r.Entries[0].Value
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if resolver == nil {
return &PluginDeliverResponse{Error: ErrNoResolverRegistered()}
}

// R1 FIX: AUTH BEFORE IDEMPOTENCY
if !bytesEqual(resolver.ResolverAddress, msg.ResolverAddress) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}

withinWindow := now >= market.ExpiryTime+RESOLUTION_DELAY_BLOCKS &&
now <= market.ExpiryTime+RESOLUTION_DELAY_BLOCKS+GRACE_PERIOD_BLOCKS
if !withinWindow {
if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
refreshQId := nextQueryId()
resp2, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: refreshQId, Key: KeyForMarket(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp2.Error != nil {
return &PluginDeliverResponse{Error: resp2.Error}
}
for _, r := range resp2.Results {
if r.QueryId == refreshQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
}

if market.Status == STATUS_CANCELLED {
return &PluginDeliverResponse{Error: ErrMarketCancelled()}
}

if outcomeRaw != nil {
return &PluginDeliverResponse{}
}

if market.Status != STATUS_OPEN {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}
if now < market.ExpiryTime+RESOLUTION_DELAY_BLOCKS {
return &PluginDeliverResponse{Error: ErrResolutionTooEarly()}
}

poolQId     := nextQueryId()
resolAccQId := nextQueryId()
creatAccQId := nextQueryId()
treasQId    := nextQueryId()

payResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId,     Key: KeyForMarketPool(msg.MarketId)},
{QueryId: resolAccQId, Key: KeyForAccount(msg.ResolverAddress)},
{QueryId: creatAccQId, Key: KeyForAccount(market.Creator)},
{QueryId: treasQId,    Key: KeyForTreasuryReserve(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if payResp.Error != nil {
return &PluginDeliverResponse{Error: payResp.Error}
}

mPool       := &Pool{}
resolverAcc := &Account{}
creatorAcc  := &Account{}
tres        := &TreasuryReserve{}

for _, r := range payResp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
switch r.QueryId {
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, mPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case resolAccQId:
if pe := Unmarshal(r.Entries[0].Value, resolverAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case creatAccQId:
if pe := Unmarshal(r.Entries[0].Value, creatorAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case treasQId:
if pe := Unmarshal(r.Entries[0].Value, tres); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if mPool.Amount < bountyAmount+bondAmount {
return &PluginDeliverResponse{Error: ErrInsufficientPoolFunds()}
}

market.Status      = STATUS_RESOLVED
mPool.Amount      -= bountyAmount + bondAmount
resolverAcc.Amount += bountyAmount
creatorAcc.Amount  += bondAmount
tres.LockedReserve  = 0
outcome := &OutcomeState{WinningOutcome: msg.WinningOutcome, ResolvedAt: now}

rawM, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawO, pe := SafeMarshal(outcome)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawT, pe := SafeMarshal(tres)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawMP, pe := SafeMarshal(mPool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawRA, pe := SafeMarshal(resolverAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawCA, pe := SafeMarshal(creatorAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),           Value: rawM},
{Key: KeyForOutcome(msg.MarketId),           Value: rawO},
{Key: KeyForTreasuryReserve(msg.MarketId),   Value: rawT},
{Key: KeyForMarketPool(msg.MarketId),        Value: rawMP},
{Key: KeyForAccount(msg.ResolverAddress),    Value: rawRA},
{Key: KeyForAccount(market.Creator),         Value: rawCA},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
