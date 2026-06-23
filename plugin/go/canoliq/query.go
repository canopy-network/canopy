package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// query.go implements the read-only state accessors that back the HTTP
// query surface defined in rpc.go. All helpers serve from the plugin-side
// Snapshot built inside EndBlock — see snapshot.go for why a snapshot
// is the only viable read path. None of these helpers issues a state
// read; they are pure projections of an already-frozen Snapshot.
//
// Per-address routes (account composite, vesting-by-address,
// redemption-by-id, vote-by-voter, buyback-by-id) are not supported here:
// the snapshot can only enumerate state reachable from a singleton or an
// existing index. Adding them is Phase 3 §1.1 work.

// PoolsView aggregates the canoliq-owned scalar buckets and the on-chain
// committee fee pool. Validator-incentive accruals are itemized so a
// dashboard can flag uneven share-out across the registered validator set.
type PoolsView struct {
	CommitteePool       uint64               `json:"committeePool"`
	TreasuryCNPY        uint64               `json:"treasuryCnpy"`
	TreasuryCLIQ        uint64               `json:"treasuryCliq"`
	BuybackPool         uint64               `json:"buybackPool"`
	InsurancePool       uint64               `json:"insurancePool"`
	ValidatorIncentives []ValidatorIncentive `json:"validatorIncentives"`
	// PeakTvlUcnpy is the running max of total_pooled_cnpy (T4).
	PeakTvlUcnpy uint64 `json:"peakTvlUcnpy"`
	// InsuranceTargetUcnpy is the reserve target (insurance_target_bps of peak
	// TVL); 0 when the gate is disabled.
	InsuranceTargetUcnpy uint64 `json:"insuranceTargetUcnpy"`
	// InsuranceFundedBps is insurance_pool / target in bps (0 when no target).
	InsuranceFundedBps uint64 `json:"insuranceFundedBps"`
}

// ValidatorIncentive is one entry of the per-validator incentives ledger.
type ValidatorIncentive struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

// MultisigApprovalsView lists the live approvals against a spend, filtered
// against the *current* signer set so stale approvals from removed signers
// do not appear (consistent with countMultisigApprovals).
type MultisigApprovalsView struct {
	SpendID   uint64                       `json:"spendId"`
	Threshold uint64                       `json:"threshold"`
	Approvals []*contract.MultisigApproval `json:"approvals"`
}

// TVL-cap status strings reported in HealthView.TVLCapStatus. Operators
// rely on these to tell apart the three runtime behaviours of the cap
// when TVLCapBps > 0: enforcing, silently accepting (cap underdetermined
// due to thin Canopy state), or rejecting all deposits (Supply absent).
const (
	// TVLCapStatusUncapped — TvlCapBps == 0; governance has not set a cap.
	// Deposits accepted without bound.
	TVLCapStatusUncapped = "uncapped"
	// TVLCapStatusActive — TvlCapBps > 0 AND the live effective uCNPY cap
	// is non-zero. Deposits enforced against the cap.
	TVLCapStatusActive = "active"
	// TVLCapStatusAwaitingCanopyStake — TvlCapBps > 0 but the live
	// effective uCNPY cap is zero, either because Canopy Supply.Staked is
	// 0 or because mulDiv truncates to 0 at very low Canopy stake. Deposits
	// accepted this block; the cap re-engages automatically once Canopy
	// stake grows. Per H3 (cc5c789e) / M6 (385f3424).
	TVLCapStatusAwaitingCanopyStake = "awaiting-canopy-stake"
	// TVLCapStatusFailClosed — TvlCapBps > 0 AND the Canopy Supply
	// singleton is missing from state. The deposit handler rejects every
	// deposit with ErrCanopyStakeUnavailable. Operationally this is a
	// misconfiguration: either seed Supply in Canopy genesis before
	// opening canoLiq deposits, or set TvlCapBps = 0 during bring-up.
	TVLCapStatusFailClosed = "fail-closed"
)

