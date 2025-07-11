#!/bin/sh

# Define the keystore path
KEYSTORE_PATH="$HOME/.canopy/keystore.json"

# Function to extract the address for a nick
get_address_for_nick() {
    local nick="$1"
    jq -r --arg key "$nick" '.nicknameMap[$key]' "$KEYSTORE_PATH"
}
