package fsm

import (
	"fmt"
	"math"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestGetEconomicParameters(t *testing.T) {
	const (
		minPercentForPaidCommittee = 10
	)
	type expected struct {
		daoCut           uint64
		totalMint        uint64
		perCommitteeMint uint64
	}
	tests := []struct {
		name          string
		detail        string
		mintAmount    uint64
		daoCutPercent uint64
		supply        *Supply
		expected      expected
	}{
		{
			name:          "1 paid committee",
			detail:        "1 paid committee should result in 1 distribution to the DAO and 1 distribution to the committee",
			mintAmount:    100,
			daoCutPercent: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
				},
			},
			expected: expected{
				daoCut:           10,
				totalMint:        100,
				perCommitteeMint: 90,
			},
		},
		{
			name:          "2 paid committees",
			detail:        "2 paid committees should result in 1 distribution to the DAO and 2 distributions to the committees",
			mintAmount:    100,
			daoCutPercent: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 1,
						Amount: 10,
					},
				},
			},
			expected: expected{
				daoCut:           10,
				totalMint:        100,
				perCommitteeMint: 45,
			},
		},
		{
			name:          "4 paid committees with round down",
			detail:        "4 paid committees should result in 1 distribution to the DAO and 4 distributions to the committees (rounded down)",
			mintAmount:    98,
			daoCutPercent: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 1,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 2,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 3,
						Amount: 10,
					},
				},
			},
			expected: expected{
				daoCut:           10,
				totalMint:        98,
				perCommitteeMint: 22,
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
			params.Validator.StakePercentForSubsidizedCommittee = minPercentForPaidCommittee
			// override the mint amount
			sm.Config.StateMachineConfig.InitialTokensPerBlock = test.mintAmount
			// override the DAO cut percent
			params.Governance.DaoRewardPercentage = test.daoCutPercent
			// set the params back in state
			require.NoError(t, sm.SetParams(params))
			// get the supply in state
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// set the test supply
			supply.Staked = test.supply.Staked
			supply.CommitteeStaked = test.supply.CommitteeStaked
			// set the supply back in state
			require.NoError(t, sm.SetSupply(supply))
			// execute the function call
			daoCut, totalMint, perCommitteeMint, err := sm.GetBlockMintStats(lib.CanopyChainId)
			require.NoError(t, err)
			require.EqualValues(t, test.expected, expected{
				daoCut:           daoCut,
				totalMint:        totalMint,
				perCommitteeMint: perCommitteeMint,
			})
		})
	}
}

