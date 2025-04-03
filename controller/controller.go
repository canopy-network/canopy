package controller

import (
	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	"sync"
	"sync/atomic"
	"time"
)

/* This file contains the 'Controller' implementation which acts as a bus between the bft, p2p, fsm, and store modules to create the node */

var _ bft.Controller = new(Controller)

// Controller acts as the 'manager' of the modules of the application
type Controller struct {
	Address    []byte             // self address
	PublicKey  []byte             // self public key
	PrivateKey crypto.PrivateKeyI // self private key
	Config     lib.Config         // node configuration

	FSM       *fsm.StateMachine // the core protocol component responsible for maintaining and updating the state of the blockchain
	Mempool   *Mempool          // the in memory list of pending transactions
	Consensus *bft.BFT          // the async consensus process between the committee members for the chain
	P2P       *p2p.P2P          // the P2P module the node uses to connect to the network

	RootChainInfo lib.RootChainInfo // the latest information from the 'root chain'
	isSyncing     *atomic.Bool      // is the chain currently being downloaded from peers
	log           lib.LoggerI       // object for logging
	sync.Mutex                      // mutex for thread safety
}

// New() creates a new instance of a Controller, this is the entry point when initializing an instance of a Canopy application
func New(fsm *fsm.StateMachine, c lib.Config, valKey crypto.PrivateKeyI, l lib.LoggerI) (controller *Controller, err lib.ErrorI) {
	// load the maximum validators param to set limits on P2P
	maxMembersPerCommittee, err := fsm.GetMaxValidators()
	// if an error occurred when retrieving the max validators
	if err != nil {
		// exit with error
		return
	}
	// initialize the mempool using the FSM copy and the mempool config
	mempool, err := NewMempool(fsm, c.MempoolConfig, l)
	// if an error occurred when creating a new mempool
	if err != nil {
		// exit with error
		return
	}
	// create the controller
	controller = &Controller{
		FSM:           fsm,
		Mempool:       mempool,
		isSyncing:     &atomic.Bool{},
		P2P:           p2p.New(valKey, maxMembersPerCommittee, c, l),
		Address:       valKey.PublicKey().Address().Bytes(),
		PublicKey:     valKey.PublicKey().Bytes(),
		PrivateKey:    valKey,
		Config:        c,
		log:           l,
		RootChainInfo: lib.RootChainInfo{Log: l},
		Mutex:         sync.Mutex{},
	}
	// initialize the consensus in the controller, passing a reference to itself
	controller.Consensus, err = bft.New(c, valKey, fsm.Height(), fsm.Height()-1, controller, true, l)
	// if an error occurred initializing the bft module
	if err != nil {
		// exit with error
		return
	}
	// exit
	return
}

// Start() begins the Controller service
func (c *Controller) Start() {
	// start the P2P module
	c.P2P.Start()
	// start internal Controller listeners for P2P
	c.StartListeners()
	// in a non-blocking sub-function
	go func() {
		// log the beginning of the root-chain API connection
		c.log.Warnf("Attempting to connect to the root-chain")
		// set a timer to go off once per second
		t := time.NewTicker(time.Second)
		// once function completes, stop the timer
		defer t.Stop()
		// each time the timer fires
		for range t.C {
			// if the root chain height is updated
			if c.RootChainInfo.GetHeight() != 0 {
				break
			}
		}
		// start internal Controller listeners for P2P
		c.StartListeners()
		// start the syncing process (if not synced to top)
		go c.Sync()
		// start the bft consensus (if synced to top)
		go c.Consensus.Start()
	}()
}

// StartListeners() runs all listeners on separate threads
func (c *Controller) StartListeners() {
	c.log.Debug("Listening for inbound txs, block requests, and consensus messages")
	// listen for syncing peers
	go c.ListenForBlockRequests()
	// listen for inbound consensus messages
	go c.ListenForConsensus()
	// listen for inbound
	go c.ListenForTx()
	// ListenForBlock() is called once syncing finished
}

// Stop() terminates the Controller service
func (c *Controller) Stop() {
	// lock the controller
	c.Lock()
	// unlock when the function completes
	defer c.Unlock()
	// stop the store module
	if err := c.FSM.Store().(lib.StoreI).Close(); err != nil {
		c.log.Error(err.Error())
	}
	// stop the p2p module
	c.P2P.Stop()
}

// ROOT CHAIN CALLS BELOW

// UpdateRootChainInfo() receives updates from the root-chain thread
func (c *Controller) UpdateRootChainInfo(info *lib.RootChainInfo) {
	c.log.Debugf("Updating root chain info")
	// lock the controller for thread safety
	c.Lock()
	// unlock when the function completes
	defer c.Unlock()
	// use the logger from the controller
	info.Log = c.RootChainInfo.Log
	// use the get remote callback 'callback' from the controller
	info.GetRemoteCallbacks = c.RootChainInfo.GetRemoteCallbacks
	// update the root chain info
	c.RootChainInfo = *info
	// if the last validator set is empty
	if info.LastValidatorSet.NumValidators == 0 {
		// signal to reset consensus and start a new height
		c.Consensus.ResetBFT <- bft.ResetBFT{IsRootChainUpdate: false}
	} else {
		// signal to reset consensus
		c.Consensus.ResetBFT <- bft.ResetBFT{IsRootChainUpdate: true}
	}
	// update the peer 'must connect'
	c.UpdateP2PMustConnect()
}

// LoadCommittee() gets the ValidatorSet that is authorized to come to Consensus agreement on the Proposal for a specific height/chainId
func (c *Controller) LoadCommittee(rootChainId, rootHeight uint64) (lib.ValidatorSet, lib.ErrorI) {
	return c.RootChainInfo.GetValidatorSet(rootChainId, c.Config.ChainId, rootHeight)
}

