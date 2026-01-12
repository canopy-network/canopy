package rpc

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/store"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

/* This file implements the client & server logic for the 'root-chain info' and corresponding 'on-demand' calls to the rpc */

var _ lib.RCManagerI = new(RCManager)

const chainIdParamName = "chainId"

const (
	defaultRCSubscriberReadLimitBytes = int64(64 * 1024)
	defaultRCSubscriberWriteTimeout   = 10 * time.Second
	defaultRCSubscriberPongWait       = 60 * time.Second
	defaultRCSubscriberPingPeriod     = 50 * time.Second
	defaultMaxRCSubscribers           = 512
	defaultMaxRCSubscribersPerChain   = 128
)

// RCManager handles a group of root-chain sock clients
type RCManager struct {
	c             lib.Config                    // the global node config
	controller    *controller.Controller        // reference to controller for state access
	subscriptions map[uint64]*RCSubscription    // chainId -> subscription
	subscribers   map[uint64][]*RCSubscriber    // chainId -> subscribers
	l             *sync.Mutex                   // thread safety
	afterRCUpdate func(info *lib.RootChainInfo) // callback after the root chain info update
	upgrader      websocket.Upgrader            // upgrade http connection to ws
	log           lib.LoggerI                   // stdout log
	// rc subscriber limits
	rcSubscriberReadLimitBytes int64
	rcSubscriberWriteTimeout   time.Duration
	rcSubscriberPongWait       time.Duration
	rcSubscriberPingPeriod     time.Duration
	maxRCSubscribers           int
	maxRCSubscribersPerChain   int
	subscriberCount            int
	// block data subscribers
	blockDataSubscribers      []*BlockDataSubscriber
	blockDataSubscriberCount  int
	maxBlockDataSubscribers   int
}

// NewRCManager() constructs a new instance of a RCManager
func NewRCManager(controller *controller.Controller, config lib.Config, logger lib.LoggerI) (manager *RCManager) {
	readLimit := config.RCSubscriberReadLimitBytes
	if readLimit <= 0 {
		readLimit = defaultRCSubscriberReadLimitBytes
	}
	writeTimeout := time.Duration(config.RCSubscriberWriteTimeoutMS) * time.Millisecond
	if writeTimeout <= 0 {
		writeTimeout = defaultRCSubscriberWriteTimeout
	}
	pongWait := time.Duration(config.RCSubscriberPongWaitS) * time.Second
	if pongWait <= 0 {
		pongWait = defaultRCSubscriberPongWait
	}
	pingPeriod := time.Duration(config.RCSubscriberPingPeriodS) * time.Second
	if pingPeriod <= 0 || pingPeriod >= pongWait {
		pingPeriod = pongWait * 9 / 10
	}
	maxSubscribers := config.MaxRCSubscribers
	if maxSubscribers <= 0 {
		maxSubscribers = defaultMaxRCSubscribers
	}
	maxSubscribersPerChain := config.MaxRCSubscribersPerChain
	if maxSubscribersPerChain <= 0 {
		maxSubscribersPerChain = defaultMaxRCSubscribersPerChain
	}
	// create the manager
	manager = &RCManager{
		c:                          config,
		controller:                 controller,
		subscriptions:              make(map[uint64]*RCSubscription),
		subscribers:                make(map[uint64][]*RCSubscriber),
		l:                          controller.Mutex,
		afterRCUpdate:              controller.UpdateRootChainInfo,
		upgrader:                   websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		log:                        logger,
		rcSubscriberReadLimitBytes: readLimit,
		rcSubscriberWriteTimeout:   writeTimeout,
		rcSubscriberPongWait:       pongWait,
		rcSubscriberPingPeriod:     pingPeriod,
		maxRCSubscribers:           maxSubscribers,
		maxRCSubscribersPerChain:   maxSubscribersPerChain,
		blockDataSubscribers:       make([]*BlockDataSubscriber, 0),
		maxBlockDataSubscribers:    maxSubscribers, // reuse same limit
	}
	// set the manager in the controller
	controller.RCManager = manager
	// exit
	return
}

// Start() attempts to establish a websocket connection with each root chain
func (r *RCManager) Start() {
	// for each rc in the config
	for _, rc := range r.c.RootChain {
		// dial each root chain
		r.NewSubscription(rc)
	}
}

