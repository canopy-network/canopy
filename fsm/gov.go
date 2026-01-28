package fsm

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
)

/* On-chain governance: parameter changes and treasury subsidies */

// PROPOSAL CODE BELOW

// ApproveProposal() validates a 'GovProposal' message (ex. MsgChangeParameter or MsgDAOTransfer)
// - checks message sent between start height and end height
// - if APPROVE_ALL set or proposal on the APPROVE_LIST then no error
// - else return ErrRejectProposal
func (s *StateMachine) ApproveProposal(msg GovProposal) lib.ErrorI {
	// if height is before start height or height is after end height (both exclusive)
	if s.Height() < msg.GetStartHeight() || s.Height() > msg.GetEndHeight() {
		// reject the proposal
		return ErrRejectProposal()
	}
	// handle the proposal based on config
	switch s.proposeVoteConfig {
	// if approving all proposals
	default:
		// proposal passes
		return nil
	// if rejecting all proposals
	case RejectAllProposals:
		// proposal is rejected
		return ErrRejectProposal()
	// if on the local approve list
	case ProposalApproveList:
		// read the 'approve list' from the data directory
		proposals := make(GovProposals)
		// get the voted from the local proposals.json file in the data directory
		if err := proposals.NewFromFile(s.Config.DataDirPath); err != nil {
			return err
		}
		// check on this specific message for explicit rejection or complete omission
		if value, ok := proposals[msg.GetProposalHash()]; !ok || !value.Approve {
			return ErrRejectProposal()
		}
		// proposal passes
		return nil
	}
}

// PARAMETER CODE BELOW

// UpdateParam() updates a governance parameter keyed by space and name
func (s *StateMachine) UpdateParam(paramSpace, paramName string, value proto.Message) (err lib.ErrorI) {
	// save the previous parameters to check for updates
	previousParams, err := s.GetParams()
	if err != nil {
		return
	}
	// retrieve the space from the string
	var sp ParamSpace
	switch paramSpace {
	case ParamSpaceCons:
		sp, err = s.GetParamsCons()
	case ParamSpaceVal:
		sp, err = s.GetParamsVal()
	case ParamSpaceFee:
		sp, err = s.GetParamsFee()
	case ParamSpaceGov:
		sp, err = s.GetParamsGov()
	default:
		return ErrUnknownParamSpace()
	}
	if err != nil {
		return err
	}
	// set the value based on the type
	switch v := value.(type) {
	case *lib.UInt64Wrapper:
		err = sp.SetUint64(paramName, v.Value)
	case *lib.StringWrapper:
		err = sp.SetString(paramName, v.Value)
	default:
		return ErrUnknownParamType(value)
	}
	if err != nil {
		return err
	}
	// set the param space back in state
	switch paramSpace {
	case ParamSpaceCons:
		err = s.SetParamsCons(sp.(*ConsensusParams))
	case ParamSpaceVal:
		err = s.SetParamsVal(sp.(*ValidatorParams))
	case ParamSpaceFee:
		err = s.SetParamsFee(sp.(*FeeParams))
	case ParamSpaceGov:
		err = s.SetParamsGov(sp.(*GovernanceParams))
	}
	if err != nil {
		return err
	}
	// adjust the state if necessary
	return s.ConformStateToParamUpdate(previousParams)
}

