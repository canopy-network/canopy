package contract

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"math/rand"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file contains the base contract implementation that overrides the basic 'transfer' functionality */

// PluginConfig: the configuration of the contract
var ContractConfig = &PluginConfig{
	Name:    "arbor",
	Id:      1,
	Version: 1,
	SupportedTransactions: []string{
		"send",
		"create_market",
		"update_market_params",
		"pause_market",
		"resume_market",
		"deprecate_market",
		"update_price",
		"deposit_collateral",
		"withdraw_collateral",
		"deposit",
		"withdraw",
		"borrow",
		"repay",
		"liquidate_position",
		"set_asset_tier",
		"set_treasury_cut",
		"mint_nusd",
		"burn_nusd",
		"liquidate_nasm_vault",
		"set_emergency_mode",
		"set_circuit_breaker",
		"claim_faucet",
	},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageCreateMarket",
		"type.googleapis.com/types.MessageUpdateMarketParams",
		"type.googleapis.com/types.MessagePauseMarket",
		"type.googleapis.com/types.MessageResumeMarket",
		"type.googleapis.com/types.MessageDeprecateMarket",
		"type.googleapis.com/types.MessageUpdatePrice",
		"type.googleapis.com/types.MessageDepositCollateral",
		"type.googleapis.com/types.MessageWithdrawCollateral",
		"type.googleapis.com/types.MessageDeposit",
		"type.googleapis.com/types.MessageWithdraw",
		"type.googleapis.com/types.MessageBorrow",
		"type.googleapis.com/types.MessageRepay",
		"type.googleapis.com/types.MessageLiquidatePosition",
		"type.googleapis.com/types.MessageSetAssetTier",
		"type.googleapis.com/types.MessageSetTreasuryCut",
		"type.googleapis.com/types.MessageMintNusd",
		"type.googleapis.com/types.MessageBurnNusd",
		"type.googleapis.com/types.MessageLiquidateNasmVault",
		"type.googleapis.com/types.MessageSetEmergencyMode",
		"type.googleapis.com/types.MessageSetCircuitBreaker",
		"type.googleapis.com/types.MessageClaimFaucet",
	},
	EventTypeUrls: []string{
		"type.googleapis.com/types.EventIndexEncodingOverflowHalted",
		"type.googleapis.com/types.EventInsolventMarketValueRecovered",
		"type.googleapis.com/types.EventTotalSuppliedDustClamp",
		"type.googleapis.com/types.EventTotalSharesOutstandingDustClamp",
		"type.googleapis.com/types.EventTotalBorrowedDustClamp",
		"type.googleapis.com/types.EventLayer4PendingCountWarning",
		"type.googleapis.com/types.EventLayer4PendingBadDebtTotalSaturated",
		"type.googleapis.com/types.EventLayer4PendingCountUnderflow",
		"type.googleapis.com/types.EventDepositWithdrawBlockedDuringPendingLoss",
		"type.googleapis.com/types.EventLossFactorExhausted",
		"type.googleapis.com/types.EventBadDebtSocialization",
		"type.googleapis.com/types.EventLossFactorAppliedToAlreadyInsolventMarket",
		"type.googleapis.com/types.EventReserveFundEncodingMigrationCompleted",
		"type.googleapis.com/types.EventNasmVaultLiquidated",
		"type.googleapis.com/types.FaucetClaimEvent",
	},
	// CustomStatePrefixes registers Arbor's reserved state-key range {16}-{28}
	// with Canopy at handshake. Canopy panics if any of these collide with the
	// core-reserved {1}-{15} range. See ARCM v3.11.1 Section 19.1.
	CustomStatePrefixes: [][]byte{
		PrefixMarkets,
		PrefixBorrowerPositions,
		PrefixReserveFund,
		PrefixPriceCache,
		PrefixCircuitBreaker,
		PrefixEmergencyMode,
		PrefixGovernanceParams,
		PrefixBackstopQueue,
		PrefixLenderPositions,
		PrefixBorrowIndex,
		PrefixSupplyIndex,
		PrefixLossFactor,
		PrefixLossFactorQueue,
	},
}

