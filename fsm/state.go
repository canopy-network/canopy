package fsm

import (
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"runtime/debug"
	"strings"
)

const (
	ProtocolVersion = 1
)

// StateMachine the core protocol component responsible for maintaining and updating the state of the blockchain as it progresses
// it represents the collective state of all accounts, validators, and other relevant data stored on the blockchain
type StateMachine struct {
	store lib.RWStoreI

	ProtocolVersion   int
	NetworkID         uint32
	height            uint64
	vdfIterations     uint64
	slashTracker      *types.SlashTracker
	proposeVoteConfig types.GovProposalVoteConfig
	Config            lib.Config
	log               lib.LoggerI
}

// New() creates a new instance of a StateMachine
func New(c lib.Config, store lib.StoreI, log lib.LoggerI) (*StateMachine, lib.ErrorI) {
	sm := &StateMachine{
		Config:            c,
		ProtocolVersion:   ProtocolVersion,
		NetworkID:         uint32(c.P2PConfig.NetworkID),
		slashTracker:      types.NewSlashTracker(),
		proposeVoteConfig: types.AcceptAllProposals,
		log:               log,
	}
	return sm, sm.Initialize(store)
}

// Initialize() initializes a StateMachine object using the StoreI
func (s *StateMachine) Initialize(db lib.StoreI) (err lib.ErrorI) {
	s.height, s.store = db.Version(), db
	if s.height == 0 {
		// if at height zero then init from genesis file
		return s.NewFromGenesisFile()
	} else {
		blk, e := s.LoadBlock(s.Height() - 1)
		if e != nil {
			return e
		}
		s.vdfIterations = blk.BlockHeader.TotalVdfIterations
	}
	return nil
}

// ApplyBlock processes a given block, updating the state machine's state accordingly
// The function:
// - executes `BeginBlock`
// - applies all transactions within the block, generating transaction results nad a root hash
// - executes `EndBlock`
// - constructs and returns the block header, and the transaction results
func (s *StateMachine) ApplyBlock(b *lib.Block) (*lib.BlockHeader, []*lib.TxResult, lib.ErrorI) {
	defer s.catchPanic()
	store, ok := s.Store().(lib.StoreI)
	if !ok {
		return nil, nil, types.ErrWrongStoreType()
	}
	// automated execution at the 'beginning of a block'
	if err := s.BeginBlock(); err != nil {
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
	// load the previous validator set
	lastValidatorSet, err := s.LoadCanopyCommittee(s.Height() - 1)
	if err != nil {
		return nil, nil, err
	}
	// load the next validator set
	nextValidatorSet, err := s.LoadCanopyCommittee(s.Height())
	if err != nil {
		return nil, nil, err
	}
	// generate roots for block header
	lastValidatorRoot, err := lastValidatorSet.ValidatorSet.Root()
	if err != nil {
		return nil, nil, err
	}
	nextValidatorRoot, err := nextValidatorSet.ValidatorSet.Root()
	if err != nil {
		return nil, nil, err
	}
	stateRoot, err := store.Root()
	if err != nil {
		return nil, nil, err
	}
	// get the previous block for the header
	lastBlock, err := s.LoadBlock(s.height - 1)
	if err != nil {
		return nil, nil, err
	}
	// generate the block header
	header := lib.BlockHeader{
		Height:                s.Height(),
		Hash:                  nil,
		NetworkId:             s.NetworkID,
		Time:                  b.BlockHeader.Time,
		NumTxs:                uint64(numTxs),
		TotalTxs:              lastBlock.BlockHeader.TotalTxs + uint64(numTxs),
		TotalVdfIterations:    b.BlockHeader.TotalVdfIterations,
		LastBlockHash:         nonEmptyHash(lastBlock.BlockHeader.Hash),
		StateRoot:             stateRoot,
		TransactionRoot:       nonEmptyHash(txRoot),
		ValidatorRoot:         lastValidatorRoot,
		NextValidatorRoot:     nextValidatorRoot,
		ProposerAddress:       b.BlockHeader.ProposerAddress,
		Vdf:                   b.BlockHeader.Vdf,
		LastQuorumCertificate: b.BlockHeader.LastQuorumCertificate,
	}
	// create and set the block hash in the header
	if _, err = header.SetHash(); err != nil {
		return nil, nil, err
	}
	return &header, txResults, nil
}

// ApplyTransactions processes all transactions in the provided block and updates the state accordingly
func (s *StateMachine) ApplyTransactions(block *lib.Block) (results []*lib.TxResult, root []byte, n int, er lib.ErrorI) {
	var txBytes [][]byte
	blockSize := uint64(0)
	// use a map to check the block transactions for duplicates (replays)
	duplicates := make(map[string]struct{})
	// iterates over each transaction in the block
	for index, tx := range block.Transactions {
		// calculate the hex string of the hash of the transaction
		hashString := crypto.HashString(tx)
		// check if it's a duplicate
		if _, isDuplicate := duplicates[hashString]; isDuplicate {
			return results, root, n, lib.ErrDuplicateTx(hashString)
		}
		// add duplicate map
		duplicates[hashString] = struct{}{}
		// apply the tx to the state machine, generating a transaction result
		result, err := s.ApplyTransaction(uint64(index), tx, hashString)
		if err != nil {
			return nil, nil, 0, err
		}
		bz, err := lib.Marshal(result)
		if err != nil {
			return nil, nil, 0, err
		}
		results = append(results, result) // save a list of the result of the transactions
		txBytes = append(txBytes, bz)     // save a list of bytes of transactions
		blockSize += uint64(len(bz))      // size of the block of transactions
		n++                               // count
	}
	// ensure block size not exceeded
	maxBlockSize, err := s.GetMaxBlockSize()
	if err != nil {
		return nil, nil, 0, err
	}
	if blockSize > maxBlockSize {
		return nil, nil, 0, types.ErrMaxBlockSize()
	}
	// create a transaction root for the block header
	root, _, err = lib.MerkleTree(txBytes)
	return results, root, n, err
}

// TimeMachine() creates a new StateMachine instance representing the blockchain state at a specified block height, allowing for a read-only view of the past state
func (s *StateMachine) TimeMachine(height uint64) (*StateMachine, lib.ErrorI) {
	if height <= 1 {
		height = 1 // 1 is first non-genesis height
	}
	if s.height == height {
		return s, nil
	}
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, types.ErrWrongStoreType()
	}
	heightStore, err := store.NewReadOnly(height)
	if err != nil {
		return nil, err
	}
	return New(s.Config, heightStore, s.log)
}

