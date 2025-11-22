package fsm

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"os"
	"testing"
)

func TestUpdateParam(t *testing.T) {
	type paramUpdate struct {
		space string
		name  string
		value proto.Message
	}
	tests := []struct {
		name   string
		detail string
		update paramUpdate
		error  string
	}{
		{
			name:   "unknown space",
			detail: "the parameter space passed is invalid",
			update: paramUpdate{
				space: "app",
				name:  ParamMaxCommittees,
				value: &lib.UInt64Wrapper{Value: 100},
			},
			error: "unknown param space",
		},
		{
			name:   "unknown param type",
			detail: "the parameter type passed is invalid",
			update: paramUpdate{
				space: "val",
				name:  ParamMaxCommittees,
				value: &lib.ConsensusValidator{},
			},
			error: "unknown param type",
		},
		{
			name:   "unknown param",
			detail: "the parameter passed is unknown (this space doesn't have a max committees)",
			update: paramUpdate{
				space: "cons",
				name:  ParamMaxCommittees,
				value: &lib.UInt64Wrapper{Value: 100},
			},
			error: "unknown param",
		},
		{
			name:   "validator param updated",
			detail: "an update to max committees under the validator param space",
			update: paramUpdate{
				space: "val",
				name:  ParamMaxCommittees,
				value: &lib.UInt64Wrapper{Value: 100},
			},
		},
		{
			name:   "consensus param updated",
			detail: "an update to protocol version under the consensus param space",
			update: paramUpdate{
				space: "cons",
				name:  ParamProtocolVersion,
				value: &lib.StringWrapper{
					Value: NewProtocolVersion(2, 2),
				},
			},
		},
		{
			name:   "governance param updated",
			detail: "an update to dao reward percentage under the governance param space",
			update: paramUpdate{
				space: "gov",
				name:  ParamDAORewardPercentage,
				value: &lib.UInt64Wrapper{Value: 100},
			},
		},
		{
			name:   "fee param updated",
			detail: "an update to certificate result tx fee under the fee param space",
			update: paramUpdate{
				space: "fee",
				name:  ParamCertificateResultsFee,
				value: &lib.UInt64Wrapper{Value: 100},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// extract the value from the object
			var (
				uint64Value *lib.UInt64Wrapper
				stringValue *lib.StringWrapper
			)
			if i, isUint64 := test.update.value.(*lib.UInt64Wrapper); isUint64 {
				uint64Value = i
			} else if s, isString := test.update.value.(*lib.StringWrapper); isString {
				stringValue = s
			}
			// execute function call
			err := sm.UpdateParam(test.update.space, test.update.name, test.update.value)
			require.Equal(t, test.error != "", err != nil)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get params object from state
			got, err := sm.GetParams()
			require.NoError(t, err)
			// validate the update
			switch test.update.name {
			case ParamMaxCommittees: // validator
				require.Equal(t, uint64Value.Value, got.Validator.MaxCommittees)
			case ParamProtocolVersion: // consensus
				require.Equal(t, stringValue.Value, got.Consensus.ProtocolVersion)
			case ParamDAORewardPercentage: // gov
				require.Equal(t, uint64Value.Value, got.Governance.DaoRewardPercentage)
			case ParamCertificateResultsFee: // fee
				require.Equal(t, uint64Value.Value, got.Fee.CertificateResultsFee)
			}
		})
	}
}

