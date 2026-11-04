package canoliq

import (
	"bytes"
	"log"

	"github.com/canopy-network/go-plugin/contract"
)

// governancelimits.go adds a governance timelock, quorum and voting-period
// floors, and state-growth limits, all from MainnetGovernanceLimitsHeight, so
// every earlier mainnet block replays unchanged. It is released separately
// from the amount/fee checks in safetychecks.go and has its own height.
//
// Two gates are used. The governance rules (timelock, floors, sensitive
// parameters, proposal caps, minimum stake) use governanceLimitsActive, which
// only switches on mainnet: localnet/devnet genesis files run governance on
// 20-block voting windows and 5-block timelocks for fast iteration, and the
// floors would make that impossible. The bookkeeping changes (request height,
// vote cleanup, boosted-stake running total, vesting guard, treasury re-read)
// do not change any policy and use governanceBookkeepingActive: every
// profile, and mainnet from the same height.
//
// Parameter changes. A passed ProposalParamChange used to be saved in the
// block that tallied it. From activation it is queued and applied once the
// proposal tier's timelock has elapsed (applyPendingParamChanges, from
// BeginBlock). A tier timelock of 0 still applies immediately, which keeps the
// emergency fast track working as designed.
//
// Floors. ValidateParams runs on every LoadParams, so a new invariant there
// would make an already-stored record fail and halt block production. The
// floors are therefore enforced in two places that cannot strand existing
// state: on any new parameter change (validateGovernanceFloors), and at use —
// a proposal's voting period and its quorum/approval are clamped up to the
// floor (effectiveVotingPeriod, effectiveTally). That handles a stored record
// or an in-flight proposal below the floor deterministically at the
// activation height, with no migration.
//
// Sensitive parameters. The multisig signer set and threshold, the treasury
// multisig threshold, the stake output addresses and the governance knobs
// themselves can only change through a parameter-change proposal voted on the
// strictest configured tier (strictestTier), and never through the emergency
// fast track.
//
// Limits. Active proposals are capped globally and per proposer; a CPLQ stake
// position has a minimum size; the quorum denominator is a running total
// instead of a walk over every staker; votes are deleted with their proposal.

const (
	// MinGovernanceQuorumBps is the lowest quorum any proposal is tallied
	// against, and the lowest a parameter change may configure (5%, the lowest
	// default tier).
	MinGovernanceQuorumBps uint64 = 500
	// MinGovernanceApprovalBps is the lowest approval threshold: a strict
	// majority of yes+no.
	MinGovernanceApprovalBps uint64 = 5001
	// MinGovernanceVotingPeriodBlocks is the shortest voting window (the
	// default emergency tier's).
	MinGovernanceVotingPeriodBlocks uint64 = blocks24h
	// MaxActiveProposals caps the active-proposal index.
	MaxActiveProposals = 100
	// MaxActiveProposalsPerProposer caps one address's active proposals.
	MaxActiveProposalsPerProposer = 3
	// MinCplqStakeUcplq is the smallest CPLQ stake position (1 CPLQ).
	MinCplqStakeUcplq uint64 = 1_000_000
)

// MainnetGovernanceLimitsHeight is the committee-29 block from which every
// change in this file applies on mainnet. It must be above the height at which
// every committee-29 node runs this release; pick it with margin for
// pluginAutoUpdate.
const MainnetGovernanceLimitsHeight uint64 = 367_800

// governanceBookkeepingActive reports whether the bookkeeping changes in this
// file apply at `height`. Other profiles have them from genesis.
func (c *Canoliq) governanceBookkeepingActive(height uint64) bool {
	if c.Config.Profile != ProfileMainnet {
		return true
	}
	return height >= MainnetGovernanceLimitsHeight
}

// governanceLimitsActive reports whether the governance rules in this file
// apply at `height`. Mainnet only; see the file comment.
func (c *Canoliq) governanceLimitsActive(height uint64) bool {
	return c.Config.Profile == ProfileMainnet && height >= MainnetGovernanceLimitsHeight
}

