package rpc

import (
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/require"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

func TestGetValidatorSetCached(t *testing.T) {
	key, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	validators := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{{PublicKey: key.PublicKey().Bytes(), VotingPower: 1}}}
	cache, err := lru.New[validatorSetKey, *lib.ConsensusValidators](validatorSetCacheSize)
	require.NoError(t, err)
	cache.Add(validatorSetKey{rootChainId: 1, chainId: 2, rootHeight: 100}, validators)
	// the subscription has no http client, so a cache miss would panic on the remote call
	r := &RCManager{
		subscriptions: map[uint64]*RCSubscription{1: {chainId: 1, Info: &lib.RootChainInfo{Height: 110}}},
		validatorSets: cache,
		log:           lib.NewNullLogger(),
	}
	set, err := r.GetValidatorSet(1, 2, 100)
	require.NoError(t, err)
	require.Equal(t, uint64(1), set.TotalPower, "a past root height must be served from the cache")
	other, err := r.GetValidatorSet(1, 2, 100)
	require.NoError(t, err)
	require.NotSame(t, set.MultiKey, other.MultiKey, "each hit must return a fresh validator set")
}
