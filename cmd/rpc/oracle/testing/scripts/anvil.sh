#!/usr/bin/env bash

ANVIL_URL="http://localhost:8545"

# Run anvil in the background
echo "Starting anvil node..."
# RUST_LOG=info anvil --block-time 2 &
RUST_LOG=info anvil -vvv --print-traces --block-time 2 --chain-id 1 &
ANVIL_PID=$!

# Wait a moment for anvil to start
sleep 3

# Check if anvil node is running
echo "Checking if anvil node is running..."
if ! curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $ANVIL_URL > /dev/null; then
    echo "Error: Anvil node is not running at $ANVIL_URL"
    kill $ANVIL_PID 2>/dev/null
    exit 1
fi
echo "Anvil node is running"

# Run the initialization scripts
echo "Deploying USDC ERC20 contract..."
scripts/init_usdc.sh

echo "Minting USDC for test account..."
scripts/mint.sh

# Wait for anvil process to stop
echo "Waiting for anvil process to stop..."
wait $ANVIL_PID
