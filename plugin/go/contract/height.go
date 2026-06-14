package contract

import "sync"

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — Global Height Management
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED (AUDIT-9)
//
// AUDIT-9: globalHeight must be a package-level variable protected by
// sync.RWMutex. Never use c.currentHeight on the Contract struct between
// BeginBlock and DeliverTx — concurrent DeliverTx goroutines would race
// on the struct field, producing non-deterministic height reads.
//
// Pattern:
//   BeginBlock  → SetGlobalHeight(req.Height)
//   DeliverTx   → now := GetGlobalHeight()
//                 if now == 0 { return ErrHeightNotSet() }
// ═══════════════════════════════════════════════════════════════════════════════

var (
globalHeight uint64
heightMu     sync.RWMutex
)

// SetGlobalHeight stores the current block height.
// Called once per block in BeginBlock before any transactions are processed.
func SetGlobalHeight(h uint64) {
heightMu.Lock()
globalHeight = h
heightMu.Unlock()
}

// GetGlobalHeight returns the current block height.
// Called at the top of every DeliverTx handler.
// Returns 0 if BeginBlock has not been called yet — handlers must guard
// against this with: if now == 0 { return ErrHeightNotSet() }
func GetGlobalHeight() uint64 {
heightMu.RLock()
defer heightMu.RUnlock()
return globalHeight
}
