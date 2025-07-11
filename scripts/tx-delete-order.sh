#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

# Function to select a value using fzf
select_value() {
    local prompt=$1
    local options=$2

    local selected=$(echo -e "$options" | fzf --prompt="$prompt: ")

    if [ "$selected" = "Custom" ]; then
        read -p "Enter custom value: " CUSTOM_VALUE
        # Validate if custom input is an integer
        if ! [[ "$CUSTOM_VALUE" =~ ^[0-9]+$ ]]; then
            echo "Error: Custom value must be an integer." >&2
            exit 1
        fi
        echo "$CUSTOM_VALUE"
    else
        echo "$selected"
    fi
}

NICKNAME=$1

# Validate input
if [ -z "$NICKNAME" ]; then
    echo "Usage: $0 <nickname>" >&2
    exit 1
fi

# Extract the receiver details using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

CHAIN_ID=1

ORDER_ID_OPTIONS=$(canopy query orders | jq -r '.[0].orders[].id')
ORDER_ID=$(echo -e "$ORDER_ID_OPTIONS" | fzf --prompt="Select an ORDER_ID: ")

# # Check if ORDER_ID is provided, otherwise use fzf to select a value
# if [ -z "$ORDER_ID" ]; then
#     ORDER_ID=$(select_value "Select a sell amount" "1000000000\n1000000000\nCustom")
# fi

# # Check if RECV_AMOUNT is provided, otherwise use fzf to select a value
# if [ -z "$RECV_AMOUNT" ]; then
#     RECV_AMOUNT=$(select_value "Select a receive amount" "1000000000\n1000000000\nCustom")
# fi

# Create the transaction order
echo canopy admin tx-delete-order "$NICKNAME" "$ORDER_ID" "$CHAIN_ID" --password=test
canopy admin tx-delete-order "$NICKNAME" "$ORDER_ID" "$CHAIN_ID" --password=test

