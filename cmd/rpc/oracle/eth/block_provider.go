package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// header channel buffer size
	headerChannelBufferSize = 1000
	// timeout for the transaction receipt call
	transactionReceiptTimeoutS = 5
	// ethereum transaction receipt success status value
	TransactionStatusSuccess = 1
	// how many times to try to process a transaction (erc20 token fetch + transaction receipt)
	maxTransactionProcessAttempts = 3
	// how long to allow processBlock to run
	processBlockTimeLimitS = 12
)

// Ensures *EthBlockProvider implements BlockProvider interface
var _ types.BlockProvider = &EthBlockProvider{}

/* This file contains the high level functionality of the continued agreement on the blocks of the chain */

// EthereumRpcClient interface for ethereum rpc operations
type EthereumRpcClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error)
	Close()
}

// EthereumWsClient interface for ethereum websocket operations
type EthereumWsClient interface {
	SubscribeNewHead(context.Context, chan<- *ethtypes.Header) (ethereum.Subscription, error)
	Close()
}

type OrderValidator interface {
	ValidateOrderJsonBytes(jsonBytes []byte, orderType types.OrderType) error
}

// EthBlockProvider provides ethereum blocks through a channel
type EthBlockProvider struct {
	config          lib.EthBlockProviderConfig // provider configuration
	blockChan       chan types.BlockI          // channel to send blocks
	erc20TokenCache *ERC20TokenCache           // erc20 token info cache
	logger          lib.LoggerI                // logger for debug and error messages
	rpcClient       EthereumRpcClient          // rpc client for fetching blocks
	wsClient        EthereumWsClient           // websocket client for monitoring headers
	orderValidator  OrderValidator             // order validator
	nextHeight      *big.Int                   // next block height to be sent through channel
	chainId         uint64                     // ethereum chain id
	synced          bool                       // flag indicating synced to top
	heightMu        *sync.Mutex                // mutex around next height
	metrics         *lib.Metrics               // metrics for telemetry
}

// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(config lib.EthBlockProviderConfig, orderValidator OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) *EthBlockProvider {
	// create an ethereum client for the token cache
	ethClient, ethErr := ethclient.Dial(config.NodeUrl)
	if ethErr != nil {
		logger.Fatal("[ETH-CONN] " + ethErr.Error())
	}
	// create a new erc20 token cache
	tokenCache := NewERC20TokenCache(ethClient, metrics)
	// create the block output channel, this is unbuffered so the provider
	// halts processing until the receiver is ready to process more blocks
	ch := make(chan types.BlockI)
	// create new provider instance
	p := &EthBlockProvider{
		config:          config,
		blockChan:       ch,
		erc20TokenCache: tokenCache,
		logger:          logger,
		orderValidator:  orderValidator,
		nextHeight:      big.NewInt(0),
		chainId:         config.EVMChainId,
		synced:          false,
		heightMu:        &sync.Mutex{},
		metrics:         metrics,
	}
	// log provider creation
	p.logger.Infof("[ETH-CONN] created block provider with rpc: %s, ws: %s, chain id: %d", p.config.NodeUrl, p.config.NodeWSUrl, p.chainId)
	return p
}

// fetchBlock fetches the block at the specified height and wraps each transaction
func (p *EthBlockProvider) fetchBlock(ctx context.Context, height *big.Int) (*Block, error) {
	// fetch block from ethereum client
	ethBlock, err := p.rpcClient.BlockByNumber(ctx, height)
	if err != nil {
		// log error and return
		p.logger.Errorf("[ETH-RPC] BlockByNumber failed for height %d: %v", height, err)
		return nil, err
	}
	// create new block from ethereum block
	block, err := NewBlock(ethBlock)
	if err != nil {
		// log error and return
		p.logger.Errorf("[ETH-BLOCK] failed to wrap block at height %d: %v", height, err)
		return nil, err
	}
	// iterate through ethereum transactions, creating a transaction wrappers
	for _, ethTx := range ethBlock.Transactions() {
		// create new Transaction from ethereum transaction
		tx, err := NewTransaction(ethTx, p.chainId)
		if err != nil {
			p.logger.Errorf("[ETH-TX] failed to create transaction: %s", err)
			continue
			// return nil, err // return error if transaction creation fails
		}
		// append transaction to block's transaction list
		block.transactions = append(block.transactions, tx)
	}
	// log successful block creation
	// p.logger.Debugf("successfully created block at height: %d with %d transactions", height, len(block.transactions))
	return block, nil
}

// BlockCh returns the channel through which new blocks will be sent
func (p *EthBlockProvider) BlockCh() chan types.BlockI {
	// return the block channel
	return p.blockChan
}

// IsSynced returns whether the block provider has synced to the top of the chain
func (p *EthBlockProvider) IsSynced() bool {
	// lock height mutex to safely read synced state
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	// return current sync status
	return p.synced
}