// LoadCommittee() loads the Consensus Validators for a particular committee at a particular height
func (s *StateMachine) LoadCommittee(committeeID uint64, height uint64) (lib.ValidatorSet, lib.ErrorI) {
	fsm, err := s.TimeMachine(height)
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	vs, err := fsm.GetCommitteeMembers(committeeID)
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	return vs, nil
}

// LoadCanopyCommittee() loads the Committee for ID 0
func (s *StateMachine) LoadCanopyCommittee(height uint64) (lib.ValidatorSet, lib.ErrorI) {
	fsm, err := s.TimeMachine(height)
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	vs, err := fsm.GetCanopyCommitteeMembers()
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	return lib.NewValidatorSet(vs)
}

// GetMaxValidators() returns the max validators per committee
func (s *StateMachine) GetMaxValidators() (uint64, lib.ErrorI) {
	valParams, err := s.GetParamsVal()
	if err != nil {
		return 0, err
	}
	return valParams.ValidatorMaxCommitteeSize, nil
}

// GetMaxBlockSize() returns the maximum size of a block (excluding the header)
func (s *StateMachine) GetMaxBlockSize() (uint64, lib.ErrorI) {
	consParams, err := s.GetParamsCons()
	if err != nil {
		return 0, err
	}
	return consParams.BlockSize, nil
}

// LoadCertificate() loads a quorum certificate
func (s *StateMachine) LoadCertificate(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	//if height <= 1 {
	//	height = 1
	//}
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, types.ErrWrongStoreType()
	}
	return store.GetQCByHeight(height)
}

// LoadCertificateHashesOnly() loads a quorum certificate but nullifies the block and results
func (s *StateMachine) LoadCertificateHashesOnly(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	//if height <= 1 {
	//	height = 1
	//}
	qc, err := s.LoadCertificate(height)
	if err != nil {
		return nil, err
	}
	qc.Block, qc.Results = nil, nil
	return qc, nil
}

// LoadBlock() loads an indexed block at a specific height
func (s *StateMachine) LoadBlock(height uint64) (*lib.BlockResult, lib.ErrorI) {
	if height <= 1 {
		height = 1
	}
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, types.ErrWrongStoreType()
	}
	return store.GetBlockByHeight(height)
}

