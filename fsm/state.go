package fsm

import (
	"runtime/debug"
	"strings"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

const (
	CurrentProtocolVersion = 1
)

/* This is the 'main' file of the state machine store, with the structure definition and other high level operations */

// StateMachine the core protocol component responsible for maintaining and updating the state of the blockchain as it progresses
// it represents the collective state of all accounts, validators, and other relevant data stored on the blockchain
type StateMachine struct {
	store lib.RWStoreI

	ProtocolVersion    uint64                // the version of the protocol this node is running
	NetworkID          uint32                // the id of the network this node is configured to be on
	height             uint64                // the 'version' of the state based on number of blocks currently on
	totalVDFIterations uint64                // the number of 'verifiable delay iterations' in the blockchain up to this version
	slashTracker       *SlashTracker         // tracks total slashes across multiple blocks
	proposeVoteConfig  GovProposalVoteConfig // the configuration of how the state machine behaves with governance proposals
	valsCache          map[string]*Validator // cache the validator structure
	Config             lib.Config            // the main configuration as defined by the 'config.json' file
	log                lib.LoggerI           // the logger for standard output and debugging
}

// New() creates a new instance of a StateMachine
func New(c lib.Config, store lib.StoreI, log lib.LoggerI) (*StateMachine, lib.ErrorI) {
	// create the state machine object reference
	sm := &StateMachine{
		store:             nil,
		ProtocolVersion:   CurrentProtocolVersion,
		NetworkID:         uint32(c.P2PConfig.NetworkID),
		slashTracker:      NewSlashTracker(),
		proposeVoteConfig: AcceptAllProposals,
		valsCache:         make(map[string]*Validator),
		Config:            c,
		log:               log,
	}
	// initialize the state machine and exit
	return sm, sm.Initialize(store)
}

// Initialize() initializes a StateMachine object using the StoreI
func (s *StateMachine) Initialize(store lib.StoreI) (err lib.ErrorI) {
	// set height to the latest version and store to the passed store
	s.height, s.store = store.Version(), store
	// if height is genesis
	if s.height == 0 {
		// then initialize from a genesis file
		return s.NewFromGenesisFile()
	}
	// load the previous block
	blk, e := s.LoadBlock(s.Height() - 1)
	if e != nil {
		return e
	}
	// set totalVDFIterations in the state machine
	s.totalVDFIterations = blk.BlockHeader.TotalVdfIterations
	return
}

// ApplyBlock processes a given block, updating the state machine's state accordingly
// The function:
// - executes `BeginBlock`
// - applies all transactions within the block, generating transaction results nad a root hash
// - executes `EndBlock`
// - constructs and returns the block header, and the transaction results
func (s *StateMachine) ApplyBlock(b *lib.Block) (header *lib.BlockHeader, txResults []*lib.TxResult, err lib.ErrorI) {
	// catch in case there's a panic
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf(string(debug.Stack()))
			// handle the panic and set the error
			err = lib.ErrPanic()
		}
	}()
	// cast the store to a StoreI, as only the writable store main 'apply blocks'
	store, ok := s.Store().(lib.StoreI)
	// casting fails, exit with error
	if !ok {
		return nil, nil, ErrWrongStoreType()
	}
	// automated execution at the 'beginning of a block'
	if err = s.BeginBlock(); err != nil {
		return nil, nil, err
	}
	// apply all Transactions in the block
	txResults, txRoot, numTxs, err := s.ApplyTransactions(b)
	if err != nil {
		return nil, nil, err
	}
	// automated execution at the 'ending of a block'
	if err = s.EndBlock(b.BlockHeader.ProposerAddress); err != nil {
		return nil, nil, err
	}
	// load the validator set for the previous height
	lastValidatorSet, _ := s.LoadCommittee(s.Config.ChainId, s.Height()-1)
	// calculate the merkle root of the last validators to maintain validator continuity between blocks (if root)
	lastValidatorRoot, err := lastValidatorSet.ValidatorSet.Root()
	if err != nil {
		return nil, nil, err
	}
	// load the 'next validator set' from the state
	nextValidatorSet, _ := s.LoadCommittee(s.Config.ChainId, s.Height())
	// calculate the merkle root of the next validators to maintain validator continuity between blocks (if root)
	nextValidatorRoot, err := nextValidatorSet.ValidatorSet.Root()
	if err != nil {
		return nil, nil, err
	}
	// calculate the merkle root of the state database to enable consensus on the result of the state after applying the block
	stateRoot, err := store.Root()
	if err != nil {
		return nil, nil, err
	}
	// load the last block from the indexer
	lastBlock, err := s.LoadBlock(s.height - 1)
	if err != nil {
		return nil, nil, err
	}
	// generate the block header
	header = &lib.BlockHeader{
		Height:                s.Height(),                                                                   // increment the height
		Hash:                  nil,                                                                          // set hash after
		NetworkId:             s.NetworkID,                                                                  // ensure only applicable for the proper network
		Time:                  b.BlockHeader.Time,                                                           // use the pre-set block time
		NumTxs:                uint64(numTxs),                                                               // set the number of transactions
		TotalTxs:              lastBlock.BlockHeader.TotalTxs + uint64(numTxs),                              // set the total count of transactions
		TotalVdfIterations:    lastBlock.BlockHeader.TotalVdfIterations + b.BlockHeader.Vdf.GetIterations(), // add last total iterations to current iterations
		StateRoot:             stateRoot,                                                                    // set the state root generated from the resulting state of the VDF
		LastBlockHash:         nonEmptyHash(lastBlock.BlockHeader.Hash),                                     // set the last block hash to chain the blocks together
		TransactionRoot:       nonEmptyHash(txRoot),                                                         // set the transaction root to easily merkle the transactions in a block
		ValidatorRoot:         nonEmptyHash(lastValidatorRoot),                                              // set the last validator root to easily prove the validators who voted on this block
		NextValidatorRoot:     nonEmptyHash(nextValidatorRoot),                                              // set the next validator root to have continuity between validator sets
		ProposerAddress:       b.BlockHeader.ProposerAddress,                                                // set the proposer address
		Vdf:                   b.BlockHeader.Vdf,                                                            // attach the preset vdf proof
		LastQuorumCertificate: b.BlockHeader.LastQuorumCertificate,                                          // attach last quorum certificate (which is validated in the 'compare block headers' func
	}
	// create and set the block hash in the header
	if _, err = header.SetHash(); err != nil {
		return nil, nil, err
	}
	// exit
	return
}

