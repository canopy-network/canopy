package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/base64"
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

// TestPluginTransactions tests the full flow of plugin transactions via RPC
// 1. Adds two accounts to the keystore
// 2. Uses faucet to add balance to one account
// 3. Does a send transaction from the fauceted account to the other account
// 4. Sends a reward from that account back to the original account
func TestPluginTransactions(t *testing.T) {
	// Configuration - adjust these for your local setup
	queryRPCURL := "http://localhost:50002" // Query endpoints (height, account, tx submission)
	adminRPCURL := "http://localhost:50003" // Admin endpoints (keystore management)
	networkID := uint64(1)
	chainID := uint64(1)
	testPassword := "testpassword123"

	// Step 1: Create two new accounts in the keystore
	t.Log("Step 1: Creating two accounts in keystore...")

	// Use random suffixes to avoid nickname conflicts from previous test runs
	suffix := randomSuffix()
	account1Addr, err := keystoreNewKey(adminRPCURL, "test_faucet_1_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("Failed to create account 1: %v", err)
	}
	t.Logf("Created account 1: %s", account1Addr)

	account2Addr, err := keystoreNewKey(adminRPCURL, "test_faucet_2_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("Failed to create account 2: %v", err)
	}
	t.Logf("Created account 2: %s", account2Addr)

	// Get current height for transaction
	height, err := getHeight(queryRPCURL)
	if err != nil {
		t.Fatalf("Failed to get height: %v", err)
	}
	t.Logf("Current height: %d", height)

	// Get account 1's key for signing
	account1Key, err := keystoreGetKey(adminRPCURL, account1Addr, testPassword)
	if err != nil {
		t.Fatalf("Failed to get account 1 key: %v", err)
	}

	// Step 2: Use faucet to add balance to account 1
	t.Log("Step 2: Using faucet to add balance to account 1...")

	faucetAmount := uint64(1000000000) // 1000 tokens
	faucetFee := uint64(10000)

	faucetTxHash, err := sendFaucetTx(queryRPCURL, account1Key, account1Addr, faucetAmount, faucetFee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send faucet transaction: %v", err)
	}
	t.Logf("Faucet transaction sent: %s", faucetTxHash)

	// Wait for faucet transaction to be included in a block
	t.Log("Waiting for faucet transaction to be confirmed...")
	included, err := waitForTxInclusion(queryRPCURL, account1Addr, faucetTxHash, 30*time.Second)
	if err != nil {
		t.Fatalf("Faucet transaction not included: %v", err)
	}
	if !included {
		t.Fatal("Faucet transaction not included within timeout")
	}
	t.Log("Faucet transaction confirmed!")

	// Verify no failed transactions
	failedCount, err := checkTxNotFailed(queryRPCURL, account1Addr)
	if err != nil {
		t.Logf("Warning: Could not check failed transactions: %v", err)
	} else if failedCount > 0 {
		t.Fatalf("Account 1 has %d failed transactions", failedCount)
	}

	// Print balances after faucet
	bal1, _ := getAccountBalance(queryRPCURL, account1Addr)
	bal2, _ := getAccountBalance(queryRPCURL, account2Addr)
	t.Logf("Balances after faucet - Account 1: %d, Account 2: %d", bal1, bal2)

	// Step 3: Send tokens from account 1 to account 2
	t.Log("Step 3: Sending tokens from account 1 to account 2...")

	sendAmount := uint64(100000000) // 100 tokens
	sendFee := uint64(10000)

	// Update height
	height, _ = getHeight(queryRPCURL)

	sendTxHash, err := sendSendTx(queryRPCURL, account1Key, account1Addr, account2Addr, sendAmount, sendFee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}
	t.Logf("Send transaction sent: %s", sendTxHash)

	// Wait for send transaction to be included
	t.Log("Waiting for send transaction to be confirmed...")
	included, err = waitForTxInclusion(queryRPCURL, account1Addr, sendTxHash, 30*time.Second)
	if err != nil {
		t.Fatalf("Send transaction not included: %v", err)
	}
	if !included {
		t.Fatal("Send transaction not included within timeout")
	}
	t.Log("Send transaction confirmed!")

	// Verify no failed transactions
	failedCount, err = checkTxNotFailed(queryRPCURL, account1Addr)
	if err != nil {
		t.Logf("Warning: Could not check failed transactions: %v", err)
	} else if failedCount > 0 {
		t.Fatalf("Account 1 has %d failed transactions", failedCount)
	}

	// Print balances after send
	bal1, _ = getAccountBalance(queryRPCURL, account1Addr)
	bal2, _ = getAccountBalance(queryRPCURL, account2Addr)
	t.Logf("Balances after send - Account 1: %d, Account 2: %d", bal1, bal2)

	// Step 4: Send reward from account 2 back to account 1
	t.Log("Step 4: Sending reward from account 2 back to account 1...")

	// Get account 2's key for signing
	account2Key, err := keystoreGetKey(adminRPCURL, account2Addr, testPassword)
	if err != nil {
		t.Fatalf("Failed to get account 2 key: %v", err)
	}

	rewardAmount := uint64(50000000) // 50 tokens
	rewardFee := uint64(10000)

	// Update height
	height, _ = getHeight(queryRPCURL)

	rewardTxHash, err := sendRewardTx(queryRPCURL, account2Key, account2Addr, account1Addr, rewardAmount, rewardFee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send reward transaction: %v", err)
	}
	t.Logf("Reward transaction sent: %s", rewardTxHash)

	// Wait for reward transaction to be included
	t.Log("Waiting for reward transaction to be confirmed...")
	included, err = waitForTxInclusion(queryRPCURL, account2Addr, rewardTxHash, 30*time.Second)
	if err != nil {
		t.Fatalf("Reward transaction not included: %v", err)
	}
	if !included {
		t.Fatal("Reward transaction not included within timeout")
	}
	t.Log("Reward transaction confirmed!")

	// Verify no failed transactions for account 2
	failedCount, err = checkTxNotFailed(queryRPCURL, account2Addr)
	if err != nil {
		t.Logf("Warning: Could not check failed transactions: %v", err)
	} else if failedCount > 0 {
		t.Fatalf("Account 2 has %d failed transactions", failedCount)
	}

	// Print final balances after reward
	bal1, _ = getAccountBalance(queryRPCURL, account1Addr)
	bal2, _ = getAccountBalance(queryRPCURL, account2Addr)
	t.Logf("Final balances - Account 1: %d, Account 2: %d", bal1, bal2)

	t.Log("All transactions confirmed successfully!")
}

