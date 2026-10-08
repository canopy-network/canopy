package canoliq

import (
	"sort"

	"github.com/canopy-network/go-plugin/contract"
)

// restaking.go implements the policy + observability scope of WP §7
// (Restaking Optimization). Phase C as scoped: declare the desired
// per-committee allocation, observe the actual exposure derived from
// Canopy state, report the drift via /v1/restaking. Active rebalancing
// is out of scope for now — not because Canopy lacks a delegation
// primitive (lib.Validator.delegate plus fsm/committee.go's GetDelegates
// and LotteryWinner is exactly that), but because the escrow pool is
// plugin state under prefix {20} rather than a signable account, so the
// protocol cannot submit a MessageStake on its own behalf.
//
// Restaking semantics (Canopy):
//   - canoLiq's CNPY is bonded in Canopy validator/delegate records whose
//     `output` address is one canoLiq controls. Each record lists which
//     committees it serves via lib.Validator.committees[]. Same bond,
//     multiple committees.
//   - canoLiq's exposure to committee `c` is therefore:
//
//       exposure[c] = Σ staked_amount  over OWNED records whose
//                                      committees[] contains c
//
//     The same bond counts toward every committee it serves — that's the
//     point of restaking.
//
// Owned, not "every operator on the committee". A whitelisted operator's
// bond is their own collateral: WP §1.1 has them bonding their own CNPY,
// and the risk section has that same CNPY slashed rather than depositors'.
// Summing operator bonds here would overstate canoLiq's exposure by
// whatever the operators put up themselves, and it is the same conflation
// that let the reward sweep credit cCNPY holders with an entire foreign
// validator's emission (see reward.go::ProcessRewards). The ownership
// verdict comes from ValidatorRegistryEntry.Owned, so this view and the
// reward sweep cannot disagree about whose stake it is.

// CommitteeAllocation reports canoLiq's observed exposure to one Canopy
// committee plus any policy drift against the matching policy entry.
// DriftBps is signed: negative = below target weight, positive = above.
// UnderMin / OverMax are independent absolute-bound flags — drift can be
// negative while observed stake still sits above min_stake_ucnpy, and
// vice versa.
type CommitteeAllocation struct {
	CommitteeID uint64 `json:"committeeId"`
	StakeUcnpy  uint64 `json:"stakeUcnpy"`
	WeightBps   uint64 `json:"weightBps"`           // observed / total observed exposure (bps)
	TargetBps   uint64 `json:"targetBps,omitempty"` // 0 when no policy entry
	DriftBps    int64  `json:"driftBps,omitempty"`  // weightBps - targetBps
	UnderMin    bool   `json:"underMin,omitempty"`  // observed < min_stake_ucnpy
	OverMax     bool   `json:"overMax,omitempty"`   // observed > max_stake_ucnpy
}

// RestakingView is the /v1/restaking response: total observed exposure,
// the policy declaration (may be empty), the per-committee allocation list
// (sorted by committee id), and a flag that surfaces whether the policy
// is in compliance (within-min, within-max, weight-bps drift below the
// driftWarnBps threshold).
type RestakingView struct {
	TotalExposureUcnpy uint64                           `json:"totalExposureUcnpy"`
	Policy             []*contract.RestakingPolicyEntry `json:"policy"`
	Allocations        []CommitteeAllocation            `json:"allocations"`
	// PolicyCompliant is true when no allocation reports UnderMin / OverMax.
	// (Weight-bps drift is informational only — observation-only mode can't
	// correct it without active rebalancing.)
	PolicyCompliant bool `json:"policyCompliant"`
}

// The per-committee exposure map (Snapshot.CurrentRestakingAllocation)
// is populated by refreshSnapshot inline — see snapshot.go's qCanopyVal
// branch in Batch 2, which fans out one KeyForValidator read per
// registered operator alongside the existing proposal/spend/staker reads
// (no extra round-trip). This keeps the exposure derivation on the
// snapshot path it ultimately feeds (QueryRestaking) and avoids a
// second per-operator round-trip that a standalone helper would incur.

// buildRestakingView assembles the /v1/restaking response from the policy
// + observed exposure. Drift bps is computed against the *observed* total
// exposure (not against any abstract "target stake"): the policy declares
// shares, and the report says how the current operator mix divides into
// those shares.
func buildRestakingView(policy []*contract.RestakingPolicyEntry, observed map[uint64]uint64) *RestakingView {
	view := &RestakingView{Policy: policy, PolicyCompliant: true}

	var total uint64
	for _, v := range observed {
		total += v
	}
	view.TotalExposureUcnpy = total

	policyByCommittee := make(map[uint64]*contract.RestakingPolicyEntry, len(policy))
	for _, p := range policy {
		if p == nil {
			continue
		}
		policyByCommittee[p.CommitteeId] = p
	}

	// Union of observed committees and policy committees so the report
	// covers (a) committees we have exposure to without a policy entry
	// (drift purely informational, no target) and (b) policy entries with
	// no observed exposure (under-target / under-min).
	covered := make(map[uint64]bool, len(observed)+len(policy))
	for k := range observed {
		covered[k] = true
	}
	for k := range policyByCommittee {
		covered[k] = true
	}

	ids := make([]uint64, 0, len(covered))
	for id := range covered {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		stake := observed[id]
		var weightBps uint64
		if total > 0 {
			weightBps = mulDiv(stake, 10_000, total)
		}
		alloc := CommitteeAllocation{CommitteeID: id, StakeUcnpy: stake, WeightBps: weightBps}
		if p, ok := policyByCommittee[id]; ok {
			alloc.TargetBps = p.TargetWeightBps
			alloc.DriftBps = int64(weightBps) - int64(p.TargetWeightBps)
			if p.MinStakeUcnpy > 0 && stake < p.MinStakeUcnpy {
				alloc.UnderMin = true
				view.PolicyCompliant = false
			}
			if p.MaxStakeUcnpy > 0 && stake > p.MaxStakeUcnpy {
				alloc.OverMax = true
				view.PolicyCompliant = false
			}
		}
		view.Allocations = append(view.Allocations, alloc)
	}
	return view
}
