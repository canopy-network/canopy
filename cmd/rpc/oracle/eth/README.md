# Ethereum Block Provider Package

The `eth` package provides real-time Ethereum blockchain monitoring and block processing capabilities for the Canopy Network oracle system. It implements a complete block provider that fetches, validates, and processes Ethereum blocks while maintaining safe block confirmations and ERC20 token transaction handling.

## Overview

The `eth` package is designed to handle:
- Real-time monitoring of Ethereum blockchain through WebSocket connections
- Safe block processing with configurable confirmation requirements
- ERC20 token transaction analysis and validation
- Transaction receipt verification to prevent processing of failed transactions
- Canopy order detection and processing within Ethereum transactions
- Robust connection management with automatic reconnection capabilities

## Core Components

### EthBlockProvider

The main entry point for the Ethereum block monitoring system. It manages the overall blockchain monitoring process, including:
- Establishing and maintaining RPC and WebSocket connections to Ethereum nodes
- Subscribing to new block headers for real-time updates
- Processing safe blocks with required confirmations
- Managing block height tracking and sequential processing
- Coordinating transaction analysis and order extraction

### ERC20TokenCache

A caching layer for ERC20 token metadata retrieval. It manages:
- Token information caching (name, symbol, decimals)
- Smart contract interaction for token metadata
- Performance optimization through intelligent caching strategies

## Technical Details

### Safe Block Processing

The Ethereum block provider system uses a confirmation-based approach to ensure block finality. This is achieved by:

- **Confirmation Counting**: Only processing blocks that have received a configurable number of confirmations

This approach is similar to how financial institutions handle transaction confirmations - they wait for multiple confirmations before considering a transaction final to prevent issues from blockchain reorganizations.

The system ensures data integrity by avoiding processing blocks that might be reorganized out of the main chain.

### Real-time Header Monitoring

The Ethereum block provider system uses WebSocket subscriptions to achieve real-time blockchain monitoring:

1. **Subscription Establishment**: Creates a WebSocket subscription to receive new block headers as they are produced
2. **Safe Block Calculation**: Calculates which blocks are now safe to process based on the new header height
3. **Batch Processing**: Processes all newly safe blocks in sequence to maintain ordering

This provides near-real-time blockchain monitoring while maintaining safety through confirmation requirements.

## Configuration

The EthBlockProvider accepts the following configuration parameters:

- **NodeUrl**: HTTP/HTTPS RPC endpoint for the Ethereum node
- **NodeWSUrl**: WebSocket endpoint for real-time subscriptions
- **ChainID**: Ethereum chain identifier (1 for mainnet, 5 for Goerli, etc.)
- **SafeBlockConfirmations**: Number of confirmations required before processing blocks
- **RetryDelay**: Delay in seconds between connection retry attempts
