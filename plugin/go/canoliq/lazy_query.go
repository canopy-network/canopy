package canoliq

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/canopy-network/go-plugin/contract"
)

// lazy_query.go services per-address HTTP queries that the snapshot model
// cannot satisfy directly. Phase 3 §1's snapshot enumerates only state
// reachable from singletons or plugin-maintained indexes, so queries like
// "the vesting schedules for address X" or "redemption (addr, id)" need
// arbitrary StateReads — and StateReads only work inside an active FSM
// lifecycle window (see memory: plugin_stateread_constraint).
//
// The lazy queue closes the gap. HTTP handlers build a *lazyQuery,
// enqueue it on Plugin.pendingQueries, and block on its result channel.
// EndBlock (where c.fsmId is the live FSM-originated id) drains the
// queue between refreshSnapshot and returning, executes each query as
// a normal StateRead, and sends the result back. Worst-case latency:
// one block (~6s on localnet).

// lazyQueueCapacity bounds the in-flight query count to prevent OOM
// under burst load. Excess requests get a 503; clients can retry.
const lazyQueueCapacity = 256

// lazyQueryTimeout is the HTTP-side block timeout. ~2.5x the localnet
// 6s block period gives one block of safety margin while still failing
// fast on a stalled chain.
const lazyQueryTimeout = 15 * time.Second

// errLazyQueueFull is returned to the HTTP handler when the queue is
// saturated. It maps to 503 in the route layer.
var errLazyQueueFull = errors.New("lazy query queue full")

// errLazyQueryTimeout is returned when EndBlock did not drain the
// query within lazyQueryTimeout. Maps to 504 in the route layer.
var errLazyQueryTimeout = errors.New("lazy query timed out")

// lazyKind discriminates the query type within a single channel so
// EndBlock can dispatch without reflection.
type lazyKind int

const (
	lazyKindAccount lazyKind = iota
	lazyKindVesting
	lazyKindRedemption
	lazyKindVote
	lazyKindBuyback
)

// lazyQuery is one in-flight per-address query. Result is sent on the
// result channel; only one of view/views/redemption/vote/buyback is set
// per query, matching kind. err is non-nil on plugin-internal failure.
type lazyQuery struct {
	kind        lazyKind
	addr        []byte // for account / vesting / redemption
	id          uint64 // for redemption / vote / buyback
	voter       []byte // for vote
	result      chan lazyResult
}

// lazyResult is the response payload union. Exactly one field matches
// the originating lazyQuery.kind. found=false on a clean miss (404),
// distinct from err which signals plugin failure (500).
type lazyResult struct {
	found     bool
	err       error
	view      *AccountView
	views     []*VestingView
	redemption *contract.Redemption
	vote      *contract.Vote
	buyback   *contract.BuybackOrder
}

// AccountView is the composite per-address response. CNPY comes from
// the core Account record; cCNPY/CPLQ are scalar plugin keys; CPLQStake
// and validator-incentive are direct lookups; vesting / pending
// redemptions / pending unstakes are enumerated via per-address
// indexes maintained by their respective Deliver handlers.
type AccountView struct {
	Address            string                    `json:"address"`
	CNPY               uint64                    `json:"cnpy"`
	CCNPY              uint64                    `json:"ccnpy"`
	CPLQLiquid         uint64                    `json:"cplqLiquid"`
	CPLQStake          *contract.CPLQStake       `json:"cplqStake,omitempty"`
	ValidatorIncentive uint64                    `json:"validatorIncentive"`
	Vestings           []*VestingView            `json:"vestings,omitempty"`
	Redemptions        []*contract.Redemption    `json:"redemptions,omitempty"`
	Unstakes           []*contract.UnstakingCPLQ `json:"unstakes,omitempty"`
}

// VestingView is one schedule annotated with the cumulative unlocked
// amount at the plugin's current height (not just claimed_amount).
type VestingView struct {
	Schedule       *contract.VestingSchedule `json:"schedule"`
	UnlockedToDate uint64                    `json:"unlockedToDate"`
	ClaimableNow   uint64                    `json:"claimableNow"`
	CurrentHeight  uint64                    `json:"currentHeight"`
}