func (p *EthBlockProvider) closeConnections() {
	if p.rpcClient != nil {
		p.rpcClient.Close()
	}
	if p.wsClient != nil {
		p.wsClient.Close()
	}
}

// Start begins the block provider operation
func (p *EthBlockProvider) Start(ctx context.Context, height uint64) {
	p.nextHeight = new(big.Int).SetUint64(height)
	p.logger.Info("[ETH-CONN] starting block provider")
	go p.run(ctx)
}

// run handles the main loop for block provider operations
func (p *EthBlockProvider) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("[ETH-CONN] shutting down block provider")
			p.closeConnections()
			return
		default:
		}
		// try to connect to ethereum node
		err := p.connect(ctx)
		if err != nil {
			p.logger.Errorf("[ETH-CONN] error connecting to node: %s", err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(p.config.RetryDelay) * time.Second):
				continue
			}
		}
		// fetch latest block
		block, err := p.rpcClient.BlockByNumber(ctx, nil)
		if err != nil {
			p.logger.Errorf("[ETH-RPC] error fetching latest block: %s", err.Error())
			continue
		}
		// a next height of zero indicates no height was specified by the consumer
		if lib.BigIntIsZero(p.nextHeight) {
			// default to startup block depth
			p.nextHeight = lib.BigIntSub(block.Number(), lib.BigInt(p.config.StartupBlockDepth))
			// ensure next height is not negative
			if p.nextHeight.Sign() < 0 {
				p.nextHeight.SetInt64(0)
			}
			p.logger.Warnf("[ETH-SYNC] next height was 0 - initialized to %s", p.nextHeight.String())
		}
		// begin monitoring new block headers
		err = p.monitorHeaders(ctx)
		if err != nil {
			p.logger.Errorf("[ETH-WS] subscription error: %v", err)
		}
		// close any remaining connections
		p.closeConnections()
	}
}

// connect creates ethereum rpc and websocket connections
func (p *EthBlockProvider) connect(ctx context.Context) error {
	// close any existing connections
	p.closeConnections()
	// attempt to connect to rpc client
	rpcClient, err := ethclient.DialContext(ctx, p.config.NodeUrl)
	if err != nil {
		// log error and retry
		p.logger.Errorf("[ETH-CONN] failed to connect to rpc client: %v, retrying in %v", err, time.Duration(p.config.RetryDelay)*time.Second)
		return err
	}
	// set rpc client
	p.rpcClient = rpcClient
	// log successful rpc connection
	p.logger.Infof("[ETH-CONN] connected to RPC at %s", p.config.NodeUrl)
	// attempt to connect to websocket client
	wsClient, err := ethclient.DialContext(ctx, p.config.NodeWSUrl)
	if err != nil {
		p.rpcClient.Close()
		// log error and retry
		p.logger.Errorf("[ETH-WS] failed to connect: %v, retrying in %v", err, time.Duration(p.config.RetryDelay)*time.Second)
		return err
	}
	// set websocket client
	p.wsClient = wsClient
	// log successful websocket connection
	p.logger.Infof("[ETH-WS] connected at %s", p.config.NodeWSUrl)
	return nil
}

// monitorHeaders establishes a websocket subscription to monitor new block headers,
// prcessing them as they arrive from the Ethereum network.
// a received header acts as a notification that a new block has been created on ethereum,
// and our ethereum block provider should execute a process loop
func (p *EthBlockProvider) monitorHeaders(ctx context.Context) error {
	if p.wsClient == nil {
		return fmt.Errorf("websocket client not initialized")
	}
	// create header channel
	headerCh := make(chan *ethtypes.Header, headerChannelBufferSize)
	// subscribe to new headers
	sub, err := p.wsClient.SubscribeNewHead(ctx, headerCh)
	if err != nil {
		// log error and return
		p.logger.Errorf("[ETH-WS] failed to subscribe to headers: %v", err)
		return err
	}
	// log successful subscription
	p.logger.Info("[ETH-WS] subscribed to new headers")
	// create status ticker for periodic updates
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()
	// process headers in loop
	for {
		select {
		case <-statusTicker.C:
			// periodic status update
			p.logger.Infof("[ETH-SYNC] status: nextHeight=%s, synced=%v", p.nextHeight.String(), p.synced)
		case <-ctx.Done():
			p.logger.Info("[ETH-SYNC] context cancelled")
			sub.Unsubscribe()
			return ctx.Err()
		case header := <-headerCh:
			if header == nil || header.Number == nil {
				p.logger.Warn("[ETH-BLOCK] received nil header, skipping")
				continue
			}
			// ensure we haven't gotten ahead of the current chain height
			if p.nextHeight.Cmp(header.Number) > 0 {
				p.logger.Errorf("[ETH-SYNC] next height %d higher than current chain height %d", p.nextHeight, header.Number)
				p.logger.Error("[ETH-SYNC] remove state file and restart node if expected")
				// unsubscribe from new headers
				sub.Unsubscribe()
				// stop listening to new headers and return an error
				return ErrSourceHeight
			}
			// not synced to top
			if !p.synced {
				// check for source chain sync
				if p.nextHeight.Cmp(header.Number) == 0 {
					// we've caught up to the latest block, mark as synced
					p.synced = true
					p.logger.Infof("[ETH-SYNC] synced at height %s", p.nextHeight.String())
				}
			}
			// process all blocks up to current height
			p.nextHeight = p.processBlocks(ctx, p.nextHeight, header.Number)
		case err := <-sub.Err():
			// unsubscribe from new headers
			sub.Unsubscribe()
			// return the error
			return err
		}
	}
}

