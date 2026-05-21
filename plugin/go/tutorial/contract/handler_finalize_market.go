package contract

// handler_finalize_market.go — MessageFinalizeMarket
// Spec: PORS v1.0-r2-CORRECTED (P6)
//
// Permissionless — any caller finalizes an eligible market and receives
// FINALIZATION_BOUNTY (50 PRX) from the TreasuryReserve.
//
// Two eligible paths:
//   Path A: STATUS_DISPUTED + dispute.VoteStatus == VOTE_TALLIED (proposer won).
//           Disputer's bond slashed to treasury. Proposer's bond returned.
//           SlashRecord written for disputer's bond amount.
//           Market → STATUS_FINALIZED.
//   Path B: STATUS_PROPOSED + dispute window has passed with no dispute filed.
//           Proposer's bond returned from treasury. No slash.
//           Market → STATUS_FINALIZED.
//
// Idempotency: market.Status == STATUS_FINALIZED → return success immediately.
//
// CheckTx:  market_id 20 bytes, caller_addr 20 bytes. Zero StateRead (AUDIT-8).
// DeliverTx:
//   Read market + dispute + proposal + treasury + caller account + proposer account
//   Determine path, validate eligibility
//   Atomic write: market + treasury + caller account + proposal record +
//                 proposer account + slash record (Path A only) + disputer account (Path A only)

func (c *Contract) CheckMessageFinalizeMarket(msg *MessageFinalizeMarket) *PluginCheckResponse {
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

func (c *Contract) DeliverMessageFinalizeMarket(msg *MessageFinalizeMarket, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read 1: market + dispute + proposal + treasury ─────────────────
marketQId   := nextQueryId()
disputeQId  := nextQueryId()
proposalQId := nextQueryId()
treasQId    := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: disputeQId,  Key: KeyForDispute(msg.MarketId)},
{QueryId: proposalQId, Key: KeyForProposal(msg.MarketId)},
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
var dispute  *DisputeRecord
var proposal *ProposalRecord
treasury    := &TreasuryReserve{}

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
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
dispute = &DisputeRecord{}
if pe := Unmarshal(r.Entries[0].Value, dispute); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case proposalQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
proposal = &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case treasQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, treasury); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}

// Idempotency.
if market.Status == STATUS_FINALIZED {
return &PluginDeliverResponse{}
}

// Determine which path applies.
pathA := market.Status == STATUS_DISPUTED &&
dispute != nil && dispute.VoteStatus == VOTE_TALLIED
pathB := market.Status == STATUS_PROPOSED && dispute == nil
if !pathA && !pathB {
// Path B variant: dispute window passed even if dispute record exists
// but tally has not run — not eligible yet.
return &PluginDeliverResponse{Error: ErrNotFinalized()}
}

// Path B eligibility: dispute window must have passed.
if pathB {
if proposal == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}
disputeWindow   := ComputeDisputeBlocks(market.OpenTime, market.ExpiryTime)
disputeDeadline := proposal.ProposalBlock + disputeWindow
if now <= disputeDeadline {
return &PluginDeliverResponse{Error: ErrDisputeWindowClosed()}
}
}

// ── Batch read 2: caller account + proposer account (+ disputer if Path A) ─
callerQId   := nextQueryId()
proposerQId := nextQueryId()

readKeys2 := []*PluginKeyRead{
{QueryId: callerQId,   Key: KeyForAccount(msg.CallerAddr)},
}

var proposerKey []byte
var disputerKey []byte
var disputerQId uint64

if proposal != nil {
proposerKey = KeyForAccount(proposal.ResolverAddr)
readKeys2 = append(readKeys2, &PluginKeyRead{
QueryId: proposerQId,
Key:     proposerKey,
})
}

if pathA && dispute != nil {
disputerQId = nextQueryId()
disputerKey = KeyForAccount(dispute.DisputerAddress)
readKeys2 = append(readKeys2, &PluginKeyRead{
QueryId: disputerQId,
Key:     disputerKey,
})
}

resp2, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: readKeys2,
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp2.Error != nil {
return &PluginDeliverResponse{Error: resp2.Error}
}

callerAcc   := &Account{}
proposerAcc := &Account{}
disputerAcc := &Account{}

for _, r := range resp2.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
switch r.QueryId {
case callerQId:
if pe := Unmarshal(r.Entries[0].Value, callerAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case proposerQId:
if pe := Unmarshal(r.Entries[0].Value, proposerAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case disputerQId:
if pe := Unmarshal(r.Entries[0].Value, disputerAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

// ── Apply logic ───────────────────────────────────────────────────────────
// Bounty to caller from treasury.
bounty := FINALIZATION_BOUNTY
if treasury.LockedReserve < bounty {
bounty = treasury.LockedReserve // pay whatever is available
}
treasury.LockedReserve -= bounty
callerAcc.Amount       += bounty

// Return proposer's bond from treasury.
var bondReturn uint64
if proposal != nil {
bondReturn = proposal.ProposalBond
if treasury.LockedReserve < bondReturn {
bondReturn = treasury.LockedReserve
}
treasury.LockedReserve -= bondReturn
proposerAcc.Amount     += bondReturn
}

// Transition market to finalized.
market.Status = STATUS_FINALIZED

// ── Build write set ───────────────────────────────────────────────────────
rawM, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawT, pe := SafeMarshal(treasury)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawCaller, pe := SafeMarshal(callerAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

sets := []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),           Value: rawM},
{Key: KeyForTreasuryReserve(msg.MarketId),  Value: rawT},
{Key: KeyForAccount(msg.CallerAddr),         Value: rawCaller},
}

if proposal != nil && proposerKey != nil {
rawProposer, pe := SafeMarshal(proposerAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
sets = append(sets, &PluginSetOp{Key: proposerKey, Value: rawProposer})
}

// Path A: slash disputer's bond, write SlashRecord.
if pathA && dispute != nil {
slashAmount := dispute.DisputeBond
slash := &SlashRecord{
SlashedAddress: dispute.DisputerAddress,
SlashAmount:    slashAmount,
SlashedAt:      now,
}
rawSlash, pe := SafeMarshal(slash)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
// Disputer's bond is already held by the protocol (deducted at file_dispute).
// SlashRecord records it for claim_slash — disputer cannot reclaim it.
sets = append(sets, &PluginSetOp{
Key:   KeyForSlashRecord(dispute.DisputerAddress),
Value: rawSlash,
})
// Disputer account unchanged — bond was already deducted at filing.
_ = disputerAcc
_ = disputerKey
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: sets,
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
