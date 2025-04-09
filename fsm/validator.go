package fsm

import (
	"bytes"
	"encoding/json"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
	"sort"
)

/* This file implements state actions on Validators and Delegators*/

// GetValidator() gets the validator from the store via the address
func (s *StateMachine) GetValidator(address crypto.AddressI) (*Validator, lib.ErrorI) {
	// get the bytes from state using the key for a validator at a specific address
	list, err := s.GetValidators()
	if err != nil {
		return nil, err
	}
	// populate the cache
	if len(s.valsCache) == 0 {
		// create the val cache
		s.valsCache = make(map[string]*Validator)
		// populate the cache
		for _, val := range list {
			// add each validator to the map keyed by their address
			s.valsCache[lib.BytesToString(val.Address)] = val
		}
	}
	// get the validator from the cache
	v, exists := s.valsCache[address.String()]
	// if the validator doesn't exist
	if !exists {
		// exit
		return nil, ErrValidatorNotExists()
	}
	return v, nil
}

// SetValidator() sets a Validator object into the state
// IMPORTANT: 'setting' doesn't 'update' a validator unless it has the same stake as before!
// - Use s.UpdateValidator() instead
func (s *StateMachine) SetValidator(v *Validator) lib.ErrorI {
	return s.UpdateValidator(v, v.StakedAmount)
}

// UpdateValidator()
// - takes a Validator object with the 'old stake amount'
// - deletes it from the Validators list
// - updates the validator with the 'new stake amount'
// - inserts it in the proper order (sorted by stake then address)
func (s *StateMachine) UpdateValidator(val *Validator, newStakeAmount uint64) lib.ErrorI {
	// get the list of validators / delegates sorted by stake then address
	list, idx, found, err := s.GetValidatorIndexFromList(val.Address, val.StakedAmount, val.Delegate)
	// if an error occurred
	if err != nil {
		// exit with error
		return err
	}
	// if found in the list
	if found {
		// remove it from the list
		list.List = append(list.List[:idx], list.List[idx+1:]...)
	}
	// update the new stake amount
	val.StakedAmount = newStakeAmount
	// if cache is initialized
	if len(s.valsCache) != 0 {
		// update the cache
		s.valsCache[lib.BytesToString(val.Address)] = val
	}
	// get the index from the validator list
	i, _ := s.getValidatorIndexFromList(list, val.Address, val.StakedAmount)
	// if the index is at the end of the list (new entry)
	list.List = append(list.List[:i], append([]*Validator{val}, list.List[i:]...)...)
	// set the updated list back in state
	return s.SetValidatorList(list, val.Delegate)
}

// UpdateValidatorStake() updates the stake of the validator object in state - updating the corresponding committees and supply
// NOTE: new stake amount must be GTE the previous stake amount
func (s *StateMachine) UpdateValidatorStake(val *Validator, newCommittees []uint64, amountToAdd uint64) (err lib.ErrorI) {
	// update staked supply accordingly (validator stake amount can never go down except for slashing and unstaking)
	if err = s.AddToStakedSupply(amountToAdd); err != nil {
		return err
	}
	// calculate the new stake amount
	newStakedAmount := val.StakedAmount + amountToAdd
	// if the validator is a delegate or not
	if val.Delegate {
		// update the new 'total delegated tokens' amount by adding to the staked supply
		if err = s.AddToDelegateSupply(amountToAdd); err != nil {
			return err
		}
		// update the delegations with the new chainIds and stake amount
		if err = s.UpdateDelegations(val, newStakedAmount, newCommittees); err != nil {
			return err
		}
	} else {
		// update the committees with the new chainIds and stake amount
		if err = s.UpdateCommittees(val, newStakedAmount, newCommittees); err != nil {
			return err
		}
	}
	// update the validator committees in the structure
	val.Committees = newCommittees
	// set validator
	return s.UpdateValidator(val, newStakedAmount)
}

// DeleteValidator() completely removes a validator from the state
func (s *StateMachine) DeleteValidator(validator *Validator) lib.ErrorI {
	// subtract from staked supply
	if err := s.SubFromStakedSupply(validator.StakedAmount); err != nil {
		return err
	}
	// delete the validator committee information
	if validator.Delegate {
		// subtract those tokens from the delegate supply count
		if err := s.SubFromDelegateSupply(validator.StakedAmount); err != nil {
			return err
		}
		// remove the delegations for the validator
		if err := s.RemoveDelegationsFromCommittees(validator.StakedAmount, validator.Committees); err != nil {
			return err
		}
	} else {
		// removes the validator stake from the committees
		if err := s.RemoveStakeFromCommittees(validator.StakedAmount, validator.Committees); err != nil {
			return err
		}
	}
	// if cache is initialized
	if len(s.valsCache) != 0 {
		// update the cache
		delete(s.valsCache, lib.BytesToString(validator.Address))
	}
	// delete the validator from state
	return s.DeleteValidatorFromList(validator.Address, validator.StakedAmount, validator.Delegate)
}

