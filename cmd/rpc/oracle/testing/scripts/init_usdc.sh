#!/bin/bash

# Variables
ANVIL_URL="http://127.0.0.1:8545"
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
CONTRACT_PATH="testing/contracts/USDC.sol"
CONTRACT_NAME="USDC"
GAS_LIMIT="3000000"
ENV_FILE="env/usdc_contract.env"

# Check if anvil node is running
echo "Checking if anvil node is running..."
if ! curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $ANVIL_URL > /dev/null; then
    echo "Error: Anvil node is not running at $ANVIL_URL"
    exit 1
fi
echo "Anvil node is running"

# Deploy USDC contract
echo "Deploying USDC contract..."
DEPLOYMENT_OUTPUT=$(forge create $CONTRACT_PATH:$CONTRACT_NAME \
    --private-key $PRIVATE_KEY \
    --rpc-url $ANVIL_URL \
    --gas-limit $GAS_LIMIT \
    --broadcast)

if [ $? -ne 0 ]; then
    echo "Error: Contract deployment failed"
    exit 1
fi

# Extract contract address from deployment output
USDC_CONTRACT=$(echo "$DEPLOYMENT_OUTPUT" | grep "Deployed to:" | awk '{print $3}')

if [ -z "$USDC_CONTRACT" ]; then
    echo "Error: Could not extract contract address from deployment output"
    exit 1
fi

echo "USDC contract deployed at: $USDC_CONTRACT"

pwd
# Save contract address to environment file
echo "export USDC_CONTRACT=$USDC_CONTRACT" > $ENV_FILE

echo "Contract address saved to $ENV_FILE"
echo "Deployment completed successfully"
