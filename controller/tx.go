package controller

import (
	"fmt"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	"math"
	"time"
)

/* This file implements logic for transaction sending and handling as well as memory pooling */

// SendTxMsg() routes a locally generated transaction message to the listener for processing + gossiping
func (c *Controller) SendTxMsg(tx []byte) lib.ErrorI {
	// create a transaction message object using the tx bytes and the chain id
	msg := &lib.TxMessage{ChainId: c.Config.ChainId, Tx: tx}
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
			c.log.Debug("Handling transaction")
			defer lib.TimeTrack("ListenForTx", time.Now())
			// lock the controller for thread safety
			c.Lock()
			// unlock when this iteration completes
			defer c.Unlock()
			// check and add the message to the cache to prevent duplicates
			if ok := cache.Add(msg); !ok {
				// if duplicate, exit
				return
			}
			// create a convenience variable for the identity of the sender
			senderID := msg.Sender.Address.PublicKey
			// try to cast the p2p message as a tx message
			txMsg, ok := msg.Message.(*lib.TxMessage)
			// if the cast failed
			if !ok {
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
			// route the transaction to the mempool handler
			if err := c.HandleTransaction(txMsg.Tx); err != nil {
				// if the error is 'mempool already has it'
				if err.Error() == lib.ErrTxFoundInMempool(crypto.HashString(txMsg.Tx)).Error() {
					// exit
					return
				}
				// else - warn of the error
				c.log.Warnf("Handle tx from %s failed with err: %s", lib.BytesToTruncatedString(senderID), err.Error())
				// slash the peers reputation score
				c.P2P.ChangeReputation(senderID, p2p.InvalidTxRep)
				// exit
				return
			}
			// log the receipt of the valid transaction
			c.log.Infof("Received valid transaction %s from %s for chain %d", crypto.ShortHashString(txMsg.Tx), lib.BytesToTruncatedString(senderID), txMsg.ChainId)
			// bump peer reputation positively
			c.P2P.ChangeReputation(senderID, p2p.GoodTxRep)
			// gossip the transaction to peers
			if err := c.P2P.SendToPeers(Tx, msg.Message, lib.BytesToString(senderID)); err != nil {
				// log the gossip error
				c.log.Error(fmt.Sprintf("unable to gossip tx with err: %s", err.Error()))
			}
			// onto the next message
		}()
	}
}

// HandleTransaction() accepts or rejects inbound txs based on the mempool state
// - pass through call checking indexer and mempool for duplicate
func (c *Controller) HandleTransaction(tx []byte) lib.ErrorI {
	// generate hash bytes for the transaction
	hash := crypto.Hash(tx)
	// generate a hex encoded hash string for the transaction
	hashString := lib.BytesToString(hash)
	// get the transaction from the indexer using the hash bytes
	txResult, err := c.FSM.Store().(lib.StoreI).GetTxByHash(hash)
	// if an error occurred attempting to retrieve the tx
	if err != nil {
		// exit with the error
		return err
	}
	// if the transaction already exists
	if txResult.TxHash != "" {
		// exit with duplicate transaction error
		return lib.ErrDuplicateTx(hashString)
	}
	// ensure the mempool doesn't already contain the transaction
	if c.Mempool.Contains(hashString) {
		// if it does, exit with already found in the mempool
		return lib.ErrTxFoundInMempool(hashString)
	}
	// route the transaction to the mempool for handling
	return c.Mempool.HandleTransaction(tx)
}

// Mempool accepts or rejects incoming txs based on the mempool (ephemeral copy) state
// - recheck when
//   - mempool dropped some percent of the lowest fee txs
//   - new tx has higher fee than the lowest
//
// - notes:
//   - new tx added may also be evicted, this is expected behavior
type Mempool struct {
	lib.Mempool                        // the memory pool itself defined as an interface
	FSM             *fsm.StateMachine  // the ephemeral finite state machine used to validate inbound transactions
	cachedResults   lib.TxResults      // a memory cache of transaction results for efficient verification
	cachedFailedTxs *lib.FailedTxCache // a memory cache of failed transactions for tracking
	log             lib.LoggerI        // the logger
}

