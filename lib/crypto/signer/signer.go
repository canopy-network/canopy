// Package signer builds, signs, and verifies native Canopy send transactions.
//
// The package performs no network or filesystem access. Callers are responsible
// for obtaining a recent height and the current send fee before moving a request
// into an offline signing environment.
package signer

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// SendRequest contains every operator-controlled field of a native send
// AmountUCNPY and FeeUCNPY are integer micro-CNPY values. Time is a required
// Unix-microsecond value that must remain stable for every retry.
//
// Integer fields are JSON strings so values retain full uint64 precision in
// every language that invokes the command-line signer.
type SendRequest struct {
	Recipient     string `json:"recipient"`            // recipient address in hexadecimal format
	AmountUCNPY   uint64 `json:"amountUCNPY,string"`   // amount to send in micro-CNPY
	FeeUCNPY      uint64 `json:"feeUCNPY,string"`      // transaction fee in micro-CNPY
	CreatedHeight uint64 `json:"createdHeight,string"` // recent chain height used to create the transaction
	NetworkID     uint64 `json:"networkID,string"`     // destination network identifier
	ChainID       uint64 `json:"chainID,string"`       // destination chain identifier
	Memo          string `json:"memo,omitempty"`       // optional transaction memo
	Time          uint64 `json:"time,string"`          // transaction time in Unix microseconds
}

// Details is an auditable summary derived from a verified signed transaction
type Details struct {
	TransactionHash string `json:"transactionHash"`      // hash of the signed transaction in hexadecimal format
	Sender          string `json:"sender"`               // address derived from the signing public key
	Recipient       string `json:"recipient"`            // recipient address in hexadecimal format
	AmountUCNPY     uint64 `json:"amountUCNPY,string"`   // amount sent in micro-CNPY
	FeeUCNPY        uint64 `json:"feeUCNPY,string"`      // transaction fee in micro-CNPY
	CreatedHeight   uint64 `json:"createdHeight,string"` // chain height used to create the transaction
	NetworkID       uint64 `json:"networkID,string"`     // destination network identifier
	ChainID         uint64 `json:"chainID,string"`       // destination chain identifier
	Memo            string `json:"memo,omitempty"`       // optional transaction memo
	Time            uint64 `json:"time,string"`          // transaction time in Unix microseconds
}

// BuildSend() constructs an unsigned native send using Canopy's canonical
// protobuf types. It does not check the governance-controlled minimum fee or
// whether CreatedHeight is recent; those checks require current chain state.
func BuildSend(request SendRequest, publicKey crypto.PublicKeyI) (*lib.Transaction, error) {
	// validate the public key
	if publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}
	// validate the created height
	if request.CreatedHeight == 0 {
		return nil, fmt.Errorf("createdHeight must be greater than zero")
	}
	// validate the network identifier
	if request.NetworkID == 0 {
		return nil, fmt.Errorf("networkID must be greater than zero")
	}
	// validate the chain identifier
	if request.ChainID == 0 {
		return nil, fmt.Errorf("chainID must be greater than zero")
	}
	// validate the transaction time
	if request.Time == 0 {
		return nil, fmt.Errorf("time must be greater than zero")
	}
	// enforce the maximum memo length
	if len(request.Memo) > 200 {
		return nil, fmt.Errorf("memo exceeds 200 bytes")
	}
	// prevent native sends from using reserved Ethereum memo prefixes
	if lib.IsRLPMemo(request.Memo) {
		return nil, fmt.Errorf("memo %q is reserved for Ethereum transactions", request.Memo)
	}
	// decode and validate the recipient address
	recipient, err := crypto.NewAddressFromString(request.Recipient)
	if err != nil || len(recipient.Bytes()) != crypto.AddressSize {
		return nil, fmt.Errorf("recipient must be exactly 40 hexadecimal characters")
	}
	// create and validate the canonical send message
	message := &fsm.MessageSend{
		FromAddress: publicKey.Address().Bytes(),
		ToAddress:   recipient.Bytes(),
		Amount:      request.AmountUCNPY,
	}
	if checkErr := message.Check(); checkErr != nil {
		return nil, checkErr
	}
	// wrap the send message in its protobuf Any container
	anyMessage, anyErr := lib.NewAny(message)
	if anyErr != nil {
		return nil, anyErr
	}
	// assemble and return the unsigned transaction
	return &lib.Transaction{
		MessageType:   fsm.MessageSendName,
		Msg:           anyMessage,
		CreatedHeight: request.CreatedHeight,
		Time:          request.Time,
		Fee:           request.FeeUCNPY,
		Memo:          request.Memo,
		NetworkId:     request.NetworkID,
		ChainId:       request.ChainID,
	}, nil
}

