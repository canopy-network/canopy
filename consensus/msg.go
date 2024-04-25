package consensus

import (
	"bytes"
	"github.com/ginchuco/ginchu/lib"
	"github.com/ginchuco/ginchu/lib/crypto"
)

func (x *Message) SignBytes() (signBytes []byte, err lib.ErrorI) {
	switch {
	case x.IsProposerMessage():
		return lib.Marshal(&Message{
			Header: x.Header,
			Vrf:    x.Vrf,
			Qc: &QC{
				Header:      x.Qc.Header,
				Block:       x.Qc.Block,
				BlockHash:   x.Qc.BlockHash,
				ProposerKey: x.Qc.ProposerKey,
				Signature:   x.Qc.Signature,
			},
			HighQc:                 x.HighQc,
			LastDoubleSignEvidence: x.LastDoubleSignEvidence,
			BadProposerEvidence:    x.BadProposerEvidence,
		})
	case x.IsReplicaMessage():
		return lib.Marshal(&QC{
			Header:      x.Qc.Header,
			BlockHash:   x.Qc.BlockHash,
			ProposerKey: x.Qc.ProposerKey,
		})
	case x.IsPacemakerMessage():
		return lib.Marshal(&Message{Header: x.Header})
	default:
		return nil, ErrUnknownConsensusMsg(x)
	}
}

func (x *Message) CheckProposerMessage(expectedProposer, expectedBlockHash []byte, height uint64,
	loadValSet func(height uint64) (lib.ValidatorSet, lib.ErrorI)) (isPartialQC bool, err lib.ErrorI) {
	if err = x.checkBasic(0); err != nil {
		return false, err
	}
	if expectedProposer != nil {
		if !bytes.Equal(expectedProposer, x.Signature.PublicKey) {
			return false, lib.ErrInvalidProposerPubKey()
		}
	}
	if x.Header.Phase == Election {
		if x.Header.Height != height {
			return false, lib.ErrWrongHeight()
		}
		if err = checkSignatureBasic(x.Vrf); err != nil {
			return false, err
		}
		if !bytes.Equal(x.Signature.PublicKey, x.Vrf.PublicKey) {
			return false, ErrMismatchPublicKeys()
		}
	} else {
		var vals ValSet
		if x.Qc.Header == nil {
			return false, lib.ErrEmptyView()
		}
		vals, err = loadValSet(x.Qc.Header.Height)
		if err != nil {
			return false, err
		}
		isPartialQC, err = x.Qc.Check(0, vals) // REPLICAS: CAPTURE PARTIAL QCs FROM ANY HEIGHT
		if err != nil {
			return
		}
		if isPartialQC {
			return
		}
		if x.Header.Height != height || x.Qc.Header.Height != height {
			return false, lib.ErrWrongHeight()
		}
		if x.Header.Phase == Propose {
			if err = x.Qc.Block.Check(); err != nil {
				return
			}
			if len(x.Qc.ProposerKey) != crypto.BLS12381PubKeySize {
				return false, lib.ErrInvalidProposerPubKey()
			}
		} else {
			if !bytes.Equal(x.Qc.BlockHash, expectedBlockHash) {
				return false, ErrMismatchBlockHash()
			}
		}
	}
	return
}

func (x *Message) CheckReplicaMessage(height uint64, expectedBlockHash []byte, vs ValSet) lib.ErrorI {
	if err := x.checkBasic(height); err != nil {
		return err
	}
	if x.IsPacemakerMessage() {
		return nil
	}
	isPartialQC, err := x.Qc.Check(height, vs)
	if err != nil {
		return err
	}
	if isPartialQC {
		return lib.ErrNoMaj23()
	}
	if x.Qc.Header.Phase == ElectionVote {
		if len(x.Qc.ProposerKey) != crypto.BLS12381PubKeySize {
			return lib.ErrInvalidProposerPubKey()
		}
	} else {
		if x.Qc.Header.Phase == ProposeVote {
			if !bytes.Equal(x.Signature.PublicKey, x.Qc.ProposerKey) {
				return ErrMismatchPublicKeys()
			}
		}
		if !bytes.Equal(x.Qc.BlockHash, expectedBlockHash) {
			return ErrMismatchBlockHash()
		}
	}
	return nil
}

func (x *Message) Sign(privateKey crypto.PrivateKeyI) lib.ErrorI {
	bz, err := x.SignBytes()
	if err != nil {
		return err
	}
	x.Signature = new(lib.Signature)
	x.Signature.PublicKey = privateKey.PublicKey().Bytes()
	x.Signature.Signature = privateKey.Sign(bz)
	return nil
}

func (x *Message) IsReplicaMessage() bool {
	if x.Header != nil {
		return false
	}
	h := x.Qc.Header
	return h.Phase == ElectionVote || h.Phase == ProposeVote || h.Phase == PrecommitVote
}

func (x *Message) IsProposerMessage() bool {
	h := x.Header
	if h == nil {
		return false
	}
	return h.Phase == Election || h.Phase == Propose || h.Phase == Precommit || h.Phase == Commit
}

func (x *Message) IsPacemakerMessage() bool {
	h := x.Header
	if h == nil {
		return false
	}
	return h.Phase == RoundInterrupt
}

func (x *Message) checkBasic(height uint64) lib.ErrorI {
	if x == nil {
		return ErrEmptyProposerMessage()
	}
	if err := checkSignature(x.Signature, x); err != nil {
		return err
	}
	if err := x.Header.Check(height); err != nil {
		return err
	}
	return nil
}

func checkSignature(signature *lib.Signature, sb lib.SignByte) lib.ErrorI {
	if err := checkSignatureBasic(signature); err != nil {
		return err
	}
	publicKey, err := lib.PublicKeyFromBytes(signature.PublicKey)
	if err != nil {
		return err
	}
	msg, err := sb.SignBytes()
	if err != nil {
		return err
	}
	if !publicKey.VerifyBytes(msg, signature.Signature) {
		return ErrInvalidPartialSignature()
	}
	return nil
}

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