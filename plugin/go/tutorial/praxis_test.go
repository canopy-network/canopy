package tutorial

// praxis_test.go — Praxis Prediction Market RPC Integration Test
//
// Tests the send transaction against a live local Canopy node.
// Run with: go test -v -run TestSend -timeout 120s
//
// Prerequisites:
//   - Canopy node running with the Praxis plugin enabled
//   - Plugin connected ("plugin connected" in node logs)
//   - Ports: 50002 (query RPC), 50003 (admin RPC)
//
// Key design decisions:
//   - Validator key (no password) is hardcoded — keystore-get returns {} for passwordless keys.
//   - New test accounts use testPassword so keystore-get can decrypt and return private key.
//   - Sign bytes = proto.Marshal(tx with nil Signature) — binary, deterministic.
//   - TX body = JSON matching TypeScript rpc_test.ts format confirmed by Canopy CEO Adam.
//   - BLS12-381 G2 (kyber/bdn) — 96-byte signatures matching Canopy verification.
//   - "type" field = short name from ContractConfig.SupportedTransactions.
//   - []byte fields in msg = base64. Signature fields = hex.
//   - Numeric fields (time, height, fee, networkID, chainID) = JSON numbers, not strings.

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	bls12381 "github.com/drand/kyber-bls12381"
	bdn "github.com/drand/kyber/sign/bdn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	contract "github.com/canopy-network/go-plugin/contract"
)

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

const (
	queryRPCURL   = "http://localhost:50002"
	adminRPCURL   = "http://localhost:50003"
	testNetworkID = uint64(1)
	testChainID   = uint64(1)
	testPassword  = "testpassword123"
	testFee       = uint64(10000)
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

type keyGroup struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP helpers
// ─────────────────────────────────────────────────────────────────────────────

func postJSON(url, body string) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, data)
	}
	return data, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Keystore helpers
// ─────────────────────────────────────────────────────────────────────────────

// keystoreNewKey creates a new encrypted key. Must use non-empty password
// so keystoreGetKey can later decrypt and return the private key.
func keystoreNewKey(nickname, password string) (string, error) {
	body := fmt.Sprintf(`{"nickname":%q,"password":%q}`, nickname, password)
	data, err := postJSON(adminRPCURL+"/v1/admin/keystore-new-key", body)
	if err != nil {
		return "", err
	}
	var addr string
	if err := json.Unmarshal(data, &addr); err != nil {
		addr = string(bytes.Trim(data, `"`+"\n\r "))
	}
	if len(addr) != 40 {
		return "", fmt.Errorf("unexpected address %q (len %d)", addr, len(addr))
	}
	return addr, nil
}

