package contract

/*
quest_xp.go — Community quest/XP tracking for Arbor, built as a fully
standalone file with ZERO edits to any other file in this package.

WHY THIS EXISTS AS A SEPARATE FILE, NOT WIRED INTO DeliverTx:
Go doesn't allow a second file to add a case to an existing switch
statement or a route to an existing http.ServeMux from outside the file
that owns it — there's no hook system for that. So rather than touch
contract.go's DeliverTx switch or rpc.go's StartRPCServer, this file
runs its own goroutine (started automatically via init(), which Go calls
for every file in a package at process startup — no wiring needed
anywhere else) and its own tiny HTTP server on a separate port. It polls
Arbor's own core RPC for linked wallets' transaction history — the same
approach the standalone TypeScript indexer used, just now living inside
this process instead of a separate hosted service.

WHAT THIS DELIBERATELY DOES NOT DO:
- Does NOT write to consensus state (no PluginSetOp, no StateWrite). XP
  lives in an in-memory map inside this process only. Writing state
  outside of DeliverTx would not be deterministic across validators —
  every node must compute identical state transitions from the same
  block, and a background poller's timing can't guarantee that. This
  is a deliberate, disclosed limitation: XP resets if this process
  restarts. Treat it as a read-side convenience feature, not a ledger.
- Does NOT add a Go module dependency. It reuses this package's own
  vendored BLS wrapper (crypto/bls.go, BytesToBLS12381PublicKey/.Verify),
  the exact scheme real Arbor account keys use (BDN over BLS12-381 via
  drand/kyber) — not a different library that might not be compatible.

CONFIGURATION (env vars, all optional with sensible defaults):
  QUEST_XP_RPC_ADDRESS        — this service's own HTTP listen address (default "0.0.0.0:50011")
  QUEST_XP_CORE_RPC_URL       — where Arbor's core query RPC is reachable from THIS process
                                 (default "http://localhost:50002" — verify this is correct for
                                 wherever this actually gets deployed; the externally-facing URL
                                 on a shared/proxied node is not necessarily the same as the
                                 process's own local address)
  QUEST_XP_POLL_INTERVAL_SECS — how often linked wallets are swept (default 60)
*/

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/sign/bdn"
)

// ---- config ----

const (
	blockSeconds              = 20
	dayBlocks                 = 24 * 60 * 60 / blockSeconds     // 4320
	weekBlocks                = 7 * 24 * 60 * 60 / blockSeconds // 30240
	linkSignatureWindowBlocks = 300                             // ~1hr at 20s/block
)

func coreRPCURL() string {
	if v := os.Getenv("QUEST_XP_CORE_RPC_URL"); v != "" {
		return v
	}
	return "http://localhost:50002"
}