func TestConformStateToParamUpdate(t *testing.T) {
	const amount = uint64(100)
	// preset param sets to test the adjustment after the update
	defaultParams, higherMaxCommittee, lowerMaxCommittee := DefaultParams(), DefaultParams(), DefaultParams()
	// preset the default to have deterinism in this test
	defaultParams.Validator.MaxCommittees = 2
	// increment the default for the higher max committee
	higherMaxCommittee.Validator.MaxCommittees = defaultParams.Validator.MaxCommittees + 1
	// decrement the default for a lower max committee
	lowerMaxCommittee.Validator.MaxCommittees = defaultParams.Validator.MaxCommittees - 1
	// run the test cases
	tests := []struct {
		name               string
		detail             string
		previousParams     *Params
		presetValidators   []*Validator
		paramUpdate        *Params
		expectedValidators []*Validator
	}{
		{
			name:           "no conform, same max committee size",
			detail:         "no conform is required as the max committee size is the same",
			previousParams: defaultParams,
			presetValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
			},
			paramUpdate: defaultParams,
			expectedValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
			},
		},
		{
			name:           "no conform, greater than max committee size",
			detail:         "no conform is required as the max committee size grew",
			previousParams: defaultParams,
			presetValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
			},
			paramUpdate: higherMaxCommittee,
			expectedValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
			},
		},
		{
			name:           "conform, less than max committee size",
			detail:         "conform is required as the max committee size shrunk",
			previousParams: defaultParams,
			presetValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1},
				},
			},
			paramUpdate: lowerMaxCommittee,
			expectedValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{1},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0},
				},
			},
		},
		{
			name:           "conform variable committees, less than max committee size",
			detail:         "conform variable committees, is required as the max committee size shrunk",
			previousParams: defaultParams,
			presetValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0, 1, 2, 3},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{0, 1, 2, 3},
				},
			},
			paramUpdate: lowerMaxCommittee,
			expectedValidators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{0},
				},
				{
					Address:      newTestAddressBytes(t, 2),
					StakedAmount: amount,
					Committees:   []uint64{0},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: amount,
					Committees:   []uint64{1},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supply := &Supply{}
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the params
			require.NoError(t, sm.SetParams(test.paramUpdate))
			// preset the validators
			require.NoError(t, sm.SetValidators(test.presetValidators, supply))
			// set the supply
			require.NoError(t, sm.SetSupply(supply))
			// execute the function call
			require.NoError(t, sm.ConformStateToParamUpdate(test.previousParams))
			// get the validators after
			vals, err := sm.GetValidators()
			require.NoError(t, err)
			// check the got vs expected
			for i, got := range vals {
				require.EqualExportedValues(t, test.expectedValidators[i], got)
			}
		})
	}
}

func TestSetGetParams(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		expected *Params
	}{
		{
			name:     "default",
			detail:   "set the defaults",
			expected: DefaultParams(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// clear the params
			require.NoError(t, sm.SetParams(&Params{}))
			// execute the function call
			require.NoError(t, sm.SetParams(test.expected))
			// validate the set with 'get'
			got, err := sm.GetParams()
			require.NoError(t, err)
			require.EqualExportedValues(t, test.expected, got)
			// clear the params
			require.NoError(t, sm.SetParams(&Params{}))
			// execute the individual sets
			require.NoError(t, sm.SetParamsVal(test.expected.Validator))
			require.NoError(t, sm.SetParamsCons(test.expected.Consensus))
			require.NoError(t, sm.SetParamsGov(test.expected.Governance))
			require.NoError(t, sm.SetParamsFee(test.expected.Fee))
			// validate the set with 'get' from validator space
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			require.EqualExportedValues(t, test.expected.Validator, valParams)
			// validate the set with 'get' from consensus space
			consParams, err := sm.GetParamsCons()
			require.NoError(t, err)
			require.EqualExportedValues(t, test.expected.Consensus, consParams)
			// validate the set with 'get' from gov space
			govParams, err := sm.GetParamsGov()
			require.NoError(t, err)
			require.EqualExportedValues(t, test.expected.Governance, govParams)
			// validate the set with 'get' from fee space
			feeParams, err := sm.GetParamsFee()
			require.NoError(t, err)
			require.EqualExportedValues(t, test.expected.Fee, feeParams)
		})
	}
}

