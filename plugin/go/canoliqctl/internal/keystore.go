package internal

import (
	"encoding/json"
	"fmt"
)

// Key holds the address + key material returned by /v1/admin/keystore-get.
// Both PublicKey and PrivateKey are hex-encoded BLS12-381 material.
type Key struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// KeystoreNewKey creates a new key entry under the given nickname and returns
// its address. The keystore is exposed by the node's admin RPC port.
func KeystoreNewKey(adminURL, nickname, password string) (string, error) {
	body, err := PostJSON(adminURL+"/v1/admin/keystore-new-key",
		fmt.Sprintf(`{"nickname":%q,"password":%q}`, nickname, password))
	if err != nil {
		return "", err
	}
	var address string
	if err := json.Unmarshal(body, &address); err != nil {
		return "", fmt.Errorf("parse keystore-new-key: %v: %s", err, body)
	}
	return address, nil
}

// KeystoreGet retrieves the key material for address from the admin keystore.
// The password unwraps the encrypted private-key blob server-side.
func KeystoreGet(adminURL, address, password string) (*Key, error) {
	body, err := PostJSON(adminURL+"/v1/admin/keystore-get",
		fmt.Sprintf(`{"address":%q,"password":%q}`, address, password))
	if err != nil {
		return nil, err
	}
	var k Key
	if err := json.Unmarshal(body, &k); err != nil {
		return nil, fmt.Errorf("parse keystore-get: %v: %s", err, body)
	}
	return &k, nil
}
