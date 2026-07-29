package contract

// PeekLossFactorQueue reads a market's pending {28} loss-factor-application
// entry, if one exists, without deleting it.
//
// [AYIS Section 12.4] This is a read-only lookahead, used both by
// ProcessLossFactorQueue() (BeginBlock drain step) and by
// WillExhaustThisBlock() (ARCM v3.11.1 Section 9.3b Rule 3, C4) -- the
// latter calls this specifically so it can compare a market's pending
// bad_debt against SumLenderBalancesInMarket() one step earlier than the
// drain itself, without mutating anything. Both callers share this single
// implementation rather than duplicating the read.
func PeekLossFactorQueue(c *Contract, marketID string) (entry *LossFactorQueueEntry, found bool, pErr *PluginError) {
	readResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{
			{QueryId: 0, Key: KeyForLossFactorQueue(marketID)},
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
	entry = &LossFactorQueueEntry{}
	if uErr := Unmarshal(raw, entry); uErr != nil {
		return nil, false, uErr
	}
	return entry, true, nil
}

// EnqueueLossFactorApplication writes (or overwrites) a market's pending
// {28} entry, recording a bad_debt amount for later synchronous processing
// by ProcessLossFactorQueue() in a subsequent BeginBlock step.
//
// [ARCM Section 9.2, Layer 4] Used by the backstop liquidation path
// (protocol-owned forced liquidation, not yet implemented -- see the
// broader Layer 1 backstop gap) when Layer 4 is reached via a
// non-synchronous trigger, as opposed to a voluntary liquidator's
// liquidate_position call, which is expected to call ApplyLossFactor()
// directly instead (synchronous, same transaction). A market has at most
// one outstanding entry: a second enqueue against a market that already
// has a pending entry overwrites it rather than accumulating a second bad
// debt figure separately, since the eventual ApplyLossFactor() call
// operates against the market's current total bad-debt figure at drain
// time, not a per-enqueue-event ledger.
func EnqueueLossFactorApplication(c *Contract, marketID string, badDebt uint64, currentBlock uint64) *PluginError {
	entry := &LossFactorQueueEntry{
		MarketId:      marketID,
		BadDebt:       badDebt,
		EnqueuedBlock: currentBlock,
	}
	entryBytesOut, mErr := Marshal(entry)
	if mErr != nil {
		return mErr
	}
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{{Key: KeyForLossFactorQueue(marketID), Value: entryBytesOut}},
	})
	if wErr != nil {
		return wErr
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// DequeueLossFactorApplication deletes a market's pending {28} entry once
// ProcessLossFactorQueue() has consumed it (i.e. after ApplyLossFactor()
// has been called for that entry's bad_debt amount). Separated from
// ApplyLossFactor() itself so the two can be tested and reasoned about
// independently, matching this codebase's existing separation between
// state accessors and the transaction/BeginBlock logic that calls them.
func DequeueLossFactorApplication(c *Contract, marketID string) *PluginError {
	writeResp, wErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Deletes: []*PluginDeleteOp{{Key: KeyForLossFactorQueue(marketID)}},
	})
	if wErr != nil {
		return wErr
	}
	if writeResp.Error != nil {
		return writeResp.Error
	}
	return nil
}

// ProcessLossFactorQueue implements AYIS Section 12.3's BeginBlock queue-drain
// step for one market: checks for a pending {28} LossFactorQueueEntry and, if
// present, applies it via ApplyLossFactor and dequeues it. Per the Accrual
// Ordering Contract, this MUST be called after AccrueInterest has already run
// for this market in the same block (see contract.go's BeginBlock).
//
// Re-fetches the market fresh via GetMarket rather than trusting a loop-local
// *Market the caller may already hold from before AccrueInterest ran --
// AccrueInterest does its own internal GetMarket/SaveMarket round-trip and does
// not mutate any caller-held struct, so a market read before AccrueInterest ran
// may be stale (e.g. Status, LastAccrualBlock) by the time this function runs.
// ApplyLossFactor's own idempotency check (AYIS Section 5.4.3, K3) depends on
// market.Status being current, so working from a stale copy here could cause
// it to make the wrong branching decision.
//
// Returns (nil, nil) when no entry is pending -- not an error, the expected
// steady state for most markets most blocks. Follows the same single-read,
// single-write discipline as liquidate_position.go's own ApplyLossFactor call
// site: one GetMarket, mutate in-memory via ApplyLossFactor, one SaveMarket.
func ProcessLossFactorQueue(c *Contract, marketID string) (event *Event, pErr *PluginError) {
	entry, found, pkErr := PeekLossFactorQueue(c, marketID)
	if pkErr != nil {
		return nil, pkErr
	}
	if !found {
		return nil, nil
	}

	market, mFound, gErr := GetMarket(c, marketID)
	if gErr != nil {
		return nil, gErr
	}
	if !mFound {
		// Unreachable in practice: a queue entry can only be enqueued for a
		// market that exists. Guarded explicitly per this codebase's standard.
		return nil, ErrMarketNotFound(marketID)
	}

	appliedEvent, aErr := ApplyLossFactor(c, market, marketID, entry.BadDebt)
	if aErr != nil {
		return nil, aErr
	}

	if sErr := SaveMarket(c, marketID, market); sErr != nil {
		return nil, sErr
	}

	if dErr := DequeueLossFactorApplication(c, marketID); dErr != nil {
		return nil, dErr
	}

	return appliedEvent, nil
}
