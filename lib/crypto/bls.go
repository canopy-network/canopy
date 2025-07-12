package crypto

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/drand/kyber"
	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/pairing"
	"github.com/drand/kyber/sign"
	"github.com/drand/kyber/sign/bdn"
	"github.com/drand/kyber/util/random"
	"os"
)

const (
	BLS12381PrivKeySize   = 32
	BLS12381PubKeySize    = 48
	BLS12381SignatureSize = 96
)

// ensure the BLS private key conforms to the PrivateKeyI interface
var _ PrivateKeyI = &BLS12381PrivateKey{}

// BLS12381PrivateKey is a private key wrapper implementation that satisfies the PrivateKeyI interface
// Boneh-Lynn-Shacham (BLS) signature scheme enables compact, aggregable digital signatures for secure, verifiable
// messages between multiple parties
type BLS12381PrivateKey struct {
	kyber.Scalar
	scheme *bdn.Scheme
}

// newBLS12381PrivateKey() creates a new BLS private key reference from a kyber.Scalar
func newBLS12381PrivateKey(privateKey kyber.Scalar) *BLS12381PrivateKey {
	return &BLS12381PrivateKey{Scalar: privateKey, scheme: newBLSScheme()}
}

// NewBLS12381PrivateKey() generates a new BLS private key
func NewBLS12381PrivateKey() (PrivateKeyI, error) {
	privateKey, _ := newBLSScheme().NewKeyPair(random.New())
	return newBLS12381PrivateKey(privateKey), nil
}

// StringToBLS12381PrivateKey() creates a new PrivateKeyI interface  from a BLS Private Key hex string
func StringToBLS12381PrivateKey(hexString string) (PrivateKeyI, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return BytesToBLS12381PrivateKey(bz)
}

// BytesToBLS12381PrivateKey() creates a new PrivateKeyI interface from a BLS Private Key bytes
func BytesToBLS12381PrivateKey(bz []byte) (PrivateKeyI, error) {
	keyCopy := newBLSSuite().G2().Scalar()
	if err := keyCopy.UnmarshalBinary(bz); err != nil {
		return nil, err
	}
	return &BLS12381PrivateKey{
		Scalar: keyCopy,
		scheme: newBLSScheme(),
	}, nil
}

// NewBLS12381PrivateKeyFromFile() creates a new PrivateKeyI interface from a BLS12381 json file
func NewBLS12381PrivateKeyFromFile(filepath string) (PrivateKeyI, error) {
	jsonBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	ptr := new(BLS12381PrivateKey)
	if err = json.Unmarshal(jsonBytes, ptr); err != nil {
		return nil, err
	}
	return ptr, nil
}

// Bytes() gives the protobuf bytes representation of the private key
func (b *BLS12381PrivateKey) Bytes() []byte {
	bz, _ := b.MarshalBinary()
	return bz
}

// Sign() digitally signs a message and returns the signature output
func (b *BLS12381PrivateKey) Sign(msg []byte) []byte {
	bz, _ := b.scheme.Sign(b.Scalar, msg)
	return bz
}

// PublicKey() returns the individual public key that pairs with this BLS private key
// for basic signature verification
func (b *BLS12381PrivateKey) PublicKey() PublicKeyI {
	suite := newBLSSuite()
	public := suite.G1().Point().Mul(b.Scalar, suite.G1().Point().Base())
	return NewBLS12381PublicKey(public)
}

// Equals() compares two private key objects and returns if they are equal
func (b *BLS12381PrivateKey) Equals(i PrivateKeyI) bool {
	private, ok := i.(*BLS12381PrivateKey)
	if !ok {
		return false
	}
	return b.Equal(private.Scalar)
}

// String() returns the hex string representation of the private key
func (b *BLS12381PrivateKey) String() string {
	return hex.EncodeToString(b.Bytes())
}

// MarshalJSON() is the json.Marshaller implementation for the BLS12381PrivateKey object
func (b *BLS12381PrivateKey) MarshalJSON() ([]byte, error) { return json.Marshal(b.String()) }

// UnmarshalJSON() is the json.Unmarshaler implementation for the BLS12381PrivateKey object
func (b *BLS12381PrivateKey) UnmarshalJSON(bz []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(bz, &hexString); err != nil {
		return
	}
	bz, err = hex.DecodeString(hexString)
	if err != nil {
		return
	}
	pk, err := BytesToBLS12381PrivateKey(bz)
	if err != nil {
		return err
	}
	bls, ok := pk.(*BLS12381PrivateKey)
	if !ok {
		return errors.New("invalid bls key")
	}
	*b = *bls
	return
}

