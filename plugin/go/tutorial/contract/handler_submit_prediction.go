package contract

// handler_submit_prediction.go — MessageSubmitPrediction
// Spec: ADLMSR v5.6.6-r2-CORRECTED
//
// Purchases shares in a prediction market using LMSR pricing.
// Cost = C(q_new) - C(q_old) where C is the LMSR cost function.
//
// CheckTx:  bettor_address 20 bytes, market_id 20 bytes, shares >= PRECISION_SCALE,
//           max_cost non-zero. Zero StateRead (AUDIT-8).
// DeliverTx:
//   CheckAutoCancel first (AUDIT-5)
//   Re-read position in same batch as market after cancel check (AUDIT-5)
//   AUDIT-3: guard now >= market.OpenTime before any subtraction
//   AUDIT-7: re-validate shares >= PRECISION_SCALE in DeliverTx
//   AUDIT-12: finalCost <= max_cost slippage guard
//   4-key atomic write: market, position, pool, bettor account

func (c *Contract) CheckMessageSubmitPrediction(msg *MessageSubmitPrediction) *PluginCheckResponse {
if len(msg.BettorAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
// AUDIT-7: stateless shares floor check in CheckTx too.
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

// AUDIT-7: re-validate in DeliverTx — belt-and-suspenders.
if msg.Shares < PRECISION_SCALE {
return &PluginDeliverResponse{Error: ErrSharesBelowMinimum()}
}

// AUDIT-5: CheckAutoCancel before reading position.
if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// AUDIT-5: re-read market and position together after cancel check.
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

// AUDIT-3: guard now >= OpenTime before any subtraction involving heights.
if now < market.OpenTime {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}

// ── LMSR cost calculation ─────────────────────────────────────────────────
tradeCost, pe := ComputeTradeCost(market.QYes, market.QNo, market.BEff, msg.Shares, msg.Outcome)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// AUDIT-12: finalCost overflow check before MaxCost comparison.
if fee > 0 && tradeCost > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
finalCost := tradeCost + fee

// AUDIT-12: slippage protection — reject if cost exceeds bettor's limit.
if finalCost > msg.MaxCost {
return &PluginDeliverResponse{Error: ErrCostExceedsMaxCost()}
}

if bettor.Amount < finalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

// ── Mutate in memory ──────────────────────────────────────────────────────
bettor.Amount  -= finalCost
mPool.Amount   += tradeCost
feePool.Amount += fee

// Update LMSR q-values.
if msg.Outcome {
market.QYes += msg.Shares
position.SharesYes += msg.Shares
} else {
market.QNo += msg.Shares
position.SharesNo += msg.Shares
}
position.CostPaid += tradeCost

// Track total unique positions for surplus sweep in ClaimWinnings.
// Only increment if this is the first prediction from this address.
if position.SharesYes == msg.Shares && !msg.Outcome {
// first prediction was NO
market.TotalPositions++
} else if position.SharesNo == msg.Shares && msg.Outcome {
// first prediction was YES — but SharesNo is 0 so check differently
market.TotalPositions++
} else if position.SharesYes == msg.Shares && msg.Outcome && position.SharesNo == 0 {
market.TotalPositions++
} else if position.SharesNo == msg.Shares && !msg.Outcome && position.SharesYes == 0 {
market.TotalPositions++
}

// Update elevated risk flag.
market.ElevatedRisk = IsElevatedRisk(mPool.Amount)

// ── Marshal all ───────────────────────────────────────────────────────────
rawMarket, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawPos, pe := SafeMarshal(position)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawPool, pe := SafeMarshal(mPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawFee, pe := SafeMarshal(feePool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

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
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
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
