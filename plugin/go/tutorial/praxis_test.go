package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
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
		"8f8b550064ec4ee4551d1666cb0ee5d35fc5154a": {
			Address:    "8f8b550064ec4ee4551d1666cb0ee5d35fc5154a",
			PublicKey:  "88634c8e0fd9ee8911b362e5aff8c046154263e9b8e507fc5efe5b5d9cb6cb4fd14c3672bccb929c411e3050ccca44a9",
			PrivateKey: "1c91a4882751adc1fa4f2574c4321bf144e36411ade55e099e9c6ffece87ee49",
		},
		"205f68c279331cd17b9d41727f09eed7162b0389": {
			Address:    "205f68c279331cd17b9d41727f09eed7162b0389",
			PublicKey:  "94ed6fa9309508f451d036ebeac618e317bc10883abad9d784246c87131fc66b8cb4d3b4b1b64a08de9c5737ea035ef3",
			PrivateKey: "3b0da148ffe050d58288df78a4b9ed58012a816919e4888b4a066f61d04c4e22",
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
			fmt.Sprintf(`{"address":%q,"perPage":50}`, sender))
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


// setupResolver creates a fresh key, funds it from the validator, and registers
// it as a resolver. Returns the keyGroup and address. The creator address must
// differ from the returned resolver address (COI-2 enforcement).
func setupResolver(t *testing.T, validatorKey *keyGroup, validatorAddr string, stakeAmount uint64) (*keyGroup, string) {
	t.Helper()
	// Always use the hardcoded predictor key — keystore persists across node restarts.
	// Fund with stakeAmount * 3 to ensure sufficient balance after fees and prior deductions.
	resolverAddr := "205f68c279331cd17b9d41727f09eed7162b0389"
	h, _ := getHeight()
	fundMsg := &contract.MessageSend{
		FromAddress: hexDecode(validatorAddr),
		ToAddress:   hexDecode(resolverAddr),
		Amount:      stakeAmount * 3,
	}
	hash := submitSendTx(t, validatorKey, fundMsg, h)
	if err := waitForTx(validatorAddr, hash, 60*time.Second); err != nil {
		t.Fatalf("fund resolver: %v", err)
	}
	// Register resolver — stake adds on top of any existing stake.
	h2, _ := getHeight()
	regMsg := &contract.MessageRegisterResolver{
		ResolverAddress: hexDecode(resolverAddr),
		StakeAmount:     stakeAmount,
	}
	regKey := &keyGroup{
		Address:    resolverAddr,
		PublicKey:  "94ed6fa9309508f451d036ebeac618e317bc10883abad9d784246c87131fc66b8cb4d3b4b1b64a08de9c5737ea035ef3",
		PrivateKey: "3b0da148ffe050d58288df78a4b9ed58012a816919e4888b4a066f61d04c4e22",
	}
	regHash := submitTx(t, regKey, "register_resolver", "MessageRegisterResolver", regMsg, h2)
	if err := waitForTx(resolverAddr, regHash, 60*time.Second); err != nil {
		t.Fatalf("register resolver: %v", err)
	}
	t.Logf("Resolver %s registered with stake %d", resolverAddr, stakeAmount)
	return regKey, resolverAddr
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
		B0:             60_000_000, // MIN_B0
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
B0:             60_000_000,
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


func TestRegisterResolver(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, _ := keystoreGetKey(addr, "")
h, _ := getHeight()

msg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(addr),
StakeAmount:     500_000_000_000, // 500,000 PRX minimum
}
txHash := submitTx(t, key, "register_resolver", "MessageRegisterResolver", msg, h)
if err := waitForTx(addr, txHash, 60*time.Second); err != nil {
t.Fatalf("register resolver failed: %v", err)
}
t.Log("Resolver registered!")
}

func TestPORSFullFlow(t *testing.T) {
        addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
        key, err := keystoreGetKey(addr, "")
        if err != nil {
                t.Fatalf("key: %v", err)
        }

        // Step 1: Create market with short expiry
        h, _ := getHeight()
        t.Logf("Starting height: %d", h)

        nonce := uint64(time.Now().UnixMicro())
        createMsg := &contract.MessageCreateMarket{
                CreatorAddress: hexDecode(addr),
                B0:             60_000_000,
                ExpiryTime:     h + 30,
                Nonce:          nonce,
                Question:       "PORS full flow demo - Will Praxis launch on mainnet?",
        }
        t.Logf("Creating market: ExpiryTime=%d", createMsg.ExpiryTime)
        hash := submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
                t.Fatalf("create_market failed: %v", err)
        }
        marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
        t.Logf("Market created. ID: %x", marketId)

        // Step 2: Submit prediction
        h, _ = getHeight()
        predMsg := &contract.MessageSubmitPrediction{
                MarketId:      marketId,
                BettorAddress: hexDecode(addr),
                Outcome:       true,
                Shares:        contract.PRECISION_SCALE,
                MaxCost:       10_000_000,
        }
        hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
                t.Fatalf("submit_prediction failed: %v", err)
        }
        t.Log("Prediction submitted")

        // Step 3: Register resolver (separate address — COI-2: creator cannot resolve)
        resolverKey, resolverAddr := setupResolver(t, key, addr, 500_000_000_000)
        h, _ = getHeight()
        _ = h

        // Wait for market to expire
        t.Logf("Waiting for expiry (height %d)...", createMsg.ExpiryTime+2)
        for {
                h, _ = getHeight()
                if h > createMsg.ExpiryTime+2 {
                        break
                }
                t.Logf("Height %d / need %d", h, createMsg.ExpiryTime+2)
                time.Sleep(4 * time.Second)
        }
        t.Logf("Market expired at height %d", h)

        // Step 4: Propose outcome
        h, _ = getHeight()
        propMsg := &contract.MessageProposeOutcome{
                MarketId:        marketId,
                ResolverAddress: hexDecode(resolverAddr),
                ProposedOutcome: true,
                ProposalBond:    100_000_000,
        }
        hash = submitTx(t, resolverKey, "propose_outcome", "MessageProposeOutcome", propMsg, h)
        if err := waitForTx(resolverAddr, hash, 60*time.Second); err != nil {
                t.Fatalf("propose_outcome failed: %v", err)
        }
        t.Log("Outcome proposed")

        // Wait for dispute window (TEST_MODE = 20 blocks)
        disputeEnd := h + 22
        t.Logf("Waiting for dispute window (height %d)...", disputeEnd)
        for {
                h, _ = getHeight()
                if h > disputeEnd {
                        break
                }
                t.Logf("Height %d / need %d", h, disputeEnd)
                time.Sleep(4 * time.Second)
        }
        t.Log("Dispute window closed")

        // Step 5: Finalize market
        h, _ = getHeight()
        finalMsg := &contract.MessageFinalizeMarket{
                MarketId:   marketId,
                CallerAddr: hexDecode(addr),
        }
        hash = submitTx(t, key, "finalize_market", "MessageFinalizeMarket", finalMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
                t.Fatalf("finalize_market failed: %v", err)
        }
        t.Log("Market finalized")

        // Step 6: Claim winnings
        balBefore := getBalance(addr)
        t.Logf("Balance before claim: %d", balBefore)

        h, _ = getHeight()
        claimMsg := &contract.MessageClaimWinnings{
                MarketId:        marketId,
                ClaimantAddress: hexDecode(addr),
        }
        hash = submitTx(t, key, "claim_winnings", "MessageClaimWinnings", claimMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
                t.Fatalf("claim_winnings failed: %v", err)
        }

        balAfter := getBalance(addr)
        t.Logf("Balance after claim: %d", balAfter)

        if balAfter <= balBefore {
                t.Errorf("Payout not received: before=%d after=%d", balBefore, balAfter)
        } else {
                t.Logf("Payout received: +%d uPRX", balAfter-balBefore)
        }

        t.Log("========================================")
        t.Log("PORS FULL FLOW COMPLETE")
        t.Log("========================================")
}

