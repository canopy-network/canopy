package fsm

import (
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestBeginBlock(t *testing.T) {
	const stakeAmount = uint64(100)
	tests := []struct {
		name               string
		detail             string
		isGenesis          bool
		protocolVersion    uint64
		setLastCertResults bool
		error              lib.ErrorI
	}{
		{
			name:            "begin_block at genesis",
			detail:          "genesis skips begin block logic",
			protocolVersion: 1,
			isGenesis:       true,
		},
		{
			name:            "begin_block at genesis with invalid protocol version",
			detail:          "genesis skips begin block logic so invalid protocol version does not error",
			protocolVersion: 0,
			isGenesis:       true,
		},
		{
			name:            "begin_block empty certificate results",
			detail:          "the certificate results are empty",
			protocolVersion: 1,
			error:           lib.ErrNilCertResults(),
		},
		{
			name:               "begin_block after genesis",
			detail:             "after genesis with a valid protocol version",
			protocolVersion:    1,
			setLastCertResults: true,
		},
		{
			name:            "begin_block after genesis with invalid protocol version",
			detail:          "after genesis with an invalid protocol version will return error",
			protocolVersion: 0,
			error:           ErrInvalidProtocolVersion(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				expectedCommitteeMint, expectedDAOMint uint64
			)
			// create a state machine instance with default parameters
			sm := newSingleAccountStateMachine(t)
			// set the last certificate results in the indexer
			if test.setLastCertResults {
				qc := &lib.QuorumCertificate{
					Header: &lib.View{Height: sm.height},
					Results: &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{
						PaymentPercents: []*lib.PaymentPercents{{
							Address: newTestAddressBytes(t),
							Percent: 100,
						}},
					}},
				}
				// track the supply
				supply := &Supply{}
				// for 4 validators
				for i := 0; i < 4; i++ {
					// set the validator
					require.NoError(t, sm.SetValidators([]*Validator{{
						Address:      newTestAddressBytes(t, i),
						PublicKey:    newTestPublicKeyBytes(t, i),
						StakedAmount: 100,
						Committees:   []uint64{lib.CanopyChainId},
					}}, supply))
					// set the committee member
					require.NoError(t, sm.SetCommitteeMember(newTestAddress(t, i), lib.CanopyChainId, 100))
				}
				// set the supply in state
				require.NoError(t, sm.SetSupply(supply))
				// create an aggregate signature
				// get the committee members
				committee, err := sm.GetCommitteeMembers(lib.CanopyChainId)
				require.NoError(t, err)
				// create a copy of the multikey
				mk := committee.MultiKey.Copy()
				// only sign with 3/4 to test the non-signer reduction
				for i := 0; i < 3; i++ {
					privateKey := newTestKeyGroup(t, i).PrivateKey
					// search for the proper index for the signer
					for j, pubKey := range mk.PublicKeys() {
						// if found, add the signer
						if privateKey.PublicKey().Equals(pubKey) {
							// sign the qc
							require.NoError(t, mk.AddSigner(privateKey.Sign(qc.SignBytes()), j))
						}
					}
				}
				// aggregate the signature
				aggSig, e := mk.AggregateSignatures()
				require.NoError(t, e)
				// attach the signature to the message
				qc.Signature = &lib.AggregateSignature{
					Signature: aggSig,
					Bitmap:    mk.Bitmap(),
				}
				require.NoError(t, sm.store.(lib.StoreI).IndexQC(qc))
			}
			// commit the store
			_, err := sm.store.(lib.StoreI).Commit()
			require.NoError(t, err)
			sm.height += 1
			// set protocol version
			sm.ProtocolVersion = test.protocolVersion
			// if at genesis, set height to 1
			if test.isGenesis {
				sm.height = 1
			}
			// ensure expected error on function call
			require.Equal(t, test.error, sm.BeginBlock())
			if test.error != nil {
				return
			}
			// check committee reward
			if !test.isGenesis {
				expectedCommitteeMint, expectedDAOMint = sm.calculateRewardPerCommittee(t, 1)
			}
			// check canopy reward pool for proper mint
			canopyRewardPool, err := sm.GetPool(lib.CanopyChainId)
			require.NoError(t, err)
			require.Equal(t, expectedCommitteeMint, canopyRewardPool.Amount, fmt.Sprintf("%d, %d", expectedCommitteeMint, canopyRewardPool.Amount))
			// check DAO reward pool for proper mint
			daoRewardPool, err := sm.GetPool(lib.DAOPoolID)
			require.NoError(t, err)
			require.Equal(t, expectedDAOMint, daoRewardPool.Amount)
		})
	}
}

