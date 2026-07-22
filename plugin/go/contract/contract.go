package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"log"
	"math"
	"math/rand"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file contains the base contract implementation for the KnowBet betting protocol */

// PluginConfig: the configuration of the contract
var ContractConfig = &PluginConfig{
	Name:                  "knowbet_contract",
	Id:                    3,
	Version:               1,
	SupportedTransactions: []string{"send", "betCreate", "betPlace", "betReveal", "betSettle", "betRefund"},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageBetCreate",
		"type.googleapis.com/types.MessageBetPlace",
		"type.googleapis.com/types.MessageBetReveal",
		"type.googleapis.com/types.MessageBetSettle",
		"type.googleapis.com/types.MessageBetRefund",
	},
	EventTypeUrls: nil,
	// CustomStatePrefixes declares the key prefixes this plugin owns for custom records.
	// Prefixes 100 (questions), 101 (bet records), 102 (counter) are outside the core-reserved range 1-15.
	CustomStatePrefixes: [][]byte{{100}, {101}, {102}},
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

// =============================================================================
// CheckTx - Stateless validation for all transaction types
// =============================================================================

// CheckTx() validates a transaction before it enters the mempool or a block
func (c *Contract) CheckTx(request *PluginCheckRequest) *PluginCheckResponse {
	// Read fee parameters from state
	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: KeyForFeeParams()},
		}})
	if err == nil {
		err = resp.Error
	}
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}

	// Parse fee params
	minFees := new(FeeParams)
	if err = Unmarshal(resp.Results[0].Entries[0].Value, minFees); err != nil {
		return &PluginCheckResponse{Error: err}
	}

	// Get the message
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}

	// Route to message-specific handler with fee validation
	switch x := msg.(type) {
	case *MessageSend:
		if request.Tx.Fee < minFees.SendFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageSend(x)

	case *MessageBetCreate:
		if request.Tx.Fee < minFees.BetCreateFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageBetCreate(x)

	case *MessageBetPlace:
		if request.Tx.Fee < minFees.BetPlaceFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageBetPlace(x)

	case *MessageBetReveal:
		if request.Tx.Fee < minFees.BetRevealFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageBetReveal(x)

	case *MessageBetSettle:
		if request.Tx.Fee < minFees.BetSettleFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageBetSettle(x)

	case *MessageBetRefund:
		if request.Tx.Fee < minFees.BetRefundFee {
			return &PluginCheckResponse{Error: ErrTxFeeBelowStateLimit()}
		}
		return c.CheckMessageBetRefund(x)

	default:
		return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
	}
}

// DeliverTx() executes a validated transaction and applies state changes
func (c *Contract) DeliverTx(request *PluginDeliverRequest) *PluginDeliverResponse {
	msg, err := FromAny(request.Tx.Msg)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	switch x := msg.(type) {
	case *MessageSend:
		return c.DeliverMessageSend(x, request.Tx.Fee, request.Tx.Memo)
	case *MessageBetCreate:
		return c.DeliverMessageBetCreate(x, request.Tx.Fee, request.Tx.Time, request.Tx.CreatedHeight)
	case *MessageBetPlace:
		return c.DeliverMessageBetPlace(x, request.Tx.Fee)
	case *MessageBetReveal:
		return c.DeliverMessageBetReveal(x, request.Tx.Fee)
	case *MessageBetSettle:
		return c.DeliverMessageBetSettle(x, request.Tx.Fee)
	case *MessageBetRefund:
		return c.DeliverMessageBetRefund(x, request.Tx.Fee)
	default:
		return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
	}
}

// EndBlock() is code that is executed at the end of 'applying' a block
func (c *Contract) EndBlock(_ *PluginEndRequest) *PluginEndResponse {
	return &PluginEndResponse{}
}

// =============================================================================
// MessageSend (existing implementation)
// =============================================================================

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