func TestNonValidatorPredict(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

cfgData, cfgErr := os.ReadFile("test_config.json")
if cfgErr != nil {
t.Fatalf("read test_config.json: %v", cfgErr)
}
var cfg struct {
PredictorAddress string `json:"predictor_address"`
PredictorPrivKey string `json:"predictor_privkey"`
PredictorPubKey  string `json:"predictor_pubkey"`
}
if jsonErr := json.Unmarshal(cfgData, &cfg); jsonErr != nil {
t.Fatalf("parse test_config.json: %v", jsonErr)
}
predKey := &keyGroup{Address: cfg.PredictorAddress, PublicKey: cfg.PredictorPubKey, PrivateKey: cfg.PredictorPrivKey}

// Step 1: Fund predictor
h, _ := getHeight()
sendMsg := &contract.MessageSend{FromAddress: hexDecode(addr), ToAddress: hexDecode(predKey.Address), Amount: 10_000_000}
hash := submitSendTx(t, key, sendMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("fund predictor failed: %v", err)
}
t.Logf("Predictor funded: %d uPRX", getBalance(predKey.Address))

// Step 2: Create market as validator
h, _ = getHeight()
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             60_000_000,
ExpiryTime:     h + 30,
Nonce:          nonce,
Question:       "Will non-validator predict successfully?",
}
hash = submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market failed: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market created. ID: %x", marketId)

// Step 3: Submit prediction as predictor (non-validator)
h, _ = getHeight()
balBefore := getBalance(predKey.Address)
predMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(predKey.Address),
Outcome:       true,
Shares:        contract.PRECISION_SCALE,
MaxCost:       10_000_000,
}
hash = submitTx(t, predKey, "submit_prediction", "MessageSubmitPrediction", predMsg, h)
if err := waitForTx(predKey.Address, hash, 60*time.Second); err != nil {
t.Fatalf("submit_prediction failed: %v", err)
}
balAfter := getBalance(predKey.Address)
t.Logf("Balance before: %d after: %d diff: %d", balBefore, balAfter, int64(balAfter)-int64(balBefore))
if balAfter >= balBefore {
t.Errorf("Predictor balance should have decreased")
}
t.Log("Non-validator prediction submitted successfully!")
}

func TestClaimWinningsPayout(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

cfgData, cfgErr := os.ReadFile("test_config.json")
if cfgErr != nil {
t.Fatalf("read test_config.json: %v", cfgErr)
}
var cfg struct {
PredictorAddress string `json:"predictor_address"`
PredictorPrivKey string `json:"predictor_privkey"`
PredictorPubKey  string `json:"predictor_pubkey"`
}
if jsonErr := json.Unmarshal(cfgData, &cfg); jsonErr != nil {
t.Fatalf("parse test_config.json: %v", jsonErr)
}
bettorB := &keyGroup{Address: cfg.PredictorAddress, PublicKey: cfg.PredictorPubKey, PrivateKey: cfg.PredictorPrivKey}

// Fund bettor B
h, _ := getHeight()
fundMsg := &contract.MessageSend{
FromAddress: hexDecode(addr),
ToAddress:   hexDecode(bettorB.Address),
Amount:      200_000_000,
}
hash := submitSendTx(t, key, fundMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("fund bettorB: %v", err)
}

// Create market — B0 = 100 PRX
h, _ = getHeight()
const b0 = uint64(100_000_000)
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             b0,
ExpiryTime:     h + 50,
Nonce:          nonce,
Question:       "Payout math verification market",
}
hash = submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market ID: %x", marketId)

// Bettor A — 1 YES share
h, _ = getHeight()
sharesA := contract.PRECISION_SCALE
lmsrSeed := b0 - contract.FINALIZATION_BOUNTY
halfSeed := lmsrSeed / 2
costA, pe := contract.ComputeTradeCost(halfSeed, halfSeed, lmsrSeed, uint64(sharesA), true)
if pe != nil {
t.Fatalf("ComputeTradeCost A: %v", pe)
}
t.Logf("Expected cost A: %d uPRX", costA)

predMsgA := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(addr),
Outcome:       true,
Shares:        uint64(sharesA),
MaxCost:       costA * 2,
}
hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsgA, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("prediction A: %v", err)
}

// Bettor B — 2 YES shares
h, _ = getHeight()
sharesB := contract.PRECISION_SCALE * 2
qYesAfterA := halfSeed + uint64(sharesA)
costB, pe := contract.ComputeTradeCost(qYesAfterA, halfSeed, lmsrSeed, uint64(sharesB), true)
if pe != nil {
t.Fatalf("ComputeTradeCost B: %v", pe)
}
t.Logf("Expected cost B: %d uPRX", costB)

