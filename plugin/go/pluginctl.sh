#!/bin/bash
# pluginctl.sh - Control script for managing the go-plugin binary
# Usage: ./pluginctl.sh {start|stop|status|restart}
# Configuration variables for paths and files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_PATH="$SCRIPT_DIR/go-plugin"
PID_FILE="/tmp/plugin/go-plugin.pid"
LOG_FILE="/tmp/plugin/go-plugin.log"
PLUGIN_DIR="/tmp/plugin"
# Timeout in seconds for graceful shutdown
STOP_TIMEOUT=10
# Check if the process is running based on PID file
is_running() {
    # Return 1 if PID file doesn't exist
    if [ ! -f "$PID_FILE" ]; then
        return 1
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE" 2>/dev/null)
    # Return 1 if PID is empty or not a number
    if [ -z "$pid" ] || ! [[ "$pid" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    # Check if process exists and is the go-plugin binary
    if ps -p "$pid" > /dev/null 2>&1; then
        # Verify it's actually our binary
        if ps -p "$pid" -o cmd= | grep -q "go-plugin"; then
            return 0
        fi
    fi
    # Process not running
    return 1
}
# Clean up stale PID file
cleanup_pid() {
    # Remove PID file if it exists
    if [ -f "$PID_FILE" ]; then
        rm -f "$PID_FILE"
    fi
}
# Start the go-plugin binary
start() {
    # Check if already running
    if is_running; then
        echo "go-plugin is already running (PID: $(cat "$PID_FILE"))"
        return 1
    fi
    # Clean up any stale PID file
    cleanup_pid
    # Check if binary exists and is executable
    if [ ! -x "$BINARY_PATH" ]; then
        echo "Error: Binary not found or not executable at $BINARY_PATH"
        return 1
    fi
    # Ensure plugin directory exists
    mkdir -p "$PLUGIN_DIR"
    # Start the binary in background with nohup
    echo "Starting go-plugin..."
    nohup "$BINARY_PATH" > "$LOG_FILE" 2>&1 &
    local pid=$!
    # Save PID to file
    echo "$pid" > "$PID_FILE"
    # Give it a moment to start
    sleep 1
    # Verify it started successfully
    if is_running; then
        echo "go-plugin started successfully (PID: $pid)"
        echo "Log file: $LOG_FILE"
        return 0
    else
        echo "Error: go-plugin failed to start"
        cleanup_pid
        return 1
    fi
}
# Stop the go-plugin binary
stop() {
    # Check if running
    if ! is_running; then
        echo "go-plugin is not running"
        cleanup_pid
        return 0
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE")
    echo "Stopping go-plugin (PID: $pid)..."
    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid" 2>/dev/null
    # Wait for process to exit with timeout
    local count=0
    while [ $count -lt $STOP_TIMEOUT ]; do
        if ! ps -p "$pid" > /dev/null 2>&1; then
            echo "go-plugin stopped successfully"
            cleanup_pid
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    # If still running after timeout, force kill
    echo "Process did not stop gracefully, forcing shutdown..."
    kill -KILL "$pid" 2>/dev/null
    sleep 1
    # Verify it's stopped
    if ! ps -p "$pid" > /dev/null 2>&1; then
        echo "go-plugin stopped (forced)"
        cleanup_pid
        return 0
    else
        echo "Error: Failed to stop go-plugin"
        return 1
    fi
}
# Check status of go-plugin binary
status() {
    # Check if running
    if is_running; then
        local pid=$(cat "$PID_FILE")
        echo "go-plugin is running (PID: $pid)"
        return 0
    else
        echo "go-plugin is not running"
        cleanup_pid
        return 3
    fi
}
# Restart the go-plugin binary
restart() {
    echo "Restarting go-plugin..."
    # Stop the process
    stop
    # Brief pause between stop and start
    sleep 2
    # Start the process
    start
}
# Main command routing
case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    restart)
        restart
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac
