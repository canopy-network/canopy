package controller

import (
	"bytes"
	"fmt"
	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	"math/rand"
	"strings"
	"time"
)

/* This file contains the high level functionality of the continued agreement on the blocks of the chain */

// Sync() downloads the blockchain from peers until 'synced' to the latest 'height'
// 1) Get the height and begin block params from the state_machine
// 2) Get peer max_height from P2P
// 3) Ask peers for a block at a time
// 4) CheckBasic each block by checking if cert was signed by +2/3 maj (COMMIT message)
// 5) Commit block against the state machine, if error fatal
// 5) Do this until reach the max-peer-height
// 6) Stay on top by listening to incoming cert messages
func (c *Controller) Sync() {
	// log the initialization of the syncing process
	c.log.Infof("Sync started 🔄 for committee %d", c.Config.ChainId)
	// set the Controller as 'syncing'
	c.isSyncing.Store(true)
	// check if node is alone in the validator set
	if c.singleNodeNetwork() {
		// complete syncing
		c.finishSyncing()
		// exit
		return
	}
	// poll max height of all peers
	maxHeight, minVDFIterations, syncingPeers := c.pollMaxHeight(1)
	// while still below the latest height
	for !c.syncingDone(maxHeight, minVDFIterations) {
		// get a random peer to send a 'block request' to
		requested, _ := lib.StringToBytes(syncingPeers[rand.Intn(len(syncingPeers))])
		// log the initialization of the block request
		c.log.Infof("Syncing height %d 🔄 from %s", c.FSM.Height(), lib.BytesToTruncatedString(requested))
		// send the request to the
		c.RequestBlock(false, requested)
		// block until one of the two cases happens
		select {
		// a) got a block in the inbox
		case msg := <-c.P2P.Inbox(Block):
			// if the responder does not equal the requester
			responder := msg.Sender.Address.PublicKey
			// log the receipt of a 'block response'
			c.log.Debugf("Received a block response msg from %s", lib.BytesToTruncatedString(responder))
			// check to see if the 'responder' is who was 'requested'
			if !bytes.Equal(responder, requested) {
				// log this unexpected behavior
				c.log.Warn("unexpected sender")
				// slash the reputation of the unexpected responder
				c.P2P.ChangeReputation(responder, p2p.UnexpectedBlockRep)
				// exit the select to re-poll
				break
			}
			// cast the message to a block message
			blockMessage, ok := msg.Message.(*lib.BlockMessage)
			// if the cast fails
			if !ok {
				// log this unexpected behavior
				c.log.Warn("Not a block response msg")
				// slash the reputation of the peer
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit the select to re-poll
				break
			}
			// process the block message received from the peer
			if _, err := c.HandlePeerBlock(blockMessage, true); err != nil {
				// log this unexpected behavior
				c.log.Warnf("Syncing peer block invalid:\n%s", err.Error())
				// slash the reputation of the peer
				c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidBlockRep)
				// exit the select to re-poll
				break
			}
			// each peer is individually polled for 'max height' in each request
			// if the max height has grown, we accept that as the new max height
			if blockMessage.MaxHeight > maxHeight && blockMessage.TotalVdfIterations >= minVDFIterations {
				// log the update
				c.log.Debugf("Updated chain %d with max height: %d and iterations %d", c.Config.ChainId, maxHeight, minVDFIterations)
				// update the max height and vdf iterations
				maxHeight, minVDFIterations = blockMessage.MaxHeight, blockMessage.TotalVdfIterations
			}
			// success, increase the peer reputation
			c.P2P.ChangeReputation(responder, p2p.GoodBlockRep)
			// execute another iteration without polling peers
			continue
		// b) a timeout occurred before a block landed in the inbox
		case <-time.After(p2p.SyncTimeoutS * time.Second):
			// log the timeout
			c.log.Warnf("Timeout waiting for sync block")
			// slash the peer reputation
			c.P2P.ChangeReputation(requested, p2p.TimeoutRep)
		}
		// update the syncing peers and poll the peers for their max height + minimum vdf iterations
		maxHeight, minVDFIterations, syncingPeers = c.pollMaxHeight(1)
	}
	// log 'sync complete'
	c.log.Info("Synced to top ✅")
	// signal that the node is synced to top
	c.finishSyncing()
}

