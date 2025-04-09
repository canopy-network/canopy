package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"math"
	"slices"
	"testing"
)

func TestGetValidator(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
		tryGet crypto.AddressI
		error  string
	}{
		{
			name:   "no preset",
			detail: "no validator was not preset so not exists",
			tryGet: newTestAddress(t),
			error:  "validator does not exist",
		},
		{
			name:   "different validator",
			detail: "the validator that was preset doesn't correspond with the tryGet",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{lib.CanopyChainId},
			},
			tryGet: newTestAddress(t, 1),
			error:  "validator does not exist",
		},
		{
			name:   "single validator",
			detail: "set and get a single validator",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{lib.CanopyChainId},
			},
			tryGet: newTestAddress(t),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validator
			if test.preset != nil {
				require.NoError(t, sm.SetValidator(test.preset))
			}
			// execute the function call
			got, err := sm.GetValidator(test.tryGet)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			require.EqualExportedValues(t, test.preset, got)
		})
	}
}

func TestGetValidatorExists(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
		tryGet crypto.AddressI
		exists bool
	}{
		{
			name:   "no preset",
			detail: "no validator was not preset so not exists",
			tryGet: newTestAddress(t),
			exists: false,
		},
		{
			name:   "different valdiator",
			detail: "the validator that was preset doesn't correspond with the tryGet",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{lib.CanopyChainId},
			},
			tryGet: newTestAddress(t, 1),
			exists: false,
		},
		{
			name:   "single validator",
			detail: "set and get a single validator",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{lib.CanopyChainId},
			},
			tryGet: newTestAddress(t),
			exists: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validator
			if test.preset != nil {
				require.NoError(t, sm.SetValidator(test.preset))
			}
			// execute the function call
			val, _ := sm.GetValidator(test.tryGet)
			// compare got vs expected
			require.Equal(t, test.exists, val != nil)
		})
	}
}

func TestSetGetValidators(t *testing.T) {
	const amount = uint64(100)
	tests := []struct {
		name           string
		detail         string
		preset         []*Validator
		expectedSupply *Supply
	}{
		{
			name:   "validators (non-delegate)",
			detail: "set and get validators (non-delegate)",
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount + 2,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount + 1,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
			},
			expectedSupply: &Supply{
				Total:  amount*3 + 3,
				Staked: amount*3 + 3,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: amount*3 + 3,
					},
					{
						Id:     2,
						Amount: amount*3 + 3,
					},
				},
			},
		},
		{
			name:   "delegates",
			detail: "set and get delegates",
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount + 2,
					Delegate:     true,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount + 1,
					Delegate:     true,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: amount,
					Delegate:     true,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
			},
			expectedSupply: &Supply{
				Total:         amount*3 + 3,
				Staked:        amount*3 + 3,
				DelegatedOnly: amount*3 + 3,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: amount*3 + 3,
					},
					{
						Id:     2,
						Amount: amount*3 + 3,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: amount*3 + 3,
					},
					{
						Id:     2,
						Amount: amount*3 + 3,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validators
			if test.preset != nil {
				// convenience variable for supply
				supply := &Supply{}
				require.NoError(t, sm.SetValidators(test.preset, supply))
				// set the supply
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			got, err := sm.GetValidators()
			require.NoError(t, err)
			// sort got by stake
			slices.SortFunc(got, func(a *Validator, b *Validator) int {
				switch {
				case a.StakedAmount == b.StakedAmount:
					return 0
				case a.StakedAmount < b.StakedAmount:
					return 1
				default:
					return -1
				}
			})
			// compare got vs expected
			for i, v := range got {
				require.EqualExportedValues(t, test.preset[i], v)
			}
			// get the committees from state
			set, err := sm.GetCommitteePaginated(lib.PageParams{}, lib.CanopyChainId)
			require.NoError(t, err)
			// check committees got vs expected
			for i, member := range *set.Results.(*ValidatorPage) {
				require.EqualExportedValues(t, test.preset[i], member)
			}
			// get delegates from state
			set, err = sm.GetDelegatesPaginated(lib.PageParams{}, lib.CanopyChainId)
			require.NoError(t, err)
			// check delegates got vs expected
			for i, member := range *set.Results.(*ValidatorPage) {
				require.EqualExportedValues(t, test.preset[i], member)
			}
			gotSupply, err := sm.GetSupply()
			require.NoError(t, err)
			// compare supply
			require.EqualExportedValues(t, test.expectedSupply, gotSupply)
		})
	}
}

