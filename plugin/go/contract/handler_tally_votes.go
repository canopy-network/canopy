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
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case disputeQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 || len(r.Entries[0].Value) == 0 {
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

for _, member := range dispute.PanelMembers {
qId := nextQueryId()
queries = append(queries, qId)
readKeys = append(readKeys, &PluginKeyRead{
QueryId: qId,
Key:     KeyForVoteReveal(msg.MarketId, member),
})
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
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
vr := &VoteReveal{}
if pe := Unmarshal(r.Entries[0].Value, vr); pe != nil {
continue
}
if vr.Vote {
yesVotes++
} else {
noVotes++
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

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForDispute(msg.MarketId), Value: rawD},
{Key: KeyForMarket(msg.MarketId),  Value: rawM},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