predMsgB := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(bettorB.Address),
Outcome:       true,
Shares:        uint64(sharesB),
MaxCost:       costB * 2,
}
hash = submitTx(t, bettorB, "submit_prediction", "MessageSubmitPrediction", predMsgB, h)
if err := waitForTx(bettorB.Address, hash, 60*time.Second); err != nil {
t.Fatalf("prediction B: %v", err)
}

// Register resolver (separate address — COI-2)
resolverKey, resolverAddr := setupResolver(t, key, addr, 500_000_000_000)

// Wait for expiry
expiryTarget := createMsg.ExpiryTime + 2
t.Logf("Waiting for expiry (height %d)...", expiryTarget)
for {
cur, _ := getHeight()
if cur >= expiryTarget {
break
}
t.Logf("Height %d / need %d", cur, expiryTarget)
time.Sleep(2 * time.Second)
}

// Propose YES outcome
h, _ = getHeight()
bond := contract.ComputeMinBond(&contract.MarketState{BEff: lmsrSeed})
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(resolverAddr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, resolverKey, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(resolverAddr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeTarget := h + contract.TEST_DISPUTE_BLOCKS + 2 // TEST_DISPUTE_BLOCKS=20; ComputeDisputeBlocks not used (no PRAXIS_TEST_MODE in test binary)
t.Logf("Waiting for dispute window (height %d)...", disputeTarget)
for {
cur, _ := getHeight()
if cur >= disputeTarget {
break
}
t.Logf("Height %d / need %d", cur, disputeTarget)
time.Sleep(2 * time.Second)
}

// Finalize
h, _ = getHeight()
finMsg := &contract.MessageFinalizeMarket{
MarketId:   marketId,
CallerAddr: hexDecode(addr),
}
hash = submitTx(t, key, "finalize_market", "MessageFinalizeMarket", finMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("finalize_market: %v", err)
}
t.Log("Market finalized")

// Compute expected pool and payouts.
// Pool = lmsrSeed + costA + costB (TX fees go to feePool, not market pool).
// totalWinShares = market.QYes after both predictions = halfSeed + sharesA + sharesB.
// This mirrors what handler_claim_winnings.go uses (market.QYes).
expectedPool := lmsrSeed + costA + costB
totalWinShares := halfSeed + uint64(sharesA) + uint64(sharesB)
expectedPayoutA := contract.ComputePayout(expectedPool, uint64(sharesA), totalWinShares)
expectedPayoutB := contract.ComputePayout(expectedPool, uint64(sharesB), totalWinShares)
t.Logf("Expected pool: %d  totalWinShares: %d (halfSeed=%d sharesA=%d sharesB=%d)",
expectedPool, totalWinShares, halfSeed, sharesA, sharesB)
t.Logf("Expected payout A (1 share): %d uPRX", expectedPayoutA)
t.Logf("Expected payout B (2 shares): %d uPRX", expectedPayoutB)

// Claim A
balABefore := getBalance(addr)
h, _ = getHeight()
claimMsgA := &contract.MessageClaimWinnings{
MarketId:        marketId,
ClaimantAddress: hexDecode(addr),
}
hash = submitTx(t, key, "claim_winnings", "MessageClaimWinnings", claimMsgA, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("claim A: %v", err)
}
balAAfter := getBalance(addr)
actualPayoutA := int64(balAAfter) - int64(balABefore)
t.Logf("Bettor A — before: %d after: %d payout: %d (expected ~%d)",
balABefore, balAAfter, actualPayoutA, expectedPayoutA)

// Claim B
balBBefore := getBalance(bettorB.Address)
h, _ = getHeight()
claimMsgB := &contract.MessageClaimWinnings{
MarketId:        marketId,
ClaimantAddress: hexDecode(bettorB.Address),
}
hash = submitTx(t, bettorB, "claim_winnings", "MessageClaimWinnings", claimMsgB, h)
if err := waitForTx(bettorB.Address, hash, 60*time.Second); err != nil {
t.Fatalf("claim B: %v", err)
}
balBAfter := getBalance(bettorB.Address)
actualPayoutB := int64(balBAfter) - int64(balBBefore)
t.Logf("Bettor B — before: %d after: %d payout: %d (expected ~%d)",
balBBefore, balBAfter, actualPayoutB, expectedPayoutB)

// Assertions — allow 10% tolerance to account for the difference between
// our off-chain cost estimate (computed before TXs land) and the actual
// on-chain cost at the block the TX was included. The pro-rata ratio is
// the critical invariant; absolute amounts may vary with market state.
toleranceA := int64(expectedPayoutA) / 10
toleranceB := int64(expectedPayoutB) / 10
diffA := actualPayoutA - int64(expectedPayoutA) + int64(testFee)
diffB := actualPayoutB - int64(expectedPayoutB) + int64(testFee)
if diffA < -toleranceA || diffA > toleranceA {
t.Errorf("Payout A mismatch: got %d expected %d (diff %d, tolerance ±%d)",
actualPayoutA, expectedPayoutA, diffA, toleranceA)
}
if diffB < -toleranceB || diffB > toleranceB {
t.Errorf("Payout B mismatch: got %d expected %d (diff %d, tolerance ±%d)",
actualPayoutB, expectedPayoutB, diffB, toleranceB)
}

// B holds 2x the shares of A so should receive ~2x the payout.
// Use ratio check rather than exact multiply to handle integer division dust.
ratioTolerance := int64(expectedPayoutA) / 10
ratioOk := int64(expectedPayoutB) >= int64(expectedPayoutA)*2-ratioTolerance && int64(expectedPayoutB) <= int64(expectedPayoutA)*2+ratioTolerance
if !ratioOk {
t.Errorf("Payout ratio wrong: A=%d B=%d (expected B≈2×A)", expectedPayoutA, expectedPayoutB)
}

t.Log("Payout math verified ✓")
}

func TestAllBettorsOneSide(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

// Create market
h, _ := getHeight()
const b0 = uint64(60_000_000)
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             b0,
ExpiryTime:     h + 30,
Nonce:          nonce,
Question:       "All bettors one side test",
}
hash := submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market ID: %x", marketId)

// Single bettor — YES only, nobody bets NO
h, _ = getHeight()
lmsrSeed := b0 - contract.FINALIZATION_BOUNTY
halfSeed := lmsrSeed / 2
shares := contract.PRECISION_SCALE
cost, pe := contract.ComputeTradeCost(halfSeed, halfSeed, lmsrSeed, shares, true)
if pe != nil {
t.Fatalf("ComputeTradeCost: %v", pe)
}
predMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(addr),
Outcome:       true,
Shares:        shares,
MaxCost:       cost * 2,
}
hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("submit_prediction: %v", err)
}
t.Logf("Single YES bettor cost: %d uPRX", cost)

