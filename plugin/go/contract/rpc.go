package contract

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"

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
	mux.HandleFunc("/v1/query/borrowerposition", p.handleQueryBorrowerPosition)
	mux.HandleFunc("/v1/query/pool", p.handleQueryPool)
	mux.HandleFunc("/v1/query/reservefund", p.handleQueryReserveFund)
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

// handleQueryBorrowerPosition serves GET /v1/query/borrowerposition?marketId=<id>&address=<hex>
// Returns the decoded BorrowerPosition record ({17}) for the given
// (market_id, address) pair. Added to close a visibility gap: before this
// handler existed, borrower position state (collateral_quantity,
// debt_principal, borrow_index_at_open) was only inferable indirectly via
// transaction history, which made a real bug (repay.go deleting collateral
// alongside debt on full repayment) much slower to diagnose than it should
// have been. Mirrors handleQueryLenderPosition's exact structure -- same
// composite-key shape, same validation order, same error/response format.
func (p *Plugin) handleQueryBorrowerPosition(w http.ResponseWriter, r *http.Request) {
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

	posKey := KeyForBorrowerPosition(marketId, address)
	bIndexKey := KeyForBorrowIndex(marketId)
	posQueryId := rand.Uint64()
	bIndexQueryId := rand.Uint64()

	resp, pErr := p.QueryState(0 /* latest committed */, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: posQueryId, Key: posKey},
			{QueryId: bIndexQueryId, Key: bIndexKey},
		},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// QueryId lives on PluginReadResult, not on individual entries (see
	// plugin.pb.go's PluginReadResult / PluginStateEntry definitions) --
	// matches the pattern already used in deposit.go, contract.go, and
	// price_resolve.go. Each of our two queries is a single-key lookup, so
	// the matching result's Entries has at most one element.
	var posRaw, bIndexRaw []byte
	for _, result := range resp.Results {
		if result.QueryId == posQueryId && len(result.Entries) > 0 {
			posRaw = result.Entries[0].Value
		}
		if result.QueryId == bIndexQueryId && len(result.Entries) > 0 {
			bIndexRaw = result.Entries[0].Value
		}
	}

	if len(posRaw) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "borrower position not found", "marketId": marketId, "address": addressHex})
		return
	}

	position := &BorrowerPosition{}
	if err := proto.Unmarshal(posRaw, position); err != nil {
		http.Error(w, `{"error":"failed to decode borrower position: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	positionJSON, err := protojson.Marshal(position)
	if err != nil {
		http.Error(w, `{"error":"failed to encode borrower position json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Additive field: debtPrincipal above is the raw stored principal as of
	// the last write to this position, NOT the current owed amount -- see
	// ScaledDebt()'s doc comment (AYIS Section 6, ARCM Section 2.2's mandatory
	// rule that pos.DebtPrincipal alone must never be treated as current debt).
	// currentDebt is included here so callers of this endpoint (liquidation
	// bots, frontends, manual inspection) are not misled into treating raw
	// principal as owed debt. Computed best-effort: if B_index is missing
	// (should not happen for any market with an open position), currentDebt
	// is simply omitted rather than failing the whole request, since the raw
	// position data is still valid and useful on its own.
	var responseMap map[string]interface{}
	if err := json.Unmarshal(positionJSON, &responseMap); err != nil {
		http.Error(w, `{"error":"failed to build response json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if len(bIndexRaw) > 0 {
		bIndexNow := DecodeUint128(bIndexRaw)
		currentDebt := ScaledDebt(position, bIndexNow)
		responseMap["currentDebt"] = strconv.FormatUint(currentDebt, 10)
	}

	finalJSON, err := json.Marshal(responseMap)
	if err != nil {
		http.Error(w, `{"error":"failed to encode final response json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(finalJSON)
}

// handleQueryPool serves GET /v1/query/pool?marketId=<id>&purpose=supply|collateral
// Returns the decoded Pool record for the given market's escrow pool, selected
// by purpose (PoolPurposeSupply or PoolPurposeCollateral -- see pool_id.go).
// Added alongside the custody fix (deposit/withdraw/deposit_collateral/
// withdraw_collateral/borrow/repay all now move real Account.Amount/Pool.Amount
// balances instead of pure bookkeeping) specifically so a pool's real balance
// is independently verifiable on-chain, the same way handleQueryLenderPosition
// made LenderPosition state independently verifiable rather than only
// inferable from transaction history. Mirrors handleQueryMarkets' single-key
// (non-composite) query shape, since a pool id is derived from (marketId,
// purpose) rather than read from a URL-provided key directly.
func (p *Plugin) handleQueryPool(w http.ResponseWriter, r *http.Request) {
	marketId := r.URL.Query().Get("marketId")
	if marketId == "" {
		http.Error(w, `{"error":"missing marketId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateMarketID(marketId); err != nil {
		http.Error(w, `{"error":"invalid marketId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	purposeStr := r.URL.Query().Get("purpose")
	var purpose PoolPurpose
	switch purposeStr {
	case "supply":
		purpose = PoolPurposeSupply
	case "collateral":
		purpose = PoolPurposeCollateral
	default:
		http.Error(w, `{"error":"purpose query param must be \"supply\" or \"collateral\""}`, http.StatusBadRequest)
		return
	}

	poolId := KeyForMarketPoolId(marketId, purpose)
	key := KeyForFeePool(poolId)
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
		// A pool with zero balance is deleted by SetPool()'s own convention
		// (fsm/account.go: "if the pool has a 0 balance { return s.Delete(...) }"),
		// so "not found" here means balance == 0, not necessarily an error --
		// report it as a zero-balance Pool rather than a 404, since a
		// newly-created market's pools legitimately don't exist yet until the
		// first deposit/deposit_collateral.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": strconv.FormatUint(poolId, 10), "amount": "0", "note": "pool has zero balance or does not yet exist"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	pool := &Pool{}
	if err := proto.Unmarshal(raw, pool); err != nil {
		http.Error(w, `{"error":"failed to decode pool record: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	poolJSON, err := protojson.Marshal(pool)
	if err != nil {
		http.Error(w, `{"error":"failed to encode pool json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(poolJSON)
}

// handleQueryReserveFund serves GET /v1/query/reservefund?marketId=<id>
// Returns R_fund ({18}) for the given market, decoded via DecodeUint128 --
// same raw-read shape as handleQueryPool (single scalar keyed by marketId,
// not a composite key, not a protobuf record). Added specifically because
// no prior route exposed R_fund directly (ARBOR_HANDOFF_LAYER2.md Section
// 2.4): its value could previously only be inferred indirectly via a
// liquidation's covered/badDebt outcome. Mirrors handleQueryPool's
// zero-value convention: a market with R_fund == 0 (e.g. newly created,
// no interest/repay/liquidation inflow yet) returns {"amount":"0"} rather
// than a 404, since a legitimately-zero reserve is not an error state.
func (p *Plugin) handleQueryReserveFund(w http.ResponseWriter, r *http.Request) {
	marketId := r.URL.Query().Get("marketId")
	if marketId == "" {
		http.Error(w, `{"error":"missing marketId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateMarketID(marketId); err != nil {
		http.Error(w, `{"error":"invalid marketId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	key := KeyForReserveFund(marketId)
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
		json.NewEncoder(w).Encode(map[string]string{"marketId": marketId, "amount": "0", "note": "R_fund has zero balance or does not yet exist"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	rFund := DecodeUint128(raw)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"marketId": marketId, "amount": rFund.String()})
}
