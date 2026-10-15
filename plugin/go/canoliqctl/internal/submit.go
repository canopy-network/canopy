package internal

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// TxParams bundles the per-call parameters that are common to every plugin
// transaction. Fee defaults to 10000 uCNPY when zero (matches DefaultParams
// minimums in plugin/go/canoliq/config.go::DefaultParams).
type TxParams struct {
	NetworkID uint64
	ChainID   uint64
	Fee       uint64
}

// SubmitPluginTx serializes msg as a canoliq plugin transaction, signs it with
// signer's BLS12-381 private key, and POSTs the JSON envelope to /v1/tx.
// Returns the tx hash returned by the node.
//
// msgType must be one of the keys registered in MsgTypeURL (e.g.,
// "canoliq_deposit"). The signed envelope uses the msgTypeUrl/msgBytes
// path because plugin-only types are unknown to the server's tx schema —
// the server forwards the raw bytes to the plugin process.
func SubmitPluginTx(rpcURL string, signer *Key, msgType string, msg proto.Message, params TxParams) (string, error) {
	typeURL, ok := MsgTypeURL[msgType]
	if !ok {
		return "", fmt.Errorf("unknown canoliq message type %q", msgType)
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %v", err)
	}
	any := &anypb.Any{TypeUrl: typeURL, Value: msgBytes}

	height, err := GetHeight(rpcURL)
	if err != nil {
		return "", fmt.Errorf("get height: %v", err)
	}
	txTime := uint64(time.Now().UnixMicro())

	signBytes, err := SignBytes(msgType, any, txTime, height, params.Fee, "", params.NetworkID, params.ChainID)
	if err != nil {
		return "", fmt.Errorf("compute sign bytes: %v", err)
	}

	priv, err := PrivateKeyFromHex(signer.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("parse private key: %v", err)
	}
	signature := priv.Sign(signBytes)

	pubBytes, err := hex.DecodeString(signer.PublicKey)
	if err != nil {
		return "", fmt.Errorf("decode public key: %v", err)
	}

	envelope := map[string]interface{}{
		"type":       msgType,
		"msgTypeUrl": typeURL,
		"msgBytes":   hex.EncodeToString(msgBytes),
		"signature": map[string]string{
			"publicKey": hex.EncodeToString(pubBytes),
			"signature": hex.EncodeToString(signature),
		},
		"time":          txTime,
		"createdHeight": height,
		"fee":           params.Fee,
		"memo":          "",
		"networkID":     params.NetworkID,
		"chainID":       params.ChainID,
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %v", err)
	}

	respBody, err := PostJSON(rpcURL+"/v1/tx", string(envelopeJSON))
	if err != nil {
		return "", fmt.Errorf("post tx: %v", err)
	}
	var hash string
	if err := json.Unmarshal(respBody, &hash); err != nil {
		return "", fmt.Errorf("parse tx response: %v: %s", err, respBody)
	}
	return hash, nil
}
