#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

NICKNAME=$1
WASM_FILE=$2

# Validate input
if [ $# -lt 1 ] || [ $# -gt 2 ]; then
    echo "Usage: $0 <nickname> [wasm_file_path]" >&2
    exit 1
fi

# Extract the address using functions
ADDRESS=$(get_address_for_nick "$NICKNAME")

if [ -z "$ADDRESS" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

# If WASM_FILE is not provided, use fzf to choose from testdata directory
if [ -z "$WASM_FILE" ]; then
    TESTDATA_DIR="$HOME/go/src/cosmwasm/artifacts"
    if [ ! -d "$TESTDATA_DIR" ]; then
        echo "Error: testdata directory '$TESTDATA_DIR' not found" >&2
        exit 1
    fi
    
    WASM_FILE=$(find "$TESTDATA_DIR" -name "*.wasm" -type f | fzf --prompt="Select WASM file: ")
    
    if [ -z "$WASM_FILE" ]; then
        echo "No WASM file selected" >&2
        exit 1
    fi
fi

# Check if WASM file exists
if [ ! -f "$WASM_FILE" ]; then
    echo "Error: WASM file '$WASM_FILE' not found" >&2
    exit 1
fi

echo "Storing WASM code from $WASM_FILE for $NICKNAME"
# Execute the command
echo canopy admin tx-store-code "$ADDRESS" "$WASM_FILE" --fee=50000 --password=test
canopy admin tx-store-code "$ADDRESS" "$WASM_FILE" --fee=50000 --password=test