func TestApproveProposal(t *testing.T) {
	// create a new private key
	pk, err := crypto.NewEd25519PrivateKey()
	// pre-create a 'change parameter' proposal to use during testing
	a, err := lib.NewAny(&lib.StringWrapper{Value: NewProtocolVersion(3, 2)})
	require.NoError(t, err)
	msg := &MessageChangeParameter{
		ParameterSpace: "cons",
		ParameterKey:   ParamProtocolVersion,
		ParameterValue: a,
		StartHeight:    1,
		EndHeight:      2,
		Signer:         newTestAddressBytes(t),
	}
	//  pre-create the change parameter tx
	changeParamTx, err := NewChangeParamTxString(
		pk, "cons", ParamProtocolVersion, NewProtocolVersion(3, 2),
		1, 2, 1, 1, 10000, 1, "",
	)
	require.NoError(t, err)
	// convert it to json
	changeParamTxJSON, err := lib.MarshalJSONIndent(changeParamTx)
	require.NoError(t, err)
	// create a test 'list' that approves the proposal
	approveMsgList := GovProposals{}
	require.NoError(t, approveMsgList.Add(changeParamTxJSON, true))
	// create a test 'list' that rejects the proposal
	rejectMsgList := GovProposals{}
	require.NoError(t, rejectMsgList.Add(changeParamTxJSON, false))
	// populate the proposal hash in the msg
	msg.ProposalHash, err = TxHashFromJSON(changeParamTxJSON)
	// run test cases
	tests := []struct {
		name   string
		detail string
		height uint64
		preset GovProposals
		config GovProposalVoteConfig
		msg    GovProposal
		error  string
	}{
		{
			name:   "explicitly approved on list",
			detail: "no error because msg is explicitly approved via file",
			height: 1,
			preset: approveMsgList,
			config: GovProposalVoteConfig_APPROVE_LIST,
			msg:    msg,
		}, {
			name:   "explicitly rejected on list",
			detail: "error because msg is explicitly rejected via file",
			height: 1,
			preset: rejectMsgList,
			config: GovProposalVoteConfig_APPROVE_LIST,
			msg:    msg,
			error:  "proposal rejected",
		},
		{
			name:   "not on list",
			detail: "error because msg is not on list via file",
			height: 1,
			preset: GovProposals{},
			config: GovProposalVoteConfig_APPROVE_LIST,
			msg:    msg,
			error:  "proposal rejected",
		},
		{
			name:   "height before start",
			detail: "error because msg applied at height that is before start",
			height: 0,
			config: AcceptAllProposals,
			msg:    msg,
			error:  "proposal rejected",
		},
		{
			name:   "height after end",
			detail: "error because msg applied at height that is after end",
			height: 3,
			config: AcceptAllProposals,
			msg:    msg,
			error:  "proposal rejected",
		},
		{
			name:   "all proposals approved",
			detail: "configuration approves all proposals regardless of list",
			height: 1,
			preset: rejectMsgList,
			config: AcceptAllProposals,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      100,
				StartHeight: 1,
				EndHeight:   2,
			},
		},
		{
			name:   "all proposals approved with explicitly rejected",
			detail: "configuration approves all proposals regardless of list",
			height: 1,
			preset: rejectMsgList,
			config: AcceptAllProposals,
			msg:    msg,
		},
		{
			name:   "all proposals rejected with explicitly approved",
			detail: "configuration rejects all proposals regardless of list",
			height: 1,
			preset: approveMsgList,
			config: RejectAllProposals,
			msg:    msg,
			error:  "proposal rejected",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set height
			sm.height = test.height
			// set proposal config
			sm.proposeVoteConfig = test.config
			// set the config file path
			sm.Config.DataDirPath = "./"
			// write the test file
			require.NoError(t, test.preset.SaveToFile(sm.Config.DataDirPath))
			defer os.RemoveAll(lib.ProposalsFilePath)
			// execute the function
			err = sm.ApproveProposal(test.msg)
			// validate the 'approval'
			require.Equal(t, test.error != "", err != nil)
			if err != nil {
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func TestIsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		height          uint64
		protocolVersion string
		version         uint64
		expected        bool
	}{
		{
			name:            "before version",
			detail:          "before the version, on the height",
			height:          2,
			protocolVersion: "2/2",
			version:         3,
			expected:        false,
		},
		{
			name:            "before height",
			detail:          "before the height, on the version",
			height:          1,
			protocolVersion: "2/2",
			version:         2,
			expected:        false,
		},
		{
			name:            "on version / height",
			detail:          "exactly on the version and height",
			height:          2,
			protocolVersion: "2/2",
			version:         2,
			expected:        true,
		},
		{
			name:            "after height",
			detail:          "after the height",
			height:          3,
			protocolVersion: "2/2",
			version:         2,
			expected:        true,
		},
		{
			name:            "after version",
			detail:          "after the height",
			height:          3,
			protocolVersion: "2/2",
			version:         2,
			expected:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set height
			sm.height = test.height
			// update the protocol version
			require.NoError(t, sm.UpdateParam(ParamSpaceCons, ParamProtocolVersion, &lib.StringWrapper{Value: test.protocolVersion}))
			// execute the function
			require.Equal(t, test.expected, sm.IsFeatureEnabled(test.version))
		})
	}
}
