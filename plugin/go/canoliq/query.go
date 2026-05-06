package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// query.go implements read-only state accessors that back the HTTP query
// surface defined in rpc.go. All helpers take a per-request *Canoliq and
// issue StateRead calls — they never mutate state.
//
// Plugin state is point-read only (the FSM does not expose range scans), so
// collection-style queries depend on the explicit indexes already maintained
// by the write paths: VestingIndex, ProposalIndex, CLIQStakeIndex, and the
// ProposalIndex-shaped spend index. Queries that would otherwise need a scan
// (e.g. all redemptions for an address) require an id to be passed in.

// AccountView is the composite per-address snapshot returned by /v1/account.
// It pulls together CNPY (from the core Account record), cCNPY, liquid CLIQ,
// staked CLIQ, vesting schedules with cumulative unlocked-at-current-height,
// and the validator-incentives accrual. Pending unstakes and redemptions are
// not included because there is no per-address index for them — callers
// looking those up should use /v1/redemption/{addr}/{id} directly.
type AccountView struct {
	Address           string                       `json:"address"`
	CNPY              uint64                       `json:"cnpy"`
	CCNPY             uint64                       `json:"ccnpy"`
	CLIQLiquid        uint64                       `json:"cliqLiquid"`
	CLIQStake         *contract.CLIQStake          `json:"cliqStake,omitempty"`
	ValidatorIncentive uint64                      `json:"validatorIncentive"`
	Vestings          []*VestingView               `json:"vestings,omitempty"`
}

// VestingView is one vesting schedule annotated with the cumulative unlocked
// amount at the plugin's current height (not just claimed_amount).
type VestingView struct {
	Schedule        *contract.VestingSchedule `json:"schedule"`
	UnlockedToDate  uint64                    `json:"unlockedToDate"`
	ClaimableNow    uint64                    `json:"claimableNow"`
	CurrentHeight   uint64                    `json:"currentHeight"`
}

// PoolsView aggregates the canoliq-owned scalar buckets and the on-chain
// committee fee pool.
type PoolsView struct {
	CommitteePool       uint64               `json:"committeePool"`
	TreasuryCNPY        uint64               `json:"treasuryCnpy"`
	TreasuryCLIQ        uint64               `json:"treasuryCliq"`
	BuybackPool         uint64               `json:"buybackPool"`
	InsurancePool       uint64               `json:"insurancePool"`
	ValidatorIncentives []ValidatorIncentive `json:"validatorIncentives"`
}

// ValidatorIncentive is one entry of the per-validator incentives ledger.
type ValidatorIncentive struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

// MultisigApprovalsView lists the live approvals against a spend, filtered
// against the *current* signer set so stale approvals from removed signers
// do not appear.
type MultisigApprovalsView struct {
	SpendID   uint64                       `json:"spendId"`
	Threshold uint64                       `json:"threshold"`
	Approvals []*contract.MultisigApproval `json:"approvals"`
}

// HealthView is the liveness response: just enough to confirm the plugin is
// up, what height it has observed, and whether genesis has run.
type HealthView struct {
	Height          uint64 `json:"height"`
	GenesisComplete bool   `json:"genesisComplete"`
	ChainID         uint64 `json:"chainId"`
}

// QueryGlobals returns the singleton CanoliqGlobals record. Empty fields when
// genesis has not yet run.
func (c *Canoliq) QueryGlobals() (*contract.CanoliqGlobals, *contract.PluginError) {
	return c.LoadGlobals()
}

// QueryParams returns the current canoLiq parameters.
func (c *Canoliq) QueryParams() (*contract.CanoliqParams, *contract.PluginError) {
	return c.LoadParams()
}