// Publish() writes the root-chain info to each client
func (r *RCManager) Publish(chainId uint64, info *lib.RootChainInfo) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// convert the root-chain info to bytes
	protoBytes, err := lib.Marshal(info)
	if err != nil {
		return
	}
	// copy subscribers under lock to avoid map iteration races
	r.l.Lock()
	subscribers := append([]*RCSubscriber(nil), r.subscribers[chainId]...)
	r.l.Unlock()
	// for each ws client
	for _, subscriber := range subscribers {
		// publish to each client
		if e := subscriber.writeMessage(websocket.BinaryMessage, protoBytes); e != nil {
			subscriber.Stop(e)
			continue
		}
	}
}

// buildIndexerSnapshot creates a lib.IndexerSnapshot protobuf for the given height
// This is used to send state snapshots over WebSocket alongside root chain info
func (r *RCManager) buildIndexerSnapshot(height uint64) (*lib.IndexerSnapshot, error) {
	// Setup store for indexed data (blocks, txs, events)
	db := r.controller.FSM.Store().(lib.StoreI).DB()
	st, err := store.NewStoreWithDB(r.c, db, nil, r.log)
	if err != nil {
		return nil, err
	}
	defer st.Discard()

	if height == 0 {
		height = st.Version() - 1
	}
	prevHeight := height - 1

	// Get state machines for current and previous height
	smCurrent, err := r.controller.FSM.TimeMachine(height)
	if err != nil {
		return nil, err
	}
	defer smCurrent.Discard()

	smPrevious, err := r.controller.FSM.TimeMachine(prevHeight)
	if err != nil {
		return nil, err
	}
	defer smPrevious.Discard()

	snapshot := &lib.IndexerSnapshot{Height: height}

	// Fetch block data from indexer (errors result in nil, not failure)
	var blockErr error
	snapshot.Block, blockErr = st.GetBlockByHeight(height)
	if blockErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetBlockByHeight failed for height %d: %s", height, blockErr.Error())
	}
	if txPage, txErr := st.GetTxsByHeight(height, true, lib.PageParams{}); txErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetTxsByHeight failed for height %d: %s", height, txErr.Error())
	} else if txPage != nil {
		if txs, ok := txPage.Results.(*lib.TxResults); ok {
			snapshot.Transactions = []*lib.TxResult(*txs)
		}
	}
	if evtPage, evtErr := st.GetEventsByBlockHeight(height, true, lib.PageParams{}); evtErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetEventsByBlockHeight failed for height %d: %s", height, evtErr.Error())
	} else if evtPage != nil {
		if evts, ok := evtPage.Results.(*lib.Events); ok {
			snapshot.Events = []*lib.Event(*evts)
		}
	}

	// Fetch state data (current height) - serialize to bytes for proto
	if accPage, accErr := smCurrent.GetAccountsPaginated(lib.PageParams{}); accErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetAccountsPaginated failed for height %d: %s", height, accErr.Error())
	} else if accPage != nil {
		if accs, ok := accPage.Results.(*fsm.AccountPage); ok {
			snapshot.Accounts = make([][]byte, len(*accs))
			for i, acc := range *accs {
				if data, marshalErr := lib.Marshal(acc); marshalErr != nil {
					r.log.Warnf("buildIndexerSnapshot: Marshal account[%d] failed for height %d: %s", i, height, marshalErr.Error())
				} else {
					snapshot.Accounts[i] = data
				}
			}
		}
	}
	if orders, ordersErr := smCurrent.GetOrderBooks(); ordersErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetOrderBooks failed for height %d: %s", height, ordersErr.Error())
	} else {
		snapshot.Orders = orders
	}
	if prices, pricesErr := smCurrent.GetDexPrices(); pricesErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDexPrices failed for height %d: %s", height, pricesErr.Error())
	} else if prices != nil {
		snapshot.DexPrices = make([][]byte, len(prices))
		for i, p := range prices {
			if data, marshalErr := lib.Marshal(p); marshalErr != nil {
				r.log.Warnf("buildIndexerSnapshot: Marshal dex price[%d] failed for height %d: %s", i, height, marshalErr.Error())
			} else {
				snapshot.DexPrices[i] = data
			}
		}
	}
	if params, paramsErr := smCurrent.GetParams(); paramsErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetParams failed for height %d: %s", height, paramsErr.Error())
	} else if params != nil {
		if data, marshalErr := lib.Marshal(params); marshalErr != nil {
			r.log.Warnf("buildIndexerSnapshot: Marshal params failed for height %d: %s", height, marshalErr.Error())
		} else {
			snapshot.Params = data
		}
	}
	if supply, supplyErr := smCurrent.GetSupply(); supplyErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetSupply failed for height %d: %s", height, supplyErr.Error())
	} else if supply != nil {
		if data, marshalErr := lib.Marshal(supply); marshalErr != nil {
			r.log.Warnf("buildIndexerSnapshot: Marshal supply failed for height %d: %s", height, marshalErr.Error())
		} else {
			snapshot.Supply = data
		}
	}
	if committeesData, cdErr := smCurrent.GetCommitteesData(); cdErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetCommitteesData failed for height %d: %s", height, cdErr.Error())
	} else {
		snapshot.CommitteesData = committeesData
	}
	if subsidized, subErr := smCurrent.GetSubsidizedCommittees(); subErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetSubsidizedCommittees failed for height %d: %s", height, subErr.Error())
	} else {
		snapshot.SubsidizedCommittees = subsidized
	}
	if retired, retErr := smCurrent.GetRetiredCommittees(); retErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetRetiredCommittees failed for height %d: %s", height, retErr.Error())
	} else {
		snapshot.RetiredCommittees = retired
	}

	// Change detection pairs (current + H-1) - serialize validators/pools to bytes
	if valPage, valErr := smCurrent.GetValidatorsPaginated(lib.PageParams{}, lib.ValidatorFilters{}); valErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetValidatorsPaginated (current) failed for height %d: %s", height, valErr.Error())
	} else if valPage != nil {
		if vals, ok := valPage.Results.(*fsm.ValidatorPage); ok {
			snapshot.ValidatorsCurrent = make([][]byte, len(*vals))
			for i, v := range *vals {
				if data, marshalErr := lib.Marshal(v); marshalErr != nil {
					r.log.Warnf("buildIndexerSnapshot: Marshal validator current[%d] failed for height %d: %s", i, height, marshalErr.Error())
				} else {
					snapshot.ValidatorsCurrent[i] = data
				}
			}
		}
	}
	if valPage, valErr := smPrevious.GetValidatorsPaginated(lib.PageParams{}, lib.ValidatorFilters{}); valErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetValidatorsPaginated (previous) failed for height %d: %s", prevHeight, valErr.Error())
	} else if valPage != nil {
		if vals, ok := valPage.Results.(*fsm.ValidatorPage); ok {
			snapshot.ValidatorsPrevious = make([][]byte, len(*vals))
			for i, v := range *vals {
				if data, marshalErr := lib.Marshal(v); marshalErr != nil {
					r.log.Warnf("buildIndexerSnapshot: Marshal validator previous[%d] failed for height %d: %s", i, prevHeight, marshalErr.Error())
				} else {
					snapshot.ValidatorsPrevious[i] = data
				}
			}
		}
	}

	if poolPage, poolErr := smCurrent.GetPoolsPaginated(lib.PageParams{}); poolErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetPoolsPaginated (current) failed for height %d: %s", height, poolErr.Error())
	} else if poolPage != nil {
		if pools, ok := poolPage.Results.(*fsm.PoolPage); ok {
			snapshot.PoolsCurrent = make([][]byte, len(*pools))
			for i, p := range *pools {
				if data, marshalErr := lib.Marshal(p); marshalErr != nil {
					r.log.Warnf("buildIndexerSnapshot: Marshal pool current[%d] failed for height %d: %s", i, height, marshalErr.Error())
				} else {
					snapshot.PoolsCurrent[i] = data
				}
			}
		}
	}
	if poolPage, poolErr := smPrevious.GetPoolsPaginated(lib.PageParams{}); poolErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetPoolsPaginated (previous) failed for height %d: %s", prevHeight, poolErr.Error())
	} else if poolPage != nil {
		if pools, ok := poolPage.Results.(*fsm.PoolPage); ok {
			snapshot.PoolsPrevious = make([][]byte, len(*pools))
			for i, p := range *pools {
				if data, marshalErr := lib.Marshal(p); marshalErr != nil {
					r.log.Warnf("buildIndexerSnapshot: Marshal pool previous[%d] failed for height %d: %s", i, prevHeight, marshalErr.Error())
				} else {
					snapshot.PoolsPrevious[i] = data
				}
			}
		}
	}

	if ns, nsErr := smCurrent.GetNonSigners(); nsErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetNonSigners (current) failed for height %d: %s", height, nsErr.Error())
	} else if ns != nil {
		if data, marshalErr := lib.Marshal(ns); marshalErr != nil {
			r.log.Warnf("buildIndexerSnapshot: Marshal non-signers current failed for height %d: %s", height, marshalErr.Error())
		} else {
			snapshot.NonSignersCurrent = data
		}
	}
	if ns, nsErr := smPrevious.GetNonSigners(); nsErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetNonSigners (previous) failed for height %d: %s", prevHeight, nsErr.Error())
	} else if ns != nil {
		if data, marshalErr := lib.Marshal(ns); marshalErr != nil {
			r.log.Warnf("buildIndexerSnapshot: Marshal non-signers previous failed for height %d: %s", prevHeight, marshalErr.Error())
		} else {
			snapshot.NonSignersPrevious = data
		}
	}

	if doubleSigners, dsErr := st.GetDoubleSigners(); dsErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDoubleSigners failed for height %d: %s", height, dsErr.Error())
	} else {
		snapshot.DoubleSignersCurrent = doubleSigners
	}

	if dexBatches, dbErr := smCurrent.GetDexBatches(true); dbErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDexBatches (current, confirmed) failed for height %d: %s", height, dbErr.Error())
	} else {
		snapshot.DexBatchesCurrent = dexBatches
	}
	if dexBatches, dbErr := smPrevious.GetDexBatches(true); dbErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDexBatches (previous, confirmed) failed for height %d: %s", prevHeight, dbErr.Error())
	} else {
		snapshot.DexBatchesPrevious = dexBatches
	}

	if nextDexBatches, ndbErr := smCurrent.GetDexBatches(false); ndbErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDexBatches (current, next) failed for height %d: %s", height, ndbErr.Error())
	} else {
		snapshot.NextDexBatchesCurrent = nextDexBatches
	}
	if nextDexBatches, ndbErr := smPrevious.GetDexBatches(false); ndbErr != nil {
		r.log.Warnf("buildIndexerSnapshot: GetDexBatches (previous, next) failed for height %d: %s", prevHeight, ndbErr.Error())
	} else {
		snapshot.NextDexBatchesPrevious = nextDexBatches
	}

	return snapshot, nil
}

