package canoliq

import (
	"sync/atomic"

	"github.com/canopy-network/go-plugin/contract"
)

// snapshot.go maintains a plugin-side snapshot of canoliq-owned state for
// the read-only HTTP query layer (rpc.go). The Canopy FSM rejects
// plugin-initiated StateReads whose request id is not from an in-flight
// FSM-originated lifecycle call (CheckTx/Deliver/Begin/End). Freestanding
// reads from an HTTP handler therefore cannot work — instead we batch a
// canoliq-owned read inside EndBlock (where c.fsmId is the FSM-originated
// EndBlock id, valid for the duration of the call) and freeze the result.
// HTTP handlers serve from this frozen value with no plugin↔FSM round-trip.
//
// Scope: every entity reachable from a singleton key or a plugin-maintained
// index. Per-address records (Account, cCNPY/CPLQ balance, vesting, queued
// redemptions, queued unstakes) are out of scope here — there is no
// canoliq-side index that names every address that has ever had a balance,
// so a snapshot cannot enumerate them. Per-address queries are deferred to
// Phase 3 §1.1.

// Snapshot is a frozen, point-in-time copy of canoliq-owned state. Every
// field is owned by the snapshot — query helpers may return pointers to its
// values without copying. EndBlock builds a new Snapshot and atomically
// swaps it in, so reads and writes never overlap.
type Snapshot struct {
	Height              uint64
	Globals             *contract.CanoliqGlobals
	Params              *contract.CanoliqParams
	CommitteePool       uint64
	TreasuryCNPY        uint64
	TreasuryCPLQ        uint64
	BuybackPool         uint64
	InsurancePool       uint64
	ValidatorIncentives map[string]uint64
	ValidatorRegistry   *contract.ValidatorRegistry
	ActiveProposalIDs   []uint64
	Proposals           map[uint64]*contract.Proposal
	PendingSpendIDs     []uint64
	Spends              map[uint64]*contract.TreasurySpend
	StakerAddresses     [][]byte
	Stakers             map[string]*contract.CPLQStake
	MultisigApprovals   map[uint64][]*contract.MultisigApproval
}

// emptySnapshot is returned to query helpers when EndBlock has not yet run
// (or genesis is incomplete). All fields are zero values so HTTP responses
// remain well-formed during cold start.
func emptySnapshot() *Snapshot {
	return &Snapshot{
		Globals:             &contract.CanoliqGlobals{},
		Params:              DefaultParams(),
		ValidatorIncentives: map[string]uint64{},
		ValidatorRegistry:   &contract.ValidatorRegistry{},
		Proposals:           map[uint64]*contract.Proposal{},
		Spends:              map[uint64]*contract.TreasurySpend{},
		Stakers:             map[string]*contract.CPLQStake{},
		MultisigApprovals:   map[uint64][]*contract.MultisigApproval{},
	}
}

// Snapshot returns the most recently published snapshot, or an empty one
// when none has been built yet. Safe for concurrent calls.
func (p *Plugin) Snapshot() *Snapshot {
	v := p.snapshot.Load()
	if v == nil {
		return emptySnapshot()
	}
	return v
}

// publishSnapshot atomically swaps the latest snapshot. Called from
// refreshSnapshot inside EndBlock.
func (p *Plugin) publishSnapshot(s *Snapshot) {
	p.snapshot.Store(s)
}

