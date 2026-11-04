package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/canopy-network/canopy/lib/crypto"
)

func TestCertificateResultEqualsRootBuildHeight(t *testing.T) {
	a := &CertificateResult{RootBuildHeight: 100}
	b := &CertificateResult{RootBuildHeight: 100}
	require.True(t, a.Equals(b), "equal root build heights must be equal")
	b.RootBuildHeight = 101
	require.False(t, a.Equals(b), "a different root build height must not be equal")
}

// TestCertificateResultUnknownFieldRoundTrip simulates a node on an older binary that doesn't know a certificate
// result field: decoding and re-encoding must keep the bytes and hash identical, so the results hash still matches
func TestCertificateResultUnknownFieldRoundTrip(t *testing.T) {
	results := &CertificateResult{
		RewardRecipients: &RewardRecipients{PaymentPercents: []*PaymentPercents{{Address: []byte("address"), Percent: 100, ChainId: 1}}},
		SlashRecipients:  &SlashRecipients{},
		RootBuildHeight:  100,
	}
	bz, err := Marshal(results)
	require.NoError(t, err)
	// append a field number this binary doesn't know, like field 8 for an older binary
	bz = protowire.AppendTag(bz, 99, protowire.VarintType)
	bz = protowire.AppendVarint(bz, 12345)
	// decode and re-encode as an unaware node would when rebuilding the results hash
	decoded := new(CertificateResult)
	require.NoError(t, Unmarshal(bz, decoded))
	reEncoded, err := Marshal(decoded)
	require.NoError(t, err)
	require.Equal(t, bz, reEncoded, "unknown fields must survive a decode/encode round trip")
	require.Equal(t, crypto.Hash(bz), decoded.Hash(), "the results hash must match the original bytes")
}
