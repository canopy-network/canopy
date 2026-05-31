package contract

// handler_claim_investor_reward.go — MessageClaimInvestorReward
// Spec: PRIS v1.0-r3
// Signer: PRAXIS_INVESTOR_ADDR — 2-week vesting cooldown (241,920 blocks)

func (c *Contract) CheckMessageClaimInvestorReward(msg *MessageClaimInvestorReward) *PluginCheckResponse {
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{PRAXIS_INVESTOR_ADDR},
}
}

func (c *Contract) DeliverMessageClaimInvestorReward(msg *MessageClaimInvestorReward, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

poolQId := nextQueryId()
lastQId := nextQueryId()
accQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId, Key: KeyForInvestorPool()},
{QueryId: lastQId, Key: KeyForInvestorLastClaimed()},
{QueryId: accQId,  Key: KeyForAccount(PRAXIS_INVESTOR_ADDR)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

pool        := &Pool{}
lastClaimed := uint64(0)
acc         := &Account{}

for _, r := range resp.Results {
if len(r.Entries) == 0 { continue }
switch r.QueryId {
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, pool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case lastQId:
lc := &LastClaimedBlock{}
if pe := Unmarshal(r.Entries[0].Value, lc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
lastClaimed = lc.Height
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if lastClaimed > 0 && height < lastClaimed+PRIS_INVESTOR_VESTING_BLOCKS {
return &PluginDeliverResponse{Error: ErrCooldownNotElapsed()}
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
rawLast, pe := SafeMarshal(&LastClaimedBlock{Height: height})
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForInvestorPool(),                Value: rawPool},
{Key: KeyForAccount(PRAXIS_INVESTOR_ADDR), Value: rawAcc},
{Key: KeyForInvestorLastClaimed(),         Value: rawLast},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
