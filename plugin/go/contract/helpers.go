package contract

import (
"crypto/sha256"
"encoding/binary"
"math/big"
"sync/atomic"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — Shared Helpers
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED + PORS v1.0-r2-CORRECTED
// ═══════════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// QUERY ID COUNTER
// Uses atomic counter instead of math/rand — rand collisions within a batch
// silently misroute state reads. Atomic counter guarantees uniqueness.
// ─────────────────────────────────────────────────────────────────────────────

var queryCounter uint64

func nextQueryId() uint64 {
return atomic.AddUint64(&queryCounter, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// OVERFLOW-SAFE ARITHMETIC
// mulDiv computes (a * b) / c using big.Int for the intermediate product.
// Prevents uint64 overflow in proportional calculations.
// Returns 0 if c == 0. Caps at MaxUint64 if result overflows.
// ─────────────────────────────────────────────────────────────────────────────

func mulDiv(a, b, c uint64) uint64 {
if c == 0 {
return 0
}
num := new(big.Int).Mul(
new(big.Int).SetUint64(a),
new(big.Int).SetUint64(b),
)
result := new(big.Int).Div(num, new(big.Int).SetUint64(c))
maxU64 := new(big.Int).SetUint64(^uint64(0))
if result.Cmp(maxU64) > 0 {
return ^uint64(0)
}
return result.Uint64()
}

// ─────────────────────────────────────────────────────────────────────────────
// MARKET ID DERIVATION
// market_id = SHA256(creator_address || nonce_bytes)[:20]
// Deterministic — no sequential counter needed. Verifiable without state read.
// ─────────────────────────────────────────────────────────────────────────────

func DeriveMarketId(creatorAddr []byte, nonce uint64) []byte {
nonceBytes := make([]byte, 8)
binary.BigEndian.PutUint64(nonceBytes, nonce)
input := append(creatorAddr, nonceBytes...)
hash := sha256.Sum256(input)
return hash[:20]
}

// ─────────────────────────────────────────────────────────────────────────────
// COMMIT HASH VERIFICATION
// commit_hash = SHA256(vote_byte || nonce || voter_addr)
// vote_byte: 0x01 for true (YES), 0x00 for false (NO)
// ─────────────────────────────────────────────────────────────────────────────

func ComputeCommitHash(vote bool, nonce []byte, voterAddr []byte) []byte {
var voteByte byte
if vote {
voteByte = 0x01
} else {
voteByte = 0x00
}
input := make([]byte, 0, 1+len(nonce)+len(voterAddr))
input = append(input, voteByte)
input = append(input, nonce...)
input = append(input, voterAddr...)
hash := sha256.Sum256(input)
return hash[:]
}

// ─────────────────────────────────────────────────────────────────────────────
// BYTES EQUAL
// Convenience wrapper — avoids importing bytes in every handler file.
// ─────────────────────────────────────────────────────────────────────────────

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
