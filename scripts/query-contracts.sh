#!/bin/sh

set -e

# Get optional height parameter (default to 0 for latest)
HEIGHT=${1:-0}

#echo "Querying all contracts at height $HEIGHT..."

# Query contracts using canopy
canopy query contracts --height=$HEIGHT
