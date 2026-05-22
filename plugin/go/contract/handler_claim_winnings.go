package contract

func (c *Contract) CheckMessageClaimWinnings(msg *MessageClaimWinnings) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.ClaimantAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ClaimantAddress},
}
}

func (c *Contract) DeliverMessageClaimWinnings(msg *MessageClaimWinnings, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}
if len(msg.ClaimantAddress) != 20 {
return &PluginDeliverResponse{Error: ErrInvalidAddress()}
}
if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

marketQId   := nextQueryId()
posQId      := nextQueryId()
poolQId     := nextQueryId()
claimAccQId := nextQueryId()

marketKey   := KeyForMarket(msg.MarketId)
posKey      := KeyForPosition(msg.MarketId, msg.ClaimantAddress)
poolKey     := KeyForMarketPool(msg.MarketId)
claimKey    := KeyForAccount(msg.ClaimantAddress)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: marketKey},
{QueryId: posQId,      Key: posKey},
{QueryId: poolQId,     Key: poolKey},
{QueryId: claimAccQId, Key: claimKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market   *MarketState
position    := &PositionState{}
marketPool  := &Pool{}
claimantAcc := &Account{}

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
if pe := Unmarshal(r.Entries[0].Value, marketPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case claimAccQId:
if pe := Unmarshal(r.Entries[0].Value, claimantAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if position.SharesYes == 0 && position.SharesNo == 0 && position.CostPaid == 0 {
return &PluginDeliverResponse{Error: ErrNoPosition()}
}
if position.Claimed {
return &PluginDeliverResponse{Error: ErrAlreadyClaimed()}
}

var payout uint64
switch market.Status {
case STATUS_FINALIZED:
outQId := nextQueryId()
outResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: outQId, Key: KeyForOutcome(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if outResp.Error != nil {
return &PluginDeliverResponse{Error: outResp.Error}
}

var outcome *OutcomeState
for _, r := range outResp.Results {
if r.QueryId == outQId {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrInternal()}
}
outcome = &OutcomeState{}
if pe := Unmarshal(r.Entries[0].Value, outcome); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
if outcome == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}

var winnerShares, totalWinShares uint64
if outcome.WinningOutcome {
winnerShares   = position.SharesYes
totalWinShares = market.QYes
} else {
winnerShares   = position.SharesNo
totalWinShares = market.QNo
}
if winnerShares > 0 {
if totalWinShares == 0 {
return &PluginDeliverResponse{Error: ErrInternal()}
}
payout = ComputePayout(marketPool.Amount, winnerShares, totalWinShares)
}

case STATUS_CANCELLED:
payout = position.CostPaid

case STATUS_VOIDED:
payout = position.CostPaid

default:
return &PluginDeliverResponse{Error: ErrMarketNotResolved()}
}

if payout > marketPool.Amount {
return &PluginDeliverResponse{Error: ErrInsufficientPoolFunds()}
}

position.Claimed     = true
market.ClaimedCount++
marketPool.Amount   -= payout
claimantAcc.Amount  += payout

rawPos, pe := SafeMarshal(position)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawMkt, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawMP, pe := SafeMarshal(marketPool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(claimantAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: posKey,    Value: rawPos},
{Key: marketKey, Value: rawMkt},
{Key: poolKey,   Value: rawMP},
{Key: claimKey,  Value: rawAcc},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

graceEnd := market.ExpiryTime + RESOLUTION_DELAY_BLOCKS + GRACE_PERIOD_BLOCKS + CLAIM_GRACE_PERIOD
shouldSweep := (market.Status == STATUS_FINALIZED &&
(market.ClaimedCount == market.TotalPositions || now > graceEnd)) ||
(market.Status == STATUS_CANCELLED && now > graceEnd)
if shouldSweep {
sweepQId := nextQueryId()
sweepResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: sweepQId, Key: poolKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if sweepResp.Error != nil {
return &PluginDeliverResponse{Error: sweepResp.Error}
}

sweepPool := &Pool{}
for _, r := range sweepResp.Results {
if r.QueryId == sweepQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, sweepPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if sweepPool.Amount > 0 {
// Read the global treasury pool before writing.
tQId := nextQueryId()
tResp, tErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: tQId, Key: KeyForTreasuryPool()},
},
})
if tErr != nil {
return &PluginDeliverResponse{Error: tErr}
}
if tResp.Error != nil {
return &PluginDeliverResponse{Error: tResp.Error}
}

treasuryPool := &Pool{}
for _, r := range tResp.Results {
if r.QueryId == tQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, treasuryPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

// Overflow guard.
if treasuryPool.Amount > ^uint64(0)-sweepPool.Amount {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
treasuryPool.Amount += sweepPool.Amount
sweepPool.Amount = 0

rawSweep, pe := SafeMarshal(sweepPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawTreasury, pe := SafeMarshal(treasuryPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
// Write zeroed market pool and updated treasury pool atomically.
sweepWr, sweepErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: poolKey,              Value: rawSweep},
{Key: KeyForTreasuryPool(), Value: rawTreasury},
},
})
if pe := errCheckWrite(sweepWr, sweepErr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

return &PluginDeliverResponse{}
}
