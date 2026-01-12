package controller

import (
	"bytes"
	"context"
	"math/rand"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
)

/* This file contains the high level functionality of block / proposal processing */

// ListenForBlock() listens for inbound block messages, internally routes them, and gossips them to peers
func (c *Controller) ListenForBlock() {
	// log the beginning of the 'block listener' service
	c.log.Debug("Listening for inbound blocks")
	// initialize a cache that prevents duplicate messages and  create a map of peers that signal 'new block'
	cache, syncDetector := lib.NewMessageCache(), lib.NewBlockTracker(c.Sync, c.log)
	// wait and execute for each inbound message received
	for msg := range c.P2P.Inbox(Block) {
		// create a variable to signal a 'stop loop'
		var quit bool
		// wrap in a function call to use 'defer' functionality
		func() {
			// lock the controller to prevent multi-thread conflicts
			c.Lock()
			// when iteration completes, unlock
			defer c.Unlock()
			// add a convenience variable to track the sender
			sender := msg.Sender.Address.PublicKey
			// check and add the message to the cache to prevent duplicates
			if ok := cache.Add(msg); !ok {
				// if fallen out of sync
				quit = syncDetector.AddIfHas(sender, msg.Message, c.P2P.PeerCount())
				// exit iteration
				return
			}
			c.log.Debug("Handling block message")
			// log the receipt of the block message
			c.log.Infof("Received new block from %s ✉️", lib.BytesToTruncatedString(sender))
			// try to unmarshal the message to a block message
			blockMessage := new(lib.BlockMessage)
			if err := lib.Unmarshal(msg.Message, blockMessage); err != nil {
				// log the error
				c.log.Debug("Invalid Peer Block Message")
				// slash the peer's reputation
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit iteration
				return
			}
			// 'handle' the peer block and certificate appropriately
			qc, err := c.HandlePeerBlock(blockMessage, false)
			// ensure no error
			if err != nil {
				// if new height notified
				if err.Error() == lib.ErrNewHeight().Error() {
					// if fallen out of sync
					if quit = syncDetector.Add(sender, msg.Message, blockMessage.BlockAndCertificate.Header.Height, c.P2P.PeerCount()); quit {
						// exit iteration
						return
					}
				}
				// log the error
				c.log.Warnf("Peer block invalid:\n%s", err.Error())
				// slash the peer's reputation
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit iteration
				return
			}
			// if not syncing - gossip the block
			if !c.Syncing().Load() {
				// gossip the block to our peers
				c.GossipBlock(qc, sender, blockMessage.Time)
				// signal a reset to the bft module
				c.Consensus.ResetBFT <- bft.ResetBFT{StartTime: time.UnixMicro(int64(blockMessage.Time))}
			}
			// reset 'syncDetector' because a new block was received properly
			syncDetector.Reset()
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
func (c *Controller) GossipBlock(certificate *lib.QuorumCertificate, senderPubToExclude []byte, timestamp uint64) {
	// log the start of the gossip block function
	c.log.Debugf("Gossiping certificate: %s", lib.BytesToString(certificate.ResultsHash))
	// create the block message to gossip
	blockMessage := &lib.BlockMessage{
		ChainId:             c.Config.ChainId,
		BlockAndCertificate: certificate,
		Time:                timestamp,
	}
	// send the block message to all peers excluding the sender (gossip)
	if err := c.P2P.SendToPeers(Block, blockMessage, lib.BytesToString(senderPubToExclude)); err != nil {
		c.log.Errorf("unable to gossip block with err: %s", err.Error())
	}
}

// SelfSendBlock() gossips a QuorumCertificate (with block) through the P2P network for handling
func (c *Controller) SelfSendBlock(qc *lib.QuorumCertificate, timestamp uint64) {
	// create the block message
	blockMessage := &lib.BlockMessage{
		ChainId:             c.Config.ChainId,
		BlockAndCertificate: qc,
		Time:                timestamp,
	}
	// internally route the block to the 'block inbox'
	if err := c.P2P.SelfSend(c.PublicKey, Block, blockMessage); err != nil {
		c.log.Errorf("unable to gossip block with err: %s", err.Error())
	}
}

// BFT FUNCTIONS BELOW

// ProduceProposal() create a proposal in the form of a `block` and `certificate result` for the bft process
func (c *Controller) ProduceProposal(evidence *bft.ByzantineEvidence, vdf *crypto.VDF) (rcBuildHeight uint64, blockBytes []byte, results *lib.CertificateResult, err lib.ErrorI) {
	c.log.Debugf("Producing proposal as leader")
	// once done proposing, 'reset' the proposal mode back to default to 'accept all'
	defer c.FSM.Reset()
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
	// get the proposal cached in the mempool
	p := c.GetProposalBlockFromMempool()
	// load the last block from the indexer
	lastBlock, err := c.FSM.LoadBlock(c.FSM.Height() - 1)
	if err != nil {
		return
	}
	// replace the VDF and last certificate in the header
	p.Block.BlockHeader.LastQuorumCertificate, p.Block.BlockHeader.Vdf = lastCertificate, vdf
	p.Block.BlockHeader.TotalVdfIterations = vdf.GetIterations() + lastBlock.BlockHeader.TotalVdfIterations
	// set the certificate results variable and rcBuildHeight
	results, rcBuildHeight = p.CertResults, p.rcBuildHeight
	// execute the hash
	if _, err = p.Block.BlockHeader.SetHash(); err != nil {
		// exit with error
		return
	}
	// convert the block reference to bytes
	blockBytes, err = lib.Marshal(p.Block)
	if err != nil {
		// exit with error
		return
	}
	// update the 'block results' with the newly created header
	p.BlockResult.BlockHeader = p.Block.BlockHeader
	// IMPORTANT: none of the calls below should rely on the latest FSM because
	// the mempool FSM was discarded during 'check mempool'
	// set slash recipients (this is necessary because values changed)
	c.CalculateSlashRecipients(results, evidence)
	// set checkpoint (this is necessary because values changed)
	c.CalculateCheckpoint(p.BlockResult, results)
	// exit
	return
}

// ValidateProposal() fully validates a proposal in the form of a quorum certificate and resets back to begin block state
func (c *Controller) ValidateProposal(rcBuildHeight uint64, qc *lib.QuorumCertificate, evidence *bft.ByzantineEvidence) (blockResult *lib.BlockResult, err lib.ErrorI) {
	// reset the mempool at the beginning of the function to preserve the state for CommitCertificate()
	c.FSM.Reset()
	// log the beginning of proposal validation
	c.log.Debugf("Validating proposal from leader")
	// configure the FSM in 'consensus mode' for validator proposals
	resetProposalConfig := c.SetFSMInConsensusModeForProposals()
	// once done proposing, 'reset' the proposal mode back to default to 'accept all'
	defer resetProposalConfig()
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
	blockResult, err = c.ApplyAndValidateBlock(block, false)
	if err != nil {
		// exit with error
		return
	}
	// create a comparable certificate results (includes reward recipients, slash recipients, swap commands, etc)
	compareResults := c.NewCertificateResults(c.FSM, block, blockResult, evidence, rcBuildHeight)
	// ensure generated the same results
	if !qc.Results.Equals(compareResults) {
		// exit with error
		return nil, fsm.ErrMismatchCertResults()
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
func (c *Controller) CommitCertificate(qc *lib.QuorumCertificate, block *lib.Block, blockResult *lib.BlockResult, ts uint64) (err lib.ErrorI) {
	start := time.Now()
	// cancel any running mempool check
	c.Mempool.stop()
	// lock the mempool
	c.Mempool.L.Lock()
	defer c.Mempool.L.Unlock()
	// log the beginning of the commit
	c.log.Debugf("TryCommit block %s", lib.BytesToString(qc.ResultsHash))
	// cast the store to ensure the proper store type to complete this operation
	storeI := c.FSM.Store().(lib.StoreI)
	// reset the store once this code finishes; if code execution gets to `store.Commit()` - this will effectively be a noop
	defer c.FSM.Reset()
	// if the block result isn't 'pre-calculated'
	if blockResult == nil {
		// reset the FSM to ensure stale proposal validations don't come into play
		c.FSM.Reset()
		// apply the block against the state machine
		blockResult, err = c.ApplyAndValidateBlock(block, true)
		if err != nil {
			// exit with error
			return
		}
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
	// delete each transaction from the mempool
	c.Mempool.DeleteTransaction(block.Transactions...)
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
		return err
	}
	// log to signal finishing the commit
	c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), block.BlockHeader.Height)
	// set up the finite state machine for the next height
	c.FSM, err = fsm.New(c.Config, storeI, c.Metrics, c.log)
	if err != nil {
		// exit with error
		return err
	}
	// reset the current mempool store to prepare for the next height
	c.Mempool.FSM.Discard()
	// set up the mempool with the actual new FSM for the next height
	// this makes c.Mempool.FSM.Reset() is unnecessary
	if c.Mempool.FSM, err = c.FSM.Copy(); err != nil {
		// exit with error
		return err
	}
	// check the mempool to cache a proposal block and validate the mempool itself
	c.Mempool.CheckMempool()
	// reset mempool FSM
	c.Mempool.FSM.Reset()
	// update telemetry (using proper defer to ensure time.Since is evaluated at defer execution)
	defer c.UpdateTelemetry(qc, block, time.Since(start))
	// publish root chain information to all nested chain subscribers.
	for _, id := range c.RCManager.ChainIds() {
		// get the root chain info
		info, e := c.FSM.LoadRootChainInfo(id, 0)
		if e != nil {
			// don't log 'no-validators' error as this is possible
			if e.Error() != lib.ErrNoValidators().Error() {
				c.log.Error(e.Error())
			}
			continue
		}
		// set the timestamp
		info.Timestamp = ts
		// publish root chain information
		go c.RCManager.Publish(id, info)
	}
	// publish block data to all block data subscribers
	go c.RCManager.PublishBlockData(block.BlockHeader.Height)
	// exit
	return
}

// CommitCertificate() the experimental and parallelized version of the above
func (c *Controller) CommitCertificateParallel(qc *lib.QuorumCertificate, block *lib.Block, blockResult *lib.BlockResult, ts uint64) (err lib.ErrorI) {
	start := time.Now()
	// cancel any running mempool check
	c.Mempool.stop()
	// lock the mempool
	c.Mempool.L.Lock()
	defer c.Mempool.L.Unlock()
	// log the beginning of the commit
	c.log.Debugf("TryCommit block %s", lib.BytesToString(qc.ResultsHash))
	// cast the store to ensure the proper store type to complete this operation
	storeI := c.FSM.Store().(lib.StoreI)
	// reset the store once this code finishes; if code execution gets to `store.Commit()` - this will effectively be a noop
	defer c.FSM.Reset()
	// if the block result isn't 'pre-calculated'
	if blockResult == nil {
		// reset the FSM to ensure stale proposal validations don't come into play
		c.FSM.Reset()
		// apply the block against the state machine
		blockResult, err = c.ApplyAndValidateBlock(block, true)
		if err != nil {
			// exit with error
			return
		}
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
	// create an ephemeral store copy for the mempool
	memPoolStore, err := storeI.Copy()
	if err != nil {
		return err
	}
	// increase the version number of the ephemeral store
	memPoolStore.IncreaseVersion()
	// delete each transaction from the mempool
	c.Mempool.DeleteTransaction(block.Transactions...)
	// parse committed block for straw polls
	c.FSM.ParsePollTransactions(blockResult)
	// if self was the proposer
	if bytes.Equal(qc.ProposerKey, c.PublicKey) && !c.isSyncing.Load() {
		// send the certificate results transaction on behalf of the quorum
		c.SendCertificateResultsTx(qc)
	}
	// create an error group to run the commit and mempool update in parallel
	eg := errgroup.Group{}
	eg.Go(func() error {
		// log the start of the commit
		c.log.Debug("Committing to store")
		// atomically write all from the ephemeral database batch to the actual database
		if _, err = storeI.Commit(); err != nil {
			// exit with error
			return err
		}
		// log to signal finishing the commit
		c.log.Infof("Committed block %s at H:%d 🔒", lib.BytesToTruncatedString(qc.BlockHash), block.BlockHeader.Height)
		// set up the finite state machine for the next height
		c.FSM, err = fsm.New(c.Config, storeI, c.Metrics, c.log)
		if err != nil {
			// exit with error
			return err
		}
		// publish root chain information to all nested chain subscribers.
		for _, id := range c.RCManager.ChainIds() {
			// get the root chain info
			info, e := c.FSM.LoadRootChainInfo(id, 0)
			if e != nil {
				// don't log 'no-validators' error as this is possible
				if e.Error() != lib.ErrNoValidators().Error() {
					c.log.Error(e.Error())
				}
				continue
			}
			// publish root chain information
			// set the timestamp
			info.Timestamp = ts
			go c.RCManager.Publish(id, info)
		}
		// publish block data to all block data subscribers
		go c.RCManager.PublishBlockData(block.BlockHeader.Height)
		// exit
		return nil
	})
	eg.Go(func() error {
		// set up the mempool for the next height with the temporary FSM
		c.Mempool.FSM, err = fsm.New(c.Config, memPoolStore, c.Metrics, c.log)
		if err != nil {
			// exit with error
			return err
		}
		// check the mempool to cache a proposal block and validate the mempool itself
		c.Mempool.CheckMempool()
		// discard the temporary store after checking the mempool
		memPoolStore.Discard()
		// exit
		return nil
	})
	// check for any errors while committing and checking the mempool
	if e := eg.Wait(); e != nil {
		return e.(lib.ErrorI)
	}
	// reset the current mempool store to prepare for the next height
	c.Mempool.FSM.Discard()
	// set up the mempool with the actual new FSM for the next height
	// this makes c.Mempool.FSM.Reset() is unnecessary
	if c.Mempool.FSM, err = c.FSM.Copy(); err != nil {
		// exit with error
		return err
	}
	// update telemetry (using proper defer to ensure time.Since is evaluated at defer execution)
	defer c.UpdateTelemetry(qc, block, time.Since(start))
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
	compare, results, err := c.FSM.ApplyBlock(context.Background(), block, false)
	if err != nil {
		// exit with error
		return
	}
	// if any transactions failed
	if len(results.Failed) != 0 {
		for _, f := range results.Failed {
			c.log.Errorf("From: %s\nType:%s\nErr:%s", f.Address, f.Transaction.MessageType, f.Error.Error())
		}
		return nil, lib.ErrFailedTransactions()
	}
	// compare the block headers for equality
	compareHash, err := compare.SetHash()
	if err != nil {
		// exit with error
		return
	}
	// use the hash to compare two block headers for equality
	if !bytes.Equal(compareHash, candidate.Hash) {
		c.debugDumpHeaderDiff(candidate, compare)
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
	return &lib.BlockResult{BlockHeader: candidate, Transactions: results.Results, Events: results.Events}, nil
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
			// attempt to load the checkpoint from the file
			checkpoint := c.checkpointFromFile(qc.Header.Height, qc.Header.ChainId)
			// if checkpoint loading from file failed
			if checkpoint == nil {
				var err lib.ErrorI
				// get the checkpoint from the base chain (or file if independent)
				checkpoint, err = c.RCManager.GetCheckpoint(c.LoadRootChainId(qc.Header.Height), qc.Header.Height, c.Config.ChainId)
				// if getting the checkpoint failed
				if err != nil {
					// warn of the inability to get the checkpoint
					c.log.Warnf(err.Error())
				}
			}
			// if checkpoint fails
			if len(checkpoint) != 0 && !bytes.Equal(qc.BlockHash, checkpoint) {
				// log and kill program
				c.log.Fatalf("Invalid checkpoint %s vs %s at height %d", lib.BytesToString(qc.BlockHash), checkpoint, qc.Header.Height)
			}
		}
	} else {
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
		// update the non signer percent for the validators
		c.Metrics.UpdateNonSignerPercent(qc.Signature, v)
	}
	// ensure the proposal inside the quorum certificate is valid at a stateless level
	block, err := qc.CheckProposalBasic(c.FSM.Height(), c.Config.NetworkID, c.Config.ChainId)
	// if this certificate isn't finalized
	if err == nil && qc.Header.Phase != lib.Phase_PRECOMMIT_VOTE {
		// exit with error
		return nil, lib.ErrWrongPhase()
	}
	if err != nil {
		// exit with error
		return nil, err
	}
	// create a temp variable to double-check our saved block result against the peers
	result := c.Consensus.BlockResult
	// if our cached result is the same as the peer block, use that
	if result == nil || result.BlockHeader == nil || !bytes.Equal(result.BlockHeader.Hash, block.BlockHeader.Hash) {
		result = nil
	}
	// attempts to commit the QC to persistence of chain by playing it against the state machine
	if err = c.CommitCertificate(qc, block, result, msg.Time); err != nil {
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
		// the synced blocks were already validated during consensus, no need to validate again
		if !c.Syncing().Load() {
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
	elapsed := time.Since(c.Consensus.BFTStartTime).Milliseconds()
	// if consensus is below round 3 AND it hasn't been more than 3 minutes since the last block
	if c.Consensus.GetRound() < 3 && elapsed < int64(c.Config.BlockTimeMS()*3) {
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
func (c *Controller) UpdateTelemetry(qc *lib.QuorumCertificate, block *lib.Block, blockProcessingTime time.Duration) {
	// create convenience variables
	address, vdfIterations := crypto.NewAddressFromBytes(c.Address), uint64(0)
	// attempt to get VDF iterations
	if block.BlockHeader.Vdf != nil {
		vdfIterations = block.BlockHeader.Vdf.Iterations
	}
	// update node metrics
	c.Metrics.UpdateNodeMetrics(c.isSyncing.Load())
	// update the block metrics
	c.Metrics.UpdateBlockMetrics(block.BlockHeader.ProposerAddress, uint64(len(qc.Block)), block.BlockHeader.NumTxs, vdfIterations, blockProcessingTime)
	// update validator metric
	if v, _ := c.FSM.GetValidator(address); v != nil && v.StakedAmount != 0 {
		isProducer, nonSigners, doubleSigners := c.getValidatorBehaviorMetrics(address, qc)
		c.Metrics.UpdateValidator(address.String(), v.StakedAmount, v.UnstakingHeight != 0, v.MaxPausedHeight != 0, v.Delegate, v.Compound, isProducer, nonSigners, doubleSigners)
	}
	// update account metrics
	if a, _ := c.FSM.GetAccount(address); a.Amount != 0 {
		c.Metrics.UpdateAccount(address.String(), a.Amount)
	}
}

// getValidatorBehaviorMetrics() gets metrics for validator behavior in this block
func (c *Controller) getValidatorBehaviorMetrics(address crypto.AddressI, qc *lib.QuorumCertificate) (isProducer bool, nonSigners map[string]uint64, doubleSigners []crypto.AddressI) {
	nonSigners = make(map[string]uint64)
	// 1. Track block producer
	if proposerPubKey, err := crypto.NewPublicKeyFromBytes(qc.ProposerKey); err == nil {
		isProducer = proposerPubKey.Address().Equals(address)
	}

	// 2. Track non-signers - only if we have the validator set for this chain
	if ns, err := c.FSM.GetNonSigners(); err == nil {
		for _, n := range ns {
			nonSigners[lib.BytesToString(n.Address)] = n.Counter
		}
	}

	// 3. Track double signers (if evidence is available in the QC results)
	// TODO this call is really inefficient - we need to segment double signers by block or address
	if doubleSigner, err := c.FSM.GetDoubleSigners(); err == nil {
		// for each double signer
		for _, ds := range doubleSigner {
			// for each height
			for _, height := range ds.Heights {
				// if double signed on the last height
				if height == qc.Header.Height {
					// update list
					doubleSigners = append(doubleSigners, crypto.NewAddress(ds.Id))
				}
			}
		}
	}
	return
}

// debugDumpHeaderDiff() logs the differences between the candidate and the constructed
func (c *Controller) debugDumpHeaderDiff(candidate, compare *lib.BlockHeader) {
	cand, _ := lib.MarshalJSONIndentString(candidate)
	comp, _ := lib.MarshalJSONIndentString(compare)
	exported, _ := c.FSM.ExportState()
	state, _ := lib.MarshalJSONIndentString(exported)
	c.log.Errorf("Candidate:\n:%s", cand)
	c.log.Errorf("Compare:\n:%s", comp)
	c.log.Errorf("State:\n:%s", state)
}
