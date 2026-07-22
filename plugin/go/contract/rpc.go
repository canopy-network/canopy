package contract

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/*
This file implements custom RPC endpoints for the KnowBet betting protocol, backed by
the detached, read-only QueryState() path.

Endpoints:
  GET /v1/knowbet/question?id=<id>                   — GetQuestion
  GET /v1/knowbet/questions?status=&page=&limit=      — GetQuestionList
  GET /v1/knowbet/bet?question_id=&user=              — GetBet
  GET /v1/knowbet/user-bets?user=&page=&limit=        — GetUserBets
  GET /v1/knowbet/question-bets?question_id=          — GetQuestionBets
  GET /v1/knowbet/user-stats?user=                    — GetUserStats
*/

// =============================================================================
// Response helpers
// =============================================================================

// writeJSON writes a JSON response with proper Content-Type and status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// getQueryParam returns a query parameter value or a default if missing.
func getQueryParam(r *http.Request, key, defaultVal string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return defaultVal
}

// getQueryUint64 parses a uint64 query parameter.
func getQueryUint64(r *http.Request, key string) (uint64, bool) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// getQueryInt parses an int query parameter.
func getQueryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	if n < 1 {
		return defaultVal
	}
	return n
}

// bytesToHex converts address bytes to a hex string with 0x prefix.
func bytesToHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return "0x" + hex.EncodeToString(b)
}

// =============================================================================
// Question status helpers
// =============================================================================

const (
	statusOpen     = "open"
	statusRevealed = "revealed"
	statusSettled  = "settled"
	statusRefunded = "refunded"
	statusExpired  = "expired"
)

// questionStatus returns the current status of a question.
func questionStatus(q *Question) string {
	if q.Refunded {
		return statusRefunded
	}
	if q.Settled {
		return statusSettled
	}
	if q.Revealed {
		return statusRevealed
	}
	if q.CloseTime > 0 && time.Now().Unix() > q.CloseTime {
		return statusExpired
	}
	return statusOpen
}

// questionToResponse converts a Question protobuf to a JSON-safe response map.
func questionToResponse(q *Question) map[string]any {
	status := questionStatus(q)
	resp := map[string]any{
		"id":               q.Id,
		"creatorAddress":   bytesToHex(q.CreatorAddress),
		"questionText":     q.QuestionText,
		"options":          q.Options,
		"answerHash":       q.AnswerHash,
		"stakeAmount":      q.StakeAmount,
		"totalPool":        q.TotalPool,
		"optionPools":      q.OptionPools,
		"createdHeight":    q.CreatedHeight,
		"closeTime":        q.CloseTime,
		"closeHeight":      q.CloseHeight,
		"revealed":         q.Revealed,
		"settled":          q.Settled,
		"refunded":         q.Refunded,
		"status":           status,
		"participantCount": len(q.ParticipantAddresses),
	}
	if q.Revealed {
		resp["revealedAnswer"] = q.RevealedAnswer
	}
	return resp
}

// =============================================================================
// Endpoint: GET /v1/knowbet/question?id=<id>
// =============================================================================

