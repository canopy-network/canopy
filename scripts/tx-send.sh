#!/bin/sh

set -e

# Source the keystore utility functions
. ./keystore.sh

FROM=$1
TO=$2
QTY=$3

# Validate input
if [ $# -ne 3 ]; then
    echo "Usage: $0 <from_nickname> <to_nickname> <quantity>" >&2
    exit 1
fi

# Extract the receiver details using functions
FROM_ADDR=$(get_address_for_nick $FROM)
TO_ADDR=$(get_address_for_nick $TO)

if [ -z "$FROM_ADDR" ] || [ -z "$TO_ADDR" ]; then
    echo "Failed to extract address from keystore" >&2
    exit 1
fi

echo "Sending $QTY CNPY from $FROM to $TO"
# Execute the command
echo canopy admin tx-send $FROM_ADDR $TO_ADDR $QTY --password=test
canopy admin tx-send $FROM_ADDR $TO_ADDR $QTY --password=test
