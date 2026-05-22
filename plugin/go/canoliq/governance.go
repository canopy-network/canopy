package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/types/known/anypb"
)

// governance.go drives the canoLiq DAO proposal lifecycle: create, vote,
// tally on expiry, and dispatch the typed payload (param change | buyback |
// treasury_spend) on pass. WP §4 specifies stake-locked CLIQ as the
// governance weight source; voter weight is read against the
// CLIQStake.staked_at_height snapshot taken at proposal creation, so flash
// stake additions cannot retroactively gain voting power.

// CheckMessageCLIQProposalCreate validates a create_proposal request statelessly.
func (c *Canoliq) CheckMessageCLIQProposalCreate(msg *contract.MessageCLIQProposalCreate, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Payload == nil {
		return &contract.PluginCheckResponse{Error: ErrInvalidProposalPayload()}
	}
	if fee < params.ProposalFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCLIQVote validates a vote statelessly.
func (c *Canoliq) CheckMessageCLIQVote(msg *contract.MessageCLIQVote, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Choice != contract.VoteChoice_VOTE_YES && msg.Choice != contract.VoteChoice_VOTE_NO && msg.Choice != contract.VoteChoice_VOTE_ABSTAIN {
		return &contract.PluginCheckResponse{Error: ErrInvalidProposalPayload()}
	}
	if fee < params.VoteFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// DeliverMessageCLIQProposalCreate opens a new proposal: assigns the next
// id, snapshots globals.total_staked_cliq into the record, and appends to
// the active proposal index.
func (c *Canoliq) DeliverMessageCLIQProposalCreate(msg *contract.MessageCLIQProposalCreate, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	stakeKey := KeyForCLIQStake(msg.FromAddress)
	gKey := KeyForGlobals()
	idxKey := KeyForProposalIndex()
	cQ, fQ, sQ, gQ, iQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: gQ, Key: gKey},
			{QueryId: iQ, Key: idxKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	stake := new(contract.CLIQStake)
	globals := new(contract.CanoliqGlobals)
	idx := new(contract.ProposalIndex)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	if stake.Amount < params.MinStakeToPropose {
		return &contract.PluginDeliverResponse{Error: ErrBelowProposalMinStake()}
	}
	// Validate payload type up-front; ExecuteProposal does the same check on
	// pass, but rejecting at create avoids accumulating undispatchable proposals.
	payload, err := unwrapPayload(msg.Payload)
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	height := c.currentHeight()
	id := globals.NextProposalId + 1
	globals.NextProposalId = id
	// Classify the proposal and snapshot its governance tier so tally,
	// timelock, and voting window are fixed at creation even if params change
	// mid-flight. Unmatched actions (e.g. buyback) leave tier nil and fall
	// back to the scalar voting-period / quorum / timelock knobs.
	action := actionTypeForPayload(payload)
	tier := tierFor(params, action)
	votingPeriod := params.VotingPeriodBlocks
	if tier != nil && tier.VotingPeriodBlocks > 0 {
		votingPeriod = tier.VotingPeriodBlocks
	}
	prop := &contract.Proposal{
		Id:                  id,
		Proposer:            msg.FromAddress,
		CreationHeight:      height,
		ExpiryHeight:        height + votingPeriod,
		SnapshotTotalStaked: globals.TotalStakedCliq,
		Payload:             msg.Payload,
		Description:         msg.Description,
		Status:              contract.ProposalStatus_PROPOSAL_ACTIVE,
		ActionType:          action,
		Tier:                tier,
	}
	idx.Ids = append(idx.Ids, id)
	pBz, e := contract.Marshal(prop)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: KeyForProposal(id), Value: pBz},
		{Key: gKey, Value: gBz},
		{Key: idxKey, Value: idxBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageCLIQVote records a single yes/no/abstain vote on an active
// proposal. Voter weight is the CLIQStake.amount evaluated against
// proposal.creation_height — stake whose staked_at_height postdates the
// proposal earns zero weight.
func (c *Canoliq) DeliverMessageCLIQVote(msg *contract.MessageCLIQVote, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	pKey := KeyForProposal(msg.ProposalId)
	stakeKey := KeyForCLIQStake(msg.FromAddress)
	voteKey := KeyForVote(msg.ProposalId, msg.FromAddress)
	cQ, fQ, pQ, sQ, vQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: pQ, Key: pKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: vQ, Key: voteKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	prop := new(contract.Proposal)
	stake := new(contract.CLIQStake)
	pPresent, votePresent := false, false
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case pQ:
			if e := contract.Unmarshal(r.Entries[0].Value, prop); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			pPresent = prop.Id != 0
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case vQ:
			votePresent = len(r.Entries[0].Value) > 0
		}
	}
	if !pPresent {
		return &contract.PluginDeliverResponse{Error: ErrProposalNotFound()}
	}
	if prop.Status != contract.ProposalStatus_PROPOSAL_ACTIVE {
		return &contract.PluginDeliverResponse{Error: ErrProposalInactive()}
	}
	if c.currentHeight() >= prop.ExpiryHeight {
		return &contract.PluginDeliverResponse{Error: ErrProposalInactive()}
	}
	if votePresent {
		return &contract.PluginDeliverResponse{Error: ErrAlreadyVoted()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	weight := uint64(0)
	if stake.Address != nil {
		if stake.StakedAtHeight > prop.CreationHeight {
			// Voter staked after proposal creation — zero weight per snapshot.
			return &contract.PluginDeliverResponse{Error: ErrStakeAfterCreation()}
		}
		weight = stake.Amount
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	switch msg.Choice {
	case contract.VoteChoice_VOTE_YES:
		prop.YesWeight += weight
	case contract.VoteChoice_VOTE_NO:
		prop.NoWeight += weight
	case contract.VoteChoice_VOTE_ABSTAIN:
		prop.AbstainWeight += weight
	default:
		return &contract.PluginDeliverResponse{Error: ErrInvalidProposalPayload()}
	}
	vote := &contract.Vote{ProposalId: msg.ProposalId, Voter: msg.FromAddress, Choice: msg.Choice, Weight: weight}
	vBz, e := contract.Marshal(vote)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	pBz, e := contract.Marshal(prop)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: voteKey, Value: vBz},
		{Key: pKey, Value: pBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// processProposals is the BeginBlock hook. For each active proposal whose
// expiry has elapsed, tally yes/no/abstain weights, apply the quorum +
// pass-threshold rules, dispatch payload on pass (synchronously for param
// changes; for buyback / treasury_spend it parks an executable record), and
// remove the proposal + its votes from state.
func (c *Canoliq) processProposals(_ uint64) *contract.PluginError {
	params, err := c.LoadParams()
	if err != nil {
		return err
	}
	idxKey := KeyForProposalIndex()
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: idxKey}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	idx := new(contract.ProposalIndex)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
			return e
		}
	}
	if len(idx.Ids) == 0 {
		return nil
	}
	height := c.currentHeight()
	survivors := idx.Ids[:0]
	for _, id := range idx.Ids {
		prop, err := c.loadProposal(id)
		if err != nil {
			return err
		}
		if prop == nil {
			// Stale index entry; skip.
			continue
		}
		if height < prop.ExpiryHeight {
			survivors = append(survivors, id)
			continue
		}
		passed := proposalPasses(prop, params)
		if passed {
			prop.Status = contract.ProposalStatus_PROPOSAL_PASSED
			if err := c.dispatchPassed(prop, params, height); err != nil {
				return err
			}
		} else {
			prop.Status = contract.ProposalStatus_PROPOSAL_FAILED
		}
		if err := c.cleanupProposal(prop); err != nil {
			return err
		}
	}
	idx.Ids = survivors
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: idxKey, Value: idxBz}},
	}); err != nil {
		return err
	}
	return nil
}