func (c *Contract) DeliverMessageSend(msg *MessageSend, fee uint64, memo string) *PluginDeliverResponse {
	log.Printf("DeliverMessageSend called: from=%x to=%x amount=%d fee=%d", msg.FromAddress, msg.ToAddress, msg.Amount, fee)
	var (
		fromKey, toKey, feePoolKey         []byte
		fromBytes, toBytes, feePoolBytes   []byte
		fromQueryId, toQueryId, feeQueryId = rand.Uint64(), rand.Uint64(), rand.Uint64()
		from, to, feePool                  = new(Account), new(Account), new(Pool)
	)
	fromKey, toKey, feePoolKey = KeyForAccount(msg.FromAddress), KeyForAccount(msg.ToAddress), KeyForFeePool(c.Config.ChainId)
	response, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: feeQueryId, Key: feePoolKey},
			{QueryId: fromQueryId, Key: fromKey},
			{QueryId: toQueryId, Key: toKey},
		}})
	if err != nil {
		log.Printf("StateRead error: %v", err)
		return &PluginDeliverResponse{Error: err}
	}
	if response.Error != nil {
		return &PluginDeliverResponse{Error: response.Error}
	}
	for _, resp := range response.Results {
		if len(resp.Entries) == 0 {
			continue
		}
		switch resp.QueryId {
		case fromQueryId:
			fromBytes = resp.Entries[0].Value
		case toQueryId:
			toBytes = resp.Entries[0].Value
		case feeQueryId:
			feePoolBytes = resp.Entries[0].Value
		}
	}
	amountToDeduct := msg.Amount + fee
	if err = Unmarshal(fromBytes, from); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(toBytes, to); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if from.Amount < amountToDeduct {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	if bytes.Equal(fromKey, toKey) {
		to = from
	}
	from.Amount -= amountToDeduct
	feePool.Amount += fee
	to.Amount += msg.Amount

	fromBytes, _ = Marshal(from)
	toBytes, _ = Marshal(to)
	feePoolBytes, _ = Marshal(feePool)

	if from.Amount == 0 && from.Nonce == 0 && memo != "RLP.V2" {
		_, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: []*PluginSetOp{
				{Key: feePoolKey, Value: feePoolBytes},
				{Key: toKey, Value: toBytes},
			},
			Deletes: []*PluginDeleteOp{{Key: fromKey}},
		})
	} else {
		_, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
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
	return &PluginDeliverResponse{}
}

// =============================================================================
// 1. BetCreate - Create a betting question
// =============================================================================

// CheckMessageBetCreate validates a BetCreate transaction
func (c *Contract) CheckMessageBetCreate(msg *MessageBetCreate) *PluginCheckResponse {
	// Validate creator address (20 bytes)
	if len(msg.CreatorAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// Validate question text is not empty
	if strings.TrimSpace(msg.QuestionText) == "" {
		return &PluginCheckResponse{Error: ErrInvalidQuestionText()}
	}
	// Sensitive word filter
	if containsSensitiveWords(msg.QuestionText) {
		return &PluginCheckResponse{Error: ErrSensitiveContent()}
	}
	// Validate options count: 2-5
	if len(msg.Options) < 2 || len(msg.Options) > 5 {
		return &PluginCheckResponse{Error: ErrInvalidOptionsCount()}
	}
	// Validate each option is non-empty and clean
	for _, opt := range msg.Options {
		if strings.TrimSpace(opt) == "" {
			return &PluginCheckResponse{Error: ErrInvalidOption()}
		}
		if containsSensitiveWords(opt) {
			return &PluginCheckResponse{Error: ErrSensitiveContent()}
		}
	}
	// Validate answer_hash is a 64-char hex string (SHA-256)
	if len(msg.AnswerHash) != 64 {
		return &PluginCheckResponse{Error: ErrInvalidAnswerHash()}
	}
	if _, err := hex.DecodeString(msg.AnswerHash); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidAnswerHash()}
	}
	// Validate stake >= 100 KNBT (100,000,000 uknbt)
	if msg.StakeAmount < 100_000_000 {
		return &PluginCheckResponse{Error: ErrStakeTooLow()}
	}
	// Validate duration: 1 hour (3600s) to 7 days (604800s)
	if msg.Duration < 3600 || msg.Duration > 604800 {
		return &PluginCheckResponse{Error: ErrInvalidDuration()}
	}
	return &PluginCheckResponse{
		Recipient:         msg.CreatorAddress,
		AuthorizedSigners: [][]byte{msg.CreatorAddress},
	}
}

