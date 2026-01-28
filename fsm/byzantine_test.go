package fsm

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	fmt.Println(lib.Uint64ReducePercentage(1, 1))
}

func TestHandleByzantine(t *testing.T) {
	// IMPORTANT NOTE: the amount of case testing is limited here due to the amount of code covered in this function
	// The individual unit tests for the individual functions covers many more cases for each

	const stakeAmount = uint64(100)
	// pre-generate a set of 4 keys
	var validators []*Validator
	keyGroups := newTestKeyGroups(t, 4)

	// pre-define 4 equally staked validators from those keys
	for _, k := range keyGroups {
		validators = append(validators, &Validator{
			Address:      k.Address.Bytes(),
			PublicKey:    k.PublicKey.Bytes(),
			StakedAmount: stakeAmount,
			Committees:   []uint64{lib.CanopyChainId},
		})
	}

	tests := []struct {
		name                 string
		detail               string
		slashResetNonSigners bool
		qc                   *lib.QuorumCertificate
		error                lib.ErrorI
	}{
		{
			name:   "a non signer with no previous missed blocks",
			detail: "one non signer that has no history within the 'non-signers-window' of not signing blocks",
			qc: newTestQC(t, testQCParams{
				idxSigned:     map[int]bool{0: true, 1: true, 2: true, 3: false},
				committeeKeys: keyGroups,
				committee:     validators,
				results:       &lib.CertificateResult{},
			}),
		},
		{
			name:   "non signer with previous missed blocks at the reset point",
			detail: "one non signer that has no history within the 'non-signers-window' of not signing blocks",
			qc: newTestQC(t, testQCParams{
				idxSigned:     map[int]bool{0: true, 1: true, 2: true, 3: true},
				committeeKeys: keyGroups,
				committee:     validators,
				results:       &lib.CertificateResult{},
			}),
			slashResetNonSigners: true,
		},
		{
			name:   "double signer",
			detail: "a valid double signer included",
			qc: newTestQC(t, testQCParams{
				idxSigned:     map[int]bool{0: true, 1: true, 2: true, 3: true},
				committeeKeys: keyGroups,
				committee:     validators,
				results: &lib.CertificateResult{
					SlashRecipients: &lib.SlashRecipients{
						DoubleSigners: []*lib.DoubleSigner{
							{
								Id:      keyGroups[0].PublicKey.Bytes(),
								Heights: []uint64{0},
							},
						},
					},
				},
			}),
		},
	}

	// run the test cases
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params for function call
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// set state machine height
			sm.height = 3
			// if testing the non signers reset, set the height to the reset window as the modulo 0 condition will trigger
			if test.slashResetNonSigners {
				sm.height = valParams.NonSignWindow
			}

			// STEP 0) inject input data into state

			// inject validator
			for _, val := range validators {
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(val.StakedAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(val.StakedAmount))
				// add the validator to state
				require.NoError(t, sm.SetValidator(val))
				// add the test validators to the committee structure in the state
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(val.Address), val.StakedAmount, val.Committees))
			}
			// get committee to have a reference to the validator set handy
			committee, err := sm.GetCommitteeMembers(lib.CanopyChainId)
			require.NoError(t, err)
			// generate non-signer history for the first validator for 'reset and slash' testing
			for j := uint64(0); j <= valParams.MaxNonSign; j++ {
				require.NoError(t, sm.IncrementNonSigners([][]byte{committee.Validators().ValidatorSet[0].PublicKey}))
			}
			// get the non-signers of the QC from the committee
			expectedNonSigners, expectedPercent, err := test.qc.GetNonSigners(committee.Validators())
			require.NoError(t, err)

			// STEP 1) execute function call
			func() {
				// run the function call
				nonSignerPercent, e := sm.HandleByzantine(test.qc, &committee)
				// ensure expected error
				require.Equal(t, test.error, e)
				// ensure expected percent of non signers
				require.Equal(t, expectedPercent, nonSignerPercent)
			}()

			// STEP 2) validate 'non signer' logic
			func() {
				// get the non-signers from state
				nonSigners, e := sm.GetNonSigners()
				require.NoError(t, e)
				// for each expected non signer
				for _, nonSigner := range expectedNonSigners {
					// ensure the non-signers array was updated with the expected key
					require.True(t, slices.ContainsFunc(nonSigners, func(ns *NonSigner) bool {
						pub, _ := crypto.BytesToBLS12381Public(nonSigner)
						return bytes.Equal(ns.Address, pub.Address().Bytes())
					}))
				}
				// validate non-signer reset and slashing
				if test.slashResetNonSigners {
					// validate the reset
					require.Zero(t, len(nonSigners))
					// retrieve the validator object
					pub, _ := crypto.BytesToBLS12381Public(committee.ValidatorSet.ValidatorSet[0].PublicKey)
					validator, _ := sm.GetValidator(pub.Address())
					// validate the pausing
					require.NotZero(t, validator.MaxPausedHeight)
					// validate the slashing
					require.Less(t, validator.StakedAmount, stakeAmount)
				}
			}()

			// STEP 3) validate 'double signer' logic
			func() {
				if test.qc.Results.SlashRecipients != nil && test.qc.Results.SlashRecipients.DoubleSigners != nil {
					// commit because GetDoubleSigners() doesn't work for same block
					_, e := sm.store.(lib.StoreI).Commit()
					require.NoError(t, e)
					// retrieve the double signers
					doubleSigners, e := sm.GetDoubleSigners()
					require.NoError(t, e)
					// validate the count of double signers
					require.Len(t, doubleSigners, 1)
					// get the validator associated with the double signer
					// NOTE: GetDoubleSigners populates with the address NOT the public key...
					validator, e := sm.GetValidator(crypto.NewAddress(doubleSigners[0].Id))
					require.NoError(t, e)
					// validate the slash of the double signer
					require.Less(t, validator.StakedAmount, stakeAmount)
				}
			}()
		})
	}
}

