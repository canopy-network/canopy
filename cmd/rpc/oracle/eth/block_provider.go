package eth

import (
	"context"
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
	headerChannelBufferSize = 10
	// timeout for the transaction receipt call
	transactionReceiptTimeoutS = 5
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
	rpcUrl                 string            // rpc connection url
	wsUrl                  string            // websocket connection url
	blockChan              chan types.BlockI // channel to send safe blocks
	erc20TokenCache        *ERC20TokenCache  // erc20 token info cache
	logger                 lib.LoggerI       // logger for debug and error messages
	rpcClient              EthereumRpcClient // rpc client for fetching blocks
	wsClient               EthereumWsClient  // websocket client for monitoring headers
	orderValidator         OrderValidator    // order validator
	nextHeight             *big.Int          // next block height to be sent through channel
	chainId                uint64            // ethereum chain id
	retryDelay             time.Duration     // retry delay for connection failures
	safeBlockConfirmations *big.Int          // number of confirmations required for a block to be considered safe
	heightMu               *sync.Mutex       // mutex around next height
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
	// create the block output channel
	ch := make(chan types.BlockI)
	// create new provider instance
	p := &EthBlockProvider{
		rpcUrl:                 config.NodeUrl,
		wsUrl:                  config.NodeWSUrl,
		blockChan:              ch,
		erc20TokenCache:        tokenCache,
		logger:                 logger,
		chainId:                config.EVMChainId,
		orderValidator:         orderValidator,
		retryDelay:             time.Duration(config.RetryDelay) * time.Second,
		safeBlockConfirmations: big.NewInt(int64(config.SafeBlockConfirmations)),
		heightMu:               &sync.Mutex{},
	}
	// log provider creation
	p.logger.Infof("created ethereum block provider with rpc: %s, ws: %s, eth chain id: %d", p.rpcUrl, p.wsUrl, p.chainId)
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
			return nil, err // return error if transaction creation fails
		}
		// append transaction to block's transaction list
		block.transactions = append(block.transactions, tx)
	}
	// log successful block creation
	// p.logger.Debugf("successfully created block at height: %d with %d transactions", height, len(block.transactions))
	return block, nil
}

// BlockCh returns the channel this provider will send new blocks through
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
			case <-time.After(p.retryDelay):
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
	rpcClient, err := ethclient.DialContext(ctx, p.rpcUrl)
	if err != nil {
		// log error and retry
		p.logger.Errorf("Failed to connect to rpc client: %v, retrying in %v", err, p.retryDelay)
		return err
	}
	// set rpc client
	p.rpcClient = rpcClient
	// log successful rpc connection
	p.logger.Infof("Successfully connected to ethereum RPC at %s", p.rpcUrl)
	// attempt to connect to websocket client
	wsClient, err := ethclient.DialContext(ctx, p.wsUrl)
	if err != nil {
		p.rpcClient.Close()
		// log error and retry
		p.logger.Errorf("Failed to connect to websocket client: %v, retrying in %v", err, p.retryDelay)
		return err
	}
	// set websocket client
	p.wsClient = wsClient
	// log successful websocket connection
	p.logger.Infof("Websockets successfully connected to ethereum node at %s", p.wsUrl)
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
			// p.logger.Debugf("received new header at height: %d", header.Number.Uint64())
			// process safe blocks up to current height
			p.processBlocks(ctx, header.Number)
		case err := <-sub.Err():
			sub.Unsubscribe()
			return err
		}
	}
}