// enqueueLazy submits a lazy query and blocks on the result with timeout.
// The request context is honored on both legs so client disconnects abort
// the wait promptly instead of pinning a goroutine for the full timeout.
// Errors:
//   - errLazyQueueFull when the channel is saturated for >100ms
//   - errLazyQueryTimeout when no drain landed within lazyQueryTimeout
//   - ctx.Err() when the caller's context is canceled
func (p *Plugin) enqueueLazy(ctx context.Context, q *lazyQuery) lazyResult {
	if ctx == nil {
		ctx = context.Background()
	}
	q.result = make(chan lazyResult, 1)
	// Try to enqueue. Block briefly if full so transient bursts succeed,
	// but cap at 100ms so a stuck queue surfaces fast.
	select {
	case p.pendingQueries <- q:
	case <-ctx.Done():
		return lazyResult{err: ctx.Err()}
	case <-time.After(100 * time.Millisecond):
		return lazyResult{err: errLazyQueueFull}
	}
	// Wait for the drainer to fulfill.
	select {
	case r := <-q.result:
		return r
	case <-ctx.Done():
		return lazyResult{err: ctx.Err()}
	case <-time.After(lazyQueryTimeout):
		return lazyResult{err: errLazyQueryTimeout}
	}
}

// drainLazyQueries pulls every pending query off the channel and
// fulfills it via a state read against the active FSM context.
// Called from EndBlock after refreshSnapshot publishes the new
// singleton/index view.
func (c *Canoliq) drainLazyQueries() {
	if c == nil || c.plugin == nil {
		return
	}
	for {
		select {
		case q := <-c.plugin.pendingQueries:
			c.fulfillLazy(q)
		default:
			return
		}
	}
}

// fulfillLazy dispatches a single lazy query to its kind-specific reader
// and ships the result back. Errors here are protocol-level (FSM read
// failed); a clean "no such record" comes back as found=false.
func (c *Canoliq) fulfillLazy(q *lazyQuery) {
	defer func() {
		if r := recover(); r != nil {
			q.result <- lazyResult{err: errLazyQueueFull}
		}
	}()
	switch q.kind {
	case lazyKindAccount:
		view, err := c.buildAccountView(q.addr)
		if err != nil {
			q.result <- lazyResult{err: errors.New(err.Msg)}
			return
		}
		q.result <- lazyResult{found: true, view: view}
	case lazyKindVesting:
		views, found, err := c.buildVestingView(q.addr)
		if err != nil {
			q.result <- lazyResult{err: errors.New(err.Msg)}
			return
		}
		q.result <- lazyResult{found: found, views: views}
	case lazyKindRedemption:
		red, err := c.readRedemption(q.addr, q.id)
		if err != nil {
			q.result <- lazyResult{err: errors.New(err.Msg)}
			return
		}
		q.result <- lazyResult{found: red != nil, redemption: red}
	case lazyKindVote:
		v, err := c.readVote(q.id, q.voter)
		if err != nil {
			q.result <- lazyResult{err: errors.New(err.Msg)}
			return
		}
		q.result <- lazyResult{found: v != nil, vote: v}
	case lazyKindBuyback:
		o, err := c.readBuyback(q.id)
		if err != nil {
			q.result <- lazyResult{err: errors.New(err.Msg)}
			return
		}
		q.result <- lazyResult{found: o != nil, buyback: o}
	default:
		q.result <- lazyResult{err: errors.New("unknown lazy kind")}
	}
}

