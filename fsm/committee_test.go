package fsm

import (
	"github.com/ginchuco/canopy/fsm/types"
	"github.com/ginchuco/canopy/lib"
	"github.com/ginchuco/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFundCommitteeRewardPools(t *testing.T) {
	const (
		minPercentForPaidCommittee = 10
	)
	tests := []struct {
		name          string
		detail        string
		mintAmount    uint64
		daoCutPercent uint64
		supply        *types.Supply
		expected      []*types.Pool
	}{
		{
			name:          "1 paid committee",
			detail:        "1 paid committee should result in 1 distribution to the DAO and 1 distribution to the committee",
			mintAmount:    100,
			daoCutPercent: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     lib.CanopyCommitteeId,
						Amount: 10,
					},
				},
			},
			expected: []*types.Pool{
				{
					Id:     lib.CanopyCommitteeId,
					Amount: 90,
				},
				{
					Id:     lib.DAOPoolID,
					Amount: 10,
				},
			},
		},
		{
			name:          "2 paid committees",
			detail:        "2 paid committees should result in 1 distribution to the DAO and 2 distributions to the committees",
			mintAmount:    100,
			daoCutPercent: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     lib.CanopyCommitteeId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyCommitteeId + 1,
						Amount: 10,
					},
				},
			},
			expected: []*types.Pool{
				{
					Id:     lib.CanopyCommitteeId,
					Amount: 45,
				},
				{
					Id:     lib.CanopyCommitteeId + 1,
					Amount: 45,
				},
				{
					Id:     lib.DAOPoolID,
					Amount: 10,
				},
			},
		},
		{
			name:          "4 paid committees with round down",
			detail:        "4 paid committees should result in 1 distribution to the DAO and 4 distributions to the committees (rounded down)",
			mintAmount:    98,
			daoCutPercent: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     lib.CanopyCommitteeId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyCommitteeId + 1,
						Amount: 10,
					},
					{
						Id:     lib.CanopyCommitteeId + 2,
						Amount: 10,
					},
					{
						Id:     lib.CanopyCommitteeId + 3,
						Amount: 10,
					},
				},
			},
			expected: []*types.Pool{
				{
					Id:     lib.CanopyCommitteeId,
					Amount: 22,
				},
				{
					Id:     lib.CanopyCommitteeId + 1,
					Amount: 22,
				},
				{
					Id:     lib.CanopyCommitteeId + 2,
					Amount: 22,
				},
				{
					Id:     lib.CanopyCommitteeId + 3,
					Amount: 22,
				},
				{
					Id:     lib.DAOPoolID,
					Amount: 10,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params
			params, err := sm.GetParams()
			require.NoError(t, err)
			// override the minimum percent for paid committee
			params.Validator.ValidatorMinimumPercentForPaidCommittee = minPercentForPaidCommittee
			// override the mint amount
			params.Validator.ValidatorBlockReward = test.mintAmount
			// override the DAO cut percent
			params.Governance.DaoRewardPercentage = test.daoCutPercent
			// set the params back in state
			require.NoError(t, sm.SetParams(params))
			// get the supply in state
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// set the test supply
			supply.Staked = test.supply.Staked
			supply.CommitteesWithDelegations = test.supply.CommitteesWithDelegations
			// set the supply back in state
			require.NoError(t, sm.SetSupply(supply))
			// execute the function call
			require.NoError(t, sm.FundCommitteeRewardPools())
			// get the supply in state
			afterSupply, err := sm.GetSupply()
			require.NoError(t, err)
			// ensure total supply increased by the expected
			require.Equal(t, test.mintAmount, afterSupply.Total-supply.Total)
			// ensure the pools have the expected value
			for _, expected := range test.expected {
				// get the pool from state
				got, e := sm.GetPool(expected.Id)
				require.NoError(t, e)
				// validate the balance
				require.Equal(t, expected.Amount, got.Amount)
			}
		})
	}
}

