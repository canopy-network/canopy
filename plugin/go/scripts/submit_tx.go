package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ARBOR-L/ARBOR/plugin/go/contract"
	"github.com/ARBOR-L/ARBOR/plugin/go/crypto"
)

const (
	queryRPCURL = "http://localhost:50002"
	adminRPCURL = "http://localhost:50003"
	networkID   = uint64(1)
	chainID     = uint64(1)
)

// Usage: go run ./scripts create_market <address> <password> '<json-fields>'
func main() {
	if len(os.Args) < 5 {
		fmt.Println("usage: submit_tx <msgType> <address> <password> '<json-fields>'")
		os.Exit(1)
	}
	msgType := os.Args[1]
	addr := os.Args[2]
	password := os.Args[3]
	fieldsJSON := os.Args[4]

	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		fatalf("bad json fields: %v", err)
	}

	key, err := keystoreGetKey(adminRPCURL, addr, password)
	if err != nil {
		fatalf("keystore get failed: %v", err)
	}

	height, err := getHeight(queryRPCURL)
	if err != nil {
		fatalf("get height failed: %v", err)
	}

	typeURL, msgProto, err := buildMessage(msgType, addr, fields)
	if err != nil {
		fatalf("build message failed: %v", err)
	}

	txHash, err := buildSignAndSendTx(queryRPCURL, key, msgType, typeURL, msgProto, 10000, networkID, chainID, height)
	if err != nil {
		fatalf("send tx failed: %v", err)
	}

	fmt.Println("submitted tx hash:", txHash)
	fmt.Printf("check inclusion:\n  curl -s %s/v1/query/txs-by-sender -d '{\"address\":\"%s\",\"perPage\":20}'\n", queryRPCURL, addr)
}

func buildMessage(msgType, signerAddrHex string, fields map[string]interface{}) (string, proto.Message, error) {
	switch msgType {
	case "create_market":
		creator, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode creator address: %w", err)
		}
		assetTier, _ := fields["assetTier"].(float64)
		reserveFactorBps, _ := fields["reserveFactorBps"].(float64)
		var submitters [][]byte
		if rawList, ok := fields["authorizedSubmitters"].([]interface{}); ok {
			for _, item := range rawList {
				s, _ := item.(string)
				addrBytes, dErr := hex.DecodeString(s)
				if dErr != nil {
					return "", nil, fmt.Errorf("decode authorizedSubmitters entry %q: %w", s, dErr)
				}
				submitters = append(submitters, addrBytes)
			}
		}
		msg := &contract.MessageCreateMarket{
			MarketId:             str(fields["marketId"]),
			CollateralAssetId:    str(fields["collateralAssetId"]),
			DebtAssetId:          str(fields["debtAssetId"]),
			AssetTier:            uint32(assetTier),
			ReserveFactorBps:     uint64(reserveFactorBps),
			Creator:              creator,
			AuthorizedSubmitters: submitters,
		}
		return "type.googleapis.com/types.MessageCreateMarket", msg, nil
	case "deposit":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		amount, _ := fields["amount"].(float64)
		msg := &contract.MessageDeposit{
			MarketId: str(fields["marketId"]),
			Address:  addr,
			Amount:   uint64(amount),
		}
		return "type.googleapis.com/types.MessageDeposit", msg, nil
	case "withdraw":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		shares, _ := fields["shares"].(float64)
		msg := &contract.MessageWithdraw{
			MarketId: str(fields["marketId"]),
			Address:  addr,
			Shares:   uint64(shares),
		}
		return "type.googleapis.com/types.MessageWithdraw", msg, nil
	case "set_asset_tier":
		authority, err := hex.DecodeString(str(fields["authority"]))
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		tier, _ := fields["tier"].(float64)
		msg := &contract.MessageSetAssetTier{
			AssetId:   str(fields["assetId"]),
			Tier:      uint32(tier),
			Authority: authority,
		}
		return "type.googleapis.com/types.MessageSetAssetTier", msg, nil
	case "update_price":
		submitter, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode submitter address: %w", err)
		}
		price, _ := fields["price"].(float64)
		confidenceBps, _ := fields["confidenceBps"].(float64)
		msg := &contract.MessageUpdatePrice{
			MarketId:      str(fields["marketId"]),
			AssetId:       str(fields["assetId"]),
			Price:         uint64(price),
			ConfidenceBps: uint32(confidenceBps),
			Submitter:     submitter,
		}
		return "type.googleapis.com/types.MessageUpdatePrice", msg, nil
	case "deposit_collateral":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		quantity, _ := fields["quantity"].(float64)
		msg := &contract.MessageDepositCollateral{
			MarketId: str(fields["marketId"]),
			Address:  addr,
			Quantity: uint64(quantity),
		}
		return "type.googleapis.com/types.MessageDepositCollateral", msg, nil
	case "borrow":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		borrowAmount, _ := fields["borrowAmount"].(float64)
		msg := &contract.MessageBorrow{
			MarketId:     str(fields["marketId"]),
			Address:      addr,
			BorrowAmount: uint64(borrowAmount),
		}
		return "type.googleapis.com/types.MessageBorrow", msg, nil
	case "repay":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		repayAmount, _ := fields["repayAmount"].(float64)
		msg := &contract.MessageRepay{
			MarketId:    str(fields["marketId"]),
			Address:     addr,
			RepayAmount: uint64(repayAmount),
		}
		return "type.googleapis.com/types.MessageRepay", msg, nil
	case "liquidate_position":
		liquidator, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode liquidator address: %w", err)
		}
		borrowerAddr, err := hex.DecodeString(str(fields["borrowerAddress"]))
		if err != nil {
			return "", nil, fmt.Errorf("decode borrowerAddress: %w", err)
		}
		repayAmount, _ := fields["repayAmount"].(float64)
		msg := &contract.MessageLiquidatePosition{
			MarketId:        str(fields["marketId"]),
			Liquidator:      liquidator,
			BorrowerAddress: borrowerAddr,
			RepayAmount:     uint64(repayAmount),
		}
		return "type.googleapis.com/types.MessageLiquidatePosition", msg, nil
	case "pause_market":
		authority, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		msg := &contract.MessagePauseMarket{
			MarketId:  str(fields["marketId"]),
			Authority: authority,
		}
		return "type.googleapis.com/types.MessagePauseMarket", msg, nil
	case "resume_market":
		authority, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		msg := &contract.MessageResumeMarket{
			MarketId:  str(fields["marketId"]),
			Authority: authority,
		}
		return "type.googleapis.com/types.MessageResumeMarket", msg, nil
	case "deprecate_market":
		authority, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		msg := &contract.MessageDeprecateMarket{
			MarketId:  str(fields["marketId"]),
			Authority: authority,
		}
		return "type.googleapis.com/types.MessageDeprecateMarket", msg, nil
	case "update_market_params":
		authority, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		reserveFactorBps, _ := fields["reserveFactorBps"].(float64)
		msg := &contract.MessageUpdateMarketParams{
			MarketId:         str(fields["marketId"]),
			Authority:        authority,
			ReserveFactorBps: uint64(reserveFactorBps),
		}
		return "type.googleapis.com/types.MessageUpdateMarketParams", msg, nil
	case "withdraw_collateral":
		addr, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode address: %w", err)
		}
		quantity, _ := fields["quantity"].(float64)
		msg := &contract.MessageWithdrawCollateral{
			MarketId: str(fields["marketId"]),
			Address:  addr,
			Quantity: uint64(quantity),
		}
		return "type.googleapis.com/types.MessageWithdrawCollateral", msg, nil
	case "set_treasury_cut":
		authority, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode authority: %w", err)
		}
		treasuryCutBps, _ := fields["treasuryCutBps"].(float64)
		msg := &contract.MessageSetTreasuryCut{
			Authority:      authority,
			TreasuryCutBps: uint64(treasuryCutBps),
		}
		return "type.googleapis.com/types.MessageSetTreasuryCut", msg, nil
	case "mint_nusd":
		owner, err := hex.DecodeString(signerAddrHex)
		if err != nil {
			return "", nil, fmt.Errorf("decode owner: %w", err)
		}
		collateralQuantity, _ := fields["collateralQuantity"].(float64)
		nusdAmountRequested, _ := fields["nusdAmountRequested"].(float64)
		msg := &contract.MessageMintNusd{
			VaultId:             str(fields["vaultId"]),
			Owner:               owner,
			CollateralAssetId:   str(fields["collateralAssetId"]),
			CollateralQuantity:  uint64(collateralQuantity),
			NusdAmountRequested: uint64(nusdAmountRequested),
		}
		return "type.googleapis.com/types.MessageMintNusd", msg, nil
	default:
		return "", nil, fmt.Errorf("unknown or not-yet-wired msgType: %s", msgType)
	}
}

