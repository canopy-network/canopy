package contract

// NasmTierParams holds one NASM tier's LTV_n_max and LTV_n_liq (both bps),
// per NASM Consolidated Spec Section 3.1's collateral tier table. Unlike
// ARCM's TierParams (tier_params.go), NASM has no LIF field of its own --
// liquidation incentive for NUSD vaults is computed using ARCM's existing
// LIF formula, substituting these tighter LTV_n_liq values as input (NASM
// Spec Section 7: "Recomputed using NASM's tighter LTV_n_liq values --
// produces a smaller LIF band"), not a separate NASM-owned LIF constant.
//
// Values are IMMUTABLE hardcoded launch-config constants, matching ARCM's
// tierParamsTable precedent exactly -- NASM Spec Section 18's parameter
// table lists these as governance-bounded (LTV_n_max Tier N-0: 50%-70%,
// etc.), but no governance-write path exists yet for NASM parameters
// (update_nasm_params, NASM Spec Section 13, is not yet implemented). This
// mirrors interest_rate.go's disclosed hardcoded-pending-governance-store
// pattern.
type NasmTierParams struct {
	LTVMaxBps uint64
	LTVLiqBps uint64
}

// nasmTierParamsTable maps a NASM tier (0=N-0, 1=N-1) to its LTV bounds.
// NASM Spec Section 3.1: N-2 (RWA fractions) and N-3 (ARCM Tier 2/3 assets)
// are explicitly "Not eligible" for NUSD mint-backing at all -- there is no
// table entry for them, deliberately, matching tierParamsTable's own
// absent-Tier-4 precedent in tier_params.go.
var nasmTierParamsTable = map[uint8]NasmTierParams{
	0: {LTVMaxBps: 6500, LTVLiqBps: 7000}, // N-0 -- CNPY (ARCM Tier 0)
	1: {LTVMaxBps: 5500, LTVLiqBps: 6200}, // N-1 -- ARCM Tier 1 blue-chip
}

// ResolveNasmTier translates an asset's EXISTING ARCM {29} tier
// classification into a NASM eligibility decision, per NASM Spec Section
// 3.1's mandate: NASM does not maintain its own separate asset registry --
// doing so would duplicate ARCM's {29} registry and risk the two drifting
// out of sync (e.g. an asset downgraded at ARCM Tier level, via the SAME
// set_asset_tier admin authority, but never correspondingly restricted from
// NASM minting, since a second, un-synchronized registry would need its own
// separate update). Instead, NASM reads ARCM's tier and applies its own,
// tighter LTV table on top -- one source of truth for asset risk
// classification (set_asset_tier / {29}), two independent consumers of it
// (ARCM lending tiers vs. NASM tiers) applying different LTV values to the
// same classification.
//
// found=false covers THREE distinct upstream cases, all correctly meaning
// "not eligible to back NUSD minting," matching GetAssetTier's own
// found=false-is-ineligible contract:
//   - asset has no {29} registry entry at all (GetAssetTier found=false)
//   - asset is registered at ARCM Tier 2 or Tier 3 (NASM Spec Section 3.1:
//     "too volatile/illiquid for stability-grade backing")
//   - (RWA fractions, NASM Tier N-2, have no ARCM tier number at all --
//     they are excluded structurally, by never appearing in the {29}
//     registry as Tier 0/1 in the first place, not by a special-cased
//     check here)
//
// Callers (mint_nusd) MUST treat found=false as an outright rejection,
// never as a fallback to either NASM tier's parameters.
func ResolveNasmTier(c *Contract, assetID string) (nasmTier uint8, params NasmTierParams, found bool, pErr *PluginError) {
	arcmTier, arcmFound, err := GetAssetTier(c, assetID)
	if err != nil {
		return 0, NasmTierParams{}, false, err
	}
	if !arcmFound {
		return 0, NasmTierParams{}, false, nil
	}
	// arcmTier 0 and 1 map 1:1 to nasmTier 0 (N-0) and 1 (N-1) -- the two
	// enums happen to share the same integer values at this boundary, but
	// this is called out explicitly rather than passed through silently,
	// since ARCM's tier 2/3 do NOT have a NASM equivalent and the mapping
	// is not simply "identity for all inputs."
	if arcmTier != 0 && arcmTier != 1 {
		return 0, NasmTierParams{}, false, nil
	}
	nasmParams, tpFound := nasmTierParamsTable[arcmTier]
	if !tpFound {
		// Unreachable given the arcmTier check above, but guarded
		// explicitly per this project's standard of never assuming a
		// map lookup succeeded silently (matches GetTierParams' own
		// found-bool discipline).
		return 0, NasmTierParams{}, false, nil
	}
	return arcmTier, nasmParams, true, nil
}
