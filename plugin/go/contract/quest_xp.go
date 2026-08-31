package contract

/*
quest_xp.go — Community quest/XP tracking for Arbor. Self-contained
except for one deliberate 4-line addition to rpc.go (route registration
only, into the plugin's existing RPC mux — see below).

CURRENT DESIGN (as of commit d7285d8c — do not revert to what's described
below this line without reading the history first):
- Routes are served through rpc.go's existing http.ServeMux, registered
  there via questXPCORSMiddleware-wrapped handlers. An earlier version of
  this file ran its own HTTP server on a separate port (:50011); that was
  UNREACHABLE on the shared grad node because the reverse proxy only maps
  three fixed prefixes (/rpc/, /adminrpc/, /plugin/) with no generic
  "any port gets a path" mechanism. Do not reintroduce a standalone
  server — new quest routes go through rpc.go's mux like the four here.
- Sweeping still runs as its own background goroutine, started
  automatically via init() (Go calls init() for every file in a package
  at process startup — no wiring needed anywhere else). It polls Arbor's
  own core RPC for linked wallets' transaction history, the same
  approach the standalone TypeScript indexer used, just now living
  inside this process instead of a separate hosted service.
- Signature verification uses PLAIN BLS12-381 (drand/kyber/sign/bls),
  NOT the BDN variant (drand/kyber/sign/bdn). This file originally used
  BDN because plugin/go/crypto/bls.go (an existing wrapper elsewhere in
  this repo) uses BDN and looked authoritative — but that wrapper is
  never actually on the real transaction-verification path; Canopy core
  verifies transactions independently using plain BLS. Using BDN here
  made every single /v1/link call fail 100% of the time, silently, until
  traced and fixed. If you touch signature verification in this file
  again, do NOT reintroduce BDN. This is also why verification is
  reimplemented directly against drand/kyber + drand/kyber-bls12381
  below rather than importing plugin/go/crypto's wrapper package — that
  package already imports this one (contract), so importing it back here
  would be a circular import Go refuses to compile, and it uses the
  wrong (BDN) scheme regardless.

WHAT THIS DELIBERATELY DOES NOT DO:
- Does NOT write to consensus state (no PluginSetOp, no StateWrite). XP
  lives in an in-memory map inside this process only. Writing state
  outside of DeliverTx would not be deterministic across validators —
  every node must compute identical state transitions from the same
  block, and a background poller's timing can't guarantee that. This
  is a deliberate, disclosed limitation: XP resets if this process
  restarts. Treat it as a read-side convenience feature, not a ledger.

CONFIGURATION (env vars, all optional with sensible defaults):
  QUEST_XP_CORE_RPC_URL       — where Arbor's core query RPC is reachable from THIS process
                                 (default "http://localhost:50002" — verify this is correct for
                                 wherever this actually gets deployed; the externally-facing URL
                                 on a shared/proxied node is not necessarily the same as the
                                 process's own local address)
  QUEST_XP_POLL_INTERVAL_SECS — how often linked wallets are swept (default 60)
  QUEST_ADMIN_TOKEN           — shared secret for the /v1/admin/questxp/* endpoints (manual
                                 record cleanup). Unset by default, which means those routes
                                 reject every request (fail closed) — set this before you need
                                 to use them, there's no way to use them without it.
  QUEST_XP_STATE_FILE         — path to a JSON file this process reads on startup and writes
                                 to after every mutation, so identities/XP survive a process
                                 restart (default "./quest_xp_state.json", relative to wherever
                                 the process's working directory is — set an absolute path in
                                 production so a restart from a different cwd doesn't silently
                                 start fresh). This does NOT change the deliberate limitation
                                 above: state still isn't consensus/StateWrite-backed, still
                                 isn't replicated across validators, and a multi-validator setup
                                 would have each node keep its own independent file. It only
                                 protects against the single-process-restart case (redeploys,
                                 crashes, node reboots) that was previously an unqualified full
                                 data loss.
*/

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	bls12381 "github.com/drand/kyber-bls12381"
	"github.com/drand/kyber/sign/bls"
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
	EvmAddress    string `json:"evmAddress,omitempty"` // optional — see questXPHandleLink for validation
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