// QueryAccount assembles the composite per-address view.
func (c *Canoliq) QueryAccount(addr []byte) (*AccountView, *contract.PluginError) {
	if len(addr) != 20 {
		return nil, ErrInvalidAddress()
	}
	cnpyKey := contract.KeyForAccount(addr)
	ccnpyKey := KeyForCCNPYBalance(addr)
	cliqBalKey := KeyForCLIQBalance(addr)
	stakeKey := KeyForCLIQStake(addr)
	valIncentKey := KeyForValidatorIncentives(addr)
	vestIdxKey := KeyForVestingIndex(addr)
	cQ, ccQ, lcQ, sQ, vQ, viQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: ccQ, Key: ccnpyKey},
			{QueryId: lcQ, Key: cliqBalKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: viQ, Key: valIncentKey},
			{QueryId: vQ, Key: vestIdxKey},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	view := &AccountView{Address: hexAddress(addr)}
	var vestIdxBz []byte
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
			view.CLIQLiquid = DecodeUint64(raw)
		case sQ:
			stake := new(contract.CLIQStake)
			if e := contract.Unmarshal(raw, stake); e != nil {
				return nil, e
			}
			if stake.Address != nil {
				view.CLIQStake = stake
			}
		case viQ:
			view.ValidatorIncentive = DecodeUint64(raw)
		case vQ:
			vestIdxBz = raw
		}
	}
	if len(vestIdxBz) > 0 {
		idx := new(contract.VestingIndex)
		if e := contract.Unmarshal(vestIdxBz, idx); e != nil {
			return nil, e
		}
		vestings, err := c.queryVestingSchedules(addr, idx.ScheduleIds)
		if err != nil {
			return nil, err
		}
		view.Vestings = vestings
	}
	return view, nil
}

// QueryPools aggregates the plugin-owned scalar buckets and walks the
// validator registry to itemize per-validator incentive accruals. Falls back
// to the legacy aggregator address when the registry is empty.
func (c *Canoliq) QueryPools() (*PoolsView, *contract.PluginError) {
	poolKey := contract.KeyForFeePool(c.Config.ChainId)
	tCnpyKey := KeyForTreasuryCNPY()
	tCliqKey := KeyForTreasuryCLIQ()
	bbKey := KeyForBuybackPool()
	insKey := KeyForInsurancePool()
	regKey := KeyForValidatorRegistry()
	pQ, tcQ, tlQ, bQ, iQ, rQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: pQ, Key: poolKey},
			{QueryId: tcQ, Key: tCnpyKey},
			{QueryId: tlQ, Key: tCliqKey},
			{QueryId: bQ, Key: bbKey},
			{QueryId: iQ, Key: insKey},
			{QueryId: rQ, Key: regKey},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	view := &PoolsView{}
	registry := new(contract.ValidatorRegistry)
	hasRegistry := false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		raw := r.Entries[0].Value
		switch r.QueryId {
		case pQ:
			pool := new(contract.Pool)
			if e := contract.Unmarshal(raw, pool); e != nil {
				return nil, e
			}
			view.CommitteePool = pool.Amount
		case tcQ:
			view.TreasuryCNPY = DecodeUint64(raw)
		case tlQ:
			view.TreasuryCLIQ = DecodeUint64(raw)
		case bQ:
			view.BuybackPool = DecodeUint64(raw)
		case iQ:
			view.InsurancePool = DecodeUint64(raw)
		case rQ:
			if e := contract.Unmarshal(raw, registry); e != nil {
				return nil, e
			}
			hasRegistry = true
		}
	}
	addrs := make([][]byte, 0)
	if hasRegistry {
		for _, e := range registry.Entries {
			addrs = append(addrs, e.Address)
		}
	}
	if len(addrs) == 0 {
		// Legacy aggregator path — read the single fallback bucket.
		addrs = append(addrs, c.committeeAggregatorAddr())
	}
	for _, a := range addrs {
		amt := c.readScalar(KeyForValidatorIncentives(a))
		if amt == 0 {
			continue
		}
		view.ValidatorIncentives = append(view.ValidatorIncentives, ValidatorIncentive{
			Address: hexAddress(a),
			Amount:  amt,
		})
	}
	return view, nil
}

// QueryProposal returns a single proposal by id, or (nil, nil) if absent.
func (c *Canoliq) QueryProposal(id uint64) (*contract.Proposal, *contract.PluginError) {
	return c.loadProposal(id)
}

// QueryProposalIndex returns the active proposal id list. Empty slice when
// no proposals are pending.
func (c *Canoliq) QueryProposalIndex() ([]uint64, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForProposalIndex()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	idx := new(contract.ProposalIndex)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
		return nil, e
	}
	return idx.Ids, nil
}

