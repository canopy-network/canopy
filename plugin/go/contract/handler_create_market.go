package contract

func (c *Contract) CheckMessageCreateMarket(msg *MessageCreateMarket) *PluginCheckResponse {
println("PRAXIS CHECK: addr=", len(msg.CreatorAddress), "b0=", msg.B0, "expiry=", msg.ExpiryTime, "nonce=", msg.Nonce, "q=", len(msg.Question))
if len(msg.CreatorAddress) != 20 {
println("PRAXIS CHECK FAIL: invalid address")
return ErrCheckResp(ErrInvalidAddress())
}
if msg.B0 < MIN_B0 {
println("PRAXIS CHECK FAIL: invalid b0")
return ErrCheckResp(ErrInvalidB0())
}
if msg.ExpiryTime == 0 {
println("PRAXIS CHECK FAIL: expiry==0")
return ErrCheckResp(ErrInvalidParam())
}
if msg.ExpiryTime > MAX_EXPIRY_TIME {
println("PRAXIS CHECK FAIL: expiry too large")
return ErrCheckResp(ErrExpiryTooLarge())
}
if msg.Nonce == 0 {
println("PRAXIS CHECK FAIL: nonce==0")
return ErrCheckResp(ErrInvalidNonce())
}
if len(msg.Question) == 0 {
println("PRAXIS CHECK FAIL: no question")
return ErrCheckResp(ErrInvalidQuestion())
}
println("PRAXIS CHECK OK")
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.CreatorAddress},
}
}

func (c *Contract) DeliverMessageCreateMarket(msg *MessageCreateMarket, fee uint64) *PluginDeliverResponse {
println("PRAXIS DELIVER START: now=", GetGlobalHeight(), "fsmId=", c.fsmId)
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}
if msg.ExpiryTime > MAX_EXPIRY_TIME {
return &PluginDeliverResponse{Error: ErrExpiryTooLarge()}
}

marketId := DeriveMarketId(msg.CreatorAddress, msg.Nonce)

marketQId  := nextQueryId()
creatorQId := nextQueryId()
feeQId     := nextQueryId()
println("PRAXIS: queryIds market=", marketQId, "creator=", creatorQId, "fee=", feeQId)

marketKey  := KeyForMarket(marketId)
creatorKey := KeyForAccount(msg.CreatorAddress)
feePoolKey := KeyForFeePool(c.Config.ChainId)
poolKey    := KeyForMarketPool(marketId)
treasKey   := KeyForTreasuryReserve(marketId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,  Key: marketKey},
{QueryId: creatorQId, Key: creatorKey},
{QueryId: feeQId,     Key: feePoolKey},
},
})
if err != nil {
println("PRAXIS DELIVER: StateRead err=", err.Msg)
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
println("PRAXIS DELIVER: resp.Error=", resp.Error.Msg)
return &PluginDeliverResponse{Error: resp.Error}
}

creator := &Account{}
feePool := &Pool{}

for _, r := range resp.Results {
println("PRAXIS: result queryId=", r.QueryId, "entries=", len(r.Entries))
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
switch r.QueryId {
case marketQId:
return &PluginDeliverResponse{Error: ErrInvalidParam()} // already exists
case creatorQId:
if pe := Unmarshal(r.Entries[0].Value, creator); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feeQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

println("PRAXIS: past state read loop, checking funds")
	if fee > 0 && msg.B0 > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.B0 + fee
if creator.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

creator.Amount -= totalCost
feePool.Amount += fee

halfB0 := msg.B0 / 2
market := &MarketState{
Status:        STATUS_OPEN,
ExpiryTime:    msg.ExpiryTime,
QYes:          halfB0,
QNo:           halfB0,
BEff:          msg.B0,
Creator:       msg.CreatorAddress,
ClaimedCount:  0,
TotalPositions: 0,
OpenTime:      now,
ElevatedRisk:  false,
}

pool := &Pool{Id: c.Config.ChainId, Amount: msg.B0}
treasury := &TreasuryReserve{LockedReserve: 0}

rawMarket, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawCreator, pe := SafeMarshal(creator)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPool, pe := SafeMarshal(pool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawTreasury, pe := SafeMarshal(treasury)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: marketKey,  Value: rawMarket},
{Key: poolKey,    Value: rawPool},
{Key: treasKey,   Value: rawTreasury},
{Key: feePoolKey, Value: rawFee},
}
var deletes []*PluginDeleteOp
if creator.Amount > 0 {
sets = append(sets, &PluginSetOp{Key: creatorKey, Value: rawCreator})
} else {
deletes = []*PluginDeleteOp{{Key: creatorKey}}
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