func TestSlashAndResetNonSigners(t *testing.T) {
	const stakeAmount = uint64(100)
	tests := []struct {
		name       string
		detail     string
		nonSigners NonSigners
		error      lib.ErrorI
	}{
		{
			name:   "no non-signers",
			detail: "there are no preset non signers",
		},
		{
			name:   "non-slashable-signer",
			detail: "there exists one non-signer who is not eligible for slashing as they are LTE the 'max' non-signs",
			nonSigners: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: DefaultParams().Validator.MaxNonSign,
			}},
		},
		{
			name:   "slashable-signer",
			detail: "there exists one non-signer who is eligible for slashing as they are above the 'max' non-signs",
			nonSigners: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: DefaultParams().Validator.MaxNonSign + 1,
			}},
		},
		{
			name:   "one slashable, one non-slashable non-signer",
			detail: "there exists one non-signer who is eligible for slashing and one who is not eligible",
			nonSigners: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: DefaultParams().Validator.MaxNonSign,
			}, {
				Address: newTestAddressBytes(t, 1),
				Counter: DefaultParams().Validator.MaxNonSign + 1,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// retrieve the validator parameters
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// inject the non signers into state
			for _, nonSigner := range test.nonSigners {
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(stakeAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(stakeAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToCommitteeSupplyForChain(lib.CanopyChainId, stakeAmount))
				// set the non signer as a validator in state
				require.NoError(t, sm.SetValidator(&Validator{
					Address:      nonSigner.Address,
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				}))
				// convert the non signer to bytes
				bz, e := lib.Marshal(&NonSigner{
					Counter: nonSigner.Counter,
				})
				require.NoError(t, e)
				// set the non signer in state
				require.NoError(t, sm.Set(KeyForNonSigner(nonSigner.Address), bz))
			}
			// retrieve the supply object before the call
			beforeSupply, err := sm.GetSupply()
			require.NoError(t, err)
			// run the function call
			err = sm.SlashAndResetNonSigners(lib.CanopyChainId, valParams)
			// check for the expected error
			require.Equal(t, test.error, err)
			if err != nil {
				return
			}
			// retrieve the non-signers after the fact
			nonSigners, err := sm.GetNonSigners()
			require.NoError(t, err)
			// check the reset
			require.Zero(t, len(nonSigners))
			// validate the state of the actors after the fact
			for _, nonSigner := range test.nonSigners {
				// retrieve the validators after the fact
				val, e := sm.GetValidator(crypto.NewAddress(nonSigner.Address))
				require.NoError(t, e)
				// if the validator was passed the max, it qualified for a slash
				if nonSigner.Counter > valParams.MaxNonSign {
					// retrieve the supply after the fact
					afterSupply, e := sm.GetSupply()
					require.NoError(t, e)
					// validate the reduction in supply
					require.Less(t, afterSupply.Total, beforeSupply.Total)
					// validate the reduction in staked supply
					require.Less(t, afterSupply.Staked, beforeSupply.Staked)
					// validate the auto-pause
					require.NotZero(t, val.MaxPausedHeight)
					// validate the slash
					require.Less(t, val.StakedAmount, stakeAmount)
				} else {
					// validate no auto-pause
					require.Zero(t, val.MaxPausedHeight)
					// validate no slash
					require.Equal(t, stakeAmount, val.StakedAmount)
				}
			}
		})
	}
}