func TestEndBlock(t *testing.T) {
	// generate committee data for testing
	committeeData := []*lib.CommitteeData{
		{
			ChainId:                lib.CanopyChainId,
			LastChainHeightUpdated: 1,
			LastRootHeightUpdated:  2,
			PaymentPercents: []*lib.PaymentPercents{
				{Address: newTestAddressBytes(t, 1), ChainId: lib.CanopyChainId, Percent: 100},
			},
		},
		{
			ChainId:                lib.CanopyChainId,
			LastChainHeightUpdated: 2,
			LastRootHeightUpdated:  2,
			PaymentPercents: []*lib.PaymentPercents{
				{Address: newTestAddressBytes(t, 2), ChainId: lib.CanopyChainId, Percent: 100},
			},
		},
		{
			ChainId:                lib.CanopyChainId,
			LastChainHeightUpdated: 3,
			LastRootHeightUpdated:  2,
			PaymentPercents: []*lib.PaymentPercents{
				{Address: newTestAddressBytes(t, 3), ChainId: lib.CanopyChainId, Percent: 100},
			},
		},
	}
	// create test cases
	tests := []struct {
		name                  string
		detail                string
		height                uint64
		previousProposers     [][]byte
		committeeRewardAmount uint64
		committeeData         []*lib.CommitteeData
		validators            []*Validator
		error                 lib.ErrorI
	}{
		{
			name:              "genesis",
			detail:            "no previous proposers, no committee reward/data, no max paused validators, no unstaking validators",
			previousProposers: [][]byte{{}, {}, {}, {}, {}},
		},
		{
			name:   "after genesis, with previous proposers",
			detail: "with previous proposers, no committee reward/data, no max paused validators, no unstaking validators",
			previousProposers: [][]byte{
				newTestAddressBytes(t, 1),
				newTestAddressBytes(t, 2),
				newTestAddressBytes(t, 3),
				newTestAddressBytes(t, 4),
				newTestAddressBytes(t, 5),
			},
		},
		{
			name:                  "after genesis, with committee reward and data",
			detail:                "no previous proposers, with committee reward/data, no max paused validators, no unstaking validators",
			committeeRewardAmount: 100,
			committeeData:         committeeData,
			previousProposers:     [][]byte{{}, {}, {}, {}, {}},
		},
		{
			name:              "after genesis, with committee data and NO reward",
			detail:            "no previous proposers, with committee data and NO reward, no max paused validators, no unstaking validators",
			committeeData:     committeeData,
			previousProposers: [][]byte{{}, {}, {}, {}, {}},
		},
		{
			name:              "after genesis, with a max paused validator",
			detail:            "no previous proposers, no committee data/reward, one max paused validators, no unstaking validators",
			committeeData:     committeeData,
			height:            1,
			previousProposers: [][]byte{{}, {}, {}, {}, {}},
			validators: []*Validator{{
				Address:         newTestAddressBytes(t),
				NetAddress:      "http://localhost:8081",
				StakedAmount:    100,
				MaxPausedHeight: 1,
				Output:          newTestAddressBytes(t),
			}},
		},
		{
			name:              "after genesis, with an unstaking validator",
			detail:            "no previous proposers, no committee data/reward, no max paused validators, one unstaking validators",
			committeeData:     committeeData,
			height:            1,
			previousProposers: [][]byte{{}, {}, {}, {}, {}},
			validators: []*Validator{{
				Address:         newTestAddressBytes(t),
				NetAddress:      "http://localhost:8081",
				StakedAmount:    100,
				UnstakingHeight: 1,
				Output:          newTestAddressBytes(t),
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a test proposer address
			proposerAddress := newTestAddress(t).Bytes()
			// create a state machine instance with default parameters
			sm := newSingleAccountStateMachine(t)

			// STEP 0) inject the test data into state
			func() {
				// set height
				sm.height = test.height
				// set last proposers
				require.NoError(t, sm.SetLastProposers(&lib.Proposers{
					Addresses: test.previousProposers,
				}))
				// set the committee reward
				require.NoError(t, sm.MintToPool(lib.CanopyChainId, test.committeeRewardAmount))
				// set the committee data
				for _, d := range test.committeeData {
					require.NoError(t, sm.UpsertCommitteeData(d))
				}
				// set the validators
				for _, v := range test.validators {
					if v.MaxPausedHeight != 0 {
						require.NoError(t, sm.SetValidatorPaused(crypto.NewAddress(v.Address), v, v.MaxPausedHeight))
					}
					if v.UnstakingHeight != 0 {
						require.NoError(t, sm.SetValidatorUnstaking(crypto.NewAddress(v.Address), v, v.UnstakingHeight))
					}
				}
			}()

			// STEP 1) run function call and check for expected error
			func() { require.Equal(t, test.error, sm.EndBlock(proposerAddress)) }()

			// STEP 2) validate the update of addresses who proposed the block
			func() {
				// generate expected proposers
				test.previousProposers[test.height%5] = proposerAddress
				// retrieve actual proposers
				lastProposers, err := sm.GetLastProposers()
				require.NoError(t, err)
				// validate equality between the expected and got
				require.Equal(t, test.previousProposers, lastProposers.Addresses)
			}()

			// STEP 3) validate the distribution of the committee rewards based on the various Committee Data
			func() {
				for _, d := range test.committeeData {
					for _, paymentPercents := range d.PaymentPercents {
						valParams, err := sm.GetParamsVal()
						require.NoError(t, err)
						// get the account that should have been minted to
						account, err := sm.GetAccount(crypto.NewAddress(paymentPercents.Address))
						// full_reward = ROUND_DOWN( percentage / number_of_samples * available_reward )
						fullReward := uint64(float64(paymentPercents.Percent) / float64(len(committeeData)*100) * float64(test.committeeRewardAmount))
						// if not compounding, use the early withdrawal reward
						earlyWithdrawalReward := lib.Uint64ReducePercentage(fullReward, valParams.EarlyWithdrawalPenalty)
						require.NoError(t, err)
						// compare got and expected
						require.Equal(t, earlyWithdrawalReward, account.Amount)
					}
				}
			}()

			// STEP 4) validate the force unstaking of validators who have been paused for MaxPauseBlocks
			func() {
				// if testing with validators with a max pause height
				if len(test.validators) == 0 || test.validators[0].MaxPausedHeight == 0 {
					return
				}
				// ensure validator no longer exists in state
				val, err := sm.GetValidator(crypto.NewAddress(test.validators[0].Address))
				require.NoError(t, err)
				// ensure was force unstaked
				require.True(t, val.UnstakingHeight != 0)
			}()

			// STEP 5) delete validators who are finishing unstaking
			func() {
				// if testing with validators with a max pause height
				if len(test.validators) == 0 || test.validators[0].UnstakingHeight == 0 {
					return
				}
				// ensure validator no longer exists in state
				exists, err := sm.GetValidatorExists(crypto.NewAddress(test.validators[0].Address))
				require.NoError(t, err)
				// ensure no longer exists after unstaking
				require.False(t, exists)
				// ensure output address has staked funds
				balance, err := sm.GetAccountBalance(crypto.NewAddress(test.validators[0].Output))
				require.NoError(t, err)
				require.Equal(t, test.validators[0].StakedAmount, balance)
			}()
		})
	}
}

func TestCheckProtocolVersion(t *testing.T) {
	tests := []struct {
		name                  string
		detail                string
		localProtocolVersion  uint64
		localHeight           uint64
		protocolVersion       uint64
		protocolVersionHeight uint64
		error                 lib.ErrorI
	}{
		{
			name:        "same protocol version before height",
			detail:      "local protocol version == protocol version && local height < protocol version height. Like someone upgrading before the required height",
			localHeight: 0, localProtocolVersion: 1,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "same protocol version at height",
			detail:      "local protocol version == protocol version && local height == protocol version height. Like someone on the proper version at the upgrade height",
			localHeight: 1, localProtocolVersion: 1,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "same protocol version at future height",
			detail:      "local protocol version == protocol version && local height > protocol version height. Like someone on the proper version after the upgrade height",
			localHeight: 2, localProtocolVersion: 1,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "higher local protocol version before height",
			detail:      "local protocol version > protocol version && local height < protocol version height. Like someone beyond upgraded",
			localHeight: 0, localProtocolVersion: 2,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "same protocol version at height",
			detail:      "local protocol version : protocol version && local height == protocol version height. Like someone beyond upgraded at the upgrade height",
			localHeight: 1, localProtocolVersion: 2,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "same protocol version at future height",
			detail:      "local protocol version > protocol version && local height > protocol version height. Like someone upgraded before the change parameter txn sent",
			localHeight: 2, localProtocolVersion: 2,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "higher protocol version before height",
			detail:      "local protocol version < protocol version && local height < protocol version height. Like someone not upgraded before the required height",
			localHeight: 0, localProtocolVersion: 0,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: nil,
		},
		{
			name:        "higher protocol version at height",
			detail:      "local protocol version < protocol version && local height == protocol version height. Like someone not upgraded at the required height",
			localHeight: 1, localProtocolVersion: 0,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: ErrInvalidProtocolVersion(),
		},
		{
			name:        "higher protocol version after height",
			detail:      "local protocol version < protocol version && local height == protocol version height. Like someone not upgraded after the required height",
			localHeight: 2, localProtocolVersion: 0,
			protocolVersionHeight: 1, protocolVersion: 1,
			error: ErrInvalidProtocolVersion(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newSingleAccountStateMachine(t)
			// set the local protocol version
			sm.ProtocolVersion = test.localProtocolVersion
			// set the local height
			sm.height = test.localHeight
			// get consensus params
			consParams, err := sm.GetParamsCons()
			require.NoError(t, err)
			// set the protocol version in state
			consParams.ProtocolVersion = NewProtocolVersion(test.protocolVersionHeight, test.protocolVersion)
			require.NoError(t, sm.SetParamsCons(consParams))
			// run the function call and ensure expected error is returned
			require.Equal(t, test.error, sm.CheckProtocolVersion())
		})
	}
}

func TestForceUnstakeMaxPaused(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		preset          []*Validator
		expected        []*Validator
		unstakingBlocks uint64
		height          uint64
	}{
		{
			name:   "single validator",
			detail: "only 1 validator",
			preset: []*Validator{
				{
					Address:         newTestAddressBytes(t),
					MaxPausedHeight: 2,
				},
			},
			expected: []*Validator{
				{
					Address:         newTestAddressBytes(t),
					MaxPausedHeight: 0,
					UnstakingHeight: 3,
				},
			},
			height:          2,
			unstakingBlocks: 1,
		},
		{
			name:   "multi validator all max paused",
			detail: "multiple validators all max paused",
			preset: []*Validator{
				{
					Address:         newTestAddressBytes(t),
					MaxPausedHeight: 2,
				},
				{
					Address:         newTestAddressBytes(t, 1),
					MaxPausedHeight: 2,
				},
			},
			expected: []*Validator{
				{
					Address:         newTestAddressBytes(t),
					MaxPausedHeight: 0,
					UnstakingHeight: 3,
				},
				{
					Address:         newTestAddressBytes(t, 1),
					MaxPausedHeight: 0,
					UnstakingHeight: 3,
				},
			},
			height:          2,
			unstakingBlocks: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set the height
			sm.height = test.height
			// set the unstaking blocks
			require.NoError(t, sm.UpdateParam(ParamSpaceVal, ParamUnstakingBlocks, &lib.UInt64Wrapper{Value: test.unstakingBlocks}))
			// get the validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// for each validator
			for _, val := range test.preset {
				// preset the validators as paused
				require.NoError(t, sm.SetValidatorPaused(crypto.NewAddress(val.Address), val, sm.Height()))
			}
			// execute the function call
			require.NoError(t, sm.ForceUnstakeMaxPaused())
			// validate the effects
			for _, expected := range test.expected {
				address := crypto.NewAddress(expected.Address)
				// get the validator from state
				got, e := sm.GetValidator(address)
				require.NoError(t, e)
				// compare got vs expected
				require.EqualExportedValues(t, expected, got)
				// validate pause key removed
				bz, e := sm.Get(KeyForPaused(sm.Height(), address))
				require.NoError(t, e)
				require.Len(t, bz, 0)
				// validate not paused on structure
				require.Zero(t, expected.MaxPausedHeight)
				// calculate the expected unstaking height
				expectedUnstakingHeight := valParams.UnstakingBlocks + sm.height
				// validate unstaking on structure
				require.Equal(t, expectedUnstakingHeight, expected.UnstakingHeight)
				// validate unstaking key exists
				bz, e = sm.Get(KeyForUnstaking(expectedUnstakingHeight, address))
				require.NoError(t, e)
				require.Len(t, bz, 1)
			}
		})
	}
}

