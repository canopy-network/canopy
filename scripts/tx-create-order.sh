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
SELL_AMOUNT=${2:-} # If $2 is not provided, SELL_AMOUNT will be empty
RECV_AMOUNT=${3:-}  # If $3 is not provided, RECV_AMOUNT will be empty

# Validate input
if [ -z "$NICKNAME" ]; then
    echo "Usage: $0 <nickname> [sell_amount] [recv_amount]" >&2
    exit 1
fi

# Extract the receiver details using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

# Check if SELL_AMOUNT is provided, otherwise use fzf to select a value
if [ -z "$SELL_AMOUNT" ]; then
    SELL_AMOUNT=$(select_value "Select a sell amount" "1000000000\n1000000000\nCustom")
fi

# Check if RECV_AMOUNT is provided, otherwise use fzf to select a value
if [ -z "$RECV_AMOUNT" ]; then
    RECV_AMOUNT=$(select_value "Select a receive amount" "1000000000\n1000000000\nCustom")
fi

# Create the transaction order
echo canopy admin tx-create-order "$NICKNAME" "$SELL_AMOUNT" "$RECV_AMOUNT" 1 "$ADDRESS" --password=test
canopy admin tx-create-order "$NICKNAME" "$SELL_AMOUNT" "$RECV_AMOUNT" 1 "$ADDRESS" --password=test

