package controller

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/canopy-network/canopy/p2p"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

/* This file contains the high level functionality of block / proposal processing */

// ListenForBlock() listens for inbound block messages, internally routes them, and gossips them to peers
func (c *Controller) ListenForBlock() {
	// log the beginning of the 'block listener' service
	c.log.Debug("Listening for inbound blocks")
	// initialize a cache that prevents duplicate messages
	cache := lib.NewMessageCache()
	// wait and execute for each inbound message received
	for msg := range c.P2P.Inbox(Block) {
		// create a variable to signal a 'stop loop'
		var quit bool
		// wrap in a function call to use 'defer' functionality
		func() {
			c.log.Debug("Handling block message")
			defer lib.TimeTrack("ListenForBlock", time.Now())
			// lock the controller to prevent multi-thread conflicts
			c.Lock()
			// when iteration completes, unlock
			defer c.Unlock()
			// check and add the message to the cache to prevent duplicates
			if ok := cache.Add(msg); !ok {
				// if duplicate, exit iteration
				return
			}
			// add a convenience variable to track the sender
			sender := msg.Sender.Address.PublicKey
			// log the receipt of the block message
			c.log.Infof("Received new block from %s ✉️", lib.BytesToTruncatedString(sender))
			// try to cast the message to a block message
			blockMessage, ok := msg.Message.(*lib.BlockMessage)
			// if cast fails (not a block message)
			if !ok {
				// log the error
				c.log.Debug("Invalid Peer Block Message")
				// slash the peer's reputation
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit iteration
				return
			}
			// track processing time for consensus module
			startTime := time.Now()
			// 'handle' the peer block and certificate appropriately
			qc, err := c.HandlePeerBlock(blockMessage, false)
			// ensure no error
			if err != nil {
				// if the node has fallen 'out of sync' with the chain
				if err == lib.ErrOutOfSync() {
					// log the 'out of sync' message
					c.log.Warnf("Node fell out of sync for chainId: %d", blockMessage.ChainId)
					// revert to syncing mode
					go c.Sync()
					// signal exit the out loop
					quit = true
					// exit iteration
					return
				}
				// log the error
				c.log.Warnf("Peer block invalid:\n%s", err.Error())
				// slash the peer's reputation
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit iteration
				return
			}
			// gossip the block to our peers
			c.GossipBlock(qc, sender)
			c.log.Debugf("ListenForBlock -> Reset BFT: %d", len(c.Consensus.ResetBFT))
			// signal a reset to the bft module
			c.Consensus.ResetBFT <- bft.ResetBFT{ProcessTime: time.Since(startTime)}
		}()
		// if quit signaled
		if quit {
			// exit the loop
			return
		}
	}
}

// PUBLISHERS BELOW

// GossipBlockMsg() gossips a certificate (with block) through the P2P network for a specific chainId
func (c *Controller) GossipBlock(certificate *lib.QuorumCertificate, senderPubToExclude []byte) {
	// log the start of the gossip block function
	c.log.Debugf("Gossiping certificate: %s", lib.BytesToString(certificate.ResultsHash))
	// create the block message to gossip
	blockMessage := &lib.BlockMessage{
		ChainId:             c.Config.ChainId,
		BlockAndCertificate: certificate,
	}
	// send the block message to all peers excluding the sender (gossip)
	if err := c.P2P.SendToPeers(Block, blockMessage, lib.BytesToString(senderPubToExclude)); err != nil {
		c.log.Errorf("unable to gossip block with err: %s", err.Error())
	}
}

// SelfSendBlock() gossips a QuorumCertificate (with block) through the P2P network for handling
func (c *Controller) SelfSendBlock(certificate *lib.QuorumCertificate) {
	// create the block message
	blockMessage := &lib.BlockMessage{
		ChainId:             c.Config.ChainId,
		BlockAndCertificate: certificate,
	}
	// internally route the block to the 'block inbox'
	if err := c.P2P.SelfSend(c.PublicKey, Block, blockMessage); err != nil {
		c.log.Errorf("unable to gossip block with err: %s", err.Error())
	}
}

// BFT FUNCTIONS BELOW