func TestGetPaidCommittees(t *testing.T) {
	tests := []struct {
		name                       string
		detail                     string
		minPercentForPaidCommittee uint64
		supply                     *types.Supply
		paidCommitteeIds           []uint64
	}{
		{
			name:                       "0 committees",
			detail:                     "1there exists no committees",
			minPercentForPaidCommittee: 10,
			supply:                     &types.Supply{Staked: 100},
		},
		{
			name:                       "0 paid committee",
			detail:                     "1 committee that has less than the minimum committed to it",
			minPercentForPaidCommittee: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 1,
					},
				},
			},
		},
		{
			name:                       "1 100% paid committee",
			detail:                     "1 paid committee that has 100% of the stake committed to it",
			minPercentForPaidCommittee: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 100,
					},
				},
			},
			paidCommitteeIds: []uint64{0},
		},
		{
			name:                       "1 paid committee, 1 non paid committee",
			detail:                     "1 paid committee that has enough stake to be above the threshold, 1 non paid committee",
			minPercentForPaidCommittee: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 10,
					},
					{
						Id:     1,
						Amount: 1,
					},
				},
			},
			paidCommitteeIds: []uint64{0},
		},
		{
			name:                       "2 100% paid committees",
			detail:                     "2 paid committees that has 100% of the stake committed to it",
			minPercentForPaidCommittee: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 100,
					},
					{
						Id:     1,
						Amount: 100,
					},
				},
			},
			paidCommitteeIds: []uint64{0, 1},
		},
		{
			name:                       "2 10% paid committees",
			detail:                     "2 paid committees that has the exact threshold of the stake committed to it",
			minPercentForPaidCommittee: 10,
			supply: &types.Supply{
				Staked: 100,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 10,
					},
					{
						Id:     1,
						Amount: 10,
					},
				},
			},
			paidCommitteeIds: []uint64{0, 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// override the minimum percent for paid committee
			valParams.ValidatorMinimumPercentForPaidCommittee = test.minPercentForPaidCommittee
			// set the params back in state
			require.NoError(t, sm.SetParamsVal(valParams))
			// get the supply in state
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// set the test supply
			supply.Staked = test.supply.Staked
			supply.CommitteesWithDelegations = test.supply.CommitteesWithDelegations
			// set the supply back in state
			require.NoError(t, sm.SetSupply(supply))
			// execute the function call
			paidCommitteeIds, err := sm.GetPaidCommittees(valParams)
			require.NoError(t, err)
			// ensure expected = got
			require.Equal(t, test.paidCommitteeIds, paidCommitteeIds)
		})
	}
}

func TestGetCommitteeMembers(t *testing.T) {
	stakedAmount := uint64(100)
	tests := []struct {
		name     string
		detail   string
		limit    uint64
		preset   []*types.Validator
		expected map[uint64][][]byte
	}{
		{
			name:   "1 validator 1 committee",
			detail: "1 validator staked for 1 committee",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t),
				},
			},
		},
		{
			name:   "3 validators 1 committee",
			detail: "3 validators staked for 1 committee ordered by stake",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators 2 committees",
			detail: "3 validators staked for 2 committees ordered by stake",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyCommitteeId, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId, 1},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
				1: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},

		{
			name:   "3 validators 2 committees various",
			detail: "3 validators partially staked over 2 committees ordered by stake",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId, 1},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
				1: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, 1 paused, 1 committee",
			detail: "3 validators staked, 1 paused, for 1 committee ordered by stake",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:         newTestAddressBytes(t, 1),
					PublicKey:       newTestPublicKeyBytes(t, 1),
					StakedAmount:    stakedAmount + 2,
					MaxPausedHeight: 1,
					Committees:      []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, 1 unstaking, 1 committee",
			detail: "3 validators staked, 1 unstaking, for 1 committee ordered by stake",
			limit:  10,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:         newTestAddressBytes(t, 1),
					PublicKey:       newTestPublicKeyBytes(t, 1),
					StakedAmount:    stakedAmount + 2,
					UnstakingHeight: 1,
					Committees:      []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, Max 2, 1 committee",
			detail: "3 validators staked, Limit is 2, for 1 committee ordered by stake",
			limit:  2,
			preset: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyCommitteeId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// override the minimum percent for paid committee
			valParams.ValidatorMaxCommitteeSize = test.limit
			// set the params back in state
			require.NoError(t, sm.SetParamsVal(valParams))
			// preset the validators
			for _, v := range test.preset {
				// set validator in the state
				require.NoError(t, sm.SetValidator(v))
				// set committees
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// validate the function
			for id, expected := range test.expected {
				// run the function call
				got, e := sm.GetCommitteeMembers(id)
				require.NoError(t, e)
				// test 'GetCanopyCommitteeMembers'
				if id == lib.CanopyCommitteeId {
					canopyCommittee, er := sm.GetCanopyCommitteeMembers()
					require.NoError(t, er)
					require.Equal(t, got.ValidatorSet.ValidatorSet, canopyCommittee.ValidatorSet)
				}
				// ensure returned validator set is not nil
				require.NotNil(t, got.ValidatorSet)
				// ensure expected and got are the same size
				require.Equal(t, len(expected), len(got.ValidatorSet.ValidatorSet))
				// validate the equality of the sets
				for i, v := range got.ValidatorSet.ValidatorSet {
					require.Equal(t, expected[i], v.PublicKey)
				}
			}
		})
	}
}

