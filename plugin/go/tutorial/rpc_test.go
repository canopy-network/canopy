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
	"strings"
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

	// Print tip about verifying balances via RPC
	t.Log("")
	t.Log("--- Verify Account Balances ---")
	t.Log("You can manually check account balances at any time using the /v1/query/account RPC endpoint:")
	t.Logf(`  curl -X POST %s/v1/query/account -H "Content-Type: application/json" -d '{"address": "%s"}'`, queryRPCURL, account1Addr)
	t.Logf(`  curl -X POST %s/v1/query/account -H "Content-Type: application/json" -d '{"address": "%s"}'`, queryRPCURL, account2Addr)
	t.Log("See documentation: https://github.com/canopy-network/canopy/blob/main/cmd/rpc/README.md#account")
}

// TestPluginCustomRPCEndpoints tests the plugin's own custom RPC endpoints (/v1/query/faucets and
// /v1/query/rewards), which are served by the plugin process (default 0.0.0.0:50010) and backed by
// the detached, read-only QueryState path. It:
//  1. Creates a recipient and an admin account
//  2. Faucets the recipient twice and asserts the faucet record aggregates (totalAmount + count)
//  3. Faucets the admin so it can pay reward fees
//  4. Rewards the recipient and asserts the reward record (totalAmount, count, lastAdminAddress)
//  5. Asserts both records also appear in the list (range-read) endpoints
func TestPluginCustomRPCEndpoints(t *testing.T) {
	// Configuration - adjust these for your local setup
	queryRPCURL := "http://localhost:50002"  // Canopy query/tx RPC
	adminRPCURL := "http://localhost:50003"  // Canopy admin RPC (keystore)
	pluginRPCURL := "http://localhost:50010" // the plugin's own custom RPC server
	networkID := uint64(1)
	chainID := uint64(1)
	testPassword := "testpassword123"

	// Generous timeouts: a dev node can take many blocks to finalize/index a tx (especially right
	// after a restart while consensus warms up), so don't fail on transient finalization lag.
	// Run with a matching overall budget, e.g. `go test -run TestPluginCustomRPCEndpoints -timeout 600s`.
	txTimeout := 120 * time.Second    // max wait for a tx to be committed + indexed
	recordTimeout := 60 * time.Second // max wait for the plugin RPC record to reflect committed state

	// Skip early with a helpful message if the plugin RPC server isn't reachable
	t.Logf("Checking plugin RPC server reachability at %s ...", pluginRPCURL)
	if _, err := getRawJSON(pluginRPCURL + "/v1/query/faucets"); err != nil {
		t.Skipf("plugin RPC server not reachable at %s (is Canopy running with the go plugin and port 50010 exposed?): %v", pluginRPCURL, err)
	}
	t.Log("Plugin RPC server is reachable")

	// Step 1: Create a recipient and an admin account in the keystore
	t.Log("Step 1: Creating recipient and admin accounts in keystore...")

	// Use random suffixes to avoid nickname conflicts from previous test runs
	suffix := randomSuffix()
	recipientAddr, err := keystoreNewKey(adminRPCURL, "test_rpc_recipient_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("Failed to create recipient account: %v", err)
	}
	t.Logf("Created recipient account: %s", recipientAddr)

	adminAddr, err := keystoreNewKey(adminRPCURL, "test_rpc_admin_"+suffix, testPassword)
	if err != nil {
		t.Fatalf("Failed to create admin account: %v", err)
	}
	t.Logf("Created admin account: %s", adminAddr)

	// Fetch signing keys for both accounts
	recipientKey, err := keystoreGetKey(adminRPCURL, recipientAddr, testPassword)
	if err != nil {
		t.Fatalf("Failed to get recipient key: %v", err)
	}
	adminKey, err := keystoreGetKey(adminRPCURL, adminAddr, testPassword)
	if err != nil {
		t.Fatalf("Failed to get admin key: %v", err)
	}
	t.Log("Fetched signing keys for both accounts")

	fee := uint64(10000)

	// Step 2: Faucet the recipient twice and verify the aggregated faucet record
	faucetAmount1 := uint64(700000000)
	faucetAmount2 := uint64(300000000)

	t.Logf("Step 2: Sending first faucet to recipient (amount=%d, fee=%d)...", faucetAmount1, fee)
	height, _ := getHeight(queryRPCURL)
	t.Logf("Current height: %d", height)
	txHash, err := sendFaucetTx(queryRPCURL, recipientKey, recipientAddr, faucetAmount1, fee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send first faucet tx: %v", err)
	}
	t.Logf("First faucet transaction sent: %s", txHash)

	t.Log("Waiting for first faucet transaction to be confirmed...")
	if ok, err := waitForTxInclusion(queryRPCURL, recipientAddr, txHash, txTimeout); err != nil || !ok {
		t.Fatalf("First faucet tx not included: ok=%v err=%v", ok, err)
	}
	t.Log("First faucet transaction confirmed!")

	// after the first faucet, the record should exist with count=1 and totalAmount=faucetAmount1
	t.Logf("Querying plugin endpoint GET /v1/query/faucets?address=%s ...", recipientAddr)
	faucet, err := pollFaucetRecord(pluginRPCURL, recipientAddr, recordTimeout)
	if err != nil {
		t.Fatalf("Failed to query faucet record after first faucet: %v", err)
	}
	t.Logf("Faucet record after first faucet: recipient=%s totalAmount=%d count=%d", faucet.RecipientAddress, faucet.TotalAmount, faucet.Count)
	if faucet.Count != 1 {
		t.Errorf("faucet count after first faucet = %d, want 1", faucet.Count)
	}
	if faucet.TotalAmount != faucetAmount1 {
		t.Errorf("faucet totalAmount after first faucet = %d, want %d", faucet.TotalAmount, faucetAmount1)
	}
	if !strings.EqualFold(faucet.RecipientAddress, recipientAddr) {
		t.Errorf("faucet recipientAddress = %s, want %s", faucet.RecipientAddress, recipientAddr)
	}
	t.Log("Faucet record verified after first faucet (count=1)")

	t.Logf("Sending second faucet to recipient (amount=%d, fee=%d)...", faucetAmount2, fee)
	height, _ = getHeight(queryRPCURL)
	t.Logf("Current height: %d", height)
	txHash, err = sendFaucetTx(queryRPCURL, recipientKey, recipientAddr, faucetAmount2, fee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send second faucet tx: %v", err)
	}
	t.Logf("Second faucet transaction sent: %s", txHash)

	t.Log("Waiting for second faucet transaction to be confirmed...")
	if ok, err := waitForTxInclusion(queryRPCURL, recipientAddr, txHash, txTimeout); err != nil || !ok {
		t.Fatalf("Second faucet tx not included: ok=%v err=%v", ok, err)
	}
	t.Log("Second faucet transaction confirmed!")

	// after the second faucet, count should be 2 and totalAmount the sum of both
	wantTotal := faucetAmount1 + faucetAmount2
	t.Log("Waiting for faucet record to reflect the second faucet (count=2)...")
	faucet, err = pollFaucetRecordUntil(pluginRPCURL, recipientAddr, 2, recordTimeout)
	if err != nil {
		t.Fatalf("Failed to query faucet record after second faucet: %v", err)
	}
	t.Logf("Faucet record after second faucet: recipient=%s totalAmount=%d count=%d", faucet.RecipientAddress, faucet.TotalAmount, faucet.Count)
	if faucet.Count != 2 {
		t.Errorf("faucet count after second faucet = %d, want 2", faucet.Count)
	}
	if faucet.TotalAmount != wantTotal {
		t.Errorf("faucet totalAmount after second faucet = %d, want %d", faucet.TotalAmount, wantTotal)
	}
	t.Logf("Faucet record aggregation verified (totalAmount=%d, count=2)", wantTotal)

	// the recipient should also appear in the list (range-read) endpoint
	t.Log("Querying plugin endpoint GET /v1/query/faucets (list / range read)...")
	faucets, err := queryFaucetList(pluginRPCURL)
	if err != nil {
		t.Fatalf("Failed to query faucet list: %v", err)
	}
	t.Logf("Faucet list returned %d record(s)", len(faucets))
	if vErr := validateFaucetRecords(faucets); vErr != nil {
		t.Fatalf("faucet list structural validation failed: %v", vErr)
	}
	t.Logf("All %d faucet list record(s) are structurally valid", len(faucets))
	if found := findFaucet(faucets, recipientAddr); found == nil {
		t.Errorf("recipient %s not found in faucet list endpoint", recipientAddr)
	} else if found.TotalAmount != wantTotal {
		t.Errorf("faucet list totalAmount = %d, want %d", found.TotalAmount, wantTotal)
	} else {
		t.Logf("Recipient found in faucet list with totalAmount=%d", found.TotalAmount)
	}

	// Step 3: Faucet the admin so it has balance to pay the reward fee
	t.Log("Step 3: Funding admin via faucet so it can pay the reward fee...")
	height, _ = getHeight(queryRPCURL)
	t.Logf("Current height: %d", height)
	txHash, err = sendFaucetTx(queryRPCURL, adminKey, adminAddr, uint64(100000000), fee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to faucet admin: %v", err)
	}
	t.Logf("Admin faucet transaction sent: %s", txHash)

	t.Log("Waiting for admin faucet transaction to be confirmed...")
	if ok, err := waitForTxInclusion(queryRPCURL, adminAddr, txHash, txTimeout); err != nil || !ok {
		t.Fatalf("Admin faucet tx not included: ok=%v err=%v", ok, err)
	}
	t.Log("Admin faucet transaction confirmed!")

	// Step 4: Reward the recipient and verify the reward record
	rewardAmount := uint64(50000000)
	t.Logf("Step 4: Sending reward from admin to recipient (amount=%d, fee=%d)...", rewardAmount, fee)
	height, _ = getHeight(queryRPCURL)
	t.Logf("Current height: %d", height)
	txHash, err = sendRewardTx(queryRPCURL, adminKey, adminAddr, recipientAddr, rewardAmount, fee, networkID, chainID, height)
	if err != nil {
		t.Fatalf("Failed to send reward tx: %v", err)
	}
	t.Logf("Reward transaction sent: %s", txHash)

	t.Log("Waiting for reward transaction to be confirmed...")
	if ok, err := waitForTxInclusion(queryRPCURL, adminAddr, txHash, txTimeout); err != nil || !ok {
		t.Fatalf("Reward tx not included: ok=%v err=%v", ok, err)
	}
	t.Log("Reward transaction confirmed!")

	t.Logf("Querying plugin endpoint GET /v1/query/rewards?address=%s ...", recipientAddr)
	reward, err := pollRewardRecord(pluginRPCURL, recipientAddr, recordTimeout)
	if err != nil {
		t.Fatalf("Failed to query reward record: %v", err)
	}
	t.Logf("Reward record: recipient=%s lastAdmin=%s totalAmount=%d count=%d", reward.RecipientAddress, reward.LastAdminAddress, reward.TotalAmount, reward.Count)
	if reward.Count != 1 {
		t.Errorf("reward count = %d, want 1", reward.Count)
	}
	if reward.TotalAmount != rewardAmount {
		t.Errorf("reward totalAmount = %d, want %d", reward.TotalAmount, rewardAmount)
	}
	if !strings.EqualFold(reward.RecipientAddress, recipientAddr) {
		t.Errorf("reward recipientAddress = %s, want %s", reward.RecipientAddress, recipientAddr)
	}
	if !strings.EqualFold(reward.LastAdminAddress, adminAddr) {
		t.Errorf("reward lastAdminAddress = %s, want %s", reward.LastAdminAddress, adminAddr)
	}
	t.Log("Reward record verified (count=1, correct amount and lastAdminAddress)")

	// the recipient should also appear in the reward list (range-read) endpoint
	t.Log("Querying plugin endpoint GET /v1/query/rewards (list / range read)...")
	rewards, err := queryRewardList(pluginRPCURL)
	if err != nil {
		t.Fatalf("Failed to query reward list: %v", err)
	}
	t.Logf("Reward list returned %d record(s)", len(rewards))
	if vErr := validateRewardRecords(rewards); vErr != nil {
		t.Fatalf("reward list structural validation failed: %v", vErr)
	}
	t.Logf("All %d reward list record(s) are structurally valid", len(rewards))
	if found := findReward(rewards, recipientAddr); found == nil {
		t.Errorf("recipient %s not found in reward list endpoint", recipientAddr)
	} else if found.TotalAmount != rewardAmount {
		t.Errorf("reward list totalAmount = %d, want %d", found.TotalAmount, rewardAmount)
	} else {
		t.Logf("Recipient found in reward list with totalAmount=%d", found.TotalAmount)
	}

	// Step 5: an address that never received a faucet/reward must return an EMPTY record from both
	// single-record endpoints (the query finds nothing, so the record is zero-valued)
	unusedAddr := randomAddressHex()
	t.Logf("Step 5: Querying both endpoints with an unused address (%s); expecting empty records...", unusedAddr)

	t.Logf("Querying plugin endpoint GET /v1/query/faucets?address=%s ...", unusedAddr)
	emptyFaucet, err := queryFaucetRecord(pluginRPCURL, unusedAddr)
	if err != nil {
		t.Fatalf("Failed to query faucet record for unused address: %v", err)
	}
	t.Logf("Faucet record for unused address: recipient=%q totalAmount=%d count=%d", emptyFaucet.RecipientAddress, emptyFaucet.TotalAmount, emptyFaucet.Count)
	if emptyFaucet.Count != 0 || emptyFaucet.TotalAmount != 0 {
		t.Errorf("faucet record for unused address = %+v, want empty (count=0, totalAmount=0)", emptyFaucet)
	} else {
		t.Log("Faucet endpoint correctly returned an empty record for the unused address")
	}

	t.Logf("Querying plugin endpoint GET /v1/query/rewards?address=%s ...", unusedAddr)
	emptyReward, err := queryRewardRecord(pluginRPCURL, unusedAddr)
	if err != nil {
		t.Fatalf("Failed to query reward record for unused address: %v", err)
	}
	t.Logf("Reward record for unused address: recipient=%q totalAmount=%d count=%d", emptyReward.RecipientAddress, emptyReward.TotalAmount, emptyReward.Count)
	if emptyReward.Count != 0 || emptyReward.TotalAmount != 0 {
		t.Errorf("reward record for unused address = %+v, want empty (count=0, totalAmount=0)", emptyReward)
	} else {
		t.Log("Reward endpoint correctly returned an empty record for the unused address")
	}

	t.Log("")
	t.Log("--- Custom RPC endpoints verified successfully! ---")
	t.Logf("  curl '%s/v1/query/faucets?address=%s'", pluginRPCURL, recipientAddr)
	t.Logf("  curl '%s/v1/query/rewards?address=%s'", pluginRPCURL, recipientAddr)
	t.Logf("  curl '%s/v1/query/faucets'", pluginRPCURL)
	t.Logf("  curl '%s/v1/query/rewards'", pluginRPCURL)
}

