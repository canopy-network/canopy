package contract

import "os"

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
MIN_B0                  uint64 = 60_000_000
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
// Issue-18: DISPUTE_BLOCKS is block-count based, not wall-clock.
// At 5s/block the floor is ~48h. If block time deviates (e.g. 4s/block
// during validator churn), the wall-clock window shrinks proportionally.
// Future: anchor to wall-clock via a protocol time parameter.
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
// RRS VOTE WEIGHT TIERS (Layer 2 — anti-whale panel protection)
// Bronze: RRS 10–49  → weight 1
// Silver: RRS 50–199 → weight 2
// Gold:   RRS 200+   → weight 3
// Hard cap at 3 prevents infinite influence via staking.
// ─────────────────────────────────────────────────────────────────────────────
const (
RRS_SILVER_THRESHOLD uint64 = 50
RRS_GOLD_THRESHOLD   uint64 = 200
VOTE_WEIGHT_BRONZE   uint32 = 1
VOTE_WEIGHT_SILVER   uint32 = 2
VOTE_WEIGHT_GOLD     uint32 = 3
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

// TEST_MODE — enables compressed timing windows for local testing.
// Controlled by the PRAXIS_TEST_MODE environment variable.
// Defaults to false — safe for mainnet.
// To enable: PRAXIS_TEST_MODE=true ./go-plugin
// COI-3: maximum fractional share of the winning side any single address may hold.
// Expressed in basis points: 2000 = 20%.
// Cap is enforced on shares (not CostPaid) so early cheap buyers face the same
// limit as late entrants — prevents single-address dominant share accumulation.
// Limitations (by design):
//   - Does not prevent multi-address (Sybil) wash trading — on-chain identity
//     is not enforced; two addresses can each hold up to 20%.
//   - Cap is share-based so pool growth does not progressively loosen it.
const MAX_POSITION_BPS uint64 = 2000

var TEST_MODE = os.Getenv("PRAXIS_TEST_MODE") == "true"
const TEST_DISPUTE_BLOCKS        uint64 = 20
const TEST_RESOLUTION_DELAY      uint64 = 2
const TEST_GRACE_PERIOD          uint64 = 2
const TEST_CLAIM_GRACE_PERIOD    uint64 = 5