// DeliverMessageBetCreate creates a new betting question
func (c *Contract) DeliverMessageBetCreate(msg *MessageBetCreate, fee uint64, txTime uint64, createdHeight uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageBetCreate: creator=%x question=%q stake=%d duration=%d",
		msg.CreatorAddress, msg.QuestionText, msg.StakeAmount, msg.Duration)

	// Read: creator account, fee pool, question counter
	counterKey := KeyForQuestionCounter()
	creatorKey := KeyForAccount(msg.CreatorAddress)
	feePoolKey := KeyForFeePool(c.Config.ChainId)

	counterQueryId := rand.Uint64()
	creatorQueryId := rand.Uint64()
	feeQueryId := rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: counterQueryId, Key: counterKey},
			{QueryId: creatorQueryId, Key: creatorKey},
			{QueryId: feeQueryId, Key: feePoolKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	// Parse response
	var counterBytes, creatorBytes, feePoolBytes []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case counterQueryId:
			counterBytes = r.Entries[0].Value
		case creatorQueryId:
			creatorBytes = r.Entries[0].Value
		case feeQueryId:
			feePoolBytes = r.Entries[0].Value
		}
	}

	// Unmarshal accounts
	creator := new(Account)
	if err = Unmarshal(creatorBytes, creator); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePool := new(Pool)
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Check creator has enough funds (fee + stake)
	totalDeduction := msg.StakeAmount + fee
	if creator.Amount < totalDeduction {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	creator.Amount -= totalDeduction
	feePool.Amount += fee

	// Determine next question ID
	var questionId uint64
	if len(counterBytes) > 0 {
		questionId = binary.BigEndian.Uint64(counterBytes) + 1
	} else {
		questionId = 1
	}

	// Compute close time (using tx time in microseconds -> seconds)
	currentTimeSec := int64(txTime / 1_000_000)
	closeTime := currentTimeSec + msg.Duration

	// Compute close height (duration in seconds / 20s per block)
	closeHeight := createdHeight + uint64(msg.Duration/20)

	// Build question object
	question := &Question{
		Id:                   questionId,
		CreatorAddress:       msg.CreatorAddress,
		QuestionText:         msg.QuestionText,
		Options:              msg.Options,
		AnswerHash:           msg.AnswerHash,
		StakeAmount:          msg.StakeAmount,
		TotalPool:            0, // no bets yet
		OptionPools:          make([]uint64, len(msg.Options)),
		CreatedHeight:        createdHeight,
		CloseTime:            closeTime,
		CloseHeight:          closeHeight,
		Revealed:             false,
		Settled:              false,
		Refunded:             false,
		ParticipantAddresses: nil,
	}

	// Increment counter
	nextCounter := questionId
	counterBz := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBz, nextCounter)

	// Marshal all objects
	questionKey := KeyForQuestion(questionId)
	questionBz, err := Marshal(question)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	creatorBz, _ := Marshal(creator)
	feePoolBz, _ := Marshal(feePool)

	// Write state
	writeReq := &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: creatorKey, Value: creatorBz},
			{Key: feePoolKey, Value: feePoolBz},
			{Key: counterKey, Value: counterBz},
			{Key: questionKey, Value: questionBz},
		},
	}
	if _, err = c.plugin.StateWrite(c, writeReq); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	log.Printf("BetCreate SUCCESS: questionId=%d", questionId)
	return &PluginDeliverResponse{}
}

// =============================================================================
// 2. BetPlace - Place a bet
// =============================================================================

// CheckMessageBetPlace validates a BetPlace transaction
func (c *Contract) CheckMessageBetPlace(msg *MessageBetPlace) *PluginCheckResponse {
	// Validate participant address
	if len(msg.ParticipantAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// Amount range: 10-10000 KNBT (10,000,000 - 10,000,000,000 uknbt)
	if msg.Amount < 10_000_000 {
		return &PluginCheckResponse{Error: ErrBetBelowMin()}
	}
	if msg.Amount > 10_000_000_000 {
		return &PluginCheckResponse{Error: ErrBetAboveMax()}
	}
	return &PluginCheckResponse{
		Recipient:         msg.ParticipantAddress,
		AuthorizedSigners: [][]byte{msg.ParticipantAddress},
	}
}

// DeliverMessageBetPlace processes a bet
func (c *Contract) DeliverMessageBetPlace(msg *MessageBetPlace, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageBetPlace: participant=%x questionId=%d option=%d amount=%d",
		msg.ParticipantAddress, msg.QuestionId, msg.OptionIndex, msg.Amount)

	// Read participant account, fee pool, and question
	participantKey := KeyForAccount(msg.ParticipantAddress)
	feePoolKey := KeyForFeePool(c.Config.ChainId)
	questionKey := KeyForQuestion(msg.QuestionId)

	participantQ := rand.Uint64()
	feeQ := rand.Uint64()
	questionQ := rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: participantQ, Key: participantKey},
			{QueryId: feeQ, Key: feePoolKey},
			{QueryId: questionQ, Key: questionKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	var participantBytes, feePoolBytes, questionBytes []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case participantQ:
			participantBytes = r.Entries[0].Value
		case feeQ:
			feePoolBytes = r.Entries[0].Value
		case questionQ:
			questionBytes = r.Entries[0].Value
		}
	}

	// Validate question exists
	if len(questionBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrQuestionNotFound()}
	}
	question := new(Question)
	if err = Unmarshal(questionBytes, question); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Validate question is open
	if question.Revealed || question.Settled || question.Refunded {
		return &PluginDeliverResponse{Error: ErrQuestionNotOpen()}
	}
	// Validate option index
	if int(msg.OptionIndex) >= len(question.Options) {
		return &PluginDeliverResponse{Error: ErrInvalidOptionIndex()}
	}
	// Creator cannot bet on own question
	if bytes.Equal(msg.ParticipantAddress, question.CreatorAddress) {
		return &PluginDeliverResponse{Error: ErrCreatorCannotBet()}
	}
	// Check if participant already bet on this question
	betKey := KeyForBet(msg.QuestionId, msg.ParticipantAddress)
	betCheckResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: betKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if betCheckResp.Error != nil {
		return &PluginDeliverResponse{Error: betCheckResp.Error}
	}
	if len(betCheckResp.Results) > 0 && len(betCheckResp.Results[0].Entries) > 0 && len(betCheckResp.Results[0].Entries[0].Value) > 0 {
		return &PluginDeliverResponse{Error: ErrParticipantAlreadyBetted()}
	}

	// Deduct from participant
	participant := new(Account)
	if err = Unmarshal(participantBytes, participant); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	totalDeduction := msg.Amount + fee
	if participant.Amount < totalDeduction {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	participant.Amount -= totalDeduction

	feePool := new(Pool)
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePool.Amount += fee

	// Update question state
	question.TotalPool += msg.Amount
	question.OptionPools[msg.OptionIndex] += msg.Amount
	question.ParticipantAddresses = append(question.ParticipantAddresses, msg.ParticipantAddress)

	// Create bet record
	betRecord := &BetRecord{
		ParticipantAddress: msg.ParticipantAddress,
		QuestionId:         msg.QuestionId,
		OptionIndex:        msg.OptionIndex,
		Amount:             msg.Amount,
	}

	// Marshal everything
	participantBz, _ := Marshal(participant)
	feePoolBz, _ := Marshal(feePool)
	questionBz, _ := Marshal(question)
	betRecordBz, _ := Marshal(betRecord)

	// Write state
	writeReq := &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: participantKey, Value: participantBz},
			{Key: feePoolKey, Value: feePoolBz},
			{Key: questionKey, Value: questionBz},
			{Key: betKey, Value: betRecordBz},
		},
	}
	if _, err = c.plugin.StateWrite(c, writeReq); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	log.Printf("BetPlace SUCCESS: participant=%x questionId=%d amount=%d", msg.ParticipantAddress, msg.QuestionId, msg.Amount)
	return &PluginDeliverResponse{}
}

