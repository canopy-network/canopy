package contract

// handler_claim_winnings.go — MessageClaimWinnings
// Spec: ADLMSR v5.6.6-r2-CORRECTED
//
// Claims payout for a winning (or cancelled/voided) position.
// CRIT-2: gates on STATUS_FINALIZED (= 6), never STATUS_RESOLVED.
// CRIT-1: uses market.QYes/QNo and position.SharesYes/SharesNo (proto names).
// R2: surplus sweep re-reads pool from state after atomic write.
// R6: ClaimantAddress validated in CheckTx AND DeliverTx.
// AUDIT-1: overflow-safe payout formula (ComputePayout).
// AUDIT-6: ghost claimant guard — reject zero position.
//
// CheckTx: market_id 20 bytes, claimant_address 20 bytes. Zero StateRead (AUDIT-8).
// DeliverTx:
//   CheckAutoCancel first
//   Batch read: market, position, pool, claimant account
//   Compute payout based on market status
//   4-key atomic write
//   R2: surplus sweep via fresh pool re-read after write

func (c *Contract) CheckMessageClaimWinnings(msg *MessageClaimWinnings) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
// R6: ClaimantAddress must be validated in CheckTx.
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

// R6: re-validate in DeliverTx.
if len(msg.ClaimantAddress) != 20 {
return &PluginDeliverResponse{Error: ErrInvalidAddress()}
}

if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// ── Batch read ────────────────────────────────────────────────────────────
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

// AUDIT-6: ghost claimant guard — reject zero position.
if position.SharesYes == 0 && position.SharesNo == 0 && position.CostPaid == 0 {
return &PluginDeliverResponse{Error: ErrNoPosition()}
}
if position.Claimed {
return &PluginDeliverResponse{Error: ErrAlreadyClaimed()}
}

// ── Compute payout ────────────────────────────────────────────────────────
var payout uint64

switch market.Status {
case STATUS_FINALIZED:
// CRIT-2: gate on STATUS_FINALIZED, never STATUS_RESOLVED.
// Read outcome to determine winning side.
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
if len(r.Entries) == 0 {
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

// CRIT-1: market.QYes/QNo, position.SharesYes/SharesNo (proto names).
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
// AUDIT-1: overflow-safe payout formula.
payout = ComputePayout(marketPool.Amount, winnerShares, totalWinShares)
}

case STATUS_CANCELLED:
payout = position.CostPaid

case STATUS_VOIDED:
// Tier-4 void: full refund to all bettors.
payout = position.CostPaid

default:
return &PluginDeliverResponse{Error: ErrMarketNotResolved()}
}

if payout > marketPool.Amount {
return &PluginDeliverResponse{Error: ErrInsufficientPoolFunds()}
}

// ── Mutate in memory ──────────────────────────────────────────────────────
position.Claimed     = true
market.ClaimedCount++
marketPool.Amount   -= payout
claimantAcc.Amount  += payout

// ── Marshal all ───────────────────────────────────────────────────────────
rawPos, pe := SafeMarshal(position)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawMkt, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawMP, pe := SafeMarshal(marketPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawAcc, pe := SafeMarshal(claimantAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// ── 4-key atomic write ────────────────────────────────────────────────────
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

// ── R2: surplus sweep — re-read pool from state after atomic write ────────
// Never use in-memory marketPool.Amount here — R2 fix.
graceEnd := market.ExpiryTime + RESOLUTION_DELAY_BLOCKS +
GRACE_PERIOD_BLOCKS + CLAIM_GRACE_PERIOD
shouldSweep :=
(market.Status == STATUS_FINALIZED &&
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
if r.QueryId == sweepQId && len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, sweepPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if sweepPool.Amount > 0 {
// Move remaining pool balance to treasury.
sweepPool.Amount = 0
rawSweep, pe := SafeMarshal(sweepPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
sweepWr, sweepErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: poolKey, Value: rawSweep},
},
})
if pe := errCheckWrite(sweepWr, sweepErr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

return &PluginDeliverResponse{}
}
