package eth

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// safeBlockConfirmations is the number of confirmations required for a block to be considered safe
	safeBlockConfirmations = 5
	// retryDelay is the delay between connection retry attempts
	retryDelay = 5 * time.Second
	// length of valid order ids, in bytes
	orderIdLenBytes = 20
)

var _ types.BlockProvider = &EthBlockProvider{} // Ensures *EthBlockProvider implements BlockProvider interface

/* This file contains the high level functionality of the continued agreement on the blocks of the chain */

// EthereumRpcClient interface for ethereum rpc operations
type EthereumRpcClient interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*ethtypes.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error)
}

// EthereumWsClient interface for ethereum websocket operations
type EthereumWsClient interface {
	SubscribeNewHead(context.Context, chan<- *ethtypes.Header) (ethereum.Subscription, error)
}

// EthBlockProvider provides ethereum blocks through a channel
type EthBlockProvider struct {
	rpcURL          string            // rpc connection url
	wsURL           string            // websocket connection url
	blockChan       chan types.BlockI // channel to send safe blocks
	erc20TokenCache *ERC20TokenCache  // erc20 token info cache
	logger          lib.LoggerI       // logger for debug and error messages
	rpcClient       EthereumRpcClient // rpc client for fetching blocks
	wsClient        EthereumWsClient  // websocket client for monitoring headers
	nextHeight      uint64            // next block height to be sent through channel
}

// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(rpcURL string, wsURL string, blockChan chan types.BlockI, tokenCache *ERC20TokenCache, logger lib.LoggerI) *EthBlockProvider {
	// create new p instance
	p := &EthBlockProvider{
		rpcURL:          rpcURL,
		wsURL:           wsURL,
		blockChan:       blockChan,
		erc20TokenCache: tokenCache,
		logger:          logger,
	}
	// log provider creation
	p.logger.Infof("created ethereum block provider with rpc: %s, ws: %s", rpcURL, wsURL)
	return p
}

// SetHeight sets the next block height that should be sent through the channel
func (p *EthBlockProvider) SetHeight(height uint64) {
	// set the next height to process
	p.nextHeight = height
	// log the height setting
	p.logger.Infof("set next block height to: %d", height)
}

// getBlockByHeight gets the block at the specified height
func (p *EthBlockProvider) getBlockByHeight(height uint64) (*Block, error) {
	// log block retrieval attempt
	p.logger.Debugf("fetching block at height: %d", height)
	// create context for the request
	ctx := context.Background()
	// convert height to big.Int
	blockNumber := big.NewInt(int64(height))
	// fetch block from ethereum client
	ethBlock, err := p.rpcClient.BlockByNumber(ctx, blockNumber)
	if err != nil {
		// log error and return
		p.logger.Errorf("BlockByNumber failed at height %d: %v", height, err)
		return nil, err
	}
	// create new block from ethereum block
	block, err := NewBlock(ethBlock)
	if err != nil {
		// log error and return
		p.logger.Errorf("failed to wrap block at height %d: %v", height, err)
		return nil, err
	}
	// log successful block creation
	p.logger.Debugf("successfully created block at height: %d with %d transactions", height, len(block.transactions))
	return block, nil
}

// BlockCh returns the channel this provider will send new blocks through
func (p *EthBlockProvider) BlockCh() chan types.BlockI {
	// return the block channel
	return p.blockChan
}

// Start begins the block provider operation
func (p *EthBlockProvider) Start() {
	// log start of provider
	p.logger.Info("starting ethereum block provider")
	// run the provider body in a goroutine
	go func() {
		// connect to ethereum clients with retry
		p.connectWithRetry()
		// start monitoring new block headers
		p.monitorHeaders()
	}()
}

