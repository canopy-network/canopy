package lib

import (
	"bytes"
	"encoding/json"
	"sort"
	"testing"

	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/drand/kyber"
	"github.com/stretchr/testify/require"
)

func TestNewValidatorSet(t *testing.T) {
	// pre-define an ed25519 private key
	pk, e := crypto.NewEd25519PrivateKey()
	require.NoError(t, e)
	// predefine a bls point
	pub, e := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t))
	require.NoError(t, e)
	// convert to multi key
	multi, e := crypto.NewMultiBLSFromPoints([]kyber.Point{pub}, nil)
	require.NoError(t, e)
	// define test cases
	tests := []struct {
		name     string
		detail   string
		vals     *ConsensusValidators
		expected ValidatorSet
		error    string
	}{
		{
			name:   "wrong public key type",
			detail: "the public key of a Validator must be BLS",
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{{PublicKey: pk.PublicKey().Bytes()}},
			},
			error: "publicKeyFromBytes() failed",
		},
		{
			name:   "no validators",
			detail: "there are no validators in the set",
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{},
			},
			error: "there are no validators in the set",
		},
		{
			name:   "got vs expected",
			detail: "no errors, so compare got vs expected",
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{
					{
						PublicKey:   newTestPublicKeyBytes(t),
						VotingPower: 1,
					},
				},
			},
			expected: ValidatorSet{
				ValidatorSet: &ConsensusValidators{
					ValidatorSet: []*ConsensusValidator{
						{
							PublicKey:   newTestPublicKeyBytes(t),
							VotingPower: 1,
						},
					},
				},
				MultiPubKey:      multi,
				ValTotalPower:    1,
				ValMinimumMaj23:  1,
				ValNumValidators: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			got, err := NewValidatorSet(test.vals)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// compare got vs expected
			require.EqualExportedValues(t, test.expected, got)
		})
	}
}

func TestValidatorSetGetValidator(t *testing.T) {
	tests := []struct {
		name      string
		detail    string
		getPublic []byte
		vals      *ConsensusValidators
		expected  *ConsensusValidator
		error     string
	}{
		{
			name:      "not in validator set",
			detail:    "the validator doesn't exist in the set",
			getPublic: newTestPublicKeyBytes(t, 1),
			error:     "invalid validator index",
		},
		{
			name:      "not in validator set",
			detail:    "the validator doesn't exist in the set",
			getPublic: newTestPublicKeyBytes(t, 1),
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{
					{
						PublicKey:   newTestPublicKeyBytes(t),
						VotingPower: 1,
					},
				},
			},
			error: "not found in validator set",
		},
		{
			name:      "got vs expected",
			detail:    "no error, so compare got vs expected",
			getPublic: newTestPublicKeyBytes(t),
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{
					{
						PublicKey:   newTestPublicKeyBytes(t),
						VotingPower: 1,
					},
				},
			},
			expected: &ConsensusValidator{
				PublicKey:   newTestPublicKeyBytes(t),
				VotingPower: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				validatorSet ValidatorSet
				err          ErrorI
			)
			// pre-define a new validator set to test with
			if test.vals != nil {
				validatorSet, err = NewValidatorSet(test.vals)
				require.NoError(t, err)
			}
			// execute the function call
			got, err := validatorSet.GetValidator(test.getPublic)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// check got vs expected
			require.EqualExportedValues(t, test.expected, got)
		})
	}
}

