package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
)

/* This file contains logic regarding byzantine actor handling and bond slashes */

// HandleByzantine() handles the byzantine (faulty/malicious) participants from a QuorumCertificate
func (s *StateMachine) HandleByzantine(qc *lib.QuorumCertificate, vs *lib.ValidatorSet) (nonSignerPercent int, err lib.ErrorI) {
	// if validator set is nil; chain is not root, so don't handle byzantine evidence
	if vs == nil {
		return
	}
	// get the validator params
	params, err := s.GetParamsVal()
	if err != nil {
		return 0, err
	}

	// NON SIGNER LOGIC

	// if current block marks the ending of the NonSignWindow
	if s.Height()%params.NonSignWindow == 0 {
		// automatically slash and reset any non-signers that exceeded MaxNonSignPerWindow
		if err = s.SlashAndResetNonSigners(qc.Header.ChainId, params); err != nil {
			return 0, err
		}
	}
	// get those who did not sign this particular QC but should have
	nonSignerPubKeys, nonSignerPercent, err := qc.GetNonSigners(vs.ValidatorSet)
	if err != nil {
		return 0, err
	}
	// increment the non-signing count for the non-signers
	if err = s.IncrementNonSigners(nonSignerPubKeys); err != nil {
		return 0, err
	}

	// DOUBLE SIGNER LOGIC

	// define a convenience variable for the slash recipients
	slashRecipients := qc.Results.SlashRecipients
	// sanity check the slash recipient isn't nil
	if slashRecipients != nil {
		// set in state and slash double signers
		if err = s.HandleDoubleSigners(qc.Header.ChainId, params, slashRecipients.DoubleSigners); err != nil {
			return 0, err
		}
	}
	return
}

// NON-SIGNER LOGIC BELOW

// SlashAndResetNonSigners() resets the non-signer tracking and slashes those who exceeded the MaxNonSign threshold
func (s *StateMachine) SlashAndResetNonSigners(chainId uint64, params *ValidatorParams) (err lib.ErrorI) {
	var slashList [][]byte
	// get the non-signers list from the state
	nonSignersList, err := s.GetNonSignersList()
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// for each non signer in the list
	for _, nonSigner := range nonSignersList.List {
		// if the counter exceeds the max-non sign
		if nonSigner.Counter > params.MaxNonSign {
			// add the address to the 'bad list'
			slashList = append(slashList, nonSigner.Address)
		}
	}
	// pause all on the bad list
	s.SetValidatorsPaused(chainId, slashList)
	// slash all on the bad list
	if err = s.SlashNonSigners(chainId, params, slashList); err != nil {
		return
	}
	// delete all keys under 'non-signer' prefix as part of the reset
	_ = s.Delete(NonSignerPrefix())
	return
}

// SlashNonSigners() burns the staked tokens of non-quorum-certificate-signers
func (s *StateMachine) SlashNonSigners(chainId uint64, params *ValidatorParams, nonSignerAddrs [][]byte) lib.ErrorI {
	return s.SlashValidators(nonSignerAddrs, chainId, params.NonSignSlashPercentage, params)
}

// GetNonSigners() returns all non-quorum-certificate-signers save in the state
func (s *StateMachine) GetNonSigners() (results NonSigners, e lib.ErrorI) {
	// retrieve the list of non signers
	nonSigners, e := s.GetNonSignersList()
	// if an error occurred
	if e != nil {
		// exit with error
		return
	}
	// return the list
	return nonSigners.List, nil
}

// IncrementNonSigners() upserts non-(QC)-signers by incrementing the non-signer count for the list
func (s *StateMachine) IncrementNonSigners(nonSignerPubKeys [][]byte) lib.ErrorI {
	// get the non-signers list from the state
	nonSignersList, err := s.GetNonSignersList()
	if err != nil {
		return err
	}
	// for each non-signer in the list
	for _, ns := range nonSignerPubKeys {
		// extract the public key from the list
		pubKey, e := crypto.NewPublicKeyFromBytes(ns)
		if e != nil {
			return lib.ErrPubKeyFromBytes(e)
		}
		// get the address from the public key
		address := pubKey.Address().Bytes()
		// create a convenience variable to see if the non signer was found in the list
		var found bool
		// check the list
		for i, nonSigner := range nonSignersList.List {
			// if found
			if found = bytes.Equal(nonSigner.Address, address); found {
				// update the count
				nonSignersList.List[i].Counter++
			}
		}
		// check if not found in the slice
		if !found {
			// add new entry to the non-signers list
			nonSignersList.List = append(nonSignersList.List, &NonSigner{Address: address, Counter: 1})
		}
	}
	// set the non-signers list back in state
	return s.SetNonSignersList(nonSignersList)
}

