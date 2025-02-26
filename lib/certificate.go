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

const (
	GlobalMaxBlockSize         = int(32 * units.MB)
	ExpectedMaxBlockHeaderSize = 1640 // ensures developers are aware of a change to the header size (which is a consensus breaking change)
)

var MaxBlockHeaderSize uint64

func init() {
	maxBlockHeader, err := Marshal(&BlockHeader{
		Height:             math.MaxUint64,
		Hash:               crypto.MaxHash,
		NetworkId:          math.MaxInt8,
		Time:               math.MaxUint32,
		NumTxs:             math.MaxUint64,
		TotalTxs:           math.MaxUint64,
		TotalVdfIterations: math.MaxUint64,
		LastBlockHash:      crypto.MaxHash,
		StateRoot:          crypto.MaxHash[:20],
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
	if err != nil {
		panic(err)
	}
	MaxBlockHeaderSize = uint64(len(maxBlockHeader))
	if MaxBlockHeaderSize != ExpectedMaxBlockHeaderSize {
		panic(fmt.Sprintf("Max_Header_Size changed from %d to %d; This is a consensus breaking change", ExpectedMaxBlockHeaderSize, MaxBlockHeaderSize))
	}
}

// QUORUM CERTIFICATE CODE BELOW

// SignBytes() returns the canonical byte representation used to digitally sign the bytes of the structure
func (x *QuorumCertificate) SignBytes() (signBytes []byte) {
	if x.Header != nil && x.Header.Phase == Phase_ELECTION_VOTE {
		bz, _ := Marshal(&QuorumCertificate{Header: x.Header, ProposerKey: x.ProposerKey})
		return bz
	}
	// temp variables to save values
	results, block, aggregateSignature := x.Results, x.Block, x.Signature
	// remove the values from the struct
	x.Results, x.Block, x.Signature = nil, nil, nil
	// convert the structure into the sign bytes
	signBytes, _ = Marshal(x)
	// add back the removed values
	x.Results, x.Block, x.Signature = results, block, aggregateSignature
	return
}

// CheckBasic() performs 'sanity' checks on the Quorum Certificate structure
// height may be optionally passed for View checking
func (x *QuorumCertificate) CheckBasic() ErrorI {
	// a valid QC must have either the proposal hash or the proposer key set
	if x == nil || (x.ResultsHash == nil && x.ProposerKey == nil) {
		return ErrEmptyQuorumCertificate()
	}
	// sanity check the view of the QC
	if err := x.Header.CheckBasic(); err != nil {
		return err
	}
	// is QC with result (AFTER ELECTION)
	if x.ResultsHash != nil {
		// sanity check the hashes
		if len(x.BlockHash) != crypto.HashSize {
			return ErrInvalidBlockHash()
		}
		if len(x.ResultsHash) != crypto.HashSize {
			return ErrInvalidResultsHash()
		}
		// results may be omitted in certain cases like for integrated blockchain block storage
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
			blk := new(Block)
			// convert the block bytes into a block
			hash, err := blk.BytesToBlock(x.Block)
			if err != nil {
				return err
			}
			// check the block hash
			if !bytes.Equal(x.BlockHash, hash) {
				return ErrMismatchQCBlockHash()
			}
			blockSize := len(x.Block)
			// global max block size enforcement
			if blockSize > GlobalMaxBlockSize {
				return ErrExpectedMaxBlockSize()
			}
		}
	} else { // is QC with proposer key (ELECTION)
		if len(x.ProposerKey) != crypto.BLS12381PubKeySize {
			return ErrInvalidSigner()
		}
		if len(x.ResultsHash) != 0 || x.Results != nil {
			return ErrMismatchResultsHash()
		}
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
	if err := x.CheckBasic(); err != nil {
		return false, err
	}
	if err := x.Header.Check(view, enforceHeights); err != nil {
		return false, err
	}
	if x.Block != nil {
		// max block size enforcement
		if len(x.Block) > maxBlockSize {
			return false, ErrExpectedMaxBlockSize()
		}
	}
	return x.Signature.Check(x, vs)
}

// CheckHighQC() performs additional validation on the special `HighQC` (justify unlock QC)
func (x *QuorumCertificate) CheckHighQC(maxBlockSize int, view *View, lastRootHeightUpdated uint64, vs ValidatorSet) ErrorI {
	isPartialQC, err := x.Check(vs, maxBlockSize, view, false)
	if err != nil {
		return err
	}
	// `highQCs` can't justify an unlock without +2/3 majority
	if isPartialQC {
		return ErrNoMaj23()
	}
	// invalid 'historical committee', must be before the last committee height saved in the state
	// if not, there is a potential for a long range attack
	if lastRootHeightUpdated > x.Header.RootHeight {
		return ErrWrongRootHeight()
	}
	// enforce same target height
	if x.Header.Height != view.Height {
		return ErrWrongHeight()
	}
	// a valid HighQC must have the phase must be PRECOMMIT_VOTE
	// as that's the phase where replicas 'Lock'
	if x.Header.Phase != Phase_PROPOSE_VOTE {
		return ErrWrongPhase()
	}
	// the block hash nor results hash cannot be nil for a HighQC
	// as it's after the election phase
	if x.BlockHash == nil || x.ResultsHash == nil {
		return ErrNilBlock()
	}
	return nil
}

// GetNonSigners() returns the public keys and the percentage (of voting power out of total) of those who did not sign the QC
func (x *QuorumCertificate) GetNonSigners(vs *ConsensusValidators) (nonSignerPubKeys [][]byte, nonSignerPercent int, err ErrorI) {
	if x == nil || x.Signature == nil {
		return nil, 0, ErrEmptyQuorumCertificate()
	}
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
func (x *QuorumCertificate) UnmarshalJSON(b []byte) (err error) {
	var j jsonQC
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = QuorumCertificate{
		Header:      j.Header,
		Results:     j.Results,
		ResultsHash: j.ResultsHash,
		Block:       j.Block,
		BlockHash:   j.BlockHash,
		ProposerKey: j.ProposerKey,
		Signature:   j.Signature,
	}
	return nil
}

// 	A CertificateResult contains Canopy information for what happens to stakeholders as a result of the BFT

// CERTIFICATE RESULT CODE BELOW

// CheckBasic() provides basic 'sanity' checks on the CertificateResult structure
func (x *CertificateResult) CheckBasic() ErrorI {
	if x == nil {
		return ErrNilCertResults()
	}
	if err := x.RewardRecipients.CheckBasic(); err != nil {
		return err
	}
	if err := x.SlashRecipients.CheckBasic(); err != nil {
		return err
	}
	if err := x.Orders.CheckBasic(); err != nil {
		return err
	}
	return x.Checkpoint.CheckBasic()
}

// Equals() compares two certificate results to ensure equality
func (x *CertificateResult) Equals(y *CertificateResult) bool {
	if x == nil || y == nil {
		return false
	}
	if !x.RewardRecipients.Equals(y.RewardRecipients) {
		return false
	}
	if !x.SlashRecipients.Equals(y.SlashRecipients) {
		return false
	}
	if !x.Orders.Equals(y.Orders) {
		return false
	}
	if !x.Checkpoint.Equals(y.Checkpoint) {
		return false
	}
	return x.Retired == y.Retired
}

// Hash() returns the cryptographic hash of the canonical Sign Bytes of the CertificateResult
func (x *CertificateResult) Hash() []byte {
	bz, _ := Marshal(x)
	return crypto.Hash(bz)
}

// AwardPercents() adds reward distribution PaymentPercent samples to the CertificateResult structure
// NOTE: percents should not exceed 100% in a single sample
func (x *CertificateResult) AwardPercents(percents []*PaymentPercents) ErrorI {
	x.RewardRecipients.NumberOfSamples++
	for _, ep := range percents {
		x.addPercents(ep.Address, ep.Percent)
	}
	return nil
}

// addPercents() is a helper function that adds reward distribution percents on behalf of an address
func (x *CertificateResult) addPercents(address []byte, percent uint64) {
	// check to see if the address already has samples
	for i, ep := range x.RewardRecipients.PaymentPercents {
		if bytes.Equal(address, ep.Address) {
			x.RewardRecipients.PaymentPercents[i].Percent += ep.Percent
			return
		}
	}
	// if not, append a sample to PaymentPercents
	x.RewardRecipients.PaymentPercents = append(x.RewardRecipients.PaymentPercents, &PaymentPercents{
		Address: address,
		Percent: percent,
	})
}

// REWARD RECIPIENT CODE BELOW

// CheckBasic() performs a basic 'sanity check' on the structure
func (x *RewardRecipients) CheckBasic() (err ErrorI) {
	if x == nil {
		return ErrNilRewardRecipients()
	}
	// validate the number of recipients
	paymentRecipientCount := len(x.PaymentPercents)
	// ensure not zero or bigger than 25
	if paymentRecipientCount == 0 || paymentRecipientCount > 25 {
		return ErrPaymentRecipientsCount()
	}
	// validate the percents add up to 100 (or less)
	totalPercent := uint64(0)
	for _, pp := range x.PaymentPercents {
		// ensure each percent isn't nil
		if pp == nil {
			return ErrInvalidPercentAllocation()
		}
		// ensure each percent address is the right size
		if len(pp.Address) != crypto.AddressSize {
			return ErrInvalidAddress()
		}
		// ensure each percent isn't 0
		if pp.Percent == 0 {
			return ErrInvalidPercentAllocation()
		}
		// add to total
		totalPercent += pp.Percent
		// ensure the percent doesn't exceed 100
		if totalPercent > 100 {
			return ErrInvalidPercentAllocation()
		}
	}
	return
}

// Equals() compares two RewardRecipients for equality
func (x *RewardRecipients) Equals(y *RewardRecipients) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	if len(x.PaymentPercents) != len(y.PaymentPercents) {
		return false
	}
	for i, pp := range x.PaymentPercents {
		if !bytes.Equal(pp.Address, y.PaymentPercents[i].Address) {
			return false
		}
		if pp.Percent != y.PaymentPercents[i].Percent {
			return false
		}
	}
	return x.NumberOfSamples == y.NumberOfSamples
}

// jsonRewardRecipients is the RewardRecipients implementation of json.Marshaller and json.Unmarshaler
type jsonRewardRecipients struct {
	PaymentPercents []*PaymentPercents `json:"paymentPercents,omitempty"` // recipients of the block reward by percentage
	NumberOfSamples uint64             `json:"numberOfSamples,omitempty"`
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *RewardRecipients) UnmarshalJSON(i []byte) error {
	j := new(jsonRewardRecipients)
	if err := json.Unmarshal(i, j); err != nil {
		return err
	}
	*x = RewardRecipients{
		PaymentPercents: j.PaymentPercents,
		NumberOfSamples: j.NumberOfSamples,
	}
	return nil
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x *RewardRecipients) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonRewardRecipients{
		PaymentPercents: x.PaymentPercents,
		NumberOfSamples: x.NumberOfSamples,
	})
}