// ProduceProposal() create a proposal in the form of a `block` and `certificate result` for the bft process
func (c *Controller) ProduceProposal(evidence *bft.ByzantineEvidence, vdf *crypto.VDF) (blockBytes []byte, results *lib.CertificateResult, err lib.ErrorI) {
	c.log.Debugf("Producing proposal as leader")
	// configure the FSM in 'consensus mode' for validator proposals
	resetProposalConfig := c.SetFSMInConsensusModeForProposals()
	// once done proposing, 'reset' the proposal mode back to default to 'accept all'
	defer func() { resetProposalConfig(); c.FSM.Reset() }()
	// load the previous quorum height quorum certificate from the indexer
	lastCertificate, err := c.FSM.LoadCertificateHashesOnly(c.FSM.Height() - 1)
	if err != nil {
		return
	}
	// validate the verifiable delay function from the bft module
	if vdf != nil {
		// if the verifiable delay function is NOT valid for using the last block hash
		if !crypto.VerifyVDF(lastCertificate.BlockHash, vdf.Output, vdf.Proof, int(vdf.Iterations)) {
			// nullify the bad VDF
			vdf = nil
			// log the issue but still continue with the proposal
			c.log.Error(lib.ErrInvalidVDF().Error())
		}
	}
	// re-validate all transactions in the mempool as a 'double check' to ensure all transactions being proposed are valid
	c.Mempool.checkMempool()
	// get the maximum possible size of the block as defined by the governance parameters of the state machine
	maxBlockSize, err := c.FSM.GetMaxBlockSize()
	if err != nil {
		// exit with error
		return
	}
	// create the actual block structure with the maximum amount of transactions allowed or available in the mempool
	block := &lib.Block{
		BlockHeader:  &lib.BlockHeader{Time: uint64(time.Now().UnixMicro()), ProposerAddress: c.Address, LastQuorumCertificate: lastCertificate, Vdf: vdf},
		Transactions: c.Mempool.GetTransactions(maxBlockSize - lib.MaxBlockHeaderSize),
	}
	// capture the tentative block result using a new object reference
	blockResult := new(lib.BlockResult)
	// apply the block against the state machine and populate the resulting merkle `roots` in the block header
	block.BlockHeader, blockResult.Transactions, err = c.FSM.ApplyBlock(block)
	if err != nil {
		// exit with error
		return
	}
	// convert the block reference to bytes
	blockBytes, err = lib.Marshal(block)
	if err != nil {
		// exit with error
		return
	}
	// update the 'block results' with the newly created header
	blockResult.BlockHeader = block.BlockHeader
	// create a new certificate results (includes reward recipients, slash recipients, swap commands, etc)
	results = c.NewCertificateResults(block, blockResult, evidence)
	// exit
	return
}

// ValidateProposal() fully validates a proposal in the form of a quorum certificate and resets back to begin block state
func (c *Controller) ValidateProposal(qc *lib.QuorumCertificate, evidence *bft.ByzantineEvidence) (err lib.ErrorI) {
	// log the beginning of proposal validation
	c.log.Debugf("Validating proposal from leader")
	// configure the FSM in 'consensus mode' for validator proposals
	resetProposalConfig := c.SetFSMInConsensusModeForProposals()
	// once done proposing, 'reset' the proposal mode back to default to 'accept all'
	defer func() { resetProposalConfig(); c.FSM.Reset() }()
	// ensure the proposal inside the quorum certificate is valid at a stateless level
	block, err := qc.CheckProposalBasic(c.FSM.Height(), c.Config.NetworkID, c.Config.ChainId)
	if err != nil {
		// exit with error
		return
	}
	// validate the byzantine evidence portion of the proposal (bft is canopy controlled)
	if err = c.Consensus.ValidateByzantineEvidence(qc.Results.SlashRecipients, evidence); err != nil {
		// exit with error
		return
	}
	// play the block against the state machine to generate a block result
	blockResult, err := c.ApplyAndValidateBlock(block, false)
	if err != nil {
		// exit with error
		return
	}
	// create a comparable certificate results (includes reward recipients, slash recipients, swap commands, etc)
	compareResults := c.NewCertificateResults(block, blockResult, evidence)
	// ensure generated the same results
	if !qc.Results.Equals(compareResults) {
		// exit with error
		return fsm.ErrMismatchCertResults()
	}
	// exit
	return
}

