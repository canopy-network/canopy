package contract

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

/*
  Praxis — Prediction Market Plugin for Canopy
  ─────────────────────────────────────────────
  Transaction types:
    send              - standard token transfer (built-in)
    create_market     - open a new YES/NO prediction market
    submit_prediction - place a bet on a market outcome
    resolve_market    - resolver declares the winning outcome
    claim_winnings    - winner claims their payout

  State key prefixes:
    0x01  Account
    0x02  Pool (fee pool)
    0x07  FeeParams
    0x10  Market
    0x11  Prediction  (keyed by marketId + forecasterAddress)
    0x12  MarketCounter
    0x13  ForecasterRecord

  NOTE on currentHeight:
    plugin.go creates a brand new Contract instance for EVERY FSM message,
    so state on the Contract struct does NOT persist between BeginBlock and DeliverTx.
    We use package-level vars (heightMu + globalHeight) to safely share
    the block height across calls within the same block.
*/

// ── Package-level height (survives across Contract instances) ─────

var (
	heightMu     sync.RWMutex
	globalHeight uint64
)

func setGlobalHeight(h uint64) {
	heightMu.Lock()
	defer heightMu.Unlock()
	globalHeight = h
}

func getGlobalHeight() uint64 {
	heightMu.RLock()
	defer heightMu.RUnlock()
	return globalHeight
}

// ── Plugin config ─────────────────────────────────────────────────

var ContractConfig = &PluginConfig{
	Name:    "praxis_prediction_market",
	Id:      1,
	Version: 1,
	SupportedTransactions: []string{
		"send",
		"create_market",
		"submit_prediction",
		"resolve_market",
		"claim_winnings",
	},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageCreateMarket",
		"type.googleapis.com/types.MessageSubmitPrediction",
		"type.googleapis.com/types.MessageResolveMarket",
		"type.googleapis.com/types.MessageClaimWinnings",
	},
	EventTypeUrls: []string{
		"type.googleapis.com/types.Market",
	},
}

func init() {
	file_account_proto_init()
	file_event_proto_init()
	file_plugin_proto_init()
	file_tx_proto_init()

	var fds [][]byte
	for _, file := range []protoreflect.FileDescriptor{
		anypb.File_google_protobuf_any_proto,
		File_account_proto, File_event_proto, File_plugin_proto, File_tx_proto,
	} {
		fd, _ := proto.Marshal(protodesc.ToFileDescriptorProto(file))
		fds = append(fds, fd)
	}
	ContractConfig.FileDescriptorProtos = fds
}

// ── Contract struct ───────────────────────────────────────────────
// NOTE: plugin.go creates a new Contract per FSM message, so do NOT
// store block-scoped state on this struct. Use package-level vars instead.

type Contract struct {
	Config    Config
	FSMConfig *PluginFSMConfig
	plugin    *Plugin
	fsmId     uint64
}

// ── Lifecycle ─────────────────────────────────────────────────────

func (c *Contract) Genesis(_ *PluginGenesisRequest) *PluginGenesisResponse {
	return &PluginGenesisResponse{}
}

func (c *Contract) BeginBlock(req *PluginBeginRequest) *PluginBeginResponse {
	setGlobalHeight(req.Height)
	return &PluginBeginResponse{}
}

func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
	feeQId := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQId, Key: KeyForFeeParams()},
		},
	})
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginCheckResponse{Error: resp.Error}
	}
	minFees := new(FeeParams)
	var feeBytes []byte
	for _, r := range resp.Results {
		if r.QueryId == feeQId && len(r.Entries) > 0 {
			feeBytes = r.Entries[0].Value
		}
	}
	if err = Unmarshal(feeBytes, minFees); err != nil {
		return &PluginCheckResponse{Error: err}
	}
	if request.Tx.Fee < minFees.SendFee {
		return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
	}

	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	switch x := msg.(type) {
	case *MessageSend:
		return c.CheckMessageSend(x)
	case *MessageCreateMarket:
		return c.CheckCreateMarket(x)
	case *MessageSubmitPrediction:
		return c.CheckSubmitPrediction(x)
	case *MessageResolveMarket:
		return c.CheckResolveMarket(x)
	case *MessageClaimWinnings:
		return c.CheckClaimWinnings(x)
	default:
		return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
	}
}