// MASS GETTERS AND SETTERS BELOW

// GetValidators() returns a slice of all validators
func (s *StateMachine) GetValidators() (result []*Validator, err lib.ErrorI) {
	// get the validators first
	validators, err := s.GetValidatorList(false)
	if err != nil {
		return nil, err
	}
	// get the delegates next
	delegates, err := s.GetValidatorList(true)
	if err != nil {
		return nil, err
	}
	// exit
	return append(validators.List, delegates.List...), nil
}

// SetValidators() inserts multiple Validators into the state and updates the supply tracker
func (s *StateMachine) SetValidators(validators []*Validator, supply *Supply) lib.ErrorI {
	// for each validator in the list
	for _, val := range validators {
		// if the unstaking height or the max paused height is set
		if val.UnstakingHeight != 0 {
			// if the validator is unstaking - update it accordingly in state
			if err := s.SetValidatorUnstaking(crypto.NewAddress(val.Address), val, val.UnstakingHeight); err != nil {
				return err
			}
		} else if val.MaxPausedHeight != 0 {
			// if validator is paused - update it accordingly
			if err := s.SetValidatorPaused(crypto.NewAddress(val.Address), val, val.MaxPausedHeight); err != nil {
				return err
			}
		}
		// add to 'total supply' in the supply tracker
		supply.Total += val.StakedAmount
		// add to 'staked supply' in the supply tracker
		supply.Staked += val.StakedAmount
		// set the validator structure in state
		if err := s.SetValidator(val); err != nil {
			return err
		}
		// if the validator is a 'delegate'
		if val.Delegate {
			// add to the delegation supply
			supply.DelegatedOnly += val.StakedAmount
			// set the delegations in state
			if err := s.AddDelegationsToCommittees(val.StakedAmount, val.Committees); err != nil {
				return err
			}
		} else {
			if err := s.AddStakeToCommittees(val.StakedAmount, val.Committees); err != nil {
				return err
			}
		}
	}
	//
	// get the 'supply tracker' from state to update the local 'supply tracker' with the automatically populated committee/delegate staked pool
	supplyFromState, err := s.GetSupply()
	if err != nil {
		return err
	}
	// update the committee staked pool
	supply.CommitteeStaked = supplyFromState.CommitteeStaked
	// update the delegate staked pool
	supply.CommitteeDelegatedOnly = supplyFromState.CommitteeDelegatedOnly
	return nil
}

// GetValidatorsPaginated() returns a page of filtered validators
func (s *StateMachine) GetValidatorsPaginated(p lib.PageParams, f lib.ValidatorFilters) (page *lib.Page, err lib.ErrorI) {
	// get all validators
	vals, err := s.GetValidators()
	if err != nil {
		return nil, err
	}
	// initialize a page and the results slice
	page, res := lib.NewPage(p, ValidatorsPageName), make(ValidatorPage, 0)
	page.Results = &res
	// create a variable to hold the list of filtered validators
	var filteredVals []*Validator
	// for each item in the iterator
	for _, val := range vals {
		// pre-filter the possible candidates
		if val.PassesFilter(f) {
			// add to the list
			filteredVals = append(filteredVals, val)
		}
	}
	// populate the page with the list of filtered validators
	err = page.LoadArray(filteredVals, &res, func(i any) (e lib.ErrorI) {
		// cast to validator
		v, ok := i.(*Validator)
		// ensure the cast was successful
		if !ok {
			return lib.ErrInvalidArgument()
		}
		// add to the resulting page
		res = append(res, v)
		// exit
		return
	})
	return
}

// VALIDATORS LIST LOGIC BELOW
// There are only 2 master lists in state: Validators and Delegators

// SetValidatorList() sets a list of validators to the state
func (s *StateMachine) SetValidatorList(list *ValidatorsList, delegate bool) (err lib.ErrorI) {
	var key []byte
	// if delegates
	if delegate {
		// non-bft participants
		key = DelegatePrefix()
	} else {
		// bft participants
		key = ValidatorPrefix()
	}
	// convert the object to proto bytes
	protoBytes, err := lib.Marshal(list)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	return s.Set(key, protoBytes)
}