// PAYMENT PERCENTS CODE BELOW

// paymentPercents is the PaymentPercents implementation of json.Marshaller and json.Unmarshaler
type paymentPercents struct {
	Address  HexBytes `json:"address"`
	Percents uint64   `json:"percents"`
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x *PaymentPercents) MarshalJSON() ([]byte, error) {
	return json.Marshal(paymentPercents{
		Address:  x.Address,
		Percents: x.Percent,
	})
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *PaymentPercents) UnmarshalJSON(b []byte) error {
	var ep paymentPercents
	if err := json.Unmarshal(b, &ep); err != nil {
		return err
	}
	x.Address, x.Percent = ep.Address, ep.Percents
	return nil
}

// SLASH RECIPIENTS CODE BELOW

// CheckBasic() validates the ProposalMeta structure
func (x *SlashRecipients) CheckBasic() ErrorI {
	if x != nil {
		for _, r := range x.DoubleSigners {
			if r == nil {
				return ErrInvalidDoubleSigner()
			}
		}
	}
	return nil
}

// Equals() compares two SlashRecipients for equality
func (x *SlashRecipients) Equals(y *SlashRecipients) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	if len(x.DoubleSigners) != len(y.DoubleSigners) {
		return false
	}
	for i, ds := range x.DoubleSigners {
		if !bytes.Equal(ds.Id, y.DoubleSigners[i].Id) {
			return false
		}
		if !slices.Equal(ds.Heights, y.DoubleSigners[i].Heights) {
			return false
		}
	}
	return true
}