func TestGetValidatorsPaginated(t *testing.T) {
	const amount = uint64(100)
	tests := []struct {
		name            string
		detail          string
		validators      []*Validator
		pageParams      lib.PageParams
		expectedAddress [][]byte
		filters         lib.ValidatorFilters
	}{
		{
			name:       "no validators",
			detail:     "test when there exists no validators in the state",
			validators: nil,
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    100,
			},
		},
		{
			name:   "multi-validator",
			detail: "test with multiple validators and default page params",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
				},
			},
			expectedAddress: [][]byte{
				newTestAddressBytes(t, 1),
				newTestAddressBytes(t, 2),
				newTestAddressBytes(t),
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    100,
			},
		},
		{
			name:   "multi-validator filter paused",
			detail: "test with multiple validators and default page params, filtering paused",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
				},
				{
					Address:         newTestAddressBytes(t, 1),
					StakedAmount:    amount,
					MaxPausedHeight: 1,
				},
				{
					Address:         newTestAddressBytes(t, 2),
					StakedAmount:    amount,
					UnstakingHeight: 1,
				},
			},
			expectedAddress: [][]byte{
				newTestAddressBytes(t, 2),
				newTestAddressBytes(t),
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    100,
			},
			filters: lib.ValidatorFilters{
				Paused: lib.FilterOption_Exclude,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validators
			if test.validators != nil {
				require.NoError(t, sm.SetValidators(test.validators, &Supply{}))
			}
			// execute the function call
			page, err := sm.GetValidatorsPaginated(test.pageParams, test.filters)
			// validate no error
			require.NoError(t, err)
			// check got vs expected page type
			require.Equal(t, ValidatorsPageName, page.Type)
			// check got vs expected page params
			require.EqualExportedValues(t, test.pageParams, page.PageParams)
			// check got vs expected page result
			for i, got := range *page.Results.(*ValidatorPage) {
				require.Equal(t, test.expectedAddress[i], got.Address)
			}
		})
	}
}

func TestUpdateValidatorStake(t *testing.T) {
	const amount = uint64(100)
	tests := []struct {
		name           string
		detail         string
		preset         *Validator
		update         *Validator
		expectedSupply *Supply
	}{
		{
			name:   "no updates",
			detail: "no updates to the validator",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
			},
			update: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
			},
			expectedSupply: &Supply{
				Total:  amount,
				Staked: amount,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: amount,
					},
					{
						Id:     1,
						Amount: amount,
					},
				},
			},
		},
		{
			name:   "stake",
			detail: "update validator stake",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
			},
			update: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount + 1,
				Committees:   []uint64{0, 1},
			},
			expectedSupply: &Supply{
				Total:  amount, // not updated by this function
				Staked: amount + 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     1,
						Amount: amount + 1,
					},
				},
			},
		},
		{
			name:   "delegated stake",
			detail: "update delegate stake",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			update: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount + 1,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total:         amount, // not updated by this function
				Staked:        amount + 1,
				DelegatedOnly: amount + 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     1,
						Amount: amount + 1,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     1,
						Amount: amount + 1,
					},
				},
			},
		},
		{
			name:   "committees",
			detail: "update validator committees",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
			},
			update: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount + 1,
				Committees:   []uint64{0, 2},
			},
			expectedSupply: &Supply{
				Total:  amount, // not updated by this function
				Staked: amount + 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     2,
						Amount: amount + 1,
					},
				},
			},
		},
		{
			name:   "delegated committees",
			detail: "update delegate committees",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			update: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount + 1,
				Committees:   []uint64{0, 2},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total:         amount, // not updated by this function
				Staked:        amount + 1,
				DelegatedOnly: amount + 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     2,
						Amount: amount + 1,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     0,
						Amount: amount + 1,
					},
					{
						Id:     2,
						Amount: amount + 1,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validators
			if test.preset != nil {
				supply := &Supply{}
				// set the validator
				require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
				// update the supply in state
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			require.NoError(t, sm.UpdateValidatorStake(test.preset, test.update.Committees, test.update.StakedAmount-test.preset.StakedAmount))
			// get the validator
			got, err := sm.GetValidator(crypto.NewAddress(test.preset.Address))
			require.NoError(t, err)
			// check got vs expected
			require.EqualExportedValues(t, test.update, got)
			// validate committee membership
			for _, cId := range test.update.Committees {
				var page *lib.Page
				if test.update.Delegate {
					// get the delegates
					page, err = sm.GetDelegatesPaginated(lib.PageParams{}, cId)
				} else {
					// get the committee
					page, err = sm.GetCommitteePaginated(lib.PageParams{}, cId)
				}
				require.NoError(t, err)
				// ensure the slice contains the expected
				var contains bool
				for _, member := range *page.Results.(*ValidatorPage) {
					if bytes.Equal(member.PublicKey, test.update.PublicKey) {
						contains = true
						break
					}
				}
				require.True(t, contains)
			}
			// get the supply
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// validate supply update
			require.EqualExportedValues(t, test.expectedSupply, supply)
		})
	}
}

