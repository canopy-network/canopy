#!/usr/bin/env bash

GREEN='\033[32m'
RESET='\033[0m'

if [ $# -eq 0 ]; then
    echo "Usage: $0 <directory>"
    exit 1
fi

directory="$1"

if [ ! -d "$directory" ]; then
    echo "Error: '$directory' is not a valid directory"
    exit 1
fi

echo "Disk Store:"
for file in "$directory"/*; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        first_line=$(head -n 1 "$file" 2>/dev/null | jq -c -C . 2>/dev/null || head -n 1 "$file" 2>/dev/null)
        # Split filename on underscore and colorize first part

        id=$(echo "$filename" | cut -d'.' -f1)
        ext=$(echo "$filename" | cut -d'.' -f2-)

        echo -en "${GREEN}$id${RESET}"
        echo -n "."
        echo -e "${RESET}$ext"
        echo "  $first_line"
        echo
    fi
done