// init sets FileDescriptorProtos after ensuring .pb.go files are initialized
func init() {
	// Explicitly initialize the proto files first to ensure File_*_proto are set
	file_account_proto_init()
	file_event_proto_init()
	file_plugin_proto_init()
	file_tx_proto_init()
	file_arbor_proto_init()
	file_arbor_events_proto_init()
	file_arbor_state_proto_init()

	var fds [][]byte
	// Include google/protobuf/any.proto first as it's a dependency of event.proto and tx.proto
	for _, file := range []protoreflect.FileDescriptor{
		anypb.File_google_protobuf_any_proto,
		File_account_proto, File_event_proto, File_plugin_proto, File_tx_proto,
		File_arbor_proto, File_arbor_events_proto, File_arbor_state_proto,
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
//
// Wires AYIS Section 12.3's Accrual Ordering Contract into the chain's actual
// BeginBlock hook: "AccrueInterest() MUST be the first call in BeginBlock,
// before... loss-factor-application queue processing." Loss-factor-queue
// processing does not exist yet (Layer 4 is unbuilt), so this is currently
// the ONLY BeginBlock step -- but the ordering requirement is stated here
// now so a future addition (queue draining, circuit breaker eval) is added
// AFTER this call, not before it.
//
// Per-market isolation (Principle 2): one market's AccrueInterest failure is
// logged and skipped, not propagated as a block-level error. Only a failure
// to even ENUMERATE the market list (the range-read itself) aborts the
// block, since at that point no per-market isolation is possible.
func (c *Contract) BeginBlock(request *PluginBeginRequest) *PluginBeginResponse {
	// A fresh *Contract is constructed per inbound message (see Plugin.ListenForInbound),
	// so height is recorded on the long-lived *Plugin, not this short-lived Contract,
	// to survive across the BeginBlock -> DeliverTx sequence within the same block.
	c.plugin.SetCurrentHeight(request.Height)

	const qMarketsRange = 0
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: qMarketsRange, Prefix: JoinLenPrefix(PrefixMarkets)},
		},
	})
	if err != nil {
		log.Printf("BeginBlock: markets range read failed: %v", err)
		return &PluginBeginResponse{Error: err}
	}
	if readResp.Error != nil {
		log.Printf("BeginBlock: markets range read FSM error: %v", readResp.Error)
		return &PluginBeginResponse{Error: readResp.Error}
	}

	var marketEntries []*PluginStateEntry
	for _, result := range readResp.Results {
		if result.QueryId == qMarketsRange {
			marketEntries = result.Entries
			break
		}
	}

	var events []*Event

	for _, entry := range marketEntries {
		market := &Market{}
		if uErr := Unmarshal(entry.Value, market); uErr != nil {
			log.Printf("BeginBlock: failed to decode market at key %x: %v", entry.Key, uErr)
			continue
		}
		aiEvent, aErr := AccrueInterest(c, market.MarketId)
		if aErr != nil {
			log.Printf("BeginBlock: AccrueInterest failed for market %s: %v", market.MarketId, aErr)
			continue
		}
		if aiEvent != nil {
			events = append(events, aiEvent)
		}
		// Accrual Ordering Contract (AYIS Section 12.3): loss-factor-application
		// queue processing runs after AccrueInterest, same market, same block.
		lfEvent, lfErr := ProcessLossFactorQueue(c, market.MarketId)
		if lfErr != nil {
			log.Printf("BeginBlock: ProcessLossFactorQueue failed for market %s: %v", market.MarketId, lfErr)
			continue
		}
		if lfEvent != nil {
			events = append(events, lfEvent)
		}
	}

	// NASM Spec Section 6.5's recommended BeginBlock order: NASM stability
	// fee accrual runs once, globally, AFTER the per-market AYIS accrual
	// loop above completes (step 2, following step 1's AYIS.AccrueInterest
	// for all ARCM lending markets). A state-layer failure here is logged,
	// not propagated as a block-halting error -- matches this loop's own
	// established tolerance for per-market AccrueInterest/
	// ProcessLossFactorQueue failures (logged and continue, not aborted),
	// since a single accrual failure should not stop block production.
	if sfErr := AccrueStabilityFee(c); sfErr != nil {
		log.Printf("BeginBlock: AccrueStabilityFee failed: %v", sfErr)
	}

	return &PluginBeginResponse{Events: events}
}