func TestFundCommitteeRewardPools(t *testing.T) {
	const (
		minPercentForPaidCommittee = 10
	)
	tests := []struct {
		name          string
		detail        string
		mintAmount    uint64
		daoCutPercent uint64
		supply        *Supply
		expected      []*Pool
	}{
		{
			name:          "1 paid committee",
			detail:        "1 paid committee should result in 1 distribution to the DAO and 1 distribution to the committee",
			mintAmount:    100,
			daoCutPercent: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
				},
			},
			expected: []*Pool{
				{
					Id:     lib.CanopyChainId,
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
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 1,
						Amount: 10,
					},
				},
			},
			expected: []*Pool{
				{
					Id:     lib.CanopyChainId,
					Amount: 45,
				},
				{
					Id:     lib.CanopyChainId + 1,
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
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     lib.CanopyChainId,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 1,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 2,
						Amount: 10,
					},
					{
						Id:     lib.CanopyChainId + 3,
						Amount: 10,
					},
				},
			},
			expected: []*Pool{
				{
					Id:     lib.CanopyChainId,
					Amount: 22,
				},
				{
					Id:     lib.CanopyChainId + 1,
					Amount: 22,
				},
				{
					Id:     lib.CanopyChainId + 2,
					Amount: 22,
				},
				{
					Id:     lib.CanopyChainId + 3,
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
			params.Validator.StakePercentForSubsidizedCommittee = minPercentForPaidCommittee
			// override the mint amount
			sm.Config.StateMachineConfig.InitialTokensPerBlock = test.mintAmount
			// override the DAO cut percent
			params.Governance.DaoRewardPercentage = test.daoCutPercent
			// set the params back in state
			require.NoError(t, sm.SetParams(params))
			// get the supply in state
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// set the test supply
			supply.Staked = test.supply.Staked
			supply.CommitteeStaked = test.supply.CommitteeStaked
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
		supply                     *Supply
		paidChainIds               []uint64
	}{
		{
			name:                       "0 committees",
			detail:                     "1there exists no committees",
			minPercentForPaidCommittee: 10,
			supply:                     &Supply{Staked: 100},
		},
		{
			name:                       "0 paid committee",
			detail:                     "1 committee that has less than the minimum committed to it",
			minPercentForPaidCommittee: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
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
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: 100,
					},
				},
			},
			paidChainIds: []uint64{0},
		},
		{
			name:                       "1 paid committee, 1 non paid committee",
			detail:                     "1 paid committee that has enough stake to be above the threshold, 1 non paid committee",
			minPercentForPaidCommittee: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
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
			paidChainIds: []uint64{0},
		},
		{
			name:                       "2 100% paid committees",
			detail:                     "2 paid committees that has 100% of the stake committed to it",
			minPercentForPaidCommittee: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
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
			paidChainIds: []uint64{0, 1},
		},
		{
			name:                       "2 10% paid committees",
			detail:                     "2 paid committees that has the exact threshold of the stake committed to it",
			minPercentForPaidCommittee: 10,
			supply: &Supply{
				Staked: 100,
				CommitteeStaked: []*Pool{
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
			paidChainIds: []uint64{0, 1},
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
			valParams.StakePercentForSubsidizedCommittee = test.minPercentForPaidCommittee
			// set the params back in state
			require.NoError(t, sm.SetParamsVal(valParams))
			// get the supply in state
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// set the test supply
			supply.Staked = test.supply.Staked
			supply.CommitteeStaked = test.supply.CommitteeStaked
			// set the supply back in state
			require.NoError(t, sm.SetSupply(supply))
			// execute the function call
			paidChainIds, err := sm.GetSubsidizedCommittees()
			require.NoError(t, err)
			// ensure expected = got
			require.Equal(t, test.paidChainIds, paidChainIds)
		})
	}
}