// CommitCertificate() is executed after the quorum agrees on a block
// - applies block against the fsm
// - indexes the block and its transactions
// - removes block transactions from mempool
// - re-checks all transactions in mempool
// - atomically writes all to the underlying db
// - sets up the controller for the next height
func (c *Controller) CommitCertificate(qc *lib.QuorumCertificate, block *lib.Block) (err lib.ErrorI) {
	start := time.Now()
	// reset the store once this code finishes; if code execution gets to `store.Commit()` - this will effectively be a noop
	defer func() { c.FSM.Reset() }()
	// log the beginning of the commit
	c.log.Debugf("TryCommit block %s", lib.BytesToString(qc.ResultsHash))
	// cast the store to ensure the proper store type to complete this operation
	storeI := c.FSM.Store().(lib.StoreI)
	// apply the block against the state machine
	blockResult, err := c.ApplyAndValidateBlock(block, true)
	if err != nil {
		// exit with error
		return
	}
	// log indexing the quorum certificate
	c.log.Debugf("Indexing certificate for height %d", qc.Header.Height)
	// index the quorum certificate in the store
	if err = storeI.IndexQC(qc); err != nil {
		// exit with error
		return
	}
	// log indexing the block
	c.log.Debugf("Indexing block %d", block.BlockHeader.Height)
	// index the block in the store
	if err = storeI.IndexBlock(blockResult); err != nil {
		// exit with error
		return
	}
	// for each transaction included in the block
	for _, tx := range block.Transactions {
		// delete each transaction from the mempool
		c.Mempool.DeleteTransaction(tx)
	}
	// rescan mempool to ensure validity of all transactions
	c.Mempool.checkMempool()
	// parse committed block for straw polls
	c.FSM.ParsePollTransactions(blockResult)
	// if self was the proposer
	if bytes.Equal(qc.ProposerKey, c.PublicKey) && !c.isSyncing.Load() {
		// send the certificate results transaction on behalf of the quorum
		c.SendCertificateResultsTx(qc)
	}
	// log the start of the commit
	c.log.Debug("Committing to store")
	// atomically write all from the ephemeral database batch to the actual database
	if _, err = storeI.Commit(); err != nil {
		// exit with error
		return
	}
	// check if the store should partition
	if storeI.ShouldPartition() {
		// if syncing, run the partition synchronously
		if c.isSyncing.Load() {
			storeI.Partition() // TODO: make async design during syncing; worried this may create a temporary P2P deadlock in the future
		} else {
			// execute the partition as a background process to not interrupt consensus
			go storeI.Partition()
		}
	}
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), block.BlockHeader.Height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Metrics, c.log)
	if err != nil {
		// exit with error
		return
	}
	// set up the mempool for the next height
	if c.Mempool.FSM, err = c.FSM.Copy(); err != nil {
		// exit with error
		return
	}
	// execute 'begin block' on mempool FSM to play the inbound transactions at the proper phase of the lifecycle
	if err = c.Mempool.FSM.BeginBlock(); err != nil {
		// exit with error
		return
	}
	// update telemetry
	c.UpdateTelemetry(block, time.Since(start))
	// exit
	return
}

// INTERNAL HELPERS BELOW

