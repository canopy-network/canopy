package contract

func (c *Contract) CheckMessageCommitVote(msg *MessageCommitVote) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.VoterAddr) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if len(msg.CommitHash) != 32 {
return ErrCheckResp(ErrInvalidCommitHash())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.VoterAddr},
}
}

func (c *Contract) DeliverMessageCommitVote(msg *MessageCommitVote, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId  := nextQueryId()
disputeQId := nextQueryId()
commitQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,  Key: KeyForMarket(msg.MarketId)},
{QueryId: disputeQId, Key: KeyForDispute(msg.MarketId)},
{QueryId: commitQId,  Key: KeyForVoteCommit(msg.MarketId, msg.VoterAddr)},
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
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case disputeQId:
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrNotDisputed()}
}
dispute = &DisputeRecord{}
if pe := Unmarshal(r.Entries[0].Value, dispute); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case commitQId:
if len(r.Entries) > 0 {
return &PluginDeliverResponse{Error: ErrAlreadyCommitted()}
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

commitDeadline := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS
if now > commitDeadline {
return &PluginDeliverResponse{Error: ErrCommitPhaseOver()}
}

if !isPanelMember(msg.VoterAddr, dispute.PanelMembers) {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}

vc := &VoteCommit{
VoterAddr:   msg.VoterAddr,
CommitHash:  msg.CommitHash,
CommittedAt: now,
}
rawVC, pe := SafeMarshal(vc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForVoteCommit(msg.MarketId, msg.VoterAddr), Value: rawVC},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}

func isPanelMember(addr []byte, panelMembers [][]byte) bool {
for _, m := range panelMembers {
if bytesEqual(addr, m) {
return true
}
}
return false
}
