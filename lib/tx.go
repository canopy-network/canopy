package lib

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file creates and implements interfaces for blockchain transactions */

var (
	_ TransactionI = &Transaction{} // TransactionI interface enforcement of the Transaction struct
	_ SignatureI   = &Signature{}   // SignatureI interface enforcement of the Signature struct
	_ TxResultI    = &TxResult{}    // TxResultI interface enforcement of the TxResult struct
	_ Pageable     = new(TxResults) // Pageable interface enforcement of the TxResult struct
)

const (
	TxResultsPageName      = "tx-results-page"      // the name of a page of transactions
	PendingResultsPageName = "pending-results-page" //  the name of a page of mempool pending transactions
	FailedTxsPageName      = "failed-txs-page"      // the name of a page of failed transactions
	EventsPageName         = "events-page"
)

// Messages must be pre-registered for Transaction JSON unmarshalling
var RegisteredMessages map[string]MessageI

func init() {
	RegisteredMessages = make(map[string]MessageI)
}

// TRANSACTION INTERFACES BELOW

// TxResultI is the model of a completed transaction object after execution
type TxResultI interface {
	proto.Message
	GetSender() []byte      // the sender of the transaction
	GetRecipient() []byte   // the receiver of the transaction (i.e. the recipient of a 'send' transaction; empty of not applicable)
	GetMessageType() string // the type of message the transaction contains
	GetHeight() uint64      // the block number the transaction is included in
	GetIndex() uint64       // the index of the transaction in the block
	GetTxHash() string      // the cryptographic hash of the transaction
	GetTx() TransactionI
}

// TransactionI is the model of a transaction object (a record of an action or event)
type TransactionI interface {
	proto.Message
	GetMsg() *anypb.Any             // message payload (send, stake, edit-stake, etc.)
	GetSig() SignatureI             // digital signature allowing public key verification
	GetTime() uint64                // a stateless - prune friendly, replay attack / hash collision defense (opposed to sequence)
	GetSignBytes() ([]byte, ErrorI) // the canonical form the bytes were signed in
	GetHash() ([]byte, ErrorI)      // the computed cryptographic hash of the transaction bytes
	GetMemo() string                // an optional 100 character descriptive string - these are often used for polling
}

// SignatureI is the model of a signature object (a signature and public key bytes pair)
type SignatureI interface {
	proto.Message
	GetPublicKey() []byte
	GetSignature() []byte
}

// MessageI is the model of a message object (send, stake, edit-stake, etc.)
type MessageI interface {
	proto.Message

	New() MessageI     // new instance of the message type
	Name() string      // name of the message
	Check() ErrorI     // stateless validation of the message
	Recipient() []byte // for transaction indexing by recipient
	json.Marshaler     // json encoding
	json.Unmarshaler   // json decoding
}

// TRANSACTION CODE BELOW

// CheckBasic() is a stateless validation function for a Transaction object
func (x *Transaction) CheckBasic() ErrorI {
	// if the transaction is empty
	if x == nil {
		// exit with empty error
		return ErrEmptyTransaction()
	}
	// if the payload is empty
	if x.Msg == nil {
		// exit with empty payload error
		return ErrEmptyMessage()
	}
	// if the message type is empty
	if x.MessageType == "" {
		// exit with empty payload name error
		return ErrUnknownMessageName(x.MessageType)
	}
	// if any parts of the signature is empty
	if x.Signature == nil || x.Signature.Signature == nil || x.Signature.PublicKey == nil {
		// exit with empty signature error
		return ErrEmptySignature()
	}
	// if the created height is empty
	if x.CreatedHeight == 0 {
		// exit with created height error
		return ErrInvalidTxHeight()
	}
	// if the time is empty
	if x.Time == 0 {
		// exit with invalid time error
		return ErrInvalidTxTime()
	}
	// if the memo is too long
	if len(x.Memo) > 200 {
		// exit with 'memo too long' error
		return ErrInvalidMemo()
	}
	// if the network id is empty
	if x.NetworkId == 0 {
		// exit with empty network id error
		return ErrNilNetworkID()
	}
	// if the chain id is empty
	if x.ChainId == 0 {
		// exit with empty chain id error
		return ErrEmptyChainId()
	}
	// exit with no error
	return nil
}

// GetHash() returns the cryptographic hash of the Transaction
func (x *Transaction) GetHash() ([]byte, ErrorI) {
	// convert the transaction into proto bytes
	protoBytes, err := Marshal(x)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, err
	}
	// exit with the hash of the proto bytes
	return crypto.Hash(protoBytes), nil
}

// GetSig() accessor for signature field (do not delete: needed to satisfy TransactionI)
func (x *Transaction) GetSig() SignatureI { return x.Signature }

// GetSignBytes() returns the canonical byte representation of the Transaction for signing and signature verification
func (x *Transaction) GetSignBytes() ([]byte, ErrorI) {
	// exit with proto bytes but omit the signature
	return Marshal(&Transaction{
		MessageType:   x.MessageType,
		Msg:           x.Msg,
		Signature:     nil,
		Time:          x.Time,
		CreatedHeight: x.CreatedHeight,
		Fee:           x.Fee,
		Memo:          x.Memo,
		NetworkId:     x.NetworkId,
		ChainId:       x.ChainId,
	})
}