// GetValidatorList() gets a list of validators from state, there are 2 lists in the state:
// - Delegates (non-bft participants)
// - Validators (bft participants)
func (s *StateMachine) GetValidatorList(delegate bool) (list *ValidatorsList, err lib.ErrorI) {
	var key []byte
	// if delegates
	if delegate {
		// non-bft participants
		key = DelegatePrefix()
	} else {
		// bft participants
		key = ValidatorPrefix()
	}
	// initialize the proto addresses
	list = new(ValidatorsList)
	// get the addresses bytes under the committee prefix
	protoBytes, err := s.Get(key)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// if not found
	if protoBytes == nil {
		// exit without error
		return new(ValidatorsList), nil
	}
	// populate the struct with proto bytes
	err = lib.Unmarshal(protoBytes, list)
	// exit
	return
}

// GetValidatorIndexFromList() is a pass through function to getValidatorIndexFromList() that retrieves the list first
func (s *StateMachine) GetValidatorIndexFromList(address []byte, stake uint64, delegate bool) (list *ValidatorsList, i int, found bool, err lib.ErrorI) {
	// get the list of validators / delegates sorted (stake then address)
	list, err = s.GetValidatorList(delegate)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// perform a binary search on the validators list
	i, found = s.getValidatorIndexFromList(list, address, stake)
	// exit
	return
}

// DeleteValidatorFromList() removes the validator from the list sorted by stake then address
func (s *StateMachine) DeleteValidatorFromList(address []byte, stake uint64, delegate bool) (err lib.ErrorI) {
	// get the validator's index in the list
	list, idx, found, err := s.GetValidatorIndexFromList(address, stake, delegate)
	// if an error occurred
	if err != nil {
		return // error
	}
	// if not found in the list
	if !found {
		return // no error
	}
	// remove it from the list
	list.List = append(list.List[:idx], list.List[idx+1:]...)
	// set the list in state
	return s.SetValidatorList(list, delegate)
}

// getValidatorIndexFromList() performs a binary search on the validators list which is sorted
// first by stake then by address and returns the list, index and if the validator is found
func (s *StateMachine) getValidatorIndexFromList(list *ValidatorsList, address []byte, stake uint64) (i int, found bool) {
	// get the size of the list
	n := len(list.List)
	// insert using binary search
	i = sort.Search(n, func(i int) bool {
		// load the validator from the state using the address
		v := list.List[i]
		// if stakes are equal, sort by address in lexicographical order
		if v.StakedAmount == stake {
			return bytes.Compare(v.Address, address) <= 0
		}
		// sort the highest stake to lowest
		return v.StakedAmount <= stake
	})
	// if the index is at the end of the list
	if i == n {
		// not found
		return
	}
	// check if it's the correct address
	found = bytes.Equal(list.List[i].Address, address)
	// exit
	return
}

// UNSTAKING VALIDATORS BELOW

// SetValidatorUnstaking() updates a Validator as 'unstaking' and removes it from its respective committees
// NOTE: finish unstaking height is the height in the future when the validator will be deleted and their
// funds be returned
func (s *StateMachine) SetValidatorUnstaking(address crypto.AddressI, validator *Validator, finishUnstakingHeight uint64) lib.ErrorI {
	// get the address list for those unstaking
	list, err := s.GetAddressList(UnstakingPrefix(finishUnstakingHeight))
	// if an error occurred
	if err != nil {
		// exit with error
		return err
	}
	// add to the unstaking list
	s.AddToAddressList(address.Bytes(), list)
	// set the address list back in state
	if err = s.SetAddressList(UnstakingPrefix(finishUnstakingHeight), list); err != nil {
		return err
	}
	// if validator is 'paused' (only happens if validator is max paused)
	if validator.MaxPausedHeight != 0 {
		// update the validator as unpaused
		if err = s.SetValidatorUnpaused(address, validator); err != nil {
			return err
		}
	}
	// update the validator structure with the finishUnstakingHeight
	validator.UnstakingHeight = finishUnstakingHeight
	// update the validator structure
	return s.SetValidator(validator)
}

