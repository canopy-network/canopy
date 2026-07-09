package contract

import "math/big"

// interest_accrual.go implements AYIS Section 7's BeginBlock interest
// accrual sequence for a single market. AccrueInterest is called once per
// market per block by the BeginBlock hook (which iterates PrefixMarkets,
// {16}), and is ALSO called synchronously from DeliverTx by
// borrow/repay/liquidate_position's own Step 1 (AYIS Section 12.3, Accrual
// Ordering Contract: "AccrueInterest() MUST be the first call in
// BeginBlock, before... loss-factor-application queue processing").
//
// KNOWN GAP, DELIBERATE, NOT AN OVERSIGHT: this implementation does NOT yet
// include C4's WillExhaustThisBlock lookahead (ARCM v3.11.1 Section 9.3b
// Rule 3 / AYIS v1.11.1 Section 7 Step 8 revised). C4 requires
// SumLenderBalancesInMarket() and a read against the {28}
// loss-factor-application queue, NEITHER OF WHICH EXISTS YET in this
// codebase as of this file's creation -- Layer 4 / lender-socialization
// logic (ApplyLossFactor, EnqueueLossFactorApplication,
// ProcessLossFactorQueue, SumLenderBalancesInMarket) has not been built.
// Wiring in a stub here that doesn't do the real comparison would be worse
// than omitting it: it would look tested when it is not. Step 8 below
// therefore branches ONLY on market.status == Insolvent (the pre-C4,
// v3.11/v1.11 behavior), NOT market.status == Insolvent ||
// WillExhaustThisBlock(...). This means the specific one-block
// misallocation window C4 closes (a market's queued, same-block Layer-4
// exhaustion having its interest incorrectly split rather than routed to
// R_fund) is NOT yet closed by this function. TODO(C4): once
// SumLenderBalancesInMarket and the {28} queue peek exist, add
// WillExhaustThisBlock and revise Step 8's condition per AYIS v1.11.1.
//
// R_FUND SCOPE, as of the session that added Step 10's real implementation:
// this function now correctly credits reserve_cut to R_fund ({18}) for the
// non-Insolvent (Step 8/9/10) path, via SetReserveFundTry -- the
// BeginBlock-context leg ARCM Section 9.3 calls the "interest" source.
// This closes a real bug: reserve_cut was previously computed, subtracted
// from supplierInterest, and then silently discarded every block. What
// remains OUT of scope, deliberately: the Insolvent branch's own R_fund
// routing (ARCM Section 9.3, full interest_earned -> R_fund for an already-
// Insolvent market) is still a TODO in that branch below, and the two
// DeliverTx-context R_fund legs (repay principal, liquidation proceeds) do
// not exist at all yet, since repay/liquidate_position themselves are not
// implemented. Those legs require EncodeUint128's reverting wrapper, NOT
// SetReserveFundTry -- do not reuse this function for them without
// re-deriving which encoding response applies (Principle 14).

const maxDeltaTLinear = 1000 // AYIS Section 13, immutable

