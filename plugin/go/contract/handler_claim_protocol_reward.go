package contract

// handler_claim_protocol_reward.go — MessageClaimProtocolReward
// Spec: PRIS v1.0-r3
// Signer: PRAXIS_PROTOCOL_ADDR — no cooldown (audits/bounties on demand)

func (c *Contract) CheckMessageClaimProtocolReward(msg *MessageClaimProtocolReward) *PluginCheckResponse {
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{PRAXIS_PROTOCOL_ADDR},
}
}

func (c *Contract) DeliverMessageClaimProtocolReward(msg *MessageClaimProtocolReward, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

poolQId := nextQueryId()
accQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId, Key: KeyForProtocolPool()},
{QueryId: accQId,  Key: KeyForAccount(PRAXIS_PROTOCOL_ADDR)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

pool := &Pool{}
acc  := &Account{}

for _, r := range resp.Results {
if len(r.Entries) == 0 { continue }
switch r.QueryId {
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, pool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if pool.Amount == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}

acc.Amount  += pool.Amount
pool.Amount  = 0

rawPool, pe := SafeMarshal(pool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForProtocolPool(),                Value: rawPool},
{Key: KeyForAccount(PRAXIS_PROTOCOL_ADDR), Value: rawAcc},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
