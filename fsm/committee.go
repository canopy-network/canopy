package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
	"slices"
)

/* This file contains logic for 'committees' or validator sets responsible for 'nestedChain' consensus */

// FundCommitteeRewardPools() mints newly created tokens to protocol subsidized committees
func (s *StateMachine) FundCommitteeRewardPools() lib.ErrorI {
	// get governance params that are needed to complete this operation
	govParams, err := s.GetParamsGov()
	if err != nil {
		return err
	}
	// get the committees that `qualify` for subsidization
	subsidizedChainIds, err := s.GetSubsidizedCommittees()
	if err != nil {
		return err
	}
	// ensure self chain is always a 'paid' chain even if there are no validators
	if !slices.Contains(subsidizedChainIds, s.Config.ChainId) {
		// this ensures nested-chains always receive Native Token payment to their pool
		subsidizedChainIds = append(subsidizedChainIds, s.Config.ChainId)
	}
	// calculate the number of halvenings
	halvenings := s.height / uint64(BlocksPerHalvening)
	// each halving, the reward is divided by 2
	totalMintAmount := uint64(InitialTokensPerBlock >> halvenings)
	// define a convenience variable for the number of subsidized committees
	subsidizedCount := uint64(len(subsidizedChainIds))
	// if there are no subsidized committees or no mint amount
	if subsidizedCount == 0 || totalMintAmount == 0 {
		return nil
	}
	// calculate the amount left for the committees after the parameterized DAO cut
	mintAmountAfterDAOCut := lib.Uint64ReducePercentage(totalMintAmount, govParams.DaoRewardPercentage)
	// calculate the DAO cut
	daoCut := totalMintAmount - mintAmountAfterDAOCut
	// mint to the DAO account
	if err = s.MintToPool(lib.DAOPoolID, daoCut); err != nil {
		return err
	}
	// calculate the amount given to each qualifying committee
	// mintAmountPerCommittee may truncate, but that's expected,
	// less mint will be created and effectively 'burned'
	mintAmountPerCommittee := mintAmountAfterDAOCut / subsidizedCount
	// issue that amount to each subsidized committee
	for _, chainId := range subsidizedChainIds {
		if err = s.MintToPool(chainId, mintAmountPerCommittee); err != nil {
			return err
		}
	}
	return nil
}

// GetSubsidizedCommittees() returns a list of chainIds that receive a portion of the 'block reward'
// Think of these committees as 'automatically subsidized' by the protocol
func (s *StateMachine) GetSubsidizedCommittees() (paidIDs []uint64, err lib.ErrorI) {
	// get validator params that are needed to complete this operation
	valParams, err := s.GetParamsVal()
	if err != nil {
		return nil, err
	}
	// retrieve the supply
	supply, err := s.GetSupply()
	if err != nil {
		return nil, err
	}
	// since re-staking is enabled in the Canopy protocol, the protocol is able to simply pick which chains are paid via a 'committed stake'
	// percentage. Since the effective cost of 're-staking' is essentially zero (minus the risk associated with slashing on the chain)
	// this may be thought of like a popularity contest or a vote.
	for _, committee := range supply.CommitteeStaked {
		// calculate the percent of stake the committee controls
		committedStakePercent := lib.Uint64PercentageDiv(committee.Amount, supply.Staked)
		// if the committee percentage is over the threshold
		if committedStakePercent >= valParams.StakePercentForSubsidizedCommittee {
			// get retired status of the committee
			retired, e := s.CommitteeIsRetired(committee.Id)
			if e != nil {
				return nil, e
			}
			// ensure the committee isn't retired
			if retired {
				s.log.Warnf("Not subsidizing retired committee: %d", committee.Id)
				continue
			}
			// add it to the paid list
			paidIDs = append(paidIDs, committee.Id)
		}
	}
	return
}