// ConformStateToParamUpdate() ensures the state does not violate the new values of the governance parameters
// - Only MaxCommitteeSize, RootChainId, MinimumStakeForValidators & MinimumStakeForDelegates require an adjustment
// - MinSellOrderSize is purposefully allowed to violate new updates
func (s *StateMachine) ConformStateToParamUpdate(previousParams *Params) lib.ErrorI {
	// retrieve the params from state
	params, err := s.GetParams()
	if err != nil {
		return err
	}
	// reset committee data if requested by consensus param update
	if params.Consensus.ResetCommittee != 0 {
		// reset committee data
		if err = s.OverwriteCommitteeData(&lib.CommitteeData{ChainId: params.Consensus.ResetCommittee}); err != nil {
			return err
		}
		// prune checkpoints
		storeI, ok := s.store.(lib.StoreI)
		if !ok {
			s.log.Errorf(ErrWrongStoreType().Error())
		} else {
			if err = storeI.DeleteCheckpointsForChain(params.Consensus.ResetCommittee); err != nil {
				return err
			}
		}
		// clear the locked dex batch data
		if err = s.Delete(KeyForLockedBatch(params.Consensus.ResetCommittee)); err != nil {
			return err
		}
		// reset the param back to 0
		params.Consensus.ResetCommittee = 0
		if err = s.SetParamsCons(params.Consensus); err != nil {
			return err
		}
	}
	// if root chain id was updated
	if previousParams.Consensus.RootChainId != params.Consensus.RootChainId {
		// get the committee for self chain
		selfCommittee, e := s.GetCommitteeData(s.Config.ChainId)
		if e != nil {
			return e
		}
		// reset the root height updated
		selfCommittee.LastRootHeightUpdated = 0
		// overwrite the committee data in state
		if err = s.OverwriteCommitteeData(selfCommittee); err != nil {
			return err
		}
	}
	// check if minimum stake requirements have increased
	validatorMinStakeIncreased := previousParams.Validator.MinimumStakeForValidators < params.Validator.MinimumStakeForValidators
	delegateMinStakeIncreased := previousParams.Validator.MinimumStakeForDelegates < params.Validator.MinimumStakeForDelegates
	// if either minimum stake has increased, force unstake those below the new minimum
	if validatorMinStakeIncreased || delegateMinStakeIncreased {
		// iterate through all validators and delegates
		if err = s.IterateAndExecute(ValidatorPrefix(), func(key, value []byte) (e lib.ErrorI) {
			// convert bytes into a validator object
			v, e := s.unmarshalValidator(value)
			if e != nil {
				return e
			}
			// set validator to unstaking if below minium
			if _, e = s.SetValidatorUnstakingIfBelowMinimum(v, params.Validator); err != nil {
				return e
			}
			return
		}); err != nil {
			return err
		}
	}
	// check for a change in MaxCommitteeSize
	if previousParams.Validator.MaxCommitteeSize <= params.Validator.MaxCommittees {
		return nil
	}
	// shrinking MaxCommitteeSize must be immediately enforced to ensure no 'grandfathered' in violators
	maxCommitteeSize := int(params.Validator.MaxCommittees)
	// maintain a counter for pseudorandom removal of the 'chain ids'
	var idx int
	// for each validator, remove the excess ids in a pseudorandom fashion
	return s.IterateAndExecute(ValidatorPrefix(), func(key, value []byte) lib.ErrorI {
		// convert bytes into a validator object
		v, e := s.unmarshalValidator(value)
		if e != nil {
			return e
		}
		// check the number of committees for this validator and see if it's above the maximum
		numCommittees := len(v.Committees)
		if numCommittees <= maxCommitteeSize {
			return nil
		}
		// create a variable to hold a copy of the new committees
		newCommittees := make([]uint64, maxCommitteeSize)
		// iterate 'maxCommitteeSize' number of times
		for i := 0; i < maxCommitteeSize; i++ {
			// calculate a pseudorandom index
			startIndex := idx % numCommittees
			// add each element in a circular queue fashion starting at random position determined by idx
			newCommittees[i] = v.Committees[(startIndex+i)%numCommittees]
		}
		// increment the index to further the 'pseuorandom' property
		idx++
		// update the committees or delegations
		if !v.Delegate {
			// update the new committees
			if err = s.UpdateCommittees(crypto.NewAddress(v.Address), v, v.StakedAmount, newCommittees); err != nil {
				return err
			}
		} else {
			// update the delegations
			if err = s.UpdateDelegations(crypto.NewAddress(v.Address), v, v.StakedAmount, newCommittees); err != nil {
				return err
			}
		}
		// update the validator and its committees
		v.Committees = newCommittees
		// set the validator back into state
		return s.SetValidator(v)
	})
}

