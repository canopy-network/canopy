package contract

import "math/big"

// ScaledNusdDebt computes a NASM vault's current owed NUSD debt, scaled by
// the stability fee index ratio between vault open and now -- NASM
// Consolidated Spec Section 6.2:
//
// ScaledNusdDebt(vault) = vault.nusd_principal * SF_index(t) / vault.sf_index_at_open
//
// Mirrors ScaledDebt's exact structure (scaled_debt.go) -- same zero-
// principal short-circuit, same defensive zero-index guard, same ceiling
// division ((numerator + divisor - 1) / divisor, protocol-favouring per
// NASM Spec Section 14.1: "vault owner owes slightly more"), same
// cast-safety guard on the final uint64 conversion. Callers (burn_nusd,
// liquidate_nusd_vault once it exists) MUST call this rather than reading
// vault.NusdPrincipal directly as "current debt" -- matches ScaledDebt's
// own mandatory-use contract (AYIS Section 6, ARCM Section 2.2).
func ScaledNusdDebt(vault *NasmVault, sfIndexNow *big.Int) (uint64, *PluginError) {
	if vault.NusdPrincipal == 0 {
		return 0, nil
	}
	sfIndexAtOpen := DecodeUint128(vault.SfIndexAtOpen)

	// Defensive-only guard: a zero sf_index_at_open should never occur in
	// practice -- mint_nusd always writes this field from a live,
	// just-read SF_index at vault-open time (defaulting to RAY if the
	// global index had never been initialized, never to zero). This only
	// protects ScaledNusdDebt() itself from a division-by-zero if a vault
	// record were ever corrupted or malformed; it is not an expected code
	// path, mirroring ScaledDebt's own identical guard.
	if sfIndexAtOpen.Sign() == 0 {
		sfIndexAtOpen = big.NewInt(1)
	}

	numerator := new(big.Int).Mul(new(big.Int).SetUint64(vault.NusdPrincipal), sfIndexNow)
	// Ceiling division: (numerator + divisor - 1) / divisor
	numerator.Add(numerator, sfIndexAtOpen)
	numerator.Sub(numerator, big.NewInt(1))
	numerator.Div(numerator, sfIndexAtOpen)

	if numerator.BitLen() > 64 {
		return 0, ErrScaledNusdDebtOverflow(vault.VaultId, vault.NusdPrincipal, numerator.String())
	}
	return numerator.Uint64(), nil
}