func TestGetCommitteePaginated(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		validators []*types.Validator
		pageParams lib.PageParams
		expected   [][]byte // address
	}{
		{
			name:   "page 1 all members",
			detail: "returns the first page with both members (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    2,
			},
			expected: [][]byte{newTestAddressBytes(t, 1), newTestAddressBytes(t)},
		},
		{
			name:   "page 1, 1 member",
			detail: "returns the first page with 1 member (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    1,
			},
			expected: [][]byte{newTestAddressBytes(t, 1)},
		},
		{
			name:   "page 2, 1 member",
			detail: "returns the second page with 1 member (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 2,
				PerPage:    1,
			},
			expected: [][]byte{newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// run the function call
			page, err := sm.GetCommitteePaginated(test.pageParams, lib.CanopyCommitteeId)
			require.NoError(t, err)
			// validate the page params
			require.Equal(t, test.pageParams, page.PageParams)
			// cast page and ensure valid
			got, castOk := page.Results.(*types.ValidatorPage)
			require.True(t, castOk)
			for i, gotItem := range *got {
				require.Equal(t, test.expected[i], gotItem.Address)
			}
		})
	}
}

func TestSetGetCommittees(t *testing.T) {
	tests := []struct {
		name                  string
		detail                string
		validators            []*types.Validator
		expected              map[uint64][][]byte // committeeId -> Public Key
		expectedTotalPower    map[uint64]uint64
		expectedMin23MajPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "preset 1 validator with 1 committee and expect to retrieve that validator",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 100,
			},
			expectedMin23MajPower: map[uint64]uint64{
				0: 67,
			},
		},
		{
			name:   "1 validator 2 committees",
			detail: "preset 1 validator with 2 committees and expect to retrieve that validator",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0, 1},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t)}, 1: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 100, 1: 100,
			},
			expectedMin23MajPower: map[uint64]uint64{
				0: 67, 1: 67,
			},
		},
		{
			name:   "2 validator 2 committees",
			detail: "preset 1 validator with 2 committees and expect to retrieve those validator",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 101,
					Committees:   []uint64{0, 1},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t, 1), newTestPublicKeyBytes(t)}, 1: {newTestPublicKeyBytes(t, 1), newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 201, 1: 201,
			},
			expectedMin23MajPower: map[uint64]uint64{
				0: 135, 1: 135,
			},
		},
		{
			name:   "2 validator mixed committees",
			detail: "preset 1 validator with mixed committees and expect to retrieve those validators",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 101,
					Committees:   []uint64{0, 1},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t, 1), newTestPublicKeyBytes(t)}, 1: {newTestPublicKeyBytes(t, 1)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 201, 1: 101,
			},
			expectedMin23MajPower: map[uint64]uint64{
				0: 135, 1: 68,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				got, err := sm.GetCommitteeMembers(id)
				require.NoError(t, err)
				// get the committee pool from the supply object
				p, err := sm.GetCommitteeStakedSupply(id)
				require.NoError(t, err)
				// compare got total power vs expected total power
				require.Equal(t, test.expectedTotalPower[id], got.TotalPower)
				// compare got supply vs total tokens
				require.Equal(t, test.expectedTotalPower[id], p.Amount)
				// compare got min 2/3 maj vs expected min 2/3 maj
				require.Equal(t, test.expectedMin23MajPower[id], got.MinimumMaj23)
				// compare got num validators vs num validators
				require.EqualValues(t, len(test.expected[id]), got.NumValidators)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
					// compare got vs expected
					require.Equal(t, expectedPublicKey, got.ValidatorSet.ValidatorSet[i].PublicKey)
				}
			}
		})
	}
}