// ChainIds() returns a list of chainIds for subscribers
func (r *RCManager) ChainIds() (list []uint64) {
	// de-duplicate the results
	deDupe := lib.NewDeDuplicator[uint64]()
	// for each client
	for chainId, chainSubscribers := range r.subscribers {
		// if the client chain id isn't empty and not duplicate
		for _, subscriber := range chainSubscribers {
			if subscriber.chainId != chainId {
				// remove subscriber with incorrect chain id
				subscriber.Stop(lib.ErrWrongChainId())
				continue
			}
			if subscriber.chainId != 0 && !deDupe.Found(subscriber.chainId) {
				list = append(list, subscriber.chainId)
			}
		}
	}
	return
}

// GetHeight() returns the height from the root-chain
func (r *RCManager) GetHeight(rootChainId uint64) uint64 {
	// check the map to see if the info exists
	if sub, found := r.subscriptions[rootChainId]; found {
		// exit with the height of the root-chain-info
		return sub.Info.Height
	}
	return 0
}

// GetRootChainInfo() retrieves the root chain info from the root chain 'on-demand'
func (r *RCManager) GetRootChainInfo(rootChainId, chainId uint64) (info *lib.RootChainInfo, err lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// lock for thread safety
	r.l.Lock()
	defer r.l.Unlock()
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// get the info
	info, err = sub.RootChainInfo(0, chainId)
	if err != nil {
		return nil, err
	}
	// update the info
	sub.Info = info
	// exit with the info
	return
}

