package contract

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

/*
This file contains an EXAMPLE HTTP server that demonstrates how a plugin builder exposes their own
custom RPC endpoints for their chain.

Canopy core only exposes a single, generic, read-only transport over the unix socket:
`Plugin.QueryState(height, read)`, which returns raw key/value state at a historical height. The
plugin process owns its HTTP server entirely, so builders may register as many routes as they want
and decode their own keys/protobufs into whatever response shapes they like. Canopy never needs to
know about chain-specific endpoints.
*/

// StartRPCServer() launches the plugin's own HTTP server exposing custom, chain-specific RPC endpoints.
// Builders are free to register any number of routes on the mux; each handler uses the detached,
// read-only QueryState() path to fetch state snapshots from Canopy.
func (p *Plugin) StartRPCServer() {
	// resolve the listen address from config
	addr := p.config.RPCAddress
	// if no address is configured, the RPC server is disabled
	if addr == "" {
		log.Println("plugin RPC server disabled (no rpcAddress configured)")
		return
	}
	// build a router and register as many custom endpoints as desired
	mux := http.NewServeMux()
	// GET /v1/query/account?address=<hex>&height=<uint64>
	mux.HandleFunc("/v1/query/account", p.handleQueryAccount)
	// GET /v1/query/pool?id=<uint64>&height=<uint64>
	mux.HandleFunc("/v1/query/pool", p.handleQueryPool)
	// log and serve
	log.Printf("plugin RPC server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("plugin RPC server error: %v", err)
	}
}

// handleQueryAccount() is an example endpoint returning an account's balance at an optional height
func (p *Plugin) handleQueryAccount(w http.ResponseWriter, r *http.Request) {
	// parse the address (hex) from the query string
	address, err := hex.DecodeString(r.URL.Query().Get("address"))
	if err != nil || len(address) != 20 {
		writeJSONError(w, http.StatusBadRequest, "address must be a 20-byte hex string")
		return
	}
	// parse the optional height (0 = latest committed)
	height := parseHeight(r)
	// query the read-only state snapshot for the account key
	value, pErr := p.queryValue(height, KeyForAccount(address))
	if pErr != nil {
		writeJSONError(w, http.StatusInternalServerError, pErr.Error())
		return
	}
	// decode the raw bytes into the plugin's own Account type
	account := new(Account)
	if uErr := Unmarshal(value, account); uErr != nil {
		writeJSONError(w, http.StatusInternalServerError, uErr.Error())
		return
	}
	// shape and write the response (builders fully control the response shape)
	writeJSON(w, map[string]any{
		"address": hex.EncodeToString(address),
		"amount":  account.Amount,
		"height":  height,
	})
}

// handleQueryPool() is an example endpoint returning a pool's balance at an optional height
func (p *Plugin) handleQueryPool(w http.ResponseWriter, r *http.Request) {
	// parse the pool id from the query string
	id, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "id must be an unsigned integer")
		return
	}
	// parse the optional height (0 = latest committed)
	height := parseHeight(r)
	// query the read-only state snapshot for the fee pool key
	value, pErr := p.queryValue(height, KeyForFeePool(id))
	if pErr != nil {
		writeJSONError(w, http.StatusInternalServerError, pErr.Error())
		return
	}
	// decode the raw bytes into the plugin's own Pool type
	pool := new(Pool)
	if uErr := Unmarshal(value, pool); uErr != nil {
		writeJSONError(w, http.StatusInternalServerError, uErr.Error())
		return
	}
	// shape and write the response
	writeJSON(w, map[string]any{
		"id":     id,
		"amount": pool.Amount,
		"height": height,
	})
}

// queryValue() is a small helper that performs a single-key detached read and returns the raw value bytes
func (p *Plugin) queryValue(height uint64, key []byte) ([]byte, *PluginError) {
	// execute a detached, read-only state query for the single key
	resp, err := p.QueryState(height, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: rand.Uint64(), Key: key}},
	})
	if err != nil {
		return nil, err
	}
	// extract the first entry value if present (nil means 'not found')
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	return resp.Results[0].Entries[0].Value, nil
}

// parseHeight() reads the optional 'height' query parameter, defaulting to 0 (latest committed)
func parseHeight(r *http.Request) uint64 {
	height, err := strconv.ParseUint(r.URL.Query().Get("height"), 10, 64)
	if err != nil {
		return 0
	}
	return height
}

// writeJSON() writes a JSON success response
func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

// writeJSONError() writes a JSON error response with the given status code
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