// BLS12381PublicKey is a public key wrapper implementation that satisfies the PublicKeyI interface
// Boneh-Lynn-Shacham (BLS) signature scheme enables compact, aggregable digital signatures for secure, verifiable
// messages between multiple parties
type BLS12381PublicKey struct {
	kyber.Point
	scheme *bdn.Scheme
}

// NewBLSPublicKey creates a new BLSPublicKey reference from a kyber point
func NewBLS12381PublicKey(publicKey kyber.Point) *BLS12381PublicKey {
	return &BLS12381PublicKey{Point: publicKey, scheme: newBLSScheme()}
}

// StringToBLSPublic() creates a new PublicKeyI interface from BLS hex string
func StringToBLS12381Public(hexString string) (PublicKeyI, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return BytesToBLS12381Public(bz)
}

// BytesToBLS12381Public() creates a new PublicKeyI interface from BLS public key bytes
func BytesToBLS12381Public(bz []byte) (PublicKeyI, error) {
	point, err := BytesToBLS12381Point(bz)
	if err != nil {
		return nil, err
	}
	return &BLS12381PublicKey{
		Point:  point,
		scheme: newBLSScheme(),
	}, nil
}

// BytesToBLS12381Point() creates a new G1 point on BLS12-381 curve which is the public key of the pair
func BytesToBLS12381Point(bz []byte) (kyber.Point, error) {
	point := newBLSSuite().G1().Point()
	if err := point.UnmarshalBinary(bz); err != nil {
		return nil, err
	}
	return point, nil
}

// Address() returns the short version of the public key
func (b *BLS12381PublicKey) Address() AddressI {
	// hash the public key
	pubHash := Hash(b.Bytes())
	// take the first 20 bytes of the public key
	address := Address(pubHash[:AddressSize])
	// return the result
	return &address
}

// Bytes() returns the protobuf bytes representation of the public key
func (b *BLS12381PublicKey) Bytes() []byte {
	bz, _ := b.MarshalBinary()
	return bz
}

// MarshalJSON() implements the json.Marshaller interface for the BLS12381PublicKey
func (b *BLS12381PublicKey) MarshalJSON() ([]byte, error) { return json.Marshal(b.String()) }

// UnmarshalJSON() implements the json.Unmarshaler interface for the BLS12381PublicKey
func (b *BLS12381PublicKey) UnmarshalJSON(bz []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(bz, &hexString); err != nil {
		return
	}
	bz, err = hex.DecodeString(hexString)
	if err != nil {
		return
	}
	pk, err := BytesToBLS12381Public(bz)
	if err != nil {
		return err
	}
	bls, ok := pk.(*BLS12381PublicKey)
	if !ok {
		return errors.New("invalid bls key")
	}
	*b = *bls
	return
}

// VerifyBytes() verifies an individual BLS signature given a message and the signature out
func (b *BLS12381PublicKey) VerifyBytes(msg []byte, sig []byte) (valid bool) {
	cached, addToCache := CheckCache(b, msg, sig)
	if cached {
		return true
	}
	if valid = b.scheme.Verify(b.Point, msg, sig) == nil; valid {
		addToCache()
	}
	return
}

// Equals() compares two public key objects and returns true if they are equal
func (b *BLS12381PublicKey) Equals(i PublicKeyI) bool {
	pub2, ok := i.(*BLS12381PublicKey)
	if !ok {
		return false
	}
	return b.Equal(pub2.Point)
}

// String() returns the hex string representation of the public key
func (b *BLS12381PublicKey) String() string {
	return hex.EncodeToString(b.Bytes())
}

var _ MultiPublicKeyI = &BLS12381MultiPublicKey{}

// BLS12381MultiPublicKey is an aggregated public key created by combining multiple BLS public keys from different signers
// This combined key is used to verify an aggregated signature, confirming that a quorum (or all) of the original signers
// have participated without needing to verify each signer individually
type BLS12381MultiPublicKey struct {
	signatures [][]byte
	mask       *sign.Mask
	scheme     *bdn.Scheme
}