// GetValidatorSet() returns the validator set from the root-chain
func (r *RCManager) GetValidatorSet(rootChainId, id, rootHeight uint64) (lib.ValidatorSet, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return lib.ValidatorSet{}, lib.ErrNotSubscribed()
	}
	// if rootHeight is the same as the RootChainInfo height
	if rootHeight == sub.Info.Height || rootHeight == 0 {
		// exit with a copy the validator set
		return lib.NewValidatorSet(sub.Info.ValidatorSet)
	}
	// if rootHeight is 1 before the RootChainInfo height
	if rootHeight == sub.Info.Height-1 {
		// exit with a copy of the previous validator set
		return lib.NewValidatorSet(sub.Info.LastValidatorSet)
	}
	// warn of the remote RPC call to the root chain API
	r.log.Warnf("Executing remote GetValidatorSet call with requested height=%d for rootChainId=%d with latest root height at %d", rootHeight, rootChainId, sub.Info.Height)
	// execute the remote RPC call to the root chain API
	return sub.ValidatorSet(rootHeight, id)
}

// GetOrders() returns the order book from the root-chain
func (r *RCManager) GetOrders(rootChainId, rootHeight, id uint64) (*lib.OrderBook, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// if the root chain id and height is the same as the info
	if sub.Info.Height == rootHeight {
		// exit with the order books from memory
		return sub.Info.Orders, nil
	}
	// warn of the remote RPC call to the root chain API
	r.log.Warnf("Executing remote GetOrders call with requested height=%d for rootChainId=%d with latest root height at %d", rootHeight, rootChainId, sub.Info.Height)
	// execute the remote call
	books, err := sub.Orders(rootHeight, id)
	// if an error occurred during the remote call
	if err != nil {
		// exit with error
		return nil, err
	}
	// ensure the order book isn't empty
	if books == nil || len(books.OrderBooks) == 0 {
		// exit with error
		return nil, lib.ErrEmptyOrderBook()
	}
	// exit with the first (and only) order book in the list
	return books.OrderBooks[0], nil
}

