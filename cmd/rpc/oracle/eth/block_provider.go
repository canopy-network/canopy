package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
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
	blockChan       chan types.BlockI          // channel to send safe blocks
	erc20TokenCache *ERC20TokenCache           // erc20 token info cache
	logger          lib.LoggerI                // logger for debug and error messages
	rpcClient       EthereumRpcClient          // rpc client for fetching blocks
	wsClient        EthereumWsClient           // websocket client for monitoring headers
	orderValidator  OrderValidator             // order validator
	nextHeight      *big.Int                   // next block height to be sent through channel
	safeHeight      *big.Int                   // the current safe height for the source chain
	chainId         uint64                     // ethereum chain id
	heightMu        *sync.Mutex                // mutex around next height
}

// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(config lib.EthBlockProviderConfig, orderValidator OrderValidator, logger lib.LoggerI) *EthBlockProvider {
	// create an ethereum client for the token cache
	ethClient, ethErr := ethclient.Dial(config.NodeUrl)
	if ethErr != nil {
		logger.Fatal(ethErr.Error())
	}
	// create a new erc20 token cache
	tokenCache := NewERC20TokenCache(ethClient)
	// create the block output channel, this is unbuffered so the provider
	// halts processing until the receiver is ready to process more blocks
	ch := make(chan types.BlockI)
	// create new provider instance
	p := &EthBlockProvider{
		config:          config,
		blockChan:       ch,
		erc20TokenCache: tokenCache,
		logger:          logger,
		chainId:         config.EVMChainId,
		orderValidator:  orderValidator,
		nextHeight:      big.NewInt(0),
		safeHeight:      big.NewInt(0),
		heightMu:        &sync.Mutex{},
	}
	// log provider creation
	p.logger.Infof("created ethereum block provider with rpc: %s, ws: %s, eth chain id: %d", p.config.NodeUrl, p.config.NodeWSUrl, p.chainId)
	return p
}

// SetHeight sets the next block height that the consumer wants to receive
func (p *EthBlockProvider) SetHeight(height *big.Int) {
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	// set the next height to process
	p.nextHeight = height
	// log the height setting
	p.logger.Infof("set next block height to: %d", height)
}

// fetchBlock fetches the block at the specified height and wraps each transaction
func (p *EthBlockProvider) fetchBlock(ctx context.Context, height *big.Int) (*Block, error) {
	// fetch block from ethereum client
	ethBlock, err := p.rpcClient.BlockByNumber(ctx, height)
	if err != nil {
		// log error and return
		p.logger.Errorf("BlockByNumber rpc called failed for height %d: %v", height, err)
		return nil, err
	}
	// create new block from ethereum block
	block, err := NewBlock(ethBlock)
	if err != nil {
		// log error and return
		p.logger.Errorf("failed to wrap block at height %d: %v", height, err)
		return nil, err
	}
	// iterate through ethereum transactions, creating a transaction wrappers
	for _, ethTx := range ethBlock.Transactions() {
		// create new Transaction from ethereum transaction
		tx, err := NewTransaction(ethTx, p.chainId)
		if err != nil {
			p.logger.Errorf("failed to create transaction: %s", height, err)
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

func (p *EthBlockProvider) closeConnections() {
	if p.rpcClient != nil {
		p.rpcClient.Close()
	}
	if p.wsClient != nil {
		p.wsClient.Close()
	}
}

// Start begins the block provider operation
func (p *EthBlockProvider) Start(ctx context.Context) {
	p.logger.Info("starting ethereum block provider")
	go p.run(ctx)
}

// run handles the main loop for block provider operations
func (p *EthBlockProvider) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("shutting down ethereum block provider")
			p.closeConnections()
			return
		default:
		}
		// try to connect to ethereum node
		err := p.connect(ctx)
		if err != nil {
			p.logger.Errorf("Error connecting to ethereum node: %s", err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(p.config.RetryDelay) * time.Second):
				continue
			}
		}
		// begin monitoring new block headers
		err = p.monitorHeaders(ctx)
		if err != nil {
			p.logger.Errorf("Subscription error: %v", err)
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
		p.logger.Errorf("Failed to connect to rpc client: %v, retrying in %v", err, time.Duration(p.config.RetryDelay)*time.Second)
		return err
	}
	// set rpc client
	p.rpcClient = rpcClient
	// log successful rpc connection
	p.logger.Infof("Successfully connected to ethereum RPC at %s", p.config.NodeUrl)
	// attempt to connect to websocket client
	wsClient, err := ethclient.DialContext(ctx, p.config.NodeWSUrl)
	if err != nil {
		p.rpcClient.Close()
		// log error and retry
		p.logger.Errorf("Failed to connect to websocket client: %v, retrying in %v", err, time.Duration(p.config.RetryDelay)*time.Second)
		return err
	}
	// set websocket client
	p.wsClient = wsClient
	// log successful websocket connection
	p.logger.Infof("Websockets successfully connected to ethereum node at %s", p.config.NodeWSUrl)
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
		p.logger.Errorf("failed to subscribe to new headers: %v", err)
		return err
	}
	// log successful subscription
	p.logger.Info("successfully subscribed to new block headers")
	// process headers in loop
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("header monitoring stopped due to context cancellation")
			sub.Unsubscribe()
			return ctx.Err()
		case header := <-headerCh:
			if header == nil || header.Number == nil {
				p.logger.Warn("received nil header or header number, skipping")
				continue
			}
			// update current safe height and next height
			p.updateHeights(header.Number)
			// process safe blocks up to current height
			p.processBlocks(ctx)
		case err := <-sub.Err():
			sub.Unsubscribe()
			return err
		}
	}
}

