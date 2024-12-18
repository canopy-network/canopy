package crypto

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeystoreImport(t *testing.T) {
	password := []byte("password")
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// encrypt the private key
	encrypted, err := EncryptPrivateKey(private.PublicKey().Bytes(), private.Bytes(), password)
	require.NoError(t, err)
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	require.NoError(t, ks.Import(address, encrypted))
	// check the key was imported
	got, err := ks.GetKey(address, string(password))
	require.NoError(t, err)
	// validate got vs expected
	require.EqualExportedValues(t, private, got)
}

func TestKeystoreImportRaw(t *testing.T) {
	password := "password"
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	gotAddress, err := ks.ImportRaw(private.Bytes(), password)
	require.NoError(t, err)
	// validate got address vs expected
	require.Equal(t, hex.EncodeToString(address), gotAddress)
	// check the key was imported
	got, err := ks.GetKeyGroup(address, password)
	require.NoError(t, err)
	// validate got vs expected private key
	require.EqualExportedValues(t, private, got.PrivateKey)
	// validate got vs expected public key
	require.EqualExportedValues(t, private.PublicKey(), got.PublicKey)
}

func TestKeystoreDeleteKey(t *testing.T) {
	password := "password"
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	gotAddress, err := ks.ImportRaw(private.Bytes(), password)
	require.NoError(t, err)
	// validate got address vs expected
	require.Equal(t, hex.EncodeToString(address), gotAddress)
	// delete the key
	ks.DeleteKey(address)
	// check the key was imported
	_, err = ks.GetKey(address, password)
	require.ErrorContains(t, err, "key not found")
}
