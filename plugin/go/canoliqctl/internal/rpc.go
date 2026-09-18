package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// postJSONStatus is PostJSON without the "non-2xx is an error" rule. The
// confirmation path needs the status code itself: tx-by-hash answers 404 for a
// transaction that was rejected AND for one that was never seen, so a 404 is
// data rather than a transport failure.
func postJSONStatus(url string, body string) ([]byte, int, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
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

// TxOutcome is the resolved fate of a submitted transaction.
type TxOutcome struct {
	Hash      string
	Committed bool   // landed in a block
	Height    uint64 // block height, when committed
	Failed    bool   // rejected and dropped from the mempool
	Code      uint64 // failure code, when Failed
	Module    string // failure module, when Failed
	Msg       string // failure message, when Failed
}

// Error renders a failed outcome the way canoliqctl reports it.
func (o *TxOutcome) Error() string {
	if !o.Failed {
		return ""
	}
	if o.Module != "" {
		return fmt.Sprintf("%s (code %d, module %s)", o.Msg, o.Code, o.Module)
	}
	return fmt.Sprintf("%s (code %d)", o.Msg, o.Code)
}

// ConfirmTx resolves what actually happened to txHash, polling until one of the
// three terminal answers arrives or timeout elapses.
//
// Two endpoints are needed, not one. `/v1/tx` admits a transaction on
// CheckBasic alone — no chain-id check, no signature check, no state — so it
// returns a hash for transactions that are about to be rejected. Real
// validation runs on the next mempool re-check, and a rejected transaction is
// added to the failed-tx cache and dropped. It therefore never appears in
// tx-by-hash at all, and tx-by-hash answers 404 identically for "rejected" and
// "never existed". failed-txs is the only endpoint carrying the reason.
//
// The failed-tx cache has a ~5 minute TTL server-side, so confirmation has to
// start promptly after submission rather than being deferred.
func ConfirmTx(rpcURL, signerAddress, txHash string, timeout time.Duration) (*TxOutcome, error) {
	deadline := time.Now().Add(timeout)
	for {
		if o := queryFailed(rpcURL, signerAddress, txHash); o != nil {
			return o, nil
		}
		if o := queryByHash(rpcURL, txHash); o != nil && o.Committed {
			return o, nil
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("tx %s not confirmed within %s (it may still be pending; re-check with tx-by-hash)", txHash, timeout)
		}
		time.Sleep(time.Second)
	}
}

// queryByHash returns a committed outcome, or nil when the tx is absent or
// still pending. A 404 is not an error here — see ConfirmTx.
func queryByHash(rpcURL, txHash string) *TxOutcome {
	body, status, err := postJSONStatus(rpcURL+"/v1/query/tx-by-hash", fmt.Sprintf(`{"hash":%q}`, txHash))
	if err != nil || status != http.StatusOK {
		return nil
	}
	var r struct {
		TxHash    string `json:"txHash"`
		Height    uint64 `json:"height"`
		Committed bool   `json:"committed"`
	}
	if json.Unmarshal(body, &r) != nil {
		return nil
	}
	return &TxOutcome{Hash: txHash, Committed: r.Committed, Height: r.Height}
}

// queryFailed looks txHash up in the node's failed-tx cache, returning the
// recorded reason. Scoped to the signer's address so the scan stays small.
func queryFailed(rpcURL, signerAddress, txHash string) *TxOutcome {
	body, status, err := postJSONStatus(rpcURL+"/v1/query/failed-txs",
		fmt.Sprintf(`{"address":%q,"perPage":100}`, signerAddress))
	if err != nil || status != http.StatusOK {
		return nil
	}
	var page struct {
		Results []struct {
			TxHash string `json:"txHash"`
			Error  struct {
				Code   uint64 `json:"code"`
				Module string `json:"module"`
				Msg    string `json:"msg"`
			} `json:"error"`
		} `json:"results"`
	}
	if json.Unmarshal(body, &page) != nil {
		return nil
	}
	for _, r := range page.Results {
		if !strings.EqualFold(r.TxHash, txHash) {
			continue
		}
		return &TxOutcome{
			Hash:   txHash,
			Failed: true,
			Code:   r.Error.Code,
			Module: r.Error.Module,
			Msg:    r.Error.Msg,
		}
	}
	return nil
}