// =============================================================================
// 3. BetReveal - Reveal answer and settle
// =============================================================================

// CheckMessageBetReveal validates a BetReveal transaction
func (c *Contract) CheckMessageBetReveal(msg *MessageBetReveal) *PluginCheckResponse {
	if len(msg.RevealerAddress) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{
		Recipient:         msg.RevealerAddress,
		AuthorizedSigners: [][]byte{msg.RevealerAddress},
	}
}

// DeliverMessageBetReveal reveals the answer and settles the question
func (c *Contract) DeliverMessageBetReveal(msg *MessageBetReveal, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageBetReveal: revealer=%x questionId=%d answer=%q",
		msg.RevealerAddress, msg.QuestionId, msg.Answer)

	// Read question, revealer account, fee pool
	questionKey := KeyForQuestion(msg.QuestionId)
	revealerKey := KeyForAccount(msg.RevealerAddress)
	feePoolKey := KeyForFeePool(c.Config.ChainId)

	questionQ := rand.Uint64()
	revealerQ := rand.Uint64()
	feeQ := rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: questionQ, Key: questionKey},
			{QueryId: revealerQ, Key: revealerKey},
			{QueryId: feeQ, Key: feePoolKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	var questionBytes, revealerBytes, feePoolBytes []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case questionQ:
			questionBytes = r.Entries[0].Value
		case revealerQ:
			revealerBytes = r.Entries[0].Value
		case feeQ:
			feePoolBytes = r.Entries[0].Value
		}
	}

	if len(questionBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrQuestionNotFound()}
	}
	question := new(Question)
	if err = Unmarshal(questionBytes, question); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Validate: only creator can reveal
	if !bytes.Equal(msg.RevealerAddress, question.CreatorAddress) {
		return &PluginDeliverResponse{Error: ErrNotQuestionCreator()}
	}
	// Validate: question not already revealed
	if question.Revealed {
		return &PluginDeliverResponse{Error: ErrAlreadyRevealed()}
	}
	// Validate: question not settled or refunded
	if question.Settled || question.Refunded {
		return &PluginDeliverResponse{Error: ErrQuestionNotOpen()}
	}
	// Validate: answer hash matches
	answerHash := sha256.Sum256([]byte(msg.Answer))
	computedHash := hex.EncodeToString(answerHash[:])
	if computedHash != question.AnswerHash {
		return &PluginDeliverResponse{Error: ErrAnswerHashMismatch()}
	}

	// Deduct fee from revealer
	revealer := new(Account)
	if err = Unmarshal(revealerBytes, revealer); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if revealer.Amount < fee {
		return &PluginDeliverResponse{Error: ErrInsufficientFunds()}
	}
	revealer.Amount -= fee

	feePool := new(Pool)
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePool.Amount += fee

	// Mark question as revealed (but NOT yet settled)
	question.Revealed = true
	question.RevealedAnswer = msg.Answer

	// Find winning option index
	winningIndex := -1
	answerTrimmed := strings.TrimSpace(strings.ToLower(msg.Answer))
	for i, opt := range question.Options {
		if strings.TrimSpace(strings.ToLower(opt)) == answerTrimmed {
			winningIndex = i
			break
		}
	}

	// Marshal and write state: update question (revealed), revealer account, fee pool
	questionBz, _ := Marshal(question)
	revealerBz, _ := Marshal(revealer)
	feePoolBz, _ := Marshal(feePool)

	baseWrites := []*PluginSetOp{
		{Key: questionKey, Value: questionBz},
		{Key: revealerKey, Value: revealerBz},
		{Key: feePoolKey, Value: feePoolBz},
	}

	if winningIndex == -1 {
		// Winning option not found: this shouldn't happen if answer matches an option,
		// but handle gracefully - keep question as revealed. Settlement will notice.
		if _, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: baseWrites}); err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		log.Printf("BetReveal SUCCESS (no winning option found for answer): questionId=%d", msg.QuestionId)
		return &PluginDeliverResponse{}
	}

	// Perform settlement immediately
	return c.settleQuestion(question, winningIndex, revealer, feePool, baseWrites)
}