func TestValidatorSetGetValidatorAndIndex(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		getPublic   []byte
		vals        *ConsensusValidators
		expected    *ConsensusValidator
		expectedIdx int
		error       string
	}{
		{
			name:      "not in validator set",
			detail:    "the validator doesn't exist in the set",
			getPublic: newTestPublicKeyBytes(t, 1),
			error:     "invalid validator index",
		},
		{
			name:      "not in validator set",
			detail:    "the validator doesn't exist in the set",
			getPublic: newTestPublicKeyBytes(t, 1),
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{
					{
						PublicKey:   newTestPublicKeyBytes(t),
						VotingPower: 1,
					},
				},
			},
			error: "not found in validator set",
		},
		{
			name:      "got vs expected",
			detail:    "no error, so compare got vs expected",
			getPublic: newTestPublicKeyBytes(t, 1),
			vals: &ConsensusValidators{
				ValidatorSet: []*ConsensusValidator{
					{
						PublicKey:   newTestPublicKeyBytes(t),
						VotingPower: 1,
					},
					{
						PublicKey:   newTestPublicKeyBytes(t, 1),
						VotingPower: 1,
					},
				},
			},
			expectedIdx: 1,
			expected: &ConsensusValidator{
				PublicKey:   newTestPublicKeyBytes(t, 1),
				VotingPower: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				validatorSet ValidatorSet
				err          ErrorI
			)
			// pre-define a new validator set to test with
			if test.vals != nil {
				validatorSet, err = NewValidatorSet(test.vals)
				require.NoError(t, err)
			}
			// execute the function call
			got, idx, err := validatorSet.GetValidatorAndIdx(test.getPublic)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// check got vs expected
			require.EqualExportedValues(t, test.expected, got)
			require.Equal(t, test.expectedIdx, idx)
		})
	}
}

func TestAggregateSignatureCheckBasic(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		as     *AggregateSignature
		error  string
	}{
		{
			name:   "empty aggregate signature",
			detail: "the aggregate signature is nil",
			as:     nil,
			error:  "empty aggregate signature",
		},
		{
			name:   "invalid signature length",
			detail: "the signature length is invalid",
			as: &AggregateSignature{
				Signature: []byte("bad_length"),
			},
			error: "invalid aggregate signature length",
		},
		{
			name:   "empty signer bitmap",
			detail: "the signer bitmap is empty",
			as: &AggregateSignature{
				Signature: bytes.Repeat([]byte("F"), 96),
			},
			error: "empty signer bitmap",
		},
		{
			name:   "empty signer bitmap",
			detail: "the signer bitmap is empty",
			as: &AggregateSignature{
				Signature: bytes.Repeat([]byte("F"), 96),
			},
			error: "empty signer bitmap",
		},
		{
			name:   "no error",
			detail: "there's no error, the aggregate signature is valid for basic checks",
			as: &AggregateSignature{
				Signature: bytes.Repeat([]byte("F"), 96),
				Bitmap:    []byte("not_empty"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			err := test.as.CheckBasic()
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func TestAggregateSignatureCheck(t *testing.T) {
	// pre-create a message to sign
	msg, msg2 := &QuorumCertificate{BlockHash: []byte("message")}, &QuorumCertificate{BlockHash: []byte("message2")}
	// pre-create the validator set
	vs, er := NewValidatorSet(&ConsensusValidators{
		ValidatorSet: []*ConsensusValidator{
			{
				PublicKey:   newTestPublicKeyBytes(t),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 1),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 2),
				VotingPower: 1,
			},
		},
	})
	require.NoError(t, er)
	// pre-create some public keys
	pub1, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t))
	require.NoError(t, err)
	pub2, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 1))
	require.NoError(t, err)
	pub3, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 2))
	require.NoError(t, err)
	// pre-create some multi-keys
	multiKey1, err := crypto.NewMultiBLSFromPoints([]kyber.Point{pub1, pub2, pub3}, nil)
	require.NoError(t, err)
	multiKey2, multiKey3 := multiKey1.Copy(), multiKey1.Copy()
	// pre-create signatures
	for i := 0; i < 3; i++ {
		signature := newTestKeyGroup(t, i).PrivateKey.Sign(msg.SignBytes())
		// full
		require.NoError(t, multiKey1.AddSigner(signature, i))
		// partial
		if i != 0 {
			require.NoError(t, multiKey2.AddSigner(signature, i))
		}
		// other message
		require.NoError(t, multiKey3.AddSigner(newTestKeyGroup(t, i).PrivateKey.Sign(msg2.SignBytes()), i))
	}
	// create the aggregate signatures
	asFull, err := multiKey1.AggregateSignatures()
	require.NoError(t, err)
	asPartial, err := multiKey2.AggregateSignatures()
	require.NoError(t, err)
	asDifferentMsg, err := multiKey3.AggregateSignatures()
	require.NoError(t, err)
	// define test cases
	tests := []struct {
		name      string
		detail    string
		as        *AggregateSignature
		error     string
		isPartial bool
	}{
		{
			name:   "fails checkBasic()",
			detail: "the AS fails the basic check as it's nil",
			as:     nil,
			error:  "empty aggregate signature",
		},
		{
			name:   "invalid aggregate signature",
			detail: "the AS is invalid because the message is different",
			as: &AggregateSignature{
				Signature: asDifferentMsg,
				Bitmap:    multiKey3.Bitmap(),
			},
			error: "invalid aggregate signature",
		},
		{
			name:   "partial aggregate signature",
			detail: "the AS is partial because < 2/3rds voting power signed",
			as: &AggregateSignature{
				Signature: asPartial,
				Bitmap:    multiKey2.Bitmap(),
			},
			isPartial: true,
		},
		{
			name:   "full aggregate signature",
			detail: "the AS is full because > 2/3rds power signed the message",
			as: &AggregateSignature{
				Signature: asFull,
				Bitmap:    multiKey1.Bitmap(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			gotIsPartial, e := test.as.Check(msg, vs)
			// validate if an error is expected
			require.Equal(t, e != nil, test.error != "", e)
			// validate actual error if any
			if e != nil {
				require.ErrorContains(t, e, test.error, err)
				return
			}
			// validate is partial
			require.Equal(t, test.isPartial, gotIsPartial)
		})
	}
}