// ApplyAndValidateBlock() plays the block against the state machine which returns a result that is compared against the candidate block header
func (c *Controller) ApplyAndValidateBlock(block *lib.Block, commit bool) (b *lib.BlockResult, err lib.ErrorI) {
	// define convenience variables for the block header, hash, and height
	candidate, candidateHash, candidateHeight := block.BlockHeader, lib.BytesToString(block.BlockHeader.Hash), block.BlockHeader.Height
	// check the last qc in the candidate and set it in the ephemeral indexer to prepare for block application
	if err = c.CheckAndSetLastCertificate(candidate); err != nil {
		// exit with error
		return
	}
	// log the start of 'apply block'
	c.log.Debugf("Applying block %s for height %d", candidateHash[:20], candidateHeight)
	// apply the block against the state machine
	compare, txResults, err := c.FSM.ApplyBlock(block)
	if err != nil {
		// exit with error
		return
	}
	// compare the block headers for equality
	compareHash, err := compare.SetHash()
	if err != nil {
		// exit with error
		return
	}
	// use the hash to compare two block headers for equality
	if !bytes.Equal(compareHash, candidate.Hash) {
		return nil, lib.ErrUnequalBlockHash()
	}
	// validate VDF if committing randomly since this randomness is pseudo-non-deterministic (among nodes)
	if commit && compare.Height > 1 && candidate.Vdf != nil {
		// this design has similar security guarantees but lowers the computational requirements at a per-node basis
		if rand.Intn(100) == 0 {
			// validate the VDF included in the block
			if !crypto.VerifyVDF(candidate.LastBlockHash, candidate.Vdf.Output, candidate.Vdf.Proof, int(candidate.Vdf.Iterations)) {
				// exit with vdf error
				return nil, lib.ErrInvalidVDF()
			}
		}
	}
	// log that the proposal is valid
	c.log.Infof("Block %s with %d txs is valid for height %d ✅ ", candidateHash[:20], len(block.Transactions), candidateHeight)
	// exit with the valid results
	return &lib.BlockResult{BlockHeader: candidate, Transactions: txResults}, nil
}

// HandlePeerBlock() validates and handles an inbound certificate (with a block) from a remote peer
func (c *Controller) HandlePeerBlock(msg *lib.BlockMessage, syncing bool) (*lib.QuorumCertificate, lib.ErrorI) {
	// log the start of 'peer block handling'
	c.log.Info("Handling peer block")
	// define a convenience variable for the certificate
	qc := msg.BlockAndCertificate
	// do a basic validation on the QC before loading the committee
	if err := qc.CheckBasic(); err != nil {
		// exit with error
		return nil, err
	}
	// if syncing the blockchain
	if syncing {
		// use checkpoints to protect against long-range attacks
		if qc.Header.Height%CheckpointFrequency == 0 {
			// get the checkpoint from the base chain (or file if independent)
			checkpoint, err := c.RootChainInfo.GetCheckpoint(c.LoadRootChainId(qc.Header.Height), qc.Header.Height, c.Config.ChainId)
			// if getting the checkpoint failed
			if err != nil {
				// warn of the inability to get the checkpoint
				c.log.Warnf(err.Error())
			}
			// if checkpoint fails
			if len(checkpoint) != 0 && !bytes.Equal(qc.BlockHash, checkpoint) {
				// log and kill program
				c.log.Fatalf("Invalid checkpoint %s vs %s at height %d", lib.BytesToString(qc.BlockHash), checkpoint, qc.Header.Height)
			}
		}
	} else {
		// TODO improve logging for LoadCommittee
		// load the committee from the root chain using the root height embedded in the certificate message
		v, err := c.Consensus.LoadCommittee(c.LoadRootChainId(qc.Header.Height), qc.Header.RootHeight)
		if err != nil {
			// exit with error
			return nil, err
		}
		// validate the quorum certificate
		isPartialQC, err := qc.Check(v, c.LoadMaxBlockSize(), &lib.View{NetworkId: c.Config.NetworkID, ChainId: c.Config.ChainId}, false)
		if err != nil {
			// exit with error
			return nil, err
		}
		// if the quorum certificate doesn't have a +2/3rds majority
		if isPartialQC {
			// exit with error
			return nil, lib.ErrNoMaj23()
		}
	}
	// ensure the proposal inside the quorum certificate is valid at a stateless level
	block, err := qc.CheckProposalBasic(c.FSM.Height(), c.Config.NetworkID, c.Config.ChainId)
	if err != nil {
		// exit with error
		return nil, err
	}
	// if this certificate isn't finalized
	if qc.Header.Phase != lib.Phase_PRECOMMIT_VOTE {
		// exit with error
		return nil, lib.ErrWrongPhase()
	}
	// check if the node has fallen out of sync
	if !syncing && c.FSM.Height() != block.BlockHeader.Height {
		// exit with error
		return nil, lib.ErrOutOfSync()
	}
	// attempts to commit the QC to persistence of chain by playing it against the state machine
	if err = c.CommitCertificate(qc, block); err != nil {
		// exit with error
		return nil, err
	}
	// exit
	return qc, nil
}