// ApplyTransactions processes all transactions in the provided block and updates the state accordingly
func (s *StateMachine) ApplyTransactions(block *lib.Block) (txResultsList []*lib.TxResult, root []byte, n int, er lib.ErrorI) {
	// define vars to track the bytes of the transaction results and the size of a block
	var (
		txResultsBytes [][]byte
		blockSize      uint64
	)
	// use a map to check for 'same-block' duplicate transactions
	deDuplicator := lib.NewDeDuplicator[string]()
	// iterates over each transaction in the block
	for index, tx := range block.Transactions {
		// calculate the hash of the transaction and convert it to a hex string
		hashString := crypto.HashString(tx)
		// check if the transaction is a 'same block' duplicate
		if found := deDuplicator.Found(hashString); found {
			return txResultsList, root, n, lib.ErrDuplicateTx(hashString)
		}
		// apply the tx to the state machine, generating a transaction result
		result, err := s.ApplyTransaction(uint64(index), tx, hashString)
		if err != nil {
			return nil, nil, 0, err
		}
		// encode the result to bytes
		txResultBz, err := lib.Marshal(result)
		if err != nil {
			return nil, nil, 0, err
		}
		// add the result to a list of transaction results
		txResultsList = append(txResultsList, result)
		// add the bytes to the list of transactions results
		txResultsBytes = append(txResultsBytes, txResultBz)
		// add to the size of the block
		blockSize += uint64(len(tx))
		// update the transaction count
		n++
	}
	// get the governance parameter for max block size
	maxBlockSize, err := s.GetMaxBlockSize()
	if err != nil {
		return nil, nil, 0, err
	}
	// ensure the block size + max block header size is less than the state defined 'MaxBlockSize'
	if blockSize+lib.MaxBlockHeaderSize > maxBlockSize {
		return nil, nil, 0, ErrMaxBlockSize()
	}
	// create a transaction root for the block header
	root, _, err = lib.MerkleTree(txResultsBytes)
	// return and exit
	return txResultsList, root, n, err
}

// TimeMachine() creates a new StateMachine instance representing the blockchain state at a specified block height, allowing for a read-only view of the past state
func (s *StateMachine) TimeMachine(height uint64) (*StateMachine, lib.ErrorI) {
	// if height is zero, use the 'latest' height
	if height == 0 || height > s.height {
		height = s.height
	}
	// don't try to create a NewReadOnly with height 0 as it'll panic
	if height == 0 {
		// return the original state machine
		return s, nil
	}
	// ensure the store is the proper type to allow historical views
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, ErrWrongStoreType()
	}
	// create a NewReadOnly store at the specific height
	heightStore, err := store.NewReadOnly(height)
	if err != nil {
		return nil, err
	}
	// initialize a new state machine
	return New(s.Config, heightStore, s.log)
}