// Sign() executes a digital signature on the transaction
func (x *Transaction) Sign(pk crypto.PrivateKeyI) (err ErrorI) {
	// get the sign bytes for the transaction
	signBytes, err := x.GetSignBytes()
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return
	}
	// populate the signature field
	x.Signature = &Signature{
		PublicKey: pk.PublicKey().Bytes(),
		Signature: pk.Sign(signBytes),
	}
	// exit
	return
}

// jsonTx implements the json.Marshaller and json.Unmarshaler interface for the Transaction type
type jsonTx struct {
	Type          string          `json:"type,omitempty"`
	Msg           json.RawMessage `json:"msg,omitempty"`
	MsgTypeURL    string          `json:"msgTypeUrl,omitempty"`
	MsgBytes      string          `json:"msgBytes,omitempty"`
	Signature     *Signature      `json:"signature,omitempty"`
	Time          uint64          `json:"time,omitempty"`
	CreatedHeight uint64          `json:"createdHeight,omitempty"`
	Fee           uint64          `json:"fee,omitempty"`
	Memo          string          `json:"memo,omitempty"`
	NetworkId     uint64          `json:"networkID,omitempty"`
	ChainId       uint64          `json:"chainID,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface for the Transaction type
func (x Transaction) MarshalJSON() (jsonBytes []byte, err error) {
	if x.Msg == nil {
		return nil, fmt.Errorf("transaction message is nil")
	}
	var (
		messageRawJSON json.RawMessage
		msgTypeURL     string
		msgBytes       string
	)
	// convert the payload from a proto.Any to a JSON object when possible
	messageRawJSON, err = MarshalAnypbJSON(x.Msg)
	if err != nil {
		msgTypeURL = x.Msg.TypeUrl
		msgBytes = BytesToString(x.Msg.Value)
	}
	// exit by converting a new json object into json bytes
	return json.Marshal(jsonTx{
		Type:          x.MessageType,
		Msg:           messageRawJSON,
		MsgTypeURL:    msgTypeURL,
		MsgBytes:      msgBytes,
		Signature:     x.Signature,
		Time:          x.Time,
		CreatedHeight: x.CreatedHeight,
		Fee:           x.Fee,
		Memo:          x.Memo,
		NetworkId:     x.NetworkId,
		ChainId:       x.ChainId,
	})
}

// MarshalJSON() implements the json.Unmarshaler interface for the Transaction type
func (x *Transaction) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non nil result
	j := new(jsonTx)
	// populate the json object with json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		return err
	}
	// first try unmarshalling using the global plugin registration
	if len(j.Msg) > 0 {
		anyMsg, e := AnyFromJSONForMessageType(j.Type, j.Msg)
		if e == nil {
			*x = Transaction{
				MessageType:   j.Type,
				Msg:           anyMsg,
				Signature:     j.Signature,
				CreatedHeight: j.CreatedHeight,
				Time:          j.Time,
				Fee:           j.Fee,
				Memo:          j.Memo,
				NetworkId:     j.NetworkId,
				ChainId:       j.ChainId,
			}
			return nil
		} else if e.Code() != CodeUnknownMsgName {
			return e
		}
	}
	// get the type of the message payload based on the 'message types' that were globally registered upon app start
	m, found := RegisteredMessages[j.Type]
	// if the message type is not found among the registered messages
	if !found {
		if j.MsgTypeURL == "" && j.MsgBytes == "" {
			// exit with error
			return ErrUnknownMessageName(j.Type)
		}
		var msgValue []byte
		if j.MsgBytes != "" {
			msgValue, err = StringToBytes(j.MsgBytes)
			if err != nil {
				return err
			}
		}
		// populate the underlying transaction object using raw any bytes
		*x = Transaction{
			MessageType:   j.Type,
			Msg:           &anypb.Any{TypeUrl: j.MsgTypeURL, Value: msgValue},
			Signature:     j.Signature,
			CreatedHeight: j.CreatedHeight,
			Time:          j.Time,
			Fee:           j.Fee,
			Memo:          j.Memo,
			NetworkId:     j.NetworkId,
			ChainId:       j.ChainId,
		}
		return nil
	}
	// create a new instance of the message
	msg := m.New()
	// populate the new message using the json bytes in the json object
	if err = json.Unmarshal(j.Msg, msg); err != nil {
		// exit with error
		return
	}
	// convert the message to a proto.Any
	a, err := NewAny(msg)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return
	}
	// populate the underlying transaction object
	*x = Transaction{
		MessageType:   j.Type,
		Msg:           a,
		Signature:     j.Signature,
		CreatedHeight: j.CreatedHeight,
		Time:          j.Time,
		Fee:           j.Fee,
		Memo:          j.Memo,
		NetworkId:     j.NetworkId,
		ChainId:       j.ChainId,
	}
	// exit
	return
}

// TRANSACTION RESULT CODE BELOW

type TxResults []*TxResult

func (t *TxResults) Len() int      { return len(*t) }
func (t *TxResults) New() Pageable { return &TxResults{} }

// GetTx() is an accessor for the Transaction field
func (x *TxResult) GetTx() TransactionI { return x.Transaction }

// jsonTxResult implements the json.Marshaller and json.Unmarshaler interfaces for TxResult
type jsonTxResult struct {
	Sender      HexBytes     `json:"sender,omitempty"`
	Recipient   HexBytes     `json:"recipient,omitempty"`
	MessageType string       `json:"messageType,omitempty"`
	Height      uint64       `json:"height,omitempty"`
	Index       uint64       `json:"index,omitempty"`
	Transaction *Transaction `json:"transaction,omitempty"`
	TxHash      string       `json:"txHash,omitempty"`
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x TxResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonTxResult{
		Sender:      x.Sender,
		Recipient:   x.Recipient,
		MessageType: x.MessageType,
		Height:      x.Height,
		Index:       x.Index,
		Transaction: x.Transaction,
		TxHash:      x.TxHash,
	})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *TxResult) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json tx result object to ensure a non-nil result
	j := new(jsonTxResult)
	// populate the object using the json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying tx result object using the json object
	*x = TxResult{
		Sender:      j.Sender,
		Recipient:   j.Recipient,
		MessageType: j.MessageType,
		Height:      j.Height,
		Index:       j.Index,
		Transaction: j.Transaction,
		TxHash:      j.TxHash,
	}
	// exit
	return
}

// APPLY TXS RESULTS CODE BELOW

// ApplyBlockResults() represents the information returned by the ApplyTransactions function
type ApplyBlockResults struct {
	// included
	Txs       [][]byte    // the bytes of the transactions 'included' in the block
	Results   []*TxResult // the results of the transactions 'included' in the block
	ResultsBz [][]byte    // the bytes of the transaction results 'included' in the block
	Events    []*Event    // the events of the transactions 'included' in the block
	TxRoot    []byte      // the root of the 'included' transaction results
	Count     int         // the count of transactions 'included' in the block
	BlockSize uint64      // the size of the current block
	LargestTx uint64      // the size of the largest transaction (for metrics)
	// excluded
	Oversized []*TxResult // the results of the transactions 'excluded' from the block due to block size
	Failed    []*FailedTx // the results of the failed transactions 'excluded' from the block
}

// Add() adds a valid transaction to the results
func (a *ApplyBlockResults) Add(tx, txResult []byte, result *TxResult, events []*Event, oversized bool) {
	if oversized {
		a.Oversized = append(a.Oversized, result)
		return
	}
	a.Results = append(a.Results, result)
	a.Events = append(a.Events, events...)
	a.Txs = append(a.Txs, tx)
	a.ResultsBz = append(a.ResultsBz, txResult)
	a.Count++
	txSize := uint64(len(tx))
	a.BlockSize += txSize
	if txSize > a.LargestTx {
		a.LargestTx = txSize
	}
}

// AddFailed() adds a failed transaction to the results
func (a *ApplyBlockResults) AddFailed(f *FailedTx) {
	a.Failed = append(a.Failed, f)
}

// AddEvent() adds a failed transaction to the results
func (a *ApplyBlockResults) AddEvent(e ...*Event) {
	a.Events = append(a.Events, e...)
}

// TransactionRoot() returns the transaction results root
func (a *ApplyBlockResults) TransactionRoot() (root []byte, err ErrorI) {
	if a.TxRoot == nil {
		a.TxRoot, _, err = MerkleTree(a.ResultsBz)
	}
	return a.TxRoot, err
}

// SIGNATURE CODE BELOW

// Signable is a proto.Message that can be signed using a crypto.PrivateKey
type Signable interface {
	proto.Message
	Sign(p crypto.PrivateKeyI) ErrorI
}

// SignByte is a object that returns canonical bytes to sign
type SignByte interface{ SignBytes() []byte }

// Copy() returns a deep clone of the Signature object
func (x *Signature) Copy() *Signature {
	// return a deep copy of the signature
	return &Signature{
		PublicKey: bytes.Clone(x.PublicKey),
		Signature: bytes.Clone(x.Signature),
	}
}

// Signature satisfies the json.Marshaller and json.Unmarshaler interfaces

// MarshalJSON() satisfies the json.Marshaller interface
func (x Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSignature{x.PublicKey, x.Signature})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *Signature) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non-nil result
	j := new(jsonSignature)
	// populate the new json object reference with json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying Signature object using the json object
	x.PublicKey, x.Signature = j.PublicKey, j.Signature
	// exit
	return
}

// jsonSignature satisfies the json.Marshaller and json.Unmarshaler interfaces for Signature
type jsonSignature struct {
	PublicKey HexBytes `json:"publicKey,omitempty"`
	Signature HexBytes `json:"signature,omitempty"`
}
