package contract

// handler_claim_unbonded_stake.go — MessageClaimUnbondedStake
// Spec: PRIS v1.0-r3 unstake extension
//
// Releases unbonded stake back to resolver account after unbonding period.
// UnbondingReleaseHeight must be <= currentHeight.

func (c *Contract) CheckMessageClaimUnbondedStake(msg *MessageClaimUnbondedStake) *PluginCheckResponse {
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageClaimUnbondedStake(msg *MessageClaimUnbondedStake, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read ────────────────────────────────────────────────────────
recQId := nextQueryId()
accQId := nextQueryId()

recKey := KeyForResolverRecord(msg.ResolverAddress)
accKey := KeyForAccount(msg.ResolverAddress)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: recQId, Key: recKey},
{QueryId: accQId, Key: accKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var record *ResolverRecord
acc := &Account{}

for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
switch r.QueryId {
case recQId:
record = &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, record); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if record == nil {
return &PluginDeliverResponse{Error: ErrResolverNotFound()}
}
if record.UnbondingAmount == 0 {
return &PluginDeliverResponse{Error: ErrNoUnbondingStake()}
}
if height < record.UnbondingReleaseHeight {
return &PluginDeliverResponse{Error: ErrUnbondingNotComplete()}
}

// ── Release unbonded stake ────────────────────────────────────────────
payout := record.UnbondingAmount
record.UnbondingAmount          = 0
record.UnbondingReleaseHeight   = 0
acc.Amount                      += payout

// ── Pay fee ───────────────────────────────────────────────────────────
if acc.Amount < fee {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}
acc.Amount -= fee

// ── Marshal ───────────────────────────────────────────────────────────
rawRec, pe := SafeMarshal(record)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

// ── 2-key atomic write ────────────────────────────────────────────────
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: recKey, Value: rawRec},
{Key: accKey, Value: rawAcc},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
