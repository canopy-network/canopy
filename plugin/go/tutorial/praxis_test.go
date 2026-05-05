package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/canopy-network/go-plugin/tutorial/contract"
	"github.com/canopy-network/go-plugin/tutorial/crypto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	queryRPCURL  = "http://localhost:50002"
	adminRPCURL  = "http://localhost:50002"
	networkID    = uint64(1)
	chainID      = uint64(1)
	testPassword = "testpassword123"
	txFee        = uint64(10000)
	txTimeout    = 45 * time.Second
)

type keyGroup struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func TestSend(t *testing.T) {
	validatorKey, err := keystoreGetKey(adminRPCURL, "validator", "")
	if err != nil {
		t.Fatalf("get validator key: %v", err)
	}
	suffix := randomSuffix()
	recipientAddr, err := keystoreNewKey(adminRPCURL, "recv_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("new key: %v", err)
	}
	height, _ := getHeight(queryRPCURL)
	hash, err := sendTx(queryRPCURL, validatorKey, validatorKey.Address, recipientAddr, 10_000_000, txFee, height)
	if err != nil {
		t.Fatalf("sendTx: %v", err)
	}
	mustConfirm(t, queryRPCURL, validatorKey.Address, hash, "send")
	bal, _ := getAccountBalance(queryRPCURL, recipientAddr)
	if bal != 10_000_000 {
		t.Fatalf("expected 10000000 got %d", bal)
	}
	t.Logf("recipient balance: %d — PASS", bal)
}

func TestCreateMarket(t *testing.T) {
	validatorKey, err := keystoreGetKey(adminRPCURL, "validator", "")
	if err != nil {
		t.Fatalf("get validator key: %v", err)
	}
	suffix := randomSuffix()
	creatorAddr, err := keystoreNewKey(adminRPCURL, "creator_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("new key: %v", err)
	}
	height, _ := getHeight(queryRPCURL)
	hash, err := sendTx(queryRPCURL, validatorKey, validatorKey.Address, creatorAddr, 500_000_000_000, txFee, height)
	if err != nil {
		t.Fatalf("fund creator: %v", err)
	}
	mustConfirm(t, queryRPCURL, validatorKey.Address, hash, "fund creator")

	creatorKey, err := keystoreGetKey(adminRPCURL, creatorAddr, testPassword)
	if err != nil {
		t.Fatalf("get creator key: %v", err)
	}
	creatorAddrBytes, _ := hex.DecodeString(creatorAddr)
	height, _ = getHeight(queryRPCURL)
	nonce := uint64(1)
	marketId := deriveMarketId(creatorAddrBytes, nonce)
	msg := &contract.MessageCreateMarket{
		CreatorAddress: creatorAddrBytes,
		B0:             10_000_000,
		ExpiryTime:     height + 500,
		Nonce:          nonce,
		Question:       "Test market " + suffix,
	}
	hash, err = sendProtoTx(queryRPCURL, creatorKey, "create_market",
		"type.googleapis.com/types.MessageCreateMarket", msg, txFee, height)
	if err != nil {
		t.Fatalf("create_market tx: %v", err)
	}
	mustConfirm(t, queryRPCURL, creatorAddr, hash, "create_market")
	t.Logf("market_id: %s — PASS", hex.EncodeToString(marketId))
}

func mustConfirm(t *testing.T, rpcURL, senderAddr, hash, label string) {
	t.Helper()
	included, err := waitForTxInclusion(rpcURL, senderAddr, hash, txTimeout)
	if err != nil {
		t.Fatalf("%s: inclusion error: %v", label, err)
	}
	if !included {
		t.Fatalf("%s: not included within timeout", label)
	}
	failed, err := checkTxNotFailed(rpcURL, senderAddr)
	if err != nil {
		t.Logf("warning: checkTxNotFailed for %s: %v", label, err)
	} else if failed > 0 {
		t.Fatalf("%s: %d failed transactions found", label, failed)
	}
	t.Logf("%s confirmed ✓", label)
}

func sendTx(rpcURL string, signerKey *keyGroup, fromAddr, toAddr string, amount, fee, height uint64) (string, error) {
	fromBytes, _ := hex.DecodeString(fromAddr)
	toBytes, _ := hex.DecodeString(toAddr)
	msg := &contract.MessageSend{
		FromAddress: fromBytes,
		ToAddress:   toBytes,
		Amount:      amount,
	}
	return sendProtoTx(rpcURL, signerKey, "send",
		"type.googleapis.com/types.MessageSend", msg, fee, height)
}