// SUBSCRIBERS BELOW

// ListenForConsensus() listens and internally routes inbound consensus messages
func (c *Controller) ListenForConsensus() {
	// wait and execute for each consensus message received
	for msg := range c.P2P.Inbox(Cons) {
		// if the node is syncing
		if c.isSyncing.Load() {
			// disregard the consensus message
			continue
		}
		// execute in a sub-function to unify error handling and enable 'defer' functionality
		if err := func() (err lib.ErrorI) {
			// lock the controller for thread safety
			c.Lock()
			// once the handler completes, unlock
			defer c.Unlock()
			// try to cast the message to a 'consensus message'
			consensusMessage, ok := msg.Message.(*lib.ConsensusMessage)
			// if cast unsuccessful
			if !ok {
				// exit with error
				return
			}
			// create a new bft message object reference to ensure non nil results
			bftMsg := new(bft.Message)
			// populate the object reference with the payload bytes of the message
			if err = lib.Unmarshal(consensusMessage.Message, bftMsg); err != nil {
				// exit with error
				return
			}
			// route the message to the consensus module
			if err = c.Consensus.HandleMessage(bftMsg); err != nil {
				// exit with error
				return
			}
			// exit
			return
		}(); err != nil {
			// log the error
			c.log.Errorf("Handling consensus message failed with err: %s", err.Error())
			// slash the reputation of the peer
			c.P2P.ChangeReputation(msg.Sender.Address.PublicKey, p2p.InvalidMsgRep)
		}
	}
}

// ListenForBlockRequests() listen for inbound block request messages from syncing peers, handles and answer them
func (c *Controller) ListenForBlockRequests() {
	// initialize a rate limiter for the inbound syncing messages
	l := lib.NewLimiter(p2p.MaxBlockReqPerWindow, c.P2P.MaxPossiblePeers()*p2p.MaxBlockReqPerWindow, p2p.BlockReqWindowS)
	// for the lifetime of the Controller
	for {
		// select one of the following cases
		select {
		// wait and execute for each inbound block request
		case msg := <-c.P2P.Inbox(BlockRequest):
			// wrap in a sub-function to enable 'defer' functionality
			func() {
				// lock the controller for thread safety
				c.Lock()
				// unlock once the message handling completes
				defer c.Unlock()
				// create a convenience variable for the sender of the block request
				senderID := msg.Sender.Address.PublicKey
				// check with the rate limiter to see if *this peer* or *all peers* are blocked
				blocked, allBlocked := l.NewRequest(lib.BytesToString(senderID))
				// if *this peer* or *all peers* are blocked
				if blocked || allBlocked {
					// if only this specific peer is blocked, slash the reputation
					if blocked {
						// log a warning about this peer that had to be rate-limited
						c.log.Warnf("Rate-limit hit for peer %s", lib.BytesToTruncatedString(senderID))
						// slash the peer's reputation
						c.P2P.ChangeReputation(senderID, p2p.BlockReqExceededRep)
					}
					// exit this iteration
					return
				}
				// try to cast the p2p msg to a block request message
				request, ok := msg.Message.(*lib.BlockRequestMessage)
				// if the cast fails
				if !ok {
					// log a warning about the failed cast
					c.log.Warnf("Invalid block-request msg from peer %s", lib.BytesToTruncatedString(senderID))
					// slash the peer's reputation
					c.P2P.ChangeReputation(senderID, p2p.InvalidMsgRep)
					// exit
					return
				}
				// log the receipt of a block request from a peer
				c.log.Debugf("Received a block request from %s", lib.BytesToTruncatedString(senderID))
				// create variables that will be populated if the request is more than 'height only'
				var (
					certificate *lib.QuorumCertificate
					err         error
				)
				// if the requesting more than just the height
				if !request.HeightOnly {
					// load the actual certificate (with block) and populate the variable
					certificate, err = c.LoadCertificate(request.Height)
					// if an error occurred
					if err != nil {
						// log the error
						c.log.Error(err.Error())
						// exit the iteration
						return
					}
				}
				// log to mark the response initialization
				c.log.Debugf("Responding to a block request from %s, heightOnly=%t", lib.BytesToString(senderID[:20]), request.HeightOnly)
				// send the block back to the requester
				c.SendBlock(c.FSM.Height(), c.FSM.TotalVDFIterations(), certificate, senderID)
			}()
		// limiter is ready to be reset
		case <-l.TimeToReset():
			// reset the limiter
			l.Reset()
		}
	}
}

