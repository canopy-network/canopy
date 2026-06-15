package contract

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file contains the base contract implementation that overrides the basic 'transfer' functionality */

// PluginConfig: the configuration of the contract
var ContractConfig = &PluginConfig{
	Name:                  "go_plugin_contract",
	Id:                    1,
	Version:               1,
	SupportedTransactions: []string{"send", "reward", "faucet"},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageReward",
		"type.googleapis.com/types.MessageFaucet",
	},
	EventTypeUrls: nil,
	// CustomStatePrefixes declares the key prefixes this plugin owns for its custom records. Canopy
	// validates these at handshake and panics if any collides with a core-reserved prefix (1-15).
	CustomStatePrefixes: [][]byte{faucetPrefix, rewardPrefix},
}

// init sets FileDescriptorProtos after ensuring .pb.go files are initialized
func init() {
	// Explicitly initialize the proto files first to ensure File_*_proto are set
	file_account_proto_init()
	file_event_proto_init()
	file_plugin_proto_init()
	file_tx_proto_init()

	var fds [][]byte
	// Include google/protobuf/any.proto first as it's a dependency of event.proto and tx.proto
	for _, file := range []protoreflect.FileDescriptor{
		anypb.File_google_protobuf_any_proto,
		File_account_proto, File_event_proto, File_plugin_proto, File_tx_proto,
	} {
		fd, _ := proto.Marshal(protodesc.ToFileDescriptorProto(file))
		fds = append(fds, fd)
	}
	ContractConfig.FileDescriptorProtos = fds
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
	case *MessageReward:
		return c.CheckMessageReward(x)
	case *MessageFaucet:
		return c.CheckMessageFaucet(x)
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
	case *MessageReward:
		return c.DeliverMessageReward(x, request.Tx.Fee)
	case *MessageFaucet:
		return c.DeliverMessageFaucet(x)
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

// CheckMessageFaucet() statelessly validates a 'faucet' message
func (c *Contract) CheckMessageFaucet(msg *MessageFaucet) *PluginCheckResponse {
	// check signer address
	if len(msg.SignerAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check recipient address
	if len(msg.RecipientAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check amount
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	// the signer authorizes the faucet (they're requesting tokens for testing)
	return &PluginCheckResponse{
		Recipient:         msg.RecipientAddress,
		AuthorizedSigners: [][]byte{msg.SignerAddress},
	}
}

// CheckMessageReward() statelessly validates a 'reward' message
func (c *Contract) CheckMessageReward(msg *MessageReward) *PluginCheckResponse {
	// check admin address
	if len(msg.AdminAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check recipient address
	if len(msg.RecipientAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// check amount
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	// the admin (not the recipient) must sign to authorize the mint
	return &PluginCheckResponse{
		Recipient:         msg.RecipientAddress,
		AuthorizedSigners: [][]byte{msg.AdminAddress},
	}
}

// DeliverMessageSend() handles a 'send' message
func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageSend called: from=%x to=%x amount=%d fee=%d", msg.FromAddress, msg.ToAddress, msg.Amount, fee)
	var (
		fromKey, toKey, feePoolKey         []byte
		fromBytes, toBytes, feePoolBytes   []byte
		fromQueryId, toQueryId, feeQueryId = rand.Uint64(), rand.Uint64(), rand.Uint64()
		from, to, feePool                  = new(Account), new(Account), new(Pool)
	)
	// calculate the from key and to key
	fromKey, toKey, feePoolKey = KeyForAccount(msg.FromAddress), KeyForAccount(msg.ToAddress), KeyForFeePool(c.Config.ChainId)
	log.Printf("Keys: fromKey=%x toKey=%x feePoolKey=%x", fromKey, toKey, feePoolKey)
	// get the from and to account
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQueryId, Key: feePoolKey},
			{QueryId: fromQueryId, Key: fromKey},
			{QueryId: toQueryId, Key: toKey},
		}})
	// check for internal error
	if err != nil {
		log.Printf("StateRead error: %v", err)
		return &PluginDeliverResponse{Error: err}
	}
	// ensure no error fsm error
	if response.Error != nil {
		log.Printf("StateRead FSM error: %v", response.Error)
		return &PluginDeliverResponse{Error: response.Error}
	}
	log.Printf("StateRead returned %d results", len(response.Results))
	// get the from bytes and to bytes
	for _, resp := range response.Results {
		log.Printf("Result QueryId=%d Entries=%d", resp.QueryId, len(resp.Entries))
		if len(resp.Entries) == 0 {
			log.Printf("WARNING: No entries for QueryId=%d", resp.QueryId)
			continue
		}
		switch resp.QueryId {
		case fromQueryId:
			fromBytes = resp.Entries[0].Value
			log.Printf("fromBytes len=%d", len(fromBytes))
		case toQueryId:
			toBytes = resp.Entries[0].Value
			log.Printf("toBytes len=%d", len(toBytes))
		case feeQueryId:
			feePoolBytes = resp.Entries[0].Value
			log.Printf("feePoolBytes len=%d", len(feePoolBytes))
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
	log.Printf("from.Amount=%d to.Amount=%d feePool.Amount=%d", from.Amount, to.Amount, feePool.Amount)
	// if the account amount is less than the amount to subtract; return insufficient funds
	if from.Amount < amountToDeduct {
		log.Printf("ERROR: Insufficient funds: from.Amount=%d amountToDeduct=%d", from.Amount, amountToDeduct)
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
	log.Printf("AFTER: from.Amount=%d to.Amount=%d feePool.Amount=%d", from.Amount, to.Amount, feePool.Amount)
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
	if err != nil {
		log.Printf("StateWrite internal error: %v", err)
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		log.Printf("StateWrite FSM error: %v", resp.Error)
		return &PluginDeliverResponse{Error: resp.Error}
	}
	log.Printf("StateWrite SUCCESS!")
	return &PluginDeliverResponse{}
}

// DeliverMessageFaucet() handles a 'faucet' message by minting tokens to the recipient (test-only).
// In addition to crediting the recipient's account, it persists a queryable Faucet state record so
// custom RPC endpoints can report faucet activity.
func (c *Contract) DeliverMessageFaucet(msg *MessageFaucet) *PluginDeliverResponse {
	log.Printf("DeliverMessageFaucet called: to=%x amount=%d", msg.RecipientAddress, msg.Amount)
	// calculate the recipient account key and the faucet record key
	recipientKey, faucetKey := KeyForAccount(msg.RecipientAddress), KeyForFaucet(msg.RecipientAddress)
	// generate query ids to correlate the batch read
	recipientQueryId, faucetQueryId := rand.Uint64(), rand.Uint64()
	// read the recipient account and any existing faucet record
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: recipientQueryId, Key: recipientKey},
			{QueryId: faucetQueryId, Key: faucetKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if response.Error != nil {
		return &PluginDeliverResponse{Error: response.Error}
	}
	// extract the raw bytes from the batch read results
	var recipientBytes, faucetBytes []byte
	for _, resp := range response.Results {
		if len(resp.Entries) == 0 {
			continue
		}
		switch resp.QueryId {
		case recipientQueryId:
			recipientBytes = resp.Entries[0].Value
		case faucetQueryId:
			faucetBytes = resp.Entries[0].Value
		}
	}
	// unmarshal the recipient account (new accounts start at 0)
	recipient := new(Account)
	if err = Unmarshal(recipientBytes, recipient); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// unmarshal the existing faucet record (defaults to empty)
	faucet := new(Faucet)
	if err = Unmarshal(faucetBytes, faucet); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// mint tokens to the recipient (created from nothing)
	recipient.Amount += msg.Amount
	// update the queryable faucet record
	faucet.RecipientAddress = msg.RecipientAddress
	faucet.TotalAmount += msg.Amount
	faucet.Count++
	// marshal the updated account and faucet record
	recipientBytes, err = Marshal(recipient)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	faucetBytes, err = Marshal(faucet)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// write both the account balance and the faucet record
	resp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: recipientKey, Value: recipientBytes},
			{Key: faucetKey, Value: faucetBytes},
		},
	})
	if err == nil {
		err = resp.Error
	}
	return &PluginDeliverResponse{Error: err}
}

