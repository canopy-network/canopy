package contract

// handler_cancel_market.go — MessageCancelMarket
// Spec: Praxis core
//
// Allows a market creator to cancel their own market before expiry.
// Conditions:
//   - Market must be STATUS_OPEN
//   - now < market.ExpiryTime (not yet expired)
//   - Signer must be market.Creator
//   - TotalPositions must be 0 (no bets placed — Option 2)
//
// On cancel:
//   - Market status → STATUS_CANCELLED
//   - Creator receives CreatorBond + TreasuryReserve.LockedReserve back
//   - Creator fee pool + resolver fee pool swept to KeyForTreasuryPool()
//   - Market pool remains — bettors claim refunds via claim_winnings (STATUS_CANCELLED path)

func (c *Contract) CheckMessageCancelMarket(msg *MessageCancelMarket) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.CreatorAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.CreatorAddress},
}
}

func (c *Contract) DeliverMessageCancelMarket(msg *MessageCancelMarket, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read ────────────────────────────────────────────────────────
marketQId     := nextQueryId()
treasQId      := nextQueryId()
creatorAccQId := nextQueryId()
creatorFeeQId := nextQueryId()
resolverFeeQId := nextQueryId()
gTreasuryQId  := nextQueryId()
feeQId        := nextQueryId()

marketKey      := KeyForMarket(msg.MarketId)
treasKey       := KeyForTreasuryReserve(msg.MarketId)
creatorAccKey  := KeyForAccount(msg.CreatorAddress)
creatorFeeKey  := KeyForCreatorFeePool(msg.MarketId)
resolverFeeKey := KeyForResolverFeePool(msg.MarketId)
gTreasuryKey   := KeyForTreasuryPool()
feePoolKey     := KeyForFeePool(c.Config.ChainId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,      Key: marketKey},
{QueryId: treasQId,       Key: treasKey},
{QueryId: creatorAccQId,  Key: creatorAccKey},
{QueryId: creatorFeeQId,  Key: creatorFeeKey},
{QueryId: resolverFeeQId, Key: resolverFeeKey},
{QueryId: gTreasuryQId,   Key: gTreasuryKey},
{QueryId: feeQId,         Key: feePoolKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market *MarketState
treas       := &TreasuryReserve{}
creatorAcc  := &Account{}
creatorFee  := &Pool{}
resolverFee := &Pool{}
gTreasury   := &Pool{}
feePool     := &Pool{}

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
case treasQId:
if pe := Unmarshal(r.Entries[0].Value, treas); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case creatorAccQId:
if pe := Unmarshal(r.Entries[0].Value, creatorAcc); pe != nil {
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
if now >= market.ExpiryTime {
return &PluginDeliverResponse{Error: ErrMarketExpired()}
}
if !bytesEqual(msg.CreatorAddress, market.Creator) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}
// Option 2: block cancel if any positions exist
if market.TotalPositions > 0 {
return &PluginDeliverResponse{Error: ErrMarketHasPositions()}
}

// ── Compute refund ────────────────────────────────────────────────────
// Creator gets back: CreatorBond + LockedReserve (finalization bounty)
refund := treas.CreatorBond + treas.LockedReserve

// ── Mutate ────────────────────────────────────────────────────────────
market.Status       = STATUS_CANCELLED
treas.CreatorBond   = 0
treas.LockedReserve = 0
creatorAcc.Amount  += refund

// Sweep creator fee pool + resolver fee pool to global treasury
gTreasury.Amount   += creatorFee.Amount + resolverFee.Amount
creatorFee.Amount   = 0
resolverFee.Amount  = 0

// TX fee split 50/50
feePool.Amount    += fee / 2
gTreasury.Amount  += fee - fee/2

// ── Marshal ───────────────────────────────────────────────────────────
rawMarket, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawTreas, pe := SafeMarshal(treas)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawCreatorAcc, pe := SafeMarshal(creatorAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawCreatorFee, pe := SafeMarshal(creatorFee)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawResolverFee, pe := SafeMarshal(resolverFee)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawGTreasury, pe := SafeMarshal(gTreasury)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

// ── 7-key atomic write ────────────────────────────────────────────────
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: marketKey,      Value: rawMarket},
{Key: treasKey,       Value: rawTreas},
{Key: creatorAccKey,  Value: rawCreatorAcc},
{Key: creatorFeeKey,  Value: rawCreatorFee},
{Key: resolverFeeKey, Value: rawResolverFee},
{Key: gTreasuryKey,   Value: rawGTreasury},
{Key: feePoolKey,     Value: rawFee},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
