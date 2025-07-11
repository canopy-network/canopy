#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
CONTRACT_ADDRESS=$2
NEW_CODE_ID=$3
MIGRATE_MSG=${4:-'{}'}

# Validate input
if [ $# -lt 3 ]; then
    echo "Usage: $0 <nickname> <contract_address> <new_code_id> [migrate_msg]" >&2
    exit 1
fi

# Extract the address using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Migrating contract $CONTRACT_ADDRESS to code ID $NEW_CODE_ID for $NICKNAME"
echo "Migrate message: $MIGRATE_MSG"

# Execute the command
echo canopy admin tx-migrate-contract "$ADDRESS" "$CONTRACT_ADDRESS" "$NEW_CODE_ID" "$MIGRATE_MSG" --fee=25000 --password=test
canopy admin tx-migrate-contract "$ADDRESS" "$CONTRACT_ADDRESS" "$NEW_CODE_ID" "$MIGRATE_MSG" --fee=25000 --password=test