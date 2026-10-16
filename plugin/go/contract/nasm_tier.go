package contract

import "math/big"

// NasmTierParams holds one NASM tier's LTV_n_max, LTV_n_liq, and LIF (all
// bps), per NASM Consolidated Spec Section 3.1's collateral tier table and
// Section 7's liquidation integration. Mirrors ARCM's TierParams shape
// (tier_params.go) exactly, including carrying its own LIFBps field --
// see nasmTierParamsTable's own comment for why NASM's LIFBps values are
// deliberately identical to ARCM's, not independently derived.
//
// Values are IMMUTABLE hardcoded launch-config constants, matching ARCM's
// tierParamsTable precedent exactly -- NASM Spec Section 18's parameter
// table lists LTV_n_max/LTV_n_liq as governance-bounded (Tier N-0:
// 50%-70%/55%-75%, etc.), but no governance-write path exists yet for
// NASM parameters (update_nasm_params, NASM Spec Section 13, is not yet
// implemented). This mirrors interest_rate.go's disclosed
// hardcoded-pending-governance-store pattern.
type NasmTierParams struct {
	LTVMaxBps uint64
	LTVLiqBps uint64
	LIFBps    uint64
}

// nasmTierParamsTable maps a NASM tier (0=N-0, 1=N-1) to its full
// LTV/LIF parameters. NASM Spec Section 3.1: N-2 (RWA fractions) and N-3
// (ARCM Tier 2/3 assets) are explicitly "Not eligible" for NUSD
// mint-backing at all -- there is no table entry for them, deliberately,
// matching tierParamsTable's own absent-Tier-4 precedent in
// tier_params.go.
//
// LIFBps DERIVATION -- worked through carefully, not guessed:
//
// ARCM's own LIFBps values (tier_params.go: 10300/10360/10500/10900) were
// reverse-derived to follow an exact formula across all four tiers:
// LIF_bps = 10000 + 0.2 * (10000 - LTVLiqBps) -- i.e. a flat 20%
// INCENTIVE_SCALING_FACTOR applied to the gap between LTVLiqBps and 100%
// (verified numerically, ratio == 0.2 exactly for all four ARCM tiers).
//
// Applying that SAME 20% factor mechanically to NASM's own, lower
// LTVLiqBps values produces LARGER LIF (N-0: 10600, N-1: 10760 -- the
// gap-to-100% term grows as LTVLiqBps shrinks) -- which contradicts NASM
// Spec Section 7's plain text ("NASM tiers are inherently safer" implies
// LESS liquidator compensation should be needed, not more).
//
// The resolution: LIF compensates a liquidator for the EXECUTION risk of
// selling the specific SEIZED ASSET (price slippage, volatility, market
// depth during the liquidation window) -- a property of the asset itself,
// not of the borrower's LTV policy. NASM's tighter LTV is an unrelated,
// ADDITIONAL safety buffer for stablecoin-grade backing (Section 16.1's
// tail-risk rationale) -- it does not mean CNPY or blue-chip assets
// themselves became harder to liquidate. Since ResolveNasmTier maps ARCM
// Tier 0 -> NASM N-0 and ARCM Tier 1 -> NASM N-1 as the literal SAME
// underlying assets (not merely similar ones), NASM's LIFBps values here
// are deliberately identical to ARCM's own already-audited values for
// those same tiers, reusing proven numbers for the identical asset risk
// profile rather than inventing a new, unverified NASM-specific
// incentive-scaling-factor.
var nasmTierParamsTable = map[uint8]NasmTierParams{
	0: {LTVMaxBps: 6500, LTVLiqBps: 7000, LIFBps: 10300}, // N-0 -- CNPY (ARCM Tier 0, LIFBps reused as-is)
	1: {LTVMaxBps: 5500, LTVLiqBps: 6200, LIFBps: 10360}, // N-1 -- ARCM Tier 1 blue-chip (LIFBps reused as-is)
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
// Callers (mint_nusd, liquidate_nusd_vault) MUST treat found=false as an
// outright rejection, never as a fallback to either NASM tier's
// parameters.
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

// CalcMaxMintableNusd computes the maximum NUSD mintable at exactly
// HF_n == 1.0 (the liquidation boundary) for a given collateral asset and
// quantity, at the current oracle price. This is the exact same formula
// mint_nusd.go's DeliverMessageMintNusd already computes inline as
// maxNusdAtHF1 to build ErrNasmHealthFactorTooLow's error message -- moved
// here as its own named, reusable function rather than left duplicated,
// matching this codebase's established discipline (see
// applyTierBackingDelta's doc comment on why a single mandatory
// implementation is preferred over parallel copies that can silently
// drift). mint_nusd.go now calls this directly instead of inlining the
// same big.Int arithmetic a second time.
//
// Read-only: no state writes, does not itself enforce anything -- callers
// decide what to do with the returned ceiling (mint_nusd.go rejects a
// request that meets or exceeds it; the read-only RPC route added
// alongside this function surfaces it to the frontend as a live preview
// so a request that would fail this check never gets submitted in the
// first place).
//
// A previewed ceiling can still shift slightly by the time an actual mint
// transaction lands, if the oracle price updates in between -- same
// inherent lag as every other live, oracle-priced figure in this
// codebase (interest accrual on debt is the closest precedent). Callers
// surfacing this as a UI hint should treat it as an estimate, not a
// guarantee.
func CalcMaxMintableNusd(c *Contract, collateralAssetID string, collateralQuantity uint64) (maxNusd uint64, nasmTier uint8, found bool, pErr *PluginError) {
	nasmTier, nasmParams, tierFound, tErr := ResolveNasmTier(c, collateralAssetID)
	if tErr != nil {
		return 0, 0, false, tErr
	}
	if !tierFound {
		return 0, 0, false, nil
	}

	collateralPrice, priceFound, pErr := ResolvePrice(c, collateralAssetID)
	if pErr != nil {
		return 0, 0, false, pErr
	}
	if !priceFound {
		return 0, 0, false, nil
	}

	// Identical scaling as mint_nusd.go's own inline comment explains in
	// full: V_nc (collateral USD value, x1e8 from the oracle price) times
	// LTV_n_liq_bps, divided by (10000 * 100) to land on NUSD's 1e6
	// precision. Floor rounding throughout (protocol-favouring, NASM Spec
	// Section 14.1).
	numerator := new(big.Int).SetUint64(collateralQuantity)
	numerator.Mul(numerator, new(big.Int).SetUint64(collateralPrice))
	numerator.Mul(numerator, new(big.Int).SetUint64(nasmParams.LTVLiqBps))
	denominator := big.NewInt(10000 * 100)
	maxNusdBig := new(big.Int).Div(numerator, denominator)

	// [CAST-SAFETY GUARD] Same discipline as every other big.Int-to-uint64
	// conversion in this codebase (see deposit.go/withdraw.go's identical
	// BitLen() > 64 guards) -- proves the result fits before using it.
	// Practically unreachable at any realistic collateral quantity or
	// price, but guarded explicitly rather than assumed.
	if maxNusdBig.BitLen() > 64 {
		return 0, nasmTier, true, ErrShareOverflow(collateralAssetID, collateralQuantity, maxNusdBig.String())
	}

	return maxNusdBig.Uint64(), nasmTier, true, nil
}
