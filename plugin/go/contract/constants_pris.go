package contract

// ─────────────────────────────────────────────────────────────────────────────
// PRIS v1.0-r3 CONSTANTS
// Spec authority: PRIS v1.0-r3
// ─────────────────────────────────────────────────────────────────────────────

const (
// Epoch timing
PRIS_EPOCH_BLOCKS            uint64 = 1_000    // ~83 minutes at 5s/block
PRIS_BUILDER_EPOCH_BLOCKS    uint64 = 120_960  // ~7 days at 5s/block
PRIS_INVESTOR_VESTING_BLOCKS uint64 = 241_920  // ~14 days at 5s/block

// Treasury distribution BPS (basis points, 10000 = 100%)
PRIS_RESOLVER_SHARE_BPS  uint64 = 2_000 // 20%
PRIS_BUILDER_SHARE_BPS   uint64 = 2_000 // 20%
PRIS_COMMUNITY_SHARE_BPS uint64 = 2_000 // 20%
PRIS_INVESTOR_SHARE_BPS  uint64 = 2_000 // 20%
PRIS_PROTOCOL_SHARE_BPS  uint64 = 2_000 // 20%

// Fee BPS
CREATOR_FEE_BPS       uint64 = 100   // 1% of tradeCost — charged on top
RESOLVER_FEE_BPS      uint64 = 100   // 1% of tradeCost — charged on top
TX_TREASURY_SPLIT_BPS uint64 = 5_000 // 50% of TX fees to treasury

// RRS
// r3 fix R3-1: RRS_INITIAL reduced from 100 to 10.
// New resolvers start at Bronze — no Sybil recycling advantage.
PRIS_RRS_INITIAL uint64 = 10
PRIS_RRS_FLOOR   uint64 = 0 // Slashed resolvers lose all tier weight
)

// ComputeBps applies basis points to an amount.
// ComputeBps(amount, 2000) = amount * 20 / 100
func ComputeBps(amount, bps uint64) uint64 {
return amount * bps / 10_000
}

// ─────────────────────────────────────────────────────────────────────────────
// PRIS AUTHORIZED WALLET ADDRESSES
// Hardcoded protocol constants — all claim TXs validate signer against these.
// ─────────────────────────────────────────────────────────────────────────────

var (
PRAXIS_BUILDER_ADDR = []byte{
0x95, 0x43, 0x78, 0xba, 0x10, 0x9c, 0x5c, 0xa4,
0x5b, 0x23, 0xbf, 0xa2, 0x84, 0xf3, 0xac, 0x70,
0xe2, 0x67, 0x1b, 0x87,
}
PRAXIS_COMMUNITY_ADDR = []byte{
0x15, 0xe6, 0x58, 0x69, 0x8d, 0x25, 0x10, 0x79,
0x93, 0x39, 0x27, 0x3f, 0x6f, 0xcc, 0xb0, 0x48,
0x4c, 0x4f, 0x4b, 0x6f,
}
PRAXIS_INVESTOR_ADDR = []byte{
0x12, 0x5c, 0x1b, 0xb8, 0x03, 0xa2, 0xdd, 0x91,
0x94, 0xdc, 0xa4, 0x0d, 0x77, 0x44, 0x5c, 0xf7,
0x56, 0x47, 0xcb, 0x12,
}
PRAXIS_PROTOCOL_ADDR = []byte{
0xc1, 0x76, 0x4f, 0x10, 0xad, 0x67, 0x25, 0x58,
0xaf, 0xe1, 0xa3, 0xb6, 0x66, 0x18, 0x5f, 0xd1,
0x41, 0xae, 0x1e, 0xa8,
}
)

// Unstake constants
const (
PRIS_UNSTAKE_UNBONDING_BLOCKS uint64 = 120_960 // 7 days at 5s/block
PRIS_UNSTAKE_PARTIAL_RRS_HIT  uint64 = 10      // RRS penalty for partial unstake
)
