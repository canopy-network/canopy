package contract

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — Named Constants
// Spec authority:
//   ADLMSR v5.6.6-r2-CORRECTED
//   PORS   v1.0-r2-CORRECTED
// ═══════════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// MARKET STATUS CONSTANTS
// CRIT-2: STATUS_FINALIZED (= 6) is the ClaimWinnings gate — never STATUS_RESOLVED.
// NF-6: STATUS_EXPIRED is never persisted — propose_outcome transitions inline.
// ─────────────────────────────────────────────────────────────────────────────

const (
STATUS_OPEN      uint32 = 0
STATUS_CANCELLED uint32 = 1
STATUS_RESOLVED  uint32 = 2
STATUS_EXPIRED   uint32 = 3
STATUS_PROPOSED  uint32 = 4
STATUS_DISPUTED  uint32 = 5
STATUS_FINALIZED uint32 = 6
STATUS_VOIDED    uint32 = 7
)

// ─────────────────────────────────────────────────────────────────────────────
// PROPOSAL STATUS CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────

const (
PROPOSAL_OPEN      uint32 = 0
PROPOSAL_DISPUTED  uint32 = 1
PROPOSAL_FINALIZED uint32 = 2
)

// ─────────────────────────────────────────────────────────────────────────────
// VOTE STATUS CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────

const (
VOTE_PENDING   uint32 = 0
VOTE_COMMITTED uint32 = 1
VOTE_REVEALED  uint32 = 2
VOTE_TALLIED   uint32 = 3
)

// ─────────────────────────────────────────────────────────────────────────────
// LMSR PRICING CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────

const (
PRECISION_SCALE         uint64 = 1_000_000
MIN_B0                  uint64 = 1_000_000
ELEVATED_RISK_THRESHOLD uint64 = 25_000_000_000
FIBONACCI_HASH_CONSTANT uint64 = 0x9e3779b97f4a7c15
)

// ─────────────────────────────────────────────────────────────────────────────
// TIMING CONSTANTS — ADLMSR (all values in blocks)
// MAX_EXPIRY_TIME: R7 fix — parenthesised expression.
// AUDIT-11: guards uint64 overflow in all post-expiry arithmetic.
// ─────────────────────────────────────────────────────────────────────────────

const (
RESOLUTION_DELAY_BLOCKS uint64 = 100
GRACE_PERIOD_BLOCKS     uint64 = 200
CLAIM_GRACE_PERIOD      uint64 = 1000
)

const MAX_EXPIRY_TIME uint64 = (^uint64(0) -
RESOLUTION_DELAY_BLOCKS - GRACE_PERIOD_BLOCKS - CLAIM_GRACE_PERIOD - 1)

// ─────────────────────────────────────────────────────────────────────────────
// TIMING CONSTANTS — PORS (all values in blocks)
// MIN_DISPUTE_BLOCKS = 34,560 ≈ 48h at ~5s block time (P5)
// ─────────────────────────────────────────────────────────────────────────────

const (
MIN_DISPUTE_BLOCKS  uint64 = 34_560
COMMIT_PHASE_BLOCKS uint64 = 17_280
REVEAL_PHASE_BLOCKS uint64 = 17_280
)

// ─────────────────────────────────────────────────────────────────────────────
// PORS ECONOMIC CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────

const (
MIN_RRS_TO_PROPOSE  uint64 = 10
FINALIZATION_BOUNTY uint64 = 50_000_000
RRS_INITIAL         uint64 = 100
)

// ─────────────────────────────────────────────────────────────────────────────
// PANEL CONSTANTS
// ─────────────────────────────────────────────────────────────────────────────

const (
MIN_PANEL_SIZE           uint32 = 3
ELEVATED_RISK_PANEL_SIZE uint32 = 7
)

// ─────────────────────────────────────────────────────────────────────────────
// PROTOCOL TREASURY
// PRAXIS_TREASURY_ID: destination pool ID for surplus sweeps (R2) and slashes.
// []byte cannot be const in Go — defined as var.
// ─────────────────────────────────────────────────────────────────────────────

var PRAXIS_TREASURY_ID = []byte{
0xe7, 0xc7, 0xda, 0xd1, 0x31, 0xa0, 0x3f, 0x7e,
0xa0, 0xcc, 0x09, 0xa6, 0x37, 0xad, 0x09, 0x6e,
0xb3, 0x49, 0x5f, 0x77,
}

// ─────────────────────────────────────────────────────────────────────────────
// PANEL ENTROPY KEY
// Singleton state key for the 0x1C rolling entropy accumulator.
// Initialised in contract.go init() via JoinLenPrefix.
// ─────────────────────────────────────────────────────────────────────────────

var panelEntropyPrefix = []byte{0x1C}
var PANEL_ENTROPY_KEY []byte
