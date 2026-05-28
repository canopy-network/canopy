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
cancelledMarket, cancelErr := c.CheckAutoCancel(msg.MarketId)
if cancelErr != nil {
return &PluginDeliverResponse{Error: cancelErr}
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
// Apply auto-cancel if triggered — status will be STATUS_CANCELLED.
if cancelledMarket != nil {
market = cancelledMarket
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

// Issue-14 fix: compute sweep eligibility BEFORE the atomic write.
// marketPool.Amount is already post-payout in memory — no re-read needed.
// Folding sweep into the same StateWrite eliminates the TOCTOU window where
// two concurrent last-winner calls could both re-read a non-zero pool and
// double-credit treasury.
resolutionDelay := RESOLUTION_DELAY_BLOCKS
gracePeriod     := GRACE_PERIOD_BLOCKS
claimGrace      := CLAIM_GRACE_PERIOD
if TEST_MODE {
resolutionDelay = TEST_RESOLUTION_DELAY
gracePeriod     = TEST_GRACE_PERIOD
claimGrace      = TEST_CLAIM_GRACE_PERIOD
}
graceEnd    := market.ExpiryTime + resolutionDelay + gracePeriod + claimGrace
shouldSweep := (market.Status == STATUS_FINALIZED &&
(market.TotalPositions > 0 && market.ClaimedCount == market.TotalPositions || now > graceEnd)) ||
(market.Status == STATUS_CANCELLED && now > graceEnd)

sets := []*PluginSetOp{
{Key: posKey,    Value: rawPos},
{Key: marketKey, Value: rawMkt},
{Key: poolKey,   Value: rawMP},
{Key: claimKey,  Value: rawAcc},
}

if shouldSweep && marketPool.Amount > 0 {
// marketPool.Amount here is the post-payout remainder (already subtracted above).
// Read treasury once, add remainder, zero the market pool — all in one write.
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
if treasuryPool.Amount > ^uint64(0)-marketPool.Amount {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
treasuryPool.Amount += marketPool.Amount
marketPool.Amount   = 0

// Re-marshal pool (now zeroed) and treasury (now incremented).
rawMP, pe = SafeMarshal(marketPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawTreasury, pe := SafeMarshal(treasuryPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// Replace pool op with zeroed value; append treasury op.
sets[2] = &PluginSetOp{Key: poolKey,              Value: rawMP}
sets    = append(sets, &PluginSetOp{Key: KeyForTreasuryPool(), Value: rawTreasury})
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

return &PluginDeliverResponse{}
}
