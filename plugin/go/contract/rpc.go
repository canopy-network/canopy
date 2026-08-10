package contract

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
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
	mux.HandleFunc("/v1/query/lossfactor", p.handleQueryLossFactor)
	mux.HandleFunc("/v1/query/treasury", p.handleQueryTreasury)
	mux.HandleFunc("/v1/query/governanceparams", p.handleQueryGovernanceParams)
	mux.HandleFunc("/v1/query/all-markets", p.handleQueryAllMarkets)
	mux.HandleFunc("/v1/query/all-borrower-positions", p.handleQueryAllBorrowerPositions)
	mux.HandleFunc("/v1/query/prices", p.handleQueryPrices)
	mux.HandleFunc("/v1/query/interestremainder", p.handleQueryInterestRemainder)
	mux.HandleFunc("/v1/query/nasmtier", p.handleQueryNasmTier)
	mux.HandleFunc("/v1/query/nasmvaultpool", p.handleQueryNasmVaultPool)
	mux.HandleFunc("/v1/query/nasmvault", p.handleQueryNasmVault)
	mux.HandleFunc("/v1/query/all-nasm-vaults", p.handleQueryAllNasmVaults)
	mux.HandleFunc("/v1/query/nusdbalance", p.handleQueryNusdBalance)
	mux.HandleFunc("/v1/query/nusdsupply", p.handleQueryNusdSupply)
	mux.HandleFunc("/v1/query/stabilityfeeindex", p.handleQueryStabilityFeeIndex)
	mux.HandleFunc("/v1/query/waterfall-events", p.handleQueryWaterfallEvents)
	mux.HandleFunc("/v1/query/nasmtierbacking", p.handleQueryNasmTierBacking)
	mux.HandleFunc("/v1/query/emergencymode", p.handleQueryEmergencyMode)
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
		// [FIXED] ScaledDebt() can now return an error (ErrScaledDebtOverflow).
		// Read-only query context, no transaction to revert -- same
		// graceful-degradation spirit as the missing-bIndexRaw case this
		// block already handles: omit currentDebt rather than fail the
		// whole request.
		if currentDebt, sdErr := ScaledDebt(position, bIndexNow); sdErr == nil {
			responseMap["currentDebt"] = strconv.FormatUint(currentDebt, 10)
		}
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