func TestUpdateCommittees(t *testing.T) {
	tests := []struct {
		name               string
		detail             string
		validators         []*types.Validator
		updates            []*types.Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "updating 1 validator and same 1 committee with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 101,
					Committees:   []uint64{0},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 101,
			},
		},
		{
			name:   "1 validator 1 different committee",
			detail: "updating 1 validator and different 1 committee with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 101,
					Committees:   []uint64{1},
				},
			},
			expected: map[uint64][][]byte{
				1: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				1: 101,
			},
		},
		{
			name:   "2 validators different committees",
			detail: "updating 2 validator with different committees with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 101,
				Committees:   []uint64{0},
			}, {
				Address:      newTestAddressBytes(t, 1),
				PublicKey:    newTestPublicKeyBytes(t, 1),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 102,
					Committees:   []uint64{1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 101,
					Committees:   []uint64{1},
				},
			},
			expected: map[uint64][][]byte{
				1: {newTestPublicKeyBytes(t), newTestPublicKeyBytes(t, 1)},
			},
			expectedTotalPower: map[uint64]uint64{
				1: 203,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each update
			for _, v := range test.updates {
				// cast the address bytes to object
				addr := crypto.NewAddress(v.Address)
				// retrieve the old validator
				val, err := sm.GetValidator(addr)
				require.NoError(t, err)
				// run the function
				require.NoError(t, sm.UpdateCommittees(addr, val, v.StakedAmount, v.Committees))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				got, err := sm.GetCommitteeMembers(id)
				require.NoError(t, err)
				// compare got num validators vs num validators
				require.EqualValues(t, len(test.expected[id]), got.NumValidators)
				// get the committee pool from the supply object
				p, err := sm.GetCommitteeStakedSupply(id)
				require.NoError(t, err)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
					// compare got supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], p.Amount)
					// compare got vs expected
					require.Equal(t, expectedPublicKey, got.ValidatorSet.ValidatorSet[i].PublicKey)
				}
			}
		})
	}
}

func TestDeleteCommittees(t *testing.T) {
	tests := []struct {
		name               string
		detail             string
		validators         []*types.Validator
		delete             []*types.Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "2 validator 1 committee, 1 delete",
			detail: "2 validator, deleting 1 validator",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}, {
				Address:      newTestAddressBytes(t, 1),
				PublicKey:    newTestPublicKeyBytes(t, 1),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			delete: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0},
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t, 1)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 100,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each update
			for _, v := range test.delete {
				// run the function
				require.NoError(t, sm.DeleteCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				got, err := sm.GetCommitteeMembers(id)
				require.NoError(t, err)
				// compare got num validators vs num validators
				require.EqualValues(t, len(test.expected[id]), got.NumValidators)
				// get the committee pool from the supply object
				p, err := sm.GetCommitteeStakedSupply(id)
				require.NoError(t, err)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
					// compare got supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], p.Amount)
					// compare got vs expected
					require.Equal(t, expectedPublicKey, got.ValidatorSet.ValidatorSet[i].PublicKey)
				}
			}
		})
	}
}

