// Create a type that uses go-ethereum to generate a websocket client that:
// - Is called EthBlockListener and has a constructor called NewEthBlockListener
// - Constructor accepts two urls, one for rpc and one for websocket
// - Constructor accepts a channel that is used to send finalized blocks. It has the type BlockI
// - Constructor accepts a lib.LoggerI type that it uses for logging
// - Connects to an ethereum node
// - Listens for new block headers
// - Before you do any processing, verify that you haven't processed this block number already.
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
// Package eth provides functionality for interacting with Ethereum blockchain
// This package contains an ethereum block listener that watches for new finalized blocks
package eth

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	
	"github.com/canopy-network/canopy/lib"
	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// FinalizedBlock represents the response from the eth_getBlockByNumber RPC call
type FinalizedBlock struct {
	Number string `json:"number"`
}

// EthBlockListener listens for new Ethereum blocks and processes them
type EthBlockListener struct {
	rpcURL       string                 // URL for RPC connections
	wsURL        string                 // URL for WebSocket connections
	blockChan    chan wstypes.BlockI    // Channel to send processed blocks
	logger       lib.LoggerI            // Logger interface for logging messages
	rpcClient    *rpc.Client            // RPC client for standard HTTP requests
	wsClient     *rpc.Client            // WebSocket client for subscriptions
	ethClient    *ethclient.Client      // Ethereum client for blockchain interactions
	lastProcessedBlock uint64           // Last block number that has been processed
}

// NewEthBlockListener creates a new ethereum block listener
func NewEthBlockListener(rpcURL, wsURL string, blockChan chan wstypes.BlockI, logger lib.LoggerI) *EthBlockListener {
	// Create and return a new listener instance
	return &EthBlockListener{
		rpcURL:    rpcURL,
		wsURL:     wsURL,
		blockChan: blockChan,
		logger:    logger,
		lastProcessedBlock: 0,
	}
}

// Start initiates the connection to the ethereum node and begins listening for new blocks
func (l *EthBlockListener) Start(ctx context.Context) error {
	// Connect to the Ethereum node via RPC
	l.logger.Info("Connecting to Ethereum node via RPC")
	var err error
	l.rpcClient, err = rpc.Dial(l.rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum RPC: %w", err)
	}
	
	// Create an ethClient from the rpcClient
	l.ethClient = ethclient.NewClient(l.rpcClient)
	
	// Connect to the Ethereum node via WebSocket
	l.logger.Info("Connecting to Ethereum node via WebSocket")
	l.wsClient, err = rpc.Dial(l.wsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum WebSocket: %w", err)
	}
	
	// Subscribe to new block headers
	l.logger.Info("Subscribing to new block headers")
	headers := make(chan *types.Header)
	sub, err := l.wsClient.EthSubscribe(ctx, headers, "newHeads")
	if err != nil {
		return fmt.Errorf("failed to subscribe to new headers: %w", err)
	}
	
	// Start goroutine to handle new headers
	go l.handleNewHeaders(ctx, headers, sub)
	
	return nil
}

// handleNewHeaders processes incoming block headers from the subscription
func (l *EthBlockListener) handleNewHeaders(ctx context.Context, headers chan *types.Header, sub ethereum.Subscription) {
	// Handle received headers and subscription errors
	l.logger.Debug("Started handling new block headers")
	
	for {
		select {
		case err := <-sub.Err():
			// Handle subscription error
			l.logger.Errorf("Subscription error: %v", err)
			return
			
		case header := <-headers:
			// Process new header
			l.processHeader(ctx, header)
			
		case <-ctx.Done():
			// Context was cancelled, stop processing
			l.logger.Info("Context cancelled, stopping block listener")
			return
		}
	}
}

// processHeader handles a single block header
func (l *EthBlockListener) processHeader(ctx context.Context, header *types.Header) {
	// Log the received header
	l.logger.Debugf("Received new block header: %d", header.Number.Uint64())
	
	// Get the latest finalized block number
	finalizedBlockNum, err := l.getLatestFinalizedBlockNumber()
	if err != nil {
		l.logger.Errorf("Failed to get latest finalized block number: %v", err)
		return
	}
	
	// Check if we've already processed this block
	if finalizedBlockNum <= l.lastProcessedBlock {
		l.logger.Debugf("Block %d already processed, current last processed: %d", 
			finalizedBlockNum, l.lastProcessedBlock)
		return
	}
	
	// Fetch and process the finalized block
	l.fetchAndProcessBlock(ctx, finalizedBlockNum)
}

// getLatestFinalizedBlockNumber retrieves the latest finalized block number
func (l *EthBlockListener) getLatestFinalizedBlockNumber() (uint64, error) {
	// Get the latest finalized block number via RPC
	l.logger.Debug("Getting latest finalized block number")
	var b FinalizedBlock
	err := l.rpcClient.Call(&b, "eth_getBlockByNumber", "finalized", true)
	if err != nil {
		return 0, fmt.Errorf("failed to get finality status: %w", err)
	}
	
	// Parse the block number from hex string
	number, err := strconv.ParseInt(b.Number, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting from hex to int: %w", err)
	}
	
	l.logger.Debugf("Latest finalized block number: %d", number)
	return uint64(number), nil
}

// fetchAndProcessBlock retrieves a block by number and sends it through the block channel
func (l *EthBlockListener) fetchAndProcessBlock(ctx context.Context, blockNumber uint64) {
	// Fetch the block by number
	l.logger.Infof("Fetching block: %d", blockNumber)
	blockNum := big.NewInt(int64(blockNumber))
	block, err := l.ethClient.BlockByNumber(ctx, blockNum)
	if err != nil {
		l.logger.Errorf("Failed to fetch block %d: %v", blockNumber, err)
		return
	}
	
	// Convert to our block type and send through channel
	l.logger.Debugf("Processing block %d with %d transactions", 
		blockNumber, len(block.Transactions()))
	
	// Convert ethereum block to our BlockI interface
	ourBlock := NewBlock(block)
	
	// Send the block through the channel
	l.logger.Infof("Sending block %d to channel", blockNumber)
	select {
	case l.blockChan <- ourBlock:
		// Block was sent successfully
		l.logger.Debugf("Block %d sent successfully", blockNumber)
		// Update the last processed block
		l.lastProcessedBlock = blockNumber
	case <-ctx.Done():
		// Context was cancelled while sending
		l.logger.Warn("Context cancelled while sending block to channel")
		return
	}
}

// Stop closes the connections to the ethereum node
func (l *EthBlockListener) Stop() {
	// Clean up resources
	l.logger.Info("Stopping Ethereum block listener")
	if l.rpcClient != nil {
		l.rpcClient.Close()
	}
	if l.wsClient != nil {
		l.wsClient.Close()
	}
}
```
