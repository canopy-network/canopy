package contract

// waterfall_log.go implements the durable, queryable rolling log of every
// Layer 2/3/4 bad-debt waterfall step (ARCM Section 9.2), backing
// /v1/query/waterfall-events (rpc.go). Closes the exact gap the Arbor
// frontend's own Events panel flags: "Discrete bad-debt waterfall events
// are emitted on-chain but this node exposes no query route for them yet."
//
// See state_keys.go's PrefixWaterfallLog/KeyForWaterfallEvent doc comments
// for the full key-design rationale (height+seq composite key, chosen so a
// Reverse=true range scan serves "most recent N events" directly).

// AppendWaterfallEvent writes one WaterfallEvent record to the {42} log at
// key (blockHeight, seq). Unlike LossFactorQueueEntry's overwrite-in-place
// semantics (loss_factor_queue.go: "a market has at most one outstanding
// entry"), this is a pure append -- every call writes a NEW key, since the
// log's entire purpose is to retain full history, not track current state.
//
// Caller-assigned seq (see KeyForWaterfallEvent's own doc comment): the
// caller is responsible for incrementing seq across multiple waterfall
// steps emitted within the same DeliverTx call (e.g. liquidate_position.go's
// Layer 2 miss -> Layer 3 hit -> Layer 4 fallthrough chain can log more
// than one entry per liquidation). This function does not track or assign
// seq itself -- doing so would require its own read-before-write step
// (peek the highest existing seq for this height), adding a second state
// round-trip to every single waterfall step for a counter the caller
// already has for free (it already knows how many events it has appended
// so far in its own local loop/counter).
// BuildWaterfallEventSetOp marshals one WaterfallEvent record into a
// *PluginSetOp for key (blockHeight, seq), WITHOUT writing it to state.
//
// [CHANGED, atomicity fix] Previously named AppendWaterfallEvent and called
// c.plugin.StateWrite directly as its own independent write, separate from
// the caller's own end-of-function atomic StateWrite. That meant a waterfall
// log entry could commit to state even when the liquidation it described
// subsequently failed and rolled back nothing -- the same non-atomicity bug
// class this codebase already found and fixed twice (market_insolvency.go's
// two-copy race; liquidate_position.go's own former SaveMarket split). Per
// the Canopy builder docs' own rule, operations in one StateWrite call are
// atomic; there is no cross-call guarantee. Callers now collect the
// returned *PluginSetOp into their own sets slice and let it ride in their
// single end-of-function StateWrite call instead.
func BuildWaterfallEventSetOp(blockHeight uint64, seq uint32, event *WaterfallEvent) (*PluginSetOp, *PluginError) {
	eventBytesOut, mErr := Marshal(event)
	if mErr != nil {
		return nil, mErr
	}
	return &PluginSetOp{Key: KeyForWaterfallEvent(blockHeight, seq), Value: eventBytesOut}, nil
}