// LoadRootChainOrderBook() gets the order book from the root-chain
func (c *Controller) LoadRootChainOrderBook(rootHeight uint64) (*lib.OrderBook, lib.ErrorI) {
	return c.RootChainInfo.GetOrders(c.LoadRootChainId(c.ChainHeight()), rootHeight, c.Config.ChainId)
}

// GetRootChainLotteryWinner() gets the pseudorandomly selected delegate to reward and their cut
func (c *Controller) GetRootChainLotteryWinner(rootHeight uint64) (winner *lib.LotteryWinner, err lib.ErrorI) {
	// get the root chain id from the state machine
	rootChainId, err := c.FSM.LoadRootChainId(c.ChainHeight())
	// if an error occurred retrieving the id
	if err != nil {
		// exit with error
		return nil, err
	}
	// execute the remote call
	return c.RootChainInfo.GetLotteryWinner(rootChainId, rootHeight, c.Config.ChainId)
}

// IsValidDoubleSigner() checks if the double signer is valid at a certain double sign height
func (c *Controller) IsValidDoubleSigner(rootHeight uint64, address []byte) bool {
	// do a remote call to the root chain to see if the double signer is valid
	isValidDoubleSigner, err := c.RootChainInfo.IsValidDoubleSigner(c.LoadRootChainId(c.ChainHeight()), rootHeight, lib.BytesToString(address))
	// if an error occurred during the remote call
	if err != nil {
		// log the error
		c.log.Errorf("IsValidDoubleSigner failed with error: %s", err.Error())
		// return is not a valid double signer for safety
		return false
	}
	// return the result from the remote call
	return *isValidDoubleSigner
}

// INTERNAL CALLS BELOW

// LoadIsOwnRoot() returns if this chain is its own root (base)
func (c *Controller) LoadIsOwnRoot() (isOwnRoot bool) {
	// use the state machine to check if this chain is the root chain
	isOwnRoot, err := c.FSM.LoadIsOwnRoot()
	// if an error occurred
	if err != nil {
		// log the error
		c.log.Error(err.Error())
	}
	// exit
	return
}

// RootChainId() returns the root chain id according to the FSM
func (c *Controller) LoadRootChainId(height uint64) (rootChainId uint64) {
	// use the state machine to get the root chain id
	rootChainId, err := c.FSM.LoadRootChainId(height)
	// if an error occurred
	if err != nil {
		// log the error
		c.log.Error(err.Error())
	}
	// exit
	return
}

// LoadCertificate() gets the certificate for from the indexer at a specific height
func (c *Controller) LoadCertificate(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	return c.FSM.LoadCertificate(height)
}

// LoadMinimumEvidenceHeight() gets the minimum evidence height from the finite state machine
func (c *Controller) LoadMinimumEvidenceHeight() (uint64, lib.ErrorI) {
	return c.FSM.LoadMinimumEvidenceHeight()
}

// LoadMaxBlockSize() gets the max block size from the state
func (c *Controller) LoadMaxBlockSize() int {
	// load the maximum block size from the nested chain FSM
	params, _ := c.FSM.GetParamsCons()
	// if the parameters are empty
	if params == nil {
		// return 0 as the 'max'
		return 0
	}
	// return the max block size as set by the governance param
	return int(params.BlockSize)
}

// LoadLastCommitTime() gets a timestamp from the most recent Quorum Block
func (c *Controller) LoadLastCommitTime(height uint64) time.Time {
	// load the certificate (and block) from the indexer
	cert, err := c.FSM.LoadCertificate(height)
	if err != nil {
		c.log.Error(err.Error())
		return time.Time{}
	}
	// create a new object reference (to ensure a non-nil result)
	block := new(lib.Block)
	// populate the object reference with bytes
	if err = lib.Unmarshal(cert.Block, block); err != nil {
		// log the error
		c.log.Error(err.Error())
		// exit with empty time
		return time.Time{}
	}
	// ensure the block isn't nil
	if block.BlockHeader == nil {
		// log the error
		c.log.Error("Last block synced is nil")
		// exit with empty time
		return time.Time{}
	}
	// return the last block time
	return time.UnixMicro(int64(block.BlockHeader.Time))
}

// LoadProposerKeys() gets the last root-chainId proposer keys
func (c *Controller) LoadLastProposers(height uint64) (*lib.Proposers, lib.ErrorI) {
	// load the last proposers as determined by the last 5 quorum certificates
	return c.FSM.LoadLastProposers(height)
}

// LoadCommitteeData() returns the state metadata for the 'self chain'
func (c *Controller) LoadCommitteeData() (data *lib.CommitteeData, err lib.ErrorI) {
	// get the committee data from the FSM
	return c.FSM.LoadCommitteeData(c.ChainHeight(), c.Config.ChainId)
}

// Syncing() returns if any of the supported chains are currently syncing
func (c *Controller) Syncing() *atomic.Bool { return c.isSyncing }

// RootChainHeight() returns the height of the canopy root-chain
func (c *Controller) RootChainHeight() uint64 { return c.RootChainInfo.GetHeight() }

// ChainHeight() returns the height of this target chain
func (c *Controller) ChainHeight() uint64 { return c.FSM.Height() }

// emptyInbox() discards all unread messages for a specific topic
func (c *Controller) emptyInbox(topic lib.Topic) {
	// for each message in the inbox
	for len(c.P2P.Inbox(topic)) > 0 {
		// discard the message
		<-c.P2P.Inbox(topic)
	}
}

// convenience aliases that reference the library package
const (
	BlockRequest = lib.Topic_BLOCK_REQUEST
	Block        = lib.Topic_BLOCK
	Tx           = lib.Topic_TX
	Cons         = lib.Topic_CONSENSUS
)
