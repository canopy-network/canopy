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
data, _ := postJSON(queryRPCURL+"/v1/query/height", "{}")
var r struct{ Height uint64 }
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

func blsSign(privKeyHex string, msg []byte) (sig, pubKey []byte, err error) {
privBytes, _ := hex.DecodeString(privKeyHex)
suite := bls12381.NewBLS12381Suite()
scheme := bdn.NewSchemeOnG2(suite)
scalar := suite.G1().Scalar()
scalar.UnmarshalBinary(privBytes)
point := suite.G1().Point().Mul(scalar, nil)
pubKey, _ = point.MarshalBinary()
sig, err = scheme.Sign(scalar, msg)
return
}

func submitTx(t *testing.T, kg *keyGroup, msgType string, msg proto.Message, height uint64) string {
t.Helper()
typeURL := "type.googleapis.com/contract." + msgType

msgBytes, err := proto.Marshal(msg)
if err != nil {
t.Fatalf("marshal msg: %v", err)
}

txTime := uint64(time.Now().UnixMicro())

msgAny := &anypb.Any{TypeUrl: typeURL, Value: msgBytes}
	tx := &contract.Transaction{
		MessageType:   msgType,
		Msg:           msgAny,
		Signature:     nil,
		CreatedHeight: height,
		Time:          txTime,
		Fee:           testFee,
		NetworkId:     testNetworkID,
		ChainId:       testChainID,
	}

signBytes, err := proto.Marshal(tx)
if err != nil {
t.Fatalf("sign bytes: %v", err)
}

sig, pubKey, err := blsSign(kg.PrivateKey, signBytes)
if err != nil {
t.Fatalf("bls sign: %v", err)
}

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
msgType, typeURL, hex.EncodeToString(msgBytes),
hex.EncodeToString(pubKey), hex.EncodeToString(sig),
txTime, height, testFee,
testNetworkID, testChainID,
)

data, err := postJSON(queryRPCURL+"/v1/tx", body)
if err != nil {
t.Fatalf("post tx: %v", err)
}
return string(bytes.Trim(data, "\"\n\r"))
}

func TestValidatorSend(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, err := keystoreGetKey(validatorAddr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

suffix := fmt.Sprintf("%d", time.Now().UnixMicro()%1000000)
recipientAddr, err := keystoreNewKey("test_recv_"+suffix, testPassword)
if err != nil {
t.Fatalf("create recipient: %v", err)
}

height, err := getHeight()
if err != nil {
t.Fatalf("get height: %v", err)
}

from, _ := hex.DecodeString(validatorAddr)
to, _ := hex.DecodeString(recipientAddr)

msg := &contract.MessageSend{
FromAddress: from,
ToAddress:   to,
Amount:      1_000_000,
}

t.Logf("Submitting send (msgTypeUrl format)...")
txHash := submitTx(t, validatorKey, "MessageSend", msg, height)
t.Logf("TX hash: %s", txHash)

if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
t.Fatalf("tx not included: %v", err)
}
t.Log("TX included!")

bal := getBalance(recipientAddr)
if bal != 1_000_000 {
t.Errorf("expected balance 1000000, got %d", bal)
}
}


func TestCreateMarket(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, err := keystoreGetKey(validatorAddr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

height, err := getHeight()
if err != nil {
t.Fatalf("get height: %v", err)
}

// Use a fresh nonce each run
nonce := uint64(time.Now().UnixMicro())
expiry := height + 20000 // far enough in the future
b0 := contract.PRECISION_SCALE * 1000 // 1000 * 1e6 = 1,000,000,000
question := "Will the test pass?"

marketId := contract.DeriveMarketId(mustHex(validatorAddr), nonce)

msg := &contract.MessageCreateMarket{
CreatorAddress: mustHex(validatorAddr),
B0:             b0,
ExpiryTime:     expiry,
Nonce:          nonce,
Question:       question,
}

t.Logf("Creating market with ID %x...", marketId)
txHash := submitTx(t, validatorKey, "MessageCreateMarket", msg, height)
t.Logf("TX hash: %s", txHash)

if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
t.Fatalf("create_market tx not included: %v", err)
}
t.Log("Market created!")
}

// Helper to convert hex to bytes (existing code uses hex.DecodeString)
func mustHex(s string) []byte {
b, _ := hex.DecodeString(s)
return b
}