// SetParams() writes an entire Params object into state
func (s *StateMachine) SetParams(p *Params) lib.ErrorI {
	// set the parameters in the consensus 'space'
	if err := s.SetParamsCons(p.GetConsensus()); err != nil {
		return err
	}
	// set the parameters in the validator 'space'
	if err := s.SetParamsVal(p.GetValidator()); err != nil {
		return err
	}
	// set the parameters in the fee 'space'
	if err := s.SetParamsFee(p.GetFee()); err != nil {
		return err
	}
	// set the parameters in the governance 'space'
	return s.SetParamsGov(p.GetGovernance())
}

// SetParamsCons() sets Consensus params into state
func (s *StateMachine) SetParamsCons(c *ConsensusParams) lib.ErrorI {
	return s.setParams(ParamSpaceCons, c)
}

// SetParamsVal() sets Validator params into state
func (s *StateMachine) SetParamsVal(v *ValidatorParams) lib.ErrorI {
	s.cache.valParams = v
	return s.setParams(ParamSpaceVal, v)
}

// SetParamsGov() sets Governance params into state
func (s *StateMachine) SetParamsGov(g *GovernanceParams) lib.ErrorI {
	return s.setParams(ParamSpaceGov, g)
}

// SetParamsFee() sets Fee params into state
func (s *StateMachine) SetParamsFee(f *FeeParams) lib.ErrorI {
	s.cache.feeParams = f
	return s.setParams(ParamSpaceFee, f)
}

// setParams() converts the ParamSpace into bytes and sets them in state
func (s *StateMachine) setParams(space string, p proto.Message) lib.ErrorI {
	// convert the param object to bytes
	bz, err := lib.Marshal(p)
	if err != nil {
		return err
	}
	// set the bytes under the 'space' for the parameters
	return s.Set(KeyForParams(space), bz)
}

// GetParams() returns the aggregated ParamSpaces in a single Params object
func (s *StateMachine) GetParams() (*Params, lib.ErrorI) {
	// get the consensus parameters from state
	cons, err := s.GetParamsCons()
	if err != nil {
		return nil, err
	}
	// get the validator parameters from state
	val, err := s.GetParamsVal()
	if err != nil {
		return nil, err
	}
	// get the fee parameters from state
	fee, err := s.GetParamsFee()
	if err != nil {
		return nil, err
	}
	// get the governance parameters from state
	gov, err := s.GetParamsGov()
	if err != nil {
		return nil, err
	}
	// return a collective 'parameters' object that holds all the spaces
	// proto copies are needed for update safety
	return &Params{
		Consensus:  proto.Clone(cons).(*ConsensusParams),
		Validator:  proto.Clone(val).(*ValidatorParams),
		Fee:        proto.Clone(fee).(*FeeParams),
		Governance: proto.Clone(gov).(*GovernanceParams),
	}, nil
}

// GetParamsCons() returns the current state of the governance params in the Consensus space
func (s *StateMachine) GetParamsCons() (ptr *ConsensusParams, err lib.ErrorI) {
	// create a new object ref for the consensus params to ensure a non-nil result
	ptr = new(ConsensusParams)
	// get the consensus parameters from state
	err = s.getParams(ParamSpaceCons, ptr, ErrEmptyConsParams)
	// exit
	return
}

// GetParamsVal() returns the current state of the governance params in the Validator space
func (s *StateMachine) GetParamsVal() (ptr *ValidatorParams, err lib.ErrorI) {
	// check cache
	if s.cache.valParams == nil {
		// create a new object ref for the validator params to ensure a non-nil result
		s.cache.valParams = new(ValidatorParams)
		// get the validator parameters from state
		err = s.getParams(ParamSpaceVal, s.cache.valParams, ErrEmptyValParams)
	}
	// exit
	return s.cache.valParams, err
}

// GetParamsGov() returns the current state of the governance params in the Governance space
func (s *StateMachine) GetParamsGov() (ptr *GovernanceParams, err lib.ErrorI) {
	// create a new object ref for the governance params to ensure a non-nil result
	ptr = new(GovernanceParams)
	// get the governance parameters from state
	err = s.getParams(ParamSpaceGov, ptr, ErrEmptyGovParams)
	// exit
	return
}