// randomSuffix generates a random hex suffix for unique nicknames
func randomSuffix() string {
	b := make([]byte, 4)
	cryptorand.Read(b)
	return hex.EncodeToString(b)
}

// keyGroup holds key information from the keystore
type keyGroup struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// keystoreNewKey creates a new key in the keystore using raw JSON
func keystoreNewKey(rpcURL, nickname, password string) (string, error) {
	reqJSON := fmt.Sprintf(`{"nickname":"%s","password":"%s"}`, nickname, password)

	respBody, err := postRawJSON(rpcURL+"/v1/admin/keystore-new-key", reqJSON)
	if err != nil {
		return "", err
	}

	var address string
	if err := json.Unmarshal(respBody, &address); err != nil {
		return "", fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}

	return address, nil
}

// keystoreGetKey gets the key info from the keystore using raw JSON
func keystoreGetKey(rpcURL, address, password string) (*keyGroup, error) {
	reqJSON := fmt.Sprintf(`{"address":"%s","password":"%s"}`, address, password)

	respBody, err := postRawJSON(rpcURL+"/v1/admin/keystore-get", reqJSON)
	if err != nil {
		return nil, err
	}

	var kg keyGroup
	if err := json.Unmarshal(respBody, &kg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}

	return &kg, nil
}

// getHeight gets the current blockchain height using raw JSON
func getHeight(rpcURL string) (uint64, error) {
	respBody, err := postRawJSON(rpcURL+"/v1/query/height", "{}")
	if err != nil {
		return 0, err
	}

	var result struct {
		Height uint64 `json:"height"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Height, nil
}

// getAccountBalance gets the balance of an account using raw JSON
func getAccountBalance(rpcURL, address string) (uint64, error) {
	reqJSON := fmt.Sprintf(`{"address":"%s"}`, address)

	respBody, err := postRawJSON(rpcURL+"/v1/query/account", reqJSON)
	if err != nil {
		return 0, err
	}

	var result struct {
		Amount uint64 `json:"amount"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}

	return result.Amount, nil
}

