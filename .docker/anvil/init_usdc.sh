#!/bin/bash

# Variables
ANVIL_URL="http://anvil:8545"
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
CONTRACT_PATH="contracts/USDC.sol"
CONTRACT_NAME="USDC"
GAS_LIMIT="3000000"
ENV_FILE="env/usdc_contract.env"

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
