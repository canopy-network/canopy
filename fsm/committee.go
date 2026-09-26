package fsm

import (
	"bytes"
	"math"
	"math/big"
	"slices"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file contains logic for 'committees' or validator sets responsible for 'nestedChain' consensus */

// FundCommitteeRewardPools() mints newly created tokens to protocol subsidized committees
func (s *StateMachine) FundCommitteeRewardPools() lib.ErrorI {
	subsidizedChainIds, daoCut, _, mintAmountPerCommittee, err := s.GetBlockMintStats(s.Config.ChainId)
	if err != nil {
		if err.Code() == lib.CodeNoSubsidizedCommittees {
			return nil
		}
		return err
	}
	// mint to the DAO account
	if err = s.MintToPool(lib.DAOPoolID, daoCut); err != nil {
		return err
	}
	s.log.Debugf("Subsidized committee rewards pools: %v, daoCut: %d, mintAmountPerCommittee: %d", subsidizedChainIds, daoCut, mintAmountPerCommittee)
	// issue that amount to each subsidized committee
	for _, chainId := range subsidizedChainIds {
		if err = s.MintToPool(chainId, mintAmountPerCommittee); err != nil {
			return err
		}
	}
	return nil
}

// GetBlockMintStats() gets the latest minting information for the blockchain
func (s *StateMachine) GetBlockMintStats(chainId uint64) (subsidizedChainIds []uint64, daoCut uint64, totalMint uint64, mintAmountPerCommittee uint64, err lib.ErrorI) {
	if s.Config.BlocksPerHalvening == 0 {
		return nil, 0, 0, 0, lib.ErrInvalidArgument()
	}
	// get governance params that are needed to complete this operation
	govParams, err := s.GetParamsGov()
	if err != nil {
		return
	}
	// get the committees that `qualify` for subsidization
	subsidizedChainIds, err = s.GetSubsidizedCommittees()
	if err != nil {
		return
	}
	// calculate the number of halvenings
	halvenings := s.height / s.Config.BlocksPerHalvening
	// each halving, the reward is divided by 2
	totalMintAmount := s.Config.InitialTokensPerBlock >> halvenings
	// define a convenience variable for the number of subsidized committees
	subsidizedCount := uint64(len(subsidizedChainIds))
	// if there are no subsidized committees or no mint amount
	if subsidizedCount == 0 || totalMintAmount == 0 {
		err = lib.ErrNoSubsidizedCommittees(chainId)
		return
	}
	// calculate the amount left for the committees after the parameterized DAO cut
	var mintAmountAfterDAOCut uint64
	switch {
	case govParams.DaoRewardPercentage >= 100 || totalMintAmount == 0:
		mintAmountAfterDAOCut = 0
	case govParams.DaoRewardPercentage == 0:
		mintAmountAfterDAOCut = totalMintAmount
	default:
		mintAmountAfterDAOCut = lib.SafeMulDiv(totalMintAmount, 100-govParams.DaoRewardPercentage, 100)
	}
	// calculate the DAO cut
	daoCut = totalMintAmount - mintAmountAfterDAOCut
	// calculate the amount given to each qualifying committee
	// mintAmountPerCommittee may truncate, but that's expected,
	// less mint will be created and effectively 'burned'
	mintAmountPerCommittee = mintAmountAfterDAOCut / subsidizedCount
	// return the variables
	return subsidizedChainIds, daoCut, totalMintAmount, mintAmountPerCommittee, nil
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
	// ensure self chain is always a 'paid' chain
	if !slices.Contains(paidIDs, s.Config.ChainId) {
		// this ensures nested-chains always receive Native Token payment to their pool
		paidIDs = append(paidIDs, s.Config.ChainId)
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
			//s.log.Debugf("Distribute committee reward for committee %d: skipped, no reward recipients", data.ChainId)
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
			distributed, er := s.DistributeCommitteeReward(stub, rewardPool.Amount, data.NumberOfSamples, data.ChainId, paramsVal)
			if er != nil {
				return er
			}
			// no event if distributed is 0
			if distributed > 0 {
				// add an event for a reward amount
				if err = s.EventReward(stub.Address, distributed, data.ChainId); err != nil {
					return err
				}
				if totalDistributed > math.MaxUint64-distributed {
					return ErrInvalidAmount()
				}
				totalDistributed += distributed
			}
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
func (s *StateMachine) DistributeCommitteeReward(stub *lib.PaymentPercents, rewardPoolAmount, numberOfSamples, chainId uint64, valParams *ValidatorParams) (distributed uint64, err lib.ErrorI) {
	address := crypto.NewAddress(stub.Address)
	// full_reward = truncate ( percentage / number_of_samples * available_reward )
	var fullReward uint64
	if numberOfSamples != 0 {
		numerator := new(big.Int).Mul(new(big.Int).SetUint64(stub.Percent), new(big.Int).SetUint64(rewardPoolAmount))
		denominator := new(big.Int).Mul(new(big.Int).SetUint64(numberOfSamples), big.NewInt(100))
		fullReward = new(big.Int).Div(numerator, denominator).Uint64()
	}
	// if not compounding, use the early withdrawal reward
	var earlyWithdrawalReward uint64
	switch {
	case valParams.EarlyWithdrawalPenalty >= 100 || fullReward == 0:
		earlyWithdrawalReward = 0
	case valParams.EarlyWithdrawalPenalty == 0:
		earlyWithdrawalReward = fullReward
	default:
		earlyWithdrawalReward = lib.SafeMulDiv(fullReward, 100-valParams.EarlyWithdrawalPenalty, 100)
	}
	//s.log.Debugf("Distributed committee %d reward: to %s percent: %d%%, rewardPoolAmount: %d, numberOfSamples: %d, FullReward: %d, EarlyWithdrawalReward: %d",
	//	chainId, address.String(), stub.Percent, rewardPoolAmount, numberOfSamples, fullReward, earlyWithdrawalReward)
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
		p, _ = s.GetDelegates(id)
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
	return s.getValidatorSet(chainId, false)
}

// UpdateCommittees() updates the committee information in state for a specific validator
func (s *StateMachine) UpdateCommittees(address crypto.AddressI, oldValidator *Validator, newStakedAmount uint64, newCommittees []uint64) lib.ErrorI {
	// delete the committee information based on the 'previous state' of the validator
	if err := s.DeleteCommittees(address, oldValidator.StakedAmount, oldValidator.Committees); err != nil {
		return err
	}
	// protocol v3+: drop 'active' status only for committees the validator is leaving (kept committees,
	// e.g. from auto-compounding edit-stakes, retain their liveness status)
	if s.IsFeatureEnabled(3) {
		if err := s.deleteActiveCommitteesOnLeave(address, oldValidator.Committees, newCommittees); err != nil {
			return err
		}
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
	// protocol v2+ derives committees from validator records; skip legacy committee index writes.
	if s.IsFeatureEnabled(2) {
		return nil
	}
	return s.Set(KeyForCommittee(chainId, address, stakeForCommittee), nil)
}

// DeleteCommitteeMember() removes the address from being a 'member' of the committee in the state
func (s *StateMachine) DeleteCommitteeMember(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	// protocol v2+ stops mutating legacy committee index keys.
	if s.IsFeatureEnabled(2) {
		return nil
	}
	return s.Delete(KeyForCommittee(chainId, address, stakeForCommittee))
}

// COMMITTEE MEMBER LIVENESS (ACTIVATION) CODE BELOW
// protocol v3+: a committee member joins with zero voting power ('provisional') until it proves liveness
// by signing a QuorumCertificate; this stops a large restaker from halting a running committee

// activeCommitteeMemberValue is a non-nil marker so 'Get' calls can differentiate from non-existing keys
var activeCommitteeMemberValue = []byte{0x1}

// SetCommitteeMemberActive() marks a validator as a liveness-proven ('active') member of the committee
func (s *StateMachine) SetCommitteeMemberActive(chainId uint64, address crypto.AddressI) lib.ErrorI {
	return s.Set(KeyForActiveCommittee(chainId, address), activeCommitteeMemberValue)
}

// DeleteCommitteeMemberActive() removes the liveness-proven ('active') status of a committee member
func (s *StateMachine) DeleteCommitteeMemberActive(chainId uint64, address crypto.AddressI) lib.ErrorI {
	return s.Delete(KeyForActiveCommittee(chainId, address))
}

// IsCommitteeMemberActive() returns if a validator has proven liveness ('active') for the committee
func (s *StateMachine) IsCommitteeMemberActive(chainId uint64, address crypto.AddressI) (bool, lib.ErrorI) {
	bz, err := s.Get(KeyForActiveCommittee(chainId, address))
	if err != nil {
		return false, err
	}
	return bz != nil, nil
}

// GetActiveCommitteeMemberSet() returns the set of liveness-proven ('active') member addresses for a chainId
func (s *StateMachine) GetActiveCommitteeMemberSet(chainId uint64) (set map[string]struct{}, err lib.ErrorI) {
	set = make(map[string]struct{})
	err = s.IterateAndExecute(ActiveCommitteePrefix(chainId), func(key, _ []byte) lib.ErrorI {
		addr, e := AddressFromKey(key)
		if e != nil {
			s.log.Warnf("skipping malformed active-committee key: %x", key)
			return nil
		}
		set[string(addr.Bytes())] = struct{}{}
		return nil
	})
	return
}

// ActivateCommitteeSigners() marks every signer of the QuorumCertificate as an 'active' committee member
func (s *StateMachine) ActivateCommitteeSigners(qc *lib.QuorumCertificate, vs *lib.ValidatorSet) lib.ErrorI {
	// nothing to do if the certificate, signature, or committee are missing
	if qc == nil || qc.Signature == nil || qc.Header == nil || vs == nil {
		return nil
	}
	// retrieve the public keys of those who signed the certificate
	signers, _, err := qc.Signature.GetSigners(*vs)
	if err != nil {
		return err
	}
	chainId := qc.Header.ChainId
	// for each signer; ensure it is recorded as an 'active' committee member
	for _, pk := range signers {
		pubKey, cryptoErr := crypto.NewPublicKeyFromBytes(pk)
		if cryptoErr != nil {
			return lib.ErrPubKeyFromBytes(cryptoErr)
		}
		address := pubKey.Address()
		active, e := s.IsCommitteeMemberActive(chainId, address)
		if e != nil {
			return e
		}
		if !active {
			if e = s.SetCommitteeMemberActive(chainId, address); e != nil {
				return e
			}
		}
	}
	return nil
}

// filterProvisionalNonSigners() drops provisional (zero-power) members from the non-signer list so they
// are not penalized for failing to sign a committee they don't yet hold voting power in
func (s *StateMachine) filterProvisionalNonSigners(vs *lib.ValidatorSet, nonSignerPubKeys [][]byte) [][]byte {
	if vs == nil {
		return nonSignerPubKeys
	}
	filtered := make([][]byte, 0, len(nonSignerPubKeys))
	for _, pk := range nonSignerPubKeys {
		val, err := vs.GetValidator(pk)
		// keep the non-signer only if it is a genuine (non-zero voting power) committee member
		if err != nil || val == nil || val.VotingPower == 0 {
			continue
		}
		filtered = append(filtered, pk)
	}
	return filtered
}

// deleteActiveCommitteesOnLeave() drops the 'active' status for committees the validator is leaving
// (in oldCommittees but not newCommittees), so a re-join must prove liveness again
func (s *StateMachine) deleteActiveCommitteesOnLeave(address crypto.AddressI, oldCommittees, newCommittees []uint64) lib.ErrorI {
	for _, chainId := range oldCommittees {
		if slices.Contains(newCommittees, chainId) {
			continue
		}
		if err := s.DeleteCommitteeMemberActive(chainId, address); err != nil {
			return err
		}
	}
	return nil
}

// COMMITTEE LIVENESS RECOVERY (STALL WATCHDOG) CODE BELOW
// protocol v3+: an already-active member holding > 1/3 of committee power can still halt the committee by
// going dark after proving liveness (a nested-chain deadlock: no QC can form to auto-pause it). The root
// chain keeps producing blocks, so it detects the stall and demotes the blocker back to 'provisional';
// remaining online members can then reach quorum, and the blocker re-activates if it signs a QC again

// CommitteeLivenessWindow is the number of root-chain blocks a committee may go without a certificate
// result before it is considered 'stalled' and eligible for automatic liveness recovery
const CommitteeLivenessWindow = uint64(60)

// HandleStalledCommittees() demotes the largest active member of any committee that has gone without a
// certificate result for CommitteeLivenessWindow blocks (root chain only, protocol v3+)
func (s *StateMachine) HandleStalledCommittees() lib.ErrorI {
	// only applies under protocol v3+
	if !s.IsFeatureEnabled(3) {
		return nil
	}
	// committees are only built/served on a root chain; nested chains have nothing to recover here
	rootChainId, err := s.GetRootChainId()
	if err != nil {
		return err
	}
	if s.Config.ChainId != rootChainId {
		return nil
	}
	// retrieve the master list of committee data
	committeesData, err := s.GetCommitteesData()
	if err != nil {
		return err
	}
	rootHeight := s.Height()
	// for each committee with recorded data
	for _, data := range committeesData.List {
		// skip the self (root) chain: its own liveness is implied by this block being produced
		if data.ChainId == s.Config.ChainId {
			continue
		}
		// skip committees that have never produced a certificate (join-gating already covers those)
		if data.LastRootHeightUpdated == 0 || rootHeight <= data.LastRootHeightUpdated {
			continue
		}
		// skip committees that are still within the liveness window (not yet considered stalled)
		if rootHeight-data.LastRootHeightUpdated < CommitteeLivenessWindow {
			continue
		}
		// the committee is stalled; attempt to restore liveness
		if err = s.demoteStalledCommitteeBlocker(data.ChainId); err != nil {
			return err
		}
	}
	return nil
}

// demoteStalledCommitteeBlocker() demotes a stalled committee's largest active member back to provisional,
// but only if it alone can block a +2/3 quorum (power > 1/3 of active total) and another active member remains
func (s *StateMachine) demoteStalledCommitteeBlocker(chainId uint64) lib.ErrorI {
	// load the 'active' member set; if empty the committee is in bootstrap and there is nothing to demote
	active, err := s.GetActiveCommitteeMemberSet(chainId)
	if err != nil {
		return err
	}
	if len(active) == 0 {
		return nil
	}
	// build the current (gated) committee to read the effective voting powers
	vs, err := s.GetCommitteeMembers(chainId)
	if err != nil {
		return err
	}
	// find the largest active member (nonzero voting power) and the total active power
	var activeTotal, largestPower uint64
	var largestPubKey []byte
	activeCount := 0
	for _, m := range vs.ValidatorSet.ValidatorSet {
		// provisional members carry zero voting power; skip them
		if m.VotingPower == 0 {
			continue
		}
		activeCount++
		activeTotal += m.VotingPower
		// deterministic selection: highest power, breaking ties by the lexicographically smaller pubkey
		if largestPubKey == nil || m.VotingPower > largestPower ||
			(m.VotingPower == largestPower && bytes.Compare(m.PublicKey, largestPubKey) < 0) {
			largestPower, largestPubKey = m.VotingPower, m.PublicKey
		}
	}
	// need at least two active members so demotion leaves someone able to run the chain
	if activeCount <= 1 || largestPubKey == nil {
		return nil
	}
	// only demote a member that can single-handedly block a +2/3 quorum (power > 1/3 of the active total)
	if largestPower <= activeTotal/3 {
		return nil
	}
	// resolve the address and demote the member back to provisional
	addrBz, err := s.pubKeyBytesToAddress(largestPubKey)
	if err != nil {
		return err
	}
	address := crypto.NewAddressFromBytes(addrBz)
	s.log.Warnf("committee %d stalled for >= %d blocks; demoting largest active member %s (power %d) to restore liveness",
		chainId, CommitteeLivenessWindow, address.String(), largestPower)
	return s.DeleteCommitteeMemberActive(chainId, address)
}

// DELEGATIONS BELOW

// GetDelegates returns the active delegates for a given chainId.
// If MaximumDelegatesPerCommittee (from governance params) is 0, it will return all delegates; otherwise it returns only the top N.
func (s *StateMachine) GetDelegates(chainId uint64) (vs lib.ValidatorSet, err lib.ErrorI) {
	return s.getValidatorSet(chainId, true)
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
	// protocol v2+ derives delegates from validator records; skip legacy delegate index writes.
	if s.IsFeatureEnabled(2) {
		return nil
	}
	return s.Set(KeyForDelegate(chainId, address, stakeForCommittee), nil)
}

// DeleteDelegate() removes a delegate from the state using the delegate prefix
func (s *StateMachine) DeleteDelegate(address crypto.AddressI, chainId, stakeForCommittee uint64) lib.ErrorI {
	// protocol v2+ stops mutating legacy delegate index keys.
	if s.IsFeatureEnabled(2) {
		return nil
	}
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
