#!/bin/bash
# pluginctl.sh - Control script for managing the TypeScript plugin
# Usage: ./pluginctl.sh {start|stop|status|restart}
# Configuration variables for paths and files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_SCRIPT="$SCRIPT_DIR/dist/main.js"
NODE_CMD="node"
PID_FILE="/tmp/plugin/typescript-plugin.pid"
LOG_FILE="/tmp/plugin/typescript-plugin.log"
PLUGIN_DIR="/tmp/plugin"
TARBALL="$SCRIPT_DIR/typescript-plugin.tar.gz"
# Timeout in seconds for graceful shutdown
STOP_TIMEOUT=10

# Extract tarball if dist doesn't exist
extract_if_needed() {
    # If dist/main.js already exists, nothing to do
    if [ -f "$NODE_SCRIPT" ]; then
        return 0
    fi
    
    # Check for tarball
    if [ -f "$TARBALL" ]; then
        echo "Extracting $TARBALL..."
        tar -xzf "$TARBALL" -C "$SCRIPT_DIR"
        if [ $? -eq 0 ] && [ -f "$NODE_SCRIPT" ]; then
            echo "Extraction complete"
            return 0
        else
            echo "Error: Failed to extract from $TARBALL"
            return 1
        fi
    fi
    
    return 1
}

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
    # Check if process exists and is running our Node.js script
    if ps -p "$pid" > /dev/null 2>&1; then
        # Verify it's actually our Node.js script
        if ps -p "$pid" -o cmd= | grep -q "node.*dist/main.js"; then
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
# Start the TypeScript plugin
start() {
    # Check if already running
    if is_running; then
        echo "TypeScript plugin is already running (PID: $(cat "$PID_FILE"))"
        return 1
    fi
    # Clean up any stale PID file
    cleanup_pid
    # Try to extract from tarball if dist doesn't exist
    extract_if_needed
    # Check if Node.js script exists
    if [ ! -f "$NODE_SCRIPT" ]; then
        echo "Error: Node.js script not found at $NODE_SCRIPT"
        echo "Run 'npm run build' to compile TypeScript or download typescript-plugin.tar.gz"
        return 1
    fi
    # Ensure plugin directory exists
    mkdir -p "$PLUGIN_DIR"
    # Start the Node.js script in background with nohup
    echo "Starting TypeScript plugin..."
    nohup "$NODE_CMD" "$NODE_SCRIPT" > "$LOG_FILE" 2>&1 &
    local pid=$!
    # Save PID to file
    echo "$pid" > "$PID_FILE"
    # Give it a moment to start
    sleep 1
    # Verify it started successfully
    if is_running; then
        echo "TypeScript plugin started successfully (PID: $pid)"
        echo "Log file: $LOG_FILE"
        return 0
    else
        echo "Error: TypeScript plugin failed to start"
        cleanup_pid
        return 1
    fi
}
# Stop the TypeScript plugin
stop() {
    # Check if running
    if ! is_running; then
        echo "TypeScript plugin is not running"
        cleanup_pid
        return 0
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE")
    echo "Stopping TypeScript plugin (PID: $pid)..."
    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid" 2>/dev/null
    # Wait for process to exit with timeout
    local count=0
    while [ $count -lt $STOP_TIMEOUT ]; do
        if ! ps -p "$pid" > /dev/null 2>&1; then
            echo "TypeScript plugin stopped successfully"
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
        echo "TypeScript plugin stopped (forced)"
        cleanup_pid
        return 0
    else
        echo "Error: Failed to stop TypeScript plugin"
        return 1
    fi
}
# Check status of TypeScript plugin
status() {
    # Check if running
    if is_running; then
        local pid=$(cat "$PID_FILE")
        echo "TypeScript plugin is running (PID: $pid)"
        return 0
    else
        echo "TypeScript plugin is not running"
        cleanup_pid
        return 3
    fi
}
# Restart the TypeScript plugin
restart() {
    echo "Restarting TypeScript plugin..."
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
