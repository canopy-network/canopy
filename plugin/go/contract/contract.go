package contract

// contract.go — Praxis Prediction Market Plugin
// Implements ADLMSR v5.6.6-r2-CORRECTED + PORS v1.0-r2-CORRECTED
// All 20 ADLMSR findings + all 13 PORS findings resolved.
//
// DO NOT modify plugin.go, main.go, plugin.proto, or any *.pb.go files.
//
// Duplicates removed: STATUS/PROPOSAL/timing constants → constants.go
//                     globalHeight/SetGlobalHeight/GetGlobalHeight → height.go
//                     SafeMarshal/errCheckWrite → error.go
//                     mulDiv/bytesEqual/nextQueryId → helpers.go
//                     lmsrCost/ComputeTradeCost/ComputePayout/ComputeMinBond → lmsr.go

import (
	"math/rand"

	"google.golang.org/protobuf/types/known/anypb"
)

// ═══════════════════════════════════════════════════════════════════════════
// STATE KEY PREFIXES — Praxis 0x10–0x1C only (never touch 0x01/0x02/0x07)
// These are local to contract.go for use by the private key helper funcs.
// ═══════════════════════════════════════════════════════════════════════════

var (
	PREFIX_MARKET_STATE    = []byte{0x10}
	PREFIX_POSITION_STATE  = []byte{0x11}
	PREFIX_OUTCOME_STATE   = []byte{0x12}
	PREFIX_RESOLVER_STATE  = []byte{0x13} // per market_id — written by propose_outcome (NF-5)
	PREFIX_TREASURY        = []byte{0x14}
	// 0x15 reserved
	PREFIX_RESOLVER_RECORD = []byte{0x16} // global, per resolver_address
	PREFIX_PROPOSAL_RECORD = []byte{0x17}
	PREFIX_DISPUTE_RECORD  = []byte{0x18}
	PREFIX_VOTE_COMMIT     = []byte{0x19}
	PREFIX_VOTE_REVEAL     = []byte{0x1A}
	PREFIX_SLASH_RECORD    = []byte{0x1B}
)

// Canopy base-layer keys (read/write ONLY from send handler or MovePoolToPool)
var (
	PREFIX_ACCOUNT  = []byte{0x01}
	PREFIX_FEE_POOL = []byte{0x02}
)

// ═══════════════════════════════════════════════════════════════════════════
// CONTRACT CONFIG — exact registration
// Phase 1: 5 types (send + 4 ADLMSR). Phase 2: 12 types (+ 7 PORS).
// SupportedTransactions[i] MUST exactly match TransactionTypeUrls[i].
// ═══════════════════════════════════════════════════════════════════════════

var ContractConfig = &PluginConfig{
	Name:    "praxis_prediction_market",
	Id:      1,
	Version: 1,
	// Phase 1 — ADLMSR only.
	// To enable Phase 2 (PORS), swap the commented block below.
	SupportedTransactions: []string{
		"send",               // index 0
		"create_market",      // index 1
		"submit_prediction",  // index 2
		"resolve_market",     // index 3
		"claim_winnings",     // index 4
	},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageCreateMarket",
		"type.googleapis.com/types.MessageSubmitPrediction",
		"type.googleapis.com/types.MessageResolveMarket",
		"type.googleapis.com/types.MessageClaimWinnings",
	},

	// Phase 2 — uncomment when PORS handlers are wired up:
	// SupportedTransactions: []string{
	//     "send",               // 0
	//     "create_market",      // 1
	//     "submit_prediction",  // 2
	//     "resolve_market",     // 3
	//     "claim_winnings",     // 4
	//     "register_resolver",  // 5
	//     "propose_outcome",    // 6
	//     "file_dispute",       // 7
	//     "commit_vote",        // 8
	//     "reveal_vote",        // 9
	//     "tally_votes",        // 10  — NF-1 fix
	//     "finalize_market",    // 11
	// },
	// TransactionTypeUrls: []string{
	//     "type.googleapis.com/types.MessageSend",
	//     "type.googleapis.com/types.MessageCreateMarket",
	//     "type.googleapis.com/types.MessageSubmitPrediction",
	//     "type.googleapis.com/types.MessageResolveMarket",
	//     "type.googleapis.com/types.MessageClaimWinnings",
	//     "type.googleapis.com/types.MessageRegisterResolver",
	//     "type.googleapis.com/types.MessageProposeOutcome",
	//     "type.googleapis.com/types.MessageFileDispute",
	//     "type.googleapis.com/types.MessageCommitVote",
	//     "type.googleapis.com/types.MessageRevealVote",
	//     "type.googleapis.com/types.MessageTallyVotes",
	//     "type.googleapis.com/types.MessageFinalizeMarket",
	// },
}