func (c *Contract) DeliverTx(request *PluginDeliverRequest) *PluginDeliverResponse {
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	switch x := msg.(type) {
	case *MessageSend:
		return c.DeliverMessageSend(x, request.Tx.Fee)
	case *MessageCreateMarket:
		return c.DeliverCreateMarket(x, request.Tx.Fee)
	case *MessageSubmitPrediction:
		return c.DeliverSubmitPrediction(x, request.Tx.Fee)
	case *MessageResolveMarket:
		return c.DeliverResolveMarket(x, request.Tx.Fee)
	case *MessageClaimWinnings:
		return c.DeliverClaimWinnings(x, request.Tx.Fee)
	default:
		return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
	}
}

// EndBlock broadcasts all markets as events so the frontend can query them
// via /v1/query/events-by-height or /v1/query/events-by-address
func (c *Contract) EndBlock(req *PluginEndRequest) *PluginEndResponse {
	rangeQId := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rangeQId, Prefix: marketPrefix, Limit: 200, Reverse: false},
		},
	})
	if err != nil || resp.Error != nil {
		return &PluginEndResponse{}
	}
	var events []*Event
	for _, result := range resp.Results {
		if result.QueryId != rangeQId {
			continue
		}
		for _, entry := range result.Entries {
			market := new(Market)
			if e := Unmarshal(entry.Value, market); e != nil {
				continue
			}
			marketAny, aerr := anypb.New(market)
			if aerr != nil {
				continue
			}
			events = append(events, &Event{
				EventType: "market_state",
				Msg: &Event_Custom{Custom: &EventCustom{
					Msg: marketAny,
				}},
				Address: market.CreatorAddress,
				Height:  req.Height,
			})
		}
	}
	return &PluginEndResponse{Events: events}
}

// ── CheckTx handlers ──────────────────────────────────────────────

func (c *Contract) CheckMessageSend(msg *MessageSend) *PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if len(msg.ToAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{Recipient: msg.ToAddress, AuthorizedSigners: [][]byte{msg.FromAddress}}
}

func (c *Contract) CheckCreateMarket(msg *MessageCreateMarket) *PluginCheckResponse {
	if len(msg.CreatorAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if len(msg.ResolverAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if len(msg.Question) == 0 || len(msg.Question) > 280 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.StakeAmount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.CreatorAddress}}
}

func (c *Contract) CheckSubmitPrediction(msg *MessageSubmitPrediction) *PluginCheckResponse {
	if len(msg.ForecasterAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.MarketId == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.Outcome != 0 && msg.Outcome != 1 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.Amount == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.ForecasterAddress}}
}

func (c *Contract) CheckResolveMarket(msg *MessageResolveMarket) *PluginCheckResponse {
	if len(msg.ResolverAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.MarketId == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.WinningOutcome != 0 && msg.WinningOutcome != 1 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.ResolverAddress}}
}

func (c *Contract) CheckClaimWinnings(msg *MessageClaimWinnings) *PluginCheckResponse {
	if len(msg.ClaimerAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.MarketId == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.ClaimerAddress}}
}

// ── DeliverTx: send ───────────────────────────────────────────────

func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageSend: from=%x to=%x amount=%d fee=%d", msg.FromAddress, msg.ToAddress, msg.Amount, fee)
	var (
		fromKey, toKey, feePoolKey      = KeyForAccount(msg.FromAddress), KeyForAccount(msg.ToAddress), KeyForFeePool(c.Config.ChainId)
		fromQId, toQId, feeQId          = rand.Uint64(), rand.Uint64(), rand.Uint64()
		from, to, feePool               = new(Account), new(Account), new(Pool)
		fromBytes, toBytes, feePoolBytes []byte
	)
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQId, Key: feePoolKey},
			{QueryId: fromQId, Key: fromKey},
			{QueryId: toQId, Key: toKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if response.Error != nil {
		return &PluginDeliverResponse{Error: response.Error}
	}
	for _, r := range response.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case fromQId:
			fromBytes = r.Entries[0].Value
		case toQId:
			toBytes = r.Entries[0].Value
		case feeQId:
			feePoolBytes = r.Entries[0].Value
		}
	}
	if err = Unmarshal(fromBytes, from); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(toBytes, to); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	amountToDeduct := msg.Amount + fee
	if from.Amount < amountToDeduct {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	if bytes.Equal(fromKey, toKey) {
		to = from
	}
	from.Amount -= amountToDeduct
	feePool.Amount += fee
	to.Amount += msg.Amount
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
	var writeResp *PluginStateWriteResponse
	if from.Amount == 0 {
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets:    []*PluginSetOp{{Key: feePoolKey, Value: feePoolBytes}, {Key: toKey, Value: toBytes}},
			Deletes: []*PluginDeleteOp{{Key: fromKey}},
		})
	} else {
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: []*PluginSetOp{
				{Key: feePoolKey, Value: feePoolBytes},
				{Key: toKey, Value: toBytes},
				{Key: fromKey, Value: fromBytes},
			},
		})
	}
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}