func TestDeleteValidator(t *testing.T) {
	const amount = uint64(100)
	tests := []struct {
		name           string
		detail         string
		preset         *Validator
		expectedSupply *Supply
	}{
		{
			name:   "delete validator",
			detail: "delete validator with 1 committee",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0},
			},
			expectedSupply: &Supply{
				Total: amount,
			},
		}, {
			name:   "delete validator multi committee",
			detail: "delete validator with multiple committees",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1, 2},
			},
			expectedSupply: &Supply{
				Total: amount,
			},
		},
		{
			name:   "delete delegate",
			detail: "delete delegate with 1 committee",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1, 2},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total: amount,
			},
		},
		{
			name:   "delete delegate multi committee",
			detail: "delete delegate with multiple committees",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: amount,
				Committees:   []uint64{0, 1, 2},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total: amount,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validators
			if test.preset != nil {
				supply := &Supply{}
				// set the validator
				require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
				// update the supply in state
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			require.NoError(t, sm.DeleteValidator(test.preset))
			// get the validator
			_, err := sm.GetValidator(crypto.NewAddress(test.preset.Address))
			require.ErrorContains(t, err, "validator does not exist")
			// validate committee non-membership
			for _, cId := range test.preset.Committees {
				var page *lib.Page
				if test.preset.Delegate {
					// get the delegates
					page, err = sm.GetDelegatesPaginated(lib.PageParams{}, cId)
				} else {
					// get the committee
					page, err = sm.GetCommitteePaginated(lib.PageParams{}, cId)
				}
				require.NoError(t, err)
				// ensure the slice contains the expected
				var contains bool
				for _, member := range *page.Results.(*ValidatorPage) {
					if bytes.Equal(member.PublicKey, test.preset.PublicKey) {
						contains = true
						break
					}
				}
				require.False(t, contains)
			}
			// get the supply
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// validate supply update
			require.EqualExportedValues(t, test.expectedSupply, supply)
		})
	}
}

