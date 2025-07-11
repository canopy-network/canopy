#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
CONTRACT_ADDRESS=$2
NEW_ADMIN_NICKNAME=$3

# Validate input
if [ $# -ne 3 ]; then
    echo "Usage: $0 <nickname> <contract_address> <new_admin_nickname>" >&2
    exit 1
fi

# Extract the addresses using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")
NEW_ADMIN_ADDRESS=$(get_address_for_nick "$NEW_ADMIN_NICKNAME")

if [ -z "$ADDRESS" ] || [ -z "$NEW_ADMIN_ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Updating admin of contract $CONTRACT_ADDRESS to $NEW_ADMIN_NICKNAME for $NICKNAME"

# Execute the command
echo canopy admin tx-update-admin "$ADDRESS" "$CONTRACT_ADDRESS" "$NEW_ADMIN_ADDRESS" --fee=15000 --password=test
canopy admin tx-update-admin "$ADDRESS" "$CONTRACT_ADDRESS" "$NEW_ADMIN_ADDRESS" --fee=15000 --password=test