// waitForTxInclusion waits for a transaction to be included in a block
func waitForTxInclusion(rpcURL, senderAddr, txHash string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Query transactions by sender
		reqJSON := fmt.Sprintf(`{"address":"%s","perPage":20}`, senderAddr)
		respBody, err := postRawJSON(rpcURL+"/v1/query/txs-by-sender", reqJSON)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		var result struct {
			Results []struct {
				TxHash string `json:"txHash"`
				Height uint64 `json:"height"`
			} `json:"results"`
			TotalCount int `json:"totalCount"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// Check if our transaction is in the results
		for _, tx := range result.Results {
			if tx.TxHash == txHash {
				return true, nil
			}
		}

		time.Sleep(1 * time.Second)
	}

	return false, fmt.Errorf("transaction %s not included within timeout", txHash)
}

// checkTxNotFailed verifies that a transaction is not in the failed transactions list
func checkTxNotFailed(rpcURL, senderAddr string) (int, error) {
	reqJSON := fmt.Sprintf(`{"address":"%s","perPage":20}`, senderAddr)
	respBody, err := postRawJSON(rpcURL+"/v1/query/failed-txs", reqJSON)
	if err != nil {
		return 0, err
	}

	var result struct {
		TotalCount int `json:"totalCount"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}

	return result.TotalCount, nil
}

// hexToBase64 converts a hex string to base64 (for protojson bytes encoding)
func hexToBase64(hexStr string) string {
	bytes, _ := hex.DecodeString(hexStr)
	return base64.StdEncoding.EncodeToString(bytes)
}

// sendFaucetTx sends a faucet transaction using raw JSON
func sendFaucetTx(rpcURL string, signerKey *keyGroup, recipientAddr string, amount, fee, networkID, chainID, height uint64) (string, error) {
	// Create the faucet message as JSON map
	// protojson expects base64 for bytes fields
	faucetMsg := map[string]interface{}{
		"signerAddress":    hexToBase64(signerKey.Address),
		"recipientAddress": hexToBase64(recipientAddr),
		"amount":           float64(amount),
	}

	return buildSignAndSendTx(rpcURL, signerKey, "faucet", faucetMsg, fee, networkID, chainID, height)
}

// sendSendTx sends a send transaction using raw JSON
func sendSendTx(rpcURL string, senderKey *keyGroup, fromAddr, toAddr string, amount, fee, networkID, chainID, height uint64) (string, error) {
	// Create the send message as JSON map
	// protojson expects base64 for bytes fields
	sendMsg := map[string]interface{}{
		"fromAddress": hexToBase64(fromAddr),
		"toAddress":   hexToBase64(toAddr),
		"amount":      float64(amount),
	}

	return buildSignAndSendTx(rpcURL, senderKey, "send", sendMsg, fee, networkID, chainID, height)
}

// sendRewardTx sends a reward transaction using raw JSON
func sendRewardTx(rpcURL string, adminKey *keyGroup, adminAddr, recipientAddr string, amount, fee, networkID, chainID, height uint64) (string, error) {
	// Create the reward message as JSON map
	// protojson expects base64 for bytes fields
	rewardMsg := map[string]interface{}{
		"adminAddress":     hexToBase64(adminAddr),
		"recipientAddress": hexToBase64(recipientAddr),
		"amount":           float64(amount),
	}

	return buildSignAndSendTx(rpcURL, adminKey, "reward", rewardMsg, fee, networkID, chainID, height)
}