// updateHeights determines the safe height based on current height and confirmations
func (p *EthBlockProvider) updateHeights(currentHeight *big.Int) {
	// protect next height
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	safeBlocks := big.NewInt(int64(p.config.SafeBlockConfirmations))
	// calculate a safe height based on current source chain height
	calculatedSafeHeight := new(big.Int).Sub(currentHeight, safeBlocks)
	// ensure safe height is never negative
	if calculatedSafeHeight.Sign() < 0 {
		calculatedSafeHeight.SetInt64(0)
	}
	// a next height of zero indicates no height was specified by the consumer
	if p.nextHeight.Cmp(big.NewInt(0)) == 0 {
		startUp := new(big.Int).SetUint64(p.config.StartupBlockDepth)
		// default to startup block depth
		p.nextHeight = new(big.Int).Sub(currentHeight, startUp)
		// ensure next height is not negative
		if p.nextHeight.Sign() < 0 {
			p.nextHeight.SetInt64(0)
		}
		// set safe height to the newly calculated one
		p.safeHeight = calculatedSafeHeight
		p.logger.Warnf("eth block provider next height was 0 - initialized block depth %s", p.nextHeight.String())
		return
	}
	// validate that we haven't gotten ahead of the current chain height
	if p.nextHeight.Cmp(currentHeight) > 0 {
		p.logger.Errorf("eth block provider next expected source chain height was %d, higher than current source chain height: %d. If this is expected, remove state file and restart node. Exiting.", p.nextHeight, currentHeight)
		os.Exit(1)
	}
	// ensure safe height is never lowered
	if calculatedSafeHeight.Cmp(p.safeHeight) < 0 {
		p.logger.Warnf("calculated safe height %d is lower than current safe height %d, keeping current", calculatedSafeHeight, p.safeHeight)
		// no safe height update
		return
	}
	p.safeHeight = calculatedSafeHeight
}

// processBlocks calculates the current safe height based on the received current height
// and sends all unprocessed blocks up to the safe height to the consumer
func (p *EthBlockProvider) processBlocks(ctx context.Context) {
	// Create a context with ethereum block time timeout
	// this is so this method does not block new eth neaders
	timeoutCtx, cancel := context.WithTimeout(ctx, processBlockTimeLimitS*time.Second)
	defer cancel()

	// protect next height
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	p.logger.Debugf("block provider processing safe blocks from %d to %d", p.nextHeight, p.safeHeight)
	// process blocks from next height to safe height
	for p.nextHeight.Cmp(p.safeHeight) <= 0 { // nextHeight <= safeHeight
		// Check if context has been cancelled or timed out
		select {
		case <-timeoutCtx.Done():
			p.logger.Errorf("processBlocks timed out after 12 seconds")
			return
		default:
		}
		// get block from ethereum node and create our Block wrapper
		block, err := p.fetchBlock(timeoutCtx, p.nextHeight)
		if err != nil {
			// log error and return without continuing
			p.logger.Errorf("failed to get block at height %d: %v", p.nextHeight, err)
			return
		}
		// process each transaction, populating orders and transfer data
		if err := p.processBlockTransactions(timeoutCtx, block); err != nil {
			p.logger.Errorf("failed to process block transactions: %v", err)
			return
		}
		// send block through channel
		p.blockChan <- block
		// log successful block processing
		// p.logger.Infof("eth block provider sent safe block at height %d through channel", p.nextHeight)
		// increment next height
		p.nextHeight.Add(p.nextHeight, big.NewInt(1))
	}
}

// processBlockTransactions validates and processes block transactions
func (p *EthBlockProvider) processBlockTransactions(ctx context.Context, block *Block) error {
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
				p.logger.Errorf("Error processing transaction %s with Canopy order in block %s: %v", tx.Hash(), block.Hash(), err)
				// non-retryable error, break immediately
				break
			}
			p.logger.Errorf("Error processing transaction %s in block %s: %v - attempt %d", tx.Hash(), block.Hash(), attempt+1)
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
			p.logger.Errorf("Error processing transaction %s in block %s: %v - all attempts failed", tx.Hash(), block.Hash())
		}
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
		p.logger.Warnf("Error parsing data for orders: %w", err)
		return nil
	}
	// check if parseDataForOrders found an order
	if tx.order == nil {
		// dev output
		p.logger.Warnf("Transaction had no Canopy order, %s", string(tx.tx.Data()))
		// no orders found, no processing required
		return nil
	}
	// set the ethereum height this order was witnessed
	tx.order.WitnessedHeight = block.Number()
	// a valid canopy order was found, check transaction success
	success, err := p.transactionSuccess(ctx, tx)
	// check for error
	if err != nil {
		p.logger.Errorf("Error fetching transaction receipt: %s", err.Error())
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
		p.logger.Errorf("failed to get token info for contract %s: %v", tx.To(), err)
		return err
	}
	p.logger.Infof("Obtained token info for contract %x: %s", tx.To(), tokenInfo)
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
	// get transaction receipt
	receipt, err := p.rpcClient.TransactionReceipt(rpcCtx, txHash)
	cancel()
	// check for error
	if err != nil {
		p.logger.Warnf("failed to get transaction receipt for tx %s: %v", txHashStr, err)
		return false, ErrTransactionReceipt
	}
	// check for success
	if receipt.Status == TransactionStatusSuccess {
		return true, nil
	}
	p.logger.Errorf("transaction %s with ERC20 transfer was a failed on-chain transaction, ignoring", txHashStr)
	// return unsuccessful transaction
	return false, nil
}
