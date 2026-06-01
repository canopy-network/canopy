package contract

func (c *Contract) CheckMessageTallyVotes(msg *MessageTallyVotes) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.CallerAddr) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.CallerAddr},
}
}

func (c *Contract) DeliverMessageTallyVotes(msg *MessageTallyVotes, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId  := nextQueryId()
disputeQId := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,  Key: KeyForMarket(msg.MarketId)},
{QueryId: disputeQId, Key: KeyForDispute(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market  *MarketState
var dispute *DisputeRecord

for _, r := range resp.Results {
switch r.QueryId {
case marketQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case disputeQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrNotDisputed()}
}
dispute = &DisputeRecord{}
if pe := Unmarshal(r.Entries[0].Value, dispute); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if market.Status != STATUS_DISPUTED {
return &PluginDeliverResponse{Error: ErrNotDisputed()}
}
if dispute == nil {
return &PluginDeliverResponse{Error: ErrNotDisputed()}
}

if dispute.VoteStatus == VOTE_TALLIED {
return &PluginDeliverResponse{}
}

revealEnd := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS + REVEAL_PHASE_BLOCKS
if now <= revealEnd {
return &PluginDeliverResponse{Error: ErrTallyNotReady()}
}

queries := make([]uint64, 0, len(dispute.PanelMembers))
readKeys := make([]*PluginKeyRead, 0, len(dispute.PanelMembers))

// Layer 2: build RRS read keys alongside vote reveal keys
rrsQueries := make([]uint64, 0, len(dispute.PanelMembers))
rrsKeys    := make([]*PluginKeyRead, 0, len(dispute.PanelMembers))

for _, member := range dispute.PanelMembers {
qId := nextQueryId()
queries = append(queries, qId)
readKeys = append(readKeys, &PluginKeyRead{
QueryId: qId,
Key:     KeyForVoteReveal(msg.MarketId, member),
})
rId := nextQueryId()
rrsQueries = append(rrsQueries, rId)
rrsKeys = append(rrsKeys, &PluginKeyRead{
QueryId: rId,
Key:     KeyForResolverRecord(member),
})
}

// Layer 2: build queryId -> vote weight map from RRS scores
weightByQId := make(map[uint64]uint32)
if len(rrsKeys) > 0 {
rrsResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{Keys: rrsKeys})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if rrsResp.Error != nil {
return &PluginDeliverResponse{Error: rrsResp.Error}
}
for idx, r := range rrsResp.Results {
weight := VOTE_WEIGHT_BRONZE
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
rec := &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, rec); pe == nil {
if rec.RrsScore >= RRS_GOLD_THRESHOLD {
weight = VOTE_WEIGHT_GOLD
} else if rec.RrsScore >= RRS_SILVER_THRESHOLD {
weight = VOTE_WEIGHT_SILVER
}
}
}
// map the corresponding vote reveal queryId to this weight
if idx < len(queries) {
weightByQId[queries[idx]] = weight
}
}
}

var yesVotes, noVotes uint32
if len(readKeys) > 0 {
voteResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: readKeys,
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if voteResp.Error != nil {
return &PluginDeliverResponse{Error: voteResp.Error}
}

for _, r := range voteResp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
vr := &VoteReveal{}
if pe := Unmarshal(r.Entries[0].Value, vr); pe != nil {
continue
}
// Layer 2: apply RRS tier weight — default Bronze if missing
w := weightByQId[r.QueryId]
if w == 0 {
w = VOTE_WEIGHT_BRONZE
}
if vr.Vote {
yesVotes += w
} else {
noVotes += w
}
}
}

disputerWins := yesVotes > noVotes
dispute.VoteStatus = VOTE_TALLIED

if disputerWins {
market.Status = STATUS_PROPOSED
}

rawD, pe := SafeMarshal(dispute)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawM, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: KeyForDispute(msg.MarketId), Value: rawD},
{Key: KeyForMarket(msg.MarketId),  Value: rawM},
}

// PRIS v1.0-r3: if proposer wins dispute, RRS +20 and increment GlobalStats
if !disputerWins {
propQId   := nextQueryId()
recQId    := nextQueryId()
statsQId  := nextQueryId()
prisResp, prisErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: propQId,  Key: KeyForProposal(msg.MarketId)},
{QueryId: recQId,   Key: KeyForGlobalStats()},
{QueryId: statsQId, Key: KeyForGlobalStats()},
},
})
if prisErr == nil && prisResp.Error == nil {
proposal   := &ProposalRecord{}
globalStats := &GlobalStats{}
for _, r := range prisResp.Results {
if len(r.Entries) == 0 { continue }
switch r.QueryId {
case propQId:
_ = Unmarshal(r.Entries[0].Value, proposal)
case statsQId:
_ = Unmarshal(r.Entries[0].Value, globalStats)
}
}
if len(proposal.ResolverAddr) == 20 {
resolverRecQId := nextQueryId()
recResp, recErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: resolverRecQId, Key: KeyForResolverRecord(proposal.ResolverAddr)},
},
})
if recErr == nil && recResp.Error == nil {
resolverRec := &ResolverRecord{}
for _, r := range recResp.Results {
if r.QueryId == resolverRecQId && len(r.Entries) > 0 {
_ = Unmarshal(r.Entries[0].Value, resolverRec)
}
}
resolverRec.RrsScore += 20
resolverRec.SuccessfulResolutions++
weight := uint64(1)
if resolverRec.RrsScore >= RRS_GOLD_THRESHOLD {
weight = uint64(VOTE_WEIGHT_GOLD)
} else if resolverRec.RrsScore >= RRS_SILVER_THRESHOLD {
weight = uint64(VOTE_WEIGHT_SILVER)
}
globalStats.TotalWeightedResolutions += weight
rawRec, pe2 := SafeMarshal(resolverRec)
rawStats, pe3 := SafeMarshal(globalStats)
if pe2 == nil && pe3 == nil {
sets = append(sets, &PluginSetOp{Key: KeyForResolverRecord(proposal.ResolverAddr), Value: rawRec})
sets = append(sets, &PluginSetOp{Key: KeyForGlobalStats(), Value: rawStats})
}
}
}
}
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