// CheckAndSetLastCertificate() validates the last quorum certificate included in the block and sets it in the ephemeral indexer
// NOTE: This must come before ApplyBlock in order to have the proposers 'lastCertificate' which is used for distributing rewards
func (c *Controller) CheckAndSetLastCertificate(candidate *lib.BlockHeader) lib.ErrorI {
	if candidate.Height > 1 {
		// load the last quorum certificate from state
		lastCertificate, err := c.FSM.LoadCertificateHashesOnly(candidate.Height - 1)
		// if an error occurred
		if err != nil {
			// exit with error
			return err
		}
		// ensure the candidate 'last certificate' is for the same block and result as the expected
		if !candidate.LastQuorumCertificate.EqualPayloads(lastCertificate) {
			// exit with error
			return lib.ErrInvalidLastQuorumCertificate()
		}
		// define a convenience variable for the 'root height'
		rHeight, height := candidate.LastQuorumCertificate.Header.RootHeight, candidate.LastQuorumCertificate.Header.Height
		// get the committee from the 'root chain' from the n-1 height because state heights represent 'end block state' once committed
		vs, err := c.LoadCommittee(c.LoadRootChainId(height), rHeight) // TODO investigate - during consensus it works without -1 but during syncing might need -1?
		if err != nil {
			// exit with error
			return err
		}
		// ensure the last quorum certificate is valid
		isPartialQC, err := candidate.LastQuorumCertificate.Check(vs, 0, &lib.View{
			Height: candidate.Height - 1, RootHeight: rHeight, NetworkId: c.Config.NetworkID, ChainId: c.Config.ChainId,
		}, true)
		// if the check failed
		if err != nil {
			// exit with error
			return err
		}
		// ensure is a full +2/3rd maj QC
		if isPartialQC {
			return lib.ErrNoMaj23()
		}
		// update the LastQuorumCertificate in the ephemeral store to ensure deterministic last-COMMIT-QC (multiple valid versions can exist)
		if err = c.FSM.Store().(lib.StoreI).IndexQC(candidate.LastQuorumCertificate); err != nil {
			// exit with error
			return err
		}
	}
	// exit
	return nil
}

// SetFSMInConsensusModeForProposals() is how the Validator is configured for `base chain` specific parameter upgrades
func (c *Controller) SetFSMInConsensusModeForProposals() (reset func()) {
	if c.Consensus.GetRound() < 3 {
		// if the node is not having 'consensus issues' refer to the approve list
		c.FSM.SetProposalVoteConfig(fsm.GovProposalVoteConfig_APPROVE_LIST)
		c.Mempool.FSM.SetProposalVoteConfig(fsm.GovProposalVoteConfig_APPROVE_LIST)
	} else {
		// if the node is exhibiting 'chain halt' like behavior, reject all proposals
		c.FSM.SetProposalVoteConfig(fsm.GovProposalVoteConfig_REJECT_ALL)
		c.Mempool.FSM.SetProposalVoteConfig(fsm.GovProposalVoteConfig_REJECT_ALL)
	}
	// a callback that resets the configuration back to default
	reset = func() {
		// the default is to accept all except in 'Consensus mode'
		c.FSM.SetProposalVoteConfig(fsm.AcceptAllProposals)
		c.Mempool.FSM.SetProposalVoteConfig(fsm.AcceptAllProposals)
	}
	return
}

// UpdateTelemetry() updates the prometheus metrics after 'committing' a block
func (c *Controller) UpdateTelemetry(block *lib.Block, blockProcessingTime time.Duration) {
	// get the address of this node
	address := crypto.NewAddressFromBytes(c.Address)
	// update node metrics
	c.Metrics.UpdateNodeMetrics(c.isSyncing.Load())
	// update the block metrics
	c.Metrics.UpdateBlockMetrics(block.BlockHeader.ProposerAddress, len(block.Transactions), blockProcessingTime)
	// update validator metric
	if v, _ := c.FSM.GetValidator(address); v != nil && v.StakedAmount != 0 {
		c.Metrics.UpdateValidator(address.String(), v.StakedAmount, v.UnstakingHeight != 0, v.MaxPausedHeight != 0, v.Delegate, v.Compound)
	}
	// update account metrics
	if a, _ := c.FSM.GetAccount(address); a.Amount != 0 {
		c.Metrics.UpdateAccount(address.String(), a.Amount)
	}
}
