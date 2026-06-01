package contract

func (c *Contract) CheckMessageReclaimStake(msg *MessageReclaimStake) *PluginCheckResponse {
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

func (c *Contract) DeliverMessageReclaimStake(msg *MessageReclaimStake, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId   := nextQueryId()
posQId      := nextQueryId()
poolQId     := nextQueryId()
treasQId    := nextQueryId()
proposalQId := nextQueryId()
claimAccQId := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: posQId,      Key: KeyForPosition(msg.MarketId, msg.ClaimantAddress)},
{QueryId: poolQId,     Key: KeyForMarketPool(msg.MarketId)},
{QueryId: treasQId,    Key: KeyForTreasuryReserve(msg.MarketId)},
{QueryId: proposalQId, Key: KeyForProposal(msg.MarketId)},
{QueryId: claimAccQId, Key: KeyForAccount(msg.ClaimantAddress)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market   *MarketState
var position *PositionState
marketPool  := &Pool{}
treasury    := &TreasuryReserve{}
claimantAcc := &Account{}
var proposalRaw []byte

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
case posQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
position = &PositionState{}
if pe := Unmarshal(r.Entries[0].Value, position); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case poolQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, marketPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case treasQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, treasury); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case proposalQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
proposalRaw = r.Entries[0].Value
}
case claimAccQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, claimantAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}

// Only reclaimable if STATUS_OPEN, expiry passed, and no proposal ever filed
if market.Status != STATUS_OPEN {
return &PluginDeliverResponse{Error: ErrMarketNotReclaimable()}
}
if proposalRaw != nil {
return &PluginDeliverResponse{Error: ErrMarketNotReclaimable()}
}
reclaimOpen := market.ExpiryTime + RESOLUTION_DELAY_BLOCKS + GRACE_PERIOD_BLOCKS
if now <= reclaimOpen {
return &PluginDeliverResponse{Error: ErrReclaimWindowClosed()}
}

// Compute refund amount
var refund uint64

// Position refund (any bettor including creator)
if position != nil {
if position.Claimed {
return &PluginDeliverResponse{Error: ErrAlreadyClaimed()}
}
if position.SharesYes > 0 || position.SharesNo > 0 {
refund += position.CostPaid
}
}

// Creator gets TreasuryReserve (FINALIZATION_BOUNTY) back — no resolver showed up
if bytesEqual(msg.ClaimantAddress, market.Creator) && treasury.LockedReserve > 0 {
refund += treasury.LockedReserve
}

if refund == 0 {
return &PluginDeliverResponse{Error: ErrNoStakeToReclaim()}
}
if refund > marketPool.Amount {
return &PluginDeliverResponse{Error: ErrInsufficientPoolFunds()}
}

// Mutate in memory
claimantAcc.Amount += refund
marketPool.Amount  -= refund
isCreator := bytesEqual(msg.ClaimantAddress, market.Creator)
if isCreator {
treasury.LockedReserve = 0
}

// Marshal all
rawAcc, pe := SafeMarshal(claimantAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPool, pe := SafeMarshal(marketPool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: KeyForAccount(msg.ClaimantAddress), Value: rawAcc},
{Key: KeyForMarketPool(msg.MarketId),     Value: rawPool},
}
var deletes []*PluginDeleteOp

// Mark position as claimed to prevent double reclaim
if position != nil && (position.SharesYes > 0 || position.SharesNo > 0) {
position.Claimed = true
rawPos, pe := SafeMarshal(position)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: KeyForPosition(msg.MarketId, msg.ClaimantAddress), Value: rawPos})
}

if isCreator && treasury.LockedReserve == 0 {
rawTreas, pe := SafeMarshal(treasury)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: KeyForTreasuryReserve(msg.MarketId), Value: rawTreas})
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