func (p *Plugin) handleGetQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := getQueryUint64(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid 'id' query parameter")
		return
	}

	key := KeyForQuestion(id)
	readResp, err := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: key},
		}})
	if err != nil {
		log.Printf("GetQuestion QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	var questionBytes []byte
	if readResp != nil && len(readResp.Results) > 0 && len(readResp.Results[0].Entries) > 0 {
		questionBytes = readResp.Results[0].Entries[0].Value
	}

	if len(questionBytes) == 0 {
		writeError(w, http.StatusNotFound, "question not found")
		return
	}

	q := new(Question)
	if err := Unmarshal(questionBytes, q); err != nil {
		log.Printf("GetQuestion Unmarshal error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to decode question")
		return
	}

	writeJSON(w, http.StatusOK, questionToResponse(q))
}

// =============================================================================
// Endpoint: GET /v1/knowbet/questions?status=&page=&limit=
// =============================================================================

func (p *Plugin) handleGetQuestionList(w http.ResponseWriter, r *http.Request) {
	statusFilter := getQueryParam(r, "status", "")
	page := getQueryInt(r, "page", 1)
	limit := getQueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Range read over all questions (prefix = questionPrefix raw bytes)
	prefix := JoinLenPrefix(questionPrefix)
	readResp, err := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: prefix, Limit: 10000, Reverse: true},
		}})
	if err != nil {
		log.Printf("GetQuestionList QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	var allQuestions []map[string]any
	if readResp != nil {
		for _, result := range readResp.Results {
			for _, entry := range result.Entries {
				q := new(Question)
				if err := Unmarshal(entry.Value, q); err != nil {
					continue
				}
				// Apply status filter
				if statusFilter != "" && questionStatus(q) != statusFilter {
					continue
				}
				allQuestions = append(allQuestions, questionToResponse(q))
			}
		}
	}

	// Manual pagination
	total := len(allQuestions)
	start := (page - 1) * limit
	if start >= total {
		writeJSON(w, http.StatusOK, map[string]any{
			"questions": []any{},
			"total":     total,
			"page":      page,
			"limit":     limit,
		})
		return
	}
	end := start + limit
	if end > total {
		end = total
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"questions": allQuestions[start:end],
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

// =============================================================================
// Endpoint: GET /v1/knowbet/bet?question_id=<id>&user=<address>
// =============================================================================

func (p *Plugin) handleGetBet(w http.ResponseWriter, r *http.Request) {
	questionID, ok := getQueryUint64(r, "question_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid 'question_id'")
		return
	}
	userHex := getQueryParam(r, "user", "")
	if userHex == "" {
		writeError(w, http.StatusBadRequest, "missing 'user' parameter")
		return
	}
	userAddr, err := hex.DecodeString(strings.TrimPrefix(userHex, "0x"))
	if err != nil || len(userAddr) != 20 {
		writeError(w, http.StatusBadRequest, "invalid user address")
		return
	}

	// Read the bet record
	betKey := KeyForBet(questionID, userAddr)
	readResp, err := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: betKey},
		}})
	if err != nil {
		log.Printf("GetBet QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	var betBytes []byte
	if readResp != nil && len(readResp.Results) > 0 && len(readResp.Results[0].Entries) > 0 {
		betBytes = readResp.Results[0].Entries[0].Value
	}
	if len(betBytes) == 0 {
		writeError(w, http.StatusNotFound, "bet not found")
		return
	}

	bet := new(BetRecord)
	if err := Unmarshal(betBytes, bet); err != nil {
		log.Printf("GetBet Unmarshal error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to decode bet")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"participantAddress": bytesToHex(bet.ParticipantAddress),
		"questionId":         bet.QuestionId,
		"optionIndex":        bet.OptionIndex,
		"amount":             bet.Amount,
	})
}

// =============================================================================
// Endpoint: GET /v1/knowbet/user-bets?user=<address>&page=&limit=
// =============================================================================

