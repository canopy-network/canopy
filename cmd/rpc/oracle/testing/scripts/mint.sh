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

# Define the mint amount (1 trillion units with 6 decimals = 1 million USDC)
MINT_AMOUNT="1000000000000"

echo "Minting USDC for all Anvil accounts..."

# Mint for accounts
for i in {0..2}; do
    account_var="ACCOUNT_$i"
    private_key_var="PRIVATE_KEY_$i"

    # Get the account and private key values
    account=${!account_var}
    private_key=${!private_key_var}

    if [ -n "$account" ] && [ -n "$private_key" ]; then
        echo "Minting $MINT_AMOUNT USDC for account $i: $account"
        mint "$USDC_CONTRACT" "$account" "$MINT_AMOUNT" "$private_key"
        echo "---"
    else
        echo "Warning: Account $i or its private key not found in environment"
    fi
done

echo "Completed minting USDC for all accounts."
