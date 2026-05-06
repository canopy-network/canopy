package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PostJSON POSTs a raw JSON body to url and returns the response body. Non-2xx
// responses produce an error containing the status and body for diagnosis.
func PostJSON(url string, body string) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
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

// GetHeight returns the current block height from the node's query RPC.
func GetHeight(rpcURL string) (uint64, error) {
	body, err := PostJSON(rpcURL+"/v1/query/height", "{}")
	if err != nil {
		return 0, err
	}
	var result struct {
		Height uint64 `json:"height"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("parse height: %v: %s", err, body)
	}
	return result.Height, nil
}

// GetAccountBalance returns the CNPY balance (uCNPY) of address.
func GetAccountBalance(rpcURL, address string) (uint64, error) {
	body, err := PostJSON(rpcURL+"/v1/query/account", fmt.Sprintf(`{"address":%q}`, address))
	if err != nil {
		return 0, err
	}
	var result struct {
		Amount uint64 `json:"amount"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("parse account: %v: %s", err, body)
	}
	return result.Amount, nil
}

// WaitForTxInclusion polls /v1/query/txs-by-sender until txHash appears or
// timeout elapses. Returns nil when the tx is observed in a block.
func WaitForTxInclusion(rpcURL, sender, txHash string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		body, err := PostJSON(rpcURL+"/v1/query/txs-by-sender", fmt.Sprintf(`{"address":%q,"perPage":20}`, sender))
		if err == nil {
			var result struct {
				Results []struct {
					TxHash string `json:"txHash"`
				} `json:"results"`
			}
			if json.Unmarshal(body, &result) == nil {
				for _, tx := range result.Results {
					if tx.TxHash == txHash {
						return nil
					}
				}
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("tx %s not included within %s", txHash, timeout)
}
