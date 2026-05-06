package canoliq

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/canopy-network/go-plugin/contract"
)

// rpc.go runs an HTTP server inside the plugin process. The server is
// read-only — every route ultimately calls a Query* helper from query.go,
// which in turn issues plugin-side StateRead calls. The server shares the
// long-lived *Plugin (and therefore the same FSM unix socket) with the FSM
// dispatch loop. Per-request *Canoliq contexts are minted with a fresh
// rand.Uint64() fsmId so concurrent HTTP requests do not collide in
// Plugin.pending.

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
	mux.HandleFunc("/v1/vote/", s.handleVote)
	mux.HandleFunc("/v1/buyback/", s.handleBuyback)
	mux.HandleFunc("/v1/spends", s.handleSpends)
	mux.HandleFunc("/v1/spend/", s.handleSpend)
	mux.HandleFunc("/v1/validators", s.handleValidators)
	mux.HandleFunc("/v1/account/", s.handleAccount)
	mux.HandleFunc("/v1/redemption/", s.handleRedemption)
	mux.HandleFunc("/v1/vesting/", s.handleVesting)
}

// queryContext mints a per-request *Canoliq sharing the long-lived plugin.
// The fsmId is randomized so concurrent HTTP requests do not stomp on each
// other's response channels in Plugin.pending.
func (s *RPCServer) queryContext() *Canoliq {
	return &Canoliq{
		Config:    s.plugin.config,
		FSMConfig: s.plugin.fsmConfig,
		plugin:    s.plugin,
		fsmId:     rand.Uint64(),
	}
}

// --- handlers ---

func (s *RPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	view, perr := s.queryContext().QueryHealth()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *RPCServer) handleGlobals(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	g, perr := s.queryContext().QueryGlobals()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *RPCServer) handleParams(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	p, perr := s.queryContext().QueryParams()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *RPCServer) handlePools(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	view, perr := s.queryContext().QueryPools()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *RPCServer) handleProposals(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	ids, perr := s.queryContext().QueryProposalIndex()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if ids == nil {
		ids = []uint64{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
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
	prop, perr := s.queryContext().QueryProposal(id)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if prop == nil {
		writeError(w, http.StatusNotFound, "proposal not found")
		return
	}
	writeJSON(w, http.StatusOK, prop)
}

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
	addr, err := parseAddress(parts[1])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	v, perr := s.queryContext().QueryVote(id, addr)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if v == nil {
		writeError(w, http.StatusNotFound, "vote not found")
		return
	}
	writeJSON(w, http.StatusOK, v)
}

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
	order, perr := s.queryContext().QueryBuybackOrder(id)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if order == nil {
		writeError(w, http.StatusNotFound, "buyback order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *RPCServer) handleSpends(w http.ResponseWriter, r *http.Request) {
	if !methodIs(w, r, http.MethodGet) {
		return
	}
	ids, perr := s.queryContext().QuerySpendIndex()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if ids == nil {
		ids = []uint64{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
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
		view, perr := s.queryContext().QueryMultisigApprovals(id)
		if perr != nil {
			writePluginError(w, perr)
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if len(parts) == 2 && parts[1] != "" {
		writeError(w, http.StatusNotFound, "unknown spend subresource")
		return
	}
	spend, perr := s.queryContext().QueryTreasurySpend(id)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
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
	reg, perr := s.queryContext().QueryValidatorRegistry()
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

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
	view, perr := s.queryContext().QueryAccount(addr)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

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
	red, perr := s.queryContext().QueryRedemption(addr, id)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if red == nil {
		writeError(w, http.StatusNotFound, "redemption not found")
		return
	}
	writeJSON(w, http.StatusOK, red)
}

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
	views, perr := s.queryContext().QueryVesting(addr)
	if perr != nil {
		writePluginError(w, perr)
		return
	}
	if views == nil {
		views = []*VestingView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"address": hexAddress(addr), "schedules": views})
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

// writePluginError maps a *contract.PluginError to an HTTP response.
// Address-shape failures map to 400; everything else is treated as 500.
func writePluginError(w http.ResponseWriter, e *contract.PluginError) {
	if e == nil {
		writeError(w, http.StatusInternalServerError, "unknown error")
		return
	}
	// Canoliq-side address validation errors carry code 100 (`ErrInvalidAddress`).
	if e.Code == 100 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s: %s", e.Module, e.Msg))
		return
	}
	writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", e.Module, e.Msg))
}

// parseAddress accepts a 20-byte address as 0x-prefixed or bare hex.
func parseAddress(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 40 {
		return nil, fmt.Errorf("address must be 20 bytes (40 hex chars)")
	}
	addr, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("address hex decode: %v", err)
	}
	return addr, nil
}

// parseUint parses a positive base-10 uint64.
func parseUint(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("expected numeric id")
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("id parse: %v", err)
	}
	return n, nil
}