// loadProposal reads a Proposal record. Returns (nil, nil) when absent.
func (c *Canoliq) loadProposal(id uint64) (*contract.Proposal, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForProposal(id)}},
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
	prop := new(contract.Proposal)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, prop); e != nil {
		return nil, e
	}
	return prop, nil
}

// proposalPasses applies the quorum + pass threshold rules to a closed proposal.
// Quorum: yes + no + abstain >= quorum_bps * snapshot / 10000.
// Threshold: yes >= pass_threshold_bps * (yes + no) / 10000. Empty (yes+no)
// fails by default.
func proposalPasses(p *contract.Proposal, params *contract.CanoliqParams) bool {
	// Prefer the tier snapshotted at creation; fall back to the scalar knobs
	// for proposals created before T1 (nil tier) or unmatched actions.
	quorumBps, approvalBps := params.QuorumBps, params.PassThresholdBps
	if p.Tier != nil {
		quorumBps, approvalBps = p.Tier.QuorumBps, p.Tier.ApprovalBps
	}
	total := p.YesWeight + p.NoWeight + p.AbstainWeight
	required := mulDiv(p.SnapshotTotalStaked, quorumBps, 10_000)
	if total < required {
		return false
	}
	denom := p.YesWeight + p.NoWeight
	if denom == 0 {
		return false
	}
	threshold := mulDiv(denom, approvalBps, 10_000)
	return p.YesWeight >= threshold
}