// Order() returns a specific order from the root order book
func (r *RCManager) GetOrder(rootChainId, height uint64, orderId string, chainId uint64) (*lib.SellOrder, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	return sub.Order(height, orderId, chainId)
}

// IsValidDoubleSigner() returns if an address is a valid double signer for a specific 'double sign height'
func (r *RCManager) IsValidDoubleSigner(rootChainId, height uint64, address string) (*bool, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// exit with the results of the remote RPC call to the API of the 'root chain'
	return sub.IsValidDoubleSigner(height, address)
}

// GetMinimumEvidenceHeight() returns the minimum height double sign evidence must have to be 'valid'
func (r *RCManager) GetMinimumEvidenceHeight(rootChainId, height uint64) (*uint64, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// exit with the results of the remote RPC call to the API of the 'root chain'
	return sub.MinimumEvidenceHeight(height)
}

// GetCheckpoint() returns the checkpoint if any for a specific chain height
// TODO should be able to get these from the file or the root-chain upon independence
func (r *RCManager) GetCheckpoint(rootChainId, height, chainId uint64) (blockHash lib.HexBytes, err lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// exit with the results of the remote RPC call to the API of the 'root chain'
	return sub.Checkpoint(height, chainId)
}

// GetLotteryWinner() returns the winner of the delegate lottery from the root-chain
func (r *RCManager) GetLotteryWinner(rootChainId, height, id uint64) (*lib.LotteryWinner, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	// if the root chain id and height is the same as the info
	if sub.Info.Height == height {
		// exit with the lottery winner
		return sub.Info.LotteryWinner, nil
	}
	// exit with the results of the remote RPC call to the API of the 'root chain'
	return sub.Lottery(height, id)
}

// Transaction() executes a transaction on the root chain
func (r *RCManager) Transaction(rootChainId uint64, tx lib.TransactionI) (hash *string, err lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	return sub.Transaction(tx)
}

// GetDexBatch() queries a 'dex batch on the root chain
func (r *RCManager) GetDexBatch(rootChainId, height, committee uint64, withPoints bool) (*lib.DexBatch, lib.ErrorI) {
	defer lib.TimeTrack(r.log, time.Now(), 500*time.Millisecond)
	// if the root chain id is the same as the info
	sub, found := r.subscriptions[rootChainId]
	if !found {
		// exit with 'not subscribed' error
		return nil, lib.ErrNotSubscribed()
	}
	return sub.DexBatch(height, committee, withPoints)
}

// SUBSCRIPTION CODE BELOW (OUTBOUND)

// RCSubscription (TransactionRoot Chain Subscription) implements an efficient subscription to root chain info
type RCSubscription struct {
	chainId uint64             // the chain id of the subscription
	Info    *lib.RootChainInfo // root-chain info cached from the publisher
	manager *RCManager         // a reference to the manager of the ws clients
	conn    *websocket.Conn    // the underlying ws connection
	*Client                    // use http for 'on-demand' requests
	log     lib.LoggerI        // stdout log
}

// Dial() dials a root chain via ws
func (r *RCManager) NewSubscription(rc lib.RootChain) {
	// create a new web socket client
	client := &RCSubscription{
		chainId: rc.ChainId,
		Info:    new(lib.RootChainInfo),
		manager: r,
		Client:  NewClient(rc.Url, rc.Url),
		log:     r.log,
	}
	// start to connect with backoff
	go client.dialWithBackoff(r.c.ChainId, rc)
}