// ---- persistence ----
//
// Snapshot-on-write to a single JSON file. Not a WAL, not atomic-per-record —
// every save writes the entire current state. That's fine at this data size
// (identities + XP records for a quest program, not a general ledger) and it
// keeps the format trivially inspectable/editable by hand if something needs
// manual correction. Writes go to a temp file then get renamed into place, so
// a crash mid-write can't leave a half-written, unparseable state file behind.

type questXPSnapshot struct {
	IdentitiesByAddr map[string]identityRecord `json:"identitiesByAddr"`
	IdentitiesByDisc map[string]string         `json:"identitiesByDisc"`
	IdentitiesByTwit map[string]string         `json:"identitiesByTwit"`
	XPRecords        []xpRecord                `json:"xpRecords"`
	CursorByAddr     map[string]int64          `json:"cursorByAddr"`
	SavedAt          int64                     `json:"savedAt"`
}

func questXPStateFilePath() string {
	if v := os.Getenv("QUEST_XP_STATE_FILE"); v != "" {
		return v
	}
	return "./quest_xp_state.json"
}

// questXPSaveState snapshots current in-memory state to disk. Called after
// every mutation (link, credit, admin delete) rather than on a timer, so a
// crash between saves never loses more than the single mutation in flight.
// Best-effort: a save failure is logged, not fatal — a stalled disk write
// shouldn't take down quest tracking, and the in-memory state (source of
// truth while the process is up) is untouched either way.
func questXPSaveState() {
	storeMu.RLock()
	snap := questXPSnapshot{
		IdentitiesByAddr: make(map[string]identityRecord, len(identitiesByAddr)),
		IdentitiesByDisc: make(map[string]string, len(identitiesByDisc)),
		IdentitiesByTwit: make(map[string]string, len(identitiesByTwit)),
		XPRecords:        make([]xpRecord, len(xpRecords)),
		CursorByAddr:     make(map[string]int64, len(cursorByAddr)),
		SavedAt:          time.Now().UnixMilli(),
	}
	for k, v := range identitiesByAddr {
		snap.IdentitiesByAddr[k] = v
	}
	for k, v := range identitiesByDisc {
		snap.IdentitiesByDisc[k] = v
	}
	for k, v := range identitiesByTwit {
		snap.IdentitiesByTwit[k] = v
	}
	copy(snap.XPRecords, xpRecords)
	for k, v := range cursorByAddr {
		snap.CursorByAddr[k] = v
	}
	storeMu.RUnlock()

	path := questXPStateFilePath()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		fmt.Printf("[quest_xp] state save failed (marshal): %v\n", err)
		return
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".quest_xp_state.*.tmp")
	if err != nil {
		fmt.Printf("[quest_xp] state save failed (tempfile in %s): %v\n", dir, err)
		return
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		fmt.Printf("[quest_xp] state save failed (write): %v\n", err)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		fmt.Printf("[quest_xp] state save failed (close): %v\n", err)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		fmt.Printf("[quest_xp] state save failed (rename into %s): %v\n", path, err)
		return
	}
}

// questXPLoadState runs once at startup, before the sweep loop starts. A
// missing file is the expected first-run case (nothing to load, not an
// error). A present-but-corrupt file is treated cautiously: it's logged
// loudly and left in place untouched (never overwritten by the loader) so a
// human can inspect it, and the process starts with empty in-memory state
// rather than guessing at partial recovery.
func questXPLoadState() {
	path := questXPStateFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[quest_xp] no state file at %s, starting fresh\n", path)
			return
		}
		fmt.Printf("[quest_xp] state load failed (read %s): %v — starting with empty state\n", path, err)
		return
	}
	var snap questXPSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		fmt.Printf("[quest_xp] state load failed (parse %s): %v — file left untouched, starting with empty state\n", path, err)
		return
	}

	storeMu.Lock()
	defer storeMu.Unlock()
	if snap.IdentitiesByAddr != nil {
		identitiesByAddr = snap.IdentitiesByAddr
	}
	if snap.IdentitiesByDisc != nil {
		identitiesByDisc = snap.IdentitiesByDisc
	}
	if snap.IdentitiesByTwit != nil {
		identitiesByTwit = snap.IdentitiesByTwit
	}
	if snap.XPRecords != nil {
		xpRecords = snap.XPRecords
	}
	if snap.CursorByAddr != nil {
		cursorByAddr = snap.CursorByAddr
	}
	fmt.Printf("[quest_xp] state loaded from %s (saved at %d): %d identities, %d xp records\n",
		path, snap.SavedAt, len(identitiesByAddr), len(xpRecords))
}

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
	xpRecords = append(xpRecords, rec)
	storeMu.Unlock()
	questXPSaveState()
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
		entry := map[string]interface{}{"address": addr, "xp": xp}
		// identitiesByAddr is keyed by the same normalized (lowercase, no 0x prefix)
		// address form used everywhere else in this file — see questXPHandleLink.
		if id, ok := identitiesByAddr[addr]; ok && id.EvmAddress != "" {
			entry["evmAddress"] = id.EvmAddress
		}
		out = append(out, entry)
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
	scheme := bls.NewSchemeOnG2(suite)
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

