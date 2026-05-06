package contract

func (c *Contract) CheckMessageSubmitPrediction(msg *MessageSubmitPrediction) *PluginCheckResponse {
if len(msg.BettorAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if msg.Shares < PRECISION_SCALE {
return ErrCheckResp(ErrSharesBelowMinimum())
}
if msg.MaxCost == 0 {
return ErrCheckResp(ErrInvalidAmount())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.BettorAddress},
}
}

func (c *Contract) DeliverMessageSubmitPrediction(msg *MessageSubmitPrediction, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}
if msg.Shares < PRECISION_SCALE {
return &PluginDeliverResponse{Error: ErrSharesBelowMinimum()}
}
if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

marketQId  := nextQueryId()
posQId     := nextQueryId()
poolQId    := nextQueryId()
bettorQId  := nextQueryId()
feeQId     := nextQueryId()

marketKey  := KeyForMarket(msg.MarketId)
posKey     := KeyForPosition(msg.MarketId, msg.BettorAddress)
poolKey    := KeyForMarketPool(msg.MarketId)
bettorKey  := KeyForAccount(msg.BettorAddress)
feePoolKey := KeyForFeePool(c.Config.ChainId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId, Key: marketKey},
{QueryId: posQId,    Key: posKey},
{QueryId: poolQId,   Key: poolKey},
{QueryId: bettorQId, Key: bettorKey},
{QueryId: feeQId,    Key: feePoolKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market *MarketState
position  := &PositionState{}
mPool     := &Pool{}
bettor    := &Account{}
feePool   := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case marketQId:
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case posQId:
if pe := Unmarshal(r.Entries[0].Value, position); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, mPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case bettorQId:
if pe := Unmarshal(r.Entries[0].Value, bettor); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feeQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if market.Status != STATUS_OPEN {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}
if now > market.ExpiryTime {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}
if now < market.OpenTime {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}

tradeCost, pe := ComputeTradeCost(market.QYes, market.QNo, market.BEff, msg.Shares, msg.Outcome)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
if fee > 0 && tradeCost > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
finalCost := tradeCost + fee
if finalCost > msg.MaxCost {
return &PluginDeliverResponse{Error: ErrCostExceedsMaxCost()}
}
if bettor.Amount < finalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

isNewPosition := position.SharesYes == 0 && position.SharesNo == 0 && position.CostPaid == 0

bettor.Amount  -= finalCost
mPool.Amount   += tradeCost
feePool.Amount += fee

if msg.Outcome {
market.QYes += msg.Shares
position.SharesYes += msg.Shares
} else {
market.QNo += msg.Shares
position.SharesNo += msg.Shares
}
position.CostPaid += tradeCost

if isNewPosition {
market.TotalPositions++
}

market.ElevatedRisk = IsElevatedRisk(mPool.Amount)

rawMarket, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPos, pe := SafeMarshal(position)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPool, pe := SafeMarshal(mPool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: marketKey, Value: rawMarket},
{Key: posKey,    Value: rawPos},
{Key: poolKey,   Value: rawPool},
{Key: feePoolKey, Value: rawFee},
}
var deletes []*PluginDeleteOp
if bettor.Amount == 0 {
deletes = []*PluginDeleteOp{{Key: bettorKey}}
} else {
rawBettor, pe := SafeMarshal(bettor)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: bettorKey, Value: rawBettor})
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets:    sets,
Deletes: deletes,
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