func TestIncrementNonSigners(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		preset     NonSigners
		nonSigners [][]byte
		expected   NonSigners
		error      string
	}{
		{
			name:       "invalid public key",
			detail:     "the non-signer passed is an invalid pub key",
			nonSigners: [][]byte{newTestAddressBytes(t)},
			error:      "publicKeyFromBytes() failed with err",
		},
		{
			name:   "zero preset and zero non-sign",
			detail: "there are no preset non signers in the state and none didn't sign",
		},
		{
			name:       "zero preset and one non-sign",
			detail:     "there are no preset non signers in the state and 1 new didn't sign",
			nonSigners: [][]byte{newTestPublicKeyBytes(t)},
			expected: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: 1,
			}},
		},
		{
			name:   "one preset and one non-sign",
			detail: "there is 1 preset non signers in the state and it didn't sign again",
			preset: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: 1,
			}},
			nonSigners: [][]byte{newTestPublicKeyBytes(t)},
			expected: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: 2,
			}},
		},
		{
			name:   "two preset and one non-sign",
			detail: "there is 2 preset non signers in the state and 1 didn't sign again",
			preset: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: 1,
			}, {
				Address: newTestAddressBytes(t, 1),
				Counter: 1,
			}},
			nonSigners: [][]byte{newTestPublicKeyBytes(t)},
			expected: NonSigners{{
				Address: newTestAddressBytes(t),
				Counter: 2,
			}, {
				Address: newTestAddressBytes(t, 1),
				Counter: 1,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// pre-set the non signers
			for _, nonSigner := range test.preset {
				// create the non signer info
				nonSignerInfo := &NonSigner{Counter: nonSigner.Counter}
				// convert it to bytes
				bz, err := lib.Marshal(nonSignerInfo)
				require.NoError(t, err)
				// set the non-signer
				require.NoError(t, sm.Set(KeyForNonSigner(nonSigner.Address), bz))
			}
			// execute the function call and check for expected error
			err := sm.IncrementNonSigners(test.nonSigners)
			// check for expected error
			if err != nil {
				require.NotEmpty(t, test.error)
				require.ErrorContains(t, err, test.error)
				return
			}
			// retrieve the non signers
			nonSigners, err := sm.GetNonSigners()
			require.NoError(t, err)
			// check against the expected
			for i, expected := range test.expected {
				require.EqualExportedValues(t, expected, nonSigners[i])
			}
		})
	}
}