// ── DeliverTx: create_market ──────────────────────────────────────

func (c *Contract) DeliverCreateMarket(msg *MessageCreateMarket, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverCreateMarket: creator=%x question=%q stakeAmount=%d", msg.CreatorAddress, msg.Question, msg.StakeAmount)

	creatorKey := KeyForAccount(msg.CreatorAddress)
	feePoolKey := KeyForFeePool(c.Config.ChainId)
	counterKey := KeyForMarketCounter()

	creatorQId, feeQId, counterQId := rand.Uint64(), rand.Uint64(), rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: creatorQId, Key: creatorKey},
			{QueryId: feeQId, Key: feePoolKey},
			{QueryId: counterQId, Key: counterKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	creator, feePool, counter := new(Account), new(Pool), new(MarketCounter)
	var creatorBytes, feePoolBytes, counterBytes []byte

	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case creatorQId:
			creatorBytes = r.Entries[0].Value
		case feeQId:
			feePoolBytes = r.Entries[0].Value
		case counterQId:
			counterBytes = r.Entries[0].Value
		}
	}

	if err = Unmarshal(creatorBytes, creator); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(counterBytes, counter); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	totalCost := msg.StakeAmount + fee
	if creator.Amount < totalCost {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}

	counter.Count++
	newMarket := &Market{
		Id:               counter.Count,
		CreatorAddress:   msg.CreatorAddress,
		Question:         msg.Question,
		Description:      msg.Description,
		ResolverAddress:  msg.ResolverAddress,
		ResolutionHeight: msg.ResolutionHeight,
		Status:           0,
		TotalYesPool:     0,
		TotalNoPool:      0,
		CreatedHeight:    getGlobalHeight(),
	}

	creator.Amount -= totalCost
	feePool.Amount += fee

	marketBytes, err := Marshal(newMarket)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	counterBytes, err = Marshal(counter)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePoolBytes, err = Marshal(feePool)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	sets := []*PluginSetOp{
		{Key: KeyForMarket(newMarket.Id), Value: marketBytes},
		{Key: counterKey, Value: counterBytes},
		{Key: feePoolKey, Value: feePoolBytes},
	}

	var writeResp *PluginStateWriteResponse
	if creator.Amount == 0 {
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets:    sets,
			Deletes: []*PluginDeleteOp{{Key: creatorKey}},
		})
	} else {
		creatorBytes, err = Marshal(creator)
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		sets = append(sets, &PluginSetOp{Key: creatorKey, Value: creatorBytes})
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
	}
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}

	// Emit event so the frontend can query this market
	marketAny, aerr := anypb.New(newMarket)
	if aerr != nil {
		log.Printf("Market #%d created (event emit failed): %v", newMarket.Id, aerr)
		return &PluginDeliverResponse{}
	}
	log.Printf("Market #%d created: %q at height=%d", newMarket.Id, newMarket.Question, newMarket.CreatedHeight)
	return &PluginDeliverResponse{
		Events: []*Event{{
			EventType: "market_created",
			Msg: &Event_Custom{Custom: &EventCustom{
				Msg: marketAny,
			}},
			Address: msg.CreatorAddress,
		}},
	}
}

// ── DeliverTx: submit_prediction ─────────────────────────────────

