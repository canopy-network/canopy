package fsm

import (
	"context"
	"math"
	"runtime/debug"
	"strings"
	"time"

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

	ProtocolVersion    uint64                                  // the version of the protocol this node is running
	NetworkID          uint32                                  // the id of the network this node is configured to be on
	height             uint64                                  // the 'version' of the state based on number of blocks currently on
	totalVDFIterations uint64                                  // the number of 'verifiable delay iterations' in the blockchain up to this version
	slashTracker       *SlashTracker                           // tracks total slashes across multiple blocks
	proposeVoteConfig  GovProposalVoteConfig                   // the configuration of how the state machine behaves with governance proposals
	Config             lib.Config                              // the main configuration as defined by the 'config.json' file
	Metrics            *lib.Metrics                            // the telemetry module
	events             *lib.EventsTracker                      // a simple event tracker for 'per-transaction' events
	Plugin             *lib.Plugin                             // extensible plugin for the FSM
	log                lib.LoggerI                             // the logger for standard output and debugging
	cache              *cache                                  // the state machine cache
	LastValidatorSet   map[uint64]map[uint64]*lib.ValidatorSet // reference to the last validator set saved in the controller
}

// cache is the set of items to be cached used by the state machine
type cache struct {
	accounts  map[uint64]*Account // cache of accounts accessed
	feeParams *FeeParams          // fee params for the current block
	valParams *ValidatorParams    // validator params for the current block
}

// New() creates a new instance of a StateMachine
func New(c lib.Config, store lib.StoreI, plugin *lib.Plugin, metrics *lib.Metrics, log lib.LoggerI) (*StateMachine, lib.ErrorI) {
	// create the state machine object reference
	sm := &StateMachine{
		store:             nil,
		ProtocolVersion:   CurrentProtocolVersion,
		NetworkID:         uint32(c.P2PConfig.NetworkID),
		slashTracker:      NewSlashTracker(),
		proposeVoteConfig: AcceptAllProposals,
		Config:            c,
		Plugin:            plugin,
		Metrics:           metrics,
		log:               log,
		events:            new(lib.EventsTracker),
		cache: &cache{
			accounts: make(map[uint64]*Account),
		},
	}
	// initialize the state machine
	genesis, err := sm.Initialize(store)
	if err != nil {
		return nil, err
	}
	// if genesis - reset the store
	if genesis {
		sm.Reset()
	}
	// initialize the state machine and exit
	return sm, nil
}

