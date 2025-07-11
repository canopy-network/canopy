#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
CONTRACT_ADDRESS=$2
EXECUTE_MSG=$3

# Validate input
if [ $# -ne 3 ]; then
    echo "Usage: $0 <nickname> <contract_address> <execute_msg>" >&2
    exit 1
fi

# Extract the address using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Executing contract $CONTRACT_ADDRESS for $NICKNAME"
echo "Execute message: $EXECUTE_MSG"

# Execute the command
echo canopy admin tx-execute-contract "$ADDRESS" "$CONTRACT_ADDRESS" "$EXECUTE_MSG" --fee=20000 --password=test
canopy admin tx-execute-contract "$ADDRESS" "$CONTRACT_ADDRESS" "$EXECUTE_MSG" --fee=20000 --password=test