func (c *Contract) DeliverSubmitPrediction(msg *MessageSubmitPrediction, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverSubmitPrediction: forecaster=%x marketId=%d outcome=%d amount=%d", msg.ForecasterAddress, msg.MarketId, msg.Outcome, msg.Amount)

	forecasterKey       := KeyForAccount(msg.ForecasterAddress)
	feePoolKey          := KeyForFeePool(c.Config.ChainId)
	marketKey           := KeyForMarket(msg.MarketId)
	predictionKey       := KeyForPrediction(msg.MarketId, msg.ForecasterAddress)
	forecasterRecordKey := KeyForForecasterRecord(msg.ForecasterAddress)

	forecasterQId, feeQId, marketQId, predQId, recordQId :=
		rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: forecasterQId, Key: forecasterKey},
			{QueryId: feeQId, Key: feePoolKey},
			{QueryId: marketQId, Key: marketKey},
			{QueryId: predQId, Key: predictionKey},
			{QueryId: recordQId, Key: forecasterRecordKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	forecaster, feePool, market, prediction, record :=
		new(Account), new(Pool), new(Market), new(Prediction), new(ForecasterRecord)
	var forecasterBytes, feePoolBytes, marketBytes, predBytes, recordBytes []byte

	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case forecasterQId:
			forecasterBytes = r.Entries[0].Value
		case feeQId:
			feePoolBytes = r.Entries[0].Value
		case marketQId:
			marketBytes = r.Entries[0].Value
		case predQId:
			predBytes = r.Entries[0].Value
		case recordQId:
			recordBytes = r.Entries[0].Value
		}
	}

	if err = Unmarshal(forecasterBytes, forecaster); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(marketBytes, market); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(predBytes, prediction); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(recordBytes, record); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	if market.Id == 0 || market.Status != 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if prediction.MarketId != 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}

	totalCost := msg.Amount + fee
	if forecaster.Amount < totalCost {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}

	forecaster.Amount -= totalCost
	feePool.Amount += fee

	if msg.Outcome == 1 {
		market.TotalYesPool += msg.Amount
	} else {
		market.TotalNoPool += msg.Amount
	}

	newPrediction := &Prediction{
		ForecasterAddress: msg.ForecasterAddress,
		MarketId:          msg.MarketId,
		Outcome:           msg.Outcome,
		Amount:            msg.Amount,
		Claimed:           false,
	}

	record.TotalBets++

	marketBytes, err = Marshal(market)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	predBytes, err = Marshal(newPrediction)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePoolBytes, err = Marshal(feePool)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	recordBytes, err = Marshal(record)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	sets := []*PluginSetOp{
		{Key: marketKey, Value: marketBytes},
		{Key: predictionKey, Value: predBytes},
		{Key: feePoolKey, Value: feePoolBytes},
		{Key: forecasterRecordKey, Value: recordBytes},
	}

	var writeResp *PluginStateWriteResponse
	if forecaster.Amount == 0 {
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets:    sets,
			Deletes: []*PluginDeleteOp{{Key: forecasterKey}},
		})
	} else {
		forecasterBytes, err = Marshal(forecaster)
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		sets = append(sets, &PluginSetOp{Key: forecasterKey, Value: forecasterBytes})
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
	}
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	log.Printf("Prediction submitted: forecaster=%x market=%d outcome=%d amount=%d", msg.ForecasterAddress, msg.MarketId, msg.Outcome, msg.Amount)
	return &PluginDeliverResponse{}
}

// ── DeliverTx: resolve_market ─────────────────────────────────────

func (c *Contract) DeliverResolveMarket(msg *MessageResolveMarket, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverResolveMarket: resolver=%x marketId=%d winningOutcome=%d", msg.ResolverAddress, msg.MarketId, msg.WinningOutcome)

	marketKey   := KeyForMarket(msg.MarketId)
	resolverKey := KeyForAccount(msg.ResolverAddress)
	feePoolKey  := KeyForFeePool(c.Config.ChainId)

	marketQId, resolverQId, feeQId := rand.Uint64(), rand.Uint64(), rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: marketQId, Key: marketKey},
			{QueryId: resolverQId, Key: resolverKey},
			{QueryId: feeQId, Key: feePoolKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	market, resolver, feePool := new(Market), new(Account), new(Pool)
	var marketBytes, resolverBytes, feePoolBytes []byte

	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case marketQId:
			marketBytes = r.Entries[0].Value
		case resolverQId:
			resolverBytes = r.Entries[0].Value
		case feeQId:
			feePoolBytes = r.Entries[0].Value
		}
	}

	if err = Unmarshal(marketBytes, market); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(resolverBytes, resolver); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	if market.Id == 0 || market.Status != 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if !bytes.Equal(market.ResolverAddress, msg.ResolverAddress) {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if resolver.Amount < fee {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}

	market.Status = 2
	market.WinningOutcome = msg.WinningOutcome
	resolver.Amount -= fee
	feePool.Amount += fee

	marketBytes, err = Marshal(market)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePoolBytes, err = Marshal(feePool)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	sets := []*PluginSetOp{
		{Key: marketKey, Value: marketBytes},
		{Key: feePoolKey, Value: feePoolBytes},
	}

	var writeResp *PluginStateWriteResponse
	if resolver.Amount == 0 {
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets:    sets,
			Deletes: []*PluginDeleteOp{{Key: resolverKey}},
		})
	} else {
		resolverBytes, err = Marshal(resolver)
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		sets = append(sets, &PluginSetOp{Key: resolverKey, Value: resolverBytes})
		writeResp, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: sets})
	}
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	log.Printf("Market #%d resolved: winningOutcome=%d", msg.MarketId, msg.WinningOutcome)
	return &PluginDeliverResponse{}
}

// ── DeliverTx: claim_winnings ─────────────────────────────────────