func TestGetDelegatesPaginated(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		validators []*types.Validator
		pageParams lib.PageParams
		expected   [][]byte // address
	}{
		{
			name:   "page 1 all members",
			detail: "returns the first page with both members (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    2,
			},
			expected: [][]byte{newTestAddressBytes(t, 1), newTestAddressBytes(t)},
		},
		{
			name:   "page 1, 1 member",
			detail: "returns the first page with 1 member (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 1,
				PerPage:    1,
			},
			expected: [][]byte{newTestAddressBytes(t, 1)},
		},
		{
			name:   "page 2, 1 member",
			detail: "returns the second page with 1 member (ordered by stake)",
			validators: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			},
			pageParams: lib.PageParams{
				PageNumber: 2,
				PerPage:    1,
			},
			expected: [][]byte{newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetDelegations(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// run the function call
			page, err := sm.GetDelegatesPaginated(test.pageParams, lib.CanopyCommitteeId)
			require.NoError(t, err)
			// validate the page params
			require.Equal(t, test.pageParams, page.PageParams)
			// cast page and ensure valid
			got, castOk := page.Results.(*types.ValidatorPage)
			require.True(t, castOk)
			for i, gotItem := range *got {
				require.Equal(t, test.expected[i], gotItem.Address)
			}
		})
	}
}

func TestUpdateDelegates(t *testing.T) {
	tests := []struct {
		name               string
		detail             string
		validators         []*types.Validator
		updates            []*types.Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "updating 1 validator and same 1 committee with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 101,
					Committees:   []uint64{0},
					Delegate:     true,
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 101,
			},
		},
		{
			name:   "1 validator 1 different committee",
			detail: "updating 1 validator and different 1 committee with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 101,
					Committees:   []uint64{1},
					Delegate:     true,
				},
			},
			expected: map[uint64][][]byte{
				1: {newTestPublicKeyBytes(t)},
			},
			expectedTotalPower: map[uint64]uint64{
				1: 101,
			},
		},
		{
			name:   "2 validators different committees",
			detail: "updating 2 validator with different committees with more tokens",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 101,
				Committees:   []uint64{0},
				Delegate:     true,
			}, {
				Address:      newTestAddressBytes(t, 1),
				PublicKey:    newTestPublicKeyBytes(t, 1),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			updates: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 102,
					Committees:   []uint64{1},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 101,
					Committees:   []uint64{1},
					Delegate:     true,
				},
			},
			expected: map[uint64][][]byte{
				1: {newTestPublicKeyBytes(t), newTestPublicKeyBytes(t, 1)},
			},
			expectedTotalPower: map[uint64]uint64{
				1: 203,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetDelegations(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each update
			for _, v := range test.updates {
				// cast the address bytes to object
				addr := crypto.NewAddress(v.Address)
				// retrieve the old validator
				val, err := sm.GetValidator(addr)
				require.NoError(t, err)
				// run the function
				require.NoError(t, sm.UpdateDelegations(addr, val, v.StakedAmount, v.Committees))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				page, err := sm.GetDelegatesPaginated(lib.PageParams{}, id)
				require.NoError(t, err)
				// cast page
				got, ok := page.Results.(*types.ValidatorPage)
				require.True(t, ok)
				// get the committee pool from the supply object
				committeePool, err := sm.GetCommitteeStakedSupply(id)
				require.NoError(t, err)
				// get the delegates pool from the supply object
				delegatePool, err := sm.GetDelegateStakedSupply(id)
				require.NoError(t, err)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
					// compare got committee supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], committeePool.Amount)
					// compare got delegate supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], delegatePool.Amount)
					// compare got vs expected
					require.Equal(t, expectedPublicKey, (*got)[i].PublicKey)
				}
			}
		})
	}
}