// NewMempool() creates a new instance of a Mempool structure
func NewMempool(fsm *fsm.StateMachine, config lib.MempoolConfig, log lib.LoggerI) (m *Mempool, err lib.ErrorI) {
	// initialize the structure
	m = &Mempool{
		Mempool:         lib.NewMempool(config),
		cachedFailedTxs: lib.NewFailedTxCache(),
		log:             log,
	}
	// make an 'mempool (ephemeral copy) state' so the mempool can maintain only 'valid' transactions despite dependencies and conflicts
	m.FSM, err = fsm.Copy()
	// if an error occurred copying the fsm
	if err != nil {
		return nil, err
	}
	// initialize the mempool at the 'begin block' phase because it's the proper lifecycle phase for transaction handling
	m.FSM.ResetToBeginBlock()
	// exit
	return m, err
}

// HandleTransaction() attempts to add a transaction to the mempool by validating, adding, and evicting overfull or newly invalid txs
func (m *Mempool) HandleTransaction(tx []byte) (err lib.ErrorI) {
	// upon completing this function
	defer func() {
		// in an error occurred while handling this transaction
		if err != nil {
			// cache failed txs for RPC display
			m.cachedFailedTxs.Add(tx, crypto.HashString(tx), err)
		}
	}()
	// validate the transaction against the mempool (ephemeral copy) state
	result, err := m.applyAndWriteTx(tx)
	// if an error occurred while applying this transaction against the ephemeral copy
	if err != nil {
		// exit with the error
		return
	}
	// extract the fee from the transaction result
	fee := result.Transaction.Fee
	// if the transaction is a special type: 'certificate result'
	if result.MessageType == fsm.MessageCertificateResultsName {
		// prioritize certificate result transactions by artificially raising the fee 'stored fee'
		fee = math.MaxUint32
	}
	// add a transaction to the mempool
	recheck, err := m.AddTransaction(tx, fee)
	// if an error occurred adding the transaction to the memory pool
	if err != nil {
		// exit with the error
		return
	}
	// cache the results for RPC display
	m.log.Infof("Added tx %s to mempool for checking", crypto.HashString(tx))
	// add the result to the cache
	m.cachedResults = append(m.cachedResults, result)
	// recheck the mempool if necessary
	if recheck {
		// recheck the mempool
		m.checkMempool()
	}
	// exit
	return
}

// checkMempool() validates all transactions the mempool using the mempool (ephemeral copy) state and evicts any that are invalid
func (m *Mempool) checkMempool() {
	// reset the mempool (ephemeral copy) state to just after the automatic 'begin block' phase
	m.FSM.ResetToBeginBlock()
	// reset the RPC cached results
	m.cachedResults = nil
	// create a list of the transactions to delete
	var toDelete [][]byte
	// create an iterator for the mempool
	it := m.Iterator()
	// at the end of the function, close the iterator
	defer it.Close()
	// for each mempool transaction
	for ; it.Valid(); it.Next() {
		// get the transaction itself from the iterator
		tx := it.Key()
		// write the transaction to the state machine
		result, err := m.applyAndWriteTx(tx)
		// if an error occurred during the application
		if err != nil {
			// log the error
			m.log.Error(err.Error())
			// add to the remove list
			toDelete = append(toDelete, tx)
			// and cache it
			m.cachedFailedTxs.Add(tx, crypto.HashString(tx), err)
			// continue with the next iteration
			continue
		}
		// cache the results
		m.cachedResults = append(m.cachedResults, result)
	}
	// evict all 'newly' invalid transactions from the mempool
	for _, tx := range toDelete {
		// log the eviction
		m.log.Infof("Removed tx %s from mempool", crypto.HashString(tx))
		// delete the transaction
		m.DeleteTransaction(tx)
	}
}

// applyAndWriteTx() checks the validity of a transaction by playing it against the mempool (ephemeral copy) state machine
func (m *Mempool) applyAndWriteTx(tx []byte) (result *lib.TxResult, err lib.ErrorI) {
	// get the ephemeral store from the mempool state machine
	store := m.FSM.Store()
	// wrap the store in a 'database transaction' in case a rollback to the previous valid transaction is needed
	txn, err := m.FSM.TxnWrap()
	// if an error occurred during the wrapping
	if err != nil {
		// exit with error
		return
	}
	// at the end of this code, set the state machine store back to the 'ephemeral store' and discard the 'database transaction'
	defer func() { m.FSM.SetStore(store); txn.Discard() }()
	// apply the transaction to the mempool (ephemeral copy) state machine
	result, err = m.FSM.ApplyTransaction(uint64(m.TxCount()), tx, crypto.HashString(tx))
	// if an error occurred when applying the transaction
	if err != nil {
		// exit with error
		return
	}
	// write the transaction to the mempool store
	if err = txn.Write(); err != nil {
		// exit with error
		return
	}
	// exit with the result
	return
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