func TestGetNonSigners(t *testing.T) {
	// pre-create a message to sign
	msg := &QuorumCertificate{BlockHash: []byte("message")}
	// pre-create the validator set
	vs, er := NewValidatorSet(&ConsensusValidators{
		ValidatorSet: []*ConsensusValidator{
			{
				PublicKey:   newTestPublicKeyBytes(t),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 1),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 2),
				VotingPower: 1,
			},
		},
	})
	require.NoError(t, er)
	// pre-create some public keys
	pub1, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t))
	require.NoError(t, err)
	pub2, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 1))
	require.NoError(t, err)
	pub3, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 2))
	require.NoError(t, err)
	// pre-create some multi-keys
	multiKey1, err := crypto.NewMultiBLSFromPoints([]kyber.Point{pub1, pub2, pub3}, nil)
	require.NoError(t, err)
	multiKey2, multiKey3 := multiKey1.Copy(), multiKey1.Copy()
	// pre-create signatures
	for i := 0; i < 3; i++ {
		signature := newTestKeyGroup(t, i).PrivateKey.Sign(msg.SignBytes())
		// full
		require.NoError(t, multiKey1.AddSigner(signature, i))
		// partial
		if i != 0 {
			require.NoError(t, multiKey2.AddSigner(signature, i))
		}
	}
	// create the aggregate signatures
	asFull, err := multiKey1.AggregateSignatures()
	require.NoError(t, err)
	partial, err := multiKey2.AggregateSignatures()
	require.NoError(t, err)
	noSigners, err := multiKey3.AggregateSignatures()
	require.NoError(t, err)
	// define test cases
	tests := []struct {
		name                    string
		detail                  string
		as                      *AggregateSignature
		expectedNonSigners      [][]byte
		expectedNonSignerPercet uint64
	}{
		{
			name:   "no signers",
			detail: "there are no signers associated with this multi-key",
			as: &AggregateSignature{
				Signature: noSigners,
				Bitmap:    multiKey3.Bitmap(),
			},
			expectedNonSigners: [][]byte{
				newTestPublicKeyBytes(t),
				newTestPublicKeyBytes(t, 1),
				newTestPublicKeyBytes(t, 2),
			},
			expectedNonSignerPercet: 100,
		},
		{
			name:   "some signers",
			detail: "there are 2/3 signers associated with this multi-key",
			as: &AggregateSignature{
				Signature: partial,
				Bitmap:    multiKey2.Bitmap(),
			},
			expectedNonSigners: [][]byte{
				newTestPublicKeyBytes(t),
			},
			expectedNonSignerPercet: 33,
		},
		{
			name:   "all signers",
			detail: "there are 3/3 signers associated with this multi-key",
			as: &AggregateSignature{
				Signature: asFull,
				Bitmap:    multiKey1.Bitmap(),
			},
			expectedNonSigners:      nil,
			expectedNonSignerPercet: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			gotNonSigners, gotNonSignerPercent, e := test.as.GetNonSigners(vs.ValidatorSet)
			require.NoError(t, e)
			// compare got vs expected
			require.Equal(t, test.expectedNonSigners, gotNonSigners)
			require.EqualValues(t, test.expectedNonSignerPercet, gotNonSignerPercent, gotNonSignerPercent)
		})
	}
}

