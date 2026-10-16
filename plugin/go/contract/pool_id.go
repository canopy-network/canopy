package contract

import (
	"crypto/sha256"
	"encoding/binary"
)

// marketPoolIdReservedBit is set on every pool id Arbor derives for a market's
// escrow pool. Canopy's own chainId-keyed pools (root chain, committee staking,
// delegation, supply tracking -- see fsm/account.go SetPoolPoints/getSupplyPool)
// are governance-registered uint64 values that will never approach 2^63; no
// legitimate chainId can ever have this bit set. Reserving the entire upper
// half of the uint64 space for plugin-derived pool ids is a structural
// partition, not a probabilistic hash-collision argument -- two disjoint
// ranges, enforced by construction, not by hoping SHA-256 doesn't collide.
const marketPoolIdReservedBit uint64 = 1 << 63

// PoolPurpose distinguishes a market's two independent escrow buckets.
//
// [NO-ASSUMPTIONS FINDING, checked against create_market.go directly] The
// only validation CheckMessageCreateMarket applies to CollateralAssetId/
// DebtAssetId is that both are non-empty (len(...) == 0 check). There is NO
// check anywhere in this codebase that the two are DISTINCT. Same-asset
// markets (e.g. an ETH-collateral/ETH-debt leverage market) are therefore
// fully permitted by the code as written today, whether or not that was an
// intentional design decision. A single pool per market_id would, for such
// a market, silently commingle collateral posted by borrowers with
// principal supplied by lenders into ONE Pool.Amount field -- two
// completely different obligations sharing one indistinguishable balance.
// This is not a same-asset-only concern either: even where the two assets
// DO differ, BorrowerPosition.CollateralQuantity and the
// total_borrowed/total_supplied accounting are separate domains that must
// never share one custody bucket regardless of whether the underlying token
// happens to be identical. Every market therefore gets TWO derived pool
// ids, unconditionally, never one -- this is a mainnet-facing correctness
// requirement, not a naming convenience.
type PoolPurpose string

const (
	// PoolPurposeSupply backs the lender-side deposit/withdraw flow
	// (AYIS Section 4, ARCM Section 19.2.1b) -- denominated in the
	// market's DebtAssetId (what lenders supply and borrowers draw down).
	PoolPurposeSupply PoolPurpose = "supply"
	// PoolPurposeCollateral backs the borrower-side deposit_collateral/
	// withdraw_collateral flow (ARCM Section 19.2) -- denominated in the
	// market's CollateralAssetId. Kept structurally separate from
	// PoolPurposeSupply even when CollateralAssetId == DebtAssetId for a
	// given market, per the finding above.
	PoolPurposeCollateral PoolPurpose = "collateral"
	// PoolPurposeNasmVault backs NASM's mint_nusd/burn_nusd collateral
	// custody (NASM Consolidated Spec Section 4). Deliberately a THIRD,
	// independent PoolPurpose value, not a reuse of PoolPurposeCollateral --
	// KeyForMarketPoolId's derivation is SHA-256("<purpose>:<id>") with no
	// type distinction between the id argument being a market_id or a
	// vault_id; it is just a string. Reusing PoolPurposeCollateral for NASM
	// vaults would mean an ARCM market and a NASM vault sharing the same
	// literal id string collide onto the IDENTICAL pool id -- not a
	// probabilistic hash risk, a guaranteed collision by construction,
	// since the purpose prefix would be identical too. This would silently
	// commingle ARCM lending collateral with NASM NUSD-backing collateral,
	// the exact cross-product contamination the Treasury split
	// (PrefixTreasuryArbor/PrefixTreasuryNASM in state_keys.go) was built
	// to prevent on the accounting side. A distinct purpose string forces
	// a distinct hash input regardless of id-string overlap, preserving
	// this file's own structural-partition-by-construction guarantee.
	PoolPurposeNasmVault PoolPurpose = "nasm-vault"
)

// KeyForMarketPoolId derives the Pool.Id used as one of a given market's
// two escrow accounts, selected by purpose. Pools are FSM-native
// "owner-less designation funds" with no private key (fsm/account.go) --
// exactly the custody primitive Section 19.1 Appendix C's poolPrefix{2}
// comment names "escrow" as a use case for, and one of only two core
// prefixes (accountPrefix, poolPrefix) the FSM's assertPluginKeyWritable()
// allowlists for plugin writes (fsm/state.go).
//
// SHA-256 over "<purpose>:<market_id>" (purpose PREFIXED with a fixed
// separator, not appended or concatenated bare -- prevents a pathological
// market_id string from ever producing the same pre-hash input as a
// different (purpose, market_id) pair; a fixed-set enum value followed by a
// literal ':' has no such ambiguity the way naive concatenation could).
// Top 63 bits of the digest; the reserved bit is cleared from the hash
// output before being forced on, so it never contributes entropy that could
// make two different (purpose, market_id) pairs collide pre-mask.
//
// NOTE: this same function is reused for NASM's vault_id-keyed pool ids
// (PoolPurposeNasmVault) -- the "market_id" parameter name is Arbor
// lending's original naming, but the function itself is purpose-agnostic
// over its second argument; it is the purpose string prefix that keeps
// the two id namespaces from colliding, not the parameter's name.
func KeyForMarketPoolId(marketId string, purpose PoolPurpose) uint64 {
	digest := sha256.Sum256([]byte(string(purpose) + ":" + marketId))
	raw := binary.BigEndian.Uint64(digest[:8])
	return (raw &^ marketPoolIdReservedBit) | marketPoolIdReservedBit
}

// IsMarketPoolId reports whether a uint64 falls in Arbor's reserved
// plugin-pool-id range. Defensive check for call sites that read a Pool.Id
// back from state and need to confirm it's actually Arbor's before trusting
// its balance semantics (e.g. a devnet invariant checker, or a future
// governance/indexer tool enumerating pools).
func IsMarketPoolId(id uint64) bool {
	return id&marketPoolIdReservedBit != 0
}