// dialWithBackoff() establishes a websocket connection given a root chain configuration
func (r *RCSubscription) dialWithBackoff(chainId uint64, config lib.RootChain) {
	// parse the config
	parsedUrl, err := url.Parse(config.Url)
	if err != nil {
		r.log.Fatal(err.Error())
	}
	// get the host
	host := parsedUrl.Host
	// if the host is empty
	if host == "" {
		// fallback if url didn't have a scheme and was treated as a path
		host = parsedUrl.Path
	}
	// create a URL to connect to the root chain with
	u := url.URL{Scheme: "ws", Host: host, Path: SubscribeRCInfoPath, RawQuery: fmt.Sprintf("%s=%d", chainIdParamName, chainId)}
	// create a new retry for backoff
	retry := lib.NewRetry(uint64(time.Second.Milliseconds()), 25)
	// until backoff fails or connection succeeds
	for retry.WaitAndDoRetry() {
		// log the connection
		r.log.Infof("Connecting to rootChainId=%d @ %s", config.ChainId, u.String())
		// dial the url
		conn, _, e := websocket.DefaultDialer.Dial(u.String(), nil)
		if e == nil {
			// set the connection
			r.conn = conn
			// call get root chain info
			info, er := r.RootChainInfo(0, chainId)
			if er != nil || info == nil || info.Height == 0 {
				if er != nil {
					r.log.Error(er.Error())
				} else if info == nil || info.Height == 0 {
					r.log.Error("invalid root chain info")
				}
				continue
			}
			// set the information
			r.Info = info
			// start the listener
			go r.Listen()
			// add to the manager
			r.manager.AddSubscription(r)
			// exit
			return
		}
		r.log.Error(e.Error())
	}
}

// Listen() begins listening on the websockets client
func (r *RCSubscription) Listen() {
	for {
		// get the message from the buffer
		_, bz, err := r.conn.ReadMessage()
		if err != nil {
			r.Stop(err)
			return
		}
		// read the message into a rootChainInfo struct
		newInfo := new(lib.RootChainInfo)
		// unmarshal proto bytes into the message
		if err = lib.Unmarshal(bz, newInfo); err != nil {
			r.Stop(err)
			return
		}
		// log the receipt of the root-chain info
		r.log.Infof("Received info from RootChainId=%d and Height=%d", newInfo.RootChainId, newInfo.Height)
		// thread safety
		r.manager.l.Lock()
		// update the root chain info
		r.Info = newInfo
		// execute the callback
		r.manager.afterRCUpdate(newInfo)
		// release
		r.manager.l.Unlock()
	}
}

// Add() adds the client to the manager
func (r *RCManager) AddSubscription(subscription *RCSubscription) {
	// lock for thread safety
	r.l.Lock()
	defer r.l.Unlock()
	// add to the map
	r.subscriptions[subscription.chainId] = subscription
}

// RemoveSubscription() gracefully deletes a RC subscription
func (r *RCManager) RemoveSubscription(chainId uint64) {
	// lock for thread safety
	r.l.Lock()
	defer r.l.Unlock()
	// remove from the map
	delete(r.subscriptions, chainId)
	// check if the chainId == a configured root chain
	for _, rc := range r.c.RootChain {
		// if found
		if rc.ChainId == chainId {
			// re-dial
			r.NewSubscription(rc)
			// exit
			return
		}
	}
}

// Stop() stops the client
func (r *RCSubscription) Stop(err error) {
	// log the error
	r.log.Errorf("WS Failed with err: %s", err.Error())
	// close the connection
	if err = r.conn.Close(); err != nil {
		r.log.Error(err.Error())
	}
	// remove from the manager
	r.manager.RemoveSubscription(r.chainId)
}

// SUBSCRIBER CODE BELOW (INBOUND)

// RCSubscriber (TransactionRoot Chain Subscriber) implements an efficient publishing service to nested chain subscribers
type RCSubscriber struct {
	chainId uint64          // the chain id of the publisher
	manager *RCManager      // a reference to the manager of the ws clients
	conn    *websocket.Conn // the underlying ws connection
	log     lib.LoggerI     // stdout log
	writeMu sync.Mutex      // protects concurrent writes
}

// WebSocket() upgrades a http request to a websockets connection
func (s *Server) WebSocket(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = w.(http.Hijacker)
	// upgrade the connection to websockets
	conn, err := s.rcManager.upgrader.Upgrade(w, r, nil)
	// if an error occurred during the upgrade
	if err != nil {
		// write the internal server error
		write(w, err, http.StatusInternalServerError)
		// log the issue
		s.logger.Error(err.Error())
		// exit
		return
	}
	// get chain id string from the parameter
	chainIdStr := r.URL.Query().Get(chainIdParamName)
	// get the chain id from the string
	chainId, err := strconv.ParseUint(chainIdStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid chain id", http.StatusBadRequest)
		return
	}
	if chainId == 0 {
		http.Error(w, "invalid chain id", http.StatusBadRequest)
		return
	}
	// create a new web sockets client
	client := &RCSubscriber{
		chainId: chainId,
		conn:    conn,
		manager: s.rcManager,
		log:     s.logger,
	}
	// add the connection to the manager
	if err := s.rcManager.AddSubscriber(client); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		if closeErr := conn.Close(); closeErr != nil {
			s.logger.Error(closeErr.Error())
		}
		return
	}
	client.Start()
}

