package contract

func (c *Contract) CheckMessageRevealVote(msg *MessageRevealVote) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.VoterAddr) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if len(msg.Nonce) == 0 {
return ErrCheckResp(ErrInvalidParam())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.VoterAddr},
}
}

func (c *Contract) DeliverMessageRevealVote(msg *MessageRevealVote, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId  := nextQueryId()
disputeQId := nextQueryId()
commitQId  := nextQueryId()
revealQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,  Key: KeyForMarket(msg.MarketId)},
{QueryId: disputeQId, Key: KeyForDispute(msg.MarketId)},
{QueryId: commitQId,  Key: KeyForVoteCommit(msg.MarketId, msg.VoterAddr)},
{QueryId: revealQId,  Key: KeyForVoteReveal(msg.MarketId, msg.VoterAddr)},
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
var vc      *VoteCommit

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
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}
vc = &VoteCommit{}
if pe := Unmarshal(r.Entries[0].Value, vc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case revealQId:
if len(r.Entries) > 0 {
return &PluginDeliverResponse{Error: ErrAlreadyRevealed()}
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
if vc == nil {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}

revealStart := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS
revealEnd   := revealStart + REVEAL_PHASE_BLOCKS
if now <= revealStart {
return &PluginDeliverResponse{Error: ErrRevealPhaseNotOpen()}
}
if now > revealEnd {
return &PluginDeliverResponse{Error: ErrRevealPhaseOver()}
}

if !isPanelMember(msg.VoterAddr, dispute.PanelMembers) {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}

expectedHash := ComputeCommitHash(msg.Vote, msg.Nonce, msg.VoterAddr)
if !bytesEqual(expectedHash, vc.CommitHash) {
return &PluginDeliverResponse{Error: ErrCommitHashMismatch()}
}

vr := &VoteReveal{
VoterAddr:  msg.VoterAddr,
Vote:       msg.Vote,
RevealedAt: now,
}
rawVR, pe := SafeMarshal(vr)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForVoteReveal(msg.MarketId, msg.VoterAddr), Value: rawVR},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
