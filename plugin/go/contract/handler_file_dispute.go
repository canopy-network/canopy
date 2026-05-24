package contract

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
entropyQId  := nextQueryId()
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
{QueryId: entropyQId,  Key: PANEL_ENTROPY_KEY},
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
var entropyVal uint64

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
case proposalQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrInternal()}
}
proposal = &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case dispAccQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, disputer); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case entropyQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
acc := &PanelEntropyAccum{}
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
entropyVal = acc.Accumulator
}
case feeQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
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

disputeWindow   := ComputeDisputeBlocks(market.OpenTime, market.ExpiryTime)
disputeDeadline := proposal.ProposalBlock + disputeWindow
if now > disputeDeadline {
return &PluginDeliverResponse{Error: ErrDisputeWindowClosed()}
}

if market.Status == STATUS_DISPUTED {
return &PluginDeliverResponse{Error: ErrAlreadyDisputed()}
}

if fee > 0 && msg.DisputeBond > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.DisputeBond + fee
if disputer.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

resolverRangeQId := nextQueryId()
rangeResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Ranges: []*PluginRangeRead{
{
QueryId: resolverRangeQId,
Prefix:  resolverRecordPrefix,
Limit:   0,
Reverse: false,
},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if rangeResp.Error != nil {
return &PluginDeliverResponse{Error: rangeResp.Error}
}

var candidates [][]byte
for _, r := range rangeResp.Results {
if r.QueryId != resolverRangeQId {
continue
}
for _, entry := range r.Entries {
rec := &ResolverRecord{}
if pe := Unmarshal(entry.Value, rec); pe != nil {
continue
}
if bytesEqual(rec.ResolverAddress, proposal.ResolverAddr) {
continue
}
if bytesEqual(rec.ResolverAddress, msg.DisputerAddress) {
continue
}
if rec.RrsScore < MIN_RRS_TO_PROPOSE {
continue
}
candidates = append(candidates, rec.ResolverAddress)
}
}

// Layer 1: position exclusion
// Any resolver with an open unclaimed position in this market is ineligible.
posQueries := make([]uint64, len(candidates))
posKeys := make([]*PluginKeyRead, len(candidates))
for i, addr := range candidates {
qId := nextQueryId()
posQueries[i] = qId
posKeys[i] = &PluginKeyRead{QueryId: qId, Key: KeyForPosition(msg.MarketId, addr)}
}
if len(posKeys) > 0 {
posResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{Keys: posKeys})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if posResp.Error != nil {
return &PluginDeliverResponse{Error: posResp.Error}
}
disqualified := make(map[string]bool)
for _, r := range posResp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
pos := &PositionState{}
if pe := Unmarshal(r.Entries[0].Value, pos); pe != nil {
continue
}
if !pos.Claimed {
for i, qId := range posQueries {
if r.QueryId == qId {
disqualified[string(candidates[i])] = true
}
}
}
}
filtered := candidates[:0]
for _, addr := range candidates {
if !disqualified[string(addr)] {
filtered = append(filtered, addr)
}
}
candidates = filtered
}

var panelSize uint32
if market.ElevatedRisk {
panelSize = ELEVATED_RISK_PANEL_SIZE
} else {
panelSize = MIN_PANEL_SIZE
}

seed := entropyVal ^ (now * FIBONACCI_HASH_CONSTANT)
panel := derivePanel(candidates, int(panelSize), seed)
if len(panel) == 0 {
return &PluginDeliverResponse{Error: ErrInsufficientPanelCandidates()}
}

market.Status    = STATUS_DISPUTED
disputer.Amount -= totalCost
feePool.Amount  += fee

dispute := &DisputeRecord{
DisputerAddress: msg.DisputerAddress,
DisputeBond:     msg.DisputeBond,
DisputeBlock:    now,
VoteStatus:      VOTE_PENDING,
PanelSize:       uint32(len(panel)),
PanelMembers:    panel,
}

rawM, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawD, pe := SafeMarshal(dispute)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawFee, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

sets := []*PluginSetOp{
{Key: marketKey,                   Value: rawM},
{Key: KeyForDispute(msg.MarketId), Value: rawD},
{Key: feePoolKey,                  Value: rawFee},
}
var deletes []*PluginDeleteOp
if disputer.Amount == 0 {
deletes = []*PluginDeleteOp{{Key: dispAccKey}}
} else {
rawDisp, pe := SafeMarshal(disputer)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
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

func derivePanel(candidates [][]byte, n int, seed uint64) [][]byte {
if len(candidates) == 0 || n == 0 {
return nil
}
pool := make([][]byte, len(candidates))
copy(pool, candidates)

s := seed
lcgNext := func() uint64 {
s = s*6364136223846793005 + 1442695040888963407
return s
}

limit := len(pool)
if n < limit {
limit = n
}
for i := 0; i < limit; i++ {
j := int(lcgNext()%uint64(len(pool)-i)) + i
pool[i], pool[j] = pool[j], pool[i]
}
return pool[:limit]
}
