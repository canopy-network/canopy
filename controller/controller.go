package controller

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
)

/* This file contains the 'Controller' implementation which acts as a bus between the bft, p2p, fsm, and store modules to create the node */

var _ bft.Controller = new(Controller)

// Controller acts as the 'manager' of the modules of the application
type Controller struct {
	Address    []byte             // self address
	PublicKey  []byte             // self public key
	PrivateKey crypto.PrivateKeyI // self private key
	Config     lib.Config         // node configuration
	Metrics    *lib.Metrics       // telemetry

	FSM       *fsm.StateMachine // the core protocol component responsible for maintaining and updating the state of the blockchain
	Mempool   *Mempool          // the in memory list of pending transactions
	Consensus *bft.BFT          // the async consensus process between the committee members for the chain
	P2P       *p2p.P2P          // the P2P module the node uses to connect to the network

	RCManager   lib.RCManagerI // the data manager for the 'root chain'
	isSyncing   *atomic.Bool   // is the chain currently being downloaded from peers
	log         lib.LoggerI    // object for logging
	*sync.Mutex                // mutex for thread safety
}

// New() creates a new instance of a Controller, this is the entry point when initializing an instance of a Canopy application
func New(fsm *fsm.StateMachine, c lib.Config, valKey crypto.PrivateKeyI, metrics *lib.Metrics, l lib.LoggerI) (controller *Controller, err lib.ErrorI) {
	address := valKey.PublicKey().Address()
	// load the maximum validators param to set limits on P2P
	maxMembersPerCommittee, err := fsm.GetMaxValidators()
	// if an error occurred when retrieving the max validators
	if err != nil {
		// exit with error
		return
	}
	// initialize the mempool using the FSM copy and the mempool config
	mempool, err := NewMempool(fsm, address, c.MempoolConfig, metrics, l)
	// if an error occurred when creating a new mempool
	if err != nil {
		// exit with error
		return
	}
	// create the controller
	controller = &Controller{
		Address:    address.Bytes(),
		PublicKey:  valKey.PublicKey().Bytes(),
		PrivateKey: valKey,
		Config:     c,
		Metrics:    metrics,
		FSM:        fsm,
		Mempool:    mempool,
		Consensus:  nil,
		P2P:        p2p.New(valKey, maxMembersPerCommittee, metrics, c, l),
		isSyncing:  &atomic.Bool{},
		log:        l,
		Mutex:      &sync.Mutex{},
	}
	// initialize the consensus in the controller, passing a reference to itself
	controller.Consensus, err = bft.New(c, valKey, fsm.Height(), fsm.Height()-1, controller, c.RunVDF, metrics, l)
	// initialize the mempool controller
	mempool.controller = controller
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
	rootChainId, err := c.FSM.GetRootChainId()
	if err != nil {
		c.log.Fatal(err.Error())
	}
	// in a non-blocking sub-function
	go func() {
		// start the P2P module
		c.P2P.Start()
		// log the beginning of the root-chain API connection
		c.log.Warnf("Attempting to connect to the root-chain")
		// set a timer to go off once per second
		t := time.NewTicker(time.Second)
		// once function completes, stop the timer
		defer t.Stop()
		// each time the timer fires
		for range t.C {
			// get the root chain info from the rpc
			rootChainInfo, e := c.RCManager.GetRootChainInfo(rootChainId, c.Config.ChainId)
			if e != nil {
				c.log.Error(e.Error()) // log error but continue
			} else if rootChainInfo != nil && rootChainInfo.Height != 0 {
				// call mempool check
				c.Mempool.CheckMempool(rootChainInfo.Height)
				// update the peer 'must connect'
				c.UpdateP2PMustConnect(rootChainInfo.ValidatorSet)
				// exit the loop
				break
			}
		}
		// start mempool service
		go c.CheckMempool()
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
	// ensure this root chain is active
	activeRootChainId, _ := c.FSM.GetRootChainId()
	// if inactive
	if activeRootChainId != info.RootChainId {
		c.log.Debugf("Detected inactive root-chain update at rootChainId=%d", info.RootChainId)
		return
	}
	// if the last validator set is empty
	if info.LastValidatorSet == nil || len(info.LastValidatorSet.ValidatorSet) == 0 {
		// signal to reset consensus and start a new height
		c.Consensus.ResetBFT <- bft.ResetBFT{IsRootChainUpdate: false, RootHeight: info.Height, BFTMeta: &lib.BFTCoordinationMeta{
			LastCommitTime: info.BlockTimeInfo.GetEstCommitTime(),
		}}
	} else {
		// signal to reset consensus
		c.Consensus.ResetBFT <- bft.ResetBFT{IsRootChainUpdate: true, RootHeight: info.Height, BFTMeta: &lib.BFTCoordinationMeta{
			LastCommitTime: info.BlockTimeInfo.GetEstCommitTime(),
		}}
	}
	// update the peer 'must connect'
	c.UpdateP2PMustConnect(info.ValidatorSet)
}

// LoadCommittee() gets the ValidatorSet that is authorized to come to Consensus agreement on the Proposal for a specific height/chainId
func (c *Controller) LoadCommittee(rootChainId, rootHeight uint64) (lib.ValidatorSet, lib.ErrorI) {
	return c.RCManager.GetValidatorSet(rootChainId, c.Config.ChainId, rootHeight)
}

// LoadRootChainOrderBook() gets the order book from the root-chain
func (c *Controller) LoadRootChainOrderBook(rootChainId, rootHeight uint64) (*lib.OrderBook, lib.ErrorI) {
	return c.RCManager.GetOrders(rootChainId, rootHeight, c.Config.ChainId)
}

// LoadRootBlockTime() gets the root chain block timing information
func (c *Controller) LoadRootBlockTime() (*lib.BlockTimeInfo, lib.ErrorI) {
	activeRootChainId, _ := c.FSM.GetRootChainId()
	return c.RCManager.BlockTime(activeRootChainId, 0)
}

// GetRootChainLotteryWinner() gets the pseudorandomly selected delegate to reward and their cut
func (c *Controller) GetRootChainLotteryWinner(fsm *fsm.StateMachine, rootHeight uint64) (winner *lib.LotteryWinner, err lib.ErrorI) {
	// get the root chain id from the state machine
	rootChainId, err := fsm.LoadRootChainId(c.ChainHeight())
	// if an error occurred retrieving the id
	if err != nil {
		// exit with error
		return nil, err
	}
	// execute the remote call
	return c.RCManager.GetLotteryWinner(rootChainId, rootHeight, c.Config.ChainId)
}

// IsValidDoubleSigner() checks if the double signer is valid at a certain double sign height
func (c *Controller) IsValidDoubleSigner(rootChainId, rootHeight uint64, address []byte) bool {
	// do a remote call to the root chain to see if the double signer is valid
	isValidDoubleSigner, err := c.RCManager.IsValidDoubleSigner(rootChainId, rootHeight, lib.BytesToString(address))
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
func (c *Controller) LoadMinimumEvidenceHeight(rootChainId, rootHeight uint64) (*uint64, lib.ErrorI) {
	return c.RCManager.GetMinimumEvidenceHeight(rootChainId, rootHeight)
}

// NextBFTCommitTime() returns the estimated next CommitTime()
func (c *Controller) NextBFTCommitTime() uint64 {
	blockTime := time.Duration(c.Config.BlockTimeMS()) * time.Millisecond
	nextBFTCommitTime := c.Consensus.LastCommitTime.Add(blockTime)
	c.log.Infof("Next BFT commit time: %s", nextBFTCommitTime.Format("15:04:05"))
	//if nextBFTStartTime.Add(-1*blockTime).Before(time.Now()) || time.Until(nextBFTStartTime) > blockTime {
	//	c.log.Warnf("Next BFT commit time too early (%s), using time.Now(%s)", nextBFTStartTime.Format("15:04:05"),
	//		time.Now().Format("15:04:05"))
	//	return uint64(time.Now().UnixMicro())
	//}
	return uint64(nextBFTCommitTime.UnixMicro())
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

// LoadProposerKeys() gets the last root-chainId proposer keys
func (c *Controller) LoadLastProposers(height uint64) (*lib.Proposers, lib.ErrorI) {
	// load the last proposers as determined by the last 5 quorum certificates
	return c.FSM.LoadLastProposers(height)
}

// LoadCommitteeData() returns the state metadata for the 'self chain'
func (c *Controller) LoadCommitteeData() (data *lib.CommitteeData, err lib.ErrorI) {
	// get the committee data from the FSM
	return c.FSM.GetCommitteeData(c.Config.ChainId)
}

// Syncing() returns if any of the supported chains are currently syncing
func (c *Controller) Syncing() *atomic.Bool { return c.isSyncing }

// ResetFSM() resets the underlying state machine to last valid state
func (c *Controller) ResetFSM() { c.FSM.Reset() }

// RootChainHeight() returns the height of the canopy root-chain
func (c *Controller) RootChainHeight() uint64 {
	chainId, err := c.FSM.GetRootChainId()
	if err != nil {
		c.log.Error(err.Error())
	}
	return c.RCManager.GetHeight(chainId)
}

// ChainHeight() returns the height of this target chain
func (c *Controller) ChainHeight() uint64 { return c.FSM.Height() }

// BlockTime() returns block time information about this chain
func (c *Controller) BlockTime(h uint64) (*lib.BlockTimeInfo, lib.ErrorI) {
	return c.FSM.LoadBlockTime(h)
}

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