// GetNonSignersList() returns a list of non-signers from the state
func (s *StateMachine) GetNonSignersList() (nonSigners *NonSignerList, err lib.ErrorI) {
	nonSigners = new(NonSignerList)
	// get the non-signers list from the state
	protoBytes, err := s.Get(NonSignerPrefix())
	if err != nil {
		return nil, err
	}
	// ensure a non-nil result
	if protoBytes == nil {
		return new(NonSignerList), nil
	}
	// convert the bytes into a structure
	err = lib.Unmarshal(protoBytes, nonSigners)
	// exit
	return
}

// SetNonSignersList() sets a list of non-signers in the state
func (s *StateMachine) SetNonSignersList(nonSigners *NonSignerList) (err lib.ErrorI) {
	// convert the non-signers list to bytes
	protoBytes, err := lib.Marshal(nonSigners)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// set the bytes in the state
	return s.Set(NonSignerPrefix(), protoBytes)
}

// DOUBLE SIGNER LOGIC BELOW

// GetDoubleSigners() returns all double signers save in the state
// IMPORTANT NOTE: this returns <address> -> <heights> NOT <pubic_key> -> <heights>
func (s *StateMachine) GetDoubleSigners() (results []*lib.DoubleSigner, e lib.ErrorI) {
	return s.Store().(lib.StoreI).GetDoubleSigners()
}

// HandleDoubleSigners() validates, sets, and slashes the list of doubleSigners
func (s *StateMachine) HandleDoubleSigners(chainId uint64, params *ValidatorParams, doubleSigners []*lib.DoubleSigner) lib.ErrorI {
	// ensure the store is a StoreI for this call
	store, ok := s.Store().(lib.StoreI)
	if !ok {
		return ErrWrongStoreType()
	}
	// create a list to hold the double signers that will be slashed
	var slashList [][]byte
	// for each double signer
	for _, doubleSigner := range doubleSigners {
		// ensure the double signer isn't nil nor the id is nil
		if doubleSigner == nil || doubleSigner.Id == nil {
			return lib.ErrEmptyDoubleSigner()
		}
		// ensure there's at least 1 height in the list
		if len(doubleSigner.Heights) == 0 {
			return lib.ErrInvalidDoubleSignHeights()
		}
		// convert the double-signer ID to a public key
		pubKey, e := crypto.NewPublicKeyFromBytes(doubleSigner.Id)
		if e != nil {
			return lib.ErrPubKeyFromBytes(e)
		}
		// convert that public key to an address
		address := pubKey.Address().Bytes()
		// for each double sign height
		for _, height := range doubleSigner.Heights {
			// check if the 'double sign' is valid for the address and height
			isValidDS, err := store.IsValidDoubleSigner(address, height)
			if err != nil {
				return err
			}
			// if - it's invalid (already exists) then return invalid
			if !isValidDS {
				return lib.ErrInvalidDoubleSigner()
			}
			// else - index the double signer by address and height
			if err = store.IndexDoubleSigner(address, height); err != nil {
				return err
			}
			// add to slash list
			slashList = append(slashList, pubKey.Address().Bytes())
		}
	}
	// slash those on the list
	return s.SlashDoubleSigners(chainId, params, slashList)
}

// SlashDoubleSigners() burns the staked tokens of double signers
func (s *StateMachine) SlashDoubleSigners(chainId uint64, params *ValidatorParams, doubleSignerAddrs [][]byte) lib.ErrorI {
	return s.SlashValidators(doubleSignerAddrs, chainId, params.DoubleSignSlashPercentage, params)
}

// SLASHING LOGIC BELOW

// SlashValidators() burns a specified percentage of multiple validator's staked tokens
func (s *StateMachine) SlashValidators(addresses [][]byte, chainId, percent uint64, p *ValidatorParams) lib.ErrorI {
	// for each address in the list
	for _, addr := range addresses {
		// retrieve the validator
		val, err := s.GetValidator(crypto.NewAddressFromBytes(addr))
		if err != nil {
			s.log.Warn(ErrSlashNonExistentValidator().Error())
			continue
		}
		// slash the validator
		if err = s.SlashValidator(val, chainId, percent, p); err != nil {
			return err
		}
	}
	return nil
}

