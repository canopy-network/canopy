package canoliq

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// rpc.go runs an HTTP server inside the plugin process. The server is
// read-only and serves entirely from the plugin-side Snapshot built inside
// EndBlock — see snapshot.go for why we cannot StateRead from outside an
// FSM-originated lifecycle call.
//
// The route surface here is the singleton + index-driven subset of canoliq
// state. Per-address composite views (account, vesting, redemption-by-id,
// vote-by-voter, buyback-by-id) are not enumerable from a snapshot and are
// deferred to Phase 3 §1.1.

// RPCServer is the HTTP read-only query surface. It is created by
// StartRPCServer and gracefully shut down via Shutdown.
type RPCServer struct {
	plugin *Plugin
	srv    *http.Server
	addr   string
}

// StartRPCServer binds an HTTP listener to addr and returns a running
// *RPCServer. Empty addr disables the server (returns nil, no error).
// The caller is responsible for invoking Shutdown on cancellation.
func StartRPCServer(p *Plugin, addr string) (*RPCServer, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, nil
	}
	s := &RPCServer{plugin: p, addr: addr}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("canoliq: rpc listening on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("canoliq: rpc server exited: %v", err)
		}
	}()
	return s, nil
}

// Shutdown drains in-flight requests with a five-second deadline.
func (s *RPCServer) Shutdown(ctx context.Context) error {
	if s == nil || s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// Addr returns the listener address (useful in tests).
func (s *RPCServer) Addr() string { return s.addr }

func (s *RPCServer) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/health", s.handleHealth)
	mux.HandleFunc("/v1/globals", s.handleGlobals)
	mux.HandleFunc("/v1/params", s.handleParams)
	mux.HandleFunc("/v1/pools", s.handlePools)
	mux.HandleFunc("/v1/proposals", s.handleProposals)
	mux.HandleFunc("/v1/proposal/", s.handleProposal)
	mux.HandleFunc("/v1/spends", s.handleSpends)
	mux.HandleFunc("/v1/spend/", s.handleSpend)
	mux.HandleFunc("/v1/validators", s.handleValidators)
	mux.HandleFunc("/v1/stakers", s.handleStakers)
	// Phase 3 §1.1 per-address routes — fulfilled via the lazy queue
	// drained in EndBlock (see lazy_query.go for the rationale).
	mux.HandleFunc("/v1/account/", s.handleAccount)
	mux.HandleFunc("/v1/vesting/", s.handleVesting)
	mux.HandleFunc("/v1/redemption/", s.handleRedemption)
	mux.HandleFunc("/v1/vote/", s.handleVote)
	mux.HandleFunc("/v1/buyback/", s.handleBuyback)
}

func (s *RPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.plugin.QueryHealth())
}

func (s *RPCServer) handleGlobals(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.plugin.QueryGlobals())
}

func (s *RPCServer) handleParams(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.plugin.QueryParams())
}

func (s *RPCServer) handlePools(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.plugin.QueryPools())
}

func (s *RPCServer) handleProposals(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": s.plugin.QueryProposalIDs()})
}

