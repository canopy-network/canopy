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
//   - Test accounts created with testPassword so keystore-get can decrypt them.
//   - Sign bytes = proto.Marshal(tx with nil Signature) — binary, deterministic.
//   - TX body = JSON with camelCase fields, base64-encoded []byte fields.
//   - BLS12-381 G2 (kyber/bdn) — 96-byte signatures matching Canopy verification.
//   - The "type" field in the tx JSON body must match ContractConfig.SupportedTransactions.

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
	queryRPCURL  = "http://localhost:50002"
	adminRPCURL  = "http://localhost:50003"
	testNetworkID = uint64(1)
	testChainID   = uint64(1)
	testPassword  = "testpassword123"
	testFee       = uint64(10000)
)

// typeURLShortName maps protobuf typeURLs to the short names in ContractConfig.
var typeURLShortName = map[string]string{
	"type.googleapis.com/types.MessageSend":             "send",
	"type.googleapis.com/types.MessageCreateMarket":     "create_market",
	"type.googleapis.com/types.MessageSubmitPrediction": "submit_prediction",
	"type.googleapis.com/types.MessageResolveMarket":    "resolve_market",
	"type.googleapis.com/types.MessageClaimWinnings":    "claim_winnings",
}

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
	// Response is a JSON-quoted hex string
	var addr string
	if err := json.Unmarshal(data, &addr); err != nil {
		addr = string(bytes.Trim(data, `"`+"\n\r "))
	}
	if len(addr) != 40 {
		return "", fmt.Errorf("unexpected address %q (len %d)", addr, len(addr))
	}
	return addr, nil
}

// keystoreGetKey decrypts and returns key material for address.
// Only works for keys created with a non-empty password.
func keystoreGetKey(address, password string) (*keyGroup, error) {
	body := fmt.Sprintf(`{"addressOrNickname":%q,"password":%q}`, address, password)
	data, err := postJSON(adminRPCURL+"/v1/admin/keystore-get", body)
	if err != nil {
		return nil, err
	}
	var kg keyGroup
	if err := json.Unmarshal(data, &kg); err != nil {
		return nil, fmt.Errorf("parse: %w (body: %s)", err, data)
	}
	if kg.PrivateKey == "" {
		return nil, fmt.Errorf("no private key returned (wrong password, or key created without one)")
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
// Sign bytes: proto.Marshal(Transaction{..., Signature: nil}) — binary, deterministic.
//
// The Canopy RPC JSON body format (from rpc_test.ts analysis):
//   {
//     "type": "send",                      // short name from ContractConfig
//     "msg": { "fromAddress": "<b64>", "toAddress": "<b64>", "amount": N },
//     "signature": { "publicKey": "<hex>", "signature": "<hex>" },
//     "time": N,
//     "createdHeight": N,
//     "fee": N,
//     "memo": "",
//     "networkID": N,
//     "chainID": N
//   }
//
// Note: For send (a registered Canopy base type), "msg" is inlined as JSON.
// For plugin-only types (create_market, etc.), use "msgTypeUrl" + "msgBytes" pattern.
func submitSendTx(t *testing.T, kg *keyGroup, from, to []byte, amount uint64, height uint64) string {
	t.Helper()

	typeURL := "type.googleapis.com/types.MessageSend"

	// Wrap in protobuf Any for sign-bytes construction
	sendMsg := &contract.MessageSend{
		FromAddress: from,
		ToAddress:   to,
		Amount:      amount,
	}
	anyMsg, err := anypb.New(sendMsg)
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}

	txTime := uint64(time.Now().UnixNano())

	// Build tx with nil Signature — this is what gets signed
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

	// Compute sign bytes (deterministic binary encoding)
	signBytes, err := proto.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal sign bytes: %v", err)
	}

	// Sign
	sig, pubKey, err := blsSign(kg.PrivateKey, signBytes)
	if err != nil {
		t.Fatalf("blsSign: %v", err)
	}

	// Build JSON body — base64 for []byte fields, hex for sig/pubkey
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

	data, err := postJSON(queryRPCURL+"/v1/tx", body)
	if err != nil {
		t.Fatalf("post /v1/tx: %v", err)
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
// TestSend
// ─────────────────────────────────────────────────────────────────────────────

func TestSend(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)

	// Step 1: Create two fresh test accounts with a password
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

	// Step 2: Retrieve sender key material
	senderKey, err := keystoreGetKey(senderAddr, testPassword)
	if err != nil {
		t.Fatalf("get sender key: %v", err)
	}
	t.Logf("PubKey: %s", senderKey.PublicKey)

	// Step 3: Get current height
	height, err := getHeight()
	if err != nil {
		t.Fatalf("get height: %v", err)
	}
	t.Logf("Height: %d", height)

	// Step 4: Decode addresses
	fromBytes, err := hex.DecodeString(senderAddr)
	if err != nil {
		t.Fatalf("decode sender addr: %v", err)
	}
	toBytes, err := hex.DecodeString(receiverAddr)
	if err != nil {
		t.Fatalf("decode receiver addr: %v", err)
	}

	// Step 5: Submit send transaction
	// Sender has 0 balance so DeliverTx will fail with ErrInsufficientFunds,
	// but this validates the full signing + RPC submission path.
	t.Log("Submitting send tx (sender has 0 balance — DeliverTx will fail, CheckTx should pass)...")
	txHash := submitSendTx(t, senderKey, fromBytes, toBytes, 1000000, height)
	t.Logf("TX: %s", txHash)

	// Step 6: Wait for inclusion
	t.Log("Waiting for inclusion (up to 60s)...")
	if err := waitForTx(senderAddr, txHash, 60*time.Second); err != nil {
		t.Logf("Note: %v", err)
	} else {
		t.Log("TX included!")
	}

	// Step 7: Report
	t.Logf("Sender balance:   %d", getBalance(senderAddr))
	t.Logf("Receiver balance: %d", getBalance(receiverAddr))
}
