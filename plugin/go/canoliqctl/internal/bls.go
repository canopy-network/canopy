package internal

import (
	"encoding/hex"

	"github.com/drand/kyber"
	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/pairing"
	"github.com/drand/kyber/sign/bdn"
)

// BLS12381PrivateKey wraps a BLS12-381 scalar with the BDN signing scheme used
// by Canopy validators and tx signers. Mirrors plugin/go/tutorial/crypto/bls.go;
// duplicated here because the tutorial lives in a separate Go module.
type BLS12381PrivateKey struct {
	kyber.Scalar
	scheme *bdn.Scheme
}

// PrivateKeyFromHex parses a hex-encoded BLS12-381 private key as produced by
// the admin keystore-get RPC.
func PrivateKeyFromHex(s string) (*BLS12381PrivateKey, error) {
	bz, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	scalar := newBLSSuite().G2().Scalar()
	if err := scalar.UnmarshalBinary(bz); err != nil {
		return nil, err
	}
	return &BLS12381PrivateKey{Scalar: scalar, scheme: newBLSScheme()}, nil
}

// Sign produces a BLS signature over msg using the BDN scheme.
func (b *BLS12381PrivateKey) Sign(msg []byte) []byte {
	bz, _ := b.scheme.Sign(b.Scalar, msg)
	return bz
}

func newBLSScheme() *bdn.Scheme   { return bdn.NewSchemeOnG2(newBLSSuite()) }
func newBLSSuite() pairing.Suite { return bls12381.NewBLS12381Suite() }