func (s *RPCServer) handleProposal(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/proposal/")
	id, err := parseUint(tail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	prop := s.plugin.QueryProposal(id)
	if prop == nil {
		writeError(w, http.StatusNotFound, "proposal not found")
		return
	}
	writeJSON(w, http.StatusOK, prop)
}

func (s *RPCServer) handleSpends(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": s.plugin.QuerySpendIDs()})
}

// handleSpend matches both `/v1/spend/{id}` and `/v1/spend/{id}/approvals`.
func (s *RPCServer) handleSpend(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/spend/")
	parts := strings.SplitN(tail, "/", 2)
	id, err := parseUint(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(parts) == 2 && parts[1] == "approvals" {
		writeJSON(w, http.StatusOK, s.plugin.QueryMultisigApprovals(id))
		return
	}
	if len(parts) == 2 && parts[1] != "" {
		writeError(w, http.StatusNotFound, "unknown spend subresource")
		return
	}
	spend := s.plugin.QuerySpend(id)
	if spend == nil {
		writeError(w, http.StatusNotFound, "spend not found")
		return
	}
	writeJSON(w, http.StatusOK, spend)
}

func (s *RPCServer) handleValidators(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.plugin.QueryValidatorRegistry())
}

func (s *RPCServer) handleStakers(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stakers": s.plugin.QueryStakers()})
}

// handleAccount serves /v1/account/{addr}. Fulfilled via the lazy queue —
// blocks the HTTP request until EndBlock drains the query (worst case
// one block ≈ 6s on localnet).
func (s *RPCServer) handleAccount(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/account/")
	addr, err := parseAddress(tail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.plugin.enqueueLazy(r.Context(), &lazyQuery{kind: lazyKindAccount, addr: addr})
	if res.err != nil {
		writeLazyError(w, res.err)
		return
	}
	writeJSON(w, http.StatusOK, res.view)
}

// handleVesting serves /v1/vesting/{addr}. 404 when the address has no
// VestingIndex entry at all (vs. an entry with zero schedules — that's
// 200 with an empty list).
func (s *RPCServer) handleVesting(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/vesting/")
	addr, err := parseAddress(tail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.plugin.enqueueLazy(r.Context(), &lazyQuery{kind: lazyKindVesting, addr: addr})
	if res.err != nil {
		writeLazyError(w, res.err)
		return
	}
	if !res.found {
		writeError(w, http.StatusNotFound, "no vesting schedules for address")
		return
	}
	views := res.views
	if views == nil {
		views = []*VestingView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"address":   hexAddress(addr),
		"schedules": views,
	})
}

// handleRedemption serves /v1/redemption/{addr}/{id}.
func (s *RPCServer) handleRedemption(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/redemption/")
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "expected /v1/redemption/{addr}/{id}")
		return
	}
	addr, err := parseAddress(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := parseUint(parts[1])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.plugin.enqueueLazy(r.Context(), &lazyQuery{kind: lazyKindRedemption, addr: addr, id: id})
	if res.err != nil {
		writeLazyError(w, res.err)
		return
	}
	if !res.found {
		writeError(w, http.StatusNotFound, "redemption not found")
		return
	}
	writeJSON(w, http.StatusOK, res.redemption)
}

// handleVote serves /v1/vote/{proposalId}/{voter}.
func (s *RPCServer) handleVote(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/vote/")
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "expected /v1/vote/{id}/{voter}")
		return
	}
	id, err := parseUint(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	voter, err := parseAddress(parts[1])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.plugin.enqueueLazy(r.Context(), &lazyQuery{kind: lazyKindVote, id: id, voter: voter})
	if res.err != nil {
		writeLazyError(w, res.err)
		return
	}
	if !res.found {
		writeError(w, http.StatusNotFound, "vote not found")
		return
	}
	writeJSON(w, http.StatusOK, res.vote)
}

// handleBuyback serves /v1/buyback/{id}.
func (s *RPCServer) handleBuyback(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/v1/buyback/")
	id, err := parseUint(tail)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.plugin.enqueueLazy(r.Context(), &lazyQuery{kind: lazyKindBuyback, id: id})
	if res.err != nil {
		writeLazyError(w, res.err)
		return
	}
	if !res.found {
		writeError(w, http.StatusNotFound, "buyback order not found")
		return
	}
	writeJSON(w, http.StatusOK, res.buyback)
}

// writeLazyError maps lazy-queue failure modes to HTTP status. Queue
// saturation → 503 (caller should retry), drain timeout → 504 (chain
// stalled), anything else → 500.
func writeLazyError(w http.ResponseWriter, err error) {
	switch err {
	case errLazyQueueFull:
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case errLazyQueryTimeout:
		writeError(w, http.StatusGatewayTimeout, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// --- shared helpers ---

func methodIs(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}
	w.Header().Set("Allow", want)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err := enc.Encode(body); err != nil {
		log.Printf("canoliq: rpc encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// parseAddress accepts a 20-byte address as 0x-prefixed or bare hex.
func parseAddress(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 40 {
		return nil, errStr("address must be 20 bytes (40 hex chars)")
	}
	addr, err := hex.DecodeString(s)
	if err != nil {
		return nil, errStr("address hex decode: " + err.Error())
	}
	return addr, nil
}

// parseUint parses a positive base-10 uint64.
func parseUint(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errStr("expected numeric id")
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, errStr("id parse: " + err.Error())
	}
	return n, nil
}

// errStr is a tiny error wrapper avoiding a dependency on fmt for the
// hot path.
type errStr string

func (e errStr) Error() string { return string(e) }
