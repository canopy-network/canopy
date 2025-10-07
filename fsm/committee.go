package fsm

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file contains logic for 'committees' or validator sets responsible for 'nestedChain' consensus */

// FundCommitteeRewardPools() mints newly created tokens to protocol subsidized committees
func (s *StateMachine) FundCommitteeRewardPools() lib.ErrorI {
	daoCut, totalMintAmount, mintAmountPerCommittee, err := s.GetBlockMintStats(s.Config.ChainId)
	if err != nil {
		return err
	}
	// log total mint
	s.log.Infof("Total mint amount: %d | Mint per committee: %d | DAO cut: %d", totalMintAmount, mintAmountPerCommittee, daoCut)
	// mint to the DAO account
	if err = s.MintToPool(lib.DAOPoolID, daoCut); err != nil {
		return err
	}
	// get the committees that `qualify` for subsidization
	subsidizedChainIds, err := s.GetSubsidizedCommittees()
	if err != nil {
		return err
	}
	// issue that amount to each subsidized committee
	for _, chainId := range subsidizedChainIds {
		if err = s.MintToPool(chainId, mintAmountPerCommittee); err != nil {
			return err
		}
	}
	return nil
}

// GetBlockMintStats() gets the latest minting information for the blockchain
func (s *StateMachine) GetBlockMintStats(chainId uint64) (daoCut uint64, totalMint uint64, mintAmountPerCommittee uint64, err lib.ErrorI) {
	// get governance params that are needed to complete this operation
	govParams, err := s.GetParamsGov()
	if err != nil {
		return
	}
	// get the committees that `qualify` for subsidization
	subsidizedChainIds, err := s.GetSubsidizedCommittees()
	if err != nil {
		return
	}
	// ensure self chain is always a 'paid' chain even if there are no validators
	if !slices.Contains(subsidizedChainIds, chainId) {
		// this ensures nested-chains always receive Native Token payment to their pool
		subsidizedChainIds = append(subsidizedChainIds, chainId)
	}
	// calculate the number of halvenings
	var halvenings uint64
	if s.Config.BlocksPerHalvening > 0 {
		halvenings = s.height / s.Config.BlocksPerHalvening
	}
	// each halving, the reward is divided by 2
	totalMintAmount := s.Config.InitialMintPerBlock >> halvenings
	// define a convenience variable for the number of subsidized committees
	subsidizedCount := uint64(len(subsidizedChainIds))
	// calculate the amount left for the committees after the parameterized DAO cut
	mintAmountAfterDAOCut := lib.Uint64ReducePercentage(totalMintAmount, govParams.DaoRewardPercentage)
	// calculate the DAO cut
	daoCut = totalMintAmount - mintAmountAfterDAOCut

	// calculate the amount given to each qualifying committee
	// mintAmountPerCommittee may truncate, but that's expected,
	// less mint will be created and effectively 'burned'
	if subsidizedCount > 0 {
		mintAmountPerCommittee = mintAmountAfterDAOCut / subsidizedCount
	}

	return daoCut, totalMintAmount, mintAmountPerCommittee, nil
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
	if p.NumValidators == 0 || p.TotalPower == 0 {
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
	validators := make([]*Validator, 0)
	// reverse iterator is slightly more efficient than forward iterator for large datasets
	it, err := s.RevIterator(ValidatorPrefix())
	if err != nil {
		return vs, err
	}
	// ensure memory cleanup
	defer it.Close()
	// for each item of the iterator
	for ; it.Valid(); it.Next() {
		// convert the bytes into a validator object reference
		val, e := s.unmarshalValidator(it.Value())
		if e != nil {
			return vs, e
		}
		// add it to the list
		validators = append(validators, val)
	}
	// filter out validators not part of the committee
	filtered := slices.Collect(func(yield func(*Validator) bool) {
		for _, v := range validators {
			// exclude validators not part of the committee
			if !v.PassesFilter(lib.ValidatorFilters{
				Unstaking: lib.FilterOption_Exclude,
				Paused:    lib.FilterOption_Exclude,
				Delegate:  lib.FilterOption_Exclude,
				Committee: chainId,
			}) {
				continue
			}
			// add validator to filtered list
			if !yield(v) {
				return
			}
		}
	})
	// sort by highest stake then address
	slices.SortFunc(filtered, func(a, b *Validator) int {
		result := cmp.Compare(b.StakedAmount, a.StakedAmount)
		if result == 0 {
			return bytes.Compare(b.Address, a.Address)
		}
		return result
	})
	// create a variable to hold the committee members
	members := make([]*lib.ConsensusValidator, 0)
	// return only the top MaxCommitteeSize validators
	// for each validator up to MaxCommitteeSize
	for _, validator := range filtered[:min(uint64(len(filtered)), p.MaxCommitteeSize)] {
		// add the member to the list
		members = append(members, &lib.ConsensusValidator{
			PublicKey:   validator.PublicKey,
			VotingPower: validator.StakedAmount,
			NetAddress:  validator.NetAddress,
		})
	}
	// convert list to a validator set (includes shared public key)
	return lib.NewValidatorSet(&lib.ConsensusValidators{ValidatorSet: members})
}

// GetCommitteePaginated() returns a 'page' of committee members ordered from the highest stake to lowest
func (s *StateMachine) GetCommitteePaginated(p lib.PageParams, chainId uint64) (page *lib.Page, err lib.ErrorI) {
	// define a page and result variables
	page, res := lib.NewPage(p, ValidatorsPageName), make(ValidatorPage, 0)
	// populate the page using an iterator over the 'committee prefix' ordered by stake (high to low)
	err = page.Load(CommitteePrefix(chainId), true, &res, s.store, func(key, value []byte) (err lib.ErrorI) {
		// get the address from the key
		address, err := AddressFromKey(key)
		if err != nil {
			return err
		}
		// get the validator from the address
		validator, err := s.GetValidator(address)
		if err != nil {
			return err
		}
		// skip if validator is not paused and not unstaking
		if validator.UnstakingHeight != 0 || validator.MaxPausedHeight != 0 {
			return
		}
		// append the validator to the page
		res = append(res, validator)
		return
	})
	return
}

// UpdateCommittees() updates the committee information in state for a specific validator
func (s *StateMachine) UpdateCommittees(address crypto.AddressI, oldValidator *Validator, newStakedAmount uint64, newCommittees []uint64) lib.ErrorI {
	// delete the committee information based on the 'previous state' of the validator
	if err := s.DeleteCommittees(address, oldValidator.StakedAmount, oldValidator.Committees); err != nil {
		return err
	}
	// set the committee information using the updated stake and committees
	return s.SetCommittees(address, newStakedAmount, newCommittees)
}

// SetCommittees() sets the membership and staked supply for all an addresses' committees
func (s *StateMachine) SetCommittees(address crypto.AddressI, totalStake uint64, committees []uint64) (err lib.ErrorI) {
	// for each committee in the list
	for _, committee := range committees {
		// set the address as a member
		if err = s.SetCommitteeMember(address, committee, totalStake); err != nil {
			return
		}
		// add to the committee staked supply
		if err = s.AddToCommitteeSupplyForChain(committee, totalStake); err != nil {
			return
		}
	}
	return
}

// DeleteCommittees() deletes the membership and staked supply for each of an address' committees
func (s *StateMachine) DeleteCommittees(address crypto.AddressI, totalStake uint64, committees []uint64) (err lib.ErrorI) {
	// for each committee in the list
	for _, committee := range committees {
		// remove the address from being a member
		if err = s.DeleteCommitteeMember(address, committee, totalStake); err != nil {
			return
		}
		// subtract from the committee staked supply
		if err = s.SubFromCommitteeStakedSupplyForChain(committee, totalStake); err != nil {
			return
		}
	}
	return
}

// SetCommitteeMember() sets the address as a 'member' of the committee in the state
func (s *StateMachine) SetCommitteeMember(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	return s.Set(KeyForCommittee(chainId, address, stakeForCommittee), nil)
}

// DeleteCommitteeMember() removes the address from being a 'member' of the committee in the state
func (s *StateMachine) DeleteCommitteeMember(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	return s.Delete(KeyForCommittee(chainId, address, stakeForCommittee))
}

// DELEGATIONS BELOW

// GetAllDelegates() returns all delegates for a certain chainId
// It is heavily cached to improve performance as the delegates are used for each block commit
func (s *StateMachine) GetAllDelegates(chainId uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	// retrieve the full list of validators
	validators, err := s.GetValidators()
	if err != nil {
		return
	}
	// filter out validators that are not delegates for the given chainId
	delegates := slices.Collect(func(yield func(*Validator) bool) {
		for _, v := range validators {
			// exclude validators not part of the committee
			if !v.PassesFilter(lib.ValidatorFilters{
				Delegate:  lib.FilterOption_MustBe,
				Paused:    lib.FilterOption_Exclude,
				Unstaking: lib.FilterOption_Exclude,
				Committee: chainId,
			}) {
				continue
			}
			// add validator to filtered list
			if !yield(v) {
				return
			}
		}
	})
	// sort by highest stake then address
	slices.SortFunc(delegates, func(a, b *Validator) int {
		result := cmp.Compare(b.StakedAmount, a.StakedAmount)
		if result == 0 {
			return bytes.Compare(b.Address, a.Address)
		}
		return result
	})
	// create a variable to hold the committee members
	members := make([]*lib.ConsensusValidator, 0)
	var totalPower uint64
	// loop through the delegates
	for _, delegate := range delegates {
		// increment the total power
		totalPower += delegate.StakedAmount
		// add the member to the list
		members = append(members, &lib.ConsensusValidator{
			PublicKey:   delegate.PublicKey,
			VotingPower: delegate.StakedAmount,
			NetAddress:  delegate.NetAddress,
		})
	}
	return lib.ValidatorSet{
		ValidatorSet:  &lib.ConsensusValidators{ValidatorSet: members},
		TotalPower:    totalPower,
		MinimumMaj23:  (2*totalPower)/3 + 1,
		NumValidators: uint64(len(members)),
	}, nil
}

// GetTopDelegates() returns top delegates according to gov params
func (s *StateMachine) GetTopDelegates(chainId uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	// get gov params for filtering
	params, err := s.GetParamsGov()
	if err != nil {
		return vs, err
	}
	// if param is set to 0 all delegates should be returned
	if params.MaximumDelegatesPerCommittee == 0 {
		return s.GetTopDelegates(chainId)
	}
	// iterate from highest stake to lowest
	it, err := s.RevIterator(DelegatePrefix(chainId))
	if err != nil {
		return vs, err
	}
	defer it.Close()
	// collect all delegates to identify ties
	type delegateInfo struct {
		address      crypto.AddressI
		validator    *Validator
		stakedAmount uint64
	}
	var allDelegates []delegateInfo
	for ; it.Valid(); it.Next() {
		// get the address from the iterator key
		address, e := AddressFromKey(it.Key())
		if e != nil {
			return vs, e
		}
		// get the validator from the address
		val, e := s.GetValidator(address)
		if e != nil {
			return vs, e
		}
		// ensure the validator is not included if it's paused or unstaking
		if val.MaxPausedHeight != 0 || val.UnstakingHeight != 0 {
			continue
		}
		allDelegates = append(allDelegates, delegateInfo{
			address:      address,
			validator:    val,
			stakedAmount: val.StakedAmount,
		})
	}
	// if we have fewer delegates than the max, return all
	if uint64(len(allDelegates)) <= params.MaximumDelegatesPerCommittee {
		members := make([]*lib.ConsensusValidator, 0, len(allDelegates))
		var totalPower uint64
		for _, d := range allDelegates {
			totalPower += d.stakedAmount
			members = append(members, &lib.ConsensusValidator{
				PublicKey:   d.validator.PublicKey,
				VotingPower: d.stakedAmount,
				NetAddress:  d.validator.NetAddress,
			})
		}
		return lib.ValidatorSet{
			ValidatorSet:  &lib.ConsensusValidators{ValidatorSet: members},
			TotalPower:    totalPower,
			MinimumMaj23:  (2*totalPower)/3 + 1,
			NumValidators: uint64(len(members)),
		}, nil
	}
	// collect top delegates up to the limit
	selected := make([]delegateInfo, 0, params.MaximumDelegatesPerCommittee)
	var cutoffStake uint64
	// take the top MaximumDelegatesPerCommittee-1 delegates for sure
	for i := uint64(0); i < params.MaximumDelegatesPerCommittee-1 && i < uint64(len(allDelegates)); i++ {
		selected = append(selected, allDelegates[i])
	}
	// identify the cutoff stake (the stake of the last delegate we want to include)
	cutoffIndex := params.MaximumDelegatesPerCommittee - 1
	if cutoffIndex < uint64(len(allDelegates)) {
		cutoffStake = allDelegates[cutoffIndex].stakedAmount
	} else {
		// shouldn't happen due to earlier check, but handle gracefully
		cutoffStake = 0
	}
	// collect all delegates at the cutoff stake (there might be ties)
	var tiedDelegates []delegateInfo
	for i := cutoffIndex; i < uint64(len(allDelegates)) && allDelegates[i].stakedAmount == cutoffStake; i++ {
		tiedDelegates = append(tiedDelegates, allDelegates[i])
	}
	// if there's no tie, just add the one at cutoff
	if len(tiedDelegates) == 1 {
		selected = append(selected, tiedDelegates[0])
	} else {
		// there's a tie, randomly select one from the tied group
		// get the last proposers for deterministic randomness
		lastProposers, e := s.GetLastProposers()
		if e != nil {
			return vs, e
		}
		// create a temporary validator set from tied delegates for weighted random selection
		tiedMembers := make([]*lib.ConsensusValidator, 0, len(tiedDelegates))
		var tiedTotalPower uint64
		for _, d := range tiedDelegates {
			tiedTotalPower += d.stakedAmount
			tiedMembers = append(tiedMembers, &lib.ConsensusValidator{
				PublicKey:   d.validator.PublicKey,
				VotingPower: d.stakedAmount,
				NetAddress:  d.validator.NetAddress,
			})
		}
		// use weighted pseudorandom to select one from the tied group
		winner := lib.WeightedPseudorandom(&lib.PseudorandomParams{
			SortitionData: &lib.SortitionData{
				LastProposerAddresses: lastProposers.Addresses,
				RootHeight:            0, // deterministic
				Height:                s.Height(),
				TotalValidators:       uint64(len(tiedDelegates)),
				TotalPower:            tiedTotalPower,
			}, ValidatorSet: &lib.ConsensusValidators{ValidatorSet: tiedMembers},
		})
		// find the matching delegate and add to selected
		winnerAddr := winner.Address()
		for _, d := range tiedDelegates {
			if bytes.Equal(d.address.Bytes(), winnerAddr.Bytes()) {
				selected = append(selected, d)
				break
			}
		}
	}
	// build the final validator set from selected delegates
	members := make([]*lib.ConsensusValidator, 0, len(selected))
	var totalPower uint64
	for _, d := range selected {
		totalPower += d.stakedAmount
		members = append(members, &lib.ConsensusValidator{
			PublicKey:   d.validator.PublicKey,
			VotingPower: d.stakedAmount,
			NetAddress:  d.validator.NetAddress,
		})
	}
	return lib.ValidatorSet{
		ValidatorSet:  &lib.ConsensusValidators{ValidatorSet: members},
		TotalPower:    totalPower,
		MinimumMaj23:  (2*totalPower)/3 + 1,
		NumValidators: uint64(len(members)),
	}, nil
}

// GetDelegatesPaginated() returns a page of delegates
func (s *StateMachine) GetDelegatesPaginated(p lib.PageParams, chainId uint64) (page *lib.Page, err lib.ErrorI) {
	// create a page of validator objects
	page, res := lib.NewPage(p, ValidatorsPageName), make(ValidatorPage, 0)
	// populate the page using the 'delegates' prefix sorted by stake (high to low)
	err = page.Load(DelegatePrefix(chainId), true, &res, s.store, func(key, _ []byte) (err lib.ErrorI) {
		// get the address from the key
		address, err := AddressFromKey(key)
		if err != nil {
			return err
		}
		// get the validator from the address
		validator, err := s.GetValidator(address)
		if err != nil {
			return err
		}
		// skip if validator is paused or unstaking
		if validator.UnstakingHeight != 0 || validator.MaxPausedHeight != 0 {
			return
		}
		// append the validator to the page
		res = append(res, validator)
		return
	})
	return
}

// UpdateDelegations() updates the delegate information for an address, first removing the outdated delegation information and then setting the new info
func (s *StateMachine) UpdateDelegations(address crypto.AddressI, oldValidator *Validator, newStakedAmount uint64, newCommittees []uint64) lib.ErrorI {
	// remove the outdated delegation information
	if err := s.DeleteDelegations(address, oldValidator.StakedAmount, oldValidator.Committees); err != nil {
		return err
	}
	// set the delegations back into state
	return s.SetDelegations(address, newStakedAmount, newCommittees)
}

// SetDelegations() sets the delegate 'membership' for an address, adding to the list and updating the supply pools
func (s *StateMachine) SetDelegations(address crypto.AddressI, totalStake uint64, committees []uint64) lib.ErrorI {
	for _, committee := range committees {
		// actually set the address in the delegate list
		if err := s.SetDelegate(address, committee, totalStake); err != nil {
			return err
		}
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

// DeleteDelegations() removes the delegate 'membership' for an address, removing from the list and updating the supply pools
func (s *StateMachine) DeleteDelegations(address crypto.AddressI, totalStake uint64, committees []uint64) lib.ErrorI {
	for _, committee := range committees {
		// remove the address from the delegate list
		if err := s.DeleteDelegate(address, committee, totalStake); err != nil {
			return err
		}
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

// SetDelegate() sets a delegate in state using the delegate prefix
func (s *StateMachine) SetDelegate(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	return s.Set(KeyForDelegate(chainId, address, stakeForCommittee), nil)
}

// DeleteDelegate() removes a delegate from the state using the delegate prefix
func (s *StateMachine) DeleteDelegate(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	return s.Delete(KeyForDelegate(chainId, address, stakeForCommittee))
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