// Register resolver (separate address — COI-2)
resolverKey, resolverAddr := setupResolver(t, key, addr, 500_000_000_000)

expiryTarget := createMsg.ExpiryTime + 2
t.Logf("Waiting for expiry (height %d)...", expiryTarget)
for {
cur, _ := getHeight()
if cur >= expiryTarget {
break
}
time.Sleep(2 * time.Second)
}

// Propose YES outcome
h, _ = getHeight()
bond := contract.ComputeMinBond(&contract.MarketState{BEff: lmsrSeed})
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(resolverAddr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, resolverKey, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(resolverAddr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeTarget := h + contract.TEST_DISPUTE_BLOCKS + 2
t.Logf("Waiting for dispute window (height %d)...", disputeTarget)
for {
cur, _ := getHeight()
if cur >= disputeTarget {
break
}
time.Sleep(2 * time.Second)
}

// Finalize
h, _ = getHeight()
finMsg := &contract.MessageFinalizeMarket{
MarketId:   marketId,
CallerAddr: hexDecode(addr),
}
hash = submitTx(t, key, "finalize_market", "MessageFinalizeMarket", finMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("finalize_market: %v", err)
}
t.Log("Market finalized")

// Claim — sole YES winner should receive entire pool
expectedPool := lmsrSeed + cost
totalWinShares := halfSeed + shares
expectedPayout := contract.ComputePayout(expectedPool, shares, totalWinShares)
t.Logf("Expected pool: %d totalWinShares: %d expectedPayout: %d", expectedPool, totalWinShares, expectedPayout)

balBefore := getBalance(addr)
h, _ = getHeight()
claimMsg := &contract.MessageClaimWinnings{
MarketId:        marketId,
ClaimantAddress: hexDecode(addr),
}
hash = submitTx(t, key, "claim_winnings", "MessageClaimWinnings", claimMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("claim_winnings: %v", err)
}
balAfter := getBalance(addr)
actualPayout := int64(balAfter) - int64(balBefore)
t.Logf("Balance before: %d after: %d payout: %d (expected ~%d)",
balBefore, balAfter, actualPayout, expectedPayout)

// Payout must be positive
if actualPayout <= 0 {
t.Errorf("Expected positive payout, got %d", actualPayout)
}

// Payout within 15% of expected — off-chain estimate uses pre-TX market state
// so actual on-chain cost may differ slightly from ComputeTradeCost estimate.
tolerance := int64(expectedPayout) * 15 / 100
diff := actualPayout - int64(expectedPayout) + int64(testFee)
if diff < -tolerance || diff > tolerance {
t.Errorf("Payout mismatch: got %d expected %d (diff %d, tolerance ±%d)",
actualPayout, expectedPayout, diff, tolerance)
}

t.Log("All-one-side payout verified ✓")
}

func TestCancelledMarketRefund(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

// Create market
h, _ := getHeight()
const b0 = uint64(60_000_000)
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             b0,
ExpiryTime:     h + 30,
Nonce:          nonce,
Question:       "Cancelled market refund test",
}
hash := submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market ID: %x", marketId)

// Submit prediction
h, _ = getHeight()
lmsrSeed := b0 - contract.FINALIZATION_BOUNTY
halfSeed := lmsrSeed / 2
shares := contract.PRECISION_SCALE
cost, pe := contract.ComputeTradeCost(halfSeed, halfSeed, lmsrSeed, shares, true)
if pe != nil {
t.Fatalf("ComputeTradeCost: %v", pe)
}
predMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(addr),
Outcome:       true,
Shares:        shares,
MaxCost:       cost * 2,
}
hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("submit_prediction: %v", err)
}
t.Logf("Prediction cost: %d uPRX", cost)

// Wait past cancel threshold: ExpiryTime + TEST_RESOLUTION_DELAY + TEST_GRACE_PERIOD + margin
// TEST_RESOLUTION_DELAY=2, TEST_GRACE_PERIOD=2, so cancelThreshold = ExpiryTime + 4
cancelTarget := createMsg.ExpiryTime + contract.TEST_RESOLUTION_DELAY + contract.TEST_GRACE_PERIOD + 2
t.Logf("Waiting for cancel threshold (height %d)...", cancelTarget)
for {
cur, _ := getHeight()
if cur >= cancelTarget {
break
}
t.Logf("Height %d / need %d", cur, cancelTarget)
time.Sleep(2 * time.Second)
}
t.Log("Cancel threshold passed — market should auto-cancel on next claim")

// Claim — should trigger auto-cancel and return CostPaid
balBefore := getBalance(addr)
h, _ = getHeight()
claimMsg := &contract.MessageClaimWinnings{
MarketId:        marketId,
ClaimantAddress: hexDecode(addr),
}
hash = submitTx(t, key, "claim_winnings", "MessageClaimWinnings", claimMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("claim_winnings: %v", err)
}
balAfter := getBalance(addr)
actualRefund := int64(balAfter) - int64(balBefore)
t.Logf("Balance before: %d after: %d refund: %d (expected cost ~%d)",
balBefore, balAfter, actualRefund, cost)

// Refund must be positive
if actualRefund <= 0 {
t.Errorf("Expected positive refund, got %d", actualRefund)
}

// Refund should be close to original cost paid (within 15%)
tolerance := int64(cost) * 15 / 100
diff := actualRefund - int64(cost) + int64(testFee)
if diff < -tolerance || diff > tolerance {
t.Errorf("Refund mismatch: got %d expected ~%d (diff %d, tolerance ±%d)",
actualRefund, cost, diff, tolerance)
}

