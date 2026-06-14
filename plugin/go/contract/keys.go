package contract

import "encoding/binary"

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — State Key Functions
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED + PORS v1.0-r2-CORRECTED
//
// All keys are built with JoinLenPrefix from plugin.go which length-prefixes
// each segment, guaranteeing no two different key types can collide even if
// their raw bytes overlap.
//
// Canopy base layer prefixes (do not reuse):
//   0x01  Account
//   0x02  Pool (fee pool)
//   0x07  FeeParams
//
// Praxis state prefixes (0x10–0x1C exclusively):
//   0x10  MarketState        per market_id
//   0x11  PositionState      per market_id + claimant_address
//   0x12  OutcomeState       per market_id
//   0x13  ResolverState      per market_id (written by propose_outcome — NF-5)
//   0x14  TreasuryReserve    per market_id
//   0x16  ResolverRecord     global per resolver_address
//   0x17  ProposalRecord     per market_id
//   0x18  DisputeRecord      per market_id
//   0x19  VoteCommit         per market_id + voter_addr
//   0x1A  VoteReveal         per market_id + voter_addr
//   0x1B  SlashRecord        per resolver_address
//   0x1C  PanelEntropyAccum  singleton
// ═══════════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// CANOPY BASE LAYER KEYS
// ─────────────────────────────────────────────────────────────────────────────

var (
accountPrefix = []byte{0x01}
poolPrefix    = []byte{0x02}
paramsPrefix  = []byte{0x07}
)

// KeyForAccount returns the state key for an Account record.
// addr must be exactly 20 bytes.
func KeyForAccount(addr []byte) []byte {
return JoinLenPrefix(accountPrefix, addr)
}

// KeyForFeePool returns the state key for the chain fee pool.
func KeyForFeePool(chainId uint64) []byte {
return JoinLenPrefix(poolPrefix, uint64ToBytes(chainId))
}

// KeyForFeeParams returns the state key for the FeeParams singleton.
func KeyForFeeParams() []byte {
return JoinLenPrefix(paramsPrefix, []byte("/f/"))
}

// ─────────────────────────────────────────────────────────────────────────────
// ADLMSR STATE KEYS
// ─────────────────────────────────────────────────────────────────────────────

var (
marketPrefix        = []byte{0x10}
positionPrefix      = []byte{0x11}
outcomePrefix       = []byte{0x12}
resolverStatePrefix = []byte{0x13}
treasuryPrefix      = []byte{0x14}
)

// KeyForMarket returns the state key for a MarketState record.
// market_id is a 20-byte derived identifier.
func KeyForMarket(marketId []byte) []byte {
return JoinLenPrefix(marketPrefix, marketId)
}

// KeyForPosition returns the state key for a PositionState record.
// Composite key: market_id (20 bytes) + claimant_address (20 bytes).
func KeyForPosition(marketId []byte, addr []byte) []byte {
composite := make([]byte, len(marketId)+len(addr))
copy(composite, marketId)
copy(composite[len(marketId):], addr)
return JoinLenPrefix(positionPrefix, composite)
}

// KeyForOutcome returns the state key for an OutcomeState record.
// Presence of this key is the idempotency sentinel for ResolveMarket.
func KeyForOutcome(marketId []byte) []byte {
return JoinLenPrefix(outcomePrefix, marketId)
}

// KeyForResolverState returns the state key for a per-market ResolverState.
// Written by propose_outcome as the 4th atomic key (NF-5 fix).
// Distinct from KeyForResolverRecord (global per resolver_address).
func KeyForResolverState(marketId []byte) []byte {
return JoinLenPrefix(resolverStatePrefix, marketId)
}

// KeyForTreasuryReserve returns the state key for a per-market TreasuryReserve.
func KeyForTreasuryReserve(marketId []byte) []byte {
return JoinLenPrefix(treasuryPrefix, marketId)
}

// ─────────────────────────────────────────────────────────────────────────────
// PORS STATE KEYS
// ─────────────────────────────────────────────────────────────────────────────

var (
resolverRecordPrefix = []byte{0x16}
proposalPrefix       = []byte{0x17}
disputePrefix        = []byte{0x18}
voteCommitPrefix     = []byte{0x19}
voteRevealPrefix     = []byte{0x1A}
slashRecordPrefix    = []byte{0x1B}
)

// KeyForResolverRecord returns the state key for a global ResolverRecord.
// Keyed by resolver_address — this is the global resolver profile.
// Distinct from KeyForResolverState (per-market).
func KeyForResolverRecord(resolverAddr []byte) []byte {
return JoinLenPrefix(resolverRecordPrefix, resolverAddr)
}

// KeyForProposal returns the state key for a ProposalRecord.
// Presence of this key is the idempotency sentinel for propose_outcome.
func KeyForProposal(marketId []byte) []byte {
return JoinLenPrefix(proposalPrefix, marketId)
}

// KeyForDispute returns the state key for a DisputeRecord.
func KeyForDispute(marketId []byte) []byte {
return JoinLenPrefix(disputePrefix, marketId)
}

// KeyForVoteCommit returns the state key for a panel member's VoteCommit.
// Composite key: market_id (20 bytes) + voter_addr (20 bytes).
func KeyForVoteCommit(marketId []byte, voterAddr []byte) []byte {
composite := make([]byte, len(marketId)+len(voterAddr))
copy(composite, marketId)
copy(composite[len(marketId):], voterAddr)
return JoinLenPrefix(voteCommitPrefix, composite)
}

// KeyForVoteReveal returns the state key for a panel member's VoteReveal.
// Composite key: market_id (20 bytes) + voter_addr (20 bytes).
func KeyForVoteReveal(marketId []byte, voterAddr []byte) []byte {
composite := make([]byte, len(marketId)+len(voterAddr))
copy(composite, marketId)
copy(composite[len(marketId):], voterAddr)
return JoinLenPrefix(voteRevealPrefix, composite)
}

// KeyForSlashRecord returns the state key for a resolver's SlashRecord.
func KeyForSlashRecord(resolverAddr []byte) []byte {
return JoinLenPrefix(slashRecordPrefix, resolverAddr)
}

// ─────────────────────────────────────────────────────────────────────────────
// ENTROPY KEY
// Initialised in contract.go init() after JoinLenPrefix is available.
// ─────────────────────────────────────────────────────────────────────────────

// KeyForPanelEntropy returns the singleton state key for the rolling
// entropy accumulator. Called once in init() to set PANEL_ENTROPY_KEY.
func KeyForPanelEntropy() []byte {
return JoinLenPrefix(panelEntropyPrefix, []byte("/pe/"))
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

// uint64ToBytes encodes a uint64 as 8 big-endian bytes.
// Used for chain ID in fee pool keys.
func uint64ToBytes(u uint64) []byte {
b := make([]byte, 8)
binary.BigEndian.PutUint64(b, u)
return b
}

// KeyForMarketPool returns the state key for a per-market liquidity pool.
// Uses the same 0x02 prefix as the chain fee pool but keyed by market_id.
// The JoinLenPrefix encoding ensures no collision with KeyForFeePool(chainId).
func KeyForMarketPool(marketId []byte) []byte {
return JoinLenPrefix(poolPrefix, marketId)
}

// KeyForTreasuryPool returns the state key for the global Praxis treasury pool.
// Prefix 0x1D — receives surplus sweeps from finalized/cancelled markets.
func KeyForTreasuryPool() []byte {
return JoinLenPrefix([]byte{0x1D}, []byte("/treasury/"))
}
