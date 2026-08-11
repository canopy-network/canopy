package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

const (
	KeyStoreName = "keystore.json"
)

// NewKeyGroup() generates a public key and address that pairs with the private key
func NewKeyGroup(pk PrivateKeyI) *KeyGroup {
	pub := pk.PublicKey()
	return &KeyGroup{
		Address:    pub.Address(),
		PublicKey:  pub,
		PrivateKey: pk,
	}
}

// KeyGroup is a structure that holds the Address and PublicKey that corresponds to PrivateKey
type KeyGroup struct {
	Address    AddressI    // short version of the public key
	PublicKey  PublicKeyI  // the public code that can cryptographically verify signatures from the private key
	PrivateKey PrivateKeyI // the secret code that is capable of producing digital signatures
}

// Keystore() represents a lightweight database of private keys that are encrypted
type Keystore struct {
	AddressMap  map[string]*EncryptedPrivateKey `json:"addressMap"`            // address -> EncriptedPrivateKey
	NicknameMap map[string]string               `json:"nicknameMap,omitempty"` // nickname -> Address
}

// NewKeystoreInMemory() creates a new in memory keystore
func NewKeystoreInMemory() *Keystore {
	return &Keystore{
		AddressMap:  make(map[string]*EncryptedPrivateKey),
		NicknameMap: make(map[string]string),
	}
}

// NewKeystoreFromFile() creates a new keystore object from a file
func NewKeystoreFromFile(dataDirPath string) (*Keystore, error) {
	path := filepath.Join(dataDirPath, KeyStoreName)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return NewKeystoreInMemory(), nil
	}
	ksBz, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ks := new(Keystore)
	err = json.Unmarshal(ksBz, ks)
	if err != nil {
		return nil, err
	}
	if ks.NicknameMap == nil {
		ks.NicknameMap = make(map[string]string)
	}
	return ks, nil
}

type ImportOpts struct {
	Address  []byte
	Nickname string
}

// Import() imports an encrypted private key to the store
func (ks *Keystore) Import(encrypted *EncryptedPrivateKey, opts ImportOpts) error {
	// TODO: better naming
	encrypted.KeyAddress = hex.EncodeToString(opts.Address)
	encrypted.KeyNickname = opts.Nickname

	// this is needed to prevent saving various nicknames for the same address
	var oldNickname string
	oldKey := ks.AddressMap[encrypted.KeyAddress]
	if oldKey != nil {
		oldNickname = oldKey.KeyNickname
	}
	if oldNickname != "" {
		delete(ks.NicknameMap, oldNickname)
	}

	if opts.Nickname != "" {
		_, ok := ks.NicknameMap[opts.Nickname]
		if ok && opts.Nickname != oldNickname {
			return errors.New("nickname already used")
		}
		ks.NicknameMap[opts.Nickname] = encrypted.KeyAddress
	}

	ks.AddressMap[encrypted.KeyAddress] = encrypted

	return nil
}

type ImportRawOpts struct {
	Nickname string
}

// ImportRaw() imports a non-encrypted private key to the store, but encrypts it given a password
func (ks *Keystore) ImportRaw(privateKeyBytes []byte, password string, opts ImportRawOpts) (address string, err error) {
	if password == "" {
		return "", fmt.Errorf("invalid password")
	}

	privateKey, err := NewPrivateKeyFromBytes(privateKeyBytes)
	if err != nil {
		return
	}
	publicKey := privateKey.PublicKey()

	encrypted, err := EncryptPrivateKey(publicKey.Bytes(), privateKeyBytes, []byte(password), address)
	if err != nil {
		return
	}

	err = ks.Import(encrypted, ImportOpts{
		Address:  publicKey.Address().Bytes(),
		Nickname: opts.Nickname,
	})
	if err != nil {
		return
	}

	address = publicKey.Address().String()

	return
}

// GetKey() returns the PrivateKeyI interface for an address and decrypts it using the password
func (ks *Keystore) GetKey(address []byte, password string) (PrivateKeyI, error) {
	v, ok := ks.AddressMap[hex.EncodeToString(address)]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return DecryptPrivateKey(v, []byte(password))
}

type GetKeyGroupOpts struct {
	Address  []byte
	Nickname string
}

// GetKeyGroup() returns the full keygroup for an address or nickname and decrypts the private key using the password
func (ks *Keystore) GetKeyGroup(password string, opts GetKeyGroupOpts) (*KeyGroup, error) {
	var stringAddress string
	if opts.Address != nil {
		stringAddress = hex.EncodeToString(opts.Address)
	}
	if opts.Nickname != "" {
		stringAddress = ks.NicknameMap[opts.Nickname]
	}

	v := ks.AddressMap[stringAddress]
	if v == nil {
		return nil, fmt.Errorf("key not found")
	}
	if password == "" {
		return nil, fmt.Errorf("invalid password")
	}
	pk, err := DecryptPrivateKey(v, []byte(password))
	if err != nil {
		return nil, err
	}
	return NewKeyGroup(pk), err
}

type DeleteOpts struct {
	Address  []byte
	Nickname string
}

// DeleteKey() removes a private key from the store given an address and/or nickname after validating the password
func (ks *Keystore) DeleteKey(password string, opts DeleteOpts) error {
	var stringAddress string
	if opts.Address != nil {
		stringAddress = hex.EncodeToString(opts.Address)
	}
	if opts.Nickname != "" {
		stringAddress = ks.NicknameMap[opts.Nickname]
	}

	pKey := ks.AddressMap[stringAddress]
	if pKey == nil {
		return fmt.Errorf("key not found")
	}
	if password == "" {
		return fmt.Errorf("invalid password")
	}
	if _, err := DecryptPrivateKey(pKey, []byte(password)); err != nil {
		return err
	}

	if pKey.KeyNickname != "" {
		delete(ks.NicknameMap, pKey.KeyNickname)
	}
	delete(ks.AddressMap, pKey.KeyAddress)
	return nil
}