// DeliverMessageReward() handles a 'reward' message by minting tokens to the recipient, with the
// admin paying the fee. It also persists a queryable Reward state record so custom RPC endpoints
// can report reward activity.
func (c *Contract) DeliverMessageReward(msg *MessageReward, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageReward called: admin=%x to=%x amount=%d fee=%d", msg.AdminAddress, msg.RecipientAddress, msg.Amount, fee)
	var (
		adminKey, recipientKey, feePoolKey, rewardKey             []byte
		adminBytes, recipientBytes, feePoolBytes, rewardBytes     []byte
		adminQueryId, recipientQueryId, feeQueryId, rewardQueryId = rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
		admin, recipient, feePool, reward                         = new(Account), new(Account), new(Pool), new(Reward)
	)
	// calculate all state keys
	adminKey, recipientKey = KeyForAccount(msg.AdminAddress), KeyForAccount(msg.RecipientAddress)
	feePoolKey, rewardKey = KeyForFeePool(c.Config.ChainId), KeyForReward(msg.RecipientAddress)
	// batch read fee pool, admin, recipient and any existing reward record
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQueryId, Key: feePoolKey},
			{QueryId: adminQueryId, Key: adminKey},
			{QueryId: recipientQueryId, Key: recipientKey},
			{QueryId: rewardQueryId, Key: rewardKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if response.Error != nil {
		return &PluginDeliverResponse{Error: response.Error}
	}
	// match each result to its variable using the query id
	for _, resp := range response.Results {
		if len(resp.Entries) == 0 {
			continue
		}
		switch resp.QueryId {
		case adminQueryId:
			adminBytes = resp.Entries[0].Value
		case recipientQueryId:
			recipientBytes = resp.Entries[0].Value
		case feeQueryId:
			feePoolBytes = resp.Entries[0].Value
		case rewardQueryId:
			rewardBytes = resp.Entries[0].Value
		}
	}
	// unmarshal all records
	if err = Unmarshal(adminBytes, admin); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(recipientBytes, recipient); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(rewardBytes, reward); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// the admin must be able to pay the fee
	if admin.Amount < fee {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	// apply state changes: admin pays the fee, recipient is minted tokens, fee pool collects the fee
	admin.Amount -= fee
	recipient.Amount += msg.Amount
	feePool.Amount += fee
	// update the queryable reward record
	reward.RecipientAddress = msg.RecipientAddress
	reward.LastAdminAddress = msg.AdminAddress
	reward.TotalAmount += msg.Amount
	reward.Count++
	// marshal the updated records
	if recipientBytes, err = Marshal(recipient); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if feePoolBytes, err = Marshal(feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if rewardBytes, err = Marshal(reward); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	// build the set operations common to both branches
	sets := []*PluginSetOp{
		{Key: feePoolKey, Value: feePoolBytes},
		{Key: recipientKey, Value: recipientBytes},
		{Key: rewardKey, Value: rewardBytes},
	}
	var resp *PluginStateWriteResponse
	// if the admin is drained, delete the account; otherwise persist the updated admin balance
	if admin.Amount == 0 {
		resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets:    sets,
			Deletes: []*PluginDeleteOp{{Key: adminKey}},
		})
	} else {
		if adminBytes, err = Marshal(admin); err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		resp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: append(sets, &PluginSetOp{Key: adminKey, Value: adminBytes}),
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
	// NOTE: the plugin shares Canopy's FSM keyspace, so these prefixes MUST NOT collide with core's
	// reserved prefixes (1-15, e.g. 3=validators, 4=committees). We use high, plugin-owned values.
	faucetPrefix  = []byte{100} // store key prefix for faucet records
	rewardPrefix  = []byte{101} // store key prefix for reward records
	paramsPrefix  = []byte{7} // store key prefix for governance parameters
)

// KeyForAccount() returns the state database key for an account
func KeyForAccount(addr []byte) []byte {
	return JoinLenPrefix(accountPrefix, addr)
}

// KeyForFaucet() returns the state database key for a recipient's faucet record
func KeyForFaucet(addr []byte) []byte {
	return JoinLenPrefix(faucetPrefix, addr)
}

// FaucetPrefix() returns the key prefix used to iterate over all faucet records
func FaucetPrefix() []byte {
	return JoinLenPrefix(faucetPrefix)
}

// KeyForReward() returns the state database key for a recipient's reward record
func KeyForReward(addr []byte) []byte {
	return JoinLenPrefix(rewardPrefix, addr)
}

// RewardPrefix() returns the key prefix used to iterate over all reward records
func RewardPrefix() []byte {
	return JoinLenPrefix(rewardPrefix)
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
