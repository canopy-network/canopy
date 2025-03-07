package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file has logic to certify the next block and result decided by a bft quorum */

const (
	// the max possible block size without checking the governance parameter in state
	GlobalMaxBlockSize = int(32 * units.MB)
	// ensures developers are aware of a change to the header size (which is a consensus breaking change)
	ExpectedMaxBlockHeaderSize = 1652
)

// MaxBlockHeaderSize is a consensus breaking change because it affects how the state machine
// checks if a block is above the MaxBlockSize as the State Machine is only aware of the txs.
var MaxBlockHeaderSize uint64

// QUORUM CERTIFICATE CODE BELOW

// CheckBasic() performs 'sanity' checks on the Quorum Certificate structure
func (x *QuorumCertificate) CheckBasic() ErrorI {
	// a valid QC must have either the results hash or the proposer key set
	if x == nil || (x.ResultsHash == nil && x.ProposerKey == nil) {
		// exit with empty qc error
		return ErrEmptyQuorumCertificate()
	}
	// sanity check the view of the QC
	if err := x.Header.CheckBasic(); err != nil {
		// exit with error
		return err
	}
	// is QC with result (AFTER ELECTION)
	if x.ResultsHash != nil {
		// check the block hash for the proper size
		if len(x.BlockHash) != crypto.HashSize {
			return ErrInvalidBlockHash()
		}
		// check the result hash for the proper size
		if len(x.ResultsHash) != crypto.HashSize {
			return ErrInvalidResultsHash()
		}
		// results may be omitted in certain cases like double sign evidence
		if x.Results != nil {
			if err := x.Results.CheckBasic(); err != nil {
				return err
			}
			// validate the ProposalHash = the hash of the proposal sign bytes
			resultsBytes, err := Marshal(x.Results)
			if err != nil {
				return err
			}
			// check the results hash
			if !bytes.Equal(x.ResultsHash, crypto.Hash(resultsBytes)) {
				return ErrMismatchResultsHash()
			}
		}
		// block may be omitted in certain cases like the 'reward transaction'
		if x.Block != nil {
			// create a new block object reference to ensure a non nil result
			block := new(Block)
			// populate the block structure with the block bytes in the certificate
			hash, e := block.BytesToBlockHash(x.Block)
			// if an error occurred during this conversion
			if e != nil {
				// exit with the error
				return e
			}
			// check the block hash
			if !bytes.Equal(x.BlockHash, hash) {
				return ErrMismatchQCBlockHash()
			}
			// ensure the number of bytes in the block doesn't exceed the global max block size
			blockSize := len(x.Block)
			// global max block size enforcement
			if blockSize > GlobalMaxBlockSize {
				return ErrExpectedMaxBlockSize()
			}
		}
		// is QC with proposer key (ELECTION)
	} else {
		// ensure the proposer key is the proper size
		if len(x.ProposerKey) != crypto.BLS12381PubKeySize {
			return ErrInvalidSigner()
		}
		// ensure the results and result hash are empty
		if len(x.ResultsHash) != 0 || x.Results != nil {
			return ErrMismatchResultsHash()
		}
		// ensure the block and block hash are empty
		if len(x.BlockHash) != 0 || len(x.Block) != 0 {
			return ErrNonNilBlock()
		}
	}
	// ensure a valid aggregate signature is possible
	return x.Signature.CheckBasic()
}

// Check() validates the QC by cross-checking the aggregate signature against the ValidatorSet
// isPartialQC means a valid aggregate signature, but not enough signers for +2/3 majority
func (x *QuorumCertificate) Check(vs ValidatorSet, maxBlockSize int, view *View, enforceHeights bool) (isPartialQC bool, error ErrorI) {
	// do basic sanity checks on the certificate
	if err := x.CheckBasic(); err != nil {
		// exit with error
		return false, err
	}
	// check the header
	if err := x.Header.Check(view, enforceHeights); err != nil {
		// exit with error
		return false, err
	}
	// enforce 'max block size'
	if len(x.Block) > maxBlockSize {
		// exit with error
		return false, ErrExpectedMaxBlockSize()
	}
	// verify the aggregate signature in the certificate
	return x.Signature.Check(x, vs)
}