// HealthView is the liveness response: enough to confirm the plugin is up,
// what height the snapshot reflects, and whether genesis has run.
type HealthView struct {
	Height          uint64 `json:"height"`
	GenesisComplete bool   `json:"genesisComplete"`
	ChainID         uint64 `json:"chainId"`
	// TVLCapBps is the governance-set TVL ceiling as a fraction of total
	// Canopy network stake (WP §9.4). 0 = uncapped.
	TVLCapBps uint64 `json:"tvlCapBps"`
	// TVLCapStatus is the runtime status of the cap at snapshot height —
	// one of the TVLCapStatus* constants. Operators need this to tell
	// apart 'cap silently inactive' from 'cap rejecting all deposits',
	// both of which look like TVLCapUcnpyEffective=0 in isolation.
	TVLCapStatus string `json:"tvlCapStatus"`
	// TVLCapUcnpyEffective is the live uCNPY cap at snapshot height —
	// mulDiv(canopy_total_stake, tvl_cap_bps, 10_000). Zero when uncapped,
	// when awaiting Canopy stake, or when fail-closed. Pair with
	// TVLCapStatus to interpret.
	TVLCapUcnpyEffective uint64 `json:"tvlCapUcnpyEffective"`
	// CanopyTotalStake is the snapshot-height value of lib.Supply.staked —
	// surfaced so operators can sanity-check the effective cap. Zero when
	// Supply is absent (then TVLCapStatus = "fail-closed") OR when Supply
	// is present with Staked = 0 (then TVLCapStatus = "awaiting-canopy-stake").
	CanopyTotalStake uint64 `json:"canopyTotalStake"`
	// TVLUtilizationBps is total_pooled_cnpy / tvl_cap_ucnpy_effective in bps
	// (0 when the effective cap is 0, regardless of status).
	TVLUtilizationBps uint64 `json:"tvlUtilizationBps"`
}

// StakerView is one entry in the active CLIQ staker list.
type StakerView struct {
	Address        string `json:"address"`
	Amount         uint64 `json:"amount"`
	StakedAtHeight uint64 `json:"stakedAtHeight"`
}

// QueryHealth returns liveness info — height + genesis flag + chain id —
// plus the live TVL cap surface (status, cap bps, computed effective
// uCNPY cap, canopy total stake, utilization). Always succeeds; falls
// back to zeros before the first snapshot.
func (p *Plugin) QueryHealth() *HealthView {
	s := p.Snapshot()
	var effectiveCap, utilBps uint64
	status := deriveTVLCapStatus(s)
	if status == TVLCapStatusActive {
		effectiveCap = mulDiv(s.CanopyTotalStake, s.Params.TvlCapBps, 10_000)
		utilBps = mulDiv(s.Globals.TotalPooledCnpy, 10_000, effectiveCap)
	}
	return &HealthView{
		Height:               s.Height,
		GenesisComplete:      s.Globals.GenesisComplete,
		ChainID:              p.config.ChainId,
		TVLCapBps:            s.Params.TvlCapBps,
		TVLCapStatus:         status,
		TVLCapUcnpyEffective: effectiveCap,
		CanopyTotalStake:     s.CanopyTotalStake,
		TVLUtilizationBps:    utilBps,
	}
}

// deriveTVLCapStatus maps the four-way TVL-cap runtime behaviour to the
// reported status string. Mirrors the deliver.go branch structure: no
// TvlCapBps → uncapped; Supply absent → fail-closed; live capUcnpy == 0
// → awaiting-canopy-stake (deposit accepts); otherwise → active.
func deriveTVLCapStatus(s *Snapshot) string {
	if s.Params.TvlCapBps == 0 {
		return TVLCapStatusUncapped
	}
	if !s.CanopySupplyPresent {
		return TVLCapStatusFailClosed
	}
	if mulDiv(s.CanopyTotalStake, s.Params.TvlCapBps, 10_000) == 0 {
		return TVLCapStatusAwaitingCanopyStake
	}
	return TVLCapStatusActive
}

// QueryRestaking returns the WP §7 policy + observed-exposure surface.
// Policy is the governance-declared per-committee allocation; allocations
// are derived from canoLiq's registered operators' Canopy Validator
// committees[] + staked_amount (restaking semantics: same bond covers
// multiple committees). When both are empty, the response is well-formed
// with zeros — RPC consumers can rely on the schema regardless of state.
//
// Phase C scope is policy + observability. Active rebalancing is out of
// scope; operators consult drift bps + under-min / over-max flags and
// can act manually (e.g. by re-targeting which operators canoLiq
// delegates to via existing governance / registry mechanisms).
func (p *Plugin) QueryRestaking() *RestakingView {
	s := p.Snapshot()
	return buildRestakingView(s.Params.GetRestakingPolicy(), s.CurrentRestakingAllocation)
}

