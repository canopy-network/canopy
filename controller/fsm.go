package controller

import (
	"bytes"
	"math"
	"math/rand"
	"time"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// HandleTransaction() accepts or rejects inbound txs based on the mempool state
// - pass through call checking indexer and mempool for duplicate
func (c *Controller) HandleTransaction(tx []byte) lib.ErrorI {
	hash := crypto.Hash(tx)
	hashString := lib.BytesToString(hash)
	// indexer
	txResult, err := c.FSM.Store().(lib.StoreI).GetTxByHash(hash)
	if err != nil {
		return err
	}
	if txResult.TxHash != "" {
		return lib.ErrDuplicateTx(hashString)
	}
	// mempool
	if c.Mempool.Contains(hashString) {
		return lib.ErrTxFoundInMempool(hashString)
	}
	return c.Mempool.HandleTransaction(tx)
}

// ProduceProposal() uses the associated `plugin` to create a Proposal with the candidate block and the `bft` to populate the byzantine evidence
func (c *Controller) ProduceProposal(be *bft.ByzantineEvidence, vdf *crypto.VDF) (blk []byte, results *lib.CertificateResult, err lib.ErrorI) {
	height, reset := c.FSM.Height(), c.ValidatorProposalConfig()
	defer func() { reset(); c.FSM.Reset() }()
	// load the previous block from the store
	qc, _, err := c.FSM.LoadBlockAndCertificate(height - 1)
	if err != nil {
		return
	}
	// get the maximum possible size of the block
	maxBlockSize, err := c.FSM.GetMaxBlockSize()
	if err != nil {
		return
	}
	// re-validate all transactions in the mempool as a fail-safe
	c.Mempool.checkMempool()
	// extract transactions from the mempool
	transactions, _ := c.Mempool.GetTransactions(maxBlockSize - lib.MaxBlockHeaderSize)
	// validate VDF
	if vdf != nil {
		if !crypto.VerifyVDF(qc.BlockHash, vdf.Output, vdf.Proof, int(vdf.Iterations)) {
			c.log.Error(lib.ErrInvalidVDF().Error())
			vdf = nil
		}
	}
	// create a block structure
	block := &lib.Block{
		BlockHeader:  &lib.BlockHeader{Time: uint64(time.Now().UnixMicro()), ProposerAddress: c.Address, LastQuorumCertificate: qc, Vdf: vdf},
		Transactions: transactions,
	}
	// capture the tentative block result here
	blockResult := new(lib.BlockResult)
	// apply the block against the state machine to fill in the merkle `roots` for the block header
	block.BlockHeader, blockResult.Transactions, err = c.FSM.ApplyBlock(block)
	if err != nil {
		return
	}
	// add the block header to the results
	blockResult.BlockHeader = block.BlockHeader
	// marshal the block into bytes
	blk, err = lib.Marshal(block)
	if err != nil {
		return
	}
	// set block reward recipients
	results = c.CalculateRewardRecipients(c.Address, c.RootChainHeight())
	// handle swaps
	c.HandleSwaps(blockResult, results, c.RootChainHeight())
	// set slash recipients
	c.CalculateSlashRecipients(results, be)
	// set checkpoint
	c.CalculateCheckpoint(blockResult, results)
	// handle retired status
	c.HandleRetired(results)
	return
}

// ValidateProposal() fully validates the proposal and resets back to begin block state
func (c *Controller) ValidateProposal(qc *lib.QuorumCertificate, evidence *bft.ByzantineEvidence) (err lib.ErrorI) {
	// the base chain has specific logic to approve or reject proposals
	reset := c.ValidatorProposalConfig()
	defer func() { reset(); c.FSM.Reset() }()
	// load the store (db) object
	storeI := c.FSM.Store().(lib.StoreI)
	// perform stateless validations on the block and evidence
	blk, err := c.ValidateProposalBasic(qc, evidence)
	if err != nil {
		return err
	}
	// update the LastQuorumCertificate in the store
	//- this is to ensure that each node has the same COMMIT QC for the last block as multiple valid versions (a few less signatures) of the same QC could exist
	// NOTE: this should come before ApplyAndValidateBlock in order to have the proposers 'LastQc' which is used for distributing rewards
	if blk.BlockHeader.Height != 1 {
		c.log.Debugf("Indexing last quorum certificate for height %d", blk.BlockHeader.LastQuorumCertificate.Header.Height)
		if err = storeI.IndexQC(blk.BlockHeader.LastQuorumCertificate); err != nil {
			return err
		}
	}
	// play the block against the state machine
	blockResult, err := c.ApplyAndValidateBlock(blk, false)
	if err != nil {
		return
	}
	// generate a comparable
	compareResults := c.CalculateRewardRecipients(blk.BlockHeader.ProposerAddress, c.RootChainHeight())
	// handle swaps
	c.HandleSwaps(blockResult, compareResults, c.RootChainHeight())
	// set slash recipients
	c.CalculateSlashRecipients(compareResults, evidence)
	// set checkpoint
	c.CalculateCheckpoint(blockResult, compareResults)
	// handle retired status
	c.HandleRetired(compareResults)
	// ensure generated the same results
	if !qc.Results.Equals(compareResults) {
		return types.ErrMismatchCertResults()
	}
	return
}

// ValidateProposalBasic() performs basic structural validates the proposal and resets back to begin block state
func (c *Controller) ValidateProposalBasic(qc *lib.QuorumCertificate, evidence *bft.ByzantineEvidence) (blk *lib.Block, err lib.ErrorI) {
	// validate the byzantine evidence portion of the proposal (bft is canopy controlled)
	if err = c.Consensus.ValidateByzantineEvidence(qc.Results.SlashRecipients, evidence); err != nil {
		return nil, err
	}
	// ensure the block is not empty
	if qc.Block == nil {
		return nil, lib.ErrNilBlock()
	}
	// ensure the results aren't empty
	if qc.Results == nil && qc.Results.RewardRecipients != nil {
		return nil, lib.ErrNilCertResults()
	}
	// convert the block bytes into a Canopy block structure
	blk = new(lib.Block)
	if err = lib.Unmarshal(qc.Block, blk); err != nil {
		return
	}
	// basic structural validations of the block
	if err = blk.Check(c.Config.NetworkID, c.Config.ChainId); err != nil {
		return
	}
	// ensure the Proposal.BlockHash corresponds to the actual hash of the block
	blockHash, e := blk.Hash()
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(qc.BlockHash, blockHash) {
		return nil, lib.ErrMismatchHeaderBlockHash()
	}
	return
}

// CommitCertificate used after consensus decides on a block
// - applies block against the fsm
// - indexes the block and its transactions
// - removes block transactions from mempool
// - re-checks all transactions in mempool
// - atomically writes all to the underlying db
// - sets up the app for the next height
func (c *Controller) CommitCertificate(qc *lib.QuorumCertificate) lib.ErrorI {
	// reset the store once this code finishes
	// NOTE: if code execution gets to `store.Commit()` - this will effectively be a noop
	defer func() { c.FSM.Reset() }()
	c.log.Debugf("TryCommit block %s", lib.BytesToString(qc.ResultsHash))
	// load the store (db) object
	storeI := c.FSM.Store().(lib.StoreI)
	// convert the proposal.block (bytes) into Block structure
	blk := new(lib.Block)
	if err := lib.Unmarshal(qc.Block, blk); err != nil {
		return err
	}
	// update the LastQuorumCertificate in the store
	//- this is to ensure that each node has the same COMMIT QC for the last block as multiple valid versions (a few less signatures) of the same QC could exist
	// NOTE: this should come before ApplyAndValidateBlock in order to have the proposers 'LastQc' which is used for distributing rewards
	if blk.BlockHeader.Height != 1 {
		c.log.Debugf("Indexing last quorum certificate for height %d", blk.BlockHeader.LastQuorumCertificate.Header.Height)
		if err := storeI.IndexQC(blk.BlockHeader.LastQuorumCertificate); err != nil {
			return err
		}
	}
	// apply the block against the state machine
	blockResult, err := c.ApplyAndValidateBlock(blk, true)
	if err != nil {
		return err
	}
	// index the quorum certificate in the store
	c.log.Debugf("Indexing quorum certificate for height %d", qc.Header.Height)
	if err = storeI.IndexQC(qc); err != nil {
		return err
	}
	// index the block in the store
	c.log.Debugf("Indexing block %d", blk.BlockHeader.Height)
	if err = storeI.IndexBlock(blockResult); err != nil {
		return err
	}
	// delete the block transactions in the mempool
	for _, tx := range blk.Transactions {
		c.log.Debugf("Tx %s was included in a block so removing from mempool", crypto.HashString(tx))
		c.Mempool.DeleteTransaction(tx)
	}
	// rescan mempool to ensure validity of all transactions
	c.log.Debug("Checking mempool for newly invalid transactions")
	c.Mempool.checkMempool()
	// parse committed block for straw polls
	c.FSM.ParsePollTransactions(blockResult)
	// atomically write this to the store
	c.log.Debug("Committing to store")
	if _, err = storeI.Commit(); err != nil {
		return err
	}
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToString(qc.ResultsHash), blk.BlockHeader.Height)
	c.log.Debug("Setting up FSM for next height")
	c.FSM, err = fsm.New(c.Config, storeI, c.log)
	if err != nil {
		return err
	}
	c.log.Debug("Setting up Mempool for next height")
	if c.Mempool.FSM, err = c.FSM.Copy(); err != nil {
		return err
	}
	if err = c.Mempool.FSM.BeginBlock(); err != nil {
		return err
	}
	c.log.Debug("Commit done")
	return nil
}