// processBlocks fetches and processes ethereum blocks in the specified range
func (p *EthBlockProvider) processBlocks(ctx context.Context, start, end *big.Int) *big.Int {
	// Create a context with ethereum block time timeout
	// this is so this method does not block new eth neaders
	timeoutCtx, cancel := context.WithTimeout(ctx, processBlockTimeLimitS*time.Second)
	defer cancel()
	// track next height to be processed
	next := new(big.Int).Set(start)
	p.logger.Debugf("[ETH-BLOCK] processing blocks from %d to %d", start, end)
	// initialize metrics counters
	var blocksProcessed, transactionsProcessed, retries int
	// process blocks from next height to current height
	for next.Cmp(end) <= 0 {
		// Check if context has been cancelled or timed out
		select {
		case <-timeoutCtx.Done():
			p.logger.Errorf("[ETH-BLOCK] max run time hit, returning")
			return next
		default:
		}
		// get block from ethereum node and create our Block wrapper
		fetchStart := time.Now()
		block, err := p.fetchBlock(timeoutCtx, next)
		if err != nil {
			// log error and return without continuing
			p.logger.Errorf("[ETH-BLOCK] failed to get block at height %d: %v", next, err)
			// update metrics before returning
			p.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 0, 1, blocksProcessed, transactionsProcessed, retries)
			// return same height so the provider tries this block again
			return next
		}
		fetchTime := time.Since(fetchStart)
		// process each transaction, populating orders and transfer data
		txProcessStart := time.Now()
		if err := p.processBlockTransactions(timeoutCtx, block); err != nil {
			p.logger.Errorf("[ETH-TX] failed to process block transactions: %v", err)
			// update metrics before returning
			p.metrics.UpdateEthBlockProviderMetrics(fetchTime, 0, 0, 0, 0, 1, blocksProcessed, transactionsProcessed, retries)
			return next
		}
		txProcessTime := time.Since(txProcessStart)
		// send block through channel
		p.blockChan <- block
		// log successful block processing
		// p.logger.Infof("eth block provider sent safe block at height %d through channel", next)
		// update counters
		blocksProcessed++
		transactionsProcessed += len(block.transactions)
		// update metrics with current block data
		p.metrics.UpdateEthBlockProviderMetrics(fetchTime, txProcessTime, 0, 0, 0, 0, 1, len(block.transactions), 0)
		// increment height for next iteration
		next.Add(next, big.NewInt(1))
	}
	return next
}

// processBlockTransactions validates and processes block transactions
func (p *EthBlockProvider) processBlockTransactions(ctx context.Context, block *Block) error {
	// track retry count for metrics
	var retryCount int
	// perform validation on transactions that had canopy orders
	for _, tx := range block.transactions {
		var err error
		// retry logic for processing transaction
		for attempt := range maxTransactionProcessAttempts {
			// process transaction - look for orders
			err = p.processTransaction(ctx, block, tx)
			// success indicates no order found, or order successfully found and validated
			if err == nil {
				break
			}
			// error condition - clear any order data that may have been set
			tx.clearOrder()
			// these errors can be temporary network errors, all others should not be retried
			if !errors.Is(err, ErrTransactionReceipt) && !errors.Is(err, ErrTokenInfo) {
				p.logger.Errorf("[ETH-TX] error processing tx %s with order in block %s: %v", tx.Hash(), block.Hash(), err)
				// non-retryable error, break immediately
				break
			}
			p.logger.Errorf("[ETH-TX] error processing tx %s in block %s: %v - attempt %d", tx.Hash(), block.Hash(), err, attempt+1)
			// count retry for metrics
			if attempt > 0 {
				retryCount++
			}
			// implement exponential backoff for failed attempts
			if attempt < maxTransactionProcessAttempts-1 {
				backoffDuration := time.Duration(1<<attempt) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoffDuration):
					// continue to next attempt after backoff
				}
			}
		}
		if err != nil {
			p.logger.Errorf("[ETH-TX] tx %s in block %s failed after all attempts: %v", tx.Hash(), block.Hash(), err)
		}
	}
	// update retry metrics if there were retries
	if retryCount > 0 {
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 0, 0, 0, 0, retryCount)
	}
	return nil
}