func TestSetValidatorUnstaking(t *testing.T) {
	tests := []struct {
		name                  string
		detail                string
		preset                *Validator
		finishUnstakingHeight uint64
	}{
		{
			name:   "set unstaking",
			detail: "set a standard validator unstaking",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				Committees:      nil,
				MaxPausedHeight: 0,
			},
			finishUnstakingHeight: 1,
		},
		{
			name:   "set paused unstaking",
			detail: "set a paused validator unstaking",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				Committees:      nil,
				MaxPausedHeight: 1,
			},
			finishUnstakingHeight: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// convenience variable for address
			address := crypto.NewAddress(test.preset.Address)
			// create a test state machine
			sm := newTestStateMachine(t)
			// set the validator
			require.NoError(t, sm.SetValidator(test.preset))
			// execute the function call
			require.NoError(t, sm.SetValidatorUnstaking(address, test.preset, test.finishUnstakingHeight))
			// get the validator
			validator, err := sm.GetValidator(address)
			require.NoError(t, err)
			// ensure validator is unpaused
			require.Zero(t, validator.MaxPausedHeight)
			// ensure validator unstaking height is expected
			require.Equal(t, test.finishUnstakingHeight, validator.UnstakingHeight)
			// ensure unstaking key exists
			list, e := sm.GetAddressList(UnstakingPrefix(test.finishUnstakingHeight))
			require.NoError(t, e)
			require.Contains(t, list.Addresses, address.Bytes())
		})
	}
}

func TestDeleteFinishedUnstaking(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
	}{
		{
			name:   "validator same output/operator",
			detail: "validator with the same output and operator address",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
			},
		},
		{
			name:   "validator different output/operator",
			detail: "validator with the different output and operator address",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
			},
		},
		{
			name:   "delegate same output/operator",
			detail: "delegate with the same output and operator address",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Delegate:     true,
			},
		},
		{
			name:   "delegate different output/operator",
			detail: "delegate with the different output and operator address",
			preset: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Delegate:     true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// convenience variable for address
			address := crypto.NewAddress(test.preset.Address)
			// create a test state machine
			sm := newTestStateMachine(t)
			// convenience variable for supply
			supply := &Supply{}
			// set the validator in state
			require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
			// set the supply in state
			require.NoError(t, sm.SetSupply(supply))
			// set the validator as unstaking
			require.NoError(t, sm.SetValidatorUnstaking(address, test.preset, sm.height))
			// execute the function call
			require.NoError(t, sm.DeleteFinishedUnstaking())
			// get the validator
			_, err := sm.GetValidator(crypto.NewAddress(test.preset.Address))
			// validate the deletion of the validator
			require.ErrorContains(t, err, "validator does not exist")
			// get the output account balance
			balance, err := sm.GetAccountBalance(crypto.NewAddress(test.preset.Output))
			require.NoError(t, err)
			// validate the addition to the account
			require.Equal(t, test.preset.StakedAmount, balance)
			// validate unstaking key exists
			list, e := sm.GetAddressList(UnstakingPrefix(sm.height))
			require.NoError(t, e)
			require.NotContains(t, list.Addresses, address.Bytes())
		})
	}
}