// ApplyAndValidateBlock() plays the block against the State Machine and
// compares the block header against a results from the state machine
func (c *Controller) ApplyAndValidateBlock(b *lib.Block, commit bool) (*lib.BlockResult, lib.ErrorI) {
	// basic structural checks on the block
	if err := b.Check(c.Config.NetworkID, c.Config.ChainId); err != nil {
		return nil, err
	}
	// define convenience variables
	blockHash, blockHeight := lib.BytesToString(b.BlockHeader.Hash), b.BlockHeader.Height
	// apply the block against the state machine
	c.log.Debugf("Applying block %s for height %d", blockHash, blockHeight)
	header, txResults, err := c.FSM.ApplyBlock(b)
	if err != nil {
		return nil, err
	}
	// compare the resulting header against the block header (should be identical)
	c.log.Debugf("Validating block header %s for height %d", blockHash, blockHeight)
	if err = c.CompareBlockHeaders(b.BlockHeader, header, commit); err != nil {
		return nil, err
	}
	// return a valid block result
	c.log.Infof("Block %s with %d txs is valid for height %d ✅ ", blockHash, len(b.Transactions), blockHeight)
	return &lib.BlockResult{
		BlockHeader:  b.BlockHeader,
		Transactions: txResults,
	}, nil
}

