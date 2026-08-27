package signer_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/lib/crypto/signer"
)

func TestSignSendGoldenVector(t *testing.T) {
	// sign the deterministic golden request
	transaction, err := signer.SignSend(goldenRequest(), goldenKey())
	if err != nil {
		t.Fatal(err)
	}
	// inspect the signed transaction
	details, err := signer.Inspect(transaction)
	if err != nil {
		t.Fatal(err)
	}
	// validate the deterministic transaction hash
	if details.TransactionHash != "3c12e6f105125934803f39291dc649dcc3488efdf55ea9d04107956cd45a90fe" {
		t.Fatalf("golden transaction hash changed: %s", details.TransactionHash)
	}
	// encode the signed transaction
	encoded, err := json.Marshal(transaction)
	if err != nil {
		t.Fatal(err)
	}
	// validate the canonical signed transaction JSON
	const goldenJSON = `{"type":"send","msg":{"fromAddress":"3097e2dee2cb4a34b53840cdb705aed71067c36f","toAddress":"502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711","amount":1234567},"signature":{"publicKey":"2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12","signature":"02dc03269c709ba3485d38a70f6e433e66b76fc7f21c886d5fb617be7b7b19907d7b9119d5bcb59f424ed44e1e785c9562d3320c87f62cecab863a901d342a06"},"time":1786569463230678,"createdHeight":2044370,"fee":10000,"memo":"withdrawal-123","networkID":1,"chainID":1}`
	if string(encoded) != goldenJSON {
		t.Fatalf("golden transaction JSON changed:\n%s", encoded)
	}

	// decode and verify the JSON round trip
	decoded := new(lib.Transaction)
	if err = json.Unmarshal(encoded, decoded); err != nil {
		t.Fatal(err)
	}
	if _, err = signer.Inspect(decoded); err != nil {
		t.Fatalf("verify JSON round trip: %v", err)
	}
}

func TestInspectRejectsTampering(t *testing.T) {
	// create a valid signed transaction
	transaction, err := signer.SignSend(goldenRequest(), goldenKey())
	if err != nil {
		t.Fatal(err)
	}
	// tamper with a signed field
	transaction.Fee++
	// ensure signature verification detects the change
	if _, err = signer.Inspect(transaction); err == nil || !strings.Contains(err.Error(), "invalid transaction signature") {
		t.Fatalf("expected invalid signature, got %v", err)
	}
}

func TestBuildSendRejectsInvalidRequests(t *testing.T) {
	// create a valid base request
	valid := goldenRequest()
	// define each invalid request mutation
	tests := []struct {
		name   string
		mutate func(*signer.SendRequest)
	}{
		{"recipient", func(r *signer.SendRequest) { r.Recipient = "abcd" }},
		{"amount", func(r *signer.SendRequest) { r.AmountUCNPY = 0 }},
		{"height", func(r *signer.SendRequest) { r.CreatedHeight = 0 }},
		{"network", func(r *signer.SendRequest) { r.NetworkID = 0 }},
		{"chain", func(r *signer.SendRequest) { r.ChainID = 0 }},
		{"time", func(r *signer.SendRequest) { r.Time = 0 }},
		{"memo bytes", func(r *signer.SendRequest) { r.Memo = strings.Repeat("é", 101) }},
		{"RLP memo", func(r *signer.SendRequest) { r.Memo = lib.RLPIndicator }},
		{"RLP.V2 memo", func(r *signer.SendRequest) { r.Memo = lib.RLPV2Indicator }},
	}
	// execute the validation cases
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// mutate a copy of the valid request
			request := valid
			test.mutate(&request)
			// ensure the invalid request is rejected
			if _, err := signer.BuildSend(request, goldenKey().PublicKey()); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSignSendRejectsNilPrivateKey(t *testing.T) {
	// ensure a nil private key is rejected
	if _, err := signer.SignSend(goldenRequest(), nil); err == nil {
		t.Fatal("expected private key error")
	}
}

// goldenRequest() returns the deterministic send request used by signer tests
func goldenRequest() signer.SendRequest {
	return signer.SendRequest{
		Recipient:     "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
		AmountUCNPY:   1_234_567,
		FeeUCNPY:      10_000,
		CreatedHeight: 2_044_370,
		NetworkID:     1,
		ChainID:       1,
		Memo:          "withdrawal-123",
		Time:          1_786_569_463_230_678,
	}
}

// goldenKey() returns the deterministic private key used by signer tests
func goldenKey() crypto.PrivateKeyI {
	// create a fixed Ed25519 key from a repeated-byte seed
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	return crypto.BytesToED25519Private(ed25519.NewKeyFromSeed(seed))
}
