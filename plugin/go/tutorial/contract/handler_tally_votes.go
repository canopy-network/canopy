package contract

// handler_tally_votes.go — MessageTallyVotes
// Spec: PORS v1.0-r2-CORRECTED (P5, NF-1)
//
// Permissionless — any address triggers tally after the reveal phase ends.
// Reads VoteReveal for each panel member (point lookups, not range scan).
// Majority YES → disputer wins → proposer slashed, market reopens for new proposal.
// Majority NO  → proposer wins → dispute rejected, market returns to STATUS_PROPOSED.
// Tie          → proposer wins (disputer bears burden of proof).
//
// Idempotency: DisputeRecord.VoteStatus == VOTE_TALLIED → return success immediately.
//
// CheckTx:  market_id 20 bytes, caller_addr 20 bytes. Zero StateRead (AUDIT-8, NF-1).
// DeliverTx:
//   Read market + dispute
//   Validate STATUS_DISPUTED and reveal phase ended
//   Point-lookup VoteReveal for each panel member
//   Count votes, determine winner
//   2-key atomic write: DisputeRecord (VOTE_TALLIED) + MarketState

func (c *Contract) CheckMessageTallyVotes(msg *MessageTallyVotes) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.CallerAddr) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
// Permissionless — no AuthorizedSigners restriction beyond caller signing.
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.CallerAddr},
}
}

func (c *Contract) DeliverMessageTallyVotes(msg *MessageTallyVotes, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read 1: market + dispute ────────────────────────────────────────
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

// Idempotency: already tallied.
if dispute.VoteStatus == VOTE_TALLIED {
return &PluginDeliverResponse{}
}

// Reveal phase must have ended.
revealEnd := dispute.DisputeBlock + COMMIT_PHASE_BLOCKS + REVEAL_PHASE_BLOCKS
if now <= revealEnd {
return &PluginDeliverResponse{Error: ErrTallyNotReady()}
}

// ── Batch read 2: VoteReveal for each panel member ────────────────────────
// Point lookups — bounded by panel size (max ELEVATED_RISK_PANEL_SIZE = 7).
type revealQuery struct {
queryId uint64
member  []byte
}
queries := make([]revealQuery, 0, len(dispute.PanelMembers))
readKeys := make([]*PluginKeyRead, 0, len(dispute.PanelMembers))

for _, member := range dispute.PanelMembers {
qId := nextQueryId()
queries = append(queries, revealQuery{queryId: qId, member: member})
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

// Build queryId → member map for matching.
qIdToMember := make(map[uint64][]byte, len(queries))
for _, q := range queries {
qIdToMember[q.queryId] = q.member
}

for _, r := range voteResp.Results {
if len(r.Entries) == 0 {
// Panel member did not reveal — abstention, counts as no vote.
continue
}
vr := &VoteReveal{}
if pe := Unmarshal(r.Entries[0].Value, vr); pe != nil {
continue // skip malformed — treat as abstention
}
if vr.Vote {
yesVotes++
} else {
noVotes++
}
}
}

// Majority YES → disputer wins (proposed outcome was wrong).
// Majority NO or tie → proposer wins (dispute rejected).
disputerWins := yesVotes > noVotes

// Update dispute record.
dispute.VoteStatus = VOTE_TALLIED

// Update market status based on tally result.
// Disputer wins: proposer's outcome was wrong → market returns to STATUS_PROPOSED
// for a new proposal (the old proposal is invalidated). Slashing handled in
// finalize_market after the panel result is used.
// Proposer wins: dispute rejected → market stays on path to STATUS_FINALIZED.
if disputerWins {
// Disputer wins — proposed outcome overturned.
// Market returns to STATUS_PROPOSED so a new propose_outcome can be filed.
// The existing ProposalRecord remains as evidence for slashing in finalize_market.
market.Status = STATUS_PROPOSED
}
// If proposer wins, market.Status stays STATUS_DISPUTED — finalize_market
// will transition it to STATUS_FINALIZED after confirming the tally.

// ── Marshal ───────────────────────────────────────────────────────────────
rawD, pe := SafeMarshal(dispute)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawM, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// ── 2-key atomic write ────────────────────────────────────────────────────
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