// CalculateRewardRecipients() calculates the block reward recipients of the proposal
func (c *Controller) CalculateRewardRecipients(proposerAddress []byte, rootChainHeight uint64) (results *lib.CertificateResult) {
	// set block reward recipients
	results = &lib.CertificateResult{
		RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{
				{
					Address: proposerAddress, // root-Chain proposer reward
					Percent: 100,             // proposer gets what's left after the delegate's cut
				},
			},
		},
		SlashRecipients: new(lib.SlashRecipients),
	}
	// get the delegate and their cut from the state machine
	baseDelegateWinner, err := c.GetRootChainLotteryWinner(rootChainHeight)
	if err != nil {
		c.log.Warnf("An error occurred choosing a root-Chain delegate lottery winner: %s", err.Error())
		// continue
	} else {
		c.AddLotteryWinner(results, baseDelegateWinner)
	}
	// load the root chain id from state
	rootChainId, err := c.FSM.GetRootChainId()
	if err != nil {
		c.log.Warnf("An error occurred getting the root chain id from state: %s", err.Error())
	}
	// is isn't root chain
	if c.Config.ChainId != rootChainId {
		// sub validator
		subValidatorWinner, e := c.FSM.LotteryWinner(c.Config.ChainId, true)
		if e != nil {
			c.log.Warnf("An error occurred choosing a sub-validator lottery winner: %s", err.Error())
			// continue
		} else {
			c.AddLotteryWinner(results, subValidatorWinner)
		}
		// sub delegate
		subDelegateWinner, e := c.FSM.LotteryWinner(c.Config.ChainId)
		if e != nil {
			c.log.Warnf("An error occurred choosing a sub-delegate lottery winner: %s", err.Error())
			// continue
		} else {
			c.AddLotteryWinner(results, subDelegateWinner)
		}
	}
	return
}

// HandleSwaps() handles the 'buy' side of the sell orders
func (c *Controller) HandleSwaps(blockResult *lib.BlockResult, results *lib.CertificateResult, rootChainHeight uint64) {
	// parse the last block for buy orders and polling
	buyOrders := c.FSM.ParseBuyOrders(blockResult)
	// get orders from the root-Chain
	orders, e := c.LoadRootChainOrderBook(rootChainHeight)
	if e != nil {
		return
	}
	// process the root-Chain order book against the state
	closeOrders, resetOrders := c.FSM.ProcessRootChainOrderBook(orders, blockResult)
	// add the orders to the certificate result
	// truncate for defensive spam protection
	results.Orders = &lib.Orders{
		BuyOrders:   lib.TruncateSlice(buyOrders, 1000),
		ResetOrders: resetOrders,
		CloseOrders: closeOrders,
	}
}

