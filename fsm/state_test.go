package fsm

import (
	"encoding/json"
	"github.com/ginchuco/canopy/fsm/types"
	"github.com/ginchuco/canopy/lib"
	"github.com/ginchuco/canopy/lib/crypto"
	"github.com/ginchuco/canopy/store"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	const dataDirPath = "./"
	tests := []struct {
		name          string
		detail        string
		presetBlock   *lib.Block
		presetGenesis *types.GenesisState
		height        uint64
		expected      *types.GenesisState
	}{
		{
			name:        "after genesis",
			detail:      "the block height is after 0, thus it's the non-genesis initialization",
			height:      2,
			presetBlock: &lib.Block{BlockHeader: &lib.BlockHeader{Height: 1, Hash: crypto.Hash([]byte("test")), TotalVdfIterations: 2}},
		},
		{
			name:          "genesis path",
			detail:        "the height is 0 so the genesis path is taken",
			presetGenesis: newTestGenesisState(t),
			expected:      newTestValidateGenesisState(t),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create the default logger
			log := lib.NewDefaultLogger()
			// create an in-memory store
			db, err := store.NewStoreInMemory(log)
			require.NoError(t, err)
			if test.presetGenesis != nil {
				// marshal genesis file to bytes
				genesisJsonBytes, e := json.MarshalIndent(&test.presetGenesis, "", "  ")
				require.NoError(t, e)
				// write test genesis to file
				require.NoError(t, os.WriteFile("genesis.json", genesisJsonBytes, 0777))
				// remove the test file
				defer os.RemoveAll("genesis.json")
			}
			if test.presetBlock != nil {
				// set the block in state
				require.NoError(t, db.IndexBlock(&lib.BlockResult{
					BlockHeader: test.presetBlock.BlockHeader,
				}))
			}
			if test.height != 0 {
				// increment the db version
				_, _ = db.Commit()
			}
			// create a state machine object
			sm := StateMachine{
				store:  db,
				height: test.height,
				Config: lib.Config{},
				log:    log,
			}
			// set the data dir path
			sm.Config.DataDirPath = dataDirPath
			// execute the function call
			require.NoError(t, sm.Initialize(db))
			// validate the initialization path
			if sm.height == 0 {
				// if genesis, validate the state
				validateWithExportedState(t, sm, test.expected)
			} else {
				// if not genesis, validate the VDF iterations
				require.Equal(t, test.presetBlock.BlockHeader.TotalVdfIterations, sm.vdfIterations)
			}
		})
	}
}