// DistributeCommitteeRewards() distributes the committee mint based on the PaymentPercents from the result of the QuorumCertificate
func (s *StateMachine) DistributeCommitteeRewards() lib.ErrorI {
	// retrieve the necessary parameters
	paramsVal, err := s.GetParamsVal()
	if err != nil {
		return err
	}
	// retrieve the master list of committee data
	committeesData, err := s.GetCommitteesData()
	if err != nil {
		return err
	}
	// for each committee data in the list
	for i, data := range committeesData.List {
		// check to see if any payment percents were issued
		if len(data.PaymentPercents) == 0 {
			// if none issued, move on to the next
			continue
		}
		// retrieve the reward pool
		rewardPool, e := s.GetPool(data.ChainId)
		if e != nil {
			return e
		}
		// create a tracker variable for total amount distributed
		var totalDistributed uint64
		// for each payment percent issued
		for _, stub := range data.PaymentPercents {
			distributed, er := s.DistributeCommitteeReward(stub, rewardPool.Amount, data.NumberOfSamples, paramsVal)
			if er != nil {
				return er
			}
			totalDistributed += distributed
		}
		// ensure the non-distributed (burned) is removed from the 'total supply'
		if err = s.SubFromTotalSupply(rewardPool.Amount - totalDistributed); err != nil {
			return err
		}
		// zero out the reward pool
		rewardPool.Amount = 0
		// update the pool in state
		if err = s.SetPool(rewardPool); err != nil {
			return err
		}
		// clear the committee data, but leave the ID, (external) chain height, and committee height
		committeesData.List[i] = &lib.CommitteeData{
			ChainId:                data.ChainId,
			LastRootHeightUpdated:  data.LastRootHeightUpdated,
			LastChainHeightUpdated: data.LastChainHeightUpdated,
		}
	}
	// set the committee data in state
	return s.SetCommitteesData(committeesData)
}

// DistributeCommitteeReward() issues a single committee reward unit based on an individual 'Payment Stub'
func (s *StateMachine) DistributeCommitteeReward(stub *lib.PaymentPercents, rewardPoolAmount, numberOfSamples uint64, valParams *ValidatorParams) (distributed uint64, err lib.ErrorI) {
	address := crypto.NewAddress(stub.Address)
	// full_reward = truncate ( percentage / number_of_samples * available_reward )
	fullReward := (stub.Percent * rewardPoolAmount) / (numberOfSamples * 100)
	// if not compounding, use the early withdrawal reward
	earlyWithdrawalReward := lib.Uint64ReducePercentage(fullReward, valParams.EarlyWithdrawalPenalty)
	// check if is validator
	validator, _ := s.GetValidator(address)
	// if non validator, send EarlyWithdrawalReward to the address
	if validator == nil {
		// add directly to the account with an early withdrawal penalty
		return earlyWithdrawalReward, s.AccountAdd(address, earlyWithdrawalReward)
	}
	// if validator and compounding, send full reward to the stake of the validator
	if validator.Compound && validator.UnstakingHeight == 0 {
		return fullReward, s.UpdateValidatorStake(validator, validator.Committees, fullReward)
	}
	// if validator is not compounding, send earlyWithdrawalReward reward to the output address of the validator
	return earlyWithdrawalReward, s.AccountAdd(crypto.NewAddress(validator.Output), earlyWithdrawalReward)
}

// LotteryWinner() selects a validator/delegate randomly weighted based on their stake within a committee
func (s *StateMachine) LotteryWinner(id uint64, validators ...bool) (lottery *lib.LotteryWinner, err lib.ErrorI) {
	// create a variable to hold the 'members' of the committee
	var p lib.ValidatorSet
	// if validators
	if len(validators) == 1 && validators[0] == true {
		p, _ = s.GetCommitteeMembers(s.Config.ChainId)
	} else {
		// else get the delegates
		p, _ = s.GetAllDelegates(id)
	}
	// get the validator params from state
	valParams, err := s.GetParamsVal()
	if err != nil {
		return nil, err
	}
	// define a convenience variable for the 'cut' of the lottery winner
	winnerCut := valParams.DelegateRewardPercentage
	// if there are no validators in the set - return
	if p.NumValidators == 0 {
		return &lib.LotteryWinner{Winner: nil, Cut: winnerCut}, nil
	}
	// get the last proposers
	lastProposers, err := s.GetLastProposers()
	if err != nil {
		return
	}
	// use un-grindable weighted pseudorandom to select a winner
	winner := lib.WeightedPseudorandom(&lib.PseudorandomParams{
		SortitionData: &lib.SortitionData{
			LastProposerAddresses: lastProposers.Addresses,
			RootHeight:            0, // deterministic
			Height:                s.Height(),
			TotalValidators:       p.NumValidators,
			TotalPower:            p.TotalPower,
		}, ValidatorSet: p.ValidatorSet,
	})
	// return the lottery winner and cut
	return &lib.LotteryWinner{
		Winner: winner.Address().Bytes(),
		Cut:    winnerCut,
	}, nil
}