// PUBLISHERS BELOW

// SendToReplicas() directly send a bft message to each validator in a set (committee)
func (c *Controller) SendToReplicas(replicas lib.ValidatorSet, msg lib.Signable) {
	// log the initialization of the send process
	c.log.Debugf("Sending to %d replicas", replicas.NumValidators)
	// sign the consensus message
	signedMessage, err := c.signConsensusMessage(msg)
	if err != nil {
		// log the error
		c.log.Error(err.Error())
		// exit
		return
	}
	// for each replica (validator) in the set
	for _, replica := range replicas.ValidatorSet.ValidatorSet {
		// check if replica is self
		if bytes.Equal(replica.PublicKey, c.PublicKey) {
			// send the message to self using internal routing
			if err = c.P2P.SelfSend(c.PublicKey, Cons, signedMessage); err != nil {
				// log the error
				c.log.Error(err.Error())
			}
		} else {
			// if not self, send directly to peer using P2P
			if err = c.P2P.SendTo(replica.PublicKey, Cons, signedMessage); err != nil {
				// log the error (warning is used in case 'some' replicas are not reachable)
				c.log.Warn(err.Error())
			}
		}
	}
}

// SendToProposer() sends a bft message to the leader of the Consensus round
func (c *Controller) SendToProposer(msg lib.Signable) {
	// sign the consensus message
	signedMessage, err := c.signConsensusMessage(msg)
	if err != nil {
		// log the error
		c.log.Error(err.Error())
		// exit
		return
	}
	// check if sending to 'self' or peer
	if c.Consensus.SelfIsProposer() {
		// send using internal routing
		if err = c.P2P.SelfSend(c.PublicKey, Cons, signedMessage); err != nil {
			// log the error
			c.log.Error(err.Error())
		}
	} else {
		// handle peer send
		if err = c.P2P.SendTo(c.Consensus.ProposerKey, Cons, signedMessage); err != nil {
			// log the error
			c.log.Error(err.Error())
		}
	}
}

// RequestBlock() sends a block request to peer(s) - `heightOnly` is a request for just the peer's max height
func (c *Controller) RequestBlock(heightOnly bool, recipients ...[]byte) {
	// define a convenience variable for the current height
	height := c.FSM.Height()
	// if the optional 'recipients' is specified
	if len(recipients) != 0 {
		// for each 'recipient' specified
		for _, pk := range recipients {
			// log the block request
			c.log.Debugf("Requesting block %d for chain %d from %s", height, c.Config.ChainId, lib.BytesToTruncatedString(pk))
			// send it to exactly who was specified in the function call
			if err := c.P2P.SendTo(pk, BlockRequest, &lib.BlockRequestMessage{
				ChainId:    c.Config.ChainId,
				Height:     height,
				HeightOnly: heightOnly,
			}); err != nil {
				// log error
				c.log.Error(err.Error())
			}
		}
	} else {
		// log the block request
		c.log.Debugf("Requesting block %d for chain %d from all", height, c.Config.ChainId)
		// send it to the peers
		if err := c.P2P.SendToPeers(BlockRequest, &lib.BlockRequestMessage{
			ChainId:    c.Config.ChainId,
			Height:     height,
			HeightOnly: heightOnly,
		}); err != nil {
			// log error
			c.log.Error(err.Error())
		}
	}
}