// buildSignAndSendTx builds a transaction, signs it with BLS, and sends it via raw JSON
func buildSignAndSendTx(rpcURL string, signerKey *keyGroup, msgType string, msgJSON map[string]interface{}, fee, networkID, chainID, height uint64) (string, error) {
	// Build the transaction structure for signing (without signature)
	txTime := uint64(time.Now().UnixMicro())

	// For signing, we need to construct the Any message bytes
	// The server uses the registered plugin schema to convert JSON to protobuf
	// But for signing, we need to compute what the server will compute

	// Create sign bytes by first figuring out what the Any will look like
	// The plugin registry maps message type to type URL
	// For plugin messages, the type URL is: type.googleapis.com/types.Message<Type>
	var typeURL string
	switch msgType {
	case "send":
		typeURL = "type.googleapis.com/types.MessageSend"
	case "reward":
		typeURL = "type.googleapis.com/types.MessageReward"
	case "faucet":
		typeURL = "type.googleapis.com/types.MessageFaucet"
	default:
		return "", fmt.Errorf("unknown message type: %s", msgType)
	}

	// Marshal the message to proto bytes for signing
	// We need to create the actual proto message
	// Addresses in msgJSON are base64-encoded (for protojson compatibility)
	var msgProto proto.Message
	switch msgType {
	case "send":
		fromAddr, _ := base64.StdEncoding.DecodeString(msgJSON["fromAddress"].(string))
		toAddr, _ := base64.StdEncoding.DecodeString(msgJSON["toAddress"].(string))
		msgProto = &contract.MessageSend{
			FromAddress: fromAddr,
			ToAddress:   toAddr,
			Amount:      uint64(msgJSON["amount"].(float64)),
		}
	case "reward":
		adminAddr, _ := base64.StdEncoding.DecodeString(msgJSON["adminAddress"].(string))
		recipientAddr, _ := base64.StdEncoding.DecodeString(msgJSON["recipientAddress"].(string))
		msgProto = &contract.MessageReward{
			AdminAddress:     adminAddr,
			RecipientAddress: recipientAddr,
			Amount:           uint64(msgJSON["amount"].(float64)),
		}
	case "faucet":
		signerAddr, _ := base64.StdEncoding.DecodeString(msgJSON["signerAddress"].(string))
		recipientAddr, _ := base64.StdEncoding.DecodeString(msgJSON["recipientAddress"].(string))
		msgProto = &contract.MessageFaucet{
			SignerAddress:    signerAddr,
			RecipientAddress: recipientAddr,
			Amount:           uint64(msgJSON["amount"].(float64)),
		}
	}

	msgBytes, err := proto.Marshal(msgProto)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %v", err)
	}

	// Create the Any message for signing
	msgAny := &anypb.Any{
		TypeUrl: typeURL,
		Value:   msgBytes,
	}

	// Create sign bytes - this must match the server's GetSignBytes() exactly
	signBytes, err := crypto.GetSignBytes(msgType, msgAny, txTime, height, fee, "", networkID, chainID)
	if err != nil {
		return "", fmt.Errorf("failed to get sign bytes: %v", err)
	}

	// Get the BLS private key
	privKey, err := crypto.StringToBLS12381PrivateKey(signerKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Sign with BLS
	signature := privKey.Sign(signBytes)

	// Get public key bytes
	pubKeyBytes, err := hex.DecodeString(signerKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %v", err)
	}

	// Marshal the message to get the exact bytes for the Any.Value
	msgProtoBytes, err := proto.Marshal(msgProto)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message proto: %v", err)
	}

	// Build the transaction
	// For "send" (which is in RegisteredMessages), we must use "msg" field
	// For plugin-only types (faucet, reward), we use msgTypeUrl/msgBytes for exact byte control
	var tx map[string]interface{}
	if msgType == "send" {
		// "send" is in RegisteredMessages, must use msg field
		tx = map[string]interface{}{
			"type": msgType,
			"msg":  msgJSON,
			"signature": map[string]string{
				"publicKey": hex.EncodeToString(pubKeyBytes),
				"signature": hex.EncodeToString(signature),
			},
			"time":          txTime,
			"createdHeight": height,
			"fee":           fee,
			"memo":          "",
			"networkID":     networkID,
			"chainID":       chainID,
		}
	} else {
		// Plugin-only types: use msgTypeUrl/msgBytes for exact byte control
		tx = map[string]interface{}{
			"type":       msgType,
			"msgTypeUrl": typeURL,
			"msgBytes":   hex.EncodeToString(msgProtoBytes),
			"signature": map[string]string{
				"publicKey": hex.EncodeToString(pubKeyBytes),
				"signature": hex.EncodeToString(signature),
			},
			"time":          txTime,
			"createdHeight": height,
			"fee":           fee,
			"memo":          "",
			"networkID":     networkID,
			"chainID":       chainID,
		}
	}

	txJSONBytes, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %v", err)
	}

	// Send the transaction
	respBody, err := postRawJSON(rpcURL+"/v1/tx", string(txJSONBytes))
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	var txHash string
	if err := json.Unmarshal(respBody, &txHash); err != nil {
		return "", fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}

	return txHash, nil
}

// HTTP helper function
func postRawJSON(url string, jsonBody string) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