func (p *Plugin) handleGetUserBets(w http.ResponseWriter, r *http.Request) {
	userHex := getQueryParam(r, "user", "")
	if userHex == "" {
		writeError(w, http.StatusBadRequest, "missing 'user' parameter")
		return
	}
	userAddr, err := hex.DecodeString(strings.TrimPrefix(userHex, "0x"))
	if err != nil || len(userAddr) != 20 {
		writeError(w, http.StatusBadRequest, "invalid user address")
		return
	}
	page := getQueryInt(r, "page", 1)
	limit := getQueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Scan all bet records and filter by user address
	prefix := JoinLenPrefix(betPrefix)
	readResp, err := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: prefix, Limit: 10000, Reverse: true},
		}})
	if err != nil {
		log.Printf("GetUserBets QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	var userBets []map[string]any
	if readResp != nil {
		for _, result := range readResp.Results {
			for _, entry := range result.Entries {
				bet := new(BetRecord)
				if err := Unmarshal(entry.Value, bet); err != nil {
					continue
				}
				if !bytesEqual(bet.ParticipantAddress, userAddr) {
					continue
				}
				// Fetch question info for context
				questionInfo := fetchQuestionSummary(p, bet.QuestionId)
				userBets = append(userBets, map[string]any{
					"participantAddress": bytesToHex(bet.ParticipantAddress),
					"questionId":         bet.QuestionId,
					"optionIndex":        bet.OptionIndex,
					"amount":             bet.Amount,
					"question":           questionInfo,
				})
			}
		}
	}

	total := len(userBets)
	start := (page - 1) * limit
	if start >= total {
		writeJSON(w, http.StatusOK, map[string]any{
			"bets":  []any{},
			"total": total,
			"page":  page,
			"limit": limit,
		})
		return
	}
	end := start + limit
	if end > total {
		end = total
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bets":  userBets[start:end],
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// =============================================================================
// Endpoint: GET /v1/knowbet/question-bets?question_id=<id>
// =============================================================================

func (p *Plugin) handleGetQuestionBets(w http.ResponseWriter, r *http.Request) {
	id, ok := getQueryUint64(r, "question_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid 'question_id'")
		return
	}

	prefix := KeyForBetPrefix(id)
	readResp, err := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: prefix, Limit: 10000},
		}})
	if err != nil {
		log.Printf("GetQuestionBets QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	var bets []map[string]any
	if readResp != nil {
		for _, result := range readResp.Results {
			for _, entry := range result.Entries {
				bet := new(BetRecord)
				if err := Unmarshal(entry.Value, bet); err != nil {
					continue
				}
				bets = append(bets, map[string]any{
					"participantAddress": bytesToHex(bet.ParticipantAddress),
					"questionId":         bet.QuestionId,
					"optionIndex":        bet.OptionIndex,
					"amount":             bet.Amount,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"questionId": id,
		"bets":       bets,
		"count":      len(bets),
	})
}

// =============================================================================
// Endpoint: GET /v1/knowbet/user-stats?user=<address>
// =============================================================================

func (p *Plugin) handleGetUserStats(w http.ResponseWriter, r *http.Request) {
	userHex := getQueryParam(r, "user", "")
	if userHex == "" {
		writeError(w, http.StatusBadRequest, "missing 'user' parameter")
		return
	}
	userAddr, err := hex.DecodeString(strings.TrimPrefix(userHex, "0x"))
	if err != nil || len(userAddr) != 20 {
		writeError(w, http.StatusBadRequest, "invalid user address")
		return
	}

	// Scan all questions
	qPrefix := JoinLenPrefix(questionPrefix)
	qResp, err := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: qPrefix, Limit: 10000},
		}})
	if err != nil {
		log.Printf("GetUserStats QueryState error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	// Scan all bets
	bPrefix := JoinLenPrefix(betPrefix)
	bResp, err := p.QueryState(0, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: rand.Uint64(), Prefix: bPrefix, Limit: 10000},
		}})
	if err != nil {
		log.Printf("GetUserStats QueryState bets error: %v", err)
		writeError(w, http.StatusInternalServerError, "query state failed")
		return
	}

	// Collect user's questions (as creator)
	questionsCreated := 0
	questionsSettled := 0
	questionsOpen := 0
	settledQuestions := make(map[uint64]*Question)

	if qResp != nil {
		for _, result := range qResp.Results {
			for _, entry := range result.Entries {
				q := new(Question)
				if err := Unmarshal(entry.Value, q); err != nil {
					continue
				}
				if !bytesEqual(q.CreatorAddress, userAddr) {
					continue
				}
				questionsCreated++
				if q.Settled {
					questionsSettled++
					settledQuestions[q.Id] = q
				}
				if !q.Revealed && !q.Settled && !q.Refunded {
					questionsOpen++
				}
			}
		}
	}

	// Collect user's bets (as participant)
	betsPlaced := 0
	betsWon := 0
	totalBetAmount := uint64(0)
	totalWonAmount := uint64(0)
	userQuestionIDs := make(map[uint64]bool)

	if bResp != nil {
		for _, result := range bResp.Results {
			for _, entry := range result.Entries {
				bet := new(BetRecord)
				if err := Unmarshal(entry.Value, bet); err != nil {
					continue
				}
				if !bytesEqual(bet.ParticipantAddress, userAddr) {
					continue
				}
				betsPlaced++
				totalBetAmount += bet.Amount
				userQuestionIDs[bet.QuestionId] = true
			}
		}
	}

	// Calculate win stats by checking settled questions
	// Re-read settled questions that the user participated in
	for qid := range userQuestionIDs {
		if _, ok := settledQuestions[qid]; ok {
			// This question was created BY the user (not as participant)
			// Skip for participant win calculations
			continue
		}
		// Check if this is a relevant settled question the user participated in
		qKey := KeyForQuestion(qid)
		qRead, qErr := p.QueryState(0, &PluginStateReadRequest{
			Keys: []*PluginKeyRead{
				{QueryId: rand.Uint64(), Key: qKey},
			}})
		if qErr != nil {
			continue
		}
		var qBytes []byte
		if qRead != nil && len(qRead.Results) > 0 && len(qRead.Results[0].Entries) > 0 {
			qBytes = qRead.Results[0].Entries[0].Value
		}
		if len(qBytes) == 0 {
			continue
		}
		q := new(Question)
		if err := Unmarshal(qBytes, q); err != nil || !q.Settled {
			continue
		}

		// Find winning option
		winIdx := -1
		answerTrimmed := strings.TrimSpace(strings.ToLower(q.RevealedAnswer))
		for i, opt := range q.Options {
			if strings.TrimSpace(strings.ToLower(opt)) == answerTrimmed {
				winIdx = i
				break
			}
		}
		if winIdx < 0 {
			continue
		}

		// Read user's bet on this question
		betKey := KeyForBet(qid, userAddr)
		betRead, betErr := p.QueryState(0, &PluginStateReadRequest{
			Keys: []*PluginKeyRead{
				{QueryId: rand.Uint64(), Key: betKey},
			}})
		if betErr != nil {
			continue
		}
		var betBytes []byte
		if betRead != nil && len(betRead.Results) > 0 && len(betRead.Results[0].Entries) > 0 {
			betBytes = betRead.Results[0].Entries[0].Value
		}
		if len(betBytes) == 0 {
			continue
		}
		userBet := new(BetRecord)
		if err := Unmarshal(betBytes, userBet); err != nil {
			continue
		}

		if int(userBet.OptionIndex) == winIdx {
			betsWon++
			// Estimate winnings: proportional share of 80% pool
			if q.TotalPool > 0 {
				// Find total winning amount for this question
				totalWinAmt := uint64(0)
				bPrefix := KeyForBetPrefix(qid)
				bAll, _ := p.QueryState(0, &PluginStateReadRequest{
					Ranges: []*PluginRangeRead{
						{QueryId: rand.Uint64(), Prefix: bPrefix, Limit: 10000},
					}})
				if bAll != nil {
					for _, br := range bAll.Results {
						for _, be := range br.Entries {
							br2 := new(BetRecord)
							if err := Unmarshal(be.Value, br2); err != nil {
								continue
							}
							if int(br2.OptionIndex) == winIdx {
								totalWinAmt += br2.Amount
							}
						}
					}
				}
				winnerShare := q.TotalPool * 80 / 100
				if totalWinAmt > 0 {
					estimatedWin := uint64(float64(userBet.Amount) / float64(totalWinAmt) * float64(winnerShare))
					totalWonAmount += estimatedWin + userBet.Amount
				}
			}
		}
	}

	winRate := 0.0
	if betsPlaced > 0 {
		winRate = float64(betsWon) / float64(betsPlaced) * 100
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"userAddress":      bytesToHex(userAddr),
		"questionsCreated": questionsCreated,
		"questionsSettled": questionsSettled,
		"questionsOpen":    questionsOpen,
		"betsPlaced":       betsPlaced,
		"betsWon":          betsWon,
		"winRate":          round2(winRate),
		"totalBetAmount":   totalBetAmount,
		"totalWonAmount":   totalWonAmount,
		"netProfit":        int64(totalWonAmount) - int64(totalBetAmount),
	})
}

