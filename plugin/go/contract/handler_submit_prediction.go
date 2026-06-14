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
if _, cancelErr := c.CheckAutoCancel(msg.MarketId); cancelErr != nil {
return &PluginDeliverResponse{Error: cancelErr}
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
feePoolKey      := KeyForFeePool(c.Config.ChainId)
creatorFeeKey   := KeyForCreatorFeePool(msg.MarketId)
resolverFeeKey  := KeyForResolverFeePool(msg.MarketId)
	gTreasuryKey    := KeyForTreasuryPool()
creatorFeeQId   := nextQueryId()
	resolverFeeQId  := nextQueryId()
	gTreasuryQId    := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId, Key: marketKey},
{QueryId: posQId,    Key: posKey},
{QueryId: poolQId,   Key: poolKey},
{QueryId: bettorQId, Key: bettorKey},
{QueryId: feeQId,         Key: feePoolKey},
{QueryId: creatorFeeQId,  Key: creatorFeeKey},
{QueryId: resolverFeeQId, Key: resolverFeeKey},
	{QueryId: gTreasuryQId,   Key: gTreasuryKey},
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
mPool       := &Pool{}
bettor      := &Account{}
feePool     := &Pool{}
creatorFee  := &Pool{}
resolverFee := &Pool{}
	gTreasury   := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
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
case creatorFeeQId:
if pe := Unmarshal(r.Entries[0].Value, creatorFee); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case resolverFeeQId:
if pe := Unmarshal(r.Entries[0].Value, resolverFee); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case gTreasuryQId:
if pe := Unmarshal(r.Entries[0].Value, gTreasury); pe != nil {
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
creatorFeeAmt  := ComputeBps(tradeCost, CREATOR_FEE_BPS)
resolverFeeAmt := ComputeBps(tradeCost, RESOLVER_FEE_BPS)
if fee > 0 && tradeCost > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
finalCost := tradeCost + fee + creatorFeeAmt + resolverFeeAmt
if finalCost > msg.MaxCost {
return &PluginDeliverResponse{Error: ErrCostExceedsMaxCost()}
}
// COI-3: per-address position cap — capped on shares, not CostPaid.
// totalSideShares is post-trade so the cap scales with actual exposure.
var totalSideShares uint64
if msg.Outcome {
totalSideShares = market.QYes + msg.Shares
if exceedsPositionCap(position.SharesYes, msg.Shares, totalSideShares) {
return &PluginDeliverResponse{Error: ErrPositionCapExceeded()}
}
} else {
totalSideShares = market.QNo + msg.Shares
if exceedsPositionCap(position.SharesNo, msg.Shares, totalSideShares) {
return &PluginDeliverResponse{Error: ErrPositionCapExceeded()}
}
}

if bettor.Amount < finalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

isNewPosition := position.SharesYes == 0 && position.SharesNo == 0 && position.CostPaid == 0

bettor.Amount      -= finalCost
mPool.Amount       += tradeCost
feePool.Amount     += fee / 2
	gTreasury.Amount   += fee - fee/2
creatorFee.Amount  += creatorFeeAmt
resolverFee.Amount += resolverFeeAmt

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
rawGTreasury, pe := SafeMarshal(gTreasury)
	if pe != nil { return &PluginDeliverResponse{Error: pe} }
	rawCreatorFee, pe := SafeMarshal(creatorFee)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawResolverFee, pe := SafeMarshal(resolverFee)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: marketKey,      Value: rawMarket},
{Key: posKey,         Value: rawPos},
{Key: poolKey,        Value: rawPool},
{Key: feePoolKey,     Value: rawFee},
{Key: creatorFeeKey,  Value: rawCreatorFee},
{Key: resolverFeeKey, Value: rawResolverFee},
		{Key: gTreasuryKey,   Value: rawGTreasury},
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
