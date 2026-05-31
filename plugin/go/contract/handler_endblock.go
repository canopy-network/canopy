package contract

// handler_endblock.go — PRIS v1.0-r3 EndBlock epoch boundary processor
//
// Triggered inside EndBlock every PRIS_EPOCH_BLOCKS blocks.
// r3 fix R3-2: epoch snapshot is NOT triggered by user TXs — it fires
// in EndBlock which executes every block regardless of TX count.
//
// On epoch boundary:
//   1. Read treasury pool balance
//   2. Write immutable EpochSnapshot
//   3. Carve 5 distribution pools atomically (20% each)
//   4. Zero treasury pool
//
// EndBlock must never return an error that halts the chain.


func (c *Contract) processEpochBoundary(height uint64) *PluginError {
epoch := height / PRIS_EPOCH_BLOCKS

treasuryQId := nextQueryId()
resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: treasuryQId, Key: KeyForTreasuryPool()},
},
})
if err != nil {
return err
}
if resp.Error != nil {
return resp.Error
}

treasury := &Pool{}
for _, r := range resp.Results {
if r.QueryId == treasuryQId && len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, treasury); pe != nil {
return pe
}
}
}

if treasury.Amount == 0 {
return nil
}

resolverShare  := ComputeBps(treasury.Amount, PRIS_RESOLVER_SHARE_BPS)
builderShare   := ComputeBps(treasury.Amount, PRIS_BUILDER_SHARE_BPS)
communityShare := ComputeBps(treasury.Amount, PRIS_COMMUNITY_SHARE_BPS)
investorShare  := ComputeBps(treasury.Amount, PRIS_INVESTOR_SHARE_BPS)
protocolShare  := ComputeBps(treasury.Amount, PRIS_PROTOCOL_SHARE_BPS)

bPoolQId := nextQueryId()
cPoolQId := nextQueryId()
iPoolQId := nextQueryId()
pPoolQId := nextQueryId()
rPoolQId := nextQueryId()

poolResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: bPoolQId, Key: KeyForBuilderPool()},
{QueryId: cPoolQId, Key: KeyForCommunityPool()},
{QueryId: iPoolQId, Key: KeyForInvestorPool()},
{QueryId: pPoolQId, Key: KeyForProtocolPool()},
{QueryId: rPoolQId, Key: KeyForResolverEpochPool(epoch)},
},
})
if err != nil {
return err
}

builderPool   := &Pool{}
communityPool := &Pool{}
investorPool  := &Pool{}
protocolPool  := &Pool{}
resolverPool  := &Pool{}

if poolResp != nil && poolResp.Error == nil {
for _, r := range poolResp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case bPoolQId:
_ = Unmarshal(r.Entries[0].Value, builderPool)
case cPoolQId:
_ = Unmarshal(r.Entries[0].Value, communityPool)
case iPoolQId:
_ = Unmarshal(r.Entries[0].Value, investorPool)
case pPoolQId:
_ = Unmarshal(r.Entries[0].Value, protocolPool)
case rPoolQId:
_ = Unmarshal(r.Entries[0].Value, resolverPool)
}
}
}

builderPool.Amount   += builderShare
communityPool.Amount += communityShare
investorPool.Amount  += investorShare
protocolPool.Amount  += protocolShare
resolverPool.Amount  += resolverShare

snapshot := &Pool{Amount: treasury.Amount}
rawSnap, pe := SafeMarshal(snapshot)
if pe != nil { return pe }

rawBuilder, pe := SafeMarshal(builderPool)
if pe != nil { return pe }
rawCommunity, pe := SafeMarshal(communityPool)
if pe != nil { return pe }
rawInvestor, pe := SafeMarshal(investorPool)
if pe != nil { return pe }
rawProtocol, pe := SafeMarshal(protocolPool)
if pe != nil { return pe }
rawResolver, pe := SafeMarshal(resolverPool)
if pe != nil { return pe }

emptyTreasury := &Pool{Amount: 0}
rawTreasury, pe := SafeMarshal(emptyTreasury)
if pe != nil { return pe }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForEpochSnapshot(epoch),    Value: rawSnap},
{Key: KeyForBuilderPool(),            Value: rawBuilder},
{Key: KeyForCommunityPool(),          Value: rawCommunity},
{Key: KeyForInvestorPool(),           Value: rawInvestor},
{Key: KeyForProtocolPool(),           Value: rawProtocol},
{Key: KeyForResolverEpochPool(epoch), Value: rawResolver},
{Key: KeyForTreasuryPool(),           Value: rawTreasury},
},
})
return errCheckWrite(wr, werr)
}