// ---- request height -------------------------------------------------------

// currentHeight returns the height a handler should evaluate against. From
// activation, CheckTx/DeliverTx use the height carried on the request, which
// the FSM sets to the block being applied; before it (and when a request
// carries no height, as in BeginBlock/EndBlock and unit tests) it is the
// latest height observed by the long-lived plugin, as before.
func (c *Canoliq) currentHeight() uint64 {
	if c == nil || c.plugin == nil {
		return 0
	}
	if c.reqHeight > 0 && c.governanceBookkeepingActive(c.reqHeight) {
		return c.reqHeight
	}
	return c.plugin.CurrentHeight()
}

// ---- floors -------------------------------------------------------------------

func maxU64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// effectiveVotingPeriod applies the voting-period floor once active.
func (c *Canoliq) effectiveVotingPeriod(period uint64) uint64 {
	if !c.governanceLimitsActive(c.currentHeight()) {
		return period
	}
	return maxU64(period, MinGovernanceVotingPeriodBlocks)
}

// validateGovernanceFloors rejects a parameter set whose quorum, approval or
// voting period is below the floors. Applied to new parameter changes only.
func validateGovernanceFloors(p *contract.CanoliqParams) *contract.PluginError {
	if p.QuorumBps < MinGovernanceQuorumBps || p.PassThresholdBps < MinGovernanceApprovalBps ||
		p.VotingPeriodBlocks < MinGovernanceVotingPeriodBlocks {
		return ErrInvalidParams()
	}
	for _, t := range p.Governance {
		if t.QuorumBps < MinGovernanceQuorumBps || t.ApprovalBps < MinGovernanceApprovalBps {
			return ErrInvalidParams()
		}
		// 0 means "use the scalar voting period", which is floored above.
		if t.VotingPeriodBlocks != 0 && t.VotingPeriodBlocks < MinGovernanceVotingPeriodBlocks {
			return ErrInvalidParams()
		}
	}
	return nil
}

// ---- sensitive parameters ---------------------------------------------------

func addrListsEqual(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func tiersEqual(a, b []*contract.GovernanceTier) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if (x == nil) != (y == nil) {
			return false
		}
		if x == nil {
			continue
		}
		if x.Action != y.Action || x.QuorumBps != y.QuorumBps || x.ApprovalBps != y.ApprovalBps ||
			x.TimelockBlocks != y.TimelockBlocks || x.VotingPeriodBlocks != y.VotingPeriodBlocks {
			return false
		}
	}
	return true
}

// sensitiveParamsChanged reports whether next changes a parameter that only
// the strictest tier may change.
func sensitiveParamsChanged(cur, next *contract.CanoliqParams) bool {
	return !addrListsEqual(cur.MultisigSigners, next.MultisigSigners) ||
		cur.MultisigThreshold != next.MultisigThreshold ||
		cur.TreasuryThreshold != next.TreasuryThreshold ||
		!addrListsEqual(cur.StakeOutputAddresses, next.StakeOutputAddresses) ||
		cur.QuorumBps != next.QuorumBps ||
		cur.PassThresholdBps != next.PassThresholdBps ||
		cur.VotingPeriodBlocks != next.VotingPeriodBlocks ||
		cur.TimelockBlocks != next.TimelockBlocks ||
		!tiersEqual(cur.Governance, next.Governance)
}