func sendProtoTx(rpcURL string, signerKey *keyGroup, msgType, typeURL string, msg proto.Message, fee, height uint64) (string, error) {
	txTime := uint64(time.Now().UnixMicro())
	msgBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal msg: %v", err)
	}
	msgAny := &anypb.Any{TypeUrl: typeURL, Value: msgBytes}
	signBytes, err := crypto.GetSignBytes(msgType, msgAny, txTime, height, fee, "", networkID, chainID)
	if err != nil {
		return "", fmt.Errorf("GetSignBytes: %v", err)
	}
	privKey, err := crypto.StringToBLS12381PrivateKey(signerKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("parse privkey: %v", err)
	}
	sig := privKey.Sign(signBytes)
	pubKeyBytes, err := hex.DecodeString(signerKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("decode pubkey: %v", err)
	}
	tx := map[string]interface{}{
		"type":       msgType,
		"msgTypeUrl": typeURL,
		"msgBytes":   hex.EncodeToString(msgBytes),
		"signature": map[string]string{
			"publicKey": hex.EncodeToString(pubKeyBytes),
			"signature": hex.EncodeToString(sig),
		},
		"time":          txTime,
		"createdHeight": height,
		"fee":           fee,
		"memo":          "",
		"networkID":     networkID,
		"chainID":       chainID,
	}
	txJSON, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("marshal tx: %v", err)
	}
	respBody, err := postJSON(rpcURL+"/v1/tx", string(txJSON))
	if err != nil {
		return "", fmt.Errorf("post tx: %v", err)
	}
	var txHash string
	if err := json.Unmarshal(respBody, &txHash); err != nil {
		return "", fmt.Errorf("parse response: %v body: %s", err, string(respBody))
	}
	return txHash, nil
}

func deriveMarketId(creatorAddr []byte, nonce uint64) []byte {
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	input := append(append([]byte{}, creatorAddr...), nonceBytes...)
	hash := sha256.Sum256(input)
	return hash[:20]
}

func randomSuffix() string {
	b := make([]byte, 4)
	cryptorand.Read(b)
	return hex.EncodeToString(b)
}

func postJSON(url, body string) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func keystoreNewKey(rpcURL, nickname, password string) (string, error) {
	body := fmt.Sprintf(`{"nickname":%q,"password":%q}`, nickname, password)
	resp, err := postJSON(rpcURL+"/v1/admin/keystore-new-key", body)
	if err != nil {
		return "", err
	}
	var addr string
	if err := json.Unmarshal(resp, &addr); err != nil {
		return "", fmt.Errorf("parse: %v body: %s", err, resp)
	}
	return addr, nil
}

func keystoreGetKey(rpcURL, address, password string) (*keyGroup, error) {
	body := fmt.Sprintf(`{"address":%q,"password":%q}`, address, password)
	resp, err := postJSON(rpcURL+"/v1/admin/keystore-get", body)
	if err != nil {
		return nil, err
	}
	var kg keyGroup
	if err := json.Unmarshal(resp, &kg); err != nil {
		return nil, fmt.Errorf("parse: %v body: %s", err, resp)
	}
	return &kg, nil
}

func getHeight(rpcURL string) (uint64, error) {
	resp, err := postJSON(rpcURL+"/v1/query/height", "{}")
	if err != nil {
		return 0, err
	}
	var r struct {
		Height uint64 `json:"height"`
	}
	json.Unmarshal(resp, &r)
	return r.Height, nil
}

func getAccountBalance(rpcURL, address string) (uint64, error) {
	body := fmt.Sprintf(`{"address":%q}`, address)
	resp, err := postJSON(rpcURL+"/v1/query/account", body)
	if err != nil {
		return 0, err
	}
	var r struct {
		Amount uint64 `json:"amount"`
	}
	json.Unmarshal(resp, &r)
	return r.Amount, nil
}

func waitForTxInclusion(rpcURL, senderAddr, txHash string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		body := fmt.Sprintf(`{"address":%q,"perPage":20}`, senderAddr)
		resp, err := postJSON(rpcURL+"/v1/query/txs-by-sender", body)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		var r struct {
			Results []struct {
				TxHash string `json:"txHash"`
			} `json:"results"`
		}
		if json.Unmarshal(resp, &r) == nil {
			for _, tx := range r.Results {
				if tx.TxHash == txHash {
					return true, nil
				}
			}
		}
		time.Sleep(time.Second)
	}
	return false, fmt.Errorf("tx %s not included within timeout", txHash)
}

func checkTxNotFailed(rpcURL, senderAddr string) (int, error) {
	body := fmt.Sprintf(`{"address":%q,"perPage":20}`, senderAddr)
	resp, err := postJSON(rpcURL+"/v1/query/failed-txs", body)
	if err != nil {
		return 0, err
	}
	var r struct {
		TotalCount int `json:"totalCount"`
	}
	json.Unmarshal(resp, &r)
	return r.TotalCount, nil
}
