package contract

// stalenessThresholdTable is ARCM Section 15's per-tier staleness
// threshold (50/30/20/10 blocks for Tier 0-3). Unlike TierParams'
// LTV/LIF fields, the Section 15 consolidated parameter table does NOT
// tag per-tier staleness as IMMUTABLE the way it explicitly tags
// MAX_PRICE_DEVIATION, GOVERNANCE_TIMELOCK_STD, and
// NATIVE_ASSET_MAX_DECIMALS -- so this is deliberately kept as its own
// table, separate from tierParamsTable, rather than folded into
// TierParams' struct, to avoid implying an immutability guarantee the
// spec does not make for this parameter.
//
// Formerly a single flat DefaultStalenessThresholdBlocks = 30 constant
// (Tier 1's value, used for every asset regardless of tier) -- replaced
// here now that ResolvePrice looks up the asset's real tier via
// GetAssetTier/GetStalenessThreshold instead of using one default for
// all assets.
var stalenessThresholdTable = map[uint8]uint64{
	0: 50, // Tier 0 -- CNPY
	1: 30, // Tier 1 -- Blue-chip
	2: 20, // Tier 2 -- Standard
	3: 10, // Tier 3 -- Restricted
}

// GetStalenessThreshold looks up a tier's staleness threshold, in blocks.
// found=false means the tier byte was not 0-3 -- mirrors GetTierParams'
// own found=false contract exactly.
func GetStalenessThreshold(tier uint8) (blocks uint64, found bool) {
	b, ok := stalenessThresholdTable[tier]
	return b, ok
}

// MinReporters is ARCM Section 10 Rule 2's oracle quorum floor.
//
// [DEVNET-ONLY OVERRIDE, TEMPORARY] Set to 1 instead of the spec'd 3
// because this devnet currently controls only a single signing key
// (see keystore.json, address 7961113f844bcf86dfd79570f23a8e3a59b10751),
// so a real 3-submitter quorum cannot be exercised yet. MUST be restored
// to 3 before anything resembling production, and before any test that
// claims to validate ARCM Section 10's quorum requirement -- this
// override means ResolvePrice currently verifies resolution mechanics
// only, NOT quorum safety.
const MinReporters = 1

// ResolvePrice implements ARCM Section 10's Rule 1 (freshness) and Rule 2
// (quorum) against the price records update_price.go ingests.
//
// NOT implemented (disclosed scope limit, not a silent gap): Rule 3
// (confidence-bps fallback) and Rule 4 (deviation / circuit breaker).
// Circuit breaker state ({20}, PrefixCircuitBreaker) is a reserved key with
// no writer yet -- ResolvePrice cannot consult what nothing produces.
//
// Resolution method: median of fresh submitter readings meeting quorum.
// Median, not mean, is used because it is harder for a single outlier
// submitter to move than a mean -- a conservative choice pending Rule 4's
// real deviation check (Principle 5, oracle paranoia).
func ResolvePrice(c *Contract, assetID string) (price uint64, found bool, pErr *PluginError) {
	rangePrefix := JoinLenPrefix(PrefixPriceCache, []byte(assetID))
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: 0, Prefix: rangePrefix},
		},
	})
	if err != nil {
		return 0, false, err
	}
	if readResp.Error != nil {
		return 0, false, readResp.Error
	}

	var entries []*PluginStateEntry
	for _, result := range readResp.Results {
		if result.QueryId == 0 {
			entries = result.Entries
			break
		}
	}

	// Per-tier staleness threshold (ARCM Section 15), replacing the old
	// flat 30-block default. An asset with no tier registry entry, or a
	// tier byte with no threshold entry (should not happen for 0-3, but
	// checked explicitly rather than assumed), resolves as found=false --
	// mirrors every other GetAssetTier caller's convention (borrow.go,
	// collateral.go, liquidate_position.go all hard-reject on
	// !tierFound) rather than silently falling back to a default
	// threshold for an asset the tier registry does not recognize.
	tier, tierFound, tErr := GetAssetTier(c, assetID)
	if tErr != nil {
		return 0, false, tErr
	}
	if !tierFound {
		return 0, false, nil
	}
	stalenessThreshold, stFound := GetStalenessThreshold(tier)
	if !stFound {
		return 0, false, nil
	}

	currentHeight := c.plugin.CurrentHeight()
	var freshPrices []uint64
	for _, entry := range entries {
		rec := &PriceRecord{}
		if uErr := Unmarshal(entry.Value, rec); uErr != nil {
			continue // one corrupted submitter record does not fail the whole resolution
		}
		if currentHeight >= rec.BlockHeight && currentHeight-rec.BlockHeight > stalenessThreshold {
			continue // Rule 1: stale
		}
		if rec.Price == 0 {
			continue
		}
		freshPrices = append(freshPrices, rec.Price)
	}

	if len(freshPrices) < MinReporters {
		return 0, false, nil // Rule 2: quorum not met
	}

	return medianUint64(freshPrices), true, nil
}

// medianUint64 sorts a COPY of vals, leaving the caller's slice (built from
// range-read iteration order -- key-ordered, not insertion-ordered, so this
// is safe regardless) untouched.
func medianUint64(vals []uint64) uint64 {
	sorted := make([]uint64, len(vals))
	copy(sorted, vals)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