// buildAccountView batches all the singleton-by-address reads for one
// address into a single StateRead, then follows up with a vesting-index
// read + per-schedule reads. Returns a fully-populated view (zero values
// for absent records — addresses with no balance still return a valid
// AccountView, useful for "is this address known?" probes).
func (c *Canoliq) buildAccountView(addr []byte) (*AccountView, *contract.PluginError) {
	cnpyKey := contract.KeyForAccount(addr)
	ccnpyKey := KeyForCCNPYBalance(addr)
	cplqBalKey := KeyForCPLQBalance(addr)
	stakeKey := KeyForCPLQStake(addr)
	valIncentKey := KeyForValidatorIncentives(addr)
	vestIdxKey := KeyForVestingIndex(addr)
	redemIdxKey := KeyForRedemptionIndex(addr)
	unstIdxKey := KeyForUnstakingIndex(addr)
	cQ, ccQ, lcQ, sQ, viQ, vQ, riQ, uiQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: ccQ, Key: ccnpyKey},
			{QueryId: lcQ, Key: cplqBalKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: viQ, Key: valIncentKey},
			{QueryId: vQ, Key: vestIdxKey},
			{QueryId: riQ, Key: redemIdxKey},
			{QueryId: uiQ, Key: unstIdxKey},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	view := &AccountView{Address: hexAddress(addr)}
	var vestIdxBz, redemIdxBz, unstIdxBz []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		raw := r.Entries[0].Value
		switch r.QueryId {
		case cQ:
			acc := new(contract.Account)
			if e := contract.Unmarshal(raw, acc); e != nil {
				return nil, e
			}
			view.CNPY = acc.Amount
		case ccQ:
			view.CCNPY = DecodeUint64(raw)
		case lcQ:
			view.CPLQLiquid = DecodeUint64(raw)
		case sQ:
			stake := new(contract.CPLQStake)
			if e := contract.Unmarshal(raw, stake); e != nil {
				return nil, e
			}
			if stake.Address != nil {
				view.CPLQStake = stake
			}
		case viQ:
			view.ValidatorIncentive = DecodeUint64(raw)
		case vQ:
			vestIdxBz = raw
		case riQ:
			redemIdxBz = raw
		case uiQ:
			unstIdxBz = raw
		}
	}
	if len(vestIdxBz) > 0 {
		idx := new(contract.VestingIndex)
		if e := contract.Unmarshal(vestIdxBz, idx); e != nil {
			return nil, e
		}
		views, _, err := c.readVestingSchedules(addr, idx.ScheduleIds)
		if err != nil {
			return nil, err
		}
		view.Vestings = views
	}
	if len(redemIdxBz) > 0 {
		idx := new(contract.RedemptionIndex)
		if e := contract.Unmarshal(redemIdxBz, idx); e != nil {
			return nil, e
		}
		reds, err := c.readRedemptions(addr, idx.Ids)
		if err != nil {
			return nil, err
		}
		view.Redemptions = reds
	}
	if len(unstIdxBz) > 0 {
		idx := new(contract.UnstakingIndex)
		if e := contract.Unmarshal(unstIdxBz, idx); e != nil {
			return nil, e
		}
		uns, err := c.readUnstakings(addr, idx.Ids)
		if err != nil {
			return nil, err
		}
		view.Unstakes = uns
	}
	return view, nil
}

// readRedemptions batch-reads Redemption records for the given ids.
// Missing entries are skipped silently — the index may briefly contain
// an id that has been claimed concurrently in the same block, though
// the write-side index maintenance keeps this rare.
func (c *Canoliq) readRedemptions(addr []byte, ids []uint64) ([]*contract.Redemption, *contract.PluginError) {
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]*contract.PluginKeyRead, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, &contract.PluginKeyRead{QueryId: rand.Uint64(), Key: KeyForRedemption(addr, id)})
	}
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: keys})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	out := make([]*contract.Redemption, 0, len(ids))
	for _, r := range resp.Results {
		if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
			continue
		}
		red := new(contract.Redemption)
		if e := contract.Unmarshal(r.Entries[0].Value, red); e != nil {
			return nil, e
		}
		if red.Address == nil {
			continue
		}
		out = append(out, red)
	}
	return out, nil
}

