package crypto

import (
	"bytes"
	"github.com/drand/kyber"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"sort"
	"testing"
)

func TestBLS(t *testing.T) {
	// generate a message to test with
	msg := []byte("hello world")
	// create a new bls private key
	k1, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// create a second bls private key
	k2, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// create a third bls private key
	k3, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// organize the 3 keys in a list
	publicKeys := [][]byte{k1.PublicKey().Bytes(), k2.PublicKey().Bytes(), k3.PublicKey().Bytes()}
	// convert the keys to kyber points and save to a list
	var points []kyber.Point
	for _, bz := range publicKeys {
		point, e := BytesToBLS12381Point(bz)
		require.NoError(t, e)
		points = append(points, point)
	}
	// generate a new multi-public key from that list
	multiKey, err := NewMultiBLSFromPoints(points, nil)
	require.NoError(t, err)
	// sign the message with the first private key
	k1Sig := k1.Sign(msg)
	// sign the message with the third private key
	k3Sig := k3.Sign(msg)
	// update the bitmap using constructor order
	require.NoError(t, multiKey.AddSigner(k1Sig, 0))
	require.NoError(t, multiKey.AddSigner(k3Sig, 2))
	enabled, err := multiKey.SignerEnabledAt(0)
	require.NoError(t, err)
	require.True(t, enabled)
	enabled, err = multiKey.SignerEnabledAt(1)
	require.NoError(t, err)
	require.False(t, enabled)
	enabled, err = multiKey.SignerEnabledAt(2)
	require.NoError(t, err)
	require.True(t, enabled)
	// aggregate the signature
	sig, err := multiKey.AggregateSignatures()
	require.NoError(t, err)
	// ensure that a +2/3rds majority passes
	require.True(t, multiKey.VerifyBytes(msg, sig))
}

func TestNewBLSPointFromBytes(t *testing.T) {
	k1, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	k1Pub := k1.PublicKey().(*BLS12381PublicKey)
	point := k1Pub.Point
	bytes := k1Pub.Bytes()
	point2, err := BytesToBLS12381Point(bytes)
	require.NoError(t, err)
	require.True(t, point.Equal(point2))
}

func TestMultiBLSBytesRoundTripPreservesOrderAndBitmap(t *testing.T) {
	keys := make([]PrivateKeyI, 3)
	points := make([]kyber.Point, 3)
	for i := range keys {
		var err error
		keys[i], err = NewBLS12381PrivateKey()
		require.NoError(t, err)
		points[i], err = BytesToBLS12381Point(keys[i].PublicKey().Bytes())
		require.NoError(t, err)
	}

	shuffledPoints := []kyber.Point{points[1], points[2], points[0]}
	multiKey, err := NewMultiBLSFromPoints(shuffledPoints, nil)
	require.NoError(t, err)
	require.NoError(t, multiKey.AddSigner(keys[1].Sign([]byte("hello")), 0))
	require.NoError(t, multiKey.AddSigner(keys[0].Sign([]byte("hello")), 2))

	canonical := multiKey.(*BLS12381MultiPublicKey).Bytes()
	roundTrip, err := NewMultiBLSFromPublicKey(canonical)
	require.NoError(t, err)

	originalPublicKeys := multiKey.PublicKeys()
	roundTripPublicKeys := roundTrip.PublicKeys()
	require.Equal(t, len(originalPublicKeys), len(roundTripPublicKeys))
	for i := range originalPublicKeys {
		require.True(t, bytes.Equal(originalPublicKeys[i].Bytes(), roundTripPublicKeys[i].Bytes()))
	}

	require.Equal(t, multiKey.Bitmap(), roundTrip.Bitmap())
	for i := range originalPublicKeys {
		enabled, err := multiKey.SignerEnabledAt(i)
		require.NoError(t, err)
		roundTripEnabled, err := roundTrip.SignerEnabledAt(i)
		require.NoError(t, err)
		require.Equal(t, enabled, roundTripEnabled)
	}

	var encoded MultiPublicKey
	require.NoError(t, proto.Unmarshal(canonical, &encoded))
	encoded.Bitmap[0] |= 0x80
	paddingVariant, err := proto.Marshal(&encoded)
	require.NoError(t, err)
	_, err = NewMultiBLSFromPublicKey(paddingVariant)
	require.Error(t, err)

	unknownFieldVariant := append(bytes.Clone(canonical), 0x78, 0x01)
	_, err = NewMultiBLSFromPublicKey(unknownFieldVariant)
	require.Error(t, err)
}