// CheckTx() is code that is executed to statelessly validate a transaction
func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
	// get the message first -- fee validation below needs to know the
	// message type, since MessageClaimFaucet is fee-exempt (testnet-only,
	// see its doc comment in arbor.proto)
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	// [FAUCET FEE EXEMPTION, testnet-only] skip the SendFee floor for
	// MessageClaimFaucet -- see arbor.proto's MessageClaimFaucet comment
	if _, isFaucetClaim := msg.(*MessageClaimFaucet); !isFaucetClaim {
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
	}
	// handle the message
	switch x := msg.(type) {
	case *MessageSend:
		return c.CheckMessageSend(x)
	case *MessageCreateMarket:
		return c.CheckMessageCreateMarket(x)
	case *MessageDeposit:
		return c.CheckMessageDeposit(x)
	case *MessageWithdraw:
		return c.CheckMessageWithdraw(x)
	case *MessageUpdatePrice:
		return c.CheckMessageUpdatePrice(x)
	case *MessageDepositCollateral:
		return c.CheckMessageDepositCollateral(x)
	case *MessageSetAssetTier:
		return c.CheckMessageSetAssetTier(x)
	case *MessageBorrow:
		return c.CheckMessageBorrow(x)
	case *MessageRepay:
		return c.CheckMessageRepay(x)
	case *MessageWithdrawCollateral:
		return c.CheckMessageWithdrawCollateral(x)
	case *MessageLiquidatePosition:
		return c.CheckMessageLiquidatePosition(x)
	case *MessagePauseMarket:
		return c.CheckMessagePauseMarket(x)
	case *MessageResumeMarket:
		return c.CheckMessageResumeMarket(x)
	case *MessageDeprecateMarket:
		return c.CheckMessageDeprecateMarket(x)
	case *MessageUpdateMarketParams:
		return c.CheckMessageUpdateMarketParams(x)
	case *MessageSetTreasuryCut:
		return c.CheckMessageSetTreasuryCut(x)
	case *MessageMintNusd:
		return c.CheckMessageMintNusd(x)
	case *MessageBurnNusd:
		return c.CheckMessageBurnNusd(x)
	case *MessageLiquidateNasmVault:
		return c.CheckMessageLiquidateNasmVault(x)
	case *MessageSetEmergencyMode:
		return c.CheckMessageSetEmergencyMode(x)
	case *MessageSetCircuitBreaker:
		return c.CheckMessageSetCircuitBreaker(x)
	case *MessageClaimFaucet:
		return c.CheckMessageClaimFaucet(x)
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
	// [FIX, session finding] This switch previously had only the
	// MessageSend case -- every Arbor-specific message type fell to
	// default and was rejected with ErrInvalidMessageCast(). Root cause:
	// commit ae03baff ("Merge branch 'main' into main") merged in
	// upstream Canopy's generic send-only plugin template over this
	// file, silently discarding the Arbor-specific routing (and
	// ContractConfig's SupportedTransactions/TransactionTypeUrls
	// registration, restored below) with no merge conflict. CheckTx's
	// own switch (above) was unaffected and continued routing all
	// Arbor-specific types correctly (14 at the time of this fix, 18 as
	// of this session's own re-verification -- MintNusd/BurnNusd/
	// LiquidateNasmVault/SetTreasuryCut were added after this comment
	// was originally written; count corrected here rather than left
	// stale), which is why transactions were admitted to the
	// mempool -- but DeliverTx rejected every one of them at the point
	// business logic would actually run. Restored to mirror CheckTx's
	// case list and order exactly, using the DeliverMessage* handlers
	// that already exist and are already used by every custody-atomicity
	// fix this session made.
	switch x := msg.(type) {
	case *MessageSend:
		return c.DeliverMessageSend(x, request.Tx.Fee, request.Tx.Memo)
	case *MessageCreateMarket:
		return c.DeliverMessageCreateMarket(x, request.Tx.Fee)
	case *MessageDeposit:
		return c.DeliverMessageDeposit(x, request.Tx.Fee)
	case *MessageWithdraw:
		return c.DeliverMessageWithdraw(x, request.Tx.Fee)
	case *MessageUpdatePrice:
		return c.DeliverMessageUpdatePrice(x, request.Tx.Fee)
	case *MessageDepositCollateral:
		return c.DeliverMessageDepositCollateral(x, request.Tx.Fee)
	case *MessageSetAssetTier:
		return c.DeliverMessageSetAssetTier(x, request.Tx.Fee)
	case *MessageBorrow:
		return c.DeliverMessageBorrow(x, request.Tx.Fee)
	case *MessageRepay:
		return c.DeliverMessageRepay(x, request.Tx.Fee)
	case *MessageWithdrawCollateral:
		return c.DeliverMessageWithdrawCollateral(x, request.Tx.Fee)
	case *MessageLiquidatePosition:
		return c.DeliverMessageLiquidatePosition(x, request.Tx.Fee)
	case *MessagePauseMarket:
		return c.DeliverMessagePauseMarket(x, request.Tx.Fee)
	case *MessageResumeMarket:
		return c.DeliverMessageResumeMarket(x, request.Tx.Fee)
	case *MessageDeprecateMarket:
		return c.DeliverMessageDeprecateMarket(x, request.Tx.Fee)
	case *MessageUpdateMarketParams:
		return c.DeliverMessageUpdateMarketParams(x, request.Tx.Fee)
	case *MessageSetTreasuryCut:
		return c.DeliverMessageSetTreasuryCut(x, request.Tx.Fee)
	case *MessageMintNusd:
		return c.DeliverMessageMintNusd(x, request.Tx.Fee)
	case *MessageBurnNusd:
		return c.DeliverMessageBurnNusd(x, request.Tx.Fee)
	case *MessageLiquidateNasmVault:
		return c.DeliverMessageLiquidateNasmVault(x, request.Tx.Fee)
	case *MessageSetEmergencyMode:
		return c.DeliverMessageSetEmergencyMode(x, request.Tx.Fee)
	case *MessageSetCircuitBreaker:
		return c.DeliverMessageSetCircuitBreaker(x, request.Tx.Fee)
	case *MessageClaimFaucet:
		return c.DeliverMessageClaimFaucet(x, request.Tx.Fee)
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
func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64, memo string) *PluginDeliverResponse {
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
	if msg.Amount > math.MaxUint64-fee {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	amountToDeduct := msg.Amount + fee
	log.Printf("from.Amount=%d to.Amount=%d feePool.Amount=%d", from.Amount, to.Amount, feePool.Amount)
	// if the account amount is less than the amount to subtract; return insufficient funds
	if from.Amount < amountToDeduct {
		log.Printf("ERROR: Insufficient funds: from.Amount=%d amountToDeduct=%d", from.Amount, amountToDeduct)
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	// for self-transfer, use same account data
	isSelfTransfer := bytes.Equal(fromKey, toKey)
	if isSelfTransfer {
		to = from
	}
	if feePool.Amount > math.MaxUint64-fee || (!isSelfTransfer && to.Amount > math.MaxUint64-msg.Amount) {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
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
	// Retain drained accounts only when they carry nonce state or core will advance the nonce after RLP.V2 delivery.
	request := &PluginStateWriteRequest{Sets: []*PluginSetOp{
		{Key: feePoolKey, Value: feePoolBytes},
		{Key: toKey, Value: toBytes},
	}}
	if from.Amount == 0 && from.Nonce == 0 && memo != "RLP.V2" {
		request.Deletes = []*PluginDeleteOp{{Key: fromKey}}
	} else {
		request.Sets = append(request.Sets, &PluginSetOp{Key: fromKey, Value: fromBytes})
	}
	resp, err := c.plugin.StateWrite(c, request)
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
