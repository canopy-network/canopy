package contract

// handler_unstake_resolver.go — MessageUnstakeResolver
// Spec: PRIS v1.0-r3 unstake extension
//
// Allows a resolver to unstake partially or fully.
// Rules:
//   - Cannot unstake if active open proposal exists (slash-evasion guard)
//   - Cannot partial unstake below MIN_RESOLVER_STAKE
//   - Partial unstake: RRS -10, 120,960 block unbonding period
//   - Full exit (amount=0 or amount=stakeAmount): RRS reset to PRIS_RRS_INITIAL, unbonding period
//   - Tokens locked in UnbondingRecord until release height
//   - claim_unbonded_stake TX releases tokens after unbonding period

func (c *Contract) CheckMessageUnstakeResolver(msg *MessageUnstakeResolver) *PluginCheckResponse {
if len(msg.ResolverAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.ResolverAddress},
}
}

func (c *Contract) DeliverMessageUnstakeResolver(msg *MessageUnstakeResolver, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

// ── Batch read ────────────────────────────────────────────────────────
recQId      := nextQueryId()
proposalQId := nextQueryId()
accQId      := nextQueryId()

recKey      := KeyForResolverRecord(msg.ResolverAddress)
proposalKey := KeyForProposal(msg.ResolverAddress)
accKey      := KeyForAccount(msg.ResolverAddress)

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: recQId,      Key: recKey},
{QueryId: proposalQId, Key: proposalKey},
{QueryId: accQId,      Key: accKey},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

var record *ResolverRecord
acc := &Account{}

for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
continue
}
switch r.QueryId {
case recQId:
record = &ResolverRecord{}
if pe := Unmarshal(r.Entries[0].Value, record); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case proposalQId:
// Slash-evasion guard: active proposal exists — reject unstake
proposal := &ProposalRecord{}
if pe := Unmarshal(r.Entries[0].Value, proposal); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
if proposal.Status == PROPOSAL_OPEN {
return &PluginDeliverResponse{Error: ErrActiveProposalExists()}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if record == nil {
return &PluginDeliverResponse{Error: ErrResolverNotFound()}
}
if !record.IsActive {
return &PluginDeliverResponse{Error: ErrResolverNotActive()}
}

// ── Determine unstake amount and type ─────────────────────────────────
fullExit := msg.Amount == 0 || msg.Amount >= record.StakeAmount
unstakeAmt := msg.Amount
if fullExit {
unstakeAmt = record.StakeAmount
}

// Partial unstake: remaining stake must stay >= MIN_RESOLVER_STAKE
if !fullExit {
remaining := record.StakeAmount - unstakeAmt
if remaining < MIN_RESOLVER_STAKE {
return &PluginDeliverResponse{Error: ErrInsufficientResolverStake()}
}
}

// ── Apply RRS penalty ─────────────────────────────────────────────────
if fullExit {
record.RrsScore = PRIS_RRS_INITIAL // reset to Bronze baseline
record.IsActive = false
} else {
if record.RrsScore > PRIS_UNSTAKE_PARTIAL_RRS_HIT {
record.RrsScore -= PRIS_UNSTAKE_PARTIAL_RRS_HIT
} else {
record.RrsScore = PRIS_RRS_FLOOR
}
}

// ── Apply unbonding ───────────────────────────────────────────────────
record.StakeAmount      -= unstakeAmt
record.UnbondingAmount   = unstakeAmt
record.UnbondingReleaseHeight = height + PRIS_UNSTAKE_UNBONDING_BLOCKS

// ── Pay fee from account ──────────────────────────────────────────────
if acc.Amount < fee {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}
acc.Amount -= fee

// ── Marshal ───────────────────────────────────────────────────────────
rawRec, pe := SafeMarshal(record)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

// ── 2-key atomic write ────────────────────────────────────────────────
sets := []*PluginSetOp{
{Key: recKey, Value: rawRec},
}
var deletes []*PluginDeleteOp
if acc.Amount == 0 {
deletes = []*PluginDeleteOp{{Key: accKey}}
} else {
sets = append(sets, &PluginSetOp{Key: accKey, Value: rawAcc})
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
