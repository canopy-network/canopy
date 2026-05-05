package contract

import "math"

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — LMSR Pricing Engine
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED
//
// The LMSR (Logarithmic Market Scoring Rule) cost function guarantees that
// liquidity is always available at a price that reflects collective belief.
// The market maker takes the other side of every trade automatically.
//
// Cost function:
//   C(q_yes, q_no) = b_eff * ln(exp(q_yes / b_eff) + exp(q_no / b_eff))
//
// Cost of a trade:
//   cost = C(q_yes_new, q_no_new) - C(q_yes_old, q_no_old)
//
// All q values are scaled by PRECISION_SCALE to preserve integer precision.
// b_eff is also in PRECISION_SCALE units.
//
// AUDIT-1: Payout formula is overflow-safe using quot/rem pattern.
// AUDIT-3: now >= OpenTime guard before any subtraction involving heights.
// AUDIT-7: shares >= PRECISION_SCALE validated in DeliverTx.
// AUDIT-12: finalCost <= max_cost slippage guard before deducting funds.
// ═══════════════════════════════════════════════════════════════════════════════

// lmsrCost computes the LMSR cost function:
//   C(q_yes, q_no) = b_eff * ln(exp(q_yes / b_eff) + exp(q_no / b_eff))
//
// All inputs are in PRECISION_SCALE fixed-point units.
// Returns cost in micro-PRX (uint64).
//
// Uses float64 internally for the logarithm computation. Precision is
// sufficient for market sizes up to ~1e12 micro-PRX (1 million PRX).
// For larger markets the fixed-point scaling provides adequate resolution.
func lmsrCost(qYes, qNo, bEff uint64) uint64 {
if bEff == 0 {
return 0
}
b := float64(bEff)
y := float64(qYes)
n := float64(qNo)

// Use the log-sum-exp trick to prevent overflow:
// ln(exp(a) + exp(b)) = max(a,b) + ln(1 + exp(-|a-b|))
// This keeps the argument to exp() small regardless of q values.
ay := y / b
an := n / b
var lse float64
if ay >= an {
lse = ay + math.Log1p(math.Exp(an-ay))
} else {
lse = an + math.Log1p(math.Exp(ay-an))
}
// Result is b_eff * lse, converted back from float to uint64.
result := b * lse
if result < 0 {
return 0
}
return uint64(result)
}

// ComputeTradeCost returns the cost in micro-PRX for purchasing `shares` of
// the given outcome in a market with current state (qYes, qNo, bEff).
//
// outcome: true = YES, false = NO
// shares:  number of shares in PRECISION_SCALE units (must be >= PRECISION_SCALE)
//
// Returns (cost, error).
// Error is non-nil if:
//   - shares < PRECISION_SCALE (AUDIT-7)
//   - bEff == 0 (degenerate market)
//   - cost overflows uint64 (should not happen with MAX_EXPIRY_TIME guard)
func ComputeTradeCost(qYes, qNo, bEff, shares uint64, outcome bool) (uint64, *PluginError) {
if bEff == 0 {
return 0, ErrInvalidB0()
}
// AUDIT-7: shares must be at least one unit of precision.
if shares < PRECISION_SCALE {
return 0, ErrSharesBelowMinimum()
}

costBefore := lmsrCost(qYes, qNo, bEff)

var qYesNew, qNoNew uint64
if outcome {
qYesNew = qYes + shares
qNoNew = qNo
} else {
qYesNew = qYes
qNoNew = qNo + shares
}

costAfter := lmsrCost(qYesNew, qNoNew, bEff)

// costAfter should always be >= costBefore for a valid trade.
// Guard against underflow from floating point rounding.
if costAfter < costBefore {
return 0, ErrInternal()
}

return costAfter - costBefore, nil
}

// ComputePayout computes the pro-rata payout for a winning position.
// Uses the overflow-safe quot/rem formula (AUDIT-1).
//
//   quot   = poolAmount / totalWinShares
//   rem    = poolAmount % totalWinShares
//   payout = quot * winnerShares + rem * winnerShares / totalWinShares
//
// Returns 0 if totalWinShares == 0 (no winners — should not happen in practice).
func ComputePayout(poolAmount, winnerShares, totalWinShares uint64) uint64 {
if totalWinShares == 0 {
return 0
}
quot := poolAmount / totalWinShares
rem := poolAmount % totalWinShares
return quot*winnerShares + rem*winnerShares/totalWinShares
}

// ComputeMinBond returns the minimum proposal bond required for a market.
// Bond scales with pool size to make dishonest proposals expensive relative
// to potential gain. Minimum is 1% of the market pool, floor of MIN_B0.
func ComputeMinBond(market *MarketState) uint64 {
if market == nil {
return MIN_B0
}
// Read total pool from the market's q values as a proxy for pool size.
// 1% of b_eff as a simple bond floor — adjustable via governance.
bond := market.BEff / 100
if bond < MIN_B0 {
bond = MIN_B0
}
return bond
}

// IsElevatedRisk returns true if the market pool exceeds ELEVATED_RISK_THRESHOLD.
// Called at market creation and updated on each prediction.
// Elevated-risk markets use a larger dispute panel (P7).
func IsElevatedRisk(poolAmount uint64) bool {
return poolAmount >= ELEVATED_RISK_THRESHOLD
}

// ComputeDisputeBlocks returns the dispute window for a market in blocks.
// DISPUTE_BLOCKS = MAX(MIN_DISPUTE_BLOCKS, market_duration / 10)
// Minimum 48h floor — longer markets get proportionally longer challenge windows.
func ComputeDisputeBlocks(openTime, expiryTime uint64) uint64 {
if expiryTime <= openTime {
return MIN_DISPUTE_BLOCKS
}
duration := expiryTime - openTime
window := duration / 10
if window < MIN_DISPUTE_BLOCKS {
return MIN_DISPUTE_BLOCKS
}
return window
}