// CheckProposalBasic() does a basic validity check on the proposal inside the QC and returns the block structure
func (x *QuorumCertificate) CheckProposalBasic(height, networkId, chainId uint64) (block *Block, err ErrorI) {
	// ensure the block is not empty
	if x.Block == nil {
		// exit with nil block error
		return nil, ErrNilBlock()
	}
	// create a new block object reference to ensure a non nil result
	block = new(Block)
	// populate the block obj ref with the block bytes in the qc
	if err = Unmarshal(x.Block, block); err != nil {
		return
	}
	// perform stateless checks on the block
	if err = block.Check(networkId, chainId); err != nil {
		return
	}
	// enforce the target height
	if x.Header.Height != height || x.Header.Height != block.BlockHeader.Height {
		return nil, ErrWrongHeight()
	}
	// don't accept any blocks below the local height
	if height > block.BlockHeader.Height {
		return nil, ErrWrongHeight()
	}
	// ensure the Proposal.BlockHash corresponds to the actual hash of the block
	blockHash, err := block.Hash()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(x.BlockHash, blockHash) {
		return nil, ErrMismatchHeaderBlockHash()
	}
	// ensure the results aren't empty
	if x.Results == nil && x.Results.RewardRecipients != nil {
		return nil, ErrNilCertResults()
	}
	// exit
	return
}

// CheckHighQC() performs validation on the special `HighQC` (justify unlock QC)
func (x *QuorumCertificate) CheckHighQC(maxBlockSize int, view *View, lastRootHeightUpdated uint64, vs ValidatorSet) ErrorI {
	// validate the certificate and check the aggregate signature
	isPartialQC, err := x.Check(vs, maxBlockSize, view, false)
	// if an error occurred
	if err != nil {
		// exit with error
		return err
	}
	// `highQCs` can't justify an unlock without +2/3 majority
	if isPartialQC {
		// exit with no +2/3
		return ErrNoMaj23()
	}
	// invalid 'historical committee', if the root height of the committee is before that saved in state
	if lastRootHeightUpdated > x.Header.RootHeight {
		// exit with wrong root height error
		return ErrWrongRootHeight()
	}
	// enforce same target height
	if x.Header.Height != view.Height {
		// exit with wrong height error
		return ErrWrongHeight()
	}
	// a valid HighQC has the phase PRECOMMIT_VOTE as that's the phase where replicas 'Lock'
	if x.Header.Phase != Phase_PROPOSE_VOTE {
		// exit with wrong phase error
		return ErrWrongPhase()
	}
	// exit
	return nil
}

// SignBytes() returns the canonical byte representation used to digitally sign the bytes of the structure
func (x *QuorumCertificate) SignBytes() (signBytes []byte) {
	// if the certificate is for the phase 'election vote'
	if x.Header != nil && x.Header.Phase == Phase_ELECTION_VOTE {
		// create a simplified version of the qc
		minified := &QuorumCertificate{Header: x.Header, ProposerKey: x.ProposerKey}
		// convert the minified version to bytes
		signBytes, _ = Marshal(minified)
		// exit with the bytes
		return
	}
	// create temp variables to save values
	results, block, aggregateSignature := x.Results, x.Block, x.Signature
	// remove the values from the struct
	x.Results, x.Block, x.Signature = nil, nil, nil
	// convert the structure into the sign bytes
	signBytes, _ = Marshal(x)
	// add back the removed values
	x.Results, x.Block, x.Signature = results, block, aggregateSignature
	// exit with the bytes
	return
}

