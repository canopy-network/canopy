package crypto

import (
	"encoding/hex"

	"github.com/drand/kyber"
	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/pairing"
	"github.com/drand/kyber/sign/bdn"
)

const (
	BLS12381PrivKeySize   = 32
	BLS12381PubKeySize    = 48
	BLS12381SignatureSize = 96
)

// BLS12381PrivateKey is a private key wrapper for BLS12-381 signing
type BLS12381PrivateKey struct {
	kyber.Scalar
	scheme *bdn.Scheme
}

// BytesToBLS12381PrivateKey creates a private key from bytes
func BytesToBLS12381PrivateKey(bz []byte) (*BLS12381PrivateKey, error) {
	keyCopy := newBLSSuite().G2().Scalar()
	if err := keyCopy.UnmarshalBinary(bz); err != nil {
		return nil, err
	}
	return &BLS12381PrivateKey{
		Scalar: keyCopy,
		scheme: newBLSScheme(),
	}, nil
}

// StringToBLS12381PrivateKey creates a private key from hex string
func StringToBLS12381PrivateKey(hexString string) (*BLS12381PrivateKey, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return BytesToBLS12381PrivateKey(bz)
}

// Sign digitally signs a message and returns the signature
func (b *BLS12381PrivateKey) Sign(msg []byte) []byte {
	bz, _ := b.scheme.Sign(b.Scalar, msg)
	return bz
}

// Bytes returns the byte representation of the private key
func (b *BLS12381PrivateKey) Bytes() []byte {
	bz, _ := b.Scalar.MarshalBinary()
	return bz
}

// PublicKey returns the public key paired with this private key
func (b *BLS12381PrivateKey) PublicKey() *BLS12381PublicKey {
	suite := newBLSSuite()
	public := suite.G1().Point().Mul(b.Scalar, suite.G1().Point().Base())
	return &BLS12381PublicKey{Point: public, scheme: newBLSScheme()}
}

// BLS12381PublicKey is a public key wrapper for BLS12-381
type BLS12381PublicKey struct {
	kyber.Point
	scheme *bdn.Scheme
}

// Bytes returns the byte representation of the public key
func (b *BLS12381PublicKey) Bytes() []byte {
	bz, _ := b.Point.MarshalBinary()
	return bz
}

// BytesToBLS12381PublicKey creates a public key from bytes
func BytesToBLS12381PublicKey(bz []byte) (*BLS12381PublicKey, error) {
	point := newBLSSuite().G1().Point()
	if err := point.UnmarshalBinary(bz); err != nil {
		return nil, err
	}
	return &BLS12381PublicKey{
		Point:  point,
		scheme: newBLSScheme(),
	}, nil
}

// StringToBLS12381PublicKey creates a public key from hex string
func StringToBLS12381PublicKey(hexString string) (*BLS12381PublicKey, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return BytesToBLS12381PublicKey(bz)
}

// Verify verifies a signature against a message
func (b *BLS12381PublicKey) Verify(msg []byte, sig []byte) bool {
	return b.scheme.Verify(b.Point, msg, sig) == nil
}

func newBLSScheme() *bdn.Scheme    { return bdn.NewSchemeOnG2(newBLSSuite()) }
func newBLSSuite() pairing.Suite  { return bls12381.NewBLS12381Suite() }