// refreshSnapshot rebuilds the snapshot from the FSM at the given height.
// Must be called from a goroutine that is currently inside an FSM-originated
// lifecycle call so c.fsmId is a valid FSM context. EndBlock is the natural
// caller — its id is live for the entire EndBlock window.
//
// Strategy: batch all bounded singleton/index reads into one StateRead, then
// follow up with id-driven reads for active proposals, pending spends, and
// active stakers. Two RPC round-trips per block — small relative to the
// reward sweep already running here.
func (c *Canoliq) refreshSnapshot(height uint64) *contract.PluginError {
	snap := emptySnapshot()
	snap.Height = height

	// Batch 1: singletons + indexes.
	const (
		qGlobals = iota
		qParams
		qPool
		qTreasuryCNPY
		qTreasuryCPLQ
		qBuyback
		qInsurance
		qRegistry
		qPropIdx
		qSpendIdx
		qStakeIdx
	)
	keys := []*contract.PluginKeyRead{
		{QueryId: qGlobals, Key: KeyForGlobals()},
		{QueryId: qParams, Key: KeyForParams()},
		{QueryId: qPool, Key: contract.KeyForFeePool(c.Config.ChainId)},
		{QueryId: qTreasuryCNPY, Key: KeyForTreasuryCNPY()},
		{QueryId: qTreasuryCPLQ, Key: KeyForTreasuryCPLQ()},
		{QueryId: qBuyback, Key: KeyForBuybackPool()},
		{QueryId: qInsurance, Key: KeyForInsurancePool()},
		{QueryId: qRegistry, Key: KeyForValidatorRegistry()},
		{QueryId: qPropIdx, Key: KeyForProposalIndex()},
		{QueryId: qSpendIdx, Key: KeyForSpendIndex()},
		{QueryId: qStakeIdx, Key: KeyForCPLQStakeIndex()},
	}
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: keys})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	var propIdxBz, spendIdxBz, stakeIdxBz []byte
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		raw := r.Entries[0].Value
		if len(raw) == 0 {
			continue
		}
		switch r.QueryId {
		case qGlobals:
			g := new(contract.CanoliqGlobals)
			if e := contract.Unmarshal(raw, g); e != nil {
				return e
			}
			snap.Globals = g
		case qParams:
			p := new(contract.CanoliqParams)
			if e := contract.Unmarshal(raw, p); e != nil {
				return e
			}
			if err := ValidateParams(p); err == nil {
				snap.Params = p
			}
		case qPool:
			pool := new(contract.Pool)
			if e := contract.Unmarshal(raw, pool); e != nil {
				return e
			}
			snap.CommitteePool = pool.Amount
		case qTreasuryCNPY:
			snap.TreasuryCNPY = DecodeUint64(raw)
		case qTreasuryCPLQ:
			snap.TreasuryCPLQ = DecodeUint64(raw)
		case qBuyback:
			snap.BuybackPool = DecodeUint64(raw)
		case qInsurance:
			snap.InsurancePool = DecodeUint64(raw)
		case qRegistry:
			reg := new(contract.ValidatorRegistry)
			if e := contract.Unmarshal(raw, reg); e != nil {
				return e
			}
			snap.ValidatorRegistry = reg
		case qPropIdx:
			propIdxBz = raw
		case qSpendIdx:
			spendIdxBz = raw
		case qStakeIdx:
			stakeIdxBz = raw
		}
	}

	// Decode the indexes (any one may be absent → leave its slice nil).
	if len(propIdxBz) > 0 {
		idx := new(contract.ProposalIndex)
		if e := contract.Unmarshal(propIdxBz, idx); e != nil {
			return e
		}
		snap.ActiveProposalIDs = idx.Ids
	}
	if len(spendIdxBz) > 0 {
		idx := new(contract.ProposalIndex)
		if e := contract.Unmarshal(spendIdxBz, idx); e != nil {
			return e
		}
		snap.PendingSpendIDs = idx.Ids
	}
	if len(stakeIdxBz) > 0 {
		idx := new(contract.CPLQStakeIndex)
		if e := contract.Unmarshal(stakeIdxBz, idx); e != nil {
			return e
		}
		snap.StakerAddresses = idx.Addresses
	}

	// Validator-incentive accruals: one entry per registry validator. Falls
	// back to the legacy aggregator address when the registry is empty so
	// Phase 1 deployments still surface the share-out.
	addrs := make([][]byte, 0, len(snap.ValidatorRegistry.Entries))
	for _, e := range snap.ValidatorRegistry.Entries {
		addrs = append(addrs, e.Address)
	}
	if len(addrs) == 0 {
		addrs = append(addrs, c.committeeAggregatorAddr())
	}

	// Batch 2: per-id reads (proposals, spends, stakers, validator
	// incentives). Skip when there is nothing to read so we don't issue an
	// empty StateRead.
	keys = keys[:0]
	queryToProp := map[uint64]uint64{}
	for _, id := range snap.ActiveProposalIDs {
		q := qid()
		queryToProp[q] = id
		keys = append(keys, &contract.PluginKeyRead{QueryId: q, Key: KeyForProposal(id)})
	}
	queryToSpend := map[uint64]uint64{}
	for _, id := range snap.PendingSpendIDs {
		q := qid()
		queryToSpend[q] = id
		keys = append(keys, &contract.PluginKeyRead{QueryId: q, Key: KeyForTreasurySpend(id)})
	}
	queryToStaker := map[uint64][]byte{}
	for _, addr := range snap.StakerAddresses {
		q := qid()
		queryToStaker[q] = addr
		keys = append(keys, &contract.PluginKeyRead{QueryId: q, Key: KeyForCPLQStake(addr)})
	}
	queryToValIncent := map[uint64][]byte{}
	for _, addr := range addrs {
		q := qid()
		queryToValIncent[q] = addr
		keys = append(keys, &contract.PluginKeyRead{QueryId: q, Key: KeyForValidatorIncentives(addr)})
	}
	if len(keys) > 0 {
		resp2, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: keys})
		if err != nil {
			return err
		}
		if resp2.Error != nil {
			return resp2.Error
		}
		for _, r := range resp2.Results {
			if len(r.Entries) == 0 {
				continue
			}
			raw := r.Entries[0].Value
			if len(raw) == 0 {
				continue
			}
			if id, ok := queryToProp[r.QueryId]; ok {
				prop := new(contract.Proposal)
				if e := contract.Unmarshal(raw, prop); e != nil {
					return e
				}
				snap.Proposals[id] = prop
				continue
			}
			if id, ok := queryToSpend[r.QueryId]; ok {
				spend := new(contract.TreasurySpend)
				if e := contract.Unmarshal(raw, spend); e != nil {
					return e
				}
				snap.Spends[id] = spend
				continue
			}
			if addr, ok := queryToStaker[r.QueryId]; ok {
				stake := new(contract.CPLQStake)
				if e := contract.Unmarshal(raw, stake); e != nil {
					return e
				}
				snap.Stakers[hexAddress(addr)] = stake
				continue
			}
			if addr, ok := queryToValIncent[r.QueryId]; ok {
				snap.ValidatorIncentives[hexAddress(addr)] = DecodeUint64(raw)
			}
		}
	}

	// Batch 3: multisig approvals — one read per (spend, signer) pair.
	// Bounded by len(MultisigSigners) * len(PendingSpendIDs); even with
	// generous params this stays well under a hundred reads.
	if len(snap.PendingSpendIDs) > 0 && len(snap.Params.GetMultisigSigners()) > 0 {
		approvalKeys := make([]*contract.PluginKeyRead, 0)
		queryToApproval := map[uint64]struct {
			SpendID uint64
			Signer  []byte
		}{}
		for _, spendID := range snap.PendingSpendIDs {
			for _, signer := range snap.Params.MultisigSigners {
				q := qid()
				queryToApproval[q] = struct {
					SpendID uint64
					Signer  []byte
				}{spendID, signer}
				approvalKeys = append(approvalKeys, &contract.PluginKeyRead{
					QueryId: q,
					Key:     KeyForMultisigApproval(spendID, signer),
				})
			}
		}
		if len(approvalKeys) > 0 {
			respA, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{Keys: approvalKeys})
			if err != nil {
				return err
			}
			if respA.Error != nil {
				return respA.Error
			}
			for _, r := range respA.Results {
				if len(r.Entries) == 0 {
					continue
				}
				raw := r.Entries[0].Value
				if len(raw) == 0 {
					continue
				}
				meta, ok := queryToApproval[r.QueryId]
				if !ok {
					continue
				}
				ap := new(contract.MultisigApproval)
				if e := contract.Unmarshal(raw, ap); e != nil {
					return e
				}
				snap.MultisigApprovals[meta.SpendID] = append(
					snap.MultisigApprovals[meta.SpendID], ap)
			}
		}
	}

	c.plugin.publishSnapshot(snap)
	return nil
}

// snapshotPointer is the Plugin-side atomic holder. We use atomic.Pointer
// (Go 1.19+) so HTTP reads are lock-free. Only EndBlock writes; HTTP only
// reads.
type snapshotPointer = atomic.Pointer[Snapshot]
