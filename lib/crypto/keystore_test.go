package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeystoreImportWithOpts(t *testing.T) {
	password := []byte("password")
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// encrypt the private key
	encrypted, err := EncryptPrivateKey(private.PublicKey().Bytes(), private.Bytes(), password, private.PublicKey().Address().String())
	require.NoError(t, err)
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	require.NoError(t, ks.Import(encrypted, ImportOpts{
		Address:  address,
		Nickname: "pablito",
	}))
	// check the key was imported
	got, err := ks.GetKey(address, string(password))
	require.NoError(t, err)
	// validate got vs expected
	require.EqualExportedValues(t, private, got)
	// check the key was imported by nickname
	pKey, err := ks.GetKeyGroup(string(password), GetKeyGroupOpts{
		Nickname: "pablito",
	})
	require.NoError(t, err)
	// validate got vs expected
	require.EqualExportedValues(t, private, pKey.PrivateKey)
}

func TestKeystoreImportRawWithOpts(t *testing.T) {
	password := "password"
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	gotAddress, err := ks.ImportRaw(private.Bytes(), password, ImportRawOpts{})
	require.NoError(t, err)
	// validate got address vs expected
	require.Equal(t, hex.EncodeToString(address), gotAddress)
	// check the key was imported
	got, err := ks.GetKeyGroup(password, GetKeyGroupOpts{
		Address: address,
	})
	require.NoError(t, err)
	// validate got vs expected private key
	require.EqualExportedValues(t, private, got.PrivateKey)
	// validate got vs expected public key
	require.EqualExportedValues(t, private.PublicKey(), got.PublicKey)
}

func TestKeystoreDeleteKeyWithOpts(t *testing.T) {
	password := "password"
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// get the address
	address := private.PublicKey().Address().Bytes()
	// create a new in-memory keystore
	ks := NewKeystoreInMemory()
	// execute the function call
	gotAddress, err := ks.ImportRaw(private.Bytes(), password, ImportRawOpts{
		Nickname: "pablito",
	})
	require.NoError(t, err)
	// validate got address vs expected
	require.Equal(t, hex.EncodeToString(address), gotAddress)
	// delete the key
	require.NoError(t, ks.DeleteKey(password, DeleteOpts{
		Address: address,
	}))
	// check the key was imported with address
	_, err = ks.GetKey(address, password)
	require.ErrorContains(t, err, "key not found")
	// check the key was imported with nickname
	_, err = ks.GetKeyGroup(password, GetKeyGroupOpts{
		Nickname: "pablito",
	})
	require.ErrorContains(t, err, "key not found")

	// execute the function call
	gotAddress, err = ks.ImportRaw(private.Bytes(), password, ImportRawOpts{
		Nickname: "pablito",
	})
	require.NoError(t, err)
	// validate got address vs expected
	require.Equal(t, hex.EncodeToString(address), gotAddress)
	// delete the key
	require.NoError(t, ks.DeleteKey(password, DeleteOpts{
		Nickname: "pablito",
	}))
	// check the key was imported with address
	_, err = ks.GetKey(address, password)
	require.ErrorContains(t, err, "key not found")
	// check the key was imported with nickname
	_, err = ks.GetKeyGroup(password, GetKeyGroupOpts{
		Nickname: "pablito",
	})
	require.ErrorContains(t, err, "key not found")
}
