package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// restaking.go implements the policy + observability scope of WP §7
// (Restaking Optimization). Phase C as scoped: declare the desired
// per-committee allocation, observe the actual exposure derived from
// Canopy state, report the drift via /v1/restaking. Active rebalancing
// (issuing delegation re-routing) is out of scope for now — that requires
// a delegation-routing primitive not yet defined in the codebase.
//
// Restaking semantics (Canopy):
//   - canoLiq pools CNPY from depositors and delegates it to whitelisted
//     operators. Each operator bonds their own CNPY plus the delegated
//     pool share, and lists which Canopy committees they serve via
//     lib.Validator.committees[]. Same bond, multiple committees.
//   - canoLiq's exposure to committee `c` is therefore:
//
//       exposure[c] = Σ operator.staked_amount  for operators whose
//                                                committees[] contains c
//
//     The same operator stake counts toward every committee they serve
//     — that's the point of restaking. canoLiq does NOT directly bond
//     across committees; the committee mix follows from which operators
//     it has delegated to.

// CommitteeAllocation reports canoLiq's observed exposure to one Canopy
// committee plus any policy drift against the matching policy entry.
// DriftBps is signed: negative = below target weight, positive = above.
// UnderMin / OverMax are independent absolute-bound flags — drift can be
// negative while observed stake still sits above min_stake_ucnpy, and
// vice versa.
type CommitteeAllocation struct {
	CommitteeID  uint64 `json:"committeeId"`
	StakeUcnpy   uint64 `json:"stakeUcnpy"`
	WeightBps    uint64 `json:"weightBps"`              // observed / total observed exposure (bps)
	TargetBps    uint64 `json:"targetBps,omitempty"`    // 0 when no policy entry
	DriftBps     int64  `json:"driftBps,omitempty"`     // weightBps - targetBps
	UnderMin     bool   `json:"underMin,omitempty"`     // observed < min_stake_ucnpy
	OverMax      bool   `json:"overMax,omitempty"`      // observed > max_stake_ucnpy
}

// RestakingView is the /v1/restaking response: total observed exposure,
// the policy declaration (may be empty), the per-committee allocation list
// (sorted by committee id), and a flag that surfaces whether the policy
// is in compliance (within-min, within-max, weight-bps drift below the
// driftWarnBps threshold).
type RestakingView struct {
	TotalExposureUcnpy uint64                            `json:"totalExposureUcnpy"`
	Policy             []*contract.RestakingPolicyEntry  `json:"policy"`
	Allocations        []CommitteeAllocation             `json:"allocations"`
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
	sortUint64Asc(ids)

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

// sortUint64Asc is a tiny in-place ascending sort to keep the
// allocation list deterministic without pulling sort.Slice into the
// hot path. Allocation lists are at most a few dozen entries; insertion
// sort is fine.
func sortUint64Asc(xs []uint64) {
	for i := 1; i < len(xs); i++ {
		v := xs[i]
		j := i - 1
		for j >= 0 && xs[j] > v {
			xs[j+1] = xs[j]
			j--
		}
		xs[j+1] = v
	}
}