// jsonSlashRecipients is the SlashRecipients implementation of json.Marshaller and json.Unmarshaler
type jsonSlashRecipients struct {
	DoubleSigners []*DoubleSigner `json:"doubleSigners,omitempty"` // who did the bft decide was a double signer
}

// UnmarshalJSON() satisfies the json.Unmarshaler interface
func (x *SlashRecipients) UnmarshalJSON(i []byte) error {
	j := new(jsonSlashRecipients)
	if err := json.Unmarshal(i, j); err != nil {
		return err
	}
	*x = SlashRecipients{
		DoubleSigners: j.DoubleSigners,
	}
	return nil
}

// MarshalJSON() satisfies the json.Marshaller interface
func (x *SlashRecipients) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSlashRecipients{DoubleSigners: x.DoubleSigners})
}

// ORDERS CODE BELOW

// CheckBasic() performs stateless validation on an Orders object
func (x *Orders) CheckBasic() ErrorI {
	if x == nil {
		return nil
	}
	// check the lock orders
	for _, lock := range x.LockOrders {
		if lock == nil {
			return ErrNilLockOrder()
		}
		if lock.LockerReceiveAddress == nil {
			return ErrInvalidLockerReceiveAddress()
		}
	}
	return nil
}

// Equals() compares two Orders for equality
func (x *Orders) Equals(y *Orders) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	if !slices.Equal(x.CloseOrders, y.CloseOrders) {
		return false
	}
	if !slices.Equal(x.ResetOrders, y.ResetOrders) {
		return false
	}
	if len(x.LockOrders) != len(y.LockOrders) {
		return false
	}
	for i, o := range x.LockOrders {
		if !o.Equals(y.LockOrders[i]) {
			return false
		}
	}
	return true
}

