package contract

// handler_file_dispute.go — MessageFileDispute
// Spec: PORS v1.0-r2-CORRECTED
//
// Filed by a disputer after propose_outcome commits a STATUS_PROPOSED market.
// Disputer puts up a bond. Market transitions to STATUS_DISPUTED.
// Panel size determined by elevated_risk flag (P7).
//
// CheckTx:  market_id 20 bytes, disputer_address 20 bytes, dispute_bond non-zero.
//           Zero StateRead (AUDIT-8).
// DeliverTx:
//   Batch read: market + proposal + disputer account
//   Validate STATUS_PROPOSED and dispute window open
//   Deduct bond from disputer
//   3-key atomic write: MarketState + DisputeRecord + disputer Account

func (c *Contract) CheckMessageFileDispute(msg *MessageFileDispute) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.DisputerAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if msg.DisputeBond == 0 {
return ErrCheckResp(ErrInvalidAmount())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.DisputerAddress},
}
}

func (c *Contract) DeliverMessageFileDispute(msg *MessageFileDispute, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId   := nextQueryId()
proposalQId := nextQueryId()
dispAccQId  := nextQueryId()
feeQId      := nextQueryId()

marketKey   := KeyForMarket(msg.MarketId)
proposalKey := KeyForProposal(msg.MarketId)
dispAccKey  := KeyForAccount(msg.DisputerAddress)
feePoolKey  := KeyForFeePool(c.Config.ChainId)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: marketKey},
{QueryId: proposalQId, Key: proposalKey},
{QueryId: dispAccQId,  Key: dispAccKey},
{QueryId: feeQId,      Key: feePoolKey},
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
disputer    := &Account{}
feePool     := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case marketQId:
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case proposalQId:
proposal = &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case dispAccQId:
if pe := Unmarshal(r.Entries[0].Value, disputer); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feeQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if market.Status != STATUS_PROPOSED {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}
if proposal == nil {
return &PluginDeliverResponse{Error: ErrInternal()}
}

// Dispute window: proposal_block + ComputeDisputeBlocks(open_time, expiry_time)
disputeWindow := ComputeDisputeBlocks(market.OpenTime, market.ExpiryTime)
disputeDeadline := proposal.ProposalBlock + disputeWindow
if now > disputeDeadline {
return &PluginDeliverResponse{Error: ErrDisputeWindowClosed()}
}

// Total cost: dispute_bond + fee.
if fee > 0 && msg.DisputeBond > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.DisputeBond + fee
if disputer.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

// Panel size determined by elevated_risk flag (P7).
var panelSize uint32
if market.ElevatedRisk {
panelSize = ELEVATED_RISK_PANEL_SIZE
} else {
panelSize = MIN_PANEL_SIZE
}

// Mutate in memory.
market.Status     = STATUS_DISPUTED
disputer.Amount  -= totalCost
feePool.Amount   += fee

dispute := &DisputeRecord{
DisputerAddress: msg.DisputerAddress,
DisputeBond:     msg.DisputeBond,
DisputeBlock:    now,
VoteStatus:      VOTE_PENDING,
PanelSize:       panelSize,
}

rawM, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawD, pe := SafeMarshal(dispute)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawFee, pe := SafeMarshal(feePool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

sets := []*PluginSetOp{
{Key: marketKey,                    Value: rawM},
{Key: KeyForDispute(msg.MarketId),  Value: rawD},
{Key: feePoolKey,                   Value: rawFee},
}
var deletes []*PluginDeleteOp
if disputer.Amount == 0 {
deletes = []*PluginDeleteOp{{Key: dispAccKey}}
} else {
rawDisp, pe := SafeMarshal(disputer)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
sets = append(sets, &PluginSetOp{Key: dispAccKey, Value: rawDisp})
}

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets:    sets,
Deletes: deletes,
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
