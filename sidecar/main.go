package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── Config ────────────────────────────────────────────────────────

func getRPC() string {
	if v := os.Getenv("PRAXIS_RPC"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:50002"
}

func getPort() string {
	if v := os.Getenv("PRAXIS_SIDECAR_PORT"); v != "" {
		return v
	}
	return "8085"
}

func getKnownCreators() []string {
	if v := os.Getenv("PRAXIS_KNOWN_CREATORS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"e7c7dad131a03f7ea0cc09a637ad096eb3495f77"}
}

const (
	pollInterval       = 6 * time.Second
	finalizationBounty = uint64(50_000_000)
)

// ── Domain types ──────────────────────────────────────────────────

type MarketStatus int

const (
	StatusOpen      MarketStatus = 0
	StatusProposed  MarketStatus = 4
	StatusDisputed  MarketStatus = 5
	StatusFinalized MarketStatus = 6
	StatusExpired   MarketStatus = 99
)

type Market struct {
	MarketID    string       `json:"marketId"`
	Question    string       `json:"question"`
	Creator     string       `json:"creator"`
	B0          uint64       `json:"b0"`
	LmsrSeed    uint64       `json:"lmsrSeed"`
	QYes        uint64       `json:"qYes"`
	QNo         uint64       `json:"qNo"`
	BEff        uint64       `json:"bEff"`
	ExpiryTime  uint64       `json:"expiryTime"`
	Nonce       uint64       `json:"nonce"`
	Status      MarketStatus `json:"status"`
	StatusLabel string       `json:"statusLabel"`
	TotalPool   uint64       `json:"totalPool"`
	YesPct      int          `json:"yesPct"`
	NoPct       int          `json:"noPct"`
	CreatedAt   uint64       `json:"createdAt"`
	TxHash      string       `json:"txHash"`
}

func (m *Market) computeDerived() {
	total := m.QYes + m.QNo
	if total > 0 {
		m.YesPct = int(m.QYes * 100 / total)
		m.NoPct  = 100 - m.YesPct
	} else {
		m.YesPct = 50
		m.NoPct  = 50
	}
	m.TotalPool = total
	switch m.Status {
	case StatusOpen:
		m.StatusLabel = "open"
	case StatusProposed:
		m.StatusLabel = "proposed"
	case StatusDisputed:
		m.StatusLabel = "disputed"
	case StatusFinalized:
		m.StatusLabel = "finalized"
	case StatusExpired:
		m.StatusLabel = "expired"
	default:
		m.StatusLabel = "unknown"
	}
}

// ── Store ─────────────────────────────────────────────────────────

type Store struct {
	mu      sync.RWMutex
	markets map[string]*Market
	height  uint64
}

func newStore() *Store {
	return &Store{markets: make(map[string]*Market)}
}

func (s *Store) getMarkets() []*Market {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Market, 0, len(s.markets))
	for _, m := range s.markets {
		cp := *m
		out = append(out, &cp)
	}
	return out
}

func (s *Store) getMarket(id string) (*Market, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.markets[id]
	if !ok {
		return nil, false
	}
	cp := *m
	return &cp, true
}

func (s *Store) getHeight() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.height
}

func (s *Store) setHeight(h uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.height = h
}

func (s *Store) applyUpdate(fresh *Market) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.markets[fresh.MarketID]
	if !ok {
		s.markets[fresh.MarketID] = fresh
		return true, false
	}
	if existing.QYes != fresh.QYes || existing.QNo != fresh.QNo ||
		existing.Status != fresh.Status {
		s.markets[fresh.MarketID] = fresh
		return false, true
	}
	return false, false
}

// ── Proto varint decoder ──────────────────────────────────────────

func decodeVarint(b []byte, pos int) (uint64, int) {
	var val uint64
	var shift uint
	for pos < len(b) {
		byt := b[pos]
		pos++
		val |= uint64(byt&0x7f) << shift
		shift += 7
		if byt&0x80 == 0 {
			break
		}
	}
	return val, pos
}

