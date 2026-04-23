// sign_tx.go — Praxis transaction signer
// Place at: ~/praxis/plugin/go/cmd/sign/main.go
// Build:  cd ~/praxis/plugin/go && GOTOOLCHAIN=local go build -o ~/go/bin/praxis-sign ./cmd/sign
// Usage:  praxis-sign -key <64-char-hex> -type create_market -rpc http://localhost:50002
//         (then fill prompts, or pipe JSON via stdin with -json flag)

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/canopy-network/go-plugin/tutorial/contract"
	"github.com/canopy-network/go-plugin/tutorial/crypto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// ── CLI flags ─────────────────────────────────────────────────────────────────

var (
	flagKey    = flag.String("key", "", "BLS12-381 private key hex (64 chars)")
	flagRPC    = flag.String("rpc", "http://localhost:50002", "RPC endpoint")
	flagType   = flag.String("type", "", "tx type: create_market | submit_prediction | resolve_market | claim_winnings")
	flagFee    = flag.Uint64("fee", 10000, "transaction fee in uPRX")
	flagNet    = flag.Uint64("net", 1, "network ID")
	flagChain  = flag.Uint64("chain", 1, "chain ID")
	flagHeight = flag.Uint64("height", 0, "created_height (0 = query from node)")
	flagDry    = flag.Bool("dry", false, "print signed tx JSON, do not submit")
	flagJSON   = flag.Bool("json", false, "read raw msgFields JSON from stdin instead of prompts")
)

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

	// Marshal the inner message to proto bytes (for anypb.Any.Value)
	msgBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msgProto)
	must("marshal inner message", err)

	msgAny := &anypb.Any{
		TypeUrl: typeURL,
		Value:   msgBytes,
	}

	txTime := uint64(time.Now().UnixNano())

	// Get sign bytes (Transaction proto with nil signature, deterministic)
	signBytes, err := crypto.GetSignBytes(
		*flagType, msgAny,
		txTime, createdHeight,
		*flagFee, "",
		*flagNet, *flagChain,
	)
	must("get sign bytes", err)

	// Sign with BDN/kyber
	sig := privKey.Sign(signBytes)
	fmt.Fprintf(os.Stderr, "✓ Signed  sig: %s (%d bytes)\n", hex.EncodeToString(sig[:8])+"...", len(sig))

	// Build final transaction for RPC
	// The RPC expects JSON with base64-encoded address fields and signature
	finalTx := buildFinalTxJSON(*flagType, typeURL, msgProto, txTime, createdHeight, *flagFee, *flagNet, *flagChain, pubKeyBytes, sig)

	txJSON, err := json.MarshalIndent(finalTx, "", "  ")
	must("marshal final tx", err)

	if *flagDry {
		fmt.Println(string(txJSON))
		return
	}

	// Submit to RPC
	resp, err := http.Post(*flagRPC+"/v1/tx", "application/json", bytes.NewReader(txJSON))
	must("post tx", err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("RPC response [%s]:\n%s\n", resp.Status, string(body))
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

func prompt(label string) string {
	fmt.Fprintf(os.Stderr, "  %s: ", label)
	var v string
	fmt.Scanln(&v)
	return v
}

func promptUint(label string) uint64 {
	fmt.Fprintf(os.Stderr, "  %s: ", label)
	var v uint64
	fmt.Fscanf(os.Stdin, "%d", &v)
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

// ── Final TX JSON builder ─────────────────────────────────────────────────────
// The Canopy RPC /v1/tx expects the same JSON shape as the frontend sends,
// with address bytes as base64 and signature fields as base64.

func buildFinalTxJSON(txType, typeURL string, msg proto.Message, txTime, createdHeight, fee, netID, chainID uint64, pubKey, sig []byte) map[string]interface{} {
	// Build msgFields as a JSON-friendly map with base64 addresses
	msgFields := protoToJSONFields(msg)

	return map[string]interface{}{
		"type":          txType,
		"msgTypeUrl":    typeURL,
		"msgFields":     msgFields,
		"time":          txTime,
		"createdHeight": createdHeight,
		"fee":           fee,
		"networkID":     netID,
		"chainID":       chainID,
		"memo":          "",
		"signature": map[string]string{
			"publicKey": base64.StdEncoding.EncodeToString(pubKey),
			"signature": base64.StdEncoding.EncodeToString(sig),
		},
	}
}

func protoToJSONFields(msg proto.Message) map[string]interface{} {
	switch m := msg.(type) {
	case *contract.MessageCreateMarket:
		return map[string]interface{}{
			"creatorAddress":   base64.StdEncoding.EncodeToString(m.CreatorAddress),
			"question":         m.Question,
			"description":      m.Description,
			"resolverAddress":  base64.StdEncoding.EncodeToString(m.ResolverAddress),
			"resolutionHeight": m.ResolutionHeight,
			"stakeAmount":      m.StakeAmount,
		}
	case *contract.MessageSubmitPrediction:
		return map[string]interface{}{
			"forecasterAddress": base64.StdEncoding.EncodeToString(m.ForecasterAddress),
			"marketId":          m.MarketId,
			"outcome":           m.Outcome,
			"amount":            m.Amount,
		}
	case *contract.MessageResolveMarket:
		return map[string]interface{}{
			"resolverAddress": base64.StdEncoding.EncodeToString(m.ResolverAddress),
			"marketId":        m.MarketId,
			"winningOutcome":  m.WinningOutcome,
		}
	case *contract.MessageClaimWinnings:
		return map[string]interface{}{
			"claimerAddress": base64.StdEncoding.EncodeToString(m.ClaimerAddress),
			"marketId":       m.MarketId,
		}
	}
	return nil
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