// SendBlock() responds to a `blockRequest` message to a peer - always sending the self.MaxHeight and sometimes sending the actual block and supporting QC
func (c *Controller) SendBlock(maxHeight, vdfIterations uint64, blockAndCert *lib.QuorumCertificate, recipient []byte) {
	// send the block to the recipient public key specified
	if err := c.P2P.SendTo(recipient, Block, &lib.BlockMessage{
		ChainId:             c.Config.ChainId,
		MaxHeight:           maxHeight,
		TotalVdfIterations:  vdfIterations,
		BlockAndCertificate: blockAndCert,
	}); err != nil {
		// log error
		c.log.Error(err.Error())
	}
}

// INTERNAL HELPERS BELOW

// UpdateP2PMustConnect() tells the P2P module which nodes are *required* to be connected to (usually fellow committee members or none if not in committee)
func (c *Controller) UpdateP2PMustConnect() {
	// resolve the port to append based on the 'chain id'
	port, err := lib.ResolvePort(c.Config.ChainId)
	// if an error occurred
	if err != nil {
		// log the error
		c.log.Error(err.Error())
		// exit
		return
	}
	// handle empty validator set
	if c.RootChainInfo.ValidatorSet.ValidatorSet == nil {
		// exit
		return
	}
	// define tracking variables for the 'must connect' peer list and if 'self' is a validator
	mustConnects, selfIsValidator := make([]*lib.PeerAddress, 0), false
	// for each member of the committee
	for _, member := range c.RootChainInfo.ValidatorSet.ValidatorSet.ValidatorSet {
		// if self is a validator
		if bytes.Equal(member.PublicKey, c.PublicKey) {
			// update the variable
			selfIsValidator = true
		}
		// create the peer object and add it to the list
		mustConnects = append(mustConnects, &lib.PeerAddress{
			PublicKey:  member.PublicKey,
			NetAddress: strings.ReplaceAll(member.NetAddress, "tcp://", "") + port,
			PeerMeta:   &lib.PeerMeta{ChainId: c.Config.ChainId},
		})
	}
	// if this node 'is validator'
	if selfIsValidator {
		// log the must connect update
		c.log.Infof("Updating must connects with %d validators", len(mustConnects))
		// send the list to the p2p module
		c.P2P.MustConnectsReceiver <- mustConnects
	}
}