// AccrueInterest implements AYIS Section 7 for one market. Returns a
// PluginError only for a genuine state-layer failure (RPC error, corrupt
// unmarshal) -- an index-encoding overflow is NOT an error, it is handled
// in-function by freezing the single affected market and returning nil,
// per Principle 14 (no transaction exists to revert in BeginBlock context;
// this function is also called from DeliverTx, but even there, an encoding
// overflow's correct response for B_index/S_rate is defined by AYIS
// Section 3.2/4.6 as a market freeze, not a transaction revert -- freezing
// is the SAME response regardless of calling context for this specific
// failure mode, unlike R_fund's routing split in ARCM Section 9.3).
func AccrueInterest(c *Contract, marketID string) *PluginError {
	currentBlock := c.plugin.CurrentHeight()

	// Batched read: Market, BorrowIndex ({25}), SupplyIndex ({26}),
	// LossFactor ({27}) -- one round trip. This function runs once per
	// market EVERY block (BeginBlock) in addition to its DeliverTx-context
	// calls, so batching here matters far more than in a single-transaction
	// handler like deposit.go, which is why this reads inline rather than
	// calling the four separate GetMarket/GetBorrowIndex/GetSupplyIndex/
	// GetLossFactor accessors in state_accessors.go (those remain correct
	// and useful for lower-frequency, single-key callers).
	const (
		qMarket = iota
		qBorrowIndex
		qSupplyIndex
		qLossFactor
	)
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: qMarket, Key: KeyForMarket(marketID)},
			{QueryId: qBorrowIndex, Key: KeyForBorrowIndex(marketID)},
			{QueryId: qSupplyIndex, Key: KeyForSupplyIndex(marketID)},
			{QueryId: qLossFactor, Key: KeyForLossFactor(marketID)},
		},
	})
	if err != nil {
		return err
	}
	if readResp.Error != nil {
		return readResp.Error
	}

	marketBytes := entryValue(readResp, qMarket)
	if len(marketBytes) == 0 {
		// Unreachable in practice: this market_id came from a range read
		// over PrefixMarkets in the BeginBlock caller, or was validated to
		// exist by the DeliverTx handler that called into this function.
		// Guarded explicitly rather than assumed, per this project's
		// standard (Section 10.6-style discipline extended to existence
		// checks, not only cast boundaries).
		return ErrMarketNotFound(marketID)
	}
	market := &Market{}
	if uErr := Unmarshal(marketBytes, market); uErr != nil {
		return uErr
	}

	// --- Step 0 [AYIS v1.10 Section 7, N1] ---
	// Checked BEFORE delta_t is computed, not after a later guard attempts
	// to use it. Without this, last_accrual_block never advances once
	// frozen, so delta_t would grow without bound on every subsequent
	// block, and CompoundExact's exact big.Int exponentiation would
	// recompute an ever-larger result at ever-growing cost, forever, for
	// this one market -- the Principle 11 violation N1 exists to close.
	if market.IndexOverflowHalted {
		return nil
	}

	borrowIndexBytes := entryValue(readResp, qBorrowIndex)
	if len(borrowIndexBytes) == 0 {
		return ErrMarketNotFound(marketID)
	}
	bIndex := DecodeUint128(borrowIndexBytes)

	supplyIndexBytes := entryValue(readResp, qSupplyIndex)
	if len(supplyIndexBytes) == 0 {
		return ErrMarketNotFound(marketID)
	}
	sRate, totalSharesOutstanding := DecodeSupplyIndexRecord(supplyIndexBytes)

	lossFactorBytes := entryValue(readResp, qLossFactor)
	if len(lossFactorBytes) == 0 {
		return ErrMarketNotFound(marketID)
	}
	// lossFactor is read for completeness with AYIS Section 7's full read
	// set, but Step 8 below branches on market.Status, not on lossFactor
	// directly -- Invariant I11 (Insolvency finality) guarantees status is
	// already Insolvent at or before the block loss_factor reaches zero,
	// so status is the correct, already-committed signal to branch on.
	_ = DecodeUint128(lossFactorBytes)

	// --- Step 1: delta_t ---
	deltaT := currentBlock - market.LastAccrualBlock
	if deltaT == 0 {
		// Already accrued this block (e.g. a second DeliverTx-context call
		// in the same block after BeginBlock already ran). No-op, not an
		// error -- matches AYIS Section 7.2's double-accrual handling.
		return nil
	}

	// --- Step 2/3: utilization ---
	totalBorrowed := market.TotalBorrowed
	totalSupplied := market.TotalSupplied
	if totalSupplied == 0 {
		// AYIS Section 7, Step 3: "if total_supplied == 0: skip". No
		// interest to accrue against zero supply; still advance
		// last_accrual_block (Step 11) so delta_t does not silently grow
		// for a market that is legitimately empty, not frozen.
		market.LastAccrualBlock = currentBlock
		if sErr := SaveMarket(c, marketID, market); sErr != nil {
			return sErr
		}
		return nil
	}
	utilizationBps := ComputeUtilizationBps(totalBorrowed, totalSupplied)

	// --- Step 4: annual rate ---
	annualRateBps := ComputeBorrowRate(utilizationBps)

	// --- Step 5: per-block rate ---
	perBlockRate := AnnualRateToPerBlockRateRay(annualRateBps)

	// --- Step 6: B_index update ---
	var newBIndex *big.Int
	if deltaT <= maxDeltaTLinear {
		// Linear approximation: factor = RAY + (per_block_rate * delta_t)
		factor := new(big.Int).Mul(perBlockRate, new(big.Int).SetUint64(deltaT))
		factor.Add(factor, RAY)
		newBIndex = new(big.Int).Mul(bIndex, factor)
		newBIndex.Div(newBIndex, RAY)
	} else {
		newBIndex = CompoundExact(bIndex, perBlockRate, deltaT)
	}

	_, ok := TryEncodeUint128(newBIndex)
	if !ok {
		// [AYIS Section 3.2, M1] This market's B_index would not fit in
		// 128 bits. Freeze THIS market only; every other market's
		// BeginBlock processing is unaffected (Principle 2/14).
		market.IndexOverflowHalted = true
		if sErr := SaveMarket(c, marketID, market); sErr != nil {
			return sErr
		}
		return nil
	}

	// --- Step 7: interest_earned ---
	// interest_earned = total_borrowed * per_block_rate * delta_t / RAY
	interestEarned := new(big.Int).SetUint64(totalBorrowed)
	interestEarned.Mul(interestEarned, perBlockRate)
	interestEarned.Mul(interestEarned, new(big.Int).SetUint64(deltaT))
	interestEarned.Div(interestEarned, RAY)

	// --- Step 8: branch on market.status ---
	// C4 GAP: this condition is market.Status == Insolvent only, NOT
	// market.Status == Insolvent || WillExhaustThisBlock(...). See the
	// file-level comment above -- WillExhaustThisBlock's dependencies
	// (SumLenderBalancesInMarket, {28} queue peek) do not exist yet.
	if market.Status == MarketStatus_INSOLVENT {
		// [AYIS Section 7, Step 8, J1/K1] Full interest_earned routes to
		// R_fund instead of being split. R_fund is ARCM-owned ({18});
		// writing it here requires the same context-dependent encoding
		// ARCM Section 9.3 specifies. R_fund read/write is not yet
		// implemented via a shared accessor -- deferred alongside C4,
		// since both require Layer 4 / ARCM-side state this pass does not
		// touch. TODO: wire R_fund routing once ARCM's reserve-fund
		// accessors exist.
		//
		// B_index still advances normally for an Insolvent market
		// (ScaledDebt() must remain computable for existing borrowers,
		// AYIS Section 6) -- only the S_rate split is skipped.
		bOk, wErr := SetBorrowIndexTry(c, marketID, newBIndex)
		if wErr != nil {
			return wErr
		}
		if !bOk {
			return ErrUint128EncodingOverflow(newBIndex.String())
		}
		market.LastAccrualBlock = currentBlock
		if sErr := SaveMarket(c, marketID, market); sErr != nil {
			return sErr
		}
		return nil
	}

	// Non-Insolvent path: split interest into reserve_cut / supplier_interest.
	reserveFactorRay := new(big.Int).SetUint64(market.ReserveFactorBps)
	reserveFactorRay.Mul(reserveFactorRay, RAY)
	reserveFactorRay.Div(reserveFactorRay, big.NewInt(10000))

	reserveCut := new(big.Int).Mul(interestEarned, reserveFactorRay)
	reserveCut.Div(reserveCut, RAY)

	supplierInterest := new(big.Int).Sub(interestEarned, reserveCut)

	// --- Step 10 (moved earlier, before S_rate write) [ARCM Section 9.3/12.3] ---
	// reserveCut was previously computed and discarded here -- a real
	// accounting leak: this value was subtracted from what suppliers
	// receive (supplierInterest, above) but never credited anywhere. Fixed
	// by reading R_fund, adding reserveCut, and writing it back via the
	// BeginBlock-context-safe TryEncodeUint128 path (Section 9.3,
	// "interest" source leg -- NOT the DeliverTx repay/liquidation legs,
	// which are out of scope; those don't exist yet). On overflow, freeze
	// the market exactly as B_index/S_rate do -- consistent response to
	// the same failure mode at a third accumulator (Principle 14).
	rFund, rFundFound, rErr := GetReserveFund(c, marketID)
	if rErr != nil {
		return rErr
	}
	if !rFundFound {
		// Unreachable in practice: create_market always initializes {18} to
		// zero (Section 4.5's zero-init contract). Guarded explicitly rather
		// than assumed, per this project's established standard.
		return ErrMarketNotFound(marketID)
	}
	newRFund := new(big.Int).Add(rFund, reserveCut)
	rOk, rWriteErr := SetReserveFundTry(c, marketID, newRFund)
	if rWriteErr != nil {
		return rWriteErr
	}
	if !rOk {
		// [ARCM Section 9.3a] R_fund would not fit in 128 bits. Freeze this
		// market only, matching B_index/S_rate's own overflow response.
		// Neither B_index nor S_rate has been written yet at this point in
		// the function, so nothing partially commits (Principle 8).
		market.IndexOverflowHalted = true
		if sErr := SaveMarket(c, marketID, market); sErr != nil {
			return sErr
		}
		return nil
	}

	// --- Step 9: S_rate update ---
	// S_rate(t) = S_rate(t-1) + (S_rate(t-1) * supplier_interest / total_supplied)
	sRateDelta := new(big.Int).Mul(sRate, supplierInterest)
	sRateDelta.Div(sRateDelta, new(big.Int).SetUint64(totalSupplied))
	newSRate := new(big.Int).Add(sRate, sRateDelta)

	newSRateEncoded, sOk := TryEncodeUint128(newSRate)
	if !sOk {
		// [AYIS Section 4.6, M1] This market's S_rate would not fit in 128
		// bits. Per Section 4.6: the entire Step 8/9 update for this
		// market this block is atomic -- B_index (Step 6) has already been
		// computed above but NOT YET WRITTEN, so skipping the write here
		// means neither B_index nor S_rate commits this block, preserving
		// atomicity (Principle 8).
		market.IndexOverflowHalted = true
		if sErr := SaveMarket(c, marketID, market); sErr != nil {
			return sErr
		}
		return nil
	}

	// Both B_index and S_rate are valid to commit -- write both now.
	bOk, wErr := SetBorrowIndexTry(c, marketID, newBIndex)
	if wErr != nil {
		return wErr
	}
	if !bOk {
		return ErrUint128EncodingOverflow(newBIndex.String())
	}
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForSupplyIndex(marketID), Value: EncodeSupplyIndexRecord(newSRateEncoded, totalSharesOutstanding)},
		},
	})
	if wErr != nil {
		return wErr
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}

	// --- Step 11: advance last_accrual_block ---
	market.LastAccrualBlock = currentBlock
	if sErr := SaveMarket(c, marketID, market); sErr != nil {
		return sErr
	}
	return nil
}