// Copy() makes a clone of the state machine
// this feature is used in mempool operation to be able to maintain a parallel ephemeral state without affecting the underlying state machine
func (s *StateMachine) Copy() (*StateMachine, lib.ErrorI) {
	st, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, types.ErrWrongStoreType()
	}
	storeCopy, err := st.Copy()
	if err != nil {
		return nil, err
	}
	return &StateMachine{
		Config:            s.Config,
		ProtocolVersion:   s.ProtocolVersion,
		NetworkID:         s.NetworkID,
		height:            s.height,
		proposeVoteConfig: s.proposeVoteConfig,
		store:             storeCopy,
		log:               s.log,
	}, nil
}

// ResetToBeginBlock() resets the store and executes 'begin-block'
func (s *StateMachine) ResetToBeginBlock() {
	s.Reset()
	if err := s.BeginBlock(); err != nil {
		panic(err)
	}
}

// Set() upserts a key-value pair under a key
func (s *StateMachine) Set(k, v []byte) lib.ErrorI {
	store := s.Store()
	if err := store.Set(k, v); err != nil {
		return err
	}
	return nil
}

// Get() retrieves a key-value pair under a key
// NOTE: returns (nil, nil) if no value is found for that key
func (s *StateMachine) Get(key []byte) ([]byte, lib.ErrorI) {
	store := s.Store()
	bz, err := store.Get(key)
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// Delete() deletes a key-value pair under a key
func (s *StateMachine) Delete(key []byte) lib.ErrorI {
	store := s.Store()
	if err := store.Delete(key); err != nil {
		return err
	}
	return nil
}

// DeleteAll() deletes all key-value pairs under a set of keys
func (s *StateMachine) DeleteAll(keys [][]byte) lib.ErrorI {
	for _, key := range keys {
		if err := s.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

// Iterator() creates and returns an iterator for the state machine's underlying store
// starting at the specified key and iterating lexicographically
func (s *StateMachine) Iterator(key []byte) (lib.IteratorI, lib.ErrorI) {
	store := s.Store()
	it, err := store.Iterator(key)
	if err != nil {
		return nil, err
	}
	return it, nil
}

// RevIterator() creates and returns an iterator for the state machine's underlying store
// starting at the end-prefix of the specified key and iterating reverse lexicographically
func (s *StateMachine) RevIterator(key []byte) (lib.IteratorI, lib.ErrorI) {
	store := s.Store()
	it, err := store.RevIterator(key)
	if err != nil {
		return nil, err
	}
	return it, nil
}

// IterateAndExecute() creates an iterator and executes a callback function for each key-value pair
func (s *StateMachine) IterateAndExecute(prefix []byte, callback func(key, value []byte) lib.ErrorI) lib.ErrorI {
	it, err := s.Iterator(prefix)
	if err != nil {
		return err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		if err = callback(it.Key(), it.Value()); err != nil {
			return err
		}
	}
	return nil
}

// TxnWrap() is an atomicity and consistency feature that enables easy rollback of changes by discarding the transaction if an error occurs
func (s *StateMachine) TxnWrap() (lib.StoreTxnI, lib.ErrorI) {
	store, ok := s.store.(lib.StoreI)
	if !ok {
		return nil, types.ErrWrongStoreType()
	}
	txn := store.NewTxn()
	s.SetStore(txn)
	return txn, nil
}

func (s *StateMachine) Store() lib.RWStoreI         { return s.store }
func (s *StateMachine) SetStore(store lib.RWStoreI) { s.store = store }
func (s *StateMachine) Height() uint64              { return s.height }
func (s *StateMachine) TotalVDFIterations() uint64  { return s.vdfIterations }
func (s *StateMachine) Reset() {
	s.slashTracker = types.NewSlashTracker()
	s.store.(lib.StoreI).Reset()
}
func (s *StateMachine) SetProposalVoteConfig(c types.GovProposalVoteConfig) { s.proposeVoteConfig = c }

// catchPanic() acts as a failsafe, recovering from a panic and logging the error with the stack trace
func (s *StateMachine) catchPanic() {
	if r := recover(); r != nil {
		s.log.Error(string(debug.Stack()))
	}
}

// nonEmptyHash() ensures the hash isn't empty
// substituting a dummy hash in its place
func nonEmptyHash(h []byte) []byte {
	if len(h) == 0 {
		h = []byte(strings.Repeat("F", crypto.HashSize))
	}
	return h
}