// EqualPayloads() compares the payloads only of two certs (can have different signatures)
func (x *QuorumCertificate) EqualPayloads(compare *QuorumCertificate) bool {
	// returns if both certificates have the same height, proposer key, block hash and result hash
	return x != nil && x.Header != nil &&
		bytes.Equal(x.ProposerKey, compare.ProposerKey) &&
		x.Header.Height == compare.Header.Height &&
		bytes.Equal(x.BlockHash, compare.BlockHash) &&
		bytes.Equal(x.ResultsHash, compare.ResultsHash)
}

// GetNonSigners() returns the public keys and the percentage (of voting power out of total) of those who did not sign the QC
func (x *QuorumCertificate) GetNonSigners(vs *ConsensusValidators) (nonSignerPubKeys [][]byte, nonSignerPercent int, err ErrorI) {
	// ensure both the certificate and the signature are non-nil
	if x == nil || x.Signature == nil {
		// exit with empty qc error
		return nil, 0, ErrEmptyQuorumCertificate()
	}
	// retrieve the non-signers from the signature using teh validator set
	return x.Signature.GetNonSigners(vs)
}

// jsonQC represents the json.Marshaller and json.Unmarshaler implementation of QC
type jsonQC struct {
	Header       *View               `json:"header,omitempty"`
	Block        HexBytes            `json:"block,omitempty"`
	BlockHash    HexBytes            `json:"blockHash,omitempty"`
	ResultsHash  HexBytes            `json:"resultsHash,omitempty"`
	Results      *CertificateResult  `json:"results,omitempty"`
	ProposalHash HexBytes            `json:"proposalHash,omitempty"`
	ProposerKey  HexBytes            `json:"proposerKey,omitempty"`
	Signature    *AggregateSignature `json:"signature,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface
func (x QuorumCertificate) MarshalJSON() ([]byte, error) {
	// convert the quorum certificate to json bytes
	return json.Marshal(jsonQC{
		Header:      x.Header,
		Results:     x.Results,
		ResultsHash: x.ResultsHash,
		Block:       x.Block,
		BlockHash:   x.BlockHash,
		ProposerKey: x.ProposerKey,
		Signature:   x.Signature,
	})
}

// UnmarshalJSON() implements the json.Unmarshaler interface
func (x *QuorumCertificate) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new jsonQC object reference to ensure a non-nil result
	j := new(jsonQC)
	// populate the jsonQC with json bytes
	if err = json.Unmarshal(jsonBytes, &j); err != nil {
		// exit with error
		return
	}
	// set the underlying object using the json qc values
	*x = QuorumCertificate{
		Header:      j.Header,
		Results:     j.Results,
		ResultsHash: j.ResultsHash,
		Block:       j.Block,
		BlockHash:   j.BlockHash,
		ProposerKey: j.ProposerKey,
		Signature:   j.Signature,
	}
	// exit
	return
}

// A CertificateResult contains actions on stakeholders as determined by the consensus process

// CERTIFICATE RESULT CODE BELOW

// CheckBasic() provides basic 'sanity' checks on the CertificateResult structure
func (x *CertificateResult) CheckBasic() (err ErrorI) {
	// ensure the certificate result is not nil
	if x == nil {
		// exit with empty certificate results error
		return ErrNilCertResults()
	}
	// do basic sanity checks on the reward recipients
	if err = x.RewardRecipients.CheckBasic(); err != nil {
		// exit with error
		return
	}
	// do basic sanity checks on the slash recipients
	if err = x.SlashRecipients.CheckBasic(); err != nil {
		// exit with error
		return
	}
	// do basic sanity checks on the swaps
	if err = x.Orders.CheckBasic(); err != nil {
		// exit with error
		return
	}
	// do basic sanity check on the 'checkpoint'
	return x.Checkpoint.CheckBasic()
}

// Equals() compares two certificate results to ensure equality
func (x *CertificateResult) Equals(y *CertificateResult) bool {
	// if either of the certificate results are nil
	if x == nil || y == nil {
		// return unequal
		return false
	}
	// if the reward recipients aren't equal
	if !x.RewardRecipients.Equals(y.RewardRecipients) {
		// return unequal
		return false
	}
	// if the slash recipients aren't equal
	if !x.SlashRecipients.Equals(y.SlashRecipients) {
		// return unequal
		return false
	}
	// if the swaps aren't equal
	if !x.Orders.Equals(y.Orders) {
		// return unequal
		return false
	}
	// if checkpoints aren't equal
	if !x.Checkpoint.Equals(y.Checkpoint) {
		// return unequal
		return false
	}
	// return equality based on the final field
	return x.Retired == y.Retired
}

// Hash() returns the cryptographic hash of the canonical Sign Bytes of the CertificateResult
func (x *CertificateResult) Hash() []byte {
	// convert the certificate results to proto bytes
	bz, _ := Marshal(x)
	// return the hash of the bytes
	return crypto.Hash(bz)
}

// REWARD RECIPIENT CODE BELOW

// CheckBasic() performs a basic 'sanity check' on the structure
func (x *RewardRecipients) CheckBasic() (err ErrorI) {
	// ensure the reward recipients aren't null
	if x == nil {
		// exit with null error
		return ErrNilRewardRecipients()
	}
	// validate the number of recipients
	paymentRecipientCount := len(x.PaymentPercents)
	// ensure the count is not zero nor is bigger than 100
	if paymentRecipientCount == 0 || paymentRecipientCount > 100 {
		// exit with invalid payment recipients count
		return ErrPaymentRecipientsCount()
	}
	// create a map to ensure the payment percents don't exceed 100% per chain
	chainMap := make(map[uint64]uint64)
	// for each payment percent
	for _, pp := range x.PaymentPercents {
		// ensure each percent isn't nil
		if pp == nil {
			// exit with an invalid payment percent allocation
			return ErrInvalidPercentAllocation()
		}
		// ensure the chain id isn't 0
		if pp.ChainId == 0 {
			// exit with empty chain id
			return ErrEmptyChainId()
		}
		// ensure each percent address is the right size
		if len(pp.Address) != crypto.AddressSize {
			// exit with invalid recipient
			return ErrInvalidAddress()
		}
		// add to total percent
		chainMap[pp.ChainId] += pp.Percent
		// ensure the percent doesn't exceed 100
		if chainMap[pp.ChainId] > 100 {
			// exit with allocation error
			return ErrInvalidPercentAllocation()
		}
	}
	// exit
	return
}

// Equals() compares two RewardRecipients for equality
func (x *RewardRecipients) Equals(y *RewardRecipients) bool {
	// if both of the reward recipients are empty
	if x == nil && y == nil {
		// exit with 'equal'
		return true
	}
	// if either of the reward recipients are empty
	if x == nil || y == nil {
		// exit with 'unequal'
		return false
	}
	// if the payment percents sizes differ
	if len(x.PaymentPercents) != len(y.PaymentPercents) {
		// exit with 'unequal'
		return false
	}
	// for each payment percent
	for i, pp := range x.PaymentPercents {
		// if the address differs in the payment percent
		if pp.ChainId != y.PaymentPercents[i].ChainId {
			// exit with 'unequal'
			return false
		}
		// if the address differs in the payment percent
		if !bytes.Equal(pp.Address, y.PaymentPercents[i].Address) {
			// exit with 'unequal'
			return false
		}
		// if the percent allocation differs
		if pp.Percent != y.PaymentPercents[i].Percent {
			// exit with 'unequal'
			return false
		}
	}
	// exit with an equality check on the final field
	return x.NumberOfSamples == y.NumberOfSamples
}

// jsonRewardRecipients is the RewardRecipients implementation of json.Marshaller and json.Unmarshaler
type jsonRewardRecipients struct {
	// recipients of the block reward by percentage
	PaymentPercents []*PaymentPercents `json:"paymentPercents,omitempty"`
	// number of samples combined (only applicable at state machine level)
	NumberOfSamples uint64 `json:"numberOfSamples,omitempty"`
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x RewardRecipients) MarshalJSON() ([]byte, error) {
	// convert the reward recipients to json bytes using the json structure
	return json.Marshal(jsonRewardRecipients{
		PaymentPercents: x.PaymentPercents,
		NumberOfSamples: x.NumberOfSamples,
	})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *RewardRecipients) UnmarshalJSON(jsonBytes []byte) (err error) {
	// initialize a new reward recipients object reference to ensure a non-nil result
	j := new(jsonRewardRecipients)
	// populate the object reference using the json object reference
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying object using the json object reference
	*x = RewardRecipients{
		PaymentPercents: j.PaymentPercents,
		NumberOfSamples: j.NumberOfSamples,
	}
	// exit
	return
}

// PAYMENT PERCENTS CODE BELOW

// jsonPaymentPercents is the PaymentPercents implementation of json.Marshaller and json.Unmarshaler
type jsonPaymentPercents struct {
	Address  HexBytes `json:"address"`
	Percents uint64   `json:"percents"`
	ChainId  uint64   `json:"chainId"`
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x PaymentPercents) MarshalJSON() ([]byte, error) {
	// convert the payment percents to json bytes using the json object
	return json.Marshal(jsonPaymentPercents{
		Address:  x.Address,
		Percents: x.Percent,
		ChainId:  x.ChainId,
	})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *PaymentPercents) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new object reference for payment percents
	pp := new(jsonPaymentPercents)
	// populate the object using the json bytes
	if err = json.Unmarshal(jsonBytes, &pp); err != nil {
		// exit with error
		return
	}
	// populate the underlying object using the json object ref
	x.Address, x.Percent, x.ChainId = pp.Address, pp.Percents, pp.ChainId
	// exit
	return
}

// SLASH RECIPIENTS CODE BELOW

// CheckBasic() validates the ProposalMeta structure
func (x *SlashRecipients) CheckBasic() (err ErrorI) {
	// if the slash recipients are nil
	if x == nil {
		// exit without error
		return
	}
	// for each double signer
	for _, r := range x.DoubleSigners {
		// if the double signer is nil
		if r == nil || r.Heights == nil || r.Id == nil {
			// exit with error
			return ErrInvalidDoubleSigner()
		}
	}
	// exit
	return
}

// Equals() compares two SlashRecipients for equality
func (x *SlashRecipients) Equals(y *SlashRecipients) bool {
	// if the slash recipients are both empty
	if x == nil && y == nil {
		// exit with 'equal'
		return true
	}
	// if either of the slash recipients are not empty
	if x == nil || y == nil {
		// exit with 'unequal'
		return false
	}
	// if the double signers differ in length
	if len(x.DoubleSigners) != len(y.DoubleSigners) {
		// exit with 'unequal'
		return false
	}
	// for each double signer
	for i, ds := range x.DoubleSigners {
		// if the id is not equal
		if !bytes.Equal(ds.Id, y.DoubleSigners[i].Id) {
			// exit with 'unequal'
			return false
		}
		// if the heights are not equal
		if !slices.Equal(ds.Heights, y.DoubleSigners[i].Heights) {
			// exit with 'unequal'
			return false
		}
	}
	// exit with 'equal'
	return true
}

// jsonSlashRecipients is the SlashRecipients implementation of json.Marshaller and json.Unmarshaler
type jsonSlashRecipients struct {
	// the actors the bft quorum agreed were double signers
	DoubleSigners []*DoubleSigner `json:"doubleSigners,omitempty"`
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x SlashRecipients) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSlashRecipients{DoubleSigners: x.DoubleSigners})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *SlashRecipients) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new object reference
	j := new(jsonSlashRecipients)
	// populate the object reference using the json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// set the underlying object using the json obj ref
	*x = SlashRecipients{
		DoubleSigners: j.DoubleSigners,
	}
	// exit
	return
}

// ORDERS CODE BELOW

// CheckBasic() performs stateless validation on an Orders object
func (x *Orders) CheckBasic() (err ErrorI) {
	// if the orders are empty
	if x == nil {
		// exit with no error
		return
	}
	// for each lock order
	for _, lock := range x.LockOrders {
		// if the lock order is empty
		if lock == nil {
			// exit with empty error
			return ErrNilLockOrder()
		}
		// ensure the sending address actually has some bytes
		if len(lock.BuyerSendAddress) == 0 {
			// exit with address error
			return ErrInvalidBuyerSendAddress()
		}
		// ensure the receive address is exactly 20 bytes
		if len(lock.BuyerReceiveAddress) != crypto.AddressSize {
			// exit with address error
			return ErrInvalidBuyerReceiveAddress()
		}
	}
	// ensure no duplicates in the resets
	deDuplicator := NewDeDuplicator[uint64]()
	// for each reset order
	for _, reset := range x.ResetOrders {
		// if a duplicate found
		if deDuplicator.Found(reset) {
			// exit with the duplicate reset order
			return ErrDuplicateResetOrder()
		}
	}
	// ensure no duplicates in the closes
	deDuplicator = NewDeDuplicator[uint64]()
	// for each close order
	for _, reset := range x.CloseOrders {
		// if a duplicate found
		if deDuplicator.Found(reset) {
			// exit with the duplicate close order
			return ErrDuplicateCloseOrder()
		}
	}
	// exit
	return
}

// Equals() compares two Orders for equality
func (x *Orders) Equals(y *Orders) bool {
	// if both of the orders are empty
	if x == nil && y == nil {
		// exit with 'equal'
		return true
	}
	// if either of the orders are empty
	if x == nil || y == nil {
		// exit with 'unequal'
		return false
	}
	// if the close orders lists are not equal
	if !slices.Equal(x.CloseOrders, y.CloseOrders) {
		// exit with 'unequal'
		return false
	}
	// if the reset orders lists are not equal
	if !slices.Equal(x.ResetOrders, y.ResetOrders) {
		// exit with 'unequal'
		return false
	}
	// if the lock orders lists are not equal size
	if len(x.LockOrders) != len(y.LockOrders) {
		// exit with 'unequal'
		return false
	}
	// for each lock order
	for i, lockOrder := range x.LockOrders {
		// if the individual lock orders are unequal
		if !lockOrder.Equals(y.LockOrders[i]) {
			// exit with 'unequal'
			return false
		}
	}
	// exit with 'equal'
	return true
}

// Equals() compares two LockOrders for equality
func (x *LockOrder) Equals(y *LockOrder) bool {
	// if both the lock orders are empty
	if x == nil && y == nil {
		// exit with 'equal'
		return true
	}
	// if either of the lock orders are empty
	if x == nil || y == nil {
		// exit with 'unequal'
		return false
	}
	// if the order buyers receive addresses are not the same
	if !bytes.Equal(x.BuyerReceiveAddress, y.BuyerReceiveAddress) {
		// exit with 'unequal'
		return false
	}
	// if the order buyers send addresses are not the same
	if !bytes.Equal(x.BuyerSendAddress, y.BuyerSendAddress) {
		// exit with 'unequal'
		return false
	}
	// if the order ids are not the same
	if x.OrderId != y.OrderId {
		// exit with 'unequal'
		return false
	}
	// exit with the final equality check
	return x.BuyerChainDeadline == y.BuyerChainDeadline
}

// lockOrderJSON implements the json.Marshaller & json.Unmarshaler interfaces for LockOrder
type lockOrderJSON struct {
	// order_id: is the number id that is unique to this committee to identify the order
	OrderId uint64 `json:"order_id,omitempty"`
	// buyers_send_address: the Canopy address where the tokens may be received
	BuyersSendAddress HexBytes `json:"buyers_send_address,omitempty"`
	// buyer_receive_address: the Canopy address where the tokens may be received
	BuyerReceiveAddress HexBytes `json:"buyer_receive_address,omitempty"`
	// buyer_chain_deadline: the 'counter asset' chain height at which the buyer must send the 'counter asset' by
	// or the 'intent to buy' will be voided
	BuyerChainDeadline uint64 `json:"buyer_chain_deadline,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface for LockOrder
func (x LockOrder) MarshalJSON() ([]byte, error) {
	// convert the lock order to json bytes using the json object
	return json.Marshal(&lockOrderJSON{
		OrderId:             x.OrderId,
		BuyersSendAddress:   x.BuyerSendAddress,
		BuyerReceiveAddress: x.BuyerReceiveAddress,
		BuyerChainDeadline:  x.BuyerChainDeadline,
	})
}

// UnmarshalJSON() implements the json.Unmarshaler interface for LockOrder
func (x *LockOrder) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non nil result
	j := new(lockOrderJSON)
	// populate the json object ref with json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		// exit with error
		return
	}
	// populate the underlying structure using the json object
	*x = LockOrder{
		OrderId:             j.OrderId,
		BuyerReceiveAddress: j.BuyerReceiveAddress,
		BuyerSendAddress:    j.BuyersSendAddress,
		BuyerChainDeadline:  j.BuyerChainDeadline,
	}
	// exit
	return
}

