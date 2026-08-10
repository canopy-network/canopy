package contract

// oracle_untrustworthy.go implements NASM Consolidated Spec Section 9.2's
// OracleUntrustworthy() predicate: the single, precise definition combining
// staleness, ARCM's circuit breaker state ({20}), and ARCM's Emergency Mode
// flag ({21}), used to gate burn_nusd and liquidate_nasm_vault against a
// specific collateral asset.
//
// Per the spec's own text: the predicate checks Emergency Mode ({21}) IN
// ADDITION TO the circuit breaker ({20}), not the circuit breaker alone --
// Emergency Mode is a superset condition (stale oracle beyond threshold, OR
// risk-committee override) that does not always coincide with
// circuit-breaker-active, so checking both closes a gap where an asset
// could be in Emergency Mode via committee override without also tripping
// the circuit breaker.
//
// EMERGENCY_THRESHOLD (100 blocks, ARCM v3.10 parameter table) is
// deliberately a DIFFERENT, LARGER threshold than
// DefaultStalenessThresholdBlocks (30 blocks, price_resolve.go) -- the two
// represent different severities. ResolvePrice's 30-block threshold governs
// whether a price is fresh enough to RESOLVE AT ALL (Rule 1); this
// function's 100-block threshold governs whether staleness is severe enough
// to constitute an EMERGENCY. A price can be too stale to resolve
// (ResolvePrice returns found=false) well before it's stale enough to
// trigger Emergency Mode -- the two thresholds are not meant to be the same
// value, and this function does not read or derive from
// DefaultStalenessThresholdBlocks.
//
// CIRCUIT BREAKER SCOPE, DISCLOSED: as of this commit, CircuitBreakerState
// (arbor_state.proto) has no automatic trigger -- see that message's own
// doc comment for the full reasoning (the upstream ARCM/AYIS spec's own
// TWAP/deviation algorithm is marked "OPEN (NF) / Deferred" in its audit
// trail, not simply unimplemented here). This function still checks
// CircuitBreakerState.Active, which will only ever be true after a manual
// set_circuit_breaker override -- the predicate's SHAPE is spec-complete;
// its circuit-breaker INPUT is currently manual-only, not automatic.
const EmergencyThresholdBlocks = 100

// OracleUntrustworthy implements the predicate exactly as specified:
//
// staleness_age := current_block - price.block_height
// circuit_breaker_active := ReadCircuitBreakerState(asset_id)
// emergency_mode_active := ReadEmergencyModeFlag(asset_id)
// return staleness_age > STALENESS_THRESHOLD[tier(asset_id)] ||
//
//	circuit_breaker_active ||
//	emergency_mode_active
//
// staleness_age here is computed against the FRESHEST available submitter
// reading for assetID (the max block_height across all {19} PriceRecord
// entries for this asset), not any single submitter's record -- if even one
// submitter has posted within the threshold, staleness alone should not
// trip this predicate; that is what ResolvePrice's own quorum/median
// resolution already handles for pricing purposes, and this function's
// staleness check exists for a different purpose (emergency detection, not
// price resolution), so it deliberately does not call ResolvePrice or
// duplicate its logic -- a genuinely stale price is caught here directly
// off the freshest raw record, independent of whether ResolvePrice can
// currently resolve a price for this asset at all (an asset already
// failing ResolvePrice's own Rule 1/2 checks should ALSO read as
// untrustworthy here, not bypass this check because resolution already
// failed elsewhere).
func OracleUntrustworthy(c *Contract, assetID string) (bool, *PluginError) {
	currentHeight := c.plugin.CurrentHeight()

	rangePrefix := JoinLenPrefix(PrefixPriceCache, []byte(assetID))
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Ranges: []*PluginRangeRead{
			{QueryId: 0, Prefix: rangePrefix},
		},
	})
	if err != nil {
		return false, err
	}
	if readResp.Error != nil {
		return false, readResp.Error
	}

	var entries []*PluginStateEntry
	for _, result := range readResp.Results {
		if result.QueryId == 0 {
			entries = result.Entries
			break
		}
	}

	var freshestBlock uint64
	var anySubmission bool
	for _, entry := range entries {
		rec := &PriceRecord{}
		if uErr := Unmarshal(entry.Value, rec); uErr != nil {
			continue // one corrupted submitter record does not fail the whole check
		}
		anySubmission = true
		if rec.BlockHeight > freshestBlock {
			freshestBlock = rec.BlockHeight
		}
	}

	// No submission at all for this asset, ever -- maximally untrustworthy
	// (there is nothing to be stale relative to; treat as an immediate
	// staleness trip rather than a false negative from an empty range).
	//
	// [VERIFIED, session finding] This branch is DEAD CODE for this
	// function's two current real callers (burn_nusd.go,
	// liquidate_nasm_vault.go) -- both already call ResolvePrice first and
	// already hard-reject via ErrNasmPriceUnavailable when priceFound is
	// false, before this function is ever reached, so a genuinely empty
	// price range for the asset in question can never actually arrive
	// here in practice today. Kept anyway as a safety net for any FUTURE
	// caller that invokes OracleUntrustworthy() independently, without a
	// prior ResolvePrice gate -- confirmed via direct inspection of both
	// current call sites, not assumed.
	if !anySubmission {
		return true, nil
	}

	var stalenessAge uint64
	if currentHeight >= freshestBlock {
		stalenessAge = currentHeight - freshestBlock
	}
	if stalenessAge > EmergencyThresholdBlocks {
		return true, nil
	}

	cbState, cbFound, cbErr := GetCircuitBreakerState(c, assetID)
	if cbErr != nil {
		return false, cbErr
	}
	if cbFound && cbState.Active {
		return true, nil
	}

	emState, emFound, emErr := GetEmergencyMode(c, assetID)
	if emErr != nil {
		return false, emErr
	}
	if emFound && emState.Active {
		return true, nil
	}

	return false, nil
}
