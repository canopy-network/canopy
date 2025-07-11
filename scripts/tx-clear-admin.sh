#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
CONTRACT_ADDRESS=$2

# Validate input
if [ $# -ne 2 ]; then
    echo "Usage: $0 <nickname> <contract_address>" >&2
    exit 1
fi

# Extract the address using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Clearing admin of contract $CONTRACT_ADDRESS for $NICKNAME"

# Execute the command
echo canopy admin tx-clear-admin "$ADDRESS" "$CONTRACT_ADDRESS" --fee=10000 --password=test
canopy admin tx-clear-admin "$ADDRESS" "$CONTRACT_ADDRESS" --fee=10000 --password=test