// CalculateSlashRecipients() calculates the addresses who receive slashes on the root-Chain
func (c *Controller) CalculateSlashRecipients(results *lib.CertificateResult, be *bft.ByzantineEvidence) {
	var err lib.ErrorI
	// use the bft object to fill in the Byzantine Evidence
	results.SlashRecipients.DoubleSigners, err = c.Consensus.ProcessDSE(be.DSE.Evidence...)
	if err != nil {
		c.log.Warn(err.Error()) // still produce proposal
	}
	if srLen := len(results.SlashRecipients.DoubleSigners); srLen != 0 {
		c.log.Infof("Added %d slash recipients due to byzantine evidence", srLen)
	}
}

// CalculateCheckpoint() calculates the checkpoint for the checkpoint as a service functionality
func (c *Controller) CalculateCheckpoint(blockResult *lib.BlockResult, results *lib.CertificateResult) {
	// checkpoint every 100 heights
	if blockResult.BlockHeader.Height%c.GetCheckpointFrequency() == 0 {
		c.log.Info("Checkpoint set in certificate results")
		results.Checkpoint = &lib.Checkpoint{
			Height:    blockResult.BlockHeader.Height,
			BlockHash: blockResult.BlockHeader.Hash,
		}
	}
}

// GetCheckpointFrequency() returns how frequently the chain checkpoints
func (c *Controller) GetCheckpointFrequency() uint64 {
	return 100
}

// HandleRetired() checks if the committee is retiring and sets in the results accordingly
func (c *Controller) HandleRetired(results *lib.CertificateResult) {
	cons, err := c.FSM.GetParamsCons()
	if err != nil {
		c.log.Error(err.Error())
		return
	}
	results.Retired = cons.Retired != 0
}

// AddLotteryWinner() adds a lottery winner (delegate, sub-delegate, or sub-validator) to the reward recipients
func (c *Controller) AddLotteryWinner(results *lib.CertificateResult, lotteryWinner *lib.LotteryWinner) {
	if len(lotteryWinner.Winner) == 0 {
		c.log.Debug("Nil lottery winner (no delegates in set)")
		return
	}
	if results.RewardRecipients.PaymentPercents[0].Percent <= lotteryWinner.Cut {
		c.log.Warn("No cut left for block proposer, skipping lottery winner (ensure root-Chain-lottery-cut + 2 x nested-chain-lottery-cut < 100%)")
		return
	}
	results.RewardRecipients.PaymentPercents[0].Percent -= lotteryWinner.Cut
	results.RewardRecipients.PaymentPercents = append(results.RewardRecipients.PaymentPercents, &lib.PaymentPercents{
		Address: lotteryWinner.Winner,
		Percent: lotteryWinner.Cut,
	})
}

// CompareBlockHeaders() compares two block headers for equality and validates the last quorum certificate and block time
func (c *Controller) CompareBlockHeaders(candidate *lib.BlockHeader, compare *lib.BlockHeader, commit bool) lib.ErrorI {
	// compare the block headers for equality
	hash, e := compare.SetHash()
	if e != nil {
		return e
	}
	if !bytes.Equal(hash, candidate.Hash) {
		return lib.ErrUnequalBlockHash()
	}
	// validate the last quorum certificate of the block header
	// by using the historical committee
	if candidate.Height > 2 {
		lastCertificate, err := c.FSM.LoadCertificateHashesOnly(candidate.Height - 1)
		if err != nil {
			return err
		}
		// validate the expected QC hash
		if candidate.LastQuorumCertificate == nil ||
			candidate.LastQuorumCertificate.Header == nil ||
			!bytes.Equal(candidate.LastQuorumCertificate.BlockHash, lastCertificate.BlockHash) ||
			!bytes.Equal(candidate.LastQuorumCertificate.ResultsHash, lastCertificate.ResultsHash) {
			return lib.ErrInvalidLastQuorumCertificate()
		}
		// load the committee for the last qc
		committeeHeight := candidate.LastQuorumCertificate.Header.RootHeight
		vs, err := c.LoadCommittee(committeeHeight)
		if err != nil {
			return err
		}
		// check the last QC
		isPartialQC, err := candidate.LastQuorumCertificate.Check(vs, 0, &lib.View{
			Height:     candidate.Height - 1,
			RootHeight: committeeHeight,
			NetworkId:  c.Config.NetworkID,
			ChainId:    c.Config.ChainId,
		}, true)
		if err != nil {
			return err
		}
		// ensure is a full +2/3rd maj QC
		if isPartialQC {
			return lib.ErrNoMaj23()
		}
	}
	// validate VDF
	if candidate.Vdf != nil {
		// if syncing or committing - only verify the vdf randomly since this randomness is pseudo-non-deterministic (among nodes) this holds
		// similar security guarantees but lowers the computational requirements at a per-node basis
		if !commit || rand.Intn(100) == 0 {
			if !crypto.VerifyVDF(candidate.LastQuorumCertificate.BlockHash, candidate.Vdf.Output, candidate.Vdf.Proof, int(candidate.Vdf.Iterations)) {
				return lib.ErrInvalidVDF()
			}
		}
	}
	return nil
}