// DeleteFinishedUnstaking() deletes the Validator structure and unstaking keys for those who have finished unstaking
func (s *StateMachine) DeleteFinishedUnstaking() (err lib.ErrorI) {
	// define the key for the list
	key := UnstakingPrefix(s.Height())
	// get the address list from state
	list, err := s.GetAddressList(key)
	// if an error occurred
	if err != nil {
		// exit with error
		return
	}
	// for each address in the list
	for _, address := range list.Addresses {
		// get the validator associated with that address
		v, e := s.GetValidator(crypto.NewAddress(address))
		if e != nil {
			return e
		}
		// transfer the staked tokens to the designated output address
		if err = s.AccountAdd(crypto.NewAddressFromBytes(v.Output), v.StakedAmount); err != nil {
			return err
		}
		// delete the validator structure
		if err = s.DeleteValidator(v); err != nil {
			return err
		}
	}
	// remove the list
	return s.Delete(key)
}

// PAUSED VALIDATORS BELOW

// SetValidatorsPaused() automatically updates all validators as if they'd submitted a MessagePause
func (s *StateMachine) SetValidatorsPaused(chainId uint64, addresses [][]byte) {
	// for each validator in the list
	for _, addr := range addresses {
		// get the validator
		val, err := s.GetValidator(crypto.NewAddress(addr))
		if err != nil {
			// log error
			s.log.Debugf("can't pause validator %s not found", lib.BytesToString(addr))
			// move on to the next iteration
			continue
		}
		// ensure no unauthorized auto-pauses
		if !slices.Contains(val.Committees, chainId) {
			// NOTE: expected - this can happen during a race between edit-stake and pause
			s.log.Warnf("unauthorized pause from %d, this can happen occasionally", chainId)
			// exit
			return
		}
		// handle pausing the validator
		if err = s.HandleMessagePause(&MessagePause{Address: addr}); err != nil {
			// log error
			s.log.Debugf("can't pause validator %s with err %s", lib.BytesToString(addr), err.Error())
			// move on to the next iteration
			continue
		}
	}
}

// SetValidatorPaused() updates a Validator as 'paused' with a MaxPausedHeight (height at which the Validator is force-unstaked for being paused too long)
func (s *StateMachine) SetValidatorPaused(address crypto.AddressI, validator *Validator, maxPausedHeight uint64) lib.ErrorI {
	// get the list of validators that have this max paused height
	list, err := s.GetAddressList(PausedPrefix(maxPausedHeight))
	if err != nil {
		return err
	}
	// add the address to the list
	s.AddToAddressList(address.Bytes(), list)
	// set the list in state
	if err = s.SetAddressList(PausedPrefix(maxPausedHeight), list); err != nil {
		return err
	}
	// update the validator max paused height
	validator.MaxPausedHeight = maxPausedHeight
	// set the updated validator in state
	return s.SetValidator(validator)
}

// SetValidatorUnpaused() updates a Validator as 'unpaused'
func (s *StateMachine) SetValidatorUnpaused(address crypto.AddressI, validator *Validator) lib.ErrorI {
	// get the address list from state
	list, err := s.GetAddressList(PausedPrefix(validator.MaxPausedHeight))
	// if an error occurred
	if err != nil {
		// exit with error
		return err
	}
	// remove the address from the list
	s.DeleteFromAddressList(address.Bytes(), list)
	// set the list back in state
	if err = s.SetAddressList(PausedPrefix(validator.MaxPausedHeight), list); err != nil {
		return err
	}
	// update the validator max paused height to 0
	validator.MaxPausedHeight = 0
	// set the updated validator in state
	return s.SetValidator(validator)
}

// GetAuthorizedSignersForValidator() returns the addresses that are able to sign messages on behalf of the validator
func (s *StateMachine) GetAuthorizedSignersForValidator(address []byte) (signers [][]byte, err lib.ErrorI) {
	// retrieve the validator from state
	v, err := s.GetValidator(crypto.NewAddressFromBytes(address))
	if err != nil {
		return nil, err
	}
	// return the operator only if custodial
	if bytes.Equal(v.Address, v.Output) {
		return [][]byte{v.Address}, nil
	}
	// return the operator and output
	return [][]byte{v.Address, v.Output}, nil
}

// pubKeyBytesToAddress() is a convenience function that converts a public key to an address
func (s *StateMachine) pubKeyBytesToAddress(public []byte) ([]byte, lib.ErrorI) {
	// get the public key object ref from the bytes
	pk, err := crypto.NewPublicKeyFromBytes(public)
	if err != nil {
		return nil, ErrInvalidPublicKey(err)
	}
	// get the address bytes from the public key
	return pk.Address().Bytes(), nil
}

// marshalValidator() converts the Validator object to bytes
func (s *StateMachine) marshalValidator(validator *Validator) ([]byte, lib.ErrorI) {
	// convert the object ref into bytes
	return lib.Marshal(validator)
}

