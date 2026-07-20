package contract

import "math/big"

// TierParams holds one tier's LTV_max, LTV_liq (both bps), and LIF
// (bps-scaled, 10000 = 1.0x), per ARCM Section 4's LTV table and Section
// 8's LIF table. All four are IMMUTABLE, hardcoded per-tier constants --
// ARCM does not list these as individually governance-adjustable; only
// INCENTIVE_SCALING_FACTOR (the LIF formula's input, Section 15) is
// governance-bounded, and the resulting table for tiers 0-3 is fixed.
type TierParams struct {
LTVMaxBps uint64
LTVLiqBps uint64
LIFBps    uint64
}

// Tier 4 (Blacklisted) is deliberately absent -- see state_keys.go's
// PrefixAssetTier comment: an asset with no registry entry is treated as
// ineligible, never looked up here.
var tierParamsTable = map[uint8]TierParams{
0: {LTVMaxBps: 8000, LTVLiqBps: 8500, LIFBps: 10300}, // Tier 0 -- CNPY
1: {LTVMaxBps: 7500, LTVLiqBps: 8200, LIFBps: 10360}, // Tier 1 -- Blue-chip
2: {LTVMaxBps: 6500, LTVLiqBps: 7500, LIFBps: 10500}, // Tier 2 -- Standard
3: {LTVMaxBps: 4000, LTVLiqBps: 5500, LIFBps: 10900}, // Tier 3 -- Restricted
}

// GetTierParams looks up a tier's LTV/LIF parameters. found=false means the
// tier byte was not 0-3 -- callers MUST treat this as "not eligible to
// borrow against," never as a zero-value TierParams that would silently
// permit borrowing at 0% LTV.
func GetTierParams(tier uint8) (params TierParams, found bool) {
p, ok := tierParamsTable[tier]
return p, ok
}

// GetAssetTier reads and decodes an asset's tier from the {29} registry.
// found=false means the asset has no registry entry -- per
// PrefixAssetTier's own comment (state_keys.go), this means "not eligible
// as collateral/debt asset," NOT "default to Tier 0."
func GetAssetTier(c *Contract, assetID string) (tier uint8, found bool, pErr *PluginError) {
readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{{QueryId: 0, Key: KeyForAssetTier(assetID)}},
})
if err != nil {
return 0, false, err
}
if readResp.Error != nil {
return 0, false, readResp.Error
}
raw := entryValue(readResp, 0)
if len(raw) == 0 {
return 0, false, nil
}
return DecodeAssetTierRecord(raw), true, nil
}

// MaxBorrowQuantity computes ARCM Section 4's max-borrow formula:
// floor((Q_c * P_c * LTV_max_bps) / (P_d * 10000)). math/big throughout
// (ARCM Section 17.4) -- Q_c * P_c alone can exceed uint64 range for large
// collateral quantities against high-priced assets.
func MaxBorrowQuantity(collateralQty, collateralPrice, ltvMaxBps, debtPrice uint64) uint64 {
if debtPrice == 0 {
return 0
}
numerator := new(big.Int).Mul(new(big.Int).SetUint64(collateralQty), new(big.Int).SetUint64(collateralPrice))
numerator.Mul(numerator, new(big.Int).SetUint64(ltvMaxBps))
denominator := new(big.Int).Mul(new(big.Int).SetUint64(debtPrice), big.NewInt(10000))
result := new(big.Int).Div(numerator, denominator)
if result.BitLen() > 64 {
// Saturate rather than wrap -- caller compares this against a real
// borrow request, so an implausibly large max-borrow capped to
// MaxUint64 still correctly ALLOWS any realistic request; it never
// masks a rejection the way silent wrapping could.
return ^uint64(0)
}
return result.Uint64()
}