// keystoreGetKey returns key material for an address.
//
// The Canopy validator key is created without a password, so keystore-get
// returns {} for it. We hardcode the known validator private key here.
// For test keys created with testPassword, keystore-get works normally.
func keystoreGetKey(address, password string) (*keyGroup, error) {
	// Hardcoded no-password validator keys.
	// keystore-get returns {} for keys created without a password.
	knownKeys := map[string]keyGroup{
		"e7c7dad131a03f7ea0cc09a637ad096eb3495f77": {
			Address:    "e7c7dad131a03f7ea0cc09a637ad096eb3495f77",
			PublicKey:  "ae13ea1c3a3a180b821b961561fedab3864fe037c7e159ef79c606c4399210f76f8bbb2ef7fe580c335a02cb48441b32",
			PrivateKey: "14f43ca8c7f31a63d144564e8826186383844b5da679dfc2c9352d665d69f0f6",
		},
	}

	if kg, ok := knownKeys[address]; ok {
		return &kg, nil
	}

	// For test accounts created with a password, use keystore-get.
	body := fmt.Sprintf(`{"addressOrNickname":%q,"password":%q}`, address, password)
	data, err := postJSON(adminRPCURL+"/v1/admin/keystore-get", body)
	if err != nil {
		return nil, fmt.Errorf("keystore-get: %w", err)
	}
	var kg keyGroup
	if err := json.Unmarshal(data, &kg); err != nil {
		return nil, fmt.Errorf("parse keystore-get: %w (body: %s)", err, data)
	}
	if kg.PrivateKey == "" {
		return nil, fmt.Errorf("no private key for %s (wrong password, or key created without one)", address)
	}
	return &kg, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Chain query helpers
// ─────────────────────────────────────────────────────────────────────────────

func getHeight() (uint64, error) {
	data, err := postJSON(queryRPCURL+"/v1/query/height", "{}")
	if err != nil {
		return 0, err
	}
	var r struct {
		Height uint64 `json:"height"`
	}
	json.Unmarshal(data, &r)
	return r.Height, nil
}

func getBalance(address string) uint64 {
	data, _ := postJSON(queryRPCURL+"/v1/query/account",
		fmt.Sprintf(`{"address":%q}`, address))
	var r struct {
		Amount uint64 `json:"amount"`
	}
	json.Unmarshal(data, &r)
	return r.Amount
}

func waitForTx(senderAddr, txHash string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := postJSON(queryRPCURL+"/v1/query/txs-by-sender",
			fmt.Sprintf(`{"address":%q,"perPage":20}`, senderAddr))
		if err == nil {
			var r struct {
				Results []struct {
					TxHash string `json:"txHash"`
				} `json:"results"`
			}
			if json.Unmarshal(data, &r) == nil {
				for _, tx := range r.Results {
					if tx.TxHash == txHash {
						return nil
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("tx %s not confirmed within %v", txHash, timeout)
}

// ─────────────────────────────────────────────────────────────────────────────
// BLS12-381 signing — G2 long signatures (96 bytes, kyber/bdn)
// ─────────────────────────────────────────────────────────────────────────────

func blsSign(privKeyHex string, msg []byte) (sig, pubKey []byte, err error) {
	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("decode privkey: %w", err)
	}
	suite := bls12381.NewBLS12381Suite()
	scheme := bdn.NewSchemeOnG2(suite)

	scalar := suite.G1().Scalar()
	if err := scalar.UnmarshalBinary(privBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshal scalar: %w", err)
	}
	point := suite.G1().Point().Mul(scalar, nil)
	pubKey, err = point.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	sig, err = scheme.Sign(scalar, msg)
	return sig, pubKey, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Transaction builder
// ─────────────────────────────────────────────────────────────────────────────

// submitSendTx builds, signs, and submits a MessageSend transaction.
//
// Sign bytes rule: proto.Marshal(Transaction with Signature = nil).
// This is binary protobuf, deterministic. NEVER sign JSON.
//
// TX body format (confirmed from TypeScript rpc_test.ts, Canopy CEO Adam):
//
//	{
//	  "type": "send",
//	  "msg": { "fromAddress": "<base64>", "toAddress": "<base64>", "amount": <number> },
//	  "signature": { "publicKey": "<hex>", "signature": "<hex>" },
//	  "time": <number>,          // nanoseconds (UnixNano)
//	  "createdHeight": <number>,
//	  "fee": <number>,
//	  "memo": "",
//	  "networkID": <number>,     // capital ID
//	  "chainID": <number>        // capital ID
//	}
func submitSendTx(t *testing.T, kg *keyGroup, from, to []byte, amount uint64, height uint64) string {
	t.Helper()

	typeURL := "type.googleapis.com/types.MessageSend"

	// Build the inner message and wrap in Any for sign-bytes construction.
	sendMsg := &contract.MessageSend{
		FromAddress: from,
		ToAddress:   to,
		Amount:      amount,
	}
	anyMsg, err := anypb.New(sendMsg)
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}

	// txTime in nanoseconds — same value used for both sign bytes and JSON body.
	txTime := uint64(time.Now().UnixNano())

	// Build Transaction with nil Signature — this is the sign-bytes input.
	tx := &contract.Transaction{
		MessageType:   typeURL,
		Msg:           anyMsg,
		Signature:     nil,
		CreatedHeight: height,
		Time:          txTime,
		Fee:           testFee,
		NetworkId:     testNetworkID,
		ChainId:       testChainID,
	}

	// Sign bytes = deterministic binary proto encoding.
	signBytes, err := proto.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal sign bytes: %v", err)
	}

	// BLS12-381 G2 longSignatures — 96-byte output.
	sig, pubKey, err := blsSign(kg.PrivateKey, signBytes)
	if err != nil {
		t.Fatalf("blsSign: %v", err)
	}

	// JSON body: []byte fields as base64, signature fields as hex, numerics as numbers.
	fromB64 := base64.StdEncoding.EncodeToString(from)
	toB64 := base64.StdEncoding.EncodeToString(to)

	body := fmt.Sprintf(`{
		"type": "send",
		"msg": {
			"fromAddress": %q,
			"toAddress": %q,
			"amount": %d
		},
		"signature": {
			"publicKey": %q,
			"signature": %q
		},
		"time": %d,
		"createdHeight": %d,
		"fee": %d,
		"memo": "",
		"networkID": %d,
		"chainID": %d
	}`,
		fromB64, toB64, amount,
		hex.EncodeToString(pubKey),
		hex.EncodeToString(sig),
		txTime, height, testFee,
		testNetworkID, testChainID,
	)

	t.Logf("TX body: %s", body)

	data, err := postJSON(queryRPCURL+"/v1/tx", body)
	if err != nil {
		t.Fatalf("post tx: %v", err)
	}

	var txHash string
	if err := json.Unmarshal(data, &txHash); err != nil {
		txHash = string(bytes.Trim(data, `"`+"\n\r "))
	}
	if txHash == "" {
		t.Fatalf("empty txHash, response: %s", data)
	}
	return txHash
}

// ─────────────────────────────────────────────────────────────────────────────
// TestSend — validates the full signing + RPC submission path
// ─────────────────────────────────────────────────────────────────────────────

func TestSend(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)

	// Step 1: Create two fresh test accounts with a password.
	t.Logf("Creating accounts (suffix=%s)...", suffix)
	senderAddr, err := keystoreNewKey("praxis_s_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("create sender: %v", err)
	}
	receiverAddr, err := keystoreNewKey("praxis_r_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("create receiver: %v", err)
	}
	t.Logf("Sender:   %s", senderAddr)
	t.Logf("Receiver: %s", receiverAddr)

	// Step 2: Retrieve sender key material (password-protected, keystore-get works).
	senderKey, err := keystoreGetKey(senderAddr, testPassword)
	if err != nil {
		t.Fatalf("get sender key: %v", err)
	}
	t.Logf("PubKey: %s", senderKey.PublicKey)

	// Step 3: Get current height.
	height, err := getHeight()
	if err != nil {
		t.Fatalf("get height: %v", err)
	}
	t.Logf("Height: %d", height)

	// Step 4: Decode addresses from hex to bytes.
	fromBytes, err := hex.DecodeString(senderAddr)
	if err != nil {
		t.Fatalf("decode sender addr: %v", err)
	}
	toBytes, err := hex.DecodeString(receiverAddr)
	if err != nil {
		t.Fatalf("decode receiver addr: %v", err)
	}

	// Step 5: Submit send transaction.
	// Sender has 0 balance — DeliverTx will fail with ErrInsufficientFunds,
	// but this validates the full signing + RPC submission path end-to-end.
	t.Log("Submitting send tx (0 balance — DeliverTx expected fail, CheckTx must pass)...")
	txHash := submitSendTx(t, senderKey, fromBytes, toBytes, 1000000, height)
	t.Logf("TX hash: %s", txHash)

	// Step 6: Wait for block inclusion (up to 60s).
	t.Log("Waiting for tx inclusion...")
	if err := waitForTx(senderAddr, txHash, 60*time.Second); err != nil {
		t.Logf("Note: %v", err)
	} else {
		t.Log("TX included in block!")
	}

	// Step 7: Report final balances.
	t.Logf("Sender balance:   %d", getBalance(senderAddr))
	t.Logf("Receiver balance: %d", getBalance(receiverAddr))
}

// ─────────────────────────────────────────────────────────────────────────────
// TestValidatorSend — sends from the funded validator account to test full flow
// ─────────────────────────────────────────────────────────────────────────────

func TestValidatorSend(t *testing.T) {
	// Use the hardcoded validator key (no password, keystore-get returns {}).
	validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
	validatorKey, err := keystoreGetKey(validatorAddr, "")
	if err != nil {
		t.Fatalf("get validator key: %v", err)
	}
	t.Logf("Validator pubkey: %s", validatorKey.PublicKey)

	// Create a recipient account.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	recipientAddr, err := keystoreNewKey("praxis_recv_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("create recipient: %v", err)
	}
	t.Logf("Recipient: %s", recipientAddr)

	height, err := getHeight()
	if err != nil {
		t.Fatalf("get height: %v", err)
	}

	fromBytes, err := hex.DecodeString(validatorAddr)
	if err != nil {
		t.Fatalf("decode validator addr: %v", err)
	}
	toBytes, err := hex.DecodeString(recipientAddr)
	if err != nil {
		t.Fatalf("decode recipient addr: %v", err)
	}

	// Send 1,000,000 units from validator to recipient.
	sendAmount := uint64(1_000_000)
	t.Logf("Sending %d from validator to recipient...", sendAmount)
	txHash := submitSendTx(t, validatorKey, fromBytes, toBytes, sendAmount, height)
	t.Logf("TX hash: %s", txHash)

	t.Log("Waiting for tx inclusion...")
	if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
		t.Fatalf("tx not included: %v", err)
	}
	t.Log("TX included!")

	recipientBal := getBalance(recipientAddr)
	t.Logf("Recipient balance after send: %d", recipientBal)
	if recipientBal != sendAmount {
		t.Errorf("expected recipient balance %d, got %d", sendAmount, recipientBal)
	}
}

