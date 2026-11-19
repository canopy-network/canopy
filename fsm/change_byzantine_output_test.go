package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestHandleMessageChangeByzantineOutput(t *testing.T) {
	tests := []struct {
		name              string
		proposalConfig    GovProposalVoteConfig
		unstakingHeight   uint64
		expectedError     string
		expectedOutputNew bool
	}{
		{
			name:              "approved unstaking validator",
			proposalConfig:    AcceptAllProposals,
			unstakingHeight:   10,
			expectedOutputNew: true,
		},
		{
			name:            "approved active validator",
			proposalConfig:  AcceptAllProposals,
			expectedError:   "validator is not unstaking",
			unstakingHeight: 0,
		},
		{
			name:            "rejected proposal",
			proposalConfig:  RejectAllProposals,
			unstakingHeight: 10,
			expectedError:   "proposal rejected",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			sm.height = 1
			sm.proposeVoteConfig = test.proposalConfig
			validatorAddress := newTestAddressBytes(t)
			oldOutput := newTestAddressBytes(t, 1)
			newOutput := newTestAddressBytes(t, 2)
			validator := &Validator{
				Address:         validatorAddress,
				PublicKey:       newTestPublicKeyBytes(t),
				NetAddress:      "tcp://validator.example:9001",
				StakedAmount:    123,
				Committees:      []uint64{1, 2},
				MaxPausedHeight: 0,
				UnstakingHeight: test.unstakingHeight,
				Output:          oldOutput,
				Compound:        true,
			}
			require.NoError(t, sm.SetValidator(validator))
			before := proto.Clone(validator).(*Validator)

			err := sm.HandleMessageChangeByzantineOutput(&MessageChangeByzantineOutput{
				ValidatorAddress: validatorAddress,
				OutputAddress:    newOutput,
				Signer:           newTestAddressBytes(t, 3),
				StartHeight:      1,
				EndHeight:        2,
			})
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
			}

			got, err := sm.GetValidator(crypto.NewAddressFromBytes(validatorAddress))
			require.NoError(t, err)
			expected := proto.Clone(before).(*Validator)
			if test.expectedOutputNew {
				expected.Output = newOutput
			}
			require.Equal(t, expected, got)
		})
	}
}

func TestMessageChangeByzantineOutputCheck(t *testing.T) {
	valid := &MessageChangeByzantineOutput{
		ValidatorAddress: newTestAddressBytes(t),
		OutputAddress:    newTestAddressBytes(t, 1),
		Signer:           newTestAddressBytes(t, 2),
		StartHeight:      1,
		EndHeight:        2,
	}
	require.NoError(t, valid.Check())

	invalidOutput := proto.Clone(valid).(*MessageChangeByzantineOutput)
	invalidOutput.OutputAddress = []byte{1}
	require.ErrorContains(t, invalidOutput.Check(), "output address size is invalid")

	invalidRange := proto.Clone(valid).(*MessageChangeByzantineOutput)
	invalidRange.EndHeight = invalidRange.StartHeight
	require.ErrorContains(t, invalidRange.Check(), "proposal block range is invalid")

	populatedHash := proto.Clone(valid).(*MessageChangeByzantineOutput)
	populatedHash.ProposalHash = "internal-only"
	require.Error(t, populatedHash.Check())
}

func TestChangeByzantineOutputRedirectsFinishedUnstakingFunds(t *testing.T) {
	sm := newTestStateMachine(t)
	sm.height = 1
	sm.proposeVoteConfig = AcceptAllProposals
	validatorAddress := newTestAddressBytes(t)
	oldOutput := newTestAddressBytes(t, 1)
	newOutput := newTestAddressBytes(t, 2)
	validator := &Validator{
		Address:      validatorAddress,
		PublicKey:    newTestPublicKeyBytes(t),
		StakedAmount: 123,
		Output:       oldOutput,
	}
	supply := new(Supply)
	require.NoError(t, sm.SetValidators([]*Validator{validator}, supply))
	require.NoError(t, sm.SetSupply(supply))
	require.NoError(t, sm.SetValidatorUnstaking(crypto.NewAddress(validatorAddress), validator, sm.Height()))

	require.NoError(t, sm.HandleMessageChangeByzantineOutput(&MessageChangeByzantineOutput{
		ValidatorAddress: validatorAddress,
		OutputAddress:    newOutput,
		Signer:           newTestAddressBytes(t, 3),
		StartHeight:      1,
		EndHeight:        2,
	}))
	require.NoError(t, sm.DeleteFinishedUnstaking())

	oldBalance, err := sm.GetAccountBalance(crypto.NewAddress(oldOutput))
	require.NoError(t, err)
	require.Zero(t, oldBalance)
	newBalance, err := sm.GetAccountBalance(crypto.NewAddress(newOutput))
	require.NoError(t, err)
	require.Equal(t, validator.StakedAmount, newBalance)
}

func TestChangeByzantineOutputUsesDedicatedFee(t *testing.T) {
	sm := newTestStateMachine(t)
	fees, err := sm.GetParamsFee()
	require.NoError(t, err)
	fees.ChangeParameterFee = 11
	fees.ChangeByzantineOutputFee = 22
	require.NoError(t, sm.SetParamsFee(fees))

	got, err := sm.GetFeeForMessageName(MessageChangeByzantineOutputName)
	require.NoError(t, err)
	require.Equal(t, uint64(22), got)
}