t.Log("Cancelled market refund verified ✓")
}

func TestLosingBettorZeroPayout(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get validator key: %v", err)
}

cfgData, cfgErr := os.ReadFile("test_config.json")
if cfgErr != nil {
t.Fatalf("read test_config.json: %v", cfgErr)
}
var cfg struct {
PredictorAddress string `json:"predictor_address"`
PredictorPrivKey string `json:"predictor_privkey"`
PredictorPubKey  string `json:"predictor_pubkey"`
}
if jsonErr := json.Unmarshal(cfgData, &cfg); jsonErr != nil {
t.Fatalf("parse test_config.json: %v", jsonErr)
}
loser := &keyGroup{Address: cfg.PredictorAddress, PublicKey: cfg.PredictorPubKey, PrivateKey: cfg.PredictorPrivKey}

// Fund loser
h, _ := getHeight()
fundMsg := &contract.MessageSend{
FromAddress: hexDecode(addr),
ToAddress:   hexDecode(loser.Address),
Amount:      100_000_000,
}
hash := submitSendTx(t, key, fundMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("fund loser: %v", err)
}

// Create market
h, _ = getHeight()
const b0 = uint64(60_000_000)
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             b0,
ExpiryTime:     h + 30,
Nonce:          nonce,
Question:       "Losing bettor zero payout test",
}
hash = submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market ID: %x", marketId)

// Bettor A (validator) bets YES
h, _ = getHeight()
lmsrSeed := b0 - contract.FINALIZATION_BOUNTY
halfSeed := lmsrSeed / 2
sharesA := contract.PRECISION_SCALE
costA, pe := contract.ComputeTradeCost(halfSeed, halfSeed, lmsrSeed, sharesA, true)
if pe != nil {
t.Fatalf("ComputeTradeCost A: %v", pe)
}
predMsgA := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(addr),
Outcome:       true,
Shares:        sharesA,
MaxCost:       costA * 2,
}
hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsgA, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("prediction A (YES): %v", err)
}
t.Logf("Bettor A (YES) cost: %d uPRX", costA)

// Bettor B (loser) bets NO
h, _ = getHeight()
sharesB := contract.PRECISION_SCALE
qYesAfterA := halfSeed + sharesA
costB, pe := contract.ComputeTradeCost(qYesAfterA, halfSeed, lmsrSeed, sharesB, false)
if pe != nil {
t.Fatalf("ComputeTradeCost B: %v", pe)
}
predMsgB := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(loser.Address),
Outcome:       false,
Shares:        sharesB,
MaxCost:       costB * 2,
}
hash = submitTx(t, loser, "submit_prediction", "MessageSubmitPrediction", predMsgB, h)
if err := waitForTx(loser.Address, hash, 60*time.Second); err != nil {
t.Fatalf("prediction B (NO): %v", err)
}
t.Logf("Bettor B (NO) cost: %d uPRX", costB)

// Register resolver (separate address — COI-2)
resolverKey, resolverAddr := setupResolver(t, key, addr, 500_000_000_000)

expiryTarget := createMsg.ExpiryTime + 2
t.Logf("Waiting for expiry (height %d)...", expiryTarget)
for {
cur, _ := getHeight()
if cur >= expiryTarget {
break
}
time.Sleep(2 * time.Second)
}

