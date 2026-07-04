#!/usr/bin/env bash
set -e

ANVIL_URL="http://anvil:8545"
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
MINT_AMOUNT="1000000000000"

ACCOUNTS=(
  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
  "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
  "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
)

echo "Deploying USDC contract..."
DEPLOY_OUTPUT=$(forge create /anvil/USDC.sol:USDC \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$ANVIL_URL" \
  --broadcast)

USDC_CONTRACT=$(echo "$DEPLOY_OUTPUT" | grep "Deployed to:" | awk '{print $3}')
if [ -z "$USDC_CONTRACT" ]; then
  echo "Error: failed to extract contract address"
  exit 1
fi
echo "USDC deployed at: $USDC_CONTRACT"

echo "Minting USDC to test accounts..."
for ACCOUNT in "${ACCOUNTS[@]}"; do
  cast send "$USDC_CONTRACT" "mint(address,uint256)" "$ACCOUNT" "$MINT_AMOUNT" \
    --private-key "$PRIVATE_KEY" \
    --rpc-url "$ANVIL_URL" > /dev/null
  echo "Minted to $ACCOUNT"
done

echo "Done"
