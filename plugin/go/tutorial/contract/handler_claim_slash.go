package contract

// handler_claim_slash.go — MessageClaimSlash
// Spec: PORS v1.0-r2-CORRECTED
//
// Called by the PROPOSER (winner) after finalize_market to claim the slashed
// disputer bond. finalize_market writes SlashRecord keyed by dispute.DisputerAddress
// (the loser). claim_slash reads that key via the dispute record.
//
// Flow:
//   1. finalize_market (Path A, proposer wins) writes:
//      SlashRecord at KeyForSlashRecord(dispute.DisputerAddress)
//   2. Proposer calls claim_slash — we read dispute to find DisputerAddress,
//      then read SlashRecord at that key, pay slash_amount to proposer from treasury.
//
// Idempotency: slash.SlashAmount == 0 after first claim.
//
// CheckTx:  market_id 20 bytes, claimant_address 20 bytes. Zero StateRead (AUDIT-8).
// DeliverTx:
//   Batch read 1: market + proposal + dispute
//   Validate market STATUS_FINALIZED, claimant == proposal.ResolverAddr
//   Batch read 2: SlashRecord (keyed by disputer) + claimant account + treasury
//   Pay slash_amount from treasury to claimant
//   3-key atomic write: SlashRecord (zeroed) + claimant account + treasury

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

// ── Batch read 1: market + proposal + dispute ─────────────────────────────
marketQId   := nextQueryId()
proposalQId := nextQueryId()
disputeQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: proposalQId, Key: KeyForProposal(msg.MarketId)},
{QueryId: disputeQId,  Key: KeyForDispute(msg.MarketId)},
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
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrInternal()}
}
proposal = &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case disputeQId:
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrInternal()}
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
// Market must be finalized for slash claims.
if market.Status != STATUS_FINALIZED {
return &PluginDeliverResponse{Error: ErrNotFinalized()}
}
if proposal == nil || dispute == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}
// Claimant must be the proposer (the winner whose opponent was slashed).
if !bytesEqual(msg.ClaimantAddress, proposal.ResolverAddr) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}
// Dispute must have been tallied (proposer won).
if dispute.VoteStatus != VOTE_TALLIED {
return &PluginDeliverResponse{Error: ErrTallyNotReady()}
}

// ── Batch read 2: SlashRecord (keyed by DISPUTER) + claimant account + treasury ──
// finalize_market wrote SlashRecord at KeyForSlashRecord(dispute.DisputerAddress).
slashQId    := nextQueryId()
claimAccQId := nextQueryId()
treasQId    := nextQueryId()

resp2, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: slashQId,    Key: KeyForSlashRecord(dispute.DisputerAddress)},
{QueryId: claimAccQId, Key: KeyForAccount(msg.ClaimantAddress)},
{QueryId: treasQId,    Key: KeyForTreasuryReserve(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp2.Error != nil {
return &PluginDeliverResponse{Error: resp2.Error}
}

var slash *SlashRecord
claimAcc := &Account{}
treasury := &TreasuryReserve{}

for _, r := range resp2.Results {
switch r.QueryId {
case slashQId:
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrNoSlashToClaim()}
}
slash = &SlashRecord{}
if pe := Unmarshal(r.Entries[0].Value, slash); pe != nil {
return &PluginDeliverResponse{Error: pe}
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

if slash == nil || slash.SlashAmount == 0 {
return &PluginDeliverResponse{Error: ErrNoSlashToClaim()}
}

// Pay slash amount from treasury to claimant.
slashAmount := slash.SlashAmount
if treasury.LockedReserve < slashAmount {
slashAmount = treasury.LockedReserve
}
treasury.LockedReserve -= slashAmount
claimAcc.Amount        += slashAmount

// Zero out slash amount — idempotency sentinel.
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

// 3-key atomic write: SlashRecord (zeroed) + claimant account + treasury.
// SlashRecord key uses disputer address (where finalize_market wrote it).
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForSlashRecord(dispute.DisputerAddress), Value: rawSlash},
{Key: KeyForAccount(msg.ClaimantAddress),         Value: rawAcc},
{Key: KeyForTreasuryReserve(msg.MarketId),        Value: rawT},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
