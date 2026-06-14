package contract

// handler_claim_builder_reward.go — MessageClaimBuilderReward
// Spec: PRIS v1.0-r3
//
// Transfers the full KeyForBuilderPool() balance to PRAXIS_BUILDER_ADDR.
// Cooldown: PRIS_BUILDER_EPOCH_BLOCKS (120,960 blocks ~ 7 days)
// Signer:   PRAXIS_BUILDER_ADDR only
//
// r3 fix R3-3: cooldown tracked in KeyForBuilderLastClaimed() singleton.
// r3 fix R3-5: claims full pool balance — rewards compound across missed epochs.

func (c *Contract) CheckMessageClaimBuilderReward(msg *MessageClaimBuilderReward) *PluginCheckResponse {
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{PRAXIS_BUILDER_ADDR},
}
}

func (c *Contract) DeliverMessageClaimBuilderReward(msg *MessageClaimBuilderReward, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

poolQId := nextQueryId()
lastQId := nextQueryId()
accQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId, Key: KeyForBuilderPool()},
{QueryId: lastQId, Key: KeyForBuilderLastClaimed()},
{QueryId: accQId,  Key: KeyForAccount(PRAXIS_BUILDER_ADDR)},
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
if len(r.Entries) == 0 {
continue
}
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

if lastClaimed > 0 && height < lastClaimed+PRIS_BUILDER_EPOCH_BLOCKS {
return &PluginDeliverResponse{Error: ErrCooldownNotElapsed()}
}

if pool.Amount == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}

payout     := pool.Amount
acc.Amount += payout
pool.Amount = 0

rawPool, pe := SafeMarshal(pool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawLast, pe := SafeMarshal(&LastClaimedBlock{Height: height})
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForBuilderPool(),                Value: rawPool},
{Key: KeyForAccount(PRAXIS_BUILDER_ADDR), Value: rawAcc},
{Key: KeyForBuilderLastClaimed(),         Value: rawLast},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
