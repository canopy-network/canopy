package contract

func (c *Contract) CheckMessageProposeOutcome(msg *MessageProposeOutcome) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if msg.ProposalBond == 0 {
return ErrCheckResp(ErrInvalidAmount())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageProposeOutcome(msg *MessageProposeOutcome, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

resolRecQId := nextQueryId()
marketQId   := nextQueryId()
proposalQId := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: resolRecQId, Key: KeyForResolverRecord(msg.ResolverAddress)},
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: proposalQId, Key: KeyForProposal(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var resolverRec *ResolverRecord
var market     *MarketState
var proposalRaw []byte

for _, r := range resp.Results {
switch r.QueryId {
case resolRecQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrResolverNotRegistered()}
}
resolverRec = &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, resolverRec); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case marketQId:
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
market = &MarketState{}
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case proposalQId:
if len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
proposalRaw = r.Entries[0].Value
}
}
}

if resolverRec == nil {
return &PluginDeliverResponse{Error: ErrResolverNotRegistered()}
}
if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}

// COI-1: resolver must not hold a position in this market
coiPosQId := nextQueryId()
coiPosResp, coiPosErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: coiPosQId, Key: KeyForPosition(msg.MarketId, msg.ResolverAddress)},
},
})
if coiPosErr != nil {
return &PluginDeliverResponse{Error: ErrStateReadFailed()}
}
for _, r := range coiPosResp.Results {
if r.QueryId == coiPosQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
resolPos := &PositionState{}
if pe := Unmarshal(r.Entries[0].Value, resolPos); pe != nil {
return &PluginDeliverResponse{Error: ErrUnmarshalFailed()}
}
if resolPos.SharesYes > 0 || resolPos.SharesNo > 0 {
return &PluginDeliverResponse{Error: ErrResolverHasPosition()}
}
}
}

// COI-2: market creator cannot be the resolver
if bytesEqual(market.Creator, msg.ResolverAddress) {
return &PluginDeliverResponse{Error: ErrCreatorCannotResolve()}
}

if resolverRec.RrsScore < MIN_RRS_TO_PROPOSE {
return &PluginDeliverResponse{Error: ErrResolverSuspended()}
}

if market.Status == STATUS_OPEN {
if now <= market.ExpiryTime {
return &PluginDeliverResponse{Error: ErrMarketNotExpired()}
}
} else if market.Status != STATUS_EXPIRED {
return &PluginDeliverResponse{Error: ErrMarketNotExpired()}
}

if proposalRaw != nil {
return &PluginDeliverResponse{Error: ErrAlreadyProposed()}
}

minBond := ComputeMinBond(market)
if msg.ProposalBond < minBond {
return &PluginDeliverResponse{Error: ErrInsufficientBond()}
}
if resolverRec.StakeAmount < msg.ProposalBond {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

// Layer 3: dynamic ELEVATED_RISK re-evaluation at propose time.
// A market that grew beyond the threshold after creation gets upgraded
// to the 7-person panel if disputed — regardless of creation-time flag.
poolQId := nextQueryId()
poolResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId, Key: KeyForMarketPool(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if poolResp.Error != nil {
return &PluginDeliverResponse{Error: poolResp.Error}
}
for _, r := range poolResp.Results {
if r.QueryId == poolQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
pool := &Pool{}
if pe := Unmarshal(r.Entries[0].Value, pool); pe == nil {
if pool.Amount >= ELEVATED_RISK_THRESHOLD {
market.ElevatedRisk = true
}
}
}
}

market.Status             = STATUS_PROPOSED
resolverRec.StakeAmount  -= msg.ProposalBond

proposal := &ProposalRecord{
ResolverAddr:    msg.ResolverAddress,
ProposedOutcome: msg.ProposedOutcome,
ProposalBond:    msg.ProposalBond,
ProposalBlock:   now,
Status:          PROPOSAL_OPEN,
}

resolverState := &ResolverState{ResolverAddress: msg.ResolverAddress}

rawM, pe := SafeMarshal(market)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawRR, pe := SafeMarshal(resolverRec)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawPR, pe := SafeMarshal(proposal)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawRS, pe := SafeMarshal(resolverState)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),              Value: rawM},
{Key: KeyForResolverRecord(msg.ResolverAddress), Value: rawRR},
{Key: KeyForProposal(msg.MarketId),             Value: rawPR},
{Key: KeyForResolverState(msg.MarketId),        Value: rawRS},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
