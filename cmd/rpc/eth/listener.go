// Package eth provides an Ethereum block listener that connects to an Ethereum node,
// listens for new block headers, and processes finalized blocks.
package eth

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// FinalizedBlock represents the response from eth_getBlockByNumber for finalized blocks
type FinalizedBlock struct {
	Number string `json:"number"`
}

// EthBlockListener listens for new Ethereum blocks and sends finalized blocks through a channel
type EthBlockListener struct {
	rpcURL      string              // URL for RPC connection
	wsURL       string              // URL for WebSocket connection
	blockChan   chan wstypes.BlockI // Channel to send finalized blocks
	logger      lib.LoggerI         // Logger for verbose logging
	rpcClient   *rpc.Client         // RPC client connection
	wsClient    *ethclient.Client   // WebSocket client connection
	lastBlockNo int64               // Last processed block number
}

// NewEthBlockListener creates a new EthBlockListener
func NewEthBlockListener(rpcURL, wsURL string, blockChan chan wstypes.BlockI, logger lib.LoggerI) (*EthBlockListener, error) {
	// Create a new EthBlockListener instance
	listener := &EthBlockListener{
		rpcURL:      rpcURL,
		wsURL:       wsURL,
		blockChan:   blockChan,
		logger:      logger,
		lastBlockNo: -1, // Initialize to -1 to process the first block
	}

	// Connect to the Ethereum node via RPC
	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum RPC node: %w", err)
	}
	listener.rpcClient = rpcClient
	listener.logger.Infof("Connected to Ethereum RPC node at %s", rpcURL)

	// Connect to the Ethereum node via WebSocket
	wsClient, err := ethclient.Dial(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum WebSocket node: %w", err)
	}
	listener.wsClient = wsClient
	listener.logger.Infof("Connected to Ethereum WebSocket node at %s", wsURL)

	return listener, nil
}

// Start begins listening for new block headers
func (e *EthBlockListener) Start(ctx context.Context) error {
	// Create a channel for receiving new block headers
	headers := make(chan *types.Header)

	// Subscribe to new block headers
	sub, err := e.wsClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		return fmt.Errorf("failed to subscribe to new block headers: %w", err)
	}
	e.logger.Info("Subscribed to new block headers")

	// Process new block headers in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Context was canceled, stop the listener
				e.logger.Info("Context canceled, stopping EthBlockListener")
				sub.Unsubscribe()
				return

			case err := <-sub.Err():
				// Subscription error occurred
				e.logger.Errorf("Subscription error: %v", err)
				return

			case header := <-headers:
				// New block header received
				e.logger.Infof("New block header received: %d", header.Number.Int64())

				// Process the new block header
				err := e.processNewHeader(ctx)
				if err != nil {
					e.logger.Errorf("Error processing new header: %v", err)
				}
			}
		}
	}()

	return nil
}

// processNewHeader fetches the latest finalized block and processes it
func (e *EthBlockListener) processNewHeader(ctx context.Context) error {
	// Get the latest finalized block number
	var b FinalizedBlock
	err := e.rpcClient.Call(&b, "eth_getBlockByNumber", "finalized", true)
	if err != nil {
		return fmt.Errorf("failed to get finality status: %w", err)
	}

	// Convert hex block number to int64
	number, err := strconv.ParseInt(b.Number, 0, 64)
	if err != nil {
		return fmt.Errorf("error converting from hex to int: %w", err)
	}

	e.logger.Infof("Latest finalized block number: %d", number)

	// Check if we've already processed this block
	if number <= e.lastBlockNo {
		e.logger.Infof("Block %d already processed (last processed: %d), skipping", number, e.lastBlockNo)
		return nil
	}

	// Fetch the finalized block
	blockNumber := big.NewInt(number)
	block, err := e.wsClient.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch block %d: %w", number, err)
	}

	e.logger.Infof("Fetched finalized block %d with %d transactions", number, len(block.Transactions()))

	// Convert to our block type
	ourBlock := NewBlock(block)

	// Send the block through the channel
	e.blockChan <- ourBlock
	e.logger.Infof("Sent block %d through channel", number)

	// Update the last processed block number
	e.lastBlockNo = number

	return nil
}

// Close closes the connections to the Ethereum node
func (e *EthBlockListener) Close() {
	// Close the RPC client
	if e.rpcClient != nil {
		e.rpcClient.Close()
		e.logger.Info("Closed RPC client connection")
	}

	// WebSocket client doesn't have an explicit Close method in go-ethereum
	e.logger.Info("EthBlockListener closed")
}
