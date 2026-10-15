package canoliq

// tvlcapfixed.go replaces the percentage TVL cap with a fixed ceiling on
// mainnet, from a pinned height.
//
// The percentage cap (WP §9.4) bounds canoLiq to tvl_cap_bps of the stake on
// its own chain so it can't accumulate a controlling share of the stake that
// secures it. On committee 29 that stake does not secure consensus (blocks are
// produced by the root-chain committee), and pooled deposits are held liquid,
// not bonded, so they never add to canoLiq's share of it. What the cap did do
// was bound the pool to about a third of the chain-29 stake, which leaves
// almost no room for new depositors.
//
// From MainnetFixedTvlCapHeight the pool is bounded by a fixed amount instead.
// It still limits how much is exposed while the protocol is young, and it is
// the TVL that autonomy graduation requires (GraduationMinTvlUcnpy). Raising
// or removing it later is another height-gated release or a governance
// decision. Other profiles keep the percentage cap their tests and devnets
// were built on.

// MainnetFixedTvlCapHeight is the committee-29 block from which deposits are
// bounded by MainnetFixedTvlCapUcnpy instead of the percentage cap. It must be
// above the height at which every committee-29 node runs this release; pick it
// with margin for pluginAutoUpdate.
const MainnetFixedTvlCapHeight uint64 = 132_200

// MainnetFixedTvlCapUcnpy is the fixed ceiling on total pooled CNLQ: 50M.
const MainnetFixedTvlCapUcnpy uint64 = 50_000_000_000_000

// TVLCapStatusFixed — the fixed mainnet ceiling governs deposits.
const TVLCapStatusFixed = "fixed"

// fixedTvlCapActive reports whether the fixed ceiling replaces the
// percentage cap at `height` for `profile`.
func fixedTvlCapActive(profile string, height uint64) bool {
	return profile == ProfileMainnet && height >= MainnetFixedTvlCapHeight
}
