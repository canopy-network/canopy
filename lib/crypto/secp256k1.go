package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/ripemd160"
)

/* This file implements logic for SECP256K1 when the public key is compressed (33 bytes) - this affects the 'addressing' and 'verify bytes' logic */

const (
	SECP256K1PrivKeySize   = 32
	SECP256K1PubKeySize    = 33
	SECP256K1SignatureSize = 64
)

// Private Key Below

// ensure SECP256K1PrivateKey conforms to the PrivateKeyI interface
var _ PrivateKeyI = &SECP256K1PrivateKey{}

// NewSECP256K1PrivateKey() generates a new SECP256K1 private key
func NewSECP256K1PrivateKey() (PrivateKeyI, error) {
	pk, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return BytesToSECP256K1Private(ethCrypto.FromECDSA(pk))
}

// NewETHSECP256K1PrivateKey() generates a new ETHSECP256K1 private key
func NewETHSECP256K1PrivateKey() (PrivateKeyI, error) {
	pk, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	priv, err := BytesToSECP256K1Private(ethCrypto.FromECDSA(pk))
	if err != nil {
		return nil, err
	}
	return &ETHSECP256K1PrivateKey{*priv}, nil
}

// BytesToSECP256K1Private() converts bytes to SECP256K1 private key using go-ethereum
func BytesToSECP256K1Private(b []byte) (*SECP256K1PrivateKey, error) {
	pk, err := ethCrypto.ToECDSA(b)
	if err != nil {
		return nil, err
	}
	return &SECP256K1PrivateKey{PrivateKey: pk}, nil
}

// StringToSECP256K1Private() creates a new PrivateKeyI interface from an SECP256K1 hex string
func StringToSECP256K1Private(hexString string) (PrivateKeyI, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return BytesToSECP256K1Private(bz)
}

// SECP256K1PrivateKey is the private key of a cryptographic key pair used in elliptic curve signing and verification, based on the SECP256K1 elliptic curve
// It is used to create 'unique' digital signatures of messages
type SECP256K1PrivateKey struct {
	*ecdsa.PrivateKey
}

// MarshalJSON() is the json.Marshaller implementation for SECP256K1PrivateKey
func (s *SECP256K1PrivateKey) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// UnmarshalJSON() is the json.Unmarshaler implementation for SECP256K1PrivateKey
func (s *SECP256K1PrivateKey) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	pk, err := BytesToSECP256K1Private(bz)
	if err != nil {
		return
	}
	*s = *pk
	return
}

// Sign() returns digital signature bytes from the message
func (s *SECP256K1PrivateKey) Sign(msg []byte) []byte {
	sig, _ := ethCrypto.Sign(Hash(msg), s.PrivateKey)
	// a 1-byte value used to indicate the Ethereum 'recovery byte' is omitted
	return sig[:len(sig)-1]
}

// PublicKey() returns the public pair to this private key
func (s *SECP256K1PrivateKey) PublicKey() PublicKeyI {
	return &SECP256K1PublicKey{PublicKey: &s.PrivateKey.PublicKey}
}

// Bytes() returns the byte representation of the private key
func (s *SECP256K1PrivateKey) Bytes() []byte { return ethCrypto.FromECDSA(s.PrivateKey) }

// String() returns the hex string representation of the private key
func (s *SECP256K1PrivateKey) String() string { return hex.EncodeToString(s.Bytes()) }

// Equals() compares to private keys and returns true if they are equal
func (s *SECP256K1PrivateKey) Equals(i PrivateKeyI) bool { return bytes.Equal(s.Bytes(), i.Bytes()) }

// Public Key Below

// ensure SECP256K1PublicKey conforms to the PublicKeyI interface
var _ PublicKeyI = &SECP256K1PublicKey{}

// SECP256K1PublicKey is the public key of a cryptographic key pair used in elliptic curve signing and verification, based on the SECP256K1 elliptic curve
// It is used to verify ownership of the private key as well as validate digital signatures created by the private key
type SECP256K1PublicKey struct {
	compressed []byte
	*ecdsa.PublicKey
}

// BytesToSECP256K1Public() returns SECP256K1PublicKey from bytes
func BytesToSECP256K1Public(b []byte) (*SECP256K1PublicKey, error) {
	pub, err := ethCrypto.DecompressPubkey(b)
	if err != nil {
		return nil, err
	}
	return &SECP256K1PublicKey{PublicKey: pub}, nil
}

// MarshalJSON() is the json.Marshaller implementation for SECP256K1PublicKey
func (s *SECP256K1PublicKey) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// UnmarshalJSON() is the json.Unmarshaler implementation for SECP256K1PublicKey
func (s *SECP256K1PublicKey) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	pk, err := BytesToSECP256K1Public(bz)
	if err != nil {
		return
	}
	*s = *pk
	return
}

// Address() returns the short version of the public key
// Address format varies between chains:
// - Cosmos, Harmony, Binance, Avalanche RIPEMD-160(SHA-256(pubkey)) <Tendermint>
// - BTC, BCH, BSV, (and other forks) <1 byte version> + RIPEMD-160(SHA-256(pubkey)) + <4 byte Checksum>
// `RIPEMD-160(SHA-256(pubkey))` seems to be the most common theme in addressing for SECP256K1 public keys
func (s *SECP256K1PublicKey) Address() AddressI {
	hasher := ripemd160.New()
	hasher.Write(Hash(s.Bytes()))
	address := Address(hasher.Sum(nil))
	return &address
}

// VerifyBytes() returns true if the digital signature is valid for this public key and the given message
func (s *SECP256K1PublicKey) VerifyBytes(msg []byte, sig []byte) (valid bool) {
	cached, addToCache := CheckCache(s, msg, sig)
	if cached {
		return true
	}
	if valid = ethCrypto.VerifySignature(s.Bytes(), Hash(msg), sig); valid {
		addToCache()
	}
	return
}

// Bytes() returns the byte representation of the Public Key
func (s *SECP256K1PublicKey) Bytes() []byte {
	if s.compressed == nil {
		s.compressed = ethCrypto.CompressPubkey(s.PublicKey)
	}
	return s.compressed
}

// String() returns the hex string representation of the public key
func (s *SECP256K1PublicKey) String() string { return hex.EncodeToString(s.Bytes()) }

// Equals() compares two SECP256K1PublicKey objects and returns true if they're equal
func (s *SECP256K1PublicKey) Equals(i PublicKeyI) bool { return bytes.Equal(s.Bytes(), i.Bytes()) }
