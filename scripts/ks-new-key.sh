#!/bin/sh

# Check if expect is installed
command -v expect >/dev/null 2>&1 || { echo >&2 "expect is required but not installed. Aborting."; exit 1; }

# Check for correct number of arguments
if [ $# -ne 2 ]; then
    echo "Usage: $0 <nickname> <password>"
    exit 1
fi

NICKNAME="$1"
PASSWORD="$2"

expect <<EOF
spawn canopy admin ks-new-key
expect "password:"
send "$PASSWORD\r"
expect "nickname:"
send "$NICKNAME\r"
expect eof
EOF