func TestHandleDoubleSigners(t *testing.T) {
	tests := []struct {
		name          string
		detail        string
		preset        []*lib.DoubleSigner
		doubleSigners []*lib.DoubleSigner
		error         lib.ErrorI
	}{
		{
			name:   "0",
			detail: "there are no double signers and no slashes",
		},
		{
			name:          "nil",
			detail:        "there is 1 invalid double signer, empty",
			doubleSigners: []*lib.DoubleSigner{nil},
			error:         lib.ErrEmptyDoubleSigner(),
		},
		{
			name:   "bad heights",
			detail: "there is 1 invalid double signer, bad heights",
			doubleSigners: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{},
			}},
			error: lib.ErrInvalidDoubleSignHeights(),
		},
		{
			name:   "already indexed",
			detail: "there is 1 invalid double signer, already indexed",
			preset: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1},
			}},
			doubleSigners: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1},
			}},
			error: lib.ErrInvalidDoubleSigner(),
		},
		{
			name:   "1 double signer for 1 height",
			detail: "there is 1 valid double signer for a single height",
			doubleSigners: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1},
			}},
		},
		{
			name:   "1 double signer for 2 heights",
			detail: "there is 1 valid double signer for two heights",
			doubleSigners: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1, 2},
			}},
		},
		{
			name:   "2 double signer for various heights",
			detail: "there is 2 valid double signer for various heights",
			doubleSigners: []*lib.DoubleSigner{{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1, 2},
			}, {
				Id:      newTestPublicKeyBytes(t, 1),
				Heights: []uint64{3, 4, 5},
			}},
		},
	}
	for _, test := range tests {
		const stakeAmount = uint64(100)
		t.Run(test.name, func(t *testing.T) {
			var pubs []crypto.PublicKeyI
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// nullify the slash tracker to ensure no conflicts with slash amount validation
			valParams, e := sm.GetParamsVal()
			require.NoError(t, e)
			valParams.DoubleSignSlashPercentage = 1
			require.NoError(t, sm.SetParamsVal(valParams))
			s := sm.Store().(lib.StoreI)
			// preset the double signers
			for _, doubleSigner := range test.preset {
				// get the address of the double signer
				pub, err := crypto.NewPublicKeyFromBytes(doubleSigner.Id)
				require.NoError(t, err)
				// pre-index the double signer
				for _, h := range doubleSigner.Heights {
					// generate address
					addr := pub.Address().Bytes()
					// ensure is a valid double signer
					ok, er := s.IsValidDoubleSigner(addr, h)
					require.NoError(t, er)
					require.True(t, ok)
					// index the double signer
					require.NoError(t, s.IndexDoubleSigner(addr, h))
					// ensure no longer is a valid double signer
					ok, er = s.IsValidDoubleSigner(addr, h)
					require.NoError(t, er)
					require.False(t, ok)
				}
			}
			// preset the validators
			for _, doubleSigner := range test.doubleSigners {
				// if the double signer is empty, skip
				if doubleSigner == nil {
					continue
				}
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(stakeAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(stakeAmount))
				// get the address of the double signer
				pub, err := crypto.NewPublicKeyFromBytes(doubleSigner.Id)
				require.NoError(t, err)
				// save the public key for later use in the test
				pubs = append(pubs, pub)
				// add to the committee supply
				require.NoError(t, sm.AddToCommitteeSupplyForChain(lib.CanopyChainId, stakeAmount))
				// set the double signer as a validator in state
				require.NoError(t, sm.SetValidator(&Validator{
					Address:      pub.Address().Bytes(),
					PublicKey:    pub.Bytes(),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				}))
			}
			// get the validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// run the function call
			err = sm.HandleDoubleSigners(lib.CanopyChainId, valParams, test.doubleSigners)
			// check for expected error
			require.Equal(t, test.error, err)
			if err != nil {
				return
			}
			// validate the slash
			for i, doubleSigner := range test.doubleSigners {
				// get the validator
				validator, e := sm.GetValidator(pubs[i].Address())
				require.NoError(t, e)
				// calculate the expected stake after slash
				expected := stakeAmount
				for _, height := range doubleSigner.Heights {
					// ensure no longer is a valid double signer
					ok, er := s.IsValidDoubleSigner(validator.Address, height)
					require.NoError(t, er)
					require.False(t, ok)
					// re-calculate the expected
					expected = lib.Uint64ReducePercentage(expected, valParams.DoubleSignSlashPercentage)
				}
				// validate the slash
				require.Equal(t, validator.StakedAmount, expected)
			}
		})
	}
}

