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

The endpoints below are intentionally plugin-specific (faucet and reward records) so they showcase
data that does NOT exist in the Canopy node's own RPC. Account/pool queries already exist in core,
so they make poor examples of a *custom* endpoint.
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
	// GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
	mux.HandleFunc("/v1/query/faucets", p.handleQueryFaucets)
	// GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
	mux.HandleFunc("/v1/query/rewards", p.handleQueryRewards)
	// log the build marker and the registered routes so the running version is obvious in the log
	log.Printf("plugin RPC server (%s) listening on %s", PluginBuild, addr)
	log.Printf("plugin RPC routes registered: GET /v1/query/faucets, GET /v1/query/rewards")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("plugin RPC server error: %v", err)
	}
}

// handleQueryFaucets() returns faucet records. With ?address=<hex> it returns a single recipient's
// record; otherwise it returns every faucet record via a range read over the faucet prefix.
func (p *Plugin) handleQueryFaucets(w http.ResponseWriter, r *http.Request) {
	// optional single-record lookup by recipient address
	if addrHex := r.URL.Query().Get("address"); addrHex != "" {
		address, err := hex.DecodeString(addrHex)
		if err != nil || len(address) != 20 {
			writeJSONError(w, http.StatusBadRequest, "address must be a 20-byte hex string")
			return
		}
		height := parseHeight(r)
		value, pErr := p.queryValue(height, KeyForFaucet(address))
		if pErr != nil {
			writeJSONError(w, http.StatusInternalServerError, pErr.Error())
			return
		}
		faucet := new(Faucet)
		if uErr := Unmarshal(value, faucet); uErr != nil {
			writeJSONError(w, http.StatusInternalServerError, uErr.Error())
			return
		}
		writeJSON(w, map[string]any{"faucet": faucetToJSON(faucet), "height": height})
		return
	}
	// otherwise return all faucet records via a range read
	height := parseHeight(r)
	entries, pErr := p.queryRange(height, FaucetPrefix())
	if pErr != nil {
		writeJSONError(w, http.StatusInternalServerError, pErr.Error())
		return
	}
	// decode each entry into the plugin's own Faucet type
	faucets := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		faucet := new(Faucet)
		if uErr := Unmarshal(entry.Value, faucet); uErr != nil {
			writeJSONError(w, http.StatusInternalServerError, uErr.Error())
			return
		}
		faucets = append(faucets, faucetToJSON(faucet))
	}
	writeJSON(w, map[string]any{"faucets": faucets, "count": len(faucets), "height": height})
}

// handleQueryRewards() returns reward records. With ?address=<hex> it returns a single recipient's
// record; otherwise it returns every reward record via a range read over the reward prefix.
func (p *Plugin) handleQueryRewards(w http.ResponseWriter, r *http.Request) {
	// optional single-record lookup by recipient address
	if addrHex := r.URL.Query().Get("address"); addrHex != "" {
		address, err := hex.DecodeString(addrHex)
		if err != nil || len(address) != 20 {
			writeJSONError(w, http.StatusBadRequest, "address must be a 20-byte hex string")
			return
		}
		height := parseHeight(r)
		value, pErr := p.queryValue(height, KeyForReward(address))
		if pErr != nil {
			writeJSONError(w, http.StatusInternalServerError, pErr.Error())
			return
		}
		reward := new(Reward)
		if uErr := Unmarshal(value, reward); uErr != nil {
			writeJSONError(w, http.StatusInternalServerError, uErr.Error())
			return
		}
		writeJSON(w, map[string]any{"reward": rewardToJSON(reward), "height": height})
		return
	}
	// otherwise return all reward records via a range read
	height := parseHeight(r)
	entries, pErr := p.queryRange(height, RewardPrefix())
	if pErr != nil {
		writeJSONError(w, http.StatusInternalServerError, pErr.Error())
		return
	}
	// decode each entry into the plugin's own Reward type
	rewards := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		reward := new(Reward)
		if uErr := Unmarshal(entry.Value, reward); uErr != nil {
			writeJSONError(w, http.StatusInternalServerError, uErr.Error())
			return
		}
		rewards = append(rewards, rewardToJSON(reward))
	}
	writeJSON(w, map[string]any{"rewards": rewards, "count": len(rewards), "height": height})
}

// faucetToJSON() shapes a Faucet record into a JSON-friendly map (hex-encoding addresses)
func faucetToJSON(faucet *Faucet) map[string]any {
	return map[string]any{
		"recipientAddress": hex.EncodeToString(faucet.RecipientAddress),
		"totalAmount":      faucet.TotalAmount,
		"count":            faucet.Count,
	}
}

// rewardToJSON() shapes a Reward record into a JSON-friendly map (hex-encoding addresses)
func rewardToJSON(reward *Reward) map[string]any {
	return map[string]any{
		"recipientAddress": hex.EncodeToString(reward.RecipientAddress),
		"lastAdminAddress": hex.EncodeToString(reward.LastAdminAddress),
		"totalAmount":      reward.TotalAmount,
		"count":            reward.Count,
	}
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

// queryRange() is a small helper that performs a detached range read over a key prefix
func (p *Plugin) queryRange(height uint64, prefix []byte) ([]*PluginStateEntry, *PluginError) {
	// execute a detached, read-only range query over the prefix
	resp, err := p.QueryState(height, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{{QueryId: rand.Uint64(), Prefix: prefix}},
	})
	if err != nil {
		return nil, err
	}
	// return the entries of the first (only) range result, if present
	if len(resp.Results) == 0 {
		return nil, nil
	}
	return resp.Results[0].Entries, nil
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
