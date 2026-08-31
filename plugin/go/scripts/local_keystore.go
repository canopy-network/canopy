package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

const (
	defaultKdfTime     = 3
	defaultKdfMemoryKB = 32 * 1024
	defaultKdfThreads  = 4
	kdfKeyLen          = 32
)

type encryptedPrivateKey struct {
	PublicKey   string `json:"publicKey"`
	Salt        string `json:"salt"`
	Encrypted   string `json:"encrypted"`
	KeyAddress  string `json:"keyAddress"`
	KeyNickname string `json:"keyNickname,omitempty"`
	KdfTime     uint32 `json:"kdfTime,omitempty"`
	KdfMemoryKB uint32 `json:"kdfMemoryKb,omitempty"`
	KdfThreads  uint8  `json:"kdfThreads,omitempty"`
}

type localKeystoreFile struct {
	AddressMap  map[string]*encryptedPrivateKey `json:"addressMap"`
	NicknameMap map[string]string               `json:"nicknameMap,omitempty"`
}

// defaultKeystorePath returns ~/.canopy/keystore.json, matching Canopy's default data dir.
func defaultKeystorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".canopy", "keystore.json"), nil
}

// keystoreGetKeyLocal reads and decrypts a key entirely locally, mirroring
// lib/crypto/keystore.go's DecryptPrivateKey -- no network call, so private
// key material never leaves this device. Replaces the old remote
// POST {adminRPCURL}/v1/admin/keystore-get flow, which required the node
// itself to hold and serve key material -- wrong assumption for a shared
// grad node not under our control.
func keystoreGetKeyLocal(address, password string) (*keyGroup, error) {
	path, err := defaultKeystorePath()
	if err != nil {
		return nil, err
	}
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keystore at %s: %w", path, err)
	}
	var ks localKeystoreFile
	if err := json.Unmarshal(bz, &ks); err != nil {
		return nil, fmt.Errorf("parse keystore: %w", err)
	}
	epk, ok := ks.AddressMap[address]
	if !ok {
		return nil, fmt.Errorf("address %s not found in local keystore %s", address, path)
	}
	privKeyBytes, err := decryptPrivateKeyLocal(epk, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("decrypt key for %s: %w", address, err)
	}
	return &keyGroup{
		Address:    address,
		PublicKey:  epk.PublicKey,
		PrivateKey: hex.EncodeToString(privKeyBytes),
	}, nil
}

func decryptPrivateKeyLocal(epk *encryptedPrivateKey, password []byte) ([]byte, error) {
	salt, err := hex.DecodeString(epk.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	encrypted, err := hex.DecodeString(epk.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted blob: %w", err)
	}
	kt, km, kthr := epk.KdfTime, epk.KdfMemoryKB, uint32(epk.KdfThreads)
	if kt == 0 && km == 0 && kthr == 0 {
		kt, km, kthr = defaultKdfTime, defaultKdfMemoryKB, defaultKdfThreads
	}
	gcm, nonce, err := kdfLocal(password, salt, kt, km, uint8(kthr))
	if err != nil {
		return nil, err
	}
	plainText, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open (likely wrong password): %w", err)
	}
	return plainText, nil
}

func kdfLocal(password, salt []byte, time, memoryKB uint32, threads uint8) (cipher.AEAD, []byte, error) {
	key := argon2.Key(password, salt, time, memoryKB, threads, kdfKeyLen)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	return gcm, key[:12], nil
}