// questXPHandleIdentity answers "is this address already linked, and to what" so the
// frontend can restore state on page load / wallet reconnect instead of re-running the
// OAuth+sign flow every time. Read-only, no signature required — this only returns what
// was already committed via a signed /v1/link call; it doesn't grant anything new.
func questXPHandleIdentity(w http.ResponseWriter, r *http.Request) {
	address := strings.ToLower(strings.TrimPrefix(r.URL.Query().Get("address"), "0x"))
	if address == "" {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing address query param"})
		return
	}
	storeMu.RLock()
	rec, linked := identitiesByAddr[address]
	storeMu.RUnlock()
	if !linked {
		questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"linked": false})
		return
	}
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{
		"linked":        true,
		"address":       rec.Address,
		"discordId":     rec.DiscordID,
		"twitterHandle": rec.TwitterHandle,
		"evmAddress":    rec.EvmAddress,
		"linkedAt":      rec.LinkedAt,
	})
}

// ---- admin: manual cleanup of individual xpRecords ----
// Protected by a shared-secret header, not by anything cryptographic — this is a
// destructive operation (removes real XP data) so it's intentionally NOT open like the
// rest of this file's read endpoints. Fails closed: if QUEST_ADMIN_TOKEN isn't set in
// the environment, every request here is rejected, rather than defaulting to open.
//
// Deliberately NOT a "clear all of today for this address" bulk operation: the daily
// cap (questXPAlreadyCreditedToday) guarantees at most one record can ever exist per
// (address, questId, dayId) — which means a stale backfilled record and a record from
// something the user genuinely just did can land on the exact same dayId and be
// indistinguishable by day alone. CreditedAt (wall-clock, not chain-derived) is the
// only reliable way to tell them apart, so list first, confirm which ones are actually
// stale by their CreditedAt clustering, then delete by exact (address, txHash, questId).

func questXPAdminAuthorized(r *http.Request) bool {
	token := os.Getenv("QUEST_ADMIN_TOKEN")
	return token != "" && r.Header.Get("X-Admin-Token") == token
}

func questXPHandleAdminListRecords(w http.ResponseWriter, r *http.Request) {
	if !questXPAdminAuthorized(r) {
		questXPWriteJSON(w, http.StatusForbidden, map[string]string{"error": "missing or invalid X-Admin-Token"})
		return
	}
	address := strings.ToLower(strings.TrimPrefix(r.URL.Query().Get("address"), "0x"))
	if address == "" {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing address query param"})
		return
	}
	storeMu.RLock()
	var out []xpRecord
	for _, rec := range xpRecords {
		if rec.Address == address {
			out = append(out, rec)
		}
	}
	storeMu.RUnlock()
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"address": address, "count": len(out), "records": out})
}

type adminDeleteRecordRequest struct {
	Address string `json:"address"`
	TxHash  string `json:"txHash"`
	QuestID string `json:"questId"`
}

