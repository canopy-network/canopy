package contract

import "bytes"

// handler_propose_outcome.go — MessageProposeOutcome
// Spec: PORS v1.0-r2-CORRECTED
//
// NF-5 FIX: writes 0x13 ResolverState as the 4th atomic key.
//           0x13 was never written before — auth check always failed.
// NF-6 FIX: accepts STATUS_OPEN markets where now > ExpiryTime and transitions
//           directly to STATUS_PROPOSED. STATUS_EXPIRED is never persisted.
//
// CheckTx:  market_id 20 bytes, resolver_address 20 bytes, proposal_bond non-zero.
//           No status check (stateful — DeliverTx enforces). Zero StateRead (AUDIT-8).
// DeliverTx:
//   Batch read: global ResolverRecord (0x16) + MarketState (0x10) + ProposalRecord (0x17)
//   Auth: RRS score check
//   NF-6: accept STATUS_OPEN if now > ExpiryTime — inline transition to PROPOSED
//   Idempotency: ProposalRecord nil = first call
//   Bond sufficiency check
//   4-key atomic write: MarketState + ResolverRecord + ProposalRecord + ResolverState (NF-5)

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
// NF-6: no status check here — status is stateful, DeliverTx enforces.
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageProposeOutcome(msg *MessageProposeOutcome, fee uint64) *PluginDeliverResponse {
now := GetGlobalHeight()
if now == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read: resolver record + market state + proposal (idempotency) ───
// CRIT-3: QueryId per PluginKeyRead.
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
if len(r.Entries) == 0 {
return &PluginDeliverResponse{Error: ErrResolverNotRegistered()}
}
resolverRec = &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, resolverRec); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
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

// ── Global registry check (0x16) ─────────────────────────────────────────
if resolverRec.RrsScore < MIN_RRS_TO_PROPOSE {
return &PluginDeliverResponse{Error: ErrResolverSuspended()}
}

	// ── COI-1: resolver must not hold a position in this market ─────────────────
	resolPosQId := nextQueryId()
	posResp, posErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: resolPosQId, Key: KeyForPosition(msg.MarketId, msg.ResolverAddress)},
		},
	})
	if posErr != nil {
		return &PluginDeliverResponse{Error: posErr}
	}
	for _, r := range posResp.Results {
		if r.QueryId == resolPosQId && len(r.Entries) > 0 && len(r.Entries[0].Value) > 0 {
			resolPos := &PositionState{}
			if pe := Unmarshal(r.Entries[0].Value, resolPos); pe != nil {
				return &PluginDeliverResponse{Error: pe}
			}
			if resolPos.SharesYes > 0 || resolPos.SharesNo > 0 {
				return &PluginDeliverResponse{Error: ErrResolverHasPosition()}
			}
		}
	}

// ── COI-2: market creator cannot be the resolver ────────────────────────────
	if bytes.Equal(market.Creator, msg.ResolverAddress) {
		return &PluginDeliverResponse{Error: ErrCreatorCannotResolve()}
	}

	// ── Market status check ───────────────────────────────────────────────────
// NF-6 FIX: accept STATUS_OPEN if now > ExpiryTime — inline transition.
// STATUS_OPEN -> STATUS_PROPOSED is atomic; STATUS_EXPIRED never persisted.
if market.Status == STATUS_OPEN {
if now <= market.ExpiryTime {
return &PluginDeliverResponse{Error: ErrMarketNotExpired()}
}
// now > ExpiryTime: we are the expiry trigger.
// market.Status will be set to STATUS_PROPOSED in the atomic write below.
} else if market.Status != STATUS_EXPIRED {
// Any other status (PROPOSED, DISPUTED, FINALIZED, CANCELLED, VOIDED) is invalid.
return &PluginDeliverResponse{Error: ErrMarketNotExpired()}
}

// ── Idempotency guard: ProposalRecord nil = first call ────────────────────
if proposalRaw != nil {
return &PluginDeliverResponse{Error: ErrAlreadyProposed()}
}

// ── Bond sufficiency check ────────────────────────────────────────────────
minBond := ComputeMinBond(market)
if msg.ProposalBond < minBond {
return &PluginDeliverResponse{Error: ErrInsufficientBond()}
}
if resolverRec.StakeAmount < msg.ProposalBond {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

// ── Mutate in memory ──────────────────────────────────────────────────────
market.Status             = STATUS_PROPOSED
resolverRec.StakeAmount  -= msg.ProposalBond

proposal := &ProposalRecord{
ResolverAddr:    msg.ResolverAddress,
ProposedOutcome: msg.ProposedOutcome,
ProposalBond:    msg.ProposalBond,
ProposalBlock:   now,
Status:          PROPOSAL_OPEN,
}

// NF-5 FIX: ResolverState for this market — written here for the first time.
// This is what makes the auth check in ResolveMarket work.
resolverState := &ResolverState{ResolverAddress: msg.ResolverAddress}

// ── Marshal all ───────────────────────────────────────────────────────────
rawM, pe := SafeMarshal(market)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawRR, pe := SafeMarshal(resolverRec)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawPR, pe := SafeMarshal(proposal)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawRS, pe := SafeMarshal(resolverState)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// ── NF-5 FIX: 4-key atomic write ─────────────────────────────────────────
// All four commit together or none do.
// On failure: market.Status stays OPEN/EXPIRED, 0x13 stays nil, retry is safe.
wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForMarket(msg.MarketId),              Value: rawM},
{Key: KeyForResolverRecord(msg.ResolverAddress), Value: rawRR},
{Key: KeyForProposal(msg.MarketId),             Value: rawPR},
{Key: KeyForResolverState(msg.MarketId),        Value: rawRS}, // NF-5
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