// GetParamsFee() returns the current state of the governance params in the Fee space
func (s *StateMachine) GetParamsFee() (ptr *FeeParams, err lib.ErrorI) {
	// check cache
	if s.cache.feeParams == nil {
		// create a new object ref for the fee params to ensure a non-nil result
		s.cache.feeParams = new(FeeParams)
		// get the fee parameters from state
		err = s.getParams(ParamSpaceFee, s.cache.feeParams, ErrEmptyFeeParams)
	}
	// exit
	return s.cache.feeParams, err
}

// getParams() is a generic helper function loads the params for a specific ParamSpace into a ptr object
func (s *StateMachine) getParams(space string, ptr any, emptyErr func() lib.ErrorI) (err lib.ErrorI) {
	// get the parameters bytes using the key for the parameter space
	bz, err := s.Get(KeyForParams(space))
	if err != nil {
		return err
	}
	// if the bytes are empty, execute and return the  callback error
	if bz == nil {
		return emptyErr()
	}
	// convert the parameters bytes to the params object reference
	if err = lib.Unmarshal(bz, ptr); err != nil {
		return err
	}
	// exit
	return
}

// POLLING CODE BELOW

// ParsePollTransactions() parses the last valid block for memo commands to execute specialized 'straw polling' functionality
func (s *StateMachine) ParsePollTransactions(b *lib.BlockResult) {
	// create a new object reference to ensure non-nil results
	ap := new(ActivePolls)
	// load the active polls from the json file
	if err := ap.NewFromFile(s.Config.DataDirPath); err != nil {
		return
	}
	// for each transaction in the block
	for _, tx := range b.Transactions {
		// get the public key object
		pub, e := crypto.NewPublicKeyFromBytes(tx.Transaction.Signature.PublicKey)
		if e != nil {
			return
		}
		// check for a poll transaction
		if err := ap.CheckForPollTransaction(pub.Address(), tx.Transaction.Memo, s.Height()); err != nil {
			// simply log the error
			s.log.Error(err.Error())
			// exit
			return
		}
	}
	// save to file: NOTE: this is non-atomic and can be inconsistent with the database
	// but this is a non-critical function that won't cause a consensus failure
	if err := ap.SaveToFile(s.Config.DataDirPath); err != nil {
		// simply log the error
		s.log.Error(err.Error())
		// exit
		return
	}
}

// PollsToResults() coverts the polling objects to a compressed result based on the voting power
func (s *StateMachine) PollsToResults(polls *ActivePolls) (result Poll, err lib.ErrorI) {
	// create a new poll object ref to ensure non-nil results
	result = make(Poll)
	// create caches to span over multiple blocks
	accountCache, valList := map[string]uint64{}, map[string]uint64{} // address -> power (tokens)
	// get the canopy validator set
	members, err := s.GetCommitteeMembers(s.Config.ChainId)
	if err != nil {
		// NOTE: nested-chains may have no validators - so not returning an error here
		return result, nil
	}
	// get the supply
	supply, err := s.GetSupply()
	if err != nil {
		return
	}
	// get the dao account
	dao, err := s.GetPool(lib.DAOPoolID)
	if err != nil {
		return
	}
	// add the canopy validators to the cache
	for _, member := range members.ValidatorSet.ValidatorSet {
		public, _ := crypto.NewPublicKeyFromBytes(member.PublicKey)
		valList[public.Address().String()] = member.VotingPower
	}
	// for each active poll in list
	for proposalHash, addresses := range polls.Polls {
		// initialize the poll result
		r := PollResult{
			ProposalHash: proposalHash,
			ProposalURL:  polls.PollMeta[proposalHash].Url,
			Accounts:     VoteStats{TotalTokens: supply.Total - supply.Staked - dao.Amount},
			Validators:   VoteStats{TotalTokens: members.ValTotalPower},
		}
		// for each vote in the active poll
		for address, approve := range addresses {
			// check if is validator
			valPower, isValidator := valList[address]
			// if address is a validator
			if isValidator {
				// add validator vote to the validators total voted power
				r.Validators.TotalVotedTokens += valPower
				// if the validator approves...
				if approve {
					// add to the approved tokens
					r.Validators.ApproveTokens += valPower
				} else {
					// add to the rejected tokens
					r.Validators.RejectTokens += valPower
				}
			}
			// check the account balance
			accTokens, inCache := accountCache[address]
			// if the account is not in cache
			if !inCache {
				// convert the string into an address object
				addr, _ := crypto.NewAddressFromString(address)
				// get the account from the state
				accTokens, _ = s.GetAccountBalance(addr)
				// set in cache
				accountCache[address] = accTokens
			}
			// add account vote to the accounts total voted power
			r.Accounts.TotalVotedTokens += accTokens
			// if the account approves...
			if approve {
				// add to the approved tokens
				r.Accounts.ApproveTokens += accTokens
			} else {
				// add to the rejected tokens
				r.Accounts.RejectTokens += accTokens
			}
		}
		// calculate stats for validators
		r.Validators.ApprovePercentage = uint64(float64(r.Validators.ApproveTokens) / float64(r.Validators.TotalTokens) * 100)
		r.Validators.RejectPercentage = uint64(float64(r.Validators.RejectTokens) / float64(r.Validators.TotalTokens) * 100)
		r.Validators.VotedPercentage = uint64(float64(r.Validators.ApproveTokens+r.Validators.RejectTokens) / float64(r.Validators.TotalTokens) * 100)
		// calculate stats for accounts
		r.Accounts.ApprovePercentage = uint64(float64(r.Accounts.ApproveTokens) / float64(r.Accounts.TotalTokens) * 100)
		r.Accounts.RejectPercentage = uint64(float64(r.Accounts.RejectTokens) / float64(r.Accounts.TotalTokens) * 100)
		r.Accounts.VotedPercentage = uint64(float64(r.Accounts.ApproveTokens+r.Accounts.RejectTokens) / float64(r.Accounts.TotalTokens) * 100)
		// set results
		result[proposalHash] = r
	}
	return
}

