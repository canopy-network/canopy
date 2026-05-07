package main

import (
	"bytes"
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

	contract "github.com/canopy-network/canopy/plugin/go/contract"
)

const (
	queryRPCURL   = "http://localhost:50002"
	adminRPCURL   = "http://localhost:50003"
	testNetworkID = uint64(1)
	testChainID   = uint64(1)
	testFee       = uint64(10000)
	testPassword  = "testpassword123"
)

type keyGroup struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

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

func keystoreNewKey(nickname, password string) (string, error) {
	body := fmt.Sprintf(`{"nickname":%q,"password":%q}`, nickname, password)
	data, err := postJSON(adminRPCURL+"/v1/admin/keystore-new-key", body)
	if err != nil {
		return "", err
	}
	return string(bytes.Trim(data, "\"\n\r")), nil
}

func keystoreGetKey(address, password string) (*keyGroup, error) {
	// Hardcoded validator key — avoids password issues with existing keys
	known := map[string]keyGroup{
		"e7c7dad131a03f7ea0cc09a637ad096eb3495f77": {
			Address:    "e7c7dad131a03f7ea0cc09a637ad096eb3495f77",
			PublicKey:  "ae13ea1c3a3a180b821b961561fedab3864fe037c7e159ef79c606c4399210f76f8bbb2ef7fe580c335a02cb48441b32",
			PrivateKey: "14f43ca8c7f31a63d144564e8826186383844b5da679dfc2c9352d665d69f0f6",
		},
	}
	if kg, ok := known[address]; ok {
		return &kg, nil
	}
	body := fmt.Sprintf(`{"addressOrNickname":%q,"password":%q}`, address, password)
	data, err := postJSON(adminRPCURL+"/v1/admin/keystore-get", body)
	if err != nil {
		return nil, err
	}
	var kg keyGroup
	json.Unmarshal(data, &kg)
	return &kg, nil
}

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

func getBalance(addr string) uint64 {
	data, _ := postJSON(queryRPCURL+"/v1/query/account",
		fmt.Sprintf(`{"address":%q}`, addr))
	var r struct{ Amount uint64 }
	json.Unmarshal(data, &r)
	return r.Amount
}

func waitForTx(sender, txHash string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, _ := postJSON(queryRPCURL+"/v1/query/txs-by-sender",
			fmt.Sprintf(`{"address":%q,"perPage":20}`, sender))
		var r struct {
			Results []struct {
				TxHash string `json:"txHash"`
			} `json:"results"`
		}
		json.Unmarshal(data, &r)
		for _, tx := range r.Results {
			if tx.TxHash == txHash {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("tx not confirmed")
}

// blsSign signs msg with privKeyHex using BLS12-381 G2 (long signatures).
// The kyber bdn scheme on G2 matches what Canopy nodes verify.
// IMPORTANT: privKeyHex must be exactly 32 bytes (64 hex chars).
func blsSign(t *testing.T, privKeyHex string, msg []byte) (sig, pubKey []byte, err error) {
	t.Helper()

	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("decode privkey: %w", err)
	}

	suite := bls12381.NewBLS12381Suite()
	scheme := bdn.NewSchemeOnG2(suite)

	// Reconstruct scalar from raw private key bytes
	scalar := suite.G1().Scalar()
	if err := scalar.UnmarshalBinary(privBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshal scalar: %w", err)
	}

	// Derive public key
	point := suite.G1().Point().Mul(scalar, nil)
	pubKey, err = point.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("marshal pubkey: %w", err)
	}

	t.Logf("pubKey len=%d hex=%x", len(pubKey), pubKey)

	// Sign — scheme uses G2 for signatures (96 bytes)
	sig, err = scheme.Sign(scalar, msg)
	if err != nil {
		return nil, nil, fmt.Errorf("sign: %w", err)
	}

	t.Logf("sig len=%d", len(sig))
	return sig, pubKey, nil
}

// submitSendTx handles the built-in send tx which uses msg field not msgTypeUrl/msgBytes
func submitSendTx(t *testing.T, kg *keyGroup, msg *contract.MessageSend, height uint64) string {
	t.Helper()

	typeURL := "type.googleapis.com/types.MessageSend"
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal msg: %v", err)
	}

	txTime := uint64(time.Now().UnixMicro())

	tx := &contract.Transaction{
		MessageType:   "send",
		Msg:           &anypb.Any{TypeUrl: typeURL, Value: msgBytes},
		Signature:     nil,
		CreatedHeight: height,
		Time:          txTime,
		Fee:           testFee,
		NetworkId:     testNetworkID,
		ChainId:       testChainID,
	}

	signBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal sign bytes: %v", err)
	}

	sig, pubKey, err := blsSign(t, kg.PrivateKey, signBytes)
	if err != nil {
		t.Fatalf("bls sign: %v", err)
	}

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
		hex.EncodeToString(msg.FromAddress),
		hex.EncodeToString(msg.ToAddress),
		msg.Amount,
		hex.EncodeToString(pubKey),
		hex.EncodeToString(sig),
		txTime, height, testFee,
		testNetworkID, testChainID,
	)

	data, err := postJSON(queryRPCURL+"/v1/tx", body)
	if err != nil {
		t.Fatalf("post tx: %v", err)
	}
	var hash string
	if err := json.Unmarshal(data, &hash); err != nil {
		t.Fatalf("parse hash: %v: %s", err, data)
	}
	return hash
}

