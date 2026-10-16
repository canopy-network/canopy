package contract

import "math/big"

// nasm_tier_backing.go implements NASM Consolidated Spec Section 3.3's
// per-tier mint concentration cap: no single NASM tier (N-0 or N-1) may
// back more than Max_tier_share_bps (7000 = 70%, governance-bounded
// 5000-8500) of total NUSD supply. Originally deferred by mint_nusd.go's
// own disclosed-gap comment pending burn_nusd's existence, so the
// accumulator could be built correctly in sync on mint AND burn from the
// start rather than shipping an increment-only value that silently drifts
// -- burn_nusd (and liquidate_nasm_vault) now both exist, closing that
// dependency.

// GetNasmTierBacking reads the single {36} NasmTierBacking record.
// found=false is the normal, expected steady state before any NASM vault
// has ever been minted -- matches GetNusdSupply's own found=false-is-
// normal contract (state_accessors.go), not GetMarket's found=false-is-an-
// error contract. Callers MUST treat a zero-value NasmTierBacking{} as the
// correct default in that case.
func GetNasmTierBacking(c *Contract) (backing *NasmTierBacking, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForNasmTierBacking()},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if readResp.Error != nil {
		return nil, false, readResp.Error
	}
	raw := entryValue(readResp, 0)
	if len(raw) == 0 {
		return nil, false, nil
	}
	b := &NasmTierBacking{}
	if uErr := Unmarshal(raw, b); uErr != nil {
		return nil, false, uErr
	}
	return b, true, nil
}

// MaxTierShareBps is NASM Spec Section 3.3/18's launch-default per-tier
// mint concentration cap (70%, governance range 50%-85%). Hardcoded
// pending NASM's own governance parameter store, same disclosed pattern as
// nasm_tier.go's LTV/LIF constants and nasm_accrual.go's
// stabilityFeeBaseBps (update_nasm_params, NASM Spec Section 13, does not
// exist yet).
const MaxTierShareBps = 7000

// applyTierBackingDelta is the SINGLE mandatory write path for every
// NasmTierBacking mutation (mirrors debt_delta.go's applyDebtDelta and its
// own centralization rationale exactly: a guard written beside this
// function instead of inside it can be silently bypassed by a future
// caller; a guard inside it cannot). mint_nusd (positive delta), burn_nusd
// and liquidate_nasm_vault (negative delta) all call through here.
//
// Takes *NasmTierBacking by pointer and mutates the field for the given
// nasmTier (0 or 1) in place; caller is responsible for the actual state
// write, matching GetNasmTierBacking's read/mutate/write pattern rather
// than this function owning its own state I/O -- same division of
// responsibility as applyDebtDelta/SaveMarket.
//
// delta is a signed int64: SafeInt64FromUint64 (debt_delta.go) is reused
// directly rather than duplicated, since the identical MaxInt64-wraparound
// risk applies here.
//
// On an oversized decrease, this dust-clamps to zero rather than
// underflowing -- mirrors applyDebtDelta's identical clampedFrom
// mechanism.
func applyTierBackingDelta(backing *NasmTierBacking, nasmTier uint8, delta int64) (clampedFrom uint64, pErr *PluginError) {
	var current uint64
	switch nasmTier {
	case 0:
		current = backing.TierN0Backing
	case 1:
		current = backing.TierN1Backing
	default:
		return 0, ErrInvalidNasmTier(nasmTier)
	}

	var newValue uint64
	if delta > 0 {
		increase := uint64(delta)
		if increase > (^uint64(0) - current) {
			return 0, ErrNasmTierBackingOverflow(nasmTier, current, increase)
		}
		newValue = current + increase
	} else if delta < 0 {
		decrease := uint64(-delta)
		if decrease >= current {
			clampedFrom = current
			newValue = 0
		} else {
			newValue = current - decrease
		}
	} else {
		newValue = current
	}

	switch nasmTier {
	case 0:
		backing.TierN0Backing = newValue
	case 1:
		backing.TierN1Backing = newValue
	}
	return clampedFrom, nil
}

// CheckTierConcentrationCap implements NASM Spec Section 3.3's mint-time
// enforcement: rejects a mint if, after adding newMintBackingValue to the
// resolved tier's running total, that tier's share of the NEW total NUSD
// supply (existing total_supply + this mint's own newMintBackingValue)
// would exceed MaxTierShareBps. Read-only -- callers apply the actual
// accumulator increase separately via applyTierBackingDelta, only after
// this check (and every other mint_nusd guard) has passed.
//
// Uses big.Int throughout for the same reason ARCM/AYIS mandate it
// everywhere else in this codebase (Section 14's math/big requirement).
func CheckTierConcentrationCap(c *Contract, nasmTier uint8, newMintBackingValue uint64) *PluginError {
	backing, found, err := GetNasmTierBacking(c)
	if err != nil {
		return err
	}
	if !found {
		backing = &NasmTierBacking{}
	}

	var tierTotal uint64
	switch nasmTier {
	case 0:
		tierTotal = backing.TierN0Backing
	case 1:
		tierTotal = backing.TierN1Backing
	default:
		return ErrInvalidNasmTier(nasmTier)
	}

	supply, sFound, sErr := GetNusdSupply(c)
	if sErr != nil {
		return sErr
	}
	var existingTotalSupply uint64
	if sFound && supply != nil {
		existingTotalSupply = supply.TotalSupply
	}

	newTierTotal := new(big.Int).SetUint64(tierTotal)
	newTierTotal.Add(newTierTotal, new(big.Int).SetUint64(newMintBackingValue))

	newGrandTotal := new(big.Int).SetUint64(existingTotalSupply)
	newGrandTotal.Add(newGrandTotal, new(big.Int).SetUint64(newMintBackingValue))

	// [CONCENTRATION CAP FIX] Prior guard checked newGrandTotal.Sign() == 0
	// -- i.e. "the grand total AFTER this mint is zero" -- which can only
	// ever be true for a zero-amount mint (newMintBackingValue == 0), a
	// case CheckMessageMintNusd already rejects earlier as ErrInvalidAmount
	// before this function is ever reached. It therefore never actually
	// fired for any real mint.
	//
	// The real gap: system's first-ever mint (existingTotalSupply == 0,
	// newMintBackingValue > 0) is mathematically guaranteed to make
	// newTierTotal == newGrandTotal (this mint is the only backing for its
	// own tier AND the only supply that would exist), i.e. exactly 100% of
	// a brand-new total -- which always exceeds MaxTierShareBps(70%)
	// regardless of mint size, permanently blocking every first mint on
	// any fresh deployment. NASM Spec Section 3.3's diversification intent
	// is about balance across tiers as the backing portfolio grows -- it
	// has nothing to bind against yet when no supply exists at all, so
	// there is nothing to be disproportionate relative to.
	//
	// Scoped to SYSTEM-wide existingTotalSupply, not per-tier: a mint into
	// an already-nonzero total (e.g. Tier N-1's first mint when Tier N-0
	// already carries real backing) is a genuine concentration event and
	// must still be checked normally below -- only the very first unit of
	// NUSD supply, system-wide, is exempt, and only once.
	if existingTotalSupply == 0 {
		return nil
	}

	lhs := new(big.Int).Mul(newTierTotal, big.NewInt(10_000))
	rhs := new(big.Int).Mul(big.NewInt(int64(MaxTierShareBps)), newGrandTotal)

	if lhs.Cmp(rhs) > 0 {
		return ErrNasmTierConcentrationCapExceeded(nasmTier, newTierTotal.String(), newGrandTotal.String(), MaxTierShareBps)
	}
	return nil
}
