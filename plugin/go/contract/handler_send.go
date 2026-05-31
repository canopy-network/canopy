package contract

import "bytes"

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

func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64) *PluginDeliverResponse {
isSelfTransfer := bytes.Equal(msg.FromAddress, msg.ToAddress)

fromQId  := nextQueryId()
toQId    := nextQueryId()
feeQId        := nextQueryId()
	gTreasuryQId  := nextQueryId()

fromKey    := KeyForAccount(msg.FromAddress)
toKey      := KeyForAccount(msg.ToAddress)
feePoolKey    := KeyForFeePool(c.Config.ChainId)
	gTreasuryKey  := KeyForTreasuryPool()

var readKeys []*PluginKeyRead
if isSelfTransfer {
readKeys = []*PluginKeyRead{
{QueryId: fromQId, Key: fromKey},
{QueryId: feeQId,  Key: feePoolKey},
	{QueryId: gTreasuryQId, Key: gTreasuryKey},
}
} else {
readKeys = []*PluginKeyRead{
{QueryId: fromQId, Key: fromKey},
{QueryId: toQId,   Key: toKey},
{QueryId: feeQId,  Key: feePoolKey},
	{QueryId: gTreasuryQId, Key: gTreasuryKey},
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
feePool   := &Pool{}
	gTreasury := &Pool{}

for _, r := range resp.Results {
if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
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
case gTreasuryQId:
if pe := Unmarshal(r.Entries[0].Value, gTreasury); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

// Self-transfer: pay fee only, amount stays
if isSelfTransfer {
if from.Amount < fee {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}
from.Amount -= fee
feePool.Amount   += fee / 2
	gTreasury.Amount += fee - fee/2

rawFrom, pe := SafeMarshal(from)
if pe != nil {
return &PluginDeliverResponse{Error: pe}
}
rawFee, pe := SafeMarshal(feePool)
	if pe != nil {
		return &PluginDeliverResponse{Error: pe}
	}
	rawGTreasury, pe := SafeMarshal(gTreasury)
	if pe != nil {
		return &PluginDeliverResponse{Error: pe}
	}

var wr *PluginStateWriteResponse
if from.Amount == 0 {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets:    []*PluginSetOp{{Key: feePoolKey, Value: rawFee}, {Key: gTreasuryKey, Value: rawGTreasury}},
Deletes: []*PluginDeleteOp{{Key: fromKey}},
})
} else {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: fromKey, Value: rawFrom},
{Key: feePoolKey, Value: rawFee},
				{Key: gTreasuryKey, Value: rawGTreasury},
},
})
}
if pe := errCheckWrite(wr, err); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}

// Normal transfer
if fee > 0 && msg.Amount > ^uint64(0)-fee {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}
totalDeduct := msg.Amount + fee
if from.Amount < totalDeduct {
return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
}

from.Amount    -= totalDeduct
to.Amount      += msg.Amount
feePool.Amount   += fee / 2
	gTreasury.Amount += fee - fee/2

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
	rawGTreasury, pe := SafeMarshal(gTreasury)
	if pe != nil {
		return &PluginDeliverResponse{Error: pe}
	}

var wr *PluginStateWriteResponse
if from.Amount == 0 {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets:    []*PluginSetOp{{Key: toKey, Value: rawTo}, {Key: feePoolKey, Value: rawFee}, {Key: gTreasuryKey, Value: rawGTreasury}},
Deletes: []*PluginDeleteOp{{Key: fromKey}},
})
} else {
wr, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: fromKey, Value: rawFrom},
{Key: toKey, Value: rawTo},
{Key: feePoolKey, Value: rawFee},
				{Key: gTreasuryKey, Value: rawGTreasury},
},
})
}
if pe := errCheckWrite(wr, err); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
