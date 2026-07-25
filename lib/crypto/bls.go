package crypto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/drand/kyber"
	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/pairing"
	"github.com/drand/kyber/sign"
	"github.com/drand/kyber/sign/bdn"
	"github.com/drand/kyber/util/random"
	"google.golang.org/protobuf/proto"
	"os"
	"sort"
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

// BLS12381MultiPublicKey is an aggregated public key created by combining multiple BLS public keys from different signers.
// This type intentionally exposes two representations:
//   - Address(): canonical multisig account identity for the signer set
//   - Bytes()/Bitmap()/AddSigner(): order-preserving verification state where signer indices are meaningful
type BLS12381MultiPublicKey struct {
	signatures [][]byte
	mask       *sign.Mask
	scheme     *bdn.Scheme
	threshold  uint32
}

var errAccountAuthThreshold = errors.New("account-auth multisig requires threshold > 0")

// NewBLSMultiPublicKey() creates a new BLS12381MultiPublicKey reference from a kyber mask object
func newBLSMultiPublicKey(mask *sign.Mask, threshold uint32) *BLS12381MultiPublicKey {
	return &BLS12381MultiPublicKey{mask: mask, scheme: newBLSScheme(), signatures: make([][]byte, len(mask.Publics())), threshold: threshold}
}

// NewMultiBLSFromPoints() creates a multi public key from a list of G1 points on a BLS12381 curve.
// Important: this preserves caller order for backwards compatibility. Bitmap indices and AddSigner() indices refer to
// this exact order, so this constructor must not canonicalize or sort the signer list.
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
	return newBLSMultiPublicKey(mask, 0), nil
}

// NewAccountAuthMultiBLSFromPoints creates a multisig public key intended for account authorization.
// Unlike the consensus-oriented constructor, this requires a positive threshold so callers cannot
// accidentally create an open multisig account.
func NewAccountAuthMultiBLSFromPoints(publicKeys []kyber.Point, bitmap []byte, threshold uint32) (MultiPublicKeyI, error) {
	if threshold == 0 {
		return nil, errAccountAuthThreshold
	}
	mask, err := sign.NewMask(newBLSSuite(), publicKeys, nil)
	if err != nil {
		return nil, err
	}
	if bitmap != nil {
		if err = mask.SetMask(bitmap); err != nil {
			return nil, err
		}
	}
	if threshold > uint32(len(mask.Publics())) {
		return nil, errors.New("invalid public key")
	}
	return newBLSMultiPublicKey(mask, threshold), nil
}