// GetCommitteeMembers() retrieves the ValidatorSet that is responsible for the 'chainId'
func (s *StateMachine) GetCommitteeMembers(chainId uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	// get the validator params
	p, err := s.GetParamsVal()
	if err != nil {
		return
	}
	return s.GetValidatorsFromCommitteeList(chainId, false, p.MaxCommitteeSize)
}

// GetCommitteePaginated() returns a 'page' of committee members ordered from the highest stake to lowest
func (s *StateMachine) GetCommitteePaginated(p lib.PageParams, chainId uint64) (page *lib.Page, err lib.ErrorI) {
	return s.GetCommitteeListPaginated(p, chainId, false)
}

// UpdateCommittees() updates the committee information in state for a specific validator
func (s *StateMachine) UpdateCommittees(oldValidator *Validator, newStakedAmount uint64, newCommittees []uint64) lib.ErrorI {
	// delete the committee information based on the 'previous state' of the validator
	if err := s.RemoveStakeFromCommittees(oldValidator.StakedAmount, oldValidator.Committees); err != nil {
		return err
	}
	// set the committee information using the updated stake and committees
	return s.AddStakeToCommittees(newStakedAmount, newCommittees)
}

// AddStakeToCommittees() sets the membership and staked supply for all an addresses' committees
func (s *StateMachine) AddStakeToCommittees(totalStake uint64, committees []uint64) (err lib.ErrorI) {
	// for each committee in the list
	for _, committee := range committees {
		// add to the committee staked supply
		if err = s.AddToCommitteeSupplyForChain(committee, totalStake); err != nil {
			return
		}
	}
	return
}

// RemoveStakeFromCommittees() deletes the membership and staked supply for each of an address' committees
func (s *StateMachine) RemoveStakeFromCommittees(totalStake uint64, committees []uint64) (err lib.ErrorI) {
	// for each committee in the list
	for _, committee := range committees {
		// subtract from the committee staked supply
		if err = s.SubFromCommitteeStakedSupplyForChain(committee, totalStake); err != nil {
			return
		}
	}
	return
}

// DELEGATIONS BELOW

// GetAllDelegates() returns all delegates for a certain chainId
func (s *StateMachine) GetAllDelegates(chainId uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	return s.GetValidatorsFromCommitteeList(chainId, true, math.MaxUint64)
}

// GetDelegatesPaginated() returns a page of delegates
func (s *StateMachine) GetDelegatesPaginated(p lib.PageParams, chainId uint64) (page *lib.Page, err lib.ErrorI) {
	return s.GetCommitteeListPaginated(p, chainId, true)
}

// UpdateDelegations() updates the delegate information for an address, first removing the outdated delegation information and then setting the new info
func (s *StateMachine) UpdateDelegations(oldValidator *Validator, newStakedAmount uint64, newCommittees []uint64) lib.ErrorI {
	// remove the outdated delegation information
	if err := s.RemoveDelegationsFromCommittees(oldValidator.StakedAmount, oldValidator.Committees); err != nil {
		return err
	}
	// set the delegations back into state
	return s.AddDelegationsToCommittees(newStakedAmount, newCommittees)
}

// AddDelegationsToCommittees() sets the delegation -- updating the supply pools
func (s *StateMachine) AddDelegationsToCommittees(totalStake uint64, committees []uint64) lib.ErrorI {
	for _, committee := range committees {
		// add to the delegate supply (used for tracking amounts)
		if err := s.AddToDelegateSupplyForChain(committee, totalStake); err != nil {
			return err
		}
		// add to the committee supply as well (used for tracking amounts)
		if err := s.AddToCommitteeSupplyForChain(committee, totalStake); err != nil {
			return err
		}
	}
	return nil
}

// RemoveDelegationsFromCommittees() removes the delegation -- updating the supply pools
func (s *StateMachine) RemoveDelegationsFromCommittees(totalStake uint64, committees []uint64) lib.ErrorI {
	for _, committee := range committees {
		// remove from the delegate supply (used for tracking amounts)
		if err := s.SubFromDelegateStakedSupplyForChain(committee, totalStake); err != nil {
			return err
		}
		// remove from the committee supply as well (used for tracking amounts)
		if err := s.SubFromCommitteeStakedSupplyForChain(committee, totalStake); err != nil {
			return err
		}
	}
	return nil
}

// COMMITTEE LIST FUNCTIONALITY BELOW