// dispatchPassed reacts to a passed proposal by either applying the param
// change synchronously or queuing the buyback / treasury_spend artifact.
func (c *Canoliq) dispatchPassed(prop *contract.Proposal, params *contract.CanoliqParams, height uint64) *contract.PluginError {
	payload, err := unwrapPayload(prop.Payload)
	if err != nil {
		return err
	}
	switch p := payload.(type) {
	case *contract.ProposalParamChange:
		if p.Params == nil {
			return ErrInvalidProposalPayload()
		}
		if err := ValidateParams(p.Params); err != nil {
			return err
		}
		return c.SaveParams(p.Params)
	case *contract.ProposalBuyback:
		if p.PriceMicroCnpyPerCliq == 0 || p.CnpyAmount == 0 {
			return ErrInvalidProposalPayload()
		}
		order := &contract.BuybackOrder{
			ProposalId: prop.Id,
			CnpyDrawn:  0,
			Mode:       p.Mode,
			Executed:   false,
			Payload:    p,
		}
		bz, e := contract.Marshal(order)
		if e != nil {
			return e
		}
		if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
			Sets: []*contract.PluginSetOp{{Key: KeyForBuybackOrder(prop.Id), Value: bz}},
		}); err != nil {
			return err
		}
		return nil
	case *contract.ProposalTreasurySpend:
		return c.queueTreasurySpend(prop, p, params, height)
	case *contract.ProposalValidatorEject:
		// F12: drop the validator from the committee registry and clear its
		// accrued incentives. Idempotent.
		return c.ejectValidator(p.ValidatorAddress)
	case *contract.ProposalEmergency:
		// F13: emergency actions run on the fast-track tier (24h vote, no
		// timelock). The optional param diff applies immediately on pass.
		if p.ParamChange != nil && p.ParamChange.Params != nil {
			if err := ValidateParams(p.ParamChange.Params); err != nil {
				return err
			}
			return c.SaveParams(p.ParamChange.Params)
		}
		return nil
	case *contract.ProposalProtocolUpgrade:
		// Coordinated off-chain; recorded for audit, no on-chain dispatch.
		return nil
	default:
		return ErrUnknownProposalPayload()
	}
}

// cleanupProposal removes the proposal record + per-voter vote records from
// state. Caller is responsible for trimming the proposal index list.
func (c *Canoliq) cleanupProposal(prop *contract.Proposal) *contract.PluginError {
	deletes := []*contract.PluginDeleteOp{{Key: KeyForProposal(prop.Id)}}
	// Vote records are per-(proposal, voter); without a per-proposal voter
	// index we cannot enumerate them. We do not persist a voter list because
	// votes are accessed only by direct key. Stale vote keys after proposal
	// deletion are inert (no proposal id to look them up by) and reclaimable
	// by future indexers if needed.
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Deletes: deletes}); err != nil {
		return err
	}
	return nil
}

// unwrapPayload unmarshals an Any payload into one of the supported proposal
// payload types.
func unwrapPayload(any *anypb.Any) (interface{}, *contract.PluginError) {
	if any == nil {
		return nil, ErrInvalidProposalPayload()
	}
	msg, err := contract.FromAny(any)
	if err != nil {
		return nil, err
	}
	switch msg.(type) {
	case *contract.ProposalParamChange, *contract.ProposalBuyback, *contract.ProposalTreasurySpend,
		*contract.ProposalValidatorEject, *contract.ProposalEmergency, *contract.ProposalProtocolUpgrade:
		return msg, nil
	default:
		return nil, ErrUnknownProposalPayload()
	}
}

// largeSpendCliqThreshold is the "> 1M CLIQ" boundary (in uCLIQ) separating
// the small- and large-treasury-spend governance tiers (Tokenomics §7).
const largeSpendCliqThreshold = 1_000_000 * 1_000_000

// actionTypeForPayload classifies an unwrapped proposal payload into its
// governance ActionType. ProposalBuyback has no §7 tier and maps to
// ACTION_UNKNOWN so tier resolution falls back to the scalar knobs.
func actionTypeForPayload(payload interface{}) contract.ActionType {
	switch p := payload.(type) {
	case *contract.ProposalParamChange:
		return contract.ActionType_ACTION_FEE_CHANGE
	case *contract.ProposalTreasurySpend:
		if p.Denomination == contract.SpendDenomination_SPEND_CLIQ && p.Amount > largeSpendCliqThreshold {
			return contract.ActionType_ACTION_TREASURY_SPEND_LARGE
		}
		return contract.ActionType_ACTION_TREASURY_SPEND_SMALL
	case *contract.ProposalValidatorEject:
		return contract.ActionType_ACTION_VALIDATOR_EJECT
	case *contract.ProposalEmergency:
		return contract.ActionType_ACTION_EMERGENCY
	case *contract.ProposalProtocolUpgrade:
		return contract.ActionType_ACTION_PROTOCOL_UPGRADE
	default:
		return contract.ActionType_ACTION_UNKNOWN
	}
}