// CHECKPOINT CODE BELOW

// CheckBasic() performs stateless validation on a Checkpoint object
func (x *Checkpoint) CheckBasic() (err ErrorI) {
	// if the checkpoint is empty
	if x == nil {
		// exit without error
		return
	}
	// if the block hash size is larger than 100
	if len(x.BlockHash) > 100 {
		// exit with error
		return ErrInvalidBlockHash()
	}
	// exit
	return
}

// Equals() compares two Checkpoints for equality
func (x *Checkpoint) Equals(y *Checkpoint) bool {
	// if both of the checkpoints are empty
	if x == nil && y == nil {
		// exit with 'equal'
		return true
	}
	// if either of the checkpoints are empty
	if x == nil || y == nil {
		// exit with 'unequal'
		return false
	}
	// if the block hashes are not equal
	if !bytes.Equal(x.BlockHash, y.BlockHash) {
		// exit with 'unequal'
		return false
	}
	// exit with the final equality check
	return x.Height == y.Height
}

// Combine() merges the Reward Recipients' Payment Percents of the current Proposal with those of another Proposal
// such that the Payment Percentages may be equally weighted when performing reward distribution calculations
// NOTE: merging percents will exceed 100% over multiple samples, but are normalized using the NumberOfSamples field
// NOTE: if the 'chainId' designation doesn't match the 'self' chainId, the payment percent is ignored
func (x *CommitteeData) Combine(data *CommitteeData, chainId uint64) (err ErrorI) {
	// safety check to ensure the data is not null
	if data == nil {
		// exit without error
		return
	}
	// for each payment percent
	for _, p := range data.PaymentPercents {
		// ignore any payment percent not designated for our chain id
		if p.ChainId == chainId {
			// combine the percents with the existing stubs
			// percents can/will exceed 100 but are re-normalized using NumberOfSamples later
			x.addPercents(p.Address, p.Percent, chainId)
		}
	}
	// new Proposal purposefully overwrites the Block and Meta of the current Proposal
	// this is to ensure both Proposals have the latest Block and Meta information
	// in the case where the caller uses a pattern where there may be a stale Block/Meta
	*x = CommitteeData{
		PaymentPercents:        x.PaymentPercents,           // maintain the payment percents
		NumberOfSamples:        x.NumberOfSamples + 1,       // add to the number of samples
		ChainId:                data.ChainId,                // (defensively) update the chain id
		LastRootHeightUpdated:  data.LastRootHeightUpdated,  // update the root height
		LastChainHeightUpdated: data.LastChainHeightUpdated, // update the chain height
	}
	// exit
	return
}