// LoadCommittee() loads the committee validators for a particular committee at a particular height
func (s *StateMachine) LoadCommittee(chainId uint64, height uint64) (lib.ValidatorSet, lib.ErrorI) {
	// get the historical state at the height
	historicalFSM, err := s.TimeMachine(height)
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	// memory management for the historical FSM call
	defer historicalFSM.Discard()
	// return the 'committee members' (validator set) for that height
	return historicalFSM.GetCommitteeMembers(chainId)
}

// LoadCertificate() loads a quorum certificate (block, results + 2/3rd committee signatures)
func (s *StateMachine) LoadCertificate(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	// ensure the 'load height' is not genesis
	if height <= 1 {
		height = 1
	}
	// ensure the store is the proper type to allow indexer actions
	store, ok := s.store.(lib.RIndexerI)
	if !ok {
		return nil, ErrWrongStoreType()
	}
	// load the quorum certificate by height
	return store.GetQCByHeight(height)
}

// LoadCertificateHashesOnly() loads a quorum certificate but nullifies the block
func (s *StateMachine) LoadCertificateHashesOnly(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	// ensure the 'load height' is not genesis
	if height <= 1 {
		height = 1
	}
	// load the quorum certificate at a specific height
	qc, err := s.LoadCertificate(height)
	if err != nil {
		return nil, err
	}
	// nullify the block
	qc.Block = nil
	// return the quorum certificate
	return qc, nil
}

// LoadBlock() loads an indexed block at a specific height
func (s *StateMachine) LoadBlock(height uint64) (*lib.BlockResult, lib.ErrorI) {
	// ensure the 'load height' is not genesis
	if height <= 1 {
		height = 1
	}
	// ensure the store is the proper type to allow indexer actions
	store, ok := s.store.(lib.RIndexerI)
	if !ok {
		return nil, ErrWrongStoreType()
	}
	// get the block result from the indexer at the 'load height'
	return store.GetBlockByHeight(height)
}

// LoadBlock() loads an indexed block at a specific height
func (s *StateMachine) LoadBlockAndCertificate(height uint64) (cert *lib.QuorumCertificate, block *lib.BlockResult, err lib.ErrorI) {
	// ensure the 'load height' is not genesis
	if height <= 1 {
		height = 1
	}
	// ensure the store is the proper type to allow indexer actions
	store, ok := s.store.(lib.RIndexerI)
	if !ok {
		return nil, nil, ErrWrongStoreType()
	}
	// get the block result at a specific height
	block, err = store.GetBlockByHeight(height)
	if err != nil {
		return nil, nil, err
	}
	// load the quorum certificate from the indexer
	cert, err = s.LoadCertificateHashesOnly(height)
	// exit
	return
}

// GetMaxValidators() returns the max validators per committee
func (s *StateMachine) GetMaxValidators() (uint64, lib.ErrorI) {
	// get the parameters for the validator space from state
	valParams, err := s.GetParamsVal()
	if err != nil {
		return 0, err
	}
	// return the max committee size
	return valParams.MaxCommitteeSize, nil
}

// GetMaxBlockSize() returns the maximum size of a block
func (s *StateMachine) GetMaxBlockSize() (uint64, lib.ErrorI) {
	// get the parameters for the consensus space from state
	consParams, err := s.GetParamsCons()
	if err != nil {
		return 0, err
	}
	// return the max block size
	return consParams.BlockSize, nil
}

// Copy() makes a clone of the state machine
// this feature is used in mempool operation to be able to maintain a parallel ephemeral state without affecting the underlying state machine
func (s *StateMachine) Copy() (*StateMachine, lib.ErrorI) {
	// ensure the store is the right type to 'clone' itself
	st, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, ErrWrongStoreType()
	}
	// make a clone of the store
	storeCopy, err := st.Copy()
	if err != nil {
		return nil, err
	}
	// return the clone state machine object reference
	return &StateMachine{
		store:              storeCopy,
		ProtocolVersion:    s.ProtocolVersion,
		NetworkID:          s.NetworkID,
		height:             s.height,
		totalVDFIterations: s.totalVDFIterations,
		slashTracker:       NewSlashTracker(),
		proposeVoteConfig:  s.proposeVoteConfig,
		Config:             s.Config,
		log:                s.log,
	}, nil
}