// =============================================================================
// Helpers
// =============================================================================

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// round2 rounds a float64 to 2 decimal places.
func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}

// fetchQuestionSummary reads a question and returns a summary map (nil on error).
func fetchQuestionSummary(p *Plugin, questionID uint64) map[string]any {
	key := KeyForQuestion(questionID)
	resp, err := p.QueryState(0, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: rand.Uint64(), Key: key},
		}})
	if err != nil {
		return nil
	}
	var qBytes []byte
	if resp != nil && len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		qBytes = resp.Results[0].Entries[0].Value
	}
	if len(qBytes) == 0 {
		return nil
	}
	q := new(Question)
	if err := Unmarshal(qBytes, q); err != nil {
		return nil
	}
	return map[string]any{
		"id":           q.Id,
		"questionText": q.QuestionText,
		"status":       questionStatus(q),
		"totalPool":    q.TotalPool,
		"options":      q.Options,
		"optionPools":  q.OptionPools,
		"revealed":     q.Revealed,
		"settled":      q.Settled,
		"refunded":     q.Refunded,
	}
}

// =============================================================================
// StartRPCServer - Override the skeleton to register KnowBet endpoints
// =============================================================================

// StartRPCServer() launches the plugin's own HTTP server with KnowBet RPC endpoints.
func (p *Plugin) StartRPCServer() {
	addr := p.config.RPCAddress
	if addr == "" {
		log.Println("KnowBet RPC server disabled (no rpcAddress configured)")
		return
	}

	mux := http.NewServeMux()

	// Register KnowBet query endpoints
	mux.HandleFunc("/v1/knowbet/question", p.handleGetQuestion)
	mux.HandleFunc("/v1/knowbet/questions", p.handleGetQuestionList)
	mux.HandleFunc("/v1/knowbet/bet", p.handleGetBet)
	mux.HandleFunc("/v1/knowbet/user-bets", p.handleGetUserBets)
	mux.HandleFunc("/v1/knowbet/question-bets", p.handleGetQuestionBets)
	mux.HandleFunc("/v1/knowbet/user-stats", p.handleGetUserStats)

	// Health check
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "knowbet"})
	})

	log.Printf("KnowBet RPC server listening on %s", addr)
	log.Printf("  GET /v1/knowbet/question      — get question by id")
	log.Printf("  GET /v1/knowbet/questions      — list questions (filter by status, page, limit)")
	log.Printf("  GET /v1/knowbet/bet            — get bet (question_id, user)")
	log.Printf("  GET /v1/knowbet/user-bets      — list user bets (user, page, limit)")
	log.Printf("  GET /v1/knowbet/question-bets  — list question bets (question_id)")
	log.Printf("  GET /v1/knowbet/user-stats     — get user stats (user)")
	log.Printf("  GET /v1/health                 — health check")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("KnowBet RPC server error: %v", err)
	}
}