// processTransaction processes a single transaction
// the return value of nil means there was no canopy order in this transaction and processing need not be retried
// a return of an error means there was a canopy order and what could be a temporary error. A retry should be attempted
func (p *EthBlockProvider) processTransaction(ctx context.Context, block *Block, tx *Transaction) error {
	// examine transaction data for canopy orders
	err := tx.parseDataForOrders(p.orderValidator)
	// check for error
	if err != nil {
		// p.logger.Warnf("Error parsing data for orders: %s", err)
		p.logAsciiBytes(tx.tx.Data())
		// a transaction having non-JSON data is an expected conditions
		return nil
	}
	// check if parseDataForOrders found an order
	if tx.order == nil {
		// dev output
		// no orders found, no processing required
		return nil
	}
	// set the ethereum height this order was witnessed
	tx.order.WitnessedHeight = block.Number()
	// a valid canopy order was found, check transaction success
	success, err := p.transactionSuccess(ctx, tx)
	// check for error
	if err != nil {
		p.logger.Errorf("[ETH-RPC] error fetching transaction receipt: %s", err.Error())
		// there was an error fetching the transaction receipt
		return err
	}
	if !success {
		// process next transaction
		return nil
	}
	// test if this was an erc20 transfer
	if !tx.isERC20 {
		// no more processing required
		return nil
	}
	// fetch erc20 token info (name, symbol, decimals)
	tokenInfo, err := p.erc20TokenCache.TokenInfo(ctx, tx.To())
	if err != nil {
		p.logger.Errorf("[ETH-ERC20] failed to get token info for %s: %v", tx.To(), err)
		return err
	}
	p.logger.Infof("[ETH-ERC20] obtained token info for %s: %s", tx.To(), tokenInfo)
	// store the erc20 token info
	tx.tokenInfo = tokenInfo
	return nil
}

// transactionSuccess fetches the transaction receipt and determines transaction success
// This prevents scenarios where failed ERC20 transactions are processed as successful transfers
func (p *EthBlockProvider) transactionSuccess(ctx context.Context, tx *Transaction) (bool, error) {
	txHash := tx.tx.Hash()
	txHashStr := txHash.String()

	// create a fresh context with timeout for the RPC call
	rpcCtx, cancel := context.WithTimeout(ctx, transactionReceiptTimeoutS*time.Second)
	// get transaction receipt with timing
	receiptStart := time.Now()
	receipt, err := p.rpcClient.TransactionReceipt(rpcCtx, txHash)
	cancel()
	receiptTime := time.Since(receiptStart)
	// check for error
	if err != nil {
		p.logger.Warnf("[ETH-RPC] failed to get receipt for tx %s: %v", txHashStr, err)
		// update receipt fetch metrics on error
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, receiptTime, 0, 0, 1, 0, 0, 0)
		return false, ErrTransactionReceipt
	}
	// check for success using transaction receipt status
	// This approach works for ALL ERC20 tokens including non-standard ones like USDT:
	//   - Receipt status reflects actual on-chain execution success/failure
	//   - Independent of function return values (USDT returns void, standard returns bool)
	//   - Also catches failures from paused contracts, blacklisted addresses, etc.
	// By validating via receipt rather than parsing return values, we sidestep all
	// token-specific quirks and get a universal success indicator.
	if receipt.Status == TransactionStatusSuccess {
		// update receipt fetch metrics on success
		p.metrics.UpdateEthBlockProviderMetrics(0, 0, receiptTime, 0, 0, 0, 0, 0, 0)
		return true, nil
	}
	p.logger.Errorf("[ETH-TX] tx %s ERC20 transfer failed on-chain, ignoring", txHashStr)
	// return unsuccessful transaction
	return false, nil
}

// logAsciiBytes logs the first 100 bytes of data only if all bytes are printable ASCII
func (p *EthBlockProvider) logAsciiBytes(data []byte) {
	if len(data) == 0 {
		return
	}
	// determine how many bytes to check (up to 100)
	limit := len(data)
	if limit > 100 {
		limit = 100
	}
	// check if ALL bytes are printable ASCII
	for i := 0; i < limit; i++ {
		if data[i] < 32 || data[i] > 126 {
			// non-ASCII byte found, don't log
			return
		}
	}
	// all bytes are printable ASCII, log them
	p.logger.Debugf("[ETH-TX] first 100 bytes ASCII: %s", string(data[:limit]))
}