// QueryGlobals returns the snapshotted globals record.
func (p *Plugin) QueryGlobals() *contract.CanoliqGlobals {
	return p.Snapshot().Globals
}

// QueryParams returns the snapshotted params.
func (p *Plugin) QueryParams() *contract.CanoliqParams {
	return p.Snapshot().Params
}

// QueryPools assembles the pool-balance view from the snapshot.
func (p *Plugin) QueryPools() *PoolsView {
	s := p.Snapshot()
	view := &PoolsView{
		CommitteePool: s.CommitteePool,
		TreasuryCNPY:  s.TreasuryCNPY,
		TreasuryCLIQ:  s.TreasuryCLIQ,
		BuybackPool:   s.BuybackPool,
		InsurancePool: s.InsurancePool,
		PeakTvlUcnpy:  s.Globals.PeakTvlUcnpy,
	}
	if s.Params.InsuranceTargetBps > 0 {
		view.InsuranceTargetUcnpy = mulDiv(s.Globals.PeakTvlUcnpy, s.Params.InsuranceTargetBps, 10_000)
		if view.InsuranceTargetUcnpy > 0 {
			view.InsuranceFundedBps = mulDiv(s.InsurancePool, 10_000, view.InsuranceTargetUcnpy)
		}
	}
	for addr, amt := range s.ValidatorIncentives {
		if amt == 0 {
			continue
		}
		view.ValidatorIncentives = append(view.ValidatorIncentives, ValidatorIncentive{
			Address: addr,
			Amount:  amt,
		})
	}
	return view
}

// QueryProposalIDs returns the active proposal id list.
func (p *Plugin) QueryProposalIDs() []uint64 {
	ids := p.Snapshot().ActiveProposalIDs
	if ids == nil {
		return []uint64{}
	}
	return ids
}

// QueryProposal returns one active proposal by id, or nil when not present
// in the snapshot.
func (p *Plugin) QueryProposal(id uint64) *contract.Proposal {
	return p.Snapshot().Proposals[id]
}

// QuerySpendIDs returns the pending treasury-spend id list.
func (p *Plugin) QuerySpendIDs() []uint64 {
	ids := p.Snapshot().PendingSpendIDs
	if ids == nil {
		return []uint64{}
	}
	return ids
}

// QuerySpend returns one treasury spend by id, or nil.
func (p *Plugin) QuerySpend(id uint64) *contract.TreasurySpend {
	return p.Snapshot().Spends[id]
}

// QueryMultisigApprovals returns the live approvals against a spend.
func (p *Plugin) QueryMultisigApprovals(id uint64) *MultisigApprovalsView {
	s := p.Snapshot()
	return &MultisigApprovalsView{
		SpendID:   id,
		Threshold: s.Params.MultisigThreshold,
		Approvals: s.MultisigApprovals[id],
	}
}

// QueryValidatorRegistry returns the snapshotted registry.
func (p *Plugin) QueryValidatorRegistry() *contract.ValidatorRegistry {
	return p.Snapshot().ValidatorRegistry
}

// QueryStakers returns the active CLIQ stake records, ordered by the
// snapshot's stake-index entry order.
func (p *Plugin) QueryStakers() []*StakerView {
	s := p.Snapshot()
	out := make([]*StakerView, 0, len(s.StakerAddresses))
	for _, addr := range s.StakerAddresses {
		hex := hexAddress(addr)
		stake := s.Stakers[hex]
		if stake == nil {
			continue
		}
		out = append(out, &StakerView{
			Address:        hex,
			Amount:         stake.Amount,
			StakedAtHeight: stake.StakedAtHeight,
		})
	}
	return out
}

// hexAddress encodes a 20-byte address as a 0x-prefixed lowercase hex
// string. Centralized here so query results and rpc handlers agree on
// encoding.
func hexAddress(addr []byte) string {
	const hexchars = "0123456789abcdef"
	out := make([]byte, 2+len(addr)*2)
	out[0] = '0'
	out[1] = 'x'
	for i, b := range addr {
		out[2+i*2] = hexchars[b>>4]
		out[2+i*2+1] = hexchars[b&0x0f]
	}
	return string(out)
}
