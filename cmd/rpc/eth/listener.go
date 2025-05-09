// Create a type that uses go-ethereum to generate a websocket client that:
// - Is called EthBlockListener and has a constructor called NewEthBlockListener
// - Constructor accepts two urls, one for rpc and one for websocket
// - Constructor accepts a channel that is used to send finalized blocks. It has the type BlockI
// - Constructor accepts a lib.LoggerI type that it uses for logging
// - Connects to an ethereum node
// - Listens for new block headers
// - When receiving a new block header, fetches the latest finalized block number
// - Fetches that block
// - Sends block through channel

// Add a package comment describing this as an ethereum block listener

// Use these imports
// "github.com/ethereum/go-ethereum"
// "github.com/ethereum/go-ethereum/core/types"
// "github.com/ethereum/go-ethereum/ethclient"
// "github.com/ethereum/go-ethereum/rpc"
// "math/big"
// "github.com/canopy-network/canopy/lib"
// To convert eth types to our types, use NewBlock(block) and NewTransaction(tx). These already exist in the current package

// This is how you get the last finalized block:
// var b FinalizedBlock
// err = rpcClient.Client().Call(&b, "eth_getBlockByNumber", "finalized", true)
// if err != nil {
// 	return nil, fmt.Errorf("failed to get finality status: %w", err)
// }
// number, err := strconv.ParseInt(b.Number, 0, 64)
// if err != nil {
// 	fmt.Println("Error converting from hex to int:", err)
// }

// package name is eth

// START
// Package eth provides an Ethereum block listener implementation that connects to
// an Ethereum node, listens for new block headers, and fetches the latest finalized
// blocks to send through a channel.
package eth

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// FinalizedBlock represents the response from eth_getBlockByNumber for the finalized block
type FinalizedBlock struct {
	Number string `json:"number"`
}

// EthBlockListener is responsible for listening to new Ethereum blocks and
// forwarding the finalized blocks through a channel.
type EthBlockListener struct {
	rpcURL       string
	wsURL        string
	blockCh      chan wstypes.BlockI
	logger       lib.LoggerI
	rpcClient    *ethclient.Client
	wsClient     *ethclient.Client
	rpcRawClient *rpc.Client
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewEthBlockListener creates a new instance of EthBlockListener.
func NewEthBlockListener(rpcURL, wsURL string, blockCh chan wstypes.BlockI, logger lib.LoggerI) *EthBlockListener {
	ctx, cancel := context.WithCancel(context.Background())

	return &EthBlockListener{
		rpcURL:  rpcURL,
		wsURL:   wsURL,
		blockCh: blockCh,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start initiates the EthBlockListener to connect to Ethereum node and
// begin listening for new blocks.
func (e *EthBlockListener) Start() error {
	e.logger.Info("Starting Ethereum block listener")

	var err error

	// Connect to RPC client
	e.rpcRawClient, err = rpc.Dial(e.rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC client: %w", err)
	}
	e.rpcClient = ethclient.NewClient(e.rpcRawClient)
	e.logger.Infof("Connected to RPC endpoint: %s", e.rpcURL)

	// Connect to WebSocket client
	wsRawClient, err := rpc.Dial(e.wsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket client: %w", err)
	}
	e.wsClient = ethclient.NewClient(wsRawClient)
	e.logger.Infof("Connected to WebSocket endpoint: %s", e.wsURL)

	// Subscribe to new block headers
	headers := make(chan *types.Header)
	sub, err := e.wsClient.SubscribeNewHead(e.ctx, headers)
	if err != nil {
		return fmt.Errorf("failed to subscribe to new block headers: %w", err)
	}
	e.logger.Info("Subscribed to new block headers")

	// Start the listener loop
	go e.listenForBlocks(headers, sub)

	return nil
}

// listenForBlocks handles incoming block headers and processes them.
func (e *EthBlockListener) listenForBlocks(headers chan *types.Header, sub ethereum.Subscription) {
	for {
		select {
		case err := <-sub.Err():
			e.logger.Errorf("Subscription error: %v", err)
			return

		case header := <-headers:
			e.logger.Debugf("Received new block header: %d", header.Number.Uint64())
			e.processFinalizedBlock()

		case <-e.ctx.Done():
			e.logger.Info("Block listener stopped")
			return
		}
	}
}

// processFinalizedBlock fetches the latest finalized block and sends it through the channel.
func (e *EthBlockListener) processFinalizedBlock() {
	blockNumber, err := e.getLatestFinalizedBlockNumber()
	if err != nil {
		e.logger.Errorf("Failed to get latest finalized block number: %v", err)
		return
	}

	e.logger.Infof("Latest finalized block number: %d", blockNumber)

	block, err := e.fetchBlock(blockNumber)
	if err != nil {
		e.logger.Errorf("Failed to fetch block %d: %v", blockNumber, err)
		return
	}

	// Send the block through the channel
	e.blockCh <- NewBlock(block)
	e.logger.Infof("Sent block %d through channel", blockNumber)
}

// getLatestFinalizedBlockNumber retrieves the latest finalized block number from the Ethereum node.
func (e *EthBlockListener) getLatestFinalizedBlockNumber() (int64, error) {
	var b FinalizedBlock
	err := e.rpcRawClient.Call(&b, "eth_getBlockByNumber", "finalized", true)
	if err != nil {
		return 0, fmt.Errorf("failed to get finality status: %w", err)
	}

	number, err := strconv.ParseInt(b.Number, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting from hex to int: %w", err)
	}

	return number, nil
}

// fetchBlock retrieves a specific block by number from the Ethereum node.
func (e *EthBlockListener) fetchBlock(blockNumber int64) (*types.Block, error) {
	e.logger.Debugf("Fetching block %d", blockNumber)

	block, err := e.rpcClient.BlockByNumber(e.ctx, big.NewInt(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block %d: %w", blockNumber, err)
	}

	return block, nil
}

// Stop terminates the EthBlockListener.
func (e *EthBlockListener) Stop() {
	e.logger.Info("Stopping Ethereum block listener")
	e.cancel()

	if e.rpcClient != nil {
		e.rpcClient.Close()
	}

	if e.wsClient != nil {
		e.wsClient.Close()
	}

	e.logger.Info("Ethereum block listener stopped")
}
