#!/bin/sh
set -e

# Source the keystore utility functions
. ./keystore.sh

if [ $# -ne 1 ]; then
  echo "Usage: $0 <nickname>"
  exit 1
fi

NICKNAME=$1

# Get address for the provided nickname
ADDR=$(get_address_for_nick $NICKNAME)

if [ -z "$ADDR" ]; then
  echo "No address found for nickname: $NICKNAME"
  exit 1
fi

# Query account balance using canopy
canopy query account $ADDR | jq -r '.amount'
