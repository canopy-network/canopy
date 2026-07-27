package contract

import "math/big"

// bad_debt_layer3.go implements Layer 3 of ARCM Section 9.2's four-layer
// bad-debt waterfall: protocol treasury draw-down.
//
// [REVERSED, SAME SESSION] T_fund was originally designed as a SINGLE
// GLOBAL pool shared across Arbor's own lending markets AND NASM/NUSD.
// That decision was reopened and reversed before any caller depended on
// it: a shared pool means a NUSD-side bad-debt event (bad collateral, an
// oracle attack on a NASM-exclusive asset, a mint-side panic) could
// silently drain the Layer 3 protection Arbor lenders were counting on,
// and vice versa -- a hidden risk coupling between two products a user
// would reasonably expect to be independent. This is now TWO separate,
// fully isolated treasuries: Arbor's own (state key {40}) and NASM/NUSD's
// own (state key {41}) -- see PrefixTreasuryArbor/PrefixTreasuryNASM's
// comment in state_keys.go for the full reversal reasoning. This file now
// defines two functions, Layer3DrawDownArbor and Layer3DrawDownNASM,
// rather than one parameterized function -- deliberately, so a caller
// cannot accidentally draw from the wrong pool at the type level, matching
// the safety-over-brevity priority this reversal was made under.
//
// DESIGN DECISION, all-or-nothing gate -- identical shape to Layer2DrawDown
// (bad_debt_layer2.go): ARCM Section 9.2's own Layer 3 row states the
// mechanism literally as "if bad_debt <= T_fund: T_fund -= bad_debt" -- a
// binary gate, not a partial-cover-then-continue step. This choice was
// deliberated explicitly (not merely inherited by default) against a
// partial-cover alternative. Binary gate was chosen: it matches ARCM's
// literal spec text exactly, it does not introduce a new, unaudited
// mechanic into a live bad-debt path, and it keeps this function's
// contract simple enough to reason about under a "maximum security over
// optimization" priority. The accepted cost, stated plainly rather than
// hidden: a shortfall marginally larger than a pool's current balance
// draws nothing at all from that pool, even when it could have absorbed
// nearly all of it. Layer 2 itself is explicitly NOT touched or
// redesigned by this file.
//
// SCOPE, stated explicitly: each of T_fund's two pools (Arbor's own,
// NASM/NUSD's own) is a single GLOBAL balance within its own protocol,
// not market-keyed like R_fund ({18}). Arbor's markets all share ONE
// Arbor-side pool ({40}); NASM's own bad-debt waterfall (not yet built)
// will read/write the separate NASM-side pool ({41}) -- the two pools
// never mix. marketID is accepted as a parameter to both functions below
// -- not for storage (GetTreasuryArbor/GetTreasuryNASM take no marketID,
// see state_accessors.go), but so the caller can attribute which market's
// liquidation drew down the pool, for the EventTreasuryDrawDown emitted on
// a covered draw (see arbor_events.proto). Do not assume marketID scopes
// either balance itself -- it does not; only which pool function is
// called (Arbor vs. NASM) determines that.
//
// UNIT CONTRACT: identical to Layer2DrawDown -- badDebtNative MUST already
// be the caller's native-debt-asset-unit shortfall, already unrecoverable
// after a Layer 2 miss. Neither function below performs any price
// conversion.
//
// ENCODING CONTEXT: DeliverTx-context only (called from
// liquidate_position.go's Layer 2-miss branch, a real transaction that can
// revert) -- uses SetTreasuryArbor/SetTreasuryNASM (the reverting
// EncodeUint128 wrapper), NOT SetTreasuryArborTry/SetTreasuryNASMTry
// (BeginBlock-context; no caller exists for that path as of this file's
// creation). Do not call either function below from BeginBlock code
// without re-deriving which encoding response applies, per Principle 14.
//
// Layer3DrawDownArbor attempts to fully cover badDebtNative from Arbor's
// own T_fund ({40}). Returns covered=true and mutates that pool (write
// persisted) only if the pool's balance >= badDebtNative. Returns
// covered=false and leaves the pool completely untouched (comparison
// happens before any write is attempted) if the pool is insufficient.
// pErr is non-nil only for a genuine state-layer failure (read error,
// encode-overflow revert, write error) -- insufficient coverage is a
// normal, expected return value (covered=false), not a PluginError,
// matching Layer2DrawDown's exact contract: the caller decides what
// happens next (as of this file's creation: liquidate_position.go falls
// through to ApplyLossFactor/Layer 4 on covered=false, exactly as it
// already does on a Layer 2 miss).
func Layer3DrawDownArbor(c *Contract, marketID string, badDebtNative uint64) (covered bool, pErr *PluginError) {
	tFund, found, err := GetTreasuryArbor(c)
	if err != nil {
		return false, err
	}
	if !found {
		tFund = big.NewInt(0)
	}

	badDebtBig := new(big.Int).SetUint64(badDebtNative)
	if tFund.Cmp(badDebtBig) < 0 {
		// Arbor's T_fund insufficient to fully cover -- all-or-nothing gate
		// (see doc comment above). Pool is NOT mutated; no write attempted.
		return false, nil
	}

	newTFund := new(big.Int).Sub(tFund, badDebtBig)
	if wErr := SetTreasuryArbor(c, newTFund); wErr != nil {
		return false, wErr
	}
	return true, nil
}

// Layer3DrawDownNASM is the NASM/NUSD-owned analog of Layer3DrawDownArbor
// above -- identical contract, draws from the separate, isolated NASM pool
// ({41}) instead of Arbor's own ({40}). No current caller exists yet --
// NASM's own bad-debt waterfall is not yet built.
func Layer3DrawDownNASM(c *Contract, marketID string, badDebtNative uint64) (covered bool, pErr *PluginError) {
	tFund, found, err := GetTreasuryNASM(c)
	if err != nil {
		return false, err
	}
	if !found {
		tFund = big.NewInt(0)
	}

	badDebtBig := new(big.Int).SetUint64(badDebtNative)
	if tFund.Cmp(badDebtBig) < 0 {
		// NASM's T_fund insufficient to fully cover -- all-or-nothing gate
		// (see doc comment above). Pool is NOT mutated; no write attempted.
		return false, nil
	}

	newTFund := new(big.Int).Sub(tFund, badDebtBig)
	if wErr := SetTreasuryNASM(c, newTFund); wErr != nil {
		return false, wErr
	}
	return true, nil
}
