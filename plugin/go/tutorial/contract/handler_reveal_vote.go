package contract

// handler_reveal_vote.go — MessageRevealVote
// Spec: PORS v1.0-r2-CORRECTED (P5)
//
// Panel members reveal their vote during the reveal phase.
// The revealed vote must match the previously committed hash:
//   commit_hash == SHA256(vote_byte || nonce || voter_addr)
//
// CheckTx:  market_id 20 bytes, voter_addr 20 bytes, nonce non-empty.
//           Zero StateRead (AUDIT-8).
// DeliverTx:
//   Read market + dispute + VoteCommit + VoteReveal
//   Validate STATUS_DISPUTED and within reveal phase window
//   Validate voter is a panel member
//   Validate VoteCommit exists and VoteReveal does not
//   Verify commit hash matches SHA256(vote || nonce || voter_addr)
//   1-key atomic write: VoteReveal

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
// Cannot reveal without first committing.
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}
vc = &VoteCommit{}
if pe := Unmarshal(r.Entries[0].Value, vc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case revealQId:
// Already revealed.
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

// Reveal phase window: after commit phase ends, before reveal phase ends.
revealStart := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS
revealEnd   := revealStart + REVEAL_PHASE_BLOCKS
if now <= revealStart {
return &PluginDeliverResponse{Error: ErrRevealPhaseNotOpen()}
}
if now > revealEnd {
return &PluginDeliverResponse{Error: ErrRevealPhaseOver()}
}

// Verify voter is a panel member.
if !isPanelMember(msg.VoterAddr, dispute.PanelMembers) {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}

// Verify commit hash: SHA256(vote_byte || nonce || voter_addr).
expectedHash := ComputeCommitHash(msg.Vote, msg.Nonce, msg.VoterAddr)
if !bytesEqual(expectedHash, vc.CommitHash) {
return &PluginDeliverResponse{Error: ErrCommitHashMismatch()}
}

// Write VoteReveal — single atomic key.
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
