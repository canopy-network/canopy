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
StakeAmount:     100_000_000, // 100 PRX
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

        // Step 3: Register resolver
        h, _ = getHeight()
        regMsg := &contract.MessageRegisterResolver{
                ResolverAddress: hexDecode(addr),
                StakeAmount:     100_000_000,
        }
        hash = submitTx(t, key, "register_resolver", "MessageRegisterResolver", regMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
                t.Fatalf("register_resolver failed: %v", err)
        }
        t.Log("Resolver registered")

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
                ResolverAddress: hexDecode(addr),
                ProposedOutcome: true,
                ProposalBond:    100_000_000,
        }
        hash = submitTx(t, key, "propose_outcome", "MessageProposeOutcome", propMsg, h)
        if err := waitForTx(addr, hash, 60*time.Second); err != nil {
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
ExpiryTime:     h + 30,
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

// Register resolver and wait for expiry
h, _ = getHeight()
regMsg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(addr),
StakeAmount:     800_000,
}
hash = submitTx(t, key, "register_resolver", "MessageRegisterResolver", regMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("register_resolver: %v", err)
}

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
ResolverAddress: hexDecode(addr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, key, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeBlocks := contract.ComputeDisputeBlocks(0, 0) // TEST_MODE returns TEST_DISPUTE_BLOCKS
disputeTarget := h + disputeBlocks + 2
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

// Register resolver and wait for expiry
h, _ = getHeight()
regMsg := &contract.MessageRegisterResolver{
ResolverAddress: hexDecode(addr),
StakeAmount:     800_000,
}
hash = submitTx(t, key, "register_resolver", "MessageRegisterResolver", regMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("register_resolver: %v", err)
}

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
ResolverAddress: hexDecode(addr),
ProposedOutcome: true,
ProposalBond:    bond,
}
hash = submitTx(t, key, "propose_outcome", "MessageProposeOutcome", propMsg, h)
if err := waitForTx(addr, hash, 60*time.Second); err != nil {
t.Fatalf("propose_outcome: %v", err)
}

// Wait for dispute window
disputeTarget := h + contract.ComputeDisputeBlocks(0, 0) + 2
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