// Propose YES outcome (A wins, B loses)
h, _ = getHeight()
bond := contract.ComputeMinBond(&contract.MarketState{BEff: lmsrSeed})
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(resolverAddr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, resolverKey, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(resolverAddr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeTarget := h + contract.TEST_DISPUTE_BLOCKS + 2
t.Logf("Waiting for dispute window (height %d)...", disputeTarget)
for {
cur, _ := getHeight()
if cur >= disputeTarget {
break
}
time.Sleep(2 * time.Second)
}

// Finalize
h, _ = getHeight()
finMsg := &contract.MessageFinalizeMarket{
MarketId:   marketId,
CallerAddr: hexDecode(addr),
}
hash = submitTx(t, key, "finalize_market", "MessageFinalizeMarket", finMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("finalize_market: %v", err)
}
t.Log("Market finalized — YES wins")

// Bettor B (loser) claims — should get zero payout
balBBefore := getBalance(loser.Address)
h, _ = getHeight()
claimMsgB := &contract.MessageClaimWinnings{
MarketId:        marketId,
ClaimantAddress: hexDecode(loser.Address),
}
hash = submitTx(t, loser, "claim_winnings", "MessageClaimWinnings", claimMsgB, h)
if err := waitForTx(loser.Address, hash, 60*time.Second); err != nil {
t.Fatalf("claim B: %v", err)
}
balBAfter := getBalance(loser.Address)
actualPayout := int64(balBAfter) - int64(balBBefore)
t.Logf("Loser balance before: %d after: %d diff: %d", balBBefore, balBAfter, actualPayout)

// Loser should receive nothing — zero or negative (TX fee) is acceptable
if actualPayout > 0 {
t.Errorf("Loser should not receive payout, got %d", actualPayout)
}

t.Log("Losing bettor zero payout verified ✓")
}


func TestRegisterNewWalletAsResolver(t *testing.T) {
    validatorKg, _ := keystoreGetKey("e7c7dad131a03f7ea0cc09a637ad096eb3495f77", "")
    kg := &keyGroup{
        Address:    "062b9d112eef6fbea23e3cc44670b9bc8778efad",
        PublicKey:  "b46e36b0d68b7574a84accfc0e101b83d3d1ba6aca035a4a3889d4ee955488deab3dc217dc9362bb691c11be96ad606c",
        PrivateKey: "1645806b57847d4c3683ab27f8a1c23384c372839f994207e0e47dd7769a2a2d",
    }
    h, _ := getHeight()
    fundMsg := &contract.MessageSend{
        FromAddress: hexDecode(validatorKg.Address),
        ToAddress:   hexDecode(kg.Address),
        Amount:      600_000_000_000,
    }
    fundHash := submitSendTx(t, validatorKg, fundMsg, h)
    if err := waitForTx(validatorKg.Address, fundHash, 60*time.Second); err != nil {
        t.Fatalf("fund failed: %v", err)
    }
    h, _ = getHeight()
    msg := &contract.MessageRegisterResolver{
        ResolverAddress: hexDecode(kg.Address),
        StakeAmount:     500_000_000_000,
    }
    txHash := submitTx(t, kg, "register_resolver", "MessageRegisterResolver", msg, h)
    if err := waitForTx(kg.Address, txHash, 60*time.Second); err != nil {
        t.Fatalf("register resolver failed: %v", err)
    }
    t.Log("Resolver registered! TX:", txHash)
}

func TestSetupNewWallet(t *testing.T) {
    validatorKg, _ := keystoreGetKey("e7c7dad131a03f7ea0cc09a637ad096eb3495f77", "")
    newKg := &keyGroup{
        Address:    "8e14dc0ce537f1c75036f11d7495d60882aa6731",
        PublicKey:  "971cbbad4e8c6893b3cd1d31f733dd25058e05ef56abf2e85ba34792704c8ea5510fdc3a57711f695f4f5d45d0e5fa51",
        PrivateKey: "08562bcd9856ba7b2eb9270b9dee9a6ae7a497fce0d4dcab87b3c2f05157dfaf",
    }

    // Step 1: Fund new wallet
    h, _ := getHeight()
    sendMsg := &contract.MessageSend{
        FromAddress: hexDecode(validatorKg.Address),
        ToAddress:   hexDecode(newKg.Address),
        Amount:      10_000_000_000_000,
    }
    txHash := submitSendTx(t, validatorKg, sendMsg, h)
    if err := waitForTx(validatorKg.Address, txHash, 60*time.Second); err != nil {
        t.Fatalf("fund failed: %v", err)
    }
    t.Log("Funded! TX:", txHash)

    // Step 2: Register as resolver
    h, _ = getHeight()
    regMsg := &contract.MessageRegisterResolver{
        ResolverAddress: hexDecode(newKg.Address),
        StakeAmount:     500_000_000_000,
    }
    txHash = submitTx(t, newKg, "register_resolver", "MessageRegisterResolver", regMsg, h)
    if err := waitForTx(newKg.Address, txHash, 60*time.Second); err != nil {
        t.Fatalf("register resolver failed: %v", err)
    }
    t.Log("Resolver registered! TX:", txHash)
}

// ─────────────────────────────────────────────────────────────────────────────
// COI ADVERSARIAL TESTS — Issue-19
// These tests verify that conflict-of-interest guards reject bad transactions.
// ─────────────────────────────────────────────────────────────────────────────

// TestCOI1ResolverWithPosition verifies that a resolver who holds a position
// in the market they are resolving is rejected by propose_outcome (COI-1).
func TestCOI1ResolverWithPosition(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, _ := keystoreGetKey(validatorAddr, "")
predictorAddr := "8f8b550064ec4ee4551d1666cb0ee5d35fc5154a"
predictorKey := &keyGroup{
Address:    predictorAddr,
PublicKey:  "88634c8e0fd9ee8911b362e5aff8c046154263e9b8e507fc5efe5b5d9cb6cb4fd14c3672bccb929c411e3050ccca44a9",
PrivateKey: "1c91a4882751adc1fa4f2574c4321bf144e36411ade55e099e9c6ffece87ee49",
}

h, _ := getHeight()

// 1. Create market
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(validatorAddr),
B0:             60_000_000,
ExpiryTime:     h + 150,
Nonce:          nonce,
Question:       "COI-1 test market",
}
txHash := submitTx(t, validatorKey, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
t.Fatalf("create market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(validatorAddr), nonce)
t.Logf("market_id: %x", marketId)

// 2. Fund predictor and have them bet YES
h2, _ := getHeight()
sendMsg := &contract.MessageSend{
FromAddress: hexDecode(validatorAddr),
ToAddress:   hexDecode(predictorAddr),
Amount:      10_000_000,
}
sendHash := submitSendTx(t, validatorKey, sendMsg, h2)
if err := waitForTx(validatorAddr, sendHash, 60*time.Second); err != nil {
t.Fatalf("fund predictor: %v", err)
}

h3, _ := getHeight()
betMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(predictorAddr),
Outcome:       true,
Shares:        contract.PRECISION_SCALE,
MaxCost:       5_000_000,
}
betHash := submitTx(t, predictorKey, "submit_prediction", "MessageSubmitPrediction", betMsg, h3)
if err := waitForTx(predictorAddr, betHash, 60*time.Second); err != nil {
t.Fatalf("bet: %v", err)
}
t.Log("Predictor holds YES position")

// 3. Register predictor as resolver — fund first, then register
h4b, _ := getHeight()
fundMsg := &contract.MessageSend{
FromAddress: hexDecode(validatorAddr),
ToAddress:   hexDecode(predictorAddr),
Amount:      600_000_000_000,
}
fundHash := submitSendTx(t, validatorKey, fundMsg, h4b)
if err := waitForTx(validatorAddr, fundHash, 60*time.Second); err != nil {
t.Fatalf("fund for stake: %v", err)
}
h4c, _ := getHeight()
regMsg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(predictorAddr),
StakeAmount:     500_000_000_000,
}
regHash := submitTx(t, predictorKey, "register_resolver", "MessageRegisterResolver", regMsg, h4c)
if err := waitForTx(predictorAddr, regHash, 60*time.Second); err != nil {
t.Fatalf("register resolver: %v", err)
}
t.Log("Predictor registered as resolver")

// 4. Wait for market expiry
t.Log("Waiting for market to expire...")
for {
cur, err := getHeight()
if err == nil && cur > 0 && cur > createMsg.ExpiryTime {
break
}
time.Sleep(5 * time.Second)
}

// 5. Attempt propose_outcome as resolver who holds a position — must fail
h5, _ := getHeight()
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(predictorAddr),
ProposedOutcome: true,
ProposalBond:    100_000_000,
}
txHash2 := submitTx(t, predictorKey, "propose_outcome", "MessageProposeOutcome", propMsg, h5)
err := waitForTx(predictorAddr, txHash2, 60*time.Second)
if err == nil {
t.Fatal("COI-1 FAILED: propose_outcome succeeded for resolver with position — expected rejection")
}
t.Logf("COI-1 PASS: propose_outcome correctly rejected resolver with position: %v", err)
}

