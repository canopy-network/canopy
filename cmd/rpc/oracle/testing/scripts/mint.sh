#!/usr/bin/env bash
source env/usdc_contract.env
source env/anvil.env

mint() {
    local contract="$1"
    local account="$2"
    local amount="$3"
    local private_key="$4"
    local rpc_url="${5:-http://localhost:8545}"

    # Execute the cast send command and capture both output and exit code
    output=$(cast send "$contract" "mint(address,uint256)" "$account" "$amount" --private-key "$private_key" --rpc-url "$rpc_url" --json 2>&1)
    exit_code=$?

    # Check if the command was successful
    if [ $exit_code -eq 0 ]; then
        echo "Minted $amount to $account"

        # Parse transaction hash from JSON output if needed
        tx_hash=$(echo "$output" | jq -r '.transactionHash // empty' 2>/dev/null)
        if [ -n "$tx_hash" ]; then
            echo "Transaction hash: $tx_hash"
        fi
    else
        echo "Transaction failed with exit code: $exit_code"
        echo "Error output: $output"
        return $exit_code
    fi
}

mint "$USDC_CONTRACT" "$ACCOUNT_1" "1000000000000" "$PRIVATE_KEY_1"
