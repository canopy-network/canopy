package contract

// DefaultStalenessThresholdBlocks is a TEMPORARY, disclosed placeholder for
// ARCM Section 18.1's per-tier staleness threshold (50/30/20/10 blocks for
// Tier 0-3). Proper per-tier thresholds require ResolvePrice to know the
// asset's tier, which is straightforward to wire in later but deferred here
// in favor of one conservative default -- same disclosed-placeholder
// pattern as asset_tier.go's assetTierAuthority and deposit.go's MIN_DEPOSIT.
// Tier 1's threshold (30 blocks) is used as the default: stricter than
// Tier 2/3 would require, looser than Tier 0.
const DefaultStalenessThresholdBlocks = 30

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

	currentHeight := c.plugin.CurrentHeight()
	var freshPrices []uint64
	for _, entry := range entries {
		rec := &PriceRecord{}
		if uErr := Unmarshal(entry.Value, rec); uErr != nil {
			continue // one corrupted submitter record does not fail the whole resolution
		}
		if currentHeight >= rec.BlockHeight && currentHeight-rec.BlockHeight > DefaultStalenessThresholdBlocks {
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
