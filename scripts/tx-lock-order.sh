#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1

# Validate input
if [ -z "$1" ]; then
    echo "Usage: $0 <nickname>" >&2
    exit 1
fi

# Extract the receiver details using functions
ADDRESS=$(get_address_for_nick $1)

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

# Output the details
echo "Nickname: $NICKNAME"
echo "Address: $ADDRESS"

SELECTED_ORDER=$(canopy query orders | jq -c '.[] | .orders[] | {id, committee, amountForSale, requestedAmount, sellerReceiveAddress, sellersSendAddress}' | \
                     fzf --height 40% --reverse --inline-info --preview 'echo {}' --with-nth=1 | jq -r '.id')


# Check if the user made a selection
if [ -z "$SELECTED_ORDER" ]; then
    echo "No order selected." >&2
    exit 1
fi

echo $SELECTED_ORDER
# Execute the command

echo canopy admin tx-lock-order $NICKNAME $ADDRESS $SELECTED_ORDER --password=test
canopy admin tx-lock-order $NICKNAME $ADDRESS $SELECTED_ORDER --password=test