// ResetToBeginBlock() resets the store and executes 'begin-block'
func (s *StateMachine) ResetToBeginBlock() {
	// reset the state machine (store)
	s.Reset()
	// run the 'begin block' code
	if err := s.BeginBlock(); err != nil {
		s.log.Errorf("BEGIN_BLOCK FAILURE: %s", err.Error())
	}
}

// Set() upserts a key-value pair under a key
func (s *StateMachine) Set(k, v []byte) (err lib.ErrorI) { return s.Store().Set(k, v) }

// Get() retrieves a key-value pair under a key
// NOTE: returns (nil, nil) if no value is found for that key
func (s *StateMachine) Get(key []byte) (bz []byte, err lib.ErrorI) { return s.Store().Get(key) }

// Delete() deletes a key-value pair under a key
func (s *StateMachine) Delete(key []byte) lib.ErrorI { return s.Store().Delete(key) }

// Iterator() creates and returns an iterator for the state machine's underlying store
// starting at the specified key and iterating lexicographically
func (s *StateMachine) Iterator(key []byte) (lib.IteratorI, lib.ErrorI) {
	return s.Store().Iterator(key)
}

// RevIterator() creates and returns an iterator for the state machine's underlying store
// starting at the end-prefix of the specified key and iterating reverse lexicographically
func (s *StateMachine) RevIterator(key []byte) (lib.IteratorI, lib.ErrorI) {
	return s.Store().RevIterator(key)
}

// DeleteAll() deletes all key-value pairs under a set of keys
func (s *StateMachine) DeleteAll(keys [][]byte) (err lib.ErrorI) {
	// for each key in the key list
	for _, key := range keys {
		// delete the key
		if err = s.Delete(key); err != nil {
			// if err then exit
			return
		}
	}
	// exit
	return
}

// IterateAndExecute() creates an iterator and executes a callback function for each key-value pair
func (s *StateMachine) IterateAndExecute(prefix []byte, callback func(key, value []byte) lib.ErrorI) (err lib.ErrorI) {
	// create an iterator for the prefix
	it, err := s.Iterator(prefix)
	if err != nil {
		return err
	}
	// ensure it's cleaned up
	defer it.Close()
	// for each value in the iterator
	for ; it.Valid(); it.Next() {
		// execute the callback
		if err = callback(it.Key(), it.Value()); err != nil {
			// if err then exit
			return
		}
	}
	// exit
	return
}

// TxnWrap() is an atomicity and consistency feature that enables easy rollback of changes by discarding the transaction if an error occurs
func (s *StateMachine) TxnWrap() (lib.StoreTxnI, lib.ErrorI) {
	// ensure the store may be 'cache wrapped' in a 'database transaction'
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, ErrWrongStoreType()
	}
	// create a new 'database transaction'
	txn := store.NewTxn()
	// set the store as that transaction
	s.SetStore(txn)
	// return the transaction to be cleaned up by the caller
	return txn, nil
}

// catchPanic() acts as a failsafe, recovering from a panic and logging the error with the stack trace
func (s *StateMachine) catchPanic() {
	if r := recover(); r != nil {
		s.log.Error(string(debug.Stack()))
	}
}

// Reset() resets the state store and the slash tracker
func (s *StateMachine) Reset() {
	// reset the slash tracker
	s.slashTracker = NewSlashTracker()
	// reset the slash tracker
	s.valsCache = make(map[string]*Validator)
	// reset the state store
	s.store.(lib.StoreI).Reset()
}

// nonEmptyHash() ensures the hash isn't empty
// substituting a dummy hash in its place
func nonEmptyHash(h []byte) []byte {
	if len(h) == 0 {
		h = []byte(strings.Repeat("F", crypto.HashSize))
	}
	return h
}

// various self-explanatory 1 line functions below
func (s *StateMachine) Store() lib.RWStoreI                           { return s.store }
func (s *StateMachine) SetStore(store lib.RWStoreI)                   { s.store = store }
func (s *StateMachine) Height() uint64                                { return s.height }
func (s *StateMachine) TotalVDFIterations() uint64                    { return s.totalVDFIterations }
func (s *StateMachine) Discard()                                      { s.store.(lib.StoreI).Discard() }
func (s *StateMachine) SetProposalVoteConfig(c GovProposalVoteConfig) { s.proposeVoteConfig = c }
