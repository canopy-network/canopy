package bft

import (
	"bytes"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
)

// HandleMessage handles and routes incoming consensus message from a Validator peer
func (b *BFT) HandleMessage(message proto.Message) lib.ErrorI {
	// ensure is a valid `Consensus Message` type
	switch msg := message.(type) {
	case *Message:
		// capture the paramters needed to validate the message
		params, err := b.GetValidateMessageParams(msg)
		if err != nil {
			return err
		}
		switch {
		case msg.IsReplicaMessage() || msg.IsPacemakerMessage(): // consensus message from a Replica
			// validate the Replica message
			if err = b.CheckReplicaMessage(msg, params); err != nil {
				b.log.Errorf("Received invalid vote from %s", lib.BytesToString(msg.Signature.PublicKey))
				return err
			}
			b.log.Debugf("Received %s message from replica: %s", msg.Qc.Header.ToString(), lib.BytesToTruncatedString(msg.Signature.PublicKey))
			// store pacemaker messages separate from 'votes'
			if msg.IsPacemakerMessage() {
				return b.AddPacemakerMessage(msg)
			}
			// store Vote
			return b.AddVote(msg)
		case msg.IsProposerMessage(): // consensus message from the Leader
			// validate the Leader message
			partialQC, err := b.CheckProposerMessage(msg, params)
			if err != nil {
				b.log.Errorf("Received invalid proposal from %s", lib.BytesToString(msg.Signature.PublicKey))
				return err
			}
			b.log.Debugf("Received %s message from proposer: %s", msg.Header.ToString(), lib.BytesToTruncatedString(msg.Signature.PublicKey))
			// store partial QCs as they may be indicative of byzantine behavior
			if partialQC {
				b.log.Errorf("Received partial QC from proposer %s", lib.BytesToTruncatedString(msg.Signature.PublicKey))
				return b.AddPartialQC(msg)
			}
			// store Proposal
			return b.AddProposal(msg)
		}
	}
	return ErrUnknownConsensusMsg(message)
}

// CheckProposerMessage() validates an inbound message from the Leader Validator
func (b *BFT) CheckProposerMessage(x *Message, p *validateMessageParams) (isPartialQC bool, err lib.ErrorI) {
	// basic sanity checks on the message
	if err = x.checkBasic(p.view); err != nil {
		return false, err
	}
	// ELECTION
	if x.Header.Phase == Election {
		// validate target height
		if x.Header.Height != p.height {
			return false, lib.ErrWrongCertHeight(x.Header.Height, p.height)
		}
		// sanity check the VRF
		if err = checkSignatureBasic(x.Vrf); err != nil {
			return false, err
		}
		// ensure the VRF is for the sender
		if !bytes.Equal(x.Signature.PublicKey, x.Vrf.PublicKey) {
			return false, ErrMismatchPublicKeys()
		}
		return
	}
	// PROPOSE, PRECOMMIT, COMMIT
	var vals = p.vals
	// any message from the Leader after the ELECTION phase contains a justification (Quorum Certificate)
	// sanity check the Quorum Certificate
	if err = x.Qc.CheckBasic(); err != nil {
		return
	}
	// if an unexpected root height
	if x.Qc.Header.RootHeight != p.rootHeight {
		// load the proper committee
		b.Controller.Lock()
		vals, err = b.LoadCommittee(b.LoadRootChainId(x.Qc.Header.Height), x.Qc.Header.RootHeight) // REPLICAS: CAPTURE PARTIAL QCs FROM ANY HEIGHT
		b.Controller.Unlock()
		if err != nil {
			return false, err
		}
	}
	// validate the Quorum Certificate
	isPartialQC, err = x.Qc.Check(vals, lib.GlobalMaxBlockSize, p.view, false)
	if err != nil {
		return
	}
	// if it doesn't have +2/3 majority
	if isPartialQC {
		return
	}
	// validate header height, qc height, and committee height
	// NOTE: these height checks are correct even when sending a highQC as the header is updated when using a highQC
	if x.Header.Height != p.height {
		return false, lib.ErrWrongCertHeight(x.Header.Height, p.height)
	}
	if x.Qc.Header.Height < p.cHeightUpdated {
		return false, lib.ErrInvalidQCCommitteeHeight()
	}
	if x.Header.Phase == Propose {
		// ensure the sender is justified as the proposer
		if !bytes.Equal(x.Qc.ProposerKey, x.Signature.PublicKey) {
			return false, lib.ErrInvalidSigner()
		}
		// ensure the block isn't nil
		if x.Qc.Block == nil {
			return false, lib.ErrNilBlock()
		}
		// ensure the results aren't nil
		if x.Qc.Results == nil {
			return false, lib.ErrNilCertResults()
		}
	} else {
		// in PRECOMMIT or COMMIT phase
		if p.blockHash == nil || p.resultsHash == nil {
			return false, lib.ErrNoSavedBlockOrResults()
		}
		// PROPOSE-VOTE and PRECOMMIT-VOTE Replica message
		if !bytes.Equal(x.Qc.BlockHash, p.blockHash) {
			return false, lib.ErrMismatchConsBlockHash()
		}
		if !bytes.Equal(x.Qc.ResultsHash, p.resultsHash) {
			return false, lib.ErrMismatchResultsHash()
		}
	}
	return
}

