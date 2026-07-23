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
