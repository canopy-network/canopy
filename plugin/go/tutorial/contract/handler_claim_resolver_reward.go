package contract

// handler_claim_resolver_reward.go — MessageClaimResolverReward
// Spec: PRIS v1.0-r3
//
// Pays a resolver their weighted share of the epoch resolver pool.
// Share formula:
//   share = resolverEpochPool * (resolver_resolutions * tier_weight) / SUM(all * tier_weight)
//
// Qualifying conditions:
//   - RRS > 0
//   - SuccessfulResolutions > 0 in the epoch
//   - LastClaimedEpoch < epoch
//
// r3 fix R3-4: qualification is RRS > 0 (not >= 10) — Bronze tier starts at 1.

func rrsWeight(rrs uint64) uint32 {
if rrs >= RRS_GOLD_THRESHOLD {
return VOTE_WEIGHT_GOLD
} else if rrs >= RRS_SILVER_THRESHOLD {
return VOTE_WEIGHT_SILVER
}
return VOTE_WEIGHT_BRONZE
}

func (c *Contract) CheckMessageClaimResolverReward(msg *MessageClaimResolverReward) *PluginCheckResponse {
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageClaimResolverReward(msg *MessageClaimResolverReward, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

currentEpoch := height / PRIS_EPOCH_BLOCKS
claimEpoch   := msg.Epoch

// Can only claim past epochs — current epoch not yet snapshotted.
if claimEpoch >= currentEpoch {
return &PluginDeliverResponse{Error: ErrInvalidParam()}
}

recQId      := nextQueryId()
poolQId     := nextQueryId()
accQId      := nextQueryId()
statsQId    := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: recQId,   Key: KeyForResolverRecord(msg.ResolverAddress)},
{QueryId: poolQId,  Key: KeyForResolverEpochPool(claimEpoch)},
{QueryId: accQId,   Key: KeyForAccount(msg.ResolverAddress)},
{QueryId: statsQId, Key: KeyForGlobalStats()},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

rec   := &ResolverRecord{}
pool  := &Pool{}
acc   := &Account{}
stats := &GlobalStats{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case recQId:
if pe := Unmarshal(r.Entries[0].Value, rec); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, pool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case statsQId:
if pe := Unmarshal(r.Entries[0].Value, stats); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

// Qualification checks
if rec.RrsScore == 0 {
return &PluginDeliverResponse{Error: ErrInsufficientRRS()}
}
if rec.SuccessfulResolutions == 0 {
return &PluginDeliverResponse{Error: ErrNoResolutions()}
}
if rec.LastClaimedEpoch >= claimEpoch {
return &PluginDeliverResponse{Error: ErrAlreadyClaimed()}
}
if pool.Amount == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}
if stats.TotalWeightedResolutions == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}

// Compute weighted share
weight  := uint64(rrsWeight(rec.RrsScore))
myScore := rec.SuccessfulResolutions * weight
payout  := pool.Amount * myScore / stats.TotalWeightedResolutions

if payout == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}
if payout > pool.Amount {
payout = pool.Amount
}

acc.Amount          += payout
pool.Amount         -= payout
rec.LastClaimedEpoch = claimEpoch

rawRec, pe := SafeMarshal(rec)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPool, pe := SafeMarshal(pool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForResolverRecord(msg.ResolverAddress),  Value: rawRec},
{Key: KeyForResolverEpochPool(claimEpoch),        Value: rawPool},
{Key: KeyForAccount(msg.ResolverAddress),         Value: rawAcc},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
