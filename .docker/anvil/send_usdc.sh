#!/usr/bin/env bash
source ./contracts.sh

# Create the ABI-encoded call data
CALL_DATA=$(cast calldata "transfer(address,uint256)" $ACCOUNT_0 1000000)

# Append extra data (replace "0xabcdef" with your data)
FULL_DATA="${CALL_DATA}abcdef"

# Send the transaction with the combined data
cast send $USDC_CONTRACT "$FULL_DATA" --private-key $PRIVATE_KEY_1 --rpc-url http://localhost:8545
