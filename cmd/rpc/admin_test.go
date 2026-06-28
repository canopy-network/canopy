package rpc

import (
	"testing"

	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestSetRequestPublicKeyPreservesExplicitPubKey(t *testing.T) {
	operatorKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	signerKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	ptr := &txRequest{PubKey: operatorKey.PublicKey().String()}

	setRequestPublicKey(ptr, signerKey)

	require.Equal(t, operatorKey.PublicKey().String(), ptr.PubKey)
}

func TestSetRequestPublicKeyFallsBackToSignerPubKey(t *testing.T) {
	signerKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	ptr := new(txRequest)

	setRequestPublicKey(ptr, signerKey)

	require.Equal(t, signerKey.PublicKey().String(), ptr.PubKey)
}

func TestStakePublicKeyFromRequestRequiresValidatorAddress(t *testing.T) {
	operatorKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	signerKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	ptr := &txRequest{
		PubKey:         signerKey.PublicKey().String(),
		addressRequest: addressRequest{Address: operatorKey.PublicKey().Address().Bytes()},
	}

	_, err = stakePublicKeyFromRequest(ptr)

	require.ErrorContains(t, err, "stake public key address must match validator address")
}

func TestStakePublicKeyFromRequestAcceptsValidatorPubKey(t *testing.T) {
	operatorKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	ptr := &txRequest{
		PubKey:         operatorKey.PublicKey().String(),
		addressRequest: addressRequest{Address: operatorKey.PublicKey().Address().Bytes()},
	}

	got, err := stakePublicKeyFromRequest(ptr)

	require.NoError(t, err)
	require.True(t, operatorKey.PublicKey().Equals(got))
}