// Equals() compares two LockOrders for equality
func (x *LockOrder) Equals(y *LockOrder) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	if !bytes.Equal(x.LockerReceiveAddress, y.LockerReceiveAddress) {
		return false
	}
	if !bytes.Equal(x.LockerSendAddress, y.LockerSendAddress) {
		return false
	}
	if x.OrderId != y.OrderId {
		return false
	}
	return x.LockerChainDeadline == y.LockerChainDeadline
}

// lockOrderJSON implements the json.Marshaller & json.Unmarshaler interfaces for LockOrder
type lockOrderJSON struct {
	// order_id: is the number id that is unique to this committee to identify the order
	OrderId uint64 `json:"order_id,omitempty"`
	// lockers_send_address: the Canopy address where the tokens may be received
	LockersSendAddress HexBytes `json:"lockers_send_address,omitempty"`
	// locker_receive_address: the Canopy address where the tokens may be received
	LockerReceiveAddress HexBytes `json:"locker_receive_address,omitempty"`
	// locker_chain_deadline: the 'counter asset' chain height at which the locker must send the 'counter asset' by
	// or the 'intent to lock' will be voided
	LockerChainDeadline uint64 `json:"locker_chain_deadline,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface for LockOrder
func (x LockOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(&lockOrderJSON{
		OrderId:             x.OrderId,
		LockersSendAddress:   x.LockerSendAddress,
		LockerReceiveAddress: x.LockerReceiveAddress,
		LockerChainDeadline:  x.LockerChainDeadline,
	})
}

// UnmarshalJSON() implements the json.Unmarshaler interface for LockOrder
func (x *LockOrder) UnmarshalJSON(b []byte) (err error) {
	j := new(lockOrderJSON)
	if err = json.Unmarshal(b, j); err != nil {
		return
	}
	*x = LockOrder{
		OrderId:             j.OrderId,
		LockerReceiveAddress: j.LockerReceiveAddress,
		LockerSendAddress:    j.LockersSendAddress,
		LockerChainDeadline:  j.LockerChainDeadline,
	}
	return
}

// CHECKPOINT CODE BELOW

// CheckBasic() performs stateless validation on a Checkpoint object
func (x *Checkpoint) CheckBasic() ErrorI {
	if x == nil {
		return nil
	}
	if len(x.BlockHash) > 100 {
		return ErrInvalidBlockHash()
	}
	return nil
}

// Equals() compares two Checkpoints for equality
func (x *Checkpoint) Equals(y *Checkpoint) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	if !bytes.Equal(x.BlockHash, y.BlockHash) {
		return false
	}
	return x.Height == y.Height
}

// Combine() merges the Reward Recipients' Payment Percents of the current Proposal with those of another Proposal
// such that the Payment Percentages may be equally weighted when performing reward distribution calculations
// NOTE: percents will exceed 100% over multiple samples, but are normalized using the NumberOfSamples field
func (x *CommitteeData) Combine(data *CommitteeData) ErrorI {
	// safety check to ensure the data is not null
	if data == nil {
		return nil
	}
	// for each payment percent
	for _, p := range data.PaymentPercents {
		// combine the percents with the existing stubs
		// percents can/will exceed 100 but are re-normalized using NumberOfSamples later
		x.addPercents(p.Address, p.Percent)
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
	return nil
}

// addPercents() is a helper function that adds reward distribution percents on behalf of an address
func (x *CommitteeData) addPercents(address []byte, percent uint64) {
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
	return MarshalJSON(jsonDoubleSigner{Id: x.Id, Heights: x.Heights})
}

// MarshalJSON() implements the json.Unmarshaler interface for double signers
func (x *DoubleSigner) UnmarshalJSON(bz []byte) (err error) {
	j := new(jsonDoubleSigner)
	if err = json.Unmarshal(bz, j); err != nil {
		return
	}
	*x = DoubleSigner{Id: j.Id, Heights: j.Heights}
	return
}