func TestLastProposers(t *testing.T) {
	tests := []struct {
		name              string
		detail            string
		height            uint64
		newProposer       []byte
		previousProposers [][]byte
		expectedProposers [][]byte
	}{
		{
			name:              "genesis",
			detail:            "no previous proposers",
			height:            0,
			newProposer:       newTestAddressBytes(t),
			expectedProposers: [][]byte{newTestAddressBytes(t), {}, {}, {}, {}},
		},
		{
			name:              "after genesis",
			detail:            "1 previous proposer",
			height:            1,
			newProposer:       newTestAddressBytes(t),
			previousProposers: [][]byte{newTestAddressBytes(t, 1), {}, {}, {}, {}},
			expectedProposers: [][]byte{newTestAddressBytes(t, 1), newTestAddressBytes(t), {}, {}, {}},
		},
		{
			name:              "with previous proposers",
			detail:            "with 5 previous proposers",
			height:            5,
			newProposer:       newTestAddressBytes(t),
			previousProposers: [][]byte{newTestAddressBytes(t, 1), newTestAddressBytes(t, 2), newTestAddressBytes(t, 3), newTestAddressBytes(t, 4), newTestAddressBytes(t, 5)},
			expectedProposers: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 2), newTestAddressBytes(t, 3), newTestAddressBytes(t, 4), newTestAddressBytes(t, 5)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newSingleAccountStateMachine(t)
			// set the local height
			sm.height = test.height
			// set previous proposers (if any)
			if test.previousProposers != nil {
				require.NoError(t, sm.SetLastProposers(&lib.Proposers{
					Addresses: test.previousProposers,
				}))
			}
			// update previous proposers
			require.NoError(t, sm.UpdateLastProposers(test.newProposer))
			// retrieve the last proposers from state
			lastProposers, err := sm.GetLastProposers()
			require.NoError(t, err)
			// ensure expected
			require.Equal(t, test.expectedProposers, lastProposers.Addresses)
		})
	}
}

func (s *StateMachine) calculateRewardPerCommittee(t *testing.T, numberOfSubsidizedCommittees int) (mintAmountPerCommittee uint64, daoCut uint64) {
	govParams, err := s.GetParamsGov()
	require.NoError(t, err)
	// calculate the number of halvenings
	halvings := float64(s.height / uint64(BlocksPerHalvening))
	// each halving, the reward is divided by 2
	totalMintAmount := uint64(float64(InitialTokensPerBlock) / (math.Pow(2, halvings)))
	// calculate the amount left for the committees after the parameterized DAO cut
	mintAmountAfterDAOCut := lib.Uint64ReducePercentage(totalMintAmount, govParams.DaoRewardPercentage)
	// calculate the DAO cut
	daoCut = totalMintAmount - mintAmountAfterDAOCut
	// calculate the amount given to each qualifying committee
	mintAmountPerCommittee = mintAmountAfterDAOCut / uint64(numberOfSubsidizedCommittees)
	return
}