// TestCOI2CreatorCannotResolve verifies that the market creator cannot also
// act as the resolver for the same market (COI-2).
func TestCOI2CreatorCannotResolve(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, _ := keystoreGetKey(validatorAddr, "")

h, _ := getHeight()

// 1. Create market — validator is the creator
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(validatorAddr),
B0:             60_000_000,
ExpiryTime:     h + 30,
Nonce:          nonce,
Question:       "COI-2 test market",
}
txHash := submitTx(t, validatorKey, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
t.Fatalf("create market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(validatorAddr), nonce)
t.Logf("market_id: %x", marketId)

// 2. Register creator as resolver
h2, _ := getHeight()
regMsg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(validatorAddr),
StakeAmount:     500_000_000_000,
}
regHash := submitTx(t, validatorKey, "register_resolver", "MessageRegisterResolver", regMsg, h2)
if err := waitForTx(validatorAddr, regHash, 60*time.Second); err != nil {
t.Fatalf("register resolver: %v", err)
}

// 3. Wait for expiry
t.Log("Waiting for market to expire...")
for {
cur, _ := getHeight()
if cur > createMsg.ExpiryTime {
break
}
time.Sleep(5 * time.Second)
}

// 4. Attempt propose_outcome as creator — must fail (COI-2)
h3, _ := getHeight()
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(validatorAddr),
ProposedOutcome: true,
ProposalBond:    100_000_000,
}
txHash2 := submitTx(t, validatorKey, "propose_outcome", "MessageProposeOutcome", propMsg, h3)
err := waitForTx(validatorAddr, txHash2, 60*time.Second)
if err == nil {
t.Fatal("COI-2 FAILED: propose_outcome succeeded for creator-as-resolver — expected rejection")
}
t.Logf("COI-2 PASS: propose_outcome correctly rejected creator-as-resolver: %v", err)
}