func str(v interface{}) string {
	s, _ := v.(string)
	return s
}

func buildSignAndSendTx(rpcURL string, signerKey *keyGroup, msgType, typeURL string, msgProto proto.Message, fee, networkID, chainID, height uint64) (string, error) {
	msgBytes, err := proto.Marshal(msgProto)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	msgAny := &anypb.Any{TypeUrl: typeURL, Value: msgBytes}
	txTime := uint64(time.Now().UnixMicro())

	tx := &contract.Transaction{
		MessageType:   msgType,
		Msg:           msgAny,
		Signature:     nil,
		CreatedHeight: height,
		Time:          txTime,
		Fee:           fee,
		Memo:          "",
		NetworkId:     networkID,
		ChainId:       chainID,
	}
	signBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("marshal sign bytes: %w", err)
	}

	privKey, err := crypto.StringToBLS12381PrivateKey(signerKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	signature := privKey.Sign(signBytes)

	pubKeyBytes, err := hex.DecodeString(signerKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}

	txJSON := map[string]interface{}{
		"type":       msgType,
		"msgTypeUrl": typeURL,
		"msgBytes":   hex.EncodeToString(msgBytes),
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
	body, err := json.MarshalIndent(txJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal tx json: %w", err)
	}

	respBody, err := postRawJSON(rpcURL+"/v1/tx", string(body))
	if err != nil {
		return "", fmt.Errorf("post tx: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(respBody, &txHash); err != nil {
		return "", fmt.Errorf("parse tx response: %w, body=%s", err, string(respBody))
	}
	return txHash, nil
}

type keyGroup struct {
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func keystoreGetKey(rpcURL, address, password string) (*keyGroup, error) {
	reqJSON := fmt.Sprintf(`{"address":"%s","password":"%s"}`, address, password)
	respBody, err := postRawJSON(rpcURL+"/v1/admin/keystore-get", reqJSON)
	if err != nil {
		return nil, err
	}
	var kg keyGroup
	if err := json.Unmarshal(respBody, &kg); err != nil {
		return nil, fmt.Errorf("parse keystore-get response: %w, body=%s", err, string(respBody))
	}
	return &kg, nil
}

func getHeight(rpcURL string) (uint64, error) {
	respBody, err := postRawJSON(rpcURL+"/v1/query/height", "{}")
	if err != nil {
		return 0, err
	}
	var result struct {
		Height uint64 `json:"height"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, err
	}
	return result.Height, nil
}

func postRawJSON(url, jsonBody string) ([]byte, error) {
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

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