// strictestTier returns a tier that is, field by field, the strictest of the
// configured tiers and the scalar knobs, floored. Its Action is
// ACTION_FEE_CHANGE so a parameter change keeps its action type.
func strictestTier(p *contract.CanoliqParams) *contract.GovernanceTier {
	t := &contract.GovernanceTier{
		Action:             contract.ActionType_ACTION_FEE_CHANGE,
		QuorumBps:          maxU64(p.QuorumBps, MinGovernanceQuorumBps),
		ApprovalBps:        maxU64(p.PassThresholdBps, MinGovernanceApprovalBps),
		TimelockBlocks:     p.TimelockBlocks,
		VotingPeriodBlocks: maxU64(p.VotingPeriodBlocks, MinGovernanceVotingPeriodBlocks),
	}
	for _, g := range p.Governance {
		if g == nil {
			continue
		}
		t.QuorumBps = maxU64(t.QuorumBps, g.QuorumBps)
		t.ApprovalBps = maxU64(t.ApprovalBps, g.ApprovalBps)
		t.TimelockBlocks = maxU64(t.TimelockBlocks, g.TimelockBlocks)
		t.VotingPeriodBlocks = maxU64(t.VotingPeriodBlocks, g.VotingPeriodBlocks)
	}
	return t
}

// tierAtLeast reports whether t is at least as strict as want on every field.
func tierAtLeast(t, want *contract.GovernanceTier) bool {
	return t != nil && t.QuorumBps >= want.QuorumBps && t.ApprovalBps >= want.ApprovalBps &&
		t.TimelockBlocks >= want.TimelockBlocks && t.VotingPeriodBlocks >= want.VotingPeriodBlocks
}

// checkParamChangeAllowed is the gate every parameter change passes before it
// is saved: it must validate, respect the floors, and, if it touches a
// sensitive parameter, have been voted on a tier at least as strict as the
// strictest one now configured. emergency changes may not touch sensitive
// parameters at all.
func checkParamChangeAllowed(cur, next *contract.CanoliqParams, tier *contract.GovernanceTier, emergency bool) *contract.PluginError {
	if err := ValidateParams(next); err != nil {
		return err
	}
	if err := validateGovernanceFloors(next); err != nil {
		return err
	}
	if sensitiveParamsChanged(cur, next) {
		if emergency || !tierAtLeast(tier, strictestTier(cur)) {
			return ErrInvalidProposalPayload()
		}
	}
	return nil
}

// ---- proposal creation ----------------------------------------------------

// prepareProposalCreate applies the creation-time limits once active: the
// active-proposal caps, the sensitive-parameter tier upgrade, and the
// emergency restriction. It returns the tier to record (possibly upgraded).
func (c *Canoliq) prepareProposalCreate(proposer []byte, payload interface{}, params *contract.CanoliqParams, idx *contract.ProposalIndex, tier *contract.GovernanceTier) (*contract.GovernanceTier, *contract.PluginError) {
	if len(idx.Ids) >= MaxActiveProposals {
		return nil, ErrTooManyProposals()
	}
	mine := 0
	for _, id := range idx.Ids {
		p, err := c.loadProposal(id)
		if err != nil {
			return nil, err
		}
		if p != nil && bytes.Equal(p.Proposer, proposer) {
			mine++
		}
	}
	if mine >= MaxActiveProposalsPerProposer {
		return nil, ErrTooManyProposals()
	}
	switch p := payload.(type) {
	case *contract.ProposalParamChange:
		if p.Params != nil && sensitiveParamsChanged(params, p.Params) {
			return strictestTier(params), nil
		}
	case *contract.ProposalEmergency:
		if p.ParamChange != nil && p.ParamChange.Params != nil && sensitiveParamsChanged(params, p.ParamChange.Params) {
			return nil, ErrInvalidProposalPayload()
		}
	}
	return tier, nil
}

// ---- tally ------------------------------------------------------------------

// proposalPassesAt is proposalPasses with the floors applied once active.
func (c *Canoliq) proposalPassesAt(p *contract.Proposal, params *contract.CanoliqParams, height uint64) bool {
	if !c.governanceLimitsActive(height) {
		return proposalPasses(p, params)
	}
	quorumBps, approvalBps := params.QuorumBps, params.PassThresholdBps
	if p.Tier != nil {
		quorumBps, approvalBps = p.Tier.QuorumBps, p.Tier.ApprovalBps
	}
	quorumBps = maxU64(quorumBps, MinGovernanceQuorumBps)
	approvalBps = maxU64(approvalBps, MinGovernanceApprovalBps)
	total := p.YesWeight + p.NoWeight + p.AbstainWeight
	if total < mulDiv(p.SnapshotTotalStaked, quorumBps, 10_000) {
		return false
	}
	denom := p.YesWeight + p.NoWeight
	if denom == 0 {
		return false
	}
	return p.YesWeight >= mulDiv(denom, approvalBps, 10_000)
}

