package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

/* This file implements logic for SECP256K1 when the public key is not compressed (64 bytes) - this affects the 'addressing' and 'verify bytes' logic */

const (
	ETHSECP256K1PubKeySize = 64 // represents the uncompressed SECP256K1 public key size
)

// ensure ETHSECP256K1PublicKey conforms to the PublicKeyI interface
var _ PublicKeyI = &ETHSECP256K1PublicKey{}
var _ PrivateKeyI = &ETHSECP256K1PrivateKey{}

type ETHSECP256K1PrivateKey struct {
	SECP256K1PrivateKey
}

// EthPublicKey() returns the ethereum public pair to this private key
func (s *ETHSECP256K1PrivateKey) PublicKey() PublicKeyI {
	return &ETHSECP256K1PublicKey{PublicKey: &s.PrivateKey.PublicKey}
}

// BytesToEthSECP256K1Private() converts bytes to SECP256K1 private key using go-ethereum
func BytesToEthSECP256K1Private(b []byte) (*ETHSECP256K1PrivateKey, error) {
	pk, err := ethCrypto.ToECDSA(b)
	if err != nil {
		return nil, err
	}
	return &ETHSECP256K1PrivateKey{SECP256K1PrivateKey{PrivateKey: pk}}, nil
}

// ETHSECP256K1PublicKey is the ethereum variant of the public key of a cryptographic key pair used in elliptic curve signing and verification,
// based on the SECP256K1 elliptic curve, it is used to verify ownership of the private key as well as validate digital signatures created by the private key
type ETHSECP256K1PublicKey struct {
	*ecdsa.PublicKey
}

// BytesToEthSECP256K1Public() returns ETHSECP256K1PublicKey from bytes
func BytesToEthSECP256K1Public(b []byte) (*ETHSECP256K1PublicKey, error) {
	if len(b) == ETHSECP256K1PubKeySize {
		b = append([]byte{0x04}, b...) // add the SEC1 prefix
	}
	pub, err := ethCrypto.UnmarshalPubkey(b)
	if err != nil {
		return nil, err
	}
	return &ETHSECP256K1PublicKey{PublicKey: pub}, nil
}

// Bytes() returns the byte representation of the Public Key
func (s *ETHSECP256K1PublicKey) Bytes() []byte {
	return s.BytesWithPrefix()[1:]
}

// Bytes() returns the byte representation of the Public Key
func (s *ETHSECP256K1PublicKey) BytesWithPrefix() []byte {
	return ethCrypto.FromECDSAPub(s.PublicKey)
}

// MarshalJSON() is the json.Marshaller implementation for ETHSECP256K1PublicKey
func (s *ETHSECP256K1PublicKey) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// UnmarshalJSON() is the json.Unmarshaler implementation for ETHSECP256K1PublicKey
func (s *ETHSECP256K1PublicKey) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	pk, err := BytesToEthSECP256K1Public(bz)
	if err != nil {
		return
	}
	*s = *pk
	return
}

// Address() returns the short version of the public key
func (s *ETHSECP256K1PublicKey) Address() AddressI {
	a := Address(ethCrypto.PubkeyToAddress(*s.PublicKey).Bytes())
	return &a
}

// VerifyBytes() returns true if the digital signature is valid for this public key and the given message
func (s *ETHSECP256K1PublicKey) VerifyBytes(msg []byte, sig []byte) (valid bool) {
	cached, addToCache := CheckCache(s, msg, sig)
	if cached {
		return true
	}
	if valid = ethCrypto.VerifySignature(s.BytesWithPrefix(), Hash(msg), sig); valid {
		addToCache()
	}
	return
}

// String() returns the hex string representation of the public key
func (s *ETHSECP256K1PublicKey) String() string { return hex.EncodeToString(s.Bytes()) }

// Equals() compares two ETHSECP256K1PublicKey objects and returns true if they're equal
func (s *ETHSECP256K1PublicKey) Equals(i PublicKeyI) bool { return bytes.Equal(s.Bytes(), i.Bytes()) }

// RecoverPublicKey() recovers a public key from ethereum the transaction and validates the signature
func RecoverPublicKey(signer types.Signer, tx types.Transaction) (PublicKeyI, error) {
	// extract signature values
	Vb, R, S := tx.RawSignatureValues()
	if Vb == nil || R == nil || S == nil {
		return nil, types.ErrInvalidSig
	}
	// compute sighash (what was signed)
	sigHash := signer.Hash(&tx)
	// normalize V to 0 or 1
	var recoveryID byte
	V := Vb.Uint64()
	switch {
	case V == 27 || V == 28:
		recoveryID = byte(V - 27)
	case V >= 35:
		recoveryID = byte((V - 35) % 2) // EIP-155 recovery ID
	case V == 0 || V == 1:
		recoveryID = byte(V) // Typed txs (EIP-2930, 1559, etc.)
	default:
		return nil, types.ErrInvalidSig
	}
	// validate signature values
	if !ethCrypto.ValidateSignatureValues(recoveryID, R, S, true) {
		return nil, types.ErrInvalidSig
	}
	// assemble 65-byte signature: R || S || V (recovery ID)
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = recoveryID
	// recover the public key
	pubKeyBytes, err := ethCrypto.Ecrecover(sigHash[:], sig)
	if err != nil {
		return nil, err
	}
	// verify signature to guard against invalid keys
	if !ethCrypto.VerifySignature(pubKeyBytes, sigHash[:], sig[:64]) {
		return nil, types.ErrInvalidSig
	}
	// convert to PublicKeyI
	return NewPublicKeyFromBytes(pubKeyBytes)
}