// CheckPeerQc() performs a basic checks on a block received from a peer and returns if the self is out of sync
func (c *Controller) CheckPeerQC(maxHeight uint64, qc *lib.QuorumCertificate) (stillCatchingUp bool, err lib.ErrorI) {
	// convert the proposal block bytes into a block structure
	blk := new(lib.Block)
	if err = lib.Unmarshal(qc.Block, blk); err != nil {
		return
	}
	// ensure the unmarshalled block is not nil
	if blk == nil || blk.BlockHeader == nil {
		return false, lib.ErrNilBlock()
	}
	// get the hash from the block
	hash, err := blk.Hash()
	if err != nil {
		return false, err
	}
	// ensure the Proposal.BlockHash corresponds to the actual hash of the block
	if !bytes.Equal(qc.BlockHash, hash) {
		err = lib.ErrMismatchQCBlockHash()
		return
	}
	// check the height of the block
	h := c.FSM.Height()
	// don't accept any blocks below the local height
	if h > blk.BlockHeader.Height {
		err = lib.ErrWrongMaxHeight()
		return
	}
	// don't accept any blocks greater than 'max height'
	if blk.BlockHeader.Height > maxHeight {
		err = lib.ErrWrongMaxHeight()
		return
	}
	// height of this block is not local height, the node is still catching up
	stillCatchingUp = h != blk.BlockHeader.Height
	return
}

// ValidatorProposalConfig() is how the Validator is configured for `base chain` specific parameter upgrades
func (c *Controller) ValidatorProposalConfig() (reset func()) {
	if c.Consensus.GetRound() < 3 {
		// if the node is not having 'consensus issues' refer to the approve list
		c.FSM.SetProposalVoteConfig(types.GovProposalVoteConfig_APPROVE_LIST)
	} else {
		// if the node is exhibiting 'chain halt' like behavior, reject all proposals
		c.FSM.SetProposalVoteConfig(types.GovProposalVoteConfig_REJECT_ALL)
	}
	// a callback that resets the configuration back to default
	reset = func() {
		// the default is to accept all except in 'Consensus mode'
		c.FSM.SetProposalVoteConfig(types.AcceptAllProposals)
	}
	return
}

// GetPendingPage() returns a page of unconfirmed mempool transactions
func (c *Controller) GetPendingPage(p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// lock the controller for thread safety
	c.Lock()
	defer c.Unlock()
	page, txResults := lib.NewPage(p, lib.PendingResultsPageName), make(lib.TxResults, 0)
	err = page.LoadArray(c.Mempool.cachedResults, &txResults, func(i any) lib.ErrorI {
		v, ok := i.(*lib.TxResult)
		if !ok {
			return lib.ErrInvalidArgument()
		}
		txResults = append(txResults, v)
		return nil
	})
	return
}

// GetFailedTxsPage() returns a list of failed mempool transactions
func (c *Controller) GetFailedTxsPage(address string, p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// lock the controller for thread safety
	c.Lock()
	defer c.Unlock()
	page, failedTxs := lib.NewPage(p, lib.FailedTxsPageName), make(lib.FailedTxs, 0)
	err = page.LoadArray(c.Mempool.cachedFailedTxs.GetAddr(address), &failedTxs, func(i any) lib.ErrorI {
		v, ok := i.(*lib.FailedTx)
		if !ok {
			return lib.ErrInvalidArgument()
		}
		failedTxs = append(failedTxs, v)
		return nil
	})
	return
}

// GetFailedTxsPage() returns a list of failed transactions
// func (c *Controller) GetFailedTxPage(p lib.PageParams) (page *lib.Page)

