// Praxis transaction signer — CLI tool for signing and submitting transactions
// Build:  cd ~/praxis/plugin/go && GOTOOLCHAIN=local /usr/local/go/bin/go build -o ~/go/bin/praxis-sign ./cmd/sign
// Usage:  praxis-sign -key <64-char-hex> -type create_market [-dry]

package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/canopy-network/go-plugin/contract"
	"github.com/canopy-network/go-plugin/crypto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	flagKey    = flag.String("key", "", "BLS12-381 private key hex (64 chars)")
	flagRPC    = flag.String("rpc", "http://localhost:50002", "RPC endpoint")
	flagType   = flag.String("type", "", "tx type: create_market | submit_prediction | resolve_market | claim_winnings")
	flagFee    = flag.Uint64("fee", 10000, "transaction fee in uPRX")
	flagNet    = flag.Uint64("net", 1, "network ID")
	flagChain  = flag.Uint64("chain", 1, "chain ID")
	flagHeight = flag.Uint64("height", 0, "created_height (0 = query from node)")
	flagDry    = flag.Bool("dry", false, "print signed tx JSON, do not submit")
)

var scanner = bufio.NewScanner(os.Stdin)

func main() {
	flag.Parse()

	if *flagKey == "" || *flagType == "" {
		fmt.Fprintln(os.Stderr, "Usage: praxis-sign -key <hex> -type <txtype> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Load private key
	privKey, err := crypto.StringToBLS12381PrivateKey(*flagKey)
	must("load private key", err)
	pubKeyBytes := privKey.PublicKey().Bytes()
	fmt.Fprintf(os.Stderr, "✓ Key loaded  pubkey: %s\n", hex.EncodeToString(pubKeyBytes))

	// Get created height
	createdHeight := *flagHeight
	if createdHeight == 0 {
		createdHeight = queryHeight(*flagRPC)
		fmt.Fprintf(os.Stderr, "✓ Chain height: %d\n", createdHeight)
	}

	// Build the inner proto message
	msgProto, typeURL := buildMessage(*flagType)

	// Marshal inner message bytes and wrap in Any
	innerBytes, merr := proto.MarshalOptions{Deterministic: true}.Marshal(msgProto)
	must("marshal inner message", merr)
	msgAny := &anypb.Any{
		TypeUrl: typeURL,
		Value:   innerBytes,
	}

	txTime := uint64(time.Now().UnixNano())

	// ── Step 1: Get sign bytes (proto.Marshal of tx with nil signature) ──────
	signBytes, serr := crypto.GetSignBytes(
		*flagType, msgAny,
		txTime, createdHeight,
		*flagFee, "",
		*flagNet, *flagChain,
	)
	must("get sign bytes", serr)

	// ── Step 2: Sign with BLS12-381 ─────────────────────────────────────────
	sig := privKey.Sign(signBytes)
	fmt.Fprintf(os.Stderr, "✓ Signed  sig: %s (%d bytes)\n", hex.EncodeToString(sig[:8])+"...", len(sig))

	// ── Step 3: Build the msg JSON using protojson on the inner message ──────
	// AnyFromJSONForMessageType (called by UnmarshalJSON on the server) runs
	// protojson.Unmarshal on j.Msg, so j.Msg must be the inner message fields
	// rendered as proto-JSON (e.g. {"creatorAddress":"<base64>", "question":"..."})
	// NOT a nested Any with a "value" blob.
	msgJSON, mjerr := protojson.MarshalOptions{}.Marshal(msgProto)
	must("marshal inner message to proto-JSON", mjerr)

	// ── Step 4: Build the outer transaction JSON ─────────────────────────────
	// Field names must match jsonTx struct tags in lib/tx.go:
	//   "type"          ← messageType (NOT "messageType")
	//   "msg"           ← json.RawMessage of the inner message fields
	//   "signature"     ← { "publicKey": "<hex>", "signature": "<hex>" }
	//   "createdHeight" ← uint64
	//   "time"          ← uint64
	//   "fee"           ← uint64
	//   "networkID"     ← uint64
	//   "chainID"       ← uint64
	//
	// Signature bytes must be HEX (not base64) — see jsonSignature in lib/tx.go.
	// The "msg" field is passed to AnyFromJSONForMessageType which calls
	// protojson.Unmarshal, so it must be proto-JSON (not a raw Any value blob).
	finalTx := map[string]interface{}{
		"type": *flagType,
		"msg":  json.RawMessage(msgJSON),
		"signature": map[string]string{
			"publicKey": hex.EncodeToString(pubKeyBytes),
			"signature": hex.EncodeToString(sig),
		},
		"createdHeight": createdHeight,
		"time":          txTime,
		"fee":           *flagFee,
		"memo":          "",
		"networkID":     *flagNet,
		"chainID":       *flagChain,
	}

	txJSON, jerr := json.MarshalIndent(finalTx, "", "  ")
	must("marshal final tx JSON", jerr)

	if *flagDry {
		fmt.Println(string(txJSON))
		return
	}

	// ── Step 5: POST to /v1/tx ───────────────────────────────────────────────
	resp, rerr := http.Post(*flagRPC+"/v1/tx", "application/json", bytes.NewReader(txJSON))
	must("post tx", rerr)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("RPC response [%s]:\n%s\n", resp.Status, string(body))
}

// ── Prompt helpers ────────────────────────────────────────────────────────────

func prompt(label string) string {
	fmt.Fprintf(os.Stderr, "  %s: ", label)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func promptUint(label string) uint64 {
	s := prompt(label)
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid number %q: %v\n", s, err)
		os.Exit(1)
	}
	return v
}

func hexToBytes(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid hex %q: %v\n", h, err)
		os.Exit(1)
	}
	return b
}

// ── Message builders ──────────────────────────────────────────────────────────

func buildMessage(txType string) (proto.Message, string) {
	switch txType {
	case "create_market":
		return buildCreateMarket(), "type.googleapis.com/types.MessageCreateMarket"
	case "submit_prediction":
		return buildSubmitPrediction(), "type.googleapis.com/types.MessageSubmitPrediction"
	case "resolve_market":
		return buildResolveMarket(), "type.googleapis.com/types.MessageResolveMarket"
	case "claim_winnings":
		return buildClaimWinnings(), "type.googleapis.com/types.MessageClaimWinnings"
	default:
		fmt.Fprintf(os.Stderr, "unknown tx type: %s\n", txType)
		os.Exit(1)
		return nil, ""
	}
}

func buildCreateMarket() *contract.MessageCreateMarket {
	fmt.Fprintln(os.Stderr, "\n── create_market ─────────────────────")
	return &contract.MessageCreateMarket{
		CreatorAddress:   hexToBytes(prompt("creatorAddress (hex)")),
		Question:         prompt("question"),
		Description:      prompt("description"),
		ResolverAddress:  hexToBytes(prompt("resolverAddress (hex)")),
		ResolutionHeight: promptUint("resolutionHeight"),
		StakeAmount:      promptUint("stakeAmount"),
	}
}

func buildSubmitPrediction() *contract.MessageSubmitPrediction {
	fmt.Fprintln(os.Stderr, "\n── submit_prediction ─────────────────")
	return &contract.MessageSubmitPrediction{
		ForecasterAddress: hexToBytes(prompt("forecasterAddress (hex)")),
		MarketId:          promptUint("marketId"),
		Outcome:           uint32(promptUint("outcome (1=YES, 0=NO)")),
		Amount:            promptUint("amount"),
	}
}

func buildResolveMarket() *contract.MessageResolveMarket {
	fmt.Fprintln(os.Stderr, "\n── resolve_market ────────────────────")
	return &contract.MessageResolveMarket{
		ResolverAddress: hexToBytes(prompt("resolverAddress (hex)")),
		MarketId:        promptUint("marketId"),
		WinningOutcome:  uint32(promptUint("winningOutcome (1=YES, 0=NO)")),
	}
}

func buildClaimWinnings() *contract.MessageClaimWinnings {
	fmt.Fprintln(os.Stderr, "\n── claim_winnings ────────────────────")
	return &contract.MessageClaimWinnings{
		ClaimerAddress: hexToBytes(prompt("claimerAddress (hex)")),
		MarketId:       promptUint("marketId"),
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func queryHeight(rpc string) uint64 {
	resp, err := http.Post(rpc+"/v1/query/height", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not query height: %v — using 0\n", err)
		return 0
	}
	defer resp.Body.Close()
	var result struct {
		Height uint64 `json:"height"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Height
}

func must(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", msg, err)
		os.Exit(1)
	}
}