func TestSetValidatorsPaused(t *testing.T) {
	tests := []struct {
		name    string
		detail  string
		preset  []*Validator
		chainId uint64
		toPause [][]byte
	}{
		{
			name:   "single validator pause",
			detail: "single validator pause",
			preset: []*Validator{{
				Address:    newTestAddressBytes(t),
				Committees: []uint64{1},
			}},
			chainId: 1,
			toPause: [][]byte{newTestAddressBytes(t)},
		},
		{
			name:   "unauthorized validator pause",
			detail: "unauthorized validator pause",
			preset: []*Validator{{
				Address:    newTestAddressBytes(t),
				Committees: []uint64{1},
			}},
			chainId: 2,
			toPause: [][]byte{},
		},
		{
			name:   "multi validator pause",
			detail: "multi validator pause",
			preset: []*Validator{{
				Address:    newTestAddressBytes(t),
				Committees: []uint64{1},
			}, {
				Address:    newTestAddressBytes(t, 1),
				Committees: []uint64{1},
			}},
			chainId: 1,
			toPause: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		},
		{
			name:   "mixed authorized multi validator pause",
			detail: "mixed authorized multi validator pause",
			preset: []*Validator{{
				Address:    newTestAddressBytes(t),
				Committees: []uint64{1},
			}, {
				Address:    newTestAddressBytes(t, 1),
				Committees: []uint64{2},
			}},
			chainId: 1,
			toPause: [][]byte{newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the validator
			if test.preset != nil {
				supply := &Supply{}
				require.NoError(t, sm.SetValidators(test.preset, supply))
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			sm.SetValidatorsPaused(test.chainId, test.toPause)
			for _, validator := range test.toPause {
				paused := crypto.NewAddress(validator)
				// validate the unstaking of the validator object
				val, e := sm.GetValidator(paused)
				require.NoError(t, e)
				// get validator params
				valParams, e := sm.GetParamsVal()
				require.NoError(t, e)
				// calculate the finish unstaking height
				maxPauseBlocks := valParams.MaxPauseBlocks + sm.Height()
				// compare got vs expected
				require.Equal(t, maxPauseBlocks, val.MaxPausedHeight)
				// check for the paused key
				list, e := sm.GetAddressList(PausedPrefix(maxPauseBlocks))
				require.NoError(t, e)
				require.Contains(t, list.Addresses, paused.Bytes())
			}
		})
	}
}

func TestSetValidatorPausedAndUnpaused(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		validator      *Validator
		maxPauseHeight uint64
	}{
		{
			name:           "pause height 1",
			detail:         "this function creates a validator object and a key for the validator under the unstaking prefix",
			validator:      &Validator{Address: newTestAddressBytes(t)},
			maxPauseHeight: 1,
		},
		{
			name:           "pause height 100",
			detail:         "this function creates a validator object and a key for the validator under the unstaking prefix",
			validator:      &Validator{Address: newTestAddressBytes(t)},
			maxPauseHeight: 100,
		},
		{
			name:           "pause height max",
			detail:         "this function creates a validator object and a key for the validator under the unstaking prefix",
			validator:      &Validator{Address: newTestAddressBytes(t)},
			maxPauseHeight: math.MaxUint64,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address := crypto.NewAddress(test.validator.Address)
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function 1
			require.NoError(t, sm.SetValidatorPaused(address, test.validator, test.maxPauseHeight))
			// validate the pause of the validator object
			val, e := sm.GetValidator(address)
			require.NoError(t, e)
			// compare got vs expected
			require.Equal(t, test.maxPauseHeight, val.MaxPausedHeight)
			// check for the paused key
			list, e := sm.GetAddressList(PausedPrefix(test.maxPauseHeight))
			require.NoError(t, e)
			require.Contains(t, list.Addresses, address.Bytes())
			// execute the function 2
			require.NoError(t, sm.SetValidatorUnpaused(address, test.validator))
			// validate the un-pause of the validator object
			val, e = sm.GetValidator(address)
			require.NoError(t, e)
			// compare got vs expected
			require.Zero(t, val.MaxPausedHeight)
			// validate no paused key
			list, e = sm.GetAddressList(PausedPrefix(test.maxPauseHeight))
			require.NoError(t, e)
			require.NotContains(t, list.Addresses, address.Bytes())
		})
	}
}

func TestGetAuthorizedSignersForValidator(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		preset   *Validator
		address  []byte
		expected [][]byte
		error    string
	}{
		{
			name:    "validator doesn't exist",
			detail:  "the operation fails because the validator doesn't exist",
			address: newTestAddressBytes(t),
			expected: [][]byte{
				newTestAddressBytes(t),
			},
			error: "validator does not exist",
		}, {
			name:    "custodial",
			detail:  "same output and operator",
			address: newTestAddressBytes(t),
			preset: &Validator{
				Address: newTestAddressBytes(t),
				Output:  newTestAddressBytes(t),
			},
			expected: [][]byte{
				newTestAddressBytes(t),
			},
		},
		{
			name:    "non-custodial",
			detail:  "different output and operator",
			address: newTestAddressBytes(t),
			preset: &Validator{
				Address: newTestAddressBytes(t),
				Output:  newTestAddressBytes(t, 1),
			},
			expected: [][]byte{
				newTestAddressBytes(t), newTestAddressBytes(t, 1),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the validator
			if test.preset != nil {
				require.NoError(t, sm.SetValidator(test.preset))
			}
			// execute the function call
			got, err := sm.GetAuthorizedSignersForValidator(test.address)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// compare got vs expected
			require.Equal(t, test.expected, got)
		})
	}
}