func TestGetDoubleSigners(t *testing.T) {
	// pre-create a message to sign
	msg := &QuorumCertificate{BlockHash: []byte("message")}
	// pre-create the validator set
	vs, er := NewValidatorSet(&ConsensusValidators{
		ValidatorSet: []*ConsensusValidator{
			{
				PublicKey:   newTestPublicKeyBytes(t),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 1),
				VotingPower: 1,
			},
			{
				PublicKey:   newTestPublicKeyBytes(t, 2),
				VotingPower: 1,
			},
		},
	})
	require.NoError(t, er)
	// pre-create some public keys
	pub1, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t))
	require.NoError(t, err)
	pub2, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 1))
	require.NoError(t, err)
	pub3, err := crypto.BytesToBLS12381Point(newTestPublicKeyBytes(t, 2))
	require.NoError(t, err)
	// pre-create some multi-keys
	multiKey1, err := crypto.NewMultiBLSFromPoints([]kyber.Point{pub1, pub2, pub3}, nil)
	require.NoError(t, err)
	multiKey2, multiKey3 := multiKey1.Copy(), multiKey1.Copy()
	// pre-create signatures
	for i := 0; i < 3; i++ {
		signature := newTestKeyGroup(t, i).PrivateKey.Sign(msg.SignBytes())
		// full
		require.NoError(t, multiKey1.AddSigner(signature, i))
		// partial
		if i != 0 {
			require.NoError(t, multiKey2.AddSigner(signature, i))
		} else {
			require.NoError(t, multiKey3.AddSigner(signature, i))
		}
	}
	// create the aggregate signatures
	asFull, err := multiKey1.AggregateSignatures()
	require.NoError(t, err)
	partial, err := multiKey2.AggregateSignatures()
	require.NoError(t, err)
	otherPartial, err := multiKey3.AggregateSignatures()
	require.NoError(t, err)
	// define test cases
	tests := []struct {
		name                  string
		detail                string
		as1                   *AggregateSignature
		as2                   *AggregateSignature
		expectedDoubleSigners [][]byte
	}{
		{
			name:   "no double signers",
			detail: "there are no overlapping signers between the two keys",
			as1: &AggregateSignature{
				Signature: partial,
				Bitmap:    multiKey2.Bitmap(),
			},
			as2: &AggregateSignature{
				Signature: otherPartial,
				Bitmap:    multiKey3.Bitmap(),
			},
			expectedDoubleSigners: nil,
		},
		{
			name:   "one double signers",
			detail: "there is 1 overlapping signer between the two keys",
			as1: &AggregateSignature{
				Signature: asFull,
				Bitmap:    multiKey1.Bitmap(),
			},
			as2: &AggregateSignature{
				Signature: otherPartial,
				Bitmap:    multiKey3.Bitmap(),
			},
			expectedDoubleSigners: [][]byte{newTestPublicKeyBytes(t)},
		},
		{
			name:   "all double signers",
			detail: "all signers are overlapping between the two keys",
			as1: &AggregateSignature{
				Signature: asFull,
				Bitmap:    multiKey1.Bitmap(),
			},
			as2: &AggregateSignature{
				Signature: asFull,
				Bitmap:    multiKey1.Bitmap(),
			},
			expectedDoubleSigners: [][]byte{newTestPublicKeyBytes(t), newTestPublicKeyBytes(t, 1), newTestPublicKeyBytes(t, 2)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			doubleSigners, e := test.as1.GetDoubleSigners(test.as2, vs)
			require.NoError(t, e)
			// check got vs expected
			require.Equal(t, test.expectedDoubleSigners, doubleSigners)
		})
	}
}