// submitTx builds sign bytes by marshaling a Transaction proto with Signature=nil,
// signs it, then POSTs the JSON body to /v1/tx.
func submitTx(t *testing.T, kg *keyGroup, shortType, msgName string, msg proto.Message, height uint64) string {
	t.Helper()

	typeURL := "type.googleapis.com/types." + msgName

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal msg: %v", err)
	}
	t.Logf("msgBytes hex: %x", msgBytes)

	// Wrap in google.protobuf.Any
	msgAny := &anypb.Any{
		TypeUrl: typeURL,
		Value:   msgBytes,
	}

	// Compute txTime ONCE — same value must appear in sign bytes AND tx body
	txTime := uint64(time.Now().UnixMicro())

	// Build Transaction for sign bytes — Signature must be nil/omitted
	// CreatedHeight is included when non-zero (proto3 omits zero values)
	tx := &contract.Transaction{
		MessageType:   shortType,
		Msg:           msgAny,
		Signature:     nil, // MUST be nil for sign bytes
		CreatedHeight: height,
		Time:          txTime,
		Fee:           testFee,
		NetworkId:     testNetworkID,
		ChainId:       testChainID,
	}

	// Deterministic marshal — this is exactly what the node recomputes
	signBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal sign bytes: %v", err)
	}
	t.Logf("signBytes hex: %x", signBytes)

	sig, pubKey, err := blsSign(t, kg.PrivateKey, signBytes)
	if err != nil {
		t.Fatalf("bls sign: %v", err)
	}

	// POST as plugin tx format (msgTypeUrl + msgBytes)
	body := fmt.Sprintf(`{
  "type": %q,
  "msgTypeUrl": %q,
  "msgBytes": %q,
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
		shortType, typeURL, hex.EncodeToString(msgBytes),
		hex.EncodeToString(pubKey), hex.EncodeToString(sig),
		txTime, height, testFee,
		testNetworkID, testChainID,
	)

	t.Logf("TX body:\n%s", body)

	data, err := postJSON(queryRPCURL+"/v1/tx", body)
	if err != nil {
		t.Fatalf("post tx: %v", err)
	}
	hash := string(bytes.Trim(data, "\"\n\r"))
	t.Logf("TX hash: %s", hash)
	return hash
}

func hexDecode(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidatorSend(t *testing.T) {
	validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
	validatorKey, err := keystoreGetKey(validatorAddr, "")
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixMicro()%1000000)
	recipientAddr, err := keystoreNewKey("tst_s_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("new key: %v", err)
	}

	height, err := getHeight()
	if err != nil {
		t.Fatalf("height: %v", err)
	}
	t.Logf("height=%d recipientAddr=%s", height, recipientAddr)

	msg := &contract.MessageSend{
		FromAddress: hexDecode(validatorAddr),
		ToAddress:   hexDecode(recipientAddr),
		Amount:      1_000_000,
	}
	txHash := submitSendTx(t, validatorKey, msg, height)
	if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	if bal := getBalance(recipientAddr); bal != 1_000_000 {
		t.Errorf("expected balance 1000000, got %d", bal)
	}
	t.Log("Send confirmed!")
}

func TestCreateMarket(t *testing.T) {
	validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
	validatorKey, err := keystoreGetKey(validatorAddr, "")
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	height, err := getHeight()
	if err != nil {
		t.Fatalf("height: %v", err)
	}
	t.Logf("height=%d", height)

	msg := &contract.MessageCreateMarket{
		CreatorAddress: hexDecode(validatorAddr),
		B0:             contract.PRECISION_SCALE * 10, // 10_000_000
		ExpiryTime:     height + 1000,
		Nonce:          uint64(time.Now().UnixMicro()),
		Question:       "Will Praxis launch on mainnet?",
	}

	t.Logf("CreateMarket: B0=%d ExpiryTime=%d Nonce=%d", msg.B0, msg.ExpiryTime, msg.Nonce)

	txHash := submitTx(t, validatorKey, "create_market", "MessageCreateMarket", msg, height)
	if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	t.Log("Market created successfully!")
}


func TestSubmitPrediction(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil { t.Fatalf("key: %v", err) }

// 1. Create a market first
h, _ := getHeight()
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             1_000_000,
ExpiryTime:     h + 50000,
Nonce:          uint64(time.Now().UnixMicro()),
Question:       "Prediction test market",
}
marketHash := submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, marketHash, 60*time.Second); err != nil {
t.Fatalf("create market failed: %v", err)
}
t.Logf("Market created: %s", marketHash)

// 2. Get the market state to know the market ID

marketId := contract.DeriveMarketId(hexDecode(addr), createMsg.Nonce)
t.Logf("Derived market ID: %x", marketId)

// 3. Submit a YES prediction (1 share = 1 PRX precision unit)
h2, _ := getHeight()
predMsg := &contract.MessageSubmitPrediction{
MarketId: marketId,
BettorAddress: hexDecode(addr),
Outcome:  true,
Shares:   contract.PRECISION_SCALE,
MaxCost:  10_000_000, // enough for a small trade
}
txHash := submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsg, h2)
if err := waitForTx(addr, txHash, 60*time.Second); err != nil {
t.Fatalf("submit prediction failed: %v", err)
}
t.Log("Prediction submitted!")
}