// ---- timelocked parameter changes --------------------------------------------

// queueParamChange records a passed parameter change to apply at
// executable. The value is the 8-byte executable height followed by the
// proposal record, which carries the payload and the tier it was voted on.
func (c *Canoliq) queueParamChange(prop *contract.Proposal, executable uint64) *contract.PluginError {
	pBz, e := contract.Marshal(prop)
	if e != nil {
		return e
	}
	idx, err := c.loadPendingParamIndex()
	if err != nil {
		return err
	}
	idx.Ids = append(idx.Ids, prop.Id)
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return e
	}
	val := append(FormatUint64(executable), pBz...)
	_, err = c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{
			{Key: KeyForPendingParamChange(prop.Id), Value: val},
			{Key: KeyForPendingParamIndex(), Value: idxBz},
		},
	})
	return err
}

func (c *Canoliq) loadPendingParamIndex() (*contract.ProposalIndex, *contract.PluginError) {
	idx := new(contract.ProposalIndex)
	if bz := c.readBytes(KeyForPendingParamIndex()); len(bz) > 0 {
		if e := contract.Unmarshal(bz, idx); e != nil {
			return nil, e
		}
	}
	return idx, nil
}

// dispatchParamChangeGated replaces the immediate SaveParams of a passed
// parameter change once active: it validates against the current params and
// then either applies (timelock 0) or queues the change.
func (c *Canoliq) dispatchParamChangeGated(prop *contract.Proposal, next *contract.CanoliqParams, params *contract.CanoliqParams, height uint64, emergency bool) *contract.PluginError {
	if err := checkParamChangeAllowed(params, next, prop.Tier, emergency); err != nil {
		return err
	}
	timelock := params.TimelockBlocks
	if prop.Tier != nil {
		timelock = prop.Tier.TimelockBlocks
	}
	if timelock == 0 {
		return c.SaveParams(next)
	}
	return c.queueParamChange(prop, height+timelock)
}

// applyPendingParamChanges applies every queued parameter change whose
// timelock has elapsed, in queue order. A change that no longer passes
// checkParamChangeAllowed against the then-current params is dropped and
// reported, never retried, so it cannot stall block production.
func (c *Canoliq) applyPendingParamChanges(height uint64) *contract.PluginError {
	if !c.governanceLimitsActive(height) {
		return nil
	}
	idx, err := c.loadPendingParamIndex()
	if err != nil {
		return err
	}
	if len(idx.Ids) == 0 {
		return nil
	}
	var keep []uint64
	var deletes []*contract.PluginDeleteOp
	for _, id := range idx.Ids {
		bz := c.readBytes(KeyForPendingParamChange(id))
		if len(bz) < 8 {
			continue // stale index entry
		}
		if DecodeUint64(bz[:8]) > height {
			keep = append(keep, id)
			continue
		}
		deletes = append(deletes, &contract.PluginDeleteOp{Key: KeyForPendingParamChange(id)})
		prop := new(contract.Proposal)
		if e := contract.Unmarshal(bz[8:], prop); e != nil {
			c.reportDispatchFailure(prop, height, e)
			continue
		}
		payload, perr := unwrapPayload(prop.Payload)
		pc, ok := payload.(*contract.ProposalParamChange)
		if perr != nil || !ok || pc.Params == nil {
			c.reportDispatchFailure(prop, height, ErrInvalidProposalPayload())
			continue
		}
		cur, err := c.LoadParams()
		if err != nil {
			return err
		}
		if err := checkParamChangeAllowed(cur, pc.Params, prop.Tier, false); err != nil {
			c.reportDispatchFailure(prop, height, err)
			continue
		}
		if err := c.SaveParams(pc.Params); err != nil {
			c.reportDispatchFailure(prop, height, err)
			continue
		}
		log.Printf("canoliq: applied parameter change from proposal %d at height %d", prop.Id, height)
	}
	idx.Ids = keep
	if len(keep) == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: KeyForPendingParamIndex()})
		_, err = c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Deletes: deletes})
		return err
	}
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return e
	}
	_, err = c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets:    []*contract.PluginSetOp{{Key: KeyForPendingParamIndex(), Value: idxBz}},
		Deletes: deletes,
	})
	return err
}