// randomSuffix generates a random hex suffix for unique nicknames
func randomSuffix() string {
	b := make([]byte, 4)
	cryptorand.Read(b)
	return hex.EncodeToString(b)
}

// randomAddressHex returns a random 20-byte address as hex; used to query an address that has no
// faucet/reward record so the endpoint is expected to return an empty (zero-valued) record
func randomAddressHex() string {
	b := make([]byte, 20)
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

// getRawJSON performs an HTTP GET and returns the raw response body (used by the plugin RPC endpoints)
func getRawJSON(url string) ([]byte, error) {
	resp, err := http.Get(url)
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

// faucetRecord mirrors the JSON shape returned by the plugin's /v1/query/faucets endpoint
type faucetRecord struct {
	RecipientAddress string `json:"recipientAddress"`
	TotalAmount      uint64 `json:"totalAmount"`
	Count            uint64 `json:"count"`
}

// rewardRecord mirrors the JSON shape returned by the plugin's /v1/query/rewards endpoint
type rewardRecord struct {
	RecipientAddress string `json:"recipientAddress"`
	LastAdminAddress string `json:"lastAdminAddress"`
	TotalAmount      uint64 `json:"totalAmount"`
	Count            uint64 `json:"count"`
}

// queryFaucetRecord fetches a single recipient's faucet record from the plugin RPC server
func queryFaucetRecord(pluginURL, address string) (*faucetRecord, error) {
	respBody, err := getRawJSON(pluginURL + "/v1/query/faucets?address=" + address)
	if err != nil {
		return nil, err
	}
	var result struct {
		Faucet faucetRecord `json:"faucet"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}
	return &result.Faucet, nil
}

// queryFaucetList fetches all faucet records from the plugin RPC server (range read)
func queryFaucetList(pluginURL string) ([]faucetRecord, error) {
	respBody, err := getRawJSON(pluginURL + "/v1/query/faucets")
	if err != nil {
		return nil, err
	}
	var result struct {
		Faucets []faucetRecord `json:"faucets"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}
	return result.Faucets, nil
}

// queryRewardRecord fetches a single recipient's reward record from the plugin RPC server
func queryRewardRecord(pluginURL, address string) (*rewardRecord, error) {
	respBody, err := getRawJSON(pluginURL + "/v1/query/rewards?address=" + address)
	if err != nil {
		return nil, err
	}
	var result struct {
		Reward rewardRecord `json:"reward"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}
	return &result.Reward, nil
}

// queryRewardList fetches all reward records from the plugin RPC server (range read)
func queryRewardList(pluginURL string) ([]rewardRecord, error) {
	respBody, err := getRawJSON(pluginURL + "/v1/query/rewards")
	if err != nil {
		return nil, err
	}
	var result struct {
		Rewards []rewardRecord `json:"rewards"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, string(respBody))
	}
	return result.Rewards, nil
}

// pollFaucetRecord polls the faucet endpoint until a record with count > 0 appears or the timeout elapses
func pollFaucetRecord(pluginURL, address string, timeout time.Duration) (*faucetRecord, error) {
	return pollFaucetRecordUntil(pluginURL, address, 1, timeout)
}

// pollFaucetRecordUntil polls the faucet endpoint until the record's count reaches minCount or the timeout elapses
func pollFaucetRecordUntil(pluginURL, address string, minCount uint64, timeout time.Duration) (*faucetRecord, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		rec, err := queryFaucetRecord(pluginURL, address)
		if err != nil {
			lastErr = err
		} else if rec.Count >= minCount {
			return rec, nil
		}
		time.Sleep(1 * time.Second)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("faucet record for %s did not reach count %d within timeout", address, minCount)
}

// pollRewardRecord polls the reward endpoint until a record with count > 0 appears or the timeout elapses
func pollRewardRecord(pluginURL, address string, timeout time.Duration) (*rewardRecord, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		rec, err := queryRewardRecord(pluginURL, address)
		if err != nil {
			lastErr = err
		} else if rec.Count > 0 {
			return rec, nil
		}
		time.Sleep(1 * time.Second)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("reward record for %s not found within timeout", address)
}

// findFaucet returns the faucet record matching the address (case-insensitive), or nil
func findFaucet(records []faucetRecord, address string) *faucetRecord {
	for i := range records {
		if strings.EqualFold(records[i].RecipientAddress, address) {
			return &records[i]
		}
	}
	return nil
}

// findReward returns the reward record matching the address (case-insensitive), or nil
func findReward(records []rewardRecord, address string) *rewardRecord {
	for i := range records {
		if strings.EqualFold(records[i].RecipientAddress, address) {
			return &records[i]
		}
	}
	return nil
}

// validateFaucetRecords asserts every record returned by the range/list endpoint is a well-formed
// faucet record. This guards against the endpoint scanning a colliding key prefix and returning
// unrelated core records (e.g. validators), which a lenient decoder would silently accept as a
// faucet with count=0. A genuine faucet record always has count>=1 and totalAmount>=1.
func validateFaucetRecords(records []faucetRecord) error {
	for _, rec := range records {
		if addr, err := hex.DecodeString(rec.RecipientAddress); err != nil || len(addr) != 20 {
			return fmt.Errorf("malformed recipientAddress %q", rec.RecipientAddress)
		}
		if rec.Count < 1 || rec.TotalAmount < 1 {
			return fmt.Errorf("malformed record (recipient=%s, totalAmount=%d, count=%d); endpoint may be reading colliding core state", rec.RecipientAddress, rec.TotalAmount, rec.Count)
		}
	}
	return nil
}

// validateRewardRecords asserts every record returned by the range/list endpoint is a well-formed
// reward record (guards against colliding-prefix scans returning core state like committees).
func validateRewardRecords(records []rewardRecord) error {
	for _, rec := range records {
		if addr, err := hex.DecodeString(rec.RecipientAddress); err != nil || len(addr) != 20 {
			return fmt.Errorf("malformed recipientAddress %q", rec.RecipientAddress)
		}
		if addr, err := hex.DecodeString(rec.LastAdminAddress); err != nil || len(addr) != 20 {
			return fmt.Errorf("malformed lastAdminAddress %q", rec.LastAdminAddress)
		}
		if rec.Count < 1 || rec.TotalAmount < 1 {
			return fmt.Errorf("malformed record (recipient=%s, totalAmount=%d, count=%d); endpoint may be reading colliding core state", rec.RecipientAddress, rec.TotalAmount, rec.Count)
		}
	}
	return nil
}
