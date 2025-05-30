package eth

import (
	"context"
	"fmt"
	"math/big"
	"time"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// safeBlockConfirmations is the number of confirmations required for a block to be considered safe
	safeBlockConfirmations = 5
	// reconnectDelay is the delay between reconnection attempts
	reconnectDelay = 5 * time.Second
)

// EthBlockProvider implements the BlockProvider interface for Ethereum blocks
type EthBlockProvider struct {
	rpcURL     string              // rpc url for ethereum node connection
	wsURL      string              // websocket url for ethereum node connection
	blockChan  chan wstypes.BlockI // channel to send safe blocks through
	logger     lib.LoggerI         // logger for operation logging
	nextHeight uint64              // next expected block height to send
	rpcClient  *ethclient.Client   // rpc client for block retrieval
	wsClient   *ethclient.Client   // websocket client for header subscription
}

// NewEthBlockProvider creates a new EthBlockProvider instance
func NewEthBlockProvider(rpcURL string, wsURL string, blockChan chan wstypes.BlockI, logger lib.LoggerI) *EthBlockProvider {
	// create and return new provider instance
	return &EthBlockProvider{
		rpcURL:    rpcURL,
		wsURL:     wsURL,
		blockChan: blockChan,
		logger:    logger,
	}
}

// Start begins the block provider process
func (p *EthBlockProvider) Start() {
	// log start of block provider
	p.logger.Info("starting ethereum block provider")
	// continuously attempt to connect and process blocks
	for {
		// attempt to establish connections
		err := p.connect()
		if err != nil {
			// log connection failure and retry
			p.logger.Errorf("failed to connect to ethereum node: %v", err)
			time.Sleep(reconnectDelay)
			continue
		}
		// process blocks until connection is lost
		err = p.processBlocks()
		if err != nil {
			// log processing error and reconnect
			p.logger.Errorf("block processing error: %v", err)
			p.cleanup()
			time.Sleep(reconnectDelay)
			continue
		}
	}
}

// SetNext sets the next expected block height to be sent through channel
func (p *EthBlockProvider) SetNext(height uint64) {
	// set the next height to process
	p.nextHeight = height
	// log the height setting
	p.logger.Infof("set next block height to %d", height)
}

// BlockCh returns the channel this provider will send new blocks through
func (p *EthBlockProvider) BlockCh() chan wstypes.BlockI {
	// return the block channel
	return p.blockChan
}

// GetBlockByHeight gets the block at the specified height
func (p *EthBlockProvider) GetBlockByHeight(height uint64) (wstypes.BlockI, error) {
	// check if rpc client is available
	if p.rpcClient == nil {
		return nil, fmt.Errorf("Error getting block by height, client not connected")
	}
	// get block by number from ethereum node
	block, err := p.rpcClient.BlockByNumber(context.Background(), big.NewInt(int64(height)))
	if err != nil {
		// log error getting block
		p.logger.Errorf("failed to get block at height %d: %v", height, err)
		return nil, err
	}
	// create and return wrapped block
	return NewBlock(block), nil
}

// connect establishes connections to ethereum node
func (p *EthBlockProvider) connect() error {
	// establish rpc connection
	rpcClient, err := ethclient.Dial(p.rpcURL)
	if err != nil {
		return err
	}
	p.rpcClient = rpcClient
	// establish websocket connection
	wsClient, err := ethclient.Dial(p.wsURL)
	if err != nil {
		// cleanup rpc client on websocket failure
		rpcClient.Close()
		return err
	}
	p.wsClient = wsClient
	// log successful connection
	p.logger.Info("successfully connected to ethereum node")
	return nil
}

// processBlocks handles the main block processing loop
func (p *EthBlockProvider) processBlocks() error {
	// subscribe to new block headers
	headerChan := make(chan *types.Header)
	sub, err := p.wsClient.SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	// log subscription success
	p.logger.Info("subscribed to new block headers")
	// initialize next height if not set
	if p.nextHeight == 0 {
		// get current safe height
		safeHeight, err := p.getSafeHeight()
		if err != nil {
			return err
		}
		p.nextHeight = safeHeight
		// log initialization of next height
		p.logger.Infof("initialized next height to safe height: %d", p.nextHeight)
	}
	// process headers as they arrive
	for {
		select {
		case err := <-sub.Err():
			// subscription error occurred
			return err
		case header := <-headerChan:
			// new header received, process safe blocks
			p.logger.Debugf("received new header at height %d", header.Number.Uint64())
			err := p.processSafeBlocks()
			if err != nil {
				// log error processing safe blocks
				p.logger.Errorf("error processing safe blocks: %v", err)
				return err
			}
		}
	}
}

// processSafeBlocks sends all available safe blocks through the channel
func (p *EthBlockProvider) processSafeBlocks() error {
	// get current safe height
	safeHeight, err := p.getSafeHeight()
	if err != nil {
		return err
	}
	// send all blocks from next height to safe height
	for height := p.nextHeight; height <= safeHeight; height++ {
		// get block at current height
		block, err := p.GetBlockByHeight(height)
		if err != nil {
			// log error getting block
			p.logger.Errorf("failed to get block at height %d: %v", height, err)
			return err
		}
		// send block through channel
		p.blockChan <- block
		// log block sent
		p.logger.Infof("sent safe block at height %d through channel", height)
		// increment next height
		p.nextHeight = height + 1
	}
	return nil
}

// getSafeHeight calculates the current safe block height
func (p *EthBlockProvider) getSafeHeight() (uint64, error) {
	// get latest block number
	latestBlock, err := p.rpcClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	latestHeight := latestBlock.NumberU64()
	// calculate safe height with confirmations
	if latestHeight < safeBlockConfirmations {
		// not enough blocks for safe confirmations
		return 0, nil
	}
	safeHeight := latestHeight - safeBlockConfirmations
	// log safe height calculation
	p.logger.Debugf("calculated safe height: %d (latest: %d)", safeHeight, latestHeight)
	return safeHeight, nil
}

// cleanup closes connections and cleans up resources
func (p *EthBlockProvider) cleanup() {
	// close rpc client if exists
	if p.rpcClient != nil {
		p.rpcClient.Close()
		p.rpcClient = nil
	}
	// close websocket client if exists
	if p.wsClient != nil {
		p.wsClient.Close()
		p.wsClient = nil
	}
	// log cleanup completion
	p.logger.Debug("cleaned up ethereum client connections")
}
