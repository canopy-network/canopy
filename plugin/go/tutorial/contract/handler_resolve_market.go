package contract

// handler_resolve_market.go — MessageResolveMarket
// Spec: ADLMSR v5.6.6-r2-CORRECTED
//
// Called by the registered resolver after the resolution window opens.
// Sets market to STATUS_RESOLVED (ADLMSR intermediate — PORS leads to FINALIZED).
// Pays bounty to resolver and bond return to creator from the market pool.
//
// CheckTx:  market_id 20 bytes, resolver_address 20 bytes. Zero StateRead (AUDIT-8).
// DeliverTx:
//   Batch read 1: market, resolver state (auth), outcome (idempotency)
//   R1 FIX: auth check BEFORE idempotency guard
//   Auto-cancel check if outside resolution window
//   Batch read 2: pool, resolver account, creator account, treasury
//   6-key atomic write (NEW-1 + R1)
//   R5: two batch reads are deliberate — avoids pool/account reads for bad callers

const bountyAmount uint64 = 50_000_000  // 50 PRX
const bondAmount   uint64 = 100_000_000 // 100 PRX

func (c *Contract) CheckMessageResolveMarket(msg *MessageResolveMarket) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageResolveMarket(msg *MessageResolveMarket, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read 1: market, resolver state, outcome (idempotency) ──────────
marketQId   := nextQueryId()
resolverQId := nextQueryId()
outcomeQId  := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,   Key: KeyForMarket(msg.MarketId)},
{QueryId: resolverQId, Key: KeyForResolverState(msg.MarketId)},
{QueryId: outcomeQId,  Key: KeyForOutcome(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var market     *MarketState
var resolver   *ResolverState
var outcomeRaw []byte

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
case resolverQId:
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrNoResolverRegistered()}
}
resolver = &ResolverState{}
if pe := Unmarshal(r.Entries[0].Value, resolver); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case outcomeQId:
if len(r.Entries) > 0 {
outcomeRaw = r.Entries[0].Value
}
}
}

if market == nil {
return &PluginDeliverResponse{Error: ErrMarketNotFound()}
}
if resolver == nil {
return &PluginDeliverResponse{Error: ErrNoResolverRegistered()}
}

// ── R1 FIX: AUTH BEFORE IDEMPOTENCY ──────────────────────────────────────
// Wrong resolver must never receive success on retry.
if !bytesEqual(resolver.ResolverAddress, msg.ResolverAddress) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}

// ── Auto-cancel check + market state refresh ─────────────────────────────
withinWindow := now >= market.ExpiryTime+RESOLUTION_DELAY_BLOCKS &&
now <= market.ExpiryTime+RESOLUTION_DELAY_BLOCKS+GRACE_PERIOD_BLOCKS
if !withinWindow {
if pe := c.CheckAutoCancel(msg.MarketId); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
// Re-read market after potential auto-cancel.
refreshQId := nextQueryId()
resp2, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: refreshQId, Key: KeyForMarket(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp2.Error != nil {
return &PluginDeliverResponse{Error: resp2.Error}
}
for _, r := range resp2.Results {
if r.QueryId == refreshQId && len(r.Entries) > 0 {
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}
}

if market.Status == STATUS_CANCELLED {
return &PluginDeliverResponse{Error: ErrMarketCancelled()}
}

// ── Idempotency guard — AFTER auth (R1 fix) ───────────────────────────────
// OutcomeState exists iff all 6 keys committed in the atomic write below.
if outcomeRaw != nil {
return &PluginDeliverResponse{}
}

// ── Timing and status guards ──────────────────────────────────────────────
if market.Status != STATUS_OPEN {
return &PluginDeliverResponse{Error: ErrMarketNotOpen()}
}
if now < market.ExpiryTime+RESOLUTION_DELAY_BLOCKS {
return &PluginDeliverResponse{Error: ErrResolutionTooEarly()}
}

// ── Batch read 2: keys that will be mutated ───────────────────────────────
// R5: deferred until after auth + timing pass to avoid wasted reads.
poolQId     := nextQueryId()
resolAccQId := nextQueryId()
creatAccQId := nextQueryId()
treasQId    := nextQueryId()

payResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: poolQId,     Key: KeyForMarketPool(msg.MarketId)},
{QueryId: resolAccQId, Key: KeyForAccount(msg.ResolverAddress)},
{QueryId: creatAccQId, Key: KeyForAccount(market.Creator)},
{QueryId: treasQId,    Key: KeyForTreasuryReserve(msg.MarketId)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if payResp.Error != nil {
return &PluginDeliverResponse{Error: payResp.Error}
}

mPool       := &Pool{}
resolverAcc := &Account{}
creatorAcc  := &Account{}
tres        := &TreasuryReserve{}

for _, r := range payResp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case poolQId:
if pe := Unmarshal(r.Entries[0].Value, mPool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case resolAccQId:
if pe := Unmarshal(r.Entries[0].Value, resolverAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case creatAccQId:
if pe := Unmarshal(r.Entries[0].Value, creatorAcc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case treasQId:
if pe := Unmarshal(r.Entries[0].Value, tres); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if mPool.Amount < bountyAmount+bondAmount {
return &PluginDeliverResponse{Error: ErrInsufficientPoolFunds()}
}

// ── Mutate in memory ──────────────────────────────────────────────────────
market.Status      = STATUS_RESOLVED // ADLMSR intermediate — PORS leads to FINALIZED
mPool.Amount      -= bountyAmount + bondAmount
resolverAcc.Amount += bountyAmount
creatorAcc.Amount  += bondAmount
tres.LockedReserve  = 0
outcome := &OutcomeState{WinningOutcome: msg.WinningOutcome, ResolvedAt: now}

// ── Marshal all ───────────────────────────────────────────────────────────
rawM, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawO, pe := SafeMarshal(outcome)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawT, pe := SafeMarshal(tres)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawMP, pe := SafeMarshal(mPool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawRA, pe := SafeMarshal(resolverAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawCA, pe := SafeMarshal(creatorAcc)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// ── 6-key atomic write (NEW-1 + R1) ──────────────────────────────────────
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),           Value: rawM},
{Key: KeyForOutcome(msg.MarketId),           Value: rawO},
{Key: KeyForTreasuryReserve(msg.MarketId),   Value: rawT},
{Key: KeyForMarketPool(msg.MarketId),        Value: rawMP},
{Key: KeyForAccount(msg.ResolverAddress),    Value: rawRA},
{Key: KeyForAccount(market.Creator),         Value: rawCA},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