// connectWithRetry establishes connections to ethereum clients with retry logic
func (p *EthBlockProvider) connectWithRetry() {
	// retry connection indefinitely
	for {
		// attempt to connect to rpc client
		rpcClient, err := ethclient.Dial(p.rpcURL)
		if err != nil {
			// log error and retry
			p.logger.Errorf("failed to connect to rpc client: %v, retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		// set rpc client
		p.rpcClient = rpcClient
		// log successful rpc connection
		p.logger.Info("successfully connected to ethereum rpc client")
		// attempt to connect to websocket client
		wsClient, err := ethclient.Dial(p.wsURL)
		if err != nil {
			// log error and retry
			p.logger.Errorf("failed to connect to websocket client: %v, retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		// set websocket client
		p.wsClient = wsClient
		// log successful websocket connection
		p.logger.Info("successfully connected to ethereum websocket client")
		break
	}
}

// monitorHeaders monitors new block headers and processes safe blocks
func (p *EthBlockProvider) monitorHeaders() {
	// create header channel
	headerChan := make(chan *ethtypes.Header)
	// create context for subscription
	ctx := context.Background()
	// subscribe to new headers
	sub, err := p.wsClient.SubscribeNewHead(ctx, headerChan)
	if err != nil {
		// log error and return
		p.logger.Fatalf("failed to subscribe to new headers: %v", err)
		return
	}
	// log successful subscription
	p.logger.Info("successfully subscribed to new block headers")
	// process headers in loop
	for {
		select {
		case header := <-headerChan:
			// log new header received
			p.logger.Debugf("received new header at height: %d", header.Number.Uint64())
			// process safe blocks up to current height
			p.processSafeBlocks(header.Number.Uint64())
		case err := <-sub.Err():
			// log subscription error
			p.logger.Errorf("header subscription error: %v", err)
			// reconnect and restart monitoring
			p.connectWithRetry()
			p.monitorHeaders()
			return
		}
	}
}

// processSafeBlocks processes safe blocks up to the current height
func (p *EthBlockProvider) processSafeBlocks(currentHeight uint64) {
	// calculate safe height with confirmations
	safeHeight := currentHeight - safeBlockConfirmations
	// log safe height calculation
	p.logger.Debugf("oracle eth block provider processing safe blocks up to height: %d (current: %d)", safeHeight, currentHeight)
	// use safe height if next height is 0
	if p.nextHeight == 0 {
		p.nextHeight = safeHeight
		p.logger.Infof("Oracle eth block provider initialized next height to safe height: %d", safeHeight)
	}
	// process blocks from next height to safe height
	for p.nextHeight <= safeHeight {
		// log block processing attempt
		p.logger.Debugf("processing block at height: %d", p.nextHeight)
		// get block from ethereum node and create our Block wrapper
		block, err := p.getBlockByHeight(p.nextHeight)
		if err != nil {
			// log error and return without continuing
			p.logger.Errorf("failed to get block at height %d: %v", p.nextHeight, err)
			return
		}

		// Examine transactions for any erc20 transfers
		for _, tx := range block.transactions {
			// populate each transaction's tokenTransfer field
			p.checkTransfer(tx)
			// send block through channel
		}
		p.blockChan <- block
		// log successful block processing
		p.logger.Infof("Oracle eth block provider sent safe block at height: %d through channel", p.nextHeight)
		// increment next height
		p.nextHeight++
	}
}

// checkTransfer examines the transaction, populating tokenTransfer if there was an ERC20 token transfer
func (p *EthBlockProvider) checkTransfer(tx *Transaction) {
	// parse erc20 transfer from transaction data
	recipient, amount, extraData, err := parseERC20Transfer(tx.tx.Data())
	if err != nil {
		// not an erc20 transfer
		return
	}
	// ERC20 transfers with close/lock orders will have extra data
	if len(extraData) == 0 {
		// No order json data present
		return
	}
	// Validate extra data, ensure it is a valid close order
	if !validateCloseOrder(extraData) {
		p.logger.Errorf("failed to validate close order data for tx %s", tx.tx.Hash().String())
		return
	}
	// Get transaction receipt
	receipt, err := p.rpcClient.TransactionReceipt(context.Background(), tx.tx.Hash())
	if err != nil {
		p.logger.Errorf("failed to get transaction receipt for tx %s: %s", tx.tx.Hash().String(), err.Error())
		return
	}
	// Ensure this transaction was successful
	if receipt.Status != 1 {
		p.logger.Errorf("transaction %s with ERC20 transfer was a failed transaction, ignoring", tx.tx.Hash().String())
		return
	}
	p.logger.Debugf("detected erc20 transfer in tx: %s, recipient: %s, amount: %f", tx.Hash(), recipient, amount)
	tokenInfo, err := p.erc20TokenCache.TokenInfo(tx.To())
	if err != nil {
		// log error getting token info
		p.logger.Errorf("failed to get token info for contract %s: %v", tx.To(), err)
		return
	}
	// populate token transfer information
	err = tx.populateTokenTransfer(tokenInfo, recipient, amount)
	if err != nil {
		// log error populating token transfer
		p.logger.Errorf("failed to populate token transfer for tx %s: %v", tx.Hash(), err)
		return
	}
	// log successful token transfer population
	p.logger.Debugf("populated token transfer for tx: %s, token: %s, amount: %f", tx.Hash(), tokenInfo.Symbol, amount)
	// Store any data present that was after the ABI encoded ERC20 function call data
	tx.extraData = extraData
}

// validateCloseOrder ensures data contains valid json with expected fields
func validateCloseOrder(data []byte) bool {
	var rawData map[string]interface{}
	err := json.Unmarshal(data, &rawData)
	if err != nil {
		return false
	}

	// Check if required fields are present
	requiredFields := []string{"orderId", "chain_id", "closeOrder"}
	for _, field := range requiredFields {
		if _, exists := rawData[field]; !exists {
			return false
		}
	}
	// Number of fields must be 3
	if len(rawData) != 3 {
		return false
	}
	closeOrder := lib.CloseOrder{}
	err = closeOrder.UnmarshalJSON(data)
	if err != nil {
		return false
	}

	if len(closeOrder.OrderId) != orderIdLenBytes {
		return false
	}

	return true
}