// Initialize() initializes a StateMachine object using the StoreI
func (s *StateMachine) Initialize(store lib.StoreI) (genesis bool, err lib.ErrorI) {
	// set height to the latest version and store to the passed store
	s.height, s.store = store.Version(), store
	// if height is genesis
	if s.height == 0 {
		// then initialize from a genesis file
		return true, s.NewFromGenesisFile()
	}
	// load the previous block
	blk, e := s.LoadBlock(s.Height() - 1)
	if e != nil {
		return false, e
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
// NOTES:
// - this function may be used to validate 'additional' transactions outside the normal block size as if they were to be included
// - a list of failed transactions are returned
func (s *StateMachine) ApplyBlock(ctx context.Context, b *lib.Block, allowOversize bool) (header *lib.BlockHeader, r *lib.ApplyBlockResults, err lib.ErrorI) {
	// catch in case there's a panic
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("panic recovered, err: %s, stack: %s", r, string(debug.Stack()))
			// handle the panic and set the error
			err = lib.ErrPanic()
		}
	}()
	// define vars to track the bytes of the transaction results and the size of a block
	r = new(lib.ApplyBlockResults)
	// cast the store to a StoreI, as only the writable store main 'apply blocks'
	store, ok := s.Store().(lib.StoreI)
	// casting fails, exit with error
	if !ok {
		return nil, nil, ErrWrongStoreType()
	}
	// automated execution at the 'beginning of a block'
	events, err := s.BeginBlock()
	if err != nil {
		return nil, nil, err
	}
	// add the events from begin block
	r.AddEvent(events...)
	// apply all Transactions in the block
	if err = s.ApplyTransactions(ctx, b.Transactions, r, allowOversize); err != nil {
		return nil, nil, err
	}
	// sub-out transactions for those that succeeded (only useful for mempool application)
	b.Transactions = r.Txs
	// automated execution at the 'ending of a block'
	events, err = s.EndBlock(b.BlockHeader.ProposerAddress)
	if err != nil {
		return nil, nil, err
	}
	// add the events from end block
	r.AddEvent(events...)
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
	// get the transaction root
	transactionRoot, err := r.TransactionRoot()
	if err != nil {
		return nil, nil, err
	}
	// generate the block header
	header = &lib.BlockHeader{
		Height:                s.Height(),                                                                   // increment the height
		Hash:                  nil,                                                                          // set hash after
		NetworkId:             s.NetworkID,                                                                  // ensure only applicable for the proper network
		Time:                  b.BlockHeader.Time,                                                           // use the pre-set block time
		NumTxs:                uint64(r.Count),                                                              // set the number of transactions
		TotalTxs:              lastBlock.BlockHeader.TotalTxs + uint64(r.Count),                             // set the total count of transactions
		TotalVdfIterations:    lastBlock.BlockHeader.TotalVdfIterations + b.BlockHeader.Vdf.GetIterations(), // add last total iterations to current iterations
		StateRoot:             stateRoot,                                                                    // set the state root generated from the resulting state of the VDF
		LastBlockHash:         nonEmptyHash(lastBlock.BlockHeader.Hash),                                     // set the last block hash to chain the blocks together
		TransactionRoot:       nonEmptyHash(transactionRoot),                                                // set the transaction root to easily merkle the transactions in a block
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

// ApplyTransactions()
// 1. Batch validate signatures for every transaction provided
// 2. Processes all transactions provided against the state machine
// 3. Allows ephemeral 'oversize' transaction processing without applying 'oversize txn' changes to the state
// 4. Returns the following for successful transactions within a block: <results, tx-list, root, count>
// 5. Returns all transactions that failed during processing
func (s *StateMachine) ApplyTransactions(ctx context.Context, txs [][]byte, r *lib.ApplyBlockResults, allowOversize bool) lib.ErrorI {
	// use a map to check for 'same-block' duplicate transactions
	deDuplicator := lib.NewDeDuplicator[string]()
	// use a batch verifier for signatures
	batchVerifier := crypto.NewBatchVerifier()
	// get the governance parameter for max block size
	maxBlockSize, err := s.GetMaxBlockSize()
	if err != nil {
		return err
	}
	// keep a map to track transactions that failed 'check'
	failedCheckTxs := map[int]error{}
	// first batch validate signatures over the entire set
	for i, tx := range txs {
		if _, err = s.CheckTx(tx, "", batchVerifier); err != nil {
			failedCheckTxs[i] = err
		}
	}
	// execute batch verification of the signatures in the block
	for _, failedIdx := range batchVerifier.Verify() {
		failedCheckTxs[failedIdx] = ErrInvalidSignature()
	}
	// set the store back to the original at the end of processing
	originalStore := s.Store().(lib.StoreI)
	defer s.SetStore(originalStore)
	// create a variable to track if the block is over size
	var oversize bool
	// iterates over each transaction in the block
	for i, tx := range txs {
		// if interrupt signal
		if ctx.Err() != nil {
			return lib.ErrMempoolStopSignal()
		}
		// if already failed check tx or signature
		if e, found := failedCheckTxs[i]; found {
			r.AddFailed(lib.NewFailedTx(tx, e))
			continue
		}
		// calculate the hash of the transaction and convert it to a hex string
		hashString := crypto.HashString(tx)
		// check if the transaction is a 'same block' duplicate
		if found := deDuplicator.Found(hashString); found {
			return lib.ErrDuplicateTx(hashString)
		}
		// get the tx size
		txSize := uint64(len(tx))
		// if the max block size is exceeded and we're not yet marked as 'oversize'
		if txSize+r.BlockSize > maxBlockSize && !oversize {
			// if validating a block - oversize shouldn't happen
			if !allowOversize {
				return ErrMaxBlockSize()
			}
			// set oversize to 'true'
			oversize = true
			// wrap the store in a 'database transaction' to rollback all the 'oversize transactions'
			if _, e := s.TxnWrap(); e != nil {
				return e
			}
		}
		// get the store from the state machine, it may be the original or a wrapped 'txn' if processing oversize transactions
		currentStore := s.Store().(lib.StoreI)
		// snapshot trackers that must not persist across failed transactions
		preTxSlashTracker := s.slashTracker.Clone()
		// wrap the store in a 'database transaction' in case a rollback to the previous valid transaction is needed
		txn, e := s.TxnWrap()
		if e != nil {
			return e
		}
		// apply the tx to the state machine, generating a transaction result
		result, events, e := s.ApplyTransaction(uint64(r.Count), tx, hashString, crypto.NewBatchVerifier(true))
		if e != nil {
			// add to the failed list
			r.AddFailed(lib.NewFailedTx(tx, e))
			// discard the FSM cache
			s.ResetCaches()
			// clear any events accumulated for the failed transaction to avoid leaking them to subsequent txs
			s.events.Reset()
			// restore slash tracker to its pre-transaction state
			s.slashTracker = preTxSlashTracker
			//txn.Discard()
			s.SetStore(currentStore)
			continue
		} else {
			// write the transaction to the underlying store
			if err = txn.Flush(); err != nil {
				return err
			}
			s.SetStore(currentStore)
		}
		// encode the result to bytes
		txResultBz, e := lib.Marshal(result)
		if e != nil {
			return e
		}
		r.Add(tx, txResultBz, result, events, oversize)
	}
	// update metrics
	s.Metrics.UpdateLargestTxSize(r.LargestTx)
	// return and exit
	return err
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
	return New(s.Config, heightStore, s.Plugin, s.Metrics, s.log)
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
	return consParams.BlockSize - lib.MaxBlockHeaderSize, nil
}

// LoadRootChainInfo() returns the 'need-to-know' information for a nested chain
func (s *StateMachine) LoadRootChainInfo(id, height uint64) (*lib.RootChainInfo, lib.ErrorI) {
	lastHeight := uint64(1)
	// update the metrics once complete
	defer s.Metrics.UpdateGetRootChainInfo(time.Now())
	// if height is 0; use the latest height
	if height == 0 {
		height = s.height
	}
	// ensure lastHeight is not < 0
	if height != 1 {
		lastHeight = height - 1
	}
	// get the latest state machine
	sm, err := s.TimeMachine(height)
	if err != nil {
		return nil, err
	}
	// if height is equal to latest height, provide the validator cache to the FSM
	if height == s.height {
		sm.cache = s.cache
	}
	// get the previous state machine height
	lastSM, err := s.TimeMachine(lastHeight)
	if err != nil {
		return nil, err
	}
	// get the committee
	validatorSet, err := sm.GetCommitteeMembers(id)
	if err != nil {
		return nil, err
	}
	// get the n-1 committee
	lvs, err := lastSM.GetCommitteeMembers(id)
	if err != nil {
		return nil, err
	}
	// get the delegate lottery winner
	lotteryWinner, err := sm.LotteryWinner(id)
	if err != nil {
		return nil, err
	}
	// get the order book
	orders, err := sm.GetOrderBook(id)
	if err != nil {
		return nil, err
	}
	// return the root chain info
	return &lib.RootChainInfo{
		RootChainId:      s.Config.ChainId,
		Height:           sm.height,
		ValidatorSet:     validatorSet.ValidatorSet,
		LastValidatorSet: lvs.ValidatorSet,
		LotteryWinner:    lotteryWinner,
		Orders:           orders,
	}, nil
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
		events:             new(lib.EventsTracker),
		Config:             s.Config,
		Plugin:             s.Plugin,
		log:                s.log,
		cache: &cache{
			accounts: make(map[uint64]*Account),
		},
		LastValidatorSet: s.LastValidatorSet,
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

var _ lib.PluginCompatibleFSM = new(StateMachine)

// StateRead() implements the 'state read' interface for plugins
func (s *StateMachine) StateRead(request *lib.PluginStateReadRequest) (response lib.PluginStateReadResponse, err lib.ErrorI) {
	// for each 'get' request
	for _, getRequest := range request.Keys {
		var value []byte
		// execute the 'get'
		value, err = s.Get(getRequest.Key)
		if err != nil {
			return
		}
		// add to the response
		response.Results = append(response.Results, &lib.PluginReadResult{
			QueryId: getRequest.QueryId,
			Entries: []*lib.PluginStateEntry{{Key: getRequest.Key, Value: value}},
		})
	}
	// for each 'iteration' request
	for _, r := range request.Ranges {
		var it lib.IteratorI
		// execute the 'iteration'
		if r.Reverse {
			it, err = s.Iterator(r.Prefix)
		} else {
			it, err = s.RevIterator(r.Prefix)
		}
		// handle error
		if err != nil {
			return
		}
		// calculate entries
		var entries []*lib.PluginStateEntry
		// allow 0 limit
		if r.Limit == 0 {
			r.Limit = math.MaxUint64
		}
		// while the iterator is valid and the limit is not reached
		for i := uint64(0); i < r.Limit && it.Valid(); it.Next() {
			entries = append(entries, &lib.PluginStateEntry{
				Key:   it.Key(),
				Value: it.Value(),
			})
		}
		// add to the response
		response.Results = append(response.Results, &lib.PluginReadResult{
			QueryId: r.QueryId,
			Entries: entries,
		})
	}
	return
}

// StateWrite() implements the 'state write' interface for plugins
func (s *StateMachine) StateWrite(request *lib.PluginStateWriteRequest) (response lib.PluginStateWriteResponse, err lib.ErrorI) {
	// for each 'set' request
	for _, setRequest := range request.Sets {
		// execute the 'set'
		if err = s.Set(setRequest.Key, setRequest.Value); err != nil {
			return
		}
	}
	// for each 'del' request
	for _, delRequest := range request.Deletes {
		// execute the 'delete'
		if err = s.Delete(delRequest.Key); err != nil {
			return
		}
	}
	return
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
func (s *StateMachine) TxnWrap() (lib.StoreI, lib.ErrorI) {
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
	// reset caches
	s.ResetCaches()
	// reset the state store
	s.store.(lib.StoreI).Reset()
}

// ResetCaches() dumps the state machine caches
func (s *StateMachine) ResetCaches() {
	s.cache.accounts = make(map[uint64]*Account)
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