func TestApplyBlock(t *testing.T) {
	// define a key group to use in testing
	kg := newTestKeyGroup(t)
	// predefine a send-transaction to insert into the block
	sendTx, err := types.NewSendTransaction(kg.PrivateKey, newTestAddress(t), 1, 1)
	// ensure no error
	require.NoError(t, err)
	// convert the object to bytes
	sendTxBytes, err := lib.Marshal(sendTx)
	// ensure no error
	require.NoError(t, err)
	// define test cases
	tests := []struct {
		name            string
		detail          string
		accountPreset   uint64
		storeError      bool
		beginBlockError bool
		block           *lib.Block
		expectedHeader  *lib.BlockHeader
		expectedResults *lib.TxResults
		error           string
	}{
		{
			name:       "store error",
			detail:     "an error occurred in casting the store to lib.Store",
			storeError: true,
			error:      "wrong store type",
		},
		{
			name:            "begin_block error",
			detail:          "an error occurred in begin block",
			block:           &lib.Block{BlockHeader: &lib.BlockHeader{}, Transactions: [][]byte{sendTxBytes}},
			beginBlockError: true,
			error:           "invalid protocol version",
		},
		{
			name:   "transaction error",
			detail: "an error occurred in the transaction",
			block:  &lib.Block{BlockHeader: &lib.BlockHeader{}, Transactions: [][]byte{sendTxBytes}},
			error:  "insufficient funds",
		},
		{
			name:          "",
			detail:        "",
			accountPreset: 2,
			block: &lib.Block{
				BlockHeader: &lib.BlockHeader{
					Height:                1,
					NumTxs:                1,
					TotalTxs:              1,
					LastBlockHash:         crypto.Hash([]byte("block_hash")),
					StateRoot:             nil,
					TransactionRoot:       nil,
					ValidatorRoot:         nil,
					NextValidatorRoot:     nil,
					ProposerAddress:       newTestAddressBytes(t),
					Vdf:                   nil,
					LastQuorumCertificate: nil,
				},
				Transactions: [][]byte{sendTxBytes},
			},
			expectedHeader: &lib.BlockHeader{
				Height:                0,
				Hash:                  nil,
				NetworkId:             0,
				Time:                  0,
				NumTxs:                0,
				TotalTxs:              0,
				TotalVdfIterations:    0,
				LastBlockHash:         nil,
				StateRoot:             nil,
				TransactionRoot:       nil,
				ValidatorRoot:         nil,
				NextValidatorRoot:     nil,
				ProposerAddress:       nil,
				Vdf:                   nil,
				LastQuorumCertificate: nil,
			},
			expectedResults: &lib.TxResults{
				{
					Sender:      newTestAddressBytes(t),
					Recipient:   newTestAddressBytes(t),
					MessageType: "send",
					Height:      1,
					Index:       0,
					Transaction: sendTx.(*lib.Transaction),
					TxHash:      crypto.HashString(sendTxBytes),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			sm.height = 1
			if test.storeError {
				// set the store to the wrong type
				sm.store = lib.RWStoreI(nil)
			} else {
				// preset the 'last block' in state
				require.NoError(t, sm.store.(lib.StoreI).IndexBlock(&lib.BlockResult{
					BlockHeader: &lib.BlockHeader{
						Height: 1,
						Hash:   test.block.BlockHeader.LastBlockHash,
						Time:   uint64(time.Now().UnixMicro()),
					},
				}))
			}
			if !test.beginBlockError {
				// set the protocol version to trigger an error
				sm.ProtocolVersion = 1
			}
			// set the minimum fee to 1 for send transactions
			require.NoError(t, sm.UpdateParam("fee", types.ParamMessageSendFee, &lib.UInt64Wrapper{Value: 1}))
			// preset the account with funds
			require.NoError(t, sm.AccountAdd(newTestAddress(t), test.accountPreset))
			// preset a committee member for canopy
			require.NoError(t, sm.SetValidators([]*types.Validator{
				{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 1,
					Committees:   []uint64{lib.CanopyCommitteeId},
				},
			}, &types.Supply{}))
			// execute the function call
			header, txResults, e := sm.ApplyBlock(test.block)
			// validate the expected error
			require.Equal(t, test.error != "", e != nil, e)
			if e != nil {
				require.ErrorContains(t, e, test.error)
				return
			}
			// validate got vs expected block header
			require.EqualExportedValues(t, test.expectedHeader, header)
			// validate got vs expected tx results
			require.EqualExportedValues(t, test.expectedResults, txResults)
		})
	}
}

func newSingleAccountStateMachine(t *testing.T) StateMachine {
	sm := newTestStateMachine(t)
	keyGroup := newTestKeyGroup(t)
	require.NoError(t, sm.SetParams(types.DefaultParams()))
	require.NoError(t, sm.SetAccount(&types.Account{
		Address: keyGroup.Address.Bytes(),
		Amount:  1000000,
	}))
	require.NoError(t, sm.HandleMessageStake(&types.MessageStake{
		PublicKey:     keyGroup.PublicKey.Bytes(),
		Amount:        1000000,
		Committees:    []uint64{lib.CanopyCommitteeId},
		NetAddress:    "http://localhost:80",
		OutputAddress: keyGroup.Address.Bytes(),
		Delegate:      false,
		Compound:      true,
	}))
	require.NoError(t, sm.SetParams(types.DefaultParams()))
	return sm
}

func newTestStateMachine(t *testing.T) StateMachine {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	sm := StateMachine{
		store:             db,
		ProtocolVersion:   0,
		NetworkID:         0,
		height:            2,
		vdfIterations:     0,
		slashTracker:      types.NewSlashTracker(),
		proposeVoteConfig: types.AcceptAllProposals,
		Config:            lib.Config{},
		log:               log,
	}
	require.NoError(t, sm.SetParams(types.DefaultParams()))
	db.Commit()
	require.NoError(t, sm.SetParams(types.DefaultParams()))
	return sm
}

func newTestAddress(t *testing.T, variation ...int) crypto.AddressI {
	kg := newTestKeyGroup(t, variation...)
	return kg.Address
}

func newTestAddressBytes(t *testing.T, variation ...int) []byte {
	return newTestAddress(t, variation...).Bytes()
}

func newTestPublicKey(t *testing.T, variation ...int) crypto.PublicKeyI {
	kg := newTestKeyGroup(t, variation...)
	return kg.PublicKey
}

func newTestPublicKeyBytes(t *testing.T, variation ...int) []byte {
	return newTestPublicKey(t, variation...).Bytes()
}

func newTestKeyGroup(t *testing.T, variation ...int) *crypto.KeyGroup {
	var (
		key  crypto.PrivateKeyI
		err  error
		keys = []string{
			"01553a101301cd7019b78ffa1186842dd93923e563b8ae22e2ab33ae889b23ee",
			"1b6b244fbdf614acb5f0d00a2b56ffcbe2aa23dabd66365dffcd3f06491ae50a",
			"2ee868f74134032eacba191ca529115c64aa849ac121b75ca79b37420a623036",
			"3e3ab94c10159d63a12cb26aca4b0e76070a987d49dd10fc5f526031e05801da",
			"479839d3edbd0eefa60111db569ded6a1a642cc84781600f0594bd8d4a429319",
			"51eb5eb6eca0b47c8383652a6043aadc66ddbcbe240474d152f4d9a7439eae42",
			"637cb8e916bba4c1773ed34d89ebc4cb86e85c145aea5653a58de930590a2aa4",
			"7235e5757e6f52e6ae4f9e20726d9c514281e58e839e33a7f667167c524ff658"}
	)

	if len(variation) == 1 {
		key, err = crypto.NewBLSPrivateKeyFromString(keys[variation[0]])
	} else {
		key, err = crypto.NewBLSPrivateKeyFromString(keys[0])
	}
	require.NoError(t, err)
	return crypto.NewKeyGroup(key)
}

func newTestKeyGroups(t *testing.T, count int) (groups []*crypto.KeyGroup) {
	for i := 0; i < count; i++ {
		groups = append(groups, newTestKeyGroup(t, i))
	}
	return
}

// testQCParams are the associate parameters needed to generate a testQC
type testQCParams struct {
	height        uint64
	idxSigned     map[int]bool
	committeeKeys []*crypto.KeyGroup
	committee     []*types.Validator
	results       *lib.CertificateResult
}

// newTestQC is a utility function for this test to generate various quorum certificates in the test cases
func newTestQC(t *testing.T, params testQCParams) (qc *lib.QuorumCertificate) {
	// convert committee members to consensus validators
	var vals []*lib.ConsensusValidator
	for _, m := range params.committee {
		vals = append(vals, &lib.ConsensusValidator{PublicKey: m.PublicKey, VotingPower: m.StakedAmount})
	}
	// create a validator set object in order to generate a multi-public key for the set
	validatorSet, err := lib.NewValidatorSet(&lib.ConsensusValidators{ValidatorSet: vals})
	require.NoError(t, err)
	// create the 'justification' object
	justification := validatorSet.MultiKey.Copy()
	// create the certificate results object to put in the QC
	// create the QC object
	qc = &lib.QuorumCertificate{
		Header: &lib.View{
			Height:          params.height,
			CommitteeHeight: params.height,
			CommitteeId:     lib.CanopyCommitteeId,
		},
		Results:     params.results,
		ResultsHash: params.results.Hash(),
		BlockHash:   crypto.Hash([]byte("some block that's not included here")),
	}
	// generate the bytes to be signed in the justification (multi-key)
	bytesToBeSigned := qc.SignBytes()
	// have the 'signers' sign the justification (multi-key)
	for i, s := range params.committeeKeys {
		if params.idxSigned[i] {
			require.NoError(t, justification.AddSigner(s.PrivateKey.Sign(bytesToBeSigned), i))
		}
	}
	// aggregate the signature
	aggregateSignatures, e := justification.AggregateSignatures()
	require.NoError(t, e)
	// wrap in object
	qc.Signature = &lib.AggregateSignature{
		Signature: aggregateSignatures,
		Bitmap:    justification.Bitmap(),
	}
	return
}