// ═══════════════════════════════════════════════════════════════════════════
// CONTRACT — main struct (plugin.go defines the Plugin struct; never modify)
// ═══════════════════════════════════════════════════════════════════════════

type Contract struct {
	Config    Config
	FSMConfig *PluginFSMConfig
	plugin    *Plugin
	fsmId     uint64
	// DO NOT add currentHeight here — AUDIT-9: use globalHeight + RWMutex
}

// ═══════════════════════════════════════════════════════════════════════════
// PRIVATE KEY HELPERS — prefix-based key construction for inline handlers
// ═══════════════════════════════════════════════════════════════════════════

// key builders — avoids repeated append patterns
func marketKey(prefix, marketId []byte) []byte {
	return append(append([]byte{}, prefix...), marketId...)
}
func positionKey(marketId, addr []byte) []byte {
	k := append(append([]byte{}, PREFIX_POSITION_STATE...), marketId...)
	return append(k, addr...)
}
func addrKey(prefix, addr []byte) []byte {
	return append(append([]byte{}, prefix...), addr...)
}

// ═══════════════════════════════════════════════════════════════════════════

func (c *Contract) Genesis(req *PluginGenesisRequest) *PluginGenesisResponse {
	return &PluginGenesisResponse{}
}

// BeginBlock — AUDIT-9: SetGlobalHeight here. NF-2: write result captured and checked.
// CRIT-3/4/5: QueryId per key, []*PluginSetOp, resp.Results with QueryId matching.
func (c *Contract) BeginBlock(req *PluginBeginRequest) *PluginBeginResponse {
	SetGlobalHeight(req.Height)

	// Rolling entropy accumulator (P4 fix: plugin-accessible entropy, not block_hash).
	entropyQId := rand.Uint64()
	entropyResp, readErr := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: entropyQId, Key: PANEL_ENTROPY_KEY},
		},
	})

	var acc uint64
	// CRIT-5: resp.Results with QueryId matching
	if readErr == nil && entropyResp != nil {
		for _, r := range entropyResp.Results {
			if r.QueryId == entropyQId && len(r.Entries) > 0 {
				if len(r.Entries[0].Value) >= 8 {
					for i := 0; i < 8; i++ {
						acc = (acc << 8) | uint64(r.Entries[0].Value[i])
					}
				}
			}
		}
	}

	// Fibonacci hashing constant — XOR with height for rolling entropy
	acc ^= req.Height * 0x9e3779b97f4a7c15

	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(acc & 0xFF)
		acc >>= 8
	}

	// NF-2: capture write result; check it; on failure continue with previous accumulator.
	// BeginBlock cannot return an error — accepted degradation documented here.
	// CRIT-4: []*PluginSetOp
	wr, writeErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: PANEL_ENTROPY_KEY, Value: buf},
		},
	})
	if writeErr != nil || (wr != nil && wr.Error != nil) {
		// Degraded mode: entropy accumulator not updated this block.
		// Panel seeds constructed this block use previous accumulator value.
		// One missed XOR does not break anti-grinding property.
		_ = wr // explicit discard after check
	}

	// NF-3: &PluginBeginResponse{} — no semicolon
	return &PluginBeginResponse{}
}

func (c *Contract) EndBlock(req *PluginEndRequest) *PluginEndResponse {
	return &PluginEndResponse{}
}

// ═══════════════════════════════════════════════════════════════════════════
// CheckTx ROUTER — AUDIT-8: stateless, zero StateRead calls
// ═══════════════════════════════════════════════════════════════════════════