func decodeMsgBytes(hexStr string) (map[int]interface{}, error) {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	fields := make(map[int]interface{})
	pos := 0
	for pos < len(b) {
		tag, p := decodeVarint(b, pos)
		if p == pos {
			break
		}
		pos = p
		fieldNum := int(tag >> 3)
		wireType := tag & 0x7

		switch wireType {
		case 0:
			val, p2 := decodeVarint(b, pos)
			pos = p2
			fields[fieldNum] = val
		case 2:
			length, p2 := decodeVarint(b, pos)
			pos = p2
			end := pos + int(length)
			if end > len(b) {
				end = len(b)
			}
			chunk := make([]byte, end-pos)
			copy(chunk, b[pos:end])
			pos = end
			fields[fieldNum] = chunk
		case 1:
			pos += 8
		case 5:
			pos += 4
		default:
			pos = len(b)
		}
	}
	return fields, nil
}

func getBytes(fields map[int]interface{}, n int) []byte {
	v, ok := fields[n]
	if !ok {
		return nil
	}
	b, ok := v.([]byte)
	if !ok {
		return nil
	}
	return b
}

func getUint(fields map[int]interface{}, n int) uint64 {
	v, ok := fields[n]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case uint64:
		return val
	case float64:
		return uint64(val)
	case string:
		u, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return 0
		}
		return u
	default:
		return 0
	}
}

func base64ToHex(b64 string) string {
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(decoded) != 20 {
		return ""
	}
	return hex.EncodeToString(decoded)
}

func getString(fields map[int]interface{}, n int) string {
	b := getBytes(fields, n)
	if b == nil {
		return ""
	}
	return string(b)
}

// ── Market ID derivation ──────────────────────────────────────────

func deriveMarketID(creatorHex string, nonce uint64) string {
	creator, err := hex.DecodeString(creatorHex)
	if err != nil || len(creator) != 20 {
		return ""
	}
	nb := make([]byte, 8)
	binary.BigEndian.PutUint64(nb, nonce)
	input := append(creator, nb...)
	h := sha256.Sum256(input)
	return hex.EncodeToString(h[:20])
}

func lmsrCost(qYes, qNo, bEff uint64) float64 {
	if bEff == 0 {
		return 0
	}
	b := float64(bEff)
	y := float64(qYes)
	n := float64(qNo)
	ay := y / b
	an := n / b
	var lse float64
	if ay >= an {
		lse = ay + math.Log1p(math.Exp(an-ay))
	} else {
		lse = an + math.Log1p(math.Exp(ay-an))
	}
	return b * lse
}

// ── RPC client ────────────────────────────────────────────────────

type rpcClient struct {
	base string
	http *http.Client
}