func TestGetCommitteeMembers(t *testing.T) {
	stakedAmount := uint64(100)
	tests := []struct {
		name     string
		detail   string
		limit    uint64
		preset   []*Validator
		expected map[uint64][][]byte
	}{
		{
			name:   "1 validator 1 committee",
			detail: "1 validator staked for 1 committee",
			limit:  10,
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
					newTestPublicKeyBytes(t),
				},
			},
		},
		{
			name:   "3 validators 1 committee",
			detail: "3 validators staked for 1 committee ordered by stake",
			limit:  10,
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
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
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
				2: {
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
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId, 2},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
					newTestPublicKeyBytes(t, 1),
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
				2: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, 1 paused, 1 committee",
			detail: "3 validators staked, 1 paused, for 1 committee ordered by stake",
			limit:  10,
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:         newTestAddressBytes(t, 1),
					PublicKey:       newTestPublicKeyBytes(t, 1),
					StakedAmount:    stakedAmount + 2,
					MaxPausedHeight: 1,
					Committees:      []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, 1 unstaking, 1 committee",
			detail: "3 validators staked, 1 unstaking, for 1 committee ordered by stake",
			limit:  10,
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:         newTestAddressBytes(t, 1),
					PublicKey:       newTestPublicKeyBytes(t, 1),
					StakedAmount:    stakedAmount + 2,
					UnstakingHeight: 1,
					Committees:      []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
					newTestPublicKeyBytes(t, 2),
					newTestPublicKeyBytes(t, 0),
				},
			},
		},
		{
			name:   "3 validators, Max 2, 1 committee",
			detail: "3 validators staked, Limit is 2, for 1 committee ordered by stake",
			limit:  2,
			preset: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: stakedAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: stakedAmount + 2,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: stakedAmount + 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			expected: map[uint64][][]byte{
				lib.CanopyChainId: {
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
			valParams.MaxCommitteeSize = test.limit
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
		validators []*Validator
		pageParams lib.PageParams
		expected   [][]byte // address
	}{
		{
			name:   "page 1 all members",
			detail: "returns the first page with both members (ordered by stake)",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			page, err := sm.GetCommitteePaginated(test.pageParams, lib.CanopyChainId)
			require.NoError(t, err)
			// validate the page params
			require.Equal(t, test.pageParams, page.PageParams)
			// cast page and ensure valid
			got, castOk := page.Results.(*ValidatorPage)
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
		validators            []*Validator
		expected              map[uint64][][]byte // chainId -> Public Key
		expectedTotalPower    map[uint64]uint64
		expectedMin23MajPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "preset 1 validator with 1 committee and expect to retrieve that validator",
			validators: []*Validator{
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
			validators: []*Validator{
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
			validators: []*Validator{
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
			validators: []*Validator{
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
				p, err := sm.GetCommitteeStakedSupplyForChain(id)
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
		validators         []*Validator
		updates            []*Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "updating 1 validator and same 1 committee with more tokens",
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			updates: []*Validator{
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
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
			}},
			updates: []*Validator{
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
			validators: []*Validator{{
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
			updates: []*Validator{
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
				// update the validator committeee
				require.NoError(t, sm.SetValidator(v))
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
				p, err := sm.GetCommitteeStakedSupplyForChain(id)
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
		validators         []*Validator
		delete             []*Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "2 validator 1 committee, 1 delete",
			detail: "2 validator, deleting 1 validator",
			validators: []*Validator{{
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
			delete: []*Validator{
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
			// create a supply tracker
			supply := &Supply{}
			// set the validators with proper supply tracking
			require.NoError(t, sm.SetValidators(test.validators, supply))
			// update the supply in state
			require.NoError(t, sm.SetSupply(supply))
			// for each validator to delete
			for _, v := range test.delete {
				// DeleteValidator internally calls DeleteCommittees, so we don't need to call it explicitly
				require.NoError(t, sm.DeleteValidator(v))
			}
			// for each expected committee
			for id, publicKeys := range test.expected {
				// execute 'get' function call
				got, err := sm.GetCommitteeMembers(id)
				require.NoError(t, err)
				// compare got num validators vs num validators
				require.EqualValues(t, len(test.expected[id]), got.NumValidators)
				// get the committee pool from the supply object
				p, err := sm.GetCommitteeStakedSupplyForChain(id)
				require.NoError(t, err)
				// compare got supply vs total tokens
				require.Equal(t, test.expectedTotalPower[id], p.Amount)
				// for each expected public key
				for i, expectedPublicKey := range publicKeys {
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
		validators []*Validator
		pageParams lib.PageParams
		expected   [][]byte // address
	}{
		{
			name:   "page 1 all members",
			detail: "returns the first page with both members (ordered by stake)",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 2,
					Committees:   []uint64{lib.CanopyChainId},
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
			page, err := sm.GetDelegatesPaginated(test.pageParams, lib.CanopyChainId)
			require.NoError(t, err)
			// validate the page params
			require.Equal(t, test.pageParams, page.PageParams)
			// cast page and ensure valid
			got, castOk := page.Results.(*ValidatorPage)
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
		validators         []*Validator
		updates            []*Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "1 validator 1 committee",
			detail: "updating 1 validator and same 1 committee with more tokens",
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			updates: []*Validator{
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
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Committees:   []uint64{0},
				Delegate:     true,
			}},
			updates: []*Validator{
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
			validators: []*Validator{{
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
			updates: []*Validator{
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
				got, ok := page.Results.(*ValidatorPage)
				require.True(t, ok)
				// get the committee pool from the supply object
				committeePool, err := sm.GetCommitteeStakedSupplyForChain(id)
				require.NoError(t, err)
				// get the delegates pool from the supply object
				delegatePool, err := sm.GetDelegateStakedSupplyForChain(id)
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
		validators         []*Validator
		delete             []*Validator
		expected           map[uint64][][]byte
		expectedTotalPower map[uint64]uint64
	}{
		{
			name:   "2 validator 1 committee, 1 delete",
			detail: "2 validator, deleting 1 validator",
			validators: []*Validator{{
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
			delete: []*Validator{
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
				got, ok := page.Results.(*ValidatorPage)
				require.True(t, ok)
				// get the committee pool from the supply object
				committeePool, err := sm.GetCommitteeStakedSupplyForChain(id)
				// get the committee pool from the supply object
				delegatePool, err := sm.GetDelegateStakedSupplyForChain(id)
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
		upsert   []*lib.CommitteeData
		expected []*lib.CommitteeData
		error    map[int]lib.ErrorI // error with idx
	}{
		{
			name:   "inserts only",
			detail: "1 insert for 2 different committees i.e. no 'updates'",
			upsert: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							ChainId: 1,
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					ChainId:                2,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							ChainId: 1,
							Percent: 2,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
			},
			expected: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							ChainId: 1,
							Percent: 1,
						},
					},
					NumberOfSamples: 1,
				},
				{
					ChainId:                2,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							ChainId: 1,
							Percent: 2,
						},
					},
					NumberOfSamples: 1,
				},
			},
		},
		{
			name:   "inserts but ignore 1 payment percent",
			detail: "2 inserts but only 1 payment percent is used due to chainId",
			upsert: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							ChainId: 1,
							Percent: 1,
						},
						{
							Address: newTestAddressBytes(t),
							ChainId: 2,
							Percent: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					ChainId:                1,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							ChainId: 2,
							Percent: 2,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
			},
			expected: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							ChainId: 1,
							Percent: 1,
						},
					},
					NumberOfSamples: 2,
				},
			},
		},
		{
			name:   "update",
			detail: "2 'sets' for the same committees i.e. one 'update'",
			upsert: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
							ChainId: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					ChainId:                1,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
							ChainId: 1,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			expected: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
							ChainId: 1,
						},
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
							ChainId: 1,
						},
					},
					NumberOfSamples: 2,
				},
			},
		},
		{
			name:   "update with chain height error",
			detail: "can't update with a LTE chain height",
			upsert: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
							ChainId: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					ChainId:                1,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
							ChainId: 1,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			error: map[int]lib.ErrorI{1: ErrInvalidCertificateResults()},
		},
		{
			name:   "update with committee height error",
			detail: "can't update with a smaller committee height",
			upsert: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
							ChainId: 1,
						},
					},
					NumberOfSamples: 2, // can't overwrite number of samples
				},
				{
					ChainId:                1,
					LastRootHeightUpdated:  0,
					LastChainHeightUpdated: 2,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t, 1),
							Percent: 2,
							ChainId: 1,
						},
					},
					NumberOfSamples: 3, // can't overwrite number of samples
				},
			},
			error: map[int]lib.ErrorI{1: ErrInvalidCertificateResults()},
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
				got, err := sm.GetCommitteeData(expected.ChainId)
				require.NoError(t, err)
				// check chainId
				require.Equal(t, expected.ChainId, got.ChainId)
				// check number of samples
				require.Equal(t, expected.NumberOfSamples, got.NumberOfSamples)
				// check chain heights
				require.Equal(t, expected.LastChainHeightUpdated, got.LastChainHeightUpdated)
				// check payment percents length
				require.Equal(t, len(expected.PaymentPercents), len(got.PaymentPercents), fmt.Sprintf("%v, %v", expected.PaymentPercents, got.PaymentPercents))
				// check actualy payment percents
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
		set    *lib.CommitteesData
	}{
		{
			name:   "a single committee",
			detail: "only one committee data inserted",
			set: &lib.CommitteesData{List: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
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
			set: &lib.CommitteesData{List: []*lib.CommitteeData{
				{
					ChainId:                1,
					LastRootHeightUpdated:  1,
					LastChainHeightUpdated: 1,
					PaymentPercents: []*lib.PaymentPercents{
						{
							Address: newTestAddressBytes(t),
							Percent: 1,
						},
					},
					NumberOfSamples: 1,
				},
				{
					ChainId:                0,
					LastRootHeightUpdated:  2,
					LastChainHeightUpdated: 2,
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

func TestHalvening(t *testing.T) {
	tests := []struct {
		name                  string
		height                uint64
		initialTokensPerBlock uint64
		blocksPerHalvening    uint64
		expected              uint64
	}{
		{
			name:                  "no halvenings",
			height:                0,
			blocksPerHalvening:    210000,
			initialTokensPerBlock: 50000000,
			expected:              50000000,
		},
		{
			name:                  "1 halvening",
			height:                1 * 210000,
			blocksPerHalvening:    210000,
			initialTokensPerBlock: 50000000,
			expected:              25000000,
		},
		{
			name:                  "2 halvening",
			height:                2 * 210000,
			blocksPerHalvening:    210000,
			initialTokensPerBlock: 50000000,
			expected:              12500000,
		},
		{
			name:                  "max halvenings",
			height:                32 * 210000,
			blocksPerHalvening:    210000,
			initialTokensPerBlock: 50000000,
			expected:              0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// calculate the number of halvenings
			halvenings := test.height / test.blocksPerHalvening
			// each halving, the reward is divided by 2
			got := uint64(float64(test.initialTokensPerBlock) / (math.Pow(2, float64(halvenings))))
			// compare got vs expected
			require.Equal(t, test.expected, got)
		})
	}
}

func TestRetireCommittee(t *testing.T) {
	tests := []struct {
		name                      string
		detail                    string
		expectedRetiredCommittees []uint64
	}{
		{
			name:                      "one",
			detail:                    "one retired committees",
			expectedRetiredCommittees: []uint64{1},
		},
		{
			name:                      "multi",
			detail:                    "multiple retired committees",
			expectedRetiredCommittees: []uint64{1, 2, 3},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// retire the committee
			require.NoError(t, sm.SetRetiredCommittees(test.expectedRetiredCommittees))
			// ensure they're retired
			for _, id := range test.expectedRetiredCommittees {
				isRetired, err := sm.CommitteeIsRetired(id)
				require.NoError(t, err)
				require.True(t, isRetired)
			}
			gotCommittees, err := sm.GetRetiredCommittees()
			require.NoError(t, err)
			require.Equal(t, test.expectedRetiredCommittees, gotCommittees)
		})
	}
}

func TestGetTopDelegates(t *testing.T) {
	chainId := uint64(1)
	tests := []struct {
		name                         string
		detail                       string
		maximumDelegatesPerCommittee uint64
		validators                   []*Validator
		expectedDelegates            int
		expectedTotalPower           uint64
		mustInclude                  [][]byte // public keys that must be included
		mustExclude                  [][]byte // public keys that must be excluded
	}{
		{
			name:                         "fewer delegates than maximum",
			detail:                       "when there are fewer delegates than MaximumDelegatesPerCommittee, return all of them",
			maximumDelegatesPerCommittee: 5,
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t, 0),
					PublicKey:    newTestPublicKeyBytes(t, 0),
					StakedAmount: 100,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 200,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: 150,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
			},
			expectedDelegates:  3,
			expectedTotalPower: 450,
			mustInclude: [][]byte{
				newTestPublicKeyBytes(t, 0),
				newTestPublicKeyBytes(t, 1),
				newTestPublicKeyBytes(t, 2),
			},
		},
		{
			name:                         "exact number of delegates",
			detail:                       "when delegates equal MaximumDelegatesPerCommittee, return all of them",
			maximumDelegatesPerCommittee: 3,
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t, 0),
					PublicKey:    newTestPublicKeyBytes(t, 0),
					StakedAmount: 100,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 200,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: 150,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
			},
			expectedDelegates:  3,
			expectedTotalPower: 450,
			mustInclude: [][]byte{
				newTestPublicKeyBytes(t, 0),
				newTestPublicKeyBytes(t, 1),
				newTestPublicKeyBytes(t, 2),
			},
		},
		{
			name:                         "paused and unstaking validators excluded",
			detail:                       "paused and unstaking validators should not be included in top delegates",
			maximumDelegatesPerCommittee: 3,
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t, 0),
					PublicKey:    newTestPublicKeyBytes(t, 0),
					StakedAmount: 500,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:         newTestAddressBytes(t, 1),
					PublicKey:       newTestPublicKeyBytes(t, 1),
					StakedAmount:    400,
					MaxPausedHeight: 1, // paused
					Committees:      []uint64{chainId},
					Delegate:        true,
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: 300,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:         newTestAddressBytes(t, 3),
					PublicKey:       newTestPublicKeyBytes(t, 3),
					StakedAmount:    250,
					UnstakingHeight: 1, // unstaking
					Committees:      []uint64{chainId},
					Delegate:        true,
				},
				{
					Address:      newTestAddressBytes(t, 4),
					PublicKey:    newTestPublicKeyBytes(t, 4),
					StakedAmount: 200,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
			},
			expectedDelegates:  3,
			expectedTotalPower: 1000,
			mustInclude: [][]byte{
				newTestPublicKeyBytes(t, 0), // 500
				newTestPublicKeyBytes(t, 2), // 300
				newTestPublicKeyBytes(t, 4), // 200
			},
			mustExclude: [][]byte{
				newTestPublicKeyBytes(t, 1), // paused
				newTestPublicKeyBytes(t, 3), // unstaking
			},
		},
		{
			name:                         "zero maximum returns all",
			detail:                       "when MaximumDelegatesPerCommittee is 0, return all delegates",
			maximumDelegatesPerCommittee: 0,
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t, 0),
					PublicKey:    newTestPublicKeyBytes(t, 0),
					StakedAmount: 100,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					PublicKey:    newTestPublicKeyBytes(t, 1),
					StakedAmount: 200,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
				{
					Address:      newTestAddressBytes(t, 2),
					PublicKey:    newTestPublicKeyBytes(t, 2),
					StakedAmount: 150,
					Committees:   []uint64{chainId},
					Delegate:     true,
				},
			},
			expectedDelegates:  3,
			expectedTotalPower: 450,
			mustInclude: [][]byte{
				newTestPublicKeyBytes(t, 0),
				newTestPublicKeyBytes(t, 1),
				newTestPublicKeyBytes(t, 2),
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
			// set the maximum delegates per committee
			valParams.MaximumDelegatesPerCommittee = test.maximumDelegatesPerCommittee
			// set the params back in state
			require.NoError(t, sm.SetParamsVal(valParams))
			// preset the validators
			for _, v := range test.validators {
				// set validator in the state
				require.NoError(t, sm.SetValidator(v))
				// set delegations
				require.NoError(t, sm.SetDelegations(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// execute the function call
			got, err := sm.GetDelegates(chainId)
			require.NoError(t, err)
			// validate the returned validator set is not nil
			require.NotNil(t, got.ValidatorSet)
			// validate the number of delegates
			require.Equal(t, len(got.ValidatorSet.ValidatorSet), test.expectedDelegates)
			// validate total power
			require.Equal(t, test.expectedTotalPower, got.TotalPower)
			// validate num validators
			require.Equal(t, uint64(len(got.ValidatorSet.ValidatorSet)), got.NumValidators)
			// validate minimum 2/3 majority
			require.Equal(t, (2*got.TotalPower)/3+1, got.MinimumMaj23)

			// collect all returned public keys for validation
			returnedKeys := make(map[string]bool)
			for _, v := range got.ValidatorSet.ValidatorSet {
				returnedKeys[string(v.PublicKey)] = true
			}

			// ensure all mustInclude keys are present
			for _, pubKey := range test.mustInclude {
				require.True(t, returnedKeys[string(pubKey)], "expected public key %x to be included", pubKey)
			}

			// ensure all mustExclude keys are not present
			for _, pubKey := range test.mustExclude {
				require.False(t, returnedKeys[string(pubKey)], "expected public key %x to be excluded", pubKey)
			}
		})
	}
}
