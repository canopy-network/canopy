package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestAddressFromKeyMalformed(t *testing.T) {
	_, err := AddressFromKey([]byte{0xff})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid key")
}

func TestIdFromKeyMalformed(t *testing.T) {
	_, err := IdFromKey(lib.JoinLenPrefix([]byte{1}, []byte{1, 2}))
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid key")
}
