package contract

// handler_file_dispute.go — MessageFileDispute
// Spec: PORS v1.0-r2-CORRECTED (P4, P5, P7)
//
// Filed by a disputer after propose_outcome commits a STATUS_PROPOSED market.
// Derives and stores the panel member list atomically at filing time (P4).
// Panel excludes the proposer and disputer (PORS spec).
// Panel size: MIN_PANEL_SIZE normally, ELEVATED_RISK_PANEL_SIZE for elevated-risk markets (P7).
//
// CheckTx:  market_id 20 bytes, disputer_address 20 bytes, dispute_bond non-zero.
//           Zero StateRead (AUDIT-8).
// DeliverTx:
//   Batch read: market + proposal + resolver record (proposer) + disputer account + entropy
//   Range scan: all ResolverRecords (0x16 prefix) for panel candidate pool
//   Derive panel deterministically from entropy seed
//   4-key atomic write: MarketState + DisputeRecord + disputer Account + fee pool

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

// ── Batch read 1: market, proposal, disputer account, entropy, fee pool ──
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
case dispAccQId:
if len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, disputer); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
case entropyQId:
if len(r.Entries) > 0 {
acc := &PanelEntropyAccum{}
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
entropyVal = acc.Accumulator
}
case feeQId:
if len(r.Entries) > 0 {
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

// Dispute window: proposal_block + ComputeDisputeBlocks(open_time, expiry_time).
disputeWindow   := ComputeDisputeBlocks(market.OpenTime, market.ExpiryTime)
disputeDeadline := proposal.ProposalBlock + disputeWindow
if now > disputeDeadline {
return &PluginDeliverResponse{Error: ErrDisputeWindowClosed()}
}

// Only one dispute per market.
// (idempotency: DisputeRecord written atomically — STATUS_DISPUTED set simultaneously)
if market.Status == STATUS_DISPUTED {
return &PluginDeliverResponse{Error: ErrAlreadyDisputed()}
}

// Cost check: dispute_bond + fee.
if fee > 0 && msg.DisputeBond > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalCost := msg.DisputeBond + fee
if disputer.Amount < totalCost {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

// ── Range scan: all ResolverRecords for panel candidate pool ─────────────
resolverRangeQId := nextQueryId()
rangeResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Ranges: []*PluginRangeRead{
{
QueryId: resolverRangeQId,
Prefix:  resolverRecordPrefix,
Limit:   0, // no limit — need all candidates
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

// Collect candidate addresses: exclude proposer and disputer (PORS spec).
var candidates [][]byte
for _, r := range rangeResp.Results {
if r.QueryId != resolverRangeQId {
continue
}
for _, entry := range r.Entries {
rec := &ResolverRecord{}
if pe := Unmarshal(entry.Value, rec); pe != nil {
continue // skip malformed entries
}
if bytesEqual(rec.ResolverAddress, proposal.ResolverAddr) {
continue // exclude proposer
}
if bytesEqual(rec.ResolverAddress, msg.DisputerAddress) {
continue // exclude disputer
}
// Only include resolvers with sufficient RRS score.
if rec.RrsScore < MIN_RRS_TO_PROPOSE {
continue
}
candidates = append(candidates, rec.ResolverAddress)
}
}

// Panel size determined by elevated_risk flag (P7).
var panelSize uint32
if market.ElevatedRisk {
panelSize = ELEVATED_RISK_PANEL_SIZE
} else {
panelSize = MIN_PANEL_SIZE
}

// Derive panel deterministically from entropy seed.
// seed = entropyVal XOR (now * FIBONACCI_HASH_CONSTANT)
seed := entropyVal ^ (now * FIBONACCI_HASH_CONSTANT)
panel := derivePanel(candidates, int(panelSize), seed)

// If not enough eligible candidates, use all of them.
// The panel may be smaller than panelSize — this is acceptable per spec.

// ── Mutate in memory ──────────────────────────────────────────────────────
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

// ── Marshal all ───────────────────────────────────────────────────────────
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
{Key: marketKey,                   Value: rawM},
{Key: KeyForDispute(msg.MarketId), Value: rawD},
{Key: feePoolKey,                  Value: rawFee},
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

// derivePanel selects up to n addresses from candidates using a deterministic
// Fisher-Yates shuffle seeded with the given uint64 seed.
// Returns fewer than n if candidates has fewer than n entries.
func derivePanel(candidates [][]byte, n int, seed uint64) [][]byte {
if len(candidates) == 0 || n == 0 {
return nil
}
// Copy to avoid mutating the original slice.
pool := make([][]byte, len(candidates))
copy(pool, candidates)

// Deterministic Fisher-Yates using a simple LCG seeded with seed.
s := seed
lcgNext := func() uint64 {
// LCG parameters from Knuth TAOCP.
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