// pollMaxHeight() polls all peers for their local MaxHeight and totalVDFIterations for a specific chainId
// NOTE: unlike other P2P transmissions - RequestBlock enforces a minimum reputation on `mustConnects`
// to ensure a byzantine validator cannot cause syncing issues above max_height
func (c *Controller) pollMaxHeight(backoff int) (max, minVDF uint64, syncingPeerList []string) {
	// initialize max height and minimumVDFIterations to -1
	maxHeight, minimumVDFIterations := -1, -1
	// empty inbox to start fresh
	c.emptyInbox(Block)
	// log the initialization
	c.log.Infof("Polling chain peers for max height")
	// initialize the syncing peers list
	syncingPeerList = make([]string, 0)
	// ask only for 'max height' from all peers
	c.RequestBlock(true)
	// debug log the current status
	c.log.Debug("Waiting for peer max heights")
	// loop until timeout case
	for {
		// block until one of the cases is satisfied
		select {
		// handle the inbound message
		case m := <-c.P2P.Inbox(Block):
			// cast the inbound message payload as a block message
			blockMessage, ok := m.Message.(*lib.BlockMessage)
			// if the cast fails
			if !ok {
				// log the unexpected behavior
				c.log.Warnf("Invalid block message response from %s", lib.BytesToTruncatedString(m.Sender.Address.PublicKey))
				// slash the peer reputation
				c.P2P.ChangeReputation(m.Sender.Address.PublicKey, p2p.InvalidMsgRep)
				// reset loop
				continue
			}
			// log the receipt of the block message
			c.log.Debugf("Received a block response from peer %s with max height at %d", lib.BytesToTruncatedString(m.Sender.Address.PublicKey), maxHeight)
			// don't listen to any peers below the minimumVDFIterations
			if int(blockMessage.TotalVdfIterations) < minimumVDFIterations {
				// log the status
				c.log.Warnf("Ignoring below the minimum vdf iterations")
				// reset loop
				continue
			}
			// update the minimum vdf iterations
			minimumVDFIterations = int(blockMessage.TotalVdfIterations)
			// add to syncing peer list
			syncingPeerList = append(syncingPeerList, lib.BytesToString(m.Sender.Address.PublicKey))
			// if the maximum height is exceeded, update the max height
			if int(blockMessage.MaxHeight) > maxHeight {
				// reset syncing variables if peer exceeds the previous minimum vdf iterations
				maxHeight = int(blockMessage.MaxHeight)
			}
		// if a timeout occurred
		case <-time.After(p2p.PollMaxHeightTimeoutS * time.Second * time.Duration(backoff)):
			// if the maximum height or vdf iterations remains unset
			if maxHeight == -1 || minimumVDFIterations == -1 {
				// log the status of no heights received
				c.log.Warn("No heights received from peers. Trying again")
				// try again with greater backoff
				return c.pollMaxHeight(backoff + 1)
			}
			// log the max height among the peers
			c.log.Debugf("Peer max height is %d 🔝", maxHeight)
			// return the max height, minimum vdf iterations, and list of syncing peers
			return uint64(maxHeight), uint64(minimumVDFIterations), syncingPeerList
		}
	}
}

// singleNodeNetwork() returns true if there are no other participants in the committee besides self
func (c *Controller) singleNodeNetwork() bool {
	// if self is the only validator, return true
	return c.RootChainInfo.ValidatorSet.NumValidators == 0 || c.RootChainInfo.ValidatorSet.NumValidators == 1 &&
		bytes.Equal(c.RootChainInfo.ValidatorSet.ValidatorSet.ValidatorSet[0].PublicKey, c.PublicKey)
}

// syncingDone() checks if the syncing loop may complete for a specific chainId
func (c *Controller) syncingDone(maxHeight, minVDFIterations uint64) bool {
	// if the plugin height is GTE the max height
	if c.FSM.Height() >= maxHeight {
		// ensure node did not lie about VDF iterations in their chain
		if c.FSM.TotalVDFIterations() < minVDFIterations {
			// if the node lied, on unsafe fork - exit application immediately for safety
			c.log.Fatalf("Unsafe fork detected - VDFIterations error: localVDFIterations: %d, minimumVDFIterations: %d", c.FSM.TotalVDFIterations(), minVDFIterations)
		}
		// exit with syncing done
		return true
	}
	// exit with syncing not done
	return false
}

// finishSyncing() is called when the syncing loop is completed for a specific chainId
func (c *Controller) finishSyncing() {
	// lock the controller for thread safety
	c.Lock()
	// when function completes, unlock
	defer c.Unlock()
	// signal a reset of bft for the chain
	c.Consensus.ResetBFT <- bft.ResetBFT{ProcessTime: time.Since(c.LoadLastCommitTime(c.FSM.Height()))}
	// set syncing to false
	c.isSyncing.Store(false)
	// enable listening for a block
	go c.ListenForBlock()
}

// signConsensusMessage() signs, encodes, and wraps a consensus message in preparation for sending
func (c *Controller) signConsensusMessage(msg lib.Signable) (*lib.ConsensusMessage, lib.ErrorI) {
	// sign the message
	if err := msg.Sign(c.PrivateKey); err != nil {
		return nil, err
	}
	// convert the message to bytes
	messageBytes, err := lib.Marshal(msg)
	if err != nil {
		return nil, err
	}
	// wrap the message in consensus
	return &lib.ConsensusMessage{
		ChainId: c.Config.ChainId,
		Message: messageBytes,
	}, nil
}

