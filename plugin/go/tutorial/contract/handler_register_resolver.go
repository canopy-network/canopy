package contract

func (c *Contract) CheckMessageRegisterResolver(msg *MessageRegisterResolver) *PluginCheckResponse {
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
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
feeQId          := nextQueryId()
	gTreasuryQId    := nextQueryId()
	ridxQId         := nextQueryId()

recKey     := KeyForResolverRecord(msg.ResolverAddress)
accKey     := KeyForAccount(msg.ResolverAddress)
feePoolKey      := KeyForFeePool(c.Config.ChainId)
	gTreasuryKey    := KeyForTreasuryPool()
	ridxKey         := KeyForResolverIndex()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: recQId, Key: recKey},
{QueryId: accQId, Key: accKey},
{QueryId: feeQId,       Key: feePoolKey},
		{QueryId: gTreasuryQId, Key: gTreasuryKey},
		{QueryId: ridxQId,       Key: ridxKey},
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
feePool     := &Pool{}
	gTreasury   := &Pool{}
	ridx        := &ResolverIndex{}

for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
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
case gTreasuryQId:
			if pe := Unmarshal(r.Entries[0].Value, gTreasury); pe != nil {
				return &PluginDeliverResponse{Error: pe}
			}
		case ridxQId:
			if pe := Unmarshal(r.Entries[0].Value, ridx); pe != nil {
				return &PluginDeliverResponse{Error: pe}
			}
}
}
if existing == nil && msg.StakeAmount < MIN_RESOLVER_STAKE {
	return &PluginDeliverResponse{Error: ErrInsufficientResolverStake()}
}
if fee > 0 && msg.StakeAmount > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.StakeAmount + fee
if account.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

account.Amount -= totalCost
feePool.Amount  += fee / 2
	gTreasury.Amount += fee - fee/2

var record *ResolverRecord
if existing != nil {
existing.StakeAmount += msg.StakeAmount
		if existing.StakeAmount < MIN_RESOLVER_STAKE {
			return &PluginDeliverResponse{Error: ErrInsufficientResolverStake()}
		}
existing.IsActive = true // re-activate on re-registration after full exit
		if existing.RrsScore < PRIS_RRS_INITIAL {
			existing.RrsScore = PRIS_RRS_INITIAL
		}
record = existing
} else {
record = &ResolverRecord{
ResolverAddress: msg.ResolverAddress,
StakeAmount:     msg.StakeAmount,
RrsScore:        PRIS_RRS_INITIAL,
RegisteredAt:    now,
IsActive:        true,
}
}

rawRec, pe := SafeMarshal(record)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawGTreasury, pe := SafeMarshal(gTreasury)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawRidx, pe := SafeMarshal(ridx)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: recKey,     Value: rawRec},
{Key: feePoolKey,    Value: rawFee},
	{Key: gTreasuryKey,  Value: rawGTreasury},
	{Key: ridxKey,       Value: rawRidx},
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