// UPGRADE CODE BELOW

// IsFeatureEnabled() checks if a feature is enabled based on the protocol version
// stored in the state compared to the required activation version
func (s *StateMachine) IsFeatureEnabled(requiredVersion uint64) bool {
	// retrieve the current consensus parameters from the state
	consensusParams, err := s.GetParamsCons()
	if err != nil {
		// simply log the failure
		s.log.Error("Failed to retrieve consensus parameters: " + err.Error())
		// return 'feature not enabled'
		return false
	}
	// extract the protocol version from the consensus parameters
	currentProtocol, err := consensusParams.ParseProtocolVersion()
	if err != nil {
		// simply log the failure
		s.log.Error("Failed to parse protocol version: " + err.Error())
		// return 'feature not enabled'
		return false
	}
	// check if the current protocol version is beyond the required activation version
	if currentProtocol.Version > requiredVersion {
		// return 'feature is enabled'
		return true
	}
	// if the current protocol version matches the required version,
	// enable the feature only if the state height is at or beyond the activation height
	return currentProtocol.Version == requiredVersion && s.Height() >= currentProtocol.Height
}

// ROOT CHAIN CODE BELOW

// LoadIsOwnRoot() returns if this chain is its own root (base)
func (s *StateMachine) LoadIsOwnRoot() (bool, lib.ErrorI) {
	// get the latest root chain id from the state
	rootId, err := s.LoadRootChainId(s.Height())
	if err != nil {
		return false, err
	}
	// return whether self is root
	return s.Config.ChainId == rootId, nil
}

// GetRootChainId() gets the latest root chain id from the state
func (s *StateMachine) GetRootChainId() (uint64, lib.ErrorI) {
	// get the consensus params from state
	consParams, err := s.GetParamsCons()
	if err != nil {
		return 0, err
	}
	// return the root chain id from the consensus params
	return consParams.RootChainId, nil
}

// LoadRootChainId() loads the root chain id from the state at a certain height
func (s *StateMachine) LoadRootChainId(height uint64) (uint64, lib.ErrorI) {
	// create a read-only historical version of the state
	historicalFSM, err := s.TimeMachine(height)
	// if an error occurred when loading the historical state machine
	if err != nil {
		// exit with error
		return 0, err
	}
	// memory cleanup
	if height != s.Height() {
		defer historicalFSM.Discard()
	}
	// return the root chain id at that height
	return historicalFSM.GetRootChainId()
}
