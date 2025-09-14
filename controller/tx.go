package controller

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/bft"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	lru "github.com/hashicorp/golang-lru/v2"
)

/* This file implements logic for transaction sending and handling as well as memory pooling */

// SendTxMsg() routes a locally generated transaction message to the listener for processing + gossiping
func (c *Controller) SendTxMsg(tx []byte) lib.ErrorI {
	// create a transaction message object using the tx bytes and the chain id
	msg := &lib.TxMessage{ChainId: c.Config.ChainId, Txs: [][]byte{tx}}
	// send the transaction message to the listener using internal routing
	return c.P2P.SelfSend(c.PublicKey, Tx, msg)
}

// ListenForTx() listen for inbound tx messages, internally route them, and gossip them to peers
func (c *Controller) ListenForTx() {
	// create a new message cache to filter out duplicate transaction messages
	cache := lib.NewMessageCache()
	// wait and execute for each inbound transaction message
	for msg := range c.P2P.Inbox(Tx) {
		// if the chain is syncing, just return without handling
		if c.isSyncing.Load() {
			// exit
			continue
		}
		func() {
			// check and add the message to the cache to prevent duplicates
			if ok := cache.Add(msg); !ok {
				// if duplicate, exit
				return
			}
			c.log.Debug("Handling transaction")
			// create a convenience variable for the identity of the sender
			senderID := msg.Sender.Address.PublicKey
			// try to unmarshal the p2p message as a tx message
			txMsg := new(lib.TxMessage)
			if err := lib.Unmarshal(msg.Message, txMsg); err != nil {
				// log the unexpected behavior
				c.log.Warnf("Non-Tx message from %s", lib.BytesToTruncatedString(senderID))
				// slash the peer's reputation score
				c.P2P.ChangeReputation(senderID, p2p.InvalidMsgRep)
				// exit
				return
			}
			// if the message is empty
			if txMsg == nil {
				// log the unexpected behavior
				c.log.Warnf("Empty tx message from %s", lib.BytesToTruncatedString(senderID))
				// slash the peers reputation score
				c.P2P.ChangeReputation(senderID, p2p.InvalidMsgRep)
				// exit
				return
			}
			// route the transactions to the mempool handler
			if err := c.Mempool.HandleTransactions(txMsg.Txs...); err != nil {
				// else - warn of the error
				c.log.Warnf("Handle tx from %s failed with err: %s", lib.BytesToTruncatedString(senderID), err.Error())
				// slash the peers reputation score
				c.P2P.ChangeReputation(senderID, p2p.InvalidTxRep)
				// exit
				return
			}
			// onto the next message
		}()
	}
}

// GetProposalBlockFromMempool() returns the cached proposal block
func (c *Controller) GetProposalBlockFromMempool() *CachedProposal {
	return c.Mempool.cachedProposal.Load().(*CachedProposal)
}

// CheckMempool() periodically checks the mempool:
// - Validates all transactions in the mempool
// - Caches a proposal block based on the current state and the mempool transactions
// - P2P Gossip out any transactions that weren't previously gossiped
func (c *Controller) CheckMempool() {
	deDupe, _ := lru.New[string, struct{}](100_000)
	// if configured to not check mempool besides right after CommitBlock
	if c.Config.LazyMempoolCheckFrequencyS == 0 {
		return
	}
	for {
		// get root chain height
		c.Lock()
		rcBuildHeight := c.RootChainHeight()
		c.Unlock()
		// keep a list of transaction needing to be gossipped
		var toGossip [][]byte
		// if recheck is necessary
		if c.Mempool.recheck.Load() {
			// execute in a function call to allow defer
			func() {
				c.Mempool.L.Lock()
				defer c.Mempool.L.Unlock()
				// be mempool strict on proposals
				resetProposalConfig := c.SetFSMInConsensusModeForProposals()
				// once done proposing, 'reset' the proposal mode back to default to 'accept all'
				defer func() { resetProposalConfig() }()
				// reset the mempool
				c.Mempool.FSM.Reset()
				// calculate rc build height
				ownRoot, err := c.Mempool.FSM.LoadIsOwnRoot()
				if err != nil {
					c.log.Error(err.Error())
				}
				// if ownRoot
				if ownRoot {
					rcBuildHeight = c.Mempool.FSM.Height()
				}
				// check the mempool to cache a proposal block and validate the mempool itself
				c.Mempool.CheckMempool(rcBuildHeight)
				// get the transactions to gossip
				toGossip = c.Mempool.GetTransactions(math.MaxUint64)
				// set recheck to false
				c.Mempool.recheck.Store(false)
			}()
		}
		// for each transaction to gossip
		var dedupedTxs [][]byte
		for _, tx := range toGossip {
			// get the key for the transaction
			key := crypto.HashString(tx)
			// if not already gossiped
			if _, found := deDupe.Get(key); !found {
				// add to the de-dupe list
				deDupe.Add(key, struct{}{})
				dedupedTxs = append(dedupedTxs, tx)
			}
		}
		if len(dedupedTxs) != 0 {
			// gossip the transactions to peers
			if err := c.P2P.SendToPeers(Tx, &lib.TxMessage{ChainId: c.Config.ChainId, Txs: dedupedTxs}); err != nil {
				// log the gossip error
				c.log.Error(fmt.Sprintf("unable to gossip tx with err: %s", err.Error()))
			}
		}
		// sleep for the recheck time
		time.Sleep(time.Duration(c.Config.LazyMempoolCheckFrequencyS) * time.Second)
	}
}

