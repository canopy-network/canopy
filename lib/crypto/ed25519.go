package crypto

import (
	ed25519 "crypto/ed25519"
	"encoding/hex"
	"encoding/json"
)

const (
	Ed25519PrivKeySize   = ed25519.PrivateKeySize
	Ed25519PubKeySize    = ed25519.PublicKeySize
	Ed25519SignatureSize = ed25519.SignatureSize
)

// Private Key Below

// ED25519PrivateKey is the private key of a cryptographic key pair used in elliptic curve signing and verification, based on the Curve25519 elliptic curve
// It is used to create 'unique' digital signatures of messages
type ED25519PrivateKey struct{ ed25519.PrivateKey }

// NewPrivateKeyED25519() creates a new ED25519PrivateKey wrapper that satisfies the PrivateKeyI interface
func NewPrivateKeyED25519(privateKey ed25519.PrivateKey) *ED25519PrivateKey {
	return &ED25519PrivateKey{PrivateKey: privateKey}
}

// ensure ED25519PrivateKey satisfies PrivateKeyI interface
var _ PrivateKeyI = &ED25519PrivateKey{}

// String() returns the hex string representation of the private key
func (p *ED25519PrivateKey) String() string { return hex.EncodeToString(p.Bytes()) }

// Bytes() casts the private key to bytes
func (p *ED25519PrivateKey) Bytes() []byte { return p.PrivateKey }

// Sign() returns the digital signature out of an Ed25519 private key sign function given a message
func (p *ED25519PrivateKey) Sign(msg []byte) []byte { return ed25519.Sign(p.PrivateKey, msg) }

// PublicKey() returns the public key that pairs with this private key object
func (p *ED25519PrivateKey) PublicKey() PublicKeyI {
	return &ED25519PublicKey{p.PrivateKey.Public().(ed25519.PublicKey)}
}

// Equals() compares two private key objects and returns true if they are equal
func (p *ED25519PrivateKey) Equals(key PrivateKeyI) bool {
	return p.PrivateKey.Equal(ed25519.PrivateKey(key.Bytes()))
}

// MarshalJSON() implements the json.Marshaller interface for ED25519PrivateKey
func (p *ED25519PrivateKey) MarshalJSON() ([]byte, error) { return json.Marshal(p.String()) }

// UnmarshalJSON() implements the json.Marshaller interface for ED25519PrivateKey
func (p *ED25519PrivateKey) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	*p = *NewPrivateKeyED25519(bz)
	return
}

// Public Key Below

// ED25519PublicKey is the public key of a cryptographic key pair used in elliptic curve signing and verification, based on the Curve25519 elliptic curve
// It is used to verify ownership of the private key as well as validate digital signatures created by the private key
type ED25519PublicKey struct{ ed25519.PublicKey }

// NewPublicKeyED25519() returns a ED25519PublicKey reference that satisfies the PublicKeyI interface
func NewPublicKeyED25519(publicKey ed25519.PublicKey) *ED25519PublicKey {
	return &ED25519PublicKey{PublicKey: publicKey}
}

// ensure the ED25519PublicKey object satisfies the PublicKeyI interface
var _ PublicKeyI = &ED25519PublicKey{}

// Address() returns the short version of the public key
func (p *ED25519PublicKey) Address() AddressI {
	// hash the public key
	pubHash := Hash(p.Bytes())
	// take the first 20 bytes of the hash
	address := Address(pubHash[:AddressSize])
	// return the result
	return &address
}

// MarshalJSON() implements the json.Marshaller interface for ED25519PublicKey
func (p *ED25519PublicKey) MarshalJSON() ([]byte, error) { return json.Marshal(p.String()) }

// UnmarshalJSON() implements the json.Unmarshaler interface for ED25519PublicKey
func (p *ED25519PublicKey) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	*p = *NewPublicKeyED25519(bz)
	return
}

// Bytes() casts the public key to bytes
func (p *ED25519PublicKey) Bytes() []byte { return p.PublicKey }

// String() returns the hex string representation of the public key
func (p *ED25519PublicKey) String() string { return hex.EncodeToString(p.Bytes()) }

// VerifyBytes() validates a digital signature was signed by the paired private key given the message signed
func (p *ED25519PublicKey) VerifyBytes(msg []byte, sig []byte) bool {
	return ed25519.Verify(p.PublicKey, msg, sig)
}

// Equals() compares two public key objects and returns if the two are equal
func (p *ED25519PublicKey) Equals(i PublicKeyI) bool {
	return p.PublicKey.Equal(ed25519.PublicKey(i.Bytes()))
}