func TestForceUnstakeValidator(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		validators     []*Validator
		forceUnstakers []crypto.AddressI
		success        bool
	}{
		{
			name:           "validator not found",
			detail:         "the validator does not exist",
			forceUnstakers: []crypto.AddressI{newTestAddress(t)},
		},
		{
			name:   "validator already unstaking",
			detail: "the validator is already unstaking",
			validators: []*Validator{
				{
					Address:         newTestAddressBytes(t),
					UnstakingHeight: 1,
					StakedAmount:    100,
				},
			},
			forceUnstakers: []crypto.AddressI{newTestAddress(t)},
		},
		{
			name:   "1 validator",
			detail: "one validator is force unstaked",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: 100,
				},
			},
			forceUnstakers: []crypto.AddressI{newTestAddress(t)},
			success:        true,
		},
		{
			name:   "2 validators both are force unstaked",
			detail: "two validators are force unstaked",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: 100,
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: 100,
				},
			},
			forceUnstakers: []crypto.AddressI{newTestAddress(t), newTestAddress(t, 1)},
			success:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params
			p, err := sm.GetParamsVal()
			require.NoError(t, err)
			// preset the validators
			for _, v := range test.validators {
				// set the bad proposer as a validator in state
				require.NoError(t, sm.SetValidator(v))
			}
			// for each test force unstaker
			for _, addr := range test.forceUnstakers {
				beforeVal, _ := sm.GetValidator(addr)
				// run the function call ensuring no errors
				require.NoError(t, sm.ForceUnstakeValidator(addr))
				// get the validator
				afterVal, _ := sm.GetValidator(addr)
				// if not supposed to succeed
				if !test.success {
					// ensure the validator is the same as before
					require.EqualExportedValues(t, beforeVal, afterVal)
					return
				}
				unstakingBlocks := p.GetUnstakingBlocks()
				unstakingHeight := sm.Height() + unstakingBlocks
				// validate the exact height
				require.Equal(t, afterVal.UnstakingHeight, unstakingHeight)
				// validate no committees
				require.Zero(t, len(afterVal.Committees))
			}
		})
	}
}

