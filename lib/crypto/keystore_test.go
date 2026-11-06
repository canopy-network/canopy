package crypto

import (
	ed25519 "crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDecryptPrivateKeyPrependedNonce ensures a keystore produced with a random,
// prepended nonce (the format exported by launchpad / newer wallets) can be
// decrypted here. Previously decrypt only supported the key-derived-nonce layout,
// so importing a launchpad-exported keyfile failed (surfaced as a 400 / bad password).
func TestDecryptPrivateKeyPrependedNonce(t *testing.T) {
	password := []byte("password")
	// pre-create a new private key
	private, err := NewBLS12381PrivateKey()
	require.NoError(t, err)
	// random 16-byte salt, matching the encrypt path
	salt := make([]byte, 16)
	_, err = rand.Read(salt)
	require.NoError(t, err)
	// derive the AES-GCM cipher using the default Argon2 params
	gcm, _, err := kdf(password, salt, DefaultKdfTime, DefaultKdfMemoryKB, DefaultKdfThreads)
	require.NoError(t, err)
	// generate an independent random nonce and prepend it to the ciphertext
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	combined := append(nonce, gcm.Seal(nil, nonce, private.Bytes(), nil)...)
	// build the encrypted key record as launchpad would export it
	epk := &EncryptedPrivateKey{
		PublicKey:  hex.EncodeToString(private.PublicKey().Bytes()),
		Salt:       hex.EncodeToString(salt),
		Encrypted:  hex.EncodeToString(combined),
		KeyAddress: private.PublicKey().Address().String(),
	}
	// decrypt and validate round-trip
	got, err := DecryptPrivateKey(epk, password)
	require.NoError(t, err)
	require.EqualExportedValues(t, private, got)
	// a wrong password must still fail
	_, err = DecryptPrivateKey(epk, []byte("wrong"))
	require.Error(t, err)
}

// TestDecryptPrivateKeyLaunchpadEd25519Seed ensures a launchpad ED25519 key stored as a raw 32-byte
// seed (with prepended nonce) is reconstructed as ED25519, not misread as a 32-byte BLS12-381 key
func TestDecryptPrivateKeyLaunchpadEd25519Seed(t *testing.T) {
	password := []byte("password")
	// use the key's 32-byte seed, as the launchpad exports it
	edPriv, err := NewEd25519PrivateKey()
	require.NoError(t, err)
	seed := edPriv.Bytes()[:ed25519.SeedSize]
	// random 16-byte salt, matching the encrypt path
	salt := make([]byte, 16)
	_, err = rand.Read(salt)
	require.NoError(t, err)
	// derive the AES-GCM cipher using the default Argon2 params
	gcm, _, err := kdf(password, salt, DefaultKdfTime, DefaultKdfMemoryKB, DefaultKdfThreads)
	require.NoError(t, err)
	// prepend an independent random nonce to the ciphertext (launchpad format)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	combined := append(nonce, gcm.Seal(nil, nonce, seed, nil)...)
	// record the ED25519 public key, as launchpad exports it
	epk := &EncryptedPrivateKey{
		PublicKey:  hex.EncodeToString(edPriv.PublicKey().Bytes()),
		Salt:       hex.EncodeToString(salt),
		Encrypted:  hex.EncodeToString(combined),
		KeyAddress: edPriv.PublicKey().Address().String(),
	}
	// decrypt and ensure the key is ED25519 with the original address, not a BLS key
	got, err := DecryptPrivateKey(epk, password)
	require.NoError(t, err)
	require.Equal(t, Ed25519PubKeySize, len(got.PublicKey().Bytes()))
	require.Equal(t, edPriv.PublicKey().Address().String(), got.PublicKey().Address().String())
	require.Equal(t, edPriv.PublicKey().Bytes(), got.PublicKey().Bytes())
}

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
