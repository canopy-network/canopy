package fsm

import (
	"slices"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
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

	// SLASH RECIPIENTS LOGIC

	// define a convenience variable for the slash recipients
	slashRecipients := qc.Results.SlashRecipients
	// sanity check the slash recipient isn't nil
	if slashRecipients != nil {
		// set in state and slash double signers
		if err = s.HandleDoubleSigners(qc.Header.ChainId, params, slashRecipients.DoubleSigners); err != nil {
			return 0, err
		}
		// slash the explicit recipients
		for _, recipient := range slashRecipients.SlashRecipients {
			// slash the validator
			if err = s.SlashValidator(recipient.Address, qc.Header.ChainId, recipient.Percent, params); err != nil {
				return 0, err
			}
		}
	}

	return
}

// SlashAndResetNonSigners() resets the non-signer tracking and slashes those who exceeded the MaxNonSign threshold
func (s *StateMachine) SlashAndResetNonSigners(chainId uint64, params *ValidatorParams) (err lib.ErrorI) {
	var keys, slashList [][]byte
	// execute the callback for each key under 'non-signer' prefix
	// for every key under 'non-signer' prefix; slashes the validator if exceeded the MaxNonSign threshold
	err = s.IterateAndExecute(NonSignerPrefix(), func(k, v []byte) (err lib.ErrorI) {
		// track non-signer keys to delete
		keys = append(keys, k)
		// for each non-signer, see if they exceeded the threshold
		// if so - add them to the bad list
		addr, err := AddressFromKey(k)
		if err != nil {
			return
		}
		// ensure no nil NonSigner
		ptr := new(NonSigner)
		// convert the value into a non-signer
		if err = lib.Unmarshal(v, ptr); err != nil {
			return
		}
		// if the counter exceeds the max-non sign
		if ptr.Counter > params.MaxNonSign {
			// add the address to the 'bad list'
			slashList = append(slashList, addr.Bytes())
		}
		return
	})
	if err != nil {
		return err
	}
	// pause all on the bad list
	s.SetValidatorsPaused(chainId, slashList)
	// slash all on the bad list
	if err = s.SlashNonSigners(chainId, params, slashList); err != nil {
		return
	}
	// delete all keys under 'non-signer' prefix as part of the reset
	_ = s.DeleteAll(keys)
	return
}

// GetNonSigners() returns all non-quorum-certificate-signers save in the state
func (s *StateMachine) GetNonSigners() (results NonSigners, e lib.ErrorI) {
	// create an iterator for the non-signer prefix
	it, e := s.Iterator(NonSignerPrefix())
	if e != nil {
		return
	}
	defer it.Close()
	// for each item of the iterator
	for ; it.Valid(); it.Next() {
		// get the address from the iterator key
		addr, err := AddressFromKey(it.Key())
		if err != nil {
			return nil, err
		}
		// define a non-signer object reference to ensure no nil
		ptr := new(NonSigner)
		// unmarshal the value into a ptr
		if err = lib.Unmarshal(it.Value(), ptr); err != nil {
			return nil, err
		}
		// add the non-signer to the list
		results = append(results, &NonSigner{
			Address: addr.Bytes(), // address is stored in the 'key' not the 'value'
			Counter: ptr.Counter,
		})
	}
	return
}

// GetDoubleSigners() returns all double signers save in the state
// IMPORTANT NOTE: this returns <address> -> <heights> NOT <pubic_key> -> <heights>
// CONTRACT: Querying uncommitted double signers is not supported
func (s *StateMachine) GetDoubleSigners() (results []*lib.DoubleSigner, e lib.ErrorI) {
	return s.Store().(lib.StoreI).GetDoubleSigners()
}

// IncrementNonSigners() upserts non-(QC)-signers by incrementing the non-signer count for the list
func (s *StateMachine) IncrementNonSigners(nonSignerPubKeys [][]byte) lib.ErrorI {
	// for each non-signer in the list
	for _, ns := range nonSignerPubKeys {
		// extract the public key from the list
		pubKey, e := crypto.NewPublicKeyFromBytes(ns)
		if e != nil {
			return lib.ErrPubKeyFromBytes(e)
		}
		// create a key for the non-signer
		key := KeyForNonSigner(pubKey.Address().Bytes())
		// get the value bytes for the non-signer
		bz, err := s.Get(key)
		if err != nil {
			return err
		}
		// create a non-signer object reference to ensure a non-nil result
		ptr := new(NonSigner)
		// convert the value bytes into a non-signer object
		if err = lib.Unmarshal(bz, ptr); err != nil {
			return err
		}
		// increment the counter for the non-signer
		ptr.Counter++
		// set convert the object ref back to bytes
		bz, err = lib.Marshal(ptr)
		if err != nil {
			return err
		}
		// set the object bytes in the store
		if err = s.Set(key, bz); err != nil {
			return err
		}
	}
	return nil
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
	// pause all on the bad list
	//s.SetValidatorsPaused(chainId, slashList)
	// slash those on the list
	return s.SlashDoubleSigners(chainId, params, slashList)
}