func TestAggregateSignatureJSON(t *testing.T) {
	aggregateSignature := &AggregateSignature{
		Signature: []byte("signature"),
		Bitmap:    []byte("bitmap"),
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(aggregateSignature)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(AggregateSignature)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.Equal(t, aggregateSignature, got)
}

func TestConsensusValidatorJSON(t *testing.T) {
	expected := &ConsensusValidator{
		PublicKey:   newTestPublicKeyBytes(t),
		VotingPower: 100,
		NetAddress:  "tcp://example.com",
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(ConsensusValidator)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.Equal(t, expected, got)
}

func TestViewCheckBasic(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		view   *View
		error  string
	}{
		{
			name:   "nil view",
			detail: "view is empty or nil",
			view:   nil,
			error:  "empty view",
		},
		{
			name:   "no error",
			detail: "view passes with no error",
			view:   &View{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			err := test.view.CheckBasic()
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func TestViewCheck(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		view           *View
		checkView      *View
		enforceHeights bool
		error          string
	}{
		{
			name:   "nil view",
			detail: "view is empty or nil",
			view:   nil,
			error:  "empty view",
		},
		{
			name:      "the network id is incorrect",
			detail:    "the network ids are different",
			view:      &View{NetworkId: 1},
			checkView: &View{NetworkId: 2},
			error:     "wrong network id",
		},
		{
			name:      "the chain id is incorrect",
			detail:    "the chain ids are different",
			view:      &View{ChainId: 1},
			checkView: &View{ChainId: 2},
			error:     "wrong chain id",
		},
		{
			name:           "the height is incorrect",
			detail:         "the height are different",
			view:           &View{Height: 1},
			checkView:      &View{Height: 2},
			enforceHeights: true,
			error:          "wrong view height",
		},
		{
			name:           "the canopy height is incorrect",
			detail:         "the canopy heights are different",
			view:           &View{RootHeight: 1},
			checkView:      &View{RootHeight: 2},
			enforceHeights: true,
			error:          "wrong root height",
		},
		{
			name:           "the height is incorrect but not enforcing heights",
			detail:         "the height are different but not enforcing heights",
			view:           &View{Height: 1},
			checkView:      &View{Height: 2},
			enforceHeights: false,
		},
		{
			name:           "the canopy height is incorrect but not enforcing heights",
			detail:         "the canopy heights are different but not enforcing heights",
			view:           &View{RootHeight: 1},
			checkView:      &View{RootHeight: 2},
			enforceHeights: false,
		},
		{
			name:           "no error",
			detail:         "view passes with no error",
			view:           &View{},
			checkView:      &View{},
			enforceHeights: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			err := test.view.Check(test.checkView, test.enforceHeights)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func TestViewCopy(t *testing.T) {
	// pre-define view
	view := &View{
		NetworkId:  1,
		ChainId:    2,
		Height:     3,
		RootHeight: 4,
		Round:      5,
		Phase:      6,
	}
	// execute function call
	got := view.Copy()
	// compare got vs expected
	require.EqualExportedValues(t, view, got)
}

func TestViewEquals(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		v1     *View
		v2     *View
		equals bool
	}{
		{
			name:   "nil or empty",
			detail: "a nil view triggers 'not equals' automatically",
		},
		{
			name:   "different heights",
			detail: "different heights triggers 'not equals'",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     999,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
		},
		{
			name:   "different canopy height",
			detail: "different canopy heights triggers 'not equals'",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 999,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
		},
		{
			name:   "different round",
			detail: "different rounds triggers 'not equals'",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      999,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
		},
		{
			name:   "different phase",
			detail: "different phase triggers 'not equals'",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      999,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
		},

		{
			name:   "different network id",
			detail: "different network id triggers 'not equals'",
			v1: &View{
				NetworkId:  999,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			require.Equal(t, test.equals, test.v1.Equals(test.v2))
		})
	}
}

func TestViewLess(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		v1     *View
		v2     *View
		less   bool
	}{
		{
			name:   "v1 nil or empty",
			detail: "v1 nil or empty triggers 'less'",
			v1:     nil,
			v2:     &View{},
			less:   true,
		},
		{
			name:   "v2 nil or empty",
			detail: "v2 nil or empty triggers 'not less'",
			v1:     &View{},
		},
		{
			name:   "height less",
			detail: "v1's height is less than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     999,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: true,
		},
		{
			name:   "canopy height less",
			detail: "v1's canopy height is less than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 999,
				Round:      4,
				Phase:      5,
			},
			less: true,
		},
		{
			name:   "round less",
			detail: "v1's round is less than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      999,
				Phase:      5,
			},
			less: true,
		},
		{
			name:   "phase less",
			detail: "v1's phase is less than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      999,
			},
			less: true,
		},
		{
			name:   "equal",
			detail: "v1 is equal to v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: false,
		},
		{
			name:   "greater height",
			detail: "v1's height is greater than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     999,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: false,
		},
		{
			name:   "greater canopy height",
			detail: "v1's canopy height is greater than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 999,
				Round:      4,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: false,
		},
		{
			name:   "greater round",
			detail: "v1's round is greater than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      999,
				Phase:      5,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: false,
		},
		{
			name:   "greater phase",
			detail: "v1's phase is greater than v2's",
			v1: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      5,
				Phase:      999,
			},
			v2: &View{
				NetworkId:  0,
				ChainId:    1,
				Height:     2,
				RootHeight: 3,
				Round:      4,
				Phase:      5,
			},
			less: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			require.Equal(t, test.less, test.v1.Less(test.v2))
		})
	}
}

