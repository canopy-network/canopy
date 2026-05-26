package contract

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

if market.Status == STATUS_FINALIZED {
return &PluginDeliverResponse{}
}

pathA := market.Status == STATUS_DISPUTED && dispute != nil && dispute.VoteStatus == VOTE_TALLIED
pathB := market.Status == STATUS_PROPOSED && dispute == nil
if !pathA && !pathB {
return &PluginDeliverResponse{Error: ErrNotFinalized()}
}

if pathB {
if proposal == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}
disputeWindow   := ComputeDisputeBlocks(market.OpenTime, market.ExpiryTime)
disputeDeadline := proposal.ProposalBlock + disputeWindow
// Reject if dispute window is still open — too early to finalize.
// In TEST_MODE, skip the window check so tests don't wait 34,560 blocks.
disputeWindowOpen := now <= disputeDeadline
if !TEST_MODE && disputeWindowOpen {
return &PluginDeliverResponse{Error: ErrDisputeWindowOpen()}
}
}

callerQId   := nextQueryId()
proposerQId := nextQueryId()

readKeys2 := []*PluginKeyRead{
{QueryId: callerQId,   Key: KeyForAccount(msg.CallerAddr)},
}

var proposerKey []byte
var disputerKey []byte
var disputerQId uint64

if proposal == nil { return &PluginDeliverResponse{Error: ErrInternal()} }
proposerKey = KeyForAccount(proposal.ResolverAddr)
readKeys2 = append(readKeys2, &PluginKeyRead{QueryId: proposerQId, Key: proposerKey})

if pathA && dispute != nil {
disputerQId = nextQueryId()
disputerKey = KeyForAccount(dispute.DisputerAddress)
readKeys2 = append(readKeys2, &PluginKeyRead{QueryId: disputerQId, Key: disputerKey})
}

resp2, err := c.plugin.StateRead(c, &PluginStateReadRequest{Keys: readKeys2})
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

bounty := FINALIZATION_BOUNTY
if treasury.LockedReserve < bounty {
bounty = treasury.LockedReserve
}
treasury.LockedReserve -= bounty
callerAcc.Amount       += bounty

var bondReturn uint64
if proposal != nil {
bondReturn = proposal.ProposalBond
if treasury.LockedReserve < bondReturn {
bondReturn = treasury.LockedReserve
}
treasury.LockedReserve -= bondReturn
proposerAcc.Amount     += bondReturn
}

market.Status = STATUS_FINALIZED

rawM, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawT, pe := SafeMarshal(treasury)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawCaller, pe := SafeMarshal(callerAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),          Value: rawM},
{Key: KeyForTreasuryReserve(msg.MarketId), Value: rawT},
{Key: KeyForAccount(msg.CallerAddr),        Value: rawCaller},
}

if proposal != nil && proposerKey != nil {
rawProposer, pe := SafeMarshal(proposerAcc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: proposerKey, Value: rawProposer})
}

// Write OutcomeState so claim_winnings can find the winning outcome.
if proposal != nil {
outcome := &OutcomeState{WinningOutcome: proposal.ProposedOutcome, ResolvedAt: now}
rawO, pe := SafeMarshal(outcome)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: KeyForOutcome(msg.MarketId), Value: rawO})
}

if pathA && dispute != nil {
slashAmount := dispute.DisputeBond
slash := &SlashRecord{
SlashedAddress: dispute.DisputerAddress,
SlashAmount:    slashAmount,
SlashedAt:      now,
}
rawSlash, pe := SafeMarshal(slash)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
sets = append(sets, &PluginSetOp{Key: KeyForSlashRecord(dispute.DisputerAddress), Value: rawSlash})
_ = disputerAcc
_ = disputerKey
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
