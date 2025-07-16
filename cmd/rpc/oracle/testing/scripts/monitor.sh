#!/usr/bin/env bash
# Array to store which scripts to run
scripts_to_run=()

# Function to run balances script
run_balances() {
    if [ -f "scripts/balances.sh" ]; then
        bash scripts/balances.sh
    fi
}

# Function to run canopy orders script
run_orders() {
    if [ -f "scripts/orders.sh" ]; then
        bash scripts/orders.sh
    fi
}

# Function to run store script
run_store() {
    if [ -f "scripts/store.sh" ]; then
        bash scripts/store.sh /home/enielson/.canopy/oracle/orders
    fi
}

# If no parameters provided, run all scripts
if [ $# -eq 0 ]; then
    scripts_to_run=("balances" "orders" "store")
else
    # Loop through command line parameters
    for param in "$@"; do
        case $param in
            "balances"|"1")
                scripts_to_run+=("balances")
                ;;
            "orders"|"2")
                scripts_to_run+=("orders")
                ;;
            "store"|"3")
                scripts_to_run+=("store")
                ;;
        esac
    done
fi

# Main loop to run scripts at regular intervals
while true; do
    printf '\033[2J\033[H'
    balances_output=""
    orders_output=""
    store_output=""
    for script in "${scripts_to_run[@]}"; do
        case $script in
            "balances")
                balances_output=$(run_balances)
                ;;
            "orders")
                orders_output=$(run_orders)
                ;;
            "store")
                store_output=$(run_store)
                ;;
        esac
    done

    # Print all outputs at once
    for script in "${scripts_to_run[@]}"; do
        case $script in
            "balances")
                echo "$balances_output"
                echo
                ;;
            "orders")
                echo "$orders_output"
                echo
                ;;
            "store")
                echo "$store_output"
                echo
                ;;
        esac
    done

    # Sleep for 60 seconds before next iteration
    sleep 5
done