// ---- vote cleanup -------------------------------------------------------------

// voteDeletesFor returns delete ops for every vote record of a proposal once
// active; before activation votes were left in place.
func (c *Canoliq) voteDeletesFor(proposalID uint64) ([]*contract.PluginDeleteOp, *contract.PluginError) {
	if !c.governanceBookkeepingActive(c.currentHeight()) {
		return nil, nil
	}
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Ranges: []*contract.PluginRangeRead{{QueryId: q, Prefix: VotePrefixForProposal(proposalID)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var deletes []*contract.PluginDeleteOp
	for _, r := range resp.Results {
		for _, e := range r.Entries {
			deletes = append(deletes, &contract.PluginDeleteOp{Key: e.Key})
		}
	}
	return deletes, nil
}

// ---- boosted stake running total ---------------------------------------------

// loadBoostedStakeTotal returns the running total of lock-boosted stake
// weight. The first call after activation finds no record and computes it
// once by walking the staker index; the caller persists it, after which every
// stake change keeps it current. Idempotent: the walk and the running total
// agree by construction.
func (c *Canoliq) loadBoostedStakeTotal() (uint64, *contract.PluginError) {
	if bz := c.readBytes(KeyForBoostedStakeTotal()); len(bz) >= 8 {
		return DecodeUint64(bz), nil
	}
	return c.boostedStakeTotal()
}

// boostedTotalUpdate returns the set op that moves the running total from a
// staker's old weight to its new one, or nil before activation.
func (c *Canoliq) boostedTotalUpdate(oldWeight, newWeight uint64) (*contract.PluginSetOp, *contract.PluginError) {
	if !c.governanceBookkeepingActive(c.currentHeight()) {
		return nil, nil
	}
	total, err := c.loadBoostedStakeTotal()
	if err != nil {
		return nil, err
	}
	if total >= oldWeight {
		total -= oldWeight
	} else {
		total = 0
	}
	if addOverflows(total, newWeight) {
		return nil, ErrAmountOutOfRange()
	}
	return &contract.PluginSetOp{Key: KeyForBoostedStakeTotal(), Value: EncodeUint64(total + newWeight)}, nil
}

// snapshotBoostedTotal is the quorum denominator for a new proposal, plus the
// set op that persists the running total when it was just computed.
func (c *Canoliq) snapshotBoostedTotal() (uint64, *contract.PluginSetOp, *contract.PluginError) {
	if !c.governanceBookkeepingActive(c.currentHeight()) {
		total, err := c.boostedStakeTotal()
		return total, nil, err
	}
	if bz := c.readBytes(KeyForBoostedStakeTotal()); len(bz) >= 8 {
		return DecodeUint64(bz), nil, nil
	}
	total, err := c.boostedStakeTotal()
	if err != nil {
		return 0, nil, err
	}
	return total, &contract.PluginSetOp{Key: KeyForBoostedStakeTotal(), Value: EncodeUint64(total)}, nil
}

// ---- vesting ------------------------------------------------------------------

// unlockedAmountAt is unlockedAmount with a guard, once active, for a
// schedule whose cliff precedes its start: nothing unlocks before the start.
func (c *Canoliq) unlockedAmountAt(s *contract.VestingSchedule, height uint64) uint64 {
	if s != nil && height < s.StartHeight && c.governanceBookkeepingActive(c.currentHeight()) {
		return 0
	}
	return unlockedAmount(s, height)
}
