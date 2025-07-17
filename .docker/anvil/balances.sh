#!/usr/bin/env bash

source env/anvil.env
source env/usdc_contract.env
source env/testing.env

# Function to get USDC balance for an account
get_usdc_balance() {
    account=$1
    balance=$(cast call $USDC_CONTRACT "balanceOf(address)(uint256)" $account --rpc-url http://localhost:8545 2>&1)
    if [ $? -eq 0 ]; then
        echo "$account: $balance"
    else
        echo "Error getting USDC balance for $account"
    fi
}

# Function to get ETH balance for an account
get_eth_balance() {
    account=$1
    balance=$(cast balance $account --rpc-url http://localhost:8545 2>&1)
    if [ $? -eq 0 ]; then
        echo "$account: $balance"
    else
        echo "Error getting ETH balance for $account"
    fi
}

# Function to get all CNPY balances
get_cnpy_balances() {
    canopy_output=$(canopy query accounts 2>&1)
    canopy_exit_code=$?
    if [ $canopy_exit_code -eq 0 ]; then
        local jq_output=$(echo "$canopy_output" | jq -r '.results[] | "\(.address): \(.amount)"' 2>&1)
        local jq_exit_code=$?
        if [ $jq_exit_code -eq 0 ]; then
            echo "$jq_output"
        else
            echo "Error parsing CNPY balance data: $jq_output"
        fi
    else
        echo "Error getting CNPY balances"
    fi
}

# Display balances
echo "Ethereum USDC Balances:"
for i in {0..2}; do
    account_var="ACCOUNT_$i"
    account=${!account_var}
    if [ -n "$account" ]; then
        get_usdc_balance "$account"
    else
        echo "Warning: Account $i not found in environment"
    fi
done
echo

echo "Ethereum ETH Balances:"
for i in {0..2}; do
    account_var="ACCOUNT_$i"
    account=${!account_var}
    if [ -n "$account" ]; then
        get_eth_balance "$account"
    else
        echo "Warning: Account $i not found in environment"
    fi
done
echo

echo "CNPY Balances:"
get_cnpy_balances
echo