// SlashValidator() burns a specified percentage of a validator's staked tokens
func (s *StateMachine) SlashValidator(validator *Validator, chainId, percent uint64, p *ValidatorParams) (err lib.ErrorI) {
	// ensure no unauthorized slashes may occur
	if !slices.Contains(validator.Committees, chainId) {
		// This may happen if an async event causes a validator edit stake to occur before being slashed
		// Non-byzantine actors order 'certificate result' messages before 'edit stake'
		s.log.Warn(ErrInvalidChainId().Error())
		return nil
	}
	// create a convenience variable to hold the new validator committees (in case the validator was ejected)
	newCommittees := slices.Clone(validator.Committees)
	// a 'slash tracker' is used to limit the max slash per committee per block
	// get the slashed percent so far in this block by this committee
	slashTotal := s.slashTracker.GetTotalSlashPercent(validator.Address, chainId)
	// check to see if it exceeds the max
	if slashTotal >= p.MaxSlashPerCommittee {
		return nil // no slash nor no removal logic occurs because this block already hit the limit with a previous slash
	}
	// check to see if it 'now' exceeds the max
	if slashTotal+percent >= p.MaxSlashPerCommittee {
		// only slash up to the maximum
		percent = p.MaxSlashPerCommittee - slashTotal
		// for each committee
		for i, id := range newCommittees {
			// if id is the slash chain id
			if id == chainId {
				// remove the validator from the committee
				newCommittees = append(newCommittees[:i], newCommittees[i+1:]...)
				// exit the loop
				break
			}
		}
	}
	// update the slash tracker
	s.slashTracker.AddSlash(validator.Address, chainId, percent)
	// initialize address and new stake variable
	stakeAfterSlash := lib.Uint64ReducePercentage(validator.StakedAmount, percent)
	// calculate the slash amount
	slashAmount := validator.StakedAmount - stakeAfterSlash
	// subtract from total supply
	if err = s.SubFromTotalSupply(slashAmount); err != nil {
		return err
	}
	// if stake after slash is 0, remove the validator
	if stakeAfterSlash == 0 {
		// DeleteValidator subtracts from staked supply
		return s.DeleteValidator(validator)
	}
	// subtract from staked supply
	if err = s.SubFromStakedSupply(slashAmount); err != nil {
		return err
	}
	// update the committees based on the new stake amount
	if err = s.UpdateCommittees(validator, stakeAfterSlash, newCommittees); err != nil {
		return err
	}
	// set the committees in the validator structure
	validator.Committees = newCommittees
	// update the validator
	return s.UpdateValidator(validator, stakeAfterSlash)
}

// LoadMinimumEvidenceHeight() loads the minimum height the evidence must be to still be usable
func (s *StateMachine) LoadMinimumEvidenceHeight() (uint64, lib.ErrorI) {
	// use the time machine to ensure a clean database transaction
	historicalFSM, err := s.TimeMachine(s.Height())
	// if an error occurred
	if err != nil {
		// exit with error
		return 0, err
	}
	// once function completes, discard it
	defer historicalFSM.Discard()
	// get the validator params from state
	valParams, err := historicalFSM.GetParamsVal()
	if err != nil {
		return 0, err
	}
	// define convenience variables
	height, unstakingBlocks := historicalFSM.Height(), valParams.GetUnstakingBlocks()
	// if height is less than staking blocks, use *genesis* as the minimum evidence height
	if height < unstakingBlocks {
		return 0, nil
	}
	// minimum evidence = unstaking blocks ago
	return height - unstakingBlocks, nil
}

// BYZANTINE HELPERS BELOW

type NonSigners []*NonSigner

// SlashTracker is a map of address -> committee -> slash percentage
// which is used to ensure no committee exceeds max slash within a single block
// NOTE: this slash tracker is naive and doesn't account for the consecutive reduction
// of a slash percentage impact i.e. two 10% slashes = 20%, but technically it's 19%
type SlashTracker map[string]map[uint64]uint64

func NewSlashTracker() *SlashTracker {
	slashTracker := make(SlashTracker)
	return &slashTracker
}

// AddSlash() adds a slash for an address at by a committee for a certain percent
func (s *SlashTracker) AddSlash(address []byte, chainId, percent uint64) {
	// add the percent to the total
	(*s)[s.toKey(address)][chainId] += percent
}

// GetTotalSlashPercent() returns the total percent for a slash
func (s *SlashTracker) GetTotalSlashPercent(address []byte, chainId uint64) (percent uint64) {
	// return the total percent
	return (*s)[s.toKey(address)][chainId]
}

// toKey() converts the address bytes to a string and ensures the map is initialized for that address
func (s *SlashTracker) toKey(address []byte) string {
	// convert the address to a string
	addr := lib.BytesToString(address)
	// if the address has not yet been slashed by any committee
	// create the corresponding committee map
	if _, ok := (*s)[addr]; !ok {
		(*s)[addr] = make(map[uint64]uint64)
	}
	return addr
}
