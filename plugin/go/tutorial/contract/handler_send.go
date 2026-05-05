package contract

import "bytes"

// ═══════════════════════════════════════════════════════════════════════════════
// handler_send.go — MessageSend
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED
//
// Moves $PRX between addresses. Essential for the Praxis token economy —
// without send, winnings, bounties, and slash proceeds cannot leave the protocol.
//
// CheckTx:  validates addresses (20 bytes each), non-zero amount
// DeliverTx: reads sender + recipient + fee pool, deducts amount+fee from
//            sender, credits amount to recipient, credits fee to pool.
//            Deletes sender account if balance reaches zero (state minimisation).
//            Self-transfer handled separately to avoid pointer aliasing.
// ═══════════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// CHECKTX
// ─────────────────────────────────────────────────────────────────────────────

func (c *Contract) CheckMessageSend(msg *MessageSend) *PluginCheckResponse {
if len(msg.FromAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if len(msg.ToAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
if msg.Amount == 0 {
return ErrCheckResp(ErrInvalidAmount())
}
return &PluginCheckResponse{
Recipient:         msg.ToAddress,
AuthorizedSigners: [][]byte{msg.FromAddress},
}
}

// ─────────────────────────────────────────────────────────────────────────────
// DELIVERTX
// ─────────────────────────────────────────────────────────────────────────────

func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64) *PluginDeliverResponse {
isSelfTransfer := bytes.Equal(msg.FromAddress, msg.ToAddress)

fromQId  := nextQueryId()
toQId    := nextQueryId()
feeQId   := nextQueryId()

fromKey    := KeyForAccount(msg.FromAddress)
toKey      := KeyForAccount(msg.ToAddress)
feePoolKey := KeyForFeePool(c.Config.ChainId)

// For self-transfers only read two keys — avoids pointer aliasing on
// the underlying account bytes when from and to are the same address.
var readKeys []*PluginKeyRead
if isSelfTransfer {
readKeys = []*PluginKeyRead{
{QueryId: fromQId, Key: fromKey},
{QueryId: feeQId, Key: feePoolKey},
}
} else {
readKeys = []*PluginKeyRead{
{QueryId: fromQId, Key: fromKey},
{QueryId: toQId, Key: toKey},
{QueryId: feeQId, Key: feePoolKey},
}
}

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{Keys: readKeys})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

from    := &Account{}
to      := &Account{}
feePool := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case fromQId:
if pe := Unmarshal(r.Entries[0].Value, from); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case toQId:
if pe := Unmarshal(r.Entries[0].Value, to); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feeQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

// ── Self-transfer: sender pays fee only, amount stays in same account ──
if isSelfTransfer {
if from.Amount < fee {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}
from.Amount -= fee
feePool.Amount += fee

rawFrom, pe := SafeMarshal(from)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawFee, pe := SafeMarshal(feePool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

var wr *PluginStateWriteResponse
if from.Amount == 0 {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets:    []*PluginSetOp{{Key: feePoolKey, Value: rawFee}},
Deletes: []*PluginDeleteOp{{Key: fromKey}},
})
} else {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: fromKey, Value: rawFrom},
{Key: feePoolKey, Value: rawFee},
},
})
}
if pe := errCheckWrite(wr, err); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}

// ── Normal transfer ────────────────────────────────────────────────────
// Guard against overflow: amount + fee could wrap if amount is near MaxUint64.
if fee > 0 && msg.Amount > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalDeduct := msg.Amount + fee
if from.Amount < totalDeduct {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

from.Amount    -= totalDeduct
to.Amount      += msg.Amount
feePool.Amount += fee

rawFrom, pe := SafeMarshal(from)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawTo, pe := SafeMarshal(to)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawFee, pe := SafeMarshal(feePool)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}

// Delete sender account if drained — keeps state minimal.
var wr *PluginStateWriteResponse
if from.Amount == 0 {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: toKey, Value: rawTo},
{Key: feePoolKey, Value: rawFee},
},
Deletes: []*PluginDeleteOp{{Key: fromKey}},
})
} else {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: fromKey, Value: rawFrom},
{Key: toKey, Value: rawTo},
{Key: feePoolKey, Value: rawFee},
},
})
}
if pe := errCheckWrite(wr, err); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