func TestMultiBLSThresholdEnforcedByVerifyBytes(t *testing.T) {
	msg := []byte("hello")
	keys := make([]PrivateKeyI, 3)
	points := make([]kyber.Point, 3)
	publicKeys := make([][]byte, 3)
	for i := range keys {
		var err error
		keys[i], err = NewBLS12381PrivateKey()
		require.NoError(t, err)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].PublicKey().Bytes(), keys[j].PublicKey().Bytes()) < 0
	})
	for i := range keys {
		publicKeys[i] = keys[i].PublicKey().Bytes()
		var err error
		points[i], err = BytesToBLS12381Point(publicKeys[i])
		require.NoError(t, err)
	}

	multiKey, err := NewMultiBLSFromPoints(points, nil)
	require.NoError(t, err)
	require.NoError(t, multiKey.AddSigner(keys[0].Sign(msg), 0))
	require.NoError(t, multiKey.AddSigner(keys[1].Sign(msg), 1))
	sig, err := multiKey.AggregateSignatures()
	require.NoError(t, err)

	withThreshold, err := proto.Marshal(&MultiPublicKey{
		PublicKeys: publicKeys,
		Bitmap:     multiKey.Bitmap(),
		Threshold:  3,
	})
	require.NoError(t, err)

	thresholdKey, err := NewMultiBLSFromPublicKey(withThreshold)
	require.NoError(t, err)
	require.False(t, thresholdKey.VerifyBytes(msg, sig))

	sameSignerSetDifferentThreshold, err := proto.Marshal(&MultiPublicKey{
		PublicKeys: publicKeys,
		Bitmap:     multiKey.Bitmap(),
		Threshold:  2,
	})
	require.NoError(t, err)

	thresholdTwoKey, err := NewMultiBLSFromPublicKey(sameSignerSetDifferentThreshold)
	require.NoError(t, err)
	require.True(t, thresholdTwoKey.VerifyBytes(msg, sig))
	require.False(t, thresholdKey.Address().Equals(thresholdTwoKey.Address()))
}

func TestAccountAuthMultiBLSConstructorsRequirePositiveThreshold(t *testing.T) {
	keys := make([]PrivateKeyI, 3)
	points := make([]kyber.Point, 3)
	for i := range keys {
		var err error
		keys[i], err = NewBLS12381PrivateKey()
		require.NoError(t, err)
		points[i], err = BytesToBLS12381Point(keys[i].PublicKey().Bytes())
		require.NoError(t, err)
	}

	_, err := NewAccountAuthMultiBLSFromPoints(points, nil, 0)
	require.ErrorIs(t, err, errAccountAuthThreshold)

	sort.Slice(points, func(i, j int) bool {
		left, leftErr := points[i].MarshalBinary()
		require.NoError(t, leftErr)
		right, rightErr := points[j].MarshalBinary()
		require.NoError(t, rightErr)
		return bytes.Compare(left, right) < 0
	})
	accountAuthKey, err := NewAccountAuthMultiBLSFromPoints(points, nil, 2)
	require.NoError(t, err)
	require.Equal(t, uint32(2), accountAuthKey.(*BLS12381MultiPublicKey).Threshold())

	roundTrip, err := NewAccountAuthMultiBLSFromPublicKey(accountAuthKey.(*BLS12381MultiPublicKey).Bytes())
	require.NoError(t, err)
	require.Equal(t, uint32(2), roundTrip.(*BLS12381MultiPublicKey).Threshold())

	consensusStyleKey, err := NewMultiBLSFromPoints(points, nil)
	require.NoError(t, err)
	_, err = NewAccountAuthMultiBLSFromPublicKey(consensusStyleKey.(*BLS12381MultiPublicKey).Bytes())
	require.ErrorIs(t, err, errAccountAuthThreshold)

	reversed := accountAuthKey.PublicKeys()
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&MultiPublicKey{
		PublicKeys: [][]byte{reversed[2].Bytes(), reversed[1].Bytes(), reversed[0].Bytes()},
		Bitmap:     accountAuthKey.Bitmap(),
		Threshold:  2,
	})
	require.NoError(t, err)
	_, err = NewMultiBLSFromPublicKey(encoded)
	require.Error(t, err)
}

func TestBLSRejectsPointAtInfinity(t *testing.T) {
	infinity, err := newBLSSuite().G1().Point().Null().MarshalBinary()
	require.NoError(t, err)

	_, err = BytesToBLS12381Public(infinity)
	require.Error(t, err)

	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&MultiPublicKey{
		PublicKeys: [][]byte{infinity},
		Bitmap:     []byte{1},
		Threshold:  1,
	})
	require.NoError(t, err)
	_, err = NewMultiBLSFromPublicKey(encoded)
	require.Error(t, err)
}