// GetValidatorsFromCommitteeList() returns a validator set from a committee list
func (s *StateMachine) GetValidatorsFromCommitteeList(chainId uint64, delegate bool, max uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	// get the delegates from the list
	list, err := s.GetCommitteeList(chainId, delegate, max)
	// create a variable to hold the committee members
	members := make([]*lib.ConsensusValidator, 0)
	// loop through the iterator
	for _, val := range list {
		// add the member to the list
		members = append(members, &lib.ConsensusValidator{
			PublicKey:   val.PublicKey,
			VotingPower: val.StakedAmount,
			NetAddress:  val.NetAddress,
		})
	}
	return lib.NewValidatorSet(&lib.ConsensusValidators{ValidatorSet: members})
}

// GetCommitteeListPaginated() returns a page of committee members
func (s *StateMachine) GetCommitteeListPaginated(p lib.PageParams, chainId uint64, delegate bool) (page *lib.Page, err lib.ErrorI) {
	// get the delegates from the list
	list, err := s.GetCommitteeList(chainId, delegate, math.MaxUint64)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// create a page of validator objects
	page, res := lib.NewPage(p, ValidatorsPageName), make(ValidatorPage, 0)
	page.Results = &res
	// populate the page using the 'delegates' prefix sorted by stake (high to low)
	err = page.LoadArray(list, &res, func(item any) (err lib.ErrorI) {
		// append the validator to the page
		res = append(res, item.(*Validator))
		// exit
		return
	})
	// exit
	return
}

// GetCommitteeList() returns every address that is registered for a committee as a validator or delegator
func (s *StateMachine) GetCommitteeList(chainId uint64, delegate bool, max uint64) (members []*Validator, err lib.ErrorI) {
	// ensure a non-nil result
	members = make([]*Validator, 0)
	// get the list of validators / delegates sorted by stake then address
	list, err := s.GetValidatorList(delegate)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// for each validator in the list
	for idx, val := range list.List {
		// if index == max validator count
		if uint64(idx) == max {
			// exit
			break
		}
		// if disallow < unstaking or paused >
		if val.UnstakingHeight != 0 || val.MaxPausedHeight != 0 {
			// next iteration
			continue
		}
		// if the validator doesn't have the committee
		if !slices.Contains(val.Committees, chainId) {
			// next iteration
			continue
		}
		// add the validator to the members list
		members = append(members, val)
	}
	// exit
	return
}

// COMMITTEE DATA CODE BELOW
// 'Committee Data' is information saved in state that is aggregated from CertificateResult messages
// This information secures and dictates the distribution of a Committee's reward pool

// UpsertCommitteeData() updates or inserts a committee data to the committees data list
func (s *StateMachine) UpsertCommitteeData(new *lib.CommitteeData) lib.ErrorI {
	// retrieve the committees' data list, the target and index in the list based on the chainId
	committeesData, targetData, idx, err := s.getCommitteeDataAndList(new.ChainId)
	if err != nil {
		return err
	}
	// check the new committee data is not 'outdated'
	// new.ChainHeight must be > state.ChainHeight
	if new.LastChainHeightUpdated <= targetData.LastChainHeightUpdated {
		return lib.ErrInvalidQCCommitteeHeight()
	}
	// new.RootHeight must be >= state.RootHeight
	if new.LastRootHeightUpdated < targetData.LastRootHeightUpdated {
		return lib.ErrInvalidQCRootChainHeight()
	}
	// combine the new data with the target, only capturing payment percents for self chainId
	if err = targetData.Combine(new, s.Config.ChainId); err != nil {
		return err
	}
	// add the target back into the list
	committeesData.List[idx] = targetData
	// set the list back into state
	return s.SetCommitteesData(committeesData)
}

// OverwriteCommitteeData() overwrites the committee data in state
// note: use UpsertCommitteeData for the safe committee upsert
func (s *StateMachine) OverwriteCommitteeData(d *lib.CommitteeData) lib.ErrorI {
	// retrieve the committees' data list, the target and index in the list based on the chainId
	committeesData, _, idx, err := s.getCommitteeDataAndList(d.ChainId)
	if err != nil {
		return err
	}
	// add the target back into the list
	committeesData.List[idx] = d
	// set the list back into state
	return s.SetCommitteesData(committeesData)
}

// LoadCommitteeData() loads a historical (or clean latest) committee data from the master list
func (s *StateMachine) LoadCommitteeData(height, targetChainId uint64) (*lib.CommitteeData, lib.ErrorI) {
	// get a historical FSM (or clean latest)
	historicalFSM, err := s.TimeMachine(height)
	if err != nil {
		return nil, err
	}
	// ensure the historical fsm is discarded for memory management
	defer historicalFSM.Discard()
	// exit
	return historicalFSM.GetCommitteeData(targetChainId)
}