// processBlocks calculates the current safe height based on the received current height
// once the safe height is determined all unsent blocks up to that height will be sent to
// the consumer
func (p *EthBlockProvider) processBlocks(ctx context.Context, currentHeight *big.Int) {
	p.heightMu.Lock()
	defer p.heightMu.Unlock()
	// calculate safe height with confirmations
	safeHeight := new(big.Int).Sub(currentHeight, p.safeBlockConfirmations)
	// Ensure safe height is not negative
	if safeHeight.Sign() < 0 {
		safeHeight.SetInt64(0) // or handle the error case appropriately
	}
	// log safe height calculation
	// p.logger.Debugf("eth block provider processing safe blocks up to height: %d (current: %d)", safeHeight, currentHeight)
	// use safe height if next height is 0
	if p.nextHeight.Cmp(big.NewInt(0)) == 0 {
		p.nextHeight = new(big.Int).Set(safeHeight)
		p.logger.Warnf("eth block provider next expected height was 0 - initialized next height to safe height: %s", safeHeight.String())
	}
	// process blocks from next height to safe height
	for p.nextHeight.Cmp(safeHeight) <= 0 {
		// get block from ethereum node and create our Block wrapper
		block, err := p.fetchBlock(ctx, p.nextHeight)
		if err != nil {
			// log error and return without continuing
			p.logger.Errorf("failed to get block at height %d: %v", p.nextHeight, err)
			return
		}
		// process each transaction, populating orders and transfer data
		if err := p.processBlockTransactions(ctx, block); err != nil {
			p.logger.Errorf("failed to process block transactions: %v", err)
			return
		}
		// send block through channel
		p.blockChan <- block
		// log successful block processing
		// p.logger.Infof("eth block provider sent safe block at height %d through channel", p.nextHeight)
		// increment next height
		var one = big.NewInt(1)
		p.nextHeight.Add(p.nextHeight, one)
	}
}

// processBlockTransactions validates and processes block transactions
func (p *EthBlockProvider) processBlockTransactions(ctx context.Context, block *Block) error {
	// perform validation on transactions that had canopy orders
	for _, tx := range block.transactions {
		// examine transaction data for canopy orders
		err := tx.parseDataForOrders(p.orderValidator)
		if err != nil {
			p.logger.Warnf("Error parsing data for orders: %w", err)
			continue
		}
		// look for a canopy order in this transaction
		if tx.order == nil {
			p.logger.Warnf("Transaction had no Canopy order, %s", string(tx.tx.Data()))
			// no orders found, no processing required
			continue
		}
		// set the ethereum height this order was witnessed
		tx.order.WitnessedHeight = block.Number()
		// a valid canopy order was found, check transaction success
		if !p.transactionSuccess(ctx, tx) {
			// ignore all orders in failed transactions
			tx.clearOrder()
			// process next transaction
			continue
		}
		// test if this was an erc20 transfer
		if !tx.isERC20 {
			// no more processing required
			continue
		}
		// fetch erc20 token info (name, symbol, decimals)
		tokenInfo, err := p.erc20TokenCache.TokenInfo(ctx, tx.To())
		if err != nil {
			p.logger.Errorf("failed to get token info for contract %s: %v", tx.To(), err)
			// close order not valid if token info call fails
			tx.clearOrder()
			continue
		}
		// store the erc20 token info
		tx.tokenInfo = tokenInfo
	}
	return nil
}

// transactionSuccess fetches the transaction receipt and determines transaction success
// This prevents potential exploits or bugs where failed ERC20 transactions are processed
func (p *EthBlockProvider) transactionSuccess(ctx context.Context, tx *Transaction) bool {
	// create a fresh context with timeout for the RPC call
	rpcCtx, cancel := context.WithTimeout(ctx, transactionReceiptTimeoutS*time.Second)
	defer cancel()

	// get transaction receipt
	receipt, err := p.rpcClient.TransactionReceipt(rpcCtx, tx.tx.Hash())
	if err != nil {
		p.logger.Errorf("failed to get transaction receipt for tx %s: %s", tx.tx.Hash().String(), err.Error())
		return false
	}
	// check for success
	if receipt.Status != 1 {
		p.logger.Errorf("transaction %s with ERC20 transfer was a failed transaction, ignoring", tx.tx.Hash().String())
		return false
	}
	return true
}