// SaveToFile() persists the keystore to a filepath
func (ks *Keystore) SaveToFile(dataDirPath string) error {
	bz, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDirPath, KeyStoreName), bz, 0600)
}

// Default Argon2 KDF cost params, used when an EncryptedPrivateKey has none of its own
// (i.e. any key encrypted before these fields existed).
const (
	DefaultKdfTime     = 3
	DefaultKdfMemoryKB = 32 * 1024 // 32MB
	DefaultKdfThreads  = 4
	kdfKeyLen          = 32
)

// KdfTime/KdfMemoryKB/KdfThreads record the Argon2 params used to encrypt this key, so decrypt
// always matches; zero values fall back to Default* for pre-existing keys.
type EncryptedPrivateKey struct {
	PublicKey   string `json:"publicKey"`
	Salt        string `json:"salt"`
	Encrypted   string `json:"encrypted"`
	KeyAddress  string `json:"keyAddress"`            // TODO: better naming
	KeyNickname string `json:"keyNickname,omitempty"` // TODO: better naming
	KdfTime     uint32 `json:"kdfTime,omitempty"`
	KdfMemoryKB uint32 `json:"kdfMemoryKb,omitempty"`
	KdfThreads  uint8  `json:"kdfThreads,omitempty"`
}

// EncryptPrivateKey keeps its existing signature/behavior, using the default Argon2 params.
func EncryptPrivateKey(publicKey, privateKey, password []byte, address string) (*EncryptedPrivateKey, error) {
	return EncryptPrivateKeyWithParams(publicKey, privateKey, password, address, DefaultKdfTime, DefaultKdfMemoryKB, DefaultKdfThreads)
}

// EncryptPrivateKeyWithParams lets a caller pick cheaper (or stricter) Argon2 params;
// they're stored on the result, so DecryptPrivateKey doesn't need to be told separately.
func EncryptPrivateKeyWithParams(publicKey, privateKey, password []byte, address string, time, memoryKB uint32, threads uint8) (*EncryptedPrivateKey, error) {
	// generate random 16 bytes salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	// derive an AES-GCM encryption key and nonce using the password and salt
	gcm, nonce, err := kdf(password, salt, time, memoryKB, threads)
	if err != nil {
		return nil, err
	}
	// encrypt the private key with AES-GCM using the derived key and nonce
	epk := &EncryptedPrivateKey{
		PublicKey:  hex.EncodeToString(publicKey),
		Salt:       hex.EncodeToString(salt),
		Encrypted:  hex.EncodeToString(gcm.Seal(nil, nonce, privateKey, nil)),
		KeyAddress: address,
	}
	// omit kdf* fields at the defaults so the common case marshals like a pre-kdf-params keystore.
	if time != DefaultKdfTime || memoryKB != DefaultKdfMemoryKB || threads != DefaultKdfThreads {
		epk.KdfTime = time
		epk.KdfMemoryKB = memoryKB
		epk.KdfThreads = threads
	}
	return epk, nil
}

// DecryptPrivateKey takes an EncryptedPrivateKey and decrypts it to a PrivateKeyI interface using the password
func DecryptPrivateKey(epk *EncryptedPrivateKey, password []byte) (pk PrivateKeyI, err error) {
	salt, err := hex.DecodeString(epk.Salt)
	if err != nil {
		return nil, err
	}
	encrypted, err := hex.DecodeString(epk.Encrypted)
	if err != nil {
		return nil, err
	}
	// use this key's own recorded params, falling back to defaults for pre-existing keys
	time, memoryKB, threads := epk.KdfTime, epk.KdfMemoryKB, uint32(epk.KdfThreads)
	if time == 0 && memoryKB == 0 && threads == 0 {
		time, memoryKB, threads = DefaultKdfTime, DefaultKdfMemoryKB, DefaultKdfThreads
	}
	gcm, nonce, err := kdf(password, salt, time, memoryKB, uint8(threads))
	if err != nil {
		return nil, err
	}
	plainText, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}
	return NewPrivateKeyFromBytes(plainText)
}

// kdf derives an AES-GCM encryption key and nonce from a password and salt using Argon2 key derivation
// This key is used to initialize AES-GCM, and a 12-byte nonce is returned for encryption
func kdf(password, salt []byte, time, memoryKB uint32, threads uint8) (gcm cipher.AEAD, nonce []byte, err error) {
	// use Argon2 to derive a 32 byte key from the password and salt
	key := argon2.Key(password, salt, time, memoryKB, threads, kdfKeyLen)
	// init AES block cipher with the derived key
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	// init AES-GCM mode with the AES cipher block
	if gcm, err = cipher.NewGCM(block); err != nil {
		return
	}
	// return the gcm and the 12 byte nonce
	return gcm, key[:12], nil
}

// UnmarshalJSON() implements the json.unmarshaler interface for Keygroup
func (k *KeyGroup) UnmarshalJSON(b []byte) error {
	j := new(struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	})
	if err := json.Unmarshal(b, j); err != nil {
		return err
	}
	address, err := NewAddressFromString(j.Address)
	if err != nil {
		return err
	}
	publicKey, err := NewPublicKeyFromString(j.PublicKey)
	if err != nil {
		return err
	}
	privateKey, err := NewPrivateKeyFromString(j.PrivateKey)
	if err != nil {
		return err
	}
	*k = KeyGroup{
		Address:    address,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
	return nil
}