func TestSlash(t *testing.T) {
	// pre-define a stake amount for the validators
	stakeAmount := uint64(100)
	// pre define a slash structure
	type slash struct {
		Type    string
		Address []byte
		ChainId uint64
	}
	// pre-define the slash types
	const (
		doubleSignerSlash = "double_signer"
		nonSignerSlash    = "non_signer"
	)
	tests := []struct {
		name       string
		detail     string
		validators []*Validator
		slashes    []slash
		error      string
	}{
		{
			name:   "one double signer",
			detail: "one validator slashed as a double signer",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			slashes: []slash{
				{
					Type:    doubleSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
			},
		},
		{
			name:   "one non signer",
			detail: "one validator slashed as a non signer",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			slashes: []slash{
				{
					Type:    nonSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
			},
		},
		{
			name:   "one slashed for all",
			detail: "one validator slashed with all types",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			slashes: []slash{
				{
					Type:    doubleSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
				{
					Type:    nonSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
			},
		},
		{
			name:   "two slashed for all",
			detail: "two validators slashed with all types",
			validators: []*Validator{
				{
					Address:      newTestAddressBytes(t),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
				{
					Address:      newTestAddressBytes(t, 1),
					StakedAmount: stakeAmount,
					Committees:   []uint64{lib.CanopyChainId},
				},
			},
			slashes: []slash{
				{
					Type:    doubleSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
				{
					Type:    nonSignerSlash,
					Address: newTestAddressBytes(t),
					ChainId: lib.CanopyChainId,
				},
				{
					Type:    doubleSignerSlash,
					Address: newTestAddressBytes(t, 1),
					ChainId: lib.CanopyChainId,
				},
				{
					Type:    nonSignerSlash,
					Address: newTestAddressBytes(t, 1),
					ChainId: lib.CanopyChainId,
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
			// preset the validators
			for _, v := range test.validators {
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(stakeAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(stakeAmount))
				// set the bad proposer as a validator in state
				require.NoError(t, sm.SetValidator(v))
				// add to the committee supply
				require.NoError(t, sm.AddToCommitteeSupplyForChain(lib.CanopyChainId, stakeAmount))
			}
			// execute the slashes
			for _, s := range test.slashes {
				addr := crypto.NewAddress(s.Address)
				// get the validator before
				before, _ := sm.GetValidator(addr)
				// create a variable to hold the expected stake amount after the slash
				var expected uint64
				if before != nil {
					expected = before.StakedAmount
				}
				// slash based on the type
				switch s.Type {
				case doubleSignerSlash:
					err = sm.SlashDoubleSigners(s.ChainId, valParams, [][]byte{s.Address})
					expected = lib.Uint64ReducePercentage(expected, valParams.DoubleSignSlashPercentage)
				case nonSignerSlash:
					err = sm.SlashNonSigners(s.ChainId, valParams, [][]byte{s.Address})
					expected = lib.Uint64ReducePercentage(expected, valParams.NonSignSlashPercentage)
				default:
					t.Fatal("unknown slash type")
				}
				// check for expected error
				if err != nil {
					require.NotEmpty(t, test.error)
					require.ErrorContains(t, err, test.error)
					continue
				}
				// get the validator after
				after, e := sm.GetValidator(addr)
				require.NoError(t, e)
				// validate got vs expected
				require.Equal(t, expected, after.StakedAmount)
			}
		})
	}
}

func TestSlashTracker(t *testing.T) {
	// pre-define a stake amount for the validators
	stakeAmount := uint64(100)
	// pre define a slash structure
	type slash struct {
		Percent                   uint64
		Address                   []byte
		ChainId                   uint64
		expectedRemovedCommittees []uint64
		expectedTotalSlashState   uint64
	}
	tests := []struct {
		name                 string
		detail               string
		maxSlashPerCommittee uint64
		validators           []*Validator
		slashes              []slash
	}{
		{
			name:                 "1 validator, 1 committee, not max",
			detail:               "1 validator is slashed for 1 committee under the maximum slash",
			maxSlashPerCommittee: 15,
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}},
			slashes: []slash{{
				Percent:                   10,
				Address:                   newTestAddressBytes(t),
				ChainId:                   0,
				expectedRemovedCommittees: nil,
				expectedTotalSlashState:   10,
			}},
		},
		{
			name:                 "1 validator, 1 committee, over max",
			detail:               "1 validator is slashed for 1 committee over the maximum slash",
			maxSlashPerCommittee: 15,
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}},
			slashes: []slash{{
				Percent:                   20,
				Address:                   newTestAddressBytes(t),
				ChainId:                   1,
				expectedRemovedCommittees: []uint64{1},
				expectedTotalSlashState:   15,
			}},
		},
		{
			name:                 "1 validator, 2 committee, over max",
			detail:               "1 validator is slashed for 2 committees over the maximum slash",
			maxSlashPerCommittee: 15,
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}},
			slashes: []slash{{
				Percent:                   20,
				Address:                   newTestAddressBytes(t),
				ChainId:                   1,
				expectedRemovedCommittees: []uint64{1},
				expectedTotalSlashState:   15,
			}, {
				Percent:                   20,
				Address:                   newTestAddressBytes(t),
				ChainId:                   0,
				expectedRemovedCommittees: []uint64{0},
				expectedTotalSlashState:   15,
			}},
		},
		{
			name:                 "2 validator, 1 committee, under max",
			detail:               "2 validators are slashed for 1 committees under the maximum slash",
			maxSlashPerCommittee: 15,
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}, {
				Address:      newTestAddressBytes(t, 1),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}},
			slashes: []slash{{
				Percent:                 10,
				Address:                 newTestAddressBytes(t),
				ChainId:                 0,
				expectedTotalSlashState: 10,
			}, {
				Percent:                 10,
				Address:                 newTestAddressBytes(t, 1),
				ChainId:                 0,
				expectedTotalSlashState: 10,
			}},
		},

		{
			name:                 "2 validator, 1 committee, one over one under max",
			detail:               "2 validators are slashed for 1 committees. One of the slashes is over and one is under the maximum slash",
			maxSlashPerCommittee: 15,
			validators: []*Validator{{
				Address:      newTestAddressBytes(t),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}, {
				Address:      newTestAddressBytes(t, 1),
				Committees:   []uint64{0, 1},
				StakedAmount: stakeAmount,
			}},
			slashes: []slash{{
				Percent:                 10,
				Address:                 newTestAddressBytes(t),
				ChainId:                 0,
				expectedTotalSlashState: 10,
			}, {
				Percent:                   20,
				Address:                   newTestAddressBytes(t, 1),
				ChainId:                   0,
				expectedRemovedCommittees: []uint64{0},
				expectedTotalSlashState:   15,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// set max slash per committee based on test param
			valParams.MaxSlashPerCommittee = test.maxSlashPerCommittee
			// preset the validators
			for _, v := range test.validators {
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(v.StakedAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(v.StakedAmount))
				// set the bad proposer as a validator in state
				require.NoError(t, sm.SetValidator(v))
				// set validator committees
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			}
			// execute the slashes
			for _, s := range test.slashes {
				// convert to address object
				addr := crypto.NewAddress(s.Address)
				// retrieve the validator
				val, e := sm.GetValidator(addr)
				require.NoError(t, e)
				// execute the slash function call and ensure no error
				require.NoError(t, sm.SlashValidator(val, s.ChainId, s.Percent, valParams))
				// validate the slash tracker state
				require.Equal(t, s.expectedTotalSlashState, sm.slashTracker.GetTotalSlashPercent(s.Address, s.ChainId))
				// retrieve the validator
				val, e = sm.GetValidator(addr)
				require.NoError(t, err)
				// validate the removal of the committees
				for _, committee := range s.expectedRemovedCommittees {
					require.NotContains(t, val.Committees, committee)
				}
			}
		})
	}
}