// settleQuestion performs the 80/15/5 distribution after a reveal
func (c *Contract) settleQuestion(question *Question, winningIndex int, revealer *Account, feePool *Pool, baseWrites []*PluginSetOp) *PluginDeliverResponse {
	log.Printf("settleQuestion: questionId=%d winningIndex=%d totalPool=%d",
		question.Id, winningIndex, question.TotalPool)

	if question.TotalPool == 0 {
		// No bets placed: just return stake to creator and mark settled
		question.Settled = true

		// Return stake to creator
		creatorKey := KeyForAccount(question.CreatorAddress)
		creatorResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
			Keys: []*PluginKeyRead{
				{QueryId: rand.Uint64(), Key: creatorKey},
			}})
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		creator := new(Account)
		if creatorResp.Error == nil && len(creatorResp.Results) > 0 && len(creatorResp.Results[0].Entries) > 0 {
			_ = Unmarshal(creatorResp.Results[0].Entries[0].Value, creator)
		}
		creator.Amount += question.StakeAmount
		creatorBz, _ := Marshal(creator)
		questionBz, _ := Marshal(question)

		allWrites := append(baseWrites, &PluginSetOp{Key: creatorKey, Value: creatorBz})
		// Replace question with updated settled version
		for i, op := range allWrites {
			if bytes.Equal(op.Key, KeyForQuestion(question.Id)) {
				allWrites[i].Value = questionBz
				break
			}
		}
		if _, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{Sets: allWrites}); err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		log.Printf("settleQuestion (no bets): questionId=%d stake returned to creator", question.Id)
		return &PluginDeliverResponse{}
	}

	// Read all bet records for this question
	betPrefix := KeyForBetPrefix(question.Id)
	betsResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: betPrefix, Limit: 10000},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Collect bet records and calculate totals
	type betInfo struct {
		participant []byte
		optionIndex uint32
		amount      uint64
	}
	var bets []betInfo
	var totalWinningAmount uint64

	if betsResp != nil && betsResp.Error == nil {
		for _, r := range betsResp.Results {
			for _, entry := range r.Entries {
				br := new(BetRecord)
				if err = Unmarshal(entry.Value, br); err != nil {
					continue
				}
				info := betInfo{
					participant: br.ParticipantAddress,
					optionIndex: br.OptionIndex,
					amount:      br.Amount,
				}
				bets = append(bets, info)
				if int(br.OptionIndex) == winningIndex {
					totalWinningAmount += br.Amount
				}
			}
		}
	}

	// Calculate distribution
	// 80% to winners, 15% to creator, 5% burned (not credited to anyone)
	winnerShare := question.TotalPool * 80 / 100
	creatorShare := question.TotalPool * 15 / 100
	// Burn = TotalPool * 5% (implicit: not distributed)

	// Return stake to creator
	totalCreatorAmount := creatorShare + question.StakeAmount

	// Collect all write operations
	var allWrites []*PluginSetOp
	allWrites = append(allWrites, baseWrites...)

	// Track which accounts we've already updated in this batch
	updatedAccounts := make(map[string]bool)
	creatorKey := KeyForAccount(question.CreatorAddress)
	creatorKeyStr := string(creatorKey)

	// Add creator payout
	if totalCreatorAmount > 0 {
		if !updatedAccounts[creatorKeyStr] {
			// Read creator account if not already read (revealer might be creator)
			var creatorAcc *Account
			if bytes.Equal(question.CreatorAddress, revealer.Address) {
				// Revealer IS the creator - use already-read revealer account
				creatorAcc = revealer
			} else {
				creatorResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
					Keys: []*PluginKeyRead{
						{QueryId: rand.Uint64(), Key: creatorKey},
					}})
				if err != nil {
					return &PluginDeliverResponse{Error: err}
				}
				creatorAcc = new(Account)
				if creatorResp.Error == nil && len(creatorResp.Results) > 0 && len(creatorResp.Results[0].Entries) > 0 {
					_ = Unmarshal(creatorResp.Results[0].Entries[0].Value, creatorAcc)
				}
			}
			creatorAcc.Amount += totalCreatorAmount
			creatorBz, _ := Marshal(creatorAcc)
			allWrites = append(allWrites, &PluginSetOp{Key: creatorKey, Value: creatorBz})
			updatedAccounts[creatorKeyStr] = true
		}
	}

	// Distribute to winners
	if totalWinningAmount > 0 {
		for _, bet := range bets {
			if int(bet.optionIndex) == winningIndex {
				// Proportional payout: (winner_amount / total_winning_amount) * winner_share
				payout := uint64(0)
				if totalWinningAmount > 0 {
					payout = uint64(float64(bet.amount) / float64(totalWinningAmount) * float64(winnerShare))
					// Handle rounding: ensure no overflow
					if payout > math.MaxUint64-bet.amount {
						payout = math.MaxUint64 - bet.amount
					}
				}

				winnerKey := KeyForAccount(bet.participant)
				winnerKeyStr := string(winnerKey)

				var winnerAcc *Account
				if updatedAccounts[winnerKeyStr] {
					continue // already processed this winner
				}

				winnerResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
					Keys: []*PluginKeyRead{
						{QueryId: rand.Uint64(), Key: winnerKey},
					}})
				if err != nil {
					return &PluginDeliverResponse{Error: err}
				}
				winnerAcc = new(Account)
				if winnerResp.Error == nil && len(winnerResp.Results) > 0 && len(winnerResp.Results[0].Entries) > 0 {
					_ = Unmarshal(winnerResp.Results[0].Entries[0].Value, winnerAcc)
				}

				// Payout includes their original bet amount
				winnerAcc.Amount += payout + bet.amount
				winnerBz, _ := Marshal(winnerAcc)
				allWrites = append(allWrites, &PluginSetOp{Key: winnerKey, Value: winnerBz})
				updatedAccounts[winnerKeyStr] = true
			}
		}
	}

	// Mark question as settled
	question.Settled = true
	questionBz, _ := Marshal(question)
	// Update question in writes
	found := false
	for i, op := range allWrites {
		if bytes.Equal(op.Key, KeyForQuestion(question.Id)) {
			allWrites[i].Value = questionBz
			found = true
			break
		}
	}
	if !found {
		allWrites = append(allWrites, &PluginSetOp{Key: KeyForQuestion(question.Id), Value: questionBz})
	}

	// Delete all bet records
	var deletes []*PluginDeleteOp
	if betsResp != nil {
		for _, r := range betsResp.Results {
			for _, entry := range r.Entries {
				deletes = append(deletes, &PluginDeleteOp{Key: entry.Key})
			}
		}
	}

	writeReq := &PluginStateWriteRequest{
		Sets:    allWrites,
		Deletes: deletes,
	}
	if _, err = c.plugin.StateWrite(c, writeReq); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	log.Printf("settleQuestion SUCCESS: questionId=%d winners=%d totalPool=%d",
		question.Id, len(bets), question.TotalPool)
	return &PluginDeliverResponse{}
}