// Mempool accepts or rejects incoming txs based on the mempool (ephemeral copy) state
// - recheck when
//   - mempool dropped some percent of the lowest fee txs
//   - new tx has higher fee than the lowest
//
// - notes:
//   - new tx added may also be evicted, this is expected behavior
type Mempool struct {
	controller      *Controller
	lib.Mempool                        // the memory pool itself defined as an interface
	L               *sync.Mutex        // thread safety at the mempool level
	FSM             *fsm.StateMachine  // the ephemeral finite state machine used to validate inbound transactions
	cachedResults   lib.TxResults      // a memory cache of transaction results for the json rpc
	cachedFailedTxs *lib.FailedTxCache // a memory cache of failed transactions for tracking
	metrics         *lib.Metrics       // telemetry
	address         crypto.AddressI    // validator identity
	cachedProposal  atomic.Value       // the cached block proposal set when mempool is 'checked'
	recheck         atomic.Bool        // a signal to recheck the mempool
	stop            context.CancelFunc // the cancellable context of the mempool
	log             lib.LoggerI        // the logger
}

type CachedProposal struct {
	Block         *lib.Block
	BlockResult   *lib.BlockResult
	CertResults   *lib.CertificateResult
	rcBuildHeight uint64
}

// NewMempool() creates a new instance of a Mempool structure
func NewMempool(fsm *fsm.StateMachine, address crypto.AddressI, config lib.MempoolConfig, metrics *lib.Metrics, log lib.LoggerI) (m *Mempool, err lib.ErrorI) {
	// initialize the structure
	m = &Mempool{
		Mempool:         lib.NewMempool(config),
		L:               &sync.Mutex{},
		cachedProposal:  atomic.Value{},
		recheck:         atomic.Bool{},
		cachedFailedTxs: lib.NewFailedTxCache(),
		metrics:         metrics,
		address:         address,
		log:             log,
	}
	// make an 'mempool (ephemeral copy) state' so the mempool can maintain only 'valid' transactions despite dependencies and conflicts
	m.FSM, err = fsm.Copy()
	// if an error occurred copying the fsm
	if err != nil {
		return nil, err
	}
	// exit
	return m, err
}

// HandleTransactions() attempts to add a transaction to the mempool by validating, adding, and evicting overfull or newly invalid txs
func (m *Mempool) HandleTransactions(tx ...[]byte) (err lib.ErrorI) {
	// lock the mempool
	m.L.Lock()
	defer m.L.Unlock()
	// signal a recheck
	m.recheck.Store(true)
	// add a transaction to the mempool
	if err = m.AddTransactions(tx...); err != nil {
		// exit with the error
		return
	}
	// exit
	return
}

