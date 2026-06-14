package contract

// handler_commit_vote.go — MessageCommitVote
// Spec: PORS v1.0-r2-CORRECTED (P5)
//
// Panel members submit a blinded vote during the commit phase.
// commit_hash = SHA256(vote_byte || nonce || voter_addr)
// The hash is stored but not revealed until the reveal phase.
//
// CheckTx:  market_id 20 bytes, voter_addr 20 bytes, commit_hash 32 bytes.
//           Zero StateRead (AUDIT-8).
// DeliverTx:
//   Read market + dispute + existing VoteCommit
//   Validate STATUS_DISPUTED and within commit phase window
//   Validate voter is a panel member
//   Reject if already committed
//   1-key atomic write: VoteCommit

func (c *Contract) CheckMessageCommitVote(msg *MessageCommitVote) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.VoterAddr) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
// commit_hash must be exactly 32 bytes (SHA256 output).
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
// Already committed if this key exists.
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

// Commit phase window: dispute_block to dispute_block + COMMIT_PHASE_BLOCKS.
commitDeadline := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS
if now > commitDeadline {
return &PluginDeliverResponse{Error: ErrCommitPhaseOver()}
}

// Verify voter is a panel member.
if !isPanelMember(msg.VoterAddr, dispute.PanelMembers) {
return &PluginDeliverResponse{Error: ErrNotAPanelMember()}
}

// Write VoteCommit — single atomic key.
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

// isPanelMember returns true if addr is in the panelMembers list.
func isPanelMember(addr []byte, panelMembers [][]byte) bool {
for _, m := range panelMembers {
if bytesEqual(addr, m) {
return true
}
}
return false
}