func newRPCClient(base string) *rpcClient {
	return &rpcClient{
		base: base,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *rpcClient) post(path string, body interface{}) (map[string]interface{}, error) {
	b, _ := json.Marshal(body)
	resp, err := c.http.Post(c.base+path, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rpcClient) getHeight() (uint64, error) {
	d, err := c.post("/v1/query/height", map[string]interface{}{})
	if err != nil {
		return 0, err
	}
	if h, ok := d["height"].(float64); ok {
		return uint64(h), nil
	}
	return 0, fmt.Errorf("height not found")
}

func (c *rpcClient) getTxsBySender(addr string, perPage int) ([]interface{}, error) {
	d, err := c.post("/v1/query/txs-by-sender", map[string]interface{}{"address": addr, "perPage": perPage})
	if err != nil {
		return nil, err
	}
	results, _ := d["results"].([]interface{})
	return results, nil
}

func (c *rpcClient) getFailedTxs(addr string, perPage int) ([]interface{}, error) {
	d, err := c.post("/v1/query/failed-txs", map[string]interface{}{"address": addr, "perPage": perPage})
	if err != nil {
		return nil, err
	}
	results, _ := d["results"].([]interface{})
	return results, nil
}

// ── Indexer ───────────────────────────────────────────────────────

type Indexer struct {
	rpc      *rpcClient
	store    *Store
	creators []string
	failedMu sync.Mutex
	failed   map[string]bool
}

func newIndexer(rpc *rpcClient, store *Store, creators []string) *Indexer {
	return &Indexer{
		rpc:      rpc,
		store:    store,
		creators: creators,
		failed:   make(map[string]bool),
	}
}

func (idx *Indexer) loadFailedTxs(addr string) {
	results, _ := idx.rpc.getFailedTxs(addr, 500)
	idx.failedMu.Lock()
	defer idx.failedMu.Unlock()
	for _, r := range results {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if hash, ok := m["txHash"].(string); ok {
			idx.failed[hash] = true
		}
	}
}

func (idx *Indexer) isFailed(txHash string) bool {
	idx.failedMu.Lock()
	defer idx.failedMu.Unlock()
	return idx.failed[txHash]
}

func (idx *Indexer) poll() {
	height, err := idx.rpc.getHeight()
	if err != nil {
		log.Printf("[indexer] RPC unreachable: %v", err)
		return
	}

	idx.store.setHeight(height)

	for _, creator := range idx.creators {
		idx.loadFailedTxs(creator)
	}

	freshMarkets := make(map[string]*Market)

	for _, creator := range idx.creators {
		txs, err := idx.rpc.getTxsBySender(creator, 500)
		if err != nil {
			continue
		}
		for _, raw := range txs {
			tx, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			txHash, _ := tx["txHash"].(string)
			if idx.isFailed(txHash) {
				continue
			}
			t, _ := tx["transaction"].(map[string]interface{})
			if t == nil {
				t = tx
			}
			msgType, _ := t["type"].(string)
			if msgType == "" {
				msgType, _ = t["messageType"].(string)
			}
			if msgType != "create_market" {
				continue
			}

			var creatorAddr string
			var b0, expiry, nonce, createdAt uint64
			var question string

			if msgBytes, ok := t["msgBytes"].(string); ok && msgBytes != "" {
				fields, err := decodeMsgBytes(msgBytes)
				if err != nil {
					continue
				}
				cb := getBytes(fields, 1)
				if len(cb) == 20 {
					creatorAddr = hex.EncodeToString(cb)
				}
				b0 = getUint(fields, 2)
				expiry = getUint(fields, 3)
				nonce = getUint(fields, 4)
				question = getString(fields, 5)
			} else if msg, ok := t["msg"].(map[string]interface{}); ok {
				question, _ = msg["question"].(string)
				if v, ok := msg["creatorAddress"].(string); ok {
					creatorAddr = base64ToHex(v)
				}
				if v, ok := msg["b0"].(float64); ok {
					b0 = uint64(v)
				} else if v, ok := msg["b0"].(string); ok {
					u, _ := strconv.ParseUint(v, 10, 64)
					b0 = u
				}
				if v, ok := msg["expiryTime"].(float64); ok {
					expiry = uint64(v)
				} else if v, ok := msg["expiryTime"].(string); ok {
					u, _ := strconv.ParseUint(v, 10, 64)
					expiry = u
				}
				if v, ok := msg["nonce"].(float64); ok {
					nonce = uint64(v)
				} else if v, ok := msg["nonce"].(string); ok {
					u, _ := strconv.ParseUint(v, 10, 64)
					nonce = u
				}
			}
			if creatorAddr == "" || b0 == 0 {
				continue
			}
			if v, ok := tx["height"].(float64); ok {
				createdAt = uint64(v)
			}

			marketID := deriveMarketID(creatorAddr, nonce)
			if marketID == "" {
				continue
			}
			lmsrSeed := b0
			if b0 > finalizationBounty {
				lmsrSeed = b0 - finalizationBounty
			}
			halfSeed := lmsrSeed / 2

			m := &Market{
				MarketID:   marketID,
				Question:   question,
				Creator:    creatorAddr,
				B0:         b0,
				LmsrSeed:   lmsrSeed,
				QYes:       halfSeed,
				QNo:        halfSeed,
				BEff:       lmsrSeed,
				ExpiryTime: expiry,
				Nonce:      nonce,
				Status:     StatusOpen,
				CreatedAt:  createdAt,
				TxHash:     txHash,
			}
			if height > expiry && expiry > 0 {
				m.Status = StatusExpired
			}
			freshMarkets[marketID] = m
		}
	}

	// Replay submit_prediction
	for _, creator := range idx.creators {
		txs, err := idx.rpc.getTxsBySender(creator, 500)
		if err != nil {
			continue
		}
		for _, raw := range txs {
			tx, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			txHash, _ := tx["txHash"].(string)
			if idx.isFailed(txHash) {
				continue
			}
			t, _ := tx["transaction"].(map[string]interface{})
			if t == nil {
				t = tx
			}
			msgType, _ := t["type"].(string)
			if msgType == "" {
				msgType, _ = t["messageType"].(string)
			}
			if msgType != "submit_prediction" {
				continue
			}

			var marketID string
			var outcome bool
			var shares uint64

			if msgBytes, ok := t["msgBytes"].(string); ok && msgBytes != "" {
				fields, err := decodeMsgBytes(msgBytes)
				if err != nil {
					continue
				}
				midBytes := getBytes(fields, 1)
				if len(midBytes) == 20 {
					marketID = hex.EncodeToString(midBytes)
				}
				outVal := getUint(fields, 3)
				outcome = outVal == 1
				shares = getUint(fields, 4)
			} else if msg, ok := t["msg"].(map[string]interface{}); ok {
				if v, ok := msg["marketId"].(string); ok {
					marketID = base64ToHex(v)
				}
				if v, ok := msg["outcome"].(bool); ok {
					outcome = v
				}
				if v, ok := msg["shares"].(float64); ok {
					shares = uint64(v)
				} else if v, ok := msg["shares"].(string); ok {
					u, _ := strconv.ParseUint(v, 10, 64)
					shares = u
				}
			}
			if marketID == "" || shares == 0 {
				continue
			}
			m, ok := freshMarkets[marketID]
			if !ok {
				continue
			}
			if outcome {
				m.QYes += shares
			} else {
				m.QNo += shares
			}
		}
	}

	// Process status-changing TXs
	for _, creator := range idx.creators {
		txs, _ := idx.rpc.getTxsBySender(creator, 500)
		for _, raw := range txs {
			tx, _ := raw.(map[string]interface{})
			txHash, _ := tx["txHash"].(string)
			if idx.isFailed(txHash) {
				continue
			}
			t, _ := tx["transaction"].(map[string]interface{})
			if t == nil {
				t = tx
			}
			msgType, _ := t["type"].(string)
			if msgType == "" {
				msgType, _ = t["messageType"].(string)
			}
			var marketID string
			if msgBytes, ok := t["msgBytes"].(string); ok {
				fields, _ := decodeMsgBytes(msgBytes)
				midBytes := getBytes(fields, 1)
				if len(midBytes) == 20 {
					marketID = hex.EncodeToString(midBytes)
				}
			} else if msg, ok := t["msg"].(map[string]interface{}); ok {
				if v, ok := msg["marketId"].(string); ok {
					marketID = base64ToHex(v)
				}
			}
			if marketID == "" {
				continue
			}
			m, ok := freshMarkets[marketID]
			if !ok {
				continue
			}
			switch msgType {
			case "propose_outcome":
				m.Status = StatusProposed
			case "file_dispute":
				m.Status = StatusDisputed
			case "finalize_market":
				m.Status = StatusFinalized
			}
		}
	}

	// Apply updates to store
	for _, m := range freshMarkets {
		m.computeDerived()
		idx.store.applyUpdate(m)
	}
}

// ── HTTP handlers ─────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(v)
}

func handleHealth(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]interface{}{
			"status":  "ok",
			"height":  store.getHeight(),
			"markets": len(store.getMarkets()),
		})
	}
}

func handleMarkets(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]interface{}{
			"height":  store.getHeight(),
			"markets": store.getMarkets(),
			"count":   len(store.getMarkets()),
		})
	}
}

func handleMarket(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing id param"}`, http.StatusBadRequest)
			return
		}
		m, ok := store.getMarket(id)
		if !ok {
			http.Error(w, `{"error":"market not found"}`, http.StatusNotFound)
			return
		}
		jsonResp(w, m)
	}
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	rpcBase  := getRPC()
	port     := getPort()
	creators := getKnownCreators()

	log.Printf("[praxis-sidecar] starting on :%s", port)
	log.Printf("[praxis-sidecar] RPC: %s", rpcBase)
	log.Printf("[praxis-sidecar] creators: %v", creators)

	rpc   := newRPCClient(rpcBase)
	store := newStore()
	idx   := newIndexer(rpc, store, creators)

	// Initial poll
	idx.poll()

	// Periodic poll
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			idx.poll()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/praxis/health",  handleHealth(store))
	mux.HandleFunc("/v1/praxis/markets", handleMarkets(store))
	mux.HandleFunc("/v1/praxis/market",  handleMarket(store))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})

	log.Printf("[praxis-sidecar] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