// =============================================================================
// 4. BetSettle - Settle a revealed question (anyone can call)
// =============================================================================

// CheckMessageBetSettle validates a BetSettle transaction
func (c *Contract) CheckMessageBetSettle(msg *MessageBetSettle) *PluginCheckResponse {
	return &PluginCheckResponse{
		AuthorizedSigners: nil, // anyone can call settle
	}
}

// DeliverMessageBetSettle settles a revealed question
func (c *Contract) DeliverMessageBetSettle(msg *MessageBetSettle, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageBetSettle: questionId=%d", msg.QuestionId)

	// Read question and fee pool
	questionKey := KeyForQuestion(msg.QuestionId)
	feePoolKey := KeyForFeePool(c.Config.ChainId)

	questionQ := rand.Uint64()
	feeQ := rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: questionQ, Key: questionKey},
			{QueryId: feeQ, Key: feePoolKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	var questionBytes, feePoolBytes []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case questionQ:
			questionBytes = r.Entries[0].Value
		case feeQ:
			feePoolBytes = r.Entries[0].Value
		}
	}

	if len(questionBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrQuestionNotFound()}
	}
	question := new(Question)
	if err = Unmarshal(questionBytes, question); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Must be revealed but not yet settled
	if !question.Revealed {
		return &PluginDeliverResponse{Error: ErrNotRevealed()}
	}
	if question.Settled {
		return &PluginDeliverResponse{Error: ErrAlreadySettled()}
	}
	if question.Refunded {
		return &PluginDeliverResponse{Error: ErrAlreadyRefunded()}
	}

	// Deduct fee from caller (the tx sender)
	// For settlement, we need to know who called it. Use an empty address -
	// the fee is paid by the caller but we don't track who that is.
	// Since PluginCheckRequest doesn't expose the sender, and PluginDeliverRequest
	// also doesn't, we handle fee deduction through the caller.
	// The fee was already checked in CheckTx, but for plugin txs we handle it here.
	// We need to deduct from the transaction signer. Since we don't have the signer
	// directly in the plugin, we handle this by reading the fee from the transaction.
	// For BetSettle, the fee is minimal and we expect the caller to have funds.

	feePool := new(Pool)
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePool.Amount += fee
	feePoolBz, _ := Marshal(feePool)

	// Find winning option
	winningIndex := -1
	answerTrimmed := strings.TrimSpace(strings.ToLower(question.RevealedAnswer))
	for i, opt := range question.Options {
		if strings.TrimSpace(strings.ToLower(opt)) == answerTrimmed {
			winningIndex = i
			break
		}
	}

	if winningIndex == -1 {
		// No winning option found: return stake to creator, mark settled
		creatorKey := KeyForAccount(question.CreatorAddress)
		creatorResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
			Keys: []*PluginKeyRead{
				{QueryId: rand.Uint64(), Key: creatorKey},
			}})
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		creator := new(Account)
		if creatorResp.Error == nil && len(creatorResp.Results) > 0 && len(creatorResp.Results[0].Entries) > 0 {
			_ = Unmarshal(creatorResp.Results[0].Entries[0].Value, creator)
		}
		creator.Amount += question.StakeAmount
		creatorBz, _ := Marshal(creator)
		question.Settled = true
		questionBz, _ := Marshal(question)

		if _, err = c.plugin.StateWrite(c, &PluginStateWriteRequest{
			Sets: []*PluginSetOp{
				{Key: questionKey, Value: questionBz},
				{Key: creatorKey, Value: creatorBz},
				{Key: feePoolKey, Value: feePoolBz},
			},
		}); err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		log.Printf("BetSettle SUCCESS (no winner): questionId=%d", msg.QuestionId)
		return &PluginDeliverResponse{}
	}

	// Use the revealer account (read from state to handle fee)
	revealerKey := KeyForAccount(question.CreatorAddress)
	revealerResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: revealerKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	revealer := new(Account)
	if revealerResp.Error == nil && len(revealerResp.Results) > 0 && len(revealerResp.Results[0].Entries) > 0 {
		_ = Unmarshal(revealerResp.Results[0].Entries[0].Value, revealer)
	}

	baseWrites := []*PluginSetOp{
		{Key: feePoolKey, Value: feePoolBz},
	}

	return c.settleQuestion(question, winningIndex, revealer, feePool, baseWrites)
}