// GetCommitteeData() is a convenience function to retrieve the committee data from the master list
func (s *StateMachine) GetCommitteeData(targetChainId uint64) (*lib.CommitteeData, lib.ErrorI) {
	// retrieve the committee's data and return it
	_, targetData, _, err := s.getCommitteeDataAndList(targetChainId)
	if err != nil {
		return nil, err
	}
	// return the target committee data and error only
	return targetData, nil
}

// getCommitteeDataAndList() returns the master list of committee data and the specified target data and its index from the target chain id
func (s *StateMachine) getCommitteeDataAndList(targetChainId uint64) (list *lib.CommitteesData, d *lib.CommitteeData, idx int, err lib.ErrorI) {
	// first, get the master list of 'committee data'
	list, err = s.GetCommitteesData()
	if err != nil {
		return
	}
	// linear search for the committee data
	for i, data := range list.List {
		// if target found, return
		if data.ChainId == targetChainId {
			return list, data, i, nil
		}
	}
	// if target is not found in the list...
	// the target index is the new list end
	idx = len(list.List)
	// set the committee data in the returned variable
	d = &lib.CommitteeData{
		ChainId:                targetChainId,
		LastRootHeightUpdated:  0,
		LastChainHeightUpdated: 0,
		PaymentPercents:        make([]*lib.PaymentPercents, 0),
		NumberOfSamples:        0,
	}
	// insert a new committee fund at the end of the list
	list.List = append(list.List, d)
	// exit
	return
}

// SetCommitteesData() sets a list of committee data in the state
func (s *StateMachine) SetCommitteesData(list *lib.CommitteesData) lib.ErrorI {
	// convert the committee data list to bytes
	bz, err := lib.Marshal(list)
	if err != nil {
		return err
	}
	// set the list bytes under the 'committees data prefix'
	return s.Set(CommitteesDataPrefix(), bz)
}

// GetCommitteesData() gets a list of List from the state
func (s *StateMachine) GetCommitteesData() (list *lib.CommitteesData, err lib.ErrorI) {
	// get the CommitteesData bytes under 'committees data prefix'
	bz, err := s.Get(CommitteesDataPrefix())
	if err != nil {
		return nil, err
	}
	// create a list variable to ensure non-nil results
	list = &lib.CommitteesData{
		List: make([]*lib.CommitteeData, 0),
	}
	// populate the list reference with the CommitteesData bytes
	err = lib.Unmarshal(bz, list)
	// exit
	return
}

// RetireCommittee marks a committee as non-subsidized for eternity
// This is a useful mechanism to gracefully 'end' a committee
func (s *StateMachine) RetireCommittee(chainId uint64) lib.ErrorI {
	// set the default value for a chain id using a key for the retired committee prefix key
	return s.Set(KeyForRetiredCommittee(chainId), RetiredCommitteesPrefix())
}

// CommitteeIsRetired checks if a committee is marked as 'retired' which prevents it from being subsidized for eternity
func (s *StateMachine) CommitteeIsRetired(chainId uint64) (bool, lib.ErrorI) {
	// retrieve the bytes under the retired key for the chain id
	bz, err := s.Get(KeyForRetiredCommittee(chainId))
	if err != nil {
		return false, err
	}
	// check if the bytes equal the default value (RetiredCommitteesPrefix)
	return bytes.Equal(RetiredCommitteesPrefix(), bz), nil
}

// GetRetiredCommittees() returns a list of the retired chainIds
func (s *StateMachine) GetRetiredCommittees() (list []uint64, err lib.ErrorI) {
	// for each item under the retired committee prefix
	err = s.IterateAndExecute(RetiredCommitteesPrefix(), func(key, _ []byte) (e lib.ErrorI) {
		// extract the chain id from the key
		chainId, e := IdFromKey(key)
		if e != nil {
			return
		}
		// add the chainId to the list of retired committees
		list = append(list, chainId)
		// exit inner
		return
	})
	// exit outer
	return
}

// SetRetiredCommittees() sets a list of chainIds as retired
func (s *StateMachine) SetRetiredCommittees(chainIds []uint64) (err lib.ErrorI) {
	// for each chain id on the list
	for _, id := range chainIds {
		// set the committee as retired
		if err = s.RetireCommittee(id); err != nil {
			// exit if error
			return
		}
	}
	// exit
	return
}
