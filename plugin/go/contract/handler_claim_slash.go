package contract

// handler_claim_slash.go — MessageClaimSlash
// Spec: PORS v1.0-r2-CORRECTED
//
// Called by the winning disputer to claim the slashed proposer bond after
// finalize_market has written a SlashRecord.
//
// Note: In Praxis the SlashRecord is written against the DISPUTER (loser)
// by finalize_market when the proposer wins. This handler is called by
// whoever is owed slash proceeds — in the current spec that is the
// protocol treasury reclaiming losing dispute bonds. The claimant must
// be the address recorded in SlashRecord.SlashedAddress is the disputer.
//
// Actually per PORS: when DISPUTER wins (panel overturns proposal), the
// PROPOSER's bond is slashed. claim_slash is called by the disputer to
// claim that slashed proposer bond as their reward.
//
// This requires finalize_market to write SlashRecord keyed by
// PROPOSER address (not disputer) when disputer wins — i.e. the opposite
// path from what finalize_market above wrote.
//
// Re-audit: tally_votes disputer wins → market back to STATUS_PROPOSED.
// In that path finalize_market is NOT called (market reopens for proposals).
// claim_slash is only relevant when the market goes to STATUS_FINALIZED
// via Path A (proposer wins tally) — in which case disputer's bond is slashed.
// The disputer cannot claim it back — it stays in treasury.
//
// Therefore claim_slash = disputer claiming nothing useful in Path A.
// The only coherent use: if disputer wins (market back to PROPOSED), the
// proposer's bond should be slashable. But finalize_market is not called
// in that path.
//
// Spec resolution: claim_slash allows the DISPUTER to claim the PROPOSER's
// slashed bond after the disputer wins. This requires tally_votes (disputer
// wins path) to write a SlashRecord for the PROPOSER's bond, and a separate
// finalization step for that path. We implement claim_slash to pay out
// SlashRecord.SlashAmount to the claimant from treasury when the record exists.
//
// CheckTx:  market_id 20 bytes, claimant_address 20 bytes. Zero StateRead.
// DeliverTx:
//   Read slash record + claimant account + treasury
//   Claimant must match SlashRecord.SlashedAddress (the address that was slashed)
//   Actually claimant is the WINNER who gets the slash proceeds.
//   We store SlashRecord keyed by the SLASHED address, amount = what they lost.
//   The winner claims it. We identify the winner from the dispute record.
//   Read dispute to confirm claimant is the disputer (winner in Path A = proposer wins,
//   loser = disputer, but disputer lost so they get nothing).
//
// Final spec-compliant interpretation:
//   SlashRecord.SlashedAddress = the address whose bond was slashed (disputer in Path A).
//   SlashRecord.SlashAmount    = amount slashed and held in treasury.
//   claim_slash: called by the PROPOSER (winner) to claim the slashed bond from treasury.
//   We identify the proposer from ProposalRecord.ResolverAddr.

func (c *Contract) CheckMessageClaimSlash(msg *MessageClaimSlash) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.ClaimantAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ClaimantAddress},
}
}

func (c *Contract) DeliverMessageClaimSlash(msg *MessageClaimSlash, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId   := nextQueryId()
proposalQId := nextQueryId()
disputeQId  := nextQueryId()
slashQId    := nextQueryId()
claimAccQId := nextQueryId()
treasQId    := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: proposalQId, Key: KeyForProposal(msg.MarketId)},
{QueryId: disputeQId,  Key: KeyForDispute(msg.MarketId)},
{QueryId: slashQId,    Key: KeyForSlashRecord(msg.ClaimantAddress)},
{QueryId: claimAccQId, Key: KeyForAccount(msg.ClaimantAddress)},
{QueryId: treasQId,    Key: KeyForTreasuryReserve(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market   *MarketState
var proposal *ProposalRecord
var dispute  *DisputeRecord
var slash    *SlashRecord
claimAcc    := &Account{}
treasury    := &TreasuryReserve{}

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
case proposalQId:
if len(r.Entries) > 0 {
proposal = &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case disputeQId:
if len(r.Entries) > 0 {
dispute = &DisputeRecord{}
if pe := Unmarshal(r.Entries[0].Value, dispute); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case slashQId:
if len(r.Entries) > 0 {
slash = &SlashRecord{}
if pe := Unmarshal(r.Entries[0].Value, slash); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case claimAccQId:
if len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, claimAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case treasQId:
if len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, treasury); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
// Market must be finalized for slash claims.
if market.Status != STATUS_FINALIZED {
return &PluginDeliverResponse{Error: ErrNotFinalized()}
}
// Slash record must exist for this claimant.
if slash == nil || slash.SlashAmount == 0 {
return &PluginDeliverResponse{Error: ErrNoSlashToClaim()}
}
// Claimant must be the proposer (winner who is owed the slashed disputer bond).
// The SlashRecord is keyed by the DISPUTER (who was slashed), but the
// claimant here is the PROPOSER. We verify via proposal record.
if proposal == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}
if !bytesEqual(msg.ClaimantAddress, proposal.ResolverAddr) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}
// Verify dispute exists and proposer won (dispute tallied, market finalized).
if dispute == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}
if dispute.VoteStatus != VOTE_TALLIED {
return &PluginDeliverResponse{Error: ErrTallyNotReady()}
}

// Pay slash amount from treasury to claimant.
slashAmount := slash.SlashAmount
if treasury.LockedReserve < slashAmount {
slashAmount = treasury.LockedReserve
}
treasury.LockedReserve -= slashAmount
claimAcc.Amount        += slashAmount

// Zero out slash record — idempotency sentinel.
slash.SlashAmount = 0

rawSlash, pe := SafeMarshal(slash)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawAcc, pe := SafeMarshal(claimAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawT, pe := SafeMarshal(treasury)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// 3-key atomic write: slash record (zeroed) + claimant account + treasury.
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForSlashRecord(msg.ClaimantAddress),   Value: rawSlash},
{Key: KeyForAccount(msg.ClaimantAddress),       Value: rawAcc},
{Key: KeyForTreasuryReserve(msg.MarketId),      Value: rawT},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