// handleQueryLossFactor serves GET /v1/query/lossfactor?marketId=<id>
// Returns the market's current loss_factor ({27}, uint128, RAY-scaled) as a
// decimal string. RAY (1e18) means no haircut has ever been applied; a value
// below RAY means Layer 4 has fired at least once (AYIS Section 5.4.3);
// exactly 0 means the market has been fully exhausted (Insolvent, I11).
// Added this session specifically to make ApplyLossFactor's real on-chain
// effect observable -- prior to this route, no query surfaced loss_factor
// at all, making Layer 4's haircut branch unverifiable except by indirect
// inference (see this session's own liquidate_position test for why that
// was insufficient).
func (p *Plugin) handleQueryLossFactor(w http.ResponseWriter, r *http.Request) {
	marketId := r.URL.Query().Get("marketId")
	if marketId == "" {
		http.Error(w, `{"error":"missing marketId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateMarketID(marketId); err != nil {
		http.Error(w, `{"error":"invalid marketId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	key := KeyForLossFactor(marketId)
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
		json.NewEncoder(w).Encode(map[string]string{"marketId": marketId, "note": "loss_factor not yet initialized -- market may not exist"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	lossFactor := DecodeUint128(raw)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"marketId": marketId, "lossFactor": lossFactor.String()})
}

// handleQueryEmergencyMode serves GET /v1/query/emergencymode?assetId=CNPY
// Returns the requested asset's {21} EmergencyModeFlag record (Active,
// Trigger, TriggeredBlock, TriggeredBy). Added because set_emergency_mode
// (contract.go:296/370, oracle_admin.go) had a full write path -- handler,
// dispatch, accessor -- but no query surface at all prior to this route,
// making a submitted set_emergency_mode tx unverifiable except indirectly.
// Mirrors handleQueryNusdBalance's proto-decode shape (raw QueryState ->
// proto.Unmarshal -> protojson.Marshal) rather than GetEmergencyMode's own
// *Contract-based accessor, since that accessor is scoped to DeliverTx
// handler use, not RPC. Zero-value convention: an asset that has never
// had emergency mode set returns a zero-value EmergencyModeFlag (Active
// omitted/false, Trigger=EMERGENCY_TRIGGER_NONE per proto3 default
// field-omission), not a 404 -- matching GetEmergencyMode's own
// found=false-is-normal convention documented at state_accessors.go:883.
func (p *Plugin) handleQueryEmergencyMode(w http.ResponseWriter, r *http.Request) {
	assetId := r.URL.Query().Get("assetId")
	if assetId == "" {
		http.Error(w, `{"error":"missing assetId query param"}`, http.StatusBadRequest)
		return
	}

	key := KeyForEmergencyMode(assetId)
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
		json.NewEncoder(w).Encode(map[string]string{"assetId": assetId, "active": "false", "note": "emergency mode never set for this asset"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	flag := &EmergencyModeFlag{}
	if err := proto.Unmarshal(raw, flag); err != nil {
		http.Error(w, `{"error":"failed to decode EmergencyModeFlag record: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	flagJSON, err := protojson.Marshal(flag)
	if err != nil {
		http.Error(w, `{"error":"failed to encode emergency mode json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(flagJSON)
}

// handleQueryTreasury serves GET /v1/query/treasury?pool=arbor|nasm
// Returns the requested protocol treasury's current global balance
// (PrefixTreasuryArbor {40} or PrefixTreasuryNASM {41}), decoded via
// DecodeUint128. Unlike handleQueryReserveFund/handleQueryLossFactor, this
// route takes NO marketId param -- T_fund is a single global balance per
// pool, not market-keyed (see bad_debt_layer3.go's own doc comment on this;
// KeyForTreasuryArbor/KeyForTreasuryNASM take no marketID argument either).
// Added same session as Layer 3's wiring into liquidate_position.go
// (Layer3DrawDownArbor) -- prior to this route, T_fund had accessors but no
// query surface at all, the only balance in the four-layer waterfall
// without one (R_fund: /v1/query/reservefund; loss_factor:
// /v1/query/lossfactor; T_fund: none, until now). Mirrors both existing
// handlers' zero-value convention: a pool with balance == 0 (e.g. never
// funded yet -- treasury_cut funding mechanism is not yet built per this
// session's own handoff) returns {"amount":"0"}, not a 404, since a
// legitimately-empty treasury is not an error state.
func (p *Plugin) handleQueryTreasury(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	if pool != "arbor" && pool != "nasm" {
		http.Error(w, `{"error":"missing or invalid pool query param -- must be \"arbor\" or \"nasm\""}`, http.StatusBadRequest)
		return
	}

	var key []byte
	if pool == "arbor" {
		key = KeyForTreasuryArbor()
	} else {
		key = KeyForTreasuryNASM()
	}
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
		json.NewEncoder(w).Encode(map[string]string{"pool": pool, "amount": "0", "note": "treasury has zero balance or does not yet exist"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	tFund := DecodeUint128(raw)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"pool": pool, "amount": tFund.String()})
}

// handleQueryNasmTierBacking serves GET /v1/query/nasmtierbacking
// Returns the single {36} NasmTierBacking record's per-tier totals
// (NASM Spec Section 3.3's mint concentration cap accumulator), plus the
// current total_supply and each tier's live share in bps, computed the
// same way CheckTierConcentrationCap (nasm_tier_backing.go) computes it,
// so this route is directly useful for verifying the cap's actual state
// without re-deriving the math externally. Takes NO query params -- a
// single global record, like handleQueryTreasury/handleQueryGovernanceParams.
func (p *Plugin) handleQueryNasmTierBacking(w http.ResponseWriter, r *http.Request) {
	backingQueryId := rand.Uint64()
	supplyQueryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: backingQueryId, Key: KeyForNasmTierBacking()},
			{QueryId: supplyQueryId, Key: KeyForNusdSupply()},
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

	// entryValue (deposit.go) is the existing, proven helper for pulling a
	// specific query's result out of a batched read by QueryId -- reused
	// directly here rather than re-deriving the same lookup, matching
	// every DeliverTx handler's own established convention (mint_nusd.go,
	// burn_nusd.go, liquidate_nasm_vault.go all use it identically).
	backing := &NasmTierBacking{}
	if raw := entryValue(resp, backingQueryId); len(raw) > 0 {
		if err := proto.Unmarshal(raw, backing); err != nil {
			http.Error(w, `{"error":"unmarshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}
	supply := &NusdSupply{}
	if raw := entryValue(resp, supplyQueryId); len(raw) > 0 {
		if err := proto.Unmarshal(raw, supply); err != nil {
			http.Error(w, `{"error":"unmarshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	var n0ShareBps, n1ShareBps string
	if supply.TotalSupply > 0 {
		n0ShareBps = new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(backing.TierN0Backing)), big.NewInt(10_000)), big.NewInt(int64(supply.TotalSupply))).String()
		n1ShareBps = new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(backing.TierN1Backing)), big.NewInt(10_000)), big.NewInt(int64(supply.TotalSupply))).String()
	} else {
		n0ShareBps = "0"
		n1ShareBps = "0"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tierN0Backing":   strconv.FormatUint(backing.TierN0Backing, 10),
		"tierN1Backing":   strconv.FormatUint(backing.TierN1Backing, 10),
		"totalSupply":     strconv.FormatUint(supply.TotalSupply, 10),
		"tierN0ShareBps":  n0ShareBps,
		"tierN1ShareBps":  n1ShareBps,
		"maxTierShareBps": strconv.FormatUint(MaxTierShareBps, 10),
	})
}

// handleQueryGovernanceParams serves GET /v1/query/governanceparams
// Returns the single global {22} GovernanceParams record (currently just
// treasury_cut_bps -- see arbor_state.proto's own doc comment on this
// record's one-struct-per-key, read-modify-write convention for future
// governance parameters). Takes NO query params -- like
// handleQueryTreasury, this is a single global value, not market-keyed.
// Mirrors handleQueryTreasury's zero-value convention: before governance
// has ever called set_treasury_cut, the {22} key does not exist yet, and
// this returns a zero-value GovernanceParams, not a 404 -- an
// unconfigured governance parameter is not an error state, exactly as an
// empty treasury is not (see that handler's own comment). NOTE: unlike
// handleQueryTreasury's hand-built map (which always includes "amount":
// "0" explicitly), this handler uses protojson.Marshal with default
// options, matching every other protojson.Marshal call in this file --
// proto3's default field-omission means a zero treasury_cut_bps renders
// as {} (the field is simply absent), not {"treasuryCutBps":"0"}. Verified
// live: GET returns {} when unset, confirmed against this file's own
// established protojson convention (no EmitUnpopulated anywhere in this
// file) rather than assumed.
func (p *Plugin) handleQueryGovernanceParams(w http.ResponseWriter, r *http.Request) {
	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: queryId, Key: KeyForGovernanceParams()}},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	params := &GovernanceParams{}
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 && len(resp.Results[0].Entries[0].Value) > 0 {
		raw := resp.Results[0].Entries[0].Value
		if err := proto.Unmarshal(raw, params); err != nil {
			http.Error(w, `{"error":"unmarshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	b, err := protojson.Marshal(params)
	if err != nil {
		http.Error(w, `{"error":"marshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// handleQueryAllMarkets serves GET /v1/query/all-markets
// Returns every Market record by range-iterating the {16} market prefix --
// the same PrefixMarkets walk BeginBlock's interest accrual uses (contract.go),
// exposed over HTTP so the frontend can discover all markets dynamically
// instead of a hand-maintained id list. Emits a bare JSON array of protojson
// Market objects (uint64 fields as decimal strings; omitted status => ACTIVE).
func (p *Plugin) handleQueryAllMarkets(w http.ResponseWriter, r *http.Request) {
	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: queryId, Prefix: JoinLenPrefix(PrefixMarkets)},
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
	out := make([]json.RawMessage, 0)
	for _, result := range resp.Results {
		if result.QueryId != queryId {
			continue
		}
		for _, entry := range result.Entries {
			if len(entry.Value) == 0 {
				continue
			}
			m := &Market{}
			if err := proto.Unmarshal(entry.Value, m); err != nil {
				continue
			}
			b, err := protojson.Marshal(m)
			if err != nil {
				continue
			}
			out = append(out, b)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleQueryPrices serves GET /v1/query/prices?assetId=<id>
// Returns every PriceRecord for an asset by range-iterating the {19} price
// cache scoped to (assetId) -- the same prefix price_resolve.go aggregates for
// quorum/median/deviation -- so the frontend can compute freshness, quorum, and
// the median itself from the raw per-submitter readings. Bare JSON array of
// protojson PriceRecord objects (price & block_height as decimal strings).
func (p *Plugin) handleQueryPrices(w http.ResponseWriter, r *http.Request) {
	assetId := r.URL.Query().Get("assetId")
	if assetId == "" {
		http.Error(w, `{"error":"missing assetId query param"}`, http.StatusBadRequest)
		return
	}
	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: queryId, Prefix: JoinLenPrefix(PrefixPriceCache, []byte(assetId))},
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
	out := make([]json.RawMessage, 0)
	for _, result := range resp.Results {
		if result.QueryId != queryId {
			continue
		}
		for _, entry := range result.Entries {
			if len(entry.Value) == 0 {
				continue
			}
			pr := &PriceRecord{}
			if err := proto.Unmarshal(entry.Value, pr); err != nil {
				continue
			}
			b, err := protojson.Marshal(pr)
			if err != nil {
				continue
			}
			out = append(out, b)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleQueryAllBorrowerPositions serves GET /v1/query/all-borrower-positions
// Returns every BorrowerPosition on chain by range-iterating the {17} prefix --
// the same per-market walk the point handler does, but global, so a liquidator
// dashboard can enumerate every position (the plugin previously exposed borrower
// state only via the per-address point query). For each position it also computes
// the interest-scaled currentDebt by batch-reading that market's B_index ({25}),
// exactly mirroring handleQueryBorrowerPosition's ScaledDebt() call -- so callers
// get the real owed amount, not the raw stored principal (ARCM Section 2.2).
// Bare JSON array of protojson BorrowerPosition objects plus an additive
// currentDebt decimal-string field; positions whose market has no B_index fall
// back to debt_principal so the row is still useful.
func (p *Plugin) handleQueryAllBorrowerPositions(w http.ResponseWriter, r *http.Request) {
	posQueryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: posQueryId, Prefix: JoinLenPrefix(PrefixBorrowerPositions)},
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

	var positions []*BorrowerPosition
	bIndexQid := map[string]uint64{}
	for _, result := range resp.Results {
		if result.QueryId != posQueryId {
			continue
		}
		for _, entry := range result.Entries {
			if len(entry.Value) == 0 {
				continue
			}
			bp := &BorrowerPosition{}
			if err := proto.Unmarshal(entry.Value, bp); err != nil {
				continue
			}
			positions = append(positions, bp)
			if _, seen := bIndexQid[bp.MarketId]; !seen {
				bIndexQid[bp.MarketId] = rand.Uint64()
			}
		}
	}

	// Batch-read each distinct market's B_index so we can scale debt server-side.
	keyReads := make([]*PluginKeyRead, 0, len(bIndexQid))
	for mid, qid := range bIndexQid {
		keyReads = append(keyReads, &PluginKeyRead{QueryId: qid, Key: KeyForBorrowIndex(mid)})
	}
	bIndexRaw := map[string][]byte{}
	if len(keyReads) > 0 {
		if r2, e2 := p.QueryState(0, &PluginStateReadRequest{Keys: keyReads}); e2 == nil && r2.Error == nil {
			qidToMid := map[uint64]string{}
			for mid, qid := range bIndexQid {
				qidToMid[qid] = mid
			}
			for _, result := range r2.Results {
				if mid, ok := qidToMid[result.QueryId]; ok && len(result.Entries) > 0 && len(result.Entries[0].Value) > 0 {
					bIndexRaw[mid] = result.Entries[0].Value
				}
			}
		}
	}

	out := make([]json.RawMessage, 0, len(positions))
	for _, bp := range positions {
		pj, err := protojson.Marshal(bp)
		if err != nil {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(pj, &m); err != nil {
			continue
		}
		// [FIXED] ScaledDebt() can now return an error (ErrScaledDebtOverflow).
		// Same fallback shape as the existing !ok branch below: fall back to
		// raw debtPrincipal rather than fail this one entry in the batch.
		if raw, ok := bIndexRaw[bp.MarketId]; ok {
			if currentDebt, sdErr := ScaledDebt(bp, DecodeUint128(raw)); sdErr == nil {
				m["currentDebt"] = strconv.FormatUint(currentDebt, 10)
			} else {
				m["currentDebt"] = strconv.FormatUint(bp.DebtPrincipal, 10)
			}
		} else {
			m["currentDebt"] = strconv.FormatUint(bp.DebtPrincipal, 10)
		}
		fb, err := json.Marshal(m)
		if err != nil {
			continue
		}
		out = append(out, fb)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleQueryInterestRemainder serves GET /v1/query/interestremainder?marketId=<id>
// Returns the market's current interest_remainder_ray (Market.InterestRemainderRay,
// uint128, RAY-scaled) as a decimal string. Added alongside the interest
// sub-unit rounding-loss fix in interest_accrual.go's Step 7 -- prior to that
// fix, no such field existed at all, so there was nothing to query. A market
// that has never accrued, or has always cleared a full RAY unit exactly on
// every accrual (rare in practice), returns "0", not an error -- an empty or
// zero remainder is the normal steady state, not a fault condition.
func (p *Plugin) handleQueryInterestRemainder(w http.ResponseWriter, r *http.Request) {
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

	var remainder string
	if len(market.InterestRemainderRay) == 0 {
		remainder = "0"
	} else {
		remainder = DecodeUint128(market.InterestRemainderRay).String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"marketId": marketId, "interestRemainderRay": remainder})
}

// handleQueryNasmTier serves GET /v1/query/nasmtier?assetId=<id>
// Returns whether an asset is eligible to back NUSD minting, and at which
// NASM tier, mirroring nasm_tier.go's ResolveNasmTier logic -- the bridge
// from ARCM's {29} tier registry into NASM's own tighter LTV table (NASM
// Consolidated Spec Section 3.1). Deliberately does NOT call
// ResolveNasmTier/GetAssetTier directly: both go through
// c.plugin.StateRead, which QueryState's own doc comment states is tied to
// an in-flight tx/block lifecycle and requires real FSM request context --
// wrong primitive for a detached HTTP handler. Every other handler in this
// file reads state via p.QueryState instead (detached, no Contract
// required, safe from an RPC context per QueryState's own doc comment), so
// this handler replicates GetAssetTier/ResolveNasmTier's small amount of
// logic directly against QueryState rather than constructing a synthetic
// Contract to force the wrong read path to work.
//
// Added to make NASM tier eligibility independently verifiable against
// live devnet state before mint_nusd (which WILL correctly call
// ResolveNasmTier internally, from real transaction context) exists.
//
// found=false is NOT an error condition -- it correctly means "not
// eligible" (no {29} entry, or ARCM Tier 2/3). Reported as 200 OK with
// eligible=false, not a 404, mirroring handleQueryPool/handleQueryTreasury's
// zero-value-is-not-an-error convention.
func (p *Plugin) handleQueryNasmTier(w http.ResponseWriter, r *http.Request) {
	assetId := r.URL.Query().Get("assetId")
	if assetId == "" {
		http.Error(w, `{"error":"missing assetId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateAssetID(assetId); err != nil {
		http.Error(w, `{"error":"invalid assetId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	key := KeyForAssetTier(assetId)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"assetId":  assetId,
			"eligible": false,
			"note":     "asset has no {29} registry entry -- not eligible to back NUSD minting",
		})
		return
	}

	arcmTier := DecodeAssetTierRecord(resp.Results[0].Entries[0].Value)
	nasmParams, tpFound := nasmTierParamsTable[arcmTier]
	if !tpFound {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"assetId":  assetId,
			"eligible": false,
			"note":     "asset is registered at ARCM Tier 2/3 -- not eligible to back NUSD minting (NASM Spec Section 3.1)",
		})
		return
	}

	tierLabel := "N-1"
	if arcmTier == 0 {
		tierLabel = "N-0"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"assetId":   assetId,
		"eligible":  true,
		"nasmTier":  tierLabel,
		"ltvMaxBps": strconv.FormatUint(nasmParams.LTVMaxBps, 10),
		"ltvLiqBps": strconv.FormatUint(nasmParams.LTVLiqBps, 10),
	})
}

// handleQueryNasmVaultPool serves GET /v1/query/nasmvaultpool?vaultId=<id>
// Returns the decoded Pool record for a NASM vault's collateral escrow
// pool (PoolPurposeNasmVault, pool_id.go). Deliberately a separate route
// from handleQueryPool rather than an added "purpose" case there --
// handleQueryPool is conceptually market-scoped throughout (its marketId
// param, its error messages, its own doc comment), and a NASM vault_id is
// not a market_id even though both happen to pass the same length/
// emptiness validation. Mirrors handleQueryPool's own zero-value
// convention: a vault whose pool hasn't been funded yet (or doesn't exist)
// returns {"amount":"0"}, not a 404 -- matches every other pool/reserve
// query's established not-found-is-not-an-error convention in this file.
func (p *Plugin) handleQueryNasmVaultPool(w http.ResponseWriter, r *http.Request) {
	vaultId := r.URL.Query().Get("vaultId")
	if vaultId == "" {
		http.Error(w, `{"error":"missing vaultId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateVaultID(vaultId); err != nil {
		http.Error(w, `{"error":"invalid vaultId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	poolId := KeyForMarketPoolId(vaultId, PoolPurposeNasmVault)
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"vaultId": vaultId, "id": strconv.FormatUint(poolId, 10), "amount": "0", "note": "pool has zero balance or does not yet exist"})
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

// handleQueryNasmVault serves GET /v1/query/nasmvault?vaultId=<id>
// Returns the decoded NasmVault record ({30}) for the given vault_id.
// Mirrors handleQueryMarkets' single-key query shape exactly.
func (p *Plugin) handleQueryNasmVault(w http.ResponseWriter, r *http.Request) {
	vaultId := r.URL.Query().Get("vaultId")
	if vaultId == "" {
		http.Error(w, `{"error":"missing vaultId query param"}`, http.StatusBadRequest)
		return
	}
	if err := ValidateVaultID(vaultId); err != nil {
		http.Error(w, `{"error":"invalid vaultId: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	key := KeyForNasmVault(vaultId)
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
		json.NewEncoder(w).Encode(map[string]string{"error": "vault not found", "vaultId": vaultId})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	vault := &NasmVault{}
	if err := proto.Unmarshal(raw, vault); err != nil {
		http.Error(w, `{"error":"failed to decode vault record: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	vaultJSON, err := protojson.Marshal(vault)
	if err != nil {
		http.Error(w, `{"error":"failed to encode vault json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(vaultJSON)
}

// handleQueryNusdBalance serves GET /v1/query/nusdbalance?address=<hex>
// Returns the decoded NusdBalance record ({35}) for the given address.
// Mirrors handleQueryLenderPosition's address-keyed query shape, minus the
// composite marketId component (NusdBalance is address-keyed alone). A
// holder with no balance yet returns {"amount":"0"}, not a 404, matching
// this file's zero-value-is-not-an-error convention -- an address that
// has simply never minted or received NUSD is not an error state.
func (p *Plugin) handleQueryNusdBalance(w http.ResponseWriter, r *http.Request) {
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

	key := KeyForNusdBalance(address)
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
		json.NewEncoder(w).Encode(map[string]string{"address": addressHex, "amount": "0", "note": "no NUSD balance yet"})
		return
	}

	raw := resp.Results[0].Entries[0].Value
	balance := &NusdBalance{}
	if err := proto.Unmarshal(raw, balance); err != nil {
		http.Error(w, `{"error":"failed to decode NUSD balance record: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	balanceJSON, err := protojson.Marshal(balance)
	if err != nil {
		http.Error(w, `{"error":"failed to encode NUSD balance json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(balanceJSON)
}

// handleQueryNusdSupply serves GET /v1/query/nusdsupply
// Returns the single global {31} NusdSupply record (total_supply, 1e6
// precision). Takes NO query params -- like handleQueryGovernanceParams/
// handleQueryTreasury, this is a single global value, not keyed by
// address or vault_id. Mirrors handleQueryGovernanceParams' zero-value
// convention: before any mint_nusd has ever run, the {31} key does not
// exist yet, and this returns a zero-value NusdSupply (renders as {} per
// proto3 default field-omission, no EmitUnpopulated), not a 404 -- an
// unminted NUSD supply is not an error state.
func (p *Plugin) handleQueryNusdSupply(w http.ResponseWriter, r *http.Request) {
	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: queryId, Key: KeyForNusdSupply()}},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	supply := &NusdSupply{}
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 && len(resp.Results[0].Entries[0].Value) > 0 {
		raw := resp.Results[0].Entries[0].Value
		if err := proto.Unmarshal(raw, supply); err != nil {
			http.Error(w, `{"error":"unmarshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	b, err := protojson.Marshal(supply)
	if err != nil {
		http.Error(w, `{"error":"marshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// handleQueryStabilityFeeIndex serves GET /v1/query/stabilityfeeindex
// Returns the single global {32} StabilityFeeIndex record (sf_index,
// last_accrual_block). Added to make AccrueStabilityFee's real per-block
// effect independently observable on-chain, the same reasoning as
// handleQueryLossFactor's own doc comment on ApplyLossFactor's
// observability. Takes NO query params -- global value, not keyed.
// Mirrors handleQueryGovernanceParams' zero-value convention: before the
// first BeginBlock ever runs AccrueStabilityFee (genesis), {32} does not
// exist yet and this returns a zero-value record ({} per proto3 default
// field-omission), not a 404 -- an unaccrued index is not an error state.
// Note sf_index is returned RAW (base64 bytes, via protojson's default
// bytes encoding) rather than DecodeUint128'd to a decimal string, unlike
// handleQueryLossFactor -- kept simple/raw here since this route's primary
// purpose is confirming last_accrual_block is advancing; a decimal-string
// variant can be added later if a caller specifically needs sf_index's
// numeric value without decoding the base64 client-side.
func (p *Plugin) handleQueryStabilityFeeIndex(w http.ResponseWriter, r *http.Request) {
	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: queryId, Key: KeyForStabilityFeeIndex()}},
	})
	if pErr != nil {
		http.Error(w, `{"error":"query state failed: `+pErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		http.Error(w, `{"error":"state read error: `+resp.Error.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	record := &StabilityFeeIndex{}
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 && len(resp.Results[0].Entries[0].Value) > 0 {
		raw := resp.Results[0].Entries[0].Value
		if err := proto.Unmarshal(raw, record); err != nil {
			http.Error(w, `{"error":"unmarshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	}

	// Additive field: sf_index decoded to a decimal string alongside the
	// raw protojson output, matching handleQueryBorrowerPosition's own
	// additive-field pattern (currentDebt appended to the raw record) --
	// convenient for a human reading this endpoint directly without
	// decoding base64 themselves.
	b, err := protojson.Marshal(record)
	if err != nil {
		http.Error(w, `{"error":"marshal error: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	var responseMap map[string]interface{}
	if err := json.Unmarshal(b, &responseMap); err != nil {
		http.Error(w, `{"error":"failed to build response json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if len(record.SfIndex) > 0 {
		responseMap["sfIndexDecimal"] = DecodeUint128(record.SfIndex).String()
	} else {
		responseMap["sfIndexDecimal"] = RAY.String()
	}
	finalJSON, err := json.Marshal(responseMap)
	if err != nil {
		http.Error(w, `{"error":"failed to encode final response json: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(finalJSON)
}

// handleQueryAllNasmVaults serves GET /v1/query/all-nasm-vaults[?owner=<hex>]
// Returns every NasmVault by range-iterating the {30} vault prefix -- mirrors
// handleQueryAllMarkets's range-scan pattern. Emits a bare JSON array of
// protojson NasmVault objects. Closes the vault-enumeration gap that blocked any
// real mint/burn UI (a user can't burn against a vault they can't see).
//
// [EXTENDED] Originally shipped without server-side owner filtering (client-side
// only). Added an optional owner query param here -- 20-byte hex address,
// server-side filtered -- fully additive: every existing no-param call behaves
// identically to before (returns everything, same as always); only requests that
// now include ?owner= get the filtered behavior. No dedicated owner index exists
// (or is planned) -- this still range-scans the full {30} prefix and filters in
// this handler, matching handleQueryAllBorrowerPositions' own "scan everything,
// filter/derive server-side" precedent rather than introducing a second index
// mint_nusd/burn_nusd would need to additionally maintain.
func (p *Plugin) handleQueryAllNasmVaults(w http.ResponseWriter, r *http.Request) {
	var ownerFilter []byte
	if ownerHex := r.URL.Query().Get("owner"); ownerHex != "" {
		decoded, err := hex.DecodeString(ownerHex)
		if err != nil {
			http.Error(w, `{"error":"invalid owner hex: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		if len(decoded) != 20 {
			http.Error(w, `{"error":"owner must decode to 20 bytes"}`, http.StatusBadRequest)
			return
		}
		ownerFilter = decoded
	}

	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: queryId, Prefix: JoinLenPrefix(PrefixNasmVaults)},
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
	out := make([]json.RawMessage, 0)
	for _, result := range resp.Results {
		if result.QueryId != queryId {
			continue
		}
		for _, entry := range result.Entries {
			if len(entry.Value) == 0 {
				continue
			}
			v := &NasmVault{}
			if err := proto.Unmarshal(entry.Value, v); err != nil {
				continue
			}
			if ownerFilter != nil && !bytes.Equal(v.Owner, ownerFilter) {
				continue
			}
			b, err := protojson.Marshal(v)
			if err != nil {
				continue
			}
			out = append(out, b)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleQueryWaterfallEvents serves GET /v1/query/waterfall-events?limit=<n>&marketId=<id>
// Returns the most recent bad-debt waterfall events (Layer 2/3/4, ARCM
// Section 9.2) from the {42} durable log (waterfall_log.go), most-recent-
// first. Closes the exact gap the Arbor frontend's own Events panel flags
// directly: "Discrete bad-debt waterfall events are emitted on-chain but
// this node exposes no query route for them yet... Awaiting
// /v1/query/waterfall-events (plugin-persisted rolling log, range-scanned
// like all-markets)."
//
// limit: optional, defaults to 50, capped at 500 -- matches this codebase's
// general defensive-default convention (no existing route lets an
// unbounded range scan run, see handleQueryPrices's own scoped-prefix
// precedent). Applied via PluginRangeRead.Limit combined with
// Reverse: true, so the underlying range scan itself only ever walks the
// N most recent entries, not the full log then truncates client-side.
//
// marketId: optional. Because {42}'s key is (block_height, seq), NOT
// market_id (see state_keys.go's KeyForWaterfallEvent doc comment on why --
// chronological range-scan ordering was the design priority), a
// marketId filter cannot be pushed into the range scan's own prefix the
// way handleQueryPrices's assetId filter can. Filtering happens here,
// after decode, over the already-limit-bounded result set -- accepted
// as a real, disclosed limitation rather than silently pretending this
// route supports efficient per-market history the way /v1/query/reservefund
// or /v1/query/lossfactor do.
func (p *Plugin) handleQueryWaterfallEvents(w http.ResponseWriter, r *http.Request) {
	limit := uint64(50)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		parsed, err := strconv.ParseUint(limitParam, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid limit query param"}`, http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if limit == 0 || limit > 500 {
		limit = 500
	}
	marketIdFilter := r.URL.Query().Get("marketId")

	queryId := rand.Uint64()
	resp, pErr := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: queryId, Prefix: JoinLenPrefix(PrefixWaterfallLog), Limit: limit, Reverse: true},
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

	out := make([]json.RawMessage, 0)
	for _, result := range resp.Results {
		if result.QueryId != queryId {
			continue
		}
		for _, entry := range result.Entries {
			if len(entry.Value) == 0 {
				continue
			}
			evt := &WaterfallEvent{}
			if err := proto.Unmarshal(entry.Value, evt); err != nil {
				continue
			}
			if marketIdFilter != "" && evt.MarketId != marketIdFilter {
				continue
			}
			b, err := protojson.Marshal(evt)
			if err != nil {
				continue
			}
			out = append(out, b)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