func TestLoadMinimumEvidenceHeight(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		height          uint64
		unstakingBlocks uint64
		expected        uint64
	}{
		{
			name:            "height 0 so max evidence is 0",
			detail:          "the min evidence height is zero due to that being the only possible height",
			height:          0,
			unstakingBlocks: 25,
			expected:        0,
		},
		{
			name:            "height is less than unstaking blocks so max evidence is 0",
			detail:          "the min evidence height is zero due to unstaking blocks being less than the height",
			height:          24,
			unstakingBlocks: 25,
			expected:        0,
		},
		{
			name:            "height is exactly unstaking blocks so max evidence is 0",
			detail:          "the min evidence height is zero due to unstaking blocks being exactly the height",
			height:          25,
			unstakingBlocks: 25,
			expected:        0,
		},
		{
			name:            "height is exactly unstaking blocks so max evidence is 0",
			detail:          "the min evidence height is zero due to unstaking blocks being exactly the height",
			height:          26,
			unstakingBlocks: 25,
			expected:        1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set the state machine height
			sm.height = test.height
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// set unstaking blocks
			valParams.UnstakingBlocks = test.unstakingBlocks
			// set the params
			require.NoError(t, sm.SetParamsVal(valParams))
			sm.Store().(lib.StoreI).Commit()
			// run the function call with no errors
			got, err := sm.LoadMinimumEvidenceHeight()
			require.NoError(t, err)
			// validate got is expected
			require.Equal(t, test.expected, got)
		})
	}
}