// NewMultiBLSFromPublicKey creates a BLS multikey from serialized bytes.
// The encoded public key order is preserved exactly so bitmap signer indices survive a Bytes()/decode roundtrip.
func NewMultiBLSFromPublicKey(publicKey []byte) (MultiPublicKeyI, error) {
	size, errInvalidPK := len(publicKey), errors.New("invalid public key")
	if size == 0 || size > 1_000_000 {
		return nil, errInvalidPK
	}
	// unmarshal into a multi-public-key
	mpk := new(MultiPublicKey)
	if err := proto.Unmarshal(publicKey, mpk); err != nil {
		return nil, err
	}
	// sanity check the size
	if len(mpk.PublicKeys) == 0 || len(mpk.Bitmap) == 0 || mpk.Threshold > uint32(len(mpk.PublicKeys)) {
		return nil, errInvalidPK
	}
	var points []kyber.Point
	seen := make(map[string]struct{}, len(mpk.PublicKeys))
	// convert to a kyber.point
	for _, key := range mpk.PublicKeys {
		if _, exists := seen[string(key)]; exists {
			return nil, errors.New("duplicate bls public key")
		}
		seen[string(key)] = struct{}{}
		point, err := BytesToBLS12381Point(key)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	mask, err := sign.NewMask(newBLSSuite(), points, nil)
	if err != nil {
		return nil, err
	}
	if err = mask.SetMask(mpk.Bitmap); err != nil {
		return nil, err
	}
	return newBLSMultiPublicKey(mask, mpk.Threshold), nil
}

// NewAccountAuthMultiBLSFromPublicKey creates a serialized multisig public key intended for account authorization.
// It rejects threshold-0 keys so callers do not accidentally accept an open account policy.
func NewAccountAuthMultiBLSFromPublicKey(publicKey []byte) (MultiPublicKeyI, error) {
	key, err := NewMultiBLSFromPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	bls, ok := key.(*BLS12381MultiPublicKey)
	if !ok {
		return nil, errors.New("invalid bls key")
	}
	if bls.threshold == 0 {
		return nil, errAccountAuthThreshold
	}
	return bls, nil
}

// Address() returns the canonical 20 byte account identity for the signer set.
// It sorts public keys so the address is stable across different signer orderings and ignores the bitmap.
func (b *BLS12381MultiPublicKey) Address() AddressI {
	var together []byte
	for _, k := range b.PubKeys() {
		together = append(together, k...)
	}
	threshold := make([]byte, 4)
	binary.BigEndian.PutUint32(threshold, b.threshold)
	together = append(together, threshold...)
	return Address(Hash(together)[:20])
}

// VerifyBytes() verifies a digital signature given the original message payload and the signature out
func (b *BLS12381MultiPublicKey) VerifyBytes(msg, sig []byte) bool {
	publicKey, _ := b.scheme.AggregatePublicKeys(b.mask)
	if b.scheme.Verify(publicKey, msg, sig) != nil {
		return false
	}
	return b.threshold == 0 || uint32(b.mask.CountEnabled()) >= b.threshold
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

// AddSigner() adds a signature to the list to later be aggregated.
// The index refers to the preserved internal signer order, not the canonicalized order used by Address().
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
	mask, _ := sign.NewMask(newBLSSuite(), pCopy, nil)
	_ = mask.SetMask(mCopy)
	return newBLSMultiPublicKey(mask, b.threshold)
}

// PublicKeys() returns the public keys in the preserved internal order used by the bitmap and signer indices.
func (b *BLS12381MultiPublicKey) PublicKeys() (keys []PublicKeyI) {
	for _, key := range b.mask.Publics() {
		keys = append(keys, NewBLS12381PublicKey(key))
	}
	return
}

// Bytes() serializes the concrete MPK instance, preserving internal key order and bitmap semantics.
// Two MPKs for the same signer set may therefore have different bytes if their preserved order or bitmap differs.
func (b *BLS12381MultiPublicKey) Bytes() []byte {
	publicKeys := make([][]byte, 0, len(b.mask.Publics()))
	for _, key := range b.mask.Publics() {
		publicKeys = append(publicKeys, NewBLS12381PublicKey(key).Bytes())
	}
	bz, _ := proto.Marshal(&MultiPublicKey{
		PublicKeys: publicKeys,
		Bitmap:     b.Bitmap(),
		Threshold:  b.threshold,
	})
	return bz
}

// String() converts BLS12381MultiPublicKey to hex bytes
func (b *BLS12381MultiPublicKey) String() string {
	return hex.EncodeToString(b.Bytes())
}

// Equals() evaluates equality using canonical multisig account identity rather than serialized MPK bytes.
func (b *BLS12381MultiPublicKey) Equals(i PublicKeyI) bool {
	return bytes.Equal(b.Address().Bytes(), i.Address().Bytes())
}

// PubKeys() returns the raw public keys sorted for canonical signer-set operations such as Address().
// It must not be used for signer-indexed bitmap semantics.
func (b *BLS12381MultiPublicKey) PubKeys() [][]byte {
	var pubs [][]byte
	for _, k := range b.PublicKeys() {
		pubs = append(pubs, k.Bytes())
	}
	sort.Slice(pubs, func(i, j int) bool {
		return bytes.Compare(pubs[i], pubs[j]) < 0
	})
	return pubs
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

// EnabledSignerCount returns the number of enabled signers in the bitmap.
func (b *BLS12381MultiPublicKey) EnabledSignerCount() int { return b.mask.CountEnabled() }

// Threshold returns the minimum enabled signers required for this multisig policy.
func (b *BLS12381MultiPublicKey) Threshold() uint32 { return b.threshold }
func newBLSScheme() *bdn.Scheme                     { return bdn.NewSchemeOnG2(newBLSSuite()) }
func newBLSSuite() pairing.Suite                    { return bls12381.NewBLS12381Suite() }

func MaxBitmapSize(numValidators uint64) int {
	return int((numValidators + 7) / 8)
}

// jsonBLS12381MultiPublicKey is a helper for the json marshal / unmarshal interface
type jsonBLS12381MultiPublicKey string

// MarshalJSON() convert BLS12381MultiPublicKey into json bytes (string)
func (b *BLS12381MultiPublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonBLS12381MultiPublicKey(b.String()))
}

// UnmarshalJSON() convert json bytes into BLS12381MultiPublicKey
func (b *BLS12381MultiPublicKey) UnmarshalJSON(i []byte) (err error) {
	j := new(jsonBLS12381MultiPublicKey)
	if err = json.Unmarshal(i, j); err != nil {
		return err
	}
	bz, err := hex.DecodeString(string(*j))
	if err != nil {
		return err
	}
	mpk, err := NewMultiBLSFromPublicKey(bz)
	if err != nil {
		return err
	}
	k, ok := mpk.(*BLS12381MultiPublicKey)
	if !ok {
		return errors.New("invalid bls key")
	}
	*b = *k
	return
}