// CheckReplicaMessage() validates an inbound message from a Replica Validator
func (b *BFT) CheckReplicaMessage(x *Message, params *validateMessageParams) lib.ErrorI {
	// NOTE: x.CheckBasic() but without the 'header' check - Replicas always use the QC so their communications may be aggregable
	if x == nil {
		return ErrEmptyMessage()
	}
	// ensure the Quorum certificate isn't nil as checkSignature uses the sign bytes
	if x.Qc == nil {
		return lib.ErrEmptyQuorumCertificate()
	}
	// check signature using the message sign bytes
	if err := checkSignature(x.Signature, x); err != nil {
		return err
	}
	// validate the header of the Quorum  Certificate
	if err := x.Qc.Header.Check(params.view, true); err != nil {
		return err
	}
	// the validation is done for Pacemaker message types
	if x.IsPacemakerMessage() {
		return nil
	}
	if x.Qc.Header.Phase == ElectionVote {
		// ELECTION-VOTE Replica message
		if len(x.Qc.ProposerKey) != crypto.BLS12381PubKeySize {
			return lib.ErrInvalidSigner()
		}
	} else {
		// PROPOSE-VOTE and PRECOMMIT-VOTE Replica message
		if params.blockHash == nil {
			if !bytes.Equal(x.Qc.BlockHash, b.BlockToHash(x.Qc.Block)) {
				return lib.ErrMismatchQCBlockHash()
			}
		} else {
			if !bytes.Equal(x.Qc.BlockHash, params.blockHash) {
				return lib.ErrMismatchConsBlockHash()
			}
		}
		if !bytes.Equal(x.Qc.ResultsHash, params.resultsHash) {
			return lib.ErrMismatchResultsHash()
		}
	}
	return nil
}

// SignBytes() returns the canonical bytes representation of a Message used as input for a digital signature
func (x *Message) SignBytes() (signBytes []byte) {
	switch {
	case x.IsProposerMessage():
		// create a clone of the Message object without the QC and signature
		msg := &Message{
			Header:                 x.Header,
			Vrf:                    x.Vrf,
			HighQc:                 x.HighQc,
			LastDoubleSignEvidence: x.LastDoubleSignEvidence,
		}
		// phase ELECTION doesn't have a QC, but also
		// the sign bytes function is used prior to the
		// QC.checkBasic() - thus this is good defensive coding
		if x.Qc != nil {
			msg.Qc = &QC{
				Header:      x.Qc.Header,
				BlockHash:   x.Qc.BlockHash,   // omit the block, but lock in with the block hash
				ResultsHash: x.Qc.ResultsHash, // omit the results, but lock in with the results hash
				ProposerKey: x.Qc.ProposerKey,
				Signature:   x.Qc.Signature,
			}
		}
		signBytes, _ = lib.Marshal(msg)
	case x.IsReplicaMessage():
		// the sign bytes function is used prior to the
		// QC.checkBasic() - thus this is good defensive coding
		if x.Qc == nil {
			return nil
		}
		return (&QC{
			Header:      x.Qc.Header,
			BlockHash:   x.Qc.BlockHash,
			ResultsHash: x.Qc.ResultsHash,
			ProposerKey: x.Qc.ProposerKey,
		}).SignBytes()
	case x.IsPacemakerMessage():
		signBytes, _ = lib.Marshal(&Message{Header: x.Header})
	}
	return
}

