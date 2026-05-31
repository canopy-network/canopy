package contract

// handler_claim_creator_fee.go — MessageClaimCreatorFee
// Spec: PRIS v1.0-r3
//
// Transfers the full KeyForCreatorFeePool(marketId) balance to market.Creator.
// Condition: market must be STATUS_FINALIZED.
// Signer:    market.Creator only.

func (c *Contract) CheckMessageClaimCreatorFee(msg *MessageClaimCreatorFee) *PluginCheckResponse {
if len(msg.MarketId) != 20 {
return ErrCheckResp(ErrInvalidParam())
}
if len(msg.CreatorAddress) != 20 {
return ErrCheckResp(ErrInvalidAddress())
}
return &PluginCheckResponse{
AuthorizedSigners: [][]byte{msg.CreatorAddress},
}
}

func (c *Contract) DeliverMessageClaimCreatorFee(msg *MessageClaimCreatorFee, fee uint64) *PluginDeliverResponse {
height := GetGlobalHeight()
if height == 0 {
return &PluginDeliverResponse{Error: ErrHeightNotSet()}
}

marketQId  := nextQueryId()
feePoolQId := nextQueryId()
accQId     := nextQueryId()

resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: marketQId,  Key: KeyForMarket(msg.MarketId)},
{QueryId: feePoolQId, Key: KeyForCreatorFeePool(msg.MarketId)},
{QueryId: accQId,     Key: KeyForAccount(msg.CreatorAddress)},
},
})
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if resp.Error != nil {
return &PluginDeliverResponse{Error: resp.Error}
}

market  := &MarketState{}
feePool := &Pool{}
acc     := &Account{}

for _, r := range resp.Results {
if len(r.Entries) == 0 {
continue
}
switch r.QueryId {
case marketQId:
if pe := Unmarshal(r.Entries[0].Value, market); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case feePoolQId:
if pe := Unmarshal(r.Entries[0].Value, feePool); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
case accQId:
if pe := Unmarshal(r.Entries[0].Value, acc); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
}
}

if market.Status != STATUS_FINALIZED {
return &PluginDeliverResponse{Error: ErrMarketNotFinalized()}
}
if !bytesEqual(market.Creator, msg.CreatorAddress) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}
if feePool.Amount == 0 {
return &PluginDeliverResponse{Error: ErrEmptyPool()}
}
if fee > 0 && acc.Amount > ^uint64(0)-feePool.Amount {
return &PluginDeliverResponse{Error: ErrInvalidAmount()}
}

payout        := feePool.Amount
acc.Amount    += payout
feePool.Amount = 0

rawFeePool, pe := SafeMarshal(feePool)
if pe != nil { return &PluginDeliverResponse{Error: pe} }
rawAcc, pe := SafeMarshal(acc)
if pe != nil { return &PluginDeliverResponse{Error: pe} }

wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: KeyForCreatorFeePool(msg.MarketId), Value: rawFeePool},
{Key: KeyForAccount(msg.CreatorAddress),  Value: rawAcc},
},
})
if pe := errCheckWrite(wr, werr); pe != nil {
return &PluginDeliverResponse{Error: pe}
}
return &PluginDeliverResponse{}
}