// unmarshalValidator() converts bytes into a Validator object
func (s *StateMachine) unmarshalValidator(bz []byte) (*Validator, lib.ErrorI) {
	// create a new validator object reference to ensure a non-nil result
	val := new(Validator)
	// populate the object reference with validator bytes
	if err := lib.Unmarshal(bz, val); err != nil {
		return nil, err
	}
	// return the object ref
	return val, nil
}

// VALIDATOR HELPERS BELOW

// validator is the json.Marshaller and json.Unmarshaler implementation for the Validator object
type validator struct {
	Address         *crypto.Address           `json:"address"`
	PublicKey       *crypto.BLS12381PublicKey `json:"publicKey"`
	Committees      []uint64                  `json:"committees"`
	NetAddress      string                    `json:"netAddress"`
	StakedAmount    uint64                    `json:"stakedAmount"`
	MaxPausedHeight uint64                    `json:"maxPausedHeight"`
	UnstakingHeight uint64                    `json:"unstakingHeight"`
	Output          *crypto.Address           `json:"output"`
	Delegate        bool                      `json:"delegate"`
	Compound        bool                      `json:"compound"`
}

// MarshalJSON() is the json.Marshaller implementation for the Validator object
func (x *Validator) MarshalJSON() ([]byte, error) {
	publicKey, err := crypto.BytesToBLS12381Public(x.PublicKey)
	if err != nil {
		return nil, err
	}
	return json.Marshal(validator{
		Address:         crypto.NewAddressFromBytes(x.Address).(*crypto.Address),
		PublicKey:       publicKey.(*crypto.BLS12381PublicKey),
		Committees:      x.Committees,
		NetAddress:      x.NetAddress,
		StakedAmount:    x.StakedAmount,
		MaxPausedHeight: x.MaxPausedHeight,
		UnstakingHeight: x.UnstakingHeight,
		Output:          crypto.NewAddressFromBytes(x.Output).(*crypto.Address),
		Delegate:        x.Delegate,
		Compound:        x.Compound,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the Validator object
func (x *Validator) UnmarshalJSON(bz []byte) error {
	val := new(validator)
	if err := json.Unmarshal(bz, val); err != nil {
		return err
	}
	*x = Validator{
		Address:         val.Address.Bytes(),
		PublicKey:       val.PublicKey.Bytes(),
		NetAddress:      val.NetAddress,
		StakedAmount:    val.StakedAmount,
		Committees:      val.Committees,
		MaxPausedHeight: val.MaxPausedHeight,
		UnstakingHeight: val.UnstakingHeight,
		Output:          val.Output.Bytes(),
		Delegate:        val.Delegate,
		Compound:        val.Compound,
	}
	return nil
}

// PassesFilter() returns if the Validator object passes the filter (true) or is filtered out (false)
func (x *Validator) PassesFilter(f lib.ValidatorFilters) (ok bool) {
	switch {
	case f.Unstaking == lib.FilterOption_MustBe:
		if x.UnstakingHeight == 0 {
			return
		}
	case f.Unstaking == lib.FilterOption_Exclude:
		if x.UnstakingHeight != 0 {
			return
		}
	}
	switch {
	case f.Paused == lib.FilterOption_MustBe:
		if x.MaxPausedHeight == 0 {
			return
		}
	case f.Paused == lib.FilterOption_Exclude:
		if x.MaxPausedHeight != 0 {
			return
		}
	}
	switch {
	case f.Delegate == lib.FilterOption_MustBe:
		if !x.Delegate {
			return
		}
	case f.Delegate == lib.FilterOption_Exclude:
		if x.Delegate {
			return
		}
	}
	if f.Committee != 0 {
		if !slices.Contains(x.Committees, f.Committee) {
			return
		}
	}
	return true
}

func init() {
	// Register the pages for converting bytes of Page into the correct Page object
	lib.RegisteredPageables[ValidatorsPageName] = new(ValidatorPage)
	lib.RegisteredPageables[ConsValidatorsPageName] = new(ConsValidatorPage)
}

const (
	ValidatorsPageName     = "validators"           // name for page of 'Validators'
	ConsValidatorsPageName = "consensus_validators" // name for page of 'Consensus Validators' (only essential val info needed for consensus)
)

type ValidatorPage []*Validator

// ValidatorPage satisfies the Page interface
func (p *ValidatorPage) New() lib.Pageable { return &ValidatorPage{{}} }

type ConsValidatorPage []*lib.ConsensusValidator

// ConsValidatorPage satisfies the Page interface
func (p *ConsValidatorPage) New() lib.Pageable { return &ConsValidatorPage{{}} }