// QueryVote returns one vote record for (proposalId, voter) or (nil, nil).
func (c *Canoliq) QueryVote(proposalID uint64, voter []byte) (*contract.Vote, *contract.PluginError) {
	if len(voter) != 20 {
		return nil, ErrInvalidAddress()
	}
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
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	v := new(contract.Vote)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, v); e != nil {
		return nil, e
	}
	return v, nil
}

// QueryBuybackOrder returns the post-execution buyback receipt for a
// proposal id, or (nil, nil) when not present.
func (c *Canoliq) QueryBuybackOrder(proposalID uint64) (*contract.BuybackOrder, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForBuybackOrder(proposalID)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	order := new(contract.BuybackOrder)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, order); e != nil {
		return nil, e
	}
	return order, nil
}

// QueryTreasurySpend returns the spend by id (or nil, nil if not found).
func (c *Canoliq) QueryTreasurySpend(id uint64) (*contract.TreasurySpend, *contract.PluginError) {
	return c.loadSpend(id)
}

// QuerySpendIndex returns pending spend ids. Empty when none queued.
func (c *Canoliq) QuerySpendIndex() ([]uint64, *contract.PluginError) {
	idx, err := c.loadSpendIndex()
	if err != nil {
		return nil, err
	}
	return idx.Ids, nil
}

// QueryMultisigApprovals lists the approvals against a spend, filtered to
// signers currently in params.multisig_signers.
func (c *Canoliq) QueryMultisigApprovals(spendID uint64) (*MultisigApprovalsView, *contract.PluginError) {
	params, err := c.LoadParams()
	if err != nil {
		return nil, err
	}
	view := &MultisigApprovalsView{SpendID: spendID, Threshold: params.MultisigThreshold}
	for _, signer := range params.MultisigSigners {
		raw := c.readBytes(KeyForMultisigApproval(spendID, signer))
		if len(raw) == 0 {
			continue
		}
		approval := new(contract.MultisigApproval)
		if e := contract.Unmarshal(raw, approval); e != nil {
			return nil, e
		}
		view.Approvals = append(view.Approvals, approval)
	}
	return view, nil
}

// QueryValidatorRegistry returns the singleton registry, or an empty one.
func (c *Canoliq) QueryValidatorRegistry() (*contract.ValidatorRegistry, *contract.PluginError) {
	reg, err := c.loadValidatorRegistry()
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return &contract.ValidatorRegistry{}, nil
	}
	return reg, nil
}

// QueryRedemption returns one redemption record by (address, id).
func (c *Canoliq) QueryRedemption(addr []byte, id uint64) (*contract.Redemption, *contract.PluginError) {
	if len(addr) != 20 {
		return nil, ErrInvalidAddress()
	}
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
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	red := new(contract.Redemption)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, red); e != nil {
		return nil, e
	}
	return red, nil
}

// QueryVesting returns every vesting schedule for an address with the
// cumulative unlocked-to-date computed against the plugin's current height.
func (c *Canoliq) QueryVesting(addr []byte) ([]*VestingView, *contract.PluginError) {
	if len(addr) != 20 {
		return nil, ErrInvalidAddress()
	}
	idxKey := KeyForVestingIndex(addr)
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: idxKey}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	idx := new(contract.VestingIndex)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
		return nil, e
	}
	return c.queryVestingSchedules(addr, idx.ScheduleIds)
}

// queryVestingSchedules batch-loads the schedule records named by ids and
// annotates each with cumulative unlocked + claimable-now relative to the
// current height.
func (c *Canoliq) queryVestingSchedules(addr []byte, ids []uint64) ([]*VestingView, *contract.PluginError) {
	if len(ids) == 0 {
		return nil, nil
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
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	height := c.currentHeight()
	views := make([]*VestingView, 0, len(ids))
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		sched := new(contract.VestingSchedule)
		if e := contract.Unmarshal(r.Entries[0].Value, sched); e != nil {
			return nil, e
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
	return views, nil
}

// QueryHealth returns liveness info — height + genesis flag + chain id.
func (c *Canoliq) QueryHealth() (*HealthView, *contract.PluginError) {
	g, err := c.LoadGlobals()
	if err != nil {
		return nil, err
	}
	return &HealthView{
		Height:          c.currentHeight(),
		GenesisComplete: g.GenesisComplete,
		ChainID:         c.Config.ChainId,
	}, nil
}

// hexAddress encodes a 20-byte address as a 0x-prefixed lowercase hex string.
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
