package contract

import (
	"bytes"
	"encoding/binary"
	"math/rand"
)

/* This file contains the base contract implementation that overrides the basic 'transfer' functionality */

// PluginConfig: the configuration of the contract
var ContractConfig = &PluginConfig{
	Name:                  "send",
	Id:                    1,
	Version:               1,
	SupportedTransactions: []string{"send"},
}

// Contract() defines the smart contract that implements the extended logic of the nested chain
type Contract struct {
	Config    Config
	FSMConfig *PluginFSMConfig // fsm configuration
	plugin    *Plugin          // plugin connection
	fsmId     uint64           // the id of the requesting fsm
}

// Genesis() implements logic to import a json file to create the state at height 0 and export the state at any height
func (c *Contract) Genesis(_ *PluginGenesisRequest) *PluginGenesisResponse {
	return &PluginGenesisResponse{} // TODO map out original token holders
}

// BeginBlock() is code that is executed at the start of `applying` the block
func (c *Contract) BeginBlock(_ *PluginBeginRequest) *PluginBeginResponse {
	return &PluginBeginResponse{}
}

// CheckTx() is code that is executed to statelessly validate a transaction
func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
	// validate fee
	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: KeyForFeeParams()},
		}})
	if err == nil {
		err = resp.Error
	}
	// handle error
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	// convert bytes into fee parameters
	minFees := new(FeeParams)
	if err = Unmarshal(resp.Results[0].Entries[0].Value, minFees); err != nil {
		return &PluginCheckResponse{Error: err}
	}
	// check for the minimum fee
	if request.Tx.Fee < minFees.SendFee {
		return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
	}
	// get the message
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	// handle the message
	switch x := msg.(type) {
	case *MessageSend:
		return c.CheckMessageSend(x)
	default:
		return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
	}
}

// DeliverTx() is code that is executed to apply a transaction
func (c *Contract) DeliverTx(request *PluginDeliverRequest) *PluginDeliverResponse {
	// get the message
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// handle the message
	switch x := msg.(type) {
	case *MessageSend:
		return c.DeliverMessageSend(x, request.Tx.Fee)
	default:
		return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
	}
}

// EndBlock() is code that is executed at the end of 'applying' a block
func (c *Contract) EndBlock(_ *PluginEndRequest) *PluginEndResponse {
	return &PluginEndResponse{}
}

// CheckMessageSend() statelessly validates a 'send' message
func (c *Contract) CheckMessageSend(msg *MessageSend) *PluginCheckResponse {
	// check sender address
	if len(msg.FromAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check recipient address
	if len(msg.ToAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check amount
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	// return the authorized signers
	return &PluginCheckResponse{Recipient: msg.ToAddress, AuthorizedSigners: [][]byte{msg.FromAddress}}
}

// DeliverMessageSend() handles a 'send' message
func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64) *PluginDeliverResponse {
	var (
		fromKey, toKey, feePoolKey         []byte
		fromBytes, toBytes, feePoolBytes   []byte
		fromQueryId, toQueryId, feeQueryId = rand.Uint64(), rand.Uint64(), rand.Uint64()
		from, to, feePool                  = new(Account), new(Account), new(Pool)
	)
	// calculate the from key and to key
	fromKey, toKey, feePoolKey = KeyForAccount(msg.FromAddress), KeyForAccount(msg.ToAddress), KeyForFeePool(c.Config.ChainId)
	// get the from and to account
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQueryId, Key: feePoolKey},
			{QueryId: fromQueryId, Key: fromKey},
			{QueryId: toQueryId, Key: toKey},
		}})
	// check for internal error
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// ensure no error fsm error
	if response.Error != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// get the from bytes and to bytes
	for _, resp := range response.Results {
		switch resp.QueryId {
		case fromQueryId:
			fromBytes = resp.Entries[0].Value
		case toQueryId:
			toBytes = resp.Entries[0].Value
		case feeQueryId:
			feePoolBytes = resp.Entries[0].Value
		}
	}
	// add fee to 'amount to deduct'
	amountToDeduct := msg.Amount + fee
	// convert the bytes to account structures
	if err = Unmarshal(fromBytes, from); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(toBytes, to); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// if the account amount is less than the amount to subtract; return insufficient funds
	if from.Amount < amountToDeduct {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	// for self-transfer, use same account data
	if bytes.Equal(fromKey, toKey) {
		to = from
	}
	// subtract from sender
	from.Amount -= amountToDeduct
	// add the fee to the 'fee pool'
	feePool.Amount += fee
	// add to recipient
	to.Amount += msg.Amount
	// convert the accounts to bytes
	fromBytes, err = Marshal(from)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	toBytes, err = Marshal(to)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePoolBytes, err = Marshal(feePool)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// execute writes to the database
	var resp *PluginStateWriteResponse
	// if the from account is drained - delete the from account
	if from.Amount == 0 {
		resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: []*PluginSetOp{
				{Key: feePoolKey, Value: feePoolBytes},
				{Key: toKey, Value: toBytes},
			},
			Deletes: []*PluginDeleteOp{{Key: fromKey}},
		})
	} else {
		resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: []*PluginSetOp{
				{Key: feePoolKey, Value: feePoolBytes},
				{Key: toKey, Value: toBytes},
				{Key: fromKey, Value: fromBytes},
			},
		})
	}
	if err == nil {
		err = resp.Error
	}
	return &PluginDeliverResponse{Error: err}
}

var (
	accountPrefix = []byte{1} // store key prefix for accounts
	poolPrefix    = []byte{2} // store key prefix for pools
	paramsPrefix  = []byte{7} // store key prefix for governance parameters
)

// KeyForAccount() returns the state database key for an account
func KeyForAccount(addr []byte) []byte {
	return JoinLenPrefix(accountPrefix, addr)
}

// KeyForFeeParams() returns the state database key for governance controlled 'fee parameters'
func KeyForFeeParams() []byte {
	return JoinLenPrefix(paramsPrefix, []byte("/f/"))
}

// KeyForFeeParams() returns the state database key for governance controlled 'fee parameters'
func KeyForFeePool(chainId uint64) []byte {
	return JoinLenPrefix(poolPrefix, formatUint64(chainId))
}

func formatUint64(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}