// SignSend() builds and signs a native send, then verifies the resulting
// signature and signed fields before returning it.
func SignSend(request SendRequest, privateKey crypto.PrivateKeyI) (*lib.Transaction, error) {
	// validate the private key
	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	// retrieve and validate the paired public key
	publicKey := privateKey.PublicKey()
	if publicKey == nil {
		return nil, fmt.Errorf("private key public key is nil")
	}
	// build the unsigned send transaction
	transaction, err := BuildSend(request, publicKey)
	if err != nil {
		return nil, err
	}
	// create the canonical transaction sign bytes
	signBytes, signErr := transaction.GetSignBytes()
	if signErr != nil {
		return nil, signErr
	}
	// sign the transaction and ensure the signer returned a signature
	signature := privateKey.Sign(signBytes)
	if len(signature) == 0 {
		return nil, fmt.Errorf("signer returned an empty signature")
	}
	// attach the public key and signature to the transaction
	transaction.Signature = &lib.Signature{
		PublicKey: publicKey.Bytes(),
		Signature: signature,
	}
	// verify the complete transaction before returning it
	if _, err = Inspect(transaction); err != nil {
		return nil, fmt.Errorf("verify signed transaction: %w", err)
	}
	return transaction, nil
}

// Inspect() verifies a signed native send and returns the exact fields covered by
// its signature. Verification is stateless: final fee and height acceptance
// remain the responsibility of a synchronized Canopy node.
func Inspect(transaction *lib.Transaction) (*Details, error) {
	// validate the transaction reference
	if transaction == nil {
		return nil, fmt.Errorf("transaction is nil")
	}
	// execute the stateless transaction validation
	if err := transaction.CheckBasic(); err != nil {
		return nil, err
	}
	// ensure the transaction declares a native send
	if transaction.MessageType != fsm.MessageSendName {
		return nil, fmt.Errorf("transaction type must be %q", fsm.MessageSendName)
	}
	// decode the wrapped transaction message
	message, messageErr := lib.FromAny(transaction.Msg)
	if messageErr != nil {
		return nil, messageErr
	}
	// ensure the decoded payload is a native send
	send, ok := message.(*fsm.MessageSend)
	if !ok {
		return nil, fmt.Errorf("transaction payload is not a native send")
	}
	// validate the native send message
	if checkErr := send.Check(); checkErr != nil {
		return nil, checkErr
	}
	// decode the signing public key
	publicKey, err := crypto.NewPublicKeyFromBytes(transaction.Signature.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	// ensure the message sender matches the signing key
	if !bytes.Equal(send.FromAddress, publicKey.Address().Bytes()) {
		return nil, fmt.Errorf("sender does not match signing public key")
	}
	// recreate the canonical transaction sign bytes
	signBytes, signErr := transaction.GetSignBytes()
	if signErr != nil {
		return nil, signErr
	}
	// verify the transaction signature
	if !publicKey.VerifyBytes(signBytes, transaction.Signature.Signature) {
		return nil, fmt.Errorf("invalid transaction signature")
	}
	// calculate the verified transaction hash
	hash, hashErr := transaction.GetHash()
	if hashErr != nil {
		return nil, hashErr
	}
	// return the human-readable signed fields
	return &Details{
		TransactionHash: hex.EncodeToString(hash),
		Sender:          publicKey.Address().String(),
		Recipient:       hex.EncodeToString(send.ToAddress),
		AmountUCNPY:     send.Amount,
		FeeUCNPY:        transaction.Fee,
		CreatedHeight:   transaction.CreatedHeight,
		NetworkID:       transaction.NetworkId,
		ChainID:         transaction.ChainId,
		Memo:            transaction.Memo,
		Time:            transaction.Time,
	}, nil
}
