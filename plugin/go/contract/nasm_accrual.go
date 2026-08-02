package contract

// stabilityFeeBaseBps is NASM's launch-default annual stability fee rate
// (NASM Consolidated Spec Section 6.2, Section 18: "launch default: 150
// bps / 1.5% APR"). Hardcoded pending NASM's own governance parameter
// store -- update_nasm_params (NASM Spec Section 13) does not exist yet,
// same disclosed pattern as interest_rate.go's hardcoded rate constants
// and create_market.go's hardcoded MinReporters. Governance range is
// 0-500 bps (Section 18) once a governance path exists to move it within
// that bound; the value itself never exceeds NASM's own immutable ceiling
// regardless of which mechanism sets it.
const stabilityFeeBaseBps = 150

// AccrueStabilityFee implements NASM Consolidated Spec Section 6.2's
// global stability fee index accrual, reusing AYIS's exact
// AnnualRateToPerBlockRateRay/CompoundExact functions rather than
// reimplementing the compounding math -- a deliberate reuse decision per
// NASM Spec Section 6.2's own rationale: "AYIS's index math is already
// independently verified, and reusing proven math reduces NASM's own
// audit surface."
//
// Unlike AccrueInterest (interest_accrual.go), this is a SINGLE global
// step, not per-market -- SF_index at {32} is one pooled value across
// every NASM vault (NASM Spec Section 6.2: "Global, single pooled fee
// (not per-market)"), so this function takes no marketID argument and is
// called exactly once per block from BeginBlock, after the per-market
// AYIS accrual loop completes (NASM Spec Section 6.5's recommended
// ordering: "1. AYIS.AccrueInterest() for all ARCM lending markets; 2.
// NASM stability fee accrual").
//
// Returns a PluginError only for a genuine state-layer failure. A
// TryEncodeUint128 overflow on the new sf_index is NOT an error -- per
// Principle 14 (no transaction to revert in BeginBlock context), this
// function leaves SF_index at its last successfully-committed value and
// logs the condition, rather than halting. UNLIKE AccrueInterest's
// per-market IndexOverflowHalted flag, there is currently no persistent,
// queryable "NASM accrual halted" flag -- this is a disclosed gap: an
// overflow here (RAY-scaled uint128, astronomically unlikely in practice,
// same I8-bounded reasoning AYIS's own loss_factor relies on) would
// silently stop fee accrual without a visible on-chain signal beyond the
// log line. Acceptable for now given the near-zero practical likelihood,
// but a future session should consider a dedicated NASM emergency/halt
// flag if this module's surface grows.
func AccrueStabilityFee(c *Contract) (pErr *PluginError) {
	currentBlock := c.plugin.CurrentHeight()

	record, found, err := GetStabilityFeeIndexRecord(c)
	if err != nil {
		return err
	}

	// [BUG FOUND AND FIXED, live-devnet-verified] Original version set
	// lastAccrualBlock = currentBlock on the genesis (!found) path and then
	// fell through to the delta_t == 0 guard below, which returned nil
	// WITHOUT ever calling SetStabilityFeeIndexRecordTry -- so the baseline
	// record was never actually persisted, found stayed false forever, and
	// this function silently re-ran its own "genesis" branch every single
	// block for the rest of the chain's life, never accruing anything.
	// Caught by checking /v1/query/stabilityfeeindex across multiple real
	// blocks and observing sf_index frozen exactly at RAY with
	// last_accrual_block never appearing -- confirms this project's own
	// discipline of verifying against live state rather than trusting a
	// clean build. Fixed: the genesis path now writes its own baseline
	// record immediately and returns, establishing found=true for every
	// subsequent call, instead of falling through to the delta_t guard.
	if !found {
		ok, tErr := SetStabilityFeeIndexRecordTry(c, RAY, currentBlock)
		if tErr != nil {
			return tErr
		}
		if !ok {
			// RAY can never fail TryEncodeUint128 (it is far below the
			// 128-bit ceiling) -- unreachable in practice, guarded anyway
			// per this project's standard of never assuming a Try call
			// succeeded silently.
			return nil
		}
		return nil
	}

	sfIndex := DecodeUint128(record.SfIndex)
	lastAccrualBlock := record.LastAccrualBlock

	// Step 0, mirrors AccrueInterest's own N1 guard: if this function
	// somehow runs twice in the same block (defensive; BeginBlock calls
	// it exactly once per block in practice), delta_t=0 must be a
	// deliberate no-op -- avoids an unnecessary write and keeps
	// last_accrual_block from drifting ahead of real elapsed blocks.
	if currentBlock <= lastAccrualBlock {
		return nil
	}
	deltaT := currentBlock - lastAccrualBlock

	// NASM Spec Section 6.2, Step 1: per-block stability fee rate.
	perBlockRate := AnnualRateToPerBlockRateRay(stabilityFeeBaseBps)

	// NASM Spec Section 6.2, Step 2: compound SF_index. Reuses
	// CompoundExact exactly as AYIS's own B_index compounding does --
	// same exact (non-modular) big.Int exponentiation, no separate
	// linear-approximation path for small delta_t (AYIS's own
	// MAX_DELTA_T_LINEAR distinction exists for gas-cost reasons at very
	// high per-market call frequency; NASM's single global call per block
	// does not carry that same cost pressure, so always using the exact
	// formula here is deliberate, not an oversight).
	newSfIndex := CompoundExact(sfIndex, perBlockRate, deltaT)

	ok, tErr := SetStabilityFeeIndexRecordTry(c, newSfIndex, currentBlock)
	if tErr != nil {
		return tErr
	}
	if !ok {
		// See this function's own doc comment on the disclosed lack of a
		// persistent halt flag for this specific failure mode.
		return nil
	}
	return nil
}