func questXPHandleAdminDeleteRecord(w http.ResponseWriter, r *http.Request) {
	if !questXPAdminAuthorized(r) {
		questXPWriteJSON(w, http.StatusForbidden, map[string]string{"error": "missing or invalid X-Admin-Token"})
		return
	}
	if r.Method != http.MethodPost {
		questXPWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	var req adminDeleteRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	address := strings.ToLower(strings.TrimPrefix(req.Address, "0x"))
	if address == "" || req.TxHash == "" || req.QuestID == "" {
		questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "address, txHash, and questId are all required — this deletes by exact match only, never a bulk clear"})
		return
	}
	storeMu.Lock()
	kept := xpRecords[:0]
	removed := 0
	for _, rec := range xpRecords {
		if rec.Address == address && rec.TxHash == req.TxHash && rec.QuestID == req.QuestID {
			removed++
			continue
		}
		kept = append(kept, rec)
	}
	xpRecords = kept
	storeMu.Unlock()
	if removed > 0 {
		questXPSaveState()
	}
	questXPWriteJSON(w, http.StatusOK, map[string]interface{}{"removed": removed})
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
	EvmAddress     string `json:"evmAddress"` // optional — see NOTE below on the mandatory/optional call
	IssuedAtHeight int64  `json:"issuedAtHeight"`
	SignatureHex   string `json:"signatureHex"`
}

// evmAddressRe matches a well-formed 0x-prefixed, 20-byte (40 hex char) EVM address.
// Deliberately does NOT enforce EIP-55 checksum casing — the reward-export step (Snag)
// almost certainly lowercases/normalizes anyway, and rejecting valid-but-uncapitalized
// addresses from users who paste rather than connect would be user-hostile for no
// real safety benefit.
var evmAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// NOTE — mandatory vs. optional (open product question per handoff, not yet decided
// by Adam/Eric as of this commit): implemented here as OPTIONAL. Reasoning: Adam's
// Discord message describes crediting rewards "on a discretionary basis" from a
// curated list, not a fully automated 1:1 export — so a user without MetaMask losing
// XP-earning entirely seemed like the wrong default. If EVM linking is later decided
// to be mandatory, change the empty-string check below to require it alongside
// Discord/Twitter, and update the frontend button's readyToLink condition accordingly.

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
	normEvmAddr := ""
	if req.EvmAddress != "" {
		if !evmAddressRe.MatchString(req.EvmAddress) {
			questXPWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "evmAddress is not a well-formed 0x-prefixed 20-byte hex address"})
			return
		}
		normEvmAddr = strings.ToLower(req.EvmAddress)
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

	// evmAddress is folded into the signed message (not just the JSON body) so it's
	// cryptographically bound to this wallet's signature, same as discord/twitter —
	// otherwise a client could submit any evmAddress value with a signature that never
	// actually attested to it. "none" is an explicit literal (not empty string) so a
	// link-without-EVM request can't be replayed/edited into a link-with-EVM request
	// against the same signature by just appending a field.
	evmForMessage := "none"
	if normEvmAddr != "" {
		evmForMessage = normEvmAddr
	}
	message := fmt.Sprintf("Link Arbor identity\ndiscord:%s\ntwitter:%s\nevm:%s\nissuedAt:%d", req.DiscordID, req.TwitterHandle, evmForMessage, req.IssuedAtHeight)
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
	identitiesByAddr[normAddr] = identityRecord{Address: normAddr, DiscordID: req.DiscordID, TwitterHandle: req.TwitterHandle, EvmAddress: normEvmAddr, LinkedAt: time.Now().UnixMilli()}
	identitiesByDisc[req.DiscordID] = normAddr
	identitiesByTwit[req.TwitterHandle] = normAddr
	// Seed the sweep cursor to the height at link time. Without this, cursorByAddr
	// defaults to Go's zero-value (0) for a never-before-seen address, so the first
	// sweep scans the address's ENTIRE on-chain history from block 0 — crediting any
	// prior activity (from before the user ever linked identity) as XP, sometimes
	// landing in "today"'s dayID purely by coincidence of recency, not because the
	// user actually did anything today. Only activity from this point forward should
	// ever earn XP — this was flagged as an open product question in earlier design
	// discussion; this resolves it in favor of "linking starts the clock, it doesn't
	// retroactively reward past activity."
	if _, alreadySwept := cursorByAddr[normAddr]; !alreadySwept {
		cursorByAddr[normAddr] = height
	}
	storeMu.Unlock()
	questXPSaveState()

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
	questXPLoadState()
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
