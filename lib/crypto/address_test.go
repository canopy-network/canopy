package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddress(t *testing.T) {
	// create a new public key object
	public, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	// cast the public key to bytes
	addressBytes := Hash(public)[:20]
	// convert to an address object
	address := NewAddress(addressBytes)
	// validate string function
	require.Equal(t, address.String(), hex.EncodeToString(addressBytes))
	// validate bytes function
	require.Equal(t, addressBytes, address.Bytes())
	// validate equals function
	require.True(t, address.Equals(NewAddress(addressBytes)))
	// validate json marshalling
	marshalled, err := json.Marshal(address)
	require.NoError(t, err)
	// validate expected json vs got
	require.Equal(t, string(marshalled), "\""+address.String()+"\"")
	// validate unmarshalling
	unmarshalled := new(Address)
	require.NoError(t, json.Unmarshal(marshalled, unmarshalled))
	// validate expected unmarshalled
	require.Equal(t, address, unmarshalled)
}