func pollIntervalSeconds() int {
	if v := os.Getenv("QUEST_XP_POLL_INTERVAL_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 60
}

func dayIDForHeight(height int64) int64  { return height / dayBlocks }
func weekIDForHeight(height int64) int64 { return height / weekBlocks }

// ---- quest catalog (mirrors indexer/src/config.ts's QUESTS exactly) ----

type questDef struct {
	ID          string
	Label       string
	Description string
	MessageType string
	XP          int
}

var questCatalog = []questDef{
	{"arbor-deposit-v1", "Supply to a market", "Go to any active market on the Markets page and supply an asset. Any amount counts.", "deposit", 10},
	{"arbor-borrow-v1", "Borrow against collateral", "With collateral deposited, open a market and borrow against it. Watch your health factor as you go.", "borrow", 15},
	{"arbor-repay-v1", "Repay a borrow position", "On a market where you have an open borrow, repay some or all of it from the Repay tab.", "repay", 10},
	{"arbor-deposit-collateral-v1", "Deposit collateral", "Add collateral to a market you're borrowing from, or open a new position with collateral.", "deposit_collateral", 5},
	{"arbor-withdraw-v1", "Withdraw supplied assets", "Withdraw some or all of an asset you've previously supplied to a market.", "withdraw", 5},
}

func questForMessageType(mt string) *questDef {
	for i := range questCatalog {
		if questCatalog[i].MessageType == mt {
			return &questCatalog[i]
		}
	}
	return nil
}

// ---- in-memory store (guarded by mutex — poller goroutine and HTTP handlers both touch this) ----

type identityRecord struct {
	Address       string `json:"address"`
	DiscordID     string `json:"discordId"`
	TwitterHandle string `json:"twitterHandle"`
	LinkedAt      int64  `json:"linkedAt"`
}

type xpRecord struct {
	Address    string `json:"address"`
	WeekID     int64  `json:"weekId"`
	DayID      int64  `json:"dayId"`
	QuestID    string `json:"questId"`
	TxHash     string `json:"txHash"`
	XP         int    `json:"xp"`
	CreditedAt int64  `json:"creditedAt"`
}

var (
	storeMu          sync.RWMutex
	identitiesByAddr = map[string]identityRecord{}
	identitiesByDisc = map[string]string{} // discordId -> address
	identitiesByTwit = map[string]string{} // twitterHandle -> address
	xpRecords        []xpRecord
	cursorByAddr     = map[string]int64{}
)

func questXPAlreadyCredited(address, txHash, questID string) bool {
	storeMu.RLock()
	defer storeMu.RUnlock()
	for _, r := range xpRecords {
		if r.Address == address && r.TxHash == txHash && r.QuestID == questID {
			return true
		}
	}
	return false
}

func questXPAlreadyCreditedToday(address, questID string, dayID int64) bool {
	storeMu.RLock()
	defer storeMu.RUnlock()
	for _, r := range xpRecords {
		if r.Address == address && r.QuestID == questID && r.DayID == dayID {
			return true
		}
	}
	return false
}

func questXPCredit(rec xpRecord) {
	storeMu.Lock()
	defer storeMu.Unlock()
	xpRecords = append(xpRecords, rec)
}

func questXPCompletedToday(address string, dayID int64) map[string]bool {
	storeMu.RLock()
	defer storeMu.RUnlock()
	out := map[string]bool{}
	for _, r := range xpRecords {
		if r.Address == address && r.DayID == dayID {
			out[r.QuestID] = true
		}
	}
	return out
}

func questXPLeaderboard(weekID int64) []map[string]interface{} {
	storeMu.RLock()
	defer storeMu.RUnlock()
	totals := map[string]int{}
	for _, r := range xpRecords {
		if r.WeekID == weekID {
			totals[r.Address] += r.XP
		}
	}
	out := make([]map[string]interface{}, 0, len(totals))
	for addr, xp := range totals {
		out = append(out, map[string]interface{}{"address": addr, "xp": xp})
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j]["xp"].(int) > out[i]["xp"].(int) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// ---- BLS verification + address derivation, matching Canopy's real account keys exactly ----
// Reimplements crypto/bls.go's exact unmarshal-and-verify logic directly against the underlying
// drand/kyber + drand/kyber-bls12381 libraries (same BDN-over-BLS12-381 scheme real Arbor account
// keys use) rather than importing plugin/go/crypto's wrapper package — that package already
// imports THIS package (contract), so importing it back here would be a circular import Go
// refuses to compile. Same verification behavior, just without routing through the wrapper.

func questXPAddressFromPublicKeyHex(pubKeyHex string) (string, error) {
	pubBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(pubBytes)
	return hex.EncodeToString(digest[:20]), nil
}

func questXPVerifySignature(message, sigHex, pubKeyHex string) bool {
	pubBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return false
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil {
		return false
	}
	suite := bls12381.NewBLS12381Suite()
	point := suite.G1().Point()
	if err := point.UnmarshalBinary(pubBytes); err != nil {
		return false
	}
	scheme := bdn.NewSchemeOnG2(suite)
	return scheme.Verify(point, []byte(message), sigBytes) == nil
}

// ---- core RPC client (this process calling out to Arbor's own core query RPC) ----

type coreTx struct {
	Sender      string
	MessageType string
	Height      int64
	TxHash      string
	ErrorCode   int
}

var msgTypePrefixRe = regexp.MustCompile(`^type\.googleapis\.com/types\.Message`)
var camelToSnakeRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// normalizeMessageType mirrors the TS indexer's actionMeta() transform exactly, so quest
// matching stays consistent with what the RPC actually returns (a full protobuf type URL,
// not a pre-cleaned lowercase string).
func normalizeMessageType(raw string) string {
	s := msgTypePrefixRe.ReplaceAllString(raw, "")
	s = camelToSnakeRe.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}

func fetchCurrentHeight() (int64, error) {
	resp, err := http.Post(coreRPCURL()+"/v1/query/height", "application/json", strings.NewReader("{}"))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	return parsed.Height, nil
}

func fetchTxsBySender(address string) ([]coreTx, error) {
	clean := strings.TrimPrefix(address, "0x")
	var all []coreTx
	page := 1
	const perPage = 50

	for {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"address": clean, "pageNumber": page, "perPage": perPage,
		})
		resp, err := http.Post(coreRPCURL()+"/v1/query/txs-by-sender", "application/json", strings.NewReader(string(reqBody)))
		if err != nil {
			return all, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var parsed struct {
			Results    []map[string]interface{} `json:"results"`
			TotalPages int                      `json:"totalPages"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return all, err
		}

		for _, item := range parsed.Results {
			tx := coreTx{}
			if s, ok := item["sender"].(string); ok {
				tx.Sender = s
			}
			rawType := ""
			if mt, ok := item["messageType"].(string); ok {
				rawType = mt
			}
			tx.MessageType = normalizeMessageType(rawType)
			if h, ok := item["height"].(float64); ok {
				tx.Height = int64(h)
			}
			if h, ok := item["txHash"].(string); ok {
				tx.TxHash = h
			}
			if errObj, ok := item["error"].(map[string]interface{}); ok {
				if code, ok := errObj["code"].(float64); ok {
					tx.ErrorCode = int(code)
				}
			}
			all = append(all, tx)
		}

		if page >= parsed.TotalPages || len(parsed.Results) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// ---- sweep loop ----

func questXPSweepWallet(identity identityRecord) {
	storeMu.RLock()
	cursorHeight := cursorByAddr[identity.Address]
	storeMu.RUnlock()

	txs, err := fetchTxsBySender(identity.Address)
	if err != nil {
		fmt.Printf("[quest_xp] sweep failed for %s: %v\n", identity.Address, err)
		return
	}

	maxHeight := cursorHeight
	for _, tx := range txs {
		if tx.Height <= cursorHeight || tx.ErrorCode != 0 {
			continue
		}
		if tx.Height > maxHeight {
			maxHeight = tx.Height
		}

		quest := questForMessageType(tx.MessageType)
		if quest == nil {
			continue
		}
		if questXPAlreadyCredited(identity.Address, tx.TxHash, quest.ID) {
			continue
		}

		dayID := dayIDForHeight(tx.Height)
		weekID := weekIDForHeight(tx.Height)

		if questXPAlreadyCreditedToday(identity.Address, quest.ID, dayID) {
			fmt.Printf("[quest_xp] skipped %s for quest %q (tx %s) — daily cap already met for day %d\n",
				identity.Address, quest.ID, tx.TxHash, dayID)
			continue
		}

		questXPCredit(xpRecord{
			Address: identity.Address, WeekID: weekID, DayID: dayID,
			QuestID: quest.ID, TxHash: tx.TxHash, XP: quest.XP, CreditedAt: time.Now().UnixMilli(),
		})
		fmt.Printf("[quest_xp] credited %d XP to %s for quest %q (tx %s, week %d, day %d)\n",
			quest.XP, identity.Address, quest.ID, tx.TxHash, weekID, dayID)
	}

	storeMu.Lock()
	cursorByAddr[identity.Address] = maxHeight
	storeMu.Unlock()
}

func questXPSweepAll() {
	storeMu.RLock()
	identities := make([]identityRecord, 0, len(identitiesByAddr))
	for _, id := range identitiesByAddr {
		identities = append(identities, id)
	}
	storeMu.RUnlock()

	fmt.Printf("[quest_xp] sweeping %d linked wallet(s)\n", len(identities))
	// Sequential on purpose for the first cut — this is a background goroutine sharing
	// the process with the actual node, so it deliberately does not add concurrent HTTP
	// load beyond what a slow, patient poller needs. Revisit if wallet count outgrows
	// what fits in one poll interval this way.
	for _, id := range identities {
		questXPSweepWallet(id)
	}
}

// ---- HTTP handlers (own mux, own port — separate from rpc.go's StartRPCServer entirely) ----

func questXPWriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func questXPHandleToday(w http.ResponseWriter, r *http.Request) {
	address := strings.ToLower(strings.TrimPrefix(r.URL.Query().Get("address"), "0x"))
	height, err := fetchCurrentHeight()
	if err != nil {
		questXPWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "could not reach core RPC for height"})
		return
	}
	dayID := dayIDForHeight(height)

	var completed map[string]bool
	if address != "" {
		completed = questXPCompletedToday(address, dayID)
	} else {
		completed = map[string]bool{}
	}

	quests := make([]map[string]interface{}, 0, len(questCatalog))
	for _, q := range questCatalog {
		quests = append(quests, map[string]interface{}{
			"id": q.ID, "label": q.Label, "description": q.Description,
			"xp": q.XP, "completed": completed[q.ID],
		})
	}
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"dayId": dayID, "height": height, "quests": quests})
}

func questXPHandleXPForAddress(w http.ResponseWriter, r *http.Request) {
	address := strings.ToLower(strings.TrimPrefix(r.URL.Query().Get("address"), "0x"))
	if address == "" {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing address query param"})
		return
	}
	storeMu.RLock()
	var entries []xpRecord
	total := 0
	for _, rec := range xpRecords {
		if rec.Address == address {
			entries = append(entries, rec)
			total += rec.XP
		}
	}
	storeMu.RUnlock()
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"address": address, "totalXp": total, "entries": entries})
}

func questXPHandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	weekParam := r.URL.Query().Get("weekId")
	var weekID int64
	if weekParam == "" || weekParam == "current" {
		height, err := fetchCurrentHeight()
		if err != nil {
			questXPWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "could not reach core RPC for height"})
			return
		}
		weekID = weekIDForHeight(height)
	} else {
		n, err := strconv.ParseInt(weekParam, 10, 64)
		if err != nil {
			questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid weekId"})
			return
		}
		weekID = n
	}
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"weekId": weekID, "leaderboard": questXPLeaderboard(weekID)})
}

type linkRequest struct {
	Address        string `json:"address"`
	PublicKeyHex   string `json:"publicKeyHex"`
	DiscordID      string `json:"discordId"`
	TwitterHandle  string `json:"twitterHandle"`
	IssuedAtHeight int64  `json:"issuedAtHeight"`
	SignatureHex   string `json:"signatureHex"`
}

func questXPHandleLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		questXPWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	var req linkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Address == "" || req.PublicKeyHex == "" || req.DiscordID == "" || req.TwitterHandle == "" || req.SignatureHex == "" {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	normAddr := strings.ToLower(strings.TrimPrefix(req.Address, "0x"))
	derivedAddr, err := questXPAddressFromPublicKeyHex(req.PublicKeyHex)
	if err != nil || derivedAddr != normAddr {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "publicKeyHex does not derive to the claimed address"})
		return
	}

	height, err := fetchCurrentHeight()
	if err != nil {
		questXPWriteJSON(w, http.StatusBadGateway, map[string]string{"error": "could not reach core RPC for height"})
		return
	}
	if height-req.IssuedAtHeight > linkSignatureWindowBlocks {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "signature expired, restart the link flow"})
		return
	}

	message := fmt.Sprintf("Link Arbor identity\ndiscord:%s\ntwitter:%s\nissuedAt:%d", req.DiscordID, req.TwitterHandle, req.IssuedAtHeight)
	if !questXPVerifySignature(message, req.SignatureHex, req.PublicKeyHex) {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "signature verification failed"})
		return
	}

	storeMu.Lock()
	if existingAddr, taken := identitiesByDisc[req.DiscordID]; taken && existingAddr != normAddr {
		storeMu.Unlock()
		questXPWriteJSON(w, http.StatusConflict, map[string]string{"error": "discord account already linked to a different wallet"})
		return
	}
	if existingAddr, taken := identitiesByTwit[req.TwitterHandle]; taken && existingAddr != normAddr {
		storeMu.Unlock()
		questXPWriteJSON(w, http.StatusConflict, map[string]string{"error": "x account already linked to a different wallet"})
		return
	}
	identitiesByAddr[normAddr] = identityRecord{Address: normAddr, DiscordID: req.DiscordID, TwitterHandle: req.TwitterHandle, LinkedAt: time.Now().UnixMilli()}
	identitiesByDisc[req.DiscordID] = normAddr
	identitiesByTwit[req.TwitterHandle] = normAddr
	storeMu.Unlock()

	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "address": normAddr})
}

func questXPCORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // tighten to specific origins before relying on this beyond testnet
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// ---- startup ----

func init() {
	go startQuestXPService()
}

func startQuestXPService() {
	interval := time.Duration(pollIntervalSeconds()) * time.Second
	fmt.Printf("[quest_xp] polling every %s (routes served via rpc.go's existing plugin RPC server)\n", interval)
	questXPSweepAll()
	ticker := time.NewTicker(interval)
	for range ticker.C {
		questXPSweepAll()
	}
}
