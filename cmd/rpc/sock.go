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
	"github.com/canopy-network/canopy/lib"
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
		subscriber.writeMu.Lock()
		_ = subscriber.conn.SetWriteDeadline(time.Now().Add(r.rcSubscriberWriteTimeout))
		// publish to each client
		if e := subscriber.conn.WriteMessage(websocket.BinaryMessage, protoBytes); e != nil {
			subscriber.writeMu.Unlock()
			subscriber.Stop(e)
			continue
		}
		subscriber.writeMu.Unlock()
	}
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
		r.writeMu.Lock()
		_ = r.conn.SetWriteDeadline(time.Now().Add(r.manager.rcSubscriberWriteTimeout))
		if err := r.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			r.writeMu.Unlock()
			r.Stop(err)
			return
		}
		r.writeMu.Unlock()
	}
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