// Sign() is a convenience method for performing a digital signature with this message using a Private Key
// and fills the 'signature' field of the Message
func (x *Message) Sign(privateKey crypto.PrivateKeyI) lib.ErrorI {
	x.Signature = new(lib.Signature)
	x.Signature.PublicKey = privateKey.PublicKey().Bytes()
	x.Signature.Signature = privateKey.Sign(x.SignBytes())
	return nil
}

// IsReplicaMessage() determines if the message should originate from a Validator acting as a Replica (voter)
func (x *Message) IsReplicaMessage() bool {
	if x.Qc == nil || x.Qc.Header == nil || x.Header != nil {
		return false
	}
	h := x.Qc.Header
	return h.Phase == ElectionVote || h.Phase == ProposeVote || h.Phase == PrecommitVote
}

// IsProposerMessage() determines if the message should originate from a Validator acting as a Leader
func (x *Message) IsProposerMessage() bool {
	h := x.Header
	if h == nil {
		return false
	}
	return h.Phase == Election || h.Phase == Propose || h.Phase == Precommit || h.Phase == Commit
}

// IsPacemakerMessage() determines if the message should describe the View of a Validator for the Pacemaker logic
func (x *Message) IsPacemakerMessage() bool {
	if x.Qc == nil || x.Qc.Header == nil {
		return false
	}
	return x.Qc.Header.Phase == RoundInterrupt
}

// checkBasic() performs basic sanity checks on the Message
func (x *Message) checkBasic(view *lib.View) lib.ErrorI {
	if x == nil {
		return ErrEmptyMessage()
	}
	if err := x.Header.Check(view, false); err != nil {
		return err
	}
	return checkSignature(x.Signature, x)
}

// GetValidateMessageParams() executes a blocking function to collect the params needed to validate a consensus message
func (b *BFT) GetValidateMessageParams(msg *Message) (*validateMessageParams, lib.ErrorI) {
	// lock the controller for thread safety
	b.Controller.Lock()
	defer b.Controller.Unlock()
	// check if a validator
	if _, err := b.ValidatorSet.GetValidator(msg.Signature.PublicKey); err != nil {
		return nil, err
	}
	// get variables needed to validate the messages
	var blockHash, resultsHash []byte
	if b.Block != nil {
		blockHash, resultsHash = b.GetBlockHash(), b.Results.Hash()
	}
	return &validateMessageParams{
		view:           b.View.Copy(),
		vals:           b.ValidatorSet,
		rootHeight:     b.RootHeight,
		height:         b.Height,
		cHeightUpdated: b.CommitteeData.LastChainHeightUpdated,
		blockHash:      blockHash,
		resultsHash:    resultsHash,
	}, nil
}

// validateMessageParams defines the parameters structure for validating consensus messages
type validateMessageParams struct {
	view           *lib.View // the current BFT view of this node
	vals           ValSet    // the validator set corresponding with the message
	rootHeight     uint64    // the height of the root chain
	height         uint64    // the chain height
	cHeightUpdated uint64    // the last 'chain' height the committee updated
	blockHash      []byte    // the hash of the proposal.Block (if any)
	resultsHash    []byte    // the hash of the proposal.Results (if any)
}

// checkSignature() validates the signature of a SignByte implementation (object that can be converted to Sign Bytes)
func checkSignature(signature *lib.Signature, sb lib.SignByte) lib.ErrorI {
	if err := checkSignatureBasic(signature); err != nil {
		return err
	}
	publicKey, err := lib.PublicKeyFromBytes(signature.PublicKey)
	if err != nil {
		return err
	}
	if !publicKey.VerifyBytes(sb.SignBytes(), signature.Signature) {
		return ErrInvalidPartialSignature()
	}
	return nil
}

// checkSignatureBasic() performs basic 'sanity checks' on a Signature object
func checkSignatureBasic(signature *lib.Signature) lib.ErrorI {
	if signature == nil || len(signature.PublicKey) == 0 || len(signature.Signature) == 0 {
		return ErrPartialSignatureEmpty()
	}
	if len(signature.PublicKey) != crypto.BLS12381PubKeySize {
		return ErrInvalidPublicKey()
	}
	if len(signature.Signature) != crypto.BLS12381SignatureSize {
		return ErrInvalidSignatureLength()
	}
	return nil
}
