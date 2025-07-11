#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
CODE_ID=$2
INIT_MSG=${3:-'{}'}
LABEL=${4:-'contract'}

# Validate input
if [ $# -lt 2 ]; then
    echo "Usage: $0 <nickname> <code_id> [init_msg] [label]" >&2
    exit 1
fi

# Extract the address using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Instantiating contract with code ID $CODE_ID for $NICKNAME"
echo "Init message: $INIT_MSG"
echo "Label: $LABEL"

# Execute the command
echo canopy admin tx-instantiate-contract "$ADDRESS" "$CODE_ID" "$INIT_MSG" "$LABEL" --fee=30000 --password=test
canopy admin tx-instantiate-contract "$ADDRESS" "$CODE_ID" "$INIT_MSG" "$LABEL" --fee=30000 --password=test