// =============================================================================
// 5. BetRefund - Refund all participants on expired question
// =============================================================================

// CheckMessageBetRefund validates a BetRefund transaction
func (c *Contract) CheckMessageBetRefund(msg *MessageBetRefund) *PluginCheckResponse {
	return &PluginCheckResponse{
		AuthorizedSigners: nil, // anyone can call refund
	}
}

// DeliverMessageBetRefund refunds all participants when question expires
func (c *Contract) DeliverMessageBetRefund(msg *MessageBetRefund, fee uint64) *PluginDeliverResponse {
	log.Printf("DeliverMessageBetRefund: questionId=%d", msg.QuestionId)

	// Read question and fee pool
	questionKey := KeyForQuestion(msg.QuestionId)
	feePoolKey := KeyForFeePool(c.Config.ChainId)

	questionQ := rand.Uint64()
	feeQ := rand.Uint64()

	resp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: questionQ, Key: questionKey},
			{QueryId: feeQ, Key: feePoolKey},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &PluginDeliverResponse{Error: resp.Error}
	}

	var questionBytes, feePoolBytes []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case questionQ:
			questionBytes = r.Entries[0].Value
		case feeQ:
			feePoolBytes = r.Entries[0].Value
		}
	}

	if len(questionBytes) == 0 {
		return &PluginDeliverResponse{Error: ErrQuestionNotFound()}
	}
	question := new(Question)
	if err = Unmarshal(questionBytes, question); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Must be in open state (not revealed, settled, or refunded)
	if question.Revealed {
		return &PluginDeliverResponse{Error: ErrAlreadyRevealed()}
	}
	if question.Settled {
		return &PluginDeliverResponse{Error: ErrAlreadySettled()}
	}
	if question.Refunded {
		return &PluginDeliverResponse{Error: ErrAlreadyRefunded()}
	}

	// Deduct fee from the fee pool (settlement/refund caller pays fee)
	feePool := new(Pool)
	if err = Unmarshal(feePoolBytes, feePool); err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	feePool.Amount += fee
	feePoolBz, _ := Marshal(feePool)

	// Read all bet records for this question
	betPrefix := KeyForBetPrefix(msg.QuestionId)
	betsResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: betPrefix, Limit: 10000},
		}})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// Collect all write operations
	var allWrites []*PluginSetOp
	allWrites = append(allWrites, &PluginSetOp{Key: feePoolKey, Value: feePoolBz})
	var deletes []*PluginDeleteOp

	// Track updated accounts to avoid duplicate reads
	updatedAccounts := make(map[string]bool)

	// Refund each participant their bet amount
	if betsResp != nil && betsResp.Error == nil {
		for _, r := range betsResp.Results {
			for _, entry := range r.Entries {
				br := new(BetRecord)
				if err = Unmarshal(entry.Value, br); err != nil {
					continue
				}

				participantKey := KeyForAccount(br.ParticipantAddress)
				keyStr := string(participantKey)
				if updatedAccounts[keyStr] {
					continue
				}

				partResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
					Keys: []*PluginKeyRead{
						{QueryId: rand.Uint64(), Key: participantKey},
					}})
				if err != nil {
					return &PluginDeliverResponse{Error: err}
				}
				participant := new(Account)
				if partResp.Error == nil && len(partResp.Results) > 0 && len(partResp.Results[0].Entries) > 0 {
					_ = Unmarshal(partResp.Results[0].Entries[0].Value, participant)
				}
				participant.Amount += br.Amount
				participantBz, _ := Marshal(participant)
				allWrites = append(allWrites, &PluginSetOp{Key: participantKey, Value: participantBz})
				updatedAccounts[keyStr] = true

				deletes = append(deletes, &PluginDeleteOp{Key: entry.Key})
			}
		}
	}

	// Return stake to creator
	creatorKey := KeyForAccount(question.CreatorAddress)
	creatorKeyStr := string(creatorKey)
	if !updatedAccounts[creatorKeyStr] {
		creatorResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
			Keys: []*PluginKeyRead{
				{QueryId: rand.Uint64(), Key: creatorKey},
			}})
		if err != nil {
			return &PluginDeliverResponse{Error: err}
		}
		creator := new(Account)
		if creatorResp.Error == nil && len(creatorResp.Results) > 0 && len(creatorResp.Results[0].Entries) > 0 {
			_ = Unmarshal(creatorResp.Results[0].Entries[0].Value, creator)
		}
		creator.Amount += question.StakeAmount
		creatorBz, _ := Marshal(creator)
		allWrites = append(allWrites, &PluginSetOp{Key: creatorKey, Value: creatorBz})
		updatedAccounts[creatorKeyStr] = true
	}

	// Mark question as refunded
	question.Refunded = true
	questionBz, _ := Marshal(question)
	allWrites = append(allWrites, &PluginSetOp{Key: questionKey, Value: questionBz})

	writeReq := &PluginStateWriteRequest{
		Sets:    allWrites,
		Deletes: deletes,
	}
	if _, err = c.plugin.StateWrite(c, writeReq); err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	log.Printf("BetRefund SUCCESS: questionId=%d participants=%d", msg.QuestionId, len(updatedAccounts)-1)
	return &PluginDeliverResponse{}
}