func (c *Contract) DeliverClaimWinnings(msg *MessageClaimWinnings, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverClaimWinnings: claimer=%x marketId=%d", msg.ClaimerAddress, msg.MarketId)

	marketKey     := KeyForMarket(msg.MarketId)
	predictionKey := KeyForPrediction(msg.MarketId, msg.ClaimerAddress)
	claimerKey    := KeyForAccount(msg.ClaimerAddress)
	feePoolKey    := KeyForFeePool(c.Config.ChainId)
	recordKey     := KeyForForecasterRecord(msg.ClaimerAddress)

	marketQId, predQId, claimerQId, feeQId, recordQId :=
		rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: marketQId, Key: marketKey},
			{QueryId: predQId, Key: predictionKey},
			{QueryId: claimerQId, Key: claimerKey},
			{QueryId: feeQId, Key: feePoolKey},
			{QueryId: recordQId, Key: recordKey},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	market, prediction, claimer, feePool, record :=
		new(Market), new(Prediction), new(Account), new(Pool), new(ForecasterRecord)
	var marketBytes, predBytes, claimerBytes, feePoolBytes, recordBytes []byte

	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case marketQId:
			marketBytes = r.Entries[0].Value
		case predQId:
			predBytes = r.Entries[0].Value
		case claimerQId:
			claimerBytes = r.Entries[0].Value
		case feeQId:
			feePoolBytes = r.Entries[0].Value
		case recordQId:
			recordBytes = r.Entries[0].Value
		}
	}

	if err = Unmarshal(marketBytes, market); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(predBytes, prediction); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(claimerBytes, claimer); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(recordBytes, record); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	if market.Id == 0 || market.Status != 2 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if prediction.MarketId == 0 {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if prediction.Claimed {
		return &PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if prediction.Outcome != market.WinningOutcome {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}

	// Payout = stake + proportional share of losing pool
	var winnerPool, loserPool uint64
	if market.WinningOutcome == 1 {
		winnerPool = market.TotalYesPool
		loserPool  = market.TotalNoPool
	} else {
		winnerPool = market.TotalNoPool
		loserPool  = market.TotalYesPool
	}
	var payout uint64
	if winnerPool > 0 {
		payout = prediction.Amount + (prediction.Amount*loserPool)/winnerPool
	} else {
		payout = prediction.Amount
	}
	totalPool := winnerPool + loserPool
	if payout > totalPool {
		payout = totalPool
	}
	if claimer.Amount+payout < fee {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}

	claimer.Amount += payout
	claimer.Amount -= fee
	feePool.Amount += fee
	prediction.Claimed = true
	record.CorrectBets++
	record.TotalEarned += payout

	claimerBytes, err = Marshal(claimer)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	predBytes, err = Marshal(prediction)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePoolBytes, err = Marshal(feePool)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	recordBytes, err = Marshal(record)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: claimerKey, Value: claimerBytes},
			{Key: predictionKey, Value: predBytes},
			{Key: feePoolKey, Value: feePoolBytes},
			{Key: recordKey, Value: recordBytes},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	log.Printf("Winnings claimed: claimer=%x market=%d payout=%d", msg.ClaimerAddress, msg.MarketId, payout)
	return &PluginDeliverResponse{}
}

// ── State key prefixes ────────────────────────────────────────────

var (
	accountPrefix       = []byte{0x01}
	poolPrefix          = []byte{0x02}
	paramsPrefix        = []byte{0x07}
	marketPrefix        = []byte{0x10}
	predictionPrefix    = []byte{0x11}
	marketCounterPrefix = []byte{0x12}
	forecasterRecPrefix = []byte{0x13}
)

func KeyForAccount(addr []byte) []byte {
	return JoinLenPrefix(accountPrefix, addr)
}

func KeyForFeeParams() []byte {
	return JoinLenPrefix(paramsPrefix, []byte("/f/"))
}

func KeyForFeePool(chainId uint64) []byte {
	return JoinLenPrefix(poolPrefix, formatUint64(chainId))
}

func KeyForMarket(id uint64) []byte {
	return JoinLenPrefix(marketPrefix, formatUint64(id))
}

func KeyForPrediction(marketId uint64, forecaster []byte) []byte {
	idBytes := formatUint64(marketId)
	combined := append(idBytes, forecaster...)
	return JoinLenPrefix(predictionPrefix, combined)
}

func KeyForMarketCounter() []byte {
	return JoinLenPrefix(marketCounterPrefix, []byte("/mc/"))
}

func KeyForForecasterRecord(addr []byte) []byte {
	return JoinLenPrefix(forecasterRecPrefix, addr)
}

func formatUint64(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}
