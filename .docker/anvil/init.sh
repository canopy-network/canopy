#!/usr/bin/env bash

ANVIL_URL="http://anvil:8545"

# Run the initialization scripts
echo "Deploying USDC ERC20 contract..."
./init_usdc.sh

echo "Minting USDC for test accounts..."
./mint_usdc.sh