// Mempool accepts or rejects incoming txs based on the mempool (ephemeral copy) state
// - recheck when
//   - mempool dropped some percent of the lowest fee txs
//   - new tx has higher fee than the lowest
//
// - notes:
//   - new tx added may also be evicted, this is expected behavior
type Mempool struct {
	log             lib.LoggerI
	FSM             *fsm.StateMachine
	cachedResults   lib.TxResults
	cachedFailedTxs *lib.FailedTxCache
	lib.Mempool
}

// NewMempool() creates a new instance of a Mempool structure
func NewMempool(fsm *fsm.StateMachine, config lib.MempoolConfig, log lib.LoggerI) (m *Mempool, err lib.ErrorI) {
	// initialize the structure
	m = &Mempool{
		log:             log,
		Mempool:         lib.NewMempool(config),
		cachedFailedTxs: lib.NewFailedTxCache(),
	}
	// make an 'mempool (ephemeral copy) state' so the mempool can maintain only 'valid' transactions
	// despite dependencies and conflicts
	m.FSM, err = fsm.Copy()
	if err != nil {
		return nil, err
	}
	m.FSM.ResetToBeginBlock()
	return m, err
}

// HandleTransaction() attempts to add a transaction to the mempool by validating, adding, and evicting overfull or newly invalid txs
func (m *Mempool) HandleTransaction(tx []byte) (err lib.ErrorI) {
	defer func() {
		// cache failed txs for RPC display
		if err != nil {
			m.cachedFailedTxs.Add(tx, crypto.HashString(tx), err)
		}
	}()

	// validate the transaction against the mempool (ephemeral copy) state
	result, err := m.applyAndWriteTx(tx)
	if err != nil {
		return err
	}
	fee := result.Transaction.Fee
	// prioritize certificate result transactions by artificially raising the fee 'stored fee'
	if result.MessageType == types.MessageCertificateResultsName {
		fee = math.MaxUint32
	}
	// add a transaction to the mempool
	recheck, err := m.AddTransaction(tx, fee)
	if err != nil {
		return err
	}
	// cache the results for RPC display
	m.log.Infof("Added tx %s to mempool for checking", crypto.HashString(tx))
	m.cachedResults = append(m.cachedResults, result)
	// recheck the mempool if necessary
	if recheck {
		m.checkMempool()
	}
	return nil
}

// checkMempool() validates all transactions the mempool using the mempool (ephemeral copy) state and evicts any that are invalid
func (m *Mempool) checkMempool() {
	// reset the mempool (ephemeral copy) state to just after the automatic 'begin block' phase
	m.FSM.ResetToBeginBlock()
	// reset the RPC cached results
	m.cachedResults = nil
	// define convenience variables
	var remove [][]byte
	// create an iterator for the mempool
	it := m.Iterator()
	defer it.Close()
	// for each mempool transaction
	for ; it.Valid(); it.Next() {
		// write the transaction to the state machine
		tx := it.Key()
		result, err := m.applyAndWriteTx(tx)
		if err != nil {
			// if invalid, add to the remove list
			m.log.Error(err.Error())
			remove = append(remove, tx)
			// and cache it
			m.cachedFailedTxs.Add(tx, crypto.HashString(tx), err)
			continue
		}
		// cache the results
		m.cachedResults = append(m.cachedResults, result)
	}
	// evict all 'newly' invalid transactions from the mempool
	for _, tx := range remove {
		m.log.Infof("removed tx %s from mempool", crypto.HashString(tx))
		m.DeleteTransaction(tx)
	}
}

// applyAndWriteTx() checks the validity of a transaction by playing it against the mempool (ephemeral copy) state machine
func (m *Mempool) applyAndWriteTx(tx []byte) (result *lib.TxResult, err lib.ErrorI) {
	store := m.FSM.Store()
	// wrap the store in a transaction in case a rollback to the previous valid transaction is needed
	txn, err := m.FSM.TxnWrap()
	if err != nil {
		return nil, err
	}
	// at the end of this code, reset the state machine store and discard the transaction
	defer func() { m.FSM.SetStore(store); txn.Discard() }()
	// apply the transaction to the mempool (ephemeral copy) state machine
	result, err = m.FSM.ApplyTransaction(uint64(m.TxCount()), tx, crypto.HashString(tx))
	if err != nil {
		// if invalid return error
		return nil, err
	}
	// write the transaction to the mempool store
	if err = txn.Write(); err != nil {
		return nil, err
	}
	// return the result
	return result, nil
}
