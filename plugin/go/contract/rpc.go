package contract

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

/*
This file is the plugin's own HTTP server, where a builder exposes custom,
chain-specific RPC endpoints.

Canopy core only exposes a single, generic, read-only transport over the unix socket:
`Plugin.QueryState(height, read)`, which returns raw key/value state at a historical height
(0 = latest committed). The plugin process owns its HTTP server entirely, so builders may register
as many routes as they want and decode their own keys/protobufs into whatever response shapes they
like. Canopy never needs to know about chain-specific endpoints.
*/

// StartRPCServer() launches the plugin's own HTTP server.
func (p *Plugin) StartRPCServer() {
	addr := p.config.RPCAddress
	if addr == "" {
		log.Println("plugin RPC server disabled (no rpcAddress configured)")
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/query/markets", p.handleQueryMarkets)
	mux.HandleFunc("/v1/query/lenderposition", p.handleQueryLenderPosition)
	log.Printf("plugin RPC server (%s) listening on %s", PluginBuild, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("plugin RPC server error: %v", err)
	}
}

// handleQueryMarkets serves GET /v1/query/markets?marketId=<id>
// Returns the decoded Market record ({16}) for the given market_id, reading
// the latest committed state via the detached QueryState() path.
func (p *Plugin) handleQueryMarkets(w http.ResponseWriter, r *http.Request) {
	marketId := r.URL.Query().Get("marketId")
	if marketId == "" {
		http.Error(w, `{"error":"missing marketId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateMarketID(marketId); err != nil {
		http.Error(w, `{"error":"invalid marketId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	key := KeyForMarket(marketId)
	queryId := rand.Uint64()

	resp, pErr := p.QueryState(0 /* latest committed */, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: queryId, Key: key}},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 || len(resp.Results[0].Entries[0].Value) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "market not found", "marketId": marketId})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	market := &Market{}
	if err := proto.Unmarshal(raw, market); err != nil {
		http.Error(w, `{"error":"failed to decode market record: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	marketJSON, err := protojson.Marshal(market)
	if err != nil {
		http.Error(w, `{"error":"failed to encode market json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(marketJSON)
}

// handleQueryLenderPosition serves GET /v1/query/lenderposition?marketId=<id>&address=<hex>
// Returns the decoded LenderPosition record ({24}) for the given market_id/address pair,
// reading the latest committed state via the detached QueryState() path.
func (p *Plugin) handleQueryLenderPosition(w http.ResponseWriter, r *http.Request) {
	marketId := r.URL.Query().Get("marketId")
	if marketId == "" {
		http.Error(w, `{"error":"missing marketId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateMarketID(marketId); err != nil {
		http.Error(w, `{"error":"invalid marketId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	addressHex := r.URL.Query().Get("address")
	if addressHex == "" {
		http.Error(w, `{"error":"missing address query param"}`, http.StatusBadRequest)
		return
	}
	address, err := hex.DecodeString(addressHex)
	if err != nil {
		http.Error(w, `{"error":"invalid address hex: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if len(address) != 20 {
		http.Error(w, `{"error":"address must decode to 20 bytes"}`, http.StatusBadRequest)
		return
	}

	key := KeyForLenderPosition(marketId, address)
	queryId := rand.Uint64()

	resp, pErr := p.QueryState(0 /* latest committed */, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: queryId, Key: key}},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 || len(resp.Results[0].Entries[0].Value) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "lender position not found", "marketId": marketId, "address": addressHex})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	position := &LenderPosition{}
	if err := proto.Unmarshal(raw, position); err != nil {
		http.Error(w, `{"error":"failed to decode lender position: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	positionJSON, err := protojson.Marshal(position)
	if err != nil {
		http.Error(w, `{"error":"failed to encode lender position json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(positionJSON)
}