func TestDeleteDelegates(t *testing.T) {
	tests := []struct {
		name               string
		detail             string
		validators         []*types.Validator
		delete             []*types.Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "2 validator 1 committee, 1 delete",
			detail: "2 validator, deleting 1 validator",
			validators: []*types.Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}, {
				Address:      newTestAddressBytes(t, 1),
				PublicKey:    newTestPublicKeyBytes(t, 1),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			delete: []*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{0},
					Delegate:     true,
				},
			},
			expected: map[uint64][][]byte{
				0: {newTestPublicKeyBytes(t, 1)},
			},
			expectedTotalPower: map[uint64]uint64{
				0: 100,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// for each test validator
			for _, v := range test.validators {
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set the validator committees in state
				require.NoError(t, sm.SetDelegations(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each update
			for _, v := range test.delete {
				// run the function
				require.NoError(t, sm.DeleteDelegations(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				page, err := sm.GetDelegatesPaginated(lib.PageParams{}, id)
				require.NoError(t, err)
				// cast page
				got, ok := page.Results.(*types.ValidatorPage)
				require.True(t, ok)
				// get the committee pool from the supply object
				committeePool, err := sm.GetCommitteeStakedSupply(id)
				// get the committee pool from the supply object
				delegatePool, err := sm.GetDelegateStakedSupply(id)
				require.NoError(t, err)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
					// compare got delegate supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], delegatePool.Amount)
					// compare got committee supply vs total tokens
					require.Equal(t, test.expectedTotalPower[id], committeePool.Amount)
					// compare got vs expected
					require.Equal(t, expectedPublicKey, (*got)[i].PublicKey)
				}
			}
		})
	}
}

func TestUpsertGetCommitteeData(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		upsert   []*types.CommitteeData
		expected []*types.CommitteeData
		error    map[int]lib.ErrorI // error with idx
	}{
		{
			name:   "inserts only",
			detail: "1 insert for 2 different committees i.e. no 'updates'",
			upsert: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					CommitteeId:     2,
					CommitteeHeight: 2,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
			},
			expected: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 1,
				},
				{
					CommitteeId:     2,
					CommitteeHeight: 2,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 1,
				},
			},
		},
		{
			name:   "update",
			detail: "2 'sets' for the same committees i.e. one 'update'",
			upsert: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					CommitteeId:     1,
					CommitteeHeight: 2,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			expected: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 2,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 2,
				},
			},
		},
		{
			name:   "update with chain height error",
			detail: "can't update with a LTE chain height",
			upsert: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					CommitteeId:     1,
					CommitteeHeight: 2,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			error: map[int]lib.ErrorI{1: types.ErrInvalidCertificateResults()},
		},
		{
			name:   "update with committee height error",
			detail: "can't update with a smaller committee height",
			upsert: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					CommitteeId:     1,
					CommitteeHeight: 0,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			error: map[int]lib.ErrorI{1: types.ErrInvalidCertificateResults()},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// 'upsert' the committee data
			for i, upsert := range test.upsert {
				err := sm.UpsertCommitteeData(upsert)
				if test.error != nil {
					require.Equal(t, test.error[i], err)
					return
				}
			}
			// 'get' the expected committee data
			for _, expected := range test.expected {
				got, err := sm.GetCommitteeData(expected.CommitteeId)
				require.NoError(t, err)
				// check committeeId
				require.Equal(t, expected.CommitteeId, got.CommitteeId)
				// check number of samples
				require.Equal(t, expected.NumberOfSamples, got.NumberOfSamples)
				// check chain heights
				require.Equal(t, expected.ChainHeight, got.ChainHeight)
				// check payment percents
				for i, expectedPP := range expected.PaymentPercents {
					require.EqualExportedValues(t, expectedPP, got.PaymentPercents[i])
				}
			}
		})
	}
}

func TestGetSetCommitteesData(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		set    *types.CommitteesData
	}{
		{
			name:   "a single committee",
			detail: "only one committee data inserted",
			set: &types.CommitteesData{List: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 1,
				},
			}},
		},
		{
			name:   "two committee data",
			detail: "two different committee data inserted",
			set: &types.CommitteesData{List: []*types.CommitteeData{
				{
					CommitteeId:     1,
					CommitteeHeight: 1,
					ChainHeight:     1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 1,
				},
				{
					CommitteeId:     0,
					CommitteeHeight: 2,
					ChainHeight:     2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
						},
					},
					NumberOfSamples: 2,
				},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set the committee data
			require.NoError(t, sm.SetCommitteesData(test.set))
			// execute the function call
			got, err := sm.GetCommitteesData()
			require.NoError(t, err)
			// compare got vs expected
			require.EqualExportedValues(t, test.set, got)
		})
	}
}