// readUnstakings batch-reads UnstakingCPLQ records for the given ids.
func (c *Canoliq) readUnstakings(addr []byte, ids []uint64) ([]*contract.UnstakingCPLQ, *contract.PluginError) {
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]*contract.PluginKeyRead, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, &contract.PluginKeyRead{QueryId: rand.Uint64(), Key: KeyForCPLQUnstaking(addr, id)})
	}
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: keys})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	out := make([]*contract.UnstakingCPLQ, 0, len(ids))
	for _, r := range resp.Results {
		if len(r.Entries) == 0 || len(r.Entries[0].Value) == 0 {
			continue
		}
		u := new(contract.UnstakingCPLQ)
		if e := contract.Unmarshal(r.Entries[0].Value, u); e != nil {
			return nil, e
		}
		if u.Address == nil {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

// buildVestingView returns every vesting schedule for an address.
// found=false when the address has no VestingIndex entry at all
// (route handler maps this to 404).
func (c *Canoliq) buildVestingView(addr []byte) ([]*VestingView, bool, *contract.PluginError) {
	idxKey := KeyForVestingIndex(addr)
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: idxKey}},
	})
	if err != nil {
		return nil, false, err
	}
	if resp.Error != nil {
		return nil, false, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return nil, false, nil
	}
	idx := new(contract.VestingIndex)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
		return nil, false, e
	}
	views, _, err := c.readVestingSchedules(addr, idx.ScheduleIds)
	if err != nil {
		return nil, false, err
	}
	return views, true, nil
}

// readVestingSchedules batch-loads schedule records and annotates each
// with cumulative unlocked + claimable-now relative to current height.
// The bool is unused by callers but kept symmetric with the parsing
// path; nil schedules are skipped.
func (c *Canoliq) readVestingSchedules(addr []byte, ids []uint64) ([]*VestingView, bool, *contract.PluginError) {
	if len(ids) == 0 {
		return nil, false, nil
	}
	reads := make([]*contract.PluginKeyRead, 0, len(ids))
	for _, id := range ids {
		reads = append(reads, &contract.PluginKeyRead{
			QueryId: rand.Uint64(),
			Key:     KeyForVesting(addr, id),
		})
	}
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: reads})
	if err != nil {
		return nil, false, err
	}
	if resp.Error != nil {
		return nil, false, resp.Error
	}
	height := c.currentHeight()
	views := make([]*VestingView, 0, len(ids))
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		sched := new(contract.VestingSchedule)
		if e := contract.Unmarshal(r.Entries[0].Value, sched); e != nil {
			return nil, false, e
		}
		unlocked := unlockedAmount(sched, height)
		claimable := uint64(0)
		if unlocked > sched.ClaimedAmount {
			claimable = unlocked - sched.ClaimedAmount
		}
		views = append(views, &VestingView{
			Schedule:       sched,
			UnlockedToDate: unlocked,
			ClaimableNow:   claimable,
			CurrentHeight:  height,
		})
	}
	return views, true, nil
}

// readRedemption returns one redemption record by (addr, id) or
// (nil, nil) when absent.
func (c *Canoliq) readRedemption(addr []byte, id uint64) (*contract.Redemption, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForRedemption(addr, id)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return nil, nil
	}
	red := new(contract.Redemption)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, red); e != nil {
		return nil, e
	}
	return red, nil
}

// readVote returns one vote record by (proposalId, voter) or (nil, nil).
func (c *Canoliq) readVote(proposalID uint64, voter []byte) (*contract.Vote, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForVote(proposalID, voter)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return nil, nil
	}
	v := new(contract.Vote)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, v); e != nil {
		return nil, e
	}
	return v, nil
}

// readBuyback returns one buyback order by proposal id or (nil, nil).
func (c *Canoliq) readBuyback(id uint64) (*contract.BuybackOrder, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForBuybackOrder(id)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return nil, nil
	}
	order := new(contract.BuybackOrder)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, order); e != nil {
		return nil, e
	}
	return order, nil
}
