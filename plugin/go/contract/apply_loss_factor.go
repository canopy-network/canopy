package contract

import (
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"
)

// apply_loss_factor.go implements AYIS Section 5.4.3: Layer 4 of ARCM
// Section 9.2's bad-debt waterfall (lender socialization via loss_factor).
//
// [STALE COMMENT CORRECTED] This function IS wired in: liquidate_position.go
// calls it directly on a Layer 2 miss (see that file's Layer 2/4 fallthrough,
// skipping the not-yet-built Layer 3 treasury). The claim below that no
// caller exists predates that wiring and was never updated -- caught while
// investigating a real on-chain discrepancy (a liquidation that drove
// loss_factor to 0 without market.status showing INSOLVENT) that this stale
// comment nearly caused to be mis-traced. ProcessLossFactorQueue (the
// BeginBlock caller) is still unbuilt as of this writing.
//
// CALLER UNIT CONTRACT (per bad_debt_layer2.go's own documented contract):
// Layer2DrawDown is an all-or-nothing gate -- on covered == false, R_fund is
// left completely untouched and the FULL original badDebtNative amount is
// still outstanding (never a partial residual, since Layer 2 either covers
// everything or nothing). A future caller passing bad_debt into this
// function after a Layer 2 miss MUST pass that same full, unmodified
// badDebtNative value -- not a partial amount -- matching this function's
// own bad_debt parameter's meaning per AYIS Section 5.4.3.
//
// EVENT EMISSION, RETURNED NOT EMITTED (same root gap as
// market_insolvency.go's DecrementLayer4Pending TODO, resolved here by
// signature rather than punted): every existing event-emission call site in
// this codebase (liquidate_position.go, repay.go, withdraw.go) is
// DeliverTx-context, building a local `var events []*Event` slice returned
// as part of a PluginDeliverResponse. This function may be called from
// EITHER a future DeliverTx-context caller (a synchronous liquidation Layer
// 4 fallthrough) OR a future BeginBlock-context caller
// (ProcessLossFactorQueue) -- neither of which is built yet, and the latter
// has NO established event-emission path in this codebase at all. Rather
// than guess at a BeginBlock emission mechanism that hasn't been verified to
// exist (matching DecrementLayer4Pending's own documented reasoning),
// ApplyLossFactor returns its event as a value instead of emitting it
// directly. The caller decides what to do with it: append to its own
// `events` slice (DeliverTx) or hold it until a real BeginBlock event path
// exists (BeginBlock). This keeps the function usable from both future
// contexts without needing to change its signature once that path exists.
// [FIXED] Signature changed from (c, marketID) to (market *Market),
// matching the same in-place-mutation fix as SetMarketInsolvent and
// DecrementLayer4Pending (market_insolvency.go). This function's own
// idempotency check previously called GetMarketStatus(c, marketID) -- a
// THIRD independent read of the same market within one liquidation's call
// graph, on top of liquidate_position.go's own outer read. Now reads
// market.Status directly off the caller's already-in-memory struct instead.
// Caller (liquidate_position.go) owns the single SaveMarket() write at the
// end of its own function, exactly as it already does for TotalBorrowed via
// applyDebtDelta().
func ApplyLossFactor(c *Contract, market *Market, marketID string, badDebt uint64) (event *Event, pErr *PluginError) {
	// [AYIS Section 5.4.3, K3] Idempotency guard. A queued application
	// (or, in the future, a synchronous one) against a market that is
	// already Insolvent must not re-run the exhaustion side effects --
	// it only needs to decrement the Layer 4 pending counters and record
	// that this particular queued/synchronous event was consumed.
	if market.Status == MarketStatus_INSOLVENT {
		// [FIX, session finding] TotalBorrowed write-back gap: this branch
		// (like the other two below) previously never decremented
		// market.TotalBorrowed by badDebt -- the caller's own Step 7
		// (liquidate_position.go) only decrements by msg.RepayAmount (the
		// COVERED portion actually repaid), never by the WRITTEN-OFF
		// remainder. That remainder is exactly this function's own badDebt
		// parameter. Left uncorrected, every bad-debt write-off permanently
		// overstates market.TotalBorrowed, inflating ComputeUtilizationBps
		// (interest_accrual.go) and therefore the borrow rate every
		// remaining borrower pays, forever, compounding with each
		// subsequent write-off. This branch specifically: even though the
		// market is already Insolvent (loss_factor already at 0, no more
		// lender-claim value to haircut), THIS liquidation's own bad_debt
		// portion is just as real and just as uncounted as in the other two
		// branches -- see liquidate_position.go's own comment confirming
		// this K3 path "changes no balance at all" as of before this fix,
		// which was the actual gap, not a correct-by-design no-op.
		badDebtI64, safeOk := SafeInt64FromUint64(badDebt)
		if !safeOk {
			return nil, ErrInt64CastOverflow("applyLossFactor.badDebt", badDebt)
		}
		if _, dbErr := applyDebtDelta(market, marketID, -badDebtI64); dbErr != nil {
			return nil, dbErr
		}
		if dErr := DecrementLayer4Pending(market, badDebt); dErr != nil {
			return nil, dErr
		}
		payload := &EventLossFactorAppliedToAlreadyInsolventMarket{
			MarketId: marketID,
			BadDebt:  badDebt,
		}
		anyMsg, aErr := anypb.New(payload)
		if aErr != nil {
			return nil, ErrMarshal(aErr)
		}
		return &Event{
			EventType: "loss_factor_applied_to_already_insolvent_market",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		}, nil
	}

	// [AYIS Section 5.4.3] total_supplied_equiv is the O(1) exact
	// aggregate this bad_debt amount is compared against to decide
	// haircut vs. total exhaustion.
	totalSuppliedEquiv, tErr := SumLenderBalancesInMarket(c, marketID)
	if tErr != nil {
		return nil, tErr
	}

	currentLossFactor, found, lErr := GetLossFactor(c, marketID)
	if lErr != nil {
		return nil, lErr
	}
	if !found {
		// Unreachable in practice: create_market always initializes {27}
		// to RAY (AYIS Section 4.5). Guarded explicitly rather than
		// assumed, matching SumLenderBalancesInMarket's own discipline.
		return nil, ErrMarketNotFound(marketID)
	}

	if badDebt >= totalSuppliedEquiv {
		// Total exhaustion: loss_factor -> exactly 0, market transitions
		// to Insolvent. [ARCM v3.11.1 Section 9.3b Rule 1] This
		// transition MUST NOT be gated by index_overflow_halted in
		// either direction -- SetMarketInsolvent() has no dependency on
		// that flag at all, satisfying Rule 1 by construction.
		if zErr := SetLossFactor(c, marketID, big.NewInt(0)); zErr != nil {
			return nil, zErr
		}
		SetMarketInsolvent(market)
		// [FIX, session finding] See K3 branch's own comment above for the
		// full rationale -- same TotalBorrowed write-back, this branch's
		// own badDebt portion.
		badDebtI64, safeOk := SafeInt64FromUint64(badDebt)
		if !safeOk {
			return nil, ErrInt64CastOverflow("applyLossFactor.badDebt", badDebt)
		}
		if _, dbErr := applyDebtDelta(market, marketID, -badDebtI64); dbErr != nil {
			return nil, dbErr
		}
		if dErr := DecrementLayer4Pending(market, badDebt); dErr != nil {
			return nil, dErr
		}
		payload := &EventLossFactorExhausted{
			MarketId:           marketID,
			BadDebt:            badDebt,
			TotalSuppliedEquiv: totalSuppliedEquiv,
		}
		anyMsg, aErr := anypb.New(payload)
		if aErr != nil {
			return nil, ErrMarshal(aErr)
		}
		return &Event{
			EventType: "loss_factor_exhausted",
			Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
		}, nil
	}

	// Partial haircut: new_loss_factor = current_loss_factor *
	// (total_supplied_equiv - bad_debt) / total_supplied_equiv, floored
	// (AYIS Section 10.2: loss_factor haircut rounds floor, dust favors
	// remaining lenders).
	remaining := new(big.Int).SetUint64(totalSuppliedEquiv)
	remaining.Sub(remaining, new(big.Int).SetUint64(badDebt))
	newLossFactor := new(big.Int).Mul(currentLossFactor, remaining)
	newLossFactor.Div(newLossFactor, new(big.Int).SetUint64(totalSuppliedEquiv))

	if wErr := SetLossFactor(c, marketID, newLossFactor); wErr != nil {
		return nil, wErr
	}
	// [FIX, session finding] See K3 branch's own comment above for the full
	// rationale -- same TotalBorrowed write-back, this branch's own badDebt
	// portion.
	badDebtI64, safeOk := SafeInt64FromUint64(badDebt)
	if !safeOk {
		return nil, ErrInt64CastOverflow("applyLossFactor.badDebt", badDebt)
	}
	if _, dbErr := applyDebtDelta(market, marketID, -badDebtI64); dbErr != nil {
		return nil, dbErr
	}
	if dErr := DecrementLayer4Pending(market, badDebt); dErr != nil {
		return nil, dErr
	}
	payload := &EventBadDebtSocialization{
		MarketId:      marketID,
		BadDebt:       badDebt,
		NewLossFactor: newLossFactor.String(),
	}
	anyMsg, aErr := anypb.New(payload)
	if aErr != nil {
		return nil, ErrMarshal(aErr)
	}
	return &Event{
		EventType: "bad_debt_socialization",
		Msg:       &Event_Custom{Custom: &EventCustom{Msg: anyMsg}},
	}, nil
}