// ConsensusSummary() for the RPC - returns the summary json object of the bft for a specific chainID
func (c *Controller) ConsensusSummary() ([]byte, lib.ErrorI) {
	// lock for thread safety
	c.Lock()
	defer c.Unlock()
	// convert self public key from bytes into an object
	selfKey, _ := crypto.NewPublicKeyFromBytes(c.PublicKey)
	// create the consensus summary object
	consensusSummary := &ConsensusSummary{
		Syncing:              c.isSyncing.Load(),
		View:                 c.Consensus.View,
		Locked:               c.Consensus.HighQC != nil,
		Address:              selfKey.Address().Bytes(),
		PublicKey:            c.PublicKey,
		Proposer:             c.Consensus.ProposerKey,
		Proposals:            c.Consensus.Proposals,
		PartialQCs:           c.Consensus.PartialQCs,
		PacemakerVotes:       c.Consensus.PacemakerMessages,
		MinimumPowerFor23Maj: c.Consensus.ValidatorSet.MinimumMaj23,
		Votes:                c.Consensus.Votes,
		Status:               "",
	}
	consensusSummary.BlockHash = c.Consensus.BlockHash
	// if exists, populate the proposal hash
	if c.Consensus.Results != nil {
		consensusSummary.ResultsHash = c.Consensus.Results.Hash()
	}
	// if high qc exists, populate the block hash and results hash
	if c.Consensus.HighQC != nil {
		consensusSummary.BlockHash = c.Consensus.BlockHash
		consensusSummary.ResultsHash = c.Consensus.HighQC.ResultsHash
	}
	// if exists, populate the proposer address
	if c.Consensus.ProposerKey != nil {
		propKey, _ := crypto.NewPublicKeyFromBytes(c.Consensus.ProposerKey)
		consensusSummary.ProposerAddress = propKey.Address().Bytes()
	}
	// create a status string
	switch c.Consensus.View.Phase {
	case bft.Election, bft.Propose, bft.Precommit, bft.Commit:
		proposal := c.Consensus.GetProposal()
		if proposal == nil {
			consensusSummary.Status = "waiting for proposal"
		} else {
			consensusSummary.Status = "received proposal"
		}
	case bft.ElectionVote, bft.ProposeVote, bft.CommitProcess:
		if bytes.Equal(c.Consensus.ProposerKey, c.PublicKey) {
			_, _, votedPercentage := c.Consensus.GetLeadingVote()
			consensusSummary.Status = fmt.Sprintf("received %d%% of votes", votedPercentage)
		} else {
			consensusSummary.Status = "voting on proposal"
		}
	}
	// convert the object into json
	return lib.MarshalJSONIndent(&consensusSummary)
}

// ConsensusSummary is simply a json informational structure about the local status of the BFT
type ConsensusSummary struct {
	Syncing              bool                   `json:"isSyncing"`
	View                 *lib.View              `json:"view"`
	BlockHash            lib.HexBytes           `json:"blockHash"`
	ResultsHash          lib.HexBytes           `json:"resultsHash"`
	Locked               bool                   `json:"locked"`
	Address              lib.HexBytes           `json:"address"`
	PublicKey            lib.HexBytes           `json:"publicKey"`
	ProposerAddress      lib.HexBytes           `json:"proposerAddress"`
	Proposer             lib.HexBytes           `json:"proposer"`
	Proposals            bft.ProposalsForHeight `json:"proposals"`
	PartialQCs           bft.PartialQCs         `json:"partialQCs"`
	PacemakerVotes       bft.PacemakerMessages  `json:"pacemakerVotes"`
	MinimumPowerFor23Maj uint64                 `json:"minimumPowerFor23Maj"`
	Votes                bft.VotesForHeight     `json:"votes"`
	Status               string                 `json:"status"`
}
