package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// PublicKeyI is an interface model for a cryptographic code shared openly, used to verify digital signatures of its paired private key
type PublicKeyI interface {
	// Address() creates a unique shorter fixed length version of the public key
	Address() AddressI
	// Bytes() casts the public key to bytes
	Bytes() []byte
	// VerifyBytes() verifies a digital signature from its corresponding private key
	VerifyBytes(msg []byte, sig []byte) bool
	// String() returns the hex string representation
	String() string
	// Equals() compares two PublicKeys and returns true if they're equal
	Equals(PublicKeyI) bool
	// models the json.Marshaller encoding interface
	json.Marshaler
	// models the json.Unmarshaler decoding interface
	json.Unmarshaler
}

// PrivateKeyI is an interface model for a secret cryptographic code that is used to produce digital signatures
type PrivateKeyI interface {
	Bytes() []byte
	Sign(msg []byte) []byte
	PublicKey() PublicKeyI
	// String() returns the hex string representation
	String() string
	Equals(PrivateKeyI) bool
	// models the json.Marshaller encoding interface
	json.Marshaler
	// models the json.Unmarshaler decoding interface
	json.Unmarshaler
}

// AddressI is an interface model for the short version of the Public Key
type AddressI interface {
	// Marshal() models the protobuf.Marshaller interface
	Marshal() ([]byte, error)
	// Bytes() casts the public key to bytes
	Bytes() []byte
	// String() returns the hex string representation
	String() string
	Equals(AddressI) bool
}

// MultiPublicKeyI is an interface model for a multi-signature public key, representing multiple signers in a single structure
// It allows aggregation of individual signatures, validation of aggregated signatures, and management of signers through a bitmap
// that tracks which participants have signed
type MultiPublicKeyI interface {
	AggregateSignatures() ([]byte, error)
	// VerifyBytes() verifies a digital aggregate signature from multiple signers
	VerifyBytes(msg, aggregatedSignature []byte) bool
	// AddSigner() is used to track signers by setting a bit at the index position (from the pre-created public key list)
	AddSigner(signature []byte, index int) error
	// SignerEnabledAt() returns true if a signer is enabled at a certain bit
	SignerEnabledAt(i int) (bool, error)
	// PublicKeys() returns the list of public keys
	PublicKeys() (keys []PublicKeyI)
	// SetBitmap() loads the values of a bitmap into the MultiPublicKey
	SetBitmap(bm []byte) error
	// Bitmap() returns a clone of the bitmap of the MPK
	// The bitmap is used to track who signed or not
	Bitmap() []byte
	// Copy() returns a safe clone of the MPK
	Copy() MultiPublicKeyI
	// Reset() resets the PublicKey list and the Bitmap values
	Reset()
}

// NewPublicKeyFromString() creates a new PublicKeyI interface from a hex string
func NewPublicKeyFromString(s string) (PublicKeyI, error) {
	bz, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return NewPublicKeyFromBytes(bz)
}

// NewPublicKeyFromBytes() creates a new PublicKeyI interface from a byte slice
func NewPublicKeyFromBytes(bz []byte) (PublicKeyI, error) {
	switch len(bz) {
	case Ed25519PubKeySize:
		return BytesToED25519Public(bz), nil
	case ETHSECP256K1PubKeySize, ETHSECP256K1PubKeySize + 1:
		return BytesToEthSECP256K1Public(bz)
	case SECP256K1PubKeySize:
		return BytesToSECP256K1Public(bz)
	case BLS12381PubKeySize:
		return BytesToBLS12381Public(bz)
	}
	return nil, fmt.Errorf("unrecognized public key format")
}

// PrivateKeyToFile() writes a private key to a file located at filepath
func PrivateKeyToFile(key PrivateKeyI, filepath string) error {
	bz, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, bz, 0777)
}

// NewPrivateKeyFromString() creates a new PrivateKeyI interface from a hex string
func NewPrivateKeyFromString(s string) (PrivateKeyI, error) {
	bz, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return NewPrivateKeyFromBytes(bz)
}

// NewPrivateKeyFromBytes() creates a new PrivateKeyI interface from bytes
func NewPrivateKeyFromBytes(bz []byte) (PrivateKeyI, error) {
	switch len(bz) {
	case BLS12381PrivKeySize:
		//pk, err := BytesToSECP256K1Private(bz)
		//if err == nil {
		//	return pk, nil
		//}
		return BytesToBLS12381PrivateKey(bz)
	case Ed25519PrivKeySize:
		return BytesToED25519Private(bz), nil
	default:
		return nil, fmt.Errorf("unrecognized private key format: %d", len(bz))
	}
}