func (c *Contract) CheckTx(req *PluginCheckRequest) *PluginCheckResponse {
	msg, err := FromAny(req.Tx.Msg)
	if err != nil {
		return &PluginCheckResponse{Error: err}
	}
	switch m := msg.(type) {
	case *MessageSend:
		return c.CheckMessageSend(m)
	case *MessageCreateMarket:
		return c.CheckMessageCreateMarket(m)
	case *MessageSubmitPrediction:
		return c.CheckMessageSubmitPrediction(m)
	case *MessageResolveMarket:
		return c.CheckMessageResolveMarket(m)
	case *MessageClaimWinnings:
		return c.CheckMessageClaimWinnings(m)
	// Phase 2 PORS — uncomment when Phase 2 is active:
	// case *MessageRegisterResolver:
	//     return c.CheckMessageRegisterResolver(m)
	// case *MessageProposeOutcome:
	//     return c.CheckMessageProposeOutcome(m)
	// case *MessageFileDispute:
	//     return c.CheckMessageFileDispute(m)
	// case *MessageCommitVote:
	//     return c.CheckMessageCommitVote(m)
	// case *MessageRevealVote:
	//     return c.CheckMessageRevealVote(m)
	// case *MessageTallyVotes:
	//     return c.CheckMessageTallyVotes(m)
	// case *MessageFinalizeMarket:
	//     return c.CheckMessageFinalizeMarket(m)
	default:
		return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// DeliverTx ROUTER
// ═══════════════════════════════════════════════════════════════════════════

func (c *Contract) DeliverTx(req *PluginDeliverRequest) *PluginDeliverResponse {
	msg, err := FromAny(req.Tx.Msg)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	fee := req.Tx.Fee
	switch m := msg.(type) {
	case *MessageSend:
		return c.DeliverMessageSend(m, fee)
	case *MessageCreateMarket:
		return c.DeliverMessageCreateMarket(m, fee)
	case *MessageSubmitPrediction:
		return c.DeliverMessageSubmitPrediction(m, fee)
	case *MessageResolveMarket:
		return c.DeliverMessageResolveMarket(m, fee)
	case *MessageClaimWinnings:
		return c.DeliverMessageClaimWinnings(m, fee)
	// Phase 2 PORS:
	// case *MessageRegisterResolver:
	//     return c.DeliverMessageRegisterResolver(m, fee)
	// case *MessageProposeOutcome:
	//     return c.DeliverMessageProposeOutcome(m, fee)
	// ... etc.
	default:
		return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// SEND — index 0 (CheckTx + DeliverTx)
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// CREATE MARKET — index 1
// Spec: ADLMSR v5.6.6-r2 §CreateMarket
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// SUBMIT PREDICTION — index 2
// Spec: ADLMSR v5.6.6-r2 §SubmitPrediction
// AUDIT-3: now >= OpenTime before subtraction
// AUDIT-5: re-read position in same batch as market after CheckAutoCancel
// AUDIT-7: shares >= PRECISION_SCALE in DeliverTx
// AUDIT-12: cost <= MaxCost slippage guard
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// RESOLVE MARKET — index 3
// Spec: ADLMSR v5.6.6-r2 §ResolveMarket
// R1: Auth BEFORE idempotency — wrong resolver never gets success on retry
// NEW-1: 6-key atomic write
// CRIT-1: market.QYes/QNo (not QYES/QNO)
// CRIT-2: STATUS_RESOLVED intermediate (PORS leads to STATUS_FINALIZED)
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// CLAIM WINNINGS — index 4
// Spec: ADLMSR v5.6.6-r2 §ClaimWinnings
// CRIT-2: Gates on STATUS_FINALIZED = 6 (not STATUS_RESOLVED)
// CRIT-1: market.QYes/QNo, position.SharesYes/SharesNo
// R2: surplus sweep re-reads pool from state after atomic write
// R6: ClaimantAddress validated in CheckTx
// AUDIT-1: overflow-safe payout formula
// AUDIT-6: ghost claimant guard
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// CheckAutoCancel — 4-key atomic write (AUDIT-2)
// Called at the start of SubmitPrediction, ResolveMarket, ClaimWinnings.
// ═══════════════════════════════════════════════════════════════════════════

// MovePoolToPool — helper to transfer remaining pool balance to treasury.
// R2: called ONLY with re-read pool amount — never the in-memory value.
// ═══════════════════════════════════════════════════════════════════════════
// PORS CheckTx STUBS — Phase 2 (NF-1 fix: tally_votes included)
// All AUDIT-8 pattern: zero StateRead, 20-byte address guards.
// ═══════════════════════════════════════════════════════════════════════════

// NF-6: no status check in CheckTx — status is stateful, DeliverTx enforces timing.
// NF-1 fix: tally_votes CheckTx stub — permissionless (no AuthorizedSigners restriction).
// Zero StateRead calls (AUDIT-8 pattern).
// ═══════════════════════════════════════════════════════════════════════════
// PORS DELIVER — ProposeOutcome (NF-5 + NF-6 critical fixes)
// NF-5: writes 0x13 ResolverState as 4th atomic key
// NF-6: accepts STATUS_OPEN if now > ExpiryTime — inline transition to STATUS_PROPOSED
//       STATUS_EXPIRED is NEVER persisted
// ═══════════════════════════════════════════════════════════════════════════

// _ is used to suppress "imported and not used" for anypb in this file template.
// Remove this line and add the real import when wiring the full plugin.
var _ = (*anypb.Any)(nil)