// CheckMempool() Checks each transaction in the mempool and caches a block proposal
func (m *Mempool) CheckMempool(rcBuildHeight uint64) {
	m.log.Info("Validating mempool and caching a new proposal block")
	var err lib.ErrorI
	// create the actual block structure with the maximum amount of transactions allowed or available in the mempool
	block := &lib.Block{
		BlockHeader:  &lib.BlockHeader{Time: m.controller.LastBFTCommitTime(), ProposerAddress: m.address.Bytes()},
		Transactions: m.GetTransactions(math.MaxUint64), // get all transactions in mempool - but apply block will only keep 'max-block' amount
	}
	// capture the tentative block result using a new object reference
	blockResult, oversized, failed := new(lib.BlockResult), make([]*lib.TxResult, 0), make([]*lib.FailedTx, 0)
	// setup a context with cancel
	ctx, stop := context.WithCancel(context.Background())
	// set the cancel function
	m.stop = stop
	// apply the block against the state machine and populate the resulting merkle `roots` in the block header
	lastVs := m.controller.LastValidatorSet[m.FSM.Height()][m.controller.Config.ChainId]
	block.BlockHeader, blockResult.Transactions, oversized, failed, err = m.FSM.ApplyBlock(ctx, block, lastVs, true)
	if err != nil {
		m.log.Warnf("Check Mempool error: %s", err.Error())
		return
	}
	// set the block result block header
	blockResult.BlockHeader = block.BlockHeader
	// cache the proposal
	m.cachedProposal.Store(&CachedProposal{
		Block:         block,
		BlockResult:   blockResult,
		CertResults:   m.controller.NewCertificateResults(m.FSM, block, blockResult, &bft.ByzantineEvidence{DSE: bft.DoubleSignEvidences{}}, rcBuildHeight),
		rcBuildHeight: rcBuildHeight,
	})
	// create a cache of failed tx bytes to evict from the mempool
	var failedTxBz [][]byte
	// mark as failed in the cache
	for _, tx := range failed {
		// cache failed txs for RPC display
		m.cachedFailedTxs.Add(tx)
		// save the bytes
		failedTxBz = append(failedTxBz, tx.GetBytes())
	}
	// evict all invalid transactions from the mempool
	m.DeleteTransaction(failedTxBz...)
	// log a warning
	if len(failed) != 0 {
		m.log.Warnf("Removed failed %d txs from mempool", len(failed))
	}
	// reset the RPC cached results
	m.cachedResults = nil
	// add results to cache
	for _, result := range blockResult.Transactions {
		// cache the results
		m.cachedResults = append(m.cachedResults, result)
	}
	// add results to cache
	for _, result := range oversized {
		// cache the results
		result.Index = uint64(len(m.cachedResults))
		m.cachedResults = append(m.cachedResults, result)
	}
	m.log.Info("Done checking mempool")
	// update the mempool metrics
	m.metrics.UpdateMempoolMetrics(m.Mempool.TxCount(), m.Mempool.TxsBytes())
}

// GetPendingPage() returns a page of unconfirmed mempool transactions
func (c *Controller) GetPendingPage(p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// lock the controller for thread safety
	c.Lock()
	// unlock the controller when the function completes
	defer c.Unlock()
	// create a new page and transaction results list to populate
	page, txResults := lib.NewPage(p, lib.PendingResultsPageName), make(lib.TxResults, 0)
	// define a callback to execute when loading the page
	callback := func(item any) (e lib.ErrorI) {
		// cast the item to a transaction result pointer
		v, ok := item.(*lib.TxResult)
		// if the cast failed
		if !ok {
			// exit with error
			return lib.ErrInvalidMessageCast()
		}
		// add to the list
		txResults = append(txResults, v)
		// exit callback
		return
	}
	// populate the page using the 'cached results'
	err = page.LoadArray(c.Mempool.cachedResults, &txResults, callback)
	// exit
	return
}

// GetFailedTxsPage() returns a list of failed mempool transactions
func (c *Controller) GetFailedTxsPage(address string, p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// lock the controller for thread safety
	c.Lock()
	// unlock the controller when the function completes
	defer c.Unlock()
	// create a new page and failed transaction results list to populate
	page, failedTxs := lib.NewPage(p, lib.FailedTxsPageName), make(lib.FailedTxs, 0)
	// define a callback to execute when loading the page
	callback := func(item any) (e lib.ErrorI) {
		// cast the item to a 'failed transaction' object
		v, ok := item.(*lib.FailedTx)
		// if the cast failed
		if !ok {
			// exit with error
			return lib.ErrInvalidMessageCast()
		}
		// add to the failed list
		failedTxs = append(failedTxs, v)
		// exit callback
		return
	}
	// populate the page using the 'failed cache'
	err = page.LoadArray(c.Mempool.cachedFailedTxs.GetFailedForAddress(address), &failedTxs, callback)
	// exit
	return
}
