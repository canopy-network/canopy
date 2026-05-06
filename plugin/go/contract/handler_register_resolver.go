package contract

func (c *Contract) CheckMessageRegisterResolver(msg *MessageRegisterResolver) *PluginCheckResponse {
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if msg.StakeAmount == 0 {
return ErrCheckResp(ErrInvalidAmount())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageRegisterResolver(msg *MessageRegisterResolver, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

recQId     := nextQueryId()
accQId     := nextQueryId()
feeQId     := nextQueryId()

recKey     := KeyForResolverRecord(msg.ResolverAddress)
accKey     := KeyForAccount(msg.ResolverAddress)
feePoolKey := KeyForFeePool(c.Config.ChainId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: recQId, Key: recKey},
{QueryId: accQId, Key: accKey},
{QueryId: feeQId, Key: feePoolKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var existing *ResolverRecord
account := &Account{}
feePool := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case recQId:
existing = &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, existing); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, account); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feeQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if fee > 0 && msg.StakeAmount > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.StakeAmount + fee
if account.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

account.Amount -= totalCost
feePool.Amount += fee

var record *ResolverRecord
if existing != nil {
existing.StakeAmount += msg.StakeAmount
record = existing
} else {
record = &ResolverRecord{
ResolverAddress: msg.ResolverAddress,
StakeAmount:     msg.StakeAmount,
RrsScore:        RRS_INITIAL,
RegisteredAt:    now,
}
}

rawRec, pe := SafeMarshal(record)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: recKey,     Value: rawRec},
{Key: feePoolKey, Value: rawFee},
}
var deletes []*PluginDeleteOp
if account.Amount == 0 {
deletes = []*PluginDeleteOp{{Key: accKey}}
} else {
rawAcc, pe := SafeMarshal(account)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: accKey, Value: rawAcc})
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