// Add() adds the client to the manager
func (r *RCManager) AddSubscriber(subscriber *RCSubscriber) error {
	// lock for thread safety
	r.l.Lock()
	defer r.l.Unlock()
	if r.maxRCSubscribers > 0 && r.subscriberCount >= r.maxRCSubscribers {
		return fmt.Errorf("subscriber limit reached")
	}
	if r.maxRCSubscribersPerChain > 0 && len(r.subscribers[subscriber.chainId]) >= r.maxRCSubscribersPerChain {
		return fmt.Errorf("subscriber limit reached for chainId=%d", subscriber.chainId)
	}
	// add to the map
	r.subscribers[subscriber.chainId] = append(r.subscribers[subscriber.chainId], subscriber)
	r.subscriberCount++
	return nil
}

// RemoveSubscriber() gracefully deletes a RC subscriber
func (r *RCManager) RemoveSubscriber(chainId uint64, subscriber *RCSubscriber) {
	// lock for thread safety
	r.l.Lock()
	defer r.l.Unlock()
	// remove from the slice
	before := len(r.subscribers[chainId])
	r.subscribers[chainId] = slices.DeleteFunc(r.subscribers[chainId], func(sub *RCSubscriber) bool {
		return sub == subscriber
	})
	if len(r.subscribers[chainId]) == 0 {
		delete(r.subscribers, chainId)
	}
	if len(r.subscribers[chainId]) < before {
		r.subscriberCount--
	}
}

// Start() configures and starts subscriber lifecycle goroutines
func (r *RCSubscriber) Start() {
	r.conn.SetReadLimit(r.manager.rcSubscriberReadLimitBytes)
	_ = r.conn.SetReadDeadline(time.Now().Add(r.manager.rcSubscriberPongWait))
	r.conn.SetPongHandler(func(string) error {
		_ = r.conn.SetReadDeadline(time.Now().Add(r.manager.rcSubscriberPongWait))
		return nil
	})
	go r.readLoop()
	go r.pingLoop()
}

func (r *RCSubscriber) readLoop() {
	for {
		if _, _, err := r.conn.ReadMessage(); err != nil {
			r.Stop(err)
			return
		}
	}
}

func (r *RCSubscriber) pingLoop() {
	ticker := time.NewTicker(r.manager.rcSubscriberPingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		if err := r.writeMessage(websocket.PingMessage, nil); err != nil {
			r.Stop(err)
			return
		}
	}
}

func (r *RCSubscriber) writeMessage(messageType int, data []byte) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_ = r.conn.SetWriteDeadline(time.Now().Add(r.manager.rcSubscriberWriteTimeout))
	return r.conn.WriteMessage(messageType, data)
}

// Stop() stops the client
func (r *RCSubscriber) Stop(err error) {
	// log the error
	r.log.Errorf("WS Failed with err: %s", err.Error())
	// close the connection
	if err = r.conn.Close(); err != nil {
		r.log.Error(err.Error())
	}
	// remove from the manager
	r.manager.RemoveSubscriber(r.chainId, r)
}

// BlockDataSubscriber represents a WebSocket client subscribed to block data updates
type BlockDataSubscriber struct {
	conn    *websocket.Conn // the underlying ws connection
	manager *RCManager      // reference to manager
	log     lib.LoggerI     // stdout log
	writeMu sync.Mutex      // protects concurrent writes
}

// BlockDataWebSocket() upgrades a http request to a websockets connection for block data
func (s *Server) BlockDataWebSocket(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = w.(http.Hijacker)
	// upgrade the connection to websockets
	conn, err := s.rcManager.upgrader.Upgrade(w, r, nil)
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		s.logger.Error(err.Error())
		return
	}
	// create a new block data subscriber
	subscriber := &BlockDataSubscriber{
		conn:    conn,
		manager: s.rcManager,
		log:     s.logger,
	}
	// add the subscriber to the manager
	if err := s.rcManager.AddBlockDataSubscriber(subscriber); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		if closeErr := conn.Close(); closeErr != nil {
			s.logger.Error(closeErr.Error())
		}
		return
	}
	// send initial snapshot at current height
	go subscriber.sendInitialSnapshot()
	// start the subscriber lifecycle
	subscriber.Start()
}