// addPercents() is a helper function that adds reward distribution percents on behalf of an address
func (x *CommitteeData) addPercents(address []byte, percent, chainId uint64) {
	// check to see if the address already exists
	for i, p := range x.PaymentPercents {
		// if already exists
		if bytes.Equal(address, p.Address) {
			// simply add the percent to the previous
			x.PaymentPercents[i].Percent += percent
			// exit
			return
		}
	}
	// if the address doesn't already exist, append a sample to PaymentPercents
	x.PaymentPercents = append(x.PaymentPercents, &PaymentPercents{
		Address: address,
		Percent: percent,
		ChainId: chainId,
	})
}

// jsonDoubleSigner implements the json.Marshaller and json.Unmarshaler interfaces for double signers
type jsonDoubleSigner struct {
	// id: the cryptographic identifier of the malicious actor
	Id HexBytes `json:"id,omitempty"`
	// heights: the list of heights when the infractions occurred
	Heights []uint64 `json:"heights,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface for double signers
func (x DoubleSigner) MarshalJSON() ([]byte, error) {
	// convert the double signers to json bytes using a json object
	return MarshalJSON(jsonDoubleSigner{Id: x.Id, Heights: x.Heights})
}

// MarshalJSON() implements the json.Unmarshaler interface for double signers
func (x *DoubleSigner) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non nil result
	j := new(jsonDoubleSigner)
	// populate the object ref using json bytes
	if err = json.Unmarshal(jsonBytes, j); err != nil {
		return
	}
	// populate the underlying struct using the json object
	*x = DoubleSigner{Id: j.Id, Heights: j.Heights}
	// exit
	return
}

func init() {
	// calculate the MaxBlockHeader programmatically
	maxBlockHeader, err := Marshal(&BlockHeader{
		Height:             math.MaxUint64,
		Hash:               crypto.MaxHash,
		NetworkId:          math.MaxInt8,
		Time:               math.MaxUint32,
		NumTxs:             math.MaxUint64,
		TotalTxs:           math.MaxUint64,
		TotalVdfIterations: math.MaxUint64,
		LastBlockHash:      crypto.MaxHash,
		StateRoot:          crypto.MaxHash,
		TransactionRoot:    crypto.MaxHash,
		ValidatorRoot:      crypto.MaxHash,
		NextValidatorRoot:  crypto.MaxHash,
		ProposerAddress:    crypto.MaxHash,
		Vdf: &crypto.VDF{
			Proof:      bytes.Repeat([]byte("F"), 528),
			Output:     bytes.Repeat([]byte("F"), 528),
			Iterations: math.MaxUint64,
		},
		LastQuorumCertificate: &QuorumCertificate{
			Header: &View{
				NetworkId:  math.MaxInt8,
				ChainId:    math.MaxUint64,
				Height:     math.MaxUint64,
				RootHeight: math.MaxUint64,
				Round:      math.MaxUint64,
				Phase:      math.MaxInt8,
			},
			ResultsHash: crypto.MaxHash,
			BlockHash:   crypto.MaxHash,
			ProposerKey: bytes.Repeat([]byte("F"), crypto.BLS12381PubKeySize),
			Signature: &AggregateSignature{
				Signature: bytes.Repeat([]byte("F"), crypto.BLS12381SignatureSize),
				Bitmap:    bytes.Repeat([]byte("F"), crypto.MaxBitmapSize(100)),
			},
		},
	})
	// if an error occurred during the byte conversion or calculation
	if err != nil {
		// fatal exit program
		panic(err)
	}
	// set the max block header
	MaxBlockHeaderSize = uint64(len(maxBlockHeader))
	// do a sanity check of the expected size to make developers aware if something changed
	if MaxBlockHeaderSize != ExpectedMaxBlockHeaderSize {
		// fatal exit and descriptive warning
		panic(fmt.Sprintf("Max_Header_Size changed from %d to %d; This is a consensus breaking change", ExpectedMaxBlockHeaderSize, MaxBlockHeaderSize))
	}
}