func TestViewJSON(t *testing.T) {
	expected := &View{
		NetworkId:  0,
		ChainId:    1,
		Height:     2,
		RootHeight: 3,
		Round:      4,
		Phase:      5,
	}
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(View)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.Equal(t, expected, got)
}

func TestPhaseJSON(t *testing.T) {
	expected := Phase(2)
	// convert structure to json bytes
	gotBytes, err := json.Marshal(expected)
	require.NoError(t, err)
	// convert bytes to structure
	got := new(Phase)
	// unmarshal into bytes
	require.NoError(t, json.Unmarshal(gotBytes, got))
	// compare got vs expected
	require.Equal(t, &expected, got)
}

func TestAddHeight(t *testing.T) {
	// pre-define expected
	expected := []uint64{0, 1, 2, 3, 4, 5}
	// pre-define a double signer
	doubleSigner := &DoubleSigner{
		Id:      newTestPublicKeyBytes(t),
		Heights: []uint64{1, 2},
	}
	for i := 0; i < len(expected); i++ {
		// execute func call
		doubleSigner.AddHeight(uint64(i))
	}
	// sort the heights for comparison
	sort.Slice(doubleSigner.Heights, func(i, j int) bool {
		return doubleSigner.Heights[i] < doubleSigner.Heights[j]
	})
	// check got vs expected
	require.Equal(t, expected, doubleSigner.Heights)
}

func TestDoubleSignerEquals(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		ds1    *DoubleSigner
		ds2    *DoubleSigner
		equals bool
	}{
		{
			name:   "nil or empty",
			detail: "both nil means equals is true",
			ds1:    nil,
			ds2:    nil,
			equals: true,
		},
		{
			name:   "one nil or empty",
			detail: "one nil means equals is false",
			ds1:    &DoubleSigner{},
			ds2:    nil,
			equals: false,
		},
		{
			name:   "public keys",
			detail: "the public keys are different",
			ds1: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t),
				Heights: nil,
			},
			ds2: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t, 1),
				Heights: nil,
			},
			equals: false,
		},
		{
			name:   "heights",
			detail: "the heights are different",
			ds1: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{0},
			},
			ds2: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{1},
			},
			equals: false,
		},
		{
			name:   "same",
			detail: "both double signer objects are the same",
			ds1: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{0},
			},
			ds2: &DoubleSigner{
				Id:      newTestPublicKeyBytes(t),
				Heights: []uint64{0},
			},
			equals: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			require.Equal(t, test.equals, test.ds1.Equals(test.ds2))
		})
	}
}
