package contract

// handler_forfeit_position.go — Issue-2 fix
// Allows a resolver to voluntarily exit a position before proposing/resolving.
// Refunds full CostPaid, zeroes shares atomically. Satisfies COI-1 requirement
// without permanent disqualification.

func (c *Contract) CheckMessageForfeitPosition(msg *MessageForfeitPosition) *PluginCheckResponse {
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

func (c *Contract) DeliverMessageForfeitPosition(msg *MessageForfeitPosition, fee uint64) *PluginDeliverResponse {
	now := GetGlobalHeight()
	if now == 0 {
		return &PluginDeliverResponse{Error: ErrHeightNotSet()}
	}
	posQId  := nextQueryId()
	accQId  := nextQueryId()
	poolQId := nextQueryId()
	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: posQId,  Key: KeyForPosition(msg.MarketId, msg.ResolverAddress)},
			{QueryId: accQId,  Key: KeyForAccount(msg.ResolverAddress)},
			{QueryId: poolQId, Key: KeyForMarketPool(msg.MarketId)},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: ErrStateReadFailed()}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}
	var position *PositionState
	var account  *Account
	var pool     *Pool
	for _, r := range resp.Results {
		if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
			continue
		}
		switch r.QueryId {
		case posQId:
			position = &PositionState{}
			if pe := Unmarshal(r.Entries[0].Value, position); pe != nil {
				return &PluginDeliverResponse{Error: ErrUnmarshalFailed()}
			}
		case accQId:
			account = &Account{}
			if pe := Unmarshal(r.Entries[0].Value, account); pe != nil {
				return &PluginDeliverResponse{Error: ErrUnmarshalFailed()}
			}
		case poolQId:
			pool = &Pool{}
			if pe := Unmarshal(r.Entries[0].Value, pool); pe != nil {
				return &PluginDeliverResponse{Error: ErrUnmarshalFailed()}
			}
		}
	}
	if position == nil || (position.SharesYes == 0 && position.SharesNo == 0) {
		return &PluginDeliverResponse{Error: ErrNoPosition()}
	}
	if account == nil {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	if pool == nil {
		return &PluginDeliverResponse{Error: ErrMarketNotFound()}
	}
	// Overflow guard
	if account.Amount > ^uint64(0)-position.CostPaid {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	// Pool must cover the refund
	if pool.Amount < position.CostPaid {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	refund         := position.CostPaid
	account.Amount += refund
	pool.Amount    -= refund
	// Zero the position
	position.SharesYes = 0
	position.SharesNo  = 0
	position.CostPaid  = 0
	rawPos,  pe := SafeMarshal(position)
	if pe != nil { return &PluginDeliverResponse{Error: pe} }
	rawAcc,  pe := SafeMarshal(account)
	if pe != nil { return &PluginDeliverResponse{Error: pe} }
	rawPool, pe := SafeMarshal(pool)
	if pe != nil { return &PluginDeliverResponse{Error: pe} }
	wr, werr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForPosition(msg.MarketId, msg.ResolverAddress), Value: rawPos},
			{Key: KeyForAccount(msg.ResolverAddress),                Value: rawAcc},
			{Key: KeyForMarketPool(msg.MarketId),                    Value: rawPool},
		},
	})
	if pe := errCheckWrite(wr, werr); pe != nil {
		return &PluginDeliverResponse{Error: pe}
	}
	return &PluginDeliverResponse{}
}