// NewBLSMultiPublicKey() creates a new BLS12381MultiPublicKey reference from a kyber mask object
func newBLSMultiPublicKey(mask *sign.Mask) *BLS12381MultiPublicKey {
	return &BLS12381MultiPublicKey{mask: mask, scheme: newBLSScheme(), signatures: make([][]byte, len(mask.Publics()))}
}

// NewMultiBLSFromPoints() creates a multi public key from a list of G1 points on a BLS12381 curve
func NewMultiBLSFromPoints(publicKeys []kyber.Point, bitmap []byte) (MultiPublicKeyI, error) {
	mask, err := sign.NewMask(newBLSSuite(), publicKeys, nil)
	if err != nil {
		return nil, err
	}
	if bitmap != nil {
		if err = mask.SetMask(bitmap); err != nil {
			return nil, err
		}
	}
	return newBLSMultiPublicKey(mask), nil
}

// VerifyBytes() verifies a digital signature given the original message payload and the signature out
func (b *BLS12381MultiPublicKey) VerifyBytes(msg, sig []byte) bool {
	publicKey, _ := b.scheme.AggregatePublicKeys(b.mask)
	return b.scheme.Verify(publicKey, msg, sig) == nil
}

// AggregateSignatures() aggregates multiple signatures into a single 96 byte signature
func (b *BLS12381MultiPublicKey) AggregateSignatures() ([]byte, error) {
	var ordered [][]byte
	// for each signature
	for _, signature := range b.signatures {
		// append the signature to the ordered list
		if len(signature) != 0 {
			ordered = append(ordered, signature)
		}
	}
	// aggregate the signatures using the mask into a single 96 byte signature
	signature, err := b.scheme.AggregateSignatures(ordered, b.mask)
	if err != nil {
		return nil, err
	}
	// convert the object to bytes
	return signature.MarshalBinary()
}

// AddSigner() adds a signature to the list to later be aggregated, the index represents the signer's index on the
// fixed order of the public key list
func (b *BLS12381MultiPublicKey) AddSigner(signature []byte, index int) error {
	b.signatures[index] = signature
	return b.mask.SetBit(index, true)
}

// Reset() clears the mask and signature fields of the MultiPublicKey for reuse
func (b *BLS12381MultiPublicKey) Reset() {
	b.mask, _ = sign.NewMask(newBLSSuite(), b.mask.Publics(), nil)
	b.signatures = make([][]byte, len(b.mask.Publics()))
}

// Copy() creates a safe copy of the MultiPublicKey given a list of public keys
func (b *BLS12381MultiPublicKey) Copy() MultiPublicKeyI {
	p := b.mask.Publics()
	pCopy := make([]kyber.Point, len(p))
	copy(pCopy, p)
	m := b.mask.Mask()
	mCopy := make([]byte, len(m))
	copy(mCopy, m)
	k, _ := NewMultiBLSFromPoints(pCopy, mCopy)
	return k
}

// PublicKeys() returns the ordered list of public keys from the bitmap
func (b *BLS12381MultiPublicKey) PublicKeys() (keys []PublicKeyI) {
	for _, key := range b.mask.Publics() {
		keys = append(keys, NewBLS12381PublicKey(key))
	}
	return
}

// Bitmap() returns a bitfield where each bit represents the signing status of a specific signer
// in the public key list. A set bit (1) indicates the signer at that index signed, while a cleared bit (0)
// indicates they did not
func (b *BLS12381MultiPublicKey) Bitmap() []byte { return b.mask.Mask() }
func (b *BLS12381MultiPublicKey) SignerEnabledAt(i int) (bool, error) {
	if i > len(b.PublicKeys()) || i < 0 {
		return false, errors.New("invalid bitmap index")
	}
	mask := b.Bitmap()
	byteIndex := i / 8
	mm := byte(1) << (i & 7)
	return mask[byteIndex]&mm != 0, nil
}

// SetBitmap() is used to set the mask of a BLS Multi key
func (b *BLS12381MultiPublicKey) SetBitmap(bm []byte) error { return b.mask.SetMask(bm) }
func newBLSScheme() *bdn.Scheme                             { return bdn.NewSchemeOnG2(newBLSSuite()) }
func newBLSSuite() pairing.Suite                            { return bls12381.NewBLS12381Suite() }

func MaxBitmapSize(numValidators uint64) int {
	return int((numValidators + 7) / 8)
}