// SlashNonSigners() burns the staked tokens of non-quorum-certificate-signers
func (s *StateMachine) SlashNonSigners(chainId uint64, params *ValidatorParams, nonSignerAddrs [][]byte) lib.ErrorI {
	return s.SlashValidators(nonSignerAddrs, chainId, params.NonSignSlashPercentage, params)
}

// SlashDoubleSigners() burns the staked tokens of double signers
func (s *StateMachine) SlashDoubleSigners(chainId uint64, params *ValidatorParams, doubleSignerAddrs [][]byte) lib.ErrorI {
	return s.SlashValidators(doubleSignerAddrs, chainId, params.DoubleSignSlashPercentage, params)
}

// ForceUnstakeValidator() automatically begins unstaking the validator
func (s *StateMachine) ForceUnstakeValidator(address crypto.AddressI) lib.ErrorI {
	// get the validator object from state
	validator, err := s.GetValidator(address)
	if err != nil {
		s.log.Warnf("validator %s is not found to be force unstaked", address.String()) // defensive
		return nil
	}
	// check if already unstaking
	if validator.UnstakingHeight != 0 {
		s.log.Warnf("validator %s is already unstaking can't be forced to begin unstaking", address.String())
		return nil
	}
	// get params for unstaking blocks
	p, err := s.GetParamsVal()
	if err != nil {
		return err
	}
	// get the unstaking blocks from the parameters
	unstakingBlocks := p.GetUnstakingBlocks()
	// calculate the future unstaking height
	unstakingHeight := s.Height() + unstakingBlocks
	// set the validator as unstaking
	return s.SetValidatorUnstaking(address, validator, unstakingHeight)
}

// SlashValidators() burns a specified percentage of multiple validator's staked tokens
func (s *StateMachine) SlashValidators(addresses [][]byte, chainId, percent uint64, p *ValidatorParams) lib.ErrorI {
	// for each address in the list
	for _, addr := range addresses {
		// slash the validator
		if err := s.SlashValidator(addr, chainId, percent, p); err != nil {
			return err
		}
	}
	return nil
}

// SlashValidator() burns a specified percentage of a validator's staked tokens
func (s *StateMachine) SlashValidator(address []byte, chainId, percent uint64, p *ValidatorParams) (err lib.ErrorI) {
	// retrieve the validator
	validator, err := s.GetValidator(crypto.NewAddressFromBytes(address))
	if err != nil {
		s.log.Warn(ErrSlashNonExistentValidator().Error())
		return nil
	}
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
	addr, stakeAfterSlash := crypto.NewAddressFromBytes(validator.Address), lib.Uint64ReducePercentage(validator.StakedAmount, percent)
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
	if err = s.UpdateCommittees(addr, validator, stakeAfterSlash, newCommittees); err != nil {
		return err
	}
	// set the committees in the validator structure
	validator.Committees = newCommittees
	// update the stake amount and set the validator
	validator.StakedAmount = stakeAfterSlash
	// get gov params for validation
	params, err := s.GetParams()
	if err != nil {
		return err
	}
	// determine if this validator/delegate is below the new minimum
	var bellowMinimum bool
	var unstakingBlocks uint64
	// validate stake is above minimums
	if validator.Delegate {
		if validator.StakedAmount < params.Governance.MinimumStakeForDelegates {
			bellowMinimum = true
			unstakingBlocks = params.Validator.DelegateUnstakingBlocks
		}
	} else {
		if validator.StakedAmount < params.Governance.MinimumStakeForValidators {
			bellowMinimum = true
			unstakingBlocks = params.Validator.UnstakingBlocks
		}
	}
	// if below minimum, force unstake
	if bellowMinimum {
		// calculate the future unstaking height
		unstakingHeight := s.Height() + unstakingBlocks
		// set the validator/delegate as unstaking
		return s.SetValidatorUnstaking(crypto.NewAddress(validator.Address), validator, unstakingHeight)
	}
	// update the validator
	return s.SetValidator(validator)
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