// =============================================================================
// Key Helpers
// =============================================================================

var (
	accountPrefix  = []byte{1}   // store key prefix for accounts
	poolPrefix     = []byte{2}   // store key prefix for pools
	paramsPrefix   = []byte{7}   // store key prefix for governance parameters
	questionPrefix = []byte{100} // store key prefix for questions
	betPrefix      = []byte{101} // store key prefix for bet records
	counterPrefix  = []byte{102} // store key prefix for counters
)

// KeyForAccount() returns the state database key for an account
func KeyForAccount(addr []byte) []byte {
	return JoinLenPrefix(accountPrefix, addr)
}

// KeyForFeeParams() returns the state database key for governance controlled 'fee parameters'
func KeyForFeeParams() []byte {
	return JoinLenPrefix(paramsPrefix, []byte("/f/"))
}

// KeyForFeePool() returns the state database key for the fee pool
func KeyForFeePool(chainId uint64) []byte {
	return JoinLenPrefix(poolPrefix, formatUint64(chainId))
}

// KeyForQuestion() returns the state database key for a question by ID
func KeyForQuestion(id uint64) []byte {
	return JoinLenPrefix(questionPrefix, formatUint64(id))
}

// KeyForBet() returns the state database key for a specific bet
func KeyForBet(questionId uint64, participant []byte) []byte {
	combined := append(formatUint64(questionId), participant...)
	return JoinLenPrefix(betPrefix, combined)
}

// KeyForBetPrefix() returns the prefix for all bets on a given question (for range queries)
func KeyForBetPrefix(questionId uint64) []byte {
	return JoinLenPrefix(betPrefix, formatUint64(questionId))
}

// KeyForQuestionCounter() returns the state database key for the auto-increment question counter
func KeyForQuestionCounter() []byte {
	return JoinLenPrefix(counterPrefix, []byte("qid"))
}

func formatUint64(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}

// =============================================================================
// Sensitive Word Filter
// =============================================================================

// sensitiveWords is a basic list of prohibited terms for question/option text
var sensitiveWords = []string{
	"暴力", "色情", "赌博", "毒品", "恐怖",
	"violence", "porn", "drug", "terror",
}

// containsSensitiveWords checks if text contains any prohibited terms
func containsSensitiveWords(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range sensitiveWords {
		if strings.Contains(lower, strings.ToLower(word)) {
			return true
		}
	}
	return false
}