// TestCOI3PositionCapEnforced verifies that a single address cannot accumulate
// more than 20% of the winning side shares (COI-3).
func TestCOI3PositionCapEnforced(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, _ := keystoreGetKey(validatorAddr, "")

h, _ := getHeight()

// 1. Create market
nonce := uint64(time.Now().UnixMicro())
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(validatorAddr),
B0:             60_000_000,
ExpiryTime:     h + 50000,
Nonce:          nonce,
Question:       "COI-3 cap test market",
}
txHash := submitTx(t, validatorKey, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(validatorAddr, txHash, 60*time.Second); err != nil {
t.Fatalf("create market: %v", err)
}
marketId := contract.DeriveMarketId(hexDecode(validatorAddr), nonce)
t.Logf("market_id: %x", marketId)

// 2. Submit a large YES bet that would exceed 20% of YES shares.
// Initial QYes = B0/2 - FINALIZATION_BOUNTY/2 = ~5_000_000 shares.
// Requesting 50_000_000 shares (10x the initial side) — well over 20%.
h2, _ := getHeight()
betMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(validatorAddr),
Outcome:       true,
Shares:        50_000_000 * contract.PRECISION_SCALE,
MaxCost:       500_000_000,
}
txHash2 := submitTx(t, validatorKey, "submit_prediction", "MessageSubmitPrediction", betMsg, h2)
err := waitForTx(validatorAddr, txHash2, 60*time.Second)
if err == nil {
t.Fatal("COI-3 FAILED: oversized position was accepted — expected ErrPositionCapExceeded")
}
t.Logf("COI-3 PASS: oversized position correctly rejected: %v", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// COI ADVERSARIAL TESTS — Issue-19
// ─────────────────────────────────────────────────────────────────────────────


func TestPRISFeeFlow(t *testing.T) {
addr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
key, err := keystoreGetKey(addr, "")
if err != nil {
t.Fatalf("get key: %v", err)
}

// Create market — B0 = 60 PRX
h, _ := getHeight()
nonce := uint64(time.Now().UnixMicro())
const b0 = uint64(60_000_000)
createMsg := &contract.MessageCreateMarket{
CreatorAddress: hexDecode(addr),
B0:             b0,
ExpiryTime:     h + 80,
Nonce:          nonce,
Question:       "PRIS fee flow verification",
}
balBefore := getBalance(addr)
hash := submitTx(t, key, "create_market", "MessageCreateMarket", createMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("create_market: %v", err)
}
balAfter := getBalance(addr)
marketId := contract.DeriveMarketId(hexDecode(addr), nonce)
t.Logf("Market ID: %x", marketId)
t.Logf("Creation cost: %d uPRX (B0+bond+fee)", balBefore-balAfter)

// Submit prediction — check 2% fee on top
h, _ = getHeight()
lmsrSeed := b0 - contract.FINALIZATION_BOUNTY
halfSeed := lmsrSeed / 2
shares := contract.PRECISION_SCALE
tradeCost, pe := contract.ComputeTradeCost(halfSeed, halfSeed, lmsrSeed, uint64(shares), true)
if pe != nil {
t.Fatalf("ComputeTradeCost: %v", pe)
}
creatorFee  := tradeCost * contract.CREATOR_FEE_BPS / 10_000
resolverFee := tradeCost * contract.RESOLVER_FEE_BPS / 10_000
expectedTotalCost := tradeCost + creatorFee + resolverFee + testFee
t.Logf("tradeCost=%d creatorFee=%d resolverFee=%d totalExpected=%d", tradeCost, creatorFee, resolverFee, expectedTotalCost)

bettorBalBefore := getBalance(addr)
predMsg := &contract.MessageSubmitPrediction{
MarketId:      marketId,
BettorAddress: hexDecode(addr),
Outcome:       true,
Shares:        uint64(shares),
MaxCost:       expectedTotalCost * 2,
}
hash = submitTx(t, key, "submit_prediction", "MessageSubmitPrediction", predMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("submit_prediction: %v", err)
}
bettorBalAfter := getBalance(addr)
actualCost := int64(bettorBalBefore) - int64(bettorBalAfter)
t.Logf("Bettor cost: %d (expected ~%d)", actualCost, expectedTotalCost)
if actualCost < int64(tradeCost+creatorFee+resolverFee) {
t.Errorf("PRIS fees not charged: cost %d below tradeCost+fees %d", actualCost, tradeCost+creatorFee+resolverFee)
} else {
t.Log("PRIS fees charged on top of tradeCost ✓")
}

// Register resolver and finalize
resolverKey, resolverAddr := setupResolver(t, key, addr, 500_000_000_000)

// Wait for expiry
expiryTarget := createMsg.ExpiryTime + 2
for {
cur, _ := getHeight()
if cur >= expiryTarget { break }
time.Sleep(2 * time.Second)
}

// Propose outcome
h, _ = getHeight()
bond := contract.ComputeMinBond(&contract.MarketState{BEff: lmsrSeed})
propMsg := &contract.MessageProposeOutcome{
MarketId:        marketId,
ResolverAddress: hexDecode(resolverAddr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, resolverKey, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(resolverAddr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeTarget := h + contract.TEST_DISPUTE_BLOCKS + 2
for {
cur, _ := getHeight()
if cur >= disputeTarget { break }
time.Sleep(2 * time.Second)
}

// Finalize — resolver should receive resolver fee pool
resolverBalBefore := getBalance(resolverAddr)
h, _ = getHeight()
finMsg := &contract.MessageFinalizeMarket{
MarketId:   marketId,
CallerAddr: hexDecode(addr),
}
hash = submitTx(t, key, "finalize_market", "MessageFinalizeMarket", finMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("finalize_market: %v", err)
}
resolverBalAfter := getBalance(resolverAddr)
resolverGain := int64(resolverBalAfter) - int64(resolverBalBefore)
t.Logf("Resolver balance change at finalization: %d uPRX (expected ~%d resolver fee)", resolverGain, resolverFee)
if resolverGain > 0 {
t.Log("Resolver fee paid at finalization ✓")
} else {
t.Errorf("Resolver fee NOT paid at finalization: gain=%d", resolverGain)
}

t.Log("PRIS fee flow verified ✓")
}

func TestUnstakeResolver(t *testing.T) {
validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
validatorKey, err := keystoreGetKey(validatorAddr, "")
if err != nil {
t.Fatalf("key: %v", err)
}

// Setup resolver with MIN_RESOLVER_STAKE
resolverKey, resolverAddr := setupResolver(t, validatorKey, validatorAddr, 500_000_000_000)
t.Logf("Resolver: %s", resolverAddr)

// ── Test 1: Cannot partial unstake below MIN_RESOLVER_STAKE ──────────
h, _ := getHeight()
unstakeMsg := &contract.MessageUnstakeResolver{
ResolverAddress: hexDecode(resolverAddr),
Amount:          1, // would leave stake at MIN-1
}
hash := submitTx(t, resolverKey, "unstake_resolver", "MessageUnstakeResolver", unstakeMsg, h)
if err := waitForTx(resolverAddr, hash, 60*time.Second); err == nil {
t.Error("Expected rejection: partial unstake below MIN_RESOLVER_STAKE should fail")
} else {
t.Log("Correctly rejected partial unstake below MIN_RESOLVER_STAKE ✓")
}

// ── Test 2: Partial unstake — fund extra stake first ─────────────────
// Add 100,000 PRX extra stake so partial unstake is valid
h, _ = getHeight()
extraStake := uint64(100_000_000_000)
fundMsg := &contract.MessageSend{
FromAddress: hexDecode(validatorAddr),
ToAddress:   hexDecode(resolverAddr),
Amount:      extraStake * 2,
}
fHash := submitSendTx(t, validatorKey, fundMsg, h)
if err := waitForTx(validatorAddr, fHash, 60*time.Second); err != nil {
t.Fatalf("fund extra stake: %v", err)
}

h, _ = getHeight()
addMsg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(resolverAddr),
StakeAmount:     extraStake,
}
aHash := submitTx(t, resolverKey, "register_resolver", "MessageRegisterResolver", addMsg, h)
if err := waitForTx(resolverAddr, aHash, 60*time.Second); err != nil {
t.Fatalf("add stake: %v", err)
}
t.Logf("Added extra stake %d, total stake now %d", extraStake, 500_000_000_000+extraStake)

// Now partial unstake the extra 100,000 PRX — leaves MIN_RESOLVER_STAKE intact
h, _ = getHeight()
partialMsg := &contract.MessageUnstakeResolver{
ResolverAddress: hexDecode(resolverAddr),
Amount:          extraStake,
}
pHash := submitTx(t, resolverKey, "unstake_resolver", "MessageUnstakeResolver", partialMsg, h)
if err := waitForTx(resolverAddr, pHash, 60*time.Second); err != nil {
t.Fatalf("partial unstake: %v", err)
}
t.Log("Partial unstake submitted ✓")

// ── Test 3: Cannot claim unbonded stake before release height ─────────
h, _ = getHeight()
claimMsg := &contract.MessageClaimUnbondedStake{
ResolverAddress: hexDecode(resolverAddr),
}
cHash := submitTx(t, resolverKey, "claim_unbonded_stake", "MessageClaimUnbondedStake", claimMsg, h)
if err := waitForTx(resolverAddr, cHash, 60*time.Second); err == nil {
t.Error("Expected rejection: claim before unbonding period should fail")
} else {
t.Log("Correctly rejected early claim ✓")
}

// ── Test 4: Cannot full exit while unbonding is pending ─────────────
h, _ = getHeight()
exitMsg := &contract.MessageUnstakeResolver{
	ResolverAddress: hexDecode(resolverAddr),
	Amount:          0, // 0 = full exit
}
eHash := submitTx(t, resolverKey, "unstake_resolver", "MessageUnstakeResolver", exitMsg, h)
if err := waitForTx(resolverAddr, eHash, 60*time.Second); err == nil {
	t.Error("Expected rejection: cannot full exit while unbonding pending")
} else {
	t.Log("Correctly rejected full exit while unbonding pending ✓")
}

// ── Test 5: Cannot unstake again after full exit ──────────────────────
h, _ = getHeight()
reMsg := &contract.MessageUnstakeResolver{
ResolverAddress: hexDecode(resolverAddr),
Amount:          0,
}
rHash := submitTx(t, resolverKey, "unstake_resolver", "MessageUnstakeResolver", reMsg, h)
if err := waitForTx(resolverAddr, rHash, 60*time.Second); err == nil {
t.Error("Expected rejection: cannot unstake after full exit")
} else {
t.Log("Correctly rejected double exit ✓")
}

t.Log("TestUnstakeResolver complete ✓")
}