// sendInitialSnapshot sends the current block data snapshot when a client first connects
func (b *BlockDataSubscriber) sendInitialSnapshot() {
	snapshot, err := b.manager.buildIndexerSnapshot(0) // 0 means latest height
	if err != nil {
		b.log.Warnf("BlockDataSubscriber: failed to build initial snapshot: %s", err.Error())
		return
	}
	protoBytes, err := lib.Marshal(snapshot)
	if err != nil {
		b.log.Warnf("BlockDataSubscriber: failed to marshal initial snapshot: %s", err.Error())
		return
	}
	if err := b.writeMessage(websocket.BinaryMessage, protoBytes); err != nil {
		b.Stop(err)
	}
}

// AddBlockDataSubscriber adds a block data subscriber to the manager
func (r *RCManager) AddBlockDataSubscriber(subscriber *BlockDataSubscriber) error {
	r.l.Lock()
	defer r.l.Unlock()
	if r.maxBlockDataSubscribers > 0 && r.blockDataSubscriberCount >= r.maxBlockDataSubscribers {
		return fmt.Errorf("block data subscriber limit reached")
	}
	r.blockDataSubscribers = append(r.blockDataSubscribers, subscriber)
	r.blockDataSubscriberCount++
	return nil
}

// RemoveBlockDataSubscriber removes a block data subscriber from the manager
func (r *RCManager) RemoveBlockDataSubscriber(subscriber *BlockDataSubscriber) {
	r.l.Lock()
	defer r.l.Unlock()
	before := len(r.blockDataSubscribers)
	r.blockDataSubscribers = slices.DeleteFunc(r.blockDataSubscribers, func(sub *BlockDataSubscriber) bool {
		return sub == subscriber
	})
	if len(r.blockDataSubscribers) < before {
		r.blockDataSubscriberCount--
	}
}

// PublishBlockData sends the IndexerSnapshot to all block data subscribers
func (r *RCManager) PublishBlockData(height uint64) {
	// build the snapshot
	snapshot, err := r.buildIndexerSnapshot(height)
	if err != nil {
		r.log.Warnf("PublishBlockData: failed to build snapshot for height %d: %s", height, err.Error())
		return
	}
	// marshal to proto bytes
	protoBytes, err := lib.Marshal(snapshot)
	if err != nil {
		r.log.Warnf("PublishBlockData: failed to marshal snapshot for height %d: %s", height, err.Error())
		return
	}
	// copy subscribers under lock to avoid map iteration races
	r.l.Lock()
	subscribers := append([]*BlockDataSubscriber(nil), r.blockDataSubscribers...)
	r.l.Unlock()
	// publish to each subscriber
	for _, subscriber := range subscribers {
		if e := subscriber.writeMessage(websocket.BinaryMessage, protoBytes); e != nil {
			subscriber.Stop(e)
		}
	}
}

// Start configures and starts block data subscriber lifecycle goroutines
func (b *BlockDataSubscriber) Start() {
	b.conn.SetReadLimit(b.manager.rcSubscriberReadLimitBytes)
	_ = b.conn.SetReadDeadline(time.Now().Add(b.manager.rcSubscriberPongWait))
	b.conn.SetPongHandler(func(string) error {
		_ = b.conn.SetReadDeadline(time.Now().Add(b.manager.rcSubscriberPongWait))
		return nil
	})
	go b.readLoop()
	go b.pingLoop()
}

func (b *BlockDataSubscriber) readLoop() {
	for {
		if _, _, err := b.conn.ReadMessage(); err != nil {
			b.Stop(err)
			return
		}
	}
}

func (b *BlockDataSubscriber) pingLoop() {
	ticker := time.NewTicker(b.manager.rcSubscriberPingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		if err := b.writeMessage(websocket.PingMessage, nil); err != nil {
			b.Stop(err)
			return
		}
	}
}

func (b *BlockDataSubscriber) writeMessage(messageType int, data []byte) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	_ = b.conn.SetWriteDeadline(time.Now().Add(b.manager.rcSubscriberWriteTimeout))
	return b.conn.WriteMessage(messageType, data)
}

// Stop stops the block data subscriber
func (b *BlockDataSubscriber) Stop(err error) {
	b.log.Errorf("BlockData WS Failed with err: %s", err.Error())
	if err = b.conn.Close(); err != nil {
		b.log.Error(err.Error())
	}
	b.manager.RemoveBlockDataSubscriber